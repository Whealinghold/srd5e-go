package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

var (
	sStealth, sSurvival, sNature, sPerception = content.SkillStealth, content.SkillSurvival, content.SkillNature, content.SkillPerception
	lElvish, lDwarvish, lGiant                = content.LanguageElvish, content.LanguageDwarvish, content.LanguageGiant
)

func rgrProficient() character.WeaponProficiencies {
	return character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponSimple), character.CategoryProficiency(content.WeaponMartial))
}

// rgrSkills are the skill proficiencies of a Ranger: Stealth, Survival, Nature and Perception, with Expertise as given.
func rgrSkills(t *testing.T, expertise ...content.Skill) character.SkillProficiencies {
	t.Helper()
	return must(character.NewSkillProficiencies([]content.Skill{sStealth, sSurvival, sNature, sPerception}, expertise))
}

// rgrFacts of a choice: the Ranger of level n holding state and subclass, the character and the proficiencies.
func rgrChoiceFacts(t *testing.T, n int, state character.RangerState, subclass content.SubclassID, masteries character.WeaponMasteryChoice, expertise ...content.Skill) character.RangerChoiceFacts {
	t.Helper()
	return character.RangerChoiceFacts{Character: rgrWith(t, rgrRanger(t, n), state), Ranger: rgrFacts(t, n, state, subclass), Skills: rgrSkills(t, expertise...), Masteries: masteries}
}

func rgrDecide(t *testing.T, in character.RangerChoiceFacts, choice character.RangerChoice) character.RangerChoiceVerdict {
	t.Helper()
	return must(character.RangerChoiceRules{}.Decide(in, choice))
}

func rgrRefused(t *testing.T, label string, v character.RangerChoiceVerdict, want ...string) {
	t.Helper()
	r, ok := v.(character.RangerChoiceRefused)
	if !ok || !slices.Equal(rgrReasons(r.Reasons()), want) {
		t.Errorf("%s = %#v, want refused %v", label, v, want)
	}
}

func rgrAccepted(t *testing.T, label string, v character.RangerChoiceVerdict) character.RangerChoiceAccepted {
	t.Helper()
	a, ok := v.(character.RangerChoiceAccepted)
	if !ok {
		t.Fatalf("%s = %#v, want accepted", label, v)
	}
	return a
}

