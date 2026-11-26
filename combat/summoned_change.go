package combat

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
)

// The numbers the rules for companions fix (SRD 5.2.1, "Find Familiar", p. 130; "Otherworldly Steed", p. 131; "Simulacrum",
// p. 163), and the bound the host's amounts are checked against so the arithmetic cannot overflow at 32-bit width (plan P10).
const (
	familiarRangeFeet    = 100
	lifeBondRangeFeet    = 5
	repairRangeFeet      = 5
	repairGoldPerHitPt   = 100
	maxSummonedAmount    = 1_000_000
	maxSummonedDistance  = 1_000_000_000
	maxSummonedLevelBond = 9
)

// SummonedChange is one report or order a host gives to a SummonedRoster through ApplySummonedChange: the closed set of the
// changes the rules for summoned and companion creatures know (summoned-and-companion-creatures.md FR-072). Each variant is a
// value built with a keyed literal and checked by NewApplySummonedChangeCommand. Variants: IssueCommand, ReportZeroHitPoints,
// ReportSpellEnded, ReportSummonerDied, ReportLongRestFinished, ElapseControlTime, DismissFamiliar, RecallFamiliar,
// ShareFamiliarSenses, DeliverTouch, RepairSimulacrum, ApplySteedLifeBond, ReportTurnStarted, ReportTurnEnded, ReportTaskComplete,
// RequestFamiliarAction.
type SummonedChange interface{ isSummonedChange() }

// CommandTarget is one creature of a group order and its distance from the summoner in feet.
type CommandTarget struct {
	Creature     CombatantID
	DistanceFeet int
}

// IssueCommand gives Order to creatures of Kind that Summoner made (FR-009 to FR-017). One Bonus Action commands any number of the
// creatures of the spell, all receiving the same order, and a group order is atomic. CanGiveVerbalCommand is the host's report for
// the verbal channel. Kind is not SummonOtherworldlySteed.
type IssueCommand struct {
	Summoner             CombatantID
	Kind                 SummonKind
	Targets              []CommandTarget
	Order                SummonedOrder
	CanGiveVerbalCommand bool
}

// ReportZeroHitPoints reports that Creature dropped to 0 Hit Points (FR-019 to FR-025). Damage and Taken are the resolutions and
// the DamageTaken event of the damage that did it, which an Animated Object's overflow is read from (FR-021); a creature of
// another kind ignores them. The zero DamageTaken and no Damage mean no overflow.
type ReportZeroHitPoints struct {
	Creature CombatantID
	Damage   []DamageResolution
	Taken    DamageTaken
}

// SummonSpellEnd is how a spell that made creatures ended (FR-022). The zero value is invalid.
type SummonSpellEnd int

// The three causes.
const (
	// SummonSpellEndConcentrationEnded: the caster's Concentration ended (a ConcentrationEnded event).
	SummonSpellEndConcentrationEnded SummonSpellEnd = iota + 1
	// SummonSpellEndDurationElapsed: the duration ran out.
	SummonSpellEndDurationElapsed
	// SummonSpellEndDispelled: the spell was dispelled.
	SummonSpellEndDispelled
)

// Valid reports whether c is one of the three causes.
func (c SummonSpellEnd) Valid() bool {
	return c >= SummonSpellEndConcentrationEnded && c <= SummonSpellEndDispelled
}

// String returns "concentration_ended", "duration_elapsed", "dispelled" or "unknown".
func (c SummonSpellEnd) String() string {
	return enumName([]string{"concentration_ended", "duration_elapsed", "dispelled"}, int(c))
}

// ReportSpellEnded reports that Summoner's Spell ended (FR-022): every present creature of the spell that lasts for it departs, and
// a shared Resistance of the spell ends. A Simulacrum ends on SummonSpellEndDispelled only; spells with an Instantaneous duration end
// nothing.
type ReportSpellEnded struct {
	Summoner CombatantID
	Spell    content.SpellID
	Cause    SummonSpellEnd
}

// ReportSummonerDied reports that Summoner died (FR-023): a steed departs; no other kind does.
type ReportSummonerDied struct{ Summoner CombatantID }

// ReportLongRestFinished reports that Summoner finished a Long Rest (FR-053): a Wild Companion familiar departs. A Short Rest is
// not reported, since it ends nothing.
type ReportLongRestFinished struct{ Summoner CombatantID }

// ElapseControlTime reports that Elapsed time passed (FR-040): every undead creature's control window shrinks, and a window that
// reaches 0 lapses.
type ElapseControlTime struct{ Elapsed time.Duration }

