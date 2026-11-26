package combat_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func rrCharmed() combat.Charmed {
	return combat.Charmed{Charmer: combat.CharmedByCombatant{ID: gid("hag")}}
}

func rrExhaustion(n int) combat.ExhaustionCondition {
	return combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(n)}
}

// rrGreater is a Greater Restoration command with a baseline creature: Alive, 12 of 12 Hit Points, no conditions and the effects of table 4.2.
func rrGreater(tb testing.TB, choice combat.GreaterRestorationChoice, set combat.ConditionSet) combat.GreaterRestorationCommand {
	tb.Helper()
	return combat.GreaterRestorationCommand{Subject: gid("hero"), Choice: choice, Conditions: set, Effects: rrEffects(), Status: pcStatus(tb, 12, 12, 0)}
}

// Requirement: per case (table 4.7b, Greater Restoration; Source: SRD 5.2.1, "Greater Restoration" p. 137)
// Path: per case.
func TestApplyGreaterRestorationConditions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		choice                  combat.GreaterRestorationChoice
		set                     combat.ConditionSet
		applied                 bool
		left                    []content.ConditionKind
		exhaustion              int
	}{
		{"Exhaustion 3", "FR-034; table 4.7b", "happy", combat.EndOneExhaustionLevel{}, meConditions(rrExhaustion(3)), true, []content.ConditionKind{content.ConditionExhaustion}, 2},
		{"Exhaustion 1", "FR-034; table 4.7b", "unhappy (boundary)", combat.EndOneExhaustionLevel{}, meConditions(rrExhaustion(1)), true, nil, 0},
		{"Exhaustion 6", "FR-034", "unhappy (boundary)", combat.EndOneExhaustionLevel{}, meConditions(rrExhaustion(6)), true, []content.ConditionKind{content.ConditionExhaustion}, 5},
		{"no Exhaustion", "FR-034; table 4.7b", "unhappy (empty and minimal)", combat.EndOneExhaustionLevel{}, meConditions(combat.Blinded{}), false, []content.ConditionKind{content.ConditionBlinded}, 0},
		{"Charmed", "FR-034; table 4.7b", "happy", combat.EndCharmed{}, meConditions(rrCharmed(), combat.Blinded{}), true, []content.ConditionKind{content.ConditionBlinded}, 0},
		{"Petrified", "FR-034; table 4.7b", "happy", combat.EndPetrified{}, meConditions(combat.Petrified{}), true, nil, 0},
		{"Petrified with Paralyzed keeps Paralyzed and its implied Incapacitated", "FR-034; table 4.7b", "unhappy (rule interaction)", combat.EndPetrified{},
			meConditions(combat.Petrified{}, combat.Paralyzed{}), true, []content.ConditionKind{content.ConditionParalyzed}, 0},
		{"Charmed not held", "FR-034; table 4.7b", "unhappy (state error)", combat.EndCharmed{}, meConditions(combat.Blinded{}), false, []content.ConditionKind{content.ConditionBlinded}, 0},
		{"Petrified not held", "FR-034; table 4.7b", "unhappy (state error)", combat.EndPetrified{}, combat.ConditionSet{}, false, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			cmd := rrGreater(t, c.choice, c.set)
			before := rrGreater(t, c.choice, c.set)
			res := combat.ApplyGreaterRestoration(cmd)
			if !reflect.DeepEqual(cmd, before) {
				t.Error("the command changed")
			}
			if !c.applied {
				if res != (combat.GreaterRestorationNothingToEnd{}) {
					t.Errorf("result = %#v, want GreaterRestorationNothingToEnd [%s; %s]", res, c.requirement, c.path)
				}
				return
			}
			a, ok := res.(combat.GreaterRestorationApplied)
			if !ok {
				t.Fatalf("result = %#v, want applied [%s; %s]", res, c.requirement, c.path)
			}
			if kinds := rrConditionKinds(a.Conditions()); !slices.Equal(slices.Sorted(slices.Values(kinds)), slices.Sorted(slices.Values(c.left))) {
				t.Errorf("conditions left = %v, want %v [%s]", kinds, c.left, c.requirement)
			}
			for _, cond := range a.Conditions().Explicit() {
				if e, ok := cond.(combat.ExhaustionCondition); ok && e.Level.Value() != c.exhaustion {
					t.Errorf("Exhaustion level = %d, want %d", e.Level.Value(), c.exhaustion)
				}
			}
			if len(a.Events()) != 1 || !a.CurseSource().IsZero() || !a.Effects().Equal(cmd.Effects) || !reflect.DeepEqual(a.Status(), cmd.Status) {
				t.Errorf("events %v, curse source %v: want one event, no curse source and the rest unchanged", rrKinds(a.Events()), a.CurseSource())
			}
		})
	}
}

