package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var noSource content.SourceCitation

func dice1d(t *testing.T, text string) dice.Expression {
	t.Helper()
	e, err := dice.ParseExpression(text)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// Requirement: equipSpec FR-101 to FR-103 - weapon categories, types, damage,
// range pairs and property uses are closed typed values with validating
// constructors.
// Path: per case.
func TestWeaponParts(t *testing.T) {
	t.Parallel()
	if content.WeaponSimple.String() != "Simple" || content.WeaponMartial.String() != "Martial" || content.WeaponCategory(0).Valid() || content.WeaponCategory(3).Valid() {
		t.Error("WeaponCategory is Simple or Martial [FR-101]")
	}
	if content.WeaponMelee.String() != "Melee" || content.WeaponRanged.String() != "Ranged" || content.WeaponType(0).Valid() || content.WeaponType(3).String() != "unknown" {
		t.Error("WeaponType is Melee or Ranged [FR-101]")
	}
	if content.TwoHandedAlways.String() != "always" || content.TwoHandedUnlessMounted.String() != "unlessMounted" || content.TwoHandedCondition(0).Valid() {
		t.Error("TwoHandedCondition is always or unlessMounted [FR-103]")
	}
	for i, n := range []string{"Arrow", "Bolt", "Bullet", "Needle"} {
		if a := content.AmmunitionType(i + 1); a.String() != n || !a.Valid() {
			t.Errorf("AmmunitionType(%d) = %q, want %q [FR-103]", i+1, a, n)
		}
	}
	if content.AmmunitionType(0).Valid() || content.AmmunitionType(5).Valid() {
		t.Error("AmmunitionType outside 1 to 4 is valid [FR-103]")
	}

	// Fixed damage: 0 is rejected, 1 accepted (the Blowgun case of FR-102).
	if d, err := content.NewWeaponFixedDamage(1, content.DamagePiercing); err != nil || d.Amount() != 1 || d.DamageType() != content.DamagePiercing {
		t.Errorf("NewWeaponFixedDamage(1) = %+v, %v [FR-102 happy]", d, err)
	}
	for _, c := range []struct {
		amount int
		t      content.DamageType
		want   content.BlockProblem
	}{{0, content.DamagePiercing, content.BlockOutOfRange}, {beyond(), content.DamagePiercing, content.BlockOutOfRange}, {1, 0, content.BlockInvalidMember}} {
		_, err := content.NewWeaponFixedDamage(c.amount, c.t)
		wantBlockError(t, err, "WeaponFixedDamage", c.want)
	}
	// Dice damage: no modifier and at least one die.
	if d, err := content.NewWeaponDiceDamage(dice1d(t, "2d6"), content.DamageSlashing); err != nil || d.Dice().Notation() != "2d6" || d.DamageType() != content.DamageSlashing {
		t.Errorf("NewWeaponDiceDamage = %+v, %v [FR-102 happy]", d, err)
	}
	_, err := content.NewWeaponDiceDamage(dice1d(t, "1d8+1"), content.DamageSlashing)
	wantBlockError(t, err, "WeaponDiceDamage", content.BlockOutOfRange)
	_, err = content.NewWeaponDiceDamage(dice.Expression{}, content.DamageSlashing)
	wantBlockError(t, err, "WeaponDiceDamage", content.BlockOutOfRange)
	_, err = content.NewWeaponDiceDamage(dice1d(t, "1d8"), 0)
	wantBlockError(t, err, "WeaponDiceDamage", content.BlockInvalidMember)

	// RangePair: normal at least 1, long greater than normal.
	if r, err := content.NewRangePair(20, 60); err != nil || r.Normal() != 20 || r.Long() != 60 {
		t.Errorf("NewRangePair(20, 60) = %+v, %v [FR-103 happy]", r, err)
	}
	for _, c := range [][2]int{{0, 10}, {20, 20}, {20, 19}, {-1, 5}, {1, beyond()}} {
		_, err := content.NewRangePair(c[0], c[1])
		wantBlockError(t, err, "RangePair", content.BlockOutOfRange)
	}

	// Property uses.
	for _, p := range []content.WeaponProperty{content.WeaponPropertyFinesse, content.WeaponPropertyHeavy, content.WeaponPropertyLight, content.WeaponPropertyLoading, content.WeaponPropertyReach} {
		if u, err := content.NewPlainUse(p); err != nil || u.Property() != p {
			t.Errorf("NewPlainUse(%v) = %+v, %v [FR-103 happy]", p, u, err)
		}
	}
	for _, p := range []content.WeaponProperty{content.WeaponPropertyThrown, content.WeaponPropertyAmmunition, content.WeaponPropertyVersatile, content.WeaponPropertyTwoHanded, 0} {
		_, err := content.NewPlainUse(p)
		wantBlockError(t, err, "PlainUse", content.BlockInvalidMember)
	}
	rng, _ := content.NewRangePair(20, 60)
	if u, err := content.NewAmmunitionUse(rng, content.AmmunitionBolt); err != nil || u.Property() != content.WeaponPropertyAmmunition || u.Range() != rng || u.Ammunition() != content.AmmunitionBolt {
		t.Errorf("NewAmmunitionUse = %+v, %v [FR-103 happy]", u, err)
	}
	_, err = content.NewAmmunitionUse(content.RangePair{}, content.AmmunitionBolt)
	wantBlockError(t, err, "AmmunitionUse", content.BlockOutOfRange)
	_, err = content.NewAmmunitionUse(rng, 0)
	wantBlockError(t, err, "AmmunitionUse", content.BlockInvalidMember)
	if u, err := content.NewThrownUse(rng); err != nil || u.Property() != content.WeaponPropertyThrown || u.Range() != rng {
		t.Errorf("NewThrownUse = %+v, %v [FR-103 happy]", u, err)
	}
	_, err = content.NewThrownUse(content.RangePair{})
	wantBlockError(t, err, "ThrownUse", content.BlockOutOfRange)
	if u, err := content.NewVersatileUse(dice1d(t, "1d10")); err != nil || u.Property() != content.WeaponPropertyVersatile || u.Damage().Notation() != "1d10" {
		t.Errorf("NewVersatileUse = %+v, %v [FR-103 happy]", u, err)
	}
	_, err = content.NewVersatileUse(dice1d(t, "1d10-1"))
	wantBlockError(t, err, "VersatileUse", content.BlockOutOfRange)
	if u, err := content.NewTwoHandedUse(content.TwoHandedUnlessMounted); err != nil || u.Property() != content.WeaponPropertyTwoHanded || u.Condition() != content.TwoHandedUnlessMounted {
		t.Errorf("NewTwoHandedUse = %+v, %v [FR-103 happy]", u, err)
	}
	_, err = content.NewTwoHandedUse(0)
	wantBlockError(t, err, "TwoHandedUse", content.BlockInvalidMember)
}

func testWeapon(t *testing.T, props []content.WeaponPropertyUse, weight content.WeightSpec) (content.Weapon, error) {
	t.Helper()
	d, err := content.NewWeaponDiceDamage(dice1d(t, "1d6"), content.DamageBludgeoning)
	if err != nil {
		t.Fatal(err)
	}
	return content.NewWeapon("club_x", "Club X", noSource, content.WeaponSimple, content.WeaponMelee, d, props, content.MasterySap, weight, fixedCost(1, content.CoinSP))
}

// Requirement: equipSpec FR-001, FR-006, FR-007, FR-106 - a Weapon is an
// EquipmentEntry of kind equipment that holds what it was built with and gives
// out copies of its property list.
// Path: happy and unhappy (immutability).
func TestWeapon(t *testing.T) {
	t.Parallel()
	rng, _ := content.NewRangePair(20, 60)
	thrown, _ := content.NewThrownUse(rng)
	finesse, _ := content.NewPlainUse(content.WeaponPropertyFinesse)
	w, err := testWeapon(t, []content.WeaponPropertyUse{finesse, thrown}, content.MustWeight(2, ""))
	if err != nil {
		t.Fatal(err)
	}
	if w.Key() != content.NewKey(content.KindEquipment, content.MustID("club_x")) || w.ID() != "club_x" || w.Name() != "Club X" || w.Source() != noSource ||
		w.EquipmentCategory() != content.EquipmentWeapon || w.Category() != content.WeaponSimple || w.Type() != content.WeaponMelee ||
		w.Mastery() != content.MasterySap || w.Weight() != content.MustWeight(2, "") || w.Price() != fixedCost(1, content.CoinSP) {
		t.Errorf("Weapon accessors = %+v [FR-106]", w)
	}
	if d, ok := w.Damage().(content.WeaponDiceDamage); !ok || d.Dice().Notation() != "1d6" {
		t.Errorf("Damage = %#v [FR-106]", w.Damage())
	}
	got := w.Properties()
	got[0] = nil
	if again := w.Properties(); len(again) != 2 || again[0] != content.WeaponPropertyUse(finesse) {
		t.Errorf("mutating the returned properties changed the weapon: %v [FR-007]", again)
	}
	var _ content.EquipmentEntry = w
	// A weapon printed "—" has no properties and may have no weight.
	none, err := testWeapon(t, nil, content.NoWeight{})
	if err != nil || none.Properties() != nil || none.Weight() != content.WeightSpec(content.NoWeight{}) {
		t.Errorf("a weapon with no properties and no weight = %+v, %v [FR-103 empty case]", none, err)
	}
	// The input list is copied.
	in := []content.WeaponPropertyUse{finesse}
	cp, err := testWeapon(t, in, content.NoWeight{})
	if err != nil {
		t.Fatal(err)
	}
	in[0] = thrown
	if cp.Properties()[0] != content.WeaponPropertyUse(finesse) {
		t.Error("NewWeapon kept the caller's slice [FR-007]")
	}
}

// Requirement: equipSpec FR-103, FR-106 - NewWeapon rejects an invalid part,
// a repeated property and a Varies weight, with a typed error.
// Path: per case.
func TestNewWeaponRejects(t *testing.T) {
	t.Parallel()
	finesse, _ := content.NewPlainUse(content.WeaponPropertyFinesse)
	good, _ := content.NewWeaponFixedDamage(1, content.DamagePiercing)
	price := fixedCost(1, content.CoinGP)
	w := content.MustWeight(1, "")
	cases := []struct {
		name string
		make func() error
		want content.BlockProblem
	}{
		{"repeated property", func() error {
			_, err := testWeapon(t, []content.WeaponPropertyUse{finesse, finesse}, w)
			return err
		}, content.BlockDuplicate},
		{"Varies weight", func() error { _, err := testWeapon(t, nil, content.WeightVaries{}); return err }, content.BlockInvalidMember},
		{"nil weight", func() error { _, err := testWeapon(t, nil, nil); return err }, content.BlockInvalidMember},
		{"zero property use", func() error {
			_, err := testWeapon(t, []content.WeaponPropertyUse{content.PlainUse{}}, w)
			return err
		}, content.BlockInvalidMember},
		{"nil property use", func() error {
			_, err := testWeapon(t, []content.WeaponPropertyUse{nil}, w)
			return err
		}, content.BlockInvalidMember},
		{"zero category", func() error {
			_, err := content.NewWeapon("a", "A", noSource, 0, content.WeaponMelee, good, nil, content.MasterySap, w, price)
			return err
		}, content.BlockInvalidMember},
		{"zero type", func() error {
			_, err := content.NewWeapon("a", "A", noSource, content.WeaponSimple, 0, good, nil, content.MasterySap, w, price)
			return err
		}, content.BlockInvalidMember},
		{"nil damage", func() error {
			_, err := content.NewWeapon("a", "A", noSource, content.WeaponSimple, content.WeaponMelee, nil, nil, content.MasterySap, w, price)
			return err
		}, content.BlockInvalidMember},
		{"zero mastery", func() error {
			_, err := content.NewWeapon("a", "A", noSource, content.WeaponSimple, content.WeaponMelee, good, nil, 0, w, price)
			return err
		}, content.BlockInvalidMember},
		{"zero price", func() error {
			_, err := content.NewWeapon("a", "A", noSource, content.WeaponSimple, content.WeaponMelee, good, nil, content.MasterySap, w, content.FixedCost{})
			return err
		}, content.BlockInvalidMember},
		{"bad name", func() error {
			_, err := content.NewWeapon("a", " ", noSource, content.WeaponSimple, content.WeaponMelee, good, nil, content.MasterySap, w, price)
			return err
		}, content.BlockInvalidName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantBlockError(t, c.make(), "Weapon", c.want)
		})
	}
	_, err := content.NewWeapon("Bad Id", "A", noSource, content.WeaponSimple, content.WeaponMelee, good, nil, content.MasterySap, w, price)
	var ie *content.IDError
	if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an invalid id gave %v, want *IDError wrapping ErrInvalidInput [FR-003]", err)
	}
}

