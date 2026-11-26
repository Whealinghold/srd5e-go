package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func grpD2Traits(t *testing.T, species string, choices character.SpeciesTraitChoices) character.SpeciesTraits {
	t.Helper()
	return must2(character.NewSpeciesTraits(content.SpeciesID(species), choices))
}

// grpD2SpeciesHero is a level 5 Fighter with the given Constitution score.
func grpD2SpeciesHero(t *testing.T, con int) character.Character {
	t.Helper()
	return grpD2Hero(t, scoresFor(t, 10, 10, con, 10, 10, 10), grpD2CL("fighter", 5))
}

// A Halfling's Luck rerolls a d20 that shows a 1 with one 1d20 through the dice port, and the new face stands even when it is a 1 again. A die that does not show a 1 is not applicable and rolls
// nothing, a die that was already rerolled is refused, a Human and the zero traits are refused (SRD 5.2.1, "Luck", p. 86; "Interactions with Rerolls", p. 8).
// Requirement: doc/specs/species-trait-behaviour.md FR-042 to FR-045, FR-054, FR-060, Table H - RollHalflingLuck through the public API.
// Path: per case (happy extremes, boundary, state error, rule interaction).
func TestSpeciesHalflingLuckFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	halfling := grpD2Traits(t, "halfling", character.NoSpeciesChoices{})
	human := grpD2Traits(t, "human", character.HumanChoices{Skill: content.SkillStealth})
	hero := grpD2SpeciesHero(t, 10)
	for _, c := range []struct {
		name     string
		traits   character.SpeciesTraits
		roll     d20.Roll
		die      int
		rerolled []int
		face     int
		dice     []int
		natural  int
	}{
		{"a single 1", halfling, d20.MustRoll(modifier.Normal, 1), 0, nil, 14, []int{14}, 14},
		{"Advantage, the 1 rerolled low", halfling, d20.MustRoll(modifier.WithAdvantage, 1, 14), 0, nil, 3, []int{3, 14}, 14},
		{"Advantage, the 1 rerolled high", halfling, d20.MustRoll(modifier.WithAdvantage, 1, 14), 0, nil, 19, []int{19, 14}, 19},
		{"two 1s, one rerolled", halfling, d20.MustRoll(modifier.WithAdvantage, 1, 1), 1, nil, 9, []int{1, 9}, 9},
		{"the new roll stands when it is 1 again", halfling, d20.MustRoll(modifier.Normal, 1), 0, nil, 1, []int{1}, 1},
		{"the other die was rerolled for Inspiration", halfling, d20.MustRoll(modifier.WithAdvantage, 1, 1), 0, []int{1}, 7, []int{7, 1}, 7},
	} {
		roller := dicetest.NewFixedRoller(c.face)
		cmd := must2(character.NewRollHalflingLuckCommand(hero, c.traits, c.roll, c.die, c.rerolled))
		got, ok := must2(must2(character.NewRollHalflingLuck(roller)).Run(ctx, cmd)).(character.RollHalflingLuckRerolled)
		if !ok || got.NewValue() != c.face || got.Roll().Natural() != c.natural || !slices.Equal(got.Roll().Dice(), c.dice) || !slices.Contains(got.RerolledIndexes(), c.die) ||
			got.Event().DieIndex() != c.die || got.Event().NewValue() != c.face || got.Event().CharacterID() != hero.ID() {
			t.Errorf("%s = %#v", c.name, got)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D20(), 0) {
			t.Errorf("%s requests = %v, want one 1d20", c.name, reqs)
		}
	}
	for _, c := range []struct {
		name     string
		traits   character.SpeciesTraits
		roll     d20.Roll
		die      int
		rerolled []int
		want     string // "" is not applicable
	}{
		{"a single 2", halfling, d20.MustRoll(modifier.Normal, 2), 0, nil, ""},
		{"the other die is not a 1", halfling, d20.MustRoll(modifier.WithAdvantage, 1, 14), 1, nil, ""},
		{"a die already rerolled", halfling, d20.MustRoll(modifier.Normal, 1), 0, []int{0}, "die_already_rerolled"},
		{"not a Halfling", human, d20.MustRoll(modifier.Normal, 1), 0, nil, "species_use_not_offered"},
		{"the zero traits", character.SpeciesTraits{}, d20.MustRoll(modifier.Normal, 1), 0, nil, "species_behaviour_unknown"},
	} {
		roller := dicetest.NewFixedRoller(14)
		res := must2(must2(character.NewRollHalflingLuck(roller)).Run(ctx, must2(character.NewRollHalflingLuckCommand(hero, c.traits, c.roll, c.die, c.rerolled))))
		switch r := res.(type) {
		case character.RollHalflingLuckNotApplicable:
			if c.want != "" || r.Face() != c.roll.Dice()[c.die] {
				t.Errorf("%s = %#v", c.name, r)
			}
		case character.RollHalflingLuckRejected:
			if r.Rejection.String() != c.want {
				t.Errorf("%s rejection = %v, want %q", c.name, r.Rejection, c.want)
			}
		default:
			t.Errorf("%s = %#v, want not applicable or rejected", c.name, res)
		}
		if len(roller.Requests()) != 0 {
			t.Errorf("%s rolled %v, want no roll", c.name, roller.Requests())
		}
	}
	// Failures and invalid input.
	cmd := must2(character.NewRollHalflingLuckCommand(hero, halfling, d20.MustRoll(modifier.Normal, 1), 0, nil))
	res, err := must2(character.NewRollHalflingLuck(dicetest.NewFixedRoller())).Run(ctx, cmd)
	if res != nil {
		t.Errorf("an empty roller returned %#v", res)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)
	_, err = character.NewRollHalflingLuckCommand(character.Character{}, halfling, d20.MustRoll(modifier.Normal, 1), 0, nil)
	grpD2WantInvalid(t, err, "a zero character")
	_, err = character.NewRollHalflingLuckCommand(hero, halfling, d20.Roll{}, 0, nil)
	grpD2WantInvalid(t, err, "a zero roll")
	_, err = character.NewRollHalflingLuckCommand(hero, halfling, d20.MustRoll(modifier.Normal, 1), 1, nil)
	grpD2WantInvalid(t, err, "a die index outside the roll")
	_, err = character.NewRollHalflingLuck(nil)
	grpD2WantInvalid(t, err, "a nil roller")
	_, err = character.RollHalflingLuck{}.Run(ctx, cmd)
	grpD2WantInvalid(t, err, "the zero use case")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := must2(character.NewRollHalflingLuck(dicetest.NewFixedRoller(5))).Run(cancelled, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v", err)
	}
}

