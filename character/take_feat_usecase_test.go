package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// featSource holds the synthetic feats, the Magic Initiate spells and two tools.
func featSource(t *testing.T, extra ...content.Entry) *contenttest.FakeSource {
	t.Helper()
	entries := []content.Entry{alertFeat(), savageFeat(), initiateFeat(), skilledFeat(), asiFeat(), grapplerFeat(), thievesTools(), herbalismKit()}
	for _, s := range miSpells(t) {
		entries = append(entries, s)
	}
	return contenttest.NewFakeSource(append(entries, extra...)...)
}

func takeUseCase(t *testing.T, src character.ContentSource, opts ...character.Option) character.TakeFeat {
	t.Helper()
	return must(character.NewTakeFeat(src, opts...))
}

func recordedTake(t *testing.T, src character.ContentSource) (character.TakeFeat, *telemetrytest.Recorder) {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return takeUseCase(t, src, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)), rec
}

func customOrigin() character.FeatGrant {
	return grantOf(character.CustomBackgroundFeat{}, character.FeatFromCategory{Category: content.FeatOrigin})
}

func takeCmd(c character.Character, feat string, grant character.FeatGrant, choices character.FeatChoices) character.TakeFeatCommand {
	return character.TakeFeatCommand{Character: c, Feat: content.FeatID(feat), Grant: grant, Choices: choices}
}

// wantTakeSpans checks the take_feat span and its load_content child, and the counter and duration of one call.
func wantTakeSpans(t *testing.T, rec *telemetrytest.Recorder, status telemetry.Status, loadStatus telemetry.Status, spanAttrs, counterAttrs map[string]any) {
	t.Helper()
	spans := rec.Tracer.Spans()
	if len(spans) != 2 || spans[0].Name != character.SpanTakeFeat || spans[1].Name != character.SpanLoadContent || spans[1].ParentID != spans[0].ID || !spans[0].Ended || !spans[1].Ended {
		t.Fatalf("spans = %+v", spans)
	}
	if spans[0].Status != status || spans[1].Status != loadStatus {
		t.Errorf("statuses = %v %v, want %v %v", spans[0].Status, spans[1].Status, status, loadStatus)
	}
	sameAttrs(t, "span", spans[0].Attrs, spanAttrs)
	sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, counterAttrs)
	d := onlyOne(t, rec, character.MetricDuration)
	sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "take_feat"})
	if d.Value != 0.25 {
		t.Errorf("duration = %v, want 0.25", d.Value)
	}
	noLogsOrViolations(t, rec)
}

// Requirement: doc/specs/feats-behaviour.md FR-066, FR-071, FR-073, FR-075 (SRD 5.2.1, "Parts of a Feat", p. 87) - TakeFeat looks up the feat and every spell and tool of the choices and returns the new Character, TakenFeats, grants and spells; the command is unaltered; the no-op and a recording bundle give the same result; one span with a load_content child, one counter increment and one duration, taken/success with the category and the source (a custom background is a background), no log.
// Path: per case.
func TestTakeFeatSucceeds(t *testing.T) {
	t.Parallel()
	picks := skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledTool{Tool: "thieves_tools"}, character.SkilledTool{Tool: "herbalism_kit"})
	for _, c := range []struct {
		name     string
		feat     string
		grant    character.FeatGrant
		choices  character.FeatChoices
		category string
		source   string
		lookups  []content.Key
	}{
		{"Alert", "alert", customOrigin(), character.NoFeatChoices{}, "origin", "background", []content.Key{content.FeatID("alert").Key()}},
		{"Magic Initiate", "magic_initiate", grantOf(character.SpeciesTraitFeat{Species: "human", Trait: "human_versatile"}, character.FeatFromCategory{Category: content.FeatOrigin}), clericInitiate(), "origin", "species_trait",
			[]content.Key{content.FeatID("magic_initiate").Key(), content.SpellID("bless").Key(), content.SpellID("guidance").Key(), content.SpellID("light").Key()}},
		{"Skilled with tools", "skilled", grantOf(character.EldritchInvocationFeat{Option: "lessons_of_the_first_ones"}, character.FeatFromCategory{Category: content.FeatOrigin}), picks, "origin", "eldritch_invocation",
			[]content.Key{content.FeatID("skilled").Key(), content.ItemID("thieves_tools").Key(), content.ItemID("herbalism_kit").Key()}},
		{"Ability Score Improvement", "ability_score_improvement", anyGrant(), must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0)), "general", "class_feature", []content.Key{content.FeatID("ability_score_improvement").Key()}},
		{"a Soldier's feat", "savage_attacker", grantOf(soldierSrc, character.SpecifiedFeat{Feat: "savage_attacker", Required: character.NoRequiredSpellList{}}), character.NoFeatChoices{}, "origin", "background", []content.Key{content.FeatID("savage_attacker").Key()}},
		{"a bonus feat", "alert", grantOf(character.BonusFeat{}, character.AnyFeat{}), character.NoFeatChoices{}, "origin", "bonus_feat", []content.Key{content.FeatID("alert").Key()}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch := charAt(t, 8)
			cmd := takeCmd(ch, c.feat, c.grant, c.choices)
			src := featSource(t)
			uc, rec := recordedTake(t, src)
			res, err := uc.Run(context.Background(), cmd)
			plain, err2 := takeUseCase(t, featSource(t)).Run(context.Background(), cmd)
			ok, isOK := res.(character.TakeFeatSucceeded)
			if err != nil || err2 != nil || !isOK || !reflect.DeepEqual(res, plain) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			if ok.Character().ID() != ch.ID() || (c.feat == "skilled") == ok.ProficiencyGrants().IsZero() || (c.feat == "magic_initiate") == ok.GrantedSpells().IsZero() {
				t.Errorf("grants = %+v, spells = %+v", ok.ProficiencyGrants(), ok.GrantedSpells())
			}
			if !ok.Feats().Holds(content.FeatID(c.feat)) || ok.Feats().Len() != 1 || !ch.Equal(charAt(t, 8)) || cmd.Feats.Len() != 0 {
				t.Error("the new feats must hold the feat and the command must be unaltered")
			}
			if ev := ok.Events(); len(ev) == 0 || ev[0].(character.FeatTaken).Feat() != content.FeatID(c.feat) {
				t.Errorf("events = %#v", ev)
			}
			got := lookupKeys(src)
			slices.SortFunc(got, func(a, b content.Key) int { return compareKeys(a, b) })
			want := slices.Clone(c.lookups)
			slices.SortFunc(want, func(a, b content.Key) int { return compareKeys(a, b) })
			if !slices.Equal(got, want) {
				t.Errorf("lookups = %v, want %v", got, want)
			}
			attrs := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "taken", character.AttrFeatCategory: c.category, character.AttrFeatSource: c.source}
			counter := map[string]any{character.AttrOperation: "take_feat"}
			for k, v := range attrs {
				counter[k] = v
			}
			wantTakeSpans(t, rec, telemetry.StatusUnset, telemetry.StatusUnset, attrs, counter)
			sameAttrs(t, "load_content", rec.Tracer.Spans()[1].Attrs, outcome("success"))
		})
	}
}

