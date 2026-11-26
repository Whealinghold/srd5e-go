package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-015, FR-062 - a value the constructors cannot build (a source naming a feature and an item, a replacement missing its kind or source, an entry holding a means and a replacement or a negative means) is invalid.
// Path: unhappy (invalid input).
func TestAttackMeansFilesRejectStatesOnlyAStructLiteralReaches(t *testing.T) {
	t.Parallel()
	feature, item := content.MustFeatureID("breath_weapon"), content.MustItemID("acid")
	both := AttackReplacementSource{feature: feature, item: item}
	if both.Valid() {
		t.Error("a source naming a feature and an item is valid")
	}
	if _, ok := both.Feature(); ok {
		t.Error("an invalid source reports its feature")
	}
	if _, ok := both.Item(); ok {
		t.Error("an invalid source reports its item")
	}
	good := ReplacementByFeature(feature)
	for name, r := range map[string]AttackReplacement{
		"no source": {kind: AttackReplacementKindReplace}, "no kind": {source: good},
	} {
		if r.Valid() {
			t.Errorf("replacement with %s is valid", name)
		}
	}
	valid, _ := NewAttackReplacement(AttackReplacementKindReplace, good)
	for name, e := range map[string]AttackEntry{
		"means and replacement": {means: AttackMeansPactWeapon, replacement: valid},
		"negative means":        {means: -1, replacement: valid},
		"negative means alone":  {means: -1},
	} {
		if e.Valid() {
			t.Errorf("entry with %s is valid", name)
		}
	}
	if !(AttackEntry{replacement: valid}).Valid() || !(AttackEntry{means: AttackMeansUnarmedStrike}).Valid() {
		t.Error("a plain entry is invalid")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-065 - an AssistedProficiency naming a skill and a tool, or a negative skill with a valid tool, is invalid.
// Path: unhappy (invalid input).
func TestAssistedProficiencyRejectsStatesOnlyAStructLiteralReaches(t *testing.T) {
	t.Parallel()
	tool := content.MustItemID("thieves_tools")
	for name, p := range map[string]AssistedProficiency{
		"skill and tool": {skill: content.SkillAthletics, tool: tool}, "negative skill and tool": {skill: -1, tool: tool},
		"skill and bad tool": {skill: content.SkillAthletics, tool: "Not Valid"},
	} {
		if p.Valid() {
			t.Errorf("%s proficiency is valid", name)
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-014, FR-019 - a progress with a valid allowance but no attacker is invalid.
// Path: unhappy (invalid input).
func TestAttackActionProgressNeedsAnAttacker(t *testing.T) {
	t.Parallel()
	if (AttackActionProgress{allowance: BaselineAttackAllowance()}).Valid() {
		t.Error("a progress with no attacker is valid")
	}
	if (AttackActionProgress{attacker: "hero"}).Valid() {
		t.Error("a progress with no allowance is valid")
	}
}
