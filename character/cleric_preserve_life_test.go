package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-cleric.md FR-105 to FR-111, FR-128, table 4.12.3 PL-01 to PL-16 and RJ-07 to RJ-11 (SRD 5.2.1, "Level 3: Preserve Life", p. 40; "Bloodied", p. 177) - the pool is five times
// the Cleric level; a target must be within 30 feet and Bloodied; an allocation is not negative, not above the room (half the maximum rounded down, less the current Hit Points)
// and the total is not above the pool; the first problem in table 4.14.1 order is reported, with the room or the pool and sum.
// Path: per case (PL-01 to PL-16, RJ rows): happy / unhappy (boundary, invalid input, rule interaction).
func TestPreserveLifeValidate(t *testing.T) {
	t.Parallel()
	tg := func(key string, cur, max, dist, alloc int) character.PreserveLifeTarget {
		return character.PreserveLifeTarget{Key: key, CurrentHitPoints: cur, HitPointMaximum: max, DistanceFeet: dist, Allocation: alloc}
	}
	five := func(alloc int) []character.PreserveLifeTarget {
		var out []character.PreserveLifeTarget
		for _, k := range []string{"a", "b", "c", "d", "e"} {
			out = append(out, tg(k, 0, 80, 10, alloc))
		}
		return out
	}
	for _, c := range []struct {
		label   string
		level   int
		targets []character.PreserveLifeTarget
		want    character.PreserveLifeProblem
	}{
		{"PL-01", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, 6)}, character.PreserveLifeProblem{}},
		{"PL-02", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, 7)}, character.PreserveLifeProblem{Reason: character.ClericAllocationExceedsRoom, Target: "a", Room: 6}},
		{"PL-03", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, 5)}, character.PreserveLifeProblem{}},
		{"PL-04", 3, []character.PreserveLifeTarget{tg("a", 10, 20, 10, 1)}, character.PreserveLifeProblem{Reason: character.ClericAllocationExceedsRoom, Target: "a", Room: 0}},
		{"PL-05", 3, []character.PreserveLifeTarget{tg("a", 10, 20, 10, 0)}, character.PreserveLifeProblem{}},
		{"PL-06", 3, []character.PreserveLifeTarget{tg("a", 11, 20, 10, 1)}, character.PreserveLifeProblem{Reason: character.ClericTargetNotBloodied, Target: "a"}},
		{"PL-07", 3, []character.PreserveLifeTarget{tg("a", 10, 21, 10, 1)}, character.PreserveLifeProblem{Reason: character.ClericAllocationExceedsRoom, Target: "a", Room: 0}},
		{"PL-08", 3, []character.PreserveLifeTarget{tg("a", 0, 21, 10, 10)}, character.PreserveLifeProblem{}},
		{"PL-09", 3, []character.PreserveLifeTarget{tg("a", 0, 40, 10, 8), tg("b", 0, 40, 10, 7)}, character.PreserveLifeProblem{}},
		{"PL-10", 3, []character.PreserveLifeTarget{tg("a", 0, 40, 10, 8), tg("b", 0, 40, 10, 8)}, character.PreserveLifeProblem{Reason: character.ClericAllocationExceedsPool, Pool: 15, Sum: 16}},
		{"PL-11", 3, []character.PreserveLifeTarget{tg("a", 0, 40, 10, 14)}, character.PreserveLifeProblem{}},
		{"PL-12", 20, five(20), character.PreserveLifeProblem{}},
		{"PL-13", 3, []character.PreserveLifeTarget{tg("cleric", 2, 30, 0, 13)}, character.PreserveLifeProblem{}},
		{"PL-14", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, 1), tg("a", 4, 20, 10, 1)}, character.PreserveLifeProblem{Reason: character.ClericDuplicateTarget, Target: "a"}},
		{"PL-15", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 35, 1)}, character.PreserveLifeProblem{Reason: character.ClericTargetOutOfRange, Target: "a"}},
		{"PL-16", 3, nil, character.PreserveLifeProblem{}},
		{"exactly 30 feet", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 30, 1)}, character.PreserveLifeProblem{}},
		{"RJ-07", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 35, 1), tg("a", 4, 20, 10, 1)}, character.PreserveLifeProblem{Reason: character.ClericDuplicateTarget, Target: "a"}},
		{"RJ-08", 3, []character.PreserveLifeTarget{tg("a", 19, 20, 35, 1)}, character.PreserveLifeProblem{Reason: character.ClericTargetOutOfRange, Target: "a"}},
		{"RJ-09", 3, []character.PreserveLifeTarget{tg("a", 19, 20, 10, -1)}, character.PreserveLifeProblem{Reason: character.ClericTargetNotBloodied, Target: "a"}},
		{"RJ-10", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, -1)}, character.PreserveLifeProblem{Reason: character.ClericNegativeAllocation, Target: "a"}},
		{"RJ-11", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, 16)}, character.PreserveLifeProblem{Reason: character.ClericAllocationExceedsRoom, Target: "a", Room: 6}},
		{"first failing target in command order", 3, []character.PreserveLifeTarget{tg("a", 4, 20, 10, 1), tg("b", 19, 20, 10, 1), tg("c", 4, 20, 40, 1)}, character.PreserveLifeProblem{Reason: character.ClericTargetNotBloodied, Target: "b"}},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			got, err := character.PreserveLife{}.Validate(character.MustClericLevel(c.level), c.targets)
			if err != nil || got != c.want {
				t.Errorf("= %+v, %v; want %+v", got, err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-105, FR-117 - the pool is five times the Cleric level (a level 3 Cleric / level 5 Fighter has 15) and the room is half the maximum rounded
// down less the current Hit Points; a zero level, a negative current value, a maximum below 1 and a negative distance are invalid input.
// Path: per case (unhappy: boundary, invalid input).
func TestPreserveLifePoolRoomAndInvalid(t *testing.T) {
	t.Parallel()
	if (character.PreserveLife{}).Pool(character.MustClericLevel(3)) != 15 || (character.PreserveLife{}).Pool(character.MustClericLevel(20)) != 100 ||
		(character.PreserveLife{}).Room(4, 20) != 6 || (character.PreserveLife{}).Room(10, 21) != 0 {
		t.Error("pool or room")
	}
	_, err := character.PreserveLife{}.Validate(character.ClericLevel{}, nil)
	wantInvalid(t, err, "zero level")
	for name, tgt := range map[string]character.PreserveLifeTarget{
		"negative current":  {Key: "a", CurrentHitPoints: -1, HitPointMaximum: 20},
		"zero maximum":      {Key: "a", HitPointMaximum: 0},
		"negative distance": {Key: "a", HitPointMaximum: 20, DistanceFeet: -1},
	} {
		_, err := character.PreserveLife{}.Validate(character.MustClericLevel(3), []character.PreserveLifeTarget{tgt})
		wantInvalid(t, err, name)
	}
}
