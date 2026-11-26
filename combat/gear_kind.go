package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// DeployedGearKind is a piece of gear set out on the ground (SRD 5.2.1, "Adventuring
// Gear", pp. 96-98). The zero value is invalid. Safe for concurrent use.
type DeployedGearKind int

// The four kinds of deployed gear.
const (
	// DeployedGearKindCaltrops: Caltrops, scattered in a 5-foot square.
	DeployedGearKindCaltrops DeployedGearKind = iota + 1
	// DeployedGearKindBallBearings: Ball Bearings, spread over a 10-foot square.
	DeployedGearKindBallBearings
	// DeployedGearKindPouredOil: Oil poured on a 5-foot square.
	DeployedGearKindPouredOil
	// DeployedGearKindHuntingTrap: a Hunting Trap, set on its pressure plate.
	DeployedGearKindHuntingTrap
)

func deployedGearKindNames() []string {
	return []string{"caltrops", "ball_bearings", "poured_oil", "hunting_trap"}
}

// Valid reports whether k is one of the four kinds.
func (k DeployedGearKind) Valid() bool {
	return k >= DeployedGearKindCaltrops && k <= DeployedGearKindHuntingTrap
}

// String returns "caltrops", "ball_bearings", "poured_oil", "hunting_trap" or "unknown".
func (k DeployedGearKind) String() string { return enumName(deployedGearKindNames(), int(k)) }

// ItemID returns the content id of the equipment entry the kind is made of ("caltrops",
// "ball_bearings", "oil", "hunting_trap"); the zero ItemID for an invalid kind.
func (k DeployedGearKind) ItemID() content.ItemID {
	switch k {
	case DeployedGearKindCaltrops:
		return content.MustItemID("caltrops")
	case DeployedGearKindBallBearings:
		return content.MustItemID("ball_bearings")
	case DeployedGearKindPouredOil:
		return content.MustItemID("oil")
	case DeployedGearKindHuntingTrap:
		return content.MustItemID("hunting_trap")
	}
	return ""
}

// HoldKind is a way a creature is held by gear (SRD 5.2.1, "Adventuring Gear", pp.
// 96-99). The zero value is invalid. Safe for concurrent use.
type HoldKind int

// The five kinds of hold.
const (
	// HoldKindNet: a thrown Net.
	HoldKindNet HoldKind = iota + 1
	// HoldKindChain: a Chain wound round a creature.
	HoldKindChain
	// HoldKindManacles: Manacles locked on a creature.
	HoldKindManacles
	// HoldKindRope: a Rope tied round a creature.
	HoldKindRope
	// HoldKindHuntingTrap: a Hunting Trap's grip on a creature.
	HoldKindHuntingTrap
)

func holdKindNames() []string { return []string{"net", "chain", "manacles", "rope", "hunting_trap"} }

// Valid reports whether k is one of the five kinds.
func (k HoldKind) Valid() bool { return k >= HoldKindNet && k <= HoldKindHuntingTrap }

// String returns "net", "chain", "manacles", "rope", "hunting_trap" or "unknown".
func (k HoldKind) String() string { return enumName(holdKindNames(), int(k)) }

// ItemID returns the content id of the equipment entry the hold comes from ("net", "chain",
// "manacles", "rope", "hunting_trap"); the zero ItemID for an invalid kind.
func (k HoldKind) ItemID() content.ItemID {
	if !k.Valid() {
		return ""
	}
	return content.MustItemID(holdKindNames()[k-1])
}

// BindingKind is a way to bind an unwilling creature: Chain, Manacles or Rope. The zero
// value is invalid. Safe for concurrent use.
type BindingKind int

// The three ways to bind a creature.
const (
	// BindingKindChain: a Chain.
	BindingKindChain BindingKind = iota + 1
	// BindingKindManacles: Manacles.
	BindingKindManacles
	// BindingKindRope: a Rope.
	BindingKindRope
)

// Valid reports whether k is one of the three.
func (k BindingKind) Valid() bool { return k >= BindingKindChain && k <= BindingKindRope }

// String returns "chain", "manacles", "rope" or "unknown".
func (k BindingKind) String() string { return enumName([]string{"chain", "manacles", "rope"}, int(k)) }

