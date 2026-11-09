package character

// BardReason is why a Bard feature refused to apply (class-bard.md FR-027 to FR-042, FR-074, FR-083, FR-086). Its String is the
// srd5e.character.rejection value of the Bard use cases. The zero value is not a reason and means "no rejection".
type BardReason int

// The reasons a Bard feature is refused.
const (
	// BardNotABard: the character has no Bard levels.
	BardNotABard BardReason = iota + 1
	// BardSelfTarget: Bardic Inspiration goes to another creature, not the Bard.
	BardSelfTarget
	// BardNoUsesRemaining: every use of Bardic Inspiration is expended.
	BardNoUsesRemaining
	// BardRecipientAlreadyInspired: the recipient already holds a Bardic Inspiration die.
	BardRecipientAlreadyInspired
	// BardBelowRequiredLevel: the feature needs a higher Bard level.
	BardBelowRequiredLevel
	// BardNothingToRestore: no use of Bardic Inspiration is expended, or the trigger restores none.
	BardNothingToRestore
	// BardSlotRejected: the spell slot cannot be expended (the SlotOperationRejection says why).
	BardSlotRejected
	// BardNotAFailure: the D20 Test the die was meant for was not failed.
	BardNotAFailure
	// BardExpired: more than one hour has passed since the die was granted.
	BardExpired
	// BardFeatureUnavailable: the Bard lacks the level or the College of Lore subclass the feature needs.
	BardFeatureUnavailable
	// BardIneligibleTrigger: the trigger is a saving throw, or its outcome is not the one the feature answers.
	BardIneligibleTrigger
	// BardIneligibleEffect: the effect applies neither the Charmed nor the Frightened condition.
	BardIneligibleEffect
	// BardSaveNotFailed: the saving throw Countercharm answers was not failed.
	BardSaveNotFailed
)

func bardReasonNames() []string {
	return []string{"not_a_bard", "self_target", "no_uses_remaining", "recipient_already_inspired", "below_required_level", "nothing_to_restore",
		"slot_rejected", "not_a_failure", "expired", "feature_unavailable", "ineligible_trigger", "ineligible_effect", "save_not_failed"}
}

// String returns the snake_case name ("not_a_bard", ...), the srd5e.character.rejection value, or "unknown".
func (r BardReason) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return bardReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r BardReason) Valid() bool { return r >= 1 && int(r) <= len(bardReasonNames()) }

// BardChoiceProblem is one thing wrong with a Bard choice (class-bard.md FR-005, FR-008, FR-015, FR-062, FR-066, FR-068, FR-071,
// FR-077, FR-091, FR-110). The choice functions list every problem together. The zero value is not a problem.
type BardChoiceProblem int

// The problems a Bard choice can have.
const (
	// BardExpertiseCountWrong: an Expertise choice does not name exactly two skills.
	BardExpertiseCountWrong BardChoiceProblem = iota + 1
	// BardExpertiseDuplicate: an Expertise choice names a skill twice.
	BardExpertiseDuplicate
	// BardExpertiseNotProficient: an Expertise choice names a skill without proficiency.
	BardExpertiseNotProficient
	// BardExpertiseAlreadyHeld: an Expertise choice names a skill that already has Expertise.
	BardExpertiseAlreadyHeld
	// BardSkillCountWrong: a skill choice does not name the number of skills the feature gives.
	BardSkillCountWrong
	// BardSkillDuplicate: a skill choice names a skill twice.
	BardSkillDuplicate
	// BardSkillAlreadyProficient: a skill choice names a skill the character is already proficient in.
	BardSkillAlreadyProficient
	// BardInstrumentCountWrong: a multiclass choice does not name exactly one Musical Instrument.
	BardInstrumentCountWrong
	// BardInstrumentNotMusical: a chosen item is not a Musical Instrument.
	BardInstrumentNotMusical
	// BardSpellCountWrong: a spell choice does not name the number of spells the feature gives.
	BardSpellCountWrong
	// BardSpellDuplicate: a spell choice names a spell twice.
	BardSpellDuplicate
	// BardSpellNotOnList: a chosen spell is on none of the lists the feature allows.
	BardSpellNotOnList
	// BardSpellWithoutSlots: a chosen spell is above the highest level the Bard has slots for.
	BardSpellWithoutSlots
	// BardSubclassTooEarly: a subclass is chosen below Bard level 3.
	BardSubclassTooEarly
	// BardSubclassAlreadyChosen: the Bard already has a subclass.
	BardSubclassAlreadyChosen
	// BardSkillInvalid: a skill choice names a value that is not a skill.
	BardSkillInvalid
	// BardSubclassUnknown: a chosen subclass is not one of the Bard definition.
	BardSubclassUnknown
	// BardSubclassMissing: the subclass owed at level 3 is not chosen.
	BardSubclassMissing
	// BardFeatMissing: a feat owed by the level is not chosen.
	BardFeatMissing
	// BardReplacementTooMany: more than one Magical Discoveries spell is replaced at one level gain.
	BardReplacementTooMany
	// BardReplacedNotHeld: the Magical Discoveries spell to replace is not held.
	BardReplacedNotHeld
)

func bardChoiceProblemNames() []string {
	return []string{"expertise_count_wrong", "expertise_duplicate", "expertise_not_proficient", "expertise_already_held", "skill_count_wrong",
		"skill_duplicate", "skill_already_proficient", "instrument_count_wrong", "instrument_not_musical", "spell_count_wrong", "spell_duplicate",
		"spell_not_on_list", "spell_without_slots", "subclass_too_early", "subclass_already_chosen", "skill_invalid", "subclass_unknown", "subclass_missing", "feat_missing",
		"replacement_too_many", "replaced_not_held"}
}

// String returns the snake_case name ("expertise_count_wrong", ...) or "unknown".
func (p BardChoiceProblem) String() string {
	if !p.Valid() {
		return "unknown"
	}
	return bardChoiceProblemNames()[p-1]
}

// Valid reports whether p is one of the problems.
func (p BardChoiceProblem) Valid() bool { return p >= 1 && int(p) <= len(bardChoiceProblemNames()) }
