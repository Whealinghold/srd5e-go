package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/feats-behaviour.md FR-050, FR-051, FR-071, FR-073, FR-074, Table U - ExpendFeatUse uses no port, returns the new TakenFeats and FeatUseExpended or the rejection, leaves the command unaltered, agrees with the no-op bundle, and records one span and one counter increment (changed/success, rejected/failure with the rejection and no error status).
// Path: per case.
func TestExpendFeatUse(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20)
	feats := heldFeats(t, boonFeat("boon_of_fate"), singleOf(ability.Wisdom), alertFeat(), character.NoFeatChoices{})
	expended := expend(t, feats, "boon_of_fate").(character.FeatsChanged).Feats()
	for _, tc := range []struct {
		name    string
		feats   character.TakenFeats
		feat    string
		result  string
		outcome string
		reject  string
	}{
		{"an available use", feats, "boon_of_fate", "changed", "success", ""},
		{"an expended use", expended, "boon_of_fate", "rejected", "failure", "feat_use_already_expended"},
		{"a feat not held", feats, "grappler", "rejected", "failure", "feat_not_held"},
		{"a feat with no limited use", feats, "alert", "rejected", "failure", "feat_has_no_limited_use"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ExpendFeatUseCommand{Character: c, Feats: tc.feats, Feat: content.FeatID(tc.feat)}
			snapshot := tc.feats
			rec := telemetrytest.NewRecorder()
			res, err := character.NewExpendFeatUse(character.WithTelemetry(rec.Telemetry())).Run(context.Background(), cmd)
			plain, err2 := character.NewExpendFeatUse().Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) || !cmd.Feats.Equal(snapshot) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			attrs := map[string]any{telemetry.AttrOutcome: tc.outcome, character.AttrResult: tc.result}
			switch r := res.(type) {
			case character.ExpendFeatUseSucceeded:
				if tc.result != "changed" || r.Event().Feat() != "boon_of_fate" || r.Feats().UseStateOf("boon_of_fate") != (character.FeatUseIsExpended{}) {
					t.Errorf("= %#v", r)
				}
			case character.ExpendFeatUseRejected:
				if tc.result != "rejected" || r.Rejection.String() != tc.reject {
					t.Errorf("= %#v", r)
				}
				attrs[character.AttrRejection] = tc.reject
			}
			checkOneCall(t, rec, character.SpanExpendFeatUse, "expend_feat_use", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-059, FR-071 - ExpendFeatUse refuses a zero character, an invalid feat id and the zero use case before any signal, and a cancelled context unwrapped.
// Path: unhappy (invalid input).
func TestExpendFeatUseInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	uc := character.NewExpendFeatUse(character.WithTelemetry(rec.Telemetry()))
	good := character.ExpendFeatUseCommand{Character: charAt(t, 5), Feat: "alert"}
	for name, cmd := range map[string]character.ExpendFeatUseCommand{"zero character": {Feat: "alert"}, "invalid feat": {Character: good.Character}} {
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.ExpendFeatUse{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
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
