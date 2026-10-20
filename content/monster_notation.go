package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/dice"
)

// StatBlockAttackKind is the kind of an attack roll a stat block prints
// (SRD 5.2.1, "Attack Notation", p. 256). The wire spelling is "melee",
// "ranged" or "meleeOrRanged". The zero value is invalid. Comparable and safe
// for concurrent use.
type StatBlockAttackKind int

// The attack kinds.
const (
	AttackMelee StatBlockAttackKind = iota + 1
	AttackRanged
	AttackMeleeOrRanged
)

func statBlockAttackKindNames() []string { return []string{"melee", "ranged", "meleeOrRanged"} }

// Valid reports whether k is one of the three kinds.
func (k StatBlockAttackKind) Valid() bool { return enumValid(statBlockAttackKindNames(), int(k)) }

// String returns the wire spelling of k, or "unknown" for an undefined value.
func (k StatBlockAttackKind) String() string { return enumString(statBlockAttackKindNames(), int(k)) }

// ParseStatBlockAttackKind returns the kind spelled exactly s. Matching is
// case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseStatBlockAttackKind(s string) (StatBlockAttackKind, error) {
	v, err := parseEnum("StatBlockAttackKind", statBlockAttackKindNames(), s)
	return StatBlockAttackKind(v), err
}

// StatBlockAttackRange is the range of a ranged attack: the normal range and
// an optional long range, in feet ("range 80/320 ft."). The zero value means
// no range. Comparable and safe for concurrent use.
type StatBlockAttackRange struct{ normal, long int }

// NewStatBlockAttackRange returns the range normal and long, where long 0
// means no long range.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for a normal range below [MonsterMinRange] or a number above
// [MonsterMaxNumber], BlockConstraint for a long range that is not above the
// normal. It never panics.
func NewStatBlockAttackRange(normal, long int) (StatBlockAttackRange, error) {
	switch {
	case normal < MonsterMinRange || normal > MonsterMaxNumber || long < 0 || long > MonsterMaxNumber:
		return StatBlockAttackRange{}, blockError("StatBlockAttackRange", BlockOutOfRange)
	case long != 0 && long <= normal:
		return StatBlockAttackRange{}, blockError("StatBlockAttackRange", BlockConstraint)
	}
	return StatBlockAttackRange{normal: normal, long: long}, nil
}

// Normal returns the normal range in feet.
func (r StatBlockAttackRange) Normal() int { return r.normal }

// Long returns the long range in feet, and false when there is none.
func (r StatBlockAttackRange) Long() (int, bool) { return r.long, r.long != 0 }

// IsZero reports whether r is the zero value, meaning no range.
func (r StatBlockAttackRange) IsZero() bool { return r == StatBlockAttackRange{} }

// StatBlockAttackFields holds the members of a [StatBlockAttack], for
// [NewStatBlockAttack]. Construct it with a keyed literal.
type StatBlockAttackFields struct {
	// Kind is the kind of attack roll (required).
	Kind StatBlockAttackKind
	// Bonus is the printed signed attack bonus.
	Bonus int
	// Qualifier is the parenthetical after the bonus; "" means none.
	Qualifier string
	// Reach is the reach in feet; it counts only when HasReach is set.
	Reach int
	// HasReach says the attack prints a reach.
	HasReach bool
	// Range is the range of the attack; the zero value means none.
	Range StatBlockAttackRange
}

// StatBlockAttack is the Attack Roll notation of a part (SRD 5.2.1, "Attack
// Notation", p. 256): the kind, the printed bonus, an optional qualifier, the
// reach when printed and the range when printed. The zero value is invalid
// and means a part has no attack. Comparable and safe for concurrent use.
type StatBlockAttack struct {
	f StatBlockAttackFields
}

