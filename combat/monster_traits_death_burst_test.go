package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

func mtBurstTrait(tb testing.TB) combat.DeathBurstTrait {
	tb.Helper()
	return mtOne[combat.DeathBurstTrait](tb, combat.RecogniseTraits(mtMonster(tb, []content.StatBlockPart{mtBurstPart(mtBurst, true)}, nil)))
}

// Requirement: doc/specs/monster-traits.md FR-050, FR-051, FR-052 - a Died for the subject triggers the burst with the part's save; Undead Fortitude survival, a deferred death, other creatures' deaths, nil events and zero inputs do not; a later LetDie does.
// Path: per case.
func TestTriggerDeathBurst(t *testing.T) {
	t.Parallel()
	b := mtBurstTrait(t)
	died := func(who string, cause combat.DeathCause) combat.Event {
		return luMust(combat.NewDied(combat.MustCombatantID(who), cause))
	}
	zombie := mtZombie(t, 10, 10, 0, combat.DiesInstantly)
	hit := mtHit(t, zombie, combat.NormalHit, mtSlash(10))
	survived := luMust(combat.SettleUndeadFortitude(required(15), zombie, hit, mtID(), d20.Success)).(combat.UndeadFortitudeSurvived)
	deferred := combat.DeferRegenerationDeath(mtTroll(t), mtID(), hit).(combat.RegenerationDeferred)
	letDie := mtState(t, mtTroll(t)).AfterDamage(mtID(), damageTypes(content.DamageFire)).State().AtStartOfTurn(deferred.Status(), mtID()).(combat.RegenerationLetDie)
	cases := []struct {
		name   string
		events []combat.Event
		who    combat.CombatantID
		want   bool
	}{
		{"died", []combat.Event{died(luSelf, combat.MonsterDeath)}, mtID(), true},
		{"died of exhaustion", []combat.Event{died(luSelf, combat.Exhaustion)}, mtID(), true},
		{"after other events", []combat.Event{nil, died("other", combat.MonsterDeath), died(luSelf, combat.MassiveDamage)}, mtID(), true},
		{"Undead Fortitude survival", survived.Events(), mtID(), false},
		{"deferred death", deferred.Events(), mtID(), false},
		{"let die later", letDie.Events(), mtID(), true},
		{"another creature died", []combat.Event{died("other", combat.MonsterDeath)}, mtID(), false},
		{"another creature sorting first died", []combat.Event{died("aaa", combat.MonsterDeath)}, mtID(), false},
		{"nil events", nil, mtID(), false},
		{"zero subject", []combat.Event{died(luSelf, combat.MonsterDeath)}, "", false},
	}
	for _, c := range cases {
		got, triggered := combat.TriggerDeathBurst(b, c.who, c.events).(combat.DeathBurstTriggered)
		if triggered != c.want {
			t.Errorf("%s: triggered = %v", c.name, triggered)
		}
		if triggered && (got.Part() != b.Part() || got.Ability() != ability.Constitution || got.DC() != 11) {
			t.Errorf("%s: burst = %+v", c.name, got)
		}
	}
	if _, ok := combat.TriggerDeathBurst(combat.DeathBurstTrait{}, mtID(), []combat.Event{died(luSelf, 1)}).(combat.DeathBurstNotTriggered); !ok {
		t.Error("a zero trait triggered")
	}
}
