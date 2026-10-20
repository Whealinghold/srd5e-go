package content

// SentientCommunication is how a sentient item communicates (SRD 5.2.1,
// "Sentient Item's Communication", p. 208), a closed set of three members. The
// zero SentientCommunication is invalid. Comparable and safe for concurrent
// use.
type SentientCommunication int

// The sentient communications.
const (
	SentientTransmitsEmotion SentientCommunication = iota + 1
	SentientSpeaksLanguages
	SentientSpeaksLanguagesAndTelepathy
)

func sentientCommunicationNames() []string {
	return []string{"transmitsEmotion", "speaksLanguages", "speaksLanguagesAndTelepathy"}
}

// Valid reports whether c is one of the three communications.
func (c SentientCommunication) Valid() bool { return enumValid(sentientCommunicationNames(), int(c)) }

// String returns the name of c, or "unknown" for an undefined value.
func (c SentientCommunication) String() string {
	return enumString(sentientCommunicationNames(), int(c))
}

// SentientCommunicationForRoll maps a d10 to a communication: 1 to 6
// [SentientTransmitsEmotion], 7 to 9 [SentientSpeaksLanguages] and 10
// [SentientSpeaksLanguagesAndTelepathy].
//
// It returns a *[RollRangeError] wrapping srderr.ErrInvalidInput for a roll
// below 1 or above 10. It never panics.
func SentientCommunicationForRoll(roll int) (SentientCommunication, error) {
	if err := checkRoll(roll, 10); err != nil {
		return 0, err
	}
	switch {
	case roll <= 6:
		return SentientTransmitsEmotion, nil
	case roll <= 9:
		return SentientSpeaksLanguages, nil
	}
	return SentientSpeaksLanguagesAndTelepathy, nil
}

// SentientVision is the vision of a sentient item, a closed family with two
// variants: [StandardVisionOnly] and [StandardVisionAndSense]. Consumers switch
// over it exhaustively. Every variant is an immutable, comparable value and
// safe for concurrent use.
type SentientVision interface{ isSentientVision() }

// StandardVisionOnly is standard vision and nothing more. The zero value is
// valid.
type StandardVisionOnly struct{}

func (StandardVisionOnly) isSentientVision() {}

// StandardVisionAndSense is standard vision and one special sense. The zero
// value is invalid; build it with [NewStandardVisionAndSense].
type StandardVisionAndSense struct{ sense SpecialSense }

// NewStandardVisionAndSense returns standard vision plus sense.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for a sense that is not a member. It never panics.
func NewStandardVisionAndSense(sense SpecialSense) (StandardVisionAndSense, error) {
	if !sense.Valid() {
		return StandardVisionAndSense{}, blockError("StandardVisionAndSense", BlockInvalidMember)
	}
	return StandardVisionAndSense{sense: sense}, nil
}

// Sense returns the special sense.
func (v StandardVisionAndSense) Sense() SpecialSense { return v.sense }

func (StandardVisionAndSense) isSentientVision() {}

// SentientSenses are the senses of a sentient item: hearing and vision out to
// a range in feet (SRD 5.2.1, "Sentient Item's Senses", p. 208). The zero value
// is invalid; build it with [NewSentientSenses] or [SentientSensesForRoll].
// Comparable and safe for concurrent use.
type SentientSenses struct {
	rangeFeet int
	vision    SentientVision
}

// NewSentientSenses returns hearing and vision out to rangeFeet.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a range of 0 or less and BlockMissing for a nil vision.
// It never panics.
func NewSentientSenses(rangeFeet int, vision SentientVision) (SentientSenses, error) {
	if rangeFeet <= 0 {
		return SentientSenses{}, blockError("SentientSenses", BlockOutOfRange)
	}
	if vision == nil {
		return SentientSenses{}, blockError("SentientSenses", BlockMissing)
	}
	return SentientSenses{rangeFeet: rangeFeet, vision: vision}, nil
}

// RangeFeet returns the range of the senses in feet.
func (s SentientSenses) RangeFeet() int { return s.rangeFeet }

// Vision returns the vision of the item.
func (s SentientSenses) Vision() SentientVision { return s.vision }

// SentientSensesForRoll maps a d4 to senses: 1 hearing and standard vision out
// to 30 feet, 2 to 60 feet, 3 to 120 feet, and 4 hearing and standard vision
// plus Darkvision out to 120 feet.
//
// It returns a *[RollRangeError] wrapping srderr.ErrInvalidInput for a roll
// below 1 or above 4. It never panics.
func SentientSensesForRoll(roll int) (SentientSenses, error) {
	if err := checkRoll(roll, 4); err != nil {
		return SentientSenses{}, err
	}
	switch roll {
	case 1:
		return SentientSenses{rangeFeet: 30, vision: StandardVisionOnly{}}, nil
	case 2:
		return SentientSenses{rangeFeet: 60, vision: StandardVisionOnly{}}, nil
	case 3:
		return SentientSenses{rangeFeet: 120, vision: StandardVisionOnly{}}, nil
	}
	return SentientSenses{rangeFeet: 120, vision: StandardVisionAndSense{sense: SenseDarkvision}}, nil
}

