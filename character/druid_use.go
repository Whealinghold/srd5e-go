package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ValueClassDruid is the srd5e.class attribute value of the Druid in the shared feature_used counter (class-druid.md FR-165).
const ValueClassDruid = "druid"

// DruidSlotChange is a change of a spell slot the host applies, a closed set: the Druid owns no slot pool and never changes
// one (class-druid.md FR-140, FR-160). The host applies each with SpellcastingState.ExpendSlot or RegainSlot, which re-check
// the slot. Variants: DruidSlotChangeSpend, DruidSlotChangeRegain.
type DruidSlotChange interface{ isDruidSlotChange() }

// DruidSlotChangeSpend: expend one slot of Level from Pool.
type DruidSlotChangeSpend struct {
	Pool  SpellSlotPool
	Level int
}

// DruidSlotChangeRegain: regain one expended slot of Level in Pool.
type DruidSlotChangeRegain struct {
	Pool  SpellSlotPool
	Level int
}

func (DruidSlotChangeSpend) isDruidSlotChange()  {}
func (DruidSlotChangeRegain) isDruidSlotChange() {}

// druidUse is what a feature use reads: the state, the tables, the facts and the character's spell slots.
type druidUse struct {
	state  DruidFeatureState
	tables DruidFeatureTables
	facts  DruidCharacterFacts
	slots  SpellcastingState
}

func (u druidUse) level() int { return u.facts.lv() }

// druidUseOutcome is what a use produced: a reason, or the new state, the slot changes and the events.
type druidUseOutcome struct {
	reason  DruidRejection
	state   DruidFeatureState
	changes []DruidSlotChange
	events  []Event
}

func (u druidUse) rejected(reasons ...DruidRejection) (druidUseOutcome, bool) {
	r, bad := lowestDruidRejection(reasons...)
	return druidUseOutcome{reason: r, state: u.state}, bad
}

// slotOf returns the slot level n of the pool and how many slots of it remain and are expended. It returns an error matching
// srderr.ErrInvalidInput for a pool outside the two pools or a level outside 1 to 9.
func (u druidUse) slotOf(pool SpellSlotPool, n int) (remaining, expended int, err error) {
	l, lerr := NewSlotLevel(n)
	if lerr != nil || !pool.Valid() {
		return 0, 0, invalidInput("Druid slot operation with a pool or a slot level outside the closed sets")
	}
	v := u.slots.Slots(pool)
	return v.Remaining(l), v.Expended(l), nil
}

// DruidFeatureUse is a use of a Druid feature, a closed set. Variants: DruidUseAssumeWildShape, DruidUseEndWildShape,
// DruidUseWildCompanion, DruidUseSlotForWildShapeUse, DruidUseWildShapeUseForSlot, DruidUseLandsAid,
// DruidUseNaturalRecoveryCast, DruidUseNaturalRecoveryRecover, DruidUseCreateSanctuary, DruidUseMoveSanctuary,
// DruidUseEndSanctuary, DruidUseNatureMagician, DruidUsePrimalStrike.
type DruidFeatureUse interface {
	apply(u druidUse) (druidUseOutcome, error)
}

// UseDruidFeatureInput is what UseDruidFeature reads. Construct it with a keyed literal.
type UseDruidFeatureInput struct {
	State  DruidFeatureState
	Tables DruidFeatureTables
	Facts  DruidCharacterFacts
	// Spellcasting is the character's spell slot state, read only to check that a slot is available or expended. The zero value
	// has no slot.
	Spellcasting SpellcastingState
	Use          DruidFeatureUse
}

// UseDruidFeatureResult is the outcome of UseDruidFeature, a closed set. Variants: UseDruidFeatureUsed, UseDruidFeatureRejected.
type UseDruidFeatureResult interface{ isUseDruidFeatureResult() }

// UseDruidFeatureUsed: the feature was used. Hold State, apply the SlotChanges with ExpendSlot or RegainSlot, publish Events.
type UseDruidFeatureUsed struct {
	state   DruidFeatureState
	changes []DruidSlotChange
	events  []Event
}

