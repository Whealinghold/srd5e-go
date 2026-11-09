package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var (
	featureSubject = character.NewConcentrationSubjectFeature(content.MustFeatureID("holding_on"))
	spellSubject   = func(id content.SpellID) character.ConcentrationSubject {
		return character.NewConcentrationSubjectSpell(id)
	}
)

// beginOn begins Concentration on subject in s through the use case.
func beginOn(t *testing.T, s character.SpellcastingState, subject character.ConcentrationSubject) character.SpellcastingState {
	t.Helper()
	got, err := character.NewBeginConcentration().Do(context.Background(), character.BeginConcentrationCommand{State: s, Subject: subject})
	switch r := got.(type) {
	case character.BeginConcentrationBegan:
		return r.State()
	case character.BeginConcentrationCastingFailed:
		return r.State()
	}
	t.Fatalf("begin = %#v, %v", got, err)
	return s
}

func endReasons(events []character.CastingEvent) []character.ConcentrationEndReason {
	var out []character.ConcentrationEndReason
	for _, e := range events {
		switch e := e.(type) {
		case character.ConcentrationEnded:
			out = append(out, e.Reason())
		case character.CastingFailed:
			out = append(out, e.Reason())
		}
	}
	return out
}

// Requirement: doc/specs/casting-and-concentration.md FR-072, FR-076, FR-077, FR-099, FR-119, FR-128, Table N (N10 to N13,
// N16 to N18, N22, N25 to N28) - ConcentrationRules.BreakConcentration ends Concentration, fails a casting in progress with no
// slot spent, and with nothing to end returns the state unchanged and no events.
// Path: per case.
func TestConcentrationRulesBreak(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	onSpell := concentratingWizard(t)
	onFeature := beginOn(t, wiz, featureSubject)
	casting := castingStarted(t, wiz)
	cases := []struct {
		name, req, path string
		state           character.SpellcastingState
		reason          character.ConcentrationBreakReason
		events          []string
		reasons         []character.ConcentrationEndReason
		failed          bool
	}{
		{"N10 incapacitated while casting", "FR-072 FR-083", "unhappy (state error)", casting, character.BreakIncapacitated, []string{"ConcentrationEnded", "CastingFailed"}, []character.ConcentrationEndReason{character.EndedIncapacitated, character.EndedIncapacitated}, true},
		{"N26 dead while casting", "FR-128 FR-072", "unhappy (state error)", casting, character.BreakDead, []string{"ConcentrationEnded", "CastingFailed"}, []character.ConcentrationEndReason{character.EndedDead, character.EndedDead}, true},
		{"N13 ended while casting", "FR-072 FR-077", "unhappy (state error)", casting, character.BreakEnded, []string{"ConcentrationEnded", "CastingFailed"}, []character.ConcentrationEndReason{character.EndedByCaster, character.EndedByCaster}, true},
		{"N17 another effect while casting", "FR-119 FR-072", "unhappy (state error)", casting, character.BreakAnotherEffect, []string{"ConcentrationEnded", "CastingFailed"}, []character.ConcentrationEndReason{character.EndedAnotherEffect, character.EndedAnotherEffect}, true},
		{"N11 failed save on a spell", "FR-099", "unhappy (state error)", onSpell, character.BreakFailedSave, []string{"ConcentrationEnded"}, []character.ConcentrationEndReason{character.EndedFailedSave}, false},
		{"N25 failed save, effect on a spell", "FR-128 FR-099", "happy", onSpell, character.BreakFailedSave, []string{"ConcentrationEnded"}, []character.ConcentrationEndReason{character.EndedFailedSave}, false},
		{"N16 another effect while concentrating", "FR-119", "unhappy (rule interaction)", onSpell, character.BreakAnotherEffect, []string{"ConcentrationEnded"}, []character.ConcentrationEndReason{character.EndedAnotherEffect}, false},
		{"N22 ended on a feature", "FR-119 FR-077", "happy", onFeature, character.BreakEnded, []string{"ConcentrationEnded"}, []character.ConcentrationEndReason{character.EndedByCaster}, false},
		{"N12 failed save with none", "FR-099", "unhappy (empty and minimal)", wiz, character.BreakFailedSave, nil, nil, false},
		{"N18 another effect with none", "FR-119", "unhappy (empty and minimal)", wiz, character.BreakAnotherEffect, nil, nil, false},
		{"N27 incapacitated with none", "FR-128 FR-077", "unhappy (empty and minimal)", wiz, character.BreakIncapacitated, nil, nil, false},
		{"FR-128 an invalid reason changes nothing", "FR-128", "unhappy (invalid input)", onSpell, 0, nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			tr := character.ConcentrationRules{}.BreakConcentration(c.state, c.reason)
			if !slices.Equal(eventNames(tr.Events()), c.events) || !slices.Equal(endReasons(tr.Events()), c.reasons) {
				t.Errorf("events = %v %v, want %v %v [%s]", eventNames(tr.Events()), endReasons(tr.Events()), c.events, c.reasons, c.req)
			}
			failed, hasFailed := tr.CastingFailed()
			if hasFailed != c.failed {
				t.Fatalf("CastingFailed present = %v, want %v", hasFailed, c.failed)
			}
			if c.failed {
				if tr.Events()[1] != character.CastingEvent(failed) || failed.Spell() != preparedL1 {
					t.Errorf("CastingFailed = %+v, events %v [FR-128]", failed, tr.Events())
				}
				if _, going := tr.State().CastingInProgress(); going {
					t.Error("the casting is still in progress")
				}
				if got := tr.State().Slots(character.PoolSpellcasting); got != before.Slots(character.PoolSpellcasting) {
					t.Error("a slot was spent [FR-072]")
				}
			}
			if len(c.events) == 0 {
				if !reflect.DeepEqual(tr.State(), before) {
					t.Error("the state changed with nothing to end")
				}
			} else if _, ok := tr.State().Concentration(); ok {
				t.Error("Concentration remains")
			}
			if !reflect.DeepEqual(c.state, before) {
				t.Error("the input state changed [FR-128]")
			}
		})
	}
	t.Run("N28 events cannot be mutated through the transition", func(t *testing.T) {
		t.Parallel()
		tr := character.ConcentrationRules{}.BreakConcentration(casting, character.BreakDead)
		ev := tr.Events()
		ev[0], ev[1] = ev[1], ev[0]
		if got := eventNames(tr.Events()); !slices.Equal(got, []string{"ConcentrationEnded", "CastingFailed"}) {
			t.Errorf("events = %v after a caller changed its copy [FR-128]", got)
		}
	})
	t.Run("FR-128 the ended event names the subject and the character", func(t *testing.T) {
		t.Parallel()
		e := character.ConcentrationRules{}.BreakConcentration(onSpell, character.BreakDead).Events()[0].(character.ConcentrationEnded)
		if e.Subject() != spellSubject(spellID(1, 3)) || e.CharacterID() != caster {
			t.Errorf("event = %+v", e)
		}
	})
}

