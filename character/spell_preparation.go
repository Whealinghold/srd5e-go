package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// PreparationTrigger is the event after which a class may replace prepared spells
// (SRD 5.2.1, "Preparing Spells", Spell Preparation by Class, p. 104). The zero
// value is invalid. Comparable and safe for concurrent use.
type PreparationTrigger int

// The preparation triggers.
const (
	// TriggerLongRest: the character finished a Long Rest.
	TriggerLongRest PreparationTrigger = iota + 1
	// TriggerShortRest: the character finished a Short Rest.
	TriggerShortRest
	// TriggerGainedLevel: the character gained a level.
	TriggerGainedLevel
)

// Valid reports whether t is one of the three triggers.
func (t PreparationTrigger) Valid() bool { return t >= TriggerLongRest && t <= TriggerGainedLevel }

// String returns "long_rest", "short_rest", "gained_level" or "unknown".
func (t PreparationTrigger) String() string {
	switch t {
	case TriggerLongRest:
		return "long_rest"
	case TriggerShortRest:
		return "short_rest"
	case TriggerGainedLevel:
		return "gained_level"
	}
	return "unknown"
}

// PreparationOutcome is the closed outcome of ChoosePreparation and
// ChangePreparation. Variants: PreparationAccepted, PreparationRefused,
// PreparationContentMissing.
type PreparationOutcome interface{ isPreparationOutcome() }

// PreparationAccepted: the lists were accepted; the new state and the events,
// none when nothing changed.
type PreparationAccepted struct {
	state  SpellcastingState
	events []SpellcastingEvent
}

// State returns the new state.
func (o PreparationAccepted) State() SpellcastingState { return o.state }

// Events returns a new slice of the events.
func (o PreparationAccepted) Events() []SpellcastingEvent { return slices.Clone(o.events) }

// PreparationRefused: a rule refused the lists; the state is unchanged.
type PreparationRefused struct{ rejection PreparationRejection }

// Rejection returns every problem found.
func (o PreparationRefused) Rejection() PreparationRejection { return o.rejection }

// PreparationContentMissing: a spell id named in the command is absent from the
// spells supplied. It takes precedence over every rule rejection; the state is
// unchanged.
type PreparationContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the absent spells, in the order named, without repeats.
func (o PreparationContentMissing) Missing() []content.MissingContent { return slices.Clone(o.missing) }

func (PreparationAccepted) isPreparationOutcome()       {}
func (PreparationRefused) isPreparationOutcome()        {}
func (PreparationContentMissing) isPreparationOutcome() {}

// GrantedCantripReplacement replaces one granted cantrip of a class with another
// the host supplies as eligible for the granting feature. Construct it with a
// keyed literal.
type GrantedCantripReplacement struct {
	Replaced content.SpellID
	With     GrantedCantrip
}

// PreparationChange is what ChangePreparation changes. A nil Prepared or Cantrips
// leaves that list as it is; an empty non-nil slice empties it. Prepared is the
// complete new prepared list including the always-prepared spells, which do not
// count against the limit and must all stay. Construct it with a keyed literal.
type PreparationChange struct {
	Prepared            []content.SpellID
	Cantrips            []content.SpellID
	GrantedReplacements []GrantedCantripReplacement
}

// ChoosePreparation adds level 1+ spells and cantrips to a class's lists without
// a trigger, up to the headroom the tables leave (SRD 5.2.1, the class
// "Prepared Spells of Level 1+" and "Cantrips" entries). Each addition is checked
// as the lists are everywhere else, so too many additions are refused with
// PreparationTooManyPrepared or PreparationTooManyCantrips. An empty addition is
// accepted with no events.
//
// It returns PreparationContentMissing, before any rule check, for a spell id
// absent from spells. It returns an error matching ErrClassNotHeld (and
// srderr.ErrInvalidInput) with a nil outcome when the state does not hold class.
// It never panics and does not alter its arguments.
func ChoosePreparation(s SpellcastingState, class SpellcastingClass, prepared, cantrips []content.SpellID, spells map[content.SpellID]content.Spell) (PreparationOutcome, error) {
	cur, ok := s.Class(class)
	if !ok {
		return nil, ErrClassNotHeld
	}
	if miss := missingSpells(spells, prepared, cantrips); len(miss) > 0 {
		return PreparationContentMissing{missing: miss}, nil
	}
	next := cur
	next.prepared.ids = slices.Concat(cur.prepared.ids, prepared)
	next.cantrips.ids = slices.Concat(cur.cantrips.ids, cantrips)
	if p := next.problems(spells); len(p) > 0 {
		return PreparationRefused{rejection: PreparationRejection{problems: p}}, nil
	}
	if len(prepared)+len(cantrips) == 0 {
		return PreparationAccepted{state: s}, nil
	}
	return PreparationAccepted{state: s.withClass(next), events: []SpellcastingEvent{PreparedSpellsChanged{id: s.id, class: class}}}, nil
}

