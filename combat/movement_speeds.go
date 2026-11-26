package combat

import "github.com/Whealinghold/srd5e-go/content"

// SpeedName names one of a creature's speeds (SRD 5.2.1, "Speed", p. 188). The
// zero SpeedName is invalid.
type SpeedName int

// The five speeds.
const (
	SpeedNameWalk SpeedName = iota + 1
	SpeedNameBurrow
	SpeedNameClimb
	SpeedNameFly
	SpeedNameSwim
)

func speedNames() []string { return []string{"walk", "burrow", "climb", "fly", "swim"} }

// Valid reports whether n is one of the five speeds.
func (n SpeedName) Valid() bool { return n >= SpeedNameWalk && n <= SpeedNameSwim }

// String returns "walk", "burrow", "climb", "fly", "swim" or "unknown".
func (n SpeedName) String() string { return enumName(speedNames(), int(n)) }

// MovementSpeedEntry is one special speed given to NewMovementSpeeds.
type MovementSpeedEntry struct {
	Name SpeedName
	Feet int
}

// MovementSpeeds are the speeds a creature has on a turn, in feet: the walking
// Speed and any of the Burrow, Climb, Fly and Swim speeds (SRD 5.2.1, "Speed",
// p. 188). A special speed may be 0 and still be had. The zero value is a creature
// with a walking Speed of 0 and no special speed. Comparable with ==; immutable.
// Safe for concurrent use.
type MovementSpeeds struct {
	feet [5]int
	has  [5]bool
}

// NewMovementSpeeds returns the speeds. It returns an error matching
// srderr.ErrInvalidInput for a negative walking Speed or special speed, a special
// speed named walk or not a speed, or one named twice. It never panics.
func NewMovementSpeeds(walk int, specials ...MovementSpeedEntry) (MovementSpeeds, error) {
	if walk < 0 {
		return MovementSpeeds{}, invalidInput("negative walking Speed")
	}
	s := MovementSpeeds{}
	s.feet[0] = walk
	for _, e := range specials {
		if !e.Name.Valid() || e.Name == SpeedNameWalk || e.Feet < 0 || s.has[e.Name-1] {
			return MovementSpeeds{}, invalidInput("special speed must be a new burrow, climb, fly or swim speed of 0 or more")
		}
		s.feet[e.Name-1], s.has[e.Name-1] = e.Feet, true
	}
	return s, nil
}

// Walk returns the walking Speed.
func (s MovementSpeeds) Walk() int { return s.feet[0] }

// Of returns the speed named n and whether the creature has it. The walking
// Speed is always had; a name that is not a speed is not had.
func (s MovementSpeeds) Of(n SpeedName) (int, bool) {
	if !n.Valid() {
		return 0, false
	}
	return s.feet[n-1], n == SpeedNameWalk || s.has[n-1]
}

// mapped applies f to the walking Speed and to each special speed the creature has.
func (s MovementSpeeds) mapped(f func(int) int) MovementSpeeds {
	out := s
	for i := range out.feet {
		if i == 0 || out.has[i] {
			out.feet[i] = max(0, f(out.feet[i]))
		}
	}
	return out
}

// AdjustedBy adds feet to every speed the creature has; none falls below 0 and
// the sums saturate (SRD 5.2.1, "Changes to Your Speeds", p. 188).
func (s MovementSpeeds) AdjustedBy(feet int) MovementSpeeds {
	return s.mapped(func(v int) int { return satAdd(v, feet) })
}

// Halved halves every speed the creature has, each rounded down on its own
// (SRD 5.2.1, "Changes to Your Speeds", p. 188; "Round Down", p. 187).
func (s MovementSpeeds) Halved() MovementSpeeds { return s.mapped(func(v int) int { return v / 2 }) }

// Doubled doubles every speed the creature has, each on its own and saturating.
// Doubling every speed is an interpretation of "Changes to Your Speeds", p. 188.
func (s MovementSpeeds) Doubled() MovementSpeeds {
	return s.mapped(func(v int) int { return satAdd(v, v) })
}

// CombatantProfile is what the tracker knows of one combatant: its size and its
// speeds. Build it with a keyed literal; EstablishInitiativeOrder's command
// constructor validates it. It is comparable with ==.
type CombatantProfile struct {
	ID     CombatantID
	Size   content.CreatureSize
	Speeds MovementSpeeds
}
