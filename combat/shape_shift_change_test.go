package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func ssChange(tb testing.TB, c combat.ChangeFormCommand) combat.ChangeShapeShiftFormResult {
	tb.Helper()
	res, err := (combat.ShapeShiftSpells{}).ChangeForm(c)
	if err != nil {
		tb.Fatalf("ChangeForm: %v", err)
	}
	return res
}

func refusedChange(want combat.ChangeFormRejection) func(combat.ChangeShapeShiftFormResult) bool {
	return func(r combat.ChangeShapeShiftFormResult) bool {
		got, ok := r.(combat.ChangeFormRefused)
		return ok && reflect.DeepEqual(got.Rejection, want)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-039, FR-040, FR-060 and the check order of FR-060 (dead, FormChangeNotAllowed, NotTransformed, ActorNotCaster, then eligibility); each case is one check
// Path: per case.
func TestChangeForm(t *testing.T) {
	t.Parallel()
	shapechange := ssBegin(t, ssShapechange(t))
	animal := ssBegin(t, ssAnimalShapes(t))
	polymorph := ssBegin(t, ssPolymorph(t))
	permanent, _ := ssPermanent(t, combat.CreatureIntoCreature)
	newDragon := ssForm(t, content.CreatureTypeDragon, "9", 90, content.SizeHuge)
	newBeast := ssForm(t, content.CreatureTypeBeast, "3", 50, content.SizeLarge)
	dead := luMust(combat.NewDead(combat.MassiveDamage))
	scChange := func(f func(*combat.ChangeFormCommand)) combat.ChangeFormCommand {
		c := combat.ChangeFormCommand{Actor: gid("wizard"), State: shapechange.Transformation(), Form: newDragon, Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.ResizesToFit}
		f(&c)
		return c
	}
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ChangeFormCommand
		want                    func(combat.ChangeShapeShiftFormResult) bool
	}{
		{"Shapechange to another form", "FR-040", "happy", scChange(func(*combat.ChangeFormCommand) {}), func(r combat.ChangeShapeShiftFormResult) bool {
			a, ok := r.(combat.ChangeFormApplied)
			return ok && a.Transformation().Form() == newDragon && a.ActionCost() == combat.ShapeShiftMagicAction && a.Equipment() == combat.ResizesToFit &&
				slices.Equal(ssEventKinds(a.Events()), []string{"shape_shift_form_changed"})
		}},
		{"Animal Shapes by the caster", "FR-039", "happy", combat.ChangeFormCommand{Actor: gid("druid"), State: animal.Transformation(), Form: newBeast}, func(r combat.ChangeShapeShiftFormResult) bool {
			a, ok := r.(combat.ChangeFormApplied)
			return ok && a.Transformation().Form() == newBeast && a.Equipment() == combat.MeldsUnusable
		}},
		{"Polymorph", "FR-039", "unhappy (state error)", combat.ChangeFormCommand{Actor: gid("druid"), State: polymorph.Transformation(), Form: newBeast, Rating: ssLevel(9)}, refusedChange(combat.FormChangeNotAllowed{})},
		{"True Polymorph", "FR-039", "unhappy (state error)", combat.ChangeFormCommand{Actor: gid("wizard"), State: ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoCreature)).Transformation(), Form: newBeast, Rating: ssLevel(9)}, refusedChange(combat.FormChangeNotAllowed{})},
		{"a permanent transformation is True Polymorph", "FR-042", "unhappy (state error)", combat.ChangeFormCommand{Actor: gid("wizard"), State: permanent, Form: newBeast, Rating: ssLevel(9)}, refusedChange(combat.FormChangeNotAllowed{})},
		{"no transformation", "FR-042", "unhappy (state error)", combat.ChangeFormCommand{Actor: gid("wizard"), Form: newBeast, Rating: ssLevel(9)}, refusedChange(combat.NotTransformed{})},
		{"another actor", "FR-060", "unhappy (state error)", scChange(func(c *combat.ChangeFormCommand) { c.Actor = gid("hero") }), refusedChange(combat.ActorNotCaster{})},
		{"Animal Shapes by the target", "FR-060", "unhappy (state error)", combat.ChangeFormCommand{Actor: gid("hero"), State: animal.Transformation(), Form: newBeast}, refusedChange(combat.ActorNotCaster{})},
		{"Dead target", "FR-060", "unhappy (state error)", scChange(func(c *combat.ChangeFormCommand) { c.TargetLife = dead }), refusedChange(combat.TargetIsDead{})},
		{"Dead target and another actor", "FR-060", "unhappy (rule interaction)", scChange(func(c *combat.ChangeFormCommand) { c.TargetLife, c.Actor = dead, gid("hero") }), refusedChange(combat.TargetIsDead{})},
		{"Dead target and Polymorph", "FR-060", "unhappy (rule interaction)", combat.ChangeFormCommand{Actor: gid("druid"), State: polymorph.Transformation(), Form: newBeast, TargetLife: dead}, refusedChange(combat.TargetIsDead{})},
		{"not allowed before another actor", "FR-060", "unhappy (rule interaction)", combat.ChangeFormCommand{Actor: gid("hero"), State: polymorph.Transformation(), Form: newBeast, Rating: ssLevel(9)}, refusedChange(combat.FormChangeNotAllowed{})},
		{"another actor before eligibility", "FR-060", "unhappy (rule interaction)", scChange(func(c *combat.ChangeFormCommand) {
			c.Actor, c.Form = gid("hero"), ssForm(t, content.CreatureTypeConstruct, "1", 5, content.SizeMedium)
		}), refusedChange(combat.ActorNotCaster{})},
		{"an ineligible form", "FR-040", "unhappy (rule interaction)", scChange(func(c *combat.ChangeFormCommand) {
			c.Form = ssForm(t, content.CreatureTypeConstruct, "1", 5, content.SizeMedium)
		}), func(r combat.ChangeShapeShiftFormResult) bool {
			f, ok := r.(combat.ChangeFormRefused)
			bad, isBad := f.Rejection.(combat.FormIneligible)
			return ok && isBad && slices.Equal(bad.Reasons(), []combat.FormIneligibility{combat.ConstructOrUndead})
		}},
		{"a kind not seen", "FR-040", "unhappy (state error)", scChange(func(c *combat.ChangeFormCommand) { c.HasSeenKind = false }), func(r combat.ChangeShapeShiftFormResult) bool {
			f, ok := r.(combat.ChangeFormRefused)
			bad, isBad := f.Rejection.(combat.FormIneligible)
			return ok && isBad && slices.Equal(bad.Reasons(), []combat.FormIneligibility{combat.KindNotSeen})
		}},
		{"Animal Shapes to a Huge Beast", "FR-040", "unhappy (boundary)", combat.ChangeFormCommand{Actor: gid("druid"), State: animal.Transformation(), Form: ssForm(t, content.CreatureTypeBeast, "1", 5, content.SizeHuge)}, func(r combat.ChangeShapeShiftFormResult) bool {
			f, ok := r.(combat.ChangeFormRefused)
			bad, isBad := f.Rejection.(combat.FormIneligible)
			return ok && isBad && slices.Equal(bad.Reasons(), []combat.FormIneligibility{combat.SizeTooLarge})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := ssChange(t, c.cmd); !c.want(got) {
				t.Errorf("ChangeForm = %#v [%s]", got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-041, FR-040 - a successful ChangeForm leaves the caster, the target, the MaintainedBy, the spell and the Temporary Hit Point provenance unchanged and only replaces the form; an ineligible form leaves the state equal.
// Path: unhappy (immutability).
func TestChangeFormKeepsEverythingButTheForm(t *testing.T) {
	t.Parallel()
	begun := ssBegin(t, ssShapechange(t))
	before := begun.Transformation()
	newForm := ssForm(t, content.CreatureTypeBeast, "2", 60, content.SizeLarge)
	got := ssChange(t, combat.ChangeFormCommand{Actor: gid("wizard"), State: before, Form: newForm, Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.DropsToGround}).(combat.ChangeFormApplied).Transformation()
	if got.Form() != newForm || got.Parties() != before.Parties() || got.MaintainedBy() != before.MaintainedBy() ||
		got.TemporaryHitPoints() != before.TemporaryHitPoints() || got.Spell() != before.Spell() || got.Terms() != before.Terms() || got.IsPermanent() != before.IsPermanent() {
		t.Errorf("ChangeForm changed more than the form: %+v -> %+v", before, got)
	}
	if before.Form() == newForm {
		t.Error("the input state was changed")
	}
	refused := ssChange(t, combat.ChangeFormCommand{Actor: gid("wizard"), State: before, Form: ssForm(t, content.CreatureTypeUndead, "1", 5, content.SizeMedium), Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.DropsToGround})
	if _, ok := refused.(combat.ChangeFormRefused); !ok || before != begun.Transformation() {
		t.Errorf("an ineligible form changed the state or was not refused: %#v", refused)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-037 - each ChangeForm of Shapechange takes a fresh EquipmentOutcome choice, and Animal Shapes repeats MeldsUnusable.
// Path: unhappy (rule interaction).
func TestChangeFormTakesAFreshEquipmentChoice(t *testing.T) {
	t.Parallel()
	begun := ssBegin(t, ssShapechange(t)) // began with DropsToGround
	form := ssForm(t, content.CreatureTypeBeast, "2", 60, content.SizeLarge)
	got := ssChange(t, combat.ChangeFormCommand{Actor: gid("wizard"), State: begun.Transformation(), Form: form, Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.ResizesToFit}).(combat.ChangeFormApplied)
	if got.Equipment() != combat.ResizesToFit {
		t.Errorf("Equipment() = %v, want the new choice ResizesToFit", got.Equipment())
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-061, FR-036 - ChangeForm for Shapechange without an equipment choice, or with MeldsUnusable or TransformsWithCreature, returns an error matching ErrInvalidInput and no result; for Animal Shapes a supplied choice does too; a zero actor or form does too.
// Path: per case.
func TestChangeFormRejectsInvalidCommands(t *testing.T) {
	t.Parallel()
	shapechange := ssBegin(t, ssShapechange(t)).Transformation()
	animal := ssBegin(t, ssAnimalShapes(t)).Transformation()
	form := ssForm(t, content.CreatureTypeBeast, "2", 60, content.SizeLarge)
	sc := func(e combat.EquipmentOutcome) combat.ChangeFormCommand {
		return combat.ChangeFormCommand{Actor: gid("wizard"), State: shapechange, Form: form, Rating: ssLevel(17), HasSeenKind: true, Equipment: e}
	}
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ChangeFormCommand
	}{
		{"Shapechange without a choice", "FR-061", "unhappy (invalid input)", sc(0)},
		{"Shapechange melds", "FR-061", "unhappy (invalid input)", sc(combat.MeldsUnusable)},
		{"Shapechange transforms with creature", "FR-061", "unhappy (invalid input)", sc(combat.TransformsWithCreature)},
		{"Animal Shapes with a choice", "FR-061", "unhappy (invalid input)", combat.ChangeFormCommand{Actor: gid("druid"), State: animal, Form: form, Equipment: combat.DropsToGround}},
		{"Animal Shapes with the only outcome", "FR-061", "unhappy (invalid input)", combat.ChangeFormCommand{Actor: gid("druid"), State: animal, Form: form, Equipment: combat.MeldsUnusable}},
		{"no actor", "FR-055", "unhappy (invalid input)", combat.ChangeFormCommand{State: animal, Form: form}},
		{"no form", "FR-055", "unhappy (invalid input)", combat.ChangeFormCommand{Actor: gid("druid"), State: animal}},
		{"no rating for Shapechange", "FR-011", "unhappy (invalid input)", combat.ChangeFormCommand{Actor: gid("wizard"), State: shapechange, Form: form, HasSeenKind: true, Equipment: combat.DropsToGround}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.ShapeShiftSpells{}).ChangeForm(c.cmd)
			if got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("ChangeForm = %#v, %v, want nil and ErrInvalidInput [%s]", got, err, c.requirement)
			}
		})
	}
}
