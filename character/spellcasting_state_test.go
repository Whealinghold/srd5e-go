package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func slotLevel(n int) character.SlotLevel { return character.MustSlotLevel(n) }

// remaining reads the remaining slots of a level in a pool.
func remaining(s character.SpellcastingState, pool character.SpellSlotPool, level int) int {
	return s.Slots(pool).Remaining(slotLevel(level))
}

// Requirement: per case.
// Path: per case.
func TestExpendSlot(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	wiz5 := castingState(t, classLevelsOf(t, "wizard", 5), spells)
	wiz2 := castingState(t, classLevelsOf(t, "wizard", 2), spells)
	lock5 := castingState(t, classLevelsOf(t, "warlock", 5), spells)
	spent := func(s character.SpellcastingState, pool character.SpellSlotPool, level, times int) character.SpellcastingState {
		for range times {
			s = s.ExpendSlot(pool, slotLevel(level)).(character.SlotExpenditureExpended).State()
		}
		return s
	}
	cases := []struct {
		name, requirement, path string
		state                   character.SpellcastingState
		pool                    character.SpellSlotPool
		level                   int
		wantRemaining           int // -1: rejected
	}{
		{"Q1 level 2 of a Wizard 5", "FR-021 Table Q1", "happy", wiz5, character.PoolSpellcasting, 2, 2},
		{"Q2 level 2 with none remaining", "FR-022 Table Q2", "unhappy (state error)", spent(wiz5, character.PoolSpellcasting, 2, 3), character.PoolSpellcasting, 2, -1},
		{"FR-023 level 3 of a Wizard 2", "FR-023", "unhappy (boundary)", wiz2, character.PoolSpellcasting, 3, -1},
		{"Q3 Pact Magic slot of a Warlock 5", "FR-109 Table Q3", "happy", lock5, character.PoolPactMagic, 3, 1},
		{"Pact Magic slot off its Slot Level", "FR-109 FR-144", "unhappy (boundary)", lock5, character.PoolPactMagic, 1, -1},
		{"Q8 Pact pool of a Wizard 5", "FR-126 Table Q8", "unhappy (boundary)", wiz5, character.PoolPactMagic, 1, -1},
		{"invalid pool", "FR-126", "unhappy (invalid input)", wiz5, 0, 1, -1},
		{"zero slot level", "FR-024", "unhappy (invalid input)", wiz5, character.PoolSpellcasting, 0, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			var lvl character.SlotLevel
			if c.level > 0 {
				lvl = slotLevel(c.level)
			}
			switch r := c.state.ExpendSlot(c.pool, lvl).(type) {
			case character.SlotExpenditureExpended:
				if c.wantRemaining < 0 {
					t.Fatalf("expended, want rejected [%s]", c.requirement)
				}
				if got := r.State().Slots(c.pool).Remaining(lvl); got != c.wantRemaining {
					t.Errorf("remaining = %d, want %d [%s]", got, c.wantRemaining, c.requirement)
				}
				if e := r.Event(); e.Level() != lvl || e.Pool() != c.pool || e.CharacterID() != caster {
					t.Errorf("event = %#v", e)
				}
				for l := 1; l <= 9; l++ { // every other level is unchanged
					if l != c.level && remaining(r.State(), character.PoolSpellcasting, l) != remaining(c.state, character.PoolSpellcasting, l) {
						t.Errorf("level %d changed", l)
					}
				}
			case character.SlotExpenditureRejected:
				if c.wantRemaining >= 0 || r.Reason() != character.SlotOperationNoSlotAvailable {
					t.Errorf("rejected %v, want remaining %d [%s]", r.Reason(), c.wantRemaining, c.requirement)
				}
			}
			if !reflect.DeepEqual(before, c.state) {
				t.Error("the input state was altered")
			}
		})
	}
}

