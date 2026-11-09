package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// maxTakenFeats bounds the feats a TakenFeats holds; a character takes far
// fewer (SRD 5.2.1, "Level Advancement", p. 23).
const maxTakenFeats = 64

// TakenFeat is a feat a character took, where it came from, the choices made and
// the state of its limited use (SRD 5.2.1, "Parts of a Feat", p. 87). It holds the
// choices of Magic Initiate, which hold a slice, so compare two with Equal. The
// zero TakenFeat is invalid.
type TakenFeat struct {
	feat     content.FeatID
	category content.FeatCategory
	source   FeatSource
	choices  FeatChoices
	use      FeatUseState // nil: the feat has no limited use
}

// NewTakenFeat returns the feat id taken from source with choices. A feat with a
// limited use (Magic Initiate, Boon of Fate) starts with FeatUseIsAvailable.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid id,
// category or source, for a feat with no behaviour (only the 17 SRD feats have
// one) and for choices of another shape than the feat takes (table C). It never
// panics.
func NewTakenFeat(id content.FeatID, category content.FeatCategory, source FeatSource, choices FeatChoices) (TakenFeat, error) {
	shape, ok := shapeOf(id)
	if !ok || !category.Valid() || !validFeatSource(source) || !shape.matches(choices) {
		return TakenFeat{}, invalidInput("taken feat")
	}
	t := TakenFeat{feat: id, category: category, source: source, choices: choices}
	if shape.renewedBy.mask != 0 {
		t.use = FeatUseIsAvailable{}
	}
	return t, nil
}

// Feat returns the feat id.
func (t TakenFeat) Feat() content.FeatID { return t.feat }

// Category returns the category of the feat.
func (t TakenFeat) Category() content.FeatCategory { return t.category }

// Source returns where the feat came from.
func (t TakenFeat) Source() FeatSource { return t.source }

// Choices returns the choices made when the feat was taken.
func (t TakenFeat) Choices() FeatChoices { return t.choices }

// Use returns the state of the feat's limited use, or nil when it has none.
func (t TakenFeat) Use() FeatUseState { return t.use }

// IsZero reports whether t is the invalid zero TakenFeat.
func (t TakenFeat) IsZero() bool { return t.feat == "" }

// Equal reports whether t and o are the same feat, source, choices and state.
func (t TakenFeat) Equal(o TakenFeat) bool {
	return t.feat == o.feat && t.category == o.category && t.source == o.source && t.use == o.use && equalFeatChoices(t.choices, o.choices)
}

// TakenFeats is the feats a character holds, in the order taken. It lives
// beside the Character, as SpellcastingState does. It is immutable and copied on
// construction and on read; compare two with Equal. The zero value is the empty
// set. Safe for concurrent use.
type TakenFeats struct{ taken []TakenFeat }

// NewTakenFeats returns the set of taken, in order.
//
// It returns an error matching srderr.ErrInvalidInput for a zero TakenFeat, for
// more than 64 feats and for two Magic Initiate entries with the same spell
// list. It cannot check repeatability, which needs content. It never panics.
func NewTakenFeats(taken ...TakenFeat) (TakenFeats, error) {
	if len(taken) > maxTakenFeats {
		return TakenFeats{}, invalidInput("too many taken feats")
	}
	var lists []content.ClassID
	for _, t := range taken {
		if t.IsZero() {
			return TakenFeats{}, invalidInput("zero taken feat")
		}
		if mi, ok := t.choices.(MagicInitiateChoices); ok {
			if slices.Contains(lists, mi.list) {
				return TakenFeats{}, invalidInput("two magic initiate feats with one spell list")
			}
			lists = append(lists, mi.list)
		}
	}
	return TakenFeats{taken: slices.Clone(taken)}, nil
}

// All returns a new slice of the feats held, in the order taken.
func (f TakenFeats) All() []TakenFeat { return slices.Clone(f.taken) }

// Len returns the number of feats held.
func (f TakenFeats) Len() int { return len(f.taken) }

// Holds reports whether the feat id is held, from any source.
func (f TakenFeats) Holds(id content.FeatID) bool { return f.indexOf(id) >= 0 }

