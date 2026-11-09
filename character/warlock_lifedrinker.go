package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const lifedrinker content.FeatureOptionID = "lifedrinker"

// HitDieChoice says whether the Warlock expends a Hit Point Die with Lifedrinker, a closed set. The zero value is read as HitDieDecline.
type HitDieChoice int

const (
	// HitDieDecline: no die is expended.
	HitDieDecline HitDieChoice = iota
	// HitDieExpend: one die is expended to regain Hit Points.
	HitDieExpend
)

// ApplyLifedrinker adds 1d6 Necrotic, Psychic or Radiant damage to a hit with the pact weapon once per turn and, when choice is HitDieExpend,
// expends one Hit Point Die to regain Hit Points (SRD 5.2.1, "Lifedrinker", p. 73; class-warlock.md FR-077, FR-078, FR-149). rolled is the
// face of die the use case rolled; the extra damage dice come back as WarlockChanged.Damage, the Hit Points regained as
// WarlockChanged.HitPointsRegained and the dice spent, recorded with SpentHitDice.SpendOutsideRest so the die cannot be spent again at the
// next Short Rest, as WarlockChanged.SpentHitDice. The state is unchanged. It returns an error matching srderr.ErrInvalidInput, when a
// die is expended, for an invalid die or a roll outside 1 to its faces. It refuses with WarlockInvocationNotKnown, WarlockNotAHit,
// WarlockAlreadyUsedThisTurn, WarlockDamageTypeNotOffered (DamageType) and WarlockNoHitPointDie.
func ApplyLifedrinker(id Identifier, s WarlockState, attack WarlockAttackKind, turn WarlockTurnFacts, damageType content.DamageType, choice HitDieChoice,
	pool HitDicePool, spent SpentHitDice, die content.HitPointDie, constitution ability.Modifier, rolled int) (WarlockTransition, error) {
	if choice == HitDieExpend && (!die.Valid() || rolled < 1 || rolled > die.Faces()) {
		return nil, invalidInput("Lifedrinker with an invalid Hit Point Die or a roll outside its faces")
	}
	switch {
	case !s.Knows(lifedrinker):
		return warlockRefused(WarlockRejection{Kind: WarlockInvocationNotKnown, Option: lifedrinker}), nil
	case attack != WarlockPactWeaponHit:
		return warlockRefused(warlockRefuse(WarlockNotAHit)), nil
	case turn.used(OnceLifedrinker):
		return warlockRefused(warlockRefuse(WarlockAlreadyUsedThisTurn)), nil
	case !slices.Contains([]content.DamageType{content.DamageNecrotic, content.DamagePsychic, content.DamageRadiant}, damageType):
		return warlockRefused(WarlockRejection{Kind: WarlockDamageTypeNotOffered, DamageType: damageType}), nil
	}
	c := warlockChanged(s)
	c.damage, c.hasDamage = dice.MustExpression(1, dice.D6(), 0), true
	if choice == HitDieExpend {
		allowed, ok := spent.SpendOutsideRest(pool, die).(HitDieSpendAllowed)
		if !ok {
			return warlockRefused(warlockRefuse(WarlockNoHitPointDie)), nil
		}
		c.healed, c.hasHealed, c.spent, c.hasSpent = LifedrinkerHealing(rolled, constitution), true, allowed.Spent(), true
	}
	c.events = []Event{LifedrinkerUsed{id: id, damageType: damageType, healed: c.healed}}
	return c, nil
}