// State returns the new Druid state.
func (r UseDruidFeatureUsed) State() DruidFeatureState { return r.state }

// SlotChanges returns the slot changes the host applies, one per slot, as a new slice.
func (r UseDruidFeatureUsed) SlotChanges() []DruidSlotChange {
	return append([]DruidSlotChange{}, r.changes...)
}

// Events returns the events, as a new slice.
func (r UseDruidFeatureUsed) Events() []Event { return append([]Event{}, r.events...) }

// UseDruidFeatureRejected: the feature was not used and the state is unchanged. Reason is the first of the D-5 order. For the
// two slot reasons, Slot is the rejection the pool would give.
type UseDruidFeatureRejected struct{ Reason DruidRejection }

// Slot returns the SlotOperationRejection the Reason stands for, and false for any other reason.
func (r UseDruidFeatureRejected) Slot() (SlotOperationRejection, bool) {
	if r.Reason == DruidNoSlotAvailable {
		return SlotOperationNoSlotAvailable, true
	}
	if r.Reason == DruidNoExpendedSlot {
		return SlotOperationNoExpendedSlot, true
	}
	return 0, false
}

func (UseDruidFeatureUsed) isUseDruidFeatureResult()     {}
func (UseDruidFeatureRejected) isUseDruidFeatureResult() {}

// UseDruidFeature uses a Druid feature (class-druid.md FR-057 to FR-138, FR-155). It is pure: it reads the character's slots
// and returns the slot changes as data, never changing a pool. When several reasons hold the rejection is the first of the D-5
// order.
//
// It returns an error matching srderr.ErrInvalidInput for zero facts or tables, a State that breaks an invariant, a nil use or
// a value outside its set (a slot level outside 1 to 9, an undefined pool, a negative distance). It never panics.
func UseDruidFeature(in UseDruidFeatureInput) (UseDruidFeatureResult, error) {
	if err := in.State.check(in.Facts, in.Tables); err != nil {
		return nil, err
	}
	if in.Use == nil {
		return nil, invalidInput("Druid feature use is nil")
	}
	out, err := in.Use.apply(druidUse{state: in.State, tables: in.Tables, facts: in.Facts, slots: in.Spellcasting})
	if err != nil {
		return nil, err
	}
	if out.reason != 0 {
		return UseDruidFeatureRejected{Reason: out.reason}, nil
	}
	return UseDruidFeatureUsed{state: out.state, changes: out.changes, events: out.events}, nil
}

// DruidUseAssumeWildShape assumes a Beast form as a Bonus Action, spending one use; from a form it is a switch that ends the
// current form first (SRD 5.2.1, "Level 2: Wild Shape", p. 42). Items names every worn or carried item once. Construct it with
// a keyed literal.
type DruidUseAssumeWildShape struct {
	Form  content.MonsterID
	Items []DruidItemPlan
}

func (c DruidUseAssumeWildShape) apply(u druidUse) (druidUseOutcome, error) {
	var worn, merged []DruidItemKey
	var unassigned, cannotWear bool
	for _, it := range c.Items {
		switch it.Disposition {
		case EquipmentDispositionFallsInSpace:
		case EquipmentDispositionMergesIntoForm:
			merged = append(merged, it.Key)
		case EquipmentDispositionWorn:
			if it.FormCanWear {
				worn = append(worn, it.Key)
			}
			cannotWear = cannotWear || !it.FormCanWear
		default:
			if it.Disposition != 0 {
				return druidUseOutcome{}, invalidInput("equipment disposition outside the closed set")
			}
			unassigned = true
		}
	}
	if out, bad := u.rejected(when(!u.facts.hasWildShape(), DruidFeatureNotGranted), when(!u.state.known.Contains(c.Form), DruidFormNotKnown),
		when(unassigned, DruidItemUnassigned), when(cannotWear, DruidItemCannotBeWorn), when(u.state.uses < 1, DruidNoUsesRemaining)); bad {
		return out, nil
	}
	id := u.facts.id
	var events []Event
	if previous, inForm := u.state.ActiveForm(); inForm {
		events = append(events, WildShapeEnded{id: id, form: previous, reason: WildShapeEndUsedWildShapeAgain})
	}
	u.state.uses--
	u.state.active = c.Form
	events = append(events, WildShapeAssumed{id: id, form: c.Form, duration: time.Duration(u.level()) * 30 * time.Minute, temporaryHP: u.level(),
		worn: slices.Clone(worn), merged: slices.Clone(merged)})
	return druidUseOutcome{state: u.state, events: events}, nil
}

