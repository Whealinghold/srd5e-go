package character

import (
	"strconv"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
)

// AdjustableD20Result is a D20 Test result that a die can still change, a closed set: AdjustableCheck (an ability
// check or saving throw) and AdjustableAttack (an attack roll). A forced failure, which rolled no d20, is not one
// (class-bard.md section 5.2). The interface cannot be implemented outside this package.
type AdjustableD20Result interface {
	// Outcome returns d20.Success or d20.Failure.
	Outcome() d20.Outcome
	// Total returns the natural value plus the modifiers.
	Total() int
	// SavingThrow reports whether the result is a saving throw.
	SavingThrow() bool
	isZero() bool
	adjustedBy(delta int) (AdjustableD20Result, error)
}

// AdjustableCheck is an ability check or saving throw result. Construct it with a keyed literal.
type AdjustableCheck struct{ Result d20.CheckResult }

// AdjustableAttack is an attack roll result. Construct it with a keyed literal.
type AdjustableAttack struct{ Result d20.AttackRollResult }

// Outcome returns the outcome of the check.
func (a AdjustableCheck) Outcome() d20.Outcome { return a.Result.Outcome() }

// Total returns the total of the check.
func (a AdjustableCheck) Total() int { return a.Result.Total() }

// SavingThrow reports whether the check is a saving throw.
func (a AdjustableCheck) SavingThrow() bool { return a.Result.Kind() == d20.SavingThrow }

func (a AdjustableCheck) isZero() bool { return a.Result.Roll().IsZero() }

func (a AdjustableCheck) adjustedBy(delta int) (AdjustableD20Result, error) {
	r, err := a.Result.AdjustedBy(delta)
	return AdjustableCheck{Result: r}, err
}

// Outcome returns the outcome of the attack roll.
func (a AdjustableAttack) Outcome() d20.Outcome { return a.Result.Outcome() }

// Total returns the total of the attack roll.
func (a AdjustableAttack) Total() int { return a.Result.Total() }

// SavingThrow returns false: an attack roll is not a saving throw.
func (AdjustableAttack) SavingThrow() bool { return false }

func (a AdjustableAttack) isZero() bool { return a.Result.Roll().IsZero() }

func (a AdjustableAttack) adjustedBy(delta int) (AdjustableD20Result, error) {
	r, err := a.Result.AdjustedBy(delta)
	return AdjustableAttack{Result: r}, err
}

// triggerProblem returns an error matching srderr.ErrInvalidInput for a nil or zero result.
func triggerProblem(r AdjustableD20Result) error {
	if r == nil || r.isZero() {
		return invalidInput("Bard feature on a nil or zero D20 Test result")
	}
	return nil
}

// BardicInspirationGrant is the Bardic Inspiration die a creature holds (SRD 5.2.1, "Using Bardic Inspiration", p. 31).
// A creature holds at most one. When it was granted is the host's to track: the one-hour rule takes the elapsed time
// as an argument. Immutable, comparable with ==; the zero value is not a grant.
type BardicInspirationGrant struct{ die BardicDie }

// NewBardicInspirationGrant returns the grant of die. It returns an error matching srderr.ErrInvalidInput for an
// invalid die. It never panics.
func NewBardicInspirationGrant(die BardicDie) (BardicInspirationGrant, error) {
	if !die.Valid() {
		return BardicInspirationGrant{}, invalidInput("Bardic Inspiration die d" + strconv.Itoa(int(die)))
	}
	return BardicInspirationGrant{die: die}, nil
}

// Die returns the die held.
func (g BardicInspirationGrant) Die() BardicDie { return g.die }

// IsZero reports whether g is the invalid zero value.
func (g BardicInspirationGrant) IsZero() bool { return g.die == 0 }

// InspirationRecipient is who a Bardic Inspiration die is offered to, a closed set: InspirationRecipientSelf and
// InspirationRecipientOther. The host decides the variant; the domain compares no identities (class-bard.md FR-027).
type InspirationRecipient interface{ isInspirationRecipient() }

// InspirationRecipientSelf is the Bard itself, which the feature refuses.
type InspirationRecipientSelf struct{}

