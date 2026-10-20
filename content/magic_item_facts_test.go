package content_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var allMagicItemCategories = []content.MagicItemCategory{
	content.MagicItemArmor, content.MagicItemPotion, content.MagicItemRing, content.MagicItemRod, content.MagicItemScroll,
	content.MagicItemStaff, content.MagicItemWand, content.MagicItemWeapon, content.MagicItemWondrousItem,
}

var allItemRarities = []content.ItemRarity{
	content.RarityCommon, content.RarityUncommon, content.RarityRare, content.RarityVeryRare, content.RarityLegendary, content.RarityArtifact,
}

func wantRollError(t *testing.T, err error, roll, lo, hi int) {
	t.Helper()
	var re *content.RollRangeError
	if !errors.As(err, &re) || *re != (content.RollRangeError{Roll: roll, Min: lo, Max: hi}) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("roll %d: error = %v, want *RollRangeError{%d, %d, %d} wrapping ErrInvalidInput", roll, err, roll, lo, hi)
	}
}

// Requirement: miSpec FR-078, FR-079 - the category facts of table "Category
// facts", one row per category; a value that is not a category gets the zero
// facts; ActivationTime has five members and the described-elsewhere one is
// not tied to a category.
// Path: per case.
func TestMagicItemCategoryFacts(t *testing.T) {
	t.Parallel()
	type row struct {
		consumable, focus, quarterstaff, worn bool
		time                                  content.ActivationTime
	}
	want := map[content.MagicItemCategory]row{
		content.MagicItemArmor:        {false, false, false, true, content.ActivationMagicAction},
		content.MagicItemPotion:       {true, false, false, false, content.ActivationBonusAction},
		content.MagicItemRing:         {false, false, false, true, content.ActivationMagicAction},
		content.MagicItemRod:          {false, true, false, false, content.ActivationMagicAction},
		content.MagicItemScroll:       {true, false, false, false, content.ActivationMagicAction},
		content.MagicItemStaff:        {false, true, true, false, content.ActivationMagicAction},
		content.MagicItemWand:         {false, true, false, false, content.ActivationMagicAction},
		content.MagicItemWeapon:       {false, false, false, false, content.ActivationMagicAction},
		content.MagicItemWondrousItem: {false, false, false, false, content.ActivationMagicAction},
	}
	for _, c := range allMagicItemCategories {
		f := content.MagicItemCategoryFacts{}.Of(c)
		got := row{f.Consumable(), f.ArcaneFocus(), f.NonmagicalQuarterstaff(), f.MustBeWorn(), f.ActivationTime()}
		if got != want[c] {
			t.Errorf("%v facts = %+v, want %+v [FR-078]", c, got, want[c])
		}
	}
	if z := (content.MagicItemCategoryFacts{}).Of(0); z != (content.MagicItemCategoryFacts{}) || z.ActivationTime().Valid() {
		t.Errorf("facts of a non-category = %+v [FR-078]", z)
	}
	if z := (content.MagicItemCategoryFacts{}).Of(10); z != (content.MagicItemCategoryFacts{}) {
		t.Errorf("facts of category 10 = %+v [FR-078]", z)
	}
	times := []content.ActivationTime{
		content.ActivationMagicAction, content.ActivationBonusAction, content.ActivationAction, content.ActivationReaction, content.ActivationDescribedElsewhere,
	}
	for i, a := range times {
		if int(a) != i+1 || !a.Valid() || a.String() == "unknown" {
			t.Errorf("activation time %d = %q valid=%v [FR-079]", a, a, a.Valid())
		}
	}
	if content.ActivationTime(0).Valid() || content.ActivationTime(6).Valid() || content.ActivationTime(6).String() != "unknown" {
		t.Error("an undefined activation time is valid [FR-079]")
	}
	for _, c := range allMagicItemCategories {
		f, err := content.NewMagicItemFacts("item_x", content.MagicItemFactsFields{ActivationTime: content.OverriddenActivationTime(content.ActivationDescribedElsewhere)})
		if v, set := f.ActivationTime().Value(); err != nil || !set || v != content.ActivationDescribedElsewhere {
			t.Errorf("%v: described elsewhere rejected: %v [FR-079]", c, err)
		}
	}
}

