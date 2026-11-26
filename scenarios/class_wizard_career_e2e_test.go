package scenarios_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The Wizard career scenarios below drive the Wizard use cases through the public packages only, with the
// bundled SRD Wizard class and spells (content/data). Spell facts come from the bundled spell entries, so the
// levels, schools, Ritual tags and casting times are the content's own. Specs: class-wizard.md (SRD 5.2.1,
// "Wizard", pp. 77 to 79).

// grpD1Facts returns the Wizard facts of a bundled spell.
func grpD1Facts(t *testing.T, src content.Source, id string, dealsDamage bool) character.WizardSpellFacts {
	t.Helper()
	found, err := src.Lookup(context.Background(), content.NewKey(content.KindSpell, content.MustID(id)))
	if err != nil {
		t.Fatalf("Lookup(%s) = %v", id, err)
	}
	return character.NewWizardSpellFacts(found.(content.LookupFound).Entry.(content.Spell), dealsDamage)
}

func grpD1Facts2(t *testing.T, src content.Source, ids ...string) []character.WizardSpellFacts {
	t.Helper()
	out := make([]character.WizardSpellFacts, len(ids))
	for i, id := range ids {
		out[i] = grpD1Facts(t, src, id, false)
	}
	return out
}

func grpD1Wizard(level int) character.ClassLevels {
	return character.MustClassLevels([]character.ClassLevel{{Class: "wizard", Level: level}})
}

func grpD1Fighter() character.ClassLevels {
	return character.MustClassLevels([]character.ClassLevel{{Class: "fighter", Level: 1}})
}

func grpD1Book(ids ...string) character.Spellbook {
	spells := make([]content.SpellID, len(ids))
	for i, id := range ids {
		spells[i] = content.SpellID(id)
	}
	return character.NewSpellbook(spells)
}

// grpD1As asserts that a use case returned no error and a result of type T.
func grpD1As[T any](t *testing.T, p grpD1Pair) T {
	t.Helper()
	if p.err != nil {
		t.Fatalf("Run error = %v, want nil", p.err)
	}
	v, ok := p.res.(T)
	if !ok {
		t.Fatalf("result = %#v, want %T", p.res, v)
	}
	return v
}

func grpD1Equal(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %#v, want %#v", what, got, want)
	}
}

// grpD1PortFailure asserts that err is the content port's *srderr.PortError wrapping cause.
func grpD1PortFailure(t *testing.T, err, cause error) {
	t.Helper()
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != content.Port || !errors.Is(err, cause) {
		t.Errorf("error = %v, want a content *srderr.PortError wrapping %v", err, cause)
	}
}

var errGrpD1Down = errors.New("content source down")

