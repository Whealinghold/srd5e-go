package combat

import "github.com/Whealinghold/srd5e-go/content"

// LimitedUseEntry is what changes for one limited-use stat-block part or spell
// of one monster combatant (SRD 5.2.1, "Limited Usage", p. 257). It is a closed
// set with four variants: PerDayEntry, RechargeEntry, RestRechargeEntry and
// SpellUseEntry. Every variant is an immutable comparable value, copies the
// printed numbers it needs, and is safe for concurrent use.
type LimitedUseEntry interface{ isLimitedUseEntry() }

// PerDayEntry is an X/Day part: the printed uses, the printed uses in the lair
// when it has a figure, and the uses remaining. 0 <= Remaining <= the maximum
// for the lair status the state holds. Only a state builds one.
type PerDayEntry struct {
	part                         LimitedUsePartRef
	maximum, maximumInLair, left int
}

func (PerDayEntry) isLimitedUseEntry() {}

// Part returns the part.
func (e PerDayEntry) Part() LimitedUsePartRef { return e.part }

// Maximum returns the printed uses a day.
func (e PerDayEntry) Maximum() int { return e.maximum }

// MaximumInLair returns the printed uses a day in the lair, and false when the
// part has no such figure (it then keeps Maximum in the lair).
func (e PerDayEntry) MaximumInLair() (int, bool) { return e.maximumInLair, e.maximumInLair != 0 }

// Remaining returns the uses remaining.
func (e PerDayEntry) Remaining() int { return e.left }

// maximumFor is the maximum while the monster has status lair.
func (e PerDayEntry) maximumFor(lair LairStatus) int {
	if lair == InLair && e.maximumInLair != 0 {
		return e.maximumInLair
	}
	return e.maximum
}

// RechargeEntry is a Recharge X-Y part: the printed range and whether the part is
// ready. Only a state builds one.
type RechargeEntry struct {
	part      LimitedUsePartRef
	low, high int
	readiness RechargeReadiness
}

func (RechargeEntry) isLimitedUseEntry() {}

// Part returns the part.
func (e RechargeEntry) Part() LimitedUsePartRef { return e.part }

// Low returns the lowest roll of the 1d6 that recharges the part.
func (e RechargeEntry) Low() int { return e.low }

// High returns the highest roll of the 1d6 that recharges the part.
func (e RechargeEntry) High() int { return e.high }

// Readiness returns RechargeReady or RechargeSpent.
func (e RechargeEntry) Readiness() RechargeReadiness { return e.readiness }

// RestRechargeEntry is a "Recharge after a Short or Long Rest" part and whether
// it is ready. Only a state builds one.
type RestRechargeEntry struct {
	part      LimitedUsePartRef
	readiness RechargeReadiness
}

func (RestRechargeEntry) isLimitedUseEntry() {}

// Part returns the part.
func (e RestRechargeEntry) Part() LimitedUsePartRef { return e.part }

// Readiness returns RechargeReady or RechargeSpent.
func (e RestRechargeEntry) Readiness() RechargeReadiness { return e.readiness }

// SpellUseEntry is a spell of an N/Day or N/Day Each Spellcasting group: its own
// uses, never a pool shared by the group. 0 <= Remaining <= Maximum. Only a
// state builds one.
type SpellUseEntry struct {
	ref             SpellUseRef
	maximum, remain int
}

func (SpellUseEntry) isLimitedUseEntry() {}

// Spell returns the spell.
func (e SpellUseEntry) Spell() SpellUseRef { return e.ref }

// Maximum returns the printed casts a day.
func (e SpellUseEntry) Maximum() int { return e.maximum }

// Remaining returns the casts remaining.
func (e SpellUseEntry) Remaining() int { return e.remain }

// usageEntry returns the entry a part with the given usage needs for a monster with status lair, and false when
// the usage is not limited (no entry) or an error matching srderr.ErrInvalidInput
// when a number is outside the range the content spec gives it: content validated
// it structurally, and combat checks it again before relying on it (plan P13).
func usageEntry(ref LimitedUsePartRef, u content.PartUsage, lair LairStatus) (LimitedUseEntry, bool, error) {
	switch v := u.(type) {
	case content.PartUnlimited, content.PartUsageOther:
		return nil, false, nil
	case content.PartPerDay:
		inLair, _ := v.InLairCount()
		if !usesInRange(v.Count()) || (inLair != 0 && !usesInRange(inLair)) {
			return nil, false, invalidInput("X/Day part with a number outside the allowed range")
		}
		e := PerDayEntry{ref, v.Count(), inLair, 0}
		e.left = e.maximumFor(lair)
		return e, true, nil
	case content.PartRecharge:
		if v.Low() < content.MonsterMinRecharge || v.High() > content.MonsterMaxRecharge || v.Low() > v.High() {
			return nil, false, invalidInput("Recharge part with a range outside the die")
		}
		return RechargeEntry{ref, v.Low(), v.High(), RechargeReady}, true, nil
	case content.PartRechargeAfterRest:
		return RestRechargeEntry{ref, RechargeReady}, true, nil
	default:
		return nil, false, invalidInput("part with no usage")
	}
}

// usesInRange reports whether n is a use count content allows.
func usesInRange(n int) bool { return n >= content.MonsterMinUses && n <= content.MonsterMaxNumber }
