package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// SummonRequest is what a cast asks to bring or keep: the closed set of requests SummonCreature accepts (summoned-and-companion-
// creatures.md section 6). A request holds the cast level and the ids of the creatures to create, which the host supplies (the
// module never generates ids). Variants: AnimateObjectsRequest, GiantInsectRequest, SummonDragonRequest, FindSteedRequest,
// AnimateDeadRequest, CreateUndeadRequest, FindFamiliarRequest, SimulacrumRequest, ReassertControlRequest.
type SummonRequest interface{ isSummonRequest() }

// ObjectTarget is one object Animate Objects would animate (FR-031). Build it with a keyed literal. Creature is the id of the
// creature it becomes.
type ObjectTarget struct {
	Creature                               CombatantID
	Size                                   content.CreatureSize
	Magical, WornOrCarried, FixedToSurface bool
}

// AnimateObjectsRequest is a cast of Animate Objects (SRD 5.2.1, p. 108). Build it with a keyed literal.
type AnimateObjectsRequest struct {
	Level   int
	Objects []ObjectTarget
}

// GiantInsectRequest is a cast of Giant Insect (SRD 5.2.1, p. 135). Build it with a keyed literal.
type GiantInsectRequest struct {
	Creature CombatantID
	Level    int
	Form     InsectForm
}

// SummonDragonRequest is a cast of Summon Dragon (SRD 5.2.1, p. 166): the Resistance the summoner shares and the type of the
// Breath Weapon, two separate choices (Q-18). Build it with a keyed literal.
type SummonDragonRequest struct {
	Creature         CombatantID
	Level            int
	SharedResistance content.DamageType
	BreathDamageType content.DamageType
}

// SteedChoice says which steed a cast of Find Steed brings (FR-056). Variants: SteedChoiceDifferent, SteedChoiceSame.
type SteedChoice interface{ isSteedChoice() }

// SteedChoiceDifferent brings a new steed with the new id Creature. Build it with a keyed literal.
type SteedChoiceDifferent struct{ Creature CombatantID }

// SteedChoiceSame brings back the steed that disappeared, whose id was Creature. Build it with a keyed literal.
type SteedChoiceSame struct{ Creature CombatantID }

func (SteedChoiceDifferent) isSteedChoice() {}
func (SteedChoiceSame) isSteedChoice()      {}

// steedCreature returns the id a steed choice names, as text since no function of the package returns a CombatantID.
func steedCreature(c SteedChoice) string {
	switch v := c.(type) {
	case SteedChoiceDifferent:
		return string(v.Creature)
	case SteedChoiceSame:
		return string(v.Creature)
	}
	return ""
}

// FindSteedRequest is a cast of Find Steed (SRD 5.2.1, p. 130). Anatomy, Habitat and Saddle are the host's report on the steed
// as a mount (FR-018). Build it with a keyed literal.
type FindSteedRequest struct {
	Level   int
	Type    SteedType
	Choice  SteedChoice
	Anatomy MountAnatomy
	Habitat MountHabitat
	Saddle  ExoticSaddle
}

// RemainsID is the host's id of one pile of bones or corpse, so that no two creatures of a cast come from the same one (FR-038).
// The empty id is invalid.
type RemainsID string

// RemainsKind is what an undead creature is raised from. The zero value is invalid.
type RemainsKind int

// The two kinds.
const (
	// RemainsBones: a pile of bones, which becomes a Skeleton.
	RemainsBones RemainsKind = iota + 1
	// RemainsCorpse: the corpse of a Medium or Small Humanoid, which becomes a Zombie.
	RemainsCorpse
)

// Valid reports whether k is one of the two kinds.
func (k RemainsKind) Valid() bool { return k == RemainsBones || k == RemainsCorpse }

// UndeadTarget is one pile of bones or corpse Animate Dead would raise (FR-038). Build it with a keyed literal.
type UndeadTarget struct {
	Creature CombatantID
	Remains  RemainsID
	Kind     RemainsKind
	Size     content.CreatureSize
	Humanoid bool
}

// AnimateDeadRequest is a cast of Animate Dead (SRD 5.2.1, p. 108). Build it with a keyed literal.
type AnimateDeadRequest struct {
	Level   int
	Targets []UndeadTarget
}

// UndeadForm is the undead creature Create Undead makes of a corpse (FR-039). The zero value is invalid.
type UndeadForm int