// Requirement: doc/specs/class-wizard.md FR-004, FR-005, FR-006 - the two Wizard skills come from the seven the class lists; a wrong count, a repeat or a skill outside the list is a returned rejection (SRD 5.2.1, "Core Wizard Traits", p. 77).
// Path: happy, then unhappy (invalid input, boundary).
func TestGrpD1WizardChoosesSkills(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := character.MustIdentifier("d1-skills")
	uc := mustOK(character.NewChooseWizardSkills(data.MustBundledSource()))

	applied := grpD1As[character.WizardSkillsApplied](t, grpD1R(uc.Run(ctx, character.ChooseWizardSkillsCommand{CharacterID: id, Skills: []content.Skill{content.SkillArcana, content.SkillHistory}})))
	grpD1Equal(t, "chosen skills", applied.Skills(), []content.Skill{content.SkillArcana, content.SkillHistory})
	if len(applied.Events()) != 1 {
		t.Errorf("events = %d, want the one WizardSkillsChosen", len(applied.Events()))
	}
	for _, c := range []struct {
		name   string
		skills []content.Skill
		want   character.SkillsRejection
	}{
		{"one skill", []content.Skill{content.SkillArcana}, character.SkillsRejection{Kind: character.SkillsWrongSkillCount, Count: 1}},
		{"three skills", []content.Skill{content.SkillArcana, content.SkillHistory, content.SkillNature}, character.SkillsRejection{Kind: character.SkillsWrongSkillCount, Count: 3}},
		{"a skill twice", []content.Skill{content.SkillArcana, content.SkillArcana}, character.SkillsRejection{Kind: character.SkillsDuplicateSkill, Skill: content.SkillArcana}},
		{"a skill off the list", []content.Skill{content.SkillArcana, content.SkillAthletics}, character.SkillsRejection{Kind: character.SkillsSkillNotAllowed, Skill: content.SkillAthletics}},
	} {
		rej := grpD1As[character.ChooseWizardSkillsRejected](t, grpD1R(uc.Run(ctx, character.ChooseWizardSkillsCommand{CharacterID: id, Skills: c.skills})))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
}

// Requirement: doc/specs/class-wizard.md FR-093, FR-094 and FR-095 - a Wizard class the source lacks is a ContentMissing result, a failing source is the content port's *srderr.PortError with no partial result, and invalid input is an error matching srderr.ErrInvalidInput.
// Path: unhappy (missing content, collaborator failure, invalid input).
func TestGrpD1WizardCreationUseCasesFailTheDocumentedWays(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := character.MustIdentifier("d1-fail")
	skills := []content.Skill{content.SkillArcana, content.SkillHistory}

	empty := mustOK(character.NewChooseWizardSkills(contenttest.NewFakeSource()))
	missing := grpD1As[character.ChooseWizardSkillsContentMissing](t, grpD1R(empty.Run(ctx, character.ChooseWizardSkillsCommand{CharacterID: id, Skills: skills})))
	if len(missing.Missing()) != 1 {
		t.Errorf("missing = %v, want the Wizard class", missing.Missing())
	}
	grpD1As[character.ChooseWizardEquipmentContentMissing](t, grpD1R(mustOK(character.NewChooseWizardEquipment(contenttest.NewFakeSource())).Run(ctx, character.ChooseWizardEquipmentCommand{CharacterID: id, Option: "A"})))

	down := mustOK(character.NewChooseWizardEquipment(contenttest.NewFakeSource().FailLookup(errGrpD1Down)))
	res, err := down.Run(ctx, character.ChooseWizardEquipmentCommand{CharacterID: id, Option: "A"})
	if res != nil {
		t.Errorf("result = %#v, want none on a port failure", res)
	}
	grpD1PortFailure(t, err, errGrpD1Down)

	uc := mustOK(character.NewChooseWizardSkills(data.MustBundledSource()))
	if _, err := uc.Run(ctx, character.ChooseWizardSkillsCommand{Skills: skills}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero CharacterID: error = %v, want ErrInvalidInput", err)
	}
	if _, err := character.NewChooseWizardSkills(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil source: error = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, character.ChooseWizardSkillsCommand{CharacterID: id, Skills: skills}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: error = %v, want context.Canceled", err)
	}
}

// Requirement: doc/specs/class-wizard.md FR-009, FR-010, FR-011 - option A is the definition's gear and 5 GP, option B is 55 GP and no items, any other value is a returned rejection naming the raw option (SRD 5.2.1, "Core Wizard Traits", p. 77).
// Path: happy, then unhappy (invalid input).
func TestGrpD1WizardChoosesStartingEquipment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := character.MustIdentifier("d1-gear")
	uc := mustOK(character.NewChooseWizardEquipment(data.MustBundledSource()))

	a := grpD1As[character.WizardEquipmentApplied](t, grpD1R(uc.Run(ctx, character.ChooseWizardEquipmentCommand{CharacterID: id, Option: "A"})))
	if a.Option() != "A" || a.Equipment().Gold() != 5 || len(a.Equipment().Items()) != 5 || len(a.Events()) != 1 {
		t.Errorf("option A = %q, %d GP, %d item lines, %d events; want 5 GP, 5 lines, one event", a.Option(), a.Equipment().Gold(), len(a.Equipment().Items()), len(a.Events()))
	}
	b := grpD1As[character.WizardEquipmentApplied](t, grpD1R(uc.Run(ctx, character.ChooseWizardEquipmentCommand{CharacterID: id, Option: "B"})))
	if b.Equipment().Gold() != 55 || len(b.Equipment().Items()) != 0 {
		t.Errorf("option B = %d GP and %d item lines, want 55 GP and none", b.Equipment().Gold(), len(b.Equipment().Items()))
	}
	for _, option := range []string{"C", "a", ""} {
		rej := grpD1As[character.ChooseWizardEquipmentRejected](t, grpD1R(uc.Run(ctx, character.ChooseWizardEquipmentCommand{CharacterID: id, Option: option})))
		grpD1Equal(t, "option "+option, rej.Rejection, character.EquipmentRejection{Kind: character.EquipmentUnknownEquipmentOption, Option: option})
	}
}

// Requirement: doc/specs/class-wizard.md FR-027, FR-028 and FR-099 - three distinct Wizard cantrips at level 1; a wrong count, a repeat, a spell that is not a cantrip and a cantrip off the Wizard list are returned rejections, and a character with no Wizard levels gets NotAWizard (SRD 5.2.1, "Level 1: Spellcasting", Cantrips, p. 77).
// Path: happy, then unhappy (boundary, invalid input).
func TestGrpD1WizardChoosesCantrips(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-cantrips")
	uc := mustOK(character.NewChooseWizardCantrips(src))
	guidance := character.WizardSpellFacts{ID: "guidance", Level: 0, Classes: []content.ClassID{"cleric"}}
	cmd := func(levels character.ClassLevels, chosen ...character.WizardSpellFacts) character.ChooseWizardCantripsCommand {
		return character.ChooseWizardCantripsCommand{CharacterID: id, ClassLevels: levels, Chosen: chosen}
	}
	fire, prestidigitation, frost := grpD1Facts(t, src, "fire_bolt", true), grpD1Facts(t, src, "prestidigitation", false), grpD1Facts(t, src, "ray_of_frost", true)
	burningHands := grpD1Facts(t, src, "burning_hands", true)

	ok := grpD1As[character.WizardCantripsApplied](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(1), fire, prestidigitation, frost))))
	grpD1Equal(t, "cantrips", ok.Cantrips(), []content.SpellID{"fire_bolt", "prestidigitation", "ray_of_frost"})
	for _, c := range []struct {
		name   string
		chosen []character.WizardSpellFacts
		want   character.CantripChoiceRejection
	}{
		{"two cantrips", []character.WizardSpellFacts{fire, frost}, character.CantripChoiceRejection{Kind: character.CantripChoiceWrongCantripCount, Allowed: 3}},
		{"a cantrip twice", []character.WizardSpellFacts{fire, fire, frost}, character.CantripChoiceRejection{Kind: character.CantripChoiceDuplicateCantrip, Spell: "fire_bolt"}},
		{"a level 1 spell", []character.WizardSpellFacts{fire, frost, burningHands}, character.CantripChoiceRejection{Kind: character.CantripChoiceNotACantrip, Spell: "burning_hands"}},
		{"a cleric cantrip", []character.WizardSpellFacts{fire, frost, guidance}, character.CantripChoiceRejection{Kind: character.CantripChoiceNotOnWizardList, Spell: "guidance"}},
	} {
		rej := grpD1As[character.ChooseWizardCantripsRejected](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(1), c.chosen...))))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ChooseWizardCantripsNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), fire, prestidigitation, frost))))
	if _, err := uc.Run(ctx, cmd(grpD1Wizard(1), character.WizardSpellFacts{Level: 0})); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("facts with no id: error = %v, want ErrInvalidInput", err)
	}
	grpD1As[character.ChooseWizardCantripsContentMissing](t, grpD1R(mustOK(character.NewChooseWizardCantrips(contenttest.NewFakeSource())).Run(ctx, cmd(grpD1Wizard(1), fire, prestidigitation, frost))))
}

