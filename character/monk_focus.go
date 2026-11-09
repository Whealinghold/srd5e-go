package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// MonkSaveDC returns the save DC of the Monk's Focus features: 8 plus the Wisdom
// modifier plus the Proficiency Bonus (SRD 5.2.1, "Level 2: Monk's Focus", p. 50).
// The bonus is the character's, from the total level, not a Monk table value
// (class-monk.md FR-018).
func MonkSaveDC(wisdom ability.Modifier, bonus ProficiencyBonus) d20.DifficultyClass {
	return d20.DifficultyClass(8 + wisdom.Value() + bonus.Value())
}

// MonkFocusFeature is one of the three features the Monk knows from level 2
// (SRD 5.2.1, "Level 2: Monk's Focus", p. 50). The zero value is not a feature.
type MonkFocusFeature int

// The Focus Point features.
const (
	// MonkFlurryOfBlows: two Unarmed Strikes as a Bonus Action for 1 Focus Point.
	MonkFlurryOfBlows MonkFocusFeature = iota + 1
	// MonkPatientDefense: the Disengage action, or with 1 Focus Point also the Dodge action.
	MonkPatientDefense
	// MonkStepOfTheWind: the Dash action, or with 1 Focus Point also the Disengage action and a doubled jump.
	MonkStepOfTheWind
)

// String returns "flurry_of_blows", "patient_defense", "step_of_the_wind" or "unknown".
func (f MonkFocusFeature) String() string {
	switch f {
	case MonkFlurryOfBlows:
		return "flurry_of_blows"
	case MonkPatientDefense:
		return "patient_defense"
	case MonkStepOfTheWind:
		return "step_of_the_wind"
	}
	return "unknown"
}

// Valid reports whether f is one of the three features.
func (f MonkFocusFeature) Valid() bool { return f >= MonkFlurryOfBlows && f <= MonkStepOfTheWind }

// MonkCarriedCreature is the host's report of the creature a Monk of level 10 or
// more chooses to move with them using Step of the Wind (SRD 5.2.1, "Level 10:
// Heightened Focus", p. 52). The zero value chooses none. Construct it with a keyed
// literal.
type MonkCarriedCreature struct {
	// Chosen reports that the Monk chooses a creature.
	Chosen bool
	// Willing reports that the creature is willing.
	Willing bool
	// Size is the creature's size.
	Size content.CreatureSize
	// DistanceFeet is the distance from the Monk in feet.
	DistanceFeet int
}

// MonkFocusUse is one use of Flurry of Blows, Patient Defense or Step of the Wind.
// Construct it with a keyed literal.
type MonkFocusUse struct {
	// Feature is the feature used.
	Feature MonkFocusFeature
	// SpendPoint reports that 1 Focus Point is expended. Flurry of Blows always costs it.
	SpendPoint bool
	// Carried is the creature moved with the Monk; the zero value chooses none.
	Carried MonkCarriedCreature
}

// MonkFocus decides and resolves the Monk's Focus features (SRD 5.2.1, "Level 2:
// Monk's Focus", p. 50, "Level 10: Heightened Focus", pp. 51-52). It is stateless and
// pure; the zero value is ready to use. Safe for concurrent use.
type MonkFocus struct{}

func (MonkFocus) carryAllowed(stats MonkLevelStats, use MonkFocusUse) bool {
	c := use.Carried
	return use.Feature == MonkStepOfTheWind && use.SpendPoint && stats.level.n >= monkHeightenedLevel && c.Willing &&
		c.Size.Valid() && c.Size <= content.SizeLarge && c.DistanceFeet >= 0 && c.DistanceFeet <= 5
}

// Availability reports whether the Monk of stats' level with state can make use:
// MonkNotAMonk for zero stats; MonkLevelTooLow below level 2; MonkNotEligible for an
// unknown feature, a Flurry of Blows without the Focus Point, or a carried creature
// that is not allowed (below level 10, not Step of the Wind with the Focus Point, not
// willing, above Large or beyond 5 feet); MonkNotEnoughFocusPoints when the point is
// not available.
func (f MonkFocus) Availability(stats MonkLevelStats, state MonkState, use MonkFocusUse) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkFocusLevel),
		monkIf(!use.Feature.Valid() || (use.Feature == MonkFlurryOfBlows && !use.SpendPoint) ||
			(use.Carried.Chosen && !f.carryAllowed(stats, use)), MonkNotEligible),
		monkIf(use.SpendPoint && state.FocusPoints(stats).Available() < 1, MonkNotEnoughFocusPoints),
	)
}

// TemporaryHitPointDice returns how many Martial Arts dice the use rolls: 2 for
// Patient Defense with the Focus Point from level 10, else 0.
func (MonkFocus) TemporaryHitPointDice(stats MonkLevelStats, use MonkFocusUse) int {
	if use.Feature == MonkPatientDefense && use.SpendPoint && stats.level.n >= monkHeightenedLevel {
		return 2
	}
	return 0
}

