package combat_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-traits.md FR-001, FR-002, FR-003, FR-005, FR-006, FR-007 - one template row per trait and variant is recognised as its trait.
// Path: happy.
func TestRecogniseEachTemplate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, part, text string
		want             combat.MonsterTrait
	}{
		{"Magic Resistance", "Magic Resistance", mtMagic, nil},
		{"Undead Fortitude", "Undead Fortitude", mtUndead, nil},
		{"Regeneration first form", "Regeneration", mtRegenOne, nil},
		{"Regeneration second form", "Regeneration", mtRegenTwo, nil},
		{"Swarm", "Swarm", mtSwarm, nil},
		{"Pack Tactics singular", "Pack Tactics", mtPack, nil},
		{"Pack Tactics plural", "Pack Tactics", strings.Replace(mtPack, "an attack roll", "attack rolls", 1), nil},
		{"Sunlight Sensitivity", "Sunlight Sensitivity", mtSunSens, nil},
		{"Sunlight Weakness", "Sunlight Weakness", mtSunWeak, nil},
		{"Flyby", "Flyby", mtFlyby, nil},
		{"Amorphous", "Amorphous", mtAmorphous, nil},
		{"Siege Monster", "Siege Monster", mtSiege, nil},
		{"two word noun", "Siege Monster", "The shambling mound deals double damage to objects and structures.", nil},
		{"typographic apostrophe", "Flyby", strings.Replace(mtFlyby, "'", "’", 1), nil},
		{"whitespace runs", "Flyby", "The  drake doesn't\nprovoke an Opportunity Attack when it flies  out of an enemy's reach.", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait(c.part, c.text)}, nil))
			if len(r.Traits()) != 1 || len(r.Unrecognised()) != 0 {
				t.Fatalf("traits %v rejections %v", r.Traits(), r.Unrecognised())
			}
			if got := r.Traits()[0].Part(); got != mtRef(t, 0) {
				t.Errorf("Part = %v", got)
			}
		})
	}
}

// Requirement: doc/specs/monster-traits.md FR-001, FR-005, FR-006, FR-007 - Legendary Resistance, Death Burst and Parry are recognised with their usage, saving throw and bonus.
// Path: happy.
func TestRecogniseLegendaryBurstAndParry(t *testing.T) {
	t.Parallel()
	m := mtMonster(t, []content.StatBlockPart{mtTextPart("Legendary Resistance", mtLegendary, luPerDay(3, 4)), mtBurstPart(mtBurst, true)},
		[]content.StatBlockPart{mtTrait("Parry", mtParry)})
	r := combat.RecogniseTraits(m)
	ts := r.Traits()
	if len(ts) != 3 || len(r.Unrecognised()) != 0 {
		t.Fatalf("traits %v rejections %v", ts, r.Unrecognised())
	}
	if _, ok := ts[0].(combat.LegendaryResistanceTrait); !ok {
		t.Errorf("first = %T", ts[0])
	}
	if b, ok := ts[1].(combat.DeathBurstTrait); !ok || b.DC() != 11 || b.Ability().String() != "constitution" {
		t.Errorf("second = %#v", ts[1])
	}
	if p, ok := ts[2].(combat.ParryTrait); !ok || p.Bonus() != 2 || p.Part() != luRef(t, combat.MonsterPartReactions, 0) {
		t.Errorf("third = %#v", ts[2])
	}
}

