package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// HoldID is the id of a hold, supplied by the host. It is a comparable value, not trimmed or
// case-folded. The zero HoldID (the empty string) is invalid. Safe for concurrent use.
type HoldID string

// NewHoldID returns id unchanged. It returns a *GearError (matching srderr.ErrInvalidInput)
// with GearProblemBlankID when strings.TrimSpace(id) is empty. It never panics.
func NewHoldID(id string) (HoldID, error) {
	if strings.TrimSpace(id) == "" {
		return "", gearInvalid(GearProblemBlankID)
	}
	return HoldID(id), nil
}

// MustHoldID is NewHoldID for literals known to be valid. It panics with the error NewHoldID
// returns, and only then.
func MustHoldID(id string) HoldID {
	v, err := NewHoldID(id)
	if err != nil {
		panic(err)
	}
	return v
}

// DCOverride is an escape Difficulty Class the host supplies in place of a hold's profile
// (a Falling Net hold uses the trap's DC). Any integer is held as given. The zero value
// supplies none.
type DCOverride struct {
	dc  d20.DifficultyClass
	set bool
}

// NoDCOverride returns the override that supplies none; it equals the zero DCOverride.
func NoDCOverride() DCOverride { return DCOverride{} }

// DCOverrideOf returns the override that supplies dc.
func DCOverrideOf(dc d20.DifficultyClass) DCOverride { return DCOverride{dc, true} }

// or returns the override when one is supplied, otherwise fallback.
func (o DCOverride) or(fallback d20.DifficultyClass) d20.DifficultyClass {
	if o.set {
		return o.dc
	}
	return fallback
}

// GearHoldInput holds the values NewGearHold validates. Build it with a keyed literal.
type GearHoldInput struct {
	// ID is the id of the new hold.
	ID HoldID
	// Kind is the kind of the hold.
	Kind HoldKind
	// Target is the held creature.
	Target CombatantID
	// EscapeDC is the escape Difficulty Class; the profile's applies when none is supplied.
	EscapeDC DCOverride
	// Restraint says whether the hold gives the Restrained condition: always for a Net,
	// never for a Hunting Trap.
	Restraint HoldRestraint
	// GearID is the Hunting Trap that holds the creature; required for a Hunting Trap, absent otherwise.
	GearID DeployedGearID
	// MovementLimitFeet is the length of a Hunting Trap's chain; 0 for any other kind.
	MovementLimitFeet int
	// Status is a Net's state as an object; the zero ObjectStatus stands for NetProfile.Status.
	Status ObjectStatus
}

// GearHold is one hold on one creature: id, kind, target, escape Difficulty Class, restraint,
// and for a Hunting Trap its gear id and chain limit, for a Net its state as an object (SRD
// 5.2.1, pp. 96-99). Build it with NewGearHold; the zero GearHold is invalid. It is
// comparable on all its parts and immutable. Safe for concurrent use.
type GearHold struct {
	id            HoldID
	kind          HoldKind
	target        CombatantID
	escapeDC      d20.DifficultyClass
	restraint     HoldRestraint
	gearID        DeployedGearID
	movementLimit int
	status        ObjectStatus
}

// NewGearHold returns the hold of in. The escape Difficulty Class defaults to the profile's
// and is held as given when supplied; a Net without a status gets NetProfile.Status.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) for a blank id or target
// (GearProblemBlankID), a kind or restraint that is not valid (GearProblemInvalidValue), a Net
// that does not restrain, a Hunting Trap that does, a status on a kind other than a Net or a
// chain limit on a kind other than a Hunting Trap (GearProblemHoldKindMismatch), a negative
// chain limit (GearProblemNegativeMovementLimit), or a Hunting Trap without a gear id or any
// other kind with one (GearProblemGearIDMismatch). It never panics.
func NewGearHold(in GearHoldInput) (GearHold, error) {
	trap := in.Kind == HoldKindHuntingTrap
	switch {
	case strings.TrimSpace(string(in.ID)) == "" || in.Target.IsZero():
		return GearHold{}, gearInvalid(GearProblemBlankID)
	case !in.Kind.Valid() || !in.Restraint.Valid():
		return GearHold{}, gearInvalid(GearProblemInvalidValue)
	case in.Kind == HoldKindNet && in.Restraint != HoldRestraintRestrains,
		trap && in.Restraint != HoldRestraintDoesNotRestrain,
		!in.Status.IsZero() && in.Kind != HoldKindNet,
		!trap && in.MovementLimitFeet > 0:
		return GearHold{}, gearInvalid(GearProblemHoldKindMismatch)
	case in.MovementLimitFeet < 0:
		return GearHold{}, gearInvalid(GearProblemNegativeMovementLimit)
	case trap != (strings.TrimSpace(string(in.GearID)) != ""):
		return GearHold{}, gearInvalid(GearProblemGearIDMismatch)
	}
	status := in.Status
	if in.Kind == HoldKindNet && status.IsZero() {
		status = NetProfile{}.Status()
	}
	dc := in.EscapeDC.or(HoldProfileFor(in.Kind).escape.dc)
	return GearHold{in.ID, in.Kind, in.Target, dc, in.Restraint, in.GearID, in.MovementLimitFeet, status}, nil
}