// DruidUseEndWildShape ends the Beast form: by its duration, the Incapacitated condition, death, or by leaving it early as a
// Bonus Action. No use is refunded. Reason is one of those four; a switch of forms ends a form by DruidUseAssumeWildShape.
// Construct it with a keyed literal.
type DruidUseEndWildShape struct{ Reason WildShapeEndReason }

func (c DruidUseEndWildShape) apply(u druidUse) (druidUseOutcome, error) {
	if c.Reason < WildShapeEndDurationElapsed || c.Reason > WildShapeEndLeftEarly || c.Reason == WildShapeEndUsedWildShapeAgain {
		return druidUseOutcome{}, invalidInput("Wild Shape end reason outside the four a host reports")
	}
	form, inForm := u.state.ActiveForm()
	if out, bad := u.rejected(when(!u.facts.hasWildShape(), DruidFeatureNotGranted), when(!inForm, DruidNotInBeastForm)); bad {
		return out, nil
	}
	u.state.active = ""
	return druidUseOutcome{state: u.state, events: []Event{WildShapeEnded{id: u.facts.id, form: form, reason: c.Reason}}}, nil
}

// DruidUseWildCompanion casts Find Familiar without Material components as a Magic action, paying a spell slot (a
// DruidPaymentSlot, 1 or higher) or a use of Wild Shape (a DruidPaymentWildShape); the familiar is Fey and goes at the end of a
// Long Rest (SRD 5.2.1, "Level 2: Wild Companion", p. 43). Find Familiar need not be prepared. Construct it with a keyed literal.
type DruidUseWildCompanion struct{ Payment DruidPayment }

func (c DruidUseWildCompanion) apply(u druidUse) (druidUseOutcome, error) {
	var paid DruidRejection
	var changes []DruidSlotChange
	switch p := c.Payment.(type) {
	case DruidPaymentSlot:
		remaining, _, err := u.slotOf(p.Pool, p.Level)
		if err != nil {
			return druidUseOutcome{}, err
		}
		paid = when(remaining < 1, DruidNoSlotAvailable)
		changes = []DruidSlotChange{DruidSlotChangeSpend(p)}
	case DruidPaymentWildShape:
		paid = when(u.state.uses < 1, DruidNoUsesRemaining)
	default:
		return druidUseOutcome{}, invalidInput("Wild Companion payment is nil or not one of the closed set")
	}
	if out, bad := u.rejected(when(u.level() < 2, DruidFeatureNotGranted), beastFormCastBlock(u.state, u.level(), false), paid); bad {
		return out, nil
	}
	if _, ws := c.Payment.(DruidPaymentWildShape); ws {
		u.state.uses--
	}
	u.state.familiar = true
	return druidUseOutcome{state: u.state, changes: changes, events: []Event{WildCompanionCast{id: u.facts.id, payment: c.Payment}}}, nil
}

// DruidUseSlotForWildShapeUse is Wild Resurgence's first exchange: with no Wild Shape use left, expend one spell slot of Level
// to give yourself one use, no action required, once on each turn (SRD 5.2.1, "Level 5: Wild Resurgence", p. 43). Construct it
// with a keyed literal.
type DruidUseSlotForWildShapeUse struct {
	Pool  SpellSlotPool
	Level int
}

