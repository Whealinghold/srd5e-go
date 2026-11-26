package combat_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func sbCheck(tb testing.TB, kind d20.Kind, natural, dc int) d20.CheckResult {
	tb.Helper()
	roll := luMust(d20.NewRoll(modifier.Normal, []int{natural}))
	return luMust(d20.ResolveCheck(kind, roll, d20.NewRollModifiers(modifier.New(0)), d20.DifficultyClass(dc)))
}

func sbSaveRoll(tb testing.TB, natural, dc int) d20.CheckResult {
	tb.Helper()
	return sbCheck(tb, d20.SavingThrow, natural, dc)
}

func sbHours(n int) content.ImmuneToPartForHours {
	return luMust(content.NewImmuneToPartForHours(n))
}

const sbDC = 13

// The parts of the cited blocks (SRD 5.2.1, pp. 261-309), made of synthetic text.
func sbNagaSpittle() content.StatBlockPart {
	return sbSavePart("Poisonous Spittle", sbSave(ability.Constitution, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(31, sbExpr(7, 8, 0), content.DamagePoison)),
		sbClause(content.ClauseSuccess, content.HalfDamageOnly{}))
}

func sbMephitBreath() content.StatBlockPart {
	return sbSavePart("Steam Breath", sbSave(ability.Dexterity, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(5, sbExpr(1, 8, 1), content.DamageFire)),
		sbClause(content.ClauseSuccess, content.HalfDamageOnly{}),
		sbClause(content.ClauseFailureOrSuccess, nil))
}

func sbChimeraBreath() content.StatBlockPart {
	return sbSavePart("Fire Breath", sbSave(ability.Dexterity, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(31, sbExpr(7, 8, 0), content.DamageFire)),
		sbClause(content.ClauseSuccess, content.HalfDamage{}))
}

func sbBalorExplosion() content.StatBlockPart {
	return sbSavePart("Death Throes", sbSave(ability.Dexterity, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(50, sbExpr(20, 6, 0), content.DamageFire)),
		sbClause(content.ClauseSuccess, content.HalfDamage{}),
		sbClause(content.ClauseFailureOrSuccess, nil))
}

func sbBuletteLeap() content.StatBlockPart {
	return sbSavePart("Deadly Leap", sbSave(ability.Strength, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(0, sbExpr(3, 6, 0), content.DamageBludgeoning)),
		sbClause(content.ClauseSuccess, content.HalfDamage{}))
}

func sbMummyGlare() content.StatBlockPart {
	return sbSavePart("Dreadful Glare", sbSave(ability.Wisdom, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil),
		sbClause(content.ClauseSuccess, sbHours(24)))
}

func sbByFive(name string) content.StatBlockPart {
	return sbSavePart(name, sbSave(ability.Constitution, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil),
		sbClause(content.ClauseFailureByFiveOrMore, nil))
}

func sbDeathDogBite() content.StatBlockPart {
	return sbSavePart("Bite", sbSave(ability.Constitution, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil),
		sbClause(content.ClauseFirstFailure, nil),
		sbClause(content.ClauseSecondFailure, nil),
		sbClause(content.ClauseSubsequentFailures, nil))
}

func labelsOf(clauses []content.EffectClause) []content.EffectClauseLabel {
	var out []content.EffectClauseLabel
	for _, c := range clauses {
		out = append(out, c.Label())
	}
	return out
}

func sbResolve(tb testing.TB, part content.StatBlockPart, saves map[combat.CombatantID]d20.CheckResult, steps map[combat.CombatantID]combat.SaveFailureStep, targets ...string) []combat.StatBlockSaveTargetResult {
	tb.Helper()
	in := combat.StatBlockSaveEffectInput{Part: part, Ref: luRef(tb, combat.MonsterPartActions, 0), Source: sbSelf(), Saves: saves, Steps: steps}
	for _, t := range targets {
		in.Targets = append(in.Targets, combat.MustCombatantID(t))
	}
	out, err := combat.ResolveStatBlockSaveEffect(in)
	if err != nil {
		tb.Fatalf("ResolveStatBlockSaveEffect: %v", err)
	}
	return out
}

func sbApplies(tb testing.TB, r combat.StatBlockSaveTargetResult) combat.StatBlockSaveEffectApplies {
	tb.Helper()
	a, ok := r.(combat.StatBlockSaveEffectApplies)
	if !ok {
		tb.Fatalf("result = %#v, want StatBlockSaveEffectApplies", r)
	}
	return a
}

