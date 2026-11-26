package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// trkHygiene fails the test when a recorded signal carries a combatant id, an srd5e.d20_test signal fired,
// or anything was logged (initiative-and-actions.md FR-108, observability.md section 7).
func trkHygiene(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
	if got := rec.Logger.Records(); len(got) != 0 {
		t.Errorf("logged %v, want nothing", got)
	}
	ids := []string{string(trkA), string(trkB), string(trkC)}
	for _, m := range rec.Meter.Measurements() {
		for _, a := range m.Attrs {
			if s, ok := a.Value().(string); ok && slices.Contains(ids, s) {
				t.Errorf("%s attribute %s carries a combatant id (FR-108)", m.Name, a.Key())
			}
		}
	}
	for _, s := range rec.Tracer.Spans() {
		for _, a := range s.Attrs {
			if v, ok := a.Value().(string); ok && slices.Contains(ids, v) {
				t.Errorf("span %s attribute %s carries a combatant id (FR-108)", s.Name, a.Key())
			}
		}
	}
}

func trkRollCommand(t *testing.T, method combat.InitiativeMethod, ps ...combat.InitiativeParticipant) combat.RollInitiativeCommand {
	t.Helper()
	cmd, err := combat.NewRollInitiativeCommand(method, ps)
	if err != nil {
		t.Fatalf("NewRollInitiativeCommand: %v", err)
	}
	return cmd
}

func trkRoll(t *testing.T, roller combat.Roller, cmd combat.RollInitiativeCommand, opts ...combat.Option) []combat.InitiativeBreakdown {
	t.Helper()
	uc, err := combat.NewRollInitiative(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res.(combat.RollInitiativeRolled).Breakdowns()
}

// Requirement: per case (table 4.1a, FR-138)
// Path: per case.
func TestRollInitiativeRolledCounts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		in                      combat.InitiativeParticipantInput
		faces                   []int
		wantExpr                string
		wantNatural, wantCount  int
	}{
		{"typical", "FR-001", "happy", combat.InitiativeParticipantInput{Modifier: mod(2)}, []int{10}, "1d20", 10, 12},
		{"natural 20", "FR-001", "unhappy (boundary)", combat.InitiativeParticipantInput{}, []int{20}, "1d20", 20, 20},
		{"Advantage", "FR-002", "happy", combat.InitiativeParticipantInput{Modifier: mod(2), Surprise: combat.NotSurprised, Circumstances: []modifier.RollCircumstance{modifier.Advantage{}}}, []int{3, 18}, "2d20", 18, 20},
		{"Disadvantage by surprise", "FR-003", "happy", combat.InitiativeParticipantInput{Modifier: mod(2), Surprise: combat.Surprised}, []int{18, 3}, "2d20", 3, 5},
		{"roll-only modifier", "FR-111", "unhappy (rule interaction)", combat.InitiativeParticipantInput{Modifier: mod(-5), RollOnly: mod(-12)}, []int{1}, "1d20", 1, -16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := c.in
			in.IDs = []combat.CombatantID{trkA}
			roller := dicetest.NewFixedRoller(c.faces...)
			got := trkRoll(t, roller, trkRollCommand(t, combat.InitiativeMethodRolled, trkParticipant(t, in)))
			rolled, ok := got[0].(combat.RolledInitiative)
			if !ok || len(got) != 1 {
				t.Fatalf("breakdowns = %#v, want one RolledInitiative [FR-012]", got)
			}
			if rolled.Count() != combat.InitiativeCount(c.wantCount) || rolled.Roll().Natural() != c.wantNatural || rolled.Method() != combat.InitiativeMethodRolled {
				t.Errorf("breakdown = count %d natural %d method %v, want %d and %d [%s]", rolled.Count(), rolled.Roll().Natural(), rolled.Method(), c.wantCount, c.wantNatural, c.requirement)
			}
			if !slices.Equal(rolled.Roll().Dice(), c.faces) || rolled.InitiativeModifier() != c.in.Modifier || rolled.RollOnlyModifier() != c.in.RollOnly || rolled.Mode() != rolled.Roll().Mode() {
				t.Errorf("breakdown does not keep the roll and modifiers: %+v [FR-012]", rolled)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0].String() != c.wantExpr {
				t.Errorf("requests = %v, want one %s with modifier 0 [FR-138]", reqs, c.wantExpr)
			}
			if !slices.Equal(rolled.IDs(), []combat.CombatantID{trkA}) {
				t.Errorf("IDs = %v", rolled.IDs())
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-008, FR-009, FR-138 - an entry naming several ids is rolled once and all share the count; separate entries are rolled in the order listed, one request each.
// Path: happy.
func TestRollInitiativeGroupsAndOrder(t *testing.T) {
	t.Parallel()
	group := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkA, trkB}, Modifier: mod(1)})
	single := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkC}, Surprise: combat.Surprised})
	roller := dicetest.NewFixedRoller(7, 4, 15)
	got := trkRoll(t, roller, trkRollCommand(t, combat.InitiativeMethodRolled, group, single))
	if len(got) != 2 || got[0].Count() != 8 || got[1].Count() != 4 || !slices.Equal(got[0].IDs(), []combat.CombatantID{trkA, trkB}) {
		t.Fatalf("breakdowns = %+v, want the group at 8 and the single at 4 [FR-008, FR-009]", got)
	}
	var exprs []string
	for _, e := range roller.Requests() {
		exprs = append(exprs, e.String())
	}
	if !slices.Equal(exprs, []string{"1d20", "2d20"}) {
		t.Errorf("requests = %v, want 1d20 for the group then 2d20 for the surprised single [FR-009, FR-138]", exprs)
	}
}

