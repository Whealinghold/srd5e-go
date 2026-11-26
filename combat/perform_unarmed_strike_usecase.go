package combat

import (
	"context"
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// PerformUnarmedStrikeParams are the facts of one Unarmed Strike by a combatant of an encounter. Construct with
// a keyed literal; NewPerformUnarmedStrikeCommand checks them. The fields are those of PerformAttackParams with
// the base Strike command in place of the Attack command.
type PerformUnarmedStrikeParams struct {
	// Encounter is the encounter the strike is made in.
	Encounter Encounter
	// Ledger is the ActionLedger of the host; the zero value is a new one.
	Ledger ActionLedger
	// AttackerID and TargetID name the two combatants; the attacker may be the target.
	AttackerID, TargetID CombatantID
	// Cost is what the strike spends; its means is AttackMeansUnarmedStrike.
	Cost AttackCost
	// Scene is the host's scene; its distance equals the Strike command's.
	Scene AttackScene
	// Strike is the base Unarmed Strike command; it is never altered.
	Strike ResolveUnarmedStrikeCommand
	// KnockOut is the attacker's choice to knock the target out; read by the Damage option.
	KnockOut KnockOutChoice
	// DamageAdjustments are the host's adjustments to the damage of the Damage option, in the order given.
	DamageAdjustments []DamageAdjustment
	// AlliedHelpers are the creatures allied with the attacker, for Help benefits.
	AlliedHelpers AlliedHelpers
	// TargetConcentration is the Concentration of the target, the damaged combatant, as the host holds it; nil is
	// NoConcentrationTracking.
	TargetConcentration ConcentrationInput
	// Gear are the deployed gear holds of the host; the zero value is none (Damage option only).
	Gear GearHolds
	// TargetParry is the target's standing choice to use Parry against the Damage option; nil is NoParry.
	TargetParry ParryChoice
}

// PerformUnarmedStrikeCommand is the immutable input of PerformUnarmedStrike.Run. Build it with
// NewPerformUnarmedStrikeCommand; the zero value is invalid.
type PerformUnarmedStrikeCommand struct{ p PerformUnarmedStrikeParams }

// NewPerformUnarmedStrikeCommand returns the command, taking its own copy of the adjustments. It returns an
// error matching srderr.ErrInvalidInput for a zero encounter, a blank id, a zero Strike command, a Scene whose
// distance differs from the Strike command's, a knock out choice that is not defined, a nil adjustment or an
// invalid cost (see NewPerformAttackCommand; the form is an Unarmed Strike). It never panics.
func NewPerformUnarmedStrikeCommand(p PerformUnarmedStrikeParams) (PerformUnarmedStrikeCommand, error) {
	switch {
	case p.Encounter.IsZero() || p.AttackerID.IsZero() || p.TargetID.IsZero():
		return PerformUnarmedStrikeCommand{}, invalidInput("a strike needs an encounter, an attacker and a target")
	case p.Strike.p.Option == nil:
		return PerformUnarmedStrikeCommand{}, invalidInput("a strike needs a Strike command")
	case p.Scene.distanceFeet != p.Strike.p.Distance:
		return PerformUnarmedStrikeCommand{}, invalidInput("the scene distance differs from the strike distance")
	case !p.KnockOut.Valid():
		return PerformUnarmedStrikeCommand{}, invalidInput("a knock out choice is not defined")
	case slices.Contains(p.DamageAdjustments, nil):
		return PerformUnarmedStrikeCommand{}, invalidInput("a damage adjustment is nil")
	}
	if err := errors.Join(validOptionalConcentration(p.TargetConcentration), validParryChoice(p.TargetParry)); err != nil {
		return PerformUnarmedStrikeCommand{}, err
	}
	if err := validAttackCost(p.Cost, UnarmedStrikeAttack, p.AttackerID, p.TargetID, p.Scene); err != nil {
		return PerformUnarmedStrikeCommand{}, err
	}
	p.DamageAdjustments = slices.Clone(p.DamageAdjustments)
	return PerformUnarmedStrikeCommand{p}, nil
}

// PerformUnarmedStrikeResult is the closed set of outcomes of PerformUnarmedStrike.Run. Variants:
// PerformUnarmedStrikeCompleted, PerformUnarmedStrikeRejected, PerformUnarmedStrikeInvalid.
type PerformUnarmedStrikeResult interface{ isPerformUnarmedStrikeResult() }

// PerformUnarmedStrikeCompleted: the strike was made. It holds the new encounter and ledger, the events in
// order, the strike as the attack rules resolved it, what its effect came to and the damage resolutions.
type PerformUnarmedStrikeCompleted struct {
	ledgerCompleted
	strike      ResolveUnarmedStrikeResult
	effect      UnarmedEffectOutcome
	resolutions []DamageResolution
	parry       ParryOutcome
	concentration
	limitedUse LimitedUseState
}

// PerformUnarmedStrikeRejected: a rule stopped the strike; the encounter and ledger are as they came in.
type PerformUnarmedStrikeRejected struct{ ledgerRejected }

// PerformUnarmedStrikeInvalid: the host named a combatant the encounter does not hold, or the call was over
// a cap; the encounter and ledger are as they came in.
type PerformUnarmedStrikeInvalid struct{ ledgerInvalid }

func (PerformUnarmedStrikeCompleted) isPerformUnarmedStrikeResult() {}
func (PerformUnarmedStrikeRejected) isPerformUnarmedStrikeResult()  {}
func (PerformUnarmedStrikeInvalid) isPerformUnarmedStrikeResult()   {}

// Strike returns the strike exactly as the attack rules resolved it. For a Grapple it holds the escape DC the
// host needs to begin a grapple record (FR-049); the host does that only when Effect is UnarmedEffectApplied
// with Grappled.
func (c PerformUnarmedStrikeCompleted) Strike() ResolveUnarmedStrikeResult { return c.strike }

// Effect returns what the Grapple or Shove came to; UnarmedEffectNotTriggered for the Damage option.
func (c PerformUnarmedStrikeCompleted) Effect() UnarmedEffectOutcome { return c.effect }

// Resolutions returns a copy of the damage resolutions of the Damage option's hit; none otherwise.
func (c PerformUnarmedStrikeCompleted) Resolutions() []DamageResolution {
	return slices.Clone(c.resolutions)
}

// PerformUnarmedStrike is the use case of one Unarmed Strike in an encounter, with its Damage, Grapple or
// Shove option (SRD 5.2.1, "Unarmed Strike", p. 190). Create it with NewPerformUnarmedStrike.
type PerformUnarmedStrike struct{ resolutionUse }

// NewPerformUnarmedStrike returns the use case rolling through roller. Options: WithTelemetry, WithClock. It
// returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewPerformUnarmedStrike(roller Roller, opts ...Option) (PerformUnarmedStrike, error) {
	if roller == nil {
		return PerformUnarmedStrike{}, invalidInput("a strike needs a dice roller")
	}
	return PerformUnarmedStrike{newResolutionUse(roller, opts)}, nil
}

