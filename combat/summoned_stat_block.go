package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The cast levels a summoning spell accepts (SRD 5.2.1, "Spell Level", p. 104).
const (
	minCastLevel = 1
	maxCastLevel = 9
	// maxSummonerFact bounds every number of SummonerSpellFacts so the expressions built from them cannot overflow.
	maxSummonerFact = 1000
)

// SummonerSpellFacts are the numbers of the summoner a stat block reads: the spell attack bonus, the spell save DC, the
// spellcasting ability modifier and the Proficiency Bonus (FR-036). The stat block keeps no copy of the summoner's ability
// scores. Build it with NewSummonerSpellFacts; the zero value is invalid. It is comparable and immutable. Safe for concurrent
// use.
type SummonerSpellFacts struct{ attackBonus, saveDC, modifier, proficiencyBonus int }

// NewSummonerSpellFacts returns the facts. It returns an error matching srderr.ErrInvalidInput for a save DC or Proficiency
// Bonus below 1 or a number whose magnitude is above 1000. It never panics.
func NewSummonerSpellFacts(attackBonus, saveDC, modifier, proficiencyBonus int) (SummonerSpellFacts, error) {
	if saveDC < 1 || proficiencyBonus < 1 || max(saveDC, proficiencyBonus, attackBonus, modifier) > maxSummonerFact ||
		min(attackBonus, modifier) < -maxSummonerFact {
		return SummonerSpellFacts{}, invalidInput("summoner spell facts need a save DC and Proficiency Bonus of 1 or more and numbers within 1000")
	}
	return SummonerSpellFacts{attackBonus, saveDC, modifier, proficiencyBonus}, nil
}

// IsZero reports whether f is the invalid zero value.
func (f SummonerSpellFacts) IsZero() bool { return f.proficiencyBonus == 0 }

// AttackBonus returns the summoner's spell attack bonus.
func (f SummonerSpellFacts) AttackBonus() int { return f.attackBonus }

// SaveDC returns the summoner's spell save DC.
func (f SummonerSpellFacts) SaveDC() int { return f.saveDC }

// Modifier returns the summoner's spellcasting ability modifier.
func (f SummonerSpellFacts) Modifier() int { return f.modifier }

// ProficiencyBonus returns the summoner's Proficiency Bonus.
func (f SummonerSpellFacts) ProficiencyBonus() int { return f.proficiencyBonus }

// SummonedAttackKind names an attack, trait action or bonus action of a summoned stat block that Table S fixes. The zero value is
// invalid.
type SummonedAttackKind int

// The kinds.
const (
	// SummonedAttackSlam: the Slam of an Animated Object (Force).
	SummonedAttackSlam SummonedAttackKind = iota + 1
	// SummonedAttackPoisonJab: the Poison Jab of a Giant Insect.
	SummonedAttackPoisonJab
	// SummonedAttackWebBolt: the Web Bolt of a spider, which reduces the target's Speed to 0.
	SummonedAttackWebBolt
	// SummonedAttackVenomousSpew: the Venomous Spew bonus action of a centipede.
	SummonedAttackVenomousSpew
	// SummonedAttackRend: the Rend of a Draconic Spirit.
	SummonedAttackRend
	// SummonedAttackBreathWeapon: the Breath Weapon of a Draconic Spirit.
	SummonedAttackBreathWeapon
	// SummonedAttackOtherworldlySlam: the Slam of an Otherworldly Steed.
	SummonedAttackOtherworldlySlam
	// SummonedAttackFellGlare: the Fell Glare bonus action of a Fiend steed.
	SummonedAttackFellGlare
	// SummonedAttackFeyStep: the Fey Step bonus action of a Fey steed.
	SummonedAttackFeyStep
	// SummonedAttackHealingTouch: the Healing Touch bonus action of a Celestial steed.
	SummonedAttackHealingTouch
)

// Valid reports whether k is one of the kinds.
func (k SummonedAttackKind) Valid() bool {
	return k >= SummonedAttackSlam && k <= SummonedAttackHealingTouch
}

