package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func brbDecide(t *testing.T, facts character.BarbarianChoiceFacts, choice character.BarbarianOptionChoice) character.BarbarianChoiceVerdict {
	t.Helper()
	return must(character.BarbarianChoiceRules{}.Decide(facts, choice))
}

func brbProblems(t *testing.T, v character.BarbarianChoiceVerdict) []character.BarbarianProblem {
	t.Helper()
	r, ok := v.(character.BarbarianChoiceRefused)
	if !ok {
		t.Fatalf("verdict = %#v, want a refusal", v)
	}
	if len(r.Problems()) == 0 {
		t.Fatal("a refusal reports no problem")
	}
	return r.Problems()
}

func brbReasons(ps []character.BarbarianProblem) []character.BarbarianReason {
	out := make([]character.BarbarianReason, len(ps))
	for i, p := range ps {
		out[i] = p.Reason
	}
	return out
}

func brbMasteryFacts(t *testing.T, level int, current []string, weapons ...string) character.BarbarianChoiceFacts {
	t.Helper()
	facts := character.BarbarianChoiceFacts{Character: brbBarbarian(t, level), Stats: brbStats(t, level), Weapons: fgtWeapons(t, weapons...)}
	if len(current) > 0 {
		facts.Masteries = must(createMasteries(t, current...))
	}
	return facts
}

// Requirement: doc/specs/class-barbarian.md FR-041, FR-046, FR-103 - Weapon Mastery is created with exactly the allowed count at level 1 (2) and extended at level 4 (3 kinds keeping the old ones in order) through the shared
// rules; the choice lands under the shared weapon_mastery_choice key, the BarbarianState is untouched and BarbarianWeaponMasteriesChosen carries the kinds.
// Path: happy.
func TestBarbarianChoiceRulesChooseWeaponMasteries(t *testing.T) {
	t.Parallel()
	a, ok := brbDecide(t, brbMasteryFacts(t, 1, nil, "greataxe", "handaxe"), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}).(character.BarbarianChoiceAccepted)
	if !ok {
		t.Fatal("not accepted")
	}
	access := must(character.WeaponMasteryAccessFor(a.Character().FeatureStates())).(character.MasteryFromChoice)
	chosen, isEvent := a.Events()[0].(character.BarbarianWeaponMasteriesChosen)
	if !slices.Equal(access.Choice.Kinds(), ids("greataxe", "handaxe")) || !isEvent || !slices.Equal(chosen.Kinds(), ids("greataxe", "handaxe")) || len(a.Events()) != 1 || chosen.CharacterID() != a.Character().ID() {
		t.Errorf("choice %v, events %#v", access.Choice.Kinds(), a.Events())
	}
	if _, stored, _ := character.ReadFeatureState(a.Character().FeatureStates(), character.BarbarianStateKey()); stored {
		t.Error("a mastery choice must not write the BarbarianState")
	}
	ext := brbDecide(t, brbMasteryFacts(t, 4, []string{"greataxe", "handaxe"}, "greataxe", "handaxe", "mace"), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe", "mace")}).(character.BarbarianChoiceAccepted)
	if got := must(character.WeaponMasteryAccessFor(ext.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(); !slices.Equal(got, ids("greataxe", "handaxe", "mace")) {
		t.Errorf("extended choice = %v", got)
	}
	chosen.Kinds()[0] = "tampered"
	if chosen.Kinds()[0] == "tampered" {
		t.Error("Kinds returned a shared slice")
	}
	a.Events()[0] = nil
	if a.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-043, FR-103 - the Barbarian's eligibility is Simple or Martial Melee weapons: a ranged weapon, an item that is not a weapon, a repeated kind, a wrong count and a dropped kind are
// refused and reported together in the order wrong count, duplicate, not a Simple or Martial Melee weapon, dropped, each with the weapon id where it names one.
// Path: per case (unhappy: boundary, invalid input, rule interaction).
func TestBarbarianChoiceRulesChooseWeaponMasteriesRejections(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		level   int
		current []string
		kinds   []string
		weapons []string
		want    []character.BarbarianReason
	}{
		{"one kind at level 1", 1, nil, []string{"greataxe"}, []string{"greataxe"}, []character.BarbarianReason{character.BarbarianWrongMasteryCount}},
		{"three kinds at level 1", 1, nil, []string{"greataxe", "handaxe", "mace"}, []string{"greataxe", "handaxe", "mace"}, []character.BarbarianReason{character.BarbarianWrongMasteryCount}},
		{"a repeated kind", 1, nil, []string{"greataxe", "greataxe"}, []string{"greataxe"}, []character.BarbarianReason{character.BarbarianDuplicateMastery}},
		{"a ranged weapon", 1, nil, []string{"greataxe", "longbow"}, []string{"greataxe", "longbow"}, []character.BarbarianReason{character.BarbarianNotSimpleOrMartialMeleeWeapon}},
		{"an item that is not a weapon", 1, nil, []string{"greataxe", "leather_armor"}, []string{"greataxe"}, []character.BarbarianReason{character.BarbarianNotSimpleOrMartialMeleeWeapon}},
		{"dropping a kind", 4, []string{"greataxe", "handaxe"}, []string{"greataxe", "mace", "club"}, []string{"greataxe", "mace", "club"}, []character.BarbarianReason{character.BarbarianDropsExistingMastery}},
		{"count, duplicate and drop", 4, []string{"greataxe", "handaxe"}, []string{"greataxe", "greataxe"}, []string{"greataxe"}, []character.BarbarianReason{character.BarbarianWrongMasteryCount, character.BarbarianDuplicateMastery, character.BarbarianDropsExistingMastery}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ps := brbProblems(t, brbDecide(t, brbMasteryFacts(t, c.level, c.current, c.weapons...), character.BarbarianChooseWeaponMasteries{Kinds: ids(c.kinds...)}))
			if got := brbReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
		})
	}
	ps := brbProblems(t, brbDecide(t, brbMasteryFacts(t, 1, nil, "greataxe", "longbow"), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "longbow")}))
	if ps[0].Item != "longbow" {
		t.Errorf("problem = %+v, want the weapon id", ps[0])
	}
}

