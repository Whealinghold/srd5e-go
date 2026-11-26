package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func trkSwap(t *testing.T, standings []combat.InitiativeStanding, swapper, ally combat.CombatantID, sav, aav combat.ConditionActionAvailability, c combat.AllyConsent, opts ...combat.Option) (combat.SwapInitiativeResult, error) {
	t.Helper()
	cmd, err := combat.NewSwapInitiativeCommand(standings, swapper, ally, character.InitiativeSwap{}, sav, aav, c)
	if err != nil {
		t.Fatalf("NewSwapInitiativeCommand: %v", err)
	}
	return combat.NewSwapInitiative(opts...).Run(context.Background(), cmd)
}

// Requirement: doc/specs/initiative-and-actions.md FR-156 - the swap exchanges the two counts, keeps the list order and reports InitiativeSwapped(swapper, ally, swapper's new count, ally's new count); the command's list is not altered.
// Path: happy.
func TestSwapInitiativeExchangesCounts(t *testing.T) {
	t.Parallel()
	st := []combat.InitiativeStanding{{Combatant: "a", Count: 15}, {Combatant: "b", Count: 10}, {Combatant: "c", Count: 5}}
	snapshot := slices.Clone(st)
	rec, opts := recording()
	res, err := trkSwap(t, st, "a", "c", trkAvail, trkAvail, combat.AllyConsentWilling, opts...)
	if err != nil {
		t.Fatal(err)
	}
	applied := res.(combat.SwapInitiativeApplied)
	want := []combat.InitiativeStanding{{Combatant: "a", Count: 5}, {Combatant: "b", Count: 10}, {Combatant: "c", Count: 15}}
	if !slices.Equal(applied.Standings(), want) || !slices.Equal(st, snapshot) {
		t.Errorf("standings = %v, input %v; want %v and the input unchanged", applied.Standings(), st, want)
	}
	e := applied.Events()[0].(combat.InitiativeSwapped)
	if e.FirstCount() != 5 || e.SecondCount() != 15 || !slices.Equal(e.Combatants(), []combat.CombatantID{"a", "c"}) || len(applied.Events()) != 1 {
		t.Errorf("event = %+v [FR-156]", e)
	}
	applied.Standings()[0].Count = 99
	applied.Events()[0] = nil
	if applied.Standings()[0].Count != 5 || applied.Events()[0] == nil {
		t.Error("the result shares its lists")
	}
	plain, _ := trkSwap(t, st, "a", "c", trkAvail, trkAvail, combat.AllyConsentWilling)
	if !reflect.DeepEqual(res, plain) {
		t.Errorf("recording %#v and Noop %#v disagree [FR-107]", res, plain)
	}
	want2 := map[string]any{"srd5e.combat.operation": "swap_initiative", "srd5e.outcome": "success", "srd5e.combat.result": "applied", "srd5e.combat.result_class": "applied"}
	if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want2) {
		t.Errorf("operations = %v, want %v [FR-164]", ops, want2)
	}
	trkHygiene(t, rec)
}

