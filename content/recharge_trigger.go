package content

// RechargeTrigger is the rest that restores a limited-use feature: a Short
// Rest, a Long Rest, or either (SRD 5.2.1, "Limited Usage", p. 257). A Long
// Rest is not a Short Rest, so a feature restored by both uses
// [RechargeShortOrLongRest]. "X/Day" is data for [RechargeLongRest], and
// "Recharge X-Y" is not a trigger. The wire spelling is the package tag in
// lower camel case, not the printed phrase. The zero RechargeTrigger is
// invalid. Comparable and safe for concurrent use.
type RechargeTrigger int

// The recharge triggers.
const (
	// RechargeShortRest: restored by a Short Rest.
	RechargeShortRest RechargeTrigger = iota + 1
	// RechargeLongRest: restored by a Long Rest.
	RechargeLongRest
	// RechargeShortOrLongRest: restored by a Short Rest or a Long Rest.
	RechargeShortOrLongRest
)

func rechargeTriggerNames() []string {
	return []string{"shortRest", "longRest", "shortOrLongRest"}
}

// Valid reports whether r is one of the three triggers.
func (r RechargeTrigger) Valid() bool { return enumValid(rechargeTriggerNames(), int(r)) }

// String returns the wire spelling of r, or "unknown" for an undefined value.
func (r RechargeTrigger) String() string { return enumString(rechargeTriggerNames(), int(r)) }

// ParseRechargeTrigger returns the trigger spelled exactly s: "shortRest",
// "longRest" or "shortOrLongRest". Matching is case-sensitive and does not
// trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including the printed phrase "Short Rest". It never panics.
func ParseRechargeTrigger(s string) (RechargeTrigger, error) {
	v, err := parseEnum("RechargeTrigger", rechargeTriggerNames(), s)
	return RechargeTrigger(v), err
}