// InspirationRecipientOther is another creature. CurrentGrant is the die it already holds, or nil. Construct it with
// a keyed literal.
type InspirationRecipientOther struct{ CurrentGrant *BardicInspirationGrant }

func (InspirationRecipientSelf) isInspirationRecipient()  {}
func (InspirationRecipientOther) isInspirationRecipient() {}

// BardicInspirationGranting is the outcome of an accepted GrantBardicInspiration.
type BardicInspirationGranting struct {
	pool  BardicInspirationPool
	grant BardicInspirationGrant
	event BardicInspirationGranted
}

// Pool returns the Bard's pool with one use expended.
func (g BardicInspirationGranting) Pool() BardicInspirationPool { return g.pool }

// Grant returns the die the recipient now holds.
func (g BardicInspirationGranting) Grant() BardicInspirationGrant { return g.grant }

// Event returns the BardicInspirationGranted event.
func (g BardicInspirationGranting) Event() BardicInspirationGranted { return g.event }

// GrantBardicInspiration gives a Bardic Inspiration die of the Bard's current size to another creature as a Bonus
// Action (SRD 5.2.1, "Using Bardic Inspiration", p. 31; the host charges the action). It expends one use of pool,
// whose maximum is first recomputed from charisma. The first reason that holds is returned, with the zero
// BardicInspirationGranting and the pool unchanged: BardNotABard for a zero level, BardSelfTarget for
// InspirationRecipientSelf or any recipient that is not an InspirationRecipientOther, BardNoUsesRemaining, then
// BardRecipientAlreadyInspired. The reason is zero when the grant is made. Inputs are not altered. id is the Bard's.
func GrantBardicInspiration(id Identifier, level BardLevel, pool BardicInspirationPool, charisma ability.Modifier, recipient InspirationRecipient) (BardicInspirationGranting, BardReason) {
	pool = pool.WithMaximumFrom(charisma)
	other, isOther := recipient.(InspirationRecipientOther)
	switch {
	case level.IsZero():
		return BardicInspirationGranting{}, BardNotABard
	case !isOther:
		return BardicInspirationGranting{}, BardSelfTarget
	case pool.remaining < 1:
		return BardicInspirationGranting{}, BardNoUsesRemaining
	case other.CurrentGrant != nil && !other.CurrentGrant.IsZero():
		return BardicInspirationGranting{}, BardRecipientAlreadyInspired
	}
	die := BardicDieAt(level)
	next := pool.spend()
	return BardicInspirationGranting{pool: next, grant: BardicInspirationGrant{die: die},
		event: BardicInspirationGranted{id: id, die: die, remaining: next.remaining}}, 0
}

// BardRecoveryTrigger is what may restore uses of Bardic Inspiration, a closed set: BardRecoveryRest and
// BardRecoveryInitiative. Casting a spell is never one (class-bard.md section 6).
type BardRecoveryTrigger interface{ isBardRecoveryTrigger() }

// BardRecoveryRest is a finished rest. Construct it with a keyed literal.
type BardRecoveryRest struct{ Kind RestKind }

// BardRecoveryInitiative is the Bard rolling Initiative (Superior Inspiration).
type BardRecoveryInitiative struct{}

func (BardRecoveryRest) isBardRecoveryTrigger()       {}
func (BardRecoveryInitiative) isBardRecoveryTrigger() {}

// BardicInspirationRecovery is the outcome of an accepted RecoverBardicInspiration.
type BardicInspirationRecovery struct {
	pool  BardicInspirationPool
	event BardicInspirationRestored
}

// Pool returns the pool after the recovery.
func (r BardicInspirationRecovery) Pool() BardicInspirationPool { return r.pool }

// Event returns the BardicInspirationRestored event.
func (r BardicInspirationRecovery) Event() BardicInspirationRestored { return r.event }

