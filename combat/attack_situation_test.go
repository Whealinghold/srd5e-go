package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/attacks.md FR-117, FR-121 - the closed value sets MissOverride, AutomaticMissReason and
// AttackNotPossibleReason name their values in snake_case (the reasons are the srd5e.combat.attack.reason values) and
// an undefined value is "unknown".
// Path: per case.
func TestAttackValueNames(t *testing.T) {
	t.Parallel()
	got := []string{
		combat.MissOverrideNone.String(), combat.MissOverrideTargetNotAtLocation.String(), combat.MissOverride(0).String(),
		combat.AutomaticMissReasonUnderwaterBeyondNormalRange.String(), combat.AutomaticMissReason(0).String(),
		combat.AttackNotPossibleReasonTotalCover.String(), combat.AttackNotPossibleReasonBeyondRange.String(),
		combat.AttackNotPossibleReasonBeyondReach.String(), combat.AttackNotPossibleReasonTargetTooLarge.String(),
		combat.AttackNotPossibleReasonNoFreeHand.String(), combat.AttackNotPossibleReason(6).String(),
	}
	want := []string{"none", "target_not_at_location", "unknown", "underwater_beyond_normal_range", "unknown",
		"total_cover", "beyond_range", "beyond_reach", "target_too_large", "no_free_hand", "unknown"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d = %q, want %q [FR-117]", i, got[i], want[i])
		}
	}
}

// Requirement: doc/specs/attacks.md FR-012, FR-077, FR-120 - every result variant exposes what it carries: the cover, the
// reason, the DC, the chosen save ability and the saving throw, as data.
// Path: happy.
func TestAttackResultsExposeTheirFacts(t *testing.T) {
	t.Parallel()
	half := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	total := []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}

	p := rangedParams()
	p.Cover, p.Distance, p.Situation.Underwater = half, 81, true
	missed, _ := runAttack(t, attackCommand(t, p))
	if m := missed.(combat.AttackAutomaticallyMissed); m.Cover() != combat.HalfCover || m.Reason() != combat.AutomaticMissReasonUnderwaterBeyondNormalRange {
		t.Errorf("automatic miss = %+v, want Half Cover and its reason", m)
	}
	p = meleeParams()
	p.Cover = total
	refused, _ := runAttack(t, attackCommand(t, p))
	if r := refused.(combat.AttackNotPossible); r.Cover() != combat.TotalCover || r.Reason() != combat.AttackNotPossibleReasonTotalCover {
		t.Errorf("not possible = %+v, want Total Cover and its reason", r)
	}

	up := unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	up.Cover = half
	dmg, _ := runUnarmed(t, unarmedCommand(t, up), 14)
	if r := dmg.(combat.UnarmedStrikeDamageResolved); r.ArmorClass() != 15 || r.Cover() != combat.HalfCover || r.Attack().Target() != 17 {
		t.Errorf("damage result = %+v, want AC 15, Half Cover and the effective AC 17 as the roll's target", r)
	}
	up.Option = combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Dexterity, Attempt: rollOf(0)}
	g, _ := runUnarmed(t, unarmedCommand(t, up), 14)
	if r := g.(combat.UnarmedStrikeGrappleResolved); r.SaveAbility() != ability.Dexterity || r.Cover() != combat.HalfCover || r.DC() != 13 ||
		r.Save().(combat.UnarmedStrikeSaveRolled).Result().Kind() != d20.SavingThrow {
		t.Errorf("grapple result = %+v", r)
	}
	up.Option = combat.UnarmedStrikeShoveOption{Effect: combat.GiveProne, SaveAbility: ability.Strength, Attempt: d20.ChooseToFail{}}
	s, _ := runUnarmed(t, unarmedCommand(t, up))
	if r := s.(combat.UnarmedStrikeShoveResolved); r.SaveAbility() != ability.Strength || r.Cover() != combat.HalfCover || r.Effect() != combat.GiveProne {
		t.Errorf("shove result = %+v", r)
	}
	up.Cover = total
	n, _ := runUnarmed(t, unarmedCommand(t, up))
	if r := n.(combat.UnarmedStrikeNotPossible); r.Cover() != combat.TotalCover || r.Reason() != combat.AttackNotPossibleReasonTotalCover {
		t.Errorf("not possible = %+v", r)
	}
}
