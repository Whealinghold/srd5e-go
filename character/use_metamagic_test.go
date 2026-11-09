package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-sorcerer.md FR-059 to FR-061, FR-099, FR-110, FR-123 - UseMetamagic spends the option's cost from the class definition on the new Character, reports what the option did and emits a MetamagicUsed event, leaving
// its input unchanged; an option the definition does not list, one the Sorcerer does not know, too few points and a spell the option does not suit are results carrying the rejection.
// Path: per case (happy / unhappy: state, boundary).
func TestUseMetamagicRun(t *testing.T) {
	t.Parallel()
	uc := must(character.NewUseMetamagic(sorSource(t)))
	known := []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicHeightened}
	c := sorChar(t, 5, 16, "", sorWith(sorProfile(5, ""), 5, known, nil))
	do := func(c character.Character, option string, facts character.MetamagicSpellFacts, params character.MetamagicParameters) character.UseMetamagicResult {
		return must(uc.Run(t.Context(), character.UseMetamagicCommand{Character: c, Option: content.FeatureOptionID(option), Facts: facts, Parameters: params}))
	}
	res := do(c, "heightened_spell", sorFacts(nil), character.MetamagicParameters{}).(character.UseMetamagicApplied)
	if got := sorStateOf(t, res.Character()).SorceryPoints(); got != 3 || sorStateOf(t, c).SorceryPoints() != 5 {
		t.Errorf("points = %d (input %d), want 3 (input 5): Heightened Spell costs 2 in the class definition", got, sorStateOf(t, c).SorceryPoints())
	}
	if a := res.Application(); a.Effect != character.MetamagicHeightened || a.Cost != 2 || !a.DisadvantageOnSave {
		t.Errorf("application = %+v", a)
	}
	if ev, ok := res.Events()[0].(character.MetamagicUsed); len(res.Events()) != 1 || !ok || ev.CharacterID() != c.ID() || ev.Cost() != 2 {
		t.Errorf("events = %#v", res.Events())
	}
	res.Events()[0] = nil
	if res.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
	for _, rc := range []struct {
		name   string
		option string
		char   character.Character
		facts  character.MetamagicSpellFacts
		params character.MetamagicParameters
		kind   character.SorcererRejectionKind
		why    character.MetamagicIneligibility
	}{
		{"an option the definition does not list", "lucky_spell", c, sorFacts(nil), character.MetamagicParameters{}, character.SorcererMetamagicOptionNotKnown, 0},
		{"an option the Sorcerer does not know", "twinned_spell", c, sorFacts(nil), character.MetamagicParameters{}, character.SorcererMetamagicOptionNotKnown, 0},
		{"too few points", "heightened_spell", sorChar(t, 5, 16, "", sorWith(sorProfile(5, ""), 1, known, nil)), sorFacts(nil), character.MetamagicParameters{}, character.SorcererSorceryPointsInsufficient, 0},
		{"a spell that forces no saving throw", "heightened_spell", c, sorFacts(func(f *character.MetamagicSpellFacts) { f.ForcesSavingThrow = false }), character.MetamagicParameters{}, character.SorcererMetamagicNotEligible, character.MetamagicNoSavingThrow},
		{"a count above the cap", "careful_spell", c, sorFacts(nil), character.MetamagicParameters{Count: 4}, character.SorcererMetamagicNotEligible, character.MetamagicCountOutOfRange},
	} {
		r, ok := do(rc.char, rc.option, rc.facts, rc.params).(character.UseMetamagicRejected)
		if !ok || r.Rejection.Kind != rc.kind || r.Rejection.Ineligibility != rc.why {
			t.Errorf("%s = %#v, want %v %v", rc.name, r, rc.kind, rc.why)
		}
	}
}
