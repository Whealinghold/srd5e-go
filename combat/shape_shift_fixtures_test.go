package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The forms of these tests are synthetic: made-up stat blocks carrying only the numbers the spec's case tables
// quote (a type, a size, a Challenge Rating and the average Hit Points), never a stat block of SRD 5.2.1.

// ssFacts builds the facts of a form of kind, Challenge Rating cr and printed average Hit Points hp, with the given size options.
func ssFacts(tb testing.TB, kind content.CreatureType, cr string, hp int, sizes ...content.CreatureSize) combat.FormFacts {
	tb.Helper()
	m := msMonster(tb, func(f *content.MonsterFields) {
		f.Type, f.ChallengeRating, f.Sizes = kind, content.MustChallengeRating(cr), sizes
		f.HitPoints = luMust(content.NewHitPointsEntry(hp, dice.MustExpression(1, dice.D8(), 0)))
	})
	return luMust(combat.NewFormFacts(m))
}

// ssForm is the creature form of kind and cr in the one size it prints, with hp average Hit Points.
func ssForm(tb testing.TB, kind content.CreatureType, cr string, hp int, size content.CreatureSize) combat.CreatureForm {
	tb.Helper()
	return luMust(combat.NewCreatureForm(ssFacts(tb, kind, cr, hp, size), size))
}

func ssBeast(tb testing.TB, cr string, hp int) combat.CreatureForm {
	tb.Helper()
	return ssForm(tb, content.CreatureTypeBeast, cr, hp, content.SizeMedium)
}

func ssObject(tb testing.TB, size content.CreatureSize) combat.ObjectForm {
	tb.Helper()
	return luMust(combat.NewObjectForm(size))
}

func ssLevel(n int) combat.RatedByLevel { return luMust(combat.NewRatedByLevel(n)) }

func ssCR(s string) combat.RatedByChallenge {
	return combat.NewRatedByChallenge(content.MustChallengeRating(s))
}

// ssTracked is a living creature with the given Hit Points, Temporary Hit Points and choice.
func ssTracked(tb testing.TB, current, maximum, temp int, choice combat.TemporaryHitPointsChoice) combat.TrackedHitPoints {
	tb.Helper()
	return combat.TrackedHitPoints{Status: pcStatus(tb, current, maximum, temp), Choice: choice}
}

// ssPolymorph is a legal Polymorph of "hero" by "druid" into a Medium Beast of Challenge Rating 1 and 11 Hit Points:
// the target fails its save, holds no Temporary Hit Points and takes the new ones.
func ssPolymorph(tb testing.TB) combat.BeginShapeShiftCommand {
	tb.Helper()
	return combat.BeginShapeShiftCommand{
		Spell: combat.PolymorphSpell, Caster: gid("druid"), Target: gid("hero"), Form: ssBeast(tb, "1", 11),
		Response: combat.Unwilling{Save: d20.Failure}, Rating: ssLevel(5), HitPoints: ssTracked(tb, 20, 30, 0, combat.AcceptNew),
	}
}

// ssShapechange is a legal standard Shapechange of "wizard" into a Dragon of Challenge Rating 5 and 40 Hit Points.
func ssShapechange(tb testing.TB) combat.BeginShapeShiftCommand {
	tb.Helper()
	return combat.BeginShapeShiftCommand{
		Spell: combat.ShapechangeSpell, Caster: gid("wizard"), Target: gid("wizard"), Response: combat.Willing{},
		Form:   ssForm(tb, content.CreatureTypeDragon, "5", 40, content.SizeLarge),
		Rating: ssLevel(17), HasSeenKind: true, HitPoints: ssTracked(tb, 30, 30, 0, combat.AcceptNew), Equipment: combat.DropsToGround,
	}
}

// ssAnimalShapes is a legal Animal Shapes on "hero" into a Large Beast of Challenge Rating 2 and 20 Hit Points.
func ssAnimalShapes(tb testing.TB) combat.BeginShapeShiftCommand {
	tb.Helper()
	return combat.BeginShapeShiftCommand{
		Spell: combat.AnimalShapesSpell, Caster: gid("druid"), Target: gid("hero"), Response: combat.Willing{},
		Form:      ssForm(tb, content.CreatureTypeBeast, "2", 20, content.SizeLarge),
		HitPoints: ssTracked(tb, 20, 30, 0, combat.AcceptNew),
	}
}

// ssTruePolymorph is a legal True Polymorph of "hero" by "wizard" in the given mode, taking a form of cr and 30 Hit Points.
func ssTruePolymorph(tb testing.TB, mode combat.TrueTransformation) combat.BeginShapeShiftCommand {
	tb.Helper()
	c := combat.BeginShapeShiftCommand{Spell: combat.TruePolymorphSpell, Mode: mode, Caster: gid("wizard"), Target: gid("hero"), Response: combat.Willing{}}
	switch mode {
	case combat.CreatureIntoCreature:
		c.Form, c.Rating, c.HitPoints = ssBeast(tb, "3", 30), ssCR("5"), ssTracked(tb, 20, 30, 0, combat.AcceptNew)
	case combat.ObjectIntoCreature:
		c.Form, c.Object, c.HitPoints = ssBeast(tb, "1", 30), combat.ObjectFacts{Size: content.SizeMedium}, combat.UntrackedHitPoints{}
	case combat.CreatureIntoObject:
		c.Form, c.TargetSize, c.HitPoints = ssObject(tb, content.SizeMedium), content.SizeMedium, combat.UntrackedHitPoints{}
	}
	return c
}

// ssBegin begins the transformation of c and requires it to be applied.
func ssBegin(tb testing.TB, c combat.BeginShapeShiftCommand) combat.BeginShapeShiftApplied {
	tb.Helper()
	res, err := (combat.ShapeShiftSpells{}).Begin(c)
	if err != nil {
		tb.Fatalf("Begin: %v", err)
	}
	a, ok := res.(combat.BeginShapeShiftApplied)
	if !ok {
		tb.Fatalf("Begin = %#v, want BeginShapeShiftApplied", res)
	}
	return a
}

// ssEventKinds lists the kinds of events in order.
func ssEventKinds(events []combat.Event) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Kind()
	}
	return out
}

// exampleTB lets an Example build a fixture: a failure panics, which fails the example.
type exampleTB struct{ testing.TB }

func (exampleTB) Helper()                           {}
func (exampleTB) Fatalf(format string, args ...any) { panic(fmt.Sprintf(format, args...)) }
