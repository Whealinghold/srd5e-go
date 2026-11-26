package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// LimitedUseState is what changes for one monster combatant about the limited use
// of its stat block (SRD 5.2.1, "Limited Usage", "Legendary Actions", p. 257):
// the uses left of each X/Day part, whether each Recharge part is ready, the
// Legendary Action uses left and the uses left of each N/Day spell. Every entry
// copies the printed numbers it needs, so no operation takes the Monster again.
// The host stores one per monster combatant; every operation returns a new state
// and leaves the receiver unchanged. It holds a slice, so compare it with Equal.
// The zero value is invalid (see IsZero); its operations refuse or change
// nothing. Safe for concurrent use.
type LimitedUseState struct {
	combatant CombatantID
	monsterID content.MonsterID
	lair      LairStatus
	entries   []LimitedUseEntry
	legendary LegendaryTrack
	window    LegendaryWindowRecord
}

// StartLimitedUse returns the state of monster m for combatant at the start of an
// encounter in lair status lair: one entry for each part with an X/Day, Recharge
// or Recharge-after-a-Rest usage, in printed order (Traits, Actions, Bonus
// Actions, Reactions, then Legendary Actions), then one for each spell of an N/Day
// or N/Day Each group of the first Spellcasting; a part with no limit, a usage the
// SRD gives no rule for and an At Will spell have none. X/Day parts start with
// their uses for lair, Recharge parts ready, and Legendary Actions with all their
// uses for lair.
//
// It returns an error matching srderr.ErrInvalidInput for a zero m, a zero
// combatant, an undefined lair or a content number outside its allowed range (a
// use count below 1 or above content.MonsterMaxNumber, a Recharge bound outside
// 1 to 6 or reversed), and a LimitExceededError (plan P10) for more than 64 parts or
// 64 spells or 64 spell groups. It never panics.
func StartLimitedUse(m content.Monster, combatant CombatantID, lair LairStatus) (LimitedUseState, error) {
	if m.ID() == "" || combatant.IsZero() || !lair.Valid() {
		return LimitedUseState{}, invalidInput("a limited-use state needs a monster, a combatant and a lair status")
	}
	if err := monsterCollectionsInRange(m); err != nil {
		return LimitedUseState{}, err
	}
	s := LimitedUseState{combatant: combatant, monsterID: m.ID(), lair: lair, legendary: LegendaryNone{}, window: LegendaryWindowNoneTaken{}}
	legendary, hasLegendary := m.LegendaryActions()
	sections := []struct {
		section MonsterPartSection
		parts   []content.StatBlockPart
	}{
		{MonsterPartTraits, m.Traits()}, {MonsterPartActions, m.Actions()}, {MonsterPartBonusActions, m.BonusActions()},
		{MonsterPartReactions, m.Reactions()}, {MonsterPartLegendaryActions, legendary.Actions()},
	}
	for _, sec := range sections {
		for i, p := range sec.parts {
			entry, limited, err := usageEntry(LimitedUsePartRef{sec.section, i}, p.Usage(), lair)
			if err != nil {
				return LimitedUseState{}, err
			}
			if limited {
				s.entries = append(s.entries, entry)
			}
		}
	}
	if sc, ok := m.Spellcasting(); ok {
		for gi, g := range sc.Groups() {
			entries, err := spellEntries(gi, g.Frequency(), len(g.Spells()))
			if err != nil {
				return LimitedUseState{}, err
			}
			s.entries = append(s.entries, entries...)
		}
	}
	if hasLegendary {
		inLair, _ := legendary.UsesInLair()
		track, err := legendaryTrack(legendary.Uses(), inLair, lair)
		if err != nil {
			return LimitedUseState{}, err
		}
		s.legendary = track
	}
	return s, nil
}

