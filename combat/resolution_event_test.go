package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/combat-action-resolution.md FR-094, FR-134, FR-089 - the events of this spec are members of the ResolutionEvent family with the Kind the spec lists, in the encounter and round they happened in, naming the attacker before the target and the grappled before the grappler, and giving a copy of the creatures they name.
// Path: happy.
func TestResolutionEventsCarryTheirFacts(t *testing.T) {
	t.Parallel()
	g := combat.Grappled{Grappler: cid("g")}
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resPC(t, "g", combat.SideB), resPC(t, "t", combat.SideA, g))
	hit, _ := resAttack(t, resParams(t, e, "a", "g", resBegin()), 14, 4)
	strike, _ := resStrike(t, resStrikeParams(t, e, "a", "g", grappleOption(t, ability.Strength, strengthSave(t))), 5)
	res, err := resDeal(t, e, "g", []combat.Damage{resDamage(20, content.DamageSlashing)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	damage := resAs[combat.DealDamageToCombatantCompleted](t, res)
	want := map[string][]string{
		"attack_performed":                  {"a", "g"},
		"unarmed_strike_performed":          {"a", "g"},
		"movement_speeds_review_became_due": {"g"},
		"grapple_check_became_due":          {"t", "g"},
	}
	seen := map[string]bool{}
	for _, ev := range slices.Concat(hit.Events(), strike.Events(), damage.Events()) {
		re, ok := ev.(combat.ResolutionEvent)
		if !ok {
			continue
		}
		seen[ev.Kind()] = true
		ids := ev.Combatants()
		if wantIDs, ok := want[ev.Kind()]; !ok || !slices.Equal(ids, idsOf(wantIDs)) {
			t.Errorf("%s names %v, want %v", ev.Kind(), ids, want[ev.Kind()])
		}
		if re.EncounterID() != eid() || re.Round() != 1 {
			t.Errorf("%s in %v round %d, want %v round 1", ev.Kind(), re.EncounterID(), re.Round(), eid())
		}
		if len(ids) > 0 {
			ids[0] = cid("zz")
			if ev.Combatants()[0] == cid("zz") {
				t.Errorf("%s hands out its own slice [FR-004]", ev.Kind())
			}
		}
	}
	for k := range want {
		if !seen[k] {
			t.Errorf("no %s event was produced", k)
		}
	}
	// Outcomes: the payload carries d20's values.
	hitOutcome := hit.Events()[2].(combat.AttackPerformed).Outcome().(combat.AttackHitOutcome)
	if hitOutcome.Roll().Roll().Natural() != 14 || hitOutcome.HitKind() != combat.NormalHit {
		t.Errorf("hit outcome = %+v", hitOutcome)
	}
	save := strike.Events()[2].(combat.UnarmedStrikePerformed).Outcome().(combat.UnarmedSavePerformed).Save().(combat.TestRolled)
	if save.Result().Roll().Natural() != 5 || save.Outcome() != d20.Failure {
		t.Errorf("save outcome = %+v, want the failed 5", save)
	}
	forced := resAs[combat.PerformUnarmedStrikeCompleted](t, func() combat.PerformUnarmedStrikeResult {
		p := resStrikeParams(t, e, "a", "g", grappleOption(t, ability.Strength, d20.ChooseToFail{}))
		cmd, _ := combat.NewPerformUnarmedStrikeCommand(p)
		uc, _ := combat.NewPerformUnarmedStrike(newFixed())
		r, err := uc.Run(ctxBackground(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}())
	tf := forced.Events()[2].(combat.UnarmedStrikePerformed).Outcome().(combat.UnarmedSavePerformed).Save().(combat.TestForcedFailure)
	if tf.Result().Reason() != d20.ChosenToFail || tf.Outcome() != d20.Failure {
		t.Errorf("forced save = %+v, want a failure chosen without a die", tf)
	}
	miss, _ := resAttack(t, resParams(t, e, "a", "g", resBegin()), 3)
	if m := miss.Events()[2].(combat.AttackPerformed).Outcome().(combat.AttackMissOutcome); m.Roll().Roll().Natural() != 3 {
		t.Errorf("miss outcome = %+v", m)
	}
	damageOption := combat.UnarmedStrikeDamageOption{ArmorClass: 15, Circumstances: []modifier.RollCircumstance{}}
	d, _ := resStrike(t, resStrikeParams(t, e, "a", "g", damageOption), 14)
	if o := d.Events()[2].(combat.UnarmedStrikePerformed).Outcome().(combat.UnarmedDamageOptionPerformed).Outcome(); o.(combat.AttackHitOutcome).HitKind() != combat.NormalHit {
		t.Errorf("Damage option outcome = %+v", o)
	}
}

func idsOf(s []string) []combat.CombatantID {
	out := make([]combat.CombatantID, len(s))
	for i, v := range s {
		out[i] = cid(v)
	}
	return out
}
