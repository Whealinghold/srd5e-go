package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func gainInput(tb testing.TB, previous, target int, state character.DruidFeatureState) character.GainDruidLevelInput {
	tb.Helper()
	return character.GainDruidLevelInput{ID: druTestID, Previous: previous, Target: target, State: state, Tables: druTables(tb), Subclass: character.CircleOfTheLandSubclassID}
}

func gained(tb testing.TB, in character.GainDruidLevelInput) character.GainDruidLevelApplied {
	tb.Helper()
	r, err := character.GainDruidLevel(in)
	if err != nil {
		tb.Fatal(err)
	}
	a, ok := r.(character.GainDruidLevelApplied)
	if !ok {
		tb.Fatalf("GainDruidLevel(%d to %d) = %#v, want applied", in.Previous, in.Target, r)
	}
	return a
}

// Requirement: doc/specs/class-druid.md FR-023, FR-024, FR-161 - gaining a Druid level accepts only the next level (none to 1, n to n+1); any other target is
// LevelNotConsecutive, a level above 20 is MaximumLevelReached, and when both hold the first of the order (LevelNotConsecutive) is reported.
// Path: per case (unhappy: boundary, rule interaction).
func TestGainDruidLevelOnlyTheNextLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		previous, target int
		want             character.DruidRejection // 0 for accepted
	}{
		{0, 1, 0}, {1, 2, 0}, {19, 20, 0}, {0, 2, character.DruidLevelNotConsecutive}, {1, 3, character.DruidLevelNotConsecutive}, {5, 5, character.DruidLevelNotConsecutive},
		{5, 4, character.DruidLevelNotConsecutive}, {0, 0, character.DruidLevelNotConsecutive}, {3, -1, character.DruidLevelNotConsecutive},
		{20, 21, character.DruidMaximumLevelReached}, {19, 21, character.DruidLevelNotConsecutive}, {5, 99, character.DruidLevelNotConsecutive},
	} {
		in := gainInput(t, c.previous, c.target, character.DruidFeatureState{})
		if c.previous > 0 {
			in.State = druState(t, c.previous, true, character.DruidStateFields{})
		}
		r, err := character.GainDruidLevel(in)
		if err != nil {
			t.Fatalf("%d to %d: %v", c.previous, c.target, err)
		}
		switch got := r.(type) {
		case character.GainDruidLevelApplied:
			if c.want != 0 {
				t.Errorf("%d to %d accepted, want %v", c.previous, c.target, c.want)
			}
		case character.GainDruidLevelRejected:
			if got.Reason != c.want {
				t.Errorf("%d to %d rejected with %v, want %v", c.previous, c.target, got.Reason, c.want)
			}
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-025, FR-026, FR-027, FR-030, FR-059 (SRD 5.2.1, "Druid Features", p. 41) - the result lists the features granted at exactly the new level, never
// repeating an earlier one, and the Druid-owned pending choices: the Primal Order at level 1, four known forms at level 2, two more at levels 4 and 8, Elemental Fury at level 7;
// gaining level 2 grants Wild Shape with its 2 uses at maximum and later levels raise the remaining uses by the rise of the maximum (levels 6 and 17); the event carries the same.
// Path: per case (happy / unhappy: boundary).
func TestGainDruidLevelFeaturesChoicesAndUses(t *testing.T) {
	t.Parallel()
	forms := func(n int) []content.MonsterID { return druIDs("a", "b", "c", "d", "e", "f", "g", "h")[:n] }
	type pending = character.DruidPendingChoice
	for _, c := range []struct {
		label            string
		previous, target int
		state            character.DruidStateFields
		features         []string
		pending          []pending
		uses             int
	}{
		{"level 1", 0, 1, character.DruidStateFields{}, []string{"druid_spellcasting", "druid_druidic", "druid_primal_order"}, []pending{{Kind: character.DruidPendingPrimalOrder, Count: 1}}, 0},
		{"level 2", 1, 2, character.DruidStateFields{PrimalOrder: character.PrimalOrderWarden}, []string{"druid_wild_shape", "druid_wild_companion"}, []pending{{Kind: character.DruidPendingKnownForms, Count: 4}}, 2},
		{"level 3, forms not yet chosen", 2, 3, character.DruidStateFields{WildShapeUsesRemaining: 1}, []string{"druid_druid_subclass"}, nil, 1},
		{"level 3", 2, 3, character.DruidStateFields{WildShapeUsesRemaining: 1, KnownForms: forms(4)}, []string{"druid_druid_subclass"}, nil, 1},
		{"level 4, forms chosen", 3, 4, character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: forms(4)}, []string{"druid_ability_score_improvement"}, []pending{{Kind: character.DruidPendingKnownForms, Count: 2}}, 2},
		{"level 4, two of four chosen", 3, 4, character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: forms(2)}, []string{"druid_ability_score_improvement"}, []pending{{Kind: character.DruidPendingKnownForms, Count: 4}}, 2},
		{"level 5", 4, 5, character.DruidStateFields{KnownForms: forms(6)}, []string{"druid_wild_resurgence"}, nil, 0},
		{"level 6 raises uses", 5, 6, character.DruidStateFields{WildShapeUsesRemaining: 1, KnownForms: forms(6)}, nil, nil, 2},
		{"level 7", 6, 7, character.DruidStateFields{WildShapeUsesRemaining: 3, KnownForms: forms(6)}, []string{"druid_elemental_fury"}, []pending{{Kind: character.DruidPendingElementalFury, Count: 1}}, 3},
		{"level 8", 7, 8, character.DruidStateFields{WildShapeUsesRemaining: 3, KnownForms: forms(6), ElementalFury: character.ElementalFuryPrimalStrike}, nil, []pending{{Kind: character.DruidPendingKnownForms, Count: 2}}, 3},
		{"level 9", 8, 9, character.DruidStateFields{KnownForms: forms(8)}, nil, nil, 0},
		{"level 15", 14, 15, character.DruidStateFields{KnownForms: forms(8)}, []string{"druid_improved_elemental_fury"}, nil, 0},
		{"level 17 raises uses", 16, 17, character.DruidStateFields{WildShapeUsesRemaining: 1, KnownForms: forms(8)}, nil, nil, 2},
		{"level 20", 19, 20, character.DruidStateFields{WildShapeUsesRemaining: 4, KnownForms: forms(8)}, []string{"druid_archdruid"}, nil, 4},
	} {
		state := character.DruidFeatureState{}
		if c.previous > 0 {
			state = druState(t, c.previous, true, c.state)
		}
		in := gainInput(t, c.previous, c.target, state)
		a := gained(t, in)
		var got []string
		for _, f := range a.Features() {
			got = append(got, string(f))
		}
		if !slices.Equal(got, c.features) || !slices.Equal(a.PendingChoices(), c.pending) || a.State().WildShapeUsesRemaining() != c.uses {
			t.Errorf("%s: features %v, pending %v, uses %d; want %v, %v, %d", c.label, got, a.PendingChoices(), a.State().WildShapeUsesRemaining(), c.features, c.pending, c.uses)
		}
		ev, ok := a.Events()[0].(character.DruidLevelGained)
		if len(a.Events()) != 1 || !ok || ev.Level() != c.target || ev.CharacterID() != druTestID || !slices.Equal(ev.Features(), a.Features()) || !slices.Equal(ev.PendingChoices(), a.PendingChoices()) {
			t.Errorf("%s: events = %#v", c.label, a.Events())
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-031, FR-148 - gaining a level keeps every earlier Druid choice (Primal Order, known forms, Elemental Fury, land type, flags), the
// input state is unchanged, and the returned lists are copies.
// Path: unhappy (immutability).
func TestGainDruidLevelKeepsEarlierChoices(t *testing.T) {
	t.Parallel()
	before := druState(t, 6, true, character.DruidStateFields{PrimalOrder: character.PrimalOrderMagician, LandType: character.LandPolar,
		WildShapeUsesRemaining: 1, KnownForms: druIDs("wolf", "rat", "spider", "horse", "bat", "cat"), NaturalRecoveryCast: true, WildResurgenceUseForSlot: true})
	snapshot := before
	a := gained(t, gainInput(t, 6, 7, before))
	after := a.State()
	if before != snapshot {
		t.Error("the input state changed")
	}
	if after.PrimalOrder() != before.PrimalOrder() || after.LandType() != before.LandType() || after.KnownForms() != before.KnownForms() ||
		!after.NaturalRecoveryCast() || !after.WildResurgenceUseForSlot() || after.WildShapeUsesRemaining() != 1 {
		t.Errorf("a choice changed: %+v", after)
	}
	a.Features()[0] = "changed"
	a.PendingChoices()[0].Count = 99
	a.Events()[0] = nil
	if a.Features()[0] == "changed" || a.PendingChoices()[0].Count == 99 || a.Events()[0] == nil {
		t.Error("the result shares storage with its caller")
	}
}

// Requirement: doc/specs/class-druid.md FR-141, FR-164 - GainDruidLevel returns an error matching ErrInvalidInput for a zero id or tables, a Previous level outside 0 to 20, a state that breaks
// an invariant at the Previous level, and a state given for no Druid level.
// Path: per case (unhappy: invalid input).
func TestGainDruidLevelInvalidInput(t *testing.T) {
	t.Parallel()
	level5 := druState(t, 5, true, character.DruidStateFields{WildShapeUsesRemaining: 1})
	for _, c := range []struct {
		label  string
		mutate func(*character.GainDruidLevelInput)
	}{
		{"zero id", func(in *character.GainDruidLevelInput) { in.ID = character.Identifier{} }},
		{"zero tables", func(in *character.GainDruidLevelInput) { in.Tables = character.DruidFeatureTables{} }},
		{"previous -1", func(in *character.GainDruidLevelInput) { in.Previous = -1 }},
		{"previous 21", func(in *character.GainDruidLevelInput) { in.Previous = 21 }},
		{"state breaks an invariant", func(in *character.GainDruidLevelInput) {
			in.Previous, in.State = 1, level5
		}},
		{"state with no Druid level", func(in *character.GainDruidLevelInput) { in.Previous, in.State = 0, level5 }},
	} {
		in := gainInput(t, 5, 6, level5)
		c.mutate(&in)
		if _, err := character.GainDruidLevel(in); err == nil {
			t.Errorf("%s: no error", c.label)
		} else {
			wantInvalid(t, err, c.label)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-161 - a state holding a Circle of the Land choice is checked against the subclass given: any other subclass id (empty,
// sorting before or after the Circle's) makes it an invariant violation.
// Path: unhappy (invalid input, rule interaction).
func TestGainDruidLevelChecksTheStateAgainstTheSubclass(t *testing.T) {
	t.Parallel()
	withLand := druState(t, 5, true, character.DruidStateFields{WildShapeUsesRemaining: 1, LandType: character.LandPolar})
	for _, sub := range []content.SubclassID{"", "a", "zzz"} {
		in := gainInput(t, 5, 6, withLand)
		in.Subclass = sub
		if _, err := character.GainDruidLevel(in); err == nil {
			t.Errorf("subclass %q: no error", sub)
		} else {
			wantInvalid(t, err, "subclass "+string(sub))
		}
	}
}
