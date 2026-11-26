package combat

// grappleConfig is what GrappleOption sets.
type grappleConfig struct{ dead bool }

// GrappleOption configures a GrappleObservation or GrappleRules.Hold.
type GrappleOption func(*grappleConfig)

// WithGrapplerDead states whether the grappler is dead. A dead creature can take no
// action, so it holds nobody (SRD 5.2.1, "Dead", p. 180; an interpretation):
// GrappleRules.Hold then returns GrappleHoldGrapplerIncapacitated before the range
// test. The default is false.
func WithGrapplerDead(dead bool) GrappleOption {
	return func(c *grappleConfig) { c.dead = dead }
}

func newGrappleConfig(opts []GrappleOption) grappleConfig {
	var c grappleConfig
	for _, o := range opts {
		o(&c)
	}
	return c
}

// GrappleObservation is what the host reports about one grapple for
// Grapples.Review: the grappler's conditions, the distance in whole feet between
// target and grappler (the engine models no map) and whether the grappler is
// dead. Build it with NewGrappleObservation; the zero value names no grapple. It
// is comparable with ==. Safe for concurrent use.
type GrappleObservation struct {
	grappler, target CombatantID
	conditions       ConditionSet
	distanceFeet     int
	dead             bool
}

// NewGrappleObservation returns the observation of the grapple in which grappler
// holds target. Option: WithGrapplerDead. It returns a *GrappleProblemError
// (matching srderr.ErrInvalidInput) carrying, in this order,
// GrappleProblemMissingValue for a blank id, GrappleProblemSameGrapplerAndTarget
// for equal ids and GrappleProblemNegativeDistance for a negative distance. It
// never panics.
func NewGrappleObservation(grappler, target CombatantID, grapplerConditions ConditionSet, distanceFeet int, opts ...GrappleOption) (GrappleObservation, error) {
	switch {
	case grappler.IsZero() || target.IsZero():
		return GrappleObservation{}, problemError(GrappleProblemMissingValue)
	case grappler == target:
		return GrappleObservation{}, problemError(GrappleProblemSameGrapplerAndTarget)
	case distanceFeet < 0:
		return GrappleObservation{}, problemError(GrappleProblemNegativeDistance)
	}
	return GrappleObservation{grappler, target, grapplerConditions, distanceFeet, newGrappleConfig(opts).dead}, nil
}

// MustGrappleObservation is NewGrappleObservation for literals known to be valid.
// It panics with the error NewGrappleObservation returns, and only then.
func MustGrappleObservation(grappler, target CombatantID, grapplerConditions ConditionSet, distanceFeet int, opts ...GrappleOption) GrappleObservation {
	o, err := NewGrappleObservation(grappler, target, grapplerConditions, distanceFeet, opts...)
	if err != nil {
		panic(err)
	}
	return o
}

// Conditions returns the grappler's conditions.
func (o GrappleObservation) Conditions() ConditionSet { return o.conditions }

// DistanceFeet returns the reported distance in feet.
func (o GrappleObservation) DistanceFeet() int { return o.distanceFeet }

// GrapplerDead reports whether the grappler is dead.
func (o GrappleObservation) GrapplerDead() bool { return o.dead }
