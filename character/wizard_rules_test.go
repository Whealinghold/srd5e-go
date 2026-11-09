package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-wizard.md FR-018 (table 4.2.2, H-01 to H-13) - the highest Spell Level with a Spell Slot and whether it rises, derived from
// the real Wizard definition.
// Path: per case (happy / boundary at the rises).
func TestWizardRulesHighestSpellLevel(t *testing.T) {
	t.Parallel()
	def := wzDef(t)
	for _, c := range []struct {
		label       string
		level, high int
		rises       bool
	}{
		{"H-01", 1, 1, true}, {"H-02", 2, 1, false}, {"H-03", 3, 2, true}, {"H-04", 4, 2, false}, {"H-05", 5, 3, true}, {"H-06", 7, 4, true},
		{"H-07", 9, 5, true}, {"H-08", 11, 6, true}, {"H-09", 13, 7, true}, {"H-10", 15, 8, true}, {"H-11", 16, 8, false}, {"H-12", 17, 9, true}, {"H-13", 20, 9, false},
	} {
		high, err := character.WizardRules{}.HighestSpellLevel(def, c.level)
		rises, err2 := character.WizardRules{}.NewSpellLevelAt(def, c.level)
		if err != nil || err2 != nil || high != c.high || rises != c.rises {
			t.Errorf("%s level %d = %d, %v (%v, %v); want %d, %v", c.label, c.level, high, rises, err, err2, c.high, c.rises)
		}
	}
	for _, level := range []int{0, 21} {
		if _, err := (character.WizardRules{}).HighestSpellLevel(def, level); !srderrIsInvalid(err) {
			t.Errorf("level %d err = %v, want ErrInvalidInput", level, err)
		}
		if _, err := (character.WizardRules{}).NewSpellLevelAt(def, level); !srderrIsInvalid(err) {
			t.Errorf("NewSpellLevelAt level %d err = %v, want ErrInvalidInput", level, err)
		}
	}
}

func srderrIsInvalid(err error) bool { return err != nil && errorsIs(err, srderr.ErrInvalidInput) }

