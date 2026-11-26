package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
)

// grpD2Weapon builds a synthetic weapon of the type and category; the bundle holds no weapons yet.
func grpD2Weapon(id string, cat content.WeaponCategory, typ content.WeaponType, mastery content.MasteryProperty) content.Weapon {
	dmg := must2(content.NewWeaponDiceDamage(dice.MustExpression(1, dice.D8(), 0), content.DamageSlashing))
	return must2(content.NewWeapon(content.ItemID(id), id, content.SourceCitation{}, cat, typ, dmg, nil, mastery,
		content.MustWeight(1, ""), content.MustFixedCost(content.MustCost(1, content.CoinGP, 0))))
}

func grpD2Items(names ...string) []content.ItemID {
	out := make([]content.ItemID, len(names))
	for i, n := range names {
		out[i] = content.ItemID(n)
	}
	return out
}

func grpD2BarbarianReasons(ps []character.BarbarianProblem) []character.BarbarianReason {
	var out []character.BarbarianReason
	for _, p := range ps {
		out = append(out, p.Reason)
	}
	return out
}

// A level 1 Barbarian over the bundled Barbarian definition chooses two Weapon Mastery kinds from a source of melee weapons, reaches level 4 and chooses a third, and
// swaps one kind after a Long Rest of its own (SRD 5.2.1, "Level 1: Weapon Mastery", p. 29; "Barbarian Features", p. 28).
// Requirement: doc/specs/class-barbarian.md FR-041, FR-043, FR-046, FR-103 - ApplyBarbarianOptionChoice applies the mastery count of the level, a swap needs a finished Long Rest.
// Path: happy.
func TestBarbarianWeaponMasteryChoiceFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewBarbarianProgression(grpD2Class(t, "barbarian")))
	src := contenttest.NewFakeSource(
		grpD2Weapon("greataxe", content.WeaponMartial, content.WeaponMelee, content.MasteryCleave),
		grpD2Weapon("handaxe", content.WeaponSimple, content.WeaponMelee, content.MasteryVex),
		grpD2Weapon("mace", content.WeaponSimple, content.WeaponMelee, content.MasterySap),
		grpD2Weapon("longbow", content.WeaponMartial, content.WeaponRanged, content.MasterySlow))
	uc := must2(character.NewApplyBarbarianOptionChoice(prog, src))
	hero := grpD2Flat(t, 10, grpD2CL("barbarian", 1))

	res := must2(uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Character: hero, Choice: character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("greataxe", "handaxe")}}))
	chosen, ok := res.(character.BarbarianOptionApplied)
	if !ok || len(chosen.Events()) != 1 {
		t.Fatalf("two kinds at level 1 = %#v, want applied with one event", res)
	}
	access := must2(character.WeaponMasteryAccessFor(chosen.Character().FeatureStates())).(character.MasteryFromChoice)
	if !slices.Equal(access.Choice.Kinds(), grpD2Items("greataxe", "handaxe")) {
		t.Errorf("chosen kinds = %v", access.Choice.Kinds())
	}
	// Level 4 allows three kinds: the character holds the first two and names all three.
	four := must2(character.WithFeatureState(grpD2Flat(t, 10, grpD2CL("barbarian", 4)), character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(access.Choice)))
	more, ok := must2(uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Character: four, Choice: character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("greataxe", "handaxe", "mace")}})).(character.BarbarianOptionApplied)
	if !ok || !slices.Equal(must2(character.WeaponMasteryAccessFor(more.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(), grpD2Items("greataxe", "handaxe", "mace")) {
		t.Fatalf("three kinds at level 4 = %#v", more)
	}
	swapped, ok := must2(uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Character: chosen.Character(), Choice: character.BarbarianSwapWeaponMastery{
		Remove: "handaxe", Add: "mace", RestApplied: grpD2Rest(t, chosen.Character().ID(), character.LongRest)}})).(character.BarbarianOptionApplied)
	if !ok || !slices.Equal(must2(character.WeaponMasteryAccessFor(swapped.Character().FeatureStates())).(character.MasteryFromChoice).Choice.Kinds(), grpD2Items("greataxe", "mace")) {
		t.Fatalf("a swap after a Long Rest = %#v", swapped)
	}
}

