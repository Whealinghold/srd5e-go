package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// pc returns a level 1 Fighter with the given id and Heroic Inspiration state.
func pc(t *testing.T, id string, state character.HeroicInspiration) character.Character {
	t.Helper()
	p := newParams(t, "fighter", flatScores(t, 10))
	p.ID = character.MustIdentifier(id)
	c := character.MustNew(p).Character()
	if state == character.Inspired {
		c = c.GrantHeroicInspiration().Character()
	}
	return c
}

// inspiredCount is how many of the characters hold Heroic Inspiration.
func inspiredCount(cs ...character.Character) int {
	n := 0
	for _, c := range cs {
		if c.HeroicInspiration() == character.Inspired {
			n++
		}
	}
	return n
}

// isGainedBy reports whether events is exactly one HeroicInspirationGained for id.
func isGainedBy(events []character.Event, id character.Identifier) bool {
	if len(events) != 1 {
		return false
	}
	g, ok := events[0].(character.HeroicInspirationGained)
	return ok && g.CharacterID() == id
}
