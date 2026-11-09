package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The SRD formulas and numbers of the Paladin feature texts, kept as constants (class-paladin.md FR-125); every level gate
// comes from the definition instead.
const (
	layOnHandsPerLevel  = 5 // "five times your Paladin level"
	layOnHandsCondition = 5 // Hit Points expended to remove one condition
	paladinDetectFeet   = 60
	paladinAuraFeet     = 10
	paladinExpandedFeet = 30
	paladinLightFeet    = 20
	paladinSenseMinutes = 10
	paladinNimbusMinute = 10
	paladinWeaponMinute = 10
	paladinAbjureMinute = 1
)

// restoringTouchConditions are the six conditions Restoring Touch removes (SRD 5.2.1, "Level 14: Restoring Touch", p. 55).
func restoringTouchConditions() []content.ConditionKind {
	return []content.ConditionKind{content.ConditionBlinded, content.ConditionCharmed, content.ConditionDeafened,
		content.ConditionFrightened, content.ConditionParalyzed, content.ConditionStunned}
}

// LayOnHandsRequest is what a Lay On Hands use asks for: Hit Points to restore and conditions to remove. Construct it with a
// keyed literal.
type LayOnHandsRequest struct {
	HitPoints int
	Remove    []content.ConditionKind
}

// LayOnHandsOutcome is an accepted Lay On Hands use: the Hit Points restored, the conditions removed (each once), the Hit Points
// spent from the pool, the pool left and the action cost (a Bonus Action). Only LayOnHandsRules.Use builds one.
type LayOnHandsOutcome struct {
	hitPoints  int
	conditions []content.ConditionKind
	cost       int
	remaining  int
}

// HitPoints returns the Hit Points restored to the creature.
func (o LayOnHandsOutcome) HitPoints() int { return o.hitPoints }

// Conditions returns the conditions removed, as a new slice.
func (o LayOnHandsOutcome) Conditions() []content.ConditionKind { return slices.Clone(o.conditions) }

// Cost returns the Hit Points spent from the pool: the restore amount plus 5 for each condition removed.
func (o LayOnHandsOutcome) Cost() int { return o.cost }

// Remaining returns the Hit Points left in the pool.
func (o LayOnHandsOutcome) Remaining() int { return o.remaining }

// ActionCost returns PaladinBonusAction.
func (LayOnHandsOutcome) ActionCost() PaladinActionCost { return PaladinBonusAction }

// LayOnHandsRules decides a Lay On Hands use (SRD 5.2.1, "Level 1: Lay On Hands", p. 54; "Level 14: Restoring Touch", p. 55).
// It is stateless; the zero value is ready to use. Safe for concurrent use.
type LayOnHandsRules struct{}

// Use decides req against a pool of remaining Hit Points for a Paladin of level. It returns the outcome and no reasons when the
// use is accepted, and the outcome zero with every reason that holds otherwise, in the fixed order of class-paladin.md FR-055:
// PaladinInvalidAmount, PaladinEmptyRequest, PaladinConditionNotRemovable, PaladinRestoringTouchNotAvailable,
// PaladinExceedsPool. Poisoned may be removed at every level; the six Restoring Touch conditions only from the level of that
// feature entry. A condition named twice is removed once. The inputs are not altered. It never panics.
func (LayOnHandsRules) Use(p PaladinProgression, level PaladinLevel, remaining int, req LayOnHandsRequest) (LayOnHandsOutcome, []PaladinReason) {
	var reasons []PaladinReason
	conds := []content.ConditionKind{}
	notRemovable, touch := false, false
	for _, c := range req.Remove {
		if slices.Contains(conds, c) {
			continue
		}
		conds = append(conds, c)
		switch {
		case c == content.ConditionPoisoned:
		case slices.Contains(restoringTouchConditions(), c):
			touch = true
		default:
			notRemovable = true
		}
	}
	cost := max(req.HitPoints, 0) + layOnHandsCondition*len(conds)
	if req.HitPoints < 0 {
		reasons = append(reasons, PaladinInvalidAmount)
	}
	if req.HitPoints == 0 && len(conds) == 0 {
		reasons = append(reasons, PaladinEmptyRequest)
	}
	if notRemovable {
		reasons = append(reasons, PaladinConditionNotRemovable)
	}
	if touch && !p.has(PaladinFeatureRestoringTouch, level) {
		reasons = append(reasons, PaladinRestoringTouchNotAvailable)
	}
	if cost > remaining {
		reasons = append(reasons, PaladinExceedsPool)
	}
	if len(reasons) > 0 {
		return LayOnHandsOutcome{}, reasons
	}
	return LayOnHandsOutcome{hitPoints: req.HitPoints, conditions: conds, cost: cost, remaining: remaining - cost}, nil
}