// String returns the snake_case name of k, or "unknown".
func (k SummonedAttackKind) String() string {
	return enumName([]string{"slam", "poison_jab", "web_bolt", "venomous_spew", "rend", "breath_weapon", "otherworldly_slam", "fell_glare", "fey_step", "healing_touch"}, int(k))
}

// IsBonusAction reports whether k is taken with a Bonus Action.
func (k SummonedAttackKind) IsBonusAction() bool {
	return k == SummonedAttackVenomousSpew || k == SummonedAttackFellGlare || k == SummonedAttackFeyStep || k == SummonedAttackHealingTouch
}

// SummonedAttack is one entry of a summoned stat block with the numbers the summoner's facts and the cast level fix. It is
// comparable and immutable. The zero value is invalid.
type SummonedAttack struct {
	kind                 SummonedAttackKind
	attackBonus, saveDC  int
	damage, extra        dice.Expression
	damageType, extraTyp content.DamageType
	rangeFeet            int
}

// Kind returns the kind of the attack.
func (a SummonedAttack) Kind() SummonedAttackKind { return a.kind }

// AttackBonus returns the attack bonus, the summoner's spell attack bonus, or 0 for an entry that is not an attack roll.
func (a SummonedAttack) AttackBonus() int { return a.attackBonus }

// SaveDC returns the saving throw DC, the summoner's spell save DC, or 0 for an entry that forces no save.
func (a SummonedAttack) SaveDC() int { return a.saveDC }

// Damage returns the damage or healing expression, the zero Expression (Count 0) when the entry has none.
func (a SummonedAttack) Damage() dice.Expression { return a.damage }

// DamageType returns the Damage Type of Damage, 0 when there is none (healing included).
func (a SummonedAttack) DamageType() content.DamageType { return a.damageType }

// ExtraDamage returns the damage dealt in addition (the Poison of a Poison Jab), the zero Expression when there is none.
func (a SummonedAttack) ExtraDamage() dice.Expression { return a.extra }

// ExtraDamageType returns the Damage Type of ExtraDamage, 0 when there is none.
func (a SummonedAttack) ExtraDamageType() content.DamageType { return a.extraTyp }

// RangeFeet returns the reach, range or distance of the entry in feet, 0 for none.
func (a SummonedAttack) RangeFeet() int { return a.rangeFeet }

// SummonedStatBlock is the numbers of a summoned creature that depend on the spell, the cast level and the summoner (Table S),
// or the numbers of the content stat block a spell names. It holds no copy of the summoner's ability scores. Build it with the
// stat block constructors; the zero value is invalid. Not comparable with ==; use Equal. Safe for concurrent use.
type SummonedStatBlock struct {
	monster                                     content.MonsterID
	size                                        content.CreatureSize
	typ                                         content.CreatureType
	armorClass, hitPointMaximum                 int
	hitDice                                     dice.Expression
	walk, climb, fly, swim                      int
	resistances                                 []content.DamageType
	conditionImmunities                         []content.ConditionKind
	attacks                                     []SummonedAttack
	multiattack, proficiencyBonus, experience   int
	forbidden                                   []content.SpellID
	cannotGainLevels, cannotTakeRests, lifeBond bool
	telepathy                                   TelepathyCapability
}

// IsZero reports whether b is the invalid zero value.
func (b SummonedStatBlock) IsZero() bool { return b.monster == "" }

// Monster returns the id of the monster the creature is. Its CombatantRef is MonsterCombatant with this id (FR-008).
func (b SummonedStatBlock) Monster() content.MonsterID { return b.monster }

// Size returns the creature size.
func (b SummonedStatBlock) Size() content.CreatureSize { return b.size }

// Type returns the creature type.
func (b SummonedStatBlock) Type() content.CreatureType { return b.typ }

// ArmorClass returns the Armor Class.
func (b SummonedStatBlock) ArmorClass() int { return b.armorClass }

// HitPointMaximum returns the Hit Point maximum.
func (b SummonedStatBlock) HitPointMaximum() int { return b.hitPointMaximum }

