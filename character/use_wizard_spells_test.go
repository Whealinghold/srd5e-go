package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-wizard.md FR-045 to FR-052, FR-110 (table 4.7.1, AR-01 to AR-15) - Arcane Recovery on a finished Short Rest recovers expended slots up to half the Wizard level rounded up and
// none of level 6 or higher, marks the feature used with one event, and is refused by cause in order (NotAfterShortRest for a Long Rest or an interrupted Long Rest, AlreadyUsed, SlotLevelTooHigh,
// SlotNotExpended, ExceedsMaximum); an empty choice is a no-op that leaves it available.
// Path: per case (happy / unhappy: boundary, state error, empty and minimal).
func TestUseArcaneRecovery(t *testing.T) {
	t.Parallel()
	uc := must(character.NewUseArcaneRecovery(wzSource(t)))
	short, long, cut := wzRest(t, character.ShortRest, false), wzRest(t, character.LongRest, false), wzRest(t, character.ShortRest, true)
	used := character.WizardResources{}.WithArcaneRecoveryUsed()
	type slots = map[int]int
	type rej = character.ArcaneRecoveryRejection
	for _, c := range []struct {
		label            string
		level            int
		expended, chosen slots
		resources        character.WizardResources
		trigger          character.RestApplied
		want             rej
		levels           int
	}{
		{"AR-01", 1, slots{1: 1}, slots{1: 1}, character.WizardResources{}, short, rej{}, 1},
		{"AR-02", 2, slots{1: 2}, slots{1: 2}, character.WizardResources{}, short, rej{Kind: character.ArcaneRecoveryExceedsMaximum, Maximum: 1}, 0},
		{"AR-03", 3, slots{2: 1}, slots{2: 1}, character.WizardResources{}, short, rej{}, 2},
		{"AR-04", 4, slots{1: 2}, slots{1: 2}, character.WizardResources{}, short, rej{}, 2},
		{"AR-05", 4, slots{2: 1, 1: 1}, slots{2: 1, 1: 1}, character.WizardResources{}, short, rej{Kind: character.ArcaneRecoveryExceedsMaximum, Maximum: 2}, 0},
		{"AR-06", 4, slots{1: 1}, slots{1: 2}, character.WizardResources{}, short, rej{Kind: character.ArcaneRecoverySlotNotExpended, Level: 1}, 0},
		{"AR-07", 4, slots{2: 1}, slots{2: 1}, used, short, rej{Kind: character.ArcaneRecoveryAlreadyUsed}, 0},
		{"AR-08", 9, slots{5: 1}, slots{5: 1}, character.WizardResources{}, short, rej{}, 5},
		{"AR-09", 11, slots{6: 1}, slots{6: 1}, character.WizardResources{}, short, rej{Kind: character.ArcaneRecoverySlotLevelTooHigh, Level: 6}, 0},
		{"AR-10", 11, slots{5: 1, 1: 1}, slots{5: 1, 1: 1}, character.WizardResources{}, short, rej{}, 6},
		{"AR-11", 20, slots{5: 2}, slots{5: 2}, character.WizardResources{}, short, rej{}, 10},
		{"AR-12", 20, slots{5: 2, 1: 1}, slots{5: 2, 1: 1}, character.WizardResources{}, short, rej{Kind: character.ArcaneRecoveryExceedsMaximum, Maximum: 10}, 0},
		{"AR-13", 5, nil, nil, character.WizardResources{}, short, rej{}, 0},
		{"AR-14", 5, slots{3: 1}, slots{3: 1}, character.WizardResources{}, long, rej{Kind: character.ArcaneRecoveryNotAfterShortRest}, 0},
		{"AR-15", 5, slots{3: 1}, slots{3: 1}, character.WizardResources{}, cut, rej{Kind: character.ArcaneRecoveryNotAfterShortRest}, 0},
		{"used and a Long Rest", 5, slots{3: 1}, slots{3: 1}, used, long, rej{Kind: character.ArcaneRecoveryNotAfterShortRest}, 0},
	} {
		res, err := uc.Run(t.Context(), character.UseArcaneRecoveryCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Resources: c.resources, Expended: c.expended, Chosen: c.chosen, Trigger: c.trigger})
		if err != nil {
			t.Fatalf("%s: %v", c.label, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.UseArcaneRecoveryRejected); !ok || got.Rejection != c.want {
				t.Errorf("%s = %#v, want %+v", c.label, res, c.want)
			}
			continue
		}
		got, ok := res.(character.ArcaneRecoveryApplied)
		if !ok || got.Recovered().Levels() != c.levels {
			t.Fatalf("%s = %#v", c.label, res)
		}
		if c.levels == 0 { // AR-13: a no-op that stays available
			if got.Resources().ArcaneRecoveryUsed() || len(got.Events()) != 0 {
				t.Errorf("%s: an empty choice used the feature: %#v", c.label, got)
			}
			continue
		}
		ev, evOK := got.Events()[0].(character.ArcaneRecoveryUsed)
		if !got.Resources().ArcaneRecoveryUsed() || !evOK || ev.Levels() != c.levels || ev.CharacterID() != wzID {
			t.Errorf("%s: %#v", c.label, got)
		}
		// A Short Rest does not make it available again; a Long Rest does.
		if !got.Resources().AfterRest(character.ShortRest).ArcaneRecoveryUsed() || got.Resources().AfterRest(character.LongRest).ArcaneRecoveryUsed() {
			t.Errorf("%s: rest handling is wrong", c.label)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-065, FR-066, FR-067, FR-069, FR-070 - Spell Mastery from level 18 stores a level 1 and a level 2 Action spell from the book (FeatureNotAvailable below 18, Already once held, then
// the spell causes); after a Long Rest ReplaceSpellMastery swaps one for an eligible spell of the same level, refusing a Short Rest trigger, no Mastery held, a spell that is not a Mastery spell and an ineligible new spell.
// Path: per case (happy / unhappy: state error, invalid input).
func TestSpellMasteryUseCases(t *testing.T) {
	t.Parallel()
	src := wzSource(t)
	choose := must(character.NewChooseSpellMastery(src))
	replace := must(character.NewReplaceSpellMastery(src))
	book := wzBook("one", "two", "one2", "two2", "three", "slow1")
	pair := []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2)}
	held := wzMasteryHeld(t)
	long, short := wzRest(t, character.LongRest, false), wzRest(t, character.ShortRest, false)
	type rej = character.SpellMasteryRejection
	for _, c := range []struct {
		name      string
		level     int
		resources character.WizardResources
		spells    []character.WizardSpellFacts
		want      rej
	}{
		{"level 18", 18, character.WizardResources{}, pair, rej{}},
		{"level 17", 17, character.WizardResources{}, pair, rej{Kind: character.SpellMasteryFeatureNotAvailable}},
		{"held already", 20, held, pair, rej{Kind: character.SpellMasteryAlready}},
		{"not in book", 18, character.WizardResources{}, []character.WizardSpellFacts{wzFact("one", 1), wzFact("zz", 2)}, rej{Kind: character.SpellMasteryNotInSpellbook, Spell: "zz"}},
		{"not an Action", 18, character.WizardResources{}, []character.WizardSpellFacts{wzFact("slow1", 1, wzMinutes), wzFact("two", 2)}, rej{Kind: character.SpellMasteryCastingTimeNotAction, Spell: "slow1"}},
	} {
		res, err := choose.Run(t.Context(), character.ChooseSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Resources: c.resources, Spellbook: book, Spells: c.spells})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.ChooseSpellMasteryRejected); !ok || got.Rejection.Kind != c.want.Kind || got.Rejection.Spell != c.want.Spell {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.SpellMasteryApplied)
		stored, has := got.Resources().SpellMastery()
		ev, evOK := got.Events()[0].(character.SpellMasteryChosen)
		if !ok || !has || stored != got.Mastery() || got.Mastery().LevelOne() != "one" || !evOK || ev.Mastery() != stored || ev.CharacterID() != wzID {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
	type swap struct {
		name      string
		resources character.WizardResources
		trigger   character.RestApplied
		old       content.SpellID
		next      character.WizardSpellFacts
		want      rej
		level     int
	}
	for _, c := range []swap{
		{"level 1 spell", held, long, "one", wzFact("one2", 1), rej{}, 18},
		{"level 2 spell", held, long, "two", wzFact("two2", 2), rej{}, 18},
		{"interrupted long rest (Short Rest benefits)", held, wzRest(t, character.LongRest, true), "one", wzFact("one2", 1), rej{Kind: character.SpellMasteryNotAfterLongRest}, 18},
		{"short rest", held, short, "one", wzFact("one2", 1), rej{Kind: character.SpellMasteryNotAfterLongRest}, 18},
		{"nothing held", character.WizardResources{}, long, "one", wzFact("one2", 1), rej{Kind: character.SpellMasteryNotHeld}, 18},
		{"level 17", held, long, "one", wzFact("one2", 1), rej{Kind: character.SpellMasteryFeatureNotAvailable}, 17},
		{"not a Mastery spell", held, long, "three", wzFact("one2", 1), rej{Kind: character.SpellMasteryNotAMasterySpell, Spell: "three"}, 18},
		{"not in the book", held, long, "one", wzFact("zz", 1), rej{Kind: character.SpellMasteryNotInSpellbook, Spell: "zz"}, 18},
		{"the other Mastery spell", held, long, "one", wzFact("two", 1), rej{Kind: character.SpellMasteryDuplicateSpell, Spell: "two"}, 18},
		{"different level", held, long, "one", wzFact("two2", 2), rej{Kind: character.SpellMasteryWrongSpellLevels}, 18},
		{"lower level than the swapped spell", held, long, "two", wzFact("one2", 1), rej{Kind: character.SpellMasteryWrongSpellLevels}, 18},
		{"not an Action", held, long, "one", wzFact("one2", 1, wzMinutes), rej{Kind: character.SpellMasteryCastingTimeNotAction, Spell: "one2"}, 18},
	} {
		res, err := replace.Run(t.Context(), character.ReplaceSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Resources: c.resources, Spellbook: book, Replace: c.old, New: c.next, Trigger: c.trigger})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.ReplaceSpellMasteryRejected); !ok || got.Rejection.Kind != c.want.Kind || got.Rejection.Spell != c.want.Spell {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.SpellMasteryReplaceApplied)
		ev, evOK := got.Events()[0].(character.SpellMasteryReplaced)
		if !ok || !slices.Contains(got.Mastery().Spells(), c.next.ID) || slices.Contains(got.Mastery().Spells(), c.old) || !evOK || ev.Replaced() != c.old || ev.Mastery() != got.Mastery() || ev.CharacterID() != wzID {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-071, FR-072, FR-075 - Signature Spells at level 20 store two distinct level 3 spells from the book (FeatureNotAvailable below 20, Already once held, then WrongSpellCount,
// DuplicateSpell, NotLevelThree, NotInSpellbook) with both free casts available.
// Path: per case (happy / unhappy: state error, invalid input).
func TestChooseSignatureSpells(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseSignatureSpells(wzSource(t)))
	book := wzBook("s1", "s2", "s3", "two")
	held := wzHolding(t, character.WizardResources{}, wzSignature(t))
	type rej = character.SignatureSpellsRejection
	for _, c := range []struct {
		name      string
		level     int
		resources character.WizardResources
		spells    []character.WizardSpellFacts
		want      rej
	}{
		{"level 20", 20, character.WizardResources{}, wzSigFacts(), rej{}},
		{"level 19", 19, character.WizardResources{}, wzSigFacts(), rej{Kind: character.SignatureSpellsFeatureNotAvailable}},
		{"held already", 20, held, wzSigFacts(), rej{Kind: character.SignatureSpellsAlready}},
		{"one spell", 20, character.WizardResources{}, wzSigFacts()[:1], rej{Kind: character.SignatureSpellsWrongSpellCount, Allowed: 2}},
		{"twice", 20, character.WizardResources{}, []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s1", 3)}, rej{Kind: character.SignatureSpellsDuplicateSpell, Spell: "s1"}},
		{"level 2", 20, character.WizardResources{}, []character.WizardSpellFacts{wzFact("s1", 3), wzFact("two", 2)}, rej{Kind: character.SignatureSpellsNotLevelThree, Spell: "two"}},
		{"not in book", 20, character.WizardResources{}, []character.WizardSpellFacts{wzFact("s1", 3), wzFact("zz", 3)}, rej{Kind: character.SignatureSpellsNotInSpellbook, Spell: "zz"}},
	} {
		res, err := uc.Run(t.Context(), character.ChooseSignatureSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(c.level), Resources: c.resources, Spellbook: book, Spells: c.spells})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if c.want.Kind != 0 {
			if got, ok := res.(character.ChooseSignatureSpellsRejected); !ok || got.Rejection != c.want {
				t.Errorf("%s = %#v, want %+v", c.name, res, c.want)
			}
			continue
		}
		got, ok := res.(character.SignatureSpellsApplied)
		stored, has := got.Resources().SignatureSpells()
		ev, evOK := got.Events()[0].(character.SignatureSpellsChosen)
		if !ok || !has || stored != got.Signature() || got.Resources().SignatureSpellSpent("s1") || !evOK || ev.Signature() != stored || ev.CharacterID() != wzID {
			t.Errorf("%s = %#v", c.name, res)
		}
	}
}
