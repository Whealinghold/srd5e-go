package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// LongRestUse is whether a once-per-Long-Rest use can be made. The zero value is LongRestUseAvailable, so a use of a feature not
// yet gained is available (class-paladin.md section 5). Comparable.
type LongRestUse int

const (
	// LongRestUseAvailable: the use can be made.
	LongRestUseAvailable LongRestUse = iota
	// LongRestUseSpent: the use was made; a finished Long Rest makes it available again.
	LongRestUseSpent
)

// String returns "available" or "spent" ("unknown" for another value).
func (u LongRestUse) String() string {
	switch u {
	case LongRestUseAvailable:
		return "available"
	case LongRestUseSpent:
		return "spent"
	}
	return "unknown"
}

// FreeCastFeature is a Paladin feature that casts a spell once without a spell slot. The zero value is invalid.
type FreeCastFeature int

const (
	// FreeCastPaladinsSmite casts Divine Smite (SRD 5.2.1, "Level 2: Paladin’s Smite", p. 54).
	FreeCastPaladinsSmite FreeCastFeature = iota + 1
	// FreeCastFaithfulSteed casts Find Steed (SRD 5.2.1, "Level 5: Faithful Steed", p. 55).
	FreeCastFaithfulSteed
)

// Valid reports whether f is one of the two free casts.
func (f FreeCastFeature) Valid() bool {
	return f == FreeCastPaladinsSmite || f == FreeCastFaithfulSteed
}

// String returns the srd5e.feature value, "paladins_smite" or "faithful_steed" ("unknown" for another value).
func (f FreeCastFeature) String() string {
	switch f {
	case FreeCastPaladinsSmite:
		return "paladins_smite"
	case FreeCastFaithfulSteed:
		return "faithful_steed"
	}
	return "unknown"
}

func (f FreeCastFeature) paladinFeature() PaladinFeature {
	if f == FreeCastFaithfulSteed {
		return PaladinFeatureFaithfulSteed
	}
	return PaladinFeaturePaladinsSmite
}

// ClassEntry is how a character enters a class: as its first class or by multiclassing (SRD 5.2.1, "Becoming a Paladin ...",
// p. 53). The zero value is invalid.
type ClassEntry int

const (
	// ClassEntryFirstLevel: the class is the character's first.
	ClassEntryFirstLevel ClassEntry = iota + 1
	// ClassEntryMulticlass: the class is added to a character that has another.
	ClassEntryMulticlass
)

// Valid reports whether e is one of the two entries.
func (e ClassEntry) Valid() bool { return e == ClassEntryFirstLevel || e == ClassEntryMulticlass }

// String returns "first_level" or "multiclass" ("unknown" for another value).
func (e ClassEntry) String() string {
	switch e {
	case ClassEntryFirstLevel:
		return "first_level"
	case ClassEntryMulticlass:
		return "multiclass"
	}
	return "unknown"
}

// PaladinFightingStyle is the Fighting Style a Paladin chose, a closed set. Variants: PaladinFightingStyleFeat, BlessedWarrior.
type PaladinFightingStyle interface{ isPaladinFightingStyle() }

// PaladinFightingStyleFeat is a Fighting Style feat taken as the Fighting Style. Construct it with a keyed literal.
type PaladinFightingStyleFeat struct{ Feat content.FeatID }

// BlessedWarrior is the Blessed Warrior option: two Cleric cantrips that count as Paladin spells (SRD 5.2.1, "Level 2: Fighting
// Style", p. 54). Construct it with a keyed literal.
type BlessedWarrior struct{ Cantrips [2]content.SpellID }

func (PaladinFightingStyleFeat) isPaladinFightingStyle() {}
func (BlessedWarrior) isPaladinFightingStyle()           {}

// PaladinActionCost is the action a Paladin feature costs. The zero value is invalid.
type PaladinActionCost int

const (
	// PaladinBonusAction: Lay On Hands, Divine Sense, Holy Nimbus.
	PaladinBonusAction PaladinActionCost = iota + 1
	// PaladinMagicAction: Abjure Foes.
	PaladinMagicAction
	// PaladinNoAction: Sacred Weapon (part of the Attack action), ending an effect, or restoring Holy Nimbus with a spell slot.
	PaladinNoAction
)

