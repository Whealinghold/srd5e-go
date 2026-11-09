package character

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const clericChannelColumn content.ClassColumnID = "channel_divinity"

// ClericProgressionReason is why NewClericProgression refused a definition. The
// zero value is not a reason.
type ClericProgressionReason int

const (
	// NotTheClericDefinition: the definition is not the Cleric's.
	NotTheClericDefinition ClericProgressionReason = iota + 1
	// ClericColumnMissing: the channel_divinity column is missing or not a count column.
	ClericColumnMissing
	// ClericRowsIncomplete: a level from 2 shows a dash or holds no count.
	ClericRowsIncomplete
)

// String returns "not_the_cleric_definition", "column_missing", "rows_incomplete" or "unknown".
func (r ClericProgressionReason) String() string {
	switch r {
	case NotTheClericDefinition:
		return "not_the_cleric_definition"
	case ClericColumnMissing:
		return "column_missing"
	case ClericRowsIncomplete:
		return "rows_incomplete"
	}
	return "unknown"
}

// ClericProgressionError reports one reason NewClericProgression refused a
// definition. It matches srderr.ErrInvalidInput with errors.Is and carries no
// display text. Build it with a keyed literal.
type ClericProgressionError struct{ Reason ClericProgressionReason }

// Error renders the error. The text is not a contract.
func (e *ClericProgressionError) Error() string {
	return "character: invalid Cleric progression: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (*ClericProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *ClericProgressionError of the same Reason.
func (e *ClericProgressionError) Is(target error) bool {
	t, ok := target.(*ClericProgressionError)
	return ok && t.Reason == e.Reason
}

// ClericPendingChoice is a Cleric-specific choice a level adds, a closed set of
// values (class-cleric.md FR-021). The zero value is not a choice.
type ClericPendingChoice int

const (
	// ClericPendingDivineOrder: the Divine Order, at Cleric level 1.
	ClericPendingDivineOrder ClericPendingChoice = iota + 1
	// ClericPendingBlessedStrikes: the Blessed Strikes option, at Cleric level 7.
	ClericPendingBlessedStrikes
)

// String returns "divine_order", "blessed_strikes_option" or "unknown".
func (c ClericPendingChoice) String() string {
	switch c {
	case ClericPendingDivineOrder:
		return "divine_order"
	case ClericPendingBlessedStrikes:
		return "blessed_strikes_option"
	}
	return "unknown"
}

// ClericLevelStats are the numbers a Cleric level gives (SRD 5.2.1, "Cleric
// Features" and "Level 2: Channel Divinity", pp. 36-37): the Channel Divinity
// maximum and the Divine Spark dice. It carries no Proficiency Bonus, which
// follows the total level (SRD 5.2.1, "Multiclassing", p. 25). Only
// ClericProgression.StatsAt builds one; the zero value is invalid. Immutable,
// comparable with ==, safe for concurrent use.
type ClericLevelStats struct {
	level            ClericLevel
	channel, sparkD8 int
}

// Level returns the Cleric level the numbers belong to.
func (s ClericLevelStats) Level() ClericLevel { return s.level }

// ChannelDivinity returns the maximum Channel Divinity uses; 0 below level 2.
func (s ClericLevelStats) ChannelDivinity() int { return s.channel }

// DivineSparkDice returns the number of d8s Divine Spark rolls: 1, 2 from level 7,
// 3 from level 13, 4 from level 18.
func (s ClericLevelStats) DivineSparkDice() int { return s.sparkD8 }

// PreserveLifePool returns five times the Cleric level (SRD 5.2.1, "Level 3:
// Preserve Life", p. 40).
func (s ClericLevelStats) PreserveLifePool() int { return 5 * s.level.n }

// IsZero reports whether s is the invalid zero value.
func (s ClericLevelStats) IsZero() bool { return s.level.IsZero() }

// ClericProgression turns a Cleric level into ClericLevelStats and lists the
// features and pending choices a level gains. The Channel Divinity maximum and
// the feature ids are read from the Cleric ClassDefinition when it is built, so
// a house-rule table changes the maximum with no code change; the Divine Spark
// dice are rule constants. It is immutable and holds no port or clock; the zero
// value is invalid (see IsZero). Safe for concurrent use.
type ClericProgression struct {
	channel  [MaxLevel]int
	features [MaxLevel][]content.FeatureID
	set      bool
}

// NewClericProgression reads the Channel Divinity column and the feature ids
// gained at levels 1 to 20 from def.
//
// It returns an error joining a *ClericProgressionError per reason found,
// matching srderr.ErrInvalidInput: NotTheClericDefinition for another class or
// the zero definition, ClericColumnMissing for a missing or non-count column and
// ClericRowsIncomplete for a level without its cell. It never panics.
func NewClericProgression(def content.ClassDefinition) (ClericProgression, error) {
	if def.ID() != ClericClassID {
		return ClericProgression{}, errors.Join(&ClericProgressionError{Reason: NotTheClericDefinition})
	}
	if col, err := def.Column(clericChannelColumn); err != nil || col.Kind() != content.ColumnKindCount {
		return ClericProgression{}, errors.Join(&ClericProgressionError{Reason: ClericColumnMissing})
	}
	var p ClericProgression
	for level := 1; level <= MaxLevel; level++ {
		row, _ := def.RowAt(level) // a definition holds rows 1 to 20, so the level is in range
		v, _ := row.Value(clericChannelColumn)
		switch cell := v.(type) {
		case content.ColumnCount:
			p.channel[level-1] = cell.Value()
		case content.ColumnNone:
			if level > 1 {
				return ClericProgression{}, errors.Join(&ClericProgressionError{Reason: ClericRowsIncomplete})
			}
		default:
			return ClericProgression{}, errors.Join(&ClericProgressionError{Reason: ClericRowsIncomplete})
		}
		gained, _ := def.EntriesGainedAt(level)
		for _, e := range gained {
			p.features[level-1] = append(p.features[level-1], e.ID())
		}
	}
	p.set = true
	return p, nil
}

// MustClericProgression is NewClericProgression for a definition known to be
// valid. It panics with the error NewClericProgression returns, and only then.
func MustClericProgression(def content.ClassDefinition) ClericProgression {
	p, err := NewClericProgression(def)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p ClericProgression) IsZero() bool { return !p.set }

// StatsAt returns the numbers of the Cleric level; the zero ClericLevelStats for
// the zero progression or level.
func (p ClericProgression) StatsAt(level ClericLevel) ClericLevelStats {
	if !p.set || level.IsZero() {
		return ClericLevelStats{}
	}
	return ClericLevelStats{level: level, channel: p.channel[level.n-1], sparkD8: 1 + thresholdCount(level.n, 7, 13, 18)}
}

// FeaturesGainedAt returns the ids of the features the Cleric gains at level, in
// printed order, as a new slice; empty for a level that gains none, for the zero
// progression or for the zero level.
func (p ClericProgression) FeaturesGainedAt(level ClericLevel) []content.FeatureID {
	if !p.set || level.IsZero() {
		return []content.FeatureID{}
	}
	return append([]content.FeatureID{}, p.features[level.n-1]...)
}

// ChoicesOnGaining returns the Cleric-specific choices due on gaining the level:
// the Divine Order at level 1 and the Blessed Strikes option at level 7. The
// Ability Score Improvement, Epic Boon and subclass choices belong to level
// advancement and are not reported. A level with none gives an empty slice.
func (p ClericProgression) ChoicesOnGaining(level ClericLevel) []ClericPendingChoice {
	out := []ClericPendingChoice{}
	if !p.set {
		return out
	}
	switch level.n {
	case 1:
		out = append(out, ClericPendingDivineOrder)
	case 7:
		out = append(out, ClericPendingBlessedStrikes)
	}
	return out
}
