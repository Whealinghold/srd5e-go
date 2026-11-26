package combat

import (
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// reincarnateRow is a row of the Reincarnate species table: roll again, or a species.
type reincarnateRow struct {
	species   content.SpeciesID
	rollAgain bool
}

// reincarnateTable returns table R of spell-outcome-tables.md (SRD 5.2.1, "Reincarnate", p. 158).
func reincarnateTable() outcomeTable[reincarnateRow] {
	species := []content.SpeciesID{"dragonborn", "dwarf", "elf", "gnome", "goliath", "halfling", "human", "orc", "tiefling"}
	bands := []outcomeBand[reincarnateRow]{{1, 1, reincarnateRow{rollAgain: true}}}
	for i, s := range species {
		bands = append(bands, outcomeBand[reincarnateRow]{i + 2, i + 2, reincarnateRow{species: s}})
	}
	return newOutcomeTable(dice.D10(), bands)
}

// ReincarnationSpeciesFor returns the species of the 1d10 face, or rollAgain true with no species for a 1 (SRD 5.2.1,
// "Reincarnate", p. 158). It returns an error matching ErrFaceOutOfRange for a face outside 1 to 10. It never
// panics.
func ReincarnationSpeciesFor(face int) (species content.SpeciesID, rollAgain bool, err error) {
	row, err := reincarnateTable().lookup(face)
	return row.species, row.rollAgain, err
}

// ReincarnationOutcome is the closed set of species a Reincarnate gives. Variants: ReincarnationSpeciesRolled,
// ReincarnationSpeciesChosen. Neither changes the creature: the new species' traits replace the old ones, and
// applying that is the host's.
type ReincarnationOutcome interface{ isReincarnationOutcome() }

// ReincarnationSpeciesRolled: the species of the table, after Rerolls rolls of 1.
type ReincarnationSpeciesRolled struct {
	species content.SpeciesID
	rerolls int
}

// Species returns the species rolled.
func (o ReincarnationSpeciesRolled) Species() content.SpeciesID { return o.species }

// Rerolls returns the number of 1s rolled before the species.
func (o ReincarnationSpeciesRolled) Rerolls() int { return o.rerolls }

// ReplacesSpeciesTraits returns true: the new species' traits replace the old ones.
func (ReincarnationSpeciesRolled) ReplacesSpeciesTraits() bool { return true }

// ReincarnationSpeciesChosen: the species the Game Master chose; no die was rolled.
type ReincarnationSpeciesChosen struct{ species content.SpeciesID }

// Species returns the species chosen.
func (o ReincarnationSpeciesChosen) Species() content.SpeciesID { return o.species }

// ReplacesSpeciesTraits returns true: the new species' traits replace the old ones.
func (ReincarnationSpeciesChosen) ReplacesSpeciesTraits() bool { return true }

func (ReincarnationSpeciesRolled) isReincarnationOutcome() {}
func (ReincarnationSpeciesChosen) isReincarnationOutcome() {}

// SpellTableRefusal is a rule reason a Reincarnate is refused. The zero value is invalid.
type SpellTableRefusal int

const (
	// NotHumanoid: the dead creature is not a Humanoid.
	NotHumanoid SpellTableRefusal = iota + 1
	// DeadTooLong: the creature has been dead more than 10 days.
	DeadTooLong
)

// String returns "not_humanoid", "dead_too_long" or "unknown".
func (r SpellTableRefusal) String() string {
	return enumName([]string{"not_humanoid", "dead_too_long"}, int(r))
}

// reincarnateWindow is the longest a creature may have been dead (SRD 5.2.1, "Reincarnate", p. 158).
const reincarnateWindow = 10 * 24 * time.Hour

// reincarnationRefusals returns every refusal, in the order NotHumanoid then DeadTooLong; exactly 10 days is
// accepted. A creature type that is not valid or a negative deadFor is an error matching srderr.ErrInvalidInput.
func reincarnationRefusals(t content.CreatureType, deadFor time.Duration) ([]SpellTableRefusal, error) {
	if !t.Valid() || deadFor < 0 {
		return nil, invalidInput("reincarnate needs a creature type and a dead time of 0 or more")
	}
	var out []SpellTableRefusal
	if t != content.CreatureTypeHumanoid {
		out = append(out, NotHumanoid)
	}
	if deadFor > reincarnateWindow {
		out = append(out, DeadTooLong)
	}
	return out, nil
}
