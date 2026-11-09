package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func druTableInput(tb testing.TB) character.DruidTablesInput {
	tb.Helper()
	spells := func(ids ...string) []content.SpellID {
		out := make([]content.SpellID, len(ids))
		for i, id := range ids {
			out[i] = content.SpellID(id)
		}
		return out
	}
	land := func(l character.LandType, r content.DamageType, tag string) character.DruidLand {
		return character.DruidLand{Land: l, Resistance: r,
			Spells: [4][]content.SpellID{spells(tag + "_3a"), spells(tag + "_5"), spells(tag + "_7"), spells(tag + "_9")}}
	}
	return character.DruidTablesInput{
		WildShapeUses: []int{0, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4},
		Tiers: []character.BeastShapeTier{
			{MinLevel: 2, KnownForms: 4, MaxChallengeRating: content.MustChallengeRating("1/4")},
			{MinLevel: 4, KnownForms: 6, MaxChallengeRating: content.MustChallengeRating("1/2")},
			{MinLevel: 8, KnownForms: 8, MaxChallengeRating: content.MustChallengeRating("1"), FlySpeed: true}},
		Lands: []character.DruidLand{land(character.LandArid, content.DamageFire, "arid"), land(character.LandPolar, content.DamageCold, "polar"),
			land(character.LandTemperate, content.DamageLightning, "temperate"), land(character.LandTropical, content.DamagePoison, "tropical")},
	}
}