// spellEntries returns the entries of the spells of group gi, which has frequency f
// and n spells: one each with its own uses for N/Day and N/Day Each, none for At
// Will. It returns an error matching srderr.ErrInvalidInput for a use count outside
// the allowed range (plan P13).
func spellEntries(gi int, f content.SpellFrequency, n int) ([]LimitedUseEntry, error) {
	uses, limited := spellUses(f)
	if !limited {
		return nil, nil
	}
	if !usesInRange(uses) {
		return nil, invalidInput("spell frequency with a number outside the allowed range")
	}
	out := make([]LimitedUseEntry, n)
	for si := range out {
		out[si] = SpellUseEntry{SpellUseRef{gi, si}, uses, uses}
	}
	return out, nil
}

// legendaryTrack returns the full Legendary Action uses of a monster with the printed
// uses and usesInLair (0 for none printed) in status lair. It returns an error matching
// srderr.ErrInvalidInput for a use count outside the allowed range (plan P13).
func legendaryTrack(uses, usesInLair int, lair LairStatus) (LegendaryTrack, error) {
	if !usesInRange(uses) || (usesInLair != 0 && !usesInRange(usesInLair)) {
		return nil, invalidInput("Legendary Actions with a use count outside the allowed range")
	}
	maximum := legendaryMaximum(uses, usesInLair, lair)
	return LegendaryTracked{maximum, maximum, uses, usesInLair}, nil
}

// spellUses returns the casts a day of each spell of a group with frequency f, and
// false for At Will (no limit).
func spellUses(f content.SpellFrequency) (int, bool) {
	switch v := f.(type) {
	case content.SpellFrequencyPerDay:
		return v.Count(), true
	case content.SpellFrequencyPerDayEach:
		return v.Count(), true
	default:
		return 0, false
	}
}

// Combatants returns a new slice holding the monster combatant the state is for.
func (s LimitedUseState) Combatants() []CombatantID { return []CombatantID{s.combatant} }

// MonsterID returns the id of the stat block the state was started from, for the
// host's own check that a state belongs to the right monster; no operation reads it.
func (s LimitedUseState) MonsterID() content.MonsterID { return s.monsterID }

// Lair returns the lair status.
func (s LimitedUseState) Lair() LairStatus { return s.lair }

// Entries returns the entries in printed order (parts, then spells), as a new slice.
func (s LimitedUseState) Entries() []LimitedUseEntry { return slices.Clone(s.entries) }

// Legendary returns the Legendary Action uses: LegendaryNone or LegendaryTracked.
func (s LimitedUseState) Legendary() LegendaryTrack {
	if s.legendary == nil {
		return LegendaryNone{}
	}
	return s.legendary
}

// LastLegendaryWindow returns the window the last Legendary Action was taken in.
func (s LimitedUseState) LastLegendaryWindow() LegendaryWindowRecord {
	if s.window == nil {
		return LegendaryWindowNoneTaken{}
	}
	return s.window
}

// IsZero reports whether s is the invalid zero state.
func (s LimitedUseState) IsZero() bool { return s.combatant.IsZero() }

// Equal reports whether s and o hold the same values.
func (s LimitedUseState) Equal(o LimitedUseState) bool {
	return s.combatant == o.combatant && s.monsterID == o.monsterID && s.lair == o.lair &&
		slices.Equal(s.entries, o.entries) && s.Legendary() == o.Legendary() && s.LastLegendaryWindow() == o.LastLegendaryWindow()
}

// partIndex returns the index of the part entry of ref, or -1.
func (s LimitedUseState) partIndex(ref LimitedUsePartRef) int {
	return slices.IndexFunc(s.entries, func(e LimitedUseEntry) bool {
		switch v := e.(type) {
		case PerDayEntry:
			return v.part == ref
		case RechargeEntry:
			return v.part == ref
		case RestRechargeEntry:
			return v.part == ref
		}
		return false
	})
}

// spellIndex returns the index of the spell entry of ref, or -1.
func (s LimitedUseState) spellIndex(ref SpellUseRef) int {
	return slices.IndexFunc(s.entries, func(e LimitedUseEntry) bool {
		v, ok := e.(SpellUseEntry)
		return ok && v.ref == ref
	})
}

// with returns a copy of s whose entry i is e.
func (s LimitedUseState) with(i int, e LimitedUseEntry) LimitedUseState {
	s.entries = slices.Clone(s.entries)
	s.entries[i] = e
	return s
}

