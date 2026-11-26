package combat

// MountControl says whether the rider controls the mount or it acts on its own (SRD
// 5.2.1, "Controlling a Mount", p. 16). The zero value is invalid.
type MountControl int

// The two kinds of ride.
const (
	// MountControlControlled: the mount's Initiative matches the rider's and it moves on the rider's turn.
	MountControlControlled MountControl = iota + 1
	// MountControlIndependent: the mount retains its place in the Initiative order and acts as it wishes.
	MountControlIndependent
)

// Valid reports whether c is one of the two controls.
func (c MountControl) Valid() bool {
	return c == MountControlControlled || c == MountControlIndependent
}

// String returns "controlled", "independent" or "unknown".
func (c MountControl) String() string { return enumName([]string{"controlled", "independent"}, int(c)) }

// RiderStatus says whether a rider is mounted and on what (SRD 5.2.1, "Mounted
// Combat", p. 15). The host or the encounter holds it. The zero value is NotMounted
// and valid; MountedOn builds the mounted one, and only MountCreature creates a
// mounted status in the course of a fight. It is comparable and immutable. Safe for
// concurrent use.
type RiderStatus struct {
	mount   CombatantID
	control MountControl
}

// NotMounted returns the status of a rider on no mount; it equals the zero RiderStatus.
func NotMounted() RiderStatus { return RiderStatus{} }

// MountedOn returns the status of a rider on mount with the control of the ride. It
// returns an error matching srderr.ErrInvalidInput for a blank mount or an invalid
// control. It never panics.
func MountedOn(mount CombatantID, control MountControl) (RiderStatus, error) {
	if mount.IsZero() || !control.Valid() {
		return RiderStatus{}, invalidInput("mounted status needs a mount and a control")
	}
	return RiderStatus{mount, control}, nil
}

// IsMounted reports whether the rider is on a mount.
func (s RiderStatus) IsMounted() bool { return !s.mount.IsZero() }

// IsOn reports whether the rider is on mount.
func (s RiderStatus) IsOn(mount CombatantID) bool { return s.IsMounted() && s.mount == mount }

// Control returns the control of the ride; ok is false for a rider who is not mounted.
func (s RiderStatus) Control() (control MountControl, ok bool) { return s.control, s.IsMounted() }

// MountInitiativeChange is what mounting does to the mount's Initiative (SRD 5.2.1,
// "Controlling a Mount", p. 16). It is data only: the host applies it. Variants:
// MountInitiativeMatchesRider, MountInitiativeUnchanged.
type MountInitiativeChange interface{ isMountInitiativeChange() }

// MountInitiativeMatchesRider: the controlled mount's Initiative changes to match the
// rider's. Construct with a keyed literal.
type MountInitiativeMatchesRider struct{ Rider CombatantID }

// MountInitiativeUnchanged: the independent mount retains its place in the Initiative order.
type MountInitiativeUnchanged struct{}

func (MountInitiativeMatchesRider) isMountInitiativeChange() {}
func (MountInitiativeUnchanged) isMountInitiativeChange()    {}

// MountTurn says on whose turn the mount moves (SRD 5.2.1, "Controlling a Mount", p.
// 16). The zero value is invalid.
type MountTurn int

// The two turns.
const (
	// MountTurnOnRidersTurn: the controlled mount moves on its rider's turn, as the rider directs.
	MountTurnOnRidersTurn MountTurn = iota + 1
	// MountTurnOnOwnTurn: the independent mount keeps its own turn.
	MountTurnOnOwnTurn
)

// String returns "on_riders_turn", "on_own_turn" or "unknown".
func (t MountTurn) String() string {
	return enumName([]string{"on_riders_turn", "on_own_turn"}, int(t))
}

// MountWillingness is whether the creature agrees to carry a rider (SRD 5.2.1, "Mounted
// Combat", p. 15: "A willing creature"). The zero value is invalid.
type MountWillingness int

// The two answers.
const (
	// MountWillingnessWilling: the creature agrees.
	MountWillingnessWilling MountWillingness = iota + 1
	// MountWillingnessUnwilling: the creature does not agree.
	MountWillingnessUnwilling
)

// MountAnatomy is whether the creature has an anatomy that can carry a rider; the SRD
// gives no test, so the host judges. The zero value is invalid.
type MountAnatomy int

// The two judgements.
const (
	// MountAnatomyAppropriate: the creature can serve as a mount.
	MountAnatomyAppropriate MountAnatomy = iota + 1
	// MountAnatomyNotAppropriate: it cannot.
	MountAnatomyNotAppropriate
)

