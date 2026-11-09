package character_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// brbBand is one band of Barbarian levels where every column is constant.
type brbBand struct{ from, to, rages, rageDamage, weaponMastery int }

// brbTable is the Barbarian Features table (SRD 5.2.1, p. 28) typed by hand from the printed rows, independent of content/data.
func brbTable() []brbBand {
	return []brbBand{
		{1, 2, 2, 2, 2}, {3, 3, 3, 2, 2}, {4, 5, 3, 2, 3}, {6, 8, 4, 2, 3}, {9, 9, 4, 3, 3}, {10, 11, 4, 3, 4},
		{12, 15, 5, 3, 4}, {16, 16, 5, 4, 4}, {17, 20, 6, 4, 4},
	}
}

// Requirement: doc/specs/class-barbarian.md section 4.2.1, FR-020, FR-041 and FR-108 - at every Barbarian level 1 to 20 the stats equal
// the printed row: Rages, Rage Damage and Weapon Mastery kinds; boundary pairs Rage Damage 8/9 and 15/16, Weapon Mastery 3/4 and 9/10, Rages 2/3,
// 5/6, 11/12 and 16/17.
// Path: per case (the 20 levels): happy for levels 1 and 20, unhappy (boundary) elsewhere.
func TestBarbarianProgressionStatsAtEveryLevel(t *testing.T) {
	t.Parallel()
	p := brbProgression(t)
	for _, b := range brbTable() {
		for n := b.from; n <= b.to; n++ {
			t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
				t.Parallel()
				s := p.StatsAt(character.MustBarbarianLevel(n))
				got := brbBand{n, n, s.Rages(), s.RageDamage(), s.WeaponMastery()}
				if want := (brbBand{n, n, b.rages, b.rageDamage, b.weaponMastery}); got != want || s.Level().Value() != n || s.IsZero() {
					t.Errorf("StatsAt(%d) = %+v, want %+v", n, got, want)
				}
			})
		}
	}
}

// Requirement: doc/specs/class-barbarian.md FR-020, FR-054, FR-122 and the Berserker table - the feature levels the rules read are the printed levels of the
// bundled definition (the constants of the rules equal the entries), and the Path of the Berserker holds Frenzy from 3, Mindless Rage from 6, Retaliation from 10
// and Intimidating Presence from 14, in level order, for that subclass only.
// Path: per case (unhappy: boundary, wrong subclass).
func TestBerserkerFeaturesAt(t *testing.T) {
	t.Parallel()
	def := brbDef(t)
	found := def.Subclass(character.BerserkerSubclassID).(content.SubclassFound).Subclass()
	var ids []content.FeatureID
	for _, f := range found.Features() {
		ids = append(ids, f.ID())
	}
	for _, c := range []struct {
		level int
		sub   content.SubclassID
		want  int
	}{
		{1, character.BerserkerSubclassID, 0}, {2, character.BerserkerSubclassID, 0}, {3, character.BerserkerSubclassID, 1}, {5, character.BerserkerSubclassID, 1},
		{6, character.BerserkerSubclassID, 2}, {9, character.BerserkerSubclassID, 2}, {10, character.BerserkerSubclassID, 3}, {13, character.BerserkerSubclassID, 3},
		{14, character.BerserkerSubclassID, 4}, {20, character.BerserkerSubclassID, 4}, {20, "", 0}, {20, "wild_heart", 0},
	} {
		got := character.BerserkerFeaturesAt(character.MustBarbarianLevel(c.level), c.sub)
		if got == nil || !slices.Equal(got, append([]content.FeatureID{}, ids[:c.want]...)) {
			t.Errorf("BerserkerFeaturesAt(%d, %q) = %v, want the first %d of %v", c.level, c.sub, got, c.want, ids)
		}
	}
	if got := character.BerserkerFeaturesAt(character.BarbarianLevel{}, character.BerserkerSubclassID); len(got) != 0 {
		t.Errorf("zero level features = %v, want none", got)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-051, FR-046, FR-100 - the choices due on gaining a level: level 1 two Weapon Mastery kinds, 3 the Primal
// Knowledge skill, 4 and 10 one more kind each, every other level none, an empty slice rather than nil.
// Path: per case (the 20 levels).
func TestBarbarianChoicesOnGaining(t *testing.T) {
	t.Parallel()
	p := brbProgression(t)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			want := []character.BarbarianChoice{}
			switch n {
			case 1:
				want = append(want, character.BarbarianChoiceWeaponMasteries{Count: 2})
			case 3:
				want = append(want, character.BarbarianChoicePrimalKnowledgeSkill{})
			case 4, 10:
				want = append(want, character.BarbarianChoiceWeaponMasteries{Count: 1})
			}
			if got := p.ChoicesOnGaining(character.MustBarbarianLevel(n)); got == nil || !slices.Equal(got, want) {
				t.Errorf("ChoicesOnGaining(%d) = %v, want %v", n, got, want)
			}
		})
	}
	if got := (character.BarbarianProgression{}).ChoicesOnGaining(character.MustBarbarianLevel(1)); got == nil || len(got) != 0 {
		t.Errorf("zero progression choices = %v, want an empty slice", got)
	}
	if got := p.ChoicesOnGaining(character.BarbarianLevel{}); got == nil || len(got) != 0 {
		t.Errorf("zero level choices = %v, want an empty slice", got)
	}
}

