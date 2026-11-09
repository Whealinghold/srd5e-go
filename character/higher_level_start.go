package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// HigherLevelStartOutcome is the closed outcome of HigherLevelStart.Advance.
// Variants: HigherLevelStartBuilt, HigherLevelStartRefused.
type HigherLevelStartOutcome interface{ isHigherLevelStartOutcome() }

// HigherLevelStartBuilt is a plan applied in full: the Character at the starting
// level, the events the steps produced and the feat grants they consumed. Only
// Advance builds one; the zero value is not an outcome.
type HigherLevelStartBuilt struct {
	character Character
	events    []Event
	consumed  []ConsumedFeatGrant
}

// Character returns the Character after the last step.
func (b HigherLevelStartBuilt) Character() Character { return b.character }

// Events returns a new slice of the events of the steps, in the order of
// character-aggregate FR-058: for each step LevelGained, then SubclassChosen if
// any, then the AbilityScoreChanged events and at most one HitPointMaximumChanged.
func (b HigherLevelStartBuilt) Events() []Event { return slices.Clone(b.events) }

// ConsumedGrants returns a new slice of one ConsumedFeatGrant per applied
// increase, in step order.
func (b HigherLevelStartBuilt) ConsumedGrants() []ConsumedFeatGrant { return slices.Clone(b.consumed) }

// HigherLevelStartRefused is a plan stopped by a rule. Construct it with a keyed
// literal.
type HigherLevelStartRefused struct{ Reason HigherLevelStartRejection }

func (HigherLevelStartBuilt) isHigherLevelStartOutcome()   {}
func (HigherLevelStartRefused) isHigherLevelStartOutcome() {}

// HigherLevelStart is the pure rules service of "Starting at Higher Levels"
// (SRD 5.2.1, "Creating Your Character", p. 24): it folds LevelUp, the subclass
// choice and the ability score increase over a plan of steps applied to a level 1
// Character. The zero value is ready to use. It is stateless and safe for
// concurrent use.
type HigherLevelStart struct{}

// StartingExperience returns the Experience Points a character started at level
// holds: the minimum for the level (0 for level 1, 900 for level 3, 64,000 for
// level 10).
func (HigherLevelStart) StartingExperience(level Level) ExperiencePoints {
	return ExperiencePoints{}.MinimumForLevel(level)
}

// CheckPlanLength returns HigherLevelStartPlanLengthMismatch when steps is not
// level minus 1, and nil otherwise.
func (HigherLevelStart) CheckPlanLength(level Level, steps int) HigherLevelStartRejection {
	if steps == level.Value()-1 {
		return nil
	}
	return HigherLevelStartPlanLengthMismatch{ExpectedSteps: level.Value() - 1, ActualSteps: steps}
}

// Advance applies steps in order to start, a level 1 Character that holds the
// starting Experience Points, to reach level. classes holds the definition of
// the initial class and of every class a step names.
//
// A plan of the wrong length is refused first with
// HigherLevelStartPlanLengthMismatch. Each step is then applied with the rules
// of Character.LevelUp, then SubclassSelection.Choose when it names a subclass,
// then AbilityScoreIncreaser.Apply when it carries an increase. An increase must
// match a feat grant of the class level the step reaches (General with a maximum
// of 20, or Epic Boon with a maximum of 30; a repeated grant counts at every row
// that names it), and the applied increase is reported as one ConsumedFeatGrant.
// The first failing step stops the build and no later step is applied; within a
// step the order is a level-up rejection, a subclass refusal, an increase not
// granted, an increase refused. After the last step every class at or above its
// subclass level without a subclass gives HigherLevelStartSubclassRequired. The
// receiver and start are unchanged.
//
// It returns an error matching srderr.ErrInvalidInput for a zero start, a start
// above level 1, a zero level, a zero step, or a class definition missing from
// classes. It never panics.
func (HigherLevelStart) Advance(start Character, level Level, classes map[content.ClassID]content.ClassDefinition, steps []HigherLevelStep) (HigherLevelStartOutcome, error) {
	if start.IsZero() || level.IsZero() || start.TotalLevel().n != 1 {
		return nil, invalidInput("higher-level start of a zero character, a zero level or a character above level 1")
	}
	if rej := (HigherLevelStart{}).CheckPlanLength(level, len(steps)); rej != nil {
		return HigherLevelStartRefused{Reason: rej}, nil
	}
	requirements := map[content.ClassID]content.AbilityRequirement{}
	for id, def := range classes {
		requirements[id] = def.PrimaryAbility()
	}
	if _, ok := classes[start.InitialClass()]; !ok {
		return nil, invalidInput("higher-level start without the initial class definition")
	}
	cur := start
	var events []Event
	var consumed []ConsumedFeatGrant
	for _, step := range steps {
		def, ok := classes[step.class.class]
		if step.IsZero() || !ok {
			return nil, invalidInput("higher-level step without a class definition")
		}
		next, stepEvents, grant, refusal, err := applyStep(cur, def, step, requirements)
		if err != nil {
			return nil, err
		}
		if refusal != nil {
			return HigherLevelStartRefused{Reason: refusal}, nil
		}
		cur = next
		events = append(events, stepEvents...)
		if grant != nil {
			consumed = append(consumed, *grant)
		}
	}
	var owed []content.ClassID
	for _, e := range cur.classes.entries {
		pending, err := SubclassSelection{}.IsPending(cur, classes[e.Class])
		if err != nil {
			return nil, err
		}
		if pending {
			owed = append(owed, e.Class)
		}
	}
	if len(owed) > 0 {
		return HigherLevelStartRefused{Reason: HigherLevelStartSubclassRequired{classes: owed}}, nil
	}
	return HigherLevelStartBuilt{character: cur, events: events, consumed: consumed}, nil
}

