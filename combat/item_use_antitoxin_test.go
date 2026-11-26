package combat_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func drunk(tb testing.TB) combat.AntitoxinEffect {
	tb.Helper()
	d, err := combat.AntitoxinRules{}.Drink(gid("drinker"))
	if err != nil {
		tb.Fatal(err)
	}
	return d.Effect()
}

func advanceAntitoxin(tb testing.TB, e combat.AntitoxinEffect, d time.Duration) combat.AntitoxinUpdate {
	tb.Helper()
	u, err := e.Advance(d)
	if err != nil {
		tb.Fatal(err)
	}
	return u
}

// Requirement: doc/specs/item-use-effects.md FR-030 - drinking Antitoxin returns an effect at elapsed zero with the Bonus Action cost, one vial consumed and one AntitoxinDrunk event; a zero drinker is an error.
// Path: happy; unhappy (invalid input).
func TestAntitoxinDrink(t *testing.T) {
	t.Parallel()
	d, err := combat.AntitoxinRules{}.Drink(gid("drinker"))
	if err != nil {
		t.Fatal(err)
	}
	e := d.Effect()
	if !e.Active() || e.Elapsed() != 0 || !e.IsDrunkBy(gid("drinker")) || e.IsDrunkBy(gid("a")) || e.IsDrunkBy(gid("x")) || e.IsDrunkBy("") || d.Cost() != combat.ItemActionCostBonusAction {
		t.Errorf("drink = %+v, cost %v", e, d.Cost())
	}
	if c, ok := d.Consumption().(combat.ItemConsumed); !ok || c.Count() != 1 || c.Item() != content.MustItemID("antitoxin") {
		t.Errorf("consumption = %#v", d.Consumption())
	}
	ev := d.Events()
	if len(ev) != 1 {
		t.Fatalf("events = %v", ev)
	}
	combattest.CheckEvent(t, ev[0], "antitoxin_drunk", gid("drinker"))
	ev[0] = nil
	if d.Events()[0] == nil {
		t.Error("Events returns the stored slice [FR-075]")
	}
	if _, err := (combat.AntitoxinRules{}).Drink(""); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldUser}) {
		t.Errorf("zero drinker: %v", err)
	}
}

// Requirement: per case
// Path: per case.
func TestAntitoxinAdvance(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		advance                 time.Duration
		active, wornOff         bool
	}{
		{"59 minutes 59 seconds", itemUseSpec + "FR-031 table 4.5a", "unhappy (boundary)", 59*time.Minute + 59*time.Second, true, false},
		{"60 minutes", itemUseSpec + "FR-031 table 4.5a", "unhappy (boundary)", time.Hour, false, true},
		{"nothing", itemUseSpec + "FR-031", "unhappy (empty)", 0, true, false},
		{"the largest advance", itemUseSpec + "FR-031", "unhappy (boundary)", math.MaxInt64, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			u := advanceAntitoxin(t, drunk(t), c.advance)
			if u.Effect().Active() != c.active || (len(u.Events()) == 1) != c.wornOff {
				t.Fatalf("effect active %v with events %v [%s]", u.Effect().Active(), u.Events(), c.requirement)
			}
			if c.wornOff {
				combattest.CheckEvent(t, u.Events()[0], "antitoxin_wore_off", gid("drinker"))
			}
		})
	}
	over := advanceAntitoxin(t, drunk(t), time.Hour).Effect()
	again := advanceAntitoxin(t, over, time.Hour)
	if again.Effect() != over || len(again.Events()) != 0 {
		t.Errorf("an expired effect advanced to %+v with events %v, want unchanged and silent [FR-031]", again.Effect(), again.Events())
	}
	if _, err := drunk(t).Advance(-1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldElapsed, Value: -1}) {
		t.Errorf("Advance(-1) err = %v", err)
	}
	if _, err := (combat.AntitoxinEffect{}).Advance(1); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldEffect}) {
		t.Errorf("zero effect: %v", err)
	}
	if (combat.AntitoxinEffect{}).Active() {
		t.Error("the zero effect is active")
	}
}

