package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// LevelAdvancementPreview is the closed outcome of previewing a level (SRD
// 5.2.1, "Gaining a Level", p. 23). It never changes a Character. Variants:
// LevelAdvancementGain, LevelAdvancementBlock.
type LevelAdvancementPreview interface{ isLevelAdvancementPreview() }

// LevelAdvancementGain is a level the character can gain and what it gives.
// Only the previewer builds one; the zero value is not a preview.
type LevelAdvancementGain struct {
	status          AdvancementClassStatus
	newClassLevel   int
	newTotalLevel   Level
	proficiency     ProficiencyBonus
	proficiencyUp   bool
	die             content.HitPointDie
	features        []content.FeatureEntry
	subclassFeature bool
	pending         []PendingAdvancementChoice
	slots           SpellSlotPools
}

// ClassStatus returns whether the level is in a held class or a new one.
func (g LevelAdvancementGain) ClassStatus() AdvancementClassStatus { return g.status }

// NewClassLevel returns the level in the class after the level.
func (g LevelAdvancementGain) NewClassLevel() int { return g.newClassLevel }

// NewTotalLevel returns the total level after the level.
func (g LevelAdvancementGain) NewTotalLevel() Level { return g.newTotalLevel }

// ProficiencyBonusAfter returns the Proficiency Bonus at the new total level.
func (g LevelAdvancementGain) ProficiencyBonusAfter() ProficiencyBonus { return g.proficiency }

// ProficiencyBonusIncreases reports whether the bonus is higher than before.
func (g LevelAdvancementGain) ProficiencyBonusIncreases() bool { return g.proficiencyUp }

// HitPointDie returns the Hit Point Die of the class the level is gained in.
func (g LevelAdvancementGain) HitPointDie() content.HitPointDie { return g.die }

// FeaturesGained returns a new slice of the class features of the new class
// level's row, in printed order. A repeated feature is its older entry.
func (g LevelAdvancementGain) FeaturesGained() []content.FeatureEntry {
	return slices.Clone(g.features)
}

// HasSubclassFeature reports whether the row names a subclass feature.
func (g LevelAdvancementGain) HasSubclassFeature() bool { return g.subclassFeature }

// PendingChoices returns a new slice of the choices the level leaves open: the
// subclass owed, then one feat grant per General or Epic Boon feat granted.
func (g LevelAdvancementGain) PendingChoices() []PendingAdvancementChoice {
	return slices.Clone(g.pending)
}

// SpellSlotsAfter returns the full Spell Slots of both pools after the level.
func (g LevelAdvancementGain) SpellSlotsAfter() SpellSlotPools { return g.slots }

// LevelAdvancementBlock is a level the character cannot gain. Only the previewer
// builds one; the zero value is not a preview.
type LevelAdvancementBlock struct {
	status     AdvancementClassStatus
	rejection  LevelUpRejection
	stillNeeds int
}

// ClassStatus returns whether the level would be in a held class or a new one.
func (b LevelAdvancementBlock) ClassStatus() AdvancementClassStatus { return b.status }

// Rejection returns the reason Character.LevelUp refuses the level.
func (b LevelAdvancementBlock) Rejection() LevelUpRejection { return b.rejection }

// ExperienceStillNeeded returns the Experience Points still needed for the next
// total level, 0 when there are enough or at total level 20.
func (b LevelAdvancementBlock) ExperienceStillNeeded() int { return b.stillNeeds }

func (LevelAdvancementGain) isLevelAdvancementPreview()  {}
func (LevelAdvancementBlock) isLevelAdvancementPreview() {}

// LevelAdvancementPreviewer previews gaining a level in a class. It is pure and
// stateless; the zero value is ready to use and it is safe for concurrent use.
type LevelAdvancementPreviewer struct{}

