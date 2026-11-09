package character

// DruidTrigger is something the host reports that the Druid state reacts to, a closed set. Variants: DruidTriggerRestFinished,
// DruidTriggerTurnStarted, DruidTriggerInitiativeRolled.
type DruidTrigger interface{ isDruidTrigger() }

// DruidTriggerRestFinished reports a finished rest of Kind (ShortRest or LongRest). Construct it with a keyed literal.
type DruidTriggerRestFinished struct{ Kind RestKind }

// DruidTriggerTurnStarted reports the start of the Druid's turn.
type DruidTriggerTurnStarted struct{}

// DruidTriggerInitiativeRolled reports that the Druid rolled Initiative.
type DruidTriggerInitiativeRolled struct{}

func (DruidTriggerRestFinished) isDruidTrigger()     {}
func (DruidTriggerTurnStarted) isDruidTrigger()      {}
func (DruidTriggerInitiativeRolled) isDruidTrigger() {}

// ApplyDruidTriggerInput is what ApplyDruidTrigger reads. Construct it with a keyed literal.
type ApplyDruidTriggerInput struct {
	State   DruidFeatureState
	Tables  DruidFeatureTables
	Facts   DruidCharacterFacts
	Trigger DruidTrigger
}

// ApplyDruidTriggerApplied is the outcome of ApplyDruidTrigger. There is no rejected variant. Hold State, publish Events.
type ApplyDruidTriggerApplied struct {
	state  DruidFeatureState
	events []Event
}

// State returns the new Druid state.
func (r ApplyDruidTriggerApplied) State() DruidFeatureState { return r.state }

// Events returns the events, as a new slice.
func (r ApplyDruidTriggerApplied) Events() []Event { return append([]Event{}, r.events...) }

// ApplyDruidTrigger applies a trigger to the Druid state (class-druid.md FR-153, FR-154, FR-156, FR-158, FR-159, table N). It
// takes and returns no spell slot: slot restoration after a Long Rest stays in RestoreSpellSlots.
//
//   - A finished Long Rest clears the once-until-Long-Rest flags and the known form replacement count, regains every Wild Shape
//     use, dismisses a Wild Companion familiar with WildCompanionDismissed, and replaces the open window by the permissions it
//     grants: known form replacement from level 2, land type change with the Circle of the Land from level 3.
//   - A finished Short Rest regains one expended Wild Shape use, clears no flag, and replaces the open window by natural
//     recovery recovery from level 6 with the Circle of the Land.
//   - The start of a turn clears the two once-per-turn flags and closes the window.
//   - Rolling Initiative closes the window and, at level 20 with no Wild Shape use left, regains one use
//     (WildShapeUsesChanged, Evergreen Wild Shape).
//
// It returns an error matching srderr.ErrInvalidInput for zero facts or tables, a State that breaks an invariant, a nil
// trigger or a rest kind other than ShortRest or LongRest. It never panics.
func ApplyDruidTrigger(in ApplyDruidTriggerInput) (ApplyDruidTriggerApplied, error) {
	if err := in.State.check(in.Facts, in.Tables); err != nil {
		return ApplyDruidTriggerApplied{}, err
	}
	s, level, id := in.State, in.Facts.lv(), in.Facts.id
	maximum := in.Tables.WildShapeUses(level)
	var events []Event
	switch t := in.Trigger.(type) {
	case DruidTriggerRestFinished:
		s.window = 0
		switch t.Kind {
		case LongRest:
			s.wrLongRest, s.nrCast, s.nrRecover, s.nature, s.swaps = false, false, false, false, 0
			if in.Facts.hasWildShape() {
				s.uses = maximum
				s.window = s.window.with(RestWindowKnownFormReplacement)
			}
			if in.Facts.hasCircle() && level >= 3 {
				s.window = s.window.with(RestWindowLandTypeChange)
			}
			if s.familiar {
				s.familiar = false
				events = append(events, WildCompanionDismissed{id: id})
			}
		case ShortRest:
			s.uses = min(s.uses+1, maximum)
			if in.Facts.hasCircle() && level >= 6 {
				s.window = s.window.with(RestWindowNaturalRecoveryRecovery)
			}
		default:
			return ApplyDruidTriggerApplied{}, invalidInput("Druid rest trigger with a rest kind that is neither short nor long")
		}
	case DruidTriggerTurnStarted:
		s.wrTurn, s.psTurn, s.window = false, false, 0
	case DruidTriggerInitiativeRolled:
		s.window = 0
		if level >= 20 && s.uses == 0 {
			s.uses = 1
			events = append(events, WildShapeUsesChanged{id: id, from: 0, to: 1, cause: WildShapeUsesEvergreenWildShape})
		}
	default:
		return ApplyDruidTriggerApplied{}, invalidInput("Druid trigger is nil or not one of the closed set")
	}
	return ApplyDruidTriggerApplied{state: s, events: events}, nil
}
