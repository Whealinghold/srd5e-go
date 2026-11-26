package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// seDamages is the damage of the target at i of res, flattened over its PlannedDamage values.
func seDamages(res combat.SpellEffectsResolved, i int) []combat.PlannedDamage {
	return res.Targets()[i].Damage()
}

func seOnly(t *testing.T, res combat.SpellEffectsResolved, i int) combat.PlannedDamage {
	t.Helper()
	d := seDamages(res, i)
	if len(d) != 1 {
		t.Fatalf("target %d has %d PlannedDamage values, want 1", i, len(d))
	}
	return d[0]
}

var seEightD6 = []int{3, 4, 3, 4, 3, 4, 3, 4} // 28

// Requirement: spell-effects.md table S (S1 to S13), FR-024, FR-025, FR-026, FR-028, FR-035, FR-038, FR-064, FR-065 - a saving throw part rolls its damage once for all targets before the first save, a failed save plans the full damage, a successful save of a half-damage part plans the same full amount marked HalfDamage (the damage outcomes only), a nothing-on-success part plans nothing, and a forced failure rolls no d20.
// Path: per case.
func TestSavingThrowOutcomes(t *testing.T) {
	t.Parallel()
	failed, saved := []int{5}, []int{19}
	cases := []struct {
		name, requirement, path string
		spell                   content.Spell
		report                  int // CastAtLevel
		scaling                 int
		targets                 []combat.SpellTarget
		faces                   []int
		wantDamage              [][]combat.Damage // per target; nil: no PlannedDamage
		wantHalf                []bool
		wantConditions          int
		wantRequests            int
	}{
		{"S1 Fireball failed", "S1", "happy", seFireball(), 3, 0, []combat.SpellTarget{seTarget("a", nil)}, slices.Concat(seEightD6, failed),
			[][]combat.Damage{{combat.MustDamage(28, content.DamageFire)}}, []bool{false}, 0, 2},
		{"S2 Fireball succeeded", "S2", "happy", seFireball(), 3, 0, []combat.SpellTarget{seTarget("a", nil)}, slices.Concat(seEightD6, saved),
			[][]combat.Damage{{combat.MustDamage(28, content.DamageFire)}}, []bool{true}, 0, 2},
		{"S3 Fireball two targets", "S3", "unhappy (rule interaction)", seFireball(), 3, 0, []combat.SpellTarget{seTarget("a", nil), seTarget("b", nil)}, slices.Concat(seEightD6, failed, saved),
			[][]combat.Damage{{combat.MustDamage(28, content.DamageFire)}, {combat.MustDamage(28, content.DamageFire)}}, []bool{false, true}, 0, 3},
		{"S4 Acid Splash failed", "S4", "happy", seAcidSplash(), 0, 1, []combat.SpellTarget{seTarget("a", nil)}, []int{4, 5},
			[][]combat.Damage{{combat.MustDamage(4, content.DamageAcid)}}, []bool{false}, 0, 2},
		{"S5 Acid Splash succeeded", "S5", "unhappy (boundary)", seAcidSplash(), 0, 1, []combat.SpellTarget{seTarget("a", nil)}, []int{4, 19},
			[][]combat.Damage{nil}, nil, 0, 2},
		{"S6 Hold Person failed", "S6", "happy", seHoldPerson(), 2, 0, []combat.SpellTarget{seTarget("a", nil)}, []int{5}, [][]combat.Damage{nil}, nil, 1, 1},
		{"S7 Hold Person succeeded", "S7", "unhappy (boundary)", seHoldPerson(), 2, 0, []combat.SpellTarget{seTarget("a", nil)}, []int{19}, [][]combat.Damage{nil}, nil, 0, 1},
		{"S8 Harm succeeded", "S8", "unhappy (rule interaction)", seHarm(), 6, 0, []combat.SpellTarget{seTarget("a", nil)}, slices.Concat(slices.Repeat([]int{3, 4}, 7), saved),
			[][]combat.Damage{{combat.MustDamage(49, content.DamageNecrotic)}}, []bool{true}, 0, 2},
		{"S11 Flame Strike failed", "S11", "happy", seFlameStrike(), 5, 0, []combat.SpellTarget{seTarget("a", nil)}, slices.Concat([]int{4, 4, 4, 4, 4}, []int{3, 3, 3, 4, 4}, failed),
			[][]combat.Damage{{combat.MustDamage(20, content.DamageFire), combat.MustDamage(17, content.DamageRadiant)}}, []bool{false}, 0, 3},
		{"S13 Flame Strike succeeded", "S13", "unhappy (rule interaction)", seFlameStrike(), 5, 0, []combat.SpellTarget{seTarget("a", nil)}, slices.Concat([]int{4, 4, 4, 4, 4}, []int{3, 3, 3, 4, 4}, saved),
			[][]combat.Damage{{combat.MustDamage(20, content.DamageFire), combat.MustDamage(17, content.DamageRadiant)}}, []bool{true}, 0, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := spellEffectsSpec + c.requirement
			roller := dicetest.NewFixedRoller(c.faces...)
			res := seResolved(t, seCommand(c.spell, seReport(string(c.spell.ID()), c.report, c.scaling), c.targets...), roller)
			if len(roller.Requests()) != c.wantRequests {
				t.Errorf("requests = %v, want %d [%s]", roller.Requests(), c.wantRequests, req)
			}
			for i, want := range c.wantDamage {
				got := seDamages(res, i)
				if want == nil {
					if len(got) != 0 {
						t.Errorf("target %d damage = %v, want none [%s]", i, got, req)
					}
					continue
				}
				if len(got) != 1 || !slices.Equal(got[0].Damages(), want) {
					t.Fatalf("target %d damage = %v, want %v [%s]", i, got, want, req)
				}
				if half := len(got[0].Adjustments()) == 1; half != c.wantHalf[i] || (half && got[0].Adjustments()[0] != combat.DamageAdjustment(combat.HalfDamage{})) {
					t.Errorf("target %d adjustments = %v, want half %v [%s]", i, got[0].Adjustments(), c.wantHalf[i], req)
				}
				if got[0].HitKind() != combat.NormalHit {
					t.Errorf("target %d HitKind = %v [%s]", i, got[0].HitKind(), req)
				}
			}
			if n := len(res.Targets()[0].Conditions()); n != c.wantConditions {
				t.Errorf("conditions = %d, want %d [%s]", n, c.wantConditions, req)
			}
		})
	}
}

