package content

// CountScaling is the closed set of ways a count of targets or instances
// grows: [NoCountScaling], [PerSlotLevelCount] and [ByCharacterLevelCount].
// Every variant is an immutable, comparable value and safe for concurrent use.
// Consumers switch over it exhaustively.
type CountScaling interface{ isCountScaling() }

// NoCountScaling is a count that does not grow. The zero value is valid.
type NoCountScaling struct{}

func (NoCountScaling) isCountScaling() {}

// PerSlotLevelCount is a count that grows by Extra for each slot level above
// AboveLevel (SRD 5.2.1, "Hold Person", p. 141). The zero value is invalid;
// build it with [NewPerSlotLevelCount].
type PerSlotLevelCount struct{ aboveLevel, extra int }

// NewPerSlotLevelCount returns the scaling of extra for each level above
// aboveLevel. Whether aboveLevel suits a spell is the spell's check
// ([SpellEffectModel.CheckSpellLevel]).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for extra below 1. It never panics.
func NewPerSlotLevelCount(aboveLevel, extra int) (PerSlotLevelCount, error) {
	if extra < 1 {
		return PerSlotLevelCount{}, blockError("PerSlotLevelCount", BlockOutOfRange)
	}
	return PerSlotLevelCount{aboveLevel: aboveLevel, extra: extra}, nil
}

// AboveLevel returns the slot level above which the count grows.
func (s PerSlotLevelCount) AboveLevel() int { return s.aboveLevel }

// Extra returns the growth for each slot level.
func (s PerSlotLevelCount) Extra() int { return s.extra }

func (PerSlotLevelCount) isCountScaling() {}

// ByCharacterLevelCount is a count chosen by total character level (SRD
// 5.2.1, "Eldritch Blast", p. 127). The zero value is invalid; build it with
// [NewByCharacterLevelCount].
type ByCharacterLevelCount struct{ table CountLevelTable }

// NewByCharacterLevelCount returns the scaling by the table.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// for the zero table. It never panics.
func NewByCharacterLevelCount(table CountLevelTable) (ByCharacterLevelCount, error) {
	if table.t.n == 0 {
		return ByCharacterLevelCount{}, blockError("ByCharacterLevelCount", BlockEmpty)
	}
	return ByCharacterLevelCount{table: table}, nil
}

// Table returns the table of counts by level.
func (s ByCharacterLevelCount) Table() CountLevelTable { return s.table }

func (ByCharacterLevelCount) isCountScaling() {}

// checkCountScaling reports why s is not a scaling its constructor would
// accept; a nil scaling is the valid [NoCountScaling].
func checkCountScaling(s CountScaling) error {
	switch v := s.(type) {
	case nil, NoCountScaling:
		return nil
	case PerSlotLevelCount:
		_, err := NewPerSlotLevelCount(v.aboveLevel, v.extra)
		return err
	case ByCharacterLevelCount:
		_, err := NewByCharacterLevelCount(v.table)
		return err
	}
	return blockError("CountScaling", BlockInvalidMember)
}

// scaledCount returns base grown by s for a cast, with the inputs and errors
// of [ScalableAmount.At].
func scaledCount(block string, base int, s CountScaling, slotLevelsAbove, characterLevel int) (int, error) {
	switch v := s.(type) {
	case PerSlotLevelCount:
		if slotLevelsAbove < 0 {
			return 0, blockError(block, BlockOutOfRange)
		}
		if c, ok := mulAdd(base, slotLevelsAbove, v.extra); ok {
			return c, nil
		}
		return 0, blockError(block, BlockOutOfRange)
	case ByCharacterLevelCount:
		return v.table.ValueAt(characterLevel)
	}
	if base < 1 {
		return 0, blockError(block, BlockMissing)
	}
	return base, nil
}

// TargetCount is the closed set of target counts of a chosen-target spell:
// [ExactlyTargets], [UpToTargets] and [TargetsByInstance]. Consumers switch
// over it exhaustively.
type TargetCount interface{ isTargetCount() }

// ExactlyTargets is a spell with exactly N targets (SRD 5.2.1, "Targets",
// p. 106), growing as its scaling says. The zero value is invalid; build it
// with [NewExactlyTargets]. Comparable and safe for concurrent use.
type ExactlyTargets struct {
	n       int
	scaling CountScaling
}

// NewExactlyTargets returns exactly n targets, n being at least 1, with
// scaling (nil is [NoCountScaling]).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for n below 1 and the reason of the scaling's own check for
// an invalid scaling. It never panics.
func NewExactlyTargets(n int, scaling CountScaling) (ExactlyTargets, error) {
	s, err := checkedTargetCount("ExactlyTargets", n, scaling)
	if err != nil {
		return ExactlyTargets{}, err
	}
	return ExactlyTargets{n: n, scaling: s}, nil
}

