package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// DruidRules is the stateless service of the pure Druid rules of class-druid.md table M. It is a zero-size value; its methods
// are total, deterministic, use no dice and read no state they are not given.
type DruidRules struct{}

// MagicianCheckBonus returns the bonus of the Magician Primal Order to an Intelligence (Arcana) or Intelligence (Nature) check:
// the Wisdom modifier, at least +1 (SRD 5.2.1, "Level 1: Primal Order", p. 42). It is +0 for another order, no order, another
// ability or another skill, so an Intelligence (History) or a Wisdom (Nature) check gets none.
func (DruidRules) MagicianCheckBonus(order PrimalOrder, wisdom ability.Modifier, a ability.Ability, skill content.Skill) modifier.Modifier {
	if order != PrimalOrderMagician || a != ability.Intelligence || skill != content.SkillArcana && skill != content.SkillNature {
		return modifier.Modifier{}
	}
	return modifier.New(max(wisdom.Value(), 1))
}

// RetainedStat is a game statistic a Druid keeps while shape-shifted (SRD 5.2.1, "Rules While Shape-Shifted", p. 43).
type RetainedStat int

const (
	// RetainedCreatureType is the creature type.
	RetainedCreatureType RetainedStat = iota + 1
	// RetainedHitPoints are the Hit Points.
	RetainedHitPoints
	// RetainedHitDice are the Hit Point Dice.
	RetainedHitDice
	// RetainedIntelligenceScore is the Intelligence score.
	RetainedIntelligenceScore
	// RetainedWisdomScore is the Wisdom score.
	RetainedWisdomScore
	// RetainedCharismaScore is the Charisma score.
	RetainedCharismaScore
	// RetainedClassFeatures are the class features.
	RetainedClassFeatures
	// RetainedLanguages are the languages.
	RetainedLanguages
	// RetainedFeats are the feats.
	RetainedFeats
	// RetainedSkillProficiencies are the Druid's skill proficiencies, in addition to the Beast's.
	RetainedSkillProficiencies
	// RetainedSavingThrowProficiencies are the Druid's saving throw proficiencies, in addition to the Beast's.
	RetainedSavingThrowProficiencies
	// RetainedProficiencyBonus is the Proficiency Bonus, used for those proficiencies.
	RetainedProficiencyBonus
)

// ShapeShiftedStats says which statistics are retained and which are replaced (class-druid.md FR-079). Every statistic not
// retained is replaced by the Beast's stat block.
type ShapeShiftedStats struct{ retained []RetainedStat }

// Retained returns the retained statistics in the order of the SRD, as a new slice.
func (s ShapeShiftedStats) Retained() []RetainedStat { return slices.Clone(s.retained) }

// Replaced reports whether a statistic is replaced by the Beast's stat block, that is, not retained.
func (s ShapeShiftedStats) Replaced(stat RetainedStat) bool {
	return !slices.Contains(s.retained, stat)
}

// ShapeShiftedStats returns the retained statistics; every other game statistic is read from the Beast stat block.
func (DruidRules) ShapeShiftedStats() ShapeShiftedStats {
	return ShapeShiftedStats{retained: []RetainedStat{RetainedCreatureType, RetainedHitPoints, RetainedHitDice, RetainedIntelligenceScore,
		RetainedWisdomScore, RetainedCharismaScore, RetainedClassFeatures, RetainedLanguages, RetainedFeats, RetainedSkillProficiencies,
		RetainedSavingThrowProficiencies, RetainedProficiencyBonus}}
}

// ShapeShiftedProficiency says whether the Druid is proficient in a skill or saving throw.
type ShapeShiftedProficiency int

const (
	// ShapeShiftedNotProficient: the Druid is not proficient.
	ShapeShiftedNotProficient ShapeShiftedProficiency = iota
	// ShapeShiftedProficient: the Druid is proficient.
	ShapeShiftedProficient
)

// ShapeShiftedSource says whose modifier a shape-shifted skill or saving throw uses.
type ShapeShiftedSource int

const (
	// ShapeShiftedSourceDruid: the Druid's own modifier (also when the two are equal).
	ShapeShiftedSourceDruid ShapeShiftedSource = iota + 1
	// ShapeShiftedSourceStatBlock: the higher modifier of the Beast's stat block.
	ShapeShiftedSourceStatBlock
)

// ShapeShiftedModifier is the modifier of a skill or saving throw while shape-shifted and whose it is.
type ShapeShiftedModifier struct {
	Modifier modifier.Modifier
	Source   ShapeShiftedSource
}