// TakenFrom returns the number of held feats taken from source, matching on the
// kind and the names of the source.
func (f TakenFeats) TakenFrom(source FeatSource) int {
	n := 0
	for _, t := range f.taken {
		if t.source == source {
			n++
		}
	}
	return n
}

// UseStateOf returns the state of the limited use of feat id: FeatUseIsAvailable
// when any instance has it available, FeatUseIsExpended when every instance with
// a use has it expended, and nil when the feat is not held or has no limited use.
func (f TakenFeats) UseStateOf(id content.FeatID) FeatUseState {
	var state FeatUseState
	for _, t := range f.taken {
		if t.feat != id || t.use == nil {
			continue
		}
		if _, ok := t.use.(FeatUseIsAvailable); ok {
			return t.use
		}
		state = t.use
	}
	return state
}

// ProficiencyGrants returns the proficiencies every held Skilled feat gives.
func (f TakenFeats) ProficiencyGrants() ProficiencyGrants {
	var g ProficiencyGrants
	for _, t := range f.taken {
		if s, ok := t.choices.(SkilledChoices); ok {
			g = g.with(s)
		}
	}
	return g
}

// Replacing returns the feats with the first instance of feat old removed, with
// its choices and use state, and then new added, unless new is another held feat
// that is not repeatable (repeatable is the Repeatable flag of new's feat). The
// old feat no longer counts, so replacing a feat by itself is accepted. It checks
// no grant, prerequisite or choice. The rejection is nil on success, FeatNotHeld
// when old is not held, or FeatNotRepeatable. The receiver is unchanged. It never
// panics.
func (f TakenFeats) Replacing(old content.FeatID, repl TakenFeat, repeatable bool) (TakenFeats, FeatRejection) {
	i := f.indexOf(old)
	if i < 0 {
		return f, FeatNotHeld{Feat: old}
	}
	without := f.without(i)
	if !repeatable && without.Holds(repl.feat) {
		return f, FeatNotRepeatable{Feat: repl.feat}
	}
	return without.with(repl), nil
}

// Equal reports whether f and o hold the same feats in the same order.
func (f TakenFeats) Equal(o TakenFeats) bool {
	return slices.EqualFunc(f.taken, o.taken, TakenFeat.Equal)
}

func (f TakenFeats) indexOf(id content.FeatID) int {
	return slices.IndexFunc(f.taken, func(t TakenFeat) bool { return t.feat == id })
}

func (f TakenFeats) without(i int) TakenFeats {
	return TakenFeats{taken: slices.Delete(slices.Clone(f.taken), i, i+1)}
}

func (f TakenFeats) with(t TakenFeat) TakenFeats {
	return TakenFeats{taken: append(slices.Clone(f.taken), t)}
}

// magicInitiate returns the index of the Magic Initiate instance with spell
// list, or -1.
func (f TakenFeats) magicInitiate(list content.ClassID) int {
	return slices.IndexFunc(f.taken, func(t TakenFeat) bool {
		mi, ok := t.choices.(MagicInitiateChoices)
		return ok && mi.list == list
	})
}

// ProficiencyGrants is the skills and tools a feat gives, for the host to merge
// into its proficiencies (SRD 5.2.1, "Skilled", p. 87). A pick the character
// already has is listed all the same. It is immutable and copied on read. The
// zero value grants nothing. Safe for concurrent use.
type ProficiencyGrants struct {
	skills []content.Skill
	tools  []content.ItemID
}

// Skills returns a new slice of the skills granted.
func (g ProficiencyGrants) Skills() []content.Skill { return slices.Clone(g.skills) }

// Tools returns a new slice of the tools granted.
func (g ProficiencyGrants) Tools() []content.ItemID { return slices.Clone(g.tools) }

// IsZero reports whether nothing is granted.
func (g ProficiencyGrants) IsZero() bool { return len(g.skills) == 0 && len(g.tools) == 0 }

func (g ProficiencyGrants) with(c SkilledChoices) ProficiencyGrants {
	out := ProficiencyGrants{skills: slices.Clone(g.skills), tools: slices.Clone(g.tools)}
	for _, p := range c.picks {
		switch v := p.(type) {
		case SkilledSkill:
			out.skills = append(out.skills, v.Skill)
		case SkilledTool:
			out.tools = append(out.tools, v.Tool)
		}
	}
	return out
}
