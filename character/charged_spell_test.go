package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func sp(s string) content.SpellID { return content.MustSpellID(s) }

func lvl(n int) content.SpellLevel { return content.MustSpellLevel(n) }

func plan(t *testing.T, o character.ChargedSpellOption, charges int, lowest content.SpellLevel) character.ChargedSpellPlan {
	t.Helper()
	p, ok := must((character.ChargedSpellCasting{}).Plan(o, charges, lowest)).(character.ChargedSpellPlan)
	if !ok {
		t.Fatalf("plan rejected for %d charges", charges)
	}
	return p
}

// Requirement: doc/specs/magic-item-effects.md FR-029, FR-031, FR-034, FR-035 (SRD 5.2.1, "Wand of Magic Missiles", p. 251; "Staff of Healing", p. 245;
// "Staff of Fire", "Staff of Power", pp. 245-246; "Staff of the Magi", pp. 246-247; "Staff of the Woodlands", p. 247) - every row of table "Charged
// spell rows": the charges give the listed levels, a fixed cost casts at its lowest or named level, and the stats are carried.
// Path: per case.
func TestChargedSpellRows(t *testing.T) {
	t.Parallel()
	user := character.UsersSpellStats()
	type step struct{ charges, level int }
	rows := []struct {
		name   string
		option character.ChargedSpellOption
		lowest int
		steps  []step
	}{
		{"wand of magic missiles", character.MustChargesPerLevel(sp("magic_missile"), lvl(1), 3, user), 1, []step{{1, 1}, {2, 2}, {3, 3}}},
		{"wand of lightning bolts", character.MustChargesPerLevel(sp("lightning_bolt"), lvl(3), 3, character.FixedSaveDC(15)), 3, []step{{1, 3}, {2, 4}, {3, 5}}},
		{"staff of healing, cure wounds", character.MustChargesPerLevel(sp("cure_wounds"), lvl(1), 4, user), 1, []step{{1, 1}, {2, 2}, {3, 3}, {4, 4}}},
		{"staff of healing, lesser restoration", character.MustFixedChargeCost(sp("lesser_restoration"), 2, character.LowestLevel(), user), 2, []step{{2, 2}}},
		{"staff of healing, mass cure wounds", character.MustFixedChargeCost(sp("mass_cure_wounds"), 5, character.LowestLevel(), user), 5, []step{{5, 5}}},
		{"staff of fire, burning hands", character.MustFixedChargeCost(sp("burning_hands"), 1, character.LowestLevel(), user), 1, []step{{1, 1}}},
		{"staff of fire, fireball", character.MustFixedChargeCost(sp("fireball"), 3, character.LowestLevel(), user), 3, []step{{3, 3}}},
		{"staff of fire, wall of fire", character.MustFixedChargeCost(sp("wall_of_fire"), 4, character.LowestLevel(), user), 4, []step{{4, 4}}},
		{"staff of power, fireball", character.MustFixedChargeCost(sp("fireball"), 5, character.NamedLevel(lvl(5)), user), 3, []step{{5, 5}}},
		{"staff of power, lightning bolt", character.MustFixedChargeCost(sp("lightning_bolt"), 5, character.NamedLevel(lvl(5)), user), 3, []step{{5, 5}}},
		{"staff of power, globe", character.MustFixedChargeCost(sp("globe_of_invulnerability"), 6, character.LowestLevel(), user), 6, []step{{6, 6}}},
		{"staff of power, magic missile", character.MustFixedChargeCost(sp("magic_missile"), 1, character.LowestLevel(), user), 1, []step{{1, 1}}},
		{"staff of the magi, arcane lock", character.MustFixedChargeCost(sp("arcane_lock"), 0, character.LowestLevel(), user), 2, []step{{0, 2}}},
		{"staff of the magi, fireball", character.MustFixedChargeCost(sp("fireball"), 7, character.NamedLevel(lvl(7)), user), 3, []step{{7, 7}}},
		{"staff of the magi, plane shift", character.MustFixedChargeCost(sp("plane_shift"), 7, character.LowestLevel(), user), 7, []step{{7, 7}}},
		{"staff of the woodlands, awaken", character.MustFixedChargeCost(sp("awaken"), 5, character.LowestLevel(), user), 5, []step{{5, 5}}},
		{"staff of the woodlands, wall of thorns", character.MustFixedChargeCost(sp("wall_of_thorns"), 6, character.LowestLevel(), user), 6, []step{{6, 6}}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			for _, s := range r.steps {
				p := plan(t, r.option, s.charges, lvl(r.lowest))
				if p.Charges() != s.charges || p.CastAtLevel() != lvl(s.level) {
					t.Errorf("%d charges = level %d cost %d, want level %d", s.charges, p.CastAtLevel(), p.Charges(), s.level)
				}
			}
		})
	}
	p := plan(t, character.MustChargesPerLevel(sp("lightning_bolt"), lvl(3), 3, character.FixedSaveDC(15)), 1, lvl(3))
	if dc, ok := p.Stats().SaveDC(); !ok || dc != 15 || p.SpellID() != sp("lightning_bolt") {
		t.Errorf("plan = %+v", p)
	}
	if _, ok := plan(t, character.MustFixedChargeCost(sp("fireball"), 3, character.LowestLevel(), user), 3, lvl(3)).Stats().SaveDC(); ok {
		t.Error("user stats report a fixed DC")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-030 - a plan is rejected with chargesOutOfRange for 0 or more than the maximum, chargesNotAllowed for a
// count other than the fixed cost, and a named level below the spell's lowest is refused; 1 and the maximum are accepted.
// Path: unhappy (boundary).
func TestChargedSpellPlanRejections(t *testing.T) {
	t.Parallel()
	per := character.MustChargesPerLevel(sp("magic_missile"), lvl(1), 3, character.UsersSpellStats())
	fixed := character.MustFixedChargeCost(sp("fireball"), 3, character.LowestLevel(), character.UsersSpellStats())
	named := character.MustFixedChargeCost(sp("fireball"), 3, character.NamedLevel(lvl(2)), character.UsersSpellStats())
	for _, c := range []struct {
		name    string
		option  character.ChargedSpellOption
		charges int
		lowest  int
		reason  character.ChargedSpellPlanRejection
	}{
		{"per level, 0", per, 0, 1, character.ChargesOutOfRange},
		{"per level, max + 1", per, 4, 1, character.ChargesOutOfRange},
		{"per level, negative", per, -1, 1, character.ChargesOutOfRange},
		{"fixed, one below", fixed, 2, 3, character.ChargesNotAllowed},
		{"fixed, one above", fixed, 4, 3, character.ChargesNotAllowed},
		{"fixed, zero", fixed, 0, 3, character.ChargesNotAllowed},
		{"named level below the lowest", named, 3, 3, character.LevelBelowLowest},
	} {
		r, ok := must((character.ChargedSpellCasting{}).Plan(c.option, c.charges, lvl(c.lowest))).(character.ChargedSpellPlanRejected)
		if !ok || r.Reason() != c.reason {
			t.Errorf("%s = %+v", c.name, r)
		}
	}
	for _, charges := range []int{1, 3} {
		plan(t, per, charges, lvl(1))
	}
	plan(t, fixed, 3, lvl(3))
	plan(t, character.MustFixedChargeCost(sp("fireball"), 3, character.NamedLevel(lvl(3)), character.UsersSpellStats()), 3, lvl(3))
	if _, err := (character.ChargedSpellCasting{}).Plan(nil, 1, lvl(1)); err == nil {
		t.Error("nil option accepted")
	} else {
		wantInvalid(t, err, "nil option")
	}
	_, err := (character.ChargedSpellCasting{}).Plan(per, 1, content.SpellLevel(10))
	wantInvalid(t, err, "invalid lowest")
	if character.ChargesOutOfRange.String() != "charges_out_of_range" || character.ChargesNotAllowed.String() != "charges_not_allowed" || character.LevelBelowLowest.String() != "level_below_lowest" || character.ChargedSpellPlanRejection(0).String() != "unknown" {
		t.Error("rejection names")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-028 - the option constructors reject an invalid spell, negative charges, a maximum below 1, a level
// that would pass 9, an invalid level or stats; zero charges and the largest valid maximum are accepted.
// Path: unhappy (invalid input).
func TestChargedSpellOptionConstructors(t *testing.T) {
	t.Parallel()
	user := character.UsersSpellStats()
	_, e1 := character.NewFixedChargeCost("", 1, character.LowestLevel(), user)
	_, e2 := character.NewFixedChargeCost(sp("x"), -1, character.LowestLevel(), user)
	_, e3 := character.NewFixedChargeCost(sp("x"), 1, character.ItemSpellLevel{}, user)
	_, e4 := character.NewFixedChargeCost(sp("x"), 1, character.NamedLevel(content.SpellLevel(10)), user)
	_, e5 := character.NewFixedChargeCost(sp("x"), 1, character.LowestLevel(), character.ItemSpellStats{})
	_, e6 := character.NewFixedChargeCost(sp("x"), 1, character.LowestLevel(), character.FixedSaveDC(0))
	_, e7 := character.NewChargesPerLevel("", lvl(1), 1, user)
	_, e8 := character.NewChargesPerLevel(sp("x"), lvl(1), 0, user)
	_, e9 := character.NewChargesPerLevel(sp("x"), lvl(7), 4, user)
	_, e10 := character.NewChargesPerLevel(sp("x"), content.SpellLevel(10), 1, user)
	_, e11 := character.NewChargesPerLevel(sp("x"), lvl(1), 1, character.ItemSpellStats{})
	for i, err := range []error{e1, e2, e3, e4, e5, e6, e7, e8, e9, e10, e11} {
		wantInvalid(t, err, "case "+string(rune('a'+i)))
	}
	free := must(character.NewFixedChargeCost(sp("light"), 0, character.LowestLevel(), user))
	top := must(character.NewChargesPerLevel(sp("x"), lvl(7), 3, user))
	cantrip := must(character.NewFixedChargeCost(sp("light"), 0, character.NamedLevel(lvl(0)), user))
	if free.Charges() != 0 || free.SpellID() != sp("light") || free.Level() != character.LowestLevel() || free.Stats() != user ||
		top.MaximumCharges() != 3 || top.LowestLevel() != lvl(7) || top.SpellID() != sp("x") || top.Stats() != user || cantrip.Level() != character.NamedLevel(lvl(0)) {
		t.Errorf("accessors: %+v %+v", free, top)
	}
	for name, f := range map[string]func(){
		"fixed": func() { character.MustFixedChargeCost("", 1, character.LowestLevel(), user) },
		"per":   func() { character.MustChargesPerLevel("", lvl(1), 1, user) },
	} {
		func() {
			defer func() {
				if err, ok := recover().(error); !ok {
					t.Errorf("%s: panic value is not an error", name)
				} else {
					wantInvalid(t, err, name)
				}
			}()
			f()
		}()
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-032, FR-033, FR-034, FR-036 (SRD 5.2.1, "Spells Cast from Items", p. 206; "Charges", p. 206) - Cast
// spends through ItemCharges.Spend and returns the new charges, the MagicItemChargesSpent event and a SpellFromItem with delivery CastFromItem, default
// components and terms, the lowest caster level, the cast level and Concentration as asked; too few charges returns MagicItemInsufficientCharges
// unchanged; a free spell spends nothing and has no event; no spell slot appears in the result.
// Path: unhappy (state error).
func TestChargedSpellCastingCast(t *testing.T) {
	t.Parallel()
	per := character.MustChargesPerLevel(sp("magic_missile"), lvl(1), 3, character.UsersSpellStats())
	cast := func(c character.ItemCharges, p character.ChargedSpellPlan, conc bool) character.ChargedSpellCastResult {
		return must((character.ChargedSpellCasting{}).Cast(c, p, holder, inst("wand-1"), conc))
	}
	t.Run("paid", func(t *testing.T) {
		t.Parallel()
		r, ok := cast(character.MustItemCharges(7, 5), plan(t, per, 3, lvl(1)), true).(character.ChargedSpellCastApplied)
		if !ok {
			t.Fatal("not applied")
		}
		ev, spent := r.Event()
		if r.Charges() != character.MustItemCharges(7, 2) || !spent || ev.Spent() != 3 || ev.CharacterID() != holder || ev.InstanceID() != inst("wand-1") {
			t.Errorf("result = %+v %+v", r.Charges(), ev)
		}
		s := r.Spell()
		if s.SpellID() != sp("magic_missile") || s.CastAtLevel() != lvl(3) || s.CasterLevel() != character.ItemCasterLevelLowest || s.Delivery() != character.CastFromItem ||
			s.Components() != character.NoComponentsRequired || s.CastingTerms() != character.StandardCastingTerms || !s.UserMustConcentrate() {
			t.Errorf("spell = %+v", s)
		}
		if r.Stats() != character.UsersSpellStats() {
			t.Error("stats")
		}
	})
	t.Run("no concentration asked", func(t *testing.T) {
		t.Parallel()
		r := cast(character.MustItemCharges(7, 5), plan(t, per, 1, lvl(1)), false).(character.ChargedSpellCastApplied)
		if r.Spell().UserMustConcentrate() {
			t.Error("concentration set")
		}
	})
	t.Run("exactly the charges left", func(t *testing.T) {
		t.Parallel()
		r := cast(character.MustItemCharges(7, 3), plan(t, per, 3, lvl(1)), false).(character.ChargedSpellCastApplied)
		if r.Charges().Current() != 0 {
			t.Errorf("charges = %v", r.Charges())
		}
	})
	t.Run("one too few", func(t *testing.T) {
		t.Parallel()
		r, ok := cast(character.MustItemCharges(7, 2), plan(t, per, 3, lvl(1)), false).(character.MagicItemInsufficientCharges)
		if !ok || r.Current != 2 || r.Requested != 3 {
			t.Errorf("result = %+v", r)
		}
	})
	t.Run("free spell", func(t *testing.T) {
		t.Parallel()
		free := character.MustFixedChargeCost(sp("detect_magic"), 0, character.LowestLevel(), character.UsersSpellStats())
		charges := character.MustItemCharges(50, 0)
		r := cast(charges, plan(t, free, 0, lvl(1)), true).(character.ChargedSpellCastApplied)
		if _, spent := r.Event(); spent || r.Charges() != charges || r.Spell().CastAtLevel() != lvl(1) {
			t.Errorf("result = %+v", r)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()
		good := plan(t, per, 1, lvl(1))
		for name, f := range map[string]func() error{
			"zero plan": func() error {
				return second((character.ChargedSpellCasting{}).Cast(character.MustItemCharges(7, 5), character.ChargedSpellPlan{}, holder, inst("w"), false))
			},
			"zero holder": func() error {
				return second((character.ChargedSpellCasting{}).Cast(character.MustItemCharges(7, 5), good, character.Identifier{}, inst("w"), false))
			},
			"blank instance": func() error {
				return second((character.ChargedSpellCasting{}).Cast(character.MustItemCharges(7, 5), good, holder, "", false))
			},
			"zero charges": func() error {
				return second((character.ChargedSpellCasting{}).Cast(character.ItemCharges{}, good, holder, inst("w"), false))
			},
		} {
			wantInvalid(t, f(), name)
		}
	})
}

// Requirement: doc/specs/magic-item-effects.md FR-035 - the stats values: the user's own, or a printed DC that must be positive.
// Path: unhappy (boundary).
func TestItemSpellStats(t *testing.T) {
	t.Parallel()
	if dc, ok := character.FixedSaveDC(13).SaveDC(); !ok || dc != 13 {
		t.Error("fixed 13")
	}
	if _, ok := character.UsersSpellStats().SaveDC(); ok {
		t.Error("user stats have a DC")
	}
	if _, err := character.NewFixedChargeCost(sp("web"), 1, character.LowestLevel(), character.FixedSaveDC(1)); err != nil {
		t.Errorf("DC 1: %v", err)
	}
}
