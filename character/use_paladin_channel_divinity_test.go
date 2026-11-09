package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

func useChannelDivinity(t *testing.T, opts ...character.Option) character.UsePaladinChannelDivinity {
	t.Helper()
	return must(character.NewUsePaladinChannelDivinity(pldProgression(t), opts...))
}

const oath = character.OathOfDevotionSubclassID

// sacredCmd is a Sacred Weapon command: the Oath, the Attack action and a held weapon.
func sacredCmd(c character.Character) character.UsePaladinChannelDivinityCommand {
	return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelSacredWeapon, Subclass: oath, Charisma: sacredCharisma,
		SpellSaveDC: sacredDC, TakingAttackAction: true, HoldingMeleeWeapon: true, Weapon: "longsword"}
}

var (
	aWisdom        = ability.Wisdom
	sacredCharisma = ability.MustScore(16).Modifier()
	sacredDC       = character.Stats(character.CasterPaladin, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5))).SaveDC()
)

// Requirement: doc/specs/class-paladin.md FR-065 to FR-070, FR-073, FR-074 (SRD 5.2.1, "Level 3: Channel Divinity", p. 54; "Level 9: Abjure Foes", p. 55; "Level 3: Sacred Weapon", p. 56) - UsePaladinChannelDivinity spends exactly one
// use, returns the effect data (Divine Sense; Abjure Foes with the larger of 1 and the Charisma modifier as targets and the supplied DC; Sacred Weapon with the larger of 1 and the Charisma modifier as the bonus) and one
// PaladinChannelDivinityUsed event, and keeps the Lay On Hands pool and the free casts.
// Path: per case (Tables B and W): happy / unhappy (boundary).
func TestUsePaladinChannelDivinityEffects(t *testing.T) {
	t.Parallel()
	dc := dcOf(t, 15)
	for _, c := range []struct {
		label string
		cha   int
		want  int
	}{{"lowest modifier", -5, 1}, {"negative", -1, 1}, {"zero", 0, 1}, {"one", 1, 1}, {"two", 2, 2}, {"three", 3, 3}, {"five", 5, 5}, {"highest modifier", 10, 10}} {
		cha := modifierOf(t, c.cha)
		res, err := useChannelDivinity(t).Run(context.Background(), character.UsePaladinChannelDivinityCommand{Character: pldPaladin(t, 9), Option: character.PaladinChannelAbjureFoes, Charisma: cha, SpellSaveDC: dc})
		a, ok := res.(character.UsePaladinChannelDivinityApplied)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.label, res, err)
		}
		abjure := a.Effect().(character.AbjureFoesEffect)
		ability, gotDC := abjure.SavingThrow()
		if abjure.Targets() != c.want || ability != aWisdom || gotDC != dc {
			t.Errorf("%s: Abjure Foes = %d targets, %v, DC %v; want %d, wisdom, %v", c.label, abjure.Targets(), ability, gotDC, c.want, dc)
		}
		sw, err := useChannelDivinity(t).Run(context.Background(), character.UsePaladinChannelDivinityCommand{Character: pldPaladin(t, 9), Option: character.PaladinChannelSacredWeapon, Subclass: oath, Charisma: cha, SpellSaveDC: dc,
			TakingAttackAction: true, HoldingMeleeWeapon: true, Weapon: "longsword"})
		sa, ok := sw.(character.UsePaladinChannelDivinityApplied)
		if err != nil || !ok || sa.Effect().(character.SacredWeaponEffect).AttackBonus() != c.want || sa.Effect().(character.SacredWeaponEffect).Weapon() != "longsword" {
			t.Errorf("%s: Sacred Weapon = %#v, %v", c.label, sw, err)
		}
	}
	c := pldPaladin(t, 3)
	res := must(useChannelDivinity(t).Run(context.Background(), character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelDivineSense, Charisma: modifierOf(t, 2), SpellSaveDC: dc}))
	a := res.(character.UsePaladinChannelDivinityApplied)
	s := pldStateOf(t, a.Character())
	ev, ok := a.Events()[0].(character.PaladinChannelDivinityUsed)
	if _, isSense := a.Effect().(character.DivineSenseEffect); !isSense || s.ChannelDivinityRemaining() != 1 || !s.DivineSenseActive() || s.LayOnHandsRemaining() != 15 || !s.Equal(a.State()) ||
		!ok || len(a.Events()) != 1 || ev.Option() != "divine_sense" || ev.Remaining() != 1 || ev.CharacterID() != c.ID() {
		t.Errorf("Divine Sense: state %+v, effect %#v, events %#v", s, a.Effect(), a.Events())
	}
	if _, replaced := a.ReplacedWeapon(); replaced {
		t.Error("Divine Sense replaced a weapon")
	}
}

