package combat_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// restrictionName names a restriction for comparison in tables: unrestricted,
// never, slot_<n>, or after:<ids>. It switches exhaustively over the closed set.
func restrictionName(t *testing.T, r combat.DispelRestriction) string {
	t.Helper()
	switch v := r.(type) {
	case combat.DispelUnrestricted:
		return "unrestricted"
	case combat.DispelNeverDispellable:
		return "never"
	case combat.DispelRequiresSlotLevel:
		return "slot_" + strconv.Itoa(v.Level().Value())
	case combat.DispelAfterPartsGone:
		ids := make([]string, 0, len(v.IDs()))
		for _, id := range v.IDs() {
			ids = append(ids, id.String())
		}
		return "after:" + strings.Join(ids, ",")
	default:
		t.Fatalf("restriction %T is not a known variant", r)
		return ""
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-017 - DispelRestrictions.ForSpell gives the restriction of table T-R, one row per case (SRD 5.2.1, spell entries pp. 109, 134, 138, 142, 155, 172, 173).
// Path: per case.
func TestDispelRestrictionsForSpellTableT_R(t *testing.T) {
	t.Parallel()
	siblings := []combat.OngoingEffectID{dspEffectID(t, "layer-1")}
	violet := combat.SpellPartPrismaticLayer(combat.PrismaticLayer(7))
	cases := []struct {
		name, requirement, path string
		id                      content.SpellID
		opts                    combat.ForSpellOptions
		want                    string
	}{
		{"forcecage", `SRD 5.2.1, "Forcecage", p. 134`, "happy", "forcecage", combat.ForSpellOptions{}, "never"},
		{"wall of force", `SRD 5.2.1, "Wall of Force", p. 172`, "happy", "wall_of_force", combat.ForSpellOptions{}, "never"},
		{"wall of stone, permanent", `SRD 5.2.1, "Wall of Stone", p. 173`, "happy", "wall_of_stone", combat.ForSpellOptions{Permanent: true}, "never"},
		{"wall of stone, not permanent", `SRD 5.2.1, "Wall of Stone", p. 173`, "unhappy (boundary)", "wall_of_stone", combat.ForSpellOptions{}, "unrestricted"},
		{"antimagic field", `SRD 5.2.1, "Antimagic Field", p. 109`, "happy", "antimagic_field", combat.ForSpellOptions{}, "never"},
		{"guards and wards, the parent", `SRD 5.2.1, "Guards and Wards", p. 138`, "happy", "guards_and_wards", combat.ForSpellOptions{}, "never"},
		{"guards and wards, one effect", `SRD 5.2.1, "Guards and Wards", p. 138`, "unhappy (rule interaction)", "guards_and_wards", combat.ForSpellOptions{Part: combat.SpellPartGuardsAndWardsEffect()}, "unrestricted"},
		{"imprisonment", `SRD 5.2.1, "Imprisonment", p. 142`, "happy", "imprisonment", combat.ForSpellOptions{}, "slot_9"},
		{"prismatic wall, the parent", `SRD 5.2.1, "Prismatic Wall", p. 155`, "unhappy (rule interaction)", "prismatic_wall", combat.ForSpellOptions{}, "never"},
		{"prismatic wall, layer 1 red", `SRD 5.2.1, "Prismatic Wall", p. 155`, "happy", "prismatic_wall", combat.ForSpellOptions{Part: combat.SpellPartPrismaticLayer(combat.PrismaticLayer(1))}, "never"},
		{"prismatic wall, layer 6 indigo", `SRD 5.2.1, "Prismatic Wall", p. 155`, "happy", "prismatic_wall", combat.ForSpellOptions{Part: combat.SpellPartPrismaticLayer(combat.PrismaticLayer(6))}, "never"},
		{"prismatic wall, violet waits for the others", `SRD 5.2.1, "Prismatic Wall", p. 155`, "unhappy (rule interaction)", "prismatic_wall", combat.ForSpellOptions{Part: violet, Siblings: siblings}, "after:layer-1"},
		{"unknown spell is not an error", "n/a (an unknown id is unrestricted, not refused)", "happy", "no_such_spell", combat.ForSpellOptions{}, "unrestricted"},
		{"fireball", "n/a", "happy", "fireball", combat.ForSpellOptions{}, "unrestricted"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.DispelRestrictions{}.ForSpell(c.id, c.opts)
			if err != nil {
				t.Fatalf("ForSpell(%s) error = %v [%s]", c.id, err, c.requirement)
			}
			if name := restrictionName(t, got); name != c.want {
				t.Errorf("ForSpell(%s) = %s, want %s [FR-017 %s]", c.id, name, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-017 - a Prismatic Wall layer outside 1 to 7, and a part the spell does not have, are refused (invalid input).
// Path: per case.
func TestDispelRestrictionsForSpellRefusesBadParts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		id                      content.SpellID
		opts                    combat.ForSpellOptions
	}{
		{"prismatic layer 8", "FR-017", "unhappy (boundary)", "prismatic_wall", combat.ForSpellOptions{Part: combat.SpellPartPrismaticLayer(combat.PrismaticLayer(8))}},
		{"prismatic layer 0", "FR-017", "unhappy (boundary)", "prismatic_wall", combat.ForSpellOptions{Part: combat.SpellPartPrismaticLayer(combat.PrismaticLayer(0))}},
		{"layer on guards and wards", "FR-017", "unhappy (invalid input)", "guards_and_wards", combat.ForSpellOptions{Part: combat.SpellPartPrismaticLayer(combat.PrismaticLayer(2))}},
		{"guards effect on a prismatic wall", "FR-017", "unhappy (invalid input)", "prismatic_wall", combat.ForSpellOptions{Part: combat.SpellPartGuardsAndWardsEffect()}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.DispelRestrictions{}.ForSpell(c.id, c.opts)
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("ForSpell(%s) error = %v, want ErrInvalidInput [%s]", c.id, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-017 - a restriction's id slice is copied both ways (immutability).
// Path: unhappy (immutability).
func TestDispelAfterPartsGoneCopiesItsIDs(t *testing.T) {
	t.Parallel()
	ids := []combat.OngoingEffectID{dspEffectID(t, "layer-2")}
	r := combat.NewDispelAfterPartsGone(ids)
	ids[0] = dspEffectID(t, "changed")
	out := r.IDs()
	out[0] = dspEffectID(t, "also-changed")
	if got := r.IDs()[0]; got != dspEffectID(t, "layer-2") {
		t.Errorf("restriction ids changed through the input or a returned slice: %v [FR-017 immutability]", got)
	}
}
