package combat_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rrHealed runs ApplyRestorativeHealing and returns the applied result or fails the test.
func rrHealed(tb testing.TB, spell combat.RestorativeSpell, st combat.HitPointStatus, set combat.ConditionSet, amount, level int) combat.RestorativeHealingApplied {
	tb.Helper()
	res, err := combat.ApplyRestorativeHealing(spell, gid("hero"), st, set, amount, level)
	if err != nil {
		tb.Fatalf("ApplyRestorativeHealing: %v", err)
	}
	a, ok := res.(combat.RestorativeHealingApplied)
	if !ok {
		tb.Fatalf("ApplyRestorativeHealing = %T, want RestorativeHealingApplied", res)
	}
	return a
}

// rrRemovedKinds lists the condition kinds of the ConditionRemoved events, in order.
func rrRemovedKinds(events []combat.Event) []content.ConditionKind {
	out := []content.ConditionKind{}
	for _, e := range events {
		if r, ok := e.(combat.ConditionRemoved); ok {
			out = append(out, r.ConditionKind())
		}
	}
	return out
}

// Requirement: per case (table 4.7b, the Heal family; Source: SRD 5.2.1, "Heal" p. 139, "Power Word Heal" p. 154, "Mass Heal" p. 147)
// Path: per case.
func TestApplyRestorativeHealing(t *testing.T) {
	t.Parallel()
	all := meConditions(combat.Poisoned{}, combat.Blinded{}, combat.Deafened{}, rrCharmed(), combat.Stunned{}, combat.Prone{})
	cases := []struct {
		name, requirement, path string
		spell                   combat.RestorativeSpell
		status                  combat.HitPointStatus
		set                     combat.ConditionSet
		amount, level           int
		current                 int
		removed                 []content.ConditionKind
		kept                    []content.ConditionKind
		events                  []string
	}{
		{"Heal at 6", "FR-037, FR-038; table 4.7b", "happy", combat.Heal, pcStatus(t, 5, 100, 0), all, 70, 6, 75,
			[]content.ConditionKind{content.ConditionBlinded, content.ConditionDeafened, content.ConditionPoisoned},
			[]content.ConditionKind{content.ConditionCharmed, content.ConditionStunned, content.ConditionProne},
			[]string{"hit_points_restored", "condition_removed", "condition_removed", "condition_removed"}},
		{"Heal at 7", "FR-038; table 4.7b", "unhappy (boundary)", combat.Heal, pcStatus(t, 1, 500, 0), combat.ConditionSet{}, 80, 7, 81, nil, nil, []string{"hit_points_restored"}},
		{"Heal at 8", "FR-038; table 4.7b", "unhappy (boundary)", combat.Heal, pcStatus(t, 1, 500, 0), combat.ConditionSet{}, 90, 8, 91, nil, nil, []string{"hit_points_restored"}},
		{"Heal at 9", "FR-038; table 4.7b", "unhappy (boundary)", combat.Heal, pcStatus(t, 1, 500, 0), combat.ConditionSet{}, 100, 9, 101, nil, nil, []string{"hit_points_restored"}},
		{"Heal at full", "FR-037; table 4.7b", "unhappy (boundary)", combat.Heal, pcStatus(t, 40, 40, 0), meConditions(combat.Poisoned{}), 70, 6, 40,
			[]content.ConditionKind{content.ConditionPoisoned}, nil, []string{"condition_removed"}},
		{"Heal, none held", "FR-037; table 4.7b", "unhappy (empty and minimal)", combat.Heal, pcStatus(t, 10, 40, 0), combat.ConditionSet{}, 70, 6, 40, nil, nil, []string{"hit_points_restored"}},
		{"Heal on a Dying creature", "FR-039; table 4.7b", "unhappy (rule interaction)", combat.Heal, dyingStatus(t, 40, 1, 1), meConditions(combat.Unconscious{}), 70, 6, 40, nil,
			[]content.ConditionKind{content.ConditionUnconscious}, []string{"hit_points_restored", "recovered"}},
		{"Heal keeps Temporary Hit Points", "FR-037", "unhappy (rule interaction)", combat.Heal, pcStatus(t, 10, 40, 5), combat.ConditionSet{}, 70, 6, 40, nil, nil, []string{"hit_points_restored"}},
		{"Power Word Heal", "FR-037, FR-038; table 4.7b", "happy", combat.PowerWordHeal, pcStatus(t, 3, 40, 0), meConditions(combat.Stunned{}, combat.Prone{}), 0, 0, 40,
			[]content.ConditionKind{content.ConditionStunned}, []content.ConditionKind{content.ConditionProne}, []string{"hit_points_restored", "condition_removed"}},
		{"Power Word Heal ignores the amount", "FR-038", "unhappy (rule interaction)", combat.PowerWordHeal, pcStatus(t, 3, 40, 0), combat.ConditionSet{}, 5, 0, 40, nil, nil, []string{"hit_points_restored"}},
		{"Power Word Heal, every condition of its list in order", "FR-037", "unhappy (rule interaction)", combat.PowerWordHeal, pcStatus(t, 3, 40, 0),
			meConditions(combat.Stunned{}, combat.Poisoned{}, combat.Paralyzed{}, combat.Frightened{Source: gid("x")}, rrCharmed(), combat.Blinded{}), 0, 0, 40,
			[]content.ConditionKind{content.ConditionCharmed, content.ConditionFrightened, content.ConditionParalyzed, content.ConditionPoisoned, content.ConditionStunned},
			[]content.ConditionKind{content.ConditionBlinded}, []string{"hit_points_restored", "condition_removed", "condition_removed", "condition_removed", "condition_removed", "condition_removed"}},
		{"Mass Heal for a creature allotted 30", "FR-037, FR-040", "happy", combat.MassHeal, pcStatus(t, 3, 40, 0), meConditions(combat.Blinded{}, combat.Poisoned{}, combat.Stunned{}), 30, 0, 33,
			[]content.ConditionKind{content.ConditionBlinded, content.ConditionPoisoned}, []content.ConditionKind{content.ConditionStunned}, []string{"hit_points_restored", "condition_removed", "condition_removed"}},
		{"Mass Heal, allotted 0 is not healed", "FR-040; table 4.7b", "unhappy (boundary)", combat.MassHeal, pcStatus(t, 3, 40, 0), meConditions(combat.Blinded{}), 0, 0, 3,
			nil, []content.ConditionKind{content.ConditionBlinded}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			a := rrHealed(t, c.spell, c.status, c.set, c.amount, c.level)
			if a.Status().HitPoints().Current() != c.current || a.Status().HitPoints().Maximum() != c.status.HitPoints().Maximum() ||
				a.Status().HitPoints().Temporary() != c.status.HitPoints().Temporary() {
				t.Errorf("Hit Points = %+v, want %d of %d [%s; %s]", a.Status().HitPoints(), c.current, c.status.HitPoints().Maximum(), c.requirement, c.path)
			}
			if _, alive := a.Status().LifeState().(combat.Alive); !alive {
				t.Errorf("life state = %T, want Alive", a.Status().LifeState())
			}
			if got := rrRemovedKinds(a.Events()); !slices.Equal(got, append([]content.ConditionKind{}, c.removed...)) {
				t.Errorf("conditions ended = %v, want %v in that order [%s]", got, c.removed, c.requirement)
			}
			held := rrConditionKinds(a.Conditions())
			for _, k := range c.kept {
				if !slices.Contains(held, k) {
					t.Errorf("%v was removed, want it kept [%s]", k, c.requirement)
				}
			}
			for _, k := range c.removed {
				if slices.Contains(held, k) {
					t.Errorf("%v is still held, want it ended [%s]", k, c.requirement)
				}
			}
			if got := rrKinds(a.Events()); !slices.Equal(got, append([]string{}, c.events...)) {
				t.Errorf("events = %v, want %v (the Healing before the conditions) [%s]", got, c.events, c.requirement)
			}
		})
	}
}

