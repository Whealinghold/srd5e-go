package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// passiveBase is the 10 of a passive score (SRD 5.2.1, "Passive Perception", p. 186).
const passiveBase = 10

// advantagePassiveStep is the 5 Advantage adds to a passive score and
// Disadvantage takes from it (SRD 5.2.1, "Passive Perception", p. 186).
const advantagePassiveStep = 5

// DerivedStatsInput is everything DerivedStatsCalculator needs, as plain values
// and resolved content. Armor and Shield are the zero content.Armor when absent.
// Construct it with a keyed literal.
type DerivedStatsInput struct {
	// ClassLevels, AbilityScores, HitPointMaximum and InitialClass are required.
	ClassLevels   ClassLevels
	AbilityScores ability.Scores
	// HitPointMaximum is the stored maximum, without any bonus in HitPointMaximumBonuses.
	HitPointMaximum HitPointMaximum
	HitDicePool     HitDicePool
	// InitialClass is the definition of ClassLevels.Initial().
	InitialClass content.ClassDefinition
	// Armor is the body armor worn, Shield the Shield wielded; the zero Armor means none.
	Armor, Shield           content.Armor
	SkillProficiencies      SkillProficiencies
	ChosenBasis             ArmorClassBasisChoice
	AlternativeBases        []AbilityBasedArmorClassBasis
	MulticlassArmorTraining ArmorTraining
	FlatArmorClassBonuses   []FlatBonus
	// ArmorClassFloor is a minimum on the finished Armor Class; 0 means none.
	ArmorClassFloor        int
	FlatSavingThrowBonuses []FlatSavingThrowBonus
	AdditionalSavingThrows SavingThrowProficiencies
	HitPointMaximumBonuses []HitPointMaximumBonus
	// PerceptionRollMode is the already-resolved mode of Wisdom (Perception) checks.
	PerceptionRollMode modifier.RollMode
}

// DerivedStatsCalculation is the closed set of outcomes of
// DerivedStatsCalculator.Calculate. Variants: DerivedStatsComputed,
// DerivedStatsBasisNotApplicable, DerivedStatsSlotMismatch.
type DerivedStatsCalculation interface{ isDerivedStatsCalculation() }

// DerivedStatsComputed is the outcome for a calculation that succeeded.
type DerivedStatsComputed struct{ Stats DerivedStats }

// DerivedStatsBasisNotApplicable is the outcome for a chosen Armor Class basis
// that cannot be used.
type DerivedStatsBasisNotApplicable struct {
	Choice ArmorClassBasisChoice
	Reason ArmorClassBasisRejection
}

// DerivedStatsSlotMismatch is the outcome for an armor whose rule does not suit
// the slot it was supplied for.
type DerivedStatsSlotMismatch struct {
	Key      content.Key
	Expected ArmorSlot
}

func (DerivedStatsComputed) isDerivedStatsCalculation()           {}
func (DerivedStatsBasisNotApplicable) isDerivedStatsCalculation() {}
func (DerivedStatsSlotMismatch) isDerivedStatsCalculation()       {}

// DerivedStatsCalculator is the stateless domain service that computes the
// numbers of a character sheet from plain values. It is pure: no ports, no
// signals. The zero value is ready to use and safe for concurrent use.
type DerivedStatsCalculator struct{}