// Preview previews a level of class for c without changing c. Availability,
// the rejection, the levels and the Proficiency Bonus are those of
// Character.LevelUp called with the class's fixed Hit Point value, so they
// cannot disagree with the aggregate. requirements holds the Primary Ability of
// every class c holds and is read only when class is new to c.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, a
// class that cannot build an AdvancementClass, or, for a new class, a
// requirements map lacking a current class. It never panics.
func (LevelAdvancementPreviewer) Preview(c Character, class content.ClassDefinition, requirements map[content.ClassID]content.AbilityRequirement) (LevelAdvancementPreview, error) {
	if c.IsZero() {
		return nil, invalidInput("level preview of a zero character")
	}
	adv, err := NewAdvancementClass(class.ID(), class.HitPointDie(), class.PrimaryAbility())
	if err != nil {
		return nil, err
	}
	gain, err := NewFixedHitPointGain(class.HitPointDie().Faces()/2 + 1)
	if err != nil {
		return nil, err
	}
	tr, err := c.LevelUp(adv, gain, requirements)
	if err != nil {
		return nil, err
	}
	current := c.classes.LevelIn(class.ID())
	var status AdvancementClassStatus = AdvancementClassStatusExisting{CurrentClassLevel: current}
	if current == 0 {
		status = AdvancementClassStatusNew{grants: class.MulticlassGrants()}
	}
	switch t := tr.(type) {
	case TransitionRejected[LevelUpRejection]:
		return LevelAdvancementBlock{status: status, rejection: t.rejection, stillNeeds: c.experienceStillNeeded()}, nil
	case TransitionChanged[LevelUpRejection]:
		return c.gainOf(t.character, class, status)
	default:
		panic(invalidInput("unknown transition")) // unreachable: Transition is closed
	}
}

// experienceStillNeeded is the Experience Points missing for the next total
// level, 0 when enough or at the maximum level.
func (c Character) experienceStillNeeded() int {
	next, err := NewLevel(c.TotalLevel().n + 1)
	if err != nil { // the maximum level has no next level
		return 0
	}
	return max(0, ExperiencePoints{}.MinimumForLevel(next).n-c.xp.n)
}

// gainOf describes the level that took c to next in class.
func (c Character) gainOf(next Character, class content.ClassDefinition, status AdvancementClassStatus) (LevelAdvancementGain, error) {
	classLevel := next.classes.LevelIn(class.ID())
	row, err := class.RowFeatures(classLevel)
	if err != nil { // unreachable: the class level is 1 to 20
		return LevelAdvancementGain{}, err
	}
	pending, err := c.pendingChoices(class, classLevel, row.Entries())
	if err != nil {
		return LevelAdvancementGain{}, err
	}
	return LevelAdvancementGain{
		status: status, newClassLevel: classLevel, newTotalLevel: next.TotalLevel(),
		proficiency: next.ProficiencyBonus(), proficiencyUp: next.ProficiencyBonus().n > c.ProficiencyBonus().n,
		die: class.HitPointDie(), features: row.Entries(), subclassFeature: row.HasSubclassFeature(),
		pending: pending, slots: SlotsFor(next.classes),
	}, nil
}

// pendingChoices lists the subclass owed at classLevel and the General and Epic
// Boon feat grants of the row's entries, matched by typed fact.
func (c Character) pendingChoices(class content.ClassDefinition, classLevel int, entries []content.FeatureEntry) ([]PendingAdvancementChoice, error) {
	out := []PendingAdvancementChoice{}
	key, err := SubclassStateKey{}.ForClass(class.ID())
	if err != nil {
		return nil, err
	}
	_, held, err := ReadFeatureState(c.states, key)
	if err != nil {
		return nil, err
	}
	if classLevel >= class.SubclassLevel() && !held {
		out = append(out, PendingSubclassChoice{Class: class.ID(), ClassLevel: classLevel})
	}
	for _, e := range entries {
		for _, f := range e.Facts() {
			g, ok := f.(content.GrantsFeat)
			if ok && (g.Category() == content.FeatGeneral || g.Category() == content.FeatEpicBoon) {
				out = append(out, PendingFeatGrant{Class: class.ID(), ClassLevel: classLevel, Feature: e.ID(), Category: g.Category()})
			}
		}
	}
	return out, nil
}