// brbMoveFastMovement returns the Barbarian document with Fast Movement moved from level 5 to level 6, row reference included.
func brbMoveFastMovement(doc string) string {
	doc = strings.Replace(doc, "\"featureId\": \"barbarian_extra_attack\"\n            },\n            {\n              \"featureId\": \"barbarian_fast_movement\"\n            }", "\"featureId\": \"barbarian_extra_attack\"\n            }", 1)
	doc = strings.Replace(doc, "\"name\": \"Fast Movement\",\n          \"level\": 5", "\"name\": \"Fast Movement\",\n          \"level\": 6", 1)
	return strings.Replace(doc, "\"kind\": \"subclassFeature\"\n            }", "\"kind\": \"subclassFeature\"\n            },\n            {\n              \"featureId\": \"barbarian_fast_movement\"\n            }", 1)
}

// Requirement: doc/specs/class-barbarian.md FR-108 - NewBarbarianProgression accepts the Barbarian definition and house-rule Rages values, and joins a typed
// reason for each discrepancy: another class or the zero definition (NotTheBarbarianDefinition), a column that is missing or of another kind
// (BarbarianColumnMissing), a level whose cell holds no value (BarbarianRowsIncomplete), a feature the rules read that is missing or misplaced
// (BarbarianFeatureMissing) and a missing or misplaced Path of the Berserker feature (BarbarianSubclassMissing); each matches srderr.ErrInvalidInput and
// the Must helper panics with the same error. No SRD value is compared.
// Path: per case (unhappy: missing content).
func TestNewBarbarianProgression(t *testing.T) {
	t.Parallel()
	barbarian := fgtBundledText(t, "class_barbarian.json")
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		want character.BarbarianProgressionReason
	}{
		{"another class", fgtDecode(t, fgtBundledText(t, "class_wizard.json")), character.NotTheBarbarianDefinition},
		{"the zero definition", content.ClassDefinition{}, character.NotTheBarbarianDefinition},
		{"a renamed column", fgtDecode(t, strings.ReplaceAll(barbarian, `"rages"`, `"fury"`)), character.BarbarianColumnMissing},
		{"a column of another kind", fgtDecode(t, strings.Replace(barbarian, `"id": "rage_damage",
          "header": "Rage Damage",
          "kind": "bonus"`, `"id": "rage_damage",
          "header": "Rage Damage",
          "kind": "count"`, 1)), character.BarbarianColumnMissing},
		{"a cell with no value", fgtDecode(t, strings.Replace(barbarian, `"rages": 2`, `"rages": "—"`, 1)), character.BarbarianRowsIncomplete},
		{"a feature renamed", fgtDecode(t, strings.ReplaceAll(barbarian, `"barbarian_relentless_rage"`, `"barbarian_relentless_fury"`)), character.BarbarianFeatureMissing},
		{"a feature at another level", fgtDecode(t, brbMoveFastMovement(barbarian)), character.BarbarianFeatureMissing},
		{"a Berserker feature renamed", fgtDecode(t, strings.ReplaceAll(barbarian, `"path_of_the_berserker_frenzy"`, `"path_of_the_berserker_fury"`)), character.BarbarianSubclassMissing},
		{"another subclass", fgtDecode(t, strings.ReplaceAll(barbarian, "path_of_the_berserker", "path_of_the_wild")), character.BarbarianSubclassMissing},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := character.NewBarbarianProgression(c.def)
			wantInvalid(t, err, c.name)
			var pe *character.BarbarianProgressionError
			if !p.IsZero() || !errors.As(err, &pe) || !errors.Is(err, &character.BarbarianProgressionError{Reason: c.want}) {
				t.Errorf("NewBarbarianProgression = %v, %v; want the zero progression and reason %v", p, err, c.want)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustBarbarianProgression did not panic")
				}
			}()
			character.MustBarbarianProgression(c.def)
		})
	}
	houseRule := strings.Replace(barbarian, `"rages": 2`, `"rages": 9`, 1)
	p, err := character.NewBarbarianProgression(fgtDecode(t, houseRule))
	if err != nil || p.StatsAt(character.MustBarbarianLevel(1)).Rages() != 9 || p.StatsAt(character.MustBarbarianLevel(10)).WeaponMastery() != 4 {
		t.Errorf("house-rule Rages: %v, %v", p, err)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-108 - one failure lists every problem at once: several discrepancies give several joined errors.
// Path: unhappy (missing or bad content).
func TestNewBarbarianProgressionListsEveryProblem(t *testing.T) {
	t.Parallel()
	broken := strings.ReplaceAll(strings.ReplaceAll(fgtBundledText(t, "class_barbarian.json"), `"rages"`, `"fury"`), `"barbarian_relentless_rage"`, `"barbarian_x"`)
	_, err := character.NewBarbarianProgression(fgtDecode(t, broken))
	if !errors.Is(err, &character.BarbarianProgressionError{Reason: character.BarbarianColumnMissing}) || !errors.Is(err, &character.BarbarianProgressionError{Reason: character.BarbarianFeatureMissing}) {
		t.Errorf("error %v lacks the column or the feature problem", err)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-108 - the progression reads its columns: changed Weapon Mastery values change the stats, and the zero
// progression and the zero level give the zero stats.
// Path: unhappy (boundary).
func TestBarbarianProgressionReadsColumnsAndZeroValues(t *testing.T) {
	t.Parallel()
	if !(character.BarbarianProgression{}).StatsAt(character.MustBarbarianLevel(1)).IsZero() || !brbProgression(t).StatsAt(character.BarbarianLevel{}).IsZero() {
		t.Error("the zero progression or level must give the zero stats")
	}
	if !(character.BarbarianProgression{}).IsZero() || brbProgression(t).IsZero() {
		t.Error("IsZero must tell the zero progression apart")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-108 - the reasons and the error of a refused definition have their names and match by reason.
// Path: unhappy (invalid input).
func TestBarbarianProgressionReasonNames(t *testing.T) {
	t.Parallel()
	for r, name := range map[character.BarbarianProgressionReason]string{
		character.NotTheBarbarianDefinition: "not_the_barbarian_definition", character.BarbarianColumnMissing: "column_missing",
		character.BarbarianRowsIncomplete: "rows_incomplete", character.BarbarianFeatureMissing: "feature_missing",
		character.BarbarianSubclassMissing: "subclass_missing", 0: "unknown",
	} {
		if r.String() != name {
			t.Errorf("reason %d = %q, want %q", int(r), r.String(), name)
		}
	}
	e := &character.BarbarianProgressionError{Reason: character.BarbarianColumnMissing}
	if e.Error() == "" || errors.Is(e, &character.BarbarianProgressionError{Reason: character.BarbarianRowsIncomplete}) || errors.Is(e, errors.New("other")) || !errors.Is(e, e) {
		t.Errorf("error %v matched the wrong target", e)
	}
}
