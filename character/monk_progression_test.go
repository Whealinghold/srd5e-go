package character_test

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// mnkBand is one band of Monk levels of the Monk Features table where the Martial Arts die and the Unarmored Movement bonus are constant.
type mnkBand struct {
	from, to, sides, feet int
}

// mnkTable is typed from the Monk Features table (SRD 5.2.1, p. 50), independent of content/data. The Focus Points column is the
// level from 2 and a dash at level 1.
func mnkTable() []mnkBand {
	return []mnkBand{
		{1, 1, 6, 0}, {2, 4, 6, 10}, {5, 5, 8, 10}, {6, 9, 8, 15}, {10, 10, 8, 20}, {11, 13, 10, 20}, {14, 16, 10, 25}, {17, 17, 12, 25}, {18, 20, 12, 30},
	}
}

// Requirement: doc/specs/class-monk.md FR-033, FR-044, FR-047, FR-152 - at every Monk level 1 to 20 the stats equal the printed row: the
// Martial Arts die (d6, d8, d10, d12), the Focus Points maximum (0 at level 1, then the level) and the Unarmored Movement feet (0 at level 1).
// Path: per case (the 20 levels).
func TestMonkProgressionStatsAtEveryLevel(t *testing.T) {
	t.Parallel()
	p := mnkProgression(t)
	for _, b := range mnkTable() {
		for n := b.from; n <= b.to; n++ {
			t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
				t.Parallel()
				s := p.StatsAt(character.MustMonkLevel(n))
				focus := n
				if n == 1 {
					focus = 0
				}
				if s.MartialArtsDie() != dice.MustDie(b.sides) || s.FocusPoints() != focus || s.UnarmoredMovementFeet() != b.feet || s.Level().Value() != n || s.IsZero() {
					t.Errorf("StatsAt(%d) = d%d, %d points, %d ft; want d%d, %d, %d", n, s.MartialArtsDie().Sides(), s.FocusPoints(), s.UnarmoredMovementFeet(), b.sides, focus, b.feet)
				}
			})
		}
	}
	if !p.StatsAt(character.MonkLevel{}).IsZero() || !(character.MonkProgression{}).StatsAt(character.MustMonkLevel(3)).IsZero() || !(character.MonkProgression{}).IsZero() || p.IsZero() {
		t.Error("the zero progression and the zero level must give zero stats")
	}
}

// Requirement: doc/specs/class-monk.md FR-018 - the Proficiency Bonus of each decoded Monk row equals the rule for a total level equal to that
// Monk level, and MonkLevelStats exposes none: a Monk 5 / Fighter 5 has +4 (total level 10), not the +3 of the Monk row 5.
// Path: unhappy (rule interaction).
func TestMonkProficiencyBonusFollowsTotalLevel(t *testing.T) {
	t.Parallel()
	def := mnkDef(t)
	for n := 1; n <= 20; n++ {
		row, err := def.RowAt(n)
		if err != nil || row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() {
			t.Errorf("row %d bonus = %d (%v), want %d", n, row.ProficiencyBonus(), err, character.ProficiencyBonusFor(character.MustLevel(n)).Value())
		}
	}
	multi := mnkChar(t, cl("monk", 5), cl("fighter", 5))
	if got := multi.ProficiencyBonus().Value(); got != 4 {
		t.Errorf("Monk 5 / Fighter 5 Proficiency Bonus = %d, want 4", got)
	}
}

// Requirement: doc/specs/class-monk.md FR-125, FR-140, FR-152 - NewMonkProgression accepts the Monk definition and joins a typed reason for another
// class (NotTheMonkDefinition), the zero definition, a column that is missing (MonkColumnMissing) and a level whose cell is of
// the wrong kind or whose die is not a single die (MonkRowsIncomplete); each matches srderr.ErrInvalidInput and the Must helper panics with the same error.
// Path: per case (unhappy: missing content).
func TestNewMonkProgression(t *testing.T) {
	t.Parallel()
	monk := fgtBundledText(t, "class_monk.json")
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		want character.MonkProgressionReason
	}{
		{"another class", fgtDecode(t, fgtBundledText(t, "class_fighter.json")), character.NotTheMonkDefinition},
		{"the zero definition", content.ClassDefinition{}, character.NotTheMonkDefinition},
		{"a missing column", fgtDecode(t, strings.ReplaceAll(monk, "focus_points", "focus_pointz")), character.MonkColumnMissing},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := character.NewMonkProgression(c.def)
			var pe *character.MonkProgressionError
			if !errors.As(err, &pe) || pe.Reason != c.want || !errors.Is(err, &character.MonkProgressionError{Reason: c.want}) {
				t.Fatalf("error = %v, want reason %v", err, c.want)
			}
			wantInvalid(t, err, c.name)
			defer func() {
				if recover() == nil {
					t.Error("MustMonkProgression did not panic")
				}
			}()
			character.MustMonkProgression(c.def)
		})
	}
	if _, err := character.NewMonkProgression(fgtDecode(t, monk)); err != nil {
		t.Errorf("the Monk definition = %v, want accepted", err)
	}
}

