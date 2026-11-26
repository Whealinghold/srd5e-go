package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func grappleOption(tb testing.TB, save ability.Ability, attempt d20.SaveAttempt) combat.UnarmedStrikeOption {
	tb.Helper()
	return combat.UnarmedStrikeGrappleOption{SaveAbility: save, Attempt: attempt}
}

// Requirement: doc/specs/combat-action-resolution.md FR-042, FR-046, FR-048, FR-049, 4.7.1 grapple, 4.15.1 - a Grapple spends the action, counts the attack, makes the target save against DC 13 (8 + 3 + 2); a failed save applies Grappled(attacker) through the encounter and prompts a review of the target's Speed, a made save applies nothing, and the result carries the escape DC for the host's grapple record.
// Path: per case.
func TestPerformUnarmedStrikeGrapple(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		face                    int
		events                  []string
		applied                 bool
	}{
		{"save fails", resSpec + "FR-046", "happy", 5, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "condition_applied", "movement_speeds_review_became_due"}, true},
		{"save made at the DC", resSpec + "FR-048, 4.7.1 (13 meets DC 13)", "unhappy (boundary)", 13, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed"}, false},
		{"save one below the DC", resSpec + "FR-046, 4.7.1 (12 against DC 13)", "unhappy (boundary)", 12, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "condition_applied", "movement_speeds_review_became_due"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, dice := resStrike(t, resStrikeParams(t, encABC(t), "a", "b", grappleOption(t, ability.Strength, strengthSave(t))), c.face)
			if !slices.Equal(resKinds(got.Events()), c.events) || !slices.Equal(dice, []string{"1d20"}) {
				t.Errorf("events %v, dice %v, want %v [%s]", resKinds(got.Events()), dice, c.events, c.requirement)
			}
			applied, ok := got.Effect().(combat.UnarmedEffectApplied)
			if ok != c.applied || (ok && applied.Condition() != (combat.Grappled{Grappler: cid("a")})) || resHas(t, got.Encounter(), "b", content.ConditionGrappled) != c.applied {
				t.Errorf("effect %#v, Grappled %v, want applied=%v [%s]", got.Effect(), resHas(t, got.Encounter(), "b", content.ConditionGrappled), c.applied, c.requirement)
			}
			if g := got.Strike().(combat.UnarmedStrikeGrappleResolved); g.DC() != 13 {
				t.Errorf("DC = %d, want 13 for the host's grapple record [FR-049]", g.DC())
			}
			up := got.Events()[2].(combat.UnarmedStrikePerformed).Outcome().(combat.UnarmedSavePerformed)
			if up.DC() != 13 || (up.Save().Outcome() == d20.Failure) != c.applied {
				t.Errorf("event outcome = %+v, want DC 13 and a save that matches the effect", up)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-046, 4.7.1 blocked - a failed save against a target that already has the condition (the first grappler is kept) or is Immune to it does not end the call and does not undo the roll: the outcome is UnarmedEffectBlocked with the encounter's rejection, and no condition event is produced.
// Path: per case.
func TestPerformUnarmedStrikeEffectBlocked(t *testing.T) {
	t.Parallel()
	shoveProne := combat.UnarmedStrikeShoveOption{Effect: combat.GiveProne, SaveAbility: ability.Strength, Attempt: strengthSave(t)}
	for _, c := range []struct {
		name, requirement string
		target            combat.Combatant
		option            combat.UnarmedStrikeOption
		want              combat.EncounterRejection
	}{
		{"already Grappled by another", resSpec + "FR-046, 4.7.1 blocked", resPC(t, "b", combat.SideB, combat.Grappled{Grappler: cid("g")}),
			grappleOption(t, ability.Strength, strengthSave(t)), combat.AlreadyHasConditionRejection{ID: cid("b"), Kind: content.ConditionGrappled}},
		{"Immune to Grappled", resSpec + "FR-046, 4.7.1 blocked", resImmune(t, "b", combat.SideB, content.ConditionGrappled),
			grappleOption(t, ability.Strength, strengthSave(t)), combat.ImmuneToConditionRejection{ID: cid("b"), Kind: content.ConditionGrappled}},
		{"already Prone", resSpec + "FR-047, 4.7.1 blocked", resPC(t, "b", combat.SideB, combat.Prone{}), shoveProne,
			combat.AlreadyHasConditionRejection{ID: cid("b"), Kind: content.ConditionProne}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, _ := encStart(t, resPC(t, "a", combat.SideA), c.target)
			got, _ := resStrike(t, resStrikeParams(t, e, "a", "b", c.option), 5)
			blocked, ok := got.Effect().(combat.UnarmedEffectBlocked)
			if want := []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed"}; !ok || blocked.Rejection() != c.want || !slices.Equal(resKinds(got.Events()), want) {
				t.Errorf("effect %#v, events %v, want blocked by %#v with no condition event [%s]", got.Effect(), resKinds(got.Events()), c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-045, FR-029, FR-030, FR-125, 4.7.1 forced results and Dexterity save - the target's save is the command's attempt adjusted by the target's conditions: Paralyzed fails a Strength save with no die, ChooseToFail stays a choice, Restrained gives Disadvantage to a Dexterity save only, Dodge gives Advantage to a Dexterity save only and not under a condition that zeroes the Speed, Exhaustion lowers the total.
// Path: per case.
func TestPerformUnarmedStrikeSaveAdjustments(t *testing.T) {
	t.Parallel()
	dodging := func(conds ...combat.Condition) combat.Encounter {
		e, _ := encStart(t, resPC(t, "b", combat.SideB, conds...), resPC(t, "a", combat.SideA))
		d := resDodge(t, e, combat.ActionLedger{}, "b")
		return resAdvance(t, d.Encounter(), d.Ledger()).Encounter()
	}
	pair := func(conds ...combat.Condition) combat.Encounter {
		e, _ := encStart(t, resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB, conds...))
		return e
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		option                  combat.UnarmedStrikeOption
		faces                   []int
		dice                    []string
		mode                    modifier.RollMode
		forced                  d20.ForcedFailureReason
		failed                  bool
	}{
		{"Paralyzed, Strength save: automatic failure, no die", resSpec + "FR-045, 4.7.1 forced", "unhappy (rule interaction)", pair(combat.Paralyzed{}),
			grappleOption(t, ability.Strength, strengthSave(t)), nil, nil, 0, d20.Automatic, true},
		{"ChooseToFail: no die", resSpec + "FR-045, 4.7.1 forced", "unhappy (rule interaction)", pair(), grappleOption(t, ability.Strength, d20.ChooseToFail{}), nil, nil, 0, d20.ChosenToFail, true},
		{"Paralyzed and ChooseToFail keeps the choice", resSpec + "FR-045", "unhappy (rule interaction)", pair(combat.Paralyzed{}), grappleOption(t, ability.Strength, d20.ChooseToFail{}), nil, nil, 0, d20.ChosenToFail, true},
		{"Restrained, Dexterity save: Disadvantage appended", resSpec + "FR-045, 4.7.1 Dexterity", "unhappy (rule interaction)", pair(combat.Restrained{}),
			grappleOption(t, ability.Dexterity, strengthSave(t)), []int{3, 18}, []string{"2d20"}, modifier.WithDisadvantage, 0, true},
		{"Restrained, Strength save: unaffected", resSpec + "FR-045", "unhappy (rule interaction)", pair(combat.Restrained{}), grappleOption(t, ability.Strength, strengthSave(t)),
			[]int{18}, []string{"1d20"}, modifier.Normal, 0, false},
		{"dodging, Dexterity save: Advantage appended", resSpec + "FR-029, 4.7.1 Dexterity", "unhappy (rule interaction)", dodging(), grappleOption(t, ability.Dexterity, strengthSave(t)),
			[]int{2, 18}, []string{"2d20"}, modifier.WithAdvantage, 0, false},
		{"dodging, Strength save: no Advantage", resSpec + "FR-029, 4.7.1", "unhappy (rule interaction)", dodging(), grappleOption(t, ability.Strength, strengthSave(t)),
			[]int{18}, []string{"1d20"}, modifier.Normal, 0, false},
		{"Restrained and dodging: the Speed 0 cancels Dodge, only Disadvantage", resSpec + "FR-125 over the 4.7.1 row that says both", "unhappy (rule interaction)", dodging(combat.Restrained{}),
			grappleOption(t, ability.Dexterity, strengthSave(t)), []int{18, 3}, []string{"2d20"}, modifier.WithDisadvantage, 0, true},
		{"Grappled and dodging with a stale profile Speed 30: no Advantage", resSpec + "FR-125, 4.7.1", "unhappy (rule interaction)", dodging(combat.Grappled{Grappler: cid("g")}),
			grappleOption(t, ability.Dexterity, strengthSave(t)), []int{18}, []string{"1d20"}, modifier.Normal, 0, false},
		{"Exhaustion 2: 13 less 4 fails the DC 13", resSpec + "FR-030, 4.7.1 Exhaustion", "unhappy (boundary)", pair(exhaustion(2)),
			grappleOption(t, ability.Strength, strengthSave(t)), []int{13}, []string{"1d20"}, modifier.Normal, 0, true},
		{"Exhaustion 2: 17 less 4 meets the DC 13", resSpec + "FR-030, 4.7.1 Exhaustion", "unhappy (boundary)", pair(exhaustion(2)),
			grappleOption(t, ability.Strength, strengthSave(t)), []int{17}, []string{"1d20"}, modifier.Normal, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, dice := resStrike(t, resStrikeParams(t, c.e, "a", "b", c.option), c.faces...)
			if !slices.Equal(dice, c.dice) {
				t.Errorf("dice = %v, want %v [%s]", dice, c.dice, c.requirement)
			}
			save := got.Strike().(combat.UnarmedStrikeGrappleResolved).Save()
			switch s := save.(type) {
			case combat.UnarmedStrikeSaveRolled:
				if s.Result().Roll().Mode() != c.mode {
					t.Errorf("mode = %v, want %v [%s]", s.Result().Roll().Mode(), c.mode, c.requirement)
				}
			case combat.UnarmedStrikeSaveForced:
				if s.Result().Reason() != c.forced {
					t.Errorf("forced for %v, want %v [%s]", s.Result().Reason(), c.forced, c.requirement)
				}
			}
			if (save.Outcome() == d20.Failure) != c.failed {
				t.Errorf("failed = %v, want %v [%s]", save.Outcome() == d20.Failure, c.failed, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-047, 4.7.1 shove - a failed Shove to Prone applies Prone, a failed Shove 5 feet changes no state and is recorded, a made save applies nothing; the Speed of a Prone target is not reviewed.
// Path: per case.
func TestPerformUnarmedStrikeShove(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		effect combat.ShoveEffect
		face   int
		events []string
		want   combat.UnarmedEffectOutcome
	}{
		{"to Prone", combat.GiveProne, 5, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "condition_applied"}, combat.UnarmedEffectApplied{}},
		{"5 feet", combat.PushFiveFeet, 5, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed"}, combat.UnarmedEffectPushRecorded{}},
		{"save made", combat.GiveProne, 15, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed"}, combat.UnarmedEffectNotTriggered{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := encABC(t)
			got, _ := resStrike(t, resStrikeParams(t, e, "a", "b", combat.UnarmedStrikeShoveOption{Effect: c.effect, SaveAbility: ability.Strength, Attempt: strengthSave(t)}), c.face)
			if !slices.Equal(resKinds(got.Events()), c.events) || reflectTypeOf(got.Effect()) != reflectTypeOf(c.want) {
				t.Errorf("events %v, effect %#v, want %v and %T [%sFR-047]", resKinds(got.Events()), got.Effect(), c.events, c.want, resSpec)
			}
			if prone := resHas(t, got.Encounter(), "b", content.ConditionProne); prone != (c.effect == combat.GiveProne && c.face == 5) {
				t.Errorf("Prone = %v", prone)
			}
			for _, id := range []string{"a", "b", "c"} {
				if c.effect == combat.PushFiveFeet && !encMember(t, e, id).Equal(encMember(t, got.Encounter(), id)) {
					t.Errorf("a push changed combatant %s [FR-047]", id)
				}
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-042, 4.7.1 size and hands - a Grapple or Shove is refused before any die and nothing is spent when the target is two sizes larger or the attacker has no free hand (a target one size larger is allowed).
// Path: per case.
func TestPerformUnarmedStrikeNotPossible(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		edit              func(*combat.ResolveUnarmedStrikeParams)
		want              combat.AttackNotPossibleReason
		ok                bool
	}{
		{"two sizes larger", resSpec + "4.7.1 size and hands", func(p *combat.ResolveUnarmedStrikeParams) { p.TargetSize = content.SizeHuge }, combat.AttackNotPossibleReasonTargetTooLarge, false},
		{"one size larger", resSpec + "4.7.1 size and hands", func(p *combat.ResolveUnarmedStrikeParams) { p.TargetSize = content.SizeLarge }, 0, true},
		{"no free hand", resSpec + "4.7.1 size and hands", func(p *combat.ResolveUnarmedStrikeParams) { p.Hand = combat.HandAvailabilityNoHandFree }, combat.AttackNotPossibleReasonNoFreeHand, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := encABC(t)
			sp := unarmedParams(grappleOption(t, ability.Strength, strengthSave(t)))
			c.edit(&sp)
			p := resStrikeParams(t, e, "a", "b", sp.Option)
			p.Strike = unarmedCommand(t, sp)
			cmd, err := combat.NewPerformUnarmedStrikeCommand(p)
			if err != nil {
				t.Fatal(err)
			}
			roller := newFixed(5)
			uc, _ := combat.NewPerformUnarmedStrike(roller)
			res, err := uc.Run(ctxBackground(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if c.ok {
				resAs[combat.PerformUnarmedStrikeCompleted](t, res)
				return
			}
			resWant(t, resAs[combat.PerformUnarmedStrikeRejected](t, res), e, combat.ActionLedger{}, combat.AttackNotPossibleRejection{Reason: c.want}, roller)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-043, FR-044, FR-031, 4.7 - the Damage option rolls an attack with its conditions derived as an attack's, and a hit deals 1 plus the Strength modifier Bludgeoning with no die: a hit of 4, a Critical Hit of the same 4 (no die to double), a miss, Disadvantage from a Poisoned attacker, an automatic Critical Hit and Advantage against a Paralyzed target, and a knock-out that leaves the target at 1 Hit Point.
// Path: per case.
func TestPerformUnarmedStrikeDamageOption(t *testing.T) {
	t.Parallel()
	damage := combat.UnarmedStrikeDamageOption{ArmorClass: 15}
	pair := func(a, b []combat.Condition, hp int) combat.Encounter {
		e, _ := encStart(t, resPC(t, "a", combat.SideA, a...), encC(t, "b", combat.SideB, pcStatus(t, hp, 20, 0), b...))
		return e
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		knock                   combat.KnockOutChoice
		faces                   []int
		dice                    []string
		hp                      int
		events                  []string
	}{
		{"hit", resSpec + "FR-043", "happy", pair(nil, nil, 20), combat.NoKnockOut, []int{14}, []string{"1d20"}, 16,
			[]string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "damage_taken"}},
		{"Critical Hit has no die to double", resSpec + "FR-043, attacks.md FR-067", "unhappy (boundary)", pair(nil, nil, 20), combat.NoKnockOut, []int{20}, []string{"1d20"}, 16,
			[]string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "damage_taken"}},
		{"miss", resSpec + "FR-023", "unhappy (boundary)", pair(nil, nil, 20), combat.NoKnockOut, []int{9}, []string{"1d20"}, 20,
			[]string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed"}},
		{"Poisoned attacker: Disadvantage", resSpec + "FR-044", "unhappy (rule interaction)", pair([]combat.Condition{combat.Poisoned{}}, nil, 20), combat.NoKnockOut, []int{20, 2}, []string{"2d20"}, 20,
			[]string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed"}},
		{"Paralyzed target: Advantage and an automatic Critical Hit", resSpec + "FR-044, FR-031", "unhappy (rule interaction)", pair(nil, []combat.Condition{combat.Paralyzed{}}, 20), combat.NoKnockOut,
			[]int{2, 14}, []string{"2d20"}, 16, []string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "damage_taken"}},
		{"knock-out to 1 Hit Point", resSpec + "FR-043, FR-037", "unhappy (rule interaction)", pair(nil, nil, 3), combat.KnockOut, []int{14}, []string{"1d20"}, 1,
			[]string{"action_taken", "attack_made_in_attack_action", "unarmed_strike_performed", "damage_taken", "knocked_out", "condition_applied", "movement_speeds_review_became_due"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resStrikeParams(t, c.e, "a", "b", damage)
			p.KnockOut = c.knock
			got, dice := resStrike(t, p, c.faces...)
			if !slices.Equal(dice, c.dice) || resHP(t, got.Encounter(), "b") != c.hp || !slices.Equal(resKinds(got.Events()), c.events) {
				t.Errorf("dice %v, target %d, events %v; want %v, %d, %v [%s]", dice, resHP(t, got.Encounter(), "b"), resKinds(got.Events()), c.dice, c.hp, c.events, c.requirement)
			}
			if got.Effect() != (combat.UnarmedEffectNotTriggered{}) {
				t.Errorf("effect = %#v, want none for the Damage option", got.Effect())
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-050, FR-012 - a dead target stops the call before the save; an unknown target is an invalid result.
// Path: unhappy (state).
func TestPerformUnarmedStrikeStops(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	roller := newFixed(5)
	uc, _ := combat.NewPerformUnarmedStrike(roller)
	cmd, _ := combat.NewPerformUnarmedStrikeCommand(resStrikeParams(t, e, "a", "b", grappleOption(t, ability.Strength, strengthSave(t))))
	res, err := uc.Run(ctxBackground(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.PerformUnarmedStrikeRejected](t, res), e, combat.ActionLedger{}, combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, roller)
	cmd, _ = combat.NewPerformUnarmedStrikeCommand(resStrikeParams(t, e, "a", "ghost", grappleOption(t, ability.Strength, strengthSave(t))))
	res, err = uc.Run(ctxBackground(), cmd)
	if inv := resAs[combat.PerformUnarmedStrikeInvalid](t, res); err != nil || inv.Problem() != (combat.UnknownCombatantProblem{ID: cid("ghost")}) {
		t.Errorf("= %#v, %v, want an invalid result naming ghost", res, err)
	}
}
