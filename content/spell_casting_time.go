package content

import "slices"

// The limits of the casting time types (content-spells.md FR-029, FR-030,
// FR-111).
const (
	// MinCastingTimeAmount is the least number of minutes or hours.
	MinCastingTimeAmount = 1
	// MinCastingTimeModes is the least number of labelled casting time modes.
	MinCastingTimeModes = 2
)

// ValidCastingTimeAmount reports whether n is a number of minutes or hours a
// casting time may have, at least [MinCastingTimeAmount].
func ValidCastingTimeAmount(n int) bool { return n >= MinCastingTimeAmount }

// CastingTime is the closed set of casting times of SRD 5.2.1 ("Casting
// Time", p. 105): [CastingTimeMagicAction], [CastingTimeBonusAction],
// [CastingTimeReaction], [CastingTimeMinutes] and [CastingTimeHours]. Every
// variant is an immutable, comparable value and safe for concurrent use.
// Consumers switch over it exhaustively.
type CastingTime interface{ isCastingTime() }

// CastingTimeMagicAction is a casting time of one Magic action (printed
// "Action"). The zero value is valid.
type CastingTimeMagicAction struct{}

// NewCastingTimeMagicAction returns the Magic action casting time.
func NewCastingTimeMagicAction() CastingTimeMagicAction { return CastingTimeMagicAction{} }

func (CastingTimeMagicAction) isCastingTime() {}

// CastingTimeBonusAction is a casting time of one Bonus Action, with the
// optional trigger text printed after "which you take". The zero value is a
// Bonus Action with no trigger and is valid.
type CastingTimeBonusAction struct{ trigger string }

// NewCastingTimeBonusAction returns the Bonus Action casting time with no
// trigger.
func NewCastingTimeBonusAction() CastingTimeBonusAction { return CastingTimeBonusAction{} }

// NewCastingTimeBonusActionTriggered returns the Bonus Action casting time
// with the verbatim trigger text.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText for a blank trigger (an empty one included) and
// BlockInvalidCharacter for an unclean one. It never panics.
func NewCastingTimeBonusActionTriggered(trigger string) (CastingTimeBonusAction, error) {
	if err := checkText("CastingTimeBonusAction", trigger); err != nil {
		return CastingTimeBonusAction{}, err
	}
	return CastingTimeBonusAction{trigger: trigger}, nil
}

// Trigger returns the trigger text, and false when there is none.
func (c CastingTimeBonusAction) Trigger() (string, bool) { return c.trigger, c.trigger != "" }

func (CastingTimeBonusAction) isCastingTime() {}

// CastingTimeReaction is a casting time of one Reaction with its trigger text.
// The zero value is invalid; build it with [NewCastingTimeReaction].
type CastingTimeReaction struct{ trigger string }

// NewCastingTimeReaction returns the Reaction casting time with the verbatim
// trigger text.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText for a blank trigger (an empty one included) and
// BlockInvalidCharacter for an unclean one. It never panics.
func NewCastingTimeReaction(trigger string) (CastingTimeReaction, error) {
	if err := checkText("CastingTimeReaction", trigger); err != nil {
		return CastingTimeReaction{}, err
	}
	return CastingTimeReaction{trigger: trigger}, nil
}

// Trigger returns the trigger text.
func (c CastingTimeReaction) Trigger() string { return c.trigger }

func (CastingTimeReaction) isCastingTime() {}

// CastingTimeMinutes is a casting time of a number of minutes. The zero value
// is invalid; build it with [NewCastingTimeMinutes].
type CastingTimeMinutes struct{ amount int }

// NewCastingTimeMinutes returns a casting time of n minutes.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for n below [MinCastingTimeAmount]. It never panics.
func NewCastingTimeMinutes(n int) (CastingTimeMinutes, error) {
	if !ValidCastingTimeAmount(n) {
		return CastingTimeMinutes{}, blockError("CastingTimeMinutes", BlockOutOfRange)
	}
	return CastingTimeMinutes{amount: n}, nil
}

// Amount returns the number of minutes.
func (c CastingTimeMinutes) Amount() int { return c.amount }

func (CastingTimeMinutes) isCastingTime() {}

// CastingTimeHours is a casting time of a number of hours. The zero value is
// invalid; build it with [NewCastingTimeHours].
type CastingTimeHours struct{ amount int }

// NewCastingTimeHours returns a casting time of n hours.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for n below [MinCastingTimeAmount]. It never panics.
func NewCastingTimeHours(n int) (CastingTimeHours, error) {
	if !ValidCastingTimeAmount(n) {
		return CastingTimeHours{}, blockError("CastingTimeHours", BlockOutOfRange)
	}
	return CastingTimeHours{amount: n}, nil
}

// Amount returns the number of hours.
func (c CastingTimeHours) Amount() int { return c.amount }

func (CastingTimeHours) isCastingTime() {}

