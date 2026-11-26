package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

func resPCs(tb testing.TB, id string, conds ...combat.Condition) combat.Combatant {
	tb.Helper()
	return resPC(tb, id, combat.SideA, conds...)
}

// Requirement: per case
// Path: per case.
func TestConsequenceRulesSpeedReviews(t *testing.T) {
	t.Parallel()
	g := combat.Grappled{Grappler: cid("g")}
	cases := []struct {
		name, requirement, path string
		before, after           []combat.Combatant
		want                    []combat.CombatantID
	}{
		{"nothing changed", resSpec + "FR-085", "unhappy (empty and minimal)", []combat.Combatant{resPCs(t, "a", g)}, []combat.Combatant{resPCs(t, "a", g)}, nil},
		{"no combatants", resSpec + "FR-085", "unhappy (empty and minimal)", nil, nil, nil},
		{"gains Exhaustion", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", exhaustion(1))}, []combat.CombatantID{cid("a")}},
		{"gains Grappled", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", g)}, []combat.CombatantID{cid("a")}},
		{"gains Paralyzed", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", combat.Paralyzed{})}, []combat.CombatantID{cid("a")}},
		{"gains Petrified", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", combat.Petrified{})}, []combat.CombatantID{cid("a")}},
		{"gains Restrained", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", combat.Restrained{})}, []combat.CombatantID{cid("a")}},
		{"gains Unconscious", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", combat.Unconscious{})}, []combat.CombatantID{cid("a")}},
		{"loses Restrained", resSpec + "FR-085", "happy", []combat.Combatant{resPCs(t, "a", combat.Restrained{})}, []combat.Combatant{resPCs(t, "a")}, []combat.CombatantID{cid("a")}},
		{"Exhaustion level 1 to 2", resSpec + "FR-085", "unhappy (boundary)", []combat.Combatant{resPCs(t, "a", exhaustion(1))}, []combat.Combatant{resPCs(t, "a", exhaustion(2))}, []combat.CombatantID{cid("a")}},
		{"gains Grappled while Restrained", resSpec + "FR-085", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "a", combat.Restrained{})},
			[]combat.Combatant{resPCs(t, "a", combat.Restrained{}, g)}, []combat.CombatantID{cid("a")}},
		{"gains Prone", resSpec + "FR-085", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", combat.Prone{})}, nil},
		{"gains Poisoned", resSpec + "FR-085", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "a")}, []combat.Combatant{resPCs(t, "a", combat.Poisoned{})}, nil},
		{"two kinds at once give one entry", resSpec + "FR-085", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "a")},
			[]combat.Combatant{resPCs(t, "a", combat.Restrained{}, g)}, []combat.CombatantID{cid("a")}},
		{"turn order, not id order", resSpec + "FR-085", "unhappy (rule interaction)",
			[]combat.Combatant{resPCs(t, "b"), resPCs(t, "a"), resPCs(t, "c")},
			[]combat.Combatant{resPCs(t, "b", combat.Restrained{}), resPCs(t, "a", combat.Restrained{}), resPCs(t, "c")}, []combat.CombatantID{cid("b"), cid("a")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (combat.ConsequenceRules{}).SpeedReviews(c.before, c.after); !slices.Equal(got, c.want) {
				t.Errorf("= %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestConsequenceRulesGrappleChecks(t *testing.T) {
	t.Parallel()
	byG := combat.Grappled{Grappler: cid("g")}
	cue := func(grappled, grappler string) combat.GrappleCue {
		return combat.GrappleCue{Grappled: cid(grappled), Grappler: cid(grappler)}
	}
	cases := []struct {
		name, requirement, path string
		before, after           []combat.Combatant
		want                    []combat.GrappleCue
	}{
		{"grappler falls Unconscious", resSpec + "FR-120", "happy", []combat.Combatant{resPCs(t, "g"), resPCs(t, "t", byG)},
			[]combat.Combatant{resPCs(t, "g", combat.Unconscious{}), resPCs(t, "t", byG)}, []combat.GrappleCue{cue("t", "g")}},
		{"grappler becomes Incapacitated", resSpec + "FR-120", "happy", []combat.Combatant{resPCs(t, "g"), resPCs(t, "t", byG)},
			[]combat.Combatant{resPCs(t, "g", combat.Incapacitated{}), resPCs(t, "t", byG)}, []combat.GrappleCue{cue("t", "g")}},
		{"grappler dies without being Incapacitated", resSpec + "FR-120, Q-15", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "g"), resPCs(t, "t", byG)},
			[]combat.Combatant{resDead(t, "g", combat.SideA), resPCs(t, "t", byG)}, []combat.GrappleCue{cue("t", "g")}},
		{"grappler already Incapacitated", resSpec + "FR-120", "unhappy (boundary)", []combat.Combatant{resPCs(t, "g", combat.Incapacitated{}), resPCs(t, "t", byG)},
			[]combat.Combatant{resPCs(t, "g", combat.Incapacitated{}, combat.Unconscious{}), resPCs(t, "t", byG)}, nil},
		{"grappler already dead", resSpec + "FR-120", "unhappy (boundary)", []combat.Combatant{resDead(t, "g", combat.SideA), resPCs(t, "t", byG)},
			[]combat.Combatant{resDead(t, "g", combat.SideA), resPCs(t, "t", byG)}, nil},
		{"nobody grappled", resSpec + "FR-120", "unhappy (empty and minimal)", []combat.Combatant{resPCs(t, "g"), resPCs(t, "t")},
			[]combat.Combatant{resPCs(t, "g", combat.Unconscious{}), resPCs(t, "t")}, nil},
		{"the grappled one is Incapacitated, not its grappler", resSpec + "FR-120", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "g"), resPCs(t, "t", byG)},
			[]combat.Combatant{resPCs(t, "g"), resPCs(t, "t", byG, combat.Unconscious{})}, nil},
		{"two grappled, in turn order", resSpec + "FR-120", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "u", byG), resPCs(t, "g"), resPCs(t, "t", byG)},
			[]combat.Combatant{resPCs(t, "u", byG), resPCs(t, "g", combat.Unconscious{}), resPCs(t, "t", byG)}, []combat.GrappleCue{cue("u", "g"), cue("t", "g")}},
		{"grappled by another grappler", resSpec + "FR-120", "unhappy (rule interaction)", []combat.Combatant{resPCs(t, "g"), resPCs(t, "t", combat.Grappled{Grappler: cid("h")}), resPCs(t, "h")},
			[]combat.Combatant{resPCs(t, "g", combat.Unconscious{}), resPCs(t, "t", combat.Grappled{Grappler: cid("h")}), resPCs(t, "h")}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (combat.ConsequenceRules{}).GrappleChecks(c.before, c.after); !slices.Equal(got, c.want) {
				t.Errorf("= %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}
