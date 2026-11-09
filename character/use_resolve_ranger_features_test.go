package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func resolveRanger(t *testing.T, opts ...character.Option) character.ResolveRangerFeatures {
	t.Helper()
	return must(character.NewResolveRangerFeatures(rgrProgression(t), opts...))
}

func resolveCmd(t *testing.T, c character.Character, subclass content.SubclassID) character.ResolveRangerFeaturesCommand {
	t.Helper()
	return character.ResolveRangerFeaturesCommand{Character: c, Subclass: subclass, Speed: character.MustSpeedInput(30, true), Wisdom: modifierOf(t, 3), HuntersMarkDie: dice.D6()}
}

func resolved(t *testing.T, cmd character.ResolveRangerFeaturesCommand) character.ResolveRangerFeaturesResolved {
	t.Helper()
	res, err := resolveRanger(t).Run(context.Background(), cmd)
	r, ok := res.(character.ResolveRangerFeaturesResolved)
	if err != nil || !ok {
		t.Fatalf("Run = %#v, %v", res, err)
	}
	return r
}

// Requirement: doc/specs/class-ranger.md FR-032, FR-065 to FR-070, FR-078, FR-082, FR-083, FR-096 (SRD 5.2.1, "Ranger Class Features", pp. 58-59; "Ranger Subclass: Hunter", p. 61) - ResolveRangerFeatures reports the effects of the level and choices: attacks per Attack action, the Speed profile
// (Roving, Heavy armor), the senses, the always-prepared Hunter’s Mark, the Druidic Warrior cantrips, the Hunter’s Mark die, the Concentration rule, the maximum uses of Favored Enemy, Tireless and Nature’s Veil and the Hunter features held; effects of a choice not made are absent.
// Path: per case (happy / unhappy: boundary levels, rule interaction, empty case).
func TestResolveRangerFeaturesEffects(t *testing.T) {
	t.Parallel()
	druid := rgrState(t, character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "starry_wisp"}}})
	for _, c := range []struct {
		name                     string
		char                     character.Character
		subclass                 content.SubclassID
		heavy                    bool
		attacks, walk            int
		climb                    bool
		senses                   int
		die                      dice.Die
		relentless               bool
		favored, tireless, veil  int
		hunterFeatures, cantrips int
	}{
		{"level 1", rgrRanger(t, 1), "", false, 1, 30, false, 0, dice.D6(), false, 2, 0, 0, 0, 0},
		{"level 5", rgrRanger(t, 5), "", false, 2, 30, false, 0, dice.D6(), false, 3, 0, 0, 0, 0},
		{"level 6", rgrRanger(t, 6), "", false, 2, 40, true, 0, dice.D6(), false, 3, 0, 0, 0, 0},
		{"level 6 in Heavy armor", rgrRanger(t, 6), "", true, 2, 30, true, 0, dice.D6(), false, 3, 0, 0, 0, 0},
		{"level 10", rgrRanger(t, 10), "", false, 2, 40, true, 0, dice.D6(), false, 4, 3, 0, 0, 0},
		{"level 13", rgrRanger(t, 13), hunter, false, 2, 40, true, 0, dice.D6(), true, 5, 3, 0, 4, 0},
		{"level 14", rgrRanger(t, 14), hunter, false, 2, 40, true, 0, dice.D6(), true, 5, 3, 3, 4, 0},
		{"level 18", rgrRanger(t, 18), hunter, false, 2, 40, true, 1, dice.D6(), true, 6, 3, 3, 5, 0},
		{"level 20", rgrRanger(t, 20), hunter, false, 2, 40, true, 1, dice.D10(), true, 6, 3, 3, 5, 0},
		{"level 3 with Druidic Warrior", rgrWith(t, rgrRanger(t, 3), druid), hunter, false, 1, 30, false, 0, dice.D6(), false, 2, 0, 0, 2, 2},
		{"level 20 without a subclass", rgrRanger(t, 20), "", false, 2, 40, true, 1, dice.D10(), true, 6, 3, 3, 0, 0},
	} {
		before := c.char
		cmd := resolveCmd(t, c.char, c.subclass)
		cmd.HeavyArmor = c.heavy
		r := resolved(t, cmd)
		walk := r.Speed().Walk()
		_, hasClimb := r.Speed().Climb()
		if r.AttacksPerAttackAction() != c.attacks || walk != c.walk || hasClimb != c.climb || len(r.Senses()) != c.senses || r.HuntersMarkDie() != c.die || r.RelentlessHunter() != c.relentless ||
			r.FavoredEnemyUses() != c.favored || r.TirelessUses() != c.tireless || r.NaturesVeilUses() != c.veil || len(r.HunterFeatures()) != c.hunterFeatures || len(r.GrantedCantrips()) != c.cantrips ||
			!slices.Equal(r.AlwaysPreparedSpells(), []content.SpellID{"hunters_mark"}) || !c.char.Equal(before) {
			t.Errorf("%s: attacks %d, walk %d, climb %v, senses %v, die %v, relentless %v, uses %d/%d/%d, hunter %v, cantrips %v", c.name, r.AttacksPerAttackAction(), walk, hasClimb, r.Senses(), r.HuntersMarkDie(),
				r.RelentlessHunter(), r.FavoredEnemyUses(), r.TirelessUses(), r.NaturesVeilUses(), r.HunterFeatures(), r.GrantedCantrips())
		}
	}
	r := resolved(t, resolveCmd(t, rgrRanger(t, 20), hunter))
	if want := []character.RangerFeature{character.RangerFeatureHuntersLore, character.RangerFeatureHuntersPrey, character.RangerFeatureDefensiveTactics, character.RangerFeatureSuperiorHuntersPrey, character.RangerFeatureSuperiorHuntersDefense}; !slices.Equal(r.HunterFeatures(), want) {
		t.Errorf("Hunter features = %v, want %v", r.HunterFeatures(), want)
	}
	low := resolveCmd(t, rgrRanger(t, 14), hunter)
	low.Wisdom = modifierOf(t, -2)
	if r := resolved(t, low); r.TirelessUses() != 1 || r.NaturesVeilUses() != 1 {
		t.Errorf("a negative Wisdom modifier gives %d and %d uses, want the minimum 1", r.TirelessUses(), r.NaturesVeilUses())
	}
	zero := resolveCmd(t, rgrRanger(t, 20), hunter)
	zero.Speed = character.MustSpeedInput(0, false)
	if w := resolved(t, zero).Speed().Walk(); w != 0 {
		t.Errorf("Speed 0 that cannot increase walks %d", w)
	}
	for _, mutate := range []func(character.ResolveRangerFeaturesResolved){
		func(r character.ResolveRangerFeaturesResolved) { r.AlwaysPreparedSpells()[0] = "x" },
		func(r character.ResolveRangerFeaturesResolved) { r.HunterFeatures()[0] = 0 },
		func(r character.ResolveRangerFeaturesResolved) { r.PendingChoices()[0] = 0 },
		func(r character.ResolveRangerFeaturesResolved) { r.Senses()[0] = character.RangerSenseGrant{} },
	} {
		mutate(r)
	}
	if again := resolved(t, resolveCmd(t, rgrRanger(t, 20), hunter)); !reflect.DeepEqual(again, r) {
		t.Error("a returned slice shares storage with the result")
	}
}

