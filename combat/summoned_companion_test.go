package combat_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// sumSteedWith casts Find Steed at level 2 with the choice, in or out of combat, for "pc".
func sumSteedWith(tb testing.TB, roster combat.SummonedRoster, choice combat.SteedChoice, scene combat.SummonScene) combat.SummonedTransition {
	tb.Helper()
	req := sumSteedRequest("unused", 2)
	req.Choice = choice
	return sumTry(tb, roster, "pc", req, scene)
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-056, Table R - Find Steed creates the steed in a space the host reports within 30 feet (SpaceUnavailableRejection when there is none, checked last), with the chosen type and the stat block of FR-034 at the cast level.
// Path: per case.
func TestFindSteedCreatesTheSteed(t *testing.T) {
	t.Parallel()
	scene := sumScene()
	noSpace := scene
	noSpace.SpaceAvailable = false
	a := sumApplied(t, sumSteedWith(t, combat.SummonedRoster{}, combat.SteedChoiceDifferent{Creature: cid("st")}, scene))
	if b := a.Creatures()[0].StatBlock(); b.HitPointMaximum() != 25 || b.Type() != content.CreatureTypeFey || b.ArmorClass() != 12 {
		t.Errorf("steed block = %+v, want the Fey steed of level 2 [FR-034, FR-056]", b)
	}
	if rej := sumRejection(t, sumSteedWith(t, combat.SummonedRoster{}, combat.SteedChoiceDifferent{Creature: cid("st")}, noSpace)); rej != (combat.SpaceUnavailableRejection{}) {
		t.Errorf("rejection = %#v, want SpaceUnavailableRejection [FR-056]", rej)
	}
	if rej := sumRejection(t, sumTry(t, combat.SummonedRoster{}, "pc", sumSteedRequest("st", 1), noSpace)); rej != (combat.LevelBelowSpellRejection{Minimum: 2}) {
		t.Errorf("rejection = %#v, want the level before the space [Table R]", rej)
	}
	if rej := sumRejection(t, sumSteedWith(t, combat.SummonedRoster{}, combat.SteedChoiceSame{Creature: cid("st")}, noSpace)); rej != (combat.SteedNotDisappearedRejection{}) {
		t.Errorf("rejection = %#v, want SteedNotDisappearedRejection before the space [Table R]", rej)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-056, Q-8 - casting Find Steed with a steed already linked replaces it: the old steed departs with cause Replaced (it drops what it carried, a SummonedRosterLeave of RemovalCauseDisappeared, pending when it is the active combatant) before the new one joins; another summoner's steed is untouched (SRD 5.2.1, "Find Steed", p. 130).
// Path: unhappy (rule interaction).
func TestANewSteedReplacesTheOld(t *testing.T) {
	t.Parallel()
	first := sumApplied(t, sumSteedWith(t, combat.SummonedRoster{}, combat.SteedChoiceDifferent{Creature: cid("st1")}, sumCombat(""))).Roster()
	other := sumDo(t, first, "other", sumSteedRequest("ost", 2), sumScene()).Roster()
	a := sumApplied(t, sumSteedWith(t, other, combat.SteedChoiceDifferent{Creature: cid("st2")}, sumCombat("st1")))
	if got := sumKinds(a.Events()); !slices.Equal(got, []string{"summoned_creature_departed", "summoned_creature_joined"}) {
		t.Fatalf("events = %v, want the old steed departing before the new one joins [FR-056]", got)
	}
	d := sumDeparted(a)[0]
	if d.Cause() != combat.SummonEndReplaced || !d.DropsCarriedItems() || !d.PendingRemoval() || d.SummonKind() != combat.SummonOtherworldlySteed {
		t.Errorf("departure = %#v, want Replaced dropping items, pending as the active combatant [FR-056, FR-077]", d)
	}
	want := []combat.SummonedRosterAction{
		combat.SummonedRosterLeave{Creature: cid("st1"), Cause: combat.RemovalCauseDisappeared, AfterTurn: true},
		combat.SummonedRosterJoin{Creature: cid("st2"), Position: combat.RosterAfterCombatant{Anchor: cid("pc")}, Reason: combat.JoinReasonSummoned},
	}
	if !reflect.DeepEqual(a.RosterActions(), want) {
		t.Errorf("actions = %v, want %v [FR-006, FR-056]", a.RosterActions(), want)
	}
	if _, ok := a.Roster().Link("st1"); ok {
		t.Errorf("the old steed is still linked")
	}
	if _, ok := a.Roster().Link("ost"); !ok || a.Roster().Len() != 2 {
		t.Errorf("another summoner's steed was touched, or the roster has %d links [FR-056]", a.Roster().Len())
	}
	if len(a.Creatures()) != 1 || a.Creatures()[0].Creature().ID != cid("st2") {
		t.Errorf("specs = %v, want only the new steed", a.Creatures())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-056, Q-8 - after a steed disappeared the cast takes SteedChoiceSame (the previous CombatantID) or a different steed (a new id); either begins at the full maximum Hit Points of the stat block rebuilt at the cast level; the same steed is refused (SteedNotDisappearedRejection) when none disappeared, when it is still linked or for another id, and a different steed cannot reuse the id of the one that disappeared.
// Path: per case.
func TestTheSameSteedOrADifferentOne(t *testing.T) {
	t.Parallel()
	linked := sumApplied(t, sumSteedWith(t, combat.SummonedRoster{}, combat.SteedChoiceDifferent{Creature: cid("st")}, sumScene())).Roster()
	killed := sumApplied(t, sumChange(t, linked, combat.ReportZeroHitPoints{Creature: cid("st")}, "")).Roster()
	diedWith := sumApplied(t, sumChange(t, linked, combat.ReportSummonerDied{Summoner: cid("pc")}, "")).Roster()
	same, diff := func(id string) combat.SteedChoice { return combat.SteedChoiceSame{Creature: cid(id)} }, func(id string) combat.SteedChoice { return combat.SteedChoiceDifferent{Creature: cid(id)} }
	cases := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		choice                  combat.SteedChoice
		want                    combat.SummonRejection
	}{
		{"the same steed after it dropped to 0", "FR-056", "happy", killed, same("st"), nil},
		{"the same steed after its summoner died", "FR-056", "happy", diedWith, same("st"), nil},
		{"a different steed after it disappeared", "FR-056", "happy", killed, diff("st2"), nil},
		{"the same steed when none disappeared", "FR-056", "unhappy (state error)", combat.SummonedRoster{}, same("st"), combat.SteedNotDisappearedRejection{}},
		{"the same steed that is still linked", "FR-008", "unhappy (state error)", linked, same("st"), combat.SummonCreatureIDInUseRejection{ID: cid("st")}},
		{"the same steed under another id while one is linked", "FR-056", "unhappy (state error)", linked, same("other"), combat.SteedNotDisappearedRejection{}},
		{"the same steed under another id", "FR-056", "unhappy (invalid input)", killed, same("other"), combat.SteedNotDisappearedRejection{}},
		{"a different steed under the old id", "FR-056", "unhappy (invalid input)", killed, diff("st"), combat.SummonCreatureIDInUseRejection{ID: cid("st")}},
		{"a different steed under a linked id", "FR-008", "unhappy (invalid input)", linked, diff("st"), combat.SummonCreatureIDInUseRejection{ID: cid("st")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumSteedWith(t, c.roster, c.choice, sumScene())
			if c.want != nil {
				if rej := sumRejection(t, got); rej != c.want {
					t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			if b := a.Creatures()[0].StatBlock(); b.HitPointMaximum() != 25 || len(sumDeparted(a)) != 0 {
				t.Errorf("steed = %d Hit Points, departures %v, want the full 25 and no departure [Q-8]", b.HitPointMaximum(), sumDeparted(a))
			}
			if _, ok := a.Roster().Link(steedCreature(c.choice)); !ok {
				t.Errorf("the steed %v is not linked", c.choice)
			}
		})
	}
}

// steedCreature is the id a steed choice names.
func steedCreature(c combat.SteedChoice) combat.CombatantID {
	switch v := c.(type) {
	case combat.SteedChoiceSame:
		return v.Creature
	case combat.SteedChoiceDifferent:
		return v.Creature
	}
	return ""
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-057, Table P - Life Bond gives the steed the same number of Hit Points as its summoner regained from a spell of level 1 or higher when the steed is within 5 feet (exactly 5 accepted, 6 refused with LifeBondOutOfReachRejection); a cantrip or a source that is not a spell gives 0 (SRD 5.2.1, "Otherworldly Steed", p. 131).
// Path: per case.
func TestLifeBond(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	cases := []struct {
		name, requirement, path string
		steed                   string
		regained, level, dist   int
		want                    int
		rejection               combat.SummonRejection
	}{
		{"a level 1 spell within 5 feet", "FR-057", "happy", "st", 7, 1, 5, 7, nil},
		{"a level 9 spell", "FR-057", "unhappy (boundary)", "st", 40, 9, 0, 40, nil},
		{"a cantrip", "FR-057", "unhappy (boundary)", "st", 5, 0, 5, 0, nil},
		{"a source that is not a spell", "FR-057", "unhappy (boundary)", "st", 5, 0, 0, 0, nil},
		{"nothing regained", "FR-057", "unhappy (empty case)", "st", 0, 3, 5, 0, nil},
		{"6 feet", "FR-057", "unhappy (boundary)", "st", 7, 1, 6, 0, combat.LifeBondOutOfReachRejection{}},
		{"a creature that is not a steed", "FR-057", "unhappy (invalid input)", "i1", 7, 1, 0, 0, combat.CommandNotYourCreatureRejection{Creature: cid("i1")}},
		{"an unknown steed", "FR-057", "unhappy (invalid input)", "ghost", 7, 1, 0, 0, combat.CommandNotYourCreatureRejection{Creature: cid("ghost")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumChange(t, roster, combat.ApplySteedLifeBond{Summoner: cid("pc"), Steed: cid(c.steed), Regained: c.regained, SpellLevel: c.level, DistanceFeet: c.dist}, "")
			if c.rejection != nil {
				if rej := sumRejection(t, got); rej != c.rejection {
					t.Errorf("rejection = %#v, want %#v [%s]", rej, c.rejection, c.requirement)
				}
				return
			}
			if a := sumApplied(t, got); a.LifeBondHitPoints() != c.want || !a.Roster().Equal(roster) || len(a.Events()) != 0 {
				t.Errorf("LifeBondHitPoints = %d, want %d with the roster and events untouched [%s]", a.LifeBondHitPoints(), c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-060, FR-061, Table P (simulacrum of a 1-Hit-Point original) - a simulacrum is refused (OriginalNotEligibleRejection) unless the original is a Beast or Humanoid within 10 feet for the casting; it is a Construct that cannot cast Simulacrum, gain levels or take rests, and its Hit Point maximum is half the original's rounded down with a minimum of 1 (SRD 5.2.1, "Simulacrum", p. 163; "Round Down", p. 187).
// Path: per case.
func TestSimulacrumOriginalsAndHitPoints(t *testing.T) {
	t.Parallel()
	mk := func(typ content.CreatureType, hp int, near bool) combat.SimulacrumRequest {
		q := sumSimulacrumRequest("s", 7)
		q.Original.Type, q.Original.HitPointMaximum, q.Original.WithinTenFeet = typ, hp, near
		return q
	}
	cases := []struct {
		name, requirement, path string
		request                 combat.SimulacrumRequest
		hp                      int
	}{
		{"a Humanoid of 20", "FR-060, p. 163", "happy", mk(content.CreatureTypeHumanoid, 20, true), 10},
		{"a Beast of 21", "FR-060, p. 187", "happy", mk(content.CreatureTypeBeast, 21, true), 10},
		{"an original of 1", "Table P, Q-16", "unhappy (boundary)", mk(content.CreatureTypeHumanoid, 1, true), 1},
		{"an original of 2", "Table P, Q-16", "unhappy (boundary)", mk(content.CreatureTypeHumanoid, 2, true), 1},
		{"an original of 3", "Table P, Q-16", "unhappy (boundary)", mk(content.CreatureTypeHumanoid, 3, true), 1},
		{"an original of 4", "Table P, Q-16", "unhappy (boundary)", mk(content.CreatureTypeHumanoid, 4, true), 2},
		{"a Dragon", "FR-060", "unhappy (invalid input)", mk(content.CreatureTypeDragon, 20, true), 0},
		{"an original too far away", "FR-060", "unhappy (state error)", mk(content.CreatureTypeHumanoid, 20, false), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumTry(t, combat.SummonedRoster{}, "pc", c.request, sumScene())
			if c.hp == 0 {
				if _, ok := sumRejection(t, got).(combat.OriginalNotEligibleRejection); !ok {
					t.Errorf("transition = %#v, want OriginalNotEligibleRejection [%s]", got, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			b := a.Creatures()[0].StatBlock()
			if b.HitPointMaximum() != c.hp || b.Type() != content.CreatureTypeConstruct || b.CanGainLevels() || b.CanTakeRests() || b.Size() != content.SizeMedium ||
				!slices.Equal(b.ForbiddenSpells(), []content.SpellID{"simulacrum"}) || b.ZeroHitPointsRule() != combat.DiesInstantly {
				t.Errorf("stat block = %+v, want %d Hit Points, a Construct that cannot cast Simulacrum, gain levels or rest [%s]", b, c.hp, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-063, FR-064, FR-008 - casting Simulacrum again first departs the summoner's own simulacrum (cause Replaced, reverting to snow); another summoner's is untouched; a simulacrum is friendly to its summoner and the creatures designated at the casting, and not to a stranger or itself (SRD 5.2.1, "Simulacrum", p. 163).
// Path: unhappy (rule interaction).
func TestACastOfSimulacrumReplacesOnlyTheCastersOwn(t *testing.T) {
	t.Parallel()
	r := sumDo(t, combat.SummonedRoster{}, "pc", sumSimulacrumRequest("s1", 7, "ally"), sumScene()).Roster()
	r = sumDo(t, r, "other", sumSimulacrumRequest("os", 7), sumScene()).Roster()
	a := sumDo(t, r, "pc", sumSimulacrumRequest("s2", 7), sumCombat("s1"))
	d := sumDeparted(a)
	if len(d) != 1 || d[0].Cause() != combat.SummonEndReplaced || !d[0].RevertsToSnow() || !d[0].PendingRemoval() || d[0].DropsCarriedItems() {
		t.Errorf("departures = %#v, want the old simulacrum Replaced, melting to snow, pending as the active combatant [FR-063]", d)
	}
	if _, ok := a.Roster().Link("os"); !ok || a.Roster().Len() != 2 {
		t.Errorf("another summoner's simulacrum was touched [FR-063]")
	}
	l, _ := r.Link("s1")
	for _, c := range []struct {
		id   string
		want bool
	}{{"pc", true}, {"ally", true}, {"stranger", false}, {"s1", false}} {
		if got := l.FriendlyTo(cid(c.id)); got != c.want {
			t.Errorf("FriendlyTo(%s) = %v, want %v [FR-064]", c.id, got, c.want)
		}
	}
	if fam, _ := sumWorld(t).Link("fam"); fam.FriendlyTo("pc") {
		t.Errorf("a familiar is recorded friendly: only a simulacrum is [FR-064]")
	}
	if got := l.DesignatedFriends(); !slices.Equal(got, []combat.CombatantID{"ally"}) {
		t.Errorf("DesignatedFriends = %v, want [ally]", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-062, Table R - a simulacrum's Hit Points are restored only by its repair, which needs a Long Rest, the simulacrum within 5 feet (exactly 5 accepted, 6 refused) and components worth 100 GP for each Hit Point restored (up to the missing ones, never 0); healing from any other source is refused (SRD 5.2.1, "Simulacrum", p. 163).
// Path: per case.
func TestRepairingASimulacrum(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	rep := func(creature string, missing, restore, d int, rest bool) combat.RepairSimulacrum {
		return combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid(creature), Missing: missing, Restore: restore, DistanceFeet: d, LongRestTaken: rest}
	}
	cases := []struct {
		name, requirement, path string
		change                  combat.RepairSimulacrum
		gold                    int
		want                    combat.SummonRejection
	}{
		{"all that is missing", "FR-062", "happy", rep("sim", 6, 6, 5, true), 600, nil},
		{"part of it", "FR-062", "happy", rep("sim", 6, 1, 0, true), 100, nil},
		{"the largest amount allowed", "FR-062, P10", "unhappy (boundary)", rep("sim", 1_000_000, 1_000_000, 5, true), 100_000_000, nil},
		{"6 feet", "FR-062", "unhappy (boundary)", rep("sim", 6, 6, 6, true), 0, combat.SimulacrumTooFarRejection{}},
		{"no Long Rest", "FR-062", "unhappy (state error)", rep("sim", 6, 6, 5, false), 0, combat.SimulacrumRepairNeedsLongRestRejection{}},
		{"no Long Rest before the distance", "Table R", "unhappy (rule interaction)", rep("sim", 6, 6, 50, false), 0, combat.SimulacrumRepairNeedsLongRestRejection{}},
		{"nothing restored", "FR-062", "unhappy (boundary)", rep("sim", 6, 0, 0, true), 0, combat.SimulacrumRepairAmountRejection{Missing: 6}},
		{"more than missing", "FR-062", "unhappy (boundary)", rep("sim", 6, 7, 0, true), 0, combat.SimulacrumRepairAmountRejection{Missing: 6}},
		{"nothing missing", "FR-062", "unhappy (boundary)", rep("sim", 0, 1, 0, true), 0, combat.SimulacrumRepairAmountRejection{Missing: 0}},
		{"a creature that is not a simulacrum", "FR-062", "unhappy (invalid input)", rep("i1", 6, 6, 0, true), 0, combat.CommandNotYourCreatureRejection{Creature: cid("i1")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumChange(t, roster, c.change, "")
			if c.want != nil {
				if rej := sumRejection(t, got); rej != c.want {
					t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			ev, ok := a.Events()[0].(combat.SimulacrumRepaired)
			if !ok || len(a.Events()) != 1 || ev.HitPointsRestored() != c.change.Restore || ev.GoldPieces() != c.gold || !slices.Equal(ev.Combatants(), []combat.CombatantID{"sim", "pc"}) || !a.Roster().Equal(roster) {
				t.Errorf("events = %#v, want SimulacrumRepaired of %d Hit Points and %d GP [%s]", a.Events(), c.change.Restore, c.gold, c.requirement)
			}
		})
	}
	if rej := roster.CheckHealing("sim"); rej != (combat.SimulacrumHealingRefusedRejection{}) {
		t.Errorf("CheckHealing(sim) = %#v, want SimulacrumHealingRefusedRejection [FR-062]", rej)
	}
	for _, id := range []string{"i1", "fam", "ghost"} {
		if rej := roster.CheckHealing(cid(id)); rej != nil {
			t.Errorf("CheckHealing(%s) = %#v, want none: only a simulacrum cannot be healed [FR-062]", id, rej)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008, Table P (simulacrum of a Humanoid character) - a simulacrum of a Humanoid character reaching 0 Hit Points departs, melting to snow, with no Death Saving Throws and no Dying state, because its stat block dies instantly whatever the original did.
// Path: unhappy (rule interaction).
func TestASimulacrumOfACharacterDiesInstantly(t *testing.T) {
	t.Parallel()
	a := sumDo(t, combat.SummonedRoster{}, "pc", sumSimulacrumRequest("s", 7), sumScene())
	if rule := a.Creatures()[0].StatBlock().ZeroHitPointsRule(); rule != combat.DiesInstantly {
		t.Fatalf("rule = %v, want DiesInstantly", rule)
	}
	d := sumDeparted(sumApplied(t, sumChange(t, a.Roster(), combat.ReportZeroHitPoints{Creature: cid("s")}, "")))
	if len(d) != 1 || !d[0].RevertsToSnow() || d[0].Cause() != combat.SummonEndDroppedToZeroHitPoints {
		t.Errorf("departures = %#v, want one departure reverting to snow [Table P]", d)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-018, FR-004 - the MountFacts of a steed (trained, willing, from the host's report on its anatomy, habitat and saddle) and its Large size let the rider mount it and choose the controlled mount, through MountCreature; a steed whose host reports no Exotic Saddle for an aquatic or flying habitat is refused as the mounting rules say, so the facts carry the host's report faithfully (SRD 5.2.1, "Find Steed", p. 130; "Controlling a Mount", p. 16).
// Path: per case.
func TestASteedsMountFactsLetItsRiderControlIt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		habitat                 combat.MountHabitat
		saddle                  combat.ExoticSaddle
		controlled              bool
	}{
		{"an ordinary steed", "FR-018", "happy", combat.MountHabitatOther, combat.ExoticSaddleNotFitted, true},
		{"a flying steed with an Exotic Saddle", "FR-018", "unhappy (rule interaction)", combat.MountHabitatAquaticOrFlying, combat.ExoticSaddleFitted, true},
		{"a flying steed without one", "FR-018", "unhappy (rule interaction)", combat.MountHabitatAquaticOrFlying, combat.ExoticSaddleNotFitted, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := sumSteedRequest("horse", 2)
			req.Habitat, req.Saddle = c.habitat, c.saddle
			spec := sumDo(t, combat.SummonedRoster{}, "rider", req, sumScene()).Creatures()[0]
			facts, ok := spec.MountFacts()
			if !ok {
				t.Fatal("the steed has no MountFacts")
			}
			in := mntInput(t)
			in.Facts, in.MountSize = facts, spec.StatBlock().Size()
			res, err := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in))
			if err != nil {
				t.Fatal(err)
			}
			_, applied := res.(combat.MountCreatureApplied)
			if applied != c.controlled {
				t.Errorf("MountCreature = %#v, want applied %v [%s]", res, c.controlled, c.requirement)
			}
		})
	}
}
