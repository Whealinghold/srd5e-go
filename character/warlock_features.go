package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// WarlockTransition is the closed outcome of a Warlock domain function that changes the WarlockState. Variants: WarlockChanged,
// WarlockRefused.
type WarlockTransition interface{ isWarlockTransition() }

// WarlockSlotChange is the change to the Pact Magic slots a Warlock operation asks the host to make with the spellcasting service
// (class-warlock.md FR-163). The operations read the Pact Magic slots and never change them; a change names the Pact Magic pool only.
// Variants: WarlockSlotChangeExpendPact, WarlockSlotChangeRegainPact.
type WarlockSlotChange interface{ isWarlockSlotChange() }

// WarlockSlotChangeExpendPact asks the host to expend one Pact Magic slot of the Level.
type WarlockSlotChangeExpendPact struct{ Level int }

// WarlockSlotChangeRegainPact asks the host to regain Count expended Pact Magic slots.
type WarlockSlotChangeRegainPact struct{ Count int }

func (WarlockSlotChangeExpendPact) isWarlockSlotChange() {}
func (WarlockSlotChangeRegainPact) isWarlockSlotChange() {}

// WarlockChanged is the outcome of an operation the rules allowed: the new state, the events it produced and, for the operations that
// have them, a spell to cast without a slot, a Pact Magic slot change, a modifier to add to a roll or extra damage dice.
type WarlockChanged struct {
	state     WarlockState
	events    []Event
	grant     WarlockSpellGrant
	slot      WarlockSlotChange
	bonus     modifier.Modifier
	hasBonus  bool
	damage    dice.Expression
	hasDamage bool
	healed    int
	hasHealed bool
	spent     SpentHitDice
	hasSpent  bool
}

// State returns the new WarlockState.
func (r WarlockChanged) State() WarlockState { return r.state }

// Events returns the events, as a new slice.
func (r WarlockChanged) Events() []Event { return slices.Clone(r.events) }

// SpellGrant returns the spell to cast without a spell slot and true for Contact Patron, Mystic Arcanum and a spell-granting invocation.
func (r WarlockChanged) SpellGrant() (WarlockSpellGrant, bool) { return r.grant, r.grant.Spell != "" }

// SlotChange returns the Pact Magic slot change for the host to apply and true for Eldritch Smite, Magical Cunning with slots to
// regain and the restoration of Hurl Through Hell.
func (r WarlockChanged) SlotChange() (WarlockSlotChange, bool) { return r.slot, r.slot != nil }

// Bonus returns the d10 of Dark One's Own Luck as a Modifier for the host to add to the roll, and true for that feature only.
func (r WarlockChanged) Bonus() (modifier.Modifier, bool) { return r.bonus, r.hasBonus }

// Damage returns the extra damage dice of Eldritch Smite (Force) or Lifedrinker (1d6) and true for those features only.
func (r WarlockChanged) Damage() (dice.Expression, bool) { return r.damage, r.hasDamage }

// HitPointsRegained returns the Hit Points Lifedrinker regains and true when a Hit Point Die was expended.
func (r WarlockChanged) HitPointsRegained() (int, bool) { return r.healed, r.hasHealed }

// SpentHitDice returns the spent Hit Point Dice after Lifedrinker expended one, and true for that case only.
func (r WarlockChanged) SpentHitDice() (SpentHitDice, bool) { return r.spent, r.hasSpent }

// WarlockRefused is the outcome of an operation a rule refused: the state is unchanged and there are no events.
type WarlockRefused struct{ rejection WarlockRejection }

// Rejection returns why the operation was refused.
func (r WarlockRefused) Rejection() WarlockRejection { return r.rejection }

func (WarlockChanged) isWarlockTransition() {}
func (WarlockRefused) isWarlockTransition() {}

func warlockChanged(s WarlockState, events ...Event) WarlockChanged {
	return WarlockChanged{state: s, events: events}
}
func warlockRefused(r WarlockRejection) WarlockTransition { return WarlockRefused{rejection: r} }
func warlockGranted(s WarlockState, g WarlockSpellGrant, events ...Event) WarlockTransition {
	c := warlockChanged(s, events...)
	c.grant = g
	return c
}

// WarlockAttackKind says what an attack roll was, a closed set. The zero value is read as WarlockMissed.
type WarlockAttackKind int

