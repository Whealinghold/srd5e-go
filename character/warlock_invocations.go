package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// CantripEffect classifies what a cantrip does for the Eldritch Invocation prerequisites (class-warlock.md FR-156, table 4.6.8). The
// set is closed, so an attack roll without damage cannot be built. The zero value is invalid.
type CantripEffect int

// The cantrip effects.
const (
	// CantripNoDamage: the cantrip deals no damage.
	CantripNoDamage CantripEffect = iota + 1
	// CantripDamage: the cantrip deals damage without an attack roll.
	CantripDamage
	// CantripDamageViaAttackRoll: the cantrip deals damage on a hit with an attack roll.
	CantripDamageViaAttackRoll
)

// Valid reports whether e is one of the three effects.
func (e CantripEffect) Valid() bool { return e >= CantripNoDamage && e <= CantripDamageViaAttackRoll }

// CantripKind is a kind of known Warlock cantrip an invocation prerequisite asks for. The zero value is invalid.
type CantripKind int

// The cantrip kinds.
const (
	// CantripDealsDamage is met by CantripDamage and CantripDamageViaAttackRoll.
	CantripDealsDamage CantripKind = iota + 1
	// CantripDealsDamageViaAttackRoll is met by CantripDamageViaAttackRoll only.
	CantripDealsDamageViaAttackRoll
)

// String returns "deals_damage", "deals_damage_via_attack_roll" or "unknown".
func (k CantripKind) String() string {
	switch k {
	case CantripDealsDamage:
		return "deals_damage"
	case CantripDealsDamageViaAttackRoll:
		return "deals_damage_via_attack_roll"
	}
	return "unknown"
}

// CantripFacts is the classification of one cantrip for the invocation prerequisites: its id, its effect and its printed range
// (class-warlock.md FR-156). No content spec stores these facts, so the caller supplies them. Build it with NewCantripFacts; the zero
// value is invalid. It is immutable and comparable with ==.
type CantripFacts struct {
	spell      content.SpellID
	effect     CantripEffect
	spellRange content.SpellRange
}

// NewCantripFacts returns the facts of the cantrip. It returns an error matching srderr.ErrInvalidInput for an invalid spell id, an
// invalid effect or a nil range. It never panics.
func NewCantripFacts(spell content.SpellID, effect CantripEffect, spellRange content.SpellRange) (CantripFacts, error) {
	if !spell.Valid() || !effect.Valid() || spellRange == nil {
		return CantripFacts{}, invalidInput("cantrip facts with an invalid spell id, effect or range")
	}
	return CantripFacts{spell: spell, effect: effect, spellRange: spellRange}, nil
}

// Spell returns the id of the cantrip.
func (f CantripFacts) Spell() content.SpellID { return f.spell }

// Effect returns the effect.
func (f CantripFacts) Effect() CantripEffect { return f.effect }

// Range returns the printed range.
func (f CantripFacts) Range() content.SpellRange { return f.spellRange }

// IsKind reports whether the cantrip is of the kind.
func (f CantripFacts) IsKind(k CantripKind) bool {
	switch k {
	case CantripDealsDamage:
		return f.effect == CantripDamage || f.effect == CantripDamageViaAttackRoll
	case CantripDealsDamageViaAttackRoll:
		return f.effect == CantripDamageViaAttackRoll
	}
	return false
}

// HasRangeOfTenFeet reports whether the printed range is a distance of 10 or more feet or any distance in miles; Self, Touch, Sight,
// Unlimited and Special do not qualify (class-warlock.md FR-157).
func (f CantripFacts) HasRangeOfTenFeet() bool {
	d, ok := f.spellRange.(content.SpellRangeDistance)
	return ok && (d.Unit() == content.DistanceMiles || d.Amount() >= 10)
}

// invocationCantripKind is the kind of known cantrip an invocation requires and the cantrip it works on, decided by option id and
// never by the printed note (class-warlock.md section 4.6.1).
func invocationCantripKind(id content.FeatureOptionID) (CantripKind, bool) {
	switch id {
	case "agonizing_blast", "eldritch_spear":
		return CantripDealsDamage, true
	case "repelling_blast":
		return CantripDealsDamageViaAttackRoll, true
	}
	return 0, false
}

// UnmetPrerequisiteKind says which part of an invocation prerequisite is unmet. The zero value is invalid.
type UnmetPrerequisiteKind int

