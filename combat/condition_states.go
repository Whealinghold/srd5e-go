package combat

// The named states of the condition rules. Each is a typed enum with a String
// method; the Go constant is the type name followed by the value, and the zero
// value of each is invalid. All are comparable and safe for concurrent use.

// enumName returns names[v-1] for v in 1..len(names), else "unknown".
func enumName(names []string, v int) string {
	if v < 1 || v > len(names) {
		return "unknown"
	}
	return names[v-1]
}

// Sense is a sense a check can require.
type Sense int

// The senses an ability check can require (SRD 5.2.1, "Blinded", "Deafened").
const (
	SenseSight Sense = iota + 1
	SenseHearing
)

// String returns "sight", "hearing" or "unknown".
func (s Sense) String() string { return enumName([]string{"sight", "hearing"}, int(s)) }

// CheckRequirement says what an ability check needs.
type CheckRequirement int

// The requirements of an ability check.
const (
	CheckRequirementSight CheckRequirement = iota + 1
	CheckRequirementHearing
	CheckRequirementNone
)

// String returns "sight", "hearing", "none" or "unknown".
func (c CheckRequirement) String() string {
	return enumName([]string{"sight", "hearing", "none"}, int(c))
}

// SightRelation says whether one creature can somehow see another.
type SightRelation int

// The sight relations. Only SightRelationCanSee counts as seeing.
const (
	SightRelationCanSee SightRelation = iota + 1
	SightRelationCannotSee
)

// String returns "can_see", "cannot_see" or "unknown".
func (s SightRelation) String() string { return enumName([]string{"can_see", "cannot_see"}, int(s)) }

// LineOfSight says whether a source of fear is within line of sight.
type LineOfSight int

// The lines of sight. Only LineOfSightClear counts as within line of sight.
const (
	LineOfSightClear LineOfSight = iota + 1
	LineOfSightBlocked
)

// String returns "clear", "blocked" or "unknown".
func (l LineOfSight) String() string { return enumName([]string{"clear", "blocked"}, int(l)) }

// EffectDamage says whether an ability or magical effect deals damage.
type EffectDamage int

// The two kinds of effect.
const (
	EffectDamageDamaging EffectDamage = iota + 1
	EffectDamageNotDamaging
)

// String returns "damaging", "not_damaging" or "unknown".
func (d EffectDamage) String() string { return enumName([]string{"damaging", "not_damaging"}, int(d)) }

// ConditionPermission is whether a condition lets a creature do something.
type ConditionPermission int

// The permissions.
const (
	ConditionPermissionAllowed ConditionPermission = iota + 1
	ConditionPermissionForbidden
)

// String returns "allowed", "forbidden" or "unknown".
func (p ConditionPermission) String() string {
	return enumName([]string{"allowed", "forbidden"}, int(p))
}

// SavingThrowEffect is what the conditions do to a saving throw with one ability.
type SavingThrowEffect int

// The effects on a saving throw. Automatic failure takes precedence over
// disadvantage.
const (
	SavingThrowEffectAutomaticFailure SavingThrowEffect = iota + 1
	SavingThrowEffectDisadvantage
	SavingThrowEffectUnaffected
)

// String returns "automatic_failure", "disadvantage", "unaffected" or "unknown".
func (e SavingThrowEffect) String() string {
	return enumName([]string{"automatic_failure", "disadvantage", "unaffected"}, int(e))
}

// AbilityCheckEffect is what the conditions do to an ability check that needs
// a sense.
type AbilityCheckEffect int

// The effects on a sense-dependent ability check.
const (
	AbilityCheckEffectAutomaticFailure AbilityCheckEffect = iota + 1
	AbilityCheckEffectUnaffected
)

// String returns "automatic_failure", "unaffected" or "unknown".
func (e AbilityCheckEffect) String() string {
	return enumName([]string{"automatic_failure", "unaffected"}, int(e))
}

// AutomaticCritical says whether a hit is a Critical Hit because of the
// conditions of its target.
type AutomaticCritical int

