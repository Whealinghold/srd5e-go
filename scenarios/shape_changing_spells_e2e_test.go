package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpAHealthy is a creature of 30 Hit Points that falls Unconscious at 0.
func grpAHealthy() combat.HitPointStatus {
	return combat.MustHitPointStatus(30, combat.FallsUnconscious, combat.NoAffinities())
}

// grpADead is a creature that has died of massive damage.
func grpADead(t *testing.T) combat.HitPointStatus {
	t.Helper()
	start := combat.MustHitPointStatus(5, combat.DiesInstantly, combat.NoAffinities())
	hit, err := start.ApplyDamage(combat.MustCombatantID("fallen"), []combat.Damage{combat.MustDamage(50, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	return hit.(combat.HitPointTransitionApplied).Status()
}

func grpABeast(t *testing.T, size content.CreatureSize, cr string) combat.CreatureForm {
	t.Helper()
	return ssForm(t, 12, size, content.CreatureTypeBeast, cr)
}

// grpABegin runs BeginShapeShifting and returns the result.
func grpABegin(t *testing.T, cmd combat.BeginShapeShiftCommand) combat.BeginShapeShiftingResult {
	t.Helper()
	res, err := combat.NewBeginShapeShifting().Run(context.Background(), combat.BeginShapeShiftingCommand(cmd))
	if err != nil {
		t.Fatalf("BeginShapeShifting: %v", err)
	}
	return res
}

// Requirement: doc/specs/shape-changing-spells.md FR-009, FR-010, FR-016, FR-017, FR-018, FR-020, FR-021, FR-022, FR-023, FR-024, FR-025, FR-027, FR-036 - a druid shape-shifts willing allies with Animal Shapes: a Large Beast of Challenge Rating 4 is applied with Temporary Hit Points, no Concentration and melded gear; an unwilling target, a Challenge Rating 5 form, a Humanoid form and a dead target are each refused for the first failing check; Polymorph lets a target resist with its save, Shapechange only the caster and needs the gear choice.
// Path: unhappy (rule interaction).
func TestShapeChangingSpellsBeginScenario(t *testing.T) {
	t.Parallel()
	druid, ally := combat.MustCombatantID("druid"), combat.MustCombatantID("ally")
	rating := grpAMust(combat.NewRatedByLevel(5))
	animal := func(form combat.CreatureForm, response combat.TargetResponse, hp combat.TargetHitPoints) combat.BeginShapeShiftCommand {
		return combat.BeginShapeShiftCommand{Spell: combat.AnimalShapesSpell, Caster: druid, Target: ally, Form: form, Response: response, Rating: rating, HitPoints: hp}
	}
	tracked := combat.TrackedHitPoints{Status: grpAHealthy(), Choice: combat.AcceptNew}

	applied, ok := grpABegin(t, animal(grpABeast(t, content.SizeLarge, "4"), combat.Willing{}, tracked)).(combat.BeginShapeShiftApplied)
	if !ok || applied.Status().HitPoints().Temporary() != 12 || applied.Equipment() != combat.MeldsUnusable || !applied.GainedTemporaryHitPointsEvent() ||
		applied.Transformation().MaintainedBy() != combat.NotMaintained || applied.Transformation().IsPermanent() {
		t.Fatalf("a Large Beast of CR 4 = %#v, want applied: 12 Temporary Hit Points, gear melded, no Concentration", applied)
	}
	if applied.Effect().IsZero() {
		t.Errorf("the effect handed to ShiftShape must be the spell's source")
	}

	refusals := map[string]struct {
		cmd  combat.BeginShapeShiftCommand
		want string
	}{
		"an unwilling target":       {animal(grpABeast(t, content.SizeLarge, "1"), combat.Unwilling{Save: d20.Failure}, tracked), "target_not_willing"},
		"a Challenge Rating 5 form": {animal(grpABeast(t, content.SizeLarge, "5"), combat.Willing{}, tracked), "form_ineligible"},
		"a Huge form":               {animal(grpABeast(t, content.SizeHuge, "1"), combat.Willing{}, tracked), "form_ineligible"},
		"a Humanoid form":           {animal(ssForm(t, 12, content.SizeMedium, content.CreatureTypeHumanoid, "1"), combat.Willing{}, tracked), "form_ineligible"},
		"a dead target":             {animal(grpABeast(t, content.SizeMedium, "1"), combat.Willing{}, combat.TrackedHitPoints{Status: grpADead(t), Choice: combat.AcceptNew}), "target_is_dead"},
		"dead outranks not willing": {animal(grpABeast(t, content.SizeMedium, "1"), combat.Unwilling{Save: d20.Failure}, combat.TrackedHitPoints{Status: grpADead(t), Choice: combat.AcceptNew}), "target_is_dead"},
	}
	for name, c := range refusals {
		r, ok := grpABegin(t, c.cmd).(combat.BeginShapeShiftRefused)
		if !ok || r.Rejection.String() != c.want {
			t.Errorf("%s: result = %#v, want refused %s", name, r, c.want)
		}
	}
	ineligible := grpABegin(t, animal(ssForm(t, 12, content.SizeHuge, content.CreatureTypeHumanoid, "9"), combat.Willing{}, tracked)).(combat.BeginShapeShiftRefused)
	if reasons := ineligible.Rejection.(combat.FormIneligible).Reasons(); len(reasons) < 3 || !slices.IsSorted(reasons) {
		t.Errorf("reasons = %v, want every violated rule listed in the fixed order", reasons)
	}

	polymorph := func(response combat.TargetResponse) combat.BeginShapeShiftCommand {
		return combat.BeginShapeShiftCommand{Spell: combat.PolymorphSpell, Caster: druid, Target: ally, Form: grpABeast(t, content.SizeMedium, "1"), Response: response, Rating: rating, HitPoints: tracked}
	}
	if got := grpABegin(t, polymorph(combat.Unwilling{Save: d20.Success})); got != combat.BeginShapeShiftingResult(combat.BeginShapeShiftResisted{}) {
		t.Errorf("an unwilling target that saves = %#v, want resisted", got)
	}
	if p, ok := grpABegin(t, polymorph(combat.Unwilling{Save: d20.Failure})).(combat.BeginShapeShiftApplied); !ok || p.Transformation().MaintainedBy() == combat.NotMaintained {
		t.Errorf("an unwilling target that fails = %#v, want applied and held by Concentration", p)
	}

	shapechange := combat.BeginShapeShiftCommand{Spell: combat.ShapechangeSpell, Caster: druid, Target: druid, Form: grpABeast(t, content.SizeMedium, "1"), Response: combat.Willing{},
		Rating: rating, HasSeenKind: true, Equipment: combat.DropsToGround, HitPoints: tracked}
	if _, ok := grpABegin(t, shapechange).(combat.BeginShapeShiftApplied); !ok {
		t.Errorf("Shapechange on the caster with a gear choice must apply")
	}
	other := shapechange
	other.Target = ally
	if r, ok := grpABegin(t, other).(combat.BeginShapeShiftRefused); !ok || r.Rejection.String() != "shapechange_not_self" {
		t.Errorf("Shapechange on another creature = %#v, want refused shapechange_not_self", r)
	}
	noChoice := shapechange
	noChoice.Equipment = 0
	if _, err := combat.NewBeginShapeShifting().Run(context.Background(), combat.BeginShapeShiftingCommand(noChoice)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("Shapechange with no gear choice err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-039, FR-040, FR-041, FR-042, FR-060, FR-061, FR-031 - an Animal Shapes druid changes the target's form with a Magic action: the Hit Points, Temporary Hit Points and the caster stay; another actor, a Polymorph, a creature with no transformation, an ineligible new form and a dead target are each refused; a gear choice for Animal Shapes is invalid input.
// Path: unhappy (state error).
func TestShapeChangingSpellsChangeFormScenario(t *testing.T) {
	t.Parallel()
	ctx, druid, ally := context.Background(), combat.MustCombatantID("druid"), combat.MustCombatantID("ally")
	rating := grpAMust(combat.NewRatedByLevel(5))
	tracked := combat.TrackedHitPoints{Status: grpAHealthy(), Choice: combat.AcceptNew}
	begun := grpABegin(t, combat.BeginShapeShiftCommand{Spell: combat.AnimalShapesSpell, Caster: druid, Target: ally, Form: grpABeast(t, content.SizeLarge, "2"),
		Response: combat.Willing{}, Rating: rating, HitPoints: tracked}).(combat.BeginShapeShiftApplied)
	polymorphed := grpABegin(t, combat.BeginShapeShiftCommand{Spell: combat.PolymorphSpell, Caster: druid, Target: ally, Form: grpABeast(t, content.SizeMedium, "1"),
		Response: combat.Willing{}, Rating: rating, HitPoints: tracked}).(combat.BeginShapeShiftApplied)

	change := func(c combat.ChangeFormCommand) combat.ChangeShapeShiftFormResult {
		t.Helper()
		res, err := combat.NewChangeShapeShiftForm().Run(ctx, combat.ChangeShapeShiftFormCommand(c))
		if err != nil {
			t.Fatalf("ChangeShapeShiftForm: %v", err)
		}
		return res
	}
	good := combat.ChangeFormCommand{Actor: druid, State: begun.Transformation(), Form: grpABeast(t, content.SizeMedium, "1"), Rating: rating}
	changed, ok := change(good).(combat.ChangeFormApplied)
	if !ok || changed.ActionCost() != combat.ShapeShiftMagicAction || changed.Transformation().Form() == begun.Transformation().Form() ||
		changed.Transformation().TemporaryHitPoints() != begun.Transformation().TemporaryHitPoints() || changed.Transformation().MaintainedBy() != begun.Transformation().MaintainedBy() ||
		changed.Equipment() != combat.MeldsUnusable || len(changed.Events()) != 1 || changed.Events()[0].Kind() != "shape_shift_form_changed" {
		t.Fatalf("a new eligible form = %#v, want applied: a Magic action, a new form, one shape_shift_form_changed, everything else unchanged", changed)
	}
	with := func(f func(*combat.ChangeFormCommand)) combat.ChangeFormCommand {
		c := good
		f(&c)
		return c
	}
	for name, c := range map[string]struct {
		cmd  combat.ChangeFormCommand
		want string
	}{
		"another actor":                  {with(func(c *combat.ChangeFormCommand) { c.Actor = ally }), "actor_not_caster"},
		"a Polymorph cannot change form": {with(func(c *combat.ChangeFormCommand) { c.State = polymorphed.Transformation() }), "form_change_not_allowed"},
		"no transformation":              {with(func(c *combat.ChangeFormCommand) { c.State = combat.ActiveTransformation{} }), "not_transformed"},
		"a Challenge Rating 9 form":      {with(func(c *combat.ChangeFormCommand) { c.Form = grpABeast(t, content.SizeMedium, "9") }), "form_ineligible"},
		"a dead target":                  {with(func(c *combat.ChangeFormCommand) { c.TargetLife = grpAMust(combat.NewDead(combat.MassiveDamage)) }), "target_is_dead"},
	} {
		if r, ok := change(c.cmd).(combat.ChangeFormRefused); !ok || r.Rejection.String() != c.want {
			t.Errorf("%s: result = %#v, want refused %s", name, r, c.want)
		}
	}
	if _, err := combat.NewChangeShapeShiftForm().Run(ctx, combat.ChangeShapeShiftFormCommand(with(func(c *combat.ChangeFormCommand) { c.Equipment = combat.DropsToGround }))); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a gear choice for Animal Shapes err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-032, FR-033, FR-034, FR-043 to FR-053 - a Polymorphed hero takes 15 damage: the form's 12 Temporary Hit Points are gone, the spell ends (Hit Points kept as reduced), and ending it clears the transformation; Concentration ending of another creature does not touch it, the caster's does; a healthy hero's spell continues; Animal Shapes can be ended at will by the target (a Bonus Action) and Polymorph cannot; a dead target accepts only its death as the cause; with no transformation nothing happens.
// Path: unhappy (rule interaction).
func TestShapeChangingSpellsAssessAndEndScenario(t *testing.T) {
	t.Parallel()
	ctx, druid, hero := context.Background(), combat.MustCombatantID("druid"), combat.MustCombatantID("hero")
	rating := grpAMust(combat.NewRatedByLevel(5))
	tracked := combat.TrackedHitPoints{Status: grpAHealthy(), Choice: combat.AcceptNew}
	polymorph := grpABegin(t, combat.BeginShapeShiftCommand{Spell: combat.PolymorphSpell, Caster: druid, Target: hero, Form: grpABeast(t, content.SizeMedium, "1"),
		Response: combat.Unwilling{Save: d20.Failure}, Rating: rating, HitPoints: tracked}).(combat.BeginShapeShiftApplied)
	animal := grpABegin(t, combat.BeginShapeShiftCommand{Spell: combat.AnimalShapesSpell, Caster: druid, Target: hero, Form: grpABeast(t, content.SizeLarge, "2"),
		Response: combat.Willing{}, Rating: rating, HitPoints: tracked}).(combat.BeginShapeShiftApplied)

	assess := func(state combat.ActiveTransformation, status combat.HitPointStatus) combat.AssessShapeShiftResult {
		t.Helper()
		res, err := combat.NewAssessShapeShift().Run(ctx, combat.AssessShapeShiftCommand{State: state, Status: status})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	end := func(c combat.EndShapeShiftCommand) combat.EndShapeShiftingResult {
		t.Helper()
		res, err := combat.NewEndShapeShifting().Run(ctx, combat.EndShapeShiftingCommand(c))
		if err != nil {
			t.Fatalf("EndShapeShifting: %v", err)
		}
		return res
	}

	if got := assess(polymorph.Transformation(), polymorph.Status()); got != combat.AssessShapeShiftResult(combat.AssessShapeShiftContinues{}) {
		t.Errorf("a hero with its form's Temporary Hit Points = %#v, want the spell to continue", got)
	}
	damage, err := polymorph.Status().ApplyDamage(hero, []combat.Damage{combat.MustDamage(15, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	hurt := damage.(combat.HitPointTransitionApplied).Status()
	depleted, ok := assess(polymorph.Transformation(), hurt).(combat.AssessShapeShiftEnded)
	if !ok || depleted.Cause() != combat.CauseFormHitPointsDepleted || hurt.HitPoints().Current() != 27 || hurt.HitPoints().Temporary() != 0 {
		t.Fatalf("after 15 damage = %#v with %d Hit Points, want ended by depleted form Hit Points at 27", depleted, hurt.HitPoints().Current())
	}
	ended, ok := end(combat.EndShapeShiftCommand{State: polymorph.Transformation(), Cause: depleted.Cause(), HitPoints: combat.TrackedHitPoints{Status: hurt}}).(combat.EndShapeShiftApplied)
	if !ok || !ended.Transformation().IsZero() || ended.Status() != hurt || len(ended.Events()) == 0 || ended.Events()[len(ended.Events())-1].Kind() != "shape_shift_ended" {
		t.Fatalf("ending = %#v, want the transformation cleared, the status unchanged and shape_shift_ended last", ended)
	}
	if _, unaffected := end(combat.EndShapeShiftCommand{State: polymorph.Transformation(), Cause: combat.CauseConcentrationEnded,
		HitPoints: combat.TrackedHitPoints{Status: polymorph.Status()}, ConcentrationOf: hero}).(combat.EndShapeShiftUnaffected); !unaffected {
		t.Errorf("another creature's Concentration ending must not end the Polymorph")
	}
	if got, ok := end(combat.EndShapeShiftCommand{State: polymorph.Transformation(), Cause: combat.CauseConcentrationEnded,
		HitPoints: combat.TrackedHitPoints{Status: polymorph.Status()}, ConcentrationOf: druid}).(combat.EndShapeShiftApplied); !ok || got.Status().HitPoints().Temporary() != 0 {
		t.Errorf("the caster's Concentration ending = %#v, want the Polymorph ended and its Temporary Hit Points gone", got)
	}
	if got, ok := end(combat.EndShapeShiftCommand{State: polymorph.Transformation(), Cause: combat.CauseEndedByTarget, HitPoints: combat.TrackedHitPoints{Status: polymorph.Status()}}).(combat.EndShapeShiftRefused); !ok || got.Rejection != combat.CannotEndAtWill {
		t.Errorf("a Polymorph ended by its target = %#v, want refused CannotEndAtWill", got)
	}
	willing, ok := end(combat.EndShapeShiftCommand{State: animal.Transformation(), Cause: combat.CauseEndedByTarget, HitPoints: combat.TrackedHitPoints{Status: animal.Status()}}).(combat.EndShapeShiftApplied)
	if cost, has := willing.ActionCost(); !ok || !has || cost != combat.ShapeShiftBonusAction {
		t.Errorf("Animal Shapes ended by its target = %#v, want applied for a Bonus Action", willing)
	}
	if _, unaffected := end(combat.EndShapeShiftCommand{State: animal.Transformation(), Cause: combat.CauseConcentrationEnded,
		HitPoints: combat.TrackedHitPoints{Status: animal.Status()}, ConcentrationOf: druid}).(combat.EndShapeShiftUnaffected); !unaffected {
		t.Errorf("Animal Shapes needs no Concentration, so its end must be unaffected")
	}
	deadHP := combat.TrackedHitPoints{Status: grpADead(t)}
	if got, ok := end(combat.EndShapeShiftCommand{State: animal.Transformation(), Cause: combat.CauseDurationElapsed, HitPoints: deadHP}).(combat.EndShapeShiftRefused); !ok || got.Rejection != combat.EndTargetIsDead {
		t.Errorf("an elapsed duration on a dead target = %#v, want refused EndTargetIsDead", got)
	}
	if got, ok := end(combat.EndShapeShiftCommand{State: animal.Transformation(), Cause: combat.CauseTargetDied, HitPoints: deadHP}).(combat.EndShapeShiftApplied); !ok || !got.Transformation().IsZero() {
		t.Errorf("a dead target's own death = %#v, want the transformation ended", got)
	}
	if _, unaffected := end(combat.EndShapeShiftCommand{State: combat.ActiveTransformation{}, Cause: combat.CauseDispelled, HitPoints: tracked}).(combat.EndShapeShiftUnaffected); !unaffected {
		t.Errorf("ending nothing must be unaffected")
	}
	if _, err := combat.NewEndShapeShifting().Run(ctx, combat.EndShapeShiftingCommand{State: animal.Transformation(), Cause: 0, HitPoints: tracked}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined cause err = %v, want ErrInvalidInput", err)
	}
}
