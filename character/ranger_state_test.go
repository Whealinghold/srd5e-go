package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-ranger.md FR-073, FR-074, FR-080, T4 (SRD 5.2.1, "Level 10: Tireless", p. 59; "Level 14: Nature’s Veil", p. 59) - the uses of Tireless and Nature’s Veil are the greater of 1 and the
// Wisdom modifier: rows TL-01 to TL-08 of T4.
// Path: per case (T4): happy / unhappy (boundary: negative and zero modifier, lowest and highest score).
func TestTirelessUses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label    string
		modifier int
		want     int
	}{{"TL-01", 3, 3}, {"TL-02", 5, 5}, {"TL-03", 0, 1}, {"TL-04", -1, 1}, {"TL-05", -5, 1}, {"TL-06", 1, 1}, {"TL-07", 2, 2}, {"TL-08", 10, 10}} {
		if got := character.TirelessUses(modifierOf(t, c.modifier)); got != c.want {
			t.Errorf("%s: TirelessUses(%+d) = %d, want %d", c.label, c.modifier, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-035, FR-074, FR-110 - remaining uses are the maximum minus the expended count, never below zero: a rise of the Favored Enemy maximum at levels 5, 9, 13 and 17 leaves the expended count
// unchanged so the remaining rises by the increase, and a Wisdom decrease below the expended count leaves zero.
// Path: per case: unhappy (boundary).
func TestRangerStateRemaining(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	s := rgrState(t, character.RangerStateParts{FavoredEnemySpent: 2, TirelessSpent: 3, NaturesVeilSpent: 1})
	for _, c := range []struct{ level, want int }{{4, 0}, {5, 1}, {9, 2}, {13, 3}, {17, 4}} {
		if got := s.FavoredEnemyRemaining(p, character.MustRangerLevel(c.level)); got != c.want {
			t.Errorf("level %d remaining = %d, want %d", c.level, got, c.want)
		}
	}
	for _, c := range []struct{ wisdom, tireless, veil int }{{3, 0, 2}, {5, 2, 4}, {2, 0, 1}, {0, 0, 0}, {-3, 0, 0}} {
		if got := s.TirelessRemaining(modifierOf(t, c.wisdom)); got != c.tireless {
			t.Errorf("Wisdom %+d Tireless remaining = %d, want %d", c.wisdom, got, c.tireless)
		}
		if got := s.NaturesVeilRemaining(modifierOf(t, c.wisdom)); got != c.veil {
			t.Errorf("Wisdom %+d Nature’s Veil remaining = %d, want %d", c.wisdom, got, c.veil)
		}
	}
	if s.FavoredEnemySpent() != 2 || s.TirelessSpent() != 3 || s.NaturesVeilSpent() != 1 {
		t.Errorf("spent = %d, %d, %d", s.FavoredEnemySpent(), s.TirelessSpent(), s.NaturesVeilSpent())
	}
}

// Requirement: doc/specs/class-ranger.md FR-110 - RestoreRangerState round-trips every field through the accessors, InitialRangerState holds nothing and nothing spent, Equal compares fields and refuses another type, and the Expertise
// pair is returned as a new slice.
// Path: happy / unhappy (empty and minimal, mutation of the result).
func TestRestoreRangerStateRoundTrips(t *testing.T) {
	t.Parallel()
	deft := character.DeftExplorerChoice{Skill: content.SkillStealth, Languages: [2]content.Language{content.LanguageElvish, content.LanguageDwarvish}}
	expertise := [2]content.Skill{content.SkillSurvival, content.SkillNature}
	parts := character.RangerStateParts{FavoredEnemySpent: 1, TirelessSpent: 2, NaturesVeilSpent: 3, DeftExplorer: &deft, Expertise: &expertise,
		FightingStyle: character.RangerFightingStyleFeat{Feat: "archery"}, HuntersPrey: "horde_breaker", DefensiveTactics: "escape_the_horde", GeneralFeats: 2, EpicBoons: 1}
	s := rgrState(t, parts)
	got, hasDeft := s.DeftExplorer()
	skills, hasExpertise := s.Expertise()
	style, hasStyle := s.FightingStyle()
	prey, hasPrey := s.HuntersPrey()
	tactics, hasTactics := s.DefensiveTactics()
	general, epic := s.FeatsTaken()
	if !hasDeft || got != deft || !hasExpertise || !slices.Equal(skills, expertise[:]) || !hasStyle || style != (character.RangerFightingStyleFeat{Feat: "archery"}) ||
		!hasPrey || prey != "horde_breaker" || !hasTactics || tactics != "escape_the_horde" || general != 2 || epic != 1 {
		t.Errorf("round trip = %+v", s)
	}
	skills[0] = content.SkillArcana
	if again, _ := s.Expertise(); again[0] != content.SkillSurvival {
		t.Error("Expertise returned shared storage")
	}
	initial := character.InitialRangerState()
	_, d := initial.DeftExplorer()
	e, ex := initial.Expertise()
	_, st := initial.FightingStyle()
	_, pr := initial.HuntersPrey()
	_, ta := initial.DefensiveTactics()
	if d || ex || len(e) != 0 || st || pr || ta || !initial.Equal(rgrState(t, character.RangerStateParts{})) || initial.Equal(s) || initial.Equal(character.PaladinState{}) {
		t.Errorf("initial state = %+v", initial)
	}
}

// Requirement: doc/specs/class-ranger.md FR-045, FR-048, FR-054, FR-110 - RestoreRangerState refuses a negative count, a Deft Explorer choice with an invalid skill or language or twice the same language, Expertise of an invalid skill or twice the same
// skill, and a Fighting Style that is nil-typed, has an invalid feat id or invalid or equal cantrips, each as invalid input.
// Path: per case (unhappy: invalid input).
func TestRestoreRangerStateRejects(t *testing.T) {
	t.Parallel()
	goodDeft := character.DeftExplorerChoice{Skill: content.SkillStealth, Languages: [2]content.Language{content.LanguageElvish, content.LanguageDwarvish}}
	for _, c := range []struct {
		name  string
		parts character.RangerStateParts
	}{
		{"negative Favored Enemy", character.RangerStateParts{FavoredEnemySpent: -1}},
		{"negative Tireless", character.RangerStateParts{TirelessSpent: -1}},
		{"negative Nature’s Veil", character.RangerStateParts{NaturesVeilSpent: -1}},
		{"negative general feats", character.RangerStateParts{GeneralFeats: -1}},
		{"negative Epic Boons", character.RangerStateParts{EpicBoons: -1}},
		{"invalid Deft Explorer skill", character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Languages: goodDeft.Languages}}},
		{"invalid first language", character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: goodDeft.Skill, Languages: [2]content.Language{0, content.LanguageElvish}}}},
		{"invalid second language", character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: goodDeft.Skill, Languages: [2]content.Language{content.LanguageElvish, 0}}}},
		{"equal languages", character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: goodDeft.Skill, Languages: [2]content.Language{content.LanguageElvish, content.LanguageElvish}}}},
		{"invalid first Expertise", character.RangerStateParts{Expertise: &[2]content.Skill{0, content.SkillNature}}},
		{"invalid second Expertise", character.RangerStateParts{Expertise: &[2]content.Skill{content.SkillNature, 0}}},
		{"equal Expertise", character.RangerStateParts{Expertise: &[2]content.Skill{content.SkillNature, content.SkillNature}}},
		{"invalid style feat", character.RangerStateParts{FightingStyle: character.RangerFightingStyleFeat{}}},
		{"invalid first cantrip", character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"", "guidance"}}}},
		{"invalid second cantrip", character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", ""}}}},
		{"equal cantrips", character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "guidance"}}}},
		{"a pointer style", character.RangerStateParts{FightingStyle: &character.DruidicWarrior{}}},
	} {
		_, err := character.RestoreRangerState(c.parts)
		wantInvalid(t, err, c.name)
	}
}

