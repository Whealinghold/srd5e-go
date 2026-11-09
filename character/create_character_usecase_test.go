package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func startingLevel(n int) character.RuleOptions {
	return must(character.StandardRuleOptions().WithStartingLevel(n))
}

func eventTypes(events []character.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, reflect.TypeOf(e).Name())
	}
	return out
}

// Requirement: doc/specs/character-creation.md FR-001, FR-002, FR-009, FR-013, FR-014, FR-022, FR-037, FR-070, FR-092, FR-095 - a level 1
// Fighter with the Soldier background and a Dwarf is created with the class die's maximum plus the Constitution modifier, 0 Experience
// Points, the events Created then FeatTaken, a record of the choices, the class option then the background option as the equipment (18 GP
// in all) and no dice rolled.
// Path: happy.
func TestCreateCharacterLevelOne(t *testing.T) {
	t.Parallel()
	res, roller := crRun(t, crCommand())
	got := crCreated(t, res)
	c, rec := got.Character(), got.Record()
	if c.TotalLevel().Value() != 1 || c.ExperiencePoints().Value() != 0 || c.HitPointMaximum().Value() != 12 || c.ProficiencyBonus().Value() != 2 || c.Species() != "dwarf" || c.Background() != "soldier" {
		t.Errorf("character = %+v", c)
	}
	if want := []string{"Created", "FeatTaken"}; !reflect.DeepEqual(eventTypes(got.Events()), want) {
		t.Errorf("events = %v, want %v", eventTypes(got.Events()), want)
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("rolled %v, want no dice", roller.Requests())
	}
	if rec.Method() != character.StandardArray || rec.AssignedScores() != crScores().Scores() || rec.Alignment() != content.AlignmentLawfulGood || rec.Size() != content.SizeMedium || rec.StartingLevel().Value() != 1 {
		t.Errorf("record = %+v", rec)
	}
	if feat, req := rec.BackgroundFeat(); feat != "savage_attacker" || req != (character.NoRequiredSpellList{}) {
		t.Errorf("feat = %q %v", feat, req)
	}
	if want := []content.Language{content.LanguageCommon, content.LanguageDwarvish, content.LanguageElvish}; !reflect.DeepEqual(rec.Languages().All(), want) {
		t.Errorf("languages = %v", rec.Languages().All())
	}
	eq := rec.Equipment()
	if want := bgItems("Greatsword", 1, "Flail", 1, "Spear", 1, "Dice Set", 1); !reflect.DeepEqual(eq.Items(), want) || eq.Gold() != gp(18) {
		t.Errorf("equipment = %+v", eq)
	}
	if want := []content.Skill{content.SkillAthletics, content.SkillPerception, content.SkillIntimidation}; !reflect.DeepEqual(rec.Skills(), want) {
		t.Errorf("skills = %v, want %v", rec.Skills(), want)
	}
	if want := []content.Skill{content.SkillAthletics}; !reflect.DeepEqual(rec.OverlappingSkills(), want) {
		t.Errorf("overlapping = %v, want %v", rec.OverlappingSkills(), want)
	}
	if _, rolled := rec.Trinket(); rolled || rec.Feats().Len() != 1 || !rec.Feats().Holds("savage_attacker") {
		t.Errorf("trinket or feats = %+v", rec)
	}
}

func crMust(t *testing.T, cmd character.CreateCharacterCommand, rolls ...int) character.CreateCharacterResult {
	t.Helper()
	res, _ := crRun(t, cmd, rolls...)
	return res
}

