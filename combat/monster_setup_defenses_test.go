package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

type defenseOutcome struct {
	resist, vuln, immune []content.DamageType
	conditions           []content.ConditionKind
	unapplied            []unappliedRow
}

func outcomeOf(s combat.MonsterSetup) defenseOutcome {
	aff := s.HitPointStatus().Affinities()
	var resist []content.DamageType
	for _, r := range aff.Resistances() {
		resist = append(resist, r.(combat.DamageOfType).Type)
	}
	return defenseOutcome{resist, aff.Vulnerabilities(), aff.Immunities(), s.Conditions().Immunities(), rowsOf(s.UnappliedDefenses())}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-018 to FR-024, table 4.4a - unqualified damage entries become affinities and unqualified condition entries become condition immunities, never crossing; qualified and described entries are listed as unapplied, ordered by list; the host's extra affinities are merged by union (SRD 5.2.1, "Resistances and Vulnerabilities", "Immunities", p. 255; "Resistance and Vulnerability", "No Stacking", p. 17).
// Path: per case.
func TestMonsterSetupDefenses(t *testing.T) {
	t.Parallel()
	eight := []content.ConditionKind{content.ConditionExhaustion, content.ConditionGrappled, content.ConditionParalyzed, content.ConditionPetrified,
		content.ConditionPoisoned, content.ConditionProne, content.ConditionRestrained, content.ConditionUnconscious}
	var eightEntries []content.DefenseEntry
	for _, k := range eight {
		eightEntries = append(eightEntries, msConditionDefense(k, ""))
	}
	four := []content.DamageType{content.DamageBludgeoning, content.DamageLightning, content.DamagePiercing, content.DamageSlashing}
	fourEntries := func() []content.DefenseEntry {
		var out []content.DefenseEntry
		for _, d := range four {
			out = append(out, msDamageDefense(d, ""))
		}
		return out
	}
	qualifiedPiercing := msDamageDefense(content.DamagePiercing, "from weapons wielded by creatures under a Bless spell")
	qualifiedCharmed := msConditionDefense(content.ConditionCharmed, "with Mind Blank")
	chosen := msDescribed("Damage type chosen for the Draconic Origin trait")
	fire := msAffinities([]combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}}, nil, nil)
	piercing := msAffinities([]combat.DamageScope{combat.DamageOfType{Type: content.DamagePiercing}}, nil, nil)
	immunePiercing := msAffinities(nil, nil, []content.DamageType{content.DamagePiercing})
	cases := []struct {
		name, requirement, path string
		edit                    func(*content.MonsterFields)
		options                 []combat.SetupOption
		want                    defenseOutcome
	}{
		{"plain Resistances", "FR-018", "happy", func(f *content.MonsterFields) { f.Resistances = fourEntries() }, nil, defenseOutcome{resist: four}},
		{"damage then conditions", "FR-018 FR-019 FR-020", "happy", func(f *content.MonsterFields) {
			f.Immunities = append([]content.DefenseEntry{msDamageDefense(content.DamagePoison, ""), msDamageDefense(content.DamageThunder, "")}, eightEntries...)
		}, nil, defenseOutcome{immune: []content.DamageType{content.DamagePoison, content.DamageThunder}, conditions: eight}},
		{"vulnerability", "FR-018", "happy", func(f *content.MonsterFields) {
			f.Vulnerabilities = []content.DefenseEntry{msDamageDefense(content.DamageFire, "")}
		}, nil,
			defenseOutcome{vuln: []content.DamageType{content.DamageFire}}},
		{"no defence entries", "FR-018", "unhappy (empty and minimal)", nil, nil, defenseOutcome{}},
		{"qualified condition", "FR-021", "unhappy (rule interaction)", func(f *content.MonsterFields) {
			f.Immunities = []content.DefenseEntry{msDamageDefense(content.DamagePsychic, ""), qualifiedCharmed}
		}, nil, defenseOutcome{immune: []content.DamageType{content.DamagePsychic},
			unapplied: []unappliedRow{unappliedOf(combat.DefenseImmunity, qualifiedCharmed, combat.DefenseQualified)}}},
		{"qualified damage", "FR-021", "unhappy (rule interaction)", func(f *content.MonsterFields) { f.Vulnerabilities = []content.DefenseEntry{qualifiedPiercing} }, nil,
			defenseOutcome{unapplied: []unappliedRow{unappliedOf(combat.DefenseVulnerability, qualifiedPiercing, combat.DefenseQualified)}}},
		{"described", "FR-022", "unhappy (rule interaction)", func(f *content.MonsterFields) { f.Resistances = []content.DefenseEntry{chosen} }, nil,
			defenseOutcome{unapplied: []unappliedRow{unappliedOf(combat.DefenseResistance, chosen, combat.DefenseDescribed)}}},
		{"host supplies the choice", "FR-022 FR-024", "unhappy (rule interaction)", func(f *content.MonsterFields) { f.Resistances = []content.DefenseEntry{chosen} },
			[]combat.SetupOption{combat.WithExtraAffinities(fire)},
			defenseOutcome{resist: []content.DamageType{content.DamageFire}, unapplied: []unappliedRow{unappliedOf(combat.DefenseResistance, chosen, combat.DefenseDescribed)}}},
		{"extra duplicates the block", "FR-020 FR-024", "unhappy (rule interaction)", func(f *content.MonsterFields) { f.Resistances = fourEntries() },
			[]combat.SetupOption{combat.WithExtraAffinities(piercing)}, defenseOutcome{resist: four}},
		{"order of unapplied", "FR-023", "unhappy (boundary)", func(f *content.MonsterFields) {
			f.Vulnerabilities = []content.DefenseEntry{qualifiedPiercing}
			f.Resistances = []content.DefenseEntry{chosen}
			f.Immunities = []content.DefenseEntry{qualifiedCharmed, msDescribed("Chosen immunity")}
		}, nil, defenseOutcome{unapplied: []unappliedRow{
			unappliedOf(combat.DefenseVulnerability, qualifiedPiercing, combat.DefenseQualified),
			unappliedOf(combat.DefenseResistance, chosen, combat.DefenseDescribed),
			unappliedOf(combat.DefenseImmunity, qualifiedCharmed, combat.DefenseQualified),
			unappliedOf(combat.DefenseImmunity, msDescribed("Chosen immunity"), combat.DefenseDescribed)}}},
		{"extra contradicts the block", "FR-020 FR-024", "unhappy (rule interaction)", func(f *content.MonsterFields) {
			f.Resistances = []content.DefenseEntry{msDamageDefense(content.DamagePiercing, "")}
		},
			[]combat.SetupOption{combat.WithExtraAffinities(immunePiercing)}, defenseOutcome{resist: []content.DamageType{content.DamagePiercing}, immune: []content.DamageType{content.DamagePiercing}}},
		{"a type in two lists is kept in both", "FR-020", "unhappy (rule interaction)", func(f *content.MonsterFields) {
			f.Vulnerabilities = []content.DefenseEntry{msDamageDefense(content.DamageFire, "")}
			f.Resistances = []content.DefenseEntry{msDamageDefense(content.DamageFire, "")}
		}, nil, defenseOutcome{resist: []content.DamageType{content.DamageFire}, vuln: []content.DamageType{content.DamageFire}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := setupSpec + c.requirement
			s := msSetup(t, msMonster(t, c.edit), c.options...)
			got := outcomeOf(s)
			if !slices.Equal(got.resist, c.want.resist) || !slices.Equal(got.vuln, c.want.vuln) || !slices.Equal(got.immune, c.want.immune) ||
				!slices.Equal(got.conditions, c.want.conditions) || !slices.Equal(got.unapplied, c.want.unapplied) {
				t.Errorf("defences = %+v, want %+v [%s]", got, c.want, req)
			}
			if len(s.Conditions().Explicit()) != 0 {
				t.Errorf("the set holds explicit conditions %v, want none (FR-019)", s.Conditions().Explicit())
			}
		})
	}
}

