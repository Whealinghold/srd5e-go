package character_test

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-ranger.md FR-023, FR-030, T1 (SRD 5.2.1, "Ranger Features", p. 58; "Proficiency", p. 8) - the Proficiency Bonus column of the bundled Ranger Features table equals the general
// Proficiency Bonus for the same level, it has no Cantrips column, and its Prepared Spells and highest Spell Slot level agree with what spellcasting derives for the Ranger. A disagreement fails
// this consistency test, not a decode.
// Path: per case (the 20 levels): unhappy (rule interaction).
func TestRangerTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	def := rgrDef(t)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must(def.RowAt(n))
			sc := row.Spellcasting().(content.SlotSpellcastingRow)
			_, hasCantrips := sc.Cantrips()
			highest := 0
			for i, s := range sc.SpellSlots() {
				if s > 0 {
					highest = i + 1
				}
			}
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() || hasCantrips ||
				sc.PreparedSpells() != character.PreparedLimit(character.CasterRanger, n) || highest != character.MaxPreparableLevel(character.CasterRanger, n) {
				t.Errorf("level %d row = bonus %d cantrips %v prepared %d highest slot %d; the rules say %d no cantrips %d %d", n, row.ProficiencyBonus(), hasCantrips, sc.PreparedSpells(), highest,
					character.ProficiencyBonusFor(character.MustLevel(n)).Value(), character.PreparedLimit(character.CasterRanger, n), character.MaxPreparableLevel(character.CasterRanger, n))
			}
		})
	}
}

// Requirement: doc/specs/class-ranger.md FR-023, FR-032, T1 (SRD 5.2.1, "Ranger Features", p. 58) - FavoredEnemyUses reads the Favored Enemy column of the bundled table: 2 at levels 1 to 4, 3 at 5 to 8, 4 at 9 to
// 12, 5 at 13 to 16 and 6 at 17 to 20; the zero progression and level give 0.
// Path: per case (the 20 levels): happy / unhappy (boundary, empty case).
func TestRangerFavoredEnemyUsesByLevel(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	want := []int{2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6}
	for n := 1; n <= 20; n++ {
		if got := p.FavoredEnemyUses(character.MustRangerLevel(n)); got != want[n-1] {
			t.Errorf("FavoredEnemyUses(%d) = %d, want %d", n, got, want[n-1])
		}
	}
	if p.IsZero() || !(character.RangerProgression{}).IsZero() || p.FavoredEnemyUses(character.RangerLevel{}) != 0 || (character.RangerProgression{}).FavoredEnemyUses(character.MustRangerLevel(5)) != 0 {
		t.Error("zero progression or level gives a non-zero maximum")
	}
}

