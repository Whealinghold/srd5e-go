package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

// WizardResources is the Wizard's class feature state: whether Arcane Recovery is
// used, the Overchannel use count held for the Evoker, the Scholar skill, the Spell
// Mastery and Signature Spells choices and which Signature Spells have had their free
// cast. It holds no level, no subclass and no pending choice (those are derived).
// It is stored under WizardResourcesKey; an absent state is the zero value (nothing
// used, nothing chosen). Immutable, comparable with ==, safe for concurrent use.
type WizardResources struct {
	arcaneRecoveryUsed bool
	overchannelUses    int
	scholar            content.Skill
	mastery            SpellMastery
	signature          SignatureSpells
	signatureSpent     [2]bool
}

// WizardResourcesKey returns the FeatureStateKey named wizard_resources under which a
// character holds its WizardResources.
func WizardResourcesKey() FeatureStateKey[WizardResources] {
	return MustFeatureStateKey[WizardResources]("wizard_resources")
}

// ArcaneRecoveryUsed reports whether Arcane Recovery was used since the last Long Rest.
func (r WizardResources) ArcaneRecoveryUsed() bool { return r.arcaneRecoveryUsed }

// OverchannelUses returns the Overchannel uses since the last Long Rest (0 is a first use next).
func (r WizardResources) OverchannelUses() int { return r.overchannelUses }

// Scholar returns the Scholar skill and true, or false when none is chosen.
func (r WizardResources) Scholar() (content.Skill, bool) { return r.scholar, r.scholar != 0 }

// SpellMastery returns the Spell Mastery choice and true, or false when none is held.
func (r WizardResources) SpellMastery() (SpellMastery, bool) { return r.mastery, !r.mastery.IsZero() }

// SignatureSpells returns the Signature Spells choice and true, or false when none is held.
func (r WizardResources) SignatureSpells() (SignatureSpells, bool) {
	return r.signature, !r.signature.IsZero()
}

// SignatureSpellSpent reports whether the free cast of the Signature Spell is spent
// since the last Short or Long Rest; false for a spell that is not a Signature Spell.
func (r WizardResources) SignatureSpellSpent(spell content.SpellID) bool {
	i := slices.Index(r.signature.Spells(), spell)
	return i >= 0 && r.signatureSpent[i]
}

// Equal reports whether other is a WizardResources holding the same values.
func (r WizardResources) Equal(other FeatureState) bool {
	o, ok := other.(WizardResources)
	return ok && o == r
}

// WithOverchannelUses returns the state with the Overchannel use count n. It returns an
// error matching srderr.ErrInvalidInput for a negative count.
func (r WizardResources) WithOverchannelUses(n int) (WizardResources, error) {
	if n < 0 {
		return WizardResources{}, invalidInput("Wizard resources with a negative Overchannel count " + strconv.Itoa(n))
	}
	r.overchannelUses = n
	return r, nil
}

// WithArcaneRecoveryUsed returns the state with Arcane Recovery marked as used.
func (r WizardResources) WithArcaneRecoveryUsed() WizardResources {
	r.arcaneRecoveryUsed = true
	return r
}

// withScholar, withMastery and withSignature return the state holding the choice.
func (r WizardResources) withScholar(s content.Skill) WizardResources {
	r.scholar = s
	return r
}

func (r WizardResources) withMastery(m SpellMastery) WizardResources {
	r.mastery = m
	return r
}

func (r WizardResources) withSignature(s SignatureSpells) WizardResources {
	r.signature = s
	r.signatureSpent = [2]bool{}
	return r
}

// RecordSignatureSpellCast returns the state with the free cast of spell recorded as
// spent. It returns an error matching srderr.ErrInvalidInput when spell is not a
// Signature Spell. It produces no event: the cast event is the spellcasting one.
func (r WizardResources) RecordSignatureSpellCast(spell content.SpellID) (WizardResources, error) {
	i := slices.Index(r.signature.Spells(), spell)
	if i < 0 {
		return WizardResources{}, invalidInput("record a Signature Spell cast for a spell that is not one")
	}
	r.signatureSpent[i] = true
	return r, nil
}

