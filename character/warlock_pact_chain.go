package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

const investmentOfTheChainMaster content.FeatureOptionID = "investment_of_the_chain_master"

// specialFamiliarForms are the seven special forms of Pact of the Chain (SRD 5.2.1, "Pact of the Chain", p. 74).
func specialFamiliarForms() []content.MonsterID {
	return []content.MonsterID{"imp", "pseudodragon", "quasit", "skeleton", "sphinx_of_wonder", "sprite", "venomous_snake"}
}

// SelectFamiliarForm sets the familiar's form (class-warlock.md FR-090). monster is the entry of the form loaded from the content source.
// A special form must be one of the seven; a normal form must be a Beast of Challenge Rating 0 ("Find Familiar", p. 130). It refuses with
// WarlockInvocationNotKnown (Option Pact of the Chain) and WarlockFamiliarFormNotOffered (Monster).
func SelectFamiliarForm(id Identifier, s WarlockState, pact WarlockPactState, form FamiliarForm, monster content.Monster) WarlockPactTransition {
	offered := slices.Contains(specialFamiliarForms(), form.id)
	if form.normal {
		offered = monster.Type() == content.CreatureTypeBeast && monster.ChallengeRating().IsZero()
	}
	switch {
	case !s.Knows(pactOfTheChain):
		return pactNotKnown(pactOfTheChain)
	case form.id == "" || !offered:
		return pactRefused(WarlockRejection{Kind: WarlockFamiliarFormNotOffered, Monster: form.id})
	}
	pact.familiar.form = form
	return pactChanged(pact, FamiliarFormChosen{id: id, form: form})
}

// ForgoAttackForFamiliar forgoes one of the Warlock's attacks of the Attack action so the familiar attacks with its Reaction
// (class-warlock.md FR-091). It consumes exactly one attack, returned by WarlockPactChanged.AttacksRemaining. It refuses with
// WarlockInvocationNotKnown (Option Pact of the Chain), WarlockNoFamiliar, WarlockNoAttackToForgo and WarlockReactionSpent.
func ForgoAttackForFamiliar(id Identifier, s WarlockState, pact WarlockPactState, turn WarlockTurnFacts) WarlockPactTransition {
	switch {
	case !s.Knows(pactOfTheChain):
		return pactNotKnown(pactOfTheChain)
	case pact.familiar.form.id == "":
		return pactRefused(warlockRefuse(WarlockNoFamiliar))
	case turn.AttacksRemaining < 1:
		return pactRefused(warlockRefuse(WarlockNoAttackToForgo))
	case turn.Reaction == ActionSpent:
		return pactRefused(warlockRefuse(WarlockReactionSpent))
	}
	c := pactChanged(pact, FamiliarAttackForgone{id: id})
	c.attacks = turn.AttacksRemaining - 1
	return c
}

// CommandFamiliarAttack commands the familiar to take the Attack action as a Bonus Action (class-warlock.md FR-093). It refuses with
// WarlockInvocationNotKnown (Option Investment of the Chain Master), WarlockNoFamiliar and WarlockNoBonusAction.
func CommandFamiliarAttack(id Identifier, s WarlockState, pact WarlockPactState, turn WarlockTurnFacts) WarlockPactTransition {
	switch {
	case !s.Knows(investmentOfTheChainMaster):
		return pactNotKnown(investmentOfTheChainMaster)
	case pact.familiar.form.id == "":
		return pactRefused(warlockRefuse(WarlockNoFamiliar))
	case turn.BonusAction == ActionSpent:
		return pactRefused(warlockRefuse(WarlockNoBonusAction))
	}
	return pactChanged(pact, FamiliarAttackCommanded{id: id})
}

// ChooseFamiliarSpeed gives the familiar a Fly or Swim Speed of 40 feet (class-warlock.md FR-092). It refuses with
// WarlockInvocationNotKnown (Option Investment of the Chain Master) and WarlockNoFamiliar. It returns an error matching srderr.ErrInvalidInput
// for a speed that is neither FamiliarFlies nor FamiliarSwims.
func ChooseFamiliarSpeed(id Identifier, s WarlockState, pact WarlockPactState, kind FamiliarSpeedKind) (WarlockPactTransition, error) {
	switch {
	case kind != FamiliarFlies && kind != FamiliarSwims:
		return nil, invalidInput("familiar Speed outside Fly and Swim")
	case !s.Knows(investmentOfTheChainMaster):
		return pactNotKnown(investmentOfTheChainMaster), nil
	case pact.familiar.form.id == "":
		return pactRefused(warlockRefuse(WarlockNoFamiliar)), nil
	}
	pact.familiar.speed = kind
	return pactChanged(pact, FamiliarSpeedChosen{id: id, kind: kind}), nil
}

// ChangeFamiliarDamageType changes Bludgeoning, Piercing or Slashing damage the familiar deals to Necrotic or Radiant (class-warlock.md
// FR-094). It returns the new type and the zero WarlockRejection, or the zero type and WarlockInvocationNotKnown (Option Investment of the
// Chain Master), WarlockDamageNotChangeable (dealt) or WarlockTargetTypeNotOffered (requested).
func ChangeFamiliarDamageType(s WarlockState, dealt, requested content.DamageType) (content.DamageType, WarlockRejection) {
	switch {
	case !s.Knows(investmentOfTheChainMaster):
		return 0, WarlockRejection{Kind: WarlockInvocationNotKnown, Option: investmentOfTheChainMaster}
	case !slices.Contains([]content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing}, dealt):
		return 0, WarlockRejection{Kind: WarlockDamageNotChangeable, DamageType: dealt}
	case requested != content.DamageNecrotic && requested != content.DamageRadiant:
		return 0, WarlockRejection{Kind: WarlockTargetTypeNotOffered, DamageType: requested}
	}
	return requested, WarlockRejection{}
}

// FamiliarSaveDc returns the save DC of a saving throw the familiar forces: the character's own spell save DC, unchanged (class-warlock.md
// FR-095). It returns the zero WarlockRejection or the zero DC and WarlockInvocationNotKnown (Option Investment of the Chain Master).
func FamiliarSaveDc(s WarlockState, dc SpellSaveDC) (SpellSaveDC, WarlockRejection) {
	if !s.Knows(investmentOfTheChainMaster) {
		return SpellSaveDC{}, WarlockRejection{Kind: WarlockInvocationNotKnown, Option: investmentOfTheChainMaster}
	}
	return dc, WarlockRejection{}
}

// GrantFamiliarResistance grants the familiar Resistance to the damage it took, with the Warlock's Reaction (class-warlock.md FR-096).
// The type comes back from WarlockPactChanged.Resistance. It refuses with WarlockInvocationNotKnown (Option Investment of the Chain
// Master), WarlockNoFamiliar and WarlockReactionSpent; an invalid damage type is WarlockDamageTypeNotOffered.
func GrantFamiliarResistance(id Identifier, s WarlockState, pact WarlockPactState, turn WarlockTurnFacts, taken content.DamageType) WarlockPactTransition {
	switch {
	case !s.Knows(investmentOfTheChainMaster):
		return pactNotKnown(investmentOfTheChainMaster)
	case pact.familiar.form.id == "":
		return pactRefused(warlockRefuse(WarlockNoFamiliar))
	case turn.Reaction == ActionSpent:
		return pactRefused(warlockRefuse(WarlockReactionSpent))
	case !taken.Valid():
		return pactRefused(WarlockRejection{Kind: WarlockDamageTypeNotOffered, DamageType: taken})
	}
	c := pactChanged(pact, FamiliarResistanceGranted{id: id, damageType: taken})
	c.resistance = taken
	return c
}