// Requirement: doc/specs/class-ranger.md FR-097, T11 (SRD 5.2.1, "Ranger Features", p. 58; pp. 57-59 and 61) - the pending choices at a level with nothing made, rows PC-01 to PC-10 of T11; a choice made leaves the list: Weapon Mastery (held pair), Deft Explorer, Fighting Style, the Ranger Subclass (chosen),
// Hunter’s Prey, Ability Score Improvement grants used, Defensive Tactics, Expertise and the Epic Boon.
// Path: per case (T11): happy / unhappy (boundary, empty and minimal, multiclass entry).
func TestResolveRangerFeaturesPendingChoices(t *testing.T) {
	t.Parallel()
	const (
		wm, de, fs, sub, prey, asi, dt, ex, boon = character.RangerPendingWeaponMastery, character.RangerPendingDeftExplorer, character.RangerPendingFightingStyle, character.RangerPendingSubclass,
			character.RangerPendingHuntersPrey, character.RangerPendingAbilityScoreImprovement, character.RangerPendingDefensiveTactics, character.RangerPendingExpertise, character.RangerPendingEpicBoon
	)
	multiclass := restored(t, 355000, 200, flatScores(t, 10), cl("fighter", 3), cl("ranger", 1))
	all := rgrState(t, character.RangerStateParts{DeftExplorer: &character.DeftExplorerChoice{Skill: sStealth, Languages: [2]content.Language{lElvish, lDwarvish}}, Expertise: &[2]content.Skill{sSurvival, sNature},
		FightingStyle: character.RangerFightingStyleFeat{Feat: "archery"}, HuntersPrey: "colossus_slayer", DefensiveTactics: "escape_the_horde", GeneralFeats: 4, EpicBoons: 1})
	held := must(character.WithFeatureState(rgrWith(t, rgrRanger(t, 20), all), character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(choiceOf(t, "longbow", "shortsword"))))
	for _, c := range []struct {
		label    string
		char     character.Character
		subclass content.SubclassID
		want     []character.RangerPendingChoice
	}{
		{"PC-01", rgrRanger(t, 1), "", []character.RangerPendingChoice{wm}},
		{"PC-02", rgrRanger(t, 2), "", []character.RangerPendingChoice{wm, de, fs}},
		{"PC-03", rgrRanger(t, 3), "", []character.RangerPendingChoice{wm, de, fs, sub}},
		{"PC-04", rgrRanger(t, 3), hunter, []character.RangerPendingChoice{wm, de, fs, prey}},
		{"PC-05", rgrRanger(t, 4), hunter, []character.RangerPendingChoice{wm, de, fs, prey, asi}},
		{"PC-06", rgrRanger(t, 7), hunter, []character.RangerPendingChoice{wm, de, fs, prey, asi, dt}},
		{"PC-07", rgrRanger(t, 9), hunter, []character.RangerPendingChoice{wm, de, fs, prey, asi, asi, dt, ex}},
		{"PC-08", rgrRanger(t, 19), hunter, []character.RangerPendingChoice{wm, de, fs, prey, asi, asi, asi, asi, dt, ex, boon}},
		{"PC-09", held, hunter, []character.RangerPendingChoice{}},
		{"PC-10", multiclass, "", []character.RangerPendingChoice{wm}},
	} {
		if got := resolved(t, resolveCmd(t, c.char, c.subclass)).PendingChoices(); !slices.Equal(got, c.want) {
			t.Errorf("%s = %v, want %v", c.label, got, c.want)
		}
	}
	partial := rgrWith(t, rgrRanger(t, 19), rgrState(t, character.RangerStateParts{GeneralFeats: 3, EpicBoons: 1}))
	if got := resolved(t, resolveCmd(t, partial, hunter)).PendingChoices(); !slices.Equal(got, []character.RangerPendingChoice{wm, de, fs, prey, asi, dt, ex}) {
		t.Errorf("with feats taken = %v", got)
	}
}

