package character

import (
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ResolveWishRejectionKind is why a Wish choice was refused, a closed enum. The zero value is
// invalid.
type ResolveWishRejectionKind int

// The rejection kinds, in the order the checks run: WishCannotCastAgain first, then the checks of
// the choice.
const (
	// WishCannotCastAgain: the creature is unable to cast Wish ever again.
	WishCannotCastAgain ResolveWishRejectionKind = iota + 1
	// WishSpellLevelTooHigh: the spell to duplicate is level 9.
	WishSpellLevelTooHigh
	// WishObjectValueTooHigh: the object is worth more than 25,000 GP.
	WishObjectValueTooHigh
	// WishObjectIsMagicItem: the object is a magic item.
	WishObjectIsMagicItem
	// WishObjectTooLarge: the object is more than 300 feet in its largest dimension.
	WishObjectTooLarge
	// WishTooManyCreatures: more creatures than the effect allows.
	WishTooManyCreatures
	// WishRollNotRecent: the roll was not made within the last round.
	WishRollNotRecent
	// WishFeatRequiredByAnother: the feat to replace is a prerequisite of another feat or feature.
	WishFeatRequiredByAnother
	// WishFeatReplacementRejected: the feat replacement was refused; the nested FeatRejection says why.
	WishFeatReplacementRejected
)

// Valid reports whether k is a member.
func (k ResolveWishRejectionKind) Valid() bool {
	return k >= WishCannotCastAgain && k <= WishFeatReplacementRejected
}

// String returns the snake_case name, the value of the rejection attribute, or "unknown".
func (k ResolveWishRejectionKind) String() string {
	if !k.Valid() {
		return "unknown"
	}
	names := [...]string{"cannot_cast_wish_again", "spell_level_too_high", "object_value_too_high", "object_is_magic_item",
		"object_too_large", "too_many_creatures", "roll_not_recent", "feat_required_by_another", "feat_replacement_rejected"}
	return names[k-1]
}

// ResolveWishRejection is a Wish choice the rules refuse. The zero value names no rejection.
type ResolveWishRejection struct {
	kind ResolveWishRejectionKind
	feat FeatRejection
}

// Kind returns why the choice was refused.
func (r ResolveWishRejection) Kind() ResolveWishRejectionKind { return r.kind }

// FeatRejection returns the rejection of the feat replacement for WishFeatReplacementRejected and
// nil for every other kind.
func (r ResolveWishRejection) FeatRejection() FeatRejection { return r.feat }

// WishCheck is the outcome of CheckWish: a rejection or the effect the choice hands off. The zero
// value is neither.
type WishCheck struct {
	rejection ResolveWishRejection
	effect    WishEffect
}

// Rejection returns the rejection and true when the choice is refused.
func (c WishCheck) Rejection() (ResolveWishRejection, bool) {
	return c.rejection, c.rejection.kind != 0
}

// Effect returns the effect of an accepted choice; nil when it was refused.
func (c WishCheck) Effect() WishEffect { return c.effect }

// WishApplication is a Wish choice applied: the new Character (with the stress written back when
// the choice incurs it), the new TakenFeats, the effect and the events in order (the effect's, then
// WishStressEndured). Only ApplyWish builds one.
type WishApplication struct {
	character Character
	feats     TakenFeats
	effect    WishEffect
	events    []Event
}

// Character returns the new Character.
func (a WishApplication) Character() Character { return a.character }

// Feats returns the new TakenFeats for Sudden Learning and the zero TakenFeats for every other choice.
func (a WishApplication) Feats() TakenFeats { return a.feats }

// Effect returns the effect handed off.
func (a WishApplication) Effect() WishEffect { return a.effect }

// Events returns a new slice of the events, in order.
func (a WishApplication) Events() []Event { return slices.Clone(a.events) }

// WishRequiresStress reports whether choice incurs the stress: every choice except DuplicateSpell
// (also a Reshape Reality the Game Master later refuses).
func WishRequiresStress(choice WishChoice) bool {
	_, duplicate := choice.(DuplicateSpell)
	return !duplicate
}

// wishResolution is a choice run through the rules without dice.
type wishResolution struct {
	rejection ResolveWishRejection
	stress    WishStress
	character Character
	feats     TakenFeats
	effect    WishEffect
	events    []Event
}

func wishRejected(kind ResolveWishRejectionKind) wishResolution {
	return wishResolution{rejection: ResolveWishRejection{kind: kind}}
}

// resolveWishChoice runs the checks of c's choice in their fixed order and builds the effect.
func resolveWishChoice(c Character, choice WishChoice) (wishResolution, error) {
	if c.IsZero() || choice == nil {
		return wishResolution{}, invalidInput("resolve a wish for a zero character or no choice")
	}
	if err := choice.validate(); err != nil {
		return wishResolution{}, err
	}
	stress, err := wishStressOf(c)
	if err != nil {
		return wishResolution{}, err
	}
	if !stress.CanCastWish() {
		return wishRejected(WishCannotCastAgain), nil
	}
	res := wishResolution{stress: stress, character: c}
	switch v := choice.(type) {
	case DuplicateSpell:
		if v.spell.Level() >= WishLevel() {
			return wishRejected(WishSpellLevelTooHigh), nil
		}
		res.effect = WishDuplicate{spell: v.spell.ID(), level: v.spell.Level(), concentration: v.spell.RequiresConcentration()}
		res.events = []Event{WishDuplicated{id: c.ID(), spell: v.spell.ID(), level: v.spell.Level()}}
	case ObjectCreation:
		switch {
		case v.value.ValueInCopper() > wishObjectMaxCopper:
			return wishRejected(WishObjectValueTooHigh), nil
		case v.magicItem:
			return wishRejected(WishObjectIsMagicItem), nil
		case v.feet > wishObjectMaxFeet:
			return wishRejected(WishObjectTooLarge), nil
		}
		res.effect = WishObject{value: v.value, feet: v.feet}
	case InstantHealth:
		if v.others > wishHealMaxOthers {
			return wishRejected(WishTooManyCreatures), nil
		}
		res.effect = WishInstantHealth(v)
	case ResistanceGrant:
		if len(v.creatures) > wishGrantMaxCreature {
			return wishRejected(WishTooManyCreatures), nil
		}
		res.effect = WishResistance{damage: v.damage, creatures: slices.Clone(v.creatures)}
	case SpellImmunityGrant:
		if len(v.creatures) > wishGrantMaxCreature {
			return wishRejected(WishTooManyCreatures), nil
		}
		res.effect = WishSpellImmunity{subject: v.subject, creatures: slices.Clone(v.creatures)}
	case SuddenLearning:
		return resolveSuddenLearning(res, v)
	case RollRedo:
		if !v.withinRound {
			return wishRejected(WishRollNotRecent), nil
		}
		res.effect = WishRollRedo{mode: v.mode}
	case ReshapeReality:
		res.effect = WishReshapeReality(v)
	default:
		panic(fmt.Errorf("unreachable: wish choice %T", choice))
	}
	return res, nil
}

// resolveSuddenLearning checks the feat held, the feat required by another, then runs the
// FeatReplacement service with the feat source of the feat replaced.
func resolveSuddenLearning(res wishResolution, v SuddenLearning) (wishResolution, error) {
	var source FeatSource
	for _, t := range v.feats.All() {
		if t.Feat() == v.old {
			source = t.Source()
		}
	}
	if source == nil {
		res.rejection = ResolveWishRejection{kind: WishFeatReplacementRejected, feat: FeatNotHeld{Feat: v.old}}
		return res, nil
	}
	if slices.Contains(v.requiredBy, v.old) {
		return wishRejected(WishFeatRequiredByAnother), nil
	}
	grant, err := NewFeatGrant(source, AnyFeat{})
	if err != nil {
		return wishResolution{}, err
	}
	taking, err := FeatReplacement{}.Replace(res.character, v.feats, v.old, v.feat, grant, v.choices, v.held, v.resolved)
	if err != nil {
		return wishResolution{}, err
	}
	switch t := taking.(type) {
	case FeatTakingChanged:
		res.character, res.feats, res.events = t.character, t.feats, t.events
	case FeatTakingRejected:
		res.rejection = ResolveWishRejection{kind: WishFeatReplacementRejected, feat: t.Rejection}
		return res, nil
	}
	res.effect = WishSuddenLearning{old: v.old, repl: v.feat.ID()}
	return res, nil
}

// CheckWish runs the rules of choice for c without rolling or changing anything. The checks run in
// a fixed order: WishCannotCastAgain, then those of the choice (for Sudden Learning: the old feat
// is held, WishFeatRequiredByAnother, then the checks of the feat replacement). A rejection is a
// result, not an error.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, a nil or zero choice
// or a stored WishStress of another type. It never panics.
func CheckWish(c Character, choice WishChoice) (WishCheck, error) {
	res, err := resolveWishChoice(c, choice)
	if err != nil {
		return WishCheck{}, err
	}
	return WishCheck{rejection: res.rejection, effect: res.effect}, nil
}

// ApplyWish applies an accepted choice for c: the effect, the new Character and the events, with the
// stress of roll when WishRequiresStress (WishStressEndured follows the events of the effect, and
// the new WishStress is written to the Character). roll is ignored for DuplicateSpell. It rolls
// nothing.
//
// It returns an error matching srderr.ErrInvalidInput for the reasons CheckWish does, for a choice
// the rules refuse, and for the zero roll when the choice incurs the stress. It never panics.
func ApplyWish(c Character, choice WishChoice, roll WishStressRoll) (WishApplication, error) {
	res, err := resolveWishChoice(c, choice)
	if err != nil {
		return WishApplication{}, err
	}
	if res.rejection.kind != 0 {
		return WishApplication{}, invalidInput("apply a wish the rules refuse")
	}
	app := WishApplication{character: res.character, feats: res.feats, effect: res.effect, events: res.events}
	if !WishRequiresStress(choice) {
		return app, nil
	}
	if roll.IsZero() {
		return WishApplication{}, invalidInput("apply a wish that incurs stress without a stress roll")
	}
	next := IncurWishStress(res.stress, roll)
	app.character, err = WithFeatureState(app.character, WishStressKey(), next)
	if err != nil {
		return WishApplication{}, err
	}
	app.events = append(app.events, WishStressEndured{id: c.ID(), days: roll.days, percentile: roll.percentile, lockedOut: roll.LocksOut()})
	return app, nil
}

// WishCastOrigin says whether a cast is the cast of a spell duplicated by Wish, a closed enum. The
// zero value is an ordinary cast.
type WishCastOrigin int

// The cast origins.
const (
	// WishOrdinaryCast is a cast that pays the stress damage when it is active.
	WishOrdinaryCast WishCastOrigin = iota
	// WishDuplicateCast is the cast of a spell duplicated by Wish, which pays nothing: the Wish cast pays once.
	WishDuplicateCast
)

// String returns "ordinary_cast", "wish_duplicate" or "unknown".
func (o WishCastOrigin) String() string {
	switch o {
	case WishOrdinaryCast:
		return "ordinary_cast"
	case WishDuplicateCast:
		return "wish_duplicate"
	}
	return "unknown"
}

// WishDamageNoneReason is why a cast under the Wish stress costs no damage, a closed enum. The zero
// value is invalid.
type WishDamageNoneReason int

// The reasons, in the order they are checked.
const (
	// WishNoDamageCantrip: a cantrip has no level to multiply.
	WishNoDamageCantrip WishDamageNoneReason = iota + 1
	// WishNoDamageNotActive: no Long Rest was needed since the stress, or none came.
	WishNoDamageNotActive
	// WishNoDamageDuplicatedSpell: the cast is the duplicate of a Wish, which pays through the Wish cast.
	WishNoDamageDuplicatedSpell
)

// String returns "cantrip", "not_active", "duplicated_spell" or "unknown".
func (r WishDamageNoneReason) String() string {
	switch r {
	case WishNoDamageCantrip:
		return "cantrip"
	case WishNoDamageNotActive:
		return "not_active"
	case WishNoDamageDuplicatedSpell:
		return "duplicated_spell"
	}
	return "unknown"
}

// WishLevelOfCast returns the level of the spell for that casting: CastAtLevel of cast when it has
// one (a slot cast, a feature-granted spell of level 1 or higher), otherwise the level of spell (a
// Ritual, a cantrip). It is pure and never panics.
func WishLevelOfCast(cast SpellCast, spell content.Spell) content.SpellLevel {
	if at, ok := cast.CastAtLevel(); ok {
		return content.SpellLevel(at)
	}
	return spell.Level()
}

// WishDamageDice returns the dice of the stress damage of a cast of level: that many d10.
//
// It returns an error matching srderr.ErrInvalidInput for a level outside 1 to 9. It never panics.
func WishDamageDice(level content.SpellLevel) (dice.Expression, error) {
	if level < 1 || level > WishLevel() {
		return dice.Expression{}, invalidInput("wish stress damage dice for a level outside 1 to 9")
	}
	return dice.NewExpression(int(level), dice.D10(), 0)
}

// wishDamagePlan is the decision of RollWishStressDamage before any die: a reason for no damage, or
// the level and the dice.
type wishDamagePlan struct {
	none  WishDamageNoneReason
	level content.SpellLevel
	dice  dice.Expression
}

// planWishStressDamage validates a cast under the stress and decides whether it costs damage.
func planWishStressDamage(c Character, cast SpellCast, spell content.Spell, origin WishCastOrigin) (wishDamagePlan, error) {
	if c.IsZero() || cast.CharacterID() != c.ID() || spell.ID() != cast.Spell() || (origin != WishOrdinaryCast && origin != WishDuplicateCast) {
		return wishDamagePlan{}, invalidInput("wish stress damage for a zero character, another character's cast, another spell or an unknown origin")
	}
	if at, ok := cast.CastAtLevel(); ok && (spell.Level() == 0 || content.SpellLevel(at) < spell.Level() || !content.ValidSpellLevel(at)) {
		return wishDamagePlan{}, invalidInput("wish stress damage for a cast level below the spell's or on a cantrip")
	}
	stress, err := wishStressOf(c)
	if err != nil {
		return wishDamagePlan{}, err
	}
	level := WishLevelOfCast(cast, spell)
	switch {
	case level == 0:
		return wishDamagePlan{none: WishNoDamageCantrip}, nil
	case !stress.costly:
		return wishDamagePlan{none: WishNoDamageNotActive}, nil
	case origin == WishDuplicateCast:
		return wishDamagePlan{none: WishNoDamageDuplicatedSpell}, nil
	}
	expr, err := WishDamageDice(level)
	return wishDamagePlan{level: level, dice: expr}, err
}
