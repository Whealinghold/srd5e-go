package combat

import "github.com/Whealinghold/srd5e-go/content"

// WaterAmount is an amount of water in whole eighths of a gallon, 0 or more
// (SRD 5.2.1, "Dehydration [Hazard]", p. 181). It is a distinct type from
// FoodAmount. The zero value is no water. Comparable.
type WaterAmount struct{ eighths int }

// NewWaterAmount returns eighths of a gallon of water. It returns an error matching
// srderr.ErrInvalidInput for a negative amount. It never panics.
func NewWaterAmount(eighths int) (WaterAmount, error) {
	if eighths < 0 {
		return WaterAmount{}, invalidInput("water amount below 0")
	}
	return WaterAmount{eighths}, nil
}

// Eighths returns the amount in eighths of a gallon.
func (a WaterAmount) Eighths() int { return a.eighths }

// FoodAmount is an amount of food in whole eighths of a pound, 0 or more (SRD
// 5.2.1, "Malnutrition [Hazard]", p. 185). It is a distinct type from
// WaterAmount. The zero value is no food. Comparable.
type FoodAmount struct{ eighths int }

// NewFoodAmount returns eighths of a pound of food. It returns an error matching
// srderr.ErrInvalidInput for a negative amount. It never panics.
func NewFoodAmount(eighths int) (FoodAmount, error) {
	if eighths < 0 {
		return FoodAmount{}, invalidInput("food amount below 0")
	}
	return FoodAmount{eighths}, nil
}

// Eighths returns the amount in eighths of a pound.
func (a FoodAmount) Eighths() int { return a.eighths }

// needsEighths is the daily need of a creature of size in eighths: 1/4 for Tiny, 1,
// 1, 4, 16 and 64 above (the same for water in gallons and food in pounds). It
// returns 0 for a size that is not valid.
func needsEighths(size content.CreatureSize) int {
	switch size {
	case content.SizeTiny:
		return 2
	case content.SizeSmall, content.SizeMedium:
		return 8
	case content.SizeLarge:
		return 32
	case content.SizeHuge:
		return 128
	case content.SizeGargantuan:
		return 512
	}
	return 0
}

// WaterNeedsPerDay is the table of the water a creature needs each day (SRD 5.2.1,
// "Dehydration [Hazard]", p. 181). The zero value is ready to use.
type WaterNeedsPerDay struct{}

// ForSize returns the daily water need of a creature of size. It returns an error
// matching srderr.ErrInvalidInput for a size that is not valid.
func (WaterNeedsPerDay) ForSize(size content.CreatureSize) (WaterAmount, error) {
	if !size.Valid() {
		return WaterAmount{}, invalidInput("invalid creature size")
	}
	return WaterAmount{needsEighths(size)}, nil
}

// FoodNeedsPerDay is the table of the food a creature needs each day (SRD 5.2.1,
// "Malnutrition [Hazard]", p. 185). The zero value is ready to use.
type FoodNeedsPerDay struct{}

// ForSize returns the daily food need of a creature of size. It returns an error
// matching srderr.ErrInvalidInput for a size that is not valid.
func (FoodNeedsPerDay) ForSize(size content.CreatureSize) (FoodAmount, error) {
	if !size.Valid() {
		return FoodAmount{}, invalidInput("invalid creature size")
	}
	return FoodAmount{needsEighths(size)}, nil
}

// isLessThanHalfOf reports whether amount is less than half of required, in
// integer arithmetic: exactly half is not less than half. Both are in eighths.
func isLessThanHalfOf(amount, required int) bool { return amount < required-amount }
