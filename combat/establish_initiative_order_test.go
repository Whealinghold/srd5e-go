package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// trkIDs builds ids from names.
func trkIDs(names ...string) []combat.CombatantID {
	out := make([]combat.CombatantID, len(names))
	for i, n := range names {
		out[i] = combat.CombatantID(n)
	}
	return out
}

func trkStandings(counts ...int) []combat.InitiativeStanding {
	out := make([]combat.InitiativeStanding, len(counts))
	for i, c := range counts {
		out[i] = combat.InitiativeStanding{Combatant: combat.CombatantID(string(rune('a' + i))), Count: combat.InitiativeCount(c)}
	}
	return out
}

func trkProfiles(standings []combat.InitiativeStanding) []combat.CombatantProfile {
	out := make([]combat.CombatantProfile, len(standings))
	for i, s := range standings {
		out[i] = combat.CombatantProfile{ID: s.Combatant, Size: content.SizeMedium, Speeds: trkSpeeds(nil2t{}, 30)}
	}
	return out
}

// nil2t lets trkProfiles build speeds without a *testing.T: 30 is always valid.
type nil2t struct{ testing.TB }

func (nil2t) Helper() {}

func trkBreak(tb testing.TB, names ...string) combat.TieBreak {
	tb.Helper()
	tieBreak, err := combat.NewTieBreak(trkIDs(names...))
	if err != nil {
		tb.Fatal(err)
	}
	return tieBreak
}

func trkEstablish(tb testing.TB, standings []combat.InitiativeStanding, breaks []combat.TieBreak, opts ...combat.Option) combat.EstablishInitiativeOrderResult {
	tb.Helper()
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, trkProfiles(standings), breaks)
	if err != nil {
		tb.Fatalf("NewEstablishInitiativeOrderCommand: %v", err)
	}
	res, err := combat.NewEstablishInitiativeOrder(opts...).Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res
}

func trkOrderIDs(order combat.TurnOrder) []string {
	var out []string
	for _, s := range order.Standings() {
		out = append(out, string(s.Combatant))
	}
	return out
}

