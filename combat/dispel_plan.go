package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
)

// DispelBlockReason says why Dispel Magic cannot end a spell without a check. The
// zero value is not a reason.
type DispelBlockReason int

const (
	// DispelBlockNeverDispellable: the spell "can't be dispelled" (SRD 5.2.1, p. 5).
	DispelBlockNeverDispellable DispelBlockReason = iota + 1
	// DispelBlockSlotLevelTooLow: the cast slot is below the level the spell needs (SRD 5.2.1, p. 142).
	DispelBlockSlotLevelTooLow
	// DispelBlockPartsRemain: a part the spell waits for is still on the target (SRD 5.2.1, p. 155).
	DispelBlockPartsRemain
)

// String returns "never_dispellable", "slot_level_too_low", "parts_remain" or "unknown".
func (r DispelBlockReason) String() string {
	return enumName([]string{"never_dispellable", "slot_level_too_low", "parts_remain"}, int(r))
}

// DispelStep is Dispel Magic's decision for one ongoing spell on the target. It is
// a closed set: DispelEnds, DispelNeedsCheck and DispelBlocked. Every step carries
// its spell.
type DispelStep interface {
	isDispelStep()
	// Spell returns the spell the step is about.
	Spell() OngoingSpell
}

// DispelEnds is a step that ends the spell without a check: the spell's level is
// 3 or lower, or no higher than the slot (SRD 5.2.1, "Dispel Magic", p. 124).
type DispelEnds struct{ spell OngoingSpell }

// DispelNeedsCheck is a step that needs an ability check: the spell is above the
// slot, and the check's DC is 10 plus the spell's level (SRD 5.2.1, p. 124). The
// check uses AbilityModifier and Modifiers; it adds no Proficiency Bonus.
type DispelNeedsCheck struct {
	spell           OngoingSpell
	dc              d20.DifficultyClass
	abilityModifier ability.Modifier
	modifiers       d20.RollModifiers
}

// DispelBlocked is a step that cannot end the spell: a restriction forbids it
// (Reason says which). A blocked spell is never given a check.
type DispelBlocked struct {
	spell  OngoingSpell
	reason DispelBlockReason
}

func (DispelEnds) isDispelStep()       {}
func (DispelNeedsCheck) isDispelStep() {}
func (DispelBlocked) isDispelStep()    {}

// Spell returns the spell.
func (s DispelEnds) Spell() OngoingSpell { return s.spell }

// Spell returns the spell.
func (s DispelNeedsCheck) Spell() OngoingSpell { return s.spell }

// Spell returns the spell.
func (s DispelBlocked) Spell() OngoingSpell { return s.spell }

// DC returns the difficulty class of the check.
func (s DispelNeedsCheck) DC() d20.DifficultyClass { return s.dc }

// AbilityModifier returns the spellcasting ability modifier the check adds.
func (s DispelNeedsCheck) AbilityModifier() ability.Modifier { return s.abilityModifier }

// Modifiers returns the other roll modifiers the check applies.
func (s DispelNeedsCheck) Modifiers() d20.RollModifiers { return s.modifiers }

// Reason returns why the spell is blocked.
func (s DispelBlocked) Reason() DispelBlockReason { return s.reason }

// DispelMagicPlan is the plan Dispel Magic makes for one target: one step per
// spell on it, in registry order. It stores the target and the ids of its steps,
// not the registry; Conclude checks the registry it is given still holds them
// (FR-059). The zero value is an empty plan with no target.
type DispelMagicPlan struct {
	target DispelTarget
	steps  []DispelStep
}

// Target returns the target the plan is for.
func (p DispelMagicPlan) Target() DispelTarget { return p.target }

// Steps returns a copy of the steps in plan order.
func (p DispelMagicPlan) Steps() []DispelStep {
	out := make([]DispelStep, len(p.steps))
	copy(out, p.steps)
	return out
}

// ids returns the effect ids of the steps, in plan order.
func (p DispelMagicPlan) ids() []OngoingEffectID {
	out := make([]OngoingEffectID, 0, len(p.steps))
	for _, s := range p.steps {
		out = append(out, s.Spell().Effect().ID())
	}
	return out
}

// DispelMagic plans and concludes Dispel Magic (SRD 5.2.1, p. 124). It is
// stateless: use the zero value. Plan decides each spell; the host rolls one
// ability check per DispelNeedsCheck through the dice port and the D20 resolver;
// Conclude applies the checks.
type DispelMagic struct{}

// Plan decides every spell on target in registry order (dispel-and-counterspell.md
// section 4.2). castLevel is the slot the Dispel Magic was cast with; abilityModifier
// is the spellcasting ability modifier; modifiers are any other roll modifiers.
// It returns an error matching srderr.ErrInvalidInput when castLevel is below 3
// (Dispel Magic is a level 3 spell) or target has a zero id. It never panics.
func (DispelMagic) Plan(registry OngoingSpellRegistry, target DispelTarget, castLevel character.SlotLevel, abilityModifier ability.Modifier, modifiers d20.RollModifiers) (DispelMagicPlan, error) {
	if castLevel.Value() < 3 {
		return DispelMagicPlan{}, invalidInput("dispel magic is cast with a slot of level 3 or higher")
	}
	if !validDispelTarget(target) {
		return DispelMagicPlan{}, invalidInput("a dispel needs a target with an id")
	}
	spells := registry.SpellsOn(target)
	steps := make([]DispelStep, 0, len(spells))
	for _, s := range spells {
		steps = append(steps, planSpell(s, registry, target, castLevel, abilityModifier, modifiers))
	}
	return DispelMagicPlan{target: target, steps: steps}, nil
}

// planSpell decides one spell: the restriction first, then the level against the slot.
func planSpell(s OngoingSpell, registry OngoingSpellRegistry, target DispelTarget, castLevel character.SlotLevel, abilityModifier ability.Modifier, modifiers d20.RollModifiers) DispelStep {
	switch r := s.Restriction().(type) {
	case DispelNeverDispellable:
		return DispelBlocked{spell: s, reason: DispelBlockNeverDispellable}
	case DispelRequiresSlotLevel:
		if castLevel.Value() < r.Level().Value() {
			return DispelBlocked{spell: s, reason: DispelBlockSlotLevelTooLow}
		}
	case DispelAfterPartsGone:
		for _, id := range r.IDs() {
			if registry.holdsOn(target, id) {
				return DispelBlocked{spell: s, reason: DispelBlockPartsRemain}
			}
		}
	case DispelUnrestricted:
	default:
		panic(unreachableError("dispel restriction", r))
	}
	level := s.CastAtLevel().Int()
	if level <= 3 || level <= castLevel.Value() {
		return DispelEnds{spell: s}
	}
	return DispelNeedsCheck{
		spell:           s,
		dc:              d20.DifficultyClass(10 + level),
		abilityModifier: abilityModifier,
		modifiers:       modifiers,
	}
}