// Requirement: doc/specs/monster-traits.md FR-007 - Regeneration parameters (amount and the suppressing types) are parsed from the text.
// Path: happy.
func TestRecogniseRegenerationParameters(t *testing.T) {
	t.Parallel()
	two := mtOne[combat.RegenerationTrait](t, combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Regeneration", mtRegenTwo)}, nil)))
	if two.Amount() != 15 || !slices.Equal(two.SuppressingTypes(), []content.DamageType{content.DamageAcid, content.DamageFire}) {
		t.Errorf("second form = %d %v", two.Amount(), two.SuppressingTypes())
	}
	one := mtOne[combat.RegenerationTrait](t, combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Regeneration", mtRegenOne)}, nil)))
	if one.Amount() != 10 || len(one.SuppressingTypes()) != 0 {
		t.Errorf("first form = %d %v", one.Amount(), one.SuppressingTypes())
	}
}

// Requirement: doc/specs/monster-traits.md FR-002, FR-003, FR-004, FR-005, FR-006, FR-007 - a part with a trait name that fails a check is reported with its rejection and never recognised.
// Path: per case.
func TestRecogniseRejections(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		traits, reactions       []content.StatBlockPart
		want                    combat.TraitRejection
	}{
		{"Parry in traits", "FR-004", "unhappy (rule interaction)", []content.StatBlockPart{mtTrait("Parry", mtParry)}, nil, combat.TraitSectionMismatch},
		{"Flyby in reactions", "FR-004", "unhappy (rule interaction)", nil, []content.StatBlockPart{mtTrait("Flyby", mtFlyby)}, combat.TraitSectionMismatch},
		{"Legendary Resistance unlimited", "FR-005", "unhappy (missing content)", []content.StatBlockPart{mtTrait("Legendary Resistance", mtLegendary)}, nil, combat.TraitUsageMismatch},
		{"extra word", "FR-002", "unhappy (invalid input)", []content.StatBlockPart{mtTrait("Flyby", mtFlyby+" Extra.")}, nil, combat.TraitTextMismatch},
		{"changed word", "FR-003", "unhappy (boundary)", []content.StatBlockPart{mtTrait("Flyby", strings.Replace(mtFlyby, "doesn't", "does not", 1))}, nil, combat.TraitTextMismatch},
		{"capitalised noun", "FR-003", "unhappy (boundary)", []content.StatBlockPart{mtTrait("Siege Monster", "The Drake deals double damage to objects and structures.")}, nil, combat.TraitTextMismatch},
		{"noun differs between occurrences", "FR-003", "unhappy (boundary)", []content.StatBlockPart{mtTrait("Pack Tactics", strings.Replace(mtPack, "of the drake's", "of the wolf's", 1))}, nil, combat.TraitTextMismatch},
		{"empty noun", "FR-003", "unhappy (empty and minimal)", []content.StatBlockPart{mtTrait("Siege Monster", "The  deals double damage to objects and structures.")}, nil, combat.TraitTextMismatch},
		{"wrong case name text swap", "FR-002", "unhappy (invalid input)", []content.StatBlockPart{mtTrait("Flyby", mtAmorphous)}, nil, combat.TraitTextMismatch},
		{"Regeneration amount 0", "FR-007", "unhappy (boundary)", []content.StatBlockPart{mtTrait("Regeneration", strings.Replace(mtRegenOne, "10", "0", 1))}, nil, combat.TraitParameterOutOfRange},
		{"Regeneration amount past int", "FR-007", "unhappy (boundary)", []content.StatBlockPart{mtTrait("Regeneration", strings.Replace(mtRegenOne, "10", "99999999999999999999", 1))}, nil, combat.TraitParameterOutOfRange},
		{"Regeneration above content maximum", "FR-007", "unhappy (boundary)", []content.StatBlockPart{mtTrait("Regeneration", strings.Replace(mtRegenOne, "10", "2147483648", 1))}, nil, combat.TraitParameterOutOfRange},
		{"Regeneration unknown type", "FR-007", "unhappy (invalid input)", []content.StatBlockPart{mtTrait("Regeneration", strings.Replace(mtRegenTwo, "Acid", "Sound", 1))}, nil, combat.TraitUnknownDamageType},
		{"Regeneration lower-case type", "FR-007", "unhappy (invalid input)", []content.StatBlockPart{mtTrait("Regeneration", strings.Replace(mtRegenTwo, "Acid", "acid", 1))}, nil, combat.TraitUnknownDamageType},
		{"Parry bonus 0", "FR-007", "unhappy (boundary)", nil, []content.StatBlockPart{mtTrait("Parry", strings.Replace(mtParry, "adds 2", "adds 0", 1))}, combat.TraitParameterOutOfRange},
		{"Death Burst without a save", "FR-006", "unhappy (missing content)", []content.StatBlockPart{mtBurstPart(mtBurst, false)}, nil, combat.TraitNoSavingThrow},
		{"Death Burst text", "FR-006", "unhappy (missing content)", []content.StatBlockPart{mtBurstPart("It explodes.", true)}, nil, combat.TraitTextMismatch},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := combat.RecogniseTraits(mtMonster(t, c.traits, c.reactions))
			rej := r.Unrecognised()
			if len(r.Traits()) != 0 || len(rej) != 1 || rej[0].Reason() != c.want {
				t.Fatalf("traits %v rejections %v, want %v [%s%s]", r.Traits(), rej, c.want, traitsSpec, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-traits.md FR-001, FR-002 - a monster with none of the names, a part with another name and the zero monster give two empty lists; order is printed order.
// Path: unhappy (empty and minimal).
func TestRecogniseEmptyUnrelatedAndOrder(t *testing.T) {
	t.Parallel()
	if r := combat.RecogniseTraits(content.Monster{}); len(r.Traits())+len(r.Unrecognised()) != 0 {
		t.Errorf("zero monster = %v", r)
	}
	if r := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Spider Climb", "The drake can climb."), mtTrait("flyby", mtFlyby)}, nil)); len(r.Traits())+len(r.Unrecognised()) != 0 {
		t.Errorf("unrelated names = %v", r)
	}
	r := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Flyby", "bad"), mtTrait("Amorphous", mtAmorphous), mtTrait("Siege Monster", mtSiege)}, nil))
	if ts := r.Traits(); len(ts) != 2 || ts[0].Part() != mtRef(t, 1) || ts[1].Part() != mtRef(t, 2) {
		t.Errorf("traits = %v", ts)
	}
	if u := r.Unrecognised(); len(u) != 1 || u[0].Part() != mtRef(t, 0) {
		t.Errorf("unrecognised = %v", u)
	}
}

// Requirement: doc/specs/monster-traits.md FR-001 - the legendary, action and bonus action sections are scanned too, so a trait name there is a section mismatch.
// Path: unhappy (rule interaction).
func TestRecogniseScansEverySection(t *testing.T) {
	t.Parallel()
	m := luMonster(t, func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{mtTrait("Flyby", mtFlyby)}
		f.BonusActions = []content.StatBlockPart{mtTrait("Flyby", mtFlyby)}
		f.LegendaryActions = luMust(content.NewLegendaryActions(3, 0, []content.StatBlockPart{mtTrait("Flyby", mtFlyby)}))
	})
	u := combat.RecogniseTraits(m).Unrecognised()
	want := []combat.MonsterPartSection{combat.MonsterPartActions, combat.MonsterPartBonusActions, combat.MonsterPartLegendaryActions}
	if len(u) != 3 {
		t.Fatalf("unrecognised = %v", u)
	}
	for i, s := range want {
		if u[i].Part().Section() != s || u[i].Reason() != combat.TraitSectionMismatch {
			t.Errorf("rejection %d = %v", i, u[i])
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-062 - a monster with Swarm and Regeneration has its Regeneration marked, so it never functions.
// Path: unhappy (rule interaction).
func TestRecogniseMarksRegenerationOfASwarm(t *testing.T) {
	t.Parallel()
	r := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Regeneration", mtRegenOne), mtTrait("Swarm", mtSwarm)}, nil))
	reg, ok := r.Traits()[0].(combat.RegenerationTrait)
	if !ok || !reg.SwarmCannotRegain() {
		t.Errorf("regeneration = %#v", r.Traits()[0])
	}
	lone := mtOne[combat.RegenerationTrait](t, combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Regeneration", mtRegenOne)}, nil)))
	if lone.SwarmCannotRegain() {
		t.Error("a lone Regeneration is marked as a swarm's")
	}
}

// Requirement: doc/specs/monster-traits.md FR-008, FR-091 - results are values: equal for equal input, the lists are copies, and the monster is unchanged.
// Path: unhappy (immutability).
func TestRecogniseImmutability(t *testing.T) {
	t.Parallel()
	m := mtMonster(t, []content.StatBlockPart{mtTrait("Flyby", mtFlyby), mtTrait("Amorphous", "bad")}, nil)
	a, b := combat.RecogniseTraits(m), combat.RecogniseTraits(m)
	if !a.Equal(b) {
		t.Error("equal input gave unequal results")
	}
	ts := a.Traits()
	ts[0] = nil
	us := a.Unrecognised()
	us[0] = combat.UnrecognisedTrait{}
	if !a.Equal(b) || a.Traits()[0] == nil || a.Unrecognised()[0].Reason() == 0 {
		t.Error("mutating a returned slice changed the value")
	}
	if len(m.Traits()) != 2 {
		t.Error("the monster changed")
	}
	if !(combat.RecognisedTraits{}).Equal(combat.RecognisedTraits{}) || a.Equal(combat.RecognisedTraits{}) {
		t.Error("Equal is wrong for the zero value")
	}
}

// Requirement: doc/specs/monster-traits.md FR-002 - the rejection names are the snake_case values and an undefined one is unknown.
// Path: unhappy (invalid input).
func TestTraitRejectionString(t *testing.T) {
	t.Parallel()
	want := []string{"section_mismatch", "text_mismatch", "usage_mismatch", "parameter_out_of_range", "unknown_damage_type", "no_saving_throw"}
	for i, w := range want {
		if got := combat.TraitRejection(i + 1).String(); got != w {
			t.Errorf("TraitRejection(%d) = %q, want %q", i+1, got, w)
		}
	}
	if combat.TraitRejection(0).String() != "unknown" || combat.TraitRejection(7).String() != "unknown" {
		t.Error("an undefined rejection is not unknown")
	}
}

// Requirement: doc/specs/monster-traits.md FR-002, FR-003, FR-007 - a slot that is missing, truncated or holds a character outside its class is a text mismatch (or an unknown Damage Type for a letter word), never a panic; the bounds 1 and the content maximum are accepted.
// Path: per case.
func TestRecogniseSlotEdges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, part, text string
		want             combat.TraitRejection
		recognised       bool
	}{
		{"regeneration amount missing", "Regeneration", "The drake regains many Hit Points at the start of each of its turns if it has at least 1 Hit Point.", combat.TraitTextMismatch, false},
		{"regeneration cut after the number", "Regeneration", "The drake regains 10", combat.TraitTextMismatch, false},
		{"regeneration cut after the noun", "Regeneration", "The drake", combat.TraitTextMismatch, false},
		{"regeneration cut after the type", "Regeneration", "The drake regains 15 Hit Points at the start of each of its turns. If the drake takes Acid or Fire", combat.TraitTextMismatch, false},
		{"type missing", "Regeneration", strings.Replace(mtRegenTwo, "Acid or Fire", "or Fire", 1), combat.TraitTextMismatch, false},
		{"type with a symbol", "Regeneration", strings.Replace(mtRegenTwo, "Acid", "Acid{", 1), combat.TraitTextMismatch, false},
		{"type with a symbol before a", "Regeneration", strings.Replace(mtRegenTwo, "Acid", "`cid", 1), combat.TraitTextMismatch, false},
		{"type with z", "Regeneration", strings.Replace(mtRegenTwo, "Acid", "Fizz", 1), combat.TraitUnknownDamageType, false},
		{"type with Z", "Regeneration", strings.Replace(mtRegenTwo, "Acid", "FIZZ", 1), combat.TraitUnknownDamageType, false},
		{"number with a letter", "Regeneration", strings.Replace(mtRegenOne, "10", "1x0", 1), combat.TraitTextMismatch, false},
		{"amount 1", "Regeneration", strings.Replace(mtRegenOne, "10", "1", 1), 0, true},
		{"amount at the content maximum", "Regeneration", strings.Replace(mtRegenOne, "10", "2147483647", 1), 0, true},
		{"amount with a leading zero", "Regeneration", strings.Replace(mtRegenOne, "10", "010", 1), 0, true},
		{"parry bonus missing", "Parry", strings.Replace(mtParry, "adds 2", "adds some", 1), combat.TraitTextMismatch, false},
		{"parry bonus at the content maximum", "Parry", strings.Replace(mtParry, "adds 2", "adds 2147483647", 1), 0, true},
		{"noun cut off", "Flyby", "The drake", combat.TraitTextMismatch, false},
		{"noun with a digit", "Flyby", "The dr4ke doesn't provoke an Opportunity Attack when it flies out of an enemy's reach.", combat.TraitTextMismatch, false},
		{"noun with a symbol", "Flyby", "The dra~ke doesn't provoke an Opportunity Attack when it flies out of an enemy's reach.", combat.TraitTextMismatch, false},
		{"noun with z and a", "Flyby", "The zazaz doesn't provoke an Opportunity Attack when it flies out of an enemy's reach.", 0, true},
		{"noun with a space then a symbol", "Flyby", "The drake 5 doesn't provoke an Opportunity Attack when it flies out of an enemy's reach.", combat.TraitTextMismatch, false},
	}
	for _, c := range cases {
		reactions := []content.StatBlockPart(nil)
		traits := []content.StatBlockPart{mtTrait(c.part, c.text)}
		if c.part == "Parry" {
			traits, reactions = nil, traits
		}
		r := combat.RecogniseTraits(mtMonster(t, traits, reactions))
		if c.recognised {
			if len(r.Traits()) != 1 {
				t.Errorf("%s: rejections %v", c.name, r.Unrecognised())
			}
			continue
		}
		if u := r.Unrecognised(); len(r.Traits()) != 0 || len(u) != 1 || u[0].Reason() != c.want {
			t.Errorf("%s: traits %v rejections %v, want %v", c.name, r.Traits(), u, c.want)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-008 - Equal compares the traits and the rejections: a difference in either is unequal.
// Path: unhappy (boundary).
func TestRecognisedTraitsEqualComparesBothLists(t *testing.T) {
	t.Parallel()
	good, bad := mtTrait("Flyby", mtFlyby), mtTrait("Amorphous", "bad")
	a := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{good, bad}, nil))
	onlyGood := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{good}, nil))
	if !onlyGood.Equal(onlyGood) || a.Equal(onlyGood) {
		t.Error("Equal is wrong")
	}
	sameTraits := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{good}, []content.StatBlockPart{mtTrait("Flyby", mtFlyby)}))
	sameRejections := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Siege Monster", mtSiege), bad}, nil))
	if a.Equal(sameTraits) || sameTraits.Equal(a) || a.Equal(sameRejections) || sameRejections.Equal(a) {
		t.Error("Equal ignored a difference in one list")
	}
}
