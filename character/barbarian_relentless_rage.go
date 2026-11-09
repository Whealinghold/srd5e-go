package character

import "github.com/Whealinghold/srd5e-go/d20"

// RelentlessRageAvailability is the closed outcome of RelentlessRage.Availability. Variants:
// RelentlessRageAvailable, RelentlessRageUnavailable.
type RelentlessRageAvailability interface{ isRelentlessRageAvailability() }

// RelentlessRageAvailable: Relentless Rage can be used now.
type RelentlessRageAvailable struct{}

// RelentlessRageUnavailable: Relentless Rage cannot be used. Construct it with a keyed literal.
type RelentlessRageUnavailable struct{ Reason BarbarianReason }

func (RelentlessRageAvailable) isRelentlessRageAvailability()   {}
func (RelentlessRageUnavailable) isRelentlessRageAvailability() {}

// RelentlessRageResolution is the closed outcome of RelentlessRage.Resolve. Variants:
// RelentlessRageResolutionSucceeded, RelentlessRageResolutionStillFailed.
type RelentlessRageResolution interface{ isRelentlessRageResolution() }

// RelentlessRageResolutionSucceeded: the Constitution saving throw succeeded. Only
// RelentlessRage.Resolve builds one.
type RelentlessRageResolutionSucceeded struct{ outcome relentlessOutcome }

// RelentlessRageResolutionStillFailed: the Constitution saving throw failed; the use is expended all
// the same and no Hit Points are restored. Only RelentlessRage.Resolve builds one.
type RelentlessRageResolutionStillFailed struct{ outcome relentlessOutcome }

type relentlessOutcome struct {
	hitPoints int
	check     d20.CheckResult
	state     BarbarianState
}

// HitPoints returns the Hit Points to set: twice the Barbarian level.
func (r RelentlessRageResolutionSucceeded) HitPoints() int { return r.outcome.hitPoints }

// Check returns the saving throw.
func (r RelentlessRageResolutionSucceeded) Check() d20.CheckResult { return r.outcome.check }

// State returns the new state: one more use since the last rest, so the next DC is 5 higher.
func (r RelentlessRageResolutionSucceeded) State() BarbarianState { return r.outcome.state }

// Check returns the saving throw.
func (r RelentlessRageResolutionStillFailed) Check() d20.CheckResult { return r.outcome.check }

// State returns the new state: one more use since the last rest, so the next DC is 5 higher.
func (r RelentlessRageResolutionStillFailed) State() BarbarianState { return r.outcome.state }

func (RelentlessRageResolutionSucceeded) isRelentlessRageResolution()   {}
func (RelentlessRageResolutionStillFailed) isRelentlessRageResolution() {}

// RelentlessRage decides and resolves a use of Relentless Rage (SRD 5.2.1, "Level 11: Relentless
// Rage", p. 30). It is stateless and pure; the zero value is ready to use. Safe for concurrent use.
type RelentlessRage struct{}

// Availability reports whether the Barbarian can use Relentless Rage after dropping to 0 Hit
// Points. The first reason that holds is reported: BarbarianNotABarbarian,
// BarbarianFeatureNotGained (below level 11), BarbarianRageNotActive, BarbarianNotDroppedToZero.
func (RelentlessRage) Availability(stats BarbarianLevelStats, state BarbarianState, facts BarbarianHostFacts) RelentlessRageAvailability {
	switch {
	case stats.IsZero():
		return RelentlessRageUnavailable{Reason: BarbarianNotABarbarian}
	case stats.level.n < relentlessRageLevel:
		return RelentlessRageUnavailable{Reason: BarbarianFeatureNotGained}
	case !state.rageActive:
		return RelentlessRageUnavailable{Reason: BarbarianRageNotActive}
	case !facts.DroppedToZeroHitPoints:
		return RelentlessRageUnavailable{Reason: BarbarianNotDroppedToZero}
	}
	return RelentlessRageAvailable{}
}

