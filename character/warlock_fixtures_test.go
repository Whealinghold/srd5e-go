package character_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// warSource is the bundled content source (it holds the Warlock and its spells).
func warSource(tb testing.TB) character.ContentSource {
	tb.Helper()
	return must(data.NewBundledSource())
}

// warDef is the bundled Warlock ClassDefinition (content/data/class_warlock.json).
func warDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	res := must(warSource(tb).Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("warlock"))))
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

// warSpell is the bundled spell with the id.
func warSpell(tb testing.TB, id string) content.Spell {
	tb.Helper()
	res := must(warSource(tb).Lookup(context.Background(), content.NewKey(content.KindSpell, content.MustID(id))))
	return res.(content.LookupFound).Entry.(content.Spell)
}

// warProfile is the profile of a Warlock of level n with the subclass ("" for none).
func warProfile(n int, subclass string) character.WarlockProfile {
	return must(character.NewWarlockProfile(character.MustWarlockLevel(n), content.SubclassID(subclass)))
}

// warFiend is the profile of a Fiend Patron Warlock of level n.
func warFiend(n int) character.WarlockProfile { return warProfile(n, "fiend_patron") }

// warState restores a state at the profile from parts.
func warState(p character.WarlockProfile, parts character.WarlockStateParts) character.WarlockState {
	return must(character.RestoreWarlockState(p, parts))
}

// warKnows is a state at level 20 that knows the invocations.
func warKnows(ids ...content.FeatureOptionID) character.WarlockState {
	return warState(warProfile(20, ""), character.WarlockStateParts{Invocations: ids})
}

// warScores are the ability scores of a Warlock with the Charisma score.
func warScores(t *testing.T, charisma int) ability.Scores {
	t.Helper()
	return must(flatScores(t, 10).With(ability.Charisma, ability.MustScore(charisma)))
}

// warChar is a Warlock of level n with the Charisma score, holding the state and, when subclass is not empty, that subclass.
func warChar(t *testing.T, n, charisma int, subclass string, state character.WarlockState) character.Character {
	t.Helper()
	c := restored(t, 355000, 200, warScores(t, charisma), cl("warlock", n))
	if subclass != "" {
		key := must(character.SubclassStateKey{}.ForClass("warlock"))
		c = must(character.WithFeatureState(c, key, must(character.NewChosenSubclass(content.SubclassID(subclass)))))
	}
	return must(character.WithFeatureState(c, character.WarlockStateKey(), state))
}

// warStateOf reads the state c stores.
func warStateOf(t *testing.T, c character.Character) character.WarlockState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.WarlockStateKey())
	if err != nil || !ok {
		t.Fatalf("WarlockState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// warMod is the ability modifier of a score.
func warMod(score int) ability.Modifier { return ability.MustScore(score).Modifier() }

// warTelemetry is a recording bundle and the options that inject it with a stepping clock.
func warTelemetry() (*telemetrytest.Recorder, []character.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), time.Second)
	return rec, []character.Option{character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)}
}

// warOptions are the Eldritch Invocation options of the bundled Warlock.
func warOptions(tb testing.TB) []content.FeatureOption {
	tb.Helper()
	var out []content.FeatureOption
	for _, f := range warDef(tb).Features() {
		out = append(out, f.Options()...)
	}
	return out
}

// warOption is the invocation option with the id.
func warOption(tb testing.TB, id string) content.FeatureOption {
	tb.Helper()
	for _, o := range warOptions(tb) {
		if string(o.ID()) == id {
			return o
		}
	}
	tb.Fatalf("no option %s", id)
	return content.FeatureOption{}
}

// warFacts builds the facts of a cantrip.
func warFacts(id string, effect character.CantripEffect, r content.SpellRange) character.CantripFacts {
	return must(character.NewCantripFacts(content.SpellID(id), effect, r))
}

func warFeet(n int) content.SpellRange {
	return must(content.NewSpellRangeDistance(n, content.DistanceFeet))
}

// warCantrips are the facts of table 4.6.8 of class-warlock.md, rows CC-01 to CC-07.
func warCantrips() map[string]character.CantripFacts {
	return map[string]character.CantripFacts{
		"chill_touch":      warFacts("chill_touch", character.CantripDamageViaAttackRoll, content.NewSpellRangeTouch()),
		"eldritch_blast":   warFacts("eldritch_blast", character.CantripDamageViaAttackRoll, warFeet(120)),
		"mage_hand":        warFacts("mage_hand", character.CantripNoDamage, warFeet(30)),
		"minor_illusion":   warFacts("minor_illusion", character.CantripNoDamage, warFeet(30)),
		"poison_spray":     warFacts("poison_spray", character.CantripDamageViaAttackRoll, warFeet(30)),
		"prestidigitation": warFacts("prestidigitation", character.CantripNoDamage, warFeet(10)),
		"true_strike":      warFacts("true_strike", character.CantripDamageViaAttackRoll, content.NewSpellRangeSelf()),
	}
}

// warCasting is the spellcasting state of a Warlock of level n with the Pact Magic slots expended count times.
func warCasting(t *testing.T, n, expended int) character.SpellcastingState {
	t.Helper()
	s := castingState(t, classLevelsOf(t, "warlock", n), spellCatalogue(t))
	level := character.PoolPactMagic
	for range expended {
		l := 1
		for ; l <= 5; l++ {
			if s.Slots(level).Maximum(slotLevel(l)) > 0 {
				break
			}
		}
		s = s.ExpendSlot(level, slotLevel(l)).(character.SlotExpenditureExpended).State()
	}
	return s
}

// warModifier is the ability modifier m, built from the score that gives it (-5 to +10).
func warModifier(m int) ability.Modifier { return ability.MustScore(max(10+2*m, 1)).Modifier() }
