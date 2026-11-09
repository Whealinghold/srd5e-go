package character_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// rogueTable is the Rogue Features table (SRD 5.2.1, "Rogue Features", p. 62): the class feature ids and the Sneak Attack dice of
// each level. The Subclass feature rows (9, 13, 17) have no class feature id.
var rogueTable = [20]struct {
	features []string
	sneak    int
}{
	{[]string{"rogue_expertise", "rogue_sneak_attack", "rogue_thieves_cant", "rogue_weapon_mastery"}, 1},
	{[]string{"rogue_cunning_action"}, 1},
	{[]string{"rogue_rogue_subclass", "rogue_steady_aim"}, 2},
	{[]string{"rogue_ability_score_improvement"}, 2},
	{[]string{"rogue_cunning_strike", "rogue_uncanny_dodge"}, 3},
	{[]string{"rogue_expertise"}, 3},
	{[]string{"rogue_evasion", "rogue_reliable_talent"}, 4},
	{[]string{"rogue_ability_score_improvement"}, 4},
	{nil, 5},
	{[]string{"rogue_ability_score_improvement"}, 5},
	{[]string{"rogue_improved_cunning_strike"}, 6},
	{[]string{"rogue_ability_score_improvement"}, 6},
	{nil, 7},
	{[]string{"rogue_devious_strikes"}, 7},
	{[]string{"rogue_slippery_mind"}, 8},
	{[]string{"rogue_ability_score_improvement"}, 8},
	{nil, 9},
	{[]string{"rogue_elusive"}, 9},
	{[]string{"rogue_epic_boon"}, 10},
	{[]string{"rogue_stroke_of_luck"}, 10},
}

func featureStrings(ids []content.FeatureID) []string {
	out := []string{}
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

// Requirement: doc/specs/class-rogue.md FR-010, FR-011, FR-012, FR-008 - FeaturesAt and SneakAttackDice equal the Rogue Features
// table at every level 1 to 20, the Proficiency Bonus column of the bundled entry equals the Character Advancement one, and the
// bundled entry holds no Spellcasting or Pact Magic at any level.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level).
func TestRogueFeaturesTable(t *testing.T) {
	t.Parallel()
	def := rogDef(t)
	p := rogProgression(t)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			want := rogueTable[n-1]
			got := featureStrings(p.FeaturesAt(rl(n)))
			if strings.Join(got, ",") != strings.Join(want.features, ",") {
				t.Errorf("FeaturesAt(%d) = %v, want %v", n, got, want.features)
			}
			dice := p.SneakAttackDice(rl(n))
			if dice.Count() != want.sneak || dice.Die().Sides() != 6 {
				t.Errorf("SneakAttackDice(%d) = %dd%d, want %dd6", n, dice.Count(), dice.Die().Sides(), want.sneak)
			}
			row, err := def.RowAt(n)
			if err != nil {
				t.Fatal(err)
			}
			if bonus := character.ProficiencyBonusFor(must(character.NewLevel(n))); row.ProficiencyBonus() != bonus.Value() {
				t.Errorf("row %d Proficiency Bonus = %d, want %d", n, row.ProficiencyBonus(), bonus.Value())
			}
			if row.Spellcasting() != nil {
				t.Errorf("row %d has a Spellcasting cell", n)
			}
		})
	}
	if def.Spellcasting() != nil {
		t.Error("the Rogue entry declares Spellcasting")
	}
}

