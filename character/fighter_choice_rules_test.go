package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// fgtDecide runs the rules and returns the verdict.
func fgtDecide(t *testing.T, facts character.FighterChoiceFacts, choice character.FighterOptionChoice) character.FighterChoiceVerdict {
	t.Helper()
	v, err := character.FighterChoiceRules{}.Decide(facts, choice)
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	return v
}

// fgtProblems returns the reasons of a refusal, failing on an acceptance.
func fgtProblems(t *testing.T, v character.FighterChoiceVerdict) []character.FighterProblem {
	t.Helper()
	r, ok := v.(character.FighterChoiceRefused)
	if !ok {
		t.Fatalf("verdict = %#v, want a refusal", v)
	}
	return r.Problems()
}

func fgtReasons(ps []character.FighterProblem) []character.FighterReason {
	out := make([]character.FighterReason, len(ps))
	for i, p := range ps {
		out[i] = p.Reason
	}
	return out
}

func fgtAccepted(t *testing.T, v character.FighterChoiceVerdict) character.FighterChoiceAccepted {
	t.Helper()
	a, ok := v.(character.FighterChoiceAccepted)
	if !ok {
		t.Fatalf("verdict = %#v, want an acceptance", v)
	}
	return a
}

func styleChoice(id string) character.FightingStyleChoice {
	return must(character.NewFightingStyleChoice(content.FeatID(id)))
}

// styleFacts are the facts of a Fighter of the given level choosing the feat.
func styleFacts(t *testing.T, level int, state character.FighterState, feats character.TakenFeats, feat content.Feat) character.FighterChoiceFacts {
	t.Helper()
	return character.FighterChoiceFacts{Character: fgtWith(t, fgtFighter(t, level), state), Feats: feats, State: state, Stats: fgtStats(t, level), Feat: feat}
}

// Requirement: doc/specs/class-fighter.md FR-032, FR-035, FR-039, FR-163 - choosing the level 1 Fighting Style takes the feat through
// FeatTakingRules (the Fighter counts as holding the Fighting Style feature), records only the level 1 marker in the FighterState and returns
// the events of taking the feat then FighterFightingStyleChosen; the inputs are unchanged.
// Path: happy.
func TestFighterChoiceRulesChooseFightingStyle(t *testing.T) {
	t.Parallel()
	facts := styleFacts(t, 1, character.InitialFighterState(), character.TakenFeats{}, fgtStyle("archery"))
	before := facts.Character
	a := fgtAccepted(t, fgtDecide(t, facts, character.FighterChooseFightingStyle{Choice: styleChoice("archery")}))
	state := fgtStateOf(t, a.Character())
	if state.FightingStyle() != "archery" || state.AdditionalFightingStyle() != "" || !a.Feats().Holds("archery") || a.Feats().Len() != 1 {
		t.Errorf("state %#v, feats %v", state, a.Feats().All())
	}
	events := a.Events()
	if len(events) != 2 {
		t.Fatalf("events = %#v, want FeatTaken then FighterFightingStyleChosen", events)
	}
	if _, ok := events[0].(character.FeatTaken); !ok {
		t.Errorf("first event = %#v, want FeatTaken", events[0])
	}
	chosen, ok := events[1].(character.FighterFightingStyleChosen)
	if !ok || chosen.Feat() != "archery" || chosen.Additional() || chosen.CharacterID() != facts.Character.ID() {
		t.Errorf("second event = %#v", events[1])
	}
	if !facts.Character.Equal(before) {
		t.Error("Decide altered the character of the facts")
	}
}