// Requirement: miSpec FR-080 to FR-082 - the Potion Miscibility table: eight
// bands that cover 1 to 100 exactly once, the typed facts of every effect,
// and a roll outside 1 to 100 is a *RollRangeError.
// Path: per case.
func TestPotionMiscibilityTable(t *testing.T) {
	t.Parallel()
	table := content.PotionMiscibilityTable{}
	bands := table.Bands()
	wantEdges := [][2]int{{1, 1}, {2, 8}, {9, 15}, {16, 25}, {26, 35}, {36, 90}, {91, 99}, {100, 100}}
	if len(bands) != 8 {
		t.Fatalf("%d bands, want 8 [FR-080]", len(bands))
	}
	wantTypes := []reflect.Type{
		reflect.TypeOf(content.PotionsExplode{}), reflect.TypeOf(content.PotionsBecomeIngestedPoison{}), reflect.TypeOf(content.BothPotionsLoseEffects{}),
		reflect.TypeOf(content.OnePotionLosesEffect{}), reflect.TypeOf(content.PotionEffectsHalved{}), reflect.TypeOf(content.PotionsWorkNormally{}),
		reflect.TypeOf(content.OnePotionEffectsDoubled{}), reflect.TypeOf(content.OnePotionPermanent{}),
	}
	for i, b := range bands {
		if b.MinRoll() != wantEdges[i][0] || b.MaxRoll() != wantEdges[i][1] || reflect.TypeOf(b.Effect()) != wantTypes[i] {
			t.Errorf("band %d = %d..%d %T, want %v %v [FR-080]", i, b.MinRoll(), b.MaxRoll(), b.Effect(), wantEdges[i], wantTypes[i])
		}
	}
	bands[0] = bands[1]
	if table.Bands()[0].MaxRoll() != 1 {
		t.Error("Bands returned the internal slice [FR-008]")
	}
	for roll := 1; roll <= 100; roll++ {
		var in []int
		for i, e := range wantEdges {
			if roll >= e[0] && roll <= e[1] {
				in = append(in, i)
			}
		}
		eff, err := table.ForRoll(roll)
		if len(in) != 1 || err != nil || reflect.TypeOf(eff) != wantTypes[in[0]] {
			t.Errorf("roll %d: bands %v, effect %T, err %v, want one band [FR-080, FR-082]", roll, in, eff, err)
		}
	}
	for _, roll := range []int{0, 101, -1} {
		if _, err := table.ForRoll(roll); err == nil {
			t.Errorf("ForRoll(%d) accepted [FR-082]", roll)
		} else {
			wantRollError(t, err, roll, 1, 100)
		}
		func() {
			defer func() {
				var re *content.RollRangeError
				if r := recover(); !errors.As(r.(error), &re) || re.Roll != roll {
					t.Errorf("MustForRoll(%d) panicked with %v [FR-082]", roll, r)
				}
			}()
			table.MustForRoll(roll)
		}()
	}
	if _, ok := table.MustForRoll(50).(content.PotionsWorkNormally); !ok {
		t.Error("MustForRoll(50) is not PotionsWorkNormally")
	}
	ex := content.PotionsExplode{}.Explosion()
	if ex.RadiusFeet() != 5 || ex.Shape() != content.ExplosionSphere || ex.Origin() != content.ExplosionOriginMixture ||
		ex.Damage() != dice.MustParseExpression("4d10") || ex.DamageType() != content.DamageForce {
		t.Errorf("explosion = %+v [FR-081]", ex)
	}
	type facts struct {
		both    bool
		choices []content.MixtureChoice
	}
	want := []facts{
		{true, nil}, {true, []content.MixtureChoice{content.MixtureChoicePoison}}, {true, nil},
		{false, []content.MixtureChoice{content.MixtureChoiceLosingPotion}}, {false, nil}, {false, nil},
		{false, []content.MixtureChoice{content.MixtureChoiceDoubledPotion}},
		{false, []content.MixtureChoice{content.MixtureChoiceSurvivingPotion, content.MixtureChoicePermanentEffect}},
	}
	for i, b := range table.Bands() {
		e := b.Effect()
		if e.BothPotionsLoseEffects() != want[i].both || !slices.Equal(e.HostChoices(), want[i].choices) {
			t.Errorf("band %d: both=%v choices=%v, want %+v [FR-081]", i, e.BothPotionsLoseEffects(), e.HostChoices(), want[i])
		}
	}
	if !(content.PotionEffectsHalved{}).PotionWithNothingToHalveLosesEffect() || !(content.OnePotionEffectsDoubled{}).WorkNormallyIfNothingToDouble() {
		t.Error("the halved and doubled flags are false [FR-081]")
	}
	for i, c := range []content.MixtureChoice{
		content.MixtureChoicePoison, content.MixtureChoiceLosingPotion, content.MixtureChoiceDoubledPotion,
		content.MixtureChoiceSurvivingPotion, content.MixtureChoicePermanentEffect,
	} {
		if int(c) != i+1 || !c.Valid() || c.String() == "unknown" {
			t.Errorf("mixture choice %d = %q [FR-081]", c, c)
		}
	}
	if content.MixtureChoice(0).Valid() || content.MixtureChoice(6).Valid() || content.MixtureChoice(6).String() != "unknown" {
		t.Error("an undefined mixture choice is valid [FR-081]")
	}
	if !content.ExplosionOriginMixture.Valid() || content.ExplosionOrigin(0).Valid() || content.ExplosionOrigin(2).Valid() ||
		content.ExplosionOriginMixture.String() != "mixture" || content.ExplosionOrigin(2).String() != "unknown" ||
		!content.ExplosionSphere.Valid() || content.ExplosionShape(0).Valid() || content.ExplosionShape(2).Valid() ||
		content.ExplosionSphere.String() != "Sphere" || content.ExplosionShape(2).String() != "unknown" {
		t.Error("explosion origin or shape vocabulary is wrong [FR-081]")
	}
}

// Requirement: miSpec FR-080 - a band needs minRoll not above maxRoll and an
// effect.
// Path: per case.
func TestNewPotionMiscibilityBand(t *testing.T) {
	t.Parallel()
	e := content.PotionsWorkNormally{}
	for _, c := range []struct {
		lo, hi int
		eff    content.PotionMiscibilityEffect
		want   content.BlockProblem
	}{
		{5, 4, e, content.BlockOutOfRange}, {0, 4, e, content.BlockOutOfRange}, {1, 101, e, content.BlockOutOfRange},
		{-1, 0, e, content.BlockOutOfRange}, {1, 4, nil, content.BlockMissing},
		{1, 4, (*content.PotionsWorkNormally)(nil), content.BlockInvalidMember}, {1, 4, &content.PotionsExplode{}, content.BlockInvalidMember},
	} {
		_, err := content.NewPotionMiscibilityBand(c.lo, c.hi, c.eff)
		wantBlockError(t, err, "PotionMiscibilityBand", c.want)
	}
	for _, c := range [][2]int{{1, 1}, {1, 100}, {100, 100}, {7, 8}} {
		b, err := content.NewPotionMiscibilityBand(c[0], c[1], e)
		if err != nil || b.MinRoll() != c[0] || b.MaxRoll() != c[1] || b.Effect() != content.PotionMiscibilityEffect(e) {
			t.Errorf("band %v = %+v, %v", c, b, err)
		}
	}
}

