package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func craftWorker(id string, tools ...string) character.CraftingWorker {
	return character.NewCraftingWorker(character.MustIdentifier(id), character.NewToolProficiencies(idsOf(tools...)...))
}

// Requirement: tool-proficiency-and-crafting.md FR-023 (SRD 5.2.1, "Time",
// p. 103) - a party is immutable: helpers are copied on construction and on read.
// Path: unhappy (immutability).
func TestCraftingPartyCopiesHelpers(t *testing.T) {
	t.Parallel()
	in := []character.CraftingWorker{craftWorker("h1", "smiths_tools")}
	p, err := character.NewCraftingParty(craftWorker("c"), in...)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = craftWorker("changed")
	got := p.Helpers()
	got[0] = craftWorker("changed too")
	if again := p.Helpers(); len(again) != 1 || again[0].ID() != character.MustIdentifier("h1") {
		t.Errorf("helpers = %v, want the helper h1 unchanged", again)
	}
	if p.Crafter().ID() != character.MustIdentifier("c") || p.IsZero() || !p.Helpers()[0].Tools().Contains("smiths_tools") {
		t.Error("accessors are wrong")
	}
	if !(character.CraftingParty{}).IsZero() {
		t.Error("the zero party must report IsZero")
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-023 (architecture.md
// section 6) - a repeated or zero id is an invalid-input error; MustCraftingParty
// panics with it.
// Path: per case (unhappy invalid input).
func TestCraftingPartyRejectsBadIDs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		crafter character.CraftingWorker
		helpers []character.CraftingWorker
	}{
		{"helper equal to the crafter", craftWorker("a"), []character.CraftingWorker{craftWorker("a")}},
		{"two equal helpers", craftWorker("a"), []character.CraftingWorker{craftWorker("b"), craftWorker("b")}},
		{"zero crafter id", character.CraftingWorker{}, nil},
		{"zero helper id", craftWorker("a"), []character.CraftingWorker{{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := character.NewCraftingParty(c.crafter, c.helpers...)
			wantInvalid(t, err, c.name)
			defer func() {
				e, ok := recover().(error)
				if !ok || !errors.Is(e, srderr.ErrInvalidInput) {
					t.Errorf("MustCraftingParty panic = %v, want an error matching ErrInvalidInput", e)
				}
			}()
			character.MustCraftingParty(c.crafter, c.helpers...)
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-027 - the String of CraftingRole.
// Path: unhappy (invalid input).
func TestCraftingRoleString(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.CraftingRole]string{character.CraftingRoleCrafter: "crafter", character.CraftingRoleHelper: "helper", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("role %d = %q, want %q", int(r), r.String(), want)
		}
	}
}
