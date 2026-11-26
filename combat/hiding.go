package combat

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// hideDC is the DC of the Hide check and the least FindDC of a hidden creature
// (SRD 5.2.1, "Hide [Action]", p. 183).
const hideDC d20.DifficultyClass = 15

// HideCoverDegree is the degree of cover a hider is behind. The zero value is invalid.
type HideCoverDegree int

// The degrees of cover (SRD 5.2.1, "Cover", p. 179).
const (
	// HideCoverDegreeHalfCover: Half Cover.
	HideCoverDegreeHalfCover HideCoverDegree = iota + 1
	// HideCoverDegreeThreeQuartersCover: Three-Quarters Cover.
	HideCoverDegreeThreeQuartersCover
	// HideCoverDegreeTotalCover: Total Cover.
	HideCoverDegreeTotalCover
)

// String returns the snake_case name of c or "unknown".
func (c HideCoverDegree) String() string {
	return enumName([]string{"half_cover", "three_quarters_cover", "total_cover"}, int(c))
}

// EnemySighting is what the host knows about one enemy of the hider: the enemy,
// the line from the enemy to the hider, and whether the hider can see the enemy.
// Construct it with a keyed literal; NewHideScene checks it. No function returns
// the id, so the fields are exported.
type EnemySighting struct {
	Enemy          CombatantID
	Line           LineOfSight
	HiderSeesEnemy SightRelation
}

func (e EnemySighting) valid() bool {
	return !e.Enemy.IsZero() && (e.Line == LineOfSightClear || e.Line == LineOfSightBlocked) &&
		(e.HiderSeesEnemy == SightRelationCanSee || e.HiderSeesEnemy == SightRelationCannotSee)
}

// HideScene is the host's facts about a Hide attempt: the hider, the raw
// obscurement of the hider's space, the cover, whether a larger creature obscures
// the hider, and one EnemySighting per enemy. Immutable; the lists are copied in.
// The zero HideScene is invalid and Assess reports HideBlocked for it. Safe for
// concurrent use.
type HideScene struct {
	hider            CombatantID
	space            content.Obscurement
	cover            []HideCoverDegree
	obscuredByLarger bool
	enemies          []EnemySighting
}

// NewHideScene returns the scene; the lists are copied. obscuredByLargerCreature
// is the host fact that the sole creature obscuring the hider is larger (SRD 5.2.1,
// "Naturally Stealthy", p. 86). It returns an error matching srderr.ErrInvalidInput
// for a blank hider, an undefined obscurement, an undefined cover degree or an
// invalid EnemySighting, and a LimitExceededError for more than 4 cover degrees or more than 256 enemies
// (plan P10). It never panics.
func NewHideScene(hider CombatantID, space content.Obscurement, cover []HideCoverDegree, obscuredByLargerCreature bool, enemies []EnemySighting) (HideScene, error) {
	if err := errors.Join(overCap("hide cover", len(cover), maxHideCover), overCap("hide enemies", len(enemies), maxInputList)); err != nil {
		return HideScene{}, err
	}
	switch {
	case hider.IsZero():
		return HideScene{}, invalidInput("hide scene needs a hider")
	case !space.Valid():
		return HideScene{}, invalidInput("hide scene obscurement")
	}
	for _, c := range cover {
		if c < HideCoverDegreeHalfCover || c > HideCoverDegreeTotalCover {
			return HideScene{}, invalidInput("hide scene cover degree")
		}
	}
	for _, e := range enemies {
		if !e.valid() {
			return HideScene{}, invalidInput("hide scene enemy sighting")
		}
	}
	return HideScene{hider, space, slices.Clone(cover), obscuredByLargerCreature, slices.Clone(enemies)}, nil
}

// HideBlock says which requirement of Hide is unmet. The zero value is invalid.
type HideBlock int

// The blocks.
const (
	// HideBlockNotConcealed: the hider is not Heavily Obscured or behind Three-Quarters or Total Cover.
	HideBlockNotConcealed HideBlock = iota + 1
	// HideBlockInEnemyLineOfSight: an enemy has a clear line of sight to the hider.
	HideBlockInEnemyLineOfSight
)

// String returns the snake_case name of b or "unknown".
func (b HideBlock) String() string {
	return enumName([]string{"not_concealed", "in_enemy_line_of_sight"}, int(b))
}

// HideCheckSpec is the check of the Hide action: Dexterity (Stealth) against DC 15
// (SRD 5.2.1, "Hide [Action]", p. 183). The zero value is not the Hide check.
type HideCheckSpec struct {
	ability ability.Ability
	skill   content.Skill
	dc      d20.DifficultyClass
}

// Ability returns Dexterity.
func (s HideCheckSpec) Ability() ability.Ability { return s.ability }

