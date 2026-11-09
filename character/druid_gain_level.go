package character

import "github.com/Whealinghold/srd5e-go/content"

// GainDruidLevelInput is what GainDruidLevel reads. Construct it with a keyed literal.
type GainDruidLevelInput struct {
	// ID is the character's id, carried by the event.
	ID Identifier
	// Previous is the Druid level before the gain, 0 for a character with no Druid level.
	Previous int
	// Target is the Druid level to gain; any value is accepted and judged by the rules.
	Target int
	// State is the Druid state before the gain (the zero value for a new Druid).
	State  DruidFeatureState
	Tables DruidFeatureTables
	// Subclass is the subclass the character holds, "" for none.
	Subclass content.SubclassID
}

// GainDruidLevelResult is the outcome of GainDruidLevel, a closed set. Variants: GainDruidLevelApplied, GainDruidLevelRejected.
type GainDruidLevelResult interface{ isGainDruidLevelResult() }

// GainDruidLevelApplied: the level was gained. Hold State, publish Events.
type GainDruidLevelApplied struct {
	state    DruidFeatureState
	features []content.FeatureID
	pending  []DruidPendingChoice
	events   []Event
}

// State returns the new Druid state.
func (r GainDruidLevelApplied) State() DruidFeatureState { return r.state }

// Features returns the ids of the features granted at exactly the new level, as a new slice; none repeats an earlier level.
func (r GainDruidLevelApplied) Features() []content.FeatureID {
	return append([]content.FeatureID{}, r.features...)
}

// PendingChoices returns the Druid-owned choices the level opens, as a new slice.
func (r GainDruidLevelApplied) PendingChoices() []DruidPendingChoice {
	return append([]DruidPendingChoice{}, r.pending...)
}

// Events returns the events, as a new slice.
func (r GainDruidLevelApplied) Events() []Event { return append([]Event{}, r.events...) }

// GainDruidLevelRejected: the level was not gained; Reason is DruidLevelNotConsecutive or DruidMaximumLevelReached.
type GainDruidLevelRejected struct{ Reason DruidRejection }

func (GainDruidLevelApplied) isGainDruidLevelResult()  {}
func (GainDruidLevelRejected) isGainDruidLevelResult() {}

// GainDruidLevel gains one Druid level (class-druid.md FR-023 to FR-031, FR-056, FR-059). It accepts only the next level, none
// to 1 or n to n+1 (DruidLevelNotConsecutive), and none above 20 (DruidMaximumLevelReached). The result lists the features
// granted at exactly the new level and the pending Druid choices: the Primal Order at level 1, the known forms at levels 2, 4
// and 8 (the difference from the forms already known) and Elemental Fury at level 7. Gaining level 2 grants Wild Shape with
// all its uses; a later level that raises the maximum raises the remaining uses by the difference. No earlier choice is
// changed. The skills and Starting Equipment choices of a starting Druid and the training of a multiclass Druid belong to
// character creation and level advancement.
//
// It returns an error matching srderr.ErrInvalidInput for a zero id or tables, a Previous level outside 0
// to 20, or a State that breaks an invariant at the Previous level. It never panics.
func GainDruidLevel(in GainDruidLevelInput) (GainDruidLevelResult, error) {
	if in.ID.IsZero() || in.Tables.IsZero() || in.Previous < 0 || in.Previous > MaxLevel {
		return nil, invalidInput("GainDruidLevel with a zero id or tables, or a previous level outside 0 to 20")
	}
	if in.Previous > 0 {
		if err := druidViolations(in.State.violations(in.Previous, in.Subclass == CircleOfTheLandSubclassID, in.Tables)); err != nil {
			return nil, err
		}
	} else if in.State != (DruidFeatureState{}) {
		return nil, invalidInput("GainDruidLevel from no Druid level with a Druid state")
	}
	var consecutive, maximum DruidRejection
	if in.Target != in.Previous+1 {
		consecutive = DruidLevelNotConsecutive
	}
	if in.Target > MaxLevel {
		maximum = DruidMaximumLevelReached
	}
	if reason, bad := lowestDruidRejection(consecutive, maximum); bad {
		return GainDruidLevelRejected{Reason: reason}, nil
	}
	state := in.State
	if in.Target == 2 {
		state.uses = in.Tables.WildShapeUses(2)
	} else if in.Target > 2 {
		state.uses += in.Tables.WildShapeUses(in.Target) - in.Tables.WildShapeUses(in.Previous)
	}
	var pending []DruidPendingChoice
	if in.Target == 1 && state.order == 0 {
		pending = append(pending, DruidPendingChoice{Kind: DruidPendingPrimalOrder, Count: 1})
	}
	if raised := in.Tables.KnownFormsAt(in.Target) > in.Tables.KnownFormsAt(in.Previous); raised && in.Tables.KnownFormsAt(in.Target) > state.known.Len() {
		pending = append(pending, DruidPendingChoice{Kind: DruidPendingKnownForms, Count: in.Tables.KnownFormsAt(in.Target) - state.known.Len()})
	}
	if in.Target == 7 && state.fury == 0 {
		pending = append(pending, DruidPendingChoice{Kind: DruidPendingElementalFury, Count: 1})
	}
	features := in.Tables.FeaturesAt(in.Target)
	return GainDruidLevelApplied{state: state, features: features, pending: pending,
		events: []Event{DruidLevelGained{id: in.ID, level: in.Target, features: features, pending: pending}}}, nil
}
