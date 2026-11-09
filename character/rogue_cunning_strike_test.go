package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

const (
	csPoison   = character.CunningStrikePoison
	csTrip     = character.CunningStrikeTrip
	csWithdraw = character.CunningStrikeWithdraw
	csDaze     = character.CunningStrikeDaze
	csKnockOut = character.CunningStrikeKnockOut
	csObscure  = character.CunningStrikeObscure
	csStealth  = character.CunningStrikeStealthAttack
)

// csInput is a Cunning Strike input for a Rogue of level with the Sneak Attack dice, a kit, Dexterity modifier +3, Proficiency
// Bonus +3, a Medium target and Speed 30.
func csInput(level, sneak int, effects ...character.CunningStrikeEffect) character.CunningStrikeInput {
	return character.CunningStrikeInput{
		Level: rl(level), SneakAttack: character.SneakAttack{}.Evaluate(character.SneakAttackInput{Dice: dice6(sneak), Attack: rogAttackT(), Finesse: true, Mode: 1}),
		Effects: effects, PoisonersKit: true, DexterityModifier: ability.Modifier{}, ProficiencyBonus: character.ProficiencyBonusFor(must(character.NewLevel(5))),
		TargetSize: content.SizeMedium, SpeedFeet: 30,
	}
}

// Requirement: doc/specs/class-rogue.md FR-050 to FR-056, FR-066, FR-067, FR-107 (SRD 5.2.1, "Level 5: Cunning Strike", "Level 11:
// Improved Cunning Strike", "Level 14: Devious Strikes", pp. 63-64) - a valid selection reports its cost and the Sneak Attack dice
// left; an invalid one reports every problem once, in check order.
// Path: per case (the Cunning Strike selection cases of the spec): happy / unhappy (boundary, rule interaction, state error, invalid input).
func TestCunningStrikeSelection(t *testing.T) {
	t.Parallel()
	thief := func(i character.CunningStrikeInput) character.CunningStrikeInput {
		i.Subclass = character.ThiefSubclassID
		return i
	}
	noKit := func(i character.CunningStrikeInput) character.CunningStrikeInput { i.PoisonersKit = false; return i }
	notDealt := func(i character.CunningStrikeInput) character.CunningStrikeInput {
		i.SneakAttack = character.SneakAttackDoesNotApply{}
		return i
	}
	for _, c := range []struct {
		name, path string
		in         character.CunningStrikeInput
		cost, left int
		problems   []character.CunningStrikeProblem
	}{
		{"level 5 Poison", "happy", csInput(5, 3, csPoison), 1, 2, nil},
		{"level 5 Withdraw", "happy", csInput(5, 3, csWithdraw), 1, 2, nil},
		{"level 14 Obscure", "happy", csInput(14, 7, csObscure), 3, 4, nil},
		{"level 14 Knock Out", "happy", csInput(14, 7, csKnockOut), 6, 1, nil},
		{"level 19 Knock Out and Daze", "happy", csInput(19, 10, csKnockOut, csDaze), 8, 2, nil},
		{"Thief level 9 Stealth Attack", "happy", thief(csInput(9, 5, csStealth)), 1, 4, nil},
		{"empty selection", "unhappy (empty case)", csInput(5, 3), 0, 3, nil},
		{"level 11 Trip and Withdraw", "unhappy (boundary)", csInput(11, 6, csTrip, csWithdraw), 2, 4, nil},
		{"level 14 Daze", "unhappy (boundary)", csInput(14, 7, csDaze), 2, 5, nil},
		{"level 14 Knock Out and Trip uses every die", "unhappy (boundary)", csInput(14, 7, csKnockOut, csTrip), 7, 0, nil},
		{"Trip on a Huge target is paid for", "unhappy (boundary)", func() character.CunningStrikeInput {
			i := csInput(5, 3, csTrip)
			i.TargetSize = content.SizeHuge
			return i
		}(), 1, 2, nil},
		{"level 4 Trip", "unhappy (boundary)", csInput(4, 2, csTrip), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeEffectNotAvailable}},
		{"level 13 Daze", "unhappy (boundary)", csInput(13, 7, csDaze), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeEffectNotAvailable}},
		{"Thief level 8 Stealth Attack", "unhappy (boundary)", thief(csInput(8, 4, csStealth)), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeEffectNotAvailable}},
		{"level 9 without a subclass Stealth Attack", "unhappy (rule interaction)", csInput(9, 5, csStealth), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeEffectNotAvailable}},
		{"level 5 Trip and Withdraw", "unhappy (boundary)", csInput(5, 3, csTrip, csWithdraw), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeTooManyEffects}},
		{"level 10 Trip and Withdraw", "unhappy (boundary)", csInput(10, 5, csTrip, csWithdraw), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeTooManyEffects}},
		{"level 11 Poison twice", "unhappy (boundary)", csInput(11, 6, csPoison, csPoison), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeDuplicateEffect}},
		{"level 14 Knock Out twice", "unhappy (rule interaction)", csInput(14, 7, csKnockOut, csKnockOut), 0, 0,
			[]character.CunningStrikeProblem{character.CunningStrikeDuplicateEffect, character.CunningStrikeInsufficientDice}},
		{"level 14 Knock Out and Daze", "unhappy (boundary)", csInput(14, 7, csKnockOut, csDaze), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeInsufficientDice}},
		{"Poison with no kit", "unhappy (state error)", noKit(csInput(5, 3, csPoison)), 0, 0, []character.CunningStrikeProblem{character.CunningStrikePoisonersKitMissing}},
		{"Trip when Sneak Attack was not dealt", "unhappy (state error)", notDealt(csInput(5, 3, csTrip)), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeSneakAttackNotDealt}},
		{"an invalid effect", "unhappy (invalid input)", csInput(5, 3, character.CunningStrikeEffect(0)), 0, 0, []character.CunningStrikeProblem{character.CunningStrikeEffectNotAvailable}},
		{"several problems", "unhappy (invalid input)", noKit(csInput(5, 3, csPoison, csDaze)), 0, 0,
			[]character.CunningStrikeProblem{character.CunningStrikeTooManyEffects, character.CunningStrikeEffectNotAvailable, character.CunningStrikePoisonersKitMissing}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			sel, err := character.NewCunningStrikeSelection(c.in)
			if len(c.problems) > 0 {
				if got := rogProblems[character.CunningStrikeProblem](t, err); !slices.Equal(got, c.problems) {
					t.Errorf("problems = %v, want %v", got, c.problems)
				}
				return
			}
			if err != nil || sel.Cost().Count() != c.cost || sel.RemainingDice().Count() != c.left || len(sel.Outcomes()) != len(c.in.Effects) {
				t.Errorf("selection = cost %d, left %d, %d outcomes, err %v; want cost %d, left %d, %d outcomes", sel.Cost().Count(), sel.RemainingDice().Count(), len(sel.Outcomes()), err, c.cost, c.left, len(c.in.Effects))
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-052 - each effect costs the die count of the effect table; an invalid effect costs none.
// Path: per case (the seven effects): happy.
func TestCunningStrikeEffectCosts(t *testing.T) {
	t.Parallel()
	names := []string{"poison", "trip", "withdraw", "daze", "knock_out", "obscure", "stealth_attack"}
	for i, want := range []int{1, 1, 1, 2, 6, 3, 1} {
		e := character.CunningStrikeEffect(i + 1)
		if e.Cost().Count() != want || e.Cost().Die().Sides() != 6 || e.String() != names[i] {
			t.Errorf("%s costs %d, renders %q; want %d, %q", names[i], e.Cost().Count(), e.String(), want, names[i])
		}
	}
	for _, e := range []character.CunningStrikeEffect{0, 8} {
		if e.Cost().Count() != 0 || e.String() != "unknown" {
			t.Errorf("effect %d costs %d, renders %q", e, e.Cost().Count(), e.String())
		}
	}
	for p, want := range map[character.CunningStrikeProblem]string{character.CunningStrikeSneakAttackNotDealt: "sneak_attack_not_dealt", character.CunningStrikeTooManyEffects: "too_many_effects",
		character.CunningStrikeDuplicateEffect: "duplicate_effect", character.CunningStrikeEffectNotAvailable: "effect_not_available", character.CunningStrikeInsufficientDice: "insufficient_dice",
		character.CunningStrikePoisonersKitMissing: "poisoners_kit_missing", 0: "unknown", 7: "unknown"} {
		if p.String() != want {
			t.Errorf("problem %d = %q, want %q", p, p.String(), want)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-057 - the DC of a Cunning Strike saving throw is 8 plus the Dexterity modifier plus the
// Proficiency Bonus.
// Path: per case (the DC cases of the spec): happy / unhappy (boundary: lowest and highest modifier).
func TestCunningStrikeDC(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		dex, bonus, want int
		path             string
	}{{-5, 3, 6, "boundary"}, {0, 3, 11, "happy"}, {3, 3, 14, "happy"}, {5, 4, 17, "happy"}, {5, 6, 19, "happy"}, {-5, 6, 9, "boundary"}, {10, 6, 24, "boundary"}} {
		t.Run(strconv.Itoa(c.dex)+" and "+strconv.Itoa(c.bonus), func(t *testing.T) {
			t.Parallel()
			score := must(ability.NewScore(max(10+2*c.dex, 1)))
			bonus := character.ProficiencyBonusFor(must(character.NewLevel(map[int]int{3: 5, 4: 9, 6: 17}[c.bonus])))
			if got := character.CunningStrikeDC(score.Modifier(), bonus); got != d20.DifficultyClass(c.want) {
				t.Errorf("DC = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-058 to FR-065, FR-068 (SRD 5.2.1, pp. 63-64) - every effect yields the outcome variant
// with its save, DC, condition, duration, repeat save, timing and cost, and no display text.
// Path: per case (one test per effect): happy.
func TestCunningStrikeOutcomes(t *testing.T) {
	t.Parallel()
	in := func(level, sneak int, e character.CunningStrikeEffect) character.CunningStrikeInput {
		i := csInput(level, sneak, e)
		i.Subclass = character.ThiefSubclassID
		i.DexterityModifier = must(ability.NewScore(16)).Modifier() // +3
		return i
	}
	outcome := func(i character.CunningStrikeInput) character.CunningStrikeOutcome {
		sel := must(character.NewCunningStrikeSelection(i))
		return sel.Outcomes()[0]
	}
	repeat := character.CunningStrikeRepeatSave{Ability: ability.Constitution, Trigger: character.CunningStrikeEndOfEachOfTargetsTurns, EndsOnSuccess: true}
	common := func(e character.CunningStrikeEffect) character.CunningStrikeCommon {
		return character.CunningStrikeCommon{Effect: e, Cost: e.Cost(), Timing: character.CunningStrikeImmediatelyAfterDamage}
	}
	huge := in(5, 3, csTrip)
	huge.TargetSize = content.SizeHuge
	large := in(5, 3, csTrip)
	large.TargetSize = content.SizeLarge
	slow := in(5, 3, csWithdraw)
	slow.SpeedFeet = 25
	for _, c := range []struct {
		name string
		got  character.CunningStrikeOutcome
		want character.CunningStrikeOutcome
	}{
		{"Poison", outcome(in(5, 3, csPoison)), character.CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common(csPoison), SaveAbility: ability.Constitution, DC: 14,
			Condition: content.ConditionPoisoned, Duration: character.CunningStrikeOneMinute, RepeatSave: repeat}},
		{"Trip", outcome(in(5, 3, csTrip)), character.CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common(csTrip), SaveAbility: ability.Dexterity, DC: 14,
			Condition: content.ConditionProne, Duration: character.CunningStrikeNoneSpecified, RepeatSave: character.CunningStrikeNoRepeatSave{}}},
		{"Trip on a Large target", outcome(large), character.CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common(csTrip), SaveAbility: ability.Dexterity, DC: 14,
			Condition: content.ConditionProne, Duration: character.CunningStrikeNoneSpecified, RepeatSave: character.CunningStrikeNoRepeatSave{}}},
		{"Trip on a Huge target", outcome(huge), character.CunningStrikeNoEffect{CunningStrikeCommon: common(csTrip), Reason: character.CunningStrikeTargetTooLarge}},
		{"Withdraw", outcome(in(5, 3, csWithdraw)), character.CunningStrikeWithdrawMovement{CunningStrikeCommon: common(csWithdraw), MaxFeet: 15}},
		{"Withdraw at Speed 25", outcome(slow), character.CunningStrikeWithdrawMovement{CunningStrikeCommon: common(csWithdraw), MaxFeet: 12}},
		{"Daze", outcome(in(14, 7, csDaze)), character.CunningStrikeDazeOnFailedSave{CunningStrikeCommon: common(csDaze), SaveAbility: ability.Constitution, DC: 14, Restriction: character.CunningStrikeOneOfMoveActionBonusAction, AppliesTo: character.CunningStrikeNextTurnOfTarget}},
		{"Knock Out", outcome(in(14, 7, csKnockOut)), character.CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common(csKnockOut), SaveAbility: ability.Constitution, DC: 14,
			Condition: content.ConditionUnconscious, Duration: character.CunningStrikeOneMinute, RepeatSave: repeat, EndsOnDamage: true}},
		{"Obscure", outcome(in(14, 7, csObscure)), character.CunningStrikeConditionOnFailedSave{CunningStrikeCommon: common(csObscure), SaveAbility: ability.Dexterity, DC: 14,
			Condition: content.ConditionBlinded, Duration: character.CunningStrikeUntilEndOfTargetsNextTurn, RepeatSave: character.CunningStrikeNoRepeatSave{}}},
		{"Stealth Attack", outcome(in(9, 5, csStealth)), character.CunningStrikeStealthAttackKeepsInvisible{CunningStrikeCommon: common(csStealth), Condition: content.ConditionInvisible, RequiredCover: character.CunningStrikeThreeQuartersOrTotal}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("outcome = %#v, want %#v", c.got, c.want)
			}
			if c.got.Common().Timing != character.CunningStrikeImmediatelyAfterDamage {
				t.Errorf("timing = %v", c.got.Common().Timing)
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-101, FR-102, FR-103 - a zero Rogue level or a negative Speed is refused with
// ErrInvalidInput; Outcomes returns a copy; a selection does not change its input; a selection with no Sneak Attack and no effects is valid.
// Path: unhappy (immutability, invalid input).
func TestCunningStrikeInputAndCopies(t *testing.T) {
	t.Parallel()
	bad := csInput(5, 3, csPoison)
	bad.Level = character.RogueLevel{}
	_, err := character.NewCunningStrikeSelection(bad)
	wantInvalid(t, err, "zero level")
	bad = csInput(5, 3, csPoison)
	bad.SpeedFeet = -1
	_, err = character.NewCunningStrikeSelection(bad)
	wantInvalid(t, err, "negative Speed")
	in := csInput(5, 3, csPoison)
	sel := must(character.NewCunningStrikeSelection(in))
	outs := sel.Outcomes()
	outs[0] = nil
	if sel.Outcomes()[0] == nil || in.Effects[0] != csPoison {
		t.Error("Outcomes shares storage or the input changed")
	}
	none := csInput(5, 3)
	none.SneakAttack = nil
	empty, err := character.NewCunningStrikeSelection(none)
	if err != nil || empty.RemainingDice().Count() != 0 || empty.Cost().Count() != 0 || len(empty.Outcomes()) != 0 {
		t.Errorf("empty selection without Sneak Attack = %v, %v", empty, err)
	}
}
