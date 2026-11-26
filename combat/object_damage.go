package combat

import "errors"

// DamageThreshold is the least damage from a single attack or effect that harms a
// creature or an object that has one (SRD 5.2.1, "Damage Threshold", p. 180). The
// zero value means no threshold. Comparable.
type DamageThreshold struct{ amount int }

// NewDamageThreshold returns a threshold of amount. It returns an error matching
// srderr.ErrInvalidInput for an amount below 1. It never panics.
func NewDamageThreshold(amount int) (DamageThreshold, error) {
	if amount < 1 {
		return DamageThreshold{}, invalidInput("damage threshold below 1")
	}
	return DamageThreshold{amount}, nil
}

// Amount returns the threshold, 0 for none.
func (t DamageThreshold) Amount() int { return t.amount }

// ThresholdOutcome is what a Damage Threshold did to an instance of damage. The
// zero value is invalid.
type ThresholdOutcome int

// The outcomes.
const (
	// ThresholdNotApplicable: there is no threshold.
	ThresholdNotApplicable ThresholdOutcome = iota + 1
	// ThresholdMet: the instance equals or exceeds the threshold, so all of it is taken.
	ThresholdMet
	// ThresholdNotMet: the instance is below the threshold, so none of it is taken.
	ThresholdNotMet
)

// String returns "not_applicable", "met", "not_met" or "unknown".
func (o ThresholdOutcome) String() string {
	return enumName([]string{"not_applicable", "met", "not_met"}, int(o))
}

// DamageThresholdGate decides whether an instance of damage gets past a Damage
// Threshold. It knows nothing of objects, so a creature with a threshold uses it
// too; the host evaluates it, skips ApplyDamage for ThresholdNotMet and passes the
// whole instance on for ThresholdMet. The zero value is ready to use.
type DamageThresholdGate struct{}

// Evaluate compares the sum of the amounts of damages, before any adjustment,
// Resistance or Vulnerability, with threshold (SRD 5.2.1, "Damage Threshold", p.
// 180): ThresholdNotApplicable for no threshold, ThresholdMet when the sum equals or
// exceeds it, otherwise ThresholdNotMet. The method has no error return, so the caller
// bounds damages (an instance holds at most 64, see NewObjectDamageInstance); it sums
// whatever it is given.
func (DamageThresholdGate) Evaluate(threshold DamageThreshold, damages []Damage) ThresholdOutcome {
	if threshold.amount == 0 {
		return ThresholdNotApplicable
	}
	total := 0
	for _, d := range damages {
		total = satAdd(total, d.amount)
	}
	if total >= threshold.amount {
		return ThresholdMet
	}
	return ThresholdNotMet
}

// ObjectHarm is what is done to an object. Variants: ObjectDamageInstance,
// ObjectBrokenAutomatically.
type ObjectHarm interface{ isObjectHarm() }

// ObjectDamageInstance is one attack or effect's damage to an object: every Damage
// of the instance, and the adjustments that apply to it. Build it with
// NewObjectDamageInstance; the zero value is invalid.
type ObjectDamageInstance struct {
	damages     []Damage
	adjustments []DamageAdjustment
}

// NewObjectDamageInstance returns the instance. The slices are copied. It returns an
// error matching srderr.ErrInvalidInput for no damage, an invalid Damage or a nil
// adjustment, and a LimitExceededError for more than 64 damages or more than 64 adjustments (plan P10). It never
// panics.
func NewObjectDamageInstance(damages []Damage, adjustments []DamageAdjustment) (ObjectDamageInstance, error) {
	if err := errors.Join(overCap("object damages", len(damages), maxDamageParts), overCap("object adjustments", len(adjustments), maxDamageParts)); err != nil {
		return ObjectDamageInstance{}, err
	}
	if len(damages) == 0 {
		return ObjectDamageInstance{}, invalidInput("object damage instance without damage")
	}
	for _, d := range damages {
		if d.amount < 0 || !d.typ.Valid() {
			return ObjectDamageInstance{}, invalidInput("object damage instance holds an invalid Damage")
		}
	}
	for _, a := range adjustments {
		if a == nil {
			return ObjectDamageInstance{}, invalidInput("object damage instance holds a nil adjustment")
		}
	}
	return ObjectDamageInstance{append([]Damage(nil), damages...), append([]DamageAdjustment(nil), adjustments...)}, nil
}

// Damages returns a copy of the damage of the instance.
func (i ObjectDamageInstance) Damages() []Damage { return append([]Damage(nil), i.damages...) }