func compareKeys(a, b content.Key) int {
	if a.Kind() != b.Kind() {
		return int(a.Kind()) - int(b.Kind())
	}
	switch {
	case a.ID().String() < b.ID().String():
		return -1
	case a.ID().String() > b.ID().String():
		return 1
	}
	return 0
}

// Requirement: doc/specs/feats-behaviour.md FR-069, FR-074 - a domain rejection is a TakeFeatRejected result recorded as failure/rejected with the rejection and no error status, and the Character and TakenFeats stand.
// Path: per case.
func TestTakeFeatRejectedIsRecordedOnce(t *testing.T) {
	t.Parallel()
	uc, rec := recordedTake(t, featSource(t))
	cmd := takeCmd(charAt(t, 8), "alert", grantOf(soldierSrc, character.SpecifiedFeat{Feat: "savage_attacker", Required: character.NoRequiredSpellList{}}), character.NoFeatChoices{})
	res, err := uc.Run(context.Background(), cmd)
	rj, ok := res.(character.TakeFeatRejected)
	if err != nil || !ok || rj.Rejection != (character.FeatOutsideGrant{Feat: "alert"}) {
		t.Fatalf("= %#v, %v", res, err)
	}
	attrs := map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "feat_outside_grant", character.AttrFeatCategory: "origin", character.AttrFeatSource: "background"}
	counter := map[string]any{character.AttrOperation: "take_feat"}
	for k, v := range attrs {
		counter[k] = v
	}
	wantTakeSpans(t, rec, telemetry.StatusUnset, telemetry.StatusUnset, attrs, counter)
}

