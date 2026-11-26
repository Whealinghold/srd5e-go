package scenarios_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

func trollMonster(t *testing.T) content.Monster {
	t.Helper()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	scores, saves := map[ability.Ability]ability.Score{}, map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	init, err := content.NewStatBlockInitiative(1, 11)
	must(err)
	hp, err := content.NewHitPointsEntry(84, dice.MustExpression(8, dice.D10(), 40))
	must(err)
	speed, err := content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, "")
	must(err)
	sc, err := ability.NewScores(scores)
	must(err)
	block, err := content.NewAbilityBlock(sc, saves)
	must(err)
	regen, err := content.NewStatBlockPart("Regeneration", content.StatBlockPartFields{Usage: content.PartUnlimited{},
		Text: "The troll regains 15 Hit Points at the start of each of its turns. If the troll takes Acid or Fire damage, this trait doesn't function on the troll's next turn. The troll dies only if it starts its turn with 0 Hit Points and doesn't regenerate."})
	must(err)
	m, err := content.NewMonster("fixture_troll", "Fixture Troll", content.MustSourceCitation("Fixture", content.MonsterSection, 333), content.MonsterFields{
		Sizes: []content.CreatureSize{content.SizeLarge}, Type: content.CreatureTypeGiant, Alignment: content.AlignmentUnaligned, ArmorClass: 15, Initiative: init,
		HitPoints: hp, Speeds: []content.SpeedEntry{speed}, Abilities: block, PassivePerception: 10, Traits: []content.StatBlockPart{regen}})
	must(err)
	return m
}

// Requirement: doc/specs/monster-traits.md FR-094 (scenario row), FR-041, FR-045, FR-046 - a Troll-like monster is recognised, hit by Fire, dropped to 0, kept alive Stable, and at its next turn start dies because Regeneration is suppressed, through the public API only.
// Path: unhappy (rule interaction).
func TestRegeneratingMonsterSuppressedByFireDiesAtTurnStart(t *testing.T) {
	t.Parallel()
	id := combat.MustCombatantID("troll")
	recognised := combat.RecogniseTraits(trollMonster(t))
	trait, ok := recognised.Traits()[0].(combat.RegenerationTrait)
	if !ok || len(recognised.Unrecognised()) != 0 {
		t.Fatalf("recognised = %v %v", recognised.Traits(), recognised.Unrecognised())
	}
	state, err := combat.NewRegenerationState(trait)
	if err != nil {
		t.Fatal(err)
	}
	status := combat.MustHitPointStatus(84, combat.DiesInstantly, combat.NoAffinities())
	dmg := []combat.Damage{combat.MustDamage(30, content.DamageFire), combat.MustDamage(54, content.DamageSlashing)}
	tr, err := status.ApplyDamage(id, dmg, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	hit := tr.(combat.HitPointTransitionApplied)
	state = state.AfterDamage(id, []content.DamageType{content.DamageFire, content.DamageSlashing}).State()
	deferral, ok := combat.DeferRegenerationDeath(trait, id, hit).(combat.RegenerationDeferred)
	if !ok || !state.Suppressed() {
		t.Fatalf("deferral = %#v suppressed %v", deferral, state.Suppressed())
	}
	turn, ok := state.AtStartOfTurn(deferral.Status(), id).(combat.RegenerationLetDie)
	if !ok {
		t.Fatalf("turn start did not let the troll die")
	}
	events := turn.Events()
	died, isDied := events[0].(combat.Died)
	if !isDied || died.Cause() != combat.MonsterDeath || turn.Status().LifeState().(combat.Dead).Cause() != combat.MonsterDeath {
		t.Errorf("events = %v status %v", events, turn.Status().LifeState())
	}
}