// The four forms.
const (
	// UndeadGhoul: a Ghoul.
	UndeadGhoul UndeadForm = iota + 1
	// UndeadGhast: a Ghast.
	UndeadGhast
	// UndeadWight: a Wight.
	UndeadWight
	// UndeadMummy: a Mummy.
	UndeadMummy
)

// Valid reports whether f is one of the four forms.
func (f UndeadForm) Valid() bool { return f >= UndeadGhoul && f <= UndeadMummy }

// String returns "ghoul", "ghast", "wight", "mummy" or "unknown".
func (f UndeadForm) String() string {
	return enumName([]string{"ghoul", "ghast", "wight", "mummy"}, int(f))
}

// monster returns the id of the monster of the form.
func (f UndeadForm) monster() content.MonsterID { return content.MonsterID(f.String()) }

// CorpseTarget is one corpse Create Undead would raise as Becomes (FR-039). Build it with a keyed literal.
type CorpseTarget struct {
	Creature CombatantID
	Remains  RemainsID
	Size     content.CreatureSize
	Humanoid bool
	Becomes  UndeadForm
}

// CreateUndeadRequest is a cast of Create Undead (SRD 5.2.1, p. 121). Build it with a keyed literal.
type CreateUndeadRequest struct {
	Level   int
	Corpses []CorpseTarget
}

// FindFamiliarRequest is a cast of Find Familiar, or the use of Pact of the Chain or Wild Companion that casts it (SRD 5.2.1,
// p. 130; FR-043 to FR-054). Form is the form chosen. Type is the creature type chosen in place of Beast, one of Celestial, Fey or
// Fiend; it is ignored for a Wild Companion familiar, which is Fey, and for a special form, which keeps its printed type. Build it
// with a keyed literal.
type FindFamiliarRequest struct {
	Creature CombatantID
	Level    int
	Form     character.FamiliarForm
	Type     content.CreatureType
	Origin   FamiliarOrigin
}

// SimulacrumRequest is a cast of Simulacrum (SRD 5.2.1, p. 162): the original, and the creatures the simulacrum is friendly to
// besides its summoner (the player's choice, FR-064). Build it with a keyed literal.
type SimulacrumRequest struct {
	Creature CombatantID
	Level    int
	Original SimulacrumOriginal
	Friends  []CombatantID
}

// ReassertControlRequest asks to take control of the named creatures again instead of animating a new one (SRD 5.2.1, "Animate
// Dead", p. 108; "Create Undead", p. 121; FR-041). Kind is SummonAnimatedUndead or SummonCreatedUndead. Build it with a keyed
// literal.
type ReassertControlRequest struct {
	Kind      SummonKind
	Level     int
	Creatures []CombatantID
}

func (AnimateObjectsRequest) isSummonRequest()  {}
func (GiantInsectRequest) isSummonRequest()     {}
func (SummonDragonRequest) isSummonRequest()    {}
func (FindSteedRequest) isSummonRequest()       {}
func (AnimateDeadRequest) isSummonRequest()     {}
func (CreateUndeadRequest) isSummonRequest()    {}
func (FindFamiliarRequest) isSummonRequest()    {}
func (SimulacrumRequest) isSummonRequest()      {}
func (ReassertControlRequest) isSummonRequest() {}

// SummonScene is what the host reports about the place and time of a cast (FR-003, FR-005, FR-056, FR-077). Build it with a keyed
// literal. Active is the combatant the host's tracker has active, the zero value when none. SpaceAvailable is whether an
// unoccupied space within range exists for a steed. Initiatives holds one RosterAtInitiative the host built from RollInitiative
// for each creature that rolls its own, in order, when InCombat, and none otherwise.
type SummonScene struct {
	InCombat, IsNight, SpaceAvailable bool
	Active                            CombatantID
	Initiatives                       []RosterAtInitiative
}

// SummonCreatureCommand is the immutable input of SummonCreature.Run. Build it with NewSummonCreatureCommand; the zero value is
// invalid.
type SummonCreatureCommand struct {
	roster   SummonedRoster
	summoner CombatantID
	side     Side
	request  SummonRequest
	facts    SummonerSpellFacts
	scene    SummonScene
}

// IsZero reports whether c is the zero command.
func (c SummonCreatureCommand) IsZero() bool { return c.request == nil }

