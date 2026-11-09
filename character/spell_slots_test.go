package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spellcasting.md FR-024 - slot level 1 to 9 is valid; 0, 10 and
// negatives are rejected with ErrSlotLevelRange, which wraps srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestSlotLevelRange(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 9; n++ {
		l, err := character.NewSlotLevel(n)
		if err != nil || l.Value() != n || l.IsZero() {
			t.Errorf("NewSlotLevel(%d) = %v, %v", n, l, err)
		}
	}
	for _, n := range []int{-1, 0, 10} {
		if _, err := character.NewSlotLevel(n); !errors.Is(err, character.ErrSlotLevelRange) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewSlotLevel(%d) err = %v", n, err)
		}
	}
	if !(character.SlotLevel{}).IsZero() {
		t.Error("zero SlotLevel is not zero")
	}
	defer func() {
		if err, ok := recover().(error); !ok || !errors.Is(err, character.ErrSlotLevelRange) {
			t.Errorf("MustSlotLevel(10) panic = %v", err)
		}
	}()
	character.MustSlotLevel(10)
}

// Requirement: doc/specs/spellcasting.md FR-019, FR-020 - NewSpellSlots refuses a
// remaining count above the maximum or below zero; a derived pool starts full.
// Path: unhappy (invalid input).
func TestNewSpellSlots(t *testing.T) {
	t.Parallel()
	max := [9]int{4, 3, 2}
	ok, err := character.NewSpellSlots(max, [9]int{1, 3, 0})
	if err != nil || ok.Remaining(character.MustSlotLevel(1)) != 1 || ok.Maximum(character.MustSlotLevel(1)) != 4 || ok.Expended(character.MustSlotLevel(1)) != 3 {
		t.Fatalf("NewSpellSlots = %v, %v", ok, err)
	}
	for name, rem := range map[string][9]int{"above maximum": {5, 0, 0}, "negative": {-1, 0, 0}, "above a zero maximum": {0, 0, 0, 1}} {
		if _, err := character.NewSpellSlots(max, rem); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := character.NewSpellSlots([9]int{-1}, [9]int{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative maximum: err = %v", err)
	}
	full := character.SlotsFor(classLevelsOf(t, "wizard", 5)).Spellcasting()
	for l := 1; l <= 9; l++ {
		sl := character.MustSlotLevel(l)
		if full.Remaining(sl) != full.Maximum(sl) || full.Expended(sl) != 0 {
			t.Errorf("level %d not full", l)
		}
	}
	if full.Maximum(character.SlotLevel{}) != 0 || full.Remaining(character.SlotLevel{}) != 0 {
		t.Error("zero SlotLevel reads non-zero")
	}
	defer func() {
		if err, isErr := recover().(error); !isErr || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("MustSpellSlots panic = %v", err)
		}
	}()
	character.MustSpellSlots(max, [9]int{9})
}

// Requirement: per case.
// Path: per case.
func TestSlotsForTableH(t *testing.T) {
	t.Parallel()
	cases := []struct {
		label  string
		levels []any
		slots  []int
		pact   [2]int // count, slot level
		path   string
	}{
		{"H1 Ranger 4 Sorcerer 3", []any{"ranger", 4, "sorcerer", 3}, []int{4, 3, 2}, [2]int{}, "happy"},
		{"H2 Paladin 1", []any{"paladin", 1}, []int{2}, [2]int{}, "happy"},
		{"H3 Paladin 1 Wizard 1", []any{"paladin", 1, "wizard", 1}, []int{3}, [2]int{}, "unhappy (rule interaction)"},
		{"H4 Paladin 1 Ranger 1", []any{"paladin", 1, "ranger", 1}, []int{2}, [2]int{}, "unhappy (boundary, rounding)"},
		{"H5 Paladin 3 Ranger 3", []any{"paladin", 3, "ranger", 3}, []int{4, 2}, [2]int{}, "happy"},
		{"H6 Paladin 2 Fighter 3", []any{"paladin", 2, "fighter", 3}, []int{2}, [2]int{}, "unhappy (rule interaction)"},
		{"H7 Wizard 5 Fighter 4", []any{"wizard", 5, "fighter", 4}, []int{4, 3, 2}, [2]int{}, "happy"},
		{"H8 Wizard 3 Cleric 3", []any{"wizard", 3, "cleric", 3}, []int{4, 3, 3}, [2]int{}, "happy"},
		{"H9 Warlock 5 Wizard 3", []any{"warlock", 5, "wizard", 3}, []int{4, 2}, [2]int{2, 3}, "happy"},
		{"H10 Warlock 5", []any{"warlock", 5}, nil, [2]int{2, 3}, "happy"},
		{"H11 Bard 10 Cleric 10", []any{"bard", 10, "cleric", 10}, []int{4, 3, 3, 3, 3, 2, 2, 1, 1}, [2]int{}, "unhappy (boundary)"},
		{"H12 Paladin 20", []any{"paladin", 20}, []int{4, 3, 3, 3, 2}, [2]int{}, "unhappy (boundary)"},
		{"H13 Fighter 5", []any{"fighter", 5}, nil, [2]int{}, "unhappy (empty and minimal)"},
		{"H14 Wizard 5 Fighter 2", []any{"wizard", 5, "fighter", 2}, []int{4, 3, 2}, [2]int{}, "unhappy (rule interaction)"},
		{"order does not matter", []any{"cleric", 3, "wizard", 3}, []int{4, 3, 3}, [2]int{}, "unhappy (rule interaction)"},
		{"Paladin 1 Ranger 2 Sorcerer 1", []any{"paladin", 1, "ranger", 2, "sorcerer", 1}, []int{4, 2}, [2]int{}, "unhappy (boundary, rounding)"},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			pools := character.SlotsFor(classLevelsOf(t, c.levels...))
			for l := 1; l <= 9; l++ {
				want := 0
				if l <= len(c.slots) {
					want = c.slots[l-1]
				}
				sl := character.MustSlotLevel(l)
				if pools.Spellcasting().Maximum(sl) != want || pools.Spellcasting().Remaining(sl) != want {
					t.Errorf("spell level %d = %d/%d, want %d (FR-030 to FR-034 Table H)", l, pools.Spellcasting().Remaining(sl), pools.Spellcasting().Maximum(sl), want)
				}
			}
			p := pools.PactMagic()
			if p.Count() != c.pact[0] || p.Level().Value() != c.pact[1] || p.Remaining() != c.pact[0] || p.HasPactMagic() != (c.pact[0] > 0) {
				t.Errorf("pact = %d x %d, want %v", p.Count(), p.Level().Value(), c.pact)
			}
			if c.pact == [2]int{} && p != (character.PactMagicSlots{}) {
				t.Errorf("PactMagic = %+v, want the zero value (FR-146)", p)
			}
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-144, FR-034 - a SlotPoolView reads
// remaining, maximum and expended per level; the Pact Magic pool reads zero off its Slot
// Level and an empty pool reads zero everywhere.
// Path: happy.
func TestSlotPoolView(t *testing.T) {
	t.Parallel()
	levels := classLevelsOf(t, "warlock", 5, "wizard", 3)
	st := castingState(t, levels, spellCatalogue(t))
	res := st.ExpendSlot(character.PoolPactMagic, character.MustSlotLevel(3))
	st = res.(character.SlotExpenditureExpended).State()
	pact := st.Slots(character.PoolPactMagic)
	l3, l1 := character.MustSlotLevel(3), character.MustSlotLevel(1)
	if pact.Maximum(l3) != 2 || pact.Remaining(l3) != 1 || pact.Expended(l3) != 1 || pact.Maximum(l1) != 0 || pact.Remaining(l1) != 0 || pact.Maximum(character.SlotLevel{}) != 0 || pact.Remaining(character.SlotLevel{}) != 0 {
		t.Errorf("pact view = %+v", pact)
	}
	sc := st.Slots(character.PoolSpellcasting)
	if sc.Maximum(l1) != 4 || sc.Remaining(l1) != 4 {
		t.Errorf("spellcasting view = %+v", sc)
	}
	if empty := castingState(t, classLevelsOf(t, "wizard", 5), spellCatalogue(t)).Slots(character.PoolPactMagic); empty != (character.SlotPoolView{}) {
		t.Errorf("empty pact view = %+v", empty)
	}
	if st.Slots(0) != (character.SlotPoolView{}) {
		t.Error("invalid pool view is not empty")
	}
}

// Requirement: doc/specs/spellcasting.md FR-024, FR-109 - Must constructors accept valid input; pool and
// feature enums reject values outside their range.
// Path: unhappy (boundary).
func TestSlotPoolAndMustConstructors(t *testing.T) {
	t.Parallel()
	if character.MustSlotLevel(5).Value() != 5 {
		t.Error("MustSlotLevel(5)")
	}
	if got := character.MustSpellSlots([9]int{2}, [9]int{1}).Remaining(character.MustSlotLevel(1)); got != 1 {
		t.Errorf("MustSpellSlots remaining = %d", got)
	}
	for _, p := range []character.SpellSlotPool{0, 3, 100} {
		if p.Valid() || p.String() != "unknown" {
			t.Errorf("pool %d is valid", p)
		}
	}
	if !character.PoolSpellcasting.Valid() || !character.PoolPactMagic.Valid() {
		t.Error("the two pools are valid")
	}
}
