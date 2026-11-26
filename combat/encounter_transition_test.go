package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// namedCommands are the commands that name a combatant, each for the id given.
func namedCommands(e combat.Encounter, id combat.CombatantID) map[string]func() (encTr, error) {
	return map[string]func() (encTr, error){
		"ApplyDamage": func() (encTr, error) {
			return e.ApplyDamage(id, encSlash(1), nil, combat.OtherSource, combat.NormalHit)
		},
		"ApplyHealing":           func() (encTr, error) { return e.ApplyHealing(id, 1) },
		"ApplyCondition":         func() (encTr, error) { return e.ApplyCondition(id, combat.Blinded{}) },
		"RemoveCondition":        func() (encTr, error) { return e.RemoveCondition(id, content.ConditionBlinded) },
		"ReduceExhaustion":       func() (encTr, error) { return e.ReduceExhaustion(id, 1) },
		"ApplyDeathSavingThrow":  func() (encTr, error) { return e.ApplyDeathSavingThrow(id, 10, 10) },
		"Stabilize":              func() (encTr, error) { return e.Stabilize(id) },
		"RecoverStable":          func() (encTr, error) { return e.RecoverStable(id) },
		"GainTemporaryHitPoints": func() (encTr, error) { return e.GainTemporaryHitPoints(id, 1, combat.AcceptNew) },
		"ReduceHitPointMaximum":  func() (encTr, error) { return e.ReduceHitPointMaximum(id, 1) },
		"CheckGrapple":           func() (encTr, error) { return e.CheckGrapple(id, 1, 1) },
		"ReleaseGrapple":         func() (encTr, error) { return e.ReleaseGrapple(id) },
		"EscapeGrapple":          func() (encTr, error) { return e.EscapeGrapple(id, d20.Success) },
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-046, FR-048, FR-079 - every command naming a combatant returns UnknownCombatant carrying the id for an id the encounter does not hold (even while another combatant is dead), and CombatantIsDead for a dead one, with the unchanged encounter and no events.
// Path: unhappy (invalid input, state).
func TestEveryNamedCommandRefusesUnknownAndDeadCombatants(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, deadStatus(t, combat.MonsterDeath)))
	for name, f := range namedCommands(e, cid("zzz")) {
		tr, err := f()
		if r := encNo(t, e, tr, err); r != (combat.UnknownCombatantRejection{ID: cid("zzz")}) {
			t.Errorf("%s with an unknown id: %#v [FR-046]", name, r)
		}
	}
	for name, f := range namedCommands(e, cid("b")) {
		tr, err := f()
		if r := encNo(t, e, tr, err); r != (combat.CombatantIsDeadRejection{ID: cid("b")}) {
			t.Errorf("%s for a dead combatant: %#v [FR-079]", name, r)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-047, FR-049, FR-054 - a command changes only the combatant it names, leaves the receiver unchanged, and gives equal transitions on every call.
// Path: unhappy (immutability, determinism).
func TestCommandsChangeOnlyTheirCombatantAndAreDeterministic(t *testing.T) {
	t.Parallel()
	mk := func() combat.Encounter {
		e, _ := encStart(t, encC(t, "a", combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}, combat.Grappled{Grappler: cid("c")}), encPC(t, "b", combat.SideB, 20, 20), encPC(t, "c", combat.SideA, 20, 20))
		return e
	}
	e, snapshot := mk(), mk()
	for name, f := range namedCommands(e, cid("a")) {
		tr, err := f()
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !e.Equal(snapshot) {
			t.Errorf("%s changed the receiver [FR-049]", name)
		}
		encOthersEqual(t, e, tr.Encounter(), "a")
		again, err2 := f()
		if err2 != nil || !deepEqual(tr.Events(), again.Events()) || !tr.Encounter().Equal(again.Encounter()) || !deepEqual(tr.Rejection(), again.Rejection()) {
			t.Errorf("%s: two calls gave different transitions [FR-054]", name)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-053 - encounters differing in the id, the status, the tracker or any combatant are not equal.
// Path: happy.
func TestEncounterEquality(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	if !e.Equal(encABC(t)) {
		t.Fatalf("two equal encounters differ [FR-053]")
	}
	other, _, err := (combat.Encounter{}).Start(combat.MustEncounterID("other"), encEstablished(t, "a", "b", "c"), e.Combatants())
	if err != nil {
		t.Fatal(err)
	}
	tr, err := e.ApplyHealing(cid("a"), 0)
	same, _ := encOK(t, tr, err)
	tr, err = e.ApplyDamage(cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
	damaged, _ := encOK(t, tr, err)
	adv, _ := advance(t, e)
	for name, o := range map[string]combat.Encounter{"id": other, "combatant": damaged, "tracker": adv, "status": endedEncounter(t, e)} {
		if e.Equal(o) || o.Equal(e) {
			t.Errorf("an encounter differing in the %s is equal [FR-053]", name)
		}
	}
	if !e.Equal(same) {
		t.Errorf("healing nothing changed the encounter")
	}
	if (combat.Encounter{}).IsZero() != true || e.IsZero() {
		t.Errorf("IsZero is wrong")
	}
}

// Requirement: doc/specs/encounter-aggregate.md table 4.9.1, FR-078 - when several rejections apply the first in the order wins: ended over unknown and dead, unknown over dead, dead over the command's own reason (an immune condition, a save for a non-Dying creature, a grapple).
// Path: unhappy (rule interaction).
func TestRejectionPrecedence(t *testing.T) {
	t.Parallel()
	dead := encWith(t, encC(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath)))
	ended := endedEncounter(t, dead)
	cases := []struct {
		name string
		e    combat.Encounter
		run  func(e combat.Encounter) (encTr, error)
		want combat.EncounterRejection
	}{
		{"ended and unknown", ended, func(e combat.Encounter) (encTr, error) {
			return e.ApplyDamage(cid("zzz"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
		}, combat.EncounterAlreadyEndedRejection{}},
		{"ended and dead", ended, func(e combat.Encounter) (encTr, error) {
			return e.ApplyDamage(cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
		}, combat.EncounterAlreadyEndedRejection{}},
		{"unknown id, another combatant dead", dead, func(e combat.Encounter) (encTr, error) { return e.ApplyHealing(cid("zzz"), 1) }, combat.UnknownCombatantRejection{ID: cid("zzz")}},
		{"dead and not dying", dead, func(e combat.Encounter) (encTr, error) { return e.ApplyDeathSavingThrow(cid("a"), 10, 10) }, combat.CombatantIsDeadRejection{ID: cid("a")}},
		{"remove on dead", dead, func(e combat.Encounter) (encTr, error) { return e.RemoveCondition(cid("a"), content.ConditionBlinded) }, combat.CombatantIsDeadRejection{ID: cid("a")}},
		{"end or advance on ended", ended, func(e combat.Encounter) (encTr, error) { return e.AdvanceTurn(combat.TurnAdvance{}) }, combat.EncounterAlreadyEndedRejection{}},
	}
	for _, c := range cases {
		tr, err := c.run(c.e)
		if r := encNo(t, c.e, tr, err); r != c.want {
			t.Errorf("%s: rejection = %#v, want %#v [FR-078]", c.name, r, c.want)
		}
	}
	// A dead creature does not stop AdvanceTurn, ApplyTurnChange or end (FR-079).
	if _, events := advance(t, dead); len(events) == 0 {
		t.Errorf("AdvanceTurn must be unaffected by a dead combatant [FR-079]")
	}
	if _, ok := runEnd(t, dead, combat.EndReasonMutualAgreement{}).(combat.EndEncounterApplied); !ok {
		t.Errorf("end must be unaffected by a dead combatant [FR-079]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-015, FR-052 - every command type and many advances keep the combatants in the tracker's order and in the same number.
// Path: unhappy (rule interaction).
func TestNoCommandChangesTheOrderOrSizeOfTheCombatants(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	want := ids(e.Combatants())
	for name, f := range namedCommands(e, cid("b")) {
		tr, err := f()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := ids(tr.Encounter().Combatants()); !slices.Equal(got, want) {
			t.Errorf("%s: combatants = %v, want %v [FR-015]", name, got, want)
		}
	}
	for range 40 {
		e, _ = advance(t, e)
	}
	if got := ids(e.Combatants()); !slices.Equal(got, want) {
		t.Errorf("combatants after 40 advances = %v, want %v [FR-015]", got, want)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-084 - the wrapper constructors return an ErrInvalidInput error when the wrapped event is not about the combatant (for GrappleEnded, not led by the grappled one), carry the encounter id and the round, and report the wrapped kind and combatants.
// Path: unhappy (invalid input).
func TestWrapperConstructors(t *testing.T) {
	t.Parallel()
	died, _ := combat.NewDied(cid("a"), combat.MassiveDamage)
	applied, _ := combat.NewConditionApplied(cid("a"), combat.Blinded{})
	ended, _ := combat.NewGrappleEnded(cid("b"), cid("a"), combat.GrappleEndCauseReleased)
	if w, err := combat.NewHitPointEventOccurred(eid(), 2, cid("a"), died); err != nil || w.Kind() != "died" || w.Round() != 2 || w.EncounterID() != eid() || w.Event() != died {
		t.Errorf("hit point wrapper = %#v, %v", w, err)
	}
	if _, err := combat.NewHitPointEventOccurred(eid(), 2, cid("b"), died); err == nil {
		t.Errorf("a hit point wrapper for another combatant must be an error")
	}
	if w, err := combat.NewConditionEventOccurred(eid(), 1, cid("a"), applied); err != nil || w.Kind() != "condition_applied" || w.Event() != applied {
		t.Errorf("condition wrapper = %#v, %v", w, err)
	}
	if _, err := combat.NewConditionEventOccurred(eid(), 1, cid("b"), applied); err == nil {
		t.Errorf("a condition wrapper for another combatant must be an error")
	}
	if w, err := combat.NewGrappleEventOccurred(eid(), 1, cid("b"), ended); err != nil || !slices.Equal(w.Combatants(), []combat.CombatantID{"b", "a"}) {
		t.Errorf("grapple wrapper = %#v, %v", w, err)
	}
	if _, err := combat.NewGrappleEventOccurred(eid(), 1, cid("a"), ended); err == nil {
		t.Errorf("a grapple wrapper under the grappler must be an error")
	}
	step := trkTracker(t, trkProfile(t, cid("a"), content.SizeMedium, trkSpeeds(t, 30)))
	st, err := step.InteractWithObject(cid("a"))
	_, tev := trkApplied(t)(st, err)
	if w, err := combat.NewTurnEventOccurred(eid(), 1, tev[0]); err != nil || w.Kind() != "object_interacted" || w.Event().Kind() != "object_interacted" {
		t.Errorf("turn wrapper = %#v, %v", w, err)
	}
	for name, f := range map[string]func() error{
		"blank id":    func() error { _, err := combat.NewTurnEventOccurred("", 1, tev[0]); return err },
		"round 0":     func() error { _, err := combat.NewTurnEventOccurred(eid(), 0, tev[0]); return err },
		"nil turn":    func() error { _, err := combat.NewTurnEventOccurred(eid(), 1, nil); return err },
		"nil hp":      func() error { _, err := combat.NewHitPointEventOccurred(eid(), 1, cid("a"), nil); return err },
		"nil cond":    func() error { _, err := combat.NewConditionEventOccurred(eid(), 1, cid("a"), nil); return err },
		"nil grapple": func() error { _, err := combat.NewGrappleEventOccurred(eid(), 1, cid("a"), nil); return err },
	} {
		if f() == nil {
			t.Errorf("%s: want an ErrInvalidInput error", name)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-078, FR-097 - with two unknown ids the smallest in code unit order is reported, whichever comes first; a wrapper under a grappled id greater than the event's first combatant is an error.
// Path: unhappy (rule interaction).
func TestSmallestUnknownIDWhateverTheOrderAndWrapperOrdering(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	step, err := e.Tracker().UpdateMovementSpeeds(cid("a"), trkSpeeds(t, 20))
	tk, _ := trkApplied(t)(step, err)
	tr, err := e.ApplyTurnChange(tk, []combat.TurnEvent{unknownSpeeds(t, "y"), unknownSpeeds(t, "z")})
	if r := encNo(t, e, tr, err); r != (combat.UnknownCombatantRejection{ID: cid("y")}) {
		t.Errorf("rejection = %#v, want y, the smaller of y and z", r)
	}
	ended, _ := combat.NewGrappleEnded(cid("b"), cid("a"), combat.GrappleEndCauseReleased)
	if _, err := combat.NewGrappleEventOccurred(eid(), 1, cid("z"), ended); err == nil {
		t.Errorf("a grappled id greater than the event's first combatant must be an error [FR-084]")
	}
}