// Requirement: miSpec FR-083 to FR-085 - the rarity facts of table "Rarity
// facts": six rows in order, values in Copper Pieces, five crafting rows with
// workHours 8 times days and none for the Artifact.
// Path: per case.
func TestItemRarityFacts(t *testing.T) {
	t.Parallel()
	gold := func(n int64) content.PricedMagicItemValue {
		v, err := content.PricedMagicItemValueFromGold(n)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	type row struct {
		value, days, cost int64
	}
	want := []row{{100, 5, 50}, {400, 10, 200}, {4000, 50, 2000}, {40000, 125, 20000}, {200000, 250, 100000}}
	for i, r := range allItemRarities[:5] {
		f := content.ItemRarityFacts{}.Of(r)
		found, ok := f.Crafting().(content.MagicItemCraftingRowFound)
		if f.Value() != content.MagicItemValue(gold(want[i].value)) || !ok {
			t.Fatalf("%v facts = %+v [FR-083, FR-084]", r, f)
		}
		row := found.Row()
		if row.Rarity() != r || row.Days() != int(want[i].days) || row.WorkHours() != 8*int(want[i].days) || row.Cost() != gold(want[i].cost) {
			t.Errorf("%v crafting row = %+v [FR-085]", r, row)
		}
	}
	art := content.ItemRarityFacts{}.Of(content.RarityArtifact)
	if art.Value() != content.MagicItemValue(content.PricelessMagicItemValue{}) || art.Crafting() != content.MagicItemCraftingRowLookup(content.MagicItemCraftingRowAbsent{}) {
		t.Errorf("artifact facts = %+v [FR-084, FR-085]", art)
	}
	if z := (content.ItemRarityFacts{}).Of(0); z.Value() != nil || z.Crafting() != nil {
		t.Errorf("facts of a non-rarity = %+v [FR-083]", z)
	}
	if z := (content.ItemRarityFacts{}).Of(7); z.Value() != nil {
		t.Errorf("facts of rarity 7 = %+v [FR-083]", z)
	}
}

// Requirement: miSpec FR-084, FR-085 - the value and crafting-row
// constructors reject negative amounts and impossible rows.
// Path: per case.
func TestMagicItemValueConstructors(t *testing.T) {
	t.Parallel()
	if v, err := content.NewPricedMagicItemValue(0); err != nil || v.CopperPieces() != 0 {
		t.Errorf("NewPricedMagicItemValue(0) = %v, %v", v, err)
	}
	_, err := content.NewPricedMagicItemValue(-1)
	wantBlockError(t, err, "PricedMagicItemValue", content.BlockOutOfRange)
	if v, err := content.PricedMagicItemValueFromGold(3); err != nil || v.CopperPieces() != 300 {
		t.Errorf("FromGold(3) = %v, %v [FR-084]", v, err)
	}
	_, err = content.PricedMagicItemValueFromGold(-1)
	wantBlockError(t, err, "PricedMagicItemValue", content.BlockOutOfRange)
	// Negative gold must be rejected before multiplying: these wrapped to 0 copper.
	for _, g := range []int64{math.MinInt64, -(1 << 62)} {
		_, err = content.PricedMagicItemValueFromGold(g)
		wantBlockError(t, err, "PricedMagicItemValue", content.BlockOutOfRange)
	}
	if v, err := content.PricedMagicItemValueFromGold(math.MaxInt64 / 100); err != nil || v.CopperPieces() != math.MaxInt64/100*100 {
		t.Errorf("FromGold(max) = %v, %v", v, err)
	}
	_, err = content.PricedMagicItemValueFromGold(math.MaxInt64/100 + 1)
	wantBlockError(t, err, "PricedMagicItemValue", content.BlockOutOfRange)
	_, err = content.PricedMagicItemValueFromGold(1 << 62) // 100 * 2^62 wraps to 0 in int64, so the overflow guard alone rejects it
	wantBlockError(t, err, "PricedMagicItemValue", content.BlockOutOfRange)
	cost := content.PricedMagicItemValue{}
	for _, c := range []struct {
		r           content.ItemRarity
		days, hours int
		want        content.BlockProblem
	}{
		{content.RarityArtifact, 5, 40, content.BlockInvalidMember}, {0, 5, 40, content.BlockInvalidMember}, {7, 5, 40, content.BlockInvalidMember},
		{content.RarityRare, 0, 0, content.BlockOutOfRange}, {content.RarityRare, -1, -8, content.BlockOutOfRange},
		{content.RarityRare, 5, 41, content.BlockConstraint}, {content.RarityRare, 5, 39, content.BlockConstraint},
		{content.RarityRare, math.MaxInt32, math.MaxInt32, content.BlockConstraint},
	} {
		_, err := content.NewMagicItemCraftingRow(c.r, c.days, c.hours, cost)
		wantBlockError(t, err, "MagicItemCraftingRow", c.want)
	}
	if r, err := content.NewMagicItemCraftingRow(content.RarityRare, 1, 8, cost); err != nil || r.Days() != 1 {
		t.Errorf("row of one day = %+v, %v [FR-085]", r, err)
	}
	// The largest row whose hours fit a 32-bit int is accepted; one day more is
	// rejected on every platform width, so the hours never wrap.
	const maxDays = math.MaxInt32 / 8
	if _, err := content.NewMagicItemCraftingRow(content.RarityRare, maxDays, 8*maxDays, cost); err != nil {
		t.Errorf("the largest row = %v [FR-085]", err)
	}
	over := maxDays + 1
	_, err = content.NewMagicItemCraftingRow(content.RarityRare, over, 8*over, cost)
	wantBlockError(t, err, "MagicItemCraftingRow", content.BlockConstraint)
}

// Requirement: miSpec FR-086 - the Magic Item Tools table, one case per
// category; FixedTools needs a non-empty set of valid ids.
// Path: per case.
func TestMagicItemToolTable(t *testing.T) {
	t.Parallel()
	fixed := func(ids ...content.ItemID) content.MagicItemToolRule {
		f, err := content.NewFixedTools(ids)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	table := content.MagicItemToolTable{}
	cases := map[content.MagicItemCategory]content.MagicItemToolRule{
		content.MagicItemArmor:        content.ByBaseItemTools{},
		content.MagicItemPotion:       fixed("alchemists_supplies", "herbalism_kit"),
		content.MagicItemRing:         fixed("jewelers_tools"),
		content.MagicItemRod:          fixed("woodcarvers_tools"),
		content.MagicItemScroll:       fixed("calligraphers_supplies"),
		content.MagicItemStaff:        fixed("woodcarvers_tools"),
		content.MagicItemWand:         fixed("woodcarvers_tools"),
		content.MagicItemWeapon:       content.ByBaseItemTools{},
		content.MagicItemWondrousItem: content.TinkersOrBaseItemTools{},
	}
	for c, want := range cases {
		got := table.Of(c)
		gf, gok := got.(content.FixedTools)
		wf, wok := want.(content.FixedTools)
		if gok != wok || (gok && !gf.Equal(wf)) || (!gok && got != want) {
			t.Errorf("%v rule = %#v, want %#v [FR-086]", c, got, want)
		}
	}
	if table.Of(0) != nil || table.Of(10) != nil {
		t.Error("a non-category has a tool rule [FR-086]")
	}
	for _, c := range []struct {
		ids  []content.ItemID
		want content.BlockProblem
	}{
		{nil, content.BlockOutOfRange}, {[]content.ItemID{}, content.BlockOutOfRange}, {[]content.ItemID{"Bad"}, content.BlockInvalidMember},
		{[]content.ItemID{"a", "a"}, content.BlockDuplicate},
	} {
		_, err := content.NewFixedTools(c.ids)
		wantBlockError(t, err, "FixedTools", c.want)
	}
	in := []content.ItemID{"a", "b"}
	f, _ := content.NewFixedTools(in)
	in[0] = "z"
	out := f.Tools()
	out[1] = "z"
	if !slices.Equal(f.Tools(), []content.ItemID{"a", "b"}) {
		t.Error("FixedTools did not copy its slice [FR-008]")
	}
}

// Requirement: miSpec FR-087 - table "Resilience facts": the Artifact rule
// wins in every category, then potions and scrolls, then the rest.
// Path: per case.
func TestMagicItemResilienceFacts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		label string
		c     content.MagicItemCategory
		r     content.ItemRarity
		want  content.MagicItemResilience
	}{
		{"ring", content.MagicItemRing, content.RarityUncommon, content.ResilienceResistantToAllDamage},
		{"plate armor", content.MagicItemArmor, content.RarityRare, content.ResilienceResistantToAllDamage},
		{"potion", content.MagicItemPotion, content.RarityCommon, content.ResilienceNonmagicalDurability},
		{"scroll", content.MagicItemScroll, content.RarityRare, content.ResilienceNonmagicalDurability},
		{"artifact wondrous item", content.MagicItemWondrousItem, content.RarityArtifact, content.ResilienceImpervious},
		{"artifact potion", content.MagicItemPotion, content.RarityArtifact, content.ResilienceImpervious},
		{"invalid category", 0, content.RarityRare, 0},
		{"invalid rarity", content.MagicItemRing, 0, 0},
	}
	for _, c := range cases {
		if got := (content.MagicItemResilienceFacts{}).DefaultFor(c.c, c.r); got != c.want {
			t.Errorf("%s: %v, want %v [FR-087]", c.label, got, c.want)
		}
	}
	for _, c := range allMagicItemCategories {
		if (content.MagicItemResilienceFacts{}).DefaultFor(c, content.RarityArtifact) != content.ResilienceImpervious {
			t.Errorf("%v artifact is not impervious [FR-087]", c)
		}
		want := content.ResilienceResistantToAllDamage
		if c == content.MagicItemPotion || c == content.MagicItemScroll {
			want = content.ResilienceNonmagicalDurability
		}
		for _, r := range allItemRarities[:5] {
			if got := (content.MagicItemResilienceFacts{}).DefaultFor(c, r); got != want {
				t.Errorf("%v %v resilience = %v, want %v [FR-087]", c, r, got, want)
			}
		}
	}
	for i, r := range []content.MagicItemResilience{content.ResilienceResistantToAllDamage, content.ResilienceNonmagicalDurability, content.ResilienceImpervious} {
		if int(r) != i+1 || !r.Valid() || r.String() == "unknown" {
			t.Errorf("resilience %d = %q", r, r)
		}
	}
	if content.MagicItemResilience(0).Valid() || content.MagicItemResilience(4).Valid() || content.MagicItemResilience(4).String() != "unknown" {
		t.Error("an undefined resilience is valid")
	}
}