// Requirement: doc/specs/class-druid.md FR-172, FR-058, FR-072, FR-119, FR-131 and content-classes-data.md tables CC, TB, FA (SRD 5.2.1, "Druid
// Features", p. 41; "Beast Shapes", p. 42; "Circle of the Land", p. 46) - the tables built from the bundled Druid class data hold the Wild Shape uses
// of the 20 levels (none at 1, 2 at 2-5, 3 at 6-16, 4 at 17-20), the three Beast Shapes tiers ("1/4" parsed, "Yes" and "No" as booleans), the Circle
// Spells of the four lands at the levels 3, 5, 7 and 9, the Resistance of each land and the features granted at exactly each level, written here by
// hand from the printed pages.
// Path: per case (happy / unhappy: boundary at each level).
func TestDruidTablesFromClassMatchTheBook(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	wantUses := []int{0, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4}
	for level, want := range wantUses {
		if got := tables.WildShapeUses(level + 1); got != want {
			t.Errorf("WildShapeUses(%d) = %d, want %d", level+1, got, want)
		}
	}
	if tables.WildShapeUses(0) != 0 || tables.WildShapeUses(21) != 0 || tables.IsZero() {
		t.Error("a level outside 1 to 20 gives uses, or the built tables are zero")
	}
	wantTiers := []character.BeastShapeTier{{MinLevel: 2, KnownForms: 4, MaxChallengeRating: content.MustChallengeRating("1/4")},
		{MinLevel: 4, KnownForms: 6, MaxChallengeRating: content.MustChallengeRating("1/2")}, {MinLevel: 8, KnownForms: 8, MaxChallengeRating: content.MustChallengeRating("1"), FlySpeed: true}}
	if got := tables.BeastShapes(); !slices.Equal(got, wantTiers) {
		t.Errorf("BeastShapes = %v, want %v", got, wantTiers)
	}
	for _, c := range []struct{ level, known int }{{1, 0}, {2, 4}, {3, 4}, {4, 6}, {7, 6}, {8, 8}, {20, 8}} {
		if got := tables.KnownFormsAt(c.level); got != c.known {
			t.Errorf("KnownFormsAt(%d) = %d, want %d", c.level, got, c.known)
		}
	}
	rules := character.DruidRules{}
	for _, c := range []struct {
		land     character.LandType
		perLevel [4]string // the spells added at 3, 5, 7 and 9
		ward     content.DamageType
	}{
		{character.LandArid, [4]string{"blur burning_hands fire_bolt", "fireball", "blight", "wall_of_stone"}, content.DamageFire},
		{character.LandPolar, [4]string{"fog_cloud hold_person ray_of_frost", "sleet_storm", "ice_storm", "cone_of_cold"}, content.DamageCold},
		{character.LandTemperate, [4]string{"misty_step shocking_grasp sleep", "lightning_bolt", "freedom_of_movement", "tree_stride"}, content.DamageLightning},
		{character.LandTropical, [4]string{"acid_splash ray_of_sickness web", "stinking_cloud", "polymorph", "insect_plague"}, content.DamagePoison},
	} {
		if r, ok := tables.Resistance(c.land); !ok || r != c.ward {
			t.Errorf("Resistance(%v) = %v, %v; want %v", c.land, r, ok, c.ward)
		}
		var want []content.SpellID
		for i, row := range c.perLevel {
			for _, id := range splitWords(row) {
				want = append(want, content.SpellID(id))
			}
			level := 3 + 2*i
			for _, l := range []int{level, level + 1} { // a level between rows adds nothing
				if got := rules.CircleSpells(tables, c.land, character.MustDruidLevel(l)); !slices.Equal(got, want) {
					t.Errorf("CircleSpells(%v, %d) = %v, want %v", c.land, l, got, want)
				}
			}
		}
		if got := rules.CircleSpells(tables, c.land, character.MustDruidLevel(2)); len(got) != 0 {
			t.Errorf("CircleSpells(%v, 2) = %v, want none", c.land, got)
		}
	}
	if _, ok := tables.Resistance(0); ok {
		t.Error("the zero land has a Resistance")
	}
	wantFeatures := map[int][]string{1: {"druid_spellcasting", "druid_druidic", "druid_primal_order"}, 2: {"druid_wild_shape", "druid_wild_companion"}, 3: {"druid_druid_subclass"},
		4: {"druid_ability_score_improvement"}, 5: {"druid_wild_resurgence"}, 7: {"druid_elemental_fury"}, 15: {"druid_improved_elemental_fury"}, 18: {"druid_beast_spells"},
		19: {"druid_epic_boon"}, 20: {"druid_archdruid"}}
	for level := 0; level <= 21; level++ {
		var got []string
		for _, id := range tables.FeaturesAt(level) {
			got = append(got, string(id))
		}
		if !slices.Equal(got, wantFeatures[level]) {
			t.Errorf("FeaturesAt(%d) = %v, want %v", level, got, wantFeatures[level])
		}
	}
}

func splitWords(s string) []string {
	var out []string
	word := ""
	for _, r := range s + " " {
		if r == ' ' {
			if word != "" {
				out = append(out, word)
			}
			word = ""
			continue
		}
		word += string(r)
	}
	return out
}

