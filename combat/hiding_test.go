package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var hideR combat.HideRules

type cvr = []combat.HideCoverDegree

func enemy(id string, line combat.LineOfSight, sees combat.SightRelation) combat.EnemySighting {
	return combat.EnemySighting{Enemy: gid(id), Line: line, HiderSeesEnemy: sees}
}

func hideScene(tb testing.TB, o content.Obscurement, cover cvr, larger bool, enemies ...combat.EnemySighting) combat.HideScene {
	tb.Helper()
	s, err := combat.NewHideScene(gid("hider"), o, cover, larger, enemies)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

func available(tb testing.TB) combat.HideAvailable {
	tb.Helper()
	a, ok := hideR.Assess(hideScene(tb, content.ObscurementHeavily, nil, false)).(combat.HideAvailable)
	if !ok {
		tb.Fatal("Assess is not HideAvailable")
	}
	return a
}

// Requirement: per case (doc/specs/hiding-searching-and-senses.md 4.9a)
// Path: per case.
func TestAssessHide(t *testing.T) {
	t.Parallel()
	none, lightly, heavily := content.ObscurementNone, content.ObscurementLightly, content.ObscurementHeavily
	half, tq, total := combat.HideCoverDegreeHalfCover, combat.HideCoverDegreeThreeQuartersCover, combat.HideCoverDegreeTotalCover
	clear, blocked := combat.LineOfSightClear, combat.LineOfSightBlocked
	see, no := combat.SightRelationCanSee, combat.SightRelationCannotSee
	nc, el := combat.HideBlockNotConcealed, combat.HideBlockInEnemyLineOfSight
	one := func(l combat.LineOfSight) []combat.EnemySighting { return []combat.EnemySighting{enemy("e1", l, no)} }
	cases := []struct {
		name, requirement string
		o                 content.Obscurement
		cover             cvr
		larger            bool
		enemies           []combat.EnemySighting
		reasons           []combat.HideBlock // nil: HideAvailable
		clearIDs          []combat.CombatantID
		discerned         []combat.EnemySighting
	}{
		{"heavily obscured, no line", "FR-052, FR-054", heavily, nil, false, one(blocked), nil, nil, nil},
		{"Three-Quarters Cover", "FR-052", none, cvr{tq}, false, one(blocked), nil, nil, nil},
		{"Total Cover", "FR-052", none, cvr{total}, false, one(blocked), nil, nil, nil},
		{"Half Cover only", "FR-053", none, cvr{half}, false, one(blocked), []combat.HideBlock{nc}, nil, nil},
		{"Lightly Obscured only", "FR-053", lightly, nil, false, one(blocked), []combat.HideBlock{nc}, nil, nil},
		{"nothing", "FR-053", none, nil, false, one(blocked), []combat.HideBlock{nc}, nil, nil},
		{"Half and Three-Quarters Cover", "FR-052", none, cvr{half, tq}, false, one(blocked), nil, nil, nil},
		{"larger creature obscures", "FR-105", none, nil, true, one(blocked), nil, nil, nil},
		{"larger creature, enemy line clear", "FR-105, FR-054", none, nil, true, one(clear), []combat.HideBlock{el}, []combat.CombatantID{gid("e1")}, nil},
		{"concealed, second enemy sees", "FR-054", heavily, nil, false, []combat.EnemySighting{enemy("e1", blocked, no), enemy("e2", clear, no)}, []combat.HideBlock{el}, []combat.CombatantID{gid("e2")}, nil},
		{"concealed, two clear in order", "FR-054", heavily, nil, false, []combat.EnemySighting{enemy("e2", clear, no), enemy("e1", clear, no)}, []combat.HideBlock{el}, []combat.CombatantID{gid("e2"), gid("e1")}, nil},
		{"no enemies", "FR-055", heavily, nil, false, nil, nil, nil, nil},
		{"neither requirement", "FR-056", none, nil, false, one(clear), []combat.HideBlock{nc, el}, []combat.CombatantID{gid("e1")}, nil},
		{"discern: seen, blocked", "FR-057", heavily, nil, false, []combat.EnemySighting{enemy("e1", blocked, see)}, nil, nil, []combat.EnemySighting{enemy("e1", blocked, see)}},
		{"discern: unseen, blocked", "FR-057", heavily, nil, false, []combat.EnemySighting{enemy("e1", blocked, no)}, nil, nil, nil},
		{"discern: clear, seen", "FR-057, FR-054", heavily, nil, false, []combat.EnemySighting{enemy("e1", clear, see)}, []combat.HideBlock{el}, []combat.CombatantID{gid("e1")}, []combat.EnemySighting{enemy("e1", clear, see)}},
		{"discern: clear, unseen", "FR-057, FR-054", heavily, nil, false, []combat.EnemySighting{enemy("e1", clear, no)}, []combat.HideBlock{el}, []combat.CombatantID{gid("e1")}, nil},
		{"discern keeps scene order and drops unseen", "FR-057", heavily, nil, false, []combat.EnemySighting{enemy("e3", blocked, see), enemy("e2", blocked, no), enemy("e1", blocked, see)}, nil, nil,
			[]combat.EnemySighting{enemy("e3", blocked, see), enemy("e1", blocked, see)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := hideR.Assess(hideScene(t, c.o, c.cover, c.larger, c.enemies...))
			switch a := got.(type) {
			case combat.HideAvailable:
				if c.reasons != nil || !slices.Equal(a.DiscernedSightings(), c.discerned) || a.Hider != gid("hider") || a.Check.DC() != 15 {
					t.Errorf("HideAvailable %+v, want reasons %v, discerned %v [%s]", a, c.reasons, c.discerned, c.requirement)
				}
			case combat.HideBlocked:
				if !slices.Equal(a.Reasons(), c.reasons) || !slices.Equal(a.EnemiesWithLineOfSight(), c.clearIDs) || !slices.Equal(a.DiscernedSightings(), c.discerned) {
					t.Errorf("HideBlocked %v %v %v, want %v %v %v [%s]", a.Reasons(), a.EnemiesWithLineOfSight(), a.DiscernedSightings(), c.reasons, c.clearIDs, c.discerned, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-051, FR-098 (SRD 5.2.1, "Hide [Action]", p. 183; "Actions" table, p. 10) - the
// Hide check is Dexterity (Stealth) against DC 15.
// Path: happy.
func TestHideCheckSpec(t *testing.T) {
	t.Parallel()
	c := available(t).Check
	if c.Ability() != ability.Dexterity || c.Skill() != content.SkillStealth || c.DC() != d20.DifficultyClass(15) {
		t.Errorf("check = %v %v %v", c.Ability(), c.Skill(), c.DC())
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-097 - NewHideScene rejects a blank hider, an undefined obscurement, an
// undefined cover degree and an invalid sighting (blank enemy, undefined line, undefined relation) with ErrInvalidInput; the
// zero HideScene is blocked as not concealed.
// Path: unhappy (invalid input).
func TestNewHideSceneRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	good := enemy("e", combat.LineOfSightBlocked, combat.SightRelationCannotSee)
	for name, c := range map[string]struct {
		hider   combat.CombatantID
		o       content.Obscurement
		cover   cvr
		enemies []combat.EnemySighting
	}{
		"blank hider":   {"", content.ObscurementNone, nil, nil},
		"bad obscure":   {"h", 0, nil, nil},
		"bad cover":     {"h", content.ObscurementNone, cvr{0}, nil},
		"cover too big": {"h", content.ObscurementNone, cvr{4}, nil},
		"blank enemy":   {"h", content.ObscurementNone, nil, []combat.EnemySighting{good, {Line: 1, HiderSeesEnemy: 1}}},
		"bad line":      {"h", content.ObscurementNone, nil, []combat.EnemySighting{{Enemy: "e", HiderSeesEnemy: 1}}},
		"bad relation":  {"h", content.ObscurementNone, nil, []combat.EnemySighting{{Enemy: "e", Line: 1}}},
	} {
		if _, err := combat.NewHideScene(c.hider, c.o, c.cover, false, c.enemies); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	b, ok := hideR.Assess(combat.HideScene{}).(combat.HideBlocked)
	if !ok || !slices.Equal(b.Reasons(), []combat.HideBlock{combat.HideBlockNotConcealed}) {
		t.Errorf("Assess(zero scene) = %v", b)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-058, FR-100 - Resolve and ResolveForcedFailure take only an assessment from
// Assess: a zero HideAvailable, one with a blank hider or one whose check is not the Hide check is an ErrInvalidInput error.
// Path: unhappy (state error).
func TestResolveRejectsForgedAssessment(t *testing.T) {
	t.Parallel()
	forged := []combat.HideAvailable{{}, {Hider: gid("h")}, {Check: available(t).Check}}
	for i, a := range forged {
		if _, err := hideR.Resolve(a, 20, d20.Success); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Resolve forged %d: error = %v", i, err)
		}
		if _, err := hideR.ResolveForcedFailure(a, d20.Automatic); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ResolveForcedFailure forged %d: error = %v", i, err)
		}
	}
}

// Requirement: per case (doc/specs/hiding-searching-and-senses.md 4.9b)
// Path: per case.
func TestResolveHide(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		total                   int
		outcome                 d20.Outcome
		wantDC                  d20.DifficultyClass // 0: HideCheckFailed
		wantErr                 bool
	}{
		{"exactly DC 15", "FR-059", "unhappy (boundary)", 15, d20.Success, 15, false},
		{"one below", "FR-060", "unhappy (boundary)", 14, d20.Failure, 0, false},
		{"very low", "FR-060", "unhappy (boundary)", 2, d20.Failure, 0, false},
		{"one above", "FR-059", "unhappy (boundary)", 16, d20.Success, 16, false},
		{"high", "FR-059", "unhappy (boundary)", 23, d20.Success, 23, false},
		{"success below 15", "FR-059", "unhappy (invalid input)", 14, d20.Success, 0, true},
		{"failure at 15", "FR-059", "unhappy (invalid input)", 15, d20.Failure, 0, true},
		{"failure above 15", "FR-059", "unhappy (invalid input)", 16, d20.Failure, 0, true},
		{"undefined outcome", "FR-059", "unhappy (invalid input)", 20, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := hideR.Resolve(available(t), c.total, c.outcome)
			if c.wantErr {
				if !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
					t.Errorf("Resolve = %v, %v, want ErrInvalidInput [%s]", got, err, c.requirement)
				}
				return
			}
			switch r := got.(type) {
			case combat.HideSucceeded:
				ev, _ := combat.NewCreatureHid(gid("hider"), c.wantDC)
				if err != nil || r.State().FindDC() != c.wantDC || c.wantDC == 0 || !slices.Equal(r.Events(), []combat.Event{ev}) {
					t.Errorf("HideSucceeded %v, %v want find DC %v and one CreatureHid [%s]", r, err, c.wantDC, c.requirement)
				}
			case combat.HideCheckFailed:
				if c.wantDC != 0 || r.Total() != c.total {
					t.Errorf("HideCheckFailed(%d) [%s]", r.Total(), c.requirement)
				}
			default:
				t.Errorf("Resolve = %T", got)
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-061, FR-099, FR-100 - a forced failure is HideNotRolled with its reason and
// no event or state; HideSucceeded hands out its events as a copy.
// Path: unhappy (state error).
func TestResolveForcedFailureAndEventsCopy(t *testing.T) {
	t.Parallel()
	got, err := hideR.ResolveForcedFailure(available(t), d20.ChosenToFail)
	if err != nil || got.Reason() != d20.ChosenToFail {
		t.Errorf("ResolveForcedFailure = %v, %v", got, err)
	}
	r, _ := hideR.Resolve(available(t), 18, d20.Success)
	s := r.(combat.HideSucceeded)
	combattest.CheckEventsCopy(t, s.Events)
	if s.State() != (mustHidden(t, 18)) {
		t.Errorf("state = %v", s.State())
	}
}

func mustHidden(tb testing.TB, dc d20.DifficultyClass) combat.HiddenState {
	tb.Helper()
	s, err := combat.NewHiddenState(dc)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-059 - a HiddenState below DC 15 is an ErrInvalidInput error; 15 is valid.
// Path: unhappy (boundary).
func TestNewHiddenState(t *testing.T) {
	t.Parallel()
	if s, err := combat.NewHiddenState(15); err != nil || s.FindDC() != 15 {
		t.Errorf("NewHiddenState(15) = %v, %v", s, err)
	}
	for _, dc := range []d20.DifficultyClass{14, 0, -1} {
		if _, err := combat.NewHiddenState(dc); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewHiddenState(%d) error = %v", dc, err)
		}
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-062 (SRD 5.2.1, "Hide [Action]", p. 183) - while hidden the implied conditions are
// {invisible}, otherwise empty, as a fresh slice.
// Path: happy.
func TestImpliedConditions(t *testing.T) {
	t.Parallel()
	hidden := combat.HidingStatusHidden{State: mustHidden(t, 18)}
	if got := hideR.ImpliedConditions(hidden); !slices.Equal(got, ck{content.ConditionInvisible}) {
		t.Errorf("ImpliedConditions(hidden) = %v", got)
	}
	hideR.ImpliedConditions(hidden)[0] = 0
	if hideR.ImpliedConditions(hidden)[0] != content.ConditionInvisible {
		t.Error("ImpliedConditions is not a fresh slice [FR-100]")
	}
	for _, s := range []combat.HidingStatus{combat.HidingStatusNotHidden{}, nil} {
		if got := hideR.ImpliedConditions(s); got == nil || len(got) != 0 {
			t.Errorf("ImpliedConditions(%v) = %#v, want empty", s, got)
		}
	}
}

// Requirement: per case (doc/specs/hiding-searching-and-senses.md 4.10a, hider Hidden with FindDC 16)
// Path: per case.
func TestApplyActivityAndAttemptToFind(t *testing.T) {
	t.Parallel()
	hidden := combat.HidingStatus(combat.HidingStatusHidden{State: mustHidden(t, 16)})
	notHidden := combat.HidingStatus(combat.HidingStatusNotHidden{})
	stops := func(a combat.HidingActivity) []combat.Event {
		ev, _ := combat.NewHidingEnded(gid("hider"), a)
		return []combat.Event{ev}
	}
	finder, notEnemy := combat.HidingFinderEnemy, combat.HidingFinderNotEnemy
	cases := []struct {
		name, requirement, path string
		status                  combat.HidingStatus
		activity                combat.HidingActivity // 0: use finder
		finder                  combat.HidingFinder
		total                   int
		want                    combat.HidingStatus
		events                  []combat.Event
	}{
		{"shout", "FR-064, FR-071", "happy", hidden, combat.HidingActivitySoundLouderThanWhisper, 0, 0, notHidden, stops(combat.HidingActivitySoundLouderThanWhisper)},
		{"whisper", "FR-065", "unhappy (boundary)", hidden, combat.HidingActivitySoundNoLouderThanWhisper, 0, 0, hidden, nil},
		{"silent spell", "FR-070", "unhappy (boundary)", hidden, combat.HidingActivitySpellWithoutVerbalComponent, 0, 0, hidden, nil},
		{"enemy finds", "FR-066", "happy", hidden, combat.HidingActivityFoundByEnemy, 0, 0, notHidden, stops(combat.HidingActivityFoundByEnemy)},
		{"ally finds", "FR-067", "unhappy (boundary)", hidden, combat.HidingActivityFoundByNonEnemy, 0, 0, hidden, nil},
		{"attack roll", "FR-068", "unhappy (rule interaction)", hidden, combat.HidingActivityAttackRoll, 0, 0, notHidden, stops(combat.HidingActivityAttackRoll)},
		{"Verbal spell", "FR-069", "happy", hidden, combat.HidingActivitySpellWithVerbalComponent, 0, 0, notHidden, stops(combat.HidingActivitySpellWithVerbalComponent)},
		{"already not hidden", "FR-071", "unhappy (state error)", notHidden, combat.HidingActivityAttackRoll, 0, 0, notHidden, nil},
		{"undefined activity", "FR-102", "unhappy (invalid input)", hidden, 99, 0, 0, hidden, nil},
		{"nil status", "FR-071", "unhappy (invalid input)", nil, combat.HidingActivityAttackRoll, 0, 0, nil, nil},
		{"enemy total equals DC", "FR-072", "unhappy (boundary)", hidden, 0, finder, 16, notHidden, stops(combat.HidingActivityFoundByEnemy)},
		{"enemy total above", "FR-072", "unhappy (boundary)", hidden, 0, finder, 22, notHidden, stops(combat.HidingActivityFoundByEnemy)},
		{"enemy total one below", "FR-072", "unhappy (boundary)", hidden, 0, finder, 15, hidden, nil},
		{"non-enemy, total 30", "FR-067, FR-072", "unhappy (boundary)", hidden, 0, notEnemy, 30, hidden, nil},
		{"undefined finder", "FR-072", "unhappy (invalid input)", hidden, 0, 0, 30, hidden, nil},
		{"enemy on not hidden", "FR-071", "unhappy (state error)", notHidden, 0, finder, 30, notHidden, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var got combat.HidingChange
			if c.activity != 0 {
				got = hideR.ApplyActivity(c.status, c.activity, gid("hider"))
			} else {
				got = hideR.AttemptToFind(c.status, gid("hider"), c.finder, c.total)
			}
			if got.Status() != c.want || !slices.Equal(got.Events(), c.events) || got.Events() == nil {
				t.Errorf("change = %v %v, want %v %v [%s]", got.Status(), got.Events(), c.want, c.events, c.requirement)
			}
		})
	}
	// FR-068: the status the caller read before applying stays Hidden.
	before := hidden
	hideR.ApplyActivity(before, combat.HidingActivityAttackRoll, gid("hider"))
	if before != hidden {
		t.Error("ApplyActivity changed the status passed to it [FR-068, FR-073]")
	}
	// FR-071: ended once only.
	first := hideR.ApplyActivity(hidden, combat.HidingActivityAttackRoll, gid("hider"))
	second := hideR.ApplyActivity(first.Status(), combat.HidingActivityAttackRoll, gid("hider"))
	if len(first.Events())+len(second.Events()) != 1 {
		t.Error("hiding ended more than once [FR-071]")
	}
	// A blank hider cannot end hiding: no event can be built, so the status is unchanged.
	if got := hideR.ApplyActivity(hidden, combat.HidingActivityAttackRoll, ""); got.Status() != hidden || len(got.Events()) != 0 {
		t.Errorf("blank hider change = %v", got)
	}
	hc := hideR.ApplyActivity(hidden, combat.HidingActivityAttackRoll, gid("hider"))
	hc.Events()[0] = nil
	if hc.Events()[0] == nil {
		t.Error("HidingChange.Events is not a copy [FR-100]")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-102 (4.10a "every activity") - each of the seven activities has exactly
// one EffectOf, an undefined one has none, and ApplyActivity on a Hidden hider agrees with it.
// Path: unhappy (rule interaction).
func TestEffectOfAgreesWithApplyActivity(t *testing.T) {
	t.Parallel()
	hidden := combat.HidingStatus(combat.HidingStatusHidden{State: mustHidden(t, 16)})
	stops := map[combat.HidingActivity]bool{
		combat.HidingActivitySoundLouderThanWhisper: true, combat.HidingActivityFoundByEnemy: true,
		combat.HidingActivityAttackRoll: true, combat.HidingActivitySpellWithVerbalComponent: true,
	}
	for a := combat.HidingActivitySoundLouderThanWhisper; a <= combat.HidingActivitySpellWithoutVerbalComponent; a++ {
		want := combat.HidingEffectRemainsHidden
		if stops[a] {
			want = combat.HidingEffectStopsBeingHidden
		}
		ended := hideR.ApplyActivity(hidden, a, gid("hider")).Status() != hidden
		if got := hideR.EffectOf(a); got != want || ended != stops[a] || !a.Valid() || a.String() == "unknown" {
			t.Errorf("%v: EffectOf = %v, ended = %v", a, got, ended)
		}
	}
	if hideR.EffectOf(0) != 0 || hideR.EffectOf(8) != 0 || combat.HidingActivity(0).Valid() || combat.HidingActivity(8).Valid() {
		t.Error("an undefined activity has an effect")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-100, FR-073 - the lists of an assessment and of a HideScene are copied: changing
// the caller's slices after construction, or a returned list, changes nothing.
// Path: unhappy (immutability).
func TestHideScenesAndAssessmentsAreCopies(t *testing.T) {
	t.Parallel()
	cover := cvr{combat.HideCoverDegreeTotalCover}
	enemies := []combat.EnemySighting{enemy("e1", combat.LineOfSightBlocked, combat.SightRelationCanSee)}
	scene, err := combat.NewHideScene(gid("hider"), content.ObscurementNone, cover, false, enemies)
	if err != nil {
		t.Fatal(err)
	}
	cover[0], enemies[0] = combat.HideCoverDegreeHalfCover, enemy("x", combat.LineOfSightClear, combat.SightRelationCannotSee)
	a, ok := hideR.Assess(scene).(combat.HideAvailable)
	if !ok || len(a.DiscernedSightings()) != 1 || a.DiscernedSightings()[0].Enemy != gid("e1") {
		t.Fatalf("assessment changed with the caller's slices: %+v", a)
	}
	a.DiscernedSightings()[0].Enemy = "z"
	if a.DiscernedSightings()[0].Enemy != gid("e1") {
		t.Error("DiscernedSightings is not a copy")
	}
	blocked := hideR.Assess(hideScene(t, content.ObscurementNone, nil, false, enemy("e1", combat.LineOfSightClear, combat.SightRelationCanSee))).(combat.HideBlocked)
	blocked.Reasons()[0], blocked.EnemiesWithLineOfSight()[0], blocked.DiscernedSightings()[0].Enemy = 0, "z", "z"
	if blocked.Reasons()[0] == 0 || blocked.EnemiesWithLineOfSight()[0] != gid("e1") || blocked.DiscernedSightings()[0].Enemy != gid("e1") {
		t.Error("HideBlocked lists are not copies")
	}
	if hideR.Assess(scene) == nil || hideR.Assess(hideScene(t, content.ObscurementHeavily, nil, false)).(combat.HideAvailable).DiscernedSightings() == nil {
		t.Error("empty lists are returned as nil")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md section 5 - the hiding enums print their snake_case names.
// Path: unhappy (boundary).
func TestHidingEnumStrings(t *testing.T) {
	t.Parallel()
	got := []string{combat.HideCoverDegreeHalfCover.String(), combat.HideCoverDegreeTotalCover.String(), combat.HideCoverDegree(0).String(),
		combat.HideBlockNotConcealed.String(), combat.HideBlockInEnemyLineOfSight.String(), combat.HideBlock(0).String(),
		combat.HidingActivityAttackRoll.String(), combat.HidingActivity(0).String()}
	want := []string{"half_cover", "total_cover", "unknown", "not_concealed", "in_enemy_line_of_sight", "unknown", "attack_roll", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("strings = %v, want %v", got, want)
	}
}

func FuzzNewHideScene(f *testing.F) {
	f.Add("hider", 1, 1, "enemy", 1, 1)
	f.Add("", 0, 0, "", 0, 0)
	f.Add("h", 9, 9, "e", 9, 9)
	f.Fuzz(func(t *testing.T, hider string, o, cover int, enemyID string, line, sees int) {
		s, err := combat.NewHideScene(combat.CombatantID(hider), content.Obscurement(o), cvr{combat.HideCoverDegree(cover)}, false,
			[]combat.EnemySighting{{Enemy: combat.CombatantID(enemyID), Line: combat.LineOfSight(line), HiderSeesEnemy: combat.SightRelation(sees)}})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		switch hideR.Assess(s).(type) {
		case combat.HideAvailable, combat.HideBlocked:
		default:
			t.Fatal("Assess returned neither variant")
		}
	})
}
