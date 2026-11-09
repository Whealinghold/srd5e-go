package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-049 - SubclassChosen
// is an immutable, past-tense Event and LevelAdvancementEvent that carries the
// character id, class, subclass and class level, and compares by value.
// Path: happy.
func TestSubclassChosenIsALevelAdvancementEvent(t *testing.T) {
	t.Parallel()
	ch := levelsChar(t, cl("cleric", 4))
	tr, err := character.SubclassSelection{}.Choose(ch, advCleric(t, "life_domain"), "life_domain")
	if err != nil {
		t.Fatal(err)
	}
	ev := tr.(character.TransitionChanged[character.SubclassRejected]).Events()[0]
	var fam character.LevelAdvancementEvent
	fam, ok := ev.(character.LevelAdvancementEvent)
	if !ok {
		t.Fatalf("%T is not a LevelAdvancementEvent", ev)
	}
	got := fam.(character.SubclassChosen)
	if fam.CharacterID() != ch.ID() || got.Class() != content.ClassID("cleric") || got.Subclass() != content.SubclassID("life_domain") || got.ClassLevel() != 4 {
		t.Errorf("event = %#v", got)
	}
	if got != tr.(character.TransitionChanged[character.SubclassRejected]).Events()[0] {
		t.Error("an event must compare by value")
	}
}
