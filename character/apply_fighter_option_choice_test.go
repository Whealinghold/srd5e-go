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
)

// fgtSource holds the Fighting Style feats, a feat outside the category, every weapon fixture and one armor.
func fgtSource(t *testing.T) *contenttest.FakeSource {
	t.Helper()
	entries := []content.Entry{fgtStyle("archery"), fgtStyle("defense"), fgtStyle("two_weapon_fighting"), alertFeat(), leatherArmor(t)}
	for _, w := range weaponFixtures(t) {
		entries = append(entries, w)
	}
	return contenttest.NewFakeSource(entries...)
}

func applyUseCase(t *testing.T, src character.ContentSource, opts ...character.Option) character.ApplyFighterOptionChoice {
	t.Helper()
	return must(character.NewApplyFighterOptionChoice(fgtProgression(t), src, opts...))
}

func apply(t *testing.T, src character.ContentSource, c character.Character, feats character.TakenFeats, choice character.FighterOptionChoice) character.ApplyFighterOptionChoiceResult {
	t.Helper()
	res, err := applyUseCase(t, src).Run(context.Background(), character.ApplyFighterOptionChoiceCommand{Character: c, Feats: feats, Choice: choice})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

// Requirement: doc/specs/class-fighter.md FR-032, FR-036, FR-039, FR-157, FR-163 - the three Fighting Style choices through the content source: the level 1 style, its
// replacement on gaining a Fighter level and the Champion's Additional Fighting Style each return the new Character, the new TakenFeats and the events, and write only
// their own marker of the FighterState.
// Path: happy.
func TestApplyFighterOptionChoiceStyles(t *testing.T) {
	t.Parallel()
	src := fgtSource(t)
	first, ok := apply(t, src, fgtFighter(t, 1), character.TakenFeats{}, character.FighterChooseFightingStyle{Choice: styleChoice("defense")}).(character.FighterOptionApplied)
	if !ok || fgtStateOf(t, first.Character()).FightingStyle() != "defense" || !first.Feats().Holds("defense") || len(first.Events()) != 2 {
		t.Fatalf("level 1 style: %#v", first)
	}
	facts, lg := fgtReplaceFacts(t, fgtStyle("archery"))
	replaced, ok := apply(t, src, facts.Character, facts.Feats, character.FighterReplaceFightingStyle{Choice: styleChoice("archery"), LevelGained: lg}).(character.FighterOptionApplied)
	if !ok || fgtStateOf(t, replaced.Character()).FightingStyle() != "archery" || replaced.Feats().Holds("defense") || !replaced.Feats().Holds("archery") || len(replaced.Events()) != 3 {
		t.Fatalf("replacement: %#v", replaced)
	}
	seven := fgtWith(t, fgtFighter(t, 7), character.MustFighterState(0, 0, 0, "defense", ""))
	extra, ok := apply(t, src, seven, fgtFeats(t, content.FeatFightingStyle, "defense"), character.FighterChooseAdditionalFightingStyle{Choice: styleChoice("archery"), Subclass: champion}).(character.FighterOptionApplied)
	state := fgtStateOf(t, extra.Character())
	if !ok || state.FightingStyle() != "defense" || state.AdditionalFightingStyle() != "archery" || !extra.Feats().Holds("defense") || !extra.Feats().Holds("archery") {
		t.Fatalf("additional style: %#v", extra)
	}
	extra.Events()[0] = nil
	if extra.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
}

// Requirement: doc/specs/class-fighter.md FR-046, FR-163 - the Weapon Mastery choices through the content source: three kinds at level 1 (resolved as weapons from the source)
// and one swap after a Long Rest of this character, with the TakenFeats unchanged.
// Path: happy.
func TestApplyFighterOptionChoiceMasteries(t *testing.T) {
	t.Parallel()
	src := fgtSource(t)
	feats := fgtFeats(t, content.FeatFightingStyle, "defense")
	chosen, ok := apply(t, src, fgtFighter(t, 1), feats, character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}).(character.FighterOptionApplied)
	if !ok || !chosen.Feats().Equal(feats) {
		t.Fatalf("masteries: %#v", chosen)
	}
	access := must(character.WeaponMasteryAccessFor(chosen.Character().FeatureStates())).(character.MasteryFromChoice)
	if !slices.Equal(access.Choice.Kinds(), ids("greatsword", "longbow", "mace")) {
		t.Errorf("choice = %v", access.Choice.Kinds())
	}
	swapped, ok := apply(t, src, chosen.Character(), feats, character.FighterSwapWeaponMastery{Remove: "mace", Add: "rapier", RestApplied: fgtRest(t, chosen.Character().ID(), character.LongRest)}).(character.FighterOptionApplied)
	if !ok || !slices.Equal(must(character.WeaponMasteryAccessFor(swapped.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(), ids("greatsword", "longbow", "rapier")) {
		t.Fatalf("swap: %#v", swapped)
	}
}

// Requirement: doc/specs/class-fighter.md FR-034, FR-120, FR-121, FR-129, FR-141 - a feat or weapon the source cannot find is FighterOptionContentMissing carrying every missing entry
// (once each, ordered by kind then id), with no rule reason even when a rule would refuse (Additional Fighting Style below level 7); a found item that is not a weapon is not
// missing but a notSimpleOrMartialWeapon problem; the Character is unchanged.
// Path: per case (unhappy: missing content).
func TestApplyFighterOptionChoiceContentMissing(t *testing.T) {
	t.Parallel()
	src := fgtSource(t)
	for _, c := range []struct {
		name    string
		char    character.Character
		choice  character.FighterOptionChoice
		missing []content.MissingContent
	}{
		{"a missing style feat", fgtFighter(t, 1), character.FighterChooseFightingStyle{Choice: styleChoice("no_such_style")}, []content.MissingContent{content.FeatID("no_such_style").Missing()}},
		{"a missing feat below level 7 for the Additional style", fgtFighter(t, 3), character.FighterChooseAdditionalFightingStyle{Choice: styleChoice("no_such_style"), Subclass: ""}, []content.MissingContent{content.FeatID("no_such_style").Missing()}},
		{"missing weapons, once each and in order", fgtFighter(t, 1), character.FighterChooseWeaponMasteries{Kinds: ids("zweihander", "mace", "aklys", "zweihander")}, []content.MissingContent{content.ItemID("aklys").Missing(), content.ItemID("zweihander").Missing()}},
		{"missing weapons named in ascending order", fgtFighter(t, 1), character.FighterChooseWeaponMasteries{Kinds: ids("aklys", "zweihander", "mace")}, []content.MissingContent{content.ItemID("aklys").Missing(), content.ItemID("zweihander").Missing()}},
		{"a missing added kind of a swap", fgtFighter(t, 1), character.FighterSwapWeaponMastery{Remove: "mace", Add: "aklys"}, []content.MissingContent{content.ItemID("aklys").Missing()}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := c.char
			res := apply(t, src, in, character.TakenFeats{}, c.choice)
			got, ok := res.(character.FighterOptionContentMissing)
			if !ok || !slices.Equal(got.Missing(), c.missing) || !in.Equal(c.char) {
				t.Fatalf("= %#v, want content missing %v", res, c.missing)
			}
			got.Missing()[0] = content.MissingContent{}
			if got.Missing()[0] == (content.MissingContent{}) {
				t.Error("Missing returned a shared slice")
			}
		})
	}
	armor := apply(t, src, fgtFighter(t, 1), character.TakenFeats{}, character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "mace", "leather_armor")})
	rej, ok := armor.(character.FighterOptionRejected)
	if !ok || !slices.Equal(fgtReasons(rej.Problems()), []character.FighterReason{character.FighterNotSimpleOrMartialWeapon}) || rej.Problems()[0].Item != "leather_armor" {
		t.Errorf("an armor: %#v", armor)
	}
}

