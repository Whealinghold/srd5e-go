package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func useIntervention(t *testing.T, roller character.Roller, opts ...character.Option) character.UseDivineIntervention {
	t.Helper()
	return must(character.NewUseDivineIntervention(roller, opts...))
}

func interventionCmd(c character.Character, spell content.Spell) character.UseDivineInterventionCommand {
	return character.UseDivineInterventionCommand{Character: c, Spell: spell, Rules: character.DefaultClericRuleData()}
}

func wishSpell(t *testing.T) content.Spell {
	t.Helper()
	return clcSpell(t, "wish", 9, []string{"wizard"}, "")
}

// Requirement: doc/specs/class-cleric.md FR-086, FR-091, FR-092, FR-119, FR-125, table 4.11.1 DI-01, DI-11 (SRD 5.2.1, "Level 10: Divine Intervention", p. 38) - a use hands the host a free cast (the spell and its casting time, no slot, no Material
// components), rolls no dice, leaves Divine Intervention unavailable until a Long Rest and produces one DivineInterventionUsed event; a Short Rest does not make it available, a Long Rest does; a second use is
// rejected with the state.
// Path: per case: happy / unhappy (state error).
func TestUseDivineInterventionApplied(t *testing.T) {
	t.Parallel()
	spell := clcClericSpell(t, "cure_wounds", 1)
	roller := dicetest.NewFixedRoller()
	c := clcCleric(t, 10)
	r, ok := must(useIntervention(t, roller).Run(context.Background(), interventionCmd(c, spell))).(character.UseDivineInterventionApplied)
	if !ok {
		t.Fatal("not applied")
	}
	cast := r.Cast()
	if cast.Spell != "cure_wounds" || cast.CastingTime != (content.CastingTimeMagicAction{}) || cast.SpellSlotNeeded() || cast.MaterialComponentsNeeded() {
		t.Errorf("cast = %+v", cast)
	}
	if _, rolled := r.RechargeRoll(); rolled || len(roller.Requests()) != 0 {
		t.Error("a Cleric-list spell rolled the recharge dice")
	}
	if r.Resources().DivineIntervention() != (character.DivineInterventionUnavailableUntilLongRest{}) || clcResourcesOf(t, r.Character()) != r.Resources() {
		t.Errorf("resources = %+v", r.Resources())
	}
	ev, isEvent := r.Events()[0].(character.DivineInterventionUsed)
	if len(r.Events()) != 1 || !isEvent || ev.Spell() != "cure_wounds" || ev.LongRestsToGo() != 0 || ev.CharacterID() != c.ID() {
		t.Errorf("events = %#v", r.Events())
	}
	if _, held, _ := character.ReadFeatureState(c.FeatureStates(), character.ClericResourcesKey()); held {
		t.Error("Run altered the input")
	}
	// DI-12: a second use before a Long Rest.
	again, err := useIntervention(t, roller).Run(context.Background(), interventionCmd(r.Character(), spell))
	if err != nil || again != character.UseDivineInterventionResult(character.UseDivineInterventionRejected{Reason: character.ClericNotAvailable, State: character.DivineInterventionUnavailableUntilLongRest{}}) {
		t.Errorf("second use = %#v, %v", again, err)
	}
	short := clcWith(t, r.Character(), r.Resources().AfterRest(character.ShortRest))
	if res, _ := useIntervention(t, roller).Run(context.Background(), interventionCmd(short, spell)); !isRejected(res, character.ClericNotAvailable) {
		t.Errorf("after a Short Rest = %#v", res)
	}
	long := clcWith(t, r.Character(), r.Resources().AfterRest(character.LongRest))
	if res, _ := useIntervention(t, roller).Run(context.Background(), interventionCmd(long, spell)); !isApplied(res) {
		t.Errorf("after a Long Rest = %#v", res)
	}
	r.Events()[0] = nil
	if r.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
}

func isRejected(res character.UseDivineInterventionResult, reason character.ClericReason) bool {
	r, ok := res.(character.UseDivineInterventionRejected)
	return ok && r.Reason == reason
}

func isApplied(res character.UseDivineInterventionResult) bool {
	_, ok := res.(character.UseDivineInterventionApplied)
	return ok
}

