package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-druid.md FR-163, FR-169, FR-170 - NewDruidCreationChoices accepts the Magician and the Warden and refuses the zero order and any other value with ErrInvalidInput; the choices seed a
// level 1 DruidFeatureState holding the order and nothing else under the key "druid", keep every other entry, leave the input unchanged, return states that already hold the key unchanged, and name no content.
// Path: per case (happy / unhappy: invalid input, state error, immutability).
func TestDruidCreationChoices(t *testing.T) {
	t.Parallel()
	for _, o := range []character.PrimalOrder{character.PrimalOrderMagician, character.PrimalOrderWarden} {
		c, err := character.NewDruidCreationChoices(o)
		if err != nil || c.PrimalOrder() != o {
			t.Fatalf("NewDruidCreationChoices(%v) = %v, %v", o, c, err)
		}
		other := must(character.SetFeatureState(character.FeatureStates{}, character.RogueStateKey(), character.InitialRogueState()))
		seeded := c.SeedFeatureStates(other)
		got, held, rerr := character.ReadFeatureState(seeded, character.DruidFeatureStateKey())
		_, keptOther, _ := character.ReadFeatureState(seeded, character.RogueStateKey())
		if rerr != nil || !held || !keptOther || got != (druState(t, 1, false, character.DruidStateFields{PrimalOrder: o})) {
			t.Errorf("seeded %v: %+v held %v kept %v err %v", o, got, held, keptOther, rerr)
		}
		if _, heldBefore, _ := character.ReadFeatureState(other, character.DruidFeatureStateKey()); heldBefore {
			t.Error("the input states were changed")
		}
		if again := c.SeedFeatureStates(seeded); !again.Equal(seeded) {
			t.Error("states already holding the key were changed")
		}
		if keys := c.ContentKeys(); keys == nil || len(keys) != 0 {
			t.Errorf("ContentKeys = %v, want an empty slice", keys)
		}
	}
	for _, o := range []character.PrimalOrder{0, 3, -1} {
		if _, err := character.NewDruidCreationChoices(o); err == nil {
			t.Errorf("order %v was accepted", o)
		} else {
			wantInvalid(t, err, "order")
		}
	}
	if got := (character.DruidCreationChoices{}).SeedFeatureStates(character.FeatureStates{}); got.Equal(character.FeatureStates{}) {
		t.Error("the zero choices seed nothing")
	}
}