const (
	// WarlockMissed: the attack missed.
	WarlockMissed WarlockAttackKind = iota
	// WarlockPactWeaponHit: the attack hit with the pact weapon.
	WarlockPactWeaponHit
	// WarlockOtherHit: the attack hit with something other than the pact weapon.
	WarlockOtherHit
)

// OncePerTurnFeature is a Warlock feature usable once per turn, a closed set.
type OncePerTurnFeature int

const (
	// OnceEldritchSmite is Eldritch Smite.
	OnceEldritchSmite OncePerTurnFeature = iota + 1
	// OnceHurlThroughHell is Hurl Through Hell.
	OnceHurlThroughHell
	// OnceLifedrinker is Lifedrinker.
	OnceLifedrinker
)

// WarlockActionAvailability says whether an action, Bonus Action, Reaction or Magic action is still available this turn, a closed set. The
// zero value is read as ActionAvailable, like the zero WarlockTurnFacts.
type WarlockActionAvailability int

const (
	// ActionAvailable: the action can still be taken.
	ActionAvailable WarlockActionAvailability = iota
	// ActionSpent: the action was already taken.
	ActionSpent
)

// WarlockTurnFacts are the facts about the current turn a Warlock operation needs, supplied by the host. The zero value says nothing
// was used this turn. Actions are available and no attack is left. Construct it with a keyed literal.
type WarlockTurnFacts struct {
	// UsedThisTurn lists the once-per-turn features already used this turn.
	UsedThisTurn []OncePerTurnFeature
	// BonusAction, Reaction and MagicAction say whether each is still available.
	BonusAction, Reaction, MagicAction WarlockActionAvailability
	// AttacksRemaining is the number of attacks of the Attack action not yet made.
	AttacksRemaining int
}

func (f WarlockTurnFacts) used(o OncePerTurnFeature) bool { return slices.Contains(f.UsedThisTurn, o) }

// ---- Pact Magic and the Level 2 to 20 features ---------------------------

// CompleteMagicalCunning completes the rite of Magical Cunning (class-warlock.md FR-105 to FR-109, FR-163, table 4.7.1). st is the
// spellcasting state read, never changed; the regained count is returned as a WarlockSlotChangeRegainPact for the host to apply, and
// none when no slot was expended (the use is still used up, FR-108). It refuses with WarlockFeatureNotGained below level 2,
// WarlockMagicalCunningAlreadyUsed after a use, and WarlockRiteTooShort (Required and Given in seconds) for a rite under 1 minute.
func CompleteMagicalCunning(id Identifier, s WarlockState, p WarlockProfile, rite time.Duration, st SpellcastingState) WarlockTransition {
	switch {
	case p.level.n < warlockMagicalCunningLevel:
		return warlockRefused(warlockNotGained("magical_cunning"))
	case s.cunningUsed:
		return warlockRefused(warlockRefuse(WarlockMagicalCunningAlreadyUsed))
	case rite < time.Minute:
		return warlockRefused(WarlockRejection{Kind: WarlockRiteTooShort, Required: int(time.Minute / time.Second), Given: int(rite / time.Second)})
	}
	_, count, remaining := pactSlotsOf(st)
	regained := MagicalCunningRegain(p.level, count, count-remaining)
	s.cunningUsed = true
	c := warlockChanged(s, MagicalCunningUsed{id: id, regains: regained})
	if regained > 0 {
		c.slot = WarlockSlotChangeRegainPact{Count: regained}
	}
	return c
}

// UseContactPatron casts Contact Other Plane without a spell slot (class-warlock.md FR-115, FR-116, FR-165). It refuses with
// WarlockFeatureNotGained below level 9 and WarlockContactPatronAlreadyUsed after a use until a Long Rest.
func UseContactPatron(id Identifier, s WarlockState, p WarlockProfile) WarlockTransition {
	switch {
	case p.level.n < warlockContactPatronLevel:
		return warlockRefused(warlockNotGained("contact_patron"))
	case s.patronUsed:
		return warlockRefused(warlockRefuse(WarlockContactPatronAlreadyUsed))
	}
	s.patronUsed = true
	return warlockGranted(s, WarlockSpellGrant{Spell: "contact_other_plane", Mode: CastModeFeatureGranted}, ContactPatronUsed{id: id})
}

