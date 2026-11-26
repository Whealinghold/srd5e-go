package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func grpD2PaladinProgression(t *testing.T) character.PaladinProgression {
	t.Helper()
	def := grpD2Class(t, "paladin")
	oath := def.Subclasses()[0]
	return must2(character.NewPaladinProgression(def, &oath))
}

func grpD2PaladinReasons(rs []character.PaladinReason) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}

func grpD2PaladinState(t *testing.T, c character.Character) character.PaladinState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.PaladinStateKey())
	if err != nil || !ok {
		t.Fatalf("PaladinState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// grpD2Paladin is a Paladin of level n holding the given state parts (the full state of the level when parts is nil).
func grpD2Paladin(t *testing.T, prog character.PaladinProgression, n int, parts *character.PaladinStateParts) character.Character {
	t.Helper()
	hero := grpD2Flat(t, 10, grpD2CL("paladin", n))
	level := character.MustPaladinLevel(n)
	state := character.InitialPaladinState(level, prog.ChannelDivinityMaximum(level))
	if parts != nil {
		state = must2(character.RestorePaladinState(level, prog.ChannelDivinityMaximum(level), *parts))
	}
	return must2(character.WithFeatureState(hero, character.PaladinStateKey(), state))
}

// A level 5 Paladin over the bundled Paladin takes a Fighting Style feat, then (as another character) the Blessed Warrior style with two Cleric cantrips and replaces one of them,
// and spends the Paladin's Smite and Faithful Steed free casts (SRD 5.2.1, "Level 2: Fighting Style", "Level 2: Paladin's Smite", "Level 5: Faithful Steed", pp. 54-55). Each choice stores
// itself in the Paladin state, leaves the input untouched and ends with a PaladinChoiceMade event.
// Requirement: doc/specs/class-paladin.md FR-038 to FR-041, FR-060 to FR-064, FR-105, FR-124 - ChoosePaladinOption and UseFreeCast through the public API.
// Path: happy.
func TestPaladinOptionAndFreeCastFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := grpD2PaladinProgression(t)
	choose := must2(character.NewChoosePaladinOption(prog))
	guidance, flame, mending := grpD2Spell(t, "guidance", 0, "cleric", "druid"), grpD2Spell(t, "sacred_flame", 0, "cleric"), grpD2Spell(t, "mending", 0, "cleric", "wizard")

	hero := grpD2Paladin(t, prog, 5, nil)
	style := must2(choose.Run(ctx, character.ChoosePaladinOptionCommand{Character: hero, Choice: character.ChoosePaladinFightingStyle{Feat: grpD2StyleFeat("archery")}})).(character.ChoosePaladinOptionAccepted)
	chosen, _ := grpD2PaladinState(t, style.Character()).FightingStyle()
	last, isChoice := style.Events()[len(style.Events())-1].(character.PaladinChoiceMade)
	if chosen != (character.PaladinFightingStyleFeat{Feat: "archery"}) || style.Feats().Len() != 1 || len(style.Events()) != 2 || !isChoice || last.Choice() != "fighting_style_feat" {
		t.Errorf("Fighting Style feat: style %v, feats %d, events %#v", chosen, style.Feats().Len(), style.Events())
	}
	if !hero.Equal(grpD2Paladin(t, prog, 5, nil)) {
		t.Error("the input character changed")
	}

	warrior := must2(choose.Run(ctx, character.ChoosePaladinOptionCommand{Character: hero, Choice: character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{guidance, flame}}})).(character.ChoosePaladinOptionAccepted)
	if got, _ := grpD2PaladinState(t, warrior.Character()).FightingStyle(); got != (character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}) || warrior.Feats().Len() != 0 {
		t.Errorf("Blessed Warrior style = %v, feats %d", got, warrior.Feats().Len())
	}
	swapped := must2(choose.Run(ctx, character.ChoosePaladinOptionCommand{Character: warrior.Character(), Choice: character.ReplaceBlessedWarriorCantrip{Old: "guidance", New: mending}})).(character.ChoosePaladinOptionAccepted)
	if got, _ := grpD2PaladinState(t, swapped.Character()).FightingStyle(); got != (character.BlessedWarrior{Cantrips: [2]content.SpellID{"mending", "sacred_flame"}}) {
		t.Errorf("after the replacement = %v", got)
	}

	free := must2(character.NewUseFreeCast(prog))
	smite := must2(free.Run(ctx, character.UseFreeCastCommand{Character: hero, Feature: character.FreeCastPaladinsSmite})).(character.UseFreeCastApplied)
	if smite.Spell() != "divine_smite" || grpD2PaladinState(t, smite.Character()).FreeCast(character.FreeCastPaladinsSmite) != character.LongRestUseSpent ||
		grpD2PaladinState(t, smite.Character()).FreeCast(character.FreeCastFaithfulSteed) != character.LongRestUseAvailable {
		t.Errorf("Paladin's Smite free cast = %v, state %+v", smite.Spell(), grpD2PaladinState(t, smite.Character()))
	}
	steed := must2(free.Run(ctx, character.UseFreeCastCommand{Character: smite.Character(), Feature: character.FreeCastFaithfulSteed})).(character.UseFreeCastApplied)
	if steed.Spell() != "find_steed" || len(steed.Events()) != 1 {
		t.Errorf("Faithful Steed free cast = %v, events %v", steed.Spell(), steed.Events())
	}
	again := must2(free.Run(ctx, character.UseFreeCastCommand{Character: steed.Character(), Feature: character.FreeCastPaladinsSmite}))
	if r, ok := again.(character.UseFreeCastRejected); !ok || !slices.Equal(grpD2PaladinReasons(r.Reasons()), []string{"already_used_since_long_rest"}) {
		t.Errorf("a second Paladin's Smite = %#v", again)
	}
}

