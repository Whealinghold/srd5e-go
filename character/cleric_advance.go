package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// AdvanceClericResult is the closed set of outcomes of AdvanceCleric. Variants: AdvanceClericApplied,
// AdvanceClericNotACleric.
type AdvanceClericResult interface{ isAdvanceClericResult() }

// AdvanceClericApplied is the outcome of a Cleric level gained. Only AdvanceCleric builds one.
type AdvanceClericApplied struct {
	character Character
	resources ClericResources
	advanced  ClericAdvanced
}

// Character returns the Character holding the new ClericResources (and the ClericChoices, the empty ones when it held none).
func (r AdvanceClericApplied) Character() Character { return r.character }

// Resources returns the resources at the new level.
func (r AdvanceClericApplied) Resources() ClericResources { return r.resources }

// FeaturesGained returns the ids of the features the new level gains, as a new slice.
func (r AdvanceClericApplied) FeaturesGained() []content.FeatureID {
	return r.advanced.FeaturesGained()
}

// PendingChoices returns the Cleric-specific choices the level adds, as a new slice.
func (r AdvanceClericApplied) PendingChoices() []ClericPendingChoice {
	return r.advanced.PendingChoices()
}

// AlwaysPreparedSpells returns the Life Domain spells newly granted at the level, as a new slice.
func (r AdvanceClericApplied) AlwaysPreparedSpells() []content.SpellID {
	return r.advanced.AlwaysPreparedSpellsGranted()
}

// Events returns a new slice holding the one ClericAdvanced event.
func (r AdvanceClericApplied) Events() []Event { return []Event{r.advanced} }

// AdvanceClericNotACleric is the outcome for a character that holds no Cleric level at all.
type AdvanceClericNotACleric struct{}

func (AdvanceClericApplied) isAdvanceClericResult()    {}
func (AdvanceClericNotACleric) isAdvanceClericResult() {}

// AdvanceCleric applies the Cleric side of a gained level. It is called after level advancement has recorded the
// new level, so c already holds the new Cleric level and previous is the Cleric level before it (0 for the first).
// It raises the Channel Divinity maximum to the progression's value at the new level and the remaining uses by the
// same number, never above the maximum (2 at level 2, 1 at 6 and at 18 for the SRD data); a character that holds no
// ClericResources gets the initial ones first (Channel Divinity 0 of 0, Divine Intervention available). held is the
// Life Domain SubclassDefinition the character holds, or nil; the spells of its first grant are not reported at level 3, where the subclass is chosen. It reports the features gained, the pending choices, the
// uses gained and the always-prepared spells newly granted in one ClericAdvanced event, whatever the level.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or progression, a previous level that is not
// one below the new level, or another type of state under a Cleric key. There is no rejection.
func AdvanceCleric(c Character, previous int, progression ClericProgression, held *content.SubclassDefinition) (AdvanceClericResult, error) {
	if c.IsZero() || progression.IsZero() {
		return nil, invalidInput("AdvanceCleric: zero Character or progression")
	}
	level, ok := clericLevelOf(c)
	if !ok {
		return AdvanceClericNotACleric{}, nil
	}
	if previous != level.n-1 {
		return nil, invalidInput("AdvanceCleric: previous Cleric level " + strconv.Itoa(previous) + " is not one below " + strconv.Itoa(level.n))
	}
	choices, err := readClericChoices(c)
	if err != nil {
		return nil, err
	}
	res, err := readClericResources(c, 0)
	if err != nil {
		return nil, err
	}
	maximum := progression.StatsAt(level).channel
	gained := max(maximum-res.channelMax, 0)
	res.channelRemaining = min(res.channelRemaining+gained, maximum)
	res.channelMax = maximum
	res.intervention = res.DivineIntervention()
	// The grant of the subclass level itself is made when the caller reacts to SubclassChosen (class-cleric.md FR-134, AC-03).
	var spells []content.SpellID
	if level.n > clericSubclassLevel {
		spells = LifeDomainSpells{}.NewlyGrantedAt(held, level.n)
	}
	next, err := WithFeatureState(c, ClericResourcesKey(), res)
	if err == nil {
		next, err = WithFeatureState(next, ClericChoicesKey(), choices)
	}
	if err != nil { // unreachable: the keys are valid and the states are read from the same keys
		return nil, err
	}
	return AdvanceClericApplied{character: next, resources: res, advanced: ClericAdvanced{
		id: c.ID(), level: level.n, features: progression.FeaturesGainedAt(level), pending: progression.ChoicesOnGaining(level),
		usesGained: gained, spells: spells,
	}}, nil
}

// ApplyClericRestApplied is the outcome of ApplyClericRest: the resources after the rest and the events it produced.
type ApplyClericRestApplied struct {
	resources ClericResources
	events    []Event
}

// Resources returns the resources after the rest.
func (r ApplyClericRestApplied) Resources() ClericResources { return r.resources }

// Events returns a new slice of the events: ClericChannelDivinityRegained when uses were regained and
// DivineInterventionRecharged when Divine Intervention became available again.
func (r ApplyClericRestApplied) Events() []Event { return slices.Clone(r.events) }

// ApplyClericRest applies the benefits of a finished rest (RestApplied.BenefitsOf) to a Cleric's resources for the
// character id: see ClericResources.AfterRest. There is no rejection and no NotACleric: it takes no Character.
func ApplyClericRest(id Identifier, resources ClericResources, kind RestKind) ApplyClericRestApplied {
	after := resources.AfterRest(kind)
	out := ApplyClericRestApplied{resources: after, events: []Event{}}
	if regained := after.channelRemaining - resources.channelRemaining; regained > 0 {
		out.events = append(out.events, ClericChannelDivinityRegained{id: id, uses: regained})
	}
	if _, wasAvailable := resources.DivineIntervention().(DivineInterventionAvailable); !wasAvailable {
		if _, isAvailable := after.intervention.(DivineInterventionAvailable); isAvailable {
			out.events = append(out.events, DivineInterventionRecharged{id: id})
		}
	}
	return out
}

// clericSubclassLevel is the Cleric level at which the subclass is chosen; the Life Domain spells of that level are granted by the
// caller when it reacts to SubclassChosen.
const clericSubclassLevel = 3