// Dismissal is how a summoner dismisses a familiar (FR-050, FR-052). The zero value is invalid.
type Dismissal int

// The two ways.
const (
	// DismissalTemporary: the familiar goes to its pocket dimension and can be recalled.
	DismissalTemporary Dismissal = iota + 1
	// DismissalForever: the link ends and the summoner may cast for a new familiar.
	DismissalForever
)

// Valid reports whether d is one of the two ways.
func (d Dismissal) Valid() bool { return d == DismissalTemporary || d == DismissalForever }

// DismissFamiliar dismisses Summoner's familiar (FR-050, FR-052), costing the Magic action the host takes (FR-055).
type DismissFamiliar struct {
	Summoner CombatantID
	How      Dismissal
}

// RecallFamiliar recalls Summoner's familiar from its pocket dimension into an unoccupied space within 30 feet the host reports
// (FR-051). When InCombat the host gives an Initiative for a familiar that has none retained, built from RollInitiative; otherwise
// Initiative is the zero value.
type RecallFamiliar struct {
	Summoner       CombatantID
	SpaceAvailable bool
	InCombat       bool
	Initiative     RosterAtInitiative
}

// ShareFamiliarSenses lets Summoner see through the eyes and hear through the ears of its familiar, DistanceFeet away, until the
// start of Summoner's next turn (FR-046).
type ShareFamiliarSenses struct {
	Summoner     CombatantID
	DistanceFeet int
}

// DeliverTouch has Summoner's familiar deliver a spell with range Touch (FR-047). SpellIsTouch is whether the spell has that range
// and Familiar whether the familiar can take a Reaction.
type DeliverTouch struct {
	Summoner     CombatantID
	SpellIsTouch bool
	DistanceFeet int
	Familiar     ConditionActionAvailability
}

// RepairSimulacrum restores Restore of the Missing Hit Points of Summoner's simulacrum Creature, which is DistanceFeet away, after a
// Long Rest, for 100 GP of components for each Hit Point (FR-062). Missing and Restore are at most 1,000,000.
type RepairSimulacrum struct {
	Summoner, Creature CombatantID
	Missing, Restore   int
	DistanceFeet       int
	LongRestTaken      bool
}

// ApplySteedLifeBond works out what Life Bond gives Summoner's Steed when the summoner regains Regained Hit Points from a spell of
// SpellLevel (0 for a cantrip or a source that is not a spell) while the steed is DistanceFeet away (FR-057).
type ApplySteedLifeBond struct {
	Summoner, Steed CombatantID
	Regained        int
	SpellLevel      int
	DistanceFeet    int
}

// ReportTurnStarted reports that the turn of Combatant started: a sight link of its familiar ends (FR-046).
type ReportTurnStarted struct{ Combatant CombatantID }

// ReportTurnEnded reports that the turn of Creature ended: its order is consumed, except an undead creature's (FR-013).
type ReportTurnEnded struct{ Creature CombatantID }

// ReportTaskComplete reports that an undead creature finished the task it was ordered to do, so its standing order ends (FR-013).
type ReportTaskComplete struct{ Creature CombatantID }

// RequestFamiliarAction asks whether a familiar may take Action (FR-048). It may not take the Attack action without a Grant; other
// actions are allowed.
type RequestFamiliarAction struct {
	Creature CombatantID
	Action   StandardAction
	Grant    FamiliarAttackGrant
}

func (IssueCommand) isSummonedChange()           {}
func (ReportZeroHitPoints) isSummonedChange()    {}
func (ReportSpellEnded) isSummonedChange()       {}
func (ReportSummonerDied) isSummonedChange()     {}
func (ReportLongRestFinished) isSummonedChange() {}
func (ElapseControlTime) isSummonedChange()      {}
func (DismissFamiliar) isSummonedChange()        {}
func (RecallFamiliar) isSummonedChange()         {}
func (ShareFamiliarSenses) isSummonedChange()    {}
func (DeliverTouch) isSummonedChange()           {}
func (RepairSimulacrum) isSummonedChange()       {}
func (ApplySteedLifeBond) isSummonedChange()     {}
func (ReportTurnStarted) isSummonedChange()      {}
func (ReportTurnEnded) isSummonedChange()        {}
func (ReportTaskComplete) isSummonedChange()     {}
func (RequestFamiliarAction) isSummonedChange()  {}