// Requirement: spell-effects.md S9, S10, FR-024 - a ChooseToFail or AutomaticFailure save fails without a d20 and the full plan applies; the evidence holds the forced failure unchanged.
// Path: unhappy (rule interaction).
func TestForcedSavesRollNoD20(t *testing.T) {
	t.Parallel()
	for name, attempt := range map[string]d20.SaveAttempt{"S9 ChooseToFail": d20.ChooseToFail{}, "S10 AutomaticFailure": d20.AutomaticFailure{}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			target := seTarget("a", func(p *combat.SpellTargetParams) {
				p.Saves = map[ability.Ability]d20.SaveAttempt{ability.Dexterity: attempt}
			})
			roller := dicetest.NewFixedRoller(seEightD6...)
			res := seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0), target), roller)
			if got := roller.Requests(); len(got) != 1 || got[0] != dice.MustExpression(8, dice.D6(), 0) {
				t.Errorf("requests = %v, want only 8d6", got)
			}
			if d := seOnly(t, res, 0); len(d.Adjustments()) != 0 || d.Damages()[0].Amount() != 28 {
				t.Errorf("plan = %v, want the full 28", d)
			}
			saves := res.Parts()[0].Instances()[0].Saves()
			forced, ok := saves[0].Save().(combat.UnarmedStrikeSaveForced)
			if len(saves) != 1 || !ok || forced.Outcome() != d20.Failure {
				t.Errorf("save evidence = %#v, want one forced failure", saves)
			}
		})
	}
}

// Requirement: spell-effects.md table N (N1 to N8), FR-043, FR-050 - slot scaling uses n = the larger of 0 and CastAtLevel minus AboveLevel, a cast with no CastAtLevel uses the base values, and a cantrip is scaled by the character level.
// Path: per case.
func TestScalingFromTheCast(t *testing.T) {
	t.Parallel()
	d6 := func(n int) dice.Expression { return dice.MustExpression(n, dice.D6(), 0) }
	cases := []struct {
		name, requirement, path string
		spell                   content.Spell
		castAt, scaling         int
		want                    dice.Expression
	}{
		{"N1 Fireball at its level", "N1", "unhappy (boundary)", seFireball(), 3, 0, d6(8)},
		{"N2 Fireball one above", "N2", "happy", seFireball(), 4, 0, d6(9)},
		{"N3 Fireball at 9", "N3", "unhappy (boundary)", seFireball(), 9, 0, d6(14)},
		{"N4 Cure Wounds at its level", "N4", "unhappy (boundary)", seCureWounds(), 1, 0, dice.MustExpression(2, dice.D8(), 0)},
		{"N6 ritual without a cast level", "N6", "unhappy (rule interaction)", seWardenBolt(), 0, 0, d6(2)},
		{"N7 Fire Bolt at character level 5", "N7", "unhappy (rule interaction)", seFireBolt(), 0, 5, dice.MustExpression(2, dice.D10(), 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := spellEffectsSpec + c.requirement
			var roller combat.Roller = dicetest.NewMaximumRoller()
			fixed := dicetest.NewFixedRoller(15, 1, 1)
			if c.spell.ID() == "fire_bolt" {
				roller = fixed
			}
			target := seWithFacts("a")
			cmd := seCommand(c.spell, seReport(string(c.spell.ID()), c.castAt, c.scaling), target)
			if _, err := seRun(t, cmd, roller); err != nil {
				t.Fatalf("Run: %v [%s]", err, req)
			}
			var found bool
			for _, r := range append(fixed.Requests(), roller.(interface{ Requests() []dice.Expression }).Requests()...) {
				found = found || r == c.want
			}
			if !found {
				t.Errorf("requests = %v, want one for %v [%s]", roller.(interface{ Requests() []dice.Expression }).Requests(), c.want, req)
			}
		})
	}
	t.Run("N5 Hold Person one above takes two targets", func(t *testing.T) {
		t.Parallel()
		res := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 3, 0), seTarget("a", nil), seTarget("b", nil)), dicetest.NewFixedRoller(5, 5))
		if len(res.Targets()) != 2 {
			t.Errorf("targets = %d, want 2 [N5]", len(res.Targets()))
		}
	})
}

// Requirement: spell-effects.md FR-043, FR-032 - a wider slot grows every count and amount from its own AboveLevel: Magic Missile gets one dart more per slot above 1 and Cure Wounds one 2d8 more.
// Path: unhappy (boundary).
func TestPerSlotCountsAndAmountsGrow(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(slices.Repeat([]int{2}, 4)...)
	res := seResolved(t, seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil)), roller)
	if got := len(res.Parts()[0].Instances()); got != 4 {
		t.Errorf("darts = %d, want 4", got)
	}
	heal := dicetest.NewMaximumRoller()
	cure := seResolved(t, seCommand(seCureWounds(), seReport("cure_wounds", 3, 0), seTarget("a", nil)), heal)
	if got := heal.Requests()[0]; got != dice.MustExpression(6, dice.D8(), 0) {
		t.Errorf("Cure Wounds at 3 rolled %v, want 6d8", got)
	}
	if got := cure.Targets()[0].Healing(); len(got) != 1 || got[0].Amount() != 6*8+3 {
		t.Errorf("healing = %v, want 6d8 + 3", got)
	}
}

