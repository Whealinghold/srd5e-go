package combat_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestConditionDefinitionsMatchTheSRDTable(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	always := combat.EffectConditionAlways
	sd := []ability.Ability{ability.Strength, ability.Dexterity}
	cases := []struct {
		name, requirement string
		kind              content.ConditionKind
		implies           []content.ConditionKind
		effects           []combat.ConditionEffect
	}{
		{"Blinded", `conditions.md FR-048, FR-050, FR-051; SRD 5.2.1, "Blinded", p. 177`, content.ConditionBlinded, nil,
			[]combat.ConditionEffect{combat.AutoFailAbilityChecks{Requiring: combat.SenseSight},
				combat.AttackRollsAgainst{Circumstance: adv, When: always}, combat.OwnAttackRolls{Circumstance: dis, When: always}}},
		{"Charmed", `conditions.md FR-052, FR-054; SRD 5.2.1, "Charmed", p. 178`, content.ConditionCharmed, nil,
			[]combat.ConditionEffect{combat.CharmerRestriction{}, combat.SocialAdvantageForSource{}}},
		{"Deafened", `conditions.md FR-055; SRD 5.2.1, "Deafened", p. 181`, content.ConditionDeafened, nil,
			[]combat.ConditionEffect{combat.AutoFailAbilityChecks{Requiring: combat.SenseHearing}}},
		{"Exhaustion", `conditions.md FR-041, FR-042; SRD 5.2.1, "Exhaustion", p. 181`, content.ConditionExhaustion, nil,
			[]combat.ConditionEffect{combat.D20TestReducedBy{PerLevel: 2}, combat.SpeedReducedBy{FeetPerLevel: 5}}},
		{"Frightened", `conditions.md FR-057, FR-059; SRD 5.2.1, "Frightened", p. 182`, content.ConditionFrightened, nil,
			[]combat.ConditionEffect{combat.OwnAbilityChecks{Circumstance: dis, When: combat.EffectConditionSourceInLineOfSight},
				combat.OwnAttackRolls{Circumstance: dis, When: combat.EffectConditionSourceInLineOfSight}, combat.CannotApproachSource{}}},
		{"Grappled", `conditions.md FR-060, FR-061; SRD 5.2.1, "Grappled", p. 182`, content.ConditionGrappled, nil,
			[]combat.ConditionEffect{combat.SpeedZero{}, combat.OwnAttackRolls{Circumstance: dis, When: combat.EffectConditionTargetIsNotGrappler}}},
		{"Incapacitated", `conditions.md FR-064 to FR-067; SRD 5.2.1, "Incapacitated", p. 184`, content.ConditionIncapacitated, nil,
			[]combat.ConditionEffect{combat.NoActions{}, combat.ConcentrationBroken{}, combat.CannotSpeak{}, combat.InitiativeCircumstance{Circumstance: dis}}},
		{"Invisible", `conditions.md FR-068 to FR-072; SRD 5.2.1, "Invisible", p. 184`, content.ConditionInvisible, nil,
			[]combat.ConditionEffect{combat.InitiativeCircumstance{Circumstance: adv}, combat.Concealed{},
				combat.AttackRollsAgainst{Circumstance: dis, When: combat.EffectConditionUnlessSeen}, combat.OwnAttackRolls{Circumstance: adv, When: combat.EffectConditionUnlessSeen}}},
		{"Paralyzed", `conditions.md FR-026, FR-073 to FR-077; SRD 5.2.1, "Paralyzed", p. 186`, content.ConditionParalyzed, condKinds(content.ConditionIncapacitated),
			[]combat.ConditionEffect{combat.SpeedZero{}, combat.SavingThrowsAutoFail{Abilities: sd},
				combat.AttackRollsAgainst{Circumstance: adv, When: always}, combat.AutomaticCriticalHit{WithinFeet: 5}}},
		{"Petrified", `conditions.md FR-026, FR-079 to FR-083, FR-023; SRD 5.2.1, "Petrified", p. 186`, content.ConditionPetrified, condKinds(content.ConditionIncapacitated),
			[]combat.ConditionEffect{combat.SpeedZero{}, combat.AttackRollsAgainst{Circumstance: adv, When: always},
				combat.SavingThrowsAutoFail{Abilities: sd}, combat.DamageResistanceAll{}, combat.ImmuneTo{Kind: content.ConditionPoisoned}}},
		{"Poisoned", `conditions.md FR-084, FR-085; SRD 5.2.1, "Poisoned", p. 186`, content.ConditionPoisoned, nil,
			[]combat.ConditionEffect{combat.OwnAttackRolls{Circumstance: dis, When: always}, combat.OwnAbilityChecks{Circumstance: dis, When: always}}},
		{"Prone", `conditions.md FR-088 to FR-090; SRD 5.2.1, "Prone", p. 186`, content.ConditionProne, nil,
			[]combat.ConditionEffect{combat.OwnAttackRolls{Circumstance: dis, When: always},
				combat.AttackRollsAgainst{Circumstance: adv, When: combat.EffectConditionAttackerWithin5Feet},
				combat.AttackRollsAgainst{Circumstance: dis, When: combat.EffectConditionAttackerBeyond5Feet}}},
		{"Restrained", `conditions.md FR-091 to FR-095; SRD 5.2.1, "Restrained", p. 187`, content.ConditionRestrained, nil,
			[]combat.ConditionEffect{combat.SpeedZero{}, combat.AttackRollsAgainst{Circumstance: adv, When: always},
				combat.OwnAttackRolls{Circumstance: dis, When: always}, combat.SavingThrowDisadvantage{Ability: ability.Dexterity}}},
		{"Stunned", `conditions.md FR-026, FR-096 to FR-098; SRD 5.2.1, "Stunned", p. 189`, content.ConditionStunned, condKinds(content.ConditionIncapacitated),
			[]combat.ConditionEffect{combat.SavingThrowsAutoFail{Abilities: sd}, combat.AttackRollsAgainst{Circumstance: adv, When: always}}},
		{"Unconscious", `conditions.md FR-026, FR-099 to FR-104; SRD 5.2.1, "Unconscious", p. 191`, content.ConditionUnconscious,
			condKinds(content.ConditionIncapacitated, content.ConditionProne),
			[]combat.ConditionEffect{combat.DropsHeldItems{}, combat.SpeedZero{}, combat.AttackRollsAgainst{Circumstance: adv, When: always},
				combat.SavingThrowsAutoFail{Abilities: sd}, combat.AutomaticCriticalHit{WithinFeet: 5}, combat.Unaware{}}},
	}
	if len(cases) != 15 {
		t.Fatalf("table has %d conditions, want 15", len(cases))
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := combat.ConditionDefinitionOf(c.kind)
			if err != nil {
				t.Fatal(err)
			}
			if d.Kind() != c.kind || !reflect.DeepEqual(d.Implies(), c.implies) || !reflect.DeepEqual(d.Effects(), c.effects) {
				t.Errorf("definition of %s = %v implies %v effects %v, want implies %v effects %v [%s]",
					c.name, d.Kind(), d.Implies(), d.Effects(), c.implies, c.effects, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-007, FR-014 - ConditionDefinitionOf rejects an invalid kind with an error
// matching ErrInvalidInput, and the slices of a definition are copies.
// Path: unhappy (immutability).
func TestConditionDefinitionOfRejectsAndCopies(t *testing.T) {
	t.Parallel()
	for _, k := range []content.ConditionKind{0, 16, -1} {
		if d, err := combat.ConditionDefinitionOf(k); !errors.Is(err, srderr.ErrInvalidInput) || d.Kind() != 0 {
			t.Errorf("ConditionDefinitionOf(%d) = %v, %v, want the zero definition and ErrInvalidInput", k, d, err)
		}
	}
	d, _ := combat.ConditionDefinitionOf(content.ConditionUnconscious)
	imp, eff := d.Implies(), d.Effects()
	imp[0], eff[0] = content.ConditionBlinded, combat.Unaware{}
	if d.Implies()[0] != content.ConditionIncapacitated || d.Effects()[0] != (combat.DropsHeldItems{}) {
		t.Errorf("a definition's slices alias its storage")
	}
}
