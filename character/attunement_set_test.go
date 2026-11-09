package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/magic-item-rules.md FR-048, FR-049 and table "Attunement limit" - the limit is at least 3; Standard is 3.
// Path: unhappy (boundary).
func TestAttunementLimit(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1, 0, 2} {
		if l, err := character.NewAttunementLimit(n); !errors.Is(err, srderr.ErrInvalidInput) || l != 0 {
			t.Errorf("NewAttunementLimit(%d) = %v, %v", n, l, err)
		}
	}
	for _, n := range []int{3, 4, 10} {
		if l, err := character.NewAttunementLimit(n); err != nil || int(l) != n {
			t.Errorf("NewAttunementLimit(%d) = %v, %v", n, l, err)
		}
	}
	if got := character.AttunementLimit(0).Standard(); got != 3 {
		t.Errorf("Standard = %d", got)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-038 - NewAttunableItem refuses NoAttunement with an *AttunableItemNotAttunable
// and accepts both requiring variants; invalid ids and a nil attunement are invalid input.
// Path: per case.
func TestNewAttunableItem(t *testing.T) {
	t.Parallel()
	required, by := content.AttunementRequired{}, requiredBy(t, "a Wizard")
	for _, a := range []content.Attunement{required, by} {
		it, err := character.NewAttunableItem(mid("ring"), inst("r1"), a)
		if err != nil || it.ItemID() != "ring" || it.InstanceID() != "r1" || it.Attunement() != a || it.IsZero() {
			t.Errorf("%#v: %+v, %v", a, it, err)
		}
	}
	it, err := character.NewAttunableItem(mid("ring"), inst("r1"), content.NoAttunement{})
	var rej *character.AttunableItemNotAttunable
	if !errors.As(err, &rej) || !errors.Is(err, srderr.ErrInvalidInput) || !it.IsZero() {
		t.Errorf("NoAttunement: %+v, %v", it, err)
	}
	for name, run := range map[string]func() (character.AttunableItem, error){
		"nil": func() (character.AttunableItem, error) {
			return character.NewAttunableItem(mid("ring"), inst("r1"), nil)
		},
		"blank instance": func() (character.AttunableItem, error) { return character.NewAttunableItem(mid("ring"), "", required) },
		"invalid id":     func() (character.AttunableItem, error) { return character.NewAttunableItem("", inst("r1"), required) },
	} {
		if _, err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustAttunableItem(NoAttunement) did not panic")
		}
	}()
	character.MustAttunableItem(mid("ring"), inst("r1"), content.NoAttunement{})
}

// Requirement: doc/specs/magic-item-rules.md FR-061 - AttunementSet is immutable and equal when built from equal items;
// repeated instances and invalid items are rejected; a copy of the items cannot corrupt it.
// Path: unhappy (immutability).
func TestAttunementSetImmutability(t *testing.T) {
	t.Parallel()
	a, b := attunedSet(t, "ring", "r1", "cloak", "c1"), attunedSet(t, "cloak", "c1", "ring", "r1")
	if !a.Equal(b) || a.Len() != 2 || !a.IsAttunedTo(inst("r1")) || a.IsAttunedTo(inst("zz")) {
		t.Errorf("sets differ: %v vs %v", a.Items(), b.Items())
	}
	items := a.Items()
	items[0] = character.AttunedItem{}
	if !a.Equal(b) {
		t.Error("mutating the copy changed the set")
	}
	if attunedSet(t).Len() != 0 || a.Equal(attunedSet(t, "ring", "r1")) {
		t.Error("Equal/Len")
	}
	if _, err := character.NewAttunementSet([]character.AttunedItem{items[1], items[1]}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("repeated instance: %v", err)
	}
	if _, err := character.NewAttunementSet([]character.AttunedItem{{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero item: %v", err)
	}
	if _, err := character.NewAttunedItem("", inst("x")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewAttunedItem: %v", err)
	}
	if it := items[1]; it.ItemID() == "" || it.InstanceID() == "" {
		t.Errorf("accessors: %+v", it)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-055 - another creature attuning ends the first creature's Attunement with
// anotherCreatureAttuned; for an instance not in the set the same set and no event.
// Path: unhappy (rule interaction).
func TestEndBecauseAnotherAttuned(t *testing.T) {
	t.Parallel()
	s := attunedSet(t, "ring", "r1", "cloak", "c1")
	next, ev, ok := s.EndBecauseAnotherAttuned(holder, inst("r1"))
	if !ok || next.IsAttunedTo(inst("r1")) || next.Len() != 1 || ev.Reason() != character.AttunementEndedAnotherCreatureAttuned || ev.CharacterID() != holder || ev.ItemID() != "ring" || ev.InstanceID() != "r1" {
		t.Errorf("ended: %+v %+v %v", next.Items(), ev, ok)
	}
	if !s.IsAttunedTo(inst("r1")) {
		t.Error("input changed")
	}
	same, _, ok := s.EndBecauseAnotherAttuned(holder, inst("zz"))
	if ok || !same.Equal(s) {
		t.Errorf("absent instance: %v", ok)
	}
}
