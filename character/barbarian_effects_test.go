package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func brbEffects(t *testing.T, level int, rage bool, sub content.SubclassID, turn character.BarbarianTurnState, facts character.BarbarianHostFacts) character.BarbarianEffects {
	t.Helper()
	state := character.InitialBarbarianState()
	if rage {
		state = brbRaging(t, level, 0)
	}
	return character.NewBarbarianEffects(brbStats(t, level), sub, state, turn, facts)
}

// Requirement: doc/specs/class-barbarian.md FR-027, FR-029, FR-030, FR-099 - while a Rage is active the effects report Resistance to Bludgeoning, Piercing and Slashing, Advantage on Strength
// checks and saving throws, and that Concentration and spells are lost; with no Rage, or no Barbarian levels, none of them.
// Path: per case (happy / unhappy: state error).
func TestBarbarianEffectsRageRules(t *testing.T) {
	t.Parallel()
	on := brbEffects(t, 1, true, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{})
	if got := on.DamageResistances(); !slices.Equal(got, []content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing}) {
		t.Errorf("resistances = %v", got)
	}
	got := on.DamageResistances()
	got[0] = content.DamageFire
	if on.DamageResistances()[0] != content.DamageBludgeoning {
		t.Error("DamageResistances returned a shared slice")
	}
	if !on.StrengthAdvantage() || !on.CannotMaintainConcentration() || !on.CannotCastSpells() {
		t.Error("an active Rage must give Strength Advantage and forbid Concentration and spells")
	}
	for _, off := range []character.BarbarianEffects{brbEffects(t, 1, false, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{}), {}} {
		if d := off.DamageResistances(); d == nil || len(d) != 0 || off.StrengthAdvantage() || off.CannotMaintainConcentration() || off.CannotCastSpells() || off.RageDamageBonusFor(brbStrengthAttack()) != 0 {
			t.Errorf("effects with no Rage = %#v", off)
		}
	}
}