// Requirement: doc/specs/class-ranger.md FR-037, FR-040, FR-041, FR-108 (SRD 5.2.1, "Level 1: Weapon Mastery", pp. 58-59) - the initial Weapon Mastery choice is two kinds the Ranger is proficient with, made without a rest;
// a choice held already, one kind, a repeated kind and an unknown weapon are refused (already_chosen, invalid_choice); replacing needs a held pair (nothing_to_replace) and a Long Rest (wrong_moment for a Short Rest, a level gained or none),
// accepts either or both kinds and refuses a kind not held or not eligible with invalid_choice.
// Path: per case: happy / unhappy (invalid input, state error, boundary).
func TestRangerChoiceWeaponMastery(t *testing.T) {
	t.Parallel()
	weapons := fgtWeapons(t, "longbow", "shortsword", "scimitar", "longsword")
	pair := choiceOf(t, "longbow", "shortsword")
	none := character.WeaponMasteryChoice{}
	initial := func(kinds ...string) character.RangerChooseWeaponMasteries {
		return character.RangerChooseWeaponMasteries{Kinds: ids(kinds...), Weapons: weapons, Proficiencies: rgrProficient()}
	}
	swap := func(moment character.RangerReplacementMoment, swaps ...character.MasterySwap) character.RangerReplaceWeaponMasteries {
		return character.RangerReplaceWeaponMasteries{Swaps: swaps, Moment: moment, Weapons: weapons, Proficiencies: rgrProficient()}
	}
	got := rgrAccepted(t, "initial pair", rgrDecide(t, rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none), initial("longbow", "shortsword")))
	m, held, _ := character.ReadFeatureState(got.Character().FeatureStates(), character.WeaponMasteryChoiceKey())
	made, isMade := got.Events()[0].(character.RangerChoiceMade)
	if !held || !m.Choice().Equal(pair) || !isMade || made.Choice() != "weapon_mastery" || made.CharacterID() != got.Character().ID() {
		t.Errorf("initial pair stored %v (%v), events %#v", m.Choice().Kinds(), held, got.Events())
	}
	ranger := rgrChoiceFacts(t, 1, character.InitialRangerState(), "", pair)
	for _, c := range []struct {
		label  string
		in     character.RangerChoiceFacts
		choice character.RangerChoice
		want   []string
	}{
		{"a pair already held", ranger, initial("scimitar", "longsword"), []string{"already_chosen"}},
		{"one kind", rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none), initial("longbow"), []string{"invalid_choice"}},
		{"a repeated kind", rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none), initial("longbow", "longbow"), []string{"invalid_choice"}},
		{"an unknown weapon", rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none), initial("longbow", "club"), []string{"invalid_choice"}},
		{"not proficient", func() character.RangerChoiceFacts {
			return rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none)
		}(), character.RangerChooseWeaponMasteries{Kinds: ids("longbow", "shortsword"), Weapons: weapons}, []string{"invalid_choice"}},
		{"replace with none held", rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none), swap(character.RangerFinishedLongRest), []string{"nothing_to_replace"}},
		{"replace after a Short Rest", ranger, swap(character.RangerFinishedShortRest), []string{"wrong_moment"}},
		{"replace on gaining a level", ranger, swap(character.RangerGainedLevel), []string{"wrong_moment"}},
		{"replace with no moment", ranger, swap(0), []string{"wrong_moment"}},
		{"replace a kind not held", ranger, swap(character.RangerFinishedLongRest, character.MasterySwap{Remove: "scimitar", Add: "longsword"}), []string{"invalid_choice"}},
		{"nothing held and wrong moment", rgrChoiceFacts(t, 1, character.InitialRangerState(), "", none), swap(character.RangerFinishedShortRest), []string{"wrong_moment", "nothing_to_replace"}},
	} {
		rgrRefused(t, c.label, rgrDecide(t, c.in, c.choice), c.want...)
	}
	for _, c := range []struct {
		label string
		swaps []character.MasterySwap
		want  []string
	}{
		{"either kind", []character.MasterySwap{{Remove: "longbow", Add: "scimitar"}}, []string{"scimitar", "shortsword"}},
		{"both kinds", []character.MasterySwap{{Remove: "longbow", Add: "scimitar"}, {Remove: "shortsword", Add: "longsword"}}, []string{"scimitar", "longsword"}},
	} {
		a := rgrAccepted(t, c.label, rgrDecide(t, ranger, swap(character.RangerFinishedLongRest, c.swaps...)))
		m, _, _ := character.ReadFeatureState(a.Character().FeatureStates(), character.WeaponMasteryChoiceKey())
		replaced, isReplaced := a.Events()[0].(character.RangerChoiceReplaced)
		if !slices.Equal(rgrIDStrings(m.Choice().Kinds()), c.want) || !isReplaced || replaced.Choice() != "weapon_mastery" {
			t.Errorf("%s = %v, events %#v", c.label, m.Choice().Kinds(), a.Events())
		}
	}
}

func rgrIDStrings(ids []content.ItemID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// Requirement: doc/specs/class-ranger.md FR-042 to FR-046 (SRD 5.2.1, "Level 2: Deft Explorer", p. 59) - Deft Explorer takes Expertise in one proficient skill without Expertise and exactly two distinct languages from level 2; below level 2, a second choice, a skill without
// proficiency, a skill with Expertise and one, three or repeated languages are refused with feature_not_available, already_chosen, not_proficient, already_has_expertise and invalid_choice.
// Path: per case: happy / unhappy (boundary, invalid input, state error, rule interaction).
func TestRangerChoiceDeftExplorer(t *testing.T) {
	t.Parallel()
	start := character.InitialRangerState()
	deft := func(s content.Skill, l ...content.Language) character.RangerChooseDeftExplorer {
		return character.RangerChooseDeftExplorer{Skill: s, Languages: l}
	}
	a := rgrAccepted(t, "happy", rgrDecide(t, rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), deft(sStealth, lElvish, lDwarvish)))
	got, ok := a.State().DeftExplorer()
	if !ok || got != (character.DeftExplorerChoice{Skill: sStealth, Languages: [2]content.Language{lElvish, lDwarvish}}) {
		t.Errorf("stored %+v, %v", got, ok)
	}
	if e := a.Events()[0].(character.RangerChoiceMade); e.Choice() != "deft_explorer" || !a.State().Equal(rgrStateOf(t, a.Character())) {
		t.Errorf("event %v", e)
	}
	chosen := rgrState(t, character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: sNature, Languages: [2]content.Language{lElvish, lGiant}}})
	for _, c := range []struct {
		label  string
		in     character.RangerChoiceFacts
		choice character.RangerChoiceFacts
		skill  content.Skill
		langs  []content.Language
		want   []string
	}{
		{"level 1", rgrChoiceFacts(t, 1, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish, lDwarvish}, []string{"feature_not_available"}},
		{"chosen already", rgrChoiceFacts(t, 2, chosen, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish, lDwarvish}, []string{"already_chosen"}},
		{"not proficient", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, content.SkillArcana, []content.Language{lElvish, lDwarvish}, []string{"not_proficient"}},
		{"has Expertise", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}, sStealth), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish, lDwarvish}, []string{"already_has_expertise"}},
		{"an invalid skill", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, 0, []content.Language{lElvish, lDwarvish}, []string{"invalid_choice"}},
		{"one language", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish}, []string{"invalid_choice"}},
		{"three languages", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish, lDwarvish, lGiant}, []string{"invalid_choice"}},
		{"a repeated language", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish, lElvish}, []string{"invalid_choice"}},
		{"an invalid language", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{lElvish, 0}, []string{"invalid_choice"}},
		{"an invalid first language", rgrChoiceFacts(t, 2, start, "", character.WeaponMasteryChoice{}), character.RangerChoiceFacts{}, sStealth, []content.Language{0, lElvish}, []string{"invalid_choice"}},
	} {
		rgrRefused(t, c.label, rgrDecide(t, c.in, deft(c.skill, c.langs...)), c.want...)
	}
}

