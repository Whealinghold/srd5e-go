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
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// grpD2FeatSource holds the synthetic feats and the Magic Initiate spells the feat scenarios take.
func grpD2FeatSource(t *testing.T) *contenttest.FakeSource {
	t.Helper()
	level19 := must2(content.NewLevelPrerequisite(19))
	spellcasting := must2(content.NewFeaturePrerequisite("Spellcasting Feature"))
	return contenttest.NewFakeSource(
		grpD2Feat("alert", content.FeatOrigin, false),
		grpD2Feat("magic_initiate", content.FeatOrigin, true),
		grpD2Feat("boon_of_fate", content.FeatEpicBoon, false, level19),
		grpD2Feat("boon_of_spell_recall", content.FeatEpicBoon, false, level19, spellcasting),
		grpD2Spell(t, "guidance", 0, "cleric", "druid"), grpD2Spell(t, "light", 0, "cleric", "wizard"), grpD2Spell(t, "mending", 0, "cleric", "wizard", "druid"),
		grpD2Spell(t, "fire_bolt", 0, "wizard"), grpD2Spell(t, "bless", 1, "cleric"))
}

func grpD2ClericInitiate() character.MagicInitiateChoices {
	return must2(character.NewMagicInitiateChoices("cleric", ability.Wisdom, []content.SpellID{"guidance", "light"}, "bless"))
}

var (
	grpD2Origin = func() character.FeatGrant {
		return must2(character.NewFeatGrant(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin}))
	}
	grpD2AnyFeat = func() character.FeatGrant {
		return must2(character.NewFeatGrant(character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_asi"}, character.AnyFeat{}))
	}
)

// grpD2LevelGained is the LevelGained event of the hero's last level, built by levelling a Fighter one level lower with the same id.
func grpD2LevelGained(t *testing.T, hero character.Character) character.LevelGained {
	t.Helper()
	n := hero.TotalLevel().Value()
	lv := character.MustClassLevels([]character.ClassLevel{{Class: "fighter", Level: n - 1}})
	before := must2(character.Restore(character.Parts{
		ID: hero.ID(), ExperiencePoints: character.ExperiencePoints{}.MinimumForLevel(character.MustLevel(n)), ClassLevels: lv, Species: "human", Background: "soldier",
		AbilityScores: scoresFor(t, 10, 10, 10, 10, 10, 10), HitPointMaximum: character.MustHitPointMaximum(50),
		HitDicePool: character.MustHitDicePoolFromLevels(lv, map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}), HeroicInspiration: character.NotInspired,
	}))
	adv := must2(character.NewAdvancementClass("fighter", content.HitDieD10, must2(content.NewAnyOf(ability.Strength, ability.Dexterity))))
	tr := must2(before.LevelUp(adv, must2(character.NewFixedHitPointGain(6)), nil)).(character.TransitionChanged[character.LevelUpRejection])
	for _, e := range tr.Events() {
		if lg, ok := e.(character.LevelGained); ok {
			return lg
		}
	}
	t.Fatal("no LevelGained event")
	return character.LevelGained{}
}

