package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Conclude applies the caster's save to an eligible Counterspell (dispel-and-counterspell.md
// FR-033 to FR-037, FR-041, FR-042). A failed save (rolled or forced) counters the cast:
// the result is CounterspellCountered. A successful rolled save resists it:
// CounterspellResistedResult. The save must be a saving throw targeting the counterspeller's
// DC, else the result is CounterspellInvalid. A zero eligible assessment or a nil save is CounterspellInvalid with CounterspellMissingInput.
// It changes nothing and never panics.
func (Counterspell) Conclude(eligible CounterspellEligible, save CounterspellSave) CounterspellResult {
	if eligible.counterspeller.IsZero() || save == nil {
		return CounterspellInvalid{problem: CounterspellMissingInput}
	}
	switch s := save.(type) {
	case CounterspellRolledSave:
		if p, bad := saveProblem(s.check.Kind(), s.check.Target(), eligible.dc); bad {
			return CounterspellInvalid{problem: p}
		}
		if s.check.Outcome() == d20.Success {
			return resisted(eligible)
		}
		return countered(eligible)
	case CounterspellForcedFailure:
		if s.failure.Kind() != d20.SavingThrow {
			return CounterspellInvalid{problem: CounterspellNotASavingThrow}
		}
		return countered(eligible)
	default:
		panic(unreachableError("counterspell save", save))
	}
}

// saveProblem reports the first problem of a rolled save against the eligible DC.
func saveProblem(kind d20.Kind, target d20.DifficultyClass, dc character.SpellSaveDC) (CounterspellProblem, bool) {
	switch {
	case kind != d20.SavingThrow:
		return CounterspellNotASavingThrow, true
	case target != dc.DifficultyClass():
		return CounterspellSaveTargetMismatch, true
	}
	return 0, false
}

// countered builds the countered result: the event, the slot rule (FR-037), the
// Concentration rule (FR-036) and the wasted action (FR-035).
func countered(e CounterspellEligible) CounterspellCountered {
	o := e.obs
	return CounterspellCountered{
		event:              SpellCountered{counterspeller: e.counterspeller, caster: o.caster, spell: o.spell},
		slotNotExpended:    o.source == character.CastSourceSlot,
		priorConcentration: o.requiresConcentration,
		wasted:             wastedCost(o.castingTime),
	}
}

// resisted builds the resisted result and its event.
func resisted(e CounterspellEligible) CounterspellResistedResult {
	o := e.obs
	return CounterspellResistedResult{event: CounterspellResisted{counterspeller: e.counterspeller, caster: o.caster, spell: o.spell}}
}

// wastedCost is the action a cast of ct uses: the Magic action, the Bonus Action or the
// Reaction; a cast of one minute or longer wastes none.
func wastedCost(ct content.CastingTime) WastedActionCost {
	switch ct.(type) {
	case content.CastingTimeMagicAction:
		return WastedMagicAction
	case content.CastingTimeBonusAction:
		return WastedBonusAction
	case content.CastingTimeReaction:
		return WastedReaction
	case content.CastingTimeMinutes, content.CastingTimeHours:
		return WastedNone
	default:
		panic(unreachableError("casting time", ct))
	}
}
