package character_test

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func hoursAgo(t *testing.T, h int) character.LongRestSpacing {
	t.Helper()
	s, err := character.NewLongRestFinishedHoursAgo(h)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: per case.
// Path: per case.
func TestCheckRestStart(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		hp, exhaustion          int
		spacing                 character.LongRestSpacing
		wantBlockers            []string
	}{
		{"short rest at 1 Hit Point may start", "FR-007 (SRD 5.2.1, \"Short Rest\", p. 187)", "unhappy (boundary: 1)", character.ShortRest, 1, 0, nil, nil},
		{"short rest at 0 Hit Points may not", "FR-007", "unhappy (boundary: 0)", character.ShortRest, 0, 0, nil, []string{"hp"}},
		{"long rest at 1 Hit Point may start", "FR-008 (SRD 5.2.1, \"Long Rest\", p. 185)", "unhappy (boundary: 1)", character.LongRest, 1, 0, nil, nil},
		{"long rest at 0 Hit Points may not", "FR-008", "unhappy (boundary: 0)", character.LongRest, 0, 0, nil, []string{"hp"}},
		{"long rest 15 hours after the last", "FR-009", "unhappy (boundary: 15)", character.LongRest, 5, 0, hoursAgo(t, 15), []string{"wait 1"}},
		{"long rest 16 hours after the last", "FR-009", "unhappy (boundary: 16)", character.LongRest, 5, 0, hoursAgo(t, 16), nil},
		{"long rest 17 hours after the last", "FR-009", "unhappy (boundary: 17)", character.LongRest, 5, 0, hoursAgo(t, 17), nil},
		{"long rest straight after the last waits 16", "FR-009", "unhappy (boundary: 0)", character.LongRest, 5, 0, hoursAgo(t, 0), []string{"wait 16"}},
		{"long rest with no previous long rest", "FR-009", "happy", character.LongRest, 5, 0, character.NoLongRestFinishedYet{}, nil},
		{"long rest with nil spacing", "FR-009", "happy", character.LongRest, 5, 0, nil, nil},
		{"short rest straight after a long rest", "FR-010", "unhappy (rule interaction)", character.ShortRest, 5, 0, hoursAgo(t, 0), nil},
		{"exhaustion 5 may start", "FR-011 (SRD 5.2.1, \"Exhaustion\", p. 181)", "unhappy (boundary: 5)", character.LongRest, 5, 5, nil, nil},
		{"exhaustion 6 short rest may not", "FR-011", "unhappy (state error)", character.ShortRest, 5, 6, nil, []string{"six"}},
		{"exhaustion 6 long rest may not", "FR-011", "unhappy (state error)", character.LongRest, 5, 6, nil, []string{"six"}},
		{"every blocker is listed in order", "FR-012", "unhappy (rule interaction)", character.LongRest, 0, 6, hoursAgo(t, 3), []string{"hp", "six", "wait 13"}},
		{"two blockers keep the order", "FR-012", "unhappy (rule interaction)", character.LongRest, 5, 6, hoursAgo(t, 1), []string{"six", "wait 15"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := character.CheckRestStart(c.kind, c.hp, c.exhaustion, c.spacing)
			if err != nil {
				t.Fatalf("CheckRestStart: %v", err)
			}
			var got []string
			if n, ok := d.(character.RestMayNotStart); ok {
				for _, b := range n.Blockers() {
					switch b := b.(type) {
					case character.RestStartNoHitPoints:
						got = append(got, "hp")
					case character.RestStartExhaustionLevelSix:
						got = append(got, "six")
					case character.RestStartTooSoonAfterLongRest:
						got = append(got, "wait "+strconv.Itoa(b.HoursStillToWait()))
					}
				}
			} else if _, ok := d.(character.RestMayStart); !ok {
				t.Fatalf("decision = %#v", d)
			}
			if !reflect.DeepEqual(got, c.wantBlockers) {
				t.Errorf("blockers = %v, want %v [%s]", got, c.wantBlockers, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-013 - invalid host values are
// errors matching ErrInvalidInput, never rule outcomes.
// Path: per case.
func TestCheckRestStartRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		kind           character.RestKind
		hp, exhaustion int
	}{
		"negative Hit Points": {character.LongRest, -1, 0}, "negative exhaustion": {character.LongRest, 1, -1},
		"exhaustion 7": {character.LongRest, 1, 7}, "invalid kind": {0, 1, 0},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d, err := character.CheckRestStart(c.kind, c.hp, c.exhaustion, nil)
			if !errors.Is(err, srderr.ErrInvalidInput) || d != nil {
				t.Errorf("CheckRestStart = %v, %v, want nil and ErrInvalidInput", d, err)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-012, architecture.md section 4
// (handoffs) - the blockers of a decision are a copy.
// Path: unhappy (immutability).
func TestRestMayNotStartBlockersAreACopy(t *testing.T) {
	t.Parallel()
	d, _ := character.CheckRestStart(character.LongRest, 0, 0, nil)
	n := d.(character.RestMayNotStart)
	n.Blockers()[0] = character.RestStartExhaustionLevelSix{}
	if _, ok := n.Blockers()[0].(character.RestStartNoHitPoints); !ok {
		t.Error("mutating the returned slice changed the decision")
	}
}
