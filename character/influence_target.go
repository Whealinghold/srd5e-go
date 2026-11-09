package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// InfluenceTarget is the monster an Influence action is aimed at: its attitude,
// Intelligence score, creature type and whether it understands the speaker (SRD
// 5.2.1, "Influence [Action]", p. 184). A monster is "a creature controlled by
// the GM" and a nonplayer character is a monster with a personal name (p. 185),
// so one type serves both. Build it with NewInfluenceTarget; the zero value is
// invalid (see Valid). Comparable and safe for concurrent use.
type InfluenceTarget struct {
	attitude      Attitude
	intelligence  ability.Score
	creatureType  content.CreatureType
	comprehension InfluenceComprehension
}

// TargetOption configures NewInfluenceTarget.
type TargetOption func(*targetConfig)

type targetConfig struct{ attitude Attitude }

// WithAttitude sets the target's attitude; the default is MonsterDefaultAttitude.
func WithAttitude(a Attitude) TargetOption {
	return func(c *targetConfig) { c.attitude = a }
}

// NewInfluenceTarget returns the target with the given Intelligence score,
// creature type and comprehension.
//
// It returns an error matching srderr.ErrInvalidInput, and the zero target, for
// a zero score, an invalid creature type or comprehension, or an invalid
// attitude set with WithAttitude. The score's range is the ability package's
// concern. It never panics.
func NewInfluenceTarget(intelligence ability.Score, creatureType content.CreatureType, comprehension InfluenceComprehension, opts ...TargetOption) (InfluenceTarget, error) {
	cfg := targetConfig{attitude: MonsterDefaultAttitude}
	for _, o := range opts {
		o(&cfg)
	}
	switch {
	case intelligence.IsZero():
		return InfluenceTarget{}, invalidInput("influence target: zero Intelligence score")
	case !creatureType.Valid():
		return InfluenceTarget{}, invalidInput("influence target: invalid creature type")
	case !comprehension.Valid():
		return InfluenceTarget{}, invalidInput("influence target: invalid comprehension")
	case !cfg.attitude.Valid():
		return InfluenceTarget{}, invalidInput("influence target: invalid attitude")
	}
	return InfluenceTarget{attitude: cfg.attitude, intelligence: intelligence, creatureType: creatureType, comprehension: comprehension}, nil
}

// Valid reports whether t was built by NewInfluenceTarget.
func (t InfluenceTarget) Valid() bool { return t.attitude.Valid() }

// Attitude returns the target's attitude.
func (t InfluenceTarget) Attitude() Attitude { return t.attitude }

// Intelligence returns the target's Intelligence score.
func (t InfluenceTarget) Intelligence() ability.Score { return t.intelligence }

// CreatureType returns the target's creature type.
func (t InfluenceTarget) CreatureType() content.CreatureType { return t.creatureType }

// Comprehension returns whether the target understands the speaker.
func (t InfluenceTarget) Comprehension() InfluenceComprehension { return t.comprehension }
