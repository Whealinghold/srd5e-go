package character

// Attitude is a monster's starting attitude toward a player character: Friendly,
// Indifferent or Hostile, in that order (SRD 5.2.1, "Attitude", p. 177; "Social
// Interaction", p. 10). It is a static fact and carries no behaviour; the
// Advantage or Disadvantage an attitude contributes is AttitudeCircumstances.
// The zero Attitude is invalid; see Valid. Comparable and safe for concurrent use.
type Attitude int

// The attitudes.
const (
	// Friendly gives Advantage on an Influence check (p. 182).
	Friendly Attitude = iota + 1
	// Indifferent is "the default attitude of a monster" (p. 184).
	Indifferent
	// Hostile gives Disadvantage on an Influence check (p. 183).
	Hostile
)

// MonsterDefaultAttitude is the attitude of a monster whose attitude is not
// stated: Indifferent (SRD 5.2.1, "Indifferent [Attitude]", p. 184).
const MonsterDefaultAttitude = Indifferent

// Valid reports whether a is Friendly, Indifferent or Hostile.
func (a Attitude) Valid() bool { return a >= Friendly && a <= Hostile }

// String returns "friendly", "indifferent" or "hostile", the value of the
// srd5e.influence.attitude attribute, or "unknown" for an undefined value.
func (a Attitude) String() string {
	switch a {
	case Friendly:
		return "friendly"
	case Indifferent:
		return "indifferent"
	case Hostile:
		return "hostile"
	}
	return "unknown"
}
