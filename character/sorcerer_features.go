package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// SorcererTransition is the closed outcome of a Sorcerer domain function that changes the SorcererState. Variants:
// SorcererChanged, SorcererRefused.
type SorcererTransition interface{ isSorcererTransition() }

// SorcererChanged is the outcome of an operation the rules allowed: the new state and the events it produced.
type SorcererChanged struct {
	state  SorcererState
	events []Event
}

// State returns the new SorcererState.
func (r SorcererChanged) State() SorcererState { return r.state }

// Events returns the events, as a new slice.
func (r SorcererChanged) Events() []Event { return slices.Clone(r.events) }

// SorcererRefused is the outcome of an operation a rule refused: the state is unchanged.
type SorcererRefused struct{ rejection SorcererRejection }

// Rejection returns why the operation was refused.
func (r SorcererRefused) Rejection() SorcererRejection { return r.rejection }

func (SorcererChanged) isSorcererTransition() {}
func (SorcererRefused) isSorcererTransition() {}

func sorcChanged(s SorcererState, events ...Event) SorcererTransition {
	return SorcererChanged{state: s, events: events}
}
func sorcRefused(r SorcererRejection) SorcererTransition { return SorcererRefused{rejection: r} }

// SorcererSlotTransition is the closed outcome of ConvertSpellSlot. Variants: SorcererSlotsChanged, SorcererSlotsRefused.
type SorcererSlotTransition interface{ isSorcererSlotTransition() }

// SorcererSlotsChanged is the outcome of a conversion the rules allowed: the new Sorcerer state, the new spellcasting state
// (unchanged for a created slot) and the events.
type SorcererSlotsChanged struct {
	state        SorcererState
	spellcasting SpellcastingState
	events       []Event
}

// State returns the new SorcererState.
func (r SorcererSlotsChanged) State() SorcererState { return r.state }

// Spellcasting returns the new SpellcastingState.
func (r SorcererSlotsChanged) Spellcasting() SpellcastingState { return r.spellcasting }

// Events returns the events, as a new slice.
func (r SorcererSlotsChanged) Events() []Event { return slices.Clone(r.events) }

// SorcererSlotsRefused is the outcome of a conversion a rule refused: nothing changed.
type SorcererSlotsRefused struct{ rejection SorcererRejection }

// Rejection returns why the conversion was refused.
func (r SorcererSlotsRefused) Rejection() SorcererRejection { return r.rejection }

func (SorcererSlotsChanged) isSorcererSlotTransition() {}
func (SorcererSlotsRefused) isSorcererSlotTransition() {}

// ---- Innate Sorcery -----------------------------------------------------

// InnateSorceryPayment is how Innate Sorcery is activated, a closed set. The zero value is read as InnateSorceryNormalUse.
type InnateSorceryPayment int

const (
	// InnateSorceryNormalUse spends one of the two uses (SRD 5.2.1, "Level 1: Innate Sorcery", p. 66).
	InnateSorceryNormalUse InnateSorceryPayment = iota
	// InnateSorceryPaidWithPoints spends 2 Sorcery Points with no uses left, from level 7 (SRD 5.2.1, "Level 7: Sorcery Incarnate", p. 66).
	InnateSorceryPaidWithPoints
)

// ActivateInnateSorcery activates Innate Sorcery (a Bonus Action; class-sorcerer.md FR-037 to FR-041, FR-044). It refuses
// with SorcererInnateSorceryAlreadyActive when it is active; by the normal route with SorcererInnateSorceryNoUsesLeft when no
// use remains; by the points route with SorcererInnateSorceryUsesRemain when a use remains, SorcererSorceryIncarnateUnavailable
// below level 7 and SorcererSorceryPointsInsufficient with fewer than 2 points. It never panics; id is carried by the event.
func ActivateInnateSorcery(id Identifier, s SorcererState, p SorcererProfile, pay InnateSorceryPayment) SorcererTransition {
	switch {
	case s.active:
		return sorcRefused(refuse(SorcererInnateSorceryAlreadyActive))
	case pay != InnateSorceryPaidWithPoints && s.uses == 0:
		return sorcRefused(refuse(SorcererInnateSorceryNoUsesLeft))
	case pay != InnateSorceryPaidWithPoints:
		s.uses--
	case s.uses > 0:
		return sorcRefused(refuse(SorcererInnateSorceryUsesRemain))
	case p.level.n < sorcererIncarnateLevel:
		return sorcRefused(refuse(SorcererSorceryIncarnateUnavailable))
	case s.points < innateSorceryPointCost:
		return sorcRefused(SorcererRejection{Kind: SorcererSorceryPointsInsufficient, Required: innateSorceryPointCost, Available: s.points})
	default:
		s.points -= innateSorceryPointCost
	}
	s.active = true
	return sorcChanged(s, InnateSorceryActivated{id: id, paidPoints: pay == InnateSorceryPaidWithPoints})
}