// Requirement: miSpec FR-088, FR-089 - the four prerequisite forms and
// their constructors, and the four Attunement constants.
// Path: per case.
func TestAttunementPrerequisitesAndFacts(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		ids  []content.ClassID
		want content.BlockProblem
	}{
		{nil, content.BlockOutOfRange}, {[]content.ClassID{"Bad"}, content.BlockInvalidMember}, {[]content.ClassID{"wizard", "wizard"}, content.BlockDuplicate},
	} {
		_, err := content.NewClassPrerequisite(c.ids)
		wantBlockError(t, err, "ClassPrerequisite", c.want)
	}
	in := []content.ClassID{"wizard", "cleric"}
	p, err := content.NewClassPrerequisite(in)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = "bard"
	got := p.Classes()
	got[1] = "bard"
	q, _ := content.NewClassPrerequisite([]content.ClassID{"wizard", "cleric"})
	if !p.Equal(q) || p.Equal(content.ClassPrerequisite{}) {
		t.Error("ClassPrerequisite did not copy its slice or Equal is wrong [FR-008]")
	}
	_, err = content.NewDescribedPrerequisite(" ")
	wantBlockError(t, err, "DescribedPrerequisite", content.BlockBlankText)
	d, err := content.NewDescribedPrerequisite("a Dwarf")
	if err != nil || d.Text() != "a Dwarf" {
		t.Errorf("NewDescribedPrerequisite = %v, %v", d, err)
	}
	forms := []content.AttunementPrerequisite{content.NoPrerequisite{}, p, content.SpellcasterPrerequisite{}, d}
	if len(forms) != 4 {
		t.Fatal("four forms")
	}
	f := content.AttunementFacts{}
	if f.StandardLimit() != 3 || f.MaxCopiesPerItem() != 1 || f.EndingDistanceFeet() != 100 || f.EndingHoursAway() != 24 {
		t.Errorf("attunement facts = %d %d %d %d [FR-089]", f.StandardLimit(), f.MaxCopiesPerItem(), f.EndingDistanceFeet(), f.EndingHoursAway())
	}
}

