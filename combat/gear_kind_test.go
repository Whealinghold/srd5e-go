package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

const gearSpec = "doc/specs/deployed-and-restraining-gear.md "

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-001 - DeployedGearKind has exactly four members, HoldKind five and BindingKind three, so no other kind can be expressed (SRD 5.2.1, "Adventuring Gear", pp. 96-99).
// Path: unhappy (invalid input made unrepresentable).
func TestGearKindMemberSets(t *testing.T) {
	t.Parallel()
	for v := -1; v <= 7; v++ {
		if got, want := combat.DeployedGearKind(v).Valid(), v >= 1 && v <= 4; got != want {
			t.Errorf("DeployedGearKind(%d).Valid() = %v, want %v", v, got, want)
		}
		if got, want := combat.HoldKind(v).Valid(), v >= 1 && v <= 5; got != want {
			t.Errorf("HoldKind(%d).Valid() = %v, want %v", v, got, want)
		}
		if got, want := combat.BindingKind(v).Valid(), v >= 1 && v <= 3; got != want {
			t.Errorf("BindingKind(%d).Valid() = %v, want %v", v, got, want)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-003, FR-066, FR-067 - every kind has a distinct valid ItemID, and the snake_case String that is the value of srd5e.combat.gear.kind; an invalid kind has neither.
// Path: happy; unhappy (invalid input).
func TestGearKindItemIDsAndNames(t *testing.T) {
	t.Parallel()
	deployed := map[combat.DeployedGearKind][2]string{combat.DeployedGearKindCaltrops: {"caltrops", "caltrops"},
		combat.DeployedGearKindBallBearings: {"ball_bearings", "ball_bearings"}, combat.DeployedGearKindPouredOil: {"oil", "poured_oil"},
		combat.DeployedGearKindHuntingTrap: {"hunting_trap", "hunting_trap"}}
	for k, w := range deployed {
		if k.ItemID().String() != w[0] || !k.ItemID().Valid() || k.String() != w[1] {
			t.Errorf("kind %d: ItemID %q, String %q; want %q, %q", k, k.ItemID(), k, w[0], w[1])
		}
	}
	holds := map[combat.HoldKind]string{combat.HoldKindNet: "net", combat.HoldKindChain: "chain", combat.HoldKindManacles: "manacles",
		combat.HoldKindRope: "rope", combat.HoldKindHuntingTrap: "hunting_trap"}
	for k, w := range holds {
		if k.ItemID().String() != w || !k.ItemID().Valid() || k.String() != w {
			t.Errorf("hold kind %d: ItemID %q, String %q; want %q", k, k.ItemID(), k, w)
		}
	}
	if combat.DeployedGearKind(0).ItemID() != "" || combat.HoldKind(9).ItemID() != "" || combat.HoldKind(0).String() != "unknown" || combat.DeployedGearKind(5).String() != "unknown" {
		t.Error("an invalid kind must have no ItemID and the name unknown")
	}
	for b, h := range map[combat.BindingKind]combat.HoldKind{combat.BindingKindChain: combat.HoldKindChain,
		combat.BindingKindManacles: combat.HoldKindManacles, combat.BindingKindRope: combat.HoldKindRope, 0: 0, 4: 0} {
		if b.HoldKind() != h {
			t.Errorf("BindingKind(%d).HoldKind() = %v, want %v", b, b.HoldKind(), h)
		}
	}
	if combat.BindingKindRope.String() != "rope" || combat.BindingKind(0).String() != "unknown" {
		t.Error("BindingKind.String")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md section 3 - the small enums are valid for their members only and name them in snake_case.
// Path: unhappy (invalid input).
func TestGearEnums(t *testing.T) {
	t.Parallel()
	type enum struct {
		name  string
		valid func(int) bool
		str   func(int) string
		want  []string
	}
	enums := []enum{
		{"GroundLevel", func(v int) bool { return combat.GroundLevel(v).Valid() }, func(v int) string { return combat.GroundLevel(v).String() }, []string{"level", "not_level"}},
		{"AreaTrigger", func(v int) bool { return combat.AreaTrigger(v).Valid() }, func(v int) string { return combat.AreaTrigger(v).String() }, []string{"enters", "ends_turn"}},
		{"HoldRestraint", func(v int) bool { return combat.HoldRestraint(v).Valid() }, func(v int) string { return combat.HoldRestraint(v).String() }, []string{"restrains", "does_not_restrain"}},
		{"HoldReleaseMethod", func(v int) bool { return combat.HoldReleaseMethod(v).Valid() }, func(v int) string { return combat.HoldReleaseMethod(v).String() }, []string{"escape", "burst", "pick_lock", "key"}},
		{"HoldEndCause", func(v int) bool { return combat.HoldEndCause(v).Valid() }, func(v int) string { return combat.HoldEndCause(v).String() }, []string{"escaped", "burst", "destroyed", "unlocked"}},
		{"Willingness", func(v int) bool { return combat.Willingness(v).Valid() }, func(v int) string { return combat.Willingness(v).String() }, []string{"willing", "unwilling"}},
		{"ThrowerSight", func(v int) bool { return combat.ThrowerSight(v).Valid() }, func(v int) string { return combat.ThrowerSight(v).String() }, []string{"sees_target", "cannot_see_target"}},
	}
	for _, e := range enums {
		for v := -1; v <= len(e.want)+1; v++ {
			in := v >= 1 && v <= len(e.want)
			if e.valid(v) != in {
				t.Errorf("%s(%d).Valid() = %v, want %v", e.name, v, e.valid(v), in)
			}
			want := "unknown"
			if in {
				want = e.want[v-1]
			}
			if e.str(v) != want {
				t.Errorf("%s(%d).String() = %q, want %q", e.name, v, e.str(v), want)
			}
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-002, FR-006, FR-008, table 4.9a - DeployedGearProfileFor returns the facts of the table as typed values: reach, area side, level ground, saving throw and recovery; an invalid kind has the zero profile (SRD 5.2.1, pp. 96-98).
// Path: per case.
func TestDeployedGearProfiles(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		kind       combat.DeployedGearKind
		reach      int
		stated     bool
		side       int
		level      bool
		dc         d20.DifficultyClass
		recovery   int
	}{
		{"caltrops", "happy", combat.DeployedGearKindCaltrops, 5, true, 5, false, 15, 10},
		{"ball bearings", "happy", combat.DeployedGearKindBallBearings, 10, true, 10, true, 10, 10},
		{"poured oil has no save", "happy", combat.DeployedGearKindPouredOil, 5, true, 5, true, 0, 0},
		{"hunting trap has no area and an unstated reach", "unhappy (boundary)", combat.DeployedGearKindHuntingTrap, 0, false, 0, false, 13, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := combat.DeployedGearProfileFor(c.kind)
			feet, stated := p.Reach().Feet()
			dc, hasSave := p.SaveDifficultyClass()
			ab, hasAbility := p.SaveAbility()
			if p.Kind() != c.kind || feet != c.reach || stated != c.stated || p.AreaSideFeet() != c.side || p.RequiresLevelGround() != c.level ||
				dc != c.dc || hasSave != (c.dc != 0) || p.RecoveryMinutes() != c.recovery || hasAbility != hasSave || ab != ability.Dexterity {
				t.Errorf("profile = %+v, want reach %d/%v side %d level %v DC %d recovery %d [%s FR-002]", p, c.reach, c.stated, c.side, c.level, c.dc, c.recovery, gearSpec)
			}
		})
	}
	if combat.DeployedGearProfileFor(0) != (combat.DeployedGearProfile{}) {
		t.Error("an invalid kind must give the zero profile")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-006 - ReachWithin(n) allows 0 to n and refuses n + 1, ReachUnstated allows any distance, and the zero reach is unstated.
// Path: unhappy (boundary).
func TestDeployReach(t *testing.T) {
	t.Parallel()
	r := combat.ReachWithin(5)
	if !r.Allows(0) || !r.Allows(5) || r.Allows(6) {
		t.Error("ReachWithin(5) must allow 0 and 5 and refuse 6")
	}
	if u := combat.ReachUnstated(); !u.Allows(1<<30) || !u.Allows(0) {
		t.Error("ReachUnstated must allow any distance")
	}
	if z := (combat.DeployReach{}); z != combat.ReachUnstated() || !z.Allows(1) {
		t.Error("the zero reach must be unstated")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-002, FR-050, FR-051, FR-052, table 4.9b - HoldProfileFor returns, as typed values, every check (skill, Difficulty Class) of the table, held exactly as printed, and whether a helper may release; the ability is Skill.Ability (SRD 5.2.1, pp. 96-99).
// Path: per case.
func TestHoldProfiles(t *testing.T) {
	t.Parallel()
	type check struct {
		skill content.Skill
		dc    d20.DifficultyClass
	}
	cases := []struct {
		name, path string
		kind       combat.HoldKind
		bind       *check
		escape     check
		burst      *check
		helper     bool
		limit      int
	}{
		{"net", "happy", combat.HoldKindNet, nil, check{content.SkillAthletics, 10}, nil, true, 0},
		{"chain", "happy", combat.HoldKindChain, &check{content.SkillAthletics, 13}, check{content.SkillAcrobatics, 18}, &check{content.SkillAthletics, 20}, false, 0},
		{"manacles", "happy", combat.HoldKindManacles, &check{content.SkillSleightOfHand, 13}, check{content.SkillSleightOfHand, 20}, &check{content.SkillAthletics, 25}, false, 0},
		{"rope", "happy", combat.HoldKindRope, &check{content.SkillSleightOfHand, 10}, check{content.SkillAcrobatics, 15}, &check{content.SkillAthletics, 20}, false, 0},
		{"hunting trap", "unhappy (boundary)", combat.HoldKindHuntingTrap, nil, check{content.SkillAthletics, 13}, nil, true, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := combat.HoldProfileFor(c.kind)
			bind, hasBind := p.BindCheck()
			burst, hasBurst := p.BurstCheck()
			esc := p.EscapeCheck()
			if p.Kind() != c.kind || esc.Skill() != c.escape.skill || esc.DifficultyClass() != c.escape.dc || esc.Ability() != c.escape.skill.Ability() ||
				hasBind != (c.bind != nil) || hasBurst != (c.burst != nil) || p.HelperMayRelease() != c.helper || p.DefaultMovementLimitFeet() != c.limit {
				t.Errorf("profile = %+v, want escape %+v helper %v limit %d [%s FR-051]", p, c.escape, c.helper, c.limit, gearSpec)
			}
			if c.bind != nil && (bind.Skill() != c.bind.skill || bind.DifficultyClass() != c.bind.dc) {
				t.Errorf("bind check = %+v, want %+v", bind, *c.bind)
			}
			if c.burst != nil && (burst.Skill() != c.burst.skill || burst.DifficultyClass() != c.burst.dc) {
				t.Errorf("burst check = %+v, want %+v", burst, *c.burst)
			}
		})
	}
	if combat.HoldProfileFor(0) != (combat.HoldProfile{}) {
		t.Error("an invalid kind must give the zero profile")
	}
	if (combat.HoldCheck{}).Ability() != 0 {
		t.Error("the zero check has no ability")
	}
}
