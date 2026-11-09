package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/character-creation.md FR-011, FR-012, FR-010 - Run does not alter its command, every collection of the result and
// record is a copy, a rejection cannot be empty, and the same command gives an equal result.
// Path: unhappy (immutability, determinism).
func TestCreateCharacterKeepsCommandAndCopiesCollections(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.SpeciesID, cmd.SpeciesSelections = "dragonborn", []character.SpeciesTraitSelection{{Trait: "Draconic Ancestry", Option: "Gold"}}
	before := crCommand()
	before.SpeciesID, before.SpeciesSelections = "dragonborn", []character.SpeciesTraitSelection{{Trait: "Draconic Ancestry", Option: "Gold"}}
	first := crCreated(t, crMust(t, cmd))
	if !reflect.DeepEqual(cmd, before) {
		t.Error("Run altered its command")
	}
	first.Record().Skills()[0] = content.SkillArcana
	first.Record().SpeciesSelections()[0].Option = "Red"
	first.Record().Equipment().Items()[0] = content.EquipmentItem{}
	first.Events()[0] = nil
	if second := crCreated(t, crMust(t, cmd)); !reflect.DeepEqual(second.Record(), first.Record()) || !reflect.DeepEqual(second.Events(), first.Events()) {
		t.Error("the same command gave an unequal result, or a returned slice reached the stored value")
	}
	if got := first.Record().Skills()[0]; got != content.SkillAthletics {
		t.Errorf("Skills()[0] = %v after a caller changed a copy", got)
	}
	if _, err := character.NewCreateCharacterRejected(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("empty rejection error = %v", err)
	}
	rej, err := character.NewCreateCharacterRejected([]character.CreateCharacterProblem{{}})
	if err != nil || len(rej.Problems()) != 1 {
		t.Fatalf("NewCreateCharacterRejected = %+v, %v", rej, err)
	}
	rej.Problems()[0] = character.CreateCharacterProblem{}
}
