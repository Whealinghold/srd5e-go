package character

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

func internalInitiate() MagicInitiateChoices {
	return MagicInitiateChoices{list: "cleric", ability: ability.Wisdom, cantrips: []content.SpellID{"guidance", "light"}, levelOne: "bless"}
}

// Requirement: doc/specs/feats-behaviour.md FR-034, FR-036 - Spell Change is recorded per total level 1 to 20: the bit of the level is set by replacing and read by SpellChangedAt, no other level and not the zero Level is set, and replacing twice at one level keeps the bit.
// Path: unhappy (boundary).
func TestMagicInitiateChoicesRecordSpellChangeLevels(t *testing.T) {
	t.Parallel()
	for n := 1; n <= MaxLevel; n++ {
		level := MustLevel(n)
		changed := internalInitiate().replacing("guidance", "mending", level)
		for m := 1; m <= MaxLevel; m++ {
			if got := changed.SpellChangedAt(MustLevel(m)); got != (m == n) {
				t.Errorf("replaced at %d: SpellChangedAt(%d) = %v", n, m, got)
			}
		}
		if changed.SpellChangedAt(Level{}) {
			t.Errorf("replaced at %d: the zero level is set", n)
		}
		if again := changed.replacing("mending", "guidance", level); !again.SpellChangedAt(level) || again.changed != changed.changed {
			t.Errorf("replacing twice at %d changed the bit set: %b", n, again.changed)
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-037 - replacing a cantrip changes only that cantrip, wherever it stands, and the level 1 spell only when it is the one replaced.
// Path: per case.
func TestMagicInitiateChoicesReplacing(t *testing.T) {
	t.Parallel()
	l := MustLevel(5)
	first := internalInitiate().replacing("guidance", "mending", l)
	second := internalInitiate().replacing("light", "mending", l)
	one := internalInitiate().replacing("bless", "cure_wounds", l)
	if first.cantrips[0] != "mending" || first.cantrips[1] != "light" || first.levelOne != "bless" ||
		second.cantrips[0] != "guidance" || second.cantrips[1] != "mending" || second.levelOne != "bless" ||
		one.cantrips[0] != "guidance" || one.cantrips[1] != "light" || one.levelOne != "cure_wounds" {
		t.Errorf("replacing = %+v %+v %+v", first, second, one)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-057 - choices of different shapes are never equal, and a zero Magic Initiate choice is not a no-choice.
// Path: unhappy (rule interaction).
func TestEqualFeatChoices(t *testing.T) {
	t.Parallel()
	if equalFeatChoices(MagicInitiateChoices{}, NoFeatChoices{}) || equalFeatChoices(NoFeatChoices{}, MagicInitiateChoices{}) {
		t.Error("a Magic Initiate choice equals no choices")
	}
	if !equalFeatChoices(NoFeatChoices{}, NoFeatChoices{}) || !equalFeatChoices(internalInitiate(), internalInitiate()) || equalFeatChoices(internalInitiate(), MagicInitiateChoices{}) {
		t.Error("equal choices must compare equal")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-057 - MagicInitiateChoices compares the Spell Change levels too, whichever way they differ.
// Path: per case.
func TestMagicInitiateChoicesEqualComparesSpellChangeLevels(t *testing.T) {
	t.Parallel()
	base := internalInitiate()
	for _, n := range []int{1, 5, 20} {
		changed := base.replacing("guidance", "guidance", MustLevel(n))
		if base.Equal(changed) || changed.Equal(base) || !changed.Equal(changed) {
			t.Errorf("a Spell Change at level %d must be part of the value", n)
		}
	}
}
