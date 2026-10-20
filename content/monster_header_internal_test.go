package content

import "testing"

// Requirement: doc/specs/content-monsters.md FR-016 - the duplicate speed rule keys on the set of modes
// and the qualifier: a different mode set or a different qualifier is not a
// duplicate, the same pair is.
// Path: per case.
func TestSpeedIssuesKey(t *testing.T) {
	t.Parallel()
	mk := func(modes []SpeedMode, q string) SpeedEntry {
		e, err := NewSpeedEntry(modes, 30, false, q)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	walk, fly := []SpeedMode{SpeedWalk}, []SpeedMode{SpeedFly}
	if got := speedIssues([]SpeedEntry{mk(walk, ""), mk(fly, "")}); len(got) != 0 {
		t.Errorf("different modes flagged: %v [happy]", got)
	}
	if got := speedIssues([]SpeedEntry{mk(walk, ""), mk(walk, "x")}); len(got) != 0 {
		t.Errorf("different qualifiers flagged: %v [happy]", got)
	}
	if got := speedIssues([]SpeedEntry{mk(walk, "x"), mk(walk, "x")}); len(got) != 1 {
		t.Errorf("duplicate not flagged: %v [unhappy]", got)
	}
}