// ChooseInvocations makes sel the Eldritch Invocations known (class-warlock.md FR-043 to FR-053). It never refuses: SelectInvocations
// has already validated the selection.
func ChooseInvocations(id Identifier, s WarlockState, sel InvocationSelection) WarlockTransition {
	s.invocations = [28]uint8{}
	for _, pick := range sel.picks {
		s.invocations[slices.Index(warlockInvocationIDs(), pick.Option)]++
	}
	return warlockChanged(s, InvocationsChosen{id: id, picks: sel.Picks()})
}

// ChooseMysticArcanum chooses or replaces an arcanum spell (class-warlock.md FR-117, FR-118, FR-120). replacementsThisLevel is how many
// arcanum spells the Warlock has already replaced on gaining this level. The spell is of exactly the level the feature grants (6 to
// 5 plus the count at the level) and a Warlock spell. It refuses with WarlockNoArcanumAtLevel below level 11,
// WarlockArcanumWrongSpellLevel (ExpectedLevel is the highest level the character has a slot for), WarlockArcanumNotOnWarlockList and
// WarlockArcanumTooManyReplacements (Given) for a second replacement in one level.
func ChooseMysticArcanum(id Identifier, s WarlockState, p WarlockProfile, spell content.Spell, replacementsThisLevel int) WarlockTransition {
	count, level := p.level.ArcanumCount(), int(spell.Level())
	i := level - warlockArcanumFirstSpellLvl
	switch {
	case count == 0:
		return warlockRefused(warlockRefuse(WarlockNoArcanumAtLevel))
	case i < 0 || i >= count:
		return warlockRefused(WarlockRejection{Kind: WarlockArcanumWrongSpellLevel, Spell: spell.ID(), ExpectedLevel: warlockArcanumFirstSpellLvl + count - 1, ActualLevel: level})
	case !slices.Contains(spell.Classes(), WarlockClassID):
		return warlockRefused(WarlockRejection{Kind: WarlockArcanumNotOnWarlockList, Spell: spell.ID()})
	}
	replaced := s.arcanum[i] != ""
	if replaced && replacementsThisLevel >= 1 {
		return warlockRefused(WarlockRejection{Kind: WarlockArcanumTooManyReplacements, Given: replacementsThisLevel + 1})
	}
	s.arcanum[i] = spell.ID()
	return warlockChanged(s, MysticArcanumChosen{id: id, spellLevel: level, spell: spell.ID(), replaced: replaced})
}

// CastMysticArcanum casts an arcanum spell once without a spell slot (class-warlock.md FR-119, FR-165). It refuses with
// WarlockNotAnArcanumSpell for a spell that is not one of the arcanum spells and WarlockArcanumAlreadyCast until a Long Rest.
func CastMysticArcanum(id Identifier, s WarlockState, spell content.SpellID) WarlockTransition {
	i := slices.Index(s.arcanum[:], spell)
	switch {
	case spell == "" || i < 0:
		return warlockRefused(WarlockRejection{Kind: WarlockNotAnArcanumSpell, Spell: spell})
	case s.arcanumCast[i]:
		return warlockRefused(WarlockRejection{Kind: WarlockArcanumAlreadyCast, Spell: spell})
	}
	s.arcanumCast[i] = true
	return warlockGranted(s, WarlockSpellGrant{Spell: spell, Mode: CastModeFeatureGranted},
		MysticArcanumCast{id: id, spellLevel: warlockArcanumFirstSpellLvl + i, spell: spell})
}

// ---- Eldritch Smite ------------------------------------------------------

// SmiteProne is the Warlock's choice to give the target of Eldritch Smite the Prone condition, a closed set. The zero value is read as
// SmiteDeclineProne.
type SmiteProne int

const (
	// SmiteDeclineProne leaves the target as it is.
	SmiteDeclineProne SmiteProne = iota
	// SmiteApplyProne gives a Huge or smaller target the Prone condition.
	SmiteApplyProne
)