// validDistance reports whether d is a distance within the bound.
func validDistance(d int) bool { return d >= 0 && d <= maxSummonedDistance }

// validChange reports whether the change holds well-formed values (FR-067).
func validChange(c SummonedChange) bool {
	switch v := c.(type) {
	case IssueCommand:
		ids := make([]CombatantID, 0, len(v.Targets))
		for _, t := range v.Targets {
			if !validDistance(t.DistanceFeet) {
				return false
			}
			ids = append(ids, t.Creature)
		}
		return !v.Summoner.IsZero() && v.Kind.commandable() && len(ids) > 0 && !slices.Contains(ids, CombatantID("")) &&
			!slices.Contains(ids, v.Summoner) && distinct(ids) && validSummonedOrder(v.Order)
	case ReportZeroHitPoints:
		return !v.Creature.IsZero()
	case ReportSpellEnded:
		return !v.Summoner.IsZero() && v.Spell.Valid() && v.Cause.Valid()
	case ReportSummonerDied:
		return !v.Summoner.IsZero()
	case ReportLongRestFinished:
		return !v.Summoner.IsZero()
	case ElapseControlTime:
		return v.Elapsed >= 0
	case DismissFamiliar:
		return !v.Summoner.IsZero() && v.How.Valid()
	case RecallFamiliar:
		if v.InCombat {
			return !v.Summoner.IsZero() && v.Initiative.Tie.Valid()
		}
		return !v.Summoner.IsZero() && v.Initiative == (RosterAtInitiative{})
	case ShareFamiliarSenses:
		return !v.Summoner.IsZero() && validDistance(v.DistanceFeet)
	case DeliverTouch:
		return !v.Summoner.IsZero() && validDistance(v.DistanceFeet) &&
			(v.Familiar == ConditionActionAvailabilityAvailable || v.Familiar == ConditionActionAvailabilityNone)
	case RepairSimulacrum:
		return !v.Summoner.IsZero() && !v.Creature.IsZero() && v.Summoner != v.Creature && validDistance(v.DistanceFeet) &&
			v.Missing >= 0 && v.Missing <= maxSummonedAmount && v.Restore >= 0 && v.Restore <= maxSummonedAmount
	case ApplySteedLifeBond:
		return !v.Summoner.IsZero() && !v.Steed.IsZero() && v.Summoner != v.Steed && validDistance(v.DistanceFeet) &&
			v.Regained >= 0 && v.Regained <= maxSummonedAmount && v.SpellLevel >= 0 && v.SpellLevel <= maxSummonedLevelBond
	case ReportTurnStarted:
		return !v.Combatant.IsZero()
	case ReportTurnEnded:
		return !v.Creature.IsZero()
	case ReportTaskComplete:
		return !v.Creature.IsZero()
	case RequestFamiliarAction:
		return !v.Creature.IsZero() && v.Action.Valid() && (v.Grant == 0 || v.Grant.Valid())
	}
	return false
}

// cloneChange returns c with its slices copied.
func cloneChange(c SummonedChange) SummonedChange {
	switch v := c.(type) {
	case IssueCommand:
		v.Targets = slices.Clone(v.Targets)
		return v
	case ReportZeroHitPoints:
		v.Damage = slices.Clone(v.Damage)
		return v
	}
	return c
}

// OverflowDamage returns the damage that carries over from an Animated Object to the object it reverts to (SRD 5.2.1, "Animate
// Objects", p. 109; FR-021): the post-pipeline amount of the resolutions, Immunity included, less the Hit Points lost and the part
// Temporary Hit Points absorbed, never negative. The arithmetic saturates.
func OverflowDamage(resolutions []DamageResolution, taken DamageTaken) int {
	total := 0
	for _, r := range resolutions {
		total = satAdd(total, r.Dealt())
	}
	return max(0, satAdd(total, -satAdd(taken.HitPointsLost(), taken.AbsorbedByTemporary())))
}

// activeLink returns the link of a present creature of the summoner and kind, and its index.
func (r SummonedRoster) activeLink(creature, summoner CombatantID) (SummonedLink, bool) {
	l, ok := r.Link(creature)
	return l, ok && l.summoner == summoner && l.presence == presenceInEncounter
}