type unappliedRow struct {
	kind   combat.MonsterDefenseKind
	entry  content.DefenseEntry
	reason combat.UnappliedDefenseReason
}

func unappliedOf(kind combat.MonsterDefenseKind, entry content.DefenseEntry, reason combat.UnappliedDefenseReason) unappliedRow {
	return unappliedRow{kind, entry, reason}
}

func rowsOf(ds []combat.UnappliedDefense) []unappliedRow {
	var out []unappliedRow
	for _, d := range ds {
		out = append(out, unappliedRow{d.Kind(), d.Entry(), d.Reason()})
	}
	return out
}

func msAffinities(resist []combat.DamageScope, vuln, immune []content.DamageType) combat.DamageAffinities {
	return luMust(combat.NewDamageAffinities(resist, vuln, immune))
}

// Requirement: doc/specs/monster-combatant-setup.md FR-015, FR-016, FR-017 - the rule defaults to DiesInstantly, the status carries the rule given unchanged, and the default does not depend on the monster (Challenge Rating 0 or 30); dropping to 0 Hit Points kills under the default and makes the creature Dying under FallsUnconscious (SRD 5.2.1, "Monster Death", p. 17).
// Path: per case.
func TestMonsterSetupZeroHitPointsRule(t *testing.T) {
	t.Parallel()
	cr := func(s string) func(*content.MonsterFields) {
		return func(f *content.MonsterFields) { f.ChallengeRating = content.MustChallengeRating(s) }
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*content.MonsterFields)
		options                 []combat.SetupOption
		want                    combat.ZeroHitPointsRule
	}{
		{"default", "FR-015", "happy", nil, nil, combat.DiesInstantly},
		{"given DiesInstantly", "FR-015", "happy", nil, []combat.SetupOption{combat.WithZeroHitPointsRule(combat.DiesInstantly)}, combat.DiesInstantly},
		{"FallsUnconscious", "FR-016", "unhappy (rule interaction)", nil, []combat.SetupOption{combat.WithZeroHitPointsRule(combat.FallsUnconscious)}, combat.FallsUnconscious},
		{"default for Challenge Rating 0", "FR-017", "unhappy (rule interaction)", cr("0"), nil, combat.DiesInstantly},
		{"default for Challenge Rating 30", "FR-017", "unhappy (rule interaction)", cr("30"), nil, combat.DiesInstantly},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := msSetup(t, msMonster(t, c.edit), c.options...)
			if got := s.HitPointStatus().ZeroHitPointsRule(); got != c.want {
				t.Fatalf("rule = %v, want %v [%s]", got, c.want, setupSpec+c.requirement)
			}
			res, err := s.HitPointStatus().ApplyDamage(combat.MustCombatantID("a"), []combat.Damage{combat.MustDamage(150, content.DamageSlashing)}, nil,
				combat.MeleeAttackWithoutKnockOut, combat.NormalHit, nil)
			if err != nil {
				t.Fatal(err)
			}
			life := res.(combat.HitPointTransitionApplied).Status().LifeState()
			_, dead := life.(combat.Dead)
			_, dying := life.(combat.Dying)
			if (c.want == combat.DiesInstantly && !dead) || (c.want == combat.FallsUnconscious && !dying) {
				t.Errorf("after lethal damage the creature is %T, want per rule %v", life, c.want)
			}
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-020, FR-024 - extra affinities merge by union in each of the three lists: a vulnerability or an immunity the extras repeat is kept once, and one they add is added.
// Path: unhappy (rule interaction).
func TestMonsterSetupExtraAffinitiesUnionEveryList(t *testing.T) {
	t.Parallel()
	edit := func(f *content.MonsterFields) {
		f.Vulnerabilities = []content.DefenseEntry{msDamageDefense(content.DamageFire, "")}
		f.Immunities = []content.DefenseEntry{msDamageDefense(content.DamagePoison, "")}
	}
	extra := msAffinities(nil, []content.DamageType{content.DamageFire, content.DamageCold}, []content.DamageType{content.DamagePoison, content.DamageAcid})
	got := outcomeOf(msSetup(t, msMonster(t, edit), combat.WithExtraAffinities(extra)))
	if !slices.Equal(got.vuln, []content.DamageType{content.DamageCold, content.DamageFire}) || !slices.Equal(got.immune, []content.DamageType{content.DamageAcid, content.DamagePoison}) {
		t.Errorf("vulnerabilities %v, immunities %v, want the unions", got.vuln, got.immune)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-023 - the defence kinds and reasons have their snake_case names, an undefined value is unknown.
// Path: unhappy (boundary).
func TestMonsterDefenseNames(t *testing.T) {
	t.Parallel()
	got := []string{combat.DefenseVulnerability.String(), combat.DefenseResistance.String(), combat.DefenseImmunity.String(), combat.MonsterDefenseKind(0).String(),
		combat.DefenseQualified.String(), combat.DefenseDescribed.String(), combat.UnappliedDefenseReason(0).String()}
	want := []string{"vulnerability", "resistance", "immunity", "unknown", "qualified", "described", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}
