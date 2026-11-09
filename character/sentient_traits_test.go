package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-090 and table "Sentient ability roll" - 4d6 with the lowest die dropped).
// Path: per case.
func TestSentientAbilityRollFromDice(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Abilities", p. 207`
	cases := []struct {
		name, requirement, path string
		faces                   []int
		want                    int
	}{
		{"all ones", p, "unhappy (boundary)", []int{1, 1, 1, 1}, 3},
		{"all sixes", p, "unhappy (boundary)", []int{6, 6, 6, 6}, 18},
		{"lowest dropped", p, "happy", []int{2, 5, 3, 6}, 14},
		{"lowest tied", p, "unhappy (rule interaction)", []int{4, 4, 4, 1}, 12},
		{"unsorted", p, "happy", []int{6, 1, 5, 4}, 15},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := must(dice.Roll(dicetest.NewFixedRoller(c.faces...), dice.MustExpression(4, dice.D6(), 0)))
			got, err := character.SentientAbilityRoll{}.FromDice(res)
			if err != nil || got.Value() != c.want {
				t.Errorf("= %v, %v, want %d [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	three := must(dice.Roll(dicetest.NewFixedRoller(1, 2, 3), dice.MustExpression(3, dice.D6(), 0)))
	if _, err := (character.SentientAbilityRoll{}).FromDice(three); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("three dice: %v", err)
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-095 and table "Aligned purpose").
// Path: per case.
func TestNewAlignedPurpose(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Special Purpose" (1 Aligned), p. 208`
	lg, ng, ln, n := content.AlignmentLawfulGood, content.AlignmentNeutralGood, content.AlignmentLawfulNeutral, content.AlignmentNeutral
	ce, ne, cn, le := content.AlignmentChaoticEvil, content.AlignmentNeutralEvil, content.AlignmentChaoticNeutral, content.AlignmentLawfulEvil
	cases := []struct {
		name, requirement, path string
		item, opposed           content.Alignment
		want                    character.AlignedPurposeReason
	}{
		{"lawful good", p, "happy", lg, ce, 0},
		{"neutral good", p, "happy", ng, ne, 0},
		{"lawful neutral", p, "happy", ln, cn, 0},
		{"chaotic good", p, "happy", content.AlignmentChaoticGood, content.AlignmentLawfulEvil, 0},
		{"neutral evil", p, "happy", ne, ng, 0},
		{"item is Neutral", p, "unhappy (invalid input)", n, ce, character.AlignedItemIsNeutral},
		{"not opposed", p, "unhappy (invalid input)", lg, le, character.AlignedNotDiametricallyOpposed},
		{"same alignment", p, "unhappy (invalid input)", lg, lg, character.AlignedNotDiametricallyOpposed},
		{"opposed is unaligned", p, "unhappy (invalid input)", lg, content.AlignmentUnaligned, character.AlignedNotDiametricallyOpposed},
		{"item is unaligned", p, "unhappy (invalid input)", content.AlignmentUnaligned, lg, character.AlignedNotDiametricallyOpposed},
		{"opposed is Neutral", p, "unhappy (invalid input)", lg, n, character.AlignedNotDiametricallyOpposed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewAlignedPurpose(c.item, c.opposed)
			if c.want == 0 {
				if err != nil || got.Opposed() != c.opposed || got.Kind() != content.SpecialPurposeAligned {
					t.Errorf("= %+v, %v [%s]", got, err, c.requirement)
				}
				return
			}
			var rej *character.AlignedPurposeRejected
			if !errors.As(err, &rej) || rej.Reason != c.want || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.AlignedPurposeRejected{Reason: c.want}) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
		})
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("MustAlignedPurpose did not panic")
			}
		}()
		character.MustAlignedPurpose(n, ce)
	}()
}

