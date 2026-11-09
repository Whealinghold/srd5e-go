package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-cleric.md FR-059, FR-060, FR-065 (SRD 5.2.1, "Level 2: Channel Divinity" (Turn Undead), p. 37; "Level 5: Sear Undead", p. 37) - a failed save gives
// Frightened and Incapacitated; the effect ends early when the creature takes damage, when the Cleric has the Incapacitated condition or dies; the names are pinned.
// Path: happy.
func TestTurnUndeadConditionsAndEndings(t *testing.T) {
	t.Parallel()
	if got := (character.TurnUndead{}).Conditions(); !slices.Equal(got, []content.ConditionKind{content.ConditionFrightened, content.ConditionIncapacitated}) {
		t.Errorf("conditions = %v", got)
	}
	got := (character.TurnUndead{}).EarlyEndings()
	want := []character.TurnUndeadEnding{character.TurnUndeadEndsOnDamage, character.TurnUndeadEndsOnClericIncapacitated, character.TurnUndeadEndsOnClericDeath}
	names := []string{"damage", "cleric_incapacitated", "cleric_death"}
	if !slices.Equal(got, want) {
		t.Fatalf("endings = %v", got)
	}
	for i, e := range want {
		if e.String() != names[i] {
			t.Errorf("ending %d = %q", e, e)
		}
	}
	if character.TurnUndeadEnding(0).String() != "unknown" {
		t.Error("zero ending name")
	}
}

// Requirement: doc/specs/class-cleric.md FR-061, FR-062, FR-063, FR-128 - the first target in command order that is not Undead or is beyond 30 feet is named (not Undead first; exactly 30 feet is
// accepted); an empty list and a negative distance are handled (empty is fine, negative is invalid input).
// Path: per case (unhappy: boundary, invalid input, empty and minimal).
func TestTurnUndeadTargetProblem(t *testing.T) {
	t.Parallel()
	u := func(key string, d int) character.UndeadTarget {
		return character.UndeadTarget{Key: key, Type: content.CreatureTypeUndead, DistanceFeet: d}
	}
	beast := character.UndeadTarget{Key: "wolf", Type: content.CreatureTypeBeast, DistanceFeet: 10}
	for _, c := range []struct {
		name    string
		targets []character.UndeadTarget
		reason  character.ClericReason
		key     string
	}{
		{"none", nil, 0, ""}, {"exactly 30", []character.UndeadTarget{u("a", 30)}, 0, ""}, {"31", []character.UndeadTarget{u("a", 31)}, character.ClericTargetOutOfRange, "a"},
		{"not Undead", []character.UndeadTarget{beast}, character.ClericTargetNotUndead, "wolf"},
		{"not Undead and far", []character.UndeadTarget{{Key: "wolf", Type: content.CreatureTypeBeast, DistanceFeet: 35}}, character.ClericTargetNotUndead, "wolf"},
		{"command order", []character.UndeadTarget{u("a", 5), u("b", 40), beast}, character.ClericTargetOutOfRange, "b"},
		{"second is not Undead", []character.UndeadTarget{u("a", 5), beast, u("b", 40)}, character.ClericTargetNotUndead, "wolf"},
	} {
		reason, key, err := character.TurnUndead{}.TargetProblem(c.targets)
		if err != nil || reason != c.reason || key != c.key {
			t.Errorf("%s: = %v, %q, %v; want %v, %q", c.name, reason, key, err, c.reason, c.key)
		}
	}
	_, _, err := character.TurnUndead{}.TargetProblem([]character.UndeadTarget{u("a", -1)})
	wantInvalid(t, err, "negative distance")
}

// Requirement: doc/specs/class-cleric.md FR-064, FR-065, FR-066, FR-067, table 4.8.1 SU-01 to SU-08 (SRD 5.2.1, "Level 5: Sear Undead", p. 37; "Ability Modifiers", p. 6) - the dice
// are the Wisdom modifier, at least 1; none below Cleric level 5; the total is the sum of the d8s as Radiant damage that does not end the effect; a wrong count or a roll outside 1 to 8
// is invalid input.
// Path: per case (SU-01 to SU-08): happy / unhappy (boundary, invalid input).
func TestSearUndead(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label   string
		level   int
		wis     int
		dice    int
		rolls   []int
		total   int
		invalid bool
	}{
		{"SU-01", 5, 3, 3, []int{2, 4, 6}, 12, false}, {"SU-02", 5, 1, 1, []int{8}, 8, false}, {"SU-03", 5, 0, 1, []int{1}, 1, false}, {"SU-04", 5, -2, 1, []int{5}, 5, false},
		{"SU-05", 12, 5, 5, []int{8, 8, 8, 8, 8}, 40, false}, {"SU-06", 5, 2, 2, []int{1, 1}, 2, false}, {"SU-08", 5, 3, 3, []int{2, 4}, 0, true},
		{"a 9", 5, 1, 1, []int{9}, 0, true}, {"a 0", 5, 1, 1, []int{0}, 0, true},
	} {
		n, ok := character.SearUndead{}.Dice(character.MustClericLevel(c.level), modifierOf(t, c.wis))
		if !ok || n != c.dice {
			t.Errorf("%s: Dice = %d, %v; want %d", c.label, n, ok, c.dice)
		}
		d, err := character.SearUndead{}.Resolve(character.MustClericLevel(c.level), modifierOf(t, c.wis), c.rolls)
		if c.invalid {
			wantInvalid(t, err, c.label)
			continue
		}
		if err != nil || d.Total() != c.total || !slices.Equal(d.Rolls(), c.rolls) || d.DamageType() != content.DamageRadiant || d.EndsEffect() {
			t.Errorf("%s: = %+v, %v", c.label, d, err)
		}
		d.Rolls()[0] = 99
		if d.Rolls()[0] == 99 {
			t.Error("Rolls returned a shared slice")
		}
	}
	// SU-07: not yet gained.
	if n, ok := (character.SearUndead{}).Dice(character.MustClericLevel(4), modifierOf(t, 3)); ok || n != 0 {
		t.Errorf("SU-07: Dice at level 4 = %d, %v", n, ok)
	}
	_, err := character.SearUndead{}.Resolve(character.MustClericLevel(4), modifierOf(t, 3), []int{1, 2, 3})
	wantInvalid(t, err, "level 4")
	if _, ok := (character.SearUndead{}).Dice(character.ClericLevel{}, modifierOf(t, 3)); ok {
		t.Error("Dice of the zero level")
	}
}