// String returns "bonus_action", "magic_action", "no_action" or "unknown".
func (a PaladinActionCost) String() string {
	switch a {
	case PaladinBonusAction:
		return "bonus_action"
	case PaladinMagicAction:
		return "magic_action"
	case PaladinNoAction:
		return "no_action"
	}
	return "unknown"
}

// PaladinState is a Paladin's resources and choices: the Lay On Hands pool, the Channel Divinity uses, the three
// once-per-Long-Rest uses, the Fighting Style and the active effects. The maximums follow the Paladin level and the
// definition, so they are not stored; a stored amount above the maximum is read as the maximum. It is a FeatureState stored
// under PaladinStateKey; an absent state is InitialPaladinState. Immutable, comparable with ==. Safe for concurrent use.
type PaladinState struct {
	layRemaining, channelRemaining int
	smite, steed, nimbus           LongRestUse
	style                          PaladinFightingStyle
	divineSense, holyNimbus        bool
	sacredWeapon                   string
}

// PaladinStateKey returns the FeatureStateKey named paladin_state.
func PaladinStateKey() FeatureStateKey[PaladinState] {
	return MustFeatureStateKey[PaladinState]("paladin_state")
}

// PaladinStateParts are the fields of a PaladinState, for RestorePaladinState. Construct it with a keyed literal.
type PaladinStateParts struct {
	LayOnHandsRemaining      int
	ChannelDivinityRemaining int
	PaladinsSmite            LongRestUse
	FaithfulSteed            LongRestUse
	HolyNimbus               LongRestUse
	FightingStyle            PaladinFightingStyle
	DivineSenseActive        bool
	SacredWeaponActive       string // the held weapon's host key, "" for none
	HolyNimbusActive         bool
}

// LayOnHandsMaximum returns the Lay On Hands pool maximum, five times the Paladin level (SRD 5.2.1, "Level 1: Lay On Hands",
// p. 53).
func LayOnHandsMaximum(level PaladinLevel) int { return 5 * level.n }

// InitialPaladinState returns the state of a Paladin of level with every pool full and every use available. channelMaximum is
// the Channel Divinity maximum at that level.
func InitialPaladinState(level PaladinLevel, channelMaximum int) PaladinState {
	return PaladinState{layRemaining: LayOnHandsMaximum(level), channelRemaining: max(channelMaximum, 0)}
}

// RestorePaladinState returns the state of parts for a Paladin of level whose Channel Divinity maximum is channelMaximum. It
// returns an error matching srderr.ErrInvalidInput for the zero level, a Lay On Hands amount outside 0 to the maximum
// (class-paladin.md FR-045), a Channel Divinity amount outside 0 to channelMaximum, a use that is not a LongRestUse, a Fighting
// Style that is not one of the closed set or an invalid Blessed Warrior cantrip.
// It never panics.
func RestorePaladinState(level PaladinLevel, channelMaximum int, parts PaladinStateParts) (PaladinState, error) {
	switch {
	case level.IsZero():
		return PaladinState{}, invalidInput("Paladin state of the zero level")
	case parts.LayOnHandsRemaining < 0 || parts.LayOnHandsRemaining > LayOnHandsMaximum(level):
		return PaladinState{}, invalidInput("Lay On Hands " + strconv.Itoa(parts.LayOnHandsRemaining) + " outside 0 to " + strconv.Itoa(LayOnHandsMaximum(level)))
	case parts.ChannelDivinityRemaining < 0 || parts.ChannelDivinityRemaining > channelMaximum:
		return PaladinState{}, invalidInput("Channel Divinity " + strconv.Itoa(parts.ChannelDivinityRemaining) + " outside 0 to " + strconv.Itoa(channelMaximum))
	case !validUse(parts.PaladinsSmite) || !validUse(parts.FaithfulSteed) || !validUse(parts.HolyNimbus):
		return PaladinState{}, invalidInput("Paladin once-per-Long-Rest use is neither available nor spent")
	}
	switch s := parts.FightingStyle.(type) {
	case nil:
	case PaladinFightingStyleFeat:
		if !s.Feat.Valid() {
			return PaladinState{}, invalidInput("Paladin Fighting Style feat id")
		}
	case BlessedWarrior:
		if !s.Cantrips[0].Valid() || !s.Cantrips[1].Valid() || s.Cantrips[0] == s.Cantrips[1] {
			return PaladinState{}, invalidInput("Blessed Warrior cantrips")
		}
	default:
		return PaladinState{}, invalidInput("Paladin Fighting Style is not one of the closed set")
	}
	return PaladinState{layRemaining: parts.LayOnHandsRemaining, channelRemaining: parts.ChannelDivinityRemaining,
		smite: parts.PaladinsSmite, steed: parts.FaithfulSteed, nimbus: parts.HolyNimbus, style: parts.FightingStyle,
		divineSense: parts.DivineSenseActive, sacredWeapon: parts.SacredWeaponActive, holyNimbus: parts.HolyNimbusActive}, nil
}

