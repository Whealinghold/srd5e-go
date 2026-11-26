package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// mgaDepart reports 0 Hit Points for the creature and returns the roster after the departure.
func mgaDepart(tb testing.TB, roster combat.SummonedRoster, id string) combat.SummonedRoster {
	tb.Helper()
	return sumApplied(tb, sumChange(tb, roster, combat.ReportZeroHitPoints{Creature: cid(id)}, "")).Roster()
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-007 - the stat block of a familiar takes the first size of the monster
// it is, here Tiny, not the Medium a monster with no size gets.
// Path: happy.
func TestFamiliarStatBlockTakesTheSizeOfItsMonster(t *testing.T) {
	t.Parallel()
	a := sumApplied(t, sumFamiliarCast(t, combat.SummonedRoster{}, "fam", normalForm("cat"), content.CreatureTypeFey, combat.FamiliarOriginFindFamiliarSpell, sumMonsters(t)))
	specs := a.Creatures()
	if len(specs) != 1 || specs[0].StatBlock().Size() != content.SizeTiny {
		t.Fatalf("Creatures = %v, want one Tiny creature", specs)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-006 - a link that is not in the roster is the zero link, which is not
// in a pocket dimension; a link in the encounter is not either.
// Path: unhappy (creature not in the roster).
func TestAnUnknownCreatureHasNoLinkAndIsNotInAPocketDimension(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	zero, ok := roster.Link(cid("nobody"))
	if ok || zero.InPocketDimension() {
		t.Errorf("Link(nobody) = %+v, %v, want the zero link, not in a pocket dimension", zero, ok)
	}
	if fam, ok := roster.Link(cid("fam")); !ok || fam.InPocketDimension() {
		t.Errorf("Link(fam).InPocketDimension() = %v, want false while it is in the encounter", fam.InPocketDimension())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-043 - only the link of a familiar holds familiar state; every other
// kind of link, a simulacrum included, reports none.
// Path: per case (every kind in the world roster).
func TestOnlyAFamiliarLinkHoldsFamiliarState(t *testing.T) {
	t.Parallel()
	seen := map[combat.SummonKind]bool{}
	for _, l := range sumWorld(t).Links() {
		seen[l.SummonKind()] = true
		if _, ok := l.Familiar(); ok != (l.SummonKind() == combat.SummonFamiliar) {
			t.Errorf("%v link: Familiar() ok = %v", l.SummonKind(), ok)
		}
	}
	if len(seen) != 7 {
		t.Fatalf("the world roster has %d kinds of link, want 7", len(seen))
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-064 - a simulacrum is friendly to its summoner and to each creature
// the summoner designated, whatever the order of the ids, and to no one else, itself included; no other kind of link is friendly to
// anyone (SRD 5.2.1, "Simulacrum", p. 163).
// Path: per case.
func TestASimulacrumIsFriendlyToItsSummonerAndDesignatedFriends(t *testing.T) {
	t.Parallel()
	roster := sumDo(t, combat.SummonedRoster{}, "pc", sumSimulacrumRequest("mid", 7, "zed", "abe"), sumScene()).Roster()
	link, ok := roster.Link(cid("mid"))
	if !ok {
		t.Fatal("no link for the simulacrum")
	}
	for _, c := range []struct {
		id   string
		want bool
	}{{"pc", true}, {"zed", true}, {"abe", true}, {"mid", false}, {"bob", false}, {"zzz", false}, {"aaa", false}} {
		if got := link.FriendlyTo(cid(c.id)); got != c.want {
			t.Errorf("FriendlyTo(%s) = %v, want %v", c.id, got, c.want)
		}
	}
	for _, l := range sumWorld(t).Links() {
		if l.SummonKind() != combat.SummonSimulacrum && l.FriendlyTo(cid("pc")) {
			t.Errorf("a %v link is friendly to its summoner", l.SummonKind())
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-044, FR-056 - the links of a summoner are the ones it made, in order,
// and a summoner whose id sorts before or after another's does not see the other's links.
// Path: per case (summoners on both sides of the id order).
func TestLinksOfASummonerAreOnlyItsOwn(t *testing.T) {
	t.Parallel()
	r := combat.SummonedRoster{}
	for _, s := range []struct{ summoner, id string }{{"aa", "x1"}, {"pc", "x2"}, {"zz", "x3"}} {
		r = sumInsect(t, r, s.summoner, s.id)
	}
	for _, s := range []struct{ summoner, id string }{{"aa", "x1"}, {"pc", "x2"}, {"zz", "x3"}} {
		links := r.LinksOf(cid(s.summoner))
		if len(links) != 1 || links[0].Creature().ID != cid(s.id) {
			t.Errorf("LinksOf(%s) = %v, want only %s", s.summoner, links, s.id)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-049 - FamiliarGone is true for the summoner whose familiar went and
// for no other summoner, and a later departure of another summoner's familiar or of the same summoner's steed does not erase it.
// Path: per case (departure orders that expose a comparison on either side of the ids).
func TestAFamiliarThatWentIsRememberedPerSummoner(t *testing.T) {
	t.Parallel()
	build := func(t *testing.T) combat.SummonedRoster {
		t.Helper()
		r := combat.SummonedRoster{}
		for _, s := range []string{"aa", "pc", "zz"} {
			r = sumFamiliar(t, r, s, "f-"+s, combat.FamiliarOriginFindFamiliarSpell)
		}
		return r
	}
	for _, order := range [][]string{{"aa", "pc", "zz"}, {"zz", "pc", "aa"}} {
		r := build(t)
		for i, s := range order {
			r = mgaDepart(t, r, "f-"+s)
			for _, gone := range order[:i+1] {
				if !r.FamiliarGone(cid(gone)) {
					t.Errorf("order %v: after %s departed, FamiliarGone(%s) = false, want true", order, s, gone)
				}
			}
			for _, kept := range order[i+1:] {
				if r.FamiliarGone(cid(kept)) {
					t.Errorf("order %v: after %s departed, FamiliarGone(%s) = true, want false", order, s, kept)
				}
			}
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-049, FR-056 - the departure of a summoner's steed neither erases the
// record of its familiar that went nor is mistaken for it, and the departure of its familiar does not erase the record of its steed,
// so the same steed can be brought back.
// Path: per case (familiar then steed, steed then familiar).
func TestAFamiliarAndASteedThatWentAreRememberedApart(t *testing.T) {
	t.Parallel()
	start := func(t *testing.T) combat.SummonedRoster {
		t.Helper()
		r := sumDo(t, combat.SummonedRoster{}, "pc", sumSteedRequest("st", 2), sumScene()).Roster()
		return sumFamiliar(t, r, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
	}
	same := combat.FindSteedRequest{Level: 2, Type: combat.SteedFey, Choice: combat.SteedChoiceSame{Creature: cid("st")},
		Anatomy: combat.MountAnatomyAppropriate, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleNotFitted}
	for _, order := range [][]string{{"fam", "st"}, {"st", "fam"}} {
		r := start(t)
		if r.FamiliarGone(cid("pc")) {
			t.Fatal("FamiliarGone before any departure")
		}
		for _, id := range order {
			r = mgaDepart(t, r, id)
		}
		if !r.FamiliarGone(cid("pc")) {
			t.Errorf("order %v: FamiliarGone(pc) = false, want true", order)
		}
		if _, ok := sumTry(t, r, "pc", same, sumScene()).(combat.SummonedApplied); !ok {
			t.Errorf("order %v: the same steed cannot be brought back", order)
		}
	}
	r := mgaDepart(t, sumDo(t, combat.SummonedRoster{}, "pc", sumSteedRequest("st", 2), sumScene()).Roster(), "st")
	if r.FamiliarGone(cid("pc")) {
		t.Error("FamiliarGone(pc) = true after only a steed went")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-056, FR-049 - only a steed or a familiar that went is remembered;
// any other summoned creature that departs leaves the roster exactly as if it had never been there.
// Path: per case (every other kind).
func TestOnlyASteedOrFamiliarIsRememberedWhenItDeparts(t *testing.T) {
	t.Parallel()
	for _, req := range []combat.SummonRequest{
		sumObjects("o1"), combat.GiantInsectRequest{Creature: cid("o1"), Level: 4, Form: combat.InsectSpider}, sumSimulacrumRequest("o1", 7),
		combat.AnimateDeadRequest{Level: 4, Targets: []combat.UndeadTarget{sumBones("o1", "b1")}},
	} {
		r := sumDo(t, combat.SummonedRoster{}, "pc", req, sumScene()).Roster()
		id := string(r.Links()[0].Creature().ID)
		if got := mgaDepart(t, r, id); !got.Equal(combat.SummonedRoster{}) {
			t.Errorf("a %T creature that departed is remembered", req)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-053 - the end of a long rest departs only the Wild Companion familiar
// of the summoner it names, whatever the order of the summoners' ids; another summoner's familiar stays and the change is Unchanged.
// Path: unhappy (summoner without a familiar).
func TestALongRestOnlyEndsTheFamiliarOfItsSummoner(t *testing.T) {
	t.Parallel()
	r := sumFamiliar(t, combat.SummonedRoster{}, "aa", "f-aa", combat.FamiliarOriginWildCompanion)
	if _, unchanged := sumChange(t, r, combat.ReportLongRestFinished{Summoner: cid("zz")}, "").(combat.SummonedUnchanged); !unchanged {
		t.Error("a long rest of a summoner with no familiar departed another summoner's familiar")
	}
	if _, applied := sumChange(t, r, combat.ReportLongRestFinished{Summoner: cid("aa")}, "").(combat.SummonedApplied); !applied {
		t.Error("a long rest did not depart the Wild Companion familiar of its summoner")
	}
}
