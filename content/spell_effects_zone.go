package content

import "slices"

// ZoneTrigger is when a lingering area of effect acts on a creature (SRD
// 5.2.1, "Spirit Guardians", pp. 164-165). The zero ZoneTrigger is invalid.
type ZoneTrigger int

// The zone triggers.
const (
	// CreatureEnters is a creature entering the area.
	CreatureEnters ZoneTrigger = iota + 1
	// CreatureFirstEntryOnTurn is a creature entering the area for the first
	// time on a turn.
	CreatureFirstEntryOnTurn
	// CreatureEndsTurnInside is a creature ending its turn in the area.
	CreatureEndsTurnInside
	// CreatureStartsTurnInside is a creature starting its turn in the area.
	CreatureStartsTurnInside
	// ZoneMovesIntoSpace is the area moving into a creature's space.
	ZoneMovesIntoSpace
	// TravelsWithin is a creature travelling a distance within the area; the
	// rule then carries that distance in feet.
	TravelsWithin
)

func zoneTriggerNames() []string {
	return []string{
		"creatureEnters", "creatureFirstEntryOnTurn", "creatureEndsTurnInside",
		"creatureStartsTurnInside", "zoneMovesIntoSpace", "travelsWithin",
	}
}

// Valid reports whether t is one of the six triggers.
func (t ZoneTrigger) Valid() bool { return enumValid(zoneTriggerNames(), int(t)) }

// String returns the name of t, or "unknown" for an undefined value.
func (t ZoneTrigger) String() string { return enumString(zoneTriggerNames(), int(t)) }

// ZoneTargetScope is which creatures a zone rule acts on. The zero
// ZoneTargetScope is invalid.
type ZoneTargetScope int

// The zone target scopes.
const (
	// AnyCreature is every creature.
	AnyCreature ZoneTargetScope = iota + 1
	// EnemiesOfCaster is the creatures the caster is hostile to.
	EnemiesOfCaster
	// CreaturesCasterCanSee is the creatures the caster can see.
	CreaturesCasterCanSee
)

func zoneTargetScopeNames() []string {
	return []string{"anyCreature", "enemiesOfCaster", "creaturesCasterCanSee"}
}

// Valid reports whether s is one of the three scopes.
func (s ZoneTargetScope) Valid() bool { return enumValid(zoneTargetScopeNames(), int(s)) }

// String returns the name of s, or "unknown" for an undefined value.
func (s ZoneTargetScope) String() string { return enumString(zoneTargetScopeNames(), int(s)) }

// ZoneActionKind is what the caster spends to move a zone. The zero
// ZoneActionKind is invalid.
type ZoneActionKind int

// The zone action kinds.
const (
	// BonusAction is a Bonus Action.
	BonusAction ZoneActionKind = iota + 1
	// MagicAction is the Magic action.
	MagicAction
	// WithCasterMovement is moving the zone as part of the caster's movement.
	WithCasterMovement
)

func zoneActionKindNames() []string {
	return []string{"bonusAction", "magicAction", "withCasterMovement"}
}

// Valid reports whether k is one of the three kinds.
func (k ZoneActionKind) Valid() bool { return enumValid(zoneActionKindNames(), int(k)) }

// String returns the name of k, or "unknown" for an undefined value.
func (k ZoneActionKind) String() string { return enumString(zoneActionKindNames(), int(k)) }

// DriftDirectionRule is how a drifting zone picks its direction. The zero
// DriftDirectionRule is invalid.
type DriftDirectionRule int

// The drift direction rules.
const (
	// AwayFromCaster drifts directly away from the caster.
	AwayFromCaster DriftDirectionRule = iota + 1
	// AwayFromCasterChosen drifts away from the caster in a direction the
	// caster chooses.
	AwayFromCasterChosen
)

func driftDirectionRuleNames() []string { return []string{"awayFromCaster", "awayFromCasterChosen"} }

// Valid reports whether d is one of the two rules.
func (d DriftDirectionRule) Valid() bool { return enumValid(driftDirectionRuleNames(), int(d)) }

// String returns the name of d, or "unknown" for an undefined value.
func (d DriftDirectionRule) String() string { return enumString(driftDirectionRuleNames(), int(d)) }

// DesignationPolicy is whether the caster designates creatures the zone spares
// or acts on. The zero DesignationPolicy is invalid; [NewZoneFacts] reads it
// as [NoDesignation].
type DesignationPolicy int

// The designation policies.
const (
	// NoDesignation is a zone with no designation.
	NoDesignation DesignationPolicy = iota + 1
	// DesignationAtCastingOnly is a designation the caster makes when casting
	// and only then.
	DesignationAtCastingOnly
)

func designationPolicyNames() []string { return []string{"none", "atCastingOnly"} }