// Requirement: doc/specs/class-monk.md FR-152 - a Martial Arts cell that is a dash, or a die with two dice, is a row that is not a single die
// (MonkRowsIncomplete); a Focus Points cell that is a Unarmored Movement cell kind is rejected as the column kind (MonkColumnMissing).
// Path: per case (unhappy: invalid input).
func TestNewMonkProgressionRowsAndKinds(t *testing.T) {
	t.Parallel()
	monk := fgtBundledText(t, "class_monk.json")
	firstDie := regexp.MustCompile(`"martial_arts": \{[^}]*\}`)
	twoDice := firstDie.ReplaceAllStringFunc(monk, func(string) string { return `"martial_arts": {"count": 2, "sides": 6}` })
	dash := firstDie.ReplaceAllStringFunc(monk, func(string) string { return `"martial_arts": "—"` })
	asCount := strings.Replace(monk, `"kind": "count"`, `"kind": "bonus"`, 1)
	for _, c := range []struct {
		name, text string
		want       character.MonkProgressionReason
	}{
		{"two dice", twoDice, character.MonkRowsIncomplete}, {"a dash for the die", dash, character.MonkRowsIncomplete},
		{"Focus Points of another kind", asCount, character.MonkColumnMissing},
	} {
		if c.text == monk {
			t.Fatalf("%s: the fixture edit did not apply", c.name)
		}
		_, err := character.NewMonkProgression(fgtDecode(t, c.text))
		var pe *character.MonkProgressionError
		if !errors.As(err, &pe) || pe.Reason != c.want {
			t.Errorf("%s: error = %v, want reason %v", c.name, err, c.want)
		}
		wantInvalid(t, err, c.name)
	}
}

// Requirement: doc/specs/class-monk.md FR-152 - the reason names are snake_case and an undefined one is unknown.
// Path: per case.
func TestMonkProgressionReasonNames(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.MonkProgressionReason]string{
		character.NotTheMonkDefinition: "not_the_monk_definition", character.MonkColumnMissing: "column_missing",
		character.MonkRowsIncomplete: "rows_incomplete", 0: "unknown", 9: "unknown",
	} {
		if r.String() != want {
			t.Errorf("reason %d = %q, want %q", int(r), r.String(), want)
		}
	}
	if (&character.MonkProgressionError{Reason: character.MonkColumnMissing}).Is(errors.New("x")) {
		t.Error("a *MonkProgressionError must not match a foreign error")
	}
}

// Requirement: doc/specs/class-monk.md FR-125, FR-152 - the Must helper returns the progression of the Monk definition without panicking, a definition of a class after "monk" in the
// alphabet is refused as another class, a Martial Arts column declared as a count (a kind below the die) or a Focus Points column declared as a bonus (a kind above the count) is a
// missing column, and one reason never matches another through errors.Is, whichever side is greater.
// Path: per case (unhappy: missing content, boundary).
func TestNewMonkProgressionKindsAndIdentity(t *testing.T) {
	t.Parallel()
	if p := character.MustMonkProgression(mnkDef(t)); p.IsZero() {
		t.Error("MustMonkProgression returned the zero progression")
	}
	_, err := character.NewMonkProgression(fgtDecode(t, fgtBundledText(t, "class_wizard.json")))
	var pe *character.MonkProgressionError
	if !errors.As(err, &pe) || pe.Reason != character.NotTheMonkDefinition {
		t.Errorf("a Wizard definition = %v, want NotTheMonkDefinition", err)
	}
	monk := fgtBundledText(t, "class_monk.json")
	dieCell := regexp.MustCompile(`"martial_arts": \{[^}]*\}`)
	asCount := strings.Replace(dieCell.ReplaceAllString(monk, `"martial_arts": 3`), `"kind": "dice"`, `"kind": "count"`, 1)
	_, err = character.NewMonkProgression(fgtDecode(t, asCount))
	if !errors.As(err, &pe) || pe.Reason != character.MonkColumnMissing {
		t.Errorf("Martial Arts as a count = %v, want MonkColumnMissing", err)
	}
	reasons := []character.MonkProgressionReason{character.NotTheMonkDefinition, character.MonkColumnMissing, character.MonkRowsIncomplete}
	for _, have := range reasons {
		for _, other := range reasons {
			if got := errors.Is(&character.MonkProgressionError{Reason: have}, &character.MonkProgressionError{Reason: other}); got != (have == other) {
				t.Errorf("Is(%v, %v) = %v", have, other, got)
			}
		}
	}
}
