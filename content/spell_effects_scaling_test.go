package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

func wantRolled(t *testing.T, got content.SpellAmount, notation, req string) {
	t.Helper()
	r, ok := got.(content.RolledSpellAmount)
	if !ok || r.Roll() != dice.MustParseExpression(notation) {
		t.Errorf("amount = %#v, want %s [%s]", got, notation, req)
	}
}

// Requirement: per case
// Path: per case.
func TestSlotLevelScalingOfAmounts(t *testing.T) {
	t.Parallel()
	const fr = "spell-effect-data.md FR-016, FR-041, table L"
	cases := []struct {
		name, path string
		base       content.SpellAmount
		step       [3]int // above, extra dice, extra flat
		n          int
		want       string
		wantFixed  int
	}{
		{"L1 Fireball n 0", "unhappy (boundary)", rolled("8d6"), [3]int{3, 1, 0}, 0, "8d6", 0},
		{"L2 Fireball n 1", "happy", rolled("8d6"), [3]int{3, 1, 0}, 1, "9d6", 0},
		{"L3 Fireball n 6", "unhappy (boundary)", rolled("8d6"), [3]int{3, 1, 0}, 6, "14d6", 0},
		{"L4 Cure Wounds n 0", "unhappy (boundary)", rolled("2d8"), [3]int{1, 2, 0}, 0, "2d8", 0},
		{"L5 Cure Wounds n 1", "happy", rolled("2d8"), [3]int{1, 2, 0}, 1, "4d8", 0},
		{"L6 Cure Wounds n 8", "unhappy (boundary)", rolled("2d8"), [3]int{1, 2, 0}, 8, "18d8", 0},
		{"L7 False Life n 0", "unhappy (boundary)", rolled("2d4+4"), [3]int{1, 0, 5}, 0, "2d4+4", 0},
		{"L8 False Life n 2", "happy", rolled("2d4+4"), [3]int{1, 0, 5}, 2, "2d4+14", 0},
		{"L9 False Life n 8", "unhappy (boundary)", rolled("2d4+4"), [3]int{1, 0, 5}, 8, "2d4+44", 0},
		{"L10 Heal n 0", "unhappy (boundary)", fixed(70), [3]int{6, 0, 10}, 0, "", 70},
		{"L11 Heal n 1", "happy", fixed(70), [3]int{6, 0, 10}, 1, "", 80},
		{"L12 Heal n 3", "unhappy (boundary)", fixed(70), [3]int{6, 0, 10}, 3, "", 100},
		{"L13 Flame Strike n 2", "happy", rolled("5d6"), [3]int{5, 1, 0}, 2, "7d6", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := amount(c.base, perSlot(c.step[0], c.step[1], c.step[2]))
			got, err := a.At(c.n, 1)
			if err != nil {
				t.Fatalf("At(%d, 1): %v [%s]", c.n, err, fr)
			}
			if c.want != "" {
				wantRolled(t, got, c.want, fr)
			} else if f, ok := got.(content.FixedSpellAmount); !ok || f.Value() != c.wantFixed {
				t.Errorf("At(%d, 1) = %#v, want fixed %d [%s]", c.n, got, c.wantFixed, fr)
			}
		})
	}
}

// Requirement: spell-effect-data.md FR-021, table L13 - two outcomes of one
// part scale independently, each by its own scaling.
// Path: happy.
func TestTwoOutcomesScaleIndependently(t *testing.T) {
	t.Parallel()
	fire := amount(rolled("5d6"), perSlot(5, 1, 0))
	radiant := amount(rolled("5d6"), perSlot(5, 2, 0))
	f, _ := fire.At(2, 1)
	r, _ := radiant.At(2, 1)
	wantRolled(t, f, "7d6", "FR-021")
	wantRolled(t, r, "9d6", "FR-021")
}