// Skill returns Stealth.
func (s HideCheckSpec) Skill() content.Skill { return s.skill }

// DC returns 15.
func (s HideCheckSpec) DC() d20.DifficultyClass { return s.dc }

func hideCheck() HideCheckSpec {
	return HideCheckSpec{ability.Dexterity, content.SkillStealth, hideDC}
}

// HideAssessment is the closed set of answers of HideRules.Assess. Variants:
// HideAvailable, HideBlocked.
type HideAssessment interface{ isHideAssessment() }

// HideAvailable: both requirements are met, so the hider may roll Check. It is the
// only assessment Resolve and ResolveForcedFailure accept. Construct it only from
// Assess; Resolve rejects one whose Hider is blank or whose Check is not the Hide check.
type HideAvailable struct {
	Hider     CombatantID
	Check     HideCheckSpec
	discerned []EnemySighting
}

// DiscernedSightings returns a copy of the sightings of enemies the hider can see,
// in scene order.
func (a HideAvailable) DiscernedSightings() []EnemySighting {
	return append([]EnemySighting{}, a.discerned...)
}

// HideBlocked: a requirement is unmet, so no check is rolled and the action is
// spent. Reasons lists the unmet requirements, NotConcealed then InEnemyLineOfSight.
type HideBlocked struct {
	Hider     CombatantID
	reasons   []HideBlock
	enemies   []CombatantID
	discerned []EnemySighting
}

// Reasons returns a copy of the unmet requirements.
func (b HideBlocked) Reasons() []HideBlock { return append([]HideBlock{}, b.reasons...) }

// EnemiesWithLineOfSight returns a copy of the ids of the enemies whose line is
// clear, in scene order, whether or not the hider can see them.
func (b HideBlocked) EnemiesWithLineOfSight() []CombatantID {
	return append([]CombatantID{}, b.enemies...)
}

// DiscernedSightings returns a copy of the sightings of enemies the hider can see.
func (b HideBlocked) DiscernedSightings() []EnemySighting {
	return append([]EnemySighting{}, b.discerned...)
}

func (HideAvailable) isHideAssessment() {}
func (HideBlocked) isHideAssessment()   {}

// HiddenState is the state of a hidden creature: the DC for a creature to find it.
// The zero HiddenState is invalid. Comparable; safe for concurrent use.
type HiddenState struct{ findDC d20.DifficultyClass }

// NewHiddenState returns the state. It returns an error matching
// srderr.ErrInvalidInput when findDC is below 15 (SRD 5.2.1, "Hide [Action]", p. 183).
func NewHiddenState(findDC d20.DifficultyClass) (HiddenState, error) {
	if findDC < hideDC {
		return HiddenState{}, invalidInput("find DC below the Hide DC")
	}
	return HiddenState{findDC}, nil
}

// FindDC returns the DC for a creature to find the hider.
func (s HiddenState) FindDC() d20.DifficultyClass { return s.findDC }

// HidingStatus is the closed set of hiding states of a creature. Variants:
// HidingStatusHidden, HidingStatusNotHidden. Comparable. It is never stored in a
// ConditionSet; the host holds it.
type HidingStatus interface{ isHidingStatus() }

// HidingStatusHidden: the creature is hidden. Construct it with a keyed literal
// holding a HiddenState from NewHiddenState.
type HidingStatusHidden struct{ State HiddenState }

// HidingStatusNotHidden: the creature is not hidden.
type HidingStatusNotHidden struct{}

func (HidingStatusHidden) isHidingStatus()    {}
func (HidingStatusNotHidden) isHidingStatus() {}

// HideResolution is the closed set of results of a Hide attempt. Variants:
// HideSucceeded, HideCheckFailed, HideNotRolled.
type HideResolution interface{ isHideResolution() }

// HideSucceeded: the creature is hidden. It carries the state and the events, one CreatureHid.
type HideSucceeded struct {
	state  HiddenState
	events []Event
}

// State returns the new hidden state.
func (r HideSucceeded) State() HiddenState { return r.state }

// Events returns a copy of the events.
func (r HideSucceeded) Events() []Event { return append([]Event{}, r.events...) }

// HideCheckFailed: the check total was below the DC. There is no state and no event.
type HideCheckFailed struct{ total int }

// Total returns the failed total.
func (r HideCheckFailed) Total() int { return r.total }

// HideNotRolled: the check failed without a roll. There is no state and no event.
type HideNotRolled struct{ reason d20.ForcedFailureReason }

// Reason returns why the check was not rolled.
func (r HideNotRolled) Reason() d20.ForcedFailureReason { return r.reason }