// A level 20 Fighter takes feats through TakeFeat, each looked up in a source: an Origin feat from a custom background, Magic Initiate (with its three spells looked up) from a species trait,
// and two Epic Boons as the level 19 grants. The returned feats carry on to the next command, the input character is never altered, and each taking emits a FeatTaken event. Then the use
// of Boon of Fate is expended, refused a second time, and renewed by a finished Short Rest (SRD 5.2.1, "Feats", pp. 87-88; "Boon of Fate", p. 90).
// Requirement: doc/specs/feats-behaviour.md FR-050 to FR-052, FR-066, FR-071, FR-073, FR-075, Table U - TakeFeat, ExpendFeatUse and RenewFeatUses through the public API.
// Path: happy.
func TestFeatsTakeExpendAndRenewFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := grpD2FeatSource(t)
	hero := grpD2Hero(t, scoresFor(t, 14, 14, 10, 10, 10, 10), grpD2CL("fighter", 20))
	take := must2(character.NewTakeFeat(src))
	cmd := func(feats character.TakenFeats, feat string, grant character.FeatGrant, choices character.FeatChoices, held ...string) character.TakeFeatCommand {
		ids := make([]content.FeatureID, len(held))
		for i, h := range held {
			ids[i] = content.FeatureID(h)
		}
		return character.TakeFeatCommand{Character: hero, Feats: feats, Feat: content.FeatID(feat), Grant: grant, Choices: choices, Held: must2(character.NewHeldFeatures(ids...))}
	}
	var feats character.TakenFeats
	step := func(feat string, grant character.FeatGrant, choices character.FeatChoices, held ...string) character.TakeFeatSucceeded {
		t.Helper()
		got, ok := must2(take.Run(ctx, cmd(feats, feat, grant, choices, held...))).(character.TakeFeatSucceeded)
		if !ok || !got.Feats().Holds(content.FeatID(feat)) {
			t.Fatalf("taking %s was not applied: %#v", feat, got)
		}
		if ev, isEv := got.Events()[0].(character.FeatTaken); !isEv || ev.Feat() != content.FeatID(feat) {
			t.Errorf("%s events = %#v", feat, got.Events())
		}
		feats = got.Feats()
		return got
	}
	step("alert", grpD2Origin(), character.NoFeatChoices{})
	initiate := step("magic_initiate", must2(character.NewFeatGrant(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin})), grpD2ClericInitiate())
	if initiate.GrantedSpells().IsZero() {
		t.Error("Magic Initiate must grant its spells")
	}
	step("boon_of_fate", grpD2AnyFeat(), must2(character.NewSingleAbilityIncrease(ability.Wisdom)))
	step("boon_of_spell_recall", grpD2AnyFeat(), must2(character.NewSingleAbilityIncrease(ability.Wisdom)), "wizard_spellcasting")
	if feats.Len() != 4 || hero.TotalLevel().Value() != 20 {
		t.Errorf("feats held = %d, want 4", feats.Len())
	}

	expend := character.NewExpendFeatUse()
	spent, ok := must2(expend.Run(ctx, character.ExpendFeatUseCommand{Character: hero, Feats: feats, Feat: "boon_of_fate"})).(character.ExpendFeatUseSucceeded)
	if !ok || spent.Event().Feat() != "boon_of_fate" || spent.Feats().UseStateOf("boon_of_fate") != (character.FeatUseIsExpended{}) || feats.UseStateOf("boon_of_fate") != (character.FeatUseIsAvailable{}) {
		t.Fatalf("expending Boon of Fate = %#v", spent)
	}
	for _, c := range []struct {
		name  string
		feats character.TakenFeats
		feat  string
		want  string
	}{
		{"an expended use", spent.Feats(), "boon_of_fate", "feat_use_already_expended"},
		{"a feat not held", feats, "grappler", "feat_not_held"},
		{"a feat with no limited use", feats, "alert", "feat_has_no_limited_use"},
	} {
		if r, isRej := must2(expend.Run(ctx, character.ExpendFeatUseCommand{Character: hero, Feats: c.feats, Feat: content.FeatID(c.feat)})).(character.ExpendFeatUseRejected); !isRej || r.Rejection.String() != c.want {
			t.Errorf("%s = %#v, want %s", c.name, r, c.want)
		}
	}
	renew := character.NewRenewFeatUses()
	renewed, ok := must2(renew.Run(ctx, character.RenewFeatUsesCommand{Character: hero, Feats: spent.Feats(), Renewal: character.RenewOnShortRestFinished})).(character.RenewFeatUsesSucceeded)
	if !ok || len(renewed.Events()) != 1 || renewed.Feats().UseStateOf("boon_of_fate") != (character.FeatUseIsAvailable{}) {
		t.Errorf("a finished Short Rest = %#v, want the use renewed with one event", renewed)
	}
	nothing := must2(renew.Run(ctx, character.RenewFeatUsesCommand{Character: hero, Feats: feats, Renewal: character.RenewOnLongRestFinished})).(character.RenewFeatUsesSucceeded)
	if len(nothing.Events()) != 0 || !nothing.Feats().Equal(feats) {
		t.Errorf("renewing nothing = %#v, want no events and the same feats", nothing)
	}
}