// Valid reports whether p is one of the two policies.
func (p DesignationPolicy) Valid() bool { return enumValid(designationPolicyNames(), int(p)) }

// String returns the name of p, or "unknown" for an undefined value.
func (p DesignationPolicy) String() string { return enumString(designationPolicyNames(), int(p)) }

// ZoneRegionID names a part of a zone, such as its body. The zero ZoneRegionID
// is invalid; [NewZoneRuleFact] reads it as [DefaultZoneRegion].
type ZoneRegionID string

// DefaultZoneRegion is the region of a rule that names none.
const DefaultZoneRegion ZoneRegionID = "body"

// NewZoneRegionID returns the region named s. A name has no surrounding white
// space and holds only clean characters (schema FR-027).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName for a blank or unclean name. It never panics.
func NewZoneRegionID(s string) (ZoneRegionID, error) {
	if err := checkName("ZoneRegionID", s); err != nil {
		return "", err
	}
	return ZoneRegionID(s), nil
}

// String returns the name.
func (r ZoneRegionID) String() string { return string(r) }

// ZoneRuleFields are the arguments of [NewZoneRuleFact]. Build it with a keyed
// literal.
type ZoneRuleFields struct {
	Trigger ZoneTrigger
	// PartIndex names a part of the same model. The constructor does not
	// range-check it (FR-048).
	PartIndex int
	Scope     ZoneTargetScope
	// Region is the part of the zone the rule is about; empty reads as
	// [DefaultZoneRegion].
	Region ZoneRegionID
	// UnitFeet is set (at least 1) for [TravelsWithin] and 0 for any other
	// trigger.
	UnitFeet int
	// CasterChoice states that the caster chooses whether the rule applies.
	CasterChoice bool
}

// ZoneRuleFact is one rule of a lingering area: a trigger, the part of the
// model it applies and who it acts on. It holds a fact, no state. The zero
// value is invalid; build it with [NewZoneRuleFact]. Comparable and safe for
// concurrent use.
type ZoneRuleFact struct{ f ZoneRuleFields }

// NewZoneRuleFact returns the rule of f.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid trigger or scope, BlockInvalidName for an
// invalid region, BlockOutOfRange for a negative UnitFeet and BlockConstraint
// for UnitFeet missing on [TravelsWithin] or set on another trigger. It never
// panics.
func NewZoneRuleFact(f ZoneRuleFields) (ZoneRuleFact, error) {
	if !f.Trigger.Valid() || !f.Scope.Valid() {
		return ZoneRuleFact{}, blockError("ZoneRuleFact", BlockInvalidMember)
	}
	if f.Region == "" {
		f.Region = DefaultZoneRegion
	}
	if _, err := NewZoneRegionID(string(f.Region)); err != nil {
		return ZoneRuleFact{}, err
	}
	switch {
	case f.UnitFeet < 0:
		return ZoneRuleFact{}, blockError("ZoneRuleFact", BlockOutOfRange)
	case (f.Trigger == TravelsWithin) != (f.UnitFeet > 0):
		return ZoneRuleFact{}, blockError("ZoneRuleFact", BlockConstraint)
	}
	return ZoneRuleFact{f: f}, nil
}

// Trigger returns when the rule acts.
func (r ZoneRuleFact) Trigger() ZoneTrigger { return r.f.Trigger }

// PartIndex returns the index of the part of the model the rule applies.
func (r ZoneRuleFact) PartIndex() int { return r.f.PartIndex }

// Scope returns which creatures the rule acts on.
func (r ZoneRuleFact) Scope() ZoneTargetScope { return r.f.Scope }

// Region returns the part of the zone the rule is about.
func (r ZoneRuleFact) Region() ZoneRegionID { return r.f.Region }

// UnitFeet returns the distance of a [TravelsWithin] rule, and 0 for any other.
func (r ZoneRuleFact) UnitFeet() int { return r.f.UnitFeet }

// CasterChoice reports whether the caster chooses whether the rule applies.
func (r ZoneRuleFact) CasterChoice() bool { return r.f.CasterChoice }

// ZoneMobility is the closed set of ways a lingering area moves:
// [StationaryZone], [MovesWithOrigin], [MovedByCaster] and
// [DriftsAtTurnStart]. Every variant is an immutable, comparable value and
// safe for concurrent use. Consumers switch over it exhaustively.
type ZoneMobility interface{ isZoneMobility() }

// StationaryZone is a zone that stays where it was made. The zero value is
// valid.
type StationaryZone struct{}

func (StationaryZone) isZoneMobility() {}

// MovesWithOrigin is a zone that moves with its origin (SRD 5.2.1, "Spirit
// Guardians", pp. 164-165). The zero value is valid.
type MovesWithOrigin struct{}

func (MovesWithOrigin) isZoneMobility() {}