// Requirement: doc/specs/class-wizard.md FR-029, FR-033 - cantrips gained (3 at level 1, 1 at 4 and 10), cantrips pending (derived, never below 0) and the two
// spells each Wizard level after 1 owes.
// Path: per case (happy / boundary: levels 1, 3, 4, 5, 10; below, at and above the column).
func TestWizardRulesCantripsAndSpellsOwed(t *testing.T) {
	t.Parallel()
	def := wzDef(t)
	r := character.WizardRules{}
	for level, want := range map[int]int{1: 3, 2: 0, 3: 0, 4: 1, 5: 0, 9: 0, 10: 1, 11: 0, 20: 0} {
		if got, err := r.CantripsGainedAt(def, level); err != nil || got != want {
			t.Errorf("CantripsGainedAt(%d) = %d, %v; want %d", level, got, err, want)
		}
	}
	for _, c := range []struct{ level, known, want int }{{1, 0, 3}, {1, 2, 1}, {1, 3, 0}, {1, 5, 0}, {4, 3, 1}, {10, 4, 1}} {
		if got, err := r.CantripsPending(def, c.level, c.known); err != nil || got != c.want {
			t.Errorf("CantripsPending(%d, %d) = %d, %v; want %d", c.level, c.known, got, err, c.want)
		}
	}
	for _, level := range []int{0, 21} {
		if _, err := r.CantripsGainedAt(def, level); !srderrIsInvalid(err) {
			t.Errorf("CantripsGainedAt(%d) err = %v", level, err)
		}
		if _, err := r.CantripsPending(def, level, 0); !srderrIsInvalid(err) {
			t.Errorf("CantripsPending(%d) err = %v", level, err)
		}
	}
	for level, want := range map[int]int{-1: 0, 0: 0, 1: 0, 2: 2, 20: 2, 21: 0} {
		if got := r.SpellsOwedAtLevelUp(level); got != want {
			t.Errorf("SpellsOwedAtLevelUp(%d) = %d, want %d", level, got, want)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-045 - Arcane Recovery recovers at most half the Wizard level rounded up (1 at levels 1-2, 2 at 3-4, 10 at 19-20).
// Path: per case (happy / boundary).
func TestArcaneRecoveryMaxLevels(t *testing.T) {
	t.Parallel()
	for level, want := range map[int]int{0: 0, 1: 1, 2: 1, 3: 2, 4: 2, 9: 5, 11: 6, 19: 10, 20: 10} {
		if got := (character.ArcaneRecovery{}).MaxLevels(level); got != want {
			t.Errorf("MaxLevels(%d) = %d, want %d", level, got, want)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-046 to FR-049, FR-052 (table 4.7.1) - an Arcane Recovery choice is parsed against the expended slots and the maximum: accepted, or
// rejected with SlotLevelTooHigh, SlotNotExpended or ExceedsMaximum in that order, naming the lowest offending level; empty counts are ignored.
// Path: per case (happy / unhappy: boundary at the maximum and at level 5 versus 6).
func TestNewArcaneRecoveryChoice(t *testing.T) {
	t.Parallel()
	type slots = map[int]int
	for _, c := range []struct {
		label            string
		chosen, expended slots
		max              int
		want             character.ArcaneRecoveryRejection
		levels           int
	}{
		{"AR-01", slots{1: 1}, slots{1: 1}, 1, character.ArcaneRecoveryRejection{}, 1},
		{"AR-02", slots{1: 2}, slots{1: 2}, 1, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoveryExceedsMaximum, Maximum: 1}, 0},
		{"AR-03", slots{2: 1}, slots{2: 1}, 2, character.ArcaneRecoveryRejection{}, 2},
		{"AR-04", slots{1: 2}, slots{1: 2}, 2, character.ArcaneRecoveryRejection{}, 2},
		{"AR-05", slots{2: 1, 1: 1}, slots{2: 1, 1: 1}, 2, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoveryExceedsMaximum, Maximum: 2}, 0},
		{"AR-06", slots{1: 2}, slots{1: 1}, 2, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoverySlotNotExpended, Level: 1}, 0},
		{"AR-08", slots{5: 1}, slots{5: 1}, 5, character.ArcaneRecoveryRejection{}, 5},
		{"AR-09", slots{6: 1}, slots{6: 1}, 6, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoverySlotLevelTooHigh, Level: 6}, 0},
		{"AR-10", slots{5: 1, 1: 1}, slots{5: 1, 1: 1}, 6, character.ArcaneRecoveryRejection{}, 6},
		{"AR-11", slots{5: 2}, slots{5: 2}, 10, character.ArcaneRecoveryRejection{}, 10},
		{"AR-12", slots{5: 2, 1: 1}, slots{5: 2, 1: 1}, 10, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoveryExceedsMaximum, Maximum: 10}, 0},
		{"AR-13 empty", slots{}, slots{}, 3, character.ArcaneRecoveryRejection{}, 0},
		{"zero counts ignored", slots{3: 0, 1: 1}, slots{1: 1}, 1, character.ArcaneRecoveryRejection{}, 1},
		{"too high before not expended", slots{9: 1, 1: 3}, slots{1: 1}, 20, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoverySlotLevelTooHigh, Level: 9}, 0},
		{"lowest offender", slots{3: 2, 2: 2}, slots{3: 2, 2: 1}, 20, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoverySlotNotExpended, Level: 2}, 0},
		{"level zero", slots{0: 1}, slots{0: 1}, 5, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoverySlotNotExpended, Level: 0}, 0},
		{"negative count", slots{1: -1}, slots{1: 1}, 5, character.ArcaneRecoveryRejection{Kind: character.ArcaneRecoverySlotNotExpended, Level: 1}, 0},
	} {
		choice, rej := character.NewArcaneRecoveryChoice(c.chosen, c.expended, c.max)
		if rej != c.want || choice.Levels() != c.levels || choice.IsEmpty() != (c.levels == 0) {
			t.Errorf("%s = %+v levels %d, %+v; want %+v levels %d", c.label, choice.Slots(), choice.Levels(), rej, c.want, c.levels)
		}
	}
	if zero, rej := character.NewArcaneRecoveryChoice(map[int]int{1: 0}, nil, 3); rej.Kind != 0 || !zero.IsEmpty() || len(zero.Slots()) != 0 {
		t.Errorf("a zero entry is dropped: %+v, %+v", zero.Slots(), rej)
	}
	choice, _ := character.NewArcaneRecoveryChoice(map[int]int{1: 1}, map[int]int{1: 1}, 1)
	choice.Slots()[1] = 9
	if choice.Slots()[1] != 1 {
		t.Error("Slots returned a shared map")
	}
}
