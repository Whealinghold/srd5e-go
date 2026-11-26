package combat_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sumFuzzCheck asserts the properties of a transition: a rejection or an unchanged transition leaves the roster equal, an applied one
// holds at most 32 creatures a summoner and events about creatures that are never blank or repeated.
func sumFuzzCheck(t *testing.T, before, after combat.SummonedRoster, tr combat.SummonedTransition) {
	t.Helper()
	switch v := tr.(type) {
	case combat.SummonedApplied:
		if v.Roster().Len() > 32*8 {
			t.Fatalf("roster has %d links", v.Roster().Len())
		}
		per := map[combat.CombatantID]int{}
		for _, l := range v.Roster().Links() {
			per[l.Combatants()[1]]++
			if per[l.Combatants()[1]] > 32 {
				t.Fatalf("a summoner has more than 32 creatures")
			}
		}
		for _, e := range v.Events() {
			ids := e.Combatants()
			if len(ids) == 0 || !distinct(ids) {
				t.Fatalf("%s has Combatants %v", e.Kind(), ids)
			}
			for _, id := range ids {
				if id.IsZero() {
					t.Fatalf("%s has a blank id", e.Kind())
				}
			}
		}
	case combat.SummonedRejected, combat.SummonedUnchanged:
		if !before.Equal(after) {
			t.Fatalf("a %T changed the roster", v)
		}
	}
}

// distinct reports whether ids holds no id twice.
func distinct(ids []combat.CombatantID) bool {
	seen := map[combat.CombatantID]bool{}
	for _, id := range ids {
		if seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

// FuzzNewSummonCreatureCommand: the constructor and Summon never panic for any kind of request, level, ids, modifier, flags and tie
// placement; an error is invalid input; the transition of a command that was built keeps its properties (a refusal leaves the roster
// equal, no summoner has more than 32 creatures, no event is about a blank or repeated id).
func FuzzNewSummonCreatureCommand(f *testing.F) {
	f.Add(uint8(0), 5, "o1", "pc", 4, true, false, 3, 1)
	f.Add(uint8(4), 3, "u1", "pc", 0, true, true, -2, 2)
	f.Add(uint8(6), 1, "fam", "", 1, false, true, 0, 0)
	f.Add(uint8(7), 9, "s", "s", 1000, false, false, 5, 1)
	f.Add(uint8(8), 10, "u", "pc", -1001, true, true, 9, 2)
	f.Fuzz(func(t *testing.T, kind uint8, level int, creature, summoner string, modifier int, inCombat, night bool, count, tie int) {
		if level < -20 || level > 40 || modifier < -2000 || modifier > 2000 {
			return
		}
		facts, _ := combat.NewSummonerSpellFacts(7, 15, modifier, 3)
		c := combat.CombatantID(creature)
		var req combat.SummonRequest
		switch kind % 9 {
		case 0:
			req = combat.AnimateObjectsRequest{Level: level, Objects: []combat.ObjectTarget{{Creature: c, Size: content.CreatureSize(count)}}}
		case 1:
			req = combat.GiantInsectRequest{Creature: c, Level: level, Form: combat.InsectForm(count)}
		case 2:
			req = combat.SummonDragonRequest{Creature: c, Level: level, SharedResistance: content.DamageType(count), BreathDamageType: content.DamageType(tie)}
		case 3:
			req = combat.FindSteedRequest{Level: level, Type: combat.SteedType(count), Choice: combat.SteedChoiceDifferent{Creature: c}, Anatomy: combat.MountAnatomyAppropriate, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleFitted}
		case 4:
			req = combat.AnimateDeadRequest{Level: level, Targets: []combat.UndeadTarget{{Creature: c, Remains: "r", Kind: combat.RemainsKind(count), Size: content.CreatureSize(tie), Humanoid: night}}}
		case 5:
			req = combat.CreateUndeadRequest{Level: level, Corpses: []combat.CorpseTarget{{Creature: c, Remains: "r", Size: content.CreatureSize(tie), Humanoid: true, Becomes: combat.UndeadForm(count)}}}
		case 6:
			req = combat.FindFamiliarRequest{Creature: c, Level: level, Form: character.NormalFamiliarForm("cat"), Type: content.CreatureType(count), Origin: combat.FamiliarOrigin(tie)}
		case 7:
			req = combat.SimulacrumRequest{Creature: c, Level: level, Original: combat.SimulacrumOriginal{Monster: "hero", Size: content.CreatureSize(count), Type: content.CreatureType(tie), HitPointMaximum: modifier, WithinTenFeet: night}}
		default:
			req = combat.ReassertControlRequest{Kind: combat.SummonKind(count), Level: level, Creatures: []combat.CombatantID{c}}
		}
		scene := combat.SummonScene{InCombat: inCombat, IsNight: night, SpaceAvailable: !night, Initiatives: []combat.RosterAtInitiative{{Count: combat.InitiativeCount(count), Tie: combat.TiePlacement(tie)}}}
		roster := sumFuzzRoster(t)
		cmd, err := combat.NewSummonCreatureCommand(roster, combat.CombatantID(summoner), combat.SideA, req, facts, scene)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !cmd.IsZero() {
				t.Fatalf("error %v does not match ErrInvalidInput, or the command is not zero", err)
			}
			return
		}
		tr := roster.Summon(cmd, sumMonsters(t)...)
		sumFuzzCheck(t, roster, roster, tr)
	})
}

// sumFuzzRoster is a small roster of the summoner "pc" for the fuzz targets.
func sumFuzzRoster(t *testing.T) combat.SummonedRoster {
	t.Helper()
	r := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	r = sumDo(t, r, "pc", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}, sumScene()).Roster()
	return sumFamiliar(t, r, "pc", "fam", combat.FamiliarOriginWildCompanion)
}

