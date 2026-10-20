package content_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: equipSpec FR-007, FR-030, FR-107 - tool constructors copy the
// lists they are given, and a family lists the sites of its variants followed
// by those of its Craft list (existence only).
// Path: unhappy (caller mutates after construction) and happy (references).
func TestToolConstructorsCopyInputs(t *testing.T) {
	t.Parallel()
	item, _ := content.NewCraftItem("club_x")
	utilize := testUtilize()
	craft := []content.CraftReference{item}
	variants := []content.ItemID{"v_a"}
	fam, err := content.NewToolFamily("family_x", "Family X", noSource, content.ToolGroupOther, ability.Charisma, utilize, craft, variants, content.CostVaries{}, content.WeightVaries{})
	if err != nil {
		t.Fatal(err)
	}
	utilize[0] = content.ToolUtilizeAction{}
	craft[0] = nil
	variants[0] = "z"
	if fam.Utilize()[0].DC() != 10 || fam.Craft()[0] == nil || fam.Variants()[0] != "v_a" {
		t.Error("mutating the constructor arguments changed the family [FR-007]")
	}
	if got, want := refFields(fam.References()), []string{"variants/0=v_a", "craft/0/id=club_x"}; !slices.Equal(got, want) {
		t.Errorf("ToolFamily references = %v, want %v [FR-107]", got, want)
	}
}
