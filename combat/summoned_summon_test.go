package combat_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// sumObjects is a cast of Animate Objects at level 5 for one Medium object.
func sumObjects(ids ...string) combat.AnimateObjectsRequest {
	q := combat.AnimateObjectsRequest{Level: 5}
	for _, id := range ids {
		q.Objects = append(q.Objects, combat.ObjectTarget{Creature: cid(id), Size: content.SizeMedium})
	}
	return q
}

func sumSteedRequest(id string, level int) combat.FindSteedRequest {
	return combat.FindSteedRequest{Level: level, Type: combat.SteedFey, Choice: combat.SteedChoiceDifferent{Creature: cid(id)},
		Anatomy: combat.MountAnatomyAppropriate, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleNotFitted}
}

func sumSimulacrumRequest(id string, level int, friends ...string) combat.SimulacrumRequest {
	q := combat.SimulacrumRequest{Creature: cid(id), Level: level, Original: combat.SimulacrumOriginal{Monster: "hero", Size: content.SizeMedium,
		Type: content.CreatureTypeHumanoid, HitPointMaximum: 20, WithinTenFeet: true}}
	for _, f := range friends {
		q.Friends = append(q.Friends, cid(f))
	}
	return q
}

func sumBones(creature, remains string) combat.UndeadTarget {
	return combat.UndeadTarget{Creature: cid(creature), Remains: combat.RemainsID(remains), Kind: combat.RemainsBones, Size: content.SizeMedium}
}

func sumCorpse(creature, remains string, size content.CreatureSize, humanoid bool) combat.UndeadTarget {
	return combat.UndeadTarget{Creature: cid(creature), Remains: combat.RemainsID(remains), Kind: combat.RemainsCorpse, Size: size, Humanoid: humanoid}
}