// HitDice returns the Hit Dice of the creature, the zero Expression when the stat block gives none.
func (b SummonedStatBlock) HitDice() dice.Expression { return b.hitDice }

// Speed returns the Speed of the mode in feet, 0 when the creature has none. Burrow is always 0.
func (b SummonedStatBlock) Speed(mode content.SpeedMode) int {
	switch mode {
	case content.SpeedWalk:
		return b.walk
	case content.SpeedClimb:
		return b.climb
	case content.SpeedFly:
		return b.fly
	case content.SpeedSwim:
		return b.swim
	case content.SpeedBurrow:
		return 0
	}
	return 0
}

// Resistances returns a copy of the Damage Types the creature resists.
func (b SummonedStatBlock) Resistances() []content.DamageType { return slices.Clone(b.resistances) }

// ConditionImmunities returns a copy of the conditions the creature is immune to.
func (b SummonedStatBlock) ConditionImmunities() []content.ConditionKind {
	return slices.Clone(b.conditionImmunities)
}

// Attacks returns a copy of the attacks and bonus actions of the stat block.
func (b SummonedStatBlock) Attacks() []SummonedAttack { return slices.Clone(b.attacks) }

// Attack returns the entry of the kind and whether the stat block has it.
func (b SummonedStatBlock) Attack(k SummonedAttackKind) (SummonedAttack, bool) {
	i := slices.IndexFunc(b.attacks, func(a SummonedAttack) bool { return a.kind == k })
	if i < 0 {
		return SummonedAttack{}, false
	}
	return b.attacks[i], true
}

// MultiattackCount returns the number of Rend or other attacks of the Multiattack, 0 when the creature has none.
func (b SummonedStatBlock) MultiattackCount() int { return b.multiattack }

// ProficiencyBonus returns the Proficiency Bonus, the summoner's for a stat block with the printed "PB equals your Proficiency
// Bonus" (FR-036).
func (b SummonedStatBlock) ProficiencyBonus() int { return b.proficiencyBonus }

// ExperiencePoints returns the XP the creature awards: 0 for a stat block with Challenge Rating None, the printed XP otherwise
// (FR-007).
func (b SummonedStatBlock) ExperiencePoints() int { return b.experience }

// ForbiddenSpells returns a copy of the spells the creature cannot cast (a simulacrum cannot cast Simulacrum, FR-060).
func (b SummonedStatBlock) ForbiddenSpells() []content.SpellID { return slices.Clone(b.forbidden) }

// CanGainLevels reports whether the creature can gain levels (FR-061).
func (b SummonedStatBlock) CanGainLevels() bool { return !b.cannotGainLevels }

// CanTakeRests reports whether the creature can take rests (FR-061).
func (b SummonedStatBlock) CanTakeRests() bool { return !b.cannotTakeRests }

// HasLifeBond reports whether the creature has the Life Bond trait (FR-057).
func (b SummonedStatBlock) HasLifeBond() bool { return b.lifeBond }

// Telepathy returns the telepathy of the creature: 1 mile for an Otherworldly Steed (FR-058), none otherwise.
func (b SummonedStatBlock) Telepathy() TelepathyCapability { return b.telepathy }

// ZeroHitPointsRule returns DiesInstantly: every summoned creature dies at 0 Hit Points, a simulacrum included (FR-008).
func (SummonedStatBlock) ZeroHitPointsRule() ZeroHitPointsRule { return DiesInstantly }

// Equal reports whether b and o are the same stat block by value.
func (b SummonedStatBlock) Equal(o SummonedStatBlock) bool {
	return b.monster == o.monster && b.size == o.size && b.typ == o.typ && b.armorClass == o.armorClass &&
		b.hitPointMaximum == o.hitPointMaximum && b.hitDice == o.hitDice && b.walk == o.walk && b.climb == o.climb &&
		b.fly == o.fly && b.swim == o.swim && slices.Equal(b.resistances, o.resistances) &&
		slices.Equal(b.conditionImmunities, o.conditionImmunities) && slices.Equal(b.attacks, o.attacks) &&
		b.multiattack == o.multiattack && b.proficiencyBonus == o.proficiencyBonus && b.experience == o.experience &&
		slices.Equal(b.forbidden, o.forbidden) && b.cannotGainLevels == o.cannotGainLevels && b.cannotTakeRests == o.cannotTakeRests &&
		b.lifeBond == o.lifeBond && b.telepathy == o.telepathy
}

