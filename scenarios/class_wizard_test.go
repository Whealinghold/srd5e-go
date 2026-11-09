package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A level 4 Wizard, through the public packages only: it builds its spellbook (six spells at level 1, two more
// at each of levels 2 to 4), finishes a Short Rest and uses Arcane Recovery to recover two levels' worth of
// slots, after which the feature is used until a Long Rest (SRD 5.2.1, "Level 1: Spellcasting", Spellbook,
// "Level 1: Arcane Recovery", p. 78). Cites class-wizard.md FR-030, FR-033, FR-045, FR-046, FR-050, FR-051.
func TestWizardBuildsASpellbookAndRecoversSlots(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("wiz-1")
	spell := func(name string, level int) character.WizardSpellFacts {
		return character.WizardSpellFacts{ID: content.SpellID(name), Level: content.SpellLevel(level), Classes: []content.ClassID{"wizard"}}
	}
	levels := func(n int) character.ClassLevels {
		return character.MustClassLevels([]character.ClassLevel{{Class: "wizard", Level: n}})
	}

	start, err := mustOK(character.NewStartSpellbook(src)).Run(ctx, character.StartSpellbookCommand{CharacterID: id, ClassLevels: levels(1),
		Spells: []character.WizardSpellFacts{spell("a", 1), spell("b", 1), spell("c", 1), spell("d", 1), spell("e", 1), spell("f", 1)}})
	book := start.(character.SpellbookChanged).Spellbook()
	if err != nil || len(book.Spells()) != 6 {
		t.Fatalf("StartSpellbook = %#v, %v", start, err)
	}
	for _, step := range []struct {
		level  int
		spells []character.WizardSpellFacts
	}{{2, []character.WizardSpellFacts{spell("g", 1), spell("h", 1)}}, {3, []character.WizardSpellFacts{spell("i", 2), spell("j", 2)}}, {4, []character.WizardSpellFacts{spell("k", 2), spell("l", 1)}}} {
		res, err := mustOK(character.NewAddLevelUpSpells(src)).Run(ctx, character.AddLevelUpSpellsCommand{CharacterID: id, ClassLevels: levels(step.level), Spellbook: book, Spells: step.spells})
		changed, ok := res.(character.SpellbookChanged)
		if err != nil || !ok {
			t.Fatalf("AddLevelUpSpells at level %d = %#v, %v", step.level, res, err)
		}
		book = changed.Spellbook()
	}
	if len(book.Spells()) != 12 {
		t.Fatalf("a level 4 Wizard's book holds %d spells, want 12", len(book.Spells()))
	}

	rest := finishedRest(t, id, character.ShortRest)
	recover := mustOK(character.NewUseArcaneRecovery(src))
	first, err := recover.Run(ctx, character.UseArcaneRecoveryCommand{CharacterID: id, ClassLevels: levels(4), Expended: map[int]int{1: 2}, Chosen: map[int]int{1: 2}, Trigger: rest})
	applied, ok := first.(character.ArcaneRecoveryApplied)
	if err != nil || !ok || applied.Recovered().Levels() != 2 || !applied.Resources().ArcaneRecoveryUsed() {
		t.Fatalf("UseArcaneRecovery = %#v, %v", first, err)
	}
	again, err := recover.Run(ctx, character.UseArcaneRecoveryCommand{CharacterID: id, ClassLevels: levels(4), Resources: applied.Resources(), Expended: map[int]int{1: 2}, Chosen: map[int]int{1: 1}, Trigger: rest})
	if rej, ok := again.(character.UseArcaneRecoveryRejected); err != nil || !ok || rej.Rejection.Kind != character.ArcaneRecoveryAlreadyUsed {
		t.Fatalf("a second use = %#v, %v", again, err)
	}
	if !applied.Resources().AfterRest(character.ShortRest).ArcaneRecoveryUsed() || applied.Resources().AfterRest(character.LongRest).ArcaneRecoveryUsed() {
		t.Error("a Short Rest must not, and a Long Rest must, make Arcane Recovery available again")
	}
}

func mustOK[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func finishedRest(t *testing.T, id character.Identifier, kind character.RestKind) character.RestApplied {
	t.Helper()
	hours := 1
	if kind == character.LongRest {
		hours = 8
	}
	report, err := character.NewRestReport(kind, character.StandardLongRest, character.NewRestRanItsCourse(hours, 0))
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := character.NewTakeRestCommand(id, report, 12, 0, nil, character.SpentHitDice{})
	if err != nil {
		t.Fatal(err)
	}
	return character.NewRestTaker().TakeRest(context.Background(), cmd).(character.TakeRestApplied).Applied()
}

// A level 14 Evoker, through the public packages only: the Evoker subclass comes from the bundled Wizard, the first
// Overchannel use rolls nothing, the second takes 2d12 Necrotic damage for each level of a level 3 slot (6d12), a Short
// Rest leaves the count and a Long Rest resets it (SRD 5.2.1, "Level 14: Overchannel", p. 82). Cites
// class-wizard-evoker.md FR-011 to FR-014, FR-017.
func TestEvokerUsesOverchannelTwice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	found, err := src.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("wizard")))
	if err != nil {
		t.Fatal(err)
	}
	sub := found.(content.LookupFound).Entry.(content.ClassDefinition).Subclass("evoker").(content.SubclassFound).Subclass()
	id := character.MustIdentifier("evoker-1")
	use := mustOK(character.NewUseOverchannel(dicetest.NewMaximumRoller()))
	cast := character.OverchannelRequest{Spell: character.WizardSpellFacts{ID: "fireball", Level: 3, School: content.SchoolEvocation, Classes: []content.ClassID{"wizard"}, DealsDamage: true},
		Source: character.CastSourceSlot, SlotLevel: 3}
	cmd := character.UseOverchannelCommand{CharacterID: id, ClassLevels: character.MustClassLevels([]character.ClassLevel{{Class: "wizard", Level: 14}}), Subclass: sub, Request: cast}

	first, err := use.Run(ctx, cmd)
	one, ok := first.(character.OverchannelApplied)
	if _, rolled := one.Necrotic(); err != nil || !ok || one.Use() != 1 || rolled {
		t.Fatalf("first use = %#v, %v; want use 1 and no Necrotic damage", first, err)
	}
	cmd.Resources = one.Resources()
	second, err := use.Run(ctx, cmd)
	two, ok := second.(character.OverchannelApplied)
	necrotic, rolled := two.Necrotic()
	if err != nil || !ok || two.Use() != 2 || !rolled || necrotic.Total() != 6*12 || necrotic.Expression().Count() != 6 {
		t.Fatalf("second use = %#v, %v; want 6d12 Necrotic damage", second, err)
	}
	if two.Resources().AfterRest(character.ShortRest).OverchannelUses() != 2 || two.Resources().AfterRest(character.LongRest).OverchannelUses() != 0 {
		t.Error("a Short Rest must leave, and a Long Rest must reset, the Overchannel count")
	}
}
