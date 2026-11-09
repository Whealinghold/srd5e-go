package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// BackgroundAbilityIncrease is how a background raises the starting ability
// scores: one by 2 and another by 1, or all three listed by 1 (SRD 5.2.1,
// "Adjust Ability Scores", p. 21; "Parts of a Background", p. 83). It is a closed
// set. Variants: PlusTwoPlusOne, PlusOneToAll.
type BackgroundAbilityIncrease interface {
	// ApplyTo applies the increase to scores for a background that lists the
	// abilities in listed. It alters nothing: it returns the new scores or the
	// refusal, which names every ability not listed and every ability that would
	// pass 20.
	ApplyTo(scores ability.Scores, listed []ability.Ability) BackgroundIncreaseResult
	isBackgroundAbilityIncrease()
}

// PlusTwoPlusOne adds 2 to one ability and 1 to a different one. Build it with
// NewPlusTwoPlusOne; the zero value is invalid. It is comparable with ==.
type PlusTwoPlusOne struct{ plusTwo, plusOne ability.Ability }

// NewPlusTwoPlusOne returns the increase of plusTwo by 2 and plusOne by 1. It
// returns an error matching srderr.ErrInvalidInput when an ability is invalid or
// both are the same. It never panics.
func NewPlusTwoPlusOne(plusTwo, plusOne ability.Ability) (PlusTwoPlusOne, error) {
	if !plusTwo.Valid() || !plusOne.Valid() || plusTwo == plusOne {
		return PlusTwoPlusOne{}, invalidInput("plus two plus one needs two different valid abilities")
	}
	return PlusTwoPlusOne{plusTwo: plusTwo, plusOne: plusOne}, nil
}

// MustPlusTwoPlusOne is like NewPlusTwoPlusOne but panics with its error.
func MustPlusTwoPlusOne(plusTwo, plusOne ability.Ability) PlusTwoPlusOne {
	p, err := NewPlusTwoPlusOne(plusTwo, plusOne)
	if err != nil {
		panic(err)
	}
	return p
}

// PlusTwo returns the ability raised by 2.
func (p PlusTwoPlusOne) PlusTwo() ability.Ability { return p.plusTwo }

// PlusOne returns the ability raised by 1.
func (p PlusTwoPlusOne) PlusOne() ability.Ability { return p.plusOne }

// PlusOneToAll adds 1 to each of the three abilities a background lists. It is
// always valid.
type PlusOneToAll struct{}

func (PlusTwoPlusOne) isBackgroundAbilityIncrease() {}
func (PlusOneToAll) isBackgroundAbilityIncrease()   {}

// ApplyTo implements BackgroundAbilityIncrease.
func (p PlusTwoPlusOne) ApplyTo(scores ability.Scores, listed []ability.Ability) BackgroundIncreaseResult {
	return applyBackgroundIncrease(scores, listed, map[ability.Ability]int{p.plusTwo: 2, p.plusOne: 1})
}

// ApplyTo implements BackgroundAbilityIncrease.
func (PlusOneToAll) ApplyTo(scores ability.Scores, listed []ability.Ability) BackgroundIncreaseResult {
	by := map[ability.Ability]int{}
	for _, a := range listed {
		by[a] = 1
	}
	return applyBackgroundIncrease(scores, listed, by)
}

// applyBackgroundIncrease adds by to scores, refusing it as a whole when an
// ability is not listed or would pass 20.
func applyBackgroundIncrease(scores ability.Scores, listed []ability.Ability, by map[ability.Ability]int) BackgroundIncreaseResult {
	var notListed, exceeds []ability.Ability
	next := scores
	for _, a := range ability.All() {
		n, ok := by[a]
		if !ok {
			continue
		}
		if !slices.Contains(listed, a) {
			notListed = append(notListed, a)
		}
		sum := scores.Score(a).Value() + n
		if sum > maxStartingScore {
			exceeds = append(exceeds, a)
			continue
		}
		if s, err := ability.NewScore(sum); err == nil {
			if updated, err := next.With(a, s); err == nil {
				next = updated
			}
		}
	}
	if len(notListed) > 0 || len(exceeds) > 0 {
		return BackgroundIncreaseRefused{notListed: notListed, exceeds: exceeds}
	}
	return BackgroundIncreaseApplied{scores: next}
}

// BackgroundIncreaseResult is the closed outcome of ApplyTo. Variants:
// BackgroundIncreaseApplied, BackgroundIncreaseRefused.
type BackgroundIncreaseResult interface{ isBackgroundIncreaseResult() }

// BackgroundIncreaseApplied holds the scores after the increase.
type BackgroundIncreaseApplied struct{ scores ability.Scores }

// Scores returns the increased scores.
func (r BackgroundIncreaseApplied) Scores() ability.Scores { return r.scores }

// BackgroundIncreaseRefused is an increase a rule stopped; no score changed. It
// lists every ability the background does not list and every ability that would
// pass 20 (SRD 5.2.1, "Adjust Ability Scores", p. 21). The zero value lists
// nothing.
type BackgroundIncreaseRefused struct{ notListed, exceeds []ability.Ability }

// NotListed returns a new slice of the increased abilities the background does
// not list, in Ability declaration order.
func (r BackgroundIncreaseRefused) NotListed() []ability.Ability { return slices.Clone(r.notListed) }

// ExceedsMaximum returns a new slice of the abilities whose score would pass 20,
// in Ability declaration order.
func (r BackgroundIncreaseRefused) ExceedsMaximum() []ability.Ability { return slices.Clone(r.exceeds) }

func (BackgroundIncreaseApplied) isBackgroundIncreaseResult() {}
func (BackgroundIncreaseRefused) isBackgroundIncreaseResult() {}
