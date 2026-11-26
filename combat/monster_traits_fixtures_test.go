package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

const traitsSpec = "doc/specs/monster-traits.md "

// The texts are the templates of table T-1 with a made-up noun, never a stat block.
const (
	mtLegendary = "If the drake fails a saving throw, it can choose to succeed instead."
	mtMagic     = "The drake has Advantage on saving throws against spells and other magical effects."
	mtUndead    = "If damage reduces the drake to 0 Hit Points, it makes a Constitution saving throw (DC 5 plus the damage taken) unless the damage is Radiant or from a Critical Hit. On a successful save, the drake drops to 1 Hit Point instead."
	mtRegenOne  = "The drake regains 10 Hit Points at the start of each of its turns if it has at least 1 Hit Point."
	mtRegenTwo  = "The drake regains 15 Hit Points at the start of each of its turns. If the drake takes Acid or Fire damage, this trait doesn't function on the drake's next turn. The drake dies only if it starts its turn with 0 Hit Points and doesn't regenerate."
	mtBurst     = "The drake explodes when it dies. Constitution Saving Throw: DC 11."
	mtSwarm     = "The swarm can occupy another creature's space and vice versa, and the swarm can move through any opening large enough for a Tiny insect. The swarm can't regain Hit Points or gain Temporary Hit Points."
	mtPack      = "The drake has Advantage on an attack roll against a creature if at least one of the drake's allies is within 5 feet of the creature and the ally doesn't have the Incapacitated condition."
	mtSunSens   = "While in sunlight, the drake has Disadvantage on ability checks and attack rolls."
	mtSunWeak   = "While in sunlight, the drake has Disadvantage on D20 Tests."
	mtFlyby     = "The drake doesn't provoke an Opportunity Attack when it flies out of an enemy's reach."
	mtAmorphous = "The drake can move through a space as narrow as 1 inch without expending extra movement to do so."
	mtSiege     = "The drake deals double damage to objects and structures."
	mtParry     = "Trigger: The drake is hit by a melee attack roll while holding a weapon. Response: The drake adds 2 to its AC against that attack, possibly causing it to miss."
)

func mtTextPart(name, text string, usage content.PartUsage) content.StatBlockPart {
	return luMust(content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: usage, Text: text}))
}

func mtBurstPart(text string, withSave bool) content.StatBlockPart {
	f := content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: text}
	if withSave {
		f.SavingThrow = luMust(content.NewStatBlockSavingThrow(ability.Constitution, 11, "", ""))
	}
	return luMust(content.NewStatBlockPart("Death Burst", f))
}

func mtMonster(tb testing.TB, traits, reactions []content.StatBlockPart) content.Monster {
	tb.Helper()
	return luMonster(tb, func(f *content.MonsterFields) { f.Traits, f.Reactions = traits, reactions })
}

func mtTrait(name, text string) content.StatBlockPart {
	return mtTextPart(name, text, content.PartUnlimited{})
}

func mtRef(tb testing.TB, i int) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartTraits, i)
}

func mtOne[T combat.MonsterTrait](tb testing.TB, r combat.RecognisedTraits) T {
	tb.Helper()
	ts := r.Traits()
	if len(ts) != 1 {
		tb.Fatalf("traits = %v, rejections = %v, want one trait", ts, r.Unrecognised())
	}
	t, ok := ts[0].(T)
	if !ok {
		tb.Fatalf("trait = %#v", ts[0])
	}
	return t
}

func mtRegen(tb testing.TB, amount int, types ...content.DamageType) combat.RegenerationTrait {
	tb.Helper()
	return luMust(combat.NewRegenerationTrait(mtRef(tb, 0), amount, types))
}

func mtTroll(tb testing.TB) combat.RegenerationTrait {
	tb.Helper()
	return mtRegen(tb, 15, content.DamageAcid, content.DamageFire)
}

func mtOni(tb testing.TB) combat.RegenerationTrait {
	tb.Helper()
	return mtRegen(tb, 10)
}

func mtAttack(tb testing.TB, natural, bonus, ac int) d20.AttackRollResult {
	tb.Helper()
	roll := luMust(d20.NewRoll(modifier.Normal, []int{natural}))
	return luMust(d20.ResolveAttack(roll, d20.NewRollModifiers(modifier.New(bonus)), d20.ArmorClass(ac), d20.DefaultCriticalRange))
}

func mtSave(tb testing.TB, natural, bonus, dc int) d20.CheckResult {
	tb.Helper()
	roll := luMust(d20.NewRoll(modifier.Normal, []int{natural}))
	return luMust(d20.ResolveCheck(d20.SavingThrow, roll, d20.NewRollModifiers(modifier.New(bonus)), d20.DifficultyClass(dc)))
}

func mtID() combat.CombatantID { return combat.MustCombatantID(luSelf) }

// mtHit applies damage to a creature and returns the transition.
func mtHit(tb testing.TB, s combat.HitPointStatus, kind combat.HitKind, damages ...combat.Damage) combat.HitPointTransitionApplied {
	tb.Helper()
	tr, err := s.ApplyDamage(mtID(), damages, nil, combat.OtherSource, kind, nil)
	return applied(tb, tr, err)
}

func mtZombie(tb testing.TB, current, maximum, temp int, rule combat.ZeroHitPointsRule) combat.HitPointStatus {
	tb.Helper()
	return status(tb, current, maximum, temp, combat.Alive{}, rule, combat.NoAffinities())
}

func mtKindsOf(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

func mtLimited(tb testing.TB) (combat.LimitedUseState, combat.LegendaryResistanceTrait) {
	tb.Helper()
	m := mtMonster(tb, []content.StatBlockPart{mtTextPart("Legendary Resistance", mtLegendary, luPerDay(3, 4))}, nil)
	trait := mtOne[combat.LegendaryResistanceTrait](tb, combat.RecogniseTraits(m))
	return luStart(tb, m, combat.OutsideLair), trait
}

func mtMonsterOf(tb testing.TB) content.Monster {
	tb.Helper()
	return mtMonster(tb, []content.StatBlockPart{mtTextPart("Legendary Resistance", mtLegendary, luPerDay(3, 4))}, nil)
}

func modifierZero() modifier.Modifier { return modifier.New(0) }

func restLong() character.RestKind { return character.LongRest }

// mtOverMax is one above the largest stat block number; on a 32-bit int it wraps to a negative number, which is
// invalid as well.
func mtOverMax() int {
	v := int64(content.MonsterMaxNumber)
	return int(v + 1)
}