// IssueCommand gives an order to the creatures it names (FR-009 to FR-017). It returns SummonedRejected with
// CommandNotYourCreatureRejection for a creature that is not in the roster, not linked to the summoner, in the pocket dimension
// or of another spell than the command's, then ControlLapsedRejection for an undead creature whose window lapsed, then
// CommandOutOfRangeRejection for a creature beyond the mental channel's range (exactly at the range is accepted), then
// CannotGiveVerbalCommandRejection; a group order is atomic, so one creature refused refuses all and each reason is checked for
// every creature before the next. It never panics.
func (r SummonedRoster) IssueCommand(c IssueCommand) SummonedTransition {
	links := make([]SummonedLink, len(c.Targets))
	for i, t := range c.Targets {
		l, ok := r.activeLink(t.Creature, c.Summoner)
		if !ok || l.kind != c.Kind {
			return rejected(CommandNotYourCreatureRejection{Creature: t.Creature})
		}
		links[i] = l
	}
	for _, l := range links {
		if l.window.Lapsed() {
			return rejected(ControlLapsedRejection{Creature: l.creature})
		}
	}
	switch ch := c.Kind.Channel().(type) {
	case ChannelBonusActionMental:
		for _, t := range c.Targets {
			if t.DistanceFeet > ch.RangeFeet {
				return rejected(CommandOutOfRangeRejection{Creature: t.Creature, RangeFeet: ch.RangeFeet})
			}
		}
	case ChannelVerbalNoAction:
		if !c.CanGiveVerbalCommand {
			return rejected(CannotGiveVerbalCommandRejection{})
		}
	}
	d := r.draft("")
	ids := []CombatantID{c.Summoner}
	for _, t := range c.Targets {
		d.links[d.indexOf(t.Creature)].order = c.Order
		ids = append(ids, t.Creature)
	}
	d.events = append(d.events, SummonedCreatureCommanded{summonEventBase: summonEventBase{ids}, kind: c.Kind, order: c.Order})
	return d.applied()
}

// ReportZeroHitPoints removes the link of a creature that dropped to 0 Hit Points (FR-019 to FR-026). A Giant Insect, Draconic
// Spirit, Otherworldly Steed or familiar departs and a steed or familiar is remembered as gone; an Animated Object departs
// reverting to the object with the overflow of FR-021; a simulacrum departs reverting to snow and makes no Death Saving Throws; an
// undead creature is unlinked and stays as a body that dies under Monster Death, with no roster instruction. active is the
// combatant the host's tracker has active: a departing creature that is active has a pending removal (FR-077). A creature the
// roster does not hold, or one in its pocket dimension, is SummonedUnchanged. It never panics.
func (r SummonedRoster) ReportZeroHitPoints(c ReportZeroHitPoints, active CombatantID) SummonedTransition {
	l, ok := r.Link(c.Creature)
	if !ok || l.presence != presenceInEncounter {
		return SummonedUnchanged{}
	}
	d := r.draft(active)
	how := departure{}
	switch {
	case l.kind == SummonAnimatedObject:
		how.overflow = OverflowDamage(c.Damage, c.Taken)
	case l.kind.undead():
		how.body = true
	}
	d.depart(c.Creature, SummonEndDroppedToZeroHitPoints, how)
	return d.applied()
}

// ReportSpellEnded ends the spell of a summoner (FR-022): every present creature of the summoner and spell whose kind lasts for the
// spell departs with SummonEndSpellEnded, a Simulacrum on SummonSpellEndDispelled only, and the shared Resistance of the spell ends
// whether or not a spirit is still present. Kinds whose spell is Instantaneous are not touched. It returns SummonedUnchanged when
// nothing ended. It never panics.
func (r SummonedRoster) ReportSpellEnded(c ReportSpellEnded, active CombatantID) SummonedTransition {
	d := r.draft(active)
	for _, l := range r.links {
		ends := l.kind.lastsForSpell() || (l.kind == SummonSimulacrum && c.Cause == SummonSpellEndDispelled)
		if l.summoner == c.Summoner && l.kind.Spell() == c.Spell && l.presence == presenceInEncounter && ends {
			d.depart(l.creature, SummonEndSpellEnded, departure{})
		}
	}
	d.endGrant(c.Summoner, c.Spell)
	if len(d.events) == 0 {
		return SummonedUnchanged{}
	}
	return d.applied()
}

// ReportSummonerDied departs the summoner's steed (FR-023); a familiar, simulacrum or undead creature persists and a Concentration
// summons ends through ReportSpellEnded. It returns SummonedUnchanged when there is no steed. It never panics.
func (r SummonedRoster) ReportSummonerDied(c ReportSummonerDied, active CombatantID) SummonedTransition {
	steed, ok := r.steedOf(c.Summoner)
	if !ok || steed.presence != presenceInEncounter {
		return SummonedUnchanged{}
	}
	d := r.draft(active)
	d.depart(steed.creature, SummonEndSummonerDied, departure{})
	return d.applied()
}