// Requirement: per case.
// Path: per case.
func TestRegainSlot(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	wiz5 := castingState(t, classLevelsOf(t, "wizard", 5), spells)
	two := wiz5.ExpendSlot(character.PoolSpellcasting, slotLevel(1)).(character.SlotExpenditureExpended).State()
	two = two.ExpendSlot(character.PoolSpellcasting, slotLevel(1)).(character.SlotExpenditureExpended).State()
	lock := castingState(t, classLevelsOf(t, "warlock", 5), spells)
	lock = lock.ExpendSlot(character.PoolPactMagic, slotLevel(3)).(character.SlotExpenditureExpended).State()
	cases := []struct {
		name, requirement, path string
		state                   character.SpellcastingState
		pool                    character.SpellSlotPool
		level, count            int
		want                    int // slots restored; 0 means rejected
	}{
		{"Q5 one of two expended", "FR-110 Table Q5", "happy", two, character.PoolSpellcasting, 1, 1, 1},
		{"Q6 three requested, two expended", "FR-110 Table Q6", "unhappy (boundary)", two, character.PoolSpellcasting, 1, 3, 2},
		{"Q7 none expended", "FR-110 Table Q7", "unhappy (empty and minimal)", wiz5, character.PoolSpellcasting, 1, 1, 0},
		{"Q9 Pact pool of a Wizard 5", "FR-110 FR-126 Table Q9", "unhappy (boundary)", wiz5, character.PoolPactMagic, 1, 1, 0},
		{"pact slot restored", "FR-110", "happy", lock, character.PoolPactMagic, 3, 1, 1},
		{"count zero", "FR-110", "unhappy (boundary)", two, character.PoolSpellcasting, 1, 0, 0},
		{"count negative", "FR-110", "unhappy (boundary)", two, character.PoolSpellcasting, 1, -2, 0},
		{"another level", "FR-110", "unhappy (state error)", two, character.PoolSpellcasting, 2, 1, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			switch r := c.state.RegainSlot(c.pool, slotLevel(c.level), c.count).(type) {
			case character.SlotRecoveryRegained:
				got := r.Event().Spellcasting() + r.Event().PactMagic()
				if got != c.want || r.State().Slots(c.pool).Remaining(slotLevel(c.level)) != c.state.Slots(c.pool).Remaining(slotLevel(c.level))+c.want {
					t.Errorf("restored %d, want %d [%s]", got, c.want, c.requirement)
				}
				if (c.pool == character.PoolPactMagic) != (r.Event().PactMagic() > 0) || r.Event().CharacterID() != caster {
					t.Errorf("event = %#v", r.Event())
				}
			case character.SlotRecoveryRejected:
				if c.want != 0 || r.Reason() != character.SlotOperationNoExpendedSlot {
					t.Errorf("rejected %v, want %d restored [%s]", r.Reason(), c.want, c.requirement)
				}
			}
			if !reflect.DeepEqual(before, c.state) {
				t.Error("the input state was altered")
			}
		})
	}
	if _, ok := wiz5.RegainSlot(character.PoolSpellcasting, character.SlotLevel{}, 1).(character.SlotRecoveryRejected); !ok {
		t.Error("a zero slot level regained a slot")
	}
}

// Requirement: doc/specs/spellcasting.md FR-025, FR-026, FR-027 - a Long Rest restores both
// pools, a Short Rest only Pact Magic, and a rest with nothing expended returns an equal
// state and no events.
// Path: unhappy (rule interaction).
func TestRestoreSlots(t *testing.T) {
	t.Parallel()
	st := castingState(t, classLevelsOf(t, "warlock", 5, "wizard", 3), spellCatalogue(t))
	spend := func(s character.SpellcastingState, pool character.SpellSlotPool, level int) character.SpellcastingState {
		return s.ExpendSlot(pool, slotLevel(level)).(character.SlotExpenditureExpended).State()
	}
	st = spend(spend(spend(st, character.PoolSpellcasting, 1), character.PoolSpellcasting, 2), character.PoolPactMagic, 3)

	short, events := st.RestoreSlots(character.ShortRest)
	if remaining(short, character.PoolPactMagic, 3) != 2 || remaining(short, character.PoolSpellcasting, 1) != 3 || remaining(short, character.PoolSpellcasting, 2) != 1 {
		t.Errorf("Short Rest: pact %d, level 1 %d, level 2 %d", remaining(short, character.PoolPactMagic, 3), remaining(short, character.PoolSpellcasting, 1), remaining(short, character.PoolSpellcasting, 2))
	}
	if e, ok := events[0].(character.SpellSlotsRestored); len(events) != 1 || !ok || e.Spellcasting() != 0 || e.PactMagic() != 1 || e.CharacterID() != caster {
		t.Errorf("Short Rest events = %#v", events)
	}
	long, events := st.RestoreSlots(character.LongRest)
	if remaining(long, character.PoolPactMagic, 3) != 2 || remaining(long, character.PoolSpellcasting, 1) != 4 || remaining(long, character.PoolSpellcasting, 2) != 2 {
		t.Error("Long Rest did not restore both pools")
	}
	if e := events[0].(character.SpellSlotsRestored); len(events) != 1 || e.Spellcasting() != 2 || e.PactMagic() != 1 {
		t.Errorf("Long Rest events = %#v", events)
	}
	again, none := long.RestoreSlots(character.LongRest)
	if !reflect.DeepEqual(again, long) || len(none) != 0 {
		t.Error("restoring a full state changed it or raised events (FR-027)")
	}
	if same, none := st.RestoreSlots(0); !reflect.DeepEqual(same, st) || len(none) != 0 {
		t.Error("an invalid rest kind changed the state")
	}
	if remaining(st, character.PoolPactMagic, 3) != 1 {
		t.Error("the input state was altered")
	}
}

