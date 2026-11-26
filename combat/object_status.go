package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ObjectAbilityScores says whether an object has ability scores. Variants:
// NoAbilityScores, AssignedAbilityScores. A nil ObjectAbilityScores is
// NoAbilityScores.
type ObjectAbilityScores interface{ isObjectAbilityScores() }

// NoAbilityScores: the object has none. It cannot make ability checks and fails
// every saving throw (SRD 5.2.1, "Breaking Objects", "No Ability Scores", p. 178).
type NoAbilityScores struct{}

// AssignedAbilityScores: the host gave the object scores for some abilities. Build it
// with NewAssignedAbilityScores; the zero value assigns none.
type AssignedAbilityScores struct{ held [6]ability.Score }

// NewAssignedAbilityScores returns the scores m assigns. It returns an error matching
// srderr.ErrInvalidInput for an invalid ability key or a zero score. It never panics.
func NewAssignedAbilityScores(m map[ability.Ability]ability.Score) (AssignedAbilityScores, error) {
	var a AssignedAbilityScores
	for k, s := range m {
		if !k.Valid() || s.IsZero() {
			return AssignedAbilityScores{}, invalidInput("assigned score needs a valid ability and score")
		}
		a.held[k-ability.Strength] = s
	}
	return a, nil
}

// Score returns the score of a, the zero Score when none is assigned.
func (a AssignedAbilityScores) Score(k ability.Ability) ability.Score {
	if !k.Valid() {
		return ability.Score{}
	}
	return a.held[k-ability.Strength]
}

func (NoAbilityScores) isObjectAbilityScores()       {}
func (AssignedAbilityScores) isObjectAbilityScores() {}

// has reports whether the object has a score for k.
func hasScore(s ObjectAbilityScores, k ability.Ability) bool {
	a, ok := s.(AssignedAbilityScores)
	return ok && !a.Score(k).IsZero()
}

// ObjectAffinities is the rule for an object's damage affinities: Immunity to Poison
// and Psychic damage, plus whatever the GM rules (SRD 5.2.1, "Breaking Objects",
// "Damage Types and Objects", pp. 177-178). The zero value is ready to use.
type ObjectAffinities struct{}

// Standard returns the affinities of an object with no GM ruling: Immunity to Poison
// and Psychic damage and no Resistance or Vulnerability.
func (o ObjectAffinities) Standard() DamageAffinities { return o.StandardWith(NoAffinities()) }

// StandardWith returns the standard affinities with the rulings added: for example
// Vulnerability to Fire for paper or cloth.
func (ObjectAffinities) StandardWith(rulings DamageAffinities) DamageAffinities {
	rulings.immune |= typeBit(content.DamagePoison) | typeBit(content.DamagePsychic)
	return rulings
}

// ObjectStatus is an object's state: its size, Armor Class, Hit Points, optional
// Damage Threshold, damage affinities and ability scores (SRD 5.2.1, "Object", p.
// 185; "Breaking Objects", pp. 177-178). The SRD only suggests Armor Class and Hit
// Points, so the host may set any values. It holds no Temporary Hit Points. It is
// immutable and comparable. The zero value is invalid; see IsZero.
type ObjectStatus struct {
	size       content.CreatureSize
	armorClass d20.ArmorClass
	hitPoints  HitPoints
	threshold  DamageThreshold
	affinities DamageAffinities
	scores     ObjectAbilityScores
}

// NewObjectStatus returns the status of an object. A zero threshold means none and a
// nil scores means NoAbilityScores.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid size, Hit Points
// with Temporary Hit Points above 0 (an object has none), or an ability scores value
// that is neither NoAbilityScores nor AssignedAbilityScores. It never panics.
func NewObjectStatus(size content.CreatureSize, armorClass d20.ArmorClass, hitPoints HitPoints, threshold DamageThreshold,
	affinities DamageAffinities, scores ObjectAbilityScores) (ObjectStatus, error) {
	switch scores.(type) {
	case nil:
		scores = NoAbilityScores{}
	case NoAbilityScores, AssignedAbilityScores:
	default:
		return ObjectStatus{}, invalidInput("object ability scores are not a known variant")
	}
	if !size.Valid() || hitPoints.Temporary() != 0 {
		return ObjectStatus{}, invalidInput("object needs a valid size and no Temporary Hit Points")
	}
	return ObjectStatus{size, armorClass, hitPoints, threshold, affinities, scores}, nil
}

// IsZero reports whether s is the invalid zero ObjectStatus.
func (s ObjectStatus) IsZero() bool { return s.size == 0 }

// Size returns the object's size.
func (s ObjectStatus) Size() content.CreatureSize { return s.size }

// ArmorClass returns the object's Armor Class.
func (s ObjectStatus) ArmorClass() d20.ArmorClass { return s.armorClass }

// HitPoints returns the object's Hit Points.
func (s ObjectStatus) HitPoints() HitPoints { return s.hitPoints }

// Threshold returns the Damage Threshold; the zero DamageThreshold means none.
func (s ObjectStatus) Threshold() DamageThreshold { return s.threshold }

// Affinities returns the object's damage affinities.
func (s ObjectStatus) Affinities() DamageAffinities { return s.affinities }

// AbilityScores returns the object's ability scores.
func (s ObjectStatus) AbilityScores() ObjectAbilityScores { return s.scores }

// IsDestroyed reports whether the object is at 0 Hit Points (SRD 5.2.1, "Breaking
// Objects", pp. 177-178).
func (s ObjectStatus) IsDestroyed() bool { return s.hitPoints.Current() == 0 }

// AbilityCheck says whether the object can make an ability check with k: forbidden
// without a score for it (SRD 5.2.1, "Breaking Objects", "No Ability Scores", p. 178).
func (s ObjectStatus) AbilityCheck(k ability.Ability) ConditionPermission {
	if hasScore(s.scores, k) {
		return ConditionPermissionAllowed
	}
	return ConditionPermissionForbidden
}

// SavingThrow says what happens to the object's saving throw with k: an automatic
// failure without a score for it, otherwise unaffected. The caller builds the
// failure with d20.FailWithoutRoll and d20.Automatic.
func (s ObjectStatus) SavingThrow(k ability.Ability) SavingThrowEffect {
	if hasScore(s.scores, k) {
		return SavingThrowEffectUnaffected
	}
	return SavingThrowEffectAutomaticFailure
}