// The kinds of unmet prerequisite.
const (
	// UnmetLevel: the Warlock level is below the minimum.
	UnmetLevel UnmetPrerequisiteKind = iota + 1
	// UnmetOption: a required invocation is not among those chosen.
	UnmetOption
	// UnmetCantrip: no known cantrip is of the required kind.
	UnmetCantrip
)

// UnmetPrerequisite is the first part of an invocation prerequisite the character does not meet (class-warlock.md FR-045 to FR-047):
// the level required, the invocation required or the cantrip kind required, by Kind. Construct it with a keyed literal.
type UnmetPrerequisite struct {
	Kind    UnmetPrerequisiteKind
	Level   int
	Option  content.FeatureOptionID
	Cantrip CantripKind
}

// InvocationEligibility evaluates the prerequisite of the option for a Warlock of level who has chosen the invocations in chosen and
// knows the cantrips in known (class-warlock.md FR-045 to FR-048). It returns the first unmet part and false, or the zero value and
// true when the option has no prerequisite or all of it is met: the level first, then the required invocations in listed order, then
// the cantrip kind. A known cantrip with no CantripFacts is not in known and so counts for nothing.
func InvocationEligibility(opt content.FeatureOption, level WarlockLevel, chosen []content.FeatureOptionID, known []CantripFacts) (UnmetPrerequisite, bool) {
	pre, has := opt.Prerequisite()
	if !has {
		return UnmetPrerequisite{}, true
	}
	if pre.MinClassLevel() > level.n {
		return UnmetPrerequisite{Kind: UnmetLevel, Level: pre.MinClassLevel()}, false
	}
	for _, r := range pre.RequiresOptionIDs() {
		if !slices.Contains(chosen, r) {
			return UnmetPrerequisite{Kind: UnmetOption, Option: r}, false
		}
	}
	if kind, needs := invocationCantripKind(opt.ID()); needs && !slices.ContainsFunc(known, func(f CantripFacts) bool { return f.IsKind(kind) }) {
		return UnmetPrerequisite{Kind: UnmetCantrip, Cantrip: kind}, false
	}
	return UnmetPrerequisite{}, true
}

// InvocationPick is one invocation chosen. A Repeatable option names the cantrip it works on in Cantrip (Agonizing Blast, Eldritch
// Spear, Repelling Blast) or the Origin feat in Feat (Lessons of the First Ones); the other options leave both empty. Construct it
// with a keyed literal.
type InvocationPick struct {
	Option  content.FeatureOptionID
	Cantrip content.SpellID
	Feat    content.FeatID
}

// InvocationProblemKind is one thing wrong with an invocation selection (class-warlock.md section 6.1). The zero value is invalid.
type InvocationProblemKind int

// The invocation problems.
const (
	// InvocationWrongCount: the selection is not the entitled number (Expected and Given).
	InvocationWrongCount InvocationProblemKind = iota + 1
	// InvocationUnknown: the option is not one the class definition lists.
	InvocationUnknown
	// InvocationPrerequisiteUnmet: the prerequisite is not met (Unmet).
	InvocationPrerequisiteUnmet
	// InvocationDuplicateNotRepeatable: a non-Repeatable option is chosen twice.
	InvocationDuplicateNotRepeatable
	// InvocationRepeatChoiceDuplicated: a Repeatable option repeats the same cantrip or feat.
	InvocationRepeatChoiceDuplicated
	// InvocationCantripNotEligible: the cantrip named is not a known cantrip of the right kind (or, for Eldritch Spear, lacks a 10 foot range), or none is named.
	InvocationCantripNotEligible
	// InvocationFeatMissing: Lessons of the First Ones names no feat.
	InvocationFeatMissing
	// InvocationTooManyReplacements: more than one known invocation is dropped (Given).
	InvocationTooManyReplacements
	// InvocationIsPrerequisite: a dropped invocation is a prerequisite of one that stays (Option and Dependent).
	InvocationIsPrerequisite
)

// String returns the snake_case name ("wrong_count", "unknown_invocation", ...) or "unknown".
func (k InvocationProblemKind) String() string {
	names := []string{"wrong_count", "unknown_invocation", "prerequisite_unmet", "duplicate_not_repeatable", "repeat_choice_duplicated",
		"cantrip_not_eligible", "feat_missing", "too_many_replacements", "invocation_is_prerequisite"}
	if k < 1 || int(k) > len(names) {
		return "unknown"
	}
	return names[k-1]
}

