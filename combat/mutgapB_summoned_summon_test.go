package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// mgbGoneSteed summons a steed for the summoner and reports it at 0 Hit Points, so the roster remembers it as gone.
func mgbGoneSteed(tb testing.TB, r combat.SummonedRoster, summoner, id string) combat.SummonedRoster {
	tb.Helper()
	r = sumDo(tb, r, summoner, sumSteedRequest(id, 2), sumScene()).Roster()
	return sumApplied(tb, sumChange(tb, r, combat.ReportZeroHitPoints{Creature: cid(id)}, "")).Roster()
}

// mgbGoneFamiliar is mgbGoneSteed for a familiar.
func mgbGoneFamiliar(tb testing.TB, r combat.SummonedRoster, summoner, id string) combat.SummonedRoster {
	tb.Helper()
	r = sumFamiliar(tb, r, summoner, id, combat.FamiliarOriginFindFamiliarSpell)
	return sumApplied(tb, sumChange(tb, r, combat.ReportZeroHitPoints{Creature: cid(id)}, "")).Roster()
}

// mgbSameSteed is a cast of Find Steed bringing back the steed id.
func mgbSameSteed(id string) combat.FindSteedRequest {
	q := sumSteedRequest("unused", 2)
	q.Choice = combat.SteedChoiceSame{Creature: cid(id)}
	return q
}

