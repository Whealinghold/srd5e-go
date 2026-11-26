package combat

import "github.com/Whealinghold/srd5e-go/srderr"

// maxHazardLevels is the most Exhaustion levels a record can attribute: the
// condition's maximum (SRD 5.2.1, "Exhaustion", p. 181).
const maxHazardLevels = maxExhaustionLevel

// HazardRecord is the ledger of the Exhaustion levels a creature holds from each
// source, kept by the host beside its ConditionSet: Dehydration and Malnutrition
// levels (and levels an effect holds until it ends, the contagion levels) cannot
// be removed until their cause is gone, Suffocation levels can. It also counts the
// consecutive days without food. The four level counts sum to at most 6. It is
// comparable and immutable, and the zero value is the empty record (see
// EmptyHazardRecord). Safe for concurrent use.
type HazardRecord struct{ dehydration, malnutrition, suffocation, daysWithoutFood, contagion int }

// HazardRecordRejection says why NewHazardRecord or WithContagionLevels rejected
// its arguments. The zero value is not a rejection.
type HazardRecordRejection int

// The rejections.
const (
	// HazardRecordNegativeCount: a level count or the days without food is below 0.
	HazardRecordNegativeCount HazardRecordRejection = iota + 1
	// HazardRecordLevelSumAboveSix: the four level counts sum to more than 6.
	HazardRecordLevelSumAboveSix
)

// String returns the snake_case name of r, or "unknown".
func (r HazardRecordRejection) String() string {
	return enumName([]string{"negative_count", "level_sum_above_six"}, int(r))
}

// HazardRecordError reports a rejected HazardRecord. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type HazardRecordError struct{ Reason HazardRecordRejection }

// Error renders the error. The text is not a contract.
func (e *HazardRecordError) Error() string {
	return "combat: invalid hazard record: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *HazardRecordError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *HazardRecordError with the same Reason.
func (e *HazardRecordError) Is(target error) bool {
	t, ok := target.(*HazardRecordError)
	return ok && t != nil && e != nil && *t == *e
}

// EmptyHazardRecord returns the record with every count 0.
func EmptyHazardRecord() HazardRecord { return HazardRecord{} }

// NewHazardRecord returns the record of the given counts, with no contagion
// levels (see WithContagionLevels). A host restoring a persisted ledger uses it.
//
// It returns a *HazardRecordError (matching srderr.ErrInvalidInput) with
// HazardRecordNegativeCount for any negative count and HazardRecordLevelSumAboveSix
// when the three level counts sum to more than 6. It never panics.
func NewHazardRecord(dehydrationLevels, malnutritionLevels, suffocationLevels, daysWithoutFood int) (HazardRecord, error) {
	r := HazardRecord{dehydration: dehydrationLevels, malnutrition: malnutritionLevels,
		suffocation: suffocationLevels, daysWithoutFood: daysWithoutFood}
	if err := r.check(); err != nil {
		return HazardRecord{}, err
	}
	return r, nil
}

// MustHazardRecord is NewHazardRecord for values known to be valid. It panics
// with the error NewHazardRecord returns, and only then.
func MustHazardRecord(dehydrationLevels, malnutritionLevels, suffocationLevels, daysWithoutFood int) HazardRecord {
	r, err := NewHazardRecord(dehydrationLevels, malnutritionLevels, suffocationLevels, daysWithoutFood)
	if err != nil {
		panic(err)
	}
	return r
}

// check reports the rejection of r, nil when it is valid. The sum is compared one
// term at a time so no int overflows.
func (r HazardRecord) check() error {
	counts := []int{r.dehydration, r.malnutrition, r.suffocation, r.daysWithoutFood, r.contagion}
	for _, c := range counts {
		if c < 0 {
			return &HazardRecordError{HazardRecordNegativeCount}
		}
	}
	sum := 0
	for _, c := range []int{r.dehydration, r.malnutrition, r.suffocation, r.contagion} {
		if c > maxHazardLevels-sum {
			return &HazardRecordError{HazardRecordLevelSumAboveSix}
		}
		sum += c
	}
	return nil
}

// WithContagionLevels returns r with count contagion levels: Exhaustion levels an
// effect holds until it ends, which cannot be removed meanwhile (SRD 5.2.1,
// "Cackle Fever", p. 194). No use case raises it; the host sets it when the effect
// applies and lowers it when the effect ends.
//
// It returns a *HazardRecordError (matching srderr.ErrInvalidInput) for a negative
// count or a sum of the four level counts above 6. It never panics.
func (r HazardRecord) WithContagionLevels(count int) (HazardRecord, error) {
	r.contagion = count
	if err := r.check(); err != nil {
		return HazardRecord{}, err
	}
	return r, nil
}

// DehydrationLevels returns the Exhaustion levels attributed to Dehydration.
func (r HazardRecord) DehydrationLevels() int { return r.dehydration }

// MalnutritionLevels returns the Exhaustion levels attributed to Malnutrition.
func (r HazardRecord) MalnutritionLevels() int { return r.malnutrition }

// SuffocationLevels returns the Exhaustion levels attributed to Suffocation.
func (r HazardRecord) SuffocationLevels() int { return r.suffocation }

// DaysWithoutFood returns the consecutive days the creature has eaten nothing.
func (r HazardRecord) DaysWithoutFood() int { return r.daysWithoutFood }

// ContagionLevels returns the Exhaustion levels an effect holds until it ends.
func (r HazardRecord) ContagionLevels() int { return r.contagion }

// RemovableLevels returns how many of currentLevel Exhaustion levels can be
// removed: the level less the Dehydration, Malnutrition and contagion levels,
// never below 0. A level below 0 counts as 0.
func (r HazardRecord) RemovableLevels(currentLevel int) int {
	locked := satAdd(satAdd(r.dehydration, r.malnutrition), r.contagion)
	return max(0, max(0, currentLevel)-locked)
}

// clamped returns r with the level counts cut down so their sum does not exceed
// level, reducing Suffocation first, then Malnutrition, Dehydration and contagion.
func (r HazardRecord) clamped(level int) HazardRecord {
	over := r.dehydration + r.malnutrition + r.suffocation + r.contagion - max(0, level)
	for _, c := range []*int{&r.suffocation, &r.malnutrition, &r.dehydration, &r.contagion} {
		cut := min(over, *c)
		if cut > 0 {
			*c -= cut
			over -= cut
		}
	}
	return r
}
