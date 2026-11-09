package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

const berserker = character.BerserkerSubclassID

// Requirement: doc/specs/class-barbarian.md FR-077, FR-078, FR-122 and table 4.9.1 (FZ-01 to FZ-04) - Frenzy gives d6s equal to the Rage Damage bonus of the damage type of the attack to the first target hit
// with a Strength attack while raging after Reckless Attack, and records the target; it is refused in the order no Barbarian levels, level 3, Path of the Berserker, Rage, Reckless Attack, Strength, a hit, a second target.
// Path: per case (FZ-01 to FZ-04 and the refusals).
func TestBerserkerApplyFrenzy(t *testing.T) {
	t.Parallel()
	reckless := character.NewBarbarianTurn().DeclareReckless()
	dex := brbStrengthAttack()
	dex.UsesStrength = false
	miss := brbStrengthAttack()
	miss.Hit = false
	for _, c := range []struct {
		label  string
		level  int
		sub    content.SubclassID
		rage   bool
		turn   character.BarbarianTurnState
		attack character.BarbarianAttackFacts
		want   character.BarbarianReason
		dice   int
	}{
		{"FZ-01", 3, berserker, true, reckless, brbStrengthAttack(), 0, 2}, {"FZ-02", 9, berserker, true, reckless, brbStrengthAttack(), 0, 3}, {"FZ-03", 16, berserker, true, reckless, brbStrengthAttack(), 0, 4},
		{"FZ-04", 3, berserker, false, reckless, brbStrengthAttack(), character.BarbarianRageNotActive, 0},
		{"level 2", 2, berserker, true, reckless, brbStrengthAttack(), character.BarbarianFeatureNotGained, 0},
		{"another subclass", 3, "wild_heart", true, reckless, brbStrengthAttack(), character.BarbarianNotABerserker, 0},
		{"no subclass", 20, "", true, reckless, brbStrengthAttack(), character.BarbarianNotABerserker, 0},
		{"no Reckless Attack", 3, berserker, true, character.NewBarbarianTurn(), brbStrengthAttack(), character.BarbarianRecklessNotDeclared, 0},
		{"a Dexterity attack", 3, berserker, true, reckless, dex, character.BarbarianNotAStrengthAttack, 0},
		{"a miss", 3, berserker, true, reckless, miss, character.BarbarianAttackMissed, 0},
		{"a second target", 3, berserker, true, reckless.RecordFrenzyTarget(), brbStrengthAttack(), character.BarbarianFrenzyTargetAlreadyHit, 0},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			state := character.InitialBarbarianState()
			if c.rage {
				state = brbRaging(t, c.level, 0)
			}
			res := character.BerserkerRules{}.ApplyFrenzy(brbStats(t, c.level), c.sub, state, c.turn, c.attack)
			if c.want != 0 {
				if got, ok := res.(character.FrenzyNotApplicable); !ok || got.Reason != c.want {
					t.Errorf("= %#v, want not applicable %v", res, c.want)
				}
				return
			}
			got, ok := res.(character.FrenzyApplied)
			extra := got.ExtraDamage()
			if !ok || extra.Count() != c.dice || extra.Sides() != 6 || extra.DamageType() != content.DamageSlashing || !got.Turn().FrenzyTargetHit() || c.turn.FrenzyTargetHit() {
				t.Errorf("= %#v, want %dd6", res, c.dice)
			}
		})
	}
	if got := (character.BerserkerRules{}).ApplyFrenzy(character.BarbarianLevelStats{}, berserker, brbRaging(t, 3, 0), reckless, brbStrengthAttack()); got != (character.FrenzyNotApplicable{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("zero stats = %#v", got)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-080, FR-122 - Retaliation is offered from level 10 to the Path of the Berserker when the source is within 5 feet and the Reaction is available, and spends nothing; it is
// refused in the order no Barbarian levels, level, subclass, distance, Reaction.
// Path: per case (unhappy: boundary).
func TestBerserkerOfferRetaliation(t *testing.T) {
	t.Parallel()
	fresh := character.NewBarbarianTurn()
	for _, c := range []struct {
		name   string
		level  int
		sub    content.SubclassID
		turn   character.BarbarianTurnState
		within bool
		want   character.BarbarianReason
	}{
		{"level 10", 10, berserker, fresh, true, 0}, {"level 9", 9, berserker, fresh, true, character.BarbarianFeatureNotGained},
		{"another subclass", 10, "wild_heart", fresh, true, character.BarbarianNotABerserker}, {"farther than 5 feet", 10, berserker, fresh, false, character.BarbarianSourceNotWithinFiveFeet},
		{"Reaction spent", 10, berserker, fresh.SpendReaction(), true, character.BarbarianNoReaction},
	} {
		res := character.BerserkerRules{}.OfferRetaliation(brbStats(t, c.level), c.sub, c.turn, c.within)
		if c.want == 0 {
			if res != (character.RetaliationOffered{}) {
				t.Errorf("%s: %#v, want offered", c.name, res)
			}
		} else if res != (character.RetaliationNotOffered{Reason: c.want}) {
			t.Errorf("%s: %#v, want not offered %v", c.name, res, c.want)
		}
	}
	if got := (character.BerserkerRules{}).OfferRetaliation(character.BarbarianLevelStats{}, berserker, fresh, true); got != (character.RetaliationNotOffered{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("zero stats = %#v", got)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-081, FR-104, FR-120, FR-121 and table 4.9.2 (IP-01 to IP-05) - Intimidating Presence yields the DC 8 plus the Strength modifier plus the Proficiency Bonus of the total level,
// spends the Bonus Action and the use, produces IntimidatingPresenceUsed for the character, and is rejected with the first reason of no Barbarian levels, level, subclass, Incapacitated, no Bonus Action, not restored; a
// rejection spends nothing.
// Path: per case (IP-01 to IP-05 and the rejections).
func TestBerserkerUseIntimidatingPresence(t *testing.T) {
	t.Parallel()
	id := brbID(t)
	stats := brbStats(t, 14)
	for _, c := range []struct {
		label     string
		strScore  int
		prof, dc  int
		totalNote string
	}{{"IP-01", 16, 5, 16, ""}, {"IP-02", 20, 6, 19, ""}, {"IP-03", 8, 5, 12, ""}, {"IP-04", 10, 5, 13, ""}, {"IP-05", 16, 6, 17, "total level 20"}} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			start := character.NewBarbarianTurn()
			prof := character.ProficiencyBonusFor(character.MustLevel(map[int]int{5: 13, 6: 17}[c.prof]))
			res := must(character.BerserkerRules{}.UseIntimidatingPresence(id, stats, berserker, character.InitialBarbarianState(), start, character.BarbarianHostFacts{}, brbMod(t, c.strScore), prof))
			got, ok := res.(character.IntimidatingPresenceApplied)
			if !ok || got.DC() != c.dc || !got.State().IntimidatingPresenceUsed() || got.Turn().BonusActionAvailable() || !start.BonusActionAvailable() {
				t.Fatalf("= %#v, want applied with DC %d", res, c.dc)
			}
			ev, isUsed := got.Events()[0].(character.IntimidatingPresenceUsed)
			if len(got.Events()) != 1 || !isUsed || ev.DC() != c.dc || ev.CharacterID() != id {
				t.Errorf("events = %#v", got.Events())
			}
		})
	}
	prof := character.ProficiencyBonusFor(character.MustLevel(14))
	spentBA := character.NewBarbarianTurn().SpendBonusAction()
	used := must(character.BerserkerRules{}.UseIntimidatingPresence(id, stats, berserker, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, brbMod(t, 10), prof)).(character.IntimidatingPresenceApplied).State()
	for _, c := range []struct {
		name  string
		level int
		sub   content.SubclassID
		state character.BarbarianState
		turn  character.BarbarianTurnState
		facts character.BarbarianHostFacts
		want  character.BarbarianReason
	}{
		{"level 13", 13, berserker, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, character.BarbarianFeatureNotGained},
		{"another subclass", 14, "wild_heart", character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, character.BarbarianNotABerserker},
		{"Incapacitated", 14, berserker, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{Incapacitated: true}, character.BarbarianIncapacitated},
		{"no Bonus Action", 14, berserker, character.InitialBarbarianState(), spentBA, character.BarbarianHostFacts{}, character.BarbarianNoBonusAction},
		{"not restored", 14, berserker, used, character.NewBarbarianTurn(), character.BarbarianHostFacts{}, character.BarbarianPresenceNotRestored},
		{"Incapacitated beats the Bonus Action", 14, berserker, used, spentBA, character.BarbarianHostFacts{Incapacitated: true}, character.BarbarianIncapacitated},
		{"the Bonus Action beats not restored", 14, berserker, used, spentBA, character.BarbarianHostFacts{}, character.BarbarianNoBonusAction},
	} {
		res := must(character.BerserkerRules{}.UseIntimidatingPresence(id, brbStats(t, c.level), c.sub, c.state, c.turn, c.facts, brbMod(t, 10), prof))
		if res != (character.IntimidatingPresenceRejected{Reason: c.want}) {
			t.Errorf("%s: %#v, want rejected %v", c.name, res, c.want)
		}
	}
	zero := must(character.BerserkerRules{}.UseIntimidatingPresence(id, character.BarbarianLevelStats{}, berserker, used, spentBA, character.BarbarianHostFacts{}, brbMod(t, 10), prof))
	if zero != (character.IntimidatingPresenceRejected{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("zero stats = %#v", zero)
	}
	_, err := character.BerserkerRules{}.UseIntimidatingPresence(character.Identifier{}, stats, berserker, used, spentBA, character.BarbarianHostFacts{}, brbMod(t, 10), prof)
	wantInvalid(t, err, "zero id")
	_, err = character.BerserkerRules{}.UseIntimidatingPresence(id, stats, berserker, used, spentBA, character.BarbarianHostFacts{}, brbMod(t, 10), character.ProficiencyBonus{})
	wantInvalid(t, err, "zero Proficiency Bonus")
}

// Requirement: doc/specs/class-barbarian.md FR-082 - a spent Intimidating Presence is restored by expending a Rage use, with no action; the restoration is rejected when it was not used, when no Rage use remains,
// below level 14, for another subclass and with no Barbarian levels.
// Path: per case (unhappy: state error).
func TestBerserkerRestoreIntimidatingPresence(t *testing.T) {
	t.Parallel()
	id, stats := brbID(t), brbStats(t, 14)
	prof := character.ProficiencyBonusFor(character.MustLevel(14))
	used := must(character.BerserkerRules{}.UseIntimidatingPresence(id, stats, berserker, character.MustBarbarianState(1, 0), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, brbMod(t, 10), prof)).(character.IntimidatingPresenceApplied).State()
	got, ok := character.BerserkerRules{}.RestoreIntimidatingPresence(stats, berserker, used).(character.IntimidatingPresenceRestored)
	if !ok || got.State().IntimidatingPresenceUsed() || got.State().RagesExpended() != 2 {
		t.Fatalf("restoration = %#v, want a Rage use expended and the presence available", got)
	}
	empty := must(character.BerserkerRules{}.UseIntimidatingPresence(id, stats, berserker, character.MustBarbarianState(5, 0), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, brbMod(t, 10), prof)).(character.IntimidatingPresenceApplied).State()
	for _, c := range []struct {
		name  string
		level int
		sub   content.SubclassID
		state character.BarbarianState
		want  character.BarbarianReason
		zero  bool
	}{
		{"not used", 14, berserker, character.InitialBarbarianState(), character.BarbarianPresenceNotUsed, false},
		{"no Rage use remaining", 14, berserker, empty, character.BarbarianNoRageUsesRemaining, false},
		{"level 13", 13, berserker, used, character.BarbarianFeatureNotGained, false},
		{"another subclass", 14, "wild_heart", used, character.BarbarianNotABerserker, false},
		{"no Barbarian levels", 14, berserker, used, character.BarbarianNotABarbarian, true},
	} {
		s := brbStats(t, c.level)
		if c.zero {
			s = character.BarbarianLevelStats{}
		}
		if res := (character.BerserkerRules{}).RestoreIntimidatingPresence(s, c.sub, c.state); res != (character.IntimidatingPresenceRejected{Reason: c.want}) {
			t.Errorf("%s: %#v, want rejected %v", c.name, res, c.want)
		}
	}
}
