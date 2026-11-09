package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// The pact invocations (SRD 5.2.1, "Eldritch Invocation Options", pp. 73-74).
const (
	pactOfTheBlade content.FeatureOptionID = "pact_of_the_blade"
	pactOfTheChain content.FeatureOptionID = "pact_of_the_chain"
	pactOfTheTome  content.FeatureOptionID = "pact_of_the_tome"
)

// PactWeaponBenefit is what a pact weapon bond gives the Warlock while it lasts, a closed set (class-warlock.md FR-084).
type PactWeaponBenefit int

const (
	// PactWeaponProficiency: proficiency with the pact weapon.
	PactWeaponProficiency PactWeaponBenefit = iota + 1
	// PactWeaponSpellcastingFocus: the pact weapon works as a Spellcasting Focus.
	PactWeaponSpellcastingFocus
)

// PactWeaponBond is the weapon bonded by Pact of the Blade (SRD 5.2.1, "Pact of the Blade", p. 74). The zero value is no bond. It is
// immutable and comparable with ==.
type PactWeaponBond struct {
	item     content.ItemID
	conjured bool
}

// Bonded reports whether a weapon is bonded.
func (b PactWeaponBond) Bonded() bool { return b.item != "" }

// Weapon returns the bonded weapon, empty for no bond.
func (b PactWeaponBond) Weapon() content.ItemID { return b.item }

// Conjured reports whether the weapon was conjured by the feature (it disappears when the bond ends) and not a magic weapon.
func (b PactWeaponBond) Conjured() bool { return b.conjured }

// Benefits returns both benefits while the bond lasts and none without one, as a new slice (class-warlock.md FR-084).
func (b PactWeaponBond) Benefits() []PactWeaponBenefit {
	if !b.Bonded() {
		return []PactWeaponBenefit{}
	}
	return []PactWeaponBenefit{PactWeaponProficiency, PactWeaponSpellcastingFocus}
}

// FamiliarForm is the form of the Pact of the Chain familiar: one of the seven special forms or a normal form of Find Familiar (class-warlock.md
// FR-090). The zero value is no form. It is immutable and comparable with ==. Build it with SpecialFamiliarForm or NormalFamiliarForm.
type FamiliarForm struct {
	id     content.MonsterID
	normal bool
}

// SpecialFamiliarForm returns the special form with the monster id. Whether it is one of the seven is checked by SelectFamiliarForm.
func SpecialFamiliarForm(id content.MonsterID) FamiliarForm { return FamiliarForm{id: id} }

// NormalFamiliarForm returns the normal form with the monster id. Whether it is a Beast of Challenge Rating 0 is checked by SelectFamiliarForm.
func NormalFamiliarForm(id content.MonsterID) FamiliarForm { return FamiliarForm{id: id, normal: true} }

// Monster returns the monster id of the form, empty for the zero form.
func (f FamiliarForm) Monster() content.MonsterID { return f.id }

// IsNormal reports whether the form is a normal form of Find Familiar.
func (f FamiliarForm) IsNormal() bool { return f.normal }

// FamiliarSpeedKind is the Speed Investment of the Chain Master gives the familiar, a closed set (class-warlock.md FR-092). The zero value
// means none chosen.
type FamiliarSpeedKind int

const (
	// FamiliarFlies gives a Fly Speed of 40 feet.
	FamiliarFlies FamiliarSpeedKind = iota + 1
	// FamiliarSwims gives a Swim Speed of 40 feet.
	FamiliarSwims
)

// FamiliarPact is the Pact of the Chain familiar as the character knows it: its form and the Speed chosen. The zero value is no familiar.
// It is immutable and comparable with ==; combat reads it.
type FamiliarPact struct {
	form  FamiliarForm
	speed FamiliarSpeedKind
}

// Form returns the form and false when the Warlock has no familiar form chosen.
func (f FamiliarPact) Form() (FamiliarForm, bool) { return f.form, f.form.id != "" }

// Speed returns the Speed Investment of the Chain Master gave and false for none.
func (f FamiliarPact) Speed() (FamiliarSpeedKind, bool) { return f.speed, f.speed != 0 }

// BookOfShadows is the book conjured by Pact of the Tome: three cantrips, two level 1 spells and, when the Warlock knew Gift of the
// Protectors, the page with the names written on it (SRD 5.2.1, "Pact of the Tome", p. 74). The zero value is no book. It holds slices,
// so compare two with Equal, not ==.
type BookOfShadows struct {
	cantrips, spells []content.SpellID
	page             bool
	names            []Identifier
}

// Conjured reports whether a book exists.
func (b BookOfShadows) Conjured() bool { return len(b.cantrips) > 0 }

// Cantrips returns the cantrips in the book as a new slice.
func (b BookOfShadows) Cantrips() []content.SpellID { return slices.Clone(b.cantrips) }

// Spells returns the level 1 spells in the book as a new slice.
func (b BookOfShadows) Spells() []content.SpellID { return slices.Clone(b.spells) }

// HasPage reports whether the book has the Gift of the Protectors page.
func (b BookOfShadows) HasPage() bool { return b.page }

