package scenarios_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// The bundled Warlock definition agrees with the rules code at all 20 levels: the Proficiency Bonus of the Features table equals ProficiencyBonusFor, the Cantrips and
// Prepared Spells columns equal CantripLimit and PreparedLimit, the Pact Magic Spell Slots and Slot Level equal SlotsFor, and the Mystic Arcanum entries of the Class
// Features cell equal ArcanumCount (SRD 5.2.1, "Warlock Features", p. 71). Cites class-warlock.md FR-117, content-classes-data.md FR-033, FR-035.
func TestWarlockTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle := must2(data.NewBundledSource())
	def := must2(bundle.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("warlock")))).(content.LookupFound).Entry.(content.ClassDefinition)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must2(def.RowAt(n))
			pact, isPact := row.Spellcasting().(content.PactSpellcastingRow)
			slots := character.SlotsFor(character.MustClassLevels([]character.ClassLevel{{Class: "warlock", Level: n}})).PactMagic()
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() {
				t.Errorf("Proficiency Bonus = %d", row.ProficiencyBonus())
			}
			if !isPact || pact.PreparedSpells() != character.PreparedLimit(character.CasterWarlock, n) || pact.Cantrips() != character.CantripLimit(character.CasterWarlock, n) {
				t.Errorf("prepared %d (rules %d), cantrips %d (rules %d)", pact.PreparedSpells(), character.PreparedLimit(character.CasterWarlock, n), pact.Cantrips(), character.CantripLimit(character.CasterWarlock, n))
			}
			if pact.SpellSlots() != slots.Count() || pact.SlotLevel() != slots.Level().Value() {
				t.Errorf("Pact Magic = %d slots of level %d, rules = %d of %d", pact.SpellSlots(), pact.SlotLevel(), slots.Count(), slots.Level().Value())
			}
			arcanum := 0
			for _, f := range row.Features() {
				if rf, ok := f.(content.RowClassFeature); ok && rf.FeatureID() == "warlock_mystic_arcanum" {
					arcanum++
				}
			}
			// The cell lists the arcanum entry once, at the level that gives a new spell, so the rules count rises there.
			rises := n >= 11 && character.MustWarlockLevel(n).ArcanumCount() > character.MustWarlockLevel(n-1).ArcanumCount()
			if (arcanum == 1) != rises {
				t.Errorf("Mystic Arcanum entries = %d, rules say a new spell: %v", arcanum, rises)
			}
		})
	}
}

// Every option of the bundled Eldritch Invocations feature that names a cantrip kind is one the rules decide by id: the printed notes are exactly the two cantrip requirements
// and are carried by Agonizing Blast, Eldritch Spear and Repelling Blast alone. Cites class-warlock.md section 4.6.1 (the guard test) and FR-047.
func TestWarlockInvocationNotesAreTheCantripRequirements(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	def := must2(must2(data.NewBundledSource()).Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("warlock")))).(content.LookupFound).Entry.(content.ClassDefinition)
	notes := map[string]string{}
	for _, f := range def.Features() {
		for _, o := range f.Options() {
			if p, ok := o.Prerequisite(); ok && p.Note() != "" {
				notes[string(o.ID())] = p.Note()
			}
		}
	}
	want := map[string]string{"agonizing_blast": "a Warlock Cantrip That Deals Damage", "eldritch_spear": "a Warlock Cantrip That Deals Damage",
		"repelling_blast": "a Warlock Cantrip That Deals Damage via an Attack Roll"}
	if len(notes) != len(want) {
		t.Fatalf("notes = %v, want %v", notes, want)
	}
	for id, note := range want {
		if notes[id] != note {
			t.Errorf("%s note = %q, want %q", id, notes[id], note)
		}
	}
}

