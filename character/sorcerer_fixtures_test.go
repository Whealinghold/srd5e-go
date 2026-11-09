package character_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// sorID is the id of the character restored by the aggregate fixtures, the Sorcerer of every Sorcerer test.
var sorID = character.MustIdentifier("pc-1")

// sorSource is the bundled content source (it holds the Sorcerer).
func sorSource(tb testing.TB) character.ContentSource {
	tb.Helper()
	return must(data.NewBundledSource())
}

// sorDef is the bundled Sorcerer ClassDefinition (content/data/class_sorcerer.json).
func sorDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	res := must(sorSource(tb).Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("sorcerer"))))
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

// sorProfile is the profile of a Sorcerer of level n with the subclass ("" for none).
func sorProfile(n int, subclass string) character.SorcererProfile {
	return must(character.NewSorcererProfile(character.MustSorcererLevel(n), content.SubclassID(subclass)))
}

// sorDraconic is the profile of a Draconic Sorcerer of level n.
func sorDraconic(n int) character.SorcererProfile { return sorProfile(n, "draconic_sorcery") }

// sorState restores a state at the profile from parts.
func sorState(p character.SorcererProfile, parts character.SorcererStateParts) character.SorcererState {
	return must(character.RestoreSorcererState(p, parts))
}

// sorFresh is the state a new Sorcerer has: the points of its level and both Innate Sorcery uses.
func sorFresh(p character.SorcererProfile) character.SorcererState {
	return sorState(p, character.SorcererStateParts{SorceryPoints: p.Level().SorceryPointsMaximum(), InnateSorceryUses: 2})
}

// sorWith restores a state at the profile with Sorcery Points, the Metamagic known and a change to the rest of the parts.
func sorWith(p character.SorcererProfile, points int, known []character.MetamagicEffect, edit func(*character.SorcererStateParts)) character.SorcererState {
	parts := character.SorcererStateParts{SorceryPoints: points, InnateSorceryUses: 2, Metamagic: known}
	if edit != nil {
		edit(&parts)
	}
	return sorState(p, parts)
}

// sorScores are the ability scores of a Sorcerer with the Charisma score.
func sorScores(t *testing.T, charisma int) ability.Scores {
	t.Helper()
	return must(flatScores(t, 10).With(ability.Charisma, ability.MustScore(charisma)))
}

// sorChar is a Sorcerer of level n with the Charisma score, holding the state and, when subclass is not empty, that subclass.
func sorChar(t *testing.T, n, charisma int, subclass string, state character.SorcererState) character.Character {
	t.Helper()
	c := restored(t, 355000, 200, sorScores(t, charisma), cl("sorcerer", n))
	if subclass != "" {
		key := must(character.SubclassStateKey{}.ForClass("sorcerer"))
		c = must(character.WithFeatureState(c, key, must(character.NewChosenSubclass(content.SubclassID(subclass)))))
	}
	return must(character.WithFeatureState(c, character.SorcererStateKey(), state))
}

// sorStateOf reads the state c stores.
func sorStateOf(t *testing.T, c character.Character) character.SorcererState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.SorcererStateKey())
	if err != nil || !ok {
		t.Fatalf("SorcererState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// sorLevelUp gains one Sorcerer level and returns the new character and its LevelGained event.
func sorLevelUp(t *testing.T, c character.Character) (character.Character, character.LevelGained) {
	t.Helper()
	reqs := map[content.ClassID]content.AbilityRequirement{"sorcerer": allOf(ability.Charisma), "fighter": anyOf(ability.Strength, ability.Dexterity)}
	class := "sorcerer"
	ch := changedLevel(t, levelUp(t, c, advancement(t, class, reqs["sorcerer"]), fixedGain(4), reqs))
	for _, e := range ch.Events() {
		if lg, ok := e.(character.LevelGained); ok {
			return ch.Character(), lg
		}
	}
	t.Fatal("no LevelGained event")
	return character.Character{}, character.LevelGained{}
}

// sorMod is the ability modifier of a score.
func sorMod(score int) ability.Modifier { return ability.MustScore(score).Modifier() }

// sorOptionIDs are the ids of the Metamagic effects, in the printed order.
func sorOptionIDs(effects ...character.MetamagicEffect) []content.FeatureOptionID {
	var out []content.FeatureOptionID
	for _, e := range effects {
		out = append(out, e.OptionID())
	}
	return out
}

// sorCasting is a CastingContext whose only fact is whether a slot was spent this turn.
func sorCasting(slotSpent bool) character.CastingContext {
	return must(character.NewCastingContext(character.CastingFacts{SlotSpentThisTurn: slotSpent, CanCastSpells: true}))
}

// sorFacts builds spell facts that suit every option: a saving throw spell on three other creatures, with an attack roll that missed,
// Verbal and Somatic components, a range of 60 feet, a 1 minute duration, the Magic action, Fire damage on 8 dice and a higher-level slot target.
func sorFacts(edit func(*character.MetamagicSpellFacts)) character.MetamagicSpellFacts {
	f := character.MetamagicSpellFacts{
		Range:                      must(content.NewSpellRangeDistance(60, content.DistanceFeet)),
		Duration:                   must(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitFixed)),
		CastingTime:                content.NewCastingTimeMagicAction(),
		Components:                 content.NewSpellComponents(true, true),
		ForcesSavingThrow:          true,
		OtherCreaturesSaving:       5,
		AttackRoll:                 sorAttack(false),
		DamageTypes:                []content.DamageType{content.DamageFire},
		DamageDice:                 8,
		HigherSlotAdditionalTarget: true,
	}
	if edit != nil {
		edit(&f)
	}
	return f
}

// sorAttack is a spell attack roll that hit or missed.
func sorAttack(hit bool) *d20.AttackRollResult {
	natural := 5
	if hit {
		natural = 15
	}
	r := must(d20.ResolveAttack(d20.MustRoll(modifier.Normal, natural), d20.NewRollModifiers(modifier.New(0)), d20.ArmorClass(12), d20.DefaultCriticalRange))
	return &r
}

// sorTelemetry is a recording bundle and the options that inject it with a stepping clock.
func sorTelemetry() (*telemetrytest.Recorder, []character.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), time.Second)
	return rec, []character.Option{character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)}
}

// sorSelection is the valid Metamagic selection of the effects for a Sorcerer of the level.
func sorSelection(t *testing.T, level int, effects ...character.MetamagicEffect) character.MetamagicSelection {
	t.Helper()
	var listed []content.FeatureOption
	for _, f := range sorDef(t).Features() {
		listed = append(listed, f.Options()...)
	}
	sel, problems := character.NewMetamagicSelection(character.MustSorcererLevel(level), sorOptionIDs(effects...), listed)
	if len(problems) != 0 {
		t.Fatalf("selection problems = %v", problems)
	}
	return sel
}

// sorRenamed is c under another character id.
func sorRenamed(t *testing.T, c character.Character, id string) character.Character {
	t.Helper()
	return must(character.Restore(character.Parts{ID: character.MustIdentifier(id), ExperiencePoints: c.ExperiencePoints(), ClassLevels: c.ClassLevels(), Species: c.Species(),
		Background: c.Background(), AbilityScores: c.AbilityScores(), HitPointMaximum: c.HitPointMaximum(), HitDicePool: c.HitDicePool(),
		HeroicInspiration: c.HeroicInspiration(), FeatureStates: c.FeatureStates()}))
}
