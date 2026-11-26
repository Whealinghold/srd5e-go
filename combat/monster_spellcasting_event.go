package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// MonsterSpellcastingEvent is the closed family of events the monster spellcasting
// rules produce (monster-spellcasting.md FR-034). Each is an immutable value,
// comparable with ==, named in the past tense, with a constant snake_case Kind and
// Combatants equal to the one monster; it carries the spell or Concentration subject,
// never text. Variants: MonsterSpellCast, MonsterCastingStarted,
// MonsterConcentrationBegan, MonsterConcentrationEnded, MonsterCastingFailed. Each
// embeds monsterSpellcastingEventBase.
type MonsterSpellcastingEvent interface {
	Event
	monsterSpellcastingEvent()
}

// monsterSpellcastingEventBase carries the monster an event is about and puts the
// event in the Event and MonsterSpellcastingEvent families.
type monsterSpellcastingEventBase struct{ caster CombatantID }

func (monsterSpellcastingEventBase) event()                    {}
func (monsterSpellcastingEventBase) monsterSpellcastingEvent() {}

// Combatants returns a new slice holding the monster.
func (b monsterSpellcastingEventBase) Combatants() []CombatantID { return []CombatantID{b.caster} }

// MonsterSpellCast: a cast finished. Its Kind is monster_spell_cast.
type MonsterSpellCast struct {
	monsterSpellcastingEventBase
	spell        content.SpellID
	castAtLevel  int
	scalingLevel int
}

// Kind returns "monster_spell_cast".
func (MonsterSpellCast) Kind() string { return "monster_spell_cast" }

// Spell returns the spell id.
func (e MonsterSpellCast) Spell() content.SpellID { return e.spell }

// CastAtLevel returns the level the spell was cast at and false for a cantrip.
func (e MonsterSpellCast) CastAtLevel() (int, bool) { return e.castAtLevel, e.castAtLevel > 0 }

// ScalingLevel returns the level the spell scales with (1 for a cantrip).
func (e MonsterSpellCast) ScalingLevel() int { return e.scalingLevel }

// MonsterCastingStarted: a casting of 1 minute or more began. Its Kind is
// monster_casting_started.
type MonsterCastingStarted struct {
	monsterSpellcastingEventBase
	spell   content.SpellID
	minutes int
}

// Kind returns "monster_casting_started".
func (MonsterCastingStarted) Kind() string { return "monster_casting_started" }

// Spell returns the spell id.
func (e MonsterCastingStarted) Spell() content.SpellID { return e.spell }

// TotalMinutes returns the whole minutes the casting takes.
func (e MonsterCastingStarted) TotalMinutes() int { return e.minutes }

// MonsterConcentrationBegan: the monster began to concentrate. Its Kind is
// monster_concentration_began.
type MonsterConcentrationBegan struct {
	monsterSpellcastingEventBase
	subject character.ConcentrationSubject
	kind    character.ConcentrationKind
}

// Kind returns "monster_concentration_began".
func (MonsterConcentrationBegan) Kind() string { return "monster_concentration_began" }

// Subject returns what the monster concentrates on.
func (e MonsterConcentrationBegan) Subject() character.ConcentrationSubject { return e.subject }

// ConcentrationKind returns how Concentration began.
func (e MonsterConcentrationBegan) ConcentrationKind() character.ConcentrationKind { return e.kind }

// MonsterConcentrationEnded: the monster's Concentration ended. Its Kind is
// monster_concentration_ended.
type MonsterConcentrationEnded struct {
	monsterSpellcastingEventBase
	subject character.ConcentrationSubject
	reason  character.ConcentrationEndReason
}

// Kind returns "monster_concentration_ended".
func (MonsterConcentrationEnded) Kind() string { return "monster_concentration_ended" }

// Subject returns what the monster concentrated on.
func (e MonsterConcentrationEnded) Subject() character.ConcentrationSubject { return e.subject }

// Reason returns why Concentration ended.
func (e MonsterConcentrationEnded) Reason() character.ConcentrationEndReason { return e.reason }

// MonsterCastingFailed: a casting of 1 minute or more failed and no use was spent.
// Its Kind is monster_casting_failed.
type MonsterCastingFailed struct {
	monsterSpellcastingEventBase
	spell  content.SpellID
	reason character.ConcentrationEndReason
}

// Kind returns "monster_casting_failed".
func (MonsterCastingFailed) Kind() string { return "monster_casting_failed" }

// Spell returns the spell that failed.
func (e MonsterCastingFailed) Spell() content.SpellID { return e.spell }

// Reason returns why the casting failed.
func (e MonsterCastingFailed) Reason() character.ConcentrationEndReason { return e.reason }
