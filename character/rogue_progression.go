package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	// RogueClassID is the id of the Rogue class.
	RogueClassID content.ClassID = "rogue"
	// ThiefSubclassID is the id of the Thief, the only Rogue subclass in SRD 5.2.1 (p. 64).
	ThiefSubclassID content.SubclassID = "thief"

	rogueSneakAttackColumn content.ClassColumnID = "sneak_attack"
)

// The Rogue rule constants (SRD 5.2.1, "Rogue Features", pp. 62-63); a test pins them to the bundled definition.
const (
	rogueSubclassLevel      = 3
	rogueEpicBoonLevel      = 19
	rogueWeaponMasteryCount = 2
	rogueSlipperyMindLevel  = 15
)

// rogueFeatLevels are the Rogue levels of Ability Score Improvement (SRD 5.2.1, "Level 4: Ability Score Improvement", p. 63).
func rogueFeatLevels() []int { return []int{4, 8, 10, 12, 16} }

// RogueEntry says how a character becomes a Rogue (SRD 5.2.1, "Becoming a Rogue ...", p. 61). The zero value is invalid.
type RogueEntry int

// The ways to become a Rogue.
const (
	// RogueFirstClass: Rogue is the character's level 1 class.
	RogueFirstClass RogueEntry = iota + 1
	// RogueMulticlass: the character adds Rogue levels to other classes.
	RogueMulticlass
)

// String returns "first_class", "multiclass" or "unknown".
func (e RogueEntry) String() string {
	switch e {
	case RogueFirstClass:
		return "first_class"
	case RogueMulticlass:
		return "multiclass"
	}
	return "unknown"
}

// Valid reports whether e is one of the two entries.
func (e RogueEntry) Valid() bool { return e == RogueFirstClass || e == RogueMulticlass }

// RogueProgressionReason is why NewRogueProgression refused a definition. The zero value is not a reason.
type RogueProgressionReason int

const (
	// NotTheRogueDefinition: the definition is not the Rogue's.
	NotTheRogueDefinition RogueProgressionReason = iota + 1
	// RogueColumnMissing: the sneak_attack column is missing or not a dice column.
	RogueColumnMissing
	// RogueRowsIncomplete: the Sneak Attack cell of a level is not a number of d6.
	RogueRowsIncomplete
	// RogueSubclassMissing: the definition has no Thief subclass at the subclass level.
	RogueSubclassMissing
	// RogueSpellcastingPresent: the definition declares Spellcasting or Pact Magic, which the Rogue lacks (SRD 5.2.1, p. 62).
	RogueSpellcastingPresent
	// RogueSkillListMissing: the skill choice of the class or of the multiclass grant is not a listed choice.
	RogueSkillListMissing
)

// String returns the reason as snake_case, or "unknown".
func (r RogueProgressionReason) String() string {
	switch r {
	case NotTheRogueDefinition:
		return "not_the_rogue_definition"
	case RogueColumnMissing:
		return "column_missing"
	case RogueRowsIncomplete:
		return "rows_incomplete"
	case RogueSubclassMissing:
		return "subclass_missing"
	case RogueSpellcastingPresent:
		return "spellcasting_present"
	case RogueSkillListMissing:
		return "skill_list_missing"
	}
	return "unknown"
}

// RogueProgressionError reports one reason NewRogueProgression refused a definition. It matches srderr.ErrInvalidInput
// with errors.Is and carries no display text. Build it with a keyed literal.
type RogueProgressionError struct{ Reason RogueProgressionReason }