// NewStatBlockAttack returns the attack of f.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid kind, BlockOutOfRange for a bonus above
// [MonsterMaxNumber] in magnitude or a reach below 0 or above it, then
// BlockBlankText or BlockInvalidCharacter for a non-empty qualifier that
// breaks the text rule, and BlockInvalidMember for a range that is not the
// zero value or a valid range. It never panics.
func NewStatBlockAttack(f StatBlockAttackFields) (StatBlockAttack, error) {
	const block = "StatBlockAttack"
	switch {
	case !f.Kind.Valid():
		return StatBlockAttack{}, blockError(block, BlockInvalidMember)
	case !monsterNumberOK(f.Bonus) || f.Reach < 0 || f.Reach > MonsterMaxNumber:
		return StatBlockAttack{}, blockError(block, BlockOutOfRange)
	}
	if err := optionalText(block, f.Qualifier); err != nil {
		return StatBlockAttack{}, err
	}
	if !f.Range.IsZero() {
		if _, err := NewStatBlockAttackRange(f.Range.normal, f.Range.long); err != nil {
			return StatBlockAttack{}, blockError(block, BlockInvalidMember)
		}
	}
	if !f.HasReach {
		f.Reach = 0
	}
	return StatBlockAttack{f: f}, nil
}

// Kind returns the kind of attack roll.
func (a StatBlockAttack) Kind() StatBlockAttackKind { return a.f.Kind }

// Bonus returns the printed signed attack bonus.
func (a StatBlockAttack) Bonus() int { return a.f.Bonus }

// Qualifier returns the parenthetical after the bonus, or "" when there is
// none.
func (a StatBlockAttack) Qualifier() string { return a.f.Qualifier }

// Reach returns the reach in feet, and false when the attack prints none.
func (a StatBlockAttack) Reach() (int, bool) { return a.f.Reach, a.f.HasReach }

// Range returns the range, and false when the attack prints none.
func (a StatBlockAttack) Range() (StatBlockAttackRange, bool) { return a.f.Range, !a.f.Range.IsZero() }

// IsZero reports whether a is the zero value, meaning no attack.
func (a StatBlockAttack) IsZero() bool { return a == StatBlockAttack{} }

// StatBlockSavingThrow is the Saving Throw notation of a part (SRD 5.2.1,
// "Saving Throw Effect Notation", p. 256): the ability, the printed difficulty
// class, an optional target description and optional printed text before the
// notation ("Response—"). The zero value is invalid and means a part has no
// saving throw. Comparable and safe for concurrent use.
type StatBlockSavingThrow struct {
	ability ability.Ability
	dc      int
	target  string
	prefix  string
}

// NewStatBlockSavingThrow returns the saving throw of ab and dc, with target
// and prefix kept verbatim ("" for none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid ability, BlockOutOfRange for a DC below
// [MonsterMinSaveDC] or above [MonsterMaxNumber], then BlockBlankText or
// BlockInvalidCharacter for a non-empty target or prefix that breaks the text
// rule. It never panics.
func NewStatBlockSavingThrow(ab ability.Ability, dc int, target, prefix string) (StatBlockSavingThrow, error) {
	const block = "StatBlockSavingThrow"
	switch {
	case !ab.Valid():
		return StatBlockSavingThrow{}, blockError(block, BlockInvalidMember)
	case dc < MonsterMinSaveDC || dc > MonsterMaxNumber:
		return StatBlockSavingThrow{}, blockError(block, BlockOutOfRange)
	}
	for _, s := range []string{target, prefix} {
		if err := optionalText(block, s); err != nil {
			return StatBlockSavingThrow{}, err
		}
	}
	return StatBlockSavingThrow{ability: ab, dc: dc, target: target, prefix: prefix}, nil
}

// Ability returns the ability of the save.
func (s StatBlockSavingThrow) Ability() ability.Ability { return s.ability }

// DC returns the printed difficulty class.
func (s StatBlockSavingThrow) DC() int { return s.dc }