// A Stone Goliath reduces the damage it takes by 1d12 plus its Constitution modifier (never below 0 of reduction), spending one of its uses; the pool of two is then spent. A Hill Goliath,
// a Dwarf and the zero traits are refused and nothing is rolled (SRD 5.2.1, "Stone's Endurance", p. 84).
// Requirement: doc/specs/species-trait-behaviour.md FR-038, FR-062, Table G - RollStonesEndurance through the public API.
// Path: per case (happy extremes, boundary, state error, collaborator failure).
func TestSpeciesStonesEnduranceFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stone := grpD2Traits(t, "goliath", character.GoliathChoices{Ancestry: character.AncestryStone})
	for _, c := range []struct {
		name                       string
		con, roll, damage          int
		modifier, reduction, after int
	}{
		{"roll 6, Con +2, damage 20", 14, 6, 20, 2, 8, 12}, {"roll 12, Con +3, damage 10", 16, 12, 10, 3, 15, 0},
		{"roll 1, Con -3, damage 5", 4, 1, 5, -3, 0, 5}, {"roll 1, Con -5, damage 9", 1, 1, 9, -5, 0, 9}, {"no damage", 14, 5, 0, 2, 7, 0},
	} {
		hero := grpD2Hero(t, scoresFor(t, 10, 10, c.con, 10, 10, 10), grpD2CL("fighter", 1))
		roller := dicetest.NewFixedRoller(c.roll)
		got, ok := must2(must2(character.NewRollStonesEndurance(roller)).Run(ctx, must2(character.NewRollStonesEnduranceCommand(hero, stone, c.damage)))).(character.RollStonesEnduranceRolled)
		if !ok || got.Roll() != c.roll || got.Modifier() != c.modifier || got.Reduction() != c.reduction || got.DamageAfter() != c.after ||
			got.Traits().ExpendedOf(character.GiantAncestryUse) != 1 || got.Event().Roll() != c.roll || got.Event().Reduction() != c.reduction || got.Event().CharacterID() != hero.ID() {
			t.Errorf("%s = %#v", c.name, got)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D12(), 0) {
			t.Errorf("%s requests = %v, want one 1d12", c.name, reqs)
		}
	}
	hero := grpD2Hero(t, scoresFor(t, 10, 10, 14, 10, 10, 10), grpD2CL("fighter", 1))
	spent := stone
	for i := 0; i < 2; i++ {
		res := must2(must2(character.NewRollStonesEndurance(dicetest.NewFixedRoller(6))).Run(ctx, must2(character.NewRollStonesEnduranceCommand(hero, spent, 20)))).(character.RollStonesEnduranceRolled)
		spent = res.Traits()
	}
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		want   string
	}{
		{"the pool spent", spent, "species_use_exhausted"},
		{"a Hill Goliath", grpD2Traits(t, "goliath", character.GoliathChoices{Ancestry: character.AncestryHill}), "species_use_not_offered"},
		{"a Dwarf", grpD2Traits(t, "dwarf", character.NoSpeciesChoices{}), "species_use_not_offered"},
		{"the zero traits", character.SpeciesTraits{}, "species_behaviour_unknown"},
	} {
		roller := dicetest.NewFixedRoller(6)
		res := must2(must2(character.NewRollStonesEndurance(roller)).Run(ctx, must2(character.NewRollStonesEnduranceCommand(hero, c.traits, 20))))
		if r, ok := res.(character.RollStonesEnduranceRejected); !ok || r.Rejection.String() != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s = %#v (requests %v), want rejected %s with no roll", c.name, res, roller.Requests(), c.want)
		}
	}
	cmd := must2(character.NewRollStonesEnduranceCommand(hero, stone, 10))
	res, err := must2(character.NewRollStonesEndurance(dicetest.NewFixedRoller())).Run(ctx, cmd)
	if res != nil {
		t.Errorf("an empty roller returned %#v", res)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)
	_, err = character.NewRollStonesEnduranceCommand(character.Character{}, stone, 10)
	grpD2WantInvalid(t, err, "a zero character")
	_, err = character.NewRollStonesEnduranceCommand(hero, stone, -1)
	grpD2WantInvalid(t, err, "a negative damage")
	_, err = character.NewRollStonesEndurance(nil)
	grpD2WantInvalid(t, err, "a nil roller")
	_, err = character.RollStonesEndurance{}.Run(ctx, cmd)
	grpD2WantInvalid(t, err, "the zero use case")
}

