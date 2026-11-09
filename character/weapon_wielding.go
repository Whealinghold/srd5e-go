package character

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/content"
)

// Grip says how many hands hold a weapon. The zero value reads as one hand.
// These are host facts (SRD 5.2.1, pp. 89-90).
type Grip int

const (
	// GripOneHanded: the weapon is held in one hand.
	GripOneHanded Grip = iota + 1
	// GripTwoHanded: the weapon is held in two hands.
	GripTwoHanded
)

// HandState says whether the attacker has a free hand. The zero value reads as a
// free hand being available.
type HandState int

const (
	// HandStateFreeHandAvailable: a hand is free.
	HandStateFreeHandAvailable HandState = iota + 1
	// HandStateNoFreeHand: no hand is free.
	HandStateNoFreeHand
)

// LoadingState says whether a Loading weapon was already fired in this action,
// Bonus Action or Reaction (SRD 5.2.1, "Loading", p. 90). The host resets it
// per activation. The zero value reads as ready.
type LoadingState int

const (
	// LoadingReady: the weapon has not been fired this activation.
	LoadingReady LoadingState = iota + 1
	// LoadingAlreadyFiredThisActivation: the weapon was fired this activation.
	LoadingAlreadyFiredThisActivation
)

// TwoHandedExemption is the host's report that the weapon's Two-Handed
// condition is waived (the Lance "unless mounted"). The zero value reads as no
// exemption.
type TwoHandedExemption int

const (
	// TwoHandedExemptionApplies: the condition is met.
	TwoHandedExemptionApplies TwoHandedExemption = iota + 1
	// TwoHandedNoExemption: no exemption applies.
	TwoHandedNoExemption
)

// ErrSupplyEmpty is matched (errors.Is) by the error AmmunitionSupply.Expend
// returns on a supply of 0 pieces.
var ErrSupplyEmpty = errors.New("character: ammunition supply is empty")

// AmmunitionSupply is a count of pieces of one ammunition type (SRD 5.2.1,
// "Ammunition", p. 89). Build it with NewAmmunitionSupply. Immutable and
// comparable.
type AmmunitionSupply struct {
	pieces int
	kind   content.AmmunitionType
}

// NewAmmunitionSupply returns a supply of pieces of kind. It returns an error
// matching srderr.ErrInvalidInput, naming pieces, for a negative count and
// naming ammunitionType for an invalid type.
func NewAmmunitionSupply(pieces int, kind content.AmmunitionType) (AmmunitionSupply, error) {
	switch {
	case pieces < 0:
		return AmmunitionSupply{}, invalidInput("pieces of an ammunition supply: negative")
	case !kind.Valid():
		return AmmunitionSupply{}, invalidInput("ammunitionType of an ammunition supply")
	}
	return AmmunitionSupply{pieces: pieces, kind: kind}, nil
}

// MustAmmunitionSupply is NewAmmunitionSupply for values known to be valid. It
// panics with the error that function returns, and only then.
func MustAmmunitionSupply(pieces int, kind content.AmmunitionType) AmmunitionSupply {
	s, err := NewAmmunitionSupply(pieces, kind)
	if err != nil {
		panic(err)
	}
	return s
}

// Pieces returns the number of pieces.
func (s AmmunitionSupply) Pieces() int { return s.pieces }

// Type returns the ammunition type.
func (s AmmunitionSupply) Type() content.AmmunitionType { return s.kind }

// Expend returns the supply with one piece fewer. On 0 pieces it returns an
// error matching ErrSupplyEmpty.
func (s AmmunitionSupply) Expend() (AmmunitionSupply, error) {
	if s.pieces == 0 {
		return AmmunitionSupply{}, ErrSupplyEmpty
	}
	return AmmunitionSupply{pieces: s.pieces - 1, kind: s.kind}, nil
}

// RecoverAfterFight returns the pieces recovered after a fight: half of
// piecesUsedInFight, rounded down. It returns an error matching
// srderr.ErrInvalidInput for a negative count. The 1 minute is the host's.
func (s AmmunitionSupply) RecoverAfterFight(piecesUsedInFight int) (int, error) {
	if piecesUsedInFight < 0 {
		return 0, invalidInput("piecesUsedInFight: negative")
	}
	return piecesUsedInFight / 2, nil
}

// WeaponWielding is how a weapon is held and fed: host facts. Construct it with
// a keyed literal; WithAmmunition adds the supply.
type WeaponWielding struct {
	Grip      Grip
	Exemption TwoHandedExemption
	Hands     HandState
	Loading   LoadingState

	supply    AmmunitionSupply
	hasSupply bool
}

// WithAmmunition returns w carrying supply; w is unchanged.
func (w WeaponWielding) WithAmmunition(supply AmmunitionSupply) WeaponWielding {
	w.supply, w.hasSupply = supply, true
	return w
}

// Ammunition returns the carried supply and true, or false when none is carried.
func (w WeaponWielding) Ammunition() (AmmunitionSupply, bool) { return w.supply, w.hasSupply }

// defaultReachFeet is a creature's reach unless a feature says otherwise (SRD
// 5.2.1, "Reach", p. 186).
const defaultReachFeet = 5

// ReachInFeet is a whole number of feet of reach, at least 1. Build it with
// NewReachInFeet. As an input field, the zero value reads as the default 5
// feet. Comparable.
type ReachInFeet struct{ feet int }

// NewReachInFeet returns a reach of feet. It returns an error matching
// srderr.ErrInvalidInput, naming feet, below 1.
func NewReachInFeet(feet int) (ReachInFeet, error) {
	if feet < 1 {
		return ReachInFeet{}, invalidInput("feet of a reach: below 1")
	}
	return ReachInFeet{feet: feet}, nil
}

// MustReachInFeet is NewReachInFeet for values known to be valid. It panics
// with the error that function returns, and only then.
func MustReachInFeet(feet int) ReachInFeet {
	r, err := NewReachInFeet(feet)
	if err != nil {
		panic(err)
	}
	return r
}

// Feet returns the reach in feet; the zero value returns 0.
func (r ReachInFeet) Feet() int { return r.feet }

// orDefault returns r, or the default 5 feet for the zero value.
func (r ReachInFeet) orDefault() ReachInFeet {
	if r.feet == 0 {
		return ReachInFeet{feet: defaultReachFeet}
	}
	return r
}

// weaponReach returns r extended by 5 feet for a weapon with the Reach
// property (SRD 5.2.1, "Reach", p. 90).
func weaponReach(w content.Weapon, r ReachInFeet) ReachInFeet {
	r = r.orDefault()
	if hasProperty(w, content.WeaponPropertyReach) {
		r.feet += 5
	}
	return r
}

// OpportunityAttackReach returns the reach with weapon w for Opportunity
// Attacks: the attacker's reach plus 5 feet for a Reach weapon, otherwise the
// attacker's reach (SRD 5.2.1, "Reach", p. 90; "Opportunity Attacks", p. 185).
func OpportunityAttackReach(w content.Weapon, attackerReach ReachInFeet) ReachInFeet {
	return weaponReach(w, attackerReach)
}

// hasProperty reports whether w has property p.
func hasProperty(w content.Weapon, p content.WeaponProperty) bool {
	for _, u := range w.Properties() {
		if u.Property() == p {
			return true
		}
	}
	return false
}