// Names returns the creatures whose names are on the page as a new slice.
func (b BookOfShadows) Names() []Identifier { return slices.Clone(b.names) }

func (b BookOfShadows) equal(o BookOfShadows) bool {
	return b.page == o.page && slices.Equal(b.cantrips, o.cantrips) && slices.Equal(b.spells, o.spells) && slices.Equal(b.names, o.names)
}

// GazeLinkStatus is the state of a Gaze of Two Minds link, a closed set. The zero value is no link.
type GazeLinkStatus int

const (
	// GazeLinkActive: the link lasts until the end of the Warlock's next turn.
	GazeLinkActive GazeLinkStatus = iota + 1
	// GazeLinkLapsed: the Warlock did not maintain the link on the turn, so it has ended.
	GazeLinkLapsed
)

// SensorLink is the Gaze of Two Minds link with a creature (class-warlock.md FR-069 to FR-072). The zero value is no link. It is immutable
// and comparable with ==. Build it with GazeLinkOf or let UseGazeOfTwoMinds return it.
type SensorLink struct {
	creature Identifier
	status   GazeLinkStatus
}

// GazeLinkOf returns the link with the creature in the status, as the host stored it between turns. It returns the zero link for a zero
// creature or an invalid status.
func GazeLinkOf(creature Identifier, status GazeLinkStatus) SensorLink {
	if creature.IsZero() || status < GazeLinkActive || status > GazeLinkLapsed {
		return SensorLink{}
	}
	return SensorLink{creature: creature, status: status}
}

// Creature returns the linked creature, the zero Identifier for no link.
func (l SensorLink) Creature() Identifier { return l.creature }

// Status returns the status, 0 for no link.
func (l SensorLink) Status() GazeLinkStatus { return l.status }

// Active reports whether the link is active.
func (l SensorLink) Active() bool { return l.status == GazeLinkActive }

const gazeSpellReachFeet = 60

// CanCastFromLinkedSpace reports whether the Warlock can cast spells as if in the creature's space: the link is active and the two are
// within 60 feet of each other (class-warlock.md FR-072).
func (l SensorLink) CanCastFromLinkedSpace(distanceFeet int) bool {
	return l.Active() && distanceFeet <= gazeSpellReachFeet
}

// LinkedSenses returns the special senses the linked creature has while the link is active and none otherwise, as a new slice
// (class-warlock.md FR-151).
func (l SensorLink) LinkedSenses(creatureSenses []content.SenseEntry) []content.SenseEntry {
	if !l.Active() {
		return []content.SenseEntry{}
	}
	return slices.Clone(creatureSenses)
}

// WarlockPactState is the feature state of the Warlock's pact: the Pact of the Blade bond, the Pact of the Chain familiar, the Book of
// Shadows and the Gaze of Two Minds link (class-warlock.md section 5.2). It is stored under WarlockPactStateKey, apart from the WarlockState,
// because it holds lists. The zero value is a Warlock with no bond, familiar, book or link. It is immutable and holds slices, so compare two
// with Equal, not ==. Safe for concurrent use.
type WarlockPactState struct {
	bond     PactWeaponBond
	familiar FamiliarPact
	book     BookOfShadows
	link     SensorLink
}

// WarlockPactStateKey returns the FeatureStateKey named warlock_pact.
func WarlockPactStateKey() FeatureStateKey[WarlockPactState] {
	return MustFeatureStateKey[WarlockPactState]("warlock_pact")
}

// WarlockPactStateParts are the fields of a WarlockPactState for RestoreWarlockPactState. Construct it with a keyed literal.
type WarlockPactStateParts struct {
	// Bond is the bonded weapon, empty for none; BondConjured says it was conjured.
	Bond         content.ItemID
	BondConjured bool
	// Familiar is the form, empty for none; FamiliarNormal says it is a normal form; FamiliarSpeed is the Speed chosen, 0 for none.
	Familiar       content.MonsterID
	FamiliarNormal bool
	FamiliarSpeed  FamiliarSpeedKind
	// BookCantrips and BookSpells are the contents of the book, both empty for no book; BookPage says it has the page; PageNames are on it.
	BookCantrips, BookSpells []content.SpellID
	BookPage                 bool
	PageNames                []Identifier
	// Link is the creature linked and LinkStatus the status, 0 for no link.
	Link       Identifier
	LinkStatus GazeLinkStatus
}

