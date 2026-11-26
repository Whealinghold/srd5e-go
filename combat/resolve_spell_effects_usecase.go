package combat

import (
	"context"
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// rollAmount rolls e through the dice port and returns the face of each die and the total with
// the flat modifier of e. A roller failure is a *srderr.PortError.
func (f attackFlow) rollAmount(e dice.Expression) ([]int, int, error) {
	res, err := dice.Roll(f.roller, e)
	if err != nil {
		return nil, 0, err
	}
	var faces []int
	for _, r := range res.Rolls() {
		faces = append(faces, r.Value())
	}
	return faces, res.Total(), nil
}

// rollSpellDamage is rollDamage that also returns the faces rolled for each component.
func (f attackFlow) rollSpellDamage(spec DamageSpec, abilityModifier int, critical bool) ([]Damage, [][]int, error) {
	var composer AttackDamageComposer
	rolls := make([][]int, len(spec.components))
	for i, c := range spec.components {
		expr, ok := composer.Dice(c, critical)
		if !ok {
			continue
		}
		res, err := dice.Roll(f.roller, expr)
		if err != nil {
			return nil, nil, err
		}
		for _, r := range res.Rolls() {
			rolls[i] = append(rolls[i], r.Value())
		}
	}
	damages, err := composer.Compose(spec, rolls, abilityModifier, critical)
	return damages, rolls, err
}

var _ spellRolls = attackFlow{}

// ResolveSpellEffectsCommand is the input of ResolveSpellEffects. Construct it with a keyed
// literal. The use case copies the slices and changes nothing in the command.
type ResolveSpellEffectsCommand struct {
	// Spell is the spell cast.
	Spell content.Spell
	// Report is what the cast reported: the spell, the level, the spell save DC and attack bonus.
	Report character.CastReport
	// Caster is the creature casting the spell.
	Caster CombatantID
	// AttackModifiers are the caster's spell attack modifiers (spellcasting.md FR-116).
	AttackModifiers d20.RollModifiers
	// AbilityModifier is the casting class's spellcasting ability modifier.
	AbilityModifier int
	// Targets are the targets the host chose, in a fixed order.
	Targets []SpellTarget
	// Assignments say which target each instance of a part is aimed at.
	Assignments []InstanceAssignment
	// ChosenDamageType is the Damage Type chosen for a spell that offers a choice; zero is none.
	ChosenDamageType content.DamageType
	// PartIndexes, when not nil, resolves only these parts of the model (spell-effects.md FR-098).
	PartIndexes []int
	// InstanceCount, when above 0 with PartIndexes, replaces the instances of those parts.
	InstanceCount int
}

// ResolveSpellEffectsResult is the closed set of outcomes of ResolveSpellEffects.Run. Variants:
// SpellEffectsResolved, SpellEffectsTextOnly, SpellEffectsRejected.
type ResolveSpellEffectsResult interface{ isResolveSpellEffectsResult() }

// SpellEffectsResolved: the plan of what the spell does, with the evidence of every roll. Nothing
// has been applied. Immutable and safe for concurrent use.
type SpellEffectsResolved struct {
	spell    content.SpellID
	level    int
	coverage content.EffectCoverage
	parts    []ResolvedPart
	targets  []TargetEffects
	valid    int
}

// SpellID returns the spell.
func (r SpellEffectsResolved) SpellID() content.SpellID { return r.spell }

// EffectiveLevel returns the level the spell was cast at, or its own level when the report has none.
func (r SpellEffectsResolved) EffectiveLevel() int { return r.level }

// Coverage returns the coverage of the model; with CoveragePartial the host must resolve the rest
// of the description itself.
func (r SpellEffectsResolved) Coverage() content.EffectCoverage { return r.coverage }

// Parts returns the evidence of each resolved part, as a new slice.
func (r SpellEffectsResolved) Parts() []ResolvedPart { return slices.Clone(r.parts) }

// Targets returns the effects on each target, in command order, as a new slice.
func (r SpellEffectsResolved) Targets() []TargetEffects { return slices.Clone(r.targets) }

// Equal reports whether r and o are the same result.
func (r SpellEffectsResolved) Equal(o SpellEffectsResolved) bool {
	return r.spell == o.spell && r.level == o.level && r.coverage == o.coverage && r.valid == o.valid &&
		slices.EqualFunc(r.parts, o.parts, ResolvedPart.Equal) && slices.EqualFunc(r.targets, o.targets, TargetEffects.Equal)
}

// SpellEffectsTextOnly: the spell has no structured effects, so nothing was rolled and nothing
// planned; its text is the GM's to apply.
type SpellEffectsTextOnly struct {
	spell content.SpellID
	level int
}

// SpellID returns the spell.
func (r SpellEffectsTextOnly) SpellID() content.SpellID { return r.spell }

// EffectiveLevel returns the level the spell was cast at, or its own level when the report has none.
func (r SpellEffectsTextOnly) EffectiveLevel() int { return r.level }

// SpellEffectsRejection lists every reason a command was refused, in the fixed order of
// SpellEffectsRejectionKind. Immutable and safe for concurrent use.
type SpellEffectsRejection struct{ reasons []SpellEffectsRejectionKind }

// Reasons returns the reasons, as a new slice.
func (r SpellEffectsRejection) Reasons() []SpellEffectsRejectionKind { return slices.Clone(r.reasons) }

// Equal reports whether r and o list the same reasons.
func (r SpellEffectsRejection) Equal(o SpellEffectsRejection) bool {
	return slices.Equal(r.reasons, o.reasons)
}

// SpellEffectsRejected: the command was refused before any die was rolled.
type SpellEffectsRejected struct {
	rejection SpellEffectsRejection
	coverage  content.EffectCoverage
}

// Rejection returns the reasons.
func (r SpellEffectsRejected) Rejection() SpellEffectsRejection { return r.rejection }

// Coverage returns the coverage of the model.
func (r SpellEffectsRejected) Coverage() content.EffectCoverage { return r.coverage }

func (SpellEffectsResolved) isResolveSpellEffectsResult() {}
func (SpellEffectsTextOnly) isResolveSpellEffectsResult() {}
func (SpellEffectsRejected) isResolveSpellEffectsResult() {}

// ResolveSpellEffects is the use case of what a spell does once cast: it interprets the effect
// model of the spell, rolls the damage and the d20s through the dice port, composes the pure d20
// functions and the attack domain services, and returns a plan for ApplyDamage,
// RestoreHitPoints, GainTemporaryHitPoints and ApplyConditionToSet (SRD 5.2.1, "Effects",
// p. 106). It applies nothing, reads no content.Source (the host passes the Spell), tracks no
// duration, and calls no other use case. Create it with NewResolveSpellEffects; the zero value is
// not usable. It is safe for concurrent use only if its roller, telemetry and clock are.
type ResolveSpellEffects struct {
	in   instrumenter
	tel  telemetry.Telemetry
	flow attackFlow
}

// NewResolveSpellEffects returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveSpellEffects(roller Roller, opts ...Option) (ResolveSpellEffects, error) {
	if roller == nil {
		return ResolveSpellEffects{}, invalidInput("spell effects without a dice roller")
	}
	o := newOptions(opts)
	return ResolveSpellEffects{newInstrumenter(o.tel, o.clock), o.tel, attackFlow{roller}}, nil
}

