package character

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The printed phrases the SRD ties an equipment item to an earlier choice
// (SRD 5.2.1, "Soldier", p. 83; "Core Monk Traits", p. 49).
const (
	sameAsAbovePhrase     = "(same as above)"
	chosenToolAbovePhrase = "chosen for the tool proficiency above"
	artisansToolsCategory = "Artisan's Tools"
	versatileTrait        = "versatile"
)

// creationInput is a normalized command with the content it needs, all looked up.
// The creation rules (phase 1 and 3 validation, the mapping of the plan, the
// assembly of the character and its record) are its methods: pure, synchronous and
// silent (FR-004, FR-005, FR-074).
type creationInput struct {
	cmd         CreateCharacterCommand
	level       Level
	class       content.ClassDefinition
	background  content.Background
	species     content.Species
	bgFeat      content.Feat
	speciesFeat *content.Feat
	tools       map[content.ItemID]content.Tool
	classes     map[content.ClassID]content.ClassDefinition
	spells      map[content.SpellID]content.Spell
}

func problem(step CreationStep, kind CreateCharacterProblemKind) CreateCharacterProblem {
	return CreateCharacterProblem{step: step, kind: kind}
}

// commandProblems returns the phase 1 problems of cmd: those that need no content.
func commandProblems(cmd CreateCharacterCommand) []CreateCharacterProblem {
	var ps []CreateCharacterProblem
	if !slices.Contains(cmd.Options.PermittedMethods(), cmd.AbilityScores.Method()) {
		ps = append(ps, problem(CreationStepAbilityScores, AbilityScoreMethodNotPermitted))
	}
	if len(cmd.Languages) != 2 {
		p := problem(CreationStepLanguages, LanguageCountWrong)
		p.expected, p.actual = 2, len(cmd.Languages)
		ps = append(ps, p)
	}
	for i, l := range cmd.Languages {
		if !l.IsStandard() {
			p := problem(CreationStepLanguages, LanguageNotStandard)
			p.language = l
			ps = append(ps, p)
		}
		if l == content.LanguageCommon || slices.Contains(cmd.Languages[:i], l) {
			p := problem(CreationStepLanguages, LanguageRepeated)
			p.language = l
			ps = append(ps, p)
		}
	}
	if cmd.Alignment == content.AlignmentUnaligned {
		ps = append(ps, problem(CreationStepAlignment, AlignmentNotChoosable))
	}
	if level, err := NewLevel(cmd.Options.StartingLevel()); err == nil {
		if rej := (HigherLevelStart{}).CheckPlanLength(level, len(cmd.Plan)); rej != nil {
			p := problem(CreationStepStartingLevel, LevelAdvancementCountWrong)
			if m, ok := rej.(HigherLevelStartPlanLengthMismatch); ok {
				p.expected, p.actual = m.ExpectedSteps, m.ActualSteps
			}
			ps = append(ps, p)
		}
	}
	for i, s := range cmd.ClassSkills {
		if slices.Contains(cmd.ClassSkills[:i], s) {
			p := problem(CreationStepClass, ClassSkillRepeated)
			p.skill = s
			ps = append(ps, p)
		}
	}
	for _, a := range sortedAbilities(cmd.SpeciesGrants.abilities) {
		if v := cmd.SpeciesGrants.abilities[a]; !spellcastingAbility(v) {
			p := problem(CreationStepSpecies, SpeciesAbilityNotOffered)
			p.ability, p.name = v, a
			ps = append(ps, p)
		}
	}
	return sortProblems(ps)
}

// sortedAbilities returns the keys of m in order.
func sortedAbilities(m map[string]ability.Ability) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func spellcastingAbility(a ability.Ability) bool {
	return a == ability.Intelligence || a == ability.Wisdom || a == ability.Charisma
}

// keys returns the content keys the command needs besides those of the entries it
// finds later (the background feat and the tool families), in lookup order.
func (c CreateCharacterCommand) keys() []content.Key {
	keys := []content.Key{c.ClassID.Key(), c.BackgroundID.Key(), c.SpeciesID.Key()}
	for _, s := range c.Plan {
		keys = append(keys, s.ClassID.Key())
	}
	for _, t := range c.ClassTools {
		keys = append(keys, t.Tool.Key())
	}
	if t, ok := c.BackgroundTool.(ChosenTool); ok {
		keys = append(keys, t.Tool.Key())
	}
	if f, ok := c.SpeciesGrants.Feat(); ok {
		keys = append(keys, f.Key())
	}
	keys = append(keys, c.ClassChoices.ContentKeys()...)
	keys = append(keys, featChoiceKeys(c.BackgroundFeatChoices)...)
	return append(keys, featChoiceKeys(c.SpeciesFeatChoices)...)
}

