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
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// grpD2Warlock is a Warlock of level n with Charisma 16 knowing the invocations (the state is restored at level 20 so any option can be held, as the package tests do).
func grpD2Warlock(t *testing.T, n int, known ...string) character.Character {
	t.Helper()
	ids := make([]content.FeatureOptionID, len(known))
	for i, k := range known {
		ids[i] = content.FeatureOptionID(k)
	}
	profile := must2(character.NewWarlockProfile(character.MustWarlockLevel(20), ""))
	state := must2(character.RestoreWarlockState(profile, character.WarlockStateParts{Invocations: ids}))
	hero := grpD2Hero(t, scoresFor(t, 10, 10, 10, 10, 10, 16), grpD2CL("warlock", n))
	return must2(character.WithFeatureState(hero, character.WarlockStateKey(), state))
}

func grpD2WarlockState(t *testing.T, c character.Character) character.WarlockState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.WarlockStateKey())
	if err != nil || !ok {
		t.Fatalf("WarlockState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

func grpD2WarlockPact(t *testing.T, c character.Character) (character.WarlockPactState, bool) {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.WarlockPactStateKey())
	if err != nil {
		t.Fatal(err)
	}
	return s, ok
}

// grpD2Monster is a synthetic Tiny monster of the type and challenge rating, with made-up numbers.
func grpD2Monster(t *testing.T, id string, typ content.CreatureType, cr string) content.Monster {
	t.Helper()
	scores, saves := map[ability.Ability]ability.Score{}, map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	return must2(content.NewMonster(content.MonsterID(id), id, content.MustSourceCitation("Fixture", content.MonsterSection, 258), content.MonsterFields{
		Sizes: []content.CreatureSize{content.SizeTiny}, Type: typ, Alignment: content.AlignmentUnaligned, ArmorClass: 12,
		Initiative: must2(content.NewStatBlockInitiative(0, 10)), HitPoints: must2(content.NewHitPointsEntry(9, dice.MustExpression(2, dice.D8(), 0))),
		Speeds:    []content.SpeedEntry{must2(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, ""))},
		Abilities: must2(content.NewAbilityBlock(must2(ability.NewScores(scores)), saves)), PassivePerception: 10, ChallengeRating: content.MustChallengeRating(cr),
	}))
}

