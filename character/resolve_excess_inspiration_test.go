package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

type gainCase struct {
	name, requirement, path string
	holder                  character.HeroicInspiration
	give                    string // "" lose, "r" give to the recipient, "h" give to the holder's id
	recipient               character.HeroicInspiration
	want                    string // received, lost, given, or the rejection
	change                  int
}

var gainCases = []gainCase{
	{"G1 not held, lose", "FR-003", "happy", character.NotInspired, "", character.NotInspired, "received", 1},
	{"G2 not held, give (choice not consulted)", "FR-009", "unhappy (rule interaction)", character.NotInspired, "r", character.NotInspired, "received", 1},
	{"G3 not held, give to own id", "FR-009", "unhappy (rule interaction)", character.NotInspired, "h", character.NotInspired, "received", 1},
	{"G4 not held, recipient already inspired", "FR-009", "unhappy (rule interaction)", character.NotInspired, "r", character.Inspired, "received", 1},
	{"G5 held, lose excess", "FR-004", "unhappy (rule interaction)", character.Inspired, "", character.NotInspired, "lost", 0},
	{"G6 held, give to one who lacks it", "FR-005", "happy", character.Inspired, "r", character.NotInspired, "given", 1},
	{"G7 held, recipient already inspired", "FR-006", "unhappy (state error)", character.Inspired, "r", character.Inspired, "recipient_already_inspired", 0},
	{"G8 held, give to self", "FR-007", "unhappy (invalid input)", character.Inspired, "h", character.NotInspired, "recipient_is_holder", 0},
	{"G9 held, give to self, both conditions", "FR-008", "unhappy (rule interaction)", character.Inspired, "h", character.Inspired, "recipient_is_holder", 0},
}

// choiceFor builds the choice of a gain case; "h" names a recipient with the
// holder's id.
func choiceFor(t *testing.T, c gainCase) (character.ExcessInspirationChoice, character.Character, character.Character) {
	t.Helper()
	holder, recipient := pc(t, "holder", c.holder), pc(t, "friend", c.recipient)
	switch c.give {
	case "r":
		return character.GiveExcessInspirationTo{Recipient: recipient}, holder, recipient
	case "h":
		self := pc(t, "holder", c.recipient)
		return character.GiveExcessInspirationTo{Recipient: self}, holder, self
	}
	return character.LoseExcessInspiration{}, holder, recipient
}

// Requirement: doc/specs/heroic-inspiration.md FR-003 to FR-010, FR-038, table G1 to
// G9 - ResolveExcessInspiration returns the variant, holder, recipient and events
// of each gaining case, reads the choice only for an Inspired holder, checks
// RecipientIsHolder before RecipientAlreadyInspired, and never raises the number of
// Inspired characters by more than one (SRD 5.2.1, "Heroic Inspiration", p. 8;
// "Rules Glossary", p. 183).
// Path: per case.
func TestResolveExcessInspirationCases(t *testing.T) {
	t.Parallel()
	for _, c := range gainCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			choice, holder, recipient := choiceFor(t, c)
			before := inspiredCount(holder, recipient)
			if c.give == "h" {
				before = inspiredCount(holder)
			}
			res := character.ResolveExcessInspiration(holder, choice)
			var after int
			switch r := res.(type) {
			case character.ExcessResolutionReceived:
				ev := r.Events()
				if c.want != "received" || r.Holder().HeroicInspiration() != character.Inspired || !isGainedBy(ev, holder.ID()) {
					t.Errorf("received = %#v [%s]", r, c.requirement)
				}
				after = inspiredCount(r.Holder(), recipient)
				if c.give == "h" {
					after = inspiredCount(r.Holder())
				}
			case character.ExcessResolutionExcessLost:
				if c.want != "lost" || !r.Holder().Equal(holder) || len(r.Events()) != 0 {
					t.Errorf("lost = %#v [%s]", r, c.requirement)
				}
				after = inspiredCount(r.Holder(), recipient)
			case character.ExcessResolutionExcessGiven:
				ev := r.Events()
				if c.want != "given" || !r.Holder().Equal(holder) || r.Recipient().HeroicInspiration() != character.Inspired ||
					!isGainedBy(ev, recipient.ID()) {
					t.Errorf("given = %#v [%s]", r, c.requirement)
				}
				after = inspiredCount(r.Holder(), r.Recipient())
			case character.ExcessResolutionRejected:
				if r.Reason().String() != c.want {
					t.Errorf("rejected %v, want %s [%s]", r.Reason(), c.want, c.requirement)
				}
				after = before
			}
			if after-before != c.change {
				t.Errorf("Inspired count changed by %d, want %d [FR-010]", after-before, c.change)
			}
		})
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-038 - a nil choice is a loss for
// an Inspired holder and is not read for one who is not; the function does not
// alter its inputs and is deterministic.
// Path: unhappy (invalid input).
func TestResolveExcessInspirationNilChoiceAndPurity(t *testing.T) {
	t.Parallel()
	held, free := pc(t, "holder", character.Inspired), pc(t, "holder", character.NotInspired)
	if _, ok := character.ResolveExcessInspiration(held, nil).(character.ExcessResolutionExcessLost); !ok {
		t.Error("nil choice for an Inspired holder is not ExcessLost")
	}
	if _, ok := character.ResolveExcessInspiration(free, nil).(character.ExcessResolutionReceived); !ok {
		t.Error("nil choice for a holder who lacks it is not Received")
	}
	friend := pc(t, "friend", character.NotInspired)
	choice := character.GiveExcessInspirationTo{Recipient: friend}
	a, b := character.ResolveExcessInspiration(held, choice), character.ResolveExcessInspiration(held, choice)
	if !reflect.DeepEqual(a, b) || !held.Equal(pc(t, "holder", character.Inspired)) || !friend.Equal(pc(t, "friend", character.NotInspired)) {
		t.Error("ResolveExcessInspiration is not pure")
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-011 - the Events slice of Received,
// ExcessLost (empty) and ExcessGiven is a copy: changing it leaves the value
// unchanged.
// Path: unhappy (immutability).
func TestResolveExcessInspirationEventsAreCopies(t *testing.T) {
	t.Parallel()
	free, held, friend := pc(t, "holder", character.NotInspired), pc(t, "holder", character.Inspired), pc(t, "friend", character.NotInspired)
	for name, res := range map[string]interface{ Events() []character.Event }{
		"received": character.ResolveExcessInspiration(free, nil).(character.ExcessResolutionReceived),
		"given":    character.ResolveExcessInspiration(held, character.GiveExcessInspirationTo{Recipient: friend}).(character.ExcessResolutionExcessGiven),
	} {
		res.Events()[0] = nil
		if res.Events()[0] == nil {
			t.Errorf("%s: Events shares its storage", name)
		}
	}
	lost := character.ResolveExcessInspiration(held, nil).(character.ExcessResolutionExcessLost)
	grown := append(lost.Events(), nil)
	if len(lost.Events()) != 0 || len(grown) != 1 {
		t.Error("lost: Events is not an independent empty slice")
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-035, FR-038 - the gain rejection
// enum has exactly RecipientIsHolder and RecipientAlreadyInspired, with String
// values the telemetry uses; the zero and unknown values are "unknown".
// Path: happy.
func TestGainHeroicInspirationRejectionStrings(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.GainHeroicInspirationRejection]string{
		character.RecipientIsHolder: "recipient_is_holder", character.RecipientAlreadyInspired: "recipient_already_inspired", 0: "unknown", 3: "unknown",
	} {
		if r.String() != want {
			t.Errorf("%d.String() = %q, want %q", r, r.String(), want)
		}
	}
}