// Requirement: spell-effect-data.md FR-041, table L24 - a negative number of
// slot levels, a character level outside 1 to 20, and a result past the dice
// limits are errors; an input the scaling does not read is not checked.
// Path: unhappy (invalid input).
func TestScalingQueryRejectsOutOfRangeInputs(t *testing.T) {
	t.Parallel()
	slot := amount(rolled("8d6"), perSlot(3, 1, 0))
	_, err := slot.At(-1, 1)
	wantProblem(t, err, "ScalableAmount", content.BlockOutOfRange)
	if _, err := slot.At(1, 0); err != nil { // the character level is not read
		t.Errorf("per-slot At(1, 0) = %v, want no error: the level is not read", err)
	}
	_, err = slot.At(dice.MaxCount, 1) // 8 + 1000 dice is past the dice limit
	wantProblem(t, err, "ScalableAmount", content.BlockOutOfRange)
	huge := amount(rolled("1d6"), perSlot(1, 0, 1_000_000))
	_, err = huge.At(2, 1) // modifier past the dice limit
	wantProblem(t, err, "ScalableAmount", content.BlockOutOfRange)
	flat := amount(fixed(1), perSlot(1, 0, 2))
	_, err = flat.At(int(^uint(0)>>1), 1) // would overflow an int
	wantProblem(t, err, "ScalableAmount", content.BlockOutOfRange)
	var zero content.ScalableAmount
	_, err = zero.At(0, 1)
	wantProblem(t, err, "ScalableAmount", content.BlockMissing)
	none := amount(rolled("1d6"), nil)
	if got, err := none.At(-5, 99); err != nil || got != content.SpellAmount(rolled("1d6")) {
		t.Errorf("unscaled At(-5, 99) = %v, %v; want the base and no error (reads neither input)", got, err)
	}
}

// Requirement: spell-effect-data.md FR-016, table L21, L22 - a step with both
// parts 0 or a negative part is an error; dice on a fixed base is a
// constraint violation; a nil scaling is no scaling.
// Path: unhappy (invalid input).
func TestAmountStepRules(t *testing.T) {
	t.Parallel()
	for _, p := range [][2]int{{0, 0}, {-1, 1}, {1, -1}} {
		_, err := content.NewAmountStep(p[0], p[1])
		wantProblem(t, err, "AmountStep", content.BlockOutOfRange)
	}
	_, err := content.NewScalableAmount(fixed(70), perSlot(6, 1, 0))
	wantProblem(t, err, "ScalableAmount", content.BlockConstraint)
	if a, err := content.NewScalableAmount(fixed(70), perSlot(6, 0, 10)); err != nil || a.Scaling() != content.AmountScaling(perSlot(6, 0, 10)) {
		t.Errorf("flat step on a fixed base: %v, %v; want it accepted", a, err)
	}
	if a := amount(rolled("1d6"), nil); a.Scaling() != (content.NoAmountScaling{}) {
		t.Errorf("nil scaling = %#v, want NoAmountScaling", a.Scaling())
	}
	_, err = content.NewScalableAmount(nil, nil)
	wantProblem(t, err, "SpellAmount", content.BlockMissing)
	_, err = content.NewScalableAmount(rolled("1d6"), content.PerSlotLevelAmount{})
	wantProblem(t, err, "AmountStep", content.BlockOutOfRange)
	_, err = content.NewPerSlotLevelAmount(1, content.AmountStep{})
	wantProblem(t, err, "AmountStep", content.BlockOutOfRange)
}

// Requirement: spell-effect-data.md FR-025 - a fixed amount is at least 1; a
// rolled amount wraps a valid expression.
// Path: unhappy (boundary).
func TestSpellAmountBoundaries(t *testing.T) {
	t.Parallel()
	for _, v := range []int{0, -1} {
		_, err := content.NewFixedSpellAmount(v)
		wantProblem(t, err, "FixedSpellAmount", content.BlockOutOfRange)
	}
	if a, err := content.NewFixedSpellAmount(1); err != nil || a.Value() != 1 {
		t.Errorf("NewFixedSpellAmount(1) = %v, %v", a, err)
	}
	_, err := content.NewRolledSpellAmount(dice.Expression{})
	wantProblem(t, err, "RolledSpellAmount", content.BlockOutOfRange)
	if a := rolled("2d4+4"); a.Roll().Notation() != "2d4+4" {
		t.Errorf("Roll() = %v", a.Roll())
	}
}