// Requirement: doc/specs/class-druid.md FR-168 - NewDruidFeatureTables accepts a complete input and lists every problem otherwise as a
// *DruidInvariantError matching ErrInvalidInput: other than 20 rows or a negative row, other than three tiers, tiers not ascending or with no known
// form, other than four distinct lands, a land with an empty or invalid spell row or no damage type, and features that are neither absent nor 20 lists;
// the returned slices are copies; MustDruidFeatureTables panics only for an invalid input.
// Path: per case (happy / unhappy: invalid input, boundary, immutability).
func TestNewDruidFeatureTables(t *testing.T) {
	t.Parallel()
	good := druTableInput(t)
	tables, err := character.NewDruidFeatureTables(good)
	if err != nil || tables.IsZero() {
		t.Fatalf("NewDruidFeatureTables(valid) = %v", err)
	}
	for _, c := range []struct {
		label  string
		mutate func(*character.DruidTablesInput)
		want   []character.DruidInvariantViolation
	}{
		{"19 rows", func(in *character.DruidTablesInput) { in.WildShapeUses = in.WildShapeUses[:19] }, []character.DruidInvariantViolation{character.DruidViolationWildShapeRows}},
		{"negative row", func(in *character.DruidTablesInput) {
			in.WildShapeUses = slices.Clone(in.WildShapeUses)
			in.WildShapeUses[4] = -1
		}, []character.DruidInvariantViolation{character.DruidViolationWildShapeRows}},
		{"two tiers", func(in *character.DruidTablesInput) { in.Tiers = in.Tiers[:2] }, []character.DruidInvariantViolation{character.DruidViolationBeastTiers}},
		{"tiers not ascending", func(in *character.DruidTablesInput) {
			in.Tiers = []character.BeastShapeTier{in.Tiers[1], in.Tiers[0], in.Tiers[2]}
		}, []character.DruidInvariantViolation{character.DruidViolationBeastTiers}},
		{"tier with no form", func(in *character.DruidTablesInput) { in.Tiers = slices.Clone(in.Tiers); in.Tiers[1].KnownForms = 0 }, []character.DruidInvariantViolation{character.DruidViolationBeastTiers}},
		{"tier level 0", func(in *character.DruidTablesInput) { in.Tiers = slices.Clone(in.Tiers); in.Tiers[0].MinLevel = 0 }, []character.DruidInvariantViolation{character.DruidViolationBeastTiers}},
		{"tier level 21", func(in *character.DruidTablesInput) { in.Tiers = slices.Clone(in.Tiers); in.Tiers[2].MinLevel = 21 }, []character.DruidInvariantViolation{character.DruidViolationBeastTiers}},
		{"three lands", func(in *character.DruidTablesInput) { in.Lands = in.Lands[:3] }, []character.DruidInvariantViolation{character.DruidViolationLands}},
		{"repeated land", func(in *character.DruidTablesInput) {
			in.Lands = slices.Clone(in.Lands)
			in.Lands[1].Land = character.LandArid
		}, []character.DruidInvariantViolation{character.DruidViolationLands}},
		{"zero land", func(in *character.DruidTablesInput) { in.Lands = slices.Clone(in.Lands); in.Lands[0].Land = 0 }, []character.DruidInvariantViolation{character.DruidViolationLands}},
		{"no resistance", func(in *character.DruidTablesInput) { in.Lands = slices.Clone(in.Lands); in.Lands[0].Resistance = 0 }, []character.DruidInvariantViolation{character.DruidViolationLands}},
		{"empty spell row", func(in *character.DruidTablesInput) { in.Lands = slices.Clone(in.Lands); in.Lands[2].Spells[1] = nil }, []character.DruidInvariantViolation{character.DruidViolationLands}},
		{"invalid spell id", func(in *character.DruidTablesInput) {
			in.Lands = slices.Clone(in.Lands)
			in.Lands[3].Spells[3] = []content.SpellID{"Not An Id"}
		}, []character.DruidInvariantViolation{character.DruidViolationLands}},
		{"19 feature lists", func(in *character.DruidTablesInput) { in.FeaturesByLevel = make([][]content.FeatureID, 19) }, []character.DruidInvariantViolation{character.DruidViolationFeatures}},
		{"several at once", func(in *character.DruidTablesInput) { in.WildShapeUses, in.Tiers, in.Lands = nil, nil, nil }, []character.DruidInvariantViolation{character.DruidViolationWildShapeRows, character.DruidViolationBeastTiers, character.DruidViolationLands}},
	} {
		in := druTableInput(t)
		c.mutate(&in)
		_, err := character.NewDruidFeatureTables(in)
		var ie *character.DruidInvariantError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ie) || !slices.Equal(ie.Violations, c.want) {
			t.Errorf("%s: error = %v, want violations %v", c.label, err, c.want)
		}
	}
	// the tables copy their input
	good.WildShapeUses[1] = 9
	good.Lands[0].Spells[0][0] = "changed"
	if tables.WildShapeUses(2) != 2 {
		t.Error("the tables share the input rows")
	}
	if got := (character.DruidRules{}).CircleSpells(tables, character.LandArid, character.MustDruidLevel(3)); got[0] != "arid_3a" {
		t.Errorf("CircleSpells = %v after the input changed", got)
	}
	got := tables.BeastShapes()
	got[0].KnownForms = 99
	if tables.BeastShapes()[0].KnownForms != 4 || !(character.DruidFeatureTables{}).IsZero() {
		t.Error("BeastShapes returned shared storage, or the zero tables are not zero")
	}
	spells := character.DruidRules{}.CircleSpells(tables, character.LandArid, character.MustDruidLevel(9))
	spells[0] = "changed"
	if (character.DruidRules{}).CircleSpells(tables, character.LandArid, character.MustDruidLevel(9))[0] != "arid_3a" {
		t.Error("CircleSpells returned shared storage")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustDruidFeatureTables(invalid) did not panic")
		}
	}()
	character.MustDruidFeatureTables(character.DruidTablesInput{})
}