// MovedByCaster is a zone the caster moves up to MaxFeet with an action (SRD
// 5.2.1, "Flaming Sphere", p. 132). The zero value is invalid; build it with
// [NewMovedByCaster].
type MovedByCaster struct {
	maxFeet        int
	action         ZoneActionKind
	laterTurnsOnly bool
}

// NewMovedByCaster returns the mobility of a zone moved up to maxFeet by
// action; laterTurnsOnly states that it may be moved only on later turns.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for maxFeet below 1 and BlockInvalidMember for an invalid
// action. It never panics.
func NewMovedByCaster(maxFeet int, action ZoneActionKind, laterTurnsOnly bool) (MovedByCaster, error) {
	if maxFeet < 1 {
		return MovedByCaster{}, blockError("MovedByCaster", BlockOutOfRange)
	}
	if !action.Valid() {
		return MovedByCaster{}, blockError("MovedByCaster", BlockInvalidMember)
	}
	return MovedByCaster{maxFeet: maxFeet, action: action, laterTurnsOnly: laterTurnsOnly}, nil
}

// MaxFeet returns the farthest the caster may move the zone, in feet.
func (m MovedByCaster) MaxFeet() int { return m.maxFeet }

// Action returns what the caster spends.
func (m MovedByCaster) Action() ZoneActionKind { return m.action }

// LaterTurnsOnly reports whether the zone may be moved only on later turns.
func (m MovedByCaster) LaterTurnsOnly() bool { return m.laterTurnsOnly }

func (MovedByCaster) isZoneMobility() {}

// DriftsAtTurnStart is a zone that drifts Feet at the start of each turn (SRD
// 5.2.1, "Cloudkill", p. 116). The zero value is invalid; build it with
// [NewDriftsAtTurnStart].
type DriftsAtTurnStart struct {
	feet      int
	direction DriftDirectionRule
}

// NewDriftsAtTurnStart returns the mobility of a zone drifting feet by
// direction.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for feet below 1 and BlockInvalidMember for an invalid
// direction. It never panics.
func NewDriftsAtTurnStart(feet int, direction DriftDirectionRule) (DriftsAtTurnStart, error) {
	if feet < 1 {
		return DriftsAtTurnStart{}, blockError("DriftsAtTurnStart", BlockOutOfRange)
	}
	if !direction.Valid() {
		return DriftsAtTurnStart{}, blockError("DriftsAtTurnStart", BlockInvalidMember)
	}
	return DriftsAtTurnStart{feet: feet, direction: direction}, nil
}

// Feet returns how far the zone drifts, in feet.
func (d DriftsAtTurnStart) Feet() int { return d.feet }

// Direction returns how the direction is chosen.
func (d DriftsAtTurnStart) Direction() DriftDirectionRule { return d.direction }

func (DriftsAtTurnStart) isZoneMobility() {}

// checkZoneMobility reports why m is not a mobility its constructor would
// accept; a nil mobility is [StationaryZone].
func checkZoneMobility(m ZoneMobility) error {
	var err error
	switch v := m.(type) {
	case nil, StationaryZone, MovesWithOrigin:
	case MovedByCaster:
		_, err = NewMovedByCaster(v.maxFeet, v.action, v.laterTurnsOnly)
	case DriftsAtTurnStart:
		_, err = NewDriftsAtTurnStart(v.feet, v.direction)
	default:
		err = blockError("ZoneMobility", BlockInvalidMember)
	}
	return err
}

// ZoneMovementFacts is how a zone changes the movement of creatures in it (SRD
// 5.2.1, "Spike Growth", p. 164; "Difficult Terrain", p. 181). The zero value
// is valid and changes nothing. Comparable and safe for concurrent use.
type ZoneMovementFacts struct {
	difficultTerrain bool
	speedHalved      bool
	extraFeetPerFoot int
}

// NewZoneMovementFacts returns the movement facts: difficult terrain, halved
// speed, and extraFeetPerFoot extra feet of movement spent for each foot moved
// (0 means none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a negative extraFeetPerFoot. It never panics.
func NewZoneMovementFacts(difficultTerrain, speedHalved bool, extraFeetPerFoot int) (ZoneMovementFacts, error) {
	if extraFeetPerFoot < 0 {
		return ZoneMovementFacts{}, blockError("ZoneMovementFacts", BlockOutOfRange)
	}
	return ZoneMovementFacts{difficultTerrain: difficultTerrain, speedHalved: speedHalved, extraFeetPerFoot: extraFeetPerFoot}, nil
}

// DifficultTerrain reports whether the zone is Difficult Terrain.
func (m ZoneMovementFacts) DifficultTerrain() bool { return m.difficultTerrain }

// SpeedHalved reports whether the speed of creatures in the zone is halved.
func (m ZoneMovementFacts) SpeedHalved() bool { return m.speedHalved }

