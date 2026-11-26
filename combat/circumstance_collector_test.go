package combat_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// scene is the input of one collector row; the zero value is a melee weapon attack at 5 feet.
type scene struct {
	form     combat.AttackForm
	distance int
	rng      combat.AttackRange
	reach    combat.Reach
	cover    combat.Cover
	enemies  []combat.NearbyEnemy
	sit      combat.AttackSituation
	damage   combat.DamageSpec
}

func (s scene) collect() combat.AttackPlan {
	form := s.form
	if form == 0 {
		form = combat.MeleeWeaponAttack
	}
	return (combat.CircumstanceCollector{}).Collect(form, s.distance, s.rng, s.reach, s.cover, s.enemies, s.sit, s.damage)
}

func proceeds(override combat.MissOverride, reveals bool, cs ...modifier.RollCircumstance) combat.AttackPlan {
	var circs []modifier.RollCircumstance
	circs = append(circs, cs...)
	return combat.AttackPlanProceeds{Circumstances: circs, MissOverride: override, RevealsAttackerLocation: reveals}
}

func plain(cs ...modifier.RollCircumstance) combat.AttackPlan {
	return proceeds(combat.MissOverrideNone, false, cs...)
}

func no(r combat.AttackNotPossibleReason) combat.AttackPlan {
	return combat.AttackPlanNotPossible{Reason: r}
}

