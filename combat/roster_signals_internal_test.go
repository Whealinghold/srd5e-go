package combat

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/combatant-roster-changes.md FR-042, FR-043, section 7 - the three operation names, their spans, the attribute key, the seven values of srd5e.combat.roster_reason and the result values of the roster use cases are pinned to their catalogue spelling (renaming one breaks consumers' dashboards), and each operation and span pair is a valid combatOperation.
// Path: per case.
func TestRosterSignalValuesArePinned(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got, want               string
	}{
		{"add operation", "FR-042", "happy", valueOperationAddCombatantToEncounter, "add_combatant_to_encounter"},
		{"add span", "FR-042", "happy", spanAddCombatantToEncounter, "srd5e.combat.add_combatant_to_encounter"},
		{"remove operation", "FR-042", "happy", valueOperationRemoveCombatantFromEncounter, "remove_combatant_from_encounter"},
		{"remove span", "FR-042", "happy", spanRemoveCombatantFromEncounter, "srd5e.combat.remove_combatant_from_encounter"},
		{"reposition operation", "FR-042", "happy", valueOperationRepositionCombatantInEncounter, "reposition_combatant_in_encounter"},
		{"reposition span", "FR-042", "happy", spanRepositionCombatantInEncounter, "srd5e.combat.reposition_combatant_in_encounter"},
		{"roster_reason key", "FR-043", "happy", attrRosterReason, "srd5e.combat.roster_reason"},
		{"repositioned", "FR-043", "happy", valueRosterReasonRepositioned, "repositioned"},
		{"summoned", "FR-043", "happy", JoinReasonSummoned.String(), "summoned"},
		{"arrived", "FR-043", "happy", JoinReasonArrived.String(), "arrived"},
		{"fled", "FR-043", "happy", RemovalCauseFled.String(), "fled"},
		{"surrendered", "FR-043", "happy", RemovalCauseSurrendered.String(), "surrendered"},
		{"disappeared", "FR-043", "happy", RemovalCauseDisappeared.String(), "disappeared"},
		{"departed", "FR-043", "happy", RemovalCauseDeparted.String(), "departed"},
		{"encounter_already_ended", "FR-042", "unhappy (state)", valueResultRosterEncounterEnded, "encounter_already_ended"},
		{"combatant_is_active", "FR-042", "unhappy (state)", valueResultCombatantIsActive, "combatant_is_active"},
		{"would_empty_side", "FR-042", "unhappy (boundary)", valueResultWouldEmptySide, "would_empty_side"},
		{"duplicate_combatant", "FR-042", "unhappy (state)", valueResultDuplicateCombatant, "duplicate_combatant"},
		{"unknown_combatant", "FR-042", "unhappy (invalid input)", valueResultUnknownCombatant, "unknown_combatant"},
		{"limit_exceeded", "FR-042, plan P10", "unhappy (boundary)", valueResultLimitExceeded, "limit_exceeded"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("%s = %q, want %q [%s%s]", c.name, c.got, c.want, "doc/specs/combatant-roster-changes.md ", c.requirement)
			}
		})
	}
	for _, pair := range [][2]string{
		{valueOperationAddCombatantToEncounter, spanAddCombatantToEncounter},
		{valueOperationRemoveCombatantFromEncounter, spanRemoveCombatantFromEncounter},
		{valueOperationRepositionCombatantInEncounter, spanRepositionCombatantInEncounter},
	} {
		if _, err := newCombatOperation(pair[0], pair[1]); err != nil {
			t.Errorf("operation %s with span %s is not valid: %v", pair[0], pair[1], err)
		}
	}
	if maxCombatantsPerEncounter != 256 {
		t.Errorf("maxCombatantsPerEncounter = %d, want 256 (plan P10)", maxCombatantsPerEncounter)
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-042 table 4.6a - a duplicate id, an unknown combatant and a roster over its cap are class invalid (error status); the ended, active and would-empty-side rejections are refused (no error status); an applied transition is applied.
// Path: per case.
func TestRosterRejectionsAreClassifiedByVariant(t *testing.T) {
	t.Parallel()
	for r, want := range map[RosterRejection]combatResultClass{
		RosterEncounterEndedRejection{}:            resultRefused,
		RosterCombatantIsActiveRejection{ID: "a"}:  resultRefused,
		RosterWouldEmptySideRejection{Side: SideA}: resultRefused,
		RosterUnknownCombatantRejection{ID: "a"}:   resultInvalid,
		RosterDuplicateCombatantRejection{ID: "a"}: resultInvalid,
		RosterLimitExceededRejection{}:             resultInvalid,
	} {
		if got := rosterRejectionClass(r); got != want {
			t.Errorf("%T class = %v, want %v", r, got, want)
		}
		if rosterRejectionClass(r).IsSpanError() != (want == resultInvalid) {
			t.Errorf("%T: error status disagrees with its class", r)
		}
	}
	if ev := (rosterRejected{rejection: RosterEncounterEndedRejection{}}).Events(); ev != nil {
		t.Errorf("a rejected transition has events %v, want none", ev)
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-041, plan P14 - the root validates the encounter at entry and at the exit of a call, never inside it: a zero or forged encounter is invalid input to every roster method and to the use case before any signal and before the work runs; an applied transition to an encounter that is not valid is an error recorded on the span, with no operation or event counted.
// Path: unhappy (invalid input, state forgery).
func TestRosterOperationsValidateTheEncounterAtEntryAndExit(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	forged := map[string]Encounter{"zero value": {}, "no id": good, "order and combatants disagree": good, "nil status": good}
	forged["no id"] = func() Encounter { e := good; e.id = ""; return e }()
	forged["order and combatants disagree"] = func() Encounter { e := good; e.combatants = e.combatants[:1]; return e }()
	forged["nil status"] = func() Encounter { e := good; e.status = nil; return e }()
	x, err := NewCombatant(CombatantRef{ID: "x", Kind: MonsterCombatant{}}, SideA, MustHitPointStatus(10, FallsUnconscious, NoAffinities()), ConditionSet{})
	if err != nil {
		t.Fatal(err)
	}
	sp, _ := NewMovementSpeeds(30)
	profile := CombatantProfile{ID: "x", Size: content.SizeMedium, Speeds: sp}
	for name, e := range forged {
		if _, err := e.AddCombatant(x, profile, RosterAfterCombatant{Anchor: "a"}, JoinReasonArrived); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: AddCombatant err = %v, want ErrInvalidInput", name, err)
		}
		if _, err := e.RemoveCombatant("b", RemovalCauseFled); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: RemoveCombatant err = %v, want ErrInvalidInput", name, err)
		}
		if _, err := e.RepositionCombatant("b", RosterAfterCombatant{Anchor: "a"}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: RepositionCombatant err = %v, want ErrInvalidInput", name, err)
		}
		if _, err := (RemoveCombatantFromEncounter{}).Run(t.Context(), RemoveCombatantFromEncounterCommand{encounter: e, combatant: "b", cause: RemovalCauseFled}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: Run err = %v, want ErrInvalidInput before any work", name, err)
		}
	}
	forgedJoiners := map[string]Combatant{"no kind": {ref: CombatantRef{ID: "x"}, side: SideA, hp: x.hp}, "no side": {ref: x.ref, hp: x.hp}, "no hit points": {ref: x.ref, side: SideA}}
	for name, c := range forgedJoiners {
		if _, err := good.AddCombatant(c, profile, RosterAfterCombatant{Anchor: "a"}, JoinReasonArrived); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("forged joiner (%s): err = %v, want ErrInvalidInput", name, err)
		}
	}

	rec := telemetrytest.NewRecorder()
	u := rosterUse{newInstrumenter(rec.Telemetry(), nil)}
	op := combatOperation{valueOperationAddCombatantToEncounter, spanAddCombatantToEncounter}
	count := func(a rosterResultApplied) int { return len(a.events) }
	refused := func(rosterResultRejected) int { return -1 }
	_, err = runRoster(t.Context(), u, op, "arrived", good, func() (rosterTransition, error) {
		return rosterApplied{Encounter{}, []Event{CombatantRemoved{}}}, nil // an applied transition to an invalid encounter
	}, count, refused)
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("exit check: err = %v, want ErrInvalidInput [plan P14]", err)
	}
	spans := rec.Tracer.SpansNamed(spanAddCombatantToEncounter)
	if len(spans) != 1 || len(spans[0].Errors) != 1 || !spans[0].Ended {
		t.Errorf("spans = %+v, want one ended span with the error recorded once", spans)
	}
	if n := len(rec.Meter.MeasurementsNamed(metricOperations)) + len(rec.Meter.MeasurementsNamed(metricEvents)); n != 0 {
		t.Errorf("%d operation or event samples recorded for an invalid exit, want none", n)
	}
	called := false
	_, err = runRoster(t.Context(), u, op, "arrived", Encounter{}, func() (rosterTransition, error) { called = true; return nil, errors.New("not reached") }, count, refused)
	if !errors.Is(err, srderr.ErrInvalidInput) || called {
		t.Errorf("zero encounter: err = %v, work called = %v, want ErrInvalidInput before the work", err, called)
	}
}
