package character

import "github.com/Whealinghold/srd5e-go/content"

// RestKind is the kind of rest a creature takes: a Short Rest or a Long Rest
// (SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185). The two are the only
// values; a Long Rest is not also a Short Rest. The zero RestKind is invalid;
// see Valid. Comparable and safe for concurrent use.
type RestKind int

// The kinds of rest.
const (
	// ShortRest is a rest of at least 1 hour.
	ShortRest RestKind = iota + 1
	// LongRest is a rest of at least 8 hours (4 with the Trance trait).
	LongRest
)

// Valid reports whether k is ShortRest or LongRest.
func (k RestKind) Valid() bool { return k == ShortRest || k == LongRest }

// String returns "short" or "long", the value of the rest_kind telemetry
// attribute, or "unknown" for an undefined value.
func (k RestKind) String() string {
	switch k {
	case ShortRest:
		return "short"
	case LongRest:
		return "long"
	}
	return "unknown"
}

// MinimumHours returns the whole hours a rest of kind k needs: 1 for a
// ShortRest whatever v is, 8 for a LongRest of StandardLongRest and 4 for
// TranceLongRest. It returns 0 for an invalid kind, and for a LongRest with an
// invalid v.
func (k RestKind) MinimumHours(v LongRestVariant) int {
	switch {
	case k == ShortRest:
		return 1
	case k == LongRest && v == StandardLongRest:
		return 8
	case k == LongRest && v == TranceLongRest:
		return 4
	}
	return 0
}

// Recharges reports whether a rest of kind k restores a feature whose recharge
// trigger is t: a ShortRest restores RechargeShortRest and
// RechargeShortOrLongRest, a LongRest restores RechargeLongRest and
// RechargeShortOrLongRest (SRD 5.2.1, "Short Rest", "Special Feature", p. 187;
// "Long Rest", "Special Feature", p. 185). A Long Rest does not recharge a
// feature that says Short Rest only. An invalid k or t recharges nothing.
func (k RestKind) Recharges(t content.RechargeTrigger) bool {
	switch t {
	case content.RechargeShortRest:
		return k == ShortRest
	case content.RechargeLongRest:
		return k == LongRest
	case content.RechargeShortOrLongRest:
		return k.Valid()
	}
	return false
}

// LongRestVariant says whether the creature finishes a Long Rest in the standard
// time or has the Trance trait (SRD 5.2.1, "Elf", Trance, p. 85). The host
// reports it. The zero LongRestVariant is invalid.
type LongRestVariant int

// The Long Rest variants.
const (
	// StandardLongRest needs 8 hours.
	StandardLongRest LongRestVariant = iota + 1
	// TranceLongRest needs 4 hours.
	TranceLongRest
)

// Valid reports whether v is one of the two variants.
func (v LongRestVariant) Valid() bool { return v == StandardLongRest || v == TranceLongRest }

// RestInterruption is what can interrupt a rest: Rolling Initiative, casting a
// spell other than a cantrip, taking any damage, or 1 hour of walking or other
// physical exertion (SRD 5.2.1, "Long Rest", "Interrupting the Rest", p. 185;
// "Short Rest", p. 187). Casting a cantrip is not representable. The zero
// RestInterruption is invalid.
type RestInterruption int

// The interruptions.
const (
	// RollingInitiative: the creature rolled Initiative.
	RollingInitiative RestInterruption = iota + 1
	// CastingSpellOtherThanCantrip: the creature cast a spell that is not a cantrip.
	CastingSpellOtherThanCantrip
	// TakingDamage: the creature took any damage.
	TakingDamage
	// PhysicalExertion: the creature walked or exerted itself.
	PhysicalExertion
)

// Valid reports whether i is one of the four interruptions.
func (i RestInterruption) Valid() bool { return i >= RollingInitiative && i <= PhysicalExertion }

// String returns the snake_case name of i, or "unknown".
func (i RestInterruption) String() string {
	switch i {
	case RollingInitiative:
		return "rolling_initiative"
	case CastingSpellOtherThanCantrip:
		return "casting_spell_other_than_cantrip"
	case TakingDamage:
		return "taking_damage"
	case PhysicalExertion:
		return "physical_exertion"
	}
	return "unknown"
}

// Stops reports whether i stops a rest of kind k. All four stop a Long Rest. A
// Short Rest is stopped by the first three (its list) and by PhysicalExertion
// (an inference from its definition, which allows nothing more strenuous than
// reading, talking, eating or standing watch). An invalid i or k stops nothing.
func (i RestInterruption) Stops(k RestKind) bool { return i.Valid() && k.Valid() }

// RestSource says how the benefits of a rest came about. The zero RestSource is
// invalid.
type RestSource int

// The sources of rest benefits.
const (
	// RestFinished: the rest ran its course.
	RestFinished RestSource = iota + 1
	// LongRestInterrupted: an interrupted Long Rest that earned the benefits of a
	// Short Rest. It recharges features but opens no "finish a Short Rest"
	// window.
	LongRestInterrupted
)

// Valid reports whether s is one of the two sources.
func (s RestSource) Valid() bool { return s == RestFinished || s == LongRestInterrupted }
