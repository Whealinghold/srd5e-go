package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A level 1 Fighter with 300 Experience Points gains a level over the bundled Fighter,
// taking the fixed Hit Point value; the Hit Point maximum and the class level follow.
func ExampleLevelUpCharacter_Run() {
	use, _ := character.NewLevelUpCharacter(character.WithSource(data.MustBundledSource()), character.WithRoller(dicetest.NewFixedRoller()))
	pc := pcExample().AwardExperience(character.MustExperienceAmount(300)).Character()
	res, _ := use.Run(context.Background(), character.LevelUpCharacterCommand{
		Character: pc, Class: content.MustClassID("fighter"), HitPointChoice: character.TakeFixedValue{},
	})
	next := res.(character.LevelUpCharacterSucceeded).Character()
	fmt.Println(next.ClassLevels().TotalLevel().Value(), next.HitPointMaximum().Value() > pc.HitPointMaximum().Value())
	// Output: 2 true
}

// A level 3 Wizard casts Magic Missile with a level 2 slot; one level 2 slot is left.
func ExampleCastSpell_Do() {
	ctx := context.Background()
	levels := character.MustClassLevels([]character.ClassLevel{{Class: "wizard", Level: 3}})
	missile := content.MustSpellID("magic_missile")
	res, _ := data.MustBundledSource().Lookup(ctx, content.NewKey(content.KindSpell, content.MustID("magic_missile")))
	spell := res.(content.LookupFound).Entry.(content.Spell)
	created, _ := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{
		Character: character.MustIdentifier("wiz"), Levels: levels, Spells: map[content.SpellID]content.Spell{missile: spell},
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Prepared: []content.SpellID{missile}, Spellbook: character.NewSpellbook([]content.SpellID{missile})}},
	})
	state := created.(character.CreateSpellcastingCreated).State()
	facts, _ := character.NewCastingContext(character.CastingFacts{CanCastSpells: true, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, HasComponentPouch: true, TriggerMet: true})
	level2 := character.MustSlotLevel(2)
	cast, _ := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Class: character.CasterWizard, Spell: spell,
		Mode: character.CastModeNormal, Slot: character.NewSpellSlotChoiceSpellcasting(level2), Context: facts})
	fmt.Println(cast.(character.CastSpellCast).State().Slots(character.PoolSpellcasting).Remaining(level2))
	// Output: 1
}