// Requirement: spell-effects.md table I (I1 to I5, I8, I9), FR-017, FR-094 - a target outside the creature type restriction, an object under a creature type restriction or a target the host says cannot be affected gets no roll and a perception: AppearedToSucceed for a nothing-on-success save part, SpellDidNothing otherwise.
// Path: per case.
func TestInvalidTargetPerception(t *testing.T) {
	t.Parallel()
	beast := seTarget("a", func(p *combat.SpellTargetParams) { p.CreatureType = content.CreatureTypeBeast })
	cannot := func(id string) combat.SpellTarget {
		return seTarget(id, func(p *combat.SpellTargetParams) {
			p.Validity = combat.TargetCannotBeAffected
			p.AttackFacts = seFacts(nil)
		})
	}
	object := luMust(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.ObjectTargetRef{ID: combat.MustSpellObjectID("door")}}))
	humanoids := luMust(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
	objectsAndPeople := seSpell("restricted_burst", 1, seModel(content.CoverageComplete, seArea(content.SubjectCreatureOrObject, humanoids),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)}, seNothing), seInstances(1, nil))), nil)
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ResolveSpellEffectsCommand
		want                    combat.InvalidTargetPerception
		wantRequests            int
	}{
		{"I1 Hold Person on a Beast", "I1", "unhappy (rule interaction)", seCommand(seHoldPerson(), seReport("hold_person", 2, 0), beast), combat.AppearedToSucceed, 0},
		{"I2 Fireball cannot be affected", "I2", "unhappy (rule interaction)", seCommand(seFireball(), seReport("fireball", 3, 0), cannot("a")), combat.SpellDidNothing, 0},
		{"I3 Fire Bolt cannot be affected", "I3", "unhappy (rule interaction)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), cannot("a")), combat.SpellDidNothing, 0},
		{"I4 Cure Wounds cannot be affected", "I4", "unhappy (rule interaction)", seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), cannot("a")), combat.SpellDidNothing, 0},
		{"I8 an object under a creature type restriction", "I8", "unhappy (rule interaction)", seCommand(objectsAndPeople, seReport("restricted_burst", 1, 0), object), combat.AppearedToSucceed, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := spellEffectsSpec + c.requirement
			roller := dicetest.NewMinimumRoller()
			res := seResolved(t, c.cmd, roller)
			got, ok := res.Targets()[0].Perception()
			if !ok || got != c.want {
				t.Errorf("perception = %v, %v, want %v [%s]", got, ok, c.want, req)
			}
			if len(roller.Requests()) != c.wantRequests {
				t.Errorf("requests = %v, want none: an invalid target gets no roll and no Damage [%s]", roller.Requests(), req)
			}
			if e := res.Targets()[0]; len(e.Damage()) != 0 || len(e.Healing()) != 0 || len(e.Conditions()) != 0 {
				t.Errorf("effects on an invalid target: %+v [%s]", e, req)
			}
		})
	}
	t.Run("I5 Hold Person on a Humanoid rolls the save", func(t *testing.T) {
		t.Parallel()
		roller := dicetest.NewFixedRoller(5)
		res := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), roller)
		if _, ok := res.Targets()[0].Perception(); ok || len(roller.Requests()) != 1 {
			t.Errorf("a valid target has no perception and rolls: %v %v [I5]", ok, roller.Requests())
		}
	})
	t.Run("I9 Ice Knife on an object is rejected", func(t *testing.T) {
		t.Parallel()
		got := seRejected(t, seCommand(seIceKnife(), seReport("ice_knife", 1, 0), object), dicetest.NewMinimumRoller())
		if !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.WrongTargetSubject}) {
			t.Errorf("reasons = %v, want WrongTargetSubject [I9]", got)
		}
	})
}