// Requirement: doc/specs/class-druid.md FR-172 - DruidTablesFromClass refuses the definition of another class and every cell it cannot read, never guessing:
// a Challenge Rating that is not one, a Fly Speed that is neither Yes nor No, a table row of the wrong width or a missing table, a land option with no
// always-prepared fact or a Circle Spells grant at a level that is not 3, 5, 7 or 9, a Resistance that is no damage type or a land without one, a missing
// land, and the Wild Shape feature missing.
// Path: per case (unhappy: invalid input, missing content).
func TestDruidTablesFromClassRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()
	wild := func(c map[string]any) map[string]any { return druFeature(t, c, "druid_wild_shape") }
	beastRows := func(c map[string]any) []any { return wild(c)["tables"].([]any)[0].(map[string]any)["rows"].([]any) }
	ward := func(c map[string]any) map[string]any {
		return druFeature(t, c, "circle_of_the_land_natures_ward")["tables"].([]any)[0].(map[string]any)
	}
	lands := func(c map[string]any) []any {
		return druFeature(t, c, "circle_of_the_land_circle_of_the_land_spells")["options"].([]any)
	}
	for _, c := range []struct {
		label  string
		mutate func(map[string]any)
	}{
		{"rating", func(c map[string]any) { beastRows(c)[0].([]any)[2] = "1/3" }},
		{"fly cell", func(c map[string]any) { beastRows(c)[2].([]any)[3] = "Maybe" }},
		{"level cell as text", func(c map[string]any) {
			beastRows(c)[0].([]any)[0] = "two"
			beastRows(c)[1].([]any)[0] = "four"
			beastRows(c)[2].([]any)[0] = "eight"
		}},
		{"forms cell as text", func(c map[string]any) {
			beastRows(c)[0].([]any)[1] = "four"
			beastRows(c)[1].([]any)[1] = "six"
			beastRows(c)[2].([]any)[1] = "eight"
		}},
		{"rating cell as number", func(c map[string]any) {
			beastRows(c)[0].([]any)[2] = 1
			beastRows(c)[1].([]any)[2] = 2
			beastRows(c)[2].([]any)[2] = 3
		}},
		{"fly cell as number", func(c map[string]any) {
			beastRows(c)[0].([]any)[3] = 1
			beastRows(c)[1].([]any)[3] = 1
			beastRows(c)[2].([]any)[3] = 1
		}},
		{"table renamed", func(c map[string]any) { wild(c)["tables"].([]any)[0].(map[string]any)["name"] = "Beast Forms" }},
		{"no table", func(c map[string]any) { delete(wild(c), "tables") }},
		{"ward table renamed", func(c map[string]any) { ward(c)["name"] = "Ward" }},
		{"resistance text", func(c map[string]any) { ward(c)["rows"].([]any)[0].([]any)[1] = "Heat" }},
		{"land name number", func(c map[string]any) {
			for _, r := range ward(c)["rows"].([]any) {
				r.([]any)[0] = 1
				r.([]any)[1] = 1
			}
		}},
		{"land without resistance", func(c map[string]any) { ward(c)["rows"].([]any)[3].([]any)[0] = "Swampy" }},
		{"land removed", func(c map[string]any) {
			druFeature(t, c, "circle_of_the_land_circle_of_the_land_spells")["options"] = lands(c)[:3]
		}},
		{"land without a fact", func(c map[string]any) { delete(lands(c)[0].(map[string]any), "facts") }},
		{"grant at level 4", func(c map[string]any) {
			g := lands(c)[0].(map[string]any)["facts"].([]any)[0].(map[string]any)["grants"].([]any)[1].(map[string]any)
			g["level"] = 4
		}},
		{"grant at level 11", func(c map[string]any) {
			g := lands(c)[0].(map[string]any)["facts"].([]any)[0].(map[string]any)["grants"].([]any)[3].(map[string]any)
			g["level"] = 11
		}},
		{"land id unknown", func(c map[string]any) { lands(c)[0].(map[string]any)["id"] = "swamp" }},
		{"no subclass", func(c map[string]any) { c["subclasses"] = []any{} }},
		{"spell grant row missing", func(c map[string]any) {
			f := lands(c)[0].(map[string]any)["facts"].([]any)[0].(map[string]any)
			f["grants"] = f["grants"].([]any)[:3]
		}},
	} {
		def := druDefFrom(t, c.mutate)
		_, err := character.DruidTablesFromClass(def)
		var ie *character.DruidInvariantError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ie) {
			t.Errorf("%s: error = %v, want a *DruidInvariantError", c.label, err)
		}
	}
	_, err := character.DruidTablesFromClass(clcDef(t))
	wantInvalid(t, err, "Cleric definition")
	_, err = character.DruidTablesFromClass(content.ClassDefinition{})
	wantInvalid(t, err, "zero definition")
}