// Requirement: per case
// Path: per case.
func TestCharacterLevelScalingSelectsGreatestLevelNotAbove(t *testing.T) {
	t.Parallel()
	const fr = "spell-effect-data.md FR-018, FR-020, table C"
	fireBoltAmount := amount(rolled("1d10"), cantripDice("d10"))
	acid := amount(rolled("1d6"), cantripDice("d6"))
	cases := []struct {
		name, path string
		a          content.ScalableAmount
		level      int
		want       string
	}{
		{"C1 Fire Bolt level 1", "unhappy (boundary)", fireBoltAmount, 1, "1d10"},
		{"C2 level 4", "unhappy (boundary)", fireBoltAmount, 4, "1d10"},
		{"C3 level 5", "unhappy (boundary)", fireBoltAmount, 5, "2d10"},
		{"C4 level 10", "unhappy (boundary)", fireBoltAmount, 10, "2d10"},
		{"C5 level 11", "unhappy (boundary)", fireBoltAmount, 11, "3d10"},
		{"C6 level 16", "unhappy (boundary)", fireBoltAmount, 16, "3d10"},
		{"C7 level 17", "unhappy (boundary)", fireBoltAmount, 17, "4d10"},
		{"C8 level 20", "unhappy (boundary)", fireBoltAmount, 20, "4d10"},
		{"C13 Acid Splash level 17", "happy", acid, 17, "4d6"},
		{"C14 multiclass total level 5 (Wizard 3 and Cleric 2)", "unhappy (rule interaction)", fireBoltAmount, 5, "2d10"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.a.At(0, c.level)
			if err != nil {
				t.Fatalf("At(0, %d): %v [%s]", c.level, err, fr)
			}
			wantRolled(t, got, c.want, fr)
		})
	}
}

// Requirement: spell-effect-data.md FR-020, table C9 to C12 - a count table
// holds the printed counts (Eldritch Blast: 1, 2, 3, 4 beams).
// Path: unhappy (boundary).
func TestCharacterLevelScalingOfCounts(t *testing.T) {
	t.Parallel()
	m := eldritchBlast()
	p := m.Parts()[0].Instances()
	for level, want := range map[int]int{1: 1, 4: 1, 5: 2, 10: 2, 11: 3, 16: 3, 17: 4, 20: 4} {
		if got, err := p.At(0, level); err != nil || got != want {
			t.Errorf("beams at character level %d = %d, %v; want %d [FR-020, C9-C12]", level, got, err, want)
		}
	}
}

// Requirement: spell-effect-data.md FR-041, table C20 - a character level of 0
// or 21 is an error, for amounts, counts and the tables themselves.
// Path: unhappy (boundary).
func TestCharacterLevelOutOfRange(t *testing.T) {
	t.Parallel()
	a := amount(rolled("1d10"), cantripDice("d10"))
	table := must(content.NewCountLevelTable([]content.CountLevelEntry{{Level: 1, Count: 1}}))
	beams := must(content.NewPartInstances(1, must(content.NewByCharacterLevelCount(table))))
	for _, level := range []int{0, 21, -3} {
		_, err := a.At(0, level)
		wantProblem(t, err, "AmountLevelTable", content.BlockOutOfRange)
		_, err = beams.At(0, level)
		wantProblem(t, err, "CountLevelTable", content.BlockOutOfRange)
		_, err = table.ValueAt(level)
		wantProblem(t, err, "CountLevelTable", content.BlockOutOfRange)
	}
	_, err := content.AmountLevelTable{}.ValueAt(1)
	wantProblem(t, err, "AmountLevelTable", content.BlockMissing)
	_, err = content.CountLevelTable{}.ValueAt(1)
	wantProblem(t, err, "CountLevelTable", content.BlockMissing)
}

