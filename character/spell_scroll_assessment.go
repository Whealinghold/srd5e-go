package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// ScrollCastTerms are the terms of a cast from a Spell Scroll: the spell, the level
// it is cast at (no slot is spent), the save DC and attack bonus, the
// spellcasting ability, the caster level and the components the scroll waives
// (SRD 5.2.1, "Spell Scroll", p. 244; "Spells Cast from Items", p. 206). It
// specialises SpellFromItem for the scroll. Immutable; the zero value is invalid.
type ScrollCastTerms struct {
	spell       content.SpellID
	castAt      content.SpellLevel
	saveDC      int
	attackBonus int
	ability     ability.Ability
	casterLevel ScrollCasterLevel
}

// SpellID returns the spell on the scroll.
func (t ScrollCastTerms) SpellID() content.SpellID { return t.spell }

// CastAtLevel returns the scroll's spell level, the level the spell is cast at.
func (t ScrollCastTerms) CastAtLevel() content.SpellLevel { return t.castAt }

// SaveDC returns the spell save DC of the cast.
func (t ScrollCastTerms) SaveDC() int { return t.saveDC }

// AttackBonus returns the spell attack bonus of the cast.
func (t ScrollCastTerms) AttackBonus() int { return t.attackBonus }

// SpellcastingAbility returns the ability of the route the reader took.
func (t ScrollCastTerms) SpellcastingAbility() ability.Ability { return t.ability }

// CasterLevel returns ScrollLowestCasterLevel, or ScribedCantripCasterLevel for a
// scribed cantrip.
func (t ScrollCastTerms) CasterLevel() ScrollCasterLevel { return t.casterLevel }

// WaivedComponents returns a new slice holding exactly ComponentMaterial: the
// Verbal and Somatic components still apply.
func (ScrollCastTerms) WaivedComponents() []ComponentKind { return []ComponentKind{ComponentMaterial} }

// CastTerms returns the terms in the form CastSpell takes with CastModeSpellScroll.
// A cantrip scales with the scribe's level for a scribed cantrip and with 1, the
// lowest, otherwise. It returns an error matching srderr.ErrInvalidInput for the
// zero terms.
func (t ScrollCastTerms) CastTerms() (SpellCastTerms, error) {
	scaling := 0
	if t.castAt == 0 {
		scaling = int(ItemCasterLevelLowest)
		if c, ok := t.casterLevel.(ScribedCantripCasterLevel); ok {
			scaling = c.level.Value()
		}
	}
	return NewSpellCastTerms(t.saveDC, t.attackBonus, scaling)
}

// ScrollAssessmentResult is the closed set of outcomes of AssessScrollReading.
// Variants: ScrollReady, ScrollCheckRequired, ScrollNotReadable, ScrollNotOnSpellList,
// ScrollAlreadyUsed.
type ScrollAssessmentResult interface{ isScrollAssessmentResult() }

// ScrollReady: the reader casts the scroll's spell with Terms and no check.
type ScrollReady struct{ terms ScrollCastTerms }

// Terms returns the terms of the cast.
func (r ScrollReady) Terms() ScrollCastTerms { return r.terms }

// ScrollCheckRequired: the spell is above the level the reader can normally cast, so
// the reader makes an ability check against DC, with one of Options.
type ScrollCheckRequired struct {
	dc      int
	options []ScrollCheckOption
	terms   []ScrollCastTerms
}

// DC returns 10 plus the spell level.
func (r ScrollCheckRequired) DC() int { return r.dc }

// Options returns a new slice of the check options, one per candidate route in route order.
func (r ScrollCheckRequired) Options() []ScrollCheckOption { return slices.Clone(r.options) }

// TermsFor returns the provisional terms of a cast made with option and whether
// option is one of the required options.
func (r ScrollCheckRequired) TermsFor(option ScrollCheckOption) (ScrollCastTerms, bool) {
	i := slices.Index(r.options, option)
	if i < 0 {
		return ScrollCastTerms{}, false
	}
	return r.terms[i], true
}

