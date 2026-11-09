package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// ScrollCheckOption is the ability, and optionally the skill, of the ability
// check that reads a scroll above the reader's level (SRD 5.2.1, "Spell Scroll",
// p. 244). A class route's option has no skill: a bare ability check. Comparable.
// The zero value is invalid.
type ScrollCheckOption struct {
	ability ability.Ability
	skill   content.Skill
}

// NewScrollCheckOption returns the option for a. skill may be the zero Skill for
// a bare ability check. It returns an error matching srderr.ErrInvalidInput for
// an invalid ability, or a nonzero skill that is not one of the 18. It never
// panics.
func NewScrollCheckOption(a ability.Ability, skill content.Skill) (ScrollCheckOption, error) {
	if !a.Valid() || (skill != 0 && !skill.Valid()) {
		return ScrollCheckOption{}, invalidInput("scroll check option: invalid ability or skill")
	}
	return ScrollCheckOption{ability: a, skill: skill}, nil
}

// Ability returns the ability of the check.
func (o ScrollCheckOption) Ability() ability.Ability { return o.ability }

// Skill returns the skill of the check and whether there is one; a bare ability
// check has none, so no Proficiency Bonus is added.
func (o ScrollCheckOption) Skill() (content.Skill, bool) { return o.skill, o.skill != 0 }

// ScrollClassRoute is the route by which a reader's class makes a scroll's spell
// part of their spell list: the class and its spellcasting ability (SRD 5.2.1,
// "Spell Scroll", p. 244). Comparable. The zero value is invalid.
type ScrollClassRoute struct {
	class   content.ClassID
	ability ability.Ability
}

// NewScrollClassRoute returns the route of class with spellcasting ability a. It
// returns an error matching srderr.ErrInvalidInput for an invalid class id or
// ability. It never panics.
func NewScrollClassRoute(class content.ClassID, a ability.Ability) (ScrollClassRoute, error) {
	if !class.Valid() || !a.Valid() {
		return ScrollClassRoute{}, invalidInput("scroll class route: invalid class or ability")
	}
	return ScrollClassRoute{class: class, ability: a}, nil
}

// ClassID returns the class whose spell list the route reads.
func (r ScrollClassRoute) ClassID() content.ClassID { return r.class }

// Ability returns the class's spellcasting ability.
func (r ScrollClassRoute) Ability() ability.Ability { return r.ability }

// AnyScrollRoute lets a reader use any Spell Scroll whatever its spell lists: the
// Rogue's Use Magic Device is NewAnyScrollRoute(Intelligence, 1, SkillArcana)
// (SRD 5.2.1, "Level 13: Use Magic Device", p. 64). It waives the spell-list
// requirement only. Comparable. The zero value is invalid and means "no such
// route".
type AnyScrollRoute struct {
	ability  ability.Ability
	reliable content.SpellLevel
	skill    content.Skill
}

// NewAnyScrollRoute returns the route. Scrolls of spell level reliableUpTo or
// lower are cast reliably; above it the check is a (a, skill) check. skill may
// be the zero Skill. It returns an error matching srderr.ErrInvalidInput for an
// invalid ability, a level outside 0 to 9 or a nonzero skill that is not one of
// the 18. It never panics.
func NewAnyScrollRoute(a ability.Ability, reliableUpTo content.SpellLevel, skill content.Skill) (AnyScrollRoute, error) {
	if !a.Valid() || !content.ValidSpellLevel(int(reliableUpTo)) || (skill != 0 && !skill.Valid()) {
		return AnyScrollRoute{}, invalidInput("any scroll route: invalid ability, level or skill")
	}
	return AnyScrollRoute{ability: a, reliable: reliableUpTo, skill: skill}, nil
}

// Ability returns the ability of the check.
func (r AnyScrollRoute) Ability() ability.Ability { return r.ability }

// ReliableUpToLevel returns the highest spell level cast without a check.
func (r AnyScrollRoute) ReliableUpToLevel() content.SpellLevel { return r.reliable }

// CheckSkill returns the skill of the check, the zero Skill when there is none.
func (r AnyScrollRoute) CheckSkill() content.Skill { return r.skill }

