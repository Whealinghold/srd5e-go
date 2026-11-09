package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-wizard.md FR-099 - a spell is on the Wizard list if and only if its classes contain the wizard, the test classSpellList applies; NewWizardSpellFacts takes the id, level, school, Ritual tag,
// classes and casting times from the content Spell and the damage flag from the caller.
// Path: per case (happy / unhappy: rule interaction).
func TestWizardSpellFacts(t *testing.T) {
	t.Parallel()
	src := wzSource(t)
	for id, c := range map[string]struct {
		wizard, ritual, action bool
		level                  int
	}{"magic_missile": {true, false, true, 1}, "detect_magic": {true, true, true, 1}, "fireball": {true, false, true, 3}, "fire_bolt": {true, false, true, 0}, "druidcraft": {false, false, true, 0}} {
		res := must(src.Lookup(context.Background(), content.NewKey(content.KindSpell, content.MustID(id))))
		spell, ok := res.(content.LookupFound).Entry.(content.Spell)
		if !ok {
			t.Fatalf("%s is not a Spell", id)
		}
		f := character.NewWizardSpellFacts(spell, true)
		if f.ID != spell.ID() || int(f.Level) != c.level || f.OnWizardList() != c.wizard || f.IsCantrip() != (c.level == 0) || !f.DealsDamage || f.School != spell.School() {
			t.Errorf("%s facts = %+v", id, f)
		}
		if f.Ritual != spell.Ritual() || f.HasActionCastingTime() != c.action {
			t.Errorf("%s Ritual %v, action %v; want %v", id, f.Ritual, f.HasActionCastingTime(), c.action)
		}
	}
	// A spell with the Magic action among several modes, and a spell with only a slower one.
	if !wzFact("x", 1).HasActionCastingTime() || wzFact("x", 1, wzMinutes).HasActionCastingTime() || (character.WizardSpellFacts{}).HasActionCastingTime() {
		t.Error("HasActionCastingTime is wrong")
	}
}

// Requirement: doc/specs/class-wizard.md FR-099 - NewWizardSpellFacts keeps every casting time of a spell with several modes, in order.
// Path: happy (several modes).
func TestWizardSpellFactsModes(t *testing.T) {
	t.Parallel()
	res := must(wzSource(t).Lookup(context.Background(), content.NewKey(content.KindSpell, content.MustID("fire_bolt"))))
	base := res.(content.LookupFound).Entry.(content.Spell)
	min10 := must(content.NewCastingTimeMinutes(10))
	slow := must(content.NewCastingTimeMode("slow", min10))
	fast := must(content.NewCastingTimeMode("fast", content.NewCastingTimeMagicAction()))
	modes := must(content.NewSpellCastingTimesModes([]content.CastingTimeMode{slow, fast}))
	spell := must(content.NewSpell(base.ID(), base.Name(), base.Source(), content.SpellFields{
		Level: base.Level(), School: base.School(), Classes: base.Classes(), CastingTimes: modes, Ritual: base.Ritual(), Range: base.Range(),
		Components: base.Components(), Duration: base.Duration(), Description: base.Description(), HigherLevel: base.HigherLevel(),
		CantripUpgrade: base.CantripUpgrade(), Effects: base.Effects(),
	}))
	f := character.NewWizardSpellFacts(spell, false)
	if len(f.CastingTimes) != 2 || f.CastingTimes[0] != content.CastingTime(min10) || f.CastingTimes[1] != content.CastingTime(content.NewCastingTimeMagicAction()) || !f.HasActionCastingTime() || f.DealsDamage {
		t.Errorf("casting times = %+v", f.CastingTimes)
	}
}
