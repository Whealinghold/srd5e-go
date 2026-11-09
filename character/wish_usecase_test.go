package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func wishResolver(t *testing.T, roller character.Roller, opts ...character.Option) character.WishResolver {
	t.Helper()
	return must(character.NewWishResolver(roller, opts...))
}

var (
	expr2d4   = dice.MustExpression(2, dice.D4(), 0)
	expr1d100 = dice.MustExpression(1, dice.D100(), 0)
)

func reshape(t *testing.T) character.WishChoice {
	t.Helper()
	return must(character.NewReshapeReality(statement(t, "A boon")))
}

// Requirement: per case.
// Path: per case.
func TestResolveWishAppliesTheStress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, requirement, path string
		rolls                   []int
		want                    character.WishStress
		lockedOut               bool
	}{
		{"S1 2d4 = 2, 34", "FR-022, FR-023, FR-024, FR-026", "unhappy (boundary)", []int{1, 1, 34}, wishStress(t, 2, true, false), false},
		{"S2 2d4 = 8, 33", "FR-022, FR-025", "unhappy (boundary)", []int{4, 4, 33}, wishStress(t, 8, true, true), true},
		{"S3 percentile 1", "FR-025", "unhappy (boundary)", []int{2, 3, 1}, wishStress(t, 5, true, true), true},
		{"S4 percentile 100", "FR-022", "unhappy (boundary)", []int{2, 3, 100}, wishStress(t, 5, true, false), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(tc.rolls...)
			c := wishWizard(t)
			got, err := wishResolver(t, roller).ResolveWish(context.Background(), character.ResolveWishCommand{Character: c, Choice: reshape(t)})
			applied, ok := got.(character.ResolveWishApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", got, err)
			}
			if reqs := roller.Requests(); len(reqs) != 2 || reqs[0] != expr2d4 || reqs[1] != expr1d100 {
				t.Errorf("requests = %v, want 2d4 then 1d100 [FR-022]", reqs)
			}
			ev := applied.Events()
			if s := stressOfChar(t, applied.Character()); s != tc.want || len(ev) != 1 || ev[0].(character.WishStressEndured).LockedOut() != tc.lockedOut {
				t.Errorf("stress %+v, events %#v, want %+v [%s %s]", s, ev, tc.want, wishSpec, tc.requirement)
			}
			if applied.Feats().Len() != 0 || applied.Effect().(character.WishReshapeReality).Statement().String() != "A boon" || !stressedFree(c) {
				t.Error("feats, effect or the input are wrong")
			}
		})
	}
}

// stressedFree reports that c still holds no stress state: ResolveWish does not alter its input.
func stressedFree(c character.Character) bool { return c.FeatureStates().Len() == 0 }

// Requirement: doc/specs/wish-spell.md FR-004, FR-020, S9, E1 - a duplication rolls no dice, adds no stress, leaves the Character equal and
// returns the handoff with the one event WishDuplicated.
// Path: unhappy (rule interaction).
func TestResolveWishDuplicationRollsNothing(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller()
	c := wishWizard(t)
	sp := levelSpell(t, 8)
	got, err := wishResolver(t, roller).ResolveWish(context.Background(), character.ResolveWishCommand{Character: c, Choice: must(character.NewDuplicateSpell(sp))})
	dup, ok := got.(character.ResolveWishDuplicated)
	if err != nil || !ok || !dup.Character().Equal(c) || len(roller.Requests()) != 0 {
		t.Fatalf("= %#v, %v, requests %v", got, err, roller.Requests())
	}
	ev := dup.Events()
	if eff := dup.Effect().(character.WishDuplicate); eff.Spell() != sp.ID() || len(ev) != 1 || ev[0].(character.WishDuplicated).Level() != 8 {
		t.Errorf("effect %+v, events %#v", dup.Effect(), ev)
	}
}

