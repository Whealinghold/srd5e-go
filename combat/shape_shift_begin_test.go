package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func ssBegun(tb testing.TB, c combat.BeginShapeShiftCommand) (combat.BeginShapeShiftingResult, error) {
	tb.Helper()
	return (combat.ShapeShiftSpells{}).Begin(c)
}

// Requirement: per case (table R of doc/specs/shape-changing-spells.md; each case is one row, in the order the table lists them)
// Path: per case.
func TestBeginOrderOfChecks(t *testing.T) {
	t.Parallel()
	dead := combat.TrackedHitPoints{Status: deadStatus(t, combat.DeathSavingThrowFailures), Choice: combat.AcceptNew}
	sc := func(f func(*combat.BeginShapeShiftCommand)) combat.BeginShapeShiftCommand {
		c := ssPolymorph(t)
		f(&c)
		return c
	}
	other := ssShapechange(t)
	other.Target = gid("friend")
	unwilling := ssAnimalShapes(t)
	unwilling.Response = combat.Unwilling{Save: d20.Failure}
	held := sc(func(c *combat.BeginShapeShiftCommand) {
		c.Response = combat.Willing{}
		c.Earlier = ssBegin(t, ssPolymorph(t)).Transformation()
		c.HitPoints = combat.TrackedHitPoints{Status: pcStatus(t, 20, 30, 11), Choice: combat.AcceptNew}
	})
	permanent := ssTruePolymorph(t, combat.CreatureIntoCreature)
	permanentState := luMust((combat.ShapeShiftSpells{}).ElapseFullDuration(ssBegin(t, permanent).Transformation(), true, permanent.HitPoints)).(combat.ElapseShapeShiftMadePermanent).Transformation()
	replacePermanent := ssPolymorph(t)
	replacePermanent.Target, replacePermanent.Response, replacePermanent.Earlier = gid("hero"), combat.Willing{}, permanentState
	objectBecomes := ssBegin(t, ssTruePolymorph(t, combat.ObjectIntoCreature))
	cases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
		want                    func(combat.BeginShapeShiftingResult) bool
	}{
		{"all good", "FR-018", "happy", ssPolymorph(t), isApplied},
		{"willing chooses to fail", "FR-018", "unhappy (rule interaction)", sc(func(c *combat.BeginShapeShiftCommand) { c.Response = combat.Willing{} }), isApplied},
		{"save succeeds", "FR-018", "unhappy (rule interaction)", sc(func(c *combat.BeginShapeShiftCommand) { c.Response = combat.Unwilling{Save: d20.Success} }),
			func(r combat.BeginShapeShiftingResult) bool { return r == combat.BeginShapeShiftResisted{} }},
		{"True Polymorph willing, no roll", "FR-019", "happy", ssTruePolymorph(t, combat.CreatureIntoCreature), isApplied},
		{"True Polymorph unwilling saves", "FR-019", "unhappy (rule interaction)", func() combat.BeginShapeShiftCommand {
			c := ssTruePolymorph(t, combat.CreatureIntoCreature)
			c.Response = combat.Unwilling{Save: d20.Success}
			return c
		}(), func(r combat.BeginShapeShiftingResult) bool { return r == combat.BeginShapeShiftResisted{} }},
		{"True Polymorph unwilling fails", "FR-019", "unhappy (rule interaction)", func() combat.BeginShapeShiftCommand {
			c := ssTruePolymorph(t, combat.CreatureIntoCreature)
			c.Response = combat.Unwilling{Save: d20.Failure}
			return c
		}(), isApplied},
		{"Animal Shapes unwilling", "FR-020", "unhappy (invalid input)", unwilling, refusedWith(combat.TargetNotWilling{})},
		{"Shapechange other target", "FR-021", "unhappy (invalid input)", other, refusedWith(combat.ShapechangeNotSelf{})},
		{"Shapechange unwilling", "FR-021", "unhappy (invalid input)", func() combat.BeginShapeShiftCommand {
			c := ssShapechange(t)
			c.Response = combat.Unwilling{Save: d20.Failure}
			return c
		}(), refusedWith(combat.TargetNotWilling{})},
		{"dead and ineligible", "FR-023, FR-022", "unhappy (rule interaction)", sc(func(c *combat.BeginShapeShiftCommand) {
			c.HitPoints, c.Form, c.Response = dead, ssForm(t, content.CreatureTypeDragon, "9", 5, content.SizeLarge), combat.Willing{}
		}), refusedWith(combat.TargetIsDead{})},
		{"dead and not self", "FR-023", "unhappy (rule interaction)", func() combat.BeginShapeShiftCommand {
			c := other
			c.HitPoints = dead
			return c
		}(), refusedWith(combat.TargetIsDead{})},
		{"not self and unwilling", "FR-023", "unhappy (rule interaction)", func() combat.BeginShapeShiftCommand {
			c := other
			c.Response = combat.Unwilling{Save: d20.Failure}
			return c
		}(), refusedWith(combat.ShapechangeNotSelf{})},
		{"unwilling and ineligible", "FR-023", "unhappy (rule interaction)", func() combat.BeginShapeShiftCommand {
			c := unwilling
			c.Form = ssForm(t, content.CreatureTypeDragon, "9", 5, content.SizeLarge)
			return c
		}(), refusedWith(combat.TargetNotWilling{})},
		{"ineligible and saves", "FR-023", "unhappy (rule interaction)", sc(func(c *combat.BeginShapeShiftCommand) {
			c.Response, c.Form = combat.Unwilling{Save: d20.Success}, ssForm(t, content.CreatureTypeDragon, "9", 5, content.SizeLarge)
		}), func(r combat.BeginShapeShiftingResult) bool {
			f, ok := r.(combat.BeginShapeShiftRefused)
			bad, isBad := f.Rejection.(combat.FormIneligible)
			return ok && isBad && slices.Equal(bad.Reasons(), []combat.FormIneligibility{combat.NotABeast, combat.ChallengeRatingTooHigh})
		}},
		{"replaces earlier", "FR-026", "unhappy (rule interaction)", held, func(r combat.BeginShapeShiftingResult) bool {
			a, ok := r.(combat.BeginShapeShiftApplied)
			return ok && slices.Equal(ssEventKinds(a.Events()), []string{"temporary_hit_points_ended", "shape_shift_ended", "shape_shift_began", "temporary_hit_points_gained"}) &&
				a.Events()[1].(combat.ShapeShiftEnded).Cause() == combat.CauseReplaced
		}},
		{"replaces a permanent transformation", "FR-046", "unhappy (rule interaction)", replacePermanent, func(r combat.BeginShapeShiftingResult) bool {
			a, ok := r.(combat.BeginShapeShiftApplied)
			return ok && a.Events()[0].(combat.ShapeShiftEnded).Cause() == combat.CauseReplaced && !a.Transformation().IsPermanent()
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := ssBegun(t, c.cmd)
			if err != nil || !c.want(got) {
				t.Errorf("Begin = %#v, %v [%s]", got, err, c.requirement)
			}
		})
	}
	// Requirement: FR-062 (table R row "object becomes creature") - the transformation is keyed by the host's id.
	if objectBecomes.Transformation().Parties().Target != gid("hero") || objectBecomes.Transformation().Mode() != combat.ObjectIntoCreature {
		t.Errorf("object into creature kept the id %q, mode %v", objectBecomes.Transformation().Parties().Target, objectBecomes.Transformation().Mode())
	}
}