func advanced(t *testing.T, s character.SpellcastingState, levels character.ClassLevels) character.SpellcastingLevelChanged {
	t.Helper()
	r, ok := s.Advance(levels).(character.SpellcastingLevelChanged)
	if !ok {
		t.Fatalf("Advance = %#v, want SpellcastingLevelChanged", s.Advance(levels))
	}
	return r
}

func count(cs []character.ChoiceCount, c character.SpellcastingClass) int {
	for _, x := range cs {
		if x.Class() == c {
			return x.Count()
		}
	}
	return -1
}

// Requirement: per case.
// Path: per case.
func TestAdvanceSlots(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	spend := func(s character.SpellcastingState, pool character.SpellSlotPool, level, n int) character.SpellcastingState {
		for range n {
			s = s.ExpendSlot(pool, slotLevel(level)).(character.SlotExpenditureExpended).State()
		}
		return s
	}
	cases := []struct {
		name, requirement, path string
		from, to                character.ClassLevels
		spend                   []int    // pool (1 spellcasting, 2 pact), level, times
		check                   [][3]int // pool, level, wantRemaining; maximums are checked against the tables
		maxima                  [][3]int // pool, level, wantMaximum
	}{
		{"O1 wizard 4 to 5", "FR-029 Table O1", "happy", classLevelsOf(t, "wizard", 4), classLevelsOf(t, "wizard", 5), []int{1, 1, 1},
			[][3]int{{1, 1, 3}, {1, 2, 3}, {1, 3, 2}}, [][3]int{{1, 1, 4}, {1, 2, 3}, {1, 3, 2}}},
		{"O2 warlock 1 to 2", "FR-111 Table O2", "unhappy (boundary)", classLevelsOf(t, "warlock", 1), classLevelsOf(t, "warlock", 2), []int{2, 1, 1},
			[][3]int{{2, 1, 1}}, [][3]int{{2, 1, 2}}},
		{"O3 warlock 2 to 3", "FR-111 Table O3", "unhappy (rule interaction)", classLevelsOf(t, "warlock", 2), classLevelsOf(t, "warlock", 3), []int{2, 1, 1},
			[][3]int{{2, 2, 1}, {2, 1, 0}}, [][3]int{{2, 2, 2}, {2, 1, 0}}},
		{"O4 warlock 10 to 11", "FR-111 Table O4", "unhappy (boundary)", classLevelsOf(t, "warlock", 10), classLevelsOf(t, "warlock", 11), nil,
			[][3]int{{2, 5, 3}}, [][3]int{{2, 5, 3}}},
		{"O5 warlock 4 to 5", "FR-111 Table O5", "unhappy (boundary)", classLevelsOf(t, "warlock", 4), classLevelsOf(t, "warlock", 5), []int{2, 2, 2},
			[][3]int{{2, 3, 0}}, [][3]int{{2, 3, 2}}},
		{"a first Warlock level starts full", "FR-111", "happy", classLevelsOf(t, "wizard", 3), classLevelsOf(t, "wizard", 3, "warlock", 1), nil,
			[][3]int{{2, 1, 1}}, [][3]int{{2, 1, 1}}},
		{"multiclass table at the new total", "FR-029 FR-031", "happy", classLevelsOf(t, "wizard", 4, "ranger", 2), classLevelsOf(t, "wizard", 4, "ranger", 3), []int{1, 2, 1},
			[][3]int{{1, 1, 4}, {1, 2, 2}, {1, 3, 3}}, [][3]int{{1, 1, 4}, {1, 2, 3}, {1, 3, 3}}},
	}
	pools := map[int]character.SpellSlotPool{1: character.PoolSpellcasting, 2: character.PoolPactMagic}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			st := castingState(t, c.from, spells)
			if c.spend != nil {
				st = spend(st, pools[c.spend[0]], c.spend[1], c.spend[2])
			}
			before := st
			got := advanced(t, st, c.to).State()
			for _, w := range c.check {
				if r := remaining(got, pools[w[0]], w[1]); r != w[2] {
					t.Errorf("pool %d level %d remaining = %d, want %d [%s]", w[0], w[1], r, w[2], c.requirement)
				}
			}
			for _, w := range c.maxima {
				if m := got.Slots(pools[w[0]]).Maximum(slotLevel(w[1])); m != w[2] {
					t.Errorf("pool %d level %d maximum = %d, want %d [%s]", w[0], w[1], m, w[2], c.requirement)
				}
			}
			if !got.ClassLevels().Equal(c.to) {
				t.Error("the state does not follow the new class levels")
			}
			if !reflect.DeepEqual(before, st) {
				t.Error("the input state was altered")
			}
		})
	}
}