// Requirement: doc/specs/casting-and-concentration.md FR-145, Table U (U12 to U15) - CounterCast ends an earlier
// Concentration only for a Concentration spell, spends nothing, begins nothing, and emits CastingCountered last; a casting
// in progress is an error.
// Path: per case.
func TestConcentrationRulesCounterCast(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	onSpell := concentratingWizard(t)
	conc, plain := castSpell(t, spellID(1, 4), 1, concentrating()), castSpell(t, spellID(1, 4), 1)
	cases := []struct {
		name, req, path string
		state           character.SpellcastingState
		spell           content.Spell
		events          []string
		keepsConc       bool
	}{
		{"U12 Concentration spell, concentrating on another", "FR-145", "unhappy (rule interaction)", onSpell, conc, []string{"ConcentrationEnded", "CastingCountered"}, false},
		{"U13 non-Concentration spell, concentrating", "FR-145", "unhappy (rule interaction)", onSpell, plain, []string{"CastingCountered"}, true},
		{"U14 Concentration spell, none in progress", "FR-145", "unhappy (empty and minimal)", wiz, conc, []string{"CastingCountered"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			tr, err := character.ConcentrationRules{}.CounterCast(c.state, c.spell)
			if err != nil || !slices.Equal(eventNames(tr.Events()), c.events) {
				t.Fatalf("events = %v, %v, want %v [%s]", eventNames(tr.Events()), err, c.events, c.req)
			}
			if _, ok := tr.State().Concentration(); ok != c.keepsConc {
				t.Errorf("Concentration kept = %v, want %v", ok, c.keepsConc)
			}
			if tr.State().Slots(character.PoolSpellcasting) != before.Slots(character.PoolSpellcasting) {
				t.Error("a slot changed [FR-145]")
			}
			last := tr.Events()[len(tr.Events())-1].(character.CastingCountered)
			if last.Spell() != c.spell.ID() || last.CharacterID() != caster {
				t.Errorf("CastingCountered = %+v", last)
			}
			if e, ok := tr.Events()[0].(character.ConcentrationEnded); ok && e.Reason() != character.EndedAnotherEffect {
				t.Errorf("reason = %v, want another_effect", e.Reason())
			}
			if !reflect.DeepEqual(c.state, before) {
				t.Error("the input state changed")
			}
		})
	}
	t.Run("U15 a casting in progress, a zero state, a zero spell", func(t *testing.T) {
		t.Parallel()
		for name, c := range map[string]struct {
			s character.SpellcastingState
			p content.Spell
		}{"casting": {castingStarted(t, wiz), conc}, "zero state": {character.SpellcastingState{}, conc}, "zero spell": {wiz, content.Spell{}}} {
			if _, err := (character.ConcentrationRules{}).CounterCast(c.s, c.p); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want ErrInvalidInput [FR-145 FR-102]", name, err)
			}
		}
	})
}