// A level 5 Fiend Patron Warlock through the public API (class-warlock.md section 8, whole slice): the bundled class gives the always-prepared Fiend Spells of levels 3 and 5 (Fireball
// among them, which a Warlock otherwise cannot prepare), ChooseWarlockInvocations stores five invocations, a Short Rest regains the Pact Magic slots through the spellcasting state, Eldritch Smite
// returns an expend of the level 3 Pact Magic slot with 4d8 Force damage, Dark One's Blessing gives Temporary Hit Points, Dark One's Own Luck rolls through a fixed roller, Magical Cunning regains
// a slot, and a Long Rest restores the uses.
func TestWarlockFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := must2(data.NewBundledSource())
	def := must2(src.Lookup(ctx, content.NewKey(content.KindCharacterClass, content.MustID("warlock")))).(content.LookupFound).Entry.(content.ClassDefinition)
	scores := must2(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(10), ability.Dexterity: ability.MustScore(14), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(16),
	}))
	levels := character.MustClassLevels([]character.ClassLevel{{Class: "warlock", Level: 5}})
	subclass := must2(character.NewChosenSubclass("fiend_patron"))
	states := must2(character.SetFeatureState(character.WarlockCreationChoices{}.SeedFeatureStates(character.FeatureStates{}), must2(character.SubclassStateKey{}.ForClass("warlock")), subclass))
	hero := must2(character.Restore(character.Parts{
		ID: character.MustIdentifier("warlock"), ExperiencePoints: character.MustExperiencePoints(6500), ClassLevels: levels, Species: "human", Background: "hermit",
		AbilityScores: scores, HitPointMaximum: character.MustHitPointMaximum(40),
		HitDicePool:       character.MustHitDicePoolFromLevels(levels, map[content.ClassID]content.HitPointDie{"warlock": content.HitDieD8}),
		HeroicInspiration: character.NotInspired, FeatureStates: states,
	}))
	profile, isWarlock, err := character.WarlockProfileOf(hero)
	if err != nil || !isWarlock || profile.Level().Value() != 5 || profile.Subclass() != character.FiendPatronSubclassID {
		t.Fatalf("profile = %v, %v, %v", profile, isWarlock, err)
	}
	always := character.WarlockAlwaysPrepared(profile, def)
	if want := []content.SpellID{"burning_hands", "command", "scorching_ray", "suggestion", "fireball", "stinking_cloud"}; !slices.Equal(always, want) {
		t.Errorf("always prepared = %v, want %v", always, want)
	}
	fireball := must2(src.Lookup(ctx, content.NewKey(content.KindSpell, content.MustID("fireball")))).(content.LookupFound).Entry.(content.Spell)
	if slices.Contains(fireball.Classes(), character.WarlockClassID) {
		t.Error("Fireball is on the Warlock list")
	}

	facts := []character.CantripFacts{must2(character.NewCantripFacts("eldritch_blast", character.CantripDamageViaAttackRoll, must2(content.NewSpellRangeDistance(120, content.DistanceFeet))))}
	picks := []character.InvocationPick{{Option: "pact_of_the_blade"}, {Option: "agonizing_blast", Cantrip: "eldritch_blast"}, {Option: "devils_sight"}, {Option: "eldritch_smite"}, {Option: "armor_of_shadows"}}
	chooser := must2(character.NewChooseWarlockInvocations(src))
	chosen := must2(chooser.Run(ctx, character.ChooseWarlockInvocationsCommand{Character: hero, Picks: picks, KnownCantrips: facts})).(character.ChooseWarlockInvocationsApplied)
	hero = chosen.Character()
	state, _, _ := character.ReadFeatureState(hero.FeatureStates(), character.WarlockStateKey())
	if !state.Knows("eldritch_smite") || len(state.Invocations()) != 5 {
		t.Fatalf("invocations = %v", state.Invocations())
	}
	rejected := must2(chooser.Run(ctx, character.ChooseWarlockInvocationsCommand{Character: hero, Picks: picks[:4], KnownCantrips: facts})).(character.ChooseWarlockInvocationsRejected)
	if rejected.Rejection.Kind != character.WarlockInvocationSelectionInvalid {
		t.Errorf("four invocations at level 5 = %+v", rejected.Rejection)
	}

	// The Pact Magic slots belong to the spellcasting state: smite with a level 3 slot, rest, and the slots are back.
	cast := must2(character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: hero.ID(), Levels: levels, Spells: map[content.SpellID]content.Spell{}})).(character.CreateSpellcastingCreated).State()
	smite := character.UseEldritchSmite(hero.ID(), state, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, cast, content.SizeLarge, character.SmiteApplyProne).(character.WarlockChanged)
	change, _ := smite.SlotChange()
	damage, _ := smite.Damage()
	if change != character.WarlockSlotChange(character.WarlockSlotChangeExpendPact{Level: 3}) || damage.String() != "4d8" {
		t.Errorf("smite = %v, %v", change, damage)
	}
	cast = cast.ExpendSlot(character.PoolPactMagic, character.MustSlotLevel(3)).(character.SlotExpenditureExpended).State()
	cast = cast.ExpendSlot(character.PoolPactMagic, character.MustSlotLevel(3)).(character.SlotExpenditureExpended).State()
	if r, ok := character.UseEldritchSmite(hero.ID(), state, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, cast, content.SizeLarge, 0).(character.WarlockRefused); !ok || r.Rejection().Kind != character.WarlockNoPactMagicSlot {
		t.Errorf("both slots spent = %#v", r)
	}
	cunning := character.CompleteMagicalCunning(hero.ID(), state, profile, time.Minute, cast).(character.WarlockChanged)
	if regain, _ := cunning.SlotChange(); regain != character.WarlockSlotChange(character.WarlockSlotChangeRegainPact{Count: 1}) {
		t.Errorf("Magical Cunning = %v", regain)
	}
	cast, _ = cast.RestoreSlots(character.ShortRest)
	if cast.Slots(character.PoolPactMagic).Remaining(character.MustSlotLevel(3)) != 2 {
		t.Error("a Short Rest must regain the Pact Magic slots")
	}

	blessing := character.ResolveDarkOnesBlessing(hero.ID(), state, profile, ability.MustScore(16).Modifier(), character.TargetEnemy, character.BlessingByWarlock, 0).(character.WarlockChanged)
	if thp := blessing.Events()[0].(character.DarkOnesBlessingGained).TemporaryHitPoints(); thp != 8 {
		t.Errorf("Dark One's Blessing = %d, want Charisma +3 and level 5", thp)
	}
	// Dark One's Own Luck is gained at level 6.
	luck := must2(character.NewUseDarkOnesOwnLuck(dicetest.NewFixedRoller(4)))
	if r, ok := must2(luck.Run(ctx, character.UseDarkOnesOwnLuckCommand{Character: hero, Charisma: ability.MustScore(16).Modifier(), Kind: d20.SavingThrow})).(character.UseDarkOnesOwnLuckRejected); !ok || r.Rejection.Kind != character.WarlockFeatureNotGained {
		t.Errorf("level 5 luck = %+v", r)
	}

	used := state
	if tr, ok := character.CompleteMagicalCunning(hero.ID(), used, profile, time.Minute, cast).(character.WarlockChanged); ok {
		used = tr.State()
	}
	rested := must2(character.ApplyWarlockRest(hero.ID(), used, character.LongRest)).(character.WarlockChanged)
	if rested.State().MagicalCunningUsed() || len(rested.Events()) != 1 {
		t.Errorf("a Long Rest must restore Magical Cunning: %+v", rested.Events())
	}
}
