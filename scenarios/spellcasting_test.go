package scenarios_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

func bundledSpells(t *testing.T) map[content.SpellID]content.Spell {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "spells.json")
	if err != nil {
		t.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[content.SpellID]content.Spell{}
	for _, e := range set.Entries() {
		sp := e.(content.Spell)
		out[sp.ID()] = sp
	}
	return out
}

func spellLevels(t *testing.T, pairs ...any) character.ClassLevels {
	t.Helper()
	var es []character.ClassLevel
	for i := 0; i < len(pairs); i += 2 {
		es = append(es, character.ClassLevel{Class: content.MustClassID(pairs[i].(string)), Level: pairs[i+1].(int)})
	}
	l, err := character.NewClassLevels(es)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func ids(names ...string) []content.SpellID {
	out := make([]content.SpellID, len(names))
	for i, n := range names {
		out[i] = content.MustSpellID(n)
	}
	return out
}

// A Wizard 5 creates a spellcasting state with bundled spells, adds a level 3 spell,
// expends the slot and finishes a Long Rest; a Ranger 4 / Sorcerer 3 shares one pool
// (the SRD example); a Warlock regains Pact Magic slots on a Short Rest only; a
// level-up raises the prepared limit (SRD 5.2.1, "Spell Slots", p. 104;
// "Multiclassing", pp. 25-26; "Warlock", Pact Magic, p. 71). Cites spellcasting.md
// FR-021, FR-025, FR-026, FR-031, FR-089, FR-092, FR-047.
func TestSpellcastingThroughThePublicAPI(t *testing.T) {
	t.Parallel()
	ctx, id, spells := context.Background(), character.MustIdentifier("wizard"), bundledSpells(t)

	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: id, Levels: spellLevels(t, "wizard", 5), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("magic_missile", "detect_magic"), Spellbook: character.NewSpellbook(ids("magic_missile", "detect_magic", "fireball", "misty_step"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state := created.(character.CreateSpellcastingCreated).State()

	chosen, err := character.NewChooseAdditionalSpells().Do(ctx, character.ChooseAdditionalSpellsCommand{State: state, Class: character.CasterWizard, Prepared: ids("fireball"), Spells: spells})
	if err != nil {
		t.Fatal(err)
	}
	state = chosen.(character.ChooseAdditionalSpellsChosen).State()

	level3 := character.MustSlotLevel(3)
	for range 2 {
		res, err := character.NewExpendSpellSlot().Do(ctx, character.ExpendSpellSlotCommand{State: state, Pool: character.PoolSpellcasting, Level: level3})
		if err != nil {
			t.Fatal(err)
		}
		state = res.(character.ExpendSpellSlotExpended).State()
	}
	again, err := character.NewExpendSpellSlot().Do(ctx, character.ExpendSpellSlotCommand{State: state, Pool: character.PoolSpellcasting, Level: level3})
	if _, ok := again.(character.ExpendSpellSlotRejected); err != nil || !ok {
		t.Fatalf("a third level 3 slot = %#v, %v, want rejected", again, err)
	}
	rested, err := character.NewRestoreSpellSlots().Do(ctx, character.RestoreSpellSlotsCommand{State: state, Rest: character.LongRest})
	if err != nil {
		t.Fatal(err)
	}
	if got := rested.(character.RestoreSpellSlotsRestored).State().Slots(character.PoolSpellcasting).Remaining(level3); got != 2 {
		t.Errorf("level 3 slots after a Long Rest = %d, want 2", got)
	}

	// H1: Ranger 4 / Sorcerer 3 share the pool of a level 5 caster.
	summary, err := character.NewCalculateSpellcasting().Do(ctx, character.CalculateSpellcastingCommand{Levels: spellLevels(t, "ranger", 4, "sorcerer", 3), Scores: scoresFor(t, 10, 10, 10, 10, 10, 10), Proficiency: character.ProficiencyBonusFor(character.MustLevel(7))})
	if err != nil {
		t.Fatal(err)
	}
	pool := summary.(character.CalculateSpellcastingCalculated).Summary().Pools().Spellcasting()
	if pool.Maximum(character.MustSlotLevel(1)) != 4 || pool.Maximum(character.MustSlotLevel(2)) != 3 || pool.Maximum(level3) != 2 || pool.Maximum(character.MustSlotLevel(4)) != 0 {
		t.Errorf("Ranger 4 / Sorcerer 3 pool = %+v", pool)
	}

	// A Warlock's Short Rest restores Pact Magic only.
	lock, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: id, Levels: spellLevels(t, "warlock", 5, "wizard", 1), Spells: spells})
	if err != nil {
		t.Fatal(err)
	}
	ls := lock.(character.CreateSpellcastingCreated).State()
	ls = ls.ExpendSlot(character.PoolPactMagic, level3).(character.SlotExpenditureExpended).State()
	ls = ls.ExpendSlot(character.PoolSpellcasting, character.MustSlotLevel(1)).(character.SlotExpenditureExpended).State()
	ls, _ = ls.RestoreSlots(character.ShortRest)
	if ls.Slots(character.PoolPactMagic).Remaining(level3) != 2 || ls.Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(1)) != 1 {
		t.Error("a Short Rest must restore Pact Magic and leave the Spellcasting pool alone")
	}

	// A level-up raises the prepared limit by the Table G difference.
	adv, err := character.NewAdvanceSpellcasting().Do(ctx, character.AdvanceSpellcastingCommand{State: state, Levels: spellLevels(t, "wizard", 6)})
	if err != nil {
		t.Fatal(err)
	}
	a := adv.(character.AdvanceSpellcastingAdvanced)
	if len(a.SpellsToChoose()) != 1 || a.SpellsToChoose()[0].Count() != 10-3 {
		t.Errorf("SpellsToChoose = %+v, want 7 (limit 10 minus 3 prepared)", a.SpellsToChoose())
	}
}
