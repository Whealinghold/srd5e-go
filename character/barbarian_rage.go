package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
)

// rageMaximumDuration is how long a Rage can be maintained (SRD 5.2.1, "Rage", p. 29).
// The host supplies elapsed time; the character context reads no clock. Exactly 10
// minutes ends the Rage ("up to 10 minutes" is a limit; class-barbarian.md section 9).
const rageMaximumDuration = 10 * time.Minute

// RageEntry is the closed outcome of BarbarianRage.Enter. Variants: RageEntered,
// RageNotEntered.
type RageEntry interface{ isRageEntry() }

// InstinctivePounce is the movement allowance entering Rage gives from Barbarian level
// 7: up to half the Speed, as part of the Bonus Action (SRD 5.2.1, "Level 7: Instinctive
// Pounce", p. 29). Only BarbarianRage.Enter builds one; the zero value allows nothing.
type InstinctivePounce struct {
	allowed          bool
	speed, halfSpeed int
}

// Allowed reports whether the Barbarian may move.
func (p InstinctivePounce) Allowed() bool { return p.allowed }

// Speed returns the Speed the allowance is computed from, in feet.
func (p InstinctivePounce) Speed() int { return p.speed }

// HalfSpeed returns half the Speed rounded down (SRD 5.2.1, "Round Down", p. 5); 0 when not allowed.
func (p InstinctivePounce) HalfSpeed() int { return p.halfSpeed }

// RageEntered is the outcome of entering a Rage. Only BarbarianRage.Enter builds one.
type RageEntered struct {
	state  BarbarianState
	turn   BarbarianTurnState
	pounce InstinctivePounce
	ended  []content.ConditionKind
	events []Event
}

// State returns the new state: a Rage active, one more use expended.
func (r RageEntered) State() BarbarianState { return r.state }

// Turn returns the new turn state: the Bonus Action spent.
func (r RageEntered) Turn() BarbarianTurnState { return r.turn }

// Pounce returns the Instinctive Pounce allowance.
func (r RageEntered) Pounce() InstinctivePounce { return r.pounce }

// EndedConditions returns the conditions Mindless Rage ends on entering (Charmed,
// Frightened), as a new slice; empty below level 6 or without the Path of the Berserker.
func (r RageEntered) EndedConditions() []content.ConditionKind { return slices.Clone(r.ended) }

// ConcentrationEnds is always true: the Barbarian can't maintain Concentration while raging,
// so the host reports the loss (SRD 5.2.1, "Rage", No Concentration or Spells, p. 29).
func (RageEntered) ConcentrationEnds() bool { return true }

// Events returns a new slice holding the RageStarted event.
func (r RageEntered) Events() []Event { return slices.Clone(r.events) }

// RageNotEntered is the outcome of an entry a rule refused: nothing is spent. Construct it
// with a keyed literal.
type RageNotEntered struct{ Reason BarbarianReason }

func (RageEntered) isRageEntry()    {}
func (RageNotEntered) isRageEntry() {}

// RageExtension is what a Barbarian does on its turn to extend a Rage. The zero value is
// not an extension.
type RageExtension int

// The extensions.
const (
	// RageExtendedByAttackRoll: the Barbarian makes an attack roll against an enemy.
	RageExtendedByAttackRoll RageExtension = iota + 1
	// RageExtendedByForcingSave: the Barbarian forces an enemy to make a saving throw.
	RageExtendedByForcingSave
	// RageExtendedByBonusAction: the Barbarian takes a Bonus Action to extend the Rage.
	RageExtendedByBonusAction
)

// RageExtensionOutcome is the closed outcome of BarbarianRage.Extend. Variants: RageExtended,
// RageAdvanceIgnored.
type RageExtensionOutcome interface{ isRageExtensionOutcome() }

// RageEndOutcome is the closed outcome of BarbarianRage.End. Variants: RageEndApplied,
// RageAdvanceIgnored.
type RageEndOutcome interface{ isRageEndOutcome() }

// RageExtended is the outcome of an extension that applies. Only BarbarianRage.Extend builds one.
type RageExtended struct {
	state BarbarianState
	turn  BarbarianTurnState
}

// State returns the new state: the Rage extended this turn.
func (r RageExtended) State() BarbarianState { return r.state }