// A Warlock who knows Pact of the Blade bonds a pact weapon. A conjured weapon is looked up and must be a Simple or Martial Melee weapon; a magic weapon is bonded by its id with no
// lookup. A ranged weapon, a weapon attuned to someone else and one bonded by another Warlock are refused (SRD 5.2.1, "Pact of the Blade", p. 70).
// Requirement: doc/specs/class-warlock.md FR-081 to FR-084, FR-086 - BondPactWeapon through the public API.
// Path: happy and unhappy (state error, invalid input).
func TestWarlockBondPactWeaponFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	source := contenttest.NewFakeSource(
		grpD2Weapon("longsword", content.WeaponMartial, content.WeaponMelee, content.MasterySap),
		grpD2Weapon("longbow", content.WeaponMartial, content.WeaponRanged, content.MasterySap))
	uc := must2(character.NewBondPactWeapon(source))
	hero := grpD2Warlock(t, 5, "pact_of_the_blade")

	conjured, ok := must2(uc.Run(ctx, character.BondPactWeaponCommand{Character: hero, Weapon: "longsword"})).(character.BondPactWeaponApplied)
	if !ok || conjured.Bond().Weapon() != "longsword" || !conjured.Bond().Conjured() {
		t.Fatalf("a conjured longsword = %#v", conjured)
	}
	pact, held := grpD2WarlockPact(t, conjured.Character())
	if ev, isEv := conjured.Events()[0].(character.PactWeaponBonded); !held || !pact.Bond().Bonded() || !isEv || !ev.Conjured() {
		t.Errorf("stored bond %+v, events %v", pact.Bond(), conjured.Events())
	}
	if _, stored := grpD2WarlockPact(t, hero); stored {
		t.Error("the command's character was altered")
	}
	idle := contenttest.NewFakeSource()
	magic, ok := must2(must2(character.NewBondPactWeapon(idle)).Run(ctx, character.BondPactWeaponCommand{Character: hero, Weapon: "flame_tongue", Magic: true})).(character.BondPactWeaponApplied)
	if !ok || magic.Bond().Conjured() || magic.Bond().Weapon() != "flame_tongue" || len(idle.Queries()) != 0 {
		t.Errorf("a magic weapon = %#v (queries %v), want bonded with no lookup", magic, idle.Queries())
	}
	for _, c := range []struct {
		name string
		cmd  character.BondPactWeaponCommand
		want character.WarlockRejectionKind
	}{
		{"a ranged weapon", character.BondPactWeaponCommand{Character: hero, Weapon: "longbow"}, character.WarlockNotMeleeSimpleOrMartial},
		{"attuned by another creature", character.BondPactWeaponCommand{Character: hero, Weapon: "flame_tongue", Magic: true, Claim: character.MagicWeaponAttunedByAnotherCreature}, character.WarlockAttunedByAnother},
		{"bonded by another Warlock", character.BondPactWeaponCommand{Character: hero, Weapon: "flame_tongue", Magic: true, Claim: character.MagicWeaponBondedByAnotherWarlock}, character.WarlockBondedByAnotherWarlock},
	} {
		if rej, isRej := must2(uc.Run(ctx, c.cmd)).(character.BondPactWeaponRejected); !isRej || rej.Rejection.Kind != c.want {
			t.Errorf("%s = %#v, want %v", c.name, rej, c.want)
		}
	}
	if r, isRej := must2(uc.Run(ctx, character.BondPactWeaponCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 5)), Weapon: "longsword"})).(character.BondPactWeaponRejected); !isRej || r.Rejection.Kind != character.WarlockNotAWarlock {
		t.Errorf("a Fighter = %#v", r)
	}
	missing, ok := must2(uc.Run(ctx, character.BondPactWeaponCommand{Character: hero, Weapon: "zweihander"})).(character.BondPactWeaponContentMissing)
	if !ok || !slices.Equal(missing.Missing(), []content.MissingContent{content.ItemID("zweihander").Missing()}) {
		t.Errorf("a weapon the source lacks = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewBondPactWeapon(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.BondPactWeaponCommand{Character: hero, Weapon: "longsword"})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = uc.Run(ctx, character.BondPactWeaponCommand{Character: hero})
	grpD2WantInvalid(t, err, "a zero weapon id")
	_, err = character.NewBondPactWeapon(nil)
	grpD2WantInvalid(t, err, "a nil source")
	_, err = character.BondPactWeapon{}.Run(ctx, character.BondPactWeaponCommand{Character: hero, Weapon: "longsword"})
	grpD2WantInvalid(t, err, "the zero use case")
}

// A Warlock casts the spell of a known spell-granting invocation with no slot: Fiendish Vigor gives False Life with the maximum die, Gift of the Depths gives Water Breathing once, and
// One with Shadows gives Invisibility in Dim Light but not in Bright Light. An invocation that grants no spell and a second Gift of the Depths are refused (SRD 5.2.1,
// "Eldritch Invocations", pp. 70-71).
// Requirement: doc/specs/class-warlock.md FR-059 to FR-063, FR-065, FR-089, FR-165 - CastInvocationSpell through the public API.
// Path: happy and unhappy (state error, missing content).
func TestWarlockCastInvocationSpellFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundled := must2(data.NewBundledSource())
	uc := must2(character.NewCastInvocationSpell(bundled))
	hero := grpD2Warlock(t, 5, "fiendish_vigor", "gift_of_the_depths", "one_with_shadows", "devils_sight")
	run := func(c character.Character, option string, light content.LightLevel) character.CastInvocationSpellResult {
		return must2(uc.Run(ctx, character.CastInvocationSpellCommand{Character: c, Option: content.FeatureOptionID(option), Light: light}))
	}
	vigor, ok := run(hero, "fiendish_vigor", content.LightBright).(character.CastInvocationSpellApplied)
	if g := vigor.Grant(); !ok || g.Spell != "false_life" || g.Mode != character.CastModeFeatureGranted || g.DieRule != character.SpellDieMaximum {
		t.Fatalf("Fiendish Vigor = %#v", vigor)
	}
	depths, ok := run(hero, "gift_of_the_depths", content.LightBright).(character.CastInvocationSpellApplied)
	if !ok || depths.Grant().Spell != "water_breathing" || !grpD2WarlockState(t, depths.Character()).GiftOfTheDepthsCast() || grpD2WarlockState(t, hero).GiftOfTheDepthsCast() {
		t.Fatalf("Gift of the Depths = %#v", depths)
	}
	if ev, isEv := depths.Events()[0].(character.InvocationSpellCastWithoutSlot); !isEv || ev.Option() != "gift_of_the_depths" || ev.Spell() != "water_breathing" {
		t.Errorf("events = %v", depths.Events())
	}
	if dim, isApplied := run(hero, "one_with_shadows", content.LightDim).(character.CastInvocationSpellApplied); !isApplied || dim.Grant().Spell != "invisibility" {
		t.Errorf("One with Shadows in Dim Light = %#v", dim)
	}
	for _, c := range []struct {
		name   string
		hero   character.Character
		option string
		light  content.LightLevel
		want   character.WarlockRejectionKind
	}{
		{"a second Gift of the Depths", depths.Character(), "gift_of_the_depths", content.LightBright, character.WarlockAlreadyCast},
		{"One with Shadows in Bright Light", hero, "one_with_shadows", content.LightBright, character.WarlockLightLevelWrong},
		{"an invocation that grants no spell", hero, "devils_sight", content.LightBright, character.WarlockNotASpellGrant},
		{"a Fighter", grpD2Flat(t, 10, grpD2CL("fighter", 5)), "fiendish_vigor", content.LightBright, character.WarlockNotAWarlock},
	} {
		if rej, isRej := run(c.hero, c.option, c.light).(character.CastInvocationSpellRejected); !isRej || rej.Rejection.Kind != c.want {
			t.Errorf("%s = %#v, want %v", c.name, rej, c.want)
		}
	}
	empty := must2(character.NewCastInvocationSpell(contenttest.NewFakeSource()))
	missing, ok := must2(empty.Run(ctx, character.CastInvocationSpellCommand{Character: hero, Option: "fiendish_vigor", Light: content.LightBright})).(character.CastInvocationSpellContentMissing)
	if !ok || !slices.Equal(missing.Missing(), []content.MissingContent{character.WarlockClassID.Missing()}) {
		t.Errorf("a source without the Warlock = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewCastInvocationSpell(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.CastInvocationSpellCommand{Character: hero, Option: "fiendish_vigor"})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewCastInvocationSpell(nil)
	grpD2WantInvalid(t, err, "a nil source")
	_, err = uc.Run(ctx, character.CastInvocationSpellCommand{})
	grpD2WantInvalid(t, err, "a zero character")
}

// A Warlock with Pact of the Chain picks a familiar form: the Imp as a special form, a CR 0 Beast as a normal form, and a Beast of CR 1/4 offered as a normal form is refused (SRD 5.2.1,
// "Pact of the Chain", p. 70).
// Requirement: doc/specs/class-warlock.md FR-042, FR-090, FR-141, FR-142 - ChooseWarlockFamiliarForm through the public API.
// Path: happy and unhappy (not offered, missing content, collaborator failure, invalid input).
func TestWarlockChooseFamiliarFormFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	source := contenttest.NewFakeSource(grpD2Monster(t, "imp", content.CreatureTypeFiend, "1"), grpD2Monster(t, "wolf", content.CreatureTypeBeast, "1/4"), grpD2Monster(t, "cat", content.CreatureTypeBeast, "0"))
	uc := must2(character.NewChooseWarlockFamiliarForm(source))
	hero := grpD2Warlock(t, 5, "pact_of_the_chain")
	for _, form := range []character.FamiliarForm{character.SpecialFamiliarForm("imp"), character.NormalFamiliarForm("cat")} {
		got, ok := must2(uc.Run(ctx, character.ChooseWarlockFamiliarFormCommand{Character: hero, Form: form})).(character.ChooseWarlockFamiliarFormApplied)
		if !ok {
			t.Fatalf("%v was not applied", form)
		}
		pact, _ := grpD2WarlockPact(t, got.Character())
		if stored, held := pact.Familiar().Form(); !held || stored != form {
			t.Errorf("stored form = %v, %v, want %v", stored, held, form)
		}
		if ev, isEv := got.Events()[0].(character.FamiliarFormChosen); !isEv || ev.Form() != form {
			t.Errorf("events = %v", got.Events())
		}
	}
	if _, stored := grpD2WarlockPact(t, hero); stored {
		t.Error("the command's character was altered")
	}
	rej, ok := must2(uc.Run(ctx, character.ChooseWarlockFamiliarFormCommand{Character: hero, Form: character.NormalFamiliarForm("wolf")})).(character.ChooseWarlockFamiliarFormRejected)
	if !ok || rej.Rejection.Kind != character.WarlockFamiliarFormNotOffered || rej.Rejection.Monster != "wolf" {
		t.Errorf("a CR 1/4 normal form = %#v", rej)
	}
	if r, isRej := must2(uc.Run(ctx, character.ChooseWarlockFamiliarFormCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 5)), Form: character.SpecialFamiliarForm("imp")})).(character.ChooseWarlockFamiliarFormRejected); !isRej || r.Rejection.Kind != character.WarlockNotAWarlock {
		t.Errorf("a Fighter = %#v", r)
	}
	missing := must2(must2(character.NewChooseWarlockFamiliarForm(contenttest.NewFakeSource())).Run(ctx, character.ChooseWarlockFamiliarFormCommand{Character: hero, Form: character.SpecialFamiliarForm("imp")}))
	if m, isMissing := missing.(character.ChooseWarlockFamiliarFormContentMissing); !isMissing || !slices.Equal(m.Missing(), []content.MissingContent{content.MonsterID("imp").Missing()}) {
		t.Errorf("a monster the source lacks = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewChooseWarlockFamiliarForm(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.ChooseWarlockFamiliarFormCommand{Character: hero, Form: character.SpecialFamiliarForm("imp")})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = uc.Run(ctx, character.ChooseWarlockFamiliarFormCommand{Character: hero})
	grpD2WantInvalid(t, err, "a zero form")
	_, err = character.NewChooseWarlockFamiliarForm(nil)
	grpD2WantInvalid(t, err, "a nil source")
}

// A level 17 Warlock chooses its level 9 Mystic Arcanum spell, replaces it on a later level, and a second replacement in the same level-up is refused (SRD 5.2.1, "Level 11: Mystic Arcanum", p. 71).
// Requirement: doc/specs/class-warlock.md FR-042, FR-117, FR-118, FR-120, FR-141, FR-142 - ChooseWarlockMysticArcanum through the public API.
// Path: happy and unhappy (boundary, missing content, collaborator failure).
func TestWarlockChooseMysticArcanumFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundled := must2(data.NewBundledSource())
	uc := must2(character.NewChooseWarlockMysticArcanum(bundled))
	hero := grpD2Hero(t, scoresFor(t, 10, 10, 10, 10, 10, 16), grpD2CL("warlock", 17))
	hero = must2(character.WithFeatureState(hero, character.WarlockStateKey(), character.WarlockState{}))
	got, ok := must2(uc.Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: hero, Spell: "power_word_kill"})).(character.ChooseWarlockMysticArcanumApplied)
	if !ok {
		t.Fatal("Power Word Kill at level 17 was not applied")
	}
	if spell, held := grpD2WarlockState(t, got.Character()).Arcanum(9); !held || spell != "power_word_kill" {
		t.Errorf("level 9 arcanum = %v, %v", spell, held)
	}
	if _, held := grpD2WarlockState(t, hero).Arcanum(9); held {
		t.Error("the command's character was altered")
	}
	if ev, isEv := got.Events()[0].(character.MysticArcanumChosen); !isEv || ev.Replaced() || ev.Spell() != "power_word_kill" || ev.SpellLevel() != 9 {
		t.Errorf("events = %v", got.Events())
	}
	again, ok := must2(uc.Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: got.Character(), Spell: "power_word_kill"})).(character.ChooseWarlockMysticArcanumApplied)
	if !ok || !again.Events()[0].(character.MysticArcanumChosen).Replaced() {
		t.Errorf("a replacement = %#v", again)
	}
	if rej, isRej := must2(uc.Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: got.Character(), Spell: "power_word_kill", Replacements: 1})).(character.ChooseWarlockMysticArcanumRejected); !isRej || rej.Rejection.Kind != character.WarlockArcanumTooManyReplacements {
		t.Errorf("a second replacement = %#v", rej)
	}
	if rej, isRej := must2(uc.Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: hero, Spell: "fire_bolt"})).(character.ChooseWarlockMysticArcanumRejected); !isRej || (rej.Rejection.Kind != character.WarlockArcanumWrongSpellLevel && rej.Rejection.Kind != character.WarlockArcanumNotOnWarlockList) {
		t.Errorf("a cantrip as the arcanum = %#v", rej)
	}
	if r, isRej := must2(uc.Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 17)), Spell: "power_word_kill"})).(character.ChooseWarlockMysticArcanumRejected); !isRej || r.Rejection.Kind != character.WarlockNotAWarlock {
		t.Errorf("a Fighter = %#v", r)
	}
	missing := must2(must2(character.NewChooseWarlockMysticArcanum(contenttest.NewFakeSource())).Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: hero, Spell: "power_word_kill"}))
	if m, isMissing := missing.(character.ChooseWarlockMysticArcanumContentMissing); !isMissing || !slices.Equal(m.Missing(), []content.MissingContent{content.SpellID("power_word_kill").Missing()}) {
		t.Errorf("a spell the source lacks = %#v", missing)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewChooseWarlockMysticArcanum(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, character.ChooseWarlockMysticArcanumCommand{Character: hero, Spell: "power_word_kill"})
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewChooseWarlockMysticArcanum(nil)
	grpD2WantInvalid(t, err, "a nil source")
	_, err = uc.Run(ctx, character.ChooseWarlockMysticArcanumCommand{})
	grpD2WantInvalid(t, err, "a zero character")
}

