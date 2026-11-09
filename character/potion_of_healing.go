package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// PotionOfHealingGrade is one of the four Potion of Healing grades (SRD 5.2.1,
// "Potions of Healing", p. 236). The zero value is invalid.
type PotionOfHealingGrade int

// The grades.
const (
	GradeStandard PotionOfHealingGrade = iota + 1
	GradeGreater
	GradeSuperior
	GradeSupreme
)

// Dice returns the grade's expression with its fixed modifier (2d4 + 2, 4d4 + 4,
// 8d4 + 8, 10d4 + 20), or the zero Expression for an invalid grade.
func (g PotionOfHealingGrade) Dice() dice.Expression {
	n := [...]int{2, 4, 8, 10}
	m := [...]int{2, 4, 8, 20}
	if g < GradeStandard || g > GradeSupreme {
		return dice.Expression{}
	}
	return dice.MustExpression(n[g-1], dice.D4(), m[g-1])
}

// Rarity returns the grade's rarity (Common to Very Rare), or 0 for an invalid grade.
func (g PotionOfHealingGrade) Rarity() content.ItemRarity {
	if g < GradeStandard || g > GradeSupreme {
		return 0
	}
	return content.RarityCommon + content.ItemRarity(g-1)
}

// String returns the snake_case name of the grade, or "unknown".
func (g PotionOfHealingGrade) String() string {
	names := []string{"standard", "greater", "superior", "supreme"}
	if g < 1 || int(g) > len(names) {
		return "unknown"
	}
	return names[g-1]
}

// PotionOfHealingGradeResult is the closed set of outcomes of
// PotionOfHealingGrades.ForRarity. Variants: PotionOfHealingGradeFound,
// PotionOfHealingGradeNotListed.
type PotionOfHealingGradeResult interface{ isPotionOfHealingGradeResult() }

// PotionOfHealingGradeFound: the grade of the rarity.
type PotionOfHealingGradeFound struct{ grade PotionOfHealingGrade }

// Grade returns the grade.
func (r PotionOfHealingGradeFound) Grade() PotionOfHealingGrade { return r.grade }

// PotionOfHealingGradeNotListed: no grade has the rarity.
type PotionOfHealingGradeNotListed struct{ rarity content.ItemRarity }

// Rarity returns the rarity asked for.
func (r PotionOfHealingGradeNotListed) Rarity() content.ItemRarity { return r.rarity }

func (PotionOfHealingGradeFound) isPotionOfHealingGradeResult()     {}
func (PotionOfHealingGradeNotListed) isPotionOfHealingGradeResult() {}

// PotionOfHealingGrades maps rarities to grades. The zero value is ready to use.
type PotionOfHealingGrades struct{}

// ForRarity maps Common, Uncommon, Rare and Very Rare to the four grades; any
// other rarity is PotionOfHealingGradeNotListed, never an error.
func (PotionOfHealingGrades) ForRarity(r content.ItemRarity) PotionOfHealingGradeResult {
	if g := PotionOfHealingGrade(r); r >= content.RarityCommon && g <= GradeSupreme {
		return PotionOfHealingGradeFound{grade: g}
	}
	return PotionOfHealingGradeNotListed{rarity: r}
}

// ResistanceTypeTable is the 1d10 table of damage types shared by the Potion of
// Resistance, the Ring of Resistance and Armor of Resistance (SRD 5.2.1, "Potion
// of Resistance", p. 236). The zero value is ready to use.
type ResistanceTypeTable struct{}

// ForRoll maps 1 to 10 to Acid, Cold, Fire, Force, Lightning, Necrotic, Poison,
// Psychic, Radiant, Thunder. It returns a zero DamageType and an error matching
// srderr.ErrInvalidInput for any other value.
func (ResistanceTypeTable) ForRoll(d10 int) (content.DamageType, error) {
	types := resistanceTypes()
	if d10 < 1 || d10 > len(types) {
		return 0, invalidInput("resistance type roll outside 1 to 10")
	}
	return types[d10-1], nil
}

func resistanceTypes() []content.DamageType {
	return []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageForce, content.DamageLightning,
		content.DamageNecrotic, content.DamagePoison, content.DamagePsychic, content.DamageRadiant, content.DamageThunder}
}

// ResistancePotionDuration is how long a Potion of Resistance lasts. A ring or
// armor lasts while worn, which GrantedResistance carries as a zero duration.
const ResistancePotionDuration = 60 * time.Minute

// GrantedResistanceRejection says why a chosen resistance was refused. The zero
// value is not a reason.
type GrantedResistanceRejection int

// The reasons.
const (
	ResistanceTypeNotListed GrantedResistanceRejection = iota + 1
	ResistanceDurationNegative
)

// String returns "type_not_listed", "duration_negative" or "unknown".
func (r GrantedResistanceRejection) String() string {
	names := []string{"type_not_listed", "duration_negative"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// GrantedResistanceResult is the closed set of outcomes of
// GrantedResistances.Chosen. Variants: GrantedResistance, GrantedResistanceRejected.
type GrantedResistanceResult interface{ isGrantedResistanceResult() }

// GrantedResistance is Resistance to one damage type for a duration; the host
// applies it. A zero duration means "while wearing".
type GrantedResistance struct {
	damage   content.DamageType
	duration time.Duration
}

// DamageType returns the type resisted.
func (g GrantedResistance) DamageType() content.DamageType { return g.damage }

// Duration returns the duration, or 0 for "while wearing".
func (g GrantedResistance) Duration() time.Duration { return g.duration }

// GrantedResistanceRejected: the choice was refused.
type GrantedResistanceRejected struct{ reason GrantedResistanceRejection }

// Reason returns why.
func (r GrantedResistanceRejected) Reason() GrantedResistanceRejection { return r.reason }

func (GrantedResistance) isGrantedResistanceResult()         {}
func (GrantedResistanceRejected) isGrantedResistanceResult() {}

// GrantedResistances builds the Resistance a GM chooses. The zero value is ready to use.
type GrantedResistances struct{}

// Chosen accepts only a type of the table's ten ("the GM chooses the type or
// determines it randomly"); Bludgeoning, Piercing and Slashing are
// ResistanceTypeNotListed and a negative duration is ResistanceDurationNegative.
func (GrantedResistances) Chosen(t content.DamageType, d time.Duration) GrantedResistanceResult {
	switch {
	case !slices.Contains(resistanceTypes(), t):
		return GrantedResistanceRejected{reason: ResistanceTypeNotListed}
	case d < 0:
		return GrantedResistanceRejected{reason: ResistanceDurationNegative}
	}
	return GrantedResistance{damage: t, duration: d}
}