// DetectionKind is what Divine Sense detects. The zero value is invalid.
type DetectionKind int

const (
	// DetectLocation: the location of a creature of the types.
	DetectLocation DetectionKind = iota + 1
	// DetectCreatureType: the creature type of a creature detected.
	DetectCreatureType
	// DetectConsecratedOrDesecrated: a place or object consecrated or desecrated, as with the Hallow spell.
	DetectConsecratedOrDesecrated
)

// String returns "location", "creature_type", "consecrated_or_desecrated" or "unknown".
func (k DetectionKind) String() string {
	switch k {
	case DetectLocation:
		return "location"
	case DetectCreatureType:
		return "creature_type"
	case DetectConsecratedOrDesecrated:
		return "consecrated_or_desecrated"
	}
	return "unknown"
}

// CoverGrant is the cover a feature grants. The zero value is invalid.
type CoverGrant int

// CoverHalf is Half Cover.
const CoverHalf CoverGrant = iota + 1

// String returns "half" or "unknown".
func (c CoverGrant) String() string {
	if c == CoverHalf {
		return "half"
	}
	return "unknown"
}

// DivineSenseEffect is the data of Divine Sense (SRD 5.2.1, "Level 3: Channel Divinity", p. 55). Only the Channel Divinity use
// builds one.
type DivineSenseEffect struct{}

// ActionCost returns PaladinBonusAction.
func (DivineSenseEffect) ActionCost() PaladinActionCost { return PaladinBonusAction }

// DurationMinutes returns 10; the effect also ends when the Paladin has the Incapacitated condition.
func (DivineSenseEffect) DurationMinutes() int { return paladinSenseMinutes }

// RadiusFeet returns 60.
func (DivineSenseEffect) RadiusFeet() int { return paladinDetectFeet }

// CreatureTypes returns Celestial, Fiend and Undead, as a new slice.
func (DivineSenseEffect) CreatureTypes() []content.CreatureType {
	return []content.CreatureType{content.CreatureTypeCelestial, content.CreatureTypeFiend, content.CreatureTypeUndead}
}

// Detects returns what is detected: the location and creature type of those creatures, and consecrated or desecrated places.
func (DivineSenseEffect) Detects() []DetectionKind {
	return []DetectionKind{DetectLocation, DetectCreatureType, DetectConsecratedOrDesecrated}
}

// AbjureFoesEffect is the data of Abjure Foes (SRD 5.2.1, "Level 9: Abjure Foes", p. 55). Only the Channel Divinity use builds
// one.
type AbjureFoesEffect struct {
	targets int
	dc      SpellSaveDC
}

// ActionCost returns PaladinMagicAction.
func (AbjureFoesEffect) ActionCost() PaladinActionCost { return PaladinMagicAction }

// Targets returns the number of creatures that can be targeted: the Charisma modifier, minimum one.
func (e AbjureFoesEffect) Targets() int { return e.targets }

// RadiusFeet returns 60.
func (AbjureFoesEffect) RadiusFeet() int { return paladinDetectFeet }

// SavingThrow returns Wisdom and the DC supplied in the command, unchanged.
func (e AbjureFoesEffect) SavingThrow() (ability.Ability, SpellSaveDC) { return ability.Wisdom, e.dc }

// DurationMinutes returns 1; the Frightened condition also ends when the target takes any damage.
func (AbjureFoesEffect) DurationMinutes() int { return paladinAbjureMinute }

// SacredWeaponEffect is the data of Sacred Weapon (SRD 5.2.1, "Level 3: Sacred Weapon", p. 56). Only the Channel Divinity use
// builds one.
type SacredWeaponEffect struct {
	weapon string
	bonus  int
}

// Weapon returns the host key of the imbued Melee weapon.
func (e SacredWeaponEffect) Weapon() string { return e.weapon }

// AttackBonus returns the bonus to attack rolls with the weapon: the Charisma modifier, minimum +1.
func (e SacredWeaponEffect) AttackBonus() int { return e.bonus }

// DurationMinutes returns 10, or until the feature is used again.
func (SacredWeaponEffect) DurationMinutes() int { return paladinWeaponMinute }