// AfterRest returns the state after the benefits of a rest of kind: a Short Rest or a
// Long Rest makes both Signature Spells free again, and a Long Rest also makes Arcane
// Recovery available and resets the Overchannel use count (SRD 5.2.1, "Level 1: Arcane
// Recovery", p. 78; "Level 20: Signature Spells", p. 79; "Level 14: Overchannel", p. 82).
// Any other kind changes nothing. The host applies it when a RestApplied for this
// character arrives, passing RestApplied.BenefitsOf.
func (r WizardResources) AfterRest(kind RestKind) WizardResources {
	switch kind {
	case LongRest:
		r.arcaneRecoveryUsed, r.overchannelUses = false, 0
		r.signatureSpent = [2]bool{}
	case ShortRest:
		r.signatureSpent = [2]bool{}
	}
	return r
}

// WizardCastRequirement says whether a cast needs a Spell Slot.
type WizardCastRequirement int

// The requirements.
const (
	// WizardCastWithoutSlot: the cast expends no Spell Slot.
	WizardCastWithoutSlot WizardCastRequirement = iota + 1
	// WizardCastNeedsSlot: the cast expends a Spell Slot.
	WizardCastNeedsSlot
)

// SpellMastery is the validated Spell Mastery choice: one level 1 and one level 2 spell
// (SRD 5.2.1, "Level 18: Spell Mastery", p. 79). Comparable; the zero value is no choice.
type SpellMastery struct{ levelOne, levelTwo content.SpellID }

// IsZero reports whether no choice is held.
func (m SpellMastery) IsZero() bool { return m == SpellMastery{} }

// LevelOne returns the level 1 spell.
func (m SpellMastery) LevelOne() content.SpellID { return m.levelOne }

// LevelTwo returns the level 2 spell.
func (m SpellMastery) LevelTwo() content.SpellID { return m.levelTwo }

// Spells returns a new slice of the two spells, level 1 first.
func (m SpellMastery) Spells() []content.SpellID {
	if m.IsZero() {
		return nil
	}
	return []content.SpellID{m.levelOne, m.levelTwo}
}

// CastRequirement reports a Spell Mastery spell cast at its lowest level as castable
// without a Spell Slot (unlimited, untracked) and anything else, including a higher
// level and a spell that is not a Mastery spell, as needing one.
func (m SpellMastery) CastRequirement(spell content.SpellID, castAtLevel int) WizardCastRequirement {
	if (spell == m.levelOne && castAtLevel == 1) || (spell == m.levelTwo && castAtLevel == 2) {
		return WizardCastWithoutSlot
	}
	return WizardCastNeedsSlot
}

// NewSpellMastery validates the choice of two spells from the book (SRD 5.2.1, "Level
// 18: Spell Mastery", p. 79): a level 1 and a level 2 spell, in the spellbook, each with
// the Magic action as a casting time. A rejection has a non-zero Kind: NotInSpellbook,
// DuplicateSpell, WrongSpellLevels, CastingTimeNotAction, the first offender in command order.
func NewSpellMastery(chosen []WizardSpellFacts, book Spellbook) (SpellMastery, SpellMasteryRejection) {
	for _, f := range chosen {
		if !book.Contains(f.ID) {
			return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryNotInSpellbook, Spell: f.ID}
		}
	}
	if id, dup := firstRepeat(wizardFactIDs(chosen)); dup {
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryDuplicateSpell, Spell: id}
	}
	levels := make([]int, len(chosen))
	for i, f := range chosen {
		levels[i] = int(f.Level)
	}
	if len(chosen) != 2 || (levels[0]+levels[1] != 3 || levels[0] == levels[1]) {
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryWrongSpellLevels, Levels: levels}
	}
	for _, f := range chosen {
		if !f.HasActionCastingTime() {
			return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryCastingTimeNotAction, Spell: f.ID}
		}
	}
	if levels[0] == 1 {
		return SpellMastery{chosen[0].ID, chosen[1].ID}, SpellMasteryRejection{}
	}
	return SpellMastery{chosen[1].ID, chosen[0].ID}, SpellMasteryRejection{}
}