// ReportLongRestFinished departs a Wild Companion familiar of the summoner (FR-053), present or in its pocket dimension; a
// familiar of another origin stays. It returns SummonedUnchanged when there is none. It never panics.
func (r SummonedRoster) ReportLongRestFinished(c ReportLongRestFinished, active CombatantID) SummonedTransition {
	f, ok := r.familiarOf(c.Summoner)
	if !ok || f.familiar.origin != FamiliarOriginWildCompanion {
		return SummonedUnchanged{}
	}
	d := r.draft(active)
	d.depart(f.creature, SummonEndLongRestFinished, departure{})
	return d.applied()
}

// ElapseControl takes elapsed off the control window of every undead creature; a window that reaches 0 lapses and reports
// SummonControlLapsed, and the creature stays linked, uncontrolled (FR-040). It returns SummonedUnchanged when no creature has a
// window to shrink. It never panics; a negative duration is not a valid change (NewApplySummonedChangeCommand refuses it) and
// shrinks nothing here.
func (r SummonedRoster) ElapseControl(elapsed time.Duration) SummonedTransition {
	d := r.draft("")
	changed := false
	for i, l := range d.links {
		if l.window.IsZero() || l.window.Lapsed() {
			continue
		}
		changed = true
		d.links[i].window = l.window.elapse(max(elapsed, 0))
		if d.links[i].window.Lapsed() {
			d.events = append(d.events, SummonControlLapsed{summonEventBase: pair(l.creature, l.summoner), kind: l.kind})
		}
	}
	if !changed {
		return SummonedUnchanged{}
	}
	return d.applied()
}

// Dismiss dismisses the summoner's familiar (FR-050, FR-052). Temporarily it moves a present familiar to the pocket dimension,
// keeping its Hit Points, with a SummonedRosterLeave of RemovalCauseDeparted, and returns SummonedRejected with
// FamiliarAlreadyDismissedRejection for one already there. Forever it ends the link from either place. It returns
// SummonedUnchanged when the summoner has no familiar. It never panics.
func (r SummonedRoster) Dismiss(c DismissFamiliar, active CombatantID) SummonedTransition {
	f, ok := r.familiarOf(c.Summoner)
	if !ok {
		return SummonedUnchanged{}
	}
	d := r.draft(active)
	if c.How == DismissalForever {
		d.depart(f.creature, SummonEndDismissedForever, departure{})
		return d.applied()
	}
	if f.presence == presencePocket {
		return rejected(FamiliarAlreadyDismissedRejection{})
	}
	pending := f.creature == active
	d.links[d.indexOf(f.creature)].presence = presencePocket
	d.events = append(d.events, FamiliarDismissedTemporarily{summonEventBase: pair(f.creature, f.summoner), pendingRemoval: pending})
	d.actions = append(d.actions, SummonedRosterLeave{Creature: f.creature, Cause: SummonEndDismissedTemporarily.removal(), AfterTurn: pending})
	return d.applied()
}

// Recall brings the summoner's familiar back from its pocket dimension with the Hit Points it had (FR-051): a
// SummonedRosterJoin at the RosterAtInitiative it had, or the one given when it had none, in combat, and none outside it. It
// returns SummonedRejected with FamiliarNotDismissedRejection for a familiar that is present or no familiar, then
// SpaceUnavailableRejection when the host reports no unoccupied space within 30 feet. It never panics.
func (r SummonedRoster) Recall(c RecallFamiliar) SummonedTransition {
	f, ok := r.familiarOf(c.Summoner)
	if !ok || f.presence != presencePocket {
		return rejected(FamiliarNotDismissedRejection{})
	}
	if !c.SpaceAvailable {
		return rejected(SpaceUnavailableRejection{})
	}
	d := r.draft("")
	i := d.indexOf(f.creature)
	d.links[i].presence = presenceInEncounter
	d.events = append(d.events, FamiliarReappeared{summonEventBase: pair(f.creature, f.summoner)})
	if c.InCombat {
		pos := f.retained
		if pos == nil {
			pos = c.Initiative
			d.links[i].retained = pos
		}
		d.actions = append(d.actions, SummonedRosterJoin{Creature: f.creature, Position: pos, Reason: JoinReasonSummoned})
	}
	return d.applied()
}

