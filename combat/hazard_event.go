package combat

// EnvironmentEvent is the closed family of events the hazard and object rules
// produce. Each is an immutable value comparable with ==, with a constant
// snake_case Kind. The seven creature events are about exactly one creature, the
// target of the producing call (their Combatants is [target]); the four object
// events are about no creature (their Combatants is empty). Variants:
// BurningEnded, DehydrationRecovered, MalnutritionRecovered, BreathHeldStarted,
// RanOutOfBreath, BeganChoking, BreathRestored, ObjectDamaged,
// ObjectDamageDeflected, ObjectDestroyed, ObjectCollapsed. The creature events
// embed creatureEnvironmentEventBase and the object events objectEnvironmentEventBase.
type EnvironmentEvent interface {
	Event
	environmentEvent()
}

// creatureEnvironmentEventBase carries the creature a hazard event is about and
// the markers that put the event in the Event and EnvironmentEvent families.
type creatureEnvironmentEventBase struct{ target CombatantID }

func (creatureEnvironmentEventBase) event()            {}
func (creatureEnvironmentEventBase) environmentEvent() {}

// Combatants returns a new slice holding the target.
func (b creatureEnvironmentEventBase) Combatants() []CombatantID { return []CombatantID{b.target} }

// objectEnvironmentEventBase marks an object event, which is about no creature.
type objectEnvironmentEventBase struct{}

func (objectEnvironmentEventBase) event()            {}
func (objectEnvironmentEventBase) environmentEvent() {}

// Combatants returns a new, empty slice.
func (objectEnvironmentEventBase) Combatants() []CombatantID { return []CombatantID{} }

func newCreatureBase(target CombatantID) (creatureEnvironmentEventBase, error) {
	if target.IsZero() {
		return creatureEnvironmentEventBase{}, invalidInput("event without a target")
	}
	return creatureEnvironmentEventBase{target}, nil
}

// BurningEnded: the creature's fire went out, by the method. Its Kind is
// burning_ended.
type BurningEnded struct {
	creatureEnvironmentEventBase
	method ExtinguishMethod
}

// NewBurningEnded returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or an invalid method.
func NewBurningEnded(target CombatantID, method ExtinguishMethod) (BurningEnded, error) {
	b, err := newCreatureBase(target)
	if err != nil || !method.Valid() {
		return BurningEnded{}, invalidInput("burning ended needs a target and a valid method")
	}
	return BurningEnded{b, method}, nil
}

// Kind returns "burning_ended".
func (BurningEnded) Kind() string { return "burning_ended" }

// Method returns how the fire went out.
func (e BurningEnded) Method() ExtinguishMethod { return e.method }

// DehydrationRecovered: the creature drank its full daily water, so the Exhaustion
// levels Dehydration holds can be removed. Its Kind is dehydration_recovered.
type DehydrationRecovered struct {
	creatureEnvironmentEventBase
	levels int
}

// NewDehydrationRecovered returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or levels below 1.
func NewDehydrationRecovered(target CombatantID, levels int) (DehydrationRecovered, error) {
	b, err := newCreatureBase(target)
	if err != nil || levels < 1 {
		return DehydrationRecovered{}, invalidInput("dehydration recovered needs a target and levels")
	}
	return DehydrationRecovered{b, levels}, nil
}

// Kind returns "dehydration_recovered".
func (DehydrationRecovered) Kind() string { return "dehydration_recovered" }

// Levels returns the Exhaustion levels that were unlocked.
func (e DehydrationRecovered) Levels() int { return e.levels }

// MalnutritionRecovered: the creature ate its full daily food, so the Exhaustion
// levels Malnutrition holds can be removed. Its Kind is malnutrition_recovered.
type MalnutritionRecovered struct {
	creatureEnvironmentEventBase
	levels int
}

// NewMalnutritionRecovered returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or levels below 1.
func NewMalnutritionRecovered(target CombatantID, levels int) (MalnutritionRecovered, error) {
	b, err := newCreatureBase(target)
	if err != nil || levels < 1 {
		return MalnutritionRecovered{}, invalidInput("malnutrition recovered needs a target and levels")
	}
	return MalnutritionRecovered{b, levels}, nil
}

// Kind returns "malnutrition_recovered".
func (MalnutritionRecovered) Kind() string { return "malnutrition_recovered" }

