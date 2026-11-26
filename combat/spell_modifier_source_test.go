package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-granted-modifiers.md FR-001, FR-084 - NewSpellModifierSource accepts the choice variant each spell takes and rejects a wrong, nil or invalid one, a blank spell, a zero caster or target and a negative CastOrder, always with an error matching srderr.ErrInvalidInput; a spell that is not modelled takes any valid choice (SRD 5.2.1, "Guidance", p. 138; "Hex", p. 140; "Resistance", p. 158; "Bestow Curse", pp. 112-113).
// Path: per case.
func TestNewSpellModifierSource(t *testing.T) {
	t.Parallel()
	curse := func(e combat.CurseEffect) combat.SpellModifierChoice { return combat.ChosenCurse{Effect: e} }
	cases := []struct {
		name, path string
		spell      content.SpellID
		caster     string
		target     string
		order      int
		choice     combat.SpellModifierChoice
		ok         bool
	}{
		{"Guidance with a skill", "happy", "guidance", "c", "a", 0, combat.ChosenSkill{Skill: content.SkillStealth}, true},
		{"Guidance without a skill", "unhappy (invalid input)", "guidance", "c", "a", 0, combat.NoChoice{}, false},
		{"Guidance with an invalid skill", "unhappy (invalid input)", "guidance", "c", "a", 0, combat.ChosenSkill{}, false},
		{"Haste with no choice", "happy", "haste", "c", "a", 0, combat.NoChoice{}, true},
		{"Haste with a skill", "unhappy (invalid input)", "haste", "c", "a", 0, combat.ChosenSkill{Skill: content.SkillStealth}, false},
		{"Hex with an ability", "happy", "hex", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength}, true},
		{"Hex with no choice", "unhappy (invalid input)", "hex", "c", "a", 0, combat.NoChoice{}, false},
		{"Hex with an invalid ability", "unhappy (invalid input)", "hex", "c", "a", 0, combat.ChosenAbility{}, false},
		{"Resistance with Fire", "happy", "resistance", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageFire}, true},
		{"Resistance with Force", "unhappy (boundary)", "resistance", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageForce}, false},
		{"Resistance with Psychic", "unhappy (boundary)", "resistance", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamagePsychic}, false},
		{"Resistance with Thunder, the last of its list", "unhappy (boundary)", "resistance", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageThunder}, true},
		{"Resistance with an invalid type", "unhappy (invalid input)", "resistance", "c", "a", 0, combat.ChosenResistanceType{}, false},
		{"Resistance with no choice", "unhappy (invalid input)", "resistance", "c", "a", 0, combat.NoChoice{}, false},
		{"Bestow Curse with an ability", "happy", "bestow_curse", "c", "a", 0, curse(combat.CurseAbility{Ability: ability.Wisdom}), true},
		{"Bestow Curse with attack rolls", "happy", "bestow_curse", "c", "a", 0, curse(combat.CurseAttackRolls{}), true},
		{"Bestow Curse with the Dodge", "happy", "bestow_curse", "c", "a", 0, curse(combat.CurseDodge{}), true},
		{"Bestow Curse with extra damage", "happy", "bestow_curse", "c", "a", 0, curse(combat.CurseExtraDamage{}), true},
		{"Bestow Curse with an invalid ability", "unhappy (invalid input)", "bestow_curse", "c", "a", 0, curse(combat.CurseAbility{}), false},
		{"Bestow Curse with no effect", "unhappy (invalid input)", "bestow_curse", "c", "a", 0, curse(nil), false},
		{"Bestow Curse with no choice", "unhappy (invalid input)", "bestow_curse", "c", "a", 0, combat.NoChoice{}, false},
		{"Slow with a curse", "unhappy (invalid input)", "slow", "c", "a", 0, curse(combat.CurseDodge{}), false},
		{"Haste with a damage type", "unhappy (invalid input)", "haste", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageFire}, false},
		{"Haste with an ability", "unhappy (invalid input)", "haste", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength}, false},
		{"nil choice", "unhappy (invalid input)", "haste", "c", "a", 0, nil, false},
		{"not modelled spell with no choice", "unhappy (missing content)", "fireball", "c", "a", 0, combat.NoChoice{}, true},
		{"not modelled spell with a skill", "unhappy (missing content)", "fireball", "c", "a", 0, combat.ChosenSkill{Skill: content.SkillStealth}, true},
		{"not modelled spell with an ability", "unhappy (missing content)", "fireball", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength}, true},
		{"not modelled spell with a damage type", "unhappy (missing content)", "fireball", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageForce}, true},
		{"not modelled spell with a curse", "unhappy (missing content)", "fireball", "c", "a", 0, curse(combat.CurseDodge{}), true},
		{"not modelled spell with an invalid skill", "unhappy (missing content)", "fireball", "c", "a", 0, combat.ChosenSkill{}, false},
		{"Barkskin with a skill", "unhappy (boundary)", "barkskin", "c", "a", 0, combat.ChosenSkill{Skill: content.SkillStealth}, false},
		{"Haste with a skill is rejected for a spell after Guidance", "unhappy (boundary)", "slow", "c", "a", 0, combat.ChosenSkill{Skill: content.SkillStealth}, false},
		{"Guidance's neighbour above Hex with an ability", "unhappy (boundary)", "guidance", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength}, false},
		{"Slow with an ability", "unhappy (boundary)", "slow", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength}, false},
		{"Haste with an ability is below Hex", "unhappy (boundary)", "haste", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength}, false},
		{"Haste with a resistance type", "unhappy (boundary)", "haste", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageFire}, false},
		{"Slow with a resistance type", "unhappy (boundary)", "slow", "c", "a", 0, combat.ChosenResistanceType{Type: content.DamageFire}, false},
		{"Barkskin with a curse", "unhappy (boundary)", "barkskin", "c", "a", 0, curse(combat.CurseDodge{}), false},
		{"Haste with a curse", "unhappy (boundary)", "haste", "c", "a", 0, curse(combat.CurseDodge{}), false},
		{"Guidance with a curse", "unhappy (boundary)", "guidance", "c", "a", 0, curse(combat.CurseDodge{}), false},
		{"blank spell", "unhappy (invalid input)", "", "c", "a", 0, combat.NoChoice{}, false},
		{"zero caster", "unhappy (invalid input)", "haste", "", "a", 0, combat.NoChoice{}, false},
		{"zero target", "unhappy (invalid input)", "haste", "c", "", 0, combat.NoChoice{}, false},
		{"negative CastOrder", "unhappy (boundary)", "haste", "c", "a", -1, combat.NoChoice{}, false},
		{"CastOrder 0", "unhappy (boundary)", "haste", "c", "a", 0, combat.NoChoice{}, true},
		{"caster is the target", "happy", "shield", "a", "a", 0, combat.NoChoice{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var caster, target combat.CombatantID
			if c.caster != "" {
				caster = cid(c.caster)
			}
			if c.target != "" {
				target = cid(c.target)
			}
			s, err := combat.NewSpellModifierSource(c.spell, caster, target, c.order, c.choice)
			if c.ok != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || c.ok == s.IsZero() {
				t.Fatalf("NewSpellModifierSource(%s, ...) = %+v, %v, want ok=%v [%s%s]", c.spell, s, err, c.ok, spellModifiersSpec, "FR-001")
			}
			if c.ok && (s.SpellID() != c.spell || s.CastOrder() != c.order || s.Choice() != c.choice || !s.CasterIs(caster) || !s.TargetIs(target)) {
				t.Errorf("accessors of %+v do not return the inputs", s)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-001 - MustSpellModifierSource panics with the error NewSpellModifierSource returns, and only for invalid input.
// Path: unhappy (invalid input).
func TestMustSpellModifierSource(t *testing.T) {
	t.Parallel()
	if s := combat.MustSpellModifierSource("haste", cid("c"), cid("a"), 0, combat.NoChoice{}); s.IsZero() {
		t.Error("MustSpellModifierSource returned the zero source")
	}
	defer func() {
		if err, ok := recover().(error); !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an error matching ErrInvalidInput", err)
		}
	}()
	combat.MustSpellModifierSource("haste", cid("c"), cid("a"), -1, combat.NoChoice{})
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-001 - the zero source is invalid; CasterIs and TargetIs answer false for the zero id and for another creature, and no accessor returns a CombatantID (combat-foundations FR-009).
// Path: unhappy (boundary).
func TestSpellModifierSourceCreatureTests(t *testing.T) {
	t.Parallel()
	var zero combat.SpellModifierSource
	if !zero.IsZero() || zero.CasterIs(combat.CombatantID("")) || zero.TargetIs(combat.CombatantID("")) {
		t.Error("the zero source must be zero and match no creature")
	}
	s := smSrc(t, "haste", "c", "a", 3, combat.NoChoice{})
	if !s.CasterIs(cid("c")) || s.CasterIs(cid("a")) || !s.TargetIs(cid("a")) || s.TargetIs(cid("c")) || s.CasterIs(combat.CombatantID("")) || s.TargetIs(combat.CombatantID("")) {
		t.Errorf("CasterIs/TargetIs wrong for %+v", s)
	}
}

// FuzzNewSpellModifierSource checks that no id, order or choice makes the constructor or a query panic, and that a returned source is never zero and
// an error always matches srderr.ErrInvalidInput.
func FuzzNewSpellModifierSource(f *testing.F) {
	f.Add("haste", "c", "a", 0, 0, 0)
	f.Add("", "", "", -1, 5, 99)
	f.Add("guidance", "c", "c", 9, 1, 3)
	f.Add("resistance", "x", "y", 1<<20, 3, 4)
	f.Fuzz(func(t *testing.T, spell, caster, target string, order, kind, arg int) {
		var choice combat.SpellModifierChoice
		switch kind % 6 {
		case 0:
			choice = combat.NoChoice{}
		case 1:
			choice = combat.ChosenSkill{Skill: content.Skill(arg)}
		case 2:
			choice = combat.ChosenAbility{Ability: ability.Ability(arg)}
		case 3:
			choice = combat.ChosenResistanceType{Type: content.DamageType(arg)}
		case 4:
			choice = combat.ChosenCurse{Effect: combat.CurseAbility{Ability: ability.Ability(arg)}}
		}
		var cc, tc combat.CombatantID
		if c, err := combat.NewCombatantID(caster); err == nil {
			cc = c
		}
		if c, err := combat.NewCombatantID(target); err == nil {
			tc = c
		}
		s, err := combat.NewSpellModifierSource(content.SpellID(spell), cc, tc, order, choice)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !s.IsZero() {
				t.Fatalf("NewSpellModifierSource = %+v, %v", s, err)
			}
			return
		}
		if s.IsZero() {
			t.Fatal("a valid source is zero")
		}
		if _, err := combat.QueryRollContributions([]combat.SpellModifierSource{s}, combat.DeathSavingThrowContext{Creature: tc}); err != nil {
			t.Fatalf("QueryRollContributions of one valid source = %v", err)
		}
	})
}
