package combat_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// wiredRow is one wired use case run on fixed inputs: the operation it records, the owner's result it returns
// unchanged and the result class and name it is classified with.
type wiredRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	owner                   any // the owner's own result for the same arguments
}

func wiredRows(t *testing.T) []wiredRow {
	t.Helper()
	ctx := context.Background()
	magic := content.CastingTimeMagicAction{}
	heldNone := combat.HoldsNoSpell{}
	contact := combat.InContact(cid("tp"), cid("other"), 60, combat.ContactedMayReply)
	spirit := pzSpirit()
	zone := pzCreate(t, spirit, nil)
	lateZone := pzCreate(t, spirit, func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(3, "caster") })
	endedZone := combat.EndZone(zone, combat.ZoneEndDispelled).(combat.EndZoneEnded).Zone
	fullZone := zoneWithLedger(t, 256)
	poly := ssBegin(t, ssPolymorph(t))
	shapechange := ssShapechange(t)
	shapechange.Target = gid("other")
	hero := gid("hero")
	rrLate := rrCommand(t, combat.Revivify)
	rrLate.Circumstances = rrCircs(t, 2*time.Hour, false, false, combat.Intact) // dead too long for Revivify
	createCmd := combat.CreateZoneCommand{ID: luMust(combat.NewPersistentZoneID("zone")), Caster: cid("caster"), Spell: spirit, Shape: luMust(content.NewSphereArea(20)), Cast: pzTurn(1, "caster")}
	ledger := rrLedger(t, rrIncrease(t, "e1", "aid", 5))
	living := pcStatus(t, 20, 20, 0)
	rows := []wiredRow{
		{"ready a Magic action spell", resSpec + "FR-168, readying-a-spell.md section 7", "happy", "ready_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewReadyCombatantSpell(o...).Run(ctx, combat.ReadyCombatantSpellCommand{Caster: trkA, Trigger: "door", Spell: "fireball", CastingTime: magic})
			}, "applied", "completed", combat.ReadySpell(trkA, "door", "fireball", magic)},
		{"ready a spell of another casting time", resSpec + "FR-168", "unhappy (state)", "ready_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewReadyCombatantSpell(o...).Run(ctx, combat.ReadyCombatantSpellCommand{Caster: trkA, Trigger: "door", Spell: "fireball", CastingTime: hsMinutes(t, 1)})
			}, "refused", "ready_not_an_action", combat.ReadySpell(trkA, "door", "fireball", hsMinutes(t, 1))},
		{"release a held spell", resSpec + "FR-168", "happy", "release_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewReleaseCombatantSpell(o...).Run(ctx, combat.ReleaseCombatantSpellCommand{Held: hsHeld, Reaction: hsReaction(t, hsHeld)})
			}, "applied", "completed", combat.ReleaseHeldSpell(hsHeld, hsReaction(t, hsHeld))},
		{"release with no held spell", resSpec + "FR-168", "unhappy (state)", "release_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewReleaseCombatantSpell(o...).Run(ctx, combat.ReleaseCombatantSpellCommand{Held: heldNone, Reaction: hsReaction(t, hsHeld)})
			}, "refused", "release_no_held_spell", combat.ReleaseHeldSpell(heldNone, hsReaction(t, hsHeld))},
		{"dissipate a held spell", resSpec + "FR-168", "happy", "dissipate_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewDissipateCombatantSpell(o...).Run(ctx, combat.DissipateCombatantSpellCommand{Held: hsHeld, Cause: combat.DissipationReadyElapsed})
			}, "applied", "completed", combat.DissipateHeldSpell(hsHeld, combat.DissipationReadyElapsed)},
		{"dissipate with no held spell", resSpec + "FR-168", "unhappy (state)", "dissipate_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewDissipateCombatantSpell(o...).Run(ctx, combat.DissipateCombatantSpellCommand{Held: heldNone, Cause: combat.DissipationReadyElapsed})
			}, "refused", "dissipate_no_held_spell", combat.DissipateHeldSpell(heldNone, combat.DissipationReadyElapsed)},
		{"dissipate with an invalid cause", resSpec + "FR-168", "unhappy (invalid input)", "dissipate_combatant_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewDissipateCombatantSpell(o...).Run(ctx, combat.DissipateCombatantSpellCommand{Held: hsHeld, Cause: 0})
			}, "invalid", "dissipate_invalid_cause", combat.DissipateHeldSpell(hsHeld, 0)},
		{"end a telepathic contact", resSpec + "FR-168, telepathy.md section 7", "happy", "change_telepathic_contact",
			func(o ...combat.Option) (any, error) {
				return combat.NewChangeTelepathicContact(o...).Run(ctx, combat.ChangeTelepathicContactCommand{Request: combat.EndContact{State: contact, Telepath: cid("tp")}})
			}, "applied", "completed", combat.EndTelepathicContact(contact, cid("tp"))},
		{"reply with no contact", resSpec + "FR-168", "unhappy (state)", "change_telepathic_contact",
			func(o ...combat.Option) (any, error) {
				return combat.NewChangeTelepathicContact(o...).Run(ctx, combat.ChangeTelepathicContactCommand{Request: combat.ReplyToContact{Replier: cid("other")}})
			}, "refused", "contact_none", combat.TelepathicReply(combat.TelepathicContactState{}, cid("other"))},
		{"a malformed initiation", resSpec + "FR-168", "unhappy (invalid input)", "change_telepathic_contact",
			func(o ...combat.Option) (any, error) {
				return combat.NewChangeTelepathicContact(o...).Run(ctx, combat.ChangeTelepathicContactCommand{Request: combat.InitiateContact{}})
			}, "invalid", "contact_invalid", combat.InitiateTelepathicContact(combat.InitiateTelepathicContactCommand{})},
		{"create a zone", resSpec + "FR-168, persistent-spell-zones.md section 7", "happy", "create_persistent_zone",
			func(o ...combat.Option) (any, error) {
				return combat.NewCreatePersistentZone(o...).Run(ctx, combat.CreatePersistentZoneCommand{ID: luMust(combat.NewPersistentZoneID("zone")), Caster: cid("caster"),
					Spell: spirit, Shape: luMust(content.NewSphereArea(20)), Cast: pzTurn(1, "caster")})
			}, "applied", "completed", combat.CreateZone(createCmd)},
		{"create a zone from a spell with no zone", resSpec + "FR-168", "unhappy (state)", "create_persistent_zone",
			func(o ...combat.Option) (any, error) {
				return combat.NewCreatePersistentZone(o...).Run(ctx, combat.CreatePersistentZoneCommand{ID: luMust(combat.NewPersistentZoneID("zone")), Caster: cid("caster"),
					Spell: content.Spell{}, Shape: luMust(content.NewSphereArea(20)), Cast: pzTurn(1, "caster")})
			}, "refused", "zone_no_facts", combat.CreateZoneRejected{Reasons: []combat.CreateZoneRejection{combat.CreateNoZoneFacts}}},
		{"assess an event on an ended zone", resSpec + "FR-168", "unhappy (state)", "assess_zone_trigger",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.AssessTrigger{Zone: endedZone, Turn: pzTurn(1, "caster"),
					Event: combat.NewCreatureEntered(pzInBody("x"))}})
			}, "refused", "zone_refused", combat.AssessZoneTrigger(endedZone, pzTurn(1, "caster"), combat.NewCreatureEntered(pzInBody("x")))},
		{"assess an event on a stale turn", resSpec + "FR-168", "unhappy (invalid input)", "assess_zone_trigger",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.AssessTrigger{Zone: lateZone, Turn: pzTurn(1, "caster"),
					Event: combat.NewCreatureEntered(pzInBody("x"))}})
			}, "invalid", "zone_stale_turn", combat.AssessZoneTrigger(lateZone, pzTurn(1, "caster"), combat.NewCreatureEntered(pzInBody("x")))},
		{"assess an event for a 257th creature", resSpec + "FR-168, persistent-spell-zones.md Limits (plan P10)", "unhappy (boundary)", "assess_zone_trigger",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.AssessTrigger{Zone: fullZone, Turn: pzTurn(1, "caster"),
					Event: combat.NewCreatureEndedTurn(pzInBody("extra"))}})
			}, "invalid", "limit_exceeded", combat.AssessZoneTrigger(fullZone, pzTurn(1, "caster"), combat.NewCreatureEndedTurn(pzInBody("extra")))},
		{"report the start of a turn", resSpec + "FR-168", "happy", "move_persistent_zone",
			func(o ...combat.Option) (any, error) {
				return combat.NewMovePersistentZone(o...).Run(ctx, combat.MovePersistentZoneCommand{Request: combat.StartOfTurnRequest{Zone: zone, Turn: pzTurn(1, "caster")}})
			}, "applied", "completed", combat.ZoneStartOfTurn(zone, pzTurn(1, "caster"))},
		{"move a zone that cannot move", resSpec + "FR-168", "unhappy (state)", "move_persistent_zone",
			func(o ...combat.Option) (any, error) {
				return combat.NewMovePersistentZone(o...).Run(ctx, combat.MovePersistentZoneCommand{Request: combat.MoveZoneRequest{Zone: zone, Move: luMust(combat.NewDriftMove(5)), Turn: pzTurn(1, "caster")}})
			}, "refused", "zone_move_refused", combat.MoveZone(zone, luMust(combat.NewDriftMove(5)), pzTurn(1, "caster"))},
		{"end a zone", resSpec + "FR-168", "happy", "end_persistent_zone",
			func(o ...combat.Option) (any, error) {
				return combat.NewEndPersistentZone(o...).Run(ctx, combat.EndPersistentZoneCommand{Request: combat.EndZoneRequest{Zone: zone, Reason: combat.ZoneEndDispelled}})
			}, "applied", "completed", combat.EndZone(zone, combat.ZoneEndDispelled)},
		{"end a zone that ended", resSpec + "FR-168", "unhappy (state)", "end_persistent_zone",
			func(o ...combat.Option) (any, error) {
				return combat.NewEndPersistentZone(o...).Run(ctx, combat.EndPersistentZoneCommand{Request: combat.EndZoneRequest{Zone: endedZone, Reason: combat.ZoneEndDispelled}})
			}, "refused", "zone_already_ended", combat.EndZone(endedZone, combat.ZoneEndDispelled)},
		{"begin a transformation", resSpec + "FR-168, shape-changing-spells.md section 7", "happy", "begin_shape_shift",
			func(o ...combat.Option) (any, error) {
				return combat.NewBeginShapeShifting(o...).Run(ctx, combat.BeginShapeShiftingCommand(ssPolymorph(t)))
			}, "applied", "completed", poly},
		{"Shapechange of another creature", resSpec + "FR-168", "unhappy (state)", "begin_shape_shift",
			func(o ...combat.Option) (any, error) {
				return combat.NewBeginShapeShifting(o...).Run(ctx, combat.BeginShapeShiftingCommand(shapechange))
			}, "refused", "shapechange_not_self", nil},
		{"assess a transformation that continues", resSpec + "FR-168", "happy", "assess_shape_shift",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessShapeShift(o...).Run(ctx, combat.AssessShapeShiftCommand{State: poly.Transformation(), Status: poly.Status()})
			}, "applied", "completed", nil},
		{"end a transformation", resSpec + "FR-168", "happy", "end_shape_shift",
			func(o ...combat.Option) (any, error) {
				return combat.NewEndShapeShifting(o...).Run(ctx, combat.EndShapeShiftingCommand{State: poly.Transformation(), Cause: combat.CauseDispelled,
					HitPoints: combat.TrackedHitPoints{Status: poly.Status()}})
			}, "applied", "completed", nil},
		{"revive a creature that died too long ago", resSpec + "FR-169, revival-and-restoration-spells.md section 7", "unhappy (state)", "revive_by_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewReviveCreatureBySpell(o...).Run(ctx, combat.ReviveCreatureBySpellCommand(rrLate))
			}, "refused", "ineligible", combat.ReviveBySpell(rrLate)},
		{"revive a dead creature", resSpec + "FR-169", "happy", "revive_by_spell",
			func(o ...combat.Option) (any, error) {
				return combat.NewReviveCreatureBySpell(o...).Run(ctx, combat.ReviveCreatureBySpellCommand(rrCommand(t, combat.Revivify)))
			}, "applied", "revived", combat.ReviveBySpell(rrCommand(t, combat.Revivify))},
		{"increase the maximum", resSpec + "FR-169", "happy", "increase_maximum",
			func(o ...combat.Option) (any, error) {
				return combat.NewIncreaseHitPointMaximum(o...).Run(ctx, combat.IncreaseHitPointMaximumCommand{Subject: hero, Status: living, Increase: rrIncrease(t, "e2", "aid", 5)})
			}, "applied", "increased", nil},
		{"end an increase that is not present", resSpec + "FR-169", "unhappy (state)", "end_maximum_increase",
			func(o ...combat.Option) (any, error) {
				return combat.NewEndHitPointMaximumIncrease(o...).Run(ctx, combat.EndHitPointMaximumIncreaseCommand{Subject: hero, Status: living, Ledger: ledger, EffectID: meEID("missing")})
			}, "refused", "refused", nil},
		{"end an increase", resSpec + "FR-169", "happy", "end_maximum_increase",
			func(o ...combat.Option) (any, error) {
				return combat.NewEndHitPointMaximumIncrease(o...).Run(ctx, combat.EndHitPointMaximumIncreaseCommand{Subject: hero, Status: living, Ledger: ledger, EffectID: meEID("e1")})
			}, "applied", "ended", nil},
		{"restore the maximum of a creature that is dead", resSpec + "FR-169", "unhappy (state)", "restore_maximum",
			func(o ...combat.Option) (any, error) {
				return combat.NewRestoreHitPointMaximum(o...).Run(ctx, combat.RestoreHitPointMaximumCommand{Subject: hero, Status: deadStatus(t, combat.DeathSavingThrowFailures), Amount: 5})
			}, "refused", "refused", nil},
		{"Lesser Restoration with nothing to end", resSpec + "FR-169", "unhappy (state)", "lesser_restoration",
			func(o ...combat.Option) (any, error) {
				return combat.NewCastLesserRestoration(o...).Run(ctx, combat.CastLesserRestorationCommand{Subject: hero, Set: meConditions(), Kind: content.ConditionBlinded})
			}, "refused", "refused", nil},
		{"Lesser Restoration", resSpec + "FR-169", "happy", "lesser_restoration",
			func(o ...combat.Option) (any, error) {
				return combat.NewCastLesserRestoration(o...).Run(ctx, combat.CastLesserRestorationCommand{Subject: hero, Set: meConditions(combat.Blinded{}), Kind: content.ConditionBlinded})
			}, "applied", "completed", nil},
		{"Greater Restoration with nothing to end", resSpec + "FR-169", "unhappy (state)", "greater_restoration",
			func(o ...combat.Option) (any, error) {
				return combat.NewCastGreaterRestoration(o...).Run(ctx, combat.CastGreaterRestorationCommand{Subject: hero, Choice: combat.EndCharmed{}, Conditions: meConditions(), Status: living})
			}, "refused", "refused", nil},
		{"Death Ward spends on a drop to 0", resSpec + "FR-169", "happy", "guard_with_death_ward",
			func(o ...combat.Option) (any, error) {
				return combat.NewGuardDamageWithDeathWard(o...).Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: hero, Status: pcStatus(t, 5, 20, 0), Ward: combat.WardActive,
					Request: combat.WardDamage{Damages: []combat.Damage{combat.MustDamage(9, content.DamageFire)}, Delivery: combat.OtherSource, Kind: combat.NormalHit}})
			}, "applied", "warded", nil},
		{"Death Ward spent", resSpec + "FR-169", "unhappy (state)", "guard_with_death_ward",
			func(o ...combat.Option) (any, error) {
				return combat.NewGuardDamageWithDeathWard(o...).Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: hero, Status: pcStatus(t, 5, 20, 0), Ward: combat.WardSpent, Request: combat.WardInstantKill{}})
			}, "applied", "not_warded", nil},
	}
	return rows
}

