package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const statBlockSpec = "doc/specs/monster-stat-block-actions.md "

// The stat blocks of these tests are synthetic: made-up parts carrying the numbers and the shape the spec's
// case tables quote from the stat blocks of SRD 5.2.1, pp. 258-316, never a stat block.

// sbPart builds an unlimited part with the given fields.
func sbPart(name string, f content.StatBlockPartFields) content.StatBlockPart {
	if f.Usage == nil {
		f.Usage = content.PartUnlimited{}
	}
	if f.Text == "" {
		f.Text = "T."
	}
	return luMust(content.NewStatBlockPart(name, f))
}

// sbPlain builds an unlimited part with no notation.
func sbPlain(names ...string) []content.StatBlockPart {
	var out []content.StatBlockPart
	for _, n := range names {
		out = append(out, sbPart(n, content.StatBlockPartFields{}))
	}
	return out
}

func sbGroup(count int, parts ...string) content.AttackSlotGroup {
	return luMust(content.NewAttackSlotGroup(count, parts))
}

func sbExtra(options ...string) content.AdditionalUse {
	return luMust(content.NewAdditionalUse(options))
}

func sbOption(part string) content.ReplacementOption {
	return luMust(content.NewReplacementOption(part, "", 0))
}

func sbSpellOption(part, spell string, level int) content.ReplacementOption {
	return luMust(content.NewReplacementOption(part, spell, level))
}

func sbOnlyPart(name string) content.ReplacementScope {
	return luMust(content.NewReplacementScopeOneAttackPart(name))
}

func sbRule(count int, scope content.ReplacementScope, limit content.ReplacementLimit, options ...content.ReplacementOption) content.ReplacementRule {
	return luMust(content.NewReplacementRule(count, scope, options, limit))
}

func sbVariant(requires string, groups []content.AttackSlotGroup, extras []content.AdditionalUse, rules ...content.ReplacementRule) content.MultiattackVariant {
	return luMust(content.NewMultiattackVariant(requires, groups, extras, rules))
}

func sbGroups(g ...content.AttackSlotGroup) []content.AttackSlotGroup { return g }

func sbExtras(e ...content.AdditionalUse) []content.AdditionalUse { return e }

// sbMultiattack is the Multiattack part with the plan of the variants.
func sbMultiattack(variants ...content.MultiattackVariant) content.StatBlockPart {
	plan := luMust(content.NewMultiattackPlan(variants))
	return sbPart(content.MultiattackName, content.StatBlockPartFields{Plan: plan})
}

// sbMonster builds a monster whose Actions are the Multiattack first (when given) then the parts, and whose Bonus
// Actions, Reactions and Traits are the other arguments.
func sbMonster(tb testing.TB, multiattack *content.StatBlockPart, actions []content.StatBlockPart, edit func(*content.MonsterFields)) content.Monster {
	tb.Helper()
	return luMonster(tb, func(f *content.MonsterFields) {
		if multiattack != nil {
			f.Actions = append([]content.StatBlockPart{*multiattack}, actions...)
		} else {
			f.Actions = actions
		}
		if edit != nil {
			edit(f)
		}
	})
}

// sbRef returns the reference of the part called name in the section.
func sbRef(tb testing.TB, m content.Monster, s combat.MonsterPartSection, name string) combat.LimitedUsePartRef {
	tb.Helper()
	var parts []content.StatBlockPart
	switch s {
	case combat.MonsterPartTraits:
		parts = m.Traits()
	case combat.MonsterPartActions:
		parts = m.Actions()
	case combat.MonsterPartBonusActions:
		parts = m.BonusActions()
	case combat.MonsterPartReactions:
		parts = m.Reactions()
	case combat.MonsterPartLegendaryActions:
		l, _ := m.LegendaryActions()
		parts = l.Actions()
	}
	for i, p := range parts {
		if p.Name() == name {
			return luRef(tb, s, i)
		}
	}
	tb.Fatalf("no part %q in %v", name, s)
	return combat.LimitedUsePartRef{}
}

// sbAction is sbRef in the Actions.
func sbAction(tb testing.TB, m content.Monster, name string) combat.LimitedUsePartRef {
	tb.Helper()
	return sbRef(tb, m, combat.MonsterPartActions, name)
}

// sbAttackNotation builds an Attack Notation.
func sbAttackNotation(kind content.StatBlockAttackKind, bonus int, qualifier string, reach int, rng content.StatBlockAttackRange) content.StatBlockAttack {
	return luMust(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: kind, Bonus: bonus, Qualifier: qualifier, Reach: reach, HasReach: reach != 0, Range: rng}))
}

func sbRange(normal, long int) content.StatBlockAttackRange {
	return luMust(content.NewStatBlockAttackRange(normal, long))
}

// sbDamage builds a damage instance of the printed number, the dice (the zero expression for none) and the type.
func sbDamage(number int, e dice.Expression, t content.DamageType) content.StatBlockDamage {
	return luMust(content.NewStatBlockDamage(number, e, luMust(content.NewSpecificDamageType(t)), false))
}

func sbClause(label content.EffectClauseLabel, rider content.SaveSuccessRider, damage ...content.StatBlockDamage) content.EffectClause {
	return luMust(content.NewEffectClause(label, "Clause "+label.String()+".", rider, damage))
}

func sbSave(ab ability.Ability, dc int, target, prefix string) content.StatBlockSavingThrow {
	return luMust(content.NewStatBlockSavingThrow(ab, dc, target, prefix))
}

// sbAttackPart is a part with an Attack Notation and clauses.
func sbAttackPart(name string, atk content.StatBlockAttack, clauses ...content.EffectClause) content.StatBlockPart {
	return sbPart(name, content.StatBlockPartFields{Attack: atk, Clauses: clauses})
}

// sbSavePart is a part with a Saving Throw notation and clauses.
func sbSavePart(name string, st content.StatBlockSavingThrow, clauses ...content.EffectClause) content.StatBlockPart {
	return sbPart(name, content.StatBlockPartFields{SavingThrow: st, Clauses: clauses})
}

func sbExpr(count, sides, modifier int) dice.Expression {
	return dice.MustExpression(count, dice.MustDie(sides), modifier)
}

func sbSelf() combat.CombatantID { return combat.MustCombatantID(luSelf) }