// ShapeShiftedModifier returns the modifier of a skill or saving throw while shape-shifted: the higher of the Druid's own
// (the ability modifier plus the Proficiency Bonus once when proficient) and the Beast stat block's modifier when it lists one
// (SRD 5.2.1, "Rules While Shape-Shifted", p. 43; "The Bonus Doesn't Stack", p. 8). The Proficiency Bonus is the caller's,
// from the total level (class-druid.md FR-152).
func (DruidRules) ShapeShiftedModifier(abilityModifier ability.Modifier, proficiency ShapeShiftedProficiency, bonus ProficiencyBonus, beast modifier.Modifier, beastLists bool) ShapeShiftedModifier {
	own := abilityModifier.Value()
	if proficiency == ShapeShiftedProficient {
		own += bonus.Value()
	}
	if beastLists && beast.Value() > own {
		return ShapeShiftedModifier{Modifier: beast, Source: ShapeShiftedSourceStatBlock}
	}
	return ShapeShiftedModifier{Modifier: modifier.New(own), Source: ShapeShiftedSourceDruid}
}

// BeastFormCastCheck is whether a spell can be cast in the current state, a closed set. Variants: BeastFormCastCheckAllowed,
// BeastFormCastCheckBlocked.
type BeastFormCastCheck interface{ isBeastFormCastCheck() }

// BeastFormCastCheckAllowed: the spell may be cast.
type BeastFormCastCheckAllowed struct{}

// BeastFormCastCheckBlocked: the spell may not be cast; Reason is DruidCannotCastInBeastForm or DruidMaterialComponentRestricted.
type BeastFormCastCheckBlocked struct{ Reason DruidRejection }

func (BeastFormCastCheckAllowed) isBeastFormCastCheck() {}
func (BeastFormCastCheckBlocked) isBeastFormCastCheck() {}

// materialRestricted reports whether the spell has a Material component with a cost specified or that it consumes.
func materialRestricted(spell content.Spell) bool {
	m, ok := spell.Components().Material()
	return ok && (len(m.CostsGP()) > 0 || m.ConsumesMaterial())
}

// beastFormCastBlock returns the reason a cast is blocked in the state at the level, or 0 when it is allowed; restricted says
// the spell has a costed or consumed Material component.
func beastFormCastBlock(state DruidFeatureState, level int, restricted bool) DruidRejection {
	switch _, inForm := state.ActiveForm(); {
	case !inForm:
		return 0
	case level < 18:
		return DruidCannotCastInBeastForm
	case restricted:
		return DruidMaterialComponentRestricted
	}
	return 0
}

// BeastFormCastCheck returns BeastFormCastCheckAllowed, or BeastFormCastCheckBlocked for a spell cast in a Beast form: with
// DruidCannotCastInBeastForm at Druid level 17 or below and DruidMaterialComponentRestricted from level 18 for a spell with a
// costed or consumed Material component (SRD 5.2.1, "Rules While Shape-Shifted", "Level 18: Beast Spells", p. 43). The host
// calls it before CastSpell and passes CastingContext.canCastSpells false when blocked. Wild Shape never breaks
// Concentration, so no Concentration is read.
func (DruidRules) BeastFormCastCheck(state DruidFeatureState, level DruidLevel, spell content.Spell) BeastFormCastCheck {
	if r := beastFormCastBlock(state, level.Value(), materialRestricted(spell)); r != 0 {
		return BeastFormCastCheckBlocked{Reason: r}
	}
	return BeastFormCastCheckAllowed{}
}

// PotentBonus is the Potent Spellcasting damage bonus of a cantrip, a closed set. Variants: PotentBonusApplied,
// PotentBonusNotApplicable.
type PotentBonus interface{ isPotentBonus() }

// PotentBonusApplied: add Modifier to the cantrip's damage. It is the Wisdom modifier, which can be 0 or negative.
type PotentBonusApplied struct{ Modifier modifier.Modifier }

// PotentBonusNotApplicable: the cantrip gets no bonus; Reason is DruidFeatureNotChosen or DruidNotADruidCantrip.
type PotentBonusNotApplicable struct{ Reason DruidRejection }

func (PotentBonusApplied) isPotentBonus()       {}
func (PotentBonusNotApplicable) isPotentBonus() {}