// spendOne spends one use of e: it returns the new entry, the uses left (0 for a
// Recharge part) and true, or e, 0 and false when no use remains or the part is
// spent.
func spendOne(e LimitedUseEntry) (LimitedUseEntry, int, bool) {
	switch v := e.(type) {
	case PerDayEntry:
		if v.left > 0 {
			v.left--
			return v, v.left, true
		}
	case RechargeEntry:
		if v.readiness == RechargeReady {
			v.readiness = RechargeSpent
			return v, 0, true
		}
	case RestRechargeEntry:
		if v.readiness == RechargeReady {
			v.readiness = RechargeSpent
			return v, 0, true
		}
	case SpellUseEntry:
		if v.remain > 0 {
			v.remain--
			return v, v.remain, true
		}
	}
	return e, 0, false
}

// availabilityAt is the availability of the entry at index i, or UseNotLimited for -1.
func (s LimitedUseState) availabilityAt(i int) LimitedUseAvailability {
	if i < 0 {
		return UseNotLimited
	}
	if _, _, ok := spendOne(s.entries[i]); !ok {
		return UseExhausted
	}
	return UseAvailable
}

// AvailabilityOf returns UseAvailable for a part with a use left or ready,
// UseExhausted for a spent or zero-use part and UseNotLimited for a reference with
// no entry (an unlimited part, a part with a usage the SRD gives no rule for, a
// position not in the stat block, or the zero reference). It never fails.
func (s LimitedUseState) AvailabilityOf(part LimitedUsePartRef) LimitedUseAvailability {
	return s.availabilityAt(s.partIndex(part))
}

// AvailabilityOfSpell is AvailabilityOf for a spell of an N/Day or N/Day Each group.
func (s LimitedUseState) AvailabilityOfSpell(spell SpellUseRef) LimitedUseAvailability {
	return s.availabilityAt(s.spellIndex(spell))
}

// RechargeRollsNeeded returns, as a new slice and in printed order, the parts that
// need a 1d6 at the start of the monster's turn: the spent Recharge X-Y parts. A
// ready part, a Recharge-after-a-Rest part, an X/Day part and a spell need none.
func (s LimitedUseState) RechargeRollsNeeded() []LimitedUsePartRef {
	var out []LimitedUsePartRef
	for _, e := range s.entries {
		if r, ok := e.(RechargeEntry); ok && r.readiness == RechargeSpent {
			out = append(out, r.part)
		}
	}
	return out
}

// spend spends the entry at index i for subject: LimitedUseSpent with one event, or the
// refusal for a missing entry (-1) or an exhausted one.
func (s LimitedUseState) spend(i int, subject LimitedUseSubject) LimitedUseSpendOutcome {
	if i < 0 {
		return LimitedUseSpendRefused{LimitedUseRefusedNotLimited}
	}
	next, left, ok := spendOne(s.entries[i])
	if !ok {
		return LimitedUseSpendRefused{LimitedUseRefusedExhausted}
	}
	return LimitedUseSpent{LimitedUseTransition{s.with(i, next),
		[]MonsterLimitedUseEvent{LimitedUsePartSpent{monsterLimitedUseEventBase{s.combatant}, subject, left}}}}
}

// Spend spends one use of an X/Day part, or the use of a ready Recharge part. It
// returns LimitedUseSpent with the entry reduced by one (a Recharge part becomes
// spent) and one LimitedUsePartSpent event, LimitedUseSpendRefused with
// LimitedUseRefusedExhausted when no use remains or the part is spent, and with
// LimitedUseRefusedNotLimited when part has no entry. A refusal changes nothing
// and has no event. It never fails.
func (s LimitedUseState) Spend(part LimitedUsePartRef) LimitedUseSpendOutcome {
	return s.spend(s.partIndex(part), PartSubject{part})
}

