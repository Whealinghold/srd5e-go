package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/telepathy.md FR-001 - a range below 1 is rejected with ErrTelepathyRange and ErrInvalidInput; 1 and the largest int are accepted; the zero value is not telepathic.
// Path: unhappy (invalid input).
func TestNewTelepathicRejectsRangeBelowOne(t *testing.T) {
	t.Parallel()
	for _, r := range []int{0, -1, math.MinInt} {
		c, err := combat.NewTelepathic(r)
		if !errors.Is(err, combat.ErrTelepathyRange) || !errors.Is(err, srderr.ErrInvalidInput) || c.IsTelepathic() {
			t.Errorf("NewTelepathic(%d) = %v, %v, want ErrTelepathyRange and ErrInvalidInput", r, c, err)
		}
	}
	for _, r := range []int{1, 60, math.MaxInt} {
		c, err := combat.NewTelepathic(r)
		if err != nil || !c.IsTelepathic() || c.RangeFeet() != r {
			t.Errorf("NewTelepathic(%d) = %v, %v", r, c, err)
		}
	}
	if (combat.TelepathyCapability{}).IsTelepathic() || (combat.TelepathyCapability{}).RangeFeet() != 0 {
		t.Error("the zero capability must be NotTelepathic [FR-001]")
	}
}

// Requirement: doc/specs/telepathy.md FR-001 - MustTelepathic panics with the error of NewTelepathic and only for an invalid range.
// Path: unhappy (invalid input).
func TestMustTelepathic(t *testing.T) {
	t.Parallel()
	if combat.MustTelepathic(30).RangeFeet() != 30 {
		t.Error("MustTelepathic(30) range")
	}
	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, combat.ErrTelepathyRange) {
			t.Errorf("panic value = %v, want an error matching ErrTelepathyRange", err)
		}
	}()
	combat.MustTelepathic(0)
}

// Requirement: doc/specs/telepathy.md FR-001 - the range is a fuzzed constructor input: any int either builds a capability with that range or fails with ErrTelepathyRange; it never panics.
// Path: unhappy (invalid input).
func FuzzNewTelepathic(f *testing.F) {
	for _, r := range []int{0, 1, -1, 60, math.MaxInt, math.MinInt} {
		f.Add(r)
	}
	f.Fuzz(func(t *testing.T, r int) {
		c, err := combat.NewTelepathic(r)
		if err != nil {
			if r >= 1 || !errors.Is(err, combat.ErrTelepathyRange) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewTelepathic(%d) error = %v", r, err)
			}
			return
		}
		if r < 1 || c.RangeFeet() != r {
			t.Fatalf("NewTelepathic(%d) range = %d", r, c.RangeFeet())
		}
	})
}

// Requirement: doc/specs/telepathy.md section 3 - InContact builds a contact with the pair, range and policy; invalid arguments give NoContact; the policy takes part in equality.
// Path: unhappy (invalid input).
func TestInContact(t *testing.T) {
	t.Parallel()
	a, b := gid("a"), gid("b")
	s := combat.InContact(a, b, 60, combat.ContactedMayReply)
	if !s.IsInContact() || s.RangeFeet() != 60 || s.ReplyPolicy() != combat.ContactedMayReply || !s.IsInPair(a) || !s.IsInPair(b) || s.IsInPair(gid("c")) {
		t.Errorf("state = %+v", s)
	}
	got := s.Combatants()
	got[0] = "x"
	if again := s.Combatants(); again[0] != a || again[1] != b {
		t.Errorf("Combatants() = %v, not a copy", again)
	}
	if s == combat.InContact(a, b, 60, combat.ContactedMayNotReply) || s != combat.InContact(a, b, 60, combat.ContactedMayReply) {
		t.Error("equality must follow the policy [FR-023]")
	}
	for name, bad := range map[string]combat.TelepathicContactState{
		"blank telepath": combat.InContact("", b, 60, 0), "blank contacted": combat.InContact(a, "", 60, 0), "same creature": combat.InContact(a, a, 60, 0),
		"range 0": combat.InContact(a, b, 0, 0), "undefined policy": combat.InContact(a, b, 60, 9), "negative policy": combat.InContact(a, b, 60, -1),
	} {
		if bad != (combat.TelepathicContactState{}) || bad.IsInContact() || bad.Combatants() != nil || bad.IsInPair(a) || bad.RangeFeet() != 0 {
			t.Errorf("%s: state = %+v, want NoContact", name, bad)
		}
	}
}

// Requirement: doc/specs/telepathy.md section 3 - the enums print their snake_case names and "unknown" outside their range.
// Path: unhappy (boundary).
func TestTelepathyEnumStrings(t *testing.T) {
	t.Parallel()
	got := []string{combat.ContactedMayReply.String(), combat.ContactedMayNotReply.String(), combat.ReplyPolicy(2).String(), combat.ReplyPolicy(-1).String(),
		combat.ContactedUnderstandingNeither.String(), combat.ContactedUnderstandingALanguage.String(), combat.ContactedUnderstandingTelepathic.String(), combat.ContactedUnderstanding(3).String(),
		combat.TelepathicContactRejectionInvalidCommand.String(), combat.TelepathicContactRejectionContactedCannotUnderstand.String(), combat.TelepathicContactRejection(0).String(), combat.TelepathicContactRejection(7).String(),
		combat.BreakCauseIncapacitated.String(), combat.BreakCauseTelepathContactedAnother.String(), combat.BreakCause(0).String(), combat.BreakCause(4).String()}
	want := []string{"contacted_may_reply", "contacted_may_not_reply", "unknown", "unknown", "neither", "a_language", "telepathic", "unknown",
		"invalid_command", "contacted_cannot_understand", "unknown", "unknown", "incapacitated", "telepath_contacted_another", "unknown", "unknown"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("string %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Requirement: doc/specs/telepathy.md FR-001, section 3 - a range of exactly 1 foot makes a contact.
// Path: unhappy (boundary).
func TestInContactSmallestRange(t *testing.T) {
	t.Parallel()
	if s := combat.InContact(gid("a"), gid("b"), 1, combat.ContactedMayReply); !s.IsInContact() || s.RangeFeet() != 1 {
		t.Errorf("state = %+v, want a contact of range 1", s)
	}
}