// Requirement: doc/specs/class-cleric.md FR-094, FR-095, FR-119, FR-125, tables 4.11.1 DI-10 and 4.11.2 GD-01 to GD-07 (SRD 5.2.1, "Level 20: Greater Divine Intervention", p. 38) - Wish at level 20 rolls the grant's 2d4 once, starts the
// recharge at their total (2 to 8 Long Rests, counted down by Long Rests only) and reports the roll; at level 19 it is not a Cleric spell; a use attempt during the recharge is not_available.
// Path: per case (GD rows): happy / unhappy (boundary, state error).
func TestUseDivineInterventionWish(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		rolls []int
		total int
	}{{"GD-01 minimum", []int{1, 1}, 2}, {"GD-04 maximum", []int{4, 4}, 8}, {"GD-06 two and three", []int{2, 3}, 5}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.rolls...)
			r, ok := must(useIntervention(t, roller).Run(context.Background(), interventionCmd(clcCleric(t, 20), wishSpell(t)))).(character.UseDivineInterventionApplied)
			if !ok {
				t.Fatal("not applied")
			}
			rolled, hasRoll := r.RechargeRoll()
			if !hasRoll || !reflect.DeepEqual(rolled, c.rolls) || r.Resources().DivineIntervention() != (character.DivineInterventionRecharging{LongRestsRemaining: c.total}) {
				t.Errorf("roll %v, state %+v", rolled, r.Resources().DivineIntervention())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(2, dice.D4(), 0) {
				t.Errorf("requests = %v, want one 2d4", reqs)
			}
			if ev := r.Events()[0].(character.DivineInterventionUsed); ev.LongRestsToGo() != c.total || ev.Spell() != "wish" {
				t.Errorf("event = %+v", ev)
			}
			rolled[0] = 99
			if again, _ := r.RechargeRoll(); again[0] == 99 {
				t.Error("RechargeRoll returned a shared slice")
			}
			// GD-06 / GD-07: Short Rests do not count and the use attempt is rejected while recharging.
			res := r.Resources().AfterRest(character.ShortRest).AfterRest(character.ShortRest)
			attempt, _ := useIntervention(t, dicetest.NewFixedRoller()).Run(context.Background(), interventionCmd(clcWith(t, r.Character(), res), clcClericSpell(t, "cure_wounds", 1)))
			if rej, ok := attempt.(character.UseDivineInterventionRejected); !ok || rej.Reason != character.ClericNotAvailable || rej.State != (character.DivineInterventionRecharging{LongRestsRemaining: c.total}) {
				t.Errorf("attempt = %#v", attempt)
			}
			after := r.Resources()
			for i := 0; i < c.total; i++ {
				if after.DivineIntervention() == (character.DivineInterventionAvailable{}) {
					t.Fatalf("available after %d of %d Long Rests", i, c.total)
				}
				after = after.AfterRest(character.LongRest)
			}
			if after.DivineIntervention() != (character.DivineInterventionAvailable{}) {
				t.Errorf("not available after %d Long Rests: %+v", c.total, after.DivineIntervention())
			}
		})
	}
	res, err := useIntervention(t, dicetest.NewFixedRoller(1, 1)).Run(context.Background(), interventionCmd(clcCleric(t, 19), wishSpell(t)))
	if err != nil || !isRejected(res, character.ClericNotClericSpell) {
		t.Errorf("DI-09 = %#v, %v", res, err)
	}
	clericWish := interventionCmd(clcCleric(t, 20), clcClericSpell(t, "cure_wounds", 1))
	if res, _ := useIntervention(t, dicetest.NewFixedRoller()).Run(context.Background(), clericWish); !isApplied(res) {
		t.Errorf("DI-11 = %#v", res)
	}
}