// featChoiceKeys returns the spells and tools the choices of a feat name.
func featChoiceKeys(choices FeatChoices) []content.Key {
	var keys []content.Key
	switch v := choices.(type) {
	case MagicInitiateChoices:
		keys = append(keys, v.levelOne.Key())
		for _, s := range v.cantrips {
			keys = append(keys, s.Key())
		}
	case SkilledChoices:
		for _, p := range v.picks {
			if tool, ok := p.(SkilledTool); ok {
				keys = append(keys, tool.Tool.Key())
			}
		}
	}
	return keys
}

// sameName compares two printed names, ignoring case and the curly apostrophe.
func sameName(a, b string) bool {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "’", "'"))) }
	return norm(a) == norm(b)
}

// inCategory reports whether tool id is a tool of one of categories: an Artisan's
// Tools category by the tool's group, any other by the family a variant belongs to
// (FR-034, FR-111).
func (in creationInput) inCategory(id content.ItemID, categories []string) bool {
	switch v := in.tools[id].(type) {
	case content.StandaloneTool:
		return slices.ContainsFunc(categories, func(c string) bool {
			return sameName(c, artisansToolsCategory) && v.Group() == content.ToolGroupArtisans
		})
	case content.ToolVariant:
		family, ok := in.tools[v.VariantOf()].(content.ToolFamily)
		return ok && slices.ContainsFunc(categories, func(c string) bool {
			return sameName(c, family.Name()) || (sameName(c, artisansToolsCategory) && family.Group() == content.ToolGroupArtisans)
		})
	}
	return false
}

// classToolRequirement returns how many tools the class asks the player to choose
// and the categories they come from.
func (in creationInput) classToolRequirement() (int, []string) {
	picks := 0
	var categories []string
	for _, p := range in.class.ToolProficiencies() {
		if c, ok := p.(content.ClassToolProficiencyChoice); ok {
			picks += c.Picks()
			categories = append(categories, c.Categories()...)
		}
	}
	return picks, categories
}

// classProblems returns the phase 3 problems of the class step other than equipment.
func (in creationInput) classProblems() []CreateCharacterProblem {
	var ps []CreateCharacterProblem
	skills := in.cmd.ClassSkills
	switch c := in.class.SkillChoice().(type) {
	case content.SkillChoiceListed:
		if len(skills) != c.Picks() {
			ps = append(ps, countProblem(CreationStepClass, ClassSkillCountWrong, c.Picks(), len(skills)))
		}
		for _, s := range skills {
			if !slices.Contains(c.Skills(), s) {
				p := problem(CreationStepClass, ClassSkillNotOffered)
				p.skill = s
				ps = append(ps, p)
			}
		}
	case content.SkillChoiceAny:
		if len(skills) != c.Picks() {
			ps = append(ps, countProblem(CreationStepClass, ClassSkillCountWrong, c.Picks(), len(skills)))
		}
	}
	picks, categories := in.classToolRequirement()
	chosen := in.cmd.ClassTools
	if len(chosen) != picks || hasRepeatedTool(chosen) {
		ps = append(ps, countProblem(CreationStepClass, ClassToolCountWrong, picks, len(chosen)))
	}
	for _, t := range chosen {
		if !in.inCategory(t.Tool, categories) {
			p := problem(CreationStepClass, ClassToolNotInCategory)
			p.name = string(t.Tool)
			ps = append(ps, p)
		}
	}
	return ps
}

func hasRepeatedTool(tools []ChosenTool) bool {
	for i, t := range tools {
		if slices.Contains(tools[:i], t) {
			return true
		}
	}
	return false
}

func countProblem(step CreationStep, kind CreateCharacterProblemKind, expected, actual int) CreateCharacterProblem {
	p := problem(step, kind)
	p.expected, p.actual = expected, actual
	return p
}

