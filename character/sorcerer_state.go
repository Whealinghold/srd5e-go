package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// MetamagicEffect is one of the ten Metamagic options of SRD 5.2.1 ("Metamagic Options", pp. 66-67), a closed set the
// code interprets: content may change a cost but never the effect (class-sorcerer.md section 6). The zero value is not an effect.
type MetamagicEffect int

// The Metamagic effects, in the printed alphabetical order.
const (
	// MetamagicCareful protects creatures from the full force of a saving throw spell.
	MetamagicCareful MetamagicEffect = iota + 1
	// MetamagicDistant doubles the range of a spell, or gives Touch a range of 30 feet.
	MetamagicDistant
	// MetamagicEmpowered rerolls some of the damage dice of a spell.
	MetamagicEmpowered
	// MetamagicExtended doubles a duration of 1 minute or longer, to 24 hours.
	MetamagicExtended
	// MetamagicHeightened gives one target Disadvantage on its saves against the spell.
	MetamagicHeightened
	// MetamagicQuickened changes a casting time of an action to a Bonus Action.
	MetamagicQuickened
	// MetamagicSeeking rerolls the d20 of a missed spell attack roll.
	MetamagicSeeking
	// MetamagicSubtle removes the components that can be removed.
	MetamagicSubtle
	// MetamagicTransmuted changes a damage type to another of Acid, Cold, Fire, Lightning, Poison or Thunder.
	MetamagicTransmuted
	// MetamagicTwinned raises the effective level of a spell by 1 to target an additional creature.
	MetamagicTwinned
)

func metamagicIDs() []string {
	return []string{"careful_spell", "distant_spell", "empowered_spell", "extended_spell", "heightened_spell", "quickened_spell",
		"seeking_spell", "subtle_spell", "transmuted_spell", "twinned_spell"}
}

// Valid reports whether e is one of the ten effects.
func (e MetamagicEffect) Valid() bool { return e >= 1 && int(e) <= len(metamagicIDs()) }

// OptionID returns the content id of the option, "careful_spell" and so on (Table OP of content-classes-data.md), or "" for an invalid effect.
func (e MetamagicEffect) OptionID() content.FeatureOptionID {
	if !e.Valid() {
		return ""
	}
	return content.FeatureOptionID(metamagicIDs()[e-1])
}

// String returns the snake_case option id or "unknown".
func (e MetamagicEffect) String() string {
	if !e.Valid() {
		return "unknown"
	}
	return metamagicIDs()[e-1]
}

// MetamagicEffectOf returns the effect of the option id and false for an id that is not one of the ten.
func MetamagicEffectOf(id content.FeatureOptionID) (MetamagicEffect, bool) {
	i := slices.Index(metamagicIDs(), string(id))
	return MetamagicEffect(i + 1), i >= 0
}

// metamagicExempt reports whether the effect may ride on top of a different option (Empowered Spell and Seeking Spell, p. 67).
func (e MetamagicEffect) exempt() bool { return e == MetamagicEmpowered || e == MetamagicSeeking }

// SorcererState is the Sorcerer's feature state, stored in the character's FeatureStates under SorcererStateKey (class-sorcerer.md
// FR-128): the Metamagic options known, the Sorcery Points on hand, the Innate Sorcery uses and whether it is active, whether
// Sorcerous Restoration was used, the created spell slots by level, the chosen Elemental Affinity type, Dragon Wings and Dragon
// Companion use, and the per-turn flags. It holds no level, subclass, cantrip or prepared spell (FR-137). It is immutable and
// comparable with ==; the zero SorcererState is a Sorcerer with nothing, not a seeded level 1 Sorcerer (see SorcererCreationChoices).
// Safe for concurrent use.
type SorcererState struct {
	metamagic     uint16
	points        int
	uses          int
	active        bool
	restoreUsed   bool
	created       [5]int
	affinity      content.DamageType
	wingsUsed     bool
	wingsActive   bool
	companionUsed bool
	freeSpent     bool
	quickened     bool
}

// SorcererStateParts are the fields of a SorcererState for RestoreSorcererState. Construct it with a keyed literal.
type SorcererStateParts struct {
	Metamagic                   []MetamagicEffect
	SorceryPoints               int
	InnateSorceryUses           int
	InnateSorceryActive         bool
	RestorationUsed             bool
	CreatedSlots                [5]int // created slots of level 1 to 5
	ElementalAffinity           content.DamageType
	DragonWingsUsed             bool
	DragonWingsActive           bool
	DragonCompanionFreeCastUsed bool
	FreeMetamagicSpent          bool
	QuickenedUsedThisTurn       bool
}

// SorcererStateKey returns the FeatureStateKey named sorcerer (class-sorcerer.md FR-128).
func SorcererStateKey() FeatureStateKey[SorcererState] {
	return MustFeatureStateKey[SorcererState]("sorcerer")
}

// ElementalAffinityTypes are the damage types Elemental Affinity may be chosen from (SRD 5.2.1, "Level 6: Elemental Affinity", p. 70).
func ElementalAffinityTypes() []content.DamageType {
	return []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageLightning, content.DamagePoison}
}

