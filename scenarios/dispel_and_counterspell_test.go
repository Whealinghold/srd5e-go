package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// dcSpell builds a synthetic ongoing spell for the scenarios: a spell id, an effect id, the level it was cast at and
// the optional restriction and Concentration facts.
func dcSpell(t *testing.T, effect string, spell content.SpellID, level int, restriction combat.DispelRestriction, caster combat.CombatantID) combat.OngoingSpell {
	t.Helper()
	id := must(combat.NewOngoingEffectID(effect))
	eff := must(combat.NewOngoingEffect(id, combat.OngoingEffectSpell, must(combat.SpellSource(spell))))
	p := combat.OngoingSpellParams{Effect: eff, CastAtLevel: must(content.NewSpellLevel(level)), Origin: combat.SpellOriginOrdinary,
		Remaining: must(combat.NewSpellTimeRemaining(60)), Restriction: restriction, Caster: caster, RequiresConcentration: !caster.IsZero()}
	return must(combat.NewOngoingSpell(p))
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-012, FR-013, FR-016, FR-021, FR-024 - Dispel Magic cast with a level 3 slot on a creature under a level 2 buff, a level 5 buff and a Forcecage ends the level 2 buff without a die, ends the level 5 buff when the d20 plus the modifier reaches DC 15, and never touches the Forcecage (SRD 5.2.1, "Dispel Magic", p. 124; "Forcecage", p. 134).
// Path: happy.
func TestDispelMagicOnABuffedCreature(t *testing.T) {
	t.Parallel()
	goblin := must(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
	wizard := combat.MustCombatantID("wizard")
	reg := must(combat.NewOngoingSpellRegistry([]combat.HeldSpell{
		{Target: goblin, Spell: dcSpell(t, "buff-2", "bless", 2, combat.DispelUnrestricted{}, combat.CombatantID(""))},
		{Target: goblin, Spell: dcSpell(t, "buff-5", "haste", 5, combat.DispelUnrestricted{}, wizard)},
		{Target: goblin, Spell: dcSpell(t, "cage", "forcecage", 7, combat.DispelNeverDispellable{}, combat.CombatantID(""))},
	}))
	roller := dicetest.NewFixedRoller(13)
	use := must(combat.NewDispelOngoingSpells(roller))
	res, err := use.Run(context.Background(), combat.DispelOngoingSpellsCommand{Registry: reg, Target: goblin, CastLevel: character.MustSlotLevel(3),
		AbilityModifier: ability.MustScore(14).Modifier(), Modifiers: d20.NewRollModifiers(modifier.New(0))})
	got, ok := res.(combat.OngoingSpellsDispelled)
	if err != nil || !ok {
		t.Fatalf("Run = %#v, %v, want OngoingSpellsDispelled", res, err)
	}
	left := got.Conclusion().Registry().SpellsOn(goblin)
	if len(left) != 1 || left[0].SpellID() != "forcecage" || len(got.Checks()) != 1 || len(roller.Requests()) != 1 {
		t.Errorf("registry after = %v with %d checks and %d dice, want only the Forcecage, one check and one die", left, len(got.Checks()), len(roller.Requests()))
	}
	if events := got.Conclusion().Events(); len(events) != 2 || !events[1].(combat.OngoingSpellEnded).EndsCasterConcentration() {
		t.Errorf("events = %v, want two endings, the Concentration spell's ending the wizard's Concentration (FR-025)", events)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031, FR-034, FR-036, FR-037 - a goblin who sees a wizard 30 feet away casting a Concentration spell from a slot counters it when the wizard fails the Constitution save against the goblin's DC: the slot is not expended, the prior Concentration ends and the Magic action is wasted (SRD 5.2.1, "Counterspell", p. 120; "Concentration", p. 179).
// Path: happy.
func TestCounterspellOnAConcentrationCast(t *testing.T) {
	t.Parallel()
	wizard := combat.MustCombatantID("wizard")
	cast := must(combat.NewCastObservation("haste", wizard, character.CastSourceSlot, combat.NewUsedComponents(true, true, true), true, content.NewCastingTimeMagicAction()))
	dc := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(must(character.NewLevel(3)))).SaveDC()
	use := must(combat.NewCounterCastSpell(dicetest.NewFixedRoller(4)))
	res, err := use.Run(context.Background(), combat.CounterCastSpellCommand{Cast: cast, Counterspeller: combat.MustCombatantID("goblin"), DistanceFeet: 30,
		SeesCaster: true, SaveDC: dc, CasterSave: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2))}})
	concluded, ok := res.(combat.CounterCastSpellConcluded)
	if err != nil || !ok {
		t.Fatalf("Run = %#v, %v, want CounterCastSpellConcluded", res, err)
	}
	countered, ok := concluded.Result().(combat.CounterspellCountered)
	if !ok || !countered.SlotNotExpended() || !countered.PriorConcentrationEnds() || countered.Wasted() != combat.WastedMagicAction {
		t.Errorf("result = %#v, want countered with the slot kept, prior Concentration ended and the Magic action wasted", concluded.Result())
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-043, FR-044, FR-048, FR-049, FR-051 - a creature inside an Antimagic Field cannot cast, so the host asks first and does not call Counterspell or Dispel Magic; its ordinary spells are suppressed while it is inside and function again when it leaves; a spell from an Artifact or a deity and another Antimagic Field keep functioning (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: unhappy (rule interaction).
func TestSpellsInsideAnAntimagicField(t *testing.T) {
	t.Parallel()
	blocked, err := combat.NewAssessAntimagicActivity().Run(context.Background(), combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell, ActorInside: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := blocked.(combat.AntimagicActivityAssessed).Verdict().(combat.AntimagicBlocks); !ok {
		t.Errorf("verdict = %#v, want AntimagicBlocks", blocked)
	}
	buff := dcSpell(t, "buff", "bless", 1, combat.DispelUnrestricted{}, combat.CombatantID(""))
	field := dcSpell(t, "field", "antimagic_field", 8, combat.DispelNeverDispellable{}, combat.CombatantID(""))
	f := combat.AntimagicField{}
	if f.Status(buff, true) != combat.Suppressed || f.Status(buff, false) != combat.Functioning || f.Status(field, true) != combat.Functioning {
		t.Error("an ordinary spell must be suppressed only inside the aura and another Antimagic Field must keep functioning")
	}
}