// Requirement: doc/specs/class-ranger.md FR-052 (SRD 5.2.1, "Level 2: Fighting Style", p. 59) - the Druidic Warrior cantrips are exposed as GrantedCantrips of the Ranger class on the Druid list, and none without Druidic Warrior.
// Path: happy / unhappy (empty case).
func TestRangerStateDruidicWarriorCantrips(t *testing.T) {
	t.Parallel()
	s := rgrState(t, character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "starry_wisp"}}})
	got := s.DruidicWarriorCantrips()
	if len(got) != 2 || got[0].Spell() != "guidance" || got[1].Spell() != "starry_wisp" || got[0].Class() != character.CasterRanger || got[0].ListClass() != "druid" {
		t.Errorf("cantrips = %+v", got)
	}
	for _, none := range []character.RangerState{character.InitialRangerState(), rgrState(t, character.RangerStateParts{FightingStyle: character.RangerFightingStyleFeat{Feat: "archery"}})} {
		if got := none.DruidicWarriorCantrips(); got == nil || len(got) != 0 {
			t.Errorf("cantrips = %v, want an empty slice", got)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-110 - RangerActionCost has the snake_case names of the three actions and "unknown" outside them; RangerStateKey is named ranger_state.
// Path: unhappy (boundary, invalid input).
func TestRangerActionCostStrings(t *testing.T) {
	t.Parallel()
	for c, want := range map[character.RangerActionCost]string{character.RangerMagicAction: "magic_action", character.RangerBonusAction: "bonus_action", character.RangerReaction: "reaction", 0: "unknown"} {
		if c.String() != want {
			t.Errorf("cost %d = %q, want %q", c, c.String(), want)
		}
	}
	if character.RangerStateKey().Name() != "ranger_state" {
		t.Errorf("key = %q", character.RangerStateKey().Name())
	}
}

// Requirement: doc/specs/class-ranger.md FR-048, FR-051, FR-110 - RestoreRangerState accepts two distinct Expertise skills and two distinct Druidic Warrior cantrips in either order.
// Path: per case (happy: boundary).
func TestRestoreRangerStateAcceptsEitherOrder(t *testing.T) {
	t.Parallel()
	for _, skills := range [][2]content.Skill{{sNature, sSurvival}, {sSurvival, sNature}} {
		s := rgrState(t, character.RangerStateParts{Expertise: &skills})
		if got, ok := s.Expertise(); !ok || got[0] != skills[0] || got[1] != skills[1] {
			t.Errorf("Expertise %v restored as %v", skills, got)
		}
	}
	for _, cantrips := range [][2]content.SpellID{{"guidance", "starry_wisp"}, {"starry_wisp", "guidance"}} {
		s := rgrState(t, character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: cantrips}})
		if style, _ := s.FightingStyle(); style != (character.DruidicWarrior{Cantrips: cantrips}) {
			t.Errorf("cantrips %v restored as %v", cantrips, style)
		}
	}
}
