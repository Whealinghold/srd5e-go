package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// RollContribution is one Advantage or Disadvantage a condition gives a roll,
// tagged with the condition that produced it. Comparable with ==.
type RollContribution struct {
	kind         content.ConditionKind
	circumstance modifier.RollCircumstance
}

// Kind returns the condition that produced the contribution.
func (c RollContribution) Kind() content.ConditionKind { return c.kind }

// Circumstance returns modifier.Advantage{} or modifier.Disadvantage{}.
func (c RollContribution) Circumstance() modifier.RollCircumstance { return c.circumstance }

// RollContributions are the circumstances the conditions give one roll, ordered
// by content.ConditionKind so a breakdown is the same however the sets were
// built. The zero value is no contribution. Not comparable with ==; the slice is
// copied on the way out.
type RollContributions struct{ items []RollContribution }

// newRollContributions orders items by kind, keeping the order of equal kinds.
func newRollContributions(items []RollContribution) RollContributions {
	slices.SortStableFunc(items, func(a, b RollContribution) int { return int(a.kind) - int(b.kind) })
	return RollContributions{items}
}

// Items returns a copy of the contributions.
func (r RollContributions) Items() []RollContribution { return slices.Clone(r.items) }

// Mode combines the contributions with modifier.Resolve: any number of
// Advantages gives Advantage, any number of Disadvantages Disadvantage, and at
// least one of each cancels to Normal (SRD 5.2.1, "Advantage", p. 176).
func (r RollContributions) Mode() modifier.RollMode {
	cs := make([]modifier.RollCircumstance, len(r.items))
	for i, c := range r.items {
		cs[i] = c.circumstance
	}
	return modifier.Resolve(cs...)
}
