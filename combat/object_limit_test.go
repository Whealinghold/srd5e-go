package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/environment-hazards-and-objects.md Limits (plan P10) - a composite object holds at most 256 sections.
// Path: unhappy (boundary).
func TestNewCompositeObjectCap(t *testing.T) {
	t.Parallel()
	live := envObject(t, 10, 0, combat.NoAffinities())
	runCap(t, "composite sections", 256, func(n int) error {
		sections := make([]combat.CompositeSection, n)
		for i := range sections {
			sections[i] = combat.CompositeSection{Status: live, Ruling: combat.DoesNotCollapse}
		}
		_, err := combat.NewCompositeObject(sections)
		return err
	})
}

// Requirement: doc/specs/environment-hazards-and-objects.md Limits (plan P10) - ReplaceSection works on a composite at the 256-section cap, the last index included, and leaves the receiver unchanged; a composite over the cap cannot be built, so the replace has no cap plus one.
// Path: unhappy (boundary).
func TestReplaceSectionAtTheCap(t *testing.T) {
	t.Parallel()
	live := envObject(t, 10, 0, combat.NoAffinities())
	sections := make([]combat.CompositeSection, 256)
	for i := range sections {
		sections[i] = combat.CompositeSection{Status: live, Ruling: combat.DoesNotCollapse}
	}
	c, err := combat.NewCompositeObject(sections)
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := c.ReplaceSection(255, envObject(t, 20, 0, combat.NoAffinities()))
	if err != nil || len(next.Sections()) != 256 || next.Sections()[255].Status.HitPoints().Current() != 20 {
		t.Fatalf("ReplaceSection(255) = %d sections, %v", len(next.Sections()), err)
	}
	if c.Sections()[255].Status.HitPoints().Current() != 10 {
		t.Error("the receiver changed")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md Limits (plan P10) - an object damage instance holds at most 64 damages and at most 64 adjustments; ApplyDamageToObject therefore resolves at most 64 Damage and emits at most two events.
// Path: unhappy (boundary).
func TestNewObjectDamageInstanceCaps(t *testing.T) {
	t.Parallel()
	t.Run("damages", func(t *testing.T) {
		t.Parallel()
		runCap(t, "object damages", 64, func(n int) error {
			ds := make([]combat.Damage, n)
			for i := range ds {
				ds[i] = slash(1)
			}
			_, err := combat.NewObjectDamageInstance(ds, nil)
			return err
		})
	})
	t.Run("adjustments", func(t *testing.T) {
		t.Parallel()
		runCap(t, "object adjustments", 64, func(n int) error {
			adj := make([]combat.DamageAdjustment, n)
			for i := range adj {
				adj[i] = combat.FlatAdjustment{}
			}
			_, err := combat.NewObjectDamageInstance([]combat.Damage{slash(1)}, adj)
			return err
		})
	})
}

// Requirement: doc/specs/environment-hazards-and-objects.md Limits (plan P10) - an instance at both caps (64 damages, 64 adjustments) is accepted by ApplyDamageToObject and resolves each damage once.
// Path: unhappy (boundary).
func TestApplyDamageToObjectAtTheCaps(t *testing.T) {
	t.Parallel()
	ds := make([]combat.Damage, 64)
	adj := make([]combat.DamageAdjustment, 64)
	for i := range ds {
		ds[i], adj[i] = slash(1), combat.FlatAdjustment{}
	}
	in, err := combat.NewObjectDamageInstance(ds, adj)
	if err != nil {
		t.Fatal(err)
	}
	res := damageObject(t, envObject(t, 30, 0, combat.NoAffinities()), in).(combat.ObjectDamageApplied)
	if got := len(res.Resolutions()); got != 64 {
		t.Errorf("resolutions = %d, want 64", got)
	}
	if got := len(res.Events()); got > 2 {
		t.Errorf("events = %d, want at most 2", got)
	}
}
