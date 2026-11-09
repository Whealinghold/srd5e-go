package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// AdvancePaladinResult is the closed set of outcomes of AdvancePaladin. Variants: AdvancePaladinApplied, AdvancePaladinNotAPaladin.
type AdvancePaladinResult interface{ isAdvancePaladinResult() }

// AdvancePaladinApplied is the outcome of a Paladin level gained. Only AdvancePaladin builds one.
type AdvancePaladinApplied struct {
	character Character
	state     PaladinState
	advanced  PaladinAdvanced
}

// Character returns the Character holding the new PaladinState.
func (r AdvancePaladinApplied) Character() Character { return r.character }

// State returns the state at the new level.
func (r AdvancePaladinApplied) State() PaladinState { return r.state }

// FeaturesGained returns the ids of the features the new level gains, as a new slice.
func (r AdvancePaladinApplied) FeaturesGained() []content.FeatureID {
	return r.advanced.FeaturesGained()
}

// PendingChoices returns the Paladin-specific choices the level adds, as a new slice.
func (r AdvancePaladinApplied) PendingChoices() []PaladinPendingChoice {
	return r.advanced.PendingChoices()
}

// AlwaysPreparedSpells returns the always-prepared spells newly granted at the level, as a new slice.
func (r AdvancePaladinApplied) AlwaysPreparedSpells() []content.SpellID {
	return r.advanced.AlwaysPreparedSpellsGranted()
}

// Events returns a new slice holding the one PaladinAdvanced event.
func (r AdvancePaladinApplied) Events() []Event { return []Event{r.advanced} }

// AdvancePaladinNotAPaladin is the outcome for a character that holds no Paladin level at all.
type AdvancePaladinNotAPaladin struct{}

func (AdvancePaladinApplied) isAdvancePaladinResult()     {}
func (AdvancePaladinNotAPaladin) isAdvancePaladinResult() {}

// AdvancePaladin applies the Paladin side of a gained level (class-paladin.md FR-098). It is called after level advancement has
// recorded the new level, so c already holds the new Paladin level and previous is the Paladin level before it. It raises the
// Lay On Hands remaining amount by the 5 the maximum rises by; raises the Channel Divinity remaining uses by the rise of the
// maximum (never above it); and makes the use of a free cast or of Holy Nimbus available at the level of its feature entry (a
// newly gained use starts available, so Divine Smite can be cast at once on reaching level 2). It refills nothing else and
// spends nothing. subclass is the subclass the Paladin holds ("" for none); the always-prepared spells newly granted at the level
// are reported in one PaladinAdvanced event with the features gained, the pending choices and the uses gained, whatever the level.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or progression, a previous level that is not one
// below the new level, or another type of state under the Paladin key. There is no rejection.
func AdvancePaladin(c Character, previous int, p PaladinProgression, subclass content.SubclassID) (AdvancePaladinResult, error) {
	if c.IsZero() || p.IsZero() {
		return nil, invalidInput("AdvancePaladin: zero Character or progression")
	}
	level, ok := paladinLevelOf(c)
	if !ok {
		return AdvancePaladinNotAPaladin{}, nil
	}
	if previous != level.n-1 {
		return nil, invalidInput("AdvancePaladin: previous Paladin level " + strconv.Itoa(previous) + " is not one below " + strconv.Itoa(level.n))
	}
	prev := PaladinLevel{n: previous} // the zero level when the level gained is the first
	maximum, previousMaximum := p.ChannelDivinityMaximum(level), p.ChannelDivinityMaximum(prev)
	gained := max(maximum-previousMaximum, 0)
	state, held, err := ReadFeatureState(c.FeatureStates(), PaladinStateKey())
	if err != nil {
		return nil, err
	}
	if held {
		state = state.clamp(PaladinLevel{n: max(previous, 1)}, previousMaximum)
		state.layRemaining += layOnHandsPerLevel
		state.channelRemaining = min(state.channelRemaining+gained, maximum)
	} else {
		state = InitialPaladinState(level, maximum)
	}
	for _, f := range []FreeCastFeature{FreeCastPaladinsSmite, FreeCastFaithfulSteed} {
		if p.LevelOf(f.paladinFeature()) == level.n {
			if f == FreeCastFaithfulSteed {
				state.steed = LongRestUseAvailable
			} else {
				state.smite = LongRestUseAvailable
			}
		}
	}
	if p.LevelOf(PaladinFeatureHolyNimbus) == level.n {
		state.nimbus = LongRestUseAvailable
	}
	oath := subclass == OathOfDevotionSubclassID
	before := p.AlwaysPrepared(prev, oath)
	spells := slices.DeleteFunc(p.AlwaysPrepared(level, oath), func(id content.SpellID) bool { return slices.Contains(before, id) })
	next, err := WithFeatureState(c, PaladinStateKey(), state)
	if err != nil { // unreachable: the key is valid and the state was read from the same key
		return nil, err
	}
	return AdvancePaladinApplied{character: next, state: state, advanced: PaladinAdvanced{
		id: c.ID(), level: level.n, features: p.FeaturesGainedAt(level), pending: p.ChoicesOnGaining(level), usesGained: gained, spells: spells,
	}}, nil
}