// Target returns the target description, or "" when it is implied.
func (s StatBlockSavingThrow) Target() string { return s.target }

// Prefix returns the printed text before the notation, or "" when there is
// none.
func (s StatBlockSavingThrow) Prefix() string { return s.prefix }

// IsZero reports whether s is the zero value, meaning no saving throw.
func (s StatBlockSavingThrow) IsZero() bool { return s == StatBlockSavingThrow{} }

// EffectClauseLabel is the printed label of an [EffectClause] (SRD 5.2.1,
// "Attack Notation" and "Saving Throw Effect Notation", p. 256). The wire
// spelling is the lower camel case name: "hit", "miss", "hitOrMiss",
// "failure", "success", "failureOrSuccess", "firstFailure", "secondFailure",
// "subsequentFailures" and "failureByFiveOrMore". The zero value is invalid.
// Comparable and safe for concurrent use.
type EffectClauseLabel int

// The effect clause labels.
const (
	ClauseHit EffectClauseLabel = iota + 1
	ClauseMiss
	ClauseHitOrMiss
	ClauseFailure
	ClauseSuccess
	ClauseFailureOrSuccess
	ClauseFirstFailure
	ClauseSecondFailure
	ClauseSubsequentFailures
	ClauseFailureByFiveOrMore
)

func effectClauseLabelNames() []string {
	return []string{
		"hit", "miss", "hitOrMiss", "failure", "success", "failureOrSuccess",
		"firstFailure", "secondFailure", "subsequentFailures", "failureByFiveOrMore",
	}
}

// Valid reports whether l is one of the ten labels.
func (l EffectClauseLabel) Valid() bool { return enumValid(effectClauseLabelNames(), int(l)) }

// String returns the wire spelling of l, or "unknown" for an undefined value.
func (l EffectClauseLabel) String() string { return enumString(effectClauseLabelNames(), int(l)) }

// ParseEffectClauseLabel returns the label spelled exactly s. Matching is
// case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseEffectClauseLabel(s string) (EffectClauseLabel, error) {
	v, err := parseEnum("EffectClauseLabel", effectClauseLabelNames(), s)
	return EffectClauseLabel(v), err
}

// SaveSuccessRider is the structured reading of a Success clause (SRD 5.2.1,
// "Saving Throw Effect Notation", p. 256; "Half Damage", p. 16), a closed
// family with three variants: [HalfDamageOnly], [HalfDamage] and
// [ImmuneToPartForHours]. The clause text stays verbatim beside it and the
// rider carries no behaviour here. Consumers switch over it exhaustively.
// Every variant is immutable, comparable and safe for concurrent use.
type SaveSuccessRider interface{ isSaveSuccessRider() }

// HalfDamageOnly is the printed phrase "Half damage only." The zero value is
// valid.
type HalfDamageOnly struct{}

func (HalfDamageOnly) isSaveSuccessRider() {}

// HalfDamage is a success text that begins "Half damage" and goes on, the
// plain half damage of a save. The zero value is valid.
type HalfDamage struct{}

func (HalfDamage) isSaveSuccessRider() {}

// ImmuneToPartForHours is the success text "The target is immune to this
// harpy's Luring Song for 24 hours": the target is immune to the part that
// holds the clause for a number of hours. The zero value is invalid; build it
// with [NewImmuneToPartForHours].
type ImmuneToPartForHours struct{ hours int }

// NewImmuneToPartForHours returns the rider for hours.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange when hours is below 1 or above [MonsterMaxNumber]. It never
// panics.
func NewImmuneToPartForHours(hours int) (ImmuneToPartForHours, error) {
	if hours < 1 || hours > MonsterMaxNumber {
		return ImmuneToPartForHours{}, blockError("ImmuneToPartForHours", BlockOutOfRange)
	}
	return ImmuneToPartForHours{hours: hours}, nil
}

// Hours returns the number of hours.
func (i ImmuneToPartForHours) Hours() int { return i.hours }

