package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
)

// The bounds of the cap on an ability score change (SRD 5.2.1, "Ability Scores",
// p. 5).
const (
	// DefaultScoreCap is the highest score unless a feature says otherwise.
	DefaultScoreCap = 20
	// HighestScoreCap is the highest score a feature can allow.
	HighestScoreCap = 30
)

// AbilityScoreChangeRejection is the closed set of reasons ChangeAbilityScore
// refuses a change. Variants: AbilityScoreChangeExceedsMaximum,
// AbilityScoreChangeConstitutionDecrease.
type AbilityScoreChangeRejection interface{ isAbilityScoreChangeRejection() }

// AbilityScoreChangeExceedsMaximum is the rejection of a score above the cap.
type AbilityScoreChangeExceedsMaximum struct{ Cap int }

// AbilityScoreChangeConstitutionDecrease is the rejection of a change that
// lowers the Constitution modifier: the SRD gives no Hit Point rule for it.
type AbilityScoreChangeConstitutionDecrease struct{}

func (AbilityScoreChangeExceedsMaximum) isAbilityScoreChangeRejection()       {}
func (AbilityScoreChangeConstitutionDecrease) isAbilityScoreChangeRejection() {}

// ChangeAbilityScore sets one ability to score (SRD 5.2.1, "Gaining a Level"
// step 5, p. 23). A raised Constitution modifier adds the rise times the total
// level to the stored Hit Point maximum, with a HitPointMaximumChanged event
// after the AbilityScoreChanged event. Setting the current score is a
// successful no-op: an equal Character and no events. Otherwise, in this order,
// a score above maximumScore is rejected with AbilityScoreChangeExceedsMaximum
// and a lowered Constitution modifier with AbilityScoreChangeConstitutionDecrease.
// The receiver is unchanged.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, an
// invalid ability or score, or a maximumScore outside 20 to 30. It never panics.
func (c Character) ChangeAbilityScore(which ability.Ability, score ability.Score, maximumScore int) (Transition[AbilityScoreChangeRejection], error) {
	if c.IsZero() || !which.Valid() || score.IsZero() {
		return nil, invalidInput("ability score change of a zero character, ability or score")
	}
	if maximumScore < DefaultScoreCap || maximumScore > HighestScoreCap {
		return nil, invalidInput("ability score cap outside 20 to 30")
	}
	old := c.scores.Score(which)
	if old == score {
		return changed[AbilityScoreChangeRejection](c), nil
	}
	if score.Value() > maximumScore {
		return rejected[AbilityScoreChangeRejection](AbilityScoreChangeExceedsMaximum{Cap: maximumScore}), nil
	}
	rise := score.Modifier().Value() - old.Modifier().Value()
	constitution := which == ability.Constitution
	if constitution && rise < 0 {
		return rejected[AbilityScoreChangeRejection](AbilityScoreChangeConstitutionDecrease{}), nil
	}
	scores, err := c.scores.With(which, score)
	if err != nil { // unreachable: which and score were checked above
		return nil, err
	}
	next := c
	next.scores = scores
	events := []Event{AbilityScoreChanged{id: c.id, ability: which, previous: old, current: score}}
	if constitution && rise > 0 {
		next.hitPoints = c.hitPoints.Plus(rise * c.TotalLevel().n)
		events = append(events, HitPointMaximumChanged{id: c.id, previous: c.hitPoints, current: next.hitPoints})
	}
	return changed[AbilityScoreChangeRejection](next, events...), nil
}