func (HideSucceeded) isHideResolution()   {}
func (HideCheckFailed) isHideResolution() {}
func (HideNotRolled) isHideResolution()   {}

// HidingActivity is something that happens to a hidden creature. The zero value is invalid.
type HidingActivity int

// The activities (SRD 5.2.1, "Hide [Action]", p. 183).
const (
	// HidingActivitySoundLouderThanWhisper: the creature makes a sound louder than a whisper.
	HidingActivitySoundLouderThanWhisper HidingActivity = iota + 1
	// HidingActivitySoundNoLouderThanWhisper: the creature makes a sound no louder than a whisper.
	HidingActivitySoundNoLouderThanWhisper
	// HidingActivityFoundByEnemy: an enemy finds the creature.
	HidingActivityFoundByEnemy
	// HidingActivityFoundByNonEnemy: a creature that is not an enemy finds it.
	HidingActivityFoundByNonEnemy
	// HidingActivityAttackRoll: the creature makes an attack roll.
	HidingActivityAttackRoll
	// HidingActivitySpellWithVerbalComponent: the creature casts a spell with a Verbal component.
	HidingActivitySpellWithVerbalComponent
	// HidingActivitySpellWithoutVerbalComponent: the creature casts a spell without one.
	HidingActivitySpellWithoutVerbalComponent
)

// Valid reports whether a is one of the seven activities.
func (a HidingActivity) Valid() bool {
	return a >= HidingActivitySoundLouderThanWhisper && a <= HidingActivitySpellWithoutVerbalComponent
}

// String returns the snake_case name of a or "unknown".
func (a HidingActivity) String() string {
	return enumName([]string{"sound_louder_than_whisper", "sound_no_louder_than_whisper", "found_by_enemy",
		"found_by_non_enemy", "attack_roll", "spell_with_verbal_component", "spell_without_verbal_component"}, int(a))
}

// HidingEffect says what an activity does to hiding. The zero value is invalid.
type HidingEffect int

// The effects.
const (
	// HidingEffectRemainsHidden: the creature stays hidden.
	HidingEffectRemainsHidden HidingEffect = iota + 1
	// HidingEffectStopsBeingHidden: the creature stops being hidden.
	HidingEffectStopsBeingHidden
)

// HidingFinder says whether a creature that looks for a hider is its enemy. The zero value is invalid.
type HidingFinder int

// The finders.
const (
	// HidingFinderEnemy: an enemy of the hider.
	HidingFinderEnemy HidingFinder = iota + 1
	// HidingFinderNotEnemy: a creature that is not an enemy.
	HidingFinderNotEnemy
)

// HidingChange is the new status of a hider and the events that produced it. The zero value is invalid.
type HidingChange struct {
	status HidingStatus
	events []Event
}

// Status returns the new status.
func (c HidingChange) Status() HidingStatus { return c.status }

// Events returns a copy of the events: none, or one HidingEnded.
func (c HidingChange) Events() []Event { return append([]Event{}, c.events...) }

// HideRules is the stateless domain service for the Hide action (SRD 5.2.1, "Hide
// [Action]", p. 183). Pure, synchronous and silent; the zero value is ready to use.
// Safe for concurrent use.
type HideRules struct{}

// Assess says whether the hider may roll the Hide check. The concealment
// requirement is met when the hider's space is Heavily Obscured, the hider is behind
// Three-Quarters or Total Cover, or a larger creature obscures it; the line-of-sight
// requirement is met when every enemy's line is blocked (so with no enemies it is
// met). It returns HideAvailable when both are met, else HideBlocked with the unmet
// requirements, NotConcealed then InEnemyLineOfSight, and the enemies whose line is
// clear. Both carry the sightings of the enemies the hider can see. The current
// hiding status is ignored, so a hider may hide again. The zero HideScene is HideBlocked
// with NotConcealed.
func (HideRules) Assess(scene HideScene) HideAssessment {
	var discerned []EnemySighting
	var clear []CombatantID
	for _, e := range scene.enemies {
		if e.HiderSeesEnemy == SightRelationCanSee {
			discerned = append(discerned, e)
		}
		if e.Line == LineOfSightClear {
			clear = append(clear, e.Enemy)
		}
	}
	concealed := scene.obscuredByLarger || scene.space == content.ObscurementHeavily ||
		slices.Contains(scene.cover, HideCoverDegreeThreeQuartersCover) || slices.Contains(scene.cover, HideCoverDegreeTotalCover)
	var reasons []HideBlock
	if !concealed || scene.hider.IsZero() {
		reasons = append(reasons, HideBlockNotConcealed)
	}
	if len(clear) > 0 {
		reasons = append(reasons, HideBlockInEnemyLineOfSight)
	}
	if len(reasons) == 0 {
		return HideAvailable{scene.hider, hideCheck(), discerned}
	}
	return HideBlocked{scene.hider, reasons, clear, discerned}
}