// Resolve resolves the Constitution saving throw check, made against the Difficulty Class
// state.RelentlessRageDC(). On success the Hit Points become twice the Barbarian level. One use is
// counted whether the save succeeded or not, so the next DC is 5 higher.
//
// It returns an error matching srderr.ErrInvalidInput when Availability says unavailable (with
// DroppedToZeroHitPoints taken as true), when check is not a saving throw, or when its target is not
// the current DC. It never panics.
func (RelentlessRage) Resolve(stats BarbarianLevelStats, state BarbarianState, check d20.CheckResult) (RelentlessRageResolution, error) {
	if _, ok := (RelentlessRage{}).Availability(stats, state, BarbarianHostFacts{DroppedToZeroHitPoints: true}).(RelentlessRageAvailable); !ok {
		return nil, invalidInput("Relentless Rage resolved while unavailable")
	}
	if check.Kind() != d20.SavingThrow || int(check.Target()) != state.RelentlessRageDC() {
		return nil, invalidInput("Relentless Rage resolved with a check that is not a saving throw against the current DC")
	}
	state.relentlessRageUses++
	out := relentlessOutcome{check: check, state: state}
	if check.Outcome() != d20.Success {
		return RelentlessRageResolutionStillFailed{out}, nil
	}
	out.hitPoints = 2 * stats.level.n
	return RelentlessRageResolutionSucceeded{out}, nil
}

// PersistentRageOutcome is the closed outcome of PersistentRage.RegainOnInitiative. Variants:
// RagesRegainedOnInitiative, PersistentRageRefused.
type PersistentRageOutcome interface{ isPersistentRageOutcome() }

// RagesRegainedOnInitiative is the regaining of every expended Rage use. Only
// PersistentRage.RegainOnInitiative builds one.
type RagesRegainedOnInitiative struct {
	state  BarbarianState
	events []Event
}

// State returns the new state: no Rage use expended, Persistent Rage spent until a Long Rest.
func (r RagesRegainedOnInitiative) State() BarbarianState { return r.state }

// Events returns a new slice holding the RagesRegained event.
func (r RagesRegainedOnInitiative) Events() []Event { return append([]Event{}, r.events...) }

// PersistentRageRefused is a regaining a rule refused. Construct it with a keyed literal.
type PersistentRageRefused struct{ Reason BarbarianReason }

func (RagesRegainedOnInitiative) isPersistentRageOutcome() {}
func (PersistentRageRefused) isPersistentRageOutcome()     {}

// PersistentRage decides the Rage-regaining half of Persistent Rage (SRD 5.2.1, "Level 15:
// Persistent Rage", p. 30); the 10-minute duration is BarbarianRage's. It is stateless and pure; the
// zero value is ready to use. Safe for concurrent use.
type PersistentRage struct{}

// RegainOnInitiative regains every expended Rage use of the Barbarian id when it rolls Initiative.
// The first reason that holds refuses it, in this order: BarbarianNotABarbarian,
// BarbarianFeatureNotGained (below level 15), BarbarianPersistentRageSpent (used since the last Long
// Rest). It returns an error matching srderr.ErrInvalidInput for a zero id.
func (PersistentRage) RegainOnInitiative(id Identifier, stats BarbarianLevelStats, state BarbarianState) (PersistentRageOutcome, error) {
	if id.IsZero() {
		return nil, invalidInput("regain Rage uses for a zero character id")
	}
	switch {
	case stats.IsZero():
		return PersistentRageRefused{Reason: BarbarianNotABarbarian}, nil
	case stats.level.n < persistentRageLevel:
		return PersistentRageRefused{Reason: BarbarianFeatureNotGained}, nil
	case state.persistentRageUsed:
		return PersistentRageRefused{Reason: BarbarianPersistentRageSpent}, nil
	}
	state.ragesExpended, state.persistentRageUsed = 0, true
	return RagesRegainedOnInitiative{state: state, events: []Event{RagesRegained{id: id}}}, nil
}