// Requirement: doc/specs/class-cleric.md FR-087 to FR-090, FR-093, FR-128, FR-122, table 4.11.1 DI-02 to DI-08, DI-13, DI-14 - the use case reports the first reason of the domain check without rolling; a character with no Cleric
// levels is UseDivineInterventionNotACleric.
// Path: per case (DI rows): unhappy (boundary, rule interaction, state error).
func TestUseDivineInterventionRejections(t *testing.T) {
	t.Parallel()
	cleric := []string{"cleric"}
	used := clcWith(t, clcCleric(t, 10), must(character.NewClericResources(3, 3, character.DivineInterventionUnavailableUntilLongRest{})))
	for _, c := range []struct {
		label  string
		char   character.Character
		spell  content.Spell
		reason character.ClericReason
	}{
		{"DI-02 cantrip", clcCleric(t, 10), clcSpell(t, "sacred_flame", 0, cleric, ""), 0},
		{"DI-03 level 5", clcCleric(t, 10), clcSpell(t, "flame_strike", 5, cleric, ""), 0},
		{"DI-04 level 6", clcCleric(t, 10), clcSpell(t, "heal", 6, cleric, ""), character.ClericSpellLevelTooHigh},
		{"DI-05 Reaction", clcCleric(t, 10), clcSpell(t, "shield_of_x", 5, cleric, "["+timeReaction+"]"), character.ClericReactionCastingTime},
		{"DI-06 Bonus Action", clcCleric(t, 10), clcSpell(t, "healing_word", 1, cleric, "["+timeBonusAction+"]"), 0},
		{"DI-07 off the list", clcCleric(t, 10), clcSpell(t, "fireball", 3, []string{"wizard"}, ""), character.ClericNotClericSpell},
		{"DI-08 level 9", clcCleric(t, 9), clcSpell(t, "cure_wounds", 1, cleric, ""), character.ClericBelowLevel10},
		{"DI-13 used and level 6", used, clcSpell(t, "heal", 6, cleric, ""), character.ClericNotAvailable},
		{"DI-14 level 9, off the list, level 6, Reaction", clcCleric(t, 9), clcSpell(t, "x", 6, []string{"wizard"}, "["+timeReaction+"]"), character.ClericBelowLevel10},
	} {
		roller := dicetest.NewFixedRoller()
		res, err := useIntervention(t, roller).Run(context.Background(), interventionCmd(c.char, c.spell))
		if err != nil || (c.reason == 0) != isApplied(res) || (c.reason != 0 && !isRejected(res, c.reason)) || len(roller.Requests()) != 0 {
			t.Errorf("%s: = %#v, %v; want reason %v", c.label, res, err, c.reason)
		}
	}
	res, err := useIntervention(t, dicetest.NewFixedRoller()).Run(context.Background(), interventionCmd(fgtFighter(t, 10), clcClericSpell(t, "cure_wounds", 1)))
	if _, ok := res.(character.UseDivineInterventionNotACleric); err != nil || !ok {
		t.Errorf("Fighter = %#v, %v", res, err)
	}
}

// Requirement: doc/specs/class-cleric.md FR-130, table 4.14.4 CF-03 - a roller that returns its error, panics or is exhausted while Wish rolls the recharge dice is a *srderr.PortError for dice.Port with the cause
// reachable; the result is nil and the Character is unchanged (Divine Intervention is not recharging or used).
// Path: per case (unhappy: collaborator failure).
func TestUseDivineInterventionRollerFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	for _, c := range []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"CF-03 an error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		{"a panic", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), boom},
		{"exhausted", dicetest.NewFixedRoller(), dicetest.ErrExhausted},
	} {
		res, err := useIntervention(t, c.roller).Run(context.Background(), interventionCmd(clcCleric(t, 20), wishSpell(t)))
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
			t.Errorf("%s: = %v, %v; want a PortError for dice with the cause", c.name, res, err)
		}
	}
	// A grant whose dice the roller answers outside the faces cannot be built: the dice port itself refuses a face of 5 on a d4.
	res, err := useIntervention(t, dicetest.NewFixedRoller(5, 1)).Run(context.Background(), interventionCmd(clcCleric(t, 20), wishSpell(t)))
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) {
		t.Errorf("a face of 5 = %v, %v", res, err)
	}
}

// Requirement: doc/specs/class-cleric.md FR-123, FR-126 - the result is identical with the no-op and with a recording telemetry; the constructor refuses a nil roller, and Run refuses the zero use case, a zero Character, the zero
// spell, a casting time the spell does not list, a foreign type of state and a cancelled context, the last before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestUseDivineInterventionInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	good := interventionCmd(clcCleric(t, 20), clcClericSpell(t, "cure_wounds", 1))
	rec, opts := fgtTelemetry()
	withRec, err := useIntervention(t, dicetest.NewFixedRoller(), opts...).Run(context.Background(), good)
	plain, err2 := useIntervention(t, dicetest.NewFixedRoller()).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) || len(rec.Tracer.Spans()) != 1 {
		t.Errorf("results differ: %#v %v vs %#v %v (spans %d)", withRec, err, plain, err2, len(rec.Tracer.Spans()))
	}
	_, err = character.NewUseDivineIntervention(nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.UseDivineIntervention{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	use := useIntervention(t, dicetest.NewFixedRoller())
	for name, cmd := range map[string]character.UseDivineInterventionCommand{
		"zero character": {Spell: good.Spell, Rules: good.Rules}, "zero spell": {Character: good.Character, Rules: good.Rules},
	} {
		_, err = use.Run(context.Background(), cmd)
		wantInvalid(t, err, name)
	}
	bad := good
	bad.CastingTime = content.CastingTimeMinutes{}
	_, err = use.Run(context.Background(), bad)
	wantInvalid(t, err, "unlisted casting time")
	foreign := must(character.WithFeatureState(clcCleric(t, 20), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_resources"), must(character.NewChosenSubclass("life_domain"))))
	_, err = use.Run(context.Background(), interventionCmd(foreign, good.Spell))
	wantInvalid(t, err, "foreign state")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	rec2, opts2 := fgtTelemetry()
	res, err := useIntervention(t, dicetest.NewFixedRoller(), opts2...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}
