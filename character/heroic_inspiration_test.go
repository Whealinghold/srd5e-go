package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/character-aggregate.md FR-089 - HeroicInspiration has
// exactly Inspired and NotInspired; the zero value is invalid; a created
// character defaults to NotInspired.
// Path: happy.
func TestHeroicInspirationEnum(t *testing.T) {
	t.Parallel()
	if !character.Inspired.Valid() || !character.NotInspired.Valid() || character.HeroicInspiration(0).Valid() || character.HeroicInspiration(3).Valid() {
		t.Error("only the two values are valid")
	}
	if character.Inspired.String() != "inspired" || character.NotInspired.String() != "notInspired" || character.HeroicInspiration(0).String() != "unknown" {
		t.Error("strings")
	}
	if got := created(t, "fighter", flatScores(t, 10)).HeroicInspiration(); got != character.NotInspired {
		t.Errorf("default = %v", got)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-090 - granting from NotInspired
// gives Inspired and one HeroicInspirationGained; granting when Inspired returns
// an equal character and no events (SRD 5.2.1, Rules Glossary, p. 183).
// Path: unhappy (boundary).
func TestGrantHeroicInspiration(t *testing.T) {
	t.Parallel()
	c := created(t, "fighter", flatScores(t, 10))
	g := c.GrantHeroicInspiration()
	evs := g.Events()
	if g.Character().HeroicInspiration() != character.Inspired || len(evs) != 1 || !isGained(evs[0], c) {
		t.Errorf("grant: %v %v", g.Character().HeroicInspiration(), evs)
	}
	if c.HeroicInspiration() != character.NotInspired {
		t.Error("the receiver must not change")
	}
	again := g.Character().GrantHeroicInspiration()
	if !again.Character().Equal(g.Character()) || len(again.Events()) != 0 {
		t.Error("granting when Inspired must be a no-op without events")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-091 - expending from Inspired
// gives NotInspired and one HeroicInspirationExpended; from NotInspired it is
// rejected with HeroicInspirationNotAvailable, no events, input unchanged
// (SRD 5.2.1, "Heroic Inspiration", p. 8).
// Path: unhappy (state error).
func TestExpendHeroicInspiration(t *testing.T) {
	t.Parallel()
	c := created(t, "fighter", flatScores(t, 10))
	rej, ok := c.ExpendHeroicInspiration().(character.TransitionRejected[character.HeroicInspirationRejection])
	if !ok || rej.Rejection() != (character.HeroicInspirationNotAvailable{}) {
		t.Fatalf("expend when not inspired: %#v", rej)
	}
	inspired := c.GrantHeroicInspiration().Character()
	ch, ok := inspired.ExpendHeroicInspiration().(character.TransitionChanged[character.HeroicInspirationRejection])
	if !ok || ch.Character().HeroicInspiration() != character.NotInspired || len(ch.Events()) != 1 {
		t.Fatalf("expend when inspired: %#v", ch)
	}
	if _, isExpended := ch.Events()[0].(character.HeroicInspirationExpended); !isExpended || ch.Events()[0].CharacterID() != c.ID() {
		t.Errorf("event = %#v", ch.Events()[0])
	}
	if inspired.HeroicInspiration() != character.Inspired {
		t.Error("the receiver must not change")
	}
}

func isGained(e character.Event, c character.Character) bool {
	_, ok := e.(character.HeroicInspirationGained)
	return ok && e.CharacterID() == c.ID()
}