// UseEldritchSmite uses Eldritch Smite on a hit with the pact weapon (class-warlock.md FR-073 to FR-076, FR-154, FR-163, table 4.6.4).
// st is the spellcasting state read for its Pact Magic slots, never changed; the slot to expend is returned as a WarlockSlotChangeExpendPact at the Slot Level
// and the extra Force damage as Damage. It refuses with WarlockInvocationNotKnown, WarlockNotAHit for a miss or a hit with another
// weapon, WarlockAlreadyUsedThisTurn, WarlockNoPactMagicSlot and WarlockProneNotAllowedForSize (Size) for Prone against a
// Gargantuan target. Declining Prone still expends the slot and deals the damage.
func UseEldritchSmite(id Identifier, s WarlockState, attack WarlockAttackKind, turn WarlockTurnFacts, st SpellcastingState, size content.CreatureSize, prone SmiteProne) WarlockTransition {
	level, _, remaining := pactSlotsOf(st)
	switch {
	case !s.Knows("eldritch_smite"):
		return warlockRefused(WarlockRejection{Kind: WarlockInvocationNotKnown, Option: "eldritch_smite"})
	case attack != WarlockPactWeaponHit:
		return warlockRefused(warlockRefuse(WarlockNotAHit))
	case turn.used(OnceEldritchSmite):
		return warlockRefused(warlockRefuse(WarlockAlreadyUsedThisTurn))
	case remaining < 1:
		return warlockRefused(warlockRefuse(WarlockNoPactMagicSlot))
	case prone == SmiteApplyProne && size == content.SizeGargantuan:
		return warlockRefused(WarlockRejection{Kind: WarlockProneNotAllowedForSize, Size: size})
	}
	damage, ok := EldritchSmiteDamage(level)
	if !ok { // unreachable: a Pact Magic slot level is 1 to 5
		return warlockRefused(warlockRefuse(WarlockNoPactMagicSlot))
	}
	c := warlockChanged(s, EldritchSmiteUsed{id: id, slotLevel: level, prone: prone == SmiteApplyProne})
	c.slot, c.damage, c.hasDamage = WarlockSlotChangeExpendPact{Level: level}, damage, true
	return c
}

// ---- Fiend Patron --------------------------------------------------------

// TargetRelation says whether a creature reduced to 0 Hit Points was an enemy, a closed set. The zero value is read as TargetNotEnemy.
type TargetRelation int

const (
	// TargetNotEnemy: the creature is not an enemy of the Warlock.
	TargetNotEnemy TargetRelation = iota
	// TargetEnemy: the creature is an enemy of the Warlock.
	TargetEnemy
)

// DarkOnesBlessingSource says who reduced the enemy to 0 Hit Points, a closed set. The zero value is read as BlessingByWarlock.
type DarkOnesBlessingSource int

const (
	// BlessingByWarlock: the Warlock reduced the enemy to 0 Hit Points.
	BlessingByWarlock DarkOnesBlessingSource = iota
	// BlessingByAnother: someone else did, so the enemy must be within 10 feet of the Warlock.
	BlessingByAnother
)

const darkOnesBlessingReachFeet = 10

// ResolveDarkOnesBlessing gives the Temporary Hit Points of Dark One's Blessing (class-warlock.md FR-123 to FR-125, table 4.8.1). The
// state is returned unchanged; the points are in the DarkOnesBlessingGained event. It refuses with WarlockNotFiendPatron,
// WarlockFeatureNotGained below level 3, WarlockNotAnEnemy and WarlockTargetTooFar (Distance) for an enemy dropped by someone else
// more than 10 feet away (exactly 10 feet applies).
func ResolveDarkOnesBlessing(id Identifier, s WarlockState, p WarlockProfile, charisma ability.Modifier, relation TargetRelation, source DarkOnesBlessingSource, distanceFeet int) WarlockTransition {
	switch {
	case p.subclass != FiendPatronSubclassID:
		return warlockRefused(warlockRefuse(WarlockNotFiendPatron))
	case p.level.n < fiendBlessingLevel:
		return warlockRefused(warlockNotGained("dark_ones_blessing"))
	case relation != TargetEnemy:
		return warlockRefused(warlockRefuse(WarlockNotAnEnemy))
	case source == BlessingByAnother && distanceFeet > darkOnesBlessingReachFeet:
		return warlockRefused(WarlockRejection{Kind: WarlockTargetTooFar, Distance: distanceFeet})
	}
	return warlockChanged(s, DarkOnesBlessingGained{id: id, thp: DarkOnesBlessingTemporaryHitPoints(p.level, charisma)})
}

