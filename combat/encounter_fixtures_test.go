package combat_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

const encSpec = "doc/specs/encounter-aggregate.md "

func eid() combat.EncounterID { return combat.MustEncounterID("enc-1") }

// encTr is the method set of the unexported transition the Encounter commands return.
type encTr interface {
	Encounter() combat.Encounter
	Events() []combat.Event
	Resolutions() []combat.DamageResolution
	Rejection() combat.EncounterRejection
}

// encEstablished establishes the order of the ids, first highest, Medium and walk 30.
func encEstablished(tb testing.TB, ids ...string) combat.InitiativeOrderEstablished {
	tb.Helper()
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for i, s := range ids {
		id := combat.MustCombatantID(s)
		standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(100 - 10*i)})
		profiles = append(profiles, trkProfile(tb, id, content.SizeMedium, trkSpeeds(tb, 30)))
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		tb.Fatalf("NewEstablishInitiativeOrderCommand: %v", err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	est, ok := res.(combat.InitiativeOrderEstablished)
	if !ok {
		tb.Fatalf("result = %#v, want InitiativeOrderEstablished", res)
	}
	return est
}

// encC builds a combatant: a monster kind on the side with the status and conditions.
func encC(tb testing.TB, id string, side combat.Side, hp combat.HitPointStatus, conds ...combat.Condition) combat.Combatant {
	tb.Helper()
	ref := combat.CombatantRef{ID: combat.MustCombatantID(id), Kind: combat.MonsterCombatant{ID: content.MonsterID("goblin")}}
	c, err := combat.NewCombatant(ref, side, hp, condSet(tb, conds))
	if err != nil {
		tb.Fatalf("NewCombatant: %v", err)
	}
	return c
}

// encPC is a living combatant with cur of max Hit Points that falls unconscious.
func encPC(tb testing.TB, id string, side combat.Side, cur, max int) combat.Combatant {
	tb.Helper()
	return encC(tb, id, side, pcStatus(tb, cur, max, 0))
}

// encStart starts an encounter whose turn order is the ids of the combatants in the order given.
func encStart(tb testing.TB, cs ...combat.Combatant) (combat.Encounter, []combat.Event) {
	tb.Helper()
	ids := make([]string, len(cs))
	for i, c := range cs {
		ids[i] = string(c.Ref().ID)
	}
	e, events, err := (combat.Encounter{}).Start(eid(), encEstablished(tb, ids...), cs)
	if err != nil {
		tb.Fatalf("Start: %v", err)
	}
	return e, events
}

// encABC is a started encounter: a and c on side A, b on side B, all healthy, a active.
func encABC(tb testing.TB) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encPC(tb, "a", combat.SideA, 20, 20), encPC(tb, "b", combat.SideB, 20, 20), encPC(tb, "c", combat.SideA, 20, 20))
	return e
}

// encWith starts a, b with the given first combatant state (a) and a healthy b on the other side.
func encWith(tb testing.TB, a combat.Combatant) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, a, encPC(tb, "b", combat.SideB, 20, 20))
	return e
}

func cid(s string) combat.CombatantID { return combat.MustCombatantID(s) }

// encKinds lists the Kind of each event.
func encKinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// encOK takes the result of a command and returns the new encounter and events, failing the test on an error or a rejection.
func encOK[T encTr](tb testing.TB, tr T, err error) (combat.Encounter, []combat.Event) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("command error: %v", err)
	}
	if r := tr.Rejection(); r != nil {
		tb.Fatalf("rejected with %#v, want applied", r)
	}
	return tr.Encounter(), tr.Events()
}

// encNo takes the result of a command and returns the rejection, failing the test on an error or when applied; it asserts the encounter is unchanged and no event came with it.
func encNo[T encTr](tb testing.TB, before combat.Encounter, tr T, err error) combat.EncounterRejection {
	tb.Helper()
	if err != nil {
		tb.Fatalf("command error: %v", err)
	}
	r := tr.Rejection()
	if r == nil {
		tb.Fatalf("applied, want a rejection")
	}
	if !tr.Encounter().Equal(before) || len(tr.Events()) != 0 || len(tr.Resolutions()) != 0 {
		tb.Errorf("a rejection must carry the unchanged encounter and no events [FR-048]")
	}
	return r
}

// encMember returns the combatant with the id.
func encMember(tb testing.TB, e combat.Encounter, id string) combat.Combatant {
	tb.Helper()
	c, ok := e.Find(cid(id))
	if !ok {
		tb.Fatalf("no combatant %q", id)
	}
	return c
}

// encOthersEqual asserts every combatant but id is equal in both encounters (FR-047),.
func encOthersEqual(tb testing.TB, before, after combat.Encounter, id string) {
	tb.Helper()
	b, a := before.Combatants(), after.Combatants()
	if len(a) != len(b) {
		tb.Fatalf("combatants changed in number: %d to %d", len(b), len(a))
	}
	for i := range b {
		if b[i].Ref().ID != cid(id) && !a[i].Equal(b[i]) {
			tb.Errorf("combatant %v changed though the command named %s [FR-047]", b[i].Ref().ID, id)
		}
		if a[i].Ref() != b[i].Ref() {
			tb.Errorf("the order of the combatants changed [FR-015]")
		}
	}
	if !before.Tracker().Equal(after.Tracker()) || before.Status() != after.Status() {
		tb.Errorf("the tracker or status changed [FR-047]")
	}
}

// encLife is the life state of the combatant as a short name.
func encLife(c combat.Combatant) string {
	switch l := c.HitPoints().LifeState().(type) {
	case combat.Alive:
		return "alive"
	case combat.Dying:
		return "dying"
	case combat.Stable:
		return "stable"
	case combat.Dead:
		return "dead:" + l.Cause().String()
	}
	return "?"
}

// encHas reports whether the combatant holds the kind effectively.
func encHas(c combat.Combatant, k content.ConditionKind) bool {
	return slices.Contains(c.Conditions().Effective(), k)
}

// encExplicit reports whether the combatant holds the kind explicitly.
func encExplicit(c combat.Combatant, k content.ConditionKind) bool {
	return slices.ContainsFunc(c.Conditions().Explicit(), func(x combat.Condition) bool { return x.Kind() == k })
}

func deepEqual(a, b any) bool { return reflect.DeepEqual(a, b) }