// Adjustments returns a copy of the adjustments of the instance.
func (i ObjectDamageInstance) Adjustments() []DamageAdjustment {
	return append([]DamageAdjustment(nil), i.adjustments...)
}

// AutomaticBreakBasis is why an object can be broken without damage. The zero value
// is invalid.
type AutomaticBreakBasis int

// The two bases.
const (
	// AutomaticBreakFragileNonmagical: a fragile, nonmagical object, broken as an action (SRD 5.2.1, p. 12).
	AutomaticBreakFragileNonmagical AutomaticBreakBasis = iota + 1
	// AutomaticBreakGMAllowed: an exceedingly fragile object the GM lets a creature break (p. 177).
	AutomaticBreakGMAllowed
)

// Valid reports whether b is one of the two bases.
func (b AutomaticBreakBasis) Valid() bool {
	return b == AutomaticBreakFragileNonmagical || b == AutomaticBreakGMAllowed
}

// String returns "fragile_nonmagical", "gm_allowed" or "unknown".
func (b AutomaticBreakBasis) String() string {
	return enumName([]string{"fragile_nonmagical", "gm_allowed"}, int(b))
}

// ObjectBrokenAutomatically is an object broken without damage, so its Damage
// Threshold and affinities do not apply. Fragility, magic and whether the action
// was spent are the host's decisions. Construct it with a keyed literal.
type ObjectBrokenAutomatically struct{ Basis AutomaticBreakBasis }

func (ObjectDamageInstance) isObjectHarm()      {}
func (ObjectBrokenAutomatically) isObjectHarm() {}

// ObjectDamageRejection is why damage to an object was refused. The zero value is
// not a rejection.
type ObjectDamageRejection int

// ObjectDamageRejectionAlreadyDestroyed: the object is already at 0 Hit Points.
const ObjectDamageRejectionAlreadyDestroyed ObjectDamageRejection = 1

// String returns "already_destroyed" or "unknown".
func (r ObjectDamageRejection) String() string {
	return enumName([]string{"already_destroyed"}, int(r))
}

// objectDamage is what the object damage rule decided.
type objectDamage struct {
	status      ObjectStatus
	events      []EnvironmentEvent
	resolutions []DamageResolution
	threshold   ThresholdOutcome
}

// damageObject is the rule of ApplyDamageToObject: the threshold test first, on the
// incoming amounts, then each Damage through the damage pipeline separately, then
// the Hit Points lost, at most the current Hit Points. An invalid harm is an error.
func damageObject(s ObjectStatus, harm ObjectHarm) (objectDamage, ObjectDamageRejection, error) {
	if s.IsDestroyed() {
		return objectDamage{}, ObjectDamageRejectionAlreadyDestroyed, nil
	}
	switch h := harm.(type) {
	case ObjectBrokenAutomatically:
		if !h.Basis.Valid() {
			return objectDamage{}, 0, invalidInput("invalid automatic break basis")
		}
		return objectDamage{s.withCurrent(0), []EnvironmentEvent{ObjectDestroyed{}}, nil, ThresholdNotApplicable}, 0, nil
	case ObjectDamageInstance:
		if len(h.damages) == 0 {
			return objectDamage{}, 0, invalidInput("object damage instance without damage")
		}
		gate := DamageThresholdGate{}.Evaluate(s.threshold, h.damages)
		if gate == ThresholdNotMet {
			return objectDamage{s, []EnvironmentEvent{ObjectDamageDeflected{}}, nil, gate}, 0, nil
		}
		out := objectDamage{status: s, threshold: gate}
		total := 0
		for _, d := range h.damages {
			r := resolveDamage(d, h.adjustments, s.affinities, false)
			out.resolutions = append(out.resolutions, r)
			total = satAdd(total, r.Dealt())
		}
		lost := min(total, s.hitPoints.Current())
		if lost == 0 {
			return out, 0, nil
		}
		out.status = s.withCurrent(s.hitPoints.Current() - lost)
		out.events = []EnvironmentEvent{ObjectDamaged{amount: lost}}
		if out.status.IsDestroyed() {
			out.events = append(out.events, ObjectDestroyed{})
		}
		return out, 0, nil
	}
	return objectDamage{}, 0, invalidInput("object harm is not a known variant")
}

// withCurrent returns s with its current Hit Points set to n, at most the maximum.
func (s ObjectStatus) withCurrent(n int) ObjectStatus {
	s.hitPoints = HitPoints{n, s.hitPoints.Maximum(), 0}
	return s
}