// Levels returns the Exhaustion levels that were unlocked.
func (e MalnutritionRecovered) Levels() int { return e.levels }

// BreathHeldStarted: the creature started holding its breath. Its Kind is
// breath_held_started.
type BreathHeldStarted struct{ creatureEnvironmentEventBase }

// NewBreathHeldStarted returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target.
func NewBreathHeldStarted(target CombatantID) (BreathHeldStarted, error) {
	b, err := newCreatureBase(target)
	return BreathHeldStarted{b}, err
}

// Kind returns "breath_held_started".
func (BreathHeldStarted) Kind() string { return "breath_held_started" }

// RanOutOfBreath: the creature held its breath as long as it could and began to
// suffocate. Its Kind is ran_out_of_breath.
type RanOutOfBreath struct{ creatureEnvironmentEventBase }

// NewRanOutOfBreath returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target.
func NewRanOutOfBreath(target CombatantID) (RanOutOfBreath, error) {
	b, err := newCreatureBase(target)
	return RanOutOfBreath{b}, err
}

// Kind returns "ran_out_of_breath".
func (RanOutOfBreath) Kind() string { return "ran_out_of_breath" }

// BeganChoking: the creature began choking. Its Kind is began_choking.
type BeganChoking struct{ creatureEnvironmentEventBase }

// NewBeganChoking returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target.
func NewBeganChoking(target CombatantID) (BeganChoking, error) {
	b, err := newCreatureBase(target)
	return BeganChoking{b}, err
}

// Kind returns "began_choking".
func (BeganChoking) Kind() string { return "began_choking" }

// BreathRestored: the creature can breathe again and lost the Exhaustion levels it
// gained from suffocating; LevelsRemoved is how many, possibly 0. Its Kind is
// breath_restored.
type BreathRestored struct {
	creatureEnvironmentEventBase
	levelsRemoved int
}

// NewBreathRestored returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero target or negative levels.
func NewBreathRestored(target CombatantID, levelsRemoved int) (BreathRestored, error) {
	b, err := newCreatureBase(target)
	if err != nil || levelsRemoved < 0 {
		return BreathRestored{}, invalidInput("breath restored needs a target and levels of 0 or more")
	}
	return BreathRestored{b, levelsRemoved}, nil
}

// Kind returns "breath_restored".
func (BreathRestored) Kind() string { return "breath_restored" }

// LevelsRemoved returns the Exhaustion levels that were removed.
func (e BreathRestored) LevelsRemoved() int { return e.levelsRemoved }

// ObjectDamaged: an object lost Hit Points. Its Kind is object_damaged.
type ObjectDamaged struct {
	objectEnvironmentEventBase
	amount int
}

// NewObjectDamaged returns the event. It returns an error matching
// srderr.ErrInvalidInput for an amount below 1.
func NewObjectDamaged(amount int) (ObjectDamaged, error) {
	if amount < 1 {
		return ObjectDamaged{}, invalidInput("object damaged needs an amount of 1 or more")
	}
	return ObjectDamaged{amount: amount}, nil
}

// Kind returns "object_damaged".
func (ObjectDamaged) Kind() string { return "object_damaged" }

// Amount returns the Hit Points the object actually lost.
func (e ObjectDamaged) Amount() int { return e.amount }

// ObjectDamageDeflected: an instance of damage was below the object's Damage
// Threshold, so the object took none of it. Its Kind is object_damage_deflected.
type ObjectDamageDeflected struct{ objectEnvironmentEventBase }

// Kind returns "object_damage_deflected".
func (ObjectDamageDeflected) Kind() string { return "object_damage_deflected" }

// ObjectDestroyed: an object reached 0 Hit Points or was broken. Its Kind is
// object_destroyed.
type ObjectDestroyed struct{ objectEnvironmentEventBase }

// Kind returns "object_destroyed".
func (ObjectDestroyed) Kind() string { return "object_destroyed" }

// ObjectCollapsed: destroying a section made a composite object collapse. Its Kind
// is object_collapsed.
type ObjectCollapsed struct{ objectEnvironmentEventBase }

// Kind returns "object_collapsed".
func (ObjectCollapsed) Kind() string { return "object_collapsed" }
