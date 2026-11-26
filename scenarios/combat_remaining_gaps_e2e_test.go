package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpGStatus is a creature at current of maximum Hit Points that is Alive (or Dying with no tallies when dying is set).
func grpGStatus(t *testing.T, current, maximum int, rule combat.ZeroHitPointsRule, dying bool) combat.HitPointStatus {
	t.Helper()
	var life combat.LifeState = combat.Alive{}
	if dying {
		life = combat.Dying{}
	}
	s, err := combat.RestoreHitPointStatus(combat.MustHitPoints(current, maximum, 0), life, rule, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-041, FR-042, FR-044 (SRD 5.2.1, "Death Ward", p. 122) - the events of a Death Ward guarded blow, in order: a blow that would drop the creature to 0 (exactly lethal, over lethal, or Massive Damage on a monster that dies instantly) gives damage_taken then death_ward_spent and leaves 1 Hit Point and Alive; a blow that leaves the creature above 0 and a blow to a creature already at 0 pass through with the ward still active; once the ward is spent the same blows drop the creature to 0 (fell unconscious), kill it by Massive Damage, or kill the monster outright.
// Path: per case.
func TestGrpGDeathWardEventOrder(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("corin")
	cases := []struct {
		name        string
		status      combat.HitPointStatus
		ward        combat.DeathWard
		damage      int
		wantEvents  []string
		wantCurrent int
		wantWard    combat.DeathWard
		check       func(t *testing.T, s combat.HitPointStatus)
	}{
		{"a blow that drops the creature to 0 spends the ward", grpGStatus(t, 5, 20, combat.FallsUnconscious, false), combat.WardActive, 10,
			[]string{"damage_taken", "death_ward_spent"}, 1, combat.WardSpent, grpGWantAlive},
		{"a blow of exactly the Hit Points left spends the ward", grpGStatus(t, 5, 20, combat.FallsUnconscious, false), combat.WardActive, 5,
			[]string{"damage_taken", "death_ward_spent"}, 1, combat.WardSpent, grpGWantAlive},
		{"Massive Damage on a monster that dies instantly is guarded", grpGStatus(t, 5, 20, combat.DiesInstantly, false), combat.WardActive, 60,
			[]string{"damage_taken", "death_ward_spent"}, 1, combat.WardSpent, grpGWantAlive},
		{"a blow that leaves Hit Points does not spend the ward", grpGStatus(t, 15, 20, combat.FallsUnconscious, false), combat.WardActive, 4,
			[]string{"damage_taken"}, 11, combat.WardActive, grpGWantAlive},
		{"damage to a creature already at 0 does not spend the ward", grpGStatus(t, 0, 20, combat.FallsUnconscious, true), combat.WardActive, 3,
			[]string{"damage_taken"}, 0, combat.WardActive, func(t *testing.T, s combat.HitPointStatus) {
				t.Helper()
				if d, ok := s.LifeState().(combat.Dying); !ok || d.Failures() != 1 {
					t.Errorf("life state = %#v, want Dying with one Death Saving Throw failure", s.LifeState())
				}
			}},
		{"a spent ward lets the blow drop the creature to 0", grpGStatus(t, 5, 20, combat.FallsUnconscious, false), combat.WardSpent, 10,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious"}, 0, combat.WardSpent, func(t *testing.T, s combat.HitPointStatus) {
				t.Helper()
				if _, ok := s.LifeState().(combat.Dying); !ok {
					t.Errorf("life state = %#v, want Dying", s.LifeState())
				}
			}},
		{"a spent ward lets Massive Damage kill", grpGStatus(t, 5, 20, combat.FallsUnconscious, false), combat.WardSpent, 60,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died"}, 0, combat.WardSpent, grpGWantDead(combat.MassiveDamage)},
		{"a spent ward lets a monster die at 0", grpGStatus(t, 5, 20, combat.DiesInstantly, false), combat.WardSpent, 10,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died"}, 0, combat.WardSpent, grpGWantDead(combat.MonsterDeath)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := combat.NewGuardDamageWithDeathWard().Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: id, Status: c.status, Ward: c.ward,
				Request: combat.WardDamage{Damages: []combat.Damage{combat.MustDamage(c.damage, content.DamageSlashing)}, Delivery: combat.OtherSource, Kind: combat.NormalHit}})
			if err != nil {
				t.Fatal(err)
			}
			a, ok := res.(combat.GuardedDamageApplied)
			if !ok {
				t.Fatalf("result = %#v, want GuardedDamageApplied", res)
			}
			if got := kinds(a.Events()); !slices.Equal(got, c.wantEvents) {
				t.Errorf("events = %v, want %v", got, c.wantEvents)
			}
			if hp := a.Status().HitPoints(); hp.Current() != c.wantCurrent || hp.Maximum() != 20 || a.Ward() != c.wantWard {
				t.Errorf("status = %+v ward %v, want %d of 20 and ward %v", hp, a.Ward(), c.wantCurrent, c.wantWard)
			}
			c.check(t, a.Status())
		})
	}
}