// checkCastingTime reports why t is not a casting time its constructor would
// accept: BlockMissing for nil and for a pointer variant.
func checkCastingTime(t CastingTime) error {
	var err error
	switch v := t.(type) {
	case CastingTimeMagicAction:
	case CastingTimeBonusAction:
		if v.trigger != "" {
			_, err = NewCastingTimeBonusActionTriggered(v.trigger)
		}
	case CastingTimeReaction:
		_, err = NewCastingTimeReaction(v.trigger)
	case CastingTimeMinutes:
		_, err = NewCastingTimeMinutes(v.amount)
	case CastingTimeHours:
		_, err = NewCastingTimeHours(v.amount)
	default:
		err = blockError("CastingTime", BlockMissing)
	}
	return err
}

// CastingTimeMode is one labelled way to cast a spell that has several (Plant
// Growth, SRD 5.2.1, p. 153). The zero value is invalid; build it with
// [NewCastingTimeMode]. Comparable.
type CastingTimeMode struct {
	label string
	time  CastingTime
}

// NewCastingTimeMode returns the mode called label that takes time.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName for a label that breaks the name rule (schema FR-027) and
// the reason of the time's own check for an invalid one. It never panics.
func NewCastingTimeMode(label string, time CastingTime) (CastingTimeMode, error) {
	if err := checkName("CastingTimeMode", label); err != nil {
		return CastingTimeMode{}, err
	}
	if err := checkCastingTime(time); err != nil {
		return CastingTimeMode{}, err
	}
	return CastingTimeMode{label: label, time: time}, nil
}

// Label returns the label.
func (m CastingTimeMode) Label() string { return m.label }

// Time returns the casting time of the mode.
func (m CastingTimeMode) Time() CastingTime { return m.time }

// SpellCastingTimes is the closed set of the casting time options of a spell:
// [SpellCastingTimesSingle], one time with no label, and
// [SpellCastingTimesModes], at least two labelled modes. A label on a lone
// option and a spell with no option are unrepresentable. Consumers switch over
// it exhaustively. Variants may hold slices; compare with
// [SpellCastingTimesEqual].
type SpellCastingTimes interface{ isSpellCastingTimes() }

// SpellCastingTimesSingle is a spell with one casting time. The zero value is
// invalid; build it with [NewSpellCastingTimesSingle].
type SpellCastingTimesSingle struct{ time CastingTime }

// NewSpellCastingTimesSingle returns the single casting time t.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the time's own check (BlockMissing for nil). It never panics.
func NewSpellCastingTimesSingle(t CastingTime) (SpellCastingTimesSingle, error) {
	if err := checkCastingTime(t); err != nil {
		return SpellCastingTimesSingle{}, err
	}
	return SpellCastingTimesSingle{time: t}, nil
}

// Time returns the casting time.
func (c SpellCastingTimesSingle) Time() CastingTime { return c.time }

func (SpellCastingTimesSingle) isSpellCastingTimes() {}

// SpellCastingTimesModes is a spell with several labelled casting times. The
// zero value is invalid; build it with [NewSpellCastingTimesModes].
type SpellCastingTimesModes struct{ modes []CastingTimeMode }

// NewSpellCastingTimesModes returns the modes, copying them.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for fewer than [MinCastingTimeModes] modes and the reason of
// a mode's own check for an invalid one. It never panics.
func NewSpellCastingTimesModes(modes []CastingTimeMode) (SpellCastingTimesModes, error) {
	if len(modes) < MinCastingTimeModes {
		return SpellCastingTimesModes{}, blockError("SpellCastingTimesModes", BlockOutOfRange)
	}
	for _, m := range modes {
		if _, err := NewCastingTimeMode(m.label, m.time); err != nil {
			return SpellCastingTimesModes{}, err
		}
	}
	return SpellCastingTimesModes{modes: slices.Clone(modes)}, nil
}

// Modes returns a copy of the modes in printed order.
func (c SpellCastingTimesModes) Modes() []CastingTimeMode { return slices.Clone(c.modes) }

func (SpellCastingTimesModes) isSpellCastingTimes() {}

// SpellCastingTimesEqual reports whether a and b are the same variant with
// equal members.
func SpellCastingTimesEqual(a, b SpellCastingTimes) bool {
	switch x := a.(type) {
	case SpellCastingTimesSingle:
		y, ok := b.(SpellCastingTimesSingle)
		return ok && x == y
	case SpellCastingTimesModes:
		y, ok := b.(SpellCastingTimesModes)
		return ok && slices.Equal(x.modes, y.modes)
	}
	return a == nil && b == nil
}

// checkSpellCastingTimes reports why c is not a value its constructor would
// accept: BlockMissing for nil and for a pointer variant.
func checkSpellCastingTimes(c SpellCastingTimes) error {
	var err error
	switch v := c.(type) {
	case SpellCastingTimesSingle:
		_, err = NewSpellCastingTimesSingle(v.time)
	case SpellCastingTimesModes:
		_, err = NewSpellCastingTimesModes(v.modes)
	default:
		err = blockError("SpellCastingTimes", BlockMissing)
	}
	return err
}
