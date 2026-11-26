package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// grpAApplied returns the tracker and events of an applied tracker step, failing the test otherwise.
func grpAApplied(t *testing.T) func(combat.TrackerStep, error) (combat.TurnTracker, []combat.TurnEvent) {
	t.Helper()
	return func(s combat.TrackerStep, err error) (combat.TurnTracker, []combat.TurnEvent) {
		t.Helper()
		if err != nil {
			t.Fatalf("tracker operation: %v", err)
		}
		a, ok := s.(combat.TrackerStepApplied)
		if !ok {
			t.Fatalf("step = %#v, want TrackerStepApplied", s)
		}
		return a.Tracker(), a.Events()
	}
}

// grpAReadied has the active combatant ready the held spell against trigger, then ends its turn, and returns the tracker
// on the next combatant's turn together with the caster's readied-action ReactionTaken fired by trigger.
func grpAReadied(t *testing.T, tr combat.TurnTracker, caster combat.CombatantID, trigger combat.ReadyTriggerID, held combat.HeldSpellInput) (combat.TurnTracker, combat.ReactionTaken) {
	t.Helper()
	av := combat.ConditionActionAvailabilityAvailable
	action, err := combat.NewReadiedAction(trigger, combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, held)
	if err != nil {
		t.Fatal(err)
	}
	tr, _ = grpAApplied(t)(tr.TakeAction(caster, av, combat.ReadyChoice{Action: action}))
	tr, _ = grpAApplied(t)(tr.Advance(combat.TurnAdvance{}))
	tr, events := grpAApplied(t)(tr.TakeReaction(caster, av, combat.ReadiedActionReaction{Trigger: trigger}))
	return tr, events[0].(combat.ReactionTaken)
}

// Requirement: doc/specs/readying-a-spell.md FR-002, FR-004, FR-005, FR-007 - a wizard readies a held Fireball on its turn (one spell_held, a Concentration hold), ends the turn, and at the trigger spends its Reaction: releasing the held spell with that Reaction asks the host to end the hold and produces one held_spell_released.
// Path: happy.
func TestReadyingASpellReadyThenReleaseScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	e := grpAEncounter(t, "wizard", "ally", "ogre", "troll")
	wizard, door := id("wizard"), combat.MustReadyTriggerID("door")

	ready, err := combat.NewReadyCombatantSpell().Run(ctx, combat.ReadyCombatantSpellCommand{Caster: wizard, Trigger: door, Spell: "fireball", CastingTime: content.NewCastingTimeMagicAction()})
	if err != nil {
		t.Fatal(err)
	}
	held, ok := ready.(combat.ReadySpellApplied)
	if !ok || held.Hold != (combat.ConcentrationHold{Spell: "fireball"}) || len(held.Events) != 1 || held.Events[0].Kind() != "spell_held" {
		t.Fatalf("ready = %#v, want applied with a Concentration hold on fireball and exactly spell_held", ready)
	}
	_, reaction := grpAReadied(t, e.Tracker(), wizard, door, combat.HoldsSpell{Spell: held.Spell})

	res, err := combat.NewReleaseCombatantSpell().Run(ctx, combat.ReleaseCombatantSpellCommand{Held: combat.HoldsSpell{Spell: held.Spell}, Reaction: reaction})
	if err != nil {
		t.Fatal(err)
	}
	released, ok := res.(combat.ReleaseHeldSpellApplied)
	if !ok || released.Release.Spell != "fireball" || len(released.Events) != 1 || released.Events[0].Kind() != "held_spell_released" || released.Events[0].Combatants()[0] != wizard {
		t.Errorf("release = %#v, want applied: end the fireball hold, one held_spell_released for the wizard", res)
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-003 - only a spell cast with a Magic action can be readied: a Bonus Action, a Reaction, a 1-minute, a 1-hour and an unset casting time are each refused with the casting time given and no event.
// Path: unhappy (invalid input).
func TestReadyingASpellRefusesOtherCastingTimesScenario(t *testing.T) {
	t.Parallel()
	reaction, err := content.NewCastingTimeReaction("when hit")
	if err != nil {
		t.Fatal(err)
	}
	minute, err := content.NewCastingTimeMinutes(1)
	if err != nil {
		t.Fatal(err)
	}
	hour, err := content.NewCastingTimeHours(1)
	if err != nil {
		t.Fatal(err)
	}
	for name, ct := range map[string]content.CastingTime{
		"bonus action": content.NewCastingTimeBonusAction(), "reaction": reaction, "one minute": minute, "one hour": hour, "unset": nil,
	} {
		res, err := combat.NewReadyCombatantSpell().Run(context.Background(), combat.ReadyCombatantSpellCommand{
			Caster: combat.MustCombatantID("wizard"), Trigger: combat.MustReadyTriggerID("door"), Spell: "shield", CastingTime: ct})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if r, ok := res.(combat.ReadySpellNotAnAction); !ok || r.CastingTime != ct {
			t.Errorf("%s: result = %#v, want ReadySpellNotAnAction carrying the casting time", name, res)
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-007, FR-008, FR-009 - a release is checked in order: a Reaction spent on something else is not a readied reaction, an empty readied action holds nothing, and a Reaction of another caster or another trigger does not match the held spell; no refusal carries an event.
// Path: unhappy (state error).
func TestReadyingASpellReleaseRefusalsScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	e := grpAEncounter(t, "wizard", "ally", "ogre", "troll")
	wizard, door := id("wizard"), combat.MustReadyTriggerID("door")
	spell := combat.ReadiedSpell{Spell: "fireball", Caster: wizard, Trigger: door}
	_, readied := grpAReadied(t, e.Tracker(), wizard, door, combat.HoldsSpell{Spell: spell})

	// A Reaction spent on Shield is no readied action.
	tr := e.Tracker()
	tr, _ = grpAApplied(t)(tr.Advance(combat.TurnAdvance{}))
	_, shieldEvents := grpAApplied(t)(tr.TakeReaction(wizard, combat.ConditionActionAvailabilityAvailable, combat.SpellReaction{Spell: "shield"}))
	shield := shieldEvents[0].(combat.ReactionTaken)

	otherCaster := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: id("ally"), Trigger: door}}
	otherTrigger := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: wizard, Trigger: combat.MustReadyTriggerID("window")}}
	cases := map[string]struct {
		held     combat.HeldSpellInput
		reaction combat.ReactionTaken
		want     combat.ReleaseCombatantSpellResult
	}{
		"a Reaction spent on Shield":               {combat.HoldsSpell{Spell: spell}, shield, combat.ReleaseHeldSpellNotAReadiedReaction{}},
		"Shield and no held spell: reaction first": {combat.HoldsNoSpell{}, shield, combat.ReleaseHeldSpellNotAReadiedReaction{}},
		"nothing is held":                          {combat.HoldsNoSpell{}, readied, combat.ReleaseHeldSpellNoHeldSpell{}},
		"an unset held input":                      {nil, readied, combat.ReleaseHeldSpellNoHeldSpell{}},
		"another caster's spell":                   {otherCaster, readied, combat.ReleaseHeldSpellReactionMismatch{}},
		"another trigger":                          {otherTrigger, readied, combat.ReleaseHeldSpellReactionMismatch{}},
	}
	for name, c := range cases {
		res, err := combat.NewReleaseCombatantSpell().Run(ctx, combat.ReleaseCombatantSpellCommand{Held: c.held, Reaction: c.reaction})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res != c.want {
			t.Errorf("%s: result = %#v, want %#v", name, res, c.want)
		}
	}
}

