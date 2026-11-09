package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-041, Table E (SRD 5.2.1, "Relentless Endurance", p. 86; "Instant Death", p. 17) - the trait triggers, dropping to 1 Hit Point, when the creature was not killed outright and does not for one killed outright or a nil outcome.
// Path: per case.
func TestRelentlessEnduranceResolve(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		outcome character.ZeroHitPointsOutcome
		instead int
		ok      bool
	}{
		{"not killed", character.ZeroHitPointsNotKilled{}, 1, true},
		{"killed outright", character.ZeroHitPointsKilledOutright{}, 0, false},
		{"no outcome", nil, 0, false},
	} {
		if instead, ok := (character.RelentlessEndurance{}).Resolve(c.outcome); instead != c.instead || ok != c.ok {
			t.Errorf("%s = %d, %v; want %d, %v", c.name, instead, ok, c.instead, c.ok)
		}
	}
}
