package combat

// MountRejection is the closed set of reasons MountCreature refuses, in the order it
// checks them (mounted-and-underwater-combat.md FR-017). Each is a value type of its own;
// String returns the snake_case reason, the value of srd5e.combat.result. Variants:
// MountingRiderNotActiveRejection, MountingRiderAlreadyMountedRejection,
// MountingMountUnwillingRejection, MountingMountNotLargerRejection,
// MountingMountAnatomyRejection, MountingExoticSaddleRequiredRejection,
// MountingOutOfReachRejection, MountingMountCannotBeControlledRejection,
// MountingNoMovementRemainingRejection, MountingInsufficientMovementRejection.
type MountRejection interface {
	isMountRejection()
	String() string
}

// DismountRejection is the closed set of reasons DismountCreature refuses. Variants:
// MountingRiderNotActiveRejection, MountingRiderNotMountedRejection,
// MountingNoMovementRemainingRejection, MountingInsufficientMovementRejection.
type DismountRejection interface {
	isDismountRejection()
	String() string
}

// FallingOffRejection is the closed set of reasons ResolveFallingOff refuses. Variants:
// MountingRiderNotMountedRejection.
type FallingOffRejection interface {
	isFallingOffRejection()
	String() string
}

// MountingRiderNotActiveRejection: it is not the rider's turn, and mounting and
// dismounting happen during the rider's move (SRD 5.2.1, "Mounting and Dismounting", p. 15).
type MountingRiderNotActiveRejection struct{}

// MountingRiderAlreadyMountedRejection: the rider is already on a mount (a package decision).
type MountingRiderAlreadyMountedRejection struct{}

// MountingMountUnwillingRejection: the creature is not willing (p. 15).
type MountingMountUnwillingRejection struct{}

// MountingMountNotLargerRejection: the mount is not at least one size larger than the rider (p. 15).
type MountingMountNotLargerRejection struct{}

// MountingMountAnatomyRejection: the creature has no appropriate anatomy (p. 15).
type MountingMountAnatomyRejection struct{}

// MountingExoticSaddleRequiredRejection: an aquatic or flying mount needs an Exotic Saddle (p. 100).
type MountingExoticSaddleRequiredRejection struct{}

// MountingOutOfReachRejection: the mount is more than 5 feet from the rider (p. 15).
type MountingOutOfReachRejection struct{ distanceFeet int }

// DistanceFeet returns the distance the host reported.
func (r MountingOutOfReachRejection) DistanceFeet() int { return r.distanceFeet }

// MountingMountCannotBeControlledRejection: the rider asked to control a mount that has not
// been trained to accept a rider (p. 16).
type MountingMountCannotBeControlledRejection struct{}

// MountingNoMovementRemainingRejection: the rider has no walking movement left.
type MountingNoMovementRemainingRejection struct{}

// MountingInsufficientMovementRejection: the rider has some movement left, but less than the cost.
type MountingInsufficientMovementRejection struct{ shortfallFeet int }

// ShortfallFeet returns how many feet short the rider is: the cost minus the movement left.
func (r MountingInsufficientMovementRejection) ShortfallFeet() int { return r.shortfallFeet }

// MountingRiderNotMountedRejection: the rider is not on a mount, so there is nothing to
// dismount or fall from.
type MountingRiderNotMountedRejection struct{}

// String returns "rider_not_active".
func (MountingRiderNotActiveRejection) String() string { return "rider_not_active" }

// String returns "rider_already_mounted".
func (MountingRiderAlreadyMountedRejection) String() string { return "rider_already_mounted" }

// String returns "mount_unwilling".
func (MountingMountUnwillingRejection) String() string { return "mount_unwilling" }

// String returns "mount_not_larger".
func (MountingMountNotLargerRejection) String() string { return "mount_not_larger" }

// String returns "mount_anatomy".
func (MountingMountAnatomyRejection) String() string { return "mount_anatomy" }

// String returns "exotic_saddle_required".
func (MountingExoticSaddleRequiredRejection) String() string { return "exotic_saddle_required" }

// String returns "out_of_reach".
func (MountingOutOfReachRejection) String() string { return "out_of_reach" }

// String returns "mount_cannot_be_controlled".
func (MountingMountCannotBeControlledRejection) String() string { return "mount_cannot_be_controlled" }

// String returns "no_movement_remaining".
func (MountingNoMovementRemainingRejection) String() string { return "no_movement_remaining" }

// String returns "insufficient_movement".
func (MountingInsufficientMovementRejection) String() string { return "insufficient_movement" }

// String returns "rider_not_mounted".
func (MountingRiderNotMountedRejection) String() string { return "rider_not_mounted" }

func (MountingRiderNotActiveRejection) isMountRejection()          {}
func (MountingRiderAlreadyMountedRejection) isMountRejection()     {}
func (MountingMountUnwillingRejection) isMountRejection()          {}
func (MountingMountNotLargerRejection) isMountRejection()          {}
func (MountingMountAnatomyRejection) isMountRejection()            {}
func (MountingExoticSaddleRequiredRejection) isMountRejection()    {}
func (MountingOutOfReachRejection) isMountRejection()              {}
func (MountingMountCannotBeControlledRejection) isMountRejection() {}
func (MountingNoMovementRemainingRejection) isMountRejection()     {}
func (MountingInsufficientMovementRejection) isMountRejection()    {}
func (MountingRiderNotActiveRejection) isDismountRejection()       {}
func (MountingRiderNotMountedRejection) isDismountRejection()      {}
func (MountingNoMovementRemainingRejection) isDismountRejection()  {}
func (MountingInsufficientMovementRejection) isDismountRejection() {}
func (MountingRiderNotMountedRejection) isFallingOffRejection()    {}

// movementRefusal is the pair of reasons shared by mounting and dismounting.
type movementRefusal interface {
	MountRejection
	DismountRejection
}