// EndInnateSorcery ends Innate Sorcery after its 1 minute (class-sorcerer.md FR-043). It refuses with
// SorcererInnateSorceryNotActive when it is not active.
func EndInnateSorcery(id Identifier, s SorcererState) SorcererTransition {
	if !s.active {
		return sorcRefused(refuse(SorcererInnateSorceryNotActive))
	}
	s.active = false
	return sorcChanged(s, InnateSorceryEnded{id: id})
}

// StartSorcererTurn clears the per-turn flags, the free Metamagic use spent and Quickened Spell used (class-sorcerer.md
// FR-109). Nothing else clears them, and ending Innate Sorcery does not. It never refuses.
func StartSorcererTurn(s SorcererState) SorcererState {
	s.freeSpent, s.quickened = false, false
	return s
}

// InnateSorceryEffects are the benefits of Innate Sorcery for the host to apply (class-sorcerer.md FR-037, FR-133). Read it;
// SorcererInnateSorceryEffects builds it.
type InnateSorceryEffects struct {
	// SaveDC is the Modifier to add to the spell save DC of Sorcerer spells: +1 while active, 0 otherwise.
	SaveDC modifier.Modifier
	// AttackRolls is the circumstance of the attack rolls of Sorcerer spells: Advantage while active, nil otherwise.
	AttackRolls modifier.RollCircumstance
}

// SorcererInnateSorceryEffects returns the effects of Innate Sorcery in the state (class-sorcerer.md FR-133). The host applies
// them to the Sorcerer spells and to the always-prepared spells a Sorcerer feature grants (FR-103).
func SorcererInnateSorceryEffects(s SorcererState) InnateSorceryEffects {
	if !s.active {
		return InnateSorceryEffects{SaveDC: modifier.New(0)}
	}
	return InnateSorceryEffects{SaveDC: modifier.New(1), AttackRolls: modifier.Advantage{}}
}

// SorcererActiveEffects are the effects a Sorcerer holds for the host to apply (class-sorcerer.md FR-135). Read it.
type SorcererActiveEffects struct {
	resistances []content.DamageType
	// FlySpeedFeet is the Fly Speed while Dragon Wings is active, 0 when it is not.
	FlySpeedFeet int
}

// Resistances returns the damage types the Sorcerer is resistant to through Elemental Affinity, as a new slice (empty before a choice).
func (e SorcererActiveEffects) Resistances() []content.DamageType { return slices.Clone(e.resistances) }

// SorcererEffects returns the active effects of the state: the Elemental Affinity Resistance and the Dragon Wings Fly Speed of 60 feet.
func SorcererEffects(s SorcererState) SorcererActiveEffects {
	e := SorcererActiveEffects{resistances: []content.DamageType{}}
	if t, ok := s.ElementalAffinity(); ok {
		e.resistances = append(e.resistances, t)
	}
	if s.wingsActive {
		e.FlySpeedFeet = 60
	}
	return e
}

// ---- Font of Magic ------------------------------------------------------

// SlotConversionSource says which slot ConvertSpellSlot spends, a closed set. Variants: ConvertOrdinarySlot, ConvertCreatedSlot.
type SlotConversionSource interface{ isSlotConversionSource() }

// ConvertOrdinarySlot converts a slot of the Spellcasting pool of State.
type ConvertOrdinarySlot struct{ State SpellcastingState }

// ConvertCreatedSlot converts a slot created with Font of Magic, held in the SorcererState.
type ConvertCreatedSlot struct{}

func (ConvertOrdinarySlot) isSlotConversionSource() {}
func (ConvertCreatedSlot) isSlotConversionSource()  {}

