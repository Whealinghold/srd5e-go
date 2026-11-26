package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
)

func hsMinutes(t *testing.T, n int) content.CastingTime {
	t.Helper()
	c, err := content.NewCastingTimeMinutes(n)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func hsHours(t *testing.T, n int) content.CastingTime {
	t.Helper()
	c, err := content.NewCastingTimeHours(n)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// hsReaction returns the ReactionTaken of a readied action that holds a spell, fired by trigger.
func hsReaction(t *testing.T, held combat.HoldsSpell) combat.ReactionTaken {
	t.Helper()
	action, err := combat.NewReadiedAction("door", combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, held)
	if err != nil {
		t.Fatal(err)
	}
	tr := trkTake(t, trkABC(t), trkA, combat.ReadyChoice{Action: action})
	adv, _ := trkAdvance(t, tr)
	_, ev := trkApplied(t)(adv.TakeReaction(trkA, trkAvail, combat.ReadiedActionReaction{Trigger: "door"}))
	return ev[0].(combat.ReactionTaken)
}

var hsHeld = combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkA, Trigger: "door"}}

// Requirement: doc/specs/readying-a-spell.md FR-002, FR-004, FR-005 - a Magic action spell is held: the result carries the ReadiedSpell, a ConcentrationHold and exactly one SpellHeld event, also for a spell with no Concentration of its own.
// Path: happy.
func TestReadySpellMagicActionIsHeld(t *testing.T) {
	t.Parallel()
	for _, spell := range []content.SpellID{"fireball", "fire_bolt"} {
		res, ok := combat.ReadySpell(trkA, "door", spell, content.NewCastingTimeMagicAction()).(combat.ReadySpellApplied)
		if !ok {
			t.Fatalf("%s: not applied", spell)
		}
		if res.Spell != (combat.ReadiedSpell{Spell: spell, Caster: trkA, Trigger: "door"}) || res.Hold != (combat.ConcentrationHold{Spell: spell}) {
			t.Errorf("%s: spell %+v hold %+v", spell, res.Spell, res.Hold)
		}
		if len(res.Events) != 1 {
			t.Fatalf("%s: %d events, want exactly [SpellHeld]", spell, len(res.Events))
		}
		combattest.CheckEvent(t, res.Events[0], "spell_held", trkA)
		if e, ok := res.Events[0].(combat.SpellHeld); !ok || e.Spell() != spell {
			t.Errorf("%s: event %+v", spell, res.Events[0])
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-003 - every other casting time (one test per kind, Minutes(1) and Hours(1) included, and a nil one) is refused with that casting time and no events.
// Path: unhappy (state error).
func TestReadySpellOtherCastingTimesRefused(t *testing.T) {
	t.Parallel()
	reaction, err := content.NewCastingTimeReaction("when hit")
	if err != nil {
		t.Fatal(err)
	}
	for name, ct := range map[string]content.CastingTime{
		"bonus action": content.NewCastingTimeBonusAction(), "reaction": reaction,
		"minutes": hsMinutes(t, 1), "hours": hsHours(t, 1), "nil": nil,
	} {
		got := combat.ReadySpell(trkA, "door", "fireball", ct)
		if got != combat.ReadyCombatantSpellResult(combat.ReadySpellNotAnAction{CastingTime: ct}) {
			t.Errorf("%s: %+v, want ReadySpellNotAnAction carrying the casting time", name, got)
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-001, FR-002 - the ReadiedSpell a ReadySpell returns is accepted by NewReadiedAction as HoldsSpell, so the two halves of a Ready fit.
// Path: happy.
func TestReadySpellResultFitsReadiedAction(t *testing.T) {
	t.Parallel()
	res := combat.ReadySpell(trkA, "door", "fireball", content.NewCastingTimeMagicAction()).(combat.ReadySpellApplied)
	if _, err := combat.NewReadiedAction("door", combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, combat.HoldsSpell{Spell: res.Spell}); err != nil {
		t.Errorf("NewReadiedAction rejected the readied spell: %v", err)
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-007, FR-009 - a matching readied-action Reaction releases the held spell with one HeldSpellReleased event and the release request.
// Path: happy.
func TestReleaseHeldSpellMatchingReaction(t *testing.T) {
	t.Parallel()
	res, ok := combat.ReleaseHeldSpell(hsHeld, hsReaction(t, hsHeld)).(combat.ReleaseHeldSpellApplied)
	if !ok || res.Release != (combat.ReleaseHoldRequest{Spell: "fireball"}) || len(res.Events) != 1 {
		t.Fatalf("result %+v, want applied with one event", res)
	}
	combattest.CheckEvent(t, res.Events[0], "held_spell_released", trkA)
	if res.Events[0].(combat.HeldSpellReleased).Spell() != "fireball" {
		t.Error("event names another spell")
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-007 (order), FR-008, FR-009 - the checks run in order: a non-readied Reaction first (even with HoldsNoSpell), then HoldsNoSpell, then caster or trigger mismatch; every refusal carries no events.
// Path: per case.
func TestReleaseHeldSpellRefusals(t *testing.T) {
	t.Parallel()
	adv, _ := trkAdvance(t, trkABC(t))
	_, ev := trkApplied(t)(adv.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
	spellReaction := ev[0].(combat.ReactionTaken)
	matching := hsReaction(t, hsHeld)
	otherCaster := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkB, Trigger: "door"}}
	otherTrigger := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkA, Trigger: "window"}}
	cases := []struct {
		name     string
		held     combat.HeldSpellInput
		reaction combat.ReactionTaken
		want     combat.ReleaseCombatantSpellResult
	}{
		{"reaction of another kind", hsHeld, spellReaction, combat.ReleaseHeldSpellNotAReadiedReaction{}},
		{"zero reaction", hsHeld, combat.ReactionTaken{}, combat.ReleaseHeldSpellNotAReadiedReaction{}},
		{"another kind and no spell: the reaction is checked first", combat.HoldsNoSpell{}, spellReaction, combat.ReleaseHeldSpellNotAReadiedReaction{}},
		{"no held spell", combat.HoldsNoSpell{}, matching, combat.ReleaseHeldSpellNoHeldSpell{}},
		{"nil held", nil, matching, combat.ReleaseHeldSpellNoHeldSpell{}},
		{"caster differs", otherCaster, matching, combat.ReleaseHeldSpellReactionMismatch{}},
		{"trigger differs", otherTrigger, matching, combat.ReleaseHeldSpellReactionMismatch{}},
		// A caster or trigger that sorts before the reaction's is still a mismatch (equality, not order).
		{"caster sorts before", combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: "", Trigger: "door"}}, matching, combat.ReleaseHeldSpellReactionMismatch{}},
		{"trigger sorts before", combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkA, Trigger: "aaa"}}, matching, combat.ReleaseHeldSpellReactionMismatch{}},
	}
	for _, c := range cases {
		if got := combat.ReleaseHeldSpell(c.held, c.reaction); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-010, FR-011, and the plan's named boundary (a held spell released or dissipated on the same turn) - both causes dissipate a held spell with one HeldSpellDissipated event and no refund; the same held value can be released and dissipated on the turn it was readied because the functions hold no state; the cause of an expiry comes from ReadiedActionExpired.
// Path: happy.
func TestDissipateHeldSpellEitherCause(t *testing.T) {
	t.Parallel()
	for _, cause := range []combat.DissipationCause{combat.DissipationConcentrationBroken, combat.DissipationReadyElapsed} {
		res, ok := combat.DissipateHeldSpell(hsHeld, cause).(combat.DissipateHeldSpellApplied)
		if !ok || len(res.Events) != 1 {
			t.Fatalf("%v: %+v, want applied with one event", cause, res)
		}
		combattest.CheckEvent(t, res.Events[0], "held_spell_dissipated", trkA)
		if e := res.Events[0].(combat.HeldSpellDissipated); e.Cause() != cause || e.Spell() != "fireball" {
			t.Errorf("%v: event %+v", cause, e)
		}
	}
	// Same turn: released, then (a second look at the same value) dissipated.
	if _, ok := combat.ReleaseHeldSpell(hsHeld, hsReaction(t, hsHeld)).(combat.ReleaseHeldSpellApplied); !ok {
		t.Error("release on the turn it was readied was refused")
	}
	// Expiry: the tracker's ReadiedActionExpired carries the held spell for the host.
	action, err := combat.NewReadiedAction("door", combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, hsHeld)
	if err != nil {
		t.Fatal(err)
	}
	tr := trkTake(t, trkABC(t), trkA, combat.ReadyChoice{Action: action})
	_, ev := trkAdvance(t, advN(t, tr, 2))
	expired := ev[len(ev)-1].(combat.ReadiedActionExpired)
	got := combat.DissipateHeldSpell(expired.Held(), combat.DissipationReadyElapsed)
	if _, ok := got.(combat.DissipateHeldSpellApplied); !ok {
		t.Errorf("dissipating the expired action's held spell: %+v", got)
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-011 - HoldsNoSpell (and a nil input) with either cause is refused with no events.
// Path: unhappy (state error).
func TestDissipateHeldSpellNoSpell(t *testing.T) {
	t.Parallel()
	for _, held := range []combat.HeldSpellInput{combat.HoldsNoSpell{}, nil} {
		for _, cause := range []combat.DissipationCause{combat.DissipationConcentrationBroken, combat.DissipationReadyElapsed} {
			if got := combat.DissipateHeldSpell(held, cause); got != combat.DissipateCombatantSpellResult(combat.DissipateHeldSpellNoHeldSpell{}) {
				t.Errorf("%v %v: %+v, want DissipateHeldSpellNoHeldSpell", held, cause, got)
			}
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-013 - the three events have the constant kinds spell_held, held_spell_released and held_spell_dissipated, unique together; DissipationCause names its values.
// Path: unhappy (name collision).
func TestHeldSpellEventFamily(t *testing.T) {
	t.Parallel()
	held := combat.ReadySpell(trkA, "door", "fireball", content.NewCastingTimeMagicAction()).(combat.ReadySpellApplied).Events[0]
	released := combat.ReleaseHeldSpell(hsHeld, hsReaction(t, hsHeld)).(combat.ReleaseHeldSpellApplied).Events[0]
	dissipated := combat.DissipateHeldSpell(hsHeld, combat.DissipationReadyElapsed).(combat.DissipateHeldSpellApplied).Events[0]
	combattest.CheckEventKinds(t, []combat.Event{held, released, dissipated})
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{held, released, dissipated} })
	for cause, want := range map[combat.DissipationCause]string{combat.DissipationConcentrationBroken: "concentration_broken", combat.DissipationReadyElapsed: "ready_elapsed", 0: "unknown"} {
		if cause.String() != want {
			t.Errorf("%d.String() = %q, want %q", int(cause), cause.String(), want)
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-011 - a cause that is not one of the two is refused with no events, whatever the held spell, and Valid names the two causes.
// Path: unhappy (invalid input).
func TestDissipateHeldSpellInvalidCause(t *testing.T) {
	t.Parallel()
	for _, cause := range []combat.DissipationCause{0, -1, 3} {
		for _, held := range []combat.HeldSpellInput{hsHeld, combat.HoldsNoSpell{}, nil} {
			if got := combat.DissipateHeldSpell(held, cause); got != combat.DissipateCombatantSpellResult(combat.DissipateHeldSpellInvalidCause{}) {
				t.Errorf("cause %d held %v: %+v, want DissipateHeldSpellInvalidCause", int(cause), held, got)
			}
		}
		if cause.Valid() {
			t.Errorf("DissipationCause(%d).Valid() = true", int(cause))
		}
	}
	for _, cause := range []combat.DissipationCause{combat.DissipationConcentrationBroken, combat.DissipationReadyElapsed} {
		if !cause.Valid() {
			t.Errorf("%v.Valid() = false", cause)
		}
	}
}
