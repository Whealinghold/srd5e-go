package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func wzMastery(t *testing.T) character.SpellMastery {
	t.Helper()
	m, rej := character.NewSpellMastery([]character.WizardSpellFacts{wzFact("two", 2), wzFact("one", 1)}, wzBook("one", "two"))
	if rej.Kind != 0 {
		t.Fatalf("NewSpellMastery: %+v", rej)
	}
	return m
}

func wzSignature(t *testing.T) character.SignatureSpells {
	t.Helper()
	s, rej := character.NewSignatureSpells([]character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 3)}, wzBook("s1", "s2"))
	if rej.Kind != 0 {
		t.Fatalf("NewSignatureSpells: %+v", rej)
	}
	return s
}

// Requirement: doc/specs/class-wizard.md FR-107, FR-050, FR-074, FR-090 - WizardResources starts with nothing used or chosen; AfterRest(ShortRest) makes both Signature Spells free
// again but leaves Arcane Recovery used and the Overchannel count; AfterRest(LongRest) also makes Arcane Recovery available and resets Overchannel; any other kind changes nothing.
// Path: per case (happy / unhappy: state error).
func TestWizardResourcesAfterRest(t *testing.T) {
	t.Parallel()
	var zero character.WizardResources
	if zero.ArcaneRecoveryUsed() || zero.OverchannelUses() != 0 {
		t.Error("the zero state has something used")
	}
	if _, ok := zero.Scholar(); ok {
		t.Error("the zero state has a Scholar skill")
	}
	if _, ok := zero.SpellMastery(); ok {
		t.Error("the zero state has Spell Mastery")
	}
	if _, ok := zero.SignatureSpells(); ok {
		t.Error("the zero state has Signature Spells")
	}
	signature := wzSignature(t)
	used, err := zero.WithArcaneRecoveryUsed().WithOverchannelUses(2)
	if err != nil {
		t.Fatal(err)
	}
	// Signature Spells are held by the use case; build the state through it in the application tests, here through RecordSignatureSpellCast on a state that holds them.
	held := wzHolding(t, used, signature)
	spent, err := held.RecordSignatureSpellCast("s1")
	if err != nil || !spent.SignatureSpellSpent("s1") || spent.SignatureSpellSpent("s2") || held.SignatureSpellSpent("s1") {
		t.Fatalf("RecordSignatureSpellCast = %+v, %v", spent, err)
	}
	short := spent.AfterRest(character.ShortRest)
	if short.SignatureSpellSpent("s1") || !short.ArcaneRecoveryUsed() || short.OverchannelUses() != 2 {
		t.Errorf("after a Short Rest: %+v", short)
	}
	long := spent.AfterRest(character.LongRest)
	if long.SignatureSpellSpent("s1") || long.ArcaneRecoveryUsed() || long.OverchannelUses() != 0 {
		t.Errorf("after a Long Rest: %+v", long)
	}
	if other := spent.AfterRest(character.RestKind(0)); other != spent {
		t.Errorf("an unknown rest kind changed the state: %+v", other)
	}
	if _, err := spent.RecordSignatureSpellCast("nope"); !srderrIsInvalid(err) {
		t.Errorf("RecordSignatureSpellCast(not a Signature Spell) err = %v", err)
	}
	if _, err := zero.WithOverchannelUses(-1); !srderrIsInvalid(err) {
		t.Errorf("WithOverchannelUses(-1) err = %v", err)
	}
	if !zero.Equal(character.WizardResources{}) || zero.Equal(used) || zero.Equal(character.FighterState{}) {
		t.Error("Equal is wrong")
	}
}

// wzHolding returns r holding the Signature Spells, built through the ChooseSignatureSpells use case so the state's invariants are the production ones.
func wzHolding(t *testing.T, r character.WizardResources, s character.SignatureSpells) character.WizardResources {
	t.Helper()
	uc := must(character.NewChooseSignatureSpells(wzSource(t)))
	res, err := uc.Run(t.Context(), character.ChooseSignatureSpellsCommand{CharacterID: wzID, ClassLevels: wzLevels(20), Resources: r, Spellbook: wzBook("s1", "s2"),
		Spells: []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 3)}})
	applied, ok := res.(character.SignatureSpellsApplied)
	if err != nil || !ok || applied.Signature() != s {
		t.Fatalf("ChooseSignatureSpells = %#v, %v", res, err)
	}
	return applied.Resources()
}

