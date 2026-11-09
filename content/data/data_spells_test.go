package data_test

import (
	"context"
	"encoding/json"
	"regexp"
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
		if n, unit, ok := strings.Cut(strings.TrimSuffix(printed, " or Ritual"), " "); ok && (strings.HasPrefix(unit, "minute") || strings.HasPrefix(unit, "hour")) {
			amount, err := strconv.Atoi(n)
			if err != nil {
				t.Fatalf("unhandled printed Casting Time %q", printed)
			}
			if strings.HasPrefix(unit, "minute") {
				ct, err = content.NewCastingTimeMinutes(amount)
			} else {
				ct, err = content.NewCastingTimeHours(amount)
			}
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if trigger, ok := strings.CutPrefix(printed, "Bonus Action, which you take "); ok {
			bt, err := content.NewCastingTimeBonusActionTriggered(trigger)
			if err != nil {
				t.Fatal(err)
			}
			single, err := content.NewSpellCastingTimesSingle(bt)
			if err != nil {
				t.Fatal(err)
			}
			return single, false
		}
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
	if r, ok := strings.CutPrefix(printed, "Concentration, up to "); ok || strings.HasPrefix(printed, "Concentration up to ") { // Protection from Evil and Good omits the comma (content-spells.md FR-053)
		if !ok {
			r = strings.TrimPrefix(printed, "Concentration up to ")
		}
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
	case "day":
		u = content.DurationUnitDays
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

// l01SpellOracle is a spell row of testdata/srd/spells_levels_0_1: the slice row plus the printed page and the
// number of description blocks.
type l01SpellOracle struct {
	spellOracle
	Page      int      `json:"page"`
	Blocks    int      `json:"blocks"`
	StatBlock []string `json:"statBlock"`
	Bullets   []string `json:"bullets"`
}

type l01OracleFile struct {
	Spells  []l01SpellOracle `json:"spells"`
	TableFA struct {
		Levels map[string]int `json:"levels"`
	} `json:"tableFA"`
}

func readSpellOracle(t *testing.T, dir string) l01OracleFile {
	t.Helper()
	raw, err := oracleFS.ReadFile("testdata/srd/" + dir + "/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	var o l01OracleFile
	if err := json.Unmarshal(raw, &o); err != nil {
		t.Fatal(err)
	}
	return o
}

func readL01Oracle(t *testing.T) l01OracleFile {
	t.Helper()
	return readSpellOracle(t, "spells_levels_0_1")
}

func readL49Oracle(t *testing.T) l01OracleFile {
	t.Helper()
	return readSpellOracle(t, "spells_levels_4_9")
}

func readL23Oracle(t *testing.T) l01OracleFile {
	t.Helper()
	return readSpellOracle(t, "spells_levels_2_3")
}

// wantGP reads every "N+ GP" figure of a printed Material description, in order, thousands separators removed.
func wantGP(t *testing.T, printed string) []int {
	t.Helper()
	var out []int
	for _, f := range regexp.MustCompile(`([0-9][0-9,]*)\+ GP`).FindAllStringSubmatch(printed, -1) {
		n, err := strconv.Atoi(strings.ReplaceAll(f[1], ",", ""))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

// Requirement: doc/specs/content-spells.md FR-004, FR-012 to FR-014, FR-031, FR-032, FR-034, FR-040 to FR-044, FR-049 to
// FR-057, FR-085 and doc/plans/plan-phase-3.md 5.10 - each of the 16 level 0 and 1 spells that the class files name has
// the id, level, school, class set, Casting Time (trigger and Ritual flag), Range, Components (Material text, GP costs
// and consumption), Duration and citation page of its oracle row.
// Path: per case: happy / unhappy (boundary: Bonus Action with a trigger; Concentration without the comma; Ritual;
// Material with a consumed cost; singular "Component").
func TestLevel01SpellHeaders(t *testing.T) {
	t.Parallel()
	checkSpellHeaders(t, readL01Oracle(t).Spells)
}

// Requirement: doc/specs/content-spells.md FR-004, FR-012 to FR-014, FR-031, FR-032, FR-034, FR-040 to FR-044, FR-049 to
// FR-057, FR-085 and doc/plans/plan-phase-3.md 5.10 - each of the 20 level 2 and 3 spells that the class files name has
// the id, level, school, class set, Casting Time (trigger and Ritual flag), Range, Components (Material text, GP costs
// and consumption), Duration and citation page of its oracle row.
// Path: per case: happy / unhappy (boundary: Bonus Action Casting Time; singular "Component" label; consumed Material
// with a cost; Concentration up to 8 hours; Self and Touch ranges).
func TestLevel23SpellHeaders(t *testing.T) {
	t.Parallel()
	checkSpellHeaders(t, readL23Oracle(t).Spells)
}

// Requirement: doc/specs/content-spells.md FR-004, FR-012 to FR-014, FR-031, FR-032, FR-040 to FR-044, FR-049 to FR-057, FR-085
// and doc/plans/plan-phase-3.md 5.10 - each of the 25 level 4 to 9 spells that the class files name has the id, level,
// school, class set, Casting Time (minutes, Ritual flag), Range, Components (singular "Component" label, Material text,
// GP costs and consumption), Duration (hours, days, Concentration) and citation page of its oracle row.
// Path: per case: happy / unhappy (boundary: levels 4, 5 and 9; Casting Time of 1 minute, 10 minutes and a Ritual;
// Duration in days; consumed Material with one and with two costs; Verbal-only components).
func TestLevel49SpellHeaders(t *testing.T) {
	t.Parallel()
	checkSpellHeaders(t, readL49Oracle(t).Spells)
}

func checkSpellHeaders(t *testing.T, rows []l01SpellOracle) {
	t.Helper()
	src := newSlice(t)
	for _, o := range rows {
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
			if c := s.Source(); c.Document() != "SRD 5.2.1" || c.Section() != "Spell Descriptions" || c.Page() != o.Page {
				t.Errorf("%s source = %q, %q, %d; want SRD 5.2.1, Spell Descriptions, %d", o.ID, c.Document(), c.Section(), c.Page(), o.Page)
			}
			var wantClasses, gotClasses []string
			for _, c := range o.Classes {
				wantClasses = append(wantClasses, strings.ToLower(c))
			}
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
			if want := strings.HasPrefix(o.Duration, "Concentration"); s.RequiresConcentration() != want {
				t.Errorf("%s RequiresConcentration = %v, want %v", o.ID, s.RequiresConcentration(), want)
			}
			letters, material, hasMaterial := strings.Cut(o.Components, " (")
			c := s.Components()
			gotMaterial, gotHas := c.Material()
			if c.Verbal() != strings.Contains(letters, "V") || c.Somatic() != strings.Contains(letters, "S") || gotHas != hasMaterial {
				t.Fatalf("%s components = V %v S %v M %v, want %q", o.ID, c.Verbal(), c.Somatic(), gotHas, o.Components)
			}
			if gotHas {
				desc := strings.TrimSuffix(material, ")")
				if gotMaterial.Description() != desc || !slices.Equal(gotMaterial.CostsGP(), wantGP(t, desc)) ||
					gotMaterial.ConsumesMaterial() != strings.Contains(desc, "consumes") {
					t.Errorf("%s material = %q, %v, %v; want %q, %v, %v", o.ID, gotMaterial.Description(), gotMaterial.CostsGP(), gotMaterial.ConsumesMaterial(), desc, wantGP(t, desc), strings.Contains(desc, "consumes"))
				}
			}
		})
	}
}

// Requirement: doc/specs/content-spells.md FR-058 to FR-064, FR-115 and doc/plans/plan-phase-3.md 5.10 - each of the 16
// spells has the description blocks, phrases and run-in entries of its oracle row; its scaling text sits in
// cantripUpgrade (level 0) or higherLevel (level 1) or is absent, never both and never in the wrong field.
// Path: per case: happy / unhappy (rule interaction: scaling field versus level; empty scaling absent).
func TestLevel01SpellText(t *testing.T) {
	t.Parallel()
	checkSpellText(t, readL01Oracle(t).Spells)
}

// Requirement: doc/specs/content-spells.md FR-058 to FR-064, FR-069, FR-115 and doc/plans/plan-phase-3.md 5.10 - each of the 20
// level 2 and 3 spells has the description blocks, phrases and run-in entries of its oracle row (Find Steed also keeps its
// stat block as one block of the description); its scaling text sits in higherLevel or is absent, never in cantripUpgrade.
// Path: per case: happy / unhappy (rule interaction: scaling field versus level; header-less sub-entry kept in the owning spell).
func TestLevel23SpellText(t *testing.T) {
	t.Parallel()
	checkSpellText(t, readL23Oracle(t).Spells)
}

// Requirement: doc/specs/content-spells.md FR-058 to FR-064, FR-069, FR-070, FR-115 and doc/plans/plan-phase-3.md 5.10 - each of
// the 25 level 4 to 9 spells has the description blocks, phrases and run-in entries of its oracle row (Greater Restoration keeps
// its bullet list as one bulletList block, Summon Dragon its Draconic Spirit stat block as one block of the description);
// its scaling text sits in higherLevel or is absent, never in cantripUpgrade.
// Path: per case: happy / unhappy (rule interaction: scaling field versus level; bullet list and stat block kept in the owning spell).
func TestLevel49SpellText(t *testing.T) {
	t.Parallel()
	checkSpellText(t, readL49Oracle(t).Spells)
}

func checkSpellText(t *testing.T, rows []l01SpellOracle) {
	t.Helper()
	src := newSlice(t)
	for _, o := range rows {
		t.Run(o.ID, func(t *testing.T) {
			t.Parallel()
			s := spellOf(t, src, o.ID)
			if len(s.Description()) != o.Blocks {
				t.Errorf("%s has %d description blocks, want %d", o.ID, len(s.Description()), o.Blocks)
			}
			text := strings.Join(blockTexts(t, s.Description()), "\n")
			for _, p := range o.Phrases {
				if !strings.Contains(text, p) {
					t.Errorf("%s description lacks %q", o.ID, p)
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
			var stat []string
			for _, b := range s.Description() {
				if sb, ok := b.(content.TextEmbeddedStatBlock); ok {
					stat = append(stat, sb.Lines()...)
				}
			}
			if !slices.Equal(stat, o.StatBlock) {
				t.Errorf("%s stat block lines = %q, want %q", o.ID, stat, o.StatBlock)
			}
			var bullets []string
			for _, b := range s.Description() {
				if l, ok := b.(content.TextBulletList); ok {
					bullets = append(bullets, l.Items()...)
				}
			}
			if !slices.Equal(bullets, o.Bullets) {
				t.Errorf("%s bullet items = %q, want %q", o.ID, bullets, o.Bullets)
			}
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
			for _, banned := range bannedTerms() {
				if containsWord(text, banned) {
					t.Errorf("%s description contains the banned term %q", o.ID, banned)
				}
			}
		})
	}
}

// Requirement: doc/specs/content-classes-data.md Table FA, doc/plans/plan-phase-3.md 5.10 (acceptance: no
// unresolvedReference for a Table FA spell id) - every spell id named by an AlwaysPreparedSpells fact, of any level 0 to 9,
// resolves in the bundle with the printed level, no bundled spell contradicts its printed level, and the bundle holds
// exactly the 12 slice spells and the Table FA spells (plan P2: no others; steps 10a, 10b and 10c).
// Path: unhappy (boundary: levels 0, 3, 4 and 9; no extra spell bundled).
func TestTableFASpellsResolve(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	levels := readL01Oracle(t).TableFA.Levels
	want := map[string]bool{}
	for _, id := range sliceIDs()[content.KindSpell] {
		want[id] = true
	}
	for id, level := range levels {
		want[id] = true
		if got := spellOf(t, src, id); int(got.Level()) != level {
			t.Errorf("Table FA spell %s has level %d, want %d", id, got.Level(), level)
		}
	}
	for _, id := range bundledSpellIDs() {
		if !want[id] {
			t.Errorf("bundled spell %s is neither a slice spell nor a Table FA spell", id)
		}
		if level, ok := levels[id]; ok {
			if got := spellOf(t, src, id); int(got.Level()) != level {
				t.Errorf("bundled spell %s has level %d, Table FA prints %d", id, got.Level(), level)
			}
		}
	}
	if len(want) != len(bundledSpellIDs()) {
		t.Errorf("expected spells = %d, bundled = %d", len(want), len(bundledSpellIDs()))
	}
}

// Requirement: doc/specs/content-classes-data.md FR-100 (staging pin amended by doc/plans/plan-phase-3.md P5) - until the
// full 339-spell set is bundled the class-list coverage check stays skipped: this pin fails when the set reaches 339,
// in the change that enables the check, and meanwhile every ClassID named by a bundled spell is one of the eight
// spellcasting classes.
// Path: unhappy (rule interaction: spell naming a non-caster class).
func TestFullSpellSetNotYetBundled(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	list, err := src.ListKind(context.Background(), content.KindSpell)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) >= 339 {
		t.Fatalf("%d spells are bundled: the full set has landed, so enable content-classes-data.md FR-100 and FR-101 and remove this pin", len(list))
	}
	casters := []string{"bard", "cleric", "druid", "paladin", "ranger", "sorcerer", "warlock", "wizard"}
	for _, e := range list {
		for _, c := range e.(content.Spell).Classes() {
			if !slices.Contains(casters, string(c)) {
				t.Errorf("spell %s names class %q, which is not a spellcasting class", e.Key().ID(), c)
			}
		}
	}
}