// Requirement: doc/specs/class-druid.md FR-142 - DruidInvariantError names every violation in its text, matches ErrInvalidInput and lists each violation once in the order of the enum.
// Path: unhappy (invalid input).
func TestDruidInvariantErrorListsEveryViolationOnce(t *testing.T) {
	t.Parallel()
	_, err := character.NewDruidFeatureTables(character.DruidTablesInput{})
	var ie *character.DruidInvariantError
	if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) || len(ie.Violations) != 3 ||
		ie.Error() != "character: invalid Druid value: wild_shape_rows, beast_tiers, lands" {
		t.Errorf("error = %v", err)
	}
}

// Requirement: doc/specs/class-druid.md FR-168 (table boundaries) - a Beast Shapes tier may start at level 20, level 1 reads the first Wild Shape
// row and the last violation name is not "unknown".
// Path: boundary (tier level 20, level 1, last violation value).
func TestDruidTablesEdges(t *testing.T) {
	t.Parallel()
	in := druTableInput(t)
	in.Tiers = slices.Clone(in.Tiers)
	in.Tiers[2].MinLevel = 20
	in.WildShapeUses = slices.Clone(in.WildShapeUses)
	in.WildShapeUses[0] = 1
	tables, err := character.NewDruidFeatureTables(in)
	if err != nil {
		t.Fatalf("a tier at level 20 and a level 1 row of 1 were refused: %v", err)
	}
	if got := tables.WildShapeUses(1); got != 1 {
		t.Errorf("WildShapeUses(1) = %d, want 1", got)
	}
	if got := character.DruidViolationUndefinedValue.String(); got != "undefined_value" {
		t.Errorf("last violation = %q", got)
	}
	if got := (character.DruidViolationUndefinedValue + 1).String(); got != "unknown" {
		t.Errorf("past the last violation = %q", got)
	}
}
