package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-druid.md FR-048, FR-104, FR-118, FR-064 (SRD 5.2.1, pp. 42-46) - the closed enums have the names of their members,
// "none" or "unknown" for the zero value and past the last, and Valid is true exactly for the members.
// Path: per case (unhappy: boundary).
func TestDruidEnums(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label     string
		got, want string
	}{
		{"Magician", character.PrimalOrderMagician.String(), "magician"}, {"Warden", character.PrimalOrderWarden.String(), "warden"}, {"no order", character.PrimalOrder(0).String(), "none"},
		{"Potent", character.ElementalFuryPotentSpellcasting.String(), "potent_spellcasting"}, {"Primal", character.ElementalFuryPrimalStrike.String(), "primal_strike"}, {"no fury", character.ElementalFury(0).String(), "none"},
		{"arid", character.LandArid.String(), "arid"}, {"polar", character.LandPolar.String(), "polar"}, {"temperate", character.LandTemperate.String(), "temperate"}, {"tropical", character.LandTropical.String(), "tropical"},
		{"no land", character.LandType(0).String(), "none"}, {"land past", character.LandType(5).String(), "none"},
		{"cost none", character.DruidActionCostNone.String(), "none"}, {"cost magic", character.DruidActionCostMagic.String(), "magic_action"}, {"cost bonus", character.DruidActionCostBonus.String(), "bonus_action"}, {"cost past", character.DruidActionCost(9).String(), "unknown"},
		{"window land", character.RestWindowLandTypeChange.String(), "land_type_change"}, {"window forms", character.RestWindowKnownFormReplacement.String(), "known_form_replacement"},
		{"window recovery", character.RestWindowNaturalRecoveryRecovery.String(), "natural_recovery_recovery"}, {"window zero", character.RestWindowPermission(0).String(), "unknown"},
		{"end elapsed", character.WildShapeEndDurationElapsed.String(), "duration_elapsed"}, {"end incapacitated", character.WildShapeEndIncapacitated.String(), "incapacitated"},
		{"end died", character.WildShapeEndDied.String(), "died"}, {"end again", character.WildShapeEndUsedWildShapeAgain.String(), "used_wild_shape_again"},
		{"end early", character.WildShapeEndLeftEarly.String(), "left_early"}, {"end zero", character.WildShapeEndReason(0).String(), "unknown"},
		{"cube minute", character.SanctuaryEndMinuteElapsed.String(), "minute_elapsed"}, {"cube incapacitated", character.SanctuaryEndIncapacitated.String(), "incapacitated"},
		{"cube died", character.SanctuaryEndDied.String(), "died"}, {"cube zero", character.SanctuaryEndReason(0).String(), "unknown"},
		{"attribute first", character.DruidFeatureWildShape.String(), "wild_shape"}, {"attribute last", character.DruidFeaturePrimalStrike.String(), "primal_strike"},
		{"attribute zero", character.DruidFeatureAttribute(0).String(), "unknown"}, {"attribute past", character.DruidFeatureAttribute(9).String(), "unknown"},
		{"invariant first", character.DruidViolationWildShapeRows.String(), "wild_shape_rows"}, {"invariant zero", character.DruidInvariantViolation(0).String(), "unknown"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.label, c.got, c.want)
		}
	}
	for _, c := range []struct {
		label string
		got   bool
		want  bool
	}{
		{"Magician", character.PrimalOrderMagician.Valid(), true}, {"order 3", character.PrimalOrder(3).Valid(), false}, {"order 0", character.PrimalOrder(0).Valid(), false},
		{"Primal Strike", character.ElementalFuryPrimalStrike.Valid(), true}, {"fury 3", character.ElementalFury(3).Valid(), false},
		{"arid", character.LandArid.Valid(), true}, {"tropical", character.LandTropical.Valid(), true}, {"land 5", character.LandType(5).Valid(), false}, {"land 0", character.LandType(0).Valid(), false},
		{"window 1", character.RestWindowLandTypeChange.Valid(), true}, {"window 3", character.RestWindowNaturalRecoveryRecovery.Valid(), true}, {"window 4", character.RestWindowPermission(4).Valid(), false},
		{"source weapon", character.PrimalStrikeSourceWeapon.Valid(), true}, {"source other", character.PrimalStrikeSourceOther.Valid(), true}, {"source 5", character.PrimalStrikeAttackSource(5).Valid(), false}, {"source 0", character.PrimalStrikeAttackSource(0).Valid(), false},
	} {
		if c.got != c.want {
			t.Errorf("Valid %s = %v, want %v", c.label, c.got, c.want)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-171 (content-monsters.md FR-089) - BeastFormFactsFromMonster maps the monster id, creature type and Challenge
// Rating and sets the Fly Speed iff any speed has the fly mode: a walking Beast is false, a flying Beast true, a swimming and climbing Beast false, a
// Beast with "Climb or Fly" true and a non-Beast keeps its type; NewBeastFormFacts refuses an invalid id.
// Path: per case (happy / unhappy: invalid input, boundary).
func TestBeastFormFacts(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label, kind, cr, speeds string
		fly                     bool
		want                    content.CreatureType
	}{
		{"walking", "Beast", "1/4", `[{"modes":["walk"],"feet":30}]`, false, content.CreatureTypeBeast},
		{"flying", "Beast", "1/2", `[{"modes":["walk"],"feet":10},{"modes":["fly"],"feet":60}]`, true, content.CreatureTypeBeast},
		{"swimming and climbing", "Beast", "0", `[{"modes":["swim"],"feet":30},{"modes":["climb"],"feet":30}]`, false, content.CreatureTypeBeast},
		{"climb or fly", "Beast", "1", `[{"modes":["climb","fly"],"feet":20}]`, true, content.CreatureTypeBeast},
		{"not a Beast", "Dragon", "3", `[{"modes":["walk"],"feet":30}]`, false, content.CreatureTypeDragon},
	} {
		f := character.BeastFormFactsFromMonster(druMonster(t, c.kind, c.cr, c.speeds))
		if f.ID() != "fixture_beast" || f.CreatureType() != c.want || f.ChallengeRating().String() != c.cr || f.HasFlySpeed() != c.fly {
			t.Errorf("%s: facts = %v %v %v %v", c.label, f.ID(), f.CreatureType(), f.ChallengeRating(), f.HasFlySpeed())
		}
	}
	_, err := character.NewBeastFormFacts("", content.CreatureTypeBeast, content.ChallengeRating{}, false)
	wantInvalid(t, err, "empty id")
	f := druForm("wolf", "1/4", false)
	if f.ID() != "wolf" || f.CreatureType() != content.CreatureTypeBeast || f.HasFlySpeed() {
		t.Errorf("form = %v", f)
	}
}
