package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	// SorcererClassID is the id of the Sorcerer class.
	SorcererClassID content.ClassID = "sorcerer"
	// DraconicSorcerySubclassID is the id of the Draconic Sorcery subclass, the only Sorcerer subclass in SRD 5.2.1.
	DraconicSorcerySubclassID content.SubclassID = "draconic_sorcery"
)

// The Sorcerer levels at which features arrive (SRD 5.2.1, "Sorcerer Features", p. 65).
const (
	sorcererFontOfMagicLevel    = 2
	sorcererSubclassLevel       = 3
	sorcererRestorationLevel    = 5
	sorcererIncarnateLevel      = 7
	sorcererApotheosisLevel     = 20
	draconicResilienceLevel     = 3
	draconicAffinityLevel       = 6
	draconicWingsLevel          = 14
	draconicCompanionLevel      = 18
	innateSorceryUses           = 2
	innateSorceryPointCost      = 2
	dragonWingsRestorePointCost = 3
)

// SorcererLevel is a level in the Sorcerer class, 1 to 20 (SRD 5.2.1, "Sorcerer Features", p. 65). It counts Sorcerer
// levels only, not the total level. It is immutable and comparable with ==. The zero SorcererLevel is invalid; see
// IsZero. Safe for concurrent use.
type SorcererLevel struct{ n int }

// NewSorcererLevel returns the Sorcerer level n. It returns an error matching srderr.ErrInvalidInput for n below 1 or
// above 20. It never panics.
func NewSorcererLevel(n int) (SorcererLevel, error) {
	if n < 1 || n > MaxLevel {
		return SorcererLevel{}, invalidInput("Sorcerer level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return SorcererLevel{n: n}, nil
}

// MustSorcererLevel is NewSorcererLevel for literals known to be valid. It panics with the error NewSorcererLevel
// returns, and only then.
func MustSorcererLevel(n int) SorcererLevel {
	l, err := NewSorcererLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l SorcererLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l SorcererLevel) IsZero() bool { return l.n == 0 }

// SorceryPointsMaximum returns the Sorcery Points maximum at the level: none at level 1 and the level from level 2
// (SRD 5.2.1, "Sorcerer Features", p. 65, the Sorcery Points column).
func (l SorcererLevel) SorceryPointsMaximum() int {
	if l.n < sorcererFontOfMagicLevel {
		return 0
	}
	return l.n
}

// MetamagicEntitled returns the number of Metamagic options known at the level: 0 at level 1, 2 from level 2, 4 from
// level 10 and 6 from level 17 (SRD 5.2.1, "Level 2: Metamagic", p. 66).
func (l SorcererLevel) MetamagicEntitled() int {
	switch {
	case l.n >= 17:
		return 6
	case l.n >= 10:
		return 4
	case l.n >= sorcererFontOfMagicLevel:
		return 2
	}
	return 0
}

// SorcererProfile is the Sorcerer level and the chosen subclass a function needs, built by the caller from ClassLevels
// and the character's ChosenSubclass and never stored (class-sorcerer.md FR-137). The zero SorcererProfile is invalid.
// It is immutable and comparable with ==.
type SorcererProfile struct {
	level    SorcererLevel
	subclass content.SubclassID
}

// NewSorcererProfile returns the profile of a Sorcerer of level with subclass (empty for none). It returns an error
// matching srderr.ErrInvalidInput for the zero level or a non-empty invalid subclass id. It never panics.
func NewSorcererProfile(level SorcererLevel, subclass content.SubclassID) (SorcererProfile, error) {
	if level.IsZero() || (subclass != "" && !subclass.Valid()) {
		return SorcererProfile{}, invalidInput("Sorcerer profile with a zero level or an invalid subclass id")
	}
	return SorcererProfile{level: level, subclass: subclass}, nil
}

// SorcererProfileOf returns the profile of c, built from its ClassLevels and the ChosenSubclass held in the Sorcerer
// subclass state, and false for a character with no Sorcerer level. It returns an error matching
// srderr.ErrInvalidInput when another type of state sits under the subclass key.
func SorcererProfileOf(c Character) (SorcererProfile, bool, error) {
	n := c.ClassLevels().LevelIn(SorcererClassID)
	if n < 1 {
		return SorcererProfile{}, false, nil
	}
	key, err := SubclassStateKey{}.ForClass(SorcererClassID)
	if err != nil { // unreachable: the class id is valid
		return SorcererProfile{}, false, err
	}
	chosen, held, err := ReadFeatureState(c.FeatureStates(), key)
	if err != nil {
		return SorcererProfile{}, false, err
	}
	p := SorcererProfile{level: SorcererLevel{n: n}}
	if held {
		p.subclass = chosen.Subclass()
	}
	return p, true, nil
}

// Level returns the Sorcerer level.
func (p SorcererProfile) Level() SorcererLevel { return p.level }

// Subclass returns the chosen subclass, empty for none.
func (p SorcererProfile) Subclass() content.SubclassID { return p.subclass }

func (p SorcererProfile) draconicAt(level int) bool {
	return p.subclass == DraconicSorcerySubclassID && p.level.n >= level
}

// SorcererFeatureAvailability says which Sorcerer features a level has gained (class-sorcerer.md FR-019). It is a value
// of exported fields, built only by SorcererFeaturesAt; read it, do not construct it.
type SorcererFeatureAvailability struct {
	// SorceryPoints is the Sorcery Points maximum: 0 at level 1.
	SorceryPoints int
	// FontOfMagic and Metamagic arrive at level 2.
	FontOfMagic, Metamagic bool
	// SorcerousRestoration arrives at level 5.
	SorcerousRestoration bool
	// SorceryIncarnate arrives at level 7.
	SorceryIncarnate bool
	// ArcaneApotheosis arrives at level 20.
	ArcaneApotheosis bool
}

// SorcererFeaturesAt reports the features the profile's level has gained (class-sorcerer.md FR-019).
func SorcererFeaturesAt(p SorcererProfile) SorcererFeatureAvailability {
	n := p.level.n
	return SorcererFeatureAvailability{
		SorceryPoints: p.level.SorceryPointsMaximum(), FontOfMagic: n >= sorcererFontOfMagicLevel, Metamagic: n >= sorcererFontOfMagicLevel,
		SorcerousRestoration: n >= sorcererRestorationLevel, SorceryIncarnate: n >= sorcererIncarnateLevel, ArcaneApotheosis: n >= sorcererApotheosisLevel,
	}
}

// SubclassFeaturesAvailable returns the ids of the features of sub whose level is at most the profile's level, in
// printed order, and none while no subclass is chosen or when sub is not the chosen subclass (class-sorcerer.md
// FR-024; Draconic Sorcery: 3, 6, 14, 18). It returns a new slice.
func SubclassFeaturesAvailable(p SorcererProfile, sub content.SubclassDefinition) []content.FeatureID {
	out := []content.FeatureID{}
	if p.subclass == "" || sub.ID() != p.subclass {
		return out
	}
	entries, err := sub.FeaturesAtOrBelow(p.level.n)
	if err != nil { // unreachable: a valid profile level is within 1 to 20
		return out
	}
	for _, e := range entries {
		out = append(out, e.ID())
	}
	return out
}