// RecoverBardicInspiration restores uses of Bardic Inspiration for a trigger (SRD 5.2.1, "Number of Uses", "Level 5:
// Font of Inspiration", "Level 18: Superior Inspiration", pp. 32-33): a Long Rest restores all uses at any level; a
// Short Rest restores all from Bard level 5; rolling Initiative restores uses until two remain (or the maximum, if
// lower) from Bard level 18 and never lowers a pool holding two or more. The maximum is first recomputed from
// charisma. A zero level gives BardNotABard; a trigger the level does not reach (or any other rest or trigger) gives
// BardBelowRequiredLevel; a trigger that would change nothing gives BardNothingToRestore. The reason is zero when
// uses are restored. Inputs are not altered.
func RecoverBardicInspiration(id Identifier, level BardLevel, pool BardicInspirationPool, charisma ability.Modifier, trigger BardRecoveryTrigger) (BardicInspirationRecovery, BardReason) {
	pool = pool.WithMaximumFrom(charisma)
	var next BardicInspirationPool
	switch t := trigger.(type) {
	case BardRecoveryRest:
		switch {
		case level.IsZero():
			return BardicInspirationRecovery{}, BardNotABard
		case t.Kind == LongRest, t.Kind == ShortRest && level.n >= bardFontOfInspirationLevel:
			next = pool.restoreAll()
		default:
			return BardicInspirationRecovery{}, BardBelowRequiredLevel
		}
	case BardRecoveryInitiative:
		switch {
		case level.IsZero():
			return BardicInspirationRecovery{}, BardNotABard
		case level.n < bardSuperiorLevel:
			return BardicInspirationRecovery{}, BardBelowRequiredLevel
		}
		next = pool.restoreToTwo()
	default:
		return BardicInspirationRecovery{}, BardBelowRequiredLevel
	}
	if next.remaining == pool.remaining {
		return BardicInspirationRecovery{}, BardNothingToRestore
	}
	return BardicInspirationRecovery{pool: next, event: BardicInspirationRestored{id: id, uses: next.remaining - pool.remaining, remaining: next.remaining}}, 0
}

// BardSlotExpenditure is the outcome of an accepted ExpendSlotForInspiration.
type BardSlotExpenditure struct {
	pool   BardicInspirationPool
	state  SpellcastingState
	events []Event
}

// Pool returns the pool with one use regained.
func (e BardSlotExpenditure) Pool() BardicInspirationPool { return e.pool }

// State returns the SpellcastingState with the slot expended.
func (e BardSlotExpenditure) State() SpellcastingState { return e.state }

// Events returns a new slice of the SpellSlotExpended event and the BardicInspirationRestored event, in that order.
func (e BardSlotExpenditure) Events() []Event { return append([]Event{}, e.events...) }

// ExpendSlotForInspiration regains one expended use of Bardic Inspiration by expending a spell slot of the given pool
// and level, no action required (SRD 5.2.1, "Level 5: Font of Inspiration", p. 32; class-bard.md FR-041, FR-042). It
// spends the slot through SpellcastingState.ExpendSlot and re-implements no slot rule. The checks run in this order
// and the first that holds is returned with the zero BardSlotExpenditure and nothing spent: BardNotABard for a zero
// level, BardBelowRequiredLevel below Bard level 5, BardNothingToRestore when no use is expended (before any slot is
// spent), then BardSlotRejected with the SlotOperationRejection of the spellcasting state. Both returned values are
// zero when the slot is spent. Inputs are not altered; the caller writes back the pool and the state.
func ExpendSlotForInspiration(id Identifier, level BardLevel, pool BardicInspirationPool, charisma ability.Modifier, state SpellcastingState, slotPool SpellSlotPool, slot SlotLevel) (BardSlotExpenditure, BardReason, SlotOperationRejection) {
	pool = pool.WithMaximumFrom(charisma)
	switch {
	case level.IsZero():
		return BardSlotExpenditure{}, BardNotABard, 0
	case level.n < bardFontOfInspirationLevel:
		return BardSlotExpenditure{}, BardBelowRequiredLevel, 0
	case pool.remaining >= pool.max:
		return BardSlotExpenditure{}, BardNothingToRestore, 0
	}
	switch s := state.ExpendSlot(slotPool, slot).(type) {
	case SlotExpenditureExpended:
		next := pool.restoreOne()
		return BardSlotExpenditure{pool: next, state: s.State(), events: []Event{s.Event(),
			BardicInspirationRestored{id: id, uses: 1, remaining: next.remaining}}}, 0, 0
	case SlotExpenditureRejected:
		return BardSlotExpenditure{}, BardSlotRejected, s.Reason()
	}
	return BardSlotExpenditure{}, BardSlotRejected, 0 // unreachable: SlotExpenditure is closed to those two variants
}