// backgroundProblems returns the phase 3 problems of the background step other than
// equipment and the feat.
func (in creationInput) backgroundProblems() []CreateCharacterProblem {
	var ps []CreateCharacterProblem
	if r, ok := in.cmd.BackgroundIncrease.ApplyTo(in.cmd.AbilityScores.Scores(), in.background.AbilityScores()).(BackgroundIncreaseRefused); ok {
		for _, a := range r.NotListed() {
			p := problem(CreationStepBackground, BackgroundIncreaseAbilityNotListed)
			p.ability = a
			ps = append(ps, p)
		}
		for _, a := range r.ExceedsMaximum() {
			p := problem(CreationStepBackground, BackgroundIncreaseExceedsMaximum)
			p.ability = a
			ps = append(ps, p)
		}
	}
	chosen, has := in.cmd.BackgroundTool.(ChosenTool)
	switch t := in.background.ToolProficiency().(type) {
	case content.SpecificToolProficiency:
		if has {
			ps = append(ps, problem(CreationStepBackground, BackgroundToolNotAllowed))
		}
	case content.ChoiceToolProficiency:
		switch {
		case !has:
			ps = append(ps, problem(CreationStepBackground, BackgroundToolMissing))
		case !in.inCategory(chosen.Tool, []string{t.Category()}):
			p := problem(CreationStepBackground, BackgroundToolNotInCategory)
			p.name = string(chosen.Tool)
			ps = append(ps, p)
		}
	}
	return ps
}

// speciesProblems returns the phase 3 problems of the species step other than the feat.
func (in creationInput) speciesProblems() []CreateCharacterProblem {
	var ps []CreateCharacterProblem
	sizes := in.species.Sizes()
	switch s := in.cmd.Size.(type) {
	case NoSizeChoice:
		if len(sizes) > 1 {
			ps = append(ps, problem(CreationStepSpecies, SizeRequired))
		}
	case ChosenSize:
		if !slices.Contains(sizes, s.Size) {
			p := problem(CreationStepSpecies, SizeNotOffered)
			p.name = s.Size.String()
			ps = append(ps, p)
		}
	}
	traits := map[string]content.SpeciesChoice{}
	for _, c := range in.species.Choices() {
		traits[c.TraitName()] = c
		var picked []SpeciesTraitSelection
		for _, s := range in.cmd.SpeciesSelections {
			if s.Trait == c.TraitName() {
				picked = append(picked, s)
			}
		}
		named := func(kind CreateCharacterProblemKind) CreateCharacterProblem {
			p := problem(CreationStepSpecies, kind)
			p.name = c.TraitName()
			return p
		}
		switch {
		case len(picked) == 0:
			ps = append(ps, named(SpeciesChoiceMissing))
		case len(picked) > 1:
			ps = append(ps, named(SpeciesChoiceRepeated))
		case !slices.ContainsFunc(c.Options(), func(o content.ChoiceOption) bool { return o.Name() == picked[0].Option }):
			ps = append(ps, named(SpeciesChoiceOptionUnknown))
		}
	}
	for _, s := range in.cmd.SpeciesSelections {
		if _, ok := traits[s.Trait]; !ok {
			p := problem(CreationStepSpecies, SpeciesChoiceNotOffered)
			p.name = s.Trait
			ps = append(ps, p)
		}
	}
	for _, trait := range sortedAbilities(in.cmd.SpeciesGrants.abilities) {
		if _, ok := traits[trait]; !ok {
			p := problem(CreationStepSpecies, SpeciesGrantNotOffered)
			p.name = trait
			ps = append(ps, p)
		}
	}
	if in.speciesFeat != nil && in.speciesFeat.FeatCategory() != content.FeatOrigin {
		ps = append(ps, problem(CreationStepSpecies, SpeciesFeatNotOrigin))
	}
	return ps
}

