package scenarios_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The scenarios of equipment-rules.md run through the public packages only. The
// module bundles no equipment yet (Phase 5), so the entries are synthetic: made-up
// ids with the numbers of the SRD 5.2.1 Armor table, p. 92.

func grpD3Must[T any](v T, err error) T {
	if err != nil {
		panic(err) // a broken fixture, not a failure of the code under test
	}
	return v
}

func grpD3GP(n int) content.Cost { return content.MustCost(n, content.CoinGP, 0) }

func grpD3CP(n int) content.Cost { return content.MustCost(n, content.CoinCP, 0) }

func grpD3Lbs(p float64) content.Weight { return content.MustWeight(p, "") }

func grpD3Price(c content.Cost) content.FixedCost { return content.MustFixedCost(c) }

func grpD3Armor(id string, cat content.ArmorCategory, rule content.ArmorClassRule, strength int, stealth bool, pounds float64, cost content.Cost) content.Armor {
	return grpD3Must(content.NewArmor(content.ItemID(id), id, content.SourceCitation{}, cat, rule, strength, stealth, grpD3Lbs(pounds), grpD3Price(cost)))
}

func grpD3Full(base int) content.BaseRule {
	return grpD3Must(content.NewBaseRule(base, content.FullDexterity{}))
}

func grpD3Capped(base, maximum int) content.BaseRule {
	return grpD3Must(content.NewBaseRule(base, grpD3Must(content.NewCappedDexterity(maximum))))
}

func grpD3NoDex(base int) content.BaseRule {
	return grpD3Must(content.NewBaseRule(base, content.NoDexterity{}))
}

func grpD3Score(n int) ability.Score { return grpD3Must(ability.NewScore(n)) }

func grpD3WantInvalid(t *testing.T, err error, what string) {
	t.Helper()
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("%s: error = %v, want srderr.ErrInvalidInput", what, err)
	}
}

// A fighter shops for armor and swaps suits: the base Armor Class follows each
// suit's Dexterity treatment, a Strength below the suit's requirement costs 10
// feet of Speed, and Stealth Disadvantage appears once whatever is worn
// (SRD 5.2.1, "Armor", p. 92).
// Requirement: doc/specs/equipment-rules.md FR-001 to FR-004, FR-006 - Light adds the whole modifier, Medium caps it at 2, Heavy ignores it, a negative modifier is never raised.
// Path: happy; unhappy (boundary, rule interaction).
func TestGrpD3SwappingArmorChangesBaseArmorClass(t *testing.T) {
	t.Parallel()
	rules := character.ArmorRules{}
	leather, hide, chain, plate := grpD3Full(11), grpD3Capped(12, 2), grpD3NoDex(16), grpD3NoDex(18)
	cases := []struct {
		name          string
		dexScore      int
		rule          content.BaseRule
		listed, dexOf int
	}{
		{"nimble in Leather", 16, leather, 11, 3},
		{"nimble in Hide, capped at +2", 16, hide, 12, 2},
		{"nimble in Chain Mail, Dexterity ignored", 16, chain, 16, 0},
		{"nimble in Plate, Dexterity ignored", 16, plate, 18, 0},
		{"clumsy in Leather, penalty applies", 6, leather, 11, -2},
		{"clumsy in Hide, a negative modifier is not raised", 6, hide, 12, -2},
		{"clumsy in Plate, still the listed base", 6, plate, 18, 0},
		{"average Dexterity in Hide gives its +0", 10, hide, 12, 0},
		{"a +2 modifier in Hide is under the cap", 14, hide, 12, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := rules.BaseArmorClass(c.rule, grpD3Score(c.dexScore).Modifier())
			if got.ListedBase() != c.listed || got.DexterityContribution() != c.dexOf || got.Total() != c.listed+c.dexOf {
				t.Errorf("base = %d + %d = %d, want %d + %d", got.ListedBase(), got.DexterityContribution(), got.Total(), c.listed, c.dexOf)
			}
		})
	}
	// A capped rule with a synthetic maximum of 0 gives nothing for a positive modifier.
	if got := rules.BaseArmorClass(grpD3Capped(10, 0), grpD3Score(18).Modifier()); got.Total() != 10 {
		t.Errorf("cap 0 with +4 = %d, want 10", got.Total())
	}
	// Equal parts compare equal; the zero rule gives the zero base.
	a, b := rules.BaseArmorClass(hide, grpD3Score(16).Modifier()), rules.BaseArmorClass(hide, grpD3Score(14).Modifier())
	if a != b {
		t.Errorf("Hide at +3 %v and +2 %v should be equal (both capped at +2)", a, b)
	}
	if got := rules.BaseArmorClass(content.BaseRule{}, grpD3Score(16).Modifier()); got != (character.ArmorBase{}) {
		t.Errorf("zero rule = %v, want the zero base", got)
	}
}