// ConvertSpellSlot expends a spell slot to gain Sorcery Points equal to its level, no action required (class-sorcerer.md
// FR-050, FR-051, FR-114). It returns an error matching srderr.ErrInvalidInput for a slot level outside 1 to 9 or a nil source,
// and refuses with SorcererFeatureNotGained below level 2, SorcererNoSlotToConvert when no such slot is available and
// SorcererSorceryPointsWouldExceedMaximum when the gain would pass the maximum (never capped, so no slot is lost).
func ConvertSpellSlot(id Identifier, s SorcererState, p SorcererProfile, src SlotConversionSource, slotLevel int) (SorcererSlotTransition, error) {
	level, err := NewSlotLevel(slotLevel)
	if err != nil || src == nil {
		return nil, firstError(err, invalidInput("ConvertSpellSlot without a source"))
	}
	if p.level.n < sorcererFontOfMagicLevel {
		return SorcererSlotsRefused{rejection: notGained("font_of_magic")}, nil
	}
	next, sp, created := s, SpellcastingState{}, false
	var events []Event
	switch v := src.(type) {
	case ConvertOrdinarySlot:
		exp, ok := v.State.ExpendSlot(PoolSpellcasting, level).(SlotExpenditureExpended)
		if !ok {
			return SorcererSlotsRefused{rejection: refuse(SorcererNoSlotToConvert)}, nil
		}
		sp = exp.State()
		events = append(events, exp.Event())
	case ConvertCreatedSlot:
		created = true
		if s.CreatedSlots(slotLevel) == 0 {
			return SorcererSlotsRefused{rejection: refuse(SorcererNoSlotToConvert)}, nil
		}
		next.created[slotLevel-1]--
	}
	if top := p.level.SorceryPointsMaximum(); s.points+slotLevel > top {
		return SorcererSlotsRefused{rejection: SorcererRejection{Kind: SorcererSorceryPointsWouldExceedMaximum, Cap: top, Available: s.points, Required: slotLevel}}, nil
	}
	next.points += slotLevel
	events = append(events, SpellSlotConvertedToSorceryPoints{id: id, level: slotLevel, gained: slotLevel, created: created})
	return SorcererSlotsChanged{state: next, spellcasting: sp, events: events}, nil
}

// SlotCreationRule is the cost and minimum Sorcerer level of creating a spell slot of one level, read from the Creating Spell
// Slots table of the class definition (SRD 5.2.1, "Creating Spell Slots", p. 66).
type SlotCreationRule struct{ Cost, MinSorcererLevel int }

// SlotCreationRuleFor reads the rule for the slot level from the Creating Spell Slots table (columns Spell Slot Level, Sorcery
// Point Cost, Min. Sorcerer Level) and false when the table has no row for the level or a row that is not three integers.
func SlotCreationRuleFor(table content.FeatureTable, slotLevel int) (SlotCreationRule, bool) {
	for _, row := range table.Rows() {
		if len(row) != 3 {
			continue
		}
		l, ok1 := row[0].(content.TableCellInt)
		c, ok2 := row[1].(content.TableCellInt)
		m, ok3 := row[2].(content.TableCellInt)
		if ok1 && ok2 && ok3 && l.Value() == slotLevel {
			return SlotCreationRule{Cost: c.Value(), MinSorcererLevel: m.Value()}, true
		}
	}
	return SlotCreationRule{}, false
}

// CreateSlot transforms Sorcery Points into one spell slot of the level, a Bonus Action (class-sorcerer.md FR-052 to FR-054,
// FR-114), under the rule read from the Creating Spell Slots table (rule, found). It returns an error matching
// srderr.ErrInvalidInput for a slot level outside 1 to 9, and refuses with SorcererFeatureNotGained below level 2,
// SorcererSlotLevelNotCreatable for a level above 5 or a level the table lacks, SorcererLevelBelowMinimum and
// SorcererSorceryPointsInsufficient. The slot is added to the created slots, not to the spellcasting state.
func CreateSlot(id Identifier, s SorcererState, p SorcererProfile, slotLevel int, rule SlotCreationRule, found bool) (SorcererTransition, error) {
	if _, err := NewSlotLevel(slotLevel); err != nil {
		return nil, err
	}
	switch {
	case p.level.n < sorcererFontOfMagicLevel:
		return sorcRefused(notGained("font_of_magic")), nil
	case slotLevel > len(s.created) || !found:
		return sorcRefused(refuse(SorcererSlotLevelNotCreatable)), nil
	case p.level.n < rule.MinSorcererLevel:
		return sorcRefused(SorcererRejection{Kind: SorcererLevelBelowMinimum, Required: rule.MinSorcererLevel, Available: p.level.n}), nil
	case s.points < rule.Cost:
		return sorcRefused(SorcererRejection{Kind: SorcererSorceryPointsInsufficient, Required: rule.Cost, Available: s.points}), nil
	}
	s.points -= rule.Cost
	s.created[slotLevel-1]++
	return sorcChanged(s, SpellSlotCreated{id: id, level: slotLevel, cost: rule.Cost}), nil
}

