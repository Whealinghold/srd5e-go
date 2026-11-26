package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestMovementChecksMove(t *testing.T) {
	t.Parallel()
	crawl, walk := trkMode(t, 10, combat.MovementModeCrawl), trkWalk(t, 10)
	speedZero := combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionSpeedZeroedByCondition}
	crawlOnly := combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionProneAllowsOnlyCrawl}
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		segments                []combat.MoveSegment
		want                    combat.ResolutionRejection
	}{
		{"no conditions", resSpec + "4.9.1 ordinary move", "happy", nil, []combat.MoveSegment{walk}, nil},
		{"Grappled", resSpec + "FR-057", "unhappy (rule interaction)", []combat.Condition{combat.Grappled{Grappler: cid("g")}}, []combat.MoveSegment{walk}, speedZero},
		{"Restrained", resSpec + "FR-057", "unhappy (rule interaction)", []combat.Condition{combat.Restrained{}}, []combat.MoveSegment{walk}, speedZero},
		{"Paralyzed", resSpec + "FR-057", "unhappy (rule interaction)", []combat.Condition{combat.Paralyzed{}}, []combat.MoveSegment{walk}, speedZero},
		{"Petrified", resSpec + "FR-057", "unhappy (rule interaction)", []combat.Condition{combat.Petrified{}}, []combat.MoveSegment{walk}, speedZero},
		{"Unconscious is Speed 0 before Prone", resSpec + "FR-057, FR-058", "unhappy (rule interaction)", []combat.Condition{combat.Unconscious{}}, []combat.MoveSegment{walk}, speedZero},
		{"Exhaustion reduces and does not zero", resSpec + "4.9.1 Speed 0 by condition", "unhappy (rule interaction)",
			[]combat.Condition{combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)}}, []combat.MoveSegment{walk}, nil},
		{"Prone walking", resSpec + "FR-058", "unhappy (rule interaction)", []combat.Condition{combat.Prone{}}, []combat.MoveSegment{walk}, crawlOnly},
		{"Prone crawl then walk", resSpec + "FR-058", "unhappy (rule interaction)", []combat.Condition{combat.Prone{}}, []combat.MoveSegment{crawl, walk}, crawlOnly},
		{"Prone crawling", resSpec + "4.9.1 Prone move", "unhappy (rule interaction)", []combat.Condition{combat.Prone{}}, []combat.MoveSegment{crawl}, nil},
		{"Poisoned moves", resSpec + "FR-057", "unhappy (rule interaction)", []combat.Condition{combat.Poisoned{}}, []combat.MoveSegment{walk}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.MovementChecks{}).Move(condSet(t, c.conds), 30, c.segments)
			if err != nil || got != c.want {
				t.Errorf("= %#v, %v, want %#v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	if _, err := (combat.MovementChecks{}).Move(condSet(t, nil), -1, []combat.MoveSegment{walk}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative Speed gives %v, want ErrInvalidInput [FR-096]", err)
	}
}

// Requirement: per case
// Path: per case.
func TestMovementChecksDropProne(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		want                    combat.ResolutionRejection
	}{
		{"standing", resSpec + "FR-060", "happy", nil, nil},
		{"already Prone", resSpec + "FR-060", "unhappy (state)", []combat.Condition{combat.Prone{}}, combat.PostureMismatchRejection{Mismatch: combat.PostureMismatchAlreadyProne}},
		{"Unconscious is Prone by implication", resSpec + "FR-060", "unhappy (state)", []combat.Condition{combat.Unconscious{}},
			combat.PostureMismatchRejection{Mismatch: combat.PostureMismatchAlreadyProne}},
		{"Grappled", resSpec + "FR-124", "unhappy (rule interaction)", []combat.Condition{combat.Grappled{Grappler: cid("g")}},
			combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionSpeedZeroedByCondition}},
		{"Poisoned", resSpec + "FR-060", "happy", []combat.Condition{combat.Poisoned{}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.MovementChecks{}).DropProne(condSet(t, c.conds), 30)
			if err != nil || got != c.want {
				t.Errorf("= %#v, %v, want %#v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestMovementChecksStandUp(t *testing.T) {
	t.Parallel()
	notProne := combat.PostureMismatchRejection{Mismatch: combat.PostureMismatchNotProne}
	cannot := combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionCannotRightItself}
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		base                    int
		movement                combat.ProneMovement
		want                    combat.ResolutionRejection
	}{
		{"Prone, Speed 30", resSpec + "4.9.1 stand up", "happy", []combat.Condition{combat.Prone{}}, 30, combat.CrawlOrRightItself{CostFeet: 15}, nil},
		{"Prone, Speed 25 rounds down", resSpec + "4.9.1 stand up", "unhappy (boundary)", []combat.Condition{combat.Prone{}}, 25, combat.CrawlOrRightItself{CostFeet: 12}, nil},
		{"Prone, Speed 1", resSpec + "4.9.1 stand up", "unhappy (boundary)", []combat.Condition{combat.Prone{}}, 1, combat.CrawlOrRightItself{CostFeet: 0}, nil},
		{"Prone, Speed 0", resSpec + "4.9.1 stand up", "unhappy (boundary)", []combat.Condition{combat.Prone{}}, 0, nil, cannot},
		{"Unconscious", resSpec + "FR-061", "unhappy (boundary)", []combat.Condition{combat.Unconscious{}}, 30, nil, cannot},
		{"standing", resSpec + "FR-061", "unhappy (state)", nil, 30, nil, notProne},
		{"Prone and Grappled", resSpec + "FR-061", "unhappy (rule interaction)", []combat.Condition{combat.Prone{}, combat.Grappled{Grappler: cid("g")}}, 30, nil, cannot},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m, got, err := (combat.MovementChecks{}).StandUp(condSet(t, c.conds), c.base)
			if err != nil || got != c.want || m != c.movement {
				t.Errorf("= %#v, %#v, %v, want %#v and %#v [%s]", m, got, err, c.movement, c.want, c.requirement)
			}
		})
	}
	for _, conds := range [][]combat.Condition{nil, {combat.Prone{}}} {
		if _, _, err := (combat.MovementChecks{}).StandUp(condSet(t, conds), -1); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("a negative Speed gives %v with %d conditions, want ErrInvalidInput [FR-143]", err, len(conds))
		}
	}
}
