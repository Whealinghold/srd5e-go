package combat

import "github.com/Whealinghold/srd5e-go/content"

// ZoneEffectDue is one recurring part of a persistent zone that is due against
// one creature. The host resolves it with ResolveSpellEffects restricted to the
// part and units instances (persistent-spell-zones FR-054); this package rolls,
// saves and applies nothing. CastMarker is the zone's cast turn (it orders the
// same spell's zones, FR-029); Turn is the turn the event was assessed on.
type ZoneEffectDue struct {
	zone         PersistentZoneID
	spell        content.SpellID
	cast         TurnMarker
	turn         TurnMarker
	caster       CombatantID
	creature     CombatantID
	part         int
	trigger      content.ZoneTrigger
	units        int
	casterChoice bool
}

// ZoneID returns the zone the effect belongs to.
func (d ZoneEffectDue) ZoneID() PersistentZoneID { return d.zone }

// Spell returns the spell the zone came from.
func (d ZoneEffectDue) Spell() content.SpellID { return d.spell }

// CastMarker returns the turn the zone was cast on.
func (d ZoneEffectDue) CastMarker() TurnMarker { return d.cast }

// Turn returns the turn the triggering event was assessed on.
func (d ZoneEffectDue) Turn() TurnMarker { return d.turn }

// Combatants returns the creature the effect is due against, then the caster.
func (d ZoneEffectDue) Combatants() []CombatantID { return []CombatantID{d.creature, d.caster} }

// PartIndex returns the part of the spell's model that is due.
func (d ZoneEffectDue) PartIndex() int { return d.part }

// Trigger returns the trigger that made the effect due.
func (d ZoneEffectDue) Trigger() content.ZoneTrigger { return d.trigger }

// Units returns how many instances of the part are due: 1 for every trigger but
// TravelsWithin, the whole units travelled for TravelsWithin.
func (d ZoneEffectDue) Units() int { return d.units }

// CasterChoice reports whether the effect is optional: the caster chooses
// whether to apply it through ApplyOptionalZoneEffect.
func (d ZoneEffectDue) CasterChoice() bool { return d.casterChoice }
