package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-062, FR-063).
// Path: per case.
func TestMagicItemIdentification(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Identifying a Magic Item", p. 102; "Cursed Items", p. 206`
	learned := character.MagicItemPropertiesLearned{Method: character.IdentifyByShortRestFocus, Curse: character.CurseNotRevealed}
	notLearned := func(r character.IdentificationNotLearnedReason) character.IdentificationOutcome {
		return character.MagicItemPropertiesNotLearned{Reason: r}
	}
	cases := []struct {
		name, requirement, path string
		focus                   character.ShortRestFocus
		want                    character.IdentificationOutcome
	}{
		{"completed, in contact", p, "happy", focusOf(t, "i", character.RestPurposeLearnProperties, character.InContact, true), learned},
		{"interrupted", p, "unhappy (state error)", focusOf(t, "i", character.RestPurposeLearnProperties, character.InContact, false), notLearned(character.IdentifyRestInterrupted)},
		{"different instance", p, "unhappy (state error)", focusOf(t, "j", character.RestPurposeLearnProperties, character.InContact, true), notLearned(character.IdentifyRestNotFocusedOnItem)},
		{"no contact", p, "unhappy (state error)", focusOf(t, "i", character.RestPurposeLearnProperties, character.NotInContact, true), notLearned(character.IdentifyRestNotFocusedOnItem)},
		{"attune purpose", p, "unhappy (state error)", focusOf(t, "i", character.RestPurposeAttune, character.InContact, true), notLearned(character.IdentifyWrongRestPurpose)},
		{"wrong purpose before wrong instance", p, "unhappy (state error)", focusOf(t, "j", character.RestPurposeEndAttunement, character.NotInContact, false), notLearned(character.IdentifyWrongRestPurpose)},
		{"wrong instance before interruption", p, "unhappy (state error)", focusOf(t, "j", character.RestPurposeLearnProperties, character.InContact, false), notLearned(character.IdentifyRestNotFocusedOnItem)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (character.MagicItemIdentification{}).ByShortRest(inst("i"), c.focus); got != c.want {
				t.Errorf("= %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
	if got := (character.MagicItemIdentification{}).BySpell(); got != (character.MagicItemPropertiesLearned{Method: character.IdentifyBySpell, Curse: character.CurseNotRevealed}) {
		t.Errorf("BySpell = %#v", got)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-062 - the zero value of each identification enum is not a valid member, and the members are distinct.
// Path: unhappy (invalid input).
func TestIdentificationEnumsAreNonZeroAndDistinct(t *testing.T) {
	t.Parallel()
	for name, vals := range map[string][]int{
		"method": {int(character.IdentifyByShortRestFocus), int(character.IdentifyBySpell)},
		"reason": {int(character.IdentifyWrongRestPurpose), int(character.IdentifyRestNotFocusedOnItem), int(character.IdentifyRestInterrupted)},
		"curse":  {int(character.CurseNotRevealed)},
	} {
		seen := map[int]bool{0: true}
		for _, v := range vals {
			if seen[v] {
				t.Errorf("%s: value %d is zero or repeated", name, v)
			}
			seen[v] = true
		}
	}
	for i, want := range []int{1, 2} {
		if got := []int{int(character.IdentifyByShortRestFocus), int(character.IdentifyBySpell)}[i]; got != want {
			t.Errorf("method %d = %d", i, got)
		}
	}
}