// Requirement: per case.
// Path: per case.
func TestAdvanceRejections(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	cases := []struct {
		name, requirement, path string
		from, to                character.ClassLevels
		want                    character.SpellcastingAdvanceRejection
	}{
		{"O6 same levels", "FR-112 Table O6", "unhappy (empty and minimal)", classLevelsOf(t, "wizard", 5), classLevelsOf(t, "wizard", 5), character.SpellcastingAdvanceNoChange},
		{"same levels in another order", "FR-112 FR-008", "unhappy (rule interaction)", classLevelsOf(t, "wizard", 5, "cleric", 2), classLevelsOf(t, "cleric", 2, "wizard", 5), character.SpellcastingAdvanceNoChange},
		{"O7 wizard 5 to 4", "FR-112 Table O7", "unhappy (state error)", classLevelsOf(t, "wizard", 5), classLevelsOf(t, "wizard", 4), character.SpellcastingAdvanceLevelLowered},
		{"O8 wizard 5 fighter 1 to wizard 5", "FR-112 Table O8", "unhappy (state error)", classLevelsOf(t, "wizard", 5, "fighter", 1), classLevelsOf(t, "wizard", 5), character.SpellcastingAdvanceClassRemoved},
		{"O9 wizard 5 to 7", "FR-112 Table O9", "unhappy (boundary)", classLevelsOf(t, "wizard", 5), classLevelsOf(t, "wizard", 7), character.SpellcastingAdvanceLevelsSkipped},
		{"two classes each raised", "FR-112", "unhappy (boundary)", classLevelsOf(t, "wizard", 5, "cleric", 2), classLevelsOf(t, "wizard", 6, "cleric", 3), character.SpellcastingAdvanceLevelsSkipped},
		{"lowered is checked before removed", "FR-112", "unhappy (rule interaction)", classLevelsOf(t, "wizard", 5, "cleric", 2), classLevelsOf(t, "wizard", 4), character.SpellcastingAdvanceLevelLowered},
		{"a swap of one level between classes", "FR-112", "unhappy (state error)", classLevelsOf(t, "wizard", 5, "cleric", 2), classLevelsOf(t, "wizard", 4, "cleric", 3), character.SpellcastingAdvanceLevelLowered},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			st := castingState(t, c.from, spells)
			r, ok := st.Advance(c.to).(character.SpellcastingLevelChangeRejected)
			if !ok || r.Reason() != c.want {
				t.Errorf("Advance = %#v, want rejection %v [%s]", st.Advance(c.to), c.want, c.requirement)
			}
		})
	}
	for i, n := range []string{"level_lowered", "class_removed", "levels_skipped", "no_change"} {
		if got := character.SpellcastingAdvanceRejection(i + 1).String(); got != n {
			t.Errorf("rejection %d = %q, want %q", i+1, got, n)
		}
	}
	if character.SpellcastingAdvanceRejection(0).String() != "unknown" || character.SlotOperationRejection(0).String() != "unknown" ||
		character.SlotOperationNoSlotAvailable.String() != "no_slot_available" || character.SlotOperationNoExpendedSlot.String() != "no_expended_slot" {
		t.Error("rejection names")
	}
}

