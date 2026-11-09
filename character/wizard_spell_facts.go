package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// WizardClassID is the id of the Wizard class.
const WizardClassID content.ClassID = "wizard"

// WizardSpellFacts is what the Wizard rules need to know about one spell: the
// facts of its content.Spell, plus whether it deals damage (a fact of the
// effect model the caller supplies). The Wizard use cases never load spells; the
// caller resolves them and passes the facts. Construct it with
// NewWizardSpellFacts or a keyed literal. The zero value has no valid id.
type WizardSpellFacts struct {
	ID           content.SpellID
	Level        content.SpellLevel
	School       content.SpellSchool
	Ritual       bool
	Classes      []content.ClassID
	CastingTimes []content.CastingTime
	DealsDamage  bool
}

// NewWizardSpellFacts returns the facts of spell, with dealsDamage as supplied.
func NewWizardSpellFacts(spell content.Spell, dealsDamage bool) WizardSpellFacts {
	f := WizardSpellFacts{ID: spell.ID(), Level: spell.Level(), School: spell.School(), Ritual: spell.Ritual(), Classes: spell.Classes(), DealsDamage: dealsDamage}
	switch t := spell.CastingTimes().(type) {
	case content.SpellCastingTimesSingle:
		f.CastingTimes = []content.CastingTime{t.Time()}
	case content.SpellCastingTimesModes:
		for _, m := range t.Modes() {
			f.CastingTimes = append(f.CastingTimes, m.Time())
		}
	}
	return f
}

// OnWizardList reports whether the spell is on the Wizard spell list: its
// Classes contain the Wizard (the test classSpellList applies; no separate list is stored).
func (f WizardSpellFacts) OnWizardList() bool { return slices.Contains(f.Classes, WizardClassID) }

// IsCantrip reports whether the spell is level 0.
func (f WizardSpellFacts) IsCantrip() bool { return f.Level == 0 }

// HasActionCastingTime reports whether one of the casting times is the Magic action.
func (f WizardSpellFacts) HasActionCastingTime() bool {
	return slices.ContainsFunc(f.CastingTimes, func(t content.CastingTime) bool {
		_, ok := t.(content.CastingTimeMagicAction)
		return ok
	})
}

func wizardFactIDs(in []WizardSpellFacts) []content.SpellID {
	out := make([]content.SpellID, len(in))
	for i, f := range in {
		out[i] = f.ID
	}
	return out
}

// firstRepeat returns the first id that repeats an earlier one.
func firstRepeat(ids []content.SpellID) (content.SpellID, bool) {
	for i, id := range ids {
		if slices.Contains(ids[:i], id) {
			return id, true
		}
	}
	return "", false
}