// ShareSenses starts the sight link of the summoner with its familiar until the start of the summoner's next turn (FR-046). A
// second link while one is active starts again. It returns SummonedRejected with SensesNotSharableRejection when the familiar is in
// the pocket dimension or gone, or farther than 100 feet (exactly 100 is accepted). It never panics.
func (r SummonedRoster) ShareSenses(c ShareFamiliarSenses) SummonedTransition {
	f, ok := r.familiarOf(c.Summoner)
	if !ok || f.presence != presenceInEncounter || c.DistanceFeet > familiarRangeFeet {
		return rejected(SensesNotSharableRejection{})
	}
	d := r.draft("")
	d.links[d.indexOf(f.creature)].familiar.sight = true
	d.events = append(d.events, FamiliarSightLinkStarted{summonEventBase: pair(f.creature, f.summoner)})
	return d.applied()
}

// TurnStarted ends the sight link of a combatant whose turn started (FR-046), reporting FamiliarSightLinkEnded. It returns
// SummonedUnchanged when there is none.
func (r SummonedRoster) TurnStarted(c ReportTurnStarted) SummonedTransition {
	d := r.draft("")
	for i, l := range d.links {
		if l.summoner == c.Combatant && l.kind == SummonFamiliar && l.familiar.sight {
			d.links[i].familiar.sight = false
			d.events = append(d.events, FamiliarSightLinkEnded{summonEventBase: pair(l.creature, l.summoner)})
		}
	}
	if len(d.events) == 0 {
		return SummonedUnchanged{}
	}
	return d.applied()
}

// DeliverTouch has the familiar deliver a touch spell, spending the familiar's Reaction through FeatureReaction(find_familiar,
// NoStandardAction) and leaving the summoner's Reaction alone (FR-047). It returns SummonedRejected with
// TouchNotDeliverableRejection for the first reason that holds: not present, not Touch, farther than 100 feet (exactly 100 is
// accepted), the familiar cannot react. The roster is unchanged. It never panics.
func (r SummonedRoster) DeliverTouch(c DeliverTouch) SummonedTransition {
	f, ok := r.familiarOf(c.Summoner)
	var reason TouchReason
	switch {
	case !ok || f.presence != presenceInEncounter:
		reason = TouchNotPresent
	case !c.SpellIsTouch:
		reason = TouchNotTouch
	case c.DistanceFeet > familiarRangeFeet:
		reason = TouchTooFar
	case c.Familiar != ConditionActionAvailabilityAvailable:
		reason = TouchFamiliarCannotReact
	default:
		a := r.draft("").applied()
		a.reaction, a.hasReaction = FeatureReaction{Feature: content.FeatureID(SummonFamiliar.Spell()), Granted: NoStandardAction{}}, true
		return a
	}
	return rejected(TouchNotDeliverableRejection{Reason: reason})
}

// FamiliarAction asks whether a familiar may take an action (FR-048): the Attack action only with a FamiliarAttackGrant, every
// other action as normal. It returns SummonedRejected with FamiliarCannotAttackRejection, CommandNotYourCreatureRejection for a
// creature that is not a familiar of the roster, and Applied with nothing changed otherwise. It never panics.
func (r SummonedRoster) FamiliarAction(c RequestFamiliarAction) SummonedTransition {
	l, ok := r.Link(c.Creature)
	switch {
	case !ok || l.kind != SummonFamiliar:
		return rejected(CommandNotYourCreatureRejection{Creature: c.Creature})
	case c.Action == StandardActionAttack && !c.Grant.Valid():
		return rejected(FamiliarCannotAttackRejection{})
	}
	return r.draft("").applied()
}

// Repair restores the Hit Points of a simulacrum (FR-062): 100 GP of components for each Hit Point. It returns SummonedRejected
// with CommandNotYourCreatureRejection for a creature that is not a simulacrum of the summoner, SimulacrumRepairNeedsLongRestRejection
// without a Long Rest, SimulacrumTooFarRejection beyond 5 feet (exactly 5 is accepted) and SimulacrumRepairAmountRejection for 0
// Hit Points or more than the missing ones. The roster is unchanged. It never panics.
func (r SummonedRoster) Repair(c RepairSimulacrum) SummonedTransition {
	l, ok := r.activeLink(c.Creature, c.Summoner)
	switch {
	case !ok || l.kind != SummonSimulacrum:
		return rejected(CommandNotYourCreatureRejection{Creature: c.Creature})
	case !c.LongRestTaken:
		return rejected(SimulacrumRepairNeedsLongRestRejection{})
	case c.DistanceFeet > repairRangeFeet:
		return rejected(SimulacrumTooFarRejection{})
	case c.Restore < 1 || c.Restore > c.Missing:
		return rejected(SimulacrumRepairAmountRejection{Missing: c.Missing})
	}
	d := r.draft("")
	d.events = append(d.events, SimulacrumRepaired{summonEventBase: pair(l.creature, l.summoner), restored: c.Restore, goldPieces: repairGoldPerHitPt * c.Restore})
	return d.applied()
}