// Requirement: doc/specs/class-fighter.md FR-121, FR-156, FR-163, section 5 - ChooseFightingStyle reports its reasons together in the order
// fightingStyleAlreadyChosen, notAFightingStyleFeat, featNotRepeatable, with the offending feat; a second choice suppresses
// featNotRepeatable; a feat held and not Repeatable is featNotRepeatable alone for a style feat and together with notAFightingStyleFeat
// for one outside the category; a Repeatable held style feat is accepted.
// Path: per case (unhappy: state error, rule interaction).
func TestFighterChoiceRulesChooseFightingStyleRejections(t *testing.T) {
	t.Parallel()
	defense := character.MustFighterState(0, 0, 0, "defense", "")
	for _, c := range []struct {
		name  string
		state character.FighterState
		feats character.TakenFeats
		feat  content.Feat
		want  []character.FighterReason
	}{
		{"a second choice", defense, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery"), []character.FighterReason{character.FighterFightingStyleAlreadyChosen}},
		{"a second choice of the same feat", defense, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("defense"), []character.FighterReason{character.FighterFightingStyleAlreadyChosen}},
		{"not a Fighting Style feat", character.InitialFighterState(), character.TakenFeats{}, alertFeat(), []character.FighterReason{character.FighterNotAFightingStyleFeat}},
		{"a held feat outside the category", character.InitialFighterState(), fgtFeats(t, content.FeatOrigin, "alert"), alertFeat(), []character.FighterReason{character.FighterNotAFightingStyleFeat, character.FighterFeatNotRepeatable}},
		{"a held style feat", character.InitialFighterState(), fgtFeats(t, content.FeatFightingStyle, "archery"), fgtStyle("archery"), []character.FighterReason{character.FighterFeatNotRepeatable}},
		{"already chosen and not a style feat", defense, fgtFeats(t, content.FeatOrigin, "alert"), alertFeat(), []character.FighterReason{character.FighterFightingStyleAlreadyChosen, character.FighterNotAFightingStyleFeat}},
		{"a feat with no behaviour", character.InitialFighterState(), character.TakenFeats{}, fgtStyle("not_an_srd_feat"), []character.FighterReason{character.FighterNotAFightingStyleFeat}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ps := fgtProblems(t, fgtDecide(t, styleFacts(t, 1, c.state, c.feats, c.feat), character.FighterChooseFightingStyle{Choice: styleChoice(string(c.feat.ID()))}))
			if got := fgtReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
			if ps[len(ps)-1].Feat == "" && ps[len(ps)-1].Reason != character.FighterFightingStyleAlreadyChosen {
				t.Errorf("a feat problem must carry the feat id: %+v", ps)
			}
		})
	}
	facts := styleFacts(t, 1, character.InitialFighterState(), fgtFeats(t, content.FeatFightingStyle, "archery"), fgtRepeatableStyle("archery"))
	a := fgtAccepted(t, fgtDecide(t, facts, character.FighterChooseFightingStyle{Choice: styleChoice("archery")}))
	if a.Feats().Len() != 2 {
		t.Errorf("a Repeatable held style feat: feats = %v, want it taken twice", a.Feats().All())
	}
}

// fgtReplaceFacts is a Fighter of level 5 who chose defense at level 1 and holds it, with the LevelGained event of level 5.
func fgtReplaceFacts(t *testing.T, feat content.Feat, held ...string) (character.FighterChoiceFacts, character.LevelGained) {
	t.Helper()
	four := fgtWith(t, fgtFighter(t, 4), character.MustFighterState(0, 0, 0, "defense", ""))
	five, lg := fgtLevelUp(t, four, "fighter")
	if len(held) == 0 {
		held = []string{"defense"}
	}
	state := character.MustFighterState(0, 0, 0, "defense", "")
	return character.FighterChoiceFacts{Character: five, Feats: fgtFeats(t, content.FeatFightingStyle, held...), State: state, Stats: fgtStats(t, 5), Feat: feat}, lg
}

