package combat_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// A Polymorph whose target fails its save takes the form's Hit Points as Temporary Hit Points; damage that uses
// them up ends the spell early, and ending it reverts the creature to its true form.
func ExampleShapeShiftSpells_Begin() {
	spells := combat.ShapeShiftSpells{}
	hero := combat.MustCombatantID("hero")
	bear := combat.MustCombatantID("druid")
	status := combat.MustHitPointStatus(30, combat.FallsUnconscious, combat.NoAffinities())
	facts, _ := combat.NewFormFacts(msMonster(&exampleTB{}, func(f *content.MonsterFields) {
		f.Type, f.ChallengeRating = content.CreatureTypeBeast, content.MustChallengeRating("1")
		f.Sizes = []content.CreatureSize{content.SizeMedium}
	}))
	form, _ := combat.NewCreatureForm(facts, content.SizeMedium)
	rating, _ := combat.NewRatedByLevel(5)

	res, _ := spells.Begin(combat.BeginShapeShiftCommand{
		Spell: combat.PolymorphSpell, Caster: bear, Target: hero, Form: form, Rating: rating,
		Response:  combat.Unwilling{Save: d20.Failure},
		HitPoints: combat.TrackedHitPoints{Status: status, Choice: combat.AcceptNew},
	})
	begun := res.(combat.BeginShapeShiftApplied)
	fmt.Println(begun.Transformation().Spell(), begun.Status().HitPoints().Temporary())

	after, _ := begun.Status().ApplyDamage(hero, []combat.Damage{combat.MustDamage(200, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil)
	hurt := after.(combat.HitPointTransitionApplied).Status()
	assessed, _ := spells.Assess(begun.Transformation(), hurt)
	fmt.Println(assessed.(combat.AssessShapeShiftEnded).Cause())
	// Output:
	// polymorph 150
	// form_hit_points_depleted
}