// kindOf returns the kind of creature a request brings or keeps.
func kindOf(r SummonRequest) SummonKind {
	switch v := r.(type) {
	case AnimateObjectsRequest:
		return SummonAnimatedObject
	case GiantInsectRequest:
		return SummonGiantInsect
	case SummonDragonRequest:
		return SummonDraconicSpirit
	case FindSteedRequest:
		return SummonOtherworldlySteed
	case AnimateDeadRequest:
		return SummonAnimatedUndead
	case CreateUndeadRequest:
		return SummonCreatedUndead
	case FindFamiliarRequest:
		return SummonFamiliar
	case SimulacrumRequest:
		return SummonSimulacrum
	case ReassertControlRequest:
		return v.Kind
	}
	return 0
}

// levelOf returns the cast level of a request.
func levelOf(r SummonRequest) int {
	switch v := r.(type) {
	case AnimateObjectsRequest:
		return v.Level
	case GiantInsectRequest:
		return v.Level
	case SummonDragonRequest:
		return v.Level
	case FindSteedRequest:
		return v.Level
	case AnimateDeadRequest:
		return v.Level
	case CreateUndeadRequest:
		return v.Level
	case FindFamiliarRequest:
		return v.Level
	case SimulacrumRequest:
		return v.Level
	case ReassertControlRequest:
		return v.Level
	}
	return 0
}

// newIDs returns the ids of the creatures a request would create, in order; a reassert and a request of an unknown variant create none.
func newIDs(r SummonRequest) []CombatantID {
	var ids []CombatantID
	switch v := r.(type) {
	case AnimateObjectsRequest:
		for _, o := range v.Objects {
			ids = append(ids, o.Creature)
		}
	case GiantInsectRequest:
		ids = append(ids, v.Creature)
	case SummonDragonRequest:
		ids = append(ids, v.Creature)
	case FindSteedRequest:
		ids = append(ids, CombatantID(steedCreature(v.Choice)))
	case AnimateDeadRequest:
		for _, t := range v.Targets {
			ids = append(ids, t.Creature)
		}
	case CreateUndeadRequest:
		for _, t := range v.Corpses {
			ids = append(ids, t.Creature)
		}
	case FindFamiliarRequest:
		ids = append(ids, v.Creature)
	case SimulacrumRequest:
		ids = append(ids, v.Creature)
	}
	return ids
}

// cloneRequest returns r with its slices copied.
func cloneRequest(r SummonRequest) SummonRequest {
	switch v := r.(type) {
	case AnimateObjectsRequest:
		v.Objects = slices.Clone(v.Objects)
		return v
	case AnimateDeadRequest:
		v.Targets = slices.Clone(v.Targets)
		return v
	case CreateUndeadRequest:
		v.Corpses = slices.Clone(v.Corpses)
		return v
	case SimulacrumRequest:
		v.Friends = slices.Clone(v.Friends)
		return v
	case ReassertControlRequest:
		v.Creatures = slices.Clone(v.Creatures)
		return v
	}
	return r
}

// validRequest reports whether the request holds well-formed values: a level of 1 to 9, defined enums and non-blank ids.
// Empty target lists are valid here: they are the refusals NothingToAnimate and NothingToReassert (FR-042).
func validRequest(r SummonRequest) bool {
	l := levelOf(r)
	inRange := l >= minCastLevel && l <= maxCastLevel
	switch v := r.(type) {
	case AnimateObjectsRequest:
		return inRange && !slices.ContainsFunc(v.Objects, func(o ObjectTarget) bool { return !o.Size.Valid() })
	case GiantInsectRequest:
		return inRange && v.Form.Valid()
	case SummonDragonRequest:
		return inRange && v.SharedResistance.Valid() && v.BreathDamageType.Valid()
	case FindSteedRequest:
		_, err := NewMountFacts(MountWillingnessWilling, v.Anatomy, MountTrainingTrainedToAcceptRider, v.Habitat, v.Saddle)
		return inRange && v.Type.Valid() && err == nil && steedCreature(v.Choice) != ""
	case AnimateDeadRequest:
		return inRange && !slices.ContainsFunc(v.Targets, func(t UndeadTarget) bool { return t.Remains == "" || !t.Kind.Valid() || !t.Size.Valid() })
	case CreateUndeadRequest:
		return inRange && !slices.ContainsFunc(v.Corpses, func(t CorpseTarget) bool { return t.Remains == "" || !t.Becomes.Valid() || !t.Size.Valid() })
	case FindFamiliarRequest:
		return inRange && v.Form.Monster().Valid() && v.Origin.Valid() &&
			(v.Origin == FamiliarOriginWildCompanion || !v.Form.IsNormal() || slices.Contains(familiarTypeChoices(), v.Type))
	case SimulacrumRequest:
		o := v.Original
		return inRange && o.Monster.Valid() && o.Size.Valid() && o.Type.Valid() && o.HitPointMaximum >= 1 && !slices.Contains(v.Friends, CombatantID(""))
	case ReassertControlRequest:
		return inRange && v.Kind.undead() && !slices.Contains(v.Creatures, CombatantID(""))
	}
	return false
}

