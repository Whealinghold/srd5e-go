package character

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ErrClassNotHeld is matched (errors.Is) by the error returned for a command that
// names a spellcasting class the SpellcastingState does not hold. It also
// matches srderr.ErrInvalidInput.
var ErrClassNotHeld = fmt.Errorf("character: spellcasting class not held: %w", srderr.ErrInvalidInput)

// SlotOperationRejection says why a slot operation did nothing. The zero value
// is not a rejection. Comparable and safe for concurrent use.
type SlotOperationRejection int

// The slot operation rejections.
const (
	// SlotOperationNoSlotAvailable: no slot of that level remains in the pool.
	SlotOperationNoSlotAvailable SlotOperationRejection = iota + 1
	// SlotOperationNoExpendedSlot: no slot of that level is expended in the pool.
	SlotOperationNoExpendedSlot
)

// String returns "no_slot_available", "no_expended_slot" or "unknown".
func (r SlotOperationRejection) String() string {
	switch r {
	case SlotOperationNoSlotAvailable:
		return "no_slot_available"
	case SlotOperationNoExpendedSlot:
		return "no_expended_slot"
	}
	return "unknown"
}

// SpellcastingAdvanceRejection says why a class level change was refused. The
// zero value is not a rejection. Comparable and safe for concurrent use.
type SpellcastingAdvanceRejection int

// The advance rejections, in the order they are checked.
const (
	// SpellcastingAdvanceLevelLowered: a class level is lower than before.
	SpellcastingAdvanceLevelLowered SpellcastingAdvanceRejection = iota + 1
	// SpellcastingAdvanceClassRemoved: a class the state holds is missing.
	SpellcastingAdvanceClassRemoved
	// SpellcastingAdvanceLevelsSkipped: the total rises by more than one.
	SpellcastingAdvanceLevelsSkipped
	// SpellcastingAdvanceNoChange: no class level changed.
	SpellcastingAdvanceNoChange
)

