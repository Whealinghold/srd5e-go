package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

func mtUF(tb testing.TB) combat.UndeadFortitudeTrait {
	tb.Helper()
	return luMust(combat.NewUndeadFortitudeTrait(mtRef(tb, 0)))
}

func mtFacts(tb testing.TB, taken int, hit combat.HitKind, types ...content.DamageType) combat.UndeadFortitudeFacts {
	tb.Helper()
	return luMust(combat.NewUndeadFortitudeFacts(taken, types, hit))
}

func mtSlash(n int) combat.Damage { return combat.MustDamage(n, content.DamageSlashing) }

// Requirement: doc/specs/monster-traits.md FR-031 to FR-033, table UF-1 - the assessment of each case row; before is 10 of 10 Hit Points, DiesInstantly.
// Path: per case.
func TestAssessUndeadFortitude(t *testing.T) {
	t.Parallel()
	slash, rad := content.DamageSlashing, content.DamageRadiant
	cases := []struct {
		name, requirement, path string
		temp                    int
		damages                 []combat.Damage
		facts                   combat.UndeadFortitudeFacts
		want                    combat.UndeadFortitudeCheck
	}{
		{"ordinary drop", "FR-033", "happy", 0, []combat.Damage{mtSlash(10)}, mtFacts(t, 10, combat.NormalHit, slash), required(15)},
		{"overkill", "FR-033", "unhappy (boundary)", 0, []combat.Damage{mtSlash(17)}, mtFacts(t, 10, combat.NormalHit, slash), required(15)},
		{"above zero", "FR-031", "unhappy (boundary)", 0, []combat.Damage{mtSlash(9)}, mtFacts(t, 9, combat.NormalHit, slash), exempt(combat.UndeadFortitudeNotReducedToZero)},
		{"Radiant", "FR-032", "unhappy (rule interaction)", 0, []combat.Damage{combat.MustDamage(10, rad)}, mtFacts(t, 10, combat.NormalHit, rad), exempt(combat.UndeadFortitudeRadiantDamage)},
		{"mixed with Radiant", "FR-032", "unhappy (rule interaction)", 0, []combat.Damage{mtSlash(6), combat.MustDamage(4, rad)}, mtFacts(t, 10, combat.NormalHit, slash, rad), exempt(combat.UndeadFortitudeRadiantDamage)},
		{"Radiant fully Immune", "FR-030", "unhappy (rule interaction)", 0, []combat.Damage{mtSlash(10)}, mtFacts(t, 10, combat.NormalHit, slash), required(15)},
		{"Critical Hit", "FR-032", "unhappy (rule interaction)", 0, []combat.Damage{mtSlash(10)}, mtFacts(t, 10, combat.CriticalHit, slash), exempt(combat.UndeadFortitudeCriticalHit)},
		{"critical and Radiant", "FR-032", "unhappy (rule interaction)", 0, []combat.Damage{combat.MustDamage(10, rad)}, mtFacts(t, 10, combat.CriticalHit, rad), exempt(combat.UndeadFortitudeCriticalHit)},
		{"Temporary Hit Points", "FR-033", "unhappy (rule interaction)", 5, []combat.Damage{mtSlash(17)}, mtFacts(t, 15, combat.NormalHit, slash), required(20)},
		{"temporary absorbs all", "FR-031", "unhappy (boundary)", 5, []combat.Damage{mtSlash(5)}, mtFacts(t, 5, combat.NormalHit, slash), exempt(combat.UndeadFortitudeNotReducedToZero)},
		{"zero facts", "FR-031", "unhappy (invalid input)", 0, []combat.Damage{mtSlash(10)}, combat.UndeadFortitudeFacts{}, exempt(combat.UndeadFortitudeNotReducedToZero)},
	}
	for _, c := range cases {
		s := status(t, 10, 10, c.temp, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
		tr := mtHit(t, s, combat.NormalHit, c.damages...)
		if got := combat.AssessUndeadFortitude(mtUF(t), tr, c.facts); got != c.want {
			t.Errorf("%s: got %#v, want %#v [%s%s, %s]", c.name, got, c.want, traitsSpec, c.requirement, c.path)
		}
	}
	// A zero trait never triggers it.
	tr := mtHit(t, mtZombie(t, 10, 10, 0, combat.DiesInstantly), combat.NormalHit, mtSlash(10))
	if got := combat.AssessUndeadFortitude(combat.UndeadFortitudeTrait{}, tr, mtFacts(t, 10, combat.NormalHit, content.DamageSlashing)); got != exempt(combat.UndeadFortitudeNotReducedToZero) {
		t.Errorf("zero trait: %#v", got)
	}
}

func required(dc int) combat.UndeadFortitudeCheck {
	return combat.UndeadFortitudeSaveRequired{DC: d20.DifficultyClass(dc), Ability: ability.Constitution}
}

func exempt(r combat.UndeadFortitudeExemption) combat.UndeadFortitudeCheck {
	return combat.UndeadFortitudeNotTriggered{Reason: r}
}

// Requirement: doc/specs/monster-traits.md FR-031 - damage at 0 Hit Points (a Dying creature) and a knock-out never trigger it.
// Path: unhappy (rule interaction).
func TestUndeadFortitudeNeedsADropToZero(t *testing.T) {
	t.Parallel()
	facts := mtFacts(t, 10, combat.NormalHit, content.DamageSlashing)
	dyingHit := mtHit(t, dyingStatus(t, 10, 0, 0), combat.NormalHit, mtSlash(3))
	if got := combat.AssessUndeadFortitude(mtUF(t), dyingHit, facts); got != exempt(combat.UndeadFortitudeNotReducedToZero) {
		t.Errorf("damage at 0: %#v", got)
	}
	tr, err := mtZombie(t, 10, 10, 0, combat.DiesInstantly).ApplyDamage(mtID(), []combat.Damage{mtSlash(10)}, nil, combat.MeleeAttackKnockOut, combat.NormalHit, nil)
	if got := combat.AssessUndeadFortitude(mtUF(t), applied(t, tr, err), facts); got != exempt(combat.UndeadFortitudeNotReducedToZero) {
		t.Errorf("knock-out: %#v", got)
	}
}

// Requirement: doc/specs/monster-traits.md FR-030 - NewUndeadFortitudeFacts rejects a damage below 1, an empty list, a bad Damage Type or hit kind; Immunity leaves the type out.
// Path: unhappy (invalid input).
func TestNewUndeadFortitudeFacts(t *testing.T) {
	t.Parallel()
	slash := []content.DamageType{content.DamageSlashing}
	cases := []struct {
		name  string
		taken int
		types []content.DamageType
		hit   combat.HitKind
		ok    bool
	}{
		{"valid", 1, slash, combat.NormalHit, true},
		{"damage 0", 0, slash, combat.NormalHit, false},
		{"negative damage", -1, slash, combat.NormalHit, false},
		{"empty types", 5, nil, combat.NormalHit, false},
		{"zero type", 5, []content.DamageType{0}, combat.NormalHit, false},
		{"bad hit kind", 5, slash, 0, false},
	}
	for _, c := range cases {
		f, err := combat.NewUndeadFortitudeFacts(c.taken, c.types, c.hit)
		if (err == nil) != c.ok || (!c.ok && (!isInvalid(err) || !f.IsZero())) || (c.ok && f.DamageTaken() != c.taken) {
			t.Errorf("%s: %#v, %v", c.name, f, err)
		}
	}
	// 32-bit width: the DC saturates instead of wrapping.
	big := mtFacts(t, int(^uint(0)>>1), combat.NormalHit, content.DamageSlashing)
	tr := mtHit(t, mtZombie(t, 10, 10, 0, combat.DiesInstantly), combat.NormalHit, mtSlash(10))
	if got := combat.AssessUndeadFortitude(mtUF(t), tr, big).(combat.UndeadFortitudeSaveRequired); got.DC != d20.DifficultyClass(int(^uint(0)>>1)) {
		t.Errorf("DC = %d", got.DC)
	}
}

// Requirement: doc/specs/monster-traits.md FR-034, FR-035 - a successful save leaves 1 Hit Point Alive with the events of the table, a failed one keeps the hit's status and events.
// Path: happy.
func TestSettleUndeadFortitude(t *testing.T) {
	t.Parallel()
	before := mtZombie(t, 10, 10, 0, combat.DiesInstantly)
	tr := mtHit(t, before, combat.NormalHit, mtSlash(10))
	check := required(15)
	won, err := combat.SettleUndeadFortitude(check, before, tr, mtID(), d20.Success)
	s, ok := won.(combat.UndeadFortitudeSurvived)
	if err != nil || !ok {
		t.Fatalf("success: %#v, %v", won, err)
	}
	hp := s.Status().HitPoints()
	if hp.Current() != 1 || hp.Maximum() != 10 || hp.Temporary() != 0 || s.Status().LifeState() != (combat.Alive{}) || s.Status().ZeroHitPointsRule() != combat.DiesInstantly {
		t.Errorf("status = %+v %v", hp, s.Status().LifeState())
	}
	want := []string{"damage_taken", "dropped_to_zero_hit_points", "became_bloodied", "undead_fortitude_saved"}
	if before.HitPoints().Current() != 10 {
		t.Error("before changed")
	}
	// Bloodied at 10 of 10? 1 of 10 is Bloodied and 10 of 10 is not.
	if got := mtKindsOf(s.Events()); !slices.Equal(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
	lost, err := combat.SettleUndeadFortitude(check, before, tr, mtID(), d20.Failure)
	f, ok := lost.(combat.UndeadFortitudeFell)
	if err != nil || !ok || f.Status() != tr.Status() || len(f.Events()) != len(tr.Events()) {
		t.Fatalf("failure: %#v, %v", lost, err)
	}
	if got := mtKindsOf(f.Events()); got[len(got)-1] != "died" || f.Events()[len(got)-1].(combat.Died).Cause() != combat.MonsterDeath {
		t.Errorf("failure events = %v", got)
	}
	e := f.Events()
	e[0] = nil
	if f.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}

// Requirement: doc/specs/monster-traits.md FR-034, FR-037 - a FallsUnconscious creature taken to Dying is Alive at 1 Hit Point with no FellUnconscious and one BecameBloodied; an already Bloodied creature gets none.
// Path: unhappy (rule interaction).
func TestSettleUndeadFortitudeDyingAndBloodied(t *testing.T) {
	t.Parallel()
	before := mtZombie(t, 10, 10, 0, combat.FallsUnconscious)
	tr := mtHit(t, before, combat.NormalHit, mtSlash(10))
	if _, ok := tr.Status().LifeState().(combat.Dying); !ok {
		t.Fatalf("status = %v", tr.Status().LifeState())
	}
	s := luMust(combat.SettleUndeadFortitude(required(15), before, tr, mtID(), d20.Success)).(combat.UndeadFortitudeSurvived)
	got := mtKindsOf(s.Events())
	if slices.Contains(got, "fell_unconscious") || slices.Contains(got, "died") || len(got) != 4 || s.Status().LifeState() != (combat.Alive{}) {
		t.Errorf("events = %v", got)
	}
	bloodied := mtZombie(t, 4, 10, 0, combat.DiesInstantly)
	tr = mtHit(t, bloodied, combat.NormalHit, mtSlash(4))
	s = luMust(combat.SettleUndeadFortitude(required(9), bloodied, tr, mtID(), d20.Success)).(combat.UndeadFortitudeSurvived)
	if got := mtKindsOf(s.Events()); slices.Contains(got, "became_bloodied") {
		t.Errorf("an already Bloodied creature got %v", got)
	}
	// A second drop is assessed alone.
	second := mtHit(t, s.Status(), combat.NormalHit, mtSlash(1))
	if got := combat.AssessUndeadFortitude(mtUF(t), second, mtFacts(t, 1, combat.NormalHit, content.DamageSlashing)); got != required(6) {
		t.Errorf("second drop = %#v", got)
	}
}

// Requirement: doc/specs/monster-traits.md FR-036 - settling needs a required save, a Dead or Dying transition, a before above 0, a subject and an outcome.
// Path: unhappy (state error).
func TestSettleUndeadFortitudeInvalid(t *testing.T) {
	t.Parallel()
	before := mtZombie(t, 10, 10, 0, combat.DiesInstantly)
	tr := mtHit(t, before, combat.NormalHit, mtSlash(10))
	alive := mtHit(t, before, combat.NormalHit, mtSlash(3))
	atZero := status(t, 0, 10, 0, combat.Stable{}, combat.DiesInstantly, combat.NoAffinities())
	cases := []struct {
		name   string
		check  combat.UndeadFortitudeCheck
		before combat.HitPointStatus
		tr     combat.HitPointTransitionApplied
		who    combat.CombatantID
		out    d20.Outcome
	}{
		{"not triggered passed", exempt(combat.UndeadFortitudeCriticalHit), before, tr, mtID(), d20.Success},
		{"nil check", nil, before, tr, mtID(), d20.Success},
		{"transition Alive", required(15), before, alive, mtID(), d20.Success},
		{"zero transition", required(15), before, combat.HitPointTransitionApplied{}, mtID(), d20.Success},
		{"before at 0", required(15), atZero, tr, mtID(), d20.Success},
		{"zero before", required(15), combat.HitPointStatus{}, tr, mtID(), d20.Success},
		{"zero subject", required(15), before, tr, "", d20.Success},
		{"zero outcome", required(15), before, tr, mtID(), 0},
		{"outcome past the last", required(15), before, tr, mtID(), 3},
	}
	for _, c := range cases {
		out, err := combat.SettleUndeadFortitude(c.check, c.before, c.tr, c.who, c.out)
		if !isInvalid(err) || out != nil {
			t.Errorf("%s: %#v, %v", c.name, out, err)
		}
	}
}