// Requirement: doc/specs/class-wizard.md FR-065, FR-066, FR-068 - Spell Mastery takes one level 1 and one level 2 spell from the spellbook, each with the Magic action; it is
// rejected by cause in order (not in the book, repeat, levels, casting time); a Mastery spell is castable without a slot at its lowest level only, and unlimited.
// Path: per case (happy / unhappy: invalid input).
func TestNewSpellMastery(t *testing.T) {
	t.Parallel()
	book := wzBook("one", "two", "three", "other1")
	type rej = character.SpellMasteryRejection
	for _, c := range []struct {
		name   string
		spells []character.WizardSpellFacts
		want   rej
	}{
		{"level 1 and 2", []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2)}, rej{}},
		{"level 2 first", []character.WizardSpellFacts{wzFact("two", 2), wzFact("one", 1)}, rej{}},
		{"not in book", []character.WizardSpellFacts{wzFact("one", 1), wzFact("zz", 2)}, rej{Kind: character.SpellMasteryNotInSpellbook, Spell: "zz"}},
		{"same spell twice", []character.WizardSpellFacts{wzFact("one", 1), wzFact("one", 1)}, rej{Kind: character.SpellMasteryDuplicateSpell, Spell: "one"}},
		{"two level 1", []character.WizardSpellFacts{wzFact("one", 1), wzFact("other1", 1)}, rej{Kind: character.SpellMasteryWrongSpellLevels, Levels: []int{1, 1}}},
		{"level 3", []character.WizardSpellFacts{wzFact("one", 1), wzFact("three", 3)}, rej{Kind: character.SpellMasteryWrongSpellLevels, Levels: []int{1, 3}}},
		{"one spell", []character.WizardSpellFacts{wzFact("one", 1)}, rej{Kind: character.SpellMasteryWrongSpellLevels, Levels: []int{1}}},
		{"three spells", []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2), wzFact("other1", 1)}, rej{Kind: character.SpellMasteryWrongSpellLevels, Levels: []int{1, 2, 1}}},
		{"not an action", []character.WizardSpellFacts{wzFact("one", 1, wzMinutes), wzFact("two", 2)}, rej{Kind: character.SpellMasteryCastingTimeNotAction, Spell: "one"}},
	} {
		got, r := character.NewSpellMastery(c.spells, book)
		if r.Kind != c.want.Kind || r.Spell != c.want.Spell || !slices.Equal(r.Levels, c.want.Levels) {
			t.Errorf("%s: rejection = %+v, want %+v", c.name, r, c.want)
		}
		if (c.want.Kind == 0) == got.IsZero() {
			t.Errorf("%s: mastery %+v", c.name, got)
		}
	}
	m := wzMastery(t)
	if m.LevelOne() != "one" || m.LevelTwo() != "two" || !slices.Equal(m.Spells(), wzIDs("one", "two")) || len((character.SpellMastery{}).Spells()) != 0 {
		t.Errorf("accessors: %+v", m)
	}
	for _, c := range []struct {
		spell content.SpellID
		level int
		want  character.WizardCastRequirement
	}{{"one", 1, character.WizardCastWithoutSlot}, {"two", 2, character.WizardCastWithoutSlot}, {"one", 2, character.WizardCastNeedsSlot},
		{"two", 3, character.WizardCastNeedsSlot}, {"two", 1, character.WizardCastNeedsSlot}, {"other", 1, character.WizardCastNeedsSlot}} {
		if got := m.CastRequirement(c.spell, c.level); got != c.want {
			t.Errorf("CastRequirement(%s, %d) = %v, want %v", c.spell, c.level, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-wizard.md FR-071 to FR-074 - Signature Spells take exactly two distinct level 3 spells from the spellbook (rejected by cause in order); each is castable once at
// level 3 without a slot, the other stays free, a second cast needs a slot, a higher-level cast needs a slot and does not use up the free cast, and a rest makes both free again.
// Path: per case (happy / unhappy: state error).
func TestNewSignatureSpellsAndCasts(t *testing.T) {
	t.Parallel()
	book := wzBook("s1", "s2", "s3")
	type rej = character.SignatureSpellsRejection
	for _, c := range []struct {
		name   string
		spells []character.WizardSpellFacts
		want   rej
	}{
		{"two level 3", []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 3)}, rej{}},
		{"one", []character.WizardSpellFacts{wzFact("s1", 3)}, rej{Kind: character.SignatureSpellsWrongSpellCount, Allowed: 2}},
		{"three", []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 3), wzFact("s3", 3)}, rej{Kind: character.SignatureSpellsWrongSpellCount, Allowed: 2}},
		{"twice", []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s1", 3)}, rej{Kind: character.SignatureSpellsDuplicateSpell, Spell: "s1"}},
		{"not level 3", []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 2)}, rej{Kind: character.SignatureSpellsNotLevelThree, Spell: "s2"}},
		{"above level 3", []character.WizardSpellFacts{wzFact("s1", 3), wzFact("s2", 4)}, rej{Kind: character.SignatureSpellsNotLevelThree, Spell: "s2"}},
		{"not in book", []character.WizardSpellFacts{wzFact("s1", 3), wzFact("zz", 3)}, rej{Kind: character.SignatureSpellsNotInSpellbook, Spell: "zz"}},
	} {
		if got, r := character.NewSignatureSpells(c.spells, book); r != c.want || (c.want.Kind == 0) == got.IsZero() {
			t.Errorf("%s: %+v, %+v; want %+v", c.name, got, r, c.want)
		}
	}
	sig := wzSignature(t)
	if !slices.Equal(sig.Spells(), wzIDs("s1", "s2")) || len((character.SignatureSpells{}).Spells()) != 0 {
		t.Errorf("Spells = %v", sig.Spells())
	}
	r := wzHolding(t, character.WizardResources{}, sig)
	if sig.CastRequirement("s1", 3, r) != character.WizardCastWithoutSlot || sig.CastRequirement("s2", 3, r) != character.WizardCastWithoutSlot {
		t.Error("a fresh Signature Spell needs a slot")
	}
	r1 := must(r.RecordSignatureSpellCast("s1"))
	if sig.CastRequirement("s1", 3, r1) != character.WizardCastNeedsSlot || sig.CastRequirement("s2", 3, r1) != character.WizardCastWithoutSlot {
		t.Error("the second cast of s1 must need a slot while s2 stays free")
	}
	if sig.CastRequirement("s2", 2, r) != character.WizardCastNeedsSlot {
		t.Error("a lower-level cast needs a slot")
	}
	if sig.CastRequirement("s2", 4, r) != character.WizardCastNeedsSlot || sig.CastRequirement("s2", 3, r) != character.WizardCastWithoutSlot {
		t.Error("a higher-level cast needs a slot and leaves the free cast")
	}
	if sig.CastRequirement("s1", 3, r1.AfterRest(character.ShortRest)) != character.WizardCastWithoutSlot || sig.CastRequirement("zz", 3, r) != character.WizardCastNeedsSlot {
		t.Error("a rest makes the free cast available; a non-Signature spell needs a slot")
	}
}

