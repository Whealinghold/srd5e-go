package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// PreparationRejectionKind is one reason a spell list is refused. The constants
// are in the fixed order a PreparationRejection lists them. The zero value is
// not a kind. Comparable and safe for concurrent use.
type PreparationRejectionKind int

// The preparation rejection kinds, in reporting order (SRD 5.2.1, "Preparing
// Spells", p. 104; the class "Spellcasting" entries).
const (
	// PreparationNotOnClassList: the spell is not on the class's spell list.
	PreparationNotOnClassList PreparationRejectionKind = iota + 1
	// PreparationSpellLevelTooHigh: the spell is above the highest preparable level.
	PreparationSpellLevelTooHigh
	// PreparationTooManyPrepared: more spells than the prepared limit.
	PreparationTooManyPrepared
	// PreparationDuplicateSpell: a spell is listed twice.
	PreparationDuplicateSpell
	// PreparationCantripNotPreparable: a cantrip is in the prepared list.
	PreparationCantripNotPreparable
	// PreparationAlwaysPreparedNotReplaceable: an always-prepared spell was dropped.
	PreparationAlwaysPreparedNotReplaceable
	// PreparationChangeNotAllowedNow: a replacement on a trigger the class lacks.
	PreparationChangeNotAllowedNow
	// PreparationTooManyReplacements: more replacements than the class may make.
	PreparationTooManyReplacements
	// PreparationNotInSpellbook: a Wizard prepared a spell outside the spellbook.
	PreparationNotInSpellbook
	// PreparationTooManyCantrips: more cantrips than the class knows.
	PreparationTooManyCantrips
	// PreparationNotACantrip: a chosen or granted cantrip is not level 0.
	PreparationNotACantrip
)

// String returns the snake_case name of the kind (the srd5e.character.rejection
// value) or "unknown".
func (k PreparationRejectionKind) String() string {
	names := [...]string{"not_on_class_list", "spell_level_too_high", "too_many_prepared", "duplicate_spell", "cantrip_not_preparable",
		"always_prepared_not_replaceable", "change_not_allowed_now", "too_many_replacements", "not_in_spellbook", "too_many_cantrips", "not_a_cantrip"}
	if k < PreparationNotOnClassList || int(k) > len(names) {
		return "unknown"
	}
	return names[k-1]
}

// PreparationProblem is one problem with a spell list: its kind, the spell it
// concerns (zero for a count problem) and the class. It is an error matching
// srderr.ErrInvalidInput and another *PreparationProblem with the same fields.
// Construct it with a keyed literal.
//
//nolint:errname // a problem value of a spell list, named by the spec
type PreparationProblem struct {
	Kind  PreparationRejectionKind
	Spell content.SpellID
	Class SpellcastingClass
}