// IsZero reports whether h is the invalid zero GearHold.
func (h GearHold) IsZero() bool { return h.id == "" }

// ID returns the id of the hold.
func (h GearHold) ID() HoldID { return h.id }

// Kind returns the kind of the hold.
func (h GearHold) Kind() HoldKind { return h.kind }

// IsOn reports whether the hold is on id.
func (h GearHold) IsOn(id CombatantID) bool { return !h.IsZero() && h.target == id }

// EscapeDifficultyClass returns the Difficulty Class to escape the hold, as held.
func (h GearHold) EscapeDifficultyClass() d20.DifficultyClass { return h.escapeDC }

// Restraint returns whether the hold gives the Restrained condition.
func (h GearHold) Restraint() HoldRestraint { return h.restraint }

// GearID returns the Hunting Trap that holds the creature, the empty id for any other kind.
func (h GearHold) GearID() DeployedGearID { return h.gearID }

// MovementLimitFeet returns the length of a Hunting Trap's chain, 0 for any other kind.
func (h GearHold) MovementLimitFeet() int { return h.movementLimit }

// Status returns a Net's state as an object, the zero ObjectStatus for any other kind.
func (h GearHold) Status() ObjectStatus { return h.status }

func (h GearHold) restrains() bool { return h.restraint == HoldRestraintRestrains }

// clashes reports whether h and o cannot both be in one set: one id, or one target and
// kind, a Hunting Trap's key including the trap (a creature may be caught by two traps).
func (h GearHold) clashes(o GearHold) (GearRejection, bool) {
	switch {
	case h.target == o.target && h.kind == o.kind && (h.kind != HoldKindHuntingTrap || h.gearID == o.gearID):
		return GearRejectionTargetAlreadyHeld, true
	case h.id == o.id:
		return GearRejectionIDInUse, true
	}
	return 0, false
}

// GearHoldsTransition is the closed set of outcomes of a change to GearHolds. Variants:
// GearHoldsApplied, GearHoldsRejected.
type GearHoldsTransition interface{ isGearHoldsTransition() }

// GearHoldsApplied: the change took effect. The host applies or removes Restrained when
// RestraintBegins or RestraintEnds says so.
type GearHoldsApplied struct {
	holds           GearHolds
	events          []GearEvent
	restraintBegins bool
	restraintEnds   bool
}

// Holds returns the new set.
func (t GearHoldsApplied) Holds() GearHolds { return t.holds }

// Events returns a copy of the events.
func (t GearHoldsApplied) Events() []GearEvent { return slices.Clone(t.events) }

// RestraintBegins reports whether the held creature is now restrained by a hold and was by
// no other: the host applies ConditionKind Restrained.
func (t GearHoldsApplied) RestraintBegins() bool { return t.restraintBegins }

// RestraintEnds reports whether no hold in the set restrains the held creature any more and
// one did: the host removes Restrained, unless something outside the set also restrains it.
func (t GearHoldsApplied) RestraintEnds() bool { return t.restraintEnds }

// GearHoldsRejected: the rules refuse the change and nothing happened.
type GearHoldsRejected struct{ reason GearRejection }

// Reason returns why.
func (t GearHoldsRejected) Reason() GearRejection { return t.reason }

func (GearHoldsApplied) isGearHoldsTransition()  {}
func (GearHoldsRejected) isGearHoldsTransition() {}

// GearHolds is the set of holds in a fight, in the order they began: at most one per
// target and kind (a Hunting Trap's key also names the trap), with distinct ids. It is
// immutable: every change returns a new set. The zero value is the empty set. Compare with
// Equal. Safe for concurrent use.
type GearHolds struct{ items []GearHold }