// Requirement: doc/specs/class-wizard.md FR-035, FR-036, FR-093, FR-094, FR-095 - a found spell is copied into the spellbook at 2 hours and 50 GP per Spell Level, up to the highest Spell Level the Wizard has slots for; a cantrip, a spell off the list, a spell too high and a spell already held are returned rejections that cost nothing (SRD 5.2.1, "Expanding and Replacing a Spellbook", p. 78).
// Path: happy, then unhappy (boundary, invalid input, missing content, collaborator failure).
func TestGrpD1WizardCopiesAFoundSpellIntoTheBook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-copy-in")
	uc := mustOK(character.NewCopySpellIntoBook(src))
	book := grpD1Book("burning_hands", "magic_missile", "sleep")
	cmd := func(levels character.ClassLevels, b character.Spellbook, spell string) character.CopySpellIntoBookCommand {
		return character.CopySpellIntoBookCommand{CharacterID: id, ClassLevels: levels, Spellbook: b, Spell: grpD1Facts(t, src, spell, false)}
	}

	changed := grpD1As[character.SpellbookChanged](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(5), book, "fireball"))))
	cost, costed := changed.Cost()
	if !costed || cost.Hours() != 6 || cost.GoldPieces() != 150 || !changed.Spellbook().Contains("fireball") || len(changed.Spellbook().Spells()) != 4 || len(changed.Events()) != 1 {
		t.Errorf("a level 3 copy = cost %+v (%v), %v, %d events; want 6 hours, 150 GP, 4 spells, one event", cost, costed, changed.Spellbook().Spells(), len(changed.Events()))
	}
	if len(book.Spells()) != 3 {
		t.Error("the input spellbook must not change")
	}

	for _, c := range []struct {
		name   string
		levels character.ClassLevels
		spell  string
		want   character.SpellbookRejection
	}{
		{"a cantrip", grpD1Wizard(5), "fire_bolt", character.SpellbookRejection{Kind: character.SpellbookIsACantrip, Spell: "fire_bolt"}},
		{"a cleric spell", grpD1Wizard(5), "divine_smite", character.SpellbookRejection{Kind: character.SpellbookNotOnWizardList, Spell: "divine_smite"}},
		{"a spell above the highest level", grpD1Wizard(5), "power_word_kill", character.SpellbookRejection{Kind: character.SpellbookAboveHighestSpellLevel, Highest: 3, Spell: "power_word_kill"}},
		{"a spell already in the book", grpD1Wizard(5), "sleep", character.SpellbookRejection{Kind: character.SpellbookAlreadyInSpellbook, Spell: "sleep"}},
	} {
		rej := grpD1As[character.CopySpellIntoBookRejected](t, grpD1R(uc.Run(ctx, cmd(c.levels, book, c.spell))))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.CopySpellIntoBookNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), book, "fireball"))))
	grpD1As[character.CopySpellIntoBookContentMissing](t, grpD1R(mustOK(character.NewCopySpellIntoBook(contenttest.NewFakeSource())).Run(ctx, cmd(grpD1Wizard(5), book, "fireball"))))
	res, err := mustOK(character.NewCopySpellIntoBook(contenttest.NewFakeSource().FailLookup(errGrpD1Down))).Run(ctx, cmd(grpD1Wizard(5), book, "fireball"))
	if res != nil {
		t.Errorf("result = %#v, want none on a port failure", res)
	}
	grpD1PortFailure(t, err, errGrpD1Down)
	if _, err := uc.Run(ctx, character.CopySpellIntoBookCommand{ClassLevels: grpD1Wizard(5), Spellbook: book, Spell: grpD1Facts(t, src, "fireball", false)}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero CharacterID: error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-wizard.md FR-037, FR-038, FR-039 - copying the book costs 1 hour and 10 GP per Spell Level of each copied spell and produces no event; a spell the source book lacks is a returned rejection; a lost book is transcribed from the prepared spells at the same cost, leaving cantrips behind (SRD 5.2.1, "Expanding and Replacing a Spellbook", Copying the Book and Replacing the Book, p. 78).
// Path: happy, then unhappy (empty, invalid input).
func TestGrpD1WizardCopiesAndReplacesTheSpellbook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	source := grpD1Book("burning_hands", "fireball", "magic_missile")
	copyBook := character.NewCopySpellbook()

	changed := grpD1As[character.SpellbookChanged](t, grpD1R(copyBook.Run(ctx, character.CopySpellbookCommand{ClassLevels: grpD1Wizard(5), Source: source, Spells: grpD1Facts2(t, src, "burning_hands", "fireball")})))
	cost, costed := changed.Cost()
	grpD1Equal(t, "copied spells", changed.Spellbook().Spells(), []content.SpellID{"burning_hands", "fireball"})
	if !costed || cost.Hours() != 4 || cost.GoldPieces() != 40 || len(changed.Events()) != 0 {
		t.Errorf("a copy of levels 1 and 3 = cost %+v, %d events; want 4 hours, 40 GP and no event", cost, len(changed.Events()))
	}
	none := grpD1As[character.SpellbookChanged](t, grpD1R(copyBook.Run(ctx, character.CopySpellbookCommand{ClassLevels: grpD1Wizard(5), Source: source})))
	if c, _ := none.Cost(); c.Hours() != 0 || c.GoldPieces() != 0 || len(none.Spellbook().Spells()) != 0 {
		t.Errorf("copying nothing = %+v and %v, want no cost and an empty book", c, none.Spellbook().Spells())
	}
	rej := grpD1As[character.CopySpellbookRejected](t, grpD1R(copyBook.Run(ctx, character.CopySpellbookCommand{ClassLevels: grpD1Wizard(5), Source: source, Spells: grpD1Facts2(t, src, "sleep")})))
	grpD1Equal(t, "a spell the source lacks", rej.Rejection, character.SpellbookRejection{Kind: character.SpellbookNotInSourceSpellbook, Spell: "sleep"})
	grpD1As[character.CopySpellbookNotAWizard](t, grpD1R(copyBook.Run(ctx, character.CopySpellbookCommand{ClassLevels: grpD1Fighter(), Source: source})))
	if _, err := copyBook.Run(ctx, character.CopySpellbookCommand{ClassLevels: grpD1Wizard(5), Spells: []character.WizardSpellFacts{{}}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("facts with no id: error = %v, want ErrInvalidInput", err)
	}

	id := character.MustIdentifier("d1-lost-book")
	replace := character.NewReplaceLostSpellbook()
	prepared := append(grpD1Facts2(t, src, "magic_missile", "fireball", "fire_bolt"), grpD1Facts2(t, src, "magic_missile")...)
	fresh := grpD1As[character.SpellbookChanged](t, grpD1R(replace.Run(ctx, character.ReplaceLostSpellbookCommand{CharacterID: id, ClassLevels: grpD1Wizard(5), Prepared: prepared})))
	cost, _ = fresh.Cost()
	grpD1Equal(t, "the new book", fresh.Spellbook().Spells(), []content.SpellID{"magic_missile", "fireball"})
	if cost.Hours() != 4 || cost.GoldPieces() != 40 || len(fresh.Events()) != 1 {
		t.Errorf("a replacement = cost %+v, %d events; want 4 hours, 40 GP, one event", cost, len(fresh.Events()))
	}
	blank := grpD1As[character.SpellbookChanged](t, grpD1R(replace.Run(ctx, character.ReplaceLostSpellbookCommand{CharacterID: id, ClassLevels: grpD1Wizard(5), Prepared: grpD1Facts2(t, src, "fire_bolt")})))
	if c, _ := blank.Cost(); len(blank.Spellbook().Spells()) != 0 || c.GoldPieces() != 0 || len(blank.Events()) != 0 {
		t.Errorf("only a cantrip prepared = %v, %+v, %d events; want an empty book at no cost", blank.Spellbook().Spells(), c, len(blank.Events()))
	}
	grpD1As[character.ReplaceLostSpellbookNotAWizard](t, grpD1R(replace.Run(ctx, character.ReplaceLostSpellbookCommand{CharacterID: id, ClassLevels: grpD1Fighter()})))
	if _, err := replace.Run(ctx, character.ReplaceLostSpellbookCommand{ClassLevels: grpD1Wizard(5)}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero CharacterID: error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-wizard.md FR-041, FR-042, FR-043, FR-044, FR-102 - a Ritual spell in a readable spellbook is eligible to cast as a Ritual without being prepared; a spell without the Ritual tag, a spell not in the book and an unreadable book are returned rejections (SRD 5.2.1, "Level 1: Ritual Adept", p. 78).
// Path: happy, then unhappy (state errors).
func TestGrpD1WizardChecksRitualAdept(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	uc := character.NewCheckRitualAdept()
	book := grpD1Book("detect_magic", "magic_missile")
	cmd := func(levels character.ClassLevels, spell string, available bool) character.CheckRitualAdeptCommand {
		return character.CheckRitualAdeptCommand{ClassLevels: levels, Spellbook: book, Spell: grpD1Facts(t, src, spell, false), SpellbookAvailable: available}
	}

	grpD1As[character.RitualAdeptEligible](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(1), "detect_magic", true))))
	for _, c := range []struct {
		name      string
		spell     string
		available bool
		want      character.RitualAdeptRejectionKind
	}{
		{"a spell without the Ritual tag", "magic_missile", true, character.RitualAdeptNotARitualSpell},
		{"a Ritual spell not in the book", "contact_other_plane", true, character.RitualAdeptNotInSpellbook},
		{"an unreadable book", "detect_magic", false, character.RitualAdeptSpellbookUnavailable},
	} {
		rej := grpD1As[character.CheckRitualAdeptRejected](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(1), c.spell, c.available))))
		if rej.Rejection.Kind != c.want {
			t.Errorf("%s: kind = %v, want %v", c.name, rej.Rejection.Kind, c.want)
		}
	}
	grpD1As[character.CheckRitualAdeptNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), "detect_magic", true))))
	if _, err := uc.Run(ctx, character.CheckRitualAdeptCommand{ClassLevels: grpD1Wizard(1), Spell: character.WizardSpellFacts{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("facts with no id: error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-wizard.md FR-053, FR-054, FR-055, FR-056 - at Wizard level 2 a proficient skill from the Scholar list gains Expertise once; a skill off the list, a skill without proficiency, a repeat and level 1 are returned rejections (SRD 5.2.1, "Level 2: Scholar", p. 78).
// Path: happy, then unhappy (boundary, state errors).
func TestGrpD1WizardChoosesScholarExpertise(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	id := character.MustIdentifier("d1-scholar")
	uc := mustOK(character.NewChooseScholarExpertise(data.MustBundledSource()))
	proficient := []content.Skill{content.SkillArcana, content.SkillInsight}
	cmd := func(levels character.ClassLevels, resources character.WizardResources, skill content.Skill) character.ChooseScholarExpertiseCommand {
		return character.ChooseScholarExpertiseCommand{CharacterID: id, ClassLevels: levels, Resources: resources, Proficient: proficient, Skill: skill}
	}

	applied := grpD1As[character.ScholarApplied](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(2), character.WizardResources{}, content.SkillArcana))))
	if held, ok := applied.Resources().Scholar(); !ok || held != content.SkillArcana || applied.Skill() != content.SkillArcana || len(applied.Events()) != 1 {
		t.Errorf("Scholar = %v (%v), skill %v, %d events; want Arcana held and one event", held, ok, applied.Skill(), len(applied.Events()))
	}
	for _, c := range []struct {
		name      string
		levels    character.ClassLevels
		resources character.WizardResources
		skill     content.Skill
		want      character.ScholarRejection
	}{
		{"level 1", grpD1Wizard(1), character.WizardResources{}, content.SkillArcana, character.ScholarRejection{Kind: character.ScholarFeatureNotAvailable}},
		{"a second choice", grpD1Wizard(2), applied.Resources(), content.SkillArcana, character.ScholarRejection{Kind: character.ScholarAlready}},
		{"Insight, off the Scholar list", grpD1Wizard(2), character.WizardResources{}, content.SkillInsight, character.ScholarRejection{Kind: character.ScholarSkillNotEligible, Skill: content.SkillInsight}},
		{"Nature without proficiency", grpD1Wizard(2), character.WizardResources{}, content.SkillNature, character.ScholarRejection{Kind: character.ScholarNotProficient, Skill: content.SkillNature}},
	} {
		rej := grpD1As[character.ChooseScholarExpertiseRejected](t, grpD1R(uc.Run(ctx, cmd(c.levels, c.resources, c.skill))))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ChooseScholarExpertiseNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), character.WizardResources{}, content.SkillArcana))))
	grpD1As[character.ChooseScholarExpertiseContentMissing](t, grpD1R(mustOK(character.NewChooseScholarExpertise(contenttest.NewFakeSource())).Run(ctx, cmd(grpD1Wizard(2), character.WizardResources{}, content.SkillArcana))))
}