// mgbHas reports whether the roster links the creature.
func mgbHas(r combat.SummonedRoster, id string) bool {
	_, ok := r.Link(cid(id))
	return ok
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008 - a creature id that names any creature of the roster, not only the first one linked, is refused with SummonCreatureIDInUseRejection.
// Path: unhappy (state error).
func TestMutgapBIDInUseIsFoundAnywhereInTheRoster(t *testing.T) {
	t.Parallel()
	r := sumInsect(t, combat.SummonedRoster{}, "pc1", "i1")
	r = sumInsect(t, r, "pc2", "i2")
	got := sumTry(t, r, "pc3", combat.GiantInsectRequest{Creature: cid("i2"), Level: 4, Form: combat.InsectWasp}, sumScene())
	if rej := sumRejection(t, got); rej != (combat.SummonCreatureIDInUseRejection{ID: cid("i2")}) {
		t.Errorf("rejection = %#v, want SummonCreatureIDInUseRejection{i2} [FR-008]", rej)
	}
	spirit := combat.SummonDragonRequest{Creature: cid("i1"), Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageFire}
	if rej := sumRejection(t, sumTry(t, r, "pc3", spirit, sumScene())); rej != (combat.SummonCreatureIDInUseRejection{ID: cid("i1")}) {
		t.Errorf("Draconic Spirit rejection = %#v, want SummonCreatureIDInUseRejection{i1} [FR-008]", rej)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md plan P10 (summons per summoner) - the cap counts the creatures a cast replaces as leaving: a summoner holding 32 creatures, one of them a steed, may cast Find Steed again for a different steed (32 - 1 + 1), while a 33rd creature of another kind is refused.
// Path: unhappy (boundary).
func TestMutgapBReplacingASteedAtTheCapKeepsTheCount(t *testing.T) {
	t.Parallel()
	big := luMust(combat.NewSummonerSpellFacts(7, 15, 40, 3))
	q := combat.AnimateObjectsRequest{Level: 5}
	for i := 0; i < 31; i++ {
		q.Objects = append(q.Objects, combat.ObjectTarget{Creature: cid(sumLabel("o", i)), Size: content.SizeMedium})
	}
	cmd, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, cid("pc"), combat.SideA, q, big, sumScene())
	if err != nil {
		t.Fatal(err)
	}
	r := sumApplied(t, (combat.SummonedRoster{}).Summon(cmd)).Roster()
	full := sumDo(t, r, "pc", sumSteedRequest("st1", 2), sumScene()).Roster()
	if n := len(full.LinksOf(cid("pc"))); n != 32 {
		t.Fatalf("setup holds %d creatures, want 32", n)
	}
	got := sumApplied(t, sumTry(t, full, "pc", sumSteedRequest("st2", 2), sumScene())).Roster()
	if !mgbHas(got, "st2") || mgbHas(got, "st1") || len(got.LinksOf(cid("pc"))) != 32 {
		t.Errorf("the replacement left st1 %v, st2 %v, %d creatures, want st2 only and 32 [P10]", mgbHas(got, "st1"), mgbHas(got, "st2"), len(got.LinksOf(cid("pc"))))
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-056 - bringing back a steed that disappeared looks the steed up among the summoner's own gone steeds wherever it sits in the roster's memory: Same is accepted for the summoner's steed after other summoners' steeds and its own familiar went, is refused for another id, and Different with the gone steed's id is refused with SummonCreatureIDInUseRejection while any other id is accepted.
// Path: unhappy (rule interaction).
func TestMutgapBGoneSteedIsLookedUpAmongTheSummonersOwn(t *testing.T) {
	t.Parallel()
	// The gone list is [pc1's steed, pc3's steed, pc2's familiar, pc2's steed].
	r := mgbGoneSteed(t, combat.SummonedRoster{}, "pc1", "s1")
	r = mgbGoneSteed(t, r, "pc3", "s3")
	r = mgbGoneFamiliar(t, r, "pc2", "f2")
	r = mgbGoneSteed(t, r, "pc2", "s2")
	t.Run("the same steed comes back", func(t *testing.T) {
		t.Parallel()
		got := sumApplied(t, sumTry(t, r, "pc2", mgbSameSteed("s2"), sumScene())).Roster()
		if !mgbHas(got, "s2") {
			t.Error("the steed that disappeared did not come back [FR-056]")
		}
	})
	t.Run("another id is not the steed that disappeared", func(t *testing.T) {
		t.Parallel()
		for _, id := range []string{"s1", "s3", "s9", "a0"} {
			if rej := sumRejection(t, sumTry(t, r, "pc2", mgbSameSteed(id), sumScene())); rej != (combat.SteedNotDisappearedRejection{}) {
				t.Errorf("Same(%s) rejection = %#v, want SteedNotDisappearedRejection [FR-056]", id, rej)
			}
		}
	})
	t.Run("a different steed may not reuse the gone one's id", func(t *testing.T) {
		t.Parallel()
		if rej := sumRejection(t, sumTry(t, r, "pc2", sumSteedRequest("s2", 2), sumScene())); rej != (combat.SummonCreatureIDInUseRejection{ID: cid("s2")}) {
			t.Errorf("rejection = %#v, want SummonCreatureIDInUseRejection{s2} [FR-056]", rej)
		}
	})
	t.Run("a different steed with an id sorting before the gone one's is accepted", func(t *testing.T) {
		t.Parallel()
		if got := sumApplied(t, sumTry(t, r, "pc2", sumSteedRequest("a0", 2), sumScene())).Roster(); !mgbHas(got, "a0") {
			t.Error("the new steed was not linked [FR-056]")
		}
	})
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-023, FR-057 - a summoner's death touches the summoner's own steed only: another summoner's steed, a Giant Insect of the summoner and a summoner with no creature leave the roster unchanged.
// Path: unhappy (rule interaction).
func TestMutgapBSummonerDeathTouchesOnlyItsOwnSteed(t *testing.T) {
	t.Parallel()
	r := sumInsect(t, combat.SummonedRoster{}, "pc1", "i1")
	r = sumDo(t, r, "pc2", sumSteedRequest("s2", 2), sumScene()).Roster()
	for _, dead := range []string{"pc1", "pc3"} {
		if _, ok := sumChange(t, r, combat.ReportSummonerDied{Summoner: cid(dead)}, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("the death of %s changed the roster [FR-023]", dead)
		}
	}
	for _, dead := range []string{"pc1", "pc3"} {
		other := sumDo(t, combat.SummonedRoster{}, "pc2", sumSteedRequest("s2", 2), sumScene()).Roster()
		if _, ok := sumChange(t, other, combat.ReportSummonerDied{Summoner: cid(dead)}, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("the death of %s took pc2's steed [FR-023]", dead)
		}
	}
	got := sumApplied(t, sumChange(t, r, combat.ReportSummonerDied{Summoner: cid("pc2")}, "")).Roster()
	if mgbHas(got, "s2") || !mgbHas(got, "i1") {
		t.Errorf("the death of pc2 left steed %v, insect %v, want the steed gone only [FR-023]", mgbHas(got, "s2"), mgbHas(got, "i1"))
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-064 - a new Simulacrum replaces the summoner's own earlier one wherever it sits in the roster, and replaces no other creature: another summoner's simulacrum (either identifier order) and the summoner's own Giant Insect stay.
// Path: unhappy (rule interaction).
func TestMutgapBSimulacrumReplacesOnlyTheSummonersOwn(t *testing.T) {
	t.Parallel()
	t.Run("another summoner's simulacrum stays", func(t *testing.T) {
		t.Parallel()
		r := sumDo(t, combat.SummonedRoster{}, "pc2", sumSimulacrumRequest("sim2", 7), sumScene()).Roster()
		for _, summoner := range []string{"pc1", "pc3"} {
			got := sumDo(t, r, summoner, sumSimulacrumRequest("sim_"+summoner, 7), sumScene()).Roster()
			if !mgbHas(got, "sim2") || !mgbHas(got, "sim_"+summoner) {
				t.Errorf("%s's cast replaced pc2's simulacrum [FR-064]", summoner)
			}
		}
	})
	t.Run("a Giant Insect is not a simulacrum", func(t *testing.T) {
		t.Parallel()
		r := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
		if got := sumDo(t, r, "pc", sumSimulacrumRequest("sim", 7), sumScene()).Roster(); !mgbHas(got, "i1") || !mgbHas(got, "sim") {
			t.Error("the cast replaced the Giant Insect [FR-064]")
		}
	})
	t.Run("the earlier simulacrum sits after another creature", func(t *testing.T) {
		t.Parallel()
		r := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
		r = sumDo(t, r, "pc", sumSimulacrumRequest("sim1", 7), sumScene()).Roster()
		got := sumDo(t, r, "pc", sumSimulacrumRequest("sim2", 7), sumScene()).Roster()
		if mgbHas(got, "sim1") || !mgbHas(got, "sim2") || !mgbHas(got, "i1") {
			t.Errorf("after the second cast sim1 %v, sim2 %v, i1 %v, want the first simulacrum replaced only [FR-064]", mgbHas(got, "sim1"), mgbHas(got, "sim2"), mgbHas(got, "i1"))
		}
	})
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-036 - Animate Dead refuses two creatures raised from the same remains with DuplicateTargetRejection naming the second.
// Path: unhappy (invalid input).
func TestMutgapBAnimateDeadRefusesDuplicateRemains(t *testing.T) {
	t.Parallel()
	q := combat.AnimateDeadRequest{Level: 4, Targets: []combat.UndeadTarget{sumBones("u1", "b1"), sumBones("u2", "b1")}}
	if rej := sumRejection(t, sumTry(t, combat.SummonedRoster{}, "pc", q, sumScene())); rej != (combat.DuplicateTargetRejection{Creature: cid("u2")}) {
		t.Errorf("rejection = %#v, want DuplicateTargetRejection{u2} [FR-036]", rej)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-039, FR-041 - Create Undead refuses a mixture of rows whichever row comes first: a Mummy then a Ghoul at level 9 is CompositionNotAllowedRejection, as a Ghoul then a Mummy is.
// Path: unhappy (rule interaction).
func TestMutgapBCreateUndeadMixedRowsAreRefusedInEitherOrder(t *testing.T) {
	t.Parallel()
	for name, forms := range map[string][]combat.UndeadForm{
		"mummy then ghoul": {combat.UndeadMummy, combat.UndeadGhoul},
		"ghoul then mummy": {combat.UndeadGhoul, combat.UndeadMummy},
	} {
		q := combat.CreateUndeadRequest{Level: 9, Corpses: []combat.CorpseTarget{sumGhoul("c1", "r1", forms[0]), sumGhoul("c2", "r2", forms[1])}}
		if rej := sumRejection(t, sumTry(t, combat.SummonedRoster{}, "pc", q, sumScene())); rej != (combat.CompositionNotAllowedRejection{}) {
			t.Errorf("%s: rejection = %#v, want CompositionNotAllowedRejection [FR-039]", name, rej)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-041, FR-042 - reasserting control names the summoner's own creatures of the request's own kind only: another summoner's creature (either identifier order) and the summoner's creature of the other undead kind are NothingToReassertRejection.
// Path: unhappy (rule interaction).
func TestMutgapBReassertNamesOnlyTheSummonersOwnCreaturesOfTheKind(t *testing.T) {
	t.Parallel()
	r := sumDo(t, combat.SummonedRoster{}, "pc2", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("g1", "r1", combat.UndeadGhoul)}}, sumScene()).Roster()
	r = sumDo(t, r, "pc2", combat.AnimateDeadRequest{Level: 4, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}, sumScene()).Roster()
	cases := []struct {
		name, summoner string
		kind           combat.SummonKind
		creature       string
	}{
		{"summoner sorting before the owner", "pc1", combat.SummonCreatedUndead, "g1"},
		{"summoner sorting after the owner", "pc3", combat.SummonCreatedUndead, "g1"},
		{"Animate Dead creature named for Create Undead", "pc2", combat.SummonCreatedUndead, "u1"},
		{"Create Undead creature named for Animate Dead", "pc2", combat.SummonAnimatedUndead, "g1"},
	}
	for _, c := range cases {
		got := r.Reassert(cid(c.summoner), combat.ReassertControlRequest{Kind: c.kind, Level: 6, Creatures: []combat.CombatantID{cid(c.creature)}})
		if rej := sumRejection(t, got); rej != (combat.NothingToReassertRejection{}) {
			t.Errorf("%s: rejection = %#v, want NothingToReassertRejection [FR-041]", c.name, rej)
		}
	}
	own := r.Reassert(cid("pc2"), combat.ReassertControlRequest{Kind: combat.SummonCreatedUndead, Level: 6, Creatures: []combat.CombatantID{cid("g1")}})
	sumApplied(t, own)
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-043, FR-054 - a normal familiar form that is not one of the eleven is eligible only as a Beast of Challenge Rating 0: an Aberration of Challenge Rating 0 is refused with FormNotEligibleRejection, a Beast of Challenge Rating 0 is accepted.
// Path: unhappy (rule interaction).
func TestMutgapBFamiliarFormOfAnotherTypeIsNotEligible(t *testing.T) {
	t.Parallel()
	monsters := append(sumMonsters(t), sumMonster(t, "fixture_aberration", content.CreatureTypeAberration, content.SizeTiny, "0", 10))
	cast := func(form string) combat.SummonedTransition {
		req := combat.FindFamiliarRequest{Creature: cid("fam"), Level: 1, Form: character.NormalFamiliarForm(content.MonsterID(form)), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}
		cmd, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, cid("pc"), combat.SideA, req, sumFacts(t), sumCombat("", sumInit(10)))
		if err != nil {
			t.Fatal(err)
		}
		return (combat.SummonedRoster{}).Summon(cmd, monsters...)
	}
	if rej := sumRejection(t, cast("fixture_aberration")); rej != (combat.FormNotEligibleRejection{}) {
		t.Errorf("Aberration form rejection = %#v, want FormNotEligibleRejection [FR-043]", rej)
	}
	sumApplied(t, cast("fixture_beast"))
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-044 - casting Find Familiar again changes the form of the familiar the summoner has, even when the request names the familiar's own id: no id or level check applies to a form change.
// Path: unhappy (rule interaction).
func TestMutgapBFormChangeNeedsNoIDCheck(t *testing.T) {
	t.Parallel()
	r := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
	req := combat.FindFamiliarRequest{Creature: cid("fam"), Level: 1, Form: normalForm("fixture_beast"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}
	a := sumApplied(t, sumTry(t, r, "pc", req, sumScene()))
	if kinds := sumKinds(a.Events()); len(kinds) != 1 || kinds[0] != "familiar_form_changed" {
		t.Errorf("events = %v, want familiar_form_changed [FR-044]", kinds)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-022, FR-016 - a summoner's shared Resistance ends with its own spell: it ends when the second of two summoners' grants is the one named, and it is left alone by another summoner's spell ending, by a spell that sorts before or after Summon Dragon and by a summoner who holds none (either identifier order).
// Path: unhappy (rule interaction).
func TestMutgapBSharedResistanceEndsOnlyWithItsOwnSummonerAndSpell(t *testing.T) {
	t.Parallel()
	spirit := func(id string) combat.SummonDragonRequest {
		return combat.SummonDragonRequest{Creature: cid(id), Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageFire}
	}
	end := func(summoner, spell string) combat.ReportSpellEnded {
		return combat.ReportSpellEnded{Summoner: cid(summoner), Spell: content.SpellID(spell), Cause: combat.SummonSpellEndConcentrationEnded}
	}
	r := sumDo(t, combat.SummonedRoster{}, "pc2", spirit("d2"), sumScene()).Roster()
	for name, change := range map[string]combat.ReportSpellEnded{
		"summoner sorting before the owner":  end("pc1", "summon_dragon"),
		"summoner sorting after the owner":   end("pc3", "summon_dragon"),
		"spell sorting before Summon Dragon": end("pc2", "animate_objects"),
		"spell sorting after Summon Dragon":  end("pc2", "wish"),
	} {
		if _, ok := sumChange(t, r, change, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("%s: the roster changed [FR-022]", name)
		}
	}
	first := sumDo(t, combat.SummonedRoster{}, "pc1", spirit("d1"), sumScene()).Roster()
	both := sumDo(t, first, "pc2", spirit("d2"), sumScene()).Roster() // pc1's grant is first, pc2's second
	a := sumApplied(t, sumChange(t, both, end("pc2", "summon_dragon"), ""))
	if ended := sumKinds(a.Events()); !slices.Contains(ended, "shared_resistance_ended") {
		t.Errorf("events = %v, want shared_resistance_ended for the second grant [FR-022]", ended)
	}
	if mgbHas(a.Roster(), "d2") || !mgbHas(a.Roster(), "d1") {
		t.Errorf("d2 linked %v, d1 linked %v, want only pc1's spirit left [FR-022]", mgbHas(a.Roster(), "d2"), mgbHas(a.Roster(), "d1"))
	}
}
