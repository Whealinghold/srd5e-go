package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-rogue.md FR-069 (SRD 5.2.1, "Level 2: Cunning Action", p. 62) - Cunning Action grants Dash, Disengage
// and Hide from Rogue level 2 and nothing at level 1.
// Path: per case (levels 1, 2, 20): unhappy (boundary).
func TestRogueGrantedActions(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		want  []character.RogueGrantedAction
	}{{1, []character.RogueGrantedAction{}}, {2, []character.RogueGrantedAction{character.RogueDash, character.RogueDisengage, character.RogueHide}}, {20, []character.RogueGrantedAction{character.RogueDash, character.RogueDisengage, character.RogueHide}}} {
		if got := (character.RogueGrantedActions{}).At(rl(c.level)); !slices.Equal(got, c.want) {
			t.Errorf("level %d: %v, want %v", c.level, got, c.want)
		}
	}
	for a, want := range map[character.RogueGrantedAction]string{character.RogueDash: "dash", character.RogueDisengage: "disengage", character.RogueHide: "hide", 0: "unknown"} {
		if a.String() != want {
			t.Errorf("action %d = %q, want %q", a, a.String(), want)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-070, FR-071, FR-072 (SRD 5.2.1, "Level 3: Steady Aim", p. 62) - from Rogue level 3
// without having moved Steady Aim is granted (Advantage on the next attack roll this turn, Speed 0, a Bonus Action); otherwise it
// is not available, with every failing reason in check order.
// Path: per case (levels 2 and 3, moved and not): happy / unhappy (boundary, state error).
func TestSteadyAim(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level   int
		moved   bool
		reasons []character.SteadyAimReason
	}{
		{3, false, nil},
		{20, false, nil},
		{3, true, []character.SteadyAimReason{character.SteadyAimMovedThisTurn}},
		{2, false, []character.SteadyAimReason{character.SteadyAimLevelTooLow}},
		{2, true, []character.SteadyAimReason{character.SteadyAimLevelTooLow, character.SteadyAimMovedThisTurn}},
	} {
		t.Run("level "+strconv.Itoa(c.level)+" moved "+strconv.FormatBool(c.moved), func(t *testing.T) {
			t.Parallel()
			got := character.SteadyAim{}.Use(rl(c.level), c.moved)
			if c.reasons == nil {
				g, ok := got.(character.SteadyAimGranted)
				if !ok || g.AdvantageOn() != character.SteadyAimNextAttackRollThisTurn || g.SpeedUntilEndOfTurn() != 0 || g.ActionCost() != character.SteadyAimBonusAction {
					t.Errorf("result = %#v, want granted", got)
				}
				return
			}
			n, ok := got.(character.SteadyAimNotAvailable)
			if !ok || !slices.Equal(n.Reasons(), c.reasons) {
				t.Errorf("result = %#v, want not available %v", got, c.reasons)
			}
			n.Reasons()[0] = 0
			if n.Reasons()[0] == 0 {
				t.Error("Reasons shares storage")
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-073 (SRD 5.2.1, "Level 5: Uncanny Dodge", p. 63) - damage is halved and rounded down; a
// negative amount is refused with ErrInvalidInput.
// Path: per case (the Uncanny Dodge cases): happy / unhappy (boundary).
func TestUncannyDodgeHalve(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ in, want int }{{0, 0}, {1, 0}, {2, 1}, {7, 3}, {8, 4}, {31, 15}} {
		if got, err := (character.RogueUncannyDodge{}).Halve(c.in); err != nil || got != c.want {
			t.Errorf("Halve(%d) = %d, %v; want %d", c.in, got, err, c.want)
		}
	}
	_, err := character.RogueUncannyDodge{}.Halve(-1)
	wantInvalid(t, err, "negative damage")
}

// Requirement: doc/specs/class-rogue.md FR-074 (SRD 5.2.1, "Level 5: Uncanny Dodge", p. 63) - Uncanny Dodge is available only at
// Rogue level 5 or higher with the Reaction available, against a seen attacker whose attack hit; each failing precondition alone
// and all four together are reported in order.
// Path: per case (each precondition alone, all four): unhappy (state error, boundary).
func TestUncannyDodgeCanUse(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                string
		level               int
		reaction, seen, hit bool
		reasons             []character.RogueUncannyDodgeReason
	}{
		{"available at level 5", 5, true, true, true, nil},
		{"level 4", 4, true, true, true, []character.RogueUncannyDodgeReason{character.RogueUncannyDodgeLevelTooLow}},
		{"no Reaction", 5, false, true, true, []character.RogueUncannyDodgeReason{character.RogueUncannyDodgeReactionUnavailable}},
		{"attacker not seen", 5, true, false, true, []character.RogueUncannyDodgeReason{character.RogueUncannyDodgeAttackerNotSeen}},
		{"the attack missed", 5, true, true, false, []character.RogueUncannyDodgeReason{character.RogueUncannyDodgeAttackMissed}},
		{"all four", 1, false, false, false, []character.RogueUncannyDodgeReason{character.RogueUncannyDodgeLevelTooLow, character.RogueUncannyDodgeReactionUnavailable, character.RogueUncannyDodgeAttackerNotSeen, character.RogueUncannyDodgeAttackMissed}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.RogueUncannyDodge{}.CanUse(rl(c.level), c.reaction, c.seen, rogAttack(t, c.hit))
			if c.reasons == nil {
				if _, ok := got.(character.RogueUncannyDodgeAvailable); !ok {
					t.Errorf("result = %#v, want available", got)
				}
				return
			}
			n, ok := got.(character.RogueUncannyDodgeNotAvailable)
			if !ok || !slices.Equal(n.Reasons(), c.reasons) {
				t.Errorf("result = %#v, want not available %v", got, c.reasons)
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-075, FR-076, FR-077 (SRD 5.2.1, "Level 7: Evasion", p. 63) - a Dexterity save for half
// damage takes none on a success and half (rounded down) on a failure from Rogue level 7; the normal result stands while
// Incapacitated, below level 7, for another ability and for an effect that does no damage on a success; negative damage is refused.
// Path: per case (the Evasion cases): happy / unhappy (boundary, state error, rule interaction).
func TestRogueEvasion(t *testing.T) {
	t.Parallel()
	in := func(damage int, save ability.Ability, half, ok, incap bool, level int) character.RogueEvasionInput {
		return character.RogueEvasionInput{Level: rl(level), FullDamage: damage, SaveAbility: save, HalfOnSuccess: half, SaveSucceeded: ok, Incapacitated: incap}
	}
	dex, con := ability.Dexterity, ability.Constitution
	for _, c := range []struct {
		name string
		in   character.RogueEvasionInput
		want int
	}{
		{"Dexterity success", in(10, dex, true, true, false, 7), 0},
		{"Dexterity failure", in(10, dex, true, false, false, 7), 5},
		{"failure with 7 damage", in(7, dex, true, false, false, 7), 3},
		{"failure with 1 damage", in(1, dex, true, false, false, 7), 0},
		{"Incapacitated success", in(10, dex, true, true, true, 7), 5},
		{"Incapacitated failure", in(10, dex, true, false, true, 7), 10},
		{"Constitution success", in(10, con, true, true, false, 7), 5},
		{"Constitution failure", in(10, con, true, false, false, 7), 10},
		{"no damage on a success", in(10, dex, false, true, false, 7), 0},
		{"no damage on a success, failure", in(10, dex, false, false, false, 7), 10},
		{"level 6 success", in(10, dex, true, true, false, 6), 5},
		{"level 6 failure", in(10, dex, true, false, false, 6), 10},
		{"level 20 failure", in(10, dex, true, false, false, 20), 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got, err := (character.RogueEvasion{}).Apply(c.in); err != nil || got != c.want {
				t.Errorf("Apply = %d, %v; want %d", got, err, c.want)
			}
		})
	}
	_, err := character.RogueEvasion{}.Apply(in(-1, dex, true, true, false, 7))
	wantInvalid(t, err, "negative damage")
}

// Requirement: doc/specs/class-rogue.md FR-078, FR-079 (SRD 5.2.1, "Level 7: Reliable Talent", p. 63) - a d20 roll of 9 or lower
// becomes 10 for an ability check that uses a skill or tool proficiency at Rogue level 7 or higher; any other roll stays; a roll
// outside 1 to 20 is refused.
// Path: per case (the Reliable Talent cases): happy / unhappy (boundary, rule interaction).
func TestRogueReliableTalent(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		level int
		kind  d20.Kind
		prof  bool
		roll  int
		want  int
	}{
		{"roll 1", 7, d20.AbilityCheck, true, 1, 10}, {"roll 9", 7, d20.AbilityCheck, true, 9, 10}, {"roll 10", 7, d20.AbilityCheck, true, 10, 10},
		{"roll 11", 7, d20.AbilityCheck, true, 11, 11}, {"level 6", 6, d20.AbilityCheck, true, 5, 5}, {"natural 20", 20, d20.AbilityCheck, true, 20, 20},
		{"no proficiency", 7, d20.AbilityCheck, false, 5, 5}, {"saving throw", 7, d20.SavingThrow, true, 5, 5}, {"attack roll", 7, d20.AttackRoll, true, 5, 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got, err := (character.RogueReliableTalent{}).Apply(rl(c.level), c.kind, c.prof, c.roll); err != nil || got != c.want {
				t.Errorf("Apply = %d, %v; want %d", got, err, c.want)
			}
		})
	}
	for _, roll := range []int{0, 21, -3} {
		_, err := character.RogueReliableTalent{}.Apply(rl(7), d20.AbilityCheck, true, roll)
		wantInvalid(t, err, "roll "+strconv.Itoa(roll))
	}
}

// Requirement: doc/specs/class-rogue.md FR-081, FR-082, FR-083 (SRD 5.2.1, "Level 18: Elusive", p. 63) - from Rogue level 18 every
// Advantage against the character is dropped and every Disadvantage kept, unless Incapacitated; at level 17 nothing changes; the
// result resolves with modifier.Resolve, and the input slice is not changed.
// Path: per case (the Elusive cases): happy / unhappy (boundary, state error, rule interaction).
func TestRogueElusive(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.RollCircumstance(modifier.Advantage{}), modifier.RollCircumstance(modifier.Disadvantage{})
	for _, c := range []struct {
		name  string
		level int
		incap bool
		in    []modifier.RollCircumstance
		want  modifier.RollMode
	}{
		{"level 18 Advantage", 18, false, []modifier.RollCircumstance{adv}, modifier.Normal},
		{"level 17 Advantage", 17, false, []modifier.RollCircumstance{adv}, modifier.WithAdvantage},
		{"level 18 Advantage Incapacitated", 18, true, []modifier.RollCircumstance{adv}, modifier.WithAdvantage},
		{"level 18 both", 18, false, []modifier.RollCircumstance{adv, dis}, modifier.WithDisadvantage},
		{"level 18 Disadvantage", 18, false, []modifier.RollCircumstance{dis}, modifier.WithDisadvantage},
		{"level 20 two Advantages", 20, false, []modifier.RollCircumstance{adv, adv}, modifier.Normal},
		{"no circumstances", 18, false, nil, modifier.Normal},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := slices.Clone(c.in)
			got := character.RogueElusive{}.Apply(rl(c.level), c.incap, c.in)
			if mode := modifier.Resolve(got...); mode != c.want {
				t.Errorf("resolved mode = %v, want %v", mode, c.want)
			}
			if !slices.Equal(c.in, before) {
				t.Error("the input changed")
			}
			if len(got) > 0 {
				got[0] = nil
				if len(c.in) > 0 && c.in[0] == nil {
					t.Error("the result shares storage with the input")
				}
			}
		})
	}
}
