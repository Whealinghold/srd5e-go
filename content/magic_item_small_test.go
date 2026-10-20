package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: miSpec FR-078 - a MagicItemResilience prints its member name
// and an undefined value prints "unknown".
// Path: per case.
func TestMagicItemResilienceString(t *testing.T) {
	t.Parallel()
	cases := map[content.MagicItemResilience]string{
		content.ResilienceResistantToAllDamage: "resistantToAllDamage",
		content.ResilienceNonmagicalDurability: "nonmagicalDurability",
		content.ResilienceImpervious:           "impervious",
		0:                                      "unknown",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("%d: String() = %q, want %q", int(r), got, want)
		}
	}
}

// Requirement: miSpec - NewAttunementRequiredBy returns the zero value, not
// the rejected text, alongside its error.
// Path: error.
func TestNewAttunementRequiredByErrorIsZero(t *testing.T) {
	t.Parallel()
	got, err := content.NewAttunementRequiredBy("  ")
	if err == nil || got != (content.AttunementRequiredBy{}) || got.Prerequisite() != "" {
		t.Errorf("got %#v, %v, want zero value and an error", got, err)
	}
}

// Requirement: miSpec FR-082 - the RollRangeError text names the roll and the
// range (rendering is not a contract, but it must not drop a field).
// Path: error.
func TestRollRangeErrorText(t *testing.T) {
	t.Parallel()
	got := (&content.RollRangeError{Roll: 7, Min: 2, Max: 5}).Error()
	if want := "content: roll 7 is outside 2 to 5"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// Requirement: miSpec FR-079 and FR-090 - the activation time and recharge
// time members print their exact names.
// Path: per case.
func TestActivationAndRechargeTimeNames(t *testing.T) {
	t.Parallel()
	for a, want := range map[content.ActivationTime]string{
		content.ActivationMagicAction: "Magic action", content.ActivationBonusAction: "Bonus Action",
		content.ActivationAction: "Action", content.ActivationReaction: "Reaction",
		content.ActivationDescribedElsewhere: "Described elsewhere",
	} {
		if got := a.String(); got != want {
			t.Errorf("activation %d = %q, want %q", int(a), got, want)
		}
	}
	for r, want := range map[content.ItemRechargeTime]string{
		content.RechargesAtNextDawn: "nextDawn", content.RechargesAtSpecifiedTime: "specifiedTime",
		content.DoesNotRecharge: "doesNotRecharge",
	} {
		if got := r.String(); got != want {
			t.Errorf("recharge %d = %q, want %q", int(r), got, want)
		}
	}
}

// Requirement: miSpec FR-086 - FixedTools.Equal compares the ids in order.
// Path: per case.
func TestFixedToolsEqual(t *testing.T) {
	t.Parallel()
	mk := func(ids ...content.ItemID) content.FixedTools {
		f, err := content.NewFixedTools(ids)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	if !mk("a", "b").Equal(mk("a", "b")) {
		t.Error("equal sets compare unequal")
	}
	for name, g := range map[string]content.FixedTools{
		"order": mk("b", "a"), "content": mk("a", "c"), "shorter": mk("a"), "longer": mk("a", "b", "c"),
	} {
		if mk("a", "b").Equal(g) {
			t.Errorf("%s: different sets compare equal", name)
		}
	}
}

// Requirement: miSpec FR-090 - RegainsFixed keeps its charge count, and
// MagicItemFacts accepts every valid ChargeRecovery variant but no foreign one.
// Path: per case.
func TestChargeRecoveryValidity(t *testing.T) {
	t.Parallel()
	if r, err := content.NewRegainsFixed(3); err != nil || r.Charges() != 3 {
		t.Errorf("NewRegainsFixed(3) = %v, %v", r, err)
	}
	fixed, _ := content.NewRegainsFixed(2)
	rolled, _ := content.NewRegainsRolled(dice.MustParseExpression("1d4"))
	for name, c := range map[string]content.ChargeRecovery{
		"all": content.RegainsAll{}, "none": content.NoChargeRecovery{}, "fixed": fixed, "rolled": rolled,
	} {
		if _, err := content.NewMagicItemFacts("item_x", content.MagicItemFactsFields{ChargeRecovery: c}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A struct embedding a variant satisfies the sealed interface without being one.
	type foreign struct{ content.RegainsAll }
	_, err := content.NewMagicItemFacts("item_x", content.MagicItemFactsFields{ChargeRecovery: foreign{}})
	wantBlockError(t, err, "MagicItemFacts", content.BlockInvalidMember)
}
