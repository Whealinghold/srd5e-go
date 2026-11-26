package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// SenseRangeReason says why a sense range or a distance was rejected. The zero
// value is not a reason.
type SenseRangeReason int

// The reasons.
const (
	// SenseRangeReasonUnknownSense: the sense is not one of the four special senses.
	SenseRangeReasonUnknownSense SenseRangeReason = iota + 1
	// SenseRangeReasonNegativeRange: the range is below 0 feet.
	SenseRangeReasonNegativeRange
	// SenseRangeReasonNegativeDistance: the distance is below 0 feet.
	SenseRangeReasonNegativeDistance
)

// String returns the snake_case name of r or "unknown".
func (r SenseRangeReason) String() string {
	return enumName([]string{"unknown_sense", "negative_range", "negative_distance"}, int(r))
}

// SenseRangeError reports a rejected sense range or distance. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type SenseRangeError struct{ Reason SenseRangeReason }

// Error renders the error. The text is not a contract.
func (e *SenseRangeError) Error() string { return "combat: invalid sense range: " + e.Reason.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *SenseRangeError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SenseRangeError with the same Reason.
func (e *SenseRangeError) Is(target error) bool {
	t, ok := target.(*SenseRangeError)
	return ok && t != nil && e != nil && *t == *e
}

// SenseRange is a special sense with the range in feet it works within (SRD
// 5.2.1, "Special Senses", p. 11). Immutable and comparable; the zero SenseRange
// is invalid (its sense is undefined). Safe for concurrent use.
type SenseRange struct {
	sense     content.SpecialSense
	rangeFeet int
	set       bool
}

// NewSenseRange returns the range of a sense; a range of 0 is valid. It returns a
// *SenseRangeError (matching srderr.ErrInvalidInput) for an undefined sense or a
// negative range. It never panics.
func NewSenseRange(sense content.SpecialSense, rangeFeet int) (SenseRange, error) {
	switch {
	case !sense.Valid():
		return SenseRange{}, &SenseRangeError{SenseRangeReasonUnknownSense}
	case rangeFeet < 0:
		return SenseRange{}, &SenseRangeError{SenseRangeReasonNegativeRange}
	}
	return SenseRange{sense, rangeFeet, true}, nil
}

// MustSenseRange is NewSenseRange for literals known to be valid. It panics with
// the error NewSenseRange returns, and only then.
func MustSenseRange(sense content.SpecialSense, rangeFeet int) SenseRange {
	r, err := NewSenseRange(sense, rangeFeet)
	if err != nil {
		panic(err)
	}
	return r
}

// Sense returns the special sense.
func (r SenseRange) Sense() content.SpecialSense { return r.sense }

// RangeFeet returns the range in feet.
func (r SenseRange) RangeFeet() int { return r.rangeFeet }

// SenseRanges is the special senses a creature has, at most one range per sense.
// It holds a fixed array, so a copy is independent and the value is comparable;
// the zero SenseRanges has no sense. Safe for concurrent use.
type SenseRanges struct{ held [4]SenseRange }

// NewSenseRanges returns the senses given. A sense given twice keeps the greater
// range (SRD 5.2.1, "Drow", p. 85); a zero (invalid) SenseRange is ignored. The
// argument is copied. It never fails and never panics.
func NewSenseRanges(ranges ...SenseRange) SenseRanges {
	var s SenseRanges
	for _, r := range ranges {
		if r.set && (!s.held[r.sense-1].set || r.rangeFeet > s.held[r.sense-1].rangeFeet) {
			s.held[r.sense-1] = r
		}
	}
	return s
}

// Range returns the range of sense and whether the creature has it.
func (s SenseRanges) Range(sense content.SpecialSense) (int, bool) {
	if !sense.Valid() {
		return 0, false
	}
	r := s.held[sense-1]
	return r.rangeFeet, r.set
}

// All returns a new slice of the ranges held, in the order of content.SpecialSense.
func (s SenseRanges) All() []SenseRange {
	out := []SenseRange{}
	for _, r := range s.held {
		if r.set {
			out = append(out, r)
		}
	}
	return out
}

// within reports whether the creature has sense and distance is at most its range.
func (s SenseRanges) within(sense content.SpecialSense, distance int) bool {
	r, ok := s.Range(sense)
	return ok && distance <= r
}

// SenseReach says whether a distance is within the range of a sense. The zero
// value is invalid.
type SenseReach int

// The reaches.
const (
	// SenseReachWithinRange: the creature has the sense and the distance is at most its range.
	SenseReachWithinRange SenseReach = iota + 1
	// SenseReachBeyondRange: the creature lacks the sense or the distance is greater than its range.
	SenseReachBeyondRange
)

// String returns "within_range", "beyond_range" or "unknown".
func (r SenseReach) String() string {
	return enumName([]string{"within_range", "beyond_range"}, int(r))
}

func reachOf(ranges SenseRanges, sense content.SpecialSense, distance int) SenseReach {
	if ranges.within(sense, distance) {
		return SenseReachWithinRange
	}
	return SenseReachBeyondRange
}

// ColorPerception says whether colors are seen. The zero value is invalid.
type ColorPerception int

// The color perceptions.
const (
	// ColorPerceptionFullColor: colors are seen as they are.
	ColorPerceptionFullColor ColorPerception = iota + 1
	// ColorPerceptionShadesOfGray: colors are seen as shades of gray.
	ColorPerceptionShadesOfGray
)

// String returns "full_color", "shades_of_gray" or "unknown".
func (c ColorPerception) String() string {
	return enumName([]string{"full_color", "shades_of_gray"}, int(c))
}

// PerceivedLight is the light as an observer perceives it. Immutable and
// comparable; the zero value is invalid.
type PerceivedLight struct {
	level content.LightLevel
	color ColorPerception
}

// Level returns the perceived light level.
func (p PerceivedLight) Level() content.LightLevel { return p.level }

// Color returns whether colors are seen in full or as shades of gray.
func (p PerceivedLight) Color() ColorPerception { return p.color }

// TremorsenseContact says how the Tremorsense observer and the target touch the
// ground. The zero value is invalid.
type TremorsenseContact int

// The contacts.
const (
	// TremorsenseContactSharedSurfaceOrLiquid: both are on the same surface or in the same liquid.
	TremorsenseContactSharedSurfaceOrLiquid TremorsenseContact = iota + 1
	// TremorsenseContactNoSharedSurface: they do not touch the same surface or liquid.
	TremorsenseContactNoSharedSurface
	// TremorsenseContactTargetInTheAir: the target is not on a surface or in a liquid.
	TremorsenseContactTargetInTheAir
)

// DetectedThing is what Tremorsense is asked to find. The zero value is invalid.
type DetectedThing int

// The things.
const (
	// DetectedThingCreature: a creature.
	DetectedThingCreature DetectedThing = iota + 1
	// DetectedThingMovingObject: an object that moves.
	DetectedThingMovingObject
	// DetectedThingStationaryObject: an object that does not move.
	DetectedThingStationaryObject
)

// TremorsenseResult says whether Tremorsense finds the target. The zero value is invalid.
type TremorsenseResult int

// The results.
const (
	// TremorsenseResultPinpointed: the observer knows where the target is.
	TremorsenseResultPinpointed TremorsenseResult = iota + 1
	// TremorsenseResultNotDetected: Tremorsense tells the observer nothing.
	TremorsenseResultNotDetected
)

// String returns "pinpointed", "not_detected" or "unknown".
func (r TremorsenseResult) String() string {
	return enumName([]string{"pinpointed", "not_detected"}, int(r))
}

// TruesightScene holds the distance and senses of a Truesight query. Immutable;
// the zero value is a valid scene of distance 0 and no senses. Safe for concurrent use.
type TruesightScene struct {
	distance int
	senses   SenseRanges
}

// NewTruesightScene returns the scene. It returns a *SenseRangeError (matching
// srderr.ErrInvalidInput) for a negative distance. It never panics.
func NewTruesightScene(distanceFeet int, senses SenseRanges) (TruesightScene, error) {
	if distanceFeet < 0 {
		return TruesightScene{}, &SenseRangeError{SenseRangeReasonNegativeDistance}
	}
	return TruesightScene{distanceFeet, senses}, nil
}

func (t TruesightScene) truesight() bool {
	return t.senses.within(content.SenseTruesight, t.distance)
}

// IllusionAppearance is how a Visual Illusion looks to a Truesight observer. The zero value is invalid.
type IllusionAppearance int

// The appearances.
const (
	// IllusionAppearanceTransparent: the illusion is seen through.
	IllusionAppearanceTransparent IllusionAppearance = iota + 1
	// IllusionAppearanceAsCreated: the illusion looks as it was created.
	IllusionAppearanceAsCreated
)

// IllusionSave says how a save against a Visual Illusion is made. The zero value is invalid.
type IllusionSave int

// The saves.
const (
	// IllusionSaveAutomaticSuccess: the save succeeds without a roll.
	IllusionSaveAutomaticSuccess IllusionSave = iota + 1
	// IllusionSaveRollNormally: the save is rolled as usual.
	IllusionSaveRollNormally
)

// TrueFormPerception says whether a transformed creature's true form is seen. The zero value is invalid.
type TrueFormPerception int

// The perceptions.
const (
	// TrueFormPerceptionDiscerned: the true form is seen.
	TrueFormPerceptionDiscerned TrueFormPerception = iota + 1
	// TrueFormPerceptionNotDiscerned: the true form is not seen.
	TrueFormPerceptionNotDiscerned
)

// EtherealView says whether the Ethereal Plane is seen. The zero value is invalid.
type EtherealView int

// The views.
const (
	// EtherealViewSeen: the observer sees into the Ethereal Plane.
	EtherealViewSeen EtherealView = iota + 1
	// EtherealViewNotSeen: the observer does not.
	EtherealViewNotSeen
)

// IllusionFacts is the host's fact about a Visual Illusion. Construct it with a keyed literal.
type IllusionFacts struct{ DeceivesSight bool }

// TrueFormFacts is the host's facts about a transformed creature. Construct it with a keyed literal.
type TrueFormFacts struct {
	TransformedByMagic bool
	ObserverSeesIt     SightRelation
}

// SpecialSenseRules is the stateless domain service for the four special senses
// (SRD 5.2.1, "Special Senses", p. 11). Its methods are pure, synchronous and
// silent. The zero value is ready to use. Safe for concurrent use.
type SpecialSenseRules struct{}

// Reach says whether distanceFeet is within the range of sense: WithinRange when
// the creature has the sense and the distance is at most its range (SRD 5.2.1,
// "Darkvision", p. 180). It returns a *SenseRangeError (matching
// srderr.ErrInvalidInput) for a negative distance. It never panics.
func (SpecialSenseRules) Reach(sense content.SpecialSense, distanceFeet int, ranges SenseRanges) (SenseReach, error) {
	if distanceFeet < 0 {
		return 0, &SenseRangeError{SenseRangeReasonNegativeDistance}
	}
	return reachOf(ranges, sense, distanceFeet), nil
}

func perceivedLight(space SpaceVisibility, distance int, ranges SenseRanges) PerceivedLight {
	p := PerceivedLight{space.light, ColorPerceptionFullColor}
	if !ranges.within(content.SenseDarkvision, distance) {
		return p
	}
	switch {
	case space.light == content.LightDim:
		p.level = content.LightBright
	case space.light == content.LightDarkness && space.kind != content.DarknessMagical:
		p.level, p.color = content.LightDim, ColorPerceptionShadesOfGray
	}
	return p
}

// PerceivedLight returns the light an observer with ranges perceives in space at
// distanceFeet: within Darkvision range Dim Light is Bright Light and nonmagical
// Darkness is Dim Light seen in shades of gray; magical Darkness and Bright Light
// are unchanged, and so is everything beyond range or without Darkvision (SRD
// 5.2.1, "Darkvision", p. 180; "Darkness", p. 122). It returns a *SenseRangeError
// (matching srderr.ErrInvalidInput) for a negative distance or an invalid space.
func (SpecialSenseRules) PerceivedLight(space SpaceVisibility, distanceFeet int, ranges SenseRanges) (PerceivedLight, error) {
	if err := checkSpace(space, distanceFeet); err != nil {
		return PerceivedLight{}, err
	}
	return perceivedLight(space, distanceFeet, ranges), nil
}

func checkSpace(space SpaceVisibility, distance int) error {
	switch {
	case distance < 0:
		return &SenseRangeError{SenseRangeReasonNegativeDistance}
	case !space.Valid():
		return &SpaceVisibilityError{SpaceVisibilityReasonInvalidLight}
	}
	return nil
}

func effectiveObscurement(space SpaceVisibility, distance int, ranges SenseRanges) content.Obscurement {
	seen := perceivedLight(space, distance, ranges)
	least := max(LightRules{}.ObscurementOf(seen.level), space.other)
	if space.light == content.LightDarkness && ranges.within(content.SenseTruesight, distance) {
		least = min(least, space.other)
	}
	if ranges.within(content.SenseBlindsight, distance) {
		least = content.ObscurementNone
	}
	return least
}

// EffectiveObscurement returns the obscurement an observer meets in space at
// distanceFeet, the least severe of: the more severe of the obscurement of the
// perceived light and the other obscurement; within Truesight range in Darkness,
// the other obscurement alone; within Blindsight range, none. Tremorsense never
// changes it (SRD 5.2.1, "Darkvision", "Blindsight", "Truesight", "Tremorsense",
// pp. 177, 180, 190). It returns a *SenseRangeError (matching srderr.ErrInvalidInput)
// for a negative distance or an invalid space.
func (SpecialSenseRules) EffectiveObscurement(space SpaceVisibility, distanceFeet int, ranges SenseRanges) (content.Obscurement, error) {
	if err := checkSpace(space, distanceFeet); err != nil {
		return 0, err
	}
	return effectiveObscurement(space, distanceFeet, ranges), nil
}

// Tremorsense returns Pinpointed when the reach is WithinRange, the observer and
// target share a surface or liquid and the target is a creature or a moving
// object, and NotDetected otherwise (SRD 5.2.1, "Tremorsense", p. 190). It is not
// sight: SightRules.CanSee ignores it.
func (SpecialSenseRules) Tremorsense(reach SenseReach, contact TremorsenseContact, thing DetectedThing) TremorsenseResult {
	if reach == SenseReachWithinRange && contact == TremorsenseContactSharedSurfaceOrLiquid &&
		(thing == DetectedThingCreature || thing == DetectedThingMovingObject) {
		return TremorsenseResultPinpointed
	}
	return TremorsenseResultNotDetected
}

// SeesInvisible reports whether the observer within Truesight range sees a
// creature or object that has Invisible (SRD 5.2.1, "Truesight", p. 190).
func (SpecialSenseRules) SeesInvisible(scene TruesightScene) bool { return scene.truesight() }

// IllusionAppearance returns Transparent for a Visual Illusion that deceives sight
// seen within Truesight range, else AsCreated (SRD 5.2.1, "Truesight", p. 190).
func (SpecialSenseRules) IllusionAppearance(scene TruesightScene, facts IllusionFacts) IllusionAppearance {
	if facts.DeceivesSight && scene.truesight() {
		return IllusionAppearanceTransparent
	}
	return IllusionAppearanceAsCreated
}

// IllusionSave returns AutomaticSuccess for a save against a Visual Illusion that
// deceives sight within Truesight range, else RollNormally. It is a reported value:
// the caller omits the d20 (SRD 5.2.1, "Truesight", p. 190).
func (SpecialSenseRules) IllusionSave(scene TruesightScene, facts IllusionFacts) IllusionSave {
	if facts.DeceivesSight && scene.truesight() {
		return IllusionSaveAutomaticSuccess
	}
	return IllusionSaveRollNormally
}

// TrueForm returns Discerned only within Truesight range for a creature or object
// transformed by magic that the observer sees, else NotDiscerned (SRD 5.2.1,
// "Truesight", p. 190).
func (SpecialSenseRules) TrueForm(scene TruesightScene, facts TrueFormFacts) TrueFormPerception {
	if scene.truesight() && facts.TransformedByMagic && facts.ObserverSeesIt == SightRelationCanSee {
		return TrueFormPerceptionDiscerned
	}
	return TrueFormPerceptionNotDiscerned
}

// EtherealView returns Seen within Truesight range, else NotSeen (SRD 5.2.1,
// "Truesight", p. 190).
func (SpecialSenseRules) EtherealView(scene TruesightScene) EtherealView {
	if scene.truesight() {
		return EtherealViewSeen
	}
	return EtherealViewNotSeen
}