// NewGearHolds returns the set of holds in the given order, copying the slice. It returns an
// error matching srderr.ErrInvalidInput for a zero hold, two holds with one id, or two with one
// target and kind (the same Hunting Trap for a trap). It returns a
// LimitExceededError for more than 256 holds (plan P10), before it reads one. It never panics.
func NewGearHolds(holds []GearHold) (GearHolds, error) {
	if err := overCap("gear holds", len(holds), maxInputList); err != nil {
		return GearHolds{}, err
	}
	for i, h := range holds {
		if h.IsZero() {
			return GearHolds{}, invalidInput("gear holds need non-zero holds")
		}
		for _, o := range holds[:i] {
			if _, clash := h.clashes(o); clash {
				return GearHolds{}, invalidInput("gear holds need distinct ids and one hold per target and kind")
			}
		}
	}
	return GearHolds{slices.Clone(holds)}, nil
}

// Holds returns a copy of the holds in the order they began.
func (s GearHolds) Holds() []GearHold { return slices.Clone(s.items) }

// Len returns the number of holds.
func (s GearHolds) Len() int { return len(s.items) }

// HoldsOn returns a copy of the holds on target, in the order they began.
func (s GearHolds) HoldsOn(target CombatantID) []GearHold {
	return slices.DeleteFunc(slices.Clone(s.items), func(h GearHold) bool { return h.target != target })
}

// HoldOf returns the hold with the id; ok is false when there is none (the zero GearHold is
// invalid, so absence is never a nil).
func (s GearHolds) HoldOf(id HoldID) (hold GearHold, ok bool) {
	i := s.index(id)
	if i < 0 {
		return GearHold{}, false
	}
	return s.items[i], true
}

// Equal reports whether s and o hold the same holds, whatever the order they began in.
func (s GearHolds) Equal(o GearHolds) bool {
	return len(s.items) == len(o.items) && !slices.ContainsFunc(s.items, func(h GearHold) bool { return !slices.Contains(o.items, h) })
}

func (s GearHolds) index(id HoldID) int {
	return slices.IndexFunc(s.items, func(h GearHold) bool { return h.id == id })
}

// holdsTrap reports whether some hold is a Hunting Trap's grip from the trap with the id.
func (s GearHolds) holdsTrap(id DeployedGearID) bool {
	return slices.ContainsFunc(s.items, func(h GearHold) bool { return h.kind == HoldKindHuntingTrap && h.gearID == id })
}

// RestrainsTarget reports whether any hold on target gives the Restrained condition. A hold
// that does not restrain (Rope or Chain on arms, loose Manacles, a Hunting Trap) does not.
func (s GearHolds) RestrainsTarget(target CombatantID) bool {
	return slices.ContainsFunc(s.items, func(h GearHold) bool { return h.target == target && h.restrains() })
}

// AttackRollCircumstances returns Disadvantage while a Manacles hold is on target, restraining
// or not, and nothing otherwise (SRD 5.2.1, "Manacles", p. 98: "While bound, a creature has
// Disadvantage on attack rolls"). It returns a new slice. The caller merges it with the
// condition circumstances through modifier.Resolve.
func (s GearHolds) AttackRollCircumstances(target CombatantID) []modifier.RollCircumstance {
	if slices.ContainsFunc(s.items, func(h GearHold) bool { return h.target == target && h.kind == HoldKindManacles }) {
		return []modifier.RollCircumstance{modifier.Disadvantage{}}
	}
	return []modifier.RollCircumstance{}
}

// othersRestrain reports whether a hold other than the one at skip restrains target.
func (s GearHolds) othersRestrain(target CombatantID, skip int) bool {
	for i, h := range s.items {
		if i != skip && h.target == target && h.restrains() {
			return true
		}
	}
	return false
}

func (s GearHolds) replace(i int, h GearHold) GearHolds {
	items := slices.Clone(s.items)
	items[i] = h
	return GearHolds{items}
}

func (s GearHolds) remove(i int) GearHolds {
	return GearHolds{slices.Delete(slices.Clone(s.items), i, i+1)}
}

// Begin adds hold. It returns a GearHoldsApplied with one GearHoldStarted, and RestraintBegins
// when the hold restrains and no other hold in the set already restrains its target (SRD
// 5.2.1, "Net", "Chain", pp. 96, 98), or a GearHoldsRejected with idInUse or targetAlreadyHeld
// (one hold per target and kind; a Hunting Trap's key also names the trap).
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for the
// zero hold, and GearProblemLimitExceeded (plan P10) for a set that already holds 256 holds. It never panics.
func (s GearHolds) Begin(hold GearHold) (GearHoldsTransition, error) {
	if hold.IsZero() {
		return nil, gearInvalid(GearProblemInvalidValue)
	}
	if len(s.items) >= maxInputList {
		return nil, gearInvalid(GearProblemLimitExceeded)
	}
	return s.begin(hold), nil
}