// ChooseFiendishResilience chooses the damage type of Fiendish Resilience on finishing a rest (class-warlock.md FR-131 to FR-133).
// rest is the rest just finished, the zero RestKind for none. It returns an error matching srderr.ErrInvalidInput for an invalid
// damage type. It refuses with WarlockNotFiendPatron, WarlockFeatureNotGained below level 10, WarlockNotAfterRest and
// WarlockForceNotAllowed. The Resistance lasts until a different type is chosen.
func ChooseFiendishResilience(id Identifier, s WarlockState, p WarlockProfile, rest RestKind, t content.DamageType) (WarlockTransition, error) {
	switch {
	case !t.Valid():
		return nil, invalidInput("Fiendish Resilience with an invalid damage type")
	case p.subclass != FiendPatronSubclassID:
		return warlockRefused(warlockRefuse(WarlockNotFiendPatron)), nil
	case p.level.n < fiendResilienceLevel:
		return warlockRefused(warlockNotGained("fiendish_resilience")), nil
	case !rest.Valid():
		return warlockRefused(warlockRefuse(WarlockNotAfterRest)), nil
	case t == content.DamageForce:
		return warlockRefused(warlockRefuse(WarlockForceNotAllowed)), nil
	}
	s.resilience = t
	return warlockChanged(s, FiendishResilienceChosen{id: id, damageType: t}), nil
}

// LuckOnRoll says whether Dark One's Own Luck was already used on the roll, a closed set. The zero value is read as LuckNotUsedOnRoll.
type LuckOnRoll int

const (
	// LuckNotUsedOnRoll: the feature was not used on this roll.
	LuckNotUsedOnRoll LuckOnRoll = iota
	// LuckUsedOnRoll: the feature was already used on this roll.
	LuckUsedOnRoll
)

// AddDarkOnesOwnLuck adds the d10 rolled to an ability check or a saving throw (class-warlock.md FR-127 to FR-130, FR-145, table
// 4.8.3). rolled is the 1d10 the use case rolled; the die comes back as a Modifier through WarlockChanged.Bonus. The timing, after
// seeing the roll, is the host's. It returns an error matching srderr.ErrInvalidInput for a roll outside 1 to 10 or an invalid kind.
// It refuses with WarlockNotFiendPatron, WarlockFeatureNotGained below level 6, WarlockRollKindNotAllowed for an attack roll,
// WarlockAlreadyUsedOnThisRoll and WarlockNoUsesLeft (Required is the maximum uses, Available what remains).
func AddDarkOnesOwnLuck(id Identifier, s WarlockState, p WarlockProfile, charisma ability.Modifier, kind d20.Kind, onRoll LuckOnRoll, rolled int) (WarlockTransition, error) {
	maximum := DarkOnesOwnLuckUses(charisma)
	switch {
	case rolled < 1 || rolled > 10 || !kind.Valid():
		return nil, invalidInput("Dark One's Own Luck with a roll outside 1 to 10 or an invalid roll kind")
	case p.subclass != FiendPatronSubclassID:
		return warlockRefused(warlockRefuse(WarlockNotFiendPatron)), nil
	case p.level.n < fiendOwnLuckLevel:
		return warlockRefused(warlockNotGained("dark_ones_own_luck")), nil
	case kind == d20.AttackRoll:
		return warlockRefused(warlockRefuse(WarlockRollKindNotAllowed)), nil
	case onRoll == LuckUsedOnRoll:
		return warlockRefused(warlockRefuse(WarlockAlreadyUsedOnThisRoll)), nil
	case s.luckSpent >= maximum:
		return warlockRefused(WarlockRejection{Kind: WarlockNoUsesLeft, Required: maximum, Available: max(maximum-s.luckSpent, 0)}), nil
	}
	s.luckSpent++
	c := warlockChanged(s, DarkOnesOwnLuckUsed{id: id, kind: kind, d10: rolled})
	c.bonus, c.hasBonus = modifier.New(rolled), true
	return c, nil
}

// HurlThroughHellEffect is what a failed save against Hurl Through Hell does to the target: 8d10 Psychic damage unless it is a Fiend,
// and the Incapacitated condition until the end of the Warlock's next turn (SRD 5.2.1, "Level 14: Hurl Through Hell", p. 76;
// class-warlock.md FR-136). The save, the damage and the condition are the host's to apply. It is a value of exported fields.
type HurlThroughHellEffect struct {
	// Damage is 8d10; DamageType is Psychic. Both apply only when Deals is true.
	Damage     dice.Expression
	DamageType content.DamageType
	// Deals is false for a Fiend target.
	Deals bool
	// Condition is Incapacitated, whatever the creature type.
	Condition content.ConditionKind
}