// Requirement: doc/specs/equipment-rules.md FR-007 to FR-010, FR-079 - a Strength below the requirement costs 10 feet, equal or above costs none, no requirement never costs; Stealth Disadvantage is one entry however many suits carry it, and none otherwise.
// Path: unhappy (boundary, empty and minimal, rule interaction, immutability).
func TestGrpD3StrengthAndStealthOfWornArmor(t *testing.T) {
	t.Parallel()
	rules := character.ArmorRules{}
	leather := grpD3Armor("leather", content.ArmorLight, grpD3Full(11), 0, false, 10, grpD3GP(10))
	chain := grpD3Armor("chain_mail", content.ArmorHeavy, grpD3NoDex(16), 13, true, 55, grpD3GP(75))
	plate := grpD3Armor("plate", content.ArmorHeavy, grpD3NoDex(18), 15, true, 65, grpD3GP(1500))
	shield := grpD3Armor("shield", content.ArmorShield, grpD3Must(content.NewBonusRule(2)), 0, false, 6, grpD3GP(10))

	for _, c := range []struct {
		armor content.Armor
		str   int
		want  int
	}{
		{chain, 12, 10}, {chain, 13, 0}, {chain, 14, 0},
		{plate, 8, 10}, {plate, 14, 10}, {plate, 15, 0}, {plate, 30, 0},
		{leather, 1, 0}, {leather, 30, 0}, {shield, 1, 0},
	} {
		if got := rules.SpeedPenaltyFeet(c.armor, grpD3Score(c.str)); got != c.want {
			t.Errorf("%s at Strength %d: penalty %d, want %d", c.armor.ID(), c.str, got, c.want)
		}
	}

	if got := rules.StealthCircumstances([]content.Armor{leather, shield}); got != nil {
		t.Errorf("Leather and a Shield give %v, want nil", got)
	}
	if got := rules.StealthCircumstances(nil); got != nil {
		t.Errorf("nothing worn gives %v, want nil", got)
	}
	one := rules.StealthCircumstances([]content.Armor{chain})
	if len(one) != 1 || one[0] != (modifier.Disadvantage{}) {
		t.Fatalf("Chain Mail gives %v, want one Disadvantage", one)
	}
	if many := rules.StealthCircumstances([]content.Armor{chain, plate}); len(many) != 1 {
		t.Errorf("two flagged suits give %d circumstances, want 1 (Disadvantage does not stack)", len(many))
	}
	one[0] = modifier.Advantage{} // the caller owns its copy
	if again := rules.StealthCircumstances([]content.Armor{chain}); again[0] != (modifier.Disadvantage{}) {
		t.Errorf("a second read = %v after mutating the first", again)
	}
}

