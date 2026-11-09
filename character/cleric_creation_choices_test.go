package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-cleric.md FR-135, FR-140 - NewClericCreationChoices applies the ChooseDivineOrder rules (without already_chosen) and refuses with a
// *ClericCreationChoicesError carrying the first reason, matching srderr.ErrInvalidInput; a valid choice seeds the validated order and the level 1 resources.
// Path: per case (unhappy: invalid input; happy).
func TestNewClericCreationChoices(t *testing.T) {
	t.Parallel()
	light := clcLight(t)
	level1 := clcSpell(t, "bless", 1, []string{"cleric"}, "")
	for _, c := range []struct {
		name   string
		names  []string
		extra  *content.Spell
		known  []content.SpellID
		reason character.ClericReason
	}{
		{"Protector", []string{"protector"}, nil, nil, 0}, {"Thaumaturge", []string{"thaumaturge"}, &light, nil, 0},
		{"missing", nil, nil, nil, character.ClericOrderMissing}, {"both", []string{"protector", "thaumaturge"}, nil, nil, character.ClericBothOrders},
		{"unknown", []string{"x"}, nil, nil, character.ClericOrderUnknown}, {"no cantrip", []string{"thaumaturge"}, nil, nil, character.ClericExtraCantripMissing},
		{"level 1 cantrip", []string{"thaumaturge"}, &level1, nil, character.ClericExtraCantripNotClericCantrip},
		{"known", []string{"thaumaturge"}, &light, []content.SpellID{"light"}, character.ClericExtraCantripAlreadyKnown},
	} {
		got, err := character.NewClericCreationChoices(c.names, c.extra, c.known)
		var rej *character.ClericCreationChoicesError
		if c.reason == 0 {
			if err != nil {
				t.Errorf("%s: error %v", c.name, err)
			}
			continue
		}
		if !errors.As(err, &rej) || rej.Reason != c.reason || !errors.Is(err, &character.ClericCreationChoicesError{Reason: c.reason}) ||
			errors.Is(err, &character.ClericCreationChoicesError{Reason: c.reason + 1}) || rej.Error() == "" || got != (character.ClericCreationChoices{}) {
			t.Errorf("%s: = %v, %v; want reason %v", c.name, got, err, c.reason)
		}
		wantInvalid(t, err, c.name)
	}
}

// Requirement: doc/specs/class-cleric.md FR-140 - SeedFeatureStates adds ClericChoices (the validated order, the extra cantrip for a Thaumaturge) and the initial
// ClericResources (0 of 0, Divine Intervention available), keeps every other entry and leaves the input unchanged; it returns states unchanged when either key is held; ContentKeys names the extra cantrip for a Thaumaturge and nothing for a Protector, as a new slice.
// Path: per case (happy / unhappy: state error).
func TestClericCreationChoicesSeedAndKeys(t *testing.T) {
	t.Parallel()
	light := clcLight(t)
	th := must(character.NewClericCreationChoices([]string{"thaumaturge"}, &light, nil))
	pr := must(character.NewClericCreationChoices([]string{"protector"}, nil, nil))
	other := must(character.SetFeatureState(character.FeatureStates{}, character.FighterStateKey(), character.InitialFighterState()))
	for _, c := range []struct {
		name  string
		in    character.ClericCreationChoices
		order character.DivineOrder
		keys  int
	}{{"Thaumaturge", th, character.DivineOrderThaumaturge, 1}, {"Protector", pr, character.DivineOrderProtector, 0}} {
		seeded := c.in.SeedFeatureStates(other)
		ch, okC, _ := character.ReadFeatureState(seeded, character.ClericChoicesKey())
		res, okR, _ := character.ReadFeatureState(seeded, character.ClericResourcesKey())
		_, okF, _ := character.ReadFeatureState(seeded, character.FighterStateKey())
		if !okC || !okR || !okF || ch.DivineOrder() != c.order || ch.BlessedStrikes() != 0 || !res.Equal(character.InitialClericResources(0)) || other.Len() != 1 || seeded.Len() != 3 {
			t.Errorf("%s: seeded %d entries, choices %+v, resources %+v", c.name, seeded.Len(), ch, res)
		}
		keys := c.in.ContentKeys()
		if len(keys) != c.keys || keys == nil || (c.keys == 1 && keys[0] != content.NewKey(content.KindSpell, content.MustID("light"))) {
			t.Errorf("%s: keys = %v", c.name, keys)
		}
	}
	held := must(character.SetFeatureState(character.FeatureStates{}, character.ClericChoicesKey(), character.ClericChoices{}))
	heldRes := must(character.SetFeatureState(character.FeatureStates{}, character.ClericResourcesKey(), character.ClericResources{}))
	for name, states := range map[string]character.FeatureStates{"choices held": held, "resources held": heldRes} {
		if got := pr.SeedFeatureStates(states); !got.Equal(states) {
			t.Errorf("%s: states were altered", name)
		}
	}
}