// Run performs the strike under the flow and the stops of PerformAttack. The Damage option derives its inputs as
// an attack does; for a Grapple or Shove the target's save is the command's attempt adjusted by the target's
// conditions, Dodge and Exhaustion. When the save fails the condition is applied through the encounter; a
// target that already has it or is Immune to it does not stop the call (UnarmedEffectBlocked). Returns as
// PerformAttack.Run does.
func (u PerformUnarmedStrike) Run(ctx context.Context, cmd PerformUnarmedStrikeCommand) (PerformUnarmedStrikeResult, error) {
	if u.flow.roller == nil || cmd.p.Strike.p.Option == nil {
		return nil, invalidInput("zero use case or command")
	}
	op := combatOperation{valueOperationPerformUnarmedStrike, spanPerformUnarmedStrike}
	return execute(ctx, u.resolutionUse, op, func() (PerformUnarmedStrikeResult, error) { return u.perform(cmd.p) })
}

// foldSave adds the condition adjustments to a rolled save attempt; ChooseToFail and AutomaticFailure are kept.
// Automatic failure takes precedence over everything else (FR-045).
func foldSave(base d20.SaveAttempt, adj SaveAdjustments) d20.SaveAttempt {
	a, ok := base.(d20.RollAttempt)
	if !ok {
		return base
	}
	if _, forced := adj.Forced(); forced {
		return d20.AutomaticFailure{}
	}
	a.Circumstances = slices.Concat(a.Circumstances, adj.Circumstances())
	a.Modifiers = a.Modifiers.WithOthers(adj.Modifiers()...)
	return a
}