// Requirement: doc/specs/class-rogue.md FR-014 - FeaturesUpTo returns the features of levels 1 up to and including the level and
// none above it, and the Thief's features of those levels only when the subclass is supplied.
// Path: per case (levels 1, 3, 8, 9, 20 with and without the Thief): unhappy (boundary).
func TestRogueFeaturesUpTo(t *testing.T) {
	t.Parallel()
	p := rogProgression(t)
	for _, c := range []struct {
		level          int
		without, withT int
	}{{1, 4, 4}, {3, 7, 9}, {8, 14, 16}, {9, 14, 17}, {20, 23, 28}} {
		t.Run("level "+strconv.Itoa(c.level), func(t *testing.T) {
			t.Parallel()
			if got := len(p.FeaturesUpTo(rl(c.level), "")); got != c.without {
				t.Errorf("FeaturesUpTo(%d) without a subclass = %d features, want %d", c.level, got, c.without)
			}
			if got := len(p.FeaturesUpTo(rl(c.level), character.ThiefSubclassID)); got != c.withT {
				t.Errorf("FeaturesUpTo(%d) as a Thief = %d features, want %d", c.level, got, c.withT)
			}
		})
	}
	if got := featureStrings(p.FeaturesUpTo(rl(3), character.ThiefSubclassID)); got[len(got)-1] != "thief_second_story_work" || got[len(got)-2] != "thief_fast_hands" {
		t.Errorf("level 3 as a Thief ends with %v", got)
	}
	if got := p.FeaturesUpTo(rl(2), "other"); len(got) != 5 {
		t.Errorf("another subclass id adds features: %v", got)
	}
}

// Requirement: doc/specs/class-rogue.md FR-009, FR-080, FR-084, FR-085, FR-090, FR-111 - the constants of RogueProgression equal the
// bundled Rogue entry: subclass level and id, the Ability Score Improvement levels, the Epic Boon level, the Weapon Mastery
// feature and Slippery Mind at 15, and the Thief's feature levels.
// Path: unhappy (missing content: a constant that drifted from the entry).
func TestRogueConstantsMatchTheBundledEntry(t *testing.T) {
	t.Parallel()
	def := rogDef(t)
	p := rogProgression(t)
	if def.SubclassLevel() != p.SubclassLevel() || p.SubclassLevel() != 3 || p.SubclassID() != character.ThiefSubclassID || def.Subclasses()[0].ID() != p.SubclassID() {
		t.Errorf("subclass level %d / id %s, entry %d / %s", p.SubclassLevel(), p.SubclassID(), def.SubclassLevel(), def.Subclasses()[0].ID())
	}
	var asi []int
	boon, slippery := 0, 0
	for _, f := range def.Features() {
		switch f.ID() {
		case "rogue_epic_boon":
			boon = f.Level()
		case "rogue_slippery_mind":
			slippery = f.Level()
		}
	}
	for n := 1; n <= 20; n++ {
		if p.GrantsFeat(rl(n)) {
			asi = append(asi, n)
		}
		for _, id := range p.FeaturesAt(rl(n)) {
			if id == "rogue_ability_score_improvement" != p.GrantsFeat(rl(n)) {
				t.Errorf("level %d: GrantsFeat = %v but the row lists the feature = %v", n, p.GrantsFeat(rl(n)), id == "rogue_ability_score_improvement")
			}
		}
	}
	if boon != p.EpicBoonLevel() || boon != 19 || slippery != 15 || p.WeaponMasteryCount() != 2 {
		t.Errorf("Epic Boon %d (entry %d), Slippery Mind %d, mastery count %d", p.EpicBoonLevel(), boon, slippery, p.WeaponMasteryCount())
	}
	if got := p.FeatLevels(); len(got) != 5 || got[0] != 4 || got[4] != 16 || len(asi) != 5 {
		t.Errorf("FeatLevels = %v, GrantsFeat levels = %v; want 4, 8, 10, 12, 16", got, asi)
	}
	feats := p.FeatLevels()
	feats[0] = 99
	if p.FeatLevels()[0] != 4 {
		t.Error("FeatLevels returned shared storage")
	}
}

