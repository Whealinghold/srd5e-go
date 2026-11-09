package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// FeatTaking is the closed outcome of FeatTakingRules.Take. Variants:
// FeatTakingChanged, FeatTakingRejected.
type FeatTaking interface{ isFeatTaking() }

// FeatTakingChanged is the outcome of a feat taken: the new Character (with the
// ability score increase applied), the new TakenFeats, the proficiencies and
// spells the feat grants and the events in order: FeatTaken (after FeatReplaced
// when a feat was replaced), then those of AbilityScoreIncreaser. Only the rules
// build one.
type FeatTakingChanged struct {
	character Character
	feats     TakenFeats
	grants    ProficiencyGrants
	spells    FeatGrantedSpells
	events    []Event
}

// Character returns the new Character.
func (t FeatTakingChanged) Character() Character { return t.character }

// Feats returns the new TakenFeats.
func (t FeatTakingChanged) Feats() TakenFeats { return t.feats }

// ProficiencyGrants returns the proficiencies the feat gives, for the host to merge.
func (t FeatTakingChanged) ProficiencyGrants() ProficiencyGrants { return t.grants }

// GrantedSpells returns the spells the feat gives; the zero value when it gives none.
func (t FeatTakingChanged) GrantedSpells() FeatGrantedSpells { return t.spells }

// Events returns a new slice of the events, in order.
func (t FeatTakingChanged) Events() []Event { return slices.Clone(t.events) }

// FeatTakingRejected is the outcome of a feat a rule refused; nothing changed and
// no events were produced. Construct it with a keyed literal.
type FeatTakingRejected struct{ Rejection FeatRejection }

func (FeatTakingChanged) isFeatTaking()  {}
func (FeatTakingRejected) isFeatTaking() {}

// FeatTakingRules checks and applies the taking of a feat (SRD 5.2.1, "Parts of a
// Feat", p. 87). The zero value is ready to use. It is stateless and safe for
// concurrent use.
type FeatTakingRules struct{}

// Take checks a feat and, when every check passes, applies its benefits. The
// checks run in this order and the first failure is reported: FeatBehaviourUnknown
// (only the 17 SRD feats have behaviour), FeatOutsideGrant, FeatNotRepeatable,
// FeatPrerequisiteNotMet (against the character before any benefit),
// FeatChoicesInvalid (every problem found), FeatIncreaseRejected. The benefits are
// the new TakenFeats, the ability score increase applied with
// AbilityScoreIncreaser, the proficiencies of Skilled and the spells of Magic
// Initiate. held feeds the feature prerequisites; resolved holds the Spell of
// every spell named in a Magic Initiate choice. Tool picks are ItemIds the caller
// checks exist.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, Feat
// or FeatGrant, nil choices, or a Magic Initiate spell missing from resolved. The
// inputs are unchanged. It never panics.
func (FeatTakingRules) Take(c Character, feats TakenFeats, feat content.Feat, grant FeatGrant, choices FeatChoices, held HeldFeatures, resolved map[content.SpellID]content.Spell) (FeatTaking, error) {
	if c.IsZero() || !feat.ID().Valid() || grant.IsZero() || choices == nil {
		return nil, invalidInput("take a feat with a zero character, feat or grant, or nil choices")
	}
	id := feat.ID()
	shape, ok := shapeOf(id)
	reject := func(r FeatRejection) (FeatTaking, error) { return FeatTakingRejected{Rejection: r}, nil }
	switch {
	case !ok:
		return reject(FeatBehaviourUnknown{Feat: id})
	case !grant.allows(id, feat.FeatCategory()):
		return reject(FeatOutsideGrant{Feat: id})
	case feats.Holds(id) && !feat.Repeatable():
		return reject(FeatNotRepeatable{Feat: id})
	}
	if unmet := (FeatQualification{}).Check(feat, FeatQualificationFacts{Level: c.TotalLevel(), Scores: c.AbilityScores(), Features: held}); len(unmet) > 0 {
		return reject(FeatPrerequisiteNotMet{unmet: unmet})
	}
	problems, err := checkFeatChoices(shape, choices, grant, feats, resolved)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return reject(FeatChoicesInvalid{problems: problems})
	}
	taken, err := NewTakenFeat(id, feat.FeatCategory(), grant.source, choices)
	if err != nil { // unreachable: the shape matched and the grant is valid
		return nil, err
	}
	next, events := c, []Event{FeatTaken{id: c.ID(), feat: id, category: feat.FeatCategory(), source: grant.source}}
	if inc, has, err := featIncrease(shape, choices); err != nil {
		return nil, err
	} else if has {
		tr, err := AbilityScoreIncreaser{}.Apply(c, inc)
		if err != nil { // unreachable: the character and the increase are valid
			return nil, err
		}
		switch t := tr.(type) {
		case TransitionRejected[AbilityScoreIncreaseRejection]:
			return reject(FeatIncreaseRejected{Rejection: t.Rejection()})
		case TransitionChanged[AbilityScoreIncreaseRejection]:
			next, events = t.Character(), append(events, t.Events()...)
		}
	}
	out := FeatTakingChanged{character: next, feats: feats.with(taken), events: events}
	switch v := choices.(type) {
	case SkilledChoices:
		out.grants = ProficiencyGrants{}.with(v)
	case MagicInitiateChoices:
		out.spells = grantedSpellsOf(v)
	}
	return out, nil
}

