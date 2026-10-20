package content

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
)

// Requirement: equipSpec FR-030, FR-108 - a tool body re-validates each Utilize
// action's DC and each Craft set's exceptions on their own; a bad DC with a
// text that passes, or a bad exception inside a valid set name, cannot be built
// outside the package.
// Path: unhappy.
func TestToolBodyRevalidatesParts(t *testing.T) {
	t.Parallel()
	if validCraftReference(CraftSet{set: CraftHeavyArmor, except: []ItemID{"Bad"}}) {
		t.Error("a Craft set with an invalid exception id is valid [FR-107]")
	}
	if !validCraftReference(CraftSet{set: CraftHeavyArmor, except: []ItemID{"ok_item"}}) {
		t.Error("a Craft set with a valid exception id is not valid [FR-107]")
	}
	_, err := newToolBody("StandaloneTool", ToolGroupOther, ability.Strength, []ToolUtilizeAction{{description: "x", dc: 0}}, nil, NoWeight{})
	var be *BlockError
	if !errors.As(err, &be) || be.Problem != BlockInvalidMember {
		t.Errorf("a Utilize action with DC 0 = %v, want BlockInvalidMember [FR-108]", err)
	}
}
