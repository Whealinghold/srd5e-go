package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

func castScores(t *testing.T, m map[ability.Ability]int) ability.Scores {
	t.Helper()
	in := map[ability.Ability]ability.Score{}
	for _, a := range ability.All() {
		in[a] = ability.MustScore(10)
	}
	for a, v := range m {
		in[a] = ability.MustScore(v)
	}
	s, err := ability.NewScores(in)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/spellcasting.md FR-001 to FR-008, FR-014 to FR-018, FR-084 to FR-088, FR-116 -
// CalculateSpellcasting returns a summary per spellcasting class in class-level order, each with the
// stats of its own ability, the limits of its own level and the highest preparable level, plus both
// slot pools; a class without Spellcasting is ignored; the result is the same with Noop and recording
// telemetry and for the same input twice; one span, one operations increment and one duration are
// emitted, nothing is logged.
// Path: happy.
func TestCalculateSpellcasting(t *testing.T) {
	t.Parallel()
	levels := classLevelsOf(t, "fighter", 2, "wizard", 5, "cleric", 3)
	cmd := character.CalculateSpellcastingCommand{Levels: levels, Scores: castScores(t, map[ability.Ability]int{ability.Intelligence: 18, ability.Wisdom: 14}), Proficiency: pbOf(10)}
	rec, opts := recordedCasting()
	got, err := character.NewCalculateSpellcasting(opts...).Do(context.Background(), cmd)
	plain, err2 := character.NewCalculateSpellcasting().Do(context.Background(), cmd)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !reflect.DeepEqual(got, plain) {
		t.Error("FR-086: the result differs between Noop and recording telemetry")
	}
	sum := got.(character.CalculateSpellcastingCalculated).Summary()
	cs := sum.Classes()
	if len(cs) != 2 || cs[0].Class() != character.CasterWizard || cs[1].Class() != character.CasterCleric {
		t.Fatalf("classes = %+v, want Wizard then Cleric", cs)
	}
	w, c := cs[0], cs[1]
	if w.Level() != 5 || w.Stats().SaveDC().Value() != 8+4+4 || w.Stats().AttackBonus().Value() != 4+4 || w.CantripLimit() != 4 || w.PreparedLimit() != 9 || w.MaxPreparableLevel() != 3 {
		t.Errorf("wizard summary = %+v", w)
	}
	if c.Level() != 3 || c.Stats().SaveDC().Value() != 8+2+4 || c.Stats().Ability() != ability.Wisdom || c.CantripLimit() != 3 || c.PreparedLimit() != 6 || c.MaxPreparableLevel() != 2 {
		t.Errorf("cleric summary = %+v", c)
	}
	if sum.Pools().Spellcasting().Maximum(slotLevel(3)) != 3 || sum.Pools().PactMagic().HasPactMagic() {
		t.Errorf("pools = %+v (Wizard 5 + Cleric 3 is total level 8)", sum.Pools())
	}
	sum.Classes()[0] = character.ClassSpellcastingSummary{}
	if sum.Classes()[0].Class() != character.CasterWizard {
		t.Error("Classes shares memory")
	}
	again, _ := character.NewCalculateSpellcasting().Do(context.Background(), cmd)
	if !reflect.DeepEqual(again, plain) {
		t.Error("FR-084: the same input gave a different result")
	}
	checkSignals(t, rec, wantSignals{
		span: character.SpanCalculateSpellcasting, operation: character.ValueOperationCalculateSpellcasting, counter: character.MetricOperations,
		spanAttrs:    attrs(res("calculated", "success"), map[string]any{character.AttrOperation: "calculate_spellcasting"}),
		counterAttrs: attrs(res("calculated", "success"), map[string]any{character.AttrOperation: "calculate_spellcasting"}),
	})
}

// Requirement: doc/specs/spellcasting.md FR-008 - a character with only non-casting classes gets an empty summary and no slots.
// Path: unhappy (empty and minimal).
func TestCalculateSpellcastingNoCaster(t *testing.T) {
	t.Parallel()
	got, err := character.NewCalculateSpellcasting().Do(context.Background(), character.CalculateSpellcastingCommand{Levels: classLevelsOf(t, "fighter", 5), Scores: castScores(t, nil), Proficiency: pbOf(5)})
	if err != nil {
		t.Fatal(err)
	}
	if s := got.(character.CalculateSpellcastingCalculated).Summary(); len(s.Classes()) != 0 || s.Pools() != (character.SpellSlotPools{}) {
		t.Errorf("summary = %+v", s)
	}
}

// Requirement: doc/specs/spellcasting.md FR-087 - a zero use case, class levels, scores or Proficiency
// Bonus is an ErrInvalidInput error before any signal; a cancelled context is returned unwrapped.
// Path: unhappy (invalid input).
func TestCalculateSpellcastingInvalid(t *testing.T) {
	t.Parallel()
	good := character.CalculateSpellcastingCommand{Levels: classLevelsOf(t, "wizard", 1), Scores: castScores(t, nil), Proficiency: pbOf(1)}
	rec, opts := recordedCasting()
	uc := character.NewCalculateSpellcasting(opts...)
	for name, cmd := range map[string]character.CalculateSpellcastingCommand{
		"levels": {Scores: good.Scores, Proficiency: good.Proficiency}, "scores": {Levels: good.Levels, Proficiency: good.Proficiency}, "proficiency": {Levels: good.Levels, Scores: good.Scores},
	} {
		if _, err := uc.Do(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("zero %s: %v", name, err)
		}
	}
	if _, err := (character.CalculateSpellcasting{}).Do(context.Background(), good); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Do(ctx, good); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("an invalid call emitted signals")
	}
}