// Requirement: spell-effect-data.md FR-019, table C15 to C17 - a table has at
// least one entry, starts at level 1, ascends strictly, and each level is
// between 1 and 20.
// Path: per case.
func TestLevelTableRules(t *testing.T) {
	t.Parallel()
	d := rolled("1d6")
	cases := []struct {
		name    string
		levels  []int
		problem content.BlockProblem
	}{
		{"C15 first entry level 2", []int{2, 5}, content.BlockConstraint},
		{"C16 levels 5 then 5", []int{1, 5, 5}, content.BlockConstraint},
		{"descending levels", []int{1, 11, 5}, content.BlockConstraint},
		{"C17 level 21", []int{1, 21}, content.BlockOutOfRange},
		{"level 0", []int{0, 5}, content.BlockOutOfRange},
		{"empty", nil, content.BlockEmpty},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			amounts := make([]content.AmountLevelEntry, len(c.levels))
			counts := make([]content.CountLevelEntry, len(c.levels))
			for i, l := range c.levels {
				amounts[i] = content.AmountLevelEntry{Level: l, Value: d}
				counts[i] = content.CountLevelEntry{Level: l, Count: 1}
			}
			_, err := content.NewAmountLevelTable(amounts)
			wantProblem(t, err, "AmountLevelTable", c.problem)
			_, err = content.NewCountLevelTable(counts)
			wantProblem(t, err, "CountLevelTable", c.problem)
		})
	}
	for _, entries := range [][]content.AmountLevelEntry{{{Level: 1}}, {{Level: 1, Value: content.FixedSpellAmount{}}}} {
		_, err := content.NewAmountLevelTable(entries)
		if err == nil {
			t.Errorf("NewAmountLevelTable(%v) accepted an invalid amount", entries)
		}
	}
	_, err := content.NewCountLevelTable([]content.CountLevelEntry{{Level: 1, Count: 0}})
	wantProblem(t, err, "CountLevelTable", content.BlockOutOfRange)
	// A full table (levels 1 to 20) fits, and a sparse table is accepted.
	full := make([]content.CountLevelEntry, 20)
	for i := range full {
		full[i] = content.CountLevelEntry{Level: i + 1, Count: i + 1}
	}
	tab, err := content.NewCountLevelTable(full)
	if v, verr := tab.ValueAt(20); err != nil || verr != nil || v != 20 {
		t.Errorf("full table: %v, %v, ValueAt(20) = %d", err, verr, v)
	}
}

// Requirement: spell-effect-data.md FR-007 - tables copy their entries on
// construction and on read.
// Path: unhappy (immutability).
func TestLevelTablesAreCopied(t *testing.T) {
	t.Parallel()
	in := []content.CountLevelEntry{{Level: 1, Count: 1}, {Level: 5, Count: 2}}
	tab := must(content.NewCountLevelTable(in))
	in[1].Count = 9
	out := tab.Entries()
	out[0].Count = 9
	if v, _ := tab.ValueAt(5); v != 2 {
		t.Errorf("ValueAt(5) = %d after mutating the input, want 2", v)
	}
	if v, _ := tab.ValueAt(1); v != 1 {
		t.Errorf("ValueAt(1) = %d after mutating a returned slice, want 1", v)
	}
	ain := []content.AmountLevelEntry{{Level: 1, Value: rolled("1d6")}}
	atab := must(content.NewAmountLevelTable(ain))
	ain[0].Value = rolled("9d6")
	atab.Entries()[0].Value = rolled("9d6")
	if v, _ := atab.ValueAt(1); v != content.SpellAmount(rolled("1d6")) {
		t.Errorf("ValueAt(1) = %v, want 1d6", v)
	}
	if got := atab.Entries(); len(got) != 1 || got[0].Level != 1 {
		t.Errorf("Entries() = %v", got)
	}
	if tab != must(content.NewCountLevelTable([]content.CountLevelEntry{{Level: 1, Count: 1}, {Level: 5, Count: 2}})) {
		t.Error("equal tables are not equal with ==")
	}
}