// Requirement: spell-effects.md table R (R1 to R10, R16, R17, R21 to R23, R26, R27), FR-057, FR-058 - every problem is listed once in the fixed order, and a rejection is returned before any die is requested.
// Path: per case.
func TestRejections(t *testing.T) {
	t.Parallel()
	dexOnly := func(id string) combat.SpellTarget {
		return seTarget(id, func(p *combat.SpellTargetParams) { p.Saves = nil })
	}
	blocked := func(id string) combat.SpellTarget {
		return seTarget(id, func(p *combat.SpellTargetParams) { p.Path = combat.PathBlockedByTotalCover })
	}
	assign := func(part int, ids ...string) combat.InstanceAssignment {
		var refs []combat.SpellTargetRef
		for _, id := range ids {
			refs = append(refs, seRef(id))
		}
		return luMust(combat.NewInstanceAssignment(part, refs))
	}
	withAssign := func(cmd combat.ResolveSpellEffectsCommand, a ...combat.InstanceAssignment) combat.ResolveSpellEffectsCommand {
		cmd.Assignments = a
		return cmd
	}
	withType := func(cmd combat.ResolveSpellEffectsCommand, t content.DamageType) combat.ResolveSpellEffectsCommand {
		cmd.ChosenDamageType = t
		return cmd
	}
	totalCover := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) {
			f.CoverSources = []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
		})
	})
	tooFar := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) { f.Distance = 500 })
	})
	object := luMust(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.ObjectTargetRef{ID: combat.MustSpellObjectID("door")}}))
	objectSave := seSpell("any_burst", 1, seModel(content.CoverageComplete, seArea(content.SubjectCreatureOrObject, nil),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)}, seNothing), seInstances(1, nil))), nil)
	hold := func(at int, ts ...combat.SpellTarget) combat.ResolveSpellEffectsCommand {
		return seCommand(seHoldPerson(), seReport("hold_person", at, 0), ts...)
	}
	wcc, ncp, iam, msa, maf := combat.WrongTargetCount, combat.NoClearPath, combat.InstanceAssignmentMismatch, combat.MissingSaveAttempt, combat.MissingAttackFacts
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ResolveSpellEffectsCommand
		want                    []combat.SpellEffectsRejectionKind
	}{
		{"R1 Hold Person at 3 with 3 targets", "R1 FR-019", "unhappy (boundary)", hold(3, seTarget("a", nil), seTarget("b", nil), seTarget("c", nil)), []combat.SpellEffectsRejectionKind{wcc}},
		{"R3 Fire Bolt blocked by Total Cover", "R3 FR-018", "unhappy (state error)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1),
			seTarget("a", func(p *combat.SpellTargetParams) {
				p.Path = combat.PathBlockedByTotalCover
				p.AttackFacts = seFacts(nil)
			})), []combat.SpellEffectsRejectionKind{ncp}},
		{"R4 Magic Missile 4 darts and 3 assignments", "R4 FR-031", "unhappy (invalid input)", withAssign(seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil), seTarget("b", nil)), assign(0, "a", "a", "b")), []combat.SpellEffectsRejectionKind{iam}},
		{"R5 assignment names a creature that is not a target", "R5 FR-031", "unhappy (invalid input)", withAssign(seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil), seTarget("b", nil)), assign(0, "a", "a", "b", "z")), []combat.SpellEffectsRejectionKind{iam}},
		{"R5 no assignment for several targets", "R5 FR-031", "unhappy (invalid input)", seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil), seTarget("b", nil)), []combat.SpellEffectsRejectionKind{iam}},
		{"R16 target C unassigned", "R16 FR-031", "unhappy (rule interaction)", withAssign(seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil), seTarget("b", nil), seTarget("c", nil)), assign(0, "a", "a", "b", "b")), []combat.SpellEffectsRejectionKind{iam}},
		{"assignment for a single plain instance", "FR-031", "unhappy (rule interaction)", withAssign(seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil)), assign(0, "a")), []combat.SpellEffectsRejectionKind{iam}},
		{"R6 Fireball without a Dexterity save", "R6 FR-059", "unhappy (missing content)", seCommand(seFireball(), seReport("fireball", 3, 0), dexOnly("a")), []combat.SpellEffectsRejectionKind{msa}},
		{"R7 Fire Bolt without attack facts", "R7 FR-059", "unhappy (missing content)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seTarget("a", nil)), []combat.SpellEffectsRejectionKind{maf}},
		{"R8 Chromatic Orb with no chosen type", "R8 FR-033", "unhappy (invalid input)", seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a")), []combat.SpellEffectsRejectionKind{combat.DamageTypeNotChosen}},
		{"R9 Chromatic Orb with Radiant", "R9 FR-033", "unhappy (invalid input)", withType(seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a")), content.DamageRadiant), []combat.SpellEffectsRejectionKind{combat.DamageTypeNotAllowed}},
		{"R10 Hold Person at 2 with 3 targets, one blocked, no save", "R10 FR-057", "unhappy (rule interaction)", hold(2, dexOnly("a"), blocked("b"), seTarget("c", nil)), []combat.SpellEffectsRejectionKind{wcc, ncp, msa}},
		{"R17 caster listed for a spell that excludes the caster", "R17 FR-092", "unhappy (rule interaction)", seCommand(seWardenBolt(), seReport("warden_bolt", 1, 0), seTarget("caster", nil)), []combat.SpellEffectsRejectionKind{combat.CasterNotAllowed}},
		{"R21 an object target of a save part", "R21 FR-093", "unhappy (missing content)", seCommand(objectSave, seReport("any_burst", 1, 0), object), []combat.SpellEffectsRejectionKind{msa}},
		{"R22 Ice Knife on an object", "R22 FR-094", "unhappy (invalid input)", seCommand(seIceKnife(), seReport("ice_knife", 1, 0), object), []combat.SpellEffectsRejectionKind{combat.WrongTargetSubject}},
		{"WrongTargetSubject for a creature of an object spell", "FR-094", "unhappy (invalid input)", seCommand(seObjectOnly(), seReport("lock_bolt", 1, 0), seTarget("a", nil)), []combat.SpellEffectsRejectionKind{combat.WrongTargetSubject}},
		{"R23 attack against Total Cover", "R23 FR-022", "unhappy (state error)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), totalCover), []combat.SpellEffectsRejectionKind{combat.SpellAttackNotPossible}},
		{"R23 attack beyond range", "R23 FR-022", "unhappy (boundary)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), tooFar), []combat.SpellEffectsRejectionKind{combat.SpellAttackNotPossible}},
		{"R26 False Life with no target", "R26 FR-019", "unhappy (empty and minimal)", seCommand(seFalseLife(), seReport("false_life", 1, 0)), []combat.SpellEffectsRejectionKind{wcc}},
		{"R27 False Life on another creature", "R27 FR-019", "unhappy (invalid input)", seCommand(seFalseLife(), seReport("false_life", 1, 0), seTarget("other", nil)), []combat.SpellEffectsRejectionKind{wcc}},
		{"R27 False Life on an object", "R27 FR-019", "unhappy (invalid input)", seCommand(seFalseLife(), seReport("false_life", 1, 0), object), []combat.SpellEffectsRejectionKind{wcc}},
		{"Hold Person with no target", "FR-019", "unhappy (empty and minimal)", hold(2), []combat.SpellEffectsRejectionKind{wcc}},
		{"Chromatic Orb with two targets and no assignment", "FR-019 FR-031", "unhappy (boundary)", withType(seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a"), seWithFacts("b")), content.DamageFire), []combat.SpellEffectsRejectionKind{wcc, iam}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := spellEffectsSpec + c.requirement
			roller := dicetest.NewMinimumRoller()
			if got := seRejected(t, c.cmd, roller); !slices.Equal(got, c.want) {
				t.Errorf("reasons = %v, want %v [%s]", got, c.want, req)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("requests = %v, want none before a rejection (FR-058) [%s]", roller.Requests(), req)
			}
		})
	}
}

// seObjectOnly is a spell for objects only.
func seObjectOnly() content.Spell {
	return seSpell("lock_bolt", 1, seModel(content.CoverageComplete, seChosen(content.SubjectObject, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)}, seNothing), seInstances(1, nil))), nil)
}

// Requirement: spell-effects.md R2, R18, R20, R28, FR-019, FR-092, FR-093 - the boundary of the target count, the caster as the target of a spell that may include the caster, an object as the target of a Fire Bolt and the caster alone for a self spell all resolve.
// Path: per case.
func TestValidTargetLists(t *testing.T) {
	t.Parallel()
	object := luMust(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.ObjectTargetRef{ID: combat.MustSpellObjectID("door")}, AttackFacts: seFacts(nil)}))
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ResolveSpellEffectsCommand
		faces                   []int
		wantTargets             int
	}{
		{"R2 Hold Person at 3 with 2 targets", "R2", "unhappy (boundary)", seCommand(seHoldPerson(), seReport("hold_person", 3, 0), seTarget("a", nil), seTarget("b", nil)), []int{5, 5}, 2},
		{"R18 Cure Wounds on the caster", "R18", "happy", seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), seTarget("caster", nil)), []int{4, 4}, 1},
		{"R20 Fire Bolt on an object", "R20", "happy", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), object), []int{15, 5}, 1},
		{"R28 False Life on the caster", "R28", "happy", seCommand(seFalseLife(), seReport("false_life", 1, 0), seTarget("caster", nil)), []int{2, 2}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := seResolved(t, c.cmd, dicetest.NewFixedRoller(c.faces...))
			if len(res.Targets()) != c.wantTargets {
				t.Errorf("targets = %d, want %d [%s]", len(res.Targets()), c.wantTargets, c.requirement)
			}
		})
	}
}

// Requirement: spell-effects.md R13, R19, O11, FR-020, Q17 - a TargetsInArea cast with no creature in the area, and one whose every listed target is invalid, resolve with no roll and no die requested (the invalid targets keep their perception).
// Path: unhappy (empty and minimal).
func TestEmptyAreaRequestsNoDie(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewMinimumRoller()
	res := seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0)), roller)
	if len(res.Targets()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("empty area: targets %d, requests %v, want none [R13]", len(res.Targets()), roller.Requests())
	}
	cannot := func(id string) combat.SpellTarget {
		return seTarget(id, func(p *combat.SpellTargetParams) { p.Validity = combat.TargetCannotBeAffected })
	}
	res = seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0), cannot("a"), cannot("b")), roller)
	if len(roller.Requests()) != 0 || len(res.Parts()[0].Instances()[0].Saves()) != 0 {
		t.Errorf("all invalid: requests %v, want none [R19]", roller.Requests())
	}
	for i, e := range res.Targets() {
		if p, ok := e.Perception(); !ok || p != combat.SpellDidNothing {
			t.Errorf("target %d perception = %v, %v", i, p, ok)
		}
	}
}

