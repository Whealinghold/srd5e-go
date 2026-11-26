package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func msTwoSizes(f *content.MonsterFields) {
	f.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeSmall}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-002, FR-003, FR-004 - a monster with one size option takes it when none is given and accepts it when given; with several options and none given the setup is rejected with SizeChoiceRequired (the engine never picks); a size outside the options, any other size for a single-option monster included, is SizeNotOffered (SRD 5.2.1, "Size", p. 254; "Assassin", p. 260).
// Path: per case.
func TestNewMonsterSetupSize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		monster                 content.Monster
		options                 []combat.SetupOption
		want                    content.CreatureSize
		wantProblems            []combat.MonsterSetupProblem
	}{
		{"one option, none given", "FR-002", "happy", msMonster(t, nil), nil, content.SizeLarge, nil},
		{"one option, equal given", "FR-002", "happy", msMonster(t, nil), []combat.SetupOption{combat.WithSize(content.SizeLarge)}, content.SizeLarge, nil},
		{"several options, none given", "FR-003", "unhappy (invalid input)", msMonster(t, msTwoSizes), nil, 0, []combat.MonsterSetupProblem{combat.SizeChoiceRequired}},
		{"several options, the first", "FR-003", "happy", msMonster(t, msTwoSizes), []combat.SetupOption{combat.WithSize(content.SizeMedium)}, content.SizeMedium, nil},
		{"several options, the second", "FR-003", "happy", msMonster(t, msTwoSizes), []combat.SetupOption{combat.WithSize(content.SizeSmall)}, content.SizeSmall, nil},
		{"several options, another size", "FR-004", "unhappy (boundary)", msMonster(t, msTwoSizes), []combat.SetupOption{combat.WithSize(content.SizeLarge)}, 0, []combat.MonsterSetupProblem{combat.SizeNotOffered}},
		{"one option, another size", "FR-004", "unhappy (boundary)", msMonster(t, nil), []combat.SetupOption{combat.WithSize(content.SizeHuge)}, 0, []combat.MonsterSetupProblem{combat.SizeNotOffered}},
		{"a size that is not a size", "FR-004", "unhappy (invalid input)", msMonster(t, nil), []combat.SetupOption{combat.WithSize(content.CreatureSize(99))}, 0, []combat.MonsterSetupProblem{combat.SizeNotOffered}},
		{"a negative size", "FR-004", "unhappy (invalid input)", msMonster(t, nil), []combat.SetupOption{combat.WithSize(content.CreatureSize(-1))}, 0, []combat.MonsterSetupProblem{combat.SizeNotOffered}},
		{"a nil option is ignored", "FR-001", "unhappy (empty and minimal)", msMonster(t, nil), []combat.SetupOption{nil}, content.SizeLarge, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := setupSpec + c.requirement
			got, err := combat.NewMonsterSetup(c.monster, combat.PrintedAverageHitPoints{}, c.options...)
			if c.wantProblems != nil {
				var rej *combat.MonsterSetupError
				if !errors.As(err, &rej) || !slices.Equal(rej.Problems(), c.wantProblems) || !got.IsZero() {
					t.Fatalf("NewMonsterSetup = %v, %v, want a rejection with %v [%s]", got, err, c.wantProblems, req)
				}
				return
			}
			if err != nil || got.Size() != c.want || got.IsZero() {
				t.Errorf("NewMonsterSetup size = %v, %v, want %v [%s]", got.Size(), err, c.want, req)
			}
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-005 - every problem found is reported once, in the order of the enum, in one *MonsterSetupError that is never empty, matches srderr.ErrInvalidInput and equals another rejection by problems; Problems returns a copy.
// Path: unhappy (invalid input).
func TestNewMonsterSetupReportsEveryProblemOnceInOrder(t *testing.T) {
	t.Parallel()
	wrong := msRolled(t, dice.MustExpression(2, dice.D6(), 0), 1, 1)
	cases := []struct {
		name    string
		monster content.Monster
		hp      combat.ResolvedMonsterHitPoints
		options []combat.SetupOption
		want    []combat.MonsterSetupProblem
	}{
		{"size required and a wrong roll", msMonster(t, msTwoSizes), wrong, nil, []combat.MonsterSetupProblem{combat.SizeChoiceRequired, combat.RolledExpressionMismatch}},
		{"size not offered and a wrong roll", msMonster(t, nil), wrong, []combat.SetupOption{combat.WithSize(content.SizeTiny)}, []combat.MonsterSetupProblem{combat.SizeNotOffered, combat.RolledExpressionMismatch}},
		{"the zero monster, no choice, an undefined rule", content.Monster{}, nil, []combat.SetupOption{combat.WithZeroHitPointsRule(99)},
			[]combat.MonsterSetupProblem{combat.ContentNumberOutOfRange, combat.SetupArgumentInvalid}},
		{"an undefined initiative mode", msMonster(t, nil), combat.PrintedAverageHitPoints{}, []combat.SetupOption{combat.WithInitiativeMode(7)}, []combat.MonsterSetupProblem{combat.SetupArgumentInvalid}},
		{"every problem", msMonster(t, msTwoSizes), wrong, []combat.SetupOption{combat.WithInitiativeMode(7)},
			[]combat.MonsterSetupProblem{combat.SizeChoiceRequired, combat.RolledExpressionMismatch, combat.SetupArgumentInvalid}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewMonsterSetup(c.monster, c.hp, c.options...)
			var rej *combat.MonsterSetupError
			if !errors.As(err, &rej) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want a *MonsterSetupError matching ErrInvalidInput", err)
			}
			got := rej.Problems()
			if !slices.Equal(got, c.want) || len(got) == 0 {
				t.Errorf("problems = %v, want %v, once each in enum order", got, c.want)
			}
			got[0] = 0
			if again := rej.Problems(); !slices.Equal(again, c.want) {
				t.Errorf("Problems is not a copy: %v", again)
			}
			_, err2 := combat.NewMonsterSetup(c.monster, c.hp, c.options...)
			if !errors.Is(err, err2) {
				t.Error("two rejections with the same problems do not match each other")
			}
		})
	}
	_, a := combat.NewMonsterSetup(msMonster(t, msTwoSizes), combat.PrintedAverageHitPoints{})
	_, b := combat.NewMonsterSetup(msMonster(t, nil), combat.PrintedAverageHitPoints{}, combat.WithSize(content.SizeTiny))
	if errors.Is(a, b) {
		t.Error("rejections with different problems match")
	}
	var nilRejection *combat.MonsterSetupError
	if errors.Is(a, nilRejection) || errors.Is(a, errors.New("other")) {
		t.Error("a rejection matches a nil rejection or an unrelated error")
	}
	for _, p := range []combat.MonsterSetupProblem{combat.SizeChoiceRequired, combat.SizeNotOffered, combat.RolledExpressionMismatch, combat.ContentNumberOutOfRange, combat.SetupArgumentInvalid, 0} {
		if p.String() == "" {
			t.Errorf("problem %d has no name", int(p))
		}
	}
	if combat.MonsterSetupProblem(0).String() != "unknown" || combat.SizeNotOffered.String() != "size_not_offered" {
		t.Error("problem names changed")
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-001, FR-005 - MustMonsterSetup returns the setup NewMonsterSetup builds and panics, with the error NewMonsterSetup returns, exactly when that rejects.
// Path: unhappy (invalid input).
func TestMustMonsterSetupPanicsIfAndOnlyIfNewRejects(t *testing.T) {
	t.Parallel()
	if got, want := combat.MustMonsterSetup(msMonster(t, nil), combat.PrintedAverageHitPoints{}), msSetup(t, msMonster(t, nil)); !got.Equal(want) {
		t.Error("MustMonsterSetup differs from NewMonsterSetup")
	}
	defer func() {
		r := recover()
		err, ok := r.(error)
		var rej *combat.MonsterSetupError
		if !ok || !errors.As(err, &rej) {
			t.Errorf("panic value = %v, want the *MonsterSetupError", r)
		}
	}()
	combat.MustMonsterSetup(msMonster(t, msTwoSizes), combat.PrintedAverageHitPoints{})
}

// Requirement: doc/specs/monster-combatant-setup.md FR-006, FR-001 - Ref returns the id with a MonsterCombatant of the stat block's id; the setup holds no id, so one setup serves several refs; the accessors return what was given.
// Path: happy.
func TestMonsterSetupRefAndAccessors(t *testing.T) {
	t.Parallel()
	m := msMonster(t, nil)
	s := msSetup(t, m)
	a, b := combat.MustCombatantID("a"), combat.MustCombatantID("b")
	for _, id := range []combat.CombatantID{a, b} {
		got := s.Ref(id)
		if got != (combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: m.ID()}}) {
			t.Errorf("Ref(%v) = %+v", id, got)
		}
	}
	if !s.Monster().Equal(m) || s.IsZero() || !(combat.MonsterSetup{}).IsZero() {
		t.Errorf("Monster/IsZero wrong for %v", s)
	}
	if s.ProficiencyBonus().Value() != 4 {
		t.Errorf("ProficiencyBonus = %v, want +4 for Challenge Rating 10", s.ProficiencyBonus())
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-001 - Equal compares the monster, the size, the Hit Points, the raised flag and the Initiative plan; setups that differ in one of them are unequal and two builds of the same inputs are equal.
// Path: per case.
func TestMonsterSetupEqual(t *testing.T) {
	t.Parallel()
	bat := dice.MustExpression(1, dice.D4(), -1)
	base := msSetup(t, msMonster(t, msTwoSizes), combat.WithSize(content.SizeSmall))
	cases := []struct {
		name  string
		other combat.MonsterSetup
		want  bool
	}{
		{"same inputs", msSetup(t, msMonster(t, msTwoSizes), combat.WithSize(content.SizeSmall)), true},
		{"another size", msSetup(t, msMonster(t, msTwoSizes), combat.WithSize(content.SizeMedium)), false},
		{"another monster", msSetup(t, msMonster(t, func(f *content.MonsterFields) { msTwoSizes(f); f.ArmorClass = 18 }), combat.WithSize(content.SizeSmall)), false},
		{"another initiative mode", msSetup(t, msMonster(t, msTwoSizes), combat.WithSize(content.SizeSmall), combat.WithInitiativeMode(combat.MonsterInitiativeFixed)), false},
		{"another Hit Points", msSetup(t, msMonster(t, msTwoSizes), combat.WithSize(content.SizeSmall), combat.WithExtraAffinities(combat.ResistantToAllDamage())), false},
	}
	for _, c := range cases {
		if got := base.Equal(c.other); got != c.want {
			t.Errorf("%s: Equal = %v, want %v", c.name, got, c.want)
		}
		if got := c.other.Equal(base); got != c.want {
			t.Errorf("%s: reversed Equal = %v, want %v", c.name, got, c.want)
		}
	}
	raised := luMust(combat.NewMonsterSetup(msHitPoints(t, 1, bat), msRolled(t, bat, 1)))
	notRaised := luMust(combat.NewMonsterSetup(msHitPoints(t, 1, bat), msRolled(t, bat, 2)))
	if raised.HitPointStatus() != notRaised.HitPointStatus() || raised.Equal(notRaised) {
		t.Error("setups that differ only in whether the total was raised to 1 are equal")
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-023, FR-035, FR-047 - the unapplied lists are copies: changing a returned slice does not change the setup, which holds content value objects only.
// Path: unhappy (immutability).
func TestMonsterSetupReturnedSlicesAreCopies(t *testing.T) {
	t.Parallel()
	m := msMonster(t, func(f *content.MonsterFields) {
		f.Resistances = []content.DefenseEntry{msDescribed("Chosen type")}
		f.Speeds = append(f.Speeds, msSpeed(40, "form only", content.SpeedWalk))
	})
	s := msSetup(t, m)
	defenses, speeds := s.UnappliedDefenses(), s.UnappliedSpeeds()
	if len(defenses) != 1 || len(speeds) != 1 {
		t.Fatalf("unapplied = %v, %v, want one each", defenses, speeds)
	}
	defenses[0], speeds[0] = combat.UnappliedDefense{}, combat.UnappliedSpeed{}
	if !slices.Equal(s.UnappliedDefenses(), msSetup(t, m).UnappliedDefenses()) || !s.Equal(msSetup(t, m)) {
		t.Error("changing a returned slice changed the setup")
	}
}

// Requirement: doc/rules/testing.md section 10 - NewMonsterSetup never panics and either rejects with a typed error or builds a setup that satisfies its own invariants, for any size, Hit Points choice, roll faces and option values.
// Path: unhappy (invalid input).
func FuzzNewMonsterSetup(f *testing.F) {
	f.Add(0, 0, 1, 1, 0)
	f.Add(3, 1, 1, 2, 1)
	f.Add(-1, 2, 99, 99, 99)
	f.Add(4, 0, 0, 0, 3)
	f.Fuzz(func(t *testing.T, size, method, rule, mode, sizes int) {
		m := msMonster(t, func(f *content.MonsterFields) {
			if sizes%2 != 0 {
				msTwoSizes(f)
			}
		})
		var hp combat.ResolvedMonsterHitPoints
		switch method % 3 {
		case 0:
			hp = combat.PrintedAverageHitPoints{}
		case 1:
			hp = combat.RolledMonsterHitPoints{Roll: luMust(dicetest.NewMinimumRoller().Roll(m.HitPoints().Dice()))}
		}
		s, err := combat.NewMonsterSetup(m, hp, combat.WithSize(content.CreatureSize(size)),
			combat.WithZeroHitPointsRule(combat.ZeroHitPointsRule(rule)), combat.WithInitiativeMode(combat.MonsterInitiativeMode(mode)))
		if err != nil {
			var rej *combat.MonsterSetupError
			if !errors.As(err, &rej) || len(rej.Problems()) == 0 || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want a non-empty *MonsterSetupError", err)
			}
			return
		}
		if !slices.Contains(m.Sizes(), s.Size()) || s.HitPointStatus().HitPoints().Maximum() < 1 {
			t.Fatalf("setup %+v breaks its invariants", s)
		}
	})
}
