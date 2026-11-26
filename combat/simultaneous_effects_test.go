package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func mePending(ids ...string) []combat.PendingEffectID {
	var out []combat.PendingEffectID
	for _, id := range ids {
		out = append(out, meMust(combat.NewPendingEffectID(id)))
	}
	return out
}

// Requirement: doc/specs/magical-effects.md FR-091 - the person whose turn it is decides: a player character's turn gives the player, a monster's turn the Game Master; an undefined holder gives the zero authority (SRD 5.2.1, "Simultaneous Effects", p. 187).
// Path: happy.
func TestSimultaneousEffectsAuthorityFor(t *testing.T) {
	t.Parallel()
	s := combat.SimultaneousEffects{}
	if s.AuthorityFor(combat.TurnHolderPlayer) != combat.OrderingAuthorityPlayer || s.AuthorityFor(combat.TurnHolderGameMaster) != combat.OrderingAuthorityGameMaster ||
		s.AuthorityFor(0) != 0 || s.AuthorityFor(combat.TurnHolderGameMaster+1) != 0 {
		t.Error("authority must follow the turn holder")
	}
	if combat.OrderingAuthorityPlayer.String() != "player" || combat.OrderingAuthorityGameMaster.String() != "game_master" || combat.OrderingAuthority(0).String() != "unknown" {
		t.Error("authority names changed")
	}
}