// SpendSpell spends one use of a spell of an N/Day or N/Day Each group, each spell
// having its own uses. It returns LimitedUseSpent with the entry reduced by one and
// one LimitedUsePartSpent event, LimitedUseSpendRefused with
// LimitedUseRefusedExhausted at 0 uses and with LimitedUseRefusedNotLimited for an
// At Will spell or a position not in the stat block. A refusal changes nothing. It
// never fails.
func (s LimitedUseState) SpendSpell(spell SpellUseRef) LimitedUseSpendOutcome {
	return s.spend(s.spellIndex(spell), SpellSubject{spell})
}

// TakeLegendaryAction takes a Legendary Action in window w, spending one use. When
// no refusal applies it returns LegendaryActionAllowed with the uses reduced by one,
// w recorded as used and one LegendaryActionTaken event. Otherwise it returns
// LegendaryActionRefused, with the state unchanged, for the first that applies of:
// no Legendary Actions, the window is the monster's own turn, availability is none,
// a Legendary Action was already taken in an equal window, no use remains.
//
// It returns an error matching srderr.ErrInvalidInput for a zero w or an
// availability that is neither of the two defined values. It never panics.
func (s LimitedUseState) TakeLegendaryAction(w LegendaryActionWindow, availability ConditionActionAvailability) (LegendaryActionOutcome, error) {
	if w.IsZero() || (availability != ConditionActionAvailabilityAvailable && availability != ConditionActionAvailabilityNone) {
		return nil, invalidInput("a Legendary Action needs a window and an action availability")
	}
	t, ok := s.Legendary().(LegendaryTracked)
	switch {
	case !ok:
		return LegendaryActionRefused{LegendaryRefusedNoActions}, nil
	case w.endedTurnOf == s.combatant:
		return LegendaryActionRefused{LegendaryRefusedOwnTurn}, nil
	case availability == ConditionActionAvailabilityNone:
		return LegendaryActionRefused{LegendaryRefusedIncapacitated}, nil
	case s.LastLegendaryWindow() == LegendaryWindowTaken{w}:
		return LegendaryActionRefused{LegendaryRefusedWindowUsed}, nil
	case t.remaining == 0:
		return LegendaryActionRefused{LegendaryRefusedNoUses}, nil
	}
	t.remaining--
	s.legendary, s.window = t, LegendaryWindowTaken{w}
	return LegendaryActionAllowed{LimitedUseTransition{s,
		[]MonsterLimitedUseEvent{LegendaryActionTaken{monsterLimitedUseEventBase{s.combatant}, w, t.remaining}}}}, nil
}

// StartOfOwnTurn applies the start of the monster's turn: every spent Recharge X-Y
// part in rolls becomes ready when its d6 face is from its Low to its High, both
// inclusive, and stays spent otherwise, and the Legendary Action uses return to their
// maximum. The events are, in this order, one MonsterRechargeRolled per roll in
// printed order, then LegendaryActionUsesRegained when uses were below the maximum.
// With no roll and nothing to regain the state is equal and there is no event. Spell
// uses and Recharge-after-a-Rest parts are not changed.
//
// It returns an error matching srderr.ErrInvalidInput unless the keys of rolls are
// exactly RechargeRollsNeeded and every value is a roll of a d6, and a LimitExceededError
// (plan P10) for a map of more than 64 rolls. It never panics.
func (s LimitedUseState) StartOfOwnTurn(rolls map[LimitedUsePartRef]dice.DieRoll) (LimitedUseTransition, error) {
	if err := overCap("recharge rolls", len(rolls), maxRechargeRolls); err != nil {
		return LimitedUseTransition{}, err
	}
	needed := s.RechargeRollsNeeded()
	if len(rolls) != len(needed) {
		return LimitedUseTransition{}, invalidInput("rolls must be exactly the Recharge parts that need one")
	}
	for _, ref := range needed {
		if r, ok := rolls[ref]; !ok || r.Die() != dice.D6() || r.IsZero() {
			return LimitedUseTransition{}, invalidInput("a Recharge roll needs a d6 for each spent part")
		}
	}
	var events []MonsterLimitedUseEvent
	base := monsterLimitedUseEventBase{s.combatant}
	s.entries = slices.Clone(s.entries)
	for _, ref := range needed {
		i := s.partIndex(ref)
		e := s.entries[i].(RechargeEntry)
		face := rolls[ref].Value()
		result := RechargeNotRegained
		if face >= e.low && face <= e.high {
			result, e.readiness = RechargeRegained, RechargeReady
			s.entries[i] = e
		}
		events = append(events, MonsterRechargeRolled{base, ref, face, result})
	}
	if t, ok := s.legendary.(LegendaryTracked); ok && t.remaining < t.maximum {
		events = append(events, LegendaryActionUsesRegained{base, t.maximum - t.remaining})
		t.remaining = t.maximum
		s.legendary = t
	}
	return LimitedUseTransition{s, events}, nil
}