// The unhappy paths of the option choice: a weapon the source lacks is one content-missing result naming every missing entry once and in id order, whatever rule would
// refuse; a ranged weapon, a wrong count and a repeated kind are rule problems reported together; a swap with a Short Rest or a kind not held is refused; a
// character with no Barbarian levels is refused before the source is asked.
// Requirement: doc/specs/class-barbarian.md FR-043, FR-085, FR-087, FR-103 - the rejection and content-missing results of ApplyBarbarianOptionChoice.
// Path: unhappy (missing content, state error, boundary).
func TestBarbarianWeaponMasteryChoiceRefusals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewBarbarianProgression(grpD2Class(t, "barbarian")))
	src := contenttest.NewFakeSource(
		grpD2Weapon("greataxe", content.WeaponMartial, content.WeaponMelee, content.MasteryCleave),
		grpD2Weapon("mace", content.WeaponSimple, content.WeaponMelee, content.MasterySap),
		grpD2Weapon("longbow", content.WeaponMartial, content.WeaponRanged, content.MasterySlow))
	uc := must2(character.NewApplyBarbarianOptionChoice(prog, src))
	run := func(hero character.Character, choice character.BarbarianOptionChoice) character.ApplyBarbarianOptionChoiceResult {
		return must2(uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Character: hero, Choice: choice}))
	}
	hero := grpD2Flat(t, 10, grpD2CL("barbarian", 1))

	missing, ok := run(hero, character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("zweihander", "mace", "aklys", "zweihander")}).(character.BarbarianOptionContentMissing)
	if want := []content.MissingContent{content.ItemID("aklys").Missing(), content.ItemID("zweihander").Missing()}; !ok || !slices.Equal(missing.Missing(), want) {
		t.Errorf("missing weapons = %#v, want %v once each in id order", missing, want)
	}
	ranged, ok := run(hero, character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("greataxe", "longbow")}).(character.BarbarianOptionRejected)
	if !ok || !slices.Equal(grpD2BarbarianReasons(ranged.Problems()), []character.BarbarianReason{character.BarbarianNotSimpleOrMartialMeleeWeapon}) || ranged.Problems()[0].Item != "longbow" {
		t.Errorf("a ranged weapon = %#v", ranged)
	}
	together, ok := run(hero, character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("greataxe", "greataxe", "mace")}).(character.BarbarianOptionRejected)
	if !ok || !slices.Equal(grpD2BarbarianReasons(together.Problems()), []character.BarbarianReason{character.BarbarianWrongMasteryCount, character.BarbarianDuplicateMastery}) {
		t.Errorf("a wrong count and a repeated kind = %#v, want both reported", together)
	}
	// A swap with a Short Rest, and a swap of a kind the character does not hold.
	held := must2(character.WithFeatureState(hero, character.WeaponMasteryChoiceKey(),
		character.NewWeaponMasteryChoiceState(must2(run2Choice(t, uc, hero, "greataxe", "mace")))))
	short, ok := run(held, character.BarbarianSwapWeaponMastery{Remove: "mace", Add: "greataxe", RestApplied: grpD2Rest(t, held.ID(), character.ShortRest)}).(character.BarbarianOptionRejected)
	if !ok || !slices.Contains(grpD2BarbarianReasons(short.Problems()), character.BarbarianNoLongRestFinished) {
		t.Errorf("a swap after a Short Rest = %#v, want no Long Rest finished", short)
	}
	notHeld, ok := run(held, character.BarbarianSwapWeaponMastery{Remove: "longbow", Add: "mace", RestApplied: grpD2Rest(t, held.ID(), character.LongRest)}).(character.BarbarianOptionRejected)
	if !ok || !slices.Contains(grpD2BarbarianReasons(notHeld.Problems()), character.BarbarianNotCurrentlyChosen) {
		t.Errorf("a swap of a kind not held = %#v, want not currently chosen", notHeld)
	}
	asked := len(src.Queries())
	wizard, ok := run(grpD2Flat(t, 10, grpD2CL("wizard", 5)), character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("aklys", "zweihander")}).(character.BarbarianOptionRejected)
	if !ok || !slices.Equal(grpD2BarbarianReasons(wizard.Problems()), []character.BarbarianReason{character.BarbarianNotABarbarian}) || len(src.Queries()) != asked {
		t.Errorf("no Barbarian levels = %#v (queries %d -> %d), want not a Barbarian and no lookup", wizard, asked, len(src.Queries()))
	}
}

// run2Choice applies a two-kind choice and returns the resulting WeaponMasteryChoice.
func run2Choice(t *testing.T, uc character.ApplyBarbarianOptionChoice, hero character.Character, a, b string) (character.WeaponMasteryChoice, error) {
	t.Helper()
	res := must2(uc.Run(context.Background(), character.ApplyBarbarianOptionChoiceCommand{Character: hero, Choice: character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items(a, b)}}))
	applied, ok := res.(character.BarbarianOptionApplied)
	if !ok {
		t.Fatalf("choice = %#v, want applied", res)
	}
	return must2(character.WeaponMasteryAccessFor(applied.Character().FeatureStates())).(character.MasteryFromChoice).Choice, nil
}

// A failing content source is a port failure of the content port with the character untouched, never a result; invalid construction and commands are invalid input.
// Requirement: doc/specs/class-barbarian.md FR-086 - the source failure propagates unchanged; architecture.md §4 - invalid input and cancellation.
// Path: unhappy (collaborator failure, invalid input, cancellation).
func TestBarbarianWeaponMasteryChoiceFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewBarbarianProgression(grpD2Class(t, "barbarian")))
	boom := errors.New("source down")
	failing := contenttest.NewFakeSource().FailLookup(boom)
	hero := grpD2Flat(t, 10, grpD2CL("barbarian", 1))
	cmd := character.ApplyBarbarianOptionChoiceCommand{Character: hero, Choice: character.BarbarianChooseWeaponMasteries{Kinds: grpD2Items("greataxe", "mace")}}
	res, err := must2(character.NewApplyBarbarianOptionChoice(prog, failing)).Run(ctx, cmd)
	if res != nil {
		t.Errorf("a failing source returned %#v", res)
	}
	grpD2WantPortFailure(t, err, content.Port, boom)

	_, err = character.NewApplyBarbarianOptionChoice(character.BarbarianProgression{}, failing)
	grpD2WantInvalid(t, err, "a zero progression")
	_, err = character.NewApplyBarbarianOptionChoice(prog, nil)
	grpD2WantInvalid(t, err, "a nil source")
	uc := must2(character.NewApplyBarbarianOptionChoice(prog, contenttest.NewFakeSource()))
	_, err = uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Choice: cmd.Choice})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Character: hero})
	grpD2WantInvalid(t, err, "a nil choice")
	_, err = uc.Run(ctx, character.ApplyBarbarianOptionChoiceCommand{Character: hero, Choice: character.BarbarianChooseWeaponMasteries{Kinds: []content.ItemID{""}}})
	grpD2WantInvalid(t, err, "an invalid item id")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v, want context.Canceled", err)
	}
}
