package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// maxStartingScore is the highest score a new character starts with (SRD 5.2.1,
// "Ability Scores", p. 5).
const maxStartingScore = 20

// NewParams are the arguments of New. Construct it with a keyed literal.
// ExperiencePoints defaults to 0, HeroicInspiration to NotInspired and
// FeatureStates to empty. HitPointDie is the Hit Point Die of Class.
type NewParams struct {
	ID                Identifier
	Class             content.ClassID
	HitPointDie       content.HitPointDie
	Species           content.SpeciesID
	Background        content.BackgroundID
	AbilityScores     ability.Scores
	ExperiencePoints  ExperiencePoints
	HeroicInspiration HeroicInspiration
	FeatureStates     FeatureStates
}

// NewCharacter is the outcome of New: the level 1 Character and the one
// Created event.
type NewCharacter struct {
	character Character
	events    []Event
}

// Character returns the new Character.
func (n NewCharacter) Character() Character { return n.character }

// Events returns a new slice holding the one Created event.
func (n NewCharacter) Events() []Event { return slices.Clone(n.events) }

// CreationProblem is one reason New refused its arguments. It is a closed set
// and every variant is an error matching srderr.ErrInvalidInput. Variants:
// InvalidCreationArgument, StartingScoreAboveMaximum.
type CreationProblem interface {
	error
	isCreationProblem()
}

// CreationArgument names an argument of New. The zero value is invalid.
type CreationArgument int

// The arguments of New that can be invalid on their own.
const (
	// CreationID: the Character id.
	CreationID CreationArgument = iota + 1
	// CreationClass: the initial class id.
	CreationClass
	// CreationHitPointDie: the Hit Point Die.
	CreationHitPointDie
	// CreationSpecies: the species id.
	CreationSpecies
	// CreationBackground: the background id.
	CreationBackground
	// CreationAbilityScores: the ability scores.
	CreationAbilityScores
	// CreationHeroicInspiration: the Heroic Inspiration value.
	CreationHeroicInspiration
)

// String returns the argument's name or "unknown".
func (a CreationArgument) String() string {
	names := [...]string{"unknown", "id", "class", "hitPointDie", "species", "background", "abilityScores", "heroicInspiration"}
	if a < 1 || int(a) >= len(names) {
		return names[0]
	}
	return names[a]
}

// InvalidCreationArgument is the problem of an argument that is zero or not a
// valid value of its type.
//
//nolint:errname // a variant of the CreationProblem family, named by the spec
type InvalidCreationArgument struct{ Argument CreationArgument }

