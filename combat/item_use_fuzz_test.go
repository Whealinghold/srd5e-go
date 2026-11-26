package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzNewResolveThrownItemCommand feeds the constructors host numbers. It asserts properties:
// nothing panics, a rejected value is an *ItemUseError matching srderr.ErrInvalidInput, and a
// command that was built runs without error and never refuses a throw within every limit.
func FuzzNewResolveThrownItemCommand(f *testing.F) {
	for _, s := range [][6]int{{1, 20, 1, 1, 4, 10}, {5, 15, 1, 1, 8, 5}, {3, -1, 2, 2, 0, 0}, {0, 0, 0, 0, 99, 99}} {
		f.Add(s[0], s[1], s[2], s[3], s[4], s[5])
	}
	f.Add(math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, math.MinInt)
	f.Fuzz(func(t *testing.T, kind, distance, sight, cover, size, creatureType int) {
		in := thrownIn()
		in.Kind, in.DistanceFeet, in.Sight, in.Cover = combat.ThrownItemKind(kind), distance, combat.ThrownSight(sight), combat.ThrownCover(cover)
		in.Target = combat.ThrownAtCreature{ID: gid("target"), Size: content.CreatureSize(size), Type: content.CreatureType(creatureType)}
		cmd, err := combat.NewResolveThrownItemCommand(in)
		if err != nil {
			var ue *combat.ItemUseError
			if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ue) {
				t.Fatalf("err = %v, want an *ItemUseError matching ErrInvalidInput", err)
			}
			return
		}
		u, _ := combat.NewResolveThrownItem(dicetest.NewMaximumRoller())
		if _, err := u.Run(context.Background(), cmd); err != nil {
			t.Fatalf("Run on a built command: %v", err)
		}
	})
}

// FuzzItemUseNumbers feeds the constructors and the clocks of the items host numbers: a kit
// size, a number of arrows, a duration and a damage amount. It asserts properties: nothing
// panics, and an accepted value stays within its stated range.
func FuzzItemUseNumbers(f *testing.F) {
	f.Add(10, 3, int64(time.Minute), 4)
	f.Add(-1, 0, int64(-1), -1)
	f.Add(math.MaxInt, math.MinInt, int64(math.MaxInt64), math.MaxInt)
	f.Add(0, 4, int64(math.MinInt64), math.MinInt)
	f.Fuzz(func(t *testing.T, uses, pieces int, d int64, damage int) {
		if k, err := combat.NewHealersKit(uses); err == nil && (k.UsesRemaining() < 0 || k.UsesRemaining() > 10) {
			t.Fatalf("kit of %d accepted", k.UsesRemaining())
		}
		if app, err := (combat.BasicPoisonRules{}).ApplyToAmmunition(gid("a"), pieces); err == nil && (len(app.Coatings()) != pieces || pieces < 1 || pieces > 3) {
			t.Fatalf("%d pieces accepted as %d coatings", pieces, len(app.Coatings()))
		}
		dur := time.Duration(d)
		coat, err := combat.NewOilCoat().Advance(dur)
		if err == nil && (dur < 0 || coat.Elapsed() < dur) {
			t.Fatalf("Advance(%v) = %v", dur, coat.Elapsed())
		}
		if err == nil {
			if _, err := coat.OnFireDamage(damage); (err == nil) != (damage >= 0) {
				t.Fatalf("OnFireDamage(%d): %v", damage, err)
			}
		}
		weapon, _ := (combat.BasicPoisonRules{}).ApplyToWeapon(gid("a"))
		if u, err := weapon.Coatings()[0].AfterWeaponDamage(content.DamagePiercing, damage); (err == nil) != (damage >= 0) || (err == nil && damage >= 1 && u.Coating().State() != combat.BasicPoisonDelivered) {
			t.Fatalf("AfterWeaponDamage(%d) = %v, %v", damage, u.Coating().State(), err)
		}
		drink, _ := (combat.AntitoxinRules{}).Drink(gid("a"))
		if u, err := drink.Effect().Advance(dur); err == nil && (dur < 0 || u.Effect().Active() != (dur < time.Hour)) {
			t.Fatalf("Antitoxin Advance(%v) active = %v", dur, u.Effect().Active())
		}
	})
}
