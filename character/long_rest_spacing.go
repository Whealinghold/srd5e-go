package character

// LongRestSpacing is the time since the previous finished Long Rest, as at the
// start of the rest (SRD 5.2.1, "Long Rest": wait at least 16 hours before
// starting another). Only a Long Rest that was completed counts as finished: an
// interrupted one that earned Short Rest benefits, one stopped without benefits,
// an incomplete one and a Short Rest do not. The host updates the spacing only
// after a TakeRestApplied whose RestApplied has BenefitsOf LongRest and Source
// RestFinished. The host owns time; the package has no clock. Variants:
// NoLongRestFinishedYet, LongRestFinishedHoursAgo. A nil LongRestSpacing is read
// as NoLongRestFinishedYet.
type LongRestSpacing interface{ isLongRestSpacing() }

// NoLongRestFinishedYet is the spacing of a creature with no previous finished
// Long Rest: nothing blocks a Long Rest.
type NoLongRestFinishedYet struct{}

// LongRestFinishedHoursAgo is the spacing when the previous Long Rest finished
// some whole hours ago. Build it with NewLongRestFinishedHoursAgo; the zero
// value is 0 hours ago.
type LongRestFinishedHoursAgo struct{ hours int }

func (NoLongRestFinishedYet) isLongRestSpacing()    {}
func (LongRestFinishedHoursAgo) isLongRestSpacing() {}

// NewLongRestFinishedHoursAgo returns the spacing of a Long Rest that finished
// hours ago. It returns an error matching srderr.ErrInvalidInput for negative
// hours. It never panics.
func NewLongRestFinishedHoursAgo(hours int) (LongRestFinishedHoursAgo, error) {
	if hours < 0 {
		return LongRestFinishedHoursAgo{}, invalidInput("negative hours since the last Long Rest")
	}
	return LongRestFinishedHoursAgo{hours: hours}, nil
}

// Hours returns the whole hours since the previous Long Rest finished.
func (s LongRestFinishedHoursAgo) Hours() int { return s.hours }