// Requirement: per case
// Path: per case.
func TestCircumstanceCollectorCollect(t *testing.T) {
	t.Parallel()
	ranged := combat.RangedWeaponAttack
	r80, r60, r120 := combat.MustAttackRange(80, 320), combat.MustAttackRange(60, 0), combat.MustAttackRange(60, 120)
	enemy := func(d int, sees, incap bool) []combat.NearbyEnemy {
		return []combat.NearbyEnemy{combat.MustNearbyEnemy(d, sees, incap)}
	}
	pierce := combat.MustDamageSpec(combat.MustFixedDamage(1, content.DamagePiercing, 0, false))
	mixed := combat.MustDamageSpec(combat.MustFixedDamage(1, content.DamageSlashing, 0, false), combat.MustFixedDamage(1, content.DamagePiercing, 0, false))
	slash := combat.MustDamageSpec(combat.MustDiceDamage(1, dice.D8(), content.DamageSlashing, 0, true))
	water := combat.AttackSituation{Underwater: true}
	swim := combat.AttackSituation{Underwater: true, SwimSpeed: true}
	cases := []struct {
		name, requirement, path string
		in                      scene
		want                    combat.AttackPlan
	}{
		// Reach (FR-029 to FR-032, FR-088).
		{"melee at reach", "FR-030", "happy", scene{distance: 5}, plain()},
		{"melee one beyond reach", "FR-031", "unhappy (boundary)", scene{distance: 6}, no(combat.AttackNotPossibleReasonBeyondReach)},
		{"reach 10 at 10", "FR-032", "unhappy (boundary)", scene{distance: 10, reach: combat.MustReach(10)}, plain()},
		{"reach 10 at 11", "FR-032", "unhappy (boundary)", scene{distance: 11, reach: combat.MustReach(10)}, no(combat.AttackNotPossibleReasonBeyondReach)},
		{"melee spell at 6", "FR-088, FR-031", "unhappy (boundary)", scene{form: combat.MeleeSpellAttack, distance: 6}, no(combat.AttackNotPossibleReasonBeyondReach)},
		{"unarmed strike ignores a supplied reach", "FR-032, FR-106", "unhappy (boundary)", scene{form: combat.UnarmedStrikeAttack, distance: 6, reach: combat.MustReach(10)}, no(combat.AttackNotPossibleReasonBeyondReach)},
		{"negative distance is read as 0", "FR-026", "unhappy (invalid input)", scene{distance: -4}, plain()},
		{"melee ignores range rules", "FR-029", "unhappy (rule interaction)", scene{distance: 5, rng: r80, enemies: enemy(5, true, false)}, plain()},
		{"melee spell with an enemy adjacent", "FR-088", "unhappy (rule interaction)", scene{form: combat.MeleeSpellAttack, distance: 5, enemies: enemy(5, true, false)}, plain()},
		// Range (FR-021 to FR-026, FR-089).
		{"single range at its limit", "FR-021", "happy", scene{form: ranged, distance: 60, rng: r60}, plain()},
		{"single range one beyond", "FR-022", "unhappy (boundary)", scene{form: ranged, distance: 61, rng: r60}, no(combat.AttackNotPossibleReasonBeyondRange)},
		{"distance 0", "FR-026", "unhappy (boundary)", scene{form: ranged, distance: 0, rng: r80}, plain()},
		{"at normal range", "FR-023", "unhappy (boundary)", scene{form: ranged, distance: 80, rng: r80}, plain()},
		{"one beyond normal", "FR-024", "unhappy (boundary)", scene{form: ranged, distance: 81, rng: r80}, plain(disadvantage)},
		{"at long range", "FR-024", "unhappy (boundary)", scene{form: ranged, distance: 320, rng: r80}, plain(disadvantage)},
		{"one beyond long range", "FR-025", "unhappy (boundary)", scene{form: ranged, distance: 321, rng: r80}, no(combat.AttackNotPossibleReasonBeyondRange)},
		{"ranged spell 61 of 60/120", "FR-089, FR-024", "unhappy (boundary)", scene{form: combat.RangedSpellAttack, distance: 61, rng: r120}, plain(disadvantage)},
		{"ranged spell 121 of 120", "FR-089, FR-022", "unhappy (boundary)", scene{form: combat.RangedSpellAttack, distance: 121, rng: combat.MustAttackRange(120, 0)}, no(combat.AttackNotPossibleReasonBeyondRange)},
		// Close combat (FR-027, FR-028).
		{"enemy at 5 feet", "FR-027, FR-028", "unhappy (boundary)", scene{form: ranged, distance: 30, rng: r80, enemies: enemy(5, true, false)}, plain(disadvantage)},
		{"enemy at 6 feet", "FR-028", "unhappy (boundary)", scene{form: ranged, distance: 30, rng: r80, enemies: enemy(6, true, false)}, plain()},
		{"enemy cannot see the attacker", "FR-028", "unhappy (rule interaction)", scene{form: ranged, distance: 30, rng: r80, enemies: enemy(5, false, false)}, plain()},
		{"enemy is Incapacitated", "FR-028", "unhappy (rule interaction)", scene{form: ranged, distance: 30, rng: r80, enemies: enemy(5, true, true)}, plain()},
		{"no enemies", "FR-028", "unhappy (empty and minimal case)", scene{form: ranged, distance: 30, rng: r80}, plain()},
		{"two enemies, one qualifies", "FR-027", "unhappy (rule interaction)", scene{form: ranged, distance: 30, rng: r80,
			enemies: []combat.NearbyEnemy{combat.MustNearbyEnemy(5, false, false), combat.MustNearbyEnemy(3, true, false)}}, plain(disadvantage)},
		{"ranged spell with an enemy adjacent", "FR-089, FR-027", "unhappy (rule interaction)", scene{form: combat.RangedSpellAttack, distance: 30, rng: r80, enemies: enemy(5, true, false)}, plain(disadvantage)},
		{"long range and close combat", "FR-043", "unhappy (rule interaction)", scene{form: ranged, distance: 100, rng: r80, enemies: enemy(5, true, false)}, plain(disadvantage, disadvantage)},
		// Unseen attackers and targets (FR-034 to FR-038, FR-104).
		{"target unseen but at the location", "FR-034", "unhappy (rule interaction)", scene{sit: combat.AttackSituation{Visibility: combat.TargetUnseen{TargetAtTargetedLocation: true}}}, plain(disadvantage)},
		{"target unseen and elsewhere", "FR-036", "unhappy (state error)", scene{sit: combat.AttackSituation{Visibility: combat.TargetUnseen{}}}, proceeds(combat.MissOverrideTargetNotAtLocation, false, disadvantage)},
		{"target cannot see the attacker", "FR-035", "unhappy (rule interaction)", scene{sit: combat.AttackSituation{TargetCannotSeeAttacker: true}}, plain(advantage)},
		{"neither can see the other", "FR-038", "unhappy (rule interaction)", scene{sit: combat.AttackSituation{Visibility: combat.TargetUnseen{TargetAtTargetedLocation: true}, TargetCannotSeeAttacker: true}}, plain(disadvantage, advantage)},
		{"seen target", "FR-104", "happy", scene{sit: combat.AttackSituation{Visibility: combat.TargetSeen{}}}, plain()},
		{"hidden attacker reveals its location", "FR-037", "unhappy (rule interaction)", scene{sit: combat.AttackSituation{AttackerHidden: true}}, proceeds(combat.MissOverrideNone, true)},
		// Underwater (FR-039 to FR-042, FR-103, FR-113).
		{"underwater melee weapon, no Swim Speed", "FR-039", "unhappy (rule interaction)", scene{sit: water, damage: slash}, plain(disadvantage)},
		{"underwater melee weapon, Piercing", "FR-040, FR-103", "unhappy (boundary)", scene{sit: water, damage: pierce}, plain()},
		{"underwater melee weapon, Slashing and Piercing", "FR-103", "unhappy (rule interaction)", scene{sit: water, damage: mixed}, plain()},
		{"underwater melee weapon, Swim Speed", "FR-040", "unhappy (boundary)", scene{sit: swim, damage: slash}, plain()},
		{"underwater melee weapon, no damage spec", "FR-103", "unhappy (empty and minimal case)", scene{sit: water}, plain(disadvantage)},
		{"underwater melee spell", "FR-040, FR-088", "unhappy (boundary)", scene{form: combat.MeleeSpellAttack, sit: water}, plain()},
		{"underwater unarmed strike", "FR-113", "unhappy (rule interaction)", scene{form: combat.UnarmedStrikeAttack, sit: water, damage: slash}, plain()},
		{"underwater ranged weapon within normal", "FR-042", "unhappy (rule interaction)", scene{form: ranged, distance: 80, rng: r80, sit: water}, plain(disadvantage)},
		{"underwater ranged weapon within normal, Swim Speed", "FR-042", "unhappy (rule interaction)", scene{form: ranged, distance: 80, rng: r80, sit: swim}, plain(disadvantage)},
		{"underwater ranged weapon within a single range", "FR-042", "unhappy (boundary)", scene{form: ranged, distance: 60, rng: r60, sit: water}, plain(disadvantage)},
		{"underwater ranged weapon beyond normal", "FR-041", "unhappy (rule interaction)", scene{form: ranged, distance: 81, rng: r80, sit: water},
			combat.AttackPlanAutomaticMiss{Reason: combat.AutomaticMissReasonUnderwaterBeyondNormalRange}},
		{"automatic miss from a hidden attacker", "FR-041, FR-037", "unhappy (rule interaction)", scene{form: ranged, distance: 81, rng: r80, sit: combat.AttackSituation{Underwater: true, AttackerHidden: true}},
			combat.AttackPlanAutomaticMiss{Reason: combat.AutomaticMissReasonUnderwaterBeyondNormalRange, RevealsAttackerLocation: true}},
		{"underwater ranged beyond long range is not possible", "FR-045", "unhappy (rule interaction)", scene{form: ranged, distance: 321, rng: r80, sit: water}, no(combat.AttackNotPossibleReasonBeyondRange)},
		{"underwater ranged spell beyond normal", "FR-089, FR-040", "unhappy (boundary)", scene{form: combat.RangedSpellAttack, distance: 81, rng: r80, sit: water}, plain(disadvantage)},
		// Total Cover and precedence (FR-019, FR-105).
		{"Total Cover", "FR-019", "unhappy (state error)", scene{cover: combat.TotalCover}, no(combat.AttackNotPossibleReasonTotalCover)},
		{"Total Cover and beyond range", "FR-105", "unhappy (rule interaction)", scene{form: ranged, distance: 999, rng: r80, cover: combat.TotalCover}, no(combat.AttackNotPossibleReasonTotalCover)},
		{"Half Cover changes nothing here", "FR-020", "happy", scene{cover: combat.HalfCover}, plain()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.in.collect(); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Collect = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-043, FR-038 - the circumstances the collector returns combine by the normal rule:
// several of one kind count once and both kinds cancel.
// Path: unhappy (rule interaction).
func TestCollectedCircumstancesCombine(t *testing.T) {
	t.Parallel()
	plan := scene{form: combat.RangedWeaponAttack, distance: 100, rng: combat.MustAttackRange(80, 320),
		enemies: []combat.NearbyEnemy{combat.MustNearbyEnemy(5, true, false)}}.collect().(combat.AttackPlanProceeds)
	if got := modifier.Resolve(plan.Circumstances...); got != modifier.WithDisadvantage {
		t.Errorf("long range plus close combat resolves to %v, want one Disadvantage", got)
	}
	both := scene{sit: combat.AttackSituation{Visibility: combat.TargetUnseen{TargetAtTargetedLocation: true}, TargetCannotSeeAttacker: true}}.
		collect().(combat.AttackPlanProceeds)
	if got := modifier.Resolve(both.Circumstances...); got != modifier.Normal {
		t.Errorf("unseen target and unseen attacker resolve to %v, want Normal", got)
	}
}