// familiarTypeChoices are the creature types the summoner of a familiar chooses from in place of Beast (SRD 5.2.1, "Find
// Familiar", p. 130).
func familiarTypeChoices() []content.CreatureType {
	return []content.CreatureType{content.CreatureTypeCelestial, content.CreatureTypeFey, content.CreatureTypeFiend}
}

// distinct reports whether ids holds no id twice.
func distinct(ids []CombatantID) bool {
	seen := make(map[CombatantID]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

// rollsOwnInitiative returns how many creatures of a cast roll their own Initiative: one for each undead creature a cast raises
// and one for a familiar the summoner does not have yet (FR-003).
func rollsOwnInitiative(r SummonedRoster, summoner CombatantID, req SummonRequest) int {
	switch v := req.(type) {
	case AnimateDeadRequest:
		return len(v.Targets)
	case CreateUndeadRequest:
		return len(v.Corpses)
	case FindFamiliarRequest:
		if _, has := r.familiarOf(summoner); !has {
			return 1
		}
	}
	return 0
}

// NewSummonCreatureCommand returns the command: summoner casts request on side, with the facts of its spell and the scene. It
// returns an error matching srderr.ErrInvalidInput, and never panics, for a zero or unknown request, a level outside 1 to 9, a
// blank id, a creature that is its own summoner or an id used twice in the command, an enum value that is not defined, a
// Wild Companion or special-form combination the request cannot hold (a normal form without a Celestial, Fey or Fiend choice),
// zero facts for a spell that reads them (Animate Objects, Giant Insect, Summon Dragon, Find Steed), a side that is not defined,
// and a scene whose Initiatives are present when the summoner is not in combat or whose count differs from the creatures that roll
// their own Initiative when it is (FR-003, FR-067). An empty target list is not an error: SummonCreature refuses it (FR-042).
func NewSummonCreatureCommand(roster SummonedRoster, summoner CombatantID, side Side, request SummonRequest, facts SummonerSpellFacts, scene SummonScene) (SummonCreatureCommand, error) {
	bad := invalidInput("a summons needs a summoner, a side, a valid request, the facts of the spell and a scene that matches it")
	if summoner.IsZero() || !side.Valid() || request == nil || !validRequest(request) {
		return SummonCreatureCommand{}, bad
	}
	ids := newIDs(request)
	if r, ok := request.(ReassertControlRequest); ok {
		ids = r.Creatures
	}
	if r, ok := request.(SimulacrumRequest); ok {
		ids = slices.Concat(ids, r.Friends)
	}
	if slices.Contains(ids, CombatantID("")) || slices.Contains(ids, summoner) || !distinct(ids) {
		return SummonCreatureCommand{}, bad
	}
	if k := kindOf(request); (k == SummonAnimatedObject || k == SummonGiantInsect || k == SummonDraconicSpirit || k == SummonOtherworldlySteed) && facts.IsZero() {
		return SummonCreatureCommand{}, bad
	}
	want := 0
	if scene.InCombat {
		want = rollsOwnInitiative(roster, summoner, request)
	}
	if len(scene.Initiatives) != want || slices.ContainsFunc(scene.Initiatives, func(p RosterAtInitiative) bool { return !p.Tie.Valid() }) {
		return SummonCreatureCommand{}, bad
	}
	scene.Initiatives = slices.Clone(scene.Initiatives)
	return SummonCreatureCommand{roster, summoner, side, cloneRequest(request), facts, scene}, nil
}

// MustSummonCreatureCommand is NewSummonCreatureCommand for values known to be valid. It panics with the error
// NewSummonCreatureCommand returns, and only then.
func MustSummonCreatureCommand(roster SummonedRoster, summoner CombatantID, side Side, request SummonRequest, facts SummonerSpellFacts, scene SummonScene) SummonCreatureCommand {
	c, err := NewSummonCreatureCommand(roster, summoner, side, request, facts, scene)
	if err != nil {
		panic(err)
	}
	return c
}
