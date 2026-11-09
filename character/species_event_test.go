package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-055 - the species trait events are a closed family of Event values: a type switch over SpeciesTraitEvent lists every variant, each carries the character id, and a slice-holding event compares with Equal and returns copies.
// Path: happy.
func TestSpeciesTraitEventFamily(t *testing.T) {
	t.Parallel()
	pc := spChar(t, 5, 14)
	expend := must(character.SpeciesUses{}.Expend(pc, spDwarf(t), character.StonecunningRequest{})).(character.SpeciesTraitsChanged).Events()[0]
	renew := must(character.SpeciesUses{}.Renew(pc, spExpended(t, spOrc(t), character.AdrenalineRushUse, 1), spRest(t, character.ShortRest, false))).(character.SpeciesTraitsChanged).Events()[0]
	swapped, tr := replaceTo(t, "light")
	_ = swapped
	swap := tr.(character.SpeciesTraitsChanged).Events()[0]
	events := []character.SpeciesTraitEvent{expend, renew, swap}
	seen := map[string]bool{}
	for _, e := range events {
		var kind string
		switch e.(type) {
		case character.SpeciesUseExpended:
			kind = "expended"
		case character.SpeciesUsesRenewed:
			kind = "renewed"
		case character.HighElfCantripReplaced:
			kind = "replaced"
		case character.HalflingLuckUsed:
			kind = "luck"
		case character.RelentlessEnduranceUsed:
			kind = "relentless"
		case character.StonesEnduranceUsed:
			kind = "stones"
		}
		seen[kind] = true
		var ev character.Event = e
		if ev.CharacterID() != pc.ID() {
			t.Errorf("%T carries id %v", e, ev.CharacterID())
		}
	}
	if len(seen) != 3 || seen[""] {
		t.Errorf("kinds seen = %v", seen)
	}
	r := renew.(character.SpeciesUsesRenewed)
	uses := r.Uses()
	uses[0] = 0
	if r.Uses()[0] == 0 || !r.Equal(r) || r.Equal(character.SpeciesUsesRenewed{}) {
		t.Error("SpeciesUsesRenewed is not a value with copies")
	}
	if len(r.Windows()) != 0 {
		t.Errorf("windows = %v", r.Windows())
	}
}

// renewedEvent renews the traits of c at a rest and returns the event.
func renewedEvent(t *testing.T, c character.Character, tr character.SpeciesTraits, rest character.RestApplied) character.SpeciesUsesRenewed {
	t.Helper()
	return must(character.SpeciesUses{}.Renew(c, tr, rest)).(character.SpeciesTraitsChanged).Events()[0].(character.SpeciesUsesRenewed)
}

// Requirement: doc/specs/species-trait-behaviour.md FR-050, FR-055 - SpeciesUsesRenewed compares by value: two events are equal only when the character, the uses cleared and the windows opened are all equal, and a difference in any one is not equal.
// Path: unhappy (immutability).
func TestSpeciesUsesRenewedEqual(t *testing.T) {
	t.Parallel()
	pc, other := spChar(t, 5, 10), spCharNamed(t, "pc-2")
	orc := spExpended(t, spOrc(t), character.AdrenalineRushUse, 1)
	base := renewedEvent(t, pc, orc, spRest(t, character.ShortRest, false))
	if !base.Equal(renewedEvent(t, pc, orc, spRest(t, character.ShortRest, false))) {
		t.Error("equal renewals are not Equal")
	}
	elf := must(spElf(t, character.ElvenHighElf).WithExpended(character.LineageSpellThreeUse, 1))
	for name, differing := range map[string]character.SpeciesUsesRenewed{
		"another character": renewedEvent(t, other, orc, fgtRest(t, other.ID(), character.ShortRest)),
		"other uses":        renewedEvent(t, pc, spExpended(t, spExpended(t, spOrc(t), character.AdrenalineRushUse, 1), character.RelentlessEnduranceUse, 1), spRest(t, character.LongRest, false)),
		"other windows":     renewedEvent(t, pc, spHuman(t), spRest(t, character.LongRest, false)),
		"uses and windows":  renewedEvent(t, pc, elf, spRest(t, character.LongRest, false)),
	} {
		if base.Equal(differing) || differing.Equal(base) {
			t.Errorf("%s: events are Equal", name)
		}
	}
	sameUsesOtherWindows := renewedEvent(t, pc, spExpended(t, spElf(t, character.ElvenHighElf), character.LineageSpellThreeUse, 1), spRest(t, character.LongRest, false))
	sameUsesNoWindows := renewedEvent(t, pc, spExpended(t, spElf(t, character.ElvenDrow), character.LineageSpellThreeUse, 1), spRest(t, character.LongRest, false))
	if sameUsesOtherWindows.Equal(sameUsesNoWindows) {
		t.Error("events with the same uses and other windows are Equal")
	}
}
