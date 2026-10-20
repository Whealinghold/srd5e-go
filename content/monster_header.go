package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// SwarmComposition names the size of the creatures that make up a swarm (SRD
// 5.2.1, "Swarm of Bats", p. 361): a swarm stat block holds the swarm's own
// size in its size options and this member size beside them. The zero
// SwarmComposition means the monster is not a swarm. Comparable and safe for
// concurrent use.
type SwarmComposition struct{ member CreatureSize }

// NewSwarmComposition returns a swarm of creatures of size member.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember when member is not a size. It never panics.
func NewSwarmComposition(member CreatureSize) (SwarmComposition, error) {
	if !member.Valid() {
		return SwarmComposition{}, blockError("SwarmComposition", BlockInvalidMember)
	}
	return SwarmComposition{member: member}, nil
}

// MemberSize returns the size of the swarm's members.
func (s SwarmComposition) MemberSize() CreatureSize { return s.member }

// IsZero reports whether s is the zero value, meaning no swarm.
func (s SwarmComposition) IsZero() bool { return s.member == 0 }

// StatBlockInitiative is the printed Initiative of a stat block: the modifier
// and the score, neither computed from the other (SRD 5.2.1, "Initiative", p.
// 254). The zero value is modifier 0 and score 0. Comparable and safe for
// concurrent use.
type StatBlockInitiative struct{ modifier, score int }

// NewStatBlockInitiative returns the printed modifier and score.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange when either has a magnitude above [MonsterMaxNumber]. It
// never panics.
func NewStatBlockInitiative(modifier, score int) (StatBlockInitiative, error) {
	if !monsterNumberOK(modifier) || !monsterNumberOK(score) {
		return StatBlockInitiative{}, blockError("StatBlockInitiative", BlockOutOfRange)
	}
	return StatBlockInitiative{modifier: modifier, score: score}, nil
}

// Modifier returns the printed Initiative modifier.
func (i StatBlockInitiative) Modifier() int { return i.modifier }

// Score returns the printed Initiative score.
func (i StatBlockInitiative) Score() int { return i.score }

// HitPointsEntry is the Hit Points of a stat block: the printed average and
// the Hit Point Dice (SRD 5.2.1, "Hit Points", p. 255). The two are separate
// members; nothing combines them. The zero value is invalid. Comparable and
// safe for concurrent use.
type HitPointsEntry struct {
	average int
	dice    dice.Expression
}

// NewHitPointsEntry returns the Hit Points with the printed average and the
// Hit Point Dice hitDice, whose count is at least 1, whose die is one of
// [MonsterHitPointDieFaces] and whose flat modifier is any integer.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an average below [MonsterMinHitPointsAverage] or above
// [MonsterMaxNumber], a count below [MonsterMinHitDiceCount] (the zero
// Expression included) or a die that is not an allowed one. It never panics.
func NewHitPointsEntry(average int, hitDice dice.Expression) (HitPointsEntry, error) {
	if average < MonsterMinHitPointsAverage || average > MonsterMaxNumber ||
		hitDice.Count() < MonsterMinHitDiceCount || !monsterDieFacesOK(hitDice.Die().Sides()) {
		return HitPointsEntry{}, blockError("HitPointsEntry", BlockOutOfRange)
	}
	return HitPointsEntry{average: average, dice: hitDice}, nil
}

// Average returns the printed average.
func (h HitPointsEntry) Average() int { return h.average }

// Dice returns the Hit Point Dice.
func (h HitPointsEntry) Dice() dice.Expression { return h.dice }

// SpeedMode is a way of moving a [SpeedEntry] lists (SRD 5.2.1, "Speed", p.
// 255). The wire spelling is the lower case word. The zero SpeedMode is
// invalid. Comparable and safe for concurrent use.
type SpeedMode int

// The speed modes.
const (
	SpeedWalk SpeedMode = iota + 1
	SpeedBurrow
	SpeedClimb
	SpeedFly
	SpeedSwim
)

