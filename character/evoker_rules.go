package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The Evoker subclass and its five features (SRD 5.2.1, "Wizard Subclass: Evoker", p. 82).
// Their levels come from the Evoker content.SubclassDefinition, never from here.
const (
	EvokerSubclassID        content.SubclassID = "evoker"
	EvocationSavantID       content.FeatureID  = "evoker_evocation_savant"
	PotentCantripID         content.FeatureID  = "evoker_potent_cantrip"
	SculptSpellsID          content.FeatureID  = "evoker_sculpt_spells"
	EmpoweredEvocationID    content.FeatureID  = "evoker_empowered_evocation"
	OverchannelID           content.FeatureID  = "evoker_overchannel"
	evocationSavantSpells                      = 2 // spells added at once by Evocation Savant
	evocationSavantMaxLevel                    = 2 // the highest level of those spells
	evokerSavantLevel                          = 3 // Evocation Savant's level is a reading, not a new-level addition (spec section 9)
	overchannelMinSlot                         = 1
	overchannelMaxSlot                         = 5
)

// EvokerGate says whether a Wizard holds an Evoker feature. The zero value is not a gate.
type EvokerGate int

// The gates. Only EvokerGateHeld lets a feature work.
const (
	// EvokerGateHeld: the subclass is the Evoker and the feature is gained at the Wizard level.
	EvokerGateHeld EvokerGate = iota + 1
	// EvokerGateNotEvoker: no subclass, or one that is not the Evoker.
	EvokerGateNotEvoker
	// EvokerGateNotGained: the Evoker, but the feature is above the Wizard level.
	EvokerGateNotGained
)

// gateKind maps a gate to the shared first two rejection causes: 0 held, 1 NotEvoker,
// 2 FeatureNotAvailable. A zero or unknown gate is NotEvoker, never held.
func gateKind(g EvokerGate) int {
	switch g {
	case EvokerGateHeld:
		return 0
	case EvokerGateNotGained:
		return 2
	case EvokerGateNotEvoker:
		return 1
	}
	return 1
}

// EvokerRules is the stateless rule maths of the Evoker features. The zero value is ready to use.
type EvokerRules struct{}

// Gate decides whether the Wizard of the given level holds feature through sub, which is
// the zero SubclassDefinition when the character has none. The features held are
// sub.FeaturesAtOrBelow(wizardLevel) (class-wizard.md FR-106); a level outside 1 to 20 holds
// none (ClassLevels never gives one to a use case).
func (EvokerRules) Gate(sub content.SubclassDefinition, wizardLevel int, feature content.FeatureID) EvokerGate {
	if sub.ID() != EvokerSubclassID {
		return EvokerGateNotEvoker
	}
	held, _ := sub.FeaturesAtOrBelow(wizardLevel)
	if slices.ContainsFunc(held, func(e content.FeatureEntry) bool { return e.ID() == feature }) {
		return EvokerGateHeld
	}
	return EvokerGateNotGained
}

// ---- spellbook additions ------------------------------------------------

// evokerSpellbookKind maps the cause of the shared spell checks to the Evoker one.
func evokerSpellbookKind(k SpellbookRejectionKind) EvokerSpellbookRejectionKind {
	switch k {
	case SpellbookWrongSpellCount:
		return EvokerSpellbookWrongSpellCount
	case SpellbookDuplicateSpell:
		return EvokerSpellbookDuplicateSpell
	case SpellbookIsACantrip:
		return EvokerSpellbookIsACantrip
	case SpellbookNotOnWizardList:
		return EvokerSpellbookNotOnWizardList
	case SpellbookAboveHighestSpellLevel:
		return EvokerSpellbookAboveHighestSpellLevel
	case SpellbookAlreadyInSpellbook, SpellbookFeatureNotAvailable, SpellbookNotInSourceSpellbook:
		// the last two are never returned by checkSpells
	}
	return EvokerSpellbookAlreadyInSpellbook
}

// maxSpellLevel is the ceiling used while only the order-first checks of checkSpells are wanted.
const maxSpellLevel = 9

// checkEvocationSpells validates facts in the order of EvokerSpellbookRejectionKind from
// the count on: the count (when count is positive), repeats, cantrips, the Wizard list,
// the Evocation school, the highest Spell Level and spells already in book, naming the
// first offender in command order.
func checkEvocationSpells(facts []WizardSpellFacts, book Spellbook, highest, count int) EvokerSpellbookRejection {
	if rej := checkSpells(facts, Spellbook{}, maxSpellLevel, count); rej.Kind != 0 {
		return EvokerSpellbookRejection{Kind: evokerSpellbookKind(rej.Kind), Allowed: rej.Allowed, Spell: rej.Spell}
	}
	for _, f := range facts {
		if f.School != content.SchoolEvocation {
			return EvokerSpellbookRejection{Kind: EvokerSpellbookNotAnEvocationSpell, Spell: f.ID}
		}
	}
	if rej := checkSpells(facts, book, highest, 0); rej.Kind != 0 {
		return EvokerSpellbookRejection{Kind: evokerSpellbookKind(rej.Kind), Highest: rej.Highest, Spell: rej.Spell}
	}
	return EvokerSpellbookRejection{}
}

