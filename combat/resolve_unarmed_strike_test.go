package combat_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

const srdUnarmed = `"Unarmed Strike", p. 190`

// Requirement: per case
// Path: per case.
func TestResolveUnarmedStrikeDamageOption(t *testing.T) {
	t.Parallel()
	dmg := func(threshold int, cs ...modifier.RollCircumstance) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeDamageOption{ArmorClass: 15, Circumstances: cs, CriticalThreshold: threshold}
	}
	cases := []struct {
		name, requirement, path string
		abilityMod              int
		option                  combat.UnarmedStrikeOption
		auto                    combat.AutomaticCritical
		faces                   []int
		total                   int
		hit, crit               bool
		damage                  []int
	}{
		{"Strength +3, Proficiency +2", `FR-065, FR-067; ` + srdUnarmed, "happy", 3, dmg(0), 0, []int{12}, 17, true, false, []int{4}},
		{"Strength 0: bonus +2, damage 1", `FR-067, FR-065`, "unhappy (boundary)", 0, dmg(0), 0, []int{13}, 15, true, false, []int{1}},
		{"Strength -1: damage 0", `FR-067`, "unhappy (boundary)", -1, dmg(0), 0, []int{14}, 15, true, false, []int{0}},
		{"Strength -5: damage 0, not -4", `FR-067, FR-055`, "unhappy (boundary)", -5, dmg(0), 0, []int{20}, 17, true, true, []int{0}},
		{"Strength +4: damage 5", `FR-067`, "unhappy (boundary)", 4, dmg(0), 0, []int{10}, 16, true, false, []int{5}},
		{"natural 20: damage unchanged", `FR-069, FR-059; "Critical Hits", p. 16`, "unhappy (boundary)", 3, dmg(0), 0, []int{20}, 25, true, true, []int{4}},
		{"automatic Critical Hit: damage unchanged", `FR-069, FR-090`, "unhappy (rule interaction)", 3, dmg(0), combat.AutomaticCriticalAutomatic, []int{12}, 17, true, true, []int{4}},
		{"miss", `FR-068`, "unhappy (boundary)", 3, dmg(0), 0, []int{9}, 14, false, false, nil},
		{"Disadvantage", `FR-068; "Advantage/Disadvantage", pp. 7-8`, "unhappy (rule interaction)", 3, dmg(0, disadvantage), 0, []int{18, 2}, 7, false, false, nil},
		{"threshold 19", `FR-134, FR-068`, "unhappy (boundary)", 3, dmg(19), 0, []int{19}, 24, true, true, []int{4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := unarmedParams(c.option)
			p.AbilityModifier, p.AutomaticCritical = abilityMod(c.abilityMod), c.auto
			res, reqs := runUnarmed(t, unarmedCommand(t, p), c.faces...)
			r, ok := res.(combat.UnarmedStrikeDamageResolved)
			if !ok {
				t.Fatalf("result = %#v [%s]", res, c.requirement)
			}
			var got []int
			for _, d := range r.Damage() {
				got = append(got, d.Amount())
				if d.Type() != content.DamageBludgeoning {
					t.Errorf("damage type = %v, want Bludgeoning [FR-067]", d.Type())
				}
			}
			if r.Attack().Total() != c.total || r.Hit() != c.hit || r.CriticalHit() != c.crit || !reflect.DeepEqual(got, c.damage) || len(reqs) != 1 {
				t.Errorf("total %d hit %v crit %v damage %v requests %v, want %d %v %v %v and only the d20 (no damage dice) [%s]",
					r.Attack().Total(), r.Hit(), r.CriticalHit(), got, reqs, c.total, c.hit, c.crit, c.damage, c.requirement)
			}
			if !r.Attack().Modifiers().Equal(combat.UnarmedStrikeModifiers(abilityMod(c.abilityMod), profBonus(2))) {
				t.Errorf("the roll modifiers are not UnarmedStrikeModifiers' [FR-124]")
			}
			if r.HitKind() != map[bool]combat.HitKind{true: combat.CriticalHit, false: combat.NormalHit}[c.crit] {
				t.Errorf("HitKind = %v [FR-114]", r.HitKind())
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-129, FR-065, FR-067, FR-074, FR-128 - the roll bonus, the damage and both DCs use the one
// ability modifier the command carries: a caller supplying Dexterity +4 with Proficiency +2 gets bonus +6, damage 5 and DC 14.
// Path: per case.
func TestResolveUnarmedStrikeUsesTheSuppliedAbilityModifier(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		mod, pb, bonus, damage, dc int
	}{{4, 2, 6, 5, 14}, {3, 2, 5, 4, 13}, {-1, 2, 1, 0, 9}} {
		p := unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: 15})
		p.AbilityModifier = abilityMod(c.mod)
		res, _ := runUnarmed(t, unarmedCommand(t, p), 15)
		r := res.(combat.UnarmedStrikeDamageResolved)
		if r.Attack().Total()-15 != c.bonus || r.Damage()[0].Amount() != c.damage {
			t.Errorf("modifier %d: bonus %d damage %d, want %d and %d [FR-129]", c.mod, r.Attack().Total()-15, r.Damage()[0].Amount(), c.bonus, c.damage)
		}
		for _, opt := range []combat.UnarmedStrikeOption{
			combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: rollOf(0)},
			combat.UnarmedStrikeShoveOption{Effect: combat.PushFiveFeet, SaveAbility: ability.Strength, Attempt: rollOf(0)},
		} {
			p.Option = opt
			res, _ := runUnarmed(t, unarmedCommand(t, p), 10)
			var dc d20.DifficultyClass
			switch v := res.(type) {
			case combat.UnarmedStrikeGrappleResolved:
				dc = v.DC()
			case combat.UnarmedStrikeShoveResolved:
				dc = v.DC()
			}
			if int(dc) != c.dc {
				t.Errorf("modifier %d: DC %d, want %d [FR-074, FR-128]", c.mod, dc, c.dc)
			}
		}
	}
}

