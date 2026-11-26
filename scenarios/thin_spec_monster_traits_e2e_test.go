package scenarios_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-traits.md FR-041, FR-045, FR-046 (SRD 5.2.1, "Troll", Regeneration) - a Troll-like monster dropped to 0 by Slashing damage is kept alive and regains 15 Hit Points at the start of its turn; Fire damage then suppresses the trait for one turn (Skipped, suppressed, nothing regained) and the turn after that it regenerates again; through the public API only.
// Path: happy, then unhappy (rule interaction over two turns).
func TestRegeneratingMonsterHealsThenIsSuppressedForExactlyOneTurn(t *testing.T) {
	t.Parallel()
	id := combat.MustCombatantID("troll")
	trait := combat.RecogniseTraits(trollMonster(t)).Traits()[0].(combat.RegenerationTrait)
	state, err := combat.NewRegenerationState(trait)
	if err != nil {
		t.Fatal(err)
	}
	status := combat.MustHitPointStatus(84, combat.DiesInstantly, combat.NoAffinities())
	tr, err := status.ApplyDamage(id, []combat.Damage{combat.MustDamage(84, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	hit := tr.(combat.HitPointTransitionApplied)
	state = state.AfterDamage(id, []content.DamageType{content.DamageSlashing}).State()
	deferral, ok := combat.DeferRegenerationDeath(trait, id, hit).(combat.RegenerationDeferred)
	if !ok || state.Suppressed() {
		t.Fatalf("deferral = %#v suppressed %v, want deferred and not suppressed", deferral, state.Suppressed())
	}
	healed, ok := state.AtStartOfTurn(deferral.Status(), id).(combat.RegenerationHealed)
	if !ok || healed.Transition().Status().HitPoints().Current() != 15 {
		t.Fatalf("turn start = %#v, want 15 Hit Points regained", healed)
	}

	burn, err := healed.Transition().Status().ApplyDamage(id, []combat.Damage{combat.MustDamage(5, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	state = healed.State().AfterDamage(id, []content.DamageType{content.DamageFire}).State()
	if !state.Suppressed() {
		t.Fatal("Fire damage must suppress Regeneration")
	}
	burned := burn.(combat.HitPointTransitionApplied).Status()
	skipped, ok := state.AtStartOfTurn(burned, id).(combat.RegenerationSkipped)
	if !ok || skipped.Reason() != combat.RegenerationSkipSuppressed || skipped.State().Suppressed() {
		t.Fatalf("suppressed turn = %#v, want skipped as suppressed and the flag cleared", skipped)
	}
	if again, ok := skipped.State().AtStartOfTurn(burned, id).(combat.RegenerationHealed); !ok || again.Transition().Status().HitPoints().Current() != 25 {
		t.Errorf("the turn after = %#v, want 15 more Hit Points (10 to 25)", again)
	}
}