// Run resolves the effects of the spell of cmd. A spell without structured effects gives
// SpellEffectsTextOnly with no die; a command the spell's rules refuse gives SpellEffectsRejected
// before any die; otherwise the dice are requested in the order of spell-effects.md FR-041 and
// SpellEffectsResolved carries the plan.
//
// It records the shared signals with the coverage and rejection extras, and, for a call that was
// not rejected, the resolutions counter and, for a resolved one, the rolls and targets signals.
// It never logs. It returns an error matching ErrInvalidSpellEffectsCommand (and so
// srderr.ErrInvalidInput) for contradictory input or a collection over its cap (plan P10: 256 targets, 256
// assignments, 64 part indexes, an instance count of 100; the error then also matches LimitExceededError),
// recorded on the span only; an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case; and ctx.Err() unwrapped,
// before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression; the failure is recorded once.
func (u ResolveSpellEffects) Run(ctx context.Context, cmd ResolveSpellEffectsCommand) (ResolveSpellEffectsResult, error) {
	if u.flow.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveSpellEffects, spanResolveSpellEffects}
	res, err := run(ctx, u.in, op, func(context.Context) (ResolveSpellEffectsResult, error) { return u.resolve(cmd) }, describeSpellEffects)
	if err != nil {
		return nil, err
	}
	u.record(ctx, res)
	return res, nil
}

// effectiveSpellLevel is the level the report says the spell was cast at, else the spell's own.
func effectiveSpellLevel(cmd ResolveSpellEffectsCommand) int {
	if at, ok := cmd.Report.CastAtLevel(); ok {
		return at
	}
	return int(cmd.Spell.Level())
}

// resolve is the work of Run.
func (u ResolveSpellEffects) resolve(cmd ResolveSpellEffectsCommand) (ResolveSpellEffectsResult, error) {
	if !cmd.Spell.ID().Valid() {
		return nil, spellCommandError("no spell", nil)
	}
	if err := spellEffectsCapsError(cmd); err != nil {
		return nil, err
	}
	level := effectiveSpellLevel(cmd)
	model, ok := cmd.Spell.Effects().(content.ModelledEffects)
	if !ok {
		return SpellEffectsTextOnly{cmd.Spell.ID(), level}, nil
	}
	res, err := resolveSpell(spellCast{
		spell: cmd.Spell, model: model, report: cmd.Report, caster: cmd.Caster, attackMods: cmd.AttackModifiers,
		abilityMod: cmd.AbilityModifier, targets: slices.Clone(cmd.Targets), assignments: slices.Clone(cmd.Assignments),
		chosen: cmd.ChosenDamageType, partIndexes: slices.Clone(cmd.PartIndexes), partIndexesSet: cmd.PartIndexes != nil,
		instanceCount: cmd.InstanceCount,
	}, u.flow)
	switch {
	case err != nil:
		return nil, err
	case len(res.rejection) > 0:
		return SpellEffectsRejected{SpellEffectsRejection{res.rejection}, model.Coverage()}, nil
	}
	return SpellEffectsResolved{cmd.Spell.ID(), level, model.Coverage(), res.parts, res.targets, res.valid}, nil
}

