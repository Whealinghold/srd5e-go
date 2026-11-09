package character

import (
	"strconv"
	"strings"
)

// The Features tables of the spellcasting classes as space-separated rows, one
// per class level 1 to 20 (SRD 5.2.1, pp. 31, 36, 41, 53, 58, 65, 71, 77).
// Package-level tables would be variables (check A4), so they are constants.
const (
	// fullCasterRows (Table C): one digit per spell level 1, 2, 3, ... with slots.
	fullCasterRows = "2 3 42 43 432 433 4331 4332 43331 43332 433321 433321 4333211 4333211 43332111 43332111 433321111 433331111 433332111 433332211"
	// halfCasterRows (Table D): Paladin and Ranger.
	halfCasterRows = "2 2 3 3 42 42 43 43 432 432 433 433 4331 4331 4332 4332 43331 43331 43332 43332"
	// pactMagicRows (Table E): slot count then Slot Level.
	pactMagicRows = "11 21 22 22 23 23 24 24 25 25 35 35 35 35 35 35 45 45 45 45"

	// Table G, prepared level 1+ spells.
	preparedBardClericDruid = "4 5 6 7 9 10 11 12 14 15 16 16 17 17 18 18 19 20 21 22"
	preparedHalfCaster      = "2 3 4 5 6 6 7 7 9 9 10 10 11 11 12 12 14 14 15 15"
	preparedSorcerer        = "2 4 6 7 9 10 11 12 14 15 16 16 17 17 18 18 19 20 21 22"
	preparedWarlock         = "2 3 4 5 6 7 8 9 10 10 11 11 12 12 13 13 14 14 15 15"
	preparedWizard          = "4 5 6 7 9 10 11 12 14 15 16 16 17 18 19 21 22 23 24 25"
)

// rowAt returns the row for a level from a table constant, "" outside 1 to 20.
func rowAt(rows string, level int) string {
	f := strings.Fields(rows)
	if level < 1 || level > len(f) {
		return ""
	}
	return f[level-1]
}

// slotsAt returns the slots per spell level 1 to 9 of a Table C or D row.
func slotsAt(rows string, level int) [9]int {
	var s [9]int
	for i, r := range rowAt(rows, level) {
		s[i] = int(r - '0')
	}
	return s
}

// pactMagicAt returns the Pact Magic slot count and Slot Level of Table E at a
// Warlock level; 0, 0 outside 1 to 20.
func pactMagicAt(level int) (count, slotLevel int) {
	r := rowAt(pactMagicRows, level)
	if r == "" {
		return 0, 0
	}
	return int(r[0] - '0'), int(r[1] - '0')
}

// PreparedLimit returns the number of prepared level 1+ spells the class has at
// a class level (Table G), counted from that class's level alone (SRD 5.2.1,
// "Multiclassing", "Spells Prepared", p. 25). It returns 0 for an invalid class
// or a level outside 1 to 20.
func PreparedLimit(c SpellcastingClass, classLevel int) int {
	var rows string
	switch c {
	case CasterBard, CasterCleric, CasterDruid:
		rows = preparedBardClericDruid
	case CasterPaladin, CasterRanger:
		rows = preparedHalfCaster
	case CasterSorcerer:
		rows = preparedSorcerer
	case CasterWarlock:
		rows = preparedWarlock
	case CasterWizard:
		rows = preparedWizard
	}
	n, _ := strconv.Atoi(rowAt(rows, classLevel)) // "" (no row) reads as 0
	return n
}

// CantripLimit returns the cantrips the class knows from its Spellcasting or
// Pact Magic feature at a class level (Table F). Paladin and Ranger have none.
// It returns 0 for an invalid class or a level outside 1 to 20.
func CantripLimit(c SpellcastingClass, classLevel int) int {
	if classLevel < 1 || classLevel > 20 {
		return 0
	}
	band := 0
	switch {
	case classLevel >= 10:
		band = 2
	case classLevel >= 4:
		band = 1
	}
	var counts string
	switch c {
	case CasterBard, CasterDruid, CasterWarlock:
		counts = "234"
	case CasterCleric, CasterWizard:
		counts = "345"
	case CasterSorcerer:
		counts = "456"
	case CasterPaladin, CasterRanger:
		return 0
	default:
		return 0
	}
	return int(counts[band] - '0')
}

// MaxPreparableLevel returns the highest spell level the class may prepare at a
// class level: the Warlock's Slot Level at that level (Table E), otherwise the
// highest spell level with slots in the class's own table (C or D), as if the
// class were single-classed (SRD 5.2.1, "Bard", p. 32; "Warlock", p. 71). It
// returns 0 for an invalid class or a level outside 1 to 20.
func MaxPreparableLevel(c SpellcastingClass, classLevel int) int {
	if c == CasterWarlock {
		_, lvl := pactMagicAt(classLevel)
		return lvl
	}
	rows := fullCasterRows
	if c.isHalfCaster() {
		rows = halfCasterRows
	}
	if !c.Valid() {
		return 0
	}
	return len(rowAt(rows, classLevel))
}