// Requirement: spell-effects.md table O (O1 to O12), FR-041, FR-029, FR-030, FR-065 - with a fixed roller the dice are requested in the order the spec fixes; an attack rolls its damage per hit, a Critical Hit doubles the dice and never the flat modifier, and a PlannedDamage is one per part, instance and target.
// Path: per case.
func TestRollOrder(t *testing.T) {
	t.Parallel()
	d := func(n int, die dice.Die, mod int) dice.Expression { return dice.MustExpression(n, die, mod) }
	d20x := d(1, dice.D20(), 0)
	oneEightPlusTwo := seSpell("scorching_ray", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d8+2"), nil), seFire, content.AddsNothing)}, nil), seInstances(1, nil))), nil)
	cases := []struct {
		name, requirement, path string
		cmd                     combat.ResolveSpellEffectsCommand
		faces                   []int
		want                    []dice.Expression
		wantPlans               []int // PlannedDamage values per target
	}{
		{"O1 Fireball A and B", "O1", "happy", seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil), seTarget("b", nil)),
			slices.Concat(seEightD6, []int{5, 19}), []dice.Expression{d(8, dice.D6(), 0), d20x, d20x}, []int{1, 1}},
		{"O2 Flame Strike", "O2", "unhappy (rule interaction)", seCommand(seFlameStrike(), seReport("flame_strike", 5, 0), seTarget("a", nil)),
			slices.Concat(seFaces(1, 10), []int{5}), []dice.Expression{d(5, dice.D6(), 0), d(5, dice.D6(), 0), d20x}, []int{1}},
		{"O3 Eldritch Blast both hit", "O3", "happy", withAssignments(seCommand(seEldritchBlast(), seReport("eldritch_blast", 0, 5), seWithFacts("a"), seWithFacts("b")), "a", "b"),
			[]int{15, 6, 15, 7}, []dice.Expression{d20x, d(1, dice.D10(), 0), d20x, d(1, dice.D10(), 0)}, []int{1, 1}},
		{"O4 Eldritch Blast first beam misses", "O4", "unhappy (rule interaction)", withAssignments(seCommand(seEldritchBlast(), seReport("eldritch_blast", 0, 5), seWithFacts("a"), seWithFacts("b")), "a", "b"),
			[]int{2, 15, 7}, []dice.Expression{d20x, d20x, d(1, dice.D10(), 0)}, []int{0, 1}},
		{"O5 Magic Missile four darts", "O5", "happy", withAssignments(seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil), seTarget("b", nil)), "a", "a", "b", "b"),
			[]int{1, 2, 3, 4}, slices.Repeat([]dice.Expression{d(1, dice.D4(), 1)}, 4), []int{2, 2}},
		{"O6 Cure Wounds", "O6", "happy", seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), seTarget("a", nil)),
			[]int{4, 4}, []dice.Expression{d(2, dice.D8(), 0)}, []int{0}},
		{"O7 Hold Person", "O7", "happy", seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), []int{5}, []dice.Expression{d20x}, []int{0}},
		{"O8 Fire Bolt hit", "O8", "happy", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), []int{15, 6}, []dice.Expression{d20x, d(1, dice.D10(), 0)}, []int{1}},
		{"O9 Fire Bolt Critical Hit", "O9", "unhappy (boundary)", seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), []int{20, 6, 7}, []dice.Expression{d20x, d(2, dice.D10(), 0)}, []int{1}},
		{"O10 Eldritch Blast hit then Critical Hit", "O10", "unhappy (rule interaction)", withAssignments(seCommand(seEldritchBlast(), seReport("eldritch_blast", 0, 5), seWithFacts("a")), "a", "a"),
			[]int{15, 6, 20, 7, 8}, []dice.Expression{d20x, d(1, dice.D10(), 0), d20x, d(2, dice.D10(), 0)}, []int{2}},
		{"O12 flat modifier on a Critical Hit", "O12", "unhappy (boundary)", seCommand(oneEightPlusTwo, seReport("scorching_ray", 1, 0), seWithFacts("a")),
			[]int{20, 3, 4}, []dice.Expression{d20x, d(2, dice.D8(), 0)}, []int{1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := spellEffectsSpec + c.requirement
			roller := dicetest.NewFixedRoller(c.faces...)
			res := seResolved(t, c.cmd, roller)
			if !slices.Equal(roller.Requests(), c.want) {
				t.Errorf("requests = %v, want %v [%s]", roller.Requests(), c.want, req)
			}
			for i, want := range c.wantPlans {
				if got := len(seDamages(res, i)); got != want {
					t.Errorf("target %d has %d PlannedDamage values, want %d [%s]", i, got, want, req)
				}
			}
		})
	}
	t.Run("O12 the flat modifier is added once", func(t *testing.T) {
		t.Parallel()
		res := seResolved(t, seCommand(oneEightPlusTwo, seReport("scorching_ray", 1, 0), seWithFacts("a")), dicetest.NewFixedRoller(20, 3, 4))
		got := seOnly(t, res, 0)
		if got.Damages()[0].Amount() != 3+4+2 || got.HitKind() != combat.CriticalHit {
			t.Errorf("damage = %v kind %v, want 9 from 2d8 and +2 once, a Critical Hit", got.Damages(), got.HitKind())
		}
	})
	t.Run("O10 two PlannedDamage values, normal then critical, in instance order", func(t *testing.T) {
		t.Parallel()
		res := seResolved(t, withAssignments(seCommand(seEldritchBlast(), seReport("eldritch_blast", 0, 5), seWithFacts("a")), "a", "a"), dicetest.NewFixedRoller(15, 6, 20, 7, 8))
		got := seDamages(res, 0)
		if got[0].HitKind() != combat.NormalHit || got[0].Damages()[0].Amount() != 6 || got[1].HitKind() != combat.CriticalHit || got[1].Damages()[0].Amount() != 15 ||
			got[0].InstanceIndex() != 0 || got[1].InstanceIndex() != 1 || got[1].Damages()[0].Type() != content.DamageForce {
			t.Errorf("plan = %v, want Force 6 then a critical 15 (FR-065)", got)
		}
	})
	t.Run("O5 a PlannedDamage per dart", func(t *testing.T) {
		t.Parallel()
		res := seResolved(t, withAssignments(seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil), seTarget("b", nil)), "a", "a", "b", "b"), dicetest.NewFixedRoller(1, 2, 3, 4))
		var amounts [][]int
		for i := range 2 {
			var row []int
			for _, pd := range seDamages(res, i) {
				row = append(row, pd.Damages()[0].Amount())
			}
			amounts = append(amounts, row)
		}
		if !slices.Equal(amounts[0], []int{2, 3}) || !slices.Equal(amounts[1], []int{4, 5}) {
			t.Errorf("amounts = %v, want [2 3] and [4 5] (1d4+1 each)", amounts)
		}
	})
}

