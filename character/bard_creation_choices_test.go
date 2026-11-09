package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-bard.md FR-117, FR-121 - the creation choices of a Bard and the multiclass choices seed a full pool sized from the Charisma modifier they carry under the typed key, leave every other entry and the argument alone, and
// return states that already hold the key unchanged; both are ClassCreationChoices that name no content.
// Path: per case (happy / unhappy: state error).
func TestBardChoicesSeedThePool(t *testing.T) {
	t.Parallel()
	instruments := map[content.ItemID]bool{"lute": true}
	multi := must(character.NewBardMulticlassChoices([]content.Skill{content.SkillArcana}, []content.ItemID{"lute"}, brdMod(t, 16), nil, instruments))
	for name, choices := range map[string]character.ClassCreationChoices{"creation": character.NewBardCreationChoices(brdMod(t, 16)), "multiclass": multi} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			start := character.FeatureStates{}
			seeded := choices.SeedFeatureStates(start)
			p, ok, err := character.ReadFeatureState(seeded, character.BardicInspirationPoolKey())
			if err != nil || !ok || p.Maximum() != 3 || p.Remaining() != 3 {
				t.Fatalf("pool = %v, %v, %v; want a full pool of 3", p, ok, err)
			}
			if _, held, _ := character.ReadFeatureState(start, character.BardicInspirationPoolKey()); held {
				t.Error("the argument was altered")
			}
			if again := choices.SeedFeatureStates(seeded); !again.Equal(seeded) {
				t.Error("seeding states that hold the key changed them")
			}
			if keys := choices.ContentKeys(); keys == nil || len(keys) != 0 {
				t.Errorf("ContentKeys = %v, want an empty non-nil slice", keys)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-015, FR-110, FR-122 (SRD 5.2.1, "As a Multiclass Character", p. 31) - a multiclass entry names exactly one skill and exactly one Musical Instrument; a skill the character already has, and an item
// outside the caller's Musical Instrument set, are refused; every problem is listed together and a valid entry is accepted.
// Path: per case (happy / unhappy: invalid input).
func TestNewBardMulticlassChoices(t *testing.T) {
	t.Parallel()
	instruments := map[content.ItemID]bool{"lute": true, "flute": true}
	held := []content.Skill{content.SkillAthletics}
	arcana, athletics := []content.Skill{content.SkillArcana}, []content.Skill{content.SkillAthletics}
	cases := []struct {
		name   string
		skills []content.Skill
		item   []content.ItemID
		want   []character.BardChoiceProblem
	}{
		{"valid", arcana, []content.ItemID{"lute"}, nil},
		{"no skill", nil, []content.ItemID{"lute"}, []character.BardChoiceProblem{character.BardSkillCountWrong}},
		{"two skills", []content.Skill{content.SkillArcana, content.SkillHistory}, []content.ItemID{"lute"}, []character.BardChoiceProblem{character.BardSkillCountWrong}},
		{"held skill", athletics, []content.ItemID{"lute"}, []character.BardChoiceProblem{character.BardSkillAlreadyProficient}},
		{"no instrument", arcana, nil, []character.BardChoiceProblem{character.BardInstrumentCountWrong}},
		{"two instruments", arcana, []content.ItemID{"lute", "flute"}, []character.BardChoiceProblem{character.BardInstrumentCountWrong}},
		{"not an instrument", arcana, []content.ItemID{"longsword"}, []character.BardChoiceProblem{character.BardInstrumentNotMusical}},
		{"everything wrong", athletics[:0], []content.ItemID{"longsword"}, []character.BardChoiceProblem{character.BardSkillCountWrong, character.BardInstrumentNotMusical}},
		{"held skill and no instrument", athletics, nil, []character.BardChoiceProblem{character.BardSkillAlreadyProficient, character.BardInstrumentCountWrong}},
		{"invalid skill", []content.Skill{99}, []content.ItemID{"lute"}, []character.BardChoiceProblem{character.BardSkillInvalid}},
		{"zero skill", []content.Skill{0}, []content.ItemID{"lute"}, []character.BardChoiceProblem{character.BardSkillInvalid}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewBardMulticlassChoices(c.skills, c.item, brdMod(t, 14), held, instruments)
			if c.want == nil {
				if err != nil || !slices.Equal(got.Skills(), c.skills) || got.Instrument() != "lute" {
					t.Fatalf("= %+v, %v; want accepted", got, err)
				}
				return
			}
			var perr *character.BardMulticlassChoicesError
			if !errors.As(err, &perr) || !errors.Is(err, srderr.ErrInvalidInput) || !slices.Equal(perr.Problems, c.want) || perr.Error() == "" {
				t.Fatalf("error = %v, want problems %v", err, c.want)
			}
			if len(got.Skills()) != 0 || got.Instrument() != "" {
				t.Errorf("a refused choice returned %+v", got)
			}
		})
	}
}
