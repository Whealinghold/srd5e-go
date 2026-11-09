package character_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-wizard-evoker.md FR-106 (class-wizard.md), section 8 - the Evoker features held are the SubclassDefinition's FeaturesAtOrBelow: none below level 3, the entries at 3, 6, 10 and 14
// only, NotEvoker for no subclass or another one, NotGained for a feature above the Wizard level or a level outside 1 to 20.
// Path: per case (happy / unhappy: boundary).
func TestEvokerGate(t *testing.T) {
	t.Parallel()
	sub := evkSub(t)
	ids := []content.FeatureID{character.EvocationSavantID, character.PotentCantripID, character.SculptSpellsID, character.EmpoweredEvocationID, character.OverchannelID}
	levelOf := []int{3, 3, 6, 10, 14}
	for _, level := range []int{1, 2, 3, 5, 6, 9, 10, 13, 14, 20} {
		for i, id := range ids {
			want := character.EvokerGateNotGained
			if level >= levelOf[i] {
				want = character.EvokerGateHeld
			}
			if got := (character.EvokerRules{}).Gate(sub, level, id); got != want {
				t.Errorf("Gate(level %d, %s) = %v; want %v", level, id, got, want)
			}
		}
	}
	if got := (character.EvokerRules{}).Gate(content.SubclassDefinition{}, 20, character.OverchannelID); got != character.EvokerGateNotEvoker {
		t.Errorf("no subclass = %v, want NotEvoker", got)
	}
	other := must(content.NewSubclassDefinition("other", "Other", sub.Source(), sub.Features()))
	if got := (character.EvokerRules{}).Gate(other, 20, character.OverchannelID); got != character.EvokerGateNotEvoker {
		t.Errorf("another subclass = %v, want NotEvoker", got)
	}
	for _, level := range []int{0, 21} {
		if got := (character.EvokerRules{}).Gate(sub, level, character.OverchannelID); got != character.EvokerGateNotGained {
			t.Errorf("Gate(level %d) = %v, want NotGained", level, got)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-001, FR-002 - Evocation Savant adds exactly two Evocation Wizard spells of level 1 or 2, none in the book, free, with one SpellsAddedToSpellbook event of source
// evocation_savant; each cause is reported in the order of the type, the gate first, and a gate that is not Held (including the zero gate) never lets it through.
// Path: per case (happy / unhappy: invalid input, boundary, rule interaction).
func TestAddEvocationSavant(t *testing.T) {
	t.Parallel()
	book := wzBook("k1")
	for _, c := range []struct {
		name   string
		gate   character.EvokerGate
		book   character.Spellbook
		spells []character.WizardSpellFacts
		want   character.EvokerSpellbookRejection
	}{
		{"two spells", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 2)}, character.EvokerSpellbookRejection{}},
		{"not the Evoker", character.EvokerGateNotEvoker, book, nil, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotEvoker}},
		{"zero gate", 0, book, nil, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotEvoker}},
		{"level below 3", character.EvokerGateNotGained, book, nil, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookFeatureNotAvailable}},
		{"one spell", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookWrongSpellCount, Allowed: 2}},
		{"three spells", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 1), evkEvoFact("c", 1)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookWrongSpellCount, Allowed: 2}},
		{"repeat", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("a", 1)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookDuplicateSpell, Spell: "a"}},
		{"cantrip", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 0), evkEvoFact("b", 1)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookIsACantrip, Spell: "a"}},
		{"off the list", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 1, wzOffList)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotOnWizardList, Spell: "b"}},
		{"not Evocation", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 1, evkAbj)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotAnEvocationSpell, Spell: "b"}},
		{"not Evocation before level", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 3), evkEvoFact("b", 1, evkAbj)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotAnEvocationSpell, Spell: "b"}},
		{"level 9 not Evocation", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 9, evkAbj)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotAnEvocationSpell, Spell: "b"}},
		{"level 3 spell", character.EvokerGateHeld, book, []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 3)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAboveHighestSpellLevel, Spell: "b", Highest: 2}},
		{"already known", character.EvokerGateHeld, wzBook("a"), []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 1)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAlreadyInSpellbook, Spell: "a"}},
		{"level before known", character.EvokerGateHeld, wzBook("a"), []character.WizardSpellFacts{evkEvoFact("a", 1), evkEvoFact("b", 3)}, character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAboveHighestSpellLevel, Spell: "b", Highest: 2}},
	} {
		got, rej := character.EvokerRules{}.AddEvocationSavant(wzID, c.gate, c.book, c.spells)
		if rej != c.want {
			t.Errorf("%s: rejection = %+v, want %+v", c.name, rej, c.want)
			continue
		}
		if c.want.Kind != 0 {
			if len(got.Spellbook().Spells()) != 0 || len(got.Events()) != 0 {
				t.Errorf("%s: a rejection changed something: %+v", c.name, got)
			}
			continue
		}
		ev, ok := got.Events()[0].(character.SpellsAddedToSpellbook)
		if !slices.Equal(got.Spellbook().Spells(), wzIDs("k1", "a", "b")) || len(got.Events()) != 1 || !ok || ev.Source() != character.SpellbookEvocationSavant ||
			!slices.Equal(ev.Spells(), wzIDs("a", "b")) || ev.CharacterID() != wzID {
			t.Errorf("%s: book %v events %+v", c.name, got.Spellbook().Spells(), got.Events())
		}
		if _, costed := got.Cost(); costed {
			t.Errorf("%s: the free spells have a cost", c.name)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-003, FR-004, section 4.1.1 rows FA-01 to FA-08 - the Evoker adds one Evocation spell of a level it has slots for on a new level of Spell Slots after level 3 (5, 7, ...),
// never at level 3 (reading, section 9); the causes are reported in the order of the type.
// Path: per case (happy: boundary; unhappy: boundary, rule interaction, invalid input).
func TestAddEvokerFree(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label    string
		gate     character.EvokerGate
		level    int
		newLevel bool
		highest  int
		book     character.Spellbook
		spell    character.WizardSpellFacts
		want     character.EvokerSpellbookRejection
	}{
		{"FA-01", character.EvokerGateHeld, 3, true, 2, wzBook(), evkEvoFact("s", 2), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNoNewSpellLevel}},
		{"FA-02", character.EvokerGateHeld, 5, true, 3, wzBook(), evkEvoFact("s", 3), character.EvokerSpellbookRejection{}},
		{"FA-03", character.EvokerGateHeld, 5, true, 3, wzBook(), evkEvoFact("s", 4), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAboveHighestSpellLevel, Spell: "s", Highest: 3}},
		{"FA-04", character.EvokerGateHeld, 6, false, 3, wzBook(), evkEvoFact("s", 3), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNoNewSpellLevel}},
		{"FA-05", character.EvokerGateHeld, 17, true, 9, wzBook(), evkEvoFact("s", 9), character.EvokerSpellbookRejection{}},
		{"FA-06", character.EvokerGateHeld, 5, true, 3, wzBook(), evkEvoFact("s", 3, evkAbj), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotAnEvocationSpell, Spell: "s"}},
		{"FA-07", character.EvokerGateHeld, 7, true, 4, wzBook("s"), evkEvoFact("s", 4), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAlreadyInSpellbook, Spell: "s"}},
		{"FA-08", character.EvokerGateHeld, 5, true, 3, wzBook(), evkEvoFact("s", 0), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookIsACantrip, Spell: "s"}},
		{"off the list", character.EvokerGateHeld, 5, true, 3, wzBook(), evkEvoFact("s", 3, wzOffList), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotOnWizardList, Spell: "s"}},
		{"not the Evoker", character.EvokerGateNotEvoker, 5, true, 3, wzBook(), evkEvoFact("s", 3), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotEvoker}},
		{"zero gate", 0, 5, true, 3, wzBook(), evkEvoFact("s", 3), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotEvoker}},
		{"level below 3", character.EvokerGateNotGained, 2, true, 1, wzBook(), evkEvoFact("s", 1), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookFeatureNotAvailable}},
		{"no new level before the spell", character.EvokerGateHeld, 6, false, 3, wzBook(), evkEvoFact("s", 0, evkAbj), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNoNewSpellLevel}},
	} {
		got, rej := character.EvokerRules{}.AddEvokerFree(wzID, c.gate, c.level, c.newLevel, c.highest, c.book, c.spell)
		if rej != c.want {
			t.Errorf("%s: rejection = %+v, want %+v", c.label, rej, c.want)
			continue
		}
		if c.want.Kind != 0 {
			if len(got.Events()) != 0 || len(got.Spellbook().Spells()) != 0 {
				t.Errorf("%s: a rejection changed something", c.label)
			}
			continue
		}
		ev := got.Events()[0].(character.SpellsAddedToSpellbook)
		if !got.Spellbook().Contains("s") || ev.Source() != character.SpellbookEvokerLevel || !slices.Equal(ev.Spells(), wzIDs("s")) {
			t.Errorf("%s: book %v events %+v", c.label, got.Spellbook().Spells(), got.Events())
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-005, FR-006, section 4.2.1 rows PCa-01 to PCa-12 - a damaging cantrip that misses its attack roll or meets a successful save does half its damage, rounded down; a hit or a failed save is
// accepted unchanged; the causes are reported in the order of the type.
// Path: per case (happy / unhappy: boundary, rule interaction, invalid input).
func TestPotentCantrip(t *testing.T) {
	t.Parallel()
	cantrip := wzFact("fire_bolt", 0, evkEvo, evkDmg)
	for _, c := range []struct {
		label   string
		gate    character.EvokerGate
		spell   character.WizardSpellFacts
		kind    d20.Kind
		outcome d20.Outcome
		damage  int
		want    string // "half N", "unchanged" or "" for a rejection
		reject  character.PotentCantripRejection
	}{
		{"PCa-01", character.EvokerGateHeld, cantrip, d20.AttackRoll, d20.Failure, 8, "half 4", character.PotentCantripRejection{}},
		{"PCa-02", character.EvokerGateHeld, cantrip, d20.SavingThrow, d20.Success, 6, "half 3", character.PotentCantripRejection{}},
		{"PCa-03", character.EvokerGateHeld, cantrip, d20.AttackRoll, d20.Success, 8, "unchanged", character.PotentCantripRejection{}},
		{"PCa-04", character.EvokerGateHeld, cantrip, d20.SavingThrow, d20.Failure, 6, "unchanged", character.PotentCantripRejection{}},
		{"PCa-05", character.EvokerGateHeld, wzFact("c", 0), d20.SavingThrow, d20.Success, 0, "", character.PotentCantripRejection{Kind: character.PotentCantripSpellDealsNoDamage}},
		{"PCa-06", character.EvokerGateNotGained, cantrip, d20.AttackRoll, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripFeatureNotAvailable}},
		{"PCa-07", character.EvokerGateHeld, wzFact("s", 1, evkDmg), d20.AttackRoll, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripNotACantrip}},
		{"PCa-08", character.EvokerGateNotEvoker, cantrip, d20.AttackRoll, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripNotEvoker}},
		{"PCa-09", character.EvokerGateHeld, cantrip, d20.AttackRoll, d20.Failure, 7, "half 3", character.PotentCantripRejection{}},
		{"PCa-10", character.EvokerGateHeld, cantrip, d20.SavingThrow, d20.Success, 7, "half 3", character.PotentCantripRejection{}},
		{"PCa-11", character.EvokerGateHeld, cantrip, d20.AttackRoll, d20.Failure, 1, "half 0", character.PotentCantripRejection{}},
		{"PCa-12", character.EvokerGateHeld, cantrip, d20.AbilityCheck, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripUnsupportedTestKind, TestKind: d20.AbilityCheck}},
		{"zero gate", 0, cantrip, d20.AttackRoll, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripNotEvoker}},
		{"check before cantrip", character.EvokerGateHeld, wzFact("s", 1), d20.AbilityCheck, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripUnsupportedTestKind, TestKind: d20.AbilityCheck}},
		{"cantrip before damage", character.EvokerGateHeld, wzFact("s", 1), d20.AttackRoll, d20.Failure, 8, "", character.PotentCantripRejection{Kind: character.PotentCantripNotACantrip}},
		{"a hit on a save fails to be a miss", character.EvokerGateHeld, cantrip, d20.AttackRoll, d20.Success, 7, "unchanged", character.PotentCantripRejection{}},
		{"a failed save is not a miss", character.EvokerGateHeld, cantrip, d20.SavingThrow, d20.Failure, 7, "unchanged", character.PotentCantripRejection{}},
	} {
		got, rej := character.EvokerRules{}.PotentCantrip(c.gate, c.spell, c.kind, c.outcome, c.damage)
		desc := ""
		switch o := got.(type) {
		case character.PotentCantripHalfDamage:
			desc = fmt.Sprintf("half %d", o.Damage())
		case character.PotentCantripUnchanged:
			desc = "unchanged"
		}
		if desc != c.want || rej != c.reject {
			t.Errorf("%s: got %q, %+v; want %q, %+v", c.label, desc, rej, c.want, c.reject)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-007, FR-008, FR-015, section 4.3.1 rows SC-01 to SC-11 - Sculpt Spells protects at most 1 plus the cast level of the visible, affected creatures other than the caster
// (an empty choice is accepted); the causes are reported in the order of the type, the caster before an ineligible creature and the first offender in command order.
// Path: per case (happy: boundary; unhappy: boundary, rule interaction, invalid input).
func TestSculptSpells(t *testing.T) {
	t.Parallel()
	caster := func(cs []character.SculptCandidate) []character.SculptCandidate {
		cs[0].IsCaster = true
		return cs
	}
	hidden := func(cs []character.SculptCandidate) []character.SculptCandidate {
		cs[0].Visible = false
		return cs
	}
	unaffected := func(cs []character.SculptCandidate) []character.SculptCandidate {
		cs[1].Affected = false
		return cs
	}
	for _, c := range []struct {
		label  string
		gate   character.EvokerGate
		spell  character.WizardSpellFacts
		level  int
		chosen []character.SculptCandidate
		want   character.SculptRejection
	}{
		{"SC-01", character.EvokerGateHeld, evkEvoFact("s", 1), 1, evkCreatures(2), character.SculptRejection{}},
		{"SC-02", character.EvokerGateHeld, evkEvoFact("s", 1), 1, evkCreatures(3), character.SculptRejection{Kind: character.SculptTooManyCreatures, Allowed: 2}},
		{"SC-03", character.EvokerGateHeld, evkEvoFact("s", 5), 5, evkCreatures(6), character.SculptRejection{}},
		{"SC-04", character.EvokerGateHeld, evkEvoFact("s", 9), 9, evkCreatures(10), character.SculptRejection{}},
		{"SC-05", character.EvokerGateHeld, evkEvoFact("s", 0), 0, evkCreatures(1), character.SculptRejection{}},
		{"SC-06", character.EvokerGateHeld, evkEvoFact("s", 3), 3, nil, character.SculptRejection{}},
		{"SC-07", character.EvokerGateHeld, evkEvoFact("s", 3, evkAbj), 3, evkCreatures(1), character.SculptRejection{Kind: character.SculptNotAnEvocationSpell}},
		{"SC-08", character.EvokerGateNotGained, evkEvoFact("s", 3), 3, evkCreatures(1), character.SculptRejection{Kind: character.SculptFeatureNotAvailable}},
		{"SC-09", character.EvokerGateHeld, evkEvoFact("s", 3), 3, caster(evkCreatures(1)), character.SculptRejection{Kind: character.SculptCaster, Creature: character.MustIdentifier("c1")}},
		{"SC-10", character.EvokerGateHeld, evkEvoFact("s", 3), 3, hidden(evkCreatures(1)), character.SculptRejection{Kind: character.SculptCreatureNotEligible, Creature: character.MustIdentifier("c1")}},
		{"SC-11", character.EvokerGateHeld, evkEvoFact("s", 1), 3, evkCreatures(4), character.SculptRejection{}},
		{"not affected", character.EvokerGateHeld, evkEvoFact("s", 3), 3, unaffected(evkCreatures(2)), character.SculptRejection{Kind: character.SculptCreatureNotEligible, Creature: character.MustIdentifier("c2")}},
		{"not the Evoker", character.EvokerGateNotEvoker, evkEvoFact("s", 3), 3, nil, character.SculptRejection{Kind: character.SculptNotEvoker}},
		{"zero gate", 0, evkEvoFact("s", 3), 3, nil, character.SculptRejection{Kind: character.SculptNotEvoker}},
		{"caster before ineligible", character.EvokerGateHeld, evkEvoFact("s", 3), 3, []character.SculptCandidate{{ID: character.MustIdentifier("c1")}, {ID: character.MustIdentifier("c2"), Visible: true, Affected: true, IsCaster: true}}, character.SculptRejection{Kind: character.SculptCaster, Creature: character.MustIdentifier("c2")}},
		{"ineligible before too many", character.EvokerGateHeld, evkEvoFact("s", 0), 0, hidden(evkCreatures(3)), character.SculptRejection{Kind: character.SculptCreatureNotEligible, Creature: character.MustIdentifier("c1")}},
		{"school before creature", character.EvokerGateHeld, evkEvoFact("s", 3, evkAbj), 3, caster(evkCreatures(1)), character.SculptRejection{Kind: character.SculptNotAnEvocationSpell}},
	} {
		got, rej := character.EvokerRules{}.SculptSpells(c.gate, c.spell, c.level, c.chosen)
		if rej != c.want {
			t.Errorf("%s: rejection = %+v, want %+v", c.label, rej, c.want)
			continue
		}
		if c.want.Kind == 0 && (len(got.Creatures()) != len(c.chosen) || got.Allowed() != 1+c.level) {
			t.Errorf("%s: protected %v of %d, allowed %d", c.label, got.Creatures(), len(c.chosen), got.Allowed())
		}
		if c.want.Kind != 0 && (len(got.Creatures()) != 0 || got.Allowed() != 0) {
			t.Errorf("%s: a rejection protected creatures", c.label)
		}
	}
	got, _ := character.EvokerRules{}.SculptSpells(character.EvokerGateHeld, evkEvoFact("s", 1), 1, evkCreatures(2))
	got.Creatures()[0] = character.MustIdentifier("mutated")
	if got.Creatures()[0] != character.MustIdentifier("c1") {
		t.Error("Creatures returned the stored slice")
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-009, FR-010, FR-016, section 4.4.1 rows EE-01 to EE-08 - Empowered Evocation adds the Intelligence modifier (a negative one as it is) to one damage roll of a Wizard Evocation spell and marks
// the cast; choosing not to apply it returns no modifier and the state unchanged; a second empowered roll is AlreadyEmpowered; the causes are reported in the order of the type.
// Path: per case (happy: boundary; unhappy: rule interaction, boundary).
func TestEmpoweredEvocation(t *testing.T) {
	t.Parallel()
	spell := evkEvoFact("s", 3)
	for _, c := range []struct {
		label   string
		gate    character.EvokerGate
		spell   character.WizardSpellFacts
		score   int
		applies bool
		state   character.EmpoweredCastState
		mod     int
		applied bool
		after   bool
		want    character.EmpoweredRejection
	}{
		{"EE-01", character.EvokerGateHeld, spell, 16, true, character.EmpoweredCastState{}, 3, true, true, character.EmpoweredRejection{}},
		{"EE-02", character.EvokerGateHeld, spell, 20, true, character.EmpoweredCastState{}, 5, true, true, character.EmpoweredRejection{}},
		{"EE-03", character.EvokerGateHeld, spell, 10, true, character.EmpoweredCastState{}, 0, true, true, character.EmpoweredRejection{}},
		{"EE-04", character.EvokerGateHeld, spell, 8, true, character.EmpoweredCastState{}, -1, true, true, character.EmpoweredRejection{}},
		{"EE-05", character.EvokerGateHeld, spell, 16, true, character.NewEmpoweredCastState(true), 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredAlreadyEmpowered}},
		{"EE-06", character.EvokerGateHeld, evkEvoFact("s", 3, evkAbj), 16, true, character.EmpoweredCastState{}, 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredNotAnEvocationSpell}},
		{"EE-07", character.EvokerGateNotGained, spell, 16, true, character.EmpoweredCastState{}, 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredFeatureNotAvailable}},
		{"EE-08", character.EvokerGateHeld, spell, 16, false, character.EmpoweredCastState{}, 0, false, false, character.EmpoweredRejection{}},
		{"declined on an empowered cast", character.EvokerGateHeld, spell, 16, false, character.NewEmpoweredCastState(true), 0, false, true, character.EmpoweredRejection{}},
		{"not the Evoker", character.EvokerGateNotEvoker, spell, 16, true, character.EmpoweredCastState{}, 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredNotEvoker}},
		{"zero gate", 0, spell, 16, true, character.EmpoweredCastState{}, 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredNotEvoker}},
		{"not a Wizard spell", character.EvokerGateHeld, evkEvoFact("s", 3, wzOffList), 16, true, character.EmpoweredCastState{}, 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredNotAWizardSpell}},
		{"Wizard spell before school", character.EvokerGateHeld, evkEvoFact("s", 3, wzOffList, evkAbj), 16, true, character.NewEmpoweredCastState(true), 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredNotAWizardSpell}},
		{"school before already", character.EvokerGateHeld, evkEvoFact("s", 3, evkAbj), 16, true, character.NewEmpoweredCastState(true), 0, false, false, character.EmpoweredRejection{Kind: character.EmpoweredNotAnEvocationSpell}},
	} {
		got, rej := character.EvokerRules{}.EmpoweredEvocation(c.gate, c.spell, evkMod(c.score), c.applies, c.state)
		mod, applied := got.Modifier()
		if rej != c.want || applied != c.applied || (applied && mod.Value() != c.mod) || got.State().DamageRollEmpowered() != c.after {
			t.Errorf("%s: got %+v %v %v %+v; want mod %d applied %v after %v and %+v", c.label, mod, applied, got.State(), rej, c.mod, c.applied, c.after, c.want)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-011 to FR-014, section 4.5.1 rows OC-01 to OC-12 - Overchannel is a use per Spell Slot of level 1 to 5 for a damaging Wizard spell; the first use since a Long Rest
// has no Necrotic damage, the n-th use n d12 per slot level; the causes are reported in the order of the type and a rejection is not a use.
// Path: per case (happy: boundary; unhappy: boundary, rule interaction).
func TestOverchannelPlan(t *testing.T) {
	t.Parallel()
	req := func(slot int, source character.CastSource, mods ...func(*character.WizardSpellFacts)) character.OverchannelRequest {
		return character.OverchannelRequest{Spell: wzFact("s", max(slot, 1), append([]func(*character.WizardSpellFacts){evkDmg}, mods...)...), Source: source, SlotLevel: slot}
	}
	slot := character.CastSourceSlot
	for _, c := range []struct {
		label string
		gate  character.EvokerGate
		req   character.OverchannelRequest
		uses  int
		use   int
		dice  int
		want  character.OverchannelRejection
	}{
		{"OC-01", character.EvokerGateHeld, req(3, slot), 0, 1, 0, character.OverchannelRejection{}},
		{"OC-02", character.EvokerGateHeld, req(3, slot), 1, 2, 6, character.OverchannelRejection{}},
		{"OC-03", character.EvokerGateHeld, req(1, slot), 2, 3, 3, character.OverchannelRejection{}},
		{"OC-04", character.EvokerGateHeld, req(5, slot), 1, 2, 10, character.OverchannelRejection{}},
		{"OC-05", character.EvokerGateHeld, req(2, slot), 3, 4, 8, character.OverchannelRejection{}},
		{"OC-06", character.EvokerGateHeld, req(3, slot), 0, 1, 0, character.OverchannelRejection{}},
		{"OC-07", character.EvokerGateHeld, req(3, slot), 1, 2, 6, character.OverchannelRejection{}},
		{"OC-08", character.EvokerGateHeld, req(6, slot), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelSlotLevelOutOfRange, Level: 6}},
		{"OC-09", character.EvokerGateHeld, character.OverchannelRequest{Spell: wzFact("s", 3), Source: slot, SlotLevel: 3}, 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelSpellDealsNoDamage}},
		{"OC-10", character.EvokerGateHeld, req(0, character.CastSourceCantrip), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNoSpellSlotUsed}},
		{"OC-11", character.EvokerGateNotGained, req(3, slot), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelFeatureNotAvailable}},
		{"OC-12", character.EvokerGateHeld, req(3, slot, wzOffList), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNotAWizardSpell}},
		{"slot level 0", character.EvokerGateHeld, req(0, slot), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelSlotLevelOutOfRange, Level: 0}},
		{"a Ritual", character.EvokerGateHeld, req(3, character.CastSourceRitual), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNoSpellSlotUsed}},
		{"a free cast", character.EvokerGateHeld, req(3, character.CastSourceFeatureGranted), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNoSpellSlotUsed}},
		{"not the Evoker", character.EvokerGateNotEvoker, req(3, slot), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNotEvoker}},
		{"zero gate", 0, req(3, slot), 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNotEvoker}},
		{"slot before damage", character.EvokerGateHeld, character.OverchannelRequest{Spell: wzFact("s", 3), Source: slot, SlotLevel: 6}, 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelSlotLevelOutOfRange, Level: 6}},
		{"source before slot", character.EvokerGateHeld, character.OverchannelRequest{Spell: wzFact("s", 3, evkDmg), Source: character.CastSourceCantrip, SlotLevel: 9}, 0, 0, 0, character.OverchannelRejection{Kind: character.OverchannelNoSpellSlotUsed}},
	} {
		plan, rej := character.EvokerRules{}.Overchannel(c.gate, c.req, c.uses)
		if rej != c.want || plan.Use() != c.use || plan.NecroticDice() != c.dice {
			t.Errorf("%s: use %d dice %d %+v; want use %d dice %d %+v", c.label, plan.Use(), plan.NecroticDice(), rej, c.use, c.dice, c.want)
		}
		expr, rolled, err := plan.Necrotic()
		if err != nil || rolled != (c.dice > 0) || (rolled && (expr.Count() != c.dice || expr.Die().Sides() != 12 || expr.Modifier() != 0)) {
			t.Errorf("%s: Necrotic = %v %v %v", c.label, expr, rolled, err)
		}
	}
	plan, _ := character.EvokerRules{}.Overchannel(character.EvokerGateHeld, req(5, slot), 300)
	if _, _, err := plan.Necrotic(); !errorsIs(err, srderr.ErrInvalidInput) {
		t.Errorf("1505 dice error = %v, want ErrInvalidInput", err)
	}
}
