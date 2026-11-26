package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// plannedCreature is one creature a cast would make, with the numbers its link and spec need.
type plannedCreature struct {
	id      CombatantID
	kind    SummonKind
	block   SummonedStatBlock
	monster content.Monster
	hasMon  bool
	mount   MountFacts
	undead  UndeadForm
	fam     FamiliarState
	friends []CombatantID
	grant   SharedResistanceGrant
}

// summonPlan is what a request comes to: the creatures to make, the creatures to replace, or the refusal.
type summonPlan struct {
	news      []plannedCreature
	replace   []CombatantID
	rejection SummonRejection
}

func refuse(r SummonRejection) summonPlan { return summonPlan{rejection: r} }

// spellLevel returns the level of the spell of kind k, the least a cast may use (FR-035).
func spellLevel(k SummonKind) int {
	if !k.Valid() {
		return 0
	}
	return [...]int{5, 4, 5, 2, 3, 6, 1, 7}[k-1]
}

// monsterByID returns the monster with the id among those the host looked up.
func monsterByID(monsters []content.Monster, id content.MonsterID) (content.Monster, bool) {
	i := slices.IndexFunc(monsters, func(m content.Monster) bool { return m.ID() == id })
	if i < 0 {
		return content.Monster{}, false
	}
	return monsters[i], true
}

// idChecks refuses ids that name a creature of the roster (FR-008) and ids that would close a cycle of summoners (plan P10), in
// that order.
func (r SummonedRoster) idChecks(summoner CombatantID, ids []CombatantID) SummonRejection {
	for _, id := range ids {
		if r.indexOf(id) >= 0 {
			return SummonCreatureIDInUseRejection{ID: id}
		}
	}
	for _, id := range ids {
		if r.createsCycle(id, summoner) {
			return SummonLinkCycleRejection{Creature: id, Summoner: summoner}
		}
	}
	return nil
}