// Requirement: doc/specs/class-ranger.md FR-103, FR-104, FR-105 - ResolveRangerFeatures records one span named srd5e.character.resolve_ranger_features with the class ranger and the feature ranger_features; a character with no Ranger levels is NotARanger and counts not_a_ranger; a foreign state is an error
// recorded on the span only; the result is identical with the no-op and with a recording telemetry; the constructor refuses the zero progression and Run the zero use case, a zero Character and a zero die before any signal, and a cancelled context.
// Path: per case (happy / unhappy: invalid input, state error, determinism, cancellation).
func TestResolveRangerFeaturesSignalsAndInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	good := resolveCmd(t, rgrRanger(t, 14), hunter)
	rec, opts := fgtTelemetry()
	withRec, err := resolveRanger(t, opts...).Run(ctx, good)
	plain, err2 := resolveRanger(t).Run(ctx, good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, rgrWant(character.SpanResolveRangerFeatures, "ranger_features", ""))
	rec, opts = fgtTelemetry()
	res, err := resolveRanger(t, opts...).Run(ctx, resolveCmd(t, fgtFighter(t, 5), ""))
	if _, ok := res.(character.ResolveRangerFeaturesNotARanger); err != nil || !ok {
		t.Fatalf("a Fighter = %#v, %v", res, err)
	}
	checkFeatureCall(t, rec, rgrWant(character.SpanResolveRangerFeatures, "ranger_features", "not_a_ranger"))
	rec, opts = fgtTelemetry()
	res, err = resolveRanger(t, opts...).Run(ctx, resolveCmd(t, rgrForeign(t, rgrRanger(t, 5)), ""))
	wantInvalid(t, err, "foreign state")
	if spans := rec.Tracer.Spans(); res != nil || len(spans) != 1 || spans[0].Status != telemetry.StatusError {
		t.Errorf("foreign state recorded %+v", spans)
	}
	foreignMastery := resolveCmd(t, must(character.WithFeatureState(rgrRanger(t, 3), character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("hunter")))), "")
	rec, opts = fgtTelemetry()
	res, err = resolveRanger(t, opts...).Run(ctx, foreignMastery)
	wantInvalid(t, err, "foreign mastery state")
	if spans := rec.Tracer.Spans(); res != nil || len(spans) != 1 || spans[0].Status != telemetry.StatusError {
		t.Errorf("foreign mastery state recorded %+v", spans)
	}
	_, err = character.NewResolveRangerFeatures(character.RangerProgression{})
	wantInvalid(t, err, "zero progression")
	_, err = character.ResolveRangerFeatures{}.Run(ctx, good)
	wantInvalid(t, err, "zero use case")
	_, err = resolveRanger(t).Run(ctx, character.ResolveRangerFeaturesCommand{HuntersMarkDie: dice.D6()})
	wantInvalid(t, err, "zero character")
	noDie := good
	noDie.HuntersMarkDie = dice.Die{}
	_, err = resolveRanger(t).Run(ctx, noDie)
	wantInvalid(t, err, "zero die")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec, opts = fgtTelemetry()
	res, err = resolveRanger(t, opts...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-ranger.md FR-064, FR-097 - more feat grants used than granted (here two Epic Boons at level 19) owe no further feat choice.
// Path: unhappy (boundary, state error).
func TestResolveRangerFeaturesNoPendingFeatWhenOverspent(t *testing.T) {
	t.Parallel()
	over := rgrWith(t, rgrRanger(t, 19), rgrState(t, character.RangerStateParts{GeneralFeats: 6, EpicBoons: 2}))
	for _, c := range resolved(t, resolveCmd(t, over, hunter)).PendingChoices() {
		if c == character.RangerPendingEpicBoon || c == character.RangerPendingAbilityScoreImprovement {
			t.Errorf("pending %v with every grant used", c)
		}
	}
}