// ScrollNotReadable: the reader cannot read the scroll; nothing is consumed.
type ScrollNotReadable struct{ Reason MagicItemScrollNotReadable }

// ScrollNotOnSpellList: the spell is on none of the reader's spell lists, so the
// scroll is unintelligible; no cast, no check, scroll retained.
type ScrollNotOnSpellList struct{}

// ScrollAlreadyUsed: the scroll was already consumed. It is also a
// ScrollCopyAssessmentResult.
type ScrollAlreadyUsed struct{}

func (ScrollReady) isScrollAssessmentResult()          {}
func (ScrollCheckRequired) isScrollAssessmentResult()  {}
func (ScrollNotReadable) isScrollAssessmentResult()    {}
func (ScrollNotOnSpellList) isScrollAssessmentResult() {}
func (ScrollAlreadyUsed) isScrollAssessmentResult()    {}

// scrollRoute is a candidate route: its check option and the highest spell level it
// casts without a check.
type scrollRoute struct {
	option   ScrollCheckOption
	reliable content.SpellLevel
}

// AssessScrollReading decides what reading scroll takes for the reader profile, given
// the scroll's use state (SRD 5.2.1, "Using a Scroll", p. 205; "Spell Scroll", p.
// 244). A consumed scroll gives ScrollAlreadyUsed and nothing else is evaluated; a
// reader who cannot read it gives ScrollNotReadable; a spell on none of the reader's
// routes gives ScrollNotOnSpellList; a spell some route casts reliably gives
// ScrollReady for the first such route; otherwise ScrollCheckRequired with DC 10 plus
// the spell level. It rolls nothing, mutates nothing and never panics. Any state
// other than ItemConsumed counts as usable.
func AssessScrollReading(profile ScrollReaderProfile, scroll SpellScroll, state MagicItemUseState) ScrollAssessmentResult {
	if state == ItemConsumed {
		return ScrollAlreadyUsed{}
	}
	if r, bad := (ScrollReading{}).CanAttempt(profile.readsLanguage, profile.restricted).(MagicItemScrollNotReadable); bad {
		return ScrollNotReadable{Reason: r}
	}
	var routes []scrollRoute
	for _, r := range profile.routes {
		if slices.Contains(scroll.classes, r.class) {
			routes = append(routes, scrollRoute{ScrollCheckOption{ability: r.ability}, profile.castable})
		}
	}
	if a := profile.anyRoute; !a.IsZero() {
		routes = append(routes, scrollRoute{ScrollCheckOption{ability: a.ability, skill: a.skill}, a.reliable})
	}
	if len(routes) == 0 {
		return ScrollNotOnSpellList{}
	}
	for _, r := range routes {
		if scroll.level <= r.reliable {
			return ScrollReady{terms: scrollTerms(scroll, r.option.ability)}
		}
	}
	required := ScrollCheckRequired{dc: 10 + int(scroll.level)}
	for _, r := range routes {
		required.options = append(required.options, r.option)
		required.terms = append(required.terms, scrollTerms(scroll, r.option.ability))
	}
	return required
}

// scrollTerms builds the terms of casting scroll with spellcasting ability a: the
// table's numbers for a found scroll, the scribe's for a scribed one.
func scrollTerms(scroll SpellScroll, a ability.Ability) ScrollCastTerms {
	row := SpellScrollTableRow(scroll.level)
	t := ScrollCastTerms{spell: scroll.spell, castAt: scroll.level, saveDC: row.saveDC, attackBonus: row.attackBonus, ability: a, casterLevel: ScrollLowestCasterLevel{}}
	if s, ok := scroll.origin.(ScribedScrollOrigin); ok {
		t.saveDC, t.attackBonus = s.stats.saveDC, s.stats.attackBonus
		if scroll.level == 0 {
			t.casterLevel = NewScribedCantripCasterLevel(s.stats.level)
		}
	}
	return t
}