func (c DruidUseSlotForWildShapeUse) apply(u druidUse) (druidUseOutcome, error) {
	remaining, _, err := u.slotOf(c.Pool, c.Level)
	if err != nil {
		return druidUseOutcome{}, err
	}
	if out, bad := u.rejected(when(u.level() < 5, DruidFeatureNotGranted), when(u.state.wrTurn, DruidAlreadyUsedThisTurn),
		when(u.state.uses > 0, DruidWildShapeUsesRemain), when(remaining < 1, DruidNoSlotAvailable)); bad {
		return out, nil
	}
	u.state.uses++
	u.state.wrTurn = true
	return druidUseOutcome{state: u.state, changes: []DruidSlotChange{DruidSlotChangeSpend(c)},
		events: []Event{SpellSlotsExchanged{id: u.facts.id, kind: SlotExchangeSlotForUse, slotLevel: c.Level, uses: 1}}}, nil
}

// DruidUseWildShapeUseForSlot is Wild Resurgence's second exchange: expend one Wild Shape use to regain an expended level 1
// spell slot, no action required, once until a Long Rest (SRD 5.2.1, "Level 5: Wild Resurgence", p. 43). Construct it with a
// keyed literal.
type DruidUseWildShapeUseForSlot struct{ Pool SpellSlotPool }

func (c DruidUseWildShapeUseForSlot) apply(u druidUse) (druidUseOutcome, error) {
	_, expended, err := u.slotOf(c.Pool, 1)
	if err != nil {
		return druidUseOutcome{}, err
	}
	if out, bad := u.rejected(when(u.level() < 5, DruidFeatureNotGranted), when(u.state.wrLongRest, DruidAlreadyUsedSinceLongRest),
		when(u.state.uses < 1, DruidNoUsesRemaining), when(expended < 1, DruidNoExpendedSlot)); bad {
		return out, nil
	}
	u.state.uses--
	u.state.wrLongRest = true
	return druidUseOutcome{state: u.state, changes: []DruidSlotChange{DruidSlotChangeRegain{Pool: c.Pool, Level: 1}},
		events: []Event{SpellSlotsExchanged{id: u.facts.id, kind: SlotExchangeUseForSlot, slotLevel: 1, uses: 1}}}, nil
}

// DruidUseLandsAid is Land's Aid: spend one Wild Shape use as a Magic action (SRD 5.2.1, "Level 3: Land's Aid", p. 46). DC is
// the spell save DC the caller computed from the character's Spellcasting. Construct it with a keyed literal.
type DruidUseLandsAid struct{ DC SpellSaveDC }

func (c DruidUseLandsAid) apply(u druidUse) (druidUseOutcome, error) {
	if c.DC.Value() < 1 {
		return druidUseOutcome{}, invalidInput("Land's Aid with a zero spell save DC")
	}
	if out, bad := u.rejected(when(!u.facts.hasCircle() || u.level() < 3, DruidFeatureNotGranted), when(u.state.uses < 1, DruidNoUsesRemaining)); bad {
		return out, nil
	}
	u.state.uses--
	d := DruidRules{}.LandsAidDice(u.facts.level)
	return druidUseOutcome{state: u.state, events: []Event{LandsAidUsed{id: u.facts.id, dc: c.DC, damage: d, healing: d}}}, nil
}

// DruidUseNaturalRecoveryCast casts one level 1+ Circle Spell of the current land without expending a slot, once until a Long
// Rest (SRD 5.2.1, "Level 6: Natural Recovery", p. 46). It consults the Beast form casting rule with the spell. Construct it
// with a keyed literal.
type DruidUseNaturalRecoveryCast struct{ Spell content.Spell }

func (c DruidUseNaturalRecoveryCast) apply(u druidUse) (druidUseOutcome, error) {
	if c.Spell.ID() == "" {
		return druidUseOutcome{}, invalidInput("Natural Recovery casting with a zero spell")
	}
	circle := slices.Contains(DruidRules{}.CircleSpells(u.tables, u.state.land, u.facts.level), c.Spell.ID())
	if out, bad := u.rejected(when(!u.facts.hasCircle() || u.level() < 6, DruidFeatureNotGranted), when(u.state.land == 0, DruidLandTypeNotChosen),
		beastFormCastBlock(u.state, u.level(), materialRestricted(c.Spell)), when(u.state.nrCast, DruidAlreadyUsedSinceLongRest),
		when(c.Spell.Level().IsCantrip() || !circle, DruidNotACircleSpell)); bad {
		return out, nil
	}
	u.state.nrCast = true
	return druidUseOutcome{state: u.state, events: []Event{NaturalRecoveryUsed{id: u.facts.id, spell: c.Spell.ID()}}}, nil
}