// withAssignments assigns the instances of part 0 to ids.
func withAssignments(cmd combat.ResolveSpellEffectsCommand, ids ...string) combat.ResolveSpellEffectsCommand {
	var refs []combat.SpellTargetRef
	for _, id := range ids {
		refs = append(refs, seRef(id))
	}
	cmd.Assignments = []combat.InstanceAssignment{luMust(combat.NewInstanceAssignment(0, refs))}
	return cmd
}

// Requirement: spell-effects.md FR-022, FR-023, R25, O8 - an attack is compared with the target's Armor Class plus the bonus of the applicable cover (Half Cover: 2), a miss applies OnMiss (nothing here), the evidence holds the d20 result unchanged, and a natural 20 is a Critical Hit.
// Path: unhappy (boundary, rule interaction).
func TestSpellAttackAgainstArmorClassAndCover(t *testing.T) {
	t.Parallel()
	half := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) {
			f.CoverSources = []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
		})
	})
	cases := []struct {
		name, requirement, path string
		target                  combat.SpellTarget
		natural                 int
		wantHit, wantCritical   bool
		wantTarget              d20.ArmorClass
	}{
		{"total equals AC", "FR-022", "unhappy (boundary)", seWithFacts("a"), 6, true, false, 12},     // 6 + 6 = 12
		{"total one below AC", "FR-022", "unhappy (boundary)", seWithFacts("a"), 5, false, false, 12}, // 5 + 6 = 11
		{"R25 Half Cover raises AC by 2", "R25", "unhappy (rule interaction)", half, 7, false, false, 14},
		{"R25 Half Cover at the new total", "R25", "unhappy (boundary)", half, 8, true, false, 14},
		{"natural 20", "FR-023", "unhappy (boundary)", seWithFacts("a"), 20, true, true, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			faces := []int{c.natural}
			if c.wantHit {
				faces = append(faces, 5, 5)
			}
			res := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), c.target), dicetest.NewFixedRoller(faces...))
			inst := res.Parts()[0].Instances()[0]
			atk, ok := inst.Attack()
			if !ok || inst.Hit() != c.wantHit || inst.CriticalHit() != c.wantCritical || atk.Roll().Natural() != c.natural {
				t.Fatalf("hit %v critical %v attack %v, want hit %v critical %v [%s]", inst.Hit(), inst.CriticalHit(), ok, c.wantHit, c.wantCritical, c.requirement)
			}
			if got := len(seDamages(res, 0)); (got == 1) != c.wantHit {
				t.Errorf("PlannedDamage values = %d, want damage exactly on a hit [%s]", got, c.requirement)
			}
		})
	}
}

// Requirement: spell-effects.md FR-022, FR-097 - the Advantage and Disadvantage of the facts (here a target the caster cannot see: Disadvantage) are resolved once, so the attack rolls two d20s, and Advantage and Disadvantage together roll one.
// Path: unhappy (rule interaction).
func TestSpellAttackRollMode(t *testing.T) {
	t.Parallel()
	unseen := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) {
			f.Situation = combat.AttackSituation{Visibility: combat.TargetUnseen{TargetAtTargetedLocation: true}}
		})
	})
	both := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) {
			f.Situation = combat.AttackSituation{Visibility: combat.TargetUnseen{TargetAtTargetedLocation: true}, TargetCannotSeeAttacker: true}
		})
	})
	roller := dicetest.NewFixedRoller(18, 3)
	res := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), unseen), roller)
	atk, _ := res.Parts()[0].Instances()[0].Attack()
	if atk.Roll().Natural() != 3 || len(atk.Roll().Dice()) != 2 {
		t.Errorf("Disadvantage rolled %v, want the lower of two d20s", atk.Roll().Dice())
	}
	roller = dicetest.NewFixedRoller(18, 4)
	res = seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), both), roller)
	atk, _ = res.Parts()[0].Instances()[0].Attack()
	if len(atk.Roll().Dice()) != 1 || roller.Requests()[0] != dice.MustExpression(1, dice.D20(), 0) {
		t.Errorf("Advantage and Disadvantage together: %v, want one d20", atk.Roll().Dice())
	}
}

// Requirement: spell-effects.md FR-023, FR-097 - a hit that the conditions of the target make automatic is a Critical Hit whatever the d20 shows, and a Critical Hit still doubles only the dice.
// Path: unhappy (rule interaction).
func TestAutomaticCriticalHit(t *testing.T) {
	t.Parallel()
	paralyzed := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) { f.AutomaticCritical = combat.AutomaticCriticalAutomatic })
	})
	roller := dicetest.NewFixedRoller(15, 3, 4)
	res := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), paralyzed), roller)
	if d := seOnly(t, res, 0); d.HitKind() != combat.CriticalHit || d.Damages()[0].Amount() != 7 {
		t.Errorf("plan = %v, want a critical 3 + 4", d)
	}
}

// Requirement: spell-effects.md FR-032, FR-036, FR-037, Q10 - an amount adds the spellcasting ability modifier once when its outcome says so, a negative total is raised to 0, and an amount of 0 produces no healing and no Temporary Hit Points.
// Path: unhappy (boundary).
func TestAmountModifierAndFloor(t *testing.T) {
	t.Parallel()
	temp := seSpell("sturdy_ward", 1, seModel(content.CoverageComplete, content.TargetsSelf{}, sePart(seAuto(
		luMust(content.NewSpellGrantsTemporaryHitPoints(seAmount(seDice("1d4"), nil), content.AddsSpellcastingAbilityModifier))), seInstances(1, nil))), nil)
	cases := []struct {
		name  string
		spell content.Spell
		mod   int
		faces []int
		want  int
	}{
		{"healing adds the modifier", seCureWounds(), 3, []int{1, 1}, 5},
		{"a negative modifier lowers the healing", seCureWounds(), -1, []int{3, 3}, 5},
		{"healing raised to 0 gives no outcome", seCureWounds(), -5, []int{1, 1}, 0},
		{"Temporary Hit Points of 0 give no outcome", temp, -9, []int{2}, 0},
		{"Temporary Hit Points with the modifier", temp, 2, []int{2}, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := seCommand(c.spell, seReport(string(c.spell.ID()), 1, 0), seTarget("caster", nil))
			cmd.AbilityModifier = c.mod
			e := seResolved(t, cmd, dicetest.NewFixedRoller(c.faces...)).Targets()[0]
			got := slices.Concat(e.Healing(), e.TemporaryHitPoints())
			if c.want == 0 {
				if len(got) != 0 {
					t.Errorf("amounts = %v, want none [FR-036, FR-037]", got)
				}
				return
			}
			if len(got) != 1 || got[0].Amount() != c.want {
				t.Errorf("amounts = %v, want %d [FR-032]", got, c.want)
			}
		})
	}
	t.Run("damage with the modifier is floored at 0 by the composer", func(t *testing.T) {
		t.Parallel()
		cmd := seCommand(seWardenBolt(), seReport("warden_bolt", 1, 0), seTarget("a", nil))
		cmd.AbilityModifier = -20
		res := seResolved(t, cmd, dicetest.NewFixedRoller(1, 1))
		if d := seOnly(t, res, 0); d.Damages()[0].Amount() != 0 {
			t.Errorf("damage = %v, want 0, never negative", d.Damages())
		}
	})
}