// Requirement: doc/specs/casting-and-concentration.md FR-118, FR-120, FR-148, Table N (N19 to N21) and Table U (U16, U17) -
// BeginConcentration ends any Concentration (another_effect), fails a casting in progress with no slot spent, and then
// begins Concentration of kind effect on the feature or the held spell.
// Path: per case.
func TestBeginConcentration(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	onSpell := concentratingWizard(t)
	casting := castingStarted(t, wiz)
	held := spellSubject(spellID(1, 6))
	cases := []struct {
		name, req, path string
		state           character.SpellcastingState
		subject         character.ConcentrationSubject
		events          []string
		failed          bool
	}{
		{"N19 feature, none in progress", "FR-118 FR-120", "happy", wiz, featureSubject, []string{"ConcentrationBegan"}, false},
		{"N20 feature while concentrating on a spell", "FR-120", "unhappy (rule interaction)", onSpell, featureSubject, []string{"ConcentrationEnded", "ConcentrationBegan"}, false},
		{"N21 feature while casting", "FR-120 FR-072 FR-083", "unhappy (state error)", casting, featureSubject, []string{"ConcentrationEnded", "CastingFailed", "ConcentrationBegan"}, true},
		{"U16 held spell while concentrating on a spell", "FR-148", "unhappy (rule interaction)", onSpell, held, []string{"ConcentrationEnded", "ConcentrationBegan"}, false},
		{"U17 held spell while casting", "FR-148 FR-120", "happy", casting, held, []string{"ConcentrationEnded", "CastingFailed", "ConcentrationBegan"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			got, err := character.NewBeginConcentration().Do(context.Background(), character.BeginConcentrationCommand{State: c.state, Subject: c.subject})
			var state character.SpellcastingState
			var events []character.CastingEvent
			switch r := got.(type) {
			case character.BeginConcentrationBegan:
				state, events = r.State(), r.Events()
				if c.failed {
					t.Fatalf("result = %#v, want CastingFailed", got)
				}
			case character.BeginConcentrationCastingFailed:
				state, events = r.State(), r.Events()
				if !c.failed || r.CastingFailed().Spell() != preparedL1 || r.CastingFailed().Reason() != character.EndedAnotherEffect || events[1] != character.CastingEvent(r.CastingFailed()) {
					t.Fatalf("CastingFailed = %+v [FR-072]", r.CastingFailed())
				}
			default:
				t.Fatalf("result = %#v, %v", got, err)
			}
			if !slices.Equal(eventNames(events), c.events) {
				t.Errorf("events = %v, want %v [%s]", eventNames(events), c.events, c.req)
			}
			conc, _ := state.Concentration()
			last := events[len(events)-1].(character.ConcentrationBegan)
			if conc.Subject() != c.subject || conc.Kind() != character.ConcentrationEffect || last.Subject() != c.subject || last.Kind() != character.ConcentrationEffect {
				t.Errorf("Concentration = %+v, began %+v", conc, last)
			}
			if _, going := state.CastingInProgress(); going || state.Slots(character.PoolSpellcasting) != before.Slots(character.PoolSpellcasting) {
				t.Error("the casting remains or a slot was spent")
			}
			for _, e := range events {
				if e, ok := e.(character.ConcentrationEnded); ok && e.Reason() != character.EndedAnotherEffect {
					t.Errorf("reason = %v, want another_effect", e.Reason())
				}
			}
		})
	}
	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()
		for name, cmd := range map[string]character.BeginConcentrationCommand{
			"nil subject":     {State: wiz},
			"zero state":      {Subject: featureSubject},
			"zero feature id": {State: wiz, Subject: character.NewConcentrationSubjectFeature("")},
			"zero spell id":   {State: wiz, Subject: character.NewConcentrationSubjectSpell("")},
		} {
			if got, err := character.NewBeginConcentration().Do(context.Background(), cmd); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: Do = %#v, %v, want ErrInvalidInput", name, got, err)
			}
		}
		if _, err := (character.BeginConcentration{}).Do(context.Background(), character.BeginConcentrationCommand{State: wiz, Subject: featureSubject}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero use case err = %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := character.NewBeginConcentration().Do(ctx, character.BeginConcentrationCommand{State: wiz, Subject: featureSubject}); !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled err = %v", err)
		}
	})
}