// AddEvocationSavant adds the two Evocation Wizard spells of level 1 or 2 that Evocation
// Savant gives at the subclass level, free of cost (SRD 5.2.1, "Level 3: Evocation Savant",
// p. 82). A rejection has a non-zero Kind and changes nothing.
func (EvokerRules) AddEvocationSavant(id Identifier, gate EvokerGate, book Spellbook, spells []WizardSpellFacts) (SpellbookChanged, EvokerSpellbookRejection) {
	if k := gateKind(gate); k != 0 {
		return SpellbookChanged{}, EvokerSpellbookRejection{Kind: EvokerSpellbookRejectionKind(k)}
	}
	if rej := checkEvocationSpells(spells, book, evocationSavantMaxLevel, evocationSavantSpells); rej.Kind != 0 {
		if rej.Kind == EvokerSpellbookAboveHighestSpellLevel {
			rej.Highest = evocationSavantMaxLevel
		}
		return SpellbookChanged{}, rej
	}
	ids := wizardFactIDs(spells)
	return SpellbookChanged{book: NewSpellbook(append(book.Spells(), ids...)), events: added(id, SpellbookEvocationSavant, ids)}, EvokerSpellbookRejection{}
}

// AddEvokerFree adds the one Evocation Wizard spell the Evoker may add free of cost on
// gaining access to a new level of Spell Slots (SRD 5.2.1, "Level 3: Evocation Savant", p. 82).
// newSpellLevel is WizardRules.NewSpellLevelAt for the Wizard level; the subclass level
// itself gets only the two spells of AddEvocationSavant (spec section 9). A rejection has a
// non-zero Kind and changes nothing.
func (EvokerRules) AddEvokerFree(id Identifier, gate EvokerGate, wizardLevel int, newSpellLevel bool, highest int, book Spellbook, spell WizardSpellFacts) (SpellbookChanged, EvokerSpellbookRejection) {
	if k := gateKind(gate); k != 0 {
		return SpellbookChanged{}, EvokerSpellbookRejection{Kind: EvokerSpellbookRejectionKind(k)}
	}
	if !newSpellLevel || wizardLevel <= evokerSavantLevel {
		return SpellbookChanged{}, EvokerSpellbookRejection{Kind: EvokerSpellbookNoNewSpellLevel}
	}
	if rej := checkEvocationSpells([]WizardSpellFacts{spell}, book, highest, 0); rej.Kind != 0 {
		return SpellbookChanged{}, rej
	}
	return SpellbookChanged{book: NewSpellbook(append(book.Spells(), spell.ID)), events: added(id, SpellbookEvokerLevel, []content.SpellID{spell.ID})}, EvokerSpellbookRejection{}
}

// ---- Potent Cantrip -----------------------------------------------------

// PotentCantripOutcome is the closed outcome of Potent Cantrip. Variants:
// PotentCantripHalfDamage, PotentCantripUnchanged.
type PotentCantripOutcome interface{ isPotentCantripOutcome() }

// PotentCantripHalfDamage: the target takes half the cantrip's damage, rounded down, and
// no additional effect.
type PotentCantripHalfDamage struct{ damage int }

// Damage returns the halved damage, rounded down (SRD 5.2.1, "Round Down", p. 5).
func (o PotentCantripHalfDamage) Damage() int { return o.damage }

// PotentCantripUnchanged: the attack hit or the save failed, so the ordinary full effect applies.
type PotentCantripUnchanged struct{}

func (PotentCantripHalfDamage) isPotentCantripOutcome() {}
func (PotentCantripUnchanged) isPotentCantripOutcome()  {}