// Requirement: per case.
// Path: per case.
func TestAdvanceChoices(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	bard2 := castingState(t, classLevelsOf(t, "bard", 2), spells, character.ClassSpellcastingSetup{Class: character.CasterBard, Prepared: spellIDs(1, 5), Cantrips: spellIDs(0, 2)})
	wiz3 := castingState(t, classLevelsOf(t, "wizard", 3), spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 3)})
	wiz4 := castingState(t, classLevelsOf(t, "wizard", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 4)})
	wiz5 := castingState(t, classLevelsOf(t, "wizard", 5), spells)
	cleric3 := castingState(t, classLevelsOf(t, "cleric", 3), spells, character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: spellIDs(0, 4), ExtraCantrips: 1})
	paladin2 := castingState(t, classLevelsOf(t, "paladin", 2), spells, character.ClassSpellcastingSetup{Class: character.CasterPaladin,
		Granted: []character.GrantedCantrip{mustGranted(t, spellID(0, 1), character.CasterPaladin, "cleric"), mustGranted(t, spellID(0, 2), character.CasterPaladin, "cleric")}})
	cases := []struct {
		name, requirement, path string
		from                    character.SpellcastingState
		to                      character.ClassLevels
		class                   character.SpellcastingClass
		spells, cantrips        int
	}{
		{"Bard 2 to 3 has one spell to choose", "FR-047 FR-090", "unhappy (boundary)", bard2, classLevelsOf(t, "bard", 3), character.CasterBard, 1, 0},
		{"O10 a Cleric enters a Wizard 5", "FR-112 FR-047 Table O10", "happy", wiz5, classLevelsOf(t, "wizard", 5, "cleric", 1), character.CasterCleric, 4, 3},
		{"O11 wizard 3 to 4 gains a cantrip", "FR-047 Table O11", "unhappy (boundary)", wiz3, classLevelsOf(t, "wizard", 4), character.CasterWizard, 7, 1},
		{"O12 wizard 4 to 5 gains none", "FR-047 Table O12", "unhappy (boundary)", wiz4, classLevelsOf(t, "wizard", 5), character.CasterWizard, 9, 0},
		{"O13 cleric with Thaumaturge 3 to 4", "FR-047 FR-096 Table O13", "unhappy (rule interaction)", cleric3, classLevelsOf(t, "cleric", 4), character.CasterCleric, 7, 1},
		{"O14 paladin with granted cantrips 2 to 3", "FR-047 FR-018 FR-097 Table O14", "unhappy (rule interaction)", paladin2, classLevelsOf(t, "paladin", 3), character.CasterPaladin, 4, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := advanced(t, c.from, c.to)
			if got := count(r.SpellsToChoose(), c.class); got != c.spells {
				t.Errorf("SpellsToChoose = %d, want %d [%s]", got, c.spells, c.requirement)
			}
			if got := count(r.CantripsToChoose(), c.class); got != c.cantrips {
				t.Errorf("CantripsToChoose = %d, want %d [%s]", got, c.cantrips, c.requirement)
			}
			if len(r.Events()) != 0 {
				t.Errorf("events = %v, want none", r.Events())
			}
		})
	}
	// The classes without Spellcasting are not reported; the report follows the class-level order.
	r := advanced(t, wiz5, classLevelsOf(t, "wizard", 5, "fighter", 1))
	if len(r.SpellsToChoose()) != 1 || r.SpellsToChoose()[0].Class() != character.CasterWizard {
		t.Errorf("SpellsToChoose = %v", r.SpellsToChoose())
	}
	r.SpellsToChoose()[0] = character.ChoiceCount{}
	if count(r.SpellsToChoose(), character.CasterWizard) != 9 {
		t.Error("SpellsToChoose shares memory")
	}
}