// The unhappy paths of the Paladin option and free cast use cases: every reason is reported in declaration order, a Fighter is not a Paladin, and nothing changes.
// Requirement: doc/specs/class-paladin.md FR-038 to FR-041, FR-060 to FR-064, FR-105, FR-106 - rejections, invalid input and cancellation.
// Path: unhappy (boundary, state error, invalid input, cancellation).
func TestPaladinOptionAndFreeCastRefusals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := grpD2PaladinProgression(t)
	choose := must2(character.NewChoosePaladinOption(prog))
	guidance, flame := grpD2Spell(t, "guidance", 0, "cleric", "druid"), grpD2Spell(t, "sacred_flame", 0, "cleric")
	for _, c := range []struct {
		name   string
		hero   character.Character
		choice character.PaladinChoice
		want   []string
	}{
		{"a style at level 1", grpD2Paladin(t, prog, 1, nil), character.ChoosePaladinFightingStyle{Feat: grpD2StyleFeat("archery")}, []string{"level_too_low"}},
		{"a feat of another category", grpD2Paladin(t, prog, 2, nil), character.ChoosePaladinFightingStyle{Feat: grpD2Feat("alert", content.FeatOrigin, false)}, []string{"not_a_fighting_style_feat"}},
		{"one cantrip", grpD2Paladin(t, prog, 2, nil), character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{guidance}}, []string{"wrong_count"}},
		{"a replacement without Blessed Warrior", grpD2Paladin(t, prog, 5, nil), character.ReplaceBlessedWarriorCantrip{Old: "guidance", New: flame}, []string{"not_blessed_warrior"}},
	} {
		before := c.hero
		res := must2(choose.Run(ctx, character.ChoosePaladinOptionCommand{Character: c.hero, Choice: c.choice}))
		if r, ok := res.(character.ChoosePaladinOptionRejected); !ok || !slices.Equal(grpD2PaladinReasons(r.Reasons()), c.want) || !c.hero.Equal(before) {
			t.Errorf("%s = %#v, want rejected %v", c.name, res, c.want)
		}
	}
	if _, ok := must2(choose.Run(ctx, character.ChoosePaladinOptionCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 5)), Choice: character.ChooseBlessedWarriorCantrips{}})).(character.ChoosePaladinOptionNotAPaladin); !ok {
		t.Error("a Fighter must be NotAPaladin")
	}
	for name, cmd := range map[string]character.ChoosePaladinOptionCommand{
		"a zero character": {Choice: character.ChooseBlessedWarriorCantrips{}}, "a nil choice": {Character: grpD2Paladin(t, prog, 2, nil)},
		"a zero feat": {Character: grpD2Paladin(t, prog, 2, nil), Choice: character.ChoosePaladinFightingStyle{}},
	} {
		_, err := choose.Run(ctx, cmd)
		grpD2WantInvalid(t, err, name)
	}

	free := must2(character.NewUseFreeCast(prog))
	spentSmite := grpD2Paladin(t, prog, 5, &character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, PaladinsSmite: character.LongRestUseSpent})
	for _, c := range []struct {
		name    string
		hero    character.Character
		feature character.FreeCastFeature
		want    []string
	}{
		{"Smite at level 1", grpD2Paladin(t, prog, 1, nil), character.FreeCastPaladinsSmite, []string{"feature_not_gained"}},
		{"Steed at level 4", grpD2Paladin(t, prog, 4, nil), character.FreeCastFaithfulSteed, []string{"feature_not_gained"}},
		{"Smite spent", spentSmite, character.FreeCastPaladinsSmite, []string{"already_used_since_long_rest"}},
	} {
		res := must2(free.Run(ctx, character.UseFreeCastCommand{Character: c.hero, Feature: c.feature}))
		if r, ok := res.(character.UseFreeCastRejected); !ok || !slices.Equal(grpD2PaladinReasons(r.Reasons()), c.want) {
			t.Errorf("%s = %#v, want rejected %v", c.name, res, c.want)
		}
	}
	steedFirst := must2(free.Run(ctx, character.UseFreeCastCommand{Character: spentSmite, Feature: character.FreeCastFaithfulSteed}))
	if _, ok := steedFirst.(character.UseFreeCastApplied); !ok {
		t.Errorf("the Steed free cast with the Smite spent = %#v, want applied (they are separate)", steedFirst)
	}
	if _, ok := must2(free.Run(ctx, character.UseFreeCastCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 5)), Feature: character.FreeCastPaladinsSmite})).(character.UseFreeCastNotAPaladin); !ok {
		t.Error("a Fighter must be NotAPaladin")
	}
	_, err := free.Run(ctx, character.UseFreeCastCommand{Character: spentSmite})
	grpD2WantInvalid(t, err, "an undefined feature")
	_, err = free.Run(ctx, character.UseFreeCastCommand{Feature: character.FreeCastPaladinsSmite})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = character.NewUseFreeCast(character.PaladinProgression{})
	grpD2WantInvalid(t, err, "a zero progression")
	_, err = character.NewChoosePaladinOption(character.PaladinProgression{})
	grpD2WantInvalid(t, err, "a zero progression")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := free.Run(cancelled, character.UseFreeCastCommand{Character: spentSmite, Feature: character.FreeCastFaithfulSteed}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v", err)
	}
}