// MonsterIDsNeeded returns the ids of the monsters a request needs a stat block for, without repeats, in the order they are first
// needed: the Skeleton or Zombie of Animate Dead, the forms of Create Undead and the form of a familiar (FR-070). The expression
// kinds need none. It returns nil for a request that is not valid.
func MonsterIDsNeeded(request SummonRequest) []content.MonsterID {
	var ids []content.MonsterID
	add := func(id content.MonsterID) {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	switch v := request.(type) {
	case AnimateDeadRequest:
		for _, t := range v.Targets {
			if t.Kind == RemainsBones {
				add("skeleton")
			} else {
				add("zombie")
			}
		}
	case CreateUndeadRequest:
		for _, t := range v.Corpses {
			add(t.Becomes.monster())
		}
	case FindFamiliarRequest:
		add(v.Form.Monster())
	}
	return ids
}

// Summon makes the creatures a cast asks for and returns the transition (FR-001 to FR-008, FR-029 to FR-064). monsters are the
// stat blocks MonsterIDsNeeded names, looked up by the host; a request that needs one that is not given leaves the roster
// unchanged (SummonedUnchanged). The receiver is the roster of c, which the command holds for its checks. Several creatures made
// by one cast are joined in creation order. A cast that replaces a steed or a simulacrum departs the old one first; Find Familiar
// with a familiar present changes its form (FR-044). It returns SummonedRejected with the first reason of Table R (FR-068) and
// the roster unchanged, and never panics. It is pure and rolls no die.
func (r SummonedRoster) Summon(c SummonCreatureCommand, monsters ...content.Monster) SummonedTransition {
	if c.IsZero() {
		return SummonedUnchanged{}
	}
	if req, ok := c.request.(ReassertControlRequest); ok {
		return r.Reassert(c.summoner, req)
	}
	var plan summonPlan
	switch v := c.request.(type) {
	case AnimateObjectsRequest:
		plan = r.planObjects(c, v)
	case GiantInsectRequest:
		plan = r.planInsect(c, v)
	case SummonDragonRequest:
		plan = r.planDragon(c, v)
	case FindSteedRequest:
		plan = r.planSteed(c, v)
	case AnimateDeadRequest:
		plan = r.planAnimateDead(c, v, monsters)
	case CreateUndeadRequest:
		plan = r.planCreateUndead(c, v, monsters)
	case FindFamiliarRequest:
		return r.summonFamiliar(c, v, monsters)
	case SimulacrumRequest:
		plan = r.planSimulacrum(c, v)
	default:
		return SummonedUnchanged{}
	}
	if plan.rejection != nil {
		return rejected(plan.rejection)
	}
	if len(plan.news) == 0 {
		return SummonedUnchanged{}
	}
	if r.countOf(c.summoner)-len(plan.replace)+len(plan.news) > maxSummonsPerSummoner {
		return rejected(SummonLimitExceededRejection{})
	}
	return r.finishSummon(c, plan)
}

// planCheck runs the checks every cast starts with: the ids and the least level.
func (r SummonedRoster) planCheck(c SummonCreatureCommand, ids []CombatantID) SummonRejection {
	if rej := r.idChecks(c.summoner, ids); rej != nil {
		return rej
	}
	if l := spellLevel(kindOf(c.request)); levelOf(c.request) < l {
		return LevelBelowSpellRejection{Minimum: l}
	}
	return nil
}

func (r SummonedRoster) planObjects(c SummonCreatureCommand, q AnimateObjectsRequest) summonPlan {
	if rej := r.planCheck(c, newIDs(q)); rej != nil {
		return refuse(rej)
	}
	weight := 0
	for _, o := range q.Objects {
		if o.Magical || o.WornOrCarried || o.FixedToSurface || o.Size == content.SizeGargantuan {
			return refuse(ObjectNotEligibleRejection{Creature: o.Creature})
		}
		weight += objectWeight(o.Size)
	}
	if weight > max(c.facts.modifier, 0) {
		return refuse(TooManyCreaturesRejection{Maximum: max(c.facts.modifier, 0)})
	}
	if len(q.Objects) == 0 {
		return refuse(NothingToAnimateRejection{})
	}
	var plan summonPlan
	for _, o := range q.Objects {
		block, rej, err := NewAnimatedObjectStatBlock(o.Size, q.Level, c.facts)
		if rej != nil || err != nil {
			return refuse(firstRejection(rej))
		}
		plan.news = append(plan.news, plannedCreature{id: o.Creature, kind: SummonAnimatedObject, block: block})
	}
	return plan
}

// objectWeight is what an object of size counts against the summoner's spellcasting ability modifier: 1 for Medium or smaller, 2
// for Large, 3 for Huge (SRD 5.2.1, "Animate Objects", p. 108).
func objectWeight(size content.CreatureSize) int {
	switch {
	case smallOrLess(size):
		return 1
	case size == content.SizeLarge:
		return 2
	}
	return 3
}

// firstRejection returns rej, or the refusal for a stat block that could not be built for a reason the checks before it should
// have caught, which cannot happen for a command from NewSummonCreatureCommand.
func firstRejection(rej SummonRejection) SummonRejection {
	if rej == nil {
		return FormNotEligibleRejection{}
	}
	return rej
}

func (r SummonedRoster) planInsect(c SummonCreatureCommand, q GiantInsectRequest) summonPlan {
	if rej := r.planCheck(c, newIDs(q)); rej != nil {
		return refuse(rej)
	}
	block, rej, err := NewGiantInsectStatBlock(q.Form, q.Level, c.facts)
	if rej != nil || err != nil {
		return refuse(firstRejection(rej))
	}
	return summonPlan{news: []plannedCreature{{id: q.Creature, kind: SummonGiantInsect, block: block}}}
}

func (r SummonedRoster) planDragon(c SummonCreatureCommand, q SummonDragonRequest) summonPlan {
	if rej := r.planCheck(c, newIDs(q)); rej != nil {
		return refuse(rej)
	}
	block, rej, err := NewDraconicSpiritStatBlock(q.Level, c.facts, q.SharedResistance, q.BreathDamageType)
	if rej != nil || err != nil {
		return refuse(firstRejection(rej))
	}
	grant := SharedResistanceGrant{Summoner: c.summoner, Spell: SummonDraconicSpirit.Spell(), Damage: q.SharedResistance}
	return summonPlan{news: []plannedCreature{{id: q.Creature, kind: SummonDraconicSpirit, block: block, grant: grant}}}
}

func (r SummonedRoster) planSteed(c SummonCreatureCommand, q FindSteedRequest) summonPlan {
	id := CombatantID(steedCreature(q.Choice))
	old, hasOld := r.steedOf(c.summoner)
	gone := slices.IndexFunc(r.gone, func(g summonedGone) bool { return g.summoner == c.summoner && g.kind == SummonOtherworldlySteed })
	if rej := r.planCheck(c, []CombatantID{id}); rej != nil {
		return refuse(rej)
	}
	switch q.Choice.(type) {
	case SteedChoiceSame:
		if hasOld || gone < 0 || r.gone[gone].creature != id {
			return refuse(SteedNotDisappearedRejection{})
		}
	case SteedChoiceDifferent:
		if gone >= 0 && r.gone[gone].creature == id {
			return refuse(SummonCreatureIDInUseRejection{ID: id})
		}
	}
	block, rej, err := NewOtherworldlySteedStatBlock(q.Type, q.Level, c.facts)
	if rej != nil || err != nil {
		return refuse(firstRejection(rej))
	}
	if !c.scene.SpaceAvailable {
		return refuse(SpaceUnavailableRejection{})
	}
	mount, err := NewMountFacts(MountWillingnessWilling, q.Anatomy, MountTrainingTrainedToAcceptRider, q.Habitat, q.Saddle)
	if err != nil {
		return refuse(FormNotEligibleRejection{})
	}
	plan := summonPlan{news: []plannedCreature{{id: id, kind: SummonOtherworldlySteed, block: block, mount: mount}}}
	if hasOld {
		plan.replace = []CombatantID{old.creature}
	}
	return plan
}

// steedOf returns the summoner's steed link.
func (r SummonedRoster) steedOf(summoner CombatantID) (SummonedLink, bool) {
	i := slices.IndexFunc(r.links, func(l SummonedLink) bool { return l.summoner == summoner && l.kind == SummonOtherworldlySteed })
	if i < 0 {
		return SummonedLink{}, false
	}
	return r.links[i], true
}

// simulacrumOf returns the summoner's simulacrum link.
func (r SummonedRoster) simulacrumOf(summoner CombatantID) (SummonedLink, bool) {
	i := slices.IndexFunc(r.links, func(l SummonedLink) bool { return l.summoner == summoner && l.kind == SummonSimulacrum })
	if i < 0 {
		return SummonedLink{}, false
	}
	return r.links[i], true
}

func (r SummonedRoster) planSimulacrum(c SummonCreatureCommand, q SimulacrumRequest) summonPlan {
	if rej := r.planCheck(c, newIDs(q)); rej != nil {
		return refuse(rej)
	}
	block, rej, err := NewSimulacrumStatBlock(q.Original)
	if rej != nil || err != nil {
		return refuse(firstRejection(rej))
	}
	plan := summonPlan{news: []plannedCreature{{id: q.Creature, kind: SummonSimulacrum, block: block, friends: slices.Clone(q.Friends)}}}
	if old, ok := r.simulacrumOf(c.summoner); ok {
		plan.replace = []CombatantID{old.creature}
	}
	return plan
}

// animateDeadMax is how many creatures one cast of Animate Dead raises at level (Table U): 1 plus 2 for each level above 3.
func animateDeadMax(level int) int { return 1 + 2*(level-3) }

// reassertDeadMax is how many creatures one cast of Animate Dead reasserts control of at level: 4 plus 2 for each level above 3.
func reassertDeadMax(level int) int { return 4 + 2*(level-3) }

// undeadRow is a row of the compositions Create Undead allows (Table U): Ghouls, Ghasts or Wights, or Mummies.
type undeadRow int

const (
	rowGhoul undeadRow = iota + 1
	rowGhastOrWight
	rowMummy
)

func (f UndeadForm) row() undeadRow {
	switch f {
	case UndeadGhast, UndeadWight:
		return rowGhastOrWight
	case UndeadMummy:
		return rowMummy
	case UndeadGhoul:
		return rowGhoul
	}
	return rowGhoul
}

// createUndeadMax is how many creatures of the row one cast of Create Undead raises, or has control reasserted, at level
// (Table U); 0 for a row the level does not allow.
func createUndeadMax(level int, row undeadRow) int {
	switch {
	case row == rowGhoul && level >= 6:
		return level - 3
	case row == rowGhastOrWight && level >= 8:
		return level - 6
	case row == rowMummy && level >= 9:
		return 2
	}
	return 0
}

// eligibleCorpse reports whether the remains are a corpse of a Medium or Small Humanoid.
func eligibleCorpse(size content.CreatureSize, humanoid bool) bool {
	return humanoid && (size == content.SizeMedium || size == content.SizeSmall)
}

// firstDuplicateRemains returns the index of the first creature whose remains an earlier creature of the cast already uses, or -1.
func firstDuplicateRemains(remains []RemainsID) int {
	seen := map[RemainsID]struct{}{}
	for i, rm := range remains {
		if _, dup := seen[rm]; dup {
			return i
		}
		seen[rm] = struct{}{}
	}
	return -1
}

func (r SummonedRoster) planAnimateDead(c SummonCreatureCommand, q AnimateDeadRequest, monsters []content.Monster) summonPlan {
	if rej := r.planCheck(c, newIDs(q)); rej != nil {
		return refuse(rej)
	}
	var remains []RemainsID
	for _, t := range q.Targets {
		if t.Kind == RemainsCorpse && !eligibleCorpse(t.Size, t.Humanoid) {
			return refuse(TargetNotEligibleRejection{Creature: t.Creature})
		}
		remains = append(remains, t.Remains)
	}
	if i := firstDuplicateRemains(remains); i >= 0 {
		return refuse(DuplicateTargetRejection{Creature: newIDs(q)[i]})
	}
	if len(q.Targets) > animateDeadMax(q.Level) {
		return refuse(TooManyCreaturesRejection{Maximum: animateDeadMax(q.Level)})
	}
	if len(q.Targets) == 0 {
		return refuse(NothingToAnimateRejection{})
	}
	var plan summonPlan
	for _, t := range q.Targets {
		id := content.MonsterID("skeleton")
		if t.Kind == RemainsCorpse {
			id = "zombie"
		}
		m, ok := monsterByID(monsters, id)
		if !ok {
			return summonPlan{}
		}
		plan.news = append(plan.news, plannedCreature{id: t.Creature, kind: SummonAnimatedUndead, block: statBlockOfMonster(m, m.Type()), monster: m, hasMon: true})
	}
	return plan
}

func (r SummonedRoster) planCreateUndead(c SummonCreatureCommand, q CreateUndeadRequest, monsters []content.Monster) summonPlan {
	if rej := r.planCheck(c, newIDs(q)); rej != nil {
		return refuse(rej)
	}
	if !c.scene.IsNight {
		return refuse(NotNightRejection{})
	}
	var remains []RemainsID
	var forms []UndeadForm
	for _, t := range q.Corpses {
		if !eligibleCorpse(t.Size, t.Humanoid) {
			return refuse(TargetNotEligibleRejection{Creature: t.Creature})
		}
		remains = append(remains, t.Remains)
		forms = append(forms, t.Becomes)
	}
	if i := firstDuplicateRemains(remains); i >= 0 {
		return refuse(DuplicateTargetRejection{Creature: newIDs(q)[i]})
	}
	if rej := compositionCheck(q.Level, forms); rej != nil {
		return refuse(rej)
	}
	if len(q.Corpses) == 0 {
		return refuse(NothingToAnimateRejection{})
	}
	var plan summonPlan
	for _, t := range q.Corpses {
		m, ok := monsterByID(monsters, t.Becomes.monster())
		if !ok {
			return summonPlan{}
		}
		plan.news = append(plan.news, plannedCreature{id: t.Creature, kind: SummonCreatedUndead, block: statBlockOfMonster(m, m.Type()),
			monster: m, hasMon: true, undead: t.Becomes})
	}
	return plan
}

// compositionCheck returns the refusal for creatures of the forms that Create Undead cannot raise or hold at level (FR-039,
// FR-041): TooManyCreatures when there are more than the largest row allows, then CompositionNotAllowed for a mixture of rows or a
// row the level does not allow, then TooManyCreatures for more than the row allows. It returns nil for no forms.
func compositionCheck(level int, forms []UndeadForm) SummonRejection {
	if len(forms) == 0 {
		return nil
	}
	most := max(createUndeadMax(level, rowGhoul), createUndeadMax(level, rowGhastOrWight), createUndeadMax(level, rowMummy))
	if len(forms) > most {
		return TooManyCreaturesRejection{Maximum: most}
	}
	row := forms[0].row()
	for _, f := range forms {
		if f.row() != row {
			return CompositionNotAllowedRejection{}
		}
	}
	rowMax := createUndeadMax(level, row)
	switch {
	case rowMax == 0:
		return CompositionNotAllowedRejection{}
	case len(forms) > rowMax:
		return TooManyCreaturesRejection{Maximum: rowMax}
	}
	return nil
}

// Reassert takes control of the named undead creatures again for 24 hours, without animating a new creature (FR-041, FR-042). It
// returns SummonedRejected with LevelBelowSpellRejection, NothingToReassertRejection when none of the named creatures is an
// undead creature of the summoner and kind, TargetNotEligibleRejection for a named creature that is not, TooManyCreaturesRejection
// above the cap of Table U, CompositionNotAllowedRejection when Create Undead creatures do not fit one row and
// ControlLapsedRejection for a creature whose window ran out (0 left; 1 minute left is accepted), in that order. It never panics.
func (r SummonedRoster) Reassert(summoner CombatantID, q ReassertControlRequest) SummonedTransition {
	if !q.Kind.undead() || summoner.IsZero() {
		return SummonedUnchanged{}
	}
	if l := spellLevel(q.Kind); q.Level < l {
		return rejected(LevelBelowSpellRejection{Minimum: l})
	}
	linked := func(id CombatantID) (SummonedLink, bool) {
		l, ok := r.Link(id)
		return l, ok && l.summoner == summoner && l.kind == q.Kind
	}
	if !slices.ContainsFunc(q.Creatures, func(id CombatantID) bool { _, ok := linked(id); return ok }) {
		return rejected(NothingToReassertRejection{})
	}
	var forms []UndeadForm
	for _, id := range q.Creatures {
		l, ok := linked(id)
		if !ok {
			return rejected(TargetNotEligibleRejection{Creature: id})
		}
		forms = append(forms, l.undead)
	}
	if q.Kind == SummonAnimatedUndead {
		if len(q.Creatures) > reassertDeadMax(q.Level) {
			return rejected(TooManyCreaturesRejection{Maximum: reassertDeadMax(q.Level)})
		}
	} else if rej := compositionCheck(q.Level, forms); rej != nil {
		return rejected(rej)
	}
	for _, id := range q.Creatures {
		if l, _ := linked(id); l.window.Lapsed() {
			return rejected(ControlLapsedRejection{Creature: id})
		}
	}
	d := r.draft("")
	for _, id := range q.Creatures {
		i := d.indexOf(id)
		d.links[i].window = startedWindow()
	}
	d.events = append(d.events, SummonControlReasserted{summonEventBase: summonEventBase{append([]CombatantID{summoner}, q.Creatures...)}, kind: q.Kind})
	return d.applied()
}

// familiarFormIDs are the eleven forms Find Familiar names (SRD 5.2.1, p. 130).
func familiarFormIDs() []content.MonsterID {
	return []content.MonsterID{"bat", "cat", "frog", "hawk", "lizard", "octopus", "owl", "rat", "raven", "spider", "weasel"}
}

// specialFamiliarFormIDs are the seven special forms of Pact of the Chain (SRD 5.2.1, p. 74).
func specialFamiliarFormIDs() []content.MonsterID {
	return []content.MonsterID{"imp", "pseudodragon", "quasit", "skeleton", "sphinx_of_wonder", "sprite", "venomous_snake"}
}

// familiarFormEligible reports whether form is an eligible familiar form for origin: a normal form is one of the eleven or a
// Beast of Challenge Rating 0, a special form is one of the seven and only for Pact of the Chain (FR-043, FR-054).
func familiarFormEligible(origin FamiliarOrigin, form character.FamiliarForm, m content.Monster) bool {
	if form.IsNormal() {
		return slices.Contains(familiarFormIDs(), form.Monster()) || (m.Type() == content.CreatureTypeBeast && m.ChallengeRating().IsZero())
	}
	return origin == FamiliarOriginPactOfTheChain && slices.Contains(specialFamiliarFormIDs(), form.Monster())
}

// familiarTypeFor returns the creature type of a familiar: Fey for Wild Companion, the printed type of a special form, the
// summoner's choice for a normal form (FR-043, FR-053).
func familiarTypeFor(q FindFamiliarRequest, m content.Monster) content.CreatureType {
	switch {
	case q.Origin == FamiliarOriginWildCompanion:
		return content.CreatureTypeFey
	case !q.Form.IsNormal():
		return m.Type()
	}
	return q.Type
}

// summonFamiliar handles Find Familiar: a new familiar, or the form change of the one the summoner has (FR-044).
func (r SummonedRoster) summonFamiliar(c SummonCreatureCommand, q FindFamiliarRequest, monsters []content.Monster) SummonedTransition {
	old, has := r.familiarOf(c.summoner)
	if !has {
		if rej := r.planCheck(c, []CombatantID{q.Creature}); rej != nil {
			return rejected(rej)
		}
	}
	m, found := monsterByID(monsters, q.Form.Monster())
	if !found {
		return SummonedUnchanged{}
	}
	if !familiarFormEligible(q.Origin, q.Form, m) {
		return rejected(FormNotEligibleRejection{})
	}
	state := FamiliarState{form: q.Form, typ: familiarTypeFor(q, m), origin: q.Origin}
	block := statBlockOfMonster(m, state.typ)
	if has {
		state.sight = old.familiar.sight
		d := r.draft(c.scene.Active)
		i := d.indexOf(old.creature)
		d.links[i].familiar, d.links[i].monster, d.links[i].level = state, block.monster, q.Level
		d.events = append(d.events, FamiliarFormChanged{summonEventBase: pair(old.creature, c.summoner), form: q.Form, typ: state.typ, origin: q.Origin, maximum: block.hitPointMaximum})
		a := d.applied()
		a.creatures = []SummonedCreatureSpec{{creature: old.creature, kind: SummonFamiliar, side: old.side, block: block, monster: m, hasMonst: true, placement: OwnCount}}
		return a
	}
	if r.countOf(c.summoner)+1 > maxSummonsPerSummoner {
		return rejected(SummonLimitExceededRejection{})
	}
	return r.finishSummon(c, summonPlan{news: []plannedCreature{{id: q.Creature, kind: SummonFamiliar, block: block, monster: m, hasMon: true, fam: state}}})
}

// finishSummon departs the creatures the plan replaces, then links, joins and reports each new creature in creation order.
func (r SummonedRoster) finishSummon(c SummonCreatureCommand, plan summonPlan) SummonedTransition {
	d := r.draft(c.scene.Active)
	for _, id := range plan.replace {
		d.depart(id, SummonEndReplaced, departure{})
	}
	own := 0
	var specs []SummonedCreatureSpec
	for _, n := range plan.news {
		link := SummonedLink{creature: n.id, summoner: c.summoner, monster: n.block.monster, kind: n.kind, level: levelOf(c.request), side: c.side,
			presence: presenceInEncounter, undead: n.undead, friends: n.friends, familiar: n.fam}
		if n.kind.undead() {
			link.window = startedWindow()
		}
		var pos RosterPosition = RosterAfterCombatant{Anchor: c.summoner}
		if n.kind.Placement() == OwnCount && c.scene.InCombat {
			pos = c.scene.Initiatives[own]
			own++
			link.retained = pos
		}
		d.forget(c.summoner, n.kind)
		if n.grant.Damage != 0 {
			d.endGrant(c.summoner, n.grant.Spell)
			d.grants = append(d.grants, n.grant)
		}
		d.links = append(d.links, link)
		d.events = append(d.events, SummonedCreatureJoined{summonEventBase: pair(n.id, c.summoner), kind: n.kind, side: c.side, level: link.level,
			placement: n.kind.Placement(), grant: n.grant})
		if c.scene.InCombat {
			d.actions = append(d.actions, SummonedRosterJoin{Creature: n.id, Position: pos, Reason: JoinReasonSummoned})
		}
		specs = append(specs, SummonedCreatureSpec{creature: n.id, kind: n.kind, side: c.side, block: n.block, mount: n.mount, monster: n.monster,
			hasMonst: n.hasMon, placement: n.kind.Placement()})
	}
	a := d.applied()
	a.creatures = specs
	return a
}

// endGrant ends the summoner's shared Resistance of the spell, reporting SharedResistanceEnded, when there is one.
func (d *summonedDraft) endGrant(summoner CombatantID, spell content.SpellID) {
	i := slices.IndexFunc(d.grants, func(g SharedResistanceGrant) bool { return g.Summoner == summoner && g.Spell == spell })
	if i < 0 {
		return
	}
	g := d.grants[i]
	d.grants = slices.Delete(d.grants, i, i+1)
	d.events = append(d.events, SharedResistanceEnded{summonEventBase: summonEventBase{[]CombatantID{summoner}}, grant: g})
}
