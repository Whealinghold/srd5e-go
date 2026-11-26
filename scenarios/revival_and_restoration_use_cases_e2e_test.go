package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/revival-and-restoration-spells.md FR-002, FR-007, FR-009, FR-010, FR-014, FR-018, FR-049 (SRD 5.2.1, "Raise Dead", p. 157) - ReviveCreatureBySpell returns a fallen ally with Raise Dead
// (1 Hit Point, a fresh ordeal of 4), reports every reason a cast past its 10-day window is ineligible, reports a refusing spirit as not revived, and refuses a creature that died of a maximum of 0 with no
// restored maximum.
// Path: happy and unhappy (boundary, state error).
func TestGrpBReviveCreatureBySpellUseCase(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("brenna")
	rec := telemetrytest.NewRecorder()
	uc := combat.NewReviveCreatureBySpell(combat.WithTelemetry(rec.Telemetry()))
	cast := func(c combat.ReviveBySpellCommand) combat.ReviveCreatureBySpellResult {
		t.Helper()
		res, err := uc.Run(ctx, combat.ReviveCreatureBySpellCommand(c))
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	base := combat.ReviveBySpellCommand{Spell: combat.RaiseDead, Subject: id, Status: grpBDead(t, combat.MassiveDamage), Consent: combat.SpiritAccepts,
		Circumstances: combat.MustDeathCircumstances(3*24*time.Hour, false, false, combat.Intact)}

	applied, ok := cast(base).(combat.ReviveBySpellApplied)
	if !ok {
		t.Fatalf("Raise Dead inside its window did not revive")
	}
	if hp := applied.Revival.Status().HitPoints(); hp.Current() != 1 || applied.Ordeal.Penalty() != 4 {
		t.Errorf("revived = %+v with an ordeal of %d, want 1 Hit Point and an ordeal of 4", hp, applied.Ordeal.Penalty())
	}

	late := base
	late.Circumstances = combat.MustDeathCircumstances(11*24*time.Hour, false, false, combat.Intact)
	if inel, isInel := cast(late).(combat.ReviveBySpellIneligible); !isInel || !slices.Contains(inel.Reasons(), combat.TimeWindowExceeded) {
		t.Errorf("a cast 11 days after death = %#v, want ineligible for TimeWindowExceeded", inel)
	}
	edge := base
	edge.Circumstances = combat.MustDeathCircumstances(10*24*time.Hour, false, false, combat.Intact)
	if _, isApplied := cast(edge).(combat.ReviveBySpellApplied); !isApplied {
		t.Errorf("a cast exactly 10 days after death was not accepted (the window is inclusive)")
	}

	refuses := base
	refuses.Consent = combat.SpiritRefuses
	if _, isNot := cast(refuses).(combat.ReviveBySpellNotRevived); !isNot {
		t.Errorf("a refusing spirit was not reported as not revived")
	}

	gone, err := combat.NewDead(combat.HitPointMaximumZero)
	if err != nil {
		t.Fatal(err)
	}
	zeroMax, err := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 0, 0), gone, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	noMax := base
	noMax.Status = zeroMax
	if nr, isNot := cast(noMax).(combat.ReviveBySpellNotRevived); !isNot {
		t.Errorf("a creature of maximum 0 with no restored maximum = %#v, want ReviveBySpellNotRevived", nr)
	} else if _, below := nr.Revival.(combat.ReviveCreatureMaximumBelowOne); !below {
		t.Errorf("the revival inside = %T, want ReviveCreatureMaximumBelowOne", nr.Revival)
	}
	none := 0
	noMax.RestoredMaximum = &none
	if _, isInvalid := cast(noMax).(combat.ReviveBySpellInvalidMaximum); !isInvalid {
		t.Errorf("a restored maximum of 0 was not refused as invalid")
	}
	lost := 20
	noMax.RestoredMaximum = &lost
	noMax.Spell = combat.Resurrection
	noMax.Circumstances = combat.MustDeathCircumstances(24*time.Hour, false, false, combat.Intact)
	if ra, isApplied := cast(noMax).(combat.ReviveBySpellApplied); !isApplied || ra.Revival.Status().HitPoints().Maximum() != 20 || ra.Revival.Status().HitPoints().Current() != 20 {
		t.Errorf("Resurrection with a GM-restored maximum of 20 = %#v, want 20 of 20", ra)
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-023, FR-026, FR-027, FR-028, FR-029, FR-031 (SRD 5.2.1, "Aid", p. 107) - Aid raises the maximum and current Hit Points by 10 for a creature at 0 Hit
// Points (waking it, Alive), a repeat of the same effect id is refused, ending Aid lowers the maximum again, ending an effect that is not held is refused, a reduction of the maximum is restored amount for amount,
// and a Dead creature refuses all of it.
// Path: happy and unhappy (rule interaction, state error, boundary).
func TestGrpBAidAndMaximumRestorationUseCases(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("corin")
	dying := grpBStatus(t, combat.Dying{}, 0, 20, 0)
	aid, err := combat.AidIncrease("aid-1", 3)
	if err != nil {
		t.Fatal(err)
	}

	inc, err := combat.NewIncreaseHitPointMaximum().Run(ctx, combat.IncreaseHitPointMaximumCommand{Subject: id, Status: dying, Increase: aid})
	if err != nil {
		t.Fatal(err)
	}
	aided := inc.(combat.IncreaseMaximumApplied)
	if hp := aided.Status().HitPoints(); hp.Maximum() != 30 || hp.Current() != 10 || aided.Status().LifeState() != (combat.Alive{}) {
		t.Fatalf("aided at 0 Hit Points = %+v %v, want 10 of 30 and Alive", hp, aided.Status().LifeState())
	}
	again, err := combat.NewIncreaseHitPointMaximum().Run(ctx, combat.IncreaseHitPointMaximumCommand{Subject: id, Status: aided.Status(), Ledger: aided.Ledger(), Increase: aid})
	if err != nil {
		t.Fatal(err)
	}
	if _, isDup := again.(combat.IncreaseMaximumDuplicate); !isDup {
		t.Errorf("the same Aid effect id twice = %#v, want IncreaseMaximumDuplicate", again)
	}
	deadInc, err := combat.NewIncreaseHitPointMaximum().Run(ctx, combat.IncreaseHitPointMaximumCommand{Subject: id, Status: grpBDead(t, combat.MonsterDeath), Increase: aid})
	if err != nil {
		t.Fatal(err)
	}
	if r, isRej := deadInc.(combat.IncreaseMaximumRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("Aid on a Dead creature = %#v, want CreatureIsDead", deadInc)
	}

	end := func(st combat.HitPointStatus, l combat.HitPointMaximumIncreases, effect combat.OngoingEffectID) combat.EndHitPointMaximumIncreaseResult {
		t.Helper()
		res, err := combat.NewEndHitPointMaximumIncrease().Run(ctx, combat.EndHitPointMaximumIncreaseCommand{Subject: id, Status: st, Ledger: l, EffectID: effect})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	ended := end(aided.Status(), aided.Ledger(), "aid-1").(combat.EndMaximumIncreaseApplied)
	if hp := ended.Status().HitPoints(); hp.Maximum() != 20 || hp.Current() != 10 {
		t.Errorf("after Aid ended = %+v, want 10 of 20", hp)
	}
	if _, isNot := end(ended.Status(), ended.Ledger(), "aid-1").(combat.EndMaximumIncreaseNotPresent); !isNot {
		t.Errorf("ending an Aid effect that is not held was not reported as NotPresent")
	}
	if r, isRej := end(grpBDead(t, combat.MonsterDeath), aided.Ledger(), "aid-1").(combat.EndMaximumIncreaseRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("ending Aid on a Dead creature = %#v, want CreatureIsDead", r)
	}

	full := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	cmd, err := combat.NewReduceHitPointMaximumCommand(id, full, 8)
	if err != nil {
		t.Fatal(err)
	}
	red, err := combat.NewReduceHitPointMaximum().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	reduced := red.(combat.HitPointMaximumReductionApplied).Status()
	res, err := combat.NewRestoreHitPointMaximum().Run(ctx, combat.RestoreHitPointMaximumCommand{Subject: id, Status: reduced, Amount: 8})
	if err != nil {
		t.Fatal(err)
	}
	if hp := res.(combat.RestoreMaximumApplied).Status().HitPoints(); hp.Maximum() != 20 {
		t.Errorf("after restoring 8 = %+v, want a maximum of 20", hp)
	}
	noop, err := combat.NewRestoreHitPointMaximum().Run(ctx, combat.RestoreHitPointMaximumCommand{Subject: id, Status: reduced, Amount: 0})
	if err != nil {
		t.Fatal(err)
	}
	if n, isApplied := noop.(combat.RestoreMaximumApplied); !isApplied || n.Status() != reduced || len(n.Events()) != 0 {
		t.Errorf("restoring 0 = %#v, want an applied no-op with no event", noop)
	}
	bad, err := combat.NewRestoreHitPointMaximum().Run(ctx, combat.RestoreHitPointMaximumCommand{Subject: id, Status: reduced, Amount: -3})
	if err != nil {
		t.Fatal(err)
	}
	if _, isBad := bad.(combat.RestoreMaximumInvalidAmount); !isBad {
		t.Errorf("restoring -3 = %#v, want RestoreMaximumInvalidAmount", bad)
	}
	deadRes, err := combat.NewRestoreHitPointMaximum().Run(ctx, combat.RestoreHitPointMaximumCommand{Subject: id, Status: grpBDead(t, combat.MonsterDeath), Amount: 8})
	if err != nil {
		t.Fatal(err)
	}
	if r, isRej := deadRes.(combat.RestoreMaximumRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("restoring a Dead creature = %#v, want CreatureIsDead", deadRes)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-032 (SRD 5.2.1, "Lesser Restoration", p. 143) - Lesser Restoration ends Blinded and leaves Poisoned, reports a held kind it cannot end, and reports a
// kind the creature does not hold as nothing to end.
// Path: happy and unhappy (invalid kind, empty case).
func TestGrpBLesserRestorationUseCase(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("lyra")
	set := combat.MustConditionSet([]combat.Condition{combat.Blinded{}, combat.Poisoned{}, combat.Stunned{}}, nil)
	cast := func(kind content.ConditionKind) combat.CastLesserRestorationResult {
		t.Helper()
		res, err := combat.NewCastLesserRestoration().Run(ctx, combat.CastLesserRestorationCommand{Subject: id, Set: set, Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	applied, ok := cast(content.ConditionBlinded).(combat.LesserRestorationApplied)
	if !ok {
		t.Fatalf("ending Blinded did not apply")
	}
	left := applied.Set().Effective()
	if slices.Contains(left, content.ConditionBlinded) || !slices.Contains(left, content.ConditionPoisoned) || len(applied.Events()) != 1 || applied.Events()[0].Kind() != "condition_removed" {
		t.Errorf("after ending Blinded: conditions %v events %v, want Poisoned kept and one condition_removed", left, kinds(applied.Events()))
	}
	if w, isWrong := cast(content.ConditionStunned).(combat.LesserRestorationWrongKind); !isWrong || w.Kind != content.ConditionStunned {
		t.Errorf("ending Stunned = %#v, want WrongKind(Stunned)", w)
	}
	if _, isNothing := cast(content.ConditionDeafened).(combat.LesserRestorationNothingToEnd); !isNothing {
		t.Errorf("ending Deafened that is not held was not reported as nothing to end")
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-033, FR-034, FR-035, FR-036 (SRD 5.2.1, "Greater Restoration", p. 137) - Greater Restoration takes one choice: it lowers Exhaustion by one level,
// ends a named curse, ends a Hit Point maximum reduction, leaves an ability score reduction to the host, and reports a choice with nothing to act on.
// Path: happy and unhappy (empty case, state error).
func TestGrpBGreaterRestorationUseCase(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("lyra")
	source, err := combat.LabeledMagicalSource(content.MustFeatureID("hag_curse"))
	if err != nil {
		t.Fatal(err)
	}
	curse, err := combat.NewOngoingEffect("curse-1", combat.OngoingEffectCurse, source)
	if err != nil {
		t.Fatal(err)
	}
	effects, err := combat.NewOngoingEffects([]combat.OngoingEffect{curse})
	if err != nil {
		t.Fatal(err)
	}
	exhausted := combat.MustConditionSet([]combat.Condition{combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(2)}}, nil)
	status := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	reducedStatus, err := status.ReduceMaximum(id, 8)
	if err != nil {
		t.Fatal(err)
	}
	reduced := reducedStatus.(combat.HitPointTransitionApplied).Status()
	cast := func(choice combat.GreaterRestorationChoice, set combat.ConditionSet, fx combat.OngoingEffects, st combat.HitPointStatus) combat.CastGreaterRestorationResult {
		t.Helper()
		res, err := combat.NewCastGreaterRestoration().Run(ctx, combat.CastGreaterRestorationCommand{Subject: id, Choice: choice, Conditions: set, Effects: fx, Status: st})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	level := cast(combat.EndOneExhaustionLevel{}, exhausted, effects, status).(combat.GreaterRestorationApplied)
	if got := level.Conditions().Explicit(); len(got) != 1 || got[0] != (combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)}) {
		t.Errorf("after one level off = %v, want Exhaustion level 1", got)
	}
	if _, isNothing := cast(combat.EndOneExhaustionLevel{}, combat.ConditionSet{}, effects, status).(combat.GreaterRestorationNothingToEnd); !isNothing {
		t.Errorf("one level off a creature with no Exhaustion was not reported as nothing to end")
	}

	cursed := cast(combat.EndCurse{ID: "curse-1"}, exhausted, effects, status).(combat.GreaterRestorationApplied)
	if cursed.Effects().Len() != 0 || cursed.CurseSource() != source {
		t.Errorf("after ending the curse: %d effects left, source %v, want none and the curse's own source", cursed.Effects().Len(), cursed.CurseSource())
	}
	if _, isNothing := cast(combat.EndCurse{ID: "missing"}, exhausted, effects, status).(combat.GreaterRestorationNothingToEnd); !isNothing {
		t.Errorf("ending a curse the creature does not hold was not reported as nothing to end")
	}

	healed := cast(combat.EndHitPointMaximumReduction{Amount: 8}, exhausted, effects, reduced).(combat.GreaterRestorationApplied)
	if got := healed.Status().HitPoints().Maximum(); got != 20 {
		t.Errorf("after ending the reduction = maximum %d, want 20", got)
	}
	if _, isHost := cast(combat.EndAbilityScoreReduction{}, exhausted, effects, status).(combat.GreaterRestorationHostResolved); !isHost {
		t.Errorf("an ability score reduction was not left to the host")
	}
	dead := grpBDead(t, combat.MonsterDeath)
	if r, isRej := cast(combat.EndHitPointMaximumReduction{Amount: 8}, exhausted, effects, dead).(combat.GreaterRestorationRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("a Dead creature = %#v, want CreatureIsDead", r)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-037, FR-038, FR-039 (SRD 5.2.1, "Heal", p. 136; "Power Word Heal", p. 154) - Heal restores the amount the host takes from the spell's data and ends
// Blinded, Deafened and Poisoned; Power Word Heal wakes a Dying creature at full Hit Points and ends Stunned; a Dead creature refuses.
// Path: happy and unhappy (state error).
func TestGrpBRestorativeHealingUseCase(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("lyra")
	cast := func(c combat.CastRestorativeHealingCommand) combat.CastRestorativeHealingResult {
		t.Helper()
		c.Subject = id
		res, err := combat.NewCastRestorativeHealing().Run(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	wounded, err := combat.RestoreHitPointStatus(combat.MustHitPoints(10, 100, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}

	heal := cast(combat.CastRestorativeHealingCommand{Spell: combat.Heal, Status: wounded, Amount: 70, CastLevel: 6,
		Set: combat.MustConditionSet([]combat.Condition{combat.Blinded{}, combat.Stunned{}}, nil)}).(combat.RestorativeHealingApplied)
	if got := heal.Status().HitPoints().Current(); got != 80 {
		t.Errorf("Heal = %d Hit Points, want 80", got)
	}
	if left := heal.Conditions().Effective(); slices.Contains(left, content.ConditionBlinded) || !slices.Contains(left, content.ConditionStunned) {
		t.Errorf("Heal left %v, want Blinded ended and Stunned kept", left)
	}

	dying := grpBStatus(t, combat.Dying{}, 0, 40, 0)
	word := cast(combat.CastRestorativeHealingCommand{Spell: combat.PowerWordHeal, Status: dying,
		Set: combat.MustConditionSet([]combat.Condition{combat.Stunned{}}, nil)}).(combat.RestorativeHealingApplied)
	if hp := word.Status().HitPoints(); hp.Current() != 40 || word.Status().LifeState() != (combat.Alive{}) || len(word.Conditions().Explicit()) != 0 {
		t.Errorf("Power Word Heal = %+v %v with %v, want 40 of 40, Alive and Stunned ended", hp, word.Status().LifeState(), word.Conditions().Explicit())
	}
	if r, isRej := cast(combat.CastRestorativeHealingCommand{Spell: combat.Heal, Status: grpBDead(t, combat.MonsterDeath), Amount: 70, CastLevel: 6}).(combat.RestorativeHealingRejected); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("Heal on a Dead creature = %#v, want CreatureIsDead", r)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-041, FR-042, FR-043, FR-044, FR-047 (SRD 5.2.1, "Death Ward", p. 122) - an active Death Ward turns a killing blow into 1 Hit Point and is spent
// with a death_ward_spent event; a spent ward lets the next blow through; an instant-kill effect spends an active ward and finds a spent one empty; a Dead creature is refused; a missing request is invalid input.
// Path: happy and unhappy (rule interaction, state error, invalid input).
func TestGrpBDeathWardUseCase(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID("corin")
	uc := combat.NewGuardDamageWithDeathWard()
	status, err := combat.RestoreHitPointStatus(combat.MustHitPoints(5, 20, 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	blow := combat.WardDamage{Damages: []combat.Damage{combat.MustDamage(30, content.DamageSlashing)}, Delivery: combat.OtherSource, Kind: combat.NormalHit}
	guard := func(st combat.HitPointStatus, ward combat.DeathWard, req combat.DeathWardRequest) combat.GuardDamageWithDeathWardResult {
		t.Helper()
		res, err := uc.Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: id, Status: st, Ward: ward, Request: req})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	saved, ok := guard(status, combat.WardActive, blow).(combat.GuardedDamageApplied)
	if !ok || saved.Status().HitPoints().Current() != 1 || saved.Status().LifeState() != (combat.Alive{}) || saved.Ward() != combat.WardSpent {
		t.Fatalf("a killing blow on a warded creature = %#v, want 1 Hit Point, Alive and the ward spent", saved)
	}
	if got := kinds(saved.Events()); got[len(got)-1] != "death_ward_spent" {
		t.Errorf("events = %v, want death_ward_spent last", got)
	}
	through := guard(saved.Status(), combat.WardSpent, blow).(combat.GuardedDamageApplied)
	if through.Status().LifeState() == (combat.Alive{}) || through.Ward() != combat.WardSpent {
		t.Errorf("a blow after the ward was spent left %v, want the creature to fall", through.Status().LifeState())
	}

	spent := guard(status, combat.WardActive, combat.WardInstantKill{}).(combat.GuardedDamageApplied)
	if spent.Ward() != combat.WardSpent || spent.Status() != status {
		t.Errorf("an instant kill on an active ward = ward %v status %+v, want the ward spent and the status unchanged", spent.Ward(), spent.Status().HitPoints())
	}
	if _, isNot := guard(status, combat.WardSpent, combat.WardInstantKill{}).(combat.GuardedDamageNotWarded); !isNot {
		t.Errorf("an instant kill on a spent ward was not reported as not warded")
	}
	if r, isRej := guard(grpBDead(t, combat.MonsterDeath), combat.WardActive, blow).(combat.GuardedDamageRefused); !isRej || r.Rejection != combat.CreatureIsDead {
		t.Errorf("a Dead creature = %#v, want CreatureIsDead", r)
	}
	if res, err := uc.Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: id, Status: status, Ward: combat.WardActive}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil request = %v, %v, want no result and ErrInvalidInput", res, err)
	}
	if res, err := uc.Run(ctx, combat.GuardDamageWithDeathWardCommand{Subject: id, Status: status, Ward: combat.DeathWard(9), Request: blow}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an unknown ward state = %v, %v, want no result and ErrInvalidInput", res, err)
	}
}
