package combat

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ErrBlankSpellObjectID is matched, with srderr.ErrInvalidInput, by the error of
// ParseSpellObjectID for a blank value.
var ErrBlankSpellObjectID = fmt.Errorf("combat: blank spell object id: %w", srderr.ErrInvalidInput)

// ErrInvalidSpellEffectsCommand is matched, with srderr.ErrInvalidInput, by the
// error of ResolveSpellEffects.Run for contradictory input that no rule gives
// meaning to (spell-effects.md FR-062).
var ErrInvalidSpellEffectsCommand = fmt.Errorf("combat: invalid spell effects command: %w", srderr.ErrInvalidInput)

// spellCommandError returns an error matching ErrInvalidSpellEffectsCommand, and cause
// when there is one. The detail is for humans; it is not a contract.
func spellCommandError(detail string, cause error) error {
	if cause == nil {
		return fmt.Errorf("combat: %s: %w", detail, ErrInvalidSpellEffectsCommand)
	}
	return fmt.Errorf("combat: %s: %w: %w", detail, ErrInvalidSpellEffectsCommand, cause)
}

// SpellObjectID names an object a spell targets. The host supplies it; the
// package tracks no object state. Build it with ParseSpellObjectID; the zero
// value is blank and invalid. Comparable and safe for concurrent use.
type SpellObjectID string

// ParseSpellObjectID returns s unchanged as a SpellObjectID. It returns an error
// matching ErrBlankSpellObjectID and srderr.ErrInvalidInput when
// strings.TrimSpace(s) is empty. It never panics.
func ParseSpellObjectID(s string) (SpellObjectID, error) {
	if strings.TrimSpace(s) == "" {
		return "", ErrBlankSpellObjectID
	}
	return SpellObjectID(s), nil
}