func speedModeNames() []string { return []string{"walk", "burrow", "climb", "fly", "swim"} }

// Valid reports whether m is one of the five modes.
func (m SpeedMode) Valid() bool { return enumValid(speedModeNames(), int(m)) }

// String returns the lower case name of m, or "unknown" for an undefined value.
func (m SpeedMode) String() string { return enumString(speedModeNames(), int(m)) }

// ParseSpeedMode returns the mode spelled exactly s, such as "fly". Matching
// is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseSpeedMode(s string) (SpeedMode, error) {
	v, err := parseEnum("SpeedMode", speedModeNames(), s)
	return SpeedMode(v), err
}

// SpeedEntry is one printed speed of a stat block: a set of modes (such as
// {climb, fly} for "Climb or Fly 20 ft."), a distance in feet, whether it
// hovers and an optional qualifier such as "wolf form only" (SRD 5.2.1,
// "Speed", p. 255). The zero value is invalid. It holds a slice, so compare it
// with [SpeedEntry.Equal].
type SpeedEntry struct {
	modes     []SpeedMode
	feet      int
	hover     bool
	qualifier string
}

// NewSpeedEntry returns a speed of a copy of modes, feet, hover and
// qualifier ("" for none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockEmpty for
// no mode, BlockInvalidMember for an invalid mode, BlockDuplicate for a
// repeated mode, BlockOutOfRange for feet below 0 or above [MonsterMaxNumber],
// BlockConstraint for hover without the fly mode, then BlockBlankText or
// BlockInvalidCharacter for a non-empty qualifier that breaks the text rule.
// It never panics.
func NewSpeedEntry(modes []SpeedMode, feet int, hover bool, qualifier string) (SpeedEntry, error) {
	const block = "SpeedEntry"
	if len(modes) == 0 {
		return SpeedEntry{}, blockError(block, BlockEmpty)
	}
	var seen [6]bool
	for _, m := range modes {
		switch {
		case !m.Valid():
			return SpeedEntry{}, blockError(block, BlockInvalidMember)
		case seen[m]:
			return SpeedEntry{}, blockError(block, BlockDuplicate)
		}
		seen[m] = true
	}
	switch {
	case feet < 0 || feet > MonsterMaxNumber:
		return SpeedEntry{}, blockError(block, BlockOutOfRange)
	case hover && !seen[SpeedFly]:
		return SpeedEntry{}, blockError(block, BlockConstraint)
	}
	if qualifier != "" {
		if err := checkText(block, qualifier); err != nil {
			return SpeedEntry{}, err
		}
	}
	return SpeedEntry{modes: slices.Clone(modes), feet: feet, hover: hover, qualifier: qualifier}, nil
}

// Modes returns the modes in printed order, as a new slice.
func (s SpeedEntry) Modes() []SpeedMode { return slices.Clone(s.modes) }

// Feet returns the distance in feet.
func (s SpeedEntry) Feet() int { return s.feet }

// Hover reports whether the speed is printed "(hover)".
func (s SpeedEntry) Hover() bool { return s.hover }

// Qualifier returns the printed qualifier, or "" when there is none.
func (s SpeedEntry) Qualifier() string { return s.qualifier }

// Equal reports whether s and o hold the same modes in the same order and the
// same distance, hover and qualifier.
func (s SpeedEntry) Equal(o SpeedEntry) bool {
	return slices.Equal(s.modes, o.modes) && s.feet == o.feet && s.hover == o.hover && s.qualifier == o.qualifier
}

// speedKey identifies a speed entry for the duplicate rule: its set of modes
// and its qualifier.
type speedKey struct {
	modes     uint8
	qualifier string
}

func (s SpeedEntry) key() speedKey {
	var mask uint8
	for _, m := range s.modes {
		mask |= 1 << uint(m)
	}
	return speedKey{modes: mask, qualifier: s.qualifier}
}
