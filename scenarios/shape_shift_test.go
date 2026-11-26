package scenarios_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ssForm builds a synthetic Beast form of the given Hit Points and size: made-up numbers, not a stat block of SRD 5.2.1.
func ssForm(t *testing.T, average int, size content.CreatureSize, kind content.CreatureType, cr string) combat.CreatureForm {
	t.Helper()
	scores, saves := map[ability.Ability]ability.Score{}, map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	m := must(content.NewMonster("fixture_form", "Fixture Form", content.MustSourceCitation("Fixture", content.MonsterSection, 258), content.MonsterFields{
		Sizes: []content.CreatureSize{size}, Type: kind, Alignment: content.AlignmentUnaligned, ArmorClass: 12,
		Initiative: must(content.NewStatBlockInitiative(0, 10)), HitPoints: must(content.NewHitPointsEntry(average, dice.MustExpression(2, dice.D8(), 2))),
		Speeds:    []content.SpeedEntry{must(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, ""))},
		Abilities: must(content.NewAbilityBlock(must(ability.NewScores(scores)), saves)), PassivePerception: 10, ChallengeRating: content.MustChallengeRating(cr),
	}))
	return must(combat.NewCreatureForm(must(combat.NewFormFacts(m)), size))
}

// Requirement: doc/specs/shape-changing-spells.md FR-001, FR-018, FR-024, FR-027, FR-032, FR-033, FR-044, FR-056 and the Scenario row of section 8 - a druid polymorphs a hero (the spell id is the bundled Polymorph): the save fails, the form gives 11 Temporary Hit Points, 15 damage ends the spell and costs 4 Hit Points, and ShapeShifting.Revert returns the true form (SRD 5.2.1, "Polymorph", p. 153; "Shape-Shifting", p. 187).
// Path: happy.
func TestPolymorphIsBrokenByDamageAndRevertsTheTrueForm(t *testing.T) {
	t.Parallel()
	spells, hero, druid := combat.ShapeShiftSpells{}, combat.MustCombatantID("hero"), combat.MustCombatantID("druid")
	id := content.SpellID("polymorph")
	spell, known := combat.ShapeShiftSpellOf(id)
	if _, found := must(data.MustBundledSource().Lookup(t.Context(), id.Key())).(content.LookupFound); !found || !known {
		t.Fatalf("the bundled slice lacks %q, or the service does not know it", id)
	}
	start := combat.MustHitPointStatus(30, combat.FallsUnconscious, combat.NoAffinities())
	res := must(spells.Begin(combat.BeginShapeShiftCommand{
		Spell: spell, Caster: druid, Target: hero, Form: ssForm(t, 11, content.SizeMedium, content.CreatureTypeBeast, "1"), Rating: must(combat.NewRatedByLevel(5)),
		Response: combat.Unwilling{Save: d20.Failure}, HitPoints: combat.TrackedHitPoints{Status: start, Choice: combat.AcceptNew},
	}))
	begun, ok := res.(combat.BeginShapeShiftApplied)
	if !ok || begun.Status().HitPoints().Temporary() != 11 {
		t.Fatalf("Begin = %#v, want an applied Polymorph with 11 Temporary Hit Points", res)
	}
	shifted := must((combat.ShapeShifting{}).Shift(combat.ShiftShapeCommand{Creature: hero, Effect: begun.Effect(), Rule: begun.CarryOver()})).(combat.ShiftShapeApplied)
	if !shifted.State().IsShifted() {
		t.Fatal("the creature is not shifted")
	}

	damage := must(begun.Status().ApplyDamage(hero, []combat.Damage{combat.MustDamage(15, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil))
	hurt := damage.(combat.HitPointTransitionApplied).Status()
	assessed, ok := must(spells.Assess(begun.Transformation(), hurt)).(combat.AssessShapeShiftEnded)
	if !ok || hurt.HitPoints().Current() != 26 || hurt.HitPoints().Temporary() != 0 {
		t.Fatalf("Assess = %#v with %d Hit Points and %d Temporary, want the spell ended at 26 and 0", assessed, hurt.HitPoints().Current(), hurt.HitPoints().Temporary())
	}
	ended := must(spells.End(combat.EndShapeShiftCommand{State: begun.Transformation(), Cause: assessed.Cause(), HitPoints: combat.TrackedHitPoints{Status: hurt}})).(combat.EndShapeShiftApplied)
	if !ended.Transformation().IsZero() || ended.Status() != hurt {
		t.Errorf("ended = %+v, want the cleared state and the same status", ended)
	}
	reverted := must((combat.ShapeShifting{}).Revert(combat.RevertToTrueFormCommand{Creature: hero, State: shifted.State(), Rule: ended.CarryOver()})).(combat.RevertToTrueFormApplied)
	if reverted.State().IsShifted() {
		t.Error("the creature is still shifted after the revert")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-045, FR-046, FR-047, FR-048 and the Scenario row of section 8 - a True Polymorph held for the full duration becomes permanent; Concentration ending no longer affects it; a Dispel Magic result (passed as the plain fact CauseDispelled) ends it (SRD 5.2.1, "True Polymorph", p. 170; "Dispel Magic", p. 124).
// Path: happy.
func TestTruePolymorphBecomesPermanentUntilDispelled(t *testing.T) {
	t.Parallel()
	spells, hero, wizard := combat.ShapeShiftSpells{}, combat.MustCombatantID("hero"), combat.MustCombatantID("wizard")
	hp := combat.TrackedHitPoints{Status: combat.MustHitPointStatus(30, combat.FallsUnconscious, combat.NoAffinities()), Choice: combat.AcceptNew}
	begun := must(spells.Begin(combat.BeginShapeShiftCommand{
		Spell: combat.TruePolymorphSpell, Mode: combat.CreatureIntoCreature, Caster: wizard, Target: hero,
		Form: ssForm(t, 30, content.SizeLarge, content.CreatureTypeDragon, "7"), Rating: must(combat.NewRatedByLevel(9)), Response: combat.Willing{}, HitPoints: hp,
	})).(combat.BeginShapeShiftApplied)

	made := must(spells.ElapseFullDuration(begun.Transformation(), true, combat.TrackedHitPoints{Status: begun.Status()})).(combat.ElapseShapeShiftMadePermanent)
	if !made.Transformation().IsPermanent() {
		t.Fatal("the True Polymorph is not permanent")
	}
	held := combat.TrackedHitPoints{Status: begun.Status()}
	if _, unaffected := must(spells.End(combat.EndShapeShiftCommand{State: made.Transformation(), Cause: combat.CauseConcentrationEnded, HitPoints: held, ConcentrationOf: wizard})).(combat.EndShapeShiftUnaffected); !unaffected {
		t.Error("Concentration ending touched a permanent True Polymorph")
	}
	ended, ok := must(spells.End(combat.EndShapeShiftCommand{State: made.Transformation(), Cause: combat.CauseDispelled, HitPoints: held})).(combat.EndShapeShiftApplied)
	if !ok || ended.Status().HitPoints().Temporary() != 0 {
		t.Fatalf("a dispel gave %#v, want the transformation ended with the form's Temporary Hit Points gone", ended)
	}
}