// Error renders the problem. The text is not a contract.
func (p *PreparationProblem) Error() string { return "character: spell list: " + p.Kind.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (p *PreparationProblem) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *PreparationProblem with the same fields.
func (p *PreparationProblem) Is(target error) bool {
	t, ok := target.(*PreparationProblem)
	return ok && t != nil && p != nil && *t == *p
}

// PreparationRejection lists every problem found with a spell list, in the fixed
// order of PreparationRejectionKind and then the order the spells were given.
// The state it was refused for is unchanged. It holds a slice, so compare two
// with reflect.DeepEqual. The zero value lists none.
type PreparationRejection struct{ problems []PreparationProblem }

// Problems returns a new slice of the problems.
func (r PreparationRejection) Problems() []PreparationProblem { return slices.Clone(r.problems) }

// First returns the kind of the first problem, the zero kind when there is none.
func (r PreparationRejection) First() PreparationRejectionKind {
	if len(r.problems) == 0 {
		return 0
	}
	return r.problems[0].Kind
}

// Has reports whether any problem has kind k.
func (r PreparationRejection) Has(k PreparationRejectionKind) bool {
	return slices.ContainsFunc(r.problems, func(p PreparationProblem) bool { return p.Kind == k })
}

// ClassSpellcastingSetup is the lists a class starts with or is built from. Spells
// are named by id. Prepared holds the chosen prepared spells only; the
// always-prepared set is separate. Construct it with a keyed literal.
type ClassSpellcastingSetup struct {
	Class          SpellcastingClass
	Cantrips       []content.SpellID
	Prepared       []content.SpellID
	AlwaysPrepared []content.SpellID
	ExtraCantrips  int
	Granted        []GrantedCantrip
	Spellbook      Spellbook
}

// ClassSpellcasting is one class's spells: the cantrips known, the extra-cantrip
// allowance a feature gives, the granted cantrips, the prepared and
// always-prepared spells and, for a Wizard, the spellbook, at a class level. A
// value built by NewClassSpellcasting or held by a SpellcastingState satisfies
// the limits of Tables F and G. It holds slices, so compare two with
// reflect.DeepEqual. The zero value is invalid.
type ClassSpellcasting struct {
	class    SpellcastingClass
	level    int
	cantrips CantripsKnown
	extra    int
	granted  []GrantedCantrip
	prepared PreparedSpells
	book     Spellbook
}

// NewClassSpellcasting returns the class's spells at a class level (1 to 20)
// after checking every list against Tables F and G and the class's spell list in
// spells (SRD 5.2.1, the class "Spellcasting" entries).
//
// It returns an error matching srderr.ErrInvalidInput, before any list check,
// for an invalid class, a level outside 1 to 20, a negative allowance or a spell
// id absent from spells. Otherwise it returns an errors.Join of one
// *PreparationProblem per problem, in the fixed order of PreparationRejectionKind,
// each matching srderr.ErrInvalidInput. It never panics.
func NewClassSpellcasting(setup ClassSpellcastingSetup, level int, spells map[content.SpellID]content.Spell) (ClassSpellcasting, error) {
	if !setup.Class.Valid() || level < 1 || level > MaxLevel || setup.ExtraCantrips < 0 {
		return ClassSpellcasting{}, invalidInput("class spellcasting: invalid class, level or allowance")
	}
	if len(setupMissing(spells, setup)) > 0 {
		return ClassSpellcasting{}, invalidInput("class spellcasting: spell not supplied")
	}
	c := buildClass(setup, level)
	problems := c.problems(spells)
	if len(problems) == 0 {
		return c, nil
	}
	errs := make([]error, len(problems))
	for i := range problems {
		errs[i] = &problems[i]
	}
	return ClassSpellcasting{}, errors.Join(errs...)
}

// MustClassSpellcasting is NewClassSpellcasting for known-valid input. It panics
// with the error NewClassSpellcasting returns, and only then.
func MustClassSpellcasting(setup ClassSpellcastingSetup, level int, spells map[content.SpellID]content.Spell) ClassSpellcasting {
	c, err := NewClassSpellcasting(setup, level, spells)
	if err != nil {
		panic(err)
	}
	return c
}

func buildClass(s ClassSpellcastingSetup, level int) ClassSpellcasting {
	return ClassSpellcasting{
		class: s.Class, level: level, extra: s.ExtraCantrips, book: NewSpellbook(s.Spellbook.ids),
		cantrips: CantripsKnown{ids: slices.Clone(s.Cantrips)}, granted: slices.Clone(s.Granted),
		prepared: PreparedSpells{ids: slices.Clone(s.Prepared), always: slices.Clone(s.AlwaysPrepared)},
	}
}

// IsZero reports whether c is the invalid zero value.
func (c ClassSpellcasting) IsZero() bool { return !c.class.Valid() }

// Class returns the class.
func (c ClassSpellcasting) Class() SpellcastingClass { return c.class }

// Level returns the class level the lists satisfy.
func (c ClassSpellcasting) Level() int { return c.level }

// Cantrips returns the chosen cantrips known.
func (c ClassSpellcasting) Cantrips() CantripsKnown {
	return CantripsKnown{ids: slices.Clone(c.cantrips.ids)}
}

// ExtraCantrips returns the extra-cantrip allowance a feature gives.
func (c ClassSpellcasting) ExtraCantrips() int { return c.extra }

// GrantedCantrips returns a new slice of the cantrips features grant.
func (c ClassSpellcasting) GrantedCantrips() []GrantedCantrip { return slices.Clone(c.granted) }

// Prepared returns the prepared spells.
func (c ClassSpellcasting) Prepared() PreparedSpells {
	return PreparedSpells{ids: slices.Clone(c.prepared.ids), always: slices.Clone(c.prepared.always)}
}

// Spellbook returns the Wizard's spellbook; empty for any other class.
func (c ClassSpellcasting) Spellbook() Spellbook { return NewSpellbook(c.book.ids) }

// SpellsToChoose returns the prepared limit minus the spells prepared, never
// below zero (always-prepared spells excluded).
func (c ClassSpellcasting) SpellsToChoose() int {
	return max(0, PreparedLimit(c.class, c.level)-len(c.prepared.ids))
}

// CantripsToChoose returns the Table F count plus the extra-cantrip allowance
// minus the cantrips known, never below zero (granted cantrips are not counted).
func (c ClassSpellcasting) CantripsToChoose() int {
	return max(0, CantripLimit(c.class, c.level)+c.extra-len(c.cantrips.ids))
}

// problems checks the lists against the limits and the spell lists in spells,
// which must hold every spell named. It lists every problem in the fixed order.
func (c ClassSpellcasting) problems(spells map[content.SpellID]content.Spell) []PreparationProblem {
	var out []PreparationProblem
	add := func(k PreparationRejectionKind, id content.SpellID) {
		out = append(out, PreparationProblem{Kind: k, Spell: id, Class: c.class})
	}
	onList := func(id content.SpellID, list content.ClassID) bool {
		return slices.Contains(spells[id].Classes(), list)
	}
	level := func(id content.SpellID) int { return int(spells[id].Level()) }
	for _, id := range slices.Concat(c.prepared.ids, c.cantrips.ids) {
		if !onList(id, c.class.ClassID()) {
			add(PreparationNotOnClassList, id)
		}
	}
	for _, g := range c.granted {
		if !onList(g.spell, g.list) {
			add(PreparationNotOnClassList, g.spell)
		}
		if level(g.spell) != 0 {
			add(PreparationNotACantrip, g.spell)
		}
	}
	for _, id := range c.prepared.ids {
		if level(id) > MaxPreparableLevel(c.class, c.level) {
			add(PreparationSpellLevelTooHigh, id)
		}
		if level(id) == 0 {
			add(PreparationCantripNotPreparable, id)
		}
		if c.class == CasterWizard && !c.book.Contains(id) {
			add(PreparationNotInSpellbook, id)
		}
	}
	if len(c.prepared.ids) > PreparedLimit(c.class, c.level) {
		add(PreparationTooManyPrepared, "")
	}
	for _, ids := range [][]content.SpellID{c.prepared.ids, c.cantrips.ids} {
		for i, id := range ids {
			if slices.Contains(ids[:i], id) {
				add(PreparationDuplicateSpell, id)
			}
		}
	}
	if len(c.cantrips.ids) > CantripLimit(c.class, c.level)+c.extra {
		add(PreparationTooManyCantrips, "")
	}
	for _, id := range c.cantrips.ids {
		if level(id) != 0 {
			add(PreparationNotACantrip, id)
		}
	}
	slices.SortStableFunc(out, func(a, b PreparationProblem) int { return int(a.Kind) - int(b.Kind) })
	return out
}

// missingSpells returns the MissingContent for each id absent from spells, in the
// order the ids appear across lists, without repeats.
func missingSpells(spells map[content.SpellID]content.Spell, lists ...[]content.SpellID) []content.MissingContent {
	var out []content.MissingContent
	var seen []content.SpellID
	for _, id := range slices.Concat(lists...) {
		if _, ok := spells[id]; !ok && !slices.Contains(seen, id) {
			seen = append(seen, id)
			out = append(out, id.Missing())
		}
	}
	return out
}

// setupMissing is missingSpells over a setup in the order prepared, cantrips,
// spellbook, granted cantrips.
func setupMissing(spells map[content.SpellID]content.Spell, s ClassSpellcastingSetup) []content.MissingContent {
	granted := make([]content.SpellID, len(s.Granted))
	for i, g := range s.Granted {
		granted[i] = g.spell
	}
	return missingSpells(spells, s.Prepared, s.Cantrips, s.Spellbook.ids, granted)
}