// MustSpellObjectID is ParseSpellObjectID for values known to be valid. It
// panics with the error ParseSpellObjectID returns, and only then.
func MustSpellObjectID(s string) SpellObjectID {
	id, err := ParseSpellObjectID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// SpellTargetRef is the closed set of things a spell can be aimed at: a creature
// or an object. Variants: CreatureTargetRef, ObjectTargetRef.
type SpellTargetRef interface{ isSpellTargetRef() }

// CreatureTargetRef names a creature of the fight. Construct with a keyed literal.
type CreatureTargetRef struct{ ID CombatantID }

// ObjectTargetRef names an object the host tracks. Construct with a keyed literal.
type ObjectTargetRef struct{ ID SpellObjectID }

func (CreatureTargetRef) isSpellTargetRef() {}
func (ObjectTargetRef) isSpellTargetRef()   {}

// validSpellTargetRef reports whether r is a creature with an id or an object
// with a non-blank id.
func validSpellTargetRef(r SpellTargetRef) bool {
	switch v := r.(type) {
	case CreatureTargetRef:
		return strings.TrimSpace(string(v.ID)) != ""
	case ObjectTargetRef:
		_, err := ParseSpellObjectID(string(v.ID))
		return err == nil
	}
	return false
}

// TargetValidity is whether the host reports that a spell can affect a target
// (SRD 5.2.1, "Invalid Targets", p. 106). The zero value is TargetCanBeAffected.
type TargetValidity int

// The two validities.
const (
	// TargetCanBeAffected: the target is a valid target.
	TargetCanBeAffected TargetValidity = iota
	// TargetCannotBeAffected: the host says the spell cannot affect it.
	TargetCannotBeAffected
)

// Valid reports whether v is one of the two validities.
func (v TargetValidity) Valid() bool { return v == TargetCanBeAffected || v == TargetCannotBeAffected }

// String returns "can_be_affected", "cannot_be_affected" or "unknown".
func (v TargetValidity) String() string {
	return enumName([]string{"can_be_affected", "cannot_be_affected"}, int(v)+1)
}

// PathToTarget is whether a clear path runs from the caster to a chosen target
// (SRD 5.2.1, "A Clear Path to the Target", p. 106). The zero value is PathClear.
type PathToTarget int

// The two paths.
const (
	// PathClear: nothing blocks the path.
	PathClear PathToTarget = iota
	// PathBlockedByTotalCover: Total Cover blocks the path.
	PathBlockedByTotalCover
)

// Valid reports whether p is one of the two paths.
func (p PathToTarget) Valid() bool { return p == PathClear || p == PathBlockedByTotalCover }

// String returns "clear", "blocked_by_total_cover" or "unknown".
func (p PathToTarget) String() string {
	return enumName([]string{"clear", "blocked_by_total_cover"}, int(p)+1)
}

// SpellAttackFactsParams are the fields of a SpellAttackFacts. The zero values of
// the optional fields are the plain case: no cover, no nearby enemy, a seen target.
type SpellAttackFactsParams struct {
	// ArmorClass is the target's Armor Class before cover.
	ArmorClass d20.ArmorClass
	// CoverSources are the cover sources the host knows of (CoverResolver picks).
	CoverSources []CoverSource
	// Distance is the distance to the target in feet; it must not be negative.
	Distance int
	// Range is the spell's range; ranged spell attacks need it.
	Range AttackRange
	// Reach is the attacker's reach; the zero Reach is 5 feet.
	Reach Reach
	// NearbyEnemies are the attacker's enemies near it.
	NearbyEnemies []NearbyEnemy
	// Situation is the visibility and environment of the attack.
	Situation AttackSituation
	// Circumstances are further Advantage and Disadvantage the host found.
	Circumstances []modifier.RollCircumstance
	// AutomaticCritical says whether a hit is a Critical Hit by the target's conditions.
	AutomaticCritical AutomaticCritical
}

// SpellAttackFacts are the target-side facts of a spell attack roll, as the host
// states them (spell-effects.md FR-097). The zero value means the host gave none.
// Build it with NewSpellAttackFacts. Immutable and safe for concurrent use.
type SpellAttackFacts struct {
	p   SpellAttackFactsParams
	set bool
}

// NewSpellAttackFacts returns the facts of p. It returns an error matching
// srderr.ErrInvalidInput for a negative distance. It never panics.
func NewSpellAttackFacts(p SpellAttackFactsParams) (SpellAttackFacts, error) {
	if p.Distance < 0 {
		return SpellAttackFacts{}, invalidInput("spell attack facts with a negative distance")
	}
	p.CoverSources = slices.Clone(p.CoverSources)
	p.NearbyEnemies = slices.Clone(p.NearbyEnemies)
	p.Circumstances = slices.Clone(p.Circumstances)
	return SpellAttackFacts{p, true}, nil
}

// IsZero reports whether f holds no facts.
func (f SpellAttackFacts) IsZero() bool { return !f.set }

// Params returns the facts with new slices.
func (f SpellAttackFacts) Params() SpellAttackFactsParams {
	p := f.p
	p.CoverSources = slices.Clone(p.CoverSources)
	p.NearbyEnemies = slices.Clone(p.NearbyEnemies)
	p.Circumstances = slices.Clone(p.Circumstances)
	return p
}

// SpellTargetParams are the fields of a SpellTarget.
type SpellTargetParams struct {
	// Ref names the target; a creature needs a CombatantID, an object a SpellObjectID.
	Ref SpellTargetRef
	// CreatureType is the type of a creature target; the zero value is unknown.
	CreatureType content.CreatureType
	// Validity is whether the host says the spell can affect the target.
	Validity TargetValidity
	// Path is whether a clear path runs to the target (TargetsChosen spells).
	Path PathToTarget
	// Saves are the target's saving throw attempts by ability.
	Saves map[ability.Ability]d20.SaveAttempt
	// AttackFacts are the facts of a spell attack against the target.
	AttackFacts SpellAttackFacts
}

// SpellTarget is a target the host chose, with the facts the resolver needs
// (spell-effects.md FR-093). Build it with NewSpellTarget; the zero value is
// invalid. Immutable and safe for concurrent use.
type SpellTarget struct {
	ref      SpellTargetRef
	typ      content.CreatureType
	validity TargetValidity
	path     PathToTarget
	saves    map[ability.Ability]d20.SaveAttempt
	attack   SpellAttackFacts
}

// NewSpellTarget returns the target of p. It returns an error matching
// srderr.ErrInvalidInput for a missing or invalid ref, an invalid creature type
// (other than the zero value), validity or path, an invalid ability or a nil
// attempt in Saves, or a CreatureType on an object. The saves are copied. It never
// panics.
func NewSpellTarget(p SpellTargetParams) (SpellTarget, error) {
	_, isObject := p.Ref.(ObjectTargetRef)
	switch {
	case !validSpellTargetRef(p.Ref):
		return SpellTarget{}, invalidInput("spell target without a valid ref")
	case p.CreatureType != 0 && (!p.CreatureType.Valid() || isObject):
		return SpellTarget{}, invalidInput("spell target with an invalid creature type")
	case !p.Validity.Valid() || !p.Path.Valid():
		return SpellTarget{}, invalidInput("spell target with an invalid validity or path")
	}
	for a, s := range p.Saves {
		if !a.Valid() || s == nil {
			return SpellTarget{}, invalidInput("spell target with an invalid saving throw attempt")
		}
	}
	return SpellTarget{p.Ref, p.CreatureType, p.Validity, p.Path, maps.Clone(p.Saves), p.AttackFacts}, nil
}

// Ref returns the target's ref.
func (t SpellTarget) Ref() SpellTargetRef { return t.ref }

// CreatureType returns the creature type and whether it is known.
func (t SpellTarget) CreatureType() (content.CreatureType, bool) { return t.typ, t.typ.Valid() }

// Validity returns whether the host says the spell can affect the target.
func (t SpellTarget) Validity() TargetValidity { return t.validity }

// Path returns whether a clear path runs to the target.
func (t SpellTarget) Path() PathToTarget { return t.path }

// Save returns the saving throw attempt for a and whether there is one.
func (t SpellTarget) Save(a ability.Ability) (d20.SaveAttempt, bool) {
	s, ok := t.saves[a]
	return s, ok
}

// AttackFacts returns the spell attack facts and whether the host gave any.
func (t SpellTarget) AttackFacts() (SpellAttackFacts, bool) { return t.attack, t.attack.set }

// InstanceAssignment lists the target of each instance of a part, in instance
// order (spell-effects.md FR-031). Build it with NewInstanceAssignment; the zero
// value is invalid. Immutable and safe for concurrent use.
type InstanceAssignment struct {
	part    int
	targets []SpellTargetRef
}

// NewInstanceAssignment returns the assignment of targets, one per instance, to
// the part at partIndex. It returns an error matching srderr.ErrInvalidInput for a
// negative index, no target or an invalid ref. The targets are copied. It never
// panics.
func NewInstanceAssignment(partIndex int, targets []SpellTargetRef) (InstanceAssignment, error) {
	if partIndex < 0 || len(targets) == 0 || slices.ContainsFunc(targets, func(r SpellTargetRef) bool { return !validSpellTargetRef(r) }) {
		return InstanceAssignment{}, invalidInput("instance assignment with a bad part index or target")
	}
	return InstanceAssignment{partIndex, slices.Clone(targets)}, nil
}

// PartIndex returns the index of the part.
func (a InstanceAssignment) PartIndex() int { return a.part }

// Targets returns the target of each instance, as a new slice.
func (a InstanceAssignment) Targets() []SpellTargetRef { return slices.Clone(a.targets) }
