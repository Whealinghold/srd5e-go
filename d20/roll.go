package d20

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/modifier"
)

const (
	lowestFace  = 1
	highestFace = 20
	maxDice     = 2
)

// Roll holds the d20s rolled for one D20 Test: the roll mode, the faces in
// the order rolled, and the die the mode keeps.
//
// It is an immutable, comparable value. [ReplaceDie] and [Roll.ApplyReroll]
// return new rolls. A roll does not track earlier replacements, so the SRD
// limit of one replaced die per roll ("Interactions with Rerolls", SRD 5.2.1,
// p. 8) is the caller's responsibility. The zero value is not a roll;
// [Roll.IsZero] reports it.
type Roll struct {
	mode    modifier.RollMode
	dice    [maxDice]int
	n       int
	natural int
}

// NewRoll returns the roll of the d20 faces in dice, in the order rolled,
// under mode. The natural value is the one [modifier.RollMode.Select] returns:
// the higher die under Advantage, the lower under Disadvantage, the only die
// otherwise. dice is copied.
//
// It returns the *modifier.InvalidError from Select (matching
// srderr.ErrInvalidInput) when mode is undefined, the count differs from the
// mode's dice count, or a face is outside 1 to 20.
func NewRoll(mode modifier.RollMode, dice []int) (Roll, error) {
	natural, err := mode.Select(dice)
	if err != nil {
		return Roll{}, err
	}
	r := Roll{mode: mode, n: len(dice), natural: natural}
	copy(r.dice[:], dice)
	return r, nil
}

// MustRoll is [NewRoll] for literals known to be valid. It panics with
// the error NewRoll would return.
func MustRoll(mode modifier.RollMode, dice ...int) Roll {
	r, err := NewRoll(mode, dice)
	if err != nil {
		panic(err)
	}
	return r
}

// IsZero reports whether r is the zero value, which holds no dice.
func (r Roll) IsZero() bool { return r.n == 0 }

// Mode returns the roll mode the dice were rolled under.
func (r Roll) Mode() modifier.RollMode { return r.mode }

// Dice returns a copy of the faces in the order rolled: one for normal, two
// under Advantage or Disadvantage.
func (r Roll) Dice() []int { return append([]int(nil), r.dice[:r.n]...) }

// Natural returns the d20 number the roll uses before modifiers.
func (r Roll) Natural() int { return r.natural }

// KeptIndex returns the index in Dice of the kept die: the first whose face
// equals Natural. With equal dice that is 0.
func (r Roll) KeptIndex() int {
	for i, d := range r.dice[:r.n] {
		if d == r.natural {
			return i
		}
	}
	return 0
}

// ReplaceDie returns the roll with the die at index replaced by value, with
// the natural value re-selected ("Interactions with Rerolls", SRD 5.2.1,
// p. 8). r is unchanged.
//
// It returns an *InvalidError with ReasonIndexOutOfRange when index is
// outside the dice (always so for the zero roll) and ReasonFaceOutOfRange
// when value is outside 1 to 20.
func (r Roll) ReplaceDie(index, value int) (Roll, error) {
	if index < 0 || index >= r.n {
		return Roll{}, invalid(ReasonIndexOutOfRange,
			"index "+strconv.Itoa(index)+" not in 0.."+strconv.Itoa(r.n-1))
	}
	if value < lowestFace || value > highestFace {
		return Roll{}, invalid(ReasonFaceOutOfRange, strconv.Itoa(value)+" is not a d20 face")
	}
	faces := r.Dice()
	faces[index] = value
	return NewRoll(r.mode, faces)
}

// ApplyReroll returns the roll the decision leads to: r itself for
// [KeepRoll], and [Roll.ReplaceDie] for [RerollDie]. The new face is used
// even when lower than the old one ("Heroic Inspiration", SRD 5.2.1, p. 8).
//
// It returns an *InvalidError with ReasonUnknownDecision for a nil decision,
// and the ReplaceDie errors for an invalid [RerollDie].
func (r Roll) ApplyReroll(d RerollDecision) (Roll, error) {
	switch d := d.(type) {
	case KeepRoll:
		return r, nil
	case RerollDie:
		return r.ReplaceDie(d.Index, d.Face)
	default:
		return Roll{}, invalid(ReasonUnknownDecision, "decision is nil or not a variant")
	}
}

// RerollDecision is the choice behind a reroll of one d20. The variants are
// [KeepRoll] and [RerollDie], both values; handle them with a type switch.
// Under Advantage or Disadvantage a reroll applies to "only one die, not
// both" (SRD 5.2.1, p. 8), so the decision names the die.
type RerollDecision interface {
	rerollDecision()
}

// KeepRoll is the decision to keep the roll as it is.
type KeepRoll struct{}

// RerollDie is the decision to replace the die at Index with the new face Face.
type RerollDie struct {
	// Index is 0 for the first die rolled, 1 for the second.
	Index int
	// Face is the newly rolled face, 1 to 20.
	Face int
}

func (KeepRoll) rerollDecision()  {}
func (RerollDie) rerollDecision() {}
