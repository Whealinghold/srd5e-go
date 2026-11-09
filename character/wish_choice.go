package character

import (
	"maps"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// The limits of the Wish effects (SRD 5.2.1, "Wish", p. 175).
const (
	wishObjectMaxCopper  = 25_000 * 100 // 25,000 GP in Copper Pieces ("Coins", p. 89)
	wishObjectMaxFeet    = 300.0
	wishHealMaxOthers    = 20
	wishGrantMaxCreature = 10
	wishStatementMax     = 2000 // a package sanity bound, not an SRD number
	wishImmunityHours    = 8
)

// WishSpellID returns the id of the Wish spell, "wish".
func WishSpellID() content.SpellID { return "wish" }

// WishLevel returns the level of the Wish spell, 9.
func WishLevel() content.SpellLevel { return content.MaxSpellLevel }

// WishStatement is a player's statement for the Game Master, opaque to the package: a non-empty
// text of at most 2,000 characters. The zero value is invalid.
type WishStatement struct{ text string }

// NewWishStatement returns the statement s.
//
// It returns an error matching srderr.ErrInvalidInput for an empty or whitespace-only s and for
// one over 2,000 characters. It never panics.
func NewWishStatement(s string) (WishStatement, error) {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > wishStatementMax {
		return WishStatement{}, invalidInput("wish statement empty or over 2,000 characters")
	}
	return WishStatement{text: s}, nil
}

// String returns the statement as given.
func (s WishStatement) String() string { return s.text }

// IsZero reports whether s is the invalid zero statement.
func (s WishStatement) IsZero() bool { return s.text == "" }

// WishCreature identifies a creature a Wish affects, by a host-chosen id. The zero value is invalid.
type WishCreature struct{ id string }

// NewWishCreature returns the creature called id, which must not be empty.
//
// It returns an error matching srderr.ErrInvalidInput for an empty id. It never panics.
func NewWishCreature(id string) (WishCreature, error) {
	if id == "" {
		return WishCreature{}, invalidInput("wish creature without an id")
	}
	return WishCreature{id: id}, nil
}

// String returns the id.
func (c WishCreature) String() string { return c.id }

// validWishCreatures checks a creature list of 1 or more distinct valid creatures.
func validWishCreatures(creatures []WishCreature) error {
	if len(creatures) == 0 {
		return invalidInput("wish effect on no creature")
	}
	seen := make(map[WishCreature]bool, len(creatures))
	for _, c := range creatures {
		if c == (WishCreature{}) || seen[c] {
			return invalidInput("wish creature list with a zero or repeated creature")
		}
		seen[c] = true
	}
	return nil
}

// WishChoice is what a Wish is cast to do, a closed set. Variants: DuplicateSpell,
// ObjectCreation, InstantHealth, ResistanceGrant, SpellImmunityGrant, SuddenLearning, RollRedo,
// ReshapeReality. Build each with its New function; a zero variant is invalid.
//
//sumtype:decl
type WishChoice interface {
	isWishChoice()
	// validate reports a zero or inconsistent variant as an error matching srderr.ErrInvalidInput.
	validate() error
}

// DuplicateSpell duplicates another spell of level 8 or lower.
type DuplicateSpell struct{ spell content.Spell }

// NewDuplicateSpell returns the choice to duplicate spell. Whether its level is allowed is a rule
// check (ResolveWishRejection), not input validation.
//
// It returns an error matching srderr.ErrInvalidInput for a spell without a valid id. It never panics.
func NewDuplicateSpell(spell content.Spell) (DuplicateSpell, error) {
	d := DuplicateSpell{spell: spell}
	return d, d.validate()
}

// Spell returns the spell duplicated.
func (d DuplicateSpell) Spell() content.Spell { return d.spell }

// ObjectCreation creates a nonmagical object of up to 25,000 GP in value and 300 feet in its
// largest dimension.
type ObjectCreation struct {
	value     content.Cost
	feet      float64
	magicItem bool
}

// NewObjectCreation returns the choice to create an object worth value, largest dimension
// largestDimensionFeet, that is a magic item when magicItem. The limits are rule checks.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid or per-rate Cost and for a
// dimension that is zero, negative, NaN or infinite. It never panics.
func NewObjectCreation(value content.Cost, largestDimensionFeet float64, magicItem bool) (ObjectCreation, error) {
	o := ObjectCreation{value: value, feet: largestDimensionFeet, magicItem: magicItem}
	return o, o.validate()
}

// MustObjectCreation is NewObjectCreation for known-good values. It panics with the error
// NewObjectCreation returns, and only then.
func MustObjectCreation(value content.Cost, largestDimensionFeet float64, magicItem bool) ObjectCreation {
	o, err := NewObjectCreation(value, largestDimensionFeet, magicItem)
	if err != nil {
		panic(err)
	}
	return o
}

// Value returns the value of the object.
func (o ObjectCreation) Value() content.Cost { return o.value }

// LargestDimensionFeet returns the largest dimension in feet.
func (o ObjectCreation) LargestDimensionFeet() float64 { return o.feet }

// MagicItem reports whether the caller says the object is a magic item.
func (o ObjectCreation) MagicItem() bool { return o.magicItem }

// InstantHealth restores the caster and up to twenty other creatures to full Hit Points.
type InstantHealth struct{ others int }

// NewInstantHealth returns the choice to heal others creatures besides the caster. More than 20 is
// a rule check.
//
// It returns an error matching srderr.ErrInvalidInput for a negative count. It never panics.
func NewInstantHealth(others int) (InstantHealth, error) {
	h := InstantHealth{others: others}
	return h, h.validate()
}

// OtherCreatures returns how many creatures besides the caster are healed.
func (h InstantHealth) OtherCreatures() int { return h.others }

// ResistanceGrant gives up to ten creatures permanent Resistance to one damage type.
type ResistanceGrant struct {
	damage    content.DamageType
	creatures []WishCreature
}

// NewResistanceGrant returns the choice to grant Resistance to damage for creatures, copied. More
// than 10 is a rule check.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid damage type, no creature or a
// zero or repeated creature. It never panics.
func NewResistanceGrant(damage content.DamageType, creatures ...WishCreature) (ResistanceGrant, error) {
	g := ResistanceGrant{damage: damage, creatures: slices.Clone(creatures)}
	return g, g.validate()
}

// DamageType returns the damage type resisted.
func (g ResistanceGrant) DamageType() content.DamageType { return g.damage }

// Creatures returns a new slice of the creatures.
func (g ResistanceGrant) Creatures() []WishCreature { return slices.Clone(g.creatures) }

// SpellImmunitySubject is what a Spell Immunity grant protects against, a closed set. Variants:
// ImmuneToSpell, ImmuneToOtherEffect.
//
//sumtype:decl
type SpellImmunitySubject interface{ isSpellImmunitySubject() }

// ImmuneToSpell is Immunity to a single spell.
type ImmuneToSpell struct{ Spell content.SpellID }

// ImmuneToOtherEffect is Immunity to another magical effect, described by the player.
type ImmuneToOtherEffect struct{ Effect WishStatement }

func (ImmuneToSpell) isSpellImmunitySubject()       {}
func (ImmuneToOtherEffect) isSpellImmunitySubject() {}

// SpellImmunityGrant gives up to ten creatures Immunity to a single spell or other magical effect
// for 8 hours.
type SpellImmunityGrant struct {
	subject   SpellImmunitySubject
	creatures []WishCreature
}

// NewSpellImmunityGrant returns the choice to protect creatures, copied, from subject. More than 10
// is a rule check.
//
// It returns an error matching srderr.ErrInvalidInput for a nil or zero subject, no creature or a
// zero or repeated creature. It never panics.
func NewSpellImmunityGrant(subject SpellImmunitySubject, creatures ...WishCreature) (SpellImmunityGrant, error) {
	g := SpellImmunityGrant{subject: subject, creatures: slices.Clone(creatures)}
	return g, g.validate()
}

// Subject returns what the grant protects against.
func (g SpellImmunityGrant) Subject() SpellImmunitySubject { return g.subject }

// Creatures returns a new slice of the creatures.
func (g SpellImmunityGrant) Creatures() []WishCreature { return slices.Clone(g.creatures) }

// SuddenLearning replaces a held feat by another for which the caster is eligible.
type SuddenLearning struct {
	old        content.FeatID
	feat       content.Feat
	choices    FeatChoices
	requiredBy []content.FeatID
	held       HeldFeatures
	resolved   map[content.SpellID]content.Spell
	feats      TakenFeats
}

// NewSuddenLearning returns the choice to replace old by feat with choices, given the feats the
// caster holds, the class features held, the Spells named by Magic Initiate choices and requiredBy,
// the feats that name old as a prerequisite. The slice and map are copied.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid old id, a feat without an id,
// nil choices, or a requiredBy entry that is not a held feat. It never panics.
func NewSuddenLearning(old content.FeatID, feat content.Feat, choices FeatChoices, requiredBy []content.FeatID,
	held HeldFeatures, resolved map[content.SpellID]content.Spell, feats TakenFeats) (SuddenLearning, error) {
	s := SuddenLearning{old: old, feat: feat, choices: choices, requiredBy: slices.Clone(requiredBy), held: held, resolved: maps.Clone(resolved), feats: feats}
	return s, s.validate()
}

// OldFeat returns the feat lost.
func (s SuddenLearning) OldFeat() content.FeatID { return s.old }

// NewFeat returns the feat gained.
func (s SuddenLearning) NewFeat() content.Feat { return s.feat }

// FeatsRequiredByOthers returns a new slice of the held feats that name the old feat as a prerequisite.
func (s SuddenLearning) FeatsRequiredByOthers() []content.FeatID { return slices.Clone(s.requiredBy) }

// RollRedo lets the caster reroll one roll made within the last round, with the mode given, and
// choose between the reroll and the original.
type RollRedo struct {
	mode        modifier.RollMode
	withinRound bool
}

// NewRollRedo returns the choice to redo a roll with mode; withinLastRound says the roll was made
// within the last round, including the caster's last turn (a rule check).
//
// It returns an error matching srderr.ErrInvalidInput for an undefined mode. It never panics.
func NewRollRedo(mode modifier.RollMode, withinLastRound bool) (RollRedo, error) {
	r := RollRedo{mode: mode, withinRound: withinLastRound}
	return r, r.validate()
}

// Mode returns the roll mode of the reroll.
func (r RollRedo) Mode() modifier.RollMode { return r.mode }

// WithinLastRound reports whether the roll was made within the last round.
func (r RollRedo) WithinLastRound() bool { return r.withinRound }

// ReshapeReality is a wish stated to the Game Master; the package decides nothing about it.
type ReshapeReality struct{ statement WishStatement }

// NewReshapeReality returns the choice to wish statement.
//
// It returns an error matching srderr.ErrInvalidInput for the zero statement. It never panics.
func NewReshapeReality(statement WishStatement) (ReshapeReality, error) {
	r := ReshapeReality{statement: statement}
	return r, r.validate()
}

// Statement returns the wish.
func (r ReshapeReality) Statement() WishStatement { return r.statement }

func (DuplicateSpell) isWishChoice()     {}
func (ObjectCreation) isWishChoice()     {}
func (InstantHealth) isWishChoice()      {}
func (ResistanceGrant) isWishChoice()    {}
func (SpellImmunityGrant) isWishChoice() {}
func (SuddenLearning) isWishChoice()     {}
func (RollRedo) isWishChoice()           {}
func (ReshapeReality) isWishChoice()     {}

func (d DuplicateSpell) validate() error {
	if !d.spell.ID().Valid() {
		return invalidInput("duplicate a spell without an id")
	}
	return nil
}

func (o ObjectCreation) validate() error {
	if !o.value.Valid() {
		return invalidInput("wish object without a valid value")
	}
	if _, per := o.value.Per(); per {
		return invalidInput("wish object value charged per day or mile")
	}
	if math.IsNaN(o.feet) || math.IsInf(o.feet, 0) || o.feet <= 0 {
		return invalidInput("wish object dimension not a positive finite number")
	}
	return nil
}

func (h InstantHealth) validate() error {
	if h.others < 0 {
		return invalidInput("wish healing of a negative number of creatures")
	}
	return nil
}

func (g ResistanceGrant) validate() error {
	if !g.damage.Valid() {
		return invalidInput("wish resistance to an invalid damage type")
	}
	return validWishCreatures(g.creatures)
}

func (g SpellImmunityGrant) validate() error {
	switch s := g.subject.(type) {
	case ImmuneToSpell:
		if !s.Spell.Valid() {
			return invalidInput("wish immunity to an invalid spell")
		}
	case ImmuneToOtherEffect:
		if s.Effect.IsZero() {
			return invalidInput("wish immunity to an empty effect")
		}
	default:
		return invalidInput("wish immunity without a subject")
	}
	return validWishCreatures(g.creatures)
}

func (s SuddenLearning) validate() error {
	if !s.old.Valid() || !s.feat.ID().Valid() || s.choices == nil {
		return invalidInput("sudden learning without an old feat, a new feat or choices")
	}
	for _, id := range s.requiredBy {
		if !s.feats.Holds(id) {
			return invalidInput("sudden learning with a required-by feat that is not held")
		}
	}
	return nil
}

func (r RollRedo) validate() error {
	if r.mode < modifier.Normal || r.mode > modifier.WithDisadvantage {
		return invalidInput("wish roll redo with an undefined roll mode")
	}
	return nil
}

func (r ReshapeReality) validate() error {
	if r.statement.IsZero() {
		return invalidInput("reshape reality without a statement")
	}
	return nil
}

// UnpreventableDamage is damage of one type that "can't be reduced or prevented in any way" (SRD
// 5.2.1, "Wish", p. 175): a damage resolver applies it without Resistance, Immunity or any flat
// reduction. It is a handoff; the package applies no damage. The zero value is invalid.
type UnpreventableDamage struct {
	damage content.DamageType
	amount int
}

// NewUnpreventableDamage returns damage of type t and amount, 1 or more.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid type or an amount below 1. It
// never panics.
func NewUnpreventableDamage(t content.DamageType, amount int) (UnpreventableDamage, error) {
	if !t.Valid() || amount < 1 {
		return UnpreventableDamage{}, invalidInput("unpreventable damage of an invalid type or below 1")
	}
	return UnpreventableDamage{damage: t, amount: amount}, nil
}

// Type returns the damage type.
func (d UnpreventableDamage) Type() content.DamageType { return d.damage }

// Amount returns the damage, 1 or more.
func (d UnpreventableDamage) Amount() int { return d.amount }

// WishEffect is what a resolved Wish hands to the host or to combat, a closed set of structured
// facts with no display text; the package applies none of them except Sudden Learning's feat
// replacement. Variants: WishDuplicate, WishObject, WishInstantHealth, WishResistance,
// WishSpellImmunity, WishSuddenLearning, WishRollRedo, WishReshapeReality.
//
//sumtype:decl
type WishEffect interface{ isWishEffect() }

// WishDuplicate is the handoff of a duplicated spell: the host resolves it through the cast flow at
// the spell's own level, with no requirement, and begins Concentration when it needs it.
type WishDuplicate struct {
	spell         content.SpellID
	level         content.SpellLevel
	concentration bool
}

// Spell returns the spell duplicated.
func (e WishDuplicate) Spell() content.SpellID { return e.spell }

// Level returns the level of the spell, 0 to 8.
func (e WishDuplicate) Level() content.SpellLevel { return e.level }

// RequiresConcentration reports whether the duplicate needs Concentration.
func (e WishDuplicate) RequiresConcentration() bool { return e.concentration }

// WishObject is the handoff of a created object.
type WishObject struct {
	value content.Cost
	feet  float64
}

// Value returns the value of the object.
func (e WishObject) Value() content.Cost { return e.value }

// LargestDimensionFeet returns the largest dimension in feet.
func (e WishObject) LargestDimensionFeet() float64 { return e.feet }

// WishInstantHealth is the handoff of the healing: the caster and OtherCreatures regain all Hit
// Points, and every effect on the list of Greater Restoration ends.
type WishInstantHealth struct{ others int }

// OtherCreatures returns how many creatures besides the caster are healed, 0 to 20.
func (e WishInstantHealth) OtherCreatures() int { return e.others }

// EndsEffectsOf returns the spell whose list of effects ends, "greater_restoration".
func (WishInstantHealth) EndsEffectsOf() content.SpellID { return "greater_restoration" }

// WishResistance is the handoff of a permanent Resistance.
type WishResistance struct {
	damage    content.DamageType
	creatures []WishCreature
}

// DamageType returns the damage type resisted.
func (e WishResistance) DamageType() content.DamageType { return e.damage }

// Creatures returns a new slice of the creatures.
func (e WishResistance) Creatures() []WishCreature { return slices.Clone(e.creatures) }

// Permanent reports that the Resistance lasts indefinitely: always true.
func (WishResistance) Permanent() bool { return true }

// WishSpellImmunity is the handoff of an Immunity lasting 8 hours.
type WishSpellImmunity struct {
	subject   SpellImmunitySubject
	creatures []WishCreature
}

// Subject returns what the creatures are immune to.
func (e WishSpellImmunity) Subject() SpellImmunitySubject { return e.subject }

// Creatures returns a new slice of the creatures.
func (e WishSpellImmunity) Creatures() []WishCreature { return slices.Clone(e.creatures) }

// DurationHours returns how long the Immunity lasts: 8.
func (WishSpellImmunity) DurationHours() int { return wishImmunityHours }

// WishSuddenLearning is the handoff of a feat replacement the package already applied.
type WishSuddenLearning struct{ old, repl content.FeatID }

// Old returns the feat lost.
func (e WishSuddenLearning) Old() content.FeatID { return e.old }

// New returns the feat gained.
func (e WishSuddenLearning) New() content.FeatID { return e.repl }

// WishRollRedo is the handoff of a roll redo: the host rolls with the mode and the caster chooses
// between the reroll and the original. The package rerolls nothing.
type WishRollRedo struct{ mode modifier.RollMode }

// Mode returns the roll mode of the reroll.
func (e WishRollRedo) Mode() modifier.RollMode { return e.mode }

// WishReshapeReality is the handoff of a wish for the Game Master to rule on.
type WishReshapeReality struct{ statement WishStatement }

// Statement returns the wish.
func (e WishReshapeReality) Statement() WishStatement { return e.statement }

func (WishDuplicate) isWishEffect()      {}
func (WishObject) isWishEffect()         {}
func (WishInstantHealth) isWishEffect()  {}
func (WishResistance) isWishEffect()     {}
func (WishSpellImmunity) isWishEffect()  {}
func (WishSuddenLearning) isWishEffect() {}
func (WishRollRedo) isWishEffect()       {}
func (WishReshapeReality) isWishEffect() {}
