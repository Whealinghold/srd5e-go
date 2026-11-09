package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func newBoost(t *testing.T) character.MountedPaceBoost {
	t.Helper()
	b, err := character.NewMountedPaceBoost(character.MustIdentifier("pc-1"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Requirement: doc/specs/exploration.md FR-019, FR-020, FR-055 - a new boost is
// available; Spend gives a spent boost and one event carrying the id (SRD 5.2.1,
// "Travel Pace", p. 12).
// Path: happy.
func TestMountedPaceBoostSpend(t *testing.T) {
	t.Parallel()
	b := newBoost(t)
	if b.State() != character.MountedBoostAvailable || b.CharacterID() != character.MustIdentifier("pc-1") {
		t.Fatalf("new boost = %v", b.State())
	}
	acc, ok := b.Spend().(character.MountedBoostSpendAccepted)
	if !ok || acc.Boost().State() != character.MountedBoostSpent || acc.Event().CharacterID() != b.CharacterID() || acc.Boost().CharacterID() != b.CharacterID() {
		t.Fatalf("Spend = %#v", b.Spend())
	}
	var _ character.MountedPaceBoostEvent = acc.Event()
	if b.State() != character.MountedBoostAvailable {
		t.Error("FR-025: Spend changed its receiver")
	}
}

// Requirement: doc/specs/exploration.md FR-021 - Spend on a spent boost is refused with
// mountsNeedRest, leaves it spent and produces no event (SRD 5.2.1, "Travel Pace", p. 12).
// Path: unhappy (state error).
func TestMountedPaceBoostSpendRefused(t *testing.T) {
	t.Parallel()
	spent := newBoost(t).Spend().(character.MountedBoostSpendAccepted).Boost()
	ref, ok := spent.Spend().(character.MountedBoostSpendRefused)
	if !ok || ref.Reason() != character.MountedBoostMountsNeedRest || ref.Boost() != spent || ref.Boost().State() != character.MountedBoostSpent {
		t.Fatalf("second Spend = %#v", spent.Spend())
	}
}

// Requirement: doc/specs/exploration.md FR-022, FR-023, FR-024, FR-025, cases 4.4b - a
// rest restores a spent boost once, leaves an available one unchanged, and the
// cycle available, spent, available repeats (SRD 5.2.1, "Travel Pace", p. 12).
// Path: unhappy (state error).
func TestMountedPaceBoostRest(t *testing.T) {
	t.Parallel()
	b := newBoost(t)
	if un, ok := b.RestFinished().(character.MountedBoostRestUnchanged); !ok || un.Boost() != b {
		t.Errorf("rest while available = %#v", b.RestFinished())
	}
	spent := b.Spend().(character.MountedBoostSpendAccepted).Boost()
	res, ok := spent.RestFinished().(character.MountedBoostRestRestored)
	if !ok || res.Boost() != b || res.Event().CharacterID() != b.CharacterID() {
		t.Fatalf("rest while spent = %#v", spent.RestFinished())
	}
	var _ character.MountedPaceBoostEvent = res.Event()
	if spent.State() != character.MountedBoostSpent {
		t.Error("FR-025: RestFinished changed its receiver")
	}
	if _, ok := res.Boost().RestFinished().(character.MountedBoostRestUnchanged); !ok {
		t.Error("rest twice: the second must be unchanged")
	}
	// full cycle: spend, spend, rest, spend
	c := b
	c = c.Spend().(character.MountedBoostSpendAccepted).Boost()
	if _, ok := c.Spend().(character.MountedBoostSpendRefused); !ok {
		t.Fatal("second spend must be refused")
	}
	c = c.RestFinished().(character.MountedBoostRestRestored).Boost()
	if _, ok := c.Spend().(character.MountedBoostSpendAccepted); !ok {
		t.Error("spend after the rest must be accepted")
	}
}

// Requirement: doc/specs/exploration.md FR-055, FR-025 - boosts of two ids are unequal,
// the zero Identifier is rejected, and the zero boost has no state.
// Path: unhappy (invalid input).
func TestMountedPaceBoostIdentity(t *testing.T) {
	t.Parallel()
	other, err := character.NewMountedPaceBoost(character.MustIdentifier("pc-2"))
	if err != nil || other == newBoost(t) {
		t.Errorf("boosts of two ids must differ, err %v", err)
	}
	_, err = character.NewMountedPaceBoost(character.Identifier{})
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ExplorationError{Input: character.ExplorationCharacterID}) {
		t.Errorf("zero id err = %v", err)
	}
	if (character.MountedPaceBoost{}).State() != 0 {
		t.Error("the zero boost must have no state")
	}
	if character.MountedBoostAvailable.String() != "available" || character.MountedBoostSpent.String() != "spent" || character.MountedBoostState(0).String() != "unknown" ||
		character.MountedBoostMountsNeedRest.String() != "mounts_need_rest" || character.MountedBoostRefusalReason(0).String() != "unknown" {
		t.Error("String values")
	}
}