// An Orc reduced to 0 Hit Points but not killed outright drops to 1 Hit Point and spends its use; a second drop is refused until a Long Rest renews it. A drop that kills outright is not
// triggered, even with the use spent. A Dwarf and the zero traits are refused (SRD 5.2.1, "Relentless Endurance", p. 86; "Instant Death", p. 17).
// Requirement: doc/specs/species-trait-behaviour.md FR-041, FR-052, FR-060, Table E - UseRelentlessEndurance through the public API.
// Path: per case (happy, boundary, state error, invalid input).
func TestSpeciesRelentlessEnduranceFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orc := grpD2Traits(t, "orc", character.NoSpeciesChoices{})
	hero := grpD2SpeciesHero(t, 14)
	uc := character.NewUseRelentlessEndurance()
	notKilled, killed := character.ZeroHitPointsNotKilled{}, character.ZeroHitPointsKilledOutright{}
	used, ok := must2(uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: orc, Outcome: notKilled})).(character.UseRelentlessEnduranceUsed)
	if !ok || used.HitPointsInstead() != 1 || used.Traits().ExpendedOf(character.RelentlessEnduranceUse) != 1 || used.Event().CharacterID() != hero.ID() {
		t.Fatalf("the first drop = %#v", used)
	}
	if r, isRej := must2(uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: used.Traits(), Outcome: notKilled})).(character.UseRelentlessEnduranceRejected); !isRej || r.Rejection.String() != "species_use_exhausted" {
		t.Errorf("the second drop = %#v", r)
	}
	if _, isNot := must2(uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: used.Traits(), Outcome: killed})).(character.UseRelentlessEnduranceNotTriggered); !isNot {
		t.Error("killed outright must not trigger, even with the use spent")
	}
	if _, isNot := must2(uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: orc, Outcome: killed})).(character.UseRelentlessEnduranceNotTriggered); !isNot {
		t.Error("killed outright must not trigger")
	}
	renewed := must2(character.SpeciesUses{}.Renew(hero, used.Traits(), grpD2Rest(t, hero.ID(), character.LongRest))).(character.SpeciesTraitsChanged).Traits()
	if _, again := must2(uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: renewed, Outcome: notKilled})).(character.UseRelentlessEnduranceUsed); !again {
		t.Error("a Long Rest must renew Relentless Endurance")
	}
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		want   string
	}{{"a Dwarf", grpD2Traits(t, "dwarf", character.NoSpeciesChoices{}), "species_use_not_offered"}, {"the zero traits", character.SpeciesTraits{}, "species_behaviour_unknown"}} {
		if r, isRej := must2(uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: c.traits, Outcome: notKilled})).(character.UseRelentlessEnduranceRejected); !isRej || r.Rejection.String() != c.want {
			t.Errorf("%s = %#v, want %s", c.name, r, c.want)
		}
	}
	_, err := uc.Run(ctx, character.UseRelentlessEnduranceCommand{Traits: orc, Outcome: notKilled})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = uc.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: orc})
	grpD2WantInvalid(t, err, "a nil outcome")
	_, err = character.UseRelentlessEndurance{}.Run(ctx, character.UseRelentlessEnduranceCommand{Character: hero, Traits: orc, Outcome: notKilled})
	grpD2WantInvalid(t, err, "the zero use case")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, character.UseRelentlessEnduranceCommand{Character: hero, Traits: orc, Outcome: notKilled}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v", err)
	}
}

