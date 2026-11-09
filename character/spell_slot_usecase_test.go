package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

func opSignals(span, op, counter string, result, outcomeValue string, extra map[string]any) wantSignals {
	return wantSignals{span: span, operation: op, counter: counter,
		spanAttrs:    attrs(res(result, outcomeValue), map[string]any{character.AttrOperation: op}, extra),
		counterAttrs: attrs(res(result, outcomeValue), map[string]any{character.AttrOperation: op}, extra)}
}

// Requirement: doc/specs/spellcasting.md FR-109, FR-084 to FR-088, Tables Q1, Q2, Q3, Q4 - ExpendSpellSlot gives the
// same outcomes as the pure transition for every row (Q4: no one-slot-per-turn rule), records the slot level
// on spell_slot_level, one operations increment with the pool and, when rejected, no_slot_available; results
// are equal with Noop and recording telemetry.
// Path: per case.
func TestExpendSpellSlotUseCase(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	wiz5 := castingState(t, classLevelsOf(t, "wizard", 5), spells)
	lock5 := castingState(t, classLevelsOf(t, "warlock", 5), spells)
	empty := wiz5
	for range 3 {
		empty = empty.ExpendSlot(character.PoolSpellcasting, slotLevel(2)).(character.SlotExpenditureExpended).State()
	}
	cases := []struct {
		name    string
		state   character.SpellcastingState
		pool    character.SpellSlotPool
		level   int
		result  string
		signals map[string]any
	}{
		{"Q1", wiz5, character.PoolSpellcasting, 2, "expended", map[string]any{character.AttrPool: "spellcasting"}},
		{"Q3", lock5, character.PoolPactMagic, 3, "expended", map[string]any{character.AttrPool: "pact_magic"}},
		{"Q2", empty, character.PoolSpellcasting, 2, "rejected", map[string]any{character.AttrPool: "spellcasting", character.AttrRejection: "no_slot_available"}},
		{"Q8", wiz5, character.PoolPactMagic, 1, "rejected", map[string]any{character.AttrPool: "pact_magic", character.AttrRejection: "no_slot_available"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ExpendSpellSlotCommand{State: c.state, Pool: c.pool, Level: slotLevel(c.level)}
			rec, opts := recordedCasting()
			got, err := character.NewExpendSpellSlot(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewExpendSpellSlot().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v (%v, %v), plain %#v", got, err, err2, plain)
			}
			outcomeValue := "success"
			switch r := got.(type) {
			case character.ExpendSpellSlotExpended:
				if c.result != "expended" || r.Event().Level() != slotLevel(c.level) || r.Event().Pool() != c.pool {
					t.Errorf("result = %#v", got)
				}
				direct := c.state.ExpendSlot(c.pool, slotLevel(c.level)).(character.SlotExpenditureExpended)
				if !reflect.DeepEqual(r.State(), direct.State()) || r.Event() != direct.Event() {
					t.Error("the use case and the transition differ")
				}
			case character.ExpendSpellSlotRejected:
				outcomeValue = "failure"
				if c.result != "rejected" || r.Reason() != character.SlotOperationNoSlotAvailable {
					t.Errorf("result = %#v", got)
				}
			}
			sig := opSignals(character.SpanExpendSpellSlot, "expend_spell_slot", character.MetricOperations, c.result, outcomeValue, c.signals)
			checkSignals(t, rec, sig)
			levels := rec.Meter.MeasurementsNamed(character.MetricSpellSlotLevel)
			switch {
			case c.result == "expended" && (len(levels) != 1 || levels[0].Value != float64(c.level)):
				t.Errorf("spell_slot_level = %+v, want one value %d", levels, c.level)
			case c.result == "expended":
				sameAttrs(t, "spell_slot_level", levels[0].Attrs, map[string]any{character.AttrPool: c.signals[character.AttrPool]})
			case len(levels) != 0:
				t.Errorf("a rejected expenditure recorded a slot level: %+v", levels)
			}
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-110, Tables Q5, Q6, Q7, Q9 - RegainSpellSlot restores up to the count
// and records regained, or a rejection no_expended_slot; equal results with Noop and recording telemetry.
// Path: per case.
func TestRegainSpellSlotUseCase(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	wiz5 := castingState(t, classLevelsOf(t, "wizard", 5), spells)
	two := wiz5
	for range 2 {
		two = two.ExpendSlot(character.PoolSpellcasting, slotLevel(1)).(character.SlotExpenditureExpended).State()
	}
	cases := []struct {
		name         string
		state        character.SpellcastingState
		pool         character.SpellSlotPool
		count        int
		restored     int
		rejection    string
		poolAttrName string
	}{
		{"Q5", two, character.PoolSpellcasting, 1, 1, "", "spellcasting"}, {"Q6", two, character.PoolSpellcasting, 3, 2, "", "spellcasting"},
		{"Q7", wiz5, character.PoolSpellcasting, 1, 0, "no_expended_slot", "spellcasting"}, {"Q9", wiz5, character.PoolPactMagic, 1, 0, "no_expended_slot", "pact_magic"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.RegainSpellSlotCommand{State: c.state, Pool: c.pool, Level: slotLevel(1), Count: c.count}
			rec, opts := recordedCasting()
			got, err := character.NewRegainSpellSlot(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewRegainSpellSlot().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v (%v, %v), plain %#v", got, err, err2, plain)
			}
			result, outcomeValue := "regained", "success"
			extra := map[string]any{character.AttrPool: c.poolAttrName}
			switch r := got.(type) {
			case character.RegainSpellSlotRegained:
				if c.restored == 0 || r.Event().Spellcasting() != c.restored {
					t.Errorf("restored %d, want %d", r.Event().Spellcasting(), c.restored)
				}
			case character.RegainSpellSlotRejected:
				result, outcomeValue, extra[character.AttrRejection] = "rejected", "failure", c.rejection
				if c.restored != 0 || r.Reason() != character.SlotOperationNoExpendedSlot {
					t.Errorf("result = %#v", got)
				}
			}
			checkSignals(t, rec, opSignals(character.SpanRegainSpellSlot, "regain_spell_slot", character.MetricOperations, result, outcomeValue, extra))
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-025 to FR-027 - RestoreSpellSlots records one spell_slots_restored
// increment with the rest kind for a rest, also one that restores nothing, and returns the transition's state and
// events.
// Path: happy.
func TestRestoreSpellSlotsUseCase(t *testing.T) {
	t.Parallel()
	st := castingState(t, classLevelsOf(t, "warlock", 5, "wizard", 3), spellCatalogue(t))
	st = st.ExpendSlot(character.PoolPactMagic, slotLevel(3)).(character.SlotExpenditureExpended).State()
	for kind, events := range map[character.RestKind]int{character.ShortRest: 1, character.LongRest: 1} {
		cmd := character.RestoreSpellSlotsCommand{State: st, Rest: kind}
		rec, opts := recordedCasting()
		got, err := character.NewRestoreSpellSlots(opts...).Do(context.Background(), cmd)
		plain, _ := character.NewRestoreSpellSlots().Do(context.Background(), cmd)
		if err != nil || !reflect.DeepEqual(got, plain) {
			t.Fatalf("%v: %#v %v", kind, got, err)
		}
		r := got.(character.RestoreSpellSlotsRestored)
		if len(r.Events()) != events || remaining(r.State(), character.PoolPactMagic, 3) != 2 {
			t.Errorf("%v: events %v", kind, r.Events())
		}
		extra := map[string]any{character.AttrRestKind: kind.String()}
		checkSignals(t, rec, wantSignals{span: character.SpanRestoreSpellSlots, operation: "restore_spell_slots", counter: character.MetricSpellSlotsRestored,
			spanAttrs: attrs(res("restored", "success"), map[string]any{character.AttrOperation: "restore_spell_slots"}, extra), counterAttrs: attrs(res("restored", "success"), extra)})
	}
	full, _ := character.NewRestoreSpellSlots().Do(context.Background(), character.RestoreSpellSlotsCommand{State: castingState(t, classLevelsOf(t, "wizard", 1), spellCatalogue(t)), Rest: character.LongRest})
	if len(full.(character.RestoreSpellSlotsRestored).Events()) != 0 {
		t.Error("a rest with nothing expended raised events")
	}
}

// Requirement: doc/specs/spellcasting.md FR-112, FR-029, FR-047, Tables O1, O6 - AdvanceSpellcasting returns the new
// state with the headroom per class, or a rejection recorded as failure with its reason; equal results with Noop and
// recording telemetry.
// Path: per case.
func TestAdvanceSpellcastingUseCase(t *testing.T) {
	t.Parallel()
	st := castingState(t, classLevelsOf(t, "wizard", 4), spellCatalogue(t))
	cases := []struct {
		name      string
		to        character.ClassLevels
		result    string
		rejection string
	}{
		{"O1", classLevelsOf(t, "wizard", 5), "advanced", ""}, {"O6", classLevelsOf(t, "wizard", 4), "rejected", "no_change"},
		{"O9", classLevelsOf(t, "wizard", 6), "rejected", "levels_skipped"}, {"O7", classLevelsOf(t, "wizard", 3), "rejected", "level_lowered"},
		{"O8", classLevelsOf(t, "cleric", 4), "rejected", "class_removed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.AdvanceSpellcastingCommand{State: st, Levels: c.to}
			rec, opts := recordedCasting()
			got, err := character.NewAdvanceSpellcasting(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewAdvanceSpellcasting().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v (%v, %v), plain %#v", got, err, err2, plain)
			}
			extra, outcomeValue := map[string]any{}, "success"
			if c.rejection != "" {
				extra[character.AttrRejection], outcomeValue = c.rejection, "failure"
				if r, ok := got.(character.AdvanceSpellcastingRejected); !ok || r.Reason().String() != c.rejection {
					t.Errorf("result = %#v", got)
				}
			} else if a, ok := got.(character.AdvanceSpellcastingAdvanced); !ok || !a.State().ClassLevels().Equal(c.to) || count(a.SpellsToChoose(), character.CasterWizard) != 9 || count(a.CantripsToChoose(), character.CasterWizard) != 4 || len(a.Events()) != 0 {
				t.Errorf("result = %#v", got)
			}
			checkSignals(t, rec, opSignals(character.SpanAdvanceSpellcasting, "advance_spellcasting", character.MetricOperations, c.result, outcomeValue, extra))
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-087, FR-126 - a zero use case or state, an invalid pool, rest kind or
// level, a count below 1 and a cancelled context are errors before any signal.
// Path: unhappy (invalid input).
func TestSlotUseCasesInvalid(t *testing.T) {
	t.Parallel()
	st := castingState(t, classLevelsOf(t, "wizard", 1), spellCatalogue(t))
	rec, opts := recordedCasting()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	l1 := slotLevel(1)
	calls := map[string]func(context.Context) error{
		"expend zero state": func(c context.Context) error {
			_, e := character.NewExpendSpellSlot(opts...).Do(c, character.ExpendSpellSlotCommand{Pool: character.PoolSpellcasting, Level: l1})
			return e
		},
		"expend invalid pool": func(c context.Context) error {
			_, e := character.NewExpendSpellSlot(opts...).Do(c, character.ExpendSpellSlotCommand{State: st, Level: l1})
			return e
		},
		"expend zero level": func(c context.Context) error {
			_, e := character.NewExpendSpellSlot(opts...).Do(c, character.ExpendSpellSlotCommand{State: st, Pool: character.PoolSpellcasting})
			return e
		},
		"expend zero use case": func(c context.Context) error {
			_, e := (character.ExpendSpellSlot{}).Do(c, character.ExpendSpellSlotCommand{State: st, Pool: character.PoolSpellcasting, Level: l1})
			return e
		},
		"regain zero count": func(c context.Context) error {
			_, e := character.NewRegainSpellSlot(opts...).Do(c, character.RegainSpellSlotCommand{State: st, Pool: character.PoolSpellcasting, Level: l1})
			return e
		},
		"regain zero level": func(c context.Context) error {
			_, e := character.NewRegainSpellSlot(opts...).Do(c, character.RegainSpellSlotCommand{State: st, Pool: character.PoolSpellcasting, Count: 1})
			return e
		},
		"regain invalid pool": func(c context.Context) error {
			_, e := character.NewRegainSpellSlot(opts...).Do(c, character.RegainSpellSlotCommand{State: st, Level: l1, Count: 1})
			return e
		},
		"regain zero state": func(c context.Context) error {
			_, e := character.NewRegainSpellSlot(opts...).Do(c, character.RegainSpellSlotCommand{Pool: character.PoolSpellcasting, Level: l1, Count: 1})
			return e
		},
		"regain zero use case": func(c context.Context) error {
			_, e := (character.RegainSpellSlot{}).Do(c, character.RegainSpellSlotCommand{State: st, Pool: character.PoolSpellcasting, Level: l1, Count: 1})
			return e
		},
		"restore invalid rest": func(c context.Context) error {
			_, e := character.NewRestoreSpellSlots(opts...).Do(c, character.RestoreSpellSlotsCommand{State: st})
			return e
		},
		"restore zero state": func(c context.Context) error {
			_, e := character.NewRestoreSpellSlots(opts...).Do(c, character.RestoreSpellSlotsCommand{Rest: character.LongRest})
			return e
		},
		"restore zero use case": func(c context.Context) error {
			_, e := (character.RestoreSpellSlots{}).Do(c, character.RestoreSpellSlotsCommand{State: st, Rest: character.LongRest})
			return e
		},
		"advance zero levels": func(c context.Context) error {
			_, e := character.NewAdvanceSpellcasting(opts...).Do(c, character.AdvanceSpellcastingCommand{State: st})
			return e
		},
		"advance zero state": func(c context.Context) error {
			_, e := character.NewAdvanceSpellcasting(opts...).Do(c, character.AdvanceSpellcastingCommand{Levels: st.ClassLevels()})
			return e
		},
		"advance zero use case": func(c context.Context) error {
			_, e := (character.AdvanceSpellcasting{}).Do(c, character.AdvanceSpellcastingCommand{State: st, Levels: st.ClassLevels()})
			return e
		},
	}
	for name, call := range calls {
		if err := call(context.Background()); !isInvalid(err) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	for _, c := range []struct {
		name string
		call func(context.Context) error
	}{
		{"expend", func(c context.Context) error {
			_, e := character.NewExpendSpellSlot(opts...).Do(c, character.ExpendSpellSlotCommand{State: st, Pool: character.PoolSpellcasting, Level: l1})
			return e
		}},
		{"regain", func(c context.Context) error {
			_, e := character.NewRegainSpellSlot(opts...).Do(c, character.RegainSpellSlotCommand{State: st, Pool: character.PoolSpellcasting, Level: l1, Count: 1})
			return e
		}},
		{"restore", func(c context.Context) error {
			_, e := character.NewRestoreSpellSlots(opts...).Do(c, character.RestoreSpellSlotsCommand{State: st, Rest: character.LongRest})
			return e
		}},
		{"advance", func(c context.Context) error {
			_, e := character.NewAdvanceSpellcasting(opts...).Do(c, character.AdvanceSpellcastingCommand{State: st, Levels: st.ClassLevels()})
			return e
		}},
	} {
		if err := c.call(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("%s cancelled: %v", c.name, err)
		}
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("an invalid call emitted signals")
	}
}