// HoldKind returns the hold the binding makes; the zero HoldKind for an invalid kind.
func (k BindingKind) HoldKind() HoldKind {
	switch k {
	case BindingKindChain:
		return HoldKindChain
	case BindingKindManacles:
		return HoldKindManacles
	case BindingKindRope:
		return HoldKindRope
	}
	return 0
}

// GroundLevel says whether the host found the ground level, a host fact the engine does not
// model (SRD 5.2.1, "Ball Bearings", p. 96; "Oil", p. 98). The zero value is invalid.
type GroundLevel int

// The two answers.
const (
	// GroundLevelLevel: the ground is level.
	GroundLevelLevel GroundLevel = iota + 1
	// GroundLevelNotLevel: the ground is not level.
	GroundLevelNotLevel
)

// Valid reports whether g is one of the two answers.
func (g GroundLevel) Valid() bool { return g == GroundLevelLevel || g == GroundLevelNotLevel }

// String returns "level", "not_level" or "unknown".
func (g GroundLevel) String() string { return enumName([]string{"level", "not_level"}, int(g)) }

// AreaTrigger is what a creature does to set deployed gear off (SRD 5.2.1, "Oil", p. 98).
// The zero value is invalid.
type AreaTrigger int

// The two triggers.
const (
	// AreaTriggerEnters: the creature enters the area (or steps on the plate).
	AreaTriggerEnters AreaTrigger = iota + 1
	// AreaTriggerEndsTurn: the creature ends its turn in the area.
	AreaTriggerEndsTurn
)

// Valid reports whether t is one of the two triggers.
func (t AreaTrigger) Valid() bool { return t == AreaTriggerEnters || t == AreaTriggerEndsTurn }

// String returns "enters", "ends_turn" or "unknown".
func (t AreaTrigger) String() string { return enumName([]string{"enters", "ends_turn"}, int(t)) }

// HoldRestraint says whether a hold gives the Restrained condition (SRD 5.2.1, "Chain",
// "Manacles", "Net", "Rope"). The host states it for Chain, Manacles and Rope. The zero value
// is invalid.
type HoldRestraint int

// The two answers.
const (
	// HoldRestraintRestrains: the hold gives the Restrained condition.
	HoldRestraintRestrains HoldRestraint = iota + 1
	// HoldRestraintDoesNotRestrain: the hold does not.
	HoldRestraintDoesNotRestrain
)

// Valid reports whether r is one of the two answers.
func (r HoldRestraint) Valid() bool {
	return r == HoldRestraintRestrains || r == HoldRestraintDoesNotRestrain
}

// String returns "restrains", "does_not_restrain" or "unknown".
func (r HoldRestraint) String() string {
	return enumName([]string{"restrains", "does_not_restrain"}, int(r))
}

// HoldReleaseMethod is a way out of a hold. The zero value is invalid.
type HoldReleaseMethod int

// The four ways.
const (
	// HoldReleaseMethodEscape: the held creature (or a helper, for a Net or Hunting Trap) frees it.
	HoldReleaseMethodEscape HoldReleaseMethod = iota + 1
	// HoldReleaseMethodBurst: a Strength (Athletics) check breaks Chain, Manacles or Rope.
	HoldReleaseMethodBurst
	// HoldReleaseMethodPickLock: Thieves' Tools pick the Manacles' lock.
	HoldReleaseMethodPickLock
	// HoldReleaseMethodKey: the Manacles' key unlocks them.
	HoldReleaseMethodKey
)

// Valid reports whether m is one of the four ways.
func (m HoldReleaseMethod) Valid() bool {
	return m >= HoldReleaseMethodEscape && m <= HoldReleaseMethodKey
}

// String returns "escape", "burst", "pick_lock", "key" or "unknown".
func (m HoldReleaseMethod) String() string {
	return enumName([]string{"escape", "burst", "pick_lock", "key"}, int(m))
}

// HoldEndCause is why a hold ended; the value of srd5e.combat.gear.end_cause. The zero
// value is invalid.
type HoldEndCause int

// The four causes.
const (
	// HoldEndCauseEscaped: the creature or a helper escaped.
	HoldEndCauseEscaped HoldEndCause = iota + 1
	// HoldEndCauseBurst: the hold was burst.
	HoldEndCauseBurst
	// HoldEndCauseDestroyed: the Net was destroyed.
	HoldEndCauseDestroyed
	// HoldEndCauseUnlocked: the Manacles were unlocked.
	HoldEndCauseUnlocked
)

