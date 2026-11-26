package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func isInvalid(err error) bool { return errors.Is(err, srderr.ErrInvalidInput) }

// Requirement: doc/specs/monster-traits.md FR-008 - every simple trait constructor accepts a part reference and rejects the zero one; the trait is comparable and keeps its part.
// Path: unhappy (invalid input).
func TestSimpleTraitConstructors(t *testing.T) {
	t.Parallel()
	ref := mtRef(t, 3)
	type mk func(combat.LimitedUsePartRef) (combat.MonsterTrait, error)
	cases := map[string]mk{
		"LegendaryResistance": func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) {
			return combat.NewLegendaryResistanceTrait(r)
		},
		"MagicResistance": func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) {
			return combat.NewMagicResistanceTrait(r)
		},
		"UndeadFortitude": func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) {
			return combat.NewUndeadFortitudeTrait(r)
		},
		"Swarm":        func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) { return combat.NewSwarmTrait(r) },
		"PackTactics":  func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) { return combat.NewPackTacticsTrait(r) },
		"Flyby":        func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) { return combat.NewFlybyTrait(r) },
		"Amorphous":    func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) { return combat.NewAmorphousTrait(r) },
		"SiegeMonster": func(r combat.LimitedUsePartRef) (combat.MonsterTrait, error) { return combat.NewSiegeMonsterTrait(r) },
	}
	for name, f := range cases {
		tr, err := f(ref)
		if err != nil || tr.IsZero() || tr.Part() != ref {
			t.Errorf("%s(%v) = %#v, %v", name, ref, tr, err)
		}
		zero, err := f(combat.LimitedUsePartRef{})
		if !isInvalid(err) || !zero.IsZero() {
			t.Errorf("%s(zero) = %#v, %v, want ErrInvalidInput and a zero trait", name, zero, err)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-007 - NewRegenerationTrait and NewParryTrait enforce 1 to the content maximum; Regeneration also rejects a bad Damage Type and the zero reference.
// Path: unhappy (boundary).
func TestRegenerationAndParryBounds(t *testing.T) {
	t.Parallel()
	ref := mtRef(t, 0)
	for _, c := range []struct {
		amount int
		ok     bool
	}{{0, false}, {1, true}, {content.MonsterMaxNumber, true}, {mtOverMax(), false}, {-1, false}} {
		_, rerr := combat.NewRegenerationTrait(ref, c.amount, nil)
		_, perr := combat.NewParryTrait(ref, c.amount)
		if (rerr == nil) != c.ok || (perr == nil) != c.ok || (!c.ok && (!isInvalid(rerr) || !isInvalid(perr))) {
			t.Errorf("amount %d: regeneration %v, parry %v, ok = %v", c.amount, rerr, perr, c.ok)
		}
	}
	if _, err := combat.NewRegenerationTrait(ref, 5, []content.DamageType{0}); !isInvalid(err) {
		t.Errorf("zero Damage Type err = %v", err)
	}
	if _, err := combat.NewRegenerationTrait(combat.LimitedUsePartRef{}, 5, nil); !isInvalid(err) {
		t.Errorf("zero reference err = %v", err)
	}
	if _, err := combat.NewParryTrait(combat.LimitedUsePartRef{}, 2); !isInvalid(err) {
		t.Errorf("parry zero reference err = %v", err)
	}
}

// Requirement: doc/specs/monster-traits.md FR-008, FR-041 - SuppressingTypes is a fresh copy in Damage Type order, a repeated type counts once, and equal traits are equal with ==.
// Path: unhappy (immutability).
func TestRegenerationSuppressingTypes(t *testing.T) {
	t.Parallel()
	tr := mtRegen(t, 15, content.DamageFire, content.DamageAcid, content.DamageFire, content.DamageThunder)
	got := tr.SuppressingTypes()
	if want := []content.DamageType{content.DamageAcid, content.DamageFire, content.DamageThunder}; !slices.Equal(got, want) {
		t.Fatalf("SuppressingTypes = %v, want %v", got, want)
	}
	got[0] = content.DamageCold
	if tr.SuppressingTypes()[0] != content.DamageAcid {
		t.Error("mutating the returned slice changed the trait")
	}
	if tr != mtRegen(t, 15, content.DamageThunder, content.DamageAcid, content.DamageFire) {
		t.Error("equal traits differ")
	}
	if tr == mtRegen(t, 15, content.DamageAcid, content.DamageFire) {
		t.Error("a different set compares equal")
	}
}

// Requirement: doc/specs/monster-traits.md FR-062 - WithSwarm marks a Regeneration and rejects a zero trait or swarm; the original is unchanged.
// Path: unhappy (rule interaction).
func TestRegenerationWithSwarm(t *testing.T) {
	t.Parallel()
	swarm := luMust(combat.NewSwarmTrait(mtRef(t, 1)))
	tr := mtOni(t)
	marked, err := tr.WithSwarm(swarm)
	if err != nil || !marked.SwarmCannotRegain() || tr.SwarmCannotRegain() {
		t.Errorf("WithSwarm = %v, %v; original marked = %v", marked, err, tr.SwarmCannotRegain())
	}
	if _, err := tr.WithSwarm(combat.SwarmTrait{}); !isInvalid(err) {
		t.Errorf("zero swarm err = %v", err)
	}
	if _, err := (combat.RegenerationTrait{}).WithSwarm(swarm); !isInvalid(err) {
		t.Errorf("zero trait err = %v", err)
	}
}

// Requirement: doc/specs/monster-traits.md FR-006 - NewDeathBurstTrait takes the saving throw of the part and rejects the zero one, a DC below 1 or above the content maximum, and the zero reference.
// Path: unhappy (boundary).
func TestNewDeathBurstTrait(t *testing.T) {
	t.Parallel()
	save := func(dc int) content.StatBlockSavingThrow {
		s, err := content.NewStatBlockSavingThrow(ability.Dexterity, dc, "", "")
		if err != nil {
			return content.StatBlockSavingThrow{}
		}
		return s
	}
	b, err := combat.NewDeathBurstTrait(mtRef(t, 0), save(11))
	if err != nil || b.DC() != 11 || b.Ability() != ability.Dexterity {
		t.Errorf("burst = %#v, %v", b, err)
	}
	for name, args := range map[string]struct {
		ref combat.LimitedUsePartRef
		s   content.StatBlockSavingThrow
	}{"zero save": {mtRef(t, 0), content.StatBlockSavingThrow{}}, "zero ref": {combat.LimitedUsePartRef{}, save(11)}} {
		if _, err := combat.NewDeathBurstTrait(args.ref, args.s); !isInvalid(err) {
			t.Errorf("%s err = %v", name, err)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-071 - a sunlight trait needs a valid scope and the scope names are stable.
// Path: unhappy (invalid input).
func TestNewSunlightPenaltyTrait(t *testing.T) {
	t.Parallel()
	for _, s := range []combat.SunlightPenaltyScope{0, 3} {
		if _, err := combat.NewSunlightPenaltyTrait(mtRef(t, 0), s); !isInvalid(err) {
			t.Errorf("scope %d err = %v", s, err)
		}
	}
	if _, err := combat.NewSunlightPenaltyTrait(combat.LimitedUsePartRef{}, combat.SunlightPenaltyAllD20Tests); !isInvalid(err) {
		t.Errorf("zero reference err = %v", err)
	}
	tr := luMust(combat.NewSunlightPenaltyTrait(mtRef(t, 0), combat.SunlightPenaltyAllD20Tests))
	if tr.Scope() != combat.SunlightPenaltyAllD20Tests {
		t.Errorf("Scope = %v", tr.Scope())
	}
	if combat.SunlightPenaltyAbilityChecksAndAttackRolls.String() != "ability_checks_and_attack_rolls" || combat.SunlightPenaltyAllD20Tests.String() != "all_d20_tests" || combat.SunlightPenaltyScope(0).String() != "unknown" {
		t.Error("scope names")
	}
}