// A Warlock with Pact of the Tome conjures a Book of Shadows of three cantrips and two spells after a Long Rest; with Gift of the Protectors the book also has a page. A spell that is
// already prepared is refused, and spell ids that are invalid are invalid input (SRD 5.2.1, "Pact of the Tome", p. 70).
// Requirement: doc/specs/class-warlock.md FR-042, FR-097 to FR-101, FR-141, FR-142, FR-152 - ConjureBookOfShadows through the public API.
// Path: happy and unhappy (state error, missing content, collaborator failure, invalid input).
func TestWarlockConjureBookOfShadowsFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundled := must2(data.NewBundledSource())
	uc := must2(character.NewConjureBookOfShadows(bundled))
	hero := grpD2Warlock(t, 5, "pact_of_the_tome", "gift_of_the_protectors")
	cmd := character.ConjureBookOfShadowsCommand{Character: hero, Rest: character.LongRest, Cantrips: []content.SpellID{"acid_splash", "dancing_lights", "druidcraft"},
		Spells: []content.SpellID{"detect_magic", "speak_with_animals"}, Prepared: []content.SpellID{"bless"}}
	got, ok := must2(uc.Run(ctx, cmd)).(character.ConjureBookOfShadowsApplied)
	if !ok {
		t.Fatal("the Book of Shadows was not conjured")
	}
	pact, _ := grpD2WarlockPact(t, got.Character())
	if !pact.Book().HasPage() || !slices.Equal(pact.Book().Spells(), []content.SpellID{"detect_magic", "speak_with_animals"}) || !got.Book().HasPage() || !slices.Equal(got.Book().Cantrips(), pact.Book().Cantrips()) {
		t.Errorf("book = %+v / %+v", pact.Book(), got.Book())
	}
	if ev, isEv := got.Events()[0].(character.BookOfShadowsConjured); !isEv || !ev.Book().Conjured() {
		t.Errorf("events = %v", got.Events())
	}
	if _, stored := grpD2WarlockPact(t, hero); stored {
		t.Error("the command's character was altered")
	}
	already := cmd
	already.Prepared = []content.SpellID{"druidcraft"}
	rej, ok := must2(uc.Run(ctx, already)).(character.ConjureBookOfShadowsRejected)
	if !ok || rej.Rejection.Kind != character.WarlockBookSpellRefused || rej.Rejection.BookRule != character.BookSpellAlreadyPrepared || rej.Rejection.Spell != "druidcraft" {
		t.Errorf("a prepared spell = %#v", rej)
	}
	noRest := cmd
	noRest.Rest = 0
	if r, isRej := must2(uc.Run(ctx, noRest)).(character.ConjureBookOfShadowsRejected); !isRej || r.Rejection.Kind != character.WarlockNotAfterRest {
		t.Errorf("with no rest finished = %#v, want not after a rest", r)
	}
	if _, isApplied := must2(uc.Run(ctx, grpD2WithRest(cmd, character.ShortRest))).(character.ConjureBookOfShadowsApplied); !isApplied {
		t.Error("a finished Short Rest also lets the Warlock conjure the book")
	}
	bad := cmd
	bad.Spells = []content.SpellID{"bad id", "x"}
	_, err := uc.Run(ctx, bad)
	grpD2WantInvalid(t, err, "invalid spell ids")
	var want []content.MissingContent
	for _, id := range []content.SpellID{"acid_splash", "dancing_lights", "druidcraft", "detect_magic", "speak_with_animals"} {
		want = append(want, id.Missing())
	}
	missing := must2(must2(character.NewConjureBookOfShadows(contenttest.NewFakeSource())).Run(ctx, cmd))
	if m, isMissing := missing.(character.ConjureBookOfShadowsContentMissing); !isMissing || !slices.Equal(m.Missing(), want) {
		t.Errorf("every spell missing = %#v, want %v", missing, want)
	}
	boom := errors.New("source down")
	res, err := must2(character.NewConjureBookOfShadows(contenttest.NewFakeSource().FailLookup(boom))).Run(ctx, cmd)
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)
	_, err = character.NewConjureBookOfShadows(nil)
	grpD2WantInvalid(t, err, "a nil source")
	if r, isRej := must2(uc.Run(ctx, character.ConjureBookOfShadowsCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 5)), Rest: character.LongRest})).(character.ConjureBookOfShadowsRejected); !isRej || r.Rejection.Kind != character.WarlockNotAWarlock {
		t.Errorf("a Fighter = %#v", r)
	}
}

