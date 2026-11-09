package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// featNameOf returns names[v-1] for v from 1, or "unknown".
func featNameOf(v int, names ...string) string {
	if v >= 1 && v <= len(names) {
		return names[v-1]
	}
	return "unknown"
}

// FeatChoiceName names the part of a feat's choices a problem concerns (SRD
// 5.2.1, "Parts of a Feat", pp. 87-88). The zero value is not a name.
// Comparable and safe for concurrent use.
type FeatChoiceName int

// The names, in the order the spec lists them.
const (
	FeatChoiceIncreasedAbility FeatChoiceName = iota + 1
	FeatChoiceSpellList
	FeatChoiceSpellcastingAbility
	FeatChoiceCantrips
	FeatChoiceLevelOneSpell
	FeatChoiceSkilledPicks
	FeatChoiceReplacementSpell
	FeatChoiceFeatChoicesShape
)

// String returns "increasedAbility", "spellList", "spellcastingAbility",
// "cantrips", "levelOneSpell", "skilledPicks", "replacementSpell",
// "featChoicesShape" or "unknown".
func (n FeatChoiceName) String() string {
	return featNameOf(int(n), "increasedAbility", "spellList", "spellcastingAbility", "cantrips", "levelOneSpell", "skilledPicks", "replacementSpell", "featChoicesShape")
}

// FeatChoiceProblemKind says what is wrong with a part of a feat's choices. The
// zero value is not a kind. Comparable and safe for concurrent use.
type FeatChoiceProblemKind int

// The kinds, in the order the spec lists them.
const (
	FeatProblemMissing FeatChoiceProblemKind = iota + 1
	FeatProblemUnexpected
	FeatProblemAbilityNotAllowed
	FeatProblemListNotAllowed
	FeatProblemWrongCount
	FeatProblemDuplicate
	FeatProblemNotACantrip
	FeatProblemNotLevelOne
	FeatProblemNotOnSpellList
	FeatProblemLevelMismatch
	FeatProblemListAlreadyUsed
)

// String returns "missing", "unexpected", "abilityNotAllowed", "listNotAllowed",
// "wrongCount", "duplicate", "notACantrip", "notLevelOne", "notOnSpellList",
// "levelMismatch", "listAlreadyUsed" or "unknown".
func (k FeatChoiceProblemKind) String() string {
	return featNameOf(int(k), "missing", "unexpected", "abilityNotAllowed", "listNotAllowed", "wrongCount", "duplicate", "notACantrip", "notLevelOne", "notOnSpellList", "levelMismatch", "listAlreadyUsed")
}

// FeatChoiceProblem is one thing wrong with a feat's choices. It carries no
// display text and is comparable. Construct it with a keyed literal.
type FeatChoiceProblem struct {
	Name FeatChoiceName
	Kind FeatChoiceProblemKind
}

// FeatChoicesError is the error NewAbilityScoreImprovementChoices and
// NewSkilledChoices return: a non-empty list of problems. It matches
// srderr.ErrInvalidInput with errors.Is. Safe for concurrent use.
type FeatChoicesError struct{ problems []FeatChoiceProblem }

// Problems returns a new slice of the problems found, in the order found.
func (e *FeatChoicesError) Problems() []FeatChoiceProblem { return slices.Clone(e.problems) }

