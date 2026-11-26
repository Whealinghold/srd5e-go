package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func athletics() combat.AssistedProficiency { return combat.AssistedSkill(content.SkillAthletics) }
func stealth() combat.AssistedProficiency   { return combat.AssistedSkill(content.SkillStealth) }
func thievesTools() combat.AssistedProficiency {
	return combat.AssistedTool(content.MustItemID("thieves_tools"))
}

// checkHelp builds ability check help from helper to ally.
func checkHelp(tb testing.TB, helper, ally string, p combat.AssistedProficiency) combat.AbilityCheckHelp {
	tb.Helper()
	b, err := combat.NewAbilityCheckHelp(gid(helper), gid(ally), p)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

// rollHelp builds attack roll help from helper against enemy.
func rollHelp(tb testing.TB, helper, enemy string) combat.AttackRollHelp {
	tb.Helper()
	b, err := combat.NewAttackRollHelp(gid(helper), gid(enemy))
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

// benefitsOf restores the pending benefits.
func benefitsOf(tb testing.TB, list ...combat.HelpBenefit) combat.HelpBenefits {
	tb.Helper()
	h, err := combat.NewHelpBenefits(list)
	if err != nil {
		tb.Fatal(err)
	}
	return h
}

// proficienciesOf builds the helper's proficiencies from skills and tools.
func proficienciesOf(tb testing.TB, skills []content.Skill, tools []content.ItemID) combat.HelpGiverProficiencies {
	tb.Helper()
	p, err := combat.NewHelpGiverProficiencies(skills, tools)
	if err != nil {
		tb.Fatal(err)
	}
	return p
}

// alliedOf builds the allied helper set.
func alliedOf(tb testing.TB, ids ...string) combat.AlliedHelpers {
	tb.Helper()
	var cs []combat.CombatantID
	for _, id := range ids {
		cs = append(cs, gid(id))
	}
	a, err := combat.NewAlliedHelpers(cs...)
	if err != nil {
		tb.Fatal(err)
	}
	return a
}

// helpKinds returns the Kind of each event, in order.
func helpKinds(events []combat.HelpEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}
