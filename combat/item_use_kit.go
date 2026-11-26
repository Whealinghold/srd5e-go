package combat

import "slices"

// healersKitUses is the number of uses a Healer's Kit holds (SRD 5.2.1, "Healer's Kit", p. 97).
const healersKitUses = 10

// HealersKit is a Healer's Kit with its remaining uses, 0 to 10 (SRD 5.2.1, "Healer's Kit",
// p. 97). No operation adds uses. Immutable and comparable. The zero value is an empty kit.
type HealersKit struct{ uses int }

// NewHealersKit returns a kit with uses remaining. It returns an *ItemUseError (matching
// srderr.ErrInvalidInput) with ItemUseFieldUsesRemaining outside 0 to 10. It never panics.
func NewHealersKit(uses int) (HealersKit, error) {
	if uses < 0 || uses > healersKitUses {
		return HealersKit{}, itemUseInvalid(ItemUseFieldUsesRemaining, int64(uses))
	}
	return HealersKit{uses}, nil
}

// MustHealersKit is NewHealersKit for a value known to be valid. It panics with the error
// NewHealersKit returns, and only then.
func MustHealersKit(uses int) HealersKit {
	k, err := NewHealersKit(uses)
	if err != nil {
		panic(err)
	}
	return k
}

// FreshHealersKit returns a kit with all ten uses.
func FreshHealersKit() HealersKit { return HealersKit{healersKitUses} }

// UsesRemaining returns the uses left.
func (k HealersKit) UsesRemaining() int { return k.uses }

// HealersKitOutcome is the outcome of using a kit to stabilize. Variants: HealersKitUsed,
// HealersKitEmpty, HealersKitRefused.
type HealersKitOutcome interface{ isHealersKitOutcome() }

// HealersKitUsed: the patient is Stable and one use was expended. Not comparable.
type HealersKitUsed struct {
	kit    HealersKit
	status HitPointStatus
	events []Event
}

// Kit returns the kit with one fewer use.
func (u HealersKitUsed) Kit() HealersKit { return u.kit }

// Status returns the patient's new status: Stable, with both tallies reset.
func (u HealersKitUsed) Status() HitPointStatus { return u.status }

// Cost returns ItemActionCostUtilizeAction: no die and no Wisdom (Medicine) check.
func (HealersKitUsed) Cost() ItemActionCost { return ItemUseRules{}.CostOf(ItemUseHealersKit) }

// Events returns a copy of the events in order: the hit point events of Stabilize, then one
// HealersKitUseSpent.
func (u HealersKitUsed) Events() []Event { return slices.Clone(u.events) }

// HealersKitEmpty: the kit has no uses; the patient was not examined.
type HealersKitEmpty struct{}

// HealersKitRefused: the patient cannot be stabilized (not Dying, or Dead); the kit keeps its
// use, because a use is expended "to stabilize" (Q-11).
type HealersKitRefused struct{ Rejection HitPointRejection }

func (HealersKitUsed) isHealersKitOutcome()    {}
func (HealersKitEmpty) isHealersKitOutcome()   {}
func (HealersKitRefused) isHealersKitOutcome() {}

// UseToStabilize has user spend one use of the kit to stabilize the Dying patient with
// status: HitPointStatus.Stabilize decides, and its events come first. A kit with no uses
// returns HealersKitEmpty before the patient is examined. A patient that is not Dying
// (Alive, Stable) or is Dead gives HealersKitRefused with the HitPointRejection (NotDying,
// CreatureIsDead) and keeps the use. The kit neither needs nor consumes the Help action, and
// the StabilizingCheck path stays available when no kit is used.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) for a zero user or patient or
// a zero status. It never panics.
func (k HealersKit) UseToStabilize(user, patient CombatantID, status HitPointStatus) (HealersKitOutcome, error) {
	switch {
	case user.IsZero():
		return nil, itemUseInvalid(ItemUseFieldUser, 0)
	case patient.IsZero():
		return nil, itemUseInvalid(ItemUseFieldPatient, 0)
	case status.IsZero():
		return nil, itemUseInvalid(ItemUseFieldStatus, 0)
	case k.uses == 0:
		return HealersKitEmpty{}, nil
	}
	tr, err := status.Stabilize(patient)
	if err != nil {
		return nil, err
	}
	switch t := tr.(type) {
	case HitPointTransitionRefused:
		return HealersKitRefused(t), nil
	case HitPointTransitionApplied:
		events := append(asEvents(t.events), HealersKitUseSpent{itemUseEventBase{user, patient}})
		return HealersKitUsed{HealersKit{k.uses - 1}, t.status, events}, nil
	default:
		panic(unreachableError("hit point transition", tr))
	}
}
