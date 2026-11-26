package combat

import (
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The terms of the Swallow, Engulf and Whelm actions: the numbers of a stat block, which the host
// transcribes (monster-engulf-and-swallow.md section 4.1). Every term is an immutable value, comparable
// with ==, whose zero value is invalid unless its doc comment says it means "absent". Every constructor
// returns a *ContainmentError (matching srderr.ErrInvalidInput) with ContainmentProblemInvalidTerms and
// never panics; the Must forms panic with that error and only then. All are safe for concurrent use.

// ContainmentMode is how a creature is held inside another (SRD 5.2.1, "Behir", p. 263;
// "Gelatinous Cube", p. 287; "Shambling Mound", p. 323). The zero value is invalid.
type ContainmentMode int

// The three modes.
const (
	// ContainmentModeSwallowed: inside the monster; the Grappled condition ends first.
	ContainmentModeSwallowed ContainmentMode = iota + 1
	// ContainmentModeEngulfed: inside the cube's space, with no grapple.
	ContainmentModeEngulfed
	// ContainmentModeEngulfedGrappled: held by a grapple of the container ("Until the grapple ends").
	ContainmentModeEngulfedGrappled
)

// String returns "swallowed", "engulfed", "engulfed_grappled" or "unknown".
func (m ContainmentMode) String() string {
	return enumName([]string{"swallowed", "engulfed", "engulfed_grappled"}, int(m))
}

// Valid reports whether m is one of the three modes.
func (m ContainmentMode) Valid() bool {
	return m >= ContainmentModeSwallowed && m <= ContainmentModeEngulfedGrappled
}

// SuffocationRule says whether a held creature is suffocating (SRD 5.2.1, "Suffocation [Hazard]", p. 189).
// The zero value is invalid.
type SuffocationRule int

// The three rules.
const (
	// SuffocationRuleNone: the creature is not suffocating.
	SuffocationRuleNone SuffocationRule = iota + 1
	// SuffocationRuleAlways: the creature is suffocating.
	SuffocationRuleAlways
	// SuffocationRuleUnlessBreathesWater: suffocating unless the creature can breathe water.
	SuffocationRuleUnlessBreathesWater
)

// String returns "none", "always", "unless_breathes_water" or "unknown".
func (r SuffocationRule) String() string {
	return enumName([]string{"none", "always", "unless_breathes_water"}, int(r))
}

// Valid reports whether r is one of the three rules.
func (r SuffocationRule) Valid() bool {
	return r >= SuffocationRuleNone && r <= SuffocationRuleUnlessBreathesWater
}

// ContainmentTiming is when a container's periodic damage is due. The zero value is invalid.
type ContainmentTiming int

// The two timings.
const (
	// ContainmentTimingStartOfContainerTurn: the start of the container's turn.
	ContainmentTimingStartOfContainerTurn ContainmentTiming = iota + 1
	// ContainmentTimingEndOfContainerTurn: the end of the container's turn.
	ContainmentTimingEndOfContainerTurn
)

// String returns "start_of_container_turn", "end_of_container_turn" or "unknown".
func (t ContainmentTiming) String() string {
	return enumName([]string{"start_of_container_turn", "end_of_container_turn"}, int(t))
}

// Valid reports whether t is one of the two timings.
func (t ContainmentTiming) Valid() bool {
	return t >= ContainmentTimingStartOfContainerTurn && t <= ContainmentTimingEndOfContainerTurn
}

// ContainmentActionKind is the action a Swallow or Engulf costs the monster (SRD 5.2.1, "Behir", p. 263;
// "Kraken", p. 303). The zero value is invalid.
type ContainmentActionKind int

// The two kinds.
const (
	// ContainmentActionKindBonusAction: a Bonus Action.
	ContainmentActionKindBonusAction ContainmentActionKind = iota + 1
	// ContainmentActionKindAction: an Action.
	ContainmentActionKindAction
)

// String returns "bonus_action", "action" or "unknown".
func (k ContainmentActionKind) String() string {
	return enumName([]string{"bonus_action", "action"}, int(k))
}

// Valid reports whether k is one of the two kinds.
func (k ContainmentActionKind) Valid() bool {
	return k >= ContainmentActionKindBonusAction && k <= ContainmentActionKindAction
}

// termsError is the error of every terms constructor.
func termsError() error { return &ContainmentError{Problem: ContainmentProblemInvalidTerms} }

// ContainmentCapacity is how many creatures a container holds: a number of slots and what each size of
// creature costs (SRD 5.2.1, "Gelatinous Cube", p. 287: one Large creature or up to four Medium or Small
// ones). The zero value is invalid. Build it with NewContainmentCapacity.
type ContainmentCapacity struct {
	slots int
	costs [6]int // by content.CreatureSize-1; 0 means the default cost of 1
}

// NewContainmentCapacity returns a capacity of totalSlots (1 or more) in which a creature of a size
// costs the mapped cost (between 1 and totalSlots), or 1 for a size not in costs. It returns a
// *ContainmentError (matching srderr.ErrInvalidInput) for fewer than 1 slot, an invalid size or a cost
// outside the range. It never panics.
func NewContainmentCapacity(totalSlots int, costs map[content.CreatureSize]int) (ContainmentCapacity, error) {
	if totalSlots < 1 {
		return ContainmentCapacity{}, termsError()
	}
	c := ContainmentCapacity{slots: totalSlots}
	for size, cost := range costs {
		if !size.Valid() || cost < 1 || cost > totalSlots {
			return ContainmentCapacity{}, termsError()
		}
		if cost != 1 { // 1 is the default, so equal capacities compare equal
			c.costs[size-1] = cost
		}
	}
	return c, nil
}

// MustContainmentCapacity is NewContainmentCapacity for literals known to be valid. It panics with the
// error NewContainmentCapacity returns, and only then.
func MustContainmentCapacity(totalSlots int, costs map[content.CreatureSize]int) ContainmentCapacity {
	c, err := NewContainmentCapacity(totalSlots, costs)
	if err != nil {
		panic(err)
	}
	return c
}

// Slots returns the number of slots.
func (c ContainmentCapacity) Slots() int { return c.slots }

// Cost returns the slots a creature of the size takes: the mapped cost or 1.
func (c ContainmentCapacity) Cost(size content.CreatureSize) int {
	if size.Valid() && c.costs[size-1] > 0 {
		return c.costs[size-1]
	}
	return 1
}

// maxDamageTerms is the number of damage terms a block holds: the Remorhaz prints two (SRD 5.2.1,
// "Remorhaz", p. 320). A design limit that keeps every term comparable with ==.
const maxDamageTerms = 2

// DamageTerm is one dice expression of a stat block and its damage type, such as 3d6 Acid. The printed
// average is not stored. The zero value is invalid. Build it with NewDamageTerm.
type DamageTerm struct {
	expression dice.Expression
	kind       content.DamageType
}

// NewDamageTerm returns the term. It returns a *ContainmentError (matching srderr.ErrInvalidInput) for
// an expression with no die (a fixed number) or an invalid damage type. It never panics.
func NewDamageTerm(expression dice.Expression, kind content.DamageType) (DamageTerm, error) {
	if expression.Count() < 1 || !kind.Valid() {
		return DamageTerm{}, termsError()
	}
	return DamageTerm{expression, kind}, nil
}

// MustDamageTerm is NewDamageTerm for literals known to be valid. It panics with the error NewDamageTerm
// returns, and only then.
func MustDamageTerm(expression dice.Expression, kind content.DamageType) DamageTerm {
	t, err := NewDamageTerm(expression, kind)
	if err != nil {
		panic(err)
	}
	return t
}

// Expression returns the dice the host rolls.
func (t DamageTerm) Expression() dice.Expression { return t.expression }

// DamageType returns the type of the damage.
func (t DamageTerm) DamageType() content.DamageType { return t.kind }

func (t DamageTerm) valid() bool { return t.expression.Count() >= 1 && t.kind.Valid() }

// damageTerms is a short list of damage terms held in an array, so a value holding it stays comparable.
type damageTerms struct {
	n     int
	items [maxDamageTerms]DamageTerm
}

// newDamageTerms copies list, which must hold between min and maxDamageTerms valid terms.
func newDamageTerms(list []DamageTerm, min int) (damageTerms, bool) {
	if len(list) < min || len(list) > maxDamageTerms {
		return damageTerms{}, false
	}
	d := damageTerms{n: len(list)}
	for i, t := range list {
		if !t.valid() {
			return damageTerms{}, false
		}
		d.items[i] = t
	}
	return d, true
}

// list returns a new slice of the terms.
func (d damageTerms) list() []DamageTerm { return append([]DamageTerm(nil), d.items[:d.n]...) }

// EntryTerms is how a creature comes to be held: with no save or with a saving throw. Variants:
// EntryAutomatic, EntryBySave. A nil EntryTerms is invalid.
type EntryTerms interface{ isEntryTerms() }

// EntryAutomatic: the monster's action needs no save (SRD 5.2.1, "Giant Frog", p. 351; "Giant Toad",
// p. 354). The zero value is that entry.
type EntryAutomatic struct{}

// EntryBySave: the target makes a saving throw. Build it with NewEntryBySave; the zero value is invalid.
type EntryBySave struct {
	ability       ability.Ability
	dc            d20.DifficultyClass
	damage        damageTerms
	halfOnSuccess bool
	relocateFeet  int
	targetAnySize bool
}

func (EntryAutomatic) isEntryTerms() {}
func (EntryBySave) isEntryTerms()    {}

// NewEntryBySave returns the entry: a saving throw of ability against dc (1 or more) that, on a
// failure, deals the damage terms (none or up to two) and holds the creature. halfOnSuccess halves the
// damage on a success and needs damage; relocateFeet, when more than 0, moves a creature that saved to
// a free space within that distance (SRD 5.2.1, "Gelatinous Cube", p. 287); targetAnySize makes the
// save and the damage apply to a target too large to be held (p. 287, p. 303). It returns a
// *ContainmentError (matching srderr.ErrInvalidInput) for an invalid ability, a dc below 1, an invalid
// or too long damage list, halfOnSuccess without damage or a negative relocateFeet. It never panics.
func NewEntryBySave(a ability.Ability, dc d20.DifficultyClass, damage []DamageTerm, halfOnSuccess bool,
	relocateFeet int, targetAnySize bool) (EntryBySave, error) {
	d, ok := newDamageTerms(damage, 0)
	if !ok || !a.Valid() || dc < 1 || (halfOnSuccess && d.n == 0) || relocateFeet < 0 {
		return EntryBySave{}, termsError()
	}
	return EntryBySave{a, dc, d, halfOnSuccess, relocateFeet, targetAnySize}, nil
}

// MustEntryBySave is NewEntryBySave for literals known to be valid. It panics with the error
// NewEntryBySave returns, and only then.
func MustEntryBySave(a ability.Ability, dc d20.DifficultyClass, damage []DamageTerm, halfOnSuccess bool,
	relocateFeet int, targetAnySize bool) EntryBySave {
	e, err := NewEntryBySave(a, dc, damage, halfOnSuccess, relocateFeet, targetAnySize)
	if err != nil {
		panic(err)
	}
	return e
}

// Ability returns the ability of the saving throw.
func (e EntryBySave) Ability() ability.Ability { return e.ability }

// DifficultyClass returns the DC of the saving throw.
func (e EntryBySave) DifficultyClass() d20.DifficultyClass { return e.dc }

// Damage returns a new slice of the damage the creature takes on a failure; empty for none.
func (e EntryBySave) Damage() []DamageTerm { return e.damage.list() }

// HalfOnSuccess says whether a success takes half the damage.
func (e EntryBySave) HalfOnSuccess() bool { return e.halfOnSuccess }

// RelocateFeet returns the distance a creature that saved is moved, 0 for none.
func (e EntryBySave) RelocateFeet() int { return e.relocateFeet }

// TargetAnySize says whether a target above the maximum size still makes the save.
func (e EntryBySave) TargetAnySize() bool { return e.targetAnySize }

func (e EntryBySave) valid() bool {
	return e.ability.Valid() && e.dc >= 1 && (!e.halfOnSuccess || e.damage.n > 0) && e.relocateFeet >= 0
}

// PeriodicDamage is the damage a container deals to what it holds each turn. The zero PeriodicDamage
// means no periodic damage. Build it with NewPeriodicDamage.
type PeriodicDamage struct {
	damage            damageTerms
	timing            ContainmentTiming
	firstTickNextTurn bool
}

// NewPeriodicDamage returns the damage: the terms (one or two) dealt at timing. firstTickNextTurn
// skips the end of the turn the creature was swallowed on (SRD 5.2.1, "Giant Frog", p. 351) and needs
// ContainmentTimingEndOfContainerTurn. It returns a *ContainmentError (matching srderr.ErrInvalidInput)
// for an empty or invalid list, an invalid timing or the flag with the start of the turn. It never
// panics.
func NewPeriodicDamage(damage []DamageTerm, timing ContainmentTiming, firstTickNextTurn bool) (PeriodicDamage, error) {
	d, ok := newDamageTerms(damage, 1)
	if !ok || !timing.Valid() || (firstTickNextTurn && timing != ContainmentTimingEndOfContainerTurn) {
		return PeriodicDamage{}, termsError()
	}
	return PeriodicDamage{d, timing, firstTickNextTurn}, nil
}

// MustPeriodicDamage is NewPeriodicDamage for literals known to be valid. It panics with the error
// NewPeriodicDamage returns, and only then.
func MustPeriodicDamage(damage []DamageTerm, timing ContainmentTiming, firstTickNextTurn bool) PeriodicDamage {
	p, err := NewPeriodicDamage(damage, timing, firstTickNextTurn)
	if err != nil {
		panic(err)
	}
	return p
}

// Damage returns a new slice of the terms.
func (p PeriodicDamage) Damage() []DamageTerm { return p.damage.list() }

// Timing returns when the damage is due.
func (p PeriodicDamage) Timing() ContainmentTiming { return p.timing }

// FirstTickNextTurn says whether the first tick is deferred past the turn of the swallow.
func (p PeriodicDamage) FirstTickNextTurn() bool { return p.firstTickNextTurn }

func (p PeriodicDamage) present() bool { return p.damage.n > 0 }

// Regurgitation is the damage threshold at which a swallowing container regurgitates (SRD 5.2.1,
// "Purple Worm", p. 317). The zero Regurgitation means none. Build it with NewRegurgitation.
type Regurgitation struct{ threshold, dc, landing int }

// NewRegurgitation returns the rule: a container that takes thresholdDamage or more from a creature
// inside it in one turn makes a Constitution saving throw against constitutionDC and, on a failure,
// releases everything within landingFeet. It returns a *ContainmentError (matching
// srderr.ErrInvalidInput) for a value below 1. It never panics.
func NewRegurgitation(thresholdDamage, constitutionDC, landingFeet int) (Regurgitation, error) {
	if thresholdDamage < 1 || constitutionDC < 1 || landingFeet < 1 {
		return Regurgitation{}, termsError()
	}
	return Regurgitation{thresholdDamage, constitutionDC, landingFeet}, nil
}

// MustRegurgitation is NewRegurgitation for literals known to be valid. It panics with the error
// NewRegurgitation returns, and only then.
func MustRegurgitation(thresholdDamage, constitutionDC, landingFeet int) Regurgitation {
	r, err := NewRegurgitation(thresholdDamage, constitutionDC, landingFeet)
	if err != nil {
		panic(err)
	}
	return r
}

// ThresholdDamage returns the damage from one creature in one turn that triggers the save.
func (r Regurgitation) ThresholdDamage() int { return r.threshold }

// ConstitutionDC returns the DC of the container's Constitution saving throw.
func (r Regurgitation) ConstitutionDC() d20.DifficultyClass { return d20.DifficultyClass(r.dc) }

// LandingFeet returns the distance within which the released creatures land.
func (r Regurgitation) LandingFeet() int { return r.landing }

func (r Regurgitation) present() bool { return r.threshold > 0 }

// ContainerDeathRule is what happens to the held creatures when the container dies (SRD 5.2.1,
// "Tarrasque", p. 332). Variants: ReleaseOnDeath, EscapeFromCorpse. A nil ContainerDeathRule is
// invalid.
type ContainerDeathRule interface{ isContainerDeathRule() }

// ReleaseOnDeath: every held creature is released where the container was. The zero value is that rule.
type ReleaseOnDeath struct{}

// EscapeFromCorpse: the held creatures stay inside until each uses feet of movement to climb out. Build
// it with NewEscapeFromCorpse; the zero value is invalid.
type EscapeFromCorpse struct{ feet int }

func (ReleaseOnDeath) isContainerDeathRule()   {}
func (EscapeFromCorpse) isContainerDeathRule() {}

// NewEscapeFromCorpse returns the rule. It returns a *ContainmentError (matching srderr.ErrInvalidInput)
// for fewer than 1 foot. It never panics.
func NewEscapeFromCorpse(feet int) (EscapeFromCorpse, error) {
	if feet < 1 {
		return EscapeFromCorpse{}, termsError()
	}
	return EscapeFromCorpse{feet}, nil
}

// MustEscapeFromCorpse is NewEscapeFromCorpse for literals known to be valid. It panics with the error
// NewEscapeFromCorpse returns, and only then.
func MustEscapeFromCorpse(feet int) EscapeFromCorpse {
	e, err := NewEscapeFromCorpse(feet)
	if err != nil {
		panic(err)
	}
	return e
}

// Feet returns the movement a creature spends to climb out.
func (e EscapeFromCorpse) Feet() int { return e.feet }

// ExitTerms is how a held creature gets out while the container lives (SRD 5.2.1, "Gelatinous Cube",
// p. 287). The zero ExitTerms means no way out. Build it with NewExitTerms.
type ExitTerms struct {
	selfEscapeDC, pullDC d20.DifficultyClass
	pullerDamage         damageTerms
}

// NewExitTerms returns the terms: a Strength (Athletics) DC for the held creature to escape on its own
// and a DC for another creature to pull it out, 0 meaning none. pullerDamage (up to two terms) is dealt
// to the puller and needs a pullDC. It returns a *ContainmentError (matching srderr.ErrInvalidInput) for
// no DC at all, a negative DC or an invalid pullerDamage. It never panics.
func NewExitTerms(selfEscapeDC, pullDC d20.DifficultyClass, pullerDamage []DamageTerm) (ExitTerms, error) {
	d, ok := newDamageTerms(pullerDamage, 0)
	if !ok || selfEscapeDC < 0 || pullDC < 0 || (selfEscapeDC == 0 && pullDC == 0) || (d.n > 0 && pullDC == 0) {
		return ExitTerms{}, termsError()
	}
	return ExitTerms{selfEscapeDC, pullDC, d}, nil
}

// MustExitTerms is NewExitTerms for literals known to be valid. It panics with the error NewExitTerms
// returns, and only then.
func MustExitTerms(selfEscapeDC, pullDC d20.DifficultyClass, pullerDamage []DamageTerm) ExitTerms {
	e, err := NewExitTerms(selfEscapeDC, pullDC, pullerDamage)
	if err != nil {
		panic(err)
	}
	return e
}

// SelfEscapeDC returns the DC of the held creature's own escape, 0 for none.
func (e ExitTerms) SelfEscapeDC() d20.DifficultyClass { return e.selfEscapeDC }

// PullDC returns the DC of a pull, 0 for none.
func (e ExitTerms) PullDC() d20.DifficultyClass { return e.pullDC }

// PullerDamage returns a new slice of the damage the puller takes; empty for none.
func (e ExitTerms) PullerDamage() []DamageTerm { return e.pullerDamage.list() }

func (e ExitTerms) present() bool { return e.selfEscapeDC > 0 || e.pullDC > 0 }

// GrappleHoldTerms is the grapple that holds an engulfed creature (SRD 5.2.1, "Shambling Mound", p. 323:
// "Grappled (escape DC 14)"). The zero value means none. Build it with NewGrappleHoldTerms.
type GrappleHoldTerms struct {
	escapeDC d20.DifficultyClass
	partName string
}

// NewGrappleHoldTerms returns the terms: the escape DC (1 or more) and the name of the part that holds,
// to which the engine appends the ordinal of the creature. It returns a *ContainmentError (matching
// srderr.ErrInvalidInput) for a DC below 1 or a name that is empty or has leading or trailing
// whitespace. It never panics.
func NewGrappleHoldTerms(escapeDC d20.DifficultyClass, partName string) (GrappleHoldTerms, error) {
	if escapeDC < 1 || partName == "" || partName != strings.TrimSpace(partName) {
		return GrappleHoldTerms{}, termsError()
	}
	return GrappleHoldTerms{escapeDC, partName}, nil
}

// MustGrappleHoldTerms is NewGrappleHoldTerms for literals known to be valid. It panics with the error
// NewGrappleHoldTerms returns, and only then.
func MustGrappleHoldTerms(escapeDC d20.DifficultyClass, partName string) GrappleHoldTerms {
	g, err := NewGrappleHoldTerms(escapeDC, partName)
	if err != nil {
		panic(err)
	}
	return g
}

// EscapeDC returns the escape DC of the grapple.
func (g GrappleHoldTerms) EscapeDC() d20.DifficultyClass { return g.escapeDC }

// PartName returns the name of the part that holds.
func (g GrappleHoldTerms) PartName() string { return g.partName }

func (g GrappleHoldTerms) present() bool { return g.escapeDC > 0 }

// ContainmentTermsInput is the raw values of a ContainmentTerms. Optional members are absent at their
// zero value. Build it with a keyed literal.
type ContainmentTermsInput struct {
	// Mode is how the creature is held.
	Mode ContainmentMode
	// MaxSize is the largest creature that can be held; equal is allowed.
	MaxSize content.CreatureSize
	// Capacity is how many creatures fit.
	Capacity ContainmentCapacity
	// Entry is how the creature comes to be held.
	Entry EntryTerms
	// Imposed lists the conditions a held creature gets: Blinded and Restrained are the two the SRD imposes.
	Imposed []content.ConditionKind
	// TotalCover says whether a held creature has Total Cover against what is outside.
	TotalCover bool
	// Suffocation says whether a held creature is suffocating.
	Suffocation SuffocationRule
	// NoVerbalSpells says a held creature cannot cast a spell with a Verbal component.
	NoVerbalSpells bool
	// NoTeleport says a held creature cannot teleport.
	NoTeleport bool
	// Periodic is the damage dealt each turn; absent at the zero value.
	Periodic PeriodicDamage
	// Regurgitation is the damage threshold rule; absent at the zero value, valid only when swallowed.
	Regurgitation Regurgitation
	// DeathRule is what happens when the container dies.
	DeathRule ContainerDeathRule
	// Exit is how a held creature gets out; absent at the zero value.
	Exit ExitTerms
	// Grapple is the grapple of ContainmentModeEngulfedGrappled, and only of it.
	Grapple GrappleHoldTerms
	// DisgorgeAfterFirstDamage makes the container release a creature that survives its first periodic damage.
	DisgorgeAfterFirstDamage bool
	// LockedAction names an action the container cannot take while it holds a creature; empty for none.
	LockedAction string
	// ActionKind is what the Swallow or Engulf costs the monster.
	ActionKind ContainmentActionKind
}

// ContainmentTerms is the numbers of one monster's Swallow, Engulf or Whelm, as transcribed by the host.
// The zero value is invalid. Build it with NewContainmentTerms. Immutable and comparable with ==.
type ContainmentTerms struct {
	mode                     ContainmentMode
	maxSize                  content.CreatureSize
	capacity                 ContainmentCapacity
	entry                    EntryTerms
	imposed                  kindMask
	totalCover               bool
	suffocation              SuffocationRule
	noVerbalSpells           bool
	noTeleport               bool
	periodic                 PeriodicDamage
	regurgitation            Regurgitation
	deathRule                ContainerDeathRule
	exit                     ExitTerms
	grapple                  GrappleHoldTerms
	disgorgeAfterFirstDamage bool
	lockedAction             string
	actionKind               ContainmentActionKind
}

// NewContainmentTerms returns the terms. It returns a *ContainmentError (matching srderr.ErrInvalidInput)
// with ContainmentProblemInvalidTerms for: an invalid mode, size, capacity, entry, suffocation rule,
// action kind or death rule; an imposed condition other than Blinded and Restrained; a Grapple unless the
// mode is ContainmentModeEngulfedGrappled (and none with it); a Regurgitation unless swallowed; a
// DisgorgeAfterFirstDamage without Periodic; a LockedAction that is not empty and has leading or trailing
// whitespace; or any member that its own constructor refuses. It copies in and never panics.
func NewContainmentTerms(in ContainmentTermsInput) (ContainmentTerms, error) {
	t := ContainmentTerms{in.Mode, in.MaxSize, in.Capacity, in.Entry, 0, in.TotalCover, in.Suffocation, in.NoVerbalSpells,
		in.NoTeleport, in.Periodic, in.Regurgitation, in.DeathRule, in.Exit, in.Grapple, in.DisgorgeAfterFirstDamage,
		in.LockedAction, in.ActionKind}
	for _, k := range in.Imposed {
		if k != content.ConditionBlinded && k != content.ConditionRestrained {
			return ContainmentTerms{}, termsError()
		}
		t.imposed |= bit(k)
	}
	if !t.valid() {
		return ContainmentTerms{}, termsError()
	}
	return t, nil
}

// MustContainmentTerms is NewContainmentTerms for literals known to be valid. It panics with the error
// NewContainmentTerms returns, and only then.
func MustContainmentTerms(in ContainmentTermsInput) ContainmentTerms {
	t, err := NewContainmentTerms(in)
	if err != nil {
		panic(err)
	}
	return t
}

// valid reports whether t satisfies every rule of NewContainmentTerms. The zero value is not valid.
func (t ContainmentTerms) valid() bool {
	if !t.mode.Valid() || !t.maxSize.Valid() || t.capacity.slots < 1 || !t.suffocation.Valid() || !t.actionKind.Valid() ||
		t.imposed&^(bit(content.ConditionBlinded)|bit(content.ConditionRestrained)) != 0 ||
		(t.lockedAction != "" && t.lockedAction != strings.TrimSpace(t.lockedAction)) ||
		(t.disgorgeAfterFirstDamage && !t.periodic.present()) ||
		(t.regurgitation.present() && t.mode != ContainmentModeSwallowed) ||
		(t.grapple.present() != (t.mode == ContainmentModeEngulfedGrappled)) {
		return false
	}
	if t.periodic.present() && (!t.periodic.timing.Valid() || (t.periodic.firstTickNextTurn && t.periodic.timing != ContainmentTimingEndOfContainerTurn)) {
		return false
	}
	switch e := t.entry.(type) {
	case EntryAutomatic:
	case EntryBySave:
		if !e.valid() {
			return false
		}
	default:
		return false
	}
	switch d := t.deathRule.(type) {
	case ReleaseOnDeath:
	case EscapeFromCorpse:
		return d.feet >= 1
	default:
		return false
	}
	return true
}

// Mode returns how a creature is held.
func (t ContainmentTerms) Mode() ContainmentMode { return t.mode }

// MaxSize returns the largest creature that can be held.
func (t ContainmentTerms) MaxSize() content.CreatureSize { return t.maxSize }

// Capacity returns how many creatures fit.
func (t ContainmentTerms) Capacity() ContainmentCapacity { return t.capacity }

// Entry returns how a creature comes to be held.
func (t ContainmentTerms) Entry() EntryTerms { return t.entry }

// Imposed returns a new slice of the conditions a held creature gets, in the order of content.ConditionKind.
func (t ContainmentTerms) Imposed() []content.ConditionKind { return kindsOf(t.imposed) }

// TotalCover says whether a held creature has Total Cover against what is outside.
func (t ContainmentTerms) TotalCover() bool { return t.totalCover }

// Suffocation returns the suffocation rule.
func (t ContainmentTerms) Suffocation() SuffocationRule { return t.suffocation }

// NoVerbalSpells says a held creature cannot cast a spell with a Verbal component.
func (t ContainmentTerms) NoVerbalSpells() bool { return t.noVerbalSpells }

// NoTeleport says a held creature cannot teleport.
func (t ContainmentTerms) NoTeleport() bool { return t.noTeleport }

// Periodic returns the periodic damage and whether there is any.
func (t ContainmentTerms) Periodic() (PeriodicDamage, bool) { return t.periodic, t.periodic.present() }

// Regurgitation returns the regurgitation rule and whether there is one.
func (t ContainmentTerms) Regurgitation() (Regurgitation, bool) {
	return t.regurgitation, t.regurgitation.present()
}

// DeathRule returns what happens to the held creatures when the container dies.
func (t ContainmentTerms) DeathRule() ContainerDeathRule { return t.deathRule }

// Exit returns the ways out and whether there are any.
func (t ContainmentTerms) Exit() (ExitTerms, bool) { return t.exit, t.exit.present() }

// Grapple returns the grapple of an engulfed creature and whether there is one.
func (t ContainmentTerms) Grapple() (GrappleHoldTerms, bool) { return t.grapple, t.grapple.present() }

// DisgorgeAfterFirstDamage says whether a creature that survives its first periodic damage is released.
func (t ContainmentTerms) DisgorgeAfterFirstDamage() bool { return t.disgorgeAfterFirstDamage }

// LockedAction returns the action the container cannot take while it holds a creature; empty for none.
func (t ContainmentTerms) LockedAction() string { return t.lockedAction }

// ActionKind returns what the Swallow or Engulf costs the monster.
func (t ContainmentTerms) ActionKind() ContainmentActionKind { return t.actionKind }
