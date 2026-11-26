package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ContainmentPhase says whether a held creature is inside a living container or trapped in the corpse of a
// dead one (SRD 5.2.1, "Tarrasque", p. 332). The zero value is inside. Comparable and safe for concurrent
// use.
type ContainmentPhase struct{ trappedFeet int }

// ContainmentInside returns the phase of a creature inside a living container.
func ContainmentInside() ContainmentPhase { return ContainmentPhase{} }

// NewContainmentTrapped returns the phase of a creature that must spend feet of movement to climb out of a
// dead container. It returns an error matching srderr.ErrInvalidInput for fewer than 1 foot. It never
// panics.
func NewContainmentTrapped(feet int) (ContainmentPhase, error) {
	if feet < 1 {
		return ContainmentPhase{}, invalidInput("trapped needs at least 1 foot")
	}
	return ContainmentPhase{feet}, nil
}

// IsTrapped says whether the creature is trapped in a dead container.
func (p ContainmentPhase) IsTrapped() bool { return p.trappedFeet > 0 }

// TrappedFeet returns the movement needed to climb out, 0 for a creature inside a living container.
func (p ContainmentPhase) TrappedFeet() int { return p.trappedFeet }

// ContainmentInput is the raw values of a Containment. Build it with a keyed literal.
type ContainmentInput struct {
	// Container holds Occupant.
	Container CombatantID
	// Occupant is the held creature.
	Occupant CombatantID
	// OccupantSize is the size of the occupant, which costs capacity.
	OccupantSize content.CreatureSize
	// Terms are the container's terms.
	Terms ContainmentTerms
	// Phase is inside or trapped.
	Phase ContainmentPhase
	// AddedConditions are the conditions the containment itself added, among Terms.Imposed.
	AddedConditions []content.ConditionKind
	// PartOrdinal numbers the grapple part of an Engulfed-grappled creature, 1 or more.
	PartOrdinal int
	// DamageFromInside is the damage the occupant dealt to the container this turn, 0 or more.
	DamageFromInside int
	// DamageTicks counts the periodic damage already dealt, 0 or more.
	DamageTicks int
	// TickDeferred skips the next end-of-turn periodic damage.
	TickDeferred bool
}

// Containment is the record of one creature held by another: who holds whom, the container's terms, and
// the counters the rules keep (SRD 5.2.1, "Gelatinous Cube", p. 287). The zero value is invalid. Build it
// with NewContainment; the ids are exported fields because no function of this package returns a
// CombatantID, so construct it with keyed literals only through NewContainment. Comparable with == and
// safe for concurrent use.
type Containment struct {
	// Container holds Occupant.
	Container CombatantID
	// Occupant is the held creature.
	Occupant         CombatantID
	occupantSize     content.CreatureSize
	terms            ContainmentTerms
	phase            ContainmentPhase
	added            kindMask
	partOrdinal      int
	damageFromInside int
	damageTicks      int
	tickDeferred     bool
}

// NewContainment returns the record. It returns an error matching srderr.ErrInvalidInput for a blank id,
// equal ids, an invalid size or terms, a negative counter, a part ordinal below 1, or added conditions that
// are not among the terms' imposed ones, or a deferred tick the terms do not defer. It copies in and never panics.
func NewContainment(in ContainmentInput) (Containment, error) {
	r := Containment{in.Container, in.Occupant, in.OccupantSize, in.Terms, in.Phase, 0, in.PartOrdinal,
		in.DamageFromInside, in.DamageTicks, in.TickDeferred}
	for _, k := range in.AddedConditions {
		if !k.Valid() {
			return Containment{}, invalidInput("invalid added condition")
		}
		r.added |= bit(k)
	}
	if !r.valid() {
		return Containment{}, invalidInput("invalid containment")
	}
	return r, nil
}

// valid reports whether r satisfies every rule of NewContainment.
func (r Containment) valid() bool {
	return !r.Container.IsZero() && !r.Occupant.IsZero() && r.Container != r.Occupant && r.occupantSize.Valid() &&
		r.terms.valid() && r.phase.trappedFeet >= 0 && r.added&^r.terms.imposed == 0 && r.partOrdinal >= 1 &&
		r.damageFromInside >= 0 && r.damageTicks >= 0 && (!r.tickDeferred || r.terms.periodic.firstTickNextTurn)
}

