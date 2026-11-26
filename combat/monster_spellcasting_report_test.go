package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-spellcasting.md FR-030, amendment A1 - ToCastReport carries the monster source, no class, no slot and the printed save DC and attack bonus exactly (a DC of 22 that no player formula gives), and the result is accepted by ResolveSpellEffects (the printed DC decides the save: a natural 19 fails DC 22) and by NewCastObservation.
// Path: happy.
func TestMonsterCastReportToCastReport(t *testing.T) {
	t.Parallel()
	rig := mscNewRig(t, mscMonster(t, func(f *content.MonsterSpellcastingFields) { f.SaveDC, f.AttackBonus = 22, 11 }))
	p := mscPerformed(t, rig.cast(mscList(t, mscFireball), seFireball()))
	cr, err := p.Report.ToCastReport()
	if err != nil {
		t.Fatalf("ToCastReport() error = %v", err)
	}
	dc, hasDC := cr.SaveDC()
	bonus, hasBonus := cr.AttackBonus()
	_, hasClass := cr.Class()
	_, _, hasSlot := cr.Slot()
	if cr.Source() != character.CastSourceMonster || cr.Spell() != "fireball" || !hasDC || dc.Value() != 22 || !hasBonus || bonus.Value() != 11 || hasClass || hasSlot {
		t.Fatalf("report = %+v, want a monster source for fireball with DC 22, bonus 11, no class and no slot", cr)
	}

	mods, _ := p.Report.AttackModifiers()
	cmd := combat.ResolveSpellEffectsCommand{Spell: seFireball(), Report: cr, Caster: seCaster, AttackModifiers: mods, AbilityModifier: p.Report.AbilityModifier().Value(),
		Targets: []combat.SpellTarget{seTarget("a", nil)}}
	res := seResolved(t, cmd, dicetest.NewFixedRoller(append(append([]int{}, seEightD6...), 19)...))
	if got := res.Targets()[0].Damage(); len(got) != 1 || got[0].HitKind() != combat.NormalHit || got[0].Damages()[0].Amount() != 28 {
		t.Errorf("planned damage = %v, want one full 28 (a natural 19 fails the printed DC 22)", got)
	}

	obs, err := combat.NewCastObservation(cr.Spell(), seCaster, cr.Source(), combat.NewUsedComponents(true, true, false), false, content.NewCastingTimeMagicAction())
	if err != nil || obs.Source() != character.CastSourceMonster {
		t.Errorf("NewCastObservation() = %v, %v, want a monster-source observation", obs, err)
	}
	if a, err := (combat.Counterspell{}).Assess(obs, dspCreature(t, "goblin"), 30, true, dspDC(t)); err != nil {
		t.Errorf("Assess() = %v, %v, want an assessment of a monster cast", a, err)
	} else if _, ok := a.(combat.CounterspellEligible); !ok {
		t.Errorf("Assess() = %T, want CounterspellEligible", a)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-030 - a figure the stat block does not print stays absent in the CastReport, and the zero MonsterCastReport is an error matching srderr.ErrInvalidInput.
// Path: unhappy (missing content, invalid input).
func TestMonsterCastReportToCastReportAbsentFigures(t *testing.T) {
	t.Parallel()
	rig := mscNewRig(t, mscMonster(t, func(f *content.MonsterSpellcastingFields) { f.HasSaveDC, f.HasAttackBonus = false, false }))
	cr, err := mscPerformed(t, rig.cast(mscList(t, mscLight), mscSpellOf("light", 0))).Report.ToCastReport()
	_, hasDC := cr.SaveDC()
	_, hasBonus := cr.AttackBonus()
	if err != nil || hasDC || hasBonus {
		t.Errorf("ToCastReport() = %+v, %v, want no DC and no bonus", cr, err)
	}
	if _, err := (combat.MonsterCastReport{}).ToCastReport(); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero report err = %v, want ErrInvalidInput", err)
	}
}
