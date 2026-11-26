package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func resolveFall(t *testing.T, r combat.Roller, feet int, surface combat.FallSurface) (combat.ResolveFallResult, error) {
	t.Helper()
	uc, err := combat.NewResolveFall(r)
	if err != nil {
		t.Fatal(err)
	}
	return uc.Run(context.Background(), combat.ResolveFallCommand{Distance: combat.MustFallDistance(feet), Surface: surface})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-014, FR-121 - a fall distance is 0 feet or more; a negative one is an ErrInvalidInput error, and the largest int is accepted.
// Path: unhappy (invalid input).
func TestNewFallDistance(t *testing.T) {
	t.Parallel()
	for _, feet := range []int{0, 1, 9, math.MaxInt} {
		if d, err := combat.NewFallDistance(feet); err != nil || d.Feet() != feet {
			t.Errorf("NewFallDistance(%d) = %v, %v", feet, d, err)
		}
	}
	for _, feet := range []int{-1, math.MinInt} {
		if _, err := combat.NewFallDistance(feet); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewFallDistance(%d) = %v, want ErrInvalidInput", feet, err)
		}
	}
	defer func() {
		if err, _ := recover().(error); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("MustFallDistance panic = %v", err)
		}
	}()
	combat.MustFallDistance(-5)
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-015, FR-016, FR-017, FR-095 (table 4.3a) - one d6 per full 10 feet, at most 20, one request of exactly that many dice, no roll under 10 feet.
// Path: per case.
func TestResolveFallDiceCount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		feet, dice int
	}{{0, 0}, {9, 0}, {10, 1}, {19, 1}, {20, 2}, {199, 19}, {200, 20}, {201, 20}, {1000, 20}, {math.MaxInt, 20}}
	for _, c := range cases {
		roller := dicetest.NewMaximumRoller()
		res, err := resolveFall(t, roller, c.feet, nil)
		if err != nil {
			t.Fatal(err)
		}
		if c.dice == 0 {
			if res != (combat.FallWithoutDamage{}) || len(roller.Requests()) != 0 {
				t.Errorf("%d ft = %#v with %d requests, want FallWithoutDamage and no roll (FR-017)", c.feet, res, len(roller.Requests()))
			}
			continue
		}
		got := res.(combat.FallDamageRolled)
		want := dice.MustExpression(c.dice, dice.D6(), 0)
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != want || got.DiceCount() != c.dice ||
			got.Damage() != combat.MustDamage(6*c.dice, content.DamageBludgeoning) || got.Roll().Total() != 6*c.dice {
			t.Errorf("%d ft = %#v with requests %v, want %v once and %d Bludgeoning (FR-015, FR-016)", c.feet, got, reqs, want, 6*c.dice)
		}
		if combat.MustFallDistance(c.feet).Dice() != c.dice {
			t.Errorf("Dice(%d) = %d, want %d", c.feet, combat.MustFallDistance(c.feet).Dice(), c.dice)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-018 to FR-022 (table 4.3b) - halving is only for a landing check made in liquid; the damage stays the full roll; a harmless fall is FallWithoutDamage whatever the surface.
// Path: per case.
func TestResolveFallSurfaces(t *testing.T) {
	t.Parallel()
	liquid := func(l combat.LiquidLanding) combat.FallSurface { return combat.FallIntoLiquid{Landing: l} }
	cases := []struct {
		name, requirement, path string
		feet                    int
		surface                 combat.FallSurface
		half                    bool
		harmless                bool
	}{
		{"ground", "FR-018", "happy", 30, combat.FallOntoGround{}, false, false},
		{"nil surface is ground", "FR-018", "unhappy (empty and minimal)", 30, nil, false, false},
		{"liquid, check succeeds", "FR-019", "happy", 30, liquid(combat.LandingCheckMade{Outcome: d20.Success}), true, false},
		{"liquid, check fails", "FR-020", "unhappy (rule interaction)", 30, liquid(combat.LandingCheckMade{Outcome: d20.Failure}), false, false},
		{"liquid, no reaction", "FR-020", "unhappy (rule interaction)", 30, liquid(combat.NoLandingReaction{}), false, false},
		{"liquid, outcome not set", "FR-020", "unhappy (invalid input)", 30, liquid(combat.LandingCheckMade{}), false, false},
		{"liquid, nil landing", "FR-020", "unhappy (empty and minimal)", 30, liquid(nil), false, false},
		{"liquid, harmless fall", "FR-022", "unhappy (rule interaction)", 9, liquid(combat.LandingCheckMade{Outcome: d20.Success}), false, true},
		{"liquid, maximum dice", "FR-019", "unhappy (boundary)", 500, liquid(combat.LandingCheckMade{Outcome: d20.Success}), true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := resolveFall(t, dicetest.NewMaximumRoller(), c.feet, c.surface)
			if err != nil {
				t.Fatal(err)
			}
			if c.harmless {
				if res != (combat.FallWithoutDamage{}) {
					t.Errorf("= %#v, want FallWithoutDamage [%s]", res, c.requirement)
				}
				return
			}
			adj := res.(combat.FallDamageRolled).Adjustments()
			if c.half != (len(adj) == 1 && adj[0] == combat.HalfDamage{}) || (!c.half && len(adj) != 0) {
				t.Errorf("adjustments = %v, want half %v [%s]", adj, c.half, c.requirement)
			}
			if c.half {
				adj[0] = nil
			}
			if c.half && res.(combat.FallDamageRolled).Adjustments()[0] == nil {
				t.Error("Adjustments is not a copy")
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-021 - the landing check is published as DC 15 with Strength (Athletics) or Dexterity (Acrobatics); a new slice each call.
// Path: happy.
func TestFallLandingCheck(t *testing.T) {
	t.Parallel()
	var c combat.FallLandingCheck
	opts := c.Options()
	want := []combat.FallLandingOption{{Ability: ability.Strength, Skill: content.SkillAthletics}, {Ability: ability.Dexterity, Skill: content.SkillAcrobatics}}
	if c.DC() != 15 || len(opts) != 2 || opts[0] != want[0] || opts[1] != want[1] {
		t.Errorf("= DC %d options %v, want 15 and %v", c.DC(), opts, want)
	}
	opts[0] = combat.FallLandingOption{}
	if c.Options()[0] != want[0] {
		t.Error("Options is not a fresh slice")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-094, FR-095 - a fall roller that fails, panics or answers another expression is a *srderr.PortError and no result; a nil roller, an unknown surface and a cancelled context are errors.
// Path: unhappy (collaborator failure).
func TestResolveFallFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	for name, r := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := resolveFall(t, r, 30, nil)
		envPortFailure(t, err)
		if res != nil || name == "fails" && !errors.Is(err, cause) || name == "wrong expression" && !errors.Is(err, srderr.ErrContractViolation) {
			t.Errorf("%s: = %v, %v", name, res, err)
		}
	}
	if _, err := combat.NewResolveFall(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller = %v", err)
	}
	var zero combat.ResolveFall
	if _, err := zero.Run(context.Background(), combat.ResolveFallCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v", err)
	}
	uc, _ := combat.NewResolveFall(dicetest.NewMaximumRoller())
	if _, err := uc.Run(context.Background(), combat.ResolveFallCommand{Surface: badSurface{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unknown surface = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, combat.ResolveFallCommand{}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

// badSurface is not a variant of FallSurface the package knows: it cannot be built outside the package, so a nil-embedded interface stands in.
type badSurface struct{ combat.FallSurface }

func landAfterFall(t *testing.T, set combat.ConditionSet, damage int) (combat.LandAfterFallResult, error) {
	t.Helper()
	return combat.NewLandAfterFall().Run(context.Background(), combat.LandAfterFallCommand{Conditions: set, Target: subject(), DamageTaken: damage})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-023 to FR-026 (table 4.3c) - damage of 1 or more lands the creature Prone through ConditionSet.Apply, 0 leaves it standing with the set unchanged, an existing Prone or an Immunity is carried without an event, and a negative amount is an error.
// Path: per case.
func TestLandAfterFall(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		set                     combat.ConditionSet
		damage                  int
		standing                bool
		app                     string
		events                  []string
	}{
		{"damage 11", "FR-023", "happy", combat.ConditionSet{}, 11, false, "added", []string{"condition_applied"}},
		{"damage 1", "FR-023", "unhappy (boundary)", combat.ConditionSet{}, 1, false, "added", []string{"condition_applied"}},
		{"damage 0", "FR-024", "unhappy (boundary)", combat.ConditionSet{}, 0, true, "", []string{}},
		{"already Prone", "FR-025", "unhappy (rule interaction)", condSet(t, condConds(combat.Prone{})), 11, false, "held", []string{}},
		{"immune to Prone", "FR-025", "unhappy (rule interaction)", condSet(t, nil, content.ConditionProne), 11, false, "immune", []string{}},
		{"huge damage", "FR-023", "unhappy (boundary)", combat.ConditionSet{}, math.MaxInt, false, "added", []string{"condition_applied"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := landAfterFall(t, c.set, c.damage)
			if err != nil {
				t.Fatal(err)
			}
			if c.standing {
				if s, ok := res.(combat.FallLandedStanding); !ok || s.Conditions() != c.set {
					t.Errorf("= %#v, want standing with the set unchanged [%s]", res, c.requirement)
				}
				return
			}
			got := res.(combat.FallLandedProne)
			app := ""
			switch got.Application().(type) {
			case combat.ConditionAdded:
				app = "added"
			case combat.ConditionAlreadyHeld:
				app = "held"
			case combat.ConditionImmune:
				app = "immune"
			}
			if app != c.app || !equalStrings(envKinds(got.Events()), c.events) || got.Conditions() != got.Application().Set() {
				t.Errorf("= application %q events %v, want %q and %v [%s]", app, envKinds(got.Events()), c.app, c.events, c.requirement)
			}
		})
	}
	if _, err := landAfterFall(t, combat.ConditionSet{}, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("damage -1 = %v, want ErrInvalidInput (FR-026)", err)
	}
	if _, err := combat.NewLandAfterFall().Run(context.Background(), combat.LandAfterFallCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := combat.NewLandAfterFall().Run(cancelled, combat.LandAfterFallCommand{Target: subject()}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
