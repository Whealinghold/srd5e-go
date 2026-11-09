package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ritualExtraMinutes is the time a Ritual adds to the casting time (SRD 5.2.1,
// "Casting without Slots", p. 104).
const ritualExtraMinutes = 10

// castOutcome is the closed internal outcome of a cast or a completion. Variants:
// castPerformed, castBegun, castRefused.
type castOutcome interface{ isCastOutcome() }

// castPerformed: the spell took effect.
type castPerformed struct {
	report CastReport
	state  SpellcastingState
	events []Event
}

// castBegun: a longer casting started.
type castBegun struct {
	state  SpellcastingState
	events []Event
}

// castRefused: a rule refused the cast; the state is unchanged.
type castRefused struct{ rejection CastRejection }

func (castPerformed) isCastOutcome() {}
func (castBegun) isCastOutcome()     {}
func (castRefused) isCastOutcome()   {}

// castInput is everything a cast reads. Optional fields are zero when absent.
type castInput struct {
	state        SpellcastingState
	class        SpellcastingClass
	stats        *SpellcastingStats
	spell        content.Spell
	mode         CastMode
	slot         SpellSlotChoice
	timeMode     content.CastingTimeMode
	ctx          CastingContext
	terms        *SpellCastTerms
	featureLevel int
	countsAsSlot bool
}

// completeInput is everything a completion reads.
type completeInput struct {
	state        SpellcastingState
	spell        content.Spell
	ctx          CastingContext
	slot         SpellSlotChoice
	stats        *SpellcastingStats
	terms        *SpellCastTerms
	featureLevel int
	countsAsSlot bool
}

// spellCasting is the stateless domain service for casting a spell and
// completing a longer casting (SRD 5.2.1, "Casting Spells", pp. 104-106).
type spellCasting struct{}

// castSourceFor derives the source from the mode and the spell's level.
func castSourceFor(mode CastMode, spell content.Spell) CastSource {
	switch {
	case mode == CastModeRitual:
		return CastSourceRitual
	case mode == CastModeFeatureGranted:
		return CastSourceFeatureGranted
	case mode == CastModeSpellScroll:
		return CastSourceScroll
	case mode == CastModeDuplicated:
		return CastSourceDuplicated
	case spell.Level().IsCantrip():
		return CastSourceCantrip
	}
	return CastSourceSlot
}

// chosenCastingTime returns the casting time in force and the mode chosen (the
// zero mode for a spell with one time). It returns an error for a mode the
// spell does not list.
func chosenCastingTime(spell content.Spell, mode content.CastingTimeMode) (content.CastingTime, content.CastingTimeMode, error) {
	switch t := spell.CastingTimes().(type) {
	case content.SpellCastingTimesSingle:
		if mode != (content.CastingTimeMode{}) {
			return nil, content.CastingTimeMode{}, invalidInput("cast: a casting time mode for a spell with one casting time")
		}
		return t.Time(), content.CastingTimeMode{}, nil
	case content.SpellCastingTimesModes:
		modes := t.Modes()
		if mode == (content.CastingTimeMode{}) {
			mode = modes[0]
		}
		if !slices.Contains(modes, mode) {
			return nil, content.CastingTimeMode{}, invalidInput("cast: casting time mode not among the spell's modes")
		}
		return mode.Time(), mode, nil
	}
	return nil, content.CastingTimeMode{}, invalidInput("cast: spell has no casting time")
}

// castingMinutes is the whole minutes of a casting time; an action counts as 0.
func castingMinutes(t content.CastingTime) int {
	switch v := t.(type) {
	case content.CastingTimeMinutes:
		return v.Amount()
	case content.CastingTimeHours:
		return 60 * v.Amount()
	}
	return 0
}

// hasTrigger reports whether the casting time defines a trigger (Reaction, or a
// Bonus Action that carries one).
func hasTrigger(t content.CastingTime) bool {
	switch v := t.(type) {
	case content.CastingTimeReaction:
		return true
	case content.CastingTimeBonusAction:
		_, ok := v.Trigger()
		return ok
	}
	return false
}

// validateStatsAndTerms checks the optional stats and terms against the class
// and the source; classless is true when no casting class is given.
func validateStatsAndTerms(class SpellcastingClass, source CastSource, spell content.Spell, stats *SpellcastingStats, terms *SpellCastTerms) error {
	classless := class == 0
	switch source {
	case CastSourceScroll, CastSourceDuplicated:
		if !classless || stats != nil || terms == nil || terms.IsZero() {
			return invalidInput("cast: a scroll or duplicated cast takes terms and no class or stats")
		}
		if spell.Level().IsCantrip() != (terms.cantripLevel > 0) {
			return invalidInput("cast: a cantrip scaling level is required for a cantrip and only for a cantrip")
		}
		return nil
	case CastSourceFeatureGranted:
		if classless && stats == nil {
			return invalidInput("cast: a feature-granted cast with no class needs stats")
		}
	case CastSourceSlot, CastSourceCantrip, CastSourceRitual:
		if classless {
			return invalidInput("cast: a casting class is required")
		}
	}
	if terms != nil {
		return invalidInput("cast: terms are for a scroll or duplicated cast only")
	}
	if stats != nil && !classless && stats.ability != class.Ability() {
		return invalidInput("cast: stats of another ability than the casting class")
	}
	if stats != nil && classless && stats.ability == 0 {
		return invalidInput("cast: zero stats")
	}
	return nil
}

// validateFeatureOptions checks the feature cast level and the slot-spell flag.
func validateFeatureOptions(source CastSource, spell content.Spell, featureLevel int, countsAsSlot bool) error {
	if featureLevel == 0 && !countsAsSlot {
		return nil
	}
	cantrip := spell.Level().IsCantrip()
	if source != CastSourceFeatureGranted || featureLevel < 0 || featureLevel > 9 || (countsAsSlot && cantrip) ||
		(featureLevel != 0 && (cantrip || featureLevel < spell.Level().Int())) {
		return invalidInput("cast: a feature cast level or slot-spell flag outside a valid feature-granted cast")
	}
	return nil
}

// validateSlotChoice checks that the slot choice suits the mode and the spell.
func validateSlotChoice(mode CastMode, spell content.Spell, slot SpellSlotChoice) error {
	needs := mode == CastModeNormal && !spell.Level().IsCantrip()
	if !needs {
		if slot != nil {
			return invalidInput("cast: a slot choice for a cast that spends no slot")
		}
		return nil
	}
	switch c := slot.(type) {
	case SpellSlotChoiceSpellcasting:
		if c.level.IsZero() {
			return invalidInput("cast: zero slot level")
		}
	case SpellSlotChoicePactMagic:
	default:
		return invalidInput("cast: a slot spell needs a slot choice")
	}
	return nil
}

// validate checks the input and returns the chosen casting time and mode.
func (in castInput) validate() (content.CastingTime, content.CastingTimeMode, error) {
	cantrip := in.spell.Level().IsCantrip()
	if in.state.IsZero() || !in.mode.Valid() || !in.spell.ID().Valid() || (in.class != 0 && !in.class.Valid()) || (in.mode == CastModeRitual && cantrip) {
		return nil, content.CastingTimeMode{}, invalidInput("cast: zero state, invalid mode, spell or class, or a Ritual cantrip")
	}
	source := castSourceFor(in.mode, in.spell)
	if in.class != 0 {
		if _, held := in.state.Class(in.class); !held {
			return nil, content.CastingTimeMode{}, ErrClassNotHeld
		}
	}
	if err := validateStatsAndTerms(in.class, source, in.spell, in.stats, in.terms); err != nil {
		return nil, content.CastingTimeMode{}, err
	}
	if err := validateFeatureOptions(source, in.spell, in.featureLevel, in.countsAsSlot); err != nil {
		return nil, content.CastingTimeMode{}, err
	}
	if err := validateSlotChoice(in.mode, in.spell, in.slot); err != nil {
		return nil, content.CastingTimeMode{}, err
	}
	if in.mode == CastModeDuplicated && in.timeMode != (content.CastingTimeMode{}) {
		return nil, content.CastingTimeMode{}, invalidInput("cast: a casting time mode for a duplicated cast")
	}
	return chosenCastingTime(in.spell, in.timeMode)
}

// materialSatisfied applies the Material component rules: the actual materials
// and a free hand, the Component Pouch, or a held Spellcasting Focus (SRD 5.2.1,
// "Material (M)", pp. 105-106). The pouch and the focus do not serve a
// component the spell consumes or gives a cost.
func materialSatisfied(m content.MaterialComponent, f CastingFacts, class SpellcastingClass) bool {
	if f.HasMaterials && f.HasFreeHand {
		return true
	}
	if m.ConsumesMaterial() || len(m.CostsGP()) > 0 {
		return false
	}
	if f.HasComponentPouch && f.HasFreeHand {
		return true
	}
	return f.HeldFocus.Valid() && (slices.Contains(class.FocusKinds(), f.HeldFocus) || slices.Contains(f.FeatureFocusKinds, f.HeldFocus))
}

// componentReasons returns the missing-component reasons; a scroll needs no
// Material component ("Spell Scroll", p. 244).
func componentReasons(spell content.Spell, f CastingFacts, class SpellcastingClass, scroll bool) []CastRejectionKind {
	var out []CastRejectionKind
	c := spell.Components()
	if c.Verbal() && !f.CanSpeak {
		out = append(out, CastRejectionMissingVerbal)
	}
	if c.Somatic() && !f.CanMakeGestures {
		out = append(out, CastRejectionMissingSomatic)
	}
	if m, ok := c.Material(); ok && !scroll && !materialSatisfied(m, f, class) {
		out = append(out, CastRejectionMissingMaterial)
	}
	return out
}

// slotReasons checks the chosen slot against the spell's level.
func slotReasons(state SpellcastingState, choice SpellSlotChoice, spellLevel int) []CastRejectionKind {
	var level, remaining int
	switch c := choice.(type) {
	case SpellSlotChoiceSpellcasting:
		level, remaining = c.level.n, state.Slots(PoolSpellcasting).Remaining(c.level)
	case SpellSlotChoicePactMagic:
		level, remaining = state.pact.level, 0
		if state.pact.HasPactMagic() {
			remaining = state.pact.rem
		}
	}
	var out []CastRejectionKind
	if level > 0 && level < spellLevel {
		out = append(out, CastRejectionSlotLevelTooLow)
	}
	if remaining == 0 {
		out = append(out, CastRejectionNoSlotAvailable)
	}
	return out
}

// slotOf returns the pool and level a choice names for state.
func slotOf(state SpellcastingState, choice SpellSlotChoice) (SpellSlotPool, SlotLevel) {
	if c, ok := choice.(SpellSlotChoiceSpellcasting); ok {
		return PoolSpellcasting, c.level
	}
	return PoolPactMagic, state.pact.Level()
}

// listReasons applies the known, prepared and Ritual Adept checks of a normal or
// Ritual cast by class (SRD 5.2.1, "Gaining Spells", p. 104; Wizard "Ritual
// Adept", p. 78). readable is CastingFacts.CanReadSpellbook.
func listReasons(state SpellcastingState, class SpellcastingClass, spell content.Spell, mode CastMode, readable bool) []CastRejectionKind {
	cs, _ := state.Class(class)
	id := spell.ID()
	switch {
	case mode == CastModeRitual:
		var out []CastRejectionKind
		if !spell.Ritual() {
			out = append(out, CastRejectionNotARitual)
		}
		switch {
		case cs.Prepared().IsPrepared(id):
		case class != CasterWizard:
			out = append(out, CastRejectionNotPrepared)
		case !cs.Spellbook().Contains(id):
			out = append(out, CastRejectionNotInSpellbook)
		case !readable:
			out = append(out, CastRejectionSpellbookNotReadable)
		}
		return out
	case spell.Level().IsCantrip():
		if !slices.Contains(cs.Cantrips().Spells(), id) && !slices.ContainsFunc(cs.GrantedCantrips(), func(g GrantedCantrip) bool { return g.Spell() == id }) {
			return []CastRejectionKind{CastRejectionNotKnown}
		}
	case !cs.Prepared().IsPrepared(id):
		return []CastRejectionKind{CastRejectionNotPrepared}
	}
	return nil
}

// rejections returns every reason the cast is refused, unsorted.
func (in castInput) rejections(t content.CastingTime, source CastSource) []CastRejectionKind {
	f := in.ctx.f
	var out []CastRejectionKind
	if !f.CanCastSpells {
		out = append(out, CastRejectionCannotCastSpells)
	}
	if slices.Contains(f.UnavailableSpells, in.spell.ID()) {
		out = append(out, CastRejectionSpellUnavailable)
	}
	if f.UntrainedArmorWorn {
		out = append(out, CastRejectionUntrainedArmor)
	}
	if _, going := in.state.CastingInProgress(); going {
		out = append(out, CastRejectionCastingAlreadyInProgress)
	}
	if in.mode == CastModeNormal || in.mode == CastModeRitual {
		out = append(out, listReasons(in.state, in.class, in.spell, in.mode, f.CanReadSpellbook)...)
	}
	if hasTrigger(t) && !f.TriggerMet {
		out = append(out, CastRejectionTriggerNotMet)
	}
	out = append(out, componentReasons(in.spell, f, in.class, source == CastSourceScroll)...)
	if (source == CastSourceSlot || (source == CastSourceFeatureGranted && in.countsAsSlot)) && f.SlotSpentThisTurn {
		out = append(out, CastRejectionSlotAlreadySpentThisTurn)
	}
	if source == CastSourceSlot {
		out = append(out, slotReasons(in.state, in.slot, in.spell.Level().Int())...)
	}
	return out
}

// castLevels returns the level a cast is made at and the level it scales with; 0 is
// absent.
func castLevels(state SpellcastingState, source CastSource, spell content.Spell, slotLevel, featureLevel int, terms *SpellCastTerms) (castAt, scaling int) {
	own := spell.Level().Int()
	switch source {
	case CastSourceCantrip:
		return 0, state.levels.TotalLevel().Value()
	case CastSourceSlot:
		return slotLevel, slotLevel
	case CastSourceFeatureGranted:
		if own == 0 {
			return 0, state.levels.TotalLevel().Value()
		}
		if featureLevel > 0 {
			return featureLevel, featureLevel
		}
		return own, own
	case CastSourceScroll, CastSourceDuplicated:
		if own == 0 {
			return 0, terms.cantripLevel
		}
		return own, own
	case CastSourceRitual:
		return 0, 0
	}
	return 0, 0
}

// figures returns the save DC and attack bonus of the stats or the terms.
func figures(stats *SpellcastingStats, terms *SpellCastTerms) (*SpellSaveDC, *SpellAttackBonus) {
	switch {
	case terms != nil:
		dc, bonus := dcOf(terms.saveDC), SpellAttackBonus{n: terms.attackBonus}
		return &dc, &bonus
	case stats != nil:
		dc, bonus := stats.SaveDC(), stats.AttackBonus()
		return &dc, &bonus
	}
	return nil, nil
}

// report builds the CastReport and the SpellCast event of a cast that took effect.
func report(state SpellcastingState, class SpellcastingClass, source CastSource, spell content.Spell, slotPool SpellSlotPool, slotLevel SlotLevel,
	featureLevel int, countsAsSlot bool, stats *SpellcastingStats, terms *SpellCastTerms) (CastReport, SpellCast) {
	castAt, scaling := castLevels(state, source, spell, slotLevel.n, featureLevel, terms)
	dc, bonus := figures(stats, terms)
	r := CastReport{spell: spell.ID(), class: class, source: source, pool: slotPool, slotLevel: slotLevel, castAt: castAt, scaling: scaling,
		hasAttack: bonus != nil, countsSlot: source == CastSourceSlot || (source == CastSourceFeatureGranted && countsAsSlot)}
	if dc != nil {
		r.saveDC, r.attack = *dc, *bonus
	}
	return r, SpellCast{id: state.id, spell: spell.ID(), class: class, source: source, castAt: castAt, scalingLevel: scaling}
}

// cast performs a cast (SRD 5.2.1, "Casting Spells", pp. 104-106). A rule
// refusal is a castRefused outcome; contradictory input is an error matching
// srderr.ErrInvalidInput.
func (spellCasting) cast(in castInput) (castOutcome, error) {
	t, mode, err := in.validate()
	if err != nil {
		return nil, err
	}
	source := castSourceFor(in.mode, in.spell)
	if source != CastSourceDuplicated {
		if r := in.rejections(t, source); len(r) > 0 {
			return castRefused{newCastRejection(r)}, nil
		}
	}
	minutes := castingMinutes(t)
	if source == CastSourceRitual {
		minutes += ritualExtraMinutes
	}
	if source != CastSourceDuplicated && minutes >= 1 {
		return beginLongerCasting(in, source, mode, minutes)
	}
	state, events := in.state, []Event(nil)
	var pool SpellSlotPool
	var level SlotLevel
	if source == CastSourceSlot {
		pool, level = slotOf(state, in.slot)
		state, events = spendSlot(state, pool, level, events)
	}
	state, events = beginSpellConcentration(state, in.spell, events)
	r, ev := report(state, in.class, source, in.spell, pool, level, in.featureLevel, in.countsAsSlot, in.stats, in.terms)
	return castPerformed{report: r, state: state, events: append(events, ev)}, nil
}

// spendSlot expends the slot, which the checks have shown to exist.
func spendSlot(state SpellcastingState, pool SpellSlotPool, level SlotLevel, events []Event) (SpellcastingState, []Event) {
	switch r := state.ExpendSlot(pool, level).(type) {
	case SlotExpenditureExpended:
		return r.state, append(events, r.event)
	default:
		panic(invalidInput("cast: slot checked but not expendable"))
	}
}

// beginSpellConcentration starts Concentration on an immediate cast of a
// Concentration spell, ending any earlier one first (SRD 5.2.1, "Concentration",
// p. 179).
func beginSpellConcentration(state SpellcastingState, spell content.Spell, events []Event) (SpellcastingState, []Event) {
	if !spell.RequiresConcentration() {
		return state, events
	}
	// beginConcentration also fails a casting in progress (a duplicated cast skips the rejections).
	next, evs, _ := beginConcentration(state, ConcentrationSubjectSpell{spell: spell.ID()})
	for _, e := range evs {
		events = append(events, e)
	}
	return next, events
}

// beginLongerCasting starts a casting of a minute or more: no slot is spent, the
// earlier Concentration ends and the caster concentrates on the casting (SRD
// 5.2.1, "Longer Casting Times", p. 105; the timing is a package decision).
func beginLongerCasting(in castInput, source CastSource, mode content.CastingTimeMode, minutes int) (castOutcome, error) {
	var slot SpellSlotChoice
	if source == CastSourceSlot {
		slot = in.slot
	}
	c, err := NewCastingInProgress(in.spell.ID(), source, in.class, mode, slot, minutes)
	if err != nil {
		return nil, err
	}
	state := in.state
	var events []Event
	if prior, ok := state.Concentration(); ok {
		events = append(events, ConcentrationEnded{id: state.id, subject: prior.Subject(), reason: EndedAnotherEffect})
	}
	subject := ConcentrationSubjectSpell{spell: in.spell.ID()}
	state.conc, state.casting = Concentration{subject: subject, kind: ConcentrationCasting}, c
	return castBegun{state: state, events: append(events, ConcentrationBegan{id: state.id, subject: subject, kind: ConcentrationCasting})}, nil
}

// complete finishes the casting in progress (SRD 5.2.1, "Longer Casting Times",
// p. 105). Nothing to complete is a castRefused outcome; a spell other than the
// one being cast, or contradictory input, is an error matching
// srderr.ErrInvalidInput.
func (spellCasting) complete(in completeInput) (castOutcome, error) {
	if in.state.IsZero() || !in.spell.ID().Valid() {
		return nil, invalidInput("complete casting: zero state or spell")
	}
	c, going := in.state.CastingInProgress()
	if !going {
		return castRefused{newCastRejection([]CastRejectionKind{CastRejectionCastingNotInProgress})}, nil
	}
	if in.spell.ID() != c.Spell() {
		return nil, invalidInput("complete casting: a spell other than the one being cast")
	}
	source := c.Source()
	if err := in.validate(c); err != nil {
		return nil, err
	}
	slot := c.Slot()
	if in.slot != nil {
		slot = in.slot
	}
	if r := in.rejections(c, slot); len(r) > 0 {
		return castRefused{newCastRejection(r)}, nil
	}
	state, events := in.state, []Event(nil)
	state.casting = CastingInProgress{}
	var pool SpellSlotPool
	var level SlotLevel
	if source == CastSourceSlot {
		pool, level = slotOf(state, slot)
		state, events = spendSlot(state, pool, level, events)
	}
	state, events = finishConcentration(state, in.spell, events)
	r, ev := report(state, c.Class(), source, in.spell, pool, level, in.featureLevel, in.countsAsSlot, in.stats, in.terms)
	return castPerformed{report: r, state: state, events: append(events, ev)}, nil
}

// validate checks the completion's optional inputs against the casting.
func (in completeInput) validate(c CastingInProgress) error {
	source := c.Source()
	if in.slot != nil && (source != CastSourceSlot || validateSlotChoice(CastModeNormal, in.spell, in.slot) != nil) {
		return invalidInput("complete casting: a replacement slot choice for a casting with no slot or an invalid choice")
	}
	if err := validateStatsAndTerms(c.Class(), source, in.spell, in.stats, in.terms); err != nil {
		return err
	}
	return validateFeatureOptions(source, in.spell, in.featureLevel, in.countsAsSlot)
}

// rejections re-checks at completion what may have changed since the start.
func (in completeInput) rejections(c CastingInProgress, slot SpellSlotChoice) []CastRejectionKind {
	f := in.ctx.f
	source := c.Source()
	var out []CastRejectionKind
	if slices.Contains(f.UnavailableSpells, in.spell.ID()) {
		out = append(out, CastRejectionSpellUnavailable)
	}
	if f.UntrainedArmorWorn {
		out = append(out, CastRejectionUntrainedArmor)
	}
	if source == CastSourceRitual && c.Class() == CasterWizard && !f.CanReadSpellbook {
		cs, _ := in.state.Class(CasterWizard)
		if !cs.Prepared().IsPrepared(in.spell.ID()) {
			out = append(out, CastRejectionSpellbookNotReadable)
		}
	}
	out = append(out, componentReasons(in.spell, f, c.Class(), source == CastSourceScroll)...)
	if (source == CastSourceSlot || (source == CastSourceFeatureGranted && in.countsAsSlot)) && f.SlotSpentThisTurn {
		out = append(out, CastRejectionSlotAlreadySpentThisTurn)
	}
	if source == CastSourceSlot {
		out = append(out, slotReasons(in.state, slot, in.spell.Level().Int())...)
	}
	return out
}

// finishConcentration settles the Concentration of a casting that completes: a
// spell that needs Concentration continues as an effect; any other ends it.
func finishConcentration(state SpellcastingState, spell content.Spell, events []Event) (SpellcastingState, []Event) {
	conc, ok := state.Concentration()
	if !ok || conc.Kind() != ConcentrationCasting {
		return state, events
	}
	if spell.RequiresConcentration() {
		state.conc = Concentration{subject: conc.Subject(), kind: ConcentrationEffect}
		return state, events
	}
	state.conc = Concentration{}
	return state, append(events, ConcentrationEnded{id: state.id, subject: conc.Subject(), reason: EndedCastingCompleted})
}