func validUse(u LongRestUse) bool { return u == LongRestUseAvailable || u == LongRestUseSpent }

// LayOnHandsRemaining returns the Hit Points left in the Lay On Hands pool.
func (s PaladinState) LayOnHandsRemaining() int { return s.layRemaining }

// ChannelDivinityRemaining returns the unexpended Channel Divinity uses.
func (s PaladinState) ChannelDivinityRemaining() int { return s.channelRemaining }

// FreeCast returns the use of the free cast; available for an invalid feature.
func (s PaladinState) FreeCast(f FreeCastFeature) LongRestUse {
	switch f {
	case FreeCastPaladinsSmite:
		return s.smite
	case FreeCastFaithfulSteed:
		return s.steed
	}
	return LongRestUseAvailable
}

// HolyNimbusUse returns the once-per-Long-Rest use of Holy Nimbus.
func (s PaladinState) HolyNimbusUse() LongRestUse { return s.nimbus }

// FightingStyle returns the chosen Fighting Style and false when none is chosen.
func (s PaladinState) FightingStyle() (PaladinFightingStyle, bool) { return s.style, s.style != nil }

// DivineSenseActive reports whether Divine Sense is active.
func (s PaladinState) DivineSenseActive() bool { return s.divineSense }

// SacredWeaponActive returns the host key of the imbued weapon and false when Sacred Weapon is not active.
func (s PaladinState) SacredWeaponActive() (string, bool) {
	return s.sacredWeapon, s.sacredWeapon != ""
}

// HolyNimbusActive reports whether the Holy Nimbus effect is active.
func (s PaladinState) HolyNimbusActive() bool { return s.holyNimbus }

// BlessedWarriorCantrips returns the Blessed Warrior cantrips as GrantedCantrips for spellcasting: each uses the Paladin's
// ability and the Cleric list, and does not count against the prepared limit (class-paladin.md FR-040). It is empty without
// Blessed Warrior.
func (s PaladinState) BlessedWarriorCantrips() []GrantedCantrip {
	bw, ok := s.style.(BlessedWarrior)
	if !ok {
		return []GrantedCantrip{}
	}
	out := make([]GrantedCantrip, 0, 2)
	for _, id := range bw.Cantrips {
		g, err := NewGrantedCantrip(id, CasterPaladin, "cleric")
		if err == nil { // the state holds valid ids
			out = append(out, g)
		}
	}
	return out
}

// Equal reports whether other is a PaladinState with the same fields.
func (s PaladinState) Equal(other FeatureState) bool {
	o, ok := other.(PaladinState)
	return ok && o == s
}

// clamp returns s with the amounts held within their maximums.
func (s PaladinState) clamp(level PaladinLevel, channelMaximum int) PaladinState {
	s.layRemaining = min(max(s.layRemaining, 0), LayOnHandsMaximum(level))
	s.channelRemaining = min(max(s.channelRemaining, 0), max(channelMaximum, 0))
	return s
}

// readPaladinState returns the PaladinState c stores read within the maximums of level, or InitialPaladinState when none is
// stored. It returns an error matching srderr.ErrInvalidInput when another type of state sits under the key.
func readPaladinState(c Character, level PaladinLevel, channelMaximum int) (PaladinState, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), PaladinStateKey())
	if err != nil {
		return PaladinState{}, err
	}
	if !ok {
		return InitialPaladinState(level, channelMaximum), nil
	}
	return s.clamp(level, channelMaximum), nil
}
