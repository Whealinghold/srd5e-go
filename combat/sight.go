package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SightScene is everything CanSee needs: the distance, the line of sight, the
// space of the thing seen, the observer's senses and the effective conditions of
// observer and target (SRD 5.2.1, "Blinded", "Invisible", pp. 177, 184). Only
// blinded in the observer's set and invisible in the target's set matter; the
// other kinds are ignored. Immutable; the sets are copied in and out. The zero
// SightScene is invalid and CanSee reports CannotSee for it. Safe for concurrent use.
type SightScene struct {
	distance                             int
	line                                 LineOfSight
	space                                SpaceVisibility
	senses                               SenseRanges
	observerConditions, targetConditions []content.ConditionKind
}

// NewSightScene returns the scene; both condition sets are copied. It returns an
// error matching srderr.ErrInvalidInput for a negative distance (a *SenseRangeError),
// an invalid line of sight, an invalid space or an undefined condition kind in
// either set, and a LimitExceededError for more than 256 conditions in either set. It never panics.
func NewSightScene(distanceFeet int, line LineOfSight, space SpaceVisibility, senses SenseRanges, observerConditions, targetConditions []content.ConditionKind) (SightScene, error) {
	switch {
	case distanceFeet < 0:
		return SightScene{}, &SenseRangeError{SenseRangeReasonNegativeDistance}
	case line != LineOfSightClear && line != LineOfSightBlocked:
		return SightScene{}, invalidInput("sight scene line of sight")
	case !space.Valid():
		return SightScene{}, &SpaceVisibilityError{SpaceVisibilityReasonInvalidLight}
	}
	for _, c := range [...]struct {
		name string
		n    int
	}{{"observer conditions", len(observerConditions)}, {"target conditions", len(targetConditions)}} {
		if err := overCap(c.name, c.n, maxInputList); err != nil {
			return SightScene{}, err
		}
	}
	for _, k := range slices.Concat(observerConditions, targetConditions) {
		if !k.Valid() {
			return SightScene{}, invalidInput("sight scene condition kind")
		}
	}
	return SightScene{distanceFeet, line, space, senses, slices.Clone(observerConditions), slices.Clone(targetConditions)}, nil
}

// ObserverConditions returns a copy of the observer's condition set.
func (s SightScene) ObserverConditions() []content.ConditionKind {
	return append([]content.ConditionKind{}, s.observerConditions...)
}

// TargetConditions returns a copy of the target's condition set.
func (s SightScene) TargetConditions() []content.ConditionKind {
	return append([]content.ConditionKind{}, s.targetConditions...)
}

// SightRules is the stateless domain service that decides whether an observer can
// see a target. Pure, synchronous and silent; the zero value is ready to use. Safe
// for concurrent use.
type SightRules struct{}

// CanSee returns SightRelationCanSee or SightRelationCannotSee. It decides in
// order, stopping at the first that applies: a blocked line is CannotSee;
// Blindsight within range is CanSee, even if the observer is Blinded; a Blinded
// observer is CannotSee; an Invisible target is CannotSee unless Truesight is within
// range; Heavily Obscured per SpecialSenseRules.EffectiveObscurement is CannotSee;
// otherwise CanSee. Lightly Obscured never stops sight, and Tremorsense is not
// sight (SRD 5.2.1, "Blindsight", "Blinded", p. 177; "Heavily Obscured", p. 182;
// "Invisible", p. 184; "Truesight", "Tremorsense", p. 190). The zero SightScene is CannotSee.
func (SightRules) CanSee(s SightScene) SightRelation {
	switch {
	case !s.space.Valid() || s.line == LineOfSightBlocked:
		return SightRelationCannotSee
	case s.senses.within(content.SenseBlindsight, s.distance):
		return SightRelationCanSee
	case slices.Contains(s.observerConditions, content.ConditionBlinded):
		return SightRelationCannotSee
	case slices.Contains(s.targetConditions, content.ConditionInvisible) && !s.senses.within(content.SenseTruesight, s.distance):
		return SightRelationCannotSee
	case effectiveObscurement(s.space, s.distance, s.senses) == content.ObscurementHeavily:
		return SightRelationCannotSee
	}
	return SightRelationCanSee
}
