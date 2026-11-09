package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	pactWeaponSeparationFeet = 5
)

// MagicWeaponClaim says who else holds a magic weapon a Warlock wants to bond, a closed set. The zero value is read as
// MagicWeaponUnclaimed.
type MagicWeaponClaim int

const (
	// MagicWeaponUnclaimed: no other creature is attuned to or bonded with the weapon.
	MagicWeaponUnclaimed MagicWeaponClaim = iota
	// MagicWeaponAttunedByAnotherCreature: another creature is attuned to the weapon.
	MagicWeaponAttunedByAnotherCreature
	// MagicWeaponBondedByAnotherWarlock: another Warlock is bonded with the weapon.
	MagicWeaponBondedByAnotherWarlock
)

// PactWeaponTarget is the weapon a Pact of the Blade bond is asked for: a conjured weapon or a magic weapon the Warlock touches. The
// zero value is not a target. It is immutable and comparable with ==. Build it with ConjuredPactWeapon or MagicPactWeapon.
type PactWeaponTarget struct {
	item     content.ItemID
	conjured bool
	category content.WeaponCategory
	kind     content.WeaponType
	claim    MagicWeaponClaim
}

// ConjuredPactWeapon is a weapon conjured by the feature, with the category and type of its equipment entry. A category or type of 0
// (an entry that is not a weapon) makes FormPactWeaponBond refuse it.
func ConjuredPactWeapon(item content.ItemID, category content.WeaponCategory, kind content.WeaponType) PactWeaponTarget {
	return PactWeaponTarget{item: item, conjured: true, category: category, kind: kind}
}

// MagicPactWeapon is a magic weapon the Warlock touches, with who else holds it.
func MagicPactWeapon(item content.ItemID, claim MagicWeaponClaim) PactWeaponTarget {
	return PactWeaponTarget{item: item, claim: claim}
}

// FormPactWeaponBond bonds a pact weapon as a Bonus Action (class-warlock.md FR-081 to FR-084, FR-086). A bond already held ends with a
// PactWeaponBondEnded event before the new PactWeaponBonded. It refuses with WarlockInvocationNotKnown (Option Pact of the Blade),
// WarlockNotMeleeSimpleOrMartial (Item) for a conjured weapon that is not a Simple or Martial Melee weapon, WarlockAttunedByAnother and
// WarlockBondedByAnotherWarlock for a magic weapon another creature holds (attunement first).
func FormPactWeaponBond(id Identifier, s WarlockState, pact WarlockPactState, target PactWeaponTarget) WarlockPactTransition {
	switch {
	case !s.Knows(pactOfTheBlade):
		return pactNotKnown(pactOfTheBlade)
	case target.item == "":
		return pactRefused(WarlockRejection{Kind: WarlockNotMeleeSimpleOrMartial})
	case target.conjured && (!target.category.Valid() || target.kind != content.WeaponMelee):
		return pactRefused(WarlockRejection{Kind: WarlockNotMeleeSimpleOrMartial, Item: target.item})
	case !target.conjured && target.claim == MagicWeaponAttunedByAnotherCreature:
		return pactRefused(warlockRefuse(WarlockAttunedByAnother))
	case !target.conjured && target.claim == MagicWeaponBondedByAnotherWarlock:
		return pactRefused(warlockRefuse(WarlockBondedByAnotherWarlock))
	}
	var events []Event
	if pact.bond.Bonded() {
		events = append(events, PactWeaponBondEnded{id: id, item: pact.bond.item, conjured: pact.bond.conjured})
	}
	pact.bond = PactWeaponBond{item: target.item, conjured: target.conjured}
	return pactChanged(pact, append(events, PactWeaponBonded{id: id, item: target.item, conjured: target.conjured})...)
}

// WarlockLife says whether the Warlock is alive, a closed set. The zero value is read as WarlockAlive.
type WarlockLife int

const (
	// WarlockAlive: the Warlock lives.
	WarlockAlive WarlockLife = iota
	// WarlockDead: the Warlock died.
	WarlockDead
)