// Turn returns the new turn state; the Bonus Action is spent for a Bonus Action extension.
func (r RageExtended) Turn() BarbarianTurnState { return r.turn }

// RageEndApplied is the outcome of an end that applies: the Rage may or may not have ended. Only
// BarbarianRage.End builds one.
type RageEndApplied struct {
	state  BarbarianState
	events []Event
}

// State returns the new state.
func (r RageEndApplied) State() BarbarianState { return r.state }

// Events returns a new slice holding a RageEnded when the Rage ended, else empty.
func (r RageEndApplied) Events() []Event { return slices.Clone(r.events) }

// RageAdvanceIgnored is the outcome of an extension or an end with no effect: the state is
// unchanged. Construct it with a keyed literal.
type RageAdvanceIgnored struct{ Reason BarbarianReason }

func (RageExtended) isRageExtensionOutcome()       {}
func (RageEndApplied) isRageEndOutcome()           {}
func (RageAdvanceIgnored) isRageExtensionOutcome() {}
func (RageAdvanceIgnored) isRageEndOutcome()       {}

// BarbarianRage decides and applies the Rage rules (SRD 5.2.1, "Level 1: Rage", pp. 28-29,
// "Level 15: Persistent Rage", p. 30). It is stateless and pure; the zero value is ready to
// use. Safe for concurrent use.
type BarbarianRage struct{}

// Enter enters a Rage for the Barbarian id. The first reason that holds refuses it, in this
// order: BarbarianNotABarbarian (zero stats), BarbarianRageAlreadyActive,
// BarbarianWearingHeavyArmor, BarbarianIncapacitated, BarbarianNoBonusAction,
// BarbarianNoRageUsesRemaining; nothing is spent. Otherwise one use is expended, the Bonus
// Action is spent, Instinctive Pounce is reported from level 7 and, for the Path of the
// Berserker from level 6, the Charmed and Frightened conditions of facts are reported ended.
//
// It returns an error matching srderr.ErrInvalidInput for a negative speed or a zero id. It
// never panics.
func (BarbarianRage) Enter(id Identifier, stats BarbarianLevelStats, state BarbarianState, turn BarbarianTurnState, facts BarbarianHostFacts, subclass content.SubclassID, speed int) (RageEntry, error) {
	if speed < 0 || id.IsZero() {
		return nil, invalidInput("enter a Rage with a negative Speed or a zero character id")
	}
	var reason BarbarianReason
	switch {
	case stats.IsZero():
		reason = BarbarianNotABarbarian
	case state.rageActive:
		reason = BarbarianRageAlreadyActive
	case facts.WearingHeavyArmor:
		reason = BarbarianWearingHeavyArmor
	case facts.Incapacitated:
		reason = BarbarianIncapacitated
	case !turn.bonusActionAvailable:
		reason = BarbarianNoBonusAction
	case state.RagesRemaining(stats) < 1:
		reason = BarbarianNoRageUsesRemaining
	}
	if reason != 0 {
		return RageNotEntered{Reason: reason}, nil
	}
	state.ragesExpended++
	state.rageActive, state.rageBeganThisTurn, state.rageExtendedThisTurn = true, true, false
	out := RageEntered{state: state, turn: turn.SpendBonusAction(), ended: []content.ConditionKind{}, events: []Event{RageStarted{id: id}}}
	if stats.level.n >= feralInstinctLevel {
		out.pounce = InstinctivePounce{allowed: true, speed: speed, halfSpeed: speed / 2}
	}
	if stats.level.n >= mindlessRageLevel && subclass == BerserkerSubclassID {
		for _, c := range []content.ConditionKind{content.ConditionCharmed, content.ConditionFrightened} {
			if hasCondition(facts.ActiveConditions, c) {
				out.ended = append(out.ended, c)
			}
		}
	}
	return out, nil
}

