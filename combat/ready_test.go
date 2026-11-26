package combat_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/initiative-and-actions.md FR-137 - ParseReadyTriggerID rejects the empty and whitespace-only strings with a *ReadyTriggerIDError (reason ReadyTriggerEmpty) matching ErrInvalidInput and keeps any other string unchanged, with no trimming or case folding.
// Path: per case.
func TestParseReadyTriggerID(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", " ", "\t", "\n", " \t\n"} {
		id, err := combat.ParseReadyTriggerID(s)
		var te *combat.ReadyTriggerIDError
		if id != "" || !errors.As(err, &te) || te.Reason != combat.ReadyTriggerEmpty || !errors.Is(err, srderr.ErrInvalidInput) ||
			!errors.Is(err, &combat.ReadyTriggerIDError{Reason: combat.ReadyTriggerEmpty}) || te.Reason.String() != "empty" || te.Error() == "" {
			t.Errorf("Parse(%q) = %q, %v; want ReadyTriggerEmpty", s, id, err)
		}
	}
	for _, s := range []string{"a", " a", "Door opens", "door opens", "a "} {
		if id, err := combat.ParseReadyTriggerID(s); err != nil || string(id) != s {
			t.Errorf("Parse(%q) = %q, %v; want it unchanged", s, id, err)
		}
	}
	if combat.MustReadyTriggerID("Door") == combat.MustReadyTriggerID("door") {
		t.Error("\"Door\" and \"door\" are equal")
	}
	if (&combat.ReadyTriggerIDError{}).Is(errors.New("x")) || (*combat.ReadyTriggerIDError)(nil).Is(&combat.ReadyTriggerIDError{}) {
		t.Error("Is matches a foreign error or a nil receiver")
	}
	if combat.ReadyTriggerIDRejection(0).String() != "unknown" {
		t.Error("the zero rejection is named")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-137 - MustReadyTriggerID panics with the error ParseReadyTriggerID returns, and only then.
// Path: unhappy (invalid input).
func TestMustReadyTriggerID(t *testing.T) {
	t.Parallel()
	if combat.MustReadyTriggerID("x") != "x" {
		t.Error("Must changed a valid id")
	}
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an error matching ErrInvalidInput", r)
		}
	}()
	combat.MustReadyTriggerID(" ")
}

// Requirement: doc/specs/initiative-and-actions.md FR-137 - ParseReadyTriggerID never panics for any input, keeps a valid string unchanged and rejects exactly the blank ones.
// Path: unhappy (invalid input).
func FuzzParseReadyTriggerID(f *testing.F) {
	for _, s := range []string{"", " ", "\t", "a", " a", "Door", "\xff", strings.Repeat("x", 4096)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := combat.ParseReadyTriggerID(s)
		if blank := strings.TrimSpace(s) == ""; blank != (err != nil) {
			t.Fatalf("Parse(%q) error = %v", s, err)
		}
		if err == nil && string(id) != s {
			t.Fatalf("Parse(%q) = %q, want it unchanged", s, id)
		}
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("error %v does not match ErrInvalidInput", err)
		}
	})
}

// Requirement: doc/specs/initiative-and-actions.md FR-057, FR-150, FR-163 - NewReadiedAction rejects an empty trigger, a nil or invalid response and a held spell that is invalid or whose trigger differs; a readied move needs segments; Held is HoldsNoSpell by default.
// Path: unhappy (invalid input).
func TestNewReadiedAction(t *testing.T) {
	t.Parallel()
	ok := combat.ReadiedActionResponse{Choice: trkAttack}
	spell := func(s, c string, trig string) combat.HoldsSpell {
		return combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: contentSpell(s), Caster: combat.CombatantID(c), Trigger: combat.ReadyTriggerID(trig)}}
	}
	bad := map[string]struct {
		trigger combat.ReadyTriggerID
		resp    combat.ReadiedResponse
		held    combat.HeldSpellInput
	}{
		"empty trigger":         {"", ok, nil},
		"nil response":          {"t", nil, nil},
		"nil choice":            {"t", combat.ReadiedActionResponse{}, nil},
		"invalid choice":        {"t", combat.ReadiedActionResponse{Choice: combat.DashChoice{}}, nil},
		"move without segments": {"t", combat.ReadiedMoveResponse{}, nil},
		"spell without id":      {"t", ok, spell("", "a", "t")},
		"spell without caster":  {"t", ok, spell("fireball", "", "t")},
		"spell trigger differs": {"t", ok, spell("fireball", "a", "u")},
	}
	for name, c := range bad {
		if a, err := combat.NewReadiedAction(c.trigger, c.resp, c.held); !errors.Is(err, srderr.ErrInvalidInput) || !a.IsZero() {
			t.Errorf("%s: %+v, %v; want a zero action and invalid input", name, a, err)
		}
	}
	a, err := combat.NewReadiedAction("t", ok, nil)
	if err != nil || a.Held() != (combat.HoldsNoSpell{}) || a.Trigger() != "t" || a.Response() != ok || a.IsZero() {
		t.Errorf("good action = %+v, %v", a, err)
	}
	if (combat.ReadiedAction{}).Held() != (combat.HoldsNoSpell{}) || !(combat.ReadiedAction{}).Equal(combat.ReadiedAction{}) {
		t.Error("the zero action is wrong")
	}
	held := spell("fireball", "a", "t")
	b, err := combat.NewReadiedAction("t", ok, held)
	if err != nil || b.Held() != held || b.Equal(a) || a.Equal(b) {
		t.Errorf("held action = %+v, %v", b, err)
	}
	if _, err := combat.NewReadiedMoveResponse([]combat.MoveSegment{{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero segment: %v", err)
	}
	mv, _ := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 5)})
	m, _ := combat.NewReadiedAction("t", mv, nil)
	if m.Equal(a) || a.Equal(m) || !m.Equal(m) {
		t.Error("Equal across response kinds is wrong")
	}
	segs := mv.Segments()
	segs[0] = combat.MoveSegment{}
	if mv.Segments()[0].IsZero() {
		t.Error("ReadiedMoveResponse shares its segments")
	}
}