// Requirement: doc/specs/class-rogue.md FR-080 - Dexterity and Intelligence for a first class entry at levels 1 to 14, plus Wisdom
// and Charisma from 15 (Slippery Mind); for a multiclass entry nothing before 15 and Wisdom and Charisma from 15; never Wisdom
// or Charisma at 14.
// Path: per case (levels 14 and 15 for both entries): unhappy (boundary).
func TestRogueSavingThrowProficiencies(t *testing.T) {
	t.Parallel()
	p := rogProgression(t)
	dex, intel, wis, cha := ability.Dexterity, ability.Intelligence, ability.Wisdom, ability.Charisma
	for _, c := range []struct {
		name  string
		entry character.RogueEntry
		level int
		want  []ability.Ability
	}{
		{"first class level 1", character.RogueFirstClass, 1, []ability.Ability{dex, intel}},
		{"first class level 14", character.RogueFirstClass, 14, []ability.Ability{dex, intel}},
		{"first class level 15", character.RogueFirstClass, 15, []ability.Ability{dex, intel, wis, cha}},
		{"first class level 20", character.RogueFirstClass, 20, []ability.Ability{dex, intel, wis, cha}},
		{"multiclass level 1", character.RogueMulticlass, 1, []ability.Ability{}},
		{"multiclass level 14", character.RogueMulticlass, 14, []ability.Ability{}},
		{"multiclass level 15", character.RogueMulticlass, 15, []ability.Ability{wis, cha}},
		{"undefined entry", character.RogueEntry(0), 20, []ability.Ability{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := p.SavingThrowProficiencies(rl(c.level), c.entry)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("got %v, want %v", got, c.want)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-009, FR-103 - NewRogueProgression accepts the Rogue definition and joins a typed reason
// for another class or the zero definition (NotTheRogueDefinition), a renamed Sneak Attack column (RogueColumnMissing), a Sneak
// Attack cell that is not d6 (RogueRowsIncomplete) and a missing Thief (RogueSubclassMissing); a definition that declares
// Spellcasting and lacks the Rogue skill lists reports those too. Each matches srderr.ErrInvalidInput and the Must helper
// panics with the same error. The zero progression answers empty values.
// Path: per case (unhappy: missing content).
func TestNewRogueProgression(t *testing.T) {
	t.Parallel()
	rogue := fgtBundledText(t, "class_rogue.json")
	asRogue := fgtDecode(t, strings.ReplaceAll(strings.ReplaceAll(fgtBundledText(t, "class_wizard.json"), `"id": "wizard"`, `"id": "rogue"`), "wizard_", "rogue_"))
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		want []character.RogueProgressionReason
	}{
		{"another class", fgtDef(t), []character.RogueProgressionReason{character.NotTheRogueDefinition}},
		{"the zero definition", content.ClassDefinition{}, []character.RogueProgressionReason{character.NotTheRogueDefinition}},
		{"a renamed column", fgtDecode(t, strings.ReplaceAll(rogue, "sneak_attack", "sneak_attacks")), []character.RogueProgressionReason{character.RogueColumnMissing}},
		{"a cell that is not d6", fgtDecode(t, strings.Replace(rogue, `"sides": 6`, `"sides": 8`, 1)), []character.RogueProgressionReason{character.RogueRowsIncomplete}},
		{"no Thief", fgtDecode(t, strings.ReplaceAll(strings.ReplaceAll(rogue, `"id": "thief"`, `"id": "burglar"`), "thief_", "burglar_")), []character.RogueProgressionReason{character.RogueSubclassMissing}},
		{"a spellcaster", asRogue, []character.RogueProgressionReason{character.RogueColumnMissing, character.RogueSpellcastingPresent, character.RogueSubclassMissing, character.RogueSkillListMissing}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := character.NewRogueProgression(c.def)
			wantInvalid(t, err, c.name)
			if !p.IsZero() {
				t.Errorf("progression = %v, want the zero progression", p)
			}
			for _, r := range c.want {
				if !errors.Is(err, &character.RogueProgressionError{Reason: r}) || r.String() == "unknown" {
					t.Errorf("error %v does not carry reason %v", err, r)
				}
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustRogueProgression did not panic")
				}
			}()
			character.MustRogueProgression(c.def)
		})
	}
	var zero character.RogueProgression
	if !zero.IsZero() || zero.SneakAttackDice(rl(5)).Count() != 0 || len(zero.FeaturesAt(rl(1))) != 0 || len(zero.FeaturesUpTo(rl(1), "")) != 0 {
		t.Error("the zero progression answers non-empty values")
	}
	p := rogProgression(t)
	if p.IsZero() || p.SneakAttackDice(character.RogueLevel{}).Count() != 0 || len(p.FeaturesAt(character.RogueLevel{})) != 0 || len(p.FeaturesUpTo(character.RogueLevel{}, "")) != 0 {
		t.Error("the zero level answers non-empty values")
	}
}

// Requirement: doc/specs/class-rogue.md FR-103 - the reasons of a refused definition render as snake_case and an unknown one as
// "unknown"; a progression error matches only the same reason; SneakAttackDice, FeaturesAt and FeaturesUpTo return copies.
// Path: unhappy (immutability).
func TestRogueProgressionReasonsAndCopies(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.RogueProgressionReason]string{
		character.NotTheRogueDefinition: "not_the_rogue_definition", character.RogueColumnMissing: "column_missing", character.RogueRowsIncomplete: "rows_incomplete",
		character.RogueSubclassMissing: "subclass_missing", character.RogueSpellcastingPresent: "spellcasting_present", character.RogueSkillListMissing: "skill_list_missing", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("reason %d = %q, want %q", r, r.String(), want)
		}
	}
	err := &character.RogueProgressionError{Reason: character.RogueColumnMissing}
	if errors.Is(err, &character.RogueProgressionError{Reason: character.RogueRowsIncomplete}) || err.Error() == "" || errors.Is(err, errors.New("other")) {
		t.Error("a progression error matched another reason or has no text")
	}
	p := rogProgression(t)
	a := p.FeaturesAt(rl(1))
	a[0] = "changed"
	b := p.FeaturesUpTo(rl(1), "")
	b[0] = "changed"
	if p.FeaturesAt(rl(1))[0] != "rogue_expertise" || p.FeaturesUpTo(rl(1), "")[0] != "rogue_expertise" {
		t.Error("a returned slice shares storage with the progression")
	}
}