// applyStep applies one step to c. It returns the new Character, the step's
// events, the grant an applied increase consumed (nil for none) and, when a rule
// stops the step, the refusal.
func applyStep(c Character, def content.ClassDefinition, step HigherLevelStep, requirements map[content.ClassID]content.AbilityRequirement) (Character, []Event, *ConsumedFeatGrant, HigherLevelStartRejection, error) {
	level, err := NewLevel(c.TotalLevel().n + 1)
	if err != nil { // unreachable: the plan length keeps the total level at 20 or below
		return Character{}, nil, nil, nil, err
	}
	tr, err := c.LevelUp(step.class, step.gain, requirements)
	if err != nil {
		return Character{}, nil, nil, nil, err
	}
	lv, ok := tr.(TransitionChanged[LevelUpRejection])
	if !ok {
		return Character{}, nil, nil, HigherLevelStartLevelUpRejected{AtLevel: level, Rejection: tr.(TransitionRejected[LevelUpRejection]).rejection}, nil
	}
	cur, events := lv.character, slices.Clone(lv.events)
	if sub, has := step.Subclass(); has {
		tr, err := SubclassSelection{}.Choose(cur, def, sub)
		if err != nil {
			return Character{}, nil, nil, nil, err
		}
		ch, ok := tr.(TransitionChanged[SubclassRejected])
		if !ok {
			return Character{}, nil, nil, HigherLevelStartSubclassRejected{AtLevel: level, Rejection: tr.(TransitionRejected[SubclassRejected]).rejection}, nil
		}
		cur, events = ch.character, append(events, ch.events...)
	}
	inc, has := step.Increase()
	if !has {
		return cur, events, nil, nil, nil
	}
	classLevel := cur.classes.LevelIn(def.ID())
	feature, granted, err := grantFor(def, classLevel, inc)
	if err != nil {
		return Character{}, nil, nil, nil, err
	}
	if !granted {
		return Character{}, nil, nil, HigherLevelStartIncreaseNotGranted{AtLevel: level}, nil
	}
	tr2, err := AbilityScoreIncreaser{}.Apply(cur, inc)
	if err != nil {
		return Character{}, nil, nil, nil, err
	}
	applied, ok := tr2.(TransitionChanged[AbilityScoreIncreaseRejection])
	if !ok {
		return Character{}, nil, nil, HigherLevelStartIncreaseRejected{AtLevel: level, Rejection: tr2.(TransitionRejected[AbilityScoreIncreaseRejection]).rejection}, nil
	}
	return applied.character, append(events, applied.events...), &ConsumedFeatGrant{Class: def.ID(), ClassLevel: classLevel, Feature: feature}, nil, nil
}

// grantFor returns the id of the feature entry of the row at classLevel whose
// feat grant admits inc: General with a maximum score of at most 20, or Epic Boon
// with a maximum of at most 30 (SRD 5.2.1, "Ability Score Improvement", p. 87).
func grantFor(def content.ClassDefinition, classLevel int, inc AbilityScoreIncrease) (content.FeatureID, bool, error) {
	row, err := def.RowFeatures(classLevel)
	if err != nil { // unreachable: the class level is 1 to 20
		return "", false, err
	}
	for _, e := range row.Entries() {
		for _, f := range e.Facts() {
			g, ok := f.(content.GrantsFeat)
			if !ok {
				continue
			}
			if (g.Category() == content.FeatGeneral && inc.maximum <= DefaultScoreCap) || (g.Category() == content.FeatEpicBoon && inc.maximum <= HighestScoreCap) {
				return e.ID(), true, nil
			}
		}
	}
	return "", false, nil
}