// equipmentGrant resolves the starting equipment of the command: the class option
// then the background option, the placeholders resolved, the coins of both. It
// returns the problems of an unknown letter or an unresolved placeholder instead of
// a grant when there are any.
func (in creationInput) equipmentGrant() ([]content.EquipmentItem, int, []CreateCharacterProblem) {
	var ps []CreateCharacterProblem
	var classItems []content.ClassEquipmentItem
	classGold := 0
	start := in.class.StartingEquipment()
	switch in.cmd.ClassEquipment {
	case OptionA:
		classItems, classGold = start.A().Items(), start.A().Gold()
	case OptionB:
		classItems, classGold = start.B().Items(), start.B().Gold()
	case OptionC:
		if c, ok := start.C(); ok {
			classItems, classGold = c.Items(), c.Gold()
		} else {
			p := problem(CreationStepClass, ClassEquipmentOptionUnknown)
			p.letter = in.cmd.ClassEquipment
			ps = append(ps, p)
		}
	}
	var items []content.EquipmentItem
	classTool := ""
	if len(in.cmd.ClassTools) > 0 {
		classTool = in.tools[in.cmd.ClassTools[0].Tool].Name()
	}
	for _, i := range classItems {
		name := i.Name()
		if strings.Contains(name, chosenToolAbovePhrase) {
			if classTool == "" {
				ps = append(ps, problem(CreationStepClass, EquipmentPlaceholderUnresolved))
				continue
			}
			name = classTool
		}
		items = append(items, mustEquipmentItem(name, i.Quantity()))
	}
	var bgItems []content.EquipmentItem
	bgGold := 0
	options := in.background.EquipmentOptions()
	switch in.cmd.BackgroundEquipment {
	case OptionA:
		bgItems, bgGold = options.A().Items(), options.A().Gold()
	case OptionB:
		bgItems, bgGold = options.B().Items(), options.B().Gold()
	case OptionC:
		p := problem(CreationStepBackground, BackgroundEquipmentOptionUnknown)
		p.letter = in.cmd.BackgroundEquipment
		ps = append(ps, p)
	}
	bgTool := ""
	if t, ok := in.cmd.BackgroundTool.(ChosenTool); ok {
		bgTool = in.tools[t.Tool].Name()
	}
	for _, i := range bgItems {
		name := i.Name()
		if strings.HasSuffix(name, sameAsAbovePhrase) {
			if bgTool == "" {
				ps = append(ps, problem(CreationStepBackground, EquipmentPlaceholderUnresolved))
				continue
			}
			name = bgTool
		}
		items = append(items, mustEquipmentItem(name, i.Quantity()))
	}
	return items, classGold + bgGold, ps
}

// mustEquipmentItem builds an item from names and quantities the content already
// validated; the resolved placeholder is a tool name, which content validated too.
func mustEquipmentItem(name string, quantity int) content.EquipmentItem {
	return content.MustEquipmentItem(name, quantity)
}

// requiredList returns the spell list the background's printed qualifier names.
func (in creationInput) requiredList() RequiredSpellList {
	if q := in.background.Feat().Choice(); q != "" {
		return RequiredSpellListIs{Class: content.ClassID(strings.ToLower(q))}
	}
	return NoRequiredSpellList{}
}

// featOutcome is the result of taking the two Origin feats.
type featOutcome struct {
	character  Character
	feats      TakenFeats
	events     []Event
	profs      []ProficiencyGrants
	spells     []FeatGrantedSpells
	background FeatRejection
	species    FeatRejection
}

// takeFeats applies the background's Origin feat and then the species-granted one
// (FR-105, FR-108) with FeatTakingRules.
func (in creationInput) takeFeats(c Character) (featOutcome, error) {
	out := featOutcome{character: c}
	grant, err := NewFeatGrant(BackgroundFeat{Background: in.cmd.BackgroundID}, SpecifiedFeat{Feat: in.bgFeat.ID(), Required: in.requiredList()})
	if err != nil {
		return out, err
	}
	if err := out.take(in.bgFeat, grant, in.cmd.BackgroundFeatChoices, in.spells, &out.background); err != nil {
		return out, err
	}
	if in.speciesFeat != nil && in.speciesFeat.FeatCategory() == content.FeatOrigin {
		grant, err := NewFeatGrant(SpeciesTraitFeat{Species: in.cmd.SpeciesID, Trait: versatileTrait}, FeatFromCategory{Category: content.FeatOrigin})
		if err != nil {
			return out, err
		}
		return out, out.take(*in.speciesFeat, grant, in.cmd.SpeciesFeatChoices, in.spells, &out.species)
	}
	return out, nil
}

func (o *featOutcome) take(feat content.Feat, grant FeatGrant, choices FeatChoices, spells map[content.SpellID]content.Spell, rejected *FeatRejection) error {
	taking, err := FeatTakingRules{}.Take(o.character, o.feats, feat, grant, choices, HeldFeatures{}, spells)
	if err != nil {
		return err
	}
	switch t := taking.(type) {
	case FeatTakingChanged:
		o.character, o.feats = t.Character(), t.Feats()
		o.events = append(o.events, t.Events()...)
		o.profs = append(o.profs, t.ProficiencyGrants())
		o.spells = append(o.spells, t.GrantedSpells())
	case FeatTakingRejected:
		*rejected = t.Rejection
	}
	return nil
}