// wiredMoreRows are the rows of the variants wiredRows does not reach: every result of the telepathy and zone use cases and the rest of the shape-shift and revival results.
func wiredMoreRows(t *testing.T) []wiredRow {
	t.Helper()
	ctx := context.Background()
	contact := combat.InContact(cid("tp"), cid("other"), 60, combat.ContactedMayReply)
	silent := combat.InContact(cid("tp"), cid("other"), 60, combat.ContactedMayNotReply)
	stunned := teleStunned(t)
	spirit := pzSpirit()
	zone := pzCreate(t, spirit, nil)
	lateZone := pzCreate(t, spirit, func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(3, "caster") })
	endedZone := combat.EndZone(zone, combat.ZoneEndDispelled).(combat.EndZoneEnded).Zone
	poly := ssBegin(t, ssPolymorph(t))
	shapechange := ssBegin(t, ssShapechange(t))
	hero := gid("hero")
	living := pcStatus(t, 20, 20, 0)
	initiate := func(edit func(*combat.InitiateTelepathicContactCommand)) wiredRowRun {
		c := teleBase()
		if edit != nil {
			edit(&c)
		}
		return func(o ...combat.Option) (any, error) {
			return combat.NewChangeTelepathicContact(o...).Run(ctx, combat.ChangeTelepathicContactCommand{Request: combat.InitiateContact{Command: c}})
		}
	}
	change := func(req combat.TelepathyRequest) wiredRowRun {
		return func(o ...combat.Option) (any, error) {
			return combat.NewChangeTelepathicContact(o...).Run(ctx, combat.ChangeTelepathicContactCommand{Request: req})
		}
	}
	tele := "change_telepathic_contact"
	spec := resSpec + "FR-168, telepathy.md section 7"
	rows := []wiredRow{
		{"initiate a contact", spec, "happy", tele, initiate(nil), "applied", "completed", nil},
		{"initiate: not telepathic", spec, "unhappy (state)", tele, initiate(func(c *combat.InitiateTelepathicContactCommand) { c.Capability = combat.TelepathyCapability{} }), "refused", "contact_initiator_not_telepathic", nil},
		{"initiate: initiator incapacitated", spec, "unhappy (state)", tele, initiate(func(c *combat.InitiateTelepathicContactCommand) { c.TelepathConditions = stunned }), "refused", "contact_initiator_incapacitated", nil},
		{"initiate: contacted incapacitated", spec, "unhappy (state)", tele, initiate(func(c *combat.InitiateTelepathicContactCommand) { c.ContactedConditions = stunned }), "refused", "contact_contacted_incapacitated", nil},
		{"initiate: out of range", spec, "unhappy (boundary)", tele, initiate(func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = 61 }), "refused", "contact_out_of_range", nil},
		{"initiate: cannot understand", spec, "unhappy (state)", tele, initiate(func(c *combat.InitiateTelepathicContactCommand) {
			c.Understanding = combat.ContactedUnderstandingNeither
		}), "refused", "contact_cannot_understand", nil},
		{"initiate: already made", spec, "unhappy (state)", tele, initiate(func(c *combat.InitiateTelepathicContactCommand) {
			c.Telepath, c.Contacted, c.State = cid("tp"), cid("other"), contact
		}), "refused", "contact_already_made", nil},
		{"report an incapacitated telepath", spec, "happy", tele, change(combat.ReportIncapacitated{State: contact, Creature: cid("tp"), Conditions: stunned}), "applied", "completed", nil},
		{"report a conscious telepath", spec, "happy", tele, change(combat.ReportIncapacitated{State: contact, Creature: cid("tp")}), "applied", "completed", nil},
		{"report a distance out of range", spec, "unhappy (boundary)", tele, change(combat.ReportDistance{State: contact, DistanceFeet: 61}), "applied", "completed", nil},
		{"report a distance in range", spec, "unhappy (boundary)", tele, change(combat.ReportDistance{State: contact, DistanceFeet: 60}), "applied", "completed", nil},
		{"the contacted replies", spec, "happy", tele, change(combat.ReplyToContact{State: contact, Replier: cid("other")}), "applied", "completed", nil},
		{"the contacted may not reply", spec, "unhappy (state)", tele, change(combat.ReplyToContact{State: silent, Replier: cid("other")}), "refused", "contact_reply_not_permitted", nil},
		{"a stranger replies", spec, "unhappy (state)", tele, change(combat.ReplyToContact{State: contact, Replier: cid("stranger")}), "refused", "contact_reply_not_contacted", nil},
		{"end a contact there is none of", spec, "unhappy (state)", tele, change(combat.EndContact{Telepath: cid("tp")}), "refused", "contact_none", nil},
		{"assess an entering creature", resSpec + "FR-168", "happy", "assess_zone_trigger", func(o ...combat.Option) (any, error) {
			return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.AssessTrigger{Zone: zone, Turn: pzTurn(1, "caster"), Event: combat.NewCreatureEntered(pzInBody("x"))}})
		}, "applied", "completed", nil},
		{"record a cast application", resSpec + "FR-168", "happy", "assess_zone_trigger", func(o ...combat.Option) (any, error) {
			return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.RecordCastApplication{Zone: zone, Creature: cid("x"), Turn: pzTurn(1, "caster")}})
		}, "applied", "completed", nil},
		{"record a cast application on an ended zone", resSpec + "FR-168", "unhappy (state)", "assess_zone_trigger", func(o ...combat.Option) (any, error) {
			return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.RecordCastApplication{Zone: endedZone, Creature: cid("x"), Turn: pzTurn(1, "caster")}})
		}, "refused", "zone_refused", nil},
		{"record a cast application on a stale turn", resSpec + "FR-168", "unhappy (invalid input)", "assess_zone_trigger", func(o ...combat.Option) (any, error) {
			return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.RecordCastApplication{Zone: lateZone, Creature: cid("x"), Turn: pzTurn(1, "caster")}})
		}, "invalid", "zone_stale_turn", nil},
		{"apply an optional effect on an ended zone", resSpec + "FR-168", "unhappy (state)", "assess_zone_trigger", func(o ...combat.Option) (any, error) {
			return combat.NewAssessPersistentZoneTrigger(o...).Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.ApplyOptionalEffect{Zone: endedZone, Turn: pzTurn(1, "caster")}})
		}, "refused", "zone_refused", nil},
		{"move a zone on a stale turn", resSpec + "FR-168", "unhappy (invalid input)", "move_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewMovePersistentZone(o...).Run(ctx, combat.MovePersistentZoneCommand{Request: combat.MoveZoneRequest{Zone: lateZone, Move: luMust(combat.NewDriftMove(5)), Turn: pzTurn(1, "caster")}})
		}, "invalid", "zone_stale_turn", nil},
		{"start a turn on a stale turn", resSpec + "FR-168", "unhappy (invalid input)", "move_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewMovePersistentZone(o...).Run(ctx, combat.MovePersistentZoneCommand{Request: combat.StartOfTurnRequest{Zone: lateZone, Turn: pzTurn(1, "caster")}})
		}, "invalid", "zone_stale_turn", nil},
		{"a round starts", resSpec + "FR-168", "happy", "end_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewEndPersistentZone(o...).Run(ctx, combat.EndPersistentZoneCommand{Request: combat.RoundStartedRequest{Zone: zone, Round: 2}})
		}, "applied", "completed", nil},
		{"damage to a zone with no budget", resSpec + "FR-168", "unhappy (state)", "end_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewEndPersistentZone(o...).Run(ctx, combat.EndPersistentZoneCommand{Request: combat.RecordDamageRequest{Zone: zone, Amount: 5}})
		}, "refused", "zone_refused", nil},
		{"damage to an ended zone", resSpec + "FR-168", "unhappy (state)", "end_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewEndPersistentZone(o...).Run(ctx, combat.EndPersistentZoneCommand{Request: combat.RecordDamageRequest{Zone: endedZone, Amount: 5}})
		}, "refused", "zone_refused", nil},
		{"Polymorph resisted", resSpec + "FR-168", "unhappy (rule interaction)", "begin_shape_shift", func(o ...combat.Option) (any, error) {
			c := ssPolymorph(t)
			c.Response = combat.Unwilling{Save: d20.Success}
			return combat.NewBeginShapeShifting(o...).Run(ctx, combat.BeginShapeShiftingCommand(c))
		}, "applied", "shape_shift_resisted", nil},
		{"change form of a creature that is not transformed", resSpec + "FR-168", "unhappy (state)", "change_shape_shift_form", func(o ...combat.Option) (any, error) {
			return combat.NewChangeShapeShiftForm(o...).Run(ctx, combat.ChangeShapeShiftFormCommand{Actor: gid("druid"), Form: ssForm(t, content.CreatureTypeBeast, "3", 50, content.SizeLarge)})
		}, "refused", "not_transformed", nil},
		{"assess a transformation whose form Hit Points are gone", resSpec + "FR-168", "unhappy (boundary)", "assess_shape_shift", func(o ...combat.Option) (any, error) {
			return combat.NewAssessShapeShift(o...).Run(ctx, combat.AssessShapeShiftCommand{State: poly.Transformation(), Status: dyingStatus(t, 20, 0, 0)})
		}, "applied", "shape_shift_ended", nil},
		{"end a transformation at will", resSpec + "FR-168", "unhappy (state)", "end_shape_shift", func(o ...combat.Option) (any, error) {
			return combat.NewEndShapeShifting(o...).Run(ctx, combat.EndShapeShiftingCommand{State: poly.Transformation(), Cause: combat.CauseEndedByTarget, HitPoints: combat.TrackedHitPoints{Status: poly.Status()}})
		}, "refused", "cannot_end_at_will", nil},
		{"increase the maximum of a creature that is dead", resSpec + "FR-169", "unhappy (state)", "increase_maximum", func(o ...combat.Option) (any, error) {
			return combat.NewIncreaseHitPointMaximum(o...).Run(ctx, combat.IncreaseHitPointMaximumCommand{Subject: hero, Status: deadStatus(t, combat.DeathSavingThrowFailures), Increase: rrIncrease(t, "e2", "aid", 5)})
		}, "refused", "refused", nil},
		{"restore the maximum", resSpec + "FR-169", "happy", "restore_maximum", func(o ...combat.Option) (any, error) {
			return combat.NewRestoreHitPointMaximum(o...).Run(ctx, combat.RestoreHitPointMaximumCommand{Subject: hero, Status: pcStatus(t, 12, 12, 0), Amount: 8})
		}, "applied", "completed", nil},
		{"Greater Restoration that ends a level of Exhaustion", resSpec + "FR-169", "happy", "greater_restoration", func(o ...combat.Option) (any, error) {
			return combat.NewCastGreaterRestoration(o...).Run(ctx, combat.CastGreaterRestorationCommand{Subject: hero, Choice: combat.EndOneExhaustionLevel{}, Conditions: meConditions(condLevel(2)), Status: living})
		}, "applied", "completed", nil},
		{"restorative healing", resSpec + "FR-169", "happy", "restorative_healing", func(o ...combat.Option) (any, error) {
			return combat.NewCastRestorativeHealing(o...).Run(ctx, combat.CastRestorativeHealingCommand{Spell: combat.Heal, Subject: hero, Status: pcStatus(t, 5, 20, 0), Set: meConditions(), Amount: 70, CastLevel: 6})
		}, "applied", "completed", nil},
		{"restorative healing of a creature that is dead", resSpec + "FR-169", "unhappy (state)", "restorative_healing", func(o ...combat.Option) (any, error) {
			return combat.NewCastRestorativeHealing(o...).Run(ctx, combat.CastRestorativeHealingCommand{Spell: combat.Heal, Subject: hero, Status: deadStatus(t, combat.DeathSavingThrowFailures), Set: meConditions(), Amount: 70, CastLevel: 6})
		}, "refused", "refused", nil},
		{"Death Ward keeps the ward", resSpec + "FR-169", "happy", "guard_with_death_ward", func(o ...combat.Option) (any, error) {
			return combat.NewGuardDamageWithDeathWard(o...).Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: hero, Status: living, Ward: combat.WardActive,
				Request: combat.WardDamage{Damages: []combat.Damage{combat.MustDamage(3, content.DamageFire)}, Delivery: combat.OtherSource, Kind: combat.NormalHit}})
		}, "applied", "completed", nil},
		{"Death Ward on a creature that is dead", resSpec + "FR-169", "unhappy (state)", "guard_with_death_ward", func(o ...combat.Option) (any, error) {
			return combat.NewGuardDamageWithDeathWard(o...).Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: hero, Status: deadStatus(t, combat.DeathSavingThrowFailures), Ward: combat.WardActive, Request: combat.WardInstantKill{}})
		}, "refused", "refused", nil},
		{"create a zone with a zero cast turn", resSpec + "FR-170, persistent-spell-zones.md section 7", "unhappy (invalid input)", "create_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewCreatePersistentZone(o...).Run(ctx, combat.CreatePersistentZoneCommand{ID: luMust(combat.NewPersistentZoneID("zone")), Caster: cid("caster"),
				Spell: spirit, Shape: luMust(content.NewSphereArea(20))})
		}, "invalid", "zone_create_invalid", nil},
		{"move a zone the caster may move", resSpec + "FR-170", "happy", "move_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewMovePersistentZone(o...).Run(ctx, combat.MovePersistentZoneCommand{Request: combat.MoveZoneRequest{Zone: pzFlamingSphere(t),
				Move: luMust(combat.NewCasterMove(30, content.BonusAction)), Turn: pzTurn(1, "caster")}})
		}, "applied", "completed", nil},
		{"damage to a zone within its budget", resSpec + "FR-170", "happy", "end_persistent_zone", func(o ...combat.Option) (any, error) {
			return combat.NewEndPersistentZone(o...).Run(ctx, combat.EndPersistentZoneCommand{Request: combat.RecordDamageRequest{Zone: pzGuardian(t), Amount: 5}})
		}, "applied", "completed", nil},
		{"change form of a Shapechange", resSpec + "FR-170", "happy", "change_shape_shift_form", func(o ...combat.Option) (any, error) {
			return combat.NewChangeShapeShiftForm(o...).Run(ctx, combat.ChangeShapeShiftFormCommand{Actor: gid("wizard"), State: shapechange.Transformation(),
				Form: ssForm(t, content.CreatureTypeDragon, "9", 90, content.SizeHuge), Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.ResizesToFit})
		}, "applied", "completed", nil},
		{"form Hit Points gone on a Shapechange", resSpec + "FR-170", "unhappy (rule interaction)", "end_shape_shift", func(o ...combat.Option) (any, error) {
			return combat.NewEndShapeShifting(o...).Run(ctx, combat.EndShapeShiftingCommand{State: shapechange.Transformation(), Cause: combat.CauseFormHitPointsDepleted,
				HitPoints: combat.TrackedHitPoints{Status: shapechange.Status()}})
		}, "applied", "shape_shift_unaffected", nil},
		{"revive a creature that is not dead", resSpec + "FR-169", "unhappy (state)", "revive_by_spell", func(o ...combat.Option) (any, error) {
			c := rrCommand(t, combat.Revivify)
			c.Status = pcStatus(t, 3, 20, 0)
			return combat.NewReviveCreatureBySpell(o...).Run(ctx, combat.ReviveCreatureBySpellCommand(c))
		}, "applied", "not_revived", nil},
		{"revive with a restored maximum below one", resSpec + "FR-169", "unhappy (invalid input)", "revive_by_spell", func(o ...combat.Option) (any, error) {
			c := rrCommand(t, combat.Resurrection)
			c.Status, c.RestoredMaximum = status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities()), new(int)
			return combat.NewReviveCreatureBySpell(o...).Run(ctx, combat.ReviveCreatureBySpellCommand(c))
		}, "refused", "refused", nil},
		{"Greater Restoration that the host resolves", resSpec + "FR-169", "happy", "greater_restoration", func(o ...combat.Option) (any, error) {
			return combat.NewCastGreaterRestoration(o...).Run(ctx, combat.CastGreaterRestorationCommand{Subject: hero, Choice: combat.EndAbilityScoreReduction{}, Conditions: meConditions(), Status: living})
		}, "applied", "completed", nil},
	}
	return rows
}