// Requirement: doc/specs/class-ranger.md FR-023, T2 (SRD 5.2.1, "Ranger Features", p. 58) - the Favored Enemy and Prepared Spells steps between levels: each row of T2 is the pair of values at the from
// and to levels.
// Path: per case (T2 rows FE-01 to FE-09): unhappy (boundary).
func TestRangerFavoredEnemyAndPreparedSpellsSteps(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	def := rgrDef(t)
	for _, c := range []struct {
		label                    string
		from, to                 int
		favoredFrom, favoredTo   int
		preparedFrom, preparedTo int
	}{
		{"FE-01", 4, 5, 2, 3, 5, 6}, {"FE-02", 8, 9, 3, 4, 7, 9}, {"FE-03", 12, 13, 4, 5, 10, 11}, {"FE-04", 16, 17, 5, 6, 12, 14}, {"FE-05", 1, 2, 2, 2, 2, 3},
		{"FE-06", 5, 6, 3, 3, 6, 6}, {"FE-07", 19, 20, 6, 6, 15, 15}, {"FE-08", 6, 7, 3, 3, 6, 7}, {"FE-09", 18, 19, 6, 6, 14, 15},
	} {
		prepared := func(n int) int {
			return must(def.RowAt(n)).Spellcasting().(content.SlotSpellcastingRow).PreparedSpells()
		}
		got := []int{p.FavoredEnemyUses(character.MustRangerLevel(c.from)), p.FavoredEnemyUses(character.MustRangerLevel(c.to)), prepared(c.from), prepared(c.to)}
		if want := []int{c.favoredFrom, c.favoredTo, c.preparedFrom, c.preparedTo}; !slices.Equal(got, want) {
			t.Errorf("%s: %d to %d = %v, want %v", c.label, c.from, c.to, got, want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-046, FR-096 (SRD 5.2.1, "Ranger Class Features", pp. 57-59; "Ranger Subclass: Hunter", p. 61) - LevelOf gives the level of every gating feature entry of the bundled
// data, and 0 for an invalid feature, the zero progression, or a Hunter feature when the progression was built without the Hunter.
// Path: per case (the 20 features): happy / unhappy (invalid input, missing subclass).
func TestRangerLevelOf(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	noHunter := must(character.NewRangerProgression(rgrDef(t), nil))
	for _, c := range []struct {
		f      character.RangerFeature
		level  int
		hunter bool
	}{
		{character.RangerFeatureFavoredEnemy, 1, false}, {character.RangerFeatureWeaponMastery, 1, false}, {character.RangerFeatureDeftExplorer, 2, false},
		{character.RangerFeatureFightingStyle, 2, false}, {character.RangerFeatureExtraAttack, 5, false}, {character.RangerFeatureRoving, 6, false},
		{character.RangerFeatureExpertise, 9, false}, {character.RangerFeatureTireless, 10, false}, {character.RangerFeatureRelentlessHunter, 13, false},
		{character.RangerFeatureNaturesVeil, 14, false}, {character.RangerFeaturePreciseHunter, 17, false}, {character.RangerFeatureFeralSenses, 18, false},
		{character.RangerFeatureFoeSlayer, 20, false}, {character.RangerFeatureEpicBoon, 19, false}, {character.RangerFeatureSubclass, 3, false},
		{character.RangerFeatureHuntersLore, 3, true}, {character.RangerFeatureHuntersPrey, 3, true}, {character.RangerFeatureDefensiveTactics, 7, true},
		{character.RangerFeatureSuperiorHuntersPrey, 11, true}, {character.RangerFeatureSuperiorHuntersDefense, 15, true},
	} {
		if p.LevelOf(c.f) != c.level || c.f.String() == "unknown" || !c.f.Valid() {
			t.Errorf("LevelOf(%v) = %d, want %d", c.f, p.LevelOf(c.f), c.level)
		}
		want := c.level
		if c.hunter {
			want = 0
		}
		if noHunter.LevelOf(c.f) != want {
			t.Errorf("without the Hunter LevelOf(%v) = %d, want %d", c.f, noHunter.LevelOf(c.f), want)
		}
	}
	if p.LevelOf(0) != 0 || p.LevelOf(99) != 0 || (character.RangerProgression{}).LevelOf(character.RangerFeatureFoeSlayer) != 0 || character.RangerFeature(0).String() != "unknown" || character.RangerFeature(99).Valid() {
		t.Error("invalid feature or zero progression has a level")
	}
	if character.RangerFeatureNaturesVeil.String() != "Nature’s Veil" {
		t.Errorf("String = %q", character.RangerFeatureNaturesVeil.String())
	}
}

// Requirement: doc/specs/class-ranger.md FR-046 (SRD 5.2.1, "Level 6: Roving", p. 59) - the level gates follow the loaded definition: a definition whose Roving entry is at level 7 gives no Roving at level 6.
// Path: unhappy (rule interaction: a gate moved by the definition).
func TestRangerGatesFollowTheDefinition(t *testing.T) {
	t.Parallel()
	moved := rgrDefFrom(t, func(s string) string {
		ref := "{\n              \"featureId\": \"ranger_roving\"\n            }"
		s = strings.Replace(s, "\"features\": [\n            "+ref+"\n          ],", "\"features\": [],", 1)
		s = strings.Replace(s, "\"level\": 7,\n          \"proficiencyBonus\": 3,\n          \"features\": [\n", "\"level\": 7,\n          \"proficiencyBonus\": 3,\n          \"features\": [\n            "+ref+",\n", 1)
		return strings.Replace(s, "\"id\": \"ranger_roving\",\n          \"name\": \"Roving\",\n          \"level\": 6,", "\"id\": \"ranger_roving\",\n          \"name\": \"Roving\",\n          \"level\": 7,", 1)
	})
	p := must(character.NewRangerProgression(moved, nil))
	if p.LevelOf(character.RangerFeatureRoving) != 7 {
		t.Fatalf("Roving level = %d, want 7", p.LevelOf(character.RangerFeatureRoving))
	}
	for _, c := range []struct{ level, want int }{{6, 30}, {7, 40}} {
		f := character.RangerFacts{Progression: p, Level: character.MustRangerLevel(c.level)}
		if got := f.Speed(character.MustSpeedInput(30, true), false).Walk(); got != c.want {
			t.Errorf("level %d walks %d, want %d", c.level, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-031, FR-058 (SRD 5.2.1, "Level 1: Favored Enemy", p. 58) - AlwaysPrepared returns the Hunter’s Mark grant of the Favored Enemy fact from level 1, as a new
// slice, and HuntersMark names the spell; the zero progression and level give none.
// Path: per case: happy / unhappy (boundary, empty case, mutation of the result).
func TestRangerAlwaysPrepared(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	for _, n := range []int{1, 2, 20} {
		got := p.AlwaysPrepared(character.MustRangerLevel(n))
		if !slices.Equal(got, []content.SpellID{"hunters_mark"}) {
			t.Errorf("AlwaysPrepared(%d) = %v, want Hunter’s Mark", n, got)
		}
		got[0] = "x"
		if p.AlwaysPrepared(character.MustRangerLevel(n))[0] != "hunters_mark" {
			t.Error("AlwaysPrepared returned shared storage")
		}
	}
	if p.HuntersMark() != "hunters_mark" || (character.RangerProgression{}).HuntersMark() != "" ||
		len(p.AlwaysPrepared(character.RangerLevel{})) != 0 || len((character.RangerProgression{}).AlwaysPrepared(character.MustRangerLevel(5))) != 0 {
		t.Error("HuntersMark or the empty case is wrong")
	}
}

// Requirement: doc/specs/class-ranger.md FR-062, FR-064, T6 (SRD 5.2.1, "Ranger Features", p. 58; "Level 4: Ability Score Improvement", "Level 19: Epic Boon", p. 59) - FeatGrantsUpTo gives the sums of the General
// and Epic Boon grants for levels 1 to L: rows FG-01 to FG-09 of T6; the Fighting Style grant is not counted.
// Path: per case (T6): happy / unhappy (boundary: one below, maximum level).
func TestRangerFeatGrantsUpTo(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	for _, c := range []struct {
		label         string
		level         int
		general, epic int
	}{
		{"FG-01", 3, 0, 0}, {"FG-02", 4, 1, 0}, {"FG-03", 7, 1, 0}, {"FG-04", 8, 2, 0}, {"FG-05", 12, 3, 0}, {"FG-06", 16, 4, 0}, {"FG-07", 18, 4, 0}, {"FG-08", 19, 4, 1}, {"FG-09", 20, 4, 1},
	} {
		if g, e := p.FeatGrantsUpTo(character.MustRangerLevel(c.level)); g != c.general || e != c.epic {
			t.Errorf("%s: FeatGrantsUpTo(%d) = %d, %d; want %d, %d", c.label, c.level, g, e, c.general, c.epic)
		}
	}
	if g, e := (character.RangerProgression{}).FeatGrantsUpTo(character.MustRangerLevel(20)); g != 0 || e != 0 {
		t.Error("the zero progression grants feats")
	}
	if g, e := p.FeatGrantsUpTo(character.RangerLevel{}); g != 0 || e != 0 {
		t.Error("the zero level grants feats")
	}
}

// Requirement: doc/specs/class-ranger.md FR-096 (SRD 5.2.1, "Ranger Features", p. 58) - FeaturesGainedAt returns the ids of the class features gained at a level in printed order, as a new slice, and an empty slice for the zero level.
// Path: per case: happy / unhappy (boundary, empty case, mutation of the result).
func TestRangerFeaturesGainedAt(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	got := p.FeaturesGainedAt(character.MustRangerLevel(1))
	if want := []content.FeatureID{"ranger_spellcasting", "ranger_favored_enemy", "ranger_weapon_mastery"}; !slices.Equal(got, want) {
		t.Errorf("level 1 gains %v, want %v", got, want)
	}
	got[0] = "x"
	if p.FeaturesGainedAt(character.MustRangerLevel(1))[0] != "ranger_spellcasting" {
		t.Error("FeaturesGainedAt returned shared storage")
	}
	if len(p.FeaturesGainedAt(character.RangerLevel{})) != 0 || len((character.RangerProgression{}).FeaturesGainedAt(character.MustRangerLevel(1))) != 0 {
		t.Error("the empty case gains features")
	}
}

// Requirement: doc/specs/class-ranger.md FR-086, FR-091 (Table OP) - HuntersPreyOptions and DefensiveTacticsOptions return the option ids of the Hunter entries, as new slices, and none without the Hunter.
// Path: per case: happy / unhappy (missing subclass, mutation of the result).
func TestRangerOptionIDs(t *testing.T) {
	t.Parallel()
	p := rgrProgression(t)
	if !slices.Equal(p.HuntersPreyOptions(), []string{"colossus_slayer", "horde_breaker"}) || !slices.Equal(p.DefensiveTacticsOptions(), []string{"escape_the_horde", "multiattack_defense"}) {
		t.Errorf("options = %v, %v", p.HuntersPreyOptions(), p.DefensiveTacticsOptions())
	}
	p.HuntersPreyOptions()[0] = "x"
	if p.HuntersPreyOptions()[0] != "colossus_slayer" {
		t.Error("HuntersPreyOptions returned shared storage")
	}
	noHunter := must(character.NewRangerProgression(rgrDef(t), nil))
	if len(noHunter.HuntersPreyOptions()) != 0 || len(noHunter.DefensiveTacticsOptions()) != 0 {
		t.Error("a progression without the Hunter has options")
	}
}

// Requirement: doc/specs/class-ranger.md FR-046, FR-101 (SRD 5.2.1, "Ranger Features", p. 58) - NewRangerProgression rejects a definition that is another class, the zero definition, a Hunter given as another subclass, a missing
// Favored Enemy column, a gap in the column, or a missing gating entry, always-prepared fact or option list, and lists every problem found at once, each matching srderr.ErrInvalidInput; MustRangerProgression panics with that error.
// Path: per case (unhappy: invalid input).
func TestNewRangerProgressionRejects(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		sub  *content.SubclassDefinition
		want character.RangerProgressionError
	}{
		{"another class", fgtDef(t), nil, character.RangerProgressionError{Kind: character.RangerWrongClass}},
		{"zero definition", content.ClassDefinition{}, nil, character.RangerProgressionError{Kind: character.RangerWrongClass}},
		{"another subclass", rgrDef(t), pldOath(t), character.RangerProgressionError{Kind: character.RangerWrongClass}},
		{"no favored_enemy column", rgrDefFrom(t, func(s string) string { return strings.ReplaceAll(s, "favored_enemy", "favored_x") }), nil,
			character.RangerProgressionError{Kind: character.RangerMissingFavoredEnemyColumn}},
		{"a missing Favored Enemy fact", rgrDefFrom(t, func(s string) string { return factRE.ReplaceAllString(s, "") }), nil,
			character.RangerProgressionError{Kind: character.RangerMissingFeatureEntry, Feature: "Favored Enemy"}},
		{"a missing Roving", rgrDefFrom(t, func(s string) string { return strings.ReplaceAll(s, "ranger_roving", "ranger_x_roving") }), nil,
			character.RangerProgressionError{Kind: character.RangerMissingFeatureEntry, Feature: "Roving"}},
		{"a missing Hunter’s Prey option", rgrDef(t), hunterWithout(t, "horde_breaker"), character.RangerProgressionError{Kind: character.RangerMissingFeatureEntry, Feature: "Hunter’s Prey"}},
	} {
		_, err := character.NewRangerProgression(c.def, c.sub)
		wantInvalid(t, err, c.name)
		if !errors.Is(err, &c.want) {
			t.Errorf("%s: error = %v, want problem %v", c.name, err, c.want)
		}
	}
	multi := rgrDefFrom(t, func(s string) string {
		s = strings.ReplaceAll(s, "ranger_tireless", "ranger_x_tireless")
		return strings.ReplaceAll(s, "ranger_expertise", "ranger_x_expertise")
	})
	_, err := character.NewRangerProgression(multi, nil)
	for _, name := range []string{"Tireless", "Expertise"} {
		if !errors.Is(err, &character.RangerProgressionError{Kind: character.RangerMissingFeatureEntry, Feature: name}) {
			t.Errorf("error = %v, want a missing %s", err, name)
		}
	}
	var pe *character.RangerProgressionError
	if !errors.As(err, &pe) || pe.Error() == "" || pe.Kind.String() != "missing_feature_entry" {
		t.Errorf("errors.As = %v", pe)
	}
	for k, want := range map[character.RangerProblemKind]string{character.RangerWrongClass: "wrong_class", character.RangerMissingFavoredEnemyColumn: "missing_favored_enemy_column", 0: "unknown"} {
		if k.String() != want {
			t.Errorf("kind %d = %q, want %q", k, k.String(), want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustRangerProgression did not panic")
		}
	}()
	character.MustRangerProgression(fgtDef(t), nil)
}

var factRE = regexp.MustCompile(`,\s*"facts": \[\s*\{\s*"kind": "alwaysPreparedSpells",\s*"grants": \[\s*\{\s*"level": 1,\s*"spellIds": \[\s*"hunters_mark"\s*\]\s*\}\s*\]\s*\}\s*\]`)

// hunterWithout returns the Hunter decoded from the bundled Ranger file with the option removed.
func hunterWithout(t *testing.T, option string) *content.SubclassDefinition {
	t.Helper()
	re := regexp.MustCompile(`,\s*\{\s*"id": "` + option + `",[^}]*\}`)
	def := rgrDefFrom(t, func(s string) string { return re.ReplaceAllString(s, "") })
	sub := def.Subclasses()[0]
	return &sub
}

// Requirement: doc/specs/class-ranger.md FR-097 (SRD 5.2.1, "Ranger Class Features", pp. 58-59) - PendingChoice names are snake_case, distinct and "unknown" outside the set.
// Path: unhappy (boundary, invalid input).
func TestRangerPendingChoiceStrings(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for c := character.RangerPendingWeaponMastery; c <= character.RangerPendingEpicBoon; c++ {
		if s := c.String(); s == "unknown" || seen[s] {
			t.Errorf("choice %d = %q", c, s)
		}
		seen[c.String()] = true
	}
	if len(seen) != 9 || character.RangerPendingChoice(0).String() != "unknown" || (character.RangerPendingEpicBoon+1).String() != "unknown" {
		t.Errorf("%d choices", len(seen))
	}
}

// Requirement: doc/specs/class-ranger.md FR-046, FR-086, FR-091, FR-031 (Table OP) - NewRangerProgression refuses a class id that sorts after "ranger", a Hunter given as a subclass that sorts before "hunter", a Favored Enemy column of another kind, an option list of one or three options for
// either Hunter feature, builds from a valid definition with MustRangerProgression, reads the last level's features and honours the level of the Hunter’s Mark grant.
// Path: per case (unhappy: invalid input, boundary; happy).
func TestNewRangerProgressionEdges(t *testing.T) {
	t.Parallel()
	_, err := character.NewRangerProgression(rogDef(t), nil)
	if !errors.Is(err, &character.RangerProgressionError{Kind: character.RangerWrongClass}) {
		t.Errorf("a class id after ranger = %v", err)
	}
	champion := fgtDef(t).Subclasses()[0]
	_, err = character.NewRangerProgression(rgrDef(t), &champion)
	if !errors.Is(err, &character.RangerProgressionError{Kind: character.RangerWrongClass}) {
		t.Errorf("the Champion as the Hunter = %v", err)
	}
	for _, kind := range []string{"bonus", "feetBonus"} {
		other := rgrDefFrom(t, func(s string) string {
			return strings.Replace(s, "\"id\": \"favored_enemy\",\n          \"kind\": \"count\"", "\"id\": \"favored_enemy\",\n          \"kind\": \""+kind+"\"", 1)
		})
		_, err = character.NewRangerProgression(other, nil)
		if !errors.Is(err, &character.RangerProgressionError{Kind: character.RangerMissingFavoredEnemyColumn}) {
			t.Errorf("a %s column = %v", kind, err)
		}
	}
	for _, c := range []struct{ option, feature string }{{"multiattack_defense", "Defensive Tactics"}, {"horde_breaker", "Hunter’s Prey"}} {
		_, err = character.NewRangerProgression(rgrDef(t), hunterWithout(t, c.option))
		if !errors.Is(err, &character.RangerProgressionError{Kind: character.RangerMissingFeatureEntry, Feature: c.feature}) {
			t.Errorf("one option missing for %s = %v", c.feature, err)
		}
	}
	for _, c := range []struct{ option, feature string }{{"horde_breaker", "Hunter’s Prey"}, {"multiattack_defense", "Defensive Tactics"}} {
		re := regexp.MustCompile(`(\{\s*"id": "` + c.option + `",[^}]*\})`)
		three := rgrDefFrom(t, func(s string) string {
			return re.ReplaceAllString(s, `$1, {"id": "extra_option", "name": "Extra", "text": "Extra."}`)
		})
		sub := three.Subclasses()[0]
		_, err = character.NewRangerProgression(rgrDef(t), &sub)
		if !errors.Is(err, &character.RangerProgressionError{Kind: character.RangerMissingFeatureEntry, Feature: c.feature}) {
			t.Errorf("three options for %s = %v", c.feature, err)
		}
	}
	if p := character.MustRangerProgression(rgrDef(t), rgrHunter(t)); p.IsZero() {
		t.Error("MustRangerProgression returned the zero progression")
	}
	if got := rgrProgression(t).FeaturesGainedAt(character.MustRangerLevel(20)); !slices.Equal(got, []content.FeatureID{"ranger_foe_slayer"}) {
		t.Errorf("level 20 gains %v", got)
	}
	late := rgrDefFrom(t, func(s string) string {
		return strings.Replace(s, "\"level\": 1,\n                  \"spellIds\"", "\"level\": 2,\n                  \"spellIds\"", 1)
	})
	p := must(character.NewRangerProgression(late, nil))
	if len(p.AlwaysPrepared(character.MustRangerLevel(1))) != 0 || !slices.Equal(p.AlwaysPrepared(character.MustRangerLevel(2)), []content.SpellID{"hunters_mark"}) {
		t.Errorf("a grant at level 2: %v then %v", p.AlwaysPrepared(character.MustRangerLevel(1)), p.AlwaysPrepared(character.MustRangerLevel(2)))
	}
}