// Requirement: doc/specs/class-barbarian.md FR-041, FR-043 - after a Long Rest of this character one chosen kind is swapped for another (the choice keeps its order and BarbarianWeaponMasterySwapped is returned); a Short Rest is
// noLongRestFinished, another character's event is eventNotForCharacter, a kind not chosen is notCurrentlyChosen, one already chosen is alreadyChosen and a ranged weapon or an item that is not a weapon is refused.
// Path: per case (unhappy: state error, invalid input).
func TestBarbarianChoiceRulesSwapWeaponMastery(t *testing.T) {
	t.Parallel()
	facts := brbMasteryFacts(t, 1, []string{"greataxe", "handaxe"}, "greataxe", "handaxe", "mace", "club", "longbow")
	id := facts.Character.ID()
	long, short := fgtRest(t, id, character.LongRest), fgtRest(t, id, character.ShortRest)
	foreign := fgtRest(t, character.MustIdentifier("someone-else"), character.LongRest)
	a := brbDecide(t, facts, character.BarbarianSwapWeaponMastery{Remove: "handaxe", Add: "mace", RestApplied: long}).(character.BarbarianChoiceAccepted)
	swapped, ok := a.Events()[0].(character.BarbarianWeaponMasterySwapped)
	if got := must(character.WeaponMasteryAccessFor(a.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(); !slices.Equal(got, ids("greataxe", "mace")) ||
		!ok || swapped.Removed() != "handaxe" || swapped.Added() != "mace" || swapped.CharacterID() != id {
		t.Errorf("choice %v, events %#v", got, a.Events())
	}
	for _, c := range []struct {
		name        string
		remove, add string
		rest        character.RestApplied
		want        []character.BarbarianReason
	}{
		{"a Short Rest", "handaxe", "mace", short, []character.BarbarianReason{character.BarbarianNoLongRestFinished}},
		{"another character's Long Rest", "handaxe", "mace", foreign, []character.BarbarianReason{character.BarbarianEventNotForCharacter}},
		{"a kind not chosen", "club", "mace", long, []character.BarbarianReason{character.BarbarianNotCurrentlyChosen}},
		{"a kind already chosen", "handaxe", "greataxe", long, []character.BarbarianReason{character.BarbarianAlreadyChosen}},
		{"a ranged weapon", "handaxe", "longbow", long, []character.BarbarianReason{character.BarbarianNotSimpleOrMartialMeleeWeapon}},
		{"an item that is not a weapon", "handaxe", "leather_armor", long, []character.BarbarianReason{character.BarbarianNotSimpleOrMartialMeleeWeapon}},
		{"another character and a kind not chosen", "club", "mace", foreign, []character.BarbarianReason{character.BarbarianEventNotForCharacter, character.BarbarianNotCurrentlyChosen}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ps := brbProblems(t, brbDecide(t, facts, character.BarbarianSwapWeaponMastery{Remove: content.ItemID(c.remove), Add: content.ItemID(c.add), RestApplied: c.rest}))
			if got := brbReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
		})
	}
}

// brbForeignChoice implements the unexported marker through embedding, which a type outside the package cannot do directly.
type brbForeignChoice struct {
	character.BarbarianOptionChoice
}

// Requirement: doc/specs/class-barbarian.md FR-103 - Decide refuses a zero character or stats, a nil or foreign choice and a Character holding another type of state under the Weapon Mastery key with an error matching
// srderr.ErrInvalidInput; it never panics.
// Path: per case (unhappy: invalid input).
func TestBarbarianChoiceRulesInvalidInput(t *testing.T) {
	t.Parallel()
	good := brbMasteryFacts(t, 1, nil, "greataxe", "handaxe")
	choice := character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}
	noStats, noChar, foreignMastery := good, good, good
	noStats.Stats, noChar.Character = character.BarbarianLevelStats{}, character.Character{}
	// a swap that the rules would accept, so only the guards can refuse it
	swapFacts := brbMasteryFacts(t, 1, []string{"greataxe", "handaxe"}, "greataxe", "handaxe", "mace")
	swap := character.BarbarianSwapWeaponMastery{Remove: "handaxe", Add: "mace", RestApplied: fgtRest(t, swapFacts.Character.ID(), character.LongRest)}
	swapNoStats, swapNoChar := swapFacts, swapFacts
	swapNoStats.Stats, swapNoChar.Character = character.BarbarianLevelStats{}, character.Character{}
	foreignMastery.Character = must(character.WithFeatureState(good.Character, character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("x"))))
	for _, c := range []struct {
		name   string
		facts  character.BarbarianChoiceFacts
		choice character.BarbarianOptionChoice
	}{
		{"zero stats", noStats, choice}, {"zero character", noChar, choice}, {"nil choice", good, nil}, {"a foreign choice", good, brbForeignChoice{}},
		{"a foreign mastery state", foreignMastery, choice},
		{"zero stats with a swap", swapNoStats, swap}, {"zero character with a swap", swapNoChar, swap},
	} {
		v, err := character.BarbarianChoiceRules{}.Decide(c.facts, c.choice)
		wantInvalid(t, err, c.name)
		if v != nil {
			t.Errorf("%s: verdict = %v, want nil", c.name, v)
		}
	}
}