// Error renders the error. The text is not a contract.
func (e *RogueProgressionError) Error() string {
	return "character: invalid Rogue progression: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*RogueProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *RogueProgressionError of the same Reason.
func (e *RogueProgressionError) Is(target error) bool {
	t, ok := target.(*RogueProgressionError)
	return ok && t.Reason == e.Reason
}

// SneakAttackDice is a number of d6, the Sneak Attack damage or the cost of a Cunning Strike effect (SRD 5.2.1, "Level 1:
// Sneak Attack", p. 61). It is a count, never a rolled total, so a Critical Hit can roll the dice twice. Immutable,
// comparable with ==; the zero value is no dice.
type SneakAttackDice struct{ n int }

// NewSneakAttackDice returns n d6. It returns an error matching srderr.ErrInvalidInput for a negative n.
func NewSneakAttackDice(n int) (SneakAttackDice, error) {
	if n < 0 {
		return SneakAttackDice{}, invalidInput("negative Sneak Attack die count")
	}
	return SneakAttackDice{n: n}, nil
}

// Count returns the number of d6.
func (d SneakAttackDice) Count() int { return d.n }

// Die returns the die, a d6.
func (SneakAttackDice) Die() dice.Die { return dice.MustDie(6) }

// Expression returns the dice as count d6 with modifier 0, ready for the kernel's dice code; the zero expression for no dice.
func (d SneakAttackDice) Expression() dice.Expression {
	if d.n == 0 {
		return dice.Expression{}
	}
	return dice.MustExpression(d.n, d.Die(), 0)
}

// RogueProgression turns a Rogue level into the numbers and features the Rogue Features table gives (SRD 5.2.1, p. 62). The
// Sneak Attack dice, the feature ids of each row, the Thief's feature ids and the skill lists are read from the Rogue
// ClassDefinition when it is built; the remaining rules are constants checked against the definition by a test. It holds no
// port and no clock. It is immutable and not comparable with ==; the zero value is invalid (see IsZero). Safe for
// concurrent use.
type RogueProgression struct {
	sneak      [MaxLevel]int
	features   [MaxLevel][]content.FeatureID
	thief      [MaxLevel][]content.FeatureID
	skills     []content.Skill
	firstPicks int
	multiPicks int
	set        bool
}

// NewRogueProgression reads the Sneak Attack column, the Class Features cells, the Thief's feature ids and the skill lists
// from def. It checks structure only, never a value against the SRD, so house-rule content is accepted, except that the
// Rogue has no Spellcasting or Pact Magic (SRD 5.2.1, "Rogue Features", p. 62).
//
// It returns an error joining a *RogueProgressionError per reason found, matching srderr.ErrInvalidInput. It never panics.
func NewRogueProgression(def content.ClassDefinition) (RogueProgression, error) {
	if def.ID() != RogueClassID {
		return RogueProgression{}, errors.Join(&RogueProgressionError{Reason: NotTheRogueDefinition})
	}
	var reasons []error
	fail := func(r RogueProgressionReason) { reasons = append(reasons, &RogueProgressionError{Reason: r}) }
	var p RogueProgression
	if col, err := def.Column(rogueSneakAttackColumn); err != nil || col.Kind() != content.ColumnKindDice {
		fail(RogueColumnMissing)
	} else {
		p.readRows(def, fail)
	}
	if def.Spellcasting() != nil {
		fail(RogueSpellcastingPresent)
	}
	p.readSubclass(def, fail)
	p.readSkills(def, fail)
	if len(reasons) > 0 {
		return RogueProgression{}, errors.Join(reasons...)
	}
	p.set = true
	return p, nil
}

func (p *RogueProgression) readRows(def content.ClassDefinition, fail func(RogueProgressionReason)) {
	for level := 1; level <= MaxLevel; level++ {
		row, _ := def.RowAt(level) // a definition holds rows 1 to 20, so the level is in range
		v, _ := row.Value(rogueSneakAttackColumn)
		die, isDie := v.(content.ColumnDice)
		if !isDie || die.Sides() != 6 {
			fail(RogueRowsIncomplete)
			return
		}
		p.sneak[level-1] = die.Count()
		if row.Spellcasting() != nil {
			fail(RogueSpellcastingPresent)
		}
		for _, f := range row.Features() {
			if cf, ok := f.(content.RowClassFeature); ok {
				p.features[level-1] = append(p.features[level-1], cf.FeatureID())
			}
		}
	}
}

func (p *RogueProgression) readSubclass(def content.ClassDefinition, fail func(RogueProgressionReason)) {
	found, ok := def.Subclass(ThiefSubclassID).(content.SubclassFound)
	if !ok {
		fail(RogueSubclassMissing)
		return
	}
	for _, f := range found.Subclass().Features() {
		if f.Level() >= 1 && f.Level() <= MaxLevel {
			p.thief[f.Level()-1] = append(p.thief[f.Level()-1], f.ID())
		}
	}
}

func (p *RogueProgression) readSkills(def content.ClassDefinition, fail func(RogueProgressionReason)) {
	first, ok1 := def.SkillChoice().(content.SkillChoiceListed)
	multi, ok2 := def.MulticlassGrants().SkillChoice().(content.SkillChoiceListed)
	if !ok1 || !ok2 {
		fail(RogueSkillListMissing)
		return
	}
	p.skills, p.firstPicks, p.multiPicks = first.Skills(), first.Picks(), multi.Picks()
}

// MustRogueProgression is NewRogueProgression for a definition known to be valid. It panics with the error
// NewRogueProgression returns, and only then.
func MustRogueProgression(def content.ClassDefinition) RogueProgression {
	p, err := NewRogueProgression(def)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p RogueProgression) IsZero() bool { return !p.set }

// SneakAttackDice returns the Sneak Attack dice of the Rogue level (SRD 5.2.1, "Rogue Features", p. 62); none for the zero
// progression or level.
func (p RogueProgression) SneakAttackDice(level RogueLevel) SneakAttackDice {
	if !p.set || level.IsZero() {
		return SneakAttackDice{}
	}
	return SneakAttackDice{n: p.sneak[level.n-1]}
}

// FeaturesAt returns the class feature ids gained at the Rogue level, in printed order, as a new slice; the Subclass feature
// cells are not among them. Empty for the zero progression or level.
func (p RogueProgression) FeaturesAt(level RogueLevel) []content.FeatureID {
	if !p.set || level.IsZero() {
		return []content.FeatureID{}
	}
	return slices.Concat(p.features[level.n-1])
}

// FeaturesUpTo returns the class feature ids of Rogue levels 1 up to and including level, in level order, and none above it
// (SRD 5.2.1, "Level 3: Rogue Subclass", p. 62). The Thief's feature ids of those levels follow each level's class features
// only when subclass is ThiefSubclassID. A feature the table repeats (Expertise, Ability Score Improvement) appears at each level.
func (p RogueProgression) FeaturesUpTo(level RogueLevel, subclass content.SubclassID) []content.FeatureID {
	out := []content.FeatureID{}
	if !p.set || level.IsZero() {
		return out
	}
	for i := 0; i < level.n; i++ {
		out = append(out, p.features[i]...)
		if subclass == ThiefSubclassID {
			out = append(out, p.thief[i]...)
		}
	}
	return out
}

// SavingThrowProficiencies returns the saving throws the Rogue class gives at the Rogue level (SRD 5.2.1, "Core Rogue
// Traits", p. 61; "Level 15: Slippery Mind", p. 63): Dexterity and Intelligence for a first class entry, nothing for a
// multiclass entry, and Wisdom and Charisma added from level 15 for both, because Slippery Mind is a class feature.
func (p RogueProgression) SavingThrowProficiencies(level RogueLevel, entry RogueEntry) []ability.Ability {
	out := []ability.Ability{}
	if entry == RogueFirstClass {
		out = append(out, ability.Dexterity, ability.Intelligence)
	}
	if entry.Valid() && level.n >= rogueSlipperyMindLevel {
		out = append(out, ability.Wisdom, ability.Charisma)
	}
	return out
}

// FeatLevels returns the Rogue levels of Ability Score Improvement, 4, 8, 10, 12 and 16, as a new slice.
func (RogueProgression) FeatLevels() []int { return rogueFeatLevels() }

// GrantsFeat reports whether the Rogue level gives an Ability Score Improvement feat; the level 19 Epic Boon is EpicBoonLevel's.
func (RogueProgression) GrantsFeat(level RogueLevel) bool {
	return slices.Contains(rogueFeatLevels(), level.n)
}

// EpicBoonLevel returns the Rogue level of the Epic Boon, 19 (SRD 5.2.1, "Level 19: Epic Boon", p. 63).
func (RogueProgression) EpicBoonLevel() int { return rogueEpicBoonLevel }

// WeaponMasteryCount returns how many kinds of weapon Weapon Mastery covers, 2 (SRD 5.2.1, "Level 1: Weapon Mastery", p. 62).
func (RogueProgression) WeaponMasteryCount() int { return rogueWeaponMasteryCount }

// SubclassLevel returns the Rogue level of the subclass choice, 3 (SRD 5.2.1, "Level 3: Rogue Subclass", p. 62).
func (RogueProgression) SubclassLevel() int { return rogueSubclassLevel }

// SubclassID returns the id of the Thief, the only Rogue subclass.
func (RogueProgression) SubclassID() content.SubclassID { return ThiefSubclassID }
