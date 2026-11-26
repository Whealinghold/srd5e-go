package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ResolveStatBlockSaveParams are the facts of one saving throw effect of a stat block part on one target
// (combat-action-resolution.md FR-166). Construct with a keyed literal; NewResolveStatBlockSaveCommand checks them.
type ResolveStatBlockSaveParams struct {
	// Encounter is the encounter the effect happens in.
	Encounter Encounter
	// SourceID and TargetID name the creature using the part and the creature that saves; the source may be the target.
	SourceID, TargetID CombatantID
	// Monster is the stat block of the source and Part the position of the part with the Saving Throw notation.
	Monster content.Monster
	Part    LimitedUsePartRef
	// Immunities are the immunity windows the host holds now.
	Immunities SaveImmunities
	// Save is the host's attempt for the target (its modifiers; a RollAttempt, ChooseToFail or AutomaticFailure); the
	// conditions, Dodge and Exhaustion of the target are added. The DC and the ability are the part's.
	Save d20.SaveAttempt
	// Step says which failure a failed save is, for the Failure clauses that depend on it; the zero value is the first.
	Step SaveFailureStep
	// Damage is what the host rolled once for every target with RollStatBlockDamage; it may be empty.
	Damage []Damage
	// Delivery is the DamageDelivery of the damage.
	Delivery DamageDelivery
	// TargetConcentration is the Concentration of the target as the host holds it; nil is NoConcentrationTracking.
	TargetConcentration ConcentrationInput
}

// ResolveStatBlockSaveCommand is the immutable input of ResolveStatBlockSave.Run. Build it with
// NewResolveStatBlockSaveCommand; the zero value is invalid.
type ResolveStatBlockSaveCommand struct{ p ResolveStatBlockSaveParams }

// NewResolveStatBlockSaveCommand returns the command, taking its own copy of the damage. It returns an error matching
// srderr.ErrInvalidInput for a zero encounter or monster, a blank id, a part that is not in the stat block or has no
// Saving Throw notation, a save that is not a RollAttempt, a ChooseToFail or an AutomaticFailure, a step or delivery
// that is not defined, a damage that is not valid or an invalid Concentration input. It never panics.
func NewResolveStatBlockSaveCommand(p ResolveStatBlockSaveParams) (ResolveStatBlockSaveCommand, error) {
	switch p.Save.(type) {
	case d20.RollAttempt, d20.ChooseToFail, d20.AutomaticFailure:
	default:
		return ResolveStatBlockSaveCommand{}, invalidInput("a saving throw is a rolled attempt, a choice to fail or an automatic failure")
	}
	if p.Encounter.IsZero() || p.SourceID.IsZero() || p.TargetID.IsZero() || p.Monster.ID() == "" || !hasStatBlockPart(p.Monster, p.Part) ||
		(p.Step != 0 && (p.Step < SaveFirstFailure || p.Step > SaveSubsequentFailures)) {
		return ResolveStatBlockSaveCommand{}, invalidInput("a saving throw effect needs an encounter, two combatants, a monster, its part and a defined step")
	}
	if _, err := PlanStatBlockSaveEffect(sectionParts(p.Monster, p.Part.section)[p.Part.index]); err != nil {
		return ResolveStatBlockSaveCommand{}, err
	}
	if len(p.Damage) > 0 {
		if err := checkDamageInput(p.TargetID, p.Damage, nil, p.Delivery, NormalHit); err != nil {
			return ResolveStatBlockSaveCommand{}, err
		}
	}
	if err := validOptionalConcentration(p.TargetConcentration); err != nil {
		return ResolveStatBlockSaveCommand{}, err
	}
	p.Damage = slices.Clone(p.Damage)
	return ResolveStatBlockSaveCommand{p}, nil
}

// ResolveStatBlockSaveResult is the closed set of outcomes of ResolveStatBlockSave.Run. Variants:
// ResolveStatBlockSaveCompleted, ResolveStatBlockSaveRejected, ResolveStatBlockSaveInvalid.
type ResolveStatBlockSaveResult interface{ isResolveStatBlockSaveResult() }