func (a HideAvailable) usable() bool { return !a.Hider.IsZero() && a.Check == hideCheck() }

// Resolve records a rolled Hide check. Success needs a total of at least 15 and
// returns HideSucceeded with FindDC equal to the total and one CreatureHid;
// failure needs a total below 15 and returns HideCheckFailed with no event. It
// returns an error matching srderr.ErrInvalidInput when the outcome is neither
// success nor failure, when outcome and total disagree about the DC, or when a is
// not from Assess. It never panics.
func (HideRules) Resolve(a HideAvailable, total int, outcome d20.Outcome) (HideResolution, error) {
	if !a.usable() {
		return nil, invalidInput("hide assessment not from Assess")
	}
	switch {
	case outcome == d20.Success && total >= int(hideDC):
		findDC := d20.DifficultyClass(total)
		return HideSucceeded{HiddenState{findDC}, []Event{CreatureHid{hidingEventBase{a.Hider}, findDC}}}, nil
	case outcome == d20.Failure && total < int(hideDC):
		return HideCheckFailed{total}, nil
	}
	return nil, invalidInput("hide outcome disagrees with the total")
}

// ResolveForcedFailure records a Hide check that failed without a roll and has no
// total. The result has no state and no event. It returns an error matching
// srderr.ErrInvalidInput when a is not from Assess. It never panics.
func (HideRules) ResolveForcedFailure(a HideAvailable, reason d20.ForcedFailureReason) (HideNotRolled, error) {
	if !a.usable() {
		return HideNotRolled{}, invalidInput("hide assessment not from Assess")
	}
	return HideNotRolled{reason}, nil
}

// EffectOf returns what the activity does to hiding: StopsBeingHidden for a sound
// louder than a whisper, being found by an enemy, an attack roll and a spell with a
// Verbal component; RemainsHidden for the other three (SRD 5.2.1, "Hide [Action]",
// p. 183). It returns the zero HidingEffect for an undefined activity.
func (HideRules) EffectOf(a HidingActivity) HidingEffect {
	switch a {
	case HidingActivitySoundLouderThanWhisper, HidingActivityFoundByEnemy, HidingActivityAttackRoll, HidingActivitySpellWithVerbalComponent:
		return HidingEffectStopsBeingHidden
	case HidingActivitySoundNoLouderThanWhisper, HidingActivityFoundByNonEnemy, HidingActivitySpellWithoutVerbalComponent:
		return HidingEffectRemainsHidden
	}
	return 0
}

// ApplyActivity applies the activity to the hider's status, defined by EffectOf. A
// Hidden hider whose activity StopsBeingHidden becomes NotHidden with one
// HidingEnded; otherwise the status is returned unchanged with no event, including
// any activity on a NotHidden hider and an undefined activity or a blank hider.
// The status passed in is not changed, so for AttackRoll the caller still holds
// Hidden when it rolls.
func (r HideRules) ApplyActivity(status HidingStatus, activity HidingActivity, hider CombatantID) HidingChange {
	if _, hidden := status.(HidingStatusHidden); hidden && r.EffectOf(activity) == HidingEffectStopsBeingHidden {
		if ev, err := NewHidingEnded(hider, activity); err == nil {
			return HidingChange{HidingStatusNotHidden{}, []Event{ev}}
		}
	}
	return HidingChange{status, nil}
}

// AttemptToFind applies a Wisdom (Perception) total, active or passive, that a
// creature found by looking for the hider. For an enemy whose total is at least the
// FindDC the hiding ends with one HidingEnded(FoundByEnemy); below it, and for a
// creature that is not an enemy, nothing changes.
func (r HideRules) AttemptToFind(status HidingStatus, hider CombatantID, finder HidingFinder, total int) HidingChange {
	if h, ok := status.(HidingStatusHidden); ok && finder == HidingFinderEnemy && total >= int(h.State.findDC) {
		return r.ApplyActivity(status, HidingActivityFoundByEnemy, hider)
	}
	return HidingChange{status, nil}
}

// ImpliedConditions returns a new slice holding Invisible while the creature is
// hidden and nothing otherwise (SRD 5.2.1, "Hide [Action]", p. 183). The benefit
// lapses against a creature that can somehow see the hider. The host unions it into
// the effective condition sets; it is never stored.
func (HideRules) ImpliedConditions(status HidingStatus) []content.ConditionKind {
	if _, ok := status.(HidingStatusHidden); ok {
		return []content.ConditionKind{content.ConditionInvisible}
	}
	return []content.ConditionKind{}
}
