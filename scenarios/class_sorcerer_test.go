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
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// The bundled Sorcerer definition agrees with the rules code at all 20 levels: the Proficiency Bonus of the Features table equals ProficiencyBonusFor, the Sorcery Points column equals
// SorcererLevel.SorceryPointsMaximum (a printed dash at level 1), the Prepared Spells and Cantrips columns equal PreparedLimit and CantripLimit and the Spell Slots columns equal SlotsFor
// (SRD 5.2.1, "Sorcerer Features", p. 65). Cites class-sorcerer.md FR-017, FR-102 and content-classes-data.md FR-033, FR-035.
func TestSorcererTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle := must2(data.NewBundledSource())
	def := must2(bundle.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("sorcerer")))).(content.LookupFound).Entry.(content.ClassDefinition)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must2(def.RowAt(n))
			sc, isSlots := row.Spellcasting().(content.SlotSpellcastingRow)
			cantrips, _ := sc.Cantrips()
			points, _ := row.Value("sorcery_points")
			gotPoints := 0
			if v, ok := points.(content.ColumnCount); ok {
				gotPoints = v.Value()
			}
			slots := character.SlotsFor(character.MustClassLevels([]character.ClassLevel{{Class: "sorcerer", Level: n}})).Spellcasting()
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() {
				t.Errorf("Proficiency Bonus = %d", row.ProficiencyBonus())
			}
			if gotPoints != character.MustSorcererLevel(n).SorceryPointsMaximum() {
				t.Errorf("Sorcery Points column = %d, rules = %d", gotPoints, character.MustSorcererLevel(n).SorceryPointsMaximum())
			}
			if !isSlots || sc.PreparedSpells() != character.PreparedLimit(character.CasterSorcerer, n) || cantrips != character.CantripLimit(character.CasterSorcerer, n) {
				t.Errorf("prepared %d (rules %d), cantrips %d (rules %d)", sc.PreparedSpells(), character.PreparedLimit(character.CasterSorcerer, n), cantrips, character.CantripLimit(character.CasterSorcerer, n))
			}
			for l := 1; l <= 9; l++ {
				if sc.SpellSlots()[l-1] != slots.Maximum(character.MustSlotLevel(l)) {
					t.Errorf("level %d slots = %d, rules = %d", l, sc.SpellSlots()[l-1], slots.Maximum(character.MustSlotLevel(l)))
				}
			}
		})
	}
}