// Requirement: per case (FR-035, table 4.7b: a curse ends through GreaterRestorationSpell and its EffectSource is carried; SRD 5.2.1, "Greater Restoration" p. 137, "Curses" p. 179)
// Path: per case.
func TestApplyGreaterRestorationCurse(t *testing.T) {
	t.Parallel()
	itemCurse := meEffect("c3", combat.OngoingEffectCurse, meItem("cursed_blade"))
	effects := meEffects(meCurse("c1"), meEffect("s1", combat.OngoingEffectSpell, meSpell("bless")), itemCurse)
	run := func(id combat.OngoingEffectID) combat.CastGreaterRestorationResult {
		cmd := rrGreater(t, combat.EndCurse{ID: id}, combat.ConditionSet{})
		cmd.Effects = effects
		return combat.ApplyGreaterRestoration(cmd)
	}
	a, ok := run(meEID("c1")).(combat.GreaterRestorationApplied)
	if !ok || !slices.Equal(rrEffectIDs(a.Effects()), []string{"s1", "c3"}) || a.CurseSource() != meLabeled("hag_curse") || !slices.Equal(rrKinds(a.Events()), []string{"curse_removed"}) {
		t.Errorf("EndCurse(c1) = %#v, want c1 gone, its labeled source and one curse_removed", a)
	}
	a, ok = run(meEID("c3")).(combat.GreaterRestorationApplied)
	if !ok || a.CurseSource() != meItem("cursed_blade") || a.CurseSource().Kind() != combat.EffectSourceMagicItem {
		t.Errorf("EndCurse(c3) = %#v, want the MagicItemSource carried so the host can end Attunement", a)
	}
	if res := run(meEID("s1")); res != (combat.GreaterRestorationNotACurse{}) {
		t.Errorf("EndCurse(s1) = %#v, want GreaterRestorationNotACurse", res)
	}
	if res := run(meEID("nope")); res != (combat.GreaterRestorationNothingToEnd{}) {
		t.Errorf("EndCurse(an id not held) = %#v, want GreaterRestorationNothingToEnd", res)
	}
	if res := run(""); res != (combat.GreaterRestorationNothingToEnd{}) {
		t.Errorf("EndCurse(a blank id) = %#v, want GreaterRestorationNothingToEnd", res)
	}
}

// Requirement: per case (FR-036, table 4.7b and FR-031: ending a Hit Point maximum reduction is RestoreMaximum; an ability score reduction is the host's; SRD 5.2.1, "Greater Restoration" p. 137)
// Path: per case.
func TestApplyGreaterRestorationMaximumAndScores(t *testing.T) {
	t.Parallel()
	cmd := rrGreater(t, combat.EndAbilityScoreReduction{}, combat.ConditionSet{})
	if res := combat.ApplyGreaterRestoration(cmd); res != (combat.GreaterRestorationHostResolved{}) {
		t.Errorf("EndAbilityScoreReduction = %#v, want GreaterRestorationHostResolved", res)
	}
	cmd = rrGreater(t, combat.EndHitPointMaximumReduction{Amount: 8}, combat.ConditionSet{})
	a, ok := combat.ApplyGreaterRestoration(cmd).(combat.GreaterRestorationApplied)
	if !ok || a.Status().HitPoints().Maximum() != 20 || a.Status().HitPoints().Current() != 12 || !slices.Equal(rrKinds(a.Events()), []string{"hit_point_maximum_restored"}) {
		t.Errorf("EndHitPointMaximumReduction(8) = %#v, want maximum 20, current 12 and hit_point_maximum_restored", a)
	}
	for _, amount := range []int{0} {
		cmd = rrGreater(t, combat.EndHitPointMaximumReduction{Amount: amount}, combat.ConditionSet{})
		if res := combat.ApplyGreaterRestoration(cmd); res != (combat.GreaterRestorationNothingToEnd{}) {
			t.Errorf("EndHitPointMaximumReduction(%d) = %#v, want GreaterRestorationNothingToEnd", amount, res)
		}
	}
	for _, amount := range []int{-1, math.MinInt} {
		cmd = rrGreater(t, combat.EndHitPointMaximumReduction{Amount: amount}, combat.ConditionSet{})
		if res := combat.ApplyGreaterRestoration(cmd); res != (combat.GreaterRestorationInvalid{}) {
			t.Errorf("EndHitPointMaximumReduction(%d) = %#v, want GreaterRestorationInvalid", amount, res)
		}
	}
	cmd = rrGreater(t, combat.EndHitPointMaximumReduction{Amount: 8}, combat.ConditionSet{})
	cmd.Status = deadStatus(t, combat.MassiveDamage)
	if res := combat.ApplyGreaterRestoration(cmd); res != (combat.GreaterRestorationRejected{Rejection: combat.CreatureIsDead}) {
		t.Errorf("a Dead creature = %#v, want GreaterRestorationRejected (CreatureIsDead)", res)
	}
	cmd.Status = combat.HitPointStatus{}
	if res := combat.ApplyGreaterRestoration(cmd); res != (combat.GreaterRestorationInvalid{}) {
		t.Errorf("the zero status = %#v, want GreaterRestorationInvalid", res)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-033 - exactly one choice per cast: no choice and a zero subject are GreaterRestorationInvalid, nothing changes, and the same
// command gives an equal result each call.
// Path: unhappy (invalid input).
func TestApplyGreaterRestorationInvalid(t *testing.T) {
	t.Parallel()
	set := meConditions(rrCharmed())
	if res := combat.ApplyGreaterRestoration(rrGreater(t, nil, set)); res != (combat.GreaterRestorationInvalid{}) {
		t.Errorf("a nil choice = %#v, want GreaterRestorationInvalid", res)
	}
	cmd := rrGreater(t, combat.EndCharmed{}, set)
	cmd.Subject = ""
	if res := combat.ApplyGreaterRestoration(cmd); res != (combat.GreaterRestorationInvalid{}) {
		t.Errorf("a zero subject = %#v, want GreaterRestorationInvalid", res)
	}
	cmd.Subject = gid("hero")
	a, b := combat.ApplyGreaterRestoration(cmd), combat.ApplyGreaterRestoration(cmd)
	if !reflect.DeepEqual(a, b) {
		t.Error("equal commands gave unequal results")
	}
	ev := a.(combat.GreaterRestorationApplied).Events()
	ev[0] = nil
	if a.(combat.GreaterRestorationApplied).Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}