// DruidUseNaturalRecoveryRecover recovers expended slots after a Short Rest: Levels are the levels of the slots, combining to
// at most half the Druid level rounded up, none of level 6 or higher, once until a Long Rest (SRD 5.2.1, "Level 6: Natural
// Recovery", p. 46). No levels recovers nothing and changes nothing. Construct it with a keyed literal.
type DruidUseNaturalRecoveryRecover struct {
	Pool   SpellSlotPool
	Levels []int
}

func (c DruidUseNaturalRecoveryRecover) apply(u druidUse) (druidUseOutcome, error) {
	var total, tooHigh int
	var wanted [10]int
	for _, n := range c.Levels {
		if _, _, err := u.slotOf(c.Pool, n); err != nil {
			return druidUseOutcome{}, err
		}
		total += n
		wanted[n]++
		if n >= 6 {
			tooHigh++
		}
	}
	short := false
	for n, count := range wanted {
		if count > 0 {
			_, expended, _ := u.slotOf(c.Pool, n) // the level was validated above
			short = short || count > expended
		}
	}
	if out, bad := u.rejected(when(!u.facts.hasCircle() || u.level() < 6, DruidFeatureNotGranted),
		when(!u.state.window.has(RestWindowNaturalRecoveryRecovery), DruidRecoveryNotAllowedNow), when(u.state.nrRecover, DruidAlreadyUsedSinceLongRest),
		when(tooHigh > 0, DruidSlotLevelTooHigh), when(total > DruidRules{}.NaturalRecoveryBudget(u.facts.level), DruidRecoveryBudgetExceeded),
		when(short, DruidNoExpendedSlot)); bad {
		return out, nil
	}
	if len(c.Levels) == 0 {
		return druidUseOutcome{state: u.state}, nil
	}
	u.state.nrRecover = true
	u.state.window = u.state.window.without(RestWindowNaturalRecoveryRecovery)
	changes := make([]DruidSlotChange, len(c.Levels))
	for i, n := range c.Levels {
		changes[i] = DruidSlotChangeRegain{Pool: c.Pool, Level: n}
	}
	return druidUseOutcome{state: u.state, changes: changes, events: []Event{NaturalRecoveryUsed{id: u.facts.id, slots: slices.Clone(c.Levels)}}}, nil
}

// DruidUseCreateSanctuary is Nature's Sanctuary: spend one Wild Shape use as a Magic action to create a 15-foot Cube on the
// ground DistanceFeet from the Druid, at most 120, lasting 1 minute (SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46).
// Construct it with a keyed literal.
type DruidUseCreateSanctuary struct{ DistanceFeet int }

func (c DruidUseCreateSanctuary) apply(u druidUse) (druidUseOutcome, error) {
	if c.DistanceFeet < 0 {
		return druidUseOutcome{}, invalidInput("Nature's Sanctuary at a negative distance")
	}
	if out, bad := u.rejected(when(!u.facts.hasCircle() || u.level() < 14, DruidFeatureNotGranted), when(u.state.sanctuary, DruidSanctuaryAlreadyActive),
		when(c.DistanceFeet > 120, DruidMoveTooFar), when(u.state.uses < 1, DruidNoUsesRemaining)); bad {
		return out, nil
	}
	u.state.uses--
	u.state.sanctuary = true
	resistance, _ := u.tables.Resistance(u.state.land)
	return druidUseOutcome{state: u.state, events: []Event{NaturesSanctuaryCreated{id: u.facts.id, resistance: resistance}}}, nil
}

