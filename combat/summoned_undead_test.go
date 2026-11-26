package combat_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sumRaise raises n skeletons u00 to u<n-1> for the summoner "pc" with Animate Dead at level 9, a cast at a time of at most 13.
func sumRaise(tb testing.TB, n int) combat.SummonedRoster {
	tb.Helper()
	r := combat.SummonedRoster{}
	var q combat.AnimateDeadRequest
	q.Level = 9
	for i := 0; i < n; i++ {
		q.Targets = append(q.Targets, sumBones(sumLabel("u", i), sumLabel("b", i)))
	}
	return sumDo(tb, r, "pc", q, sumScene()).Roster()
}

func sumReassert(kind combat.SummonKind, level int, ids ...string) combat.ReassertControlRequest {
	q := combat.ReassertControlRequest{Kind: kind, Level: level}
	for _, id := range ids {
		q.Creatures = append(q.Creatures, cid(id))
	}
	return q
}

func sumNames(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, sumLabel("u", i))
	}
	return out
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-041, Table U (reassert at most: Animate Dead L3 4, L4 6, L9 16) - control is reasserted over at most the cap of the level (4 plus 2 for each level above 3), at the cap and one over, resetting each window to 24 hours without animating a new creature (SRD 5.2.1, "Animate Dead", p. 108).
// Path: per case.
func TestAnimateDeadReassertCaps(t *testing.T) {
	t.Parallel()
	roster := sumRaise(t, 13)
	aged := sumApplied(t, roster.ElapseControl(20*time.Hour)).Roster()
	cases := []struct {
		name, requirement, path string
		level, count            int
		maximum                 int
	}{
		{"L3 four", "Table U, p. 108", "happy", 3, 4, 0},
		{"L3 five", "Table U, p. 108", "unhappy (boundary)", 3, 5, 4},
		{"L4 six", "Table U, p. 108", "unhappy (boundary)", 4, 6, 0},
		{"L4 seven", "Table U, p. 108", "unhappy (boundary)", 4, 7, 6},
		{"L9 thirteen of thirteen", "Table U, p. 108", "unhappy (boundary)", 9, 13, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := aged.Reassert(cid("pc"), sumReassert(combat.SummonAnimatedUndead, c.level, sumNames(c.count)...))
			if c.maximum > 0 {
				if rej := sumRejection(t, got); rej != (combat.TooManyCreaturesRejection{Maximum: c.maximum}) {
					t.Errorf("rejection = %#v, want TooManyCreaturesRejection{%d} [%s]", rej, c.maximum, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			ev, ok := a.Events()[0].(combat.SummonControlReasserted)
			want := append([]combat.CombatantID{"pc"}, func() []combat.CombatantID {
				var ids []combat.CombatantID
				for _, n := range sumNames(c.count) {
					ids = append(ids, cid(n))
				}
				return ids
			}()...)
			if !ok || len(a.Events()) != 1 || !slices.Equal(ev.Combatants(), want) || ev.SummonKind() != combat.SummonAnimatedUndead {
				t.Errorf("events = %#v, want one SummonControlReasserted of the summoner and the creatures [FR-041]", a.Events())
			}
			for i, n := range sumNames(13) {
				l, _ := a.Roster().Link(cid(n))
				w, _ := l.ControlWindow()
				if wantLeft := map[bool]time.Duration{true: 24 * time.Hour, false: 4 * time.Hour}[i < c.count]; w.Left() != wantLeft {
					t.Errorf("%s window = %v, want %v: only the named creatures are reset [FR-041]", n, w.Left(), wantLeft)
				}
			}
			if a.Roster().Len() != 13 || len(a.RosterActions()) != 0 || len(a.Creatures()) != 0 {
				t.Errorf("a reassert animated a creature or touched the roster of the encounter [FR-041]")
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-041 - reasserting control accepts a creature with 1 minute left and refuses one with 0 left with ControlLapsedRejection; a lapsed window is not revived by the cast.
// Path: per case.
func TestReassertAtTheEdgeOfTheWindow(t *testing.T) {
	t.Parallel()
	roster := sumRaise(t, 2)
	for _, c := range []struct {
		name    string
		elapsed time.Duration
		lapsed  bool
	}{{"1 minute left", 24*time.Hour - time.Minute, false}, {"0 left", 24 * time.Hour, true}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			aged := sumApplied(t, roster.ElapseControl(c.elapsed)).Roster()
			got := aged.Reassert(cid("pc"), sumReassert(combat.SummonAnimatedUndead, 3, "u00"))
			if c.lapsed {
				if rej := sumRejection(t, got); rej != (combat.ControlLapsedRejection{Creature: cid("u00")}) {
					t.Errorf("rejection = %#v, want ControlLapsedRejection{u00} [FR-041]", rej)
				}
				return
			}
			a := sumApplied(t, got)
			if l, _ := a.Roster().Link("u00"); func() time.Duration { w, _ := l.ControlWindow(); return w.Left() }() != 24*time.Hour {
				t.Errorf("window = %v, want 24 hours [FR-041]", l)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-042, Table P (reassert vs animate) - a ReassertControl naming no linked undead creature is refused with NothingToReassertRejection, one naming a creature of another summoner, spell or none with the same, and a list with one linked and one that is not with TargetNotEligibleRejection; an animate request naming no target is NothingToAnimateRejection; a level below the spell's own is refused first.
// Path: per case.
func TestReassertNeedsLinkedUndead(t *testing.T) {
	t.Parallel()
	roster := sumRaise(t, 2)
	cases := []struct {
		name, requirement, path string
		request                 combat.ReassertControlRequest
		summoner                string
		want                    combat.SummonRejection
	}{
		{"no creature named", "FR-042, Table P", "unhappy (empty case)", sumReassert(combat.SummonAnimatedUndead, 3), "pc", combat.NothingToReassertRejection{}},
		{"a creature that is not linked", "FR-042", "unhappy (invalid input)", sumReassert(combat.SummonAnimatedUndead, 3, "ghost"), "pc", combat.NothingToReassertRejection{}},
		{"another summoner's creatures", "FR-041", "unhappy (invalid input)", sumReassert(combat.SummonAnimatedUndead, 3, "u00"), "other", combat.NothingToReassertRejection{}},
		{"creatures of another spell", "FR-041", "unhappy (invalid input)", sumReassert(combat.SummonCreatedUndead, 6, "u00"), "pc", combat.NothingToReassertRejection{}},
		{"one linked and one not", "FR-041", "unhappy (rule interaction)", sumReassert(combat.SummonAnimatedUndead, 3, "u00", "ghost"), "pc", combat.TargetNotEligibleRejection{Creature: cid("ghost")}},
		{"below the spell's level", "FR-035", "unhappy (boundary)", sumReassert(combat.SummonAnimatedUndead, 2, "u00"), "pc", combat.LevelBelowSpellRejection{Minimum: 3}},
		{"Create Undead below its level", "FR-035", "unhappy (boundary)", sumReassert(combat.SummonCreatedUndead, 5, "u00"), "pc", combat.LevelBelowSpellRejection{Minimum: 6}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := sumRejection(t, roster.Reassert(cid(c.summoner), c.request)); got != c.want {
				t.Errorf("rejection = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
	if _, ok := roster.Reassert(cid("pc"), sumReassert(combat.SummonGiantInsect, 4, "u00")).(combat.SummonedUnchanged); !ok {
		t.Errorf("a reassert of a kind that is not undead changed the roster")
	}
	if _, ok := roster.Reassert("", sumReassert(combat.SummonAnimatedUndead, 3, "u00")).(combat.SummonedUnchanged); !ok {
		t.Errorf("a reassert with no summoner changed the roster")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-041, FR-039, Table U (Create Undead), Table R - the creatures a Create Undead reassert names fit one composition row and at most its cap; a mixture is CompositionNotAllowedRejection, before ControlLapsedRejection, and too many of a row is refused with its cap (SRD 5.2.1, "Create Undead", p. 121).
// Path: per case.
func TestReassertOfCreateUndeadFollowsTheRows(t *testing.T) {
	t.Parallel()
	// mk raises the creatures with a cast each, since one cast cannot mix rows.
	mk := func(level int, forms ...combat.UndeadForm) combat.SummonedRoster {
		r := combat.SummonedRoster{}
		for i, f := range forms {
			q := combat.CreateUndeadRequest{Level: level, Corpses: []combat.CorpseTarget{sumGhoul(sumLabel("u", i), sumLabel("r", i), f)}}
			r = sumDo(t, r, "pc", q, sumScene()).Roster()
		}
		return r
	}
	g, w := combat.UndeadGhoul, combat.UndeadWight
	mixed := mk(8, g, w)
	lapsedMixed := sumApplied(t, mixed.ElapseControl(24*time.Hour)).Roster()
	fiveGhouls := mk(8, g, g, g, g, g)
	cases := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		level                   int
		ids                     []string
		want                    combat.SummonRejection
	}{
		{"a Ghoul alone", "FR-041", "happy", mixed, 8, []string{"u00"}, nil},
		{"a Wight alone", "FR-041", "happy", mixed, 8, []string{"u01"}, nil},
		{"mixed rows", "FR-041, Table P", "unhappy (rule interaction)", mixed, 8, []string{"u00", "u01"}, combat.CompositionNotAllowedRejection{}},
		{"a row the level does not allow", "FR-041", "unhappy (rule interaction)", mixed, 7, []string{"u01"}, combat.CompositionNotAllowedRejection{}},
		{"composition before a lapse", "Table R", "unhappy (rule interaction)", lapsedMixed, 8, []string{"u00", "u01"}, combat.CompositionNotAllowedRejection{}},
		{"a lapse", "FR-041", "unhappy (state error)", lapsedMixed, 8, []string{"u00"}, combat.ControlLapsedRejection{Creature: cid("u00")}},
		{"five Ghouls at 8", "Table U L8", "unhappy (boundary)", fiveGhouls, 8, []string{"u00", "u01", "u02", "u03", "u04"}, nil},
		{"five Ghouls at 7", "Table U L7", "unhappy (boundary)", fiveGhouls, 7, []string{"u00", "u01", "u02", "u03", "u04"}, combat.TooManyCreaturesRejection{Maximum: 4}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := c.roster.Reassert(cid("pc"), sumReassert(combat.SummonCreatedUndead, c.level, c.ids...))
			if c.want == nil {
				sumApplied(t, got)
				return
			}
			if rej := sumRejection(t, got); rej != c.want {
				t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-041, FR-042, FR-070 - a ReassertControl request sent through Summon takes the same path as Reassert and needs no stat block; the command constructor refuses a repeated or blank id and a kind that is not undead (FR-067).
// Path: per case.
func TestReassertThroughSummon(t *testing.T) {
	t.Parallel()
	roster := sumRaise(t, 2)
	cmd := sumCmd(t, roster, "pc", sumReassert(combat.SummonAnimatedUndead, 3, "u00"), sumScene())
	a := sumApplied(t, roster.Summon(cmd))
	if got := sumKinds(a.Events()); !slices.Equal(got, []string{"summon_control_reasserted"}) {
		t.Errorf("events = %v, want summon_control_reasserted [FR-041]", got)
	}
	if ids := combat.MonsterIDsNeeded(sumReassert(combat.SummonAnimatedUndead, 3, "u00")); len(ids) != 0 {
		t.Errorf("MonsterIDsNeeded = %v, want none for a reassert [FR-070]", ids)
	}
	for _, bad := range []combat.SummonRequest{
		sumReassert(combat.SummonAnimatedUndead, 3, "u00", "u00"), combat.ReassertControlRequest{Kind: combat.SummonAnimatedUndead, Level: 3, Creatures: []combat.CombatantID{""}},
		sumReassert(combat.SummonGiantInsect, 4, "u00"), sumReassert(combat.SummonAnimatedUndead, 0, "u00"), sumReassert(combat.SummonAnimatedUndead, 10, "u00"), sumReassert(combat.SummonAnimatedUndead, 3, "pc"),
	} {
		if _, err := combat.NewSummonCreatureCommand(roster, "pc", combat.SideA, bad, sumFacts(t), sumScene()); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%#v error = %v, want srderr.ErrInvalidInput [FR-067]", bad, err)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-070, FR-043 - MonsterIDsNeeded names the stat blocks a cast reads from content, without repeats and in the order first needed: the Skeleton or Zombie of Animate Dead, the forms of Create Undead and the form of a familiar; the expression kinds need none.
// Path: per case.
func TestMonsterIDsNeeded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		request combat.SummonRequest
		want    []content.MonsterID
	}{
		{"bones and corpses", combat.AnimateDeadRequest{Level: 5, Targets: []combat.UndeadTarget{sumBones("a", "1"), sumCorpse("b", "2", content.SizeSmall, true), sumBones("c", "3")}}, []content.MonsterID{"skeleton", "zombie"}},
		{"create undead forms", combat.CreateUndeadRequest{Level: 9, Corpses: []combat.CorpseTarget{sumGhoul("a", "1", combat.UndeadWight), sumGhoul("b", "2", combat.UndeadGhast), sumGhoul("c", "3", combat.UndeadWight)}}, []content.MonsterID{"wight", "ghast"}},
		{"a familiar form", combat.FindFamiliarRequest{Creature: cid("f"), Level: 1, Form: specialForm("imp"), Origin: combat.FamiliarOriginPactOfTheChain}, []content.MonsterID{"imp"}},
		{"objects", sumObjects("a"), nil},
		{"an insect", combat.GiantInsectRequest{Creature: cid("a"), Level: 4, Form: combat.InsectWasp}, nil},
		{"a steed", sumSteedRequest("a", 2), nil},
		{"a simulacrum", sumSimulacrumRequest("a", 7), nil},
		{"no request", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.MonsterIDsNeeded(c.request); !slices.Equal(got, c.want) {
				t.Errorf("MonsterIDsNeeded = %v, want %v [FR-070]", got, c.want)
			}
		})
	}
}
