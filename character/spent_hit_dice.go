package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// SpentHitDiceProblem is a reason NewSpentHitDice refused its counts. The zero
// value is not a problem. It is an error matching srderr.ErrInvalidInput.
//
//nolint:errname // an enum of problems, named by the spec
type SpentHitDiceProblem int

// CountBelowOne: a die is listed with a spent count below 1.
const CountBelowOne SpentHitDiceProblem = iota + 1 //nolint:errname // a member of the problem enum

// String returns "count_below_one" or "unknown".
func (p SpentHitDiceProblem) String() string {
	if p == CountBelowOne {
		return "count_below_one"
	}
	return "unknown"
}

// Error renders the problem. The text is not a contract.
func (p SpentHitDiceProblem) Error() string {
	return "character: invalid spent Hit Dice: " + p.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (SpentHitDiceProblem) Unwrap() error { return srderr.ErrInvalidInput }

// SpentHitDice is how many Hit Point Dice of each size are spent and not yet
// regained (SRD 5.2.1, "Hit Point Dice", p. 22). Dice with none spent are
// omitted and entries are ordered by ascending faces, so equal spent sets are
// Equal. It is a FeatureState, stored under SpentHitDiceKey. It is immutable
// and defensively copied; it holds a slice, so compare two with Equal, not ==.
// The zero SpentHitDice is the empty set. Safe for concurrent use.
type SpentHitDice struct{ entries []HitDiceCount }

// SpentHitDiceKey returns the key under which a Character stores its
// SpentHitDice (the name "spent_hit_dice").
func SpentHitDiceKey() FeatureStateKey[SpentHitDice] {
	return MustFeatureStateKey[SpentHitDice]("spent_hit_dice")
}

// NewSpentHitDice returns the spent dice in counts, copied. It returns an error
// matching srderr.ErrInvalidInput for an invalid die, and otherwise an
// errors.Join of CountBelowOne when any count is below 1. It never panics.
func NewSpentHitDice(counts map[content.HitPointDie]int) (SpentHitDice, error) {
	below := false
	for die, n := range counts {
		if !die.Valid() {
			return SpentHitDice{}, invalidInput("invalid Hit Point Die")
		}
		below = below || n < 1
	}
	if below {
		return SpentHitDice{}, errors.Join(CountBelowOne)
	}
	return SpentHitDice{entries: poolOf(counts).entries}, nil
}

// MustSpentHitDice is NewSpentHitDice for known-valid input. It panics with the
// error NewSpentHitDice returns, and only then.
func MustSpentHitDice(counts map[content.HitPointDie]int) SpentHitDice {
	s, err := NewSpentHitDice(counts)
	if err != nil {
		panic(err)
	}
	return s
}

// Entries returns a new slice of the spent counts, ordered by ascending faces.
func (s SpentHitDice) Entries() []HitDiceCount { return slices.Clone(s.entries) }

// Count returns how many dice of type die are spent.
func (s SpentHitDice) Count(die content.HitPointDie) int {
	return HitDicePool(s).Count(die)
}

// Total returns the number of dice spent.
func (s SpentHitDice) Total() int { return HitDicePool(s).Total() }

// Equal reports whether other is a SpentHitDice holding the same counts. It
// makes SpentHitDice a FeatureState.
func (s SpentHitDice) Equal(other FeatureState) bool {
	o, ok := other.(SpentHitDice)
	return ok && slices.Equal(s.entries, o.entries)
}

// HitDieSpendRejection is why a Hit Point Die could not be spent. The zero value
// is not a rejection.
type HitDieSpendRejection int

// The reasons a spend is rejected, in the order they are reported when several
// apply.
const (
	// NotAShortRestBenefit: the benefits being claimed are those of a Long Rest
	// (Spend only).
	NotAShortRestBenefit HitDieSpendRejection = iota + 1
	// SpentExceedsPool: a spent count is above the pool's count of that die
	// (host-stored state that disagrees with the pool).
	SpentExceedsPool
	// NotInPool: the pool holds no die of that size.
	NotInPool
	// NoneRemaining: every die of that size is spent.
	NoneRemaining
)

// String returns the snake_case name of r, or "unknown".
func (r HitDieSpendRejection) String() string {
	switch r {
	case NotAShortRestBenefit:
		return "not_a_short_rest_benefit"
	case SpentExceedsPool:
		return "spent_exceeds_pool"
	case NotInPool:
		return "not_in_pool"
	case NoneRemaining:
		return "none_remaining"
	}
	return "unknown"
}

// HitDieSpendResult is the outcome of spending one Hit Point Die. Variants:
// HitDieSpendAllowed, HitDieSpendRejected.
type HitDieSpendResult interface{ isHitDieSpendResult() }

// HitDieSpendAllowed: the die was spent; Spent is the new set.
type HitDieSpendAllowed struct{ spent SpentHitDice }

// Spent returns the spent dice after the spend.
func (r HitDieSpendAllowed) Spent() SpentHitDice { return r.spent }

// HitDieSpendRejected: the die was not spent and nothing changed.
type HitDieSpendRejected struct{ reason HitDieSpendRejection }

// Reason returns why the spend was refused.
func (r HitDieSpendRejected) Reason() HitDieSpendRejection { return r.reason }

func (HitDieSpendAllowed) isHitDieSpendResult()  {}
func (HitDieSpendRejected) isHitDieSpendResult() {}

// Spend spends one die of type die from pool during a rest whose benefits are
// those of benefitsOf (SRD 5.2.1, "Short Rest", "Spend Hit Point Dice", p. 187).
// Only the benefits of a ShortRest allow it; a finished Short Rest and a Long
// Rest that earned them both claim ShortRest. When several reasons apply the
// reported one follows the order NotAShortRestBenefit, SpentExceedsPool,
// NotInPool, NoneRemaining. The receiver is unchanged. It returns no error and
// never panics.
func (s SpentHitDice) Spend(pool HitDicePool, die content.HitPointDie, benefitsOf RestKind) HitDieSpendResult {
	if benefitsOf != ShortRest {
		return HitDieSpendRejected{reason: NotAShortRestBenefit}
	}
	return s.SpendOutsideRest(pool, die)
}

// SpendOutsideRest spends one die as Spend does, but without a rest and without
// the NotAShortRestBenefit check. It is for the one spend that is not a rest
// benefit, the Warlock's Lifedrinker (SRD 5.2.1, "Lifedrinker", p. 73), so the
// die is recorded here and cannot be spent again at the next Short Rest. The
// receiver is unchanged. It returns no error and never panics.
func (s SpentHitDice) SpendOutsideRest(pool HitDicePool, die content.HitPointDie) HitDieSpendResult {
	for _, e := range s.entries {
		if e.Count > pool.Count(e.Die) {
			return HitDieSpendRejected{reason: SpentExceedsPool}
		}
	}
	switch left := pool.Count(die) - s.Count(die); {
	case pool.Count(die) == 0:
		return HitDieSpendRejected{reason: NotInPool}
	case left == 0:
		return HitDieSpendRejected{reason: NoneRemaining}
	}
	counts := make(map[content.HitPointDie]int, len(s.entries)+1)
	for _, e := range s.entries {
		counts[e.Die] = e.Count
	}
	counts[die]++
	return HitDieSpendAllowed{spent: SpentHitDice(poolOf(counts))}
}
