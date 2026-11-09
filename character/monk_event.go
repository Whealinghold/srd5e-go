package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// MonkEvent is something that happened to a Monk, in the past tense and immutable, the
// closed family of Monk events. Every one is also an Event and carries the Identifier it
// concerns. The interface cannot be implemented outside this package. Members:
// FocusPointsExpended, FocusPointsRegained, UncannyMetabolismUsed, AttackDeflected,
// DeflectedForceRedirected, WholenessOfBodyUsed, StunningStrikeAttempted,
// QuiveringPalmStarted, QuiveringPalmEnded, DisciplinedSurvivorUsed,
// SuperiorDefenseActivated.
type MonkEvent interface {
	Event
	isMonkEvent()
}

// FocusPointsExpended is the event of Focus Points expended on a Flurry of Blows, Patient
// Defense or Step of the Wind.
type FocusPointsExpended struct {
	id      Identifier
	feature MonkFocusFeature
	count   int
}

// CharacterID returns the id of the Character.
func (e FocusPointsExpended) CharacterID() Identifier { return e.id }

// Feature returns the feature the points were spent on.
func (e FocusPointsExpended) Feature() MonkFocusFeature { return e.feature }

// Count returns the Focus Points expended.
func (e FocusPointsExpended) Count() int { return e.count }

// FocusPointsRegained is the event of Focus Points regained by Perfect Focus.
type FocusPointsRegained struct {
	id    Identifier
	count int
}

// CharacterID returns the id of the Character.
func (e FocusPointsRegained) CharacterID() Identifier { return e.id }

// Count returns the Focus Points regained.
func (e FocusPointsRegained) Count() int { return e.count }

// UncannyMetabolismUsed is the event of a use of Uncanny Metabolism.
type UncannyMetabolismUsed struct {
	id                          Identifier
	rolled, hitPoints, regained int
}

// CharacterID returns the id of the Character.
func (e UncannyMetabolismUsed) CharacterID() Identifier { return e.id }

// Rolled returns the Martial Arts die rolled.
func (e UncannyMetabolismUsed) Rolled() int { return e.rolled }

// HitPointsToRegain returns the uncapped Hit Points to regain.
func (e UncannyMetabolismUsed) HitPointsToRegain() int { return e.hitPoints }

// FocusPointsRegained returns the Focus Points regained.
func (e UncannyMetabolismUsed) FocusPointsRegained() int { return e.regained }

// AttackDeflected is the event of an attack's damage reduced by Deflect Attacks.
type AttackDeflected struct {
	id                       Identifier
	rolled, reduction, after int
}

// CharacterID returns the id of the Character.
func (e AttackDeflected) CharacterID() Identifier { return e.id }

// Rolled returns the d10 rolled.
func (e AttackDeflected) Rolled() int { return e.rolled }

// Reduction returns the damage reduction.
func (e AttackDeflected) Reduction() int { return e.reduction }

// DamageAfter returns the damage that remained.
func (e AttackDeflected) DamageAfter() int { return e.after }

// DeflectedForceRedirected is the event of a deflected attack's force redirected at a creature.
type DeflectedForceRedirected struct {
	id         Identifier
	dc         d20.DifficultyClass
	damageType content.DamageType
	applied    int
}

// CharacterID returns the id of the Character.
func (e DeflectedForceRedirected) CharacterID() Identifier { return e.id }

// DifficultyClass returns the save DC used.
func (e DeflectedForceRedirected) DifficultyClass() d20.DifficultyClass { return e.dc }

// DamageType returns the type of the redirected damage.
func (e DeflectedForceRedirected) DamageType() content.DamageType { return e.damageType }

// DamageApplied returns the damage the creature takes.
func (e DeflectedForceRedirected) DamageApplied() int { return e.applied }

// WholenessOfBodyUsed is the event of a use of Wholeness of Body.
type WholenessOfBodyUsed struct {
	id                Identifier
	rolled, hitPoints int
}

