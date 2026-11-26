package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestEffectSourceIsMagical(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		source                  combat.EffectSource
		want                    bool
	}{
		{"a spell is a magical effect", meSpec + `FR-001; SRD 5.2.1, "Magical Effect", p. 185; "Spell", p. 188`, "happy", meSpell("bless"), true},
		{"a magic item is a magical effect", meSpec + `FR-002; SRD 5.2.1, "Magical Effect", p. 185`, "happy", meItem("ring_of_x"), true},
		{"a rule-labeled phenomenon is a magical effect", meSpec + `FR-003; SRD 5.2.1, "Magical Effect", p. 185`, "happy", meLabeled("hag_curse"), true},
		{"a non-magical source is not a magical effect", meSpec + `FR-004; SRD 5.2.1, "Magical Effect", p. 185`, "unhappy (boundary)", meNonMagical("rockfall"), false},
		{"the zero source is not a magical effect", meSpec + `FR-004; package: the zero value is invalid`, "unhappy (boundary)", combat.EffectSource{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.source.IsMagical(); got != c.want {
				t.Errorf("IsMagical() = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-005 - sources are equal by value, and a spell and a magic item over the same id text are different variants and not equal.
// Path: unhappy (rule interaction).
func TestEffectSourceEqualityByValue(t *testing.T) {
	t.Parallel()
	spell, item := meSpell("shared_name"), meItem("shared_name")
	if spell != meSpell("shared_name") {
		t.Error("equal spell sources must be ==")
	}
	if spell == item || spell == meLabeled("shared_name") || meLabeled("shared_name") == meNonMagical("shared_name") {
		t.Error("sources of different kinds over the same id must differ")
	}
	if spell == meSpell("other") {
		t.Error("sources with different ids must differ")
	}
	if spell.Kind() != combat.EffectSourceSpell || spell.ID() != "shared_name" || spell.IsZero() || !(combat.EffectSource{}).IsZero() {
		t.Errorf("accessors: kind %v id %q", spell.Kind(), spell.ID())
	}
}

// Requirement: doc/specs/magical-effects.md section 5 - each source constructor rejects an invalid content id with ErrInvalidInput and returns the zero source.
// Path: unhappy (invalid input).
func TestEffectSourceConstructorsRejectInvalidIDs(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func() (combat.EffectSource, error){
		"spell":    func() (combat.EffectSource, error) { return combat.SpellSource(content.SpellID("")) },
		"item":     func() (combat.EffectSource, error) { return combat.MagicItemSource(content.MagicItemID("Bad Id")) },
		"labeled":  func() (combat.EffectSource, error) { return combat.LabeledMagicalSource(content.FeatureID("")) },
		"nonmagic": func() (combat.EffectSource, error) { return combat.NonMagicalSource(content.FeatureID("X")) },
	} {
		src, err := f()
		if !errors.Is(err, srderr.ErrInvalidInput) || !src.IsZero() {
			t.Errorf("%s: got %v, %v, want the zero source and ErrInvalidInput", name, src, err)
		}
	}
}

// Requirement: none - pins the String values of the source kinds.
// Path: happy.
func TestEffectSourceKindString(t *testing.T) {
	t.Parallel()
	for k, want := range map[combat.EffectSourceKind]string{
		combat.EffectSourceSpell: "spell", combat.EffectSourceMagicItem: "magic_item",
		combat.EffectSourceLabeledMagical: "labeled_magical", combat.EffectSourceNonMagical: "non_magical", 0: "unknown", 5: "unknown",
	} {
		if got := k.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", int(k), got, want)
		}
	}
}