// SpendCreatedSlot removes one created slot of createdLevel for a spell of spellLevel (class-sorcerer.md FR-136). The host then
// casts with CastModeFeatureGranted at the slot's level and sets SlotSpentThisTurn from the report. It returns an error matching
// srderr.ErrInvalidInput for a level outside 1 to 9, and refuses with SorcererCreatedSlotUnavailable when none exists,
// SorcererCreatedSlotLevelTooLow when the spell's level is above the slot's and SorcererCreatedSlotSpentThisTurn when a slot was
// already spent this turn (the cast context's SlotSpentThisTurn), in that order.
func SpendCreatedSlot(id Identifier, s SorcererState, ctx CastingContext, createdLevel, spellLevel int) (SorcererTransition, error) {
	if _, err := NewSlotLevel(createdLevel); err != nil {
		return nil, err
	}
	if _, err := NewSlotLevel(spellLevel); err != nil {
		return nil, err
	}
	switch {
	case s.CreatedSlots(createdLevel) == 0:
		return sorcRefused(refuse(SorcererCreatedSlotUnavailable)), nil
	case spellLevel > createdLevel:
		return sorcRefused(refuse(SorcererCreatedSlotLevelTooLow)), nil
	case ctx.Facts().SlotSpentThisTurn:
		return sorcRefused(refuse(SorcererCreatedSlotSpentThisTurn)), nil
	}
	s.created[createdLevel-1]--
	return sorcChanged(s, CreatedSlotSpent{id: id, slotLevel: createdLevel, spellLevel: spellLevel}), nil
}

// ---- Rests --------------------------------------------------------------

// ApplySorcererRest applies a finished rest (class-sorcerer.md FR-042, FR-049, FR-054, FR-083, FR-084, FR-093, FR-095,
// FR-106 to FR-108); the host calls it beside RestoreSpellSlots with the same kind. It returns an error matching
// srderr.ErrInvalidInput for a kind that is neither ShortRest nor LongRest.
//
// A LongRest restores all Sorcery Points, both Innate Sorcery uses, Sorcerous Restoration, Dragon Wings and Dragon Companion and
// removes every created slot (CreatedSlotsExpired); it refuses with SorcererRestorationNotAShortRest when restore is above 0.
// A ShortRest leaves created slots and runs Sorcerous Restoration for restore points, which must be the whole request: it
// refuses with SorcererFeatureNotGained below level 5, SorcererRestorationAlreadyUsed after a use, and
// SorcererRestorationAmountInvalid (Cap and Missing) for an amount below 1, above half the level rounded down or above the points
// missing, never clamping. A refusal leaves the whole state unchanged.
func ApplySorcererRest(id Identifier, s SorcererState, p SorcererProfile, kind RestKind, restore int) (SorcererTransition, error) {
	switch kind {
	case LongRest:
		if restore > 0 {
			return sorcRefused(refuse(SorcererRestorationNotAShortRest)), nil
		}
		expired := 0
		for _, n := range s.created {
			expired += n
		}
		s.points, s.uses, s.restoreUsed, s.wingsUsed, s.companionUsed, s.created = p.level.SorceryPointsMaximum(), innateSorceryUses, false, false, false, [5]int{}
		if expired > 0 {
			return sorcChanged(s, CreatedSlotsExpired{id: id, count: expired}), nil
		}
		return sorcChanged(s), nil
	case ShortRest:
		limit, missing := p.level.n/2, p.level.SorceryPointsMaximum()-s.points
		switch {
		case p.level.n < sorcererRestorationLevel:
			return sorcRefused(notGained("sorcerous_restoration")), nil
		case s.restoreUsed:
			return sorcRefused(refuse(SorcererRestorationAlreadyUsed)), nil
		case restore < 1 || restore > limit || restore > missing:
			return sorcRefused(SorcererRejection{Kind: SorcererRestorationAmountInvalid, Cap: limit, Missing: missing}), nil
		}
		s.points += restore
		s.restoreUsed = true
		return sorcChanged(s, SorceryPointsRestored{id: id, amount: restore, remaining: s.points}), nil
	}
	return nil, invalidInput("ApplySorcererRest with a rest that is neither short nor long")
}

