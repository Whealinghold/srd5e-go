package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpAZoneRule builds one zone rule on part 0, in the default region, for any creature.
func grpAZoneRule(trigger content.ZoneTrigger, scope content.ZoneTargetScope, choice bool) content.ZoneRuleFact {
	return grpAMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: trigger, PartIndex: 0, Scope: scope, CasterChoice: choice}))
}

// grpAZoneSpell returns a synthetic 3rd-level spell with one damaging part (1d6 radiant, a Wisdom save for half) that
// holds the zone facts, or no zone facts when facts is nil. It is a fixture: the engine reads its data, not its name.
func grpAZoneSpell(id string, facts *content.ZoneFactsFields) content.Spell {
	d6 := grpAMust(content.NewRolledSpellAmount(dice.MustParseExpression("1d6")))
	radiant := grpAMust(content.NewFixedDamageType(content.DamageRadiant))
	dmg := grpAMust(content.NewSpellDealsDamage(grpAMust(content.NewScalableAmount(d6, nil)), radiant, content.AddsNothing))
	save := grpAMust(content.NewSavingThrowResolution(ability.Wisdom, []content.SpellOutcome{dmg}, content.HalfDamageOnSuccess))
	part := grpAMust(content.NewSpellEffectPart(save, grpAMust(content.NewPartInstances(1, nil))))
	area := grpAMust(content.NewTargetsInArea(grpAMust(content.NewEmanationArea(15)), content.SubjectCreature, nil))
	model := grpAMust(content.NewModelledEffects(content.CoverageComplete, area, []content.SpellEffectPart{part}))
	if facts != nil {
		model = grpAMust(model.WithZoneFacts(grpAMust(content.NewZoneFacts(*facts))))
	}
	return grpAMust(content.NewSpell(content.SpellID(id), id, content.SourceCitation{}, content.SpellFields{
		Level: content.SpellLevel(3), School: content.SchoolEvocation, Classes: []content.ClassID{"cleric"},
		CastingTimes: grpAMust(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction())),
		Range:        content.NewSpellRangeSelf(), Components: content.NewSpellComponents(true, true),
		Duration:    grpAMust(content.NewSpellDurationTimed(10, content.DurationUnitMinutes, content.LimitConcentrationUpTo)),
		Description: []content.TextBlock{content.MustTextParagraph("Synthetic fixture.")}, Effects: model,
	}))
}

// grpAZoneCommand is the command that creates the zone of spell cast by caster on turn marker, with the given extras.
func grpAZoneCommand(zoneID, caster string, spell content.Spell, cast combat.TurnMarker) combat.CreatePersistentZoneCommand {
	return combat.CreatePersistentZoneCommand{ID: combat.PersistentZoneID(zoneID), Caster: combat.MustCombatantID(caster), Spell: spell,
		Shape: grpAMust(content.NewEmanationArea(15)), Cast: cast}
}

func grpATurn(round int, who string) combat.TurnMarker {
	return grpAMust(combat.NewTurnMarker(round, combat.MustCombatantID(who)))
}

// grpAZone creates an active zone through the use case, failing the test otherwise.
func grpAZone(t *testing.T, cmd combat.CreatePersistentZoneCommand) combat.PersistentZone {
	t.Helper()
	res, err := combat.NewCreatePersistentZone().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	created, ok := res.(combat.CreateZoneCreated)
	if !ok {
		t.Fatalf("CreatePersistentZone = %#v, want created", res)
	}
	return created.Zone
}

var grpAGuardians = content.ZoneFactsFields{
	Rules: []content.ZoneRuleFact{
		grpAZoneRule(content.CreatureEnters, content.EnemiesOfCaster, false),
		grpAZoneRule(content.CreatureEndsTurnInside, content.EnemiesOfCaster, false),
	},
	Mobility: content.MovesWithOrigin{}, OncePerTurn: true, Designation: content.DesignationAtCastingOnly,
}

