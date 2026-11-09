package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// shortRestFinished reports whether trigger is the RestApplied of a finished Short
// Rest (benefits of a Short Rest, source finished). Arcane Recovery and Memorize Spell
// open only on it; an interrupted Long Rest does not.
func shortRestFinished(trigger RestApplied) bool {
	return trigger.BenefitsOf() == ShortRest && trigger.Source() == RestFinished
}

// longRestBenefits reports whether trigger carries the benefits of a Long Rest.
func longRestBenefits(trigger RestApplied) bool { return trigger.BenefitsOf() == LongRest }

// wizardFeatureLevel returns the level at which def gives the feature entry id, or 0 when
// def has no such entry.
func wizardFeatureLevel(def content.ClassDefinition, id content.FeatureID) int {
	if found, ok := def.Feature(id).(content.FeatureEntryFound); ok {
		return found.Entry().Level()
	}
	return 0
}

// scholarSkills are the skills Scholar can give Expertise in (SRD 5.2.1, "Level 2: Scholar", p. 78).
func scholarSkills() []content.Skill {
	return []content.Skill{content.SkillArcana, content.SkillHistory, content.SkillInvestigation, content.SkillMedicine, content.SkillNature, content.SkillReligion}
}

// checkScholar validates a Scholar choice: the feature is gained (available), none is held
// already, the skill is one of the six and the character is proficient in it. A rejection has a non-zero Kind.
func checkScholar(available bool, held WizardResources, skill content.Skill, proficient []content.Skill) ScholarRejection {
	switch {
	case !available:
		return ScholarRejection{Kind: ScholarFeatureNotAvailable}
	case held.scholar != 0:
		return ScholarRejection{Kind: ScholarAlready}
	case !slices.Contains(scholarSkills(), skill):
		return ScholarRejection{Kind: ScholarSkillNotEligible, Skill: skill}
	case !slices.Contains(proficient, skill):
		return ScholarRejection{Kind: ScholarNotProficient, Skill: skill}
	}
	return ScholarRejection{}
}

// MemorizeRequest is what Memorize Spell decides on. Construct it with a keyed literal.
type MemorizeRequest struct {
	Available    bool // the feature is gained (Wizard level 5 or higher)
	Trigger      RestApplied
	Prepared     []content.SpellID
	AlwaysPrep   []content.SpellID
	Spellbook    Spellbook
	Replace      content.SpellID
	New          WizardSpellFacts
	HighestLevel int
}

// checkMemorize applies the Memorize Spell rules (SRD 5.2.1, "Level 5: Memorize Spell", p. 79)
// and returns the new prepared list, with the replaced spell's place taken by the new one.
// A rejection has a non-zero Kind; the list is then nil.
func checkMemorize(r MemorizeRequest) ([]content.SpellID, MemorizeRejection) {
	i := slices.Index(r.Prepared, r.Replace)
	switch {
	case !r.Available:
		return nil, MemorizeRejection{Kind: MemorizeFeatureNotAvailable}
	case !shortRestFinished(r.Trigger):
		return nil, MemorizeRejection{Kind: MemorizeNotAfterShortRest}
	case i < 0:
		return nil, MemorizeRejection{Kind: MemorizeNotPrepared, Spell: r.Replace}
	case slices.Contains(r.AlwaysPrep, r.Replace):
		return nil, MemorizeRejection{Kind: MemorizeAlwaysPreparedNotReplaceable, Spell: r.Replace}
	case r.New.IsCantrip():
		return nil, MemorizeRejection{Kind: MemorizeNewSpellIsCantrip, Spell: r.New.ID}
	case !r.Spellbook.Contains(r.New.ID):
		return nil, MemorizeRejection{Kind: MemorizeNewSpellNotInSpellbook, Spell: r.New.ID}
	case int(r.New.Level) > r.HighestLevel:
		return nil, MemorizeRejection{Kind: MemorizeNewSpellAboveHighestLevel, Spell: r.New.ID, Highest: r.HighestLevel}
	case slices.Contains(r.Prepared, r.New.ID):
		return nil, MemorizeRejection{Kind: MemorizeNewSpellAlreadyPrepared, Spell: r.New.ID}
	}
	next := slices.Clone(r.Prepared)
	next[i] = r.New.ID
	return next, MemorizeRejection{}
}

// checkRitualAdept decides whether the spell can be cast as a Ritual by Ritual Adept (SRD
// 5.2.1, "Level 1: Ritual Adept", p. 78): the spellbook can be read, the spell is in it
// and has the Ritual tag. It is advisory; whether the spell is prepared does not matter.
// A rejection has a non-zero Kind.
func checkRitualAdept(book Spellbook, spell WizardSpellFacts, available bool) RitualAdeptRejection {
	switch {
	case !available:
		return RitualAdeptRejection{Kind: RitualAdeptSpellbookUnavailable}
	case !book.Contains(spell.ID):
		return RitualAdeptRejection{Kind: RitualAdeptNotInSpellbook}
	case !spell.Ritual:
		return RitualAdeptRejection{Kind: RitualAdeptNotARitualSpell}
	}
	return RitualAdeptRejection{}
}
