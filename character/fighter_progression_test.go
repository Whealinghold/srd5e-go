package character_test

import (
	"errors"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// fgtBand is one band of Fighter levels of Table A where every column is constant.
type fgtBand struct {
	from, to                                                     int
	secondWind, weaponMastery, actionSurge, indomitable, attacks int
}

// fgtTableA is Table A of the spec, typed from the Fighter Features table (SRD 5.2.1, p. 47) and the feature
// headings (p. 48), independent of content/data.
func fgtTableA() []fgtBand {
	return []fgtBand{
		{1, 1, 2, 3, 0, 0, 1}, {2, 3, 2, 3, 1, 0, 1}, {4, 4, 3, 4, 1, 0, 1}, {5, 8, 3, 4, 1, 0, 2}, {9, 9, 3, 4, 1, 1, 2},
		{10, 10, 4, 5, 1, 1, 2}, {11, 12, 4, 5, 1, 1, 3}, {13, 15, 4, 5, 1, 2, 3}, {16, 16, 4, 6, 1, 2, 3}, {17, 19, 4, 6, 2, 3, 3}, {20, 20, 4, 6, 2, 3, 4},
	}
}

// Requirement: doc/specs/class-fighter.md Table A, FR-022, FR-023, FR-024, FR-025, FR-026 and FR-027 - at every Fighter level 1 to 20
// the stats equal the band row: Second Wind uses, Weapon Mastery kinds, Action Surge uses (0, 1, 2), Indomitable uses (0 to 3)
// and attacks per Attack action (1 to 4), from the Fighter level alone.
// Path: per case (the 20 levels).
func TestFighterProgressionStatsAtEveryLevel(t *testing.T) {
	t.Parallel()
	p := fgtProgression(t)
	for _, b := range fgtTableA() {
		for n := b.from; n <= b.to; n++ {
			t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
				t.Parallel()
				s := p.StatsAt(character.MustFighterLevel(n))
				got := fgtBand{n, n, s.SecondWind(), s.WeaponMastery(), s.ActionSurge(), s.Indomitable(), s.Attacks()}
				if want := (fgtBand{n, n, b.secondWind, b.weaponMastery, b.actionSurge, b.indomitable, b.attacks}); got != want || s.Level().Value() != n || s.IsZero() {
					t.Errorf("StatsAt(%d) = %+v, want %+v", n, got, want)
				}
			})
		}
	}
}

// Requirement: doc/specs/class-fighter.md FR-021, FR-114 - the Proficiency Bonus of each decoded Fighter row equals the rule for a
// total level equal to that Fighter level, and FighterLevelStats exposes none (it follows the total level: a Fighter 3 / Rogue 2
// has +3, not the +2 of row L02).
// Path: unhappy (rule interaction).
func TestFighterProficiencyBonusFollowsTotalLevel(t *testing.T) {
	t.Parallel()
	def := fgtDef(t)
	for n := 1; n <= 20; n++ {
		row, err := def.RowAt(n)
		if err != nil || row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() {
			t.Errorf("row %d bonus = %d (%v), want %d", n, row.ProficiencyBonus(), err, character.ProficiencyBonusFor(character.MustLevel(n)).Value())
		}
	}
	multi := fgtChar(t, cl("fighter", 3), cl("rogue", 2))
	if got := multi.ProficiencyBonus().Value(); got != 3 {
		t.Errorf("Fighter 3 / Rogue 2 Proficiency Bonus = %d, want 3 (total level 5)", got)
	}
}