// AfterRest applies a finished rest, using the recharge mapping of the rest rules
// (character.RestKind.Recharges). A Short Rest and a Long Rest ready every spent
// Recharge part and every spent Recharge-after-a-Rest part. A Long Rest also sets
// every X/Day entry to its maximum for the current lair status and every spell
// entry to its maximum; a Short Rest does not. Legendary Action uses are not
// changed. One LimitedUsesRestored event lists, in printed order, exactly the
// parts and spells that changed, and there is none when nothing changed. An invalid
// rest changes nothing. It never fails.
func (s LimitedUseState) AfterRest(rest character.RestKind) LimitedUseTransition {
	var changed []LimitedUseSubject
	s.entries = slices.Clone(s.entries)
	for i, e := range s.entries {
		switch v := e.(type) {
		case PerDayEntry:
			if limit := v.maximumFor(s.lair); rest.Recharges(content.RechargeLongRest) && v.left != limit {
				v.left = limit
				s.entries[i], changed = v, append(changed, PartSubject{v.part})
			}
		case RechargeEntry:
			if rest.Recharges(content.RechargeShortOrLongRest) && v.readiness == RechargeSpent {
				v.readiness = RechargeReady
				s.entries[i], changed = v, append(changed, PartSubject{v.part})
			}
		case RestRechargeEntry:
			if rest.Recharges(content.RechargeShortOrLongRest) && v.readiness == RechargeSpent {
				v.readiness = RechargeReady
				s.entries[i], changed = v, append(changed, PartSubject{v.part})
			}
		case SpellUseEntry:
			if rest.Recharges(content.RechargeLongRest) && v.remain != v.maximum {
				v.remain = v.maximum
				s.entries[i], changed = v, append(changed, SpellSubject{v.ref})
			}
		}
	}
	if len(changed) == 0 {
		return LimitedUseTransition{s, nil}
	}
	return LimitedUseTransition{s, []MonsterLimitedUseEvent{LimitedUsesRestored{monsterLimitedUseEventBase{s.combatant}, changed}}}
}

// ChangeLair moves the monster to lair status status. Each X/Day maximum and the
// Legendary Action maximum are recomputed for status and the uses already spent are
// kept: remaining becomes max(0, new maximum - spent), spent being the old maximum
// minus remaining. Spell uses are not changed. It adds one MonsterLairStatusChanged
// event when the status differs; when it does not, or when status is undefined, the
// state is equal and there is no event. It never fails.
func (s LimitedUseState) ChangeLair(status LairStatus) LimitedUseTransition {
	if !status.Valid() || status == s.lair {
		return LimitedUseTransition{s, nil}
	}
	old := s.lair
	s.lair = status
	s.entries = slices.Clone(s.entries)
	for i, e := range s.entries {
		if v, ok := e.(PerDayEntry); ok {
			v.left = max(0, v.maximumFor(status)-(v.maximumFor(old)-v.left))
			s.entries[i] = v
		}
	}
	if t, ok := s.legendary.(LegendaryTracked); ok {
		next := legendaryMaximum(t.uses, t.usesInLair, status)
		t.remaining, t.maximum = max(0, next-(t.maximum-t.remaining)), next
		s.legendary = t
	}
	return LimitedUseTransition{s, []MonsterLimitedUseEvent{MonsterLairStatusChanged{monsterLimitedUseEventBase{s.combatant}, old, status}}}
}
