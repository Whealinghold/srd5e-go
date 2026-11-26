package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/attacks.md Limits (plan P10) - the cover, nearby enemy and circumstance lists of an attack command each hold at most 256 elements; one more is a LimitExceededError and the params are not altered.
// Path: per case.
func TestNewResolveAttackCommandListCaps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path, collection string
		edit                                func(*combat.ResolveAttackParams, int)
		n                                   int
		over                                bool
	}{
		{"256 cover sources", "P10 at the cap", "unhappy (boundary)", "cover sources", func(p *combat.ResolveAttackParams, n int) { p.Cover = make([]combat.CoverSource, n) }, 256, false},
		{"257 cover sources", "P10 cap plus one", "unhappy (boundary)", "cover sources", func(p *combat.ResolveAttackParams, n int) { p.Cover = make([]combat.CoverSource, n) }, 257, true},
		{"256 nearby enemies", "P10 at the cap", "unhappy (boundary)", "nearby enemies", func(p *combat.ResolveAttackParams, n int) { p.NearbyEnemies = make([]combat.NearbyEnemy, n) }, 256, false},
		{"257 nearby enemies", "P10 cap plus one", "unhappy (boundary)", "nearby enemies", func(p *combat.ResolveAttackParams, n int) { p.NearbyEnemies = make([]combat.NearbyEnemy, n) }, 257, true},
		{"256 circumstances", "P10 at the cap", "unhappy (boundary)", "circumstances", func(p *combat.ResolveAttackParams, n int) { p.Circumstances = make([]modifier.RollCircumstance, n) }, 256, false},
		{"257 circumstances", "P10 cap plus one", "unhappy (boundary)", "circumstances", func(p *combat.ResolveAttackParams, n int) { p.Circumstances = make([]modifier.RollCircumstance, n) }, 257, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := meleeParams()
			c.edit(&p, c.n)
			_, err := combat.NewResolveAttackCommand(p)
			if c.over {
				wantLimit(t, err, c.collection, 256)
				return
			}
			if err != nil {
				t.Errorf("error = %v, want none [%s]", err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md Limits (plan P10) - WithAdditions that pushes the circumstances past 256 is refused with the same error and leaves the command unchanged.
// Path: unhappy (boundary).
func TestWithAdditionsCircumstanceCap(t *testing.T) {
	t.Parallel()
	p := meleeParams()
	p.Circumstances = make([]modifier.RollCircumstance, 256)
	cmd := attackCommand(t, p)
	_, err := cmd.WithAdditions(combat.Additions{Circumstances: make([]modifier.RollCircumstance, 1)})
	wantLimit(t, err, "circumstances", 256)
}

// Requirement: doc/specs/attacks.md Limits (plan P10) - a damage spec holds at most 64 components; 65 is a LimitExceededError.
// Path: per case.
func TestNewDamageSpecCap(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		n                 int
		over              bool
	}{{"64 components", "P10 at the cap", 64, false}, {"65 components", "P10 cap plus one", 65, true}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			parts := make([]combat.DamageComponent, c.n)
			for i := range parts {
				d, err := combat.NewDiceDamage(1, dice.D6(), content.DamageSlashing, 0, false)
				if err != nil {
					t.Fatal(err)
				}
				parts[i] = d
			}
			_, err := combat.NewDamageSpec(parts...)
			if c.over {
				wantLimit(t, err, "damage components", 64)
				return
			}
			if err != nil {
				t.Errorf("error = %v, want none [%s]", err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md Limits (plan P10) - WithAdditions reads at most 256 added modifiers; 257 is a LimitExceededError and the command is unchanged.
// Path: unhappy (boundary).
func TestWithAdditionsModifierCap(t *testing.T) {
	t.Parallel()
	cmd := attackCommand(t, meleeParams())
	runCap(t, "addition modifiers", 256, func(n int) error {
		mods := make([]modifier.Modifier, n)
		_, err := cmd.WithAdditions(combat.Additions{Modifiers: mods})
		return err
	})
}