// A Sorcerer seeded by SorcererCreationChoices climbs from level 1 to 20 through the real aggregate and ApplySorcererLevelUp: the Sorcery Points follow the table, the Metamagic options owed come at levels 2, 10 and 17,
// and the feat grants at 4, 8, 12, 16 and 19; then, at level 9, Font of Magic creates a level 5 slot, SpendCreatedSlot spends it on a level 3 spell, Innate Sorcery is activated and a Metamagic option is used, and a Long Rest
// removes the created slot and restores the points; a Fighter 4 / Sorcerer 3 multiclass is a caster level 3 Sorcerer whose slots are the Spellcasting slots of caster level 3. Cites class-sorcerer.md FR-010, FR-011, FR-035, FR-037,
// FR-047, FR-052, FR-054, FR-059, FR-085, FR-086, FR-127, FR-129, FR-136.
func TestSorcererFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := must2(data.NewBundledSource())
	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(14), ability.Dexterity: ability.MustScore(14), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(16),
	}))
	reqs := map[content.ClassID]content.AbilityRequirement{"sorcerer": must2(content.NewAllOf(ability.Charisma)), "fighter": must2(content.NewAnyOf(ability.Strength, ability.Dexterity))}
	classes := func(entries ...character.ClassLevel) character.ClassLevels { return character.MustClassLevels(entries) }
	hero := func(l character.ClassLevels, states character.FeatureStates) character.Character {
		return must2(character.Restore(character.Parts{
			ID: character.MustIdentifier("sorcerer"), ExperiencePoints: character.MustExperiencePoints(355000), ClassLevels: l, Species: "human", Background: "hermit",
			AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(100),
			HitDicePool:       character.MustHitDicePoolFromLevels(l, map[content.ClassID]content.HitPointDie{"sorcerer": content.HitDieD6, "fighter": content.HitDieD10}),
			HeroicInspiration: character.NotInspired, FeatureStates: states,
		}))
	}
	levelUp := func(c character.Character, class content.ClassID) (character.Character, character.LevelGained) {
		die := map[content.ClassID]content.HitPointDie{"sorcerer": content.HitDieD6, "fighter": content.HitDieD10}[class]
		adv := must2(character.NewAdvancementClass(class, die, reqs[class]))
		tr := must2(c.LevelUp(adv, must2(character.NewFixedHitPointGain(4)), reqs)).(character.TransitionChanged[character.LevelUpRejection])
		for _, e := range tr.Events() {
			if lg, ok := e.(character.LevelGained); ok {
				return tr.Character(), lg
			}
		}
		t.Fatal("no LevelGained event")
		return character.Character{}, character.LevelGained{}
	}
	state := func(c character.Character) character.SorcererState {
		s, _, _ := character.ReadFeatureState(c.FeatureStates(), character.SorcererStateKey())
		return s
	}
	apply := must2(character.NewApplySorcererLevelUp(src))
	owedAt := map[int]int{2: 2, 10: 2, 17: 2}
	feats := map[int]content.FeatCategory{4: content.FeatGeneral, 8: content.FeatGeneral, 12: content.FeatGeneral, 16: content.FeatGeneral, 19: content.FeatEpicBoon}
	choices := character.SorcererCreationChoices{}
	c := hero(classes(character.ClassLevel{Class: "sorcerer", Level: 1}), choices.SeedFeatureStates(character.FeatureStates{}))
	if s := state(c); s.SorceryPoints() != 0 || s.InnateSorceryUses() != 2 {
		t.Fatalf("seeded state = %+v", s.Parts())
	}
	known := 0
	for n := 2; n <= 20; n++ {
		next, gained := levelUp(c, "sorcerer")
		var pick []content.FeatureOptionID
		if owedAt[n] != 0 {
			effects := []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicDistant, character.MetamagicEmpowered, character.MetamagicExtended, character.MetamagicHeightened, character.MetamagicQuickened}
			pick = make([]content.FeatureOptionID, 0, 6)
			for _, e := range effects[:known+owedAt[n]] {
				pick = append(pick, e.OptionID())
			}
		}
		res := must2(apply.Run(ctx, character.ApplySorcererLevelUpCommand{Character: next, Gained: gained, Metamagic: pick})).(character.ApplySorcererLevelUpApplied)
		c = res.Character()
		known += owedAt[n]
		if got := state(c); got.SorceryPoints() != n || len(got.Metamagic()) != known || res.OwedMetamagicOptions() != 0 {
			t.Errorf("level %d: %d points, %d options known, %d owed; want %d, %d and 0", n, got.SorceryPoints(), len(got.Metamagic()), res.OwedMetamagicOptions(), n, known)
		}
		want := []content.FeatCategory{}
		if f, ok := feats[n]; ok {
			want = append(want, f)
		}
		if !slices.Equal(res.Grants().FeatGrants(), want) {
			t.Errorf("level %d feat grants = %v, want %v", n, res.Grants().FeatGrants(), want)
		}
	}

	// Level 9 Draconic Sorcerer with the options: create a level 5 slot (7 points), spend it on a level 3 spell, activate Innate Sorcery, use Twinned-like Careful, then a Long Rest.
	profile := must2(character.NewSorcererProfile(character.MustSorcererLevel(9), character.DraconicSorcerySubclassID))
	nine := must2(character.RestoreSorcererState(profile, character.SorcererStateParts{SorceryPoints: 9, InnateSorceryUses: 2,
		Metamagic: []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicDistant}}))
	hero9 := must2(character.WithFeatureState(hero(classes(character.ClassLevel{Class: "sorcerer", Level: 9}), character.FeatureStates{}), character.SorcererStateKey(), nine))
	make5 := must2(character.NewCreateSpellSlot(src))
	created := make5.Run
	res := must2(created(ctx, character.CreateSpellSlotCommand{Character: hero9, SlotLevel: 5})).(character.CreateSpellSlotApplied)
	if s := state(res.Character()); s.CreatedSlots(5) != 1 || s.SorceryPoints() != 2 {
		t.Fatalf("after creating a level 5 slot = %+v", s.Parts())
	}
	spent := must2(character.SpendCreatedSlot(res.Character().ID(), state(res.Character()), must2(character.NewCastingContext(character.CastingFacts{CanCastSpells: true})), 5, 3))
	after := spent.(character.SorcererChanged).State()
	if after.CreatedSlots(5) != 0 {
		t.Errorf("created slot not spent: %+v", after.Parts())
	}
	active := character.ActivateInnateSorcery(res.Character().ID(), after, profile, character.InnateSorceryNormalUse).(character.SorcererChanged).State()
	if effects := character.SorcererInnateSorceryEffects(active); effects.SaveDC.Value() != 1 || effects.AttackRolls != (modifier.Advantage{}) {
		t.Errorf("Innate Sorcery effects = %+v", effects)
	}
	careful := must2(character.NewUseMetamagic(src))
	hit := must2(d20.ResolveAttack(d20.MustRoll(modifier.Normal, 5), d20.NewRollModifiers(modifier.New(0)), d20.ArmorClass(12), d20.DefaultCriticalRange))
	facts := character.MetamagicSpellFacts{ForcesSavingThrow: true, OtherCreaturesSaving: 4, AttackRoll: &hit}
	use := must2(careful.Run(ctx, character.UseMetamagicCommand{Character: hero9, Option: "careful_spell", Facts: facts, Parameters: character.MetamagicParameters{Count: 3}})).(character.UseMetamagicApplied)
	if use.Application().ProtectedCreatures != 3 || state(use.Character()).SorceryPoints() != 8 {
		t.Errorf("Careful Spell = %+v, points %d", use.Application(), state(use.Character()).SorceryPoints())
	}
	rested := character.ApplySorcererRest
	long := must2(rested(res.Character().ID(), state(res.Character()), profile, character.LongRest, 0)).(character.SorcererChanged).State()
	if long.CreatedSlots(5) != 0 || long.SorceryPoints() != 9 {
		t.Errorf("after a Long Rest = %+v, want no created slot and 9 points", long.Parts())
	}

	// Multiclass: Fighter 4 / Sorcerer 3 gains its first Sorcerer level through the aggregate; the Sorcerer slots follow the combined caster level.
	fighter := hero(classes(character.ClassLevel{Class: "fighter", Level: 4}), character.FeatureStates{})
	entrant, gained := levelUp(fighter, "sorcerer")
	seeded := must2(apply.Run(ctx, character.ApplySorcererLevelUpCommand{Character: entrant, Gained: gained})).(character.ApplySorcererLevelUpApplied)
	if s := state(seeded.Character()); s.SorceryPoints() != 0 || s.InnateSorceryUses() != 2 {
		t.Errorf("multiclass entrant = %+v", s.Parts())
	}
	slots := character.SlotsFor(classes(character.ClassLevel{Class: "fighter", Level: 4}, character.ClassLevel{Class: "sorcerer", Level: 3})).Spellcasting()
	if got := []int{slots.Maximum(character.MustSlotLevel(1)), slots.Maximum(character.MustSlotLevel(2)), slots.Maximum(character.MustSlotLevel(3))}; !slices.Equal(got, []int{4, 2, 0}) {
		t.Errorf("Fighter 4 / Sorcerer 3 slots = %v, want 4 and 2 (caster level 3)", got)
	}
}