// Requirement: per case
// Path: per case.
func TestResolveUnarmedStrikeSaves(t *testing.T) {
	t.Parallel()
	grapple := func(a ability.Ability, at d20.SaveAttempt) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeGrappleOption{SaveAbility: a, Attempt: at}
	}
	shove := func(e combat.ShoveEffect, a ability.Ability, at d20.SaveAttempt) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeShoveOption{Effect: e, SaveAbility: a, Attempt: at}
	}
	half := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	tq := []combat.CoverSource{{Cover: combat.ThreeQuartersCover, AttackOriginatesOppositeSide: true}}
	cases := []struct {
		name, requirement, path string
		option                  combat.UnarmedStrikeOption
		cover                   []combat.CoverSource
		faces                   []int
		requests                []string
		mods                    d20.RollModifiers
		total                   int
		failed                  bool
	}{
		{"save equals the DC (13)", `FR-076, FR-094; "D20 Tests", p. 6`, "unhappy (boundary)", grapple(ability.Strength, rollOf(3)), nil, []int{10}, []string{"1d20"}, mods(3, 0), 13, false},
		{"save one below the DC", `FR-076`, "unhappy (boundary)", grapple(ability.Strength, rollOf(3)), nil, []int{9}, []string{"1d20"}, mods(3, 0), 12, true},
		{"Shove save equals the DC", `FR-076, FR-074`, "unhappy (boundary)", shove(combat.GiveProne, ability.Strength, rollOf(3)), nil, []int{10}, []string{"1d20"}, mods(3, 0), 13, false},
		{"Shove save one below the DC", `FR-076, FR-073`, "happy", shove(combat.PushFiveFeet, ability.Dexterity, rollOf(3)), nil, []int{9}, []string{"1d20"}, mods(3, 0), 12, true},
		{"Dexterity save behind Half Cover gains +2", `FR-092, FR-094; ` + srdCover, "unhappy (rule interaction)", grapple(ability.Dexterity, rollOf(1)), half, []int{10}, []string{"1d20"}, mods(1, 0, 2), 13, false},
		{"Shove Dexterity save behind Three-Quarters gains +5", `FR-092`, "unhappy (rule interaction)", shove(combat.PushFiveFeet, ability.Dexterity, rollOf(0)), tq, []int{8}, []string{"1d20"}, mods(0, 0, 5), 13, false},
		{"Strength save behind Half Cover gains nothing", `FR-092`, "unhappy (rule interaction)", grapple(ability.Strength, rollOf(1)), half, []int{10}, []string{"1d20"}, mods(1, 0), 11, true},
		{"a caller's +1 and cover once", `FR-094`, "unhappy (rule interaction)", grapple(ability.Dexterity, d20.RollAttempt{Modifiers: mods(0, 0, 1)}), half, []int{10}, []string{"1d20"}, mods(0, 0, 1, 2), 13, false},
		{"Restrained target: Disadvantage supplied", `FR-094; "Restrained", p. 187`, "unhappy (rule interaction)", grapple(ability.Dexterity, rollOf(0, disadvantage)), nil, []int{15, 4}, []string{"2d20"}, mods(0, 0), 4, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := unarmedParams(c.option)
			p.Cover = c.cover
			res, reqs := runUnarmed(t, unarmedCommand(t, p), c.faces...)
			var save combat.UnarmedStrikeSave
			switch v := res.(type) {
			case combat.UnarmedStrikeGrappleResolved:
				save = v.Save()
				if v.Grappled() != c.failed {
					t.Errorf("Grappled = %v, want %v [%s]", v.Grappled(), c.failed, c.requirement)
				}
			case combat.UnarmedStrikeShoveResolved:
				save = v.Save()
				if v.Applied() != c.failed || v.Effect() != c.option.(combat.UnarmedStrikeShoveOption).Effect {
					t.Errorf("Applied = %v effect %v, want %v [%s]", v.Applied(), v.Effect(), c.failed, c.requirement)
				}
			default:
				t.Fatalf("result = %#v", res)
			}
			rolled := save.(combat.UnarmedStrikeSaveRolled).Result()
			want, err := d20.ResolveCheck(d20.SavingThrow, rolled.Roll(), c.mods, 13)
			if err != nil || !rolled.Equal(want) || rolled.Total() != c.total || !reflect.DeepEqual(reqs, c.requests) || (rolled.Outcome() == d20.Failure) != c.failed {
				t.Errorf("save = %+v requests %v, want d20's %+v with total %d [%s]", rolled, reqs, want, c.total, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-095, FR-094 - ChooseToFail and AutomaticFailure draw no die, resolve through
// d20.FailWithoutRoll to a failed save with the reason, and get no cover modifier.
// Path: per case.
func TestResolveUnarmedStrikeForcedFailures(t *testing.T) {
	t.Parallel()
	half := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	cases := []struct {
		name    string
		option  combat.UnarmedStrikeOption
		reason  d20.ForcedFailureReason
		grapple bool
	}{
		{"Grapple, chosen to fail", combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Dexterity, Attempt: d20.ChooseToFail{}}, d20.ChosenToFail, true},
		{"Shove, automatic failure", combat.UnarmedStrikeShoveOption{Effect: combat.GiveProne, SaveAbility: ability.Dexterity, Attempt: d20.AutomaticFailure{}}, d20.Automatic, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := unarmedParams(c.option)
			p.Cover = half
			res, reqs := runUnarmed(t, unarmedCommand(t, p))
			want, _ := d20.FailWithoutRoll(d20.SavingThrow, c.reason)
			var save combat.UnarmedStrikeSave
			var failed bool
			switch v := res.(type) {
			case combat.UnarmedStrikeGrappleResolved:
				save, failed = v.Save(), v.Grappled()
			case combat.UnarmedStrikeShoveResolved:
				save, failed = v.Save(), v.Applied()
			}
			f, ok := save.(combat.UnarmedStrikeSaveForced)
			if !ok || f.Result() != want || !failed || save.Outcome() != d20.Failure || len(reqs) != 0 {
				t.Errorf("save = %#v failed %v requests %v, want the forced failure %v and no die [FR-095]", save, failed, reqs, want)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestResolveUnarmedStrikeNotPossible(t *testing.T) {
	t.Parallel()
	total := []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	g := combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: rollOf(0)}
	s := combat.UnarmedStrikeShoveOption{Effect: combat.PushFiveFeet, SaveAbility: ability.Strength, Attempt: rollOf(0)}
	d := combat.UnarmedStrikeDamageOption{ArmorClass: 15}
	with := func(o combat.UnarmedStrikeOption, edit func(*combat.ResolveUnarmedStrikeParams)) combat.ResolveUnarmedStrikeParams {
		p := unarmedParams(o)
		edit(&p)
		return p
	}
	size := func(a, tg content.CreatureSize) func(*combat.ResolveUnarmedStrikeParams) {
		return func(p *combat.ResolveUnarmedStrikeParams) { p.AttackerSize, p.TargetSize = a, tg }
	}
	cases := []struct {
		name, requirement, path string
		p                       combat.ResolveUnarmedStrikeParams
		want                    combat.AttackNotPossibleReason
		possible                bool
	}{
		{"target at 6 feet", `FR-064; ` + srdUnarmed, "unhappy (boundary)", with(d, func(p *combat.ResolveUnarmedStrikeParams) { p.Distance = 6 }), combat.AttackNotPossibleReasonBeyondReach, false},
		{"target at 5 feet", `FR-063, FR-106`, "unhappy (boundary)", with(d, func(p *combat.ResolveUnarmedStrikeParams) {}), 0, true},
		{"Total Cover, Damage option", `FR-019`, "unhappy (state error)", with(d, func(p *combat.ResolveUnarmedStrikeParams) { p.Cover = total }), combat.AttackNotPossibleReasonTotalCover, false},
		{"Total Cover, Grapple", `FR-093`, "unhappy (state error)", with(g, func(p *combat.ResolveUnarmedStrikeParams) { p.Cover = total }), combat.AttackNotPossibleReasonTotalCover, false},
		{"Total Cover, Shove", `FR-093`, "unhappy (state error)", with(s, func(p *combat.ResolveUnarmedStrikeParams) { p.Cover = total }), combat.AttackNotPossibleReasonTotalCover, false},
		{"Grapple: one size larger", `FR-128`, "unhappy (boundary)", with(g, size(content.SizeMedium, content.SizeLarge)), 0, true},
		{"Grapple: smaller target", `FR-128`, "happy", with(g, size(content.SizeMedium, content.SizeTiny)), 0, true},
		{"Grapple: two sizes larger", `FR-128`, "unhappy (boundary)", with(g, size(content.SizeMedium, content.SizeHuge)), combat.AttackNotPossibleReasonTargetTooLarge, false},
		{"Grapple: no free hand", `FR-128`, "unhappy (state error)", with(g, func(p *combat.ResolveUnarmedStrikeParams) { p.Hand = combat.HandAvailabilityNoHandFree }), combat.AttackNotPossibleReasonNoFreeHand, false},
		{"Grapple: too large and no hand reports too large", `FR-105`, "unhappy (rule interaction)", with(g, func(p *combat.ResolveUnarmedStrikeParams) {
			size(content.SizeMedium, content.SizeHuge)(p)
			p.Hand = combat.HandAvailabilityNoHandFree
		}), combat.AttackNotPossibleReasonTargetTooLarge, false},
		{"Grapple: distance 6 and two sizes larger reports reach", `FR-105`, "unhappy (rule interaction)", with(g, func(p *combat.ResolveUnarmedStrikeParams) {
			size(content.SizeMedium, content.SizeHuge)(p)
			p.Distance = 6
		}), combat.AttackNotPossibleReasonBeyondReach, false},
		{"Grapple: Total Cover and too large reports cover", `FR-105`, "unhappy (rule interaction)", with(g, func(p *combat.ResolveUnarmedStrikeParams) {
			size(content.SizeMedium, content.SizeHuge)(p)
			p.Cover = total
		}), combat.AttackNotPossibleReasonTotalCover, false},
		{"Shove: two sizes larger", `FR-075`, "unhappy (boundary)", with(s, size(content.SizeMedium, content.SizeHuge)), combat.AttackNotPossibleReasonTargetTooLarge, false},
		{"Shove: one size larger", `FR-075`, "unhappy (boundary)", with(s, size(content.SizeMedium, content.SizeLarge)), 0, true},
		{"Shove needs no free hand", `FR-075`, "unhappy (rule interaction)", with(s, func(p *combat.ResolveUnarmedStrikeParams) { p.Hand = combat.HandAvailabilityNoHandFree }), 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, reqs := runUnarmed(t, unarmedCommand(t, c.p), 10, 10)
			n, refused := res.(combat.UnarmedStrikeNotPossible)
			if refused == c.possible || (refused && (n.Reason() != c.want || len(reqs) != 0)) || (!refused && len(reqs) == 0) {
				t.Errorf("= %#v requests %v, want reason %v (possible %v) [%s]", res, reqs, c.want, c.possible, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-079, FR-080, FR-082 - the same command and dice give the same result and the command
// is not altered; a returned damage slice is a copy; extreme rolls give the table result.
// Path: unhappy (determinism, immutability, boundary).
func TestResolveUnarmedStrikeDeterminismAndExtremes(t *testing.T) {
	t.Parallel()
	cmd := unarmedCommand(t, unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: 15}))
	before := unarmedCommand(t, unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: 15}))
	a, _ := runUnarmed(t, cmd, 12)
	b, _ := runUnarmed(t, cmd, 12)
	r := a.(combat.UnarmedStrikeDamageResolved)
	r.Damage()[0] = combat.MustDamage(50, content.DamageFire)
	if !reflect.DeepEqual(b, a) || !reflect.DeepEqual(cmd, before) || a.(combat.UnarmedStrikeDamageResolved).Damage()[0].Amount() != 4 {
		t.Errorf("results differ, the command changed or a returned slice reached the result")
	}
	for _, c := range []struct {
		roller *dicetest.FixedRoller
		hit    bool
		crit   bool
	}{{dicetest.NewMinimumRoller(), false, false}, {dicetest.NewMaximumRoller(), true, true}} {
		uc, _ := combat.NewResolveUnarmedStrike(c.roller)
		res, err := uc.Run(context.Background(), cmd)
		if r, ok := res.(combat.UnarmedStrikeDamageResolved); err != nil || !ok || r.Hit() != c.hit || r.CriticalHit() != c.crit {
			t.Errorf("extreme roller: %#v, %v, want hit %v crit %v", res, err, c.hit, c.crit)
		}
	}
}

// Requirement: doc/specs/attacks.md FR-081, FR-085, FR-110 - a roller failing in the save d20 gives one resolve_unarmed_strike
// error and no resolve_attack operation and no d20 signal; a panic and a wrong answer likewise; nothing is logged.
// Path: per case.
func TestResolveUnarmedStrikePortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	rollers := map[string]combat.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) }),
		"wrong expression": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustParseExpression("1d6"), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 3)})
		}),
	}
	options := map[string]combat.UnarmedStrikeOption{
		"damage":  combat.UnarmedStrikeDamageOption{ArmorClass: 15},
		"grapple": combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: rollOf(0)},
		"shove":   combat.UnarmedStrikeShoveOption{Effect: combat.GiveProne, SaveAbility: ability.Strength, Attempt: rollOf(0)},
	}
	for rname, roller := range rollers {
		for oname, o := range options {
			t.Run(rname+" "+oname, func(t *testing.T) {
				t.Parallel()
				rec, opts := recording()
				uc, _ := combat.NewResolveUnarmedStrike(roller, opts...)
				res, err := uc.Run(context.Background(), unarmedCommand(t, unarmedParams(o)))
				var pe *srderr.PortError
				if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port {
					t.Fatalf("= %v, %v, want nil and a dice_roller *PortError", res, err)
				}
				errs := measured(rec, "srd5e.combat.errors")
				if len(errs) != 1 || errs[0]["srd5e.combat.operation"] != "resolve_unarmed_strike" || errs[0]["error.type"] != srderr.ErrorType(err) {
					t.Errorf("errors = %v, want exactly one for resolve_unarmed_strike [FR-085]", errs)
				}
				if m := rec.Meter.MeasurementsNamed("srd5e.combat.operations"); len(m) != 0 {
					t.Errorf("operations = %v on a failure", m)
				}
				for _, s := range rec.Tracer.Spans() {
					if s.Name != "srd5e.combat.resolve_unarmed_strike" || s.Status != telemetry.StatusError || len(s.Errors) != 1 {
						t.Errorf("span = %+v, want only one errored resolve_unarmed_strike span", s)
					}
				}
				if len(rec.Logger.Records()) != 0 {
					t.Errorf("logged on a failure [FR-131]")
				}
				checkNoD20Signals(t, rec)
			})
		}
	}
}