// expr builds an expression of validated numbers: a count of 1 to 9 dice of a Die constant and a modifier within 1000 plus the
// level, which dice.NewExpression accepts.
func expr(count int, d dice.Die, modifier int) dice.Expression {
	return dice.MustExpression(count, d, modifier)
}

// checkLevelFacts returns an error matching srderr.ErrInvalidInput for a level outside 1 to 9 or facts that are the zero value.
func checkLevelFacts(level int, f SummonerSpellFacts) error {
	if level < minCastLevel || level > maxCastLevel || f.IsZero() {
		return invalidInput("a cast level of 1 to 9 and the facts of the summoner")
	}
	return nil
}

// smallOrLess reports whether s is Medium or smaller.
func smallOrLess(s content.CreatureSize) bool { return s <= content.SizeMedium }

// NewAnimatedObjectStatBlock returns the Animated Object of a cast of Animate Objects at level for an object of size (SRD 5.2.1,
// pp. 108-109; FR-029, FR-030): Armor Class 15; Hit Points 10 (Medium or smaller), 20 (Large) or 40 (Huge); Speed 30; a Slam
// that uses the summoner's spell attack bonus and deals Force damage of 1d4+3 (Medium or smaller), 2d6+3+modifier (Large) or
// 2d12+3+modifier (Huge), plus 1d4, 1d6 or 1d12 for each level above 5.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil rejection, for a size that is not defined, a level outside 1
// to 9 or the zero facts; a LevelBelowSpellRejection for a level below 5 and an ObjectNotEligibleRejection for a Gargantuan
// object, checked in that order. Exactly one of the rejection and the error is set when the block is the zero value. It never
// panics.
func NewAnimatedObjectStatBlock(size content.CreatureSize, level int, facts SummonerSpellFacts) (SummonedStatBlock, SummonRejection, error) {
	if err := checkLevelFacts(level, facts); err != nil || !size.Valid() {
		return SummonedStatBlock{}, nil, invalidInput("an object size, a cast level of 1 to 9 and the facts of the summoner")
	}
	if level < 5 {
		return SummonedStatBlock{}, LevelBelowSpellRejection{Minimum: 5}, nil
	}
	if size == content.SizeGargantuan {
		return SummonedStatBlock{}, ObjectNotEligibleRejection{}, nil
	}
	extra := level - 5
	hp, slam := 10, expr(1+extra, dice.D4(), 3)
	switch size {
	case content.SizeLarge:
		hp, slam = 20, expr(2+extra, dice.D6(), 3+facts.modifier)
	case content.SizeHuge:
		hp, slam = 40, expr(2+extra, dice.D12(), 3+facts.modifier)
	case content.SizeTiny, content.SizeSmall, content.SizeMedium, content.SizeGargantuan:
		// Medium or smaller keep the first numbers; Gargantuan was refused above.
	}
	return SummonedStatBlock{
		monster: "animated_object", size: size, typ: content.CreatureTypeConstruct, armorClass: 15, hitPointMaximum: hp, walk: 30,
		attacks:          []SummonedAttack{{kind: SummonedAttackSlam, attackBonus: facts.attackBonus, damage: slam, damageType: content.DamageForce, rangeFeet: 5}},
		proficiencyBonus: facts.proficiencyBonus,
	}, nil, nil
}

