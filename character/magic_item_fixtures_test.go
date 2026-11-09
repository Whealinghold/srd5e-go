package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

var holder = character.MustIdentifier("pc-1")

func inst(s string) character.MagicItemInstanceID {
	i, err := character.NewMagicItemInstanceID(s)
	if err != nil {
		panic(err)
	}
	return i
}

func mid(s string) content.MagicItemID { return content.MustMagicItemID(s) }

// restOf resolves a rest of kind that ran its course (hours) or was stopped.
func restOf(t *testing.T, kind character.RestKind, end character.RestEnd) (character.RestKind, character.RestResolution) {
	t.Helper()
	report := must(character.NewRestReport(kind, character.StandardLongRest, end))
	return kind, must(character.ResolveRest(report))
}

// focusOf builds a Short Rest focus on instance for purpose, completed or not.
func focusOf(t *testing.T, instance string, purpose character.RestPurpose, contact character.PhysicalContact, completed bool) character.ShortRestFocus {
	t.Helper()
	end := character.RestEnd(character.NewRestRanItsCourse(1, 0))
	if !completed {
		end = character.NewRestStopped(character.TakingDamage, 0, 0)
	}
	kind, res := restOf(t, character.ShortRest, end)
	return must(character.NewShortRestFocus(inst(instance), purpose, contact, kind, res))
}

func attunable(t *testing.T, item, instance string, a content.Attunement) character.AttunableItem {
	t.Helper()
	return must(character.NewAttunableItem(mid(item), inst(instance), a))
}

func requiredBy(t *testing.T, text string) content.AttunementRequiredBy {
	t.Helper()
	return must(content.NewAttunementRequiredBy(text))
}

func candidate(t *testing.T, cast character.OwnSpellcasting, classes ...string) character.AttuneCandidate {
	t.Helper()
	ids := make([]content.ClassID, len(classes))
	for i, c := range classes {
		ids[i] = classOf(c)
	}
	return must(character.NewAttuneCandidate(ids, cast))
}

func attunedSet(t *testing.T, pairs ...string) character.AttunementSet {
	t.Helper()
	var items []character.AttunedItem
	for i := 0; i < len(pairs); i += 2 {
		items = append(items, must(character.NewAttunedItem(mid(pairs[i]), inst(pairs[i+1]))))
	}
	return must(character.NewAttunementSet(items))
}