// CheckHealing refuses healing of a simulacrum from any source but its repair (FR-062): it returns SimulacrumHealingRefusedRejection
// for a simulacrum of the roster and nil for any other creature.
func (r SummonedRoster) CheckHealing(creature CombatantID) SummonRejection {
	if l, ok := r.Link(creature); ok && l.kind == SummonSimulacrum {
		return SimulacrumHealingRefusedRejection{}
	}
	return nil
}

// LifeBond returns the Hit Points a steed regains when its summoner regains Hit Points from a spell of level 1 or higher while the
// steed is within 5 feet (FR-057): the same number, and 0 for a cantrip or a source that is not a spell. It returns
// SummonedRejected with CommandNotYourCreatureRejection for a steed that is not the summoner's and a rejection with
// LifeBondOutOfReachRejection beyond 5 feet (exactly 5 is accepted). The roster is unchanged. It never panics.
func (r SummonedRoster) LifeBond(c ApplySteedLifeBond) SummonedTransition {
	l, ok := r.activeLink(c.Steed, c.Summoner)
	switch {
	case !ok || l.kind != SummonOtherworldlySteed:
		return rejected(CommandNotYourCreatureRejection{Creature: c.Steed})
	case c.DistanceFeet > lifeBondRangeFeet:
		return rejected(LifeBondOutOfReachRejection{})
	}
	a := r.draft("").applied()
	if c.SpellLevel >= 1 {
		a.lifeBond = c.Regained
	}
	return a
}

// TurnEnded consumes the order of a creature whose turn ended, except an undead creature's, which lasts until the task is complete
// (FR-013). It returns SummonedUnchanged when the creature has no order to consume.
func (r SummonedRoster) TurnEnded(c ReportTurnEnded) SummonedTransition {
	l, ok := r.Link(c.Creature)
	if !ok || l.order == nil || l.kind.undead() {
		return SummonedUnchanged{}
	}
	d := r.draft("")
	d.links[d.indexOf(c.Creature)].order = nil
	return d.applied()
}

// TaskComplete ends the standing order of an undead creature (FR-013). It returns SummonedUnchanged for a creature that is not an
// undead creature with an order.
func (r SummonedRoster) TaskComplete(c ReportTaskComplete) SummonedTransition {
	l, ok := r.Link(c.Creature)
	if !ok || l.order == nil || !l.kind.undead() {
		return SummonedUnchanged{}
	}
	d := r.draft("")
	d.links[d.indexOf(c.Creature)].order = nil
	return d.applied()
}

// TurnPlan returns what a summoned creature does on its turn (FR-013): its order, or the Dodge action for a creature of a kind
// that takes it when it has none (Animated Object, Giant Insect, Draconic Spirit, undead). It returns false for a creature the
// roster does not hold or has in the pocket dimension, for an undead creature whose control lapsed (what it does is the Game
// Master's, FR-040) and for a familiar, steed or simulacrum with no order, which act on their own.
func (r SummonedRoster) TurnPlan(creature CombatantID) (SummonedTurnPlan, bool) {
	l, ok := r.Link(creature)
	if !ok || l.presence != presenceInEncounter || l.window.Lapsed() {
		return nil, false
	}
	if l.order != nil {
		return TurnPlanOrdered{Order: l.order}, true
	}
	if l.kind.dodgesByDefault() {
		return TurnPlanDefaultDodge{}, true
	}
	return nil, false
}

// FamiliarTelepathy returns the telepathy of a summoner toward its present familiar: NewTelepathic(100) while the familiar is
// within 100 feet, none otherwise (FR-045). Contact, ending and reply stay with the telepathy rules.
func (r SummonedRoster) FamiliarTelepathy(summoner CombatantID, distanceFeet int) TelepathyCapability {
	if f, ok := r.familiarOf(summoner); ok && f.presence == presenceInEncounter && distanceFeet >= 0 && distanceFeet <= familiarRangeFeet {
		return MustTelepathic(familiarRangeFeet)
	}
	return TelepathyCapability{}
}