// Requirement: doc/specs/wish-spell.md FR-021, FR-031, E2, E7, E22, E32, E33 - a rejected choice is a result, never an error: the roller is never called,
// the rejection names its kind (the nested FeatRejection for a refused replacement), and a locked-out creature is refused first.
// Path: per case.
func TestResolveWishRejections(t *testing.T) {
	t.Parallel()
	feats := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	for _, tc := range []struct {
		name, requirement, path string
		c                       character.Character
		choice                  character.WishChoice
		want                    character.ResolveWishRejectionKind
	}{
		{"E2 Wish itself", "FR-002", "unhappy (boundary)", wishWizard(t), must(character.NewDuplicateSpell(levelSpell(t, 9))), character.WishSpellLevelTooHigh},
		{"E7 object worth 25,001 GP", "FR-005", "unhappy (boundary)", wishWizard(t), must(character.NewObjectCreation(gpCost(25001), 5, false)), character.WishObjectValueTooHigh},
		{"E22 feat required by another", "FR-012", "unhappy (state error)", wishWizard(t), suddenLearning(t, feats, "alert", savageFeat(), []content.FeatID{"alert"}, character.HeldFeatures{}), character.WishFeatRequiredByAnother},
		{"E20 feat not held", "FR-011", "unhappy (state error)", wishWizard(t), suddenLearning(t, feats, "savage_attacker", alertFeat(), nil, character.HeldFeatures{}), character.WishFeatReplacementRejected},
		{"E33 locked out, duplication", "FR-031", "unhappy (state error)", stressed(t, wishWizard(t), 0, false, true), must(character.NewDuplicateSpell(levelSpell(t, 1))), character.WishCannotCastAgain},
		{"E33 locked out, roll redo", "FR-031", "unhappy (state error)", stressed(t, wishWizard(t), 3, true, true), must(character.NewRollRedo(modifier.Normal, true)), character.WishCannotCastAgain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(1, 1, 50)
			got, err := wishResolver(t, roller).ResolveWish(context.Background(), character.ResolveWishCommand{Character: tc.c, Choice: tc.choice})
			rej, ok := got.(character.ResolveWishRejected)
			if err != nil || !ok || rej.Rejection().Kind() != tc.want || len(roller.Requests()) != 0 {
				t.Errorf("= %#v, %v, requests %v, want %v [%s %s]", got, err, roller.Requests(), tc.want, wishSpec, tc.requirement)
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-064, S10 - a failing roller (on the second roll too), a panicking roller and one answering another
// expression are a *srderr.PortError for the dice port with the cause reachable, a nil result and no applied stress; the first roll is not applied.
// Path: per case.
func TestResolveWishPortFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	calls := 0
	secondFails := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		calls++
		if calls == 2 {
			return dice.Result{}, boom
		}
		return dicetest.NewMinimumRoller().Roll(e)
	})
	for _, tc := range []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"first roll fails", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		{"second roll fails", secondFails, boom},
		{"roller panics", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("down") }), srderr.ErrPortPanic},
		{"another expression", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
		}), srderr.ErrContractViolation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := wishWizard(t)
			got, err := wishResolver(t, tc.roller).ResolveWish(context.Background(), character.ResolveWishCommand{Character: c, Choice: reshape(t)})
			var pe *srderr.PortError
			if got != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, tc.cause) || !stressedFree(c) {
				t.Errorf("= %#v, %v", got, err)
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-063, FR-065 - equal commands with equal fixed rollers give equal results, and the command is not altered.
// Path: unhappy (determinism).
func TestResolveWishIsDeterministic(t *testing.T) {
	t.Parallel()
	cmd := character.ResolveWishCommand{Character: wishWizard(t), Choice: reshape(t)}
	a, errA := wishResolver(t, dicetest.NewFixedRoller(2, 3, 40)).ResolveWish(context.Background(), cmd)
	b, errB := wishResolver(t, dicetest.NewFixedRoller(2, 3, 40)).ResolveWish(context.Background(), cmd)
	c, _ := wishResolver(t, dicetest.NewFixedRoller(2, 3, 41)).ResolveWish(context.Background(), cmd)
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) || reflect.DeepEqual(a, c) || !stressedFree(cmd.Character) {
		t.Errorf("results differ or the command changed: %#v %#v", a, b)
	}
}