// InvocationProblem is one problem of an invocation selection with the figures that explain it. Only the fields the Kind names are set.
type InvocationProblem struct {
	Kind            InvocationProblemKind
	Option          content.FeatureOptionID
	Dependent       content.FeatureOptionID
	Expected, Given int
	Unmet           UnmetPrerequisite
	Cantrip         content.SpellID
	Feat            content.FeatID
}

// InvocationSelection is a validated set of invocations, the full set a Warlock holds. Build it with SelectInvocations; the zero value
// is the empty selection. It holds a slice: read it with Picks.
type InvocationSelection struct{ picks []InvocationPick }

// Picks returns the selection in the order given, as a new slice.
func (s InvocationSelection) Picks() []InvocationPick { return slices.Clone(s.picks) }

// SelectInvocations validates picks, the full set of invocations the Warlock holds afterwards, against the options the class
// definition lists (listed), the number the level entitles it to (entitled, the Eldritch Invocations column of the row), the
// invocations held now (current, from WarlockState.Invocations) and the facts of the known cantrips (known)
// (class-warlock.md FR-043 to FR-053, FR-104). It lists every problem: the wrong count, an option not listed, an unmet prerequisite, a
// repeated non-Repeatable option, a repeat of the same cantrip or feat, a cantrip not known or not of the kind (and, for Eldritch Spear,
// without a range of 10 feet), a Lessons of the First Ones with no feat, more than one invocation dropped, and a dropped invocation that
// a kept one requires. The second result is empty and the first holds the selection when valid. Neither input is altered. That the feat
// of Lessons of the First Ones is an Origin feat is the caller's check, because the feat category is content this function does not read.
func SelectInvocations(level WarlockLevel, entitled int, picks []InvocationPick, current []content.FeatureOptionID, listed []content.FeatureOption, known []CantripFacts) (InvocationSelection, []InvocationProblem) {
	problems := []InvocationProblem{}
	if len(picks) != entitled {
		problems = append(problems, InvocationProblem{Kind: InvocationWrongCount, Expected: entitled, Given: len(picks)})
	}
	chosen := make([]content.FeatureOptionID, len(picks))
	for i, p := range picks {
		chosen[i] = p.Option
	}
	for i, p := range picks {
		oi := slices.IndexFunc(listed, func(o content.FeatureOption) bool { return o.ID() == p.Option })
		if oi < 0 || !slices.Contains(warlockInvocationIDs(), p.Option) {
			problems = append(problems, InvocationProblem{Kind: InvocationUnknown, Option: p.Option})
			continue
		}
		opt := listed[oi]
		if unmet, ok := InvocationEligibility(opt, level, chosen, known); !ok {
			problems = append(problems, InvocationProblem{Kind: InvocationPrerequisiteUnmet, Option: p.Option, Unmet: unmet})
		}
		earlier := slices.IndexFunc(picks[:i], func(q InvocationPick) bool { return q.Option == p.Option })
		switch {
		case earlier >= 0 && !opt.Repeatable():
			problems = append(problems, InvocationProblem{Kind: InvocationDuplicateNotRepeatable, Option: p.Option})
		case earlier >= 0 && slices.ContainsFunc(picks[:i], func(q InvocationPick) bool { return q == p }):
			problems = append(problems, InvocationProblem{Kind: InvocationRepeatChoiceDuplicated, Option: p.Option, Cantrip: p.Cantrip, Feat: p.Feat})
		}
		if kind, works := invocationCantripKind(p.Option); works && !cantripEligible(p, kind, known) {
			problems = append(problems, InvocationProblem{Kind: InvocationCantripNotEligible, Option: p.Option, Cantrip: p.Cantrip})
		}
		if p.Option == "lessons_of_the_first_ones" && p.Feat == "" {
			problems = append(problems, InvocationProblem{Kind: InvocationFeatMissing, Option: p.Option})
		}
	}
	problems = append(problems, replacementProblems(picks, current, listed)...)
	if len(problems) != 0 {
		return InvocationSelection{}, problems
	}
	return InvocationSelection{picks: slices.Clone(picks)}, problems
}

func cantripEligible(p InvocationPick, kind CantripKind, known []CantripFacts) bool {
	i := slices.IndexFunc(known, func(f CantripFacts) bool { return f.spell == p.Cantrip })
	if p.Cantrip == "" || i < 0 || !known[i].IsKind(kind) {
		return false
	}
	return p.Option != "eldritch_spear" || known[i].HasRangeOfTenFeet()
}