// DamageTypes returns the weapon's normal damage type (the host's) and Radiant, the Paladin's choice per hit: only Radiant is a
// fixed value, returned as a new slice.
func (SacredWeaponEffect) DamageTypes() []content.DamageType {
	return []content.DamageType{content.DamageRadiant}
}

// Light returns Bright Light in a 20-foot radius and Dim Light 20 feet beyond that.
func (SacredWeaponEffect) Light() (bright content.LightLevel, brightFeet int, dim content.LightLevel, dimFeet int) {
	return content.LightBright, paladinLightFeet, content.LightDim, paladinLightFeet
}

// minOne returns the larger of 1 and the modifier (the "minimum of one" and "minimum bonus of +1" of the feature texts).
func minOne(m ability.Modifier) int { return max(m.Value(), 1) }

// AuraEffects is the Aura of Protection of a Paladin and what rides on it (SRD 5.2.1, "Level 6: Aura of Protection", p. 55; "Level
// 10: Aura of Courage", p. 55; "Level 7: Aura of Devotion", p. 56). The zero value is an inactive aura. Immutable.
type AuraEffects struct {
	active     bool
	radiusFeet int
	saveBonus  int
	immunities []content.ConditionKind
}

// Active reports whether the aura is active: the Paladin has the feature and is not Incapacitated.
func (a AuraEffects) Active() bool { return a.active }

// RadiusFeet returns the Emanation radius in feet, 0 when inactive.
func (a AuraEffects) RadiusFeet() int { return a.radiusFeet }

// SaveBonus returns the bonus to saving throws for the Paladin and allies in the aura, 0 when inactive.
func (a AuraEffects) SaveBonus() int { return a.saveBonus }

// Immunities returns the conditions the Paladin and allies in the aura are immune to (Frightened, Charmed), as a new slice.
func (a AuraEffects) Immunities() []content.ConditionKind { return slices.Clone(a.immunities) }

// PaladinAuras computes the auras (class-paladin.md FR-077 to FR-083). It is stateless; the zero value is ready to use.
type PaladinAuras struct{}

// Compute returns the aura of a Paladin of level: absent below the Aura of Protection level and inactive while Incapacitated; a
// 10-foot Emanation, 30 feet from Aura Expansion; the larger of 1 and the Charisma modifier as the save bonus; Immunity to
// Frightened from Aura of Courage and, with Oath of Devotion chosen, to Charmed from Aura of Devotion.
func (PaladinAuras) Compute(p PaladinProgression, level PaladinLevel, oath bool, charisma ability.Modifier, incapacitated bool) AuraEffects {
	if !p.has(PaladinFeatureAuraOfProtection, level) || incapacitated {
		return AuraEffects{immunities: []content.ConditionKind{}}
	}
	a := AuraEffects{active: true, radiusFeet: paladinAuraFeet, saveBonus: minOne(charisma), immunities: []content.ConditionKind{}}
	if p.has(PaladinFeatureAuraExpansion, level) {
		a.radiusFeet = paladinExpandedFeet
	}
	if oath && p.has(PaladinFeatureAuraOfDevotion, level) {
		a.immunities = append(a.immunities, content.ConditionCharmed)
	}
	if p.has(PaladinFeatureAuraOfCourage, level) {
		a.immunities = append(a.immunities, content.ConditionFrightened)
	}
	return a
}

// AuraOffer is one Aura of Protection a creature stands in: the host's key of the Paladin and the bonus it gives. Construct it
// with a keyed literal.
type AuraOffer struct {
	Paladin   string
	SaveBonus int
}

// Choose returns the bonus of the aura the creature chose among the ones it stands in: only one Aura of Protection applies at a
// time (SRD 5.2.1, "Level 6: Aura of Protection", p. 55; class-paladin.md FR-081). No candidates give a bonus of 0 and no
// reason; a chosen Paladin that is not a candidate gives PaladinNotACandidate.
func (PaladinAuras) Choose(candidates []AuraOffer, chosen string) (int, PaladinReason) {
	if len(candidates) == 0 {
		return 0, 0
	}
	for _, c := range candidates {
		if c.Paladin == chosen {
			return c.SaveBonus, 0
		}
	}
	return 0, PaladinNotACandidate
}

// AttacksPerAttackAction returns how many attacks the Attack action makes: 1 below the level of the Extra Attack entry and 2
// from it (SRD 5.2.1, "Level 5: Extra Attack", p. 55). Combining with another class's Extra Attack is out of scope.
func AttacksPerAttackAction(p PaladinProgression, level PaladinLevel) int {
	if p.has(PaladinFeatureExtraAttack, level) {
		return 2
	}
	return 1
}

