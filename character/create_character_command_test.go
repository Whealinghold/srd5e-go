package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/character-creation.md section 6 and go.md section 3 - a zero use case, a missing source or roller, a zero or invalid
// required field are invalid-input errors before any lookup or signal, and a cancelled context is ctx.Err().
// Path: per case.
func TestCreateCharacterInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := character.NewCreateCharacter(); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewCreateCharacter() error = %v", err)
	}
	if _, err := (character.CreateCharacter{}).Run(context.Background(), crCommand()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case error = %v", err)
	}
	mutations := map[string]func(*character.CreateCharacterCommand){
		"zero id":                func(c *character.CreateCharacterCommand) { c.CharacterID = character.Identifier{} },
		"invalid class":          func(c *character.CreateCharacterCommand) { c.ClassID = "" },
		"invalid background":     func(c *character.CreateCharacterCommand) { c.BackgroundID = "Bad Id" },
		"invalid species":        func(c *character.CreateCharacterCommand) { c.SpeciesID = "" },
		"zero scores":            func(c *character.CreateCharacterCommand) { c.AbilityScores = character.AbilityScoreAssignment{} },
		"zero plus two plus one": func(c *character.CreateCharacterCommand) { c.BackgroundIncrease = character.PlusTwoPlusOne{} },
		"nil increase":           func(c *character.CreateCharacterCommand) { c.BackgroundIncrease = nil },
		"zero class letter":      func(c *character.CreateCharacterCommand) { c.ClassEquipment = 0 },
		"zero background letter": func(c *character.CreateCharacterCommand) { c.BackgroundEquipment = 4 },
		"zero alignment":         func(c *character.CreateCharacterCommand) { c.Alignment = 0 },
		"invalid skill":          func(c *character.CreateCharacterCommand) { c.ClassSkills = []content.Skill{0} },
		"invalid class tool":     func(c *character.CreateCharacterCommand) { c.ClassTools = []character.ChosenTool{{}} },
		"invalid language":       func(c *character.CreateCharacterCommand) { c.Languages = []content.Language{0, 1} },
		"invalid chosen tool":    func(c *character.CreateCharacterCommand) { c.BackgroundTool = character.ChosenTool{} },
		"invalid chosen size":    func(c *character.CreateCharacterCommand) { c.Size = character.ChosenSize{} },
		"foreign tool selection": func(c *character.CreateCharacterCommand) { c.BackgroundTool = &character.ChosenTool{Tool: "x"} },
		"foreign size choice":    func(c *character.CreateCharacterCommand) { c.Size = &character.ChosenSize{} },
		"foreign trinket choice": func(c *character.CreateCharacterCommand) { c.Trinket = &character.RollForTrinket{} },
		"foreign equipment": func(c *character.CreateCharacterCommand) {
			c.HigherLevelEquipment = &character.GrantHigherLevelEquipment{}
		},
		"step without a choice": func(c *character.CreateCharacterCommand) {
			c.Options, c.Plan = startingLevel(2), []character.StartingLevelStep{{ClassID: "fighter"}}
		},
		"step with a bad class": func(c *character.CreateCharacterCommand) {
			c.Options, c.Plan = startingLevel(2), []character.StartingLevelStep{{HitPointChoice: character.RollHitDie{}}}
		},
		"step with a bad subclass": func(c *character.CreateCharacterCommand) {
			c.Options, c.Plan = startingLevel(2), []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.RollHitDie{}, SubclassID: "Bad Id"}}
		},
	}
	for name, mutate := range mutations {
		cmd := crCommand()
		mutate(&cmd)
		src := crSource(t)
		res, err := crUseCase(t, src, dicetest.NewFixedRoller()).Run(context.Background(), cmd)
		if res != nil || !errors.Is(err, srderr.ErrInvalidInput) || len(src.Queries()) != 0 {
			t.Errorf("%s: = %v, %v, %d lookups", name, res, err, len(src.Queries()))
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := telemetrytest.NewRecorder()
	res, err := crUseCase(t, crSource(t), dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry())).Run(ctx, crCommand())
	if res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 || errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("cancelled = %v, %v", res, err)
	}
}

// Requirement: doc/specs/character-creation.md section 6 (the defaults of CreateCharacterCommand) and FR-108 - a command that leaves every
// optional field at its zero value gives the same result as one that spells the defaults out, and the choices of the species-granted feat are
// honoured, not replaced by NoFeatChoices.
// Path: per case.
func TestCreateCharacterCommandDefaults(t *testing.T) {
	t.Parallel()
	plain := crCommand()
	explicit := crCommand()
	explicit.ClassChoices, explicit.Trinket, explicit.Options, explicit.HigherLevelEquipment = character.NoClassCreationChoices{}, character.NoTrinket{}, character.StandardRuleOptions(), character.DeclineHigherLevelEquipment{}
	explicit.Size, explicit.BackgroundFeatChoices, explicit.SpeciesFeatChoices = character.NoSizeChoice{}, character.NoFeatChoices{}, character.NoFeatChoices{}
	a, b := crCreated(t, crMust(t, plain)), crCreated(t, crMust(t, explicit))
	if !reflect.DeepEqual(a.Record(), b.Record()) || !a.Character().Equal(b.Character()) {
		t.Error("explicit defaults gave another result")
	}

	cmd := crCommand()
	cmd.BackgroundID, cmd.SpeciesID, cmd.BackgroundTool, cmd.BackgroundIncrease = "acolyte", "human", nil, character.PlusOneToAll{}
	cmd.Size, cmd.BackgroundFeatChoices, cmd.SpeciesFeatChoices = character.ChosenSize{Size: content.SizeMedium}, clericInitiate(), wizardInitiate()
	cmd.SpeciesGrants = must(character.NewSpeciesGrantSelections(0, "magic_initiate", nil))
	rec := crCreated(t, crMust(t, cmd)).Record()
	if spells := rec.GrantedSpells(); len(spells) != 2 || spells[0].IsZero() || spells[1].IsZero() {
		t.Errorf("granted spells = %+v", spells)
	}
}