func (ImmuneToPartForHours) isSaveSuccessRider() {}

// validSaveSuccessRider reports whether r is a variant its constructor would
// accept.
func validSaveSuccessRider(r SaveSuccessRider) bool {
	switch v := r.(type) {
	case HalfDamageOnly, HalfDamage:
		return true
	case ImmuneToPartForHours:
		_, err := NewImmuneToPartForHours(v.hours)
		return err == nil
	}
	return false
}

// DamageTypeRef is the damage type of a [StatBlockDamage], a closed family
// with two variants: [SpecificDamageType] and [DescribedDamageType].
// Consumers switch over it exhaustively. Every variant is immutable,
// comparable and safe for concurrent use.
type DamageTypeRef interface{ isDamageTypeRef() }

// SpecificDamageType is a damage type of the vocabulary. The zero value is
// invalid; build it with [NewSpecificDamageType].
type SpecificDamageType struct{ t DamageType }

// NewSpecificDamageType returns the reference to t.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember when t is not a damage type. It never panics.
func NewSpecificDamageType(t DamageType) (SpecificDamageType, error) {
	if !t.Valid() {
		return SpecificDamageType{}, blockError("SpecificDamageType", BlockInvalidMember)
	}
	return SpecificDamageType{t: t}, nil
}

// DamageType returns the damage type.
func (s SpecificDamageType) DamageType() DamageType { return s.t }

func (SpecificDamageType) isDamageTypeRef() {}

// DescribedDamageType is a damage type given as text, such as "damage of the
// type chosen for the Draconic Origin trait". The zero value is invalid; build
// it with [NewDescribedDamageType].
type DescribedDamageType struct{ text string }

// NewDescribedDamageType returns the reference that holds text verbatim.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter when text breaks the text rule. It
// never panics.
func NewDescribedDamageType(text string) (DescribedDamageType, error) {
	if err := checkText("DescribedDamageType", text); err != nil {
		return DescribedDamageType{}, err
	}
	return DescribedDamageType{text: text}, nil
}

// Text returns the verbatim text.
func (d DescribedDamageType) Text() string { return d.text }

func (DescribedDamageType) isDamageTypeRef() {}

// validDamageTypeRef reports whether r is a variant its constructor would
// accept.
func validDamageTypeRef(r DamageTypeRef) bool {
	var err error
	switch v := r.(type) {
	case SpecificDamageType:
		_, err = NewSpecificDamageType(v.t)
	case DescribedDamageType:
		_, err = NewDescribedDamageType(v.text)
	default:
		return false
	}
	return err == nil
}

// StatBlockDamage is one damage instance of a clause (SRD 5.2.1, "Damage
// Notation", p. 257): the printed fixed number, the dice when printed, the
// damage type and whether being underwater withholds Resistance to this Fire
// damage. The number is as printed, never recomputed from the dice. The zero
// value is invalid. Comparable and safe for concurrent use.
type StatBlockDamage struct {
	number     int
	dice       dice.Expression
	typ        DamageTypeRef
	noUnderRes bool
}

// NewStatBlockDamage returns the damage of the printed number, the dice
// (the zero Expression for a fixed number with no dice), the type ref and the
// flag underwaterResistanceDenied, which is valid only on a
// [SpecificDamageType] of Fire.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for a number outside 0 to [MonsterMaxNumber] or dice whose
// die is not one of [MonsterHitPointDieFaces], BlockMissing for a nil ref,
// BlockInvalidMember for an invalid ref, BlockConstraint for the flag on any
// other type. It never panics.
func NewStatBlockDamage(number int, e dice.Expression, ref DamageTypeRef, underwaterResistanceDenied bool) (StatBlockDamage, error) {
	const block = "StatBlockDamage"
	switch {
	case number < 0 || number > MonsterMaxNumber:
		return StatBlockDamage{}, blockError(block, BlockOutOfRange)
	case !e.IsZero() && (e.Count() < MonsterMinHitDiceCount || !monsterDieFacesOK(e.Die().Sides())):
		return StatBlockDamage{}, blockError(block, BlockOutOfRange)
	case ref == nil:
		return StatBlockDamage{}, blockError(block, BlockMissing)
	case !validDamageTypeRef(ref):
		return StatBlockDamage{}, blockError(block, BlockInvalidMember)
	}
	if s, ok := ref.(SpecificDamageType); underwaterResistanceDenied && (!ok || s.t != DamageFire) {
		return StatBlockDamage{}, blockError(block, BlockConstraint)
	}
	return StatBlockDamage{number: number, dice: e, typ: ref, noUnderRes: underwaterResistanceDenied}, nil
}

