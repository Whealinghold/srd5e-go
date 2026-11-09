package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// brbSource holds every weapon fixture and one armor.
func brbSource(t *testing.T) *contenttest.FakeSource {
	t.Helper()
	entries := []content.Entry{leatherArmor(t)}
	for _, w := range weaponFixtures(t) {
		entries = append(entries, w)
	}
	return contenttest.NewFakeSource(entries...)
}

func brbApplyUseCase(t *testing.T, src character.ContentSource, opts ...character.Option) character.ApplyBarbarianOptionChoice {
	t.Helper()
	return must(character.NewApplyBarbarianOptionChoice(brbProgression(t), src, opts...))
}

func brbApply(t *testing.T, src character.ContentSource, c character.Character, choice character.BarbarianOptionChoice) character.ApplyBarbarianOptionChoiceResult {
	t.Helper()
	res, err := brbApplyUseCase(t, src).Run(context.Background(), character.ApplyBarbarianOptionChoiceCommand{Character: c, Choice: choice})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

// Requirement: doc/specs/class-barbarian.md FR-041, FR-043, FR-103, FR-046 - the Weapon Mastery choices through the content source: two kinds at level 1 (resolved as weapons from the source), one more kind at level 4 and one swap
// after a Long Rest of this character, each returning the new Character and the events.
// Path: happy.
func TestApplyBarbarianOptionChoiceMasteries(t *testing.T) {
	t.Parallel()
	src := brbSource(t)
	chosen, ok := brbApply(t, src, brbBarbarian(t, 1), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}).(character.BarbarianOptionApplied)
	if !ok || len(chosen.Events()) != 1 {
		t.Fatalf("masteries: %#v", chosen)
	}
	access := must(character.WeaponMasteryAccessFor(chosen.Character().FeatureStates())).(character.MasteryFromChoice)
	if !slices.Equal(access.Choice.Kinds(), ids("greataxe", "handaxe")) {
		t.Errorf("choice = %v", access.Choice.Kinds())
	}
	levelFour := must(character.WithFeatureState(brbBarbarian(t, 4), character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(access.Choice)))
	more, ok := brbApply(t, src, levelFour, character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe", "mace")}).(character.BarbarianOptionApplied)
	if !ok || !slices.Equal(must(character.WeaponMasteryAccessFor(more.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(), ids("greataxe", "handaxe", "mace")) {
		t.Fatalf("level 4: %#v", more)
	}
	swapped, ok := brbApply(t, src, chosen.Character(), character.BarbarianSwapWeaponMastery{Remove: "handaxe", Add: "mace", RestApplied: fgtRest(t, chosen.Character().ID(), character.LongRest)}).(character.BarbarianOptionApplied)
	if !ok || !slices.Equal(must(character.WeaponMasteryAccessFor(swapped.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(), ids("greataxe", "mace")) {
		t.Fatalf("swap: %#v", swapped)
	}
	swapped.Events()[0] = nil
	if swapped.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-085, FR-103 - a weapon the source cannot find is BarbarianOptionContentMissing carrying every missing entry (once each, ordered by id), with no rule reason even when a rule would refuse;
// a found item that is not a weapon is not missing but a not-a-Simple-or-Martial-Melee-weapon problem; the Character is unchanged.
// Path: per case (unhappy: missing content).
func TestApplyBarbarianOptionChoiceContentMissing(t *testing.T) {
	t.Parallel()
	src := brbSource(t)
	for _, c := range []struct {
		name    string
		choice  character.BarbarianOptionChoice
		missing []content.MissingContent
	}{
		{"missing weapons, once each and in order", character.BarbarianChooseWeaponMasteries{Kinds: ids("zweihander", "mace", "aklys", "zweihander")}, []content.MissingContent{content.ItemID("aklys").Missing(), content.ItemID("zweihander").Missing()}},
		{"a missing added kind of a swap", character.BarbarianSwapWeaponMastery{Remove: "mace", Add: "aklys"}, []content.MissingContent{content.ItemID("aklys").Missing()}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := brbBarbarian(t, 1)
			got, ok := brbApply(t, src, in, c.choice).(character.BarbarianOptionContentMissing)
			if !ok || !slices.Equal(got.Missing(), c.missing) {
				t.Fatalf("= %#v, want content missing %v", got, c.missing)
			}
			got.Missing()[0] = content.MissingContent{}
			if got.Missing()[0] == (content.MissingContent{}) {
				t.Error("Missing returned a shared slice")
			}
		})
	}
	armor := brbApply(t, src, brbBarbarian(t, 1), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "leather_armor")})
	rej, ok := armor.(character.BarbarianOptionRejected)
	if !ok || !slices.Equal(brbReasons(rej.Problems()), []character.BarbarianReason{character.BarbarianNotSimpleOrMartialMeleeWeapon}) || rej.Problems()[0].Item != "leather_armor" {
		t.Errorf("an armor: %#v", armor)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-087, FR-103 - a character with no Barbarian levels is rejected with notABarbarian before any lookup (the source is never asked, so never content-missing) and the rule problems of a
// Barbarian come back together, in the order of the option's reason table, as copies.
// Path: per case (unhappy: state error).
func TestApplyBarbarianOptionChoiceRejections(t *testing.T) {
	t.Parallel()
	src := brbSource(t)
	res := brbApply(t, src, fgtChar(t, cl("wizard", 5)), character.BarbarianChooseWeaponMasteries{Kinds: ids("aklys", "zweihander")})
	rej, ok := res.(character.BarbarianOptionRejected)
	if !ok || !slices.Equal(rej.Problems(), []character.BarbarianProblem{{Reason: character.BarbarianNotABarbarian}}) || len(src.Queries()) != 0 {
		t.Errorf("no Barbarian levels: %#v, queries %v", res, src.Queries())
	}
	res = brbApply(t, src, brbBarbarian(t, 1), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "greataxe", "mace")})
	rej, ok = res.(character.BarbarianOptionRejected)
	if !ok || !slices.Equal(brbReasons(rej.Problems()), []character.BarbarianReason{character.BarbarianWrongMasteryCount, character.BarbarianDuplicateMastery}) {
		t.Fatalf("masteries: %#v", res)
	}
	rej.Problems()[0] = character.BarbarianProblem{}
	if rej.Problems()[0].Reason == 0 {
		t.Error("Problems returned a shared slice")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-086 - a source that returns its error, panics or answers a weapon key with a non-weapon entry is a *srderr.PortError for content.Port with the cause reachable where there is one,
// the result nil and the Character unchanged.
// Path: per case (unhappy: collaborator failure).
func TestApplyBarbarianOptionChoiceSourceFailures(t *testing.T) {
	t.Parallel()
	down := errors.New("source down")
	choice := character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}
	for _, c := range []struct {
		name  string
		src   character.ContentSource
		cause error
	}{
		{"an error", brbSource(t).FailLookup(down), down}, {"a panic", fgtPanicSource{}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := brbApplyUseCase(t, c.src).Run(context.Background(), character.ApplyBarbarianOptionChoiceCommand{Character: brbBarbarian(t, 1), Choice: choice})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || (c.cause != nil && !errors.Is(err, c.cause)) {
				t.Errorf("= %v, %v; want a PortError for content", res, err)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-088 - the result is the same with the no-op and a recording telemetry; the constructor refuses the zero progression and a nil source; Run refuses the zero use case, a zero Character,
// a nil or foreign choice, an invalid item id, a foreign state under the Barbarian key and a cancelled context, before any lookup.
// Path: per case (unhappy: invalid input, cancellation, state error).
func TestApplyBarbarianOptionChoiceInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	src := brbSource(t)
	good := character.ApplyBarbarianOptionChoiceCommand{Character: brbBarbarian(t, 1), Choice: character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}}
	_, opts := fgtTelemetry()
	withRec, err := brbApplyUseCase(t, src, opts...).Run(context.Background(), good)
	plain, err2 := brbApplyUseCase(t, src).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v vs %#v", withRec, plain)
	}
	_, err = character.NewApplyBarbarianOptionChoice(character.BarbarianProgression{}, src)
	wantInvalid(t, err, "zero progression")
	_, err = character.NewApplyBarbarianOptionChoice(brbProgression(t), nil)
	wantInvalid(t, err, "nil source")
	foreignMastery := must(character.WithFeatureState(brbBarbarian(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("x"))))
	foreignBarbarian := must(character.WithFeatureState(brbBarbarian(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("barbarian"), must(character.NewChosenSubclass("x"))))
	uc := brbApplyUseCase(t, src)
	bad := func(c character.Character, ch character.BarbarianOptionChoice) character.ApplyBarbarianOptionChoiceCommand {
		return character.ApplyBarbarianOptionChoiceCommand{Character: c, Choice: ch}
	}
	for name, cmd := range map[string]character.ApplyBarbarianOptionChoiceCommand{
		"zero character":    bad(character.Character{}, good.Choice),
		"nil choice":        bad(good.Character, nil),
		"foreign choice":    bad(good.Character, brbForeignChoice{}),
		"invalid item id":   bad(good.Character, character.BarbarianChooseWeaponMasteries{Kinds: []content.ItemID{"Not Valid"}}),
		"invalid swap id":   bad(good.Character, character.BarbarianSwapWeaponMastery{Remove: "Not Valid", Add: "mace"}),
		"invalid swap add":  bad(good.Character, character.BarbarianSwapWeaponMastery{Remove: "mace", Add: "Not Valid"}),
		"foreign mastery":   bad(foreignMastery, good.Choice),
		"foreign Barbarian": bad(foreignBarbarian, good.Choice),
	} {
		if r, err := uc.Run(context.Background(), cmd); r != nil {
			t.Errorf("%s: result %v", name, r)
		} else {
			wantInvalid(t, err, name)
		}
	}
	_, err = character.ApplyBarbarianOptionChoice{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	probe := brbSource(t)
	if r, err := brbApplyUseCase(t, probe).Run(cancelled, good); r != nil || !errors.Is(err, context.Canceled) || len(probe.Queries()) != 0 {
		t.Errorf("cancelled: %v, %v, queries %v", r, err, probe.Queries())
	}
}

// Requirement: doc/specs/class-barbarian.md FR-089, FR-090, FR-091, FR-124 - ApplyBarbarianOptionChoice records feature weapon_mastery for class barbarian; its span and duration cover the lookups; content-missing sets error
// status with the rejection content_missing, no error.type and no log; a source failure sets error status, records the error once with error.type content_error and logs nothing; a rejection with several reasons records the
// first one as the single rejection attribute; a success is logged once.
// Path: per case (happy / unhappy: missing content, collaborator failure, rule interaction).
func TestFeatureSignalsApplyBarbarianOption(t *testing.T) {
	t.Parallel()
	run := func(t *testing.T, src character.ContentSource, c character.Character, choice character.BarbarianOptionChoice, w wantCall) {
		t.Helper()
		rec, opts := fgtTelemetry()
		_, _ = brbApplyUseCase(t, src, opts...).Run(context.Background(), character.ApplyBarbarianOptionChoiceCommand{Character: c, Choice: choice})
		w.span, w.class, w.feature = character.SpanApplyBarbarianOption, character.ValueClassBarbarian, "weapon_mastery"
		checkFeatureCall(t, rec, w)
	}
	choice := character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "handaxe")}
	t.Run("applied", func(t *testing.T) {
		t.Parallel()
		run(t, brbSource(t), brbBarbarian(t, 1), choice, wantCall{outcome: "success", logged: true})
	})
	t.Run("a missing weapon", func(t *testing.T) {
		t.Parallel()
		run(t, brbSource(t), brbBarbarian(t, 1), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "aklys")}, wantCall{status: telemetry.StatusError, outcome: "failure", rejection: "content_missing"})
	})
	t.Run("a source failure", func(t *testing.T) {
		t.Parallel()
		run(t, brbSource(t).FailLookup(errBoom), brbBarbarian(t, 1), choice, wantCall{status: telemetry.StatusError, outcome: "failure", errorType: "content_error"})
	})
	t.Run("the first of several reasons", func(t *testing.T) {
		t.Parallel()
		run(t, brbSource(t), brbBarbarian(t, 1), character.BarbarianChooseWeaponMasteries{Kinds: ids("greataxe", "greataxe", "mace")}, wantCall{outcome: "failure", rejection: "wrong_mastery_count", logged: true})
	})
	t.Run("not a Barbarian", func(t *testing.T) {
		t.Parallel()
		run(t, brbSource(t), fgtChar(t, cl("wizard", 4)), choice, wantCall{outcome: "failure", rejection: "not_a_barbarian", logged: true})
	})
}