// RestoreSorcererState validates parts against the profile and returns the state. It returns an error matching
// srderr.ErrInvalidInput for Sorcery Points outside 0 to the level's maximum, Innate Sorcery uses outside 0 to 2, a negative
// created slot count, a repeated or invalid Metamagic effect, or an Elemental Affinity type outside ElementalAffinityTypes.
// It never panics.
func RestoreSorcererState(p SorcererProfile, parts SorcererStateParts) (SorcererState, error) {
	s := SorcererState{points: parts.SorceryPoints, uses: parts.InnateSorceryUses, active: parts.InnateSorceryActive, restoreUsed: parts.RestorationUsed,
		created: parts.CreatedSlots, affinity: parts.ElementalAffinity, wingsUsed: parts.DragonWingsUsed, wingsActive: parts.DragonWingsActive,
		companionUsed: parts.DragonCompanionFreeCastUsed, freeSpent: parts.FreeMetamagicSpent, quickened: parts.QuickenedUsedThisTurn}
	if p.level.IsZero() || s.points < 0 || s.points > p.level.SorceryPointsMaximum() || s.uses < 0 || s.uses > innateSorceryUses ||
		slices.ContainsFunc(s.created[:], func(n int) bool { return n < 0 }) ||
		(s.affinity != 0 && !slices.Contains(ElementalAffinityTypes(), s.affinity)) {
		return SorcererState{}, invalidInput("Sorcerer state outside its ranges")
	}
	for _, e := range parts.Metamagic {
		if !e.Valid() || s.knows(e) {
			return SorcererState{}, invalidInput("Sorcerer state with an invalid or repeated Metamagic effect")
		}
		s.metamagic |= 1 << e
	}
	return s, nil
}

func (s SorcererState) knows(e MetamagicEffect) bool { return e.Valid() && s.metamagic&(1<<e) != 0 }

// Parts returns the fields of s, with the Metamagic effects in the printed order as a new slice.
func (s SorcererState) Parts() SorcererStateParts {
	return SorcererStateParts{Metamagic: s.Metamagic(), SorceryPoints: s.points, InnateSorceryUses: s.uses, InnateSorceryActive: s.active,
		RestorationUsed: s.restoreUsed, CreatedSlots: s.created, ElementalAffinity: s.affinity, DragonWingsUsed: s.wingsUsed, DragonWingsActive: s.wingsActive,
		DragonCompanionFreeCastUsed: s.companionUsed, FreeMetamagicSpent: s.freeSpent, QuickenedUsedThisTurn: s.quickened}
}

// Metamagic returns the Metamagic effects known, in the printed order, as a new slice.
func (s SorcererState) Metamagic() []MetamagicEffect {
	out := []MetamagicEffect{}
	for e := MetamagicCareful; e <= MetamagicTwinned; e++ {
		if s.knows(e) {
			out = append(out, e)
		}
	}
	return out
}

// Knows reports whether the Sorcerer knows the Metamagic effect.
func (s SorcererState) Knows(e MetamagicEffect) bool { return s.knows(e) }

// SorceryPoints returns the Sorcery Points on hand.
func (s SorcererState) SorceryPoints() int { return s.points }

// InnateSorceryUses returns the uses of Innate Sorcery remaining, 0 to 2.
func (s SorcererState) InnateSorceryUses() int { return s.uses }

// InnateSorceryActive reports whether Innate Sorcery is active.
func (s SorcererState) InnateSorceryActive() bool { return s.active }

// RestorationUsed reports whether Sorcerous Restoration was used since the last Long Rest.
func (s SorcererState) RestorationUsed() bool { return s.restoreUsed }

// CreatedSlots returns the number of created spell slots of the level (1 to 5), 0 for any other level.
func (s SorcererState) CreatedSlots(level int) int {
	if level < 1 || level > len(s.created) {
		return 0
	}
	return s.created[level-1]
}

// ElementalAffinity returns the damage type chosen for Elemental Affinity and false before a choice.
func (s SorcererState) ElementalAffinity() (content.DamageType, bool) {
	return s.affinity, s.affinity != 0
}

// DragonWingsUsed reports whether the use of Dragon Wings is expended.
func (s SorcererState) DragonWingsUsed() bool { return s.wingsUsed }

// DragonWingsActive reports whether the wings are out.
func (s SorcererState) DragonWingsActive() bool { return s.wingsActive }

// DragonCompanionFreeCastUsed reports whether the slot-free cast of Summon Dragon is expended.
func (s SorcererState) DragonCompanionFreeCastUsed() bool { return s.companionUsed }

// FreeMetamagicSpent reports whether this turn's free Metamagic use (Arcane Apotheosis) is spent.
func (s SorcererState) FreeMetamagicSpent() bool { return s.freeSpent }

// QuickenedUsedThisTurn reports whether Quickened Spell was used this turn.
func (s SorcererState) QuickenedUsedThisTurn() bool { return s.quickened }

// Equal reports whether other is a SorcererState with the same fields.
func (s SorcererState) Equal(other FeatureState) bool {
	o, ok := other.(SorcererState)
	return ok && o == s
}

// SorcererCreationChoices is the ClassCreationChoices of a new Sorcerer: it has no field because the Sorcerer has no
// class-specific creation choice, and it seeds a level 1 SorcererState (0 Sorcery Points, 2 Innate Sorcery uses, no Metamagic,
// no created slots) under SorcererStateKey (class-sorcerer.md FR-010, FR-129). It names no content. Safe for concurrent use.
type SorcererCreationChoices struct{}

// SeedFeatureStates returns states with the level 1 SorcererState added and every other entry unchanged; states itself is
// unchanged. States that already hold the key are returned unchanged: CreateCharacter starts from empty states, so that is a
// programmer error, and the package does not panic outside Must helpers.
func (SorcererCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	return seedSorcererState(states)
}

// ContentKeys returns an empty slice: the first-level choices name no content the choices themselves load.
func (SorcererCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

func seedSorcererState(states FeatureStates) FeatureStates {
	if _, held, err := ReadFeatureState(states, SorcererStateKey()); held || err != nil {
		return states
	}
	next, err := SetFeatureState(states, SorcererStateKey(), SorcererState{uses: innateSorceryUses})
	if err != nil { // unreachable: the key is valid, the states are non-nil and the name is not held
		return states
	}
	return next
}

var _ ClassCreationChoices = SorcererCreationChoices{}
