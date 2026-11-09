package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// sentientSaveRoll makes the bearer's Charisma saving throw of a conflict, one
// way for the three sentient use cases. It composes the pure d20 functions,
// holds no rule and records no signal.
type sentientSaveRoll struct{ roller Roller }

// roll makes the save at SentientConflict.DC(itemCharisma): a rolling attempt
// makes one request for 1d20 (2d20 under Advantage or Disadvantage); ChooseToFail
// and AutomaticFailure make none. A roller failure comes back as a
// *srderr.PortError.
func (s sentientSaveRoll) roll(attempt d20.SaveAttempt, itemCharisma ability.Score) (SentientSave, error) {
	switch a := attempt.(type) {
	case d20.RollAttempt:
		mode := modifier.Resolve(a.Circumstances...)
		expr, err := dice.NewExpression(mode.DiceCount(), dice.D20(), 0)
		if err != nil { // unreachable: the count is 1 or 2 and the die is a d20
			return nil, err
		}
		res, err := dice.Roll(s.roller, expr)
		if err != nil {
			return nil, err
		}
		faces := make([]int, 0, mode.DiceCount())
		for _, r := range res.Rolls() {
			faces = append(faces, r.Value())
		}
		roll, err := d20.NewRoll(mode, faces) // unreachable error: dice.Roll vouches for the faces
		if err != nil {
			return nil, err
		}
		check, err := d20.ResolveCheck(d20.SavingThrow, roll, a.Modifiers, SentientConflict{}.DC(itemCharisma))
		return SentientSaveRolled{Check: check}, err
	case d20.ChooseToFail:
		return s.forced(d20.ChosenToFail)
	default: // d20.AutomaticFailure; the use cases reject other attempts first
		return s.forced(d20.Automatic)
	}
}

func (sentientSaveRoll) forced(reason d20.ForcedFailureReason) (SentientSave, error) {
	f, err := d20.FailWithoutRoll(d20.SavingThrow, reason)
	return SentientSaveForced{Failure: f}, err
}