// begin is Begin for a hold known to be valid.
func (s GearHolds) begin(hold GearHold) GearHoldsTransition {
	for _, o := range s.items {
		if r, clash := hold.clashes(o); clash {
			return GearHoldsRejected{r}
		}
	}
	return GearHoldsApplied{
		holds:           GearHolds{append(slices.Clone(s.items), hold)},
		events:          []GearEvent{GearHoldStarted{gearEventBase{hold.target, ""}, hold.id, hold.kind}},
		restraintBegins: hold.restrains() && !s.RestrainsTarget(hold.target),
	}
}

// End removes the hold with the id. It returns a GearHoldsApplied with one GearHoldEnded and
// RestraintEnds when the hold restrained and no other hold in the set still restrains the
// target (another source of Restrained outside the set is the host's concern), or a
// GearHoldsRejected with unknownHold.
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for a
// cause that is not valid. It never panics.
func (s GearHolds) End(id HoldID, cause HoldEndCause) (GearHoldsTransition, error) {
	if !cause.Valid() {
		return nil, gearInvalid(GearProblemInvalidValue)
	}
	return s.end(id, cause), nil
}

// end is End for a cause known to be valid.
func (s GearHolds) end(id HoldID, cause HoldEndCause) GearHoldsTransition {
	i := s.index(id)
	if i < 0 {
		return GearHoldsRejected{GearRejectionUnknownHold}
	}
	h := s.items[i]
	return GearHoldsApplied{
		holds:         s.remove(i),
		events:        []GearEvent{GearHoldEnded{gearEventBase{h.target, ""}, h.id, h.kind, cause}},
		restraintEnds: h.restrains() && !s.othersRestrain(h.target, i),
	}
}

// Reanchor changes whether a Manacles hold restrains: attached to a chain or hook fixed in
// place, it does (SRD 5.2.1, "Manacles", p. 98). It returns a GearHoldsApplied with one
// GearHoldRestraintChanged and RestraintBegins or RestraintEnds as Begin and End define, no
// event and no change when the restraint is the one it has, or a GearHoldsRejected with
// unknownHold or methodNotAvailable (any hold other than Manacles).
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for a
// restraint that is not valid. It never panics.
func (s GearHolds) Reanchor(id HoldID, restraint HoldRestraint) (GearHoldsTransition, error) {
	if !restraint.Valid() {
		return nil, gearInvalid(GearProblemInvalidValue)
	}
	i := s.index(id)
	switch {
	case i < 0:
		return GearHoldsRejected{GearRejectionUnknownHold}, nil
	case s.items[i].kind != HoldKindManacles:
		return GearHoldsRejected{GearRejectionMethodNotAvailable}, nil
	case s.items[i].restraint == restraint:
		return GearHoldsApplied{holds: s}, nil
	}
	h := s.items[i]
	h.restraint = restraint
	others := s.othersRestrain(h.target, i)
	return GearHoldsApplied{
		holds:           s.replace(i, h),
		events:          []GearEvent{GearHoldRestraintChanged{gearEventBase{h.target, ""}, h.id, restraint}},
		restraintBegins: h.restrains() && !others,
		restraintEnds:   !h.restrains() && !others,
	}, nil
}

// RecordNetDamage stores the state of a Net after damage (the status from ApplyDamageToObject).
// A destroyed status ends the hold with HoldEndCauseDestroyed, freeing the target (SRD 5.2.1,
// "Net", p. 98); any other is stored and nothing else happens. It returns a GearHoldsApplied
// or a GearHoldsRejected with unknownHold (also a Net destroyed already) or notDestructible
// (a hold other than a Net).
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for the
// zero status. It never panics.
func (s GearHolds) RecordNetDamage(id HoldID, status ObjectStatus) (GearHoldsTransition, error) {
	if status.IsZero() {
		return nil, gearInvalid(GearProblemInvalidValue)
	}
	i := s.index(id)
	switch {
	case i < 0:
		return GearHoldsRejected{GearRejectionUnknownHold}, nil
	case s.items[i].kind != HoldKindNet:
		return GearHoldsRejected{GearRejectionNotDestructible}, nil
	case status.IsDestroyed():
		return s.end(id, HoldEndCauseDestroyed), nil
	}
	h := s.items[i]
	h.status = status
	return GearHoldsApplied{holds: s.replace(i, h)}, nil
}
