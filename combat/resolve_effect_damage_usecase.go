package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// EffectDamageTarget is one creature to damage and its attempt at the saving throw, if the plan
// has one. Construct it with a keyed literal.
type EffectDamageTarget struct {
	// Target is the creature.
	Target CombatantID
	// Attempt is how the creature attempts the save; nil when the plan has no save.
	Attempt d20.SaveAttempt
}

// ResolveEffectDamageCommand is the input of ResolveEffectDamage. Construct it with a keyed literal.
// The use case copies Targets and changes nothing in the command.
type ResolveEffectDamageCommand struct {
	// Plan is the damage an effect called for.
	Plan EffectDamagePlan
	// Targets are the creatures to damage, in a fixed order: the host lists the area members.
	Targets []EffectDamageTarget
}

// EffectDamageAssignment is the damage for one target, for ApplyDamage: Damage of the plan's type
// with the Adjustment of a successful save (HalfDamage, or nil), and the Save evidence (nil when the
// plan has no save). A success on a save of no damage gives Damage of 0. Construct it with a keyed
// literal.
type EffectDamageAssignment struct {
	// Target is the creature.
	Target CombatantID
	// Damage is the damage to apply.
	Damage Damage
	// Adjustment is HalfDamage for a success on a save of half damage, else nil.
	Adjustment DamageAdjustment
	// Save is the saving throw made, or nil.
	Save UnarmedStrikeSave
}

// ResolveEffectDamageResult is the closed set of outcomes of ResolveEffectDamage.Run. Variants:
// ResolveEffectDamageResolved, ResolveEffectDamageRejected.
type ResolveEffectDamageResult interface{ isResolveEffectDamageResult() }

// ResolveEffectDamageResolved: the damage was rolled once for all targets and one assignment built
// per target, in command order. Nothing has been applied. Immutable and safe for concurrent use.
type ResolveEffectDamageResolved struct {
	assignments []EffectDamageAssignment
	rolled      int
}

// Assignments returns the assignments in command order, as a new slice.
func (r ResolveEffectDamageResolved) Assignments() []EffectDamageAssignment {
	return slices.Clone(r.assignments)
}

// Rolled returns the damage rolled, before any save; 0 when the roll came to less than 0.
func (r ResolveEffectDamageResolved) Rolled() int { return r.rolled }

// ResolveEffectDamageRejected: the command was refused before any die (FR-039).
type ResolveEffectDamageRejected struct{ rejection ActiveEffectRejection }

// Rejection returns every problem of the first failing group of checks (no_targets, then
// duplicate_target, then missing_save_attempt).
func (r ResolveEffectDamageRejected) Rejection() ActiveEffectRejection { return r.rejection }

func (ResolveEffectDamageResolved) isResolveEffectDamageResult() {}
func (ResolveEffectDamageRejected) isResolveEffectDamageResult() {}

// ResolveEffectDamage is the use case that rolls the damage an effect planned, once for all its
// targets, and makes the saving throws that can halve or remove it (SRD 5.2.1, "Saving Throws and
// Damage", p. 16). It applies nothing and calls no other use case. Create it with
// NewResolveEffectDamage; the zero value is not usable. Safe for concurrent use if its roller,
// telemetry and clock are.
type ResolveEffectDamage struct {
	use  activeEffectsUse
	flow attackFlow
}

// NewResolveEffectDamage returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveEffectDamage(roller Roller, opts ...Option) (ResolveEffectDamage, error) {
	if roller == nil {
		return ResolveEffectDamage{}, invalidInput("resolve effect damage without a dice roller")
	}
	return ResolveEffectDamage{newActiveEffectsUse(opts), attackFlow{roller}}, nil
}

// Run resolves the plan of cmd. The damage is rolled first, then each save in target order. It
// records the shared signals and the saves, and never logs. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case; ctx.Err() unwrapped, before any
// signal, for a cancelled ctx; an error matching srderr.ErrInvalidInput, recorded on the span
// only, for the zero plan or a zero target; and a *srderr.PortError (error type "dice_roller_error")
// with a nil result when the roller fails, panics or answers another expression, recorded once.
func (u ResolveEffectDamage) Run(ctx context.Context, cmd ResolveEffectDamageCommand) (ResolveEffectDamageResult, error) {
	if u.flow.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveEffectDamage, spanResolveEffectDamage}
	res, err := run(ctx, u.use.in, op, func(context.Context) (ResolveEffectDamageResult, error) { return u.resolve(cmd) },
		func(r ResolveEffectDamageResult) description {
			switch v := r.(type) {
			case ResolveEffectDamageResolved:
				return description{Class: resultApplied, Name: valueResultEffectDamageResolved}
			case ResolveEffectDamageRejected:
				return describeRejection(v.rejection)
			default:
				panic(unreachableError("resolve effect damage result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(ResolveEffectDamageResolved); ok {
		for _, a := range v.assignments {
			if a.Save != nil {
				s, _ := cmd.Plan.Save()
				u.use.recordSave(ctx, EffectSaveDamage, s.ability, a.Save.Outcome())
			}
		}
	}
	return res, nil
}

func (u ResolveEffectDamage) resolve(cmd ResolveEffectDamageCommand) (ResolveEffectDamageResult, error) {
	p := cmd.Plan
	if p.IsZero() || slices.ContainsFunc(cmd.Targets, func(t EffectDamageTarget) bool { return t.Target.IsZero() }) {
		return nil, invalidInput("resolve effect damage: the plan or a target is not well formed")
	}
	if rej := damageProblems(p, cmd.Targets); len(rej.problems) > 0 {
		return ResolveEffectDamageRejected{rej}, nil
	}
	_, total, err := u.flow.rollAmount(p.amount)
	if err != nil {
		return nil, err
	}
	total = max(total, 0)
	save, hasSave := p.Save()
	out := make([]EffectDamageAssignment, 0, len(cmd.Targets))
	for _, t := range cmd.Targets {
		a := EffectDamageAssignment{Target: t.Target, Damage: MustDamage(total, p.typ)}
		if hasSave {
			res, err := u.flow.resolveSave(t.Attempt, save.dc)
			if err != nil {
				return nil, err
			}
			a.Save = res
			if res.Outcome() == d20.Success {
				if save.onSuccess == SaveSuccessHalfDamage {
					a.Adjustment = HalfDamage{}
				} else {
					a.Damage = MustDamage(0, p.typ)
				}
			}
		}
		out = append(out, a)
	}
	return ResolveEffectDamageResolved{out, total}, nil
}

// damageProblems checks a resolution (FR-039, FR-063): no targets, else a creature listed twice,
// else a target without the attempt a save needs.
func damageProblems(p EffectDamagePlan, targets []EffectDamageTarget) ActiveEffectRejection {
	if len(targets) == 0 {
		return rejection(ActiveEffectProblem{Reason: EffectRejectNoTargets, Effect: p.effect})
	}
	var dup, missing []ActiveEffectProblem
	seen := map[CombatantID]bool{}
	for _, t := range targets {
		if seen[t.Target] {
			dup = append(dup, ActiveEffectProblem{Reason: EffectRejectDuplicateTarget, Effect: p.effect, Combatant: t.Target})
		}
		seen[t.Target] = true
		if s, ok := p.Save(); ok && t.Attempt == nil {
			missing = append(missing, ActiveEffectProblem{EffectRejectMissingSaveAttempt, p.effect, t.Target, s.ability})
		}
	}
	if len(dup) > 0 {
		return rejection(dup...)
	}
	return rejection(missing...)
}
