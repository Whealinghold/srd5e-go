package character

import "github.com/Whealinghold/srd5e-go/d20"

// The bounds of the Concentration saving-throw DC (SRD 5.2.1, "Concentration",
// p. 179).
const (
	concentrationSaveDCFloor = 10
	concentrationSaveDCCap   = 30
)

// ConcentrationSaveDC is the DC of the Constitution saving throw that keeps
// Concentration after damage (SRD 5.2.1, "Concentration", p. 179). Comparable
// and safe for concurrent use. The zero value is not the result of any valid
// input.
type ConcentrationSaveDC struct{ dc d20.DifficultyClass }

// DifficultyClass returns the DC for the d20 package.
func (c ConcentrationSaveDC) DifficultyClass() d20.DifficultyClass { return c.dc }

// Value returns the DC as an int.
func (c ConcentrationSaveDC) Value() int { return int(c.dc) }

// ConcentrationSaveDCForDamage returns the DC for damageTaken: the greater of 10
// and half the damage rounded down, to a maximum of 30. damageTaken is the
// damage after Resistance or Immunity, counting damage that Temporary Hit
// Points absorbed (a package interpretation of SRD 5.2.1, p. 18 and p. 179).
//
// It returns an error matching srderr.ErrInvalidInput for an amount below 1: no
// damage taken means no save. It records nothing and never panics.
func ConcentrationSaveDCForDamage(damageTaken int) (ConcentrationSaveDC, error) {
	if damageTaken < 1 {
		return ConcentrationSaveDC{}, invalidInput("concentration save DC: damage taken below 1")
	}
	return ConcentrationSaveDC{dc: d20.DifficultyClass(min(concentrationSaveDCCap, max(concentrationSaveDCFloor, damageTaken/2)))}, nil
}