// Calculate computes the derived stats of in (SRD 5.2.1, "Fill In Numbers", p.
// 22). It returns a DerivedStatsSlotMismatch when an armor's rule does not suit
// its slot (the armor first), then a DerivedStatsBasisNotApplicable when the
// chosen basis cannot be used (an alternative the input does not supply is
// reported before the applicability matrix), otherwise a DerivedStatsComputed.
// These are rule outcomes, not errors.
//
// It returns an error matching srderr.ErrInvalidInput, and no calculation, for
// an invalid input: a zero required value, a negative floor, a zero bonus or
// basis, two alternative bases with one id, or an unknown roll mode. It never
// panics.
func (DerivedStatsCalculator) Calculate(in DerivedStatsInput) (DerivedStatsCalculation, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	if in.InitialClass.ID() == "" {
		return nil, invalidInput("zero initial class")
	}
	hasArmor, hasShield := in.Armor != (content.Armor{}), in.Shield != (content.Armor{})
	if hasArmor && !armorFitsSlot(in.Armor, BodyArmorSlot) {
		return DerivedStatsSlotMismatch{Key: in.Armor.Key(), Expected: BodyArmorSlot}, nil
	}
	if hasShield && !armorFitsSlot(in.Shield, ShieldSlot) {
		return DerivedStatsSlotMismatch{Key: in.Shield.Key(), Expected: ShieldSlot}, nil
	}

	trained := classArmorTraining(in.InitialClass).Union(in.MulticlassArmorTraining)
	var worn []content.ArmorCategory
	if hasArmor {
		worn = append(worn, in.Armor.Category())
	}
	if hasShield {
		worn = append(worn, content.ArmorShield)
	}
	effects, err := ArmorTrainingRules{}.Assess(worn, trained.Categories())
	if err != nil {
		return nil, err
	}

	total := in.ClassLevels.TotalLevel()
	prof := ProficiencyBonusFor(total)
	ac, basis, rejected := in.armorClass(hasArmor, hasShield, trained)
	if rejected != nil {
		return *rejected, nil
	}

	hp := []BreakdownTerm{{Modifier: modifier.New(in.HitPointMaximum.n), Source: StoredHitPointMaximumTerm}}
	for _, b := range in.HitPointMaximumBonuses {
		hp = append(hp, BreakdownTerm{Modifier: modifier.New(b.ContributionFor(in.ClassLevels)), Source: HitPointMaximumBonusTerm})
	}
	skills := in.skillModifiers(prof)
	return DerivedStatsComputed{Stats: DerivedStats{
		totalLevel:       total,
		proficiencyBonus: prof,
		hitPoints:        breakdownOf(hp),
		hitDicePool:      in.HitDicePool,
		armorClass:       breakdownOf(ac),
		armorClassBasis:  basis,
		training:         effects,
		savingThrows:     in.savingThrowModifiers(prof),
		skills:           skills,
		passivePerception: PassivePerception{breakdown: breakdownOf(slices.Concat(
			[]BreakdownTerm{{Modifier: modifier.New(passiveBase), Source: BaseValueTerm}},
			skills[content.SkillPerception-content.SkillAcrobatics].breakdown.terms,
			in.rollModeTerms(),
		))},
	}}, nil
}

// validate checks the values that do not depend on content.
func (in DerivedStatsInput) validate() error {
	switch {
	case in.ClassLevels.IsZero():
		return invalidInput("no class levels")
	case in.AbilityScores.IsZero():
		return invalidInput("zero ability scores")
	case in.HitPointMaximum.IsZero():
		return invalidInput("zero Hit Point maximum")
	case in.ArmorClassFloor < 0:
		return invalidInput("negative Armor Class floor")
	case in.PerceptionRollMode < modifier.Normal || in.PerceptionRollMode > modifier.WithDisadvantage:
		return invalidInput("unknown roll mode")
	case slices.ContainsFunc(in.FlatArmorClassBonuses, FlatBonus.IsZero):
		return invalidInput("flat bonus without a source")
	case slices.ContainsFunc(in.FlatSavingThrowBonuses, FlatSavingThrowBonus.IsZero):
		return invalidInput("flat saving throw bonus without a source")
	case slices.ContainsFunc(in.HitPointMaximumBonuses, HitPointMaximumBonus.IsZero):
		return invalidInput("zero Hit Point maximum bonus")
	case slices.ContainsFunc(in.AlternativeBases, AbilityBasedArmorClassBasis.IsZero):
		return invalidInput("zero alternative basis")
	}
	switch in.ChosenBasis.(type) {
	case nil, ChooseStandard, ChooseArmored, ChooseAlternative:
	default:
		return invalidInput("unknown basis choice")
	}
	seen := make(map[content.FeatureID]bool, len(in.AlternativeBases))
	for _, b := range in.AlternativeBases {
		if seen[b.id] {
			return invalidInput("two alternative bases with one id")
		}
		seen[b.id] = true
	}
	return nil
}