// Requirement: doc/specs/class-ranger.md FR-046 to FR-048 (SRD 5.2.1, "Level 9: Expertise", p. 59) - Expertise takes exactly two distinct proficient skills without Expertise, counting the Deft Explorer skill, from level 9; level 8, a second choice, one or three skills, a repeated or
// invalid skill, a skill without proficiency and a skill with Expertise (including the Deft Explorer skill) are refused.
// Path: per case: happy / unhappy (boundary, invalid input, state error, rule interaction).
func TestRangerChoiceExpertise(t *testing.T) {
	t.Parallel()
	start := character.InitialRangerState()
	none := character.WeaponMasteryChoice{}
	pick := func(s ...content.Skill) character.RangerChooseExpertise {
		return character.RangerChooseExpertise{Skills: s}
	}
	a := rgrAccepted(t, "happy", rgrDecide(t, rgrChoiceFacts(t, 9, start, "", none), pick(sSurvival, sNature)))
	if skills, ok := a.State().Expertise(); !ok || !slices.Equal(skills, []content.Skill{sSurvival, sNature}) || a.Events()[0].(character.RangerChoiceMade).Choice() != "expertise" {
		t.Errorf("stored %v, %v", skills, ok)
	}
	deft := rgrState(t, character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: sStealth, Languages: [2]content.Language{lElvish, lDwarvish}}})
	both := rgrState(t, character.RangerStateParts{Expertise: &[2]content.Skill{sSurvival, sNature}})
	for _, c := range []struct {
		label string
		in    character.RangerChoiceFacts
		skill []content.Skill
		want  []string
	}{
		{"level 8", rgrChoiceFacts(t, 8, start, "", none), []content.Skill{sSurvival, sNature}, []string{"feature_not_available"}},
		{"chosen already", rgrChoiceFacts(t, 9, both, "", none), []content.Skill{sSurvival, sNature}, []string{"already_chosen"}},
		{"one skill", rgrChoiceFacts(t, 9, start, "", none), []content.Skill{sSurvival}, []string{"invalid_choice"}},
		{"three skills", rgrChoiceFacts(t, 9, start, "", none), []content.Skill{sSurvival, sNature, sStealth}, []string{"invalid_choice"}},
		{"a repeated skill", rgrChoiceFacts(t, 9, start, "", none), []content.Skill{sNature, sNature}, []string{"invalid_choice"}},
		{"an invalid first skill", rgrChoiceFacts(t, 9, start, "", none), []content.Skill{0, sNature}, []string{"invalid_choice"}},
		{"an invalid second skill", rgrChoiceFacts(t, 9, start, "", none), []content.Skill{sNature, 0}, []string{"invalid_choice"}},
		{"not proficient", rgrChoiceFacts(t, 9, start, "", none), []content.Skill{sNature, content.SkillArcana}, []string{"not_proficient"}},
		{"has Expertise", rgrChoiceFacts(t, 9, start, "", none, sNature), []content.Skill{sNature, sSurvival}, []string{"already_has_expertise"}},
		{"the Deft Explorer skill", rgrChoiceFacts(t, 9, deft, "", none), []content.Skill{sStealth, sSurvival}, []string{"already_has_expertise"}},
	} {
		rgrRefused(t, c.label, rgrDecide(t, c.in, pick(c.skill...)), c.want...)
	}
}