// A High Elf whose cantrip window is open (after a Long Rest) swaps its Magic Initiate-style Wizard cantrip for another Wizard cantrip: the new spell is looked up, the window closes, and a
// spell that is not a cantrip, a closed window, a Drow, the zero traits and a spell the source lacks are refused or reported (SRD 5.2.1, "High Elf Lineage", p. 83).
// Requirement: doc/specs/species-trait-behaviour.md FR-025, FR-060, FR-061, FR-065, Table L - ReplaceHighElfCantrip through the public API.
// Path: happy and unhappy (state error, missing content, collaborator failure, invalid input).
func TestSpeciesReplaceHighElfCantripFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := contenttest.NewFakeSource(grpD2Spell(t, "light", 0, "wizard"), grpD2Spell(t, "shield", 1, "wizard"))
	elf := func(l character.ElvenLineage) character.SpeciesTraits {
		return grpD2Traits(t, "elf", character.ElfChoices{Lineage: l, Ability: ability.Wisdom, KeenSensesSkill: content.SkillInsight})
	}
	open := must2(elf(character.ElvenHighElf).WithCantripWindow(true))
	hero := grpD2SpeciesHero(t, 10)
	uc := must2(character.NewReplaceHighElfCantrip(src))
	got, ok := must2(uc.Run(ctx, character.ReplaceHighElfCantripCommand{Character: hero, Traits: open, Spell: "light"})).(character.ReplaceHighElfCantripSucceeded)
	if !ok || got.Event().New() != "light" || got.Traits().CantripWindowOpen() {
		t.Fatalf("swapping to Light = %#v", got)
	}
	if cantrip, held := got.Traits().HighElfCantrip(); !held || cantrip != "light" {
		t.Errorf("the held cantrip = %v, %v", cantrip, held)
	}
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		spell  content.SpellID
		want   string
	}{
		{"a spell that is not a cantrip", open, "shield", "species_choices_invalid"},
		{"a closed window", elf(character.ElvenHighElf), "light", "cantrip_window_closed"},
		{"a Drow", elf(character.ElvenDrow), "light", "species_use_not_offered"},
		{"the zero traits", character.SpeciesTraits{}, "light", "species_behaviour_unknown"},
	} {
		if r, isRej := must2(uc.Run(ctx, character.ReplaceHighElfCantripCommand{Character: hero, Traits: c.traits, Spell: c.spell})).(character.ReplaceHighElfCantripRejected); !isRej || r.Rejection.String() != c.want {
			t.Errorf("%s = %#v, want %s", c.name, r, c.want)
		}
	}
	missing, ok := must2(uc.Run(ctx, character.ReplaceHighElfCantripCommand{Character: hero, Traits: open, Spell: "unknown_spell"})).(character.ReplaceHighElfCantripContentMissing)
	if !ok || !slices.Equal(missing.Missing(), []content.MissingContent{content.SpellID("unknown_spell").Missing()}) {
		t.Errorf("a spell the source lacks = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewReplaceHighElfCantrip(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.ReplaceHighElfCantripCommand{Character: hero, Traits: open, Spell: "light"})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewReplaceHighElfCantrip(nil)
	grpD2WantInvalid(t, err, "a nil source")
	_, err = uc.Run(ctx, character.ReplaceHighElfCantripCommand{Traits: open, Spell: "light"})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = uc.Run(ctx, character.ReplaceHighElfCantripCommand{Character: hero, Traits: open})
	grpD2WantInvalid(t, err, "an invalid spell id")
	_, err = character.ReplaceHighElfCantrip{}.Run(ctx, character.ReplaceHighElfCantripCommand{Character: hero, Traits: open, Spell: "light"})
	grpD2WantInvalid(t, err, "the zero use case")
}
