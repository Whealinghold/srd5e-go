package combat

// RetainedStatistic is a statistic a creature keeps in a new form; every other game
// statistic is replaced by the form's stat block (SRD 5.2.1, "Polymorph", p. 153). The zero
// value is invalid.
type RetainedStatistic int

// The statistics the four spells let a creature keep.
const (
	RetainedAlignment RetainedStatistic = iota + 1
	RetainedPersonality
	RetainedCreatureType
	RetainedHitPoints
	RetainedHitPointDice
	// RetainedMentalScores is the Intelligence, Wisdom and Charisma scores.
	RetainedMentalScores
	RetainedProficiencies
	RetainedAbilityToCommunicate
	// RetainedSpellcastingFeature is the Spellcasting feature, kept by Shapechange when held.
	RetainedSpellcastingFeature
)

// String returns the snake_case name, or "unknown".
func (s RetainedStatistic) String() string {
	return enumName([]string{"alignment", "personality", "creature_type", "hit_points", "hit_point_dice",
		"mental_scores", "proficiencies", "ability_to_communicate", "spellcasting_feature"}, int(s))
}

// FormStatistics is the set of statistics a creature keeps in a form. Every other game
// statistic is replaced by the form's stat block. It is comparable with ==, immutable and
// safe for concurrent use; the zero value keeps nothing.
type FormStatistics struct{ kept uint16 }

func keepStatistics(ss ...RetainedStatistic) FormStatistics {
	var f FormStatistics
	for _, s := range ss {
		f.kept |= 1 << s
	}
	return f
}

// Retained returns the kept statistics in the order of the RetainedStatistic enum, as a new slice.
func (f FormStatistics) Retained() []RetainedStatistic {
	var out []RetainedStatistic
	for s := RetainedAlignment; s <= RetainedSpellcastingFeature; s++ {
		if f.kept&(1<<s) != 0 {
			out = append(out, s)
		}
	}
	return out
}

// Keeps reports whether s is kept.
func (f FormStatistics) Keeps(s RetainedStatistic) bool { return f.kept&(1<<s) != 0 }

// ReplacesEveryOther reports that each statistic not kept is replaced by the form's stat
// block: always true.
func (FormStatistics) ReplacesEveryOther() bool { return true }

// SpeechRestriction is what a form does to speech. The zero value means the spell and mode
// state none (a True Polymorph into or out of an object).
type SpeechRestriction int

// The speech restrictions.
const (
	FormCannotSpeak SpeechRestriction = iota + 1
	FormAbilityToCommunicateRetained
)

// SpellcastingRestriction is what a form does to spellcasting. The zero value means the
// spell and mode state none.
type SpellcastingRestriction int

// The spellcasting restrictions.
const (
	FormCannotCast SpellcastingRestriction = iota + 1
	FormSpellcastingFeatureRetained
	// FormNoSpellcastingFeature: a Shapechange caster without the Spellcasting feature; nothing
	// is retained and the spell forbids nothing.
	FormNoSpellcastingFeature
)

// FormRestrictions is what a form forbids (table B of shape-changing-spells.md). Construct
// it only through RestrictionsOf; comparable.
type FormRestrictions struct {
	// ActionsLimitedByAnatomy: the creature can only take actions its form's anatomy allows.
	ActionsLimitedByAnatomy bool
	// Speech is the speech restriction.
	Speech SpeechRestriction
	// Spellcasting is the spellcasting restriction.
	Spellcasting SpellcastingRestriction
}

// StatisticsOf returns the statistics a creature keeps under spell (and mode, for True
// Polymorph), per table B (SRD 5.2.1, "Polymorph", p. 153; "True Polymorph", p. 170;
// "Shapechange", p. 161; "Animal Shapes", p. 108). Shapechange lists the Spellcasting
// feature as kept "if held". True Polymorph into or out of an object keeps nothing: the
// object's statistics replace them (shape-changing-spells.md Q-7).
//
// It returns an error matching srderr.ErrInvalidInput for an invalid spell, or for True
// Polymorph with an invalid mode. It never panics.
func (ShapeShiftSpells) StatisticsOf(spell ShapeShiftSpell, mode TrueTransformation) (FormStatistics, error) {
	switch spell {
	case PolymorphSpell:
		return keepStatistics(RetainedAlignment, RetainedPersonality, RetainedCreatureType, RetainedHitPoints, RetainedHitPointDice), nil
	case TruePolymorphSpell:
		switch mode {
		case CreatureIntoCreature:
			return keepStatistics(RetainedHitPoints, RetainedHitPointDice, RetainedAlignment, RetainedPersonality), nil
		case ObjectIntoCreature, CreatureIntoObject:
			return FormStatistics{}, nil
		}
		return FormStatistics{}, invalidInput("True Polymorph needs a mode")
	case ShapechangeSpell:
		return keepStatistics(RetainedCreatureType, RetainedAlignment, RetainedPersonality, RetainedMentalScores, RetainedHitPoints,
			RetainedHitPointDice, RetainedProficiencies, RetainedAbilityToCommunicate, RetainedSpellcastingFeature), nil
	case AnimalShapesSpell:
		return keepStatistics(RetainedCreatureType, RetainedHitPoints, RetainedHitPointDice, RetainedAlignment,
			RetainedAbilityToCommunicate, RetainedMentalScores), nil
	}
	return FormStatistics{}, invalidInput("a shape-shifting spell is needed")
}

// RestrictionsOf returns what a form forbids under spell and mode, per table B. Shapechange
// states none of the three limits (shape-changing-spells.md Q-6): it retains the ability to
// communicate and, when holdsSpellcastingFeature, the Spellcasting feature; without the
// feature it reports FormNoSpellcastingFeature. holdsSpellcastingFeature is ignored for the
// other spells. True Polymorph into or out of an object returns the zero FormRestrictions.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid spell, or for True
// Polymorph with an invalid mode. It never panics.
func (ShapeShiftSpells) RestrictionsOf(spell ShapeShiftSpell, mode TrueTransformation, holdsSpellcastingFeature bool) (FormRestrictions, error) {
	switch spell {
	case PolymorphSpell:
		return FormRestrictions{true, FormCannotSpeak, FormCannotCast}, nil
	case TruePolymorphSpell:
		switch mode {
		case CreatureIntoCreature:
			return FormRestrictions{true, FormCannotSpeak, FormCannotCast}, nil
		case ObjectIntoCreature, CreatureIntoObject:
			return FormRestrictions{}, nil
		}
		return FormRestrictions{}, invalidInput("True Polymorph needs a mode")
	case ShapechangeSpell:
		if holdsSpellcastingFeature {
			return FormRestrictions{false, FormAbilityToCommunicateRetained, FormSpellcastingFeatureRetained}, nil
		}
		return FormRestrictions{false, FormAbilityToCommunicateRetained, FormNoSpellcastingFeature}, nil
	case AnimalShapesSpell:
		return FormRestrictions{true, FormAbilityToCommunicateRetained, FormCannotCast}, nil
	}
	return FormRestrictions{}, invalidInput("a shape-shifting spell is needed")
}
