package combat

// ResolutionPrecedence is the order in which the combat action resolution use cases report a stop when
// several apply (combat-action-resolution.md FR-010). It is pure and stateless; the zero value is usable.
type ResolutionPrecedence struct{}

// Preconditions applies steps 1 to 3 of the stop order for the combatants an action names, in the order
// given (the actor before the target, the helper before the patient): the encounter has ended, then a
// combatant the encounter does not hold (the problem names the first), then a dead combatant (the
// rejection names the first). It returns a nil rejection and a nil problem when none applies.
func (ResolutionPrecedence) Preconditions(e Encounter, ids ...CombatantID) (ResolutionRejection, ResolutionProblem) {
	if e.ended() {
		return EncounterStateRejection{EncounterAlreadyEndedRejection{}}, nil
	}
	for _, id := range ids {
		if _, ok := e.Find(id); !ok {
			return nil, UnknownCombatantProblem{id}
		}
	}
	for _, id := range ids {
		if c, _ := e.Find(id); c.hp.isDead() {
			return EncounterStateRejection{CombatantIsDeadRejection{id}}, nil
		}
	}
	return nil, nil
}

// DeathSavingThrow applies steps 3a, 5a and 5b for a Death Saving Throw of a combatant that passed
// Preconditions: the combatant is not Dying (NotDying), is not the active combatant
// (DeathSavingThrowOutOfTurnRejection), has already made its throw this turn
// (DeathSavingThrowAlreadyMade). It returns nil when the throw may be rolled.
func (ResolutionPrecedence) DeathSavingThrow(e Encounter, id CombatantID) ResolutionRejection {
	outOfTurn := e.tracker.order[e.tracker.active].Combatant != id
	switch r := e.PrecheckDeathSavingThrow(id).(type) {
	case nil:
		if outOfTurn {
			return DeathSavingThrowOutOfTurnRejection{}
		}
		return nil
	case DeathSavingThrowAlreadyMadeRejection:
		if outOfTurn {
			return DeathSavingThrowOutOfTurnRejection{}
		}
		return EncounterStateRejection{r}
	default:
		return EncounterStateRejection{r}
	}
}
