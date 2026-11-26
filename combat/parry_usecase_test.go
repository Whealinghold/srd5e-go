package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// parryChoice is the target's standing choice: Parry +2 with a Reaction it can spend, holding a weapon unless
// holding is false.
func parryChoice(tb testing.TB, holding bool) combat.ParryIfAble {
	tb.Helper()
	return combat.ParryIfAble{Trait: mtParryTrait(tb, 2), Reaction: combat.FeatureReaction{Feature: "parry", Granted: combat.NoStandardAction{}}, HoldingWeapon: holding}
}

// Requirement: doc/specs/combat-action-resolution.md FR-161, FR-091, 4.15.1 help used, Parry - the target's Parry on a melee hit: a hit that Parry turns into a miss spends the target's Reaction (ReactionTaken after the attacker's ledger events), is a miss without damage and is followed by ParryUsed; a hit the raised Armor Class still beats stands and is damaged; a natural 20 stays a Critical Hit; a Reaction the tracker refuses, an unavailable Parry (no weapon held, ranged attack, a miss) and NoParry leave the result unchanged and stop nothing.
// Path: per case.
func TestPerformAttackParry(t *testing.T) {
	t.Parallel()
	const spec = resSpec
	spent := func(t *testing.T) combat.Encounter {
		t.Helper()
		e := encABC(t)
		return resTake(t, e, combat.ActionLedger{}, "b", combat.SpendReaction{Choice: combat.FeatureReaction{Feature: "other", Granted: combat.NoStandardAction{}}}).Encounter()
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		choice                  combat.ParryChoice
		faces                   []int
		parry                   string
		kinds                   []string
		hit                     bool
	}{
		{"hit becomes a miss", spec + "FR-161", "unhappy (rule interaction)", encABC(t), parryChoice(t, true), []int{10, 4}, "applied, miss",
			[]string{"action_taken", "attack_made_in_attack_action", "reaction_taken", "attack_performed", "parry_used"}, false},
		{"hit stands against the raised class", spec + "FR-161", "unhappy (boundary)", encABC(t), parryChoice(t, true), []int{12, 4}, "applied, hit",
			[]string{"action_taken", "attack_made_in_attack_action", "reaction_taken", "attack_performed", "parry_used", "damage_taken"}, true},
		{"natural 20 stays a hit", spec + "FR-161; SRD 5.2.1, \"Rolling 20 or 1\", p. 7", "unhappy (boundary)", encABC(t), parryChoice(t, true), []int{20, 4, 4}, "applied, hit",
			[]string{"action_taken", "attack_made_in_attack_action", "reaction_taken", "attack_performed", "parry_used", "damage_taken"}, true},
		{"no weapon held", spec + "FR-161", "unhappy (rule interaction)", encABC(t), parryChoice(t, false), []int{10, 4}, "not used, not_holding_weapon",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken"}, true},
		{"attack missed", spec + "FR-161", "unhappy (rule interaction)", encABC(t), parryChoice(t, true), []int{2}, "not used, attack_missed",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed"}, false},
		{"Reaction already spent", spec + "FR-161", "unhappy (state)", spent(t), parryChoice(t, true), []int{10, 4}, "not used, refused",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken"}, true},
		{"NoParry", spec + "FR-161", "happy", encABC(t), combat.NoParry{}, []int{10, 4}, "not offered",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken"}, true},
		{"nil choice", spec + "FR-161", "happy", encABC(t), nil, []int{10, 4}, "not offered",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, c.e, "a", "b", resBegin())
			p.TargetParry = c.choice
			got, _ := resAttack(t, p, c.faces...)
			var label string
			switch v := got.Parry().(type) {
			case combat.ParryNotOffered:
				label = "not offered"
			case combat.ParryNotUsed:
				label = "not used, " + v.Unavailable.String()
				if v.Refused != nil {
					label = "not used, refused"
				}
			case combat.ParryApplied:
				label = "applied, hit"
				if v.Response.BecameMiss() {
					label = "applied, miss"
				}
			}
			if label != c.parry {
				t.Errorf("Parry = %q, want %q [%s]", label, c.parry, c.requirement)
			}
			if kinds := resKinds(got.Events()); !slices.Equal(kinds, c.kinds) {
				t.Errorf("events = %v, want %v [%s]", kinds, c.kinds, c.requirement)
			}
			if hasDamage := len(got.Resolutions()) > 0; hasDamage != c.hit {
				t.Errorf("damage applied = %v, want %v [%s]", hasDamage, c.hit, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-161, FR-009 - a Parry the host asks for with the zero trait is refused when the command is built, and a Reaction spent on Parry is the target's, not the attacker's: the attacker's own Reaction and the active combatant are unchanged.
// Path: unhappy (invalid input).
func TestPerformAttackParryChoiceIsValidated(t *testing.T) {
	t.Parallel()
	p := resParams(t, encABC(t), "a", "b", resBegin())
	p.TargetParry = combat.ParryIfAble{}
	if _, err := combat.NewPerformAttackCommand(p); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero trait: err = %v, want ErrInvalidInput [FR-161]", err)
	}
	p.TargetParry = struct{ combat.ParryChoice }{}
	if _, err := combat.NewPerformAttackCommand(p); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unknown choice: err = %v, want ErrInvalidInput [FR-161]", err)
	}
	p = resParams(t, encABC(t), "a", "b", resBegin())
	p.TargetParry = parryChoice(t, true)
	got, _ := resAttack(t, p, 10, 4)
	if got.Encounter().Tracker().Active().Combatant != cid("a") {
		t.Errorf("the active combatant changed: Parry is a Reaction of the target [FR-161]")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-159, FR-028 - Manacles on the attacker add one Disadvantage to its attack roll; a duplicate from a condition (Poisoned) changes nothing; a hold on another combatant adds none.
// Path: per case.
func TestPerformAttackGearHolds(t *testing.T) {
	t.Parallel()
	manacled := func(t *testing.T, who string) combat.GearHolds {
		t.Helper()
		in := holdIn("m", combat.HoldKindManacles)
		in.Target = cid(who)
		return gearHolds(t, gearHold(t, in))
	}
	poisoned, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Poisoned{}), resPC(t, "b", combat.SideB))
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		gear                    combat.GearHolds
		dice                    []string
	}{
		{"Manacles on the attacker", resSpec + "FR-159", "happy", encABC(t), manacled(t, "a"), []string{"2d20"}},
		{"Manacles on the target", resSpec + "FR-159", "unhappy (rule interaction)", encABC(t), manacled(t, "b"), []string{"1d20"}},
		{"Manacles and Poisoned", resSpec + "FR-028, FR-159", "unhappy (rule interaction)", poisoned, manacled(t, "a"), []string{"2d20"}},
		{"no holds", resSpec + "FR-159", "happy", encABC(t), combat.GearHolds{}, []string{"1d20"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, c.e, "a", "b", resBegin())
			p.Gear = c.gear
			_, got := resAttack(t, p, 2, 2)
			if len(got) == 0 || got[0] != c.dice[0] {
				t.Errorf("first die = %v, want %v [%s]", got, c.dice, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-078, FR-082, FR-091, FR-103, FR-134, 4.15.1 Concentration - an attack that hits a concentrating target rolls the save after the damage, the ConcentrationChecked comes last in the stream and is counted once in srd5e.combat.concentration_checks with its outcome; a miss and damage that does not land check nothing.
// Path: per case.
func TestPerformAttackConcentration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		faces                   []int
		outcome                 string
		kinds                   []string
		counted                 []string
	}{
		{"hit, save holds", resSpec + "FR-078, FR-103", "happy", []int{10, 3, 15}, "held DC 10",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken", "concentration_checked"}, []string{"held"}},
		{"hit, save fails", resSpec + "FR-078, FR-103", "unhappy (boundary)", []int{10, 3, 9}, "failed DC 10",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed", "damage_taken", "concentration_checked"}, []string{"broken"}},
		{"miss", resSpec + "FR-078 step 4, FR-081", "unhappy (empty and minimal)", []int{2}, "not checked",
			[]string{"action_taken", "attack_made_in_attack_action", "attack_performed"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, encABC(t), "a", "b", resBegin())
			p.TargetConcentration = concTracks(t, true, 0, 0)
			rec, opts := recording()
			cmd, err := combat.NewPerformAttackCommand(p)
			if err != nil {
				t.Fatal(err)
			}
			roller := dicetest.NewFixedRoller(c.faces...)
			uc, _ := combat.NewPerformAttack(roller, opts...)
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			got := resAs[combat.PerformAttackCompleted](t, res)
			if l := concLabel(got.Concentration()); l != c.outcome {
				t.Errorf("Concentration = %q, want %q [%s]", l, c.outcome, c.requirement)
			}
			if kinds := resKinds(got.Events()); !slices.Equal(kinds, c.kinds) {
				t.Errorf("events = %v, want %v [%s]", kinds, c.kinds, c.requirement)
			}
			counted := values(measured(rec, "srd5e.combat.concentration_checks"), "srd5e.combat.concentration_outcome")
			if !slices.Equal(counted, append([]string{}, c.counted...)) {
				t.Errorf("concentration_checks = %v, want %v [FR-103]", counted, c.counted)
			}
			resHygiene(t, rec, false)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-083, FR-095, FR-104 - a roller that fails at the Concentration save (after the attack and the damage were rolled) is a *srderr.PortError returned with no result, and the host's encounter, ledger and SpellcastingState are as they were: the same call with a working roller still gives its result.
// Path: unhappy (collaborator failure).
func TestPerformAttackConcentrationSaveRollerFailure(t *testing.T) {
	t.Parallel()
	p := resParams(t, encABC(t), "a", "b", resBegin())
	in := concTracks(t, true, 0, 0)
	p.TargetConcentration = in
	cmd, err := combat.NewPerformAttackCommand(p)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("no dice")
	calls := 0
	good := dicetest.NewFixedRoller(10, 3, 15)
	failing := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		if calls++; calls == 3 {
			return dice.Result{}, cause
		}
		return good.Roll(e)
	})
	uc, _ := combat.NewPerformAttack(failing)
	res, err := uc.Run(context.Background(), cmd)
	var pe *srderr.PortError
	if res != nil || !errors.As(err, &pe) || !errors.Is(err, cause) {
		t.Fatalf("= %v, %v, want nil and a *PortError wrapping the cause [FR-083]", res, err)
	}
	if _, going := in.State().Concentration(); !going || !p.Encounter.Equal(resParams(t, encABC(t), "a", "b", resBegin()).Encounter) {
		t.Errorf("the host's state changed [FR-083]")
	}
	if again, err := func() (any, error) {
		uc, _ := combat.NewPerformAttack(dicetest.NewFixedRoller(10, 3, 15))
		return uc.Run(context.Background(), cmd)
	}(); err != nil || again == nil {
		t.Errorf("a working roller after the failure gives %v, %v", again, err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-078, FR-161, FR-159 - the Damage option of an Unarmed Strike reads the same inputs: Concentration of the target after a hit, Parry, and Manacles on the attacker.
// Path: unhappy (rule interaction).
func TestPerformUnarmedStrikeParryGearAndConcentration(t *testing.T) {
	t.Parallel()
	damageOption := func(*testing.T) combat.UnarmedStrikeOption { return combat.UnarmedStrikeDamageOption{ArmorClass: 15} }
	e := encABC(t)
	p := resStrikeParams(t, e, "a", "b", damageOption(t))
	p.TargetConcentration = concTracks(t, true, 0, 0)
	got, dice := resStrike(t, p, 15, 15)
	if l := concLabel(got.Concentration()); l != "held DC 10" || !slices.Equal(dice, []string{"1d20", "1d20"}) {
		t.Errorf("Concentration = %q after dice %v, want held DC 10 after the attack and the save", l, dice)
	}
	p = resStrikeParams(t, e, "a", "b", damageOption(t))
	p.TargetParry = parryChoice(t, true)
	p.Gear = func() combat.GearHolds {
		in := holdIn("m", combat.HoldKindManacles)
		in.Target = cid("a")
		return gearHolds(t, gearHold(t, in))
	}()
	got, dice = resStrike(t, p, 2, 20)
	if dice[0] != "2d20" {
		t.Errorf("dice = %v, want the Manacles Disadvantage (2d20) [FR-159]", dice)
	}
	if _, ok := got.Parry().(combat.ParryNotUsed); !ok {
		t.Errorf("Parry = %#v, want not used: the attack missed [FR-161]", got.Parry())
	}
	grapple := resStrikeParams(t, e, "a", "b", grappleOption(t, ability.Strength, strengthSave(t)))
	grapple.TargetParry = parryChoice(t, true)
	if g, _ := resStrike(t, grapple, 5); g.Parry() != (combat.ParryNotOffered{}) {
		t.Errorf("Grapple Parry = %#v, want not offered: no attack roll [FR-161]", g.Parry())
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-161, FR-091 - a Damage option of an Unarmed Strike that Parry turns into a miss spends the target's free Reaction: ReactionTaken follows UnarmedStrikePerformed, the Parry is applied and becomes a miss, ParryUsed comes last and no DamageTaken follows.
// Path: unhappy (rule interaction).
func TestPerformUnarmedStrikeParryTurnsAHitIntoAMiss(t *testing.T) {
	t.Parallel()
	p := resStrikeParams(t, encABC(t), "a", "b", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	p.TargetParry = parryChoice(t, true)
	got, _ := resStrike(t, p, 10)
	applied, ok := got.Parry().(combat.ParryApplied)
	if !ok || !applied.Response.BecameMiss() {
		t.Fatalf("Parry = %#v, want applied and a miss [FR-161]", got.Parry())
	}
	want := []string{"action_taken", "attack_made_in_attack_action", "reaction_taken", "unarmed_strike_performed", "parry_used"}
	if kinds := resKinds(got.Events()); !slices.Equal(kinds, want) {
		t.Errorf("events = %v, want %v [FR-161]", kinds, want)
	}
	if len(got.Resolutions()) != 0 || resHP(t, got.Encounter(), "b") != resHP(t, p.Encounter, "b") {
		t.Errorf("a parried strike dealt damage [FR-161]")
	}
}