// bardInspirationDuration is how long a Bardic Inspiration die lasts: "Once within the next hour" (SRD 5.2.1, p. 31).
const bardInspirationDuration = time.Hour

// BardicInspirationUsePlan is what a recipient's use of its die needs rolled: the die. Only
// PlanBardicInspirationUse builds one.
type BardicInspirationUsePlan struct {
	id      Identifier
	die     BardicDie
	trigger AdjustableD20Result
}

// Die returns the die to roll.
func (p BardicInspirationUsePlan) Die() BardicDie { return p.die }

// PlanBardicInspirationUse decides whether the holder of grant can use the die on trigger, the D20 Test it just
// failed, elapsed after the grant (SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32; class-bard.md FR-033 to FR-035).
// Nothing is rolled here. The checks run in this order: BardNotAFailure when the trigger's outcome is not a failure
// (the die stays), then BardExpired when elapsed is more than one hour (the die is lost; exactly one hour is accepted,
// and a negative elapsed time counts as none). The reason is zero when the die may be rolled. id is the holder's.
//
// It returns an error matching srderr.ErrInvalidInput for a zero grant or a nil or zero trigger.
func PlanBardicInspirationUse(id Identifier, grant BardicInspirationGrant, elapsed time.Duration, trigger AdjustableD20Result) (BardicInspirationUsePlan, BardReason, error) {
	if grant.IsZero() {
		return BardicInspirationUsePlan{}, 0, invalidInput("Bardic Inspiration use with a zero grant")
	}
	if err := triggerProblem(trigger); err != nil {
		return BardicInspirationUsePlan{}, 0, err
	}
	switch {
	case trigger.Outcome() != d20.Failure:
		return BardicInspirationUsePlan{}, BardNotAFailure, nil
	case elapsed > bardInspirationDuration:
		return BardicInspirationUsePlan{}, BardExpired, nil
	}
	return BardicInspirationUsePlan{id: id, die: grant.die, trigger: trigger}, 0, nil
}

// BardicInspirationApplied is the outcome of ApplyBardicInspirationDie.
type BardicInspirationApplied struct {
	result AdjustableD20Result
	event  BardicInspirationExpended
}

// Result returns the D20 Test with the number rolled added to its total; the d20 is the original.
func (a BardicInspirationApplied) Result() AdjustableD20Result { return a.result }

// Event returns the BardicInspirationExpended event: the die is expended whether or not the test now succeeds.
func (a BardicInspirationApplied) Event() BardicInspirationExpended { return a.event }

// ApplyBardicInspirationDie adds the number rolled on the die of plan to the failed test through adjustedBy, so the
// natural 1 and 20 rules of an attack roll hold (class-bard.md FR-030 to FR-032, FR-104). The die is expended either
// way. It returns an error matching srderr.ErrInvalidInput for a zero plan or a number outside 1 to the die's sides.
func ApplyBardicInspirationDie(plan BardicInspirationUsePlan, rolled int) (BardicInspirationApplied, error) {
	if err := rollProblem(plan.die, rolled); err != nil || plan.trigger == nil {
		return BardicInspirationApplied{}, firstError(err, invalidInput("Bardic Inspiration die applied from a zero plan"))
	}
	adjusted, err := plan.trigger.adjustedBy(rolled)
	if err != nil {
		return BardicInspirationApplied{}, err
	}
	return BardicInspirationApplied{result: adjusted, event: BardicInspirationExpended{id: plan.id, feature: bardFeatureInspiration, die: plan.die, rolled: rolled}}, nil
}

// rollProblem returns an error matching srderr.ErrInvalidInput when die is invalid or rolled is outside 1 to its sides.
func rollProblem(die BardicDie, rolled int) error {
	if !die.Valid() || rolled < 1 || rolled > die.Sides() {
		return invalidInput("Bardic die roll " + strconv.Itoa(rolled) + " outside 1 to the die, or no die")
	}
	return nil
}

// firstError returns err when it is not nil and otherwise fallback.
func firstError(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}