// PotentCantrip decides Potent Cantrip (SRD 5.2.1, "Level 3: Potent Cantrip", p. 82) for a
// cantrip cast at a creature: a missed attack roll (a Failure of an AttackRoll) or a
// successful saving throw (a Success of a SavingThrow) gives half of damage, rounded down;
// a hit or a failed save leaves the effect unchanged, and is accepted. A rejection has a
// non-zero Kind. damage must be 0 or more; the caller checks that.
func (EvokerRules) PotentCantrip(gate EvokerGate, spell WizardSpellFacts, kind d20.Kind, outcome d20.Outcome, damage int) (PotentCantripOutcome, PotentCantripRejection) {
	switch {
	case gateKind(gate) != 0:
		return nil, PotentCantripRejection{Kind: PotentCantripRejectionKind(gateKind(gate))}
	case kind != d20.AttackRoll && kind != d20.SavingThrow:
		return nil, PotentCantripRejection{Kind: PotentCantripUnsupportedTestKind, TestKind: kind}
	case !spell.IsCantrip():
		return nil, PotentCantripRejection{Kind: PotentCantripNotACantrip}
	case !spell.DealsDamage:
		return nil, PotentCantripRejection{Kind: PotentCantripSpellDealsNoDamage}
	}
	if (kind == d20.AttackRoll && outcome == d20.Failure) || (kind == d20.SavingThrow && outcome == d20.Success) {
		return PotentCantripHalfDamage{damage: damage / 2}, PotentCantripRejection{}
	}
	return PotentCantripUnchanged{}, PotentCantripRejection{}
}

// ---- Sculpt Spells ------------------------------------------------------

// SculptCandidate is one creature the caster may protect, with the facts the caller
// supplies. Construct it with a keyed literal.
type SculptCandidate struct {
	ID       Identifier
	Visible  bool // the caster can see it
	Affected bool // the spell affects it
	IsCaster bool
}

// SculptedCreatures is the accepted outcome of Sculpt Spells: the creatures that
// automatically succeed on their saving throws and take no damage on a success.
type SculptedCreatures struct {
	creatures []Identifier
	allowed   int
}

// Creatures returns a new slice of the protected creatures, in the order chosen.
func (s SculptedCreatures) Creatures() []Identifier { return slices.Clone(s.creatures) }

// Allowed returns the most creatures that could be chosen: 1 plus the level the spell is cast at.
func (s SculptedCreatures) Allowed() int { return s.allowed }

// SculptSpells decides Sculpt Spells (SRD 5.2.1, "Level 6: Sculpt Spells", p. 82) for an
// Evocation spell cast at castLevel (the slot level; 0 for a cantrip). It reports the first
// cause in the order of SculptRejectionKind, and the first offending creature in
// command order. A rejection has a non-zero Kind. castLevel must be valid for the spell;
// the use case checks that.
func (EvokerRules) SculptSpells(gate EvokerGate, spell WizardSpellFacts, castLevel int, chosen []SculptCandidate) (SculptedCreatures, SculptRejection) {
	if k := gateKind(gate); k != 0 {
		return SculptedCreatures{}, SculptRejection{Kind: SculptRejectionKind(k)}
	}
	if spell.School != content.SchoolEvocation {
		return SculptedCreatures{}, SculptRejection{Kind: SculptNotAnEvocationSpell}
	}
	for _, c := range chosen {
		if c.IsCaster {
			return SculptedCreatures{}, SculptRejection{Kind: SculptCaster, Creature: c.ID}
		}
	}
	for _, c := range chosen {
		if !c.Visible || !c.Affected {
			return SculptedCreatures{}, SculptRejection{Kind: SculptCreatureNotEligible, Creature: c.ID}
		}
	}
	allowed := 1 + castLevel
	if len(chosen) > allowed {
		return SculptedCreatures{}, SculptRejection{Kind: SculptTooManyCreatures, Allowed: allowed}
	}
	ids := make([]Identifier, len(chosen))
	for i, c := range chosen {
		ids[i] = c.ID
	}
	return SculptedCreatures{creatures: ids, allowed: allowed}, SculptRejection{}
}

// ---- Empowered Evocation ------------------------------------------------

// EmpoweredCastState records, for one cast, whether a damage roll of the spell is already
// empowered. The zero value is a cast with none. Comparable.
type EmpoweredCastState struct{ empowered bool }

// NewEmpoweredCastState returns the state of a cast whose damage roll is empowered or not.
func NewEmpoweredCastState(damageRollEmpowered bool) EmpoweredCastState {
	return EmpoweredCastState{empowered: damageRollEmpowered}
}

// DamageRollEmpowered reports whether a damage roll of the cast is already empowered.
func (s EmpoweredCastState) DamageRollEmpowered() bool { return s.empowered }

// EmpoweredRoll is the accepted outcome of Empowered Evocation: the modifier to add to one
// damage roll, if the caster chose to, and the cast state to carry on with.
type EmpoweredRoll struct {
	modifier ability.Modifier
	applied  bool
	state    EmpoweredCastState
}

