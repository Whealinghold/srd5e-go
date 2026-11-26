package combat_test

import (
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-traits.md FR-060 - a Swarm cannot regain Hit Points or gain Temporary Hit Points, shares spaces and fits through a Tiny opening.
// Path: happy.
func TestSwarmRules(t *testing.T) {
	t.Parallel()
	r := luMust(combat.NewSwarmTrait(mtRef(t, 0))).Rules()
	if !r.CannotRegainHitPoints() || r.CanGainTemporaryHitPoints() || !r.SharesSpaces() || r.SmallestOpening() != content.SizeTiny {
		t.Errorf("rules = %+v", r)
	}
}

// Requirement: doc/specs/monster-traits.md FR-061 - healing and Temporary Hit Points are refused, a maximum reduction, a nil change and a zero trait are permitted.
// Path: per case.
func TestCheckSwarmChange(t *testing.T) {
	t.Parallel()
	sw := luMust(combat.NewSwarmTrait(mtRef(t, 0)))
	cases := []struct {
		name    string
		trait   combat.SwarmTrait
		change  combat.SwarmChange
		refusal combat.SwarmRefusal
		refused bool
	}{
		{"healing", sw, combat.SwarmHealing{}, combat.SwarmCannotRegainHitPoints, true},
		{"temporary", sw, combat.SwarmTemporaryHitPoints{}, combat.SwarmCannotGainTemporaryHitPoints, true},
		{"maximum reduction", sw, combat.SwarmMaximumReduction{}, 0, false},
		{"nil change", sw, nil, 0, false},
		{"zero trait", combat.SwarmTrait{}, combat.SwarmHealing{}, 0, false},
	}
	for _, c := range cases {
		if r, ok := combat.CheckSwarmChange(c.trait, c.change); r != c.refusal || ok != c.refused {
			t.Errorf("%s: %v, %v", c.name, r, ok)
		}
	}
	if combat.SwarmCannotRegainHitPoints.String() != "cannot_regain_hit_points" || combat.SwarmCannotGainTemporaryHitPoints.String() != "cannot_gain_temporary_hit_points" || combat.SwarmRefusal(0).String() != "unknown" {
		t.Error("refusal names")
	}
}

// Requirement: doc/specs/monster-traits.md FR-073 - Flyby exempts only a flying departure.
// Path: per case.
func TestFlybyExemptsFromOpportunityAttack(t *testing.T) {
	t.Parallel()
	fb := luMust(combat.NewFlybyTrait(mtRef(t, 0)))
	for _, m := range []combat.MovementMode{combat.MovementModeWalk, combat.MovementModeBurrow, combat.MovementModeClimb, combat.MovementModeCrawl, combat.MovementModeFly, combat.MovementModeSwim, 0} {
		if got := combat.FlybyExemptsFromOpportunityAttack(fb, m); got != (m == combat.MovementModeFly) {
			t.Errorf("%v = %v", m, got)
		}
	}
	if combat.FlybyExemptsFromOpportunityAttack(combat.FlybyTrait{}, combat.MovementModeFly) {
		t.Error("a zero trait exempts")
	}
}

// Requirement: doc/specs/monster-traits.md FR-072 - Amorphous is a fact: 1 inch, no extra movement.
// Path: happy.
func TestAmorphousFacts(t *testing.T) {
	t.Parallel()
	a := luMust(combat.NewAmorphousTrait(mtRef(t, 0)))
	if a.NarrowestSpaceInches() != 1 || a.NarrowOpeningCostsExtraMovement() {
		t.Error("amorphous facts")
	}
}

// Requirement: doc/specs/monster-traits.md FR-074 - Siege Monster doubles each amount and keeps the type for an object or structure, leaves a creature's damage equal, saturates, and rejects an empty list or a bad input.
// Path: per case.
func TestApplySiegeMonster(t *testing.T) {
	t.Parallel()
	sm := luMust(combat.NewSiegeMonsterTrait(mtRef(t, 0)))
	in := []combat.Damage{combat.MustDamage(7, content.DamageBludgeoning), combat.MustDamage(0, content.DamageFire), combat.MustDamage(math.MaxInt, content.DamageForce)}
	obj, err := combat.ApplySiegeMonster(sm, in, combat.DamageRecipientObjectOrStructure)
	want := []combat.Damage{combat.MustDamage(14, content.DamageBludgeoning), combat.MustDamage(0, content.DamageFire), combat.MustDamage(math.MaxInt, content.DamageForce)}
	if err != nil || !slices.Equal(obj, want) {
		t.Errorf("object = %v, %v", obj, err)
	}
	cre, err := combat.ApplySiegeMonster(sm, in, combat.DamageRecipientCreature)
	if err != nil || !slices.Equal(cre, in) {
		t.Errorf("creature = %v, %v", cre, err)
	}
	if in[0].Amount() != 7 {
		t.Error("the input changed")
	}
	cre[0] = combat.Damage{}
	if in[0].Amount() != 7 {
		t.Error("the result aliases the input")
	}
	for name, args := range map[string]struct {
		trait combat.SiegeMonsterTrait
		in    []combat.Damage
		r     combat.DamageRecipient
	}{
		"empty": {sm, nil, combat.DamageRecipientCreature}, "empty slice": {sm, []combat.Damage{}, combat.DamageRecipientCreature},
		"zero trait": {combat.SiegeMonsterTrait{}, in, combat.DamageRecipientCreature}, "bad recipient": {sm, in, 0},
		"zero Damage": {sm, []combat.Damage{{}}, combat.DamageRecipientCreature}, "recipient past the last": {sm, in, 3},
	} {
		if out, err := combat.ApplySiegeMonster(args.trait, args.in, args.r); !isInvalid(err) || out != nil {
			t.Errorf("%s: %v, %v", name, out, err)
		}
	}
	if combat.DamageRecipientCreature.String() != "creature" || combat.DamageRecipientObjectOrStructure.String() != "object_or_structure" || combat.DamageRecipient(0).String() != "unknown" {
		t.Error("recipient names")
	}
}
