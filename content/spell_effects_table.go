package content

// spellMaxCharacterLevel is the highest total character level a scaling query
// accepts (SRD 5.2.1, "Cantrips", p. 25; spell-effect-data.md FR-041).
const spellMaxCharacterLevel = 20

// levelTable is the one unexported generic helper behind AmountLevelTable and
// CountLevelTable (spell-effect-data.md section 5.2, go.md section 6): up to
// spellMaxCharacterLevel entries in two arrays, so the type is comparable and
// copying it copies the data. The zero value has no entry and is invalid.
type levelTable[V comparable] struct {
	n      int
	levels [spellMaxCharacterLevel]int
	values [spellMaxCharacterLevel]V
}

// newLevelTable applies FR-019: at least one entry (BlockEmpty), every level
// in 1 to 20 (BlockOutOfRange), the first level 1 and the levels strictly
// ascending (BlockConstraint). Strictly ascending levels in 1 to 20 are at
// most 20 entries, so the arrays always fit.
func newLevelTable[V comparable](block string, levels []int, values []V) (levelTable[V], error) {
	if len(levels) == 0 {
		return levelTable[V]{}, blockError(block, BlockEmpty)
	}
	for _, l := range levels {
		if l < 1 || l > spellMaxCharacterLevel {
			return levelTable[V]{}, blockError(block, BlockOutOfRange)
		}
	}
	ascending := levels[0] == 1
	for i := 1; i < len(levels); i++ {
		ascending = ascending && levels[i] > levels[i-1]
	}
	if !ascending {
		return levelTable[V]{}, blockError(block, BlockConstraint)
	}
	t := levelTable[V]{n: len(levels)}
	copy(t.levels[:], levels)
	copy(t.values[:], values)
	return t, nil
}

// at returns the value of the entry with the greatest level not above level
// (FR-018). The table must have an entry and level must be at least 1.
func (t levelTable[V]) at(level int) V {
	i := 0
	for j := 1; j < t.n && t.levels[j] <= level; j++ {
		i = j
	}
	return t.values[i]
}

// checkCharacterLevel is the range rule of FR-041 for a total character level.
func checkCharacterLevel(block string, level int) error {
	if level < 1 || level > spellMaxCharacterLevel {
		return blockError(block, BlockOutOfRange)
	}
	return nil
}

// AmountLevelEntry is one row of an [AmountLevelTable]: the amount that
// replaces the base from Level. Build it with a keyed literal.
type AmountLevelEntry struct {
	Level int
	Value SpellAmount
}

// AmountLevelTable is a Cantrip Upgrade table of amounts (SRD 5.2.1, "Fire
// Bolt", p. 132): the value of the entry with the greatest level not above the
// total character level. It holds the printed totals (1d10, 2d10, ...), not
// increments. The zero AmountLevelTable is invalid; build it with
// [NewAmountLevelTable]. Comparable and safe for concurrent use.
type AmountLevelTable struct{ t levelTable[SpellAmount] }

// NewAmountLevelTable returns the table of entries, copying them.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// for no entry, BlockOutOfRange for a level outside 1 to 20, BlockConstraint
// for a first level other than 1 or levels that do not strictly ascend, and
// the reason of the amount's own check (BlockMissing for a nil amount,
// BlockOutOfRange for an invalid one). It never panics.
func NewAmountLevelTable(entries []AmountLevelEntry) (AmountLevelTable, error) {
	levels := make([]int, len(entries))
	values := make([]SpellAmount, len(entries))
	for i, e := range entries {
		levels[i], values[i] = e.Level, e.Value
	}
	t, err := newLevelTable("AmountLevelTable", levels, values)
	if err != nil {
		return AmountLevelTable{}, err
	}
	for _, v := range values {
		if err := checkSpellAmount(v); err != nil {
			return AmountLevelTable{}, err
		}
	}
	return AmountLevelTable{t: t}, nil
}

// ValueAt returns the amount for a total character level (FR-018, FR-041).
//
// It returns a *[BlockError] with BlockOutOfRange for a level outside 1 to 20
// and with BlockMissing for the zero table. It never panics.
func (t AmountLevelTable) ValueAt(level int) (SpellAmount, error) {
	if err := checkCharacterLevel("AmountLevelTable", level); err != nil {
		return nil, err
	}
	if t.t.n == 0 {
		return nil, blockError("AmountLevelTable", BlockMissing)
	}
	return t.t.at(level), nil
}

// Entries returns a copy of the entries in ascending level order.
func (t AmountLevelTable) Entries() []AmountLevelEntry {
	out := make([]AmountLevelEntry, t.t.n)
	for i := range out {
		out[i] = AmountLevelEntry{Level: t.t.levels[i], Value: t.t.values[i]}
	}
	return out
}

// CountLevelEntry is one row of a [CountLevelTable]: the count that replaces
// the base from Level. Build it with a keyed literal.
type CountLevelEntry struct {
	Level int
	Count int
}

// CountLevelTable is a Cantrip Upgrade table of counts (SRD 5.2.1, "Eldritch
// Blast", p. 127): the count of the entry with the greatest level not above
// the total character level. It holds the printed counts, not increments. The
// zero CountLevelTable is invalid; build it with [NewCountLevelTable].
// Comparable and safe for concurrent use.
type CountLevelTable struct{ t levelTable[int] }

// NewCountLevelTable returns the table of entries, copying them.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput under the rules
// of [NewAmountLevelTable], and with BlockOutOfRange for a count below 1. It
// never panics.
func NewCountLevelTable(entries []CountLevelEntry) (CountLevelTable, error) {
	levels := make([]int, len(entries))
	values := make([]int, len(entries))
	for i, e := range entries {
		levels[i], values[i] = e.Level, e.Count
	}
	t, err := newLevelTable("CountLevelTable", levels, values)
	if err != nil {
		return CountLevelTable{}, err
	}
	for _, v := range values {
		if v < 1 {
			return CountLevelTable{}, blockError("CountLevelTable", BlockOutOfRange)
		}
	}
	return CountLevelTable{t: t}, nil
}

// ValueAt returns the count for a total character level (FR-018, FR-041).
//
// It returns a *[BlockError] with BlockOutOfRange for a level outside 1 to 20
// and with BlockMissing for the zero table. It never panics.
func (t CountLevelTable) ValueAt(level int) (int, error) {
	if err := checkCharacterLevel("CountLevelTable", level); err != nil {
		return 0, err
	}
	if t.t.n == 0 {
		return 0, blockError("CountLevelTable", BlockMissing)
	}
	return t.t.at(level), nil
}

// Entries returns a copy of the entries in ascending level order.
func (t CountLevelTable) Entries() []CountLevelEntry {
	out := make([]CountLevelEntry, t.t.n)
	for i := range out {
		out[i] = CountLevelEntry{Level: t.t.levels[i], Count: t.t.values[i]}
	}
	return out
}