// Requirement: doc/specs/class-wizard.md FR-070 - WizardAlwaysPrepared.Of lists the Spell Mastery spells, then the Signature Spells, and nothing when neither is held.
// Path: per case (happy / unhappy: empty and minimal).
func TestWizardAlwaysPrepared(t *testing.T) {
	t.Parallel()
	if got := (character.WizardAlwaysPrepared{}).Of(character.WizardResources{}); len(got) != 0 {
		t.Errorf("Of(zero) = %v", got)
	}
	// Both choices are held by running the use cases.
	r := wzHolding(t, character.WizardResources{}, wzSignature(t))
	masteryUC := must(character.NewChooseSpellMastery(wzSource(t)))
	res, err := masteryUC.Run(t.Context(), character.ChooseSpellMasteryCommand{CharacterID: wzID, ClassLevels: wzLevels(20), Resources: r, Spellbook: wzBook("one", "two"),
		Spells: []character.WizardSpellFacts{wzFact("one", 1), wzFact("two", 2)}})
	applied, ok := res.(character.SpellMasteryApplied)
	if err != nil || !ok {
		t.Fatalf("ChooseSpellMastery = %#v, %v", res, err)
	}
	if got := (character.WizardAlwaysPrepared{}).Of(applied.Resources()); !slices.Equal(got, wzIDs("one", "two", "s1", "s2")) {
		t.Errorf("Of = %v", got)
	}
}