// Requirement: doc/specs/class-ranger.md FR-049, FR-054, FR-055 (SRD 5.2.1, "Level 2: Fighting Style", p. 59; "Fighting Style Feats", pp. 87-88) - a Fighting Style feat is taken through the feat rules from level 2 and recorded as RangerFightingStyleFeat with the feat events first;
// below level 2, a second choice and a feat of another category are refused (feature_not_available, already_chosen, invalid_choice) with the Character unchanged; the choice is a feat or Druidic Warrior, never both.
// Path: per case: happy / unhappy (boundary, state error, rule interaction).
func TestRangerChoiceFightingStyleFeat(t *testing.T) {
	t.Parallel()
	start := character.InitialRangerState()
	none := character.WeaponMasteryChoice{}
	a := rgrAccepted(t, "archery", rgrDecide(t, rgrChoiceFacts(t, 2, start, "", none), character.RangerChooseFightingStyleFeat{Feat: fgtStyle("archery")}))
	style, _ := a.State().FightingStyle()
	if style != (character.RangerFightingStyleFeat{Feat: "archery"}) || a.Feats().Len() != 1 || len(a.Events()) != 2 || a.Events()[1].(character.RangerChoiceMade).Choice() != "fighting_style" {
		t.Errorf("style %v, feats %v, events %#v", style, a.Feats().All(), a.Events())
	}
	for _, c := range []struct {
		label string
		in    character.RangerChoiceFacts
		feat  content.Feat
		want  []string
	}{
		{"level 1", rgrChoiceFacts(t, 1, start, "", none), fgtStyle("archery"), []string{"feature_not_available"}},
		{"chosen already", rgrChoiceFacts(t, 2, rgrState(t, character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "starry_wisp"}}}), "", none), fgtStyle("archery"), []string{"already_chosen"}},
		{"another category", rgrChoiceFacts(t, 2, start, "", none), alertFeat(), []string{"invalid_choice"}},
		{"level 1 and chosen", rgrChoiceFacts(t, 1, rgrState(t, character.RangerStateParts{FightingStyle: character.RangerFightingStyleFeat{Feat: "defense"}}), "", none), fgtStyle("archery"), []string{"feature_not_available", "already_chosen"}},
	} {
		rgrRefused(t, c.label, rgrDecide(t, c.in, character.RangerChooseFightingStyleFeat{Feat: c.feat}), c.want...)
	}
	_, err := character.RangerChoiceRules{}.Decide(rgrChoiceFacts(t, 2, start, "", none), character.RangerChooseFightingStyleFeat{})
	wantInvalid(t, err, "a zero feat")
}