// Requirement: doc/specs/class-fighter.md FR-057, FR-113, FR-121, FR-129 - a character with no Fighter levels is rejected with notAFighter before any lookup (the source is never
// asked, so never content-missing, even for a feat it lacks) and the rule problems of a Fighter come back together, in the order of the option's reason table, as copies.
// Path: per case (unhappy: state error, invalid input).
func TestApplyFighterOptionChoiceRejections(t *testing.T) {
	t.Parallel()
	src := fgtSource(t)
	wizard := fgtChar(t, cl("wizard", 5))
	res := apply(t, src, wizard, character.TakenFeats{}, character.FighterChooseFightingStyle{Choice: styleChoice("no_such_style")})
	rej, ok := res.(character.FighterOptionRejected)
	if !ok || !slices.Equal(rej.Problems(), []character.FighterProblem{{Reason: character.FighterNotAFighter}}) || len(src.Queries()) != 0 {
		t.Errorf("no Fighter levels: %#v, queries %v", res, src.Queries())
	}
	res = apply(t, src, fgtFighter(t, 1), character.TakenFeats{}, character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "mace", "mace", "club")})
	rej, ok = res.(character.FighterOptionRejected)
	want := []character.FighterReason{character.FighterWrongMasteryCount, character.FighterDuplicateMastery}
	if !ok || !slices.Equal(fgtReasons(rej.Problems()), want) {
		t.Fatalf("masteries: %#v", res)
	}
	rej.Problems()[0] = character.FighterProblem{}
	if rej.Problems()[0].Reason == 0 {
		t.Error("Problems returned a shared slice")
	}
	res = apply(t, src, fgtFighter(t, 3), character.TakenFeats{}, character.FighterChooseAdditionalFightingStyle{Choice: styleChoice("archery")})
	if rej, ok := res.(character.FighterOptionRejected); !ok || !slices.Equal(fgtReasons(rej.Problems()), []character.FighterReason{character.FighterFeatureNotGained}) {
		t.Errorf("Additional style at level 3: %#v", res)
	}
}