// featIncrease returns the ability score increase the choices make, and false
// when the feat gives none.
func featIncrease(shape featShape, choices FeatChoices) (AbilityScoreIncrease, bool, error) {
	var inc AbilityScoreIncrease
	var err error
	switch v := choices.(type) {
	case AbilityScoreImprovementChoices:
		inc, err = v.Increase()
	case SingleAbilityIncrease: // Grappler: maximum 20; an Epic Boon: maximum 30 (what NewEpicBoonIncrease builds)
		inc, err = NewAbilityScoreIncrease(map[ability.Ability]int{v.ability: 1}, shape.maximum)
	default:
		return AbilityScoreIncrease{}, false, nil
	}
	return inc, err == nil, err
}

// checkFeatChoices returns every problem with choices for a feat of shape (tables
// C and M), in a fixed order. It returns an error for a Magic Initiate spell
// missing from resolved.
func checkFeatChoices(shape featShape, choices FeatChoices, grant FeatGrant, feats TakenFeats, resolved map[content.SpellID]content.Spell) ([]FeatChoiceProblem, error) {
	if !shape.matches(choices) {
		kind := FeatProblemUnexpected
		if _, none := choices.(NoFeatChoices); none {
			kind = FeatProblemMissing
		}
		return []FeatChoiceProblem{{FeatChoiceFeatChoicesShape, kind}}, nil
	}
	switch v := choices.(type) {
	case SingleAbilityIncrease:
		if !slices.Contains(shape.abilities, v.ability) {
			return []FeatChoiceProblem{{FeatChoiceIncreasedAbility, FeatProblemAbilityNotAllowed}}, nil
		}
	case AbilityScoreImprovementChoices:
		if _, err := v.Increase(); err != nil { // the zero value
			return []FeatChoiceProblem{{FeatChoiceIncreasedAbility, FeatProblemAbilityNotAllowed}}, nil
		}
	case SkilledChoices:
		if v.picks[0] == nil { // the zero value
			return []FeatChoiceProblem{{FeatChoiceSkilledPicks, FeatProblemMissing}}, nil
		}
	case MagicInitiateChoices:
		return checkMagicInitiate(v, grant, feats, resolved)
	}
	return nil, nil
}

// magicInitiateListAllowed reports whether list is Cleric, Druid or Wizard.
func magicInitiateListAllowed(list content.ClassID) bool {
	return slices.Contains([]content.ClassID{"cleric", "druid", "wizard"}, list)
}

func checkMagicInitiate(v MagicInitiateChoices, grant FeatGrant, feats TakenFeats, resolved map[content.SpellID]content.Spell) ([]FeatChoiceProblem, error) {
	var out []FeatChoiceProblem
	add := func(n FeatChoiceName, k FeatChoiceProblemKind) { out = append(out, FeatChoiceProblem{n, k}) }
	spell := func(id content.SpellID) (content.Spell, error) {
		s, ok := resolved[id]
		if !ok {
			return content.Spell{}, invalidInput("magic initiate spell not resolved")
		}
		return s, nil
	}
	onList := func(s content.Spell) bool { return slices.Contains(s.Classes(), v.list) }
	required, hasRequired := grant.scope.(SpecifiedFeat)
	switch {
	case !magicInitiateListAllowed(v.list):
		add(FeatChoiceSpellList, FeatProblemListNotAllowed)
	case hasRequired && requiredList(required) != "" && requiredList(required) != v.list:
		add(FeatChoiceSpellList, FeatProblemListNotAllowed)
	case feats.magicInitiate(v.list) >= 0:
		add(FeatChoiceSpellList, FeatProblemListAlreadyUsed)
	}
	if !slices.Contains([]ability.Ability{ability.Intelligence, ability.Wisdom, ability.Charisma}, v.ability) {
		add(FeatChoiceSpellcastingAbility, FeatProblemAbilityNotAllowed)
	}
	switch {
	case len(v.cantrips) != 2:
		add(FeatChoiceCantrips, FeatProblemWrongCount)
	case v.cantrips[0] == v.cantrips[1]:
		add(FeatChoiceCantrips, FeatProblemDuplicate)
	}
	var notCantrip, offList bool
	for _, id := range v.cantrips {
		s, err := spell(id)
		if err != nil {
			return nil, err
		}
		notCantrip = notCantrip || s.Level() != 0
		offList = offList || !onList(s)
	}
	if notCantrip {
		add(FeatChoiceCantrips, FeatProblemNotACantrip)
	}
	if offList {
		add(FeatChoiceCantrips, FeatProblemNotOnSpellList)
	}
	s, err := spell(v.levelOne)
	if err != nil {
		return nil, err
	}
	if s.Level() != 1 {
		add(FeatChoiceLevelOneSpell, FeatProblemNotLevelOne)
	}
	if !onList(s) {
		add(FeatChoiceLevelOneSpell, FeatProblemNotOnSpellList)
	}
	return out, nil
}

// requiredList returns the class of the spell list a SpecifiedFeat requires, or "".
func requiredList(s SpecifiedFeat) content.ClassID {
	if r, ok := s.Required.(RequiredSpellListIs); ok {
		return r.Class
	}
	return ""
}