// Requirement: doc/specs/wish-spell.md FR-062, FR-068, errors-and-results.md FR-054 - the zero resolver, a nil roller, a zero Character, a nil choice, a zero
// variant and a stress state of another type are invalid-input errors; a cancelled context is ctx.Err(); none rolls or emits a signal.
// Path: per case.
func TestResolveWishInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller(1, 1, 50)
	uc := wishResolver(t, roller, character.WithTelemetry(rec.Telemetry()))
	c := wishWizard(t)
	foreign := must(character.WithFeatureState(c, character.MustFeatureStateKey[foreignState]("wish_stress"), foreignState{}))
	for _, tc := range []struct {
		name string
		uc   character.WishResolver
		cmd  character.ResolveWishCommand
	}{
		{"zero resolver", character.WishResolver{}, character.ResolveWishCommand{Character: c, Choice: reshape(t)}},
		{"zero character", uc, character.ResolveWishCommand{Choice: reshape(t)}},
		{"nil choice", uc, character.ResolveWishCommand{Character: c}},
		{"zero variant", uc, character.ResolveWishCommand{Character: c, Choice: character.ObjectCreation{}}},
		{"foreign state", uc, character.ResolveWishCommand{Character: foreign, Choice: reshape(t)}},
	} {
		if got, err := tc.uc.ResolveWish(context.Background(), tc.cmd); got != nil || !isInvalid(err) {
			t.Errorf("%s: = %#v, %v", tc.name, got, err)
		}
	}
	if _, err := character.NewWishResolver(nil); !isInvalid(err) {
		t.Errorf("nil roller: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.ResolveWish(ctx, character.ResolveWishCommand{Character: c, Choice: reshape(t)}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()) + len(roller.Requests()); n != 0 {
		t.Errorf("%d signals or rolls before validation", n)
	}
}