// NewGiantInsectStatBlock returns the Giant Insect of a cast of Giant Insect at level in form (SRD 5.2.1, pp. 135-136; FR-032):
// Armor Class 11 plus the level; Hit Points 30 plus 10 for each level above 4; Speed 40 and Climb 40, and Fly 40 for the wasp;
// Multiattack of half the level (round down) attacks; Poison Jab of 1d6+3+level Piercing damage plus 1d4 Poison, reach 10; the
// spider's Web Bolt of 1d10+3+level Bludgeoning damage, range 60, which reduces Speed to 0; the centipede's Venomous Spew Bonus
// Action, a Constitution save at the spell save DC against a creature within 10 feet that is Poisoned until the start of the
// insect's next turn.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil rejection, for a form that is not defined, a level outside 1
// to 9 or the zero facts; a LevelBelowSpellRejection for a level below 4. It never panics.
func NewGiantInsectStatBlock(form InsectForm, level int, facts SummonerSpellFacts) (SummonedStatBlock, SummonRejection, error) {
	if err := checkLevelFacts(level, facts); err != nil || !form.Valid() {
		return SummonedStatBlock{}, nil, invalidInput("an insect form, a cast level of 1 to 9 and the facts of the summoner")
	}
	if level < 4 {
		return SummonedStatBlock{}, LevelBelowSpellRejection{Minimum: 4}, nil
	}
	attacks := []SummonedAttack{{kind: SummonedAttackPoisonJab, attackBonus: facts.attackBonus, damage: expr(1, dice.D6(), 3+level),
		damageType: content.DamagePiercing, extra: expr(1, dice.D4(), 0), extraTyp: content.DamagePoison, rangeFeet: 10}}
	fly := 0
	switch form {
	case InsectWasp:
		fly = 40
	case InsectSpider:
		attacks = append(attacks, SummonedAttack{kind: SummonedAttackWebBolt, attackBonus: facts.attackBonus, damage: expr(1, dice.D10(), 3+level),
			damageType: content.DamageBludgeoning, rangeFeet: 60})
	case InsectCentipede:
		attacks = append(attacks, SummonedAttack{kind: SummonedAttackVenomousSpew, saveDC: facts.saveDC, rangeFeet: 10})
	}
	return SummonedStatBlock{
		monster: "giant_insect", size: content.SizeLarge, typ: content.CreatureTypeBeast, armorClass: 11 + level,
		hitPointMaximum: 30 + 10*(level-4), walk: 40, climb: 40, fly: fly, attacks: attacks, multiattack: level / 2,
		proficiencyBonus: facts.proficiencyBonus,
	}, nil, nil
}

// draconicResistances are the five Damage Types a Draconic Spirit resists (SRD 5.2.1, "Draconic Spirit", p. 166).
func draconicResistances() []content.DamageType {
	return []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageLightning, content.DamagePoison}
}

// NewDraconicSpiritStatBlock returns the Draconic Spirit of a cast of Summon Dragon at level (SRD 5.2.1, p. 166; FR-033): Armor
// Class 14 plus the level; Hit Points 50 plus 10 for each level above 5; Speed 30, Fly 60 and Swim 30; Resistance to Acid, Cold,
// Fire, Lightning and Poison; Immunity to the Charmed, Frightened and Poisoned conditions; a Multiattack of half the level (round
// down) Rend attacks (1d6+4+level Piercing damage, reach 10) and the Breath Weapon (a Dexterity save at the spell save DC, a
// 30-foot Cone, 2d6 damage of breathType, half on a success). shared is the Resistance the summoner shares and breathType the
// Breath Weapon's type; each must be one of the five and they may differ (Q-18).
//
// It returns an error matching srderr.ErrInvalidInput, with a nil rejection, for a level outside 1 to 9 or the zero facts; a
// LevelBelowSpellRejection for a level below 5; a ResistanceNotOfferedRejection for a type outside the five, with the field
// SharedResistanceField checked before BreathDamageTypeField. It never panics.
func NewDraconicSpiritStatBlock(level int, facts SummonerSpellFacts, shared, breathType content.DamageType) (SummonedStatBlock, SummonRejection, error) {
	if err := checkLevelFacts(level, facts); err != nil {
		return SummonedStatBlock{}, nil, err
	}
	if level < 5 {
		return SummonedStatBlock{}, LevelBelowSpellRejection{Minimum: 5}, nil
	}
	if !slices.Contains(draconicResistances(), shared) {
		return SummonedStatBlock{}, ResistanceNotOfferedRejection{Field: SharedResistanceField}, nil
	}
	if !slices.Contains(draconicResistances(), breathType) {
		return SummonedStatBlock{}, ResistanceNotOfferedRejection{Field: BreathDamageTypeField}, nil
	}
	return SummonedStatBlock{
		monster: "draconic_spirit", size: content.SizeLarge, typ: content.CreatureTypeDragon, armorClass: 14 + level,
		hitPointMaximum: 50 + 10*(level-5), walk: 30, fly: 60, swim: 30, resistances: draconicResistances(),
		conditionImmunities: []content.ConditionKind{content.ConditionCharmed, content.ConditionFrightened, content.ConditionPoisoned},
		attacks: []SummonedAttack{
			{kind: SummonedAttackRend, attackBonus: facts.attackBonus, damage: expr(1, dice.D6(), 4+level), damageType: content.DamagePiercing, rangeFeet: 10},
			{kind: SummonedAttackBreathWeapon, saveDC: facts.saveDC, damage: expr(2, dice.D6(), 0), damageType: breathType, rangeFeet: 30},
		},
		multiattack: level / 2, proficiencyBonus: facts.proficiencyBonus,
	}, nil, nil
}

