package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func chooseRanger(t *testing.T, opts ...character.Option) character.ChooseRangerOption {
	t.Helper()
	return must(character.NewChooseRangerOption(rgrProgression(t), opts...))
}

// Requirement: doc/specs/class-ranger.md FR-037, FR-042, FR-047, FR-049, FR-051, FR-086, FR-091, FR-108, FR-111, FR-113 - ChooseRangerOption stores each accepted choice (Deft Explorer, a Fighting Style feat, Druidic Warrior, Expertise, Hunter’s Prey, Defensive Tactics, a feat grant and
// the initial Weapon Mastery pair) in the RangerState and the Character, returns the events with the Character id (the feat events first), and leaves the input unchanged.
// Path: per case (happy).
func TestChooseRangerOptionAccepted(t *testing.T) {
	t.Parallel()
	general := featOf("grappler_general", content.FeatGeneral, false)
	for _, c := range []struct {
		name   string
		char   character.Character
		choice character.RangerChoice
		feat   string
		events int
	}{
		{"Deft Explorer", rgrRanger(t, 2), character.RangerChooseDeftExplorer{Skill: sStealth, Languages: []content.Language{lElvish, lDwarvish}}, "deft_explorer", 1},
		{"Fighting Style feat", rgrRanger(t, 2), character.RangerChooseFightingStyleFeat{Feat: fgtStyle("archery")}, "fighting_style", 2},
		{"Druidic Warrior", rgrRanger(t, 2), character.RangerChooseDruidicWarrior{Cantrips: []content.Spell{rgrCantrip(t, "guidance", "druid"), rgrCantrip(t, "starry_wisp", "druid")}}, "fighting_style", 1},
		{"Expertise", rgrRanger(t, 9), character.RangerChooseExpertise{Skills: []content.Skill{sSurvival, sNature}}, "expertise", 1},
		{"Hunter’s Prey", rgrRanger(t, 3), character.RangerChooseHuntersPrey{Option: "colossus_slayer"}, "hunters_prey", 1},
		{"Defensive Tactics", rgrRanger(t, 7), character.RangerChooseDefensiveTactics{Option: "multiattack_defense"}, "defensive_tactics", 1},
		{"a feat grant", rgrRanger(t, 4), character.RangerChooseFeat{Feat: general}, "feat_grant", 1},
		{"Weapon Mastery", rgrRanger(t, 1), character.RangerChooseWeaponMasteries{Kinds: ids("longbow", "shortsword"), Weapons: fgtWeapons(t, "longbow", "shortsword"), Proficiencies: rgrProficient()}, "weapon_mastery", 1},
	} {
		before := c.char
		res, err := chooseRanger(t).Run(context.Background(), character.ChooseRangerOptionCommand{Character: c.char, Subclass: hunter, Skills: rgrSkills(t), Choice: c.choice})
		a, ok := res.(character.ChooseRangerOptionAccepted)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.name, res, err)
		}
		last, isChoice := a.Events()[len(a.Events())-1].(character.RangerChoiceMade)
		if !a.State().Equal(rgrStateOf(t, a.Character())) || len(a.Events()) != c.events || !isChoice || last.Choice() != c.feat || last.CharacterID() != c.char.ID() || !c.char.Equal(before) {
			t.Errorf("%s: events %#v", c.name, a.Events())
		}
		if (c.name == "Fighting Style feat") != (a.Feats().Len() == 1) {
			t.Errorf("%s: feats = %v", c.name, a.Feats().All())
		}
		if (c.name == "a feat grant") == a.Grant().IsZero() {
			t.Errorf("%s: grant = %v", c.name, a.Grant())
		}
		a.Events()[0] = nil
		if a.Events()[0] == nil {
			t.Errorf("%s: Events returned shared storage", c.name)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-040, FR-053, FR-087, FR-108 - a replacement is accepted at its moment (a Long Rest for the Weapon Mastery pair, a gained level for a Druidic Warrior cantrip, a Short or Long Rest for the Hunter options) and returns the RangerChoiceReplaced event.
// Path: per case (happy).
func TestChooseRangerOptionReplaced(t *testing.T) {
	t.Parallel()
	druid := rgrState(t, character.RangerStateParts{FightingStyle: character.DruidicWarrior{Cantrips: [2]content.SpellID{"guidance", "starry_wisp"}}, HuntersPrey: "colossus_slayer"})
	withPair := must(character.WithFeatureState(rgrWith(t, rgrRanger(t, 7), druid), character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(choiceOf(t, "longbow", "shortsword"))))
	for _, c := range []struct {
		name   string
		choice character.RangerChoice
		feat   string
	}{
		{"Weapon Mastery", character.RangerReplaceWeaponMasteries{Swaps: []character.MasterySwap{{Remove: "longbow", Add: "scimitar"}}, Moment: character.RangerFinishedLongRest, Weapons: fgtWeapons(t, "scimitar"), Proficiencies: rgrProficient()}, "weapon_mastery"},
		{"Druidic Warrior cantrip", character.RangerReplaceDruidicCantrip{Old: "guidance", New: rgrCantrip(t, "druidcraft", "druid"), Moment: character.RangerGainedLevel}, "fighting_style"},
		{"Hunter’s Prey", character.RangerReplaceHuntersPrey{Option: "horde_breaker", Moment: character.RangerFinishedShortRest}, "hunters_prey"},
	} {
		res, err := chooseRanger(t).Run(context.Background(), character.ChooseRangerOptionCommand{Character: withPair, Subclass: hunter, Skills: rgrSkills(t), Choice: c.choice})
		a, ok := res.(character.ChooseRangerOptionAccepted)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.name, res, err)
		}
		if e, isReplaced := a.Events()[0].(character.RangerChoiceReplaced); !isReplaced || e.Choice() != c.feat {
			t.Errorf("%s: events %#v", c.name, a.Events())
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-103, FR-105 - a rejected choice reports every reason in declaration order, records the first as the one rejection with outcome failure and no error status, leaves the Character unchanged and emits no event; NotARanger for a Fighter records not_a_ranger.
// Path: per case (unhappy: invalid input, state error, boundary).
func TestChooseRangerOptionRejected(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		char   character.Character
		choice character.RangerChoice
		want   []string
	}{
		{"Deft Explorer at level 1", rgrRanger(t, 1), character.RangerChooseDeftExplorer{Skill: sStealth, Languages: []content.Language{lElvish, lDwarvish}}, []string{"feature_not_available"}},
		{"another category", rgrRanger(t, 2), character.RangerChooseFightingStyleFeat{Feat: alertFeat()}, []string{"invalid_choice"}},
		{"unknown option", rgrRanger(t, 3), character.RangerChooseHuntersPrey{Option: "x"}, []string{"unknown_option"}},
		{"replace with none held", rgrRanger(t, 3), character.RangerReplaceHuntersPrey{Option: "horde_breaker", Moment: character.RangerFinishedLongRest}, []string{"nothing_to_replace"}},
		{"nothing to replace and no Hunter", rgrRanger(t, 3), character.RangerReplaceDruidicCantrip{Old: "guidance", New: rgrCantrip(t, "x", "druid"), Moment: character.RangerFinishedShortRest}, []string{"nothing_to_replace"}},
	} {
		before := c.char
		sub := hunter
		if c.name == "Deft Explorer at level 1" {
			sub = ""
		}
		res, err := chooseRanger(t).Run(context.Background(), character.ChooseRangerOptionCommand{Character: c.char, Subclass: sub, Skills: rgrSkills(t), Choice: c.choice})
		rj, ok := res.(character.ChooseRangerOptionRejected)
		if err != nil || !ok || !slices.Equal(rgrReasons(rj.Reasons()), c.want) || !c.char.Equal(before) {
			t.Errorf("%s: result = %#v, %v; want reasons %v", c.name, res, err, c.want)
		}
		if len(rj.Reasons()) > 0 {
			rj.Reasons()[0] = 0
			if rj.Reasons()[0] == 0 {
				t.Errorf("%s: Reasons returned shared storage", c.name)
			}
		}
	}
	res, err := chooseRanger(t).Run(context.Background(), character.ChooseRangerOptionCommand{Character: fgtFighter(t, 5), Choice: character.RangerChooseHuntersPrey{Option: "colossus_slayer"}})
	if _, ok := res.(character.ChooseRangerOptionNotARanger); err != nil || !ok {
		t.Errorf("a Fighter = %#v, %v", res, err)
	}
}

// Requirement: doc/specs/class-ranger.md FR-104, FR-105, FR-110 - ChooseRangerOption records one span named srd5e.character.choose_ranger_option with the class ranger and the feature chosen, one feature_used increment and one duration; a rejection counts the first reason with no error
// status; NotARanger counts not_a_ranger; a foreign state is an error recorded on the span only; the result is identical with the no-op and with a recording telemetry.
// Path: per case (happy / unhappy: telemetry on failure, determinism).
func TestChooseRangerOptionSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	good := character.ChooseRangerOptionCommand{Character: rgrRanger(t, 3), Subclass: hunter, Choice: character.RangerChooseHuntersPrey{Option: "colossus_slayer"}}
	rec, opts := fgtTelemetry()
	withRec, err := chooseRanger(t, opts...).Run(ctx, good)
	plain, err2 := chooseRanger(t).Run(ctx, good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, rgrWant(character.SpanChooseRangerOption, "hunters_prey", ""))
	rec, opts = fgtTelemetry()
	bad := good
	bad.Choice = character.RangerChooseHuntersPrey{Option: "x"}
	if _, err := chooseRanger(t, opts...).Run(ctx, bad); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, rgrWant(character.SpanChooseRangerOption, "hunters_prey", "unknown_option"))
	rec, opts = fgtTelemetry()
	if _, err := chooseRanger(t, opts...).Run(ctx, character.ChooseRangerOptionCommand{Character: fgtFighter(t, 5), Choice: character.RangerChooseExpertise{}}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, rgrWant(character.SpanChooseRangerOption, "expertise", "not_a_ranger"))
	rec, opts = fgtTelemetry()
	foreign := good
	foreign.Character = rgrForeign(t, rgrRanger(t, 3))
	res, err := chooseRanger(t, opts...).Run(ctx, foreign)
	spans := rec.Tracer.Spans()
	wantInvalid(t, err, "foreign state")
	if res != nil || len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("foreign state recorded %+v", spans)
	}
	foreignMastery := good
	foreignMastery.Character = must(character.WithFeatureState(rgrRanger(t, 3), character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("hunter"))))
	rec, opts = fgtTelemetry()
	res, err = chooseRanger(t, opts...).Run(ctx, foreignMastery)
	wantInvalid(t, err, "foreign mastery state")
	if res != nil || len(rec.Tracer.Spans()) != 1 || rec.Tracer.Spans()[0].Status != telemetry.StatusError {
		t.Errorf("foreign mastery state recorded %+v", rec.Tracer.Spans())
	}
}