// spellCoverage is the srd5e.combat.spell_effects.coverage value of c.
func spellCoverage(c content.EffectCoverage) string {
	if c == content.CoveragePartial {
		return valueSpellCoveragePartial
	}
	return valueSpellCoverageComplete
}

// describeSpellEffects classifies a result: its class, srd5e.combat.result and the coverage and
// rejection extras (spell-effects.md FR-080, FR-081).
func describeSpellEffects(r ResolveSpellEffectsResult) description {
	switch v := r.(type) {
	case SpellEffectsResolved:
		return description{Class: resultApplied, Name: valueResultSpellResolved,
			Extras: []telemetry.Attr{telemetry.String(attrSpellEffectsCoverage, spellCoverage(v.coverage))}}
	case SpellEffectsTextOnly:
		return description{Class: resultApplied, Name: valueResultSpellTextOnly,
			Extras: []telemetry.Attr{telemetry.String(attrSpellEffectsCoverage, valueSpellCoverageTextOnly)}}
	case SpellEffectsRejected:
		class, name := resultInvalid, valueResultSpellRejectedInvalid
		if !slices.ContainsFunc(v.rejection.reasons, func(k SpellEffectsRejectionKind) bool { return !k.refusedOnly() }) {
			class, name = resultRefused, valueResultSpellRejectedRefused
		}
		return description{Class: class, Name: name, Extras: []telemetry.Attr{
			telemetry.String(attrSpellEffectsCoverage, spellCoverage(v.coverage)),
			telemetry.String(attrSpellEffectsRejection, v.rejection.reasons[0].String())}}
	default:
		panic(unreachableError("spell effects result", r))
	}
}

// record adds the signals that are not on the shared span: the resolutions of a call that was not
// rejected and the rolls and targets of a resolved one.
func (u ResolveSpellEffects) record(ctx context.Context, r ResolveSpellEffectsResult) {
	tel := u.tel.WithDefaults()
	resolutions := func(coverage string, level int) {
		l := valueSpellLevelCantrip
		if level > 0 {
			l = strconv.Itoa(level)
		}
		tel.Meter.Counter(telemetry.Descriptor{Name: metricSpellEffectsResolutions, Unit: "{resolution}",
			Description: "Spell effect resolutions that were not rejected, by coverage and level."}).
			Add(ctx, 1, telemetry.String(attrSpellEffectsCoverage, coverage), telemetry.String(attrSpellEffectsLevel, l))
	}
	switch v := r.(type) {
	case SpellEffectsTextOnly:
		resolutions(valueSpellCoverageTextOnly, v.level)
	case SpellEffectsResolved:
		resolutions(spellCoverage(v.coverage), v.level)
		tel.Meter.Histogram(telemetry.Descriptor{Name: metricSpellEffectsTargets, Unit: "{target}",
			Description: "Valid targets of a resolved spell effects call."}).Record(ctx, float64(v.valid))
		rolls := tel.Meter.Counter(telemetry.Descriptor{Name: metricSpellEffectsRolls, Unit: "{roll}",
			Description: "Attack rolls and saving throws made by spell effects."})
		for _, part := range v.parts {
			for _, inst := range part.instances {
				if _, rolled := inst.Attack(); rolled {
					rolls.Add(ctx, 1, telemetry.String(attrSpellEffectsRoll, valueSpellRollAttack), outcomeAttr(inst.hit))
				}
				for _, s := range inst.saves {
					kind, saved := valueSpellRollSave, s.save.Outcome() == d20.Success
					if _, forced := s.save.(UnarmedStrikeSaveForced); forced {
						kind = valueSpellRollSaveForced
					}
					rolls.Add(ctx, 1, telemetry.String(attrSpellEffectsRoll, kind), outcomeAttr(saved),
						telemetry.String(d20.AttrAbility, part.save.String()))
				}
			}
		}
	}
}

// outcomeAttr is srd5e.outcome: success when ok, else failure.
func outcomeAttr(ok bool) telemetry.Attr {
	if ok {
		return telemetry.String(telemetry.AttrOutcome, valueOutcomeSuccess)
	}
	return telemetry.String(telemetry.AttrOutcome, valueOutcomeFailure)
}