// perform is the work of Run.
func (u PerformUnarmedStrike) perform(p PerformUnarmedStrikeParams) (PerformUnarmedStrikeResult, error) {
	rejected := func(r ResolutionRejection) PerformUnarmedStrikeResult {
		return PerformUnarmedStrikeRejected{ledgerRejected{resolutionRejected{r, p.Encounter}, p.Ledger}}
	}
	invalid := func(pr ResolutionProblem) PerformUnarmedStrikeResult {
		return PerformUnarmedStrikeInvalid{ledgerInvalid{resolutionInvalid{pr, p.Encounter}, p.Ledger}}
	}
	if s := preconditions(p.Encounter, p.AttackerID, p.TargetID); s.stopped() {
		if s.problem != nil {
			return invalid(s.problem), nil
		}
		return rejected(s.rejection), nil
	}
	attacker, _ := p.Encounter.Find(p.AttackerID)
	target, _ := p.Encounter.Find(p.TargetID)

	recorded, err := p.Ledger.recordAttack(p.AttackerID, p.Cost)
	if err != nil {
		return nil, err
	}
	if recorded.refusal != nil {
		return rejected(LedgerRejection{recorded.refusal}), nil
	}

	tracker := p.Encounter.tracker
	step, err := costStep(tracker, p.AttackerID, attacker.conditions, p.Cost)
	if err != nil {
		return nil, err
	}
	var applied *TrackerStepApplied
	switch s := step.(type) {
	case TrackerStepRejected:
		return rejected(TrackerRejection{s.Reason}), nil
	case TrackerStepApplied:
		applied = &s
	}

	profile, _ := tracker.ProfileOf(p.TargetID)
	dodge, err := tracker.DodgeBenefits(p.TargetID, availabilityOf(target.conditions))
	if err != nil {
		return nil, err
	}
	inputs, err := (AttackConditionInputs{}).Attack(attacker.conditions, target.conditions, p.AttackerID, p.TargetID, p.Scene,
		dodge, profile.Speeds.Walk())
	if err != nil {
		return nil, err
	}
	if inputs.TargetingForbidden() {
		return rejected(TargetingForbiddenRejection{}), nil
	}

	params := p.Strike.p
	helps := p.Ledger.helps
	var helpEvents []HelpEvent
	var extra []modifier.Modifier
	switch o := params.Option.(type) {
	case UnarmedStrikeDamageOption:
		if used, ok := helps.ConsumeForAttackRoll(p.AttackerID, p.TargetID, p.AlliedHelpers).(AttackRollHelpUsed); ok {
			o.Circumstances = slices.Concat(o.Circumstances, inputs.Circumstances(), p.Gear.AttackRollCircumstances(p.AttackerID), used.Circumstances())
			helpEvents, helps = used.Events(), used.Benefits()
		} else {
			o.Circumstances = slices.Concat(o.Circumstances, inputs.Circumstances(), p.Gear.AttackRollCircumstances(p.AttackerID))
		}
		if inputs.AutomaticCritical() == AutomaticCriticalAutomatic {
			params.AutomaticCritical = AutomaticCriticalAutomatic
		}
		params.Option, extra = o, inputs.Modifiers()
	case UnarmedStrikeGrappleOption:
		adj, err := (AttackConditionInputs{}).Save(target.conditions, o.SaveAbility, dodge, profile.Speeds.Walk())
		if err != nil {
			return nil, err
		}
		o.Attempt = foldSave(o.Attempt, adj)
		params.Option = o
	case UnarmedStrikeShoveOption:
		adj, err := (AttackConditionInputs{}).Save(target.conditions, o.SaveAbility, dodge, profile.Speeds.Walk())
		if err != nil {
			return nil, err
		}
		o.Attempt = foldSave(o.Attempt, adj)
		params.Option = o
	}
	derived, err := NewResolveUnarmedStrikeCommand(params)
	if err != nil {
		return nil, err
	}
	res, err := u.flow.resolveUnarmedStrike(derived, extra...)
	if err != nil {
		return nil, err
	}
	if n, no := res.(UnarmedStrikeNotPossible); no {
		return rejected(AttackNotPossibleRejection{n.Reason()}), nil
	}

	var roll d20.AttackRollResult
	damaging, isDamage := res.(UnarmedStrikeDamageResolved)
	if isDamage {
		roll = damaging.Attack()
	}
	after := tracker
	if applied != nil {
		after = applied.Tracker()
	}
	choice := p.TargetParry
	if !isDamage {
		choice = nil // Grapple and Shove roll no attack roll for Parry to answer
	}
	parry, err := planParry(choice, after, p.TargetID, target, roll, true)
	if err != nil {
		return nil, err
	}

	r := &resolution{enc: p.Encounter}
	if applied != nil {
		if err := r.commitTurn(*applied); err != nil {
			return nil, err
		}
	}
	r.add(recorded.events...)
	if parry.step != nil {
		if err := r.commitTurn(*parry.step); err != nil {
			return nil, err
		}
	}
	outcome := unarmedOutcome(res)
	if d, ok := outcome.(UnarmedDamageOptionPerformed); ok && parry.response != nil {
		outcome = UnarmedDamageOptionPerformed{parriedOutcome(d.outcome, *parry.response)}
	}
	r.add(UnarmedStrikePerformed{r.base(), p.AttackerID, p.TargetID, outcome})
	if parry.response != nil {
		r.add(parry.response.Event())
	}
	for _, e := range helpEvents {
		r.add(e)
	}
	damageFrom := len(r.events)
	effect, resolutions, err := u.settle(r, p, res, parry.becameMiss())
	if err != nil {
		return nil, err
	}
	var damage []Event
	if len(resolutions) > 0 {
		damage = slices.Clone(r.events[damageFrom:])
	}
	r.prompts(p.Encounter)
	conc, err := u.checkConcentration(r, p.TargetID, p.TargetConcentration, damage)
	if err != nil {
		return nil, err
	}
	r.concentrated(p.TargetID, conc)
	switch {
	case r.overLimit():
		return invalid(LimitExceededProblem{}), nil
	case r.exitInvalid():
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return PerformUnarmedStrikeCompleted{ledgerCompleted{resolutionCompleted{r.enc, r.events},
		recorded.ledger.withHelps(helps)}, res, effect, resolutions, parryOutcomeOf(parry), conc, recorded.limitedUse}, nil
}

// settle hands the damage of the Damage option to the encounter, or applies the effect of a Grapple or Shove.
func (PerformUnarmedStrike) settle(r *resolution, p PerformUnarmedStrikeParams, res ResolveUnarmedStrikeResult, parried bool) (UnarmedEffectOutcome, []DamageResolution, error) {
	var plan UnarmedEffectPlan
	switch v := res.(type) {
	case UnarmedStrikeDamageResolved:
		if !v.Hit() || parried || len(v.Damage()) == 0 {
			return UnarmedEffectNotTriggered{}, nil, nil
		}
		delivery, err := (DamageHandoff{}).Delivery(AttackDeliveryMelee, p.KnockOut)
		if err != nil {
			return nil, nil, err
		}
		tr, err := r.enc.ApplyDamage(p.TargetID, v.Damage(), p.DamageAdjustments, delivery, v.HitKind())
		if err != nil {
			return nil, nil, err
		}
		return UnarmedEffectNotTriggered{}, r.mustCommit(tr), nil
	case UnarmedStrikeGrappleResolved:
		plan = (UnarmedEffects{}).Grapple(p.AttackerID, v.Grappled())
	case UnarmedStrikeShoveResolved:
		plan = (UnarmedEffects{}).Shove(v.Effect(), v.Applied())
	}
	cond, ok := plan.Condition()
	switch {
	case ok:
		tr, err := r.enc.ApplyCondition(p.TargetID, cond)
		if err != nil {
			return nil, nil, err
		}
		switch rej := tr.Rejection().(type) {
		case nil:
			r.mustCommit(tr)
			return UnarmedEffectApplied{cond}, nil, nil
		case AlreadyHasConditionRejection, ImmuneToConditionRejection:
			return UnarmedEffectBlocked{rej}, nil, nil
		default:
			panic(unreachableError("condition rejection", rej))
		}
	case plan.Push():
		return UnarmedEffectPushRecorded{}, nil, nil
	}
	return UnarmedEffectNotTriggered{}, nil, nil
}

// unarmedOutcome is the UnarmedStrikePerformedOutcome of a resolved Unarmed Strike.
func unarmedOutcome(r ResolveUnarmedStrikeResult) UnarmedStrikePerformedOutcome {
	switch v := r.(type) {
	case UnarmedStrikeDamageResolved:
		if v.Hit() {
			return UnarmedDamageOptionPerformed{AttackHitOutcome{v.Attack(), v.HitKind()}}
		}
		return UnarmedDamageOptionPerformed{AttackMissOutcome{v.Attack()}}
	case UnarmedStrikeGrappleResolved:
		return UnarmedSavePerformed{v.DC(), testResolutionOf(v.Save())}
	case UnarmedStrikeShoveResolved:
		return UnarmedSavePerformed{v.DC(), testResolutionOf(v.Save())}
	default:
		panic(unreachableError("unarmed strike result", r))
	}
}

// testResolutionOf is the TestResolution of a saving throw the attack rules resolved.
func testResolutionOf(s UnarmedStrikeSave) TestResolution {
	switch v := s.(type) {
	case UnarmedStrikeSaveRolled:
		return TestRolled{v.Result()}
	case UnarmedStrikeSaveForced:
		return TestForcedFailure{v.Result()}
	default:
		panic(unreachableError("saving throw", s))
	}
}

// Parry returns what became of the target's Parry against the Damage option: ParryNotOffered, ParryNotUsed or
// ParryApplied (FR-161).
func (c PerformUnarmedStrikeCompleted) Parry() ParryOutcome { return c.parry }

// Concentration returns how the Concentration of the target came out: ConcentrationNotChecked when the host
// gave none, none was in progress or the strike did not damage.
func (c PerformUnarmedStrikeCompleted) Concentration() ConcentrationCheckOutcome { return c.outcome }

// TargetSpellcasting returns the new SpellcastingState of the target and whether the host gave a
// TracksConcentration; it is the state of the next call.
func (c PerformUnarmedStrikeCompleted) TargetSpellcasting() (character.SpellcastingState, bool) {
	return c.targetSpellcasting()
}

// LimitedUse returns the new LimitedUseState of an attack made with RecordsMultiattack and true; the zero state
// and false for any other cost (FR-165).
func (c PerformUnarmedStrikeCompleted) LimitedUse() (LimitedUseState, bool) {
	return c.limitedUse, !c.limitedUse.IsZero()
}
