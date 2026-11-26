package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func mscCasting(tb testing.TB, spell string) character.CastingInProgress {
	tb.Helper()
	return luMust(character.NewCastingInProgress(content.SpellID(spell), character.CastSourceFeatureGranted, 0, content.CastingTimeMode{}, nil, 10))
}

func mscConcentration2(tb testing.TB, subject character.ConcentrationSubject, kind character.ConcentrationKind) character.Concentration {
	tb.Helper()
	return luMust(character.NewConcentration(subject, kind))
}

// Requirement: doc/specs/monster-spellcasting.md FR-032 - the state holds a combatant and, optionally, Concentration and a casting in progress with the reference of the entry that began it; it compares by value with ==.
// Path: happy.
func TestNewMonsterSpellcastingState(t *testing.T) {
	t.Parallel()
	bare := luMust(combat.NewMonsterSpellcastingState(mscID()))
	if _, held := bare.Concentration(); held || bare.IsZero() || !slices.Equal(bare.Combatants(), []combat.CombatantID{mscID()}) {
		t.Errorf("a bare state = %+v", bare)
	}
	if _, _, going := bare.CastingInProgress(); going {
		t.Errorf("a bare state holds a casting")
	}
	subject := character.NewConcentrationSubjectSpell("scrying")
	conc := mscConcentration2(t, subject, character.ConcentrationCasting)
	ref := mscRef(t, mscScryingToo)
	full := luMust(combat.NewMonsterSpellcastingState(mscID(), combat.WithMonsterConcentration(conc), combat.WithMonsterCasting(mscCasting(t, "scrying"), ref)))
	gotConc, held := full.Concentration()
	casting, gotRef, going := full.CastingInProgress()
	if !held || gotConc != conc || !going || casting.Spell() != "scrying" || gotRef != ref {
		t.Errorf("a full state = %v %v, %v %v %v", gotConc, held, casting.Spell(), gotRef, going)
	}
	again := luMust(combat.NewMonsterSpellcastingState(mscID(), combat.WithMonsterConcentration(conc), combat.WithMonsterCasting(mscCasting(t, "scrying"), ref)))
	if full != again || full == bare {
		t.Errorf("states compare by value: equal pair %v, different pair equal %v", full == again, full == bare)
	}
	effect := luMust(combat.NewMonsterSpellcastingState(mscID(), combat.WithMonsterConcentration(mscConcentration2(t, character.NewConcentrationSubjectFeature("rage"), character.ConcentrationEffect))))
	if c, _ := effect.Concentration(); c.Kind() != character.ConcentrationEffect {
		t.Errorf("Concentration on a feature = %v", c.Kind())
	}
	noCasting := luMust(combat.NewMonsterSpellcastingState(mscID(), combat.WithMonsterCasting(character.CastingInProgress{}, ref)))
	if noCasting != bare {
		t.Errorf("a reference with no casting is kept: %+v", noCasting)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-032 - a zero combatant, and a casting in progress without Concentration of kind casting on the same spell (none, of kind effect, on another spell, on a feature), are errors matching srderr.ErrInvalidInput.
// Path: per case.
func TestNewMonsterSpellcastingStateRefusals(t *testing.T) {
	t.Parallel()
	spell := func(id string) character.ConcentrationSubject {
		return character.NewConcentrationSubjectSpell(content.SpellID(id))
	}
	cases := []struct {
		name, requirement, path string
		combatant               combat.CombatantID
		opts                    []combat.MonsterSpellcastingStateOption
	}{
		{"zero combatant", "FR-032", "unhappy (invalid input)", "", nil},
		{"casting without Concentration", "FR-032", "unhappy (invalid input)", mscID(),
			[]combat.MonsterSpellcastingStateOption{combat.WithMonsterCasting(mscCasting(t, "scrying"), combat.SpellUseRef{})}},
		{"Concentration of kind effect", "FR-032", "unhappy (invalid input)", mscID(), []combat.MonsterSpellcastingStateOption{
			combat.WithMonsterConcentration(mscConcentration2(t, spell("scrying"), character.ConcentrationEffect)), combat.WithMonsterCasting(mscCasting(t, "scrying"), combat.SpellUseRef{})}},
		{"Concentration on a spell that sorts after it", "FR-032", "unhappy (invalid input)", mscID(), []combat.MonsterSpellcastingStateOption{
			combat.WithMonsterConcentration(mscConcentration2(t, spell("zebra"), character.ConcentrationCasting)), combat.WithMonsterCasting(mscCasting(t, "scrying"), combat.SpellUseRef{})}},
		{"Concentration on another spell", "FR-032", "unhappy (invalid input)", mscID(), []combat.MonsterSpellcastingStateOption{
			combat.WithMonsterConcentration(mscConcentration2(t, spell("geas"), character.ConcentrationCasting)), combat.WithMonsterCasting(mscCasting(t, "scrying"), combat.SpellUseRef{})}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewMonsterSpellcastingState(c.combatant, c.opts...)
			if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
				t.Errorf("NewMonsterSpellcastingState() = %v, %v, want the zero state and ErrInvalidInput [%s]", got, err, monsterSpellcastingSpec+c.requirement)
			}
		})
	}
}