// ExtraFeetPerFoot returns the extra feet of movement spent for each foot
// moved.
func (m ZoneMovementFacts) ExtraFeetPerFoot() int { return m.extraFeetPerFoot }

// ZoneFactsFields are the arguments of [NewZoneFacts]. Build it with a keyed
// literal.
type ZoneFactsFields struct {
	// Rules is the non-empty set of rules.
	Rules []ZoneRuleFact
	// Mobility is how the zone moves; nil reads as [StationaryZone].
	Mobility ZoneMobility
	// OncePerTurn states that a rule acts on a creature once on a turn.
	OncePerTurn bool
	// Designation is the designation policy; the zero value reads as
	// [NoDesignation].
	Designation DesignationPolicy
	// DamageBudget is the most damage the zone can deal in total; 0 means
	// no budget.
	DamageBudget int
	// StopsMoverOnContact states that a creature moved into the zone stops.
	StopsMoverOnContact bool
	// Movement is how the zone changes movement.
	Movement ZoneMovementFacts
}

// ZoneFacts are the static facts of a lingering area of effect (spell-effect-data.md
// FR-048): its rules, how it moves, and its budget. They hold no state, ledger
// or event. The zero value is invalid; build it with [NewZoneFacts]. Not
// comparable with ==; use [ZoneFacts.Equal].
type ZoneFacts struct{ f ZoneFactsFields }

// NewZoneFacts returns the facts of f, copying the rules. A rule's PartIndex
// is not checked against any model here.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// for no rule, the reason of a rule's or the mobility's own check for an
// invalid one, BlockDuplicate for a repeated identical rule,
// BlockInvalidMember for an invalid designation, BlockOutOfRange for a
// negative DamageBudget or an invalid movement. It never panics.
func NewZoneFacts(f ZoneFactsFields) (ZoneFacts, error) {
	if len(f.Rules) == 0 {
		return ZoneFacts{}, blockError("ZoneFacts", BlockEmpty)
	}
	for i, r := range f.Rules {
		if _, err := NewZoneRuleFact(r.f); err != nil {
			return ZoneFacts{}, err
		}
		if slices.Contains(f.Rules[:i], r) {
			return ZoneFacts{}, blockError("ZoneFacts", BlockDuplicate)
		}
	}
	if err := checkZoneMobility(f.Mobility); err != nil {
		return ZoneFacts{}, err
	}
	if f.Mobility == nil {
		f.Mobility = StationaryZone{}
	}
	if f.Designation == 0 {
		f.Designation = NoDesignation
	}
	switch {
	case !f.Designation.Valid():
		return ZoneFacts{}, blockError("ZoneFacts", BlockInvalidMember)
	case f.DamageBudget < 0:
		return ZoneFacts{}, blockError("ZoneFacts", BlockOutOfRange)
	}
	if _, err := NewZoneMovementFacts(f.Movement.difficultTerrain, f.Movement.speedHalved, f.Movement.extraFeetPerFoot); err != nil {
		return ZoneFacts{}, err
	}
	f.Rules = slices.Clone(f.Rules)
	return ZoneFacts{f: f}, nil
}

// Rules returns a copy of the rules.
func (z ZoneFacts) Rules() []ZoneRuleFact { return slices.Clone(z.f.Rules) }

// Mobility returns how the zone moves.
func (z ZoneFacts) Mobility() ZoneMobility { return z.f.Mobility }

// OncePerTurn reports whether a rule acts on a creature once on a turn.
func (z ZoneFacts) OncePerTurn() bool { return z.f.OncePerTurn }

// Designation returns the designation policy.
func (z ZoneFacts) Designation() DesignationPolicy { return z.f.Designation }

// DamageBudget returns the most damage the zone can deal in total, or 0 for no
// budget.
func (z ZoneFacts) DamageBudget() int { return z.f.DamageBudget }

// StopsMoverOnContact reports whether a creature moved into the zone stops.
func (z ZoneFacts) StopsMoverOnContact() bool { return z.f.StopsMoverOnContact }

// Movement returns how the zone changes movement.
func (z ZoneFacts) Movement() ZoneMovementFacts { return z.f.Movement }

// Equal reports whether z and o hold the same facts, the rules compared as a
// set.
func (z ZoneFacts) Equal(o ZoneFacts) bool {
	a, b := z.f, o.f
	if len(a.Rules) != len(b.Rules) || a.Mobility != b.Mobility || a.OncePerTurn != b.OncePerTurn ||
		a.Designation != b.Designation || a.DamageBudget != b.DamageBudget ||
		a.StopsMoverOnContact != b.StopsMoverOnContact || a.Movement != b.Movement {
		return false
	}
	for _, r := range a.Rules {
		if !slices.Contains(b.Rules, r) {
			return false
		}
	}
	return true
}
