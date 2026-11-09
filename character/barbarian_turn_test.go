package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-barbarian.md FR-037 - the Rage end reasons have their names and the zero value is unknown; the attack kinds have theirs.
// Path: per case.
func TestBarbarianEnumNames(t *testing.T) {
	t.Parallel()
	for r, name := range map[character.RageEndReason]string{
		character.RageNotExtended: "not_extended", character.RageEndedByHeavyArmor: "heavy_armor_donned", character.RageEndedByIncapacitated: "incapacitated",
		character.RageEndedByUnconscious: "unconscious", character.RageEndedByTenMinutes: "ten_minutes_elapsed", 0: "unknown", 6: "unknown", -1: "unknown",
	} {
		if r.String() != name {
			t.Errorf("end reason %d = %q, want %q", int(r), r.String(), name)
		}
	}
	for k, name := range map[character.BarbarianAttackKind]string{character.BarbarianWeaponAttack: "weapon", character.BarbarianUnarmedStrike: "unarmed_strike", 0: "unknown", 3: "unknown", -1: "unknown"} {
		if k.String() != name {
			t.Errorf("attack kind %d = %q, want %q", int(k), k.String(), name)
		}
	}
	e := character.BarbarianExtraDamage{}
	if e.Count() != 0 || e.Sides() != 0 || e.DamageType() != content.DamageType(0) {
		t.Errorf("zero extra damage = %v", e)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-083, FR-097, FR-098, FR-125 - BarbarianTurnState is a value: a fresh turn has the Bonus Action and the Reaction and nothing declared, each transition changes only its flag and returns
// a new state, the next turn start clears Reckless Attack and every other flag, and the zero value has neither action; BarbarianHostFacts has neither (they live in the turn state only).
// Path: per case.
func TestBarbarianTurnStateTransitions(t *testing.T) {
	t.Parallel()
	fresh := character.NewBarbarianTurn()
	if fresh.RecklessDeclared() || fresh.FirstAttackRollMade() || fresh.FrenzyTargetHit() || !fresh.BonusActionAvailable() || !fresh.ReactionAvailable() {
		t.Errorf("fresh turn = %#v", fresh)
	}
	if zero := (character.BarbarianTurnState{}); zero.BonusActionAvailable() || zero.ReactionAvailable() {
		t.Errorf("zero turn = %#v", zero)
	}
	for name, got := range map[string]character.BarbarianTurnState{
		"reckless": fresh.DeclareReckless(), "attack roll": fresh.RecordAttackRoll(), "frenzy target": fresh.RecordFrenzyTarget(), "bonus action": fresh.SpendBonusAction(), "reaction": fresh.SpendReaction(),
	} {
		if got == fresh {
			t.Errorf("%s: the transition changed nothing", name)
		}
	}
	if !fresh.DeclareReckless().RecklessDeclared() || !fresh.RecordAttackRoll().FirstAttackRollMade() || !fresh.RecordFrenzyTarget().FrenzyTargetHit() ||
		fresh.SpendBonusAction().BonusActionAvailable() || !fresh.SpendBonusAction().ReactionAvailable() || fresh.SpendReaction().ReactionAvailable() || !fresh.SpendReaction().BonusActionAvailable() {
		t.Error("a transition changed the wrong flag")
	}
	spent := fresh.DeclareReckless().RecordAttackRoll().RecordFrenzyTarget().SpendBonusAction().SpendReaction()
	if spent.StartNextTurn() != fresh || fresh.RecklessDeclared() {
		t.Error("the next turn must start fresh without altering the old state")
	}
}
