package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	rangerStyleFeature    content.FeatureID = "ranger_fighting_style"
	rangerASIFeature      content.FeatureID = "ranger_ability_score_improvement"
	rangerEpicBoonFeature content.FeatureID = "ranger_epic_boon"
)

// RangerReplacementMoment is the moment at which a Ranger replaces a choice already made (class-ranger.md FR-040, FR-053, FR-087).
// The zero value is not a moment, and is refused with RangerWrongMoment.
type RangerReplacementMoment int

const (
	// RangerFinishedLongRest: the character finished a Long Rest.
	RangerFinishedLongRest RangerReplacementMoment = iota + 1
	// RangerFinishedShortRest: the character finished a Short Rest.
	RangerFinishedShortRest
	// RangerGainedLevel: the character gained a Ranger level.
	RangerGainedLevel
)

// RangerChoice is a choice of the Ranger, a closed set. It has no variant for the subclass, which belongs to level advancement,
// or for class skills and Starting Equipment, which belong to character creation. Variants: RangerChooseWeaponMasteries,
// RangerReplaceWeaponMasteries, RangerChooseDeftExplorer, RangerChooseExpertise, RangerChooseFightingStyleFeat,
// RangerChooseDruidicWarrior, RangerReplaceDruidicCantrip, RangerChooseHuntersPrey, RangerReplaceHuntersPrey,
// RangerChooseDefensiveTactics, RangerReplaceDefensiveTactics, RangerChooseFeat.
type RangerChoice interface{ isRangerChoice() }

// RangerChooseWeaponMasteries is the initial Weapon Mastery choice: exactly two kinds the Ranger is proficient with, built by
// CreateWeaponMasteryChoice (class-ranger.md FR-037). Weapons holds the weapons the kinds name that exist. Construct it with a
// keyed literal.
type RangerChooseWeaponMasteries struct {
	Kinds         []content.ItemID
	Weapons       map[content.ItemID]content.Weapon
	Proficiencies WeaponProficiencies
}

// RangerReplaceWeaponMasteries replaces either or both kinds after a Long Rest (class-ranger.md FR-040, FR-041). Construct it
// with a keyed literal.
type RangerReplaceWeaponMasteries struct {
	Swaps         []MasterySwap
	Moment        RangerReplacementMoment
	Weapons       map[content.ItemID]content.Weapon
	Proficiencies WeaponProficiencies
}

// RangerChooseDeftExplorer is the Deft Explorer choice: the skill that gains Expertise and exactly two distinct languages
// (class-ranger.md FR-042 to FR-046). Construct it with a keyed literal.
type RangerChooseDeftExplorer struct {
	Skill     content.Skill
	Languages []content.Language
}

// RangerChooseExpertise is the level 9 Expertise choice: exactly two distinct skills (class-ranger.md FR-047, FR-048). Construct
// it with a keyed literal.
type RangerChooseExpertise struct{ Skills []content.Skill }

// RangerChooseFightingStyleFeat takes a Fighting Style feat as the Fighting Style (class-ranger.md FR-049). Feat is the feat's
// content. Construct it with a keyed literal.
type RangerChooseFightingStyleFeat struct{ Feat content.Feat }

// RangerChooseDruidicWarrior takes the Druidic Warrior option with two Druid cantrips (class-ranger.md FR-051). Construct it with
// a keyed literal.
type RangerChooseDruidicWarrior struct{ Cantrips []content.Spell }

// RangerReplaceDruidicCantrip replaces one Druidic Warrior cantrip with another Druid cantrip, whenever a Ranger level is gained
// (class-ranger.md FR-053). Construct it with a keyed literal.
type RangerReplaceDruidicCantrip struct {
	Old    content.SpellID
	New    content.Spell
	Moment RangerReplacementMoment
}

// RangerChooseHuntersPrey is the initial Hunter’s Prey choice by option id (class-ranger.md FR-086). Construct it with a keyed
// literal.
type RangerChooseHuntersPrey struct{ Option string }

