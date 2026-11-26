package combat

import (
	"math"

	"github.com/Whealinghold/srd5e-go/content"
)

// DamageResolution is how one instance of damage went through the pipeline of
// SRD 5.2.1, "Order of Application", p. 17: the incoming amount, the amount
// after adjustments, after Resistance and after Vulnerability, and whether
// Immunity applied. It is comparable. Only the damage resolver builds one.
type DamageResolution struct {
	typ                                                             content.DamageType
	incoming, afterAdjustments, afterResistance, afterVulnerability int
	immune, unpreventable                                           bool
}

// Type returns the Damage Type of the instance.
func (r DamageResolution) Type() content.DamageType { return r.typ }

// Incoming returns the amount before the pipeline.
func (r DamageResolution) Incoming() int { return r.incoming }

// AfterAdjustments returns the amount after flat adjustments and Half Damage.
func (r DamageResolution) AfterAdjustments() int { return r.afterAdjustments }

// AfterResistance returns the amount after Resistance.
func (r DamageResolution) AfterResistance() int { return r.afterResistance }

// AfterVulnerability returns the amount after Vulnerability, before Immunity.
func (r DamageResolution) AfterVulnerability() int { return r.afterVulnerability }

// Immune reports whether Immunity applied.
func (r DamageResolution) Immune() bool { return r.immune }

// Unpreventable reports whether the instance skipped adjustments, Resistance and
// Immunity.
func (r DamageResolution) Unpreventable() bool { return r.unpreventable }

// Dealt returns the damage that reaches the creature: 0 under Immunity,
// otherwise the amount after Vulnerability.
func (r DamageResolution) Dealt() int {
	if r.immune {
		return 0
	}
	return r.afterVulnerability
}

// satAdd adds without overflowing int.
func satAdd(a, b int) int {
	switch {
	case b > 0 && a > math.MaxInt-b:
		return math.MaxInt
	case b < 0 && a < math.MinInt-b:
		return math.MinInt
	}
	return a + b
}

// resolveDamage runs one instance through the pipeline: flat adjustments
// (floored at 0), Half Damage, Resistance, Vulnerability; Immunity overrides.
// An unpreventable instance skips all but Vulnerability. Amounts saturate at
// math.MaxInt instead of overflowing.
func resolveDamage(d Damage, adjustments []DamageAdjustment, aff DamageAffinities, unpreventable bool) DamageResolution {
	r := DamageResolution{typ: d.typ, incoming: d.amount, unpreventable: unpreventable}
	a := d.amount
	if !unpreventable {
		flat, half := 0, false
		for _, adj := range adjustments {
			switch x := adj.(type) {
			case FlatAdjustment:
				if covers(x.scope, d.typ) {
					flat = satAdd(flat, x.amount)
				}
			case HalfDamage:
				half = true
			}
		}
		a = max(0, satAdd(a, flat))
		if half {
			a /= 2
		}
	}
	r.afterAdjustments = a
	if !unpreventable && aff.resists(d.typ) {
		a /= 2
	}
	r.afterResistance = a
	if aff.vulnerable(d.typ) {
		a = satAdd(a, a)
	}
	r.afterVulnerability = a
	r.immune = !unpreventable && aff.immuneTo(d.typ)
	return r
}
