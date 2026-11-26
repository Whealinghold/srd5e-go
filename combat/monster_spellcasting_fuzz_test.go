package combat_test

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-spellcasting.md FR-017 - the statement parser never panics, reads a statement as a unique set of kinds or as unrecognised text it keeps verbatim, and reads a statement the same whatever its case and surrounding white space.
// Path: unhappy (invalid input).
func FuzzParseComponentWaiver(f *testing.F) {
	for _, s := range []string{"", "requiring no Material components.", "requiring no spell components", "requiring no Verbal, Somatic, or Material components",
		"requiring no Material or Material components", "REQUIRING   NO\tSOMATIC components", "requiring no ", "\x00", "requiring no é components"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		switch got := combat.ParseComponentWaiver(s).(type) {
		case combat.ComponentsWaived:
			kinds := got.Kinds()
			if !slices.IsSorted(kinds) || len(slices.Compact(slices.Clone(kinds))) != len(kinds) {
				t.Fatalf("kinds of %q = %v, want a unique ordered set", s, kinds)
			}
		case combat.ComponentStatementUnrecognized:
			if got.Statement() != s {
				t.Fatalf("unrecognised text %q, want %q verbatim", got.Statement(), s)
			}
		default:
			t.Fatalf("ParseComponentWaiver(%q) = %T, want a variant", s, got)
		}
		padded := " \t" + strings.ToUpper(s) + " "
		if strings.EqualFold(strings.ToUpper(s), s) {
			a, aOK := combat.ParseComponentWaiver(s).(combat.ComponentsWaived)
			b, bOK := combat.ParseComponentWaiver(padded).(combat.ComponentsWaived)
			if aOK != bOK || (aOK && !slices.Equal(a.Kinds(), b.Kinds())) {
				t.Fatalf("%q and %q read differently", s, padded)
			}
		}
	})
}

// Requirement: doc/specs/monster-spellcasting.md FR-013 - reading a note never panics and is deterministic.
// Path: unhappy (invalid input).
func FuzzRestrictionsOf(f *testing.F) {
	for _, s := range []string{"", "self only", "Self  Only,", "selfonly", "no Concentration", "\xff\xfeself only", "self only self only"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		first, second := (combat.MonsterSpellRestrictions{}).Of(s), (combat.MonsterSpellRestrictions{}).Of(s)
		if first != second {
			t.Fatalf("Of(%q) is not deterministic", s)
		}
	})
}

// Requirement: doc/specs/monster-spellcasting.md FR-039, FR-004 - the constructors of part values and part casts never panic; a value they accept reads back as given and one they refuse matches srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func FuzzPartSpellcasting(f *testing.F) {
	f.Add(int(ability.Wisdom), 21, true, -3, true, 0, "requiring no spell components")
	f.Add(0, 0, false, 0, false, 10, "")
	f.Add(7, -1, true, math.MaxInt, true, -1, "x")
	f.Fuzz(func(t *testing.T, a, dc int, hasDC bool, bonus int, hasBonus bool, level int, statement string) {
		p, err := combat.NewPartSpellcasting(combat.PartSpellcastingFields{Ability: ability.Ability(a), SaveDC: dc, HasSaveDC: hasDC, AttackBonus: bonus, HasAttackBonus: hasBonus, Components: statement})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !p.IsZero() {
				t.Fatalf("NewPartSpellcasting() = %v, %v, want the zero value and ErrInvalidInput", p, err)
			}
		} else if got, ok := p.SaveDC(); ok != hasDC || (ok && got != dc) || !p.Ability().Valid() {
			t.Fatalf("SaveDC() = %d, %v, want %d, %v", got, ok, dc, hasDC)
		}
		c, err := combat.NewMonsterPartCast(mscPart(t, mscMistyStep), level, p)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewMonsterPartCast() error = %v, want ErrInvalidInput", err)
			}
		} else if got, ok := c.CastAtLevel(); ok != (level != 0) || got != level {
			t.Fatalf("CastAtLevel() = %d, %v, want %d", got, ok, level)
		}
	})
}

// Requirement: doc/specs/monster-spellcasting.md FR-002, FR-004, FR-013, FR-017, FR-035 - a cast built from an untrusted note, components statement and stated level never panics and ends as a performed cast, a started casting, a refusal or an error matching srderr.ErrInvalidInput, never anything else.
// Path: unhappy (invalid input).
func FuzzCast(f *testing.F) {
	f.Add("self only", "requiring no spell components", 3, true, true)
	f.Add("", "", 0, false, false)
	f.Add("no Concentration", "requiring no Arcane components", 9, true, false)
	f.Fuzz(func(t *testing.T, note, statement string, level int, speak, gesture bool) {
		entry, err := content.NewMonsterSpellEntry("fireball", note, level)
		if err != nil {
			return
		}
		own, err := combat.NewPartSpellcasting(combat.PartSpellcastingFields{Ability: ability.Wisdom, SaveDC: 15, HasSaveDC: true, Components: statement})
		if err != nil {
			return
		}
		rig := mscSingle(t, luPerDayEach(1), entry, nil)
		rig.mc = mscContext(speak, gesture, combat.ConditionActionAvailabilityAvailable)
		for _, req := range []combat.MonsterCastRequest{mscList(t, mscFirst), mscPartCast(t, mscMistyStep, 0, own)} {
			res := rig.cast(req, seFireball(), combat.MustCombatantID("ally"))
			switch res.Out.(type) {
			case combat.MonsterCastPerformed, combat.MonsterCastStarted, combat.MonsterCastRefused:
				if res.Err != nil {
					t.Fatalf("an outcome and an error: %v", res.Err)
				}
			case nil:
				if !errors.Is(res.Err, srderr.ErrInvalidInput) {
					t.Fatalf("no outcome and error %v, want ErrInvalidInput", res.Err)
				}
			default:
				t.Fatalf("outcome %T is not a variant", res.Out)
			}
		}
	})
}
