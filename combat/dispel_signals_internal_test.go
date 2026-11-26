package combat

import "testing"

// Requirement: doc/specs/dispel-and-counterspell.md FR-056 and FR-057 - each srd5e.combat.result value of the Dispel Magic, Counterspell and Antimagic Field use cases is pinned to its catalogue spelling and carries the class of section 7 (SRD 5.2.1, "Dispel Magic", p. 124; "Counterspell", p. 120; "Antimagic Field", p. 109).
// Path: per case.
func TestDispelSignalValuesArePinned(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got, want               string
		class, wantClass        string
	}{
		{"Dispel Magic ended a spell", "FR-056", "happy", valueResultCompleted, "completed", valueResultClassApplied, "applied"},
		{"Dispel Magic ended nothing", "FR-056", "happy", valueResultDispelNothingEnded, "dispel_nothing_ended", valueResultClassApplied, "applied"},
		{"Dispel Magic invalid checks", "FR-056", "unhappy (invalid input)", valueResultDispelInvalidChecks, "dispel_invalid_checks", valueResultClassInvalid, "invalid"},
		{"Counterspell resisted", "FR-056", "happy", valueResultCounterspellResisted, "counterspell_resisted", valueResultClassApplied, "applied"},
		{"Counterspell ineligible", "FR-056", "unhappy (rule interaction)", valueResultCounterspellIneligible, "counterspell_ineligible", valueResultClassRefused, "refused"},
		{"Counterspell invalid save", "FR-056", "unhappy (invalid input)", valueResultCounterspellInvalidSave, "counterspell_invalid_save", valueResultClassInvalid, "invalid"},
		{"Antimagic Field blocked", "FR-056", "unhappy (rule interaction)", valueResultAntimagicBlocked, "antimagic_blocked", valueResultClassRefused, "refused"},
		{"Counterspell countered", "FR-056", "happy", valueResultCompleted, "completed", valueResultClassApplied, "applied"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("value = %q, want %q [%s]", c.got, c.want, c.requirement)
			}
			if c.class != c.wantClass {
				t.Errorf("class = %q, want %q [%s]", c.class, c.wantClass, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-054 - the three ending event kinds are distinct (combat-foundations FR-015 to FR-019).
// Path: happy.
func TestSpellEndingEventKindsAreDistinct(t *testing.T) {
	t.Parallel()
	kinds := []string{OngoingSpellEnded{}.Kind(), SpellCountered{}.Kind(), CounterspellResisted{}.Kind()}
	want := []string{"ongoing_spell_ended", "spell_countered", "counterspell_resisted"}
	seen := map[string]bool{}
	for i, k := range kinds {
		if k != want[i] {
			t.Errorf("kind = %q, want %q [FR-054]", k, want[i])
		}
		if seen[k] {
			t.Errorf("kind %q is not unique [FR-054]", k)
		}
		seen[k] = true
	}
}