// Modifier returns the Intelligence modifier to add to one damage roll and true, or false
// when the caster chose not to apply the feature. A negative modifier is added as it is.
func (r EmpoweredRoll) Modifier() (ability.Modifier, bool) { return r.modifier, r.applied }

// State returns the cast state after the decision.
func (r EmpoweredRoll) State() EmpoweredCastState { return r.state }

// EmpoweredEvocation decides Empowered Evocation (SRD 5.2.1, "Level 10: Empowered
// Evocation", p. 82). The feature is optional: with applies false it returns no modifier and
// the state unchanged, and a cast already empowered is not a conflict. A rejection has a
// non-zero Kind.
func (EvokerRules) EmpoweredEvocation(gate EvokerGate, spell WizardSpellFacts, intelligence ability.Modifier, applies bool, state EmpoweredCastState) (EmpoweredRoll, EmpoweredRejection) {
	switch {
	case gateKind(gate) != 0:
		return EmpoweredRoll{}, EmpoweredRejection{Kind: EmpoweredRejectionKind(gateKind(gate))}
	case !spell.OnWizardList():
		return EmpoweredRoll{}, EmpoweredRejection{Kind: EmpoweredNotAWizardSpell}
	case spell.School != content.SchoolEvocation:
		return EmpoweredRoll{}, EmpoweredRejection{Kind: EmpoweredNotAnEvocationSpell}
	case !applies:
		return EmpoweredRoll{state: state}, EmpoweredRejection{}
	case state.empowered:
		return EmpoweredRoll{}, EmpoweredRejection{Kind: EmpoweredAlreadyEmpowered}
	}
	return EmpoweredRoll{modifier: intelligence, applied: true, state: EmpoweredCastState{empowered: true}}, EmpoweredRejection{}
}

// ---- Overchannel --------------------------------------------------------

// OverchannelRequest is the cast Overchannel decides on. Construct it with a keyed literal.
type OverchannelRequest struct {
	Spell     WizardSpellFacts
	Source    CastSource // must be CastSourceSlot
	SlotLevel int        // the level of the slot expended
}

// OverchannelPlan is what an accepted Overchannel use comes to, before the Necrotic
// damage is rolled.
type OverchannelPlan struct {
	use, slot int
}

// Use returns the use number since the last Long Rest (1 is the first).
func (p OverchannelPlan) Use() int { return p.use }

// SlotLevel returns the level of the slot expended.
func (p OverchannelPlan) SlotLevel() int { return p.slot }

// NecroticDice returns the number of d12 of Necrotic damage: none for the first use, then
// the use number times the slot level (2d12 per slot level on the second use, 3d12 on the third).
func (p OverchannelPlan) NecroticDice() int {
	if p.use < 2 {
		return 0
	}
	return p.use * p.slot
}

// Necrotic returns the Necrotic damage expression and true, or false for the first use. It
// returns an error matching srderr.ErrInvalidInput when the use number is so high that the
// dice exceed dice.MaxCount.
func (p OverchannelPlan) Necrotic() (dice.Expression, bool, error) {
	n := p.NecroticDice()
	if n == 0 {
		return dice.Expression{}, false, nil
	}
	e, err := dice.NewExpression(n, dice.D12(), 0)
	if err != nil {
		return dice.Expression{}, false, invalidInput("Overchannel Necrotic damage with too many dice")
	}
	return e, true, nil
}

// Overchannel decides one use of Overchannel (SRD 5.2.1, "Level 14: Overchannel", p. 82)
// given the uses since the last Long Rest. A rejection has a non-zero Kind and is not a use.
func (EvokerRules) Overchannel(gate EvokerGate, req OverchannelRequest, uses int) (OverchannelPlan, OverchannelRejection) {
	switch {
	case gateKind(gate) != 0:
		return OverchannelPlan{}, OverchannelRejection{Kind: OverchannelRejectionKind(gateKind(gate))}
	case !req.Spell.OnWizardList():
		return OverchannelPlan{}, OverchannelRejection{Kind: OverchannelNotAWizardSpell}
	case req.Source != CastSourceSlot:
		return OverchannelPlan{}, OverchannelRejection{Kind: OverchannelNoSpellSlotUsed}
	case req.SlotLevel < overchannelMinSlot || req.SlotLevel > overchannelMaxSlot:
		return OverchannelPlan{}, OverchannelRejection{Kind: OverchannelSlotLevelOutOfRange, Level: req.SlotLevel}
	case !req.Spell.DealsDamage:
		return OverchannelPlan{}, OverchannelRejection{Kind: OverchannelSpellDealsNoDamage}
	}
	return OverchannelPlan{use: uses + 1, slot: req.SlotLevel}, OverchannelRejection{}
}