// Requirement: doc/specs/class-rogue.md FR-048, FR-102 - SneakAttackDice carries a count of d6, never a rolled total; Expression
// converts to count d6 with modifier 0 (the zero expression for no dice); a negative count is refused with ErrInvalidInput.
// Path: per case (counts 0, 1, 10, -1): unhappy (boundary).
func TestSneakAttackDice(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 1, 10} {
		d := dice6(n)
		if d.Count() != n || d.Die() != dice.MustDie(6) {
			t.Errorf("dice(%d) = %d d%d", n, d.Count(), d.Die().Sides())
		}
		e := d.Expression()
		if n == 0 {
			if !e.IsZero() {
				t.Errorf("Expression of no dice = %v, want the zero expression", e)
			}
			continue
		}
		if e.Count() != n || e.Die().Sides() != 6 || e.Modifier() != 0 {
			t.Errorf("Expression(%d) = %dd%d%+d", n, e.Count(), e.Die().Sides(), e.Modifier())
		}
	}
	_, err := character.NewSneakAttackDice(-1)
	wantInvalid(t, err, "negative die count")
}

// Requirement: doc/specs/class-rogue.md FR-021 - RogueEntry renders and validates its two members.
// Path: unhappy (boundary: the zero value).
func TestRogueEntry(t *testing.T) {
	t.Parallel()
	for e, want := range map[character.RogueEntry]string{character.RogueFirstClass: "first_class", character.RogueMulticlass: "multiclass", 0: "unknown"} {
		if e.String() != want || e.Valid() != (want != "unknown") {
			t.Errorf("entry %d = %q valid %v", e, e.String(), e.Valid())
		}
	}
}
