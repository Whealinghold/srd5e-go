package combat

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// basicPoisonPotency is how long Basic Poison keeps its potency (SRD 5.2.1, "Poison, Basic", p. 99).
const basicPoisonPotency = time.Minute

// BasicPoisonObject is what a coating is on. The zero value is invalid.
type BasicPoisonObject int

// The two objects.
const (
	// BasicPoisonWeapon: one weapon.
	BasicPoisonWeapon BasicPoisonObject = iota + 1
	// BasicPoisonAmmunitionPiece: one piece of ammunition.
	BasicPoisonAmmunitionPiece
)

// String returns "weapon", "ammunition_piece" or "unknown".
func (o BasicPoisonObject) String() string {
	return enumName([]string{"weapon", "ammunition_piece"}, int(o))
}

// BasicPoisonState is the state of a coating. The zero value is invalid.
type BasicPoisonState int

// The three states.
const (
	// BasicPoisonPotent: the poison retains its potency.
	BasicPoisonPotent BasicPoisonState = iota + 1
	// BasicPoisonExpired: the minute passed.
	BasicPoisonExpired
	// BasicPoisonDelivered: its damage was dealt.
	BasicPoisonDelivered
)

// String returns "potent", "expired", "delivered" or "unknown".
func (s BasicPoisonState) String() string {
	return enumName([]string{"potent", "expired", "delivered"}, int(s))
}

// BasicPoisonCoating is Basic Poison on one weapon or one piece of ammunition (SRD 5.2.1,
// "Poison, Basic", p. 99): potent for 1 minute or until its damage is dealt. It holds no
// weapon identity (the host pairs coating and weapon), no save and no Poison type; each
// coating is independent of the others. Immutable and comparable. The zero value is not a
// coating: its methods report an *ItemUseError or, for ExtraDamage, BasicPoisonNotPotent.
type BasicPoisonCoating struct {
	owner   CombatantID
	object  BasicPoisonObject
	elapsed time.Duration
	state   BasicPoisonState
}

// IsOwnedBy reports whether id is the creature that applied the poison.
func (c BasicPoisonCoating) IsOwnedBy(id CombatantID) bool { return !id.IsZero() && c.owner == id }

// Object returns what the coating is on.
func (c BasicPoisonCoating) Object() BasicPoisonObject { return c.object }

// State returns the state of the coating.
func (c BasicPoisonCoating) State() BasicPoisonState { return c.state }

// Elapsed returns the time since the poison was applied.
func (c BasicPoisonCoating) Elapsed() time.Duration { return c.elapsed }

// BasicPoisonUpdate is a coating after a change and the events it produced. Not comparable.
type BasicPoisonUpdate struct {
	coating BasicPoisonCoating
	events  []ItemUseEvent
}

// Coating returns the new coating.
func (u BasicPoisonUpdate) Coating() BasicPoisonCoating { return u.coating }

// Events returns a copy of the events: one BasicPoisonSpent when the coating was spent.
func (u BasicPoisonUpdate) Events() []ItemUseEvent { return slices.Clone(u.events) }

func (c BasicPoisonCoating) valid() bool {
	return !c.owner.IsZero() && c.state >= BasicPoisonPotent && c.state <= BasicPoisonDelivered
}

func (c BasicPoisonCoating) spent(cause BasicPoisonState) BasicPoisonUpdate {
	c.state = cause
	return BasicPoisonUpdate{c, []ItemUseEvent{BasicPoisonSpent{itemUseEventBase{first: c.owner}, cause}}}
}

