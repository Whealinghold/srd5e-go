package scenarios_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

func paladinDefinition(t *testing.T) content.ClassDefinition {
	t.Helper()
	bundle := must2(data.NewBundledSource())
	return must2(bundle.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("paladin")))).(content.LookupFound).Entry.(content.ClassDefinition)
}

// The bundled Paladin definition agrees with the rules code at all 20 levels: the Proficiency Bonus of the Features table equals ProficiencyBonusFor, there is no Cantrips
// column, the Prepared Spells equal PreparedLimit and the nine Spell Slot counts equal what SlotsFor derives for the Paladin alone (SRD 5.2.1, "Paladin Features", p. 53).
// Cites class-paladin.md FR-057, content-classes-data.md FR-033, FR-035.
func TestPaladinTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	def := paladinDefinition(t)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must2(def.RowAt(n))
			sc := row.Spellcasting().(content.SlotSpellcastingRow)
			_, hasCantrips := sc.Cantrips()
			slots := character.SlotsFor(character.MustClassLevels([]character.ClassLevel{{Class: "paladin", Level: n}})).Spellcasting()
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() || hasCantrips || sc.PreparedSpells() != character.PreparedLimit(character.CasterPaladin, n) {
				t.Errorf("bonus %d, cantrips %v, prepared %d (rules %d)", row.ProficiencyBonus(), hasCantrips, sc.PreparedSpells(), character.PreparedLimit(character.CasterPaladin, n))
			}
			for l := 1; l <= 9; l++ {
				if got, want := sc.SpellSlots()[l-1], slots.Maximum(character.MustSlotLevel(l)); got != want {
					t.Errorf("level %d slots = %d, rules = %d", l, got, want)
				}
			}
		})
	}
}