// The two values.
const (
	AutomaticCriticalAutomatic AutomaticCritical = iota + 1
	AutomaticCriticalNotAutomatic
)

// String returns "automatic", "not_automatic" or "unknown".
func (c AutomaticCritical) String() string {
	return enumName([]string{"automatic", "not_automatic"}, int(c))
}

// SpeedIncrease says whether a Speed of 0 can be raised.
type SpeedIncrease int

// The two values.
const (
	SpeedIncreaseAllowed SpeedIncrease = iota + 1
	SpeedIncreaseForbidden
)

// String returns "allowed", "forbidden" or "unknown".
func (s SpeedIncrease) String() string { return enumName([]string{"allowed", "forbidden"}, int(s)) }

// ConditionConcentration is the Concentration fact the conditions report.
type ConditionConcentration int

// The two values. The Concentration state and rule are not this package's.
const (
	ConditionConcentrationIntact ConditionConcentration = iota + 1
	ConditionConcentrationBroken
)

// String returns "intact", "broken" or "unknown".
func (c ConditionConcentration) String() string {
	return enumName([]string{"intact", "broken"}, int(c))
}

// Speech says whether a creature can speak.
type Speech int

// The two values.
const (
	SpeechAble Speech = iota + 1
	SpeechUnable
)

// String returns "able", "unable" or "unknown".
func (s Speech) String() string { return enumName([]string{"able", "unable"}, int(s)) }

// ConditionActionAvailability says whether a creature can take actions, Bonus
// Actions and Reactions.
type ConditionActionAvailability int

// The two values.
const (
	ConditionActionAvailabilityAvailable ConditionActionAvailability = iota + 1
	ConditionActionAvailabilityNone
)

// String returns "available", "none" or "unknown".
func (a ConditionActionAvailability) String() string {
	return enumName([]string{"available", "none"}, int(a))
}

// HeldItems says whether a creature drops what it holds.
type HeldItems int

// The two values.
const (
	HeldItemsKept HeldItems = iota + 1
	HeldItemsDropped
)

// String returns "kept", "dropped" or "unknown".
func (h HeldItems) String() string { return enumName([]string{"kept", "dropped"}, int(h)) }

// Awareness says whether a creature is aware of its surroundings.
type Awareness int

// The two values.
const (
	AwarenessAware Awareness = iota + 1
	AwarenessUnaware
)

// String returns "aware", "unaware" or "unknown".
func (a Awareness) String() string { return enumName([]string{"aware", "unaware"}, int(a)) }

// AllDamageResistance says whether a creature has Resistance to all damage.
type AllDamageResistance int

// The two values. The flag carries no damage type: the consumer passes it to
// the hit point rules, which interpret it.
const (
	AllDamageResistanceNone AllDamageResistance = iota + 1
	AllDamageResistanceAll
)

// String returns "none", "all" or "unknown".
func (r AllDamageResistance) String() string { return enumName([]string{"none", "all"}, int(r)) }

// EffectCondition says when an effect of a condition applies.
type EffectCondition int

// The circumstances that gate an effect.
const (
	// EffectConditionAlways: the effect always applies.
	EffectConditionAlways EffectCondition = iota + 1
	// EffectConditionAttackerWithin5Feet: the attacker is 0 to 5 feet away.
	EffectConditionAttackerWithin5Feet
	// EffectConditionAttackerBeyond5Feet: the attacker is more than 5 feet away.
	EffectConditionAttackerBeyond5Feet
	// EffectConditionSourceInLineOfSight: the source of fear is within line of sight.
	EffectConditionSourceInLineOfSight
	// EffectConditionTargetIsNotGrappler: the target is not the creature grappling this one.
	EffectConditionTargetIsNotGrappler
	// EffectConditionUnlessSeen: the other creature cannot somehow see this one.
	EffectConditionUnlessSeen
)

// String returns the snake_case name of c, or "unknown".
func (c EffectCondition) String() string {
	return enumName([]string{"always", "attacker_within_5_feet", "attacker_beyond_5_feet",
		"source_in_line_of_sight", "target_is_not_grappler", "unless_seen"}, int(c))
}