// NewOtherworldlySteedStatBlock returns the Otherworldly Steed of a cast of Find Steed at level (SRD 5.2.1, pp. 130-131; FR-034):
// Large; Armor Class 10 plus the level; Hit Points 5 plus 10 for each level, with that many d10 Hit Dice; Speed 60, and Fly 60
// only for a level of 4 or more; Life Bond; telepathy of 1 mile that works only with the summoner; a Slam of 1d8 plus the level
// of Radiant (Celestial), Psychic (Fey) or Necrotic (Fiend) damage; and the Bonus Action of the chosen type only: Fell Glare
// (Fiend), Fey Step (Fey) or Healing Touch of 2d8 plus the level (Celestial).
//
// It returns an error matching srderr.ErrInvalidInput, with a nil rejection, for a type that is not defined, a level outside 1 to
// 9 or the zero facts; a LevelBelowSpellRejection for a level below 2. It never panics.
func NewOtherworldlySteedStatBlock(steed SteedType, level int, facts SummonerSpellFacts) (SummonedStatBlock, SummonRejection, error) {
	if err := checkLevelFacts(level, facts); err != nil || !steed.Valid() {
		return SummonedStatBlock{}, nil, invalidInput("a steed type, a cast level of 1 to 9 and the facts of the summoner")
	}
	if level < 2 {
		return SummonedStatBlock{}, LevelBelowSpellRejection{Minimum: 2}, nil
	}
	slam := SummonedAttack{kind: SummonedAttackOtherworldlySlam, attackBonus: facts.attackBonus, damage: expr(1, dice.D8(), level), rangeFeet: 5}
	var bonus SummonedAttack
	switch steed {
	case SteedCelestial:
		slam.damageType = content.DamageRadiant
		bonus = SummonedAttack{kind: SummonedAttackHealingTouch, damage: expr(2, dice.D8(), level), rangeFeet: 5}
	case SteedFey:
		slam.damageType = content.DamagePsychic
		bonus = SummonedAttack{kind: SummonedAttackFeyStep, rangeFeet: 60}
	case SteedFiend:
		slam.damageType = content.DamageNecrotic
		bonus = SummonedAttack{kind: SummonedAttackFellGlare, saveDC: facts.saveDC, rangeFeet: 60}
	}
	fly := 0
	if level >= 4 {
		fly = 60
	}
	return SummonedStatBlock{
		monster: "otherworldly_steed", size: content.SizeLarge, typ: steed.CreatureType(), armorClass: 10 + level,
		hitPointMaximum: 5 + 10*level, hitDice: expr(level, dice.D10(), 0), walk: 60, fly: fly,
		attacks: []SummonedAttack{slam, bonus}, proficiencyBonus: facts.proficiencyBonus, lifeBond: true,
		telepathy: MustTelepathic(steedTelepathyFeet),
	}, nil, nil
}