// RangerReplaceHuntersPrey replaces the Hunter’s Prey option with the other one after a Short or Long Rest (class-ranger.md
// FR-087). Construct it with a keyed literal.
type RangerReplaceHuntersPrey struct {
	Option string
	Moment RangerReplacementMoment
}

// RangerChooseDefensiveTactics is the initial Defensive Tactics choice by option id (class-ranger.md FR-091). Construct it with a
// keyed literal.
type RangerChooseDefensiveTactics struct{ Option string }

// RangerReplaceDefensiveTactics replaces the Defensive Tactics option with the other one after a Short or Long Rest
// (class-ranger.md FR-087). Construct it with a keyed literal.
type RangerReplaceDefensiveTactics struct {
	Option string
	Moment RangerReplacementMoment
}

// RangerChooseFeat uses one feat grant of the class (Ability Score Improvement at 4, 8, 12 and 16, Epic Boon at 19) on a feat:
// the grant is counted and the feat is validated against its category and qualification, then handed to the host's TakeFeat with
// RangerChoiceAccepted.Grant (class-ranger.md FR-062, FR-064). Construct it with a keyed literal.
type RangerChooseFeat struct{ Feat content.Feat }

func (RangerChooseWeaponMasteries) isRangerChoice()   {}
func (RangerReplaceWeaponMasteries) isRangerChoice()  {}
func (RangerChooseDeftExplorer) isRangerChoice()      {}
func (RangerChooseExpertise) isRangerChoice()         {}
func (RangerChooseFightingStyleFeat) isRangerChoice() {}
func (RangerChooseDruidicWarrior) isRangerChoice()    {}
func (RangerReplaceDruidicCantrip) isRangerChoice()   {}
func (RangerChooseHuntersPrey) isRangerChoice()       {}
func (RangerReplaceHuntersPrey) isRangerChoice()      {}
func (RangerChooseDefensiveTactics) isRangerChoice()  {}
func (RangerReplaceDefensiveTactics) isRangerChoice() {}
func (RangerChooseFeat) isRangerChoice()              {}

// RangerChoiceFacts is everything RangerChoiceRules needs, resolved by the caller. Skills are the character's skill
// proficiencies and Expertise, Masteries its Weapon Mastery choice and Feats the feats it holds. Construct it with a keyed
// literal.
type RangerChoiceFacts struct {
	Character Character
	Feats     TakenFeats
	Ranger    RangerFacts
	Skills    SkillProficiencies
	Masteries WeaponMasteryChoice
}

// RangerChoiceVerdict is the closed outcome of RangerChoiceRules.Decide. Variants: RangerChoiceAccepted, RangerChoiceRefused.
type RangerChoiceVerdict interface{ isRangerChoiceVerdict() }

// RangerChoiceAccepted is a choice the rules allow: the new Character (holding the new RangerState and, for a mastery choice,
// the Weapon Mastery choice), the feats, the state and the events. Only Decide builds one.
type RangerChoiceAccepted struct {
	character Character
	feats     TakenFeats
	state     RangerState
	events    []Event
	grant     FeatGrant
}

// Character returns the new Character.
func (a RangerChoiceAccepted) Character() Character { return a.character }

// Feats returns the new TakenFeats; unchanged unless a Fighting Style feat was taken.
func (a RangerChoiceAccepted) Feats() TakenFeats { return a.feats }

// State returns the new RangerState.
func (a RangerChoiceAccepted) State() RangerState { return a.state }

// Grant returns the FeatGrant to pass to TakeFeat with the feat of a RangerChooseFeat, and the zero grant for any other choice.
func (a RangerChoiceAccepted) Grant() FeatGrant { return a.grant }

// Events returns a new slice of the events: those of taking the feat, then the RangerChoiceMade or RangerChoiceReplaced event.
func (a RangerChoiceAccepted) Events() []Event { return slices.Clone(a.events) }