// Requirement: per case (table 4.1c, FR-010)
// Path: per case.
func TestRollInitiativeScore(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	cases := []struct {
		name, path string
		in         combat.InitiativeParticipantInput
		want       int
		mode       modifier.RollMode
	}{
		{"plain", "happy", combat.InitiativeParticipantInput{Modifier: mod(2)}, 12, modifier.Normal},
		{"zero modifier", "happy", combat.InitiativeParticipantInput{}, 10, modifier.Normal},
		{"Advantage", "happy", combat.InitiativeParticipantInput{Modifier: mod(2), Circumstances: []modifier.RollCircumstance{adv}}, 17, modifier.WithAdvantage},
		{"Disadvantage", "happy", combat.InitiativeParticipantInput{Modifier: mod(2), Surprise: combat.Surprised}, 7, modifier.WithDisadvantage},
		{"both cancel", "happy", combat.InitiativeParticipantInput{Modifier: mod(2), Surprise: combat.Surprised, Circumstances: []modifier.RollCircumstance{adv}}, 12, modifier.Normal},
		{"two Disadvantage, -5 once", "unhappy (rule interaction)", combat.InitiativeParticipantInput{Modifier: mod(2), Surprise: combat.Surprised, Circumstances: []modifier.RollCircumstance{dis}}, 7, modifier.WithDisadvantage},
		{"lowest modifier", "unhappy (boundary)", combat.InitiativeParticipantInput{Modifier: mod(-5)}, 5, modifier.Normal},
		{"lowest with Disadvantage", "unhappy (boundary)", combat.InitiativeParticipantInput{Modifier: mod(-5), Surprise: combat.Surprised}, 0, modifier.WithDisadvantage},
		{"highest with Advantage", "unhappy (boundary)", combat.InitiativeParticipantInput{Modifier: mod(10), Circumstances: []modifier.RollCircumstance{adv}}, 25, modifier.WithAdvantage},
		{"non-Dexterity modifier with Advantage", "unhappy (rule interaction)", combat.InitiativeParticipantInput{Modifier: mod(7), Circumstances: []modifier.RollCircumstance{adv}}, 22, modifier.WithAdvantage},
		{"roll-only modifier ignored", "unhappy (rule interaction)", combat.InitiativeParticipantInput{Modifier: mod(2), RollOnly: mod(-4)}, 12, modifier.Normal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := c.in
			in.IDs = []combat.CombatantID{trkA}
			roller := dicetest.NewFixedRoller()
			got := trkRoll(t, roller, trkRollCommand(t, combat.InitiativeMethodScore, trkParticipant(t, in)))
			scored, ok := got[0].(combat.ScoredInitiative)
			if !ok || scored.Count() != combat.InitiativeCount(c.want) || scored.Mode() != c.mode || scored.Method() != combat.InitiativeMethodScore || scored.InitiativeModifier() != c.in.Modifier {
				t.Errorf("breakdown = %#v, want a ScoredInitiative of %d with mode %v [FR-010, FR-012]", got[0], c.want, c.mode)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("a score rolled %v, want no request [FR-010]", roller.Requests())
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-013, FR-014 - no participant, a zero participant, an undefined method or an id twice (in one entry or two) is an invalid-input error.
// Path: unhappy (invalid input).
func TestNewRollInitiativeCommandRejects(t *testing.T) {
	t.Parallel()
	a := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkA, trkB}})
	b := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkB}})
	cases := map[string]struct {
		method combat.InitiativeMethod
		ps     []combat.InitiativeParticipant
	}{
		"no participants":   {combat.InitiativeMethodRolled, nil},
		"zero participant":  {combat.InitiativeMethodRolled, []combat.InitiativeParticipant{{}}},
		"no method":         {0, []combat.InitiativeParticipant{a}},
		"id in two entries": {combat.InitiativeMethodRolled, []combat.InitiativeParticipant{a, b}},
	}
	for name, c := range cases {
		if _, err := combat.NewRollInitiativeCommand(c.method, c.ps); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want invalid input", name, err)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-015, FR-016, FR-106 - a roller that fails, panics or answers another expression is a *srderr.PortError for dice.Port, returned with no result, recorded once on the span and once in srd5e.combat.errors, never logged, and no count is recorded; a wrong expression is a contract violation, never invalid input.
// Path: unhappy (collaborator failure).
func TestRollInitiativePortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	cmd := trkRollCommand(t, combat.InitiativeMethodRolled, trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkA}}))
	rollers := map[string]combat.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) }),
		"wrong expression": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustParseExpression("1d6"), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 3)})
		}),
	}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			uc, _ := combat.NewRollInitiative(roller, opts...)
			res, err := uc.Run(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port {
				t.Fatalf("= %v, %v, want nil and a dice_roller *PortError", res, err)
			}
			if (name == "wrong expression" && (!errors.Is(err, srderr.ErrContractViolation) || errors.Is(err, srderr.ErrInvalidInput))) || (name == "fails" && !errors.Is(err, cause)) {
				t.Errorf("cause not as specified: %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.roll_initiative")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
			}
			errs := measured(rec, "srd5e.combat.errors")
			if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != "roll_initiative" {
				t.Errorf("errors = %v, want one with error.type", errs)
			}
			for _, n := range []string{"operations", "events", "initiative_counts", "initiative_participants"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
					t.Errorf("%s recorded %d on an error [FR-106]", n, len(m))
				}
			}
			trkHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-097, FR-133 - NewRollInitiative rejects a nil roller; the zero use case and zero command are invalid input before any signal; a cancelled ctx returns ctx.Err() before any signal and before the roller is called.
