package character_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var brdID = character.MustIdentifier("pc-1")

func brdPool(t *testing.T, maximum, remaining int) character.BardicInspirationPool {
	t.Helper()
	return must(character.NewBardicInspirationPool(maximum, remaining))
}

func brdGrant(t *testing.T, d character.BardicDie) *character.BardicInspirationGrant {
	t.Helper()
	g := must(character.NewBardicInspirationGrant(d))
	return &g
}

// Requirement: doc/specs/class-bard.md FR-026 to FR-029, FR-037, FR-121 (SRD 5.2.1, "Using Bardic Inspiration", p. 31) - granting expends one use and gives another creature a die of the Bard's current size; the Bard itself (or a recipient that is
// not an Other) is refused with self_target, no uses left with no_uses_remaining, and a recipient already holding a die with recipient_already_inspired, which expends nothing; the zero level is not_a_bard; the maximum is first recomputed from the
// Charisma modifier; the input pool is unchanged.
// Path: per case (happy / unhappy: invalid input, state error, boundary).
func TestGrantBardicInspiration(t *testing.T) {
	t.Parallel()
	other := character.InspirationRecipientOther{}
	cases := []struct {
		name      string
		level     int
		pool      character.BardicInspirationPool
		charisma  int
		recipient character.InspirationRecipient
		reason    character.BardReason
		die       character.BardicDie
		remaining int
	}{
		{"level 1 d6", 1, brdPool(t, 3, 3), 16, other, 0, character.BardicD6, 2},
		{"level 5 d8", 5, brdPool(t, 3, 3), 16, other, 0, character.BardicD8, 2},
		{"level 10 d10", 10, brdPool(t, 3, 2), 16, other, 0, character.BardicD10, 1},
		{"level 15 d12, last use", 15, brdPool(t, 3, 1), 16, other, 0, character.BardicD12, 0},
		{"the Bard itself", 5, brdPool(t, 3, 3), 16, character.InspirationRecipientSelf{}, character.BardSelfTarget, 0, 0},
		{"a nil recipient", 5, brdPool(t, 3, 3), 16, nil, character.BardSelfTarget, 0, 0},
		{"no uses remaining", 5, brdPool(t, 3, 0), 16, other, character.BardNoUsesRemaining, 0, 0},
		{"recipient already inspired", 5, brdPool(t, 3, 3), 16, character.InspirationRecipientOther{CurrentGrant: brdGrant(t, character.BardicD6)}, character.BardRecipientAlreadyInspired, 0, 0},
		{"a zero held grant is no grant", 5, brdPool(t, 3, 3), 16, character.InspirationRecipientOther{CurrentGrant: &character.BardicInspirationGrant{}}, 0, character.BardicD8, 2},
		{"stale maximum shrinks to the Charisma modifier", 5, brdPool(t, 3, 3), 10, other, 0, character.BardicD8, 0},
		{"not a Bard", 0, brdPool(t, 3, 3), 16, other, character.BardNotABard, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			level := character.BardLevel{}
			if c.level > 0 {
				level = character.MustBardLevel(c.level)
			}
			before := c.pool
			got, reason := character.GrantBardicInspiration(brdID, level, c.pool, brdMod(t, c.charisma), c.recipient)
			if reason != c.reason || c.pool != before {
				t.Fatalf("reason = %v, want %v (input pool changed: %v)", reason, c.reason, c.pool != before)
			}
			if reason != 0 {
				if got.Grant().Die() != 0 || got.Pool() != (character.BardicInspirationPool{}) {
					t.Errorf("a refused grant returned %+v", got)
				}
				return
			}
			if got.Grant().Die() != c.die || got.Pool().Remaining() != c.remaining || got.Event().Die() != c.die || got.Event().Remaining() != c.remaining || got.Event().CharacterID() != brdID {
				t.Errorf("grant %v, pool %d, event %v/%d; want %v and %d left", got.Grant().Die(), got.Pool().Remaining(), got.Event().Die(), got.Event().Remaining(), c.die, c.remaining)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-038 to FR-040, FR-043 to FR-045, FR-108 (SRD 5.2.1, "Number of Uses", "Level 5: Font of Inspiration", "Level 18: Superior Inspiration", pp. 32-33) - a Long Rest restores every use at any level; a Short
// Rest restores every use from Bard level 5 and none at 4; rolling Initiative restores uses until two remain from level 18 and none at 17, never lowers a pool of two or more and stops at a maximum below two; no recovery rises above the maximum; a
// recovery that changes nothing is nothing_to_restore.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestRecoverBardicInspiration(t *testing.T) {
	t.Parallel()
	long, short, initiative := character.BardRecoveryRest{Kind: character.LongRest}, character.BardRecoveryRest{Kind: character.ShortRest}, character.BardRecoveryInitiative{}
	cases := []struct {
		name     string
		level    int
		pool     character.BardicInspirationPool
		charisma int
		trigger  character.BardRecoveryTrigger
		reason   character.BardReason
		after    int
	}{
		{"Long Rest at level 1", 1, brdPool(t, 3, 0), 16, long, 0, 3},
		{"Long Rest at level 20", 20, brdPool(t, 3, 1), 16, long, 0, 3},
		{"Long Rest on a full pool", 5, brdPool(t, 3, 3), 16, long, character.BardNothingToRestore, 0},
		{"Short Rest at level 4", 4, brdPool(t, 3, 0), 16, short, character.BardBelowRequiredLevel, 0},
		{"Short Rest at level 5", 5, brdPool(t, 3, 0), 16, short, 0, 3},
		{"Short Rest on a full pool", 5, brdPool(t, 3, 3), 16, short, character.BardNothingToRestore, 0},
		{"Initiative at level 17", 17, brdPool(t, 4, 0), 18, initiative, character.BardBelowRequiredLevel, 0},
		{"Initiative at level 18, 0 to 2", 18, brdPool(t, 4, 0), 18, initiative, 0, 2},
		{"Initiative at level 18, 1 to 2", 18, brdPool(t, 4, 1), 18, initiative, 0, 2},
		{"Initiative at level 18, 2 stays", 18, brdPool(t, 4, 2), 18, initiative, character.BardNothingToRestore, 0},
		{"Initiative at level 18, 3 stays", 18, brdPool(t, 4, 3), 18, initiative, character.BardNothingToRestore, 0},
		{"Initiative stops at a maximum of 1", 18, brdPool(t, 1, 0), 10, initiative, 0, 1},
		{"stale maximum shrinks first", 5, brdPool(t, 5, 5), 10, long, character.BardNothingToRestore, 0},
		{"not a Bard, rest", 0, brdPool(t, 3, 0), 16, long, character.BardNotABard, 0},
		{"not a Bard, Initiative", 0, brdPool(t, 3, 0), 16, initiative, character.BardNotABard, 0},
		{"a nil trigger", 5, brdPool(t, 3, 0), 16, nil, character.BardBelowRequiredLevel, 0},
		{"an unknown rest", 5, brdPool(t, 3, 0), 16, character.BardRecoveryRest{}, character.BardBelowRequiredLevel, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			level := character.BardLevel{}
			if c.level > 0 {
				level = character.MustBardLevel(c.level)
			}
			got, reason := character.RecoverBardicInspiration(brdID, level, c.pool, brdMod(t, c.charisma), c.trigger)
			if reason != c.reason {
				t.Fatalf("reason = %v, want %v", reason, c.reason)
			}
			if reason != 0 {
				return
			}
			regained := got.Pool().Remaining() - c.pool.Remaining()
			if got.Pool().Remaining() != c.after || got.Event().Remaining() != c.after || got.Event().Uses() != regained || got.Event().CharacterID() != brdID || got.Pool().Remaining() > got.Pool().Maximum() {
				t.Errorf("pool %d of %d, event %d/%d; want %d left", got.Pool().Remaining(), got.Pool().Maximum(), got.Event().Uses(), got.Event().Remaining(), c.after)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-041, FR-042, FR-109, FR-124 (SRD 5.2.1, "Level 5: Font of Inspiration", p. 32) - from Bard level 5 expending a spell slot through the SpellcastingState regains one use, forwards the SpellSlotExpended event and
// adds BardicInspirationRestored; below level 5, with nothing to restore (checked before any slot is spent) or with no slot of that level it is refused with the pool and state unchanged and the SlotOperationRejection reported; slot levels 1 and 9
// work, 0 and 10 are refused by NewSlotLevel.
// Path: per case (happy / unhappy: boundary, state error).
func TestExpendSlotForInspiration(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	bard := func(n int) character.SpellcastingState { return castingState(t, classLevelsOf(t, "bard", n), spells) }
	cases := []struct {
		name     string
		level    int
		state    character.SpellcastingState
		pool     character.BardicInspirationPool
		slotPool character.SpellSlotPool
		slot     int
		reason   character.BardReason
		slotRej  character.SlotOperationRejection
	}{
		{"level 1 slot at level 5", 5, bard(5), brdPool(t, 3, 1), character.PoolSpellcasting, 1, 0, 0},
		{"level 9 slot at level 17", 17, bard(17), brdPool(t, 3, 0), character.PoolSpellcasting, 9, 0, 0},
		{"below level 5", 4, bard(4), brdPool(t, 3, 1), character.PoolSpellcasting, 1, character.BardBelowRequiredLevel, 0},
		{"nothing to restore", 5, bard(5), brdPool(t, 3, 3), character.PoolSpellcasting, 1, character.BardNothingToRestore, 0},
		{"no slot of that level", 5, bard(5), brdPool(t, 3, 1), character.PoolSpellcasting, 4, character.BardSlotRejected, character.SlotOperationNoSlotAvailable},
		{"the Pact Magic pool of a Bard", 5, bard(5), brdPool(t, 3, 1), character.PoolPactMagic, 1, character.BardSlotRejected, character.SlotOperationNoSlotAvailable},
		{"not a Bard", 0, bard(5), brdPool(t, 3, 1), character.PoolSpellcasting, 1, character.BardNotABard, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			level := character.BardLevel{}
			if c.level > 0 {
				level = character.MustBardLevel(c.level)
			}
			got, reason, slotRej := character.ExpendSlotForInspiration(brdID, level, c.pool, brdMod(t, 16), c.state, c.slotPool, slotLevel(c.slot))
			if reason != c.reason || slotRej != c.slotRej {
				t.Fatalf("reason %v / %v, want %v / %v", reason, slotRej, c.reason, c.slotRej)
			}
			if reason != 0 {
				if got.Pool() != (character.BardicInspirationPool{}) || len(got.Events()) != 0 || !got.State().IsZero() {
					t.Errorf("a refused expenditure returned %+v", got)
				}
				return
			}
			ev := got.Events()
			if got.Pool().Remaining() != c.pool.Remaining()+1 || remaining(got.State(), c.slotPool, c.slot) != remaining(c.state, c.slotPool, c.slot)-1 || len(ev) != 2 {
				t.Fatalf("pool %d, slots %d, %d events", got.Pool().Remaining(), remaining(got.State(), c.slotPool, c.slot), len(ev))
			}
			if _, ok := ev[0].(character.SpellSlotExpended); !ok {
				t.Errorf("first event = %T, want SpellSlotExpended", ev[0])
			}
			if r, ok := ev[1].(character.BardicInspirationRestored); !ok || r.Uses() != 1 || r.Remaining() != got.Pool().Remaining() {
				t.Errorf("second event = %#v, want BardicInspirationRestored of 1", ev[1])
			}
			if remaining(c.state, c.slotPool, c.slot) == 0 {
				t.Error("the input state was altered")
			}
		})
	}
	for _, n := range []int{0, 10} {
		if _, err := character.NewSlotLevel(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewSlotLevel(%d) error = %v, want ErrInvalidInput", n, err)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-030 to FR-035, FR-104, FR-116 (SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32; "Rolling 20 or 1", p. 7) - the using-a-die table: the number rolled is added to the total of the failed test through adjustedBy,
// the die is expended whether or not the test then succeeds, a natural 1 attack stays a miss, a natural 1 ability check uses the total; a test that was not failed is not_a_failure; exactly one hour is accepted and one hour and a tick is expired.
// Path: per case (happy / unhappy: boundary).
func TestPlanAndApplyBardicInspirationDie(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		die     character.BardicDie
		trigger character.AdjustableD20Result
		roll    int
		total   int
		want    d20.Outcome
	}{
		{"exactly reaches the DC", character.BardicD6, brdAbility(t, 14, 0, 15), 1, 15, d20.Success},
		{"one short of the DC", character.BardicD6, brdAbility(t, 12, 0, 15), 2, 14, d20.Failure},
		{"comfortably succeeds", character.BardicD6, brdAbility(t, 12, 0, 15), 6, 18, d20.Success},
		{"maximum d6 just enough", character.BardicD6, brdAbility(t, 9, 0, 15), 6, 15, d20.Success},
		{"d12 at level 15+", character.BardicD12, brdAbility(t, 3, 0, 15), 12, 15, d20.Success},
		{"minimum roll, still fails", character.BardicD8, brdAbility(t, 5, 0, 15), 1, 6, d20.Failure},
		{"attack roll against AC 14", character.BardicD6, brdAttack(t, 12, 0, 14), 2, 14, d20.Success},
		{"attack roll one short of AC 14", character.BardicD6, brdAttack(t, 12, 0, 14), 1, 13, d20.Failure},
		{"natural 1 attack stays a miss", character.BardicD6, brdAttack(t, 1, 0, 5), 6, 7, d20.Failure},
		{"natural 1 ability check uses the total", character.BardicD6, brdAbility(t, 1, 10, 15), 6, 17, d20.Success},
		{"a saving throw", character.BardicD6, brdSave(t, 10, 0, 15), 5, 15, d20.Success},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.trigger.Outcome() != d20.Failure {
				t.Fatalf("fixture: the original is %v, want a failure", c.trigger.Outcome())
			}
			plan, reason, err := character.PlanBardicInspirationUse(brdID, must(character.NewBardicInspirationGrant(c.die)), time.Hour, c.trigger)
			if err != nil || reason != 0 || plan.Die() != c.die {
				t.Fatalf("plan = %v, %v, %v; want the %v", plan, reason, err, c.die)
			}
			got, err := character.ApplyBardicInspirationDie(plan, c.roll)
			if err != nil || got.Result().Total() != c.total || got.Result().Outcome() != c.want {
				t.Fatalf("applied = total %d, %v (%v); want %d, %v", got.Result().Total(), got.Result().Outcome(), err, c.total, c.want)
			}
			ev := got.Event()
			if ev.Feature() != "bardic_inspiration" || ev.Die() != c.die || ev.Rolled() != c.roll || ev.CharacterID() != brdID {
				t.Errorf("event = %s %v %d", ev.Feature(), ev.Die(), ev.Rolled())
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-033 to FR-035, FR-116, FR-119 - planning refuses a test that was not failed (the die stays: not_a_failure), a die held more than one hour (expired), accepts exactly one hour and zero elapsed time, and returns an
// error matching ErrInvalidInput for a zero grant or a nil or zero test; applying refuses a number outside the die and a zero plan.
// Path: per case (unhappy: boundary, state error, invalid input).
func TestPlanBardicInspirationUseRefusals(t *testing.T) {
	t.Parallel()
	d6 := must(character.NewBardicInspirationGrant(character.BardicD6))
	failed, passed := brdAbility(t, 5, 0, 15), brdAbility(t, 18, 0, 15)
	for _, c := range []struct {
		name    string
		elapsed time.Duration
		trigger character.AdjustableD20Result
		reason  character.BardReason
	}{
		{"not failed", time.Minute, passed, character.BardNotAFailure},
		{"exactly one hour", time.Hour, failed, 0},
		{"one hour and a tick", time.Hour + time.Nanosecond, failed, character.BardExpired},
		{"no time elapsed", 0, failed, 0},
		{"a negative elapsed time", -time.Hour, failed, 0},
		{"not failed and expired", 2 * time.Hour, passed, character.BardNotAFailure},
	} {
		if _, reason, err := character.PlanBardicInspirationUse(brdID, d6, c.elapsed, c.trigger); err != nil || reason != c.reason {
			t.Errorf("%s: reason %v, error %v, want %v", c.name, reason, err, c.reason)
		}
	}
	for name, run := range map[string]func() error{
		"zero grant": func() error {
			_, _, e := character.PlanBardicInspirationUse(brdID, character.BardicInspirationGrant{}, 0, failed)
			return e
		},
		"nil trigger": func() error { _, _, e := character.PlanBardicInspirationUse(brdID, d6, 0, nil); return e },
		"zero trigger": func() error {
			_, _, e := character.PlanBardicInspirationUse(brdID, d6, 0, character.AdjustableCheck{})
			return e
		},
		"zero plan": func() error {
			_, e := character.ApplyBardicInspirationDie(character.BardicInspirationUsePlan{}, 1)
			return e
		},
		"roll too high": func() error {
			p, _, _ := character.PlanBardicInspirationUse(brdID, d6, 0, failed)
			_, e := character.ApplyBardicInspirationDie(p, 7)
			return e
		},
		"roll zero": func() error {
			p, _, _ := character.PlanBardicInspirationUse(brdID, d6, 0, failed)
			_, e := character.ApplyBardicInspirationDie(p, 0)
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := character.NewBardicInspirationGrant(7); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewBardicInspirationGrant(7) error = %v", err)
	}
}
