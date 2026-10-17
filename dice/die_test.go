package dice_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func reasonOf(t *testing.T, err error) dice.Reason {
	t.Helper()
	var ie *dice.InvalidError
	if !errors.As(err, &ie) {
		t.Fatalf("error %v is not an *InvalidError", err)
	}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("error %v does not match ErrInvalidInput", err)
	}
	return ie.Reason
}

func TestNewDie(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sides int
		want  dice.Reason
	}{{dice.MinSides, 0}, {20, 0}, {dice.MaxSides, 0}, {1, dice.ReasonSidesOutOfRange}, {0, dice.ReasonSidesOutOfRange}, {-3, dice.ReasonSidesOutOfRange}, {dice.MaxSides + 1, dice.ReasonSidesOutOfRange}} {
		d, err := dice.NewDie(tc.sides)
		if tc.want == 0 {
			if err != nil || d.Sides() != tc.sides || d.IsZero() {
				t.Errorf("NewDie(%d) = %v, %v", tc.sides, d, err)
			}
			continue
		}
		if err == nil || reasonOf(t, err) != tc.want || !d.IsZero() {
			t.Errorf("NewDie(%d) = %v, %v", tc.sides, d, err)
		}
	}
}

func TestDieNamed(t *testing.T) {
	t.Parallel()
	for want, d := range map[int]dice.Die{3: dice.D3(), 4: dice.D4(), 6: dice.D6(), 8: dice.D8(), 10: dice.D10(), 12: dice.D12(), 20: dice.D20(), 100: dice.D100()} {
		if d.Sides() != want || d != dice.MustDie(want) {
			t.Errorf("named die %d = %v", want, d)
		}
	}
	if dice.D20().String() != "d20" {
		t.Errorf("String = %q", dice.D20())
	}
}

func TestMustPanicsWithError(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(){
		"die":        func() { dice.MustDie(1) },
		"roll":       func() { dice.MustDieRoll(dice.D6(), 7) },
		"expression": func() { dice.MustExpression(0, dice.D6(), 0) },
		"parse":      func() { dice.MustParseExpression("x") },
	} {
		func() {
			defer func() {
				err, ok := recover().(error)
				if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("%s: panic value %v is not an invalid-input error", name, err)
				}
			}()
			f()
		}()
	}
}

func TestDieRoll(t *testing.T) {
	t.Parallel()
	r, err := dice.NewDieRoll(dice.D8(), 5)
	if err != nil || r.Value() != 5 || r.Die() != dice.D8() || r.IsZero() || r.String() != "d8=5" {
		t.Fatalf("NewDieRoll = %v, %v", r, err)
	}
	for _, tc := range []struct {
		die   dice.Die
		value int
	}{{dice.D8(), 0}, {dice.D8(), 9}, {dice.D8(), -1}, {dice.Die{}, 1}} {
		r, err := dice.NewDieRoll(tc.die, tc.value)
		if err == nil || reasonOf(t, err) != dice.ReasonValueOutOfRange || !r.IsZero() {
			t.Errorf("NewDieRoll(%v, %d) = %v, %v", tc.die, tc.value, r, err)
		}
	}
	if !(dice.Die{}).IsZero() {
		t.Error("zero die not zero")
	}
}

func TestReasonAndInvalidError(t *testing.T) {
	t.Parallel()
	for r := dice.ReasonEmpty; r <= dice.ReasonSelectionOutOfRange; r++ {
		if r.String() == "unknown" || r.String() == "" {
			t.Errorf("Reason %d has no name", r)
		}
	}
	for _, r := range []dice.Reason{0, -1, dice.ReasonSelectionOutOfRange + 1, 99} {
		if r.String() != "unknown" {
			t.Errorf("Reason(%d) = %q", r, r)
		}
	}
	err := error(&dice.InvalidError{Reason: dice.ReasonMalformed, Detail: "x"})
	if err.Error() != "dice: malformed: x" || (&dice.InvalidError{Reason: dice.ReasonEmpty}).Error() != "dice: empty" {
		t.Errorf("Error = %q", err)
	}
	if !errors.Is(err, &dice.InvalidError{Reason: dice.ReasonMalformed}) || errors.Is(err, &dice.InvalidError{Reason: dice.ReasonEmpty}) || errors.Is(err, errors.New("x")) {
		t.Error("Is matching wrong")
	}
}
