package content

// The small closed sets of the spell effect model (spell-effect-data.md
// section 5.2). Each is numbered from 1 in declaration order, 0 is invalid,
// and the name of member v is names[v-1], as in enum.go. Each is comparable
// and safe for concurrent use.

// EffectCoverage says how much of a spell's effect text its model covers
// (spell-effect-data.md FR-004). The zero EffectCoverage is invalid.
type EffectCoverage int

// The effect coverages.
const (
	// CoverageComplete states that every outcome the text describes is modelled.
	CoverageComplete EffectCoverage = iota + 1
	// CoveragePartial states that some described outcome stays in the text.
	CoveragePartial
)

func effectCoverageNames() []string { return []string{"complete", "partial"} }

// Valid reports whether c is one of the two coverages.
func (c EffectCoverage) Valid() bool { return enumValid(effectCoverageNames(), int(c)) }

// String returns the name of c, or "unknown" for an undefined value.
func (c EffectCoverage) String() string { return enumString(effectCoverageNames(), int(c)) }

// TargetSubject is what a spell may target: a creature, an object, or either
// (SRD 5.2.1, "Targets", p. 106). The zero TargetSubject is invalid.
type TargetSubject int

// The target subjects.
const (
	// SubjectCreature is a creature.
	SubjectCreature TargetSubject = iota + 1
	// SubjectObject is an object.
	SubjectObject
	// SubjectCreatureOrObject is a creature or an object.
	SubjectCreatureOrObject
)

func targetSubjectNames() []string { return []string{"creature", "object", "creatureOrObject"} }

// Valid reports whether s is one of the three subjects.
func (s TargetSubject) Valid() bool { return enumValid(targetSubjectNames(), int(s)) }

// String returns the name of s, or "unknown" for an undefined value.
func (s TargetSubject) String() string { return enumString(targetSubjectNames(), int(s)) }

// SelfTargetPolicy says whether a chosen target may be the caster (SRD 5.2.1,
// "Targeting Yourself", p. 106). The zero SelfTargetPolicy is invalid.
type SelfTargetPolicy int

// The self-target policies.
const (
	// MayIncludeCaster is the default: the caster may be a target.
	MayIncludeCaster SelfTargetPolicy = iota + 1
	// ExcludesCaster states that the target must be Hostile or specifically a
	// creature other than the caster.
	ExcludesCaster
)

func selfTargetPolicyNames() []string { return []string{"mayIncludeCaster", "excludesCaster"} }

// Valid reports whether p is one of the two policies.
func (p SelfTargetPolicy) Valid() bool { return enumValid(selfTargetPolicyNames(), int(p)) }

// String returns the name of p, or "unknown" for an undefined value.
func (p SelfTargetPolicy) String() string { return enumString(selfTargetPolicyNames(), int(p)) }

// SpellAttackKind is the kind of a spell attack roll (SRD 5.2.1, "Spell
// Attack", p. 188). The zero SpellAttackKind is invalid.
type SpellAttackKind int

// The spell attack kinds.
const (
	// SpellAttackMelee is a melee spell attack.
	SpellAttackMelee SpellAttackKind = iota + 1
	// SpellAttackRanged is a ranged spell attack.
	SpellAttackRanged
)

func spellAttackKindNames() []string { return []string{"melee", "ranged"} }

// Valid reports whether k is one of the two kinds.
func (k SpellAttackKind) Valid() bool { return enumValid(spellAttackKindNames(), int(k)) }

// String returns the name of k, or "unknown" for an undefined value.
func (k SpellAttackKind) String() string { return enumString(spellAttackKindNames(), int(k)) }

// SuccessEffect is what a successful saving throw does to a spell's effect
// (SRD 5.2.1, "Half Damage", p. 16). The zero SuccessEffect is invalid.
type SuccessEffect int

// The success effects.
const (
	// NothingOnSuccess states that a success avoids the effect.
	NothingOnSuccess SuccessEffect = iota + 1
	// HalfDamageOnSuccess states that a success halves the damage.
	HalfDamageOnSuccess
)

func successEffectNames() []string { return []string{"nothingOnSuccess", "halfDamageOnSuccess"} }

// Valid reports whether e is one of the two success effects.
func (e SuccessEffect) Valid() bool { return enumValid(successEffectNames(), int(e)) }

// String returns the name of e, or "unknown" for an undefined value.
func (e SuccessEffect) String() string { return enumString(successEffectNames(), int(e)) }

// ModifierAddition says whether an amount adds the spellcasting ability
// modifier. The zero ModifierAddition is invalid.
type ModifierAddition int

// The modifier additions.
const (
	// AddsNothing adds no modifier.
	AddsNothing ModifierAddition = iota + 1
	// AddsSpellcastingAbilityModifier adds the caster's spellcasting ability
	// modifier (SRD 5.2.1, "Cure Wounds", p. 121).
	AddsSpellcastingAbilityModifier
)

func modifierAdditionNames() []string { return []string{"none", "spellcastingAbilityModifier"} }

// Valid reports whether a is one of the two additions.
func (a ModifierAddition) Valid() bool { return enumValid(modifierAdditionNames(), int(a)) }

// String returns the name of a, or "unknown" for an undefined value.
func (a ModifierAddition) String() string { return enumString(modifierAdditionNames(), int(a)) }

// TurnOwner is whose turn a condition lasts to (SRD 5.2.1, "Ray of Sickness",
// p. 158). It is not the TurnHolder of another spec. The zero TurnOwner is
// invalid.
type TurnOwner int

// The turn owners.
const (
	// OwnerCaster is the caster's turn.
	OwnerCaster TurnOwner = iota + 1
	// OwnerTarget is the target's turn.
	OwnerTarget
)

func turnOwnerNames() []string { return []string{"caster", "target"} }

// Valid reports whether o is one of the two owners.
func (o TurnOwner) Valid() bool { return enumValid(turnOwnerNames(), int(o)) }

// String returns the name of o, or "unknown" for an undefined value.
func (o TurnOwner) String() string { return enumString(turnOwnerNames(), int(o)) }

// TurnBoundary is the start or the end of a turn. The zero TurnBoundary is
// invalid.
type TurnBoundary int

// The turn boundaries.
const (
	// BoundaryStart is the start of a turn.
	BoundaryStart TurnBoundary = iota + 1
	// BoundaryEnd is the end of a turn.
	BoundaryEnd
)

func turnBoundaryNames() []string { return []string{"start", "end"} }

// Valid reports whether b is one of the two boundaries.
func (b TurnBoundary) Valid() bool { return enumValid(turnBoundaryNames(), int(b)) }

// String returns the name of b, or "unknown" for an undefined value.
func (b TurnBoundary) String() string { return enumString(turnBoundaryNames(), int(b)) }