func isApplied(r combat.BeginShapeShiftingResult) bool {
	_, ok := r.(combat.BeginShapeShiftApplied)
	return ok
}

func refusedWith(want combat.BeginShapeShiftRejection) func(combat.BeginShapeShiftingResult) bool {
	return func(r combat.BeginShapeShiftingResult) bool {
		got, ok := r.(combat.BeginShapeShiftRefused)
		return ok && reflect.DeepEqual(got.Rejection, want)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-022 - a Dead target is refused with TargetIsDead and nothing changes (SRD 5.2.1, "Dead", p. 180), whatever its response; an untracked Dead target (a creature turned into an object) is refused too.
// Path: unhappy (state error).
func TestBeginRefusesADeadTarget(t *testing.T) {
	t.Parallel()
	c := ssTruePolymorph(t, combat.CreatureIntoObject)
	c.HitPoints = combat.UntrackedHitPoints{Life: luMust(combat.NewDead(combat.MassiveDamage))}
	res, err := ssBegun(t, c)
	if err != nil || !refusedWith(combat.TargetIsDead{})(res) {
		t.Errorf("Begin = %#v, %v, want TargetIsDead", res, err)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-024 - an applied Begin carries the new transformation, the new status, the ordered events and the two inputs of ShapeShifting.Shift: SpellSource of the spell id and CarryOverAll (SRD 5.2.1, "Shape-Shifting", p. 187).
// Path: per case.
func TestBeginCarriesTheShiftShapeInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cmd  combat.BeginShapeShiftCommand
		id   content.SpellID
	}{
		{"polymorph", ssPolymorph(t), "polymorph"},
		{"true polymorph", ssTruePolymorph(t, combat.CreatureIntoCreature), "true_polymorph"},
		{"shapechange", ssShapechange(t), "shapechange"},
		{"animal shapes", ssAnimalShapes(t), "animal_shapes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := ssBegin(t, c.cmd)
			want := luMust(combat.SpellSource(c.id))
			if a.Effect() != want || a.CarryOver() != (combat.CarryOverAll{}) {
				t.Errorf("Effect = %v, CarryOver = %v, want %v and CarryOverAll [FR-024]", a.Effect(), a.CarryOver(), want)
			}
			shifted, err := (combat.ShapeShifting{}).Shift(combat.ShiftShapeCommand{Creature: c.cmd.Target, Effect: a.Effect(), Rule: a.CarryOver()})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := shifted.(combat.ShiftShapeApplied); !ok {
				t.Errorf("Shift = %#v, want ShiftShapeApplied", shifted)
			}
			if a.Transformation().Spell() != c.cmd.Spell || a.Transformation().Parties().Target != c.cmd.Target || a.Transformation().Parties().Caster != c.cmd.Caster ||
				a.Transformation().Form() != c.cmd.Form {
				t.Errorf("Transformation = %+v", a.Transformation())
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-025 - the transformation is MaintainedByConcentration for Polymorph, True Polymorph and standard Shapechange and NotMaintained for Animal Shapes and the stat block Shapechange, and it records the caster.
// Path: per case.
func TestBeginRecordsHowTheSpellIsMaintained(t *testing.T) {
	t.Parallel()
	statBlock := ssShapechange(t)
	statBlock.Terms = combat.StatBlockShapechange
	statBlock.Form = ssForm(t, content.CreatureTypeBeast, "5", 40, content.SizeLarge)
	cases := []struct {
		name string
		cmd  combat.BeginShapeShiftCommand
		want combat.MaintainedBy
	}{
		{"polymorph", ssPolymorph(t), combat.MaintainedByConcentration},
		{"true polymorph", ssTruePolymorph(t, combat.CreatureIntoCreature), combat.MaintainedByConcentration},
		{"standard shapechange", ssShapechange(t), combat.MaintainedByConcentration},
		{"animal shapes", ssAnimalShapes(t), combat.NotMaintained},
		{"stat block shapechange", statBlock, combat.NotMaintained},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := ssBegin(t, c.cmd).Transformation(); got.MaintainedBy() != c.want || got.Parties().Caster != c.cmd.Caster {
				t.Errorf("MaintainedBy = %v, Caster = %q, want %v and %q [FR-025]", got.MaintainedBy(), got.Parties().Caster, c.want, c.cmd.Caster)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-036 - the result names the EquipmentOutcome: MeldsUnusable for Polymorph, Animal Shapes and True Polymorph into a creature, TransformsWithCreature for a creature turned into an object, and the caster's choice for Shapechange (SRD 5.2.1, "Polymorph", p. 154; "Shapechange", p. 161).
// Path: per case.
func TestBeginNamesTheEquipmentOutcome(t *testing.T) {
	t.Parallel()
	resizes := ssShapechange(t)
	resizes.Equipment = combat.ResizesToFit
	cases := []struct {
		name string
		cmd  combat.BeginShapeShiftCommand
		want combat.EquipmentOutcome
	}{
		{"polymorph", ssPolymorph(t), combat.MeldsUnusable},
		{"animal shapes", ssAnimalShapes(t), combat.MeldsUnusable},
		{"true polymorph creature into creature", ssTruePolymorph(t, combat.CreatureIntoCreature), combat.MeldsUnusable},
		{"true polymorph object into creature", ssTruePolymorph(t, combat.ObjectIntoCreature), combat.MeldsUnusable},
		{"true polymorph creature into object", ssTruePolymorph(t, combat.CreatureIntoObject), combat.TransformsWithCreature},
		{"shapechange drops", ssShapechange(t), combat.DropsToGround},
		{"shapechange resizes", resizes, combat.ResizesToFit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := ssBegin(t, c.cmd).Equipment(); got != c.want {
				t.Errorf("Equipment() = %v, want %v [FR-036]", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-036, FR-030, FR-055 - Begin returns an error matching ErrInvalidInput and no result for a command no state could make sense of: an equipment choice that is missing, MeldsUnusable or TransformsWithCreature for Shapechange, or set for another spell; UntrackedHitPoints where Temporary Hit Points are granted; a missing or invalid response, choice, Hit Point side, spell, id or form; an unwilling object; an earlier transformation of another target or one with form Temporary Hit Points and no status.
// Path: per case.
func TestBeginRejectsInvalidCommands(t *testing.T) {
	t.Parallel()
	mut := func(base combat.BeginShapeShiftCommand, f func(*combat.BeginShapeShiftCommand)) combat.BeginShapeShiftCommand {
		f(&base)
		return base
	}
	earlierOther := mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) {
		c.Earlier = ssBegin(t, mut(ssPolymorph(t), func(o *combat.BeginShapeShiftCommand) { o.Target = gid("other") })).Transformation()
	})
	earlierTemp := mut(ssTruePolymorph(t, combat.CreatureIntoObject), func(c *combat.BeginShapeShiftCommand) { c.Earlier = ssBegin(t, ssPolymorph(t)).Transformation() })
	cases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
	}{
		{"Shapechange without a choice", "FR-036", "unhappy (invalid input)", mut(ssShapechange(t), func(c *combat.BeginShapeShiftCommand) { c.Equipment = 0 })},
		{"Shapechange melds", "FR-036", "unhappy (invalid input)", mut(ssShapechange(t), func(c *combat.BeginShapeShiftCommand) { c.Equipment = combat.MeldsUnusable })},
		{"Shapechange transforms with creature", "FR-036", "unhappy (invalid input)", mut(ssShapechange(t), func(c *combat.BeginShapeShiftCommand) { c.Equipment = combat.TransformsWithCreature })},
		{"Polymorph with a choice", "FR-036", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Equipment = combat.DropsToGround })},
		{"untracked where Temporary Hit Points are granted", "FR-030", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.HitPoints = combat.UntrackedHitPoints{} })},
		{"untracked True Polymorph creature into creature", "FR-030", "unhappy (invalid input)", mut(ssTruePolymorph(t, combat.CreatureIntoCreature), func(c *combat.BeginShapeShiftCommand) { c.HitPoints = combat.UntrackedHitPoints{} })},
		{"no choice where Temporary Hit Points are granted", "FR-027", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) {
			c.HitPoints = combat.TrackedHitPoints{Status: pcStatus(t, 20, 30, 0)}
		})},
		{"a zero status", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) {
			c.HitPoints = combat.TrackedHitPoints{Choice: combat.AcceptNew}
		})},
		{"no Hit Point side", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.HitPoints = nil })},
		{"no response", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Response = nil })},
		{"a response with no outcome", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Response = combat.Unwilling{} })},
		{"an unwilling object", "FR-019", "unhappy (invalid input)", mut(ssTruePolymorph(t, combat.ObjectIntoCreature), func(c *combat.BeginShapeShiftCommand) { c.Response = combat.Unwilling{Save: d20.Failure} })},
		{"no spell", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Spell = 0 })},
		{"no target", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Target = "" })},
		{"no caster", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Caster = "" })},
		{"no form", "FR-055", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Form = nil })},
		{"an object form for Polymorph", "FR-054", "unhappy (invalid input)", mut(ssPolymorph(t), func(c *combat.BeginShapeShiftCommand) { c.Form = ssObject(t, content.SizeMedium) })},
		{"an earlier transformation of another target", "FR-026", "unhappy (invalid input)", earlierOther},
		{"an earlier form Temporary Hit Points and no status", "FR-034", "unhappy (invalid input)", earlierTemp},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := ssBegun(t, c.cmd)
			if got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("Begin = %#v, %v, want nil and ErrInvalidInput [%s]", got, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-030, FR-029 - UntrackedHitPoints is accepted where none are granted (True Polymorph into an object and of an object into a creature, a stat block Shapechange), with NotGranted provenance and no Temporary Hit Point event; a tracked status is returned unchanged there.
// Path: per case.
func TestBeginGrantsNoTemporaryHitPointsWhereTheSpellGrantsNone(t *testing.T) {
	t.Parallel()
	statBlock := ssShapechange(t)
	statBlock.Terms, statBlock.Form = combat.StatBlockShapechange, ssForm(t, content.CreatureTypeBeast, "5", 40, content.SizeLarge)
	trackedObject := ssTruePolymorph(t, combat.ObjectIntoCreature)
	trackedObject.HitPoints = ssTracked(t, 10, 10, 3, combat.AcceptNew)
	untrackedStatBlock := statBlock
	untrackedStatBlock.HitPoints = combat.UntrackedHitPoints{}
	cases := []struct {
		name string
		cmd  combat.BeginShapeShiftCommand
	}{
		{"object into creature, untracked", ssTruePolymorph(t, combat.ObjectIntoCreature)},
		{"creature into object", ssTruePolymorph(t, combat.CreatureIntoObject)},
		{"stat block Shapechange, tracked", statBlock},
		{"stat block Shapechange, untracked", untrackedStatBlock},
		{"object into creature, tracked", trackedObject},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := ssBegin(t, c.cmd)
			if a.Transformation().TemporaryHitPoints() != combat.TemporaryNotGranted || a.GainedTemporaryHitPointsEvent() ||
				a.Status() != statusOf(c.cmd.HitPoints) {
				t.Errorf("provenance %v, gained %v, status changed %v [FR-029, FR-030]", a.Transformation().TemporaryHitPoints(), a.GainedTemporaryHitPointsEvent(), a.Status() != statusOf(c.cmd.HitPoints))
			}
			for _, k := range ssEventKinds(a.Events()) {
				if k == "temporary_hit_points_gained" {
					t.Errorf("events %v hold a Temporary Hit Point event", ssEventKinds(a.Events()))
				}
			}
		})
	}
}

