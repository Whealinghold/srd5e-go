package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-012 to FR-016 and table T-D - the plan for one spell, one row per case (SRD 5.2.1, "Dispel Magic", p. 124; "Imprisonment", p. 142; "Exceptions Supersede General Rules", p. 5).
// Path: per case.
func TestDispelMagicPlanTableT_D(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		castLevel, spellLevel   int
		restriction             string
		want                    string
	}{
		{"cantrip with a duration", "FR-012", "happy", 3, 0, "none", "ends"},
		{"level 1", "FR-012", "happy", 3, 1, "none", "ends"},
		{"level 3 at the threshold", "FR-012", "unhappy (boundary)", 3, 3, "none", "ends"},
		{"level 4, slot 3", "FR-013", "unhappy (boundary)", 3, 4, "none", "needs_check:14"},
		{"level 5, slot 4", "FR-014", "happy", 4, 5, "none", "needs_check:15"},
		{"level 4, slot 4", "FR-013", "unhappy (boundary)", 4, 4, "none", "ends"},
		{"level 5, slot 5", "FR-013", "unhappy (boundary)", 5, 5, "none", "ends"},
		{"level 6, slot 5", "FR-014", "happy", 5, 6, "none", "needs_check:16"},
		{"level 9, slot 8", "FR-014", "unhappy (boundary)", 8, 9, "none", "needs_check:19"},
		{"level 9, slot 9", "FR-013", "unhappy (boundary)", 9, 9, "none", "ends"},
		{"level 2 spell upcast to 5, slot 3", "FR-013", "unhappy (rule interaction)", 3, 5, "none", "needs_check:15"},
		{"level 1 cast with a slot of 4, slot 3", "FR-013", "unhappy (rule interaction)", 3, 4, "none", "needs_check:14"},
		{"level 2, never dispellable", "FR-016", "unhappy (rule interaction)", 9, 2, "never", "blocked:never_dispellable"},
		{"level 9, slot 9 needed, slot 8", "FR-016", "unhappy (boundary)", 8, 9, "slot9", "blocked:slot_level_too_low"},
		{"level 9, slot 9 needed, slot 9", "FR-016", "happy", 9, 9, "slot9", "ends"},
		{"antimagic field spell, level 8", "FR-018", "unhappy (rule interaction)", 9, 8, "never", "blocked:never_dispellable"},
		{"after a part, a present", "FR-016", "unhappy (rule interaction)", 5, 2, "after_a_present", "blocked:parts_remain"},
		{"after a part, a absent", "FR-016", "happy", 5, 2, "after_a_absent", "ends"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			goblin := dspTarget(t, "goblin")
			restriction := dspRestriction(t, c.restriction)
			spec := dspSpec{effect: "x", spell: "bless", level: c.spellLevel, restriction: restriction}
			held := []combat.HeldSpell{dspHold(goblin, dspSpell(t, spec))}
			if c.restriction == "after_a_present" {
				held = append(held, dspHold(goblin, dspSpell(t, dspSpec{effect: "a", spell: "shield_of_faith", level: 1})))
			}
			plan := dspPlan(t, dspRegistry(t, held...), goblin, c.castLevel)
			if got := dspStepName(dspStepFor(t, plan, "x")); got != c.want {
				t.Errorf("step = %s, want %s [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// dspRestriction returns the restriction a T-D case names.
func dspRestriction(t *testing.T, name string) combat.DispelRestriction {
	t.Helper()
	switch name {
	case "never":
		return combat.DispelNeverDispellable{}
	case "slot9":
		return combat.NewDispelRequiresSlotLevel(character.MustSlotLevel(9))
	case "after_a_present", "after_a_absent":
		return combat.NewDispelAfterPartsGone([]combat.OngoingEffectID{dspEffectID(t, "a")})
	default:
		return combat.DispelUnrestricted{}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-015 - Dispel Magic is a level 3 spell, so a slot below 3 (or none) is refused (invalid input).
// Path: unhappy (invalid input).
func TestDispelMagicPlanRefusesCastLevelBelowThree(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	mod := ability.MustScore(14).Modifier()
	for name, level := range map[string]character.SlotLevel{"slot 2": character.MustSlotLevel(2), "no slot": {}} {
		_, err := combat.DispelMagic{}.Plan(dspRegistry(t), goblin, level, mod, d20.NewRollModifiers(modifier.New(0)))
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: Plan error = %v, want ErrInvalidInput [FR-015]", name, err)
		}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-011 - a target with no spells gets an empty plan, and a plan has one step per spell in registry order (empty and minimal).
// Path: per case.
func TestDispelMagicPlanOneStepPerSpellInOrder(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	empty := dspPlan(t, dspRegistry(t), goblin, 3)
	if len(empty.Steps()) != 0 {
		t.Errorf("empty target has %d steps, want 0 [FR-011]", len(empty.Steps()))
	}
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "first", spell: "bless", level: 1})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "second", spell: "haste", level: 2})),
	)
	steps := dspPlan(t, r, goblin, 3).Steps()
	if len(steps) != 2 {
		t.Fatalf("steps = %d, want 2 [FR-011]", len(steps))
	}
	if steps[0].Spell().Effect().ID() != dspEffectID(t, "first") || steps[1].Spell().Effect().ID() != dspEffectID(t, "second") {
		t.Errorf("steps are not in registry order [FR-011]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-028 - several spells from different casters on one target are planned separately, each with its own check (SRD 5.2.1, "Dispel Magic", "For each ongoing spell", p. 124).
// Path: unhappy (rule interaction).
func TestDispelMagicPlanPlansEachCasterSeparately(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "by-a", spell: "bless", level: 4, caster: "a"})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "by-b", spell: "shield_of_faith", level: 4, caster: "b"})),
	)
	steps := dspPlan(t, r, goblin, 3).Steps()
	if len(steps) != 2 {
		t.Fatalf("steps = %d, want one per caster [FR-028]", len(steps))
	}
	for _, s := range steps {
		if _, ok := s.(combat.DispelNeedsCheck); !ok {
			t.Errorf("step for %v is %T, want its own check [FR-028]", s.Spell().Effect().ID(), s)
		}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-029 - a caster's Concentration spell on another target is not a spell on the chosen target (SRD 5.2.1, "Dispel Magic", "on the target", p. 124).
// Path: unhappy (boundary).
func TestDispelMagicPlanIgnoresSpellsOnOtherTargets(t *testing.T) {
	t.Parallel()
	wizard, goblin := dspTarget(t, "wizard"), dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "held", spell: "haste", level: 3, caster: "wizard", concentration: true})))
	if got := len(dspPlan(t, r, wizard, 3).Steps()); got != 0 {
		t.Fatalf("caster's own plan has %d steps, want 0 [FR-029]", got)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-058 - a magical effect that is not a creature is keyed by DispelEffect; a creature with the same text is a different target (SRD 5.2.1, "Dispel Magic", "creature, object, or magical effect", p. 124).
// Path: unhappy (boundary).
func TestDispelMagicPlanKeysAWallByEffect(t *testing.T) {
	t.Parallel()
	wallID := dspEffectID(t, "wall")
	wall, err := combat.NewDispelEffect(wallID)
	if err != nil {
		t.Fatal(err)
	}
	r := dspRegistry(t, dspHold(wall, dspSpell(t, dspSpec{effect: "wall", spell: "wall_of_force"})))
	if got := len(dspPlan(t, r, wall, 3).Steps()); got != 1 {
		t.Fatalf("wall plan has %d steps, want 1 [FR-058]", got)
	}
	if got := len(dspPlan(t, r, dspTarget(t, "wall"), 3).Steps()); got != 0 {
		t.Fatalf("creature named wall has %d steps, want 0 [FR-058]", got)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-020 - the check adds only the spellcasting ability modifier and the given modifiers, no Proficiency Bonus (SRD 5.2.1, "Dispel Magic", p. 124; "Ability Checks", p. 6).
// Path: happy.
func TestDispelMagicPlanCarriesModifiersOnTheCheck(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 4})))
	mods := d20.NewRollModifiers(modifier.New(2))
	plan, err := combat.DispelMagic{}.Plan(r, goblin, character.MustSlotLevel(3), ability.MustScore(14).Modifier(), mods)
	if err != nil {
		t.Fatal(err)
	}
	need, ok := dspStepFor(t, plan, "x").(combat.DispelNeedsCheck)
	if !ok {
		t.Fatal("step is not a check [FR-020]")
	}
	if need.AbilityModifier() != ability.MustScore(14).Modifier() {
		t.Errorf("AbilityModifier() = %v, want the spellcasting modifier only [FR-020]", need.AbilityModifier())
	}
	if !need.Modifiers().Equal(mods) {
		t.Errorf("Modifiers() differ from the modifiers given [FR-020]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-027 - Plan changes neither input, returns equal plans for equal inputs, and returns copies (determinism, immutability).
// Path: unhappy (immutability).
func TestDispelMagicPlanIsPureAndCopies(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 4})))
	before := r
	first := dspPlan(t, r, goblin, 3)
	second := dspPlan(t, r, goblin, 3)
	if !r.Equal(before) {
		t.Fatal("Plan changed the registry [FR-027]")
	}
	if dspStepName(dspStepFor(t, first, "x")) != dspStepName(dspStepFor(t, second, "x")) {
		t.Error("equal inputs gave different plans [FR-027]")
	}
	steps := first.Steps()
	steps[0] = combat.DispelEnds{}
	if dspStepName(dspStepFor(t, first, "x")) != "needs_check:14" {
		t.Error("changing a returned step list changed the plan [FR-027]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-030 - a held readied spell is not in the registry, so a plan never includes it (SRD 5.2.1, "Ready [Action]", p. 187).
// Path: unhappy (rule interaction).
func TestDispelMagicPlanNeverIncludesAHeldSpell(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "on-goblin", spell: "bless"})))
	if got := len(dspPlan(t, r, dspTarget(t, "readied-target"), 3).Steps()); got != 0 {
		t.Fatalf("plan for a target with no spells has %d steps [FR-030]", got)
	}
}

// planSignature is the signature Plan must keep: no aura fact is a parameter (FR-019).
type planSignature func(combat.OngoingSpellRegistry, combat.DispelTarget, character.SlotLevel, ability.Modifier, d20.RollModifiers) (combat.DispelMagicPlan, error)

// Requirement: doc/specs/dispel-and-counterspell.md FR-019 - Plan takes no aura fact: its signature has no aura parameter, so a target the Antimagic Field covers is still planned, and the host asks AntimagicField.Blocks first (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: unhappy (rule interaction).
func TestDispelMagicPlanTakesNoAuraFact(t *testing.T) {
	t.Parallel()
	// The typed assignment fails to compile if Plan gains an aura parameter or changes a type.
	var plan planSignature = combat.DispelMagic{}.Plan
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1})))
	p, err := plan(r, goblin, character.MustSlotLevel(3), ability.MustScore(14).Modifier(), d20.NewRollModifiers(modifier.New(0)))
	if err != nil {
		t.Fatal(err)
	}
	if got := dspStepName(dspStepFor(t, p, "x")); got != "ends" {
		t.Errorf("step = %s, want ends [FR-019]", got)
	}
}