// builtCharacter is the character at the starting level, before the feats.
type builtCharacter struct {
	character Character
	events    []Event
	consumed  []ConsumedFeatGrant
}

// build creates the character with character.New and, above level 1, the pure
// HigherLevelStart service over the plan (FR-002, FR-074). gains holds the rolled
// Hit Point gain by plan step; a RollHitDie step without one takes the fixed value,
// which is how the dry run of phase 3 goes. A plan rule refuses it as the second
// result.
func (in creationInput) build(gains map[int]int) (builtCharacter, HigherLevelStartRejection, error) {
	scores := in.cmd.AbilityScores.Scores()
	if r, ok := in.cmd.BackgroundIncrease.ApplyTo(scores, in.background.AbilityScores()).(BackgroundIncreaseApplied); ok {
		scores = r.Scores()
	}
	created, err := New(NewParams{
		ID: in.cmd.CharacterID, Class: in.cmd.ClassID, HitPointDie: in.class.HitPointDie(), Species: in.cmd.SpeciesID,
		Background: in.cmd.BackgroundID, AbilityScores: scores, ExperiencePoints: HigherLevelStart{}.StartingExperience(in.level),
		FeatureStates: in.cmd.ClassChoices.SeedFeatureStates(FeatureStates{}),
	})
	if err != nil {
		return builtCharacter{}, nil, err
	}
	out := builtCharacter{character: created.Character(), events: created.Events()}
	plan, _, err := planSteps(in.cmd.Plan, in.classes, gains)
	if err != nil {
		return builtCharacter{}, nil, err
	}
	outcome, err := HigherLevelStart{}.Advance(out.character, in.level, in.classes, plan)
	if err != nil {
		return builtCharacter{}, nil, err
	}
	switch o := outcome.(type) {
	case HigherLevelStartRefused:
		return builtCharacter{}, o.Reason, nil
	case HigherLevelStartBuilt:
		return builtCharacter{character: o.Character(), events: slices.Concat(out.events, o.Events()), consumed: o.ConsumedGrants()}, nil, nil
	}
	return builtCharacter{}, nil, invalidInput("higher-level start answered neither built nor refused")
}

// planProblem turns a plan rejection into its problem. A rejection that no
// validated command reaches (a wrong plan length, invalid starting arguments, too
// little Experience Points, the maximum level) panics with an error value (FR-076).
func planProblem(r HigherLevelStartRejection) CreateCharacterProblem {
	switch v := r.(type) {
	case HigherLevelStartLevelUpRejected:
		switch v.Rejection.(type) {
		case LevelUpPrerequisitesNotMet:
		default:
			panic(fmt.Errorf("character: unreachable level-up rejection %T in a created character", v.Rejection))
		}
	case HigherLevelStartSubclassRejected, HigherLevelStartIncreaseRejected, HigherLevelStartIncreaseNotGranted, HigherLevelStartSubclassRequired:
	default:
		panic(fmt.Errorf("character: unreachable higher-level start rejection %T in a created character", r))
	}
	p := problem(CreationStepStartingLevel, LevelAdvancementRejected)
	p.plan = r
	return p
}

// contentProblems returns the phase 3 problems of the command with its content
// (FR-003): the choices that need content and the dry run, with fixed Hit Point
// gains, of the character and of both feats.
func (in creationInput) contentProblems() ([]CreateCharacterProblem, error) {
	ps := slices.Concat(in.classProblems(), in.backgroundProblems(), in.speciesProblems())
	_, _, equipment := in.equipmentGrant()
	ps = append(ps, equipment...)
	built, refusal, err := in.build(nil)
	if err != nil {
		return nil, err
	}
	if refusal != nil {
		ps = append(ps, planProblem(refusal))
	} else {
		feats, err := in.takeFeats(built.character)
		if err != nil {
			return nil, err
		}
		if feats.background != nil {
			p := problem(CreationStepBackground, BackgroundFeatRejected)
			p.feat = feats.background
			ps = append(ps, p)
		}
		if feats.species != nil {
			p := problem(CreationStepSpecies, SpeciesFeatRejected)
			p.feat = feats.species
			ps = append(ps, p)
		}
	}
	return sortProblems(ps), nil
}

