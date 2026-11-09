package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/feats-behaviour.md FR-052, FR-071, FR-073, FR-074, Table U - RenewFeatUses uses no port, returns the new TakenFeats and one FeatUsesRenewed, or the same state and no events when nothing is expended, agrees with the no-op bundle, and records changed/success once.
// Path: per case.
func TestRenewFeatUses(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	feats := fateAndInitiate(t)
	spent := expend(t, feats, "boon_of_fate").(character.FeatsChanged).Feats()
	for _, tc := range []struct {
		name   string
		feats  character.TakenFeats
		events int
	}{{"an expended use", spent, 1}, {"nothing expended", feats, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.RenewFeatUsesCommand{Character: c, Feats: tc.feats, Renewal: character.RenewOnShortRestFinished}
			rec := telemetrytest.NewRecorder()
			res, err := character.NewRenewFeatUses(character.WithTelemetry(rec.Telemetry())).Run(context.Background(), cmd)
			plain, err2 := character.NewRenewFeatUses().Run(context.Background(), cmd)
			ok, isOK := res.(character.RenewFeatUsesSucceeded)
			if err != nil || err2 != nil || !isOK || !reflect.DeepEqual(res, plain) || len(ok.Events()) != tc.events {
				t.Fatalf("= %#v, %v", res, err)
			}
			if tc.events == 0 && !ok.Feats().Equal(tc.feats) {
				t.Error("nothing renewed must leave the feats as they were")
			}
			if tc.events == 1 && ok.Feats().UseStateOf("boon_of_fate") != (character.FeatUseIsAvailable{}) {
				t.Error("the use must be renewed")
			}
			checkOneCall(t, rec, character.SpanRenewFeatUses, "renew_feat_uses", telemetry.StatusUnset, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"})
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-059, FR-071 - RenewFeatUses refuses a zero character, an invalid renewal and the zero use case before any signal, and a cancelled context unwrapped.
// Path: unhappy (invalid input).
func TestRenewFeatUsesInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	uc := character.NewRenewFeatUses(character.WithTelemetry(rec.Telemetry()))
	good := character.RenewFeatUsesCommand{Character: charAt(t, 5), Renewal: character.RenewOnLongRestFinished}
	for name, cmd := range map[string]character.RenewFeatUsesCommand{"zero character": {Renewal: good.Renewal}, "invalid renewal": {Character: good.Character}} {
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.RenewFeatUses{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, good); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a refused call emitted a signal")
	}
}