// RestoreWarlockPactState validates parts and returns the state. It returns an error matching srderr.ErrInvalidInput for an invalid id, a
// book without exactly three cantrips and two level 1 spells (or with one but not the other), names on a book with no page, a duplicate
// name, a speed or link status outside the sets, or a link status without a creature. It never panics.
func RestoreWarlockPactState(p WarlockPactStateParts) (WarlockPactState, error) {
	hasBook := len(p.BookCantrips) > 0 || len(p.BookSpells) > 0
	bad := (p.Bond != "" && !p.Bond.Valid()) || (p.Bond == "" && p.BondConjured) ||
		(p.Familiar != "" && !p.Familiar.Valid()) || (p.Familiar == "" && (p.FamiliarNormal || p.FamiliarSpeed != 0)) ||
		p.FamiliarSpeed < 0 || p.FamiliarSpeed > FamiliarSwims ||
		(hasBook && (len(p.BookCantrips) != bookCantripCount || len(p.BookSpells) != bookSpellCount)) ||
		(!hasBook && (p.BookPage || len(p.PageNames) > 0)) || (!p.BookPage && len(p.PageNames) > 0) ||
		slices.ContainsFunc(slices.Concat(p.BookCantrips, p.BookSpells), func(s content.SpellID) bool { return !s.Valid() }) ||
		slices.ContainsFunc(p.PageNames, Identifier.IsZero) || duplicated(p.PageNames) ||
		(p.Link.IsZero() != (p.LinkStatus == 0)) || p.LinkStatus < 0 || p.LinkStatus > GazeLinkLapsed
	if bad {
		return WarlockPactState{}, invalidInput("Warlock pact state outside its ranges")
	}
	return WarlockPactState{
		bond:     PactWeaponBond{item: p.Bond, conjured: p.BondConjured},
		familiar: FamiliarPact{form: FamiliarForm{id: p.Familiar, normal: p.FamiliarNormal}, speed: p.FamiliarSpeed},
		book:     BookOfShadows{cantrips: slices.Clone(p.BookCantrips), spells: slices.Clone(p.BookSpells), page: p.BookPage, names: slices.Clone(p.PageNames)},
		link:     SensorLink{creature: p.Link, status: p.LinkStatus},
	}, nil
}

func duplicated(names []Identifier) bool {
	seen := map[Identifier]bool{}
	for _, n := range names {
		if seen[n] {
			return true
		}
		seen[n] = true
	}
	return false
}

// Parts returns the fields of s with new slices.
func (s WarlockPactState) Parts() WarlockPactStateParts {
	return WarlockPactStateParts{Bond: s.bond.item, BondConjured: s.bond.conjured, Familiar: s.familiar.form.id, FamiliarNormal: s.familiar.form.normal,
		FamiliarSpeed: s.familiar.speed, BookCantrips: s.book.Cantrips(), BookSpells: s.book.Spells(), BookPage: s.book.page, PageNames: s.book.Names(),
		Link: s.link.creature, LinkStatus: s.link.status}
}

// Bond returns the Pact of the Blade bond.
func (s WarlockPactState) Bond() PactWeaponBond { return s.bond }

// Familiar returns the Pact of the Chain familiar.
func (s WarlockPactState) Familiar() FamiliarPact { return s.familiar }

// Book returns the Book of Shadows.
func (s WarlockPactState) Book() BookOfShadows { return s.book }

// Link returns the Gaze of Two Minds link.
func (s WarlockPactState) Link() SensorLink { return s.link }

// Equal reports whether other is a WarlockPactState with the same fields.
func (s WarlockPactState) Equal(other FeatureState) bool {
	o, ok := other.(WarlockPactState)
	return ok && s.bond == o.bond && s.familiar == o.familiar && s.link == o.link && s.book.equal(o.book)
}

// WarlockPactTransition is the closed outcome of a Warlock domain function that changes the WarlockPactState. Variants:
// WarlockPactChanged, WarlockPactRefused.
type WarlockPactTransition interface{ isWarlockPactTransition() }

// WarlockPactChanged is the outcome of an operation the rules allowed: the new pact state, the events and, for the operations that have
// them, the attacks left or the Resistance granted.
type WarlockPactChanged struct {
	state      WarlockPactState
	events     []Event
	attacks    int
	resistance content.DamageType
}

// State returns the new WarlockPactState.
func (r WarlockPactChanged) State() WarlockPactState { return r.state }

// Events returns the events, as a new slice.
func (r WarlockPactChanged) Events() []Event { return slices.Clone(r.events) }

// AttacksRemaining returns the attacks left after ForgoAttackForFamiliar, 0 for the other operations.
func (r WarlockPactChanged) AttacksRemaining() int { return r.attacks }

// Resistance returns the damage type GrantFamiliarResistance gives the familiar Resistance to and false for the other operations.
func (r WarlockPactChanged) Resistance() (content.DamageType, bool) {
	return r.resistance, r.resistance != 0
}

// WarlockPactRefused is the outcome of an operation a rule refused: the state is unchanged and there are no events.
type WarlockPactRefused struct{ rejection WarlockRejection }

// Rejection returns why the operation was refused.
func (r WarlockPactRefused) Rejection() WarlockRejection { return r.rejection }

func (WarlockPactChanged) isWarlockPactTransition() {}
func (WarlockPactRefused) isWarlockPactTransition() {}

func pactChanged(s WarlockPactState, events ...Event) WarlockPactChanged {
	return WarlockPactChanged{state: s, events: events}
}
func pactRefused(r WarlockRejection) WarlockPactTransition { return WarlockPactRefused{rejection: r} }
func pactNotKnown(option content.FeatureOptionID) WarlockPactTransition {
	return pactRefused(WarlockRejection{Kind: WarlockInvocationNotKnown, Option: option})
}