// A level 20 Oath of Devotion Paladin activates Holy Nimbus: a Bonus Action effect of 10 minutes with Holy Ward (Fiend and Undead), Radiant damage of the Charisma modifier plus
// the Proficiency Bonus and Sunlight, spending no Channel Divinity. A second activation is refused until the use is restored by expending a level 5 slot (SRD 5.2.1,
// "Level 20: Holy Nimbus", p. 57). At level 15 Smite of Protection gives Half Cover when Divine Smite was cast with the aura active, and not otherwise ("Level 15: Smite of Protection", p. 57).
// Requirement: doc/specs/class-paladin.md FR-084, FR-089 to FR-095, FR-112, Table SP, Table N - UseHolyNimbus and ApplySmiteOfProtection through the public API.
// Path: happy and unhappy (boundary levels, state error, rule interaction).
func TestPaladinHolyNimbusAndSmiteOfProtection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := grpD2PaladinProgression(t)
	oath := character.OathOfDevotionSubclassID
	nimbus := must2(character.NewUseHolyNimbus(prog))
	proficiency := character.ProficiencyBonusFor(character.MustLevel(20))
	cha := ability.MustScore(16).Modifier() // +3

	hero := grpD2Paladin(t, prog, 20, nil)
	res := must2(nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: hero, Subclass: oath, Action: character.HolyNimbusActivate, Charisma: cha, Proficiency: proficiency})).(character.UseHolyNimbusApplied)
	eff, has := res.Effect()
	damage, damageActive := eff.RadiantDamage()
	ward, wardActive := eff.HolyWard()
	if !has || damage != 3+6 || !damageActive || !wardActive || !eff.AuraActive() || !eff.Sunlight() || eff.ActionCost() != character.PaladinBonusAction || eff.DurationMinutes() != 10 ||
		!slices.Equal(ward, []content.CreatureType{content.CreatureTypeFiend, content.CreatureTypeUndead}) {
		t.Errorf("Holy Nimbus effect = %+v (damage %d)", eff, damage)
	}
	state := grpD2PaladinState(t, res.Character())
	if state.HolyNimbusUse() != character.LongRestUseSpent || !state.HolyNimbusActive() || state.ChannelDivinityRemaining() != grpD2PaladinState(t, hero).ChannelDivinityRemaining() {
		t.Errorf("state after activating = %+v", state)
	}
	if _, ok := res.Events()[0].(character.HolyNimbusActivated); !ok || len(res.Events()) != 1 {
		t.Errorf("events = %#v", res.Events())
	}
	// A second activation is refused; restoring needs a level 5 slot, and an exhausted pool refuses it.
	again := must2(nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: res.Character(), Subclass: oath, Action: character.HolyNimbusActivate, Charisma: cha, Proficiency: proficiency}))
	if r, ok := again.(character.UseHolyNimbusRejected); !ok || !slices.Equal(grpD2PaladinReasons(r.Reasons()), []string{"already_used_since_long_rest"}) {
		t.Errorf("a second activation = %#v", again)
	}
	levels := character.MustClassLevels([]character.ClassLevel{{Class: "paladin", Level: 20}})
	slots := must2(character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: hero.ID(), Levels: levels, Spells: map[content.SpellID]content.Spell{}})).(character.CreateSpellcastingCreated).State()
	restored := must2(nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: res.Character(), Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: slots, Pool: character.PoolSpellcasting}))
	back, ok := restored.(character.UseHolyNimbusApplied)
	if !ok || grpD2PaladinState(t, back.Character()).HolyNimbusUse() != character.LongRestUseAvailable {
		t.Fatalf("restoring with a slot = %#v", restored)
	}
	if next, hasSlots := back.Spellcasting(); !hasSlots || next.Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(5)) != slots.Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(5))-1 {
		t.Error("restoring must expend one level 5 slot")
	}
	if r, ok := must2(nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: hero, Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: slots, Pool: character.PoolSpellcasting})).(character.UseHolyNimbusRejected); !ok ||
		!slices.Equal(grpD2PaladinReasons(r.Reasons()), []string{"not_expended"}) {
		t.Errorf("restoring an unspent use = %#v", r)
	}
	for _, c := range []struct {
		name     string
		hero     character.Character
		subclass content.SubclassID
	}{{"level 19", grpD2Paladin(t, prog, 19, nil), oath}, {"no oath", hero, ""}} {
		if r, ok := must2(nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: c.hero, Subclass: c.subclass, Action: character.HolyNimbusActivate, Charisma: cha, Proficiency: proficiency})).(character.UseHolyNimbusRejected); !ok ||
			!slices.Equal(grpD2PaladinReasons(r.Reasons()), []string{"feature_not_gained"}) {
			t.Errorf("Holy Nimbus, %s = %#v", c.name, r)
		}
	}
	if _, ok := must2(nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 20)), Subclass: oath, Action: character.HolyNimbusActivate})).(character.UseHolyNimbusNotAPaladin); !ok {
		t.Error("a Fighter must be NotAPaladin")
	}
	_, err := nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: hero, Subclass: oath})
	grpD2WantInvalid(t, err, "an undefined action")
	_, err = nimbus.Run(ctx, character.UseHolyNimbusCommand{Character: hero, Subclass: oath, Action: character.HolyNimbusRestoreWithSlot})
	grpD2WantInvalid(t, err, "a restore with no spellcasting state")
	_, err = character.NewUseHolyNimbus(character.PaladinProgression{})
	grpD2WantInvalid(t, err, "a zero progression")

	// Smite of Protection: the five rows of Table SP.
	protect := must2(character.NewApplySmiteOfProtection(prog))
	for _, c := range []struct {
		label       string
		level       int
		subclass    content.SubclassID
		cast, incap bool
		effect      bool
	}{
		{"before the feature (level 14)", 14, oath, true, false, false}, {"level 15 with a Divine Smite", 15, oath, true, false, true},
		{"no Divine Smite cast", 15, oath, false, false, false}, {"no oath", 15, "", true, false, false}, {"Incapacitated", 15, oath, true, true, false},
	} {
		got := must2(protect.Run(ctx, character.ApplySmiteOfProtectionCommand{Character: grpD2Paladin(t, prog, c.level, nil), Subclass: c.subclass, DivineSmiteCast: c.cast, Charisma: cha, Incapacitated: c.incap}))
		applied, isApplied := got.(character.ApplySmiteOfProtectionApplied)
		if isApplied != c.effect || (isApplied && (applied.Effect().Cover() != character.CoverHalf || !applied.Effect().UntilNextTurn() || len(applied.Events()) != 1)) {
			t.Errorf("Smite of Protection, %s = %#v, want effect %v", c.label, got, c.effect)
		}
		if _, noEffect := got.(character.ApplySmiteOfProtectionNoEffect); noEffect == c.effect {
			t.Errorf("Smite of Protection, %s = %#v", c.label, got)
		}
	}
	if _, ok := must2(protect.Run(ctx, character.ApplySmiteOfProtectionCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 15)), Subclass: oath, DivineSmiteCast: true})).(character.ApplySmiteOfProtectionNotAPaladin); !ok {
		t.Error("a Fighter must be NotAPaladin")
	}
	_, err = protect.Run(ctx, character.ApplySmiteOfProtectionCommand{Subclass: oath, DivineSmiteCast: true})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = character.NewApplySmiteOfProtection(character.PaladinProgression{})
	grpD2WantInvalid(t, err, "a zero progression")
}
