package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
)

// DeathBurstResult is the closed set of results of TriggerDeathBurst. Variants:
// DeathBurstTriggered, DeathBurstNotTriggered, DeathBurstLimitExceeded.
type DeathBurstResult interface{ isDeathBurstResult() }

// DeathBurstTriggered: the creature died, so the burst of the part happens. It names the
// saving throw of the part (its ability and difficulty class); its damage, Emanation and
// targets are the part's StatBlockSavingThrow and clauses, which the host resolves.
type DeathBurstTriggered struct {
	part LimitedUsePartRef
	save ability.Ability
	dc   int
}

// Part returns the position of the Death Burst part in the stat block.
func (t DeathBurstTriggered) Part() LimitedUsePartRef { return t.part }

// Ability returns the ability of the burst's saving throw.
func (t DeathBurstTriggered) Ability() ability.Ability { return t.save }

// DC returns the difficulty class of the burst's saving throw.
func (t DeathBurstTriggered) DC() int { return t.dc }

// DeathBurstNotTriggered: the events hold no death of the creature.
type DeathBurstNotTriggered struct{}

// DeathBurstLimitExceeded: events held more than 256 events (plan P10), so none was read and nothing can be said
// of the burst. The host reads it as the shared limit_exceeded.
type DeathBurstLimitExceeded struct{}

func (DeathBurstTriggered) isDeathBurstResult()     {}
func (DeathBurstNotTriggered) isDeathBurstResult()  {}
func (DeathBurstLimitExceeded) isDeathBurstResult() {}

// TriggerDeathBurst says whether the Death Burst of t happens (SRD 5.2.1, "Magmin", p. 305;
// "Mephits", p. 307). It does when events holds a Died for subject, whatever its cause,
// including the events of a later RegenerationLetDie. The events of an Undead Fortitude
// survival or of a deferred death carry no Died, so they do not trigger it. The burst is a
// fact only: nothing is rolled, targeted or applied. A zero trait or subject, a nil event
// and the events of another creature give DeathBurstNotTriggered, and more than 256 events
// DeathBurstLimitExceeded (plan P10). It never fails.
func TriggerDeathBurst(t DeathBurstTrait, subject CombatantID, events []Event) DeathBurstResult {
	if t.IsZero() || subject.IsZero() {
		return DeathBurstNotTriggered{}
	}
	if len(events) > maxDeathBurstEvents {
		return DeathBurstLimitExceeded{}
	}
	for _, e := range events {
		if d, ok := e.(Died); ok && d.subject == subject {
			return DeathBurstTriggered{t.part, t.ability, t.dc}
		}
	}
	return DeathBurstNotTriggered{}
}
