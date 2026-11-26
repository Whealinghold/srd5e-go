package combat

// Landing, obstacles and reach (SRD 5.2.1, "Long Jump", p. 185; "High Jump", p. 183). Pure
// and silent; whole feet, every division and multiplication rounded down ("Round Down", p. 187).

// JumpLanding is the closed set of what a landing needs. Variants: JumpLandsClear,
// JumpLandingCheckRequired.
//
//sumtype:decl
type JumpLanding interface{ isJumpLanding() }

// JumpLandsClear is a landing that needs no check.
type JumpLandsClear struct{}

// JumpLandingCheckRequired is a Long Jump landing in Difficult Terrain: the check, whose
// failure means the creature has the Prone condition.
type JumpLandingCheckRequired struct{ Requirement MovementCheckRequirement }

func (JumpLandsClear) isJumpLanding()           {}
func (JumpLandingCheckRequired) isJumpLanding() {}

// JumpLandingFor returns the check a landing needs: only a Long Jump into Difficult Terrain
// has one (the SRD states it under Long Jump alone).
func JumpLandingFor(kind JumpKind, terrain Terrain) JumpLanding {
	if kind == JumpKindLong && terrain == TerrainDifficult {
		return JumpLandingCheckRequired{RequirementFor(MovementCheckKindLongJumpLanding)}
	}
	return JumpLandsClear{}
}

// JumpObstacle is the closed set of what a low obstacle in a Long Jump needs. Variants:
// JumpObstacleCheckRequired, JumpObstacleNotCovered.
//
//sumtype:decl
type JumpObstacle interface{ isJumpObstacle() }

// JumpObstacleCheckRequired is an obstacle no taller than a quarter of the jump: the check,
// whose failure means the creature hits the obstacle.
type JumpObstacleCheckRequired struct{ Requirement MovementCheckRequirement }

// JumpObstacleNotCovered is an obstacle, or a jump of 0 feet, for which the SRD states no rule.
type JumpObstacleNotCovered struct{}

func (JumpObstacleCheckRequired) isJumpObstacle() {}
func (JumpObstacleNotCovered) isJumpObstacle()    {}

// ObstacleForLongJump returns the check for an obstacle of obstacleHeightFeet in a Long Jump
// of distanceFeet: required when the jump is at least 1 foot and the obstacle is at most
// distanceFeet / 4 (rounded down), not covered otherwise. It returns an error matching
// srderr.ErrInvalidInput for a negative distance or height. It never panics.
func ObstacleForLongJump(distanceFeet, obstacleHeightFeet int) (JumpObstacle, error) {
	if distanceFeet < 0 || obstacleHeightFeet < 0 {
		return nil, invalidInput("negative jump distance or obstacle height")
	}
	if distanceFeet >= 1 && obstacleHeightFeet <= distanceFeet/4 {
		return JumpObstacleCheckRequired{RequirementFor(MovementCheckKindLongJumpObstacle)}, nil
	}
	return JumpObstacleNotCovered{}, nil
}

// HighJumpReachFeet returns the jump height plus one and a half times the creature's height,
// that product rounded down, in whole feet; a sum past the largest int saturates. It returns
// an error matching srderr.ErrInvalidInput for a negative jump height or a creature height
// below 1 foot. It never panics.
func HighJumpReachFeet(jumpHeightFeet, creatureHeightFeet int) (int, error) {
	if jumpHeightFeet < 0 || creatureHeightFeet < 1 {
		return 0, invalidInput("negative jump height or creature height below 1 foot")
	}
	// height * 3 / 2 rounded down is height + height / 2, which cannot overflow.
	return satAdd(jumpHeightFeet, satAdd(creatureHeightFeet, creatureHeightFeet/2)), nil
}
