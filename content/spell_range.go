package content

// MinSpellRangeDistance is the least distance of a spell range (content-spells.md
// FR-037, FR-111).
const MinSpellRangeDistance = 1

// ValidSpellRangeDistance reports whether n is a distance a spell range may
// have, at least [MinSpellRangeDistance].
func ValidSpellRangeDistance(n int) bool { return n >= MinSpellRangeDistance }

// SpellRange is the closed set of spell ranges of SRD 5.2.1 ("Range", p. 105):
// [SpellRangeSelf], [SpellRangeTouch], [SpellRangeDistance], [SpellRangeSight],
// [SpellRangeUnlimited] and [SpellRangeSpecial]. Every variant is an immutable,
// comparable value and safe for concurrent use. Consumers switch over it
// exhaustively.
type SpellRange interface{ isSpellRange() }

// SpellRangeSelf is a range of Self. The zero value is valid.
type SpellRangeSelf struct{}

// NewSpellRangeSelf returns the Self range.
func NewSpellRangeSelf() SpellRangeSelf { return SpellRangeSelf{} }

func (SpellRangeSelf) isSpellRange() {}

// SpellRangeTouch is a range of Touch. The zero value is valid.
type SpellRangeTouch struct{}

// NewSpellRangeTouch returns the Touch range.
func NewSpellRangeTouch() SpellRangeTouch { return SpellRangeTouch{} }

func (SpellRangeTouch) isSpellRange() {}

// SpellRangeDistance is a range of an amount of feet or miles. The zero value
// is invalid; build it with [NewSpellRangeDistance].
type SpellRangeDistance struct {
	amount int
	unit   DistanceUnit
}

// NewSpellRangeDistance returns the range of amount units.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an amount below [MinSpellRangeDistance] and
// BlockInvalidMember for an invalid unit. It never panics.
func NewSpellRangeDistance(amount int, unit DistanceUnit) (SpellRangeDistance, error) {
	if !ValidSpellRangeDistance(amount) {
		return SpellRangeDistance{}, blockError("SpellRangeDistance", BlockOutOfRange)
	}
	if !unit.Valid() {
		return SpellRangeDistance{}, blockError("SpellRangeDistance", BlockInvalidMember)
	}
	return SpellRangeDistance{amount: amount, unit: unit}, nil
}

// Amount returns the number of units.
func (r SpellRangeDistance) Amount() int { return r.amount }

// Unit returns the unit.
func (r SpellRangeDistance) Unit() DistanceUnit { return r.unit }

func (SpellRangeDistance) isSpellRange() {}

// SpellRangeSight is a range of Sight. The zero value is valid.
type SpellRangeSight struct{}

// NewSpellRangeSight returns the Sight range.
func NewSpellRangeSight() SpellRangeSight { return SpellRangeSight{} }

func (SpellRangeSight) isSpellRange() {}

// SpellRangeUnlimited is a range of Unlimited. The zero value is valid.
type SpellRangeUnlimited struct{}

// NewSpellRangeUnlimited returns the Unlimited range.
func NewSpellRangeUnlimited() SpellRangeUnlimited { return SpellRangeUnlimited{} }

func (SpellRangeUnlimited) isSpellRange() {}

// SpellRangeSpecial is a range the description explains (SRD 5.2.1, "Dream",
// p. 126). The zero value is valid.
type SpellRangeSpecial struct{}

// NewSpellRangeSpecial returns the Special range.
func NewSpellRangeSpecial() SpellRangeSpecial { return SpellRangeSpecial{} }

func (SpellRangeSpecial) isSpellRange() {}

// checkSpellRange reports why r is not a range its constructor would accept:
// BlockMissing for nil and for a pointer variant.
func checkSpellRange(r SpellRange) error {
	switch v := r.(type) {
	case SpellRangeSelf, SpellRangeTouch, SpellRangeSight, SpellRangeUnlimited, SpellRangeSpecial:
		return nil
	case SpellRangeDistance:
		_, err := NewSpellRangeDistance(v.amount, v.unit)
		return err
	}
	return blockError("SpellRange", BlockMissing)
}