// Requirement: per case (table 4.2a, FR-018 to FR-023, FR-102, FR-120). Counts: A 15, B 15, C 10, D 10, E 10, F 5; groups {A, B} and {C, D, E}.
// Path: per case.
func TestEstablishInitiativeOrderTies(t *testing.T) {
	t.Parallel()
	type groupsT = [][]string
	cases := []struct {
		name, requirement, path string
		breaks                  [][]string
		wantOrder               []string // established
		wantUnresolved          groupsT  // InitiativeTiesUnresolved
		wantRejected            []struct {
			ids    []string
			reason combat.TieBreakRejectionReason
		}
		ties []string
	}{
		{name: "no tie-breaks", requirement: "FR-019", path: "unhappy (rule interaction)", wantUnresolved: groupsT{{"a", "b"}, {"c", "d", "e"}}, ties: []string{"unresolved", "unresolved"}},
		{name: "one group covered", requirement: "FR-019", path: "unhappy (rule interaction)", breaks: [][]string{{"a", "b"}}, wantUnresolved: groupsT{{"c", "d", "e"}}, ties: []string{"resolved", "unresolved"}},
		{name: "both covered, reordered", requirement: "FR-020", path: "happy", breaks: [][]string{{"b", "a"}, {"e", "c", "d"}}, wantOrder: []string{"b", "a", "e", "c", "d", "f"}, ties: []string{"resolved", "resolved"}},
		{name: "both covered, kept order", requirement: "FR-020", path: "happy", breaks: [][]string{{"a", "b"}, {"c", "d", "e"}}, wantOrder: []string{"a", "b", "c", "d", "e", "f"}, ties: []string{"resolved", "resolved"}},
		{name: "partial", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"a"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"a"}, combat.TieBreakPartial}}, ties: []string{"rejected", "unresolved"}},
		{name: "repeated id", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"a", "a"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"a", "a"}, combat.TieBreakRepeatedID}}, ties: []string{"rejected", "unresolved"}},
		{name: "spans two groups", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"a", "b", "c"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"a", "b", "c"}, combat.TieBreakSpansGroups}}, ties: []string{"rejected", "rejected"}},
		{name: "all five span", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"a", "b", "c", "d", "e"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"a", "b", "c", "d", "e"}, combat.TieBreakSpansGroups}}, ties: []string{"rejected", "rejected"}},
		{name: "not tied", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"f"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"f"}, combat.TieBreakNotTied}}, ties: []string{"unresolved", "unresolved", "rejected"}},
		{name: "unknown ids", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"x", "y"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"x", "y"}, combat.TieBreakUnknownID}}, ties: []string{"unresolved", "unresolved", "rejected"}},
		{name: "group covered twice", requirement: "FR-021", path: "unhappy (invalid input)", breaks: [][]string{{"a", "b"}, {"b", "a"}, {"c", "d", "e"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"b", "a"}, combat.TieBreakDuplicateGroup}}, ties: []string{"rejected", "resolved"}},
		{name: "unknown beats partial", requirement: "FR-120", path: "unhappy (rule interaction)", breaks: [][]string{{"a", "x"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"a", "x"}, combat.TieBreakUnknownID}}, ties: []string{"rejected", "unresolved"}},
		{name: "repeated beats not tied", requirement: "FR-120", path: "unhappy (rule interaction)", breaks: [][]string{{"a", "a", "f"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"a", "a", "f"}, combat.TieBreakRepeatedID}}, ties: []string{"rejected", "unresolved"}},
		{name: "two rejections keep the submitted order", requirement: "FR-120", path: "unhappy (rule interaction)", breaks: [][]string{{"c"}, {"a"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"c"}, combat.TieBreakPartial}, {[]string{"a"}, combat.TieBreakPartial}}, ties: []string{"rejected", "rejected"}},
		{name: "one valid, one rejected", requirement: "FR-120", path: "unhappy (rule interaction)", breaks: [][]string{{"a", "b"}, {"f"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"f"}, combat.TieBreakNotTied}}, ties: []string{"resolved", "unresolved", "rejected"}},
		{name: "two lists naming no group", requirement: "FR-102", path: "unhappy (rule interaction)", breaks: [][]string{{"x"}, {"f"}},
			wantRejected: []struct {
				ids    []string
				reason combat.TieBreakRejectionReason
			}{{[]string{"x"}, combat.TieBreakUnknownID}, {[]string{"f"}, combat.TieBreakNotTied}}, ties: []string{"unresolved", "unresolved", "rejected", "rejected"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var breaks []combat.TieBreak
			for _, b := range c.breaks {
				breaks = append(breaks, trkBreak(t, b...))
			}
			rec, opts := recording()
			res := trkEstablish(t, trkStandings(15, 15, 10, 10, 10, 5), breaks, opts...)
			switch v := res.(type) {
			case combat.InitiativeOrderEstablished:
				if c.wantOrder == nil || !slices.Equal(trkOrderIDs(v.Tracker().Order()), c.wantOrder) {
					t.Errorf("order = %v, want %v [%s]", trkOrderIDs(v.Tracker().Order()), c.wantOrder, c.requirement)
				}
			case combat.InitiativeTiesUnresolved:
				var got groupsT
				for _, g := range v.Groups() {
					got = append(got, ids2s(g.IDs()))
				}
				if !reflect.DeepEqual(got, c.wantUnresolved) {
					t.Errorf("unresolved = %v, want %v [%s]", got, c.wantUnresolved, c.requirement)
				}
			case combat.InitiativeTieBreakRejected:
				if len(v.Rejections()) != len(c.wantRejected) {
					t.Fatalf("rejections = %+v, want %d", v.Rejections(), len(c.wantRejected))
				}
				for i, r := range v.Rejections() {
					if !slices.Equal(ids2s(r.IDs()), c.wantRejected[i].ids) || r.Reason() != c.wantRejected[i].reason {
						t.Errorf("rejection %d = %v %v, want %v %v [%s]", i, r.IDs(), r.Reason(), c.wantRejected[i].ids, c.wantRejected[i].reason, c.requirement)
					}
				}
			}
			gotTies := values(measured(rec, "srd5e.combat.initiative_ties"), "srd5e.combat.tie_outcome")
			wantTies := slices.Clone(c.ties)
			slices.Sort(wantTies)
			if !slices.Equal(gotTies, wantTies) {
				t.Errorf("initiative_ties = %v, want %v [FR-102]", gotTies, wantTies)
			}
			trkHygiene(t, rec)
		})
	}
}