// A character who has not trained with Heavy armor puts on Plate: the loss of
// training shows as Disadvantage on Strength and Dexterity D20 Tests only, an
// untrained Shield only loses its bonus, and Disadvantage meets other Advantage
// through the kernel (SRD 5.2.1, "Armor Training", p. 177).
// Requirement: doc/specs/equipment-rules.md FR-071 to FR-078 - Assess lists the untrained categories; Disadvantage for Strength or Dexterity only; a lone Shield gives none; two suits are invalid; monsters are trained with their stat block.
// Path: happy; unhappy (rule interaction, invalid input, immutability).
func TestGrpD3ArmorTrainingGivesDisadvantageOnlyForStrengthAndDexterity(t *testing.T) {
	t.Parallel()
	rules := character.ArmorTrainingRules{}
	light, medium, heavy, shield := content.ArmorLight, content.ArmorMedium, content.ArmorHeavy, content.ArmorShield
	trained := []content.ArmorCategory{light, medium, shield}

	fx, err := rules.Assess([]content.ArmorCategory{heavy, shield}, trained)
	if err != nil {
		t.Fatal(err)
	}
	if got := fx.Untrained(); !slices.Equal(got, []content.ArmorCategory{heavy}) {
		t.Fatalf("Untrained = %v, want [heavy]", got)
	}
	if !fx.WearsUntrainedBodyArmor() {
		t.Error("untrained Heavy armor must be reported as untrained body armor")
	}
	for _, c := range []struct {
		abilities []ability.Ability
		want      int
	}{
		{[]ability.Ability{ability.Strength}, 1},
		{[]ability.Ability{ability.Dexterity}, 1},
		{[]ability.Ability{ability.Strength, ability.Dexterity}, 1},
		{[]ability.Ability{ability.Constitution}, 0},
		{[]ability.Ability{ability.Charisma, ability.Wisdom}, 0},
		{nil, 0},
	} {
		got := fx.RollCircumstancesFor(c.abilities)
		if len(got) != c.want {
			t.Errorf("circumstances for %v = %v, want %d element(s)", c.abilities, got, c.want)
		}
	}

	// The Disadvantage joins another source's Advantage through modifier.Resolve and cancels it.
	dis := fx.RollCircumstancesFor([]ability.Ability{ability.Dexterity})
	if got := modifier.Resolve(dis...); got != modifier.WithDisadvantage {
		t.Errorf("Disadvantage alone = %v", got)
	}
	if got := modifier.Resolve(append(dis, modifier.Advantage{})...); got != modifier.Normal {
		t.Errorf("Disadvantage with one Advantage = %v, want normal", got)
	}

	// A Shield used untrained loses its bonus and nothing else.
	shieldOnly, err := rules.Assess([]content.ArmorCategory{shield}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(shieldOnly.Untrained(), []content.ArmorCategory{shield}) || shieldOnly.WearsUntrainedBodyArmor() {
		t.Errorf("untrained Shield = %v, body armor %v", shieldOnly.Untrained(), shieldOnly.WearsUntrainedBodyArmor())
	}
	for _, a := range []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution} {
		if got := shieldOnly.RollCircumstancesFor([]ability.Ability{a}); len(got) != 0 {
			t.Errorf("untrained Shield on %v gives %v, want none", a, got)
		}
	}

	// Nothing worn is valid; a trained wearer has nothing untrained.
	empty, err := rules.Assess(nil, nil)
	if err != nil || len(empty.Untrained()) != 0 {
		t.Errorf("nothing worn = %v, %v", empty.Untrained(), err)
	}
	ready, err := rules.Assess([]content.ArmorCategory{medium}, trained)
	if err != nil || len(ready.Untrained()) != 0 || len(ready.RollCircumstancesFor([]ability.Ability{ability.Strength})) != 0 {
		t.Errorf("trained Medium = %v, %v", ready.Untrained(), err)
	}

	// Two suits at once is not a legal outfit.
	_, err = rules.Assess([]content.ArmorCategory{light, heavy}, trained)
	grpD3WantInvalid(t, err, "Light and Heavy together")

	// A monster is trained with the armor in its stat block.
	statBlock := []content.ArmorCategory{heavy, shield}
	mfx, err := rules.Assess(statBlock, rules.MonsterTraining(statBlock))
	if err != nil || len(mfx.Untrained()) != 0 || len(mfx.RollCircumstancesFor([]ability.Ability{ability.Strength, ability.Dexterity})) != 0 {
		t.Errorf("monster in its own armor = %v, %v", mfx.Untrained(), err)
	}

	// The returned slice is the caller's copy and equal inputs give equal effects.
	got := fx.Untrained()
	got[0] = shield
	if !slices.Equal(fx.Untrained(), []content.ArmorCategory{heavy}) {
		t.Error("mutating Untrained() changed the effects")
	}
	again, _ := rules.Assess([]content.ArmorCategory{heavy, shield}, trained)
	if !fx.Equal(again) {
		t.Error("equal inputs gave unequal effects")
	}
}

