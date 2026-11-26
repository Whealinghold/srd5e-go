package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-003, FR-067 - one bad field alone makes a request invalid: a Simulacrum with a level outside 1 to 9, a blank original monster, an original size or type that is not defined, or an original Hit Point maximum under 1; Animate Dead bones with a size that is not defined; Create Undead with a blank remains id (mutation gap: only requests that were wrong in every field were tried, so each conjunct of validRequest survived being turned into an OR).
// Path: per case (unhappy, invalid input).
func TestSummonCreatureCommandRejectsOneBadField(t *testing.T) {
	t.Parallel()
	sim := func(edit func(*combat.SimulacrumRequest)) combat.SummonRequest {
		q := sumSimulacrumRequest("s", 7)
		edit(&q)
		return q
	}
	cases := []struct {
		name    string
		request combat.SummonRequest
	}{
		{"a simulacrum at level 0", sim(func(q *combat.SimulacrumRequest) { q.Level = 0 })},
		{"a simulacrum at level 10", sim(func(q *combat.SimulacrumRequest) { q.Level = 10 })},
		{"a simulacrum with a blank original monster", sim(func(q *combat.SimulacrumRequest) { q.Original.Monster = "" })},
		{"a simulacrum with an original size that is not defined", sim(func(q *combat.SimulacrumRequest) { q.Original.Size = 0 })},
		{"a simulacrum with an original type that is not defined", sim(func(q *combat.SimulacrumRequest) { q.Original.Type = 0 })},
		{"a simulacrum with an original Hit Point maximum of 0", sim(func(q *combat.SimulacrumRequest) { q.Original.HitPointMaximum = 0 })},
		{"bones with a size that is not defined", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{{Creature: cid("u"), Remains: "r", Kind: combat.RemainsBones}}}},
		{"a corpse with a blank remains id", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{{Creature: cid("u"), Size: content.SizeSmall, Becomes: combat.UndeadGhoul}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "pc", combat.SideA, c.request, sumFacts(t), combat.SummonScene{})
			if !errors.Is(err, srderr.ErrInvalidInput) || !cmd.IsZero() {
				t.Errorf("command = %v, error = %v, want the zero command and srderr.ErrInvalidInput [FR-067]", cmd, err)
			}
		})
	}
	// The same Simulacrum with every field good is accepted, so each case above fails for its one field only.
	if _, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "pc", combat.SideA, sumSimulacrumRequest("s", 7), sumFacts(t), combat.SummonScene{}); err != nil {
		t.Errorf("a valid Simulacrum = %v, want no error", err)
	}
}