// String returns the snake_case name or "unknown".
func (r SpellcastingAdvanceRejection) String() string {
	names := [...]string{"level_lowered", "class_removed", "levels_skipped", "no_change"}
	if r < SpellcastingAdvanceLevelLowered || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// SlotExpenditure is the closed outcome of SpellcastingState.ExpendSlot.
// Variants: SlotExpenditureExpended, SlotExpenditureRejected.
type SlotExpenditure interface{ isSlotExpenditure() }

// SlotExpenditureExpended: a slot was expended.
type SlotExpenditureExpended struct {
	state SpellcastingState
	event SpellSlotExpended
}

// State returns the new state.
func (r SlotExpenditureExpended) State() SpellcastingState { return r.state }

// Event returns the SpellSlotExpended event.
func (r SlotExpenditureExpended) Event() SpellSlotExpended { return r.event }

// SlotExpenditureRejected: no slot was available; the state is unchanged.
type SlotExpenditureRejected struct{ reason SlotOperationRejection }

// Reason returns SlotOperationNoSlotAvailable.
func (r SlotExpenditureRejected) Reason() SlotOperationRejection { return r.reason }

func (SlotExpenditureExpended) isSlotExpenditure() {}
func (SlotExpenditureRejected) isSlotExpenditure() {}

// SlotRecovery is the closed outcome of SpellcastingState.RegainSlot. Variants:
// SlotRecoveryRegained, SlotRecoveryRejected.
type SlotRecovery interface{ isSlotRecovery() }

// SlotRecoveryRegained: slots were restored.
type SlotRecoveryRegained struct {
	state SpellcastingState
	event SpellSlotsRestored
}

// State returns the new state.
func (r SlotRecoveryRegained) State() SpellcastingState { return r.state }

// Event returns the SpellSlotsRestored event.
func (r SlotRecoveryRegained) Event() SpellSlotsRestored { return r.event }

// SlotRecoveryRejected: no slot of that level was expended; the state is unchanged.
type SlotRecoveryRejected struct{ reason SlotOperationRejection }

// Reason returns SlotOperationNoExpendedSlot.
func (r SlotRecoveryRejected) Reason() SlotOperationRejection { return r.reason }

func (SlotRecoveryRegained) isSlotRecovery() {}
func (SlotRecoveryRejected) isSlotRecovery() {}

// ChoiceCount is a number of spells or cantrips still to choose for one class.
// It is comparable.
type ChoiceCount struct {
	class SpellcastingClass
	n     int
}

// Class returns the class.
func (c ChoiceCount) Class() SpellcastingClass { return c.class }

// Count returns the number still to choose, never below zero.
func (c ChoiceCount) Count() int { return c.n }

// SpellcastingLevelChange is the closed outcome of SpellcastingState.Advance.
// Variants: SpellcastingLevelChanged, SpellcastingLevelChangeRejected.
type SpellcastingLevelChange interface{ isSpellcastingLevelChange() }

// SpellcastingLevelChanged: the class levels changed and the slots and lists
// follow them.
type SpellcastingLevelChanged struct {
	state            SpellcastingState
	spellsToChoose   []ChoiceCount
	cantripsToChoose []ChoiceCount
	events           []SpellcastingEvent
}

// State returns the new state.
func (r SpellcastingLevelChanged) State() SpellcastingState { return r.state }

// SpellsToChoose returns a new slice of the level 1+ spells each held class may
// still choose, in class-level order.
func (r SpellcastingLevelChanged) SpellsToChoose() []ChoiceCount {
	return slices.Clone(r.spellsToChoose)
}

// CantripsToChoose returns a new slice of the cantrips each held class may still
// choose, in class-level order.
func (r SpellcastingLevelChanged) CantripsToChoose() []ChoiceCount {
	return slices.Clone(r.cantripsToChoose)
}

// Events returns a new slice of the events; a level change raises none today.
func (r SpellcastingLevelChanged) Events() []SpellcastingEvent { return slices.Clone(r.events) }

// SpellcastingLevelChangeRejected: the new levels were refused; the state is unchanged.
type SpellcastingLevelChangeRejected struct{ reason SpellcastingAdvanceRejection }

// Reason returns the rejection.
func (r SpellcastingLevelChangeRejected) Reason() SpellcastingAdvanceRejection { return r.reason }

func (SpellcastingLevelChanged) isSpellcastingLevelChange()        {}
func (SpellcastingLevelChangeRejected) isSpellcastingLevelChange() {}

// SpellcastingState is the spellcasting of one character, held beside the
// Character aggregate: its class levels, each spellcasting class's lists, the
// Spellcasting and Pact Magic slot pools, and the Concentration and spell in
// progress when there are any (SRD 5.2.1, "Spells", p. 104). Every transition
// returns a new state and never alters the receiver. It holds slices, so compare
// two with reflect.DeepEqual. The zero value is invalid; see IsZero. Safe for
// concurrent use.
type SpellcastingState struct {
	id      Identifier
	levels  ClassLevels
	classes []ClassSpellcasting
	slots   SpellSlots
	pact    PactMagicSlots
	conc    Concentration
	casting CastingInProgress
}

// IsZero reports whether s is the invalid zero value.
func (s SpellcastingState) IsZero() bool { return s.id.IsZero() }

// CharacterID returns the id of the character the state belongs to.
func (s SpellcastingState) CharacterID() Identifier { return s.id }

// ClassLevels returns the class levels the state follows.
func (s SpellcastingState) ClassLevels() ClassLevels { return ClassLevels{entries: s.levels.Entries()} }

// Classes returns a new slice of the held spellcasting classes' lists, in class-level order.
func (s SpellcastingState) Classes() []ClassSpellcasting {
	out := make([]ClassSpellcasting, len(s.classes))
	for i, c := range s.classes {
		out[i] = ClassSpellcasting{class: c.class, level: c.level, extra: c.extra, cantrips: c.Cantrips(), granted: c.GrantedCantrips(), prepared: c.Prepared(), book: c.Spellbook()}
	}
	return out
}

// Class returns the lists of class c and whether the state holds it.
func (s SpellcastingState) Class(c SpellcastingClass) (ClassSpellcasting, bool) {
	for _, h := range s.Classes() {
		if h.class == c {
			return h, true
		}
	}
	return ClassSpellcasting{}, false
}

// Concentration returns the Concentration in effect and whether there is one.
func (s SpellcastingState) Concentration() (Concentration, bool) { return s.conc, !s.conc.IsZero() }

// CastingInProgress returns the spell being cast over time and whether there is one.
func (s SpellcastingState) CastingInProgress() (CastingInProgress, bool) {
	return s.casting, !s.casting.IsZero()
}

// Slots returns a read-only view of one pool: for the Pact Magic pool the counts
// are those of its one Slot Level and zero at every other level; an invalid or
// empty pool reads zero everywhere.
func (s SpellcastingState) Slots(pool SpellSlotPool) SlotPoolView {
	switch pool {
	case PoolSpellcasting:
		return SlotPoolView{max: s.slots.max, rem: s.slots.rem}
	case PoolPactMagic:
		var v SlotPoolView
		if s.pact.HasPactMagic() {
			v.max[s.pact.level-1], v.rem[s.pact.level-1] = s.pact.max, s.pact.rem
		}
		return v
	}
	return SlotPoolView{}
}

// ExpendSlot expends one slot of level l from the pool (SRD 5.2.1, "Spell Slots",
// p. 104). It returns SlotExpenditureRejected, with the state unchanged, when the
// pool has no slot of that level remaining (including a level with a maximum of
// zero, an empty pool, a zero level or an invalid pool). It applies no casting
// rule such as the one-slot-per-turn limit.
func (s SpellcastingState) ExpendSlot(pool SpellSlotPool, l SlotLevel) SlotExpenditure {
	v := s.Slots(pool)
	if v.Remaining(l) == 0 {
		return SlotExpenditureRejected{reason: SlotOperationNoSlotAvailable}
	}
	if pool == PoolSpellcasting {
		s.slots.rem[l.n-1]--
	} else {
		s.pact.rem--
	}
	return SlotExpenditureExpended{state: s, event: SpellSlotExpended{id: s.id, level: l, pool: pool}}
}

// RegainSlot restores up to count expended slots of level l in the pool, never
// above the maximum. It returns SlotRecoveryRejected, with the state unchanged,
// when no slot of that level is expended or count is below 1.
func (s SpellcastingState) RegainSlot(pool SpellSlotPool, l SlotLevel, count int) SlotRecovery {
	n := min(count, s.Slots(pool).Expended(l))
	if n < 1 {
		return SlotRecoveryRejected{reason: SlotOperationNoExpendedSlot}
	}
	ev := SpellSlotsRestored{id: s.id}
	if pool == PoolSpellcasting {
		s.slots.rem[l.n-1] += n
		ev.spellcasting = n
	} else {
		s.pact.rem += n
		ev.pacts = n
	}
	return SlotRecoveryRegained{state: s, event: ev}
}

// RestoreSlots returns the state after a rest: a Long Rest restores every
// expended slot of both pools, a Short Rest only the Pact Magic slots (SRD 5.2.1,
// "Spell Slots", p. 104; "Warlock", Pact Magic, p. 71). It returns the state
// unchanged and no events when nothing was expended or kind is invalid.
func (s SpellcastingState) RestoreSlots(kind RestKind) (SpellcastingState, []SpellcastingEvent) {
	ev := SpellSlotsRestored{id: s.id}
	if kind == LongRest {
		for i := range s.slots.rem {
			ev.spellcasting += s.slots.max[i] - s.slots.rem[i]
		}
		s.slots.rem = s.slots.max
	}
	if kind.Valid() {
		ev.pacts = s.pact.max - s.pact.rem
		s.pact.rem = s.pact.max
	}
	if ev.spellcasting+ev.pacts == 0 {
		return s, nil
	}
	return s, []SpellcastingEvent{ev}
}

// Advance follows the character to the class levels next (SRD 5.2.1, "Level
// Advancement", p. 23). Slot maximums come from the tables; remaining Spellcasting
// slots rise by the rise in the maximum at each level and never fall; the Pact
// Magic pool becomes min(new maximum, old remaining plus any rise in the maximum)
// at the new Slot Level. A class that enters has empty lists. It returns
// SpellcastingLevelChangeRejected, with the state unchanged, for a lowered class
// level, a removed class, a total that rises by more than one or no change, in
// that order of checking.
func (s SpellcastingState) Advance(next ClassLevels) SpellcastingLevelChange {
	if r := advanceRejection(s.levels, next); r != 0 {
		return SpellcastingLevelChangeRejected{reason: r}
	}
	grow := func(oldMax, oldRem, newMax int) int { return min(newMax, oldRem+max(0, newMax-oldMax)) }
	pools := SlotsFor(next)
	out := s
	out.levels = next
	for i := range out.slots.max {
		out.slots.max[i] = pools.spellcasting.max[i]
		out.slots.rem[i] = grow(s.slots.max[i], s.slots.rem[i], out.slots.max[i])
	}
	out.pact = PactMagicSlots{max: pools.pactMagic.max, level: pools.pactMagic.level, rem: grow(s.pact.max, s.pact.rem, pools.pactMagic.max)}
	out.classes = nil
	res := SpellcastingLevelChanged{}
	for _, e := range next.Entries() {
		c, ok := SpellcastingClassFor(e.Class)
		if !ok {
			continue
		}
		held, had := s.Class(c)
		if !had {
			held = ClassSpellcasting{class: c}
		}
		held.level = e.Level
		out.classes = append(out.classes, held)
		res.spellsToChoose = append(res.spellsToChoose, ChoiceCount{class: c, n: held.SpellsToChoose()})
		res.cantripsToChoose = append(res.cantripsToChoose, ChoiceCount{class: c, n: held.CantripsToChoose()})
	}
	res.state = out
	return res
}

// advanceRejection returns the first reason next cannot follow prev, or 0.
func advanceRejection(prev, next ClassLevels) SpellcastingAdvanceRejection {
	lowered, removed, changed := false, false, false
	for _, e := range prev.Entries() {
		n := next.LevelIn(e.Class)
		lowered = lowered || (n > 0 && n < e.Level)
		removed = removed || n == 0
		changed = changed || n != e.Level
	}
	switch {
	case lowered:
		return SpellcastingAdvanceLevelLowered
	case removed:
		return SpellcastingAdvanceClassRemoved
	case next.TotalLevel().n-prev.TotalLevel().n > 1:
		return SpellcastingAdvanceLevelsSkipped
	case !changed && next.Len() == prev.Len():
		return SpellcastingAdvanceNoChange
	}
	return 0
}

// withClass returns a copy of s with the lists of c replaced.
func (s SpellcastingState) withClass(c ClassSpellcasting) SpellcastingState {
	s.classes = slices.Clone(s.classes)
	i := slices.IndexFunc(s.classes, func(h ClassSpellcasting) bool { return h.class == c.class })
	s.classes[i] = c
	return s
}

// createState builds the initial state for the use case CreateSpellcasting: it
// returns the state, or the missing spells (which take precedence), or the
// problems of every class, or a command error. Every held spellcasting class gets
// an entry, empty when no setup names it.
func createState(id Identifier, levels ClassLevels, setups []ClassSpellcastingSetup, spells map[content.SpellID]content.Spell) (SpellcastingState, []content.MissingContent, []PreparationProblem, error) {
	byClass := map[SpellcastingClass]ClassSpellcastingSetup{}
	var missing []content.MissingContent
	for _, set := range setups {
		if _, dup := byClass[set.Class]; dup || !set.Class.Valid() || set.ExtraCantrips < 0 {
			return SpellcastingState{}, nil, nil, invalidInput("create spellcasting: invalid or repeated class setup")
		}
		if levels.LevelIn(set.Class.ClassID()) == 0 {
			return SpellcastingState{}, nil, nil, ErrClassNotHeld
		}
		for _, g := range set.Granted {
			if levels.LevelIn(g.class.ClassID()) == 0 {
				return SpellcastingState{}, nil, nil, ErrClassNotHeld
			}
			if g.class != set.Class {
				return SpellcastingState{}, nil, nil, invalidInput("create spellcasting: granted cantrip of another class")
			}
		}
		byClass[set.Class] = set
		missing = append(missing, setupMissing(spells, set)...)
	}
	if len(missing) > 0 {
		return SpellcastingState{}, dedupMissing(missing), nil, nil
	}
	st := SpellcastingState{id: id, levels: levels}
	pools := SlotsFor(levels)
	st.slots, st.pact = pools.spellcasting, pools.pactMagic
	var problems []PreparationProblem
	for _, e := range levels.Entries() {
		c, ok := SpellcastingClassFor(e.Class)
		if !ok {
			continue
		}
		set := byClass[c]
		set.Class = c
		cs := buildClass(set, e.Level)
		problems = append(problems, cs.problems(spells)...)
		st.classes = append(st.classes, cs)
	}
	if len(problems) > 0 {
		return SpellcastingState{}, nil, problems, nil
	}
	return st, nil, nil, nil
}

// dedupMissing removes repeats of the same missing entry, keeping first-seen order.
func dedupMissing(in []content.MissingContent) []content.MissingContent {
	var out []content.MissingContent
	for _, m := range in {
		if !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	return out
}
