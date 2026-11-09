package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// AttuneRejection is why an attempt to attune was refused. The zero value is no rejection.
type AttuneRejection int

// The rejections of AttunementSet.Attune, in the order they are checked (the SRD
// gives none, so the order is package design).
const (
	AttuneWrongRestPurpose AttuneRejection = iota + 1
	AttuneRestNotFocusedOnItem
	AttuneAlreadyAttunedToItem
	AttuneDuplicateCopy
	AttunePrerequisiteJudgementMissing
	AttunePrerequisiteNotMet
	AttuneLimitReached
	AttuneRestInterrupted
)

// String returns the snake_case name of the rejection, or "unknown".
func (r AttuneRejection) String() string {
	names := []string{"wrong_rest_purpose", "rest_not_focused", "already_attuned", "duplicate_copy", "prerequisite_judgement_missing", "prerequisite_not_met", "limit_reached", "rest_interrupted"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// Attune attunes to item at the end of a Short Rest focused on it, or reports
// the first rejection in the order AttuneWrongRestPurpose, AttuneRestNotFocusedOnItem
// (a different instance or no contact), AttuneAlreadyAttunedToItem,
// AttuneDuplicateCopy, AttunePrerequisiteJudgementMissing, AttunePrerequisiteNotMet,
// AttuneLimitReached (the set is at or above limit), AttuneRestInterrupted. On a
// rejection the set is returned unchanged and the rejection is nonzero; on
// success the rejection is zero. It returns an error matching
// srderr.ErrInvalidInput for a zero holder, item, focus, limit or candidate, or
// an empty class prerequisite.
func (s AttunementSet) Attune(holder Identifier, item AttunableItem, candidate AttuneCandidate, judgement PrerequisiteJudgement, focus ShortRestFocus, limit AttunementLimit) (AttunementSet, MagicItemAttuned, AttuneRejection, error) {
	if holder.IsZero() || item.IsZero() || focus.IsZero() || limit < limit.Standard() || candidate.spellcasting == 0 {
		return s, MagicItemAttuned{}, 0, invalidInput("attune: zero holder, item, focus or candidate, or limit below 3")
	}
	prereq, err := AttunementPrerequisiteReader{}.Read(item.attunement)
	if err != nil {
		return s, MagicItemAttuned{}, 0, err
	}
	elig := AttunementEligibility{}
	met, err := elig.IsMet(prereq, candidate, judgement)
	if err != nil {
		return s, MagicItemAttuned{}, 0, err
	}
	reject := func(r AttuneRejection) (AttunementSet, MagicItemAttuned, AttuneRejection, error) {
		return s, MagicItemAttuned{}, r, nil
	}
	switch {
	case focus.purpose != RestPurposeAttune:
		return reject(AttuneWrongRestPurpose)
	case focus.instance != item.instance || focus.contact != InContact:
		return reject(AttuneRestNotFocusedOnItem)
	case s.IsAttunedTo(item.instance):
		return reject(AttuneAlreadyAttunedToItem)
	case s.holdsCopyOf(item.item):
		return reject(AttuneDuplicateCopy)
	case elig.JudgementMissing(prereq, judgement):
		return reject(AttunePrerequisiteJudgementMissing)
	case !met:
		return reject(AttunePrerequisiteNotMet)
	case s.Len() >= int(limit):
		return reject(AttuneLimitReached)
	case !focus.isCompleted:
		return reject(AttuneRestInterrupted)
	}
	return s.with(AttunedItem{item: item.item, instance: item.instance}), MagicItemAttuned{id: holder, item: item.item, instance: item.instance}, 0, nil
}

func (s AttunementSet) holdsCopyOf(item content.MagicItemID) bool {
	for _, a := range s.items {
		if a.item == item {
			return true
		}
	}
	return false
}

// EndRejection is why an end of Attunement was refused. The zero value is no rejection.
type EndRejection int

// The rejections of AttunementSet.End.
const (
	EndCurseBlocksEnding EndRejection = iota + 1
	EndNotAttuned
	EndNotCursed
	EndSpellDoesNotEndAttunement
	EndRestInterrupted
	EndRestNotFocusedOnItem
	EndWrongRestPurpose
	EndEndingConditionNotMet
)

// String returns the snake_case name of the rejection, or "unknown".
func (r EndRejection) String() string {
	names := []string{"curse_blocks_ending", "not_attuned", "not_cursed", "spell_does_not_end_attunement", "rest_interrupted", "rest_not_focused", "wrong_rest_purpose", "ending_condition_not_met"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// CurseBreak is the host's report that an item's curse was broken first. The
// zero value is CurseIntact.
type CurseBreak int

// Whether the curse is broken.
const (
	CurseIntact CurseBreak = iota
	CurseBroken
)

// AttunementEndTrigger is the closed set of facts that end Attunement without a
// choice. Variants: AttunementEndDied, AttunementEndAnotherCreatureAttuned,
// AttunementEndPrerequisiteCheck, AttunementEndDistanceCheck.
type AttunementEndTrigger interface{ isAttunementEndTrigger() }

// AttunementEndDied: the creature died; every item ends.
type AttunementEndDied struct{}

// AttunementEndAnotherCreatureAttuned: another creature attuned to the instance.
type AttunementEndAnotherCreatureAttuned struct{}

// AttunementEndPrerequisiteCheck asks whether the prerequisite is still met.
type AttunementEndPrerequisiteCheck struct {
	prerequisite content.AttunementPrerequisite
	candidate    AttuneCandidate
	judgement    PrerequisiteJudgement
}

// NewAttunementEndPrerequisiteCheck returns the trigger. It returns an error
// matching srderr.ErrInvalidInput for a nil prerequisite.
func NewAttunementEndPrerequisiteCheck(p content.AttunementPrerequisite, c AttuneCandidate, j PrerequisiteJudgement) (AttunementEndPrerequisiteCheck, error) {
	if p == nil {
		return AttunementEndPrerequisiteCheck{}, invalidInput("prerequisite check without a prerequisite")
	}
	return AttunementEndPrerequisiteCheck{prerequisite: p, candidate: c, judgement: j}, nil
}

// AttunementEndDistanceCheck asks whether the item has been too far away too long.
type AttunementEndDistanceCheck struct{ distanceFeet, hoursAway int }

// NewAttunementEndDistanceCheck returns the trigger. It returns an error
// matching srderr.ErrInvalidInput for a negative distance or hours.
func NewAttunementEndDistanceCheck(distanceFeet, hoursAway int) (AttunementEndDistanceCheck, error) {
	if distanceFeet < 0 || hoursAway < 0 {
		return AttunementEndDistanceCheck{}, invalidInput("negative distance or hours away")
	}
	return AttunementEndDistanceCheck{distanceFeet: distanceFeet, hoursAway: hoursAway}, nil
}

func (AttunementEndDied) isAttunementEndTrigger()                   {}
func (AttunementEndAnotherCreatureAttuned) isAttunementEndTrigger() {}
func (AttunementEndPrerequisiteCheck) isAttunementEndTrigger()      {}
func (AttunementEndDistanceCheck) isAttunementEndTrigger()          {}

// EndMeans is the closed set of ways Attunement to an instance can end.
// Variants: EndByShortRest, EndBySpell, EndByTrigger.
type EndMeans interface{ isEndMeans() }

// EndByShortRest: a voluntary end by a Short Rest focused on the item.
type EndByShortRest struct{ Focus ShortRestFocus }

// EndBySpell: a spell ended it; Spell is the SpellID the host reports
// (SpellRemoveCurse or SpellGreaterRestoration).
type EndBySpell struct{ Spell content.SpellID }

// EndByTrigger: a fact ended it.
type EndByTrigger struct{ Trigger AttunementEndTrigger }

func (EndByShortRest) isEndMeans() {}
func (EndBySpell) isEndMeans()     {}
func (EndByTrigger) isEndMeans()   {}

// The spells that end Attunement to a cursed item.
const (
	SpellRemoveCurse        content.SpellID = "remove_curse"
	SpellGreaterRestoration content.SpellID = "greater_restoration"
)

// End ends Attunement to instance by means. The rejection is nonzero when it was
// refused, with the set unchanged and no events. A voluntary end checks, in
// order, EndNotAttuned, EndWrongRestPurpose, EndRestNotFocusedOnItem (instance
// only; contact is not required), EndCurseBlocksEnding, EndRestInterrupted. A
// spell checks EndSpellDoesNotEndAttunement, EndNotAttuned, EndNotCursed. A
// trigger ends: AttunementEndDied every item and never rejects;
// AttunementEndAnotherCreatureAttuned the instance (EndNotAttuned);
// the prerequisite and distance checks check EndNotAttuned then
// EndEndingConditionNotMet. It returns an error matching srderr.ErrInvalidInput
// for a zero holder, a blank instance, nil means, or an invalid trigger.
func (s AttunementSet) End(holder Identifier, instance MagicItemInstanceID, curse content.CurseStatus, broken CurseBreak, means EndMeans) (AttunementSet, []MagicItemAttunementEnded, EndRejection, error) {
	if holder.IsZero() || !instance.Valid() || means == nil {
		return s, nil, 0, invalidInput("end attunement: zero holder, blank instance or nil means")
	}
	reject := func(r EndRejection) (AttunementSet, []MagicItemAttunementEnded, EndRejection, error) {
		return s, nil, r, nil
	}
	endOne := func(reason AttunementEndReason) (AttunementSet, []MagicItemAttunementEnded, EndRejection, error) {
		it, _ := s.find(instance)
		return s.without(instance), []MagicItemAttunementEnded{{id: holder, item: it.item, instance: instance, reason: reason}}, 0, nil
	}
	_, attuned := s.find(instance)
	switch m := means.(type) {
	case EndByShortRest:
		switch {
		case !attuned:
			return reject(EndNotAttuned)
		case m.Focus.purpose != RestPurposeEndAttunement:
			return reject(EndWrongRestPurpose)
		case m.Focus.instance != instance:
			return reject(EndRestNotFocusedOnItem)
		case curse == content.Cursed && broken != CurseBroken:
			return reject(EndCurseBlocksEnding)
		case !m.Focus.isCompleted:
			return reject(EndRestInterrupted)
		}
		return endOne(AttunementEndedByShortRest)
	case EndBySpell:
		switch {
		case m.Spell != SpellRemoveCurse && m.Spell != SpellGreaterRestoration:
			return reject(EndSpellDoesNotEndAttunement)
		case !attuned:
			return reject(EndNotAttuned)
		case curse != content.Cursed:
			return reject(EndNotCursed)
		}
		return endOne(AttunementEndedCurseBroken)
	case EndByTrigger:
		return s.endByTrigger(holder, m.Trigger, attuned, reject, endOne)
	}
	return s, nil, 0, invalidInput("end attunement: foreign means")
}

func (s AttunementSet) endByTrigger(holder Identifier, trigger AttunementEndTrigger, attuned bool,
	reject func(EndRejection) (AttunementSet, []MagicItemAttunementEnded, EndRejection, error),
	endOne func(AttunementEndReason) (AttunementSet, []MagicItemAttunementEnded, EndRejection, error)) (AttunementSet, []MagicItemAttunementEnded, EndRejection, error) {
	var verdict AttunementEndingVerdict
	var err error
	switch t := trigger.(type) {
	case AttunementEndDied:
		events := make([]MagicItemAttunementEnded, 0, len(s.items))
		for _, it := range s.items {
			events = append(events, MagicItemAttunementEnded{id: holder, item: it.item, instance: it.instance, reason: AttunementEndedDied})
		}
		return AttunementSet{}, events, 0, nil
	case AttunementEndAnotherCreatureAttuned:
		if !attuned {
			return reject(EndNotAttuned)
		}
		return endOne(AttunementEndedAnotherCreatureAttuned)
	case AttunementEndPrerequisiteCheck:
		if !attuned {
			return reject(EndNotAttuned)
		}
		verdict, err = AttunementEnding{}.CheckPrerequisite(t.prerequisite, t.candidate, t.judgement)
	case AttunementEndDistanceCheck:
		if !attuned {
			return reject(EndNotAttuned)
		}
		verdict, err = AttunementEnding{}.CheckDistance(t.distanceFeet, t.hoursAway)
	default:
		return s, nil, 0, invalidInput("end attunement: nil or foreign trigger")
	}
	if err != nil {
		return s, nil, 0, err
	}
	if v, ends := verdict.(AttunementEnds); ends {
		return endOne(v.Reason)
	}
	return reject(EndEndingConditionNotMet)
}