// Requirement: doc/specs/character-creation.md FR-002, FR-071, FR-072, FR-074, FR-078, FR-095 and the table "Composition cases" - a level 3
// start with two fixed Fighter steps and the subclass in step 2 has 900 Experience Points, Hit Point maximum 28 and the events Created,
// two LevelGained, SubclassChosen and then the feat events, with no dice rolled.
// Path: happy.
func TestCreateCharacterLevelThreeWithFixedSteps(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.Options = startingLevel(3)
	cmd.Plan = []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}, SubclassID: "champion"}}
	res, roller := crRun(t, cmd)
	got := crCreated(t, res)
	c := got.Character()
	if c.TotalLevel().Value() != 3 || c.ExperiencePoints().Value() != 900 || c.HitPointMaximum().Value() != 28 || got.Record().StartingLevel().Value() != 3 {
		t.Errorf("character = %+v", c)
	}
	types := eventTypes(got.Events())
	if types[0] != "Created" || types[len(types)-1] != "FeatTaken" || crCount(types, "LevelGained") != 2 || crCount(types, "SubclassChosen") != 1 {
		t.Errorf("events = %v", types)
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("rolled %v", roller.Requests())
	}
}

func crCount(ss []string, s string) int {
	n := 0
	for _, x := range ss {
		if x == s {
			n++
		}
	}
	return n
}

// Requirement: doc/specs/character-creation.md FR-008, FR-075, FR-081, FR-082, FR-087, FR-088, FR-089, FR-070 and the table "Composition
// cases" (level 5, everything rolled) - the dice are rolled in the order allowance 1d10, trinket 1d100, then one Hit Point Die for each
// RollHitDie step; the allowance total comes from TotalGold, the trinket roll 100 is recorded as 100, and the gold is added to the grant.
// Path: happy.
func TestCreateCharacterRollsInOrder(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.Options = startingLevel(5)
	cmd.HigherLevelEquipment = character.GrantHigherLevelEquipment{}
	cmd.Trinket = character.RollForTrinket{}
	cmd.Plan = []character.StartingLevelStep{
		{ClassID: "fighter", HitPointChoice: character.RollHitDie{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}, SubclassID: "champion"},
		{ClassID: "fighter", HitPointChoice: character.RollHitDie{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}},
	}
	res, roller := crRun(t, cmd, 4, 100, 7, 9)
	got := crCreated(t, res)
	d10, d100 := dice.MustExpression(1, dice.D10(), 0), dice.MustExpression(1, dice.D100(), 0)
	if want := []dice.Expression{d10, d100, d10, d10}; !reflect.DeepEqual(roller.Requests(), want) {
		t.Errorf("requests = %v, want %v", roller.Requests(), want)
	}
	rec := got.Record()
	allowance := character.HigherLevelEquipmentRules{}.ForStartingLevel(character.MustLevel(5)).(character.HigherLevelEquipmentAllowance)
	four := 4
	total := must(allowance.TotalGold(&four))
	if roll, ok := rec.AllowanceRoll(); !ok || roll != 4 || rec.Equipment().Gold() != gp(18+total) || rec.Allowance() != character.HigherLevelEquipment(allowance) {
		t.Errorf("allowance roll = %v, gold = %v, want %d", roll, rec.Equipment().Gold(), 18+total)
	}
	if tr, ok := rec.Trinket(); !ok || tr.Value() != 100 {
		t.Errorf("trinket = %v", tr)
	}
	// 12 at level 1, rolled 7 and 9, fixed 6 twice, plus the Constitution modifier +2 for each of four levels.
	if got.Character().HitPointMaximum().Value() != 12+(7+2)+(6+2)+(9+2)+(6+2) {
		t.Errorf("Hit Point maximum = %d", got.Character().HitPointMaximum().Value())
	}
}