// HurlThroughHellFor returns the effect of Hurl Through Hell on a target of the creature type.
func HurlThroughHellFor(target content.CreatureType) HurlThroughHellEffect {
	return HurlThroughHellEffect{Damage: dice.MustExpression(8, dice.D10(), 0), DamageType: content.DamagePsychic,
		Deals: target != content.CreatureTypeFiend, Condition: content.ConditionIncapacitated}
}

// UseHurlThroughHell uses Hurl Through Hell on a hit with an attack roll (class-warlock.md FR-134 to FR-137). It refuses with
// WarlockNotFiendPatron, WarlockFeatureNotGained below level 14, WarlockNotAHit for a miss, WarlockAlreadyUsedThisTurn and
// WarlockHurlThroughHellAlreadySpent. The spell save DC and the effect (HurlThroughHellFor) are the host's to use.
func UseHurlThroughHell(id Identifier, s WarlockState, p WarlockProfile, attack WarlockAttackKind, turn WarlockTurnFacts) WarlockTransition {
	switch {
	case p.subclass != FiendPatronSubclassID:
		return warlockRefused(warlockRefuse(WarlockNotFiendPatron))
	case p.level.n < fiendHurlLevel:
		return warlockRefused(warlockNotGained("hurl_through_hell"))
	case attack == WarlockMissed:
		return warlockRefused(warlockRefuse(WarlockNotAHit))
	case turn.used(OnceHurlThroughHell):
		return warlockRefused(warlockRefuse(WarlockAlreadyUsedThisTurn))
	case s.hurlSpent:
		return warlockRefused(warlockRefuse(WarlockHurlThroughHellAlreadySpent))
	}
	s.hurlSpent = true
	return warlockChanged(s, HurlThroughHellUsed{id: id})
}

// RestoreHurlThroughHell restores the use of Hurl Through Hell by expending a Pact Magic slot (class-warlock.md FR-137, FR-138,
// FR-163). st is read for its Pact Magic slots, never changed; the slot is returned as a WarlockSlotChangeExpendPact at the Slot Level. It refuses with
// WarlockNotFiendPatron, WarlockFeatureNotGained below level 14, WarlockHurlThroughHellUseNotSpent and WarlockNoPactMagicSlot, and
// then returns no slot change.
func RestoreHurlThroughHell(id Identifier, s WarlockState, p WarlockProfile, st SpellcastingState) WarlockTransition {
	level, _, remaining := pactSlotsOf(st)
	switch {
	case p.subclass != FiendPatronSubclassID:
		return warlockRefused(warlockRefuse(WarlockNotFiendPatron))
	case p.level.n < fiendHurlLevel:
		return warlockRefused(warlockNotGained("hurl_through_hell"))
	case !s.hurlSpent:
		return warlockRefused(warlockRefuse(WarlockHurlThroughHellUseNotSpent))
	case remaining < 1:
		return warlockRefused(warlockRefuse(WarlockNoPactMagicSlot))
	}
	s.hurlSpent = false
	c := warlockChanged(s, HurlThroughHellRestored{id: id, slotLevel: level})
	c.slot = WarlockSlotChangeExpendPact{Level: level}
	return c
}

// ---- Rests and spells ----------------------------------------------------

// WarlockRestorableFeature is a Warlock feature a Long Rest restores, a closed set in this order. The zero value is invalid.
type WarlockRestorableFeature int

// The restorable features.
const (
	// RestoresMagicalCunning is Magical Cunning.
	RestoresMagicalCunning WarlockRestorableFeature = iota + 1
	// RestoresContactPatron is the slot-free cast of Contact Other Plane.
	RestoresContactPatron
	// RestoresMysticArcanum is the casts of the arcanum spells.
	RestoresMysticArcanum
	// RestoresGiftOfTheDepths is the slot-free cast of Water Breathing.
	RestoresGiftOfTheDepths
	// RestoresGiftOfTheProtectors is the page's benefit.
	RestoresGiftOfTheProtectors
	// RestoresDarkOnesOwnLuck is the uses of Dark One's Own Luck.
	RestoresDarkOnesOwnLuck
	// RestoresHurlThroughHell is the use of Hurl Through Hell.
	RestoresHurlThroughHell
)

