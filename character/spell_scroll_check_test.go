package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// required4 is the check of a level 4 Wizard spell for a Wizard 5 / Cleric 1: DC 14, options [Int, Wis].
func required4(tb testing.TB) character.ScrollCheckRequired {
	tb.Helper()
	p := profile(readerFacts(3, classRoute("wizard", ability.Intelligence), classRoute("cleric", ability.Wisdom)))
	return character.AssessScrollReading(p, scrollOf(tb, 4, character.FoundScroll{}, "wizard", "cleric"), character.ItemIntact).(character.ScrollCheckRequired)
}

// Requirement: spell-scroll-use.md FR-010, FR-011, FR-021 (SRD 5.2.1, "Spell Scroll", p. 244; "D20 Tests", p. 6)
// Path: per case (C1, C2).
func TestResolveScrollCheckBoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		total  int
		passed bool
	}{
		{"C1 total equal to the DC passes", 14, true},
		{"total one above passes", 15, true},
		{"C2 total one below erases", 13, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := required4(t)
			got, err := character.ResolveScrollCheck(req, intOption(), abilityCheck(t, d20.AbilityCheck, c.total, 14))
			if err != nil {
				t.Fatal(err)
			}
			if c.passed {
				p, ok := got.(character.ScrollCheckPassed)
				want, _ := req.TermsFor(intOption())
				if !ok || p.Terms() != want {
					t.Errorf("result = %#v, want ScrollCheckPassed with the Int terms", got)
				}
				return
			}
			e, ok := got.(character.ScrollSpellErased)
			if !ok || e.Disposition() != character.DispositionErased {
				t.Errorf("result = %#v, want ScrollSpellErased with DispositionErased", got)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-010, C5 - the terms of a pass carry the ability of the chosen option.
// Path: unhappy (rule interaction).
func TestResolveScrollCheckPassUsesTheChosenOption(t *testing.T) {
	t.Parallel()
	got, err := character.ResolveScrollCheck(required4(t), wisOption(), abilityCheck(t, d20.AbilityCheck, 20, 14))
	if p, ok := got.(character.ScrollCheckPassed); err != nil || !ok || p.Terms().SpellcastingAbility() != ability.Wisdom {
		t.Errorf("result = %#v, %v, want Wisdom terms", got, err)
	}
}

// Requirement: per case
// Path: unhappy (invalid input).
func TestResolveScrollCheckMismatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		option character.ScrollCheckOption
		result d20.CheckResult
	}{
		{"C3 option not offered", intArcana(), abilityCheck(t, d20.AbilityCheck, 14, 14)},
		{"C4 target 13 against DC 14", intOption(), abilityCheck(t, d20.AbilityCheck, 14, 13)},
		{"C4 target 15 against DC 14", intOption(), abilityCheck(t, d20.AbilityCheck, 14, 15)},
		{"C4 saving throw", intOption(), abilityCheck(t, d20.SavingThrow, 14, 14)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: spell-scroll-use.md FR-009
			got, err := character.ResolveScrollCheck(required4(t), c.option, c.result)
			if got != nil || !errors.Is(err, character.ErrScrollCheckMismatch) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("got %#v, %v, want nil and ErrScrollCheckMismatch", got, err)
			}
		})
	}
	// The zero requirement offers nothing.
	if _, err := character.ResolveScrollCheck(character.ScrollCheckRequired{}, intOption(), abilityCheck(t, d20.AbilityCheck, 10, 10)); !errors.Is(err, character.ErrScrollCheckMismatch) {
		t.Errorf("zero requirement: err = %v", err)
	}
}

// Requirement: spell-scroll-use.md FR-016 (SRD 5.2.1, "Spell Scroll", p. 244) - S1 to S3, and an invalid progress keeps the scroll.
// Path: per case.
func TestSettleScrollReading(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		progress character.ScrollCastProgress
		want     character.ScrollDisposition
	}{
		{"S1 completed", character.ProgressCompleted, character.DispositionConsumed},
		{"S2 interrupted", character.ProgressInterrupted, character.DispositionRetained},
		{"S3 not started", character.ProgressNotStarted, character.DispositionRetained},
		{"zero progress", 0, character.DispositionRetained},
	}
	for _, c := range cases {
		if got := character.SettleScrollReading(c.progress); got != c.want {
			t.Errorf("%s: SettleScrollReading(%v) = %v, want %v", c.name, c.progress, got, c.want)
		}
	}
}

// Requirement: spell-scroll-use.md FR-016 - the enums print their names and "unknown" outside their range.
// Path: unhappy (boundary).
func TestScrollEnumStrings(t *testing.T) {
	t.Parallel()
	cases := []struct{ got, want string }{
		{character.DispositionRetained.String(), "retained"}, {character.DispositionConsumed.String(), "consumed"}, {character.DispositionErased.String(), "erased"},
		{character.ScrollDisposition(0).String(), "unknown"}, {character.ScrollDisposition(4).String(), "unknown"},
		{character.ProgressNotStarted.String(), "not_started"}, {character.ProgressInterrupted.String(), "interrupted"}, {character.ProgressCompleted.String(), "completed"},
		{character.ScrollCastProgress(0).String(), "unknown"}, {character.ScrollCastProgress(4).String(), "unknown"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("String() = %q, want %q", c.got, c.want)
		}
	}
}
