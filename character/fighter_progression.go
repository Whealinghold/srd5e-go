package character

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	// FighterClassID is the id of the Fighter class.
	FighterClassID content.ClassID = "fighter"
	// ChampionSubclassID is the id of the Champion subclass, the only Fighter subclass in SRD 5.2.1.
	ChampionSubclassID content.SubclassID = "champion"

	fighterSecondWindColumn    content.ClassColumnID = "second_wind"
	fighterWeaponMasteryColumn content.ClassColumnID = "weapon_mastery"
)

// FighterProgressionReason is why NewFighterProgression refused a definition. The
// zero value is not a reason.
type FighterProgressionReason int

const (
	// NotTheFighterDefinition: the definition is not the Fighter's.
	NotTheFighterDefinition FighterProgressionReason = iota + 1
	// FighterColumnMissing: the second_wind or weapon_mastery column is missing or not a count column.
	FighterColumnMissing
	// FighterRowsIncomplete: a row of levels 1 to 20 is missing or holds no count in a column.
	FighterRowsIncomplete
)

// String returns "not_the_fighter_definition", "column_missing", "rows_incomplete" or "unknown".
func (r FighterProgressionReason) String() string {
	switch r {
	case NotTheFighterDefinition:
		return "not_the_fighter_definition"
	case FighterColumnMissing:
		return "column_missing"
	case FighterRowsIncomplete:
		return "rows_incomplete"
	}
	return "unknown"
}

// FighterProgressionError reports one reason NewFighterProgression refused a
// definition. It matches srderr.ErrInvalidInput with errors.Is and carries no
// display text. Build it with a keyed literal.
type FighterProgressionError struct{ Reason FighterProgressionReason }