// A Goliath's Powerful Build is the host's concern: it passes the larger size,
// and the capacity follows the row it is given. A load is then classified as
// within Carry, a Speed-limited drag, or too heavy to move (SRD 5.2.1,
// "Carrying Capacity", p. 178).
// Requirement: doc/specs/equipment-rules.md FR-011 to FR-020, FR-084 - Carry and Drag/Lift/Push from size and Strength; Small equals Medium; Tiny is exact in half pounds; Assess classifies a load; a negative load is invalid.
// Path: happy; unhappy (boundary, rule interaction, invalid input).
func TestGrpD3CarryingCapacityFollowsSizeAndStrength(t *testing.T) {
	t.Parallel()
	str15 := grpD3Score(15)

	medium := character.NewCarryingCapacity(content.SizeMedium, str15)
	if medium.CarryHalfPounds() != 450 || medium.DragLiftPushPounds() != 450 {
		t.Errorf("Medium Strength 15: carry %d half lb, drag %d lb; want 450 (225 lb) and 450", medium.CarryHalfPounds(), medium.DragLiftPushPounds())
	}
	// The Goliath row: one size larger than the current size.
	large := character.NewCarryingCapacity(content.SizeLarge, str15)
	if large.CarryHalfPounds() != 900 || large.DragLiftPushPounds() != 900 {
		t.Errorf("Large Strength 15: carry %d half lb, drag %d lb; want 900 and 900", large.CarryHalfPounds(), large.DragLiftPushPounds())
	}
	huge := character.NewCarryingCapacity(content.SizeHuge, str15)
	if huge.CarryHalfPounds() != 1800 || huge.DragLiftPushPounds() != 1800 {
		t.Errorf("Huge Strength 15: %d / %d, want 1800 / 1800", huge.CarryHalfPounds(), huge.DragLiftPushPounds())
	}
	if gar := character.NewCarryingCapacity(content.SizeGargantuan, str15); gar.DragLiftPushPounds() != 3600 {
		t.Errorf("Gargantuan Strength 15 drag = %d, want 3600", gar.DragLiftPushPounds())
	}
	for s := 1; s <= 30; s++ {
		if character.NewCarryingCapacity(content.SizeSmall, grpD3Score(s)) != character.NewCarryingCapacity(content.SizeMedium, grpD3Score(s)) {
			t.Errorf("Small and Medium differ at Strength %d", s)
		}
	}

	// Tiny: Strength x 7.5 lb, so an odd score ends in a half pound.
	tiny := character.NewCarryingCapacity(content.SizeTiny, grpD3Score(3))
	if tiny.CarryHalfPounds() != 45 || tiny.DragLiftPushPounds() != 45 {
		t.Errorf("Tiny Strength 3: carry %d half lb (22.5 lb), drag %d lb; want 45 and 45", tiny.CarryHalfPounds(), tiny.DragLiftPushPounds())
	}
	for load, want := range map[int]bool{22: true, 23: false, 0: true} {
		got, err := tiny.CanCarry(load)
		if err != nil || got != want {
			t.Errorf("Tiny CanCarry(%d) = %v, %v; want %v", load, got, err, want)
		}
	}

	// A load classified against the Medium row: 225 lb carry, 450 lb drag.
	for _, c := range []struct {
		load int
		want character.LoadAssessment
	}{
		{0, character.LoadWithinCarry{}}, {225, character.LoadWithinCarry{}},
		{226, character.LoadSpeedLimited{}}, {450, character.LoadSpeedLimited{}},
		{451, character.LoadBeyondDragLiftPush{}},
	} {
		got, err := medium.Assess(c.load)
		if err != nil || got != c.want {
			t.Errorf("Assess(%d) = %#v, %v; want %#v", c.load, got, err, c.want)
		}
	}
	limited, _ := medium.Assess(300)
	if l, ok := limited.(character.LoadSpeedLimited); !ok || l.LimitFeet() != 5 {
		t.Errorf("a dragged load limits Speed to %#v, want 5 feet", limited)
	}

	// An invalid size or score gives no capacity at all.
	if none := character.NewCarryingCapacity(0, str15); none.DragLiftPushPounds() != 0 {
		t.Errorf("invalid size drag = %d, want 0", none.DragLiftPushPounds())
	}
	if beyond, err := character.NewCarryingCapacity(0, str15).Assess(1); err != nil || beyond != (character.LoadBeyondDragLiftPush{}) {
		t.Errorf("one pound against zero capacity = %#v, %v", beyond, err)
	}

	_, err := medium.CanCarry(-1)
	grpD3WantInvalid(t, err, "CanCarry(-1)")
	_, err = medium.Assess(-1)
	grpD3WantInvalid(t, err, "Assess(-1)")
}

