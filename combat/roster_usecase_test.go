package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// rosAddRun, rosRemoveRun and rosMoveRun build one call of a roster use case.
func rosAddRun(t *testing.T, e combat.Encounter, id string, pos combat.RosterPosition, reason combat.JoinReason) func(...combat.Option) (any, error) {
	t.Helper()
	cmd, err := combat.NewAddCombatantToEncounterCommand(e, rosM(t, id, combat.SideA), rosProfile(t, id), pos, reason)
	if err != nil {
		t.Fatalf("NewAddCombatantToEncounterCommand: %v", err)
	}
	return func(opts ...combat.Option) (any, error) {
		return combat.NewAddCombatantToEncounter(opts...).Run(context.Background(), cmd)
	}
}

func rosRemoveRun(t *testing.T, e combat.Encounter, l combat.ActionLedger, id string, cause combat.RemovalCause) func(...combat.Option) (any, error) {
	t.Helper()
	cmd, err := combat.NewRemoveCombatantFromEncounterCommand(e, l, cid(id), cause)
	if err != nil {
		t.Fatalf("NewRemoveCombatantFromEncounterCommand: %v", err)
	}
	return func(opts ...combat.Option) (any, error) {
		return combat.NewRemoveCombatantFromEncounter(opts...).Run(context.Background(), cmd)
	}
}

func rosMoveRun(t *testing.T, e combat.Encounter, id string, pos combat.RosterPosition) func(...combat.Option) (any, error) {
	t.Helper()
	cmd, err := combat.NewRepositionCombatantInEncounterCommand(e, cid(id), pos)
	if err != nil {
		t.Fatalf("NewRepositionCombatantInEncounterCommand: %v", err)
	}
	return func(opts ...combat.Option) (any, error) {
		return combat.NewRepositionCombatantInEncounter(opts...).Run(context.Background(), cmd)
	}
}

// rosLedger holds a benefit of c helping d and one of a helping c, as the spec's ledger rows do.
func rosLedger(tb testing.TB) combat.ActionLedger {
	tb.Helper()
	l, err := combat.NewActionLedger(nil, benefitsOf(tb, checkHelp(tb, "c", "d", athletics()), checkHelp(tb, "a", "c", athletics()), rollHelp(tb, "a", "d")))
	if err != nil {
		tb.Fatalf("NewActionLedger: %v", err)
	}
	return l
}