// Requirement: doc/specs/class-wizard.md FR-061, FR-062, FR-063 - after a Short Rest a Wizard of level 5 or more swaps one prepared spell for a spellbook spell; below level 5, outside a finished Short Rest, an unprepared spell, a cantrip, a spell not in the book, a spell above the highest level and a spell already prepared are returned rejections (SRD 5.2.1, "Level 5: Memorize Spell", p. 78).
// Path: happy, then unhappy (boundary, state errors).
func TestGrpD1WizardMemorizesASpell(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-memorize")
	uc := mustOK(character.NewMemorizeSpell(src))
	book := grpD1Book("magic_missile", "sleep", "burning_hands", "fireball", "ice_storm")
	prepared := []content.SpellID{"magic_missile", "sleep"}
	shortRest := finishedRest(t, id, character.ShortRest)
	cmd := func(levels character.ClassLevels, trigger character.RestApplied, replace, next string) character.MemorizeSpellCommand {
		return character.MemorizeSpellCommand{CharacterID: id, ClassLevels: levels, Prepared: prepared, Spellbook: book, Replace: content.SpellID(replace), New: grpD1Facts(t, src, next, false), Trigger: trigger}
	}

	applied := grpD1As[character.MemorizeApplied](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(5), shortRest, "sleep", "fireball"))))
	grpD1Equal(t, "prepared spells", applied.Prepared(), []content.SpellID{"magic_missile", "fireball"})
	if len(applied.Events()) != 1 || len(prepared) != 2 || prepared[1] != "sleep" {
		t.Errorf("events = %d, input = %v; want one event and the input prepared list unchanged", len(applied.Events()), prepared)
	}
	for _, c := range []struct {
		name string
		cmd  character.MemorizeSpellCommand
		want character.MemorizeRejection
	}{
		{"level 4", cmd(grpD1Wizard(4), shortRest, "sleep", "fireball"), character.MemorizeRejection{Kind: character.MemorizeFeatureNotAvailable}},
		{"no finished Short Rest", cmd(grpD1Wizard(5), character.RestApplied{}, "sleep", "fireball"), character.MemorizeRejection{Kind: character.MemorizeNotAfterShortRest}},
		{"a spell that is not prepared", cmd(grpD1Wizard(5), shortRest, "fog_cloud", "fireball"), character.MemorizeRejection{Kind: character.MemorizeNotPrepared, Spell: "fog_cloud"}},
		{"a cantrip", cmd(grpD1Wizard(5), shortRest, "sleep", "fire_bolt"), character.MemorizeRejection{Kind: character.MemorizeNewSpellIsCantrip, Spell: "fire_bolt"}},
		{"a spell not in the book", cmd(grpD1Wizard(5), shortRest, "sleep", "fly"), character.MemorizeRejection{Kind: character.MemorizeNewSpellNotInSpellbook, Spell: "fly"}},
		{"a level 4 spell at level 5", cmd(grpD1Wizard(5), shortRest, "sleep", "ice_storm"), character.MemorizeRejection{Kind: character.MemorizeNewSpellAboveHighestLevel, Spell: "ice_storm", Highest: 3}},
		{"a spell already prepared", cmd(grpD1Wizard(5), shortRest, "sleep", "magic_missile"), character.MemorizeRejection{Kind: character.MemorizeNewSpellAlreadyPrepared, Spell: "magic_missile"}},
	} {
		rej := grpD1As[character.MemorizeSpellRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.MemorizeSpellNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), shortRest, "sleep", "fireball"))))
	grpD1As[character.MemorizeSpellContentMissing](t, grpD1R(mustOK(character.NewMemorizeSpell(contenttest.NewFakeSource())).Run(ctx, cmd(grpD1Wizard(5), shortRest, "sleep", "fireball"))))
}