// MonkFocusResolution is the outcome of a use of a Focus feature. Only
// MonkFocus.Resolve builds one.
type MonkFocusResolution struct {
	feature     MonkFocusFeature
	spent       int
	strikes     int
	actions     []MonkAction
	jumpDoubled bool
	temporary   int
	carried     bool
	state       MonkState
}

// Feature returns the feature used.
func (r MonkFocusResolution) Feature() MonkFocusFeature { return r.feature }

// FocusPointsSpent returns the Focus Points expended: 0 or 1.
func (r MonkFocusResolution) FocusPointsSpent() int { return r.spent }

// UnarmedStrikes returns the Unarmed Strikes of a Flurry of Blows: 2, or 3 from level 10; 0 for the others.
func (r MonkFocusResolution) UnarmedStrikes() int { return r.strikes }

// Actions returns the actions the use grants as Bonus Actions, as a new slice.
func (r MonkFocusResolution) Actions() []MonkAction { return slices.Clone(r.actions) }

// JumpDoubled reports that the jump distance is doubled for the turn.
func (r MonkFocusResolution) JumpDoubled() bool { return r.jumpDoubled }

// TemporaryHitPoints returns the Temporary Hit Points gained: two rolls of the Martial Arts die from level 10, else 0.
func (r MonkFocusResolution) TemporaryHitPoints() int { return r.temporary }

// CarriesCreature reports that the chosen willing creature moves with the Monk without provoking Opportunity Attacks.
func (r MonkFocusResolution) CarriesCreature() bool { return r.carried }

// State returns the new state: the Focus Point expended.
func (r MonkFocusResolution) State() MonkState { return r.state }

// Resolve makes the use. temporaryRolled is the total of the Martial Arts dice
// TemporaryHitPointDice reports (two dice), or 0 when it reports none.
//
// It returns an error matching srderr.ErrInvalidInput when Availability says
// unavailable, or temporaryRolled is not a possible total of the dice (or is not 0
// when no die is rolled). It never panics.
func (f MonkFocus) Resolve(stats MonkLevelStats, state MonkState, use MonkFocusUse, temporaryRolled int) (MonkFocusResolution, error) {
	if _, ok := f.Availability(stats, state, use).(MonkAvailable); !ok {
		return MonkFocusResolution{}, invalidInput("Monk Focus feature resolved while unavailable")
	}
	n := f.TemporaryHitPointDice(stats, use)
	if (n == 0 && temporaryRolled != 0) || (n > 0 && !monkRolledWithin(temporaryRolled, n, stats.martialArts)) {
		return MonkFocusResolution{}, invalidInput("Patient Defense Temporary Hit Points roll outside the Martial Arts dice")
	}
	r := MonkFocusResolution{feature: use.Feature, temporary: temporaryRolled, state: state, carried: use.Carried.Chosen}
	if use.SpendPoint {
		r.spent = 1
		r.state.focusExpended++
	}
	switch use.Feature {
	case MonkFlurryOfBlows:
		r.strikes = 2 + thresholdCount(stats.level.n, monkHeightenedLevel)
	case MonkPatientDefense:
		r.actions = []MonkAction{MonkDisengage}
		if use.SpendPoint {
			r.actions = append(r.actions, MonkDodge)
		}
	case MonkStepOfTheWind:
		r.actions = []MonkAction{MonkDash}
		if use.SpendPoint {
			r.actions, r.jumpDoubled = []MonkAction{MonkDisengage, MonkDash}, true
		}
	}
	return r, nil
}

// MonkAbilities are the numbers the Monk's saving throws and damage use: the
// Dexterity and Wisdom modifiers and the Proficiency Bonus of the total level
// (class-monk.md FR-018). Build it with MonkAbilitiesOf or a keyed literal. A value;
// comparable with ==.
type MonkAbilities struct {
	Dexterity, Wisdom ability.Modifier
	Proficiency       ProficiencyBonus
}

// MonkAbilitiesOf reads the modifiers from the base ability scores of c and the
// Proficiency Bonus from its total level. A host that applies score bonuses (magic
// items) builds the MonkAbilities itself.
func MonkAbilitiesOf(c Character) MonkAbilities {
	return MonkAbilities{
		Dexterity: c.AbilityScores().Modifier(ability.Dexterity), Wisdom: c.AbilityScores().Modifier(ability.Wisdom),
		Proficiency: c.ProficiencyBonus(),
	}
}

// SaveDC returns the save DC of the Monk's Focus features, MonkSaveDC of the Wisdom
// modifier and the Proficiency Bonus.
func (a MonkAbilities) SaveDC() d20.DifficultyClass { return MonkSaveDC(a.Wisdom, a.Proficiency) }