// Advance adds d to the elapsed time and moves a potent coating to BasicPoisonExpired, with
// one BasicPoisonSpent event, once the elapsed time is at least 1 minute (exactly 60 seconds
// is expired, Q-8). A spent coating is returned unchanged with no event.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with ItemUseFieldElapsed for
// a negative d and ItemUseFieldCoating for the zero coating. It never panics.
func (c BasicPoisonCoating) Advance(d time.Duration) (BasicPoisonUpdate, error) {
	switch {
	case !c.valid():
		return BasicPoisonUpdate{}, itemUseInvalid(ItemUseFieldCoating, 0)
	case d < 0:
		return BasicPoisonUpdate{}, itemUseInvalid(ItemUseFieldElapsed, int64(d))
	case c.state != BasicPoisonPotent:
		return BasicPoisonUpdate{coating: c}, nil
	}
	c.elapsed = satAddDuration(c.elapsed, d)
	if c.elapsed >= basicPoisonPotency {
		return c.spent(BasicPoisonExpired), nil
	}
	return BasicPoisonUpdate{coating: c}, nil
}

// BasicPoisonExtra is the extra damage a coating adds to a weapon attack. Variants:
// BasicPoisonExtraDamage, BasicPoisonNoExtraDamage.
type BasicPoisonExtra interface{ isBasicPoisonExtra() }

// BasicPoisonExtraDamage: add the component to the attack's DamageSpec; the die is rolled
// with the attack's damage and doubled on a Critical Hit (Q-5).
type BasicPoisonExtraDamage struct{ damage DiceDamage }

// Damage returns the extra 1d4 Poison damage.
func (e BasicPoisonExtraDamage) Damage() DiceDamage { return e.damage }

// BasicPoisonNoExtraReason is why a coating adds no damage. The zero value is no reason.
type BasicPoisonNoExtraReason int

// The reasons.
const (
	// BasicPoisonNotPiercingOrSlashing: the weapon's damage is neither Piercing nor Slashing.
	BasicPoisonNotPiercingOrSlashing BasicPoisonNoExtraReason = iota + 1
	// BasicPoisonNotPotent: the coating is spent (or is not a coating).
	BasicPoisonNotPotent
)

// String returns "not_piercing_or_slashing", "not_potent" or "unknown".
func (r BasicPoisonNoExtraReason) String() string {
	return enumName([]string{"not_piercing_or_slashing", "not_potent"}, int(r))
}

// BasicPoisonNoExtraDamage: the coating adds nothing.
type BasicPoisonNoExtraDamage struct{ reason BasicPoisonNoExtraReason }

// Reason returns why.
func (e BasicPoisonNoExtraDamage) Reason() BasicPoisonNoExtraReason { return e.reason }

func (BasicPoisonExtraDamage) isBasicPoisonExtra()   {}
func (BasicPoisonNoExtraDamage) isBasicPoisonExtra() {}

func piercingOrSlashing(t content.DamageType) bool {
	return t == content.DamagePiercing || t == content.DamageSlashing
}

// ExtraDamage returns the extra 1d4 Poison damage of a potent coating on a weapon whose
// damage type is Piercing or Slashing, BasicPoisonNoExtraDamage with
// BasicPoisonNotPotent for a spent coating, else with BasicPoisonNotPiercingOrSlashing. It
// does not spend the coating.
func (c BasicPoisonCoating) ExtraDamage(weaponDamage content.DamageType) BasicPoisonExtra {
	switch {
	case !c.valid() || c.state != BasicPoisonPotent:
		return BasicPoisonNoExtraDamage{BasicPoisonNotPotent}
	case !piercingOrSlashing(weaponDamage):
		return BasicPoisonNoExtraDamage{BasicPoisonNotPiercingOrSlashing}
	}
	return BasicPoisonExtraDamage{MustDiceDamage(1, dice.D4(), content.DamagePoison, 0, false)}
}

