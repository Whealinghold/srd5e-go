package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The Evoker scenarios drive the subclass use cases through the public packages only, with the Evoker subclass of
// the bundled Wizard class and bundled spells. Specs: class-wizard-evoker.md (SRD 5.2.1, "Wizard Subclass:
// Evoker", pp. 82 to 83) and class-wizard.md FR-106 (the subclass features held at a Wizard level).

// grpD1Evoker returns the Evoker subclass of the bundled Wizard class.
func grpD1Evoker(t *testing.T, src content.Source) content.SubclassDefinition {
	t.Helper()
	found, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("wizard")))
	if err != nil {
		t.Fatal(err)
	}
	return found.(content.LookupFound).Entry.(content.ClassDefinition).Subclass("evoker").(content.SubclassFound).Subclass()
}

// Requirement: doc/specs/class-wizard-evoker.md FR-001, FR-002 - at Wizard level 3 Evocation Savant adds two Evocation Wizard spells of level 1 or 2 to the book free of cost; no subclass, level 2, a wrong count, a repeat, a cantrip, a spell off the Wizard list, a spell of another school, a spell above level 2 and a spell already held are returned rejections (SRD 5.2.1, "Level 3: Evocation Savant", p. 82).
// Path: happy, then unhappy (boundary, invalid input, missing content, collaborator failure).
func TestGrpD1EvokerAddsEvocationSavantSpells(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-savant")
	uc := mustOK(character.NewAddEvocationSavantSpells(src))
	book := grpD1Book("sleep", "magic_missile")
	cmd := func(levels character.ClassLevels, sub content.SubclassID, spells ...string) character.AddEvocationSavantSpellsCommand {
		return character.AddEvocationSavantSpellsCommand{CharacterID: id, ClassLevels: levels, Subclass: sub, Spellbook: book, Spells: grpD1Facts2(t, src, spells...)}
	}

	changed := grpD1As[character.SpellbookChanged](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(3), "evoker", "burning_hands", "scorching_ray"))))
	if _, costed := changed.Cost(); costed || len(changed.Events()) != 1 {
		t.Errorf("Evocation Savant costed = %v with %d events; want free of cost and one event", costed, len(changed.Events()))
	}
	grpD1Equal(t, "the new book", changed.Spellbook().Spells(), []content.SpellID{"sleep", "magic_missile", "burning_hands", "scorching_ray"})
	if len(book.Spells()) != 2 {
		t.Error("the input spellbook must not change")
	}

	for _, c := range []struct {
		name string
		cmd  character.AddEvocationSavantSpellsCommand
		want character.EvokerSpellbookRejection
	}{
		{"no subclass", cmd(grpD1Wizard(3), "", "burning_hands", "scorching_ray"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotEvoker}},
		{"level 2", cmd(grpD1Wizard(2), "evoker", "burning_hands", "scorching_ray"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookFeatureNotAvailable}},
		{"one spell", cmd(grpD1Wizard(3), "evoker", "burning_hands"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookWrongSpellCount, Allowed: 2}},
		{"a spell twice", cmd(grpD1Wizard(3), "evoker", "burning_hands", "burning_hands"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookDuplicateSpell, Spell: "burning_hands"}},
		{"a cantrip", cmd(grpD1Wizard(3), "evoker", "burning_hands", "fire_bolt"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookIsACantrip, Spell: "fire_bolt"}},
		{"a paladin spell", cmd(grpD1Wizard(3), "evoker", "burning_hands", "divine_smite"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotOnWizardList, Spell: "divine_smite"}},
		{"a spell of another school", cmd(grpD1Wizard(3), "evoker", "burning_hands", "fog_cloud"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotAnEvocationSpell, Spell: "fog_cloud"}},
		{"a level 3 spell", cmd(grpD1Wizard(3), "evoker", "burning_hands", "fireball"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAboveHighestSpellLevel, Highest: 2, Spell: "fireball"}},
		{"a spell already in the book", cmd(grpD1Wizard(3), "evoker", "burning_hands", "magic_missile"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAlreadyInSpellbook, Spell: "magic_missile"}},
	} {
		rej := grpD1As[character.AddEvocationSavantSpellsRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	unknown := grpD1As[character.AddEvocationSavantSpellsSubclassUnknown](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(3), "diviner", "burning_hands", "scorching_ray"))))
	if unknown.Subclass() != "diviner" {
		t.Errorf("unknown subclass = %q, want diviner", unknown.Subclass())
	}
	grpD1As[character.AddEvocationSavantSpellsNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), "evoker", "burning_hands", "scorching_ray"))))
	grpD1As[character.AddEvocationSavantSpellsContentMissing](t, grpD1R(mustOK(character.NewAddEvocationSavantSpells(contenttest.NewFakeSource())).Run(ctx, cmd(grpD1Wizard(3), "evoker", "burning_hands", "scorching_ray"))))
	res, err := mustOK(character.NewAddEvocationSavantSpells(contenttest.NewFakeSource().FailLookup(errGrpD1Down))).Run(ctx, cmd(grpD1Wizard(3), "evoker", "burning_hands", "scorching_ray"))
	if res != nil {
		t.Errorf("result = %#v, want none on a port failure", res)
	}
	grpD1PortFailure(t, err, errGrpD1Down)
	if _, err := uc.Run(ctx, character.AddEvocationSavantSpellsCommand{ClassLevels: grpD1Wizard(3), Subclass: "evoker"}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero CharacterID: error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-003, FR-004 - on gaining a new level of Spell Slots after level 3 (here level 5) the Evoker adds one Evocation Wizard spell free of cost; the subclass level itself, a level with no new slot level and every spell misuse are returned rejections (SRD 5.2.1, "Level 3: Evocation Savant", p. 82).
// Path: happy, then unhappy (boundary, state errors).
func TestGrpD1EvokerAddsAFreeSpellOnANewSlotLevel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	id := character.MustIdentifier("d1-free-spell")
	uc := mustOK(character.NewAddEvokerFreeSpell(src))
	book := grpD1Book("burning_hands", "sleep")
	cmd := func(levels character.ClassLevels, sub content.SubclassID, spell string) character.AddEvokerFreeSpellCommand {
		return character.AddEvokerFreeSpellCommand{CharacterID: id, ClassLevels: levels, Subclass: sub, Spellbook: book, Spell: grpD1Facts(t, src, spell, false)}
	}

	changed := grpD1As[character.SpellbookChanged](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(5), "evoker", "fireball"))))
	if grpD1Costed(changed) || !changed.Spellbook().Contains("fireball") || len(changed.Spellbook().Spells()) != 3 || len(changed.Events()) != 1 {
		t.Errorf("the free spell = %v (costed %v), %d events; want fireball added free of cost with one event", changed.Spellbook().Spells(), grpD1Costed(changed), len(changed.Events()))
	}
	for _, c := range []struct {
		name string
		cmd  character.AddEvokerFreeSpellCommand
		want character.EvokerSpellbookRejection
	}{
		{"no subclass", cmd(grpD1Wizard(5), "", "fireball"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotEvoker}},
		{"level 2", cmd(grpD1Wizard(2), "evoker", "magic_missile"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookFeatureNotAvailable}},
		{"the subclass level", cmd(grpD1Wizard(3), "evoker", "scorching_ray"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNoNewSpellLevel}},
		{"level 6, no new slot level", cmd(grpD1Wizard(6), "evoker", "fireball"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNoNewSpellLevel}},
		{"a cantrip", cmd(grpD1Wizard(5), "evoker", "fire_bolt"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookIsACantrip, Spell: "fire_bolt"}},
		{"a paladin spell", cmd(grpD1Wizard(5), "evoker", "divine_smite"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotOnWizardList, Spell: "divine_smite"}},
		{"a spell of another school", cmd(grpD1Wizard(5), "evoker", "dispel_magic"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookNotAnEvocationSpell, Spell: "dispel_magic"}},
		{"a level 5 spell at level 5", cmd(grpD1Wizard(5), "evoker", "cone_of_cold"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAboveHighestSpellLevel, Highest: 3, Spell: "cone_of_cold"}},
		{"a spell already in the book", cmd(grpD1Wizard(5), "evoker", "burning_hands"), character.EvokerSpellbookRejection{Kind: character.EvokerSpellbookAlreadyInSpellbook, Spell: "burning_hands"}},
	} {
		rej := grpD1As[character.AddEvokerFreeSpellRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.AddEvokerFreeSpellSubclassUnknown](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(5), "diviner", "fireball"))))
	grpD1As[character.AddEvokerFreeSpellNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), "evoker", "fireball"))))
	grpD1As[character.AddEvokerFreeSpellContentMissing](t, grpD1R(mustOK(character.NewAddEvokerFreeSpell(contenttest.NewFakeSource())).Run(ctx, cmd(grpD1Wizard(5), "evoker", "fireball"))))
	res, err := mustOK(character.NewAddEvokerFreeSpell(contenttest.NewFakeSource().FailLookup(errGrpD1Down))).Run(ctx, cmd(grpD1Wizard(5), "evoker", "fireball"))
	if res != nil {
		t.Errorf("result = %#v, want none on a port failure", res)
	}
	grpD1PortFailure(t, err, errGrpD1Down)
	if _, err := character.NewAddEvokerFreeSpell(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil source: error = %v, want ErrInvalidInput", err)
	}
}