// PotentSpellcastingBonus returns PotentBonusApplied with the Wisdom modifier for a Druid cantrip, including a cantrip that
// counts as a Druid spell through a feature (the Magician's extra cantrip, Circle Spells; countsAsDruid), and
// PotentBonusNotApplicable for a spell of level 1 or higher or a cantrip of another class, or when Primal Strike was chosen
// (SRD 5.2.1, "Level 7: Elemental Fury", p. 43; "Level 1: Spellcasting", p. 42).
func (DruidRules) PotentSpellcastingBonus(state DruidFeatureState, spell content.Spell, wisdom ability.Modifier, countsAsDruid []content.SpellID) PotentBonus {
	switch {
	case state.ElementalFury() != ElementalFuryPotentSpellcasting:
		return PotentBonusNotApplicable{Reason: DruidFeatureNotChosen}
	case !spell.Level().IsCantrip() || !slices.Contains(spell.Classes(), DruidClassID) && !slices.Contains(countsAsDruid, spell.ID()):
		return PotentBonusNotApplicable{Reason: DruidNotADruidCantrip}
	}
	return PotentBonusApplied{Modifier: modifier.New(wisdom.Value())}
}

// RangeChange is the effect of Improved Potent Spellcasting on a cantrip's range, a closed set. Variants:
// RangeChangeIncreased, RangeChangeUnchanged, RangeChangeNotApplicable.
type RangeChange interface{ isRangeChange() }

// RangeChangeIncreased: the cantrip's range is NewFeet.
type RangeChangeIncreased struct{ NewFeet int }

// RangeChangeUnchanged: the range keeps its printed value.
type RangeChangeUnchanged struct{}

// RangeChangeNotApplicable: Primal Strike was chosen, so the range is not increased; Reason is DruidFeatureNotChosen.
type RangeChangeNotApplicable struct{ Reason DruidRejection }

func (RangeChangeIncreased) isRangeChange()     {}
func (RangeChangeUnchanged) isRangeChange()     {}
func (RangeChangeNotApplicable) isRangeChange() {}

// ImprovedPotentRange returns, from Druid level 15 and only when Potent Spellcasting was chosen, RangeChangeIncreased adding
// 300 feet to a distance range of 10 feet or greater, and RangeChangeUnchanged for Self, Touch, shorter ranges and every other
// range, and at level 14 or lower (SRD 5.2.1, "Level 15: Improved Elemental Fury", p. 43). With Primal Strike chosen (or no
// option) from level 15 it returns RangeChangeNotApplicable.
func (DruidRules) ImprovedPotentRange(state DruidFeatureState, level DruidLevel, r content.SpellRange) RangeChange {
	if level.Value() < 15 {
		return RangeChangeUnchanged{}
	}
	if state.ElementalFury() != ElementalFuryPotentSpellcasting {
		return RangeChangeNotApplicable{Reason: DruidFeatureNotChosen}
	}
	d, ok := r.(content.SpellRangeDistance)
	if !ok {
		return RangeChangeUnchanged{}
	}
	feet := d.Amount()
	if d.Unit() == content.DistanceMiles {
		feet *= 5280
	}
	if feet < 10 {
		return RangeChangeUnchanged{}
	}
	return RangeChangeIncreased{NewFeet: feet + 300}
}

// PrimalStrikeOutcome is what Primal Strike does for an attack, a closed set. Variants: PrimalStrikeOutcomeDamage,
// PrimalStrikeOutcomeIneligible.
type PrimalStrikeOutcome interface{ isPrimalStrikeOutcome() }

// PrimalStrikeOutcomeDamage: the target takes extra Damage of one of the Types, chosen when the attack hits.
type PrimalStrikeOutcomeDamage struct {
	Damage dice.Expression
	Types  []content.DamageType
}

// PrimalStrikeOutcomeIneligible: Primal Strike does not apply; Reason is DruidFeatureNotChosen or DruidAttackNotEligible.
type PrimalStrikeOutcomeIneligible struct{ Reason DruidRejection }

func (PrimalStrikeOutcomeDamage) isPrimalStrikeOutcome()     {}
func (PrimalStrikeOutcomeIneligible) isPrimalStrikeOutcome() {}

// PrimalStrikeDice returns the extra damage of Primal Strike, 1d8 (2d8 from Druid level 15) of Cold, Fire, Lightning or Thunder,
// for a hit with a weapon attack, in Wild Shape or not, or with a Beast form's attack in Wild Shape; and
// PrimalStrikeOutcomeIneligible with DruidFeatureNotChosen when Primal Strike was not chosen and DruidAttackNotEligible for a
// miss, a spell attack, another attack, or a Beast form's attack outside Wild Shape (SRD 5.2.1, "Level 7: Elemental Fury",
// "Level 15: Improved Elemental Fury", p. 43). It does not check the level or the once-per-turn limit.
func (DruidRules) PrimalStrikeDice(state DruidFeatureState, level DruidLevel, source PrimalStrikeAttackSource, attack d20.AttackRollResult) PrimalStrikeOutcome {
	_, inForm := state.ActiveForm()
	switch {
	case state.ElementalFury() != ElementalFuryPrimalStrike:
		return PrimalStrikeOutcomeIneligible{Reason: DruidFeatureNotChosen}
	case attack.Outcome() != d20.Success || source != PrimalStrikeSourceWeapon && (source != PrimalStrikeSourceBeastForm || !inForm):
		return PrimalStrikeOutcomeIneligible{Reason: DruidAttackNotEligible}
	}
	count := 1
	if level.Value() >= 15 {
		count = 2
	}
	return PrimalStrikeOutcomeDamage{Damage: dice.MustExpression(count, dice.D8(), 0), Types: primalStrikeTypes()}
}