// Requirement: miSpec FR-090 - ItemRechargeTime and ChargeRecovery variants;
// RegainsFixed needs at least one charge; RegainsRolled needs dice.
// Path: per case.
func TestChargeRecoveryAndRechargeTime(t *testing.T) {
	t.Parallel()
	for i, r := range []content.ItemRechargeTime{content.RechargesAtNextDawn, content.RechargesAtSpecifiedTime, content.DoesNotRecharge} {
		if int(r) != i+1 || !r.Valid() || r.String() == "unknown" {
			t.Errorf("recharge time %d = %q [FR-090]", r, r)
		}
	}
	if content.ItemRechargeTime(0).Valid() || content.ItemRechargeTime(4).Valid() || content.ItemRechargeTime(4).String() != "unknown" {
		t.Error("an undefined recharge time is valid [FR-090]")
	}
	for _, n := range []int{0, -1} {
		_, err := content.NewRegainsFixed(n)
		wantBlockError(t, err, "RegainsFixed", content.BlockOutOfRange)
	}
	if r, err := content.NewRegainsFixed(1); err != nil || r.Charges() != 1 {
		t.Errorf("NewRegainsFixed(1) = %v, %v [FR-090]", r, err)
	}
	_, err := content.NewRegainsRolled(dice.Expression{})
	wantBlockError(t, err, "RegainsRolled", content.BlockMissing)
	roll := dice.MustParseExpression("1d6+1")
	if r, err := content.NewRegainsRolled(roll); err != nil || r.Roll() != roll {
		t.Errorf("NewRegainsRolled = %v, %v [FR-090]", r, err)
	}
	var _ = []content.ChargeRecovery{content.RegainsAll{}, content.RegainsFixed{}, content.RegainsRolled{}, content.NoChargeRecovery{}}
}

// Requirement: miSpec FR-091 - MagicItemFactsForEntry returns the defaults for
// an entry of each category; every override, base item, incorporated item and
// crafting basis constructor has its boundary; collections are copied; Equal
// sees each field.
// Path: per case.
func TestMagicItemFacts(t *testing.T) {
	t.Parallel()
	for _, c := range allMagicItemCategories {
		f := miFields()
		f.Category = c
		facts := content.MagicItemFactsForEntry(miEntry(t, f))
		if facts.ID() != "rod_x" || facts.Curse() != content.NotCursed || facts.AllowedSpells() != nil || facts.BaseItem() != content.BaseItemSpec(content.NoBaseItem{}) ||
			facts.IncorporatedItem() != content.IncorporatedItemSpec(content.NoIncorporatedItem{}) || facts.CraftingBasis() != content.MagicItemCraftingBasis(content.TableCrafting{}) ||
			facts.RechargeTime() != content.DoesNotRecharge || facts.ChargeRecovery() != content.ChargeRecovery(content.NoChargeRecovery{}) || facts.Fit() != content.ItemFitAdjusts {
			t.Errorf("%v defaults = %+v [FR-091]", c, facts)
		}
		for name, o := range map[string]interface{ Value() (bool, bool) }{
			"consumable": facts.Consumable(), "focus": facts.ArcaneFocus(), "quarterstaff": facts.NonmagicalQuarterstaff(), "worn": facts.MustBeWorn(),
		} {
			if _, set := o.Value(); set {
				t.Errorf("%v %s is overridden by default [FR-091]", c, name)
			}
		}
		if _, set := facts.ActivationTime().Value(); set {
			t.Error("activation time is overridden by default")
		}
		if _, set := facts.Resilience().Value(); set {
			t.Error("resilience is overridden by default")
		}
	}
	if (content.MagicItemFacts{}).ID() != "" {
		t.Error("the zero facts have an id")
	}
}

