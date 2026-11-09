package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: per case (SRD 5.2.1, "Copying a Scroll into a Spellbook", p. 244; "Copying a Spell into the Book", p. 78)
// Path: per case.
func TestAssessScrollCopying(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, req, path string
		level           int
		classes         []string
		state           character.MagicItemUseState
		want            character.ScrollCopyAssessmentResult
	}{
		{"P3 cantrip is not copyable", "FR-028", "unhappy (invalid input)", 0, []string{"wizard"}, character.ItemIntact, character.ScrollNotCopyable{Reason: character.Cantrip}},
		{"P3 Cleric spell is not copyable", "FR-028", "unhappy (invalid input)", 2, []string{"cleric"}, character.ItemIntact, character.ScrollNotCopyable{Reason: character.NotAWizardSpell}},
		{"a non-Wizard cantrip is reported as not a Wizard spell", "FR-028", "unhappy (rule interaction)", 0, []string{"cleric"}, character.ItemIntact, character.ScrollNotCopyable{Reason: character.NotAWizardSpell}},
		{"P4 consumed scroll", "FR-028, FR-013", "unhappy (state error)", 2, []string{"wizard"}, character.ItemConsumed, character.ScrollAlreadyUsed{}},
		{"consumed wins over not a Wizard spell", "FR-028", "unhappy (state error)", 2, []string{"cleric"}, character.ItemConsumed, character.ScrollAlreadyUsed{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.AssessScrollCopying(scrollOf(t, c.level, character.FoundScroll{}, c.classes...), c.state)
			if got != c.want {
				t.Errorf("result = %#v, want %#v [spell-scroll-use.md %s %s]", got, c.want, c.req, c.path)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-029 - the copy check is Intelligence (Arcana) against 10 plus the spell level.
// Path: unhappy (boundary).
func TestScrollCopyCheckRequiredDCAndOption(t *testing.T) {
	t.Parallel()
	for level, dc := range map[int]int{1: 11, 9: 19} {
		r := character.AssessScrollCopying(scrollOf(t, level, character.FoundScroll{}, "wizard"), character.ItemIntact).(character.ScrollCopyCheckRequired)
		if r.DC() != dc || r.Option() != intArcana() {
			t.Errorf("level %d: DC %d option %v, want DC %d Int (Arcana)", level, r.DC(), r.Option(), dc)
		}
	}
}

// Requirement: per case (P1, P2)
// Path: unhappy (boundary).
func TestResolveScrollCopy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		level, total  int
		wantCopied    bool
		requirementID string
	}{
		{"P1 level 1, total 11 equals the DC", 1, 11, true, "FR-030"},
		{"level 1, total 10 is one below", 1, 10, false, "FR-030"},
		{"P2 level 9, total 18 against DC 19", 9, 18, false, "FR-030"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := character.AssessScrollCopying(scrollOf(t, c.level, character.FoundScroll{}, "wizard"), character.ItemIntact).(character.ScrollCopyCheckRequired)
			got, err := character.ResolveScrollCopy(req, abilityCheck(t, d20.AbilityCheck, c.total, req.DC()))
			if err != nil || got.SpellCopied() != c.wantCopied || !got.ScrollDestroyed() {
				t.Errorf("ResolveScrollCopy = copied %v destroyed %v, %v; want copied %v, destroyed true", got.SpellCopied(), got.ScrollDestroyed(), err, c.wantCopied)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-030 - a wrong kind or target is ErrScrollCheckMismatch.
// Path: unhappy (invalid input).
func TestResolveScrollCopyMismatch(t *testing.T) {
	t.Parallel()
	req := character.AssessScrollCopying(scrollOf(t, 2, character.FoundScroll{}, "wizard"), character.ItemIntact).(character.ScrollCopyCheckRequired)
	for name, res := range map[string]d20.CheckResult{"saving throw": abilityCheck(t, d20.SavingThrow, 12, 12), "wrong target": abilityCheck(t, d20.AbilityCheck, 12, 13)} {
		got, err := character.ResolveScrollCopy(req, res)
		if !errors.Is(err, character.ErrScrollCheckMismatch) || got != (character.ScrollCopyResolved{}) {
			t.Errorf("%s: got %+v, %v", name, got, err)
		}
	}
}

// Requirement: spell-scroll-use.md FR-028 - ScrollNotCopyableReason prints its name.
// Path: unhappy (boundary).
func TestScrollNotCopyableReasonString(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.ScrollNotCopyableReason]string{character.NotAWizardSpell: "not_a_wizard_spell", character.Cantrip: "cantrip", 0: "unknown", 3: "unknown"} {
		if r.String() != want {
			t.Errorf("%d.String() = %q, want %q", int(r), r.String(), want)
		}
	}
}