// ---- Metamagic selection ------------------------------------------------

// MetamagicSelection is a validated set of Metamagic options, the full set a Sorcerer holds (class-sorcerer.md FR-056). Build it
// with NewMetamagicSelection; the zero value is the empty selection. It holds a slice: read it with Effects.
type MetamagicSelection struct{ effects []MetamagicEffect }

// Effects returns the selected effects in the printed order, as a new slice.
func (m MetamagicSelection) Effects() []MetamagicEffect { return slices.Clone(m.effects) }

// NewMetamagicSelection validates chosen against the class definition's Metamagic options (listed) for a Sorcerer of the level
// (class-sorcerer.md FR-055 to FR-057). It lists every problem together: MetamagicSelectionWrongSize (not exactly the entitled
// number), MetamagicSelectionDuplicate and MetamagicSelectionUnknownOption (an option the definition does not list or that is
// not one of the ten). The second result is empty and the first holds the selection when valid; with any problem the first is the
// zero value. Neither input is altered.
func NewMetamagicSelection(level SorcererLevel, chosen []content.FeatureOptionID, listed []content.FeatureOption) (MetamagicSelection, []MetamagicSelectionProblem) {
	problems := []MetamagicSelectionProblem{}
	add := func(p MetamagicSelectionProblem) {
		if !slices.Contains(problems, p) {
			problems = append(problems, p)
		}
	}
	if len(chosen) != level.MetamagicEntitled() {
		add(MetamagicSelectionWrongSize)
	}
	var sel uint16
	for i, id := range chosen {
		e, known := MetamagicEffectOf(id)
		switch {
		case slices.Contains(chosen[:i], id):
			add(MetamagicSelectionDuplicate)
		case !known || !slices.ContainsFunc(listed, func(o content.FeatureOption) bool { return o.ID() == id }):
			add(MetamagicSelectionUnknownOption)
		default:
			sel |= 1 << e
		}
	}
	if len(problems) != 0 {
		return MetamagicSelection{}, problems
	}
	out := SorcererState{metamagic: sel}
	return MetamagicSelection{effects: out.Metamagic()}, problems
}

// ChooseMetamagic makes sel the Metamagic options known (class-sorcerer.md FR-056, FR-058). It refuses with
// SorcererFeatureNotGained below level 2 and SorcererMetamagicReplacementInvalid when the selection drops more than one known option.
func ChooseMetamagic(id Identifier, s SorcererState, p SorcererProfile, sel MetamagicSelection) SorcererTransition {
	if p.level.n < sorcererFontOfMagicLevel {
		return sorcRefused(notGained("metamagic"))
	}
	var next uint16
	for _, e := range sel.effects {
		next |= 1 << e
	}
	dropped := 0
	for e := MetamagicCareful; e <= MetamagicTwinned; e++ {
		if s.knows(e) && next&(1<<e) == 0 {
			dropped++
		}
	}
	if dropped > 1 {
		return sorcRefused(refuse(SorcererMetamagicReplacementInvalid))
	}
	s.metamagic = next
	return sorcChanged(s, MetamagicOptionsChosen{id: id, options: s.Metamagic()})
}

// ---- Draconic Sorcery ---------------------------------------------------

// DraconicResilienceBonus returns the Hit Point maximum bonus of Draconic Resilience: 3 on gaining the feature at level 3 and 1
// more per further Sorcerer level (class-sorcerer.md FR-087; SRD 5.2.1, "Level 3: Draconic Resilience", p. 69).
func DraconicResilienceBonus() HitPointMaximumBonus {
	return MustHitPointMaximumBonus(3, 1, draconicResilienceLevel, SorcererClassID)
}

// DraconicArmorClassBasis returns the Armor Class basis of Draconic Resilience, 10 plus the Dexterity and Charisma modifiers, for
// the alternative bases of derived stats (class-sorcerer.md FR-088). It applies with no body armor worn and a Shield does not
// disable it.
func DraconicArmorClassBasis() AbilityBasedArmorClassBasis {
	return MustAbilityBasedArmorClassBasis("draconic_resilience", []ability.Ability{ability.Charisma}, true)
}