// wiredRowRun is the run of a wiredRow.
type wiredRowRun = func(opts ...combat.Option) (any, error)

// Requirement: doc/specs/combat-action-resolution.md FR-167, FR-168, FR-169, FR-170, FR-100, FR-101, FR-105 - each wired use case runs in one Instrumenter operation with its snake-case name, records the class and srd5e.combat.result of its owner's section 7 table (refused is no error status, invalid is), returns the owner's own result unchanged and equal with the no-op and a recording bundle.
// Path: per case.
func TestWiredUseCases(t *testing.T) {
	t.Parallel()
	for _, c := range append(wiredRows(t), wiredMoreRows(t)...) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("the recording bundle and Noop disagree [FR-105]")
			}
			if c.owner != nil && !reflect.DeepEqual(got, c.owner) {
				t.Errorf("result = %#v, want the owner's %#v unchanged [FR-167]", got, c.owner)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			wantErr := c.class == "invalid"
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != wantErr || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status %v [%s]", spans, wantErr, c.requirement)
			}
			outcome := "success"
			if c.class != "applied" {
				outcome = "failure"
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if errs := measured(rec, "srd5e.combat.errors"); len(errs) != 0 {
				t.Errorf("errors = %v, want none: a rule outcome is no error [FR-101]", errs)
			}
			resHygiene(t, rec, false)
		})
	}
}