// Requirement: per case.
// Path: per case.
func TestRollWishStressDamage(t *testing.T) {
	t.Parallel()
	active := stressed(t, wishWizard(t), 4, true, false)
	inactive := stressed(t, wishWizard(t), 4, false, false)
	slot3, sp3 := slotCast(t, 3, 3)
	slot1at3, sp1 := slotCast(t, 1, 3)
	slot9, sp9 := slotCast(t, 9, 9)
	ritual, spRitual := wishCastOf(t, castCase{spell: castSpell(t, spellID(2, 9), 2, asRitual()), mode: character.CastModeRitual})
	feature, spFeature := wishCastOf(t, castCase{spell: castSpell(t, spellID(5, 1), 5), mode: character.CastModeFeatureGranted, featureLevel: 5})
	cantrip, spCantrip := wishCastOf(t, castCase{spell: castSpell(t, spellID(0, 1), 0)})
	slot4, sp4 := slotCast(t, 4, 4)
	for _, tc := range []struct {
		name, requirement, path string
		c                       character.Character
		cast                    character.SpellCast
		spell                   content.Spell
		origin                  character.WishCastOrigin
		dice                    int // 0: no roll expected
		rolls                   []int
		amount                  int
		none                    character.WishDamageNoneReason
	}{
		{"D1 level 3 slot spell, 3d10 = 17", "FR-041", "happy", active, slot3, sp3, character.WishOrdinaryCast, 3, []int{5, 6, 6}, 17, 0},
		{"D2 level 1 spell with a level 3 slot", "FR-040", "unhappy (rule interaction)", active, slot1at3, sp1, character.WishOrdinaryCast, 3, []int{1, 1, 1}, 3, 0},
		{"D3 level 9 slot", "FR-041", "unhappy (boundary)", active, slot9, sp9, character.WishOrdinaryCast, 9, []int{10, 10, 10, 10, 10, 10, 10, 10, 10}, 90, 0},
		{"D4 Ritual of a level 2 spell", "FR-040", "unhappy (rule interaction)", active, ritual, spRitual, character.WishOrdinaryCast, 2, []int{4, 9}, 13, 0},
		{"D5 feature-granted level 5 spell", "FR-040", "unhappy (rule interaction)", active, feature, spFeature, character.WishOrdinaryCast, 5, []int{1, 2, 3, 4, 5}, 15, 0},
		{"D6 cantrip", "FR-042", "unhappy (boundary)", active, cantrip, spCantrip, character.WishOrdinaryCast, 0, nil, 0, character.WishNoDamageCantrip},
		{"D7 inactive, level 4 spell", "FR-042", "unhappy (state error)", inactive, slot4, sp4, character.WishOrdinaryCast, 0, nil, 0, character.WishNoDamageNotActive},
		{"D7b no stress at all", "FR-042", "unhappy (empty and minimal)", wishWizard(t), slot4, sp4, character.WishOrdinaryCast, 0, nil, 0, character.WishNoDamageNotActive},
		{"D8 the duplicate of a Wish", "FR-043, FR-069", "unhappy (rule interaction)", active, slot4, sp4, character.WishDuplicateCast, 0, nil, 0, character.WishNoDamageDuplicatedSpell},
		{"D8b a duplicate while inactive reports not active", "FR-042, FR-043", "unhappy (rule interaction)", inactive, slot4, sp4, character.WishDuplicateCast, 0, nil, 0, character.WishNoDamageNotActive},
		{"D8c a duplicated cantrip reports the cantrip", "FR-042, FR-043", "unhappy (rule interaction)", active, cantrip, spCantrip, character.WishDuplicateCast, 0, nil, 0, character.WishNoDamageCantrip},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(tc.rolls...)
			before := tc.c
			got, err := wishResolver(t, roller).RollWishStressDamage(context.Background(), character.RollWishStressDamageCommand{Character: tc.c, SpellCast: tc.cast, Spell: tc.spell, Origin: tc.origin})
			if err != nil || !tc.c.Equal(before) {
				t.Fatalf("= %#v, %v", got, err)
			}
			if tc.dice == 0 {
				none, ok := got.(character.RollWishStressDamageNone)
				if !ok || none.Reason() != tc.none || len(roller.Requests()) != 0 {
					t.Errorf("= %#v, requests %v, want none/%v [%s %s]", got, roller.Requests(), tc.none, wishSpec, tc.requirement)
				}
				return
			}
			rolled, ok := got.(character.RollWishStressDamageRolled)
			if !ok {
				t.Fatalf("= %#v", got)
			}
			ev := rolled.Events()
			if d := rolled.Damage(); d.Type() != content.DamageNecrotic || d.Amount() != tc.amount {
				t.Errorf("damage = %+v, want %d necrotic [%s %s]", d, tc.amount, wishSpec, tc.requirement)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(tc.dice, dice.D10(), 0) {
				t.Errorf("requests = %v, want %dd10", reqs, tc.dice)
			}
			if e, ok := ev[0].(character.WishStressDamageRolled); len(ev) != 1 || !ok || int(e.Level()) != tc.dice || e.Amount() != tc.amount || e.CharacterID() != caster {
				t.Errorf("events = %#v", ev)
			}
			ev[0] = nil
			if rolled.Events()[0] == nil {
				t.Error("Events aliases the result")
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-062, FR-068, D21 to D23 - a cast of another character or of another spell, a cast level below the spell's or on a cantrip, an
// unknown origin, a zero Character, the zero resolver and a stress state of another type are invalid-input errors before any die or signal.
// Path: per case.
func TestRollWishStressDamageInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller(1, 1, 1, 1)
	uc := wishResolver(t, roller, character.WithTelemetry(rec.Telemetry()))
	active := stressed(t, wishWizard(t), 4, true, false)
	cast, sp := slotCast(t, 1, 2)
	other := stressed(t, charWithID(t, "other", 5), 4, true, false)
	foreign := must(character.WithFeatureState(wishWizard(t), character.MustFeatureStateKey[foreignState]("wish_stress"), foreignState{}))
	level3 := castSpell(t, spellID(1, 1), 3)    // the cast level 2 is below the spell's level 3
	asCantrip := castSpell(t, spellID(1, 1), 0) // a cast level on a cantrip
	for _, tc := range []struct {
		name, requirement string
		uc                character.WishResolver
		cmd               character.RollWishStressDamageCommand
	}{
		{"another character's cast", "FR-062", uc, character.RollWishStressDamageCommand{Character: other, SpellCast: cast, Spell: sp}},
		{"another spell", "FR-062", uc, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: castSpell(t, spellID(1, 2), 1)}},
		{"D22 cast level 2 on a level 3 spell", "FR-062, FR-068", uc, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: level3}},
		{"D23 cast level on a cantrip", "FR-062, FR-068", uc, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: asCantrip}},
		{"unknown origin", "FR-069", uc, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: sp, Origin: 9}},
		{"negative origin", "FR-069", uc, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: sp, Origin: -1}},
		{"zero character", "FR-062", uc, character.RollWishStressDamageCommand{SpellCast: cast, Spell: sp}},
		{"foreign state", "FR-060", uc, character.RollWishStressDamageCommand{Character: foreign, SpellCast: cast, Spell: sp}},
		{"zero resolver", "FR-066", character.WishResolver{}, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: sp}},
	} {
		if got, err := tc.uc.RollWishStressDamage(context.Background(), tc.cmd); got != nil || !isInvalid(err) {
			t.Errorf("%s [%s]: = %#v, %v", tc.name, tc.requirement, got, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.RollWishStressDamage(ctx, character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: sp}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()) + len(roller.Requests()); n != 0 {
		t.Errorf("%d signals or rolls before validation", n)
	}
}

// Requirement: doc/specs/wish-spell.md FR-064 - a failing, panicking or contract-breaking roller makes RollWishStressDamage return the *srderr.PortError for the dice
// port with a nil result.
// Path: per case.
func TestRollWishStressDamagePortFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	cast, sp := slotCast(t, 2, 2)
	active := stressed(t, wishWizard(t), 4, true, false)
	for _, tc := range []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"fails", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		{"panics", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("down") }), srderr.ErrPortPanic},
		{"another expression", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
		}), srderr.ErrContractViolation},
	} {
		got, err := wishResolver(t, tc.roller).RollWishStressDamage(context.Background(), character.RollWishStressDamageCommand{Character: active, SpellCast: cast, Spell: sp})
		var pe *srderr.PortError
		if got != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, tc.cause) {
			t.Errorf("%s: = %#v, %v", tc.name, got, err)
		}
	}
}

