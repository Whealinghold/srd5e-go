package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sumGhouls is a roster whose summoner "pc" has one Ghoul "g1" from Create Undead at level 6.
func sumGhouls(tb testing.TB) combat.SummonedRoster {
	tb.Helper()
	return sumDo(tb, combat.SummonedRoster{}, "pc", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("g1", "r1", combat.UndeadGhoul)}}, sumScene()).Roster()
}

// sumCommand is the order of the summoner to creatures of the kind at the distance, with verbal commands possible.
func sumCommand(kind combat.SummonKind, distance int, ids ...string) combat.IssueCommand {
	return combat.IssueCommand{Summoner: cid("pc"), Kind: kind, Targets: sumTargets(distance, ids...), Order: sumOrder(), CanGiveVerbalCommand: true}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-009, Table K - a mental order is refused with CommandOutOfRangeRejection when the creature is farther than the channel's range (500 feet for Animate Objects, 60 for Animate Dead, 120 for Create Undead), and accepted exactly at the range (SRD 5.2.1, "Animate Objects", p. 109; "Animate Dead", p. 108; "Create Undead", p. 121).
// Path: per case.
func TestMentalOrdersAreRangedByTheirChannel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		kind                    combat.SummonKind
		id                      string
		distance, rangeFeet     int
		accepted                bool
	}{
		{"objects at 500", "FR-009, p. 109", "unhappy (boundary)", sumWorld(t), combat.SummonAnimatedObject, "o1", 500, 500, true},
		{"objects at 501", "FR-009, p. 109", "unhappy (boundary)", sumWorld(t), combat.SummonAnimatedObject, "o1", 501, 500, false},
		{"animate dead at 60", "FR-009, p. 108", "unhappy (boundary)", sumWorld(t), combat.SummonAnimatedUndead, "u1", 60, 60, true},
		{"animate dead at 61", "FR-009, p. 108", "unhappy (boundary)", sumWorld(t), combat.SummonAnimatedUndead, "u1", 61, 60, false},
		{"create undead at 120", "FR-009, p. 121", "unhappy (boundary)", sumGhouls(t), combat.SummonCreatedUndead, "g1", 120, 120, true},
		{"create undead at 121", "FR-009, p. 121", "unhappy (boundary)", sumGhouls(t), combat.SummonCreatedUndead, "g1", 121, 120, false},
		{"within range", "FR-009", "happy", sumWorld(t), combat.SummonAnimatedUndead, "u1", 0, 60, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumChange(t, c.roster, sumCommand(c.kind, c.distance, c.id), "")
			if c.accepted {
				if _, ok := got.(combat.SummonedApplied); !ok {
					t.Errorf("transition = %#v, want applied [%s]", got, c.requirement)
				}
				return
			}
			if rej := sumRejection(t, got); rej != (combat.CommandOutOfRangeRejection{Creature: cid(c.id), RangeFeet: c.rangeFeet}) {
				t.Errorf("rejection = %#v, want CommandOutOfRangeRejection{%s, %d} [%s]", rej, c.id, c.rangeFeet, c.requirement)
			}
		})
	}
	if ch, ok := combat.SummonAnimatedObject.Channel().(combat.ChannelBonusActionMental); !ok || ch.RangeFeet != 500 {
		t.Errorf("Channel = %#v, want ChannelBonusActionMental{500} [Table K]", combat.SummonAnimatedObject.Channel())
	}
	if _, ok := combat.SummonGiantInsect.Channel().(combat.ChannelVerbalNoAction); !ok {
		t.Errorf("Channel of an insect = %#v, want ChannelVerbalNoAction [Table K]", combat.SummonGiantInsect.Channel())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-010, FR-046 - one order commands any number of the summoner's creatures of the spell, all receiving the same order, in one event; BonusActionGrantFor gives the grant of the spell's feature (no standard action) for the Bonus Action the host takes, and refuses a kind with no Bonus Action; this package spends none.
// Path: happy.
func TestOneOrderCommandsAGroup(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	a := sumApplied(t, sumChange(t, roster, sumCommand(combat.SummonAnimatedObject, 10, "o1", "o2"), ""))
	ev, ok := a.Events()[0].(combat.SummonedCreatureCommanded)
	if !ok || len(a.Events()) != 1 || ev.Order() != sumOrder() || ev.SummonKind() != combat.SummonAnimatedObject ||
		!slices.Equal(ev.Combatants(), []combat.CombatantID{"pc", "o1", "o2"}) {
		t.Errorf("events = %#v, want one SummonedCreatureCommanded of [pc o1 o2] [FR-010]", a.Events())
	}
	for _, id := range []string{"o1", "o2"} {
		if l, _ := a.Roster().Link(cid(id)); l.Order() != sumOrder() {
			t.Errorf("%s order = %v, want the order given", id, l.Order())
		}
	}
	if l, _ := a.Roster().Link("i1"); l.Order() != nil {
		t.Errorf("an insect not named got an order")
	}
	for _, kind := range []combat.SummonKind{combat.SummonAnimatedObject, combat.SummonAnimatedUndead, combat.SummonCreatedUndead, combat.SummonFamiliar} {
		want := luMust(combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: content.FeatureID(kind.Spell())}, combat.NoStandardAction{}))
		if got, err := combat.BonusActionGrantFor(kind); err != nil || got != want {
			t.Errorf("BonusActionGrantFor(%v) = %v, %v, want the grant of %s [FR-010]", kind, got, err, kind.Spell())
		}
	}
	for _, kind := range []combat.SummonKind{combat.SummonGiantInsect, combat.SummonDraconicSpirit, combat.SummonOtherworldlySteed, combat.SummonSimulacrum, 0} {
		if _, err := combat.BonusActionGrantFor(kind); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("BonusActionGrantFor(%v) error = %v, want srderr.ErrInvalidInput [FR-067]", kind, err)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-011 - Giant Insects and Draconic Spirits are commanded by a verbal command that costs no action: refused with CannotGiveVerbalCommandRejection when the host reports the summoner cannot give one, accepted otherwise at any distance (SRD 5.2.1, "Giant Insect", p. 135; "Summon Dragon", p. 166).
// Path: per case.
func TestVerbalCommandsNeedAVoice(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, id string
		kind     combat.SummonKind
		verbal   bool
	}{
		{"insect with a voice", "i1", combat.SummonGiantInsect, true}, {"insect without", "i1", combat.SummonGiantInsect, false},
		{"spirit with a voice", "d1", combat.SummonDraconicSpirit, true}, {"spirit without", "d1", combat.SummonDraconicSpirit, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := sumCommand(c.kind, 5000, c.id)
			cmd.CanGiveVerbalCommand = c.verbal
			got := sumChange(t, sumWorld(t), cmd, "")
			if c.verbal {
				if _, ok := got.(combat.SummonedApplied); !ok {
					t.Errorf("transition = %#v, want applied at any distance [FR-011]", got)
				}
				return
			}
			if _, ok := sumRejection(t, got).(combat.CannotGiveVerbalCommandRejection); !ok {
				t.Errorf("rejection = %#v, want CannotGiveVerbalCommandRejection [FR-011]", got)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-012, FR-067 - an order is a SpecificOrder or a GeneralOrder of opaque tokens; the engine resolves neither; an order with a blank token is invalid input.
// Path: per case.
func TestOrdersAreOpaqueTokens(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	for _, o := range []combat.SummonedOrder{combat.SpecificOrder{Action: "attack_the_ogre", Movement: "stay_close"}, combat.GeneralOrder{Label: "guard_the_door"}} {
		cmd := sumCommand(combat.SummonAnimatedObject, 0, "o1")
		cmd.Order = o
		a := sumApplied(t, sumChange(t, roster, cmd, ""))
		if l, _ := a.Roster().Link("o1"); l.Order() != o {
			t.Errorf("order = %#v, want %#v [FR-012]", l.Order(), o)
		}
	}
	for _, bad := range []combat.SummonedOrder{nil, combat.SpecificOrder{}, combat.SpecificOrder{Action: "a"}, combat.SpecificOrder{Movement: "m"}, combat.GeneralOrder{}} {
		cmd := sumCommand(combat.SummonAnimatedObject, 0, "o1")
		cmd.Order = bad
		if _, err := combat.NewApplySummonedChangeCommand(roster, cmd, ""); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("order %#v error = %v, want srderr.ErrInvalidInput [FR-067]", bad, err)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-013, Q-4 - with no order a creature takes the Dodge action (objects, insects, spirits, undead); an order is consumed after one turn, except an undead creature's, which lasts until the host reports the task complete; a familiar, steed or simulacrum with no order has no plan, and nor has a creature in the pocket dimension or whose control lapsed (SRD 5.2.1, "Dodge", p. 181).
// Path: per case.
func TestTurnPlanAndTheLifeOfAnOrder(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	dodge := combat.TurnPlanDefaultDodge{}
	for _, id := range []string{"o1", "i1", "d1", "u1"} {
		if p, ok := roster.TurnPlan(cid(id)); !ok || p != dodge {
			t.Errorf("TurnPlan(%s) = %v, %v, want the Dodge action [FR-013]", id, p, ok)
		}
	}
	for _, id := range []string{"fam", "st", "sim", "nobody"} {
		if p, ok := roster.TurnPlan(cid(id)); ok {
			t.Errorf("TurnPlan(%s) = %v, want none: it acts on its own [FR-013]", id, p)
		}
	}
	// A non-undead order lasts one turn.
	ordered := sumApplied(t, sumChange(t, roster, sumCommand(combat.SummonAnimatedObject, 0, "o1"), "")).Roster()
	if p, ok := ordered.TurnPlan("o1"); !ok || p != (combat.TurnPlanOrdered{Order: sumOrder()}) {
		t.Fatalf("TurnPlan(o1) = %v, %v, want the order", p, ok)
	}
	if p, _ := ordered.TurnPlan("o2"); p != dodge {
		t.Errorf("an object not commanded has plan %v, want Dodge", p)
	}
	spent := sumApplied(t, sumChange(t, ordered, combat.ReportTurnEnded{Creature: cid("o1")}, "")).Roster()
	if p, _ := spent.TurnPlan("o1"); p != dodge {
		t.Errorf("after its turn the plan is %v, want Dodge: the order was consumed [FR-013]", p)
	}
	if _, ok := sumChange(t, spent, combat.ReportTurnEnded{Creature: cid("o1")}, "").(combat.SummonedUnchanged); !ok {
		t.Errorf("ending a turn with no order changed the roster")
	}
	// An undead order stands until the task is complete.
	standing := sumApplied(t, sumChange(t, roster, sumCommand(combat.SummonAnimatedUndead, 0, "u1"), "")).Roster()
	for i := 0; i < 3; i++ {
		if _, ok := sumChange(t, standing, combat.ReportTurnEnded{Creature: cid("u1")}, "").(combat.SummonedUnchanged); !ok {
			t.Fatalf("an undead order was consumed by a turn [FR-013]")
		}
	}
	if p, _ := standing.TurnPlan("u1"); p != (combat.TurnPlanOrdered{Order: sumOrder()}) {
		t.Errorf("an undead creature's plan = %v, want its standing order", p)
	}
	done := sumApplied(t, sumChange(t, standing, combat.ReportTaskComplete{Creature: cid("u1")}, "")).Roster()
	if p, _ := done.TurnPlan("u1"); p != dodge {
		t.Errorf("after the task the plan is %v, want Dodge [FR-013]", p)
	}
	for _, id := range []string{"o1", "nobody"} {
		if _, ok := sumChange(t, ordered, combat.ReportTaskComplete{Creature: cid(id)}, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("a task completed by %s changed the roster: only an undead creature's standing order ends so [FR-013]", id)
		}
	}
	// A creature in the pocket dimension has no plan, and nor has an undead creature whose control lapsed.
	pocket := sumApplied(t, sumChange(t, roster, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	cmd := sumCommand(combat.SummonFamiliar, 0, "fam")
	if _, ok := sumChange(t, roster, cmd, "").(combat.SummonedApplied); !ok {
		t.Fatalf("a familiar did not obey")
	}
	if _, ok := pocket.TurnPlan("fam"); ok {
		t.Errorf("a creature in the pocket dimension has a plan")
	}
	lapsed := sumApplied(t, sumChange(t, standing, combat.ElapseControlTime{Elapsed: 24 * time.Hour}, "")).Roster()
	if _, ok := lapsed.TurnPlan("u1"); ok {
		t.Errorf("an undead creature whose control lapsed has a plan: its conduct is the GM's [FR-040]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-014 - an order naming a creature that is not in the roster, not linked to that summoner, in the pocket dimension or of another spell than the command's is refused with CommandNotYourCreatureRejection and changes nothing; a group order is atomic: one bad target refuses all.
// Path: per case.
func TestAnOrderToAForeignCreatureIsRefusedWhole(t *testing.T) {
	t.Parallel()
	world := sumWorld(t)
	other := sumDo(t, world, "other", sumObjects("x1"), sumScene()).Roster()
	pocket := sumApplied(t, sumChange(t, world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	cases := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		cmd                     combat.IssueCommand
		bad                     string
	}{
		{"a creature that is not in the roster", "FR-014", "unhappy (invalid input)", world, sumCommand(combat.SummonAnimatedObject, 0, "ghost"), "ghost"},
		{"a creature of another summoner", "FR-014", "unhappy (invalid input)", other, sumCommand(combat.SummonAnimatedObject, 0, "x1"), "x1"},
		{"a creature of another spell", "FR-014", "unhappy (invalid input)", world, sumCommand(combat.SummonAnimatedObject, 0, "i1"), "i1"},
		{"a familiar in the pocket dimension", "FR-014", "unhappy (state error)", pocket, sumCommand(combat.SummonFamiliar, 0, "fam"), "fam"},
		{"one bad target of two", "FR-014", "unhappy (rule interaction)", world, sumCommand(combat.SummonAnimatedObject, 0, "o1", "x1"), "x1"},
		{"the bad target first", "FR-014", "unhappy (rule interaction)", other, sumCommand(combat.SummonAnimatedObject, 0, "x1", "o1"), "x1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.roster
			rej := sumRejection(t, sumChange(t, c.roster, c.cmd, ""))
			if rej != (combat.CommandNotYourCreatureRejection{Creature: cid(c.bad)}) {
				t.Errorf("rejection = %#v, want CommandNotYourCreatureRejection{%s} [%s]", rej, c.bad, c.requirement)
			}
			if !before.Equal(c.roster) {
				t.Errorf("a refused order changed the roster [FR-014]")
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-015, FR-040, FR-041 - an undead creature starts with a control window of 24 hours; time subtracts from it and at 0 or below the window lapses with SummonControlLapsed (the creature stays linked, uncontrolled, and a lapsed window never revives); an order to it is refused with ControlLapsedRejection, 1 minute left being accepted and 0 refused.
// Path: per case.
func TestControlOfAnUndeadCreatureLapses(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	if l, _ := roster.Link("u1"); func() bool { w, ok := l.ControlWindow(); return !ok || w.Left() != 24*time.Hour || w.Lapsed() }() {
		t.Fatalf("a new undead creature has window %v, want 24 hours", l)
	}
	if _, has := func() (combat.ControlWindow, bool) { l, _ := roster.Link("o1"); return l.ControlWindow() }(); has {
		t.Errorf("an animated object has a control window")
	}
	cases := []struct {
		name, requirement, path string
		elapsed                 time.Duration
		left                    time.Duration
		lapsed                  bool
	}{
		{"an hour", "FR-040", "happy", time.Hour, 23 * time.Hour, false},
		{"all but a minute", "FR-041", "unhappy (boundary)", 24*time.Hour - time.Minute, time.Minute, false},
		{"exactly 24 hours", "FR-040", "unhappy (boundary)", 24 * time.Hour, 0, true},
		{"more than 24 hours", "FR-040", "unhappy (boundary)", 100 * time.Hour, 0, true},
		{"nothing", "FR-040", "unhappy (boundary)", 0, 24 * time.Hour, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := sumApplied(t, sumChange(t, roster, combat.ElapseControlTime{Elapsed: c.elapsed}, ""))
			l, _ := a.Roster().Link("u1")
			w, _ := l.ControlWindow()
			if w.Left() != c.left || w.Lapsed() != c.lapsed {
				t.Errorf("window = %v lapsed %v, want %v lapsed %v [%s]", w.Left(), w.Lapsed(), c.left, c.lapsed, c.requirement)
			}
			if got := sumKinds(a.Events()); c.lapsed != (len(got) == 2) || (c.lapsed && (got[0] != "summon_control_lapsed" || got[1] != "summon_control_lapsed")) {
				t.Errorf("events = %v, want a lapse for each of the two undead creatures only when lapsed [FR-040]", got)
			}
			order := sumChange(t, a.Roster(), sumCommand(combat.SummonAnimatedUndead, 0, "u1"), "")
			if c.lapsed {
				if rej := sumRejection(t, order); rej != (combat.ControlLapsedRejection{Creature: cid("u1")}) {
					t.Errorf("order after the lapse = %#v, want ControlLapsedRejection [FR-015]", rej)
				}
				if _, still := a.Roster().Link("u1"); !still || a.Roster().Len() != roster.Len() {
					t.Errorf("a lapsed creature left the roster [FR-040]")
				}
			} else if _, ok := order.(combat.SummonedApplied); !ok {
				t.Errorf("order with %v left = %#v, want applied [%s]", c.left, order, c.requirement)
			}
		})
	}
	lapsed := sumApplied(t, sumChange(t, roster, combat.ElapseControlTime{Elapsed: 24 * time.Hour}, "")).Roster()
	again := sumChange(t, lapsed, combat.ElapseControlTime{Elapsed: time.Hour}, "")
	if _, ok := again.(combat.SummonedUnchanged); !ok {
		t.Errorf("time after a lapse = %#v, want unchanged: a lapsed window never revives and lapses once [FR-040]", again)
	}
	if _, ok := sumChange(t, combat.SummonedRoster{}, combat.ElapseControlTime{Elapsed: time.Hour}, "").(combat.SummonedUnchanged); !ok {
		t.Errorf("time with no undead creature changed the roster")
	}
	if _, err := combat.NewApplySummonedChangeCommand(roster, combat.ElapseControlTime{Elapsed: -time.Second}, ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative time error = %v, want srderr.ErrInvalidInput [FR-040]", err)
	}
	if got := roster.ElapseControl(-time.Hour); !reflect.DeepEqual(got, roster.ElapseControl(0)) {
		t.Errorf("a negative duration shrank a window at the domain level")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-016, FR-017 - a familiar and a simulacrum obey commands at no action cost at any distance the host reports, and a steed takes no order (its command is invalid input).
// Path: per case.
func TestFamiliarAndSimulacrumObeyAtAnyDistance(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	for _, c := range []struct {
		kind combat.SummonKind
		id   string
	}{{combat.SummonFamiliar, "fam"}, {combat.SummonSimulacrum, "sim"}} {
		cmd := sumCommand(c.kind, 1_000_000, c.id)
		cmd.CanGiveVerbalCommand = false
		if _, ok := sumChange(t, roster, cmd, "").(combat.SummonedApplied); !ok {
			t.Errorf("a %v did not obey at any distance [FR-016, FR-017]", c.kind)
		}
		if _, ok := c.kind.Channel().(combat.ChannelObeysIndependently); !ok {
			t.Errorf("Channel of %v = %#v, want ChannelObeysIndependently", c.kind, c.kind.Channel())
		}
	}
	if _, err := combat.NewApplySummonedChangeCommand(roster, sumCommand(combat.SummonOtherworldlySteed, 0, "st"), ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an order to a steed error = %v, want srderr.ErrInvalidInput [Table K]", err)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-068, Table R - when several rejections of an order hold the reported one is the first of CommandNotYourCreature, ControlLapsed, CommandOutOfRange, CannotGiveVerbalCommand; each is checked for every creature before the next, and the mental and verbal reasons never hold for one kind.
// Path: per case.
func TestRejectionsOfAnOrderFollowTableR(t *testing.T) {
	t.Parallel()
	lapsed := sumApplied(t, sumChange(t, sumWorld(t), combat.ElapseControlTime{Elapsed: 24 * time.Hour}, "")).Roster()
	fresh := sumWorld(t)
	far := func(id string, d int) combat.CommandTarget {
		return combat.CommandTarget{Creature: cid(id), DistanceFeet: d}
	}
	cases := []struct {
		name, path string
		roster     combat.SummonedRoster
		cmd        combat.IssueCommand
		want       combat.SummonRejection
	}{
		{"not yours before lapsed", "unhappy (rule interaction)", lapsed, combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedUndead, Targets: []combat.CommandTarget{far("u1", 0), far("ghost", 0)}, Order: sumOrder()}, combat.CommandNotYourCreatureRejection{Creature: cid("ghost")}},
		{"lapsed before out of range", "unhappy (rule interaction)", lapsed, combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedUndead, Targets: []combat.CommandTarget{far("u1", 500)}, Order: sumOrder()}, combat.ControlLapsedRejection{Creature: cid("u1")}},
		{"out of range for the second creature after a lapse for the first", "unhappy (rule interaction)", fresh, combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedUndead, Targets: []combat.CommandTarget{far("u1", 0), far("u2", 61)}, Order: sumOrder()}, combat.CommandOutOfRangeRejection{Creature: cid("u2"), RangeFeet: 60}},
		{"verbal when no other reason holds", "unhappy (rule interaction)", fresh, combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonGiantInsect, Targets: []combat.CommandTarget{far("i1", 9999)}, Order: sumOrder()}, combat.CannotGiveVerbalCommandRejection{}},
		{"not yours before verbal", "unhappy (rule interaction)", fresh, combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonGiantInsect, Targets: []combat.CommandTarget{far("ghost", 0)}, Order: sumOrder()}, combat.CommandNotYourCreatureRejection{Creature: cid("ghost")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := sumRejection(t, sumChange(t, c.roster, c.cmd, "")); got != c.want {
				t.Errorf("rejection = %#v, want %#v [FR-068]", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-067 - NewApplySummonedChangeCommand returns an error matching srderr.ErrInvalidInput, and never panics, for a nil change, a blank id, a negative or oversized distance or amount, an empty or repeating list of targets, a summoner that is its own target, a kind with no order, an order that is not valid and an enum that is not defined, and accepts the values at their bounds.
// Path: per case.
func TestChangeCommandsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	dmg := combat.ReportZeroHitPoints{Creature: cid("o1")}
	bad := map[string]combat.SummonedChange{
		"nil change":                                nil,
		"order with a blank summoner":               combat.IssueCommand{Kind: combat.SummonAnimatedObject, Targets: sumTargets(0, "o1"), Order: sumOrder()},
		"order with no kind":                        combat.IssueCommand{Summoner: cid("pc"), Targets: sumTargets(0, "o1"), Order: sumOrder()},
		"order with no target":                      combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Order: sumOrder()},
		"order with a blank target":                 combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Targets: sumTargets(0, ""), Order: sumOrder()},
		"order with a repeated target":              combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Targets: sumTargets(0, "o1", "o1"), Order: sumOrder()},
		"order to the summoner":                     combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Targets: sumTargets(0, "pc"), Order: sumOrder()},
		"order at a negative distance":              combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Targets: sumTargets(-1, "o1"), Order: sumOrder()},
		"order at an oversized distance":            combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Targets: sumTargets(1_000_000_001, "o1"), Order: sumOrder()},
		"zero hit points with a blank id":           combat.ReportZeroHitPoints{},
		"spell ended with a blank spell":            combat.ReportSpellEnded{Summoner: cid("pc"), Cause: combat.SummonSpellEndDispelled},
		"spell ended with an unknown cause":         combat.ReportSpellEnded{Summoner: cid("pc"), Spell: "animate_objects"},
		"summoner died, blank":                      combat.ReportSummonerDied{},
		"long rest, blank":                          combat.ReportLongRestFinished{},
		"dismissal with no way":                     combat.DismissFamiliar{Summoner: cid("pc")},
		"dismissal, blank summoner":                 combat.DismissFamiliar{How: combat.DismissalForever},
		"recall in combat without a tie":            combat.RecallFamiliar{Summoner: cid("pc"), InCombat: true},
		"recall out of combat with a tie":           combat.RecallFamiliar{Summoner: cid("pc"), Initiative: sumInit(3)},
		"senses at a negative distance":             combat.ShareFamiliarSenses{Summoner: cid("pc"), DistanceFeet: -1},
		"senses, blank summoner":                    combat.ShareFamiliarSenses{},
		"touch at a negative distance":              combat.DeliverTouch{Summoner: cid("pc"), DistanceFeet: -1, Familiar: combat.ConditionActionAvailabilityAvailable},
		"touch with no availability":                combat.DeliverTouch{Summoner: cid("pc")},
		"repair of itself":                          combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("pc")},
		"repair of a negative amount":               combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Restore: -1},
		"repair missing too large":                  combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Missing: 1_000_001},
		"repair restore too large":                  combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Restore: 1_000_001},
		"life bond of a steed that is the summoner": combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("pc")},
		"life bond regained negative":               combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: -1},
		"life bond level 10":                        combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), SpellLevel: 10},
		"life bond level negative":                  combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), SpellLevel: -1},
		"turn started, blank":                       combat.ReportTurnStarted{},
		"turn ended, blank":                         combat.ReportTurnEnded{},
		"task complete, blank":                      combat.ReportTaskComplete{},
		"action with no action":                     combat.RequestFamiliarAction{Creature: cid("fam")},
		"action with an unknown grant":              combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionDodge, Grant: 9},
		"action, blank creature":                    combat.RequestFamiliarAction{Action: combat.StandardActionDodge},
	}
	for name, change := range bad {
		if _, err := combat.NewApplySummonedChangeCommand(roster, change, ""); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want srderr.ErrInvalidInput [FR-067]", name, err)
		}
	}
	good := []combat.SummonedChange{dmg,
		combat.IssueCommand{Summoner: cid("pc"), Kind: combat.SummonAnimatedObject, Targets: sumTargets(1_000_000_000, "o1"), Order: sumOrder()},
		combat.ElapseControlTime{}, combat.RecallFamiliar{Summoner: cid("pc"), InCombat: true, Initiative: sumInit(3)},
		combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Missing: 1_000_000, Restore: 1_000_000},
		combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: 1_000_000, SpellLevel: 9},
		combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionAttack, Grant: combat.FamiliarAttackViaReaction}}
	for _, change := range good {
		if _, err := combat.NewApplySummonedChangeCommand(roster, change, "pc"); err != nil {
			t.Errorf("%T at its bounds: error = %v, want nil [FR-067]", change, err)
		}
	}
	if cmd, err := combat.NewApplySummonedChangeCommand(roster, dmg, ""); err != nil || cmd.IsZero() || !(combat.ApplySummonedChangeCommand{}).IsZero() {
		t.Errorf("command = %v, %v, want a non-zero command and a zero zero value", cmd, err)
	}
}
