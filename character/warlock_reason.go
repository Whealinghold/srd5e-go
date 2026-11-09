package character

import "github.com/Whealinghold/srd5e-go/content"

// WarlockRejectionKind is why a Warlock feature refused to apply (class-warlock.md section 6). Its String is the
// srd5e.character.rejection value of the Warlock use cases. The zero value is not a kind and means "no rejection".
type WarlockRejectionKind int

// The kinds of Warlock rejection.
const (
	// WarlockNotAWarlock: the character has no Warlock levels, or holds no Warlock state and is not at class level 1.
	WarlockNotAWarlock WarlockRejectionKind = iota + 1
	// WarlockFeatureNotGained: the feature needs a higher Warlock level or the Fiend Patron subclass.
	WarlockFeatureNotGained
	// WarlockInvocationSelectionInvalid: the invocation selection has problems (see WarlockRejection.Problems).
	WarlockInvocationSelectionInvalid
	// WarlockInvocationNotKnown: the character does not know the invocation.
	WarlockInvocationNotKnown
	// WarlockNotASpellGrant: the invocation grants no spell.
	WarlockNotASpellGrant
	// WarlockLightLevelWrong: One with Shadows needs Dim Light or Darkness.
	WarlockLightLevelWrong
	// WarlockAlreadyCast: the slot-free cast was already made since the last Long Rest.
	WarlockAlreadyCast
	// WarlockMagicalCunningAlreadyUsed: Magical Cunning was used since the last Long Rest.
	WarlockMagicalCunningAlreadyUsed
	// WarlockRiteTooShort: the rite lasted less than 1 minute (Required and Given).
	WarlockRiteTooShort
	// WarlockContactPatronAlreadyUsed: Contact Patron was used since the last Long Rest.
	WarlockContactPatronAlreadyUsed
	// WarlockNoArcanumAtLevel: the level has no Mystic Arcanum slot.
	WarlockNoArcanumAtLevel
	// WarlockArcanumWrongSpellLevel: the arcanum spell is not of the level the slot gives (Expected and Actual).
	WarlockArcanumWrongSpellLevel
	// WarlockArcanumNotOnWarlockList: the arcanum spell is not a Warlock spell.
	WarlockArcanumNotOnWarlockList
	// WarlockArcanumTooManyReplacements: more than one arcanum spell would be replaced (Given).
	WarlockArcanumTooManyReplacements
	// WarlockNotAnArcanumSpell: the spell is not one of the character's arcanum spells.
	WarlockNotAnArcanumSpell
	// WarlockArcanumAlreadyCast: the arcanum spell was cast since the last Long Rest.
	WarlockArcanumAlreadyCast
	// WarlockNotFiendPatron: the feature needs the Fiend Patron subclass.
	WarlockNotFiendPatron
	// WarlockNotAnEnemy: Dark One's Blessing needs an enemy reduced to 0 Hit Points.
	WarlockNotAnEnemy
	// WarlockTargetTooFar: someone else reduced the enemy more than 10 feet from the Warlock (Distance).
	WarlockTargetTooFar
	// WarlockForceNotAllowed: Fiendish Resilience cannot be Force.
	WarlockForceNotAllowed
	// WarlockNotAfterRest: the choice is made when finishing a Short or Long Rest only.
	WarlockNotAfterRest
	// WarlockNotAHit: the feature needs a hit (with the pact weapon for Eldritch Smite).
	WarlockNotAHit
	// WarlockAlreadyUsedThisTurn: the once-per-turn feature was already used this turn.
	WarlockAlreadyUsedThisTurn
	// WarlockHurlThroughHellAlreadySpent: Hurl Through Hell was used since the last Long Rest.
	WarlockHurlThroughHellAlreadySpent
	// WarlockHurlThroughHellUseNotSpent: the use is not spent, so there is nothing to restore.
	WarlockHurlThroughHellUseNotSpent
	// WarlockNoPactMagicSlot: no Pact Magic slot remains.
	WarlockNoPactMagicSlot
	// WarlockProneNotAllowedForSize: Prone applies to a Huge or smaller target only (Size).
	WarlockProneNotAllowedForSize
	// WarlockNoUsesLeft: Dark One's Own Luck has no use left (Required and Available hold the maximum and the spent).
	WarlockNoUsesLeft
	// WarlockAlreadyUsedOnThisRoll: Dark One's Own Luck was already used on this roll.
	WarlockAlreadyUsedOnThisRoll
	// WarlockRollKindNotAllowed: Dark One's Own Luck adds to an ability check or a saving throw only.
	WarlockRollKindNotAllowed
	// WarlockNotMeleeSimpleOrMartial: a conjured pact weapon must be a Simple or Martial Melee weapon (Item).
	WarlockNotMeleeSimpleOrMartial
	// WarlockAttunedByAnother: another creature is attuned to the magic weapon.
	WarlockAttunedByAnother
	// WarlockBondedByAnotherWarlock: another Warlock is bonded with the magic weapon.
	WarlockBondedByAnotherWarlock
	// WarlockDamageTypeNotOffered: the damage type is not one the feature offers (DamageType).
	WarlockDamageTypeNotOffered
	// WarlockNoHitPointDie: no Hit Point Die of the chosen size can be spent.
	WarlockNoHitPointDie
	// WarlockFamiliarFormNotOffered: the familiar form is not offered (Monster).
	WarlockFamiliarFormNotOffered
	// WarlockNoFamiliar: the Warlock has no familiar form chosen.
	WarlockNoFamiliar
	// WarlockNoAttackToForgo: the Warlock has no attack left to forgo.
	WarlockNoAttackToForgo
	// WarlockReactionSpent: the Reaction is already spent.
	WarlockReactionSpent
	// WarlockNoBonusAction: the Bonus Action is already spent.
	WarlockNoBonusAction
	// WarlockDamageNotChangeable: the familiar damage type cannot be changed (DamageType).
	WarlockDamageNotChangeable
	// WarlockTargetTypeNotOffered: the damage type cannot be changed to the requested one (DamageType).
	WarlockTargetTypeNotOffered
	// WarlockNotWilling: the creature is not willing.
	WarlockNotWilling
	// WarlockNotTouched: the Warlock does not touch the creature.
	WarlockNotTouched
	// WarlockNotOnSamePlane: the creature is not on the same plane of existence.
	WarlockNotOnSamePlane
	// WarlockLinkEnded: the link was not maintained and has ended.
	WarlockLinkEnded
	// WarlockCantripCountWrong: the Book of Shadows takes exactly 3 cantrips (Required and Given).
	WarlockCantripCountWrong
	// WarlockSpellCountWrong: the Book of Shadows takes exactly 2 level 1 spells (Required and Given).
	WarlockSpellCountWrong
	// WarlockBookSpellRefused: a chosen spell breaks a rule of the Book of Shadows (Spell and BookRule).
	WarlockBookSpellRefused
	// WarlockNoPage: the Book of Shadows has no Gift of the Protectors page.
	WarlockNoPage
	// WarlockPermissionNotGiven: the Warlock has not given permission to write on the page.
	WarlockPermissionNotGiven
	// WarlockNoActionAvailable: the creature has no action to write with.
	WarlockNoActionAvailable
	// WarlockPageFull: the page holds as many names as it can (Required is the capacity).
	WarlockPageFull
	// WarlockNameAlreadyOnPage: the name is already on the page.
	WarlockNameAlreadyOnPage
	// WarlockNameNotOnPage: the creature's name is not on the page.
	WarlockNameNotOnPage
	// WarlockNoMagicAction: the Magic action is already spent.
	WarlockNoMagicAction
	// WarlockGiftAlreadySpent: the page's benefit was triggered since the last Long Rest.
	WarlockGiftAlreadySpent
	// WarlockKilledOutright: a creature killed outright does not drop to 1 Hit Point.
	WarlockKilledOutright
)

