package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-barbarian.md FR-048, FR-049, FR-050, FR-087, FR-098 - Reckless Attack is declared on the first attack roll of the turn from level 2 and marks the turn; it is refused with no
// Barbarian levels, below level 2 and after the first attack roll; an attack that does not use Strength is accepted with no Advantage on it.
// Path: per case (happy / unhappy: state error, boundary).
func TestBarbarianDeclareReckless(t *testing.T) {
	t.Parallel()
	dex := brbStrengthAttack()
	dex.UsesStrength = false
	fresh := character.NewBarbarianTurn()
	for _, c := range []struct {
		name      string
		level     int
		turn      character.BarbarianTurnState
		attack    character.BarbarianAttackFacts
		want      character.BarbarianReason
		advantage bool
	}{
		{"level 2", 2, fresh, brbStrengthAttack(), 0, true}, {"level 20", 20, fresh, brbStrengthAttack(), 0, true}, {"a Dexterity attack", 2, fresh, dex, 0, false},
		{"level 1", 1, fresh, brbStrengthAttack(), character.BarbarianFeatureNotGained, false},
		{"after the first attack roll", 5, fresh.RecordAttackRoll(), brbStrengthAttack(), character.BarbarianNotFirstAttackRoll, false},
		{"no Barbarian levels", 0, fresh, brbStrengthAttack(), character.BarbarianNotABarbarian, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := brbStats(t, max(c.level, 1))
			if c.level == 0 {
				stats = character.BarbarianLevelStats{}
			}
			res := character.BarbarianAttackRules{}.DeclareReckless(stats, c.turn, c.attack)
			if c.want != 0 {
				if got, ok := res.(character.RecklessRefused); !ok || got.Reason != c.want {
					t.Errorf("= %#v, want refused %v", res, c.want)
				}
				return
			}
			got, ok := res.(character.RecklessDeclared)
			if !ok || !got.Turn().RecklessDeclared() || got.AdvantageOnThisAttack() != c.advantage || c.turn.RecklessDeclared() {
				t.Errorf("= %#v, want declared (advantage %v)", res, c.advantage)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-060, FR-087, FR-102 - a Brutal Strike is declared before the roll from level 9 when Reckless Attack was used and the attack uses Strength, forgoing Advantage and
// naming the effects the level allows (two from 13, one more at 17); otherwise it is refused, in the order: no Barbarian levels, level, Reckless Attack, Strength.
// Path: per case (happy / unhappy: boundary, state error, rule interaction).
func TestBarbarianDeclareBrutalStrike(t *testing.T) {
	t.Parallel()
	reckless := character.NewBarbarianTurn().DeclareReckless()
	dex := brbStrengthAttack()
	dex.UsesStrength = false
	two := []character.BrutalStrikeEffect{character.ForcefulBlow, character.HamstringBlow}
	four := []character.BrutalStrikeEffect{character.ForcefulBlow, character.HamstringBlow, character.StaggeringBlow, character.SunderingBlow}
	for _, c := range []struct {
		name    string
		level   int
		turn    character.BarbarianTurnState
		attack  character.BarbarianAttackFacts
		want    character.BarbarianReason
		allowed []character.BrutalStrikeEffect
		maximum int
	}{
		{"level 9", 9, reckless, brbStrengthAttack(), 0, two, 1}, {"level 12", 12, reckless, brbStrengthAttack(), 0, two, 1}, {"level 13", 13, reckless, brbStrengthAttack(), 0, four, 1},
		{"level 16", 16, reckless, brbStrengthAttack(), 0, four, 1}, {"level 17", 17, reckless, brbStrengthAttack(), 0, four, 2},
		{"level 8", 8, reckless, brbStrengthAttack(), character.BarbarianFeatureNotGained, nil, 0},
		{"no Reckless Attack", 9, character.NewBarbarianTurn(), brbStrengthAttack(), character.BarbarianRecklessNotDeclared, nil, 0},
		{"a Dexterity attack", 9, reckless, dex, character.BarbarianNotAStrengthAttack, nil, 0},
		{"no Barbarian levels", 0, reckless, brbStrengthAttack(), character.BarbarianNotABarbarian, nil, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := brbStats(t, max(c.level, 1))
			if c.level == 0 {
				stats = character.BarbarianLevelStats{}
			}
			res := character.BarbarianAttackRules{}.DeclareBrutalStrike(stats, c.turn, c.attack)
			if c.want != 0 {
				if got, ok := res.(character.BrutalStrikeRefused); !ok || got.Reason != c.want {
					t.Errorf("= %#v, want refused %v", res, c.want)
				}
				return
			}
			got, ok := res.(character.BrutalStrikeDeclared)
			if !ok || !got.ForgoAdvantage() || !slices.Equal(got.AllowedEffects(), c.allowed) || got.MaximumEffects() != c.maximum {
				t.Fatalf("= %#v, want declared with %v, maximum %d", res, c.allowed, c.maximum)
			}
			allowed := got.AllowedEffects()
			allowed[0] = character.SunderingBlow
			if got.AllowedEffects()[0] != character.ForcefulBlow {
				t.Error("AllowedEffects returned a shared slice")
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-061, FR-062, FR-069, FR-070, FR-102, FR-114 and table 4.8.3 (BS-01 to BS-13) - ResolveBrutalStrike returns the extra 1d10 (2d10 from level 17) of the attack's damage type,
// not rolled, and the effects named; zero effects is valid; it rejects a duplicate, a locked effect and too many; a miss gives no damage and no effect; Disadvantage refuses it and a combined roll (Normal) is allowed.
// Path: per case (BS-01 to BS-13).
func TestBarbarianResolveBrutalStrike(t *testing.T) {
	t.Parallel()
	forceful, hamstring, stagger, sunder := character.ForcefulBlow, character.HamstringBlow, character.StaggeringBlow, character.SunderingBlow
	miss := brbStrengthAttack()
	miss.Hit = false
	for _, c := range []struct {
		label   string
		level   int
		effects []character.BrutalStrikeEffect
		attack  character.BarbarianAttackFacts
		want    character.BarbarianReason
		count   int
		hit     bool
	}{
		{"BS-01", 9, nil, brbStrengthAttack(), 0, 1, true}, {"BS-02", 9, []character.BrutalStrikeEffect{forceful}, brbStrengthAttack(), 0, 1, true},
		{"BS-03", 9, []character.BrutalStrikeEffect{forceful, hamstring}, brbStrengthAttack(), character.BarbarianTooManyEffects, 0, false},
		{"BS-04", 13, nil, brbStrengthAttack(), 0, 1, true}, {"BS-05", 13, []character.BrutalStrikeEffect{stagger}, brbStrengthAttack(), 0, 1, true},
		{"BS-06", 13, []character.BrutalStrikeEffect{stagger, sunder}, brbStrengthAttack(), character.BarbarianTooManyEffects, 0, false},
		{"BS-07", 9, []character.BrutalStrikeEffect{stagger}, brbStrengthAttack(), character.BarbarianEffectNotUnlocked, 0, false},
		{"BS-08", 17, nil, brbStrengthAttack(), 0, 2, true}, {"BS-09", 17, []character.BrutalStrikeEffect{sunder}, brbStrengthAttack(), 0, 2, true},
		{"BS-10", 17, []character.BrutalStrikeEffect{forceful, sunder}, brbStrengthAttack(), 0, 2, true},
		{"BS-11", 17, []character.BrutalStrikeEffect{forceful, forceful}, brbStrengthAttack(), character.BarbarianDuplicateEffect, 0, false},
		{"BS-12", 17, []character.BrutalStrikeEffect{forceful, hamstring, stagger}, brbStrengthAttack(), character.BarbarianTooManyEffects, 0, false},
		{"BS-13", 17, []character.BrutalStrikeEffect{forceful, sunder, stagger}, miss, 0, 0, false},
		{"level 8", 8, nil, brbStrengthAttack(), character.BarbarianFeatureNotGained, 0, false},
		{"a Dexterity attack", 9, nil, func() character.BarbarianAttackFacts { a := brbStrengthAttack(); a.UsesStrength = false; return a }(), character.BarbarianNotAStrengthAttack, 0, false},
		{"Disadvantage", 9, nil, func() character.BarbarianAttackFacts {
			a := brbStrengthAttack()
			a.RollMode = modifier.WithDisadvantage
			return a
		}(), character.BarbarianAttackHasDisadvantage, 0, false},
		{"Disadvantage on a miss", 9, nil, func() character.BarbarianAttackFacts { a := miss; a.RollMode = modifier.WithDisadvantage; return a }(), character.BarbarianAttackHasDisadvantage, 0, false},
		{"Advantage forgone is Normal", 9, nil, func() character.BarbarianAttackFacts {
			a := brbStrengthAttack()
			a.RollMode = modifier.Normal
			return a
		}(), 0, 1, true},
		{"a still-advantaged roll", 9, nil, func() character.BarbarianAttackFacts {
			a := brbStrengthAttack()
			a.RollMode = modifier.WithAdvantage
			return a
		}(), 0, 1, true},
		{"same effect twice below 13", 9, []character.BrutalStrikeEffect{forceful, forceful}, brbStrengthAttack(), character.BarbarianDuplicateEffect, 0, false},
		{"locked beats duplicate", 9, []character.BrutalStrikeEffect{stagger, stagger}, brbStrengthAttack(), character.BarbarianEffectNotUnlocked, 0, false},
		{"a locked effect on a miss", 9, []character.BrutalStrikeEffect{stagger}, miss, 0, 0, false},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			res := must(character.BarbarianAttackRules{}.ResolveBrutalStrike(brbStats(t, c.level), c.attack, c.effects))
			if c.want != 0 {
				if got, ok := res.(character.BrutalStrikeRefused); !ok || got.Reason != c.want {
					t.Errorf("= %#v, want refused %v", res, c.want)
				}
				return
			}
			got, ok := res.(character.BrutalStrikeResolved)
			if !ok {
				t.Fatalf("= %#v, want resolved", res)
			}
			extra, hit := got.ExtraDamage()
			wantEffects := c.effects
			if !c.hit {
				wantEffects = nil
			}
			if hit != c.hit || extra.Count() != c.count || (c.hit && (extra.Sides() != 10 || extra.DamageType() != content.DamageSlashing)) || got.Effects() == nil ||
				!slices.Equal(got.Effects(), append([]character.BrutalStrikeEffect{}, wantEffects...)) {
				t.Errorf("= %d d%d %v hit %v effects %v", extra.Count(), extra.Sides(), extra.DamageType(), hit, got.Effects())
			}
		})
	}
	stats := brbStats(t, 9)
	_, err := character.BarbarianAttackRules{}.ResolveBrutalStrike(stats, brbStrengthAttack(), []character.BrutalStrikeEffect{0})
	wantInvalid(t, err, "unknown effect")
	_, err = character.BarbarianAttackRules{}.ResolveBrutalStrike(stats, brbStrengthAttack(), []character.BrutalStrikeEffect{5})
	wantInvalid(t, err, "effect above the set")
	got := must(character.BarbarianAttackRules{}.ResolveBrutalStrike(character.BarbarianLevelStats{}, brbStrengthAttack(), nil))
	if got != (character.BrutalStrikeRefused{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("zero stats = %#v", got)
	}
	in := []character.BrutalStrikeEffect{forceful}
	res := must(character.BarbarianAttackRules{}.ResolveBrutalStrike(stats, brbStrengthAttack(), in)).(character.BrutalStrikeResolved)
	in[0] = hamstring
	res.Effects()[0] = sunder
	if res.Effects()[0] != forceful {
		t.Error("the resolution shares its effects with the caller")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-114 - ResolveBrutalStrike returns the same dice for a Critical Hit and a normal hit (the combat caller doubles them) and takes no dice.Roller.
// Path: unhappy (determinism).
func TestBarbarianBrutalStrikeIsTheSameForACriticalHit(t *testing.T) {
	t.Parallel()
	a := brbStrengthAttack()
	first, _ := must(character.BarbarianAttackRules{}.ResolveBrutalStrike(brbStats(t, 9), a, nil)).(character.BrutalStrikeResolved).ExtraDamage()
	second, _ := must(character.BarbarianAttackRules{}.ResolveBrutalStrike(brbStats(t, 9), a, nil)).(character.BrutalStrikeResolved).ExtraDamage()
	if first != second || first.Count() != 1 {
		t.Errorf("dice differ: %v, %v", first, second)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-063, FR-064, FR-068 - the effects carry their numbers: Forceful Blow pushes 15 feet and lets the Barbarian move half its Speed rounded down (HS-01 to HS-05), Hamstring
// Blow reduces Speed by 15 feet, Sundering Blow adds 5; the effects unlock at 9 (Forceful and Hamstring) and 13 (Staggering and Sundering) and have their names.
// Path: per case.
func TestBrutalStrikeEffectData(t *testing.T) {
	t.Parallel()
	if character.ForcefulBlowPushFeet != 15 || character.HamstringBlowSpeedFeet != 15 || character.SunderingBlowAttackBonus != 5 {
		t.Error("the effect numbers differ from the printed ones")
	}
	for _, c := range []struct {
		e      character.BrutalStrikeEffect
		name   string
		unlock int
	}{{character.ForcefulBlow, "forceful_blow", 9}, {character.HamstringBlow, "hamstring_blow", 9}, {character.StaggeringBlow, "staggering_blow", 13}, {character.SunderingBlow, "sundering_blow", 13}} {
		if c.e.String() != c.name || c.e.UnlockLevel() != c.unlock || !c.e.Valid() {
			t.Errorf("%v = %q unlock %d valid %v", int(c.e), c.e.String(), c.e.UnlockLevel(), c.e.Valid())
		}
	}
	for _, e := range []character.BrutalStrikeEffect{0, 5} {
		if e.String() != "unknown" || e.UnlockLevel() != 0 || e.Valid() {
			t.Errorf("effect %d = %q unlock %d valid %v, want unknown", int(e), e.String(), e.UnlockLevel(), e.Valid())
		}
	}
	for _, c := range []struct{ speed, half int }{{30, 15}, {25, 12}, {0, 0}, {1, 0}, {40, 20}} {
		if got := must(character.ForcefulBlowMove(c.speed)); got != c.half {
			t.Errorf("ForcefulBlowMove(%d) = %d, want %d", c.speed, got, c.half)
		}
	}
	_, err := character.ForcefulBlowMove(-1)
	wantInvalid(t, err, "negative Speed")
}