// replacementLimit returns how many prepared spells class may replace on trigger
// t and whether t is a trigger for it at all; a limit below 0 means any number
// (Table I; Wizard "Memorize Spell", p. 79).
func replacementLimit(class SpellcastingClass, level int, t PreparationTrigger) (limit int, ok bool) {
	switch {
	case t == TriggerLongRest && (class == CasterCleric || class == CasterDruid || class == CasterWizard):
		return -1, true
	case t == TriggerLongRest && class.isHalfCaster(), t == TriggerGainedLevel && (class == CasterBard || class == CasterSorcerer || class == CasterWarlock):
		return 1, true
	case t == TriggerShortRest && class == CasterWizard && level >= 5:
		return 1, true
	}
	return 0, false
}

// cantripTriggerOK reports whether t lets class replace a cantrip.
func cantripTriggerOK(class SpellcastingClass, t PreparationTrigger) bool {
	if class == CasterWizard {
		return t == TriggerLongRest
	}
	return t == TriggerGainedLevel && (class == CasterBard || class == CasterCleric || class == CasterDruid || class == CasterSorcerer || class == CasterWarlock)
}

// countAbsent returns how many of old are not in now.
func countAbsent(old, now []content.SpellID) int {
	n := 0
	for _, id := range old {
		if !slices.Contains(now, id) {
			n++
		}
	}
	return n
}

// ChangePreparation changes a class's lists on a trigger: replaces prepared
// spells, cantrips or granted cantrips and adds spells within the headroom (SRD
// 5.2.1, "Preparing Spells", p. 104; "Memorize Spell", p. 79). A replacement is a
// spell in the current list and absent from the new one; on a trigger the class
// lacks it is refused with PreparationChangeNotAllowedNow, and above the number
// the class may replace with PreparationTooManyReplacements. Dropping an
// always-prepared spell is refused with PreparationAlwaysPreparedNotReplaceable.
// Every list is checked as everywhere else and every problem is reported.
//
// It returns PreparationContentMissing, before any rule check, for a spell id
// absent from spells. It returns an error matching ErrClassNotHeld (and
// srderr.ErrInvalidInput) with a nil outcome when the state does not hold class
// or a replacement names a class it does not hold, and an error matching
// srderr.ErrInvalidInput for an invalid trigger, a replacement that names another
// class or a replaced cantrip the class was not granted. It never panics and
// does not alter its arguments.
func ChangePreparation(s SpellcastingState, class SpellcastingClass, trigger PreparationTrigger, change PreparationChange, spells map[content.SpellID]content.Spell) (PreparationOutcome, error) {
	cur, ok := s.Class(class)
	if !ok {
		return nil, ErrClassNotHeld
	}
	if !trigger.Valid() {
		return nil, invalidInput("change prepared spells: invalid trigger")
	}
	with := make([]content.SpellID, len(change.GrantedReplacements))
	granted := slices.Clone(cur.granted)
	for i, r := range change.GrantedReplacements {
		if _, held := s.Class(r.With.class); !held {
			return nil, ErrClassNotHeld
		}
		at := slices.IndexFunc(granted, func(g GrantedCantrip) bool { return g.spell == r.Replaced })
		if r.With.class != class || at < 0 {
			return nil, invalidInput("change prepared spells: replacement names another class or an ungranted cantrip")
		}
		granted[at], with[i] = r.With, r.With.spell
	}
	if miss := missingSpells(spells, change.Prepared, change.Cantrips, with); len(miss) > 0 {
		return PreparationContentMissing{missing: miss}, nil
	}
	next := cur
	next.granted = granted
	var extra []PreparationProblem
	note := func(k PreparationRejectionKind, id content.SpellID) {
		extra = append(extra, PreparationProblem{Kind: k, Spell: id, Class: class})
	}
	replaced := func(n, limit int, allowed bool) {
		switch {
		case n > 0 && !allowed:
			note(PreparationChangeNotAllowedNow, "")
		case n > 0 && limit >= 0 && n > limit:
			note(PreparationTooManyReplacements, "")
		}
	}
	if change.Prepared != nil {
		next.prepared.ids = slices.DeleteFunc(slices.Clone(change.Prepared), func(id content.SpellID) bool { return slices.Contains(cur.prepared.always, id) })
		for _, a := range cur.prepared.always {
			if !slices.Contains(change.Prepared, a) {
				note(PreparationAlwaysPreparedNotReplaceable, a)
			}
		}
		limit, allowed := replacementLimit(class, cur.level, trigger)
		replaced(countAbsent(cur.prepared.ids, next.prepared.ids), limit, allowed)
	}
	if change.Cantrips != nil {
		next.cantrips.ids = slices.Clone(change.Cantrips)
		replaced(countAbsent(cur.cantrips.ids, next.cantrips.ids), 1, cantripTriggerOK(class, trigger))
	}
	replaced(len(change.GrantedReplacements), 1, trigger == TriggerGainedLevel)
	problems := slices.Concat(extra, next.problems(spells))
	slices.SortStableFunc(problems, func(a, b PreparationProblem) int { return int(a.Kind) - int(b.Kind) })
	if len(problems) > 0 {
		return PreparationRefused{rejection: PreparationRejection{problems: problems}}, nil
	}
	if slices.Equal(next.prepared.ids, cur.prepared.ids) && slices.Equal(next.cantrips.ids, cur.cantrips.ids) && slices.Equal(next.granted, cur.granted) {
		return PreparationAccepted{state: s}, nil
	}
	return PreparationAccepted{state: s.withClass(next), events: []SpellcastingEvent{PreparedSpellsChanged{id: s.id, class: class}}}, nil
}
