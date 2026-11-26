package combat_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/monster-stat-block-actions.md FR-048 - StatBlockPartUsed, MultiattackAttackRecorded, MultiattackAdditionalUseRecorded, MultiattackAttackReplaced,
// SaveImmunityGranted and SaveImmunityExpired are the members of one closed StatBlockActionEvent family with a constant unique kind and combatants [source] for the first
// four and [target, source] for the two immunity events, no duplicates; an exhaustive type switch covers the family.
// Path: happy.
func TestStatBlockActionEventFamily(t *testing.T) {
	t.Parallel()
	part := luRef(t, combat.MonsterPartActions, 1)
	// Each event comes from the operation that produces it.
	beast := sbBeast(t)
	used := sbAllowed(t, sbUse(t, beast, sbRequest(combat.StatBlockCostAction, sbAction(t, beast, "Slam")), luStart(t, beast, combat.OutsideLair))).Events()[0]
	m := maGladiator(t, maPlainBash())
	state := luStart(t, m, combat.OutsideLair)
	start, _ := combat.StartMultiattack(m, sbSelf(), state)
	p := start.(combat.MultiattackStarted).Progress()
	attack, _ := p.RecordAttack(sbAction(t, m, "Spear"), state)
	replace, _ := p.RecordReplacement(sbOption("Shield Bash"), sbAction(t, m, "Spear"), state)
	a := maAboleth(t)
	astate := luStart(t, a, combat.OutsideLair)
	astart, _ := combat.StartMultiattack(a, sbSelf(), astate)
	extraStep, _ := astart.(combat.MultiattackStarted).Progress().RecordAdditionalUse(sbAction(t, a, "Consume Memories"), astate)
	last := func(o combat.MultiattackStepOutcome) combat.Event {
		events := o.(combat.MultiattackStepRecorded).Transition.Events()
		return events[len(events)-1]
	}
	granted := combat.SaveImmunities{}.Grant(sbWindow(t, "hero", "ghost", part, time.Hour))
	expired := sbElapsed(t, granted.Immunities(), time.Hour)
	events := []combat.Event{used, last(attack), last(extraStep), last(replace), granted.Events()[0], expired.Events()[0]}
	wantKinds := []string{"stat_block_part_used", "multiattack_attack_recorded", "multiattack_additional_use_recorded", "multiattack_attack_replaced", "save_immunity_granted", "save_immunity_expired"}
	combattest.CheckEventKinds(t, events)
	for i, e := range events {
		wantWho := []combat.CombatantID{sbSelf()}
		if i >= 4 {
			wantWho = []combat.CombatantID{"hero", "ghost"}
		}
		combattest.CheckEvent(t, e, wantKinds[i], wantWho...)
		var family combat.StatBlockActionEvent
		switch v := e.(type) {
		case combat.StatBlockPartUsed:
			family = v
			if v.Part() != sbAction(t, beast, "Slam") || v.Cost() != combat.StatBlockCostAction {
				t.Errorf("StatBlockPartUsed = %+v", v)
			}
		case combat.MultiattackAttackRecorded:
			family = v
		case combat.MultiattackAdditionalUseRecorded:
			family = v
		case combat.MultiattackAttackReplaced:
			family = v
		case combat.SaveImmunityGranted:
			family = v
		case combat.SaveImmunityExpired:
			family = v
		}
		if family == nil || family.Kind() != wantKinds[i] {
			t.Errorf("event %d (%T) is not in the StatBlockActionEvent family with kind %q", i, e, wantKinds[i])
		}
	}
	if len(slices.Compact(slices.Sorted(slices.Values(wantKinds)))) != 6 {
		t.Error("the six kinds are not unique")
	}
}