// Requirement: doc/specs/class-ranger.md FR-104 - the constructor refuses the zero progression; Run refuses the zero use case, a zero Character and a nil or unknown choice as invalid input, and a cancelled context, each before any signal.
// Path: per case (unhappy: invalid input, cancellation).
func TestChooseRangerOptionInvalidInput(t *testing.T) {
	t.Parallel()
	_, err := character.NewChooseRangerOption(character.RangerProgression{})
	wantInvalid(t, err, "zero progression")
	good := character.ChooseRangerOptionCommand{Character: rgrRanger(t, 3), Choice: character.RangerChooseHuntersPrey{Option: "colossus_slayer"}}
	_, err = character.ChooseRangerOption{}.Run(context.Background(), good)
	wantInvalid(t, err, "zero use case")
	rec, opts := fgtTelemetry()
	_, err = chooseRanger(t, opts...).Run(context.Background(), character.ChooseRangerOptionCommand{Choice: good.Choice})
	wantInvalid(t, err, "zero character")
	nilChoice := good
	nilChoice.Choice = nil
	_, err = chooseRanger(t, opts...).Run(context.Background(), nilChoice)
	wantInvalid(t, err, "nil choice")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := chooseRanger(t, opts...).Run(cancelled, good)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec.Tracer.Spans()))
	}
	zeroFeat := good
	zeroFeat.Choice = character.RangerChooseFeat{}
	zeroFeat.Character = rgrRanger(t, 4)
	_, err = chooseRanger(t, opts...).Run(context.Background(), zeroFeat)
	wantInvalid(t, err, "a zero feat")
	if len(rec.Tracer.Spans()) != 1 || rec.Tracer.Spans()[0].Status != telemetry.StatusError {
		t.Errorf("a zero feat recorded %+v", rec.Tracer.Spans())
	}
}