// Requirement: doc/specs/class-fighter.md FR-150 - the Action Surge and Indomitable constants equal the counts annotated on the
// Class Features cells of the decoded rows ("one use", "two uses", "three uses") at every level, and the attacks equal the
// count the Extra Attack features give; a mismatch is a test failure, not a run-time error.
// Path: unhappy (rule interaction).
func TestFighterConstantsEqualRowAnnotations(t *testing.T) {
	t.Parallel()
	def := fgtDef(t)
	p := fgtProgression(t)
	words := map[string]int{"one use": 1, "two uses": 2, "three uses": 3}
	surge, indomitable, attacks := 0, 0, 1
	for n := 1; n <= 20; n++ {
		row, _ := def.RowAt(n)
		for _, f := range row.Features() {
			rf, ok := f.(content.RowClassFeature)
			if !ok {
				continue
			}
			switch rf.FeatureID() {
			case "fighter_action_surge":
				surge = words[rf.Annotation()]
			case "fighter_indomitable":
				indomitable = words[rf.Annotation()]
			case "fighter_extra_attack":
				attacks = 2
			case "fighter_two_extra_attacks":
				attacks = 3
			case "fighter_three_extra_attacks":
				attacks = 4
			}
		}
		s := p.StatsAt(character.MustFighterLevel(n))
		if s.ActionSurge() != surge || s.Indomitable() != indomitable || s.Attacks() != attacks {
			t.Errorf("level %d: surge %d, indomitable %d, attacks %d; annotated %d, %d, %d", n, s.ActionSurge(), s.Indomitable(), s.Attacks(), surge, indomitable, attacks)
		}
	}
}

// fgtBundledText is the text of an embedded document.
func fgtBundledText(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(data.Files(), name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fgtDecode(t *testing.T, text string) content.ClassDefinition {
	t.Helper()
	set, err := content.JSONDecoder{}.Decode([]byte(text))
	if err != nil {
		t.Fatalf("Decode: %v", content.Problems(err))
	}
	return set.Entries()[0].(content.ClassDefinition)
}

// Requirement: doc/specs/class-fighter.md FR-149, FR-160 - NewFighterProgression accepts the Fighter definition and joins a typed
// reason for another class (NotTheFighterDefinition), the zero definition, a count column that is missing or has the wrong kind
// (FighterColumnMissing) and a level whose cell holds no count (FighterRowsIncomplete); each matches srderr.ErrInvalidInput and the
// Must helper panics with the same error.
// Path: per case (unhappy: missing content).
func TestNewFighterProgression(t *testing.T) {
	t.Parallel()
	fighter := fgtBundledText(t, "class_fighter.json")
	wizard := fgtDecode(t, fgtBundledText(t, "class_wizard.json"))
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		want character.FighterProgressionReason
	}{
		{"another class", wizard, character.NotTheFighterDefinition},
		{"the zero definition", content.ClassDefinition{}, character.NotTheFighterDefinition},
		{"a renamed column", fgtDecode(t, strings.ReplaceAll(fighter, "weapon_mastery", "weapon_masteries")), character.FighterColumnMissing},
		{"a column of another kind", fgtDecode(t, strings.Replace(fighter, `"kind": "count"`, `"kind": "bonus"`, 1)), character.FighterColumnMissing},
		{"a cell with no count", fgtDecode(t, strings.Replace(fighter, `"second_wind": 2`, `"second_wind": "—"`, 1)), character.FighterRowsIncomplete},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := character.NewFighterProgression(c.def)
			wantInvalid(t, err, c.name)
			var pe *character.FighterProgressionError
			if !p.IsZero() || !errors.As(err, &pe) || pe.Reason != c.want || !errors.Is(err, &character.FighterProgressionError{Reason: c.want}) {
				t.Errorf("NewFighterProgression = %v, %v; want the zero progression and reason %v", p, err, c.want)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("MustFighterProgression did not panic")
				}
			}()
			character.MustFighterProgression(c.def)
		})
	}
	p, err := character.NewFighterProgression(fgtDecode(t, fighter))
	if err != nil || p.IsZero() || p.StatsAt(character.MustFighterLevel(10)).SecondWind() != 4 {
		t.Errorf("NewFighterProgression(Fighter) = %v, %v", p, err)
	}
}

