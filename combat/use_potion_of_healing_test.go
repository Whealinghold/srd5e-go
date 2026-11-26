package combat_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func potionRun(tb testing.TB, user, patient string, st combat.HitPointStatus, use combat.PotionUse, roller combat.Roller, opts ...combat.Option) (combat.UsePotionOfHealingResult, error) {
	tb.Helper()
	cmd, err := combat.NewUsePotionOfHealingCommand(gid(user), gid(patient), st, use)
	if err != nil {
		tb.Fatal(err)
	}
	u, err := combat.NewUsePotionOfHealing(roller, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return u.Run(context.Background(), cmd)
}

// Requirement: per case
// Path: per case.
func TestUsePotionOfHealingRun(t *testing.T) {
	t.Parallel()
	spec := itemUseSpec
	cases := []struct {
		name, requirement, path string
		status                  combat.HitPointStatus
		use                     combat.PotionUse
		faces                   []int
		amount, regained, hp    int
		life                    combat.LifeState
		events                  []string
	}{
		{"rolls 1,1 at 14 of 20", spec + "FR-040 table 4.6a", "happy", pcStatus(t, 14, 20, 0), combat.DrinkPotion{}, []int{1, 1}, 4, 4, 18, combat.Alive{}, []string{"hit_points_restored", "potion_of_healing_used"}},
		{"rolls 4,4 at 14 of 20", spec + "FR-040 table 4.6a", "unhappy (boundary)", pcStatus(t, 14, 20, 0), combat.DrinkPotion{}, []int{4, 4}, 10, 6, 20, combat.Alive{}, []string{"hit_points_restored", "potion_of_healing_used"}},
		{"at the maximum regains 0 and is consumed", spec + "FR-044 table 4.6a", "unhappy (boundary)", pcStatus(t, 20, 20, 0), combat.DrinkPotion{}, []int{2, 3}, 7, 0, 20, combat.Alive{}, []string{"potion_of_healing_used"}},
		{"Dying at 0 becomes Alive", spec + "FR-043 table 4.6a", "unhappy (rule interaction)", dyingStatus(t, 20, 1, 2), combat.AdministerPotion{DistanceFeet: 5}, []int{2, 2}, 6, 6, 6, combat.Alive{}, []string{"hit_points_restored", "recovered", "potion_of_healing_used"}},
		{"Stable at 0 becomes Alive", spec + "FR-043 table 4.6a", "unhappy (rule interaction)", stablePC(t), combat.AdministerPotion{}, []int{1, 4}, 7, 7, 7, combat.Alive{}, []string{"hit_points_restored", "recovered", "potion_of_healing_used"}},
		{"drinking at 0 Hit Points is allowed", spec + "FR-041 table 4.6a", "unhappy (rule interaction)", dyingStatus(t, 20, 0, 1), combat.DrinkPotion{}, []int{3, 3}, 8, 8, 8, combat.Alive{}, []string{"hit_points_restored", "recovered", "potion_of_healing_used"}},
		{"administered at 5 feet", spec + "FR-041 table 4.6a", "unhappy (boundary)", pcStatus(t, 1, 20, 0), combat.AdministerPotion{DistanceFeet: 5}, []int{1, 1}, 4, 4, 5, combat.Alive{}, []string{"hit_points_restored", "potion_of_healing_used"}},
		{"temporary Hit Points are untouched", spec + "FR-040", "unhappy (rule interaction)", pcStatus(t, 10, 20, 5), combat.DrinkPotion{}, []int{2, 2}, 6, 6, 16, combat.Alive{}, []string{"hit_points_restored", "potion_of_healing_used"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			user := "hero"
			if _, drink := c.use.(combat.DrinkPotion); !drink {
				user = "medic"
			}
			res, err := potionRun(t, user, "hero", c.status, c.use, roller)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			used, ok := res.(combat.PotionOfHealingUsed)
			if !ok {
				t.Fatalf("result = %#v, want used [%s]", res, c.requirement)
			}
			if got := requested(roller); !slices.Equal(got, []string{"2d4+2"}) {
				t.Errorf("dice = %v, want exactly 2d4+2 [FR-040]", got)
			}
			if used.Amount() != c.amount || used.HitPointsRegained() != c.regained || used.Status().HitPoints().Current() != c.hp || used.Status().LifeState() != c.life ||
				used.Cost() != combat.ItemActionCostBonusAction || used.Roll().Total() != c.amount {
				t.Errorf("used = amount %d regained %d hp %d life %v, want %d %d %d %v [%s]", used.Amount(), used.HitPointsRegained(), used.Status().HitPoints().Current(),
					used.Status().LifeState(), c.amount, c.regained, c.hp, c.life, c.requirement)
			}
			if got, ok := used.Consumption().(combat.ItemConsumed); !ok || got.Count() != 1 || got.Item() != content.MustItemID("potion_of_healing") {
				t.Errorf("consumption = %#v, want one potion_of_healing [FR-044]", used.Consumption())
			}
			ev := used.Events()
			var kinds []string
			for _, e := range ev {
				kinds = append(kinds, e.Kind())
			}
			if !slices.Equal(kinds, c.events) {
				t.Errorf("events = %v, want %v [FR-045]", kinds, c.events)
			}
			want := []combat.CombatantID{gid(user)}
			if user != "hero" {
				want = append(want, gid("hero"))
			}
			combattest.CheckEvent(t, ev[len(ev)-1], "potion_of_healing_used", want...)
			combattest.CheckEventsCopy(t, used.Events)
			combattest.CheckEventKinds(t, ev)
		})
	}
}

// Requirement: per case
// Path: per case.
func TestUsePotionOfHealingNotPossible(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		status                  combat.HitPointStatus
		use                     combat.PotionUse
		want                    combat.PotionNotPossibleReason
	}{
		{"administered at 6 feet", itemUseSpec + "FR-041 table 4.6a", "unhappy (boundary)", pcStatus(t, 5, 20, 0), combat.AdministerPotion{DistanceFeet: 6}, combat.PotionReasonNotWithinFiveFeet},
		{"administered very far", itemUseSpec + "FR-041", "unhappy (boundary)", pcStatus(t, 5, 20, 0), combat.AdministerPotion{DistanceFeet: math.MaxInt}, combat.PotionReasonNotWithinFiveFeet},
		{"Dead patient", itemUseSpec + "FR-042 table 4.6a", "unhappy (state error)", deadStatus(t, combat.MassiveDamage), combat.DrinkPotion{}, combat.PotionReasonPatientIsDead},
		{"Dead patient administered", itemUseSpec + "FR-042", "unhappy (state error)", deadStatus(t, combat.MonsterDeath), combat.AdministerPotion{DistanceFeet: 5}, combat.PotionReasonPatientIsDead},
		{"Dead patient too far: the distance is checked first", itemUseSpec + "FR-041 FR-042", "unhappy (rule interaction)", deadStatus(t, combat.MonsterDeath), combat.AdministerPotion{DistanceFeet: 6}, combat.PotionReasonNotWithinFiveFeet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(4, 4)
			user := "medic"
			if _, drink := c.use.(combat.DrinkPotion); drink {
				user = "hero"
			}
			res, err := potionRun(t, user, "hero", c.status, c.use, roller)
			if got, ok := res.(combat.PotionOfHealingNotPossible); err != nil || !ok || got.Reason != c.want {
				t.Fatalf("result = %#v, %v, want not possible %v [%s]", res, err, c.want, c.requirement)
			}
			if n := len(roller.Requests()); n != 0 {
				t.Errorf("%d dice requested before a refusal, want none [FR-042]", n)
			}
			if c.status.LifeState() == nil {
				t.Error("the status was altered")
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-041 - the command: DrinkPotion needs the user to be the patient, a negative distance and a zero user, patient or status are typed errors, and the reasons print their snake_case values.
// Path: unhappy (invalid input).
func TestNewUsePotionOfHealingCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 5, 20, 0)
	cases := []struct {
		name          string
		user, patient combat.CombatantID
		status        combat.HitPointStatus
		use           combat.PotionUse
		field         combat.ItemUseField
		value         int64
	}{
		{"drink with another patient", "a", "b", st, combat.DrinkPotion{}, combat.ItemUseFieldPatient, 0},
		{"negative distance", "a", "b", st, combat.AdministerPotion{DistanceFeet: -1}, combat.ItemUseFieldDistanceFeet, -1},
		{"minimum distance", "a", "b", st, combat.AdministerPotion{DistanceFeet: math.MinInt}, combat.ItemUseFieldDistanceFeet, math.MinInt},
		{"nil use", "a", "b", st, nil, combat.ItemUseFieldPotionUse, 0},
		{"zero user", "", "b", st, combat.AdministerPotion{}, combat.ItemUseFieldUser, 0},
		{"zero patient", "a", "", st, combat.AdministerPotion{}, combat.ItemUseFieldPatient, 0},
		{"zero status", "a", "b", combat.HitPointStatus{}, combat.AdministerPotion{}, combat.ItemUseFieldStatus, 0},
		{"drinking, both zero", "", "", st, combat.DrinkPotion{}, combat.ItemUseFieldUser, 0},
	}
	for _, c := range cases {
		_, err := combat.NewUsePotionOfHealingCommand(c.user, c.patient, c.status, c.use)
		var ue *combat.ItemUseError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ue) || ue.Field != c.field || ue.Value != c.value {
			t.Errorf("%s: err = %v, want field %v value %d", c.name, err, c.field, c.value)
		}
	}
	defer func() {
		if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
			t.Errorf("MustUsePotionOfHealingCommand panicked with %v", r)
		}
	}()
	for r, want := range map[combat.PotionNotPossibleReason]string{combat.PotionReasonNotWithinFiveFeet: "not_within_five_feet", combat.PotionReasonPatientIsDead: "patient_is_dead", 0: "none", 9: "unknown"} {
		if r.String() != want {
			t.Errorf("reason %d = %q", r, r.String())
		}
	}
	combat.MustUsePotionOfHealingCommand("", "", st, combat.DrinkPotion{})
}