// A level 1 Paladin is created through PaladinCreationChoices, advances to level 3 (the host applying AdvancePaladin after each level) and takes the Oath of Devotion, reaches level 6
// with the aura, uses Lay On Hands and Channel Divinity until it is spent, then takes a Short and a Long Rest (the host applying ApplyPaladinRest after each), through the public
// packages only (SRD 5.2.1, "Paladin Features", pp. 53-55; "Oath of Devotion", p. 56). Cites class-paladin.md FR-043, FR-046, FR-065 to FR-067, FR-077, FR-098, FR-111, FR-122.
func TestPaladinFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	def := paladinDefinition(t)
	oath := def.Subclasses()[0]
	progression := must2(character.NewPaladinProgression(def, &oath))
	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(16), ability.Dexterity: ability.MustScore(10), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(8), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(16),
	}))
	hero := func(level, xp int, seed bool) character.Character {
		classes := character.MustClassLevels([]character.ClassLevel{{Class: "paladin", Level: level}})
		states := character.FeatureStates{}
		if seed {
			states = character.NewPaladinCreationChoices(character.WeaponMasteryChoice{}).SeedFeatureStates(states)
		}
		return must2(character.Restore(character.Parts{
			ID: character.MustIdentifier("hero"), ExperiencePoints: character.MustExperiencePoints(xp), ClassLevels: classes, Species: "human", Background: "soldier",
			AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(80),
			HitDicePool:       character.MustHitDicePoolFromLevels(classes, map[content.ClassID]content.HitPointDie{"paladin": content.HitDieD10}),
			HeroicInspiration: character.NotInspired, FeatureStates: states,
		}))
	}
	stateOf := func(c character.Character) character.PaladinState {
		s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.PaladinStateKey())
		if err != nil || !ok {
			t.Fatalf("PaladinState = %v, %v, %v", s, ok, err)
		}
		return s
	}
	subclass := content.SubclassID("")
	xp := []int{0, 0, 900, 2700}
	// Level 1 pool, then the advance to 2 and 3 carrying the state across levels.
	state := stateOf(hero(1, 0, true))
	if state.LayOnHandsRemaining() != 5 || state.ChannelDivinityRemaining() != 0 {
		t.Fatalf("level 1 state = %+v", state)
	}
	cur := hero(1, 0, true)
	for level := 2; level <= 3; level++ {
		next := hero(level, xp[level], false)
		next = must2(character.WithFeatureState(next, character.PaladinStateKey(), stateOf(cur)))
		if level == 3 {
			subclass = character.OathOfDevotionSubclassID
		}
		adv := must2(character.AdvancePaladin(next, level-1, progression, subclass)).(character.AdvancePaladinApplied)
		cur = adv.Character()
	}
	s := stateOf(cur)
	if s.LayOnHandsRemaining() != 15 || s.ChannelDivinityRemaining() != 2 {
		t.Fatalf("level 3 state = %+v", s)
	}
	// Level 6: the aura, from the summary.
	cur = hero(6, 14000, false)
	cur = must2(character.WithFeatureState(cur, character.PaladinStateKey(), character.InitialPaladinState(character.MustPaladinLevel(6), 2)))
	cha := scores.Modifier(ability.Charisma)
	summary := must2(must2(character.NewCalculatePaladinFeatures(progression)).Run(ctx, character.CalculatePaladinFeaturesCommand{Character: cur, Subclass: subclass, Charisma: cha})).(character.CalculatePaladinFeaturesSummary)
	if !summary.Auras().Active() || summary.Auras().RadiusFeet() != 10 || summary.Auras().SaveBonus() != 3 || summary.AttacksPerAttackAction() != 2 || len(summary.AlwaysPreparedSpells()) != 6 {
		t.Fatalf("level 6 summary = %+v", summary)
	}
	// Lay On Hands, then both Channel Divinity uses.
	lay := must2(must2(character.NewUseLayOnHands(progression)).Run(ctx, character.UseLayOnHandsCommand{Character: cur, HitPoints: 10, Remove: []content.ConditionKind{content.ConditionPoisoned}})).(character.UseLayOnHandsApplied)
	if lay.Outcome().Remaining() != 15 || lay.Outcome().HitPoints() != 10 {
		t.Errorf("Lay On Hands = %+v", lay.Outcome())
	}
	cur = lay.Character()
	channel := must2(character.NewUsePaladinChannelDivinity(progression))
	dc := character.Stats(character.CasterPaladin, cha, character.ProficiencyBonusFor(cur.TotalLevel())).SaveDC()
	for i := 0; i < 2; i++ {
		res := must2(channel.Run(ctx, character.UsePaladinChannelDivinityCommand{Character: cur, Option: character.PaladinChannelSacredWeapon, Subclass: subclass, Charisma: cha, SpellSaveDC: dc,
			TakingAttackAction: true, HoldingMeleeWeapon: true, Weapon: "longsword"})).(character.UsePaladinChannelDivinityApplied)
		if res.Effect().(character.SacredWeaponEffect).AttackBonus() != 3 {
			t.Errorf("Sacred Weapon bonus = %d", res.Effect().(character.SacredWeaponEffect).AttackBonus())
		}
		cur = res.Character()
	}
	if r, ok := must2(channel.Run(ctx, character.UsePaladinChannelDivinityCommand{Character: cur, Option: character.PaladinChannelDivineSense, SpellSaveDC: dc})).(character.UsePaladinChannelDivinityRejected); !ok ||
		!slices.Equal(r.Reasons(), []character.PaladinReason{character.PaladinNoUsesRemaining}) {
		t.Errorf("third use = %#v", r)
	}
	// Rests: a Short Rest regains one use, a Long Rest refills the pool and the rest of the uses.
	st := stateOf(cur)
	short := character.ApplyPaladinRest(cur.ID(), progression, character.MustPaladinLevel(6), st, character.ShortRest)
	long := character.ApplyPaladinRest(cur.ID(), progression, character.MustPaladinLevel(6), short.State(), character.LongRest)
	if short.State().ChannelDivinityRemaining() != 1 || short.State().LayOnHandsRemaining() != 15 || long.State().ChannelDivinityRemaining() != 2 || long.State().LayOnHandsRemaining() != 30 {
		t.Errorf("after rests = %+v, %+v", short.State(), long.State())
	}
}