// Requirement: miSpec FR-091 - the override types: the zero value is the
// standard fact, an override reports its value, and NewMagicItemFacts rejects
// an override of a value that is not a member.
// Path: per case.
func TestFactOverrides(t *testing.T) {
	t.Parallel()
	if v, set := content.StandardBoolFact().Value(); set || v || content.StandardBoolFact() != (content.BoolFactOverride{}) {
		t.Error("the standard bool fact is overridden [FR-091]")
	}
	if v, set := content.OverriddenBoolFact(false).Value(); !set || v {
		t.Error("an override to false is not set [FR-091]")
	}
	if v, set := content.OverriddenBoolFact(true).Value(); !set || !v {
		t.Error("an override to true is not set [FR-091]")
	}
	if v, set := content.StandardActivationTime().Value(); set || v != 0 {
		t.Error("the standard activation time is overridden [FR-091]")
	}
	if v, set := content.OverriddenActivationTime(content.ActivationReaction).Value(); !set || v != content.ActivationReaction {
		t.Error("an activation override is not set [FR-091]")
	}
	if v, set := content.StandardResilience().Value(); set || v != 0 {
		t.Error("the standard resilience is overridden [FR-091]")
	}
	if v, set := content.OverriddenResilience(content.ResilienceImpervious).Value(); !set || v != content.ResilienceImpervious {
		t.Error("a resilience override is not set [FR-091]")
	}
	for name, f := range map[string]content.MagicItemFactsFields{
		"activation 0":  {ActivationTime: content.OverriddenActivationTime(0)},
		"activation 6":  {ActivationTime: content.OverriddenActivationTime(6)},
		"resilience 0":  {Resilience: content.OverriddenResilience(0)},
		"resilience 4":  {Resilience: content.OverriddenResilience(4)},
		"curse 3":       {Curse: 3},
		"recharge 4":    {RechargeTime: 4},
		"fit 3":         {Fit: 3},
		"zero spec":     {BaseItem: content.SpecificBaseItems{}},
		"zero incorp":   {IncorporatedItem: content.IncorporatedItem{}},
		"zero crafting": {CraftingBasis: content.EquipmentRulesCrafting{}},
		"zero fixed":    {ChargeRecovery: content.RegainsFixed{}},
		"zero rolled":   {ChargeRecovery: content.RegainsRolled{}},
		"bad spell":     {AllowedSpells: []content.SpellID{"Bad"}},
	} {
		_, err := content.NewMagicItemFacts("item_x", f)
		wantBlockError(t, err, "MagicItemFacts", content.BlockInvalidMember)
		_ = name
	}
	_, err := content.NewMagicItemFacts("item_x", content.MagicItemFactsFields{AllowedSpells: []content.SpellID{"a", "a"}})
	wantBlockError(t, err, "MagicItemFacts", content.BlockDuplicate)
	var ie *content.IDError
	if _, err := content.NewMagicItemFacts("Bad", content.MagicItemFactsFields{}); !errors.As(err, &ie) {
		t.Errorf("a bad id gave %v, want *IDError", err)
	}
	if !mustPanicsWith(func() { content.MustMagicItemFacts("", content.MagicItemFactsFields{}) }) {
		t.Error("MustMagicItemFacts did not panic")
	}
}

// Requirement: miSpec FR-091 - the base item, incorporated item and crafting
// basis constructors reject an empty or invalid set, a bad id and a negative
// cost.
// Path: per case.
func TestMagicItemFactParts(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		ids  []content.ItemID
		want content.BlockProblem
	}{
		{nil, content.BlockOutOfRange}, {[]content.ItemID{"Bad"}, content.BlockInvalidMember}, {[]content.ItemID{"a", "a"}, content.BlockDuplicate},
	} {
		_, err := content.NewSpecificBaseItems(c.ids)
		wantBlockError(t, err, "SpecificBaseItems", c.want)
	}
	in := []content.ItemID{"a", "b"}
	b, err := content.NewSpecificBaseItems(in)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = "z"
	out := b.Items()
	out[1] = "z"
	if !slices.Equal(b.Items(), []content.ItemID{"a", "b"}) {
		t.Error("SpecificBaseItems did not copy its slice [FR-008]")
	}
	_, err = content.NewIncorporatedItem("Bad", 0)
	wantBlockError(t, err, "IncorporatedItem", content.BlockInvalidMember)
	_, err = content.NewIncorporatedItem("a", -1)
	wantBlockError(t, err, "IncorporatedItem", content.BlockOutOfRange)
	if i, err := content.NewIncorporatedItem("a", 0); err != nil || i.Item() != "a" || i.CostCopperPieces() != 0 {
		t.Errorf("NewIncorporatedItem(a, 0) = %v, %v", i, err)
	}
	_, err = content.NewEquipmentRulesCrafting("Bad")
	wantBlockError(t, err, "EquipmentRulesCrafting", content.BlockInvalidMember)
	if c, err := content.NewEquipmentRulesCrafting("a"); err != nil || c.Item() != "a" {
		t.Errorf("NewEquipmentRulesCrafting = %v, %v", c, err)
	}
	for i, f := range []content.ItemFit{content.ItemFitAdjusts, content.ItemFitFixed} {
		if int(f) != i+1 || !f.Valid() || f.String() == "unknown" {
			t.Errorf("fit %d = %q", f, f)
		}
	}
	if content.ItemFit(0).Valid() || content.ItemFit(3).Valid() || content.ItemFit(3).String() != "unknown" ||
		content.CurseStatus(0).Valid() || content.CurseStatus(3).Valid() || content.CurseStatus(3).String() != "unknown" ||
		!content.NotCursed.Valid() || !content.Cursed.Valid() || content.Cursed.String() != "cursed" || content.NotCursed.String() != "notCursed" {
		t.Error("the fit or curse vocabulary is wrong [FR-091]")
	}
}