// RangerChoiceRefused is a choice a rule refused: nothing changed and there are no events. Only Decide builds one.
type RangerChoiceRefused struct{ reasons []RangerReason }

// Reasons returns every reason that holds in declaration order of RangerReason, as a new slice. It is never empty.
func (r RangerChoiceRefused) Reasons() []RangerReason { return slices.Clone(r.reasons) }

func (RangerChoiceAccepted) isRangerChoiceVerdict() {}
func (RangerChoiceRefused) isRangerChoiceVerdict()  {}

// RangerChoiceRules decides the choices of a Ranger. The mastery, feat and qualification rules own their checks; it only maps
// their outcomes to RangerReason (class-ranger.md FR-037, FR-049, FR-108). It is stateless; the zero value is ready to use. Safe
// for concurrent use.
type RangerChoiceRules struct{}

// rangerChoiceFeature returns the srd5e.feature value of a choice, and "" for an unknown one.
func rangerChoiceFeature(choice RangerChoice) string {
	switch choice.(type) {
	case RangerChooseWeaponMasteries, RangerReplaceWeaponMasteries:
		return ValueFeatureRangerWeaponMastery
	case RangerChooseDeftExplorer:
		return ValueFeatureDeftExplorer
	case RangerChooseExpertise:
		return ValueFeatureRangerExpertise
	case RangerChooseFightingStyleFeat, RangerChooseDruidicWarrior, RangerReplaceDruidicCantrip:
		return ValueFeatureRangerFightingStyle
	case RangerChooseHuntersPrey, RangerReplaceHuntersPrey:
		return ValueFeatureHuntersPrey
	case RangerChooseDefensiveTactics, RangerReplaceDefensiveTactics:
		return ValueFeatureDefensiveTactics
	case RangerChooseFeat:
		return ValueFeatureRangerFeatGrant
	}
	return ""
}

// Decide checks choice against facts and, when no rule refuses it, applies it. Every reason that holds is reported together, in
// the declaration order of RangerReason. A replacement made when no choice exists is RangerNothingToReplace, and one made at the
// wrong moment RangerWrongMoment, before the new value is looked at.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, progression or level, a nil or unknown choice, and a
// zero Feat in a feat choice. The inputs are unchanged. It never panics.
func (RangerChoiceRules) Decide(in RangerChoiceFacts, choice RangerChoice) (RangerChoiceVerdict, error) {
	f := in.Ranger
	if in.Character.IsZero() || f.Progression.IsZero() || f.Level.IsZero() || rangerChoiceFeature(choice) == "" {
		return nil, invalidInput("Ranger choice with a zero character, progression or level, or no known choice")
	}
	d := &rangerDecision{in: in, state: f.State, character: in.Character, feats: in.Feats}
	var err error
	switch c := choice.(type) {
	case RangerChooseWeaponMasteries:
		d.chooseMasteries(c)
	case RangerReplaceWeaponMasteries:
		err = d.replaceMasteries(c)
	case RangerChooseDeftExplorer:
		d.deftExplorer(c)
	case RangerChooseExpertise:
		d.expertise(c)
	case RangerChooseFightingStyleFeat:
		err = d.styleFeat(c)
	case RangerChooseDruidicWarrior:
		d.druidicWarrior(c)
	case RangerReplaceDruidicCantrip:
		d.replaceCantrip(c)
	case RangerChooseHuntersPrey:
		d.option(RangerFeatureHuntersPrey, c.Option, f.Progression.HuntersPreyOptions(), &d.state.prey, false, 0)
	case RangerReplaceHuntersPrey:
		d.option(RangerFeatureHuntersPrey, c.Option, f.Progression.HuntersPreyOptions(), &d.state.prey, true, c.Moment)
	case RangerChooseDefensiveTactics:
		d.option(RangerFeatureDefensiveTactics, c.Option, f.Progression.DefensiveTacticsOptions(), &d.state.tactics, false, 0)
	case RangerReplaceDefensiveTactics:
		d.option(RangerFeatureDefensiveTactics, c.Option, f.Progression.DefensiveTacticsOptions(), &d.state.tactics, true, c.Moment)
	case RangerChooseFeat:
		err = d.featGrant(c)
	}
	if err != nil {
		return nil, err
	}
	if len(d.reasons) > 0 {
		slices.Sort(d.reasons)
		return RangerChoiceRefused{reasons: slices.Compact(d.reasons)}, nil
	}
	next, err := WithFeatureState(d.character, RangerStateKey(), d.state)
	if err != nil { // unreachable: the key is valid
		return nil, err
	}
	if d.masteries != nil {
		if next, err = WithFeatureState(next, WeaponMasteryChoiceKey(), NewWeaponMasteryChoiceState(*d.masteries)); err != nil { // unreachable: as above
			return nil, err
		}
	}
	if d.replaced {
		d.events = append(d.events, RangerChoiceReplaced{id: next.ID(), choice: rangerChoiceFeature(choice)})
	} else {
		d.events = append(d.events, RangerChoiceMade{id: next.ID(), choice: rangerChoiceFeature(choice)})
	}
	return RangerChoiceAccepted{character: next, feats: d.feats, state: d.state, events: d.events, grant: d.grant}, nil
}