// Requirement: doc/specs/class-wizard.md FR-064, FR-065, FR-066, FR-067, FR-068, FR-069, FR-070 - at level 18 a Wizard names one level 1 and one level 2 Action spell from the book as Spell Mastery, they are always prepared (so Memorize Spell refuses to replace them), and a Long Rest lets one be swapped for an eligible spell of the same level; every misuse is a returned rejection (SRD 5.2.1, "Level 18: Spell Mastery", p. 79).
// Path: happy, then unhappy (boundary, invalid input, state errors).
func TestGrpD1WizardMasterTwoSpells(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-mastery")
	choose := mustOK(character.NewChooseSpellMastery(src))
	replace := mustOK(character.NewReplaceSpellMastery(src))
	book := grpD1Book("burning_hands", "chromatic_orb", "scorching_ray", "web", "misty_step", "sleep")
	pick := func(levels character.ClassLevels, resources character.WizardResources, spells ...string) character.ChooseSpellMasteryCommand {
		return character.ChooseSpellMasteryCommand{CharacterID: id, ClassLevels: levels, Resources: resources, Spellbook: book, Spells: grpD1Facts2(t, src, spells...)}
	}

	applied := grpD1As[character.SpellMasteryApplied](t, grpD1R(choose.Run(ctx, pick(grpD1Wizard(18), character.WizardResources{}, "scorching_ray", "burning_hands"))))
	if applied.Mastery().LevelOne() != "burning_hands" || applied.Mastery().LevelTwo() != "scorching_ray" || len(applied.Events()) != 1 {
		t.Errorf("mastery = %v and %v with %d events; want burning_hands, scorching_ray and one event", applied.Mastery().LevelOne(), applied.Mastery().LevelTwo(), len(applied.Events()))
	}
	grpD1Equal(t, "always prepared", character.WizardAlwaysPrepared{}.Of(applied.Resources()), []content.SpellID{"burning_hands", "scorching_ray"})

	for _, c := range []struct {
		name string
		cmd  character.ChooseSpellMasteryCommand
		want character.SpellMasteryRejection
	}{
		{"level 17", pick(grpD1Wizard(17), character.WizardResources{}, "burning_hands", "scorching_ray"), character.SpellMasteryRejection{Kind: character.SpellMasteryFeatureNotAvailable}},
		{"a second choice", pick(grpD1Wizard(18), applied.Resources(), "burning_hands", "scorching_ray"), character.SpellMasteryRejection{Kind: character.SpellMasteryAlready}},
		{"a spell not in the book", pick(grpD1Wizard(18), character.WizardResources{}, "burning_hands", "blur"), character.SpellMasteryRejection{Kind: character.SpellMasteryNotInSpellbook, Spell: "blur"}},
		{"a spell twice", pick(grpD1Wizard(18), character.WizardResources{}, "burning_hands", "burning_hands"), character.SpellMasteryRejection{Kind: character.SpellMasteryDuplicateSpell, Spell: "burning_hands"}},
		{"two level 1 spells", pick(grpD1Wizard(18), character.WizardResources{}, "burning_hands", "chromatic_orb"), character.SpellMasteryRejection{Kind: character.SpellMasteryWrongSpellLevels, Levels: []int{1, 1}}},
		{"a bonus action spell", pick(grpD1Wizard(18), character.WizardResources{}, "burning_hands", "misty_step"), character.SpellMasteryRejection{Kind: character.SpellMasteryCastingTimeNotAction, Spell: "misty_step"}},
	} {
		rej := grpD1As[character.ChooseSpellMasteryRejected](t, grpD1R(choose.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ChooseSpellMasteryNotAWizard](t, grpD1R(choose.Run(ctx, pick(grpD1Fighter(), character.WizardResources{}, "burning_hands", "scorching_ray"))))
	grpD1As[character.ChooseSpellMasteryContentMissing](t, grpD1R(mustOK(character.NewChooseSpellMastery(contenttest.NewFakeSource())).Run(ctx, pick(grpD1Wizard(18), character.WizardResources{}, "burning_hands", "scorching_ray"))))

	// Memorize Spell cannot replace a Mastery spell, because it is always prepared.
	memorize := mustOK(character.NewMemorizeSpell(src))
	refused := grpD1As[character.MemorizeSpellRejected](t, grpD1R(memorize.Run(ctx, character.MemorizeSpellCommand{CharacterID: id, ClassLevels: grpD1Wizard(18), Resources: applied.Resources(),
		Prepared: []content.SpellID{"burning_hands", "sleep"}, Spellbook: book, Replace: "burning_hands", New: grpD1Facts(t, src, "chromatic_orb", false), Trigger: finishedRest(t, id, character.ShortRest)})))
	grpD1Equal(t, "replacing a Mastery spell", refused.Rejection, character.MemorizeRejection{Kind: character.MemorizeAlwaysPreparedNotReplaceable, Spell: "burning_hands"})

	longRest, shortRest := finishedRest(t, id, character.LongRest), finishedRest(t, id, character.ShortRest)
	swap := func(levels character.ClassLevels, resources character.WizardResources, trigger character.RestApplied, old, next string) character.ReplaceSpellMasteryCommand {
		return character.ReplaceSpellMasteryCommand{CharacterID: id, ClassLevels: levels, Resources: resources, Spellbook: book, Replace: content.SpellID(old), New: grpD1Facts(t, src, next, false), Trigger: trigger}
	}
	swapped := grpD1As[character.SpellMasteryReplaceApplied](t, grpD1R(replace.Run(ctx, swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "web"))))
	if swapped.Mastery().LevelOne() != "burning_hands" || swapped.Mastery().LevelTwo() != "web" || len(swapped.Events()) != 1 {
		t.Errorf("after the swap mastery = %v and %v with %d events; want burning_hands, web and one event", swapped.Mastery().LevelOne(), swapped.Mastery().LevelTwo(), len(swapped.Events()))
	}
	if held, _ := applied.Resources().SpellMastery(); held.LevelTwo() != "scorching_ray" {
		t.Error("the input resources must not change")
	}
	for _, c := range []struct {
		name string
		cmd  character.ReplaceSpellMasteryCommand
		want character.SpellMasteryRejection
	}{
		{"level 17", swap(grpD1Wizard(17), applied.Resources(), longRest, "scorching_ray", "web"), character.SpellMasteryRejection{Kind: character.SpellMasteryFeatureNotAvailable}},
		{"after a Short Rest", swap(grpD1Wizard(18), applied.Resources(), shortRest, "scorching_ray", "web"), character.SpellMasteryRejection{Kind: character.SpellMasteryNotAfterLongRest}},
		{"no Mastery held", swap(grpD1Wizard(18), character.WizardResources{}, longRest, "scorching_ray", "web"), character.SpellMasteryRejection{Kind: character.SpellMasteryNotHeld}},
		{"a spell that is not a Mastery spell", swap(grpD1Wizard(18), applied.Resources(), longRest, "sleep", "web"), character.SpellMasteryRejection{Kind: character.SpellMasteryNotAMasterySpell, Spell: "sleep"}},
		{"a spell not in the book", swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "blur"), character.SpellMasteryRejection{Kind: character.SpellMasteryNotInSpellbook, Spell: "blur"}},
		{"the other Mastery spell", swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "burning_hands"), character.SpellMasteryRejection{Kind: character.SpellMasteryDuplicateSpell, Spell: "burning_hands"}},
		{"a spell of another level", swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "chromatic_orb"), character.SpellMasteryRejection{Kind: character.SpellMasteryWrongSpellLevels, Levels: []int{1}}},
		{"a bonus action spell", swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "misty_step"), character.SpellMasteryRejection{Kind: character.SpellMasteryCastingTimeNotAction, Spell: "misty_step"}},
	} {
		rej := grpD1As[character.ReplaceSpellMasteryRejected](t, grpD1R(replace.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ReplaceSpellMasteryNotAWizard](t, grpD1R(replace.Run(ctx, swap(grpD1Fighter(), applied.Resources(), longRest, "scorching_ray", "web"))))
	grpD1As[character.ReplaceSpellMasteryContentMissing](t, grpD1R(mustOK(character.NewReplaceSpellMastery(contenttest.NewFakeSource())).Run(ctx, swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "web"))))
	res, err := mustOK(character.NewReplaceSpellMastery(contenttest.NewFakeSource().FailLookup(errGrpD1Down))).Run(ctx, swap(grpD1Wizard(18), applied.Resources(), longRest, "scorching_ray", "web"))
	if res != nil {
		t.Errorf("result = %#v, want none on a port failure", res)
	}
	grpD1PortFailure(t, err, errGrpD1Down)
}

