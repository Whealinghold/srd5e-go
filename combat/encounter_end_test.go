package combat_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// runEnd ends e through the use case.
func runEnd(tb testing.TB, e combat.Encounter, r combat.EndReason) combat.EndEncounterResult {
	tb.Helper()
	cmd, err := combat.NewEndEncounterCommand(e, r)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := combat.NewEndEncounter().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return res
}

// Requirement: doc/specs/encounter-aggregate.md FR-042, FR-043 - ending with a reason sets the status and produces EncounterEnded carrying it with every combatant in turn order; SideDefeated is not verified against the state.
// Path: happy.
func TestEndEncounterWithBothReasons(t *testing.T) {
	t.Parallel()
	for _, r := range []combat.EndReason{combat.EndReasonSideDefeated{Side: combat.SideA}, combat.EndReasonMutualAgreement{}} {
		e := encABC(t)
		res := runEnd(t, e, r).(combat.EndEncounterApplied)
		if st, ok := res.Encounter().Status().(combat.EncounterEndedStatus); !ok || st.Reason != r {
			t.Errorf("status = %#v, want ended for %#v [FR-042]", res.Encounter().Status(), r)
		}
		events := res.Events()
		if len(events) != 1 {
			t.Fatalf("events = %v, want only encounter_ended", encKinds(events))
		}
		ended := events[0].(combat.EncounterEnded)
		combattest.CheckEvent(t, ended, "encounter_ended", cid("a"), cid("b"), cid("c"))
		if ended.Reason() != r || ended.Round() != 1 || ended.EncounterID() != eid() {
			t.Errorf("EncounterEnded = %#v", ended)
		}
		if !slices.EqualFunc(res.Encounter().Combatants(), e.Combatants(), combat.Combatant.Equal) {
			t.Errorf("ending changed a combatant")
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-044 - the encounter does not end by itself: with every combatant of a side dead or at 0 Hit Points the status stays EncounterInProgress and no EncounterEnded is produced.
// Path: unhappy (rule interaction).
func TestEncounterDoesNotEndByItself(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encPC(t, "a", combat.SideA, 6, 12), encC(t, "b", combat.SideB, deadStatus(t, combat.MonsterDeath)))
	tr, err := e.ApplyDamage(cid("a"), encSlash(6), nil, combat.OtherSource, combat.NormalHit)
	after, events := encOK(t, tr, err)
	if after.Status() != (combat.EncounterInProgress{}) || slices.Contains(encKinds(events), "encounter_ended") {
		t.Errorf("status %#v events %v: the encounter must stay in progress [FR-044]", after.Status(), encKinds(events))
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-045, table 4.9.1 - once ended, every command, end itself and the roster-free transitions return EncounterAlreadyEnded, even for an unknown or dead combatant.
// Path: unhappy (state).
func TestEveryCommandOnAnEndedEncounterIsRefused(t *testing.T) {
	t.Parallel()
	e := endedEncounter(t, grappleEnc(t, nil))
	other := trkTracker(t, trkProfile(t, cid("a"), content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, cid("b"), content.SizeMedium, trkSpeeds(t, 30)))
	cmds := map[string]func() (encTr, error){
		"AdvanceTurn":     func() (encTr, error) { return e.AdvanceTurn(combat.TurnAdvance{}) },
		"ApplyTurnChange": func() (encTr, error) { return e.ApplyTurnChange(other, nil) },
		"ApplyDamage": func() (encTr, error) {
			return e.ApplyDamage(cid("zzz"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
		},
		"ApplyHealing":           func() (encTr, error) { return e.ApplyHealing(cid("a"), 1) },
		"ApplyCondition":         func() (encTr, error) { return e.ApplyCondition(cid("a"), combat.Blinded{}) },
		"RemoveCondition":        func() (encTr, error) { return e.RemoveCondition(cid("a"), content.ConditionBlinded) },
		"ReduceExhaustion":       func() (encTr, error) { return e.ReduceExhaustion(cid("a"), 1) },
		"ApplyDeathSavingThrow":  func() (encTr, error) { return e.ApplyDeathSavingThrow(cid("a"), 10, 10) },
		"Stabilize":              func() (encTr, error) { return e.Stabilize(cid("a")) },
		"RecoverStable":          func() (encTr, error) { return e.RecoverStable(cid("a")) },
		"GainTemporaryHitPoints": func() (encTr, error) { return e.GainTemporaryHitPoints(cid("a"), 1, combat.AcceptNew) },
		"ReduceHitPointMaximum":  func() (encTr, error) { return e.ReduceHitPointMaximum(cid("a"), 1) },
		"CheckGrapple":           func() (encTr, error) { return e.CheckGrapple(cid("b"), 1, 1) },
		"ReleaseGrapple":         func() (encTr, error) { return e.ReleaseGrapple(cid("b")) },
		"EscapeGrapple":          func() (encTr, error) { return e.EscapeGrapple(cid("b"), d20.Success) },
	}
	for name, f := range cmds {
		tr, err := f()
		if r := encNo(t, e, tr, err); r != (combat.EncounterAlreadyEndedRejection{}) || r.String() != "encounter_already_ended" {
			t.Errorf("%s: rejection = %#v, want EncounterAlreadyEnded [FR-045]", name, r)
		}
	}
	if _, ok := runEnd(t, e, combat.EndReasonMutualAgreement{}).(combat.EndEncounterRejected); !ok {
		t.Errorf("end on an ended encounter must be rejected [FR-045]")
	}
	if got := e.PrecheckDeathSavingThrow(cid("a")); got != (combat.EncounterAlreadyEndedRejection{}) {
		t.Errorf("precheck on an ended encounter = %#v", got)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-042 - NewEndEncounterCommand rejects a nil reason and a SideDefeated of a side that is not valid, and a zero encounter.
// Path: unhappy (invalid input).
func TestEndEncounterCommandValidation(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	for name, f := range map[string]func() error{
		"nil":      func() error { _, err := combat.NewEndEncounterCommand(e, nil); return err },
		"bad side": func() error { _, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{}); return err },
		"zero": func() error {
			_, err := combat.NewEndEncounterCommand(combat.Encounter{}, combat.EndReasonMutualAgreement{})
			return err
		},
	} {
		if f() == nil {
			t.Errorf("%s: want an ErrInvalidInput error", name)
		}
	}
	if combat.SideA.String() != "side_a" || combat.SideB.String() != "side_b" || combat.Side(0).String() != "unknown" || combat.Side(0).Valid() {
		t.Errorf("Side names or validity are wrong")
	}
}