// candidate is an applicable Armor Class basis with the terms of its base.
type candidate struct {
	basis ArmorClassBasis
	terms []BreakdownTerm
}

func (c candidate) total() int {
	n := 0
	for _, t := range c.terms {
		n += t.Modifier.Value()
	}
	return n
}

// matches reports whether the candidate is the basis the choice names.
func (c candidate) matches(choice ArmorClassBasisChoice) bool {
	switch ch := choice.(type) {
	case ChooseStandard:
		_, ok := c.basis.(StandardArmorClassBasis)
		return ok
	case ChooseArmored:
		_, ok := c.basis.(ArmoredArmorClassBasis)
		return ok
	case ChooseAlternative:
		alt, ok := c.basis.(AbilityBasedArmorClassBasis)
		return ok && alt.id == ch.ID
	}
	return false
}

// armorClass returns the terms of Armor Class, the basis used and, when the
// chosen basis cannot be used, the outcome to return instead.
func (in DerivedStatsInput) armorClass(hasArmor, hasShield bool, trained ArmorTraining) ([]BreakdownTerm, ArmorClassBasis, *DerivedStatsBasisNotApplicable) {
	dex := in.AbilityScores.Modifier(ability.Dexterity).Value()
	dexTerm := func(v int) BreakdownTerm {
		return BreakdownTerm{Modifier: modifier.New(v), Source: AbilityModifierTerm, Ability: ability.Dexterity}
	}
	var cands []candidate
	if hasArmor {
		rule, _ := in.Armor.ArmorClass().(content.BaseRule) // the slot check proved the rule is a BaseRule
		base := ArmorRules{}.BaseArmorClass(rule, in.AbilityScores.Modifier(ability.Dexterity))
		terms := []BreakdownTerm{{Modifier: modifier.New(base.ListedBase()), Source: ArmorBaseTerm}}
		if base.DexterityContribution() != 0 {
			terms = append(terms, dexTerm(base.DexterityContribution()))
		}
		cands = append(cands, candidate{basis: ArmoredArmorClassBasis{Armor: in.Armor}, terms: terms})
	} else {
		cands = append(cands, candidate{basis: StandardArmorClassBasis{}, terms: []BreakdownTerm{
			{Modifier: modifier.New(DefaultArmorClassBaseValue), Source: BaseValueTerm}, dexTerm(dex),
		}})
		for _, alt := range in.AlternativeBases {
			if hasShield && !alt.shieldAllowed {
				continue
			}
			terms := []BreakdownTerm{{Modifier: modifier.New(alt.baseValue), Source: BaseValueTerm}, dexTerm(dex)}
			for _, a := range alt.abilities {
				terms = append(terms, BreakdownTerm{Modifier: modifier.New(in.AbilityScores.Modifier(a).Value()), Source: AbilityModifierTerm, Ability: a})
			}
			cands = append(cands, candidate{basis: alt, terms: terms})
		}
	}

	var used candidate
	if choice := in.ChosenBasis; choice != nil {
		if alt, ok := choice.(ChooseAlternative); ok && !slices.ContainsFunc(in.AlternativeBases, func(b AbilityBasedArmorClassBasis) bool { return b.id == alt.ID }) {
			return nil, nil, &DerivedStatsBasisNotApplicable{Choice: choice, Reason: BasisAlternativeNotSupplied}
		}
		i := slices.IndexFunc(cands, func(c candidate) bool { return c.matches(choice) })
		if i < 0 {
			return nil, nil, &DerivedStatsBasisNotApplicable{Choice: choice, Reason: BasisNotApplicable}
		}
		used = cands[i]
	} else {
		used = cands[0]
		for _, c := range cands[1:] {
			if c.total() > used.total() {
				used = c
			}
		}
	}

	terms := slices.Clone(used.terms)
	if hasShield && trained.Has(content.ArmorShield) {
		bonus, _ := in.Shield.ArmorClass().(content.BonusRule) // the slot check proved the rule is a BonusRule
		terms = append(terms, BreakdownTerm{Modifier: modifier.New(bonus.Bonus()), Source: ShieldBonusTerm})
	}
	for _, b := range in.FlatArmorClassBonuses {
		terms = append(terms, BreakdownTerm{Modifier: b.modifier, Source: FlatBonusTerm, Origin: b.source})
	}
	if sum := (candidate{terms: terms}).total(); sum < in.ArmorClassFloor {
		terms = append(terms, BreakdownTerm{Modifier: modifier.New(in.ArmorClassFloor - sum), Source: FloorTerm})
	}
	return terms, used.basis, nil
}