// Requirement: doc/specs/class-fighter.md FR-036, FR-037, FR-157, FR-163 - replacing the level 1 style on gaining the Fighter level removes the
// old feat first, takes the new one under FeatReplacement, keeps the Additional Fighting Style, and returns FeatReplaced, FeatTaken and
// FighterFightingStyleReplaced.
// Path: happy.
func TestFighterChoiceRulesReplaceFightingStyle(t *testing.T) {
	t.Parallel()
	facts, lg := fgtReplaceFacts(t, fgtStyle("archery"), "defense", "two_weapon_fighting")
	facts.State = character.MustFighterState(0, 0, 0, "defense", "two_weapon_fighting")
	a := fgtAccepted(t, fgtDecide(t, facts, character.FighterReplaceFightingStyle{Choice: styleChoice("archery"), LevelGained: lg}))
	state := fgtStateOf(t, a.Character())
	if state.FightingStyle() != "archery" || state.AdditionalFightingStyle() != "two_weapon_fighting" || a.Feats().Holds("defense") || !a.Feats().Holds("archery") || !a.Feats().Holds("two_weapon_fighting") {
		t.Errorf("state %#v, feats %v", state, a.Feats().All())
	}
	events := a.Events()
	if len(events) != 3 {
		t.Fatalf("events = %#v", events)
	}
	// A new feat whose id sorts after the old one is not the same feat.
	after := fgtAccepted(t, fgtDecide(t, character.FighterChoiceFacts{Character: facts.Character, Feats: fgtFeats(t, content.FeatFightingStyle, "defense"), State: character.MustFighterState(0, 0, 0, "defense", ""), Stats: facts.Stats, Feat: fgtStyle("two_weapon_fighting")},
		character.FighterReplaceFightingStyle{Choice: styleChoice("two_weapon_fighting"), LevelGained: lg}))
	if fgtStateOf(t, after.Character()).FightingStyle() != "two_weapon_fighting" {
		t.Errorf("replacement by a later-sorting feat: state %#v", fgtStateOf(t, after.Character()))
	}
	_, replaced := events[0].(character.FeatReplaced)
	_, taken := events[1].(character.FeatTaken)
	swapped, last := events[2].(character.FighterFightingStyleReplaced)
	if !replaced || !taken || !last || swapped.Old() != "defense" || swapped.Feat() != "archery" || swapped.CharacterID() != facts.Character.ID() {
		t.Errorf("events = %#v", events)
	}
}

// Requirement: doc/specs/class-fighter.md FR-037, FR-038, FR-121, FR-157, FR-159, FR-163 - a replacement is rejected with, in order,
// noStyleToReplace (no level 1 style), eventNotForCharacter (checked before the level), noLevelGained (an event of another class or of an
// earlier Fighter level), notAFightingStyleFeat, sameFeat and featNotRepeatable (the new feat equals the held Additional style), reported together.
// Path: per case (unhappy: state error, invalid input, boundary).
func TestFighterChoiceRulesReplaceFightingStyleRejections(t *testing.T) {
	t.Parallel()
	facts, lg := fgtReplaceFacts(t, fgtStyle("archery"))
	earlier := fgtWith(t, fgtFighter(t, 3), character.MustFighterState(0, 0, 0, "defense", ""))
	_, lgEarlier := fgtLevelUp(t, earlier, "fighter") // the event of level 4, while the character is level 5
	multi := restored(t, 355000, 200, flatScores(t, 14), cl("fighter", 4))
	_, lgWizard := fgtLevelUp(t, multi, "wizard")
	fgtOther := func(id string) character.LevelGained {
		c := must(character.Restore(character.Parts{ID: character.MustIdentifier(id), ClassLevels: levels(cl("fighter", 4)), Species: "human", Background: "soldier", ExperiencePoints: character.MustExperiencePoints(355000),
			AbilityScores: flatScores(t, 10), HitPointMaximum: character.MustHitPointMaximum(40), HitDicePool: character.MustHitDicePoolFromLevels(levels(cl("fighter", 4)), map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}),
			HeroicInspiration: character.NotInspired}))
		_, lg := fgtLevelUp(t, c, "fighter")
		return lg
	}
	lgOther, lgLowerID := fgtOther("pc-2"), fgtOther("a-1")
	_, lgLater := fgtLevelUp(t, facts.Character, "fighter") // the event of level 6, while the character is level 5
	noStyle := facts
	noStyle.State = character.InitialFighterState()
	for _, c := range []struct {
		name  string
		facts character.FighterChoiceFacts
		feat  string
		lg    character.LevelGained
		want  []character.FighterReason
	}{
		{"no level 1 style", noStyle, "archery", lg, []character.FighterReason{character.FighterNoStyleToReplace}},
		{"another character's event", facts, "archery", lgOther, []character.FighterReason{character.FighterEventNotForCharacter}},
		{"an event of an earlier level", facts, "archery", lgEarlier, []character.FighterReason{character.FighterNoLevelGained}},
		{"an event of another class", facts, "archery", lgWizard, []character.FighterReason{character.FighterNoLevelGained}},
		{"an event of a later level", facts, "archery", lgLater, []character.FighterReason{character.FighterNoLevelGained}},
		{"another character's event with a lower id", facts, "archery", lgLowerID, []character.FighterReason{character.FighterEventNotForCharacter}},
		{"the same feat", facts, "defense", lg, []character.FighterReason{character.FighterSameFeat}},
		{"not a style feat", facts, "alert", lg, []character.FighterReason{character.FighterNotAFightingStyleFeat}},
		{"a held Additional style", func() character.FighterChoiceFacts {
			f := facts
			f.Feats = fgtFeats(t, content.FeatFightingStyle, "defense", "archery")
			return f
		}(), "archery", lg, []character.FighterReason{character.FighterFeatNotRepeatable}},
		{"every problem", func() character.FighterChoiceFacts {
			f := noStyle
			f.Feats = fgtFeats(t, content.FeatOrigin, "alert")
			return f
		}(), "alert", lgEarlier, []character.FighterReason{character.FighterNoStyleToReplace, character.FighterNoLevelGained, character.FighterNotAFightingStyleFeat, character.FighterFeatNotRepeatable}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := c.facts
			f.Feat = map[string]content.Feat{"archery": fgtStyle("archery"), "defense": fgtStyle("defense"), "alert": alertFeat()}[c.feat]
			ps := fgtProblems(t, fgtDecide(t, f, character.FighterReplaceFightingStyle{Choice: styleChoice(c.feat), LevelGained: c.lg}))
			if got := fgtReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
		})
	}
	// An old style marker that TakenFeats does not hold is no style to replace.
	missing := facts
	missing.Feats = character.TakenFeats{}
	if got := fgtReasons(fgtProblems(t, fgtDecide(t, missing, character.FighterReplaceFightingStyle{Choice: styleChoice("archery"), LevelGained: lg}))); !slices.Equal(got, []character.FighterReason{character.FighterNoStyleToReplace}) {
		t.Errorf("an unheld marker: reasons = %v", got)
	}
}