// Requirement: doc/specs/class-wizard.md FR-070, FR-071, FR-072, FR-073, FR-074, FR-075 - at level 20 a Wizard names two distinct level 3 spells from the book as Signature Spells, castable once each at level 3 without a slot until a rest; every misuse is a returned rejection (SRD 5.2.1, "Level 20: Signature Spells", p. 79).
// Path: happy, then unhappy (boundary, state errors).
func TestGrpD1WizardChoosesSignatureSpells(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-signature")
	uc := mustOK(character.NewChooseSignatureSpells(src))
	book := grpD1Book("fireball", "lightning_bolt", "scorching_ray", "fear")
	pick := func(levels character.ClassLevels, resources character.WizardResources, spells ...string) character.ChooseSignatureSpellsCommand {
		return character.ChooseSignatureSpellsCommand{CharacterID: id, ClassLevels: levels, Resources: resources, Spellbook: book, Spells: grpD1Facts2(t, src, spells...)}
	}

	applied := grpD1As[character.SignatureSpellsApplied](t, grpD1R(uc.Run(ctx, pick(grpD1Wizard(20), character.WizardResources{}, "fireball", "lightning_bolt"))))
	grpD1Equal(t, "Signature Spells", applied.Signature().Spells(), []content.SpellID{"fireball", "lightning_bolt"})
	grpD1Equal(t, "always prepared", character.WizardAlwaysPrepared{}.Of(applied.Resources()), []content.SpellID{"fireball", "lightning_bolt"})
	if len(applied.Events()) != 1 || applied.Resources().SignatureSpellSpent("fireball") {
		t.Errorf("events = %d, spent = %v; want one event and fireball not yet spent", len(applied.Events()), applied.Resources().SignatureSpellSpent("fireball"))
	}
	spent, err := applied.Resources().RecordSignatureSpellCast("fireball")
	if err != nil || !spent.SignatureSpellSpent("fireball") || spent.SignatureSpellSpent("lightning_bolt") || spent.AfterRest(character.ShortRest).SignatureSpellSpent("fireball") {
		t.Errorf("after casting fireball = %v, %v; want only fireball spent until a Short Rest", spent.SignatureSpellSpent("fireball"), err)
	}

	for _, c := range []struct {
		name string
		cmd  character.ChooseSignatureSpellsCommand
		want character.SignatureSpellsRejection
	}{
		{"level 19", pick(grpD1Wizard(19), character.WizardResources{}, "fireball", "lightning_bolt"), character.SignatureSpellsRejection{Kind: character.SignatureSpellsFeatureNotAvailable}},
		{"a second choice", pick(grpD1Wizard(20), applied.Resources(), "fireball", "fear"), character.SignatureSpellsRejection{Kind: character.SignatureSpellsAlready}},
		{"one spell", pick(grpD1Wizard(20), character.WizardResources{}, "fireball"), character.SignatureSpellsRejection{Kind: character.SignatureSpellsWrongSpellCount, Allowed: 2}},
		{"a spell twice", pick(grpD1Wizard(20), character.WizardResources{}, "fireball", "fireball"), character.SignatureSpellsRejection{Kind: character.SignatureSpellsDuplicateSpell, Spell: "fireball"}},
		{"a level 2 spell", pick(grpD1Wizard(20), character.WizardResources{}, "fireball", "scorching_ray"), character.SignatureSpellsRejection{Kind: character.SignatureSpellsNotLevelThree, Spell: "scorching_ray"}},
		{"a spell not in the book", pick(grpD1Wizard(20), character.WizardResources{}, "fireball", "fly"), character.SignatureSpellsRejection{Kind: character.SignatureSpellsNotInSpellbook, Spell: "fly"}},
	} {
		rej := grpD1As[character.ChooseSignatureSpellsRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ChooseSignatureSpellsNotAWizard](t, grpD1R(uc.Run(ctx, pick(grpD1Fighter(), character.WizardResources{}, "fireball", "lightning_bolt"))))
	grpD1As[character.ChooseSignatureSpellsContentMissing](t, grpD1R(mustOK(character.NewChooseSignatureSpells(contenttest.NewFakeSource())).Run(ctx, pick(grpD1Wizard(20), character.WizardResources{}, "fireball", "lightning_bolt"))))
	if _, err := uc.Run(ctx, character.ChooseSignatureSpellsCommand{ClassLevels: grpD1Wizard(20)}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero CharacterID: error = %v, want ErrInvalidInput", err)
	}
}

// grpD1R pairs a use case result with its error for grpD1As.
func grpD1R[T any](v T, err error) grpD1Pair { return grpD1Pair{v, err} }

type grpD1Pair struct {
	res any
	err error
}