// Requirement: testing.md section 4 (cancellation), combat-action-resolution.md FR-167 - an already-cancelled context is returned unwrapped by every wired use case before any signal.
// Path: unhappy (state).
func TestWiredUseCasesCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec, opts := recording()
	checks := map[string]error{}
	_, checks["ready"] = combat.NewReadyCombatantSpell(opts...).Run(ctx, combat.ReadyCombatantSpellCommand{})
	_, checks["telepathy"] = combat.NewChangeTelepathicContact(opts...).Run(ctx, combat.ChangeTelepathicContactCommand{})
	_, checks["zone"] = combat.NewEndPersistentZone(opts...).Run(ctx, combat.EndPersistentZoneCommand{})
	_, checks["shape"] = combat.NewEndShapeShifting(opts...).Run(ctx, combat.EndShapeShiftingCommand{})
	_, checks["revival"] = combat.NewRestoreHitPointMaximum(opts...).Run(ctx, combat.RestoreHitPointMaximumCommand{})
	for name, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s: err = %v, want context.Canceled unwrapped", name, err)
		}
	}
	if len(rec.Tracer.SpansNamed("srd5e.combat.ready_combatant_spell")) != 0 || len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")) != 0 {
		t.Errorf("a cancelled call recorded a signal")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-167, FR-096 - a request that is nil or unknown, a round below the zone's cast round and an invalid owner command are errors matching srderr.ErrInvalidInput, recorded on the span only and never counted as a result.
