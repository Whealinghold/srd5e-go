package combat

import (
	"cmp"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// The per-call design caps of plan P10 for the monster collections (monster-limited-use, monster-combatant-setup,
// monster-stat-block-actions, monster-traits, monster-spellcasting and spell-effects). They are design numbers, not
// SRD rules.
const (
	// maxMonsterParts is the cap of the parts of one stat block (traits, actions, bonus actions, reactions and
	// Legendary Actions together).
	maxMonsterParts = 64
	// maxMonsterSpells is the cap of the spells of one stat block, over all its spell groups.
	maxMonsterSpells = 64
	// maxMonsterSpellGroups is the cap of the spell groups of one Spellcasting trait.
	maxMonsterSpellGroups = 64
	// maxPlanVariants and maxPlanGroups cap the variants and the groups of one stat block attack or save plan.
	maxPlanVariants = 64
	maxPlanGroups   = 64
	// maxRechargeRolls is the cap of the Recharge roll map of one StartOfOwnTurn call.
	maxRechargeRolls = 64
	// maxMultiattackSteps is the cap of the steps and of the replacements of one MultiattackProgress.
	maxMultiattackSteps = 64
	// maxStatBlockTargets is the cap of the targets of one cast or one stat block save.
	maxStatBlockTargets = 256
	// maxStatBlockDamages is the cap of the damage instances of one RollStatBlockDamage call.
	maxStatBlockDamages = 64
	// maxPackTacticsAllies is the cap of the allies of one PackTacticsCircumstances.
	maxPackTacticsAllies = 256
	// maxSiegeDamages is the cap of the damages of one ApplySiegeMonster call.
	maxSiegeDamages = 64
	// maxDeathBurstEvents is the cap of the events of one TriggerDeathBurst call.
	maxDeathBurstEvents = 256
	// maxTraitTextBytes is the cap of the text, in bytes, that RecogniseTraits reads.
	maxTraitTextBytes = 1024
	// maxSpellEffectTargets and maxSpellEffectAssignments cap the targets and the instance assignments of one
	// ResolveSpellEffects call.
	maxSpellEffectTargets     = 256
	maxSpellEffectAssignments = 256
	// maxSpellEffectPartIndexes and maxSpellEffectInstances cap PartIndexes and InstanceCount of one call.
	maxSpellEffectPartIndexes = 64
	maxSpellEffectInstances   = 100
)

// monsterCollectionsInRange returns a LimitExceededError when the stat block of m holds more than 64 parts, 64 spells
// or 64 spell groups (plan P10).
func monsterCollectionsInRange(m content.Monster) error {
	legendary, _ := m.LegendaryActions()
	parts := len(m.Traits()) + len(m.Actions()) + len(m.BonusActions()) + len(m.Reactions()) + len(legendary.Actions())
	if err := overCap("monster parts", parts, maxMonsterParts); err != nil {
		return err
	}
	sc, ok := m.Spellcasting()
	if !ok {
		return nil
	}
	if err := overCap("monster spell groups", len(sc.Groups()), maxMonsterSpellGroups); err != nil {
		return err
	}
	spells := 0
	for _, g := range sc.Groups() {
		spells += len(g.Spells())
	}
	return overCap("monster spells", spells, maxMonsterSpells)
}

// monsterPlansInRange returns a LimitExceededError when the Multiattack plans of m hold, over the whole stat block, more
// than 64 variants, 64 attack groups or 64 replacement rules (plan P10).
func monsterPlansInRange(m content.Monster) error {
	legendary, _ := m.LegendaryActions()
	var variants, groups, replacements int
	for _, p := range slices.Concat(m.Traits(), m.Actions(), m.BonusActions(), m.Reactions(), legendary.Actions()) {
		plan, ok := p.Plan()
		if !ok {
			continue
		}
		variants += len(plan.Variants())
		for _, v := range plan.Variants() {
			groups += len(v.Groups())
			replacements += len(v.Replacements())
		}
	}
	return cmp.Or(overCap("multiattack variants", variants, maxPlanVariants), overCap("multiattack groups", groups, maxPlanGroups),
		overCap("multiattack replacements", replacements, maxMultiattackSteps))
}

// spellEffectsCapsError returns an error matching ErrInvalidSpellEffectsCommand, carrying a LimitExceededError, when a
// ResolveSpellEffects command holds more than 256 targets, 256 assignments, 64 part indexes or an instance count above
// 100 (plan P10; spell-effects.md FR-062). It never panics, whatever the instance count.
func spellEffectsCapsError(cmd ResolveSpellEffectsCommand) error {
	if err := cmp.Or(overCap("spell effect targets", len(cmd.Targets), maxSpellEffectTargets),
		overCap("spell effect assignments", len(cmd.Assignments), maxSpellEffectAssignments),
		overCap("spell effect part indexes", len(cmd.PartIndexes), maxSpellEffectPartIndexes),
		overCap("spell effect instances", cmd.InstanceCount, maxSpellEffectInstances)); err != nil {
		return spellCommandError("a collection over its cap", err)
	}
	return nil
}
