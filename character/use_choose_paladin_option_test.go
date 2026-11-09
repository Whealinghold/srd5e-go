package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func chooseOption(t *testing.T, opts ...character.Option) character.ChoosePaladinOption {
	t.Helper()
	return must(character.NewChoosePaladinOption(pldProgression(t), opts...))
}

// Requirement: doc/specs/class-paladin.md FR-038 to FR-041, FR-105, FR-124 - ChoosePaladinOption takes a Fighting Style feat (the feat events, then PaladinChoiceMade), Blessed Warrior with two Cleric cantrips, and replaces one of them;
// each stores the choice in the PaladinState, leaves the input unchanged and emits the PaladinChoiceMade event with the Character id.
// Path: per case (happy).
func TestChoosePaladinOptionAccepted(t *testing.T) {
	t.Parallel()
	bw := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, FightingStyle: character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}})
	for _, c := range []struct {
		name       string
		char       character.Character
		choice     character.PaladinChoice
		choiceName string
		events     int
		style      character.PaladinFightingStyle
	}{
		{"Fighting Style feat", pldPaladin(t, 2), character.ChoosePaladinFightingStyle{Feat: fgtStyle("archery")}, "fighting_style_feat", 2, character.PaladinFightingStyleFeat{Feat: "archery"}},
		{"Blessed Warrior", pldPaladin(t, 2), character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{pldGuidance(t), pldFlame(t)}}, "blessed_warrior", 1,
			character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}},
		{"replace a cantrip", pldWith(t, pldPaladin(t, 5), bw), character.ReplaceBlessedWarriorCantrip{Old: "guidance", New: pldMending(t)}, "blessed_warrior_cantrip_replaced", 1,
			character.BlessedWarrior{Cantrips: [2]content.SpellID{"mending", "sacred_flame"}}},
	} {
		before := c.char
		res, err := chooseOption(t).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: c.char, Choice: c.choice})
		a, ok := res.(character.ChoosePaladinOptionAccepted)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.name, res, err)
		}
		style, _ := pldStateOf(t, a.Character()).FightingStyle()
		last, isChoice := a.Events()[len(a.Events())-1].(character.PaladinChoiceMade)
		if style != c.style || !a.State().Equal(pldStateOf(t, a.Character())) || len(a.Events()) != c.events || !isChoice || last.Choice() != c.choiceName || last.CharacterID() != c.char.ID() || !c.char.Equal(before) {
			t.Errorf("%s: style %v, events %#v", c.name, style, a.Events())
		}
		if c.name == "Fighting Style feat" && a.Feats().Len() != 1 {
			t.Errorf("%s: feats = %v", c.name, a.Feats().All())
		}
		if c.name != "Fighting Style feat" && a.Feats().Len() != 0 {
			t.Errorf("%s: feats = %v", c.name, a.Feats().All())
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-038 to FR-041, FR-105, FR-106 - a rejected choice reports every reason in declaration order, leaves the Character and the feats unchanged and emits no event; NotAPaladin for a Fighter.
// Path: per case (unhappy: invalid input, state error, boundary).
func TestChoosePaladinOptionRejected(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		char   character.Character
		choice character.PaladinChoice
		want   []string
	}{
		{"style at level 1", pldPaladin(t, 1), character.ChoosePaladinFightingStyle{Feat: fgtStyle("archery")}, []string{"level_too_low"}},
		{"another category", pldPaladin(t, 2), character.ChoosePaladinFightingStyle{Feat: alertFeat()}, []string{"not_a_fighting_style_feat"}},
		{"one cantrip", pldPaladin(t, 2), character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{pldGuidance(t)}}, []string{"wrong_count"}},
		{"replace without Blessed Warrior", pldPaladin(t, 5), character.ReplaceBlessedWarriorCantrip{Old: "guidance", New: pldMending(t)}, []string{"not_blessed_warrior"}},
	} {
		before := c.char
		res, err := chooseOption(t).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: c.char, Choice: c.choice})
		rj, ok := res.(character.ChoosePaladinOptionRejected)
		if err != nil || !ok || !slices.Equal(pldReasons(rj.Reasons()), c.want) || !c.char.Equal(before) {
			t.Errorf("%s: result = %#v, %v; want reasons %v", c.name, res, err, c.want)
		}
	}
	if res, err := chooseOption(t).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: fgtFighter(t, 5), Choice: character.ChooseBlessedWarriorCantrips{}}); err != nil {
		t.Fatal(err)
	} else if _, ok := res.(character.ChoosePaladinOptionNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-109, FR-115 - the call records one choose_option span with success or the first reason, and the zero use case, a zero Character, a nil or unknown choice, a zero feat, a foreign
// state and a cancelled context are errors matching srderr.ErrInvalidInput (or the context error).
// Path: per case (unhappy: invalid input, telemetry on rejection).
func TestChoosePaladinOptionSignalsAndEdges(t *testing.T) {
	t.Parallel()
	rec, opts := fgtTelemetry()
	if _, err := chooseOption(t, opts...).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: pldPaladin(t, 2), Choice: character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{pldGuidance(t), pldFlame(t)}}}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanChoosePaladinOption, "choose_option", ""))
	rec, opts = fgtTelemetry()
	if _, err := chooseOption(t, opts...).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: pldPaladin(t, 1), Choice: character.ChooseBlessedWarriorCantrips{}}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanChoosePaladinOption, "choose_option", "level_too_low"))
	rec, opts = fgtTelemetry()
	if _, err := chooseOption(t, opts...).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: fgtFighter(t, 5), Choice: character.ChooseBlessedWarriorCantrips{}}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanChoosePaladinOption, "choose_option", "not_a_paladin"))
	good := character.ChooseBlessedWarriorCantrips{Cantrips: []content.Spell{pldGuidance(t), pldFlame(t)}}
	for name, cmd := range map[string]character.ChoosePaladinOptionCommand{
		"zero character": {Choice: good}, "nil choice": {Character: pldPaladin(t, 2)},
		"zero feat":     {Character: pldPaladin(t, 2), Choice: character.ChoosePaladinFightingStyle{}},
		"foreign state": {Character: pldForeign(t, pldPaladin(t, 2)), Choice: good},
	} {
		if res, err := chooseOption(t).Run(context.Background(), cmd); res != nil || err == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		} else {
			wantInvalid(t, err, name)
		}
	}
	if _, err := (character.ChoosePaladinOption{}).Run(context.Background(), character.ChoosePaladinOptionCommand{Character: pldPaladin(t, 2), Choice: good}); err == nil {
		t.Error("the zero use case ran")
	}
	if _, err := character.NewChoosePaladinOption(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := chooseOption(t).Run(ctx, character.ChoosePaladinOptionCommand{Character: pldPaladin(t, 2), Choice: good}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