func sumGhoul(creature, remains string, becomes combat.UndeadForm) combat.CorpseTarget {
	return combat.CorpseTarget{Creature: cid(creature), Remains: combat.RemainsID(remains), Size: content.SizeMedium, Humanoid: true, Becomes: becomes}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-001, FR-002, FR-008, Table K - the creatures that share their summoner's Initiative count are put on the summoner's side and joined directly after the summoner with JoinReasonSummoned, as an ally reported by SummonedCreatureJoined, with the CombatantRef of its monster id (SRD 5.2.1, "Animate Objects", p. 109; "Giant Insect", p. 135; "Summon Dragon", p. 166; "Find Steed", p. 130; "Simulacrum", p. 163).
// Path: per case.
func TestSharedCountKindsJoinAfterTheirSummoner(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		request                 combat.SummonRequest
		kind                    combat.SummonKind
		monster                 content.MonsterID
	}{
		{"animated object", "FR-002, Table K, p. 109", "happy", sumObjects("c1"), combat.SummonAnimatedObject, "animated_object"},
		{"giant insect", "FR-002, Table K, p. 135", "happy", combat.GiantInsectRequest{Creature: cid("c1"), Level: 4, Form: combat.InsectWasp}, combat.SummonGiantInsect, "giant_insect"},
		{"draconic spirit", "FR-002, Table K, p. 166", "happy", combat.SummonDragonRequest{Creature: cid("c1"), Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageFire}, combat.SummonDraconicSpirit, "draconic_spirit"},
		{"otherworldly steed", "FR-002, FR-004, Q-12, p. 130", "happy", sumSteedRequest("c1", 2), combat.SummonOtherworldlySteed, "otherworldly_steed"},
		{"simulacrum", "FR-002, Q-1, p. 163", "happy", sumSimulacrumRequest("c1", 7), combat.SummonSimulacrum, "hero"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := sumDo(t, combat.SummonedRoster{}, "pc", c.request, sumCombat(""))
			wantJoin := combat.SummonedRosterJoin{Creature: cid("c1"), Position: combat.RosterAfterCombatant{Anchor: cid("pc")}, Reason: combat.JoinReasonSummoned}
			if got := a.RosterActions(); !reflect.DeepEqual(got, []combat.SummonedRosterAction{wantJoin}) {
				t.Errorf("RosterActions = %#v, want %#v [%s]", got, wantJoin, c.requirement)
			}
			ev, ok := a.Events()[0].(combat.SummonedCreatureJoined)
			if !ok || len(a.Events()) != 1 || ev.Side() != combat.SideA || ev.SummonKind() != c.kind || ev.Placement() != combat.SharesSummonerCount ||
				!slices.Equal(ev.Combatants(), []combat.CombatantID{cid("c1"), cid("pc")}) {
				t.Errorf("events = %#v, want one SummonedCreatureJoined on side A of kind %v [%s]", a.Events(), c.kind, c.requirement)
			}
			spec := a.Creatures()[0]
			if spec.Creature() != (combat.CombatantRef{ID: cid("c1"), Kind: combat.MonsterCombatant{ID: c.monster}}) || spec.Side() != combat.SideA ||
				spec.SummonKind() != c.kind || spec.Placement() != combat.SharesSummonerCount {
				t.Errorf("spec = %#v, want a MonsterCombatant(%s) of side A [FR-008]", spec.Creature(), c.monster)
			}
			if l, ok := a.Roster().Link("c1"); !ok || l.SummonKind() != c.kind || l.Spell() != c.kind.Spell() || l.Side() != combat.SideA || l.CastLevel() < 1 {
				t.Errorf("link = %+v, want a link of kind %v [FR-001]", l, c.kind)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-003, Table K - the undead and the familiar roll their own Initiative: the joins are at the RosterAtInitiative the host built for each creature, in order, and the link keeps it.
// Path: per case.
func TestOwnCountKindsJoinAtTheHostsInitiative(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		request                 combat.SummonRequest
		inits                   []combat.RosterAtInitiative
		creatures               []string
		kind                    combat.SummonKind
	}{
		{"animate dead, one", "FR-003, p. 108", "happy", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}, []combat.RosterAtInitiative{sumInit(12)}, []string{"u1"}, combat.SummonAnimatedUndead},
		{"animate dead, three", "FR-003, p. 108", "unhappy (rule interaction)", combat.AnimateDeadRequest{Level: 4, Targets: []combat.UndeadTarget{sumBones("u1", "b1"), sumBones("u2", "b2"), sumBones("u3", "b3")}}, []combat.RosterAtInitiative{sumInit(12), sumInit(9), sumInit(-1)}, []string{"u1", "u2", "u3"}, combat.SummonAnimatedUndead},
		{"create undead", "FR-003, p. 121", "happy", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("u1", "c1", combat.UndeadGhoul)}}, []combat.RosterAtInitiative{sumInit(0)}, []string{"u1"}, combat.SummonCreatedUndead},
		{"find familiar", "FR-003, p. 130", "happy", combat.FindFamiliarRequest{Creature: cid("u1"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}, []combat.RosterAtInitiative{sumInit(17)}, []string{"u1"}, combat.SummonFamiliar},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := sumDo(t, combat.SummonedRoster{}, "pc", c.request, sumCombat("", c.inits...))
			var want []combat.SummonedRosterAction
			for i, id := range c.creatures {
				want = append(want, combat.SummonedRosterJoin{Creature: cid(id), Position: c.inits[i], Reason: combat.JoinReasonSummoned})
			}
			if got := a.RosterActions(); !reflect.DeepEqual(got, want) {
				t.Errorf("RosterActions = %#v, want %#v [%s]", got, want, c.requirement)
			}
			for i, id := range c.creatures {
				l, ok := a.Roster().Link(cid(id))
				if pos, has := l.RetainedPosition(); !ok || l.Placement() != combat.OwnCount || l.SummonKind() != c.kind || (c.kind == combat.SummonFamiliar && (!has || pos != c.inits[i])) {
					t.Errorf("link %s = %+v, want OwnCount of kind %v and, for a familiar, the retained position [FR-006]", id, l, c.kind)
				}
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-005 - a summons out of combat returns no roster instruction and still records the link with its PlacementRule, so the host places the creature when combat starts.
// Path: unhappy (empty case).
func TestASummonsOutOfCombatReturnsNoJoin(t *testing.T) {
	t.Parallel()
	shared := sumDo(t, combat.SummonedRoster{}, "pc", combat.GiantInsectRequest{Creature: cid("i1"), Level: 4, Form: combat.InsectSpider}, sumScene())
	own := sumDo(t, shared.Roster(), "pc", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}, sumScene())
	for _, c := range []struct {
		id   string
		a    combat.SummonedApplied
		want combat.PlacementRule
	}{{"i1", shared, combat.SharesSummonerCount}, {"u1", own, combat.OwnCount}} {
		l, _ := c.a.Roster().Link(cid(c.id))
		if len(c.a.RosterActions()) != 0 || l.Placement() != c.want {
			t.Errorf("%s: actions %v placement %v, want none and %v [FR-005]", c.id, c.a.RosterActions(), l.Placement(), c.want)
		}
	}
	if len(own.Events()) != 1 {
		t.Errorf("events = %v, want the join event even out of combat", sumKinds(own.Events()))
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-002, Table P (several summons, one summoner) - several creatures of one cast are joined in creation order, each directly after the summoner, and a second cast adds its own joins after them.
// Path: unhappy (rule interaction).
func TestSeveralCreaturesJoinInCreationOrder(t *testing.T) {
	t.Parallel()
	a := sumDo(t, combat.SummonedRoster{}, "pc", sumObjects("x", "y"), sumCombat(""))
	var ids []combat.CombatantID
	for _, act := range a.RosterActions() {
		j := act.(combat.SummonedRosterJoin)
		if j.Position != (combat.RosterAfterCombatant{Anchor: cid("pc")}) {
			t.Errorf("join %v position = %v, want after the summoner", j.Creature, j.Position)
		}
		ids = append(ids, j.Creature)
	}
	if !slices.Equal(ids, []combat.CombatantID{"x", "y"}) {
		t.Errorf("join order = %v, want x then y [FR-002, Table P]", ids)
	}
	if got := len(a.Creatures()); got != 2 {
		t.Errorf("specs = %d, want 2", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-004, FR-018 - a steed is independent while its summoner has Incapacitated and a controlled mount while ridden, in both with the same place; the roster supplies MountFacts for a trained, willing steed, from the host's report of its anatomy, habitat and saddle, and no other kind has them.
// Path: per case.
func TestSteedModeAndMountFacts(t *testing.T) {
	t.Parallel()
	if got := combat.SteedModeFor(combat.ConditionActionAvailabilityNone); got != combat.SteedModeIndependent {
		t.Errorf("SteedModeFor(none) = %v, want independent [FR-004]", got)
	}
	if got := combat.SteedModeFor(combat.ConditionActionAvailabilityAvailable); got != combat.SteedModeControlledWhileRidden {
		t.Errorf("SteedModeFor(available) = %v, want controlled_while_ridden [FR-004]", got)
	}
	req := sumSteedRequest("st", 2)
	req.Habitat, req.Saddle = combat.MountHabitatAquaticOrFlying, combat.ExoticSaddleFitted
	a := sumDo(t, combat.SummonedRoster{}, "pc", req, sumScene())
	want := luMust(combat.NewMountFacts(combat.MountWillingnessWilling, combat.MountAnatomyAppropriate, combat.MountTrainingTrainedToAcceptRider, combat.MountHabitatAquaticOrFlying, combat.ExoticSaddleFitted))
	if mf, ok := a.Creatures()[0].MountFacts(); !ok || mf != want {
		t.Errorf("MountFacts = %v, %v, want %v trained and willing [FR-018]", mf, ok, want)
	}
	if got := a.Creatures()[0].StatBlock().Size(); got != content.SizeLarge {
		t.Errorf("steed size = %v, want Large [FR-018]", got)
	}
	insect := sumDo(t, combat.SummonedRoster{}, "pc", combat.GiantInsectRequest{Creature: cid("i1"), Level: 4, Form: combat.InsectWasp}, sumScene())
	if _, ok := insect.Creatures()[0].MountFacts(); ok {
		t.Errorf("an insect has MountFacts [FR-018]")
	}
	if got := combat.SteedModeIndependent.String() + combat.SteedModeControlledWhileRidden.String() + combat.SteedMode(0).String(); got != "independentcontrolled_while_riddenunknown" {
		t.Errorf("SteedMode strings = %q", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008, plan P10 (a duplicate SummonedLink) - a CombatantID already linked is refused with SummonCreatureIDInUseRejection and the roster is unchanged; the id of a creature that was linked and departed may be used again.
// Path: unhappy (invalid input).
func TestADuplicateLinkIsRefused(t *testing.T) {
	t.Parallel()
	roster := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	before := roster.Links()
	rej := sumRejection(t, sumTry(t, roster, "pc", combat.GiantInsectRequest{Creature: cid("i1"), Level: 4, Form: combat.InsectWasp}, sumScene()))
	if r, ok := rej.(combat.SummonCreatureIDInUseRejection); !ok || r.ID != "i1" {
		t.Errorf("rejection = %#v, want SummonCreatureIDInUseRejection{i1} [FR-008]", rej)
	}
	other := sumRejection(t, sumTry(t, roster, "other", sumObjects("o1", "i1"), sumScene()))
	if _, ok := other.(combat.SummonCreatureIDInUseRejection); !ok {
		t.Errorf("a second summoner reusing the id got %#v, want SummonCreatureIDInUseRejection [FR-008]", other)
	}
	if !slices.EqualFunc(roster.Links(), before, combat.SummonedLink.Equal) {
		t.Errorf("a refused summons changed the roster [FR-065]")
	}
	gone := sumApplied(t, sumChange(t, roster, combat.ReportZeroHitPoints{Creature: cid("i1")}, "")).Roster()
	if _, again := sumTry(t, gone, "pc", combat.GiantInsectRequest{Creature: cid("i1"), Level: 4, Form: combat.InsectWasp}, sumScene()).(combat.SummonedApplied); !again {
		t.Errorf("the id of a departed creature cannot be used again")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md plan P10 (link cycles) - a link that would close a cycle (A summons B, then B summons A; or a longer ring) is refused with SummonLinkCycleRejection, found by walking the chain iteratively with a visited set, so a chain of hundreds of summoners ends at once; a chain that is not a ring is accepted.
// Path: per case.
func TestALinkCycleIsRejectedIteratively(t *testing.T) {
	t.Parallel()
	chain := func(n int) combat.SummonedRoster {
		r := combat.SummonedRoster{}
		for i := 0; i < n; i++ {
			r = sumInsect(t, r, sumLabel("s", i), sumLabel("s", i+1))
		}
		return r
	}
	cases := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		summoner, creature      string
		cycle                   bool
	}{
		{"A summons B summons A", "P10 link cycles", "unhappy (rule interaction)", chain(1), "s01", "s00", true},
		{"a ring of three", "P10 link cycles", "unhappy (rule interaction)", chain(2), "s02", "s00", true},
		{"a chain of 200 closed at the top", "P10 iterative walk", "unhappy (boundary)", chain(200), "s200", "s00", true},
		{"a chain of 200 extended", "P10 iterative walk", "happy", chain(200), "s200", "next", false},
		{"B summons a stranger", "P10 link cycles", "happy", chain(1), "s01", "z", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumTry(t, c.roster, c.summoner, combat.GiantInsectRequest{Creature: cid(c.creature), Level: 4, Form: combat.InsectWasp}, sumScene())
			if c.cycle {
				if r, ok := sumRejection(t, got).(combat.SummonLinkCycleRejection); !ok || r.Creature != cid(c.creature) || r.Summoner != cid(c.summoner) {
					t.Errorf("rejection = %#v, want SummonLinkCycleRejection{%s, %s} [%s]", got, c.creature, c.summoner, c.requirement)
				}
				return
			}
			if _, ok := got.(combat.SummonedApplied); !ok {
				t.Errorf("transition = %#v, want applied [%s]", got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md Table R, plan P10 - a creature that is already linked is refused as IdInUse before any cycle is looked for: a creature of the middle of a chain cannot be summoned again by the end of it.
// Path: unhappy (rule interaction).
func TestAnIdInUseComesBeforeACycle(t *testing.T) {
	t.Parallel()
	r := sumInsect(t, sumInsect(t, combat.SummonedRoster{}, "s0", "s1"), "s1", "s2")
	got := sumTry(t, r, "s2", combat.GiantInsectRequest{Creature: cid("s1"), Level: 4, Form: combat.InsectWasp}, sumScene())
	if _, ok := sumRejection(t, got).(combat.SummonCreatureIDInUseRejection); !ok {
		t.Errorf("rejection = %#v, want SummonCreatureIDInUseRejection first [Table R]", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md plan P10 (summons per summoner), FR-067 - a summoner holds at most 32 summoned creatures: the 32nd is accepted and the 33rd refused with SummonLimitExceededRejection (class invalid), for one cast of many creatures and for casts one at a time; another summoner is unaffected and a departure makes room.
// Path: per case.
func TestSummonsPerSummonerAreCapped(t *testing.T) {
	t.Parallel()
	many := func(n int) combat.AnimateObjectsRequest {
		q := combat.AnimateObjectsRequest{Level: 5}
		for i := 0; i < n; i++ {
			q.Objects = append(q.Objects, combat.ObjectTarget{Creature: cid(sumLabel("o", i)), Size: content.SizeMedium})
		}
		return q
	}
	big := luMust(combat.NewSummonerSpellFacts(7, 15, 40, 3))
	cast := func(r combat.SummonedRoster, summoner string, n int) combat.SummonedTransition {
		cmd, err := combat.NewSummonCreatureCommand(r, cid(summoner), combat.SideA, many(n), big, sumScene())
		if err != nil {
			t.Fatal(err)
		}
		return r.Summon(cmd)
	}
	full := sumApplied(t, cast(combat.SummonedRoster{}, "pc", 32)).Roster()
	cases := []struct {
		name, requirement, path string
		got                     combat.SummonedTransition
		accepted                bool
	}{
		{"one cast of 32", "P10 at the cap", "unhappy (boundary)", cast(combat.SummonedRoster{}, "pc", 32), true},
		{"one cast of 33", "P10 cap plus one", "unhappy (boundary)", cast(combat.SummonedRoster{}, "pc", 33), false},
		{"a 33rd after 32", "P10 cap plus one", "unhappy (boundary)", sumTry(t, full, "pc", combat.GiantInsectRequest{Creature: cid("extra"), Level: 4, Form: combat.InsectWasp}, sumScene()), false},
		{"another summoner after 32", "P10 per summoner", "happy", sumTry(t, full, "other", combat.GiantInsectRequest{Creature: cid("extra"), Level: 4, Form: combat.InsectWasp}, sumScene()), true},
		{"a 32nd after 31", "P10 at the cap", "unhappy (boundary)", sumTry(t, sumApplied(t, cast(combat.SummonedRoster{}, "pc", 31)).Roster(), "pc", combat.GiantInsectRequest{Creature: cid("extra"), Level: 4, Form: combat.InsectWasp}, sumScene()), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.accepted {
				if _, ok := c.got.(combat.SummonedApplied); !ok {
					t.Errorf("transition = %#v, want applied [%s]", c.got, c.requirement)
				}
				return
			}
			if _, ok := sumRejection(t, c.got).(combat.SummonLimitExceededRejection); !ok {
				t.Errorf("rejection = %#v, want SummonLimitExceededRejection [%s]", c.got, c.requirement)
			}
		})
	}
	room := sumApplied(t, sumChange(t, full, combat.ReportZeroHitPoints{Creature: cid("o00")}, "")).Roster()
	if _, ok := sumTry(t, room, "pc", combat.GiantInsectRequest{Creature: cid("extra"), Level: 4, Form: combat.InsectWasp}, sumScene()).(combat.SummonedApplied); !ok {
		t.Errorf("a departure did not make room for the 32nd [P10]")
	}
	if got := (combat.SummonLimitExceededRejection{}).String(); got != "limit_exceeded" {
		t.Errorf("limit string = %q, want limit_exceeded", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-023, FR-053 and plan P5.12 (a summoned creature outliving its summoner) - a creature outlives its summoner when the summoner dies or departs, and when its own summoner is a summoned creature that departed: only a steed goes with its summoner's death, a creature summoned by a departed creature stays linked and can still be commanded, and its own departure is still reported.
// Path: unhappy (rule interaction).
func TestASummonedCreatureOutlivesItsSummoner(t *testing.T) {
	t.Parallel()
	r := sumInsect(t, combat.SummonedRoster{}, "pc", "mid")
	r = sumDo(t, r, "pc", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("sk", "b1")}}, sumScene()).Roster()
	r = sumDo(t, r, "pc", sumSimulacrumRequest("sim", 7), sumScene()).Roster()
	r = sumFamiliar(t, r, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
	r = sumInsect(t, r, "mid", "child")
	died := sumChange(t, r, combat.ReportSummonerDied{Summoner: cid("pc")}, "")
	if _, ok := died.(combat.SummonedUnchanged); !ok {
		t.Fatalf("the summoner dying changed the roster with no steed: %#v", died)
	}
	r = sumApplied(t, sumChange(t, r, combat.ReportZeroHitPoints{Creature: cid("mid")}, "")).Roster()
	if _, ok := r.Link("child"); !ok || r.Len() != 4 {
		t.Fatalf("roster has %d links and child %v, want the child to outlive the creature that summoned it", r.Len(), ok)
	}
	cmd := combat.IssueCommand{Summoner: cid("mid"), Kind: combat.SummonGiantInsect, Targets: []combat.CommandTarget{{Creature: cid("child")}}, Order: combat.GeneralOrder{Label: "guard"}, CanGiveVerbalCommand: true}
	if _, ok := sumChange(t, r, cmd, "").(combat.SummonedApplied); !ok {
		t.Errorf("the orphaned creature cannot be commanded by the summoner that departed")
	}
	if _, ok := sumChange(t, r, combat.ReportZeroHitPoints{Creature: cid("child")}, "").(combat.SummonedApplied); !ok {
		t.Errorf("the orphaned creature's own departure is not reported")
	}
	withSteed := sumDo(t, r, "pc", sumSteedRequest("st", 2), sumScene()).Roster()
	if a := sumApplied(t, sumChange(t, withSteed, combat.ReportSummonerDied{Summoner: cid("pc")}, "")); a.Roster().Len() != withSteed.Len()-1 {
		t.Errorf("the summoner's death left %d links of %d, want only the steed gone", a.Roster().Len(), withSteed.Len())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-035, Table S rows 11, 18, 19, Table U - a cast level below the spell's own is refused with LevelBelowSpellRejection naming the spell's level (Animate Dead 3, Animate Objects 5, Create Undead 6, Find Steed 2, Giant Insect 4, Simulacrum 7, Summon Dragon 5), the level itself is accepted, and Find Familiar and Simulacrum take the base numbers at a higher level.
// Path: per case.
func TestACastBelowTheSpellsLevelIsRefused(t *testing.T) {
	t.Parallel()
	req := func(spell string, level int) combat.SummonRequest {
		switch spell {
		case "objects":
			q := sumObjects("c1")
			q.Level = level
			return q
		case "insect":
			return combat.GiantInsectRequest{Creature: cid("c1"), Level: level, Form: combat.InsectWasp}
		case "dragon":
			return combat.SummonDragonRequest{Creature: cid("c1"), Level: level, SharedResistance: content.DamageCold, BreathDamageType: content.DamageCold}
		case "steed":
			return sumSteedRequest("c1", level)
		case "dead":
			return combat.AnimateDeadRequest{Level: level, Targets: []combat.UndeadTarget{sumBones("c1", "b1")}}
		case "undead":
			return combat.CreateUndeadRequest{Level: level, Corpses: []combat.CorpseTarget{sumGhoul("c1", "r1", combat.UndeadGhoul)}}
		case "familiar":
			return combat.FindFamiliarRequest{Creature: cid("c1"), Level: level, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}
		}
		return sumSimulacrumRequest("c1", level)
	}
	cases := []struct {
		name, spell string
		level, min  int
		pages       string
	}{
		{"Animate Dead 2", "dead", 2, 3, "p. 108"}, {"Animate Objects 4", "objects", 4, 5, "p. 108"}, {"Create Undead 5", "undead", 5, 6, "p. 121"},
		{"Find Steed 1", "steed", 1, 2, "p. 130"}, {"Giant Insect 3", "insect", 3, 4, "p. 135"}, {"Simulacrum 6", "simulacrum", 6, 7, "p. 162"},
		{"Summon Dragon 4", "dragon", 4, 5, "p. 166"},
	}
	for _, c := range cases {
		t.Run(c.name+" is refused", func(t *testing.T) {
			t.Parallel()
			rej := sumRejection(t, sumTry(t, combat.SummonedRoster{}, "pc", req(c.spell, c.level), sumScene()))
			if r, ok := rej.(combat.LevelBelowSpellRejection); !ok || r.Minimum != c.min {
				t.Errorf("rejection = %#v, want LevelBelowSpellRejection{%d} [FR-035, %s]", rej, c.min, c.pages)
			}
		})
		t.Run(c.name+" plus one is accepted", func(t *testing.T) {
			t.Parallel()
			if _, ok := sumTry(t, combat.SummonedRoster{}, "pc", req(c.spell, c.min), sumScene()).(combat.SummonedApplied); !ok {
				t.Errorf("a cast at the spell's own level %d was refused [FR-035, %s]", c.min, c.pages)
			}
		})
	}
	for _, spell := range []string{"familiar", "simulacrum"} {
		level := 9
		if _, ok := sumTry(t, combat.SummonedRoster{}, "pc", req(spell, level), sumScene()).(combat.SummonedApplied); !ok {
			t.Errorf("%s at level 9 was refused: it has no higher-level text and takes the base numbers [FR-035, Table S 19]", spell)
		}
	}
	a := sumDo(t, combat.SummonedRoster{}, "pc", req("simulacrum", 9), sumScene())
	if a.Creatures()[0].StatBlock().HitPointMaximum() != 10 {
		t.Errorf("a level 9 simulacrum has %d Hit Points, want the base half of 20", a.Creatures()[0].StatBlock().HitPointMaximum())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-029, FR-031 - Animate Objects accepts objects whose weights (Medium or smaller 1, Large 2, Huge 3) sum to at most the summoner's spellcasting ability modifier (exactly the modifier accepted, one over refused with the modifier as the maximum, a modifier below 1 allows none), and refuses a magical, worn or carried, fixed or Gargantuan object, and an empty cast (SRD 5.2.1, "Animate Objects", pp. 108-109).
// Path: per case.
func TestAnimateObjectsWeightsAndEligibility(t *testing.T) {
	t.Parallel()
	obj := func(id string, size content.CreatureSize) combat.ObjectTarget {
		return combat.ObjectTarget{Creature: cid(id), Size: size}
	}
	cases := []struct {
		name, requirement, path string
		modifier                int
		objects                 []combat.ObjectTarget
		want                    any
	}{
		{"weights equal the modifier", "FR-031, p. 108", "unhappy (boundary)", 4, []combat.ObjectTarget{obj("a", content.SizeHuge), obj("b", content.SizeSmall)}, "applied"},
		{"weights one over the modifier", "FR-031, p. 108", "unhappy (boundary)", 3, []combat.ObjectTarget{obj("a", content.SizeHuge), obj("b", content.SizeSmall)}, combat.TooManyCreaturesRejection{Maximum: 3}},
		{"a Large is worth 2", "FR-031, p. 109", "unhappy (boundary)", 2, []combat.ObjectTarget{obj("a", content.SizeLarge)}, "applied"},
		{"a modifier of 0 allows none", "FR-031, p. 108", "unhappy (boundary)", 0, []combat.ObjectTarget{obj("a", content.SizeTiny)}, combat.TooManyCreaturesRejection{Maximum: 0}},
		{"a negative modifier allows none", "FR-031, p. 108", "unhappy (boundary)", -1, []combat.ObjectTarget{obj("a", content.SizeTiny)}, combat.TooManyCreaturesRejection{Maximum: 0}},
		{"a magical object", "FR-031, p. 108", "unhappy (invalid input)", 4, []combat.ObjectTarget{{Creature: cid("a"), Size: content.SizeSmall, Magical: true}}, combat.ObjectNotEligibleRejection{Creature: cid("a")}},
		{"a worn or carried object", "FR-031, p. 108", "unhappy (invalid input)", 4, []combat.ObjectTarget{obj("a", content.SizeSmall), {Creature: cid("b"), Size: content.SizeSmall, WornOrCarried: true}}, combat.ObjectNotEligibleRejection{Creature: cid("b")}},
		{"a fixed object", "FR-031, p. 108", "unhappy (invalid input)", 4, []combat.ObjectTarget{{Creature: cid("a"), Size: content.SizeSmall, FixedToSurface: true}}, combat.ObjectNotEligibleRejection{Creature: cid("a")}},
		{"a Gargantuan object", "FR-029, Table S 7", "unhappy (invalid input)", 9, []combat.ObjectTarget{obj("a", content.SizeGargantuan)}, combat.ObjectNotEligibleRejection{Creature: cid("a")}},
		{"an object that is not eligible comes before the weights", "Table R", "unhappy (rule interaction)", 1, []combat.ObjectTarget{obj("a", content.SizeHuge), {Creature: cid("b"), Size: content.SizeSmall, Magical: true}}, combat.ObjectNotEligibleRejection{Creature: cid("b")}},
		{"no object", "FR-042, Table P", "unhappy (empty case)", 4, nil, combat.NothingToAnimateRejection{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			facts := luMust(combat.NewSummonerSpellFacts(7, 15, c.modifier, 3))
			cmd, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "pc", combat.SideA, combat.AnimateObjectsRequest{Level: 5, Objects: c.objects}, facts, sumScene())
			if err != nil {
				t.Fatal(err)
			}
			got := combat.SummonedRoster{}.Summon(cmd)
			if c.want == "applied" {
				if _, ok := got.(combat.SummonedApplied); !ok {
					t.Errorf("transition = %#v, want applied [%s]", got, c.requirement)
				}
				return
			}
			if rej := sumRejection(t, got); rej != c.want {
				t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-038, Table U (Animate Dead L3, L4, L9) - Animate Dead turns bones into a Skeleton and the corpse of a Medium or Small Humanoid into a Zombie, from different piles or corpses, up to 1 creature plus 2 for each level above 3; a corpse that is Large, not a Humanoid or Tiny, a repeated pile and too many creatures are refused (SRD 5.2.1, "Animate Dead", p. 108).
// Path: per case.
func TestAnimateDeadTargetsAndCounts(t *testing.T) {
	t.Parallel()
	n := func(level int, count int) combat.AnimateDeadRequest {
		q := combat.AnimateDeadRequest{Level: level}
		for i := 0; i < count; i++ {
			q.Targets = append(q.Targets, sumBones(sumLabel("u", i), sumLabel("b", i)))
		}
		return q
	}
	cases := []struct {
		name, requirement, path string
		request                 combat.AnimateDeadRequest
		want                    any
	}{
		{"L3 one", "Table U, p. 108", "happy", n(3, 1), "applied"},
		{"L3 two", "Table U, p. 108", "unhappy (boundary)", n(3, 2), combat.TooManyCreaturesRejection{Maximum: 1}},
		{"L4 three", "Table U, p. 108", "unhappy (boundary)", n(4, 3), "applied"},
		{"L4 four", "Table U, p. 108", "unhappy (boundary)", n(4, 4), combat.TooManyCreaturesRejection{Maximum: 3}},
		{"L9 thirteen", "Table U, p. 108", "unhappy (boundary)", n(9, 13), "applied"},
		{"L9 fourteen", "Table U, p. 108", "unhappy (boundary)", n(9, 14), combat.TooManyCreaturesRejection{Maximum: 13}},
		{"a Medium Humanoid corpse", "FR-038, p. 108", "happy", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u0", "r0", content.SizeMedium, true)}}, "applied"},
		{"a Small Humanoid corpse", "FR-038, p. 108", "unhappy (boundary)", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u0", "r0", content.SizeSmall, true)}}, "applied"},
		{"a Large corpse", "FR-038, p. 108", "unhappy (boundary)", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u0", "r0", content.SizeLarge, true)}}, combat.TargetNotEligibleRejection{Creature: cid("u0")}},
		{"a Tiny corpse", "FR-038, p. 108", "unhappy (boundary)", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u0", "r0", content.SizeTiny, true)}}, combat.TargetNotEligibleRejection{Creature: cid("u0")}},
		{"a corpse that is not a Humanoid", "FR-038, p. 108", "unhappy (invalid input)", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u0", "r0", content.SizeMedium, false)}}, combat.TargetNotEligibleRejection{Creature: cid("u0")}},
		{"the same pile twice", "FR-038, p. 108", "unhappy (invalid input)", combat.AnimateDeadRequest{Level: 4, Targets: []combat.UndeadTarget{sumBones("u0", "b0"), sumBones("u1", "b0")}}, combat.DuplicateTargetRejection{Creature: cid("u1")}},
		{"no target", "FR-042, Table P", "unhappy (empty case)", combat.AnimateDeadRequest{Level: 3}, combat.NothingToAnimateRejection{}},
		{"an ineligible target comes before a repeated pile", "Table R", "unhappy (rule interaction)", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u0", "r0", content.SizeLarge, true), sumBones("u1", "r0")}}, combat.TargetNotEligibleRejection{Creature: cid("u0")}},
		{"a repeated pile comes before too many", "Table R", "unhappy (rule interaction)", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u0", "r0"), sumBones("u1", "r0")}}, combat.DuplicateTargetRejection{Creature: cid("u1")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumTry(t, combat.SummonedRoster{}, "pc", c.request, sumScene())
			if c.want == "applied" {
				a, ok := got.(combat.SummonedApplied)
				if !ok {
					t.Fatalf("transition = %#v, want applied [%s]", got, c.requirement)
				}
				for i, tg := range c.request.Targets {
					want := content.MonsterID("skeleton")
					if tg.Kind == combat.RemainsCorpse {
						want = "zombie"
					}
					if a.Creatures()[i].StatBlock().Monster() != want {
						t.Errorf("creature %d is a %v, want a %v [FR-038]", i, a.Creatures()[i].StatBlock().Monster(), want)
					}
				}
				return
			}
			if rej := sumRejection(t, got); rej != c.want {
				t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-039, Table U (Create Undead L6 to L9, below 6, mixed rows) - Create Undead needs night and corpses of Medium or Small Humanoids from different corpses, makes the creatures of one composition row (Ghouls; Ghasts or Wights; Mummies) allowed at the level, and refuses too many (3 Ghouls at L6 up to 6 at L9, 2 Ghasts or Wights at L8, 3 at L9, 2 Mummies at L9) and a mixture (SRD 5.2.1, "Create Undead", p. 121).
// Path: per case.
func TestCreateUndeadRowsAndNight(t *testing.T) {
	t.Parallel()
	mk := func(level int, forms ...combat.UndeadForm) combat.CreateUndeadRequest {
		q := combat.CreateUndeadRequest{Level: level}
		for i, f := range forms {
			q.Corpses = append(q.Corpses, sumGhoul(sumLabel("u", i), sumLabel("r", i), f))
		}
		return q
	}
	g, a, w, m := combat.UndeadGhoul, combat.UndeadGhast, combat.UndeadWight, combat.UndeadMummy
	cases := []struct {
		name, requirement, path string
		request                 combat.CreateUndeadRequest
		night                   bool
		want                    any
	}{
		{"L6 three Ghouls", "Table U L6, p. 121", "happy", mk(6, g, g, g), true, "applied"},
		{"L6 four Ghouls", "Table U L6, p. 121", "unhappy (boundary)", mk(6, g, g, g, g), true, combat.TooManyCreaturesRejection{Maximum: 3}},
		{"L6 a Wight", "Table U L6, p. 121", "unhappy (rule interaction)", mk(6, w), true, combat.CompositionNotAllowedRejection{}},
		{"L7 four Ghouls", "Table U L7, p. 121", "unhappy (boundary)", mk(7, g, g, g, g), true, "applied"},
		{"L7 five Ghouls", "Table U L7, p. 121", "unhappy (boundary)", mk(7, g, g, g, g, g), true, combat.TooManyCreaturesRejection{Maximum: 4}},
		{"L8 five Ghouls", "Table U L8, p. 121", "unhappy (rule interaction)", mk(8, g, g, g, g, g), true, "applied"},
		{"L8 two Ghasts or Wights", "Table U L8, p. 121", "unhappy (rule interaction)", mk(8, a, w), true, "applied"},
		{"L8 three Wights", "Table U L8, p. 121", "unhappy (rule interaction)", mk(8, w, w, w), true, combat.TooManyCreaturesRejection{Maximum: 2}},
		{"L8 a Mummy", "Table U L8, p. 121", "unhappy (rule interaction)", mk(8, m), true, combat.CompositionNotAllowedRejection{}},
		{"L8 mixed Ghoul and Wight", "Table U mixed rows, p. 121", "unhappy (rule interaction)", mk(8, g, w), true, combat.CompositionNotAllowedRejection{}},
		{"L9 six Ghouls", "Table U L9, p. 121", "unhappy (rule interaction)", mk(9, g, g, g, g, g, g), true, "applied"},
		{"L9 three Ghasts or Wights", "Table U L9, p. 121", "unhappy (rule interaction)", mk(9, a, w, a), true, "applied"},
		{"L9 two Mummies", "Table U L9, p. 121", "unhappy (rule interaction)", mk(9, m, m), true, "applied"},
		{"L9 three Mummies", "Table U L9, p. 121", "unhappy (rule interaction)", mk(9, m, m, m), true, combat.TooManyCreaturesRejection{Maximum: 2}},
		{"L9 seven creatures", "Table R", "unhappy (rule interaction)", mk(9, g, g, g, g, g, g, g), true, combat.TooManyCreaturesRejection{Maximum: 6}},
		{"too many and mixed: too many first", "Table R", "unhappy (rule interaction)", mk(8, g, g, g, g, w, w), true, combat.TooManyCreaturesRejection{Maximum: 5}},
		{"by day", "FR-039, p. 121", "unhappy (state error)", mk(6, g), false, combat.NotNightRejection{}},
		{"below level 6", "Table U below 6, p. 121", "unhappy (boundary)", mk(5, g), true, combat.LevelBelowSpellRejection{Minimum: 6}},
		{"the level comes before night", "Table R", "unhappy (rule interaction)", mk(5, g), false, combat.LevelBelowSpellRejection{Minimum: 6}},
		{"night comes before an ineligible corpse", "Table R", "unhappy (rule interaction)", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{{Creature: cid("u0"), Remains: "r0", Size: content.SizeLarge, Humanoid: true, Becomes: g}}}, false, combat.NotNightRejection{}},
		{"a Large corpse", "FR-039, p. 121", "unhappy (invalid input)", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{{Creature: cid("u0"), Remains: "r0", Size: content.SizeLarge, Humanoid: true, Becomes: g}}}, true, combat.TargetNotEligibleRejection{Creature: cid("u0")}},
		{"a corpse that is not a Humanoid", "FR-039, p. 121", "unhappy (invalid input)", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{{Creature: cid("u0"), Remains: "r0", Size: content.SizeSmall, Becomes: g}}}, true, combat.TargetNotEligibleRejection{Creature: cid("u0")}},
		{"the same corpse twice", "FR-039, p. 121", "unhappy (invalid input)", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("u0", "r0", g), sumGhoul("u1", "r0", g)}}, true, combat.DuplicateTargetRejection{Creature: cid("u1")}},
		{"no corpse", "FR-042, Table P", "unhappy (empty case)", combat.CreateUndeadRequest{Level: 6}, true, combat.NothingToAnimateRejection{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			scene := sumScene()
			scene.IsNight = c.night
			got := sumTry(t, combat.SummonedRoster{}, "pc", c.request, scene)
			if c.want == "applied" {
				ap, ok := got.(combat.SummonedApplied)
				if !ok {
					t.Fatalf("transition = %#v, want applied [%s]", got, c.requirement)
				}
				for i, tg := range c.request.Corpses {
					l, _ := ap.Roster().Link(tg.Creature)
					if ap.Creatures()[i].StatBlock().Monster() != content.MonsterID(tg.Becomes.String()) || l.SummonKind() != combat.SummonCreatedUndead {
						t.Errorf("creature %d is %v, want %v [FR-039]", i, ap.Creatures()[i].StatBlock().Monster(), tg.Becomes)
					}
					if f, ok := l.UndeadForm(); !ok || f != tg.Becomes {
						t.Errorf("link form = %v, want %v", f, tg.Becomes)
					}
				}
				return
			}
			if rej := sumRejection(t, got); rej != c.want {
				t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-007 - a summoned monster's XP is the XP of its stat block: 0 for the four expression stat blocks and the Otherworldly Steed (Challenge Rating None), the printed XP of a Skeleton, Zombie, Ghoul, Ghast, Wight, Mummy and familiar form (SRD 5.2.1, "Experience Points", p. 256).
// Path: per case.
func TestSummonedExperiencePoints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		request combat.SummonRequest
		xp      int
	}{
		{"object", sumObjects("c"), 0},
		{"insect", combat.GiantInsectRequest{Creature: cid("c"), Level: 4, Form: combat.InsectWasp}, 0},
		{"spirit", combat.SummonDragonRequest{Creature: cid("c"), Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageCold}, 0},
		{"steed", sumSteedRequest("c", 2), 0},
		{"skeleton", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("c", "b")}}, 50},
		{"zombie", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("c", "b", content.SizeSmall, true)}}, 50},
		{"ghoul", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("c", "r", combat.UndeadGhoul)}}, 200},
		{"ghast", combat.CreateUndeadRequest{Level: 8, Corpses: []combat.CorpseTarget{sumGhoul("c", "r", combat.UndeadGhast)}}, 450},
		{"wight", combat.CreateUndeadRequest{Level: 8, Corpses: []combat.CorpseTarget{sumGhoul("c", "r", combat.UndeadWight)}}, 700},
		{"mummy", combat.CreateUndeadRequest{Level: 9, Corpses: []combat.CorpseTarget{sumGhoul("c", "r", combat.UndeadMummy)}}, 700},
		{"familiar", combat.FindFamiliarRequest{Creature: cid("c"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := sumDo(t, combat.SummonedRoster{}, "pc", c.request, sumScene())
			if got := a.Creatures()[0].StatBlock().ExperiencePoints(); got != c.xp {
				t.Errorf("XP = %d, want %d [FR-007, p. 256]", got, c.xp)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-065, FR-066 - a summons leaves its input roster equal to its prior value (also after a refusal), returns slices that are copies, and gives an equal result for the same roster, command and facts.
// Path: unhappy (immutability).
func TestASummonsLeavesItsInputAloneAndIsDeterministic(t *testing.T) {
	t.Parallel()
	roster := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	prior := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	req := sumObjects("o1", "o2")
	a := sumDo(t, roster, "pc", req, sumCombat(""))
	again := sumDo(t, roster, "pc", req, sumCombat(""))
	if !reflect.DeepEqual(a, again) {
		t.Errorf("two identical casts differ [FR-066]")
	}
	if !roster.Equal(prior) || a.Roster().Equal(roster) || roster.Len() != 1 || a.Roster().Len() != 3 {
		t.Errorf("the cast altered its input roster or built an equal one [FR-065]")
	}
	sumTry(t, roster, "pc", combat.GiantInsectRequest{Creature: cid("i1"), Level: 4, Form: combat.InsectWasp}, sumScene())
	if !roster.Equal(prior) {
		t.Errorf("a refusal altered the roster [FR-065]")
	}
	events, actions, creatures, links := a.Events(), a.RosterActions(), a.Creatures(), a.Roster().Links()
	events[0], actions[0], creatures[0], links[0] = nil, nil, combat.SummonedCreatureSpec{}, combat.SummonedLink{}
	if a.Events()[0] == nil || a.RosterActions()[0] == nil || a.Creatures()[0].SummonKind() == 0 || a.Roster().Links()[0].SummonKind() == 0 {
		t.Errorf("a returned slice changed the result [FR-065]")
	}
	friends := sumDo(t, combat.SummonedRoster{}, "pc", sumSimulacrumRequest("s", 7, "ally"), sumScene()).Roster()
	l, _ := friends.Link("s")
	l.DesignatedFriends()[0] = "x"
	if l2, _ := friends.Link("s"); l2.DesignatedFriends()[0] != "ally" {
		t.Errorf("DesignatedFriends is not a copy [FR-065]")
	}
	req2 := combat.AnimateObjectsRequest{Level: 5, Objects: []combat.ObjectTarget{{Creature: cid("a"), Size: content.SizeSmall}}}
	cmd := sumCmd(t, combat.SummonedRoster{}, "pc", req2, sumScene())
	req2.Objects[0].Creature = "mutated"
	if got := (combat.SummonedRoster{}).Summon(cmd).(combat.SummonedApplied).Creatures()[0].Creature().ID; got != "a" {
		t.Errorf("changing the request after the command was built changed the cast to %v [FR-065]", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md plan P10 (summons per summoner), FR-044 - a first familiar cast by a summoner who holds 32 creatures is refused with SummonLimitExceededRejection, while a recast that only changes the form of the familiar the summoner has is not a new creature and is accepted at the cap.
// Path: unhappy (boundary).
func TestAFamiliarAtTheCap(t *testing.T) {
	t.Parallel()
	r := combat.SummonedRoster{}
	for i := 0; i < 31; i++ {
		r = sumInsect(t, r, "pc", sumLabel("i", i))
	}
	cast := combat.FindFamiliarRequest{Creature: cid("fam"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}
	withFamiliar := sumDo(t, r, "pc", cast, sumScene()).Roster()
	if withFamiliar.Len() != 32 {
		t.Fatalf("roster has %d links, want 32", withFamiliar.Len())
	}
	if _, ok := sumRejection(t, sumTry(t, sumInsect(t, withFamiliar, "other", "x"), "pc", combat.GiantInsectRequest{Creature: cid("y"), Level: 4, Form: combat.InsectWasp}, sumScene())).(combat.SummonLimitExceededRejection); !ok {
		t.Errorf("a 33rd creature was accepted")
	}
	recast := combat.FindFamiliarRequest{Creature: cid("fam2"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFiend, Origin: combat.FamiliarOriginFindFamiliarSpell}
	if a := sumDo(t, withFamiliar, "pc", recast, sumScene()); len(sumKinds(a.Events())) != 1 || a.Roster().Len() != 32 {
		t.Errorf("a form change at the cap gave %v, want one form change [FR-044]", sumKinds(a.Events()))
	}
	full := sumInsect(t, r, "pc", "i31")
	if _, ok := sumRejection(t, sumTry(t, full, "pc", cast, sumScene())).(combat.SummonLimitExceededRejection); !ok {
		t.Errorf("a first familiar at the cap was accepted [P10]")
	}
}
