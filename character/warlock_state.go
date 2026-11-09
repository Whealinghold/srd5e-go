package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// warlockInvocationIDs are the ids of the 28 Eldritch Invocation options in the printed alphabetical order (SRD 5.2.1, "Eldritch
// Invocation Options", pp. 72-74; Table OP of content-classes-data.md).
func warlockInvocationIDs() []content.FeatureOptionID {
	return []content.FeatureOptionID{"agonizing_blast", "armor_of_shadows", "ascendant_step", "devils_sight", "devouring_blade", "eldritch_mind",
		"eldritch_smite", "eldritch_spear", "fiendish_vigor", "gaze_of_two_minds", "gift_of_the_depths", "gift_of_the_protectors",
		"investment_of_the_chain_master", "lessons_of_the_first_ones", "lifedrinker", "mask_of_many_faces", "master_of_myriad_forms",
		"misty_visions", "one_with_shadows", "otherworldly_leap", "pact_of_the_blade", "pact_of_the_chain", "pact_of_the_tome",
		"repelling_blast", "thirsting_blade", "visions_of_distant_realms", "whispers_of_the_grave", "witch_sight"}
}

// WarlockState is the Warlock's feature state, stored in the character's FeatureStates under WarlockStateKey (class-warlock.md
// FR-158): the Eldritch Invocations known (a count per option, so a Repeatable option may appear more than once), the Mystic Arcanum
// spells and which are cast, and the uses of Magical Cunning, Contact Patron, Gift of the Depths, Gift of the Protectors, Dark One's
// Own Luck and Hurl Through Hell spent since the last Long Rest, and the Fiendish Resilience damage type. It holds no level,
// subclass, cantrip or prepared spell. It is immutable and comparable with ==; the zero WarlockState is a Warlock who knows nothing
// and has used nothing, which is also the state of a new level 1 Warlock before the first invocation is chosen. Safe for concurrent use.
type WarlockState struct {
	invocations     [28]uint8
	arcanum         [4]content.SpellID
	arcanumCast     [4]bool
	cunningUsed     bool
	patronUsed      bool
	depthsCast      bool
	protectorsSpent bool
	hurlSpent       bool
	luckSpent       int
	resilience      content.DamageType
}

// WarlockStateParts are the fields of a WarlockState for RestoreWarlockState. Construct it with a keyed literal.
type WarlockStateParts struct {
	// Invocations lists the options known, a Repeatable option once per time it was gained.
	Invocations []content.FeatureOptionID
	// Arcanum are the arcanum spells of levels 6 to 9 in that order, empty for a slot not yet chosen.
	Arcanum [4]content.SpellID
	// ArcanumCast says which arcanum spells were cast since the last Long Rest.
	ArcanumCast                                                [4]bool
	MagicalCunningUsed, ContactPatronUsed, GiftOfTheDepthsCast bool
	GiftOfTheProtectorsSpent, HurlThroughHellSpent             bool
	DarkOnesOwnLuckSpent                                       int
	FiendishResilience                                         content.DamageType
}

// WarlockStateKey returns the FeatureStateKey named warlock (class-warlock.md FR-158).
func WarlockStateKey() FeatureStateKey[WarlockState] {
	return MustFeatureStateKey[WarlockState]("warlock")
}

// RestoreWarlockState validates parts against the profile and returns the state. It returns an error matching
// srderr.ErrInvalidInput for an invocation that is not one of the 28 (whether it may repeat is the class data, checked by SelectInvocations), an arcanum spell beyond the
// level's count or an invalid id, a cast of a spell not chosen, a negative or too large number of Dark One's Own Luck uses spent
// (above 20), or a Fiendish Resilience type that is Force or invalid. It never panics.
func RestoreWarlockState(p WarlockProfile, parts WarlockStateParts) (WarlockState, error) {
	s := WarlockState{arcanum: parts.Arcanum, arcanumCast: parts.ArcanumCast, cunningUsed: parts.MagicalCunningUsed, patronUsed: parts.ContactPatronUsed,
		depthsCast: parts.GiftOfTheDepthsCast, protectorsSpent: parts.GiftOfTheProtectorsSpent, hurlSpent: parts.HurlThroughHellSpent,
		luckSpent: parts.DarkOnesOwnLuckSpent, resilience: parts.FiendishResilience}
	if p.level.IsZero() || s.luckSpent < 0 || s.luckSpent > MaxLevel ||
		(s.resilience != 0 && (!s.resilience.Valid() || s.resilience == content.DamageForce)) {
		return WarlockState{}, invalidInput("Warlock state outside its ranges")
	}
	for i, spell := range s.arcanum {
		if (spell != "" && (!spell.Valid() || i >= p.level.ArcanumCount())) || (s.arcanumCast[i] && spell == "") {
			return WarlockState{}, invalidInput("Warlock state with an arcanum spell the level does not give, or a cast of an unchosen one")
		}
	}
	for _, id := range parts.Invocations {
		i := slices.Index(warlockInvocationIDs(), id)
		if i < 0 || s.invocations[i] == 255 {
			return WarlockState{}, invalidInput("Warlock state with an unknown Eldritch Invocation")
		}
		s.invocations[i]++
	}
	return s, nil
}