// Requirement: doc/specs/spellcasting.md FR-127, FR-028, FR-008 - the state keeps the character
// id, returns copies of its class levels and lists, and holds no Concentration or casting
// in progress until the casting rules set them.
// Path: unhappy (immutability).
func TestStateAccessors(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	levels := classLevelsOf(t, "wizard", 3, "fighter", 2)
	st := castingState(t, levels, spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 2)})
	if st.IsZero() || st.CharacterID() != caster || !st.ClassLevels().Equal(levels) {
		t.Errorf("state = %+v", st)
	}
	if _, ok := st.Concentration(); ok {
		t.Error("a new state is concentrating")
	}
	if _, ok := st.CastingInProgress(); ok {
		t.Error("a new state is casting")
	}
	if _, ok := st.Class(character.CasterCleric); ok {
		t.Error("state holds a Cleric")
	}
	if len(st.Classes()) != 1 {
		t.Fatalf("Classes = %v, want only the Wizard", st.Classes())
	}
	st.Classes()[0] = character.ClassSpellcasting{}
	st.ClassLevels().Entries()[0].Level = 9
	w, _ := st.Class(character.CasterWizard)
	w.Cantrips().Spells()[0] = content.SpellID("x")
	if got, _ := st.Class(character.CasterWizard); got.IsZero() || got.Cantrips().Spells()[0] != spellID(0, 1) || st.ClassLevels().LevelIn(classOf("wizard")) != 3 {
		t.Error("the state shares memory with its accessors")
	}
	if !(character.SpellcastingState{}).IsZero() {
		t.Error("zero state is not zero")
	}
}

// Requirement: doc/specs/spellcasting.md FR-025, FR-117, FR-126, FR-028 - a Long Rest that restores equal counts in both
// pools still raises its event; an undefined rest kind restores nothing; a change to one class of several lands on
// that class; the same absent spell named by two setups is reported once; a granted cantrip of a higher class than its
// setup is refused.
// Path: unhappy (rule interaction).
func TestStateEdgeCases(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	st := castingState(t, classLevelsOf(t, "warlock", 5, "wizard", 3), spells)
	st = st.ExpendSlot(character.PoolPactMagic, slotLevel(3)).(character.SlotExpenditureExpended).State()
	st = st.ExpendSlot(character.PoolSpellcasting, slotLevel(1)).(character.SlotExpenditureExpended).State()
	if got, events := st.RestoreSlots(character.LongRest); len(events) != 1 || remaining(got, character.PoolSpellcasting, 1) != 4 {
		t.Errorf("one slot in each pool: events %v", events)
	}
	if got, events := st.RestoreSlots(character.RestKind(3)); len(events) != 0 || !reflect.DeepEqual(got, st) {
		t.Errorf("rest kind 3 restored slots: %v", events)
	}

	two := castingState(t, classLevelsOf(t, "cleric", 3, "wizard", 3), spells)
	out, err := character.ChoosePreparation(two, character.CasterWizard, nil, spellIDs(0, 1), spells)
	st2, _ := accepted(t, out, err)
	if classOfState(t, st2, character.CasterWizard).Cantrips().Count() != 1 || classOfState(t, st2, character.CasterCleric).Cantrips().Count() != 0 {
		t.Error("the change landed on the wrong class")
	}

	for name, setups := range map[string][]character.ClassSpellcastingSetup{
		"missing in two setups": {{Class: character.CasterCleric, Prepared: []content.SpellID{"gone"}}, {Class: character.CasterWizard, Cantrips: []content.SpellID{"gone"}}},
	} {
		res, err := character.NewCreateSpellcasting().Do(t.Context(), character.CreateSpellcastingCommand{Character: caster, Levels: classLevelsOf(t, "cleric", 1, "wizard", 1), Classes: setups, Spells: spells})
		if m, ok := res.(character.CreateSpellcastingContentMissing); err != nil || !ok || len(m.Missing()) != 1 {
			t.Errorf("%s: %#v %v", name, res, err)
		}
	}
	granted := mustGranted(t, spellID(0, 1), character.CasterWizard, "wizard")
	_, err = character.NewCreateSpellcasting().Do(t.Context(), character.CreateSpellcastingCommand{Character: caster, Levels: classLevelsOf(t, "cleric", 1, "wizard", 1), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterCleric, Granted: []character.GrantedCantrip{granted}}}})
	if !isInvalid(err) {
		t.Errorf("granted of a higher class: %v", err)
	}
}
