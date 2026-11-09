package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// WizardRules is the stateless rule maths of the Wizard over its Features table
// (SRD 5.2.1, "Wizard Features", p. 77). The rows and their bounds belong to the
// content ClassDefinition; there is no Wizard progression type. The zero value is
// ready to use.
type WizardRules struct{}

// slotRow returns the slot spellcasting row of def at level. It returns an error
// matching srderr.ErrInvalidInput for a level outside 1 to 20 and for a class
// without slot spellcasting rows.
func slotRow(def content.ClassDefinition, level int) (content.SlotSpellcastingRow, error) {
	row, err := def.RowAt(level)
	if err != nil {
		return content.SlotSpellcastingRow{}, err
	}
	sc, ok := row.Spellcasting().(content.SlotSpellcastingRow)
	if !ok {
		return content.SlotSpellcastingRow{}, invalidInput("Wizard rules: a class row without Spell Slots")
	}
	return sc, nil
}

// HighestSpellLevel returns the highest Spell Level with at least one Spell Slot
// in the row of the Wizard level. It returns an error matching
// srderr.ErrInvalidInput for a level outside 1 to 20.
func (WizardRules) HighestSpellLevel(wizard content.ClassDefinition, level int) (int, error) {
	sc, err := slotRow(wizard, level)
	if err != nil {
		return 0, err
	}
	slots := sc.SpellSlots()
	for i := len(slots) - 1; i >= 0; i-- {
		if slots[i] > 0 {
			return i + 1, nil
		}
	}
	return 0, nil
}

// NewSpellLevelAt reports whether the highest Spell Level rises at the Wizard level
// (true at level 1 and wherever it rises). It returns an error matching
// srderr.ErrInvalidInput for a level outside 1 to 20.
func (r WizardRules) NewSpellLevelAt(wizard content.ClassDefinition, level int) (bool, error) {
	now, err := r.HighestSpellLevel(wizard, level)
	if err != nil {
		return false, err
	}
	if level == 1 {
		return true, nil
	}
	before, err := r.HighestSpellLevel(wizard, level-1)
	return now > before, err
}

func (WizardRules) cantripColumn(wizard content.ClassDefinition, level int) (int, error) {
	sc, err := slotRow(wizard, level)
	if err != nil {
		return 0, err
	}
	n, _ := sc.Cantrips()
	return n, nil
}

// CantripsGainedAt returns the rise of the Cantrips column at the Wizard level (3 at
// level 1, 1 at levels 4 and 10, else 0). It returns an error matching
// srderr.ErrInvalidInput for a level outside 1 to 20.
func (r WizardRules) CantripsGainedAt(wizard content.ClassDefinition, level int) (int, error) {
	now, err := r.cantripColumn(wizard, level)
	if err != nil || level == 1 {
		return now, err
	}
	before, err := r.cantripColumn(wizard, level-1)
	return now - before, err
}

// CantripsPending returns the Cantrips column at the Wizard level less the cantrips
// known, never below 0. Nothing is stored: a pending choice is derived. It returns an
// error matching srderr.ErrInvalidInput for a level outside 1 to 20.
func (r WizardRules) CantripsPending(wizard content.ClassDefinition, level, known int) (int, error) {
	col, err := r.cantripColumn(wizard, level)
	return max(col-known, 0), err
}

// SpellsOwedAtLevelUp returns the Wizard spells the spellbook is owed on gaining the
// Wizard level: 2 for levels 2 to 20 and 0 otherwise.
func (WizardRules) SpellsOwedAtLevelUp(level int) int {
	if level < 2 || level > 20 {
		return 0
	}
	return 2
}

// ArcaneRecovery holds the rule maths of Arcane Recovery. The zero value is ready to use.
type ArcaneRecovery struct{}

// MaxLevels returns the most Spell Slot levels Arcane Recovery can recover: half the
// Wizard level, rounded up (SRD 5.2.1, "Level 1: Arcane Recovery", p. 78).
func (ArcaneRecovery) MaxLevels(wizardLevel int) int { return (max(wizardLevel, 0) + 1) / 2 }

// arcaneSlotCeiling is the highest slot level Arcane Recovery can recover, 5.
const arcaneSlotCeiling = 5

// ArcaneRecoveryChoice is a validated choice of expended Spell Slots to recover:
// slot level to count. Immutable; the accessors return copies.
type ArcaneRecoveryChoice struct{ slots map[int]int }

// Slots returns a new map of slot level to count recovered.
func (c ArcaneRecoveryChoice) Slots() map[int]int {
	out := make(map[int]int, len(c.slots))
	for k, v := range c.slots {
		out[k] = v
	}
	return out
}

// Levels returns the combined level of the slots.
func (c ArcaneRecoveryChoice) Levels() int {
	n := 0
	for l, c := range c.slots {
		n += l * c
	}
	return n
}

// IsEmpty reports whether nothing is recovered.
func (c ArcaneRecoveryChoice) IsEmpty() bool { return len(c.slots) == 0 }

// NewArcaneRecoveryChoice validates chosen (slot level to count) against expended
// and the maximum levels. Entries with a zero count are ignored. A rejection has a
// non-zero Kind: SlotLevelTooHigh (a level of 6 or higher), then SlotNotExpended
// (more slots of a level than are expended), then ExceedsMaximum, each naming the
// lowest offending level. A negative count or a level outside 1 to 9 is
// SlotNotExpended.
func NewArcaneRecoveryChoice(chosen, expended map[int]int, maxLevels int) (ArcaneRecoveryChoice, ArcaneRecoveryRejection) {
	clean := map[int]int{}
	var levels []int
	for l, n := range chosen {
		if n == 0 {
			continue
		}
		clean[l] = n
		levels = append(levels, l)
	}
	slices.Sort(levels)
	for _, l := range levels {
		if l > arcaneSlotCeiling {
			return ArcaneRecoveryChoice{}, ArcaneRecoveryRejection{Kind: ArcaneRecoverySlotLevelTooHigh, Level: l}
		}
	}
	total := 0
	for _, l := range levels {
		if l < 1 || clean[l] < 0 || clean[l] > expended[l] {
			return ArcaneRecoveryChoice{}, ArcaneRecoveryRejection{Kind: ArcaneRecoverySlotNotExpended, Level: l}
		}
		total += l * clean[l]
	}
	if total > maxLevels {
		return ArcaneRecoveryChoice{}, ArcaneRecoveryRejection{Kind: ArcaneRecoveryExceedsMaximum, Maximum: maxLevels}
	}
	return ArcaneRecoveryChoice{slots: clean}, ArcaneRecoveryRejection{}
}
