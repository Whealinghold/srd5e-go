package scenarios_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/revival-and-restoration-spells.md sections 4.1 to 4.4 (SRD 5.2.1, "Raise Dead", p. 157; "Resurrection", p. 159) - a fighter killed by Massive Damage is returned
// by Raise Dead (poison neutralised, a curse kept), carries an ordeal of -4 that a Long Rest shrinks, and a cast of Resurrection for a creature dead 400 days leaves its caster Strained
// until a Long Rest, through the public API only.
// Path: happy.
func TestRaiseDeadThenResurrectionStrain(t *testing.T) {
	t.Parallel()
	hero := combat.MustCombatantID("brenna")
	dead, err := combat.NewDead(combat.MassiveDamage)
	if err != nil {
		t.Fatal(err)
	}
	status, err := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 30, 0), dead, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	source, err := combat.LabeledMagicalSource(content.MustFeatureID("hag_curse"))
	if err != nil {
		t.Fatal(err)
	}
	curse, err := combat.NewOngoingEffect(combat.OngoingEffectID("curse-1"), combat.OngoingEffectCurse, source)
	if err != nil {
		t.Fatal(err)
	}
	venom, err := combat.NewOngoingEffect(combat.OngoingEffectID("venom-1"), combat.OngoingEffectMagicalEffect, source)
	if err != nil {
		t.Fatal(err)
	}
	effects, err := combat.NewOngoingEffects([]combat.OngoingEffect{curse, venom})
	if err != nil {
		t.Fatal(err)
	}
	conditions := combat.MustConditionSet([]combat.Condition{combat.Poisoned{}}, nil)
	cmd := combat.ReviveBySpellCommand{Spell: combat.RaiseDead, Subject: hero, Status: status, Conditions: conditions, Effects: effects, Consent: combat.SpiritAccepts,
		Circumstances: combat.MustDeathCircumstances(3*24*time.Hour, false, false, combat.Intact),
		Poisons:       combat.PoisonsAtDeath{Effects: []combat.OngoingEffectID{"venom-1"}, PoisonedConditionFromPoison: true}}
	applied, ok := combat.ReviveBySpell(cmd).(combat.ReviveBySpellApplied)
	if !ok {
		t.Fatalf("ReviveBySpell = %T, want ReviveBySpellApplied", combat.ReviveBySpell(cmd))
	}
	if hp := applied.Revival.Status().HitPoints(); hp.Current() != 1 || hp.Maximum() != 30 || applied.Revival.Effects().Len() != 1 || len(applied.Revival.Conditions().Explicit()) != 0 {
		t.Errorf("revived creature = %+v with %d effects and %d conditions, want 1 of 30, the curse kept and the poison gone", hp, applied.Revival.Effects().Len(), len(applied.Revival.Conditions().Explicit()))
	}
	ordeal := applied.Ordeal
	if ordeal.Modifier() != modifier.New(-4) || ordeal.AfterRest(character.ShortRest) != ordeal || ordeal.AfterRest(character.LongRest).Modifier() != modifier.New(-3) {
		t.Errorf("ordeal %d: after rests %d and %d, want 4, 4 and 3", ordeal.Penalty(), ordeal.AfterRest(character.ShortRest).Penalty(), ordeal.AfterRest(character.LongRest).Penalty())
	}

	cast := combat.ReviveBySpellCommand{Spell: combat.Resurrection, Subject: hero, Status: status, Consent: combat.SpiritAccepts,
		Circumstances: combat.MustDeathCircumstances(400*24*time.Hour, false, false, combat.Intact)}
	res := combat.ReviveBySpell(cast).(combat.ReviveBySpellApplied)
	if res.Strain != combat.Strained || res.Strain.CanCastSpells() || res.Strain.AfterRest(character.LongRest).CanCastSpells() != true {
		t.Errorf("strain = %v, want Strained until a Long Rest", res.Strain)
	}
	if got := res.Revival.Status().HitPoints().Current(); got != 30 {
		t.Errorf("Resurrection returned the creature with %d Hit Points, want all 30", got)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md sections 4.5, 4.7 and 4.8 (SRD 5.2.1, "Aid", p. 107; "Power Word Heal", p. 154; "Death Ward", p. 122) - a creature is warded and aided,
// is saved by Death Ward from a killing blow, the Aid ends and lowers the maximum, and Power Word Heal then restores it and ends Stunned, through the public API only.
// Path: happy.
func TestAidDeathWardAndPowerWordHeal(t *testing.T) {
	t.Parallel()
	hero := combat.MustCombatantID("corin")
	status := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	aid, err := combat.AidIncrease("aid-1", 3)
	if err != nil {
		t.Fatal(err)
	}
	inc, err := combat.IncreaseMaximum(hero, status, combat.HitPointMaximumIncreases{}, aid)
	if err != nil {
		t.Fatal(err)
	}
	aided := inc.(combat.IncreaseMaximumApplied)
	if hp := aided.Status().HitPoints(); hp.Maximum() != 30 || hp.Current() != 30 {
		t.Fatalf("aided = %+v, want 30 of 30", hp)
	}
	blow := []combat.Damage{combat.MustDamage(80, content.DamageSlashing)}
	warded, err := combat.ApplyDamageWithDeathWard(hero, aided.Status(), blow, nil, combat.OtherSource, combat.NormalHit, nil, combat.WardActive)
	if err != nil {
		t.Fatal(err)
	}
	saved := warded.(combat.GuardedDamageApplied)
	if saved.Status().HitPoints().Current() != 1 || saved.Ward() != combat.WardSpent {
		t.Fatalf("after the blow = %+v with ward %v, want 1 Hit Point and the ward spent", saved.Status().HitPoints(), saved.Ward())
	}
	ended, err := combat.EndMaximumIncrease(hero, saved.Status(), aided.Ledger(), "aid-1")
	if err != nil {
		t.Fatal(err)
	}
	if hp := ended.(combat.EndMaximumIncreaseApplied).Status().HitPoints(); hp.Maximum() != 20 || hp.Current() != 1 {
		t.Errorf("after Aid ended = %+v, want 1 of 20", hp)
	}
	heal, err := combat.ApplyRestorativeHealing(combat.PowerWordHeal, hero, ended.(combat.EndMaximumIncreaseApplied).Status(),
		combat.MustConditionSet([]combat.Condition{combat.Stunned{}, combat.Prone{}}, nil), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	healed := heal.(combat.RestorativeHealingApplied)
	if healed.Status().HitPoints().Current() != 20 || len(healed.Conditions().Explicit()) != 1 || healed.Conditions().Explicit()[0].Kind() != content.ConditionProne {
		t.Errorf("after Power Word Heal = %+v and %d conditions, want 20 of 20 with only Prone left", healed.Status().HitPoints(), len(healed.Conditions().Explicit()))
	}
}