func grpD1Costed(c character.SpellbookChanged) bool {
	_, ok := c.Cost()
	return ok
}

// Requirement: doc/specs/class-wizard-evoker.md FR-005, FR-006 - from level 3 a cantrip that misses or is saved against deals half its damage rounded down and a hit or a failed save is unchanged; the ability check, a spell that is not a cantrip, a cantrip that deals no damage, no subclass and level 2 are returned rejections (SRD 5.2.1, "Level 3: Potent Cantrip", p. 82).
// Path: happy, then unhappy (boundary, invalid input).
func TestGrpD1EvokerCantripsStillHurtOnAMiss(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	evoker := grpD1Evoker(t, src)
	uc := character.NewResolvePotentCantrip()
	fireBolt := grpD1Facts(t, src, "fire_bolt", true)
	cmd := func(levels character.ClassLevels, sub content.SubclassDefinition, spell character.WizardSpellFacts, kind d20.Kind, outcome d20.Outcome, damage int) character.ResolvePotentCantripCommand {
		return character.ResolvePotentCantripCommand{ClassLevels: levels, Subclass: sub, Spell: spell, Kind: kind, Outcome: outcome, Damage: damage}
	}

	for _, c := range []struct {
		name    string
		kind    d20.Kind
		outcome d20.Outcome
		damage  int
		want    int
	}{
		{"a missed attack roll, even damage", d20.AttackRoll, d20.Failure, 8, 4},
		{"a missed attack roll, odd damage rounds down", d20.AttackRoll, d20.Failure, 9, 4},
		{"a successful saving throw", d20.SavingThrow, d20.Success, 7, 3},
		{"a miss with no damage rolled", d20.AttackRoll, d20.Failure, 0, 0},
	} {
		half := grpD1As[character.PotentCantripHalfDamage](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(3), evoker, fireBolt, c.kind, c.outcome, c.damage))))
		if half.Damage() != c.want {
			t.Errorf("%s: damage = %d, want %d", c.name, half.Damage(), c.want)
		}
	}
	grpD1As[character.PotentCantripUnchanged](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(3), evoker, fireBolt, d20.AttackRoll, d20.Success, 9))))
	grpD1As[character.PotentCantripUnchanged](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(3), evoker, fireBolt, d20.SavingThrow, d20.Failure, 9))))

	for _, c := range []struct {
		name string
		cmd  character.ResolvePotentCantripCommand
		want character.PotentCantripRejection
	}{
		{"no subclass", cmd(grpD1Wizard(3), content.SubclassDefinition{}, fireBolt, d20.AttackRoll, d20.Failure, 8), character.PotentCantripRejection{Kind: character.PotentCantripNotEvoker}},
		{"level 2", cmd(grpD1Wizard(2), evoker, fireBolt, d20.AttackRoll, d20.Failure, 8), character.PotentCantripRejection{Kind: character.PotentCantripFeatureNotAvailable}},
		{"an ability check", cmd(grpD1Wizard(3), evoker, fireBolt, d20.AbilityCheck, d20.Failure, 8), character.PotentCantripRejection{Kind: character.PotentCantripUnsupportedTestKind, TestKind: d20.AbilityCheck}},
		{"a level 1 spell", cmd(grpD1Wizard(3), evoker, grpD1Facts(t, src, "burning_hands", true), d20.SavingThrow, d20.Success, 8), character.PotentCantripRejection{Kind: character.PotentCantripNotACantrip}},
		{"a cantrip that deals no damage", cmd(grpD1Wizard(3), evoker, grpD1Facts(t, src, "prestidigitation", false), d20.AttackRoll, d20.Failure, 8), character.PotentCantripRejection{Kind: character.PotentCantripSpellDealsNoDamage}},
	} {
		rej := grpD1As[character.ResolvePotentCantripRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ResolvePotentCantripNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), evoker, fireBolt, d20.AttackRoll, d20.Failure, 8))))
	for name, bad := range map[string]character.ResolvePotentCantripCommand{
		"negative damage":        cmd(grpD1Wizard(3), evoker, fireBolt, d20.AttackRoll, d20.Failure, -1),
		"an undefined test kind": cmd(grpD1Wizard(3), evoker, fireBolt, d20.Kind(0), d20.Failure, 8),
		"spell facts with no id": cmd(grpD1Wizard(3), evoker, character.WizardSpellFacts{}, d20.AttackRoll, d20.Failure, 8),
	} {
		if _, err := uc.Run(ctx, bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-007, FR-008, FR-015 - from level 6 an Evoker protects up to 1 plus the cast level of the visible creatures an Evocation spell affects, never itself; no subclass, level 5, a spell of another school, the caster, a creature that cannot be sculpted and too many creatures are returned rejections (SRD 5.2.1, "Level 6: Sculpt Spells", p. 82).
// Path: happy, then unhappy (boundary, invalid input).
func TestGrpD1EvokerSculptsAFireball(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	evoker := grpD1Evoker(t, src)
	uc := character.NewResolveSculptSpells()
	fireball := grpD1Facts(t, src, "fireball", true)
	ally := func(n string) character.SculptCandidate {
		return character.SculptCandidate{ID: character.MustIdentifier(n), Visible: true, Affected: true}
	}
	allies := []character.SculptCandidate{ally("a1"), ally("a2"), ally("a3"), ally("a4")}
	cmd := func(levels character.ClassLevels, sub content.SubclassDefinition, spell character.WizardSpellFacts, castLevel int, chosen ...character.SculptCandidate) character.ResolveSculptSpellsCommand {
		return character.ResolveSculptSpellsCommand{ClassLevels: levels, Subclass: sub, Spell: spell, CastLevel: castLevel, Chosen: chosen}
	}

	sculpted := grpD1As[character.SculptedCreatures](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(6), evoker, fireball, 3, allies...))))
	if len(sculpted.Creatures()) != 4 || sculpted.Allowed() != 4 {
		t.Errorf("a level 3 slot protects %d of %d allowed, want 4 of 4", len(sculpted.Creatures()), sculpted.Allowed())
	}
	none := grpD1As[character.SculptedCreatures](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(6), evoker, fireball, 3))))
	if len(none.Creatures()) != 0 || none.Allowed() != 4 {
		t.Errorf("choosing nobody = %d creatures, %d allowed; want none and 4", len(none.Creatures()), none.Allowed())
	}
	hidden := character.SculptCandidate{ID: character.MustIdentifier("hidden"), Affected: true}
	unaffected := character.SculptCandidate{ID: character.MustIdentifier("far"), Visible: true}
	self := character.SculptCandidate{ID: character.MustIdentifier("self"), Visible: true, Affected: true, IsCaster: true}
	for _, c := range []struct {
		name string
		cmd  character.ResolveSculptSpellsCommand
		want character.SculptRejection
	}{
		{"no subclass", cmd(grpD1Wizard(6), content.SubclassDefinition{}, fireball, 3, ally("a1")), character.SculptRejection{Kind: character.SculptNotEvoker}},
		{"level 5", cmd(grpD1Wizard(5), evoker, fireball, 3, ally("a1")), character.SculptRejection{Kind: character.SculptFeatureNotAvailable}},
		{"a spell of another school", cmd(grpD1Wizard(6), evoker, grpD1Facts(t, src, "dispel_magic", false), 3, ally("a1")), character.SculptRejection{Kind: character.SculptNotAnEvocationSpell}},
		{"the caster", cmd(grpD1Wizard(6), evoker, fireball, 3, ally("a1"), self), character.SculptRejection{Kind: character.SculptCaster, Creature: self.ID}},
		{"a creature the caster cannot see", cmd(grpD1Wizard(6), evoker, fireball, 3, hidden), character.SculptRejection{Kind: character.SculptCreatureNotEligible, Creature: hidden.ID}},
		{"a creature the spell does not affect", cmd(grpD1Wizard(6), evoker, fireball, 3, unaffected), character.SculptRejection{Kind: character.SculptCreatureNotEligible, Creature: unaffected.ID}},
		{"five creatures for a level 3 slot", cmd(grpD1Wizard(6), evoker, fireball, 3, append(allies, ally("a5"))...), character.SculptRejection{Kind: character.SculptTooManyCreatures, Allowed: 4}},
	} {
		rej := grpD1As[character.ResolveSculptSpellsRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		grpD1Equal(t, c.name+" rejection", rej.Rejection, c.want)
	}
	grpD1As[character.ResolveSculptSpellsNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), evoker, fireball, 3, ally("a1")))))
	for name, bad := range map[string]character.ResolveSculptSpellsCommand{
		"a cast level below the spell's": cmd(grpD1Wizard(6), evoker, fireball, 2, ally("a1")),
		"a cast level above nine":        cmd(grpD1Wizard(6), evoker, fireball, 10, ally("a1")),
		"a creature named twice":         cmd(grpD1Wizard(6), evoker, fireball, 3, ally("a1"), ally("a1")),
		"a creature with no id":          cmd(grpD1Wizard(6), evoker, fireball, 3, character.SculptCandidate{Visible: true, Affected: true}),
		"a cantrip cast above level 0":   cmd(grpD1Wizard(6), evoker, grpD1Facts(t, src, "fire_bolt", true), 1, ally("a1")),
	} {
		if _, err := uc.Run(ctx, bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-009, FR-010, FR-016 - from level 10 the Evoker may add its Intelligence modifier to one damage roll of an Evocation Wizard spell, once per cast; declining leaves the cast state alone, and no subclass, level 9, a spell off the Wizard list, a spell of another school and a second application are returned rejections (SRD 5.2.1, "Level 10: Empowered Evocation", p. 83).
// Path: happy, then unhappy (boundary, state errors).
func TestGrpD1EvokerEmpowersAFireball(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	evoker := grpD1Evoker(t, src)
	uc := character.NewResolveEmpoweredEvocation()
	fireball := grpD1Facts(t, src, "fireball", true)
	intelligence := ability.MustScore(20).Modifier()
	cmd := func(levels character.ClassLevels, sub content.SubclassDefinition, spell character.WizardSpellFacts, applies bool, state character.EmpoweredCastState) character.ResolveEmpoweredEvocationCommand {
		return character.ResolveEmpoweredEvocationCommand{ClassLevels: levels, Subclass: sub, Spell: spell, Intelligence: intelligence, Applies: applies, State: state}
	}
	fresh := character.NewEmpoweredCastState(false)

	roll := grpD1As[character.EmpoweredRoll](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(10), evoker, fireball, true, fresh))))
	if mod, applied := roll.Modifier(); !applied || mod.Value() != 5 || !roll.State().DamageRollEmpowered() {
		t.Errorf("Empowered Evocation = %+d (%v), empowered %v; want +5 applied and the cast empowered", mod.Value(), applied, roll.State().DamageRollEmpowered())
	}
	if fresh.DamageRollEmpowered() {
		t.Error("the input cast state must not change")
	}
	declined := grpD1As[character.EmpoweredRoll](t, grpD1R(uc.Run(ctx, cmd(grpD1Wizard(10), evoker, fireball, false, fresh))))
	if _, applied := declined.Modifier(); applied || declined.State().DamageRollEmpowered() {
		t.Errorf("declining = applied %v, empowered %v; want neither", applied, declined.State().DamageRollEmpowered())
	}
	for _, c := range []struct {
		name string
		cmd  character.ResolveEmpoweredEvocationCommand
		want character.EmpoweredRejectionKind
	}{
		{"no subclass", cmd(grpD1Wizard(10), content.SubclassDefinition{}, fireball, true, fresh), character.EmpoweredNotEvoker},
		{"level 9", cmd(grpD1Wizard(9), evoker, fireball, true, fresh), character.EmpoweredFeatureNotAvailable},
		{"a paladin spell", cmd(grpD1Wizard(10), evoker, grpD1Facts(t, src, "divine_smite", true), true, fresh), character.EmpoweredNotAWizardSpell},
		{"a spell of another school", cmd(grpD1Wizard(10), evoker, grpD1Facts(t, src, "dispel_magic", false), true, fresh), character.EmpoweredNotAnEvocationSpell},
		{"a second application", cmd(grpD1Wizard(10), evoker, fireball, true, roll.State()), character.EmpoweredAlreadyEmpowered},
	} {
		rej := grpD1As[character.ResolveEmpoweredEvocationRejected](t, grpD1R(uc.Run(ctx, c.cmd)))
		if rej.Rejection.Kind != c.want {
			t.Errorf("%s: kind = %v, want %v", c.name, rej.Rejection.Kind, c.want)
		}
	}
	grpD1As[character.ResolveEmpoweredEvocationNotAWizard](t, grpD1R(uc.Run(ctx, cmd(grpD1Fighter(), evoker, fireball, true, fresh))))
	if _, err := uc.Run(ctx, cmd(grpD1Wizard(10), evoker, character.WizardSpellFacts{}, true, fresh)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("facts with no id: error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-013, FR-018 and class-wizard.md FR-094 - the second Overchannel use before a Long Rest rolls its Necrotic damage, and a roller that fails there is the dice port's *srderr.PortError with no partial result and the use count unchanged.
// Path: unhappy (collaborator failure).
func TestGrpD1EvokerOverchannelSurvivesARollerFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	src := data.MustBundledSource()
	down := errors.New("dice backend down")
	broken := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, down })
	use := mustOK(character.NewUseOverchannel(broken))
	cast := character.OverchannelRequest{Spell: grpD1Facts(t, src, "fireball", true), Source: character.CastSourceSlot, SlotLevel: 3}
	cmd := character.UseOverchannelCommand{CharacterID: character.MustIdentifier("d1-overchannel"), ClassLevels: grpD1Wizard(14), Subclass: grpD1Evoker(t, src), Request: cast}

	first := grpD1As[character.OverchannelApplied](t, grpD1R(use.Run(ctx, cmd)))
	if _, rolled := first.Necrotic(); rolled || first.Use() != 1 {
		t.Fatalf("first use = %d, rolled %v; want use 1 and no roll", first.Use(), rolled)
	}
	cmd.Resources = first.Resources()
	res, err := use.Run(ctx, cmd)
	if res != nil {
		t.Errorf("result = %#v, want none when the roller fails", res)
	}
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, down) {
		t.Errorf("error = %v, want a dice *srderr.PortError wrapping the roller failure", err)
	}
	if cmd.Resources.OverchannelUses() != 1 {
		t.Errorf("uses = %d, want the input resources unchanged at 1", cmd.Resources.OverchannelUses())
	}
}