// Size returns the occupant's size.
func (r Containment) Size() content.CreatureSize { return r.occupantSize }

// Terms returns the container's terms.
func (r Containment) Terms() ContainmentTerms { return r.terms }

// Phase returns inside or trapped.
func (r Containment) Phase() ContainmentPhase { return r.phase }

// AddedConditions returns a new slice of the conditions the containment added, in the order of
// content.ConditionKind.
func (r Containment) AddedConditions() []content.ConditionKind { return kindsOf(r.added) }

// PartOrdinal returns the number of the grapple part.
func (r Containment) PartOrdinal() int { return r.partOrdinal }

// DamageFromInside returns the damage the occupant dealt to the container this turn.
func (r Containment) DamageFromInside() int { return r.damageFromInside }

// DamageTicks returns how many periodic damages were dealt.
func (r Containment) DamageTicks() int { return r.damageTicks }

// TickDeferred says whether the next end-of-turn periodic damage is skipped.
func (r Containment) TickDeferred() bool { return r.tickDeferred }

// ContainmentOfOccupant is the answer of Containments.ContainerOf. Variants: InContainer, NotContained.
type ContainmentOfOccupant interface{ isContainmentOfOccupant() }

// InContainer: the creature is held. Read the record with Record.
type InContainer struct{ record Containment }

// NotContained: the creature is not held.
type NotContained struct{}

func (InContainer) isContainmentOfOccupant()  {}
func (NotContained) isContainmentOfOccupant() {}

// Record returns the containment.
func (c InContainer) Record() Containment { return c.record }

// ContainmentTransition is the closed set of outcomes of a Containments operation. Variants:
// ContainmentTransitionApplied, ContainmentTransitionRejected. Whatever the variant, the receiver is
// unchanged.
type ContainmentTransition interface{ isContainmentTransition() }

// ContainmentTransitionApplied: the operation took effect. It carries the new set and the events.
type ContainmentTransitionApplied struct {
	containments Containments
	events       []Event
}

// Containments returns the set after the operation.
func (t ContainmentTransitionApplied) Containments() Containments { return t.containments }

// Events returns a copy of the events, in the order they happened.
func (t ContainmentTransitionApplied) Events() []Event { return slices.Clone(t.events) }

// ContainmentTransitionRejected: the rules refuse the operation in this state. It carries no event.
// Construct it with a keyed literal.
type ContainmentTransitionRejected struct{ Rejection ContainmentRejection }

func (ContainmentTransitionApplied) isContainmentTransition()  {}
func (ContainmentTransitionRejected) isContainmentTransition() {}

// ContainerTurnBoundary is the moment of a container's turn at which periodic damage can be due.
type ContainerTurnBoundary int

// The two boundaries.
const (
	// ContainerTurnBoundaryStart: the start of the container's turn.
	ContainerTurnBoundaryStart ContainerTurnBoundary = iota + 1
	// ContainerTurnBoundaryEnd: the end of the container's turn.
	ContainerTurnBoundaryEnd
)

// String returns "start", "end" or "unknown".
func (b ContainerTurnBoundary) String() string { return enumName([]string{"start", "end"}, int(b)) }

// Valid reports whether b is one of the two boundaries.
func (b ContainerTurnBoundary) Valid() bool {
	return b >= ContainerTurnBoundaryStart && b <= ContainerTurnBoundaryEnd
}

// Containments is the creatures held by others, in the order they were taken in (SRD 5.2.1, "Gelatinous
// Cube", p. 287). It holds the invariants of the model: a creature is held once, the records of one
// container share their terms and fit its capacity, and no container is itself held. The zero Containments
// is empty and valid. It holds a slice, so compare with Equal, not ==; it copies on construction and in
// every accessor. Safe for concurrent use.
type Containments struct{ list []Containment }