// Requirement: equipSpec FR-022, FR-104 (SRD 5.2.1, Armor table headings, p.
// 92) - ArmorCategory gives the don and doff times as typed values.
// Path: per case.
func TestArmorCategoryTimes(t *testing.T) {
	t.Parallel()
	minutes := func(n int) content.DonDoffTime {
		m, err := content.NewDonDoffMinutes(n)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	cases := []struct {
		c         content.ArmorCategory
		name      string
		don, doff content.DonDoffTime
	}{
		{content.ArmorLight, "Light", minutes(1), minutes(1)},
		{content.ArmorMedium, "Medium", minutes(5), minutes(1)},
		{content.ArmorHeavy, "Heavy", minutes(10), minutes(5)},
		{content.ArmorShield, "Shield", content.DonDoffByUtilize{}, content.DonDoffByUtilize{}},
	}
	for _, c := range cases {
		if c.c.String() != c.name || !c.c.Valid() || c.c.DonTime() != c.don || c.c.DoffTime() != c.doff {
			t.Errorf("%v: don %#v doff %#v, want %#v and %#v [FR-022]", c.c, c.c.DonTime(), c.c.DoffTime(), c.don, c.doff)
		}
	}
	if content.ArmorCategory(0).Valid() || content.ArmorCategory(0).DonTime() != nil || content.ArmorCategory(5).DoffTime() != nil || content.ArmorCategory(5).String() != "unknown" {
		t.Error("an invalid ArmorCategory has times [FR-104]")
	}
	for _, n := range []int{0, -1, beyond()} {
		_, err := content.NewDonDoffMinutes(n)
		wantBlockError(t, err, "DonDoffMinutes", content.BlockOutOfRange)
	}
	if m, _ := content.NewDonDoffMinutes(7); m.Minutes() != 7 {
		t.Error("Minutes() != 7")
	}
}

// Requirement: equipSpec FR-024, FR-093, FR-105 - ArmorClassRule is a BaseRule
// with a Dexterity treatment or a BonusRule; NewArmor ties a Shield to a bonus
// and body armor to a base.
// Path: per case.
func TestArmor(t *testing.T) {
	t.Parallel()
	capped, err := content.NewCappedDexterity(2)
	if err != nil || capped.Maximum() != 2 {
		t.Fatalf("NewCappedDexterity(2) = %+v, %v [FR-093 happy]", capped, err)
	}
	for _, n := range []int{-1, beyond()} {
		_, err := content.NewCappedDexterity(n)
		wantBlockError(t, err, "CappedDexterity", content.BlockOutOfRange)
	}
	base, err := content.NewBaseRule(14, capped)
	if err != nil || base.Base() != 14 || base.Dexterity() != content.DexterityBonus(capped) {
		t.Fatalf("NewBaseRule = %+v, %v [FR-093 happy]", base, err)
	}
	for _, b := range []int{0, 31} {
		_, err := content.NewBaseRule(b, content.FullDexterity{})
		wantBlockError(t, err, "BaseRule", content.BlockOutOfRange)
	}
	_, err = content.NewBaseRule(11, nil)
	wantBlockError(t, err, "BaseRule", content.BlockInvalidMember)
	bonus, err := content.NewBonusRule(2)
	if err != nil || bonus.Bonus() != 2 {
		t.Fatalf("NewBonusRule(2) = %+v, %v [FR-024 happy]", bonus, err)
	}
	for _, b := range []int{0, 11} {
		_, err := content.NewBonusRule(b)
		wantBlockError(t, err, "BonusRule", content.BlockOutOfRange)
	}
	for _, d := range []content.DexterityBonus{content.FullDexterity{}, capped, content.NoDexterity{}} {
		switch d.(type) { // exhaustive: the three treatments
		case content.FullDexterity, content.CappedDexterity, content.NoDexterity:
		default:
			t.Errorf("unexpected treatment %T", d)
		}
	}

	w, price := content.MustWeight(8, ""), fixedCost(10, content.CoinGP)
	mk := func(c content.ArmorCategory, r content.ArmorClassRule, strength int) (content.Armor, error) {
		return content.NewArmor("armor_x", "Armor X", noSource, c, r, strength, true, w, price)
	}
	a, err := mk(content.ArmorMedium, base, 13)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := a.StrengthRequirement(); a.Category() != content.ArmorMedium || a.ArmorClass() != content.ArmorClassRule(base) || !ok || s != 13 || !a.StealthDisadvantage() ||
		a.Weight() != w || a.Price() != price || a.EquipmentCategory() != content.EquipmentArmor || a.Key().Kind() != content.KindEquipment {
		t.Errorf("Armor accessors = %+v [FR-105]", a)
	}
	var _ content.EquipmentEntry = a
	if none, err := mk(content.ArmorLight, base, 0); err != nil {
		t.Errorf("an armor without a Strength requirement: %v [FR-105 empty case]", err)
	} else if _, ok := none.StrengthRequirement(); ok {
		t.Error("StrengthRequirement of an armor printed \"—\" is present")
	}
	if shield, err := mk(content.ArmorShield, bonus, 0); err != nil || shield.ArmorClass() != content.ArmorClassRule(bonus) {
		t.Errorf("a Shield with a BonusRule = %+v, %v [FR-024 happy]", shield, err)
	}
	cases := []struct {
		name     string
		c        content.ArmorCategory
		r        content.ArmorClassRule
		strength int
		want     content.BlockProblem
	}{
		{"Shield with a base", content.ArmorShield, base, 0, content.BlockConstraint},
		{"body armor with a bonus", content.ArmorHeavy, bonus, 0, content.BlockConstraint},
		{"strength 31", content.ArmorHeavy, base, 31, content.BlockOutOfRange},
		{"strength -1", content.ArmorHeavy, base, -1, content.BlockOutOfRange},
		{"zero category", 0, base, 0, content.BlockInvalidMember},
		{"nil rule", content.ArmorLight, nil, 0, content.BlockInvalidMember},
		{"zero base rule", content.ArmorLight, content.BaseRule{}, 0, content.BlockInvalidMember},
		{"zero bonus rule", content.ArmorShield, content.BonusRule{}, 0, content.BlockInvalidMember},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := mk(c.c, c.r, c.strength)
			wantBlockError(t, err, "Armor", c.want)
		})
	}
	_, err = content.NewArmor("armor_x", " ", noSource, content.ArmorLight, base, 0, false, w, price)
	wantBlockError(t, err, "Armor", content.BlockInvalidName)
	if _, err := content.NewArmor("armor_x", "A", noSource, content.ArmorLight, base, 0, false, content.Weight{}, content.FixedCost{}); err == nil {
		t.Error("an armor with an invalid price was accepted")
	}
}