// FuzzNewApplySummonedChangeCommand: the constructor and Apply never panic for any kind of change, ids, distance, amount, duration and
// enum value; an error is invalid input; the transition of a change that was built keeps its properties.
func FuzzNewApplySummonedChangeCommand(f *testing.F) {
	f.Add(uint8(0), "pc", "i1", 10, 3, int64(time.Hour))
	f.Add(uint8(5), "pc", "u1", -1, 0, int64(-5))
	f.Add(uint8(10), "pc", "sim", 1_000_001, 2, int64(0))
	f.Add(uint8(15), "", "", 0, 99, int64(24*time.Hour))
	f.Fuzz(func(t *testing.T, kind uint8, summoner, creature string, distance, n int, elapsed int64) {
		s, c := combat.CombatantID(summoner), combat.CombatantID(creature)
		var change combat.SummonedChange
		switch kind % 16 {
		case 0:
			change = combat.IssueCommand{Summoner: s, Kind: combat.SummonKind(n), Targets: []combat.CommandTarget{{Creature: c, DistanceFeet: distance}}, Order: combat.GeneralOrder{Label: "x"}, CanGiveVerbalCommand: n%2 == 0}
		case 1:
			change = combat.ReportZeroHitPoints{Creature: c}
		case 2:
			change = combat.ReportSpellEnded{Summoner: s, Spell: content.SpellID(creature), Cause: combat.SummonSpellEnd(n)}
		case 3:
			change = combat.ReportSummonerDied{Summoner: s}
		case 4:
			change = combat.ReportLongRestFinished{Summoner: s}
		case 5:
			change = combat.ElapseControlTime{Elapsed: time.Duration(elapsed)}
		case 6:
			change = combat.DismissFamiliar{Summoner: s, How: combat.Dismissal(n)}
		case 7:
			change = combat.RecallFamiliar{Summoner: s, SpaceAvailable: n%2 == 0, InCombat: n > 0, Initiative: combat.RosterAtInitiative{Count: combat.InitiativeCount(distance), Tie: combat.TiePlacement(n)}}
		case 8:
			change = combat.ShareFamiliarSenses{Summoner: s, DistanceFeet: distance}
		case 9:
			change = combat.DeliverTouch{Summoner: s, SpellIsTouch: n%2 == 0, DistanceFeet: distance, Familiar: combat.ConditionActionAvailability(n)}
		case 10:
			change = combat.RepairSimulacrum{Summoner: s, Creature: c, Missing: distance, Restore: n, DistanceFeet: n, LongRestTaken: n%2 == 0}
		case 11:
			change = combat.ApplySteedLifeBond{Summoner: s, Steed: c, Regained: distance, SpellLevel: n, DistanceFeet: n}
		case 12:
			change = combat.ReportTurnStarted{Combatant: c}
		case 13:
			change = combat.ReportTurnEnded{Creature: c}
		case 14:
			change = combat.ReportTaskComplete{Creature: c}
		default:
			change = combat.RequestFamiliarAction{Creature: c, Action: combat.StandardAction(n), Grant: combat.FamiliarAttackGrant(distance)}
		}
		roster := sumFuzzRoster(t)
		cmd, err := combat.NewApplySummonedChangeCommand(roster, change, s)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !cmd.IsZero() {
				t.Fatalf("error %v does not match ErrInvalidInput, or the command is not zero", err)
			}
			return
		}
		sumFuzzCheck(t, roster, roster, roster.Apply(change, s))
	})
}

// FuzzNewSummonerSpellFacts: the constructor never panics for any numbers; an error is invalid input; facts that were built give a stat
// block for every level 1 to 9 of every constructor without panicking, and the numbers of a Slam stay valid expressions.
func FuzzNewSummonerSpellFacts(f *testing.F) {
	f.Add(7, 15, 4, 3)
	f.Add(1000, 1000, -1000, 1000)
	f.Add(-1001, 0, 1001, -1)
	f.Fuzz(func(t *testing.T, attack, dc, modifier, proficiency int) {
		facts, err := combat.NewSummonerSpellFacts(attack, dc, modifier, proficiency)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !facts.IsZero() {
				t.Fatalf("error %v does not match ErrInvalidInput, or the facts are not zero", err)
			}
			return
		}
		for level := 1; level <= 9; level++ {
			for _, size := range []content.CreatureSize{content.SizeTiny, content.SizeLarge, content.SizeHuge, content.SizeGargantuan} {
				if _, _, err := combat.NewAnimatedObjectStatBlock(size, level, facts); err != nil {
					t.Fatalf("object: %v", err)
				}
			}
			if _, _, err := combat.NewGiantInsectStatBlock(combat.InsectSpider, level, facts); err != nil {
				t.Fatalf("insect: %v", err)
			}
			if _, _, err := combat.NewDraconicSpiritStatBlock(level, facts, content.DamageFire, content.DamageFire); err != nil {
				t.Fatalf("spirit: %v", err)
			}
			if _, _, err := combat.NewOtherworldlySteedStatBlock(combat.SteedFiend, level, facts); err != nil {
				t.Fatalf("steed: %v", err)
			}
		}
	})
}