// Requirement: doc/specs/class-fighter.md FR-039, FR-040, FR-158, FR-163 - a Champion of Fighter level 7 takes the Additional Fighting Style into
// TakenFeats and the additional marker only, leaving the level 1 marker; before level 7, without the Champion or with another subclass it is
// featureNotGained; a second one is additionalStyleAlreadyChosen; a held non-Repeatable feat is featNotRepeatable; all reported together.
// Path: per case (unhappy: boundary, state error).
func TestFighterChoiceRulesAdditionalFightingStyle(t *testing.T) {
	t.Parallel()
	level1 := character.MustFighterState(0, 0, 0, "defense", "")
	facts := func(level int, state character.FighterState, feats character.TakenFeats, feat content.Feat) character.FighterChoiceFacts {
		return styleFacts(t, level, state, feats, feat)
	}
	a := fgtAccepted(t, fgtDecide(t, facts(7, level1, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery")),
		character.FighterChooseAdditionalFightingStyle{Choice: styleChoice("archery"), Subclass: champion}))
	state := fgtStateOf(t, a.Character())
	last, ok := a.Events()[len(a.Events())-1].(character.FighterFightingStyleChosen)
	if state.FightingStyle() != "defense" || state.AdditionalFightingStyle() != "archery" || !a.Feats().Holds("archery") || !ok || !last.Additional() {
		t.Errorf("state %#v, last event %#v", state, a.Events())
	}
	for _, level := range []int{8, 20} {
		deep := fgtAccepted(t, fgtDecide(t, facts(level, level1, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery")),
			character.FighterChooseAdditionalFightingStyle{Choice: styleChoice("archery"), Subclass: champion}))
		if fgtStateOf(t, deep.Character()).AdditionalFightingStyle() != "archery" {
			t.Errorf("Champion level %d: the Additional Fighting Style was refused", level)
		}
	}
	for _, c := range []struct {
		name  string
		level int
		sub   content.SubclassID
		state character.FighterState
		feats character.TakenFeats
		feat  content.Feat
		want  []character.FighterReason
	}{
		{"level 6", 6, champion, level1, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery"), []character.FighterReason{character.FighterFeatureNotGained}},
		{"no subclass", 7, "", level1, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery"), []character.FighterReason{character.FighterFeatureNotGained}},
		{"another subclass", 20, "another", level1, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery"), []character.FighterReason{character.FighterFeatureNotGained}},
		{"a second one", 7, champion, character.MustFighterState(0, 0, 0, "defense", "archery"), fgtFeats(t, content.FeatFightingStyle, "defense", "archery"), fgtStyle("two_weapon_fighting"), []character.FighterReason{character.FighterAdditionalStyleAlreadyChosen}},
		{"the level 1 style again", 7, champion, level1, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("defense"), []character.FighterReason{character.FighterFeatNotRepeatable}},
		{"not a style feat", 7, champion, level1, fgtFeats(t, content.FeatFightingStyle, "defense"), alertFeat(), []character.FighterReason{character.FighterNotAFightingStyleFeat}},
		{"every problem", 6, "", character.MustFighterState(0, 0, 0, "defense", "archery"), fgtFeats(t, content.FeatOrigin, "alert"), alertFeat(),
			[]character.FighterReason{character.FighterFeatureNotGained, character.FighterAdditionalStyleAlreadyChosen, character.FighterNotAFightingStyleFeat, character.FighterFeatNotRepeatable}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ps := fgtProblems(t, fgtDecide(t, facts(c.level, c.state, c.feats, c.feat), character.FighterChooseAdditionalFightingStyle{Choice: styleChoice(string(c.feat.ID())), Subclass: c.sub}))
			if got := fgtReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
		})
	}
}

func masteryFacts(t *testing.T, level int, current []string, weapons ...string) character.FighterChoiceFacts {
	t.Helper()
	facts := character.FighterChoiceFacts{Character: fgtFighter(t, level), State: character.InitialFighterState(), Stats: fgtStats(t, level), Weapons: fgtWeapons(t, weapons...)}
	if len(current) > 0 {
		facts.Masteries = must(createMasteries(t, current...))
	}
	return facts
}

// Requirement: doc/specs/class-fighter.md FR-023, FR-163 - Weapon Mastery is created with exactly the allowed count at level 1 (3) and extended at level 4
// (4 kinds keeping the old ones in order) through the shared rules; the choice lands under the shared weapon_mastery_choice key, the
// FighterState holds no mastery and FighterWeaponMasteriesChosen carries the kinds.
// Path: happy.
func TestFighterChoiceRulesChooseWeaponMasteries(t *testing.T) {
	t.Parallel()
	a := fgtAccepted(t, fgtDecide(t, masteryFacts(t, 1, nil, "greatsword", "longbow", "mace"),
		character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}))
	access := must(character.WeaponMasteryAccessFor(a.Character().FeatureStates())).(character.MasteryFromChoice)
	chosen, ok := a.Events()[0].(character.FighterWeaponMasteriesChosen)
	if !slices.Equal(access.Choice.Kinds(), ids("greatsword", "longbow", "mace")) || !ok || !slices.Equal(chosen.Kinds(), ids("greatsword", "longbow", "mace")) || len(a.Events()) != 1 {
		t.Errorf("choice %v, events %#v", access.Choice.Kinds(), a.Events())
	}
	if _, ok, _ := character.ReadFeatureState(a.Character().FeatureStates(), character.FighterStateKey()); ok {
		t.Error("a mastery choice must not write the FighterState")
	}
	ext := fgtAccepted(t, fgtDecide(t, masteryFacts(t, 4, []string{"greatsword", "longbow", "mace"}, "greatsword", "longbow", "mace", "rapier"),
		character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace", "rapier")}))
	if got := must(character.WeaponMasteryAccessFor(ext.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(); !slices.Equal(got, ids("greatsword", "longbow", "mace", "rapier")) {
		t.Errorf("extended choice = %v", got)
	}
	chosen.Kinds()[0] = "tampered"
	if chosen.Kinds()[0] == "tampered" {
		t.Error("Kinds returned a shared slice")
	}
}

// Requirement: doc/specs/class-fighter.md FR-121, FR-163, section 5 - ChooseWeaponMasteries maps the shared problems to wrongMasteryCount,
// duplicateMastery, notSimpleOrMartialWeapon (an item that is not a weapon) and dropsExistingMastery and reports them together in that order,
// each with the weapon id where it names one.
// Path: per case (unhappy: boundary, invalid input, rule interaction).
func TestFighterChoiceRulesChooseWeaponMasteriesRejections(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		level   int
		current []string
		kinds   []string
		weapons []string
		want    []character.FighterReason
	}{
		{"two kinds at level 1", 1, nil, []string{"greatsword", "mace"}, []string{"greatsword", "mace"}, []character.FighterReason{character.FighterWrongMasteryCount}},
		{"four kinds at level 3", 3, nil, []string{"greatsword", "mace", "club", "dagger"}, []string{"greatsword", "mace", "club", "dagger"}, []character.FighterReason{character.FighterWrongMasteryCount}},
		{"a repeated kind", 1, nil, []string{"greatsword", "mace", "mace"}, []string{"greatsword", "mace"}, []character.FighterReason{character.FighterDuplicateMastery}},
		{"an item that is not a weapon", 1, nil, []string{"greatsword", "mace", "leather_armor"}, []string{"greatsword", "mace"}, []character.FighterReason{character.FighterNotSimpleOrMartialWeapon}},
		{"dropping a kind", 4, []string{"greatsword", "longbow", "mace"}, []string{"greatsword", "longbow", "rapier", "club"}, []string{"greatsword", "longbow", "rapier", "club"}, []character.FighterReason{character.FighterDropsExistingMastery}},
		{"count and drop and duplicate", 4, []string{"greatsword", "longbow", "mace"}, []string{"greatsword", "longbow", "longbow"}, []string{"greatsword", "longbow"}, []character.FighterReason{character.FighterWrongMasteryCount, character.FighterDuplicateMastery, character.FighterDropsExistingMastery}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ps := fgtProblems(t, fgtDecide(t, masteryFacts(t, c.level, c.current, c.weapons...), character.FighterChooseWeaponMasteries{Kinds: ids(c.kinds...)}))
			if got := fgtReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
		})
	}
	ps := fgtProblems(t, fgtDecide(t, masteryFacts(t, 1, nil, "greatsword", "mace"), character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "mace", "leather_armor")}))
	if ps[0].Item != "leather_armor" || ps[0].Feat != "" {
		t.Errorf("problem = %+v, want the item id", ps[0])
	}
}

// Requirement: doc/specs/class-fighter.md FR-046, FR-047, FR-121, FR-159, FR-163 - after a Long Rest of this character one chosen kind is swapped for
// another (the choice keeps its order and FighterWeaponMasterySwapped is returned); a Short Rest is noLongRestFinished, another character's
// event is eventNotForCharacter (before the rest kind), a kind not chosen is notCurrentlyChosen, one already chosen is alreadyChosen and an
// item that is not a weapon is notSimpleOrMartialWeapon.
// Path: per case (unhappy: state error, invalid input).
func TestFighterChoiceRulesSwapWeaponMastery(t *testing.T) {
	t.Parallel()
	current := []string{"greatsword", "longbow", "mace"}
	facts := masteryFacts(t, 1, current, "greatsword", "longbow", "mace", "rapier", "club")
	id := facts.Character.ID()
	long, short := fgtRest(t, id, character.LongRest), fgtRest(t, id, character.ShortRest)
	foreign := fgtRest(t, character.MustIdentifier("someone-else"), character.LongRest)
	foreignShort := fgtRest(t, character.MustIdentifier("someone-else"), character.ShortRest)
	lowerID := fgtRest(t, character.MustIdentifier("a-friend"), character.LongRest)
	a := fgtAccepted(t, fgtDecide(t, facts, character.FighterSwapWeaponMastery{Remove: "longbow", Add: "rapier", RestApplied: long}))
	swapped, ok := a.Events()[0].(character.FighterWeaponMasterySwapped)
	if got := must(character.WeaponMasteryAccessFor(a.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(); !slices.Equal(got, ids("greatsword", "rapier", "mace")) ||
		!ok || swapped.Removed() != "longbow" || swapped.Added() != "rapier" || swapped.CharacterID() != id {
		t.Errorf("choice %v, events %#v", got, a.Events())
	}
	for _, c := range []struct {
		name        string
		remove, add string
		rest        character.RestApplied
		want        []character.FighterReason
	}{
		{"a Short Rest", "longbow", "rapier", short, []character.FighterReason{character.FighterNoLongRestFinished}},
		{"another character's Long Rest", "longbow", "rapier", foreign, []character.FighterReason{character.FighterEventNotForCharacter}},
		{"another character's Short Rest", "longbow", "rapier", foreignShort, []character.FighterReason{character.FighterEventNotForCharacter}},
		{"a lower-sorting character's Long Rest", "longbow", "rapier", lowerID, []character.FighterReason{character.FighterEventNotForCharacter}},
		{"a kind not chosen", "club", "rapier", long, []character.FighterReason{character.FighterNotCurrentlyChosen}},
		{"a kind already chosen", "longbow", "mace", long, []character.FighterReason{character.FighterAlreadyChosen}},
		{"an item that is not a weapon", "longbow", "leather_armor", long, []character.FighterReason{character.FighterNotSimpleOrMartialWeapon}},
		{"a Short Rest and a kind not chosen", "club", "rapier", short, []character.FighterReason{character.FighterNoLongRestFinished}},
		{"another character and a kind not chosen", "club", "rapier", foreign, []character.FighterReason{character.FighterEventNotForCharacter, character.FighterNotCurrentlyChosen}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ps := fgtProblems(t, fgtDecide(t, facts, character.FighterSwapWeaponMastery{Remove: content.ItemID(c.remove), Add: content.ItemID(c.add), RestApplied: c.rest}))
			if got := fgtReasons(ps); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-153, FR-163 - Decide refuses a zero character or stats, a nil or foreign choice and a style choice
// whose facts hold no feat of the chosen id with an error matching srderr.ErrInvalidInput, and a Character holding another type of state under
// a Fighter key; it never panics.
// Path: per case (unhappy: invalid input).
func TestFighterChoiceRulesInvalidInput(t *testing.T) {
	t.Parallel()
	good := styleFacts(t, 1, character.InitialFighterState(), character.TakenFeats{}, fgtStyle("archery"))
	choice := character.FighterChooseFightingStyle{Choice: styleChoice("archery")}
	noStats, noChar, noFeat := good, good, good
	withMarker := noFeat
	withMarker.State = character.MustFighterState(0, 0, 0, "defense", "")
	withMarker.Feat = content.Feat{}
	noStats.Stats, noChar.Character, noFeat.Feat = character.FighterLevelStats{}, character.Character{}, content.Feat{}
	foreignMastery := masteryFacts(t, 1, nil, "greatsword", "longbow", "mace")
	foreignMastery.Character = must(character.WithFeatureState(foreignMastery.Character, character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("champion"))))
	foreignFighter := good
	foreignFighter.Character = must(character.WithFeatureState(fgtFighter(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("fighter"), must(character.NewChosenSubclass("champion"))))
	for _, c := range []struct {
		name   string
		facts  character.FighterChoiceFacts
		choice character.FighterOptionChoice
	}{
		{"zero stats", noStats, choice}, {"zero character", noChar, choice}, {"nil choice", good, nil}, {"no feat content", noFeat, choice},
		{"no feat content for a replacement", withMarker, character.FighterReplaceFightingStyle{Choice: styleChoice("archery")}},
		{"a foreign choice", good, fgtForeignChoice{}},
		{"a foreign mastery state", foreignMastery, character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}},
		{"a foreign Fighter state", foreignFighter, choice},
	} {
		v, err := character.FighterChoiceRules{}.Decide(c.facts, c.choice)
		wantInvalid(t, err, c.name)
		if v != nil {
			t.Errorf("%s: verdict = %v, want nil", c.name, v)
		}
	}
}

// fgtForeignChoice implements the unexported marker through embedding, which a type outside the package cannot do directly.
type fgtForeignChoice struct{ character.FighterOptionChoice }
