package combat_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

var teleNone = combat.ConditionSet{}

func teleStunned(tb testing.TB) combat.ConditionSet {
	tb.Helper()
	return condSet(tb, condConds(combat.Stunned{}))
}

func teleUnconscious(tb testing.TB) combat.ConditionSet {
	tb.Helper()
	return condSet(tb, condConds(combat.Unconscious{}))
}

// teleBase is the baseline of case table 4.1: telepath with 60 feet, distance 30, no conditions, contacted understands a language.
func teleBase() combat.InitiateTelepathicContactCommand {
	return combat.InitiateTelepathicContactCommand{Telepath: gid("t"), Contacted: gid("c"), Capability: combat.MustTelepathic(60),
		DistanceFeet: 30, Understanding: combat.ContactedUnderstandingALanguage}
}

func teleMade(tb testing.TB, r combat.InitiateTelepathicContactResult) combat.InitiateTelepathicContactMade {
	tb.Helper()
	m, ok := r.(combat.InitiateTelepathicContactMade)
	if !ok {
		tb.Fatalf("result = %#v, want Made", r)
	}
	return m
}

func teleEventKinds(events []combat.TelepathyEvent) []string {
	var kinds []string
	for _, e := range events {
		kinds = append(kinds, e.Kind())
	}
	return kinds
}