// PaladinAttackKind is the kind of attack a hit was made with, the input of Radiant Strikes. The zero value is invalid.
type PaladinAttackKind int

const (
	// PaladinMeleeWeapon: an attack roll with a Melee weapon (a thrown Melee weapon keeps its type).
	PaladinMeleeWeapon PaladinAttackKind = iota + 1
	// PaladinRangedWeapon: an attack roll with a Ranged weapon.
	PaladinRangedWeapon
	// PaladinUnarmedStrike: an Unarmed Strike.
	PaladinUnarmedStrike
	// PaladinSpellAttack: a spell attack.
	PaladinSpellAttack
)

// PaladinAttackKindForWeapon returns the attack kind of a weapon attack from the weapon's own type, not from how it was used
// (SRD 5.2.1, "Weapon Properties", "Thrown", p. 90): a thrown Javelin is a Melee weapon. An invalid type gives the zero kind.
func PaladinAttackKindForWeapon(t content.WeaponType) PaladinAttackKind {
	switch t {
	case content.WeaponMelee:
		return PaladinMeleeWeapon
	case content.WeaponRanged:
		return PaladinRangedWeapon
	}
	return 0
}

// RadiantStrikes is the Radiant Strikes feature (SRD 5.2.1, "Level 11: Radiant Strikes", p. 55). It is stateless; the zero value
// is ready to use.
type RadiantStrikes struct{}

// AppliesTo reports whether a hit of the kind deals the extra damage: from the level of the Radiant Strikes entry, for a Melee
// weapon or an Unarmed Strike.
func (RadiantStrikes) AppliesTo(p PaladinProgression, level PaladinLevel, kind PaladinAttackKind) bool {
	return p.has(PaladinFeatureRadiantStrikes, level) && (kind == PaladinMeleeWeapon || kind == PaladinUnarmedStrike)
}

// ExtraDamage returns the extra damage of one qualifying hit: 1d8 Radiant.
func (RadiantStrikes) ExtraDamage() (dice.Expression, content.DamageType) {
	return dice.MustExpression(1, dice.D8(), 0), content.DamageRadiant
}

// SmiteOfProtectionEffect is the data of Smite of Protection (SRD 5.2.1, "Level 15: Smite of Protection", p. 57): Half Cover
// for the Paladin and allies in the aura until the start of the Paladin's next turn. Only ApplySmiteOfProtection builds one.
type SmiteOfProtectionEffect struct{}

// Cover returns CoverHalf.
func (SmiteOfProtectionEffect) Cover() CoverGrant { return CoverHalf }

// UntilNextTurn reports that the benefit lasts until the start of the Paladin's next turn.
func (SmiteOfProtectionEffect) UntilNextTurn() bool { return true }

// smiteOfProtectionApplies reports whether Smite of Protection gives its cover: the Oath of Devotion feature is had, the aura is
// active and Divine Smite was cast.
func smiteOfProtectionApplies(p PaladinProgression, level PaladinLevel, oath bool, divineSmiteCast bool, aura AuraEffects) bool {
	return divineSmiteCast && oath && p.has(PaladinFeatureSmiteOfProtection, level) && aura.active
}

// HolyNimbusEffect is the data of Holy Nimbus (SRD 5.2.1, "Level 20: Holy Nimbus", p. 57). The aura-based effects are inactive
// while the Paladin is Incapacitated (class-paladin.md FR-095). Only the Holy Nimbus use builds one.
type HolyNimbusEffect struct {
	active bool
	damage int
}

// ActionCost returns PaladinBonusAction.
func (HolyNimbusEffect) ActionCost() PaladinActionCost { return PaladinBonusAction }

// DurationMinutes returns 10, or until the Paladin ends it (no action required).
func (HolyNimbusEffect) DurationMinutes() int { return paladinNimbusMinute }

// AuraActive reports whether the aura-based effects (Holy Ward, Radiant Damage, Sunlight) are active.
func (e HolyNimbusEffect) AuraActive() bool { return e.active }

// HolyWard returns the creature types a saving throw against forced by which gives Advantage (Fiend, Undead), and whether the
// ward is active.
func (e HolyNimbusEffect) HolyWard() ([]content.CreatureType, bool) {
	return []content.CreatureType{content.CreatureTypeFiend, content.CreatureTypeUndead}, e.active
}