// ApplyPaladinRestApplied is the outcome of ApplyPaladinRest: the state after the rest and the events it produced.
type ApplyPaladinRestApplied struct {
	state  PaladinState
	events []Event
}

// State returns the state after the rest.
func (r ApplyPaladinRestApplied) State() PaladinState { return r.state }

// Events returns a new slice holding the PaladinResourcesRestored event, and none when the rest changed nothing.
func (r ApplyPaladinRestApplied) Events() []Event { return slices.Clone(r.events) }

// ApplyPaladinRest applies the benefits of a finished rest (RestApplied.BenefitsOf) to a Paladin's resources (class-paladin.md
// FR-096, FR-046, FR-064, FR-067): a Short Rest regains one expended Channel Divinity use, never above the maximum, and nothing
// else; a Long Rest refills the Lay On Hands pool and the Channel Divinity uses and makes both free casts and Holy Nimbus
// available. Active effects are left to EndPaladinEffect. Any other kind changes nothing. The inputs are not altered. There is
// no rejection and no NotAPaladin: it takes no Character.
func ApplyPaladinRest(id Identifier, p PaladinProgression, level PaladinLevel, state PaladinState, kind RestKind) ApplyPaladinRestApplied {
	maximum := p.ChannelDivinityMaximum(level)
	after := state.clamp(level, maximum)
	var casts []FreeCastFeature
	nimbus := false
	switch kind {
	case ShortRest:
		after.channelRemaining = min(after.channelRemaining+1, maximum)
	case LongRest:
		after.layRemaining, after.channelRemaining = LayOnHandsMaximum(level), maximum
		if after.smite == LongRestUseSpent {
			casts = append(casts, FreeCastPaladinsSmite)
		}
		if after.steed == LongRestUseSpent {
			casts = append(casts, FreeCastFaithfulSteed)
		}
		nimbus = after.nimbus == LongRestUseSpent
		after.smite, after.steed, after.nimbus = LongRestUseAvailable, LongRestUseAvailable, LongRestUseAvailable
	}
	before := state.clamp(level, maximum)
	out := ApplyPaladinRestApplied{state: after, events: []Event{}}
	lay, uses := after.layRemaining-before.layRemaining, after.channelRemaining-before.channelRemaining
	if lay > 0 || uses > 0 || len(casts) > 0 || nimbus {
		out.events = append(out.events, PaladinResourcesRestored{id: id, layOnHands: lay, channelUses: uses, freeCasts: casts, holyNimbus: nimbus})
	}
	return out
}

// StartPaladinResult is the closed set of outcomes of StartPaladin. Variants: StartPaladinStarted, StartPaladinAlreadyAPaladin.
type StartPaladinResult interface{ isStartPaladinResult() }

// StartPaladinStarted is the outcome of a Paladin started. Only StartPaladin builds one.
type StartPaladinStarted struct {
	character Character
	events    []Event
}

// Character returns the Character holding the level 1 PaladinState.
func (r StartPaladinStarted) Character() Character { return r.character }

// Events returns a new slice holding the PaladinLevelOneGained event for a multiclass entry and none for a first-level entry,
// where CharacterCreated is the record.
func (r StartPaladinStarted) Events() []Event { return slices.Clone(r.events) }

// StartPaladinAlreadyAPaladin is the outcome for a Character that already holds a PaladinState.
type StartPaladinAlreadyAPaladin struct{}

func (StartPaladinStarted) isStartPaladinResult()         {}
func (StartPaladinAlreadyAPaladin) isStartPaladinResult() {}

// StartPaladin stores the level 1 PaladinState of a Paladin with every pool full and every use available (class-paladin.md
// FR-111): Lay On Hands at 5, no Channel Divinity uses. The host calls it after the level-advanced event of a multiclass entry; a
// first-level Paladin is created through PaladinCreationChoices instead, so no PaladinLevelOneGained event is emitted for it. The
// gate of each feature is the level, so the unspent uses are inert until gained.
//
// It returns StartPaladinAlreadyAPaladin when the Character already holds a PaladinState, and an error matching
// srderr.ErrInvalidInput for a zero Character or progression, an undefined entry, a Character that does not hold exactly Paladin
// level 1, or another type of state under the Paladin key.
func StartPaladin(c Character, p PaladinProgression, entry ClassEntry) (StartPaladinResult, error) {
	if c.IsZero() || p.IsZero() || !entry.Valid() {
		return nil, invalidInput("StartPaladin: zero Character or progression, or an undefined entry")
	}
	level, ok := paladinLevelOf(c)
	if !ok || level.n != 1 {
		return nil, invalidInput("StartPaladin: the Character does not hold Paladin level 1")
	}
	if _, held, err := ReadFeatureState(c.FeatureStates(), PaladinStateKey()); err != nil {
		return nil, err
	} else if held {
		return StartPaladinAlreadyAPaladin{}, nil
	}
	next, err := WithFeatureState(c, PaladinStateKey(), InitialPaladinState(level, p.ChannelDivinityMaximum(level)))
	if err != nil { // unreachable: the key is valid
		return nil, err
	}
	out := StartPaladinStarted{character: next, events: []Event{}}
	if entry == ClassEntryMulticlass {
		out.events = append(out.events, PaladinLevelOneGained{id: c.ID(), entry: entry})
	}
	return out, nil
}
