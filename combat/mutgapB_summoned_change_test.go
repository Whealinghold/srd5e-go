package combat_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// mgbPeers is a roster of three summoners whose identifiers sort pc1 < pc2 < pc3: each has a Giant Insect (i1, i2, i3), and pc2
// also has a familiar (fam2).
func mgbPeers(tb testing.TB) combat.SummonedRoster {
	tb.Helper()
	r := sumInsect(tb, combat.SummonedRoster{}, "pc1", "i1")
	r = sumInsect(tb, r, "pc2", "i2")
	r = sumInsect(tb, r, "pc3", "i3")
	return sumFamiliar(tb, r, "pc2", "fam2", combat.FamiliarOriginFindFamiliarSpell)
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-067 - a change is well formed only when every field holds: a Deliver Touch names a defined availability, a simulacrum repair and a steed's Life Bond name two distinct creatures of a summoner with distances and amounts within bounds, and a familiar action carries no undefined grant; each field breaking alone is refused with an error matching srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestMutgapBChangesAreValidatedFieldByField(t *testing.T) {
	t.Parallel()
	touch := combat.DeliverTouch{Summoner: cid("pc"), SpellIsTouch: true, DistanceFeet: 5, Familiar: trkAvail}
	repair := combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Missing: 10, Restore: 5, DistanceFeet: 5, LongRestTaken: true}
	bond := combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: 5, SpellLevel: 2, DistanceFeet: 5}
	action := combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionDash}
	good := []combat.SummonedChange{
		touch, combat.DeliverTouch{Summoner: cid("pc"), SpellIsTouch: true, DistanceFeet: 5, Familiar: trkNone}, repair, bond, action,
		combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionAttack, Grant: combat.FamiliarAttackViaReaction},
		combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Missing: 1_000_000, Restore: 1_000_000, DistanceFeet: 1_000_000_000},
		combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid("st"), Regained: 1_000_000, SpellLevel: 9, DistanceFeet: 1_000_000_000},
	}
	for i, c := range good {
		if _, err := combat.NewApplySummonedChangeCommand(combat.SummonedRoster{}, c, ""); err != nil {
			t.Errorf("good change %d (%T): %v", i, c, err)
		}
	}
	touchWith := func(f func(*combat.DeliverTouch)) combat.SummonedChange { c := touch; f(&c); return c }
	repairWith := func(f func(*combat.RepairSimulacrum)) combat.SummonedChange { c := repair; f(&c); return c }
	bondWith := func(f func(*combat.ApplySteedLifeBond)) combat.SummonedChange { c := bond; f(&c); return c }
	bad := map[string]combat.SummonedChange{
		"touch: availability 0":          touchWith(func(c *combat.DeliverTouch) { c.Familiar = 0 }),
		"touch: availability past None":  touchWith(func(c *combat.DeliverTouch) { c.Familiar = trkNone + 1 }),
		"touch: no summoner":             touchWith(func(c *combat.DeliverTouch) { c.Summoner = "" }),
		"touch: distance below 0":        touchWith(func(c *combat.DeliverTouch) { c.DistanceFeet = -1 }),
		"touch: distance above bound":    touchWith(func(c *combat.DeliverTouch) { c.DistanceFeet = 1_000_000_001 }),
		"repair: no summoner":            repairWith(func(c *combat.RepairSimulacrum) { c.Summoner = "" }),
		"repair: no creature":            repairWith(func(c *combat.RepairSimulacrum) { c.Creature = "" }),
		"repair: summoner is the target": repairWith(func(c *combat.RepairSimulacrum) { c.Creature = c.Summoner }),
		"repair: distance below 0":       repairWith(func(c *combat.RepairSimulacrum) { c.DistanceFeet = -1 }),
		"repair: distance above bound":   repairWith(func(c *combat.RepairSimulacrum) { c.DistanceFeet = 1_000_000_001 }),
		"repair: missing below 0":        repairWith(func(c *combat.RepairSimulacrum) { c.Missing = -1 }),
		"repair: missing above bound":    repairWith(func(c *combat.RepairSimulacrum) { c.Missing = 1_000_001 }),
		"repair: restore below 0":        repairWith(func(c *combat.RepairSimulacrum) { c.Restore = -1 }),
		"repair: restore above bound":    repairWith(func(c *combat.RepairSimulacrum) { c.Restore = 1_000_001 }),
		"bond: no summoner":              bondWith(func(c *combat.ApplySteedLifeBond) { c.Summoner = "" }),
		"bond: no steed":                 bondWith(func(c *combat.ApplySteedLifeBond) { c.Steed = "" }),
		"bond: summoner is the steed":    bondWith(func(c *combat.ApplySteedLifeBond) { c.Steed = c.Summoner }),
		"bond: distance below 0":         bondWith(func(c *combat.ApplySteedLifeBond) { c.DistanceFeet = -1 }),
		"bond: distance above bound":     bondWith(func(c *combat.ApplySteedLifeBond) { c.DistanceFeet = 1_000_000_001 }),
		"bond: regained below 0":         bondWith(func(c *combat.ApplySteedLifeBond) { c.Regained = -1 }),
		"bond: regained above bound":     bondWith(func(c *combat.ApplySteedLifeBond) { c.Regained = 1_000_001 }),
		"bond: spell level below 0":      bondWith(func(c *combat.ApplySteedLifeBond) { c.SpellLevel = -1 }),
		"bond: spell level above 9":      bondWith(func(c *combat.ApplySteedLifeBond) { c.SpellLevel = 10 }),
		"action: negative grant":         combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionDash, Grant: -1},
		"action: undefined grant":        combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionDash, Grant: 99},
	}
	for name, c := range bad {
		if _, err := combat.NewApplySummonedChangeCommand(combat.SummonedRoster{}, c, ""); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want srderr.ErrInvalidInput [FR-067]", name, err)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-009 - an order to a creature is refused with CommandNotYourCreatureRejection when the creature belongs to another summoner, whichever of the two identifiers sorts first, or is of another kind than the order's.
// Path: unhappy (state error).
func TestMutgapBOrdersRefuseAnotherSummonersCreatureOrKind(t *testing.T) {
	t.Parallel()
	r := mgbPeers(t)
	cases := []struct {
		name     string
		summoner string
		kind     combat.SummonKind
	}{
		{"summoner sorting before the owner", "pc1", combat.SummonGiantInsect},
		{"summoner sorting after the owner", "pc3", combat.SummonGiantInsect},
		{"kind sorting after the creature's kind", "pc2", combat.SummonDraconicSpirit},
	}
	for _, c := range cases {
		cmd := combat.IssueCommand{Summoner: cid(c.summoner), Kind: c.kind, Targets: sumTargets(5, "i2"), Order: sumOrder(), CanGiveVerbalCommand: true}
		if rej := sumRejection(t, sumChange(t, r, cmd, "")); rej != (combat.CommandNotYourCreatureRejection{Creature: cid("i2")}) {
			t.Errorf("%s: rejection = %#v, want CommandNotYourCreatureRejection{i2} [FR-009]", c.name, rej)
		}
	}
	own := combat.IssueCommand{Summoner: cid("pc2"), Kind: combat.SummonGiantInsect, Targets: sumTargets(5, "i2"), Order: sumOrder(), CanGiveVerbalCommand: true}
	sumApplied(t, sumChange(t, r, own, ""))
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-022 - a spell ending touches only the creatures of its own summoner and its own spell: another summoner's creatures, whichever identifier sorts first, and a spell that sorts before or after the creature's own, leave the roster unchanged.
// Path: unhappy (rule interaction).
func TestMutgapBSpellEndedTouchesOnlyItsOwnSummonerAndSpell(t *testing.T) {
	t.Parallel()
	r := mgbPeers(t)
	cases := []struct {
		name, summoner, spell string
	}{
		{"summoner sorting before the owner", "pc1", "giant_insect"},
		{"summoner sorting after the owner", "pc3", "giant_insect"},
		{"spell sorting before the creature's", "pc2", "animate_objects"},
		{"spell sorting after the creature's", "pc2", "summon_dragon"},
	}
	for _, c := range cases {
		change := combat.ReportSpellEnded{Summoner: cid(c.summoner), Spell: content.SpellID(c.spell), Cause: combat.SummonSpellEndConcentrationEnded}
		if c.summoner == "pc1" || c.summoner == "pc3" {
			// pc1 and pc3 do own a Giant Insect of this spell, so only the other summoner's creature is the question: i2 must stay.
			a := sumApplied(t, sumChange(t, r, change, ""))
			if _, ok := a.Roster().Link(cid("i2")); !ok {
				t.Errorf("%s: the creature of another summoner departed [FR-022]", c.name)
			}
			continue
		}
		if _, ok := sumChange(t, r, change, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("%s: the roster changed [FR-022]", c.name)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-050, FR-052 - dismissing a familiar temporarily marks its removal as pending after the turn only when the familiar itself is the active combatant, not for any active combatant that sorts after it.
// Path: unhappy (boundary).
func TestMutgapBDismissalIsPendingOnlyForTheActiveFamiliar(t *testing.T) {
	t.Parallel()
	r := mgbPeers(t)
	dismiss := combat.DismissFamiliar{Summoner: cid("pc2"), How: combat.DismissalTemporary}
	for active, want := range map[string]bool{"zz": false, "fam2": true, "": false} {
		a := sumApplied(t, sumChange(t, r, dismiss, active))
		var pending bool
		for _, e := range a.Events() {
			if d, ok := e.(combat.FamiliarDismissedTemporarily); ok {
				pending = d.PendingRemoval()
			}
		}
		var after bool
		for _, act := range a.RosterActions() {
			if l, ok := act.(combat.SummonedRosterLeave); ok {
				after = l.AfterTurn
			}
		}
		if pending != want || after != want {
			t.Errorf("active %q: pending %v, after turn %v, want %v [FR-050]", active, pending, after, want)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-046 - a sight link ends when its summoner's own turn starts and no other combatant's, whichever identifier sorts first.
// Path: unhappy (rule interaction).
func TestMutgapBSightLinkEndsOnlyOnItsSummonersTurn(t *testing.T) {
	t.Parallel()
	r := mgbPeers(t)
	r = sumApplied(t, sumChange(t, r, combat.ShareFamiliarSenses{Summoner: cid("pc2"), DistanceFeet: 5}, "")).Roster()
	for _, other := range []string{"pc1", "pc3"} {
		if _, ok := sumChange(t, r, combat.ReportTurnStarted{Combatant: cid(other)}, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("the turn of %s ended the sight link of pc2 [FR-046]", other)
		}
	}
	a := sumApplied(t, sumChange(t, r, combat.ReportTurnStarted{Combatant: cid("pc2")}, ""))
	if len(a.Events()) != 1 || a.Events()[0].Kind() != "familiar_sight_link_ended" {
		t.Errorf("events = %v, want the sight link ending [FR-046]", sumKinds(a.Events()))
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-048, FR-057, FR-058, FR-062 - the familiar, steed and simulacrum rules apply to creatures of their own kind only: an action request for a simulacrum or a steed, a Life Bond for a familiar, undead creature or simulacrum, and a steed's telepathy and contact rule for every kind sorting after the steed, are refused or empty.
// Path: unhappy (rule interaction).
func TestMutgapBKindRulesApplyToTheirOwnKindOnly(t *testing.T) {
	t.Parallel()
	w := sumWorld(t)
	for _, id := range []string{"sim", "st", "i1", "u1"} {
		got := w.FamiliarAction(combat.RequestFamiliarAction{Creature: cid(id), Action: combat.StandardActionDash})
		if rej := sumRejection(t, got); rej != (combat.CommandNotYourCreatureRejection{Creature: cid(id)}) {
			t.Errorf("FamiliarAction(%s) rejection = %#v, want CommandNotYourCreatureRejection [FR-048]", id, rej)
		}
	}
	if _, ok := w.FamiliarAction(combat.RequestFamiliarAction{Creature: cid("fam"), Action: combat.StandardActionDash}).(combat.SummonedApplied); !ok {
		t.Error("a familiar's Dash was refused [FR-048]")
	}
	for _, id := range []string{"fam", "u1", "sim"} {
		bond := combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid(id), Regained: 5, SpellLevel: 2, DistanceFeet: 5}
		if rej := sumRejection(t, w.LifeBond(bond)); rej != (combat.CommandNotYourCreatureRejection{Creature: cid(id)}) {
			t.Errorf("LifeBond(%s) rejection = %#v, want CommandNotYourCreatureRejection [FR-057]", id, rej)
		}
	}
	for _, id := range []string{"o1", "i1", "d1", "fam", "u1", "sim"} {
		if w.SteedTelepathy(cid(id)).IsTelepathic() {
			t.Errorf("SteedTelepathy(%s) is telepathic, but only a steed's is [FR-058]", id)
		}
		if rej := w.CheckSteedContact(cid(id), cid("zz")); rej != nil {
			t.Errorf("CheckSteedContact(%s) = %#v, want nil: the contact rule is for the steed [FR-058]", id, rej)
		}
	}
	if !w.SteedTelepathy(cid("st")).IsTelepathic() || w.CheckSteedContact(cid("st"), cid("zz")) == nil {
		t.Error("the steed lost its telepathy or its contact rule [FR-058]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-013, FR-040 - a standing order of an undead creature is ended by the task being complete and by nothing else: an undead creature with no order, a creature that is not undead and a creature the roster does not hold are unchanged, and an undead creature whose control lapsed has no turn plan even when it still holds an order.
// Path: unhappy (state error).
func TestMutgapBUndeadOrdersAndLapsedControl(t *testing.T) {
	t.Parallel()
	w := sumWorld(t)
	for _, id := range []string{"u1", "o1", "nobody"} {
		if _, ok := sumChange(t, w, combat.ReportTaskComplete{Creature: cid(id)}, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("TaskComplete(%s) changed the roster [FR-013]", id)
		}
	}
	ordered := sumApplied(t, sumChange(t, w, sumCommand(combat.SummonAnimatedUndead, 5, "u1"), "")).Roster()
	if _, ok := ordered.TurnPlan(cid("u1")); !ok {
		t.Fatal("an undead creature with an order has no turn plan [FR-013]")
	}
	lapsed := sumApplied(t, sumChange(t, ordered, combat.ElapseControlTime{Elapsed: 24 * time.Hour}, "")).Roster()
	if plan, ok := lapsed.TurnPlan(cid("u1")); ok {
		t.Errorf("TurnPlan(u1) after the control lapsed = %#v, want none [FR-040]", plan)
	}
}