// Valid reports whether c is one of the four causes.
func (c HoldEndCause) Valid() bool { return c >= HoldEndCauseEscaped && c <= HoldEndCauseUnlocked }

// String returns "escaped", "burst", "destroyed", "unlocked" or "unknown".
func (c HoldEndCause) String() string {
	return enumName([]string{"escaped", "burst", "destroyed", "unlocked"}, int(c))
}

// Willingness says whether the creature to bind is willing; the rules bind an unwilling
// creature (SRD 5.2.1, "Chain", p. 96). The zero value is invalid.
type Willingness int

// The two answers.
const (
	// WillingnessWilling: the creature is willing.
	WillingnessWilling Willingness = iota + 1
	// WillingnessUnwilling: the creature is unwilling.
	WillingnessUnwilling
)

// Valid reports whether w is one of the two answers.
func (w Willingness) Valid() bool { return w == WillingnessWilling || w == WillingnessUnwilling }

// String returns "willing", "unwilling" or "unknown".
func (w Willingness) String() string { return enumName([]string{"willing", "unwilling"}, int(w)) }

// ThrowerSight says whether the thrower can see the target of a Net, a host fact (SRD 5.2.1,
// "Net", p. 98). The zero value is invalid.
type ThrowerSight int

// The two answers.
const (
	// ThrowerSightSeesTarget: the thrower can see the target.
	ThrowerSightSeesTarget ThrowerSight = iota + 1
	// ThrowerSightCannotSeeTarget: it cannot.
	ThrowerSightCannotSeeTarget
)

// Valid reports whether s is one of the two answers.
func (s ThrowerSight) Valid() bool {
	return s == ThrowerSightSeesTarget || s == ThrowerSightCannotSeeTarget
}

// String returns "sees_target", "cannot_see_target" or "unknown".
func (s ThrowerSight) String() string {
	return enumName([]string{"sees_target", "cannot_see_target"}, int(s))
}

// DeployReach is how far from the deployer gear may be set: ReachWithin(n) or ReachUnstated.
// The zero value is ReachUnstated.
type DeployReach struct {
	feet   int
	stated bool
}

// ReachWithin returns the reach of gear that may be set within feet of the deployer.
func ReachWithin(feet int) DeployReach { return DeployReach{feet, true} }

// ReachUnstated returns the reach of gear whose placement the SRD states no distance for.
func ReachUnstated() DeployReach { return DeployReach{} }

// Feet returns the distance; ok is false when the reach is unstated.
func (r DeployReach) Feet() (feet int, ok bool) { return r.feet, r.stated }

// Allows reports whether a distance of 0 or more is within the reach.
func (r DeployReach) Allows(distanceFeet int) bool { return !r.stated || distanceFeet <= r.feet }

// DeployedGearProfile holds the facts of one kind of deployed gear (SRD 5.2.1, pp. 96-98;
// table 4.9a of deployed-and-restraining-gear.md). The zero value is the profile of no kind.
// Immutable and comparable.
type DeployedGearProfile struct {
	kind            DeployedGearKind
	reach           DeployReach
	areaSideFeet    int
	levelGround     bool
	saveDC          d20.DifficultyClass
	recoveryMinutes int
}

// DeployedGearProfileFor returns the profile of kind, the zero profile for an invalid kind.
func DeployedGearProfileFor(kind DeployedGearKind) DeployedGearProfile {
	switch kind {
	case DeployedGearKindCaltrops:
		return DeployedGearProfile{kind, ReachWithin(5), 5, false, 15, 10}
	case DeployedGearKindBallBearings:
		return DeployedGearProfile{kind, ReachWithin(10), 10, true, 10, 10}
	case DeployedGearKindPouredOil:
		return DeployedGearProfile{kind, ReachWithin(5), 5, true, 0, 0}
	case DeployedGearKindHuntingTrap:
		return DeployedGearProfile{kind, ReachUnstated(), 0, false, 13, 0}
	}
	return DeployedGearProfile{}
}

// Kind returns the kind the profile is of.
func (p DeployedGearProfile) Kind() DeployedGearKind { return p.kind }

// Reach returns how far from the deployer the gear may be set.
func (p DeployedGearProfile) Reach() DeployReach { return p.reach }

// AreaSideFeet returns the side of the square area, 0 for the Hunting Trap, which has none.
func (p DeployedGearProfile) AreaSideFeet() int { return p.areaSideFeet }