func grpGWantAlive(t *testing.T, s combat.HitPointStatus) {
	t.Helper()
	if s.LifeState() != (combat.Alive{}) {
		t.Errorf("life state = %#v, want Alive", s.LifeState())
	}
}

func grpGWantDead(cause combat.DeathCause) func(t *testing.T, s combat.HitPointStatus) {
	return func(t *testing.T, s combat.HitPointStatus) {
		t.Helper()
		if d, ok := s.LifeState().(combat.Dead); !ok || d.Cause() != cause {
			t.Errorf("life state = %#v, want Dead by %v", s.LifeState(), cause)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-076, FR-079, FR-078 (SRD 5.2.1, "Temporary Hit Points", p. 17) - Temporary Hit Points given through an encounter are refused for a Dead combatant (CombatantIsDead), an id outside the encounter (UnknownCombatant) and an ended encounter (EncounterAlreadyEnded, which wins when the combatant is also dead); each refusal returns the encounter unchanged, and a living combatant of the same encounter still gains them.
// Path: unhappy (state error).
func TestGrpGEncounterRefusesTemporaryHitPointsToTheWrongCombatant(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dead, err := combat.NewDead(combat.MonsterDeath)
	if err != nil {
		t.Fatal(err)
	}
	deadStatus, err := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 20, 0), dead, combat.DiesInstantly, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	e := grpBStartEncounter(t,
		grpBMember{"hero", combat.SideA, combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities()), combat.ConditionSet{}},
		grpBMember{"fallen", combat.SideA, deadStatus, combat.ConditionSet{}},
		grpBMember{"orc", combat.SideB, combat.MustHitPointStatus(15, combat.DiesInstantly, combat.NoAffinities()), combat.ConditionSet{}})
	give := func(enc combat.Encounter, who string) combat.GiveCombatantTemporaryHitPointsResult {
		t.Helper()
		cmd, err := combat.NewGiveCombatantTemporaryHitPointsCommand(enc, combat.MustCombatantID(who), 6, combat.AcceptNew)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewGiveCombatantTemporaryHitPoints().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	refused := func(enc combat.Encounter, who string, want combat.EncounterRejection) {
		t.Helper()
		r, ok := give(enc, who).(combat.GiveCombatantTemporaryHitPointsRejected)
		if !ok || r.Rejection != want || !r.Encounter().Equal(enc) {
			t.Errorf("giving to %s = %#v, want rejection %#v and the encounter unchanged", who, r, want)
		}
	}

	refused(e, "fallen", combat.CombatantIsDeadRejection{ID: combat.MustCombatantID("fallen")})
	refused(e, "ghost", combat.UnknownCombatantRejection{ID: combat.MustCombatantID("ghost")})
	if a, ok := give(e, "hero").(combat.GiveCombatantTemporaryHitPointsApplied); !ok || grpBFind(t, a.Encounter(), "hero").HitPoints().HitPoints().Temporary() != 6 {
		t.Errorf("a living combatant did not gain 6 temporary Hit Points")
	}

	end, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		t.Fatal(err)
	}
	er, err := combat.NewEndEncounter().Run(ctx, end)
	if err != nil {
		t.Fatal(err)
	}
	over := er.(combat.EndEncounterApplied).Encounter()
	refused(over, "hero", combat.EncounterAlreadyEndedRejection{})
	refused(over, "fallen", combat.EncounterAlreadyEndedRejection{})
	refused(over, "ghost", combat.EncounterAlreadyEndedRejection{})
}

// Requirement: doc/specs/grappling.md FR-049, FR-050, FR-088 and encounter-aggregate.md FR-073 (SRD 5.2.1, "Grappling", p. 182) - a hero grappled by an ogre is freed when the ogre becomes Incapacitated: the check at the range keeps the grapple until the Incapacitated condition is applied, then ends it with the cause grappler_incapacitated (also when the hero is beyond the range, because Incapacitated wins); a hero that is not grappled is refused.
// Path: unhappy (rule interaction).
func TestGrpGGrappleEndsWhenTheGrapplerIsIncapacitated(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	e := grpAEncounter(t, "hero", "ally", "ogre", "troll")
	grab, err := combat.NewApplyConditionToCombatantCommand(e, id("hero"), combat.Grappled{Grappler: id("ogre")})
	if err != nil {
		t.Fatal(err)
	}
	gr, err := combat.NewApplyConditionToCombatant().Run(ctx, grab)
	if err != nil {
		t.Fatal(err)
	}
	e = gr.(combat.ApplyConditionToCombatantApplied).Encounter()
	check := func(enc combat.Encounter, distance int) combat.CheckCombatantGrappleResult {
		t.Helper()
		cmd, err := combat.NewCheckCombatantGrappleCommand(enc, id("hero"), distance, 5)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewCheckCombatantGrapple().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if held, ok := check(e, 5).(combat.CheckCombatantGrappleApplied); !ok || len(held.Events()) != 0 {
		t.Fatalf("the ogre can still act: check = %#v, want applied with no event", held)
	}

	stun, err := combat.NewApplyConditionToCombatantCommand(e, id("ogre"), combat.Incapacitated{})
	if err != nil {
		t.Fatal(err)
	}
	sr, err := combat.NewApplyConditionToCombatant().Run(ctx, stun)
	if err != nil {
		t.Fatal(err)
	}
	e = sr.(combat.ApplyConditionToCombatantApplied).Encounter()
	for _, distance := range []int{5, 10} {
		freed, ok := check(e, distance).(combat.CheckCombatantGrappleApplied)
		if !ok {
			t.Fatalf("distance %d: check was refused", distance)
		}
		var ended []combat.GrappleEnded
		for _, ev := range freed.Events() {
			if w, isWrapped := ev.(combat.GrappleEventOccurred); isWrapped {
				if g, isEnd := w.Event().(combat.GrappleEnded); isEnd {
					ended = append(ended, g)
				}
			}
		}
		if len(ended) != 1 || ended[0].Cause() != combat.GrappleEndCauseGrapplerIncapacitated {
			t.Errorf("distance %d: events %v, want one grapple_ended with the cause grappler_incapacitated", distance, kinds(freed.Events()))
		}
		if m := grpBFind(t, freed.Encounter(), "hero"); slices.Contains(m.Conditions().Effective(), content.ConditionGrappled) {
			t.Errorf("distance %d: the hero is still Grappled", distance)
		}
	}
	e2 := check(e, 5).(combat.CheckCombatantGrappleApplied).Encounter()
	if _, ok := check(e2, 5).(combat.CheckCombatantGrappleRejected); !ok {
		t.Errorf("checking a hero that is no longer grappled must be refused")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-085, FR-088, FR-120, FR-126 and initiative-and-actions.md FR-042, FR-135 (SRD 5.2.1, "Changes to Your Speeds", p. 188; "Grappling", p. 182) - damage that drops a grappling ogre to 0 Hit Points makes it Unconscious, so the call prompts the host with movement_speeds_review_became_due for the ogre and then grapple_check_became_due for the hero; the host checks the grapple (it ends, grappler incapacitated) and sends the ogre's new Speed with AdvanceCombatTurn, which applies it as movement_speeds_updated and keeps it for the turns after; an update for a combatant that is not in the encounter is invalid and changes nothing, and an ended encounter refuses the advance.
// Path: happy, then unhappy (invalid input, state error).
func TestGrpGDamagePromptsSpeedReviewAndGrappleCheckThenTurnAdvanceAppliesTheSpeed(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	e := grpAEncounter(t, "hero", "ally", "ogre", "troll")
	grab, err := combat.NewApplyConditionToCombatantCommand(e, id("hero"), combat.Grappled{Grappler: id("ogre")})
	if err != nil {
		t.Fatal(err)
	}
	gr, err := combat.NewApplyConditionToCombatant().Run(ctx, grab)
	if err != nil {
		t.Fatal(err)
	}
	e = gr.(combat.ApplyConditionToCombatantApplied).Encounter()

	hit, err := combat.NewDealDamageToCombatantCommand(e, id("ogre"), []combat.Damage{combat.MustDamage(30, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	dd, err := combat.NewDealDamageToCombatant(dicetest.NewFixedRoller())
	if err != nil {
		t.Fatal(err)
	}
	dr, err := dd.Run(ctx, hit)
	if err != nil {
		t.Fatal(err)
	}
	burned := dr.(combat.DealDamageToCombatantCompleted)
	events := burned.Events()
	review, cue := slices.Index(kinds(events), "movement_speeds_review_became_due"), slices.Index(kinds(events), "grapple_check_became_due")
	if review < 0 || cue < 0 || review > cue {
		t.Fatalf("events = %v, want movement_speeds_review_became_due before grapple_check_became_due", kinds(events))
	}
	if got := events[review].Combatants(); !slices.Equal(got, []combat.CombatantID{id("ogre")}) {
		t.Errorf("review prompt names %v, want the ogre", got)
	}
	if got := events[cue].Combatants(); !slices.Equal(got, []combat.CombatantID{id("hero"), id("ogre")}) {
		t.Errorf("grapple cue names %v, want the hero then the ogre", got)
	}
	e = burned.Encounter()

	chk, err := combat.NewCheckCombatantGrappleCommand(e, id("hero"), 5, 5)
	if err != nil {
		t.Fatal(err)
	}
	cr, err := combat.NewCheckCombatantGrapple().Run(ctx, chk)
	if err != nil {
		t.Fatal(err)
	}
	freed := cr.(combat.CheckCombatantGrappleApplied)
	if !slices.Contains(kinds(freed.Events()), "grapple_ended") {
		t.Errorf("the host's check gave events %v, want grapple_ended", kinds(freed.Events()))
	}
	e = freed.Encounter()

	still, err := combat.NewMovementSpeeds(0)
	if err != nil {
		t.Fatal(err)
	}
	advance := func(enc combat.Encounter, updates ...combat.SpeedUpdate) combat.AdvanceCombatTurnResult {
		t.Helper()
		cmd, err := combat.NewAdvanceCombatTurnCommand(enc, combat.ActionLedger{}, updates, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewAdvanceCombatTurn().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	if p, _ := e.Tracker().ProfileOf(id("ogre")); p.Speeds.Walk() != 30 {
		t.Fatalf("before the update the ogre walks %d, want 30", p.Speeds.Walk())
	}
	done, ok := advance(e, combat.SpeedUpdate{Combatant: id("ogre"), Speeds: still}).(combat.AdvanceCombatTurnCompleted)
	if !ok || !slices.Contains(kinds(done.Events()), "movement_speeds_updated") {
		t.Fatalf("advance with the ogre's new Speed = %#v, want completed with movement_speeds_updated", done)
	}
	for turns := 0; turns < 3; turns++ {
		if p, _ := done.Encounter().Tracker().ProfileOf(id("ogre")); p.Speeds.Walk() != 0 {
			t.Errorf("after %d more turns the ogre walks %d, want 0 kept", turns, p.Speeds.Walk())
		}
		done = advance(done.Encounter()).(combat.AdvanceCombatTurnCompleted)
	}

	bad, ok := advance(e, combat.SpeedUpdate{Combatant: id("ghost"), Speeds: still}).(combat.AdvanceCombatTurnInvalid)
	if !ok || !bad.Encounter().Equal(e) {
		t.Errorf("an update for a combatant that is not in the encounter = %#v, want invalid and the encounter unchanged", bad)
	}
	end, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		t.Fatal(err)
	}
	er, err := combat.NewEndEncounter().Run(ctx, end)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := advance(er.(combat.EndEncounterApplied).Encounter()).(combat.AdvanceCombatTurnRejected); !ok {
		t.Errorf("advancing an ended encounter must be rejected")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-067, FR-068, FR-069, FR-070, FR-081 (SRD 5.2.1, "Breaking Objects", pp. 177-178) - a Gargantuan wall has no table row (split required, with no die rolled) while a Large section does: the host rolls each Large resilient section's 5d10, tracks the sections separately, and breaking one section (a further blow is refused as already destroyed) leaves the other untouched; every table row's Hit Points are its expression's average, and an undefined size or durability is invalid input.
// Path: happy, then unhappy (boundary, state error, invalid input).
func TestGrpGGargantuanWallIsSplitIntoSectionsTrackedSeparately(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	table := combat.ObjectHitPointsTable{}
	for _, size := range []content.CreatureSize{content.SizeHuge, content.SizeGargantuan} {
		got, err := table.Lookup(size, combat.ObjectResilient)
		if err != nil {
			t.Fatal(err)
		}
		if s, ok := got.(combat.ObjectHitPointsSplitRequired); !ok || s.Size != size {
			t.Errorf("lookup of %v = %#v, want split required for that size", size, got)
		}
	}
	for _, size := range []content.CreatureSize{content.SizeTiny, content.SizeSmall, content.SizeMedium, content.SizeLarge} {
		for _, durability := range []combat.ObjectDurability{combat.ObjectFragile, combat.ObjectResilient} {
			got, err := table.Lookup(size, durability)
			if err != nil {
				t.Fatal(err)
			}
			if f, ok := got.(combat.ObjectHitPointsFound); !ok || f.Entry.HitPoints() != f.Entry.Expression().Average() || f.Entry.HitPoints() < 1 {
				t.Errorf("lookup of %v %v = %#v, want a row whose Hit Points are the expression's average [FR-068]", size, durability, got)
			}
		}
	}
	if large, _ := table.Lookup(content.SizeLarge, combat.ObjectResilient); large.(combat.ObjectHitPointsFound).Entry.Expression() != dice.MustExpression(5, dice.D10(), 0) {
		t.Errorf("a Large resilient object rolls %v, want 5d10", large.(combat.ObjectHitPointsFound).Entry.Expression())
	}
	if _, err := table.Lookup(0, combat.ObjectResilient); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined size err = %v, want ErrInvalidInput", err)
	}
	if _, err := table.Lookup(content.SizeLarge, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined durability err = %v, want ErrInvalidInput", err)
	}

	// The whole wall cannot be rolled; no die is asked for.
	asked := 0
	fixed := dicetest.NewFixedRoller(2, 3, 4, 5, 6, 5, 5, 5, 5, 5)
	roller := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		asked++
		return fixed.Roll(e)
	})
	rolls, err := combat.NewRollObjectHitPoints(roller)
	if err != nil {
		t.Fatal(err)
	}
	whole, err := rolls.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeGargantuan, Durability: combat.ObjectResilient})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := whole.(combat.ObjectHitPointsRollSplitRequired); !ok || s.Size != content.SizeGargantuan || asked != 0 {
		t.Fatalf("the Gargantuan wall = %#v after %d rolls, want split required and no roll", whole, asked)
	}

	// Two Large sections, each rolled on its own: 2+3+4+5+6 = 20 and 5x5 = 25 Hit Points.
	section := func() combat.ObjectStatus {
		t.Helper()
		res, err := rolls.Run(ctx, combat.RollObjectHitPointsCommand{Size: content.SizeLarge, Durability: combat.ObjectResilient})
		if err != nil {
			t.Fatal(err)
		}
		s, err := combat.NewObjectStatus(content.SizeLarge, 17, res.(combat.ObjectHitPointsRolled).HitPoints(), combat.DamageThreshold{}, combat.ObjectAffinities{}.Standard(), nil)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	left, right := section(), section()
	if left.HitPoints().Current() != 20 || right.HitPoints().Current() != 25 {
		t.Fatalf("sections have %d and %d Hit Points, want 20 and 25", left.HitPoints().Current(), right.HitPoints().Current())
	}
	blow, err := combat.NewObjectDamageInstance([]combat.Damage{combat.MustDamage(22, content.DamageBludgeoning)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	hit := combat.NewApplyDamageToObject()
	broken, err := hit.Run(ctx, combat.ApplyDamageToObjectCommand{Status: left, Harm: blow})
	if err != nil {
		t.Fatal(err)
	}
	b := broken.(combat.ObjectDamageApplied)
	if !b.Status().IsDestroyed() || b.Events()[len(b.Events())-1].Kind() != "object_destroyed" {
		t.Errorf("a blow of 22 on the 20 Hit Point section = %#v, want it destroyed", b)
	}
	if right.HitPoints().Current() != 25 || right.IsDestroyed() {
		t.Errorf("the other section changed: %+v", right.HitPoints())
	}
	again, err := hit.Run(ctx, combat.ApplyDamageToObjectCommand{Status: b.Status(), Harm: blow})
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := again.(combat.ObjectDamageRejected); !ok || r.Rejection != combat.ObjectDamageRejectionAlreadyDestroyed {
		t.Errorf("a blow on the destroyed section = %#v, want already_destroyed", again)
	}
	next, err := hit.Run(ctx, combat.ApplyDamageToObjectCommand{Status: right, Harm: blow})
	if err != nil {
		t.Fatal(err)
	}
	if n := next.(combat.ObjectDamageApplied); n.Status().HitPoints().Current() != 3 || n.Status().IsDestroyed() {
		t.Errorf("the same blow on the 25 Hit Point section left %+v, want 3 left", n.Status().HitPoints())
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-037, FR-038, FR-039, FR-047 (SRD 5.2.1, "Mass Heal", p. 147; "Heal", p. 136) - Mass Heal has no multi-target use case: the host divides its 700 Hit Points and casts it once per creature with that creature's allotment; each call restores up to the creature's maximum (a Dying creature wakes), an allotment of 0 changes nothing, Blinded, Deafened and Poisoned end while Charmed stays, and a Dead creature or a negative allotment is refused or invalid; Heal also refuses a slot level outside 6 to 9.
// Path: happy, then unhappy (boundary, state error, invalid input).
func TestGrpGMassHealIsDividedByTheHostAcrossSeparateCasts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cast := func(c combat.CastRestorativeHealingCommand) (combat.CastRestorativeHealingResult, error) {
		t.Helper()
		return combat.NewCastRestorativeHealing().Run(ctx, c)
	}
	afflicted := combat.MustConditionSet([]combat.Condition{combat.Blinded{}, combat.Deafened{}, combat.Poisoned{}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: combat.MustCombatantID("witch")}}}, nil)
	type member struct {
		name      string
		status    combat.HitPointStatus
		allotment int
		wantHP    int
	}
	// 700 Hit Points of Mass Heal: 300 to the tank (only 90 missing), 250 to the scout (Dying, 45 maximum), 150 to the mage, 0 to the cook.
	party := []member{
		{"tank", grpGStatus(t, 10, 100, combat.FallsUnconscious, false), 300, 100},
		{"scout", grpGStatus(t, 0, 45, combat.FallsUnconscious, true), 250, 45},
		{"mage", grpGStatus(t, 40, 60, combat.FallsUnconscious, false), 150, 60},
		{"cook", grpGStatus(t, 7, 30, combat.FallsUnconscious, false), 0, 7},
	}
	for _, m := range party {
		res, err := cast(combat.CastRestorativeHealingCommand{Spell: combat.MassHeal, Subject: combat.MustCombatantID(m.name), Status: m.status, Set: afflicted, Amount: m.allotment, CastLevel: 9})
		if err != nil {
			t.Fatalf("%s: %v", m.name, err)
		}
		a, ok := res.(combat.RestorativeHealingApplied)
		if !ok {
			t.Fatalf("%s: result = %#v, want applied", m.name, res)
		}
		if a.Status().HitPoints().Current() != m.wantHP || (m.wantHP > 0 && a.Status().LifeState() != (combat.Alive{})) {
			t.Errorf("%s: %+v %v, want %d Hit Points and Alive", m.name, a.Status().HitPoints(), a.Status().LifeState(), m.wantHP)
		}
		left := a.Conditions().Effective()
		if m.allotment == 0 {
			if !a.Conditions().Equal(afflicted) || len(a.Events()) != 0 {
				t.Errorf("%s: an allotment of 0 changed %v with events %v, want nothing", m.name, left, kinds(a.Events()))
			}
			continue
		}
		if slices.Contains(left, content.ConditionBlinded) || slices.Contains(left, content.ConditionDeafened) || slices.Contains(left, content.ConditionPoisoned) || !slices.Contains(left, content.ConditionCharmed) {
			t.Errorf("%s: conditions left %v, want only Charmed", m.name, left)
		}
	}

	fallen := grpBDead(t, combat.MonsterDeath)
	if res, err := cast(combat.CastRestorativeHealingCommand{Spell: combat.MassHeal, Subject: combat.MustCombatantID("fallen"), Status: fallen, Amount: 50, CastLevel: 9}); err != nil {
		t.Fatal(err)
	} else if r, ok := res.(combat.RestorativeHealingRejected); !ok || r.Rejection != combat.CreatureIsDead {
		t.Errorf("Mass Heal on a Dead creature = %#v, want CreatureIsDead", res)
	}
	for _, level := range []int{5, 10} {
		res, err := cast(combat.CastRestorativeHealingCommand{Spell: combat.Heal, Subject: combat.MustCombatantID("tank"), Status: party[0].status, Amount: 70, CastLevel: level})
		if err != nil {
			t.Fatal(err)
		}
		if r, ok := res.(combat.RestorativeHealingInvalidLevel); !ok || r.CastLevel != level {
			t.Errorf("Heal at level %d = %#v, want InvalidLevel", level, res)
		}
	}
	if res, err := cast(combat.CastRestorativeHealingCommand{Spell: combat.MassHeal, Subject: combat.MustCombatantID("tank"), Status: party[0].status, Amount: -1, CastLevel: 9}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative allotment = %v, %v, want no result and ErrInvalidInput", res, err)
	}
}
