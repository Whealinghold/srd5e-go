package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
)

// concentration is what the Concentration step of a call hands back: the outcome and, when the host gave a
// TracksConcentration, the new SpellcastingState of the damaged combatant (FR-079).
type concentration struct {
	outcome ConcentrationCheckOutcome
	state   character.SpellcastingState
	tracked bool
}

// targetSpellcasting returns the damaged combatant's new SpellcastingState and whether the host gave one.
func (c concentration) targetSpellcasting() (character.SpellcastingState, bool) {
	return c.state, c.tracked
}

// noConcentration is the step of a call that checked nothing.
func noConcentration(in ConcentrationInput) concentration {
	c := concentration{outcome: ConcentrationNotChecked{}}
	if t, ok := in.(TracksConcentration); ok {
		c.state, c.tracked = t.state, true
	}
	return c
}

// damageTakenFor returns the first DamageTaken of the events that is the target's, wrapped in the encounter's
// envelope: one applyDamage command gives one (FR-036, FR-132).
func damageTakenFor(events []Event, target CombatantID) (DamageTaken, bool) {
	for _, ev := range events {
		if h, ok := ev.(HitPointEventOccurred); ok {
			if d, ok := h.wrapped.(DamageTaken); ok && slices.Contains(d.Combatants(), target) {
				return d, true
			}
		}
	}
	return DamageTaken{}, false
}

// checkConcentration runs the Concentration step of FR-078 for the damaged combatant, on the encounter after
// the damage and its consequences were committed (FR-082). damage are the events the damage application
// returned. It rolls one Constitution saving throw at most, through the roller; a roller failure is returned
// and nothing is changed, because r is a local of the call. It adds no event: the caller appends the
// ConcentrationChecked with concentratedEvent after the prompts (FR-088).
func (u resolutionUse) checkConcentration(r *resolution, target CombatantID, in ConcentrationInput, damage []Event) (concentration, error) {
	res := noConcentration(in)
	t, tracks := in.(TracksConcentration)
	taken, hit := damageTakenFor(damage, target)
	if !tracks || !hit {
		return res, nil
	}
	c, _ := r.enc.Find(target)
	rules := character.ConcentrationRules{}
	d := (ConsequenceRules{}).Concentration(c, t, taken.Total())
	switch d.Step() {
	case ConcentrationStepBreak:
		tr := rules.BreakConcentration(t.state, d.Reason())
		out, err := NewConcentrationEndedWithoutSave(d.Reason(), tr.Events())
		if err != nil {
			return concentration{}, err
		}
		return concentration{out, tr.State(), true}, nil
	case ConcentrationStepNone:
		return res, nil
	case ConcentrationStepSave:
		dc, err := character.ConcentrationSaveDCForDamage(taken.Total())
		if err != nil {
			return concentration{}, err
		}
		save, err := u.concentrationSave(c, t, dc.DifficultyClass())
		if err != nil {
			return concentration{}, err
		}
		if save.Outcome() == d20.Success {
			return concentration{ConcentrationHeld{dc.DifficultyClass(), save}, t.state, true}, nil
		}
		tr := rules.BreakConcentration(t.state, character.BreakFailedSave)
		return concentration{ConcentrationCheckFailed{dc.DifficultyClass(), save, tr.Events()}, tr.State(), true}, nil
	}
	return res, nil
}

// concentrationSave rolls the Constitution saving throw of the damaged combatant against dc: the host's
// modifiers and circumstances, the condition inputs of its own conditions and the Exhaustion reduction
// (FR-131, FR-030, FR-045). A forced failure of the conditions rolls no die.
func (u resolutionUse) concentrationSave(c Combatant, t TracksConcentration, dc d20.DifficultyClass) (TestResolution, error) {
	adj, err := (AttackConditionInputs{}).Save(c.conditions, ability.Constitution, DodgeBenefitsNone, 0)
	if err != nil {
		return nil, err
	}
	var attempt d20.SaveAttempt = d20.RollAttempt{Modifiers: t.modifiers.WithOthers(adj.Modifiers()...),
		Circumstances: slices.Concat(t.circumstances, adj.Circumstances())}
	if _, forcedFailure := adj.Forced(); forcedFailure {
		attempt = d20.AutomaticFailure{}
	}
	res, err := u.flow.resolveSave(attempt, dc)
	if err != nil {
		return nil, err
	}
	return testResolutionOf(res), nil
}

// concentrated appends the ConcentrationChecked of the step, unless nothing was checked (FR-134).
func (r *resolution) concentrated(target CombatantID, c concentration) {
	if _, none := c.outcome.(ConcentrationNotChecked); !none {
		r.steps++
		r.add(ConcentrationChecked{r.base(), target, c.outcome})
	}
}

// validConcentrationInput returns an error matching srderr.ErrInvalidInput for a nil input or a
// TracksConcentration that is the zero value.
func validConcentrationInput(in ConcentrationInput) error {
	switch v := in.(type) {
	case NoConcentrationTracking:
		return nil
	case TracksConcentration:
		if v.state.IsZero() || !v.rule.Valid() {
			return invalidInput("tracking Concentration needs a state and a rule: use NewTracksConcentration")
		}
		return nil
	}
	return invalidInput("a Concentration input is NoConcentrationTracking or TracksConcentration")
}

// validOptionalConcentration is validConcentrationInput for a field whose nil is NoConcentrationTracking.
func validOptionalConcentration(in ConcentrationInput) error {
	if in == nil {
		return nil
	}
	return validConcentrationInput(in)
}

// concentrationOutcomeName is the value of srd5e.combat.concentration_outcome of a check that was made.
func concentrationOutcomeName(o ConcentrationCheckOutcome) string {
	switch v := o.(type) {
	case ConcentrationHeld:
		return valueConcentrationHeld
	case ConcentrationCheckFailed:
		return valueConcentrationBroken
	case ConcentrationEndedWithoutSave:
		if v.reason == character.BreakDead {
			return valueConcentrationEndedDead
		}
		return valueConcentrationEndedIncapacitated
	default:
		panic(unreachableError("Concentration outcome", o))
	}
}