// replace returns the Mastery with the spell replaced by an eligible one of the same
// level (SRD 5.2.1, "Level 18: Spell Mastery", p. 79). A rejection has a non-zero Kind:
// NotAMasterySpell, NotInSpellbook, DuplicateSpell (the new spell is the other Mastery
// spell), WrongSpellLevels or CastingTimeNotAction.
func (m SpellMastery) replace(old content.SpellID, next WizardSpellFacts, book Spellbook) (SpellMastery, SpellMasteryRejection) {
	level, other := 1, m.levelTwo
	switch old {
	case m.levelOne:
	case m.levelTwo:
		level, other = 2, m.levelOne
	default:
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryNotAMasterySpell, Spell: old}
	}
	switch {
	case !book.Contains(next.ID):
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryNotInSpellbook, Spell: next.ID}
	case next.ID == other:
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryDuplicateSpell, Spell: next.ID}
	case int(next.Level) != level:
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryWrongSpellLevels, Levels: []int{int(next.Level)}}
	case !next.HasActionCastingTime():
		return SpellMastery{}, SpellMasteryRejection{Kind: SpellMasteryCastingTimeNotAction, Spell: next.ID}
	}
	if level == 1 {
		return SpellMastery{next.ID, m.levelTwo}, SpellMasteryRejection{}
	}
	return SpellMastery{m.levelOne, next.ID}, SpellMasteryRejection{}
}

// SignatureSpells is the validated Signature Spells choice: two level 3 spells (SRD
// 5.2.1, "Level 20: Signature Spells", p. 79). Comparable; the zero value is no choice.
type SignatureSpells struct{ first, second content.SpellID }

// IsZero reports whether no choice is held.
func (s SignatureSpells) IsZero() bool { return s == SignatureSpells{} }

// Spells returns a new slice of the two spells in the order chosen.
func (s SignatureSpells) Spells() []content.SpellID {
	if s.IsZero() {
		return nil
	}
	return []content.SpellID{s.first, s.second}
}

// CastRequirement reports a Signature Spell cast at level 3 whose free cast is not spent
// in resources as castable without a Spell Slot, and anything else as needing one.
func (s SignatureSpells) CastRequirement(spell content.SpellID, castAtLevel int, resources WizardResources) WizardCastRequirement {
	if castAtLevel == 3 && slices.Contains(s.Spells(), spell) && !resources.SignatureSpellSpent(spell) {
		return WizardCastWithoutSlot
	}
	return WizardCastNeedsSlot
}

// NewSignatureSpells validates the choice of exactly two distinct level 3 spells from the
// book. A rejection has a non-zero Kind: WrongSpellCount, DuplicateSpell, NotLevelThree,
// NotInSpellbook, the first offender in command order.
func NewSignatureSpells(chosen []WizardSpellFacts, book Spellbook) (SignatureSpells, SignatureSpellsRejection) {
	if len(chosen) != 2 {
		return SignatureSpells{}, SignatureSpellsRejection{Kind: SignatureSpellsWrongSpellCount, Allowed: 2}
	}
	if id, dup := firstRepeat(wizardFactIDs(chosen)); dup {
		return SignatureSpells{}, SignatureSpellsRejection{Kind: SignatureSpellsDuplicateSpell, Spell: id}
	}
	for _, f := range chosen {
		if f.Level != 3 {
			return SignatureSpells{}, SignatureSpellsRejection{Kind: SignatureSpellsNotLevelThree, Spell: f.ID}
		}
	}
	for _, f := range chosen {
		if !book.Contains(f.ID) {
			return SignatureSpells{}, SignatureSpellsRejection{Kind: SignatureSpellsNotInSpellbook, Spell: f.ID}
		}
	}
	return SignatureSpells{chosen[0].ID, chosen[1].ID}, SignatureSpellsRejection{}
}

// WizardAlwaysPrepared gives the spells a Wizard always has prepared through Spell
// Mastery and Signature Spells (SRD 5.2.1, "Level 18: Spell Mastery", "Level 20:
// Signature Spells", p. 79). The host passes them as the always-prepared set. The zero
// value is ready to use.
type WizardAlwaysPrepared struct{}

// Of returns a new slice of the always-prepared spell ids of resources: Spell Mastery
// first, then Signature Spells.
func (WizardAlwaysPrepared) Of(resources WizardResources) []content.SpellID {
	return append(resources.mastery.Spells(), resources.signature.Spells()...)
}