// Requirement: per case (table 4.11a, FR-163)
// Path: happy.
func TestHeldSpell(t *testing.T) {
	t.Parallel()
	held := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkA, Trigger: "door"}}
	action, err := combat.NewReadiedAction("door", combat.ReadiedActionResponse{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}, held)
	if err != nil {
		t.Fatal(err)
	}
	ready := combat.ReadyChoice{Action: action}
	// A caster other than the actor is invalid input for the command.
	if _, err := combat.NewTakeActionCommand(trkABC(t), trkB, trkAvail, ready); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("caster is not the actor: %v, want invalid input [FR-163]", err)
	}
	tr := trkTake(t, trkABC(t), trkA, ready)
	if h, err := tr.HeldSpellOf(trkA); err != nil || h != combat.HeldSpellInput(held) {
		t.Errorf("HeldSpellOf = %v, %v", h, err)
	}
	if h, _ := tr.HeldSpellOf(trkB); h != (combat.HoldsNoSpell{}) {
		t.Errorf("HeldSpellOf(b) = %v, want HoldsNoSpell", h)
	}
	// The Reaction record carries the held spell.
	adv, _ := trkAdvance(t, tr)
	_, ev := trkApplied(t)(adv.TakeReaction(trkA, trkAvail, combat.ReadiedActionReaction{Trigger: "door"}))
	if rec := ev[0].(combat.ReactionTaken).Record().(combat.ReadiedActionRecord); rec.Held != combat.HeldSpellInput(held) {
		t.Errorf("record = %+v, want the held spell [FR-163]", rec)
	}
	// Expiry carries the held spell.
	cur := advN(t, tr, 2)
	_, ev = trkAdvance(t, cur)
	if e := ev[len(ev)-1].(combat.ReadiedActionExpired); e.Held() != combat.HeldSpellInput(held) {
		t.Errorf("expiry = %+v, want the held spell [FR-163]", e)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-137, FR-163 - ReadyTriggerIDError matches only another error of its own reason; a held spell with a trigger that sorts before or after the action's is rejected; readied actions with different triggers are unequal both ways round.
// Path: unhappy (boundary).
func TestReadyBoundaries(t *testing.T) {
	t.Parallel()
	e := &combat.ReadyTriggerIDError{Reason: combat.ReadyTriggerEmpty}
	if e.Is(&combat.ReadyTriggerIDError{Reason: 9}) || e.Is(&combat.ReadyTriggerIDError{}) || !e.Is(&combat.ReadyTriggerIDError{Reason: combat.ReadyTriggerEmpty}) {
		t.Error("Is does not compare the reason")
	}
	if e.Is((*combat.ReadyTriggerIDError)(nil)) {
		t.Error("Is matches a nil target")
	}
	ok := combat.ReadiedActionResponse{Choice: trkAttack}
	for _, held := range []string{"a", "z"} {
		h := combat.HoldsSpell{Spell: combat.ReadiedSpell{Spell: "fireball", Caster: trkA, Trigger: combat.ReadyTriggerID(held)}}
		if _, err := combat.NewReadiedAction("m", ok, h); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("held trigger %q against m: %v, want invalid input", held, err)
		}
	}
	a, _ := combat.NewReadiedAction("a", ok, nil)
	b, _ := combat.NewReadiedAction("b", ok, nil)
	if a.Equal(b) || b.Equal(a) {
		t.Error("readied actions with different triggers are equal")
	}
	other, _ := combat.NewReadiedAction("a", combat.ReadiedActionResponse{Choice: trkDodge}, nil)
	if a.Equal(other) || other.Equal(a) {
		t.Error("readied actions with different choices are equal")
	}
	mv1, _ := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 5)})
	mv2, _ := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 6)})
	m1, _ := combat.NewReadiedAction("a", mv1, nil)
	m2, _ := combat.NewReadiedAction("a", mv2, nil)
	if m1.Equal(m2) || !m1.Equal(m1) {
		t.Error("readied moves compare wrongly")
	}
}