// Requirement: spell-effects.md FR-033, FR-035, R8 - the Damage Type the caster chose is the type of the damage; an unexpected choice is a contradictory command.
// Path: happy, unhappy (invalid input).
func TestChosenDamageType(t *testing.T) {
	t.Parallel()
	cmd := seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a"))
	cmd.ChosenDamageType = content.DamageThunder
	res := seResolved(t, cmd, dicetest.NewFixedRoller(15, 1, 1, 1))
	if d := seOnly(t, res, 0); d.Damages()[0].Type() != content.DamageThunder {
		t.Errorf("type = %v, want Thunder", d.Damages()[0].Type())
	}
	bad := seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil))
	bad.ChosenDamageType = content.DamageFire
	_, err := seRun(t, bad, dicetest.NewMinimumRoller())
	if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("R11 err = %v, want ErrInvalidSpellEffectsCommand", err)
	}
	bad.ChosenDamageType = content.DamageType(99)
	if _, err := seRun(t, bad, dicetest.NewMinimumRoller()); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Errorf("invalid type err = %v", err)
	}
}

// Requirement: spell-effects.md FR-038 - a condition outcome plans a ConditionToApply naming the target, the kind, the caster as the source and the duration; ToCondition gives the Condition of conditions.md: the caster as the source of Charmed, Frightened and Grappled, Exhaustion at level 1, every other kind as itself.
// Path: per case.
func TestConditionToApply(t *testing.T) {
	t.Parallel()
	caster := seCaster
	cases := []struct {
		kind content.ConditionKind
		want combat.Condition
	}{
		{content.ConditionBlinded, combat.Blinded{}}, {content.ConditionCharmed, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: caster}}},
		{content.ConditionDeafened, combat.Deafened{}}, {content.ConditionExhaustion, combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(1)}},
		{content.ConditionFrightened, combat.Frightened{Source: caster}}, {content.ConditionGrappled, combat.Grappled{Grappler: caster}},
		{content.ConditionIncapacitated, combat.Incapacitated{}}, {content.ConditionInvisible, combat.Invisible{}}, {content.ConditionParalyzed, combat.Paralyzed{}},
		{content.ConditionPetrified, combat.Petrified{}}, {content.ConditionPoisoned, combat.Poisoned{}}, {content.ConditionProne, combat.Prone{}},
		{content.ConditionRestrained, combat.Restrained{}}, {content.ConditionStunned, combat.Stunned{}}, {content.ConditionUnconscious, combat.Unconscious{}},
	}
	timed := luMust(content.NewTimedConditionDuration(1, content.DurationUnitMinutes))
	for _, c := range cases {
		t.Run(c.kind.String(), func(t *testing.T) {
			t.Parallel()
			spell := seSpell("curse_test", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
				sePart(seAuto(luMust(content.NewSpellAppliesCondition(c.kind, timed))), seInstances(1, nil))), nil)
			res := seResolved(t, seCommand(spell, seReport("curse_test", 1, 0), seTarget("a", nil)), dicetest.NewMinimumRoller())
			conds := res.Targets()[0].Conditions()
			if len(conds) != 1 || conds[0].ToCondition() != c.want || conds[0].Kind() != c.kind || conds[0].Duration() != timed ||
				conds[0].Source() != seRef("caster") || conds[0].Target() != seRef("a") || conds[0].PartIndex() != 0 {
				t.Errorf("conditions = %+v, want %#v from the caster [FR-038]", conds, c.want)
			}
		})
	}
}

// Requirement: spell-effects.md FR-040 - the result is identical for any Spell.duration and any RequiresConcentration value, all else equal.
// Path: unhappy (state error: out of this spec).
func TestResultIgnoresDurationAndConcentration(t *testing.T) {
	t.Parallel()
	until := luMust(content.NewSpellDurationTimed(10, content.DurationUnitMinutes, content.LimitConcentrationUpTo))
	model := seFireball()
	other := seSpell("fireball", 3, model.Effects(), func(f *content.SpellFields) { f.Duration = until })
	if other.Duration() == model.Duration() || !other.RequiresConcentration() || model.RequiresConcentration() {
		t.Fatal("the fixtures differ in duration")
	}
	mk := func(s content.Spell) combat.SpellEffectsResolved {
		return seResolved(t, seCommand(s, seReport("fireball", 3, 0), seTarget("a", nil)), dicetest.NewFixedRoller(slices.Concat(seEightD6, []int{5})...))
	}
	if !mk(model).Equal(mk(other)) {
		t.Error("two spells that differ only in duration give different results")
	}
}