// Requirement: doc/specs/item-use-effects.md FR-071, FR-074, FR-075 - a nil roller, the zero use case and command are invalid, a cancelled ctx is returned unwrapped before any die, a failing, panicking or wrong-expression roller is one *srderr.PortError with no result and the status unchanged, and equal inputs with equal rolls give equal results.
// Path: unhappy (invalid input, cancellation, failure of collaborator, immutability, determinism).
func TestUsePotionOfHealingFailures(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewUsePotionOfHealing(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	st := pcStatus(t, 5, 20, 0)
	cmd := combat.MustUsePotionOfHealingCommand(gid("hero"), gid("hero"), st, combat.DrinkPotion{})
	good, _ := combat.NewUsePotionOfHealing(dicetest.NewFixedRoller(1, 1))
	if _, err := good.Run(context.Background(), combat.UsePotionOfHealingCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v", err)
	}
	if _, err := (combat.UsePotionOfHealing{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	roller := dicetest.NewFixedRoller(1, 1)
	cancelled, _ := combat.NewUsePotionOfHealing(roller)
	if res, err := cancelled.Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	for name, rl := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		u, _ := combat.NewUsePotionOfHealing(rl)
		res, err := u.Run(context.Background(), cmd)
		if res != nil {
			t.Errorf("%s: partial result %#v", name, res)
		}
		envPortFailure(t, err)
	}
	if st.HitPoints().Current() != 5 {
		t.Error("the status changed")
	}
	a, _ := potionRun(t, "hero", "hero", st, combat.DrinkPotion{}, dicetest.NewFixedRoller(2, 3))
	b, _ := potionRun(t, "hero", "hero", st, combat.DrinkPotion{}, dicetest.NewFixedRoller(2, 3))
	if !reflect.DeepEqual(a, b) {
		t.Errorf("equal inputs gave %#v and %#v", a, b)
	}
}
