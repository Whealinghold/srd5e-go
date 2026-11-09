package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

func casterStats() character.SpellcastingStats {
	return character.Stats(character.CasterWizard, modOf(18), pbOf(5))
}

func storedSpell(t *testing.T, spell string, slot int) character.StoredSpell {
	t.Helper()
	st := casterStats()
	return must(character.NewStoredSpell(sp(spell), lvl(slot), st.SaveDC(), st.AttackBonus(), ability.Intelligence))
}

func ringWith(t *testing.T, slots ...int) character.SpellStorage {
	t.Helper()
	var spells []character.StoredSpell
	for i, s := range slots {
		spells = append(spells, storedSpell(t, "spell_"+string(rune('a'+i)), s))
	}
	return must(character.NewSpellStorage(spells))
}

func intoRing(slot int) character.SpellCastIntoRing {
	st := casterStats()
	return character.SpellCastIntoRing{Spell: sp("shield"), SlotLevel: lvl(slot), SaveDC: st.SaveDC(), AttackBonus: st.AttackBonus(), Ability: ability.Intelligence}
}

// Requirement: doc/specs/magic-item-effects.md FR-048, FR-055 (SRD 5.2.1, "Ring of Spell Storing", p. 239) - the storage holds at most 5 levels of
// stored spells; a stored spell needs a valid spell and ability and a slot level of 1 to 5, holds the caster's numbers as supplied, and two copies of
// a spell are allowed.
// Path: unhappy (boundary).
func TestSpellStorageConstruction(t *testing.T) {
	t.Parallel()
	if ring := ringWith(t, 2, 3); ring.UsedLevels() != 5 || len(ring.Spells()) != 2 {
		t.Errorf("exactly full = %d", ring.UsedLevels())
	}
	if ring := ringWith(t); ring.UsedLevels() != 0 || len(ring.Spells()) != 0 {
		t.Error("empty")
	}
	if _, err := character.NewSpellStorage([]character.StoredSpell{storedSpell(t, "a", 3), storedSpell(t, "b", 3)}); err == nil {
		t.Error("6 levels accepted")
	} else {
		wantInvalid(t, err, "6 levels")
	}
	wantInvalid(t, second(character.NewSpellStorage([]character.StoredSpell{{}})), "zero stored spell")
	st := casterStats()
	_, e1 := character.NewStoredSpell("", lvl(1), st.SaveDC(), st.AttackBonus(), ability.Intelligence)
	_, e2 := character.NewStoredSpell(sp("a"), lvl(0), st.SaveDC(), st.AttackBonus(), ability.Intelligence)
	_, e3 := character.NewStoredSpell(sp("a"), lvl(6), st.SaveDC(), st.AttackBonus(), ability.Intelligence)
	_, e4 := character.NewStoredSpell(sp("a"), lvl(1), st.SaveDC(), st.AttackBonus(), 0)
	for i, err := range []error{e1, e2, e3, e4} {
		wantInvalid(t, err, "stored case "+string(rune('a'+i)))
	}
	s := must(character.NewStoredSpell(sp("a"), lvl(5), st.SaveDC(), st.AttackBonus(), ability.Wisdom))
	if s.SpellID() != sp("a") || s.SlotLevel() != lvl(5) || s.SaveDC() != st.SaveDC() || s.AttackBonus() != st.AttackBonus() || s.Ability() != ability.Wisdom {
		t.Errorf("accessors: %+v", s)
	}
	two := ringWith(t, 1, 1)
	if two.Spells()[0].SpellID() == two.Spells()[1].SpellID() {
		t.Log("distinct ids in the helper")
	}
	dup := must(character.NewSpellStorage([]character.StoredSpell{storedSpell(t, "same", 1), storedSpell(t, "same", 1)}))
	if dup.UsedLevels() != 2 {
		t.Error("two copies")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-049 - Initial accepts spells whose slot levels sum exactly to the rolled levels and returns
// SpellStorageRejected(levelsDoNotMatch) otherwise; levels outside 0 to 5 are invalid input.
// Path: unhappy (boundary).
func TestSpellStorageInitial(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		levels int
		slots  []int
		ok     bool
	}{
		{"zero levels, no spells", 0, nil, true},
		{"five levels in two spells", 5, []int{2, 3}, true},
		{"one level short", 5, []int{2, 2}, false},
		{"one level over", 4, []int{2, 3}, false},
		{"levels but no spells", 3, nil, false},
		{"spells but zero levels", 0, []int{1}, false},
	} {
		var spells []character.StoredSpell
		for i, s := range c.slots {
			spells = append(spells, storedSpell(t, "s"+string(rune('a'+i)), s))
		}
		res := must((character.SpellStorage{}).Initial(c.levels, spells))
		switch r := res.(type) {
		case character.SpellStorageInitialised:
			if !c.ok || r.Storage().UsedLevels() != c.levels {
				t.Errorf("%s: initialised %d", c.name, r.Storage().UsedLevels())
			}
		case character.SpellStorageRejected:
			if c.ok || r.Reason() != character.SpellStorageLevelsDoNotMatch {
				t.Errorf("%s: rejected %v", c.name, r.Reason())
			}
		}
	}
	wantInvalid(t, second((character.SpellStorage{}).Initial(-1, nil)), "levels -1")
	wantInvalid(t, second((character.SpellStorage{}).Initial(6, nil)), "levels 6")
	wantInvalid(t, second((character.SpellStorage{}).Initial(1, []character.StoredSpell{{}})), "zero stored spell")
}

// Requirement: doc/specs/magic-item-effects.md FR-050, FR-051 - every Store row of table "Ring of Spell Storing": a spell fits when used plus its slot
// level is at most 5 (the slot level decides, not the spell's), otherwise it is expended and the ring is unchanged; slot level 0 or 6 is not storable.
// Path: per case.
func TestSpellStorageStore(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		slots []int
		slot  int
		want  string
		used  int
	}{
		{"fits exactly", []int{2}, 3, "stored", 5},
		{"empty ring, level 1", nil, 1, "stored", 1},
		{"empty ring, level 5", nil, 5, "stored", 5},
		{"one too many", []int{3}, 3, "expended", 3},
		{"full ring, level 1", []int{5}, 1, "expended", 5},
		{"upcast uses the slot level", nil, 3, "stored", 3},
		{"cantrip", nil, 0, "notstorable", 0},
		{"level 6", nil, 6, "notstorable", 0},
		{"level 6 on a full ring", []int{5}, 6, "notstorable", 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ring := ringWith(t, c.slots...)
			before := ring.Spells()
			got := must(ring.Store(holder, inst("ring-1"), intoRing(c.slot)))
			switch r := got.(type) {
			case character.SpellStorageStored:
				ev := r.Event()
				if c.want != "stored" || r.Storage().UsedLevels() != c.used || ev.SlotLevel() != lvl(c.slot) || ev.SpellID() != sp("shield") || ev.CharacterID() != holder || ev.InstanceID() != inst("ring-1") {
					t.Errorf("stored %d %+v", r.Storage().UsedLevels(), ev)
				}
				last := r.Storage().Spells()[len(r.Storage().Spells())-1]
				if last.SaveDC() != casterStats().SaveDC() || last.Ability() != ability.Intelligence {
					t.Error("numbers not kept")
				}
			case character.SpellStorageSpellExpended:
				if c.want != "expended" || r.Spell().SlotLevel != lvl(c.slot) {
					t.Errorf("expended %+v", r.Spell())
				}
			case character.SpellStorageNotStorable:
				if c.want != "notstorable" || r.SlotLevel() != lvl(c.slot) {
					t.Errorf("not storable %v", r.SlotLevel())
				}
			}
			if c.want != "stored" && !ring.Equal(ringWith(t, c.slots...)) || len(ring.Spells()) != len(before) {
				t.Error("the ring changed")
			}
		})
	}
	bad := intoRing(1)
	bad.Spell = ""
	wantInvalid(t, second(ringWith(t).Store(holder, inst("r"), bad)), "spell")
	bad = intoRing(1)
	bad.Ability = 0
	wantInvalid(t, second(ringWith(t).Store(holder, inst("r"), bad)), "ability")
	wantInvalid(t, second(ringWith(t).Store(character.Identifier{}, inst("r"), intoRing(1))), "holder")
	wantInvalid(t, second(ringWith(t).Store(holder, "", intoRing(1))), "instance")
}