// Requirement: doc/specs/magic-item-rules.md FR-096 - a Bane purpose needs a creature type; Protector and Templar need a non-blank target.
// Path: unhappy (invalid input).
func TestSpecialPurposeConstructors(t *testing.T) {
	t.Parallel()
	bane := must(character.NewBanePurpose(content.CreatureTypeUndead))
	target := must(character.NewDescribedTarget("the realm"))
	prot, templ := must(character.NewProtectorPurpose(target)), must(character.NewTemplarPurpose(target))
	if bane.CreatureType() != content.CreatureTypeUndead || prot.Target() != target || templ.Target() != target {
		t.Error("accessors")
	}
	kinds := map[character.SpecialPurpose]content.SpecialPurposeKind{
		bane: content.SpecialPurposeBane, prot: content.SpecialPurposeProtector, templ: content.SpecialPurposeTemplar,
		character.CreatorSeeker{}: content.SpecialPurposeCreatorSeeker, character.DestinySeeker{}: content.SpecialPurposeDestinySeeker,
		character.Destroyer{}: content.SpecialPurposeDestroyer, character.GlorySeeker{}: content.SpecialPurposeGlorySeeker,
		character.LoreSeeker{}: content.SpecialPurposeLoreSeeker, character.SoulmateSeeker{}: content.SpecialPurposeSoulmateSeeker,
	}
	for p, k := range kinds {
		if p.Kind() != k {
			t.Errorf("%#v kind = %v, want %v", p, p.Kind(), k)
		}
	}
	for name, run := range map[string]func() error{
		"bane without a type": func() error { _, e := character.NewBanePurpose(0); return e },
		"blank target":        func() error { _, e := character.NewDescribedTarget("  "); return e },
		"protector, blank":    func() error { _, e := character.NewProtectorPurpose(""); return e },
		"templar, blank":      func() error { _, e := character.NewTemplarPurpose(""); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, ok := character.NoSpecialPurpose().Purpose(); ok {
		t.Error("NoSpecialPurpose has a purpose")
	}
	if p, ok := character.SelectPurpose(bane).Purpose(); !ok || p != bane {
		t.Error("SelectPurpose")
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-088, FR-089, FR-091, FR-097 - sentient traits; potion and scroll are never
// sentient, every other category is; the alignment is never unaligned; an Aligned purpose must pass NewAlignedPurpose).
// Path: per case.
func TestNewSentientTraits(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Sentient Magic Items", p. 207; "Sentient Magic Item Traits", pp. 207-208`
	sc := func(n int) ability.Score { return score(t, n) }
	senses := must(content.SentientSensesForRoll(1))
	ok := func(category content.MagicItemCategory, alignment content.Alignment, purpose character.SpecialPurposeSelection) (character.SentientTraits, error) {
		return character.NewSentientTraits(category, sc(10), sc(12), sc(14), alignment, content.SentientTransmitsEmotion, senses, purpose)
	}
	for _, cat := range []content.MagicItemCategory{content.MagicItemArmor, content.MagicItemRing, content.MagicItemRod, content.MagicItemStaff, content.MagicItemWand, content.MagicItemWeapon, content.MagicItemWondrousItem} {
		if err := character.AttachSentientTraits(cat); err != nil {
			t.Errorf("category %v: %v", cat, err)
		}
		if _, err := ok(cat, content.AlignmentLawfulGood, character.NoSpecialPurpose()); err != nil {
			t.Errorf("category %v: %v", cat, err)
		}
	}
	aligned := character.SelectPurpose(character.MustAlignedPurpose(content.AlignmentLawfulGood, content.AlignmentChaoticEvil))
	cases := []struct {
		name     string
		category content.MagicItemCategory
		a        content.Alignment
		purpose  character.SpecialPurposeSelection
		want     character.SentientTraitsReason
	}{
		{"potion", content.MagicItemPotion, content.AlignmentLawfulGood, character.NoSpecialPurpose(), character.SentientSingleUseItemNotSentient},
		{"scroll", content.MagicItemScroll, content.AlignmentLawfulGood, character.NoSpecialPurpose(), character.SentientSingleUseItemNotSentient},
		{"unaligned", content.MagicItemWeapon, content.AlignmentUnaligned, character.NoSpecialPurpose(), character.SentientUnalignedNotAllowed},
		{"unknown alignment", content.MagicItemWeapon, 0, character.NoSpecialPurpose(), character.SentientUnalignedNotAllowed},
		{"Aligned purpose, opposed to the item", content.MagicItemWeapon, content.AlignmentLawfulGood, aligned, 0},
		{"Aligned purpose, not opposed", content.MagicItemWeapon, content.AlignmentChaoticGood, aligned, character.SentientAlignedPurposeInvalid},
		{"Aligned purpose, Neutral item", content.MagicItemWeapon, content.AlignmentNeutral, aligned, character.SentientAlignedPurposeInvalid},
	}
	for _, c := range cases {
		got, err := ok(c.category, c.a, c.purpose)
		if c.want == 0 {
			if err != nil || got.Intelligence().Value() != 10 || got.Wisdom().Value() != 12 || got.Charisma().Value() != 14 || got.Alignment() != c.a || got.IsZero() ||
				got.Communication() != content.SentientTransmitsEmotion || got.Senses() != senses || got.SpecialPurpose() != c.purpose {
				t.Errorf("%s: %+v, %v", c.name, got, err)
			}
			continue
		}
		var rej *character.SentientTraitsRejected
		if !errors.As(err, &rej) || rej.Reason != c.want || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.SentientTraitsRejected{Reason: c.want}) || !got.IsZero() {
			t.Errorf("%s: err = %v, want %v [%s]", c.name, err, c.want, p)
		}
	}
	if _, err := character.NewSentientTraits(content.MagicItemWeapon, sc(10), sc(10), sc(10), content.AlignmentLawfulGood, 0, senses, character.NoSpecialPurpose()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unset communication: %v", err)
	}
	if _, err := character.NewSentientTraits(content.MagicItemWeapon, sc(10), sc(10), sc(10), content.AlignmentLawfulGood, content.SentientTransmitsEmotion, content.SentientSenses{}, character.NoSpecialPurpose()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unset senses: %v", err)
	}
	for r := character.SentientSingleUseItemNotSentient; r <= character.SentientAlignedPurposeInvalid; r++ {
		if r.String() == "unknown" {
			t.Errorf("reason %d", r)
		}
	}
	if character.SentientTraitsReason(0).String() != "unknown" {
		t.Error("String")
	}
}
