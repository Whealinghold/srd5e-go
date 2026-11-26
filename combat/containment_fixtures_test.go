package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ctDamage builds one damage term from notation such as "3d6".
func ctDamage(notation string, kind content.DamageType) combat.DamageTerm {
	return combat.MustDamageTerm(dice.MustParseExpression(notation), kind)
}

func ctDamages(terms ...combat.DamageTerm) []combat.DamageTerm { return terms }

// ctCap builds a capacity of slots in which the Large creature costs large.
func ctCap(slots, large int) combat.ContainmentCapacity {
	if large == 1 {
		return combat.MustContainmentCapacity(slots, nil)
	}
	return combat.MustContainmentCapacity(slots, map[content.CreatureSize]int{content.SizeLarge: large})
}

func ctPeriodic(timing combat.ContainmentTiming, deferred bool, terms ...combat.DamageTerm) combat.PeriodicDamage {
	return combat.MustPeriodicDamage(terms, timing, deferred)
}

func ctBR() []content.ConditionKind {
	return []content.ConditionKind{content.ConditionBlinded, content.ConditionRestrained}
}

// ctTerms returns the terms of the stat block named name, as table 4.7.1 of monster-engulf-and-swallow.md
// states them: cube, mound, whelm, behir, worm, remorhaz, tarrasque, kraken, frog or toad.
func ctTerms(tb testing.TB, name string) combat.ContainmentTerms {
	tb.Helper()
	acid, fire := content.DamageAcid, content.DamageFire
	start, end := combat.ContainmentTimingStartOfContainerTurn, combat.ContainmentTimingEndOfContainerTurn
	in := combat.ContainmentTermsInput{Suffocation: combat.SuffocationRuleNone, DeathRule: combat.ReleaseOnDeath{},
		ActionKind: combat.ContainmentActionKindBonusAction, Imposed: ctBR()}
	swallow := func(slots int, dc d20.DifficultyClass, a ability.Ability, periodic combat.PeriodicDamage, reg combat.Regurgitation, corpse int) {
		in.Mode, in.MaxSize, in.Capacity = combat.ContainmentModeSwallowed, content.SizeLarge, ctCap(slots, 1)
		in.Entry = combat.MustEntryBySave(a, dc, nil, false, 0, false)
		in.TotalCover, in.Periodic, in.Regurgitation, in.DeathRule = true, periodic, reg, combat.MustEscapeFromCorpse(corpse)
	}
	switch name {
	case "cube":
		in.Mode, in.MaxSize, in.Capacity = combat.ContainmentModeEngulfed, content.SizeLarge, ctCap(4, 4)
		in.Entry = combat.MustEntryBySave(ability.Dexterity, 12, ctDamages(ctDamage("3d6", acid)), true, 5, true)
		in.Imposed, in.TotalCover, in.Suffocation, in.NoVerbalSpells = []content.ConditionKind{content.ConditionRestrained}, true, combat.SuffocationRuleAlways, true
		in.Periodic = ctPeriodic(start, false, ctDamage("3d6", acid))
		in.Exit = combat.MustExitTerms(12, 12, ctDamages(ctDamage("3d6", acid)))
		in.ActionKind = combat.ContainmentActionKindAction
	case "mound":
		in.Mode, in.MaxSize, in.Capacity = combat.ContainmentModeEngulfedGrappled, content.SizeMedium, ctCap(1, 1)
		in.Entry = combat.MustEntryBySave(ability.Strength, 15, nil, false, 0, false)
		in.Periodic = ctPeriodic(start, false, ctDamage("3d6", content.DamageLightning))
		in.Grapple = combat.MustGrappleHoldTerms(14, "Mound")
		in.ActionKind = combat.ContainmentActionKindAction
	case "whelm":
		in.Mode, in.MaxSize, in.Capacity = combat.ContainmentModeEngulfedGrappled, content.SizeLarge, ctCap(2, 2)
		in.Entry = combat.MustEntryBySave(ability.Strength, 15, ctDamages(ctDamage("4d8+4", content.DamageBludgeoning)), true, 0, true)
		in.Imposed, in.Suffocation = []content.ConditionKind{content.ConditionRestrained}, combat.SuffocationRuleUnlessBreathesWater
		in.Periodic = ctPeriodic(start, false, ctDamage("2d8", content.DamageBludgeoning))
		in.Exit = combat.MustExitTerms(0, 14, nil)
		in.Grapple = combat.MustGrappleHoldTerms(14, "Whelm")
		in.ActionKind = combat.ContainmentActionKindAction
	case "behir":
		swallow(1, 18, ability.Dexterity, ctPeriodic(start, false, ctDamage("6d6", acid)), combat.MustRegurgitation(30, 14, 10), 15)
	case "worm":
		swallow(3, 19, ability.Strength, ctPeriodic(start, false, ctDamage("5d6", acid)), combat.MustRegurgitation(30, 21, 5), 20)
	case "remorhaz":
		swallow(2, 19, ability.Strength, ctPeriodic(start, false, ctDamage("3d6", acid), ctDamage("3d6", fire)), combat.MustRegurgitation(30, 15, 5), 15)
	case "tarrasque":
		swallow(6, 27, ability.Strength, ctPeriodic(start, false, ctDamage("16d6", acid)), combat.MustRegurgitation(60, 20, 10), 20)
		in.NoTeleport = true
	case "kraken":
		swallow(4, 25, ability.Dexterity, ctPeriodic(start, false, ctDamage("7d6", acid)), combat.MustRegurgitation(50, 25, 10), 15)
		in.Entry = combat.MustEntryBySave(ability.Dexterity, 25, ctDamages(ctDamage("3d8+10", content.DamagePiercing)), false, 0, true)
		in.Imposed, in.ActionKind = []content.ConditionKind{content.ConditionRestrained}, combat.ContainmentActionKindAction
	case "frog":
		swallow(1, 1, ability.Strength, ctPeriodic(end, true, ctDamage("2d4", acid)), combat.Regurgitation{}, 5)
		in.Mode, in.MaxSize, in.Entry = combat.ContainmentModeSwallowed, content.SizeSmall, combat.EntryAutomatic{}
		in.DisgorgeAfterFirstDamage, in.LockedAction, in.ActionKind = true, "Bite", combat.ContainmentActionKindAction
	case "toad":
		swallow(1, 1, ability.Strength, ctPeriodic(end, false, ctDamage("3d6", acid)), combat.Regurgitation{}, 5)
		in.MaxSize, in.Entry, in.LockedAction, in.ActionKind = content.SizeMedium, combat.EntryAutomatic{}, "Bite", combat.ContainmentActionKindAction
	default:
		tb.Fatalf("no fixture %q", name)
	}
	t, err := combat.NewContainmentTerms(in)
	if err != nil {
		tb.Fatalf("fixture %s: %v", name, err)
	}
	return t
}