// Path: unhappy (invalid input, cancellation).
func TestRollInitiativeGuards(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewRollInitiative(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v, want invalid input", err)
	}
	cmd := trkRollCommand(t, combat.InitiativeMethodRolled, trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkA}}))
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(10)
	uc, _ := combat.NewRollInitiative(roller, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) || len(roller.Requests()) != 0 || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v with %d requests and %d spans, want ctx.Err() and nothing", res, err, len(roller.Requests()), len(rec.Tracer.Spans()))
	}
	if _, err := uc.Run(context.Background(), combat.RollInitiativeCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v, want invalid input", err)
	}
	if _, err := (combat.RollInitiative{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, want invalid input", err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-017, FR-096 - rolling does not alter the command; the breakdowns and their lists are copies.
// Path: unhappy (immutability).
func TestRollInitiativeCopies(t *testing.T) {
	t.Parallel()
	ids := []combat.CombatantID{trkA}
	p := trkParticipant(t, combat.InitiativeParticipantInput{IDs: ids})
	list := []combat.InitiativeParticipant{p}
	cmd := trkRollCommand(t, combat.InitiativeMethodRolled, list...)
	list[0] = combat.InitiativeParticipant{}
	uc, _ := combat.NewRollInitiative(dicetest.NewFixedRoller(10, 11))
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	b := res.(combat.RollInitiativeRolled).Breakdowns()
	b[0] = nil
	if got := res.(combat.RollInitiativeRolled).Breakdowns(); got[0] == nil || got[0].Count() != 10 {
		t.Errorf("breakdowns share their list [FR-017]")
	}
	again, _ := uc.Run(context.Background(), cmd)
	if !reflect.DeepEqual(again.(combat.RollInitiativeRolled).Breakdowns()[0].IDs(), []combat.CombatantID{trkA}) {
		t.Error("the command changed after a run")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-099, FR-101, FR-104, FR-107, FR-108 - RollInitiative records the shared signals, one initiative_counts per entry (also for a score) with the method and roll mode, and one initiative_participants with the number of ids; no combatant id is an attribute; Noop and a recording bundle return equal results.
// Path: per case.
func TestRollInitiativeSignals(t *testing.T) {
	t.Parallel()
	group := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkA, trkB, trkC}})
	single := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{"d"}, Surprise: combat.Surprised})
	for _, method := range []combat.InitiativeMethod{combat.InitiativeMethodRolled, combat.InitiativeMethodScore} {
		t.Run(method.String(), func(t *testing.T) {
			t.Parallel()
			cmd := trkRollCommand(t, method, group, single)
			rec, opts := recording()
			uc, _ := combat.NewRollInitiative(dicetest.NewFixedRoller(10, 4, 15), opts...)
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			plain, _ := combat.NewRollInitiative(dicetest.NewFixedRoller(10, 4, 15))
			if want, _ := plain.Run(context.Background(), cmd); !reflect.DeepEqual(res, want) {
				t.Errorf("recording %#v and Noop %#v disagree [FR-107]", res, want)
			}
			wantSpan := map[string]any{"srd5e.combat.operation": "roll_initiative", "srd5e.outcome": "success", "srd5e.combat.result": "rolled", "srd5e.combat.result_class": "applied"}
			spans := rec.Tracer.SpansNamed("srd5e.combat.roll_initiative")
			if len(spans) != 1 || spans[0].Status == telemetry.StatusError || !reflect.DeepEqual(attrMap(spans[0].Attrs), wantSpan) {
				t.Errorf("spans = %+v, want one ok span with %v [FR-099]", spans, wantSpan)
			}
			counts := measured(rec, "srd5e.combat.initiative_counts")
			wantCounts := []map[string]any{
				{"srd5e.combat.initiative_method": method.String(), d20.AttrRollMode: "normal"},
				{"srd5e.combat.initiative_method": method.String(), d20.AttrRollMode: "disadvantage"}}
			if !reflect.DeepEqual(counts, wantCounts) {
				t.Errorf("initiative_counts = %v, want %v [FR-101]", counts, wantCounts)
			}
			parts := rec.Meter.MeasurementsNamed("srd5e.combat.initiative_participants")
			if len(parts) != 1 || parts[0].Value != 4 || !reflect.DeepEqual(attrMap(parts[0].Attrs), map[string]any{"srd5e.combat.initiative_method": method.String()}) {
				t.Errorf("initiative_participants = %+v, want one sample of 4 [FR-101]", parts)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0", n)
			}
			trkHygiene(t, rec)
		})
	}
}