// Requirement: per case (doc/specs/telepathy.md case table 4.1)
// Path: per case.
func TestInitiateTelepathicContact(t *testing.T) {
	t.Parallel()
	rej := func(r combat.TelepathicContactRejection) combat.InitiateTelepathicContactResult {
		return combat.InitiateTelepathicContactRejected{Reason: r}
	}
	const (
		notTele = combat.TelepathicContactRejectionInitiatorNotTelepathic
		iInc    = combat.TelepathicContactRejectionInitiatorIncapacitated
		cInc    = combat.TelepathicContactRejectionContactedIncapacitated
		oor     = combat.TelepathicContactRejectionContactedOutOfRange
		cannot  = combat.TelepathicContactRejectionContactedCannotUnderstand
	)
	cases := []struct {
		name, requirement, path string
		change                  func(*combat.InitiateTelepathicContactCommand)
		want                    combat.InitiateTelepathicContactResult // nil: Made
	}{
		{"baseline", "FR-002", "happy", func(*combat.InitiateTelepathicContactCommand) {}, nil},
		{"at range", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = 60 }, nil},
		{"one beyond", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = 61 }, rej(oor)},
		{"distance zero", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = 0 }, nil},
		{"range 1, distance 1", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Capability = combat.MustTelepathic(1)
			c.DistanceFeet = 1
		}, nil},
		{"range 1, distance 2", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Capability = combat.MustTelepathic(1)
			c.DistanceFeet = 2
		}, rej(oor)},
		{"largest distance", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = math.MaxInt }, rej(oor)},
		{"largest range", "FR-003", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Capability = combat.MustTelepathic(math.MaxInt)
			c.DistanceFeet = math.MaxInt
		}, nil},
		{"unseen contacted (no sight fact)", "FR-007", "unhappy (rule interaction)", func(*combat.InitiateTelepathicContactCommand) {}, nil},
		{"no shared language", "FR-004", "unhappy (rule interaction)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Understanding = combat.ContactedUnderstandingALanguage
		}, nil},
		{"telepathic, no language", "FR-006", "unhappy (boundary)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Understanding = combat.ContactedUnderstandingTelepathic
		}, nil},
		{"neither", "FR-005", "unhappy (state error)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Understanding = combat.ContactedUnderstandingNeither
		}, rej(cannot)},
		{"undefined understanding", "FR-005", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { c.Understanding = 9 }, rej(cannot)},
		{"initiator not telepathic", "FR-008", "unhappy (state error)", func(c *combat.InitiateTelepathicContactCommand) { c.Capability = combat.TelepathyCapability{} }, rej(notTele)},
		{"initiator Incapacitated (Stunned)", "FR-009", "unhappy (state error)", func(c *combat.InitiateTelepathicContactCommand) { c.TelepathConditions = teleStunned(t) }, rej(iInc)},
		{"contacted Incapacitated (Unconscious)", "FR-010", "unhappy (state error)", func(c *combat.InitiateTelepathicContactCommand) { c.ContactedConditions = teleUnconscious(t) }, rej(cInc)},
		{"everything wrong", "FR-011", "unhappy (rule interaction)", func(c *combat.InitiateTelepathicContactCommand) {
			c.Capability = combat.TelepathyCapability{}
			c.TelepathConditions, c.ContactedConditions = teleStunned(t), teleStunned(t)
			c.DistanceFeet, c.Understanding = 61, combat.ContactedUnderstandingNeither
		}, rej(notTele)},
		{"both Incapacitated, out of range", "FR-011", "unhappy (rule interaction)", func(c *combat.InitiateTelepathicContactCommand) {
			c.TelepathConditions, c.ContactedConditions, c.DistanceFeet = teleStunned(t), teleStunned(t), 61
		}, rej(iInc)},
		{"contacted Incapacitated, out of range", "FR-011", "unhappy (rule interaction)", func(c *combat.InitiateTelepathicContactCommand) {
			c.ContactedConditions, c.DistanceFeet = teleStunned(t), 61
		}, rej(cInc)},
		{"out of range and cannot understand", "FR-011", "unhappy (rule interaction)", func(c *combat.InitiateTelepathicContactCommand) {
			c.DistanceFeet, c.Understanding = 61, combat.ContactedUnderstandingNeither
		}, rej(oor)},
		{"blank telepath", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { c.Telepath = "" }, rej(combat.TelepathicContactRejectionInvalidCommand)},
		{"blank contacted", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { c.Contacted = "" }, rej(combat.TelepathicContactRejectionInvalidCommand)},
		{"contact with itself", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { c.Contacted = c.Telepath }, rej(combat.TelepathicContactRejectionInvalidCommand)},
		{"negative distance", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { c.DistanceFeet = -1 }, rej(combat.TelepathicContactRejectionInvalidCommand)},
		{"undefined policy", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { c.ReplyPolicy = 5 }, rej(combat.TelepathicContactRejectionInvalidCommand)},
		{"state of another telepath", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) {
			c.State = combat.InContact(gid("x"), gid("y"), 60, combat.ContactedMayReply)
		}, rej(combat.TelepathicContactRejectionInvalidCommand)},
		{"zero command", "FR-024", "unhappy (invalid input)", func(c *combat.InitiateTelepathicContactCommand) { *c = combat.InitiateTelepathicContactCommand{} }, rej(combat.TelepathicContactRejectionInvalidCommand)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := teleBase()
			c.change(&cmd)
			got := combat.InitiateTelepathicContact(cmd)
			if c.want != nil {
				if got != c.want {
					t.Errorf("result = %#v, want %#v [%s]", got, c.want, c.requirement)
				}
				return
			}
			m := teleMade(t, got)
			if want := combat.InContact(gid("t"), gid("c"), cmd.Capability.RangeFeet(), cmd.ReplyPolicy); m.State() != want {
				t.Errorf("state = %+v, want %+v [%s]", m.State(), want, c.requirement)
			}
			if k := teleEventKinds(m.Events()); !reflect.DeepEqual(k, []string{"telepathic_contact_made"}) {
				t.Errorf("events = %v, want one telepathic_contact_made [%s]", k, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/telepathy.md FR-012 - starting and ending contact spend no action: the events are exactly the listed TelepathicContact events (the name scan is in internal/archtest).
// Path: happy.
func TestTelepathyTakesNoAction(t *testing.T) {
	t.Parallel()
	m := teleMade(t, combat.InitiateTelepathicContact(teleBase()))
	end := combat.EndTelepathicContact(m.State(), gid("t")).(combat.EndTelepathicContactApplied)
	if len(m.Events()) != 1 || len(end.Events()) != 1 {
		t.Fatalf("events = %d, %d, want 1, 1", len(m.Events()), len(end.Events()))
	}
	if _, ok := m.Events()[0].(combat.TelepathicContactMade); !ok {
		t.Errorf("initiate event = %T", m.Events()[0])
	}
	if _, ok := end.Events()[0].(combat.TelepathicContactEnded); !ok {
		t.Errorf("end event = %T", end.Events()[0])
	}
}

// Requirement: doc/specs/telepathy.md FR-013, FR-014 - ending a contact gives NoContact and one Ended event; ending with no contact, or another telepath's, changes nothing.
// Path: per case.
func TestEndTelepathicContact(t *testing.T) {
	t.Parallel()
	s := combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayReply)
	applied, ok := combat.EndTelepathicContact(s, gid("t")).(combat.EndTelepathicContactApplied)
	if !ok || applied.State().IsInContact() || !reflect.DeepEqual(teleEventKinds(applied.Events()), []string{"telepathic_contact_ended"}) {
		t.Errorf("end by telepath = %#v", applied)
	}
	if got := applied.Events()[0].Combatants(); got[0] != gid("t") || got[1] != gid("c") {
		t.Errorf("event creatures = %v, want [t c]", got)
	}
	for name, res := range map[string]combat.EndTelepathicContactResult{
		"no contact [FR-014]": combat.EndTelepathicContact(combat.TelepathicContactState{}, gid("t")),
		"the contacted ends":  combat.EndTelepathicContact(s, gid("c")),
		"a stranger ends":     combat.EndTelepathicContact(s, gid("z")),
		"blank creature ends": combat.EndTelepathicContact(s, ""),
	} {
		if res != (combat.EndTelepathicContactNoContact{}) {
			t.Errorf("%s: result = %#v, want NoContact", name, res)
		}
	}
}

// Requirement: doc/specs/telepathy.md FR-015 - either creature of the pair being Incapacitated breaks the contact (cause incapacitated); a stranger, a creature that is not Incapacitated or NoContact leaves it unchanged.
// Path: unhappy (rule interaction).
func TestReportTelepathIncapacitated(t *testing.T) {
	t.Parallel()
	s := combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayReply)
	for _, who := range []string{"t", "c"} {
		b, ok := combat.ReportTelepathIncapacitated(s, gid(who), teleStunned(t)).(combat.ReportTelepathIncapacitatedBroken)
		if !ok || b.State().IsInContact() || len(b.Events()) != 1 {
			t.Fatalf("%s Incapacitated = %#v", who, b)
		}
		ev, isBroken := b.Events()[0].(combat.TelepathicContactBroken)
		if !isBroken || ev.Cause() != combat.BreakCauseIncapacitated {
			t.Errorf("%s event = %#v, want Broken with BreakCauseIncapacitated", who, b.Events()[0])
		}
	}
	// Paralyzed implies Incapacitated (conditions.md FR-013).
	if _, ok := combat.ReportTelepathIncapacitated(s, gid("t"), condSet(t, condConds(combat.Paralyzed{}))).(combat.ReportTelepathIncapacitatedBroken); !ok {
		t.Error("Paralyzed must break the contact")
	}
	unchanged := combat.ReportTelepathIncapacitatedUnchanged{}
	for name, res := range map[string]combat.ReportTelepathIncapacitatedResult{
		"stranger":          combat.ReportTelepathIncapacitated(s, gid("z"), teleStunned(t)),
		"not incapacitated": combat.ReportTelepathIncapacitated(s, gid("t"), condSet(t, condConds(combat.Blinded{}))),
		"empty set":         combat.ReportTelepathIncapacitated(s, gid("c"), teleNone),
		"no contact":        combat.ReportTelepathIncapacitated(combat.TelepathicContactState{}, gid("t"), teleStunned(t)),
		"blank creature":    combat.ReportTelepathIncapacitated(s, "", teleStunned(t)),
	} {
		if res != unchanged {
			t.Errorf("%s: result = %#v, want Unchanged", name, res)
		}
	}
}

// Requirement: doc/specs/telepathy.md FR-016 - a distance beyond the range breaks the contact (cause contacted_left_range); at the range, below it, negative or with no contact it does not.
// Path: unhappy (boundary).
func TestReportTelepathDistance(t *testing.T) {
	t.Parallel()
	s := combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayReply)
	for _, d := range []int{61, 1000, math.MaxInt} {
		b, ok := combat.ReportTelepathDistance(s, d).(combat.ReportTelepathDistanceBroken)
		if !ok || b.State().IsInContact() || len(b.Events()) != 1 {
			t.Fatalf("distance %d = %#v, want Broken", d, b)
		}
		if ev, isBroken := b.Events()[0].(combat.TelepathicContactBroken); !isBroken || ev.Cause() != combat.BreakCauseContactedLeftRange {
			t.Errorf("distance %d event = %#v", d, b.Events()[0])
		}
	}
	for _, d := range []int{60, 59, 0, -1, math.MinInt} {
		if res := combat.ReportTelepathDistance(s, d); res != (combat.ReportTelepathDistanceUnchanged{}) {
			t.Errorf("distance %d = %#v, want Unchanged", d, res)
		}
	}
	if res := combat.ReportTelepathDistance(combat.TelepathicContactState{}, 1000); res != (combat.ReportTelepathDistanceUnchanged{}) {
		t.Errorf("no contact = %#v, want Unchanged", res)
	}
}

