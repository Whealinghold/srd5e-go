package combat_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// luDescribe renders entries as short strings, in order.
func luDescribe(entries []combat.LimitedUseEntry) []string {
	var out []string
	for _, e := range entries {
		switch v := e.(type) {
		case combat.PerDayEntry:
			lair, _ := v.MaximumInLair()
			out = append(out, fmt.Sprintf("perday %s/%d max=%d lair=%d left=%d", v.Part().Section(), v.Part().Index(), v.Maximum(), lair, v.Remaining()))
		case combat.RechargeEntry:
			out = append(out, fmt.Sprintf("recharge %s/%d %d-%d %s", v.Part().Section(), v.Part().Index(), v.Low(), v.High(), v.Readiness()))
		case combat.RestRechargeEntry:
			out = append(out, fmt.Sprintf("rest %s/%d %s", v.Part().Section(), v.Part().Index(), v.Readiness()))
		case combat.SpellUseEntry:
			out = append(out, fmt.Sprintf("spell %d/%d max=%d left=%d", v.Spell().GroupIndex(), v.Spell().SpellIndex(), v.Maximum(), v.Remaining()))
		}
	}
	return out
}

// Requirement: doc/specs/monster-limited-use.md FR-001, FR-002, FR-003, FR-004 - a state holds one entry per limited part in printed order and none for an unlimited part or a usage the SRD gives no rule for; X/Day starts at the count for the lair status (the in-lair figure only when printed), Recharge parts start ready, Legendary Actions start full for the lair status (SRD 5.2.1, "Limited Usage", "Legendary Actions", p. 257; "Aboleth", p. 258).
// Path: per case.
func TestStartLimitedUseEntries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		monster                 content.Monster
		lair                    combat.LairStatus
		wantEntries             []string
		wantLegendary           combat.LegendaryTrack
	}{
		{"drake outside the lair", limitedUseSpec + "FR-001 FR-002 FR-003 FR-004", "happy", luDrake(t), combat.OutsideLair,
			[]string{"perday traits/0 max=3 lair=4 left=3", "rest traits/1 ready", "perday actions/1 max=2 lair=0 left=2", "recharge actions/2 5-6 ready",
				"recharge actions/3 6-6 ready", "perday reactions/0 max=1 lair=0 left=1"}, nil},
		{"drake in the lair", limitedUseSpec + "FR-002 FR-004", "happy", luDrake(t), combat.InLair,
			[]string{"perday traits/0 max=3 lair=4 left=4", "rest traits/1 ready", "perday actions/1 max=2 lair=0 left=2", "recharge actions/2 5-6 ready",
				"recharge actions/3 6-6 ready", "perday reactions/0 max=1 lair=0 left=1"}, nil},
		{"no limited part", limitedUseSpec + "FR-001 FR-004", "unhappy (empty and minimal)", luMonster(t, func(f *content.MonsterFields) {
			f.Actions = []content.StatBlockPart{luPart("Bite", content.PartUnlimited{})}
		}), combat.OutsideLair, nil, combat.LegendaryNone{}},
		{"limited legendary action part", limitedUseSpec + "FR-001", "unhappy (rule interaction)", luMonster(t, func(f *content.MonsterFields) {
			f.LegendaryActions = luMust(content.NewLegendaryActions(2, 0, []content.StatBlockPart{luPart("Burst", luPerDay(1, 0))}))
		}), combat.InLair, []string{"perday legendary_actions/0 max=1 lair=0 left=1"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := luStart(t, c.monster, c.lair)
			if got := luDescribe(s.Entries()); !slices.Equal(got, c.wantEntries) {
				t.Errorf("entries = %q, want %q [%s]", got, c.wantEntries, c.requirement)
			}
			if s.Lair() != c.lair || s.MonsterID() != "fixture_beast" || !slices.Equal(s.Combatants(), []combat.CombatantID{luSelf}) || s.IsZero() {
				t.Errorf("identity = %v %v %v, want lair %v, fixture_beast, [%s]", s.Lair(), s.MonsterID(), s.Combatants(), c.lair, luSelf)
			}
			if c.wantLegendary != nil && s.Legendary() != c.wantLegendary {
				t.Errorf("Legendary() = %#v, want %#v", s.Legendary(), c.wantLegendary)
			}
			if s.LastLegendaryWindow() != (combat.LegendaryWindowNoneTaken{}) {
				t.Errorf("LastLegendaryWindow() = %#v, want none taken", s.LastLegendaryWindow())
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-004, LA-1 - Legendary Actions start with all their uses: 3 outside the lair and 4 in it when printed ("3 (4 in Lair)"), and 3 in both when no in-lair figure is printed (SRD 5.2.1, "Legendary Actions", p. 257; "Aboleth", p. 258; "Solar", p. 327).
// Path: per case.
func TestStartLimitedUseLegendaryUses(t *testing.T) {
	t.Parallel()
	solar := luMonster(t, func(f *content.MonsterFields) { f.LegendaryActions = luLegendary(3, 0) })
	cases := []struct {
		name    string
		monster content.Monster
		lair    combat.LairStatus
		want    int
	}{
		{"3 (4 in Lair), outside", luDrake(t), combat.OutsideLair, 3},
		{"3 (4 in Lair), in the lair", luDrake(t), combat.InLair, 4},
		{"3 with no in-lair figure, in the lair", solar, combat.InLair, 3},
		{"3 with no in-lair figure, outside", solar, combat.OutsideLair, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := luLegendaryOf(t, luStart(t, c.monster, c.lair))
			if got.Maximum() != c.want || got.Remaining() != c.want {
				t.Errorf("Legendary = %d of %d, want %d of %d [FR-004]", got.Remaining(), got.Maximum(), c.want, c.want)
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-001, shared-types FR-070 - starting from a zero Monster (the missing-content case: a host that found no stat block), a zero combatant or an undefined lair status is invalid input.
// Path: unhappy (invalid input, missing content).
func TestStartLimitedUseRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	drake := luDrake(t)
	cases := []struct {
		name    string
		monster content.Monster
		id      combat.CombatantID
		lair    combat.LairStatus
	}{
		{"zero monster", content.Monster{}, luSelf, combat.InLair},
		{"zero combatant", drake, "", combat.InLair},
		{"zero lair", drake, luSelf, 0},
		{"undefined lair", drake, luSelf, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.StartLimitedUse(c.monster, c.id, c.lair)
			if !errors.Is(err, srderr.ErrInvalidInput) || !s.IsZero() {
				t.Errorf("StartLimitedUse = %v, %v, want the zero state and ErrInvalidInput", s, err)
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-006, FR-007, FR-009, FR-030, XD-1 - Spend reduces an X/Day entry by exactly 1 with one LimitedUsePartSpent event carrying the uses left, spends a ready Recharge or rest-recharge part (uses left 0), refuses a spent or zero-use part as exhausted and a reference with no entry as not limited, and a refusal changes nothing (SRD 5.2.1, "Limited Usage", p. 257).
// Path: per case.
func TestSpend(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spend                   []combat.LimitedUsePartRef // all but the last must succeed
		wantRemaining           int
		wantRefusal             combat.LimitedUseRefusal
	}{
		{"2/Day, first use", "FR-006 XD-1", "happy", []combat.LimitedUsePartRef{luMind(t)}, 1, 0},
		{"2/Day, second use", "FR-006 XD-1", "unhappy (boundary)", []combat.LimitedUsePartRef{luMind(t), luMind(t)}, 0, 0},
		{"2/Day, third use is one past the last", "FR-007 XD-1", "unhappy (boundary)", []combat.LimitedUsePartRef{luMind(t), luMind(t), luMind(t)}, 0, combat.LimitedUseRefusedExhausted},
		{"1/Day reaction, once", "FR-007 XD-1", "unhappy (boundary)", []combat.LimitedUsePartRef{luParry(t)}, 0, 0},
		{"1/Day reaction, twice", "FR-007 XD-1", "unhappy (boundary)", []combat.LimitedUsePartRef{luParry(t), luParry(t)}, 0, combat.LimitedUseRefusedExhausted},
		{"ready Recharge part", "FR-009", "happy", []combat.LimitedUsePartRef{luSpray(t)}, 0, 0},
		{"spent Recharge part", "FR-009", "unhappy (state error)", []combat.LimitedUsePartRef{luSpray(t), luSpray(t)}, 0, combat.LimitedUseRefusedExhausted},
		{"ready rest-recharge part", "FR-009", "happy", []combat.LimitedUsePartRef{luRestTrait(t)}, 0, 0},
		{"spent rest-recharge part", "FR-009", "unhappy (state error)", []combat.LimitedUsePartRef{luRestTrait(t), luRestTrait(t)}, 0, combat.LimitedUseRefusedExhausted},
		{"unlimited part", "FR-030", "unhappy (missing content)", []combat.LimitedUsePartRef{luBite(t)}, 0, combat.LimitedUseRefusedNotLimited},
		{"part with another usage", "FR-001 FR-030 XD-1", "unhappy (rule interaction)", []combat.LimitedUsePartRef{luRef(t, combat.MonsterPartActions, 4)}, 0, combat.LimitedUseRefusedNotLimited},
		{"position past the list", "FR-029 XD-1", "unhappy (boundary)", []combat.LimitedUsePartRef{luRef(t, combat.MonsterPartActions, 99)}, 0, combat.LimitedUseRefusedNotLimited},
		{"zero reference", "FR-030", "unhappy (invalid input)", []combat.LimitedUsePartRef{{}}, 0, combat.LimitedUseRefusedNotLimited},
		{"section the monster has no part in", "FR-030", "unhappy (empty and minimal)", []combat.LimitedUsePartRef{luRef(t, combat.MonsterPartBonusActions, 0)}, 0, combat.LimitedUseRefusedNotLimited},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement
			s := luStart(t, luDrake(t), combat.OutsideLair)
			last := c.spend[len(c.spend)-1]
			for _, p := range c.spend[:len(c.spend)-1] {
				s = luSpent(t, s, p)
			}
			before := s
			switch out := s.Spend(last).(type) {
			case combat.LimitedUseSpent:
				if c.wantRefusal != 0 {
					t.Fatalf("Spend was applied, want refusal %v [%s]", c.wantRefusal, req)
				}
				ev := out.Transition.Events()
				if len(ev) != 1 {
					t.Fatalf("events = %v, want one LimitedUsePartSpent [%s]", ev, req)
				}
				if e, ok := ev[0].(combat.LimitedUsePartSpent); !ok || e.Subject() != (combat.PartSubject{Part: last}) || e.Remaining() != c.wantRemaining {
					t.Errorf("event = %#v, want part %v with %d remaining [%s]", ev[0], last, c.wantRemaining, req)
				}
				if out.Transition.State().Equal(before) {
					t.Errorf("state did not change [%s]", req)
				}
				if got := before.AvailabilityOf(last); got != combat.UseAvailable {
					t.Errorf("AvailabilityOf before = %v, want available", got)
				}
				wantAfter := combat.UseAvailable
				if c.wantRemaining == 0 {
					wantAfter = combat.UseExhausted
				}
				if got := out.Transition.State().AvailabilityOf(last); got != wantAfter {
					t.Errorf("AvailabilityOf after = %v, want %v [FR-029]", got, wantAfter)
				}
			case combat.LimitedUseSpendRefused:
				if out.Reason != c.wantRefusal {
					t.Errorf("refusal = %v, want %v [%s]", out.Reason, c.wantRefusal, req)
				}
				if c.wantRefusal == combat.LimitedUseRefusedNotLimited && before.AvailabilityOf(last) != combat.UseNotLimited {
					t.Errorf("AvailabilityOf = %v, want not_limited [FR-029]", before.AvailabilityOf(last))
				}
				if c.wantRefusal == combat.LimitedUseRefusedExhausted && before.AvailabilityOf(last) != combat.UseExhausted {
					t.Errorf("AvailabilityOf = %v, want exhausted [FR-029]", before.AvailabilityOf(last))
				}
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-006 - spending X/Day uses one at a time down to zero reports the uses left in each event: 2, then 1, then 0 for a 3/Day part.
// Path: unhappy (boundary).
func TestSpendReportsUsesLeftInEachEvent(t *testing.T) {
	t.Parallel()
	s := luStart(t, luDrake(t), combat.OutsideLair)
	var left []int
	for range 3 {
		out := s.Spend(luRes(t)).(combat.LimitedUseSpent)
		left = append(left, out.Transition.Events()[0].(combat.LimitedUsePartSpent).Remaining())
		s = out.Transition.State()
	}
	if !slices.Equal(left, []int{2, 1, 0}) {
		t.Errorf("uses left per event = %v, want [2 1 0]", left)
	}
	if luPerDayOf(t, s, luRes(t)).Remaining() != 0 {
		t.Error("the entry was not reduced to 0")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-010, FR-011, RC-1 - a spent Recharge L-H part becomes ready for a 1d6 face from L to H, both ends inclusive, and stays spent otherwise; one MonsterRechargeRolled per roll records the face and the result (SRD 5.2.1, "Limited Usage", p. 257).
// Path: per case.
func TestStartOfOwnTurnRechargeRanges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		low, high, face         int
		wantReady               bool
	}{
		{"5-6, face 4", "RC-1", "unhappy (boundary)", 5, 6, 4, false},
		{"5-6, face 5", "RC-1", "unhappy (boundary)", 5, 6, 5, true},
		{"5-6, face 6", "RC-1", "happy", 5, 6, 6, true},
		{"5-6, face 1", "RC-1", "unhappy (boundary)", 5, 6, 1, false},
		{"4-6, face 3", "RC-1", "unhappy (boundary)", 4, 6, 3, false},
		{"4-6, face 4", "RC-1", "unhappy (boundary)", 4, 6, 4, true},
		{"6 only, face 5", "RC-1", "unhappy (boundary)", 6, 6, 5, false},
		{"6 only, face 6", "RC-1", "unhappy (boundary)", 6, 6, 6, true},
		{"widest range, face 1", "RC-1", "unhappy (boundary)", 1, 6, 1, true},
		{"single middle value, face 2", "RC-1", "unhappy (boundary)", 3, 3, 2, false},
		{"single middle value, face 3", "RC-1", "unhappy (boundary)", 3, 3, 3, true},
		{"single middle value, face 4", "RC-1", "unhappy (boundary)", 3, 3, 4, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement + " FR-011"
			m := luMonster(t, func(f *content.MonsterFields) {
				f.Actions = []content.StatBlockPart{luPart("Burst", luRecharge(c.low, c.high))}
			})
			part := luRef(t, combat.MonsterPartActions, 0)
			s := luSpent(t, luStart(t, m, combat.OutsideLair), part)
			if got := s.RechargeRollsNeeded(); !slices.Equal(got, []combat.LimitedUsePartRef{part}) {
				t.Fatalf("RechargeRollsNeeded = %v, want [%v] [FR-010]", got, part)
			}
			tr, err := s.StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{part: luRoll(c.face)})
			if err != nil {
				t.Fatalf("StartOfOwnTurn: %v", err)
			}
			wantReadiness, wantResult := combat.RechargeSpent, combat.RechargeNotRegained
			if c.wantReady {
				wantReadiness, wantResult = combat.RechargeReady, combat.RechargeRegained
			}
			if got := luReadiness(t, tr.State(), part); got != wantReadiness {
				t.Errorf("readiness = %v, want %v [%s]", got, wantReadiness, req)
			}
			ev := tr.Events()
			if len(ev) != 1 {
				t.Fatalf("events = %v, want one MonsterRechargeRolled", ev)
			}
			if e, ok := ev[0].(combat.MonsterRechargeRolled); !ok || e.Part() != part || e.Face() != c.face || e.Result() != wantResult {
				t.Errorf("event = %#v, want part %v face %d %v [%s]", ev[0], part, c.face, wantResult, req)
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-010, FR-013, FR-024, RC-1 - only spent Recharge L-H parts are rolled for: a ready part, a Recharge-after-a-Rest part and an X/Day part need no roll, and two spent parts roll in printed order with the events in that order (SRD 5.2.1, "Limited Usage", p. 257; "Cloaker", p. 274).
// Path: unhappy (rule interaction).
func TestRechargeRollsNeeded(t *testing.T) {
	t.Parallel()
	s := luStart(t, luDrake(t), combat.OutsideLair)
	if got := s.RechargeRollsNeeded(); len(got) != 0 {
		t.Errorf("fresh state needs %v, want no roll (ready parts) [FR-010]", got)
	}
	s = luSpent(t, luSpent(t, luSpent(t, luSpent(t, s, luRestTrait(t)), luMind(t)), luLash(t)), luSpray(t))
	want := []combat.LimitedUsePartRef{luSpray(t), luLash(t)}
	if got := s.RechargeRollsNeeded(); !slices.Equal(got, want) {
		t.Errorf("RechargeRollsNeeded = %v, want %v in printed order, without the rest part and the X/Day part", got, want)
	}
	tr, err := s.StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(5), luLash(t): luRoll(5)})
	if err != nil {
		t.Fatal(err)
	}
	if luReadiness(t, tr.State(), luSpray(t)) != combat.RechargeReady || luReadiness(t, tr.State(), luLash(t)) != combat.RechargeSpent {
		t.Error("two rolls of 5: want Recharge 5-6 ready and Recharge 6 spent [RC-1]")
	}
	if luReadiness(t, tr.State(), luRestTrait(t)) != combat.RechargeSpent {
		t.Error("a Recharge-after-a-Rest part must stay spent at the start of the turn [FR-013]")
	}
	if luPerDayOf(t, tr.State(), luMind(t)).Remaining() != 1 {
		t.Error("an X/Day part must not change at the start of the turn")
	}
	var parts []combat.LimitedUsePartRef
	for _, e := range tr.Events() {
		parts = append(parts, e.(combat.MonsterRechargeRolled).Part())
	}
	if !slices.Equal(parts, want) {
		t.Errorf("event parts = %v, want %v [FR-024]", parts, want)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-012, RC-1 - StartOfOwnTurn returns an error matching srderr.ErrInvalidInput unless the keys of the roll map are exactly the parts that need a roll and every roll is a d6; the state is not changed.
// Path: per case.
func TestStartOfOwnTurnRejectsBadRollMaps(t *testing.T) {
	t.Parallel()
	base := luSpent(t, luSpent(t, luStart(t, luDrake(t), combat.OutsideLair), luSpray(t)), luLash(t))
	one := luSpent(t, luStart(t, luDrake(t), combat.OutsideLair), luSpray(t))
	d8 := dice.MustDieRoll(dice.D8(), 5)
	cases := []struct {
		name  string
		state combat.LimitedUseState
		rolls map[combat.LimitedUsePartRef]dice.DieRoll
	}{
		{"wrong die", one, map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): d8}},
		{"zero roll", one, map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): {}}},
		{"missing roll", base, map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(5)}},
		{"extra roll for a ready part", one, map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(5), luLash(t): luRoll(6)}},
		{"roll for another part", one, map[combat.LimitedUsePartRef]dice.DieRoll{luLash(t): luRoll(5)}},
		{"no roll at all", one, nil},
		{"roll with nothing spent", luStart(t, luDrake(t), combat.OutsideLair), map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): luRoll(5)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			tr, err := c.state.StartOfOwnTurn(c.rolls)
			if !errors.Is(err, srderr.ErrInvalidInput) || !tr.State().IsZero() || len(tr.Events()) != 0 {
				t.Errorf("StartOfOwnTurn = %v, %v, want a zero transition and ErrInvalidInput [%sFR-012]", tr, err, limitedUseSpec)
			}
			if !c.state.Equal(before) {
				t.Error("the receiver changed")
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-013, FR-014, FR-025, FR-008 - a Short Rest or a Long Rest readies every spent Recharge and Recharge-after-a-Rest part with no roll, and a Short Rest changes nothing else (SRD 5.2.1, "Limited Usage", p. 257).
// Path: per case.
func TestAfterRestReadiesRechargeParts(t *testing.T) {
	t.Parallel()
	for _, rest := range []character.RestKind{character.ShortRest, character.LongRest} {
		t.Run(rest.String(), func(t *testing.T) {
			t.Parallel()
			s := luStart(t, luDrake(t), combat.OutsideLair)
			for _, p := range []combat.LimitedUsePartRef{luRestTrait(t), luSpray(t), luLash(t)} {
				s = luSpent(t, s, p)
			}
			tr := s.AfterRest(rest)
			for _, p := range []combat.LimitedUsePartRef{luRestTrait(t), luSpray(t), luLash(t)} {
				if luReadiness(t, tr.State(), p) != combat.RechargeReady {
					t.Errorf("%v after a %v rest is not ready [FR-013 FR-014]", p, rest)
				}
			}
			ev := tr.Events()
			if len(ev) != 1 {
				t.Fatalf("events = %v, want one LimitedUsesRestored", ev)
			}
			want := []combat.LimitedUseSubject{combat.PartSubject{Part: luRestTrait(t)}, combat.PartSubject{Part: luSpray(t)}, combat.PartSubject{Part: luLash(t)}}
			if got := ev[0].(combat.LimitedUsesRestored).Subjects(); !slices.Equal(got, want) {
				t.Errorf("subjects = %v, want %v in printed order [FR-027]", got, want)
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-008, FR-026, FR-027, XD-1 - a Short Rest does not restore X/Day uses and adds no event; a Long Rest sets every X/Day entry to its maximum for the current lair status and lists exactly what changed in one event; a rest with nothing to restore gives an equal state and no event (SRD 5.2.1, "Limited Usage", p. 257).
// Path: per case.
func TestAfterRestRestoresPerDayUses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		lair                    combat.LairStatus
		spend                   []combat.LimitedUsePartRef
		rest                    character.RestKind
		wantRes, wantMind       int
		wantSubjects            []combat.LimitedUseSubject
	}{
		{"short rest keeps uses", "FR-008 XD-1", "unhappy (rule interaction)", combat.OutsideLair,
			[]combat.LimitedUsePartRef{luMind(t), luMind(t)}, character.ShortRest, 3, 0, nil},
		{"long rest restores", "FR-026 FR-027 XD-1", "happy", combat.OutsideLair,
			[]combat.LimitedUsePartRef{luRes(t), luRes(t), luMind(t), luMind(t)}, character.LongRest, 3, 2,
			[]combat.LimitedUseSubject{combat.PartSubject{Part: luRes(t)}, combat.PartSubject{Part: luMind(t)}}},
		{"long rest in the lair", "FR-026 XD-1", "unhappy (rule interaction)", combat.InLair,
			[]combat.LimitedUsePartRef{luRes(t), luRes(t), luRes(t)}, character.LongRest, 4, 2,
			[]combat.LimitedUseSubject{combat.PartSubject{Part: luRes(t)}}},
		{"long rest with nothing spent", "FR-027 XD-1", "unhappy (empty and minimal)", combat.OutsideLair, nil, character.LongRest, 3, 2, nil},
		{"short rest with nothing spent", "FR-027", "unhappy (empty and minimal)", combat.OutsideLair, nil, character.ShortRest, 3, 2, nil},
		{"invalid rest changes nothing", "FR-027", "unhappy (invalid input)", combat.OutsideLair,
			[]combat.LimitedUsePartRef{luMind(t)}, 0, 3, 1, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement
			s := luStart(t, luDrake(t), c.lair)
			for _, p := range c.spend {
				s = luSpent(t, s, p)
			}
			tr := s.AfterRest(c.rest)
			if got := luPerDayOf(t, tr.State(), luRes(t)).Remaining(); got != c.wantRes {
				t.Errorf("Resistance uses = %d, want %d [%s]", got, c.wantRes, req)
			}
			if got := luPerDayOf(t, tr.State(), luMind(t)).Remaining(); got != c.wantMind {
				t.Errorf("Mind uses = %d, want %d [%s]", got, c.wantMind, req)
			}
			var got []combat.LimitedUseSubject
			for _, e := range tr.Events() {
				got = append(got, e.(combat.LimitedUsesRestored).Subjects()...)
			}
			if !slices.Equal(got, c.wantSubjects) {
				t.Errorf("restored subjects = %v, want %v [%s]", got, c.wantSubjects, req)
			}
			if c.wantSubjects == nil && (!tr.State().Equal(s) || len(tr.Events()) != 0) {
				t.Errorf("a rest with nothing to restore must give an equal state and no event [FR-027]")
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-028, LA-1 - ChangeLair recomputes each X/Day maximum and the Legendary maximum for the new status and keeps the uses already spent: remaining becomes max(0, new maximum - spent); it adds one MonsterLairStatusChanged event when the status differs and returns an equal state with no event when it does not (SRD 5.2.1, "Aboleth", p. 258).
// Path: per case.
func TestChangeLair(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		from, to                combat.LairStatus
		spentRes, spentLegend   int
		wantRes, wantLegendary  int
		wantMaxLegendary        int
		wantEvent               bool
	}{
		{"enter the lair with two spent", "FR-028 LA-1", "unhappy (rule interaction)", combat.OutsideLair, combat.InLair, 2, 2, 2, 2, 4, true},
		{"enter the lair with nothing spent", "FR-028", "happy", combat.OutsideLair, combat.InLair, 0, 0, 4, 4, 4, true},
		{"leave the lair, spent exceeds the new maximum", "FR-028 LA-1", "unhappy (boundary)", combat.InLair, combat.OutsideLair, 4, 4, 0, 0, 3, true},
		{"leave the lair with one spent", "FR-028", "unhappy (boundary)", combat.InLair, combat.OutsideLair, 1, 1, 2, 2, 3, true},
		{"leave the lair, spent equals the new maximum", "FR-028", "unhappy (boundary)", combat.InLair, combat.OutsideLair, 3, 3, 0, 0, 3, true},
		{"status unchanged", "FR-028 LA-1", "unhappy (empty and minimal)", combat.InLair, combat.InLair, 1, 1, 3, 3, 4, false},
		{"undefined status changes nothing", "FR-028", "unhappy (invalid input)", combat.InLair, 0, 1, 1, 3, 3, 4, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement
			s := luStart(t, luDrake(t), c.from)
			for range c.spentRes {
				s = luSpent(t, s, luRes(t))
			}
			for i := range c.spentLegend {
				s = luTake(t, s, luWindow(t, "b", i+1))
			}
			tr := s.ChangeLair(c.to)
			if got := luPerDayOf(t, tr.State(), luRes(t)).Remaining(); got != c.wantRes {
				t.Errorf("Resistance uses = %d, want %d [%s]", got, c.wantRes, req)
			}
			leg := luLegendaryOf(t, tr.State())
			if leg.Remaining() != c.wantLegendary || leg.Maximum() != c.wantMaxLegendary {
				t.Errorf("Legendary = %d of %d, want %d of %d [%s]", leg.Remaining(), leg.Maximum(), c.wantLegendary, c.wantMaxLegendary, req)
			}
			ev := tr.Events()
			if c.wantEvent {
				e, ok := ev[0].(combat.MonsterLairStatusChanged)
				if len(ev) != 1 || !ok || e.From() != c.from || e.To() != c.to || tr.State().Lair() != c.to {
					t.Errorf("events = %#v, lair %v, want one change %v to %v [%s]", ev, tr.State().Lair(), c.from, c.to, req)
				}
			} else if len(ev) != 0 || !tr.State().Equal(s) {
				t.Errorf("events = %v, want none and an equal state [%s]", ev, req)
			}
		})
	}
}
