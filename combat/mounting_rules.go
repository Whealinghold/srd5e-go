package combat

// The mounting rules, pure and silent (SRD 5.2.1, "Mounted Combat", pp. 15 to 16). They
// run on commands the constructors validated, and report the first reason in the order of
// mounted-and-underwater-combat.md FR-017.

// maxMountingReachFeet is how far a creature may be and still be mounted: "within 5 feet" (p. 15).
const maxMountingReachFeet = 5

// movementShortfall returns the cost of mounting or dismounting, half the walking Speed
// rounded down and never the Dash extra (p. 15), and why the rider cannot pay it. The
// reason is nil when movement left covers the cost.
func movementShortfall(f MountingTurnFacts) (cost int, refusal movementRefusal) {
	cost = f.walkingSpeedFeet / 2
	switch {
	case f.walkingMovementLeftFeet <= 0:
		return cost, MountingNoMovementRemainingRejection{}
	case f.walkingMovementLeftFeet < cost:
		return cost, MountingInsufficientMovementRejection{cost - f.walkingMovementLeftFeet}
	}
	return cost, nil
}

// mountRefusal returns the movement cost of mounting and the first reason the rider cannot
// mount (nil when it can).
func mountRefusal(c MountCreatureCommand) (int, MountRejection) {
	switch {
	case c.turn.riderTurn != RiderTurnActive:
		return 0, MountingRiderNotActiveRejection{}
	case c.status.IsMounted():
		return 0, MountingRiderAlreadyMountedRejection{}
	case c.facts.willingness != MountWillingnessWilling:
		return 0, MountingMountUnwillingRejection{}
	case c.mountSize <= c.riderSize:
		return 0, MountingMountNotLargerRejection{}
	case c.facts.anatomy != MountAnatomyAppropriate:
		return 0, MountingMountAnatomyRejection{}
	case c.facts.habitat == MountHabitatAquaticOrFlying && c.facts.saddle != ExoticSaddleFitted:
		return 0, MountingExoticSaddleRequiredRejection{}
	case c.distanceFeet > maxMountingReachFeet:
		return 0, MountingOutOfReachRejection{c.distanceFeet}
	case c.control == MountControlControlled && c.facts.training != MountTrainingTrainedToAcceptRider:
		return 0, MountingMountCannotBeControlledRejection{}
	}
	cost, r := movementShortfall(c.turn)
	if r != nil {
		return cost, r
	}
	return cost, nil
}

// dismountRefusal returns the movement cost of dismounting and the first reason the rider
// cannot dismount (nil when it can).
func dismountRefusal(c DismountCreatureCommand) (int, DismountRejection) {
	switch {
	case c.turn.riderTurn != RiderTurnActive:
		return 0, MountingRiderNotActiveRejection{}
	case !c.status.IsMounted():
		return 0, MountingRiderNotMountedRejection{}
	}
	cost, r := movementShortfall(c.turn)
	if r != nil {
		return cost, r
	}
	return cost, nil
}

// ridingOutcome returns what a controlled or independent ride does to the mount's
// Initiative and whose turn it moves on (p. 16).
func ridingOutcome(control MountControl, rider CombatantID) (MountInitiativeChange, MountTurn) {
	if control == MountControlControlled {
		return MountInitiativeMatchesRider{Rider: rider}, MountTurnOnRidersTurn
	}
	return MountInitiativeUnchanged{}, MountTurnOnOwnTurn
}
