package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const limitedUseSpec = "doc/specs/monster-limited-use.md "

// The monsters of these tests are synthetic: made-up names carrying the numbers the spec's case tables
// quote from the stat blocks of SRD 5.2.1, pp. 257-259 and 316, never a stat block.

func luMust[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// luFields is the smallest valid stat block.
func luFields(tb testing.TB) content.MonsterFields {
	tb.Helper()
	scores := map[ability.Ability]ability.Score{}
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	return content.MonsterFields{
		Sizes:      []content.CreatureSize{content.SizeMedium},
		Type:       content.CreatureTypeBeast,
		Alignment:  content.AlignmentUnaligned,
		ArmorClass: 12,
		Initiative: luMust(content.NewStatBlockInitiative(1, 11)),
		HitPoints:  luMust(content.NewHitPointsEntry(9, dice.MustExpression(2, dice.D8(), 0))),
		Speeds:     []content.SpeedEntry{luMust(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, ""))},
		Abilities:  luMust(content.NewAbilityBlock(luMust(ability.NewScores(scores)), saves)), PassivePerception: 10,
	}
}

// luMonster builds a monster from the smallest valid stat block changed by f.
func luMonster(tb testing.TB, f func(*content.MonsterFields)) content.Monster {
	tb.Helper()
	fields := luFields(tb)
	if f != nil {
		f(&fields)
	}
	return luMust(content.NewMonster("fixture_beast", "Fixture Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), fields))
}

func luPart(name string, usage content.PartUsage) content.StatBlockPart {
	return luMust(content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: usage, Text: "T."}))
}

func luPerDay(count, inLair int) content.PartUsage {
	return luMust(content.NewPartPerDay(count, inLair))
}

func luRecharge(low, high int) content.PartUsage { return luMust(content.NewPartRecharge(low, high)) }

func luLegendary(uses, inLair int) content.LegendaryActions {
	return luMust(content.NewLegendaryActions(uses, inLair, []content.StatBlockPart{luPart("Legendary Move", content.PartUnlimited{})}))
}

func luSpells(frequency content.SpellFrequency, names ...string) content.MonsterSpellGroup {
	var spells []content.MonsterSpellEntry
	for _, n := range names {
		spells = append(spells, luMust(content.NewMonsterSpellEntry(n, "", 0)))
	}
	return luMust(content.NewMonsterSpellGroup(frequency, spells))
}

func luSpellcasting(groups ...content.MonsterSpellGroup) content.StatBlockPart {
	sc := luMust(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{Ability: ability.Wisdom, Groups: groups}))
	return luMust(content.NewStatBlockPart("Spellcasting", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Spellcasting: sc}))
}

func luPerDaySpells(n int) content.SpellFrequency {
	return luMust(content.NewSpellFrequencyPerDay(n))
}

func luPerDayEach(n int) content.SpellFrequency {
	return luMust(content.NewSpellFrequencyPerDayEach(n))
}

// luDrake is a monster with every kind of limited part: Legendary Resistance 3/Day (4/Day in Lair) and a
// rest-recharged Trait, Mind 2/Day, a Recharge 5-6 and a Recharge 6 Action, a 1/Day Reaction, a part with
// another usage, and Legendary Actions 3 (4 in Lair).
func luDrake(tb testing.TB) content.Monster {
	tb.Helper()
	return luMonster(tb, func(f *content.MonsterFields) {
		f.Traits = []content.StatBlockPart{luPart("Resistance", luPerDay(3, 4)), luPart("Rest Trait", content.PartRechargeAfterRest{})}
		f.Actions = []content.StatBlockPart{luPart("Bite", content.PartUnlimited{}), luPart("Mind", luPerDay(2, 0)),
			luPart("Spray", luRecharge(5, 6)), luPart("Lash", luRecharge(6, 6)), luPart("Oath", luMust(content.NewPartUsageOther("1/Day; Requires Soul Bag")))}
		f.Reactions = []content.StatBlockPart{luPart("Parry", luPerDay(1, 0))}
		f.LegendaryActions = luLegendary(3, 4)
	})
}

func luRef(tb testing.TB, s combat.MonsterPartSection, i int) combat.LimitedUsePartRef {
	tb.Helper()
	return luMust(combat.NewLimitedUsePartRef(s, i))
}

// The refs of luDrake.
func luRes(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartTraits, 0)
}

func luRestTrait(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartTraits, 1)
}

func luBite(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartActions, 0)
}

func luMind(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartActions, 1)
}

func luSpray(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartActions, 2)
}

func luLash(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartActions, 3)
}

func luParry(tb testing.TB) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartReactions, 0)
}

const luSelf = "drake"

func luStart(tb testing.TB, m content.Monster, lair combat.LairStatus) combat.LimitedUseState {
	tb.Helper()
	return luMust(combat.StartLimitedUse(m, combat.MustCombatantID(luSelf), lair))
}

// luSpent applies Spend and returns the new state, failing the test on a refusal.
func luSpent(tb testing.TB, s combat.LimitedUseState, part combat.LimitedUsePartRef) combat.LimitedUseState {
	tb.Helper()
	out, ok := s.Spend(part).(combat.LimitedUseSpent)
	if !ok {
		tb.Fatalf("Spend(%v) was refused", part)
	}
	return out.Transition.State()
}

func luWindow(tb testing.TB, who string, round int) combat.LegendaryActionWindow {
	tb.Helper()
	return luMust(combat.NewLegendaryActionWindow(combat.MustCombatantID(who), round))
}

func luRoll(face int) dice.DieRoll { return dice.MustDieRoll(dice.D6(), face) }

// luPerDayOf returns the PerDayEntry of part.
func luPerDayOf(tb testing.TB, s combat.LimitedUseState, part combat.LimitedUsePartRef) combat.PerDayEntry {
	tb.Helper()
	for _, e := range s.Entries() {
		if p, ok := e.(combat.PerDayEntry); ok && p.Part() == part {
			return p
		}
	}
	tb.Fatalf("no X/Day entry for %v", part)
	return combat.PerDayEntry{}
}

// luReadiness returns the readiness of the Recharge or rest-recharge entry of part.
func luReadiness(tb testing.TB, s combat.LimitedUseState, part combat.LimitedUsePartRef) combat.RechargeReadiness {
	tb.Helper()
	for _, e := range s.Entries() {
		switch v := e.(type) {
		case combat.RechargeEntry:
			if v.Part() == part {
				return v.Readiness()
			}
		case combat.RestRechargeEntry:
			if v.Part() == part {
				return v.Readiness()
			}
		}
	}
	tb.Fatalf("no Recharge entry for %v", part)
	return 0
}

func luLegendaryOf(tb testing.TB, s combat.LimitedUseState) combat.LegendaryTracked {
	tb.Helper()
	t, ok := s.Legendary().(combat.LegendaryTracked)
	if !ok {
		tb.Fatalf("Legendary() = %#v, want LegendaryTracked", s.Legendary())
	}
	return t
}

// luTake takes a Legendary Action in w and returns the new state, failing the test on a refusal.
func luTake(tb testing.TB, s combat.LimitedUseState, w combat.LegendaryActionWindow) combat.LimitedUseState {
	tb.Helper()
	out, err := s.TakeLegendaryAction(w, combat.ConditionActionAvailabilityAvailable)
	allowed, ok := out.(combat.LegendaryActionAllowed)
	if err != nil || !ok {
		tb.Fatalf("TakeLegendaryAction(%v) = %#v, %v, want it allowed", w, out, err)
	}
	return allowed.Transition.State()
}