// NewContainments returns the set holding rs, which is also the order they were taken in. It returns an
// error matching srderr.ErrInvalidInput for an invalid record, an occupant listed twice, two records of one
// container with unequal terms, a container whose occupants cost more than its capacity, or a creature
// that is both a container and an occupant (nested containment, which also rejects every cycle). It returns a LimitExceededError for more than 256 records (plan P10; the occupants of one container are bounded by it). The check
// walks the records once with sets, never recursing. It copies rs and never panics.
func NewContainments(rs []Containment) (Containments, error) {
	if err := overCap("containments", len(rs), maxInputList); err != nil {
		return Containments{}, err
	}
	occupants := make(map[CombatantID]struct{}, len(rs))
	terms := make(map[CombatantID]ContainmentTerms, len(rs))
	used := make(map[CombatantID]int, len(rs))
	for _, r := range rs {
		if !r.valid() {
			return Containments{}, invalidInput("invalid containment")
		}
		if _, dup := occupants[r.Occupant]; dup {
			return Containments{}, invalidInput("a creature is held once")
		}
		occupants[r.Occupant] = struct{}{}
		if t, ok := terms[r.Container]; ok && t != r.terms {
			return Containments{}, invalidInput("a container has one set of terms")
		}
		terms[r.Container] = r.terms
		used[r.Container] = satAdd(used[r.Container], r.terms.capacity.Cost(r.occupantSize))
		if used[r.Container] > r.terms.capacity.slots {
			return Containments{}, invalidInput("a container holds no more than its capacity")
		}
	}
	for c := range terms {
		if _, nested := occupants[c]; nested {
			return Containments{}, invalidInput("a held creature cannot hold another")
		}
	}
	return Containments{slices.Clone(rs)}, nil
}

// MustContainments is NewContainments for literals known to be valid. It panics with the error
// NewContainments returns, and only then.
func MustContainments(rs []Containment) Containments {
	s, err := NewContainments(rs)
	if err != nil {
		panic(err)
	}
	return s
}

// Len returns the number of records.
func (s Containments) Len() int { return len(s.list) }

// All returns a copy of the records, in the order the creatures were taken in.
func (s Containments) All() []Containment { return slices.Clone(s.list) }

// Equal reports whether s and o hold the same records, whatever order they were taken in.
func (s Containments) Equal(o Containments) bool {
	if len(s.list) != len(o.list) {
		return false
	}
	for _, r := range s.list {
		if !slices.Contains(o.list, r) {
			return false
		}
	}
	return true
}

// ContainerOf returns the record of the creature, or NotContained.
func (s Containments) ContainerOf(occupant CombatantID) ContainmentOfOccupant {
	for _, r := range s.list {
		if r.Occupant == occupant {
			return InContainer{r}
		}
	}
	return NotContained{}
}

// OccupantsOf returns a new slice of the container's records, in the order they were taken in; empty for a
// container holding none.
func (s Containments) OccupantsOf(container CombatantID) []Containment {
	out := []Containment{}
	for _, r := range s.list {
		if r.Container == container {
			out = append(out, r)
		}
	}
	return out
}

// SlotsUsed returns the capacity the container's occupants take: the sum of their costs.
func (s Containments) SlotsUsed(container CombatantID) int {
	used := 0
	for _, r := range s.list {
		if r.Container == container {
			used = satAdd(used, r.terms.capacity.Cost(r.occupantSize))
		}
	}
	return used
}

// holdsAny says whether the creature holds a record as a container.
func (s Containments) holdsAny(container CombatantID) bool {
	for _, r := range s.list {
		if r.Container == container {
			return true
		}
	}
	return false
}

// check returns the first rejection of taking occupant of size into container with terms, in the order of
// FR-019, or 0 when it can be taken in.
func (s Containments) check(container, occupant CombatantID, size content.CreatureSize, terms ContainmentTerms) ContainmentRejection {
	_, nested := s.ContainerOf(container).(InContainer)
	_, held := s.ContainerOf(occupant).(InContainer)
	switch {
	case nested:
		return ContainmentRejectionContainerIsContained
	case s.holdsAny(occupant):
		return ContainmentRejectionOccupantIsContainer
	case held:
		return ContainmentRejectionOccupantAlreadyContained
	case size > terms.maxSize:
		return ContainmentRejectionTooLarge
	case satAdd(s.SlotsUsed(container), terms.capacity.Cost(size)) > terms.capacity.slots:
		return ContainmentRejectionNoRoom
	}
	return 0
}

