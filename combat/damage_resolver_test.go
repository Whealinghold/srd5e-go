package combat_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

type pipelineRow struct {
	name, requirement, path string
	incoming                int
	typ                     content.DamageType
	adjustments             []combat.DamageAdjustment
	res                     []combat.DamageScope
	vuln, immune            []content.DamageType
	steps                   [3]int // after adjustments, resistance, vulnerability
	dealt                   int
	immuneApplied           bool
}

func fireScope() []combat.DamageScope {
	return []combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}}
}

// Requirement: doc/specs/hit-points-and-damage.md table 4.3a, FR-016 to FR-025, FR-099 - the damage
// pipeline order: flat adjustments (floored at 0), Half Damage, Resistance, Vulnerability; Immunity overrides;
// the resolution reports every step (SRD 5.2.1, "Order of Application", p. 17).
// Path: per case.
func TestDamagePipeline(t *testing.T) {
	t.Parallel()
	flat := func(n int) []combat.DamageAdjustment {
		return []combat.DamageAdjustment{combat.NewFlatAdjustment(n, nil)}
	}
	half := []combat.DamageAdjustment{combat.HalfDamage{}}
	order := `SRD 5.2.1, "Order of Application", p. 17`
	cases := []pipelineRow{
		{"worked example", order, "unhappy (rule interaction)", 28, content.DamageFire, flat(-5), fireScope(), damageTypes(content.DamageFire), nil, [3]int{23, 11, 22}, 22, false},
		{"resistance only", order, "happy", 15, content.DamageFire, nil, fireScope(), nil, nil, [3]int{15, 7, 7}, 7, false},
		{"resistance rounds to zero", order, "unhappy (boundary)", 1, content.DamageFire, nil, fireScope(), nil, nil, [3]int{1, 0, 0}, 0, false},
		{"vulnerability only", order, "happy", 15, content.DamageCold, nil, nil, damageTypes(content.DamageCold), nil, [3]int{15, 15, 30}, 30, false},
		{"both on odd amount", order, "unhappy (rule interaction)", 7, content.DamageFire, nil, fireScope(), damageTypes(content.DamageFire), nil, [3]int{7, 3, 6}, 6, false},
		{"unrelated type", order, "unhappy (rule interaction)", 9, content.DamageAcid, nil, fireScope(), nil, nil, [3]int{9, 9, 9}, 9, false},
		{"resistance to all damage", order, "happy", 9, content.DamageNecrotic, nil, []combat.DamageScope{combat.AllDamage{}}, nil, nil, [3]int{9, 4, 4}, 4, false},
		{"resistance to type and to all halves once", order, "unhappy (rule interaction)", 9, content.DamageNecrotic, nil,
			[]combat.DamageScope{combat.DamageOfType{Type: content.DamageNecrotic}, combat.AllDamage{}, combat.DamageOfType{Type: content.DamageNecrotic}}, nil, nil, [3]int{9, 4, 4}, 4, false},
		{"bonus before resistance", order, "unhappy (rule interaction)", 15, content.DamageFire, flat(3), fireScope(), nil, nil, [3]int{18, 9, 9}, 9, false},
		{"penalty floors at zero", order, "unhappy (boundary)", 3, content.DamageFire, flat(-5), nil, nil, nil, [3]int{0, 0, 0}, 0, false},
		{"penalty floors before vulnerability", order, "unhappy (boundary)", 3, content.DamageFire, flat(-5), nil, damageTypes(content.DamageFire), nil, [3]int{0, 0, 0}, 0, false},
		{"zero damage", order, "unhappy (empty and minimal)", 0, content.DamageFire, nil, nil, damageTypes(content.DamageFire), nil, [3]int{0, 0, 0}, 0, false},
		{"immunity", order, "happy", 12, content.DamageFire, nil, nil, nil, damageTypes(content.DamageFire), [3]int{12, 12, 12}, 0, true},
		{"immunity beats vulnerability", order, "unhappy (rule interaction)", 12, content.DamageFire, flat(4), nil, damageTypes(content.DamageFire), damageTypes(content.DamageFire), [3]int{16, 16, 32}, 0, true},
		{"half damage then resistance", `SRD 5.2.1, "Half Damage", p. 16; Q-5`, "unhappy (rule interaction)", 15, content.DamageFire, half, fireScope(), nil, nil, [3]int{7, 3, 3}, 3, false},
		{"half damage with flat adjustment", `FR-099`, "unhappy (rule interaction)", 15, content.DamageFire, []combat.DamageAdjustment{combat.HalfDamage{}, combat.NewFlatAdjustment(3, nil)}, nil, nil, nil, [3]int{9, 9, 9}, 9, false},
		{"half damage with vulnerability", order, "unhappy (rule interaction)", 15, content.DamageFire, half, nil, damageTypes(content.DamageFire), nil, [3]int{7, 7, 14}, 14, false},
		{"half damage with immunity", order, "unhappy (rule interaction)", 15, content.DamageFire, half, nil, nil, damageTypes(content.DamageFire), [3]int{7, 7, 7}, 0, true},
		{"half of one", `SRD 5.2.1, "Half Damage", p. 16`, "unhappy (boundary)", 1, content.DamageFire, half, nil, damageTypes(content.DamageFire), nil, [3]int{0, 0, 0}, 0, false},
		{"half of even and odd (14)", `SRD 5.2.1, "Half Damage", p. 16 (table 4.2b)`, "happy", 14, content.DamageFire, half, nil, nil, nil, [3]int{7, 7, 7}, 7, false},
		{"half of odd (15)", `SRD 5.2.1, "Half Damage", p. 16 (table 4.2b)`, "unhappy (boundary)", 15, content.DamageFire, half, nil, nil, nil, [3]int{7, 7, 7}, 7, false},
		{"half of zero", `SRD 5.2.1, "Half Damage", p. 16 (table 4.2b)`, "unhappy (empty and minimal)", 0, content.DamageFire, half, nil, nil, nil, [3]int{0, 0, 0}, 0, false},
		{"scoped adjustment changes only its type", `FR-113`, "unhappy (rule interaction)", 10, content.DamageCold,
			[]combat.DamageAdjustment{combat.NewFlatAdjustment(-4, combat.DamageOfType{Type: content.DamageFire})}, nil, nil, nil, [3]int{10, 10, 10}, 10, false},
		{"scoped adjustment changes its type", `FR-113`, "unhappy (rule interaction)", 10, content.DamageFire,
			[]combat.DamageAdjustment{combat.NewFlatAdjustment(-4, combat.DamageOfType{Type: content.DamageFire})}, nil, nil, nil, [3]int{6, 6, 6}, 6, false},
		{"flat adjustments add up", order, "unhappy (rule interaction)", 10, content.DamageFire, []combat.DamageAdjustment{combat.NewFlatAdjustment(2, nil), combat.NewFlatAdjustment(-5, combat.AllDamage{})}, nil, nil, nil, [3]int{7, 7, 7}, 7, false},
		{"overflow: vulnerability to the maximum int saturates", `plan-phase-4.md section 5 limits (overflow)`, "unhappy (boundary)", math.MaxInt, content.DamageFire, nil, nil, damageTypes(content.DamageFire), nil, [3]int{math.MaxInt, math.MaxInt, math.MaxInt}, math.MaxInt, false},
		{"overflow: a bonus on the maximum int saturates", `plan-phase-4.md section 5 limits (overflow)`, "unhappy (boundary)", math.MaxInt, content.DamageFire, flat(5), nil, nil, nil, [3]int{math.MaxInt, math.MaxInt, math.MaxInt}, math.MaxInt, false},
		{"overflow: a penalty of the minimum int floors at 0", `plan-phase-4.md section 5 limits (overflow)`, "unhappy (boundary)", 5, content.DamageFire, flat(math.MinInt), nil, nil, nil, [3]int{0, 0, 0}, 0, false},
		{"overflow: penalties summing past the minimum int floor at 0", `plan-phase-4.md section 5 limits (overflow)`, "unhappy (boundary)", 5,
			content.DamageFire, []combat.DamageAdjustment{combat.NewFlatAdjustment(math.MinInt, nil), combat.NewFlatAdjustment(-9, nil)}, nil, nil, nil, [3]int{0, 0, 0}, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			st := status(t, 1000, 1000, 0, combat.Alive{}, combat.FallsUnconscious, affinities(t, c.res, c.vuln, c.immune))
			dmg := combat.MustDamage(c.incoming, c.typ)
			tr, err := st.ApplyDamage(subject(), []combat.Damage{dmg}, c.adjustments, combat.OtherSource, combat.NormalHit, nil)
			a := applied(t, tr, err)
			rs := a.Resolutions()
			if len(rs) != 1 {
				t.Fatalf("resolutions = %v, want one", rs)
			}
			r := rs[0]
			got := [3]int{r.AfterAdjustments(), r.AfterResistance(), r.AfterVulnerability()}
			if r.Incoming() != c.incoming || got != c.steps || r.Dealt() != c.dealt || r.Immune() != c.immuneApplied || r.Type() != c.typ || r.Unpreventable() {
				t.Errorf("resolution = %+v (dealt %d, immune %v), want steps %v dealt %d immune %v [%s]", got, r.Dealt(), r.Immune(), c.steps, c.dealt, c.immuneApplied, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-114 - an unpreventable instance skips flat adjustments,
// Half Damage, Resistance and Immunity, still takes Vulnerability, and is summed with the others; Temporary Hit
// Points absorb first (SRD 5.2.1, "Wish", p. 175).
// Path: unhappy (rule interaction).
func TestUnpreventableDamageSkipsEverythingButVulnerability(t *testing.T) {
	t.Parallel()
	st := status(t, 100, 100, 3, combat.Alive{}, combat.FallsUnconscious,
		affinities(t, fireScope(), damageTypes(content.DamageFire, content.DamageCold), damageTypes(content.DamageFire, content.DamageCold)))
	adj := []combat.DamageAdjustment{combat.NewFlatAdjustment(-100, nil), combat.HalfDamage{}}
	tr, err := st.ApplyDamage(subject(), nil, adj, combat.OtherSource, combat.NormalHit,
		[]combat.Damage{fire(10), combat.MustDamage(5, content.DamagePsychic)})
	a := applied(t, tr, err)
	rs := a.Resolutions()
	if len(rs) != 2 || !rs[0].Unpreventable() || rs[0].Dealt() != 20 || rs[0].Immune() || rs[1].Dealt() != 5 {
		t.Fatalf("resolutions = %+v, want unpreventable fire doubled to 20 despite Immunity and Resistance, psychic 5", rs)
	}
	hp := a.Status().HitPoints()
	if hp.Current() != 100-22 || hp.Temporary() != 0 {
		t.Errorf("hit points = %d, temporary %d, want 78 and 0 (3 absorbed, 22 lost)", hp.Current(), hp.Temporary())
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-100 - one Damage applied to two creatures with different
// affinities gives each its own result and is not altered (SRD 5.2.1, "Damage against Multiple Targets", p. 16).
// Path: happy.
func TestOneDamageForTwoCreatures(t *testing.T) {
	t.Parallel()
	d := fire(20)
	before := d
	resistant := status(t, 50, 50, 0, combat.Alive{}, combat.FallsUnconscious, affinities(t, fireScope(), nil, nil))
	plain := pcStatus(t, 50, 50, 0)
	got := []int{}
	for _, st := range []combat.HitPointStatus{resistant, plain} {
		tr, err := st.ApplyDamage(subject(), []combat.Damage{d}, nil, combat.OtherSource, combat.NormalHit, nil)
		got = append(got, applied(t, tr, err).Status().HitPoints().Current())
	}
	if got[0] != 40 || got[1] != 30 || d != before {
		t.Errorf("current = %v, damage %v, want [40 30] and unchanged", got, d)
	}
}