// AfterWeaponDamage moves a potent coating to BasicPoisonDelivered, with one BasicPoisonSpent
// event, when the weapon's damage type is Piercing or Slashing and damageTaken is 1 or more;
// otherwise (Bludgeoning, damage reduced to 0, or a spent coating) it returns the coating
// unchanged with no event. damageTaken is DamageTaken.Total, so Temporary Hit Points that
// absorb the hit count. The host calls it only after a hit: a miss leaves the coating potent.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with ItemUseFieldDamageTaken
// for a negative damageTaken and ItemUseFieldCoating for the zero coating. It never panics.
func (c BasicPoisonCoating) AfterWeaponDamage(weaponDamage content.DamageType, damageTaken int) (BasicPoisonUpdate, error) {
	switch {
	case !c.valid():
		return BasicPoisonUpdate{}, itemUseInvalid(ItemUseFieldCoating, 0)
	case damageTaken < 0:
		return BasicPoisonUpdate{}, itemUseInvalid(ItemUseFieldDamageTaken, int64(damageTaken))
	case c.state == BasicPoisonPotent && piercingOrSlashing(weaponDamage) && damageTaken >= 1:
		return c.spent(BasicPoisonDelivered), nil
	}
	return BasicPoisonUpdate{coating: c}, nil
}

// BasicPoisonApplication is the result of applying Basic Poison: the new coatings, the cost
// and the item used up. Not comparable.
type BasicPoisonApplication struct {
	coatings []BasicPoisonCoating
	events   []ItemUseEvent
}

// Coatings returns a copy of the new coatings, independent of one another.
func (a BasicPoisonApplication) Coatings() []BasicPoisonCoating { return slices.Clone(a.coatings) }

// Cost returns ItemActionCostBonusAction.
func (BasicPoisonApplication) Cost() ItemActionCost { return ItemUseRules{}.CostOf(ItemUseBasicPoison) }

// Consumption returns the one vial used up.
func (BasicPoisonApplication) Consumption() ItemConsumption { return consumedOne(ItemUseBasicPoison) }

// Events returns a copy of the events: one BasicPoisonApplied.
func (a BasicPoisonApplication) Events() []ItemUseEvent { return slices.Clone(a.events) }

// BasicPoisonRules applies Basic Poison (SRD 5.2.1, "Poison, Basic", p. 99). The zero value is
// ready to use. Pure; safe for concurrent use.
type BasicPoisonRules struct{}

// MaxAmmunitionPieces is the most pieces of ammunition one application coats.
const MaxAmmunitionPieces = 3

// ApplyToWeapon returns one potent coating on a weapon of owner, elapsed zero. It returns an
// *ItemUseError (matching srderr.ErrInvalidInput) with ItemUseFieldOwner for a zero owner. It
// never panics.
func (r BasicPoisonRules) ApplyToWeapon(owner CombatantID) (BasicPoisonApplication, error) {
	return r.apply(owner, BasicPoisonWeapon, 1)
}

// ApplyToAmmunition returns pieces independent potent coatings on pieces of ammunition of
// owner, for pieces 1 to 3. It returns an *ItemUseError (matching srderr.ErrInvalidInput) with
// ItemUseFieldAmmunitionPieces and the count for any other count and ItemUseFieldOwner for a
// zero owner. It never panics.
func (r BasicPoisonRules) ApplyToAmmunition(owner CombatantID, pieces int) (BasicPoisonApplication, error) {
	if pieces < 1 || pieces > MaxAmmunitionPieces {
		return BasicPoisonApplication{}, itemUseInvalid(ItemUseFieldAmmunitionPieces, int64(pieces))
	}
	return r.apply(owner, BasicPoisonAmmunitionPiece, pieces)
}

func (BasicPoisonRules) apply(owner CombatantID, object BasicPoisonObject, n int) (BasicPoisonApplication, error) {
	if owner.IsZero() {
		return BasicPoisonApplication{}, itemUseInvalid(ItemUseFieldOwner, 0)
	}
	coatings := make([]BasicPoisonCoating, n)
	for i := range coatings {
		coatings[i] = BasicPoisonCoating{owner: owner, object: object, state: BasicPoisonPotent}
	}
	return BasicPoisonApplication{coatings, []ItemUseEvent{BasicPoisonApplied{itemUseEventBase{first: owner}}}}, nil
}