// Path: unhappy (invalid input).
func TestWiredUseCasesInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	zone := pzCreate(t, pzSpirit(), nil)
	for name, run := range map[string]func() error{
		"telepathy nil request": func() error {
			_, err := combat.NewChangeTelepathicContact().Run(ctx, combat.ChangeTelepathicContactCommand{})
			return err
		},
		"zone assess nil request": func() error {
			_, err := combat.NewAssessPersistentZoneTrigger().Run(ctx, combat.AssessPersistentZoneTriggerCommand{})
			return err
		},
		"zone move nil request": func() error {
			_, err := combat.NewMovePersistentZone().Run(ctx, combat.MovePersistentZoneCommand{})
			return err
		},
		"zone end nil request": func() error {
			_, err := combat.NewEndPersistentZone().Run(ctx, combat.EndPersistentZoneCommand{})
			return err
		},
		"round below the cast round": func() error {
			_, err := combat.NewEndPersistentZone().Run(ctx, combat.EndPersistentZoneCommand{Request: combat.RoundStartedRequest{Zone: zone, Round: -1}})
			return err
		},
		"negative zone damage": func() error {
			_, err := combat.NewEndPersistentZone().Run(ctx, combat.EndPersistentZoneCommand{Request: combat.RecordDamageRequest{Zone: zone, Amount: -1}})
			return err
		},
		"death ward nil request": func() error {
			_, err := combat.NewGuardDamageWithDeathWard().Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: gid("hero"), Status: pcStatus(t, 5, 20, 0), Ward: combat.WardActive})
			return err
		},
		"death ward unknown state": func() error {
			_, err := combat.NewGuardDamageWithDeathWard().Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: gid("hero"), Status: pcStatus(t, 5, 20, 0), Ward: 9, Request: combat.WardInstantKill{}})
			return err
		},
		"shape shift zero command": func() error {
			_, err := combat.NewBeginShapeShifting().Run(ctx, combat.BeginShapeShiftingCommand{})
			return err
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput [FR-096]", name, err)
		}
	}
	_ = time.Second
}