func primalStrikeTypes() []content.DamageType {
	return []content.DamageType{content.DamageCold, content.DamageFire, content.DamageLightning, content.DamageThunder}
}

// LandsAidDice returns the damage and healing dice of Land's Aid: 2d6, 3d6 from Druid level 10 and 4d6 from level 14 (SRD
// 5.2.1, "Level 3: Land's Aid", p. 46).
func (DruidRules) LandsAidDice(level DruidLevel) dice.Expression {
	count := 2
	if level.Value() >= 10 {
		count++
	}
	if level.Value() >= 14 {
		count++
	}
	return dice.MustExpression(count, dice.D6(), 0)
}

// NaturalRecoveryBudget returns the combined level of the slots Natural Recovery recovers: half the Druid level, rounded up
// (SRD 5.2.1, "Level 6: Natural Recovery", p. 46).
func (DruidRules) NaturalRecoveryBudget(level DruidLevel) int { return (level.Value() + 1) / 2 }

// CircleSpells returns the Circle Spells of the land at the Druid level: those of the levels 3, 5, 7 and 9 at or below it,
// in order, as a new slice (3 spells at levels 3 and 4, 4 at 5 and 6, 5 at 7 and 8 and 6 from 9 for the bundled lands). It is
// empty below level 3 and for the zero land.
func (DruidRules) CircleSpells(tables DruidFeatureTables, land LandType, level DruidLevel) []content.SpellID {
	return tables.circleSpellsAt(land, level.Value())
}

// FormEligibility says whether a Beast form can be a known form, a closed set. Variants: FormEligibilityEligible,
// FormEligibilityIneligible.
type FormEligibility interface{ isFormEligibility() }

// FormEligibilityEligible: the form can be a known form.
type FormEligibilityEligible struct{}

// FormEligibilityIneligible: the form cannot; Reason is DruidNotABeast, DruidChallengeRatingTooHigh or DruidFlySpeedNotAllowed
// (the first that applies, in that order), or DruidFeatureNotGranted below Druid level 2.
type FormEligibilityIneligible struct{ Reason DruidRejection }

func (FormEligibilityEligible) isFormEligibility()   {}
func (FormEligibilityIneligible) isFormEligibility() {}

// FormEligibility returns whether the form is eligible at the Druid level: a Beast of at most the maximum Challenge Rating,
// with no Fly Speed before the tier that allows it (SRD 5.2.1, "Level 2: Wild Shape", p. 42).
func (DruidRules) FormEligibility(form BeastFormFacts, tables DruidFeatureTables, level DruidLevel) FormEligibility {
	if r := formIneligibility(form, tables, level.Value()); r != 0 {
		return FormEligibilityIneligible{Reason: r}
	}
	return FormEligibilityEligible{}
}

// formIneligibility returns the first reason the form is not eligible, or 0 when it is.
func formIneligibility(form BeastFormFacts, tables DruidFeatureTables, level int) DruidRejection {
	tier, ok := tables.tierAt(level)
	switch {
	case !ok:
		return DruidFeatureNotGranted
	case form.kind != content.CreatureTypeBeast:
		return DruidNotABeast
	case content.CompareChallengeRating(form.rating, tier.MaxChallengeRating) > 0:
		return DruidChallengeRatingTooHigh
	case form.flySpeed && !tier.FlySpeed:
		return DruidFlySpeedNotAllowed
	}
	return 0
}

// DruidicPresence is whether an observer spots a Druidic message, a closed set. Variants: DruidicPresenceAutomaticallySpotted,
// DruidicPresenceRequiresCheck.
type DruidicPresence interface{ isDruidicPresence() }

// DruidicPresenceAutomaticallySpotted: the observer knows Druidic and spots the message with no check.
type DruidicPresenceAutomaticallySpotted struct{}

// DruidicPresenceRequiresCheck: the observer spots the message with a successful check; the consumer rolls. A success spots
// the presence only.
type DruidicPresenceRequiresCheck struct {
	DC      d20.DifficultyClass
	Ability ability.Ability
	Skill   content.Skill
}