// Extend extends the active Rage through to the end of the Barbarian's turn with trigger
// (SRD 5.2.1, "Rage", Duration, p. 29). An extension with no effect is RageAdvanceIgnored, the
// first reason that holds in this order: BarbarianNotABarbarian, BarbarianRageNotActive,
// BarbarianRageBeganThisTurn, BarbarianRageLastsWithoutExtension (Persistent Rage, level 15),
// BarbarianRageAlreadyExtended and, for RageExtendedByBonusAction, BarbarianNoBonusAction. A
// Bonus Action extension spends the Bonus Action.
//
// It returns an error matching srderr.ErrInvalidInput for an unknown trigger. It never panics.
func (BarbarianRage) Extend(stats BarbarianLevelStats, state BarbarianState, turn BarbarianTurnState, trigger RageExtension) (RageExtensionOutcome, error) {
	if trigger < RageExtendedByAttackRoll || trigger > RageExtendedByBonusAction {
		return nil, invalidInput("extend a Rage with an unknown trigger")
	}
	var reason BarbarianReason
	switch {
	case stats.IsZero():
		reason = BarbarianNotABarbarian
	case !state.rageActive:
		reason = BarbarianRageNotActive
	case state.rageBeganThisTurn:
		reason = BarbarianRageBeganThisTurn
	case stats.level.n >= persistentRageLevel:
		reason = BarbarianRageLastsWithoutExtension
	case state.rageExtendedThisTurn:
		reason = BarbarianRageAlreadyExtended
	case trigger == RageExtendedByBonusAction && !turn.bonusActionAvailable:
		reason = BarbarianNoBonusAction
	}
	if reason != 0 {
		return RageAdvanceIgnored{Reason: reason}, nil
	}
	state.rageExtendedThisTurn = true
	if trigger == RageExtendedByBonusAction {
		turn = turn.SpendBonusAction()
	}
	return RageExtended{state: state, turn: turn}, nil
}

// End applies cause to the active Rage of the Barbarian id (SRD 5.2.1, "Rage", Duration, p. 29,
// and "Level 15: Persistent Rage", p. 30). With no Rage active it is RageAdvanceIgnored with
// BarbarianRageNotActive (BarbarianNotABarbarian for zero stats). Otherwise:
//   - RageTurnEnded ends the Rage with RageNotExtended unless it began this turn or was extended
//     this turn, or the Barbarian has Persistent Rage; the per-turn marks are cleared;
//   - RageHeavyArmorDonned ends it;
//   - RageBecameIncapacitated ends it, except with Persistent Rage;
//   - RageBecameUnconscious ends it with RageEndedByIncapacitated below level 15 (Unconscious
//     implies Incapacitated) and RageEndedByUnconscious from level 15;
//   - RageElapsedTime ends it when Elapsed is 10 minutes or more.
//
// A Rage that ends produces a RageEnded event. It returns an error matching
// srderr.ErrInvalidInput for a nil or unknown cause, a negative elapsed time or a zero id. It
// never panics.
func (BarbarianRage) End(id Identifier, stats BarbarianLevelStats, state BarbarianState, cause RageEndCause) (RageEndOutcome, error) {
	if id.IsZero() {
		return nil, invalidInput("end a Rage for a zero character id")
	}
	persistent := stats.level.n >= persistentRageLevel
	var reason RageEndReason
	switch c := cause.(type) {
	case RageTurnEnded:
		if !state.rageActive {
			break
		}
		keeps := persistent || state.rageBeganThisTurn || state.rageExtendedThisTurn
		state.rageBeganThisTurn, state.rageExtendedThisTurn = false, false
		if !keeps {
			reason = RageNotExtended
		}
	case RageHeavyArmorDonned:
		reason = RageEndedByHeavyArmor
	case RageBecameIncapacitated:
		if !persistent {
			reason = RageEndedByIncapacitated
		}
	case RageBecameUnconscious:
		reason = RageEndedByIncapacitated
		if persistent {
			reason = RageEndedByUnconscious
		}
	case RageElapsedTime:
		if c.Elapsed < 0 {
			return nil, invalidInput("end a Rage with a negative elapsed time")
		}
		if c.Elapsed >= rageMaximumDuration {
			reason = RageEndedByTenMinutes
		}
	default:
		return nil, invalidInput("end a Rage with a nil or unknown cause")
	}
	switch {
	case stats.IsZero():
		return RageAdvanceIgnored{Reason: BarbarianNotABarbarian}, nil
	case !state.rageActive:
		return RageAdvanceIgnored{Reason: BarbarianRageNotActive}, nil
	case reason == 0:
		return RageEndApplied{state: state, events: []Event{}}, nil
	}
	state.rageActive, state.rageBeganThisTurn, state.rageExtendedThisTurn = false, false, false
	return RageEndApplied{state: state, events: []Event{RageEnded{id: id, reason: reason}}}, nil
}