func warlockRejectionNames() []string {
	return []string{"not_a_warlock", "feature_not_gained", "invocation_selection_invalid", "invocation_not_known", "not_a_spell_grant",
		"light_level", "already_cast", "magical_cunning_already_used", "rite_too_short", "contact_patron_already_used", "no_arcanum_at_level",
		"wrong_spell_level", "not_on_warlock_list", "arcanum_too_many_replacements", "not_an_arcanum_spell", "arcanum_already_cast",
		"not_fiend_patron", "not_an_enemy", "target_too_far", "force_not_allowed", "not_after_rest", "not_a_hit", "already_used_this_turn",
		"hurl_through_hell_already_spent", "hurl_through_hell_use_not_spent", "no_pact_magic_slot", "prone_not_allowed_for_size",
		"no_uses_left", "already_used_on_this_roll", "roll_kind_not_allowed", "not_melee_simple_or_martial", "attuned_by_another",
		"bonded_by_another_warlock", "damage_type_not_offered", "no_hit_point_die", "familiar_form_not_offered", "no_familiar", "no_attack_to_forgo", "reaction_spent",
		"no_bonus_action", "damage_not_changeable", "target_type_not_offered", "not_willing", "not_touched", "not_on_same_plane", "link_ended", "cantrip_count_wrong",
		"spell_count_wrong", "book_spell_refused", "no_page", "permission_not_given", "no_action_available", "page_full", "name_already_on_page", "name_not_on_page",
		"no_magic_action", "gift_of_the_protectors_already_spent", "killed_outright"}
}