// MountTraining is whether the mount has been trained to accept a rider (SRD 5.2.1,
// "Controlling a Mount", p. 16). The zero value is invalid.
type MountTraining int

// The two states.
const (
	// MountTrainingTrainedToAcceptRider: the mount can be controlled.
	MountTrainingTrainedToAcceptRider MountTraining = iota + 1
	// MountTrainingNotTrained: the mount can only act independently.
	MountTrainingNotTrained
)

// MountHabitat says whether the mount is aquatic or flying, which needs an Exotic
// Saddle (SRD 5.2.1, "Saddles", p. 100). The zero value is invalid.
type MountHabitat int

// The two habitats.
const (
	// MountHabitatOther: any mount that is neither aquatic nor flying.
	MountHabitatOther MountHabitat = iota + 1
	// MountHabitatAquaticOrFlying: an aquatic or a flying mount.
	MountHabitatAquaticOrFlying
)

// ExoticSaddle is whether an Exotic Saddle is fitted to the mount. The zero value is invalid.
type ExoticSaddle int

// The two states.
const (
	// ExoticSaddleFitted: an Exotic Saddle is on the mount.
	ExoticSaddleFitted ExoticSaddle = iota + 1
	// ExoticSaddleNotFitted: it is not.
	ExoticSaddleNotFitted
)

// MountFacts are the facts about a candidate mount that the host reports, as the
// engine models no map and no stat block (SRD 5.2.1, "Mounted Combat", p. 15).
// Build it with NewMountFacts; the zero value is invalid. It declares no field about
// the mount's own turn or resources (a controlled mount can act on the turn it is
// mounted, p. 16). It is comparable and immutable. Safe for concurrent use.
type MountFacts struct {
	willingness MountWillingness
	anatomy     MountAnatomy
	training    MountTraining
	habitat     MountHabitat
	saddle      ExoticSaddle
}

// NewMountFacts returns the facts. It returns an error matching srderr.ErrInvalidInput
// when any of the five values is not one of its constants. It never panics.
func NewMountFacts(w MountWillingness, a MountAnatomy, t MountTraining, h MountHabitat, s ExoticSaddle) (MountFacts, error) {
	if w < MountWillingnessWilling || w > MountWillingnessUnwilling || a < MountAnatomyAppropriate || a > MountAnatomyNotAppropriate ||
		t < MountTrainingTrainedToAcceptRider || t > MountTrainingNotTrained || h < MountHabitatOther || h > MountHabitatAquaticOrFlying ||
		s < ExoticSaddleFitted || s > ExoticSaddleNotFitted {
		return MountFacts{}, invalidInput("mount facts need five known values")
	}
	return MountFacts{w, a, t, h, s}, nil
}

// valid reports whether the facts came from NewMountFacts.
func (f MountFacts) valid() bool { return f.willingness != 0 }

// RiderTurnState says whether it is the rider's turn (SRD 5.2.1, "Mounting and
// Dismounting", p. 15: "during your move"). The zero value is invalid.
type RiderTurnState int

// The two states.
const (
	// RiderTurnActive: it is the rider's turn.
	RiderTurnActive RiderTurnState = iota + 1
	// RiderTurnNotActive: it is not.
	RiderTurnNotActive
)

// MountingTurnFacts are the three facts about the rider's turn the caller reads from
// its tracker: whose turn it is, the walking Speed and the walking movement left
// (Speed plus any Dash extra, minus feet moved). Build it with NewMountingTurnFacts;
// the zero value is invalid. It declares no condition or action field: mounting is
// movement, so Incapacitated does not change it. Comparable and immutable. Safe for
// concurrent use.
type MountingTurnFacts struct {
	riderTurn               RiderTurnState
	walkingSpeedFeet        int
	walkingMovementLeftFeet int
}

// NewMountingTurnFacts returns the facts. The movement left may be 0 or less (a Speed
// lowered after moving). It returns an error matching srderr.ErrInvalidInput for an
// unknown turn state or a negative walking Speed. It never panics.
func NewMountingTurnFacts(turn RiderTurnState, walkingSpeedFeet, walkingMovementLeftFeet int) (MountingTurnFacts, error) {
	if turn < RiderTurnActive || turn > RiderTurnNotActive || walkingSpeedFeet < 0 {
		return MountingTurnFacts{}, invalidInput("turn facts need a known turn state and a Speed of 0 or more")
	}
	return MountingTurnFacts{turn, walkingSpeedFeet, walkingMovementLeftFeet}, nil
}

// valid reports whether the facts came from NewMountingTurnFacts.
func (f MountingTurnFacts) valid() bool { return f.riderTurn != 0 }
