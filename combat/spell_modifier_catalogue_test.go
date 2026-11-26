package combat_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/spell-granted-modifiers.md FR-002, FR-070, FR-081, table C - each of the 14 entries is pinned to its SRD page, its subject (the caster only for Hunter's Mark), its facts and the coverage the facts imply (partial with any fact, else complete) (SRD 5.2.1, spell entries pp. 112-163).
// Path: per case.
func TestCatalogueEntriesArePinnedToTableC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		row    string
		spell  content.SpellID
		page   int
		caster bool
		facts  []combat.SpellEffectFact
	}{
		{"C1", "barkskin", 112, false, nil},
		{"C2", "bestow_curse", 112, false, []combat.SpellEffectFact{combat.ForcedDodgeUnlessWisdomSave{}}},
		{"C3", "guidance", 138, false, nil},
		{"C4", "haste", 139, false, []combat.SpellEffectFact{combat.ExtraActionGranted{}, combat.LethargyOnEnd{}}},
		{"C5", "hex", 140, false, []combat.SpellEffectFact{combat.MarkMovesOnZeroHitPoints{}}},
		{"C6", "hunters_mark", 141, true, []combat.SpellEffectFact{combat.MarkMovesOnZeroHitPoints{}}},
		{"C7", "longstrider", 145, false, nil},
		{"C8", "mage_armor", 145, false, []combat.SpellEffectFact{combat.EndsIfArmorDonned{}}},
		{"C9", "resistance", 158, false, nil},
		{"C10", "shield", 161, false, []combat.SpellEffectFact{combat.NoDamageFromMagicMissile{}}},
		{"C11", "shield_of_faith", 162, false, nil},
		{"C12", "slow", 163, false, []combat.SpellEffectFact{combat.ReactionsForbidden{}, combat.ActionOrBonusActionNotBoth{}, combat.OneAttackOnly{}, combat.SpellFailureChancePercent{}}},
	}
	for _, c := range cases {
		t.Run(c.row+" "+string(c.spell), func(t *testing.T) {
			t.Parallel()
			k, ok := combat.LookupSpellModifiers(c.spell).(combat.SpellModifiersKnown)
			if !ok {
				t.Fatalf("LookupSpellModifiers(%s) is not known", c.spell)
			}
			e := k.Entry()
			wantCoverage := content.CoverageComplete
			if len(c.facts) > 0 {
				wantCoverage = content.CoveragePartial
			}
			if e.SpellID() != c.spell || e.Page() != c.page || e.SubjectIsCaster() != c.caster || !reflect.DeepEqual(e.Facts(), c.facts) || e.Coverage() != wantCoverage {
				t.Errorf("entry = %v %d %v %v %v, want %v %d %v %v %v [%sFR-081 %s]", e.SpellID(), e.Page(), e.SubjectIsCaster(), e.Facts(), e.Coverage(), c.spell, c.page, c.caster, c.facts, wantCoverage, spellModifiersSpec, c.row)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-002, R21, Q7 - a spell id outside table C is SpellModifiersNotModelled, never an error, whatever its form (Fireball, Sleep, an invalid id, the empty id); a near miss in case is not modelled either.
// Path: unhappy (missing content).
func TestUnknownSpellIsNotModelled(t *testing.T) {
	t.Parallel()
	for _, id := range []content.SpellID{"fireball", "sleep", "Haste", "hold_person", "", "not a spell"} {
		n, ok := combat.LookupSpellModifiers(id).(combat.SpellModifiersNotModelled)
		if !ok || n.SpellID() != id {
			t.Errorf("LookupSpellModifiers(%q) = %#v, want SpellModifiersNotModelled", id, combat.LookupSpellModifiers(id))
		}
	}
}