// The unhappy paths of TakeFeat: a feat already held that is not repeatable, a feat whose prerequisite the character does not meet, a feat the source lacks, a choice that names a spell off
// the list, a failing source, and zero or nil input.
// Requirement: doc/specs/feats-behaviour.md FR-066, FR-071, FR-073, FR-075 - the rejection, content-missing and failure results of TakeFeat.
// Path: unhappy (state error, boundary, missing content, collaborator failure, invalid input, cancellation).
func TestFeatsTakeFeatRefusals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := grpD2FeatSource(t)
	take := must2(character.NewTakeFeat(src))
	hero := grpD2Hero(t, scoresFor(t, 14, 14, 10, 10, 10, 10), grpD2CL("fighter", 20))
	first := must2(take.Run(ctx, character.TakeFeatCommand{Character: hero, Feat: "alert", Grant: grpD2Origin(), Choices: character.NoFeatChoices{}})).(character.TakeFeatSucceeded)

	again, ok := must2(take.Run(ctx, character.TakeFeatCommand{Character: hero, Feats: first.Feats(), Feat: "alert", Grant: grpD2Origin(), Choices: character.NoFeatChoices{}})).(character.TakeFeatRejected)
	if !ok || again.Rejection.String() != "feat_not_repeatable" {
		t.Errorf("a second Alert = %#v, want not repeatable", again)
	}
	low := grpD2Hero(t, scoresFor(t, 14, 14, 10, 10, 10, 10), grpD2CL("fighter", 18))
	early, ok := must2(take.Run(ctx, character.TakeFeatCommand{Character: low, Feat: "boon_of_fate", Grant: grpD2AnyFeat(), Choices: must2(character.NewSingleAbilityIncrease(ability.Wisdom))})).(character.TakeFeatRejected)
	if !ok || early.Rejection.String() != "feat_prerequisite_not_met" {
		t.Errorf("an Epic Boon at level 18 = %#v, want prerequisite not met", early)
	}
	badSpell := must2(character.NewMagicInitiateChoices("cleric", ability.Wisdom, []content.SpellID{"guidance", "fire_bolt"}, "bless"))
	off, ok := must2(take.Run(ctx, character.TakeFeatCommand{Character: hero, Feat: "magic_initiate", Grant: grpD2Origin(), Choices: badSpell})).(character.TakeFeatRejected)
	if !ok || off.Rejection.String() != "feat_choices_invalid" {
		t.Errorf("a cantrip off the Cleric list = %#v, want choices invalid", off)
	}
	missing, ok := must2(take.Run(ctx, character.TakeFeatCommand{Character: hero, Feat: "grappler", Grant: grpD2AnyFeat(), Choices: character.NoFeatChoices{}})).(character.TakeFeatContentMissing)
	if !ok || !slices.Equal(missing.Missing(), []content.MissingContent{content.FeatID("grappler").Missing()}) {
		t.Errorf("a feat the source lacks = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewTakeFeat(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.TakeFeatCommand{Character: hero, Feat: "alert", Grant: grpD2Origin(), Choices: character.NoFeatChoices{}})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewTakeFeat(nil)
	grpD2WantInvalid(t, err, "a nil source")
	for name, c := range map[string]character.TakeFeatCommand{
		"a zero character": {Feat: "alert", Grant: grpD2Origin(), Choices: character.NoFeatChoices{}}, "an invalid feat id": {Character: hero, Grant: grpD2Origin(), Choices: character.NoFeatChoices{}},
		"a zero grant": {Character: hero, Feat: "alert", Choices: character.NoFeatChoices{}}, "nil choices": {Character: hero, Feat: "alert", Grant: grpD2Origin()},
	} {
		_, err := take.Run(ctx, c)
		grpD2WantInvalid(t, err, name)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := take.Run(cancelled, character.TakeFeatCommand{Character: hero, Feat: "alert", Grant: grpD2Origin(), Choices: character.NoFeatChoices{}}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v", err)
	}
	// ExpendFeatUse and RenewFeatUses refuse zero input before any work.
	_, err = character.NewExpendFeatUse().Run(ctx, character.ExpendFeatUseCommand{Feat: "alert"})
	grpD2WantInvalid(t, err, "expend with a zero character")
	_, err = character.NewExpendFeatUse().Run(ctx, character.ExpendFeatUseCommand{Character: hero})
	grpD2WantInvalid(t, err, "expend with an invalid feat id")
	_, err = character.NewRenewFeatUses().Run(ctx, character.RenewFeatUsesCommand{Renewal: character.RenewOnLongRestFinished})
	grpD2WantInvalid(t, err, "renew with a zero character")
	_, err = character.NewRenewFeatUses().Run(ctx, character.RenewFeatUsesCommand{Character: hero})
	grpD2WantInvalid(t, err, "renew with no renewal")
}

// A Fighter with Magic Initiate (Cleric) swaps one of its cantrips for another when it gains a level: the new spell is looked up, the swap is recorded on the feat and may happen once per
// level. A cantrip off the list is refused, a second swap at the same level is refused and a spell the source lacks is content-missing (SRD 5.2.1, "Magic Initiate", p. 88).
// Requirement: doc/specs/feats-behaviour.md FR-070, FR-073, FR-074, Table U "spell change use case" - ChangeMagicInitiateSpell through the public API.
// Path: happy and unhappy (boundary, state error, missing content, collaborator failure, invalid input).
func TestFeatsChangeMagicInitiateSpellFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := grpD2FeatSource(t)
	hero := grpD2Flat(t, 10, grpD2CL("fighter", 5))
	taken := must2(must2(character.NewTakeFeat(src)).Run(ctx, character.TakeFeatCommand{Character: hero, Feat: "magic_initiate", Grant: grpD2Origin(), Choices: grpD2ClericInitiate()})).(character.TakeFeatSucceeded)
	gained := grpD2LevelGained(t, hero)
	uc := must2(character.NewChangeMagicInitiateSpell(src))
	cmd := func(feats character.TakenFeats, newSpell string) character.ChangeMagicInitiateSpellCommand {
		return character.ChangeMagicInitiateSpellCommand{Character: hero, Feats: feats, List: "cleric", Old: "guidance", New: content.SpellID(newSpell), LevelGained: gained}
	}
	before := taken.Feats()
	changed, ok := must2(uc.Run(ctx, cmd(before, "mending"))).(character.ChangeMagicInitiateSpellSucceeded)
	if !ok || changed.Event().Old() != "guidance" || changed.Event().New() != "mending" || !changed.Feats().All()[0].Choices().(character.MagicInitiateChoices).SpellChangedAt(hero.TotalLevel()) {
		t.Fatalf("swapping guidance for mending = %#v", changed)
	}
	if !before.Equal(taken.Feats()) {
		t.Error("the command's feats were altered")
	}
	if r, isRej := must2(uc.Run(ctx, cmd(before, "fire_bolt"))).(character.ChangeMagicInitiateSpellRejected); !isRej || r.Rejection.String() != "feat_choices_invalid" {
		t.Errorf("a cantrip off the Cleric list = %#v", r)
	}
	second := cmd(changed.Feats(), "light")
	second.Old = "mending"
	if r, isRej := must2(uc.Run(ctx, second)).(character.ChangeMagicInitiateSpellRejected); !isRej || r.Rejection.String() != "spell_change_already_used" {
		t.Errorf("a second swap at the same level = %#v", r)
	}
	lookups := len(src.Queries())
	if m, isMissing := must2(must2(character.NewChangeMagicInitiateSpell(contenttest.NewFakeSource())).Run(ctx, cmd(before, "mending"))).(character.ChangeMagicInitiateSpellContentMissing); !isMissing ||
		!slices.Equal(m.Missing(), []content.MissingContent{content.SpellID("mending").Missing()}) {
		t.Errorf("a spell the source lacks = %#v", m)
	}
	if lookups != len(src.Queries()) {
		t.Error("the other source must not be consulted")
	}
	boom := errors.New("source down")
	res, err := must2(character.NewChangeMagicInitiateSpell(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, cmd(before, "mending"))
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewChangeMagicInitiateSpell(nil)
	grpD2WantInvalid(t, err, "a nil source")
	zero := cmd(before, "mending")
	zero.Character = character.Character{}
	_, err = uc.Run(ctx, zero)
	grpD2WantInvalid(t, err, "a zero character")
	invalid := cmd(before, "mending")
	invalid.Old = ""
	_, err = uc.Run(ctx, invalid)
	grpD2WantInvalid(t, err, "an invalid spell id")
}

// A level 20 Fighter with Boon of Spell Recall rolls 1d4 when it casts with a slot of level 1 to 4: a roll of the slot's level or less spares the slot, a higher roll spends it. A slot of
// level 5 or higher is not applicable and rolls nothing, a character without the Boon is refused, and a failing roller is a dice port failure (SRD 5.2.1, "Boon of Spell Recall", p. 91).
// Requirement: doc/specs/feats-behaviour.md FR-053 to FR-055, FR-071, FR-073, Table F - RollFreeCasting through the public API.
// Path: happy (extremes) and unhappy (boundary, state error, collaborator failure, invalid input).
func TestFeatsRollFreeCastingFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := grpD2FeatSource(t)
	hero := grpD2Hero(t, scoresFor(t, 14, 14, 10, 10, 10, 10), grpD2CL("fighter", 20))
	recall := must2(must2(character.NewTakeFeat(src)).Run(ctx, character.TakeFeatCommand{Character: hero, Feat: "boon_of_spell_recall", Grant: grpD2AnyFeat(),
		Choices: must2(character.NewSingleAbilityIncrease(ability.Wisdom)), Held: must2(character.NewHeldFeatures("wizard_spellcasting"))})).(character.TakeFeatSucceeded).Feats()
	for _, c := range []struct {
		slot, roll int
		want       character.FreeCastingOutcome
	}{{1, 1, character.FreeCastingSpared}, {2, 2, character.FreeCastingSpared}, {3, 4, character.FreeCastingSpent}, {4, 3, character.FreeCastingSpent}} {
		roller := dicetest.NewFixedRoller(c.roll)
		res, ok := must2(must2(character.NewRollFreeCasting(roller)).Run(ctx, character.RollFreeCastingCommand{Character: hero, Feats: recall, SlotLevel: character.MustSlotLevel(c.slot)})).(character.RollFreeCastingResolved)
		if !ok || res.Roll() != c.roll || res.Outcome() != c.want {
			t.Errorf("slot %d roll %d = %#v, want %v", c.slot, c.roll, res, c.want)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D4(), 0) {
			t.Errorf("slot %d requests = %v, want one 1d4", c.slot, reqs)
		}
	}
	for _, slot := range []int{5, 9} {
		roller := dicetest.NewFixedRoller(1)
		res, ok := must2(must2(character.NewRollFreeCasting(roller)).Run(ctx, character.RollFreeCastingCommand{Character: hero, Feats: recall, SlotLevel: character.MustSlotLevel(slot)})).(character.RollFreeCastingNotApplicable)
		if !ok || res.SlotLevel() != character.MustSlotLevel(slot) || len(roller.Requests()) != 0 {
			t.Errorf("slot %d = %#v (requests %v), want not applicable and no roll", slot, res, roller.Requests())
		}
	}
	none, ok := must2(must2(character.NewRollFreeCasting(dicetest.NewFixedRoller(2))).Run(ctx, character.RollFreeCastingCommand{Character: hero, Feats: character.TakenFeats{}, SlotLevel: character.MustSlotLevel(2)})).(character.RollFreeCastingRejected)
	if !ok || none.Rejection.String() != "feat_not_held" {
		t.Errorf("without the Boon = %#v", none)
	}
	res, err := must2(character.NewRollFreeCasting(dicetest.NewFixedRoller())).Run(ctx, character.RollFreeCastingCommand{Character: hero, Feats: recall, SlotLevel: character.MustSlotLevel(2)})
	if res != nil {
		t.Errorf("an empty roller returned %#v", res)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)
	_, err = character.NewRollFreeCasting(nil)
	grpD2WantInvalid(t, err, "a nil roller")
	uc := must2(character.NewRollFreeCasting(dicetest.NewFixedRoller(1)))
	_, err = uc.Run(ctx, character.RollFreeCastingCommand{Feats: recall, SlotLevel: character.MustSlotLevel(2)})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = uc.Run(ctx, character.RollFreeCastingCommand{Character: hero, Feats: recall})
	grpD2WantInvalid(t, err, "a zero slot level")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, character.RollFreeCastingCommand{Character: hero, Feats: recall, SlotLevel: character.MustSlotLevel(2)}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v", err)
	}
}