func (in DerivedStatsInput) savingThrowModifiers(prof ProficiencyBonus) []SavingThrowModifier {
	proficient := classSavingThrows(in.InitialClass).Union(in.AdditionalSavingThrows)
	var out []SavingThrowModifier
	for _, a := range ability.All() {
		terms := []BreakdownTerm{{Modifier: modifier.New(in.AbilityScores.Modifier(a).Value()), Source: AbilityModifierTerm, Ability: a}}
		if proficient.Has(a) {
			terms = append(terms, BreakdownTerm{Modifier: modifier.New(prof.n), Source: ProficiencyBonusTerm})
		}
		for _, b := range in.FlatSavingThrowBonuses {
			if b.Covers(a) {
				terms = append(terms, BreakdownTerm{Modifier: b.bonus.modifier, Source: FlatBonusTerm, Origin: b.bonus.source})
			}
		}
		out = append(out, SavingThrowModifier{ability: a, breakdown: breakdownOf(terms)})
	}
	return out
}

func (in DerivedStatsInput) skillModifiers(prof ProficiencyBonus) []SkillModifier {
	var out []SkillModifier
	for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
		a := s.Ability()
		terms := []BreakdownTerm{{Modifier: modifier.New(in.AbilityScores.Modifier(a).Value()), Source: AbilityModifierTerm, Ability: a}}
		switch {
		case in.SkillProficiencies.HasExpertise(s):
			terms = append(terms, BreakdownTerm{Modifier: modifier.New(2 * prof.n), Source: ExpertiseTerm})
		case in.SkillProficiencies.Proficient(s):
			terms = append(terms, BreakdownTerm{Modifier: modifier.New(prof.n), Source: ProficiencyBonusTerm})
		}
		out = append(out, SkillModifier{skill: s, breakdown: breakdownOf(terms)})
	}
	return out
}

// rollModeTerms returns the term Advantage or Disadvantage adds to Passive
// Perception, none for a normal roll (SRD 5.2.1, p. 186).
func (in DerivedStatsInput) rollModeTerms() []BreakdownTerm {
	switch in.PerceptionRollMode { //nolint:exhaustive // a normal roll adds nothing
	case modifier.WithAdvantage:
		return []BreakdownTerm{{Modifier: modifier.New(advantagePassiveStep), Source: RollModeTerm}}
	case modifier.WithDisadvantage:
		return []BreakdownTerm{{Modifier: modifier.New(-advantagePassiveStep), Source: RollModeTerm}}
	}
	return nil
}

// breakdownOf wraps terms the calculator built itself, which are valid by
// construction (a test checks every output against NewModifierBreakdown).
func breakdownOf(terms []BreakdownTerm) ModifierBreakdown {
	return ModifierBreakdown{terms: terms}
}

// DerivedStats is the numbers of a character sheet and the terms that produced
// each. It is immutable and every collection it exposes is a copy; it holds
// slices, so it is not comparable with ==. The zero value is invalid. Safe for
// concurrent use.
type DerivedStats struct {
	totalLevel        Level
	proficiencyBonus  ProficiencyBonus
	hitPoints         ModifierBreakdown
	hitDicePool       HitDicePool
	armorClass        ModifierBreakdown
	armorClassBasis   ArmorClassBasis
	training          ArmorTrainingEffects
	savingThrows      []SavingThrowModifier
	skills            []SkillModifier
	passivePerception PassivePerception
}

