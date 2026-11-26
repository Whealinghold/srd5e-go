package combat

// ExhaustionReductionOutcome is the optional Exhaustion reduction part of
// RestoreBreathing. Variants: ExhaustionReductionMade,
// ExhaustionReductionNotAttempted.
type ExhaustionReductionOutcome interface{ isExhaustionReductionOutcome() }

// ExhaustionReductionMade: ConditionSet.ReduceExhaustion was called; Result is its
// result (ExhaustionReduced, ExhaustionEnded or ExhaustionReductionNotPresent).
type ExhaustionReductionMade struct{ Result ReduceExhaustionOnSetResult }

// ExhaustionReductionNotAttempted: the record held no Suffocation levels, so no
// call was made.
type ExhaustionReductionNotAttempted struct{}

func (ExhaustionReductionMade) isExhaustionReductionOutcome()         {}
func (ExhaustionReductionNotAttempted) isExhaustionReductionOutcome() {}

// ExhaustionRemovalCause is why no Exhaustion level could be removed. The zero
// value is invalid.
type ExhaustionRemovalCause int

// The causes.
const (
	// ExhaustionRemovalLocked: every level is held by Dehydration, Malnutrition or a contagion.
	ExhaustionRemovalLocked ExhaustionRemovalCause = iota + 1
	// ExhaustionRemovalNoExhaustion: the creature has no Exhaustion.
	ExhaustionRemovalNoExhaustion
)

// String returns "locked", "no_exhaustion" or "unknown".
func (c ExhaustionRemovalCause) String() string {
	return enumName([]string{"locked", "no_exhaustion"}, int(c))
}

// ExhaustionRemovalRejection is why RemoveExhaustionLevels removed nothing: how
// many levels were blocked and why. The zero value is not a rejection. Comparable.
type ExhaustionRemovalRejection struct {
	blocked int
	cause   ExhaustionRemovalCause
}

// Blocked returns the number of levels asked for that could not be removed.
func (r ExhaustionRemovalRejection) Blocked() int { return r.blocked }

// Cause returns why nothing was removed.
func (r ExhaustionRemovalRejection) Cause() ExhaustionRemovalCause { return r.cause }

// restored is what the rule of RestoreBreathing decided: the set and record after,
// whether the Exhaustion was reduced, the levels actually removed and the events.
type restored struct {
	set       ConditionSet
	record    HazardRecord
	reduction ExhaustionReductionOutcome
	removed   int
	events    []Event
}

// restoreBreathing is the rule of RestoreBreathing: the creature can breathe, so
// the Exhaustion it gained from suffocating goes (SRD 5.2.1, "Suffocation
// [Hazard]", p. 189). The events are the condition events, then BreathRestored; in
// the no-op case, nothing suffocating and no Suffocation levels recorded, there is
// none. The error is that of ConditionSet.ReduceExhaustion.
func restoreBreathing(state BreathState, set ConditionSet, target CombatantID, rec HazardRecord) (restored, error) {
	levels := rec.suffocation
	_, suffocating := state.(BreathSuffocating)
	if levels == 0 {
		out := restored{set: set, record: rec.clamped(set.exhaustionLevel()), reduction: ExhaustionReductionNotAttempted{}}
		if suffocating {
			out.events = []Event{BreathRestored{creatureEnvironmentEventBase{target}, 0}}
		}
		return out, nil
	}
	held := set.exhaustionLevel()
	res, err := set.ReduceExhaustion(target, levels)
	if err != nil {
		return restored{}, err
	}
	removed := min(levels, held)
	rec.suffocation = 0
	return restored{res.Set(), rec.clamped(res.Set().exhaustionLevel()), ExhaustionReductionMade{res}, removed,
		append(reductionEvents(res), BreathRestored{creatureEnvironmentEventBase{target}, removed})}, nil
}

// removal is what the rule of RemoveExhaustionLevels decided. A non-zero rejection
// means nothing was removed and the other fields are the input unchanged.
type removal struct {
	set       ConditionSet
	record    HazardRecord
	reduction ReduceExhaustionOnSetResult
	removed   int
	blocked   int
	events    []Event
	rejection ExhaustionRemovalRejection
}

// removeExhaustion is the rule of RemoveExhaustionLevels: it removes up to levels
// levels, never those the record locks. The error is ErrInvalidInput for levels
// below 1, checked first.
func removeExhaustion(set ConditionSet, target CombatantID, rec HazardRecord, levels int) (removal, error) {
	if levels < 1 {
		return removal{}, invalidInput("levels must be at least 1")
	}
	held := set.exhaustionLevel()
	rec = rec.clamped(held)
	removable := rec.RemovableLevels(held)
	if removable == 0 {
		cause := ExhaustionRemovalLocked
		if held == 0 {
			cause = ExhaustionRemovalNoExhaustion
		}
		return removal{set: set, record: rec, blocked: levels, rejection: ExhaustionRemovalRejection{levels, cause}}, nil
	}
	removed := min(levels, removable)
	res, err := set.ReduceExhaustion(target, removed)
	if err != nil {
		return removal{}, err
	}
	return removal{res.Set(), rec.clamped(res.Set().exhaustionLevel()), res, removed, levels - removed, reductionEvents(res), ExhaustionRemovalRejection{}}, nil
}

// reductionEvents widens the condition events of a reduction result.
func reductionEvents(res ReduceExhaustionOnSetResult) []Event {
	var changes []ConditionEvent
	switch v := res.(type) {
	case ExhaustionReduced:
		changes = v.events
	case ExhaustionEnded:
		changes = v.events
	}
	events := make([]Event, len(changes))
	for i, e := range changes {
		events[i] = e
	}
	return events
}
