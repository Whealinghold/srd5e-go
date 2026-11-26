package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-action-resolution.md FR-056, FR-096, FR-004 - a MoveAlong needs at least one segment and no zero segment, takes its own copy and gives a copy back.
// Path: unhappy (invalid input).
func TestNewMoveAlong(t *testing.T) {
	t.Parallel()
	seg := trkWalk(t, 10)
	for name, in := range map[string][]combat.MoveSegment{"nil": nil, "empty": {}, "zero segment": {seg, {}}} {
		if m, err := combat.NewMoveAlong(in); !errors.Is(err, srderr.ErrInvalidInput) || len(m.Segments()) != 0 {
			t.Errorf("%s: NewMoveAlong = %v, %v, want ErrInvalidInput", name, m, err)
		}
	}
	in := []combat.MoveSegment{seg, trkWalk(t, 5)}
	m, err := combat.NewMoveAlong(in)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = trkWalk(t, 30)
	got := m.Segments()
	got[1] = trkWalk(t, 30)
	if want := []combat.MoveSegment{seg, trkWalk(t, 5)}; !m.Segments()[0].Equal(want[0]) || !m.Segments()[1].Equal(want[1]) {
		t.Errorf("segments = %v, want the copy taken at construction and a copy on the way out [FR-004]", m.Segments())
	}
}

// Requirement: per case
// Path: per case.
func TestNewStandUp(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		feet                    int
		ok                      bool
	}{
		{"30 feet", resSpec + "FR-143", "happy", 30, true},
		{"0 feet", resSpec + "FR-143", "unhappy (boundary)", 0, true},
		{"negative", resSpec + "FR-143", "unhappy (boundary)", -1, false},
		{"most negative", resSpec + "FR-143", "unhappy (boundary)", math.MinInt, false},
		{"largest", resSpec + "FR-143", "unhappy (boundary)", math.MaxInt, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.NewStandUp(c.feet)
			if c.ok != (err == nil) || (c.ok && s.BaseWalkingSpeedFeet() != c.feet) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("NewStandUp(%d) = %v, %v [%s]", c.feet, s, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-143, testing.md section 10 - NewStandUp never panics, and rejects a Speed if and only if it is negative.
// Path: unhappy (invalid input).
func FuzzNewStandUp(f *testing.F) {
	for _, v := range []int{0, 30, -1, math.MaxInt, math.MinInt} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, feet int) {
		s, err := combat.NewStandUp(feet)
		if (err != nil) != (feet < 0) || (err == nil && s.BaseWalkingSpeedFeet() != feet) {
			t.Errorf("NewStandUp(%d) = %v, %v", feet, s, err)
		}
	})
}

// Requirement: doc/specs/combat-action-resolution.md FR-051, FR-096 - a TakeCombatantAction command refuses a nil request and every request with an invalid part, before any state is read.
// Path: per case.
func TestNewTakeCombatantActionCommandRefusesInvalidRequests(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	pull := replacementOf(t)
	cases := []struct {
		name    string
		id      combat.CombatantID
		enc     combat.Encounter
		request combat.TurnRequest
		ok      bool
	}{
		{"nil request", cid("a"), e, nil, false},
		{"zero encounter", cid("a"), combat.Encounter{}, combat.SpendInteraction{}, false},
		{"blank combatant", combat.CombatantID(""), e, combat.SpendInteraction{}, false},
		{"action without a choice", cid("a"), e, combat.SpendAction{}, false},
		{"reaction without a choice", cid("a"), e, combat.SpendReaction{}, false},
		{"help without a request", cid("a"), e, combat.TakeHelpAction{}, false},
		{"help without an ally", cid("a"), e, combat.TakeHelpAction{Request: combat.AbilityCheckHelpRequest{}}, false},
		{"help without an enemy", cid("a"), e, combat.TakeHelpAction{Request: combat.AttackRollHelpRequest{}}, false},
		{"weapon change undefined", cid("a"), e, combat.HandleWeapon{}, false},
		{"replacement undefined", cid("a"), e, combat.ReplaceAttack{}, false},
		{"replacement without a cost", cid("a"), e, combat.ReplaceAttack{Replacement: pull}, false},
		{"replacement with an invalid allowance", cid("a"), e, combat.ReplaceAttack{Cost: combat.BeginsAttackActionByReplacing{}, Replacement: pull}, false},
		{"action", cid("a"), e, combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}}, true},
		{"bonus action with no grant", cid("a"), e, combat.SpendBonusAction{}, true},
		{"interaction", cid("a"), e, combat.SpendInteraction{}, true},
		{"weapon change", cid("a"), e, combat.HandleWeapon{Change: combat.WeaponChangeDrop, Timing: combat.WeaponChangeAfterAttack}, true},
		{"replacement", cid("a"), e, combat.ReplaceAttack{Cost: combat.ContinuesAttackActionByReplacing{}, Replacement: pull}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewTakeCombatantActionCommand(c.enc, combat.ActionLedger{}, c.id, c.request)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("= %v, want ok=%v with ErrInvalidInput on refusal [%sFR-051]", err, c.ok, resSpec)
			}
		})
	}
}