// Requirement: doc/specs/telepathy.md FR-017, FR-018 - contacting a different creature breaks the old contact first and makes the new one; contacting the same creature again is AlreadyMade with nothing changed.
// Path: unhappy (rule interaction).
func TestInitiateWhileInContact(t *testing.T) {
	t.Parallel()
	cmd := teleBase()
	cmd.State = combat.InContact(gid("t"), gid("old"), 30, combat.ContactedMayNotReply)
	m := teleMade(t, combat.InitiateTelepathicContact(cmd))
	if k := teleEventKinds(m.Events()); !reflect.DeepEqual(k, []string{"telepathic_contact_broken", "telepathic_contact_made"}) {
		t.Fatalf("events = %v, want broken then made", k)
	}
	broken := m.Events()[0].(combat.TelepathicContactBroken)
	if broken.Cause() != combat.BreakCauseTelepathContactedAnother || broken.Combatants()[1] != gid("old") || m.Events()[1].Combatants()[1] != gid("c") {
		t.Errorf("events = %#v", m.Events())
	}
	if want := combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayReply); m.State() != want {
		t.Errorf("state = %+v, want %+v", m.State(), want)
	}
	cmd.State = combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayReply)
	if res := combat.InitiateTelepathicContact(cmd); res != (combat.InitiateTelepathicContactAlreadyMade{}) {
		t.Errorf("same creature = %#v, want AlreadyMade", res)
	}
	// A contact that fails the rules is rejected, and does not break the old one.
	cmd.State, cmd.DistanceFeet, cmd.Contacted = combat.InContact(gid("t"), gid("old"), 30, 0), 61, gid("c")
	if res := combat.InitiateTelepathicContact(cmd); res != (combat.InitiateTelepathicContactRejected{Reason: combat.TelepathicContactRejectionContactedOutOfRange}) {
		t.Errorf("rejected switch = %#v", res)
	}
}