func (DruidicPresenceAutomaticallySpotted) isDruidicPresence() {}
func (DruidicPresenceRequiresCheck) isDruidicPresence()        {}

// DruidicContent is whether an observer can read a Druidic message, independent of the presence check.
type DruidicContent int

const (
	// DruidicContentReadable: the observer knows Druidic and reads the message.
	DruidicContentReadable DruidicContent = iota + 1
	// DruidicContentNeedsMagic: the observer cannot decipher the message without magic.
	DruidicContentNeedsMagic
)

// DruidicDetection is what an observer perceives of a Druidic hidden message: the two facts are independent.
type DruidicDetection struct {
	Presence DruidicPresence
	Content  DruidicContent
}

// DruidicDetection returns what an observer who knows the given languages perceives of a hidden message in Druidic: a creature
// that knows Druidic spots it with no check and reads it; any other needs a successful DC 15 Intelligence (Investigation) check
// to spot it and cannot decipher it without magic, whatever the check's result (SRD 5.2.1, "Level 1: Druidic", p. 42).
func (DruidRules) DruidicDetection(languages []content.Language) DruidicDetection {
	if slices.Contains(languages, content.LanguageDruidic) {
		return DruidicDetection{Presence: DruidicPresenceAutomaticallySpotted{}, Content: DruidicContentReadable}
	}
	return DruidicDetection{Presence: DruidicPresenceRequiresCheck{DC: 15, Ability: ability.Intelligence, Skill: content.SkillInvestigation}, Content: DruidicContentNeedsMagic}
}

// ArchdruidYearsPerYearAged is the Longevity of the Archdruid (SRD 5.2.1, "Level 20: Archdruid", p. 43): for every ten years that pass, the
// body ages one year. It is data for the host to apply.
const ArchdruidYearsPerYearAged = 10

const speakWithAnimals content.SpellID = "speak_with_animals"

// Languages returns the languages every Druid knows: Druidic, from level 1 (SRD 5.2.1, "Level 1: Druidic", p. 42), as a new slice.
func (DruidRules) Languages() []content.Language { return []content.Language{content.LanguageDruidic} }

// AlwaysPreparedSpells returns the spells the Druid always has prepared, to supply to spellcasting as always-prepared spells that count as Druid spells but not against
// the prepared limit (SRD 5.2.1, "Level 1: Spellcasting", p. 42): Speak with Animals, then with the Circle of the Land and a land type chosen the Circle Spells of the land
// for the Druid level. It is a new slice.
func (DruidRules) AlwaysPreparedSpells(facts DruidCharacterFacts, state DruidFeatureState, tables DruidFeatureTables) []content.SpellID {
	out := []content.SpellID{speakWithAnimals}
	if facts.hasCircle() && state.land != 0 {
		out = append(out, tables.circleSpellsAt(state.land, facts.lv())...)
	}
	return out
}

// NaturesWardEffect is what Nature's Ward gives: immunity to conditions and the Resistance of the current land. The zero value gives nothing.
type NaturesWardEffect struct {
	// ImmuneTo are the conditions the Druid is immune to: the Poisoned condition from level 10 with the Circle of the Land.
	ImmuneTo []content.ConditionKind
	// Resistance is the damage type of the current land; it is meaningful only when ResistanceReason is 0.
	Resistance content.DamageType
	// ResistanceReason is DruidLandTypeNotChosen when the immunity is given and no land type is chosen, so there is no Resistance; otherwise 0.
	ResistanceReason DruidRejection
}

// NaturesWard returns Nature's Ward: from level 10 with the Circle of the Land, immunity to the Poisoned condition and the Resistance of the land type currently in the state
// (Fire for arid, Cold for polar, Lightning for temperate, Poison for tropical), read from the tables each time and never cached (class-druid.md FR-131, FR-132, FR-138). Below
// level 10 or without the subclass it gives nothing and is no error (FR-157).
func (DruidRules) NaturesWard(facts DruidCharacterFacts, state DruidFeatureState, tables DruidFeatureTables) NaturesWardEffect {
	if !facts.hasCircle() || facts.lv() < 10 {
		return NaturesWardEffect{ImmuneTo: []content.ConditionKind{}}
	}
	effect := NaturesWardEffect{ImmuneTo: []content.ConditionKind{content.ConditionPoisoned}}
	if resistance, ok := tables.Resistance(state.land); ok {
		effect.Resistance = resistance
	} else {
		effect.ResistanceReason = DruidLandTypeNotChosen
	}
	return effect
}