// ResolveStatBlockSaveCompleted: the effect was resolved for the target. It holds the new encounter, the events in
// order, what the effect does to the target (StatBlockSaveEffectApplies with its clauses and grant, or
// StatBlockSaveTargetImmune), the target's saving throw (absent for an immune target) and the damage resolutions.
type ResolveStatBlockSaveCompleted struct {
	resolutionCompleted
	effect      StatBlockSaveTargetResult
	save        TestResolution
	resolutions []DamageResolution
	concentration
}

// ResolveStatBlockSaveRejected: a rule stopped the call; the encounter is as it came in.
type ResolveStatBlockSaveRejected struct{ resolutionRejected }

// ResolveStatBlockSaveInvalid: the host named a combatant the encounter does not hold, or the call was over a cap;
// the encounter is as it came in.
type ResolveStatBlockSaveInvalid struct{ resolutionInvalid }

func (ResolveStatBlockSaveCompleted) isResolveStatBlockSaveResult() {}
func (ResolveStatBlockSaveRejected) isResolveStatBlockSaveResult()  {}
func (ResolveStatBlockSaveInvalid) isResolveStatBlockSaveResult()   {}

// Effect returns what the effect does to the target: StatBlockSaveEffectApplies or StatBlockSaveTargetImmune. The
// host applies a condition the clauses name with the encounter, and Grant to its SaveImmunities.
func (c ResolveStatBlockSaveCompleted) Effect() StatBlockSaveTargetResult { return c.effect }

// Save returns the target's saving throw and true; nil and false for an immune target, which does not save.
func (c ResolveStatBlockSaveCompleted) Save() (TestResolution, bool) { return c.save, c.save != nil }

// Resolutions returns a copy of the damage resolutions; none when no damage was applied.
func (c ResolveStatBlockSaveCompleted) Resolutions() []DamageResolution {
	return slices.Clone(c.resolutions)
}

// Concentration returns how the Concentration of the target came out.
func (c ResolveStatBlockSaveCompleted) Concentration() ConcentrationCheckOutcome { return c.outcome }

// ResolveStatBlockSave is the use case of the saving throw effect of a stat block part on one target (SRD 5.2.1,
// "Saving Throw Effect Notation", p. 256; "Saving Throws and Damage", p. 16): the immunity windows, the target's
// save with its conditions, Dodge and Exhaustion, the clauses and the damage, halved when the part says so, then
// the consequences and Concentration of damage (FR-166). It is the spec's ResolveStatBlockSaveEffect under another
// name, because that name is the owner's pure function. Create it with NewResolveStatBlockSave.
type ResolveStatBlockSave struct{ resolutionUse }

// NewResolveStatBlockSave returns the use case rolling through roller. Options: WithTelemetry, WithClock. It returns
// an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveStatBlockSave(roller Roller, opts ...Option) (ResolveStatBlockSave, error) {
	if roller == nil {
		return ResolveStatBlockSave{}, invalidInput("a saving throw effect needs a dice roller")
	}
	return ResolveStatBlockSave{newResolutionUse(roller, opts)}, nil
}

// Run resolves the effect for the target: the stops of the encounter for the source and the target (the source
// first), then the immunity, which rolls no die, else the save, the effect and the damage. It returns
// ResolveStatBlockSaveRejected for a stop and ResolveStatBlockSaveInvalid for an unknown combatant, and the errors
// described for the package.
func (u ResolveStatBlockSave) Run(ctx context.Context, cmd ResolveStatBlockSaveCommand) (ResolveStatBlockSaveResult, error) {
	if u.flow.roller == nil || cmd.p.Save == nil {
		return nil, invalidInput("zero use case or command")
	}
	op := combatOperation{valueOperationResolveStatBlockSaveEffect, spanResolveStatBlockSaveEffect}
	return execute(ctx, u.resolutionUse, op, func() (ResolveStatBlockSaveResult, error) { return u.resolve(cmd.p) })
}