// Requirement: doc/specs/class-ranger.md FR-051 to FR-055 (SRD 5.2.1, "Level 2: Fighting Style" (Druidic Warrior), p. 59) - Druidic Warrior takes exactly two distinct Druid cantrips; one or three cantrips, a repeated one, a spell that is not a cantrip and a cantrip not on the Druid list are
// refused with invalid_choice; a second choice and level 1 are refused; one cantrip is replaced only on gaining a level, with another Druid cantrip not already held, and a replacement with no Druidic Warrior, at another moment or with a bad cantrip is refused.
// Path: per case: happy / unhappy (invalid input, state error, rule interaction).
func TestRangerChoiceDruidicWarrior(t *testing.T) {
	t.Parallel()
	start := character.InitialRangerState()
	none := character.WeaponMasteryChoice{}
	guidance, wisp, mending := rgrCantrip(t, "guidance", "cleric", "druid"), rgrCantrip(t, "starry_wisp", "druid", "bard"), rgrCantrip(t, "druidcraft", "druid")
	sacred, fireball := rgrCantrip(t, "sacred_flame", "cleric"), clcSpell(t, "thorn", 1, []string{"druid"}, "")
	choose := func(s ...content.Spell) character.RangerChooseDruidicWarrior {
		return character.RangerChooseDruidicWarrior{Cantrips: s}
	}
	a := rgrAccepted(t, "happy", rgrDecide(t, rgrChoiceFacts(t, 2, start, "", none), choose(guidance, wisp)))
	style, _ := a.State().FightingStyle()
	if style != (character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "starry_wisp"}}) || len(a.State().DruidicWarriorCantrips()) != 2 || a.Feats().Len() != 0 {
		t.Errorf("style %v", style)
	}
	for _, c := range []struct {
		label string
		in    character.RangerChoiceFacts
		picks []content.Spell
		want  []string
	}{
		{"level 1", rgrChoiceFacts(t, 1, start, "", none), []content.Spell{guidance, wisp}, []string{"feature_not_available"}},
		{"one cantrip", rgrChoiceFacts(t, 2, start, "", none), []content.Spell{guidance}, []string{"invalid_choice"}},
		{"three cantrips", rgrChoiceFacts(t, 2, start, "", none), []content.Spell{guidance, wisp, mending}, []string{"invalid_choice"}},
		{"a repeated cantrip", rgrChoiceFacts(t, 2, start, "", none), []content.Spell{guidance, guidance}, []string{"invalid_choice"}},
		{"not a cantrip", rgrChoiceFacts(t, 2, start, "", none), []content.Spell{guidance, fireball}, []string{"invalid_choice"}},
		{"not a Druid cantrip", rgrChoiceFacts(t, 2, start, "", none), []content.Spell{guidance, sacred}, []string{"invalid_choice"}},
		{"not a Druid first cantrip", rgrChoiceFacts(t, 2, start, "", none), []content.Spell{sacred, guidance}, []string{"invalid_choice"}},
		{"chosen already", rgrChoiceFacts(t, 2, a.State(), "", none), []content.Spell{mending, wisp}, []string{"already_chosen"}},
	} {
		rgrRefused(t, c.label, rgrDecide(t, c.in, choose(c.picks...)), c.want...)
	}
	held := a.State()
	swap := func(old content.SpellID, repl content.Spell, moment character.RangerReplacementMoment) character.RangerReplaceDruidicCantrip {
		return character.RangerReplaceDruidicCantrip{Old: old, New: repl, Moment: moment}
	}
	r := rgrAccepted(t, "replace", rgrDecide(t, rgrChoiceFacts(t, 3, held, "", none), swap("guidance", mending, character.RangerGainedLevel)))
	style, _ = r.State().FightingStyle()
	if style != (character.DruidicWarrior{Cantrips: [2]content.SpellID{"druidcraft", "starry_wisp"}}) || r.Events()[0].(character.RangerChoiceReplaced).Choice() != "fighting_style" {
		t.Errorf("replaced style %v, events %#v", style, r.Events())
	}
	r2 := rgrAccepted(t, "replace the second", rgrDecide(t, rgrChoiceFacts(t, 3, held, "", none), swap("starry_wisp", mending, character.RangerGainedLevel)))
	if style, _ = r2.State().FightingStyle(); style != (character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "druidcraft"}}) {
		t.Errorf("replaced the second: %v", style)
	}
	feat := rgrState(t, character.RangerStateParts{FightingStyle: character.RangerFightingStyleFeat{Feat: "archery"}})
	for _, c := range []struct {
		label string
		in    character.RangerChoiceFacts
		repl  character.RangerReplaceDruidicCantrip
		want  []string
	}{
		{"no Druidic Warrior", rgrChoiceFacts(t, 3, start, "", none), swap("guidance", mending, character.RangerGainedLevel), []string{"nothing_to_replace"}},
		{"a feat style", rgrChoiceFacts(t, 3, feat, "", none), swap("guidance", mending, character.RangerGainedLevel), []string{"nothing_to_replace"}},
		{"after a Long Rest", rgrChoiceFacts(t, 3, held, "", none), swap("guidance", mending, character.RangerFinishedLongRest), []string{"wrong_moment"}},
		{"no moment", rgrChoiceFacts(t, 3, held, "", none), swap("guidance", mending, 0), []string{"wrong_moment"}},
		{"a cantrip not held", rgrChoiceFacts(t, 3, held, "", none), swap("sacred_flame", mending, character.RangerGainedLevel), []string{"invalid_choice"}},
		{"not a Druid cantrip", rgrChoiceFacts(t, 3, held, "", none), swap("guidance", sacred, character.RangerGainedLevel), []string{"invalid_choice"}},
		{"not a cantrip", rgrChoiceFacts(t, 3, held, "", none), swap("guidance", fireball, character.RangerGainedLevel), []string{"invalid_choice"}},
		{"the other held cantrip", rgrChoiceFacts(t, 3, held, "", none), swap("guidance", wisp, character.RangerGainedLevel), []string{"invalid_choice"}},
		{"the same cantrip", rgrChoiceFacts(t, 3, held, "", none), swap("guidance", guidance, character.RangerGainedLevel), []string{"invalid_choice"}},
	} {
		rgrRefused(t, c.label, rgrDecide(t, c.in, c.repl), c.want...)
	}
}

