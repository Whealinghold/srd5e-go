package character

import (
	"errors"
	"strconv"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// AbilityScoreGenerationMethod is a way to generate starting ability scores
// (SRD 5.2.1, "Generate Your Scores", p. 21). The zero value is invalid.
// Comparable and safe for concurrent use.
type AbilityScoreGenerationMethod int

// The ability score generation methods.
const (
	// StandardArray: assign the standard array.
	StandardArray AbilityScoreGenerationMethod = iota + 1
	// RandomGeneration: roll the scores.
	RandomGeneration
	// PointCost: spend points.
	PointCost
)

// Valid reports whether m is one of the three methods.
func (m AbilityScoreGenerationMethod) Valid() bool { return m >= StandardArray && m <= PointCost }

// String returns "standardArray", "randomGeneration", "pointCost" or "unknown".
func (m AbilityScoreGenerationMethod) String() string {
	switch m {
	case StandardArray:
		return "standardArray"
	case RandomGeneration:
		return "randomGeneration"
	case PointCost:
		return "pointCost"
	}
	return "unknown"
}

// RuleOptionsParameter names a RuleOptions parameter in a problem. The zero
// value is invalid.
type RuleOptionsParameter int

// The RuleOptions parameters, in the order problems are reported.
const (
	// AbilityScoreMethods: the permitted generation methods.
	AbilityScoreMethods RuleOptionsParameter = iota + 1
	// StartingLevel: the level new characters start at.
	StartingLevel
)

// String returns "abilityScoreMethods", "startingLevel" or "unknown".
func (p RuleOptionsParameter) String() string {
	switch p {
	case AbilityScoreMethods:
		return "abilityScoreMethods"
	case StartingLevel:
		return "startingLevel"
	}
	return "unknown"
}

// RuleOptionsProblemKind says what is wrong with a RuleOptions parameter. The
// zero value is invalid.
type RuleOptionsProblemKind int

// The kinds of problem.
const (
	// RuleOptionsEmpty: no method is permitted.
	RuleOptionsEmpty RuleOptionsProblemKind = iota + 1
	// RuleOptionsBelowMinimum: the starting level is below 1.
	RuleOptionsBelowMinimum
	// RuleOptionsAboveMaximum: the starting level is above 20.
	RuleOptionsAboveMaximum
	// RuleOptionsUnknownMethod: a permitted method is not one of the three.
	RuleOptionsUnknownMethod
)

// String returns the kind's name or "unknown".
func (k RuleOptionsProblemKind) String() string {
	switch k {
	case RuleOptionsEmpty:
		return "empty"
	case RuleOptionsBelowMinimum:
		return "belowMinimum"
	case RuleOptionsAboveMaximum:
		return "aboveMaximum"
	case RuleOptionsUnknownMethod:
		return "unknownMethod"
	}
	return "unknown"
}

// RuleOptionsProblem is one invalid RuleOptions parameter: which one, what is
// wrong and the offending value (the starting level, or the method number; 0
// for an empty set). It matches srderr.ErrInvalidInput with errors.Is and
// carries no display text. It is comparable.
//
//nolint:errname // the problem type named by the rule-options spec
type RuleOptionsProblem struct {
	Parameter RuleOptionsParameter
	Kind      RuleOptionsProblemKind
	Value     int
}

// Error renders the problem. The text is not a contract.
func (p RuleOptionsProblem) Error() string {
	return "character: invalid rule option " + p.Parameter.String() + ": " + p.Kind.String() + " (" + strconv.Itoa(p.Value) + ")"
}

// Unwrap returns srderr.ErrInvalidInput.
func (p RuleOptionsProblem) Unwrap() error { return srderr.ErrInvalidInput }

// RuleOptions are the table options of character creation and advancement: the
// permitted ability score generation methods (SRD 5.2.1, "Generate Your
// Scores", p. 21), the starting level ("Starting at Higher Levels", p. 24) and
// "Bonus Feats at Level 20" (p. 24). An option is a value passed in a command,
// never a global setting. It is immutable; compare two with Equal. The zero
// value is invalid: use StandardRuleOptions or NewRuleOptions. Safe for
// concurrent use.
type RuleOptions struct {
	methods       uint8 // bit m set: method m permitted
	startingLevel int
	bonus         BonusFeatsAtLevel20
}

// NewRuleOptions returns the options permitting methods (a set: duplicates
// collapse) with the given starting level and bonus feats option; a nil bonus
// means NoBonusFeats.
//
// It returns the errors.Join of every RuleOptionsProblem, methods before the
// starting level, matching srderr.ErrInvalidInput: an empty set, an unknown
// method, a level below 1 or above 20. It never panics.
func NewRuleOptions(methods []AbilityScoreGenerationMethod, startingLevel int, bonus BonusFeatsAtLevel20) (RuleOptions, error) {
	var problems []error
	var mask uint8
	for _, m := range methods {
		if m.Valid() {
			mask |= 1 << m
		} else {
			problems = append(problems, RuleOptionsProblem{AbilityScoreMethods, RuleOptionsUnknownMethod, int(m)})
		}
	}
	if len(methods) == 0 {
		problems = append(problems, RuleOptionsProblem{AbilityScoreMethods, RuleOptionsEmpty, 0})
	}
	if _, err := NewLevel(startingLevel); err != nil {
		var le *LevelError
		kind := RuleOptionsBelowMinimum
		if errors.As(err, &le) && le.Reason == LevelAboveMaximum {
			kind = RuleOptionsAboveMaximum
		}
		problems = append(problems, RuleOptionsProblem{StartingLevel, kind, startingLevel})
	}
	if len(problems) > 0 {
		return RuleOptions{}, errors.Join(problems...)
	}
	if bonus == nil {
		bonus = NoBonusFeats{}
	}
	return RuleOptions{methods: mask, startingLevel: startingLevel, bonus: bonus}, nil
}

// MustRuleOptions is NewRuleOptions for constants known to be valid. It panics
// with the error NewRuleOptions returns, and only then; errors.As finds the
// first problem, a methods problem when both parameters are invalid.
func MustRuleOptions(methods []AbilityScoreGenerationMethod, startingLevel int, bonus BonusFeatsAtLevel20) RuleOptions {
	o, err := NewRuleOptions(methods, startingLevel, bonus)
	if err != nil {
		panic(err)
	}
	return o
}

// StandardRuleOptions returns the standard options: all three methods
// permitted, starting level 1, NoBonusFeats.
func StandardRuleOptions() RuleOptions {
	return RuleOptions{methods: 1<<StandardArray | 1<<RandomGeneration | 1<<PointCost, startingLevel: MinLevel, bonus: NoBonusFeats{}}
}

// PermittedMethods returns a new slice of the permitted methods in declaration
// order.
func (o RuleOptions) PermittedMethods() []AbilityScoreGenerationMethod {
	var out []AbilityScoreGenerationMethod
	for m := StandardArray; m <= PointCost; m++ {
		if o.methods&(1<<m) != 0 {
			out = append(out, m)
		}
	}
	return out
}

// StartingLevel returns the level new characters start at.
func (o RuleOptions) StartingLevel() int { return o.startingLevel }

// BonusFeatsAtLevel20 returns the bonus feats option.
func (o RuleOptions) BonusFeatsAtLevel20() BonusFeatsAtLevel20 { return o.bonus }

// IsZero reports whether o is the invalid zero RuleOptions.
func (o RuleOptions) IsZero() bool { return o.methods == 0 }

// Equal reports whether o and p hold the same methods (in any order), starting
// level and bonus feats option.
func (o RuleOptions) Equal(p RuleOptions) bool {
	return o.methods == p.methods && o.startingLevel == p.startingLevel && o.bonus == p.bonus
}

// WithPermittedMethods returns a copy of o with the permitted methods replaced.
// It returns the errors NewRuleOptions would; o is unchanged.
func (o RuleOptions) WithPermittedMethods(methods []AbilityScoreGenerationMethod) (RuleOptions, error) {
	return NewRuleOptions(methods, o.startingLevel, o.bonus)
}

// WithStartingLevel returns a copy of o with the starting level replaced. It
// returns the errors NewRuleOptions would; o is unchanged.
func (o RuleOptions) WithStartingLevel(level int) (RuleOptions, error) {
	return NewRuleOptions(o.PermittedMethods(), level, o.bonus)
}

// WithBonusFeatsAtLevel20 returns a copy of o with the bonus feats option
// replaced (nil means NoBonusFeats); o is unchanged.
func (o RuleOptions) WithBonusFeatsAtLevel20(bonus BonusFeatsAtLevel20) (RuleOptions, error) {
	return NewRuleOptions(o.PermittedMethods(), o.startingLevel, bonus)
}