func statusOf(hp combat.TargetHitPoints) combat.HitPointStatus {
	if t, ok := hp.(combat.TrackedHitPoints); ok {
		return t.Status
	}
	return combat.HitPointStatus{}
}

// Requirement: doc/specs/shape-changing-spells.md FR-051 - Beginning True Polymorph of an object into a creature carries AnimatedObjectTerms: Friendly to the caster and the caster's allies, turns immediately after the caster's, obeys the caster; no other transformation carries them (SRD 5.2.1, "True Polymorph", p. 170).
// Path: per case.
func TestBeginCarriesAnimatedObjectTerms(t *testing.T) {
	t.Parallel()
	terms, ok := ssBegin(t, ssTruePolymorph(t, combat.ObjectIntoCreature)).AnimatedObject()
	if !ok || !slices.Equal(terms.Combatants(), []combat.CombatantID{gid("wizard")}) || !terms.IsFriendlyToCasterAndAllies() || !terms.TakesTurnsAfterCaster() || !terms.ObeysCaster() {
		t.Errorf("AnimatedObject() = %+v, %v", terms, ok)
	}
	for _, c := range []combat.BeginShapeShiftCommand{ssPolymorph(t), ssTruePolymorph(t, combat.CreatureIntoCreature), ssTruePolymorph(t, combat.CreatureIntoObject)} {
		if got, has := ssBegin(t, c).AnimatedObject(); has || !got.IsZero() || got.ObeysCaster() {
			t.Errorf("%v carries animated object terms %+v", c.Spell, got)
		}
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-003, FR-004 - the result carries the statistics kept and what the form forbids, with the Shapechange Spellcasting fact from the command.
// Path: happy.
func TestBeginCarriesStatisticsAndRestrictions(t *testing.T) {
	t.Parallel()
	c := ssShapechange(t)
	c.HoldsSpellcastingFeature = true
	a := ssBegin(t, c)
	wantStats, _ := (combat.ShapeShiftSpells{}).StatisticsOf(combat.ShapechangeSpell, 0)
	if a.Statistics() != wantStats || a.Restrictions().Spellcasting != combat.FormSpellcastingFeatureRetained {
		t.Errorf("Statistics = %v, Restrictions = %+v", a.Statistics(), a.Restrictions())
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-055 and testing.md section 4 - Begin changes none of its inputs and gives equal results for equal inputs; returned event lists are copies.
// Path: unhappy (immutability).
func TestBeginIsPureAndReturnsCopies(t *testing.T) {
	t.Parallel()
	c := ssPolymorph(t)
	before := c
	first, second := ssBegin(t, c), ssBegin(t, c)
	if !reflect.DeepEqual(c, before) || !reflect.DeepEqual(first, second) {
		t.Error("Begin changed its command, or equal commands gave unequal results")
	}
	events := first.Events()
	events[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events() returned a view of the result")
	}
}