// Requirement: doc/specs/magic-item-effects.md FR-052, FR-053, FR-054 - Cast removes the spell at the position, frees its levels and carries the
// original caster's numbers; only fullMagical may cast (a refusal leaves the ring unchanged) and a position outside the list is NoSuchSpell.
// Path: per case.
func TestSpellStorageCast(t *testing.T) {
	t.Parallel()
	ring := ringWith(t, 2, 3)
	r, ok := must(ring.Cast(holder, inst("ring-1"), 0, character.AccessFullMagical)).(character.StoredSpellCast)
	if !ok {
		t.Fatal("not cast")
	}
	if r.Storage().UsedLevels() != 3 || len(r.Storage().Spells()) != 1 || r.Spell().SlotLevel() != lvl(2) || r.Spell().SaveDC() != casterStats().SaveDC() ||
		r.Spell().AttackBonus() != casterStats().AttackBonus() || r.Spell().Ability() != ability.Intelligence || r.Event().SlotLevel() != lvl(2) ||
		r.Event().SpellID() != r.Spell().SpellID() || r.Event().CharacterID() != holder || r.Event().InstanceID() != inst("ring-1") {
		t.Errorf("cast = %+v", r)
	}
	last := must(ring.Cast(holder, inst("ring-1"), 1, character.AccessFullMagical)).(character.StoredSpellCast)
	if last.Storage().UsedLevels() != 2 || last.Spell().SlotLevel() != lvl(3) {
		t.Errorf("last = %+v", last)
	}
	if ring.UsedLevels() != 5 || len(ring.Spells()) != 2 {
		t.Error("the ring changed")
	}
	for _, access := range []character.MagicalPropertiesAccessLevel{character.AccessNonmagicalBenefitsOnly, character.AccessNoMagic, 0} {
		nu, ok := must(ring.Cast(holder, inst("ring-1"), 0, access)).(character.SpellStorageNotUsable)
		if !ok || nu.Access() != access {
			t.Errorf("access %d = %+v", access, nu)
		}
	}
	for _, idx := range []int{-1, 2, 99} {
		ns, ok := must(ring.Cast(holder, inst("ring-1"), idx, character.AccessFullMagical)).(character.SpellStorageNoSuchSpell)
		if !ok || ns.Index() != idx {
			t.Errorf("index %d = %+v", idx, ns)
		}
	}
	if _, ok := must(ringWith(t).Cast(holder, inst("r"), 0, character.AccessNoMagic)).(character.SpellStorageNotUsable); !ok {
		t.Error("access is checked before the position")
	}
	wantInvalid(t, second(ring.Cast(character.Identifier{}, inst("r"), 0, character.AccessFullMagical)), "holder")
	wantInvalid(t, second(ring.Cast(holder, "", 0, character.AccessFullMagical)), "instance")
}