// String returns the snake_case name ("not_a_warlock", ...), the srd5e.character.rejection value, or "unknown".
func (k WarlockRejectionKind) String() string {
	if !k.Valid() {
		return "unknown"
	}
	return warlockRejectionNames()[k-1]
}

// Valid reports whether k is one of the kinds.
func (k WarlockRejectionKind) Valid() bool { return k >= 1 && int(k) <= len(warlockRejectionNames()) }

// WarlockRejection is why a Warlock operation was refused: the Kind and the figures that explain it. It is a value object held as a
// field of a result or transition, not a variant. Only the fields the Kind names are set. Construct it with a keyed literal; it is
// comparable with == only when Problems is nil.
type WarlockRejection struct {
	// Kind is the reason, never zero in a rejection.
	Kind WarlockRejectionKind
	// Feature names the feature not gained ("magical_cunning", ...) for WarlockFeatureNotGained.
	Feature string
	// Problems lists every problem of a refused invocation selection (WarlockInvocationSelectionInvalid).
	Problems []InvocationProblem
	// Option is the invocation named by WarlockInvocationNotKnown, WarlockNotASpellGrant and WarlockAlreadyCast.
	Option content.FeatureOptionID
	// LightLevel is the actual light of WarlockLightLevelWrong.
	LightLevel content.LightLevel
	// Required and Given are the least rite or the arcanum replacements asked; Available is the figure the rule compared with.
	Required, Given, Available int
	// Spell, ExpectedLevel and ActualLevel describe a refused arcanum spell.
	Spell                      content.SpellID
	ExpectedLevel, ActualLevel int
	// Distance is the feet of WarlockTargetTooFar.
	Distance int
	// Size is the target size of WarlockProneNotAllowedForSize.
	Size content.CreatureSize
	// Item is the weapon of WarlockNotMeleeSimpleOrMartial and Monster the form of WarlockFamiliarFormNotOffered.
	Item    content.ItemID
	Monster content.MonsterID
	// DamageType is the type named by the damage type kinds.
	DamageType content.DamageType
	// BookRule is the rule a spell of WarlockBookSpellRefused broke.
	BookRule BookSpellRule
}

func warlockRefuse(kind WarlockRejectionKind) WarlockRejection { return WarlockRejection{Kind: kind} }

func warlockNotGained(feature string) WarlockRejection {
	return WarlockRejection{Kind: WarlockFeatureNotGained, Feature: feature}
}