// Requirement: miSpec FR-008, FR-009, FR-091 - facts copy their collections
// on the way in and out, and Equal sees a change of any single field.
// Path: unhappy (immutability).
func TestMagicItemFactsImmutableAndEqual(t *testing.T) {
	t.Parallel()
	spells := []content.SpellID{"a", "b"}
	fixed, _ := content.NewSpecificBaseItems([]content.ItemID{"x"})
	incorporated, _ := content.NewIncorporatedItem("y", 5)
	crafting, _ := content.NewEquipmentRulesCrafting("z")
	fixedN, _ := content.NewRegainsFixed(2)
	rolled, _ := content.NewRegainsRolled(dice.MustParseExpression("1d4"))
	base := content.MagicItemFactsFields{
		Consumable: content.OverriddenBoolFact(true), ArcaneFocus: content.OverriddenBoolFact(true), NonmagicalQuarterstaff: content.OverriddenBoolFact(true),
		MustBeWorn: content.OverriddenBoolFact(true), ActivationTime: content.OverriddenActivationTime(content.ActivationAction), Curse: content.Cursed,
		Resilience: content.OverriddenResilience(content.ResilienceImpervious), AllowedSpells: spells, BaseItem: fixed, IncorporatedItem: incorporated,
		CraftingBasis: crafting, RechargeTime: content.RechargesAtNextDawn, ChargeRecovery: fixedN, Fit: content.ItemFitFixed,
	}
	f := content.MustMagicItemFacts("item_x", base)
	spells[0] = "z"
	got := f.AllowedSpells()
	got[1] = "z"
	if !slices.Equal(f.AllowedSpells(), []content.SpellID{"a", "b"}) {
		t.Error("the facts did not copy the spells [FR-008]")
	}
	same := base
	same.AllowedSpells = []content.SpellID{"a", "b"}
	if !f.Equal(content.MustMagicItemFacts("item_x", same)) {
		t.Error("equal facts are unequal [FR-009]")
	}
	fixed2, _ := content.NewSpecificBaseItems([]content.ItemID{"w"})
	incorporated2, _ := content.NewIncorporatedItem("y", 6)
	crafting2, _ := content.NewEquipmentRulesCrafting("w")
	fixedM, _ := content.NewRegainsFixed(3)
	mods := map[string]func(*content.MagicItemFactsFields){
		"consumable":    func(m *content.MagicItemFactsFields) { m.Consumable = content.OverriddenBoolFact(false) },
		"focus":         func(m *content.MagicItemFactsFields) { m.ArcaneFocus = content.StandardBoolFact() },
		"quarterstaff":  func(m *content.MagicItemFactsFields) { m.NonmagicalQuarterstaff = content.StandardBoolFact() },
		"worn":          func(m *content.MagicItemFactsFields) { m.MustBeWorn = content.StandardBoolFact() },
		"activation":    func(m *content.MagicItemFactsFields) { m.ActivationTime = content.StandardActivationTime() },
		"curse":         func(m *content.MagicItemFactsFields) { m.Curse = content.NotCursed },
		"resilience":    func(m *content.MagicItemFactsFields) { m.Resilience = content.StandardResilience() },
		"spells":        func(m *content.MagicItemFactsFields) { m.AllowedSpells = []content.SpellID{"a"} },
		"base":          func(m *content.MagicItemFactsFields) { m.BaseItem = fixed2 },
		"base kind":     func(m *content.MagicItemFactsFields) { m.BaseItem = content.UnspecifiedBaseItem{} },
		"incorporated":  func(m *content.MagicItemFactsFields) { m.IncorporatedItem = incorporated2 },
		"crafting":      func(m *content.MagicItemFactsFields) { m.CraftingBasis = crafting2 },
		"recharge":      func(m *content.MagicItemFactsFields) { m.RechargeTime = content.DoesNotRecharge },
		"recovery":      func(m *content.MagicItemFactsFields) { m.ChargeRecovery = fixedM },
		"recovery kind": func(m *content.MagicItemFactsFields) { m.ChargeRecovery = rolled },
		"fit":           func(m *content.MagicItemFactsFields) { m.Fit = content.ItemFitAdjusts },
	}
	for name, mod := range mods {
		m := base
		mod(&m)
		if f.Equal(content.MustMagicItemFacts("item_x", m)) {
			t.Errorf("facts differing in %s are equal [FR-009]", name)
		}
	}
	if f.Equal(content.MustMagicItemFacts("item_y", base)) {
		t.Error("facts of another item are equal [FR-009]")
	}
	if f.BaseItem().(content.SpecificBaseItems).Equal(fixed2) || f.Consumable() != content.OverriddenBoolFact(true) ||
		f.ArcaneFocus() != content.OverriddenBoolFact(true) || f.NonmagicalQuarterstaff() != content.OverriddenBoolFact(true) || f.MustBeWorn() != content.OverriddenBoolFact(true) ||
		f.Curse() != content.Cursed || f.RechargeTime() != content.RechargesAtNextDawn || f.Fit() != content.ItemFitFixed ||
		f.IncorporatedItem() != content.IncorporatedItemSpec(incorporated) || f.CraftingBasis() != content.MagicItemCraftingBasis(crafting) || f.ChargeRecovery() != content.ChargeRecovery(fixedN) {
		t.Errorf("accessors disagree with the fields: %+v [FR-091]", f)
	}
	if v, _ := f.ActivationTime().Value(); v != content.ActivationAction {
		t.Error("activation accessor")
	}
	if v, _ := f.Resilience().Value(); v != content.ResilienceImpervious {
		t.Error("resilience accessor")
	}
}

// Requirement: miSpec FR-092 - the d10 communication table and its enum.
// Path: unhappy (boundary).
func TestSentientCommunication(t *testing.T) {
	t.Parallel()
	want := map[int]content.SentientCommunication{
		1: content.SentientTransmitsEmotion, 6: content.SentientTransmitsEmotion, 7: content.SentientSpeaksLanguages,
		9: content.SentientSpeaksLanguages, 10: content.SentientSpeaksLanguagesAndTelepathy,
	}
	for roll, w := range want {
		if got, err := content.SentientCommunicationForRoll(roll); err != nil || got != w {
			t.Errorf("roll %d = %v, %v, want %v [FR-092]", roll, got, err, w)
		}
	}
	for _, roll := range []int{0, 11, -1} {
		_, err := content.SentientCommunicationForRoll(roll)
		wantRollError(t, err, roll, 1, 10)
	}
	for i, c := range []content.SentientCommunication{content.SentientTransmitsEmotion, content.SentientSpeaksLanguages, content.SentientSpeaksLanguagesAndTelepathy} {
		if int(c) != i+1 || !c.Valid() || c.String() == "unknown" {
			t.Errorf("communication %d = %q", c, c)
		}
	}
	if content.SentientCommunication(0).Valid() || content.SentientCommunication(4).Valid() || content.SentientCommunication(4).String() != "unknown" {
		t.Error("an undefined communication is valid")
	}
}