// Requirement: doc/specs/wish-spell.md FR-066, section 7, observability.md section 9 - each use case records one span, one operations increment and one
// duration (operation only); a rejection is a failure without error status; a port failure is recorded once with error.type and no result;
// nothing is logged; a recorder and the Noop give equal results.
// Path: per case.
func TestWishSignals(t *testing.T) {
	t.Parallel()
	type want struct {
		attrs  map[string]any
		status telemetry.Status
		errors int
	}
	ok := func(result, effect string, extra map[string]any) map[string]any {
		return attrs(map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: result, character.AttrWishEffect: effect}, extra)
	}
	c := wishWizard(t)
	cast, sp := slotCast(t, 2, 2)
	active := stressed(t, wishWizard(t), 4, true, false)
	resolve := func(choice character.WishChoice, cc character.Character) func(character.WishResolver) (any, error) {
		return func(r character.WishResolver) (any, error) {
			return r.ResolveWish(context.Background(), character.ResolveWishCommand{Character: cc, Choice: choice})
		}
	}
	damage := func(origin character.WishCastOrigin, cc character.Character) func(character.WishResolver) (any, error) {
		return func(r character.WishResolver) (any, error) {
			return r.RollWishStressDamage(context.Background(), character.RollWishStressDamageCommand{Character: cc, SpellCast: cast, Spell: sp, Origin: origin})
		}
	}
	for _, tc := range []struct {
		name, requirement string
		span, op          string
		rolls             []int
		call              func(character.WishResolver) (any, error)
		want              want
		fails             bool
	}{
		{"applied, not locked out", "FR-066", character.SpanResolveWish, "resolve_wish", []int{1, 1, 90}, resolve(reshape(t), c),
			want{attrs: ok("applied", "reshape_reality", map[string]any{character.AttrWishLockout: false})}, false},
		{"applied, locked out", "FR-066", character.SpanResolveWish, "resolve_wish", []int{1, 1, 5}, resolve(must(character.NewInstantHealth(2)), c),
			want{attrs: ok("applied", "instant_health", map[string]any{character.AttrWishLockout: true})}, false},
		{"duplicated", "FR-066", character.SpanResolveWish, "resolve_wish", []int{}, resolve(must(character.NewDuplicateSpell(levelSpell(t, 2))), c),
			want{attrs: ok("duplicated", "duplicate", nil)}, false},
		{"rejected", "FR-066", character.SpanResolveWish, "resolve_wish", []int{}, resolve(must(character.NewObjectCreation(gpCost(25001), 5, false)), c),
			want{attrs: attrs(map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrWishEffect: "object_creation", character.AttrRejection: "object_value_too_high"})}, false},
		{"rejected, locked out", "FR-031", character.SpanResolveWish, "resolve_wish", []int{}, resolve(must(character.NewRollRedo(modifier.Normal, true)), stressed(t, c, 0, false, true)),
			want{attrs: attrs(map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrWishEffect: "roll_redo", character.AttrRejection: "cannot_cast_wish_again"})}, false},
		{"port failure", "FR-064", character.SpanResolveWish, "resolve_wish", []int{}, resolve(reshape(t), c),
			want{attrs: map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error", character.AttrWishEffect: "reshape_reality"}, status: telemetry.StatusError, errors: 1}, true},
		{"damage", "FR-066", character.SpanRollWishStressDamage, "roll_wish_stress_damage", []int{3, 4}, damage(character.WishOrdinaryCast, active),
			want{attrs: map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "damage"}}, false},
		{"no damage", "FR-066", character.SpanRollWishStressDamage, "roll_wish_stress_damage", []int{}, damage(character.WishDuplicateCast, active),
			want{attrs: map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "none"}}, false},
		{"damage port failure", "FR-064", character.SpanRollWishStressDamage, "roll_wish_stress_damage", []int{}, damage(character.WishOrdinaryCast, active),
			want{attrs: map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}, status: telemetry.StatusError, errors: 1}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			got, err := tc.call(wishResolver(t, dicetest.NewFixedRoller(tc.rolls...), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)))
			if (err != nil) != tc.fails {
				t.Fatalf("err = %v", err)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Name != tc.span || spans[0].Status != tc.want.status || len(spans[0].Errors) != tc.want.errors || !spans[0].Ended {
				t.Fatalf("spans = %+v", spans)
			}
			spanAttrs := attrs(tc.want.attrs)
			delete(spanAttrs, telemetry.AttrErrorType) // error.type is on the counter only (character-aggregate.md FR-071)
			sameAttrs(t, "span", spans[0].Attrs, spanAttrs)
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs(tc.op, tc.want.attrs))
			d := onlyOne(t, rec, character.MetricDuration)
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: tc.op})
			if d.Value != 0.25 {
				t.Errorf("duration = %+v", d)
			}
			noLogsOrViolations(t, rec)
			if tc.fails {
				return
			}
			plain, _ := tc.call(wishResolver(t, dicetest.NewFixedRoller(tc.rolls...)))
			if !reflect.DeepEqual(got, plain) {
				t.Error("Noop and a recorder must give equal results [observability.md section 9]")
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md section 7, FR-013, FR-014 - every choice is reported with its own wish_effect value on the span, and the
// handoff of each effect carries the facts of the choice.
// Path: per case.
func TestResolveWishEffectAttributeAndHandoffs(t *testing.T) {
	t.Parallel()
	feats := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	creatures := wishCreatures(t, 2)
	other := character.ImmuneToOtherEffect{Effect: statement(t, "a curse")}
	cost := gpCost(5)
	for _, tc := range []struct {
		effect string
		choice character.WishChoice
		check  func(character.WishEffect) bool
	}{
		{"duplicate", must(character.NewDuplicateSpell(levelSpell(t, 1))), func(e character.WishEffect) bool { return e.(character.WishDuplicate).Level() == 1 }},
		{"object_creation", must(character.NewObjectCreation(cost, 12.5, false)), func(e character.WishEffect) bool {
			o := e.(character.WishObject)
			return o.Value() == cost && o.LargestDimensionFeet() == 12.5
		}},
		{"instant_health", must(character.NewInstantHealth(4)), func(e character.WishEffect) bool { return e.(character.WishInstantHealth).OtherCreatures() == 4 }},
		{"resistance", must(character.NewResistanceGrant(content.DamageCold, creatures...)), func(e character.WishEffect) bool {
			r := e.(character.WishResistance)
			return r.DamageType() == content.DamageCold && len(r.Creatures()) == 2 && r.Permanent()
		}},
		{"spell_immunity", must(character.NewSpellImmunityGrant(other, creatures...)), func(e character.WishEffect) bool {
			i := e.(character.WishSpellImmunity)
			return i.Subject() == character.SpellImmunitySubject(other) && len(i.Creatures()) == 2 && i.DurationHours() == 8
		}},
		{"sudden_learning", suddenLearning(t, feats, "alert", savageFeat(), nil, character.HeldFeatures{}), func(e character.WishEffect) bool {
			return e.(character.WishSuddenLearning).New() == "savage_attacker"
		}},
		{"roll_redo", must(character.NewRollRedo(modifier.WithDisadvantage, true)), func(e character.WishEffect) bool {
			return e.(character.WishRollRedo).Mode() == modifier.WithDisadvantage
		}},
		{"reshape_reality", reshape(t), func(e character.WishEffect) bool {
			return e.(character.WishReshapeReality).Statement().String() == "A boon"
		}},
	} {
		t.Run(tc.effect, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			got, err := wishResolver(t, dicetest.NewFixedRoller(1, 1, 90), character.WithTelemetry(rec.Telemetry())).ResolveWish(context.Background(), character.ResolveWishCommand{Character: wishWizard(t), Choice: tc.choice})
			if err != nil {
				t.Fatal(err)
			}
			var effect character.WishEffect
			switch r := got.(type) {
			case character.ResolveWishApplied:
				effect = r.Effect()
			case character.ResolveWishDuplicated:
				effect = r.Effect()
			default:
				t.Fatalf("= %#v", got)
			}
			if !tc.check(effect) {
				t.Errorf("effect = %#v", effect)
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || attrMap(spans[0].Attrs)[character.AttrWishEffect] != tc.effect {
				t.Errorf("spans = %+v, want wish_effect %s", spans, tc.effect)
			}
		})
	}
}