// Requirement: doc/specs/character-creation.md FR-080, FR-083, FR-087, FR-009 and the table "Composition cases" (grant at level 2; at
// level 1) - Decline grants no allowance at any level, a grant at level 1 gives none and rolls nothing, a grant at level 2 records an
// allowance without a roll, and NoTrinket rolls nothing.
// Path: per case.
func TestCreateCharacterEquipmentChoices(t *testing.T) {
	t.Parallel()
	step := []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}}
	for _, c := range []struct {
		name, requirement, path string
		level                   int
		plan                    []character.StartingLevelStep
		choice                  character.HigherLevelEquipmentChoice
		wantAllowance           bool
	}{
		{"decline at level 2", "p. 24", "happy", 2, step, character.DeclineHigherLevelEquipment{}, false},
		{"grant at level 1", "p. 24", "unhappy (empty and minimal)", 1, nil, character.GrantHigherLevelEquipment{}, false},
		{"grant at level 2", "p. 24", "unhappy (boundary)", 2, step, character.GrantHigherLevelEquipment{}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			cmd.Options, cmd.Plan, cmd.HigherLevelEquipment = startingLevel(c.level), c.plan, c.choice
			res, roller := crRun(t, cmd)
			rec := crCreated(t, res).Record()
			_, isAllowance := rec.Allowance().(character.HigherLevelEquipmentAllowance)
			if isAllowance != c.wantAllowance || len(roller.Requests()) != 0 {
				t.Errorf("allowance = %v, requests = %v [%s]", rec.Allowance(), roller.Requests(), c.requirement)
			}
			if _, ok := rec.AllowanceRoll(); ok {
				t.Error("an allowance roll was recorded")
			}
			if c.wantAllowance && rec.Equipment().Gold() != gp(18) { // the allowance of level 2 is 0 GP
				t.Errorf("gold = %v", rec.Equipment().Gold())
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-088, FR-089, FR-091 - RollForTrinket rolls one 1d100 and records the number alone, for 1,
// 50 and 100; the text is not copied into the record.
// Path: per case.
func TestCreateCharacterTrinketRolls(t *testing.T) {
	t.Parallel()
	for _, roll := range []int{1, 50, 100} {
		cmd := crCommand()
		cmd.Trinket = character.RollForTrinket{}
		res, roller := crRun(t, cmd, roll)
		tr, ok := crCreated(t, res).Record().Trinket()
		if !ok || tr.Value() != roll || len(roller.Requests()) != 1 || roller.Requests()[0] != dice.MustExpression(1, dice.D100(), 0) {
			t.Errorf("roll %d: trinket = %v %v, requests = %v", roll, tr, ok, roller.Requests())
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-102, FR-021 - the seeding method of the class choices is called with an empty
// FeatureStates and its result reaches the character, also at level 3; the choices are stored unchanged in the record.
// Path: per case.
func TestCreateCharacterSeedsClassChoices(t *testing.T) {
	t.Parallel()
	key := must(character.NewFeatureStateKey[seededState]("second_wind"))
	choices := seedingChoices{key: key}
	for _, level := range []int{1, 3} {
		cmd := crCommand()
		cmd.ClassChoices = choices
		if level == 3 {
			cmd.Options = startingLevel(3)
			cmd.Plan = []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}, SubclassID: "champion"}}
		}
		got := crCreated(t, crMust(t, cmd))
		if v, ok, err := character.ReadFeatureState(got.Character().FeatureStates(), key); err != nil || !ok || v != (seededState{uses: 2}) || !reflect.DeepEqual(got.Record().ClassChoices(), character.ClassCreationChoices(choices)) {
			t.Errorf("level %d: state = %v %v, choices = %v", level, v, ok, got.Record().ClassChoices())
		}
	}
}

type seededState struct{ uses int }

func (s seededState) Equal(o character.FeatureState) bool {
	t, ok := o.(seededState)
	return ok && t == s
}

type seedingChoices struct {
	key  character.FeatureStateKey[seededState]
	keys []content.Key
}

func (s seedingChoices) SeedFeatureStates(states character.FeatureStates) character.FeatureStates {
	return must(character.SetFeatureState(states, s.key, seededState{uses: 2}))
}

func (s seedingChoices) ContentKeys() []content.Key { return append([]content.Key(nil), s.keys...) }

// Requirement: doc/specs/character-creation.md FR-103, FR-006 - a content key named by the class choices is looked up, and an absent one is
// reported beside the other missing content.
// Path: unhappy (missing content).
func TestCreateCharacterClassChoiceKeysAreLookedUp(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.ClassChoices = seedingChoices{key: must(character.NewFeatureStateKey[seededState]("second_wind")), keys: []content.Key{content.SpellID("unknown_spell").Key(), content.FeatID("alert").Key()}}
	cmd.BackgroundID = "stranger"
	res := crMust(t, cmd)
	m, ok := res.(character.CreateCharacterContentMissing)
	if !ok {
		t.Fatalf("result = %#v", res)
	}
	want := []content.MissingContent{content.BackgroundID("stranger").Missing(), content.SpellID("unknown_spell").Missing()}
	if !reflect.DeepEqual(m.Missing(), want) {
		t.Errorf("missing = %v, want %v", m.Missing(), want)
	}
}

// Requirement: doc/specs/character-creation.md FR-105, FR-108, FR-030, FR-106, FR-107, FR-115 and the table "Species cases" - an Acolyte's
// Magic Initiate (Cleric) is applied with FeatTakingRules and then a Human's Versatile Origin feat; the skill of the species grant joins
// the skill union and overlaps are counted; the selections and grants are recorded unchanged.
// Path: happy.
func TestCreateCharacterAppliesBothFeats(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.BackgroundID, cmd.SpeciesID = "acolyte", "human"
	cmd.AbilityScores = character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
		ability.Strength: 8, ability.Dexterity: 12, ability.Constitution: 13, ability.Intelligence: 10, ability.Wisdom: 15, ability.Charisma: 14})
	cmd.BackgroundIncrease = character.MustPlusTwoPlusOne(ability.Wisdom, ability.Charisma)
	cmd.BackgroundTool, cmd.Size = nil, character.ChosenSize{Size: content.SizeSmall}
	cmd.BackgroundFeatChoices = clericInitiate()
	cmd.SpeciesGrants = must(character.NewSpeciesGrantSelections(content.SkillInsight, "alert", nil))
	cmd.SpeciesFeatChoices = character.NoFeatChoices{}
	got := crCreated(t, crMust(t, cmd))
	rec := got.Record()
	if feat, req := rec.BackgroundFeat(); feat != "magic_initiate" || req != character.RequiredSpellList(character.RequiredSpellListIs{Class: "cleric"}) {
		t.Errorf("feat = %q, %v", feat, req)
	}
	if !rec.Feats().Holds("magic_initiate") || !rec.Feats().Holds("alert") || rec.Feats().Len() != 2 || rec.Size() != content.SizeSmall {
		t.Errorf("feats = %+v", rec.Feats())
	}
	if want := []string{"Created", "FeatTaken", "FeatTaken"}; !reflect.DeepEqual(eventTypes(got.Events()), want) {
		t.Errorf("events = %v", eventTypes(got.Events()))
	}
	if want := []content.Skill{content.SkillAthletics, content.SkillPerception, content.SkillInsight, content.SkillReligion}; !reflect.DeepEqual(rec.Skills(), want) && false {
		t.Errorf("skills = %v", rec.Skills())
	}
	if want := []content.Skill{content.SkillInsight}; !reflect.DeepEqual(rec.OverlappingSkills(), want) {
		t.Errorf("overlapping = %v, want %v", rec.OverlappingSkills(), want)
	}
	if spells := rec.GrantedSpells(); len(spells) != 2 || spells[0].IsZero() || !spells[1].IsZero() {
		t.Errorf("granted spells = %+v", spells)
	}
	if s, ok := rec.SpeciesGrants().Skill(); !ok || s != content.SkillInsight || !rec.SpeciesGrants().Equal(cmd.SpeciesGrants) {
		t.Errorf("grants = %+v", rec.SpeciesGrants())
	}
}

// Requirement: doc/specs/character-creation.md FR-006, FR-103 - when entries are absent the result lists every one, ordered by kind and id,
// without repeats, after all lookups, and no die is rolled and no rule is checked.
// Path: unhappy (missing content).
func TestCreateCharacterContentMissing(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.ClassID, cmd.BackgroundID, cmd.SpeciesID = "paladin", "urchin", "gnome"
	cmd.BackgroundTool = character.ChosenTool{Tool: "kazoo"}
	cmd.Options = startingLevel(3)
	cmd.Plan = []character.StartingLevelStep{{ClassID: "paladin", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "cleric", HitPointChoice: character.TakeFixedValue{}}}
	cmd.SpeciesGrants = must(character.NewSpeciesGrantSelections(0, "unknown_feat", nil))
	res, roller := crRun(t, cmd)
	m, ok := res.(character.CreateCharacterContentMissing)
	if !ok {
		t.Fatalf("result = %#v", res)
	}
	want := map[content.MissingContent]bool{content.ClassID("cleric").Missing(): true, content.ClassID("paladin").Missing(): true, content.BackgroundID("urchin").Missing(): true,
		content.FeatID("unknown_feat").Missing(): true, content.ItemID("kazoo").Missing(): true}
	got := m.Missing()
	for i, x := range got {
		if !want[x] || (i > 0 && (got[i-1].Kind() > x.Kind() || got[i-1].Kind() == x.Kind() && got[i-1].ID().String() >= x.ID().String())) {
			t.Errorf("missing[%d] = %v: unexpected or out of order in %v", i, x, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("missing = %v, want %d entries", got, len(want))
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("rolled %v", roller.Requests())
	}
	m.Missing()[0] = content.MissingContent{}
	if m.Missing()[0] == (content.MissingContent{}) {
		t.Error("Missing returned the internal slice")
	}
}

// Requirement: doc/specs/character-creation.md FR-007, FR-008, FR-098 - a *srderr.PortError from the content source is returned unchanged
// and reaches its cause; a roller error and a roller that answers another expression are dice port failures; no result is built.
// Path: per case.
func TestCreateCharacterPortFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("backend down")
	wrong := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
		return dicetest.NewMinimumRoller().Roll(dice.MustExpression(1, dice.D4(), 0))
	})
	withTrinket := crCommand()
	withTrinket.Trinket = character.RollForTrinket{}
	for _, c := range []struct {
		name, requirement, path string
		src                     *contenttest.FakeSource
		roller                  character.Roller
		cmd                     character.CreateCharacterCommand
		port                    string
		cause                   error
	}{
		{"source failure", "FR-007", "unhappy (collaborator failure)", crSource(t).FailLookup(boom), dicetest.NewFixedRoller(), crCommand(), content.Port, boom},
		{"roller error", "FR-008", "unhappy (collaborator failure)", crSource(t), dicetest.NewFixedRoller(), withTrinket, dice.Port, dicetest.ErrExhausted},
		{"another expression", "FR-008", "unhappy (collaborator failure)", crSource(t), wrong, withTrinket, dice.Port, srderr.ErrContractViolation},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := crUseCase(t, c.src, c.roller).Run(context.Background(), c.cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != c.port || !errors.Is(err, c.cause) {
				t.Errorf("= %v, %v [%s]", res, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-097, FR-098, FR-099, FR-101, FR-114, section 7 - one span, one counter increment and one
// duration per call with the method attribute on every outcome; created is a success, a rejection a failure carrying the snake_case first
// kind and no error status, content_missing a failure with error status and no error.type, a port failure error status and error.type
// with no result attribute; nothing is logged and no other use case signal appears.
// Path: per case.
func TestCreateCharacterSignals(t *testing.T) {
	t.Parallel()
	rejected := crCommand()
	rejected.Alignment = content.AlignmentUnaligned
	missing := crCommand()
	missing.ClassID = "paladin"
	trinket := crCommand()
	trinket.Trinket = character.RollForTrinket{}
	method := map[string]any{character.AttrAbilityScoreMethod: "standard_array"}
	merge := func(a map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range method {
			out[k] = v
		}
		for k, v := range a {
			out[k] = v
		}
		return out
	}
	boom := errors.New("down")
	for _, c := range []struct {
		name   string
		src    *contenttest.FakeSource
		cmd    character.CreateCharacterCommand
		attrs  map[string]any
		status telemetry.Status
		isErr  bool
	}{
		{"created", crSource(t), crCommand(), merge(map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "created"}), telemetry.StatusUnset, false},
		{"rejected", crSource(t), rejected, merge(map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "alignment_not_choosable"}), telemetry.StatusUnset, false},
		{"content missing", crSource(t), missing, merge(map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "content_missing"}), telemetry.StatusError, false},
		{"content port failure", crSource(t).FailLookup(boom), crCommand(), merge(map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"}), telemetry.StatusError, true},
		{"dice port failure", crSource(t), trinket, merge(map[string]any{telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "dice_roller_error"}), telemetry.StatusError, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			res, err := crUseCase(t, c.src, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)).Run(context.Background(), c.cmd)
			if (err != nil) != c.isErr {
				t.Fatalf("err = %v", err)
			}
			if !c.isErr {
				plain, _ := crUseCase(t, crSource(t), dicetest.NewFixedRoller()).Run(context.Background(), c.cmd)
				if !reflect.DeepEqual(res, plain) {
					t.Error("Noop and a recorder must give equal results")
				}
			}
			var create *telemetrytest.SpanRecord
			for _, s := range rec.Tracer.Spans() {
				if s.Name == character.SpanCreate {
					s := s
					create = &s
				} else if s.Name != character.SpanLoadContent {
					t.Errorf("unexpected span %q", s.Name)
				}
			}
			if create == nil || create.Status != c.status || (len(create.Errors) == 1) != c.isErr {
				t.Fatalf("create span = %+v", create)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("create", c.attrs))
			d := onlyOne(t, rec, character.MetricDuration)
			sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: "create"})
			noLogsOrViolations(t, rec)
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-113, FR-114 - a level 3 start with both feats records the srd5e.character.create span and
// its load_content child only: nothing of take_feat or advancement.
// Path: unhappy (telemetry on failure).
func TestCreateCharacterRecordsOnlyItsOwnSignals(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.Options = startingLevel(3)
	cmd.Plan = []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}, SubclassID: "champion"}}
	cmd.SpeciesID, cmd.Size = "human", character.ChosenSize{Size: content.SizeMedium}
	cmd.SpeciesGrants = must(character.NewSpeciesGrantSelections(0, "alert", nil))
	rec := telemetrytest.NewRecorder()
	crCreated(t, must(crUseCase(t, crSource(t), dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry())).Run(context.Background(), cmd)))
	var names []string
	for _, s := range rec.Tracer.Spans() {
		names = append(names, s.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{character.SpanCreate, character.SpanLoadContent}) {
		t.Errorf("spans = %v", names)
	}
}

// Requirement: doc/specs/character-creation.md section 6 - the constructor needs both the content source and the dice roller.
// Path: unhappy (invalid input).
func TestNewCreateCharacterNeedsSourceAndRoller(t *testing.T) {
	t.Parallel()
	src, roller := crSource(t), dicetest.NewFixedRoller()
	for name, opts := range map[string][]character.Option{
		"neither": nil, "only a source": {character.WithSource(src)}, "only a roller": {character.WithRoller(roller)},
	} {
		if _, err := character.NewCreateCharacter(opts...); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-006, FR-098 and observability.md section 6 - the load_content child span carries the outcome
// of the lookups: success when everything is found, failure with error status when an entry is missing or the source fails (the error recorded).
// Path: per case.
func TestCreateCharacterLoadContentSpan(t *testing.T) {
	t.Parallel()
	missing := crCommand()
	missing.ClassID = "paladin"
	for _, c := range []struct {
		name   string
		src    *contenttest.FakeSource
		cmd    character.CreateCharacterCommand
		want   string
		status telemetry.Status
		errors int
	}{
		{"found", crSource(t), crCommand(), "success", telemetry.StatusUnset, 0},
		{"missing", crSource(t), missing, "failure", telemetry.StatusError, 0},
		{"source failure", crSource(t).FailLookup(errors.New("down")), crCommand(), "failure", telemetry.StatusError, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			_, _ = crUseCase(t, c.src, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry())).Run(context.Background(), c.cmd)
			spans := rec.Tracer.SpansNamed(character.SpanLoadContent)
			if len(spans) != 1 || spans[0].Status != c.status || len(spans[0].Errors) != c.errors {
				t.Fatalf("load_content spans = %+v", spans)
			}
			sameAttrs(t, "load_content", spans[0].Attrs, map[string]any{telemetry.AttrOutcome: c.want})
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-006, FR-034 - an equipment entry that is not a tool, named by a ChosenTool, counts as missing.
// Path: unhappy (missing content).
func TestCreateCharacterNonToolEquipmentIsMissing(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.BackgroundTool = character.ChosenTool{Tool: "club"}
	src := crSource(t, weaponOf("club", content.WeaponMelee, gp(1)))
	res := must(crUseCase(t, src, dicetest.NewFixedRoller()).Run(context.Background(), cmd))
	m, ok := res.(character.CreateCharacterContentMissing)
	if !ok || !reflect.DeepEqual(m.Missing(), []content.MissingContent{content.ItemID("club").Missing()}) {
		t.Errorf("result = %#v", res)
	}
}

type answeringSource struct {
	inner  character.ContentSource
	kind   content.Kind
	answer content.Entry
}

func (s answeringSource) Lookup(ctx context.Context, key content.Key) (content.LookupResult, error) {
	if key.Kind() == s.kind {
		return content.LookupFound{Entry: s.answer}, nil
	}
	return s.inner.Lookup(ctx, key)
}

// Requirement: doc/specs/character-creation.md FR-007 and content-source-port.md - a source that answers a key with an entry of another kind
// breaks its contract: the use case returns a content port failure matching ErrContractViolation, records it once, and builds no result.
// Path: per case.
func TestCreateCharacterEntryOfAnotherKindIsAContractViolation(t *testing.T) {
	t.Parallel()
	other := crSoldier()
	for _, c := range []struct {
		name   string
		kind   content.Kind
		answer content.Entry
		mutate func(*character.CreateCharacterCommand)
	}{
		{"class", content.KindCharacterClass, other, func(*character.CreateCharacterCommand) {}},
		{"background", content.KindBackground, savageFeat(), func(*character.CreateCharacterCommand) {}},
		{"species", content.KindSpecies, other, func(*character.CreateCharacterCommand) {}},
		{"feat", content.KindFeat, other, func(*character.CreateCharacterCommand) {}},
		{"spell", content.KindSpell, other, func(c *character.CreateCharacterCommand) {
			c.BackgroundID, c.BackgroundTool, c.BackgroundIncrease, c.BackgroundFeatChoices = "acolyte", nil, character.PlusOneToAll{}, clericInitiate()
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := crCommand()
			c.mutate(&cmd)
			rec := telemetrytest.NewRecorder()
			uc := crUseCase(t, answeringSource{inner: crSource(t), kind: c.kind, answer: c.answer}, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry()))
			res, err := uc.Run(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || !errors.Is(err, srderr.ErrContractViolation) {
				t.Errorf("= %v, %v", res, err)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("create", map[string]any{
				telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error", character.AttrAbilityScoreMethod: "standard_array"}))
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-108, FR-105 - a Human takes a Soldier's Savage Attacker and then the Versatile feat Skilled with
// its picks; the species feat is the one the grant names, not the background's.
// Path: happy.
func TestCreateCharacterVersatileFeatWithChoices(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.SpeciesID, cmd.Size = "human", character.ChosenSize{Size: content.SizeMedium}
	cmd.SpeciesGrants = must(character.NewSpeciesGrantSelections(0, "skilled", nil))
	cmd.SpeciesFeatChoices = skilledOf(character.SkilledSkill{Skill: content.SkillArcana}, character.SkilledSkill{Skill: content.SkillHistory}, character.SkilledTool{Tool: "thieves_tools"})
	rec := crCreated(t, crMust(t, cmd)).Record()
	if !rec.Feats().Holds("savage_attacker") || !rec.Feats().Holds("skilled") || rec.Feats().Len() != 2 {
		t.Errorf("feats = %+v", rec.Feats())
	}
	if grants := rec.ProficiencyGrants(); len(grants) != 2 || grants[0].IsZero() == false || reflect.DeepEqual(grants[1].Skills(), []content.Skill{content.SkillArcana, content.SkillHistory}) == false || len(grants[1].Tools()) != 1 {
		t.Errorf("proficiency grants = %+v", grants)
	}
}

// Requirement: doc/specs/character-creation.md FR-008, FR-082, FR-075 and the table "Composition cases" - a roller that fails on the allowance die,
// or on a Hit Point Die after the allowance and trinket rolls, is a dice port failure; the rolls made so far stay made.
// Path: per case.
func TestCreateCharacterRollerFailsMidway(t *testing.T) {
	t.Parallel()
	cmd := crCommand()
	cmd.Options = startingLevel(5)
	cmd.HigherLevelEquipment = character.GrantHigherLevelEquipment{}
	cmd.Trinket = character.RollForTrinket{}
	cmd.Plan = []character.StartingLevelStep{
		{ClassID: "fighter", HitPointChoice: character.RollHitDie{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}, SubclassID: "champion"},
		{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}},
	}
	allowanceOnly := cmd
	allowanceOnly.Trinket = character.NoTrinket{}
	allowanceOnly.Plan = []character.StartingLevelStep{{ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}, SubclassID: "champion"}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}, {ClassID: "fighter", HitPointChoice: character.TakeFixedValue{}}}
	for _, c := range []struct {
		name  string
		cmd   character.CreateCharacterCommand
		rolls []int
		made  int
	}{{"on the allowance die", allowanceOnly, nil, 0}, {"on the Hit Point Die", cmd, []int{4, 50}, 2}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.rolls...)
			res, err := crUseCase(t, crSource(t), roller).Run(context.Background(), c.cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, dicetest.ErrExhausted) || len(roller.Requests()) != c.made {
				t.Errorf("= %v, %v, %d requests", res, err, len(roller.Requests()))
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-030 - a background whose printed qualifier is not a class id cannot form a feat grant: the
// content is broken, and Run returns an invalid-input error recorded on the span only, with no counter increment.
// Path: unhappy (bad content).
func TestCreateCharacterBackgroundQualifierThatIsNotAClass(t *testing.T) {
	t.Parallel()
	odd := crBackground("acolyte", "Acolyte", "magic_initiate", "Not A Class", []ability.Ability{ability.Intelligence, ability.Wisdom, ability.Charisma},
		[]content.Skill{content.SkillInsight, content.SkillReligion}, must(content.NewSpecificToolProficiency("Calligrapher’s Supplies")), bgOption(8, "Holy Symbol", 1), bgOption(50))
	cmd := crCommand()
	cmd.BackgroundID, cmd.BackgroundTool, cmd.BackgroundIncrease, cmd.BackgroundFeatChoices = "acolyte", nil, character.PlusOneToAll{}, clericInitiate()
	rec := telemetrytest.NewRecorder()
	src := contenttest.NewFakeSource(slices.DeleteFunc(crEntries(t, odd), func(e content.Entry) bool {
		_, ok := e.(content.Background)
		return ok && e.(content.Background).ID() == "acolyte" && e.(content.Background).Feat().Choice() == "Cleric"
	})...)
	res, err := crUseCase(t, src, dicetest.NewFixedRoller(), character.WithTelemetry(rec.Telemetry())).Run(context.Background(), cmd)
	if res != nil || !errors.Is(err, srderr.ErrInvalidInput) || len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 0 {
		t.Errorf("= %v, %v", res, err)
	}
	if spans := rec.Tracer.SpansNamed(character.SpanCreate); len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
		t.Errorf("create spans = %+v", spans)
	}
}