// BondAction says whether the Warlock used the Pact of the Blade Bonus Action again, a closed set. The zero value is read as BondActionNone.
type BondAction int

const (
	// BondActionNone: the feature was not used again.
	BondActionNone BondAction = iota
	// BondActionUsedAgain: the feature was used again.
	BondActionUsedAgain
)

// WeaponSeparation is how far the pact weapon is from the Warlock and for how long it has been. Construct it with a keyed
// literal.
type WeaponSeparation struct {
	DistanceFeet int
	Duration     time.Duration
}

// BondEndFacts are the facts that end a pact weapon bond, supplied by the host. The zero value ends nothing.
type BondEndFacts struct {
	Action     BondAction
	Separation WeaponSeparation
	Life       WarlockLife
}

// EndPactWeaponBond ends the bond when the feature is used again, the weapon has been more than 5 feet away for 1 minute or more, or the
// Warlock died (class-warlock.md FR-085, FR-086); exactly 5 feet or 59 seconds ends nothing. It never refuses: an unmet condition, or no
// bond, leaves the state unchanged with no events. A conjured weapon disappears (PactWeaponBondEnded.Disappeared) and a magic weapon remains.
func EndPactWeaponBond(id Identifier, pact WarlockPactState, facts BondEndFacts) WarlockPactChanged {
	far := facts.Separation.DistanceFeet > pactWeaponSeparationFeet && facts.Separation.Duration >= time.Minute
	if !pact.bond.Bonded() || (facts.Action != BondActionUsedAgain && !far && facts.Life != WarlockDead) {
		return pactChanged(pact)
	}
	ended := PactWeaponBondEnded{id: id, item: pact.bond.item, conjured: pact.bond.conjured}
	pact.bond = PactWeaponBond{}
	return pactChanged(pact, ended)
}

// PactWeaponOptions are the abilities and damage types a pact weapon attack offers.
type PactWeaponOptions struct {
	// Abilities are the weapon's own abilities and, while bonded, Charisma, in ascending order.
	Abilities []ability.Ability
	// DamageTypes are the weapon's normal type and, while bonded, Necrotic, Psychic and Radiant, without repeats.
	DamageTypes []content.DamageType
}

// PactWeaponAttackOptions returns the options for attack and damage rolls with the weapon (class-warlock.md FR-084, FR-087, FR-088): the
// weapon's own abilities with Charisma as an alternative, and its normal damage type or Necrotic, Psychic or Radiant, while the bond
// lasts; without a bond the weapon's own. It returns new slices.
func PactWeaponAttackOptions(bond PactWeaponBond, own []ability.Ability, normal content.DamageType) PactWeaponOptions {
	abilities := slices.Clone(own)
	types := []content.DamageType{normal}
	if bond.Bonded() {
		abilities = append(abilities, ability.Charisma)
		for _, t := range []content.DamageType{content.DamageNecrotic, content.DamagePsychic, content.DamageRadiant} {
			if !slices.Contains(types, t) {
				types = append(types, t)
			}
		}
	}
	slices.Sort(abilities)
	return PactWeaponOptions{Abilities: slices.Compact(abilities), DamageTypes: types}
}

// ChoosePactWeaponDamageType validates the damage type chosen for a pact weapon attack (class-warlock.md FR-088). It returns the type and
// the zero WarlockRejection, or the zero type and a WarlockDamageTypeNotOffered rejection (DamageType) for a type
// PactWeaponAttackOptions does not list.
func ChoosePactWeaponDamageType(bond PactWeaponBond, normal, chosen content.DamageType) (content.DamageType, WarlockRejection) {
	if !slices.Contains(PactWeaponAttackOptions(bond, nil, normal).DamageTypes, chosen) {
		return 0, WarlockRejection{Kind: WarlockDamageTypeNotOffered, DamageType: chosen}
	}
	return chosen, WarlockRejection{}
}