// RequiresLevelGround reports whether the SRD asks for level ground.
func (p DeployedGearProfile) RequiresLevelGround() bool { return p.levelGround }

// SaveDifficultyClass returns the Difficulty Class of the Dexterity saving throw; ok is
// false for Oil, which allows none.
func (p DeployedGearProfile) SaveDifficultyClass() (dc d20.DifficultyClass, ok bool) {
	return p.saveDC, p.saveDC != 0
}

// SaveAbility returns Dexterity; ok is false for Oil, which allows no save.
func (p DeployedGearProfile) SaveAbility() (ability.Ability, bool) {
	return ability.Dexterity, p.saveDC != 0
}

// RecoveryMinutes returns the minutes it takes to recover the gear, 0 when it cannot be
// recovered. The engine reads no clock; this is data.
func (p DeployedGearProfile) RecoveryMinutes() int { return p.recoveryMinutes }

// HoldCheck is an ability check the hold rules ask for: a skill, whose ability is
// Skill.Ability(), against a Difficulty Class held exactly as printed. The zero value is no
// check.
type HoldCheck struct {
	skill content.Skill
	dc    d20.DifficultyClass
}

// Skill returns the skill of the check.
func (c HoldCheck) Skill() content.Skill { return c.skill }

// Ability returns the ability of the skill.
func (c HoldCheck) Ability() ability.Ability { return c.skill.Ability() }

// DifficultyClass returns the Difficulty Class.
func (c HoldCheck) DifficultyClass() d20.DifficultyClass { return c.dc }

// HoldProfile holds the facts of one kind of hold (SRD 5.2.1, pp. 96-99; table 4.9b of
// deployed-and-restraining-gear.md). The zero value is the profile of no kind. Immutable
// and comparable.
type HoldProfile struct {
	kind                HoldKind
	bind, escape, burst HoldCheck
	helperMayRelease    bool
	movementLimitFeet   int
}

// HoldProfileFor returns the profile of kind, the zero profile for an invalid kind.
func HoldProfileFor(kind HoldKind) HoldProfile {
	const athletics, acrobatics, sleight = content.SkillAthletics, content.SkillAcrobatics, content.SkillSleightOfHand
	switch kind {
	case HoldKindNet:
		return HoldProfile{kind: kind, escape: HoldCheck{athletics, 10}, helperMayRelease: true}
	case HoldKindChain:
		return HoldProfile{kind, HoldCheck{athletics, 13}, HoldCheck{acrobatics, 18}, HoldCheck{athletics, 20}, false, 0}
	case HoldKindManacles:
		return HoldProfile{kind, HoldCheck{sleight, 13}, HoldCheck{sleight, 20}, HoldCheck{athletics, 25}, false, 0}
	case HoldKindRope:
		return HoldProfile{kind, HoldCheck{sleight, 10}, HoldCheck{acrobatics, 15}, HoldCheck{athletics, 20}, false, 0}
	case HoldKindHuntingTrap:
		return HoldProfile{kind: kind, escape: HoldCheck{athletics, 13}, helperMayRelease: true, movementLimitFeet: 3}
	}
	return HoldProfile{}
}

// Kind returns the kind the profile is of.
func (p HoldProfile) Kind() HoldKind { return p.kind }

// BindCheck returns the binder's check; ok is false for a Net and a Hunting Trap, which
// are not applied by a check.
func (p HoldProfile) BindCheck() (HoldCheck, bool) { return p.bind, p.bind.dc != 0 }

// EscapeCheck returns the check to escape the hold.
func (p HoldProfile) EscapeCheck() HoldCheck { return p.escape }

// BurstCheck returns the Strength (Athletics) check to burst the hold; ok is false for a
// Net and a Hunting Trap, which cannot be burst.
func (p HoldProfile) BurstCheck() (HoldCheck, bool) { return p.burst, p.burst.dc != 0 }

// HelperMayRelease reports whether a creature other than the held one may escape the hold
// (a Net, a Hunting Trap).
func (p HoldProfile) HelperMayRelease() bool { return p.helperMayRelease }

// DefaultMovementLimitFeet returns the length of a Hunting Trap's chain, "typically 3 feet",
// a default the host may override; 0 for the other kinds.
func (p HoldProfile) DefaultMovementLimitFeet() int { return p.movementLimitFeet }
