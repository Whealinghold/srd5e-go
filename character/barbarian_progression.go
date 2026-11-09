package character

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	// BarbarianClassID is the id of the Barbarian class.
	BarbarianClassID content.ClassID = "barbarian"
	// BerserkerSubclassID is the id of the Path of the Berserker, the only Barbarian subclass in SRD 5.2.1.
	BerserkerSubclassID content.SubclassID = "path_of_the_berserker"

	barbarianRagesColumn         content.ClassColumnID = "rages"
	barbarianRageDamageColumn    content.ClassColumnID = "rage_damage"
	barbarianWeaponMasteryColumn content.ClassColumnID = "weapon_mastery"
)

// barbarianFeatureLevels are the class feature entries the Barbarian rules read, with the level each is gained at
// (SRD 5.2.1, "Barbarian Features", p. 28).
func barbarianFeatureLevels() map[content.FeatureID]int {
	return map[content.FeatureID]int{
		"barbarian_rage": 1, "barbarian_unarmored_defense": 1, "barbarian_weapon_mastery": 1, "barbarian_danger_sense": 2,
		"barbarian_reckless_attack": 2, "barbarian_primal_knowledge": 3, "barbarian_extra_attack": 5, "barbarian_fast_movement": 5,
		"barbarian_feral_instinct": 7, "barbarian_instinctive_pounce": 7, "barbarian_brutal_strike": 9, "barbarian_relentless_rage": 11,
		"barbarian_improved_brutal_strike_level_13": 13, "barbarian_persistent_rage": 15, "barbarian_improved_brutal_strike_level_17": 17,
		"barbarian_indomitable_might": 18, "barbarian_primal_champion": 20,
	}
}

// berserkerFeatureLevels are the Path of the Berserker feature entries (SRD 5.2.1, p. 30).
func berserkerFeatureLevels() map[content.FeatureID]int {
	return map[content.FeatureID]int{
		"path_of_the_berserker_frenzy": 3, "path_of_the_berserker_mindless_rage": 6,
		"path_of_the_berserker_retaliation": 10, "path_of_the_berserker_intimidating_presence": 14,
	}
}

// BarbarianProgressionReason is why NewBarbarianProgression refused a definition. The
// zero value is not a reason.
type BarbarianProgressionReason int

const (
	// NotTheBarbarianDefinition: the definition is not the Barbarian's.
	NotTheBarbarianDefinition BarbarianProgressionReason = iota + 1
	// BarbarianColumnMissing: the rages, rage_damage or weapon_mastery column is missing or of the wrong kind.
	BarbarianColumnMissing
	// BarbarianRowsIncomplete: a row of levels 1 to 20 is missing or holds no value in a column.
	BarbarianRowsIncomplete
	// BarbarianFeatureMissing: a feature the Barbarian rules read is missing or at another level.
	BarbarianFeatureMissing
	// BarbarianSubclassMissing: the Path of the Berserker, or one of its four features at its level, is missing.
	BarbarianSubclassMissing
)

// String returns "not_the_barbarian_definition", "column_missing", "rows_incomplete",
// "feature_missing", "subclass_missing" or "unknown".
func (r BarbarianProgressionReason) String() string {
	switch r {
	case NotTheBarbarianDefinition:
		return "not_the_barbarian_definition"
	case BarbarianColumnMissing:
		return "column_missing"
	case BarbarianRowsIncomplete:
		return "rows_incomplete"
	case BarbarianFeatureMissing:
		return "feature_missing"
	case BarbarianSubclassMissing:
		return "subclass_missing"
	}
	return "unknown"
}

// BarbarianProgressionError reports one reason NewBarbarianProgression refused a
// definition. It matches srderr.ErrInvalidInput with errors.Is and carries no
// display text. Build it with a keyed literal.
type BarbarianProgressionError struct{ Reason BarbarianProgressionReason }