// Requirement: doc/specs/attacks.md FR-115, testing.md §4 - a nil roller and the zero use case or command are validation
// errors; a cancelled ctx comes back as ctx.Err() with no signal and no die.
// Path: unhappy (invalid input, cancellation).
func TestResolveUnarmedStrikeInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewResolveUnarmedStrike(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewResolveUnarmedStrike(nil) error = %v, want ErrInvalidInput", err)
	}
	cmd := unarmedCommand(t, unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: 15}))
	if _, err := (combat.ResolveUnarmedStrike{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case error = %v", err)
	}
	uc, _ := combat.NewResolveUnarmedStrike(dicetest.NewFixedRoller(10))
	if _, err := uc.Run(context.Background(), combat.ResolveUnarmedStrikeCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	roller := dicetest.NewFixedRoller(10)
	rec, opts := recording()
	uc, _ = combat.NewResolveUnarmedStrike(roller, opts...)
	if res, err := uc.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans())+len(rec.Meter.Measurements())+len(roller.Requests()) != 0 {
		t.Errorf("cancelled ctx = %v, %v, want nil, context.Canceled and no signal or die", res, err)
	}
}

// Requirement: doc/specs/attacks.md FR-083, FR-084, FR-086, FR-110, FR-116, FR-118, FR-122, FR-131 - the signals of
// ResolveUnarmedStrike: the shared span and operations counter with the option, cover, automatic_critical and reason, a
// result and class for every result value, roll_mode on a rolled save, critical_hit only for a rolled attack, no modifiers
// histogram, one debug log, no d20 signal and an identical result with and without telemetry.
// Path: per case.
func TestResolveUnarmedStrikeTelemetry(t *testing.T) {
	t.Parallel()
	total := []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	g := func(at d20.SaveAttempt) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: at}
	}
	s := func(at d20.SaveAttempt) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeShoveOption{Effect: combat.PushFiveFeet, SaveAbility: ability.Strength, Attempt: at}
	}
	cases := []struct {
		name, requirement, path string
		option                  combat.UnarmedStrikeOption
		edit                    func(*combat.ResolveUnarmedStrikeParams)
		faces                   []int
		result, class, outcome  string
		opt, cover, reason      string
		mode                    string
		critical                any
		auto                    bool
	}{
		{"damage hit", "FR-122", "happy", combat.UnarmedStrikeDamageOption{ArmorClass: 15}, nil, []int{12}, "damage_hit", "applied", "success", "damage", "none", "", "normal", false, false},
		{"damage critical hit", "FR-122, FR-118", "happy", combat.UnarmedStrikeDamageOption{ArmorClass: 15}, nil, []int{20}, "damage_critical_hit", "applied", "success", "damage", "none", "", "normal", true, false},
		{"damage miss", "FR-122, FR-086", "unhappy (boundary)", combat.UnarmedStrikeDamageOption{ArmorClass: 15}, func(p *combat.ResolveUnarmedStrikeParams) { p.AutomaticCritical = combat.AutomaticCriticalAutomatic }, []int{2}, "damage_miss", "applied", "success", "damage", "none", "", "normal", false, true},
		{"grapple resisted", "FR-122, FR-118", "unhappy (boundary)", g(rollOf(10)), nil, []int{10}, "grapple_resisted", "applied", "success", "grapple", "none", "", "normal", nil, false},
		{"grapple applied", "FR-122", "happy", g(rollOf(0, disadvantage)), nil, []int{19, 3}, "grapple_applied", "applied", "success", "grapple", "none", "", "disadvantage", nil, false},
		{"grapple forced: no roll mode", "FR-118", "unhappy (rule interaction)", g(d20.ChooseToFail{}), nil, nil, "grapple_applied", "applied", "success", "grapple", "none", "", "", nil, false},
		{"shove resisted", "FR-122", "unhappy (boundary)", s(rollOf(10)), nil, []int{10}, "shove_resisted", "applied", "success", "shove", "none", "", "normal", nil, false},
		{"shove applied", "FR-122", "happy", s(d20.AutomaticFailure{}), nil, nil, "shove_applied", "applied", "success", "shove", "none", "", "", nil, false},
		{"not possible", "FR-086, FR-116, FR-117", "unhappy (state error)", g(rollOf(0)), func(p *combat.ResolveUnarmedStrikeParams) { p.Cover = total }, nil, "not_possible", "refused", "failure", "grapple", "total", "total_cover", "", nil, false},
		{"not possible: no free hand", "FR-117", "unhappy (state error)", g(rollOf(0)), func(p *combat.ResolveUnarmedStrikeParams) { p.Hand = combat.HandAvailabilityNoHandFree }, nil, "not_possible", "refused", "failure", "grapple", "none", "no_free_hand", "", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := unarmedParams(c.option)
			if c.edit != nil {
				c.edit(&p)
			}
			cmd := unarmedCommand(t, p)
			rec, opts := recording()
			uc, _ := combat.NewResolveUnarmedStrike(dicetest.NewFixedRoller(c.faces...), opts...)
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if plain, _ := runUnarmed(t, cmd, c.faces...); !reflect.DeepEqual(res, plain) {
				t.Errorf("a recording bundle changed the result [FR-083]")
			}
			ops := measured(rec, "srd5e.combat.operations")
			want := map[string]any{"srd5e.combat.operation": "resolve_unarmed_strike", "srd5e.outcome": c.outcome, "srd5e.combat.result": c.result,
				"srd5e.combat.result_class": c.class, "srd5e.combat.unarmed_strike.option": c.opt, "srd5e.combat.attack.cover": c.cover,
				"srd5e.combat.attack.automatic_critical": c.auto}
			if c.reason != "" {
				want["srd5e.combat.attack.reason"] = c.reason
			}
			if c.mode != "" {
				want["srd5e.d20_test.roll_mode"] = c.mode
			}
			if c.critical != nil {
				want["srd5e.d20_test.critical_hit"] = c.critical
			}
			if len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if _, resolveAttack := rec.Tracer.SpansNamed("srd5e.combat.resolve_attack"), measured(rec, "srd5e.combat.attack.modifiers"); len(resolveAttack) != 0 {
				t.Errorf("a resolve_attack signal or a modifiers sample was recorded [FR-110, FR-130]")
			}
			if spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_unarmed_strike"); len(spans) != 1 || spans[0].Status != telemetry.StatusUnset {
				t.Errorf("spans = %+v, want one without error status [FR-086]", spans)
			}
			logs := rec.Logger.Records()
			if len(logs) != 1 || logs[0].Level != telemetry.LevelDebug || logs[0].Message != "unarmed strike resolved" ||
				!reflect.DeepEqual(attrMap(logs[0].Attrs), map[string]any{"srd5e.combat.result": c.result}) {
				t.Errorf("logs = %+v, want one debug 'unarmed strike resolved' [FR-131]", logs)
			}
			checkNoD20Signals(t, rec)
		})
	}
}