// DruidUseMoveSanctuary moves the Cube up to MoveFeet, at most 60, to ground DistanceFeet from the Druid, at most 120, as a
// Bonus Action. Construct it with a keyed literal.
type DruidUseMoveSanctuary struct{ MoveFeet, DistanceFeet int }

func (c DruidUseMoveSanctuary) apply(u druidUse) (druidUseOutcome, error) {
	if c.MoveFeet < 0 || c.DistanceFeet < 0 {
		return druidUseOutcome{}, invalidInput("Nature's Sanctuary move with a negative distance")
	}
	if out, bad := u.rejected(when(!u.facts.hasCircle() || u.level() < 14, DruidFeatureNotGranted), when(!u.state.sanctuary, DruidNoActiveSanctuary),
		when(c.MoveFeet > 60 || c.DistanceFeet > 120, DruidMoveTooFar)); bad {
		return out, nil
	}
	return druidUseOutcome{state: u.state, events: []Event{NaturesSanctuaryMoved{id: u.facts.id}}}, nil
}

// DruidUseEndSanctuary ends the Cube when the minute elapses or the Druid is Incapacitated or dies. Construct it with a keyed
// literal.
type DruidUseEndSanctuary struct{ Reason SanctuaryEndReason }

func (c DruidUseEndSanctuary) apply(u druidUse) (druidUseOutcome, error) {
	if c.Reason < SanctuaryEndMinuteElapsed || c.Reason > SanctuaryEndDied {
		return druidUseOutcome{}, invalidInput("Nature's Sanctuary end reason outside the closed set")
	}
	if out, bad := u.rejected(when(!u.facts.hasCircle() || u.level() < 14, DruidFeatureNotGranted), when(!u.state.sanctuary, DruidNoActiveSanctuary)); bad {
		return out, nil
	}
	u.state.sanctuary = false
	return druidUseOutcome{state: u.state, events: []Event{NaturesSanctuaryEnded{id: u.facts.id, reason: c.Reason}}}, nil
}

// DruidUseNatureMagician converts Uses unexpended Wild Shape uses into one expended spell slot of level 2 x Uses, no action
// required, once until a Long Rest (SRD 5.2.1, "Level 20: Archdruid", p. 43). Construct it with a keyed literal.
type DruidUseNatureMagician struct {
	Pool SpellSlotPool
	Uses int
}

func (c DruidUseNatureMagician) apply(u druidUse) (druidUseOutcome, error) {
	var expended int
	if c.Uses >= 1 && 2*c.Uses <= 9 {
		var err error
		if _, expended, err = u.slotOf(c.Pool, 2*c.Uses); err != nil {
			return druidUseOutcome{}, err
		}
	} else if !c.Pool.Valid() {
		return druidUseOutcome{}, invalidInput("Nature Magician with an undefined slot pool")
	}
	if out, bad := u.rejected(when(u.level() < 20, DruidFeatureNotGranted), when(u.state.nature, DruidAlreadyUsedSinceLongRest),
		when(c.Uses < 1, DruidInvalidConversion), when(c.Uses > u.state.uses, DruidNoUsesRemaining), when(c.Uses >= 1 && expended < 1, DruidNoExpendedSlot)); bad {
		return out, nil
	}
	u.state.uses -= c.Uses
	u.state.nature = true
	return druidUseOutcome{state: u.state, changes: []DruidSlotChange{DruidSlotChangeRegain{Pool: c.Pool, Level: 2 * c.Uses}},
		events: []Event{SpellSlotsExchanged{id: u.facts.id, kind: SlotExchangeNatureMagician, slotLevel: 2 * c.Uses, uses: c.Uses}}}, nil
}

// DruidUsePrimalStrike is Primal Strike: extra damage on a hit with a weapon, or with a Beast form's attack in Wild Shape, once
// on each turn (SRD 5.2.1, "Level 7: Elemental Fury", p. 43). The host supplies the attack's source and its result. Construct
// it with a keyed literal.
type DruidUsePrimalStrike struct {
	Source PrimalStrikeAttackSource
	Attack d20.AttackRollResult
}

