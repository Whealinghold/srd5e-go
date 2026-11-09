package character

// ProficiencyBonus is the bonus a character adds for proficiency, +2 to +6
// (SRD 5.2.1, "Proficiency", p. 8; "Level Advancement", p. 23). It is an
// immutable value, comparable with ==. The zero ProficiencyBonus is invalid;
// see IsZero. Safe for concurrent use.
type ProficiencyBonus struct{ n int }

// ProficiencyBonusFor returns the Proficiency Bonus of a total level: +2 for
// levels 1 to 4, +3 for 5 to 8, +4 for 9 to 12, +5 for 13 to 16 and +6 for 17
// to 20 (SRD 5.2.1, "Level Advancement", p. 23). A multiclass character uses
// its total level (p. 25). The zero Level gives the zero bonus.
func ProficiencyBonusFor(level Level) ProficiencyBonus {
	if level.IsZero() {
		return ProficiencyBonus{}
	}
	return ProficiencyBonus{n: 2 + (level.n-1)/4}
}

// Value returns the bonus as a plain integer (2 to 6), or 0 for the zero value.
// Callers convert it to a modifier.Modifier.
func (b ProficiencyBonus) Value() int { return b.n }

// IsZero reports whether b is the invalid zero ProficiencyBonus.
func (b ProficiencyBonus) IsZero() bool { return b.n == 0 }
