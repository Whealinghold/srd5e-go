package character

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// AttunementLimit is the number of magic items a creature may be attuned to at
// once; at least 3. The zero value is invalid. Comparable.
type AttunementLimit int

// Standard returns the standard limit, 3 (SRD 5.2.1, "No More Than Three Items", p. 102).
func (AttunementLimit) Standard() AttunementLimit {
	return AttunementLimit(content.AttunementFacts{}.StandardLimit())
}

// NewAttunementLimit returns n as a limit. It returns an error matching
// srderr.ErrInvalidInput for n below 3. It never panics.
func NewAttunementLimit(n int) (AttunementLimit, error) {
	if n < (content.AttunementFacts{}).StandardLimit() {
		return 0, invalidInput("attunement limit below 3")
	}
	return AttunementLimit(n), nil
}

// AttunableItemNotAttunable is the error NewAttunableItem returns for an item
// that does not require Attunement. It matches srderr.ErrInvalidInput.
//
//nolint:errname // the name is the spec's (FR-119); it is an error with typed data
type AttunableItemNotAttunable struct{}

// Error renders the error. The text is not a contract.
func (*AttunableItemNotAttunable) Error() string {
	return "character: item does not require attunement"
}

// Unwrap returns srderr.ErrInvalidInput.
func (*AttunableItemNotAttunable) Unwrap() error { return srderr.ErrInvalidInput }

// AttunableItem is one item copy that requires Attunement. Immutable; the zero
// value is invalid. Comparable.
type AttunableItem struct {
	item       content.MagicItemID
	instance   MagicItemInstanceID
	attunement content.Attunement
}

// NewAttunableItem returns the item. It returns an *AttunableItemNotAttunable
// for content.NoAttunement and an error matching srderr.ErrInvalidInput for a
// blank id, a blank instance or a nil attunement. It never panics.
func NewAttunableItem(item content.MagicItemID, instance MagicItemInstanceID, attunement content.Attunement) (AttunableItem, error) {
	switch attunement.(type) {
	case content.NoAttunement:
		return AttunableItem{}, &AttunableItemNotAttunable{}
	case content.AttunementRequired, content.AttunementRequiredBy:
	default:
		return AttunableItem{}, invalidInput("attunable item: nil or foreign attunement")
	}
	if !item.Valid() || !instance.Valid() {
		return AttunableItem{}, invalidInput("attunable item: invalid id or instance")
	}
	return AttunableItem{item: item, instance: instance, attunement: attunement}, nil
}

// MustAttunableItem is NewAttunableItem for literals; it panics with the error.
func MustAttunableItem(item content.MagicItemID, instance MagicItemInstanceID, attunement content.Attunement) AttunableItem {
	a, err := NewAttunableItem(item, instance, attunement)
	if err != nil {
		panic(err)
	}
	return a
}

// ItemID returns the item.
func (a AttunableItem) ItemID() content.MagicItemID { return a.item }

// InstanceID returns the copy.
func (a AttunableItem) InstanceID() MagicItemInstanceID { return a.instance }

// Attunement returns the requirement.
func (a AttunableItem) Attunement() content.Attunement { return a.attunement }

// IsZero reports whether a is the invalid zero AttunableItem.
func (a AttunableItem) IsZero() bool { return a.attunement == nil }

// AttunedItem is an item copy a creature is attuned to. Comparable.
type AttunedItem struct {
	item     content.MagicItemID
	instance MagicItemInstanceID
}

// NewAttunedItem returns the pair. It returns an error matching
// srderr.ErrInvalidInput for an invalid id or a blank instance.
func NewAttunedItem(item content.MagicItemID, instance MagicItemInstanceID) (AttunedItem, error) {
	if !item.Valid() || !instance.Valid() {
		return AttunedItem{}, invalidInput("attuned item: invalid id or instance")
	}
	return AttunedItem{item: item, instance: instance}, nil
}

// ItemID returns the item.
func (a AttunedItem) ItemID() content.MagicItemID { return a.item }

// InstanceID returns the copy.
func (a AttunedItem) InstanceID() MagicItemInstanceID { return a.instance }

// AttunementSet is the items a creature is attuned to, ordered by instance id.
// Immutable: every operation returns a new set. The zero value is the empty
// set. Compare with Equal.
type AttunementSet struct{ items []AttunedItem }

// NewAttunementSet returns the set of items. It returns an error matching
// srderr.ErrInvalidInput for an invalid item or a repeated instance.
func NewAttunementSet(items []AttunedItem) (AttunementSet, error) {
	s := AttunementSet{}
	for _, it := range items {
		if !it.item.Valid() || !it.instance.Valid() || s.IsAttunedTo(it.instance) {
			return AttunementSet{}, invalidInput("attunement set: invalid item or repeated instance")
		}
		s = s.with(it)
	}
	return s, nil
}

func (s AttunementSet) with(it AttunedItem) AttunementSet {
	out := append(slices.Clone(s.items), it)
	slices.SortFunc(out, func(a, b AttunedItem) int { return strings.Compare(string(a.instance), string(b.instance)) })
	return AttunementSet{items: out}
}

func (s AttunementSet) without(instance MagicItemInstanceID) AttunementSet {
	return AttunementSet{items: slices.DeleteFunc(slices.Clone(s.items), func(a AttunedItem) bool { return a.instance == instance })}
}

// Items returns a new slice of the attuned items, ordered by instance id.
func (s AttunementSet) Items() []AttunedItem { return slices.Clone(s.items) }

// Len returns the number of attuned items.
func (s AttunementSet) Len() int { return len(s.items) }

// IsAttunedTo reports whether the set holds the instance.
func (s AttunementSet) IsAttunedTo(instance MagicItemInstanceID) bool {
	return slices.ContainsFunc(s.items, func(a AttunedItem) bool { return a.instance == instance })
}

func (s AttunementSet) find(instance MagicItemInstanceID) (AttunedItem, bool) {
	i := slices.IndexFunc(s.items, func(a AttunedItem) bool { return a.instance == instance })
	if i < 0 {
		return AttunedItem{}, false
	}
	return s.items[i], true
}

// Equal reports whether both sets hold the same items.
func (s AttunementSet) Equal(o AttunementSet) bool { return slices.Equal(s.items, o.items) }

// EndBecauseAnotherAttuned returns the set without the instance, because another
// creature attuned to it, and the event. For an instance not in the set it
// returns the same set and false (SRD 5.2.1, "Ending Attunement", pp. 102-103).
func (s AttunementSet) EndBecauseAnotherAttuned(id Identifier, instance MagicItemInstanceID) (AttunementSet, MagicItemAttunementEnded, bool) {
	it, ok := s.find(instance)
	if !ok {
		return s, MagicItemAttunementEnded{}, false
	}
	return s.without(instance), MagicItemAttunementEnded{id: id, item: it.item, instance: instance, reason: AttunementEndedAnotherCreatureAttuned}, true
}
