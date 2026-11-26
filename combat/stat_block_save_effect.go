package combat

import (
	"math"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// StatBlockSaveEffectPlan is what the Saving Throw Effect Notation of a part gives the saving
// throws (SRD 5.2.1, "Saving Throw Effect Notation", p. 256): the printed ability and the printed
// Difficulty Class, unchanged, the target description and prefix verbatim, and the damage
// instances of the Failure clause. Immutable; the zero value is invalid. Safe for concurrent use.
type StatBlockSaveEffectPlan struct {
	ability ability.Ability
	dc      d20.DifficultyClass
	target  string
	prefix  string
	damage  []content.StatBlockDamage
}

// Kind returns d20.SavingThrow: the printed ability is rolled as a saving throw.
func (p StatBlockSaveEffectPlan) Kind() d20.Kind { return d20.SavingThrow }

// Ability returns the ability of the saving throw.
func (p StatBlockSaveEffectPlan) Ability() ability.Ability { return p.ability }

// DifficultyClass returns the printed DC, never recomputed from the monster's scores or
// Proficiency Bonus (FR-032).
func (p StatBlockSaveEffectPlan) DifficultyClass() d20.DifficultyClass { return p.dc }

// Target returns the printed description of the creatures that save, or "" when it is implied;
// which creatures save is the host's choice.
func (p StatBlockSaveEffectPlan) Target() string { return p.target }

// Prefix returns the printed text before the notation, or "".
func (p StatBlockSaveEffectPlan) Prefix() string { return p.prefix }

// Damage returns the damage instances of the Failure clause in printed order, as a new slice. They
// are rolled once for the whole use and shared by every target that saves (FR-033).
func (p StatBlockSaveEffectPlan) Damage() []content.StatBlockDamage { return slices.Clone(p.damage) }

// IsZero reports whether p is the invalid zero plan.
func (p StatBlockSaveEffectPlan) IsZero() bool { return !p.ability.Valid() }

// PlanStatBlockSaveEffect plans the saving throw of part (monster-stat-block-actions.md FR-032,
// FR-053).
//
// It returns an error matching srderr.ErrInvalidInput for a part with no Saving Throw notation. It
// never panics.
func PlanStatBlockSaveEffect(part content.StatBlockPart) (StatBlockSaveEffectPlan, error) {
	st, ok := part.SavingThrow()
	if !ok {
		return StatBlockSaveEffectPlan{}, invalidInput("the part has no Saving Throw notation")
	}
	plan := StatBlockSaveEffectPlan{ability: st.Ability(), dc: d20.DifficultyClass(st.DC()), target: st.Target(), prefix: st.Prefix()}
	for _, c := range part.Clauses() {
		if c.Label() == content.ClauseFailure {
			plan.damage = c.Damage()
		}
	}
	return plan, nil
}

// SaveFailureStep says which failure of a repeated save this is, for the clauses First Failure,
// Second Failure and Subsequent Failures; the host counts the failures. The zero value is
// invalid. Comparable and safe for concurrent use.
type SaveFailureStep int

// The three steps.
const (
	// SaveFirstFailure is the first failed save.
	SaveFirstFailure SaveFailureStep = iota + 1
	// SaveSecondFailure is the second failed save.
	SaveSecondFailure
	// SaveSubsequentFailures is the third and every later failed save.
	SaveSubsequentFailures
)

// String returns "first", "second", "subsequent" or "unknown".
func (s SaveFailureStep) String() string {
	return enumName([]string{"first", "second", "subsequent"}, int(s))
}

// SaveEffectDamageAdjustment is the adjustment a saving throw makes to the rolled damage, applied
// as the HalfDamage adjustment of ApplyDamage. The zero value is invalid. Comparable and safe for
// concurrent use.
type SaveEffectDamageAdjustment int

// The two adjustments.
const (
	// SaveEffectDamageUnchanged: the target takes the damage as rolled.
	SaveEffectDamageUnchanged SaveEffectDamageAdjustment = iota + 1
	// SaveEffectDamageHalved: the target takes half the damage, rounded down (SRD 5.2.1, "Half Damage", p. 16).
	SaveEffectDamageHalved
)

// String returns "none", "halve" or "unknown".
func (a SaveEffectDamageAdjustment) String() string {
	return enumName([]string{"none", "halve"}, int(a))
}

// StatBlockSaveRefusal is why a target's save could not be used. The zero value is invalid.
// Comparable and safe for concurrent use.
type StatBlockSaveRefusal int

// The two refusals.
const (
	// StatBlockSaveMissing: a target that is not immune has no CheckResult.
	StatBlockSaveMissing StatBlockSaveRefusal = iota + 1
	// StatBlockSaveNotThisSave: the CheckResult is not a saving throw, or its target is not the printed DC.
	StatBlockSaveNotThisSave
)

// String returns "missing_save", "not_this_save" or "unknown".
func (r StatBlockSaveRefusal) String() string {
	return enumName([]string{"missing_save", "not_this_save"}, int(r))
}

// StatBlockSaveTargetResult is what a saving throw effect does to one target, a closed set.
// Variants: StatBlockSaveEffectApplies, StatBlockSaveTargetImmune, StatBlockSaveTargetRefused.
type StatBlockSaveTargetResult interface {
	isStatBlockSaveTargetResult()
	// Combatants returns a new slice holding the target the result is about.
	Combatants() []CombatantID
}

// StatBlockSaveEffectApplies: the effect applies to the target as the clauses say. The clauses are
// the part's, in printed order, and their text is never parsed; the host reads it and applies what
// it names (a condition, a push) with the rules that own them.
type StatBlockSaveEffectApplies struct {
	target     CombatantID
	clauses    []content.EffectClause
	adjustment SaveEffectDamageAdjustment
	grant      SaveImmunityWindow
}

// Combatants returns a new slice holding the target the result is about.
func (r StatBlockSaveEffectApplies) Combatants() []CombatantID { return []CombatantID{r.target} }

// Clauses returns the clauses that apply, in printed order, as a new slice.
func (r StatBlockSaveEffectApplies) Clauses() []content.EffectClause { return slices.Clone(r.clauses) }

// DamageAdjustment returns SaveEffectDamageHalved when a success halves the damage and
// SaveEffectDamageUnchanged otherwise.
func (r StatBlockSaveEffectApplies) DamageAdjustment() SaveEffectDamageAdjustment {
	return r.adjustment
}

// Grant returns the immunity window a success earned, and false when it earned none. Pass it to
// SaveImmunities.Grant.
func (r StatBlockSaveEffectApplies) Grant() (SaveImmunityWindow, bool) {
	return r.grant, !r.grant.IsZero()
}

// StatBlockSaveTargetImmune: the target is immune to this part of this creature and does not
// save; nothing applies.
type StatBlockSaveTargetImmune struct{ target CombatantID }

// Combatants returns a new slice holding the target the result is about.
func (r StatBlockSaveTargetImmune) Combatants() []CombatantID { return []CombatantID{r.target} }

// StatBlockSaveTargetRefused: the target's save could not be used, for the reason named; nothing
// applies to it and the other targets are not affected.
type StatBlockSaveTargetRefused struct {
	target CombatantID
	reason StatBlockSaveRefusal
}

// Combatants returns a new slice holding the target the result is about.
func (r StatBlockSaveTargetRefused) Combatants() []CombatantID { return []CombatantID{r.target} }

// Reason returns why the save could not be used.
func (r StatBlockSaveTargetRefused) Reason() StatBlockSaveRefusal { return r.reason }

func (StatBlockSaveEffectApplies) isStatBlockSaveTargetResult() {}
func (StatBlockSaveTargetImmune) isStatBlockSaveTargetResult()  {}
func (StatBlockSaveTargetRefused) isStatBlockSaveTargetResult() {}

// StatBlockSaveEffectInput is the input of ResolveStatBlockSaveEffect. Construct it with a keyed
// literal. The function copies what it needs and changes nothing in the input.
type StatBlockSaveEffectInput struct {
	// Part is the part with the Saving Throw notation.
	Part content.StatBlockPart
	// Ref is the position of Part in the stat block, which names the part in an immunity window.
	Ref LimitedUsePartRef
	// Source is the creature using the part.
	Source CombatantID
	// Targets are the creatures that make the save, in the order of the results; each once.
	Targets []CombatantID
	// Saves are the rolled saving throws of the targets, each rolled against the printed DC.
	Saves map[CombatantID]d20.CheckResult
	// Steps say which failure a failed save is, for the clauses that depend on it. A step for a
	// target that succeeded, or has no save, is ignored.
	Steps map[CombatantID]SaveFailureStep
	// Immunities are the immunity windows held now.
	Immunities SaveImmunities
}

// maxImmunityHours is the most hours a duration holds.
const maxImmunityHours = math.MaxInt64 / int64(time.Hour)

// successRider returns the rider of the Success clause of part, or nil.
func successRider(part content.StatBlockPart) content.SaveSuccessRider {
	for _, c := range part.Clauses() {
		if r, ok := c.SuccessRider(); ok && c.Label() == content.ClauseSuccess {
			return r
		}
	}
	return nil
}

// saveClauses returns the clauses of part that apply to a save, in printed order.
func saveClauses(part content.StatBlockPart, failed, byFive bool, step SaveFailureStep) []content.EffectClause {
	var out []content.EffectClause
	for _, c := range part.Clauses() {
		var applies bool
		switch c.Label() {
		case content.ClauseHit, content.ClauseMiss, content.ClauseHitOrMiss:
			// The clauses of an attack are no part of a saving throw effect.
		case content.ClauseFailureOrSuccess:
			applies = true
		case content.ClauseFailure:
			applies = failed
		case content.ClauseSuccess:
			applies = !failed
		case content.ClauseFailureByFiveOrMore:
			applies = failed && byFive
		case content.ClauseFirstFailure:
			applies = failed && step == SaveFirstFailure
		case content.ClauseSecondFailure:
			applies = failed && step == SaveSecondFailure
		case content.ClauseSubsequentFailures:
			applies = failed && step == SaveSubsequentFailures
		default:
		}
		if applies {
			out = append(out, c)
		}
	}
	return out
}

// ResolveStatBlockSaveEffect says what the part does to each target, given the targets' saving
// throws (monster-stat-block-actions.md FR-034 to FR-039, FR-053 to FR-056; SRD 5.2.1, "Saving
// Throw Effect Notation", p. 256; "Half Damage", p. 16). Results come in the order of Targets.
//
// A target immune to this source and part (Immunities.IsImmune) does not save: its result is
// StatBlockSaveTargetImmune and any CheckResult or step for it is ignored. A target without a
// CheckResult, or whose CheckResult is not a saving throw against the printed DC, is refused. For
// the rest, a failed save applies the Failure and Failure or Success clauses, the Failure by 5 or
// More clause when the total is at most the DC minus 5, and the First, Second or Subsequent Failure
// clause the host's step names; the damage is unchanged. A success applies the Success and Failure
// or Success clauses and never the Failure clause. A Success clause whose rider is HalfDamageOnly or
// HalfDamage halves the damage; ImmuneToPartForHours grants a window of that many hours to the
// target against the Source and Ref (none when the target is the Source). The text of a clause is
// never parsed. Where a Failure by 5 or More clause and a Failure clause conflict, both are
// returned in printed order and the by-five clause is the worse outcome that governs (Q-10).
//
// It returns an error matching srderr.ErrInvalidInput for a Part with no Saving Throw notation, a
// zero Ref or Source, a zero or repeated target, an undefined step, and an immunity of more hours
// than a duration holds, and a LimitExceededError (plan P10) for more than 256 targets. It never panics.
func ResolveStatBlockSaveEffect(in StatBlockSaveEffectInput) ([]StatBlockSaveTargetResult, error) {
	plan, err := PlanStatBlockSaveEffect(in.Part)
	if err != nil {
		return nil, err
	}
	if in.Ref.IsZero() || in.Source.IsZero() {
		return nil, invalidInput("a saving throw effect needs the part's position and its source")
	}
	if err := overCap("stat block save targets", len(in.Targets), maxStatBlockTargets); err != nil {
		return nil, err
	}
	for i, t := range in.Targets {
		if t.IsZero() || slices.Contains(in.Targets[:i], t) {
			return nil, invalidInput("targets must be distinct creatures")
		}
	}
	for _, s := range in.Steps {
		if s < SaveFirstFailure || s > SaveSubsequentFailures {
			return nil, invalidInput("undefined failure step")
		}
	}
	rider := successRider(in.Part)
	hours := time.Duration(0)
	if r, ok := rider.(content.ImmuneToPartForHours); ok {
		if int64(r.Hours()) > maxImmunityHours {
			return nil, invalidInput("an immunity of more hours than a duration holds")
		}
		hours = time.Duration(r.Hours()) * time.Hour
	}
	out := make([]StatBlockSaveTargetResult, len(in.Targets))
	for i, t := range in.Targets {
		save, rolled := in.Saves[t]
		switch {
		case in.Immunities.IsImmune(t, in.Source, in.Ref):
			out[i] = StatBlockSaveTargetImmune{t}
		case !rolled:
			out[i] = StatBlockSaveTargetRefused{t, StatBlockSaveMissing}
		case save.Kind() != d20.SavingThrow || save.Target() != plan.dc:
			out[i] = StatBlockSaveTargetRefused{t, StatBlockSaveNotThisSave}
		default:
			failed := save.Outcome() == d20.Failure
			applies := StatBlockSaveEffectApplies{target: t, adjustment: SaveEffectDamageUnchanged,
				clauses: saveClauses(in.Part, failed, save.Total() <= int(plan.dc)-5, in.Steps[t])}
			switch rider.(type) {
			case content.HalfDamageOnly, content.HalfDamage:
				if !failed {
					applies.adjustment = SaveEffectDamageHalved
				}
			case content.ImmuneToPartForHours:
				if !failed && t != in.Source {
					applies.grant, _ = NewSaveImmunityWindow(t, in.Source, in.Ref, hours)
				}
			default:
			}
			out[i] = applies
		}
	}
	return out, nil
}
