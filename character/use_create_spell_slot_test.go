package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-sorcerer.md FR-052 to FR-054, FR-114, FR-099 (SRD 5.2.1, "Creating Spell Slots", p. 66) - CreateSpellSlot reads the cost and minimum level from the class definition's table, spends the points and adds one created
// slot to the new Character with a SpellSlotCreated event, leaving its input unchanged; a refusal is a result carrying the rejection, and a slot level outside 1 to 9 is an error recorded on the span only.
// Path: per case (happy / unhappy: boundary, invalid input).
func TestCreateSpellSlotRun(t *testing.T) {
	t.Parallel()
	uc := must(character.NewCreateSpellSlot(sorSource(t)))
	c := sorChar(t, 7, 16, "", sorWith(sorProfile(7, ""), 7, nil, nil))
	res := must(uc.Run(t.Context(), character.CreateSpellSlotCommand{Character: c, SlotLevel: 4})).(character.CreateSpellSlotApplied)
	got := sorStateOf(t, res.Character())
	if got.SorceryPoints() != 1 || got.CreatedSlots(4) != 1 || sorStateOf(t, c).SorceryPoints() != 7 {
		t.Errorf("state = %+v (input points %d), want 1 point and one created level 4 slot", got.Parts(), sorStateOf(t, c).SorceryPoints())
	}
	if ev, ok := res.Events()[0].(character.SpellSlotCreated); len(res.Events()) != 1 || !ok || ev.SlotLevel() != 4 || ev.Cost() != 6 {
		t.Errorf("events = %#v, want a SpellSlotCreated of level 4 costing 6", res.Events())
	}
	for _, rc := range []struct {
		name     string
		level    int
		points   int
		slot     int
		kind     character.SorcererRejectionKind
		required int
	}{
		{"level 6 is not creatable", 20, 20, 6, character.SorcererSlotLevelNotCreatable, 0},
		{"below the minimum level", 6, 6, 4, character.SorcererLevelBelowMinimum, 7},
		{"too few points", 5, 4, 3, character.SorcererSorceryPointsInsufficient, 5},
		{"no Font of Magic at level 1", 1, 0, 1, character.SorcererFeatureNotGained, 0},
	} {
		sc := sorChar(t, rc.level, 16, "", sorWith(sorProfile(rc.level, ""), rc.points, nil, nil))
		if rj, ok := must(uc.Run(t.Context(), character.CreateSpellSlotCommand{Character: sc, SlotLevel: rc.slot})).(character.CreateSpellSlotRejected); !ok || rj.Rejection.Kind != rc.kind || rj.Rejection.Required != rc.required {
			t.Errorf("%s = %#v, want %v requiring %d", rc.name, rj, rc.kind, rc.required)
		}
	}
	rec, opts := sorTelemetry()
	for _, level := range []int{0, 10} {
		if _, err := must(character.NewCreateSpellSlot(sorSource(t), opts...)).Run(t.Context(), character.CreateSpellSlotCommand{Character: c, SlotLevel: level}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("level %d err = %v, want ErrInvalidInput", level, err)
		}
	}
	for _, s := range rec.Tracer.SpansNamed(character.SpanCreateSpellSlot) {
		if s.Status != telemetry.StatusError || !s.Ended {
			t.Errorf("an invalid slot level must end its span with error status: %+v", s)
		}
	}
	if len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Error("an invalid slot level must not count as a feature use")
	}
}