// Error renders the problem. The text is not a contract.
func (p InvalidCreationArgument) Error() string {
	return "character: invalid creation argument: " + p.Argument.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (InvalidCreationArgument) Unwrap() error { return srderr.ErrInvalidInput }

func (InvalidCreationArgument) isCreationProblem() {}

// StartingScoreAboveMaximum is the problem of starting ability scores above 20
// (SRD 5.2.1, "Ability Scores", p. 5). Build it with NewStartingScoreAboveMaximum;
// the zero value lists nothing and is invalid.
//
//nolint:errname // a variant of the CreationProblem family, named by the spec
type StartingScoreAboveMaximum struct{ abilities []ability.Ability }

// NewStartingScoreAboveMaximum returns the problem for abilities, kept in
// declaration order without duplicates. It returns an error matching
// srderr.ErrInvalidInput for an empty list or an invalid ability. It never
// panics.
func NewStartingScoreAboveMaximum(abilities []ability.Ability) (StartingScoreAboveMaximum, error) {
	if len(abilities) == 0 {
		return StartingScoreAboveMaximum{}, invalidInput("starting score problem without abilities")
	}
	var kept []ability.Ability
	for _, a := range ability.All() {
		if slices.Contains(abilities, a) {
			kept = append(kept, a)
		}
	}
	for _, a := range abilities {
		if !a.Valid() {
			return StartingScoreAboveMaximum{}, invalidInput("starting score problem with an invalid ability")
		}
	}
	return StartingScoreAboveMaximum{abilities: kept}, nil
}

// MustStartingScoreAboveMaximum is NewStartingScoreAboveMaximum for known-valid
// input. It panics with the error NewStartingScoreAboveMaximum returns, and
// only then.
func MustStartingScoreAboveMaximum(abilities []ability.Ability) StartingScoreAboveMaximum {
	p, err := NewStartingScoreAboveMaximum(abilities)
	if err != nil {
		panic(err)
	}
	return p
}

// Abilities returns a new slice of the abilities above 20, in declaration order.
func (p StartingScoreAboveMaximum) Abilities() []ability.Ability { return slices.Clone(p.abilities) }

// Error renders the problem. The text is not a contract.
func (p StartingScoreAboveMaximum) Error() string {
	return "character: starting ability score above 20"
}

// Unwrap returns srderr.ErrInvalidInput.
func (StartingScoreAboveMaximum) Unwrap() error { return srderr.ErrInvalidInput }

func (StartingScoreAboveMaximum) isCreationProblem() {}

// New returns a level 1 Character in one class with 0 XP unless p says
// otherwise, and the one Created event (SRD 5.2.1, "Create Your Character",
// p. 19). The Hit Point maximum is the largest face of the Hit Point Die plus
// the Constitution modifier and the pool holds one Hit Die of that size (p.
// 22). A character that starts higher is created with the XP of that level and
// then gains levels with LevelUp.
//
// It reports every problem at once: on failure the error is the errors.Join of
// the CreationProblem values, InvalidCreationArgument first, each variant at
// most once, matching srderr.ErrInvalidInput. It never panics.
func New(p NewParams) (NewCharacter, error) {
	var bad []ability.Ability
	var problems []error
	invalid := func(a CreationArgument) { problems = append(problems, InvalidCreationArgument{Argument: a}) }
	inspiration := p.HeroicInspiration
	if inspiration == 0 {
		inspiration = NotInspired
	}
	for _, c := range []struct {
		ok  bool
		arg CreationArgument
	}{
		{!p.ID.IsZero(), CreationID}, {p.Class.Valid(), CreationClass}, {p.HitPointDie.Valid(), CreationHitPointDie},
		{p.Species.Valid(), CreationSpecies}, {p.Background.Valid(), CreationBackground},
		{!p.AbilityScores.IsZero(), CreationAbilityScores}, {inspiration.Valid(), CreationHeroicInspiration},
	} {
		if !c.ok {
			invalid(c.arg)
		}
	}
	if !p.AbilityScores.IsZero() {
		for _, a := range ability.All() {
			if p.AbilityScores.Score(a).Value() > maxStartingScore {
				bad = append(bad, a)
			}
		}
	}
	if len(bad) > 0 {
		problems = append(problems, MustStartingScoreAboveMaximum(bad))
	}
	if len(problems) > 0 {
		return NewCharacter{}, errors.Join(problems...)
	}
	maximum, err := NewHitPointMaximum(p.HitPointDie.Faces() + p.AbilityScores.Modifier(ability.Constitution).Value())
	if err != nil { // unreachable: the smallest die gives 6 and the lowest modifier is -5
		return NewCharacter{}, err
	}
	pool, err := NewHitDicePool(map[content.HitPointDie]int{p.HitPointDie: 1})
	if err != nil { // unreachable: the die was checked above
		return NewCharacter{}, err
	}
	c := Character{
		id: p.ID, xp: p.ExperiencePoints, classes: ClassLevels{entries: []ClassLevel{{Class: p.Class, Level: 1}}},
		species: p.Species, background: p.Background, scores: p.AbilityScores, hitPoints: maximum, pool: pool,
		inspiration: inspiration, states: p.FeatureStates,
	}
	created := Created{
		id: c.id, class: p.Class, species: p.Species, background: p.Background,
		scores: c.scores, experience: c.xp, hitPointMaximum: maximum,
	}
	return NewCharacter{character: c, events: []Event{created}}, nil
}

// MustNew is New for arguments known to be valid. It panics with the error New
// returns, and only then.
func MustNew(p NewParams) NewCharacter {
	n, err := New(p)
	if err != nil {
		panic(err)
	}
	return n
}
