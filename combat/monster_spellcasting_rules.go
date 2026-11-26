package combat

import (
	"math"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// MonsterSpellcastingRules is the stateless domain service of a monster's
// spellcasting (SRD 5.2.1, "Spellcasting", p. 257): the spell is cast at its lowest
// possible level or the version the stat block states and never higher, the printed
// save DC and spell attack bonus are used as printed, "self only", "no Concentration"
// and the components statement are honoured, a casting time of 1 minute or more needs
// the Magic action on each turn and Concentration, and per-day uses are counted
// through the LimitedUseState. It rolls no dice and has no port or use case; the
// zero value is ready to use. Safe for concurrent use.
type MonsterSpellcastingRules struct{}

// castPlan is a request resolved against the trait and the spell: what the cast is,
// at what level and with which printed numbers.
type castPlan struct {
	spell                     content.Spell
	list                      bool
	ref                       SpellUseRef
	part                      LimitedUsePartRef
	castAt, scaling           int
	spellAbility              ability.Ability
	saveDC, attackBonus       int
	hasSaveDC, hasAttackBonus bool
	components                string
	restrictions              MonsterSpellRestrictions
	note                      string
	minutes                   int
	mode                      content.CastingTimeMode
}

// Cast casts spell for the caster by req. A cast whose chosen casting time is 1
// minute or more only starts: it returns MonsterCastStarted with Concentration of kind
// casting, spends no use and leaves the finish to CompleteCasting. Any other cast
// finishes at once and returns MonsterCastPerformed with the report for the spell
// effects, the new states and the events; the use of an N/Day spell or a limited part
// is spent in the same call. targets are the creatures the host says the cast affects,
// a duplicate counting once. Every refusal returns MonsterCastRefused with the first
// reason in the order of MonsterCastRefusal, and both states are unchanged.
//
// It returns an error matching srderr.ErrInvalidInput for a caller defect: a zero or
// mismatched caster, state, limited-use state, spell or context, a nil or zero request,
// a zero target, a spell whose name differs from the entry's, a stated level below the
// spell's or any stated level on a cantrip, a casting time mode the spell does not
// have, a PartCast with no values of its own on a caster with no trait, and a casting
// time too long to count in minutes, and a LimitExceededError (plan P10) for more than 256 targets. It never panics.
func (MonsterSpellcastingRules) Cast(caster MonsterSpellcaster, state MonsterSpellcastingState, uses LimitedUseState,
	req MonsterCastRequest, spell content.Spell, targets []CombatantID, mc MonsterCastContext) (MonsterCastOutcome, error) {
	if err := checkCastInputs(caster, state, uses, spell, targets, mc); err != nil {
		return nil, err
	}
	p, refusal, err := planCast(caster, req, spell)
	if err != nil {
		return nil, err
	}
	if refusal != 0 {
		return MonsterCastRefused{refusal}, nil
	}
	if _, _, going := state.CastingInProgress(); going {
		return MonsterCastRefused{MonsterCastRefusalAlreadyInProgress}, nil
	}
	if r := judgeCast(p, caster, uses, targets, mc); r != 0 {
		return MonsterCastRefused{r}, nil
	}
	if p.minutes >= 1 {
		return startCasting(caster, state, uses, p), nil
	}
	return performCast(caster, state, uses, p), nil
}

// CompleteCasting finishes the casting in progress when its casting time has passed,
// for the same spell as the one begun. req must be a MonsterListCast of the stored
// reference. It judges again every refusal of Cast except the casting already in
// progress, spends the use once, clears the casting and returns MonsterCastPerformed.
// A spell that needs Concentration and is not waived continues as Concentration of kind
// effect; any other spell ends the casting Concentration with the reason casting
// completed. A refusal leaves the casting in progress and both states unchanged; with
// none in progress the reason is MonsterCastRefusalNotInProgress, and a request that is
// not a MonsterListCast of the stored reference is MonsterCastRefusalRefMismatch.
//
// It returns an error matching srderr.ErrInvalidInput for the caller defects of Cast
// and for a spell other than the one being cast. It never panics.
func (MonsterSpellcastingRules) CompleteCasting(caster MonsterSpellcaster, state MonsterSpellcastingState, uses LimitedUseState,
	req MonsterCastRequest, spell content.Spell, targets []CombatantID, mc MonsterCastContext) (MonsterCastOutcome, error) {
	if err := checkCastInputs(caster, state, uses, spell, targets, mc); err != nil {
		return nil, err
	}
	casting, storedRef, going := state.CastingInProgress()
	if !going {
		return MonsterCastRefused{MonsterCastRefusalNotInProgress}, nil
	}
	if spell.ID() != casting.Spell() {
		return nil, invalidInput("complete casting: a spell other than the one being cast")
	}
	p, refusal, err := planCast(caster, req, spell)
	if err != nil {
		return nil, err
	}
	if refusal != 0 {
		return MonsterCastRefused{refusal}, nil
	}
	if !p.list || p.ref != storedRef {
		return MonsterCastRefused{MonsterCastRefusalRefMismatch}, nil
	}
	if r := judgeCast(p, caster, uses, targets, mc); r != 0 {
		return MonsterCastRefused{r}, nil
	}
	p.minutes = 0
	return performCast(caster, state, uses, p), nil
}

// EndMagicActionTurn is called at the end of each of the monster's turns. With a
// casting in progress and tookMagicAction false the casting fails: Concentration ends
// with the reason ended, the casting is cleared, no use is spent and the events are
// MonsterConcentrationEnded then MonsterCastingFailed. With tookMagicAction true, or
// no casting in progress, it returns the states equal and no events. The cast that
// started the casting is itself the Magic action of its turn. It never fails and never
// panics.
func (MonsterSpellcastingRules) EndMagicActionTurn(state MonsterSpellcastingState, uses LimitedUseState, tookMagicAction bool) MonsterSpellcastingTransition {
	if _, _, going := state.CastingInProgress(); tookMagicAction || !going {
		return MonsterSpellcastingTransition{state: state, limitedUse: uses}
	}
	return breakConcentration(state, uses, character.EndedByCaster)
}

// BreakConcentration ends the Concentration the monster holds for reason. With a
// casting in progress it also clears it, spends no use and emits MonsterCastingFailed.
// With no Concentration, or a reason that is not one of the five break reasons, it
// returns the states equal and no events. The Constitution saving throw after damage is
// the host's. It never fails and never panics.
func (MonsterSpellcastingRules) BreakConcentration(state MonsterSpellcastingState, uses LimitedUseState, reason character.ConcentrationBreakReason) MonsterSpellcastingTransition {
	if !reason.Valid() {
		return MonsterSpellcastingTransition{state: state, limitedUse: uses}
	}
	return breakConcentration(state, uses, character.ConcentrationEndReason(reason))
}

// breakConcentration ends the Concentration held, and the casting with it, for end.
func breakConcentration(state MonsterSpellcastingState, uses LimitedUseState, end character.ConcentrationEndReason) MonsterSpellcastingTransition {
	t := MonsterSpellcastingTransition{state: state, limitedUse: uses}
	if state.concentration.IsZero() {
		return t
	}
	base := monsterSpellcastingEventBase{state.combatant}
	t.events = []MonsterSpellcastingEvent{MonsterConcentrationEnded{base, state.concentration.Subject(), end}}
	t.state.concentration = character.Concentration{}
	if casting, _, going := state.CastingInProgress(); going {
		t.events = append(t.events, MonsterCastingFailed{base, casting.Spell(), end})
		t.state.casting, t.state.castingRef = character.CastingInProgress{}, SpellUseRef{}
	}
	return t
}

// checkCastInputs returns the caller defects common to Cast and CompleteCasting.
func checkCastInputs(caster MonsterSpellcaster, state MonsterSpellcastingState, uses LimitedUseState, spell content.Spell, targets []CombatantID, mc MonsterCastContext) error {
	if err := overCap("monster cast targets", len(targets), maxStatBlockTargets); err != nil {
		return err
	}
	switch {
	case caster.IsZero() || state.combatant != caster.combatant || uses.combatant != caster.combatant: // a zero state or limited-use state has the zero combatant, so it differs from a valid caster
		return invalidInput("a monster cast needs a caster, a spellcasting state and a limited-use state of one combatant")
	case !spell.ID().Valid():
		return invalidInput("a monster cast needs a spell")
	case mc.IsZero():
		return invalidInput("a monster cast needs a context")
	case slices.Contains(targets, CombatantID("")):
		return invalidInput("a monster cast target needs a combatant id")
	}
	return nil
}

// planCast resolves req against the caster's trait and spell. A reference that names no
// spell is the refusal MonsterCastRefusalNotInSpellList; a caller defect is the error.
func planCast(caster MonsterSpellcaster, req MonsterCastRequest, spell content.Spell) (castPlan, MonsterCastRefusal, error) {
	p := castPlan{spell: spell}
	var stated int
	switch r := req.(type) {
	case MonsterListCast:
		entry, found := caster.entry(r.spell)
		if !found {
			return castPlan{}, MonsterCastRefusalNotInSpellList, nil
		}
		if entry.Name() != spell.Name() {
			return castPlan{}, 0, invalidInput("the spell's name differs from the entry's")
		}
		trait, _ := caster.Spellcasting()
		stated, _ = entry.CastAtLevel()
		p.list, p.ref, p.note, p.restrictions = true, r.spell, entry.Note(), MonsterSpellRestrictions{}.Of(entry.Note())
		p.spellAbility, p.components = trait.Ability(), trait.Components()
		p.saveDC, p.hasSaveDC = trait.SaveDC()
		p.attackBonus, p.hasAttackBonus = trait.AttackBonus()
		minutes, mode, err := chosenCasting(spell, r.mode)
		if err != nil {
			return castPlan{}, 0, err
		}
		p.minutes, p.mode = minutes, mode
	case MonsterPartCast:
		if r.part.IsZero() {
			return castPlan{}, 0, invalidInput("a part cast needs a part")
		}
		stated, p.part = r.castAtLevel, r.part
		if err := p.takePartValues(caster, r.own); err != nil {
			return castPlan{}, 0, err
		}
	default:
		return castPlan{}, 0, invalidInput("a monster cast needs a ListCast or a PartCast, not " + typeName(req))
	}
	var err error
	if p.castAt, p.scaling, err = castLevels(spell, stated); err != nil {
		return castPlan{}, 0, err
	}
	return p, 0, nil
}

// takePartValues takes the values of a part cast: its own when it prints any, else the
// trait's ability and numbers with no components statement.
func (p *castPlan) takePartValues(caster MonsterSpellcaster, own PartSpellcasting) error {
	if !own.IsZero() {
		p.spellAbility, p.components = own.Ability(), own.Components()
		p.saveDC, p.hasSaveDC = own.SaveDC()
		p.attackBonus, p.hasAttackBonus = own.AttackBonus()
		return nil
	}
	trait, ok := caster.Spellcasting()
	if !ok {
		return invalidInput("a part cast with no values of its own needs a caster with a Spellcasting trait")
	}
	p.spellAbility = trait.Ability()
	p.saveDC, p.hasSaveDC = trait.SaveDC()
	p.attackBonus, p.hasAttackBonus = trait.AttackBonus()
	return nil
}

// entry returns the list entry at ref.
func (c MonsterSpellcaster) entry(ref SpellUseRef) (content.MonsterSpellEntry, bool) {
	groups := c.trait.Groups()
	if ref.groupIndex >= len(groups) {
		return content.MonsterSpellEntry{}, false
	}
	spells := groups[ref.groupIndex].Spells()
	if ref.spellIndex >= len(spells) {
		return content.MonsterSpellEntry{}, false
	}
	return spells[ref.spellIndex], true
}

// castLevels returns the level a spell is cast at (0 for a cantrip) and the level it
// scales with: the stated version, else the spell's own level (FR-003 to FR-007).
func castLevels(spell content.Spell, stated int) (castAt, scaling int, err error) {
	level := spell.Level().Int()
	switch {
	case level == 0 && stated != 0:
		return 0, 0, invalidInput("a cantrip has no stated level")
	case level == 0:
		return 0, 1, nil
	case stated != 0 && stated < level:
		return 0, 0, invalidInput("a stated level below the spell's level")
	case stated != 0:
		return stated, stated, nil
	}
	return level, level, nil
}

// chosenCasting returns the whole minutes of the casting time in force (0 for an
// action, a Bonus Action or a Reaction) and the mode chosen. A spell with several modes
// needs one of them, and one with a single time takes none.
func chosenCasting(spell content.Spell, mode content.CastingTimeMode) (int, content.CastingTimeMode, error) {
	var t content.CastingTime
	switch times := spell.CastingTimes().(type) {
	case content.SpellCastingTimesSingle:
		if mode != (content.CastingTimeMode{}) {
			return 0, content.CastingTimeMode{}, invalidInput("a casting time mode for a spell with one casting time")
		}
		t = times.Time()
	case content.SpellCastingTimesModes:
		if !slices.Contains(times.Modes(), mode) {
			return 0, content.CastingTimeMode{}, invalidInput("a casting time mode that is not one of the spell's")
		}
		t = mode.Time()
	default:
		return 0, content.CastingTimeMode{}, invalidInput("a spell with no casting time")
	}
	switch v := t.(type) {
	case content.CastingTimeMinutes:
		return v.Amount(), mode, nil
	case content.CastingTimeHours:
		if v.Amount() > math.MaxInt/60 {
			return 0, content.CastingTimeMode{}, invalidInput("a casting time too long to count in minutes")
		}
		return 60 * v.Amount(), mode, nil
	}
	return 0, mode, nil
}

// judgeCast returns the first refusal that applies, in the order of MonsterCastRefusal,
// or 0 (FR-031). It reads the states only.
func judgeCast(p castPlan, caster MonsterSpellcaster, uses LimitedUseState, targets []CombatantID, mc MonsterCastContext) MonsterCastRefusal {
	waiver := ParseComponentWaiver(p.components)
	waived, recognised := waiver.(ComponentsWaived)
	components := p.spell.Components()
	facts := p.spell.Effects().Facts()
	switch {
	case mc.actions == ConditionActionAvailabilityNone:
		return MonsterCastRefusalCannotTakeActions
	case !recognised:
		return MonsterCastRefusalComponentsUnrecognized
	case components.Verbal() && !waived.Waives(character.ComponentVerbal) && !mc.canSpeak:
		return MonsterCastRefusalVerbalMissing
	case components.Somatic() && !waived.Waives(character.ComponentSomatic) && !mc.canMakeGestures:
		return MonsterCastRefusalSomaticMissing
	case p.restrictions.selfOnly && slices.ContainsFunc(targets, func(id CombatantID) bool { return id != caster.combatant }):
		return MonsterCastRefusalSelfOnlyViolated
	case facts.ForcesSavingThrow() == content.EffectFact(content.FactKnown{Value: true}) && !p.hasSaveDC:
		return MonsterCastRefusalMissingSaveDC
	case facts.MakesSpellAttackRoll() == content.EffectFact(content.FactKnown{Value: true}) && !p.hasAttackBonus:
		return MonsterCastRefusalMissingSpellAttackBonus
	case p.availability(uses) == UseExhausted:
		return MonsterCastRefusalUsesExhausted
	}
	return 0
}

// availability is the availability of the use the cast would spend.
func (p castPlan) availability(uses LimitedUseState) LimitedUseAvailability {
	if p.list {
		return uses.AvailabilityOfSpell(p.ref)
	}
	return uses.AvailabilityOf(p.part)
}

// spend spends the use of the cast, and returns uses unchanged and no events for an
// unlimited spell or part.
func (p castPlan) spend(uses LimitedUseState) (LimitedUseState, []MonsterLimitedUseEvent) {
	var out LimitedUseSpendOutcome
	if p.list {
		out = uses.SpendSpell(p.ref)
	} else {
		out = uses.Spend(p.part)
	}
	if spent, ok := out.(LimitedUseSpent); ok {
		return spent.Transition.State(), spent.Transition.Events()
	}
	return uses, nil
}

// report builds the report of the plan.
func (p castPlan) report(caster MonsterSpellcaster) MonsterCastReport {
	limit := TargetLimitUnrestricted
	if p.restrictions.selfOnly {
		limit = TargetLimitSelfOnly
	}
	return MonsterCastReport{spell: p.spell.ID(), castAtLevel: p.castAt, scalingLevel: p.scaling, saveDC: p.saveDC, attackBonus: p.attackBonus,
		hasSaveDC: p.hasSaveDC, hasAttackBonus: p.hasAttackBonus, spellAbility: p.spellAbility, abilityModifier: caster.AbilityModifier(p.spellAbility),
		targetLimit: limit, restrictionNote: p.note}
}

// startCasting begins a casting of 1 minute or more: Concentration in progress ends,
// Concentration of kind casting begins and no use is spent.
func startCasting(caster MonsterSpellcaster, state MonsterSpellcastingState, uses LimitedUseState, p castPlan) MonsterCastOutcome {
	base := monsterSpellcastingEventBase{caster.combatant}
	t := MonsterSpellcastingTransition{state: state, limitedUse: uses}
	if prior, held := state.Concentration(); held {
		t.events = append(t.events, MonsterConcentrationEnded{base, prior.Subject(), character.EndedAnotherEffect})
	}
	subject := character.NewConcentrationSubjectSpell(p.spell.ID())
	casting := newMonsterCasting(p)
	t.state.concentration, t.state.casting, t.state.castingRef = newMonsterConcentration(subject, character.ConcentrationCasting), casting, p.ref
	t.events = append(t.events, MonsterConcentrationBegan{base, subject, character.ConcentrationCasting}, MonsterCastingStarted{base, p.spell.ID(), p.minutes})
	return MonsterCastStarted{t}
}

// performCast finishes a cast: it settles Concentration, spends the use and reports.
func performCast(caster MonsterSpellcaster, state MonsterSpellcastingState, uses LimitedUseState, p castPlan) MonsterCastOutcome {
	base := monsterSpellcastingEventBase{caster.combatant}
	t := MonsterSpellcastingTransition{state: state}
	needs := p.spell.RequiresConcentration() && !p.restrictions.waivesConcentration
	subject := character.NewConcentrationSubjectSpell(p.spell.ID())
	if _, _, finishing := state.CastingInProgress(); finishing {
		t.state.casting, t.state.castingRef = character.CastingInProgress{}, SpellUseRef{}
		if needs {
			t.state.concentration = newMonsterConcentration(subject, character.ConcentrationEffect)
		} else {
			t.state.concentration = character.Concentration{}
			t.events = append(t.events, MonsterConcentrationEnded{base, subject, character.EndedCastingCompleted})
		}
	} else if needs {
		if prior, held := state.Concentration(); held {
			t.events = append(t.events, MonsterConcentrationEnded{base, prior.Subject(), character.EndedAnotherEffect})
		}
		t.state.concentration = newMonsterConcentration(subject, character.ConcentrationEffect)
		t.events = append(t.events, MonsterConcentrationBegan{base, subject, character.ConcentrationEffect})
	}
	t.events = append(t.events, MonsterSpellCast{base, p.spell.ID(), p.castAt, p.scaling})
	t.limitedUse, t.limitedUseEvents = p.spend(uses)
	return MonsterCastPerformed{Report: p.report(caster), Transition: t}
}

// newMonsterConcentration returns Concentration on subject of kind. The callers pass
// a spell subject and a defined kind, which character accepts.
func newMonsterConcentration(subject character.ConcentrationSubject, kind character.ConcentrationKind) character.Concentration {
	c, err := character.NewConcentration(subject, kind)
	switch err {
	case nil:
		return c
	default:
		panic(unreachableError("monster concentration", err))
	}
}

// newMonsterCasting returns the casting in progress of p: no casting class, no slot and
// the source a feature grants.
func newMonsterCasting(p castPlan) character.CastingInProgress {
	c, err := character.NewCastingInProgress(p.spell.ID(), character.CastSourceFeatureGranted, 0, p.mode, nil, p.minutes)
	switch err {
	case nil:
		return c
	default:
		panic(unreachableError("monster casting in progress", err))
	}
}