// steedTelepathyFeet is 1 mile in feet, the range of an Otherworldly Steed's telepathy (FR-058).
const steedTelepathyFeet = 5280

// SimulacrumOriginal is the creature a simulacrum copies, as the host reports it at the time of casting (FR-060). Build it with a
// keyed literal. Monster is the MonsterID the host gives the simulacrum, which a character needs (Q-2).
type SimulacrumOriginal struct {
	Monster         content.MonsterID
	Size            content.CreatureSize
	Type            content.CreatureType
	HitPointMaximum int
	WithinTenFeet   bool
}

// NewSimulacrumStatBlock returns the stat block of the simulacrum of original (SRD 5.2.1, "Simulacrum", p. 163; FR-060 to
// FR-061): a Construct of the original's size whose Hit Point maximum is half the original's, rounded down, and at least 1
// (package design, Q-16); it cannot cast Simulacrum, gain levels or take rests. The statistics beyond these are the original's
// and the host's.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil rejection, for an original with no monster id, a size or
// creature type that is not defined or a Hit Point maximum below 1; an OriginalNotEligibleRejection for an original that is not
// a Beast or Humanoid within 10 feet. It never panics.
func NewSimulacrumStatBlock(original SimulacrumOriginal) (SummonedStatBlock, SummonRejection, error) {
	if !original.Monster.Valid() || !original.Size.Valid() || !original.Type.Valid() || original.HitPointMaximum < 1 {
		return SummonedStatBlock{}, nil, invalidInput("a simulacrum original needs a monster id, a size, a type and a Hit Point maximum of 1 or more")
	}
	if !original.WithinTenFeet || (original.Type != content.CreatureTypeBeast && original.Type != content.CreatureTypeHumanoid) {
		return SummonedStatBlock{}, OriginalNotEligibleRejection{}, nil
	}
	return SummonedStatBlock{
		monster: original.Monster, size: original.Size, typ: content.CreatureTypeConstruct, hitPointMaximum: max(1, original.HitPointMaximum/2),
		forbidden: []content.SpellID{SummonSimulacrum.Spell()}, cannotGainLevels: true, cannotTakeRests: true,
	}, nil, nil
}

// statBlockOfMonster returns the stat block of a content monster that a spell names (a Skeleton, a Zombie, a Ghoul, a Ghast, a
// Wight, a Mummy or a familiar form) with the creature type typ, which a familiar chooses (FR-043). Its size is the monster's
// first, its Hit Points the printed average, its XP the printed XP (FR-007).
func statBlockOfMonster(m content.Monster, typ content.CreatureType) SummonedStatBlock {
	b := SummonedStatBlock{monster: m.ID(), size: content.SizeMedium, typ: typ, armorClass: m.ArmorClass(), hitPointMaximum: m.HitPoints().Average(),
		proficiencyBonus: m.ProficiencyBonus(), experience: m.ExperiencePoints()}
	if sizes := m.Sizes(); len(sizes) > 0 {
		b.size = sizes[0]
	}
	for _, s := range m.Speeds() {
		for _, mode := range s.Modes() {
			switch mode {
			case content.SpeedWalk:
				b.walk = s.Feet()
			case content.SpeedClimb:
				b.climb = s.Feet()
			case content.SpeedFly:
				b.fly = s.Feet()
			case content.SpeedSwim:
				b.swim = s.Feet()
			case content.SpeedBurrow:
				// A summoned stat block has no Burrow Speed.
			}
		}
	}
	return b
}

// MustSummonerSpellFacts is NewSummonerSpellFacts for numbers known to be valid. It panics with the error
// NewSummonerSpellFacts returns, and only then.
func MustSummonerSpellFacts(attackBonus, saveDC, modifier, proficiencyBonus int) SummonerSpellFacts {
	f, err := NewSummonerSpellFacts(attackBonus, saveDC, modifier, proficiencyBonus)
	if err != nil {
		panic(err)
	}
	return f
}