// Requirement: doc/specs/class-barbarian.md FR-028 and table 4.4.1 (RD-01 to RD-10) - the Rage Damage bonus applies to an attack using Strength with a weapon or an Unarmed Strike that deals
// damage while a Rage is active, and is 0 otherwise.
// Path: per case (RD-01 to RD-10).
func TestBarbarianEffectsRageDamageBonus(t *testing.T) {
	t.Parallel()
	weapon := brbStrengthAttack()
	unarmed := weapon
	unarmed.Kind = character.BarbarianUnarmedStrike
	dex := weapon
	dex.UsesStrength = false
	noDamage := weapon
	noDamage.DealsDamage = false
	noKind := weapon
	noKind.Kind = 0
	for _, c := range []struct {
		label  string
		level  int
		rage   bool
		attack character.BarbarianAttackFacts
		want   int
	}{
		{"RD-01", 1, true, weapon, 2}, {"RD-02", 5, true, unarmed, 2}, {"RD-03", 8, true, weapon, 2}, {"RD-04", 9, true, weapon, 3},
		{"RD-05", 15, true, weapon, 3}, {"RD-06", 16, true, weapon, 4}, {"RD-07", 20, true, weapon, 4}, {"RD-08", 9, true, dex, 0},
		{"RD-09", 9, true, noDamage, 0}, {"RD-10", 9, false, weapon, 0}, {"no attack kind", 9, true, noKind, 0},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			e := brbEffects(t, c.level, c.rage, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{})
			if got := e.RageDamageBonusFor(c.attack); got != c.want {
				t.Errorf("bonus = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-079, FR-122 - the Path of the Berserker has Immunity to Charmed and Frightened while a Rage is active from level 6 (a copy of the set); below
// level 6, with no Rage, with another or no subclass the set is empty.
// Path: per case (unhappy: boundary, wrong subclass).
func TestBarbarianEffectsImmunities(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		level int
		rage  bool
		sub   content.SubclassID
		want  int
	}{
		{"level 6", 6, true, character.BerserkerSubclassID, 2}, {"level 5", 5, true, character.BerserkerSubclassID, 0}, {"no Rage", 6, false, character.BerserkerSubclassID, 0},
		{"another subclass", 6, true, "wild_heart", 0}, {"no subclass", 20, true, "", 0},
	} {
		got := brbEffects(t, c.level, c.rage, c.sub, character.NewBarbarianTurn(), character.BarbarianHostFacts{}).Immunities()
		_, charmed := got[content.ConditionCharmed]
		_, frightened := got[content.ConditionFrightened]
		if got == nil || len(got) != c.want || (c.want == 2 && (!charmed || !frightened)) {
			t.Errorf("%s: immunities = %v, want %d", c.name, got, c.want)
		}
	}
	e := brbEffects(t, 6, true, character.BerserkerSubclassID, character.NewBarbarianTurn(), character.BarbarianHostFacts{})
	delete(e.Immunities(), content.ConditionCharmed)
	if len(e.Immunities()) != 2 {
		t.Error("Immunities returned a shared map")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-047, FR-056, FR-057, FR-058, FR-073 - the level-gated effects: Danger Sense from 2 unless Incapacitated, Extra Attack from 5, Fast Movement from 5 unless
// wearing Heavy armor, Feral Instinct from 7; Indomitable Might from 18 for Strength only, using the score when the total is below it.
// Path: per case (unhappy: boundary, rule interaction).
func TestBarbarianEffectsLevelGatedFeatures(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			plain := brbEffects(t, n, false, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{})
			heavy := brbEffects(t, n, false, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{WearingHeavyArmor: true, Incapacitated: true})
			attacks, speed := 1, 0
			if n >= 5 {
				attacks, speed = 2, 10
			}
			if plain.DangerSenseAdvantage() != (n >= 2) || heavy.DangerSenseAdvantage() || plain.AttacksPerAttackAction() != attacks || plain.SpeedBonusFeet() != speed ||
				heavy.SpeedBonusFeet() != 0 || plain.InitiativeAdvantage() != (n >= 7) {
				t.Errorf("level %d: danger %v/%v attacks %d speed %d/%d initiative %v", n, plain.DangerSenseAdvantage(), heavy.DangerSenseAdvantage(), plain.AttacksPerAttackAction(),
					plain.SpeedBonusFeet(), heavy.SpeedBonusFeet(), plain.InitiativeAdvantage())
			}
			for _, c := range []struct {
				strength     bool
				total, score int
				want         int
			}{{true, 9, 16, 16}, {true, 16, 16, 16}, {true, 17, 16, 17}, {false, 9, 16, 9}} {
				want := c.want
				if n < 18 {
					want = c.total
				}
				if got := plain.IndomitableMightFloor(c.strength, c.total, c.score); got != want {
					t.Errorf("IndomitableMightFloor(%v, %d, %d) = %d, want %d", c.strength, c.total, c.score, got, want)
				}
			}
		})
	}
	if zero := (character.BarbarianEffects{}); zero.AttacksPerAttackAction() != 0 || zero.DangerSenseAdvantage() || zero.SpeedBonusFeet() != 0 || zero.InitiativeAdvantage() {
		t.Error("the zero effects must give nothing")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-048, FR-050, FR-097 - Reckless Attack gives Advantage on attack rolls using Strength, and attack rolls against the Barbarian have Advantage, while it
// is in force; not for a non-Strength attack, not before level 2, not once the next turn has started.
// Path: per case (unhappy: boundary).
func TestBarbarianEffectsRecklessAttack(t *testing.T) {
	t.Parallel()
	reckless := character.NewBarbarianTurn().DeclareReckless()
	dex := brbStrengthAttack()
	dex.UsesStrength = false
	for _, c := range []struct {
		name         string
		level        int
		turn         character.BarbarianTurnState
		attack       character.BarbarianAttackFacts
		mine, theirs bool
	}{
		{"declared", 2, reckless, brbStrengthAttack(), true, true},
		{"declared, Dexterity attack", 2, reckless, dex, false, true},
		{"not declared", 2, character.NewBarbarianTurn(), brbStrengthAttack(), false, false},
		{"level 1", 1, reckless, brbStrengthAttack(), false, false},
		{"after the next turn starts", 2, reckless.StartNextTurn(), brbStrengthAttack(), false, false},
		{"other creatures' turns, still declared", 9, reckless.SpendBonusAction().SpendReaction(), brbStrengthAttack(), true, true},
	} {
		e := brbEffects(t, c.level, false, "", c.turn, character.BarbarianHostFacts{})
		if e.RecklessAdvantageOnStrengthAttacks(c.attack) != c.mine || e.AttacksAgainstHaveAdvantage() != c.theirs {
			t.Errorf("%s: mine %v theirs %v, want %v %v", c.name, e.RecklessAdvantageOnStrengthAttacks(c.attack), e.AttacksAgainstHaveAdvantage(), c.mine, c.theirs)
		}
	}
}

// Requirement: doc/specs/class-barbarian.md FR-052 - from level 3, while a Rage is active, ability checks using Acrobatics, Intimidation, Perception, Stealth or Survival may use Strength; not for other
// skills, below level 3 or with no Rage.
// Path: per case (unhappy: boundary).
func TestBarbarianEffectsStrengthSkillSubstitutions(t *testing.T) {
	t.Parallel()
	want := []content.Skill{content.SkillAcrobatics, content.SkillIntimidation, content.SkillPerception, content.SkillStealth, content.SkillSurvival}
	if got := brbEffects(t, 3, true, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{}).StrengthSkillSubstitutions(); !slices.Equal(got, want) {
		t.Errorf("substitutions = %v, want %v", got, want)
	}
	for _, e := range []character.BarbarianEffects{brbEffects(t, 2, true, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{}), brbEffects(t, 3, false, "", character.NewBarbarianTurn(), character.BarbarianHostFacts{})} {
		if got := e.StrengthSkillSubstitutions(); got == nil || len(got) != 0 {
			t.Errorf("substitutions = %v, want none", got)
		}
	}
}

// Requirement: doc/specs/class-barbarian.md FR-039, FR-040, FR-112 - the Unarmored Defense basis is the feature barbarian_unarmored_defense with the Constitution modifier and a Shield allowed.
// Path: happy.
func TestBarbarianUnarmoredDefenseBasis(t *testing.T) {
	t.Parallel()
	b := character.BarbarianUnarmoredDefenseBasis()
	if b.ID() != "barbarian_unarmored_defense" || !slices.Equal(b.Abilities(), []ability.Ability{ability.Constitution}) || !b.ShieldAllowed() {
		t.Errorf("basis = %v %v %v", b.ID(), b.Abilities(), b.ShieldAllowed())
	}
}

// Requirement: doc/specs/class-barbarian.md FR-075 and table 4.8.2 (PC-01 to PC-06) - Primal Champion raises Strength and Constitution by 4 to a maximum of 25, in that order, never lowering a score above 25,
// each with the maximum 25 for the host to pass to ChangeAbilityScore.
// Path: per case (PC-01 to PC-06).
func TestPrimalChampion(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label                  string
		str, con, wantS, wantC int
	}{
		{"PC-01", 16, 14, 20, 18}, {"PC-02", 20, 20, 24, 24}, {"PC-03", 21, 22, 25, 25}, {"PC-04", 25, 25, 25, 25}, {"PC-05", 8, 8, 12, 12}, {"PC-06", 27, 20, 27, 24},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			got := must(character.PrimalChampion(must(ability.NewScore(c.str)), must(ability.NewScore(c.con))))
			want := []character.BarbarianAbilityChange{
				{Ability: ability.Strength, NewScore: c.wantS, Maximum: 25}, {Ability: ability.Constitution, NewScore: c.wantC, Maximum: 25},
			}
			if !slices.Equal(got, want) {
				t.Errorf("changes = %v, want %v", got, want)
			}
		})
	}
	_, err := character.PrimalChampion(ability.Score{}, must(ability.NewScore(10)))
	wantInvalid(t, err, "zero Strength")
	_, err = character.PrimalChampion(must(ability.NewScore(10)), ability.Score{})
	wantInvalid(t, err, "zero Constitution")
}

// Requirement: doc/specs/class-barbarian.md FR-087, FR-099 - effects built for zero stats (no Barbarian levels) give nothing even when a Rage state is passed in: no resistance, no Strength Advantage, no immunity, no substitution.
// Path: unhappy (state error).
func TestBarbarianEffectsWithoutBarbarianLevels(t *testing.T) {
	t.Parallel()
	e := character.NewBarbarianEffects(character.BarbarianLevelStats{}, character.BerserkerSubclassID, brbRaging(t, 20, 0), character.NewBarbarianTurn().DeclareReckless(), character.BarbarianHostFacts{})
	if len(e.DamageResistances()) != 0 || e.StrengthAdvantage() || e.CannotCastSpells() || e.CannotMaintainConcentration() || len(e.Immunities()) != 0 ||
		len(e.StrengthSkillSubstitutions()) != 0 || e.RageDamageBonusFor(brbStrengthAttack()) != 0 || e.AttacksAgainstHaveAdvantage() {
		t.Errorf("effects = %+v, want none", e)
	}
}
