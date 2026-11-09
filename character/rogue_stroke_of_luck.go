package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// strokeOfLuckLevel is the Rogue level of Stroke of Luck (SRD 5.2.1, "Level 20: Stroke of Luck", p. 63).
const strokeOfLuckLevel = 20

// StrokeOfLuckReason is why Stroke of Luck cannot be used. The values are in check order. The zero value is not a reason.
type StrokeOfLuckReason int

const (
	// StrokeOfLuckLevelTooLow: the Rogue level is below 20.
	StrokeOfLuckLevelTooLow StrokeOfLuckReason = iota + 1
	// StrokeOfLuckAlreadySpent: the feature is spent until a Short or Long Rest finishes.
	StrokeOfLuckAlreadySpent
	// StrokeOfLuckTestNotFailed: the rolled D20 Test did not fail.
	StrokeOfLuckTestNotFailed
	// StrokeOfLuckNoRoll: the failure has no roll to turn into a 20.
	StrokeOfLuckNoRoll
)

// StrokeOfLuckTest is the failed D20 Test Stroke of Luck is used on: the closed set StrokeOfLuckRolledTest and
// StrokeOfLuckUnrolledTest.
//
//sumtype:decl
type StrokeOfLuckTest interface{ isStrokeOfLuckTest() }

// StrokeOfLuckRolledTest is a D20 Test with a roll and its outcome.
type StrokeOfLuckRolledTest struct {
	Roll    d20.Roll
	Outcome d20.Outcome
}

// StrokeOfLuckUnrolledTest is a D20 Test that failed without a roll (a forced or automatic failure).
type StrokeOfLuckUnrolledTest struct{}

func (StrokeOfLuckRolledTest) isStrokeOfLuckTest()   {}
func (StrokeOfLuckUnrolledTest) isStrokeOfLuckTest() {}

// StrokeOfLuckResult is the outcome of StrokeOfLuck.Use: the closed set StrokeOfLuckSpent and StrokeOfLuckNotAvailable.
//
//sumtype:decl
type StrokeOfLuckResult interface{ isStrokeOfLuckResult() }

// StrokeOfLuckSpent means the feature was used: the roll with every die turned into a 20, the new state and the event.
type StrokeOfLuckSpent struct {
	roll  d20.Roll
	state RogueState
	event StrokeOfLuckUsed
}

// Roll returns the roll with every die a 20, so its natural value is 20 under any roll mode. The caller re-resolves the total
// through d20; Stroke of Luck does not itself declare success.
func (r StrokeOfLuckSpent) Roll() d20.Roll { return r.roll }

// State returns the new RogueState, with Stroke of Luck spent.
func (r StrokeOfLuckSpent) State() RogueState { return r.state }

// Event returns the StrokeOfLuckUsed event.
func (r StrokeOfLuckSpent) Event() StrokeOfLuckUsed { return r.event }

// StrokeOfLuckNotAvailable means the feature cannot be used, with every failing reason in check order. The state is unchanged.
type StrokeOfLuckNotAvailable struct{ reasons []StrokeOfLuckReason }

// Reasons returns the failing reasons as a new slice, in check order.
func (r StrokeOfLuckNotAvailable) Reasons() []StrokeOfLuckReason { return slices.Clone(r.reasons) }

func (StrokeOfLuckSpent) isStrokeOfLuckResult()        {}
func (StrokeOfLuckNotAvailable) isStrokeOfLuckResult() {}

// StrokeOfLuck is the Stroke of Luck rule (SRD 5.2.1, "Level 20: Stroke of Luck", p. 63). The zero value is ready to use.
type StrokeOfLuck struct{}

// Use turns the roll of a failed D20 Test into a 20. It needs Rogue level 20, an unspent feature and a rolled test that
// failed. The new roll replaces every die with a 20 (SRD 5.2.1: "turn the roll into a 20", so the natural value is 20 in all
// three roll modes), and the result carries the spent state and a StrokeOfLuckUsed event for the character. Otherwise the
// result lists every failing reason in check order. state is not changed.
func (StrokeOfLuck) Use(id Identifier, level RogueLevel, state RogueState, test StrokeOfLuckTest) StrokeOfLuckResult {
	var reasons []StrokeOfLuckReason
	if level.n < strokeOfLuckLevel {
		reasons = append(reasons, StrokeOfLuckLevelTooLow)
	}
	if state.stroke == StrokeOfLuckStateSpent {
		reasons = append(reasons, StrokeOfLuckAlreadySpent)
	}
	rolled, hasRoll := test.(StrokeOfLuckRolledTest)
	if hasRoll && rolled.Outcome != d20.Failure {
		reasons = append(reasons, StrokeOfLuckTestNotFailed)
	}
	if !hasRoll || rolled.Roll.IsZero() {
		reasons = append(reasons, StrokeOfLuckNoRoll)
	}
	if len(reasons) > 0 {
		return StrokeOfLuckNotAvailable{reasons: reasons}
	}
	roll := rolled.Roll
	for i := range roll.Dice() {
		roll, _ = roll.ReplaceDie(i, 20) // the index is a die of the roll and 20 is a d20 face
	}
	state.stroke = StrokeOfLuckStateSpent
	return StrokeOfLuckSpent{roll: roll, state: state, event: StrokeOfLuckUsed{id: id}}
}
