package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// wantCasting reads a printed Casting Time into its model value and Ritual flag.
func wantCasting(t *testing.T, printed string) (content.SpellCastingTimes, bool) {
	t.Helper()
	var ct content.CastingTime
	switch strings.TrimSuffix(printed, " or Ritual") {
	case "Action":
		ct = content.NewCastingTimeMagicAction()
	case "Bonus Action":
		ct = content.NewCastingTimeBonusAction()
	default:
		t.Fatalf("unhandled printed Casting Time %q", printed)
	}
	single, err := content.NewSpellCastingTimesSingle(ct)
	if err != nil {
		t.Fatal(err)
	}
	return single, strings.HasSuffix(printed, " or Ritual")
}

// wantRange reads a printed Range into its model value.
func wantRange(t *testing.T, printed string) content.SpellRange {
	t.Helper()
	switch printed {
	case "Self":
		return content.NewSpellRangeSelf()
	case "Touch":
		return content.NewSpellRangeTouch()
	}
	amount, unit, ok := strings.Cut(printed, " ")
	n, err := strconv.Atoi(amount)
	if !ok || err != nil || unit != "feet" {
		t.Fatalf("unhandled printed Range %q", printed)
	}
	r, err := content.NewSpellRangeDistance(n, content.DistanceFeet)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// wantDuration reads a printed Duration into its model value: Instantaneous,
// "Concentration, up to N unit", "Up to N unit" or "N unit".
func wantDuration(t *testing.T, printed string) content.SpellDuration {
	t.Helper()
	if printed == "Instantaneous" {
		return content.NewSpellDurationInstantaneous()
	}
	limit := content.LimitFixed
	rest := printed
	if r, ok := strings.CutPrefix(printed, "Concentration, up to "); ok {
		limit, rest = content.LimitConcentrationUpTo, r
	} else if r, ok := strings.CutPrefix(printed, "Up to "); ok {
		limit, rest = content.LimitUpTo, r
	}
	amount, unit, _ := strings.Cut(rest, " ")
	n, err := strconv.Atoi(amount)
	if err != nil {
		t.Fatalf("unhandled printed Duration %q", printed)
	}
	var u content.DurationUnit
	switch strings.TrimSuffix(unit, "s") {
	case "minute":
		u = content.DurationUnitMinutes
	case "hour":
		u = content.DurationUnitHours
	default:
		t.Fatalf("unhandled printed Duration unit in %q", printed)
	}
	d, err := content.NewSpellDurationTimed(n, u, limit)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Requirement: doc/specs/content-thin-slice.md FR-048, FR-050, FR-051, FR-052, FR-053, FR-061, T6 - each of the 12
// spells has the level, school, class set, Casting Time (with the Ritual flag), Range, Components (with the Material
// text) and Duration of its T6 row, and the class set sizes are 1, 2, 3, 4 and 8, none empty.
// Path: per case (T6): happy / unhappy (boundary: lists of 1, 2, 3, 4 and 8 classes; Self versus feet; optional Material).
func TestSpellHeaders(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	sizes := map[int]bool{}
	for _, o := range readOracle[spellOracleFile](t, "spells").Spells {
		sizes[len(o.Classes)] = true
		t.Run(o.ID, func(t *testing.T) {
			t.Parallel()
			s := spellOf(t, src, o.ID)
			school, err := content.ParseSpellSchool(o.School)
			if err != nil {
				t.Fatal(err)
			}
			if s.Name() != o.Name || int(s.Level()) != o.Level || s.School() != school {
				t.Errorf("%s = %q, level %d, %v; want %q, %d, %v", o.ID, s.Name(), s.Level(), s.School(), o.Name, o.Level, school)
			}
			var wantClasses []string
			for _, c := range o.Classes {
				wantClasses = append(wantClasses, strings.ToLower(c))
			}
			var gotClasses []string
			for _, c := range s.Classes() {
				gotClasses = append(gotClasses, string(c))
			}
			slices.Sort(wantClasses)
			slices.Sort(gotClasses)
			if !slices.Equal(gotClasses, wantClasses) || len(gotClasses) == 0 {
				t.Errorf("%s classes = %v, want %v", o.ID, gotClasses, wantClasses)
			}
			wantTimes, wantRitual := wantCasting(t, o.CastingTime)
			if s.CastingTimes() != wantTimes || s.Ritual() != wantRitual {
				t.Errorf("%s casting = %v ritual %v, want %v ritual %v", o.ID, s.CastingTimes(), s.Ritual(), wantTimes, wantRitual)
			}
			if got, want := s.Range(), wantRange(t, o.Range); got != want {
				t.Errorf("%s range = %v, want %v", o.ID, got, want)
			}
			if got, want := s.Duration(), wantDuration(t, o.Duration); got != want {
				t.Errorf("%s duration = %v, want %v", o.ID, got, want)
			}
			wantConcentration := strings.HasPrefix(o.Duration, "Concentration")
			if s.RequiresConcentration() != wantConcentration {
				t.Errorf("%s RequiresConcentration = %v, want %v", o.ID, s.RequiresConcentration(), wantConcentration)
			}
			letters, material, hasMaterial := strings.Cut(o.Components, " (")
			c := s.Components()
			gotMaterial, gotHas := c.Material()
			if c.Verbal() != strings.Contains(letters, "V") || c.Somatic() != strings.Contains(letters, "S") || gotHas != strings.Contains(letters, "M") || gotHas != hasMaterial {
				t.Errorf("%s components = V %v S %v M %v, want %q", o.ID, c.Verbal(), c.Somatic(), gotHas, o.Components)
			}
			if gotHas && (gotMaterial.Description() != strings.TrimSuffix(material, ")") || len(gotMaterial.CostsGP()) != 0 || gotMaterial.ConsumesMaterial()) {
				t.Errorf("%s material = %q, %v, %v; want %q with no cost", o.ID, gotMaterial.Description(), gotMaterial.CostsGP(), gotMaterial.ConsumesMaterial(), strings.TrimSuffix(material, ")"))
			}
		})
	}
	if want := []int{1, 2, 3, 4, 8}; !sizes[1] || !sizes[2] || !sizes[3] || !sizes[4] || !sizes[8] || len(sizes) != len(want) {
		t.Errorf("class set sizes = %v, want exactly 1, 2, 3, 4 and 8", sizes)
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-050 - only Detect Magic carries the Ritual tag; Fire Bolt has the
// Magic action and no Ritual, Misty Step the Bonus Action and no Ritual.
// Path: unhappy (rule interaction: action plus Ritual).
func TestRitualOnlyOnDetectMagic(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, id := range sliceIDs()[content.KindSpell] {
		if got, want := spellOf(t, src, id).Ritual(), id == "detect_magic"; got != want {
			t.Errorf("%s Ritual = %v, want %v", id, got, want)
		}
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-054 to FR-059, FR-083 - every spell description contains the phrases
// picked from its page, the named effects of Prestidigitation (six) and Druidcraft (four) are entries, and Misty
// Step names no attack roll, saving throw or damage.
// Path: happy / unhappy (a spell with no attack, save or damage).
func TestSpellDescriptions(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, o := range readOracle[spellOracleFile](t, "spells").Spells {
		t.Run(o.ID, func(t *testing.T) {
			t.Parallel()
			s := spellOf(t, src, o.ID)
			text := strings.Join(blockTexts(t, s.Description()), "\n")
			for _, p := range o.Phrases {
				if !strings.Contains(text, p) {
					t.Errorf("%s description lacks %q", o.ID, p)
				}
			}
			for _, a := range o.Absent {
				if strings.Contains(strings.ToLower(text), a) {
					t.Errorf("%s description names %q", o.ID, a)
				}
			}
			var entries []string
			for _, b := range s.Description() {
				if e, ok := b.(content.TextEntry); ok {
					entries = append(entries, e.Name())
				}
			}
			if !slices.Equal(entries, o.Entries) {
				t.Errorf("%s run-in entries = %v, want %v", o.ID, entries, o.Entries)
			}
		})
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-060, T7 and content-spells.md FR-115 - scaling is text only: Fire
// Bolt has cantripUpgrade and no higherLevel; Magic Missile, Fireball and Longstrider have higherLevel and no
// cantripUpgrade; every other slice spell has neither, absent and not empty.
// Path: per case (T7): happy / unhappy (empty case).
func TestSpellScaling(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, o := range readOracle[spellOracleFile](t, "spells").Spells {
		t.Run(o.ID, func(t *testing.T) {
			t.Parallel()
			s := spellOf(t, src, o.ID)
			up, hl := blockTexts(t, s.CantripUpgrade()), blockTexts(t, s.HigherLevel())
			var wantUp, wantHL []string
			switch o.Scaling {
			case "cantripUpgrade":
				wantUp = []string{o.ScalingPhrase}
			case "higherLevel":
				wantHL = []string{o.ScalingPhrase}
			}
			if len(up) != len(wantUp) || len(hl) != len(wantHL) ||
				(len(wantUp) == 1 && !strings.Contains(up[0], wantUp[0])) || (len(wantHL) == 1 && !strings.Contains(hl[0], wantHL[0])) {
				t.Errorf("%s scaling: cantripUpgrade %q, higherLevel %q; want %q and %q", o.ID, up, hl, wantUp, wantHL)
			}
		})
	}
}

// Requirement: doc/specs/content-thin-slice.md T7 (fire bolt thresholds match breath weapon) - the cantrip upgrade
// thresholds 5, 11 and 17 equal those of the Breath Weapon table T2.
// Path: unhappy (rule interaction: one scaling shape for two kinds).
func TestFireBoltThresholdsMatchBreathWeapon(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	up := strings.Join(blockTexts(t, spellOf(t, src, "fire_bolt").CantripUpgrade()), " ")
	breath := traitText(t, speciesOf(t, src, "dragonborn"), "Breath Weapon")
	for _, r := range readOracle[speciesOracleFile](t, "species").BreathWeaponDamage[1:] {
		want := describe("%d (%s)", r.Level, r.Dice)
		if !strings.Contains(up, want) || !strings.Contains(breath, want) {
			t.Errorf("threshold %q is missing from the Fire Bolt upgrade or the Breath Weapon text", want)
		}
	}
}
