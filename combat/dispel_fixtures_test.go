package combat_test

import (
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// dspSpec describes one ongoing spell for a test. Zero fields take the defaults
// of dspSpell: an ordinary spell with 60 seconds left, no caster, no restriction.
type dspSpec struct {
	effect        string
	spell         content.SpellID
	level         int
	caster        string
	origin        combat.SpellOrigin
	remaining     combat.SpellTimeRemaining
	restriction   combat.DispelRestriction
	partOf        string
	conditions    []content.ConditionKind
	concentration bool
}

// dspSpell builds the ongoing spell described by s, failing the test on error.
func dspSpell(t *testing.T, s dspSpec) combat.OngoingSpell {
	t.Helper()
	src, err := combat.SpellSource(s.spell)
	if err != nil {
		t.Fatal(err)
	}
	id, err := combat.NewOngoingEffectID(s.effect)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := combat.NewOngoingEffect(id, combat.OngoingEffectSpell, src)
	if err != nil {
		t.Fatal(err)
	}
	level, err := content.NewSpellLevel(s.level)
	if err != nil {
		t.Fatal(err)
	}
	p := combat.OngoingSpellParams{
		Effect:                eff,
		CastAtLevel:           level,
		Origin:                s.origin,
		Remaining:             s.remaining,
		Restriction:           s.restriction,
		Conditions:            s.conditions,
		RequiresConcentration: s.concentration,
	}
	if p.Origin == 0 {
		p.Origin = combat.SpellOriginOrdinary
	}
	if p.Remaining.IsZero() {
		p.Remaining = dspTimed(t, 60)
	}
	if p.Restriction == nil {
		p.Restriction = combat.DispelUnrestricted{}
	}
	if s.caster != "" {
		p.Caster = dspCreature(t, s.caster)
	}
	if s.partOf != "" {
		p.PartOf = dspEffectID(t, s.partOf)
	}
	sp, err := combat.NewOngoingSpell(p)
	if err != nil {
		t.Fatal(err)
	}
	return sp
}

// dspTimed returns a timed remaining duration of n seconds.
func dspTimed(t *testing.T, n int) combat.SpellTimeRemaining {
	t.Helper()
	r, err := combat.NewSpellTimeRemaining(n)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// dspCreature returns the CombatantID for id.
func dspCreature(t *testing.T, id string) combat.CombatantID {
	t.Helper()
	return combat.MustCombatantID(id)
}

// dspEffectID returns the OngoingEffectID for id.
func dspEffectID(t *testing.T, id string) combat.OngoingEffectID {
	t.Helper()
	eid, err := combat.NewOngoingEffectID(id)
	if err != nil {
		t.Fatal(err)
	}
	return eid
}

// dspTarget returns the creature target for id.
func dspTarget(t *testing.T, id string) combat.DispelTarget {
	t.Helper()
	tgt, err := combat.NewDispelCreature(dspCreature(t, id))
	if err != nil {
		t.Fatal(err)
	}
	return tgt
}

// dspHold returns the registry entry of sp on target.
func dspHold(target combat.DispelTarget, sp combat.OngoingSpell) combat.HeldSpell {
	return combat.HeldSpell{Target: target, Spell: sp}
}

// dspRegistry builds the registry of held, failing the test on error.
func dspRegistry(t *testing.T, held ...combat.HeldSpell) combat.OngoingSpellRegistry {
	t.Helper()
	r, err := combat.NewOngoingSpellRegistry(held)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// dspPlan plans Dispel Magic on target cast at castLevel, with a spellcasting
// modifier of +2 and no other modifiers.
func dspPlan(t *testing.T, r combat.OngoingSpellRegistry, target combat.DispelTarget, castLevel int) combat.DispelMagicPlan {
	t.Helper()
	plan, err := combat.DispelMagic{}.Plan(r, target, character.MustSlotLevel(castLevel), ability.MustScore(14).Modifier(), d20.NewRollModifiers(modifier.New(0)))
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// dspCheck resolves an ability check with a single natural roll and a flat modifier
// through the d20 domain function, as the host does through the resolver.
func dspCheck(t *testing.T, natural, flat int, dc d20.DifficultyClass) d20.CheckResult {
	t.Helper()
	roll, err := d20.NewRoll(modifier.Normal, []int{natural})
	if err != nil {
		t.Fatal(err)
	}
	res, err := d20.ResolveCheck(d20.AbilityCheck, roll, d20.NewRollModifiers(modifier.New(flat)), dc)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// dspChecks maps each effect id to the check given for it.
type dspChecks map[string]d20.CheckResult

// dspByEffect converts dspChecks to the map Conclude takes.
func dspByEffect(t *testing.T, c dspChecks) map[combat.OngoingEffectID]d20.CheckResult {
	t.Helper()
	out := make(map[combat.OngoingEffectID]d20.CheckResult, len(c))
	for k, v := range c {
		out[dspEffectID(t, k)] = v
	}
	return out
}

// dspStepFor returns the step of plan for the effect id.
func dspStepFor(t *testing.T, plan combat.DispelMagicPlan, effect string) combat.DispelStep {
	t.Helper()
	want := dspEffectID(t, effect)
	for _, s := range plan.Steps() {
		if s.Spell().Effect().ID() == want {
			return s
		}
	}
	t.Fatalf("no step for effect %q", effect)
	return nil
}

// dspStepName names the kind of a step: ends, needs_check:<dc>, or blocked:<reason>.
func dspStepName(s combat.DispelStep) string {
	switch v := s.(type) {
	case combat.DispelEnds:
		return "ends"
	case combat.DispelNeedsCheck:
		return "needs_check:" + strconv.Itoa(int(v.DC()))
	case combat.DispelBlocked:
		return "blocked:" + v.Reason().String()
	default:
		return "unknown"
	}
}