// IsZero reports whether r is the zero route, which means the reader has none.
func (r AnyScrollRoute) IsZero() bool { return r == AnyScrollRoute{} }

// ScrollReaderFacts are the inputs of NewScrollReaderProfile. Construct it with a
// keyed literal. The zero ClassRoutes and AnyRoute mean no route.
type ScrollReaderFacts struct {
	// UnderstandsWrittenLanguage is whether the reader understands a written language.
	UnderstandsWrittenLanguage bool
	// DescriptionRestricts is whether the scroll's description restricts who may read it.
	DescriptionRestricts bool
	// HighestSlotLevel is the highest spell slot level in the reader's maximum slots
	// across the Spellcasting and Pact Magic pools, 0 with none (0 to 9).
	HighestSlotLevel content.SpellLevel
	// GrantedCastLevel is the highest spell level the reader casts without a slot through
	// a class feature that grants a regular cast (Mystic Arcanum), 0 to 9.
	GrantedCastLevel content.SpellLevel
	// ClassRoutes are the reader's class routes, one per class, in the order of preference.
	ClassRoutes []ScrollClassRoute
	// AnyRoute is the reader's AnyScrollRoute, the zero route when there is none.
	AnyRoute AnyScrollRoute
}

// ScrollReaderProfile is what a reader brings to a Spell Scroll: whether they can
// read it, the level they can normally cast and their routes to a spell list
// (SRD 5.2.1, "Using a Scroll", p. 205; "Spell Scroll", p. 244). Build it with
// NewScrollReaderProfile. Immutable; the zero profile has no routes and cannot
// read.
type ScrollReaderProfile struct {
	readsLanguage bool
	restricted    bool
	castable      content.SpellLevel
	routes        []ScrollClassRoute
	anyRoute      AnyScrollRoute
}

// NewScrollReaderProfile returns the profile of facts, copying its routes. It
// returns an error matching srderr.ErrInvalidInput for a HighestSlotLevel or
// GrantedCastLevel outside 0 to 9, two class routes with the same class, or a
// zero ScrollClassRoute. It never panics.
func NewScrollReaderProfile(facts ScrollReaderFacts) (ScrollReaderProfile, error) {
	if !content.ValidSpellLevel(int(facts.HighestSlotLevel)) || !content.ValidSpellLevel(int(facts.GrantedCastLevel)) {
		return ScrollReaderProfile{}, invalidInput("scroll reader profile: slot or granted level outside 0 to 9")
	}
	seen := map[content.ClassID]bool{}
	for _, r := range facts.ClassRoutes {
		if !r.class.Valid() || seen[r.class] {
			return ScrollReaderProfile{}, invalidInput("scroll reader profile: invalid or duplicate class route")
		}
		seen[r.class] = true
	}
	return ScrollReaderProfile{readsLanguage: facts.UnderstandsWrittenLanguage, restricted: facts.DescriptionRestricts,
		castable: max(facts.HighestSlotLevel, facts.GrantedCastLevel), routes: slices.Clone(facts.ClassRoutes), anyRoute: facts.AnyRoute}, nil
}

// MustScrollReaderProfile is NewScrollReaderProfile for facts known to be valid;
// it panics with the error NewScrollReaderProfile returns.
func MustScrollReaderProfile(facts ScrollReaderFacts) ScrollReaderProfile {
	p, err := NewScrollReaderProfile(facts)
	if err != nil {
		panic(err)
	}
	return p
}

// CastableLevel returns the highest spell level the reader casts reliably through a
// class route: the greater of the highest slot level and the granted cast level.
func (p ScrollReaderProfile) CastableLevel() content.SpellLevel { return p.castable }

// ClassRoutes returns a new slice of the class routes.
func (p ScrollReaderProfile) ClassRoutes() []ScrollClassRoute { return slices.Clone(p.routes) }

// AnyRoute returns the AnyScrollRoute and whether the reader has one.
func (p ScrollReaderProfile) AnyRoute() (AnyScrollRoute, bool) {
	return p.anyRoute, !p.anyRoute.IsZero()
}
