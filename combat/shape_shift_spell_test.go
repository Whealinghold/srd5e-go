package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// Requirement: doc/specs/shape-changing-spells.md FR-001 - the four spells map to and from their content ids; any other id gives the zero value and false (SRD 5.2.1, "Polymorph", p. 153; "True Polymorph", p. 170; "Shapechange", p. 161; "Animal Shapes", p. 108).
// Path: per case.
func TestShapeShiftSpellOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		id                      content.SpellID
		want                    combat.ShapeShiftSpell
		found                   bool
	}{
		{"polymorph", "FR-001", "happy", "polymorph", combat.PolymorphSpell, true},
		{"true polymorph", "FR-001", "happy", "true_polymorph", combat.TruePolymorphSpell, true},
		{"shapechange", "FR-001", "happy", "shapechange", combat.ShapechangeSpell, true},
		{"animal shapes", "FR-001", "happy", "animal_shapes", combat.AnimalShapesSpell, true},
		{"another spell", "FR-001", "unhappy (invalid input)", "fireball", 0, false},
		{"empty id", "FR-001", "unhappy (empty and minimal)", "", 0, false},
		{"case differs", "FR-001", "unhappy (invalid input)", "Polymorph", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := combat.ShapeShiftSpellOf(c.id)
			if got != c.want || ok != c.found {
				t.Errorf("ShapeShiftSpellOf(%q) = %v, %v, want %v, %v [%s]", c.id, got, ok, c.want, c.found, c.requirement)
			}
			if ok && got.SpellID() != c.id {
				t.Errorf("SpellID() = %q, want %q", got.SpellID(), c.id)
			}
		})
	}
	if id := combat.ShapeShiftSpell(0).SpellID(); id != "" {
		t.Errorf("the zero spell has the id %q", id)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-002 and table A - each spell exposes the profile of its row.
// Path: per case.
func TestShapeShiftProfile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spell                   combat.ShapeShiftSpell
		want                    combat.ShapeShiftProfile
	}{
		{"polymorph", "FR-002 table A, p. 153", "happy", combat.PolymorphSpell, combat.ShapeShiftProfile{Level: 4, Concentration: true, MaximumMinutes: 60,
			TargetSaves: true, GrantsTemporaryHitPoints: true, EndsWhenTemporaryHitPointsGone: true}},
		{"true polymorph", "FR-002 table A, p. 170", "happy", combat.TruePolymorphSpell, combat.ShapeShiftProfile{Level: 9, Concentration: true, MaximumMinutes: 60,
			TargetSaves: true, GrantsTemporaryHitPoints: true, PermanentIfHeld: true}},
		{"shapechange", "FR-002 table A, p. 161", "happy", combat.ShapechangeSpell, combat.ShapeShiftProfile{Level: 9, Concentration: true, MaximumMinutes: 60,
			GrantsTemporaryHitPoints: true, CanChangeForm: true}},
		{"animal shapes", "FR-002 table A, p. 108", "happy", combat.AnimalShapesSpell, combat.ShapeShiftProfile{Level: 8, MaximumMinutes: 1440,
			GrantsTemporaryHitPoints: true, CanChangeForm: true}},
		{"the zero spell", "FR-002", "unhappy (invalid input)", 0, combat.ShapeShiftProfile{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.spell.Profile(); got != c.want {
				t.Errorf("%v.Profile() = %+v, want %+v [%s]", c.spell, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-002 - the profile's level, Concentration and duration equal the bundled Spell entry of the same id (the oracle). Only Polymorph is in the bundled slice; the other three are D (see the spec's Not yet implemented line).
// Path: happy.
func TestShapeShiftProfileMatchesTheBundledSpell(t *testing.T) {
	t.Parallel()
	src := data.MustBundledSource()
	for _, spell := range []combat.ShapeShiftSpell{combat.PolymorphSpell} {
		res, err := src.Lookup(t.Context(), spell.SpellID().Key())
		if err != nil {
			t.Fatal(err)
		}
		found, ok := res.(content.LookupFound)
		if !ok {
			t.Fatalf("the bundled slice lacks %s", spell.SpellID())
		}
		entry := found.Entry.(content.Spell)
		p := spell.Profile()
		timed, isTimed := entry.Duration().(content.SpellDurationTimed)
		if !isTimed || timed.Unit() != content.DurationUnitHours || timed.Amount()*60 != p.MaximumMinutes {
			t.Errorf("%s duration %#v, want %d minutes", spell, entry.Duration(), p.MaximumMinutes)
		}
		if int(entry.Level()) != p.Level || entry.RequiresConcentration() != p.Concentration {
			t.Errorf("%s: bundled level %d, Concentration %v; profile %d, %v", spell, entry.Level(), entry.RequiresConcentration(), p.Level, p.Concentration)
		}
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-043 and section 3 - the enums of the package report snake_case names and reject the zero value.
// Path: per case.
func TestShapeShiftEnumNames(t *testing.T) {
	t.Parallel()
	if got := combat.ShapeShiftSpell(0).String(); got != "unknown" {
		t.Errorf("the zero spell is %q", got)
	}
	causes := []string{"duration_elapsed", "concentration_ended", "form_hit_points_depleted", "target_died", "target_destroyed", "ended_by_target", "dispelled", "replaced"}
	for i, want := range causes {
		c := combat.ShapeShiftEndCause(i + 1)
		if !c.Valid() || c.String() != want {
			t.Errorf("cause %d = %q (valid %v), want %q", i+1, c, c.Valid(), want)
		}
	}
	if combat.ShapeShiftEndCause(0).Valid() || combat.ShapeShiftEndCause(9).Valid() {
		t.Error("an out of range cause is valid")
	}
	for _, e := range []combat.EquipmentOutcome{combat.MeldsUnusable, combat.DropsToGround, combat.ResizesToFit, combat.TransformsWithCreature} {
		if !e.Valid() || e.String() == "unknown" {
			t.Errorf("equipment outcome %d is not valid", e)
		}
	}
	for _, p := range []combat.FormTemporaryHitPoints{combat.TemporaryFromForm, combat.TemporaryNotGranted, combat.TemporaryDeclined, combat.TemporaryReplaced} {
		if !p.Valid() || p.String() == "unknown" {
			t.Errorf("provenance %d is not valid", p)
		}
	}
	for _, m := range []combat.TrueTransformation{combat.CreatureIntoCreature, combat.ObjectIntoCreature, combat.CreatureIntoObject} {
		if !m.Valid() || m.String() == "unknown" {
			t.Errorf("mode %d is not valid", m)
		}
	}
	if combat.ShapeShiftMagicAction.String() != "magic_action" || combat.ShapeShiftBonusAction.String() != "bonus_action" || combat.ShapeShiftActionCost(0).String() != "unknown" {
		t.Error("action cost names")
	}
	if !combat.StandardShapechange.Valid() || !combat.StatBlockShapechange.Valid() || combat.ShapechangeTerms(2).Valid() {
		t.Error("ShapechangeTerms validity")
	}
	if !combat.MaintainedByConcentration.Valid() || !combat.NotMaintained.Valid() || combat.MaintainedBy(0).Valid() {
		t.Error("MaintainedBy validity")
	}
}