// ChooseElementalAffinity records the damage type of Elemental Affinity, Acid, Cold, Fire, Lightning or Poison (class-sorcerer.md
// FR-090). It refuses with SorcererFeatureNotGained for a Sorcerer who is not a Draconic Sorcerer of level 6, and
// SorcererElementalAffinityChoiceInvalid for a type outside the list or a second choice.
func ChooseElementalAffinity(id Identifier, s SorcererState, p SorcererProfile, t content.DamageType) SorcererTransition {
	switch {
	case !p.draconicAt(draconicAffinityLevel):
		return sorcRefused(notGained("elemental_affinity"))
	case s.affinity != 0 || !slices.Contains(ElementalAffinityTypes(), t):
		return sorcRefused(refuse(SorcererElementalAffinityChoiceInvalid))
	}
	s.affinity = t
	return sorcChanged(s, ElementalAffinityChosen{id: id, damageType: t})
}

// ApplyElementalAffinity adds the Charisma modifier to one damage roll of a spell that deals the chosen type (class-sorcerer.md
// FR-091, FR-104, FR-105). dealt is the damage types the spell actually deals after Transmuted Spell, whatever its class list or
// spellcasting ability; alreadyApplied says the bonus was already added to this spell. The bonus is in the event's Bonus. It
// refuses with SorcererFeatureNotGained for a Sorcerer who is not a Draconic Sorcerer of level 6, SorcererElementalAffinityTypeMismatch
// when no type is chosen or dealt does not include it, and SorcererElementalAffinityAlreadyApplied.
func ApplyElementalAffinity(id Identifier, s SorcererState, p SorcererProfile, charisma ability.Modifier, dealt []content.DamageType, alreadyApplied bool) SorcererTransition {
	switch {
	case !p.draconicAt(draconicAffinityLevel):
		return sorcRefused(notGained("elemental_affinity"))
	case s.affinity == 0 || !slices.Contains(dealt, s.affinity):
		return sorcRefused(refuse(SorcererElementalAffinityTypeMismatch))
	case alreadyApplied:
		return sorcRefused(refuse(SorcererElementalAffinityAlreadyApplied))
	}
	return sorcChanged(s, ElementalAffinityApplied{id: id, damageType: s.affinity, bonus: charisma.Value()})
}

// DragonWingsAction is what UseDragonWings does, a closed set. The zero value is not an action.
type DragonWingsAction int

const (
	// DragonWingsActivate causes the wings to appear (a Bonus Action) for 1 hour.
	DragonWingsActivate DragonWingsAction = iota + 1
	// DragonWingsDismiss dismisses the wings (no action).
	DragonWingsDismiss
	// DragonWingsRestoreUse spends 3 Sorcery Points to restore the use (no action).
	DragonWingsRestoreUse
)

// UseDragonWings activates, dismisses or restores the use of Dragon Wings (class-sorcerer.md FR-092 to FR-094). The 1 hour and the
// Fly Speed (SorcererEffects) are the host's to time. It returns an error matching srderr.ErrInvalidInput for an unknown action and
// refuses with SorcererFeatureNotGained for a Sorcerer who is not a Draconic Sorcerer of level 14, SorcererDragonWingsAlreadyUsed
// for an activation after the use, SorcererDragonWingsNotActive for a dismissal with the wings in, SorcererDragonWingsUseNotExpended
// for a restoration with the use unspent and SorcererSorceryPointsInsufficient with fewer than 3 points.
func UseDragonWings(id Identifier, s SorcererState, p SorcererProfile, action DragonWingsAction) (SorcererTransition, error) {
	if action < DragonWingsActivate || action > DragonWingsRestoreUse {
		return nil, invalidInput("UseDragonWings with an unknown action")
	}
	if !p.draconicAt(draconicWingsLevel) {
		return sorcRefused(notGained("dragon_wings")), nil
	}
	switch action {
	case DragonWingsRestoreUse: // restored below
	case DragonWingsActivate:
		if s.wingsUsed {
			return sorcRefused(refuse(SorcererDragonWingsAlreadyUsed)), nil
		}
		s.wingsUsed, s.wingsActive = true, true
		return sorcChanged(s, DragonWingsActivated{id: id}), nil
	case DragonWingsDismiss:
		if !s.wingsActive {
			return sorcRefused(refuse(SorcererDragonWingsNotActive)), nil
		}
		s.wingsActive = false
		return sorcChanged(s, DragonWingsEnded{id: id}), nil
	}
	switch {
	case !s.wingsUsed:
		return sorcRefused(refuse(SorcererDragonWingsUseNotExpended)), nil
	case s.points < dragonWingsRestorePointCost:
		return sorcRefused(SorcererRejection{Kind: SorcererSorceryPointsInsufficient, Required: dragonWingsRestorePointCost, Available: s.points}), nil
	}
	s.points -= dragonWingsRestorePointCost
	s.wingsUsed = false
	return sorcChanged(s, DragonWingsUseRestored{id: id}), nil
}