// Requirement: doc/specs/magic-item-effects.md FR-056 - SpellStorage is immutable: the exposed list is a copy, the constructor copies its input, and
// equal contents are equal.
// Path: unhappy (immutability).
func TestSpellStorageImmutability(t *testing.T) {
	t.Parallel()
	in := []character.StoredSpell{storedSpell(t, "a", 1), storedSpell(t, "b", 2)}
	ring := must(character.NewSpellStorage(in))
	in[0] = character.StoredSpell{}
	list := ring.Spells()
	list[1] = character.StoredSpell{}
	if want := must(character.NewSpellStorage([]character.StoredSpell{storedSpell(t, "a", 1), storedSpell(t, "b", 2)})); ring.UsedLevels() != 3 || !ring.Equal(want) {
		t.Error("the ring aliases its input or its list")
	}
	if ring.Equal(ringWith(t, 1)) || ring.Equal(ringWith(t, 2, 1)) || !(character.SpellStorage{}).Equal(ringWith(t)) {
		t.Error("equality")
	}
	stored := must(ring.Store(holder, inst("r"), intoRing(1))).(character.SpellStorageStored)
	if len(ring.Spells()) != 2 || len(stored.Storage().Spells()) != 3 {
		t.Error("Store changed the receiver")
	}
}