// Requirement: doc/specs/casting-and-concentration.md FR-099, FR-119, FR-128, Table N (N29) - the BreakConcentration use case
// returns exactly the state and events of ConcentrationRules.BreakConcentration, with the variant that fits; an invalid
// reason or state is an error.
// Path: per case.
func TestBreakConcentrationUseCase(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	onSpell := concentratingWizard(t)
	casting := castingStarted(t, wiz)
	for name, c := range map[string]struct {
		state  character.SpellcastingState
		reason character.ConcentrationBreakReason
	}{"ended": {onSpell, character.BreakFailedSave}, "casting failed": {casting, character.BreakIncapacitated}, "none": {wiz, character.BreakEnded}} {
		t.Run("N29 "+name, func(t *testing.T) {
			t.Parallel()
			want := character.ConcentrationRules{}.BreakConcentration(c.state, c.reason)
			got, err := character.NewBreakConcentration().Do(context.Background(), character.BreakConcentrationCommand{State: c.state, Reason: c.reason})
			if err != nil {
				t.Fatal(err)
			}
			switch r := got.(type) {
			case character.BreakConcentrationEnded:
				if name != "ended" || !reflect.DeepEqual(r.State(), want.State()) || !slices.Equal(r.Events(), want.Events()) {
					t.Errorf("result = %#v, want the transition %#v", got, want)
				}
			case character.BreakConcentrationCastingFailed:
				failed, _ := want.CastingFailed()
				if name != "casting failed" || !reflect.DeepEqual(r.State(), want.State()) || !slices.Equal(r.Events(), want.Events()) || r.CastingFailed() != failed {
					t.Errorf("result = %#v, want the transition %#v", got, want)
				}
			case character.BreakConcentrationUnchanged:
				if name != "none" || !reflect.DeepEqual(r.State(), c.state) {
					t.Errorf("result = %#v", got)
				}
			}
		})
	}
	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()
		for name, cmd := range map[string]character.BreakConcentrationCommand{
			"zero reason": {State: wiz}, "unknown reason": {State: wiz, Reason: 9}, "zero state": {Reason: character.BreakEnded},
		} {
			if got, err := character.NewBreakConcentration().Do(context.Background(), cmd); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: Do = %#v, %v, want ErrInvalidInput", name, got, err)
			}
		}
		if _, err := (character.BreakConcentration{}).Do(context.Background(), character.BreakConcentrationCommand{State: wiz, Reason: character.BreakEnded}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero use case err = %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := character.NewBreakConcentration().Do(ctx, character.BreakConcentrationCommand{State: wiz, Reason: character.BreakEnded}); !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled err = %v", err)
		}
	})
}
