package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/monster-spellcasting.md FR-034 - the five events are the members of one closed MonsterSpellcastingEvent family with constant unique kinds and combatants equal to [combatant], each carrying the spell or Concentration subject; an exhaustive type switch covers them.
// Path: happy.
func TestMonsterSpellcastingEventFamily(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	rig := mscNewRig(t, mscMonster(t, nil))
	rig.state = mscConcentrating(t, "other_spell")
	started := mscStarted(t, rig.cast(mscList(t, mscGeas), spells["geas"])).Transition
	rig = rig.then(t, mscStarted(t, rig.cast(mscList(t, mscGeas), spells["geas"])))
	failed := combat.MonsterSpellcastingRules{}.EndMagicActionTurn(rig.state, rig.uses, false).Events()
	cast := mscPerformed(t, rig.complete(mscList(t, mscGeas), spells["geas"])).Transition.Events()
	cantrip := mscPerformed(t, mscNewRig(t, mscMonster(t, nil)).cast(mscList(t, mscLight), spells["light"])).Transition.Events()[0]

	events := []combat.Event{started.Events()[0], started.Events()[1], started.Events()[2], failed[1], cantrip}
	wantKinds := []string{"monster_concentration_ended", "monster_concentration_began", "monster_casting_started", "monster_casting_failed", "monster_spell_cast"}
	combattest.CheckEventKinds(t, events)
	for i, e := range events {
		combattest.CheckEvent(t, e, wantKinds[i], mscID())
		if _, ok := e.(combat.MonsterSpellcastingEvent); !ok {
			t.Errorf("event %d (%T) is not in the family", i, e)
		}
		switch v := e.(type) {
		case combat.MonsterConcentrationEnded:
			if v.Subject() != character.NewConcentrationSubjectSpell("other_spell") || v.Reason() != character.EndedAnotherEffect {
				t.Errorf("MonsterConcentrationEnded = %v %v", v.Subject(), v.Reason())
			}
		case combat.MonsterConcentrationBegan:
			if v.Subject() != character.NewConcentrationSubjectSpell("geas") || v.ConcentrationKind() != character.ConcentrationCasting {
				t.Errorf("MonsterConcentrationBegan = %v %v", v.Subject(), v.ConcentrationKind())
			}
		case combat.MonsterCastingStarted:
			if v.Spell() != "geas" || v.TotalMinutes() != 1 {
				t.Errorf("MonsterCastingStarted = %v %d", v.Spell(), v.TotalMinutes())
			}
		case combat.MonsterCastingFailed:
			if v.Spell() != "geas" || v.Reason() != character.EndedByCaster {
				t.Errorf("MonsterCastingFailed = %v %v", v.Spell(), v.Reason())
			}
		case combat.MonsterSpellCast:
			if at, has := v.CastAtLevel(); v.Spell() != "light" || has || at != 0 || v.ScalingLevel() != 1 {
				t.Errorf("MonsterSpellCast = %v %d %v scaling %d, want the cantrip", v.Spell(), at, has, v.ScalingLevel())
			}
		default:
			t.Errorf("event %d (%T) is not one of the five", i, e)
		}
	}
	if at, has := cast[len(cast)-1].(combat.MonsterSpellCast).CastAtLevel(); at != 5 || !has {
		t.Errorf("MonsterSpellCast.CastAtLevel() = %d, %v, want 5", at, has)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-033 - the events and limited-use events of a transition are copies: changing a returned slice changes nothing, and each call returns the same events.
// Path: unhappy (immutability).
func TestTransitionEventsAreCopies(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	rig := mscNewRig(t, mscMonster(t, nil))
	rig.state = mscConcentrating(t, "other_spell")
	tr := mscPerformed(t, rig.cast(mscList(t, mscFireball), spells["fireball"])).Transition
	first := tr.Events()
	limited := tr.LimitedUseEvents()
	if len(first) != 1 || len(limited) != 1 {
		t.Fatalf("events = %d, limited-use events = %d, want 1 and 1", len(first), len(limited))
	}
	first[0], limited[0] = nil, nil
	if tr.Events()[0] == nil || tr.LimitedUseEvents()[0] == nil {
		t.Errorf("changing a returned slice changed the transition [FR-033]")
	}
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{tr.Events()[0]} })
}