// Requirement: per case (FR-038, table 4.7b: Heal needs a cast level of 6 to 9; the amount is the caller's, so the package holds no formula; Source: SRD 5.2.1, "Heal" p. 139)
// Path: per case.
func TestApplyRestorativeHealingInvalidLevel(t *testing.T) {
	t.Parallel()
	set := meConditions(combat.Blinded{})
	for _, level := range []int{-1, 0, 5, 10, math.MaxInt} {
		res, err := combat.ApplyRestorativeHealing(combat.Heal, gid("hero"), pcStatus(t, 5, 40, 0), set, 70, level)
		if err != nil || res != (combat.RestorativeHealingInvalidLevel{CastLevel: level}) {
			t.Errorf("Heal at level %d = %v, %v, want RestorativeHealingInvalidLevel", level, res, err)
		}
	}
	for _, spell := range []combat.RestorativeSpell{combat.PowerWordHeal, combat.MassHeal} {
		if _, err := combat.ApplyRestorativeHealing(spell, gid("hero"), pcStatus(t, 5, 40, 0), set, 5, 0); err != nil {
			t.Errorf("%v with a cast level of 0: %v, want the level ignored", spell, err)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-038 - a Dead creature (for Heal, Power Word Heal and Mass Heal, including one that died of a maximum of 0)
// gives RestorativeHealingRejected (CreatureIsDead) with nothing removed; an invalid cast level is checked first for Heal ("A dead creature ... can't regain them", Q-10).
// Path: unhappy (state error).
func TestApplyRestorativeHealingDead(t *testing.T) {
	t.Parallel()
	zeroMax := status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
	for _, spell := range []combat.RestorativeSpell{combat.Heal, combat.PowerWordHeal, combat.MassHeal} {
		for _, st := range []combat.HitPointStatus{deadStatus(t, combat.MassiveDamage), zeroMax} {
			res, err := combat.ApplyRestorativeHealing(spell, gid("hero"), st, meConditions(combat.Blinded{}, combat.Stunned{}), 50, 6)
			if err != nil || res != (combat.RestorativeHealingRejected{Rejection: combat.CreatureIsDead}) {
				t.Errorf("%v on a Dead creature = %v, %v, want RestorativeHealingRejected (CreatureIsDead)", spell, res, err)
			}
		}
	}
	res, _ := combat.ApplyRestorativeHealing(combat.Heal, gid("hero"), deadStatus(t, combat.MassiveDamage), combat.ConditionSet{}, 50, 12)
	if res != (combat.RestorativeHealingInvalidLevel{CastLevel: 12}) {
		t.Errorf("Heal at level 12 on a Dead creature = %v, want the level checked first", res)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-047 - a spell that is not one of the three, a zero subject or status and a negative amount are errors matching
// srderr.ErrInvalidInput with a nil result.
// Path: unhappy (invalid input).
func TestApplyRestorativeHealingInvalidInput(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 5, 40, 0)
	for name, call := range map[string]func() (combat.CastRestorativeHealingResult, error){
		"zero spell": func() (combat.CastRestorativeHealingResult, error) {
			return combat.ApplyRestorativeHealing(0, gid("hero"), st, combat.ConditionSet{}, 5, 6)
		},
		"unknown spell": func() (combat.CastRestorativeHealingResult, error) {
			return combat.ApplyRestorativeHealing(9, gid("hero"), st, combat.ConditionSet{}, 5, 6)
		},
		"zero subject": func() (combat.CastRestorativeHealingResult, error) {
			return combat.ApplyRestorativeHealing(combat.Heal, "", st, combat.ConditionSet{}, 5, 6)
		},
		"zero status": func() (combat.CastRestorativeHealingResult, error) {
			return combat.ApplyRestorativeHealing(combat.Heal, gid("hero"), combat.HitPointStatus{}, combat.ConditionSet{}, 5, 6)
		},
		"negative Heal": func() (combat.CastRestorativeHealingResult, error) {
			return combat.ApplyRestorativeHealing(combat.Heal, gid("hero"), st, combat.ConditionSet{}, -1, 6)
		},
		"negative Mass": func() (combat.CastRestorativeHealingResult, error) {
			return combat.ApplyRestorativeHealing(combat.MassHeal, gid("hero"), st, combat.ConditionSet{}, math.MinInt, 6)
		},
	} {
		if res, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: %v, %v, want a nil result and an invalid input error", name, res, err)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-045 - ApplyRestorativeHealing changes none of its inputs, gives equal results for equal inputs and returns a copy of its
// events; a huge Heal amount stops at the maximum.
// Path: unhappy (immutability).
func TestApplyRestorativeHealingImmutable(t *testing.T) {
	t.Parallel()
	st, set := pcStatus(t, 5, 40, 0), meConditions(combat.Blinded{}, combat.Poisoned{})
	beforeStatus, beforeSet := st, set
	first, second := rrHealed(t, combat.Heal, st, set, math.MaxInt, 6), rrHealed(t, combat.Heal, st, set, math.MaxInt, 6)
	if !reflect.DeepEqual(st, beforeStatus) || set != beforeSet || !reflect.DeepEqual(first, second) || first.Status().HitPoints().Current() != 40 {
		t.Error("ApplyRestorativeHealing changed an input, gave unequal results, or did not stop at the maximum")
	}
	ev := first.Events()
	ev[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md table 4.7a - the facts of each restorative spell: the conditions it ends in order, and the names.
// Path: happy.
func TestRestorativeSpellFacts(t *testing.T) {
	t.Parallel()
	for spell, want := range map[combat.RestorativeSpell][]content.ConditionKind{
		combat.Heal:          {content.ConditionBlinded, content.ConditionDeafened, content.ConditionPoisoned},
		combat.MassHeal:      {content.ConditionBlinded, content.ConditionDeafened, content.ConditionPoisoned},
		combat.PowerWordHeal: {content.ConditionCharmed, content.ConditionFrightened, content.ConditionParalyzed, content.ConditionPoisoned, content.ConditionStunned},
		0:                    nil,
	} {
		if got := spell.ConditionsEnded(); !slices.Equal(got, want) {
			t.Errorf("%v.ConditionsEnded() = %v, want %v", spell, got, want)
		}
	}
	got := combat.Heal.ConditionsEnded()
	got[0] = content.ConditionStunned
	if combat.Heal.ConditionsEnded()[0] != content.ConditionBlinded {
		t.Error("ConditionsEnded is not a fresh slice")
	}
	for s, want := range map[combat.RestorativeSpell]string{combat.Heal: "heal", combat.PowerWordHeal: "power_word_heal", combat.MassHeal: "mass_heal", 0: "unknown"} {
		if s.String() != want {
			t.Errorf("RestorativeSpell(%d).String() = %q, want %q", int(s), s.String(), want)
		}
	}
}