// rangerDecision is one choice being decided: the facts, the state and character it builds and the reasons found.
type rangerDecision struct {
	in        RangerChoiceFacts
	state     RangerState
	character Character
	feats     TakenFeats
	masteries *WeaponMasteryChoice
	events    []Event
	grant     FeatGrant
	reasons   []RangerReason
	replaced  bool
}

func (d *rangerDecision) add(r RangerReason) { d.reasons = append(d.reasons, r) }

func (d *rangerDecision) gate(feat RangerFeature) {
	if !d.in.Ranger.has(feat) {
		d.add(RangerFeatureNotAvailable)
	}
}

func isRest(m RangerReplacementMoment) bool {
	return m == RangerFinishedLongRest || m == RangerFinishedShortRest
}

func (d *rangerDecision) chooseMasteries(c RangerChooseWeaponMasteries) {
	d.gate(RangerFeatureWeaponMastery)
	if len(d.in.Masteries.kinds) > 0 {
		d.add(RangerAlreadyChosen)
	}
	if len(d.reasons) > 0 {
		return
	}
	res, err := CreateWeaponMasteryChoice(c.Kinds, c.Weapons, 2, ProficientWeaponEligibility{Proficiencies: c.Proficiencies})
	if err != nil { // unreachable: the count is 2
		d.add(RangerInvalidChoice)
		return
	}
	switch v := res.(type) {
	case WeaponMasteryChoiceAccepted:
		d.masteries = &v.choice
	default:
		d.add(RangerInvalidChoice)
	}
}

func (d *rangerDecision) replaceMasteries(c RangerReplaceWeaponMasteries) error {
	d.replaced = true
	d.gate(RangerFeatureWeaponMastery)
	if len(d.in.Masteries.kinds) == 0 {
		d.add(RangerNothingToReplace)
	}
	if c.Moment != RangerFinishedLongRest {
		d.add(RangerWrongMoment)
	}
	if len(d.reasons) > 0 {
		return nil
	}
	res, err := ChangeWeaponMasteryChoice(WeaponMasteryChange{Current: d.in.Masteries, Swaps: c.Swaps, Moment: MasteryChangeLongRest, Rule: ReplaceAnyKinds,
		Resolved: c.Weapons, Eligibility: ProficientWeaponEligibility{Proficiencies: c.Proficiencies}})
	if err != nil {
		return err
	}
	if v, ok := res.(WeaponMasteryChoiceAccepted); ok {
		d.masteries = &v.choice
		return nil
	}
	d.add(RangerInvalidChoice)
	return nil
}