// Number returns the printed fixed number.
func (d StatBlockDamage) Number() int { return d.number }

// Dice returns the printed dice, and false when the damage is a fixed number.
func (d StatBlockDamage) Dice() (dice.Expression, bool) { return d.dice, !d.dice.IsZero() }

// Type returns the damage type.
func (d StatBlockDamage) Type() DamageTypeRef { return d.typ }

// UnderwaterResistanceDenied reports whether the stat block says being
// underwater does not grant Resistance to this Fire damage. Nothing here
// applies it.
func (d StatBlockDamage) UnderwaterResistanceDenied() bool { return d.noUnderRes }

// IsZero reports whether d is the invalid zero value.
func (d StatBlockDamage) IsZero() bool { return d == StatBlockDamage{} }

// EffectClause is one labelled clause of a part: the printed label, the
// verbatim text, the Success rider when there is one and the damage instances
// in printed order. The zero value is invalid. It holds a slice, so compare it
// with [EffectClause.Equal].
type EffectClause struct {
	label  EffectClauseLabel
	text   string
	rider  SaveSuccessRider
	damage []StatBlockDamage
}

// NewEffectClause returns the clause with label, text kept verbatim, rider
// (nil for none) and a copy of damage.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid label, then BlockBlankText or
// BlockInvalidCharacter for text that breaks the text rule, BlockConstraint
// for a rider on a label other than [ClauseSuccess], BlockInvalidMember for
// an invalid rider or a zero damage instance. It never panics.
func NewEffectClause(label EffectClauseLabel, text string, rider SaveSuccessRider, damage []StatBlockDamage) (EffectClause, error) {
	const block = "EffectClause"
	if !label.Valid() {
		return EffectClause{}, blockError(block, BlockInvalidMember)
	}
	if err := checkText(block, text); err != nil {
		return EffectClause{}, err
	}
	switch {
	case rider != nil && label != ClauseSuccess:
		return EffectClause{}, blockError(block, BlockConstraint)
	case rider != nil && !validSaveSuccessRider(rider):
		return EffectClause{}, blockError(block, BlockInvalidMember)
	case slices.ContainsFunc(damage, StatBlockDamage.IsZero):
		return EffectClause{}, blockError(block, BlockInvalidMember)
	}
	return EffectClause{label: label, text: text, rider: rider, damage: slices.Clone(damage)}, nil
}

// Label returns the printed label.
func (c EffectClause) Label() EffectClauseLabel { return c.label }

// Text returns the verbatim text.
func (c EffectClause) Text() string { return c.text }

// SuccessRider returns the Success rider, and false when there is none.
func (c EffectClause) SuccessRider() (SaveSuccessRider, bool) { return c.rider, c.rider != nil }

// Damage returns the damage instances in printed order, as a new slice.
func (c EffectClause) Damage() []StatBlockDamage { return slices.Clone(c.damage) }

// Equal reports whether c and o are the same clause.
func (c EffectClause) Equal(o EffectClause) bool {
	return c.label == o.label && c.text == o.text && c.rider == o.rider && slices.Equal(c.damage, o.damage)
}