// Parts returns the fields of s, with the invocations in the printed order as a new slice.
func (s WarlockState) Parts() WarlockStateParts {
	return WarlockStateParts{Invocations: s.Invocations(), Arcanum: s.arcanum, ArcanumCast: s.arcanumCast, MagicalCunningUsed: s.cunningUsed,
		ContactPatronUsed: s.patronUsed, GiftOfTheDepthsCast: s.depthsCast, GiftOfTheProtectorsSpent: s.protectorsSpent, HurlThroughHellSpent: s.hurlSpent,
		DarkOnesOwnLuckSpent: s.luckSpent, FiendishResilience: s.resilience}
}

// Invocations returns the invocations known in the printed order, a Repeatable one once per time gained, as a new slice.
func (s WarlockState) Invocations() []content.FeatureOptionID {
	out := []content.FeatureOptionID{}
	for i, id := range warlockInvocationIDs() {
		for range s.invocations[i] {
			out = append(out, id)
		}
	}
	return out
}

// Knows reports whether the Warlock knows the invocation.
func (s WarlockState) Knows(id content.FeatureOptionID) bool {
	i := slices.Index(warlockInvocationIDs(), id)
	return i >= 0 && s.invocations[i] > 0
}

// Arcanum returns the arcanum spell of the level (6 to 9) and false when none is chosen or the level is outside 6 to 9.
func (s WarlockState) Arcanum(spellLevel int) (content.SpellID, bool) {
	i := spellLevel - warlockArcanumFirstSpellLvl
	if i < 0 || i >= len(s.arcanum) || s.arcanum[i] == "" {
		return "", false
	}
	return s.arcanum[i], true
}

// ArcanumCast reports whether the arcanum spell of the level (6 to 9) was cast since the last Long Rest.
func (s WarlockState) ArcanumCast(spellLevel int) bool {
	i := spellLevel - warlockArcanumFirstSpellLvl
	return i >= 0 && i < len(s.arcanumCast) && s.arcanumCast[i]
}

// MagicalCunningUsed reports whether Magical Cunning was used since the last Long Rest.
func (s WarlockState) MagicalCunningUsed() bool { return s.cunningUsed }

// ContactPatronUsed reports whether the slot-free cast of Contact Other Plane was used since the last Long Rest.
func (s WarlockState) ContactPatronUsed() bool { return s.patronUsed }

// GiftOfTheDepthsCast reports whether the slot-free cast of Water Breathing was used since the last Long Rest.
func (s WarlockState) GiftOfTheDepthsCast() bool { return s.depthsCast }

// GiftOfTheProtectorsSpent reports whether the page's benefit was triggered since the last Long Rest.
func (s WarlockState) GiftOfTheProtectorsSpent() bool { return s.protectorsSpent }

// HurlThroughHellSpent reports whether the use of Hurl Through Hell is expended.
func (s WarlockState) HurlThroughHellSpent() bool { return s.hurlSpent }

// DarkOnesOwnLuckSpent returns the uses of Dark One's Own Luck spent since the last Long Rest.
func (s WarlockState) DarkOnesOwnLuckSpent() int { return s.luckSpent }

// FiendishResilience returns the damage type chosen for Fiendish Resilience and false before a choice.
func (s WarlockState) FiendishResilience() (content.DamageType, bool) {
	return s.resilience, s.resilience != 0
}

// Equal reports whether other is a WarlockState with the same fields.
func (s WarlockState) Equal(other FeatureState) bool {
	o, ok := other.(WarlockState)
	return ok && o == s
}

// WarlockCreationChoices is the ClassCreationChoices of a new Warlock: it has no field because the class-specific first-level
// choices (the skills and the Starting Equipment) are checked by CreateCharacter over the class data, and it seeds an empty
// WarlockState under WarlockStateKey (class-warlock.md FR-158, FR-159). It names no content. Safe for concurrent use.
type WarlockCreationChoices struct{}

// SeedFeatureStates returns states with the empty WarlockState added and every other entry unchanged; states itself is unchanged.
// States that already hold the key are returned unchanged: CreateCharacter starts from empty states, so that is a programmer error,
// and the package does not panic outside Must helpers.
func (WarlockCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	if _, held, err := ReadFeatureState(states, WarlockStateKey()); held || err != nil {
		return states
	}
	next, err := SetFeatureState(states, WarlockStateKey(), WarlockState{})
	if err != nil { // unreachable: the key is valid, the states are non-nil and the name is not held
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the first-level choices name no content the choices themselves load.
func (WarlockCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = WarlockCreationChoices{}