// hasExpertise reports whether the skill already has Expertise, counting the Deft Explorer skill.
func (d *rangerDecision) hasExpertise(s content.Skill) bool {
	deft, chosen := d.state.DeftExplorer()
	return d.in.Skills.HasExpertise(s) || (chosen && deft.Skill == s)
}

func (d *rangerDecision) deftExplorer(c RangerChooseDeftExplorer) {
	d.gate(RangerFeatureDeftExplorer)
	if d.state.deftChosen {
		d.add(RangerAlreadyChosen)
	}
	switch {
	case !c.Skill.Valid():
		d.add(RangerInvalidChoice)
	case !d.in.Skills.Proficient(c.Skill):
		d.add(RangerNotProficient)
	case d.hasExpertise(c.Skill):
		d.add(RangerAlreadyHasExpertise)
	}
	if len(c.Languages) != 2 || !c.Languages[0].Valid() || !c.Languages[1].Valid() || c.Languages[0] == c.Languages[1] {
		d.add(RangerInvalidChoice)
	}
	if len(d.reasons) == 0 {
		d.state.deft, d.state.deftChosen = DeftExplorerChoice{Skill: c.Skill, Languages: [2]content.Language{c.Languages[0], c.Languages[1]}}, true
	}
}

func (d *rangerDecision) expertise(c RangerChooseExpertise) {
	d.gate(RangerFeatureExpertise)
	if d.state.expertiseChosen {
		d.add(RangerAlreadyChosen)
	}
	if len(c.Skills) != 2 || c.Skills[0] == c.Skills[1] || !c.Skills[0].Valid() || !c.Skills[1].Valid() {
		d.add(RangerInvalidChoice)
	} else {
		for _, s := range c.Skills {
			switch {
			case !d.in.Skills.Proficient(s):
				d.add(RangerNotProficient)
			case d.hasExpertise(s):
				d.add(RangerAlreadyHasExpertise)
			}
		}
	}
	if len(d.reasons) == 0 {
		d.state.expertise, d.state.expertiseChosen = [2]content.Skill{c.Skills[0], c.Skills[1]}, true
	}
}

func (d *rangerDecision) styleGate() {
	d.gate(RangerFeatureFightingStyle)
	if d.state.style != nil {
		d.add(RangerAlreadyChosen)
	}
}

func (d *rangerDecision) styleFeat(c RangerChooseFightingStyleFeat) error {
	if !c.Feat.ID().Valid() {
		return invalidInput("Fighting Style choice with a zero feat")
	}
	d.styleGate()
	if len(d.reasons) > 0 {
		return nil
	}
	grant, err := NewFeatGrant(ClassFeatureFeat{Class: RangerClassID, Feature: rangerStyleFeature}, FeatFromCategory{Category: content.FeatFightingStyle})
	if err != nil {
		return err
	}
	held, err := NewHeldFeatures(rangerStyleFeature)
	if err != nil {
		return err
	}
	taking, err := FeatTakingRules{}.Take(d.in.Character, d.in.Feats, c.Feat, grant, NoFeatChoices{}, held, nil)
	if err != nil {
		return err
	}
	switch t := taking.(type) {
	case FeatTakingChanged:
		d.character, d.feats, d.events = t.Character(), t.Feats(), t.Events()
		d.state.style = RangerFightingStyleFeat{Feat: c.Feat.ID()}
	default:
		d.add(RangerInvalidChoice)
	}
	return nil
}

// druidCantrip reports whether s is a cantrip on the Druid list.
func druidCantrip(s content.Spell) bool {
	return s.Level() == 0 && slices.Contains(s.Classes(), "druid")
}