// Lifedrinker on a pact weapon hit adds 1d6 Necrotic damage and lets the Warlock expend a Hit Point Die: the die is rolled through the dice port and the result plus the Constitution modifier is
// regained. Declining the die never reaches the roller. A hit that is not with the pact weapon, a damage type other than Necrotic, Psychic or Radiant, and no die left are refused with no roll
// (SRD 5.2.1, "Lifedrinker", p. 71).
// Requirement: doc/specs/class-warlock.md FR-077, FR-078, FR-145, FR-149 - UseLifedrinker through the public API.
// Path: happy (extremes 1 and 8), unhappy (state error, collaborator failure, invalid input).
func TestWarlockUseLifedrinkerFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hero := grpD2Warlock(t, 5, "lifedrinker")
	pool := must2(character.NewHitDicePool(map[content.HitPointDie]int{content.HitDieD8: 3}))
	cmd := func(edit func(*character.UseLifedrinkerCommand)) character.UseLifedrinkerCommand {
		c := character.UseLifedrinkerCommand{Character: hero, Attack: character.WarlockPactWeaponHit, DamageType: content.DamageNecrotic, Choice: character.HitDieExpend, Pool: pool,
			Die: content.HitDieD8, Constitution: ability.MustScore(14).Modifier()}
		if edit != nil {
			edit(&c)
		}
		return c
	}
	for _, face := range []int{1, 8} {
		roller := dicetest.NewFixedRoller(face)
		got, ok := must2(must2(character.NewUseLifedrinker(roller)).Run(ctx, cmd(nil))).(character.UseLifedrinkerApplied)
		healed, hasHeal := got.HitPointsRegained()
		if !ok || !hasHeal || healed != face+2 || got.SpentHitDice().Count(content.HitDieD8) != 1 || got.ExtraDamage().Notation() != "1d6" {
			t.Errorf("face %d = %#v, healed %d", face, got, healed)
		}
		if ev, isEv := got.Events()[0].(character.LifedrinkerUsed); !isEv || ev.HitPointsRegained() != face+2 {
			t.Errorf("face %d events = %v", face, got.Events())
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D8(), 0) {
			t.Errorf("face %d requests = %v, want one 1d8", face, reqs)
		}
	}
	spent := must2(character.NewSpentHitDice(map[content.HitPointDie]int{content.HitDieD8: 3}))
	idle := dicetest.NewFixedRoller(5)
	uc := must2(character.NewUseLifedrinker(idle))
	declined, ok := must2(uc.Run(ctx, cmd(func(c *character.UseLifedrinkerCommand) { c.Choice, c.Spent = character.HitDieDecline, spent }))).(character.UseLifedrinkerApplied)
	if _, healed := declined.HitPointsRegained(); !ok || healed || !declined.SpentHitDice().Equal(spent) {
		t.Errorf("declining the die = %#v", declined)
	}
	for name, edit := range map[string]func(*character.UseLifedrinkerCommand){
		"fire damage": func(c *character.UseLifedrinkerCommand) { c.DamageType = content.DamageFire },
		"a miss":      func(c *character.UseLifedrinkerCommand) { c.Attack = character.WarlockMissed },
		"no die left": func(c *character.UseLifedrinkerCommand) { c.Spent = spent },
	} {
		if _, isRej := must2(uc.Run(ctx, cmd(edit))).(character.UseLifedrinkerRejected); !isRej {
			t.Errorf("%s was not rejected", name)
		}
	}
	if len(idle.Requests()) != 0 {
		t.Errorf("a roll happened without need: %v", idle.Requests())
	}
	_, err := uc.Run(ctx, cmd(func(c *character.UseLifedrinkerCommand) { c.Die = 0 }))
	grpD2WantInvalid(t, err, "an invalid die")
	_, err = character.NewUseLifedrinker(nil)
	grpD2WantInvalid(t, err, "a nil roller")
	_, err = character.UseLifedrinker{}.Run(ctx, cmd(nil))
	grpD2WantInvalid(t, err, "the zero use case")
	boom := errors.New("roller down")
	for name, c := range map[string]struct {
		roller character.Roller
		cause  error
	}{
		"an error": {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		"empty":    {dicetest.NewFixedRoller(), dicetest.ErrExhausted},
		"a panic":  {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), boom},
		"too high": {dicetest.NewFixedRoller(9), dicetest.ErrValueOutOfRange},
	} {
		res, err := must2(character.NewUseLifedrinker(c.roller)).Run(ctx, cmd(nil))
		if res != nil {
			t.Errorf("%s: returned %#v", name, res)
		}
		grpD2WantPortFailure(t, err, dice.Port, c.cause)
	}
}

func grpD2WithRest(c character.ConjureBookOfShadowsCommand, rest character.RestKind) character.ConjureBookOfShadowsCommand {
	c.Rest = rest
	return c
}
