package dice_test

import (
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
)

func TestParseExpressionAccepts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in, notation string
	}{
		{"d20", "1d20"}, {"1d20", "1d20"}, {"D12", "1d12"}, {"4d6", "4d6"}, {"1d100", "1d100"},
		{"2d8 + 2", "2d8+2"}, {"1d4+2", "1d4+2"}, {"1d4 - 5", "1d4-5"}, {"1d4-5", "1d4-5"},
		{"1d20 − 1", "1d20-1"}, {"1d20−1", "1d20-1"},
		{"1d20+0", "1d20"}, {"1d20-0", "1d20"}, {"1d20 + 0", "1d20"},
		{"01d6", "1d6"}, {"1d006", "1d6"}, {"1d6+007", "1d6+7"}, {"0000000000000000000001d6", "1d6"},
		{"1d3", "1d3"}, {"1000d1000+1000000", "1000d1000+1000000"}, {"1d2-1000000", "1d2-1000000"},
	} {
		e, err := dice.ParseExpression(tc.in)
		if err != nil || e.Notation() != tc.notation || e.String() != tc.notation {
			t.Errorf("ParseExpression(%q) = %v, %v; want %s", tc.in, e, err, tc.notation)
		}
		if again, err := dice.ParseExpression(e.Notation()); err != nil || again != e {
			t.Errorf("round trip of %q failed: %v, %v", tc.in, again, err)
		}
	}
}

func TestParseExpressionRejects(t *testing.T) {
	t.Parallel()
	cases := map[dice.Reason][]string{
		dice.ReasonEmpty: {""},
		dice.ReasonMalformed: {
			"d", "2d", "2x6", "1d6+", "1d6++1", "1d6 + + 1", "d+2", "2 d6", "2d 6", "1.5d6", "1d6+1.5", "-1d6", "+1d6",
			"   ", " 1d20", "1d20 ", "1d6\n", "1d4 +2", "1d4+ 2", "1d4  +  2", "1d20 −1", "1d20− 1",
			"1d20 – 1", "１d20", "1d4 × 10", "1d6+2d4", "1d6 ", "1d6 +", "1d6 *2", "1d6 + 2 ",
			// Digits only, and bytes either side of 'd' and 'D' where the d check could drift.
			"2", "20", "2a6", "2C6", "2x6", "2+6",
			// Control characters are not the space that separates a spaced sign.
			"1d6\t+ 2", "1d6\x00+ 2", "1d4 +\t2", "1d4 +23", "1d4 + ",
		},
		dice.ReasonCountOutOfRange:    {"0d6", "00d6", "1001d6", "0d0", "99999999999999999999d6"},
		dice.ReasonSidesOutOfRange:    {"1d0", "1d1", "1d1001", "1d99999999999999999999"},
		dice.ReasonModifierOutOfRange: {"1d6+1000001", "1d6-1000001", "1d6+99999999999999999999"},
	}
	for want, inputs := range cases {
		for _, in := range inputs {
			e, err := dice.ParseExpression(in)
			if err == nil || !e.IsZero() || reasonOf(t, err) != want {
				t.Errorf("ParseExpression(%q) = %v, %v; want reason %v", in, e, err, want)
			}
		}
	}
}

func TestNewExpression(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		count    int
		die      dice.Die
		modifier int
		want     dice.Reason
	}{
		{"ok", 2, dice.D8(), 2, 0},
		{"limits", dice.MaxCount, dice.D100(), -dice.MaxModifier, 0},
		{"count zero", 0, dice.D8(), 0, dice.ReasonCountOutOfRange},
		{"count high", dice.MaxCount + 1, dice.D8(), 0, dice.ReasonCountOutOfRange},
		{"zero die", 1, dice.Die{}, 0, dice.ReasonSidesOutOfRange},
		{"modifier high", 1, dice.D8(), dice.MaxModifier + 1, dice.ReasonModifierOutOfRange},
		{"modifier low", 1, dice.D8(), -dice.MaxModifier - 1, dice.ReasonModifierOutOfRange},
	} {
		e, err := dice.NewExpression(tc.count, tc.die, tc.modifier)
		if tc.want == 0 {
			if err != nil || e.Count() != tc.count || e.Die() != tc.die || e.Modifier() != tc.modifier || e.IsZero() {
				t.Errorf("%s: %v, %v", tc.name, e, err)
			}
			continue
		}
		if err == nil || reasonOf(t, err) != tc.want || !e.IsZero() {
			t.Errorf("%s: %v, %v", tc.name, e, err)
		}
	}
}

