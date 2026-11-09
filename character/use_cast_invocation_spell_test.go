package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-warlock.md FR-059 to FR-063, FR-065, FR-089, FR-165 - the use case returns the spell of a known spell-granting invocation to cast with CastModeFeatureGranted and no slot: Fiendish Vigor gives False Life with the maximum die, Gift of the Depths gives Water Breathing once (spent in the new Character, the second cast is refused until a Long Rest), One with Shadows needs Dim Light or Darkness; the command's Character is not altered.
// Path: per case (happy; unhappy: state error).
func TestCastInvocationSpellGrantsTheSpell(t *testing.T) {
	t.Parallel()
	uc := must(character.NewCastInvocationSpell(warSource(t)))
	c := warChar(t, 5, 16, "", warKnows("fiendish_vigor", "gift_of_the_depths", "one_with_shadows"))
	run := func(c character.Character, option string, light content.LightLevel) character.CastInvocationSpellResult {
		res, err := uc.Run(t.Context(), character.CastInvocationSpellCommand{Character: c, Option: content.FeatureOptionID(option), Light: light})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	vigor, ok := run(c, "fiendish_vigor", content.LightBright).(character.CastInvocationSpellApplied)
	if g := vigor.Grant(); !ok || g.Spell != "false_life" || g.Mode != character.CastModeFeatureGranted || g.DieRule != character.SpellDieMaximum {
		t.Fatalf("Fiendish Vigor = %#v", vigor)
	}
	depths, ok := run(c, "gift_of_the_depths", content.LightBright).(character.CastInvocationSpellApplied)
	if !ok || depths.Grant().Spell != "water_breathing" || !warStateOf(t, depths.Character()).GiftOfTheDepthsCast() || warStateOf(t, c).GiftOfTheDepthsCast() {
		t.Fatalf("Gift of the Depths = %#v", depths)
	}
	if ev, isEv := depths.Events()[0].(character.InvocationSpellCastWithoutSlot); !isEv || ev.Option() != "gift_of_the_depths" || ev.Spell() != "water_breathing" {
		t.Errorf("events = %v", depths.Events())
	}
	depths.Events()[0] = nil
	if depths.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	for name, c := range map[string]struct {
		char   character.Character
		option string
		light  content.LightLevel
		want   character.WarlockRejectionKind
	}{
		"second Gift of the Depths": {depths.Character(), "gift_of_the_depths", content.LightBright, character.WarlockAlreadyCast},
		"bright light":              {c, "one_with_shadows", content.LightBright, character.WarlockLightLevelWrong},
		"not a spell grant":         {warChar(t, 5, 16, "", warKnows("devils_sight")), "devils_sight", content.LightBright, character.WarlockNotASpellGrant},
	} {
		if rej, ok := run(c.char, c.option, c.light).(character.CastInvocationSpellRejected); !ok || rej.Rejection.Kind != c.want {
			t.Errorf("%s = %#v, want %v", name, rej, c.want)
		}
	}
	if dim, ok := run(c, "one_with_shadows", content.LightDim).(character.CastInvocationSpellApplied); !ok || dim.Grant().Spell != "invisibility" {
		t.Errorf("Dim Light = %#v", dim)
	}
}

// Requirement: doc/specs/class-warlock.md FR-042, FR-141, FR-142 - a source without the warlock class gives CastInvocationSpellContentMissing naming the class; a source that fails or answers with another entry is a *srderr.PortError recorded once.
// Path: per case (unhappy: missing content, collaborator failure).
func TestCastInvocationSpellContentFailures(t *testing.T) {
	t.Parallel()
	c := warChar(t, 5, 16, "", warKnows("fiendish_vigor"))
	warContentFailures(t, character.SpanCastInvocationSpell, "invocation_spell", []content.MissingContent{character.WarlockClassID.Missing()},
		func(src character.ContentSource, opts ...character.Option) (any, error) {
			return must(character.NewCastInvocationSpell(src, opts...)).Run(t.Context(), character.CastInvocationSpellCommand{Character: c, Option: "fiendish_vigor", Light: content.LightBright})
		}, func(res any) ([]content.MissingContent, bool) {
			m, ok := res.(character.CastInvocationSpellContentMissing)
			return m.Missing(), ok
		})
	if !srderrIsInvalid(errOf(character.NewCastInvocationSpell(nil))) {
		t.Error("a nil source was accepted")
	}
	if _, err := (character.CastInvocationSpell{}).Run(t.Context(), character.CastInvocationSpellCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
}