// A caravan: two draft horses pull a wagon, and the cargo plus the wagon must
// stay within five times their carrying capacity (SRD 5.2.1, "Mounts and
// Cargo", p. 100).
// Requirement: doc/specs/equipment-rules.md FR-021 to FR-025 - the limit is five times the animals' capacity, the vehicle counts, no animals means 0, negative weights are invalid, order does not matter.
// Path: happy; unhappy (boundary, empty and minimal, invalid input, immutability).
func TestGrpD3DrawnLoadOfACaravan(t *testing.T) {
	t.Parallel()
	horse := func(id string, capacity int) content.Mount {
		return grpD3Must(content.NewMount(content.ItemID(id), id, content.SourceCitation{}, capacity, grpD3Price(grpD3GP(50))))
	}
	team := []content.Mount{horse("draft_one", 540), horse("draft_two", 540)}
	d := character.DrawnLoad{}

	if got := d.LimitPounds(team); got != 5400 {
		t.Fatalf("limit = %d, want 5400", got)
	}
	for _, c := range []struct {
		wagon, cargo int
		want         bool
	}{
		{400, 5000, true}, {400, 5001, false}, {0, 5400, true}, {5400, 1, false},
	} {
		got, err := d.CanMove(team, c.wagon, c.cargo)
		if err != nil || got != c.want {
			t.Errorf("CanMove(wagon %d, cargo %d) = %v, %v; want %v", c.wagon, c.cargo, got, err, c.want)
		}
	}

	// A mixed team is the same whichever horse is listed first, and the list is untouched.
	mixed := []content.Mount{horse("pony", 225), horse("mule", 420)}
	reversed := []content.Mount{mixed[1], mixed[0]}
	if d.LimitPounds(mixed) != 3225 || d.LimitPounds(mixed) != d.LimitPounds(reversed) {
		t.Errorf("mixed team limits %d / %d, want 3225 both", d.LimitPounds(mixed), d.LimitPounds(reversed))
	}
	if mixed[0].ID() != "pony" {
		t.Error("LimitPounds reordered the caller's slice")
	}

	// Nobody pulls: only an empty wagon "moves".
	if ok, err := d.CanMove(nil, 0, 0); err != nil || !ok {
		t.Errorf("empty with no animals = %v, %v", ok, err)
	}
	if ok, err := d.CanMove(nil, 1, 0); err != nil || ok {
		t.Errorf("one pound with no animals = %v, %v", ok, err)
	}
	_, err := d.CanMove(team, -1, 0)
	grpD3WantInvalid(t, err, "negative vehicle")
	_, err = d.CanMove(team, 0, -1)
	grpD3WantInvalid(t, err, "negative cargo")
}

// A knight armors a warhorse: barding costs four times the armor and weighs
// twice as much, in the armor's own coin (SRD 5.2.1, "Barding", p. 100).
// Requirement: doc/specs/equipment-rules.md FR-026, FR-028 - barding is 4x cost and 2x weight for every Armor-table entry including the Shield; the printed coin is kept; invalid armor is rejected.
// Path: happy; unhappy (rule interaction, invalid input).
func TestGrpD3BardingForPlateAndShield(t *testing.T) {
	t.Parallel()
	plate := grpD3Armor("plate", content.ArmorHeavy, grpD3NoDex(18), 15, true, 65, grpD3GP(1500))
	b, err := character.DeriveBarding(plate)
	if err != nil {
		t.Fatal(err)
	}
	if b.Cost() != grpD3GP(6000) || b.Weight() != grpD3Lbs(130) {
		t.Errorf("Plate barding = %v, %v; want 6,000 GP and 130 lb", b.Cost(), b.Weight())
	}
	if b.Cost() == grpD3CP(600000) {
		t.Error("the printed coin must be kept (6,000 GP, not 600,000 CP)")
	}
	shield := grpD3Armor("shield", content.ArmorShield, grpD3Must(content.NewBonusRule(2)), 0, false, 6, grpD3GP(10))
	sb, err := character.DeriveBarding(shield)
	if err != nil || sb.Cost() != grpD3GP(40) || sb.Weight() != grpD3Lbs(12) {
		t.Errorf("Shield barding = %v, %v, %v", sb.Cost(), sb.Weight(), err)
	}
	again, _ := character.DeriveBarding(plate)
	if again != b {
		t.Error("equal armor gave unequal barding")
	}

	_, err = character.DeriveBarding(content.Armor{})
	grpD3WantInvalid(t, err, "the zero Armor")
	huge := grpD3Armor("huge", content.ArmorHeavy, grpD3NoDex(18), 15, true, 65, content.MustCost(content.MaxCostAmount, content.CoinPP, 0))
	_, err = character.DeriveBarding(huge)
	grpD3WantInvalid(t, err, "a price four times over the limit")
}
