package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-001 - AttackMeans has exactly three values and no value stands for a
// spell attack; an exhaustive switch over it needs no default; the zero value and undefined values are invalid.
// Path: happy.
func TestAttackMeansHasThreeValues(t *testing.T) {
	t.Parallel()
	describe := func(m combat.AttackMeans) string {
		switch m {
		case combat.AttackMeansPactWeapon:
			return "pact_weapon"
		case combat.AttackMeansOtherWeapon:
			return "other_weapon"
		case combat.AttackMeansUnarmedStrike:
			return "unarmed_strike"
		}
		return "unknown"
	}
	for _, m := range []combat.AttackMeans{combat.AttackMeansPactWeapon, combat.AttackMeansOtherWeapon, combat.AttackMeansUnarmedStrike} {
		if !m.Valid() || m.String() != describe(m) {
			t.Errorf("%d: Valid = %v, String = %q, want true, %q", m, m.Valid(), m.String(), describe(m))
		}
	}
	for _, m := range []combat.AttackMeans{0, 4, -1} {
		if m.Valid() || m.String() != "unknown" {
			t.Errorf("AttackMeans(%d) = valid %v, %q, want invalid, unknown", m, m.Valid(), m.String())
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md section 3 - an AttackReplacement has a kind (replace or forgo) and a closed
// source (a feature or an item); a bad kind, a bad id, an id of the other type or the zero value is rejected.
// Path: per case.
func TestNewAttackReplacement(t *testing.T) {
	t.Parallel()
	feature, item := content.MustFeatureID("breath_weapon"), content.MustItemID("acid")
	cases := []struct {
		name, requirement, path string
		kind                    combat.AttackReplacementKind
		source                  combat.AttackReplacementSource
		ok                      bool
	}{
		{"replace by feature", `SRD 5.2.1, "Breath Weapon", p. 84`, "happy", combat.AttackReplacementKindReplace, combat.ReplacementByFeature(feature), true},
		{"replace by item", `SRD 5.2.1, "Acid", p. 94`, "happy", combat.AttackReplacementKindReplace, combat.ReplacementByItem(item), true},
		{"forgo by feature", `SRD 5.2.1, "Quivering Palm", p. 52`, "happy", combat.AttackReplacementKindForgo, combat.ReplacementByFeature(feature), true},
		{"zero kind", "package design", "unhappy (invalid input)", 0, combat.ReplacementByFeature(feature), false},
		{"undefined kind", "package design", "unhappy (boundary)", 3, combat.ReplacementByFeature(feature), false},
		{"zero source", "package design", "unhappy (invalid input)", combat.AttackReplacementKindReplace, combat.AttackReplacementSource{}, false},
		{"invalid feature id", "package design", "unhappy (invalid input)", combat.AttackReplacementKindReplace, combat.ReplacementByFeature("Not Valid"), false},
		{"invalid item id", "package design", "unhappy (invalid input)", combat.AttackReplacementKindReplace, combat.ReplacementByItem(""), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewAttackReplacement(c.kind, c.source)
			if (err == nil) != c.ok || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || r.Valid() != c.ok {
				t.Errorf("NewAttackReplacement = %v, %v, want ok %v [%s]", r, err, c.ok, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md section 3 - a source reports the feature or the item it names, never both,
// and the kind accessor round-trips.
// Path: happy.
func TestAttackReplacementSourceAccessors(t *testing.T) {
	t.Parallel()
	feature, item := content.MustFeatureID("breath_weapon"), content.MustItemID("acid")
	f, i := combat.ReplacementByFeature(feature), combat.ReplacementByItem(item)
	if got, ok := f.Feature(); got != feature || !ok {
		t.Errorf("Feature() = %v, %v", got, ok)
	}
	if _, ok := f.Item(); ok {
		t.Error("a feature source reports an item")
	}
	if got, ok := i.Item(); got != item || !ok {
		t.Errorf("Item() = %v, %v", got, ok)
	}
	if _, ok := i.Feature(); ok {
		t.Error("an item source reports a feature")
	}
	both := combat.AttackReplacementSource{}
	if _, ok := both.Feature(); ok || both.Valid() {
		t.Error("the zero source is valid")
	}
	r, _ := combat.NewAttackReplacement(combat.AttackReplacementKindForgo, i)
	if r.Kind() != combat.AttackReplacementKindForgo || r.Source() != i || r.Kind().String() != "forgo" ||
		combat.AttackReplacementKindReplace.String() != "replace" || combat.AttackReplacementKind(0).String() != "unknown" {
		t.Error("kind or source accessors disagree")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-015, FR-062 - an AttackEntry is exactly one valid means or one valid
// replacement; the zero entry and an entry with an invalid part are invalid.
// Path: unhappy (invalid input).
func TestAttackEntryValidity(t *testing.T) {
	t.Parallel()
	r := replacementOf(t)
	for name, e := range map[string]combat.AttackEntry{
		"means": combat.MadeAttack(combat.AttackMeansUnarmedStrike), "replacement": combat.ReplacedAttack(r),
	} {
		if !e.Valid() {
			t.Errorf("%s entry is invalid", name)
		}
	}
	for name, e := range map[string]combat.AttackEntry{
		"zero": {}, "bad means": combat.MadeAttack(9), "bad replacement": combat.ReplacedAttack(combat.AttackReplacement{}),
	} {
		if e.Valid() {
			t.Errorf("%s entry is valid", name)
		}
	}
	made, rep := combat.MadeAttack(combat.AttackMeansPactWeapon), combat.ReplacedAttack(r)
	if made.IsReplacement() || made.Means() != combat.AttackMeansPactWeapon || !rep.IsReplacement() || rep.Replacement() != r || rep.Means() != 0 {
		t.Error("entry accessors disagree")
	}
}