// Requirement: per case (FR-156, FR-118, FR-164): refusals in order UnknownCombatant (swapper, then ally), incapacitated (swapper, then ally), AllyNotWilling; unknown is class invalid, the others refused.
// Path: per case.
func TestSwapInitiativeRefusals(t *testing.T) {
	t.Parallel()
	st := []combat.InitiativeStanding{{Combatant: "a", Count: 15}, {Combatant: "b", Count: 10}}
	cases := []struct {
		name          string
		swapper, ally combat.CombatantID
		sav, aav      combat.ConditionActionAvailability
		consent       combat.AllyConsent
		want          combat.TurnRejection
		class         string
	}{
		{"swapper incapacitated", "a", "b", trkNone, trkAvail, combat.AllyConsentWilling, combat.TurnIncapacitatedRejection{}, "refused"},
		{"ally incapacitated", "a", "b", trkAvail, trkNone, combat.AllyConsentWilling, combat.TurnIncapacitatedRejection{}, "refused"},
		{"ally unwilling", "a", "b", trkAvail, trkAvail, combat.AllyConsentUnwilling, combat.TurnAllyNotWillingRejection{}, "refused"},
		{"incapacitated beats unwilling", "a", "b", trkNone, trkAvail, combat.AllyConsentUnwilling, combat.TurnIncapacitatedRejection{}, "refused"},
		{"ally not in the standings", "a", "x", trkAvail, trkAvail, combat.AllyConsentWilling, combat.TurnUnknownCombatantRejection{ID: "x"}, "invalid"},
		{"swapper not in the standings", "y", "b", trkAvail, trkAvail, combat.AllyConsentWilling, combat.TurnUnknownCombatantRejection{ID: "y"}, "invalid"},
		{"swapper unknown beats ally unknown", "y", "x", trkNone, trkNone, combat.AllyConsentUnwilling, combat.TurnUnknownCombatantRejection{ID: "y"}, "invalid"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := trkSwap(t, st, c.swapper, c.ally, c.sav, c.aav, c.consent, opts...)
			if err != nil {
				t.Fatal(err)
			}
			if r, ok := res.(combat.SwapInitiativeRejected); !ok || r.Reason != c.want {
				t.Fatalf("result = %#v, want rejected with %#v [FR-156]", res, c.want)
			}
			ops := measured(rec, "srd5e.combat.operations")
			if len(ops) != 1 || ops[0]["srd5e.combat.result_class"] != c.class || ops[0]["srd5e.combat.result"] != c.want.String() {
				t.Errorf("operations = %v, want class %s and result %s [FR-164]", ops, c.class, c.want)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.swap_initiative")
			if len(spans) != 1 || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") {
				t.Errorf("spans = %+v", spans)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0", n)
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-156 - the constructor returns an invalid-input error for the same id twice, a blank id, no or a repeated standing, or an availability or consent that is not defined; the zero command and a cancelled ctx fail before any signal.
// Path: unhappy (invalid input).
func TestSwapInitiativeCommand(t *testing.T) {
	t.Parallel()
	st := []combat.InitiativeStanding{{Combatant: "a", Count: 15}, {Combatant: "b", Count: 10}}
	eff := character.InitiativeSwap{}
	bad := map[string]func() error{
		"same id": func() error {
			_, e := combat.NewSwapInitiativeCommand(st, "a", "a", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"blank swapper": func() error {
			_, e := combat.NewSwapInitiativeCommand(st, "", "b", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"blank ally": func() error {
			_, e := combat.NewSwapInitiativeCommand(st, "a", "", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"no standings": func() error {
			_, e := combat.NewSwapInitiativeCommand(nil, "a", "b", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"repeated": func() error {
			_, e := combat.NewSwapInitiativeCommand([]combat.InitiativeStanding{st[0], st[0]}, "a", "b", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"blank standing": func() error {
			_, e := combat.NewSwapInitiativeCommand([]combat.InitiativeStanding{{}}, "a", "b", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"swapper no avail": func() error {
			_, e := combat.NewSwapInitiativeCommand(st, "a", "b", eff, 0, trkAvail, combat.AllyConsentWilling)
			return e
		},
		"ally no avail": func() error {
			_, e := combat.NewSwapInitiativeCommand(st, "a", "b", eff, trkAvail, 0, combat.AllyConsentWilling)
			return e
		},
		"no consent": func() error {
			_, e := combat.NewSwapInitiativeCommand(st, "a", "b", eff, trkAvail, trkAvail, 0)
			return e
		},
	}
	for name, f := range bad {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, want invalid input", name, err)
		}
	}
	if _, err := combat.NewSwapInitiative().Run(context.Background(), combat.SwapInitiativeCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v", err)
	}
	rec, opts := recording()
	cmd, _ := combat.NewSwapInitiativeCommand(st, "a", "b", eff, trkAvail, trkAvail, combat.AllyConsentWilling)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := combat.NewSwapInitiative(opts...).Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	// A swap that creates a tie leaves it to EstablishInitiativeOrder (FR-156, FR-019).
	tied := []combat.InitiativeStanding{{Combatant: "a", Count: 10}, {Combatant: "b", Count: 10}, {Combatant: "c", Count: 5}}
	res, err := trkSwap(t, tied, "a", "c", trkAvail, trkAvail, combat.AllyConsentWilling)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.(combat.SwapInitiativeApplied).Standings(); got[0].Count != 5 || got[2].Count != 10 {
		t.Errorf("standings = %v", got)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-156 - the swapper and the ally are found by id wherever they stand in the list, the ally may be the first standing and an id that sorts before every listed id is unknown.
// Path: unhappy (boundary).
func TestSwapInitiativeFindsByIDWhereverListed(t *testing.T) {
	t.Parallel()
	st := []combat.InitiativeStanding{{Combatant: "c", Count: 3}, {Combatant: "a", Count: 1}, {Combatant: "b", Count: 2}}
	cases := []struct {
		name          string
		swapper, ally combat.CombatantID
		want          []combat.InitiativeStanding
	}{
		{"swapper first, ally last", "c", "b", []combat.InitiativeStanding{{Combatant: "c", Count: 2}, {Combatant: "a", Count: 1}, {Combatant: "b", Count: 3}}},
		{"ally first", "a", "c", []combat.InitiativeStanding{{Combatant: "c", Count: 1}, {Combatant: "a", Count: 3}, {Combatant: "b", Count: 2}}},
		{"swapper after a larger id", "a", "b", []combat.InitiativeStanding{{Combatant: "c", Count: 3}, {Combatant: "a", Count: 2}, {Combatant: "b", Count: 1}}},
		{"ally after a larger id", "b", "a", []combat.InitiativeStanding{{Combatant: "c", Count: 3}, {Combatant: "a", Count: 2}, {Combatant: "b", Count: 1}}},
	}
	for _, c := range cases {
		res, err := trkSwap(t, st, c.swapper, c.ally, trkAvail, trkAvail, combat.AllyConsentWilling)
		if err != nil {
			t.Fatal(err)
		}
		if got := res.(combat.SwapInitiativeApplied).Standings(); !slices.Equal(got, c.want) {
			t.Errorf("%s: standings = %v, want %v", c.name, got, c.want)
		}
	}
	for _, id := range []combat.CombatantID{"A", "0"} {
		if res, _ := trkSwap(t, st, id, "a", trkAvail, trkAvail, combat.AllyConsentWilling); res.(combat.SwapInitiativeRejected).Reason != (combat.TurnUnknownCombatantRejection{ID: id}) {
			t.Errorf("swapper %q is not unknown", id)
		}
		if res, _ := trkSwap(t, st, "a", id, trkAvail, trkAvail, combat.AllyConsentWilling); res.(combat.SwapInitiativeRejected).Reason != (combat.TurnUnknownCombatantRejection{ID: id}) {
			t.Errorf("ally %q is not unknown", id)
		}
	}
}