// Error renders the error. The text is not a contract.
func (e *FeatChoicesError) Error() string { return "character: invalid feat choices" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *FeatChoicesError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *FeatChoicesError with the same problems.
func (e *FeatChoicesError) Is(target error) bool {
	t, ok := target.(*FeatChoicesError)
	return ok && t != nil && e != nil && slices.Equal(t.problems, e.problems)
}

func choicesError(p ...FeatChoiceProblem) error { return &FeatChoicesError{problems: p} }

// FeatChoices is what a character chose when taking a feat, a closed set (SRD
// 5.2.1, pp. 87-88). Variants: NoFeatChoices, SingleAbilityIncrease,
// AbilityScoreImprovementChoices, MagicInitiateChoices, SkilledChoices.
type FeatChoices interface{ isFeatChoices() }

// NoFeatChoices is the choices of a feat that takes none. The zero value is
// ready to use.
type NoFeatChoices struct{}

// SingleAbilityIncrease is the choice of one ability a Grappler or Epic Boon
// feat raises by 1. It is comparable. The zero value is invalid.
type SingleAbilityIncrease struct{ ability ability.Ability }

// NewSingleAbilityIncrease returns the choice of a. It returns a
// *FeatChoicesError (increasedAbility, abilityNotAllowed) for an invalid
// ability. It never panics.
func NewSingleAbilityIncrease(a ability.Ability) (SingleAbilityIncrease, error) {
	if !a.Valid() {
		return SingleAbilityIncrease{}, choicesError(FeatChoiceProblem{FeatChoiceIncreasedAbility, FeatProblemAbilityNotAllowed})
	}
	return SingleAbilityIncrease{ability: a}, nil
}

// Ability returns the chosen ability.
func (c SingleAbilityIncrease) Ability() ability.Ability { return c.ability }

// AbilityScoreImprovementKind is the form of an Ability Score Improvement: 2 to
// one ability or 1 to each of two (SRD 5.2.1, "Ability Score Improvement", p.
// 87). The zero value is invalid.
type AbilityScoreImprovementKind int

// The forms.
const (
	// ImprovementOneByTwo raises one ability by 2.
	ImprovementOneByTwo AbilityScoreImprovementKind = iota + 1
	// ImprovementTwoByOne raises each of two distinct abilities by 1.
	ImprovementTwoByOne
)

// AbilityScoreImprovementChoices is the choice made with the Ability Score
// Improvement feat. It is comparable. The zero value is invalid.
type AbilityScoreImprovementChoices struct {
	kind          AbilityScoreImprovementKind
	first, second ability.Ability
}

// NewAbilityScoreImprovementChoices returns the choice of kind. For
// ImprovementOneByTwo second is ignored.
//
// It returns a *FeatChoicesError (matching srderr.ErrInvalidInput) for an
// invalid kind or ability (increasedAbility, abilityNotAllowed) and for two
// equal abilities in the two-ability form (increasedAbility, duplicate). It
// never panics.
func NewAbilityScoreImprovementChoices(kind AbilityScoreImprovementKind, first, second ability.Ability) (AbilityScoreImprovementChoices, error) {
	bad := choicesError(FeatChoiceProblem{FeatChoiceIncreasedAbility, FeatProblemAbilityNotAllowed})
	switch {
	case kind == ImprovementOneByTwo && first.Valid():
		return AbilityScoreImprovementChoices{kind: kind, first: first}, nil
	case kind != ImprovementTwoByOne || !first.Valid() || !second.Valid():
		return AbilityScoreImprovementChoices{}, bad
	case first == second:
		return AbilityScoreImprovementChoices{}, choicesError(FeatChoiceProblem{FeatChoiceIncreasedAbility, FeatProblemDuplicate})
	}
	return AbilityScoreImprovementChoices{kind: kind, first: first, second: second}, nil
}

// MustAbilityScoreImprovementChoices is NewAbilityScoreImprovementChoices for
// constants known to be valid. It panics with the error that function returns,
// and only then.
func MustAbilityScoreImprovementChoices(kind AbilityScoreImprovementKind, first, second ability.Ability) AbilityScoreImprovementChoices {
	c, err := NewAbilityScoreImprovementChoices(kind, first, second)
	if err != nil {
		panic(err)
	}
	return c
}

// Kind returns the form of the improvement.
func (c AbilityScoreImprovementChoices) Kind() AbilityScoreImprovementKind { return c.kind }

// Abilities returns a new slice of the abilities raised: one, or two in the
// order given.
func (c AbilityScoreImprovementChoices) Abilities() []ability.Ability {
	switch c.kind {
	case ImprovementOneByTwo:
		return []ability.Ability{c.first}
	case ImprovementTwoByOne:
		return []ability.Ability{c.first, c.second}
	}
	return nil
}

// Increase converts the choice to an AbilityScoreIncrease with maximum 20. It
// returns the errors of NewAbilityScoreImprovementOneByTwo and
// NewAbilityScoreImprovementTwoByOne, and an error matching
// srderr.ErrInvalidInput for the zero value.
func (c AbilityScoreImprovementChoices) Increase() (AbilityScoreIncrease, error) {
	switch c.kind {
	case ImprovementOneByTwo:
		return NewAbilityScoreImprovementOneByTwo(c.first)
	case ImprovementTwoByOne:
		return NewAbilityScoreImprovementTwoByOne(c.first, c.second)
	}
	return AbilityScoreIncrease{}, invalidInput("zero ability score improvement choices")
}

// maxMagicInitiateCantrips bounds the cantrips a MagicInitiateChoices holds, so
// a hostile command cannot be large; the feat takes two.
const maxMagicInitiateCantrips = 8

// MagicInitiateChoices is the choice made with Magic Initiate: a spell list, the
// ability for the feat's spells, the cantrips, the level 1 spell and the total
// levels at which Spell Change was used (SRD 5.2.1, "Magic Initiate", p. 87). It
// holds a slice, so compare two with Equal. The list and the spells are judged
// when the feat is taken. The zero value is invalid.
type MagicInitiateChoices struct {
	list     content.ClassID
	ability  ability.Ability
	cantrips []content.SpellID
	levelOne content.SpellID
	changed  uint32 // bit n set: Spell Change used at total level n
}

// NewMagicInitiateChoices returns the choice, with cantrips copied. Whether the
// list is Cleric, Druid or Wizard, whether the ability is Intelligence, Wisdom
// or Charisma and whether the spells are right is judged by TakeFeat, which
// reports every problem.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid list,
// ability or spell id, and for more than 8 cantrips. It never panics.
func NewMagicInitiateChoices(list content.ClassID, a ability.Ability, cantrips []content.SpellID, levelOne content.SpellID) (MagicInitiateChoices, error) {
	if !list.Valid() || !a.Valid() || !levelOne.Valid() || len(cantrips) > maxMagicInitiateCantrips ||
		slices.ContainsFunc(cantrips, func(s content.SpellID) bool { return !s.Valid() }) {
		return MagicInitiateChoices{}, invalidInput("magic initiate choices")
	}
	return MagicInitiateChoices{list: list, ability: a, cantrips: slices.Clone(cantrips), levelOne: levelOne}, nil
}

// List returns the chosen spell list.
func (c MagicInitiateChoices) List() content.ClassID { return c.list }

// Ability returns the ability for the feat's spells.
func (c MagicInitiateChoices) Ability() ability.Ability { return c.ability }

// Cantrips returns a new slice of the chosen cantrips.
func (c MagicInitiateChoices) Cantrips() []content.SpellID { return slices.Clone(c.cantrips) }

// LevelOneSpell returns the chosen level 1 spell.
func (c MagicInitiateChoices) LevelOneSpell() content.SpellID { return c.levelOne }

// SpellChangedAt reports whether Spell Change was used at total level l.
func (c MagicInitiateChoices) SpellChangedAt(l Level) bool {
	return l.Value() >= 1 && l.Value() <= MaxLevel && c.changed&(1<<l.Value()) != 0
}

// Equal reports whether c and o hold the same choices.
func (c MagicInitiateChoices) Equal(o MagicInitiateChoices) bool {
	return c.list == o.list && c.ability == o.ability && c.levelOne == o.levelOne && c.changed == o.changed && slices.Equal(c.cantrips, o.cantrips)
}

// replacing returns c with old replaced by repl, Spell Change used at level l.
func (c MagicInitiateChoices) replacing(old, repl content.SpellID, l Level) MagicInitiateChoices {
	out := MagicInitiateChoices{list: c.list, ability: c.ability, cantrips: slices.Clone(c.cantrips), levelOne: c.levelOne, changed: c.changed | 1<<l.Value()}
	if i := slices.Index(out.cantrips, old); i >= 0 {
		out.cantrips[i] = repl
	} else {
		out.levelOne = repl
	}
	return out
}

// SkilledPick is one proficiency chosen with Skilled, a closed set. Variants:
// SkilledSkill, SkilledTool.
type SkilledPick interface{ isSkilledPick() }

// SkilledSkill is a skill picked with Skilled.
type SkilledSkill struct{ Skill content.Skill }

// SkilledTool is a tool picked with Skilled; TakeFeat checks the source holds it.
type SkilledTool struct{ Tool content.ItemID }

func (SkilledSkill) isSkilledPick() {}
func (SkilledTool) isSkilledPick()  {}

// SkilledChoices is the three distinct proficiencies chosen with Skilled (SRD
// 5.2.1, "Skilled", p. 87). It is comparable. The zero value is invalid.
type SkilledChoices struct{ picks [3]SkilledPick }

// NewSkilledChoices returns the choice of exactly three distinct picks.
//
// It returns a *FeatChoicesError (matching srderr.ErrInvalidInput, name
// skilledPicks) of kind wrongCount for another number of picks, duplicate for a
// repeated pick and missing for a nil or invalid pick. It never panics.
func NewSkilledChoices(picks []SkilledPick) (SkilledChoices, error) {
	problem := func(k FeatChoiceProblemKind) error { return choicesError(FeatChoiceProblem{FeatChoiceSkilledPicks, k}) }
	if len(picks) != 3 {
		return SkilledChoices{}, problem(FeatProblemWrongCount)
	}
	for _, p := range picks {
		switch v := p.(type) {
		case SkilledSkill:
			if v.Skill.Valid() {
				continue
			}
		case SkilledTool:
			if v.Tool.Valid() {
				continue
			}
		}
		return SkilledChoices{}, problem(FeatProblemMissing)
	}
	if picks[0] == picks[1] || picks[0] == picks[2] || picks[1] == picks[2] {
		return SkilledChoices{}, problem(FeatProblemDuplicate)
	}
	return SkilledChoices{picks: [3]SkilledPick(picks)}, nil
}

// MustSkilledChoices is NewSkilledChoices for constants known to be valid. It
// panics with the error that function returns, and only then.
func MustSkilledChoices(picks ...SkilledPick) SkilledChoices {
	c, err := NewSkilledChoices(picks)
	if err != nil {
		panic(err)
	}
	return c
}

// Picks returns a new slice of the three picks, or nil for the zero value.
func (c SkilledChoices) Picks() []SkilledPick {
	if c.picks[0] == nil {
		return nil
	}
	return c.picks[:]
}

func (NoFeatChoices) isFeatChoices()                  {}
func (SingleAbilityIncrease) isFeatChoices()          {}
func (AbilityScoreImprovementChoices) isFeatChoices() {}
func (MagicInitiateChoices) isFeatChoices()           {}
func (SkilledChoices) isFeatChoices()                 {}

// equalFeatChoices reports whether a and b are the same choices.
func equalFeatChoices(a, b FeatChoices) bool {
	if x, ok := a.(MagicInitiateChoices); ok {
		y, ok := b.(MagicInitiateChoices)
		return ok && x.Equal(y)
	}
	return a == b
}

// FightingStyleChoice is the choice of a Fighting Style feat, made with the
// Fighting Style class feature (SRD 5.2.1, "Fighting Style", p. 47). It is
// comparable. The zero value is invalid.
type FightingStyleChoice struct{ feat content.FeatID }

// NewFightingStyleChoice returns the choice of feat. It returns an error
// matching srderr.ErrInvalidInput for an invalid id. It never panics.
func NewFightingStyleChoice(feat content.FeatID) (FightingStyleChoice, error) {
	if !feat.Valid() {
		return FightingStyleChoice{}, invalidInput("fighting style choice id")
	}
	return FightingStyleChoice{feat: feat}, nil
}

// Feat returns the chosen feat.
func (c FightingStyleChoice) Feat() content.FeatID { return c.feat }

// ToTakeFeat returns what TakeFeat needs for the choice: the feat, a grant of
// source with the Fighting Style category scope, and NoFeatChoices. It returns
// an error matching srderr.ErrInvalidInput for the zero value or a nil source.
func (c FightingStyleChoice) ToTakeFeat(source FeatSource) (content.FeatID, FeatGrant, FeatChoices, error) {
	if !c.feat.Valid() {
		return "", FeatGrant{}, nil, invalidInput("zero fighting style choice")
	}
	g, err := NewFeatGrant(source, FeatFromCategory{Category: content.FeatFightingStyle})
	if err != nil {
		return "", FeatGrant{}, nil, err
	}
	return c.feat, g, NoFeatChoices{}, nil
}