// SpecialPurposeKind is the kind of a sentient item's special purpose, in
// table order (SRD 5.2.1, "Sentient Item's Special Purpose", p. 208). It
// carries no payload. The zero SpecialPurposeKind is invalid. Comparable and
// safe for concurrent use.
type SpecialPurposeKind int

// The special purpose kinds.
const (
	SpecialPurposeAligned SpecialPurposeKind = iota + 1
	SpecialPurposeBane
	SpecialPurposeCreatorSeeker
	SpecialPurposeDestinySeeker
	SpecialPurposeDestroyer
	SpecialPurposeGlorySeeker
	SpecialPurposeLoreSeeker
	SpecialPurposeProtector
	SpecialPurposeSoulmateSeeker
	SpecialPurposeTemplar
)

func specialPurposeKindNames() []string {
	return []string{
		"aligned", "bane", "creatorSeeker", "destinySeeker", "destroyer",
		"glorySeeker", "loreSeeker", "protector", "soulmateSeeker", "templar",
	}
}

// Valid reports whether k is one of the ten kinds.
func (k SpecialPurposeKind) Valid() bool { return enumValid(specialPurposeKindNames(), int(k)) }

// String returns the name of k, or "unknown" for an undefined value.
func (k SpecialPurposeKind) String() string { return enumString(specialPurposeKindNames(), int(k)) }

// SpecialPurposeKindForRoll maps a d10 to a kind in table order, 1 being
// [SpecialPurposeAligned] and 10 [SpecialPurposeTemplar].
//
// It returns a *[RollRangeError] wrapping srderr.ErrInvalidInput for a roll
// below 1 or above 10. It never panics.
func SpecialPurposeKindForRoll(roll int) (SpecialPurposeKind, error) {
	if err := checkRoll(roll, 10); err != nil {
		return 0, err
	}
	return SpecialPurposeKind(roll), nil
}

// SentientAlignmentTableForRoll maps a d100 (100 is the table's "00") to an
// alignment (SRD 5.2.1, "Sentient Item's Alignment", p. 208): 1 to 15 Lawful
// Good, 16 to 35 Neutral Good, 36 to 50 Chaotic Good, 51 to 63 Lawful Neutral,
// 64 to 73 Neutral, 74 to 85 Chaotic Neutral, 86 to 89 Lawful Evil, 90 to 96
// Neutral Evil and 97 to 100 Chaotic Evil. It never returns
// [AlignmentUnaligned].
//
// It returns a *[RollRangeError] wrapping srderr.ErrInvalidInput for a roll
// below 1 or above 100. It never panics.
func SentientAlignmentTableForRoll(roll int) (Alignment, error) {
	if err := checkRoll(roll, 100); err != nil {
		return 0, err
	}
	switch {
	case roll <= 15:
		return AlignmentLawfulGood, nil
	case roll <= 35:
		return AlignmentNeutralGood, nil
	case roll <= 50:
		return AlignmentChaoticGood, nil
	case roll <= 63:
		return AlignmentLawfulNeutral, nil
	case roll <= 73:
		return AlignmentNeutral, nil
	case roll <= 85:
		return AlignmentChaoticNeutral, nil
	case roll <= 89:
		return AlignmentLawfulEvil, nil
	case roll <= 96:
		return AlignmentNeutralEvil, nil
	}
	return AlignmentChaoticEvil, nil
}

// SentientDemand is a demand a sentient item makes in a conflict (SRD 5.2.1,
// "Conflict", p. 208), a closed set of four members. The zero SentientDemand is
// invalid. Comparable and safe for concurrent use.
type SentientDemand int

// The sentient demands.
const (
	SentientDemandChaseMyDreams SentientDemand = iota + 1
	SentientDemandGetRidOfIt
	SentientDemandItsTimeForAChange
	SentientDemandKeepMeClose
)

func sentientDemandNames() []string {
	return []string{"Chase My Dreams", "Get Rid of It", "It's Time for a Change", "Keep Me Close"}
}

// Valid reports whether d is one of the four demands.
func (d SentientDemand) Valid() bool { return enumValid(sentientDemandNames(), int(d)) }

// String returns the printed name of d, or "unknown" for an undefined value.
func (d SentientDemand) String() string { return enumString(sentientDemandNames(), int(d)) }

// SentientRefusalPower is a power a sentient item uses to refuse its bearer
// (SRD 5.2.1, "Conflict", p. 208), a closed set of three members. The zero
// SentientRefusalPower is invalid. Comparable and safe for concurrent use.
type SentientRefusalPower int

// The sentient refusal powers.
const (
	SentientRefusalPreventAttunement SentientRefusalPower = iota + 1
	SentientRefusalSuppressProperties
	SentientRefusalAttemptControl
)

func sentientRefusalPowerNames() []string {
	return []string{"preventAttunement", "suppressProperties", "attemptControl"}
}

// Valid reports whether p is one of the three powers.
func (p SentientRefusalPower) Valid() bool { return enumValid(sentientRefusalPowerNames(), int(p)) }

// String returns the name of p, or "unknown" for an undefined value.
func (p SentientRefusalPower) String() string {
	return enumString(sentientRefusalPowerNames(), int(p))
}
