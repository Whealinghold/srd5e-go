package combat_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-granted-modifiers.md FR-093, FR-094, table R rows R3, R22 to R30 - Bless adds 1d4 and Bane subtracts 1d4 on an attack roll, a saving throw (any ability) and a Death Saving Throw of the creature they are on, and nothing to its ability checks, to another creature's roll or to a roll made against it (SRD 5.2.1, "Bless", p. 113; "Bane", p. 112; "Death Saving Throws", p. 17).
// Path: per case.
func TestBlessAndBaneDice(t *testing.T) {
	t.Parallel()
	bless, bane := smPlain(t, "bless", 0), smPlain(t, "bane", 0)
	plus, minus := smD4(), combat.MustBonusDie(dice.MustExpression(1, dice.D4(), 0), true)
	attack := combat.AttackRollContext{Attacker: cid("a"), Target: cid("m")}
	cases := []struct {
		name, row, path string
		sources         []combat.SpellModifierSource
		ctx             combat.RollContext
		want            []combat.BonusDie
	}{
		{"Bless, attack roll", "R22", "happy", []combat.SpellModifierSource{bless}, attack, []combat.BonusDie{plus}},
		{"Bless, saving throw", "R23", "happy", []combat.SpellModifierSource{bless}, combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Wisdom}, []combat.BonusDie{plus}},
		{"Bless, Death Saving Throw", "R3", "unhappy (rule interaction)", []combat.SpellModifierSource{bless}, combat.DeathSavingThrowContext{Creature: cid("a")}, []combat.BonusDie{plus}},
		{"Bane, attack roll", "R24", "happy", []combat.SpellModifierSource{bane}, attack, []combat.BonusDie{minus}},
		{"Bane, saving throw", "R25", "happy", []combat.SpellModifierSource{bane}, combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Dexterity}, []combat.BonusDie{minus}},
		{"Bane, Death Saving Throw", "R26", "unhappy (rule interaction)", []combat.SpellModifierSource{bane}, combat.DeathSavingThrowContext{Creature: cid("a")}, []combat.BonusDie{minus}},
		{"Bless and Bane on one creature add one die each", "R27", "unhappy (rule interaction)", []combat.SpellModifierSource{bless, smSrc(t, "bane", "d", "a", 1, combat.NoChoice{})}, attack, []combat.BonusDie{plus, minus}},
		{"Bless, ability check", "R28", "unhappy (boundary)", []combat.SpellModifierSource{bless}, combat.AbilityCheckContext{Creature: cid("a"), Ability: ability.Strength}, nil},
		{"Bless, another creature attacks", "R29", "unhappy (rule interaction)", []combat.SpellModifierSource{bless}, combat.AttackRollContext{Attacker: cid("m"), Target: cid("a")}, nil},
		{"Bless, another creature saves", "R29", "unhappy (rule interaction)", []combat.SpellModifierSource{bless}, combat.SavingThrowContext{Creature: cid("m"), Ability: ability.Wisdom}, nil},
		{"Bane, another creature's Death Saving Throw", "R30", "unhappy (rule interaction)", []combat.SpellModifierSource{bane}, combat.DeathSavingThrowContext{Creature: cid("m")}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.QueryRollContributions(c.sources, c.ctx)
			if err != nil || !slices.Equal(got.BonusDice(), c.want) || len(got.Modifiers()) != 0 || len(got.Circumstances()) != 0 {
				t.Errorf("contributions = %v %v %v, %v, want dice %v [%s%s]", got.BonusDice(), got.Modifiers(), got.Circumstances(), err, c.want, spellModifiersSpec, c.row)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-094, table R row R31 - Hex gives its creature Disadvantage on ability checks made with the ability its caster chose and on no other check, saving throw or attack roll, and the Disadvantage resolves with Advantage from another source as a whole (SRD 5.2.1, "Hex", p. 140).
// Path: per case.
func TestHexCheckDisadvantage(t *testing.T) {
	t.Parallel()
	hex := smSrc(t, "hex", "c", "a", 0, combat.ChosenAbility{Ability: ability.Strength})
	dis := []modifier.RollCircumstance{modifier.Disadvantage{}}
	check := func(who string, a ability.Ability) combat.RollContext {
		return combat.AbilityCheckContext{Creature: cid(who), Ability: a, Skill: content.SkillAthletics}
	}
	cases := []struct {
		name, row, path string
		ctx             combat.RollContext
		want            []modifier.RollCircumstance
	}{
		{"the chosen ability, with a skill", "R31", "happy", check("a", ability.Strength), dis},
		{"the chosen ability, raw", "R31", "happy", combat.AbilityCheckContext{Creature: cid("a"), Ability: ability.Strength}, dis},
		{"another ability", "R31", "unhappy (boundary)", check("a", ability.Dexterity), nil},
		{"a saving throw with the chosen ability", "R31", "unhappy (boundary)", combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Strength}, nil},
		{"an attack roll", "R31", "unhappy (boundary)", combat.AttackRollContext{Attacker: cid("a"), Target: cid("c")}, nil},
		{"the caster's check", "R31", "unhappy (rule interaction)", check("c", ability.Strength), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.QueryRollContributions([]combat.SpellModifierSource{hex}, c.ctx)
			if err != nil || !slices.Equal(got.Circumstances(), c.want) || len(got.BonusDice()) != 0 {
				t.Errorf("contributions = %v %v, %v, want %v [%sR31]", got.BonusDice(), got.Circumstances(), err, c.want, spellModifiersSpec)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-095, table D rows D14 to D19 - Hex adds 1d6 Necrotic and Hunter's Mark 1d6 Force to the damage of an attack roll of the caster that hits the creature, once each and in ascending CastOrder, and nothing for a spell's damage, other damage, another attacker or another target (SRD 5.2.1, "Hex", p. 140; "Hunter's Mark", p. 141).
// Path: per case.
func TestHexAndHuntersMarkExtraDamage(t *testing.T) {
	t.Parallel()
	hex := smSrc(t, "hex", "c", "m", 1, combat.ChosenAbility{Ability: ability.Wisdom})
	mark := smSrc(t, "hunters_mark", "c", "m", 0, combat.NoChoice{})
	necrotic := combat.MustDiceDamage(1, dice.D6(), content.DamageNecrotic, 0, false)
	force := combat.MustDiceDamage(1, dice.D6(), content.DamageForce, 0, false)
	cases := []struct {
		name, row, path string
		sources         []combat.SpellModifierSource
		attacker, tgt   string
		ev              combat.DamageEvent
		want            []combat.DamageComponent
	}{
		{"Hunter's Mark, attack roll hit", "D14", "happy", []combat.SpellModifierSource{mark}, "c", "m", combat.AttackRollHitEvent{}, []combat.DamageComponent{force}},
		{"Hex, attack roll hit", "D15", "happy", []combat.SpellModifierSource{hex}, "c", "m", combat.AttackRollHitEvent{}, []combat.DamageComponent{necrotic}},
		{"both, in CastOrder", "D16", "unhappy (rule interaction)", []combat.SpellModifierSource{hex, mark}, "c", "m", combat.AttackRollHitEvent{}, []combat.DamageComponent{force, necrotic}},
		{"a spell's damage", "D17", "unhappy (rule interaction)", []combat.SpellModifierSource{hex, mark}, "c", "m", combat.SpellDamageEvent{}, nil},
		{"other damage", "D17", "unhappy (rule interaction)", []combat.SpellModifierSource{hex, mark}, "c", "m", combat.OtherDamageEvent{}, nil},
		{"another attacker", "D18", "unhappy (rule interaction)", []combat.SpellModifierSource{hex, mark}, "d", "m", combat.AttackRollHitEvent{}, nil},
		{"another target", "D18", "unhappy (rule interaction)", []combat.SpellModifierSource{hex, mark}, "c", "x", combat.AttackRollHitEvent{}, nil},
		{"the marked creature hits the caster", "D19", "unhappy (rule interaction)", []combat.SpellModifierSource{hex, mark}, "m", "c", combat.AttackRollHitEvent{}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.QueryDamageRiders(c.sources, cid(c.attacker), cid(c.tgt), c.ev)
			if err != nil || !slices.Equal(got, c.want) {
				t.Errorf("riders = %v, %v, want %v [%s%s]", got, err, c.want, spellModifiersSpec, c.row)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-096, table C rows C5, C6, C13, C14 - the catalogue entry of Bless, Bane, Hex and Hunter's Mark carries its rider (the signed 1d4, Hex's check Disadvantage and 1d6 Necrotic, Hunter's Mark's 1d6 Force) and the other ten entries carry none.
// Path: per case.
func TestCatalogueRiders(t *testing.T) {
	t.Parallel()
	spells := []content.SpellID{"barkskin", "bestow_curse", "guidance", "haste", "longstrider", "mage_armor", "resistance", "shield", "shield_of_faith", "slow"}
	for _, id := range spells {
		k := combat.LookupSpellModifiers(id).(combat.SpellModifiersKnown)
		if !k.Entry().Rider().IsZero() {
			t.Errorf("%s carries a rider", id)
		}
	}
	rider := func(id content.SpellID) combat.SpellModifierRider {
		return combat.LookupSpellModifiers(id).(combat.SpellModifiersKnown).Entry().Rider()
	}
	if d, ok := rider("bless").RollDie(); !ok || d != smD4() {
		t.Errorf("Bless die = %v %v", d, ok)
	}
	if d, ok := rider("bane").RollDie(); !ok || !d.Negative() {
		t.Errorf("Bane die = %v %v", d, ok)
	}
	hex, mark := rider("hex"), rider("hunters_mark")
	if x, ok := hex.ExtraDamage(); !ok || x != combat.DamageComponent(combat.MustDiceDamage(1, dice.D6(), content.DamageNecrotic, 0, false)) || !hex.CheckDisadvantage() {
		t.Errorf("Hex rider = %v %v %v", x, ok, hex.CheckDisadvantage())
	}
	if x, ok := mark.ExtraDamage(); !ok || x != combat.DamageComponent(combat.MustDiceDamage(1, dice.D6(), content.DamageForce, 0, false)) || mark.CheckDisadvantage() {
		t.Errorf("Hunter's Mark rider = %v %v", x, ok)
	}
	if _, ok := hex.RollDie(); ok {
		t.Error("Hex has a roll die")
	}
	if _, ok := mark.RollDie(); ok {
		t.Error("Hunter's Mark has a roll die")
	}
	for _, id := range []content.SpellID{"bless", "bane"} {
		if _, ok := rider(id).ExtraDamage(); ok {
			t.Errorf("%s has extra damage", id)
		}
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-097, plan P10 - every Query function reads at most 64 sources on one creature: 64 are accepted, 65 are a LimitExceededError (matching srderr.ErrInvalidInput), and sources on other creatures do not count against the creature.
// Path: per case.
func TestSourcesPerTargetCap(t *testing.T) {
	t.Parallel()
	// The ids are valid spell ids that table C does not model, so n sources are legal and contribute nothing.
	many := func(n int, target string) []combat.SpellModifierSource {
		out := make([]combat.SpellModifierSource, n)
		for i := range out {
			out[i] = smSrc(t, content.SpellID(fmt.Sprintf("spell_%d", i)), "c", target, i, combat.NoChoice{})
		}
		return out
	}
	rc := combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Dexterity}
	queries := []struct {
		name string
		run  func([]combat.SpellModifierSource) error
	}{
		{"roll contributions", func(s []combat.SpellModifierSource) error { _, err := combat.QueryRollContributions(s, rc); return err }},
		{"damage riders", func(s []combat.SpellModifierSource) error {
			_, err := combat.QueryDamageRiders(s, cid("c"), cid("a"), combat.AttackRollHitEvent{})
			return err
		}},
		{"damage reduction", func(s []combat.SpellModifierSource) error {
			_, err := combat.QueryDamageReduction(s, cid("a"), content.DamageFire, false)
			return err
		}},
		{"armor class", func(s []combat.SpellModifierSource) error {
			_, err := combat.QueryArmorClass(s, cid("a"), false)
			return err
		}},
		{"speeds", func(s []combat.SpellModifierSource) error {
			speeds, err := combat.NewMovementSpeeds(30)
			if err != nil {
				return err
			}
			_, err = combat.QuerySpeeds(s, cid("a"), speeds)
			return err
		}},
	}
	for _, q := range queries {
		t.Run(q.name, func(t *testing.T) {
			t.Parallel()
			runCap(t, "spell modifier sources per target", 64, func(n int) error { return q.run(many(n, "a")) })
			if err := q.run(many(65, "a")); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("error %v does not match srderr.ErrInvalidInput", err)
			}
			twoTargets := append(many(64, "a"), many(64, "b")...)
			for i := 64; i < len(twoTargets); i++ {
				twoTargets[i] = smSrc(t, content.SpellID(fmt.Sprintf("other_%d", i)), "c", "b", i, combat.NoChoice{})
			}
			if err := q.run(twoTargets); err != nil {
				t.Errorf("64 sources on each of two creatures: error = %v, want none", err)
			}
		})
	}
}