func ids2s(ids []combat.CombatantID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// Requirement: doc/specs/initiative-and-actions.md FR-018, FR-022, FR-023, FR-027 - combatants rank from the highest count to the lowest whatever the listing order; one combatant is an order of one; the tracker starts in round 1 with the first active and reports TrackerRoundStarted(1) then TrackerTurnStarted.
// Path: happy.
func TestEstablishInitiativeOrderRanksAndStarts(t *testing.T) {
	t.Parallel()
	for _, counts := range [][]int{{5, 20, -3, 11}, {-3, 11, 5, 20}, {20, 11, 5, -3}} {
		res := trkEstablish(t, trkStandings(counts...), nil).(combat.InitiativeOrderEstablished)
		got := res.Tracker().Order().Standings()
		for i := 1; i < len(got); i++ {
			if got[i-1].Count <= got[i].Count {
				t.Errorf("order %v is not ranked high to low [FR-018]", got)
			}
		}
		if res.Tracker().Round() != 1 || res.Tracker().Active().Count != 20 || res.Tracker().Next().Count != 11 {
			t.Errorf("tracker starts at round %d with %v active, want round 1 and the 20 first [FR-027]", res.Tracker().Round(), res.Tracker().Active())
		}
		if kinds := trkKinds(res.Events()); !slices.Equal(kinds, []string{"tracker_round_started", "tracker_turn_started"}) {
			t.Errorf("events = %v [FR-027]", kinds)
		}
	}
	one := trkEstablish(t, trkStandings(7), nil).(combat.InitiativeOrderEstablished)
	if one.Tracker().Order().Len() != 1 || one.Tracker().Next() != one.Tracker().Active() {
		t.Errorf("order of one = %v [FR-023]", one.Tracker().Order().Standings())
	}
	ev := one.Events()
	if r := ev[0].(combat.TrackerRoundStarted).Round(); r != 1 || ev[1].(combat.TrackerTurnStarted).Round() != 1 {
		t.Errorf("round of the start events = %d", r)
	}
	ev[0] = nil
	if one.Events()[0] == nil {
		t.Error("events share their slice [FR-128]")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-024, FR-025, FR-014 - no combatant, a blank or repeated id, a missing, extra, repeated or sizeless profile, or a zero tie-break is an invalid-input error from the constructor.
// Path: unhappy (invalid input).
func TestNewEstablishInitiativeOrderCommandRejects(t *testing.T) {
	t.Parallel()
	st := trkStandings(10, 5)
	pr := trkProfiles(st)
	cases := map[string]struct {
		st []combat.InitiativeStanding
		pr []combat.CombatantProfile
		tb []combat.TieBreak
	}{
		"none":            {nil, nil, nil},
		"blank id":        {[]combat.InitiativeStanding{{Combatant: "", Count: 1}}, nil, nil},
		"id twice":        {[]combat.InitiativeStanding{st[0], st[0]}, pr, nil},
		"missing profile": {st, pr[:1], nil},
		"extra profile":   {st[:1], pr, nil},
		"profile twice":   {st, []combat.CombatantProfile{pr[0], pr[0]}, nil},
		"profile no size": {st, []combat.CombatantProfile{pr[0], {ID: st[1].Combatant}}, nil},
		"zero tie-break":  {st, pr, []combat.TieBreak{{}}},
	}
	for name, c := range cases {
		if _, err := combat.NewEstablishInitiativeOrderCommand(c.st, c.pr, c.tb); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want invalid input [FR-024, FR-025]", name, err)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-099, FR-100, FR-103, FR-124 - each result is recorded as its class: established is applied with the two start events; ties unresolved is refused without error status; a rejected tie-break is invalid with error status and no srd5e.combat.errors; no turn, round or action counter exists.
// Path: per case.
func TestEstablishInitiativeOrderSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, result, class, outcome string
		counts                       []int
		breaks                       [][]string
		isErr                        bool
		events                       []string
	}{
		{"established", "established", "applied", "success", []int{20, 10}, nil, false, []string{"tracker_round_started", "tracker_turn_started"}},
		{"ties unresolved", "ties_unresolved", "refused", "failure", []int{10, 10}, nil, false, nil},
		{"tie-break rejected", "tie_break_rejected", "invalid", "failure", []int{10, 10}, [][]string{{"a"}}, true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var breaks []combat.TieBreak
			for _, b := range c.breaks {
				breaks = append(breaks, trkBreak(t, b...))
			}
			rec, opts := recording()
			res := trkEstablish(t, trkStandings(c.counts...), breaks, opts...)
			want := map[string]any{"srd5e.combat.operation": "establish_initiative_order", "srd5e.outcome": c.outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat.establish_initiative_order")
			if len(spans) != 1 || (spans[0].Status == telemetry.StatusError) != c.isErr || len(spans[0].Errors) != 0 || !reflect.DeepEqual(attrMap(spans[0].Attrs), want) {
				t.Errorf("spans = %+v, want one span with %v, error status %v", spans, want, c.isErr)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want %v", ops, want)
			}
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, func() []string { k := slices.Clone(c.events); slices.Sort(k); return append([]string{}, k...) }()) {
				t.Errorf("events counted = %v, want %v [FR-103]", kinds, c.events)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result [FR-124]", n)
			}
			for _, banned := range []string{"turns_started", "rounds_started", "turn_actions"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + banned); len(m) != 0 {
					t.Errorf("%s recorded [FR-103]", banned)
				}
			}
			plain := trkEstablish(t, trkStandings(c.counts...), breaks)
			if !reflect.DeepEqual(res, plain) {
				t.Errorf("recording %#v and Noop %#v disagree [FR-107]", res, plain)
			}
			trkHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-017, FR-026, FR-096 - the command, the result lists and the tracker are copies; the zero command is invalid input and a cancelled ctx returns ctx.Err(), both before any signal.
// Path: unhappy (immutability, cancellation).
func TestEstablishInitiativeOrderCopiesAndGuards(t *testing.T) {
	t.Parallel()
	st := trkStandings(10, 10, 4)
	pr := trkProfiles(st)
	tb := []combat.TieBreak{trkBreak(t, "a", "b")}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(st, pr, tb)
	if err != nil {
		t.Fatal(err)
	}
	st[0].Count, pr[0].Size, tb[0] = 99, content.SizeTiny, combat.TieBreak{}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	est := res.(combat.InitiativeOrderEstablished)
	if s, _ := est.Tracker().ProfileOf(st[0].Combatant); s.Size != content.SizeMedium || est.Tracker().Active().Count != 10 {
		t.Errorf("the command shares its lists with the caller [FR-096]")
	}
	rec, opts := recording()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := combat.NewEstablishInitiativeOrder(opts...).Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v with %d spans, want ctx.Err() and nothing", res, err, len(rec.Tracer.Spans()))
	}
	if _, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), combat.EstablishInitiativeOrderCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v, want invalid input", err)
	}
	unresolved := trkEstablish(t, trkStandings(5, 5), nil).(combat.InitiativeTiesUnresolved)
	g := unresolved.Groups()
	g[0] = combat.TiedGroup{}
	if unresolved.Groups()[0].Count() != 5 {
		t.Error("Groups shares its slice")
	}
	ids := unresolved.Groups()[0].IDs()
	ids[0] = "z"
	if unresolved.Groups()[0].IDs()[0] != "a" {
		t.Error("TiedGroup shares its ids")
	}
	rej := trkEstablish(t, trkStandings(5, 5), []combat.TieBreak{trkBreak(t, "a")}).(combat.InitiativeTieBreakRejected)
	rej.Rejections()[0] = combat.TieBreakRejection{}
	rj := rej.Rejections()[0].IDs()
	rj[0] = "z"
	if rej.Rejections()[0].IDs()[0] != "a" {
		t.Error("TieBreakRejection shares its ids")
	}
}