// Requirement: per case (table 4.6a, FR-041, FR-042, FR-043, FR-044)
// Path: per case.
func TestRosterUseCaseSignals(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	ended := rosEnded(t, e)
	two := rosStart(t, 0, rosM(t, "a", combat.SideA), rosM(t, "c", combat.SideB))
	add, remove, move := "add_combatant_to_encounter", "remove_combatant_from_encounter", "reposition_combatant_in_encounter"
	const a, r, i = "applied", "refused", "invalid"
	rows := []struct {
		name, requirement, path string
		operation               string
		run                     func(opts ...combat.Option) (any, error)
		class, result, reason   string
		events                  []string
	}{
		{"add applied, arrived", "FR-042 FR-043", "happy", add, rosAddRun(t, e, "x", rosAfter("a"), combat.JoinReasonArrived), a, "applied", "arrived", []string{"combatant_joined"}},
		{"add applied, summoned", "FR-043", "happy", add, rosAddRun(t, e, "x", rosAfter("b"), combat.JoinReasonSummoned), a, "applied", "summoned", []string{"combatant_joined"}},
		{"add ended", "FR-042", "unhappy (state)", add, rosAddRun(t, ended, "x", rosAfter("a"), combat.JoinReasonArrived), r, "encounter_already_ended", "arrived", nil},
		{"add duplicate", "FR-042", "unhappy (state)", add, rosAddRun(t, e, "a", rosAfter("b"), combat.JoinReasonSummoned), i, "duplicate_combatant", "summoned", nil},
		{"add unknown anchor", "FR-042", "unhappy (invalid input)", add, rosAddRun(t, e, "x", rosAfter("z"), combat.JoinReasonArrived), i, "unknown_combatant", "arrived", nil},
		{"remove applied, fled", "FR-042 FR-043", "happy", remove, rosRemoveRun(t, e, combat.ActionLedger{}, "c", combat.RemovalCauseFled), a, "applied", "fled", []string{"combatant_removed"}},
		{"remove applied with a grapple, surrendered", "FR-044", "unhappy (rule interaction)", remove, rosRemoveRun(t, rosGrapple(t, "c", "d"), combat.ActionLedger{}, "c", combat.RemovalCauseSurrendered), a, "applied", "surrendered", []string{"combatant_removed", "condition_removed", "grapple_ended"}},
		{"remove ended", "FR-042", "unhappy (state)", remove, rosRemoveRun(t, ended, combat.ActionLedger{}, "c", combat.RemovalCauseDisappeared), r, "encounter_already_ended", "disappeared", nil},
		{"remove active", "FR-042", "unhappy (state)", remove, rosRemoveRun(t, e, combat.ActionLedger{}, "b", combat.RemovalCauseDeparted), r, "combatant_is_active", "departed", nil},
		{"remove would empty a side", "FR-042", "unhappy (boundary)", remove, rosRemoveRun(t, two, combat.ActionLedger{}, "c", combat.RemovalCauseFled), r, "would_empty_side", "fled", nil},
		{"remove unknown", "FR-042", "unhappy (invalid input)", remove, rosRemoveRun(t, e, combat.ActionLedger{}, "z", combat.RemovalCauseFled), i, "unknown_combatant", "fled", nil},
		{"reposition applied", "FR-042 FR-043", "happy", move, rosMoveRun(t, e, "d", rosAfter("b")), a, "applied", "repositioned", []string{"combatant_repositioned"}},
		{"reposition applied, no change", "FR-032 FR-044", "unhappy (boundary)", move, rosMoveRun(t, e, "c", rosAt(10, rosBefore)), a, "applied", "repositioned", []string{"combatant_repositioned"}},
		{"reposition ended", "FR-042", "unhappy (state)", move, rosMoveRun(t, ended, "c", rosAfter("a")), r, "encounter_already_ended", "repositioned", nil},
		{"reposition active", "FR-042", "unhappy (state)", move, rosMoveRun(t, e, "b", rosAfter("a")), r, "combatant_is_active", "repositioned", nil},
		{"reposition unknown", "FR-042", "unhappy (invalid input)", move, rosMoveRun(t, e, "z", rosAfter("a")), i, "unknown_combatant", "repositioned", nil},
	}
	for _, c := range rows {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := c.run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-044]", got, plain, err)
			}
			outcome := "failure"
			if c.class == "applied" {
				outcome = "success"
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result,
				"srd5e.combat.result_class": c.class, "srd5e.combat.roster_reason": c.reason}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, with error status only for invalid and no recorded error [FR-042; %s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [FR-043]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [FR-042, FR-043]", ops, want)
			}
			wantKinds := slices.Clone(c.events)
			slices.Sort(wantKinds)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, wantKinds) {
				t.Errorf("event kinds counted = %v, want %v [FR-044]", counted, wantKinds)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want none for a result [FR-042]", n)
			}
			checkTelemetryHygiene(t, rec)
			for _, m := range rec.Meter.Measurements() {
				for _, at := range m.Attrs {
					for _, id := range []string{"a", "b", "c", "d", "x", "z"} {
						if at.Value() == id {
							t.Errorf("%s attribute %s carries the id %q [FR-043]", m.Name, at.Key(), id)
						}
					}
				}
			}
		})
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-041, FR-008, FR-016 - each use case returns its own result: Applied holds the new encounter and the events; the command's encounter is not altered by Run.
// Path: happy.
func TestRosterUseCasesReturnTheirOwnResults(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	before := rosEnc(t)
	ctx := context.Background()
	addCmd, _ := combat.NewAddCombatantToEncounterCommand(e, rosM(t, "x", combat.SideA), rosProfile(t, "x"), rosAfter("a"), combat.JoinReasonSummoned)
	res, err := combat.NewAddCombatantToEncounter().Run(ctx, addCmd)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := res.(combat.AddCombatantToEncounterApplied); !ok || rosOrder(a.Encounter()) != "a,x,b,c,d" || len(a.Events()) != 1 {
		t.Errorf("add result = %#v, want AddCombatantToEncounterApplied with x after a", res)
	}
	rmCmd, _ := combat.NewRemoveCombatantFromEncounterCommand(e, combat.ActionLedger{}, cid("c"), combat.RemovalCauseFled)
	rres, err := combat.NewRemoveCombatantFromEncounter().Run(ctx, rmCmd)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := rres.(combat.RemoveCombatantFromEncounterApplied); !ok || rosOrder(a.Encounter()) != "a,b,d" || !a.Ledger().Equal(combat.ActionLedger{}) || len(a.LedgerEvents()) != 0 {
		t.Errorf("remove result = %#v, want RemoveCombatantFromEncounterApplied without c", rres)
	}
	mvCmd, _ := combat.NewRepositionCombatantInEncounterCommand(e, cid("d"), rosAfter("a"))
	mres, err := combat.NewRepositionCombatantInEncounter().Run(ctx, mvCmd)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := mres.(combat.RepositionCombatantInEncounterApplied); !ok || rosOrder(a.Encounter()) != "a,d,b,c" || len(a.Events()) != 1 {
		t.Errorf("reposition result = %#v, want RepositionCombatantInEncounterApplied with d after a", mres)
	}
	if !e.Equal(before) {
		t.Errorf("a use case altered the encounter of its command [FR-037]")
	}
	// A refusal holds the encounter that came in and the rejection (FR-041).
	rej, _ := combat.NewAddCombatantToEncounter().Run(ctx, addCmdFor(t, e, "a"))
	if r, ok := rej.(combat.AddCombatantToEncounterRejected); !ok || r.Rejection != (combat.RosterDuplicateCombatantRejection{ID: "a"}) || !r.Encounter().Equal(e) {
		t.Errorf("duplicate add = %#v, want AddCombatantToEncounterRejected holding the encounter and the duplicate", rej)
	}
}

