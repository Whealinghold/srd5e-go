package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/monster-traits.md FR-002, FR-012, FR-020, FR-031, FR-043, FR-045, FR-061, FR-074, FR-075 - each closed enum constant has its own stable snake_case name, numbered from 1, and the zero value is unknown.
// Path: unhappy (invalid input).
func TestMonsterTraitEnumNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"rejection section", combat.TraitSectionMismatch.String(), "section_mismatch"},
		{"rejection text", combat.TraitTextMismatch.String(), "text_mismatch"},
		{"rejection usage", combat.TraitUsageMismatch.String(), "usage_mismatch"},
		{"rejection parameter", combat.TraitParameterOutOfRange.String(), "parameter_out_of_range"},
		{"rejection damage type", combat.TraitUnknownDamageType.String(), "unknown_damage_type"},
		{"rejection saving throw", combat.TraitNoSavingThrow.String(), "no_saving_throw"},
		{"legendary not tracked", combat.LegendaryResistanceNotTracked.String(), "not_tracked"},
		{"legendary not a failure", combat.LegendaryResistanceNotAFailure.String(), "not_a_failure"},
		{"legendary exhausted", combat.LegendaryResistanceExhausted.String(), "exhausted"},
		{"source spell", combat.SpellOrMagicalEffect.String(), "spell_or_magical_effect"},
		{"source other", combat.OtherSaveSource.String(), "other"},
		{"exemption zero", combat.UndeadFortitudeNotReducedToZero.String(), "not_reduced_to_zero"},
		{"exemption critical", combat.UndeadFortitudeCriticalHit.String(), "critical_hit"},
		{"exemption radiant", combat.UndeadFortitudeRadiantDamage.String(), "radiant_damage"},
		{"skip suppressed", combat.RegenerationSkipSuppressed.String(), "suppressed"},
		{"skip no hit points", combat.RegenerationSkipNoHitPoints.String(), "no_hit_points"},
		{"skip swarm", combat.RegenerationSkipSwarmCannotRegain.String(), "swarm_cannot_regain"},
		{"deferral none", combat.RegenerationNoDeferral.String(), "no_deferral"},
		{"deferral no drop", combat.RegenerationNoDropToZero.String(), "no_drop_to_zero"},
		{"deferral cause", combat.RegenerationCauseNotDeferrable.String(), "cause_not_deferrable"},
		{"swarm regain", combat.SwarmCannotRegainHitPoints.String(), "cannot_regain_hit_points"},
		{"swarm temporary", combat.SwarmCannotGainTemporaryHitPoints.String(), "cannot_gain_temporary_hit_points"},
		{"recipient creature", combat.DamageRecipientCreature.String(), "creature"},
		{"recipient object", combat.DamageRecipientObjectOrStructure.String(), "object_or_structure"},
		{"parry no trait", combat.ParryNoTrait.String(), "no_trait"},
		{"parry missed", combat.ParryAttackMissed.String(), "attack_missed"},
		{"parry ranged", combat.ParryNotMeleeAttackRoll.String(), "not_melee_attack_roll"},
		{"parry no weapon", combat.ParryNotHoldingWeapon.String(), "not_holding_weapon"},
		{"scope checks", combat.SunlightPenaltyAbilityChecksAndAttackRolls.String(), "ability_checks_and_attack_rolls"},
		{"scope all", combat.SunlightPenaltyAllD20Tests.String(), "all_d20_tests"},
		{"zero skip", combat.RegenerationSkipReason(0).String(), "unknown"},
		{"zero parry", combat.ParryUnavailableReason(0).String(), "unknown"},
		{"zero deferral", combat.RegenerationNotDeferredReason(0).String(), "unknown"},
		{"zero exemption", combat.UndeadFortitudeExemption(0).String(), "unknown"},
		{"zero legendary", combat.LegendaryResistanceRefusal(0).String(), "unknown"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if !combat.SpellOrMagicalEffect.Valid() || !combat.OtherSaveSource.Valid() || combat.SaveSourceKind(0).Valid() || combat.SaveSourceKind(3).Valid() {
		t.Error("SaveSourceKind.Valid")
	}
}