// Requirement: doc/specs/class-ranger.md FR-086, FR-087, FR-091, FR-108 (SRD 5.2.1, "Level 3: Hunter’s Prey", "Level 7: Defensive Tactics", p. 61) - Hunter’s Prey (from level 3) and Defensive Tactics (from level 7) take one of their two option ids without a rest; another id is refused
// with unknown_option, a second initial choice with already_chosen, a character without the Hunter with not_hunter and a level below with feature_not_available; the option is replaced with the other one after a Short or Long Rest, and replacing with the one held
// (same_option), at another moment (wrong_moment) or with nothing held (nothing_to_replace) is refused.
// Path: per case: happy / unhappy (boundary, invalid input, state error, rule interaction).
func TestRangerChoiceHunterOptions(t *testing.T) {
	t.Parallel()
	none := character.WeaponMasteryChoice{}
	type choiceFn func(option string, replace bool, moment character.RangerReplacementMoment) character.RangerChoice
	prey := func(option string, replace bool, moment character.RangerReplacementMoment) character.RangerChoice {
		if replace {
			return character.RangerReplaceHuntersPrey{Option: option, Moment: moment}
		}
		return character.RangerChooseHuntersPrey{Option: option}
	}
	tactics := func(option string, replace bool, moment character.RangerReplacementMoment) character.RangerChoice {
		if replace {
			return character.RangerReplaceDefensiveTactics{Option: option, Moment: moment}
		}
		return character.RangerChooseDefensiveTactics{Option: option}
	}
	for _, f := range []struct {
		name         string
		fn           choiceFn
		level        int
		first, other string
		held         func(string) character.RangerStateParts
		feature      string
	}{
		{"Hunter’s Prey", prey, 3, "colossus_slayer", "horde_breaker", func(o string) character.RangerStateParts { return character.RangerStateParts{HuntersPrey: o} }, "hunters_prey"},
		{"Defensive Tactics", tactics, 7, "escape_the_horde", "multiattack_defense", func(o string) character.RangerStateParts { return character.RangerStateParts{DefensiveTactics: o} }, "defensive_tactics"},
	} {
		start := character.InitialRangerState()
		held := rgrState(t, f.held(f.first))
		a := rgrAccepted(t, f.name+" initial", rgrDecide(t, rgrChoiceFacts(t, f.level, start, hunter, none), f.fn(f.first, false, 0)))
		a2 := rgrAccepted(t, f.name+" replacement", rgrDecide(t, rgrChoiceFacts(t, f.level, held, hunter, none), f.fn(f.other, true, character.RangerFinishedShortRest)))
		a3 := rgrAccepted(t, f.name+" replacement", rgrDecide(t, rgrChoiceFacts(t, f.level, held, hunter, none), f.fn(f.other, true, character.RangerFinishedLongRest)))
		stored := func(r character.RangerChoiceAccepted) string {
			if f.name == "Hunter’s Prey" {
				s, _ := r.State().HuntersPrey()
				return s
			}
			s, _ := r.State().DefensiveTactics()
			return s
		}
		if stored(a) != f.first || stored(a2) != f.other || stored(a3) != f.other || a.Events()[0].(character.RangerChoiceMade).Choice() != f.feature || a2.Events()[0].(character.RangerChoiceReplaced).Choice() != f.feature {
			t.Errorf("%s stored %q, %q, %q", f.name, stored(a), stored(a2), stored(a3))
		}
		for _, c := range []struct {
			label  string
			in     character.RangerChoiceFacts
			choice character.RangerChoice
			want   []string
		}{
			{"unknown option", rgrChoiceFacts(t, f.level, start, hunter, none), f.fn("colossus", false, 0), []string{"unknown_option"}},
			{"unknown replacement", rgrChoiceFacts(t, f.level, held, hunter, none), f.fn("colossus", true, character.RangerFinishedLongRest), []string{"unknown_option"}},
			{"second initial choice", rgrChoiceFacts(t, f.level, held, hunter, none), f.fn(f.other, false, 0), []string{"already_chosen"}},
			{"no Hunter", rgrChoiceFacts(t, f.level, start, "", none), f.fn(f.first, false, 0), []string{"not_hunter"}},
			{"below the level", rgrChoiceFacts(t, f.level-1, start, hunter, none), f.fn(f.first, false, 0), []string{"feature_not_available"}},
			{"replace with the one held", rgrChoiceFacts(t, f.level, held, hunter, none), f.fn(f.first, true, character.RangerFinishedLongRest), []string{"same_option"}},
			{"replace on gaining a level", rgrChoiceFacts(t, f.level, held, hunter, none), f.fn(f.other, true, character.RangerGainedLevel), []string{"wrong_moment"}},
			{"replace with no moment", rgrChoiceFacts(t, f.level, held, hunter, none), f.fn(f.other, true, 0), []string{"wrong_moment"}},
			{"replace with none held", rgrChoiceFacts(t, f.level, start, hunter, none), f.fn(f.other, true, character.RangerFinishedLongRest), []string{"nothing_to_replace"}},
		} {
			rgrRefused(t, f.name+" "+c.label, rgrDecide(t, c.in, c.choice), c.want...)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-062, FR-064, T6 (SRD 5.2.1, "Level 4: Ability Score Improvement", "Level 19: Epic Boon", p. 59) - a feat grant is counted against the sums of T6: a General feat uses an Ability Score Improvement grant (levels 4, 8, 12, 16), an Epic Boon feat the level 19 grant;
// with none left the choice is refused with no_grant_remaining, an Epic Boon below 19 with feature_not_available, a feat of another category or one the character does not qualify for with invalid_choice; the accepted choice returns the FeatGrant for TakeFeat and the unspent grants stay.
// Path: per case (T6): happy / unhappy (boundary: one below, maximum level; state error).
func TestRangerChoiceFeatGrant(t *testing.T) {
	t.Parallel()
	none := character.WeaponMasteryChoice{}
	general, boon := featOf("alert_general", content.FeatGeneral, false), featOf("boon_of_fate", content.FeatEpicBoon, false)
	state := func(g, e int) character.RangerState {
		return rgrState(t, character.RangerStateParts{GeneralFeats: g, EpicBoons: e})
	}
	for _, c := range []struct {
		label       string
		level       int
		taken       character.RangerState
		feat        content.Feat
		want        []string
		generalTook int
		boonTook    int
	}{
		{"FG-01 level 3, no grant yet", 3, state(0, 0), general, []string{"no_grant_remaining"}, 0, 0},
		{"FG-02 level 4", 4, state(0, 0), general, nil, 1, 0},
		{"level 4, grant used", 4, state(1, 0), general, []string{"no_grant_remaining"}, 0, 0},
		{"FG-04 level 8, two", 8, state(1, 0), general, nil, 2, 0},
		{"level 8, both used", 8, state(2, 0), general, []string{"no_grant_remaining"}, 0, 0},
		{"FG-06 level 16, four", 16, state(3, 0), general, nil, 4, 0},
		{"FG-07 level 18, Epic Boon not yet", 18, state(4, 0), boon, []string{"feature_not_available"}, 0, 0},
		{"FG-08 level 19, Epic Boon", 19, state(4, 0), boon, nil, 4, 1},
		{"FG-09 level 20, Epic Boon used", 20, state(4, 1), boon, []string{"no_grant_remaining"}, 0, 0},
		{"another category", 4, state(0, 0), alertFeat(), []string{"invalid_choice"}, 0, 0},
		{"not qualified", 4, state(0, 0), grapplerFeat(), []string{"invalid_choice"}, 0, 0},
	} {
		v := rgrDecide(t, rgrChoiceFacts(t, c.level, c.taken, "", none), character.RangerChooseFeat{Feat: c.feat})
		if c.want != nil {
			rgrRefused(t, c.label, v, c.want...)
			continue
		}
		a := rgrAccepted(t, c.label, v)
		g, e := a.State().FeatsTaken()
		if g != c.generalTook || e != c.boonTook || a.Grant().IsZero() || a.Feats().Len() != 0 || a.Events()[0].(character.RangerChoiceMade).Choice() != "feat_grant" {
			t.Errorf("%s: took %d, %d; grant %v", c.label, g, e, a.Grant())
		}
	}
	_, err := character.RangerChoiceRules{}.Decide(rgrChoiceFacts(t, 4, state(0, 0), "", none), character.RangerChooseFeat{})
	wantInvalid(t, err, "a zero feat")
	qualified := rgrChoiceFacts(t, 4, state(0, 0), "", none)
	qualified.Character = restored(t, 355000, 200, flatScores(t, 15), cl("ranger", 4))
	rgrAccepted(t, "qualified by scores", rgrDecide(t, qualified, character.RangerChooseFeat{Feat: grapplerFeat()}))
}

// Requirement: doc/specs/class-ranger.md FR-099, FR-111 - Decide refuses a zero Character, progression or level and a nil or unknown choice as invalid input, and leaves its inputs unchanged.
// Path: per case (unhappy: invalid input, immutability).
func TestRangerChoiceRulesInvalidInput(t *testing.T) {
	t.Parallel()
	good := rgrChoiceFacts(t, 2, character.InitialRangerState(), "", character.WeaponMasteryChoice{})
	zeroLevel := good
	zeroLevel.Ranger.Level = character.RangerLevel{}
	zeroProgression := good
	zeroProgression.Ranger.Progression = character.RangerProgression{}
	zeroCharacter := good
	zeroCharacter.Character = character.Character{}
	choice := character.RangerChooseDeftExplorer{Skill: sStealth, Languages: []content.Language{lElvish, lDwarvish}}
	for label, in := range map[string]character.RangerChoiceFacts{"zero level": zeroLevel, "zero progression": zeroProgression, "zero Character": zeroCharacter} {
		_, err := character.RangerChoiceRules{}.Decide(in, choice)
		wantInvalid(t, err, label)
	}
	_, err := character.RangerChoiceRules{}.Decide(good, nil)
	wantInvalid(t, err, "nil choice")
	before := good.Character
	rgrDecide(t, good, choice)
	if !good.Character.Equal(before) {
		t.Error("Decide changed the Character")
	}
}

// Requirement: doc/specs/class-ranger.md FR-045, FR-047, FR-051, FR-053, FR-064 - the order of two languages or two cantrips does not matter, Expertise with a Deft Explorer skill of a higher value than the picks is accepted, a moment outside the three is a wrong moment, and a feat grant count above
// the grants (more taken than granted) leaves none remaining.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestRangerChoiceOrderAndOverflow(t *testing.T) {
	t.Parallel()
	none := character.WeaponMasteryChoice{}
	start := character.InitialRangerState()
	rgrAccepted(t, "ascending languages", rgrDecide(t, rgrChoiceFacts(t, 2, start, "", none), character.RangerChooseDeftExplorer{Skill: sStealth, Languages: []content.Language{lDwarvish, lElvish}}))
	deft := rgrState(t, character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: sStealth, Languages: [2]content.Language{lElvish, lDwarvish}}})
	rgrAccepted(t, "lower skills beside the Deft Explorer skill", rgrDecide(t, rgrChoiceFacts(t, 9, deft, "", none), character.RangerChooseExpertise{Skills: []content.Skill{sNature, sSurvival}}))
	guidance, wisp := rgrCantrip(t, "guidance", "druid"), rgrCantrip(t, "starry_wisp", "druid")
	rgrAccepted(t, "descending cantrips", rgrDecide(t, rgrChoiceFacts(t, 2, start, "", none), character.RangerChooseDruidicWarrior{Cantrips: []content.Spell{wisp, guidance}}))
	held := rgrState(t, character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "starry_wisp"}}, HuntersPrey: "colossus_slayer"})
	rgrRefused(t, "an undefined cantrip moment", rgrDecide(t, rgrChoiceFacts(t, 3, held, hunter, none),
		character.RangerReplaceDruidicCantrip{Old: "guidance", New: rgrCantrip(t, "druidcraft", "druid"), Moment: 9}), "wrong_moment")
	rgrRefused(t, "an undefined option moment", rgrDecide(t, rgrChoiceFacts(t, 3, held, hunter, none), character.RangerReplaceHuntersPrey{Option: "horde_breaker", Moment: 9}), "wrong_moment")
	general, boon := featOf("alert_general", content.FeatGeneral, false), featOf("boon_of_fate", content.FeatEpicBoon, false)
	rgrRefused(t, "more general feats than grants", rgrDecide(t, rgrChoiceFacts(t, 4, rgrState(t, character.RangerStateParts{GeneralFeats: 3}), "", none), character.RangerChooseFeat{Feat: general}), "no_grant_remaining")
	rgrRefused(t, "more Epic Boons than grants", rgrDecide(t, rgrChoiceFacts(t, 19, rgrState(t, character.RangerStateParts{EpicBoons: 3}), "", none), character.RangerChooseFeat{Feat: boon}), "no_grant_remaining")
}