func addCmdFor(t *testing.T, e combat.Encounter, id string) combat.AddCombatantToEncounterCommand {
	t.Helper()
	cmd, err := combat.NewAddCombatantToEncounterCommand(e, rosM(t, id, combat.SideA), rosProfile(t, id), rosAfter("b"), combat.JoinReasonArrived)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// Requirement: doc/specs/combatant-roster-changes.md FR-025 - a removal drops the leaver's pending Help benefits from the ledger (as helper, ally or enemy) and returns their HelpBenefitDropped events apart from the encounter's events, equal to HelpBenefits.DropFor, uncounted by srd5e.combat.events; a refusal returns the ledger it was given, equal.
// Path: happy.
func TestRemovalDropsTheLeaversHelpBenefits(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	ledger := rosLedger(t)
	rec, opts := recording()
	cmd, err := combat.NewRemoveCombatantFromEncounterCommand(e, ledger, cid("c"), combat.RemovalCauseFled)
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewRemoveCombatantFromEncounter(opts...).Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := res.(combat.RemoveCombatantFromEncounterApplied)
	if !ok {
		t.Fatalf("result = %#v, want applied", res)
	}
	drop := ledger.HelpBenefits().DropFor(cid("c"))
	if !a.Ledger().HelpBenefits().Equal(drop.Benefits()) || len(a.LedgerEvents()) != 2 || !slices.Equal(a.LedgerEvents(), drop.Events()) {
		t.Errorf("ledger %#v, events %v: want the two benefits naming c dropped, as DropFor says [FR-025]", a.Ledger(), a.LedgerEvents())
	}
	if !a.Ledger().HelpBenefits().Equal(benefitsOf(t, rollHelp(t, "a", "d"))) {
		t.Errorf("the ledger keeps %v, want only the benefit that does not name c", a.Ledger().HelpBenefits())
	}
	for _, ev := range a.Events() {
		if _, isHelp := any(ev).(combat.HelpEvent); isHelp {
			t.Errorf("event %T is a HelpEvent: ledger events are not in Events [FR-025]", ev)
		}
	}
	if got := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(got, []string{"combatant_removed"}) {
		t.Errorf("counted events %v, want only combatant_removed: HelpBenefitDropped is not counted [FR-044]", got)
	}
	if got := a.LedgerEvents(); len(got) > 0 {
		got[0] = nil
		if a.LedgerEvents()[0] == nil {
			t.Errorf("LedgerEvents returns the stored slice, not a copy")
		}
	}
	// A refusal returns the ledger it was given.
	refused, err := combat.NewRemoveCombatantFromEncounter().Run(context.Background(), mustRemoveCmd(t, e, ledger, "b"))
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := refused.(combat.RemoveCombatantFromEncounterRejected); !ok || !r.Ledger().Equal(ledger) || !r.Encounter().Equal(e) || r.Rejection != (combat.RosterCombatantIsActiveRejection{ID: "b"}) {
		t.Errorf("refused result = %#v, want the ledger and encounter that came in [FR-025]", refused)
	}
}

func mustRemoveCmd(t *testing.T, e combat.Encounter, l combat.ActionLedger, id string) combat.RemoveCombatantFromEncounterCommand {
	t.Helper()
	cmd, err := combat.NewRemoveCombatantFromEncounterCommand(e, l, cid(id), combat.RemovalCauseFled)
	if err != nil {
		t.Fatal(err)
	}
	return cmd
}

// Requirement: doc/specs/combatant-roster-changes.md FR-041, FR-042 - the zero command is invalid input before any signal, and a cancelled context is returned unchanged before any signal, for each use case.
// Path: unhappy (invalid input, cancellation).
func TestRosterUseCasesRejectTheZeroCommandAndACancelledContext(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	type call func(ctx context.Context, opts ...combat.Option) error
	addOK, _ := combat.NewAddCombatantToEncounterCommand(e, rosM(t, "x", combat.SideA), rosProfile(t, "x"), rosAfter("a"), combat.JoinReasonArrived)
	rmOK := mustRemoveCmd(t, e, combat.ActionLedger{}, "c")
	mvOK, _ := combat.NewRepositionCombatantInEncounterCommand(e, cid("d"), rosAfter("a"))
	calls := map[string][2]call{
		"add": {
			func(ctx context.Context, o ...combat.Option) error {
				_, err := combat.NewAddCombatantToEncounter(o...).Run(ctx, combat.AddCombatantToEncounterCommand{})
				return err
			},
			func(ctx context.Context, o ...combat.Option) error {
				_, err := combat.NewAddCombatantToEncounter(o...).Run(ctx, addOK)
				return err
			},
		},
		"remove": {
			func(ctx context.Context, o ...combat.Option) error {
				_, err := combat.NewRemoveCombatantFromEncounter(o...).Run(ctx, combat.RemoveCombatantFromEncounterCommand{})
				return err
			},
			func(ctx context.Context, o ...combat.Option) error {
				_, err := combat.NewRemoveCombatantFromEncounter(o...).Run(ctx, rmOK)
				return err
			},
		},
		"reposition": {
			func(ctx context.Context, o ...combat.Option) error {
				_, err := combat.NewRepositionCombatantInEncounter(o...).Run(ctx, combat.RepositionCombatantInEncounterCommand{})
				return err
			},
			func(ctx context.Context, o ...combat.Option) error {
				_, err := combat.NewRepositionCombatantInEncounter(o...).Run(ctx, mvOK)
				return err
			},
		},
	}
	for name, c := range calls {
		rec, opts := recording()
		if err := c[0](context.Background(), opts...); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: zero command err = %v, want ErrInvalidInput", name, err)
		}
		if err := c[1](cancelled, opts...); !errors.Is(err, context.Canceled) {
			t.Errorf("%s: cancelled ctx err = %v, want context.Canceled unwrapped", name, err)
		}
		if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
			t.Errorf("%s: a rejected call recorded a signal [FR-042]", name)
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-011, plan P10 - a roster at 255 combatants accepts an add (the cap is 256); a roster at 256 refuses it with the invalid result limit_exceeded and the encounter unchanged; a removal frees room for a later add.
// Path: unhappy (boundary).
func TestAddAtTheCapAndOverIt(t *testing.T) {
	t.Parallel()
	none := func(int) (string, bool) { return "", false }
	below, atCap := rosBig(t, 255, none), rosBig(t, 256, none)
	res, err := rosAddRun(t, below, "x", rosAfter("m0000"), combat.JoinReasonArrived)()
	if a, ok := res.(combat.AddCombatantToEncounterApplied); !ok || err != nil || len(a.Encounter().Combatants()) != 256 {
		t.Fatalf("add at 255 = %#v (%v), want applied making 256", res, err)
	}
	rec, opts := recording()
	res, err = rosAddRun(t, atCap, "x", rosAfter("m0000"), combat.JoinReasonArrived)(opts...)
	r, ok := res.(combat.AddCombatantToEncounterRejected)
	if !ok || err != nil || r.Rejection != (combat.RosterLimitExceededRejection{}) || !r.Encounter().Equal(atCap) {
		t.Fatalf("add at 256 = %#v (%v), want rejected with limit_exceeded and the encounter unchanged", res, err)
	}
	ops := measured(rec, "srd5e.combat.operations")
	if len(ops) != 1 || ops[0]["srd5e.combat.result"] != "limit_exceeded" || ops[0]["srd5e.combat.result_class"] != "invalid" || ops[0]["srd5e.outcome"] != "failure" {
		t.Errorf("operations = %v, want one invalid limit_exceeded", ops)
	}
	if spans := rec.Tracer.SpansNamed("srd5e.combat.add_combatant_to_encounter"); len(spans) != 1 || spans[0].Status != telemetry.StatusError {
		t.Errorf("spans = %+v, want error status for an invalid result", spans)
	}
	// The duplicate and unknown anchors are reported before the cap (FR-009 order).
	tr, err := atCap.AddCombatant(rosM(t, "m0001", combat.SideA), rosProfile(t, "m0001"), rosAfter("m0000"), combat.JoinReasonArrived)
	if r := rosNo(t, atCap, tr, err); r != (combat.RosterDuplicateCombatantRejection{ID: "m0001"}) {
		t.Errorf("rejection = %#v, want the duplicate before the cap", r)
	}
	freed, _ := rosOK(t)(atCap.RemoveCombatant(cid("m0003"), combat.RemovalCauseFled))
	if again, _ := rosOK(t)(rosAdd(t, freed, "x", combat.SideA, rosAfter("m0000"))); len(again.Combatants()) != 256 {
		t.Errorf("after a removal the add makes %d combatants, want 256", len(again.Combatants()))
	}
}

// Requirement: doc/specs/combatant-roster-changes.md plan P10 (events per call 4096) - a removal that would produce 4095 events (2047 grapples ended and the removal) is applied; one that would produce 4097 is refused with limit_exceeded and changes nothing.
// Path: unhappy (boundary).
func TestRemovalAtTheEventCapAndOverIt(t *testing.T) {
	t.Parallel()
	holding := func(n int) combat.Encounter {
		return rosBig(t, n+2, func(i int) (string, bool) { return "m0001", i >= 2 }) // everyone but m0000 and m0001 is grappled by m0001
	}
	atCap := holding(2047)
	after, events := rosOK(t)(atCap.RemoveCombatant(cid("m0001"), combat.RemovalCauseFled))
	if len(events) != 4095 || len(after.Combatants()) != len(atCap.Combatants())-1 {
		t.Fatalf("events = %d, want 4095 (2047 endings of two events and the removal)", len(events))
	}
	over := holding(2048)
	tr, err := over.RemoveCombatant(cid("m0001"), combat.RemovalCauseFled)
	if r := rosNo(t, over, tr, err); r != (combat.RosterLimitExceededRejection{}) {
		t.Errorf("rejection = %#v, want RosterLimitExceededRejection at 4097 events", r)
	}
	res, err := rosRemoveRun(t, over, combat.ActionLedger{}, "m0001", combat.RemovalCauseFled)()
	if r, ok := res.(combat.RemoveCombatantFromEncounterRejected); !ok || err != nil || r.Rejection.String() != "limit_exceeded" {
		t.Errorf("use case result = %#v (%v), want rejected with limit_exceeded", res, err)
	}
}

// panickingTracer panics when a span starts: a host's broken backend.
type panickingTracer struct{}

func (panickingTracer) Start(context.Context, string, ...telemetry.Attr) (context.Context, telemetry.Span) {
	panic(errors.New("tracer down"))
}

// Requirement: doc/specs/combatant-roster-changes.md FR-046 - the use cases return no port failure, and a panic of a host's telemetry propagates unchanged (a bug is not turned into a result), leaving the encounter and ledger of the command as they were.
// Path: unhappy (collaborator failure).
func TestAPanicFromTelemetryPropagatesAndChangesNothing(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	before := rosEnc(t)
	ledger := rosLedger(t)
	opt := combat.WithTelemetry(telemetry.Telemetry{Tracer: panickingTracer{}})
	for name, call := range map[string]func(){
		"add":        func() { _, _ = rosAddRun(t, e, "x", rosAfter("a"), combat.JoinReasonArrived)(opt) },
		"remove":     func() { _, _ = rosRemoveRun(t, e, ledger, "c", combat.RemovalCauseFled)(opt) },
		"reposition": func() { _, _ = rosMoveRun(t, e, "d", rosAfter("a"))(opt) },
	} {
		func() {
			defer func() {
				if v := recover(); v == nil {
					t.Errorf("%s: the panic of the tracer was swallowed", name)
				} else if err, ok := v.(error); !ok || err.Error() != "tracer down" {
					t.Errorf("%s: recovered %v, want the tracer's own error", name, v)
				}
			}()
			call()
		}()
	}
	if !e.Equal(before) || !ledger.Equal(rosLedger(t)) {
		t.Errorf("a panicking call changed the encounter or the ledger it was given")
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-042, FR-044 - a telemetry bundle with every member nil behaves as no telemetry: the same results, and the zero use cases work.
// Path: unhappy (boundary).
func TestRosterUseCasesAreUsableAsZeroValues(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	cmd, _ := combat.NewAddCombatantToEncounterCommand(e, rosM(t, "x", combat.SideA), rosProfile(t, "x"), rosAfter("a"), combat.JoinReasonArrived)
	zero, err := combat.AddCombatantToEncounter{}.Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	named, err := combat.NewAddCombatantToEncounter(combat.WithTelemetry(telemetrytest.NewRecorder().Telemetry())).Run(context.Background(), cmd)
	if err != nil || !reflect.DeepEqual(zero, named) {
		t.Errorf("zero use case %#v differs from a named one %#v (%v)", zero, named, err)
	}
	if _, err := (combat.RemoveCombatantFromEncounter{}).Run(context.Background(), mustRemoveCmd(t, e, combat.ActionLedger{}, "c")); err != nil {
		t.Errorf("zero RemoveCombatantFromEncounter: %v", err)
	}
	mv, _ := combat.NewRepositionCombatantInEncounterCommand(e, cid("d"), rosAfter("a"))
	if _, err := (combat.RepositionCombatantInEncounter{}).Run(context.Background(), mv); err != nil {
		t.Errorf("zero RepositionCombatantInEncounter: %v", err)
	}
}