// CharacterID returns the id of the Character.
func (e WholenessOfBodyUsed) CharacterID() Identifier { return e.id }

// Rolled returns the Martial Arts die rolled.
func (e WholenessOfBodyUsed) Rolled() int { return e.rolled }

// HitPointsToRegain returns the uncapped Hit Points to regain.
func (e WholenessOfBodyUsed) HitPointsToRegain() int { return e.hitPoints }

// StunningStrikeAttempted is the event of a Stunning Strike attempt.
type StunningStrikeAttempted struct {
	id      Identifier
	dc      d20.DifficultyClass
	stunned bool
}

// CharacterID returns the id of the Character.
func (e StunningStrikeAttempted) CharacterID() Identifier { return e.id }

// DifficultyClass returns the save DC used.
func (e StunningStrikeAttempted) DifficultyClass() d20.DifficultyClass { return e.dc }

// Stunned reports that the target failed its saving throw and has the Stunned condition.
func (e StunningStrikeAttempted) Stunned() bool { return e.stunned }

// QuiveringPalmStarted is the event of vibrations started in a creature.
type QuiveringPalmStarted struct {
	id     Identifier
	target QuiveringPalmTarget
	days   int
}

// CharacterID returns the id of the Character.
func (e QuiveringPalmStarted) CharacterID() Identifier { return e.id }

// Target returns the creature the vibrations are in.
func (e QuiveringPalmStarted) Target() QuiveringPalmTarget { return e.target }

// DurationDays returns the days the vibrations last.
func (e QuiveringPalmStarted) DurationDays() int { return e.days }

// QuiveringPalmEnded is the event of vibrations ended.
type QuiveringPalmEnded struct {
	id         Identifier
	withEffect bool
	damage     int
}

// CharacterID returns the id of the Character.
func (e QuiveringPalmEnded) CharacterID() Identifier { return e.id }

// WithEffect reports that the vibrations ended with effect and not harmlessly.
func (e QuiveringPalmEnded) WithEffect() bool { return e.withEffect }

// DamageApplied returns the Force damage dealt; 0 for a harmless end.
func (e QuiveringPalmEnded) DamageApplied() int { return e.damage }

// DisciplinedSurvivorUsed is the event of a Focus Point expended to reroll a saving throw.
type DisciplinedSurvivorUsed struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e DisciplinedSurvivorUsed) CharacterID() Identifier { return e.id }

// SuperiorDefenseActivated is the event of Superior Defense taken up.
type SuperiorDefenseActivated struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e SuperiorDefenseActivated) CharacterID() Identifier { return e.id }

func (FocusPointsExpended) isEvent()          {}
func (FocusPointsRegained) isEvent()          {}
func (UncannyMetabolismUsed) isEvent()        {}
func (AttackDeflected) isEvent()              {}
func (DeflectedForceRedirected) isEvent()     {}
func (WholenessOfBodyUsed) isEvent()          {}
func (StunningStrikeAttempted) isEvent()      {}
func (QuiveringPalmStarted) isEvent()         {}
func (QuiveringPalmEnded) isEvent()           {}
func (DisciplinedSurvivorUsed) isEvent()      {}
func (SuperiorDefenseActivated) isEvent()     {}
func (FocusPointsExpended) isMonkEvent()      {}
func (FocusPointsRegained) isMonkEvent()      {}
func (UncannyMetabolismUsed) isMonkEvent()    {}
func (AttackDeflected) isMonkEvent()          {}
func (DeflectedForceRedirected) isMonkEvent() {}
func (WholenessOfBodyUsed) isMonkEvent()      {}
func (StunningStrikeAttempted) isMonkEvent()  {}
func (QuiveringPalmStarted) isMonkEvent()     {}
func (QuiveringPalmEnded) isMonkEvent()       {}
func (DisciplinedSurvivorUsed) isMonkEvent()  {}
func (SuperiorDefenseActivated) isMonkEvent() {}