// SteedTelepathy returns the telepathy of a steed of the roster: NewTelepathic(5280), 1 mile; none for a creature that is not a
// steed (FR-058).
func (r SummonedRoster) SteedTelepathy(steed CombatantID) TelepathyCapability {
	if l, ok := r.Link(steed); ok && l.kind == SummonOtherworldlySteed {
		return MustTelepathic(steedTelepathyFeet)
	}
	return TelepathyCapability{}
}

// CheckSteedContact refuses a telepathic contact of a steed with any creature but its summoner (FR-058): it returns
// TelepathyOnlyWithSummonerRejection when steed is a steed of the roster and other is not its summoner, nil otherwise.
func (r SummonedRoster) CheckSteedContact(steed, other CombatantID) SummonRejection {
	if l, ok := r.Link(steed); ok && l.kind == SummonOtherworldlySteed && l.summoner != other {
		return TelepathyOnlyWithSummonerRejection{}
	}
	return nil
}

// Apply applies one change of the closed set to the roster, with active the combatant the host's tracker has active (FR-072,
// FR-077). It returns the transition of the method of the change. It never panics.
func (r SummonedRoster) Apply(change SummonedChange, active CombatantID) SummonedTransition {
	switch v := change.(type) {
	case IssueCommand:
		return r.IssueCommand(v)
	case ReportZeroHitPoints:
		return r.ReportZeroHitPoints(v, active)
	case ReportSpellEnded:
		return r.ReportSpellEnded(v, active)
	case ReportSummonerDied:
		return r.ReportSummonerDied(v, active)
	case ReportLongRestFinished:
		return r.ReportLongRestFinished(v, active)
	case ElapseControlTime:
		return r.ElapseControl(v.Elapsed)
	case DismissFamiliar:
		return r.Dismiss(v, active)
	case RecallFamiliar:
		return r.Recall(v)
	case ShareFamiliarSenses:
		return r.ShareSenses(v)
	case DeliverTouch:
		return r.DeliverTouch(v)
	case RepairSimulacrum:
		return r.Repair(v)
	case ApplySteedLifeBond:
		return r.LifeBond(v)
	case ReportTurnStarted:
		return r.TurnStarted(v)
	case ReportTurnEnded:
		return r.TurnEnded(v)
	case ReportTaskComplete:
		return r.TaskComplete(v)
	case RequestFamiliarAction:
		return r.FamiliarAction(v)
	}
	return SummonedUnchanged{}
}

// ApplySummonedChangeCommand is the immutable input of ApplySummonedChange.Run. Build it with NewApplySummonedChangeCommand; the
// zero value is invalid.
type ApplySummonedChangeCommand struct {
	roster SummonedRoster
	change SummonedChange
	active CombatantID
}

// IsZero reports whether c is the zero command.
func (c ApplySummonedChangeCommand) IsZero() bool { return c.change == nil }

// NewApplySummonedChangeCommand returns the command applying change to roster. active is the combatant the host's tracker has
// active, the zero value when none (FR-077). It returns an error matching srderr.ErrInvalidInput, and never panics, for a nil
// change or one that holds a blank id, a negative or oversized distance, duration or amount, an empty or repeating list of targets,
// a Summoner that is its own target, a kind with no order (the steed), an order that is not valid, an enum value that is not
// defined, or a recall whose Initiative is present exactly when it should not be (FR-067).
func NewApplySummonedChangeCommand(roster SummonedRoster, change SummonedChange, active CombatantID) (ApplySummonedChangeCommand, error) {
	if change == nil || !validChange(change) {
		return ApplySummonedChangeCommand{}, invalidInput("a summoned change needs well-formed ids, numbers and enums")
	}
	return ApplySummonedChangeCommand{roster, cloneChange(change), active}, nil
}

// MustApplySummonedChangeCommand is NewApplySummonedChangeCommand for values known to be valid. It panics with the error
// NewApplySummonedChangeCommand returns, and only then.
func MustApplySummonedChangeCommand(roster SummonedRoster, change SummonedChange, active CombatantID) ApplySummonedChangeCommand {
	c, err := NewApplySummonedChangeCommand(roster, change, active)
	if err != nil {
		panic(err)
	}
	return c
}