// Requirement: doc/specs/persistent-spell-zones.md FR-003, FR-004, FR-005, FR-007, FR-008, FR-032 - a cleric casts Spirit Guardians on its turn: the zone is created active with one zone_created and the designated ally exempt; a spell with no zone facts is refused alone; a missing id, a missing cast turn, a designation the facts do not allow and a repeated designation are each an invalid result listing every reason.
// Path: unhappy (invalid input).
func TestPersistentSpellZonesCreationScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	cast := grpATurn(1, "cleric")
	spirit := grpAZoneSpell("spirit_guardians", &grpAGuardians)

	cmd := grpAZoneCommand("spirit", "cleric", spirit, cast)
	cmd.Designated = []combat.CombatantID{id("paladin")}
	res, err := combat.NewCreatePersistentZone().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	created, ok := res.(combat.CreateZoneCreated)
	if !ok || created.Zone.Ended() || !created.Zone.Exemptions().Contains(id("paladin")) || !created.Zone.EndsWithConcentration() || created.Event.Kind() != "zone_created" {
		t.Fatalf("create = %#v, want an active zone with the paladin exempt, ending with Concentration, and zone_created", res)
	}

	reject := func(c combat.CreatePersistentZoneCommand) []combat.CreateZoneRejection {
		t.Helper()
		r, err := combat.NewCreatePersistentZone().Run(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		rej, ok := r.(combat.CreateZoneRejected)
		if !ok {
			t.Fatalf("result = %#v, want rejected", r)
		}
		return rej.Reasons
	}
	textOnly := grpAZoneCommand("plain", "cleric", grpAZoneSpell("plain_spell", nil), cast)
	if got := reject(textOnly); !slices.Equal(got, []combat.CreateZoneRejection{combat.CreateNoZoneFacts}) {
		t.Errorf("a spell with no zone facts = %v, want NoZoneFacts alone", got)
	}
	noDesignation := grpAZoneSpell("no_designation", &content.ZoneFactsFields{
		Rules: []content.ZoneRuleFact{grpAZoneRule(content.CreatureEnters, content.AnyCreature, false)}})
	bad := grpAZoneCommand("", "cleric", noDesignation, combat.TurnMarker{})
	bad.Designated = []combat.CombatantID{id("paladin")}
	got := reject(bad)
	for _, want := range []combat.CreateZoneRejection{combat.CreateMissingZoneID, combat.CreateMissingCastTurn, combat.CreateDesignationNotAllowed} {
		if !slices.Contains(got, want) {
			t.Errorf("reasons = %v, want %v among them (every problem is listed)", got, want)
		}
	}
	dup := grpAZoneCommand("spirit2", "cleric", spirit, cast)
	dup.Designated = []combat.CombatantID{id("paladin"), id("paladin")}
	if got := reject(dup); !slices.Contains(got, combat.CreateDuplicateExemption) {
		t.Errorf("a repeated designation = %v, want DuplicateExemption", got)
	}
	outOfRange := grpAZoneSpell("bad_part", &content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		grpAMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, PartIndex: 3, Scope: content.AnyCreature}))}})
	if got := reject(grpAZoneCommand("far", "cleric", outOfRange, cast)); !slices.Contains(got, combat.CreatePartIndexOutOfRange) {
		t.Errorf("a rule naming part 3 of a one-part spell = %v, want PartIndexOutOfRange", got)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-012, FR-013, FR-014, FR-016, FR-023, FR-025, FR-026, FR-030, FR-055, FR-022 - an enemy enters Spirit Guardians and ends its turn inside: one due effect each, and a second on the same turn is skipped (once per turn) while the next turn or another creature is due; the caster's designated ally is never due; a creature that is not an enemy is skipped; a choice rule waits for the caster to apply it, once; the appearance of the zone counts for the creature; stale and ended-zone calls are refused.
// Path: unhappy (rule interaction).
func TestPersistentSpellZonesTriggerScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	enemy, ally, friend := id("orc"), id("paladin"), id("squire")
	body := []content.ZoneRegionID{content.DefaultZoneRegion}
	cast := grpATurn(2, "cleric")
	cmd := grpAZoneCommand("spirit", "cleric", grpAZoneSpell("spirit_guardians", &grpAGuardians), cast)
	cmd.Designated = []combat.CombatantID{ally}
	zone := grpAZone(t, cmd)

	assess := func(z combat.PersistentZone, turn combat.TurnMarker, event combat.ZoneTriggerEvent) combat.AssessPersistentZoneTriggerResult {
		t.Helper()
		res, err := combat.NewAssessPersistentZoneTrigger().Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.AssessTrigger{Zone: z, Turn: turn, Event: event}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	enemyIn := combat.NewZoneCreatureFacts(enemy, body, true, true)
	entered, ok := assess(zone, cast, combat.NewCreatureEntered(enemyIn)).(combat.AssessZoneTriggerDue)
	if !ok || len(entered.Mandatory) != 1 || len(entered.Events) != 1 || entered.Events[0].Kind() != "zone_triggered" || entered.Mandatory[0].Units() != 1 {
		t.Fatalf("an enemy entering = %#v, want one due effect and one zone_triggered", entered)
	}
	zone = entered.Zone
	again, ok := assess(zone, cast, combat.NewCreatureEndedTurn(enemyIn)).(combat.AssessZoneTriggerDue)
	if !ok || len(again.Mandatory) != 0 || len(again.Skipped) != 1 || again.Skipped[0].Reason != combat.SkipAlreadyAffectedThisTurn {
		t.Errorf("the same creature on the same turn = %#v, want skipped AlreadyAffectedThisTurn", again)
	}
	later, ok := assess(zone, grpATurn(2, "orc"), combat.NewCreatureEndedTurn(enemyIn)).(combat.AssessZoneTriggerDue)
	if !ok || len(later.Mandatory) != 1 {
		t.Errorf("the next turn = %#v, want a new due effect", later)
	}
	exempt, ok := assess(zone, cast, combat.NewCreatureEntered(combat.NewZoneCreatureFacts(ally, body, true, true))).(combat.AssessZoneTriggerDue)
	if !ok || len(exempt.Mandatory) != 0 || len(exempt.Skipped) != 1 || exempt.Skipped[0].Reason != combat.SkipDesignatedExempt {
		t.Errorf("the designated ally = %#v, want skipped DesignatedExempt", exempt)
	}
	notEnemy, ok := assess(zone, cast, combat.NewCreatureEntered(combat.NewZoneCreatureFacts(friend, body, false, true))).(combat.AssessZoneTriggerDue)
	if !ok || len(notEnemy.Mandatory) != 0 || notEnemy.Skipped[0].Reason != combat.SkipNotAnEnemy {
		t.Errorf("a creature that is not an enemy = %#v, want skipped NotAnEnemy", notEnemy)
	}
	outside, ok := assess(zone, cast, combat.NewCreatureEntered(combat.NewZoneCreatureFacts(enemy, nil, true, true))).(combat.AssessZoneTriggerDue)
	if !ok || len(outside.Mandatory) != 0 || outside.Skipped[0].Reason != combat.SkipNotInside {
		t.Errorf("a creature in no region = %#v, want skipped NotInside", outside)
	}
	if got := assess(zone, grpATurn(1, "orc"), combat.NewCreatureEntered(enemyIn)); got != combat.AssessPersistentZoneTriggerResult(combat.AssessZoneTriggerStaleTurn{}) {
		t.Errorf("a turn before the cast = %#v, want stale", got)
	}

	// A zone whose rule is the caster's choice: the due effect waits, is applied once, and a second try is refused.
	choice := grpAZoneSpell("flaming_sphere", &content.ZoneFactsFields{
		Rules:       []content.ZoneRuleFact{grpAZoneRule(content.CreatureEndsTurnInside, content.AnyCreature, true)},
		Mobility:    content.StationaryZone{},
		OncePerTurn: true,
	})
	sphere := grpAZone(t, grpAZoneCommand("sphere", "cleric", choice, cast))
	ownTurn := grpATurn(2, "orc")
	waiting, ok := assess(sphere, ownTurn, combat.NewCreatureEndedTurn(enemyIn)).(combat.AssessZoneTriggerDue)
	if !ok || len(waiting.Mandatory) != 0 || len(waiting.Optional) != 1 || !waiting.Optional[0].CasterChoice() {
		t.Fatalf("a choice rule = %#v, want one optional due effect and none mandatory", waiting)
	}
	apply := func(z combat.PersistentZone, due combat.ZoneEffectDue) combat.AssessPersistentZoneTriggerResult {
		t.Helper()
		res, err := combat.NewAssessPersistentZoneTrigger().Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.ApplyOptionalEffect{Zone: z, Turn: ownTurn, Due: due}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	chosen, ok := apply(waiting.Zone, waiting.Optional[0]).(combat.OptionalZoneEffectApplied)
	if !ok {
		t.Fatalf("applying the optional effect = %#v", chosen)
	}
	if refused, ok := apply(chosen.Zone, waiting.Optional[0]).(combat.OptionalZoneEffectRefused); !ok || refused.Reason != combat.SkipAlreadyAffectedThisTurn {
		t.Errorf("applying it twice = %#v, want refused AlreadyAffectedThisTurn", refused)
	}
	if refused, ok := apply(zone, waiting.Optional[0]).(combat.OptionalZoneEffectRefused); !ok || refused.Reason != combat.SkipDueNotForThisZone {
		t.Errorf("a due effect of another zone = %#v, want refused DueNotForThisZone", refused)
	}

	record := func(z combat.PersistentZone, who combat.CombatantID, turn combat.TurnMarker) combat.AssessPersistentZoneTriggerResult {
		t.Helper()
		res, err := combat.NewAssessPersistentZoneTrigger().Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.RecordCastApplication{Zone: z, Creature: who, Turn: turn}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	recorded, ok := record(zone, friend, cast).(combat.RecordCastApplicationRecorded)
	if !ok {
		t.Fatalf("recording the appearance = %#v", recorded)
	}
	if got := assess(recorded.Zone, cast, combat.NewCreatureEntered(combat.NewZoneCreatureFacts(friend, body, true, true))).(combat.AssessZoneTriggerDue); len(got.Mandatory) != 0 {
		t.Errorf("a creature the zone appeared on = %#v, want no further due effect that turn", got)
	}
	freeZone := grpAZone(t, grpAZoneCommand("free", "cleric", grpAZoneSpell("free_zone", &content.ZoneFactsFields{
		Rules: []content.ZoneRuleFact{grpAZoneRule(content.CreatureEnters, content.AnyCreature, false)}}), cast))
	if got := record(freeZone, friend, cast); got != combat.AssessPersistentZoneTriggerResult(combat.RecordCastApplicationNotOncePerTurn{}) {
		t.Errorf("recording on a zone with no once-per-turn ledger = %#v, want refused", got)
	}
	if got := record(zone, friend, grpATurn(1, "cleric")); got != combat.AssessPersistentZoneTriggerResult(combat.RecordCastApplicationStaleTurn{}) {
		t.Errorf("recording before the cast = %#v, want stale", got)
	}

	ended := combat.EndZone(zone, combat.ZoneEndDispelled).(combat.EndZoneEnded).Zone
	if got := assess(ended, cast, combat.NewCreatureEntered(enemyIn)); got != combat.AssessPersistentZoneTriggerResult(combat.AssessZoneTriggerEnded{}) {
		t.Errorf("assessing an ended zone = %#v, want ended", got)
	}
	if got := record(ended, friend, cast); got != combat.AssessPersistentZoneTriggerResult(combat.RecordCastApplicationZoneEnded{}) {
		t.Errorf("recording on an ended zone = %#v, want ended", got)
	}
	if _, err := combat.NewAssessPersistentZoneTrigger().Run(ctx, combat.AssessPersistentZoneTriggerCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command with no request err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-033 to FR-038, FR-043, FR-055, FR-022 - a zone that follows its caster refuses a drift, a stationary zone refuses every move; a zone the caster moves 30 feet with a Bonus Action on later turns only: refused on the cast turn, on another creature's turn, with the wrong action, 31 feet, and after a creature moved into it stopped it, but accepted otherwise; a drifting cloud reports its drift only on the caster's turn start and drifts no more than its distance; moves on an ended zone or before the cast are refused or stale.
// Path: unhappy (rule interaction).
func TestPersistentSpellZonesMovementScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	body := []content.ZoneRegionID{content.DefaultZoneRegion}
	cast := grpATurn(2, "wizard")
	later := grpATurn(3, "wizard")
	rule := []content.ZoneRuleFact{grpAZoneRule(content.CreatureEndsTurnInside, content.AnyCreature, false)}
	move := func(z combat.PersistentZone, mv combat.ZoneMove, turn combat.TurnMarker) combat.MovePersistentZoneResult {
		t.Helper()
		res, err := combat.NewMovePersistentZone().Run(ctx, combat.MovePersistentZoneCommand{Request: combat.MoveZoneRequest{Zone: z, Move: mv, Turn: turn}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	startOfTurn := func(z combat.PersistentZone, turn combat.TurnMarker) combat.MovePersistentZoneResult {
		t.Helper()
		res, err := combat.NewMovePersistentZone().Run(ctx, combat.MovePersistentZoneCommand{Request: combat.StartOfTurnRequest{Zone: z, Turn: turn}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	caster := func(feet int, how content.ZoneActionKind) combat.CasterMove {
		return grpAMust(combat.NewCasterMove(feet, how))
	}

	sphere := grpAZone(t, grpAZoneCommand("sphere", "wizard", grpAZoneSpell("sphere_spell", &content.ZoneFactsFields{
		Rules: rule, Mobility: grpAMust(content.NewMovedByCaster(30, content.BonusAction, true)), StopsMoverOnContact: true}), cast))
	for name, c := range map[string]struct {
		mv   combat.ZoneMove
		turn combat.TurnMarker
		want combat.MoveRefusal
	}{
		"on the cast turn":           {caster(10, content.BonusAction), cast, combat.RefusalNotOnLaterTurn},
		"on another creature's turn": {caster(10, content.BonusAction), grpATurn(3, "orc"), combat.RefusalNotCastersTurn},
		"with the wrong action":      {caster(10, content.MagicAction), later, combat.RefusalWrongActionKind},
		"31 feet":                    {caster(31, content.BonusAction), later, combat.RefusalExceedsMaxDistance},
		"a drift on a caster zone":   {grpAMust(combat.NewDriftMove(5)), later, combat.RefusalNotMovable},
	} {
		if got, ok := move(sphere, c.mv, c.turn).(combat.MoveZoneRefused); !ok || got.Reason != c.want {
			t.Errorf("%s: result = %#v, want refused %v", name, got, c.want)
		}
	}
	moved, ok := move(sphere, caster(30, content.BonusAction), later).(combat.MoveZoneMoved)
	if !ok || moved.Event.Kind() != "zone_moved" || moved.Event.DistanceFeet() != 30 || moved.Event.How() != content.BonusAction {
		t.Fatalf("a 30-foot move on a later turn = %#v, want moved with one zone_moved", moved)
	}
	if got := move(sphere, caster(10, content.BonusAction), grpATurn(1, "wizard")); got != combat.MovePersistentZoneResult(combat.MoveZoneStaleTurn{}) {
		t.Errorf("a turn before the cast = %#v, want stale", got)
	}
	// A creature moved into the sphere stops it for this turn only; the next turn's start clears it.
	hit, err := combat.NewAssessPersistentZoneTrigger().Run(ctx, combat.AssessPersistentZoneTriggerCommand{Request: combat.AssessTrigger{
		Zone: sphere, Turn: later, Event: combat.NewZoneMovedIntoSpace(combat.NewZoneCreatureFacts(id("orc"), body, true, true))}})
	if err != nil {
		t.Fatal(err)
	}
	stopped := hit.(combat.AssessZoneTriggerDue)
	if !stopped.ZoneStoppedForTurn {
		t.Fatalf("a creature moved into the zone = %#v, want the zone stopped for the turn", stopped)
	}
	if got, ok := move(stopped.Zone, caster(10, content.BonusAction), later).(combat.MoveZoneRefused); !ok || got.Reason != combat.RefusalStoppedForTurn {
		t.Errorf("moving a stopped zone = %#v, want refused StoppedForTurn", got)
	}
	next := grpATurn(4, "wizard")
	cleared, ok := startOfTurn(stopped.Zone, next).(combat.StartOfTurnReported)
	if !ok || cleared.Drifts {
		t.Fatalf("the next turn's start = %#v, want reported with no drift", cleared)
	}
	if _, ok := move(cleared.Zone, caster(10, content.BonusAction), next).(combat.MoveZoneMoved); !ok {
		t.Errorf("a zone whose stop was cleared by the next turn must move")
	}

	following := grpAZone(t, grpAZoneCommand("guard", "wizard", grpAZoneSpell("following", &grpAGuardians), cast))
	if got, ok := move(following, grpAMust(combat.NewDriftMove(5)), later).(combat.MoveZoneRefused); !ok || got.Reason != combat.RefusalMovesWithOrigin {
		t.Errorf("a drift of a zone that follows its caster = %#v, want refused MovesWithOrigin", got)
	}
	still := grpAZone(t, grpAZoneCommand("trap", "wizard", grpAZoneSpell("stationary", &content.ZoneFactsFields{Rules: rule}), cast))
	if got, ok := move(still, caster(5, content.BonusAction), later).(combat.MoveZoneRefused); !ok || got.Reason != combat.RefusalNotMovable {
		t.Errorf("a stationary zone = %#v, want refused NotMovable", got)
	}

	cloud := grpAZone(t, grpAZoneCommand("cloud", "wizard", grpAZoneSpell("drifting", &content.ZoneFactsFields{
		Rules: rule, Mobility: grpAMust(content.NewDriftsAtTurnStart(10, content.AwayFromCaster))}), cast))
	if r, ok := startOfTurn(cloud, later).(combat.StartOfTurnReported); !ok || !r.Drifts || r.Drift.Feet() != 10 || r.Drift.Direction() != content.AwayFromCaster || r.Drift.ZoneID() != cloud.ID() {
		t.Errorf("the caster's turn start = %#v, want a drift of 10 feet away from the caster", r)
	}
	if r, ok := startOfTurn(cloud, grpATurn(3, "orc")).(combat.StartOfTurnReported); !ok || r.Drifts {
		t.Errorf("another creature's turn start = %#v, want no drift", r)
	}
	if got := startOfTurn(cloud, grpATurn(1, "wizard")); got != combat.MovePersistentZoneResult(combat.StartOfTurnStaleTurn{}) {
		t.Errorf("a turn start before the cast = %#v, want stale", got)
	}
	if _, ok := move(cloud, grpAMust(combat.NewDriftMove(10)), later).(combat.MoveZoneMoved); !ok {
		t.Errorf("a drift of exactly its distance must be accepted")
	}
	if got, ok := move(cloud, grpAMust(combat.NewDriftMove(11)), later).(combat.MoveZoneRefused); !ok || got.Reason != combat.RefusalExceedsMaxDistance {
		t.Errorf("a drift of 11 feet = %#v, want refused ExceedsMaxDistance", got)
	}
	ended := combat.EndZone(cloud, combat.ZoneEndDispersed).(combat.EndZoneEnded).Zone
	if got, ok := move(ended, grpAMust(combat.NewDriftMove(5)), later).(combat.MoveZoneRefused); !ok || got.Reason != combat.RefusalZoneHasEnded {
		t.Errorf("moving an ended zone = %#v, want refused ZoneHasEnded", got)
	}
	if _, err := combat.NewMovePersistentZone().Run(ctx, combat.MovePersistentZoneCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command with no request err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewDriftMove(-1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative drift err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-040, FR-041, FR-042, FR-053 - a Guardian-of-Faith-like zone with a 60-point budget takes 30 and 30 and ends spent on the second, with one zone_ended; a zone with no budget refuses the record; a negative amount is invalid input; a 10-round zone cast in round 1 is still active in round 10 and ends elapsed in round 11, a host-timed zone never does, a round before the cast is invalid input; ending a zone twice is refused and an undefined reason is invalid input.
// Path: unhappy (boundary).
func TestPersistentSpellZonesEndingScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cast := grpATurn(1, "cleric")
	enters := []content.ZoneRuleFact{grpAZoneRule(content.CreatureEnters, content.EnemiesOfCaster, false)}
	run := func(req combat.ZoneEndRequest) (combat.EndPersistentZoneResult, error) {
		return combat.NewEndPersistentZone().Run(ctx, combat.EndPersistentZoneCommand{Request: req})
	}

	guardian := grpAZone(t, grpAZoneCommand("guardian", "cleric", grpAZoneSpell("guardian_of_faith", &content.ZoneFactsFields{Rules: enters, DamageBudget: 60}), cast))
	first, err := run(combat.RecordDamageRequest{Zone: guardian, Amount: 30})
	if err != nil {
		t.Fatal(err)
	}
	soaked, ok := first.(combat.RecordDamageRecorded)
	if !ok || soaked.Zone.Ended() || soaked.Zone.DamageDealt() != 30 || len(soaked.Events) != 0 {
		t.Fatalf("30 of 60 = %#v, want recorded, still active, no event", first)
	}
	second, err := run(combat.RecordDamageRequest{Zone: soaked.Zone, Amount: 30})
	if err != nil {
		t.Fatal(err)
	}
	spent, ok := second.(combat.RecordDamageRecorded)
	if !ok || !spent.Zone.Ended() || spent.Zone.EndReason() != combat.ZoneEndBudgetSpent || len(spent.Events) != 1 || spent.Events[0].Kind() != "zone_ended" {
		t.Fatalf("30 more = %#v, want the zone ended with BudgetSpent and one zone_ended", second)
	}
	if got, _ := run(combat.RecordDamageRequest{Zone: spent.Zone, Amount: 1}); got != combat.EndPersistentZoneResult(combat.RecordDamageZoneEnded{}) {
		t.Errorf("damage on an ended zone = %#v, want refused", got)
	}
	if _, err := run(combat.RecordDamageRequest{Zone: guardian, Amount: -1}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative amount err = %v, want ErrInvalidInput", err)
	}
	noBudget := grpAZone(t, grpAZoneCommand("plain", "cleric", grpAZoneSpell("no_budget", &content.ZoneFactsFields{Rules: enters}), cast))
	if got, _ := run(combat.RecordDamageRequest{Zone: noBudget, Amount: 5}); got != combat.EndPersistentZoneResult(combat.RecordDamageNoBudget{}) {
		t.Errorf("damage on a zone with no budget = %#v, want refused", got)
	}

	timed := grpAZoneCommand("timed", "cleric", grpAZoneSpell("ten_rounds", &content.ZoneFactsFields{Rules: enters}), cast)
	timed.DurationOverride = grpAMust(combat.NewRoundsDuration(10))
	tenRounds := grpAZone(t, timed)
	at10, err := run(combat.RoundStartedRequest{Zone: tenRounds, Round: 10})
	if err != nil {
		t.Fatal(err)
	}
	if r := at10.(combat.ZoneRoundResult); r.Zone.Ended() || len(r.Events) != 0 {
		t.Errorf("round 10 = %#v, want the zone still active", r)
	}
	at11, err := run(combat.RoundStartedRequest{Zone: tenRounds, Round: 11})
	if err != nil {
		t.Fatal(err)
	}
	if r := at11.(combat.ZoneRoundResult); !r.Zone.Ended() || r.Zone.EndReason() != combat.ZoneEndDurationElapsed || len(r.Events) != 1 || r.Events[0].Kind() != "zone_ended" {
		t.Errorf("round 11 = %#v, want ended DurationElapsed with one zone_ended", r)
	}
	if _, err := run(combat.RoundStartedRequest{Zone: tenRounds, Round: 0}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a round before the cast err = %v, want ErrInvalidInput", err)
	}
	hostTimed := grpAZoneCommand("host", "cleric", grpAZoneSpell("host_timed", &content.ZoneFactsFields{Rules: enters}), cast)
	hostTimed.DurationOverride = combat.HostTimedDuration{}
	if r, _ := run(combat.RoundStartedRequest{Zone: grpAZone(t, hostTimed), Round: 5000}); r.(combat.ZoneRoundResult).Zone.Ended() {
		t.Errorf("a host-timed zone must never end by the round count")
	}

	ended, err := run(combat.EndZoneRequest{Zone: guardian, Reason: combat.ZoneEndConcentrationEnded})
	if err != nil {
		t.Fatal(err)
	}
	e, ok := ended.(combat.EndZoneEnded)
	if !ok || !e.Zone.Ended() || e.Event.Reason() != combat.ZoneEndConcentrationEnded || e.Event.Kind() != "zone_ended" {
		t.Fatalf("losing Concentration = %#v, want ended with that reason", ended)
	}
	if again, _ := run(combat.EndZoneRequest{Zone: e.Zone, Reason: combat.ZoneEndDispelled}); again != combat.EndPersistentZoneResult(combat.EndZoneAlreadyEnded{}) {
		t.Errorf("ending a zone twice = %#v, want AlreadyEnded", again)
	}
	if _, err := run(combat.EndZoneRequest{Zone: guardian, Reason: 0}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined end reason err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewEndPersistentZone().Run(ctx, combat.EndPersistentZoneCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command with no request err = %v, want ErrInvalidInput", err)
	}
}