// Requirement: doc/specs/readying-a-spell.md FR-010, FR-011, FR-038 (initiative-and-actions.md) - a readied action holding a spell expires at the start of the caster's next turn without spending the Reaction; the host sees readied_action_expired and dissipates the spell with ReadyElapsed, and a broken Concentration dissipates it too; the cause is checked before whether a spell is held, and an empty readied action dissipates nothing.
// Path: unhappy (rule interaction).
func TestReadyingASpellDissipationScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	e := grpAEncounter(t, "wizard", "ally", "ogre", "troll")
	wizard, door := id("wizard"), combat.MustReadyTriggerID("door")
	held := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: wizard, Trigger: door}}

	action, err := combat.NewReadiedAction(door, combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, held)
	if err != nil {
		t.Fatal(err)
	}
	tr, _ := grpAApplied(t)(e.Tracker().TakeAction(wizard, combat.ConditionActionAvailabilityAvailable, combat.ReadyChoice{Action: action}))
	var events []combat.TurnEvent
	for range 4 { // a full round: back to the wizard's own turn
		tr, events = grpAApplied(t)(tr.Advance(combat.TurnAdvance{}))
	}
	var expired combat.ReadiedActionExpired
	for _, ev := range events {
		if x, ok := ev.(combat.ReadiedActionExpired); ok {
			expired = x
		}
	}
	if expired.Combatants() == nil || expired.Held() != combat.HeldSpellInput(held) || tr.ReactionSpent(wizard) {
		t.Fatalf("expiry = %+v, reaction spent %v, want the held spell reported and the Reaction unspent", expired, tr.ReactionSpent(wizard))
	}

	run := func(h combat.HeldSpellInput, cause combat.DissipationCause) combat.DissipateCombatantSpellResult {
		t.Helper()
		res, err := combat.NewDissipateCombatantSpell().Run(ctx, combat.DissipateCombatantSpellCommand{Held: h, Cause: cause})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	for _, cause := range []combat.DissipationCause{combat.DissipationReadyElapsed, combat.DissipationConcentrationBroken} {
		a, ok := run(expired.Held(), cause).(combat.DissipateHeldSpellApplied)
		if !ok || len(a.Events) != 1 || a.Events[0].Kind() != "held_spell_dissipated" || a.Events[0].(combat.HeldSpellDissipated).Cause() != cause {
			t.Errorf("cause %v: result = %#v, want one held_spell_dissipated carrying the cause", cause, a)
		}
	}
	if got := run(combat.HoldsNoSpell{}, combat.DissipationReadyElapsed); got != combat.DissipateCombatantSpellResult(combat.DissipateHeldSpellNoHeldSpell{}) {
		t.Errorf("an empty readied action = %#v, want NoHeldSpell", got)
	}
	for _, bad := range []combat.DissipationCause{0, combat.DissipationReadyElapsed + 1} {
		for name, h := range map[string]combat.HeldSpellInput{"holding": held, "empty": combat.HoldsNoSpell{}} {
			if got := run(h, bad); got != combat.DissipateCombatantSpellResult(combat.DissipateHeldSpellInvalidCause{}) {
				t.Errorf("cause %d (%s) = %#v, want InvalidCause checked first", bad, name, got)
			}
		}
	}
}