// UseDragonCompanion records the cast of Summon Dragon without a spell slot (class-sorcerer.md FR-095, FR-096); the host casts
// with CastModeFeatureGranted and no Material component. waiveConcentration says the casting does not require Concentration, its
// duration then 1 minute. It refuses with SorcererFeatureNotGained for a Sorcerer who is not a Draconic Sorcerer of level 18 and
// SorcererDragonCompanionFreeCastUsed after the free cast.
func UseDragonCompanion(id Identifier, s SorcererState, p SorcererProfile, waiveConcentration bool) SorcererTransition {
	switch {
	case !p.draconicAt(draconicCompanionLevel):
		return sorcRefused(notGained("dragon_companion"))
	case s.companionUsed:
		return sorcRefused(refuse(SorcererDragonCompanionFreeCastUsed))
	}
	s.companionUsed = true
	return sorcChanged(s, DragonCompanionCast{id: id, waivedConcentration: waiveConcentration})
}

// ---- Level grants -------------------------------------------------------

// SorcererLevelGrants are the grants of the Sorcerer level just gained for other services to apply (class-sorcerer.md FR-134).
// Read it; SorcererLevelGrantsFor builds it.
type SorcererLevelGrants struct {
	featGrants     []content.FeatCategory
	alwaysPrepared []content.SpellID
	hitPointBonus  HitPointMaximumBonus
	hasBonus       bool
}

// FeatGrants returns the feat categories granted at the level (General at 4, 8, 12 and 16, Epic Boon at 19), as a new slice.
func (g SorcererLevelGrants) FeatGrants() []content.FeatCategory { return slices.Clone(g.featGrants) }

// AlwaysPreparedSpells returns the spells the Draconic Spells feature newly grants at exactly the level, as a new slice.
func (g SorcererLevelGrants) AlwaysPreparedSpells() []content.SpellID {
	return slices.Clone(g.alwaysPrepared)
}

// HitPointBonus returns the Draconic Resilience Hit Point maximum bonus and true for a Draconic Sorcerer of level 3 or more.
func (g SorcererLevelGrants) HitPointBonus() (HitPointMaximumBonus, bool) {
	return g.hitPointBonus, g.hasBonus
}

// SorcererLevelGrantsFor returns the grants of the profile's level from the class definition and, for a Draconic Sorcerer, the
// subclass definition found in it (class-sorcerer.md FR-085 to FR-087, FR-089, FR-134). It returns an error matching
// srderr.ErrInvalidInput when the definition cannot give the feat grants of the level.
func SorcererLevelGrantsFor(p SorcererProfile, def content.ClassDefinition) (SorcererLevelGrants, error) {
	feats, err := def.FeatGrantsAt(p.level.n)
	if err != nil {
		return SorcererLevelGrants{}, firstError(err, invalidInput("Sorcerer feat grants"))
	}
	g := SorcererLevelGrants{featGrants: feats, alwaysPrepared: []content.SpellID{}}
	if p.draconicAt(draconicResilienceLevel) {
		g.hitPointBonus, g.hasBonus = DraconicResilienceBonus(), true
	}
	if found, ok := def.Subclass(p.subclass).(content.SubclassFound); ok {
		for _, f := range found.Subclass().Features() {
			for _, fact := range f.Facts() {
				if a, ok := fact.(content.AlwaysPreparedSpells); ok {
					for _, grant := range a.Grants() {
						if grant.Level() == p.level.n {
							g.alwaysPrepared = append(g.alwaysPrepared, grant.SpellIDs()...)
						}
					}
				}
			}
		}
	}
	return g, nil
}
