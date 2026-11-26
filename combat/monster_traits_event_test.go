package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/monster-traits.md FR-090 - the five events are the members of one closed MonsterTraitEvent family with constant unique kinds and combatants [subject], produced by the operations; an exhaustive type switch covers them.
// Path: happy.
func TestMonsterTraitEventFamily(t *testing.T) {
	t.Parallel()
	state, lr := mtLimited(t)
	failure := combat.RolledSaveFailure{Result: mtSave(t, 5, 0, 15)}
	out := luMust(combat.UseLegendaryResistance(state, lr, mtID(), failure)).(combat.LegendaryResistanceApplied)
	used := out.Events()[1]

	zombie := mtZombie(t, 10, 10, 0, combat.DiesInstantly)
	fall := mtHit(t, zombie, combat.NormalHit, combat.MustDamage(10, content.DamageSlashing))
	check := combat.UndeadFortitudeSaveRequired{DC: 15, Ability: ability.Constitution}
	survived := luMust(combat.SettleUndeadFortitude(check, zombie, fall, mtID(), d20.Success)).(combat.UndeadFortitudeSurvived)
	saved := survived.Events()[len(survived.Events())-1]

	st := luMust(combat.NewRegenerationState(mtTroll(t)))
	suppressed := st.AfterDamage(mtID(), damageTypes(content.DamageFire)).Events()[0]
	troll := status(t, 10, 10, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
	drop := mtHit(t, troll, combat.NormalHit, combat.MustDamage(10, content.DamageSlashing))
	deferred := combat.DeferRegenerationDeath(mtTroll(t), mtID(), drop).(combat.RegenerationDeferred)
	deathDeferred := deferred.Events()[len(deferred.Events())-1]

	resp := luMust(combat.RespondToParry(luMust(combat.NewParryTrait(luRef(t, combat.MonsterPartReactions, 0), 2)), mtID(), mtAttack(t, 10, 5, 14)))
	events := []combat.Event{used, saved, suppressed, deathDeferred, resp.Event()}
	wantKinds := []string{"legendary_resistance_used", "undead_fortitude_saved", "regeneration_suppressed", "regeneration_death_deferred", "parry_used"}
	combattest.CheckEventKinds(t, events)
	for i, e := range events {
		combattest.CheckEvent(t, e, wantKinds[i], mtID())
		switch v := e.(type) {
		case combat.LegendaryResistanceUsed:
		case combat.UndeadFortitudeSaved:
			if v.DC() != 15 {
				t.Errorf("DC = %v", v.DC())
			}
		case combat.RegenerationSuppressed:
		case combat.RegenerationDeathDeferred:
		case combat.ParryUsed:
			if v.Bonus() != 2 || !v.BecameMiss() {
				t.Errorf("ParryUsed = %+v", v)
			}
		default:
			t.Errorf("event %d (%T) is not a MonsterTraitEvent", i, e)
		}
		if _, ok := e.(combat.MonsterTraitEvent); !ok {
			t.Errorf("event %d (%T) is not in the family", i, e)
		}
	}
}