// allowance returns the Starting Equipment at Higher Levels allowance the command
// asks for, and the rolled gold die expression when it has one.
func (in creationInput) allowance() (HigherLevelEquipment, dice.Expression) {
	if _, grant := in.cmd.HigherLevelEquipment.(GrantHigherLevelEquipment); !grant {
		return NoHigherLevelEquipment{}, dice.Expression{}
	}
	a := HigherLevelEquipmentRules{}.ForStartingLevel(in.level)
	if allowance, ok := a.(HigherLevelEquipmentAllowance); ok {
		if rolled := allowance.RolledGold(); len(rolled) > 0 {
			return a, rolled[0].Expression()
		}
	}
	return a, dice.Expression{}
}

// rolls are the dice of phase 4.
type creationRolls struct {
	allowance int // the roll of the allowance's rolled gold, 0 for none
	trinket   TrinketRoll
	gains     map[int]int
}

// assemble builds the final character and record from the rolled values (phase 5).
func (in creationInput) assemble(r creationRolls) (builtCharacter, CreationRecord, error) {
	built, refusal, err := in.build(r.gains)
	if err != nil {
		return builtCharacter{}, CreationRecord{}, err
	}
	if refusal != nil { // unreachable: the dry run of phase 3 accepted the plan and no rule depends on the gain
		return builtCharacter{}, CreationRecord{}, invalidInput("higher-level start refused after the dice were rolled")
	}
	feats, err := in.takeFeats(built.character)
	if err != nil {
		return builtCharacter{}, CreationRecord{}, err
	}
	built.character, built.events = feats.character, slices.Concat(built.events, feats.events)
	items, gold, _ := in.equipmentGrant()
	allowance, _ := in.allowance()
	if a, ok := allowance.(HigherLevelEquipmentAllowance); ok {
		var roll *int
		if r.allowance != 0 {
			roll = &r.allowance
		}
		total, err := a.TotalGold(roll)
		if err != nil {
			return builtCharacter{}, CreationRecord{}, err
		}
		gold += total
	}
	cost, err := content.NewCost(gold, content.CoinGP, 0)
	if err != nil {
		return builtCharacter{}, CreationRecord{}, err
	}
	skills, overlapping := in.skillUnion()
	rec := CreationRecord{
		method: in.cmd.AbilityScores.Method(), assigned: in.cmd.AbilityScores.Scores(), increase: in.cmd.BackgroundIncrease,
		skills: skills, overlapping: overlapping, backgroundTool: in.background.ToolProficiency(), backgroundChose: in.cmd.BackgroundTool,
		classTools: in.cmd.ClassTools, feat: in.bgFeat.ID(), required: in.requiredList(), size: in.size(),
		selections: in.cmd.SpeciesSelections, grants: in.cmd.SpeciesGrants, languages: in.languages(),
		alignment: in.cmd.Alignment, classChoices: in.cmd.ClassChoices,
		equipment: StartingEquipmentGrant{items: items, gold: cost}, allowance: allowance, allowanceRoll: r.allowance,
		trinket: r.trinket, feats: feats.feats, featProfs: feats.profs, featSpells: feats.spells, consumed: built.consumed, startingLevel: in.level,
	}
	return built, rec, nil
}

// size returns the size of the character: the one chosen, or the species' only one.
func (in creationInput) size() content.CreatureSize {
	if s, ok := in.cmd.Size.(ChosenSize); ok {
		return s.Size
	}
	return in.species.Sizes()[0]
}

// languages returns Common and the two chosen languages; the command was checked.
func (in creationInput) languages() Languages {
	l, err := NewLanguages(in.cmd.Languages...)
	if err != nil { // unreachable: the languages were validated
		return Languages{}
	}
	return l
}

// skillUnion returns the skills of the class, the background and the species grant,
// each once in that order, and those more than one source named (FR-036).
func (in creationInput) skillUnion() (all, overlapping []content.Skill) {
	sources := [][]content.Skill{in.cmd.ClassSkills, in.background.SkillProficiencies()}
	if s, ok := in.cmd.SpeciesGrants.Skill(); ok {
		sources = append(sources, []content.Skill{s})
	}
	count := map[content.Skill]int{}
	for _, src := range sources {
		for _, s := range src {
			if count[s] == 0 {
				all = append(all, s)
			}
			count[s]++
		}
	}
	for _, s := range all {
		if count[s] > 1 {
			overlapping = append(overlapping, s)
		}
	}
	return all, overlapping
}
