package combat

// AttackCost is what an attack spends (SRD 5.2.1, "Attack [Action]", p. 177; "Bonus Actions" and
// "Reactions", p. 10; "Opportunity Attacks", p. 185). Build a variant with a keyed literal; the
// PerformAttack and PerformUnarmedStrike command constructors reject an invalid one. Variants:
// BeginsAttackAction, ContinuesAttackAction, BonusActionCost, ReactionCost, AttackAlreadyPaid, RecordsMultiattack.
type AttackCost interface{ isAttackCost() }

// BeginsAttackAction: the Attack action begins with this attack, spending the action. Allowance is
// how many attacks the action gives.
type BeginsAttackAction struct {
	Allowance AttackAllowance
	Means     AttackMeans
}

// ContinuesAttackAction: a further attack of the Attack action in progress; no tracker resource is
// spent again.
type ContinuesAttackAction struct{ Means AttackMeans }

// BonusActionCost: the attack is made with the Bonus Action a rule grants. The zero Grant means none is
// granted and the tracker refuses.
type BonusActionCost struct{ Grant BonusActionGrant }

// ReactionCost: the attack is made with the attacker's Reaction, such as an Opportunity Attack.
type ReactionCost struct{ Choice ReactionChoice }

// AttackAlreadyPaid: the host states that the resource was spent by an earlier call (an attack made as
// part of a spell cast with the Magic action). It is not checked against the tracker (FR-020).
type AttackAlreadyPaid struct{}

// RecordsMultiattack: the attack is one of the Multiattack of a monster in progress in the ledger: it records the
// attack of Part against the plan, spends the usage limit of the part through LimitedUse and asks the tracker
// for nothing (combat-action-resolution.md FR-165). Build it with a keyed literal; both fields are required.
type RecordsMultiattack struct {
	Part       LimitedUsePartRef
	LimitedUse LimitedUseState
}

func (BeginsAttackAction) isAttackCost()    {}
func (ContinuesAttackAction) isAttackCost() {}
func (BonusActionCost) isAttackCost()       {}
func (ReactionCost) isAttackCost()          {}
func (AttackAlreadyPaid) isAttackCost()     {}
func (RecordsMultiattack) isAttackCost()    {}

// ReplacementCost is what an attack that another effect replaces or forgoes spends, a sibling of
// AttackCost because a replaced attack rolls no attack roll (combat-action-resolution.md FR-122).
// Variants: BeginsAttackActionByReplacing, ContinuesAttackActionByReplacing.
type ReplacementCost interface{ isReplacementCost() }

// BeginsAttackActionByReplacing: the Attack action begins and its first attack is replaced.
type BeginsAttackActionByReplacing struct{ Allowance AttackAllowance }

// ContinuesAttackActionByReplacing: a further attack of the Attack action in progress is replaced.
type ContinuesAttackActionByReplacing struct{}

func (BeginsAttackActionByReplacing) isReplacementCost()    {}
func (ContinuesAttackActionByReplacing) isReplacementCost() {}

// KnockOutChoice is the attacker's choice to knock out a creature the attack reduces to 0 Hit Points
// (SRD 5.2.1, "Knocking Out a Creature", p. 17). The zero value is NoKnockOut.
type KnockOutChoice int

// The two choices.
const (
	// NoKnockOut: the attack does ordinary damage.
	NoKnockOut KnockOutChoice = iota
	// KnockOut: a melee attack that would reduce the target to 0 Hit Points leaves it at 1.
	KnockOut
)

// Valid reports whether c is one of the two choices.
func (c KnockOutChoice) Valid() bool { return c == NoKnockOut || c == KnockOut }

// String returns "no_knock_out", "knock_out" or "unknown".
func (c KnockOutChoice) String() string {
	switch c {
	case NoKnockOut:
		return "no_knock_out"
	case KnockOut:
		return "knock_out"
	}
	return "unknown"
}

// validAttackCost reports whether c is a variant with valid parts, for the attack of the form.
// It returns an error matching srderr.ErrInvalidInput that says why not.
func validAttackCost(c AttackCost, form AttackForm, attacker, target CombatantID, scene AttackScene) error {
	switch v := c.(type) {
	case BeginsAttackAction:
		if !v.Allowance.Valid() || !v.Means.Valid() {
			return invalidInput("an Attack action needs a valid allowance and means")
		}
		return meansFit(v.Means, form)
	case ContinuesAttackAction:
		if !v.Means.Valid() {
			return invalidInput("an Attack action needs a valid means")
		}
		return meansFit(v.Means, form)
	case BonusActionCost, AttackAlreadyPaid:
		return nil
	case RecordsMultiattack:
		if v.Part.IsZero() || v.LimitedUse.IsZero() {
			return invalidInput("a Multiattack attack needs a part and a limited-use state")
		}
		return nil
	case ReactionCost:
		return validReactionCost(v, form, attacker, target, scene)
	}
	return invalidInput("an attack needs a cost")
}

// meansFit checks FR-141 and FR-142: no Attack action with a spell, and the means matches the form:
// unarmedStrike for an Unarmed Strike only.
func meansFit(m AttackMeans, form AttackForm) error {
	if form.Source() == AttackSourceSpell {
		return invalidInput("an Attack action is not made with a spell")
	}
	if (m == AttackMeansUnarmedStrike) != (form == UnarmedStrikeAttack) {
		return invalidInput("the attack means does not match the attack form")
	}
	return nil
}

// validReactionCost checks FR-137 to FR-140 for an Opportunity Attack (SRD 5.2.1, "Opportunity
// Attacks", p. 185): a melee weapon attack or an Unarmed Strike against its provoker, by another
// creature that sees it.
func validReactionCost(v ReactionCost, form AttackForm, attacker, target CombatantID, scene AttackScene) error {
	if v.Choice == nil || !validReactionChoice(v.Choice) {
		return invalidInput("an attack by Reaction needs a valid choice")
	}
	oa, ok := v.Choice.(OpportunityAttackReaction)
	if !ok {
		return nil
	}
	switch {
	case form != MeleeWeaponAttack && form != UnarmedStrikeAttack:
		return invalidInput("an Opportunity Attack is a melee weapon attack or an Unarmed Strike")
	case oa.Provoker != target:
		return invalidInput("an Opportunity Attack is made against its provoker")
	case oa.Provoker == attacker:
		return invalidInput("a creature does not provoke itself")
	case (oa.Visibility == VisibilitySeen) != (scene.attackerSeesTarget == SightRelationCanSee):
		return invalidInput("the Opportunity Attack visibility does not match the scene")
	}
	return nil
}