// Requirement: doc/specs/feats-behaviour.md FR-067, FR-074 - when the feat, a spell or a tool is not found TakeFeat returns every missing entry once, ordered by kind then id, records failure/content_missing with error status, one counter increment and no error.type, omits the category when the feat is missing, and never panics; an equipment entry that is not a Tool counts as missing.
// Path: per case.
func TestTakeFeatContentMissing(t *testing.T) {
	t.Parallel()
	picks := skilledOf(character.SkilledSkill{Skill: content.SkillInsight}, character.SkilledTool{Tool: "thieves_tools"}, character.SkilledTool{Tool: "leather_armor"})
	for _, c := range []struct {
		name         string
		src          *contenttest.FakeSource
		cmd          character.TakeFeatCommand
		want         []content.MissingContent
		withCategory bool
	}{
		{"the feat", contenttest.NewFakeSource(), takeCmd(charAt(t, 8), "alert", customOrigin(), character.NoFeatChoices{}), []content.MissingContent{content.FeatID("alert").Missing()}, false},
		{"a spell, repeated, and the feat", contenttest.NewFakeSource(), takeCmd(charAt(t, 8), "magic_initiate", customOrigin(), miOf("cleric", ability.Wisdom, []string{"light", "light"}, "bless")),
			[]content.MissingContent{content.FeatID("magic_initiate").Missing(), content.SpellID("bless").Missing(), content.SpellID("light").Missing()}, false},
		{"a tool, and a piece of equipment that is not a Tool", featSource(t, leatherArmor(t)), takeCmd(charAt(t, 8), "skilled", customOrigin(), picks), []content.MissingContent{content.ItemID("leather_armor").Missing()}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, rec := recordedTake(t, c.src)
			res, err := uc.Run(context.Background(), c.cmd)
			m, ok := res.(character.TakeFeatContentMissing)
			if err != nil || !ok || !slices.Equal(m.Missing(), c.want) {
				t.Fatalf("= %#v, %v, want %v", res, err, c.want)
			}
			m.Missing()[0] = content.MissingContent{}
			if m.Missing()[0] == (content.MissingContent{}) {
				t.Error("Missing must return a copy")
			}
			attrs := map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing", character.AttrFeatSource: "background"}
			if c.withCategory {
				attrs[character.AttrFeatCategory] = "origin"
			}
			counter := map[string]any{character.AttrOperation: "take_feat"}
			for k, v := range attrs {
				counter[k] = v
			}
			wantTakeSpans(t, rec, telemetry.StatusError, telemetry.StatusError, attrs, counter)
			sameAttrs(t, "load_content", rec.Tracer.Spans()[1].Attrs, outcome("failure"))
		})
	}
	if (character.TakeFeatContentMissing{}).Missing() != nil {
		t.Error("the zero result names nothing")
	}
}

// answering is a content source that answers every lookup with entry.
type answering struct{ entry content.Entry }

func (a answering) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	return content.LookupFound{Entry: a.entry}, nil
}

// Requirement: doc/specs/feats-behaviour.md FR-068, FR-074 - a content port failure, panic or broken contract is a *srderr.PortError (content_error) wrapped once, returned with a nil result, recorded once with error status, one counter increment with error.type, no result attribute and no log.
// Path: per case.
func TestTakeFeatPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("down")
	cmd := takeCmd(charAt(t, 8), "alert", customOrigin(), character.NoFeatChoices{})
	for _, c := range []struct {
		name string
		src  character.ContentSource
		cmd  character.TakeFeatCommand
	}{
		{"the source fails", contenttest.NewFakeSource().FailLookup(cause), cmd},
		{"a feat key answered with a spell", answering{entry: miSpells(t)["light"]}, cmd},
		{"a spell key answered with a feat", answering{entry: alertFeat()}, takeCmd(charAt(t, 8), "magic_initiate", customOrigin(), clericInitiate())},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, rec := recordedTake(t, c.src)
			res, err := uc.Run(context.Background(), c.cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || srderr.ErrorType(err) != "content_error" {
				t.Fatalf("= %v, %v", res, err)
			}
			if c.name == "the source fails" && !errors.Is(err, cause) {
				t.Error("the cause is lost")
			}
			spans := rec.Tracer.Spans()
			if spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Errorf("span = %+v", spans[0])
			}
			m := onlyOne(t, rec, character.MetricOperations)
			g := map[string]any{}
			for _, a := range m.Attrs {
				g[a.Key()] = a.Value()
			}
			if g[telemetry.AttrOutcome] != "failure" || g[telemetry.AttrErrorType] != "content_error" || g[character.AttrOperation] != "take_feat" || g[character.AttrResult] != nil {
				t.Errorf("operations attrs = %v", g)
			}
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-059, FR-071 - invalid commands, the zero use case and a cancelled context are refused before any lookup or signal.
// Path: unhappy (invalid input).
func TestTakeFeatInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	src := featSource(t)
	uc := takeUseCase(t, src, character.WithTelemetry(rec.Telemetry()))
	good := takeCmd(charAt(t, 8), "alert", customOrigin(), character.NoFeatChoices{})
	bad := map[string]character.TakeFeatCommand{
		"zero character": {Feat: "alert", Grant: customOrigin(), Choices: character.NoFeatChoices{}},
		"invalid feat":   {Character: good.Character, Feat: "", Grant: customOrigin(), Choices: character.NoFeatChoices{}},
		"zero grant":     {Character: good.Character, Feat: "alert", Choices: character.NoFeatChoices{}},
		"nil choices":    {Character: good.Character, Feat: "alert", Grant: customOrigin()},
	}
	for name, cmd := range bad {
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.TakeFeat{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	if _, err := character.NewTakeFeat(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil source: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, good); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(lookupKeys(src)) != 0 {
		t.Error("a refused call must emit no signal and look nothing up")
	}
}
