package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// StabilizingCheck is the check of the Help action that stabilizes a creature:
// a DC 10 Wisdom (Medicine) check (SRD 5.2.1, "Stabilizing a Character", p. 18).
// It is a stateless rule; the zero value is ready to use.
type StabilizingCheck struct{}

// DC returns the Difficulty Class, 10.
func (StabilizingCheck) DC() d20.DifficultyClass { return d20.DCEasy }

// Ability returns Wisdom.
func (StabilizingCheck) Ability() ability.Ability { return ability.Wisdom }

// Skill returns Medicine.
func (StabilizingCheck) Skill() content.Skill { return content.SkillMedicine }

// StableRecoveryDelay is how long a Stable creature stays at 0 Hit Points before
// it regains 1: 1 to 4 hours (SRD 5.2.1, "Stabilizing a Character", p. 18). The
// zero value is invalid.
type StableRecoveryDelay struct{ hours int }

// NewStableRecoveryDelay returns the delay of hours. It returns an error
// matching srderr.ErrInvalidInput for hours outside 1 to 4.
func NewStableRecoveryDelay(hours int) (StableRecoveryDelay, error) {
	if hours < 1 || hours > 4 {
		return StableRecoveryDelay{}, invalidInput("stable recovery delay outside 1 to 4 hours")
	}
	return StableRecoveryDelay{hours}, nil
}

// Hours returns the delay in hours, 1 to 4.
func (d StableRecoveryDelay) Hours() int { return d.hours }