// Requirement: doc/specs/telepathy.md FR-019, FR-020, FR-023 - the contacted creature may reply by default (telepathic or not); NotPermitted under ContactedMayNotReply; NoContact without a contact; NotContacted for the telepath or a stranger.
// Path: per case.
func TestTelepathicReply(t *testing.T) {
	t.Parallel()
	may := combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayReply)
	mayNot := combat.InContact(gid("t"), gid("c"), 60, combat.ContactedMayNotReply)
	none := combat.TelepathicContactState{}
	cases := []struct {
		name, requirement string
		state             combat.TelepathicContactState
		who               string
		want              combat.TelepathicReplyResult
	}{
		{"contacted replies", "FR-019", may, "c", combat.TelepathicReplyAllowed{}},
		{"no contact", "FR-020", none, "c", combat.TelepathicReplyNoContact{}},
		{"telepath replies", "FR-020", may, "t", combat.TelepathicReplyNotContacted{}},
		{"stranger replies", "FR-020", may, "z", combat.TelepathicReplyNotContacted{}},
		{"blank replies", "FR-020", may, "", combat.TelepathicReplyNotContacted{}},
		{"policy forbids reply", "FR-023", mayNot, "c", combat.TelepathicReplyNotPermitted{}},
		{"policy: telepath is still not contacted", "FR-023", mayNot, "t", combat.TelepathicReplyNotContacted{}},
		{"policy: no contact", "FR-023", none, "c", combat.TelepathicReplyNoContact{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.TelepathicReply(c.state, combat.CombatantID(c.who)); got != c.want {
				t.Errorf("reply = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/telepathy.md FR-023 - the policy changes nothing but the reply: initiation stores it, ending and breaking ignore it.
// Path: unhappy (rule interaction).
func TestReplyPolicyOnlyAffectsReplies(t *testing.T) {
	t.Parallel()
	cmd := teleBase()
	cmd.ReplyPolicy = combat.ContactedMayNotReply
	m := teleMade(t, combat.InitiateTelepathicContact(cmd))
	if m.State().ReplyPolicy() != combat.ContactedMayNotReply {
		t.Fatalf("policy = %v, want stored", m.State().ReplyPolicy())
	}
	if _, ok := combat.EndTelepathicContact(m.State(), gid("t")).(combat.EndTelepathicContactApplied); !ok {
		t.Error("ending must ignore the policy")
	}
	if _, ok := combat.ReportTelepathDistance(m.State(), 61).(combat.ReportTelepathDistanceBroken); !ok {
		t.Error("breaking by distance must ignore the policy")
	}
	if _, ok := combat.ReportTelepathIncapacitated(m.State(), gid("c"), teleStunned(t)).(combat.ReportTelepathIncapacitatedBroken); !ok {
		t.Error("breaking by Incapacitated must ignore the policy")
	}
}

// Requirement: doc/specs/telepathy.md FR-022 - inputs are unchanged, returned event slices are copies, and equal inputs give equal results.
// Path: unhappy (immutability).
func TestTelepathyOperationsAreImmutableAndDeterministic(t *testing.T) {
	t.Parallel()
	cmd := teleBase()
	cmd.State = combat.InContact(gid("t"), gid("old"), 30, 0)
	copyOfCmd := cmd
	first, second := combat.InitiateTelepathicContact(cmd), combat.InitiateTelepathicContact(cmd)
	if cmd != copyOfCmd || !reflect.DeepEqual(first, second) {
		t.Error("InitiateTelepathicContact changed its input or gave different results")
	}
	m := first.(combat.InitiateTelepathicContactMade)
	evs := m.Events()
	evs[0] = nil
	if m.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
	s := combat.InContact(gid("t"), gid("c"), 60, 0)
	for _, evs := range [][]combat.TelepathyEvent{
		combat.EndTelepathicContact(s, gid("t")).(combat.EndTelepathicContactApplied).Events(),
		combat.ReportTelepathDistance(s, 61).(combat.ReportTelepathDistanceBroken).Events(),
		combat.ReportTelepathIncapacitated(s, gid("t"), teleStunned(t)).(combat.ReportTelepathIncapacitatedBroken).Events(),
	} {
		evs[0] = nil
	}
	if s != combat.InContact(gid("t"), gid("c"), 60, 0) {
		t.Error("an operation changed its state input")
	}
	again := combat.EndTelepathicContact(s, gid("t")).(combat.EndTelepathicContactApplied)
	if again.Events()[0] == nil {
		t.Error("result events share storage with an earlier result")
	}
}

// Requirement: doc/specs/telepathy.md FR-001, FR-024 - initiation takes untrusted numbers and ids: for any input it returns one of the three variants, never panics, and a Made state always holds a valid pair within the range.
// Path: unhappy (invalid input).
func FuzzInitiateTelepathicContact(f *testing.F) {
	f.Add("t", "c", 60, 30, 1, 0, "")
	f.Add("", "", 0, -1, 0, 9, "")
	f.Add("t", "t", math.MaxInt, math.MaxInt, 2, 1, "x")
	f.Add("t", "c", 1, 2, 3, -1, "t")
	f.Fuzz(func(t *testing.T, tel, con string, rng, dist, und, pol int, stateTel string) {
		cap, _ := combat.NewTelepathic(rng)
		cmd := combat.InitiateTelepathicContactCommand{Telepath: combat.CombatantID(tel), Contacted: combat.CombatantID(con), Capability: cap,
			DistanceFeet: dist, Understanding: combat.ContactedUnderstanding(und), ReplyPolicy: combat.ReplyPolicy(pol),
			State: combat.InContact(combat.CombatantID(stateTel), gid("old"), 10, 0)}
		switch r := combat.InitiateTelepathicContact(cmd).(type) {
		case combat.InitiateTelepathicContactMade:
			s := r.State()
			if !s.IsInContact() || s.RangeFeet() != rng || dist > rng || dist < 0 || len(r.Events()) == 0 {
				t.Fatalf("Made with state %+v for %+v", s, cmd)
			}
		case combat.InitiateTelepathicContactRejected:
			if r.Reason.String() == "unknown" {
				t.Fatalf("rejection %d is undefined", r.Reason)
			}
		case combat.InitiateTelepathicContactAlreadyMade:
		default:
			t.Fatalf("result %T is not a variant", r)
		}
	})
}

// Requirement: doc/specs/telepathy.md FR-018, FR-021, FR-024 - the id checks (same creature, another telepath's state, the same contact again, a valid pair) hold whichever id sorts first.
// Path: unhappy (boundary).
func TestTelepathyIDComparisonsDoNotDependOnOrder(t *testing.T) {
	t.Parallel()
	invalid := combat.InitiateTelepathicContactRejected{Reason: combat.TelepathicContactRejectionInvalidCommand}
	for _, p := range [][2]string{{"a", "b"}, {"b", "a"}} {
		x, y := gid(p[0]), gid(p[1])
		cmd := teleBase()
		cmd.Telepath, cmd.Contacted = x, x
		if got := combat.InitiateTelepathicContact(cmd); got != invalid {
			t.Errorf("%v: same creature = %#v, want invalid", p, got)
		}
		cmd.Contacted = y
		cmd.State = combat.InContact(y, gid("z"), 60, 0)
		if got := combat.InitiateTelepathicContact(cmd); got != invalid {
			t.Errorf("%v: other telepath's state = %#v, want invalid", p, got)
		}
		cmd.State = combat.InContact(x, y, 60, 0)
		if got := combat.InitiateTelepathicContact(cmd); got != (combat.InitiateTelepathicContactAlreadyMade{}) {
			t.Errorf("%v: same contact = %#v, want AlreadyMade", p, got)
		}
		cmd.State = combat.InContact(x, gid("0"), 60, 0)
		if m := teleMade(t, combat.InitiateTelepathicContact(cmd)); len(m.Events()) != 2 {
			t.Errorf("%v: switch events = %d, want 2", p, len(m.Events()))
		}
		if !combat.InContact(x, y, 1, 0).IsInContact() || combat.InContact(x, x, 1, 0).IsInContact() {
			t.Errorf("%v: InContact pair check", p)
		}
		if _, err := combat.NewTelepathicContactMade(x, y); err != nil {
			t.Errorf("%v: valid pair rejected: %v", p, err)
		}
		if _, err := combat.NewTelepathicContactMade(x, x); err == nil {
			t.Errorf("%v: equal pair accepted", p)
		}
	}
}

// Requirement: doc/specs/telepathy.md FR-021 - every defined BreakCause builds an event and the first value below and above the range does not.
// Path: unhappy (boundary).
func TestTelepathicContactBrokenCauseBoundaries(t *testing.T) {
	t.Parallel()
	for c := combat.BreakCauseIncapacitated; c <= combat.BreakCauseTelepathContactedAnother; c++ {
		e, err := combat.NewTelepathicContactBroken(gid("a"), gid("b"), c)
		if err != nil || e.Cause() != c {
			t.Errorf("cause %v = %v, %v", c, e, err)
		}
	}
}