type fgtPanicSource struct{}

func (fgtPanicSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	panic(errors.New("source panic"))
}

// fgtWrongSource answers every key with a weapon, whatever the key is.
type fgtWrongSource struct{ w content.Weapon }

func (s fgtWrongSource) Lookup(_ context.Context, _ content.Key) (content.LookupResult, error) {
	return content.LookupFound{Entry: s.w}, nil
}

// Requirement: doc/specs/class-fighter.md FR-111 - a source that returns its error, panics or answers a feat key with another entry is a *srderr.PortError for content.Port with the
// cause reachable, the result nil and the Character unchanged.
// Path: per case (unhappy: collaborator failure).
func TestApplyFighterOptionChoiceSourceFailures(t *testing.T) {
	t.Parallel()
	down := errors.New("source down")
	choice := character.FighterChooseFightingStyle{Choice: styleChoice("archery")}
	for _, c := range []struct {
		name  string
		src   character.ContentSource
		cause error
	}{
		{"an error", fgtSource(t).FailLookup(down), down},
		{"a panic", fgtPanicSource{}, nil},
		{"another entry for a feat key", fgtWrongSource{w: weaponFixtures(t)["mace"]}, srderr.ErrContractViolation},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := applyUseCase(t, c.src).Run(context.Background(), character.ApplyFighterOptionChoiceCommand{Character: fgtFighter(t, 1), Choice: choice})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || (c.cause != nil && !errors.Is(err, c.cause)) {
				t.Errorf("= %v, %v; want a PortError for content", res, err)
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-112, FR-149 - the result is the same with the no-op and a recording telemetry; the constructor refuses the zero progression and a nil source; Run refuses the
// zero use case, a zero Character, a nil or foreign choice, an invalid feat or item id, a foreign state under a Fighter key and a cancelled context, before any lookup.
// Path: per case (unhappy: invalid input, cancellation, state error).
func TestApplyFighterOptionChoiceInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	src := fgtSource(t)
	good := character.ApplyFighterOptionChoiceCommand{Character: fgtFighter(t, 1), Choice: character.FighterChooseFightingStyle{Choice: styleChoice("archery")}}
	_, opts := fgtTelemetry()
	withRec, err := applyUseCase(t, src, opts...).Run(context.Background(), good)
	plain, err2 := applyUseCase(t, src).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v vs %#v", withRec, plain)
	}
	_, err = character.NewApplyFighterOptionChoice(character.FighterProgression{}, src)
	wantInvalid(t, err, "zero progression")
	_, err = character.NewApplyFighterOptionChoice(fgtProgression(t), nil)
	wantInvalid(t, err, "nil source")
	foreignMastery := must(character.WithFeatureState(fgtFighter(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("champion"))))
	foreignFighter := must(character.WithFeatureState(fgtFighter(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("fighter"), must(character.NewChosenSubclass("champion"))))
	uc := applyUseCase(t, src)
	bad := func(c character.Character, ch character.FighterOptionChoice) character.ApplyFighterOptionChoiceCommand {
		return character.ApplyFighterOptionChoiceCommand{Character: c, Choice: ch}
	}
	for name, cmd := range map[string]character.ApplyFighterOptionChoiceCommand{
		"zero character":    bad(character.Character{}, good.Choice),
		"nil choice":        bad(good.Character, nil),
		"foreign choice":    bad(good.Character, fgtForeignChoice{}),
		"zero style choice": bad(good.Character, character.FighterChooseFightingStyle{}),
		"invalid item id":   bad(good.Character, character.FighterChooseWeaponMasteries{Kinds: []content.ItemID{"Not Valid"}}),
		"invalid swap id":   bad(good.Character, character.FighterSwapWeaponMastery{Remove: "Not Valid", Add: "mace"}),
		"foreign mastery":   bad(foreignMastery, character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}),
		"foreign Fighter":   bad(foreignFighter, good.Choice),
	} {
		if r, err := uc.Run(context.Background(), cmd); r != nil {
			t.Errorf("%s: result %v", name, r)
		} else {
			wantInvalid(t, err, name)
		}
	}
	_, err = character.ApplyFighterOptionChoice{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	probe := fgtSource(t)
	if r, err := applyUseCase(t, probe).Run(cancelled, good); r != nil || !errors.Is(err, context.Canceled) || len(probe.Queries()) != 0 {
		t.Errorf("cancelled: %v, %v, queries %v", r, err, probe.Queries())
	}
}
