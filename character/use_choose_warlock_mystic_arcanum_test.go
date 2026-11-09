package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-warlock.md FR-117, FR-118, FR-120 - the use case loads the spell and stores it as the arcanum of its level: Power Word Kill at level 17 becomes the level 9 arcanum with a MysticArcanumChosen event, a second choice of the same level is a replacement (Replaced) and a second replacement in one level is refused; the command's Character is not altered.
// Path: per case (happy; unhappy: boundary).
func TestChooseWarlockMysticArcanumStoresTheSpell(t *testing.T) {
	t.Parallel()
	uc := must(character.NewChooseWarlockMysticArcanum(warSource(t)))
	c := warChar(t, 17, 16, "", character.WarlockState{})
	res, err := uc.Run(t.Context(), character.ChooseWarlockMysticArcanumCommand{Character: c, Spell: "power_word_kill"})
	got, ok := res.(character.ChooseWarlockMysticArcanumApplied)
	if err != nil || !ok {
		t.Fatalf("= %#v, %v", res, err)
	}
	if spell, held := warStateOf(t, got.Character()).Arcanum(9); !held || spell != "power_word_kill" {
		t.Errorf("arcanum = %v, %v", spell, held)
	}
	if _, held := warStateOf(t, c).Arcanum(9); held {
		t.Error("the command's Character was altered")
	}
	if ev, isEv := got.Events()[0].(character.MysticArcanumChosen); !isEv || ev.Replaced() || ev.Spell() != "power_word_kill" || ev.SpellLevel() != 9 {
		t.Errorf("events = %v", got.Events())
	}
	got.Events()[0] = nil
	if got.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	res, _ = uc.Run(t.Context(), character.ChooseWarlockMysticArcanumCommand{Character: got.Character(), Spell: "power_word_kill"})
	again, ok := res.(character.ChooseWarlockMysticArcanumApplied)
	if !ok || !again.Events()[0].(character.MysticArcanumChosen).Replaced() {
		t.Fatalf("replacement = %#v", res)
	}
	res, _ = uc.Run(t.Context(), character.ChooseWarlockMysticArcanumCommand{Character: got.Character(), Spell: "power_word_kill", Replacements: 1})
	if rej, ok := res.(character.ChooseWarlockMysticArcanumRejected); !ok || rej.Rejection.Kind != character.WarlockArcanumTooManyReplacements {
		t.Errorf("second replacement = %#v", res)
	}
}

// Requirement: doc/specs/class-warlock.md FR-042, FR-141, FR-142 - a spell the source lacks is a ChooseWarlockMysticArcanumContentMissing naming the spell (error status on the span, rejection content_missing, no log, no error.type); a source that fails, or answers with another entry, is a *srderr.PortError recorded once; an invalid spell id is an error recorded on the span only and a nil source is refused by the constructor.
// Path: per case (unhappy: missing content, collaborator failure, invalid input).
func TestChooseWarlockMysticArcanumContentFailures(t *testing.T) {
	t.Parallel()
	c := warChar(t, 17, 16, "", character.WarlockState{})
	cmd := character.ChooseWarlockMysticArcanumCommand{Character: c, Spell: "power_word_kill"}
	rec, opts := warTelemetry()
	res, err := must(character.NewChooseWarlockMysticArcanum(contenttest.NewFakeSource(), opts...)).Run(t.Context(), cmd)
	missing, ok := res.(character.ChooseWarlockMysticArcanumContentMissing)
	if err != nil || !ok || len(missing.Missing()) != 1 || missing.Missing()[0] != content.SpellID("power_word_kill").Missing() {
		t.Fatalf("missing = %#v, %v", res, err)
	}
	missing.Missing()[0] = content.MissingContent{}
	if missing.Missing()[0] == (content.MissingContent{}) {
		t.Error("Missing must return a copy")
	}
	warCheck(t, rec, warCall(character.SpanChooseWarlockMysticArcanum, "mystic_arcanum", "content_missing", telemetry.StatusError, ""))
	for name, src := range map[string]character.ContentSource{"an error": contenttest.NewFakeSource().FailLookup(errBoom), "wrong entry": wrongEntrySource{entry: warSource(t)}} {
		rec, opts = warTelemetry()
		res, err = must(character.NewChooseWarlockMysticArcanum(src, opts...)).Run(t.Context(), cmd)
		var pe *srderr.PortError
		if res != nil || !asPortError(err, &pe) || pe.Port != content.Port {
			t.Fatalf("%s: = %v, %v; want a PortError for content", name, res, err)
		}
		warCheck(t, rec, warWant{span: character.SpanChooseWarlockMysticArcanum, feature: "mystic_arcanum", outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err), loaded: true})
	}
	rec, opts = warTelemetry()
	if res, err := must(character.NewChooseWarlockMysticArcanum(warSource(t), opts...)).Run(t.Context(), character.ChooseWarlockMysticArcanumCommand{Character: c}); res != nil || !srderrIsInvalid(err) {
		t.Errorf("invalid spell id = %v, %v", res, err)
	}
	if len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Error("an invalid command must not count as a use")
	}
	if !srderrIsInvalid(errOf(character.NewChooseWarlockMysticArcanum(nil))) {
		t.Error("a nil source was accepted")
	}
	if _, err := (character.ChooseWarlockMysticArcanum{}).Run(t.Context(), cmd); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
}

// warCall is the warWant of a use case that stopped before loading content or whose load span is not asserted.
func warCall(span, feature, rejection string, status telemetry.Status, errorType string) warWant {
	return warWant{span: span, feature: feature, outcome: "failure", rejection: rejection, status: status, errorType: errorType, loaded: true}
}
