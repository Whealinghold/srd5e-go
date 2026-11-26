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
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// thrownIn is Acid thrown at a seen Medium Humanoid 20 feet away by a thrower with Dexterity +3
// and Proficiency Bonus +2, so the save DC is 13; the target's save has no modifier.
func thrownIn() combat.ResolveThrownItemInput {
	return combat.ResolveThrownItemInput{Kind: combat.ThrownAcid, Thrower: gid("thrower"), DexterityModifier: dexModifier(16), ProficiencyBonus: proficiency(1),
		Target: humanoid(), DistanceFeet: 20, Sight: combat.ThrownSightSeen, Cover: combat.ThrownCoverTargetable, Attempt: rollWith(0)}
}

func thrownRun(tb testing.TB, in combat.ResolveThrownItemInput, roller combat.Roller, opts ...combat.Option) (combat.ResolveThrownItemResult, error) {
	tb.Helper()
	cmd, err := combat.NewResolveThrownItemCommand(in)
	if err != nil {
		tb.Fatal(err)
	}
	u, err := combat.NewResolveThrownItem(roller, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return u.Run(context.Background(), cmd)
}

// Requirement: per case
// Path: per case.
func TestResolveThrownItemRun(t *testing.T) {
	t.Parallel()
	spec := itemUseSpec
	creature := func(size content.CreatureSize, typ content.CreatureType) combat.ThrownAtCreature {
		return combat.ThrownAtCreature{ID: gid("target"), Size: size, Type: typ}
	}
	cases := []struct {
		name, requirement, path string
		change                  func(*combat.ResolveThrownItemInput)
		faces                   []int
		dice                    []string
		result                  string
		effect                  combat.ThrownEffect
		consumed                bool
		saveRolled              bool
	}{
		{"acid, failed save", spec + "table 4.2a acid fail FR-009 FR-011 FR-019", "happy", func(*combat.ResolveThrownItemInput) {}, []int{12, 3, 4}, []string{"1d20", "2d6"},
			"damaged", combat.ThrownDamageDealt{}, true, true},
		{"acid, total equal to the DC saves", spec + "table 4.2a acid exact FR-009", "unhappy (boundary)", func(*combat.ResolveThrownItemInput) {}, []int{13}, []string{"1d20"},
			"saved", combat.ThrownNoEffect{}, true, true},
		{"acid at an object", spec + "table 4.2a acid object FR-006", "happy", func(in *combat.ResolveThrownItemInput) { in.Target = combat.ThrownAtObject{} }, []int{5, 1, 1}, []string{"1d20", "2d6"},
			"damaged", combat.ThrownDamageDealt{}, true, true},
		{"alchemist's fire, failed save", spec + "table 4.2a fire fail FR-013", "happy", func(in *combat.ResolveThrownItemInput) { in.Kind = combat.ThrownAlchemistsFire }, []int{3, 4}, []string{"1d20", "1d4"},
			"damaged_and_burning", combat.ThrownDamageAndBurningStarted{}, true, true},
		{"alchemist's fire, save", spec + "table 4.2a fire save FR-013 FR-019", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) { in.Kind = combat.ThrownAlchemistsFire }, []int{18}, []string{"1d20"},
			"saved", combat.ThrownNoEffect{}, true, true},
		{"holy water at a fiend", spec + "table 4.2a holy fiend FR-011", "happy", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Target = combat.ThrownHolyWater, creature(content.SizeMedium, content.CreatureTypeFiend)
		}, []int{4, 1, 1}, []string{"1d20", "2d8"}, "damaged", combat.ThrownDamageDealt{}, true, true},
		{"holy water at an undead", spec + "FR-011", "happy", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Target = combat.ThrownHolyWater, creature(content.SizeMedium, content.CreatureTypeUndead)
		}, []int{4, 8, 8}, []string{"1d20", "2d8"}, "damaged", combat.ThrownDamageDealt{}, true, true},
		{"holy water at a fiend that saves", spec + "FR-011", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Target = combat.ThrownHolyWater, creature(content.SizeMedium, content.CreatureTypeFiend)
		}, []int{14}, []string{"1d20"}, "saved", combat.ThrownNoEffect{}, true, true},
		{"holy water at a humanoid", spec + "table 4.2a holy human FR-012", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) { in.Kind = combat.ThrownHolyWater }, nil, []string{},
			"no_effect", combat.ThrownNoEffect{}, true, false},
		{"oil, unseen", spec + "table 4.2a oil fail FR-008 FR-016", "happy", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Sight = combat.ThrownOil, combat.ThrownSightUnseen
		}, []int{2}, []string{"1d20"},
			"covered_in_oil", combat.ThrownOilCovered{}, true, true},
		{"net, Large creature", spec + "table 4.2a net fail FR-014 FR-019", "happy", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.DistanceFeet, in.Target = combat.ThrownNet, 15, creature(content.SizeLarge, content.CreatureTypeBeast)
		}, []int{6}, []string{"1d20"}, "restrained", combat.ThrownRestrained{}, false, true},
		{"net, Huge creature", spec + "table 4.2a net huge FR-014", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.DistanceFeet, in.Target = combat.ThrownNet, 15, creature(content.SizeHuge, content.CreatureTypeBeast)
		}, nil, []string{}, "no_effect", combat.ThrownNoEffect{}, false, false},
		{"net, Gargantuan creature", spec + "FR-014", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Target = combat.ThrownNet, creature(content.SizeGargantuan, content.CreatureTypeBeast)
			in.DistanceFeet = 0
		}, nil, []string{}, "no_effect", combat.ThrownNoEffect{}, false, false},
		{"natural 20 below the DC fails", spec + "table 4.2a nat 20 fail FR-009", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) { in.Attempt = rollWith(-9) }, []int{20, 2, 2}, []string{"1d20", "2d6"},
			"damaged", combat.ThrownDamageDealt{}, true, true},
		{"natural 1 meeting the DC saves", spec + "table 4.2a nat 1 success FR-009", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) { in.Attempt = rollWith(12) }, []int{1}, []string{"1d20"},
			"saved", combat.ThrownNoEffect{}, true, true},
		{"automatic failure draws no d20", spec + "table 4.2a forced fail FR-009", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) { in.Attempt = d20.AutomaticFailure{} }, []int{6, 6}, []string{"2d6"},
			"damaged", combat.ThrownDamageDealt{}, true, false},
		{"chosen failure draws no d20", spec + "FR-009", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) { in.Attempt = d20.ChooseToFail{} }, []int{1, 1}, []string{"2d6"},
			"damaged", combat.ThrownDamageDealt{}, true, false},
		{"disadvantage keeps the lower die", spec + "table 4.2a disadvantage FR-009 FR-015", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) { in.Attempt = rollWith(0, modifier.Disadvantage{}) },
			[]int{17, 9, 2, 2}, []string{"2d20", "2d6"}, "damaged", combat.ThrownDamageDealt{}, true, true},
		{"advantage and disadvantage cancel", spec + "FR-009", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) {
			in.Attempt = rollWith(0, modifier.Advantage{}, modifier.Disadvantage{})
		}, []int{5, 2, 2}, []string{"1d20", "2d6"}, "damaged", combat.ThrownDamageDealt{}, true, true},
		{"a bonus on the save is the caller's", spec + "FR-010", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) { in.Attempt = rollWith(2) }, []int{11}, []string{"1d20"},
			"saved", combat.ThrownNoEffect{}, true, true},
		{"20 feet is in range", spec + "FR-006 table 4.2a net range", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) { in.DistanceFeet = 20 }, []int{5, 1, 1}, []string{"1d20", "2d6"},
			"damaged", combat.ThrownDamageDealt{}, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := thrownIn()
			c.change(&in)
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := thrownRun(t, in, roller)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if !slices.Equal(requested(roller), c.dice) {
				t.Errorf("dice requested = %v, want %v [%s]", requested(roller), c.dice, c.requirement)
			}
			r, ok := res.(combat.ThrownItemResolved)
			if !ok {
				t.Fatalf("result = %#v, want resolved [%s]", res, c.requirement)
			}
			if reflect.TypeOf(r.Effect()) != reflect.TypeOf(c.effect) || r.Item() != in.Kind || r.Cost() != combat.ItemActionCostReplacesOneAttack {
				t.Errorf("effect = %#v, item %v, cost %v, want %T [%s]", r.Effect(), r.Item(), r.Cost(), c.effect, c.requirement)
			}
			if _, isConsumed := r.Consumption().(combat.ItemConsumed); isConsumed != c.consumed {
				t.Errorf("consumption = %#v, want consumed %v [%s]", r.Consumption(), c.consumed, c.requirement)
			}
			switch s := r.Save().(type) {
			case combat.ThrownSaveRolled:
				if !c.saveRolled || s.DifficultyClass() != 13 {
					t.Errorf("save = %#v, want DC 13 rolled=%v", s, c.saveRolled)
				}
			case combat.ThrownSaveForced:
				if c.saveRolled || s.DifficultyClass() != 13 || s.Succeeded() || s.Result().Kind() != d20.SavingThrow || s.Result().Reason() == 0 {
					t.Errorf("save = %#v, want a forced failure at DC 13", s)
				}
			case combat.ThrownSaveSkipped:
				if c.saveRolled || s.Reason() == 0 || s.Succeeded() || r.Effect().(combat.ThrownNoEffect).Reason() != s.Reason() {
					t.Errorf("save = %#v, want skipped", s)
				}
			}
			ev := r.Events()
			if len(ev) != 1 {
				t.Fatalf("events = %v, want one thrown_item_used", ev)
			}
			want := []combat.CombatantID{gid("thrower"), gid("target")}
			if _, obj := in.Target.(combat.ThrownAtObject); obj {
				want = want[:1]
			}
			combattest.CheckEvent(t, ev[0], "thrown_item_used", want...)
			if ev[0].(combat.ThrownItemUsed).Item() != in.Kind {
				t.Error("event item differs")
			}
			combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{r.Events()[0]} })
			ev[0] = nil
			if r.Events()[0] == nil {
				t.Error("Events returns the stored slice")
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-011, FR-013, FR-014, FR-016 - the dice rolled land in the effect: the damage is the roll total in the kind's type, Oil gives a fresh wet coat, the Net the restraint; the DC is carried in the save.
// Path: happy.
func TestResolveThrownItemEffectValues(t *testing.T) {
	t.Parallel()
	in := thrownIn()
	res, _ := thrownRun(t, in, dicetest.NewFixedRoller(12, 3, 4))
	r := res.(combat.ThrownItemResolved)
	if d := r.Effect().(combat.ThrownDamageDealt).Damage(); d != combat.MustDamage(7, content.DamageAcid) {
		t.Errorf("damage = %+v, want 7 Acid", d)
	}
	s := r.Save().(combat.ThrownSaveRolled)
	if s.Result().Outcome() != d20.Failure || s.Result().Roll().Natural() != 12 {
		t.Errorf("save result = %+v", s.Result())
	}
	in.Kind = combat.ThrownAlchemistsFire
	res, _ = thrownRun(t, in, dicetest.NewFixedRoller(3, 4))
	if d := res.(combat.ThrownItemResolved).Effect().(combat.ThrownDamageAndBurningStarted).Damage(); d != combat.MustDamage(4, content.DamageFire) {
		t.Errorf("fire damage = %+v", d)
	}
	in.Kind, in.Target = combat.ThrownHolyWater, combat.ThrownAtCreature{ID: gid("target"), Size: content.SizeMedium, Type: content.CreatureTypeFiend}
	res, _ = thrownRun(t, in, dicetest.NewFixedRoller(1, 8, 8))
	if d := res.(combat.ThrownItemResolved).Effect().(combat.ThrownDamageDealt).Damage(); d != combat.MustDamage(16, content.DamageRadiant) {
		t.Errorf("holy water damage = %+v", d)
	}
	in = thrownIn()
	in.Kind = combat.ThrownOil
	res, _ = thrownRun(t, in, dicetest.NewFixedRoller(1))
	if c := res.(combat.ThrownItemResolved).Effect().(combat.ThrownOilCovered).Coat(); c != combat.NewOilCoat() {
		t.Errorf("coat = %+v, want a fresh wet coat [FR-016]", c)
	}
	in.Kind, in.DistanceFeet = combat.ThrownNet, 15
	res, _ = thrownRun(t, in, dicetest.NewFixedRoller(1))
	if res.(combat.ThrownItemResolved).Effect().(combat.ThrownRestrained).Restraint().EscapeCheck().DifficultyClass() != 10 {
		t.Error("the Net's escape DC is not 10 [FR-014]")
	}
}

// Requirement: per case
// Path: per case.
func TestResolveThrownItemNotPossible(t *testing.T) {
	t.Parallel()
	spec := itemUseSpec
	cases := []struct {
		name, requirement, path string
		change                  func(*combat.ResolveThrownItemInput)
		want                    combat.ThrownNotPossibleReason
	}{
		{"holy water at an object", spec + "table 4.2a holy object FR-006", "unhappy (invalid input)", func(in *combat.ResolveThrownItemInput) {
			in.Kind, in.Target = combat.ThrownHolyWater, combat.ThrownAtObject{}
		}, combat.ThrownReasonTargetNotACreature},
		{"total cover", spec + "table 4.2a total cover FR-006", "unhappy (state error)", func(in *combat.ResolveThrownItemInput) { in.Cover = combat.ThrownCoverTotal }, combat.ThrownReasonTotalCover},
		{"21 feet", spec + "table 4.2a net range FR-006", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) { in.DistanceFeet = 21 }, combat.ThrownReasonBeyondRange},
		{"net at 16 feet", spec + "table 4.2a net range FR-006", "unhappy (boundary)", func(in *combat.ResolveThrownItemInput) { in.Kind, in.DistanceFeet = combat.ThrownNet, 16 }, combat.ThrownReasonBeyondRange},
		{"unseen acid", spec + "FR-006 FR-008", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) { in.Sight = combat.ThrownSightUnseen }, combat.ThrownReasonTargetNotSeen},
		{"two faults at once: cover first", spec + "FR-006", "unhappy (rule interaction)", func(in *combat.ResolveThrownItemInput) {
			in.Cover, in.DistanceFeet, in.Sight = combat.ThrownCoverTotal, 99, combat.ThrownSightUnseen
		}, combat.ThrownReasonTotalCover},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := thrownIn()
			c.change(&in)
			roller := dicetest.NewFixedRoller(5, 1, 1)
			res, err := thrownRun(t, in, roller)
			if got, ok := res.(combat.ThrownItemNotPossible); err != nil || !ok || got.Reason != c.want {
				t.Fatalf("result = %#v, %v, want not possible %v [%s]", res, err, c.want, c.requirement)
			}
			if n := len(roller.Requests()); n != 0 {
				t.Errorf("%d dice requested on a refusal, want none [FR-006]", n)
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-007 - a negative DistanceFeet makes the constructor return a typed error carrying the field, MustResolveThrownItemCommand panics with it, and every other invalid field is a typed error too; nothing panics.
// Path: unhappy (invalid input).
func TestNewResolveThrownItemCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		change func(*combat.ResolveThrownItemInput)
		field  combat.ItemUseField
		value  int64
	}{
		{"negative distance", func(in *combat.ResolveThrownItemInput) { in.DistanceFeet = -1 }, combat.ItemUseFieldDistanceFeet, -1},
		{"minimum distance", func(in *combat.ResolveThrownItemInput) { in.DistanceFeet = math.MinInt }, combat.ItemUseFieldDistanceFeet, math.MinInt},
		{"kind", func(in *combat.ResolveThrownItemInput) { in.Kind = 0 }, combat.ItemUseFieldKind, 0},
		{"thrower", func(in *combat.ResolveThrownItemInput) { in.Thrower = "" }, combat.ItemUseFieldThrower, 0},
		{"target", func(in *combat.ResolveThrownItemInput) { in.Target = nil }, combat.ItemUseFieldTarget, 0},
		{"sight", func(in *combat.ResolveThrownItemInput) { in.Sight = 0 }, combat.ItemUseFieldSight, 0},
		{"cover", func(in *combat.ResolveThrownItemInput) { in.Cover = 0 }, combat.ItemUseFieldCover, 0},
		{"nil attempt", func(in *combat.ResolveThrownItemInput) { in.Attempt = nil }, combat.ItemUseFieldAttempt, 0},
	}
	for _, c := range cases {
		in := thrownIn()
		c.change(&in)
		_, err := combat.NewResolveThrownItemCommand(in)
		var ue *combat.ItemUseError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ue) || ue.Field != c.field || ue.Value != c.value {
			t.Errorf("%s: err = %v, want field %v value %d", c.name, err, c.field, c.value)
		}
		func() {
			defer func() {
				if r, _ := recover().(error); !errors.Is(r, &combat.ItemUseError{Field: c.field, Value: c.value}) {
					t.Errorf("%s: Must panicked with %v", c.name, r)
				}
			}()
			combat.MustResolveThrownItemCommand(in)
		}()
	}
	in := thrownIn()
	in.Attempt = d20.RollAttempt{Circumstances: []modifier.RollCircumstance{modifier.Advantage{}}}
	cmd, _ := combat.NewResolveThrownItemCommand(in)
	in.Attempt.(d20.RollAttempt).Circumstances[0] = modifier.Disadvantage{}
	roller := dicetest.NewFixedRoller(4, 5, 1, 1)
	u, _ := combat.NewResolveThrownItem(roller)
	if _, err := u.Run(context.Background(), cmd); err != nil || !slices.Equal(requested(roller), []string{"2d20", "2d6"}) {
		t.Errorf("the command kept the caller's circumstances: dice %v, err %v [FR-075]", requested(roller), err)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-071, FR-074, FR-075 - the zero use case or command and a nil roller are invalid, a cancelled ctx is returned unwrapped before any die, a failing, panicking or wrong-expression roller is one *srderr.PortError with no result, and equal inputs with equal rolls give equal results.
// Path: unhappy (invalid input, cancellation, failure of collaborator, determinism).
func TestResolveThrownItemFailures(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewResolveThrownItem(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	cmd := combat.MustResolveThrownItemCommand(thrownIn())
	good, _ := combat.NewResolveThrownItem(dicetest.NewFixedRoller(1))
	if _, err := good.Run(context.Background(), combat.ResolveThrownItemCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v", err)
	}
	if _, err := (combat.ResolveThrownItem{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	roller := dicetest.NewFixedRoller(1)
	cancelled, _ := combat.NewResolveThrownItem(roller)
	if res, err := cancelled.Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil || len(roller.Requests()) != 0 {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	for name, rl := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		u, _ := combat.NewResolveThrownItem(rl)
		res, err := u.Run(context.Background(), cmd)
		if res != nil {
			t.Errorf("%s: partial result %#v", name, res)
		}
		envPortFailure(t, err)
	}
	// The damage roll fails after the save was made: still one port error, no result.
	failing := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		if e.Die() == dice.D20() {
			return dicetest.NewFixedRoller(1).Roll(e)
		}
		return dice.Result{}, errors.New("down")
	})
	u, _ := combat.NewResolveThrownItem(failing)
	if res, err := u.Run(context.Background(), cmd); res != nil {
		t.Errorf("damage roll failure left %#v", res)
	} else {
		envPortFailure(t, err)
	}
	a, _ := thrownRun(t, thrownIn(), dicetest.NewFixedRoller(12, 3, 4))
	b, _ := thrownRun(t, thrownIn(), dicetest.NewFixedRoller(12, 3, 4))
	if !reflect.DeepEqual(a, b) {
		t.Errorf("equal inputs gave %#v and %#v [FR-075]", a, b)
	}
}