// Requirement: doc/specs/class-fighter.md FR-149 - a progression whose Second Wind column changes (a definition built with other counts)
// changes the stats; the zero progression and the zero level give the zero stats.
// Path: unhappy (boundary).
func TestFighterProgressionReadsColumnsAndZeroValues(t *testing.T) {
	t.Parallel()
	changed := strings.ReplaceAll(fgtBundledText(t, "class_fighter.json"), `"second_wind": 2`, `"second_wind": 5`)
	p := character.MustFighterProgression(fgtDecode(t, changed))
	if got := p.StatsAt(character.MustFighterLevel(1)).SecondWind(); got != 5 {
		t.Errorf("Second Wind uses at level 1 = %d, want 5 from the changed column", got)
	}
	if !(character.FighterProgression{}).StatsAt(character.MustFighterLevel(1)).IsZero() || !fgtProgression(t).StatsAt(character.FighterLevel{}).IsZero() {
		t.Error("the zero progression or level must give the zero stats")
	}
}

// Requirement: doc/specs/class-fighter.md Table B, FR-028, FR-115 - the choices due on gaining a level: level 1 the Fighting Style and
// 3 Weapon Mastery kinds, 4, 10 and 16 one more kind each, 7 the Additional Fighting Style for the Champion only (C07a, C07b), every
// other level none, an empty slice rather than nil.
// Path: per case (the 20 levels and both subclass inputs).
func TestFighterChoicesOnGaining(t *testing.T) {
	t.Parallel()
	p := fgtProgression(t)
	style, additional := character.FighterChoice(character.FighterChoiceFightingStyle{}), character.FighterChoice(character.FighterChoiceAdditionalFightingStyle{})
	masteries := func(n int) character.FighterChoice { return character.FighterChoiceWeaponMasteries{Count: n} }
	for n := 1; n <= 20; n++ {
		for _, sub := range []content.SubclassID{"", character.ChampionSubclassID} {
			t.Run("level "+strconv.Itoa(n)+" subclass "+string(sub), func(t *testing.T) {
				t.Parallel()
				var want []character.FighterChoice
				switch n {
				case 1:
					want = []character.FighterChoice{style, masteries(3)}
				case 4, 10, 16:
					want = []character.FighterChoice{masteries(1)}
				case 7:
					if sub == character.ChampionSubclassID {
						want = []character.FighterChoice{additional}
					}
				}
				got := p.ChoicesOnGaining(character.MustFighterLevel(n), sub)
				if got == nil || !slices.Equal(got, append([]character.FighterChoice{}, want...)) {
					t.Errorf("ChoicesOnGaining(%d, %q) = %v, want %v", n, sub, got, want)
				}
			})
		}
	}
	if got := (character.FighterProgression{}).ChoicesOnGaining(character.MustFighterLevel(1), ""); len(got) != 0 {
		t.Errorf("zero progression choices = %v, want none", got)
	}
	if got := p.ChoicesOnGaining(character.FighterLevel{}, ""); len(got) != 0 {
		t.Errorf("zero level choices = %v, want none", got)
	}
	if got := p.ChoicesOnGaining(character.MustFighterLevel(7), "another_subclass"); len(got) != 0 {
		t.Errorf("another subclass at level 7 = %v, want none", got)
	}
	if got := p.ChoicesOnGaining(character.MustFighterLevel(7), "warrior"); len(got) != 0 {
		t.Errorf("a subclass sorting after the Champion at level 7 = %v, want none", got)
	}
}

// Requirement: doc/specs/class-fighter.md FR-160 - the reasons and the error of a refused definition have their names and match by reason.
// Path: unhappy (invalid input).
func TestFighterProgressionReasonNames(t *testing.T) {
	t.Parallel()
	for r, name := range map[character.FighterProgressionReason]string{
		character.NotTheFighterDefinition: "not_the_fighter_definition", character.FighterColumnMissing: "column_missing",
		character.FighterRowsIncomplete: "rows_incomplete", 0: "unknown",
	} {
		if r.String() != name {
			t.Errorf("reason %d = %q, want %q", int(r), r.String(), name)
		}
	}
	e := &character.FighterProgressionError{Reason: character.FighterColumnMissing}
	if errors.Is(&character.FighterProgressionError{Reason: character.FighterRowsIncomplete}, e) {
		t.Error("a rows-incomplete error matched a column-missing target")
	}
	if e.Error() == "" || errors.Is(e, &character.FighterProgressionError{Reason: character.FighterRowsIncomplete}) || errors.Is(e, errors.New("other")) {
		t.Errorf("error %v matched the wrong target", e)
	}
}