// replacementProblems reports the limit of one replacement and the prerequisite rule for the invocations held now (current) that the
// new picks drop (class-warlock.md FR-051, FR-052).
func replacementProblems(picks []InvocationPick, current []content.FeatureOptionID, listed []content.FeatureOption) []InvocationProblem {
	remaining := make([]content.FeatureOptionID, len(picks))
	for i, p := range picks {
		remaining[i] = p.Option
	}
	var dropped []content.FeatureOptionID
	for _, id := range current {
		if i := slices.Index(remaining, id); i >= 0 {
			remaining = slices.Delete(remaining, i, i+1)
		} else {
			dropped = append(dropped, id)
		}
	}
	out := []InvocationProblem{}
	if len(dropped) > 1 {
		out = append(out, InvocationProblem{Kind: InvocationTooManyReplacements, Given: len(dropped)})
	}
	for _, d := range dropped {
		for _, o := range listed {
			pre, has := o.Prerequisite()
			if has && slices.Contains(pre.RequiresOptionIDs(), d) && slices.ContainsFunc(picks, func(p InvocationPick) bool { return p.Option == o.ID() }) &&
				slices.Contains(current, o.ID()) {
				out = append(out, InvocationProblem{Kind: InvocationIsPrerequisite, Option: d, Dependent: o.ID()})
			}
		}
	}
	return out
}

// SpellDieRule says how the die of a granted spell is resolved: rolled, or the maximum (Fiendish Vigor, class-warlock.md FR-063). The
// zero value is read as SpellDieRoll.
type SpellDieRule int

const (
	// SpellDieRoll rolls the die as the spell says.
	SpellDieRoll SpellDieRule = iota
	// SpellDieMaximum takes the highest number on the die without rolling.
	SpellDieMaximum
)

// WarlockSpellGrant is a spell a Warlock feature lets the character cast without a spell slot: the host passes the Spell and the Mode
// to the casting service with the casting class warlock and no slot (class-warlock.md FR-059, FR-060, FR-165). It has no slot field.
type WarlockSpellGrant struct {
	// Spell is the spell to cast.
	Spell content.SpellID
	// Mode is always CastModeFeatureGranted.
	Mode CastMode
	// DieRule is SpellDieMaximum for False Life through Fiendish Vigor and SpellDieRoll otherwise.
	DieRule SpellDieRule
}

// invocationSpell is the spell an invocation lets the Warlock cast without a slot (class-warlock.md table 4.6.3).
func invocationSpell(id content.FeatureOptionID) (content.SpellID, bool) {
	spells := map[content.FeatureOptionID]content.SpellID{"armor_of_shadows": "mage_armor", "ascendant_step": "levitate", "fiendish_vigor": "false_life",
		"mask_of_many_faces": "disguise_self", "master_of_myriad_forms": "alter_self", "misty_visions": "silent_image", "one_with_shadows": "invisibility",
		"otherworldly_leap": "jump", "visions_of_distant_realms": "arcane_eye", "whispers_of_the_grave": "speak_with_dead",
		"gift_of_the_depths": "water_breathing", "pact_of_the_chain": "find_familiar"}
	s, ok := spells[id]
	return s, ok
}

// GrantInvocationSpell lets a Warlock cast the spell of a spell-granting invocation without a spell slot (class-warlock.md FR-059 to
// FR-063, FR-065, FR-089, FR-165, table 4.6.3). It refuses with WarlockInvocationNotKnown for an invocation not known,
// WarlockNotASpellGrant for one that grants no spell, WarlockLightLevelWrong for One with Shadows in Bright Light (the light is the
// host's) and WarlockAlreadyCast for a second Gift of the Depths cast before a Long Rest. Fiendish Vigor yields SpellDieMaximum.
func GrantInvocationSpell(id Identifier, s WarlockState, option content.FeatureOptionID, light content.LightLevel) WarlockTransition {
	spell, grants := invocationSpell(option)
	switch {
	case !s.Knows(option):
		return warlockRefused(WarlockRejection{Kind: WarlockInvocationNotKnown, Option: option})
	case !grants:
		return warlockRefused(WarlockRejection{Kind: WarlockNotASpellGrant, Option: option})
	case option == "one_with_shadows" && light != content.LightDim && light != content.LightDarkness:
		return warlockRefused(WarlockRejection{Kind: WarlockLightLevelWrong, Option: option, LightLevel: light})
	case option == "gift_of_the_depths" && s.depthsCast:
		return warlockRefused(WarlockRejection{Kind: WarlockAlreadyCast, Option: option})
	}
	if option == "gift_of_the_depths" {
		s.depthsCast = true
	}
	grant := WarlockSpellGrant{Spell: spell, Mode: CastModeFeatureGranted}
	if option == "fiendish_vigor" {
		grant.DieRule = SpellDieMaximum
	}
	return warlockGranted(s, grant, InvocationSpellCastWithoutSlot{id: id, option: option, spell: spell})
}