// Requirement: doc/specs/magical-effects.md FR-092 - a chosen order that is a permutation of the pending ids is Ordered, carrying that order exactly as chosen (never sorted) and the authority.
// Path: happy.
func TestOrderSimultaneousEffectsOrdered(t *testing.T) {
	t.Parallel()
	for _, h := range []struct {
		holder combat.TurnHolder
		auth   combat.OrderingAuthority
	}{{combat.TurnHolderPlayer, combat.OrderingAuthorityPlayer}, {combat.TurnHolderGameMaster, combat.OrderingAuthorityGameMaster}} {
		pending, chosen := mePending("a", "b", "c"), mePending("c", "a", "b")
		res := (combat.SimultaneousEffects{}).Order(h.holder, pending, chosen)
		ordered, ok := res.(combat.OrderSimultaneousEffectsOrdered)
		if !ok || !slices.Equal(ordered.Order(), chosen) || ordered.Authority() != h.auth {
			t.Errorf("holder %d: got %#v", h.holder, res)
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-094 - fewer than two pending effects are Ordered with the pending list as is (empty included): there is nothing to decide, whatever was chosen.
// Path: unhappy (empty and minimal).
func TestOrderSimultaneousEffectsFewerThanTwo(t *testing.T) {
	t.Parallel()
	for name, pending := range map[string][]combat.PendingEffectID{"nil": nil, "empty": {}, "one": mePending("a")} {
		res := (combat.SimultaneousEffects{}).Order(combat.TurnHolderPlayer, pending, mePending("zzz"))
		ordered, ok := res.(combat.OrderSimultaneousEffectsOrdered)
		if !ok || !slices.Equal(ordered.Order(), pending) || ordered.Authority() != combat.OrderingAuthorityPlayer {
			t.Errorf("%s: got %#v", name, res)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestOrderSimultaneousEffectsRejected(t *testing.T) {
	t.Parallel()
	type p = combat.OrderProblem
	cases := []struct {
		name, requirement, path string
		holder                  combat.TurnHolder
		pending, chosen         []combat.PendingEffectID
		want                    []p
	}{
		{"omits a pending id", "FR-093", "unhappy (invalid input)", combat.TurnHolderPlayer, mePending("a", "b", "c"), mePending("a", "c"), []p{{combat.OrderProblemMissing, "b"}}},
		{"repeats an id", "FR-093", "unhappy (invalid input)", combat.TurnHolderPlayer, mePending("a", "b"), mePending("a", "b", "a"), []p{{combat.OrderProblemRepeated, "a"}}},
		{"names an id that is not pending", "FR-093", "unhappy (invalid input)", combat.TurnHolderPlayer, mePending("a", "b"), mePending("a", "x", "b"), []p{{combat.OrderProblemUnknown, "x"}}},
		{"lists every problem, repeated and unknown in chosen order, then missing in pending order", "FR-093", "unhappy (invalid input)", combat.TurnHolderGameMaster,
			mePending("a", "b", "c", "d"), mePending("x", "c", "x", "c"),
			[]p{{combat.OrderProblemUnknown, "x"}, {combat.OrderProblemUnknown, "x"}, {combat.OrderProblemRepeated, "c"}, {combat.OrderProblemMissing, "a"}, {combat.OrderProblemMissing, "b"}, {combat.OrderProblemMissing, "d"}}},
		{"a pending id twice", "FR-095", "unhappy (invalid input)", combat.TurnHolderPlayer, mePending("a", "b", "a"), mePending("a", "b"), []p{{combat.OrderProblemDuplicatePending, "a"}}},
		{"a pending id twice with other problems, duplicates first", "FR-093, FR-095", "unhappy (invalid input)", combat.TurnHolderPlayer, mePending("a", "a", "b"), mePending("b", "q"),
			[]p{{combat.OrderProblemDuplicatePending, "a"}, {combat.OrderProblemUnknown, "q"}, {combat.OrderProblemMissing, "a"}}},
		{"an undefined turn holder", "FR-091 (package: no authority can be named)", "unhappy (invalid input)", 0, mePending("a", "b"), mePending("a", "b"), []p{{Kind: combat.OrderProblemUnknownTurnHolder}}},
		{"an undefined turn holder with fewer than two pending", "FR-094", "unhappy (invalid input)", 0, mePending("a"), mePending("a"), []p{{Kind: combat.OrderProblemUnknownTurnHolder}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := (combat.SimultaneousEffects{}).Order(c.holder, c.pending, c.chosen)
			rej, ok := res.(combat.OrderSimultaneousEffectsRejected)
			if !ok || !slices.Equal(rej.Problems(), c.want) {
				t.Errorf("Order = %#v, want Rejected %v [%s%s; %s]", res, c.want, meSpec, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-096 - Order does not alter the lists given, the result views are copies, and equal inputs give equal results.
// Path: unhappy (immutability).
func TestOrderSimultaneousEffectsImmutability(t *testing.T) {
	t.Parallel()
	pending, chosen := mePending("a", "b"), mePending("b", "a")
	res := (combat.SimultaneousEffects{}).Order(combat.TurnHolderPlayer, pending, chosen).(combat.OrderSimultaneousEffectsOrdered)
	pending[0], chosen[0] = "zzz", "zzz" // the caller changes its slices afterwards
	if !slices.Equal(res.Order(), mePending("b", "a")) {
		t.Errorf("order = %v", res.Order())
	}
	res.Order()[0] = "zzz"
	if res.Order()[0] != "b" {
		t.Error("Order is not a copy")
	}
	again := (combat.SimultaneousEffects{}).Order(combat.TurnHolderPlayer, mePending("a", "b"), mePending("b", "a")).(combat.OrderSimultaneousEffectsOrdered)
	if !slices.Equal(again.Order(), res.Order()) || again.Authority() != res.Authority() {
		t.Error("equal inputs must give equal results")
	}
	rej := (combat.SimultaneousEffects{}).Order(combat.TurnHolderPlayer, mePending("a", "b"), nil).(combat.OrderSimultaneousEffectsRejected)
	rej.Problems()[0].ID = "zzz"
	if rej.Problems()[0].ID != "a" {
		t.Error("Problems is not a copy")
	}
	single := mePending("a")
	one := (combat.SimultaneousEffects{}).Order(combat.TurnHolderPlayer, single, nil).(combat.OrderSimultaneousEffectsOrdered)
	one.Order()[0] = "zzz"
	if single[0] != "a" {
		t.Error("the single pending list was changed")
	}
}

// Requirement: doc/specs/magical-effects.md section 5 - a blank PendingEffectID is an ErrInvalidInput error; the problem kinds have pinned names.
// Path: unhappy (invalid input).
func TestPendingEffectIDAndProblemKinds(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "  "} {
		if _, err := combat.NewPendingEffectID(s); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewPendingEffectID(%q) error = %v", s, err)
		}
	}
	got := []string{combat.OrderProblemDuplicatePending.String(), combat.OrderProblemRepeated.String(), combat.OrderProblemUnknown.String(), combat.OrderProblemMissing.String(),
		combat.OrderProblemUnknownTurnHolder.String(), combat.OrderProblemLimitExceeded.String(), combat.OrderProblemKind(0).String(), (combat.OrderProblemLimitExceeded + 1).String()}
	want := []string{"duplicate_pending", "repeated", "unknown", "missing", "unknown_turn_holder", "limit_exceeded", "unknown_kind", "unknown_kind"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
