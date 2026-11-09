package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-paladin.md FR-044, FR-045, FR-111 - InitialPaladinState starts every pool full and every use available, and RestorePaladinState keeps the invariants: Lay On Hands
// within 0 to five times the level, Channel Divinity within 0 to the maximum, a use that is available or spent, a Fighting Style of the closed set and distinct valid Blessed Warrior cantrips.
// Path: per case (unhappy: boundary, invalid input).
func TestRestorePaladinState(t *testing.T) {
	t.Parallel()
	l5, l1 := character.MustPaladinLevel(5), character.MustPaladinLevel(1)
	init := character.InitialPaladinState(l5, 2)
	if init.LayOnHandsRemaining() != 25 || init.ChannelDivinityRemaining() != 2 || init.FreeCast(character.FreeCastPaladinsSmite) != character.LongRestUseAvailable ||
		init.FreeCast(character.FreeCastFaithfulSteed) != character.LongRestUseAvailable || init.HolyNimbusUse() != character.LongRestUseAvailable || init.HolyNimbusActive() || init.DivineSenseActive() {
		t.Errorf("InitialPaladinState = %+v", init)
	}
	if _, ok := init.FightingStyle(); ok {
		t.Error("an initial state has a Fighting Style")
	}
	if _, ok := init.SacredWeaponActive(); ok {
		t.Error("an initial state has a Sacred Weapon")
	}
	if got := character.InitialPaladinState(l1, -3); got.ChannelDivinityRemaining() != 0 || got.LayOnHandsRemaining() != 5 {
		t.Errorf("negative maximum = %+v", got)
	}
	ok := character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2}
	for _, c := range []struct {
		name  string
		level character.PaladinLevel
		max   int
		parts func(character.PaladinStateParts) character.PaladinStateParts
		want  bool
	}{
		{"valid", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { return p }, true},
		{"lay on hands at 0", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.LayOnHandsRemaining = 0; return p }, true},
		{"lay on hands above the maximum", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.LayOnHandsRemaining = 26; return p }, false},
		{"lay on hands negative", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.LayOnHandsRemaining = -1; return p }, false},
		{"channel above the maximum", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.ChannelDivinityRemaining = 3
			return p
		}, false},
		{"channel negative", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.ChannelDivinityRemaining = -1
			return p
		}, false},
		{"zero level", character.PaladinLevel{}, 2, func(p character.PaladinStateParts) character.PaladinStateParts { return p }, false},
		{"negative smite use", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.PaladinsSmite = -1; return p }, false},
		{"undefined smite use", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.PaladinsSmite = 7; return p }, false},
		{"undefined steed use", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.FaithfulSteed = 7; return p }, false},
		{"undefined nimbus use", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts { p.HolyNimbus = 7; return p }, false},
		{"feat style", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.FightingStyle = character.PaladinFightingStyleFeat{Feat: "dueling"}
			return p
		}, true},
		{"feat style without a feat", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.FightingStyle = character.PaladinFightingStyleFeat{}
			return p
		}, false},
		{"Blessed Warrior", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.FightingStyle = character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}
			return p
		}, true},
		{"Blessed Warrior in reverse order", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.FightingStyle = character.BlessedWarrior{Cantrips: [2]content.SpellID{"sacred_flame", "guidance"}}
			return p
		}, true},
		{"Blessed Warrior twice the same", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.FightingStyle = character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "guidance"}}
			return p
		}, false},
		{"Blessed Warrior with an empty id", l5, 2, func(p character.PaladinStateParts) character.PaladinStateParts {
			p.FightingStyle = character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", ""}}
			return p
		}, false},
	} {
		s, err := character.RestorePaladinState(c.level, c.max, c.parts(ok))
		if (err == nil) != c.want {
			t.Errorf("%s: err = %v, want ok %v", c.name, err, c.want)
		}
		if err != nil {
			wantInvalid(t, err, c.name)
		} else if s.LayOnHandsRemaining() != c.parts(ok).LayOnHandsRemaining {
			t.Errorf("%s: state = %+v", c.name, s)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-040, FR-121 - PaladinState implements FeatureState (Equal by value, false for another type), round-trips through FeatureStates under
// PaladinStateKey, and hands its Blessed Warrior cantrips to spellcasting as GrantedCantrips of the Paladin class with the Cleric list, none without Blessed Warrior.
// Path: happy / unhappy (invalid input).
func TestPaladinStateIsAFeatureState(t *testing.T) {
	t.Parallel()
	a := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 1, PaladinsSmite: character.LongRestUseSpent,
		FightingStyle: character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}, DivineSenseActive: true, SacredWeaponActive: "sword", HolyNimbusActive: true})
	if !a.Equal(a) || a.Equal(pldFull(t, 5)) || a.Equal(character.ClericResources{}) {
		t.Error("Equal wrong")
	}
	if a.FreeCast(character.FreeCastPaladinsSmite) != character.LongRestUseSpent || a.FreeCast(character.FreeCastFaithfulSteed) != character.LongRestUseAvailable || !a.DivineSenseActive() || !a.HolyNimbusActive() {
		t.Errorf("accessors wrong: %+v", a)
	}
	if w, ok := a.SacredWeaponActive(); !ok || w != "sword" {
		t.Errorf("SacredWeaponActive = %q, %v", w, ok)
	}
	if style, ok := a.FightingStyle(); !ok || style != (character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}) {
		t.Errorf("FightingStyle = %v, %v", style, ok)
	}
	cantrips := a.BlessedWarriorCantrips()
	if len(cantrips) != 2 || cantrips[0].Spell() != "guidance" || cantrips[1].Spell() != "sacred_flame" || cantrips[0].Class() != character.CasterPaladin || cantrips[0].ListClass() != "cleric" {
		t.Errorf("BlessedWarriorCantrips = %v", cantrips)
	}
	if got := pldFull(t, 5).BlessedWarriorCantrips(); got == nil || len(got) != 0 {
		t.Errorf("no Blessed Warrior = %v, want an empty slice", got)
	}
	c := pldWith(t, pldPaladin(t, 5), a)
	if !pldStateOf(t, c).Equal(a) {
		t.Error("the state did not round-trip")
	}
	for k, want := range map[character.LongRestUse]string{character.LongRestUseAvailable: "available", character.LongRestUseSpent: "spent", 9: "unknown"} {
		if k.String() != want {
			t.Errorf("LongRestUse %d = %q, want %q", k, k.String(), want)
		}
	}
	spent := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, PaladinsSmite: character.LongRestUseSpent})
	if spent.FreeCast(3) != character.LongRestUseAvailable || spent.FreeCast(0) != character.LongRestUseAvailable || spent.FreeCast(character.FreeCastFaithfulSteed) != character.LongRestUseAvailable {
		t.Error("an invalid free cast feature must read as available, and Steed must not read Smite")
	}
	for f, want := range map[character.FreeCastFeature]string{character.FreeCastPaladinsSmite: "paladins_smite", character.FreeCastFaithfulSteed: "faithful_steed", 0: "unknown"} {
		if f.String() != want || (want != "unknown") != f.Valid() {
			t.Errorf("FreeCastFeature %d = %q valid %v", f, f.String(), f.Valid())
		}
	}
	if character.FreeCastFeature(3).Valid() || character.ClassEntry(3).Valid() || character.ClassEntry(-1).Valid() || character.FreeCastFeature(-1).Valid() {
		t.Error("an undefined value is valid")
	}
	for e, want := range map[character.ClassEntry]string{character.ClassEntryFirstLevel: "first_level", character.ClassEntryMulticlass: "multiclass", 0: "unknown"} {
		if e.String() != want || (want != "unknown") != e.Valid() {
			t.Errorf("ClassEntry %d = %q valid %v", e, e.String(), e.Valid())
		}
	}
}