// Requirement: per case
// Path: per case.
func TestSlotLevelScalingOfCounts(t *testing.T) {
	t.Parallel()
	const fr = "spell-effect-data.md FR-017, FR-041, table L14 to L19"
	missile := magicMissile().Parts()[0].Instances()
	hold := holdPerson().Targeting().(content.TargetsChosen).Count().(content.ExactlyTargets)
	cases := []struct {
		name, path string
		got        func(n int) (int, error)
		n, want    int
	}{
		{"L14 Magic Missile n 0", "unhappy (boundary)", func(n int) (int, error) { return missile.At(n, 1) }, 0, 3},
		{"L15 n 1", "happy", func(n int) (int, error) { return missile.At(n, 1) }, 1, 4},
		{"L16 n 8", "unhappy (boundary)", func(n int) (int, error) { return missile.At(n, 1) }, 8, 11},
		{"L17 Hold Person n 0", "unhappy (boundary)", func(n int) (int, error) { return hold.At(n, 1) }, 0, 1},
		{"L18 n 1", "happy", func(n int) (int, error) { return hold.At(n, 1) }, 1, 2},
		{"L19 n 7", "unhappy (boundary)", func(n int) (int, error) { return hold.At(n, 1) }, 7, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got, err := c.got(c.n); err != nil || got != c.want {
				t.Errorf("count at n %d = %d, %v; want %d [%s]", c.n, got, err, c.want, fr)
			}
		})
	}
	if _, err := missile.At(-1, 1); err == nil {
		t.Error("a negative n was accepted [FR-041]")
	} else {
		wantProblem(t, err, "PartInstances", content.BlockOutOfRange)
	}
	_, err := missile.At(int(^uint(0)>>1), 1)
	wantProblem(t, err, "PartInstances", content.BlockOutOfRange)
	_, err = hold.At(-1, 1)
	wantProblem(t, err, "ExactlyTargets", content.BlockOutOfRange)
	upTo := must(content.NewUpToTargets(3, must(content.NewPerSlotLevelCount(1, 1))))
	if got, err := upTo.At(2, 1); err != nil || got != 5 {
		t.Errorf("UpToTargets.At(2) = %d, %v; want 5", got, err)
	}
	var zero content.PartInstances
	_, err = zero.At(0, 1)
	wantProblem(t, err, "PartInstances", content.BlockMissing)
}

// Requirement: spell-effect-data.md FR-017 - the extra count of a per-slot
// scaling is at least 1.
// Path: unhappy (boundary).
func TestPerSlotLevelCountBoundary(t *testing.T) {
	t.Parallel()
	for _, extra := range []int{0, -1} {
		_, err := content.NewPerSlotLevelCount(1, extra)
		wantProblem(t, err, "PerSlotLevelCount", content.BlockOutOfRange)
	}
	s, err := content.NewPerSlotLevelCount(2, 1)
	if err != nil || s.AboveLevel() != 2 || s.Extra() != 1 {
		t.Errorf("NewPerSlotLevelCount(2, 1) = %v, %v", s, err)
	}
	_, err = content.NewByCharacterLevelCount(content.CountLevelTable{})
	wantProblem(t, err, "ByCharacterLevelCount", content.BlockEmpty)
	_, err = content.NewByCharacterLevelAmount(content.AmountLevelTable{})
	wantProblem(t, err, "ByCharacterLevelAmount", content.BlockEmpty)
}

// Requirement: spell-effect-data.md FR-016, FR-017, FR-019 - the scaling
// variants and the scalable amount give back what they were built with.
// Path: happy.
func TestScalingAccessors(t *testing.T) {
	t.Parallel()
	step := must(content.NewAmountStep(1, 2))
	if step.ExtraDice() != 1 || step.ExtraFlat() != 2 {
		t.Errorf("step = %d dice, %d flat", step.ExtraDice(), step.ExtraFlat())
	}
	ps := must(content.NewPerSlotLevelAmount(3, step))
	if ps.AboveLevel() != 3 || ps.Step() != step {
		t.Errorf("PerSlotLevelAmount = %v", ps)
	}
	by := cantripDice("d6")
	if len(by.Table().Entries()) != 4 {
		t.Errorf("table entries = %v", by.Table().Entries())
	}
	a := amount(fixed(70), nil)
	if a.Base() != content.SpellAmount(fixed(70)) {
		t.Errorf("Base() = %v", a.Base())
	}
	counts := must(content.NewByCharacterLevelCount(must(content.NewCountLevelTable([]content.CountLevelEntry{{Level: 1, Count: 2}}))))
	if counts.Table().Entries()[0].Count != 2 {
		t.Errorf("count table = %v", counts.Table().Entries())
	}
	up := must(content.NewPartInstances(2, counts))
	if up.Scaling() != content.CountScaling(counts) {
		t.Errorf("Scaling() = %v", up.Scaling())
	}
	if e := exactly(1, nil); e.Scaling() != content.CountScaling(content.NoCountScaling{}) {
		t.Errorf("ExactlyTargets scaling = %v", e.Scaling())
	}
	if u := must(content.NewUpToTargets(1, counts)); u.Scaling() != content.CountScaling(counts) {
		t.Errorf("UpToTargets scaling = %v", u.Scaling())
	}
}
