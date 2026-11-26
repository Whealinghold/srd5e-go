package combat

import (
	"context"
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// PerformAttackParams are the facts of one attack by a combatant of an encounter. Construct with a keyed
// literal; NewPerformAttackCommand checks them. The base Attack command holds what the host knows of the
// attack (Armor Class, modifiers, cover, range, damage); this use case adds what only an encounter knows:
// the conditions, Dodge, Help and the tracker.
type PerformAttackParams struct {
	// Encounter is the encounter the attack is made in.
	Encounter Encounter
	// Ledger is the ActionLedger of the host; the zero value is a new one.
	Ledger ActionLedger
	// AttackerID and TargetID name the two combatants; the attacker may be the target.
	AttackerID, TargetID CombatantID
	// Cost is what the attack spends.
	Cost AttackCost
	// Scene is the host's scene; its distance equals the Attack command's.
	Scene AttackScene
	// Attack is the base attack command; it is never altered.
	Attack ResolveAttackCommand
	// KnockOut is the attacker's choice to knock the target out; only a melee attack may.
	KnockOut KnockOutChoice
	// DamageAdjustments are the host's adjustments to the damage, applied in the order given.
	DamageAdjustments []DamageAdjustment
	// AlliedHelpers are the creatures allied with the attacker, for Help benefits.
	AlliedHelpers AlliedHelpers
	// TargetConcentration is the Concentration of the target, the damaged combatant, as the host holds it; nil is
	// NoConcentrationTracking.
	TargetConcentration ConcentrationInput
	// Gear are the deployed gear holds of the host; the zero value is none. Manacles on the attacker give one
	// Disadvantage (FR-159).
	Gear GearHolds
	// TargetParry is the target's standing choice to use Parry; nil is NoParry (FR-161).
	TargetParry ParryChoice
}

// PerformAttackCommand is the immutable input of PerformAttack.Run. Build it with NewPerformAttackCommand;
// the zero value is invalid.
type PerformAttackCommand struct{ p PerformAttackParams }

// NewPerformAttackCommand returns the command, taking its own copy of the adjustments. It returns an error
// matching srderr.ErrInvalidInput for a zero encounter, a blank id, an invalid Attack command, a Scene
// whose distance differs from the Attack command's, KnockOut with a ranged form, a nil adjustment, an
// invalid cost (an Opportunity Attack with a form other than a melee weapon attack or an Unarmed Strike,
// against someone other than its provoker, by its own provoker, or with a Visibility that does not match the
// scene; a spell attack with the Attack action; a means that does not match the form). It never panics.
func NewPerformAttackCommand(p PerformAttackParams) (PerformAttackCommand, error) {
	form := p.Attack.p.Form
	switch {
	case p.Encounter.IsZero() || p.AttackerID.IsZero() || p.TargetID.IsZero():
		return PerformAttackCommand{}, invalidInput("an attack needs an encounter, an attacker and a target")
	case !form.Valid():
		return PerformAttackCommand{}, invalidInput("an attack needs an Attack command")
	case p.Scene.distanceFeet != p.Attack.p.Distance:
		return PerformAttackCommand{}, invalidInput("the scene distance differs from the attack distance")
	case !p.KnockOut.Valid() || (p.KnockOut == KnockOut && form.Delivery() == AttackDeliveryRanged):
		return PerformAttackCommand{}, invalidInput("only a melee attack knocks a creature out")
	case slices.Contains(p.DamageAdjustments, nil):
		return PerformAttackCommand{}, invalidInput("a damage adjustment is nil")
	}
	if err := errors.Join(validOptionalConcentration(p.TargetConcentration), validParryChoice(p.TargetParry)); err != nil {
		return PerformAttackCommand{}, err
	}
	if err := validAttackCost(p.Cost, form, p.AttackerID, p.TargetID, p.Scene); err != nil {
		return PerformAttackCommand{}, err
	}
	p.DamageAdjustments = slices.Clone(p.DamageAdjustments)
	return PerformAttackCommand{p}, nil
}

// PerformAttackResult is the closed set of outcomes of PerformAttack.Run. Variants: PerformAttackCompleted,
// PerformAttackRejected, PerformAttackInvalid.
type PerformAttackResult interface{ isPerformAttackResult() }

// PerformAttackCompleted: the attack was made, a miss included. It holds the new encounter and ledger, the
// events in order, the attack as the attack rules resolved it and the damage resolutions (none for a miss).
type PerformAttackCompleted struct {
	ledgerCompleted
	attack      ResolveAttackResult
	resolutions []DamageResolution
	parry       ParryOutcome
	concentration
	limitedUse LimitedUseState
}

// PerformAttackRejected: a rule stopped the attack; the encounter and ledger are as they came in.
type PerformAttackRejected struct{ ledgerRejected }

// PerformAttackInvalid: the host named a combatant the encounter does not hold, or the call was over a
// cap; the encounter and ledger are as they came in.
type PerformAttackInvalid struct{ ledgerInvalid }

func (PerformAttackCompleted) isPerformAttackResult() {}
func (PerformAttackRejected) isPerformAttackResult()  {}
func (PerformAttackInvalid) isPerformAttackResult()   {}

// Attack returns the attack exactly as ResolveAttack's flow resolved it: with the roll and the damage, as an
// automatic miss, never AttackNotPossible (that is a rejection).
func (c PerformAttackCompleted) Attack() ResolveAttackResult { return c.attack }

// Resolutions returns a copy of the damage resolutions of the hit, one per Damage; none without damage.
func (c PerformAttackCompleted) Resolutions() []DamageResolution { return slices.Clone(c.resolutions) }

// PerformAttack is the use case of one attack in an encounter: spend the cost, roll the attack and the
// damage, hand the damage to the encounter and settle the consequences (SRD 5.2.1, "Making an Attack",
// pp. 14-15). Create it with NewPerformAttack.
type PerformAttack struct{ resolutionUse }

// NewPerformAttack returns the use case rolling through roller. Options: WithTelemetry, WithClock. It returns
// an error matching srderr.ErrInvalidInput for a nil roller.
func NewPerformAttack(roller Roller, opts ...Option) (PerformAttack, error) {
	if roller == nil {
		return PerformAttack{}, invalidInput("an attack needs a dice roller")
	}
	return PerformAttack{newResolutionUse(roller, opts)}, nil
}

// Run performs the attack, in this order and ending at the first stop: the stops of the encounter, the
// ledger, the tracker, the targeting restriction, the resolution (an impossible attack spends and rolls
// nothing), then the commit, the damage and the consequences. It returns PerformAttackRejected for a stop,
// PerformAttackInvalid for an unknown combatant, and PerformAttackCompleted otherwise. See the package
// documentation for the events and their order. It returns an error as the package documentation says.
func (u PerformAttack) Run(ctx context.Context, cmd PerformAttackCommand) (PerformAttackResult, error) {
	if u.flow.roller == nil || !cmd.p.Attack.p.Form.Valid() {
		return nil, invalidInput("zero use case or command")
	}
	op := combatOperation{valueOperationPerformAttack, spanPerformAttack}
	return execute(ctx, u.resolutionUse, op, func() (PerformAttackResult, error) { return u.perform(cmd.p) })
}

// perform is the work of Run.
func (u PerformAttack) perform(p PerformAttackParams) (PerformAttackResult, error) {
	rejected := func(r ResolutionRejection) PerformAttackResult {
		return PerformAttackRejected{ledgerRejected{resolutionRejected{r, p.Encounter}, p.Ledger}}
	}
	invalid := func(pr ResolutionProblem) PerformAttackResult {
		return PerformAttackInvalid{ledgerInvalid{resolutionInvalid{pr, p.Encounter}, p.Ledger}}
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

	help := p.Ledger.helps.ConsumeForAttackRoll(p.AttackerID, p.TargetID, p.AlliedHelpers)
	add := inputs.Additions()
	var helpCircumstances []modifier.RollCircumstance
	var helpEvents []HelpEvent
	helps := p.Ledger.helps
	if used, ok := help.(AttackRollHelpUsed); ok {
		helpCircumstances, helpEvents, helps = used.Circumstances(), used.Events(), used.Benefits()
	}
	add.Circumstances = slices.Concat(add.Circumstances, p.Gear.AttackRollCircumstances(p.AttackerID), helpCircumstances)
	derived, err := p.Attack.WithAdditions(add)
	if err != nil {
		return nil, err
	}
	res, err := u.flow.resolveAttack(derived)
	if err != nil {
		return nil, err
	}
	if n, no := res.(AttackNotPossible); no {
		return rejected(AttackNotPossibleRejection{n.Reason()}), nil
	}
	hit, rolled := res.(AttackResolved)
	var roll d20.AttackRollResult
	if rolled {
		roll = hit.Attack()
	}
	after := tracker
	if applied != nil {
		after = applied.Tracker()
	}
	parry, err := planParry(p.TargetParry, after, p.TargetID, target, roll, p.Attack.p.Form.Delivery() == AttackDeliveryMelee)
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
	outcome := attackedOutcome(res)
	if parry.response != nil {
		outcome = parriedOutcome(outcome, *parry.response)
	}
	r.add(AttackPerformed{r.base(), p.AttackerID, p.TargetID, outcome})
	if parry.response != nil {
		r.add(parry.response.Event())
	}
	for _, e := range helpEvents {
		r.add(e)
	}
	var resolutions []DamageResolution
	damageFrom := len(r.events)
	if rolled && hit.Hit() && !parry.becameMiss() && len(hit.Damage()) > 0 {
		delivery, err := (DamageHandoff{}).Delivery(hit.Delivery(), p.KnockOut)
		if err != nil {
			return nil, err
		}
		tr, err := r.enc.ApplyDamage(p.TargetID, hit.Damage(), p.DamageAdjustments, delivery, hit.HitKind())
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
		return invalid(LimitExceededProblem{}), nil
	case r.exitInvalid():
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return PerformAttackCompleted{ledgerCompleted{resolutionCompleted{r.enc, r.events},
		recorded.ledger.withHelps(helps)}, res, resolutions, parryOutcomeOf(parry), conc, recorded.limitedUse}, nil
}

// parryOutcomeOf is the ParryOutcome of the plan: NotOffered for a plan with none.
func parryOutcomeOf(p parried) ParryOutcome {
	if p.outcome == nil {
		return ParryNotOffered{}
	}
	return p.outcome
}

// parriedOutcome is the outcome of an attack the target answered with Parry: the hit stands with the roll
// resolved again, or becomes a miss (FR-161).
func parriedOutcome(was AttackPerformedOutcome, r ParryResponse) AttackPerformedOutcome {
	if r.BecameMiss() {
		return AttackMissOutcome{r.Result()}
	}
	if h, ok := was.(AttackHitOutcome); ok {
		return AttackHitOutcome{r.Result(), h.kind}
	}
	return was
}

// attackedOutcome is the AttackPerformedOutcome of a resolved attack.
func attackedOutcome(r ResolveAttackResult) AttackPerformedOutcome {
	switch v := r.(type) {
	case AttackResolved:
		if v.Hit() {
			return AttackHitOutcome{v.Attack(), v.HitKind()}
		}
		return AttackMissOutcome{v.Attack()}
	case AttackAutomaticallyMissed:
		return AttackAutomaticallyMissedOutcome{v.Reason()}
	default:
		panic(unreachableError("attack result", r))
	}
}

// Parry returns what became of the target's Parry: ParryNotOffered, ParryNotUsed or ParryApplied (FR-161).
func (c PerformAttackCompleted) Parry() ParryOutcome { return c.parry }

// Concentration returns how the Concentration of the target came out: ConcentrationNotChecked when the host
// gave none, none was in progress, no damage was taken or the hit did not damage.
func (c PerformAttackCompleted) Concentration() ConcentrationCheckOutcome { return c.outcome }

// TargetSpellcasting returns the new SpellcastingState of the target and whether the host gave a
// TracksConcentration; it is the state of the next call.
func (c PerformAttackCompleted) TargetSpellcasting() (character.SpellcastingState, bool) {
	return c.targetSpellcasting()
}

// LimitedUse returns the new LimitedUseState of an attack made with RecordsMultiattack and true; the zero state
// and false for any other cost (FR-165).
func (c PerformAttackCompleted) LimitedUse() (LimitedUseState, bool) {
	return c.limitedUse, !c.limitedUse.IsZero()
}