// Error renders the error. The text is not a contract.
func (e *FighterProgressionError) Error() string {
	return "character: invalid Fighter progression: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*FighterProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *FighterProgressionError of the same Reason.
func (e *FighterProgressionError) Is(target error) bool {
	t, ok := target.(*FighterProgressionError)
	return ok && t.Reason == e.Reason
}

// FighterLevelStats are the numbers a Fighter level gives (SRD 5.2.1, "Fighter
// Features", p. 47): Second Wind uses, Weapon Mastery kinds, Action Surge uses,
// Indomitable uses and attacks per Attack action. It carries no Proficiency
// Bonus, which follows the total level (SRD 5.2.1, "Multiclassing", p. 25). Only
// FighterProgression.StatsAt builds one; the zero value is invalid. Immutable,
// comparable with ==, safe for concurrent use.
type FighterLevelStats struct {
	level                                                        FighterLevel
	secondWind, weaponMastery, actionSurge, indomitable, attacks int
}

// Level returns the Fighter level the numbers belong to.
func (s FighterLevelStats) Level() FighterLevel { return s.level }

// SecondWind returns the maximum uses of Second Wind.
func (s FighterLevelStats) SecondWind() int { return s.secondWind }

// WeaponMastery returns how many kinds of weapon the Fighter has Weapon Mastery with.
func (s FighterLevelStats) WeaponMastery() int { return s.weaponMastery }

// ActionSurge returns the maximum uses of Action Surge; 0 below level 2.
func (s FighterLevelStats) ActionSurge() int { return s.actionSurge }

// Indomitable returns the maximum uses of Indomitable; 0 below level 9.
func (s FighterLevelStats) Indomitable() int { return s.indomitable }

// Attacks returns the attacks per Attack action, from the Fighter level alone.
func (s FighterLevelStats) Attacks() int { return s.attacks }

// IsZero reports whether s is the invalid zero value.
func (s FighterLevelStats) IsZero() bool { return s.level.IsZero() }

// FighterProgression turns a Fighter level into FighterLevelStats. Second Wind
// uses and Weapon Mastery kinds are read from the columns of the Fighter
// ClassDefinition when it is built; Action Surge, Indomitable and attack counts
// are rule constants, checked against the printed row annotations by a test. It
// holds no port and no clock. It is immutable and comparable with ==; the zero
// value is invalid (see IsZero). Safe for concurrent use.
type FighterProgression struct {
	secondWind, weaponMastery [MaxLevel]int
	set                       bool
}

// NewFighterProgression reads the Second Wind and Weapon Mastery counts of the
// levels 1 to 20 from def.
//
// It returns an error joining a *FighterProgressionError per reason found,
// matching srderr.ErrInvalidInput: NotTheFighterDefinition for a definition of
// another class or the zero definition, FighterColumnMissing for a missing or
// non-count column and FighterRowsIncomplete for a level without a count. It
// never panics.
func NewFighterProgression(def content.ClassDefinition) (FighterProgression, error) {
	if def.ID() != FighterClassID {
		return FighterProgression{}, errors.Join(&FighterProgressionError{Reason: NotTheFighterDefinition})
	}
	var problems []error
	for _, id := range []content.ClassColumnID{fighterSecondWindColumn, fighterWeaponMasteryColumn} {
		col, err := def.Column(id)
		if err != nil || col.Kind() != content.ColumnKindCount {
			problems = append(problems, &FighterProgressionError{Reason: FighterColumnMissing})
			break
		}
	}
	if len(problems) > 0 {
		return FighterProgression{}, errors.Join(problems...)
	}
	var p FighterProgression
	for level := 1; level <= MaxLevel; level++ {
		row, _ := def.RowAt(level) // a definition holds rows 1 to 20, so the level is in range
		sw, ok1 := countAt(row, fighterSecondWindColumn)
		wm, ok2 := countAt(row, fighterWeaponMasteryColumn)
		if !ok1 || !ok2 {
			return FighterProgression{}, errors.Join(&FighterProgressionError{Reason: FighterRowsIncomplete})
		}
		p.secondWind[level-1], p.weaponMastery[level-1] = sw, wm
	}
	p.set = true
	return p, nil
}

// countAt returns the count of column id in row, and false when the cell holds no count
// (a dash, or a column the row lacks).
func countAt(row content.ProgressionRow, id content.ClassColumnID) (int, bool) {
	v, _ := row.Value(id) // a missing column leaves v nil, which is not a count
	c, ok := v.(content.ColumnCount)
	return c.Value(), ok
}

// MustFighterProgression is NewFighterProgression for a definition known to be
// valid. It panics with the error NewFighterProgression returns, and only then.
func MustFighterProgression(def content.ClassDefinition) FighterProgression {
	p, err := NewFighterProgression(def)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p FighterProgression) IsZero() bool { return !p.set }

// StatsAt returns the numbers of the Fighter level; the zero FighterLevelStats
// for the zero progression or level.
func (p FighterProgression) StatsAt(level FighterLevel) FighterLevelStats {
	if !p.set || level.IsZero() {
		return FighterLevelStats{}
	}
	n := level.n
	return FighterLevelStats{
		level: level, secondWind: p.secondWind[n-1], weaponMastery: p.weaponMastery[n-1],
		actionSurge: thresholdCount(n, 2, 17), indomitable: thresholdCount(n, 9, 13, 17),
		attacks: 1 + thresholdCount(n, 5, 11, 20),
	}
}

// thresholdCount returns how many of the ascending thresholds n has reached.
func thresholdCount(n int, thresholds ...int) int {
	count := 0
	for _, t := range thresholds {
		if n >= t {
			count++
		}
	}
	return count
}

// FighterChoice is a Fighter-specific choice due on gaining a level, a closed
// set. Variants: FighterChoiceFightingStyle, FighterChoiceWeaponMasteries,
// FighterChoiceAdditionalFightingStyle.
type FighterChoice interface{ isFighterChoice() }

// FighterChoiceFightingStyle is the choice of a Fighting Style feat at level 1.
type FighterChoiceFightingStyle struct{}

// FighterChoiceWeaponMasteries is the choice of Count more kinds of weapon to
// have Weapon Mastery with. Construct it with a keyed literal.
type FighterChoiceWeaponMasteries struct{ Count int }

// FighterChoiceAdditionalFightingStyle is the Champion's choice of another
// Fighting Style feat at level 7.
type FighterChoiceAdditionalFightingStyle struct{}

func (FighterChoiceFightingStyle) isFighterChoice()           {}
func (FighterChoiceWeaponMasteries) isFighterChoice()         {}
func (FighterChoiceAdditionalFightingStyle) isFighterChoice() {}

// ChoicesOnGaining returns the Fighter-specific choices due on gaining the
// Fighter level, in order: the Fighting Style at level 1, the Weapon Mastery kinds
// the level adds (3 at level 1, then 1 each at 4, 10 and 16) and, for the
// Champion subclass at level 7, the Additional Fighting Style. A level with none
// gives an empty slice. subclass is the chosen subclass, or empty for none. The
// Ability Score Improvement, Epic Boon and subclass choices belong to level
// advancement and are not reported. The zero progression or level gives none.
func (p FighterProgression) ChoicesOnGaining(level FighterLevel, subclass content.SubclassID) []FighterChoice {
	out := []FighterChoice{}
	if !p.set || level.IsZero() {
		return out
	}
	if level.n == 1 {
		out = append(out, FighterChoiceFightingStyle{})
	}
	added := p.weaponMastery[level.n-1]
	if level.n > 1 {
		added -= p.weaponMastery[level.n-2]
	}
	if added > 0 {
		out = append(out, FighterChoiceWeaponMasteries{Count: added})
	}
	if level.n == championStyleLevel && subclass == ChampionSubclassID {
		out = append(out, FighterChoiceAdditionalFightingStyle{})
	}
	return out
}