// resolve is the work of Run.
func (u ResolveStatBlockSave) resolve(p ResolveStatBlockSaveParams) (ResolveStatBlockSaveResult, error) {
	if s := preconditions(p.Encounter, p.SourceID, p.TargetID); s.stopped() {
		if s.problem != nil {
			return ResolveStatBlockSaveInvalid{resolutionInvalid{s.problem, p.Encounter}}, nil
		}
		return ResolveStatBlockSaveRejected{resolutionRejected{s.rejection, p.Encounter}}, nil
	}
	part := sectionParts(p.Monster, p.Part.section)[p.Part.index]
	plan, err := PlanStatBlockSaveEffect(part)
	if err != nil {
		return nil, err
	}
	r := &resolution{enc: p.Encounter}
	if p.Immunities.IsImmune(p.TargetID, p.SourceID, p.Part) {
		return ResolveStatBlockSaveCompleted{resolutionCompleted{r.enc, nil}, StatBlockSaveTargetImmune{p.TargetID}, nil, nil, noConcentration(p.TargetConcentration)}, nil
	}
	target, _ := p.Encounter.Find(p.TargetID)
	profile, _ := p.Encounter.tracker.ProfileOf(p.TargetID)
	dodge, err := p.Encounter.tracker.DodgeBenefits(p.TargetID, availabilityOf(target.conditions))
	if err != nil {
		return nil, err
	}
	adj, err := (AttackConditionInputs{}).Save(target.conditions, plan.Ability(), dodge, profile.Speeds.Walk())
	if err != nil {
		return nil, err
	}
	rolled, err := u.flow.resolveSave(foldSave(p.Save, adj), plan.DifficultyClass())
	if err != nil {
		return nil, err
	}
	save := testResolutionOf(rolled)
	check, err := checkResultOf(save, plan.DifficultyClass())
	if err != nil {
		return nil, err
	}
	steps := map[CombatantID]SaveFailureStep{}
	if p.Step != 0 {
		steps[p.TargetID] = p.Step
	}
	results, err := ResolveStatBlockSaveEffect(StatBlockSaveEffectInput{Part: part, Ref: p.Part, Source: p.SourceID, Targets: []CombatantID{p.TargetID},
		Saves: map[CombatantID]d20.CheckResult{p.TargetID: check}, Steps: steps, Immunities: p.Immunities})
	if err != nil {
		return nil, err
	}
	effect := results[0]
	var resolutions []DamageResolution
	damageFrom := len(r.events)
	if applies, ok := effect.(StatBlockSaveEffectApplies); ok && len(p.Damage) > 0 {
		var adjustments []DamageAdjustment
		if applies.DamageAdjustment() == SaveEffectDamageHalved {
			adjustments = []DamageAdjustment{HalfDamage{}}
		}
		tr, err := r.enc.ApplyDamage(p.TargetID, p.Damage, adjustments, p.Delivery, NormalHit)
		if err != nil {
			return nil, err
		}
		resolutions = r.mustCommit(tr)
	}
	damage := slices.Clone(r.events[damageFrom:])
	r.prompts(p.Encounter)
	conc, err := u.checkConcentration(r, p.TargetID, p.TargetConcentration, damage)
	if err != nil {
		return nil, err
	}
	r.concentrated(p.TargetID, conc)
	switch {
	case r.overLimit():
		return ResolveStatBlockSaveInvalid{resolutionInvalid{LimitExceededProblem{}, p.Encounter}}, nil
	case r.exitInvalid():
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return ResolveStatBlockSaveCompleted{resolutionCompleted{r.enc, r.events}, effect, save, resolutions, conc}, nil
}

// checkResultOf is the d20.CheckResult of a saving throw for the effect rules. A rolled save is its own result. A
// save that failed without a die (an automatic failure of the conditions or a choice to fail) has none, so it is
// stood in for by a failed save one below the DC: the SRD gives no margin for it, so the plain Failure clause
// applies and not "Failure by 5 or More" (package decision).
func checkResultOf(t TestResolution, dc d20.DifficultyClass) (d20.CheckResult, error) {
	if rolled, ok := t.(TestRolled); ok {
		return rolled.Result(), nil
	}
	roll, err := d20.NewRoll(modifier.Normal, []int{1})
	if err != nil {
		return d20.CheckResult{}, err
	}
	return d20.ResolveCheck(d20.SavingThrow, roll, d20.NewRollModifiers(modifier.New(int(dc)-2)), dc)
}