func TestExpressionStats(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in           string
		min, max, av int
	}{
		{"1d20", 1, 20, 10}, {"2d8+2", 4, 18, 11}, {"4d6", 4, 24, 14}, {"1d4-5", -4, -1, -3}, {"1d2-1", 0, 1, 0},
	} {
		e := dice.MustParseExpression(tc.in)
		if e.Minimum() != tc.min || e.Maximum() != tc.max || e.Average() != tc.av {
			t.Errorf("%s: min %d max %d avg %d", tc.in, e.Minimum(), e.Maximum(), e.Average())
		}
	}
}

func TestExpressionDouble(t *testing.T) {
	t.Parallel()
	e := dice.MustParseExpression("2d8+2")
	d, err := e.Double()
	if err != nil || d.Notation() != "4d8+2" || !e.CanDouble() {
		t.Errorf("Double = %v, %v", d, err)
	}
	half := dice.MustExpression(dice.MaxCount/2, dice.D6(), 0)
	if d, err := half.Double(); err != nil || d.Count() != dice.MaxCount {
		t.Errorf("Double at limit = %v, %v", d, err)
	}
	for _, e := range []dice.Expression{dice.MustExpression(dice.MaxCount/2+1, dice.D6(), 0), {}} {
		d, err := e.Double()
		if e.CanDouble() || err == nil || reasonOf(t, err) != dice.ReasonCountOutOfRange || !d.IsZero() {
			t.Errorf("Double(%v) = %v, %v", e, d, err)
		}
	}
}

func TestParseHugeInput(t *testing.T) {
	t.Parallel()
	if _, err := dice.ParseExpression(strings.Repeat("9", 100000) + "d6"); reasonOf(t, err) != dice.ReasonCountOutOfRange {
		t.Errorf("huge count: %v", err)
	}
}

func FuzzParseExpression(f *testing.F) {
	for _, s := range []string{"", "d20", "2d8 + 2", "1d20−1", "1d4 -5", "0d0", "99999999999999999999d6", "1d6+1.5", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		e, err := dice.ParseExpression(s)
		if err != nil {
			if !e.IsZero() {
				t.Fatalf("error with non-zero expression %v", e)
			}
			return
		}
		if e.Count() < 1 || e.Count() > dice.MaxCount || e.Die().IsZero() {
			t.Fatalf("out-of-limit expression %v from %q", e, s)
		}
		if again, err := dice.ParseExpression(e.Notation()); err != nil || again != e {
			t.Fatalf("notation %q of %q does not round trip: %v, %v", e.Notation(), s, again, err)
		}
	})
}

// Requirement: none - pins current behaviour of ParseExpression: the order of
// range reasons and the Detail text, which is otherwise not a contract.
func TestParseExpressionRangeDetail(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"0d0":            "dice: count out of range",
		"1001d1":         "dice: count out of range",
		"1000d1":         "dice: sides out of range",
		"2d1":            "dice: sides out of range",
		"1d1":            "dice: sides out of range",
		"1d1001":         "dice: sides out of range",
		"1d6+999999999":  "dice: modifier out of range: modifier 999999999",
		"1d6+9999999999": "dice: modifier out of range: modifier 2000000000",
		"1d6-9999999999": "dice: modifier out of range: modifier -2000000000",
	} {
		_, err := dice.ParseExpression(in)
		if err == nil || err.Error() != want {
			t.Errorf("ParseExpression(%q) error = %v; want %q", in, err, want)
		}
	}
}