// nextPartOrdinal returns the lowest positive ordinal no record of the container uses.
func (s Containments) nextPartOrdinal(container CombatantID) int {
	used := map[int]struct{}{}
	for _, r := range s.list {
		if r.Container == container {
			used[r.partOrdinal] = struct{}{}
		}
	}
	n := 1
	for {
		if _, taken := used[n]; !taken {
			return n
		}
		n++
	}
}

// enter takes occupant into container: ContainmentTransitionApplied with the record in phase inside and one
// OccupantContained, or ContainmentTransitionRejected with the first rejection of check. added are the
// conditions the entry added and partOrdinal the number of its grapple part.
func (s Containments) enter(container, occupant CombatantID, size content.CreatureSize, terms ContainmentTerms,
	added kindMask, partOrdinal int, tickDeferred bool) ContainmentTransition {
	if r := s.check(container, occupant, size, terms); r != 0 {
		return ContainmentTransitionRejected{r}
	}
	rec := Containment{container, occupant, size, terms, ContainmentInside(), added, partOrdinal, 0, 0, tickDeferred}
	return ContainmentTransitionApplied{Containments{append(s.All(), rec)},
		[]Event{OccupantContained{containmentEventBase{occupant, container}, terms.mode}}}
}

// RecordDamageFromInside adds amount to the damage the occupant dealt to its container this turn (SRD
// 5.2.1, "Purple Worm", p. 317: the threshold is per creature inside, never summed). Zero is allowed. It
// returns ContainmentTransitionRejected with ContainmentRejectionNotContained for a creature that is not
// held or is trapped in a dead container, and otherwise ContainmentTransitionApplied with no event. It
// returns an error matching srderr.ErrInvalidInput, checked first, for a negative amount. The sum
// saturates at the limits of int. It never panics.
func (s Containments) RecordDamageFromInside(occupant CombatantID, amount int) (ContainmentTransition, error) {
	if amount < 0 {
		return nil, invalidInput("negative damage")
	}
	in, ok := s.ContainerOf(occupant).(InContainer)
	if !ok || in.record.phase.IsTrapped() {
		return ContainmentTransitionRejected{ContainmentRejectionNotContained}, nil
	}
	r := in.record
	r.damageFromInside = satAdd(r.damageFromInside, amount)
	return ContainmentTransitionApplied{containments: s.replacing(r)}, nil
}

// replacing returns a new set with the record of r.Occupant replaced by r.
func (s Containments) replacing(r Containment) Containments {
	out := s.All()
	for i := range out {
		if out[i].Occupant == r.Occupant {
			out[i] = r
		}
	}
	return Containments{out}
}

// without returns a new set holding every record but the occupant's.
func (s Containments) without(occupant CombatantID) Containments {
	var kept []Containment
	for _, r := range s.list {
		if r.Occupant != occupant {
			kept = append(kept, r)
		}
	}
	return Containments{kept}
}

// withoutTallies returns a new set in which the records of container have no damage from inside.
func (s Containments) withoutTallies(container CombatantID) Containments {
	out := s.All()
	for i := range out {
		if out[i].Container == container {
			out[i].damageFromInside = 0
		}
	}
	return Containments{out}
}

// advance applies the periodic damage of the container's records that is due at boundary: a record inside
// whose timing matches, in the order the creatures were taken in. At the end boundary a deferred record is
// skipped and its flag cleared. Each due increments the record's DamageTicks.
func (s Containments) advance(container CombatantID, boundary ContainerTurnBoundary) (Containments, []ContainmentDamageDue) {
	timing := ContainmentTimingStartOfContainerTurn
	if boundary == ContainerTurnBoundaryEnd {
		timing = ContainmentTimingEndOfContainerTurn
	}
	out := s.All()
	var dues []ContainmentDamageDue
	for i, r := range out {
		p, ok := r.terms.Periodic()
		if r.Container != container || r.phase.IsTrapped() || !ok || p.timing != timing {
			continue
		}
		if r.tickDeferred { // only an end-of-turn record is ever deferred
			out[i].tickDeferred = false
			continue
		}
		out[i].damageTicks = satAdd(r.damageTicks, 1)
		dues = append(dues, ContainmentDamageDue{Container: container, Target: r.Occupant, terms: p.damage, cause: ContainmentDamageCausePeriodic})
	}
	return Containments{out}, dues
}