// Requirement: spell-effects.md FR-098, FR-050 - PartIndexes resolves only the named parts and InstanceCount replaces their instances; an out-of-range index, an empty or repeated set, InstanceCount alone or a negative count is a contradictory command; the target count is replaced by at least one creature target; a cast skips the part a recurring rule of the zone names.
// Path: per case.
func TestPartIndexesAndInstanceCount(t *testing.T) {
	t.Parallel()
	two := seSpell("twin_effect", 1, seModelWithZone(), nil)
	cmd := func(edit func(*combat.ResolveSpellEffectsCommand)) combat.ResolveSpellEffectsCommand {
		c := seCommand(two, seReport("twin_effect", 1, 0), seTarget("a", nil))
		edit(&c)
		return c
	}
	t.Run("a cast skips the recurring part", func(t *testing.T) {
		t.Parallel()
		roller := dicetest.NewFixedRoller(2)
		res := seResolved(t, cmd(func(*combat.ResolveSpellEffectsCommand) {}), roller)
		if parts := res.Parts(); len(parts) != 1 || parts[0].Index() != 0 || len(roller.Requests()) != 1 {
			t.Errorf("parts %v requests %v, want only part 0", parts, roller.Requests())
		}
	})
	t.Run("PartIndexes names the due part and InstanceCount repeats it", func(t *testing.T) {
		t.Parallel()
		roller := dicetest.NewFixedRoller(2, 3, 4)
		res := seResolved(t, cmd(func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes, c.InstanceCount = []int{1}, 3 }), roller)
		parts := res.Parts()
		if len(parts) != 1 || parts[0].Index() != 1 || len(parts[0].Instances()) != 3 || len(seDamages(res, 0)) != 3 {
			t.Errorf("parts %v, want part 1 with 3 instances and 3 PlannedDamage values", parts)
		}
	})
	t.Run("PartIndexes without InstanceCount keeps the model's instances", func(t *testing.T) {
		t.Parallel()
		res := seResolved(t, cmd(func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes = []int{1} }), dicetest.NewFixedRoller(2))
		if len(res.Parts()[0].Instances()) != 1 {
			t.Errorf("instances = %d, want 1", len(res.Parts()[0].Instances()))
		}
	})
	bad := map[string]func(*combat.ResolveSpellEffectsCommand){
		"index out of range":          func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes = []int{2} },
		"negative index":              func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes = []int{-1} },
		"empty set":                   func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes = []int{} },
		"repeated index":              func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes = []int{1, 1} },
		"InstanceCount without parts": func(c *combat.ResolveSpellEffectsCommand) { c.InstanceCount = 2 },
		"negative InstanceCount":      func(c *combat.ResolveSpellEffectsCommand) { c.PartIndexes, c.InstanceCount = []int{1}, -1 },
		"assignment of a missing part": func(c *combat.ResolveSpellEffectsCommand) {
			c.Assignments = withAssignments(*c, "a").Assignments[:0:0]
			c.Assignments = []combat.InstanceAssignment{luMust(combat.NewInstanceAssignment(7, []combat.SpellTargetRef{seRef("a")}))}
		},
		"two assignments of one part": func(c *combat.ResolveSpellEffectsCommand) {
			a := luMust(combat.NewInstanceAssignment(0, []combat.SpellTargetRef{seRef("a")}))
			c.Assignments = []combat.InstanceAssignment{a, a}
		},
	}
	for name, edit := range bad {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewMinimumRoller()
			_, err := seRun(t, cmd(edit), roller)
			if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) || len(roller.Requests()) != 0 {
				t.Errorf("err = %v requests %v, want ErrInvalidSpellEffectsCommand and no die", err, roller.Requests())
			}
		})
	}
	t.Run("a zone call needs a creature target", func(t *testing.T) {
		t.Parallel()
		object := luMust(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.ObjectTargetRef{ID: combat.MustSpellObjectID("door")}}))
		c := seCommand(two, seReport("twin_effect", 1, 0), object)
		c.PartIndexes = []int{1}
		if got := seRejected(t, c, dicetest.NewMinimumRoller()); !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.WrongTargetCount, combat.WrongTargetSubject}) {
			t.Errorf("reasons = %v, want WrongTargetCount and WrongTargetSubject", got)
		}
	})
}

// seModelWithZone is a two part model: an immediate 1d6 part and a part a creature entering a zone repeats.
func seModelWithZone() content.ModelledEffects {
	rule := luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, PartIndex: 1, Scope: content.AnyCreature}))
	zone := luMust(content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{rule}}))
	dmg := func() content.SpellOutcome {
		return seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)
	}
	m := seModel(content.CoveragePartial, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAuto(dmg()), seInstances(1, nil)), sePart(seAuto(dmg()), seInstances(1, nil)))
	return luMust(m.WithZoneFacts(zone))
}

// Requirement: spell-effects.md FR-050, FR-062, R14, R15, N8 - a report of another spell, a CastAtLevel below the spell's level, a cantrip model without ScalingLevel, a part that needs the save DC or the attack bonus the report lacks, no caster, a target without a ref and a target listed twice (R12) all return an error that is errors.Is ErrInvalidSpellEffectsCommand and srderr.ErrInvalidInput, with a nil result and no die.
// Path: per case.
func TestInvalidCommands(t *testing.T) {
	t.Parallel()
	noDC := luMust(character.NewCastReport(character.CastReportParts{Spell: "fireball", Source: character.CastSourceSlot, CastAtLevel: 3}))
	noBonus := luMust(character.NewCastReport(character.CastReportParts{Spell: "fire_bolt", Source: character.CastSourceSlot, ScalingLevel: 5}))
	cases := []struct {
		name, requirement string
		cmd               combat.ResolveSpellEffectsCommand
	}{
		{"R14 the report of another spell", "R14", seCommand(seFireball(), seReport("fire_bolt", 3, 0), seTarget("a", nil))},
		{"R15 Fireball at level 2", "R15", seCommand(seFireball(), seReport("fireball", 2, 0), seTarget("a", nil))},
		{"N8 a cantrip report without ScalingLevel", "N8", seCommand(seFireBolt(), seReport("fire_bolt", 0, 0), seWithFacts("a"))},
		{"a save part and no save DC", "FR-050", seCommand(seFireball(), noDC, seTarget("a", nil))},
		{"an attack part and no attack bonus", "FR-050", seCommand(seFireBolt(), noBonus, seWithFacts("a"))},
		{"R12 a creature listed twice", "R12", seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil), seTarget("a", nil))},
		{"no caster", "FR-062", func() combat.ResolveSpellEffectsCommand {
			c := seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil))
			c.Caster = ""
			return c
		}()},
		{"a target built as the zero value", "FR-062", seCommand(seFireball(), seReport("fireball", 3, 0), combat.SpellTarget{})},
		{"the zero spell", "FR-062", seCommand(content.Spell{}, seReport("fireball", 3, 0), seTarget("a", nil))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewMinimumRoller()
			res, err := seRun(t, c.cmd, roller)
			if res != nil || !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("res %v err %v, want a nil result and ErrInvalidSpellEffectsCommand [%s]", res, err, c.requirement)
			}
			if len(roller.Requests()) != 0 {
				t.Errorf("requests = %v, want none [%s]", roller.Requests(), c.requirement)
			}
		})
	}
}

// Requirement: spell-effects.md FR-043 - a count or amount that cannot be scaled for the cast is a contradictory command, not a panic: a damage expression that scales past the dice a Critical Hit can double.
// Path: unhappy (boundary).
func TestScalingPastTheDiceLimit(t *testing.T) {
	t.Parallel()
	grows := seSpell("endless_bolt", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d4"), sePerSlot(1, dice.MaxCount, 0)), seFire, content.AddsNothing)}, nil), seInstances(1, nil))), nil)
	_, err := seRun(t, seCommand(grows, seReport("endless_bolt", 9, 0), seWithFacts("a")), dicetest.NewMinimumRoller())
	if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Errorf("err = %v, want ErrInvalidSpellEffectsCommand", err)
	}
}