// Base returns the count before scaling.
func (t ExactlyTargets) Base() int { return t.n }

// Scaling returns how the count grows.
func (t ExactlyTargets) Scaling() CountScaling { return t.scaling }

// At returns the number of targets for a cast, with the inputs of
// [ScalableAmount.At].
func (t ExactlyTargets) At(slotLevelsAbove, characterLevel int) (int, error) {
	return scaledCount("ExactlyTargets", t.n, t.scaling, slotLevelsAbove, characterLevel)
}

func (ExactlyTargets) isTargetCount() {}

// UpToTargets is a spell with up to N targets (SRD 5.2.1, "Targets", p. 106),
// growing as its scaling says. The zero value is invalid; build it with
// [NewUpToTargets]. Comparable and safe for concurrent use.
type UpToTargets struct {
	n       int
	scaling CountScaling
}

// NewUpToTargets returns up to n targets, n being at least 1, with scaling
// (nil is [NoCountScaling]).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for n below 1 and the reason of the scaling's own check for
// an invalid scaling. It never panics.
func NewUpToTargets(n int, scaling CountScaling) (UpToTargets, error) {
	s, err := checkedTargetCount("UpToTargets", n, scaling)
	if err != nil {
		return UpToTargets{}, err
	}
	return UpToTargets{n: n, scaling: s}, nil
}

// Base returns the count before scaling.
func (t UpToTargets) Base() int { return t.n }

// Scaling returns how the count grows.
func (t UpToTargets) Scaling() CountScaling { return t.scaling }

// At returns the largest number of targets for a cast, with the inputs of
// [ScalableAmount.At].
func (t UpToTargets) At(slotLevelsAbove, characterLevel int) (int, error) {
	return scaledCount("UpToTargets", t.n, t.scaling, slotLevelsAbove, characterLevel)
}

func (UpToTargets) isTargetCount() {}

// TargetsByInstance means the targets are those the part's instances are
// assigned to (SRD 5.2.1, "Magic Missile", p. 146). The zero value is valid.
type TargetsByInstance struct{}

func (TargetsByInstance) isTargetCount() {}

// checkedTargetCount applies FR-012 and returns the scaling with nil replaced
// by [NoCountScaling]; on error the returned scaling is nil.
func checkedTargetCount(block string, n int, scaling CountScaling) (CountScaling, error) {
	if n < 1 {
		return nil, blockError(block, BlockOutOfRange)
	}
	if err := checkCountScaling(scaling); err != nil {
		return nil, err
	}
	if scaling == nil {
		return NoCountScaling{}, nil
	}
	return scaling, nil
}

// checkTargetCount reports why c is not a count its constructor would accept:
// BlockMissing for nil.
func checkTargetCount(c TargetCount) error {
	switch v := c.(type) {
	case ExactlyTargets:
		_, err := NewExactlyTargets(v.n, v.scaling)
		return err
	case UpToTargets:
		_, err := NewUpToTargets(v.n, v.scaling)
		return err
	case TargetsByInstance:
		return nil
	}
	return blockError("TargetCount", BlockMissing)
}

// PartInstances is how many times a part's resolution is made, as with the
// darts of a Magic Missile or the beams of an Eldritch Blast (SRD 5.2.1,
// p. 127): a base of at least 1 and an optional scaling. The zero value is
// invalid; build it with [NewPartInstances]. Comparable and safe for
// concurrent use.
type PartInstances struct {
	base    int
	scaling CountScaling
}

// NewPartInstances returns the instances base with scaling (nil is
// [NoCountScaling]).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a base below 1 and the reason of the scaling's own check
// for an invalid scaling. It never panics.
func NewPartInstances(base int, scaling CountScaling) (PartInstances, error) {
	s, err := checkedTargetCount("PartInstances", base, scaling)
	if err != nil {
		return PartInstances{}, err
	}
	return PartInstances{base: base, scaling: s}, nil
}

// Base returns the count before scaling.
func (p PartInstances) Base() int { return p.base }

// Scaling returns how the count grows.
func (p PartInstances) Scaling() CountScaling { return p.scaling }

// At returns the number of instances for a cast, with the inputs of
// [ScalableAmount.At].
func (p PartInstances) At(slotLevelsAbove, characterLevel int) (int, error) {
	return scaledCount("PartInstances", p.base, p.scaling, slotLevelsAbove, characterLevel)
}