func (d *rangerDecision) druidicWarrior(c RangerChooseDruidicWarrior) {
	d.styleGate()
	if len(c.Cantrips) != 2 || c.Cantrips[0].ID() == c.Cantrips[1].ID() || !druidCantrip(c.Cantrips[0]) || !druidCantrip(c.Cantrips[1]) {
		d.add(RangerInvalidChoice)
	}
	if len(d.reasons) == 0 {
		d.state.style = DruidicWarrior{Cantrips: [2]content.SpellID{c.Cantrips[0].ID(), c.Cantrips[1].ID()}}
	}
}

func (d *rangerDecision) replaceCantrip(c RangerReplaceDruidicCantrip) {
	d.replaced = true
	dw, ok := d.state.style.(DruidicWarrior)
	if !ok {
		d.add(RangerNothingToReplace)
		return
	}
	if c.Moment != RangerGainedLevel {
		d.add(RangerWrongMoment)
		return
	}
	i := slices.Index(dw.Cantrips[:], c.Old)
	if i < 0 || !druidCantrip(c.New) || slices.Contains(dw.Cantrips[:], c.New.ID()) {
		d.add(RangerInvalidChoice)
		return
	}
	dw.Cantrips[i] = c.New.ID()
	d.state.style = dw
}

// option decides a Hunter’s Prey or Defensive Tactics choice: the initial choice, or a replacement at moment. held is the field of the
// state that holds the option.
func (d *rangerDecision) option(feat RangerFeature, option string, options []string, held *string, replace bool, moment RangerReplacementMoment) {
	switch {
	case !d.in.Ranger.hunter():
		d.add(RangerNotHunter)
	case !d.in.Ranger.has(feat):
		d.add(RangerFeatureNotAvailable)
	}
	if !slices.Contains(options, option) {
		d.add(RangerUnknownOption)
	}
	switch {
	case !replace && *held != "":
		d.add(RangerAlreadyChosen)
	case replace:
		d.replaced = true
		switch {
		case *held == "":
			d.add(RangerNothingToReplace)
		case !isRest(moment):
			d.add(RangerWrongMoment)
		case option == *held:
			d.add(RangerSameOption)
		}
	}
	if len(d.reasons) == 0 {
		*held = option
	}
}

// featGrant counts one feat grant of the category of the feat and validates the feat for it.
func (d *rangerDecision) featGrant(c RangerChooseFeat) error {
	if !c.Feat.ID().Valid() {
		return invalidInput("feat choice with a zero feat")
	}
	general, epic := d.in.Ranger.Progression.FeatGrantsUpTo(d.in.Ranger.Level)
	takenGeneral, takenEpic := d.state.FeatsTaken()
	var feature content.FeatureID
	switch c.Feat.FeatCategory() {
	case content.FeatGeneral:
		feature = rangerASIFeature
		if general-takenGeneral <= 0 {
			d.add(RangerNoGrantRemaining)
		}
	case content.FeatEpicBoon:
		feature = rangerEpicBoonFeature
		switch {
		case epic == 0:
			d.add(RangerFeatureNotAvailable)
		case epic-takenEpic <= 0:
			d.add(RangerNoGrantRemaining)
		}
	case content.FeatOrigin, content.FeatFightingStyle:
		d.add(RangerInvalidChoice)
	}
	if len(d.reasons) > 0 {
		return nil
	}
	held, err := NewHeldFeatures()
	if err != nil {
		return err
	}
	if len((FeatQualification{}).Check(c.Feat, FeatQualificationFacts{Level: d.in.Character.TotalLevel(), Scores: d.in.Character.AbilityScores(), Features: held})) > 0 {
		d.add(RangerInvalidChoice)
		return nil
	}
	if d.grant, err = NewFeatGrant(ClassFeatureFeat{Class: RangerClassID, Feature: feature}, FeatFromCategory{Category: c.Feat.FeatCategory()}); err != nil {
		return err
	}
	if c.Feat.FeatCategory() == content.FeatEpicBoon {
		d.state.epicBoons++
	} else {
		d.state.generalFeats++
	}
	return nil
}