func (c DruidUsePrimalStrike) apply(u druidUse) (druidUseOutcome, error) {
	if !c.Source.Valid() {
		return druidUseOutcome{}, invalidInput("Primal Strike attack source outside the closed set")
	}
	outcome := DruidRules{}.PrimalStrikeDice(u.state, u.facts.level, c.Source, c.Attack)
	var ineligible DruidRejection
	if i, isIneligible := outcome.(PrimalStrikeOutcomeIneligible); isIneligible {
		ineligible = i.Reason
	}
	if out, bad := u.rejected(when(u.level() < 7, DruidFeatureNotGranted), when(u.state.psTurn, DruidAlreadyUsedThisTurn), ineligible); bad {
		return out, nil
	}
	damage := outcome.(PrimalStrikeOutcomeDamage).Damage // no reason applied, so the outcome is damage
	u.state.psTurn = true
	return druidUseOutcome{state: u.state, events: []Event{PrimalStrikeUsed{id: u.facts.id, extra: damage}}}, nil
}

// DruidFeatureAttribute is the srd5e.feature value of a Druid feature use in the shared feature_used counter (class-druid.md
// FR-165). The zero value is not an attribute.
type DruidFeatureAttribute int

const (
	// DruidFeatureWildShape is "wild_shape": assuming or ending a form.
	DruidFeatureWildShape DruidFeatureAttribute = iota + 1
	// DruidFeatureWildCompanion is "wild_companion".
	DruidFeatureWildCompanion
	// DruidFeatureWildResurgence is "wild_resurgence": either exchange.
	DruidFeatureWildResurgence
	// DruidFeatureLandsAid is "lands_aid".
	DruidFeatureLandsAid
	// DruidFeatureNaturalRecovery is "natural_recovery": the casting or the slot recovery.
	DruidFeatureNaturalRecovery
	// DruidFeatureNaturesSanctuary is "natures_sanctuary": creating, moving or ending the Cube.
	DruidFeatureNaturesSanctuary
	// DruidFeatureNatureMagician is "nature_magician".
	DruidFeatureNatureMagician
	// DruidFeaturePrimalStrike is "primal_strike".
	DruidFeaturePrimalStrike
)

func druidFeatureAttributeNames() []string {
	return []string{"wild_shape", "wild_companion", "wild_resurgence", "lands_aid", "natural_recovery", "natures_sanctuary", "nature_magician", "primal_strike"}
}

// String returns the srd5e.feature value ("wild_shape", ...) or "unknown".
func (a DruidFeatureAttribute) String() string {
	if a < 1 || int(a) > len(druidFeatureAttributeNames()) {
		return "unknown"
	}
	return druidFeatureAttributeNames()[a-1]
}

// FeatureAttribute maps a recordable use to the srd5e.feature value of the shared counter srd5e.character.feature_used, with
// srd5e.class ValueClassDruid. It returns false for nil and for every passive effect, which is not a use. Nothing here emits
// the counter: the host does (class-druid.md FR-165); for a failure, srd5e.character.rejection is DruidRejection.String.
func (DruidRules) FeatureAttribute(use DruidFeatureUse) (DruidFeatureAttribute, bool) {
	switch use.(type) {
	case DruidUseAssumeWildShape, DruidUseEndWildShape:
		return DruidFeatureWildShape, true
	case DruidUseWildCompanion:
		return DruidFeatureWildCompanion, true
	case DruidUseSlotForWildShapeUse, DruidUseWildShapeUseForSlot:
		return DruidFeatureWildResurgence, true
	case DruidUseLandsAid:
		return DruidFeatureLandsAid, true
	case DruidUseNaturalRecoveryCast, DruidUseNaturalRecoveryRecover:
		return DruidFeatureNaturalRecovery, true
	case DruidUseCreateSanctuary, DruidUseMoveSanctuary, DruidUseEndSanctuary:
		return DruidFeatureNaturesSanctuary, true
	case DruidUseNatureMagician:
		return DruidFeatureNatureMagician, true
	case DruidUsePrimalStrike:
		return DruidFeaturePrimalStrike, true
	}
	return 0, false
}