// InvocationEffect is the closed set of standing effects an invocation gives, as data for the host and derived stats. Variants:
// InvocationEffectUnderwater, InvocationEffectDevilsSight, InvocationEffectConcentrationAdvantage, InvocationEffectWitchSight.
type InvocationEffect interface{ isInvocationEffect() }

// InvocationEffectUnderwater is Gift of the Depths: breathe underwater and a Swim Speed equal to the Speed (SRD 5.2.1, p. 73).
type InvocationEffectUnderwater struct{}

// InvocationEffectDevilsSight is Devil's Sight: see normally in Dim Light and Darkness, magical and nonmagical, within RangeFeet.
type InvocationEffectDevilsSight struct{ RangeFeet int }

// InvocationEffectConcentrationAdvantage is Eldritch Mind: Advantage on the Constitution saving throws to maintain Concentration, and
// on no other roll.
type InvocationEffectConcentrationAdvantage struct{ Circumstance modifier.RollCircumstance }

// InvocationEffectWitchSight is Witch Sight: Truesight with a range of RangeFeet.
type InvocationEffectWitchSight struct{ RangeFeet int }

func (InvocationEffectUnderwater) isInvocationEffect()             {}
func (InvocationEffectDevilsSight) isInvocationEffect()            {}
func (InvocationEffectConcentrationAdvantage) isInvocationEffect() {}
func (InvocationEffectWitchSight) isInvocationEffect()             {}

// DeriveInvocationEffects returns the standing effects of the invocations known, in the order Gift of the Depths, Devil's Sight,
// Eldritch Mind, Witch Sight, and none for an invocation not known (class-warlock.md FR-064, FR-066 to FR-068). It returns a new slice.
func DeriveInvocationEffects(s WarlockState) []InvocationEffect {
	out := []InvocationEffect{}
	if s.Knows("gift_of_the_depths") {
		out = append(out, InvocationEffectUnderwater{})
	}
	if s.Knows("devils_sight") {
		out = append(out, InvocationEffectDevilsSight{RangeFeet: 120})
	}
	if s.Knows("eldritch_mind") {
		out = append(out, InvocationEffectConcentrationAdvantage{Circumstance: modifier.Advantage{}})
	}
	if s.Knows("witch_sight") {
		out = append(out, InvocationEffectWitchSight{RangeFeet: 30})
	}
	return out
}

// AgonizingBlastBonus returns the Charisma modifier Agonizing Blast adds to a damage roll of cantrip, and false when cantrip is not
// the cantrip chosen for the invocation. The modifier is added as written, even when negative (SRD 5.2.1, "Agonizing Blast", p. 72;
// class-warlock.md FR-055). The caller holds the chosen cantrip from the InvocationPick; WarlockState keeps only a count.
func AgonizingBlastBonus(chosen, cantrip content.SpellID, charismaModifier int) (int, bool) {
	if chosen != cantrip {
		return 0, false
	}
	return charismaModifier, true
}

// RepellingBlastPushFeet is the distance in feet Repelling Blast pushes a hit creature.
const RepellingBlastPushFeet = 10

// RepellingBlastPush returns the feet a hit by the chosen cantrip pushes a target of size, 10 for Large or smaller, and false for a
// Huge or larger or invalid size, or when cantrip is not the chosen one (SRD 5.2.1, "Repelling Blast", p. 74; class-warlock.md FR-058).
func RepellingBlastPush(chosen, cantrip content.SpellID, size content.CreatureSize) (int, bool) {
	if chosen != cantrip || !size.Valid() || size > content.SizeLarge {
		return 0, false
	}
	return RepellingBlastPushFeet, true
}

// EldritchSpearRangeFor returns the feet Eldritch Spear adds to the range of cantrip, and false when cantrip is not the chosen one
// (SRD 5.2.1, "Eldritch Spear", p. 73; class-warlock.md FR-056).
func EldritchSpearRangeFor(chosen, cantrip content.SpellID, level WarlockLevel) (int, bool) {
	if chosen != cantrip {
		return 0, false
	}
	return EldritchSpearRangeIncrease(level), true
}