// TotalLevel returns the sum of the class levels.
func (s DerivedStats) TotalLevel() Level { return s.totalLevel }

// ProficiencyBonus returns the Proficiency Bonus of the total level.
func (s DerivedStats) ProficiencyBonus() ProficiencyBonus { return s.proficiencyBonus }

// HitPointMaximum returns the reported Hit Point maximum: the stored maximum
// plus the bonuses, the total of HitPointMaximumBreakdown.
func (s DerivedStats) HitPointMaximum() HitPointMaximum {
	return HitPointMaximum{n: s.hitPoints.Total().Value()}
}

// HitPointMaximumBreakdown returns the stored maximum and one term per bonus.
func (s DerivedStats) HitPointMaximumBreakdown() ModifierBreakdown { return s.hitPoints }

// HitDicePool returns the Hit Point Dice, the command's pool unchanged.
func (s DerivedStats) HitDicePool() HitDicePool { return s.hitDicePool }

// ArmorClass returns the Armor Class, the total of ArmorClassBreakdown.
func (s DerivedStats) ArmorClass() d20.ArmorClass {
	return d20.ArmorClass(s.armorClass.Total().Value())
}

// ArmorClassBreakdown returns the terms of Armor Class.
func (s DerivedStats) ArmorClassBreakdown() ModifierBreakdown { return s.armorClass }

// ArmorClassBasis returns the basis the Armor Class used.
func (s DerivedStats) ArmorClassBasis() ArmorClassBasis { return s.armorClassBasis }

// ArmorTrainingGaps returns a new slice of the worn categories without
// training, in the order Light, Medium, Heavy, Shield. A Shield in it added
// nothing to Armor Class; worn armor kept its base (package reading, SRD 5.2.1,
// p. 92 names an Armor Class loss only for the Shield).
func (s DerivedStats) ArmorTrainingGaps() []content.ArmorCategory { return s.training.Untrained() }

// ArmorTrainingCircumstances returns the Disadvantage that untrained body armor
// gives on D20 Tests involving the abilities in involved, or nil (SRD 5.2.1,
// "Armor Training", p. 177). It is never a term of a breakdown.
func (s DerivedStats) ArmorTrainingCircumstances(involved []ability.Ability) []modifier.RollCircumstance {
	return s.training.RollCircumstancesFor(involved)
}

// StealthCircumstances returns the Disadvantage that worn armor with Stealth
// Disadvantage gives on Dexterity (Stealth) checks, or nil, whatever the
// training (SRD 5.2.1, "Armor", p. 92). It is never a term of a breakdown.
func (DerivedStats) StealthCircumstances(worn []content.Armor) []modifier.RollCircumstance {
	return ArmorRules{}.StealthCircumstances(worn)
}

// SavingThrows returns a new slice of the six saving throw modifiers in SRD
// ability order.
func (s DerivedStats) SavingThrows() []SavingThrowModifier { return slices.Clone(s.savingThrows) }

// SavingThrow returns the saving throw modifier of a; the zero value for an
// invalid ability.
func (s DerivedStats) SavingThrow(a ability.Ability) SavingThrowModifier {
	if !validAbility(a) {
		return SavingThrowModifier{}
	}
	return s.savingThrows[a-ability.Strength]
}

// Skills returns a new slice of the 18 skill modifiers in Skill order.
func (s DerivedStats) Skills() []SkillModifier { return slices.Clone(s.skills) }

// Skill returns the modifier of sk; the zero value for an invalid skill.
func (s DerivedStats) Skill(sk content.Skill) SkillModifier {
	if !sk.Valid() {
		return SkillModifier{}
	}
	return s.skills[sk-content.SkillAcrobatics]
}

// PassivePerception returns the Passive Perception score.
func (s DerivedStats) PassivePerception() PassivePerception { return s.passivePerception }