// ctRecord builds the record of occupant held by container with the fixture terms, in phase inside.
func ctRecord(tb testing.TB, container, occupant string, size content.CreatureSize, terms combat.ContainmentTerms, ordinal int, mod ...func(*combat.ContainmentInput)) combat.Containment {
	tb.Helper()
	in := combat.ContainmentInput{Container: gid(container), Occupant: gid(occupant), OccupantSize: size, Terms: terms,
		Phase: combat.ContainmentInside(), PartOrdinal: ordinal}
	for _, m := range mod {
		m(&in)
	}
	r, err := combat.NewContainment(in)
	if err != nil {
		tb.Fatalf("NewContainment: %v", err)
	}
	return r
}

// ctSet builds a set from records.
func ctSet(tb testing.TB, rs ...combat.Containment) combat.Containments {
	tb.Helper()
	s, err := combat.NewContainments(rs)
	if err != nil {
		tb.Fatalf("NewContainments: %v", err)
	}
	return s
}

// ctOccupants builds a set in which container holds each (name, size) pair with the fixture terms.
func ctHolding(tb testing.TB, name, container string, occupants ...string) combat.Containments {
	tb.Helper()
	terms := ctTerms(tb, name)
	rs := make([]combat.Containment, len(occupants))
	for i, o := range occupants {
		size := content.SizeMedium
		if name == "frog" {
			size = content.SizeSmall
		}
		rs[i] = ctRecord(tb, container, o, size, terms, i+1, func(in *combat.ContainmentInput) { in.AddedConditions = terms.Imposed() })
	}
	return ctSet(tb, rs...)
}