// Requirement: miSpec FR-093 - the d4 senses table, and a range of 0 or less
// is rejected.
// Path: unhappy (boundary).
func TestSentientSenses(t *testing.T) {
	t.Parallel()
	want := map[int]struct {
		feet  int
		sense bool
	}{1: {30, false}, 2: {60, false}, 3: {120, false}, 4: {120, true}}
	for roll, w := range want {
		s, err := content.SentientSensesForRoll(roll)
		if err != nil || s.RangeFeet() != w.feet {
			t.Errorf("roll %d = %v, %v [FR-093]", roll, s, err)
			continue
		}
		v, has := s.Vision().(content.StandardVisionAndSense)
		if has != w.sense || (has && v.Sense() != content.SenseDarkvision) {
			t.Errorf("roll %d vision = %#v [FR-093]", roll, s.Vision())
		}
		if !has && s.Vision() != content.SentientVision(content.StandardVisionOnly{}) {
			t.Errorf("roll %d vision = %#v", roll, s.Vision())
		}
	}
	for _, roll := range []int{0, 5, -1} {
		_, err := content.SentientSensesForRoll(roll)
		wantRollError(t, err, roll, 1, 4)
	}
	for _, feet := range []int{0, -1} {
		_, err := content.NewSentientSenses(feet, content.StandardVisionOnly{})
		wantBlockError(t, err, "SentientSenses", content.BlockOutOfRange)
	}
	_, err := content.NewSentientSenses(30, nil)
	wantBlockError(t, err, "SentientSenses", content.BlockMissing)
	if s, err := content.NewSentientSenses(1, content.StandardVisionOnly{}); err != nil || s.RangeFeet() != 1 {
		t.Errorf("NewSentientSenses(1) = %v, %v", s, err)
	}
	_, err = content.NewStandardVisionAndSense(0)
	wantBlockError(t, err, "StandardVisionAndSense", content.BlockInvalidMember)
	if v, err := content.NewStandardVisionAndSense(content.SenseTruesight); err != nil || v.Sense() != content.SenseTruesight {
		t.Errorf("NewStandardVisionAndSense = %v, %v", v, err)
	}
}

// Requirement: miSpec FR-094 - the ten special purpose kinds in table order and
// the d10 table.
// Path: unhappy (boundary).
func TestSpecialPurposeKind(t *testing.T) {
	t.Parallel()
	names := []string{"aligned", "bane", "creatorSeeker", "destinySeeker", "destroyer", "glorySeeker", "loreSeeker", "protector", "soulmateSeeker", "templar"}
	for roll := 1; roll <= 10; roll++ {
		k, err := content.SpecialPurposeKindForRoll(roll)
		if err != nil || int(k) != roll || k.String() != names[roll-1] || !k.Valid() {
			t.Errorf("roll %d = %v, %v [FR-094]", roll, k, err)
		}
	}
	for _, roll := range []int{0, 11} {
		_, err := content.SpecialPurposeKindForRoll(roll)
		wantRollError(t, err, roll, 1, 10)
	}
	if content.SpecialPurposeKind(0).Valid() || content.SpecialPurposeKind(11).Valid() || content.SpecialPurposeKind(11).String() != "unknown" {
		t.Error("an undefined special purpose is valid [FR-094]")
	}
}

// Requirement: miSpec FR-095 - the d100 alignment table, each edge of every
// band, never unaligned.
// Path: unhappy (boundary).
func TestSentientAlignmentTable(t *testing.T) {
	t.Parallel()
	bands := []struct {
		lo, hi int
		a      content.Alignment
	}{
		{1, 15, content.AlignmentLawfulGood}, {16, 35, content.AlignmentNeutralGood}, {36, 50, content.AlignmentChaoticGood},
		{51, 63, content.AlignmentLawfulNeutral}, {64, 73, content.AlignmentNeutral}, {74, 85, content.AlignmentChaoticNeutral},
		{86, 89, content.AlignmentLawfulEvil}, {90, 96, content.AlignmentNeutralEvil}, {97, 100, content.AlignmentChaoticEvil},
	}
	for _, b := range bands {
		for _, roll := range []int{b.lo, b.hi} {
			if got, err := content.SentientAlignmentTableForRoll(roll); err != nil || got != b.a {
				t.Errorf("roll %d = %v, %v, want %v [FR-095]", roll, got, err, b.a)
			}
		}
	}
	for roll := 1; roll <= 100; roll++ {
		if got, err := content.SentientAlignmentTableForRoll(roll); err != nil || got == content.AlignmentUnaligned || !got.Valid() {
			t.Errorf("roll %d = %v, %v, want an alignment that is not unaligned [FR-095]", roll, got, err)
		}
	}
	for _, roll := range []int{0, 101, -1} {
		_, err := content.SentientAlignmentTableForRoll(roll)
		wantRollError(t, err, roll, 1, 100)
	}
}

// Requirement: miSpec FR-096 - the exact demand and refusal-power sets.
// Path: happy.
func TestSentientDemandsAndRefusals(t *testing.T) {
	t.Parallel()
	demands := []string{"Chase My Dreams", "Get Rid of It", "It's Time for a Change", "Keep Me Close"}
	for i, d := range []content.SentientDemand{
		content.SentientDemandChaseMyDreams, content.SentientDemandGetRidOfIt, content.SentientDemandItsTimeForAChange, content.SentientDemandKeepMeClose,
	} {
		if int(d) != i+1 || d.String() != demands[i] || !d.Valid() {
			t.Errorf("demand %d = %q [FR-096]", d, d)
		}
	}
	if content.SentientDemand(0).Valid() || content.SentientDemand(5).Valid() || content.SentientDemand(5).String() != "unknown" {
		t.Error("an undefined demand is valid [FR-096]")
	}
	powers := []string{"preventAttunement", "suppressProperties", "attemptControl"}
	for i, p := range []content.SentientRefusalPower{
		content.SentientRefusalPreventAttunement, content.SentientRefusalSuppressProperties, content.SentientRefusalAttemptControl,
	} {
		if int(p) != i+1 || p.String() != powers[i] || !p.Valid() {
			t.Errorf("refusal power %d = %q [FR-096]", p, p)
		}
	}
	if content.SentientRefusalPower(0).Valid() || content.SentientRefusalPower(4).Valid() || content.SentientRefusalPower(4).String() != "unknown" {
		t.Error("an undefined refusal power is valid [FR-096]")
	}
}
