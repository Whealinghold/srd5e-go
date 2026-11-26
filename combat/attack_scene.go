package combat

// AttackScene is the Game Master's scene of one attack, stated by the host because the engine has no
// map (combat-action-resolution.md section 1): the distance to the target, who sees whom and whether
// the attacker's source of fear is in line of sight. It becomes the ConditionAttackFacts of the attack
// with the two ids. Build it with NewAttackScene; the zero value is a scene at distance 0 in which
// nobody sees anybody. Immutable, comparable and safe for concurrent use.
type AttackScene struct {
	distanceFeet       int
	attackerSeesTarget SightRelation
	targetSeesAttacker SightRelation
	attackerFearSource LineOfSight
}

// NewAttackScene returns the scene. It returns an error matching srderr.ErrInvalidInput for a negative
// distance. A sight relation or line of sight that is not one of the defined values counts as not
// seeing and not in line of sight, as ConditionAttackFacts says. It never panics.
func NewAttackScene(distanceFeet int, attackerSeesTarget, targetSeesAttacker SightRelation, attackerFearSource LineOfSight) (AttackScene, error) {
	if distanceFeet < 0 {
		return AttackScene{}, invalidInput("a scene needs a distance of 0 or more")
	}
	return AttackScene{distanceFeet, attackerSeesTarget, targetSeesAttacker, attackerFearSource}, nil
}

// DistanceFeet returns the distance to the target in feet.
func (s AttackScene) DistanceFeet() int { return s.distanceFeet }

// AttackerSeesTarget returns whether the attacker can somehow see the target.
func (s AttackScene) AttackerSeesTarget() SightRelation { return s.attackerSeesTarget }

// TargetSeesAttacker returns whether the target can somehow see the attacker.
func (s AttackScene) TargetSeesAttacker() SightRelation { return s.targetSeesAttacker }

// AttackerFearSource returns whether the attacker's source of fear is in line of sight.
func (s AttackScene) AttackerFearSource() LineOfSight { return s.attackerFearSource }

// facts returns the ConditionAttackFacts of the scene for the two creatures.
func (s AttackScene) facts(attacker, target CombatantID) ConditionAttackFacts {
	return ConditionAttackFacts{AttackerID: attacker, TargetID: target, DistanceFeet: s.distanceFeet,
		AttackerSeesTarget: s.attackerSeesTarget, TargetSeesAttacker: s.targetSeesAttacker, AttackerFearSource: s.attackerFearSource}
}