func hero(roll d20.CheckResult) map[combat.CombatantID]d20.CheckResult {
	return map[combat.CombatantID]d20.CheckResult{combat.MustCombatantID("hero"): roll}
}

// Requirement: doc/specs/monster-stat-block-actions.md table SV-1 (FR-034 to FR-037, FR-056) - a failed save applies Failure and Failure or Success with the
// damage unchanged; a success applies Success and Failure or Success and never Failure, halving the damage for the riders HalfDamageOnly and HalfDamage, and
// not for a part that prints no half damage (SRD 5.2.1, "Saving Throw Effect Notation", p. 256; "Half Damage", p. 16).
// Path: per case.
func TestResolveStatBlockSaveEffectOutcomes(t *testing.T) {
	t.Parallel()
	fail, success, ok := content.ClauseFailure, content.ClauseSuccess, content.ClauseFailureOrSuccess
	onlyFailure := sbSavePart("Cold Touch", sbSave(ability.Constitution, sbDC, "", ""), sbClause(fail, nil, sbDamage(3, sbExpr(1, 6, 0), content.DamageCold)))
	successNoRider := sbSavePart("Cold Touch", sbSave(ability.Constitution, sbDC, "", ""), sbClause(fail, nil), sbClause(success, nil))
	cases := []struct {
		name, requirement, path string
		part                    content.StatBlockPart
		natural                 int
		clauses                 []content.EffectClauseLabel
		adjustment              combat.SaveEffectDamageAdjustment
	}{
		{"fail", `SV-1; "Guardian Naga", p. 296`, "happy", sbNagaSpittle(), 5, []content.EffectClauseLabel{fail}, combat.SaveEffectDamageUnchanged},
		{"failure carries the full damage", `FR-037`, "happy", sbMephitBreath(), 12, []content.EffectClauseLabel{fail, ok}, combat.SaveEffectDamageUnchanged},
		{"half damage only", `SV-1; "Guardian Naga", p. 296`, "unhappy (rule interaction)", sbNagaSpittle(), 13, []content.EffectClauseLabel{success}, combat.SaveEffectDamageHalved},
		{"Failure or Success survives half damage only", `SV-1, FR-035; "Steam Mephit", p. 308`, "unhappy (rule interaction)", sbMephitBreath(), 20, []content.EffectClauseLabel{success, ok}, combat.SaveEffectDamageHalved},
		{"plain half damage", `SV-1, FR-056; "Chimera", p. 273`, "unhappy (rule interaction)", sbChimeraBreath(), 14, []content.EffectClauseLabel{success}, combat.SaveEffectDamageHalved},
		{"half damage with more", `SV-1, FR-056; "Balor", p. 261`, "unhappy (rule interaction)", sbBalorExplosion(), 15, []content.EffectClauseLabel{success, ok}, combat.SaveEffectDamageHalved},
		{"half damage and a push", `SV-1, FR-056; "Bulette", p. 272`, "unhappy (rule interaction)", sbBuletteLeap(), 13, []content.EffectClauseLabel{success}, combat.SaveEffectDamageHalved},
		{"no success clause", `SV-1, FR-036`, "unhappy (empty and minimal)", onlyFailure, 18, nil, combat.SaveEffectDamageUnchanged},
		{"success clause with no rider", `FR-036`, "unhappy (empty and minimal)", successNoRider, 18, []content.EffectClauseLabel{success}, combat.SaveEffectDamageUnchanged},
		{"a total one below the DC fails", `FR-037`, "unhappy (boundary)", sbChimeraBreath(), sbDC - 1, []content.EffectClauseLabel{fail}, combat.SaveEffectDamageUnchanged},
		{"a total equal to the DC succeeds", `FR-037`, "unhappy (boundary)", sbChimeraBreath(), sbDC, []content.EffectClauseLabel{success}, combat.SaveEffectDamageHalved},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sbApplies(t, sbResolve(t, c.part, hero(sbSaveRoll(t, c.natural, sbDC)), nil, "hero")[0])
			if !slices.Equal(labelsOf(got.Clauses()), c.clauses) || got.DamageAdjustment() != c.adjustment || got.Combatants()[0] != combat.MustCombatantID("hero") {
				t.Errorf("clauses %v adjustment %v, want %v and %v [%s]", labelsOf(got.Clauses()), got.DamageAdjustment(), c.clauses, c.adjustment, c.requirement)
			}
			if _, granted := got.Grant(); granted {
				t.Error("a rider that grants no immunity granted a window")
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md table SV-1 "by five or more", "by four", "both by-five readings", "first failure", "subsequent", "no step
// given", "step on a success" (FR-034, FR-055) - Failure by 5 or More applies when the total is at most the DC minus 5, beside Failure and in printed order with its
// text unparsed; the host's step selects First, Second or Subsequent Failure; a step for a success is ignored.
// Path: per case.
func TestResolveStatBlockSaveEffectFailureClauses(t *testing.T) {
	t.Parallel()
	fail, five := content.ClauseFailure, content.ClauseFailureByFiveOrMore
	first, second, later := content.ClauseFirstFailure, content.ClauseSecondFailure, content.ClauseSubsequentFailures
	cases := []struct {
		name, requirement, path string
		part                    content.StatBlockPart
		natural                 int
		step                    combat.SaveFailureStep
		want                    []content.EffectClauseLabel
	}{
		{"by five or more", `SV-1; "Homunculus", p. 299`, "unhappy (boundary)", sbByFive("Sting"), sbDC - 5, 0, []content.EffectClauseLabel{fail, five}},
		{"by four", `SV-1; "Homunculus", p. 299`, "unhappy (boundary)", sbByFive("Sting"), sbDC - 4, 0, []content.EffectClauseLabel{fail}},
		{"by six", `FR-034`, "unhappy (boundary)", sbByFive("Sting"), sbDC - 6, 0, []content.EffectClauseLabel{fail, five}},
		{"by five on a Pseudodragon", `SV-1; "Pseudodragon", p. 316`, "unhappy (rule interaction)", sbByFive("Sting"), sbDC - 5, 0, []content.EffectClauseLabel{fail, five}},
		{"by five does not apply to a success", `FR-034`, "unhappy (rule interaction)", sbByFive("Sting"), sbDC, 0, nil},
		{"first failure", `SV-1; "Death Dog", p. 279`, "happy", sbDeathDogBite(), 4, combat.SaveFirstFailure, []content.EffectClauseLabel{fail, first}},
		{"second failure", `SV-1; "Death Dog", p. 279`, "unhappy (rule interaction)", sbDeathDogBite(), 4, combat.SaveSecondFailure, []content.EffectClauseLabel{fail, second}},
		{"subsequent failures", `SV-1; "Death Dog", p. 279`, "unhappy (rule interaction)", sbDeathDogBite(), 4, combat.SaveSubsequentFailures, []content.EffectClauseLabel{fail, later}},
		{"no step given", `SV-1; "Death Dog", p. 279`, "unhappy (empty and minimal)", sbDeathDogBite(), 4, 0, []content.EffectClauseLabel{fail}},
		{"step on a success is ignored", `SV-1, FR-055`, "unhappy (rule interaction)", sbDeathDogBite(), 18, combat.SaveFirstFailure, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var steps map[combat.CombatantID]combat.SaveFailureStep
			if c.step != 0 {
				steps = map[combat.CombatantID]combat.SaveFailureStep{combat.MustCombatantID("hero"): c.step}
			}
			got := sbApplies(t, sbResolve(t, c.part, hero(sbSaveRoll(t, c.natural, sbDC)), steps, "hero")[0])
			if !slices.Equal(labelsOf(got.Clauses()), c.want) {
				t.Errorf("clauses = %v, want %v [%s]", labelsOf(got.Clauses()), c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-054, FR-055, table SV-1 "missing save", "wrong roll" - a target that is not immune and has no CheckResult
// is refused as missing; a CheckResult that is not a saving throw, or whose target differs from the printed DC, is refused as not this save; one target's refusal does not
// affect the others, results come in the order of the targets, and a step or a save for a creature that is not a target is ignored.
// Path: per case.
func TestResolveStatBlockSaveEffectRefusesUnusableSaves(t *testing.T) {
	t.Parallel()
	part := sbChimeraBreath()
	saves := map[combat.CombatantID]d20.CheckResult{
		combat.MustCombatantID("a"): sbSaveRoll(t, 5, sbDC),
		combat.MustCombatantID("c"): sbCheck(t, d20.AbilityCheck, 20, sbDC),
		combat.MustCombatantID("d"): sbSaveRoll(t, 20, sbDC+1),
		combat.MustCombatantID("e"): sbSaveRoll(t, 20, sbDC),
		combat.MustCombatantID("x"): sbSaveRoll(t, 1, sbDC),
		combat.MustCombatantID("z"): {},
	}
	steps := map[combat.CombatantID]combat.SaveFailureStep{combat.MustCombatantID("y"): combat.SaveFirstFailure}
	got := sbResolve(t, part, saves, steps, "e", "b", "a", "c", "d", "z")
	wantTargets := []string{"e", "b", "a", "c", "d", "z"}
	for i, r := range got {
		if r.Combatants()[0] != combat.MustCombatantID(wantTargets[i]) {
			t.Errorf("result %d is about %q, want %q", i, r.Combatants()[0], wantTargets[i])
		}
	}
	if a := sbApplies(t, got[0]); a.DamageAdjustment() != combat.SaveEffectDamageHalved {
		t.Errorf("target e = %+v, want it halved", a)
	}
	refused := func(i int) combat.StatBlockSaveRefusal {
		r, ok := got[i].(combat.StatBlockSaveTargetRefused)
		if !ok {
			t.Fatalf("result %d = %#v, want a refusal", i, got[i])
		}
		return r.Reason()
	}
	if refused(1) != combat.StatBlockSaveMissing || refused(3) != combat.StatBlockSaveNotThisSave || refused(4) != combat.StatBlockSaveNotThisSave || refused(5) != combat.StatBlockSaveNotThisSave {
		t.Errorf("refusals = %v %v %v %v, want missing, not this save (ability check), not this save (other DC), not this save (zero result)",
			refused(1), refused(3), refused(4), refused(5))
	}
	if a := sbApplies(t, got[2]); !slices.Equal(labelsOf(a.Clauses()), []content.EffectClauseLabel{content.ClauseFailure}) {
		t.Errorf("target a = %v, want Failure: the refusals of others do not affect it", labelsOf(a.Clauses()))
	}
	if combat.StatBlockSaveMissing.String() != "missing_save" || combat.StatBlockSaveNotThisSave.String() != "not_this_save" || combat.StatBlockSaveRefusal(0).String() != "unknown" {
		t.Error("refusal names are not missing_save, not_this_save and unknown")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-038, FR-039, FR-043, table SV-1 "grant", "no grant on failure", "immune target", "other source",
// "other part", "immune with a roll" - a success on a Success clause whose rider is ImmuneToPartForHours grants a window of that many hours to the target against
// this source and part; a failure grants none; a target immune to this source and part does not save, whatever roll or step is supplied; another source or part
// is not covered (SRD 5.2.1, "Ghost", "Mummy", pp. 288, 309).
// Path: per case.
func TestResolveStatBlockSaveEffectImmunity(t *testing.T) {
	t.Parallel()
	glare := sbMummyGlare()
	self, h := sbSelf(), combat.MustCombatantID("hero")
	ref := luRef(t, combat.MonsterPartActions, 0)
	window := func(source combat.CombatantID, part combat.LimitedUsePartRef) combat.SaveImmunities {
		return combat.SaveImmunities{}.Grant(luMust(combat.NewSaveImmunityWindow(h, source, part, time.Hour))).Immunities()
	}
	run := func(natural int, held combat.SaveImmunities) combat.StatBlockSaveTargetResult {
		in := combat.StatBlockSaveEffectInput{Part: glare, Ref: ref, Source: self, Targets: []combat.CombatantID{h}, Saves: hero(sbSaveRoll(t, natural, sbDC)),
			Steps: map[combat.CombatantID]combat.SaveFailureStep{h: combat.SaveFirstFailure}, Immunities: held}
		out, err := combat.ResolveStatBlockSaveEffect(in)
		if err != nil {
			t.Fatal(err)
		}
		return out[0]
	}
	w, granted := sbApplies(t, run(13, combat.SaveImmunities{})).Grant()
	if !granted || !slices.Equal(w.Combatants(), []combat.CombatantID{h, self}) || w.Part() != ref || w.Remaining() != 24*time.Hour {
		t.Errorf("Grant() = %+v, %v, want 24 hours for hero against %s's part %v", w, granted, self, ref)
	}
	if _, granted := sbApplies(t, run(3, combat.SaveImmunities{})).Grant(); granted {
		t.Error("a failed save granted an immunity (FR-043)")
	}
	for name, held := range map[string]combat.SaveImmunities{"immune target": window(self, ref), "immune target, roll ignored": window(self, ref)} {
		if got, ok := run(3, held).(combat.StatBlockSaveTargetImmune); !ok || got.Combatants()[0] != h {
			t.Errorf("%s: result = %#v, want StatBlockSaveTargetImmune for hero", name, got)
		}
	}
	other := map[string]combat.SaveImmunities{"other source": window(combat.MustCombatantID("other"), ref), "other part": window(self, luRef(t, combat.MonsterPartActions, 1))}
	for name, held := range other {
		if _, ok := run(13, held).(combat.StatBlockSaveEffectApplies); !ok {
			t.Errorf("%s: the target was immune to a different source or part", name)
		}
	}
	// A creature is never immune to its own part, and a duration holds a limited number of hours.
	own := combat.StatBlockSaveEffectInput{Part: glare, Ref: ref, Source: self, Targets: []combat.CombatantID{self}, Saves: map[combat.CombatantID]d20.CheckResult{self: sbSaveRoll(t, 20, sbDC)}}
	if out, err := combat.ResolveStatBlockSaveEffect(own); err != nil {
		t.Fatal(err)
	} else if _, granted := sbApplies(t, out[0]).Grant(); granted {
		t.Error("a creature was granted an immunity to its own part")
	}
	maxHours := int(time.Duration(1<<63-1) / time.Hour)
	for hours, wantErr := range map[int]bool{maxHours: false, maxHours + 1: true, content.MonsterMaxNumber: true} {
		part := sbSavePart("Glare", sbSave(ability.Wisdom, sbDC, "", ""), sbClause(content.ClauseSuccess, sbHours(hours)))
		in := combat.StatBlockSaveEffectInput{Part: part, Ref: ref, Source: self, Targets: []combat.CombatantID{h}, Saves: hero(sbSaveRoll(t, 20, sbDC))}
		if _, err := combat.ResolveStatBlockSaveEffect(in); errors.Is(err, srderr.ErrInvalidInput) != wantErr || (!wantErr && err != nil) {
			t.Errorf("%d hours: error = %v, want error %v", hours, err, wantErr)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-032, FR-053, FR-033 - the plan uses the printed ability as a saving throw and the printed DC
// unchanged, never recomputed, keeps the target description and prefix verbatim and lists the damage of the Failure clause to be rolled once; a part with no
// Saving Throw notation is an error (SRD 5.2.1, "Saving Throw Effect Notation", p. 256; "Saving Throws and Damage", p. 16).
// Path: per case.
func TestPlanStatBlockSaveEffect(t *testing.T) {
	t.Parallel()
	part := sbSavePart("Reaction Cloud", sbSave(ability.Dexterity, 17, "each creature in the cloud", "Response"),
		sbClause(content.ClauseFailure, nil, sbDamage(7, sbExpr(2, 6, 0), content.DamagePoison), sbDamage(1, dice0(), content.DamageCold)),
		sbClause(content.ClauseSuccess, content.HalfDamageOnly{}, sbDamage(99, sbExpr(9, 6, 0), content.DamageFire)))
	plan, err := combat.PlanStatBlockSaveEffect(part)
	if err != nil || plan.IsZero() {
		t.Fatalf("PlanStatBlockSaveEffect = %+v, %v", plan, err)
	}
	if plan.Kind() != d20.SavingThrow || plan.Ability() != ability.Dexterity || plan.DifficultyClass() != 17 || plan.Target() != "each creature in the cloud" || plan.Prefix() != "Response" {
		t.Errorf("plan = %v %v DC %d target %q prefix %q, want a Dexterity saving throw against 17, verbatim", plan.Kind(), plan.Ability(), plan.DifficultyClass(), plan.Target(), plan.Prefix())
	}
	damage := plan.Damage()
	if len(damage) != 2 || damage[0].Number() != 7 || damage[1].Number() != 1 {
		t.Errorf("Damage() = %v, want the two instances of the Failure clause only", damage)
	}
	damage[0] = content.StatBlockDamage{}
	if plan.Damage()[0].IsZero() {
		t.Error("Damage returned the plan's own slice")
	}
	if p, err := combat.PlanStatBlockSaveEffect(sbPart("Trait", content.StatBlockPartFields{})); !errors.Is(err, srderr.ErrInvalidInput) || !p.IsZero() {
		t.Errorf("a part with no Saving Throw = %+v, %v, want the zero plan and an invalid-input error", p, err)
	}
	if p, err := combat.PlanStatBlockSaveEffect(sbAttackPart("Claw", sbAttackNotation(content.AttackMelee, 4, "", 5, sbNoRange()))); !errors.Is(err, srderr.ErrInvalidInput) || !p.IsZero() {
		t.Errorf("an attack-only part = %+v, %v, want an invalid-input error", p, err)
	}
	empty, err := combat.PlanStatBlockSaveEffect(sbSavePart("Gaze", sbSave(ability.Wisdom, 12, "", "")))
	if err != nil || len(empty.Damage()) != 0 || empty.Target() != "" {
		t.Errorf("a save with no clauses = %+v, %v, want no damage", empty, err)
	}
}

func dice0() dice.Expression { return dice.Expression{} }

// Requirement: doc/specs/monster-stat-block-actions.md FR-053, FR-049 - the resolver rejects, with an error matching srderr.ErrInvalidInput, a part with no
// Saving Throw notation, a zero Ref or Source, a zero or repeated target and an undefined step; it never panics.
// Path: per case.
func TestResolveStatBlockSaveEffectRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	ref := luRef(t, combat.MonsterPartActions, 0)
	h := combat.MustCombatantID("hero")
	valid := combat.StatBlockSaveEffectInput{Part: sbChimeraBreath(), Ref: ref, Source: sbSelf(), Targets: []combat.CombatantID{h}, Saves: hero(sbSaveRoll(t, 5, sbDC))}
	with := func(f func(*combat.StatBlockSaveEffectInput)) combat.StatBlockSaveEffectInput {
		in := valid
		f(&in)
		return in
	}
	cases := []struct {
		name string
		in   combat.StatBlockSaveEffectInput
	}{
		{"part with no save", with(func(in *combat.StatBlockSaveEffectInput) { in.Part = sbPart("Trait", content.StatBlockPartFields{}) })},
		{"attack-only part", with(func(in *combat.StatBlockSaveEffectInput) {
			in.Part = sbAttackPart("Claw", sbAttackNotation(content.AttackMelee, 4, "", 5, sbNoRange()))
		})},
		{"zero ref", with(func(in *combat.StatBlockSaveEffectInput) { in.Ref = combat.LimitedUsePartRef{} })},
		{"zero source", with(func(in *combat.StatBlockSaveEffectInput) { in.Source = "" })},
		{"zero target", with(func(in *combat.StatBlockSaveEffectInput) { in.Targets = []combat.CombatantID{h, ""} })},
		{"repeated target", with(func(in *combat.StatBlockSaveEffectInput) { in.Targets = []combat.CombatantID{h, h} })},
		{"undefined step", with(func(in *combat.StatBlockSaveEffectInput) {
			in.Steps = map[combat.CombatantID]combat.SaveFailureStep{h: 9}
		})},
		{"undefined step for a stranger", with(func(in *combat.StatBlockSaveEffectInput) {
			in.Steps = map[combat.CombatantID]combat.SaveFailureStep{"stranger": 0}
		})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if out, err := combat.ResolveStatBlockSaveEffect(c.in); out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("ResolveStatBlockSaveEffect = %#v, %v, want a nil result and an invalid-input error", out, err)
			}
		})
	}
	// No target is not an error: there is nothing to resolve.
	if out, err := combat.ResolveStatBlockSaveEffect(with(func(in *combat.StatBlockSaveEffectInput) { in.Targets = nil })); err != nil || len(out) != 0 {
		t.Errorf("no targets = %v, %v, want an empty result", out, err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - the resolver changes none of its inputs and returns copies: the saves, steps and targets given are
// not altered, and the clauses of a result are a copy.
// Path: unhappy (immutability).
func TestResolveStatBlockSaveEffectLeavesInputsUnchanged(t *testing.T) {
	t.Parallel()
	h := combat.MustCombatantID("hero")
	saves := hero(sbSaveRoll(t, 5, sbDC))
	steps := map[combat.CombatantID]combat.SaveFailureStep{h: combat.SaveFirstFailure}
	targets := []combat.CombatantID{h}
	in := combat.StatBlockSaveEffectInput{Part: sbDeathDogBite(), Ref: luRef(t, combat.MonsterPartActions, 0), Source: sbSelf(), Targets: targets, Saves: saves, Steps: steps}
	out, err := combat.ResolveStatBlockSaveEffect(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 || !saves[h].Equal(sbSaveRoll(t, 5, sbDC)) || steps[h] != combat.SaveFirstFailure || len(steps) != 1 || !slices.Equal(targets, []combat.CombatantID{h}) {
		t.Error("the resolver changed its inputs")
	}
	a := sbApplies(t, out[0])
	clauses := a.Clauses()
	clauses[0] = content.EffectClause{}
	if a.Clauses()[0].Label() != content.ClauseFailure {
		t.Error("Clauses returned the result's own slice")
	}
	if combat.SaveFirstFailure.String() != "first" || combat.SaveSecondFailure.String() != "second" || combat.SaveSubsequentFailures.String() != "subsequent" || combat.SaveFailureStep(0).String() != "unknown" {
		t.Error("step names are not first, second, subsequent and unknown")
	}
	if combat.SaveEffectDamageUnchanged.String() != "none" || combat.SaveEffectDamageHalved.String() != "halve" || combat.SaveEffectDamageAdjustment(0).String() != "unknown" {
		t.Error("adjustment names are not none, halve and unknown")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md section 8 (FR-035, FR-056) - the halve adjustment, applied through the real hit-points pipeline, gives round-down halves
// (SRD 5.2.1, "Half Damage", p. 16).
// Path: per case.
func TestHalvedSaveDamageThroughTheDamagePipeline(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ dealt, want int }{{14, 7}, {15, 7}, {1, 0}} {
		got := sbApplies(t, sbResolve(t, sbNagaSpittle(), hero(sbSaveRoll(t, 20, sbDC)), nil, "hero")[0])
		var adjustments []combat.DamageAdjustment
		if got.DamageAdjustment() == combat.SaveEffectDamageHalved {
			adjustments = []combat.DamageAdjustment{combat.HalfDamage{}}
		}
		st := status(t, 1000, 1000, 0, combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
		tr, err := st.ApplyDamage(subject(), []combat.Damage{combat.MustDamage(c.dealt, content.DamagePoison)}, adjustments, combat.OtherSource, combat.NormalHit, nil)
		rs := applied(t, tr, err).Resolutions()
		if len(rs) != 1 || rs[0].Dealt() != c.want {
			t.Errorf("%d damage halved = %v, want %d", c.dealt, rs, c.want)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-032, FR-054, FR-039 - the damage of a plan is that of the Failure clause alone, even when a clause of another label follows it; a save rolled
// against a lower DC than the printed one is not this save; an immunity is granted to a target whichever way its id compares with the source's.
// Path: per case.
func TestSaveEffectEdgeCases(t *testing.T) {
	t.Parallel()
	part := sbSavePart("Cloud", sbSave(ability.Dexterity, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(7, sbExpr(2, 6, 0), content.DamagePoison)),
		sbClause(content.ClauseHitOrMiss, nil, sbDamage(99, sbExpr(9, 6, 0), content.DamageFire)))
	plan := luMust(combat.PlanStatBlockSaveEffect(part))
	if damage := plan.Damage(); len(damage) != 1 || damage[0].Number() != 7 {
		t.Errorf("Damage() = %v, want the Failure clause's single instance", damage)
	}
	lower := map[combat.CombatantID]d20.CheckResult{"hero": sbSaveRoll(t, 20, sbDC-1)}
	if r, ok := sbResolve(t, part, lower, nil, "hero")[0].(combat.StatBlockSaveTargetRefused); !ok || r.Reason() != combat.StatBlockSaveNotThisSave {
		t.Errorf("a save against a lower DC = %#v, want not this save", r)
	}
	glare := sbMummyGlare()
	for _, who := range []string{"aaa", "zzz"} {
		id := combat.MustCombatantID(who)
		in := combat.StatBlockSaveEffectInput{Part: glare, Ref: luRef(t, combat.MonsterPartActions, 0), Source: sbSelf(), Targets: []combat.CombatantID{id},
			Saves: map[combat.CombatantID]d20.CheckResult{id: sbSaveRoll(t, 20, sbDC)}}
		out, err := combat.ResolveStatBlockSaveEffect(in)
		if err != nil {
			t.Fatal(err)
		}
		if _, granted := sbApplies(t, out[0]).Grant(); !granted {
			t.Errorf("target %q was granted no immunity", who)
		}
	}
}