// Requirement: per case
// Path: per case.
func TestAntitoxinAdvantageFor(t *testing.T) {
	t.Parallel()
	expired := advanceAntitoxin(t, drunk(t), time.Hour).Effect()
	cases := []struct {
		name, requirement, path string
		effect                  combat.AntitoxinEffect
		intent                  combat.PoisonedSaveIntent
		want                    combat.AntitoxinNothingReason
	}{
		{"avoid Poisoned", itemUseSpec + "FR-032 table 4.5a", "happy", drunk(t), combat.PoisonedSaveAvoid, 0},
		{"end Poisoned", itemUseSpec + "FR-032 table 4.5a", "happy", drunk(t), combat.PoisonedSaveEnd, 0},
		{"another save", itemUseSpec + "FR-032 table 4.5a", "unhappy (rule interaction)", drunk(t), combat.PoisonedSaveOther, combat.AntitoxinNothingNotAPoisonedSave},
		{"an unknown intent", itemUseSpec + "FR-032", "unhappy (invalid input)", drunk(t), 0, combat.AntitoxinNothingNotAPoisonedSave},
		{"expired", itemUseSpec + "FR-032 table 4.5a", "unhappy (state error)", expired, combat.PoisonedSaveAvoid, combat.AntitoxinNothingExpired},
		{"expired and another save", itemUseSpec + "FR-032", "unhappy (rule interaction)", expired, combat.PoisonedSaveOther, combat.AntitoxinNothingNotAPoisonedSave},
		{"the zero effect", itemUseSpec + "FR-032", "unhappy (state error)", combat.AntitoxinEffect{}, combat.PoisonedSaveEnd, combat.AntitoxinNothingExpired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch v := c.effect.AdvantageFor(c.intent).(type) {
			case combat.AntitoxinGivesAdvantage:
				if c.want != 0 || v.Circumstance() != (modifier.Advantage{}) {
					t.Errorf("advantage = %#v [%s]", v, c.requirement)
				}
			case combat.AntitoxinGivesNothing:
				if v.Reason() != c.want {
					t.Errorf("nothing: %v, want %v [%s]", v.Reason(), c.want, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-033, table 4.5a - drinking while active restarts the hour (a new effect at zero), and the Advantage does not stack with another source: the roll mode stays Advantage.
// Path: unhappy (rule interaction).
func TestAntitoxinRestartsAndDoesNotStack(t *testing.T) {
	t.Parallel()
	half := advanceAntitoxin(t, drunk(t), 30*time.Minute).Effect()
	fresh, _ := combat.AntitoxinRules{}.Drink(gid("drinker"))
	if half.Elapsed() != 30*time.Minute || fresh.Effect().Elapsed() != 0 || fresh.Effect() != drunk(t) {
		t.Errorf("a second drink gave %+v, want a restart at zero", fresh.Effect())
	}
	adv := half.AdvantageFor(combat.PoisonedSaveAvoid).(combat.AntitoxinGivesAdvantage)
	if got := modifier.Resolve(adv.Circumstance(), modifier.Advantage{}); got != modifier.WithAdvantage {
		t.Errorf("roll mode with two Advantages = %v, want advantage", got)
	}
	for _, p := range [][2]string{{combat.PoisonedSaveAvoid.String(), "avoid"}, {combat.PoisonedSaveEnd.String(), "end"}, {combat.PoisonedSaveOther.String(), "other"},
		{combat.PoisonedSaveIntent(0).String(), "unknown"}, {combat.AntitoxinNothingNotAPoisonedSave.String(), "not_a_poisoned_save"},
		{combat.AntitoxinNothingExpired.String(), "expired"}, {combat.AntitoxinNothingReason(0).String(), "unknown"}} {
		if p[0] != p[1] {
			t.Errorf("String() = %q, want %q", p[0], p[1])
		}
	}
}