// String returns the snake_case name ("magical_cunning", ...) or "unknown".
func (f WarlockRestorableFeature) String() string {
	names := []string{"magical_cunning", "contact_patron", "mystic_arcanum", "gift_of_the_depths", "gift_of_the_protectors", "dark_ones_own_luck", "hurl_through_hell"}
	if f < 1 || int(f) > len(names) {
		return "unknown"
	}
	return names[f-1]
}

// ApplyWarlockRest applies a finished rest (class-warlock.md FR-139, FR-140); the host calls it beside RestoreSpellSlots with the same
// kind. A LongRest restores Magical Cunning, Contact Patron, every Mystic Arcanum cast, the Gift of the Depths cast, the Gift of the
// Protectors benefit, Dark One's Own Luck and Hurl Through Hell, and emits one WarlockFeaturesRestored event listing only what was
// spent, none when nothing was. A ShortRest restores none of them (the Pact Magic slots are the spellcasting service's). It returns
// an error matching srderr.ErrInvalidInput for a kind that is neither ShortRest nor LongRest. It never refuses.
func ApplyWarlockRest(id Identifier, s WarlockState, kind RestKind) (WarlockTransition, error) {
	switch kind {
	case ShortRest:
		return warlockChanged(s), nil
	case LongRest:
		var restored []WarlockRestorableFeature
		for _, r := range []struct {
			f     WarlockRestorableFeature
			spent bool
		}{{RestoresMagicalCunning, s.cunningUsed}, {RestoresContactPatron, s.patronUsed}, {RestoresMysticArcanum, slices.Contains(s.arcanumCast[:], true)},
			{RestoresGiftOfTheDepths, s.depthsCast}, {RestoresGiftOfTheProtectors, s.protectorsSpent}, {RestoresDarkOnesOwnLuck, s.luckSpent > 0},
			{RestoresHurlThroughHell, s.hurlSpent}} {
			if r.spent {
				restored = append(restored, r.f)
			}
		}
		s.cunningUsed, s.patronUsed, s.depthsCast, s.protectorsSpent, s.hurlSpent, s.luckSpent, s.arcanumCast = false, false, false, false, false, 0, [4]bool{}
		if len(restored) == 0 {
			return warlockChanged(s), nil
		}
		return warlockChanged(s, WarlockFeaturesRestored{id: id, features: restored}), nil
	}
	return nil, invalidInput("ApplyWarlockRest with a rest that is neither short nor long")
}

// WarlockAlwaysPrepared returns the spells the Warlock always has prepared at the profile's level: the AlwaysPreparedSpells facts of
// the class definition (Contact Patron) and of the chosen subclass found in it (Fiend Spells) whose level is at most the Warlock
// level, cumulatively, in printed order, with no hard-coded list (class-warlock.md FR-033, FR-034, FR-126, table 4.8.2). A subclass the
// definition does not list gives none of its own. The spells do not count against the Prepared Spells number and are not on the Warlock
// list unless Spell.Classes says so; the spellcasting service governs both. It returns a new slice.
func WarlockAlwaysPrepared(p WarlockProfile, def content.ClassDefinition) []content.SpellID {
	out := []content.SpellID{}
	collect := func(features []content.FeatureEntry) {
		for _, f := range features {
			for _, fact := range f.Facts() {
				if a, ok := fact.(content.AlwaysPreparedSpells); ok {
					for _, grant := range a.Grants() {
						if grant.Level() <= p.level.n {
							out = append(out, grant.SpellIDs()...)
						}
					}
				}
			}
		}
	}
	collect(def.Features())
	if found, ok := def.Subclass(p.subclass).(content.SubclassFound); ok && p.subclass != "" {
		collect(found.Subclass().Features())
	}
	return out
}

// pactSlotsOf reads the Pact Magic pool of the state: the Slot Level, the number of slots and how many remain, all 0 for a state with
// no Pact Magic. The slots all share one level, so the first level with a maximum is the Slot Level.
func pactSlotsOf(st SpellcastingState) (level, count, remaining int) {
	v := st.Slots(PoolPactMagic)
	for l := 1; l <= 9; l++ {
		sl := MustSlotLevel(l)
		if m := v.Maximum(sl); m > 0 {
			return l, m, v.Remaining(sl)
		}
	}
	return 0, 0, 0
}