// Requirement: doc/specs/class-paladin.md FR-065, FR-068, FR-069, FR-073 - the use is rejected with every reason that holds in declaration order: feature_not_gained below level 3, option_not_available (Abjure Foes at 8, Sacred Weapon
// without the Oath), not_taking_attack_action and no_melee_weapon_held for Sacred Weapon, no_uses_remaining at 0; the Character is unchanged and no event is emitted.
// Path: per case (unhappy: boundary levels 2/3 and 8/9, state error).
func TestUsePaladinChannelDivinityRejected(t *testing.T) {
	t.Parallel()
	dc := dcOf(t, 15)
	none := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25})
	for _, c := range []struct {
		name string
		char character.Character
		cmd  func(character.Character) character.UsePaladinChannelDivinityCommand
		want []string
	}{
		{"Divine Sense at level 2", pldPaladin(t, 2), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelDivineSense, SpellSaveDC: dc}
		}, []string{"feature_not_gained", "no_uses_remaining"}},
		{"Divine Sense at level 3", pldPaladin(t, 3), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelDivineSense, SpellSaveDC: dc}
		}, nil},
		{"Abjure Foes at level 8", pldPaladin(t, 8), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelAbjureFoes, SpellSaveDC: dc}
		}, []string{"option_not_available"}},
		{"Abjure Foes at level 9", pldPaladin(t, 9), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelAbjureFoes, SpellSaveDC: dc}
		}, nil},
		{"Sacred Weapon without the Oath", pldPaladin(t, 5), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			cmd := sacredCmd(c)
			cmd.Subclass = ""
			return cmd
		}, []string{"option_not_available"}},
		{"Sacred Weapon without the Attack action", pldPaladin(t, 5), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			cmd := sacredCmd(c)
			cmd.TakingAttackAction = false
			return cmd
		}, []string{"not_taking_attack_action"}},
		{"Sacred Weapon without a weapon", pldPaladin(t, 5), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			cmd := sacredCmd(c)
			cmd.HoldingMeleeWeapon, cmd.Weapon = false, ""
			return cmd
		}, []string{"no_melee_weapon_held"}},
		{"Sacred Weapon, everything wrong", pldWith(t, pldPaladin(t, 5), none), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelSacredWeapon}
		}, []string{"option_not_available", "not_taking_attack_action", "no_melee_weapon_held", "no_uses_remaining"}},
		{"no uses remaining", pldWith(t, pldPaladin(t, 5), none), func(c character.Character) character.UsePaladinChannelDivinityCommand {
			return character.UsePaladinChannelDivinityCommand{Character: c, Option: character.PaladinChannelDivineSense, SpellSaveDC: dc}
		}, []string{"no_uses_remaining"}},
	} {
		before := c.char
		res, err := useChannelDivinity(t).Run(context.Background(), c.cmd(c.char))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !c.char.Equal(before) {
			t.Errorf("%s: the input changed", c.name)
		}
		if len(c.want) == 0 {
			if _, ok := res.(character.UsePaladinChannelDivinityApplied); !ok {
				t.Errorf("%s: result = %#v, want applied", c.name, res)
			}
			continue
		}
		rj, ok := res.(character.UsePaladinChannelDivinityRejected)
		if !ok || !slices.Equal(pldReasons(rj.Reasons()), c.want) {
			t.Errorf("%s: result = %#v, want reasons %v", c.name, res, c.want)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-075 (SRD 5.2.1, "Level 3: Sacred Weapon", p. 56) - using Sacred Weapon while it is active spends another use, replaces the earlier imbuement and ends the earlier weapon's effect with
// reason used_again, leaving exactly one imbued weapon.
// Path: unhappy (rule interaction).
func TestUsePaladinChannelDivinitySacredWeaponReplaced(t *testing.T) {
	t.Parallel()
	first := must(useChannelDivinity(t).Run(context.Background(), sacredCmd(pldPaladin(t, 5)))).(character.UsePaladinChannelDivinityApplied)
	if w, ok := pldStateOf(t, first.Character()).SacredWeaponActive(); !ok || w != "longsword" || len(first.Events()) != 1 {
		t.Fatalf("first use: weapon %q, events %v", w, first.Events())
	}
	cmd := sacredCmd(first.Character())
	cmd.Weapon = "mace"
	second := must(useChannelDivinity(t).Run(context.Background(), cmd)).(character.UsePaladinChannelDivinityApplied)
	s := pldStateOf(t, second.Character())
	if w, _ := s.SacredWeaponActive(); w != "mace" || s.ChannelDivinityRemaining() != 0 {
		t.Errorf("second use: weapon %q, remaining %d", w, s.ChannelDivinityRemaining())
	}
	if old, ok := second.ReplacedWeapon(); !ok || old != "longsword" {
		t.Errorf("ReplacedWeapon = %q, %v", old, ok)
	}
	ended, ok := second.Events()[0].(character.PaladinEffectEnded)
	if len(second.Events()) != 2 || !ok || ended.Effect() != character.PaladinEffectSacredWeapon || ended.Reason() != character.PaladinEndUsedAgain {
		t.Errorf("events = %#v", second.Events())
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-105, FR-109, FR-115 - NotAPaladin for a Fighter; an undefined option, a Sacred Weapon held without a weapon key, a zero Character or use case and a cancelled context are errors before
// any signal; a recording telemetry gives the same result and the span names the option.
// Path: per case (unhappy: invalid input).
func TestUsePaladinChannelDivinityEdges(t *testing.T) {
	t.Parallel()
	for option, feature := range map[character.PaladinChannelDivinityOption]string{character.PaladinChannelDivineSense: "divine_sense", character.PaladinChannelAbjureFoes: "abjure_foes", character.PaladinChannelSacredWeapon: "sacred_weapon"} {
		rec, opts := fgtTelemetry()
		cmd := sacredCmd(pldPaladin(t, 9))
		cmd.Option = option
		res, err := useChannelDivinity(t, opts...).Run(context.Background(), cmd)
		if _, ok := res.(character.UsePaladinChannelDivinityApplied); !ok || err != nil {
			t.Fatalf("%s: %#v, %v", feature, res, err)
		}
		checkFeatureCall(t, rec, pldWant(character.SpanUsePaladinChannelDivinity, feature, ""))
		if option.String() != feature || !option.Valid() {
			t.Errorf("option %d = %q", option, option.String())
		}
	}
	rec, opts := fgtTelemetry()
	cmd := sacredCmd(fgtFighter(t, 5))
	if res, _ := useChannelDivinity(t, opts...).Run(context.Background(), cmd); res == nil {
		t.Fatal("no result")
	} else if _, ok := res.(character.UsePaladinChannelDivinityNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUsePaladinChannelDivinity, "sacred_weapon", "not_a_paladin"))
	good := sacredCmd(pldPaladin(t, 5))
	for name, mutate := range map[string]func(*character.UsePaladinChannelDivinityCommand){
		"undefined option":      func(c *character.UsePaladinChannelDivinityCommand) { c.Option = 0 },
		"held without a weapon": func(c *character.UsePaladinChannelDivinityCommand) { c.Weapon = "" },
		"zero character":        func(c *character.UsePaladinChannelDivinityCommand) { c.Character = character.Character{} },
		"foreign state":         func(c *character.UsePaladinChannelDivinityCommand) { c.Character = pldForeign(t, c.Character) },
	} {
		cmd := good
		mutate(&cmd)
		if res, err := useChannelDivinity(t).Run(context.Background(), cmd); res != nil || err == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		} else {
			wantInvalid(t, err, name)
		}
	}
	if character.PaladinChannelDivinityOption(0).Valid() || character.PaladinChannelDivinityOption(0).String() != "unknown" {
		t.Error("the zero option is valid")
	}
	if _, err := (character.UsePaladinChannelDivinity{}).Run(context.Background(), good); err == nil {
		t.Error("the zero use case ran")
	}
	if _, err := character.NewUsePaladinChannelDivinity(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := useChannelDivinity(t).Run(ctx, good); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

// Requirement: doc/specs/class-paladin.md FR-069, FR-089, FR-125 - a progression built without the Oath of Devotion gives its features no level, so they are never available: Sacred Weapon is option_not_available, Holy
// Nimbus is feature_not_gained and Smite of Protection has no effect, even for a Paladin of level 20 who holds the Oath.
// Path: unhappy (missing content: no subclass definition).
func TestOathFeaturesNeedTheOathDefinition(t *testing.T) {
	t.Parallel()
	bare := must(character.NewPaladinProgression(pldDef(t), nil))
	res := must(must(character.NewUsePaladinChannelDivinity(bare)).Run(context.Background(), sacredCmd(pldPaladin(t, 20))))
	if rj, ok := res.(character.UsePaladinChannelDivinityRejected); !ok || !slices.Equal(pldReasons(rj.Reasons()), []string{"option_not_available"}) {
		t.Errorf("Sacred Weapon = %#v", res)
	}
	nimbus := must(must(character.NewUseHolyNimbus(bare)).Run(context.Background(), character.UseHolyNimbusCommand{Character: pldPaladin(t, 20), Subclass: oath, Action: character.HolyNimbusActivate}))
	if rj, ok := nimbus.(character.UseHolyNimbusRejected); !ok || !slices.Equal(pldReasons(rj.Reasons()), []string{"feature_not_gained"}) {
		t.Errorf("Holy Nimbus = %#v", nimbus)
	}
	smite := must(must(character.NewApplySmiteOfProtection(bare)).Run(context.Background(), character.ApplySmiteOfProtectionCommand{Character: pldPaladin(t, 20), Subclass: oath, DivineSmiteCast: true, Charisma: modifierOf(t, 3)}))
	if _, ok := smite.(character.ApplySmiteOfProtectionNoEffect); !ok {
		t.Errorf("Smite of Protection = %#v", smite)
	}
}