// Error renders the error. The text is not a contract.
func (e *BarbarianProgressionError) Error() string {
	return "character: invalid Barbarian progression: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*BarbarianProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *BarbarianProgressionError of the same Reason.
func (e *BarbarianProgressionError) Is(target error) bool {
	t, ok := target.(*BarbarianProgressionError)
	return ok && t.Reason == e.Reason
}

// BarbarianLevelStats are the numbers a Barbarian level gives (SRD 5.2.1,
// "Barbarian Features", p. 28): Rage uses, the Rage Damage bonus and Weapon Mastery
// kinds. It carries no Proficiency Bonus, which follows the total level (SRD 5.2.1,
// "Multiclassing", p. 25). Only BarbarianProgression.StatsAt builds one; the zero value is
// invalid. Immutable, comparable with ==, safe for concurrent use.
type BarbarianLevelStats struct {
	level                            BarbarianLevel
	rages, rageDamage, weaponMastery int
}

// Level returns the Barbarian level the numbers belong to.
func (s BarbarianLevelStats) Level() BarbarianLevel { return s.level }

// Rages returns the maximum uses of Rage.
func (s BarbarianLevelStats) Rages() int { return s.rages }

// RageDamage returns the Rage Damage bonus.
func (s BarbarianLevelStats) RageDamage() int { return s.rageDamage }

// WeaponMastery returns how many kinds of weapon the Barbarian has Weapon Mastery with.
func (s BarbarianLevelStats) WeaponMastery() int { return s.weaponMastery }

// IsZero reports whether s is the invalid zero value.
func (s BarbarianLevelStats) IsZero() bool { return s.level.IsZero() }

// BarbarianProgression turns a Barbarian level into BarbarianLevelStats. Rages, Rage
// Damage and Weapon Mastery kinds are read from the columns of the Barbarian
// ClassDefinition when it is built; it holds no port and no clock. It is immutable
// and comparable with ==; the zero value is invalid (see IsZero). Safe for
// concurrent use.
type BarbarianProgression struct {
	rages, rageDamage, weaponMastery [MaxLevel]int
	set                              bool
}

// NewBarbarianProgression reads the Rages, Rage Damage and Weapon Mastery values of
// the levels 1 to 20 from def. It checks structure only, never a value against the
// SRD, so house-rule content is accepted.
//
// It returns an error joining a *BarbarianProgressionError per discrepancy,
// matching srderr.ErrInvalidInput: NotTheBarbarianDefinition for a definition of
// another class or the zero definition, BarbarianColumnMissing for a missing or
// mistyped column, BarbarianRowsIncomplete for a level without a value,
// BarbarianFeatureMissing for a feature the rules read that is missing or at another
// level and BarbarianSubclassMissing for the Path of the Berserker or one of its
// features missing or misplaced. It never panics.
func NewBarbarianProgression(def content.ClassDefinition) (BarbarianProgression, error) {
	if def.ID() != BarbarianClassID {
		return BarbarianProgression{}, errors.Join(&BarbarianProgressionError{Reason: NotTheBarbarianDefinition})
	}
	var problems []error
	add := func(r BarbarianProgressionReason) { problems = append(problems, &BarbarianProgressionError{Reason: r}) }
	kinds := map[content.ClassColumnID]content.ClassColumnKind{
		barbarianRagesColumn: content.ColumnKindCount, barbarianRageDamageColumn: content.ColumnKindBonus, barbarianWeaponMasteryColumn: content.ColumnKindCount,
	}
	for _, id := range []content.ClassColumnID{barbarianRagesColumn, barbarianRageDamageColumn, barbarianWeaponMasteryColumn} {
		if col, err := def.Column(id); err != nil || col.Kind() != kinds[id] {
			add(BarbarianColumnMissing)
		}
	}
	for id, level := range barbarianFeatureLevels() {
		if found, ok := def.Feature(id).(content.FeatureEntryFound); !ok || found.Entry().Level() != level {
			add(BarbarianFeatureMissing)
		}
	}
	if missingBerserker(def) {
		add(BarbarianSubclassMissing)
	}
	if len(problems) > 0 {
		return BarbarianProgression{}, errors.Join(problems...)
	}
	var p BarbarianProgression
	for level := 1; level <= MaxLevel; level++ {
		row, _ := def.RowAt(level) // a definition holds rows 1 to 20, so the level is in range
		rages, ok1 := countAt(row, barbarianRagesColumn)
		wm, ok2 := countAt(row, barbarianWeaponMasteryColumn)
		v, _ := row.Value(barbarianRageDamageColumn) // a missing column leaves v nil, which is not a bonus
		dmg, ok3 := v.(content.ColumnBonus)
		if !ok1 || !ok2 || !ok3 {
			return BarbarianProgression{}, errors.Join(&BarbarianProgressionError{Reason: BarbarianRowsIncomplete})
		}
		p.rages[level-1], p.rageDamage[level-1], p.weaponMastery[level-1] = rages, dmg.Value(), wm
	}
	p.set = true
	return p, nil
}

// missingBerserker reports whether the Path of the Berserker or one of its four features at its level is missing.
func missingBerserker(def content.ClassDefinition) bool {
	found, ok := def.Subclass(BerserkerSubclassID).(content.SubclassFound)
	if !ok {
		return true
	}
	have := map[content.FeatureID]int{}
	for _, f := range found.Subclass().Features() {
		have[f.ID()] = f.Level()
	}
	for id, level := range berserkerFeatureLevels() {
		if l, ok := have[id]; !ok || l != level {
			return true
		}
	}
	return false
}

// MustBarbarianProgression is NewBarbarianProgression for a definition known to be
// valid. It panics with the error NewBarbarianProgression returns, and only then.
func MustBarbarianProgression(def content.ClassDefinition) BarbarianProgression {
	p, err := NewBarbarianProgression(def)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p BarbarianProgression) IsZero() bool { return !p.set }

// StatsAt returns the numbers of the Barbarian level; the zero BarbarianLevelStats
// for the zero progression or level.
func (p BarbarianProgression) StatsAt(level BarbarianLevel) BarbarianLevelStats {
	if !p.set || level.IsZero() {
		return BarbarianLevelStats{}
	}
	n := level.n
	return BarbarianLevelStats{level: level, rages: p.rages[n-1], rageDamage: p.rageDamage[n-1], weaponMastery: p.weaponMastery[n-1]}
}

// BarbarianChoice is a Barbarian-specific choice due on gaining a level, a closed
// set. Variants: BarbarianChoiceWeaponMasteries, BarbarianChoicePrimalKnowledgeSkill.
type BarbarianChoice interface{ isBarbarianChoice() }

// BarbarianChoiceWeaponMasteries is the choice of Count more kinds of weapon to
// have Weapon Mastery with. Construct it with a keyed literal.
type BarbarianChoiceWeaponMasteries struct{ Count int }

// BarbarianChoicePrimalKnowledgeSkill is the choice of one more skill from the
// Barbarian's level 1 skill list at level 3 (SRD 5.2.1, "Level 3: Primal Knowledge", p. 29).
type BarbarianChoicePrimalKnowledgeSkill struct{}

func (BarbarianChoiceWeaponMasteries) isBarbarianChoice()      {}
func (BarbarianChoicePrimalKnowledgeSkill) isBarbarianChoice() {}

// ChoicesOnGaining returns the Barbarian-specific choices due on gaining the
// Barbarian level, in order: the Weapon Mastery kinds the level adds (2 at level 1,
// then 1 each at 4 and 10 with the SRD columns) and, at level 3, the Primal Knowledge
// skill. A level with none gives an empty slice. The Ability Score Improvement, Epic
// Boon and subclass choices belong to level advancement and are not reported. The zero
// progression or level gives none.
func (p BarbarianProgression) ChoicesOnGaining(level BarbarianLevel) []BarbarianChoice {
	out := []BarbarianChoice{}
	if !p.set || level.IsZero() {
		return out
	}
	added := p.weaponMastery[level.n-1]
	if level.n > 1 {
		added -= p.weaponMastery[level.n-2]
	}
	if added > 0 {
		out = append(out, BarbarianChoiceWeaponMasteries{Count: added})
	}
	if level.n == primalKnowledgeLevel {
		out = append(out, BarbarianChoicePrimalKnowledgeSkill{})
	}
	return out
}