// Requirement: doc/specs/combat-action-resolution.md FR-168, readying-a-spell.md section 7 - releasing a held spell with a Reaction that is not a readied action's, or whose trigger is another's, is release_not_a_readied_reaction or release_reaction_mismatch (refused), and both are the owner's results unchanged.
// Path: per case.
func TestReleaseCombatantSpellRefusals(t *testing.T) {
	t.Parallel()
	other := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkA, Trigger: "bridge"}}
	feature := func() combat.ReactionTaken {
		_, ev := trkApplied(t)(trkABC(t).TakeReaction(trkA, trkAvail, combat.FeatureReaction{Feature: "parry", Granted: combat.NoStandardAction{}}))
		return ev[0].(combat.ReactionTaken)
	}()
	for _, c := range []struct {
		name, want string
		held       combat.HoldsSpell
		reaction   combat.ReactionTaken
	}{
		{"a Reaction that is no readied action", "release_not_a_readied_reaction", hsHeld, feature},
		{"another trigger", "release_reaction_mismatch", other, hsReaction(t, hsHeld)},
	} {
		rec, opts := recording()
		res, err := combat.NewReleaseCombatantSpell(opts...).Run(context.Background(), combat.ReleaseCombatantSpellCommand{Held: c.held, Reaction: c.reaction})
		if err != nil || !reflect.DeepEqual(res, combat.ReleaseHeldSpell(c.held, c.reaction)) {
			t.Errorf("%s: %v, %v, want the owner's result unchanged", c.name, res, err)
		}
		if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || ops[0]["srd5e.combat.result"] != c.want || ops[0]["srd5e.combat.result_class"] != "refused" {
			t.Errorf("%s: operations = %v, want %s, refused [FR-170]", c.name, ops, c.want)
		}
	}
}