// RadiantDamage returns the Radiant damage an enemy that starts its turn in the aura takes (the Charisma modifier plus the
// Proficiency Bonus), and whether it applies. The amount is at least 1 for the SRD scores and Proficiency Bonus of level 20.
func (e HolyNimbusEffect) RadiantDamage() (int, bool) { return e.damage, e.active }

// Sunlight reports whether the aura is filled with Bright Light that is sunlight.
func (e HolyNimbusEffect) Sunlight() bool { return e.active }

// PaladinEffectKind is an effect that can be active. The zero value is invalid.
type PaladinEffectKind int

const (
	// PaladinEffectDivineSense is Divine Sense.
	PaladinEffectDivineSense PaladinEffectKind = iota + 1
	// PaladinEffectSacredWeapon is Sacred Weapon.
	PaladinEffectSacredWeapon
	// PaladinEffectHolyNimbus is Holy Nimbus.
	PaladinEffectHolyNimbus
)

// String returns "divine_sense", "sacred_weapon", "holy_nimbus" or "unknown".
func (k PaladinEffectKind) String() string {
	switch k {
	case PaladinEffectDivineSense:
		return "divine_sense"
	case PaladinEffectSacredWeapon:
		return "sacred_weapon"
	case PaladinEffectHolyNimbus:
		return "holy_nimbus"
	}
	return "unknown"
}

// PaladinEndReason is why an effect ends. The zero value is invalid.
type PaladinEndReason int

const (
	// PaladinEndElapsed: the duration elapsed.
	PaladinEndElapsed PaladinEndReason = iota + 1
	// PaladinEndIncapacitated: the Paladin has the Incapacitated condition (Divine Sense).
	PaladinEndIncapacitated
	// PaladinEndEndedEarly: the Paladin ended it (no action required).
	PaladinEndEndedEarly
	// PaladinEndNotCarried: the Paladin is not carrying the weapon (Sacred Weapon).
	PaladinEndNotCarried
	// PaladinEndUsedAgain: Sacred Weapon was used again.
	PaladinEndUsedAgain
)

// String returns "elapsed", "incapacitated", "ended_early", "not_carried", "used_again" or "unknown".
func (r PaladinEndReason) String() string {
	switch r {
	case PaladinEndElapsed:
		return "elapsed"
	case PaladinEndIncapacitated:
		return "incapacitated"
	case PaladinEndEndedEarly:
		return "ended_early"
	case PaladinEndNotCarried:
		return "not_carried"
	case PaladinEndUsedAgain:
		return "used_again"
	}
	return "unknown"
}

// endable reports whether the effect can end for the reason (class-paladin.md FR-076).
func (k PaladinEffectKind) endable(r PaladinEndReason) bool {
	switch k {
	case PaladinEffectDivineSense:
		return r == PaladinEndElapsed || r == PaladinEndIncapacitated
	case PaladinEffectSacredWeapon:
		return r == PaladinEndElapsed || r == PaladinEndEndedEarly || r == PaladinEndNotCarried || r == PaladinEndUsedAgain
	case PaladinEffectHolyNimbus:
		return r == PaladinEndElapsed || r == PaladinEndEndedEarly
	}
	return false
}

// EndPaladinEffect ends an active effect of the Paladin id for the reason (class-paladin.md FR-076). It returns the new state
// and the PaladinEffectEnded event, or the state unchanged, a nil event and PaladinNotActive when the effect is not active, or
// PaladinReasonNotApplicable when the effect cannot end for the reason. It refunds no Channel Divinity use. It is pure and
// synchronous: it needs no content.
func EndPaladinEffect(id Identifier, state PaladinState, effect PaladinEffectKind, reason PaladinEndReason) (PaladinState, PaladinEvent, PaladinReason) {
	var active bool
	switch effect {
	case PaladinEffectDivineSense:
		active = state.divineSense
	case PaladinEffectSacredWeapon:
		active = state.sacredWeapon != ""
	case PaladinEffectHolyNimbus:
		active = state.holyNimbus
	}
	switch {
	case !active:
		return state, nil, PaladinNotActive
	case !effect.endable(reason):
		return state, nil, PaladinReasonNotApplicable
	}
	switch effect {
	case PaladinEffectDivineSense:
		state.divineSense = false
	case PaladinEffectSacredWeapon:
		state.sacredWeapon = ""
	case PaladinEffectHolyNimbus:
		state.holyNimbus = false
	}
	return state, PaladinEffectEnded{id: id, effect: effect, reason: reason}, 0
}
