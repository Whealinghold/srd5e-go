package character_test

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sorChanged returns the changed transition or fails.
func sorChanged(t *testing.T, tr character.SorcererTransition) character.SorcererChanged {
	t.Helper()
	c, ok := tr.(character.SorcererChanged)
	if !ok {
		t.Fatalf("transition = %#v, want SorcererChanged", tr)
	}
	return c
}

// sorRefusedAs returns the rejection of a refused transition or fails.
func sorRefusedAs(t *testing.T, tr character.SorcererTransition) character.SorcererRejection {
	t.Helper()
	r, ok := tr.(character.SorcererRefused)
	if !ok {
		t.Fatalf("transition = %#v, want SorcererRefused", tr)
	}
	return r.Rejection()
}

// sorSlots is a SpellcastingState for a Sorcerer of level n, built through the public use case.
func sorSlots(t *testing.T, n int) character.SpellcastingState {
	t.Helper()
	prepared := append(spellIDs(1, 12), spellIDs(2, 10)...)
	count := character.PreparedLimit(character.CasterSorcerer, n)
	setup := character.ClassSpellcastingSetup{Class: character.CasterSorcerer, Cantrips: spellIDs(0, character.CantripLimit(character.CasterSorcerer, n)), Prepared: prepared[:count]}
	return castingState(t, classLevelsOf(t, "sorcerer", n), spellCatalogue(t), setup)
}

// Requirement: doc/specs/class-sorcerer.md FR-037 to FR-041, FR-044 (SRD 5.2.1, "Level 1: Innate Sorcery", pp. 65-66; "Level 7: Sorcery Incarnate", p. 66) - Innate Sorcery is activated by a use or, from level 7 with none left, by 2 Sorcery Points;
// each refusal leaves the state unchanged.
// Path: per case (happy / unhappy: state, boundary).
func TestActivateInnateSorcery(t *testing.T) {
	t.Parallel()
	use := character.InnateSorceryNormalUse
	pts := character.InnateSorceryPaidWithPoints
	type result struct {
		uses, points int
		paid         bool
	}
	for _, c := range []struct {
		name, requirement string
		profile           int
		edit              func(*character.SorcererStateParts)
		points            int
		pay               character.InnateSorceryPayment
		kind              character.SorcererRejectionKind
		want              result
		path              string
	}{
		{"a use is consumed", "FR-037, FR-038", 3, nil, 3, use, 0, result{1, 3, false}, "happy"},
		{"the last use is consumed", "FR-038", 3, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 1 }, 3, use, 0, result{0, 3, false}, "unhappy (boundary)"},
		{"no uses left below level 7", "FR-039", 6, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 0 }, 6, use, character.SorcererInnateSorceryNoUsesLeft, result{}, "unhappy (state)"},
		{"no uses left at level 7 by the normal route", "FR-039", 7, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 0 }, 7, use, character.SorcererInnateSorceryNoUsesLeft, result{}, "unhappy (state)"},
		{"already active", "FR-044", 3, func(p *character.SorcererStateParts) { p.InnateSorceryActive = true }, 3, use, character.SorcererInnateSorceryAlreadyActive, result{}, "unhappy (state)"},
		{"already active, points route", "FR-044", 7, func(p *character.SorcererStateParts) { p.InnateSorceryActive, p.InnateSorceryUses = true, 0 }, 7, pts, character.SorcererInnateSorceryAlreadyActive, result{}, "unhappy (state)"},
		{"2 points with no uses at level 7", "FR-040", 7, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 0 }, 7, pts, 0, result{0, 5, true}, "happy"},
		{"exactly 2 points", "FR-040, FR-041", 7, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 0 }, 2, pts, 0, result{0, 0, true}, "unhappy (boundary)"},
		{"1 point", "FR-041", 7, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 0 }, 1, pts, character.SorcererSorceryPointsInsufficient, result{}, "unhappy (boundary)"},
		{"uses remain", "FR-041", 7, nil, 7, pts, character.SorcererInnateSorceryUsesRemain, result{}, "unhappy (state)"},
		{"below level 7", "FR-041", 6, func(p *character.SorcererStateParts) { p.InnateSorceryUses = 0 }, 6, pts, character.SorcererSorceryIncarnateUnavailable, result{}, "unhappy (boundary)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := sorProfile(c.profile, "")
			s := sorWith(p, c.points, nil, c.edit)
			tr := character.ActivateInnateSorcery(sorID, s, p, c.pay)
			if c.kind != 0 {
				if rej := sorRefusedAs(t, tr); rej.Kind != c.kind {
					t.Errorf("kind = %v, want %v [%s %s]", rej.Kind, c.kind, c.requirement, c.path)
				}
				if c.kind == character.SorcererSorceryPointsInsufficient {
					if rej := sorRefusedAs(t, tr); rej.Required != 2 || rej.Available != c.points {
						t.Errorf("rejection = %+v, want Required 2 and Available %d", rej, c.points)
					}
				}
				return
			}
			ch := sorChanged(t, tr)
			ev, ok := ch.Events()[0].(character.InnateSorceryActivated)
			if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.PaidWithPoints() != c.want.paid {
				t.Errorf("events = %#v, want one InnateSorceryActivated paid %v", ch.Events(), c.want.paid)
			}
			if got := ch.State(); !got.InnateSorceryActive() || got.InnateSorceryUses() != c.want.uses || got.SorceryPoints() != c.want.points {
				t.Errorf("state = %+v, want active, %d uses and %d points [%s %s]", got.Parts(), c.want.uses, c.want.points, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-043 (SRD 5.2.1, "Level 1: Innate Sorcery", p. 65) - the host ends Innate Sorcery; ending an inactive one is refused with SorcererInnateSorceryNotActive; ending does not clear the per-turn flags (FR-109).
// Path: happy / unhappy (state).
func TestEndInnateSorcery(t *testing.T) {
	t.Parallel()
	p := sorProfile(20, "")
	active := sorWith(p, 20, nil, func(s *character.SorcererStateParts) {
		s.InnateSorceryActive, s.FreeMetamagicSpent, s.QuickenedUsedThisTurn = true, true, true
	})
	ch := sorChanged(t, character.EndInnateSorcery(sorID, active))
	if ev, ok := ch.Events()[0].(character.InnateSorceryEnded); len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID {
		t.Errorf("events = %#v, want one InnateSorceryEnded", ch.Events())
	}
	if s := ch.State(); s.InnateSorceryActive() || !s.FreeMetamagicSpent() || !s.QuickenedUsedThisTurn() {
		t.Errorf("state = %+v, want inactive with the per-turn flags kept", s.Parts())
	}
	if rej := sorRefusedAs(t, character.EndInnateSorcery(sorID, sorFresh(p))); rej.Kind != character.SorcererInnateSorceryNotActive {
		t.Errorf("kind = %v, want SorcererInnateSorceryNotActive", rej.Kind)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-046, FR-109 - starting a turn clears the free Metamagic use spent and Quickened Spell used, and nothing else.
// Path: happy.
func TestStartSorcererTurn(t *testing.T) {
	t.Parallel()
	p := sorProfile(20, "")
	s := sorWith(p, 11, nil, func(s *character.SorcererStateParts) {
		s.FreeMetamagicSpent, s.QuickenedUsedThisTurn, s.InnateSorceryActive, s.RestorationUsed = true, true, true, true
	})
	got := character.StartSorcererTurn(s)
	want := sorWith(p, 11, nil, func(s *character.SorcererStateParts) { s.InnateSorceryActive, s.RestorationUsed = true, true })
	if got != want || !s.FreeMetamagicSpent() {
		t.Errorf("state = %+v, want %+v and the input unchanged", got.Parts(), want.Parts())
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-037, FR-133 (SRD 5.2.1, "Level 1: Innate Sorcery", pp. 65-66) - while active the effects are a +1 Modifier on the save DC and Advantage on attack rolls; inactive, a Modifier of 0 and no circumstance.
// Path: happy / unhappy (state).
func TestSorcererInnateSorceryEffects(t *testing.T) {
	t.Parallel()
	p := sorProfile(3, "")
	on := character.SorcererInnateSorceryEffects(sorWith(p, 3, nil, func(s *character.SorcererStateParts) { s.InnateSorceryActive = true }))
	if on.SaveDC != modifier.New(1) || on.AttackRolls != (modifier.Advantage{}) {
		t.Errorf("active effects = %+v, want +1 and Advantage", on)
	}
	off := character.SorcererInnateSorceryEffects(sorFresh(p))
	if off.SaveDC != modifier.New(0) || off.AttackRolls != nil {
		t.Errorf("inactive effects = %+v, want 0 and none", off)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-091, FR-092, FR-135 - the Resistance is the chosen Elemental Affinity type (none before a choice) and the Fly Speed is 60 feet while Dragon Wings is active and absent otherwise.
// Path: per case (happy / unhappy: empty).
func TestSorcererEffects(t *testing.T) {
	t.Parallel()
	p := sorDraconic(14)
	empty := character.SorcererEffects(sorFresh(p))
	if len(empty.Resistances()) != 0 || empty.Resistances() == nil || empty.FlySpeedFeet != 0 {
		t.Errorf("fresh effects = %v and %d, want none", empty.Resistances(), empty.FlySpeedFeet)
	}
	s := sorWith(p, 14, nil, func(s *character.SorcererStateParts) {
		s.ElementalAffinity, s.DragonWingsUsed, s.DragonWingsActive = content.DamageLightning, true, true
	})
	e := character.SorcererEffects(s)
	if !slices.Equal(e.Resistances(), []content.DamageType{content.DamageLightning}) || e.FlySpeedFeet != 60 {
		t.Errorf("effects = %v and %d, want Lightning and 60", e.Resistances(), e.FlySpeedFeet)
	}
	e.Resistances()[0] = content.DamageFire
	if got := e.Resistances()[0]; got != content.DamageLightning {
		t.Error("Resistances returned a shared slice")
	}
	off := sorWith(p, 14, nil, func(s *character.SorcererStateParts) { s.DragonWingsUsed = true })
	if got := character.SorcererEffects(off).FlySpeedFeet; got != 0 {
		t.Errorf("wings in: Fly Speed = %d, want 0", got)
	}
}

// Requirement: doc/specs/class-sorcerer.md Table V, FR-050, FR-051, FR-114 (SRD 5.2.1, "Converting Spell Slots to Sorcery Points", "Level 2: Font of Magic", p. 66) - converting a slot adds its level in points, never above the
// maximum and never capped; a missing slot, a slot level outside 1-9 and a level 1 Sorcerer are refused.
// Path: per case (Table V).
func TestConvertSpellSlot(t *testing.T) {
	t.Parallel()
	type conv struct {
		name, requirement, path string
		level, points, slot     int
		created                 bool
		spend                   int // level 3 slots expended first
		kind                    character.SorcererRejectionKind
		invalid                 bool
		wantPoints              int
	}
	for _, c := range []conv{
		{name: "V-1", requirement: "FR-050", path: "happy", level: 5, points: 0, slot: 1, wantPoints: 1},
		{name: "V-2", requirement: "FR-050", path: "happy", level: 20, points: 0, slot: 9, wantPoints: 9},
		{name: "V-3", requirement: "FR-114", path: "unhappy (invalid input)", level: 5, slot: 0, invalid: true},
		{name: "V-4", requirement: "FR-114", path: "unhappy (invalid input)", level: 5, slot: 10, invalid: true},
		{name: "V-5", requirement: "FR-051", path: "unhappy (boundary)", level: 20, points: 18, slot: 5, kind: character.SorcererSorceryPointsWouldExceedMaximum},
		{name: "V-6", requirement: "FR-050", path: "unhappy (boundary)", level: 20, points: 15, slot: 5, wantPoints: 20},
		{name: "V-7", requirement: "FR-051", path: "unhappy (state)", level: 5, points: 0, slot: 3, spend: 2, kind: character.SorcererNoSlotToConvert},
		{name: "V-8", requirement: "FR-050", path: "happy", level: 5, points: 0, slot: 3, created: true, wantPoints: 3},
		{name: "no created slot of the level", requirement: "FR-051", path: "unhappy (state)", level: 5, slot: 2, created: true, kind: character.SorcererNoSlotToConvert},
		{name: "a created slot of level 6", requirement: "FR-051", path: "unhappy (state)", level: 20, slot: 6, created: true, kind: character.SorcererNoSlotToConvert},
		{name: "level 2 has Font of Magic", requirement: "FR-019", path: "unhappy (boundary)", level: 2, points: 0, slot: 1, wantPoints: 1},
		{name: "level 1 has no Font of Magic", requirement: "FR-019", path: "unhappy (boundary)", level: 1, slot: 1, kind: character.SorcererFeatureNotGained},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := sorProfile(c.level, "")
			s := sorWith(p, c.points, nil, func(s *character.SorcererStateParts) {
				if c.created {
					s.CreatedSlots[2] = 1 // a created level 3 slot
				}
			})
			sp := sorSlots(t, c.level)
			for range c.spend {
				sp = sp.ExpendSlot(character.PoolSpellcasting, character.MustSlotLevel(3)).(character.SlotExpenditureExpended).State()
			}
			var src character.SlotConversionSource = character.ConvertOrdinarySlot{State: sp}
			if c.created {
				src = character.ConvertCreatedSlot{}
			}
			tr, err := character.ConvertSpellSlot(sorID, s, p, src, c.slot)
			if c.invalid {
				if !errors.Is(err, srderr.ErrInvalidInput) || tr != nil {
					t.Errorf("err = %v, want ErrInvalidInput [%s]", err, c.requirement)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.kind != 0 {
				r, ok := tr.(character.SorcererSlotsRefused)
				if !ok || r.Rejection().Kind != c.kind {
					t.Fatalf("transition = %#v, want SorcererSlotsRefused %v [%s %s]", tr, c.kind, c.requirement, c.path)
				}
				if c.kind == character.SorcererSorceryPointsWouldExceedMaximum {
					if rej := r.Rejection(); rej.Cap != 20 || rej.Available != 18 || rej.Required != 5 {
						t.Errorf("rejection = %+v, want Cap 20, Available 18, Required 5", rej)
					}
				}
				return
			}
			ch, ok := tr.(character.SorcererSlotsChanged)
			if !ok || ch.State().SorceryPoints() != c.wantPoints {
				t.Fatalf("transition = %#v, want %d points [%s %s]", tr, c.wantPoints, c.requirement, c.path)
			}
			events := ch.Events()
			last, isConv := events[len(events)-1].(character.SpellSlotConvertedToSorceryPoints)
			if !isConv || last.CharacterID() != sorID || last.SlotLevel() != c.slot || last.PointsGained() != c.slot || last.WasCreatedSlot() != c.created {
				t.Errorf("events = %#v, want a conversion of a level %d slot (created %v)", events, c.slot, c.created)
			}
			view := ch.Spellcasting().Slots(character.PoolSpellcasting)
			before := sp.Slots(character.PoolSpellcasting)
			l := character.MustSlotLevel(c.slot)
			switch {
			case c.created:
				if len(events) != 1 || ch.State().CreatedSlots(3) != 0 || !ch.Spellcasting().IsZero() || view != (character.SlotPoolView{}) {
					t.Errorf("created conversion: events %d, created %d, spellcasting state %v", len(events), ch.State().CreatedSlots(3), ch.Spellcasting())
				}
			default:
				if _, expended := events[0].(character.SpellSlotExpended); len(events) != 2 || !expended || view.Remaining(l) != before.Remaining(l)-1 {
					t.Errorf("ordinary conversion: events %#v, remaining %d of %d", events, view.Remaining(l), before.Remaining(l))
				}
			}
			if s.SorceryPoints() != c.points {
				t.Error("the input state changed")
			}
		})
	}
	if _, err := character.ConvertSpellSlot(sorID, sorFresh(sorProfile(5, "")), sorProfile(5, ""), nil, 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil source err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-052, Table C (SRD 5.2.1, "Creating Spell Slots", p. 66) - the rule for a slot level is read from the Creating Spell Slots table; a level the table lacks and a row that is not three integers give none.
// Path: per case (happy / unhappy: missing row, ragged content).
func TestSlotCreationRuleFor(t *testing.T) {
	t.Parallel()
	var table content.FeatureTable
	for _, f := range sorDef(t).Features() {
		if len(f.Tables()) == 1 {
			table = f.Tables()[0]
		}
	}
	for _, c := range []struct{ level, cost, min int }{{1, 2, 2}, {2, 3, 3}, {3, 5, 5}, {4, 6, 7}, {5, 7, 9}} {
		if got, ok := character.SlotCreationRuleFor(table, c.level); !ok || got != (character.SlotCreationRule{Cost: c.cost, MinSorcererLevel: c.min}) {
			t.Errorf("rule for %d = (%+v, %v), want cost %d, minimum %d", c.level, got, ok, c.cost, c.min)
		}
	}
	if _, ok := character.SlotCreationRuleFor(table, 6); ok {
		t.Error("the table has no row for level 6")
	}
	text := func(s string) content.TableCell { return must(content.NewTableCellText(s)) }
	n := func(i int) content.TableCell { return must(content.NewTableCellInt(i)) }
	odd := must(content.NewFeatureTable("Odd", []string{"A", "B", "C"}, [][]content.TableCell{{n(1), text("x"), n(2)}, {n(2), text("y"), n(3)}}))
	for _, level := range []int{1, 2} {
		if _, ok := character.SlotCreationRuleFor(odd, level); ok {
			t.Errorf("a row with a text cell gives no rule (level %d)", level)
		}
	}
	short := must(content.NewFeatureTable("Short", []string{"A", "B"}, [][]content.TableCell{{n(1), n(2)}}))
	if _, ok := character.SlotCreationRuleFor(short, 1); ok {
		t.Error("a two column table gives no rule")
	}
}

// Requirement: doc/specs/class-sorcerer.md Table C, FR-052 to FR-054, FR-114 (SRD 5.2.1, "Creating Spell Slots", p. 66) - a slot of level 1 to 5 costs the table's points and needs its minimum Sorcerer level; levels 6 to 9, too few
// points and too low a level are refused with state unchanged; a level outside 1-9 is an error.
// Path: per case (Table C).
func TestCreateSlot(t *testing.T) {
	t.Parallel()
	var table content.FeatureTable
	for _, f := range sorDef(t).Features() {
		if len(f.Tables()) == 1 {
			table = f.Tables()[0]
		}
	}
	for _, c := range []struct {
		name, requirement, path string
		level, points, slot     int
		kind                    character.SorcererRejectionKind
		invalid                 bool
		left                    int
	}{
		{name: "C-1", requirement: "FR-052", path: "happy", level: 2, points: 2, slot: 1, left: 0},
		{name: "C-2", requirement: "FR-052", path: "happy", level: 3, points: 3, slot: 2, left: 0},
		{name: "C-3", requirement: "FR-052", path: "happy", level: 5, points: 5, slot: 3, left: 0},
		{name: "C-4", requirement: "FR-052", path: "happy", level: 7, points: 7, slot: 4, left: 1},
		{name: "C-5", requirement: "FR-052", path: "happy", level: 9, points: 9, slot: 5, left: 2},
		{name: "C-6", requirement: "FR-053", path: "unhappy (boundary)", level: 20, points: 20, slot: 6, kind: character.SorcererSlotLevelNotCreatable},
		{name: "C-7", requirement: "FR-053", path: "unhappy (boundary)", level: 6, points: 6, slot: 4, kind: character.SorcererLevelBelowMinimum},
		{name: "C-8", requirement: "FR-053", path: "unhappy (boundary)", level: 5, points: 4, slot: 3, kind: character.SorcererSorceryPointsInsufficient},
		{name: "C-9", requirement: "FR-053", path: "unhappy (boundary)", level: 5, points: 5, slot: 3, left: 0},
		{name: "C-10", requirement: "FR-114", path: "unhappy (invalid input)", level: 5, points: 5, slot: 0, invalid: true},
		{name: "C-11", requirement: "FR-114", path: "unhappy (invalid input)", level: 5, points: 5, slot: 10, invalid: true},
		{name: "C-12", requirement: "FR-053", path: "unhappy (boundary)", level: 20, points: 20, slot: 9, kind: character.SorcererSlotLevelNotCreatable},
		{name: "level 1 has no Font of Magic", requirement: "FR-019", path: "unhappy (boundary)", level: 1, points: 0, slot: 1, kind: character.SorcererFeatureNotGained},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := sorProfile(c.level, "")
			s := sorWith(p, c.points, nil, nil)
			rule, found := character.SlotCreationRuleFor(table, c.slot)
			tr, err := character.CreateSlot(sorID, s, p, c.slot, rule, found)
			if c.invalid {
				if !errors.Is(err, srderr.ErrInvalidInput) || tr != nil {
					t.Errorf("err = %v, want ErrInvalidInput [%s]", err, c.requirement)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.kind != 0 {
				rej := sorRefusedAs(t, tr)
				if rej.Kind != c.kind {
					t.Errorf("kind = %v, want %v [%s %s]", rej.Kind, c.kind, c.requirement, c.path)
				}
				if c.kind == character.SorcererLevelBelowMinimum && (rej.Required != 7 || rej.Available != 6) {
					t.Errorf("rejection = %+v, want Required 7 and Available 6", rej)
				}
				if c.kind == character.SorcererSorceryPointsInsufficient && (rej.Required != 5 || rej.Available != 4) {
					t.Errorf("rejection = %+v, want Required 5 and Available 4", rej)
				}
				return
			}
			ch := sorChanged(t, tr)
			ev, ok := ch.Events()[0].(character.SpellSlotCreated)
			if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.SlotLevel() != c.slot || ev.Cost() != rule.Cost {
				t.Errorf("events = %#v, want one SpellSlotCreated of level %d costing %d", ch.Events(), c.slot, rule.Cost)
			}
			if got := ch.State(); got.CreatedSlots(c.slot) != 1 || got.SorceryPoints() != c.left {
				t.Errorf("state = %+v, want one created slot of level %d and %d points left [%s %s]", got.Parts(), c.slot, c.left, c.requirement, c.path)
			}
		})
	}
	p := sorProfile(20, "")
	if rej := sorRefusedAs(t, must(character.CreateSlot(sorID, sorFresh(p), p, 3, character.SlotCreationRule{}, false))); rej.Kind != character.SorcererSlotLevelNotCreatable {
		t.Errorf("a level the table lacks: kind = %v, want SorcererSlotLevelNotCreatable", rej.Kind)
	}
}

// Requirement: doc/specs/class-sorcerer.md Table C rows C-13 to C-15, FR-136, FR-114 - a created slot is spent on a spell of its level or lower; none of the level, a spell above the slot and a slot already spent this turn are refused; a level outside 1-9 is an error.
// Path: per case (Table C).
func TestSpendCreatedSlot(t *testing.T) {
	t.Parallel()
	p := sorProfile(9, "")
	held := sorWith(p, 0, nil, func(s *character.SorcererStateParts) { s.CreatedSlots = [5]int{1, 0, 2, 0, 0} })
	for _, c := range []struct {
		name, requirement, path string
		created, spell          int
		spent                   bool
		kind                    character.SorcererRejectionKind
		invalid                 bool
	}{
		{"C-13", "FR-136", "happy", 3, 3, false, 0, false},
		{"C-14", "FR-136", "unhappy (boundary)", 3, 1, false, 0, false},
		{"C-15 none of the level", "FR-136", "unhappy (state)", 2, 1, false, character.SorcererCreatedSlotUnavailable, false},
		{"C-15 spell above the slot", "FR-136", "unhappy (state)", 1, 3, false, character.SorcererCreatedSlotLevelTooLow, false},
		{"C-15 a slot spent this turn", "FR-136", "unhappy (state)", 3, 3, true, character.SorcererCreatedSlotSpentThisTurn, false},
		{"a created level 6", "FR-136", "unhappy (boundary)", 6, 6, false, character.SorcererCreatedSlotUnavailable, false},
		{"a created level 0", "FR-114", "unhappy (invalid input)", 0, 1, false, 0, true},
		{"a created level 10", "FR-114", "unhappy (invalid input)", 10, 1, false, 0, true},
		{"a spell level 0", "FR-114", "unhappy (invalid input)", 3, 0, false, 0, true},
		{"a spell level 10", "FR-114", "unhappy (invalid input)", 3, 10, false, 0, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := character.SpendCreatedSlot(sorID, held, sorCasting(c.spent), c.created, c.spell)
			if c.invalid {
				if !errors.Is(err, srderr.ErrInvalidInput) || tr != nil {
					t.Errorf("err = %v, want ErrInvalidInput [%s]", err, c.requirement)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.kind != 0 {
				if rej := sorRefusedAs(t, tr); rej.Kind != c.kind {
					t.Errorf("kind = %v, want %v [%s %s]", rej.Kind, c.kind, c.requirement, c.path)
				}
				return
			}
			ch := sorChanged(t, tr)
			ev, ok := ch.Events()[0].(character.CreatedSlotSpent)
			if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.SlotLevel() != c.created || ev.SpellLevel() != c.spell {
				t.Errorf("events = %#v, want one CreatedSlotSpent %d on %d", ch.Events(), c.created, c.spell)
			}
			if got := ch.State().CreatedSlots(3); got != 1 || held.CreatedSlots(3) != 2 {
				t.Errorf("created level 3 slots = %d (input %d), want 1 (input 2) [%s %s]", got, held.CreatedSlots(3), c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md Table R, FR-083, FR-084, FR-106 to FR-108 (SRD 5.2.1, "Level 5: Sorcerous Restoration", p. 66) - on a Short Rest at level 5+ the Sorcerer regains points up to half its level rounded
// down, never above the points missing and once until a Long Rest; every refusal leaves the whole state unchanged and does not consume the use.
// Path: per case (Table R).
func TestApplySorcererRestShortRest(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		level, missing, ask     int
		kind                    character.SorcererRejectionKind
		cap, wantMissing, gain  int
		used                    bool
	}{
		{name: "R-1", requirement: "FR-083", path: "happy", level: 5, missing: 5, ask: 2, gain: 2},
		{name: "R-2", requirement: "FR-084", path: "unhappy (boundary)", level: 5, missing: 5, ask: 3, kind: character.SorcererRestorationAmountInvalid, cap: 2, wantMissing: 5},
		{name: "R-3", requirement: "FR-084", path: "unhappy (boundary)", level: 4, missing: 4, ask: 2, kind: character.SorcererFeatureNotGained},
		{name: "R-4", requirement: "FR-083", path: "unhappy (boundary)", level: 7, missing: 7, ask: 3, gain: 3},
		{name: "R-5", requirement: "FR-083", path: "unhappy (boundary)", level: 20, missing: 20, ask: 10, gain: 10},
		{name: "R-6", requirement: "FR-108", path: "unhappy (boundary)", level: 20, missing: 4, ask: 10, kind: character.SorcererRestorationAmountInvalid, cap: 10, wantMissing: 4},
		{name: "R-7", requirement: "FR-108", path: "unhappy (state)", level: 5, missing: 0, ask: 1, kind: character.SorcererRestorationAmountInvalid, cap: 2, wantMissing: 0},
		{name: "R-8", requirement: "FR-083", path: "unhappy (boundary)", level: 20, missing: 4, ask: 4, gain: 4},
		{name: "R-9", requirement: "FR-108", path: "unhappy (boundary)", level: 5, missing: 5, ask: 0, kind: character.SorcererRestorationAmountInvalid, cap: 2, wantMissing: 5},
		{name: "a negative request", requirement: "FR-108", path: "unhappy (invalid input)", level: 5, missing: 5, ask: -1, kind: character.SorcererRestorationAmountInvalid, cap: 2, wantMissing: 5},
		{name: "R-12 already used", requirement: "FR-084", path: "unhappy (state)", level: 5, missing: 5, ask: 1, used: true, kind: character.SorcererRestorationAlreadyUsed},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := sorProfile(c.level, "")
			max := p.Level().SorceryPointsMaximum()
			s := sorWith(p, max-c.missing, nil, func(s *character.SorcererStateParts) { s.RestorationUsed = c.used })
			tr, err := character.ApplySorcererRest(sorID, s, p, character.ShortRest, c.ask)
			if err != nil {
				t.Fatal(err)
			}
			if c.kind != 0 {
				rej := sorRefusedAs(t, tr)
				if rej.Kind != c.kind || rej.Cap != c.cap || rej.Missing != c.wantMissing {
					t.Errorf("rejection = %+v, want %v with Cap %d and Missing %d [%s %s]", rej, c.kind, c.cap, c.wantMissing, c.requirement, c.path)
				}
				return
			}
			ch := sorChanged(t, tr)
			ev, ok := ch.Events()[0].(character.SorceryPointsRestored)
			if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.Amount() != c.gain || ev.Total() != max-c.missing+c.gain {
				t.Errorf("events = %#v, want %d regained, %d in all", ch.Events(), c.gain, max-c.missing+c.gain)
			}
			if got := ch.State(); got.SorceryPoints() != max-c.missing+c.gain || !got.RestorationUsed() {
				t.Errorf("state = %+v, want %d points and the use consumed [%s %s]", got.Parts(), max-c.missing+c.gain, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md Table R rows R-10, R-11, R-12, FR-042, FR-049, FR-054, FR-093, FR-095, FR-106, FR-107 (SRD 5.2.1, "Level 5: Sorcerous Restoration", "Creating Spell Slots", "Level 2: Font of Magic", p. 66) - a Long Rest restores
// every Sorcery Point, both Innate Sorcery uses, Sorcerous Restoration, Dragon Wings and Dragon Companion and removes the created slots; a Short Rest keeps the created slots; Restoration with a Long Rest is refused; a refused attempt keeps the use.
// Path: per case (happy / unhappy: state).
func TestApplySorcererRestLongRest(t *testing.T) {
	t.Parallel()
	p := sorDraconic(18)
	spent := sorWith(p, 3, []character.MetamagicEffect{character.MetamagicCareful}, func(s *character.SorcererStateParts) {
		s.InnateSorceryUses, s.RestorationUsed, s.DragonWingsUsed, s.DragonCompanionFreeCastUsed = 0, true, true, true
		s.CreatedSlots = [5]int{1, 0, 2, 0, 0}
		s.ElementalAffinity = content.DamageFire
		s.InnateSorceryActive = true
	})
	ch := sorChanged(t, must(character.ApplySorcererRest(sorID, spent, p, character.LongRest, 0)))
	got := ch.State()
	if got.SorceryPoints() != 18 || got.InnateSorceryUses() != 2 || got.RestorationUsed() || got.DragonWingsUsed() || got.DragonCompanionFreeCastUsed() || got.CreatedSlots(1)+got.CreatedSlots(3) != 0 {
		t.Errorf("after a Long Rest = %+v, want everything restored and no created slot", got.Parts())
	}
	if !got.Knows(character.MetamagicCareful) || !got.InnateSorceryActive() {
		t.Error("a Long Rest must keep the Metamagic known and not touch Innate Sorcery's active flag")
	}
	if t0, _ := got.ElementalAffinity(); t0 != content.DamageFire {
		t.Error("a Long Rest must keep the Elemental Affinity type")
	}
	ev, ok := ch.Events()[0].(character.CreatedSlotsExpired)
	if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.Count() != 3 {
		t.Errorf("events = %#v, want one CreatedSlotsExpired of 3", ch.Events())
	}
	if again := sorChanged(t, must(character.ApplySorcererRest(sorID, got, p, character.LongRest, 0))); len(again.Events()) != 0 {
		t.Errorf("a Long Rest with no created slot produced %#v", again.Events())
	}
	fresh := sorWith(p, 3, nil, func(s *character.SorcererStateParts) { s.CreatedSlots = [5]int{1, 0, 2, 0, 0} })
	short := sorChanged(t, must(character.ApplySorcererRest(sorID, fresh, p, character.ShortRest, 3)))
	if short.State().CreatedSlots(3) != 2 || short.State().CreatedSlots(1) != 1 {
		t.Errorf("a Short Rest removed created slots: %+v", short.State().Parts())
	}
	if rej := sorRefusedAs(t, must(character.ApplySorcererRest(sorID, spent, p, character.LongRest, 2))); rej.Kind != character.SorcererRestorationNotAShortRest {
		t.Errorf("R-10: kind = %v, want SorcererRestorationNotAShortRest", rej.Kind)
	}
	// R-11: a refused attempt leaves the use available, and a later valid request is accepted.
	p5 := sorProfile(5, "")
	s := sorWith(p5, 0, nil, nil)
	if rej := sorRefusedAs(t, must(character.ApplySorcererRest(sorID, s, p5, character.ShortRest, 3))); rej.Kind != character.SorcererRestorationAmountInvalid || s.RestorationUsed() {
		t.Errorf("R-11 first attempt: %+v", rej)
	}
	if ok := sorChanged(t, must(character.ApplySorcererRest(sorID, s, p5, character.ShortRest, 1))); !ok.State().RestorationUsed() || ok.State().SorceryPoints() != 1 {
		t.Errorf("R-11 later request: %+v", ok.State().Parts())
	}
	if _, err := character.ApplySorcererRest(sorID, s, p5, 0, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an unknown rest: err = %v, want ErrInvalidInput", err)
	}
	if _, err := character.ApplySorcererRest(sorID, s, p5, 3, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a rest kind past the two: err = %v, want ErrInvalidInput", err)
	}
	p1 := sorProfile(1, "")
	l1 := sorChanged(t, must(character.ApplySorcererRest(sorID, sorWith(p1, 0, nil, func(s *character.SorcererStateParts) { s.InnateSorceryUses = 0 }), p1, character.LongRest, 0)))
	if l1.State().SorceryPoints() != 0 || l1.State().InnateSorceryUses() != 2 {
		t.Errorf("level 1 Long Rest = %+v, want 0 points and 2 uses", l1.State().Parts())
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-055 to FR-057 (SRD 5.2.1, "Level 2: Metamagic", p. 66) - a selection is accepted with the entitled number of distinct listed options; it is rejected with every problem if it is the wrong
// size, repeats an option or names one that content lacks or that is not one of the ten.
// Path: per case (happy / unhappy: invalid input).
func TestNewMetamagicSelection(t *testing.T) {
	t.Parallel()
	var listed []content.FeatureOption
	for _, f := range sorDef(t).Features() {
		listed = append(listed, f.Options()...)
	}
	id := func(effects ...character.MetamagicEffect) []content.FeatureOptionID { return sorOptionIDs(effects...) }
	for _, c := range []struct {
		name, requirement string
		level             int
		chosen            []content.FeatureOptionID
		listed            []content.FeatureOption
		problems          []character.MetamagicSelectionProblem
		want              []character.MetamagicEffect
	}{
		{"two options at level 2", "FR-056", 2, id(character.MetamagicTwinned, character.MetamagicCareful), listed, nil, []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}},
		{"four options at level 10", "FR-055", 10, id(1, 2, 3, 4), listed, nil, []character.MetamagicEffect{1, 2, 3, 4}},
		{"six options at level 17", "FR-055", 17, id(1, 2, 3, 4, 5, 6), listed, nil, []character.MetamagicEffect{1, 2, 3, 4, 5, 6}},
		{"none at level 1", "FR-055", 1, nil, listed, nil, []character.MetamagicEffect{}},
		{"one too few", "FR-057", 2, id(1), listed, []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize}, nil},
		{"one too many", "FR-057", 2, id(1, 2, 3), listed, []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize}, nil},
		{"two at level 10", "FR-057", 10, id(1, 2), listed, []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize}, nil},
		{"one at level 1", "FR-057", 1, id(1), listed, []character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize}, nil},
		{"a repeat", "FR-057", 2, id(1, 1), listed, []character.MetamagicSelectionProblem{character.MetamagicSelectionDuplicate}, nil},
		{"an option that is not one of the ten", "FR-057", 2, []content.FeatureOptionID{"careful_spell", "lucky_spell"}, listed, []character.MetamagicSelectionProblem{character.MetamagicSelectionUnknownOption}, nil},
		{"an option content does not list", "FR-057", 2, id(1, 2), listed[:1], []character.MetamagicSelectionProblem{character.MetamagicSelectionUnknownOption}, nil},
		{"every problem together", "FR-057", 2, []content.FeatureOptionID{"careful_spell", "careful_spell", "lucky_spell"}, listed,
			[]character.MetamagicSelectionProblem{character.MetamagicSelectionWrongSize, character.MetamagicSelectionDuplicate, character.MetamagicSelectionUnknownOption}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			sel, problems := character.NewMetamagicSelection(character.MustSorcererLevel(c.level), c.chosen, c.listed)
			if !slices.Equal(problems, append([]character.MetamagicSelectionProblem{}, c.problems...)) || (problems == nil) {
				t.Errorf("problems = %v, want %v [%s]", problems, c.problems, c.requirement)
			}
			if got := sel.Effects(); !slices.Equal(got, c.want) {
				t.Errorf("effects = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-056, FR-058 (SRD 5.2.1, "Level 2: Metamagic", p. 66) - choosing makes the selection the options known; at most one known option may be dropped, a second replacement is refused, and level 1 has no Metamagic.
// Path: per case (happy / unhappy: state).
func TestChooseMetamagic(t *testing.T) {
	t.Parallel()
	sel := func(level int, effects ...character.MetamagicEffect) character.MetamagicSelection {
		s, problems := character.NewMetamagicSelection(character.MustSorcererLevel(level), sorOptionIDs(effects...), sorDef(t).Features()[3].Options())
		if len(problems) != 0 {
			t.Fatalf("selection problems = %v", problems)
		}
		return s
	}
	p := sorProfile(2, "")
	none := sorFresh(p)
	ch := sorChanged(t, character.ChooseMetamagic(sorID, none, p, sel(2, character.MetamagicTwinned, character.MetamagicCareful)))
	ev, ok := ch.Events()[0].(character.MetamagicOptionsChosen)
	if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || !slices.Equal(ev.Options(), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}) {
		t.Errorf("events = %#v, want one MetamagicOptionsChosen of Careful and Twinned", ch.Events())
	}
	if !slices.Equal(ch.State().Metamagic(), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}) || len(none.Metamagic()) != 0 {
		t.Error("the options must be set on the new state only")
	}
	held := ch.State()
	one := sorChanged(t, character.ChooseMetamagic(sorID, held, p, sel(2, character.MetamagicCareful, character.MetamagicDistant)))
	if !slices.Equal(one.State().Metamagic(), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicDistant}) {
		t.Errorf("one replacement: %v", one.State().Metamagic())
	}
	if rej := sorRefusedAs(t, character.ChooseMetamagic(sorID, held, p, sel(2, character.MetamagicDistant, character.MetamagicEmpowered))); rej.Kind != character.SorcererMetamagicReplacementInvalid {
		t.Errorf("two replacements: kind = %v, want SorcererMetamagicReplacementInvalid", rej.Kind)
	}
	same := sorChanged(t, character.ChooseMetamagic(sorID, held, p, sel(2, character.MetamagicCareful, character.MetamagicTwinned)))
	if same.State() != held {
		t.Error("choosing the same options changed the state")
	}
	p1 := sorProfile(1, "")
	if rej := sorRefusedAs(t, character.ChooseMetamagic(sorID, sorFresh(p1), p1, character.MetamagicSelection{})); rej.Kind != character.SorcererFeatureNotGained || rej.Feature != "metamagic" {
		t.Errorf("level 1: %+v, want SorcererFeatureNotGained for metamagic", rej)
	}
	grown := sorChanged(t, character.ChooseMetamagic(sorID, held, sorProfile(10, ""), sel(10, character.MetamagicCareful, character.MetamagicTwinned, character.MetamagicSubtle, character.MetamagicSeeking)))
	if len(grown.State().Metamagic()) != 4 {
		t.Errorf("two more options at level 10: %v", grown.State().Metamagic())
	}
	if got := sel(2, character.MetamagicCareful, character.MetamagicDistant).Effects(); len(got) != 2 {
		t.Fatal("selection")
	} else {
		got[0] = character.MetamagicTwinned
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-087, FR-088 (SRD 5.2.1, "Level 3: Draconic Resilience", p. 69) - Draconic Resilience is a Hit Point maximum bonus of 3 plus 1 per further Sorcerer level, granted at level 3 and scoped to the
// Sorcerer, and an Armor Class basis of 10 plus the Dexterity and Charisma modifiers that a Shield does not disable.
// Path: happy.
func TestDraconicResilience(t *testing.T) {
	t.Parallel()
	want := must(character.NewHitPointMaximumBonus(3, 1, 3, "sorcerer"))
	if got := character.DraconicResilienceBonus(); got != want {
		t.Errorf("bonus = %+v, want %+v", got, want)
	}
	basis := character.DraconicArmorClassBasis()
	if basis.ID() != "draconic_resilience" || !basis.ShieldAllowed() || !slices.Equal(basis.Abilities(), []ability.Ability{ability.Charisma}) || basis.BaseValue() != 10 {
		t.Errorf("basis = %+v, want 10 + Dexterity + Charisma with a Shield allowed", basis)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-090 (SRD 5.2.1, "Level 6: Elemental Affinity", p. 70) - Elemental Affinity records one of Acid, Cold, Fire, Lightning or Poison; another type or a second choice is refused; a Sorcerer that is
// not a Draconic Sorcerer of level 6 cannot choose.
// Path: per case (invalid input / state / boundary).
func TestChooseElementalAffinity(t *testing.T) {
	t.Parallel()
	d6, fresh := sorDraconic(6), sorFresh(sorDraconic(6))
	for _, ty := range character.ElementalAffinityTypes() {
		ch := sorChanged(t, character.ChooseElementalAffinity(sorID, fresh, d6, ty))
		ev, ok := ch.Events()[0].(character.ElementalAffinityChosen)
		got, has := ch.State().ElementalAffinity()
		if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.DamageType() != ty || !has || got != ty {
			t.Errorf("%v: events %#v, state %v", ty, ch.Events(), got)
		}
	}
	for _, ty := range []content.DamageType{0, content.DamageThunder, content.DamageForce, content.DamageNecrotic} {
		if rej := sorRefusedAs(t, character.ChooseElementalAffinity(sorID, fresh, d6, ty)); rej.Kind != character.SorcererElementalAffinityChoiceInvalid {
			t.Errorf("%v: kind = %v, want SorcererElementalAffinityChoiceInvalid", ty, rej.Kind)
		}
	}
	chosen := sorChanged(t, character.ChooseElementalAffinity(sorID, fresh, d6, content.DamageFire)).State()
	if rej := sorRefusedAs(t, character.ChooseElementalAffinity(sorID, chosen, d6, content.DamageCold)); rej.Kind != character.SorcererElementalAffinityChoiceInvalid {
		t.Errorf("a second choice: kind = %v", rej.Kind)
	}
	for name, p := range map[string]character.SorcererProfile{"level 5": sorDraconic(5), "no subclass": sorProfile(6, ""), "another subclass": sorProfile(6, "wild_magic")} {
		if rej := sorRefusedAs(t, character.ChooseElementalAffinity(sorID, fresh, p, content.DamageFire)); rej.Kind != character.SorcererFeatureNotGained || rej.Feature != "elemental_affinity" {
			t.Errorf("%s: %+v, want SorcererFeatureNotGained for elemental_affinity", name, rej)
		}
	}
}

// Requirement: doc/specs/class-sorcerer.md Table E, FR-091, FR-104, FR-105 (SRD 5.2.1, "Level 6: Elemental Affinity", p. 70; "Transmuted Spell", p. 67) - the Charisma modifier is added to one damage roll of a spell that deals the chosen type
// after Transmuted Spell, whatever its class list; a spell that deals none of that type, no type chosen and a second addition are refused.
// Path: per case (Table E).
func TestApplyElementalAffinity(t *testing.T) {
	t.Parallel()
	p := sorDraconic(6)
	fire := sorWith(p, 6, nil, func(s *character.SorcererStateParts) { s.ElementalAffinity = content.DamageFire })
	cold := sorWith(p, 6, nil, func(s *character.SorcererStateParts) { s.ElementalAffinity = content.DamageCold })
	cha3 := sorMod(16)
	fireDmg, coldDmg := []content.DamageType{content.DamageFire}, []content.DamageType{content.DamageCold}
	for _, c := range []struct {
		name, requirement, path string
		state                   character.SorcererState
		profile                 character.SorcererProfile
		dealt                   []content.DamageType
		applied                 bool
		kind                    character.SorcererRejectionKind
		bonus                   int
	}{
		{"E-1", "FR-091", "happy", fire, p, fireDmg, false, 0, 3},
		{"E-2", "FR-091", "unhappy (invalid input)", fire, p, coldDmg, false, character.SorcererElementalAffinityTypeMismatch, 0},
		{"E-7", "FR-091", "unhappy (empty/minimal)", fire, p, nil, false, character.SorcererElementalAffinityTypeMismatch, 0},
		{"E-3", "FR-104", "unhappy (boundary)", fire, p, []content.DamageType{content.DamageFire, content.DamageCold}, false, 0, 3},
		{"E-4", "FR-105", "unhappy (rule interaction)", fire, p, coldDmg, false, character.SorcererElementalAffinityTypeMismatch, 0},
		{"E-5", "FR-105", "unhappy (rule interaction)", cold, p, coldDmg, false, 0, 3},
		{"E-6", "FR-091", "unhappy (state)", fire, p, fireDmg, true, character.SorcererElementalAffinityAlreadyApplied, 0},
		{"E-8", "FR-090", "unhappy (state)", sorFresh(p), p, fireDmg, false, character.SorcererElementalAffinityTypeMismatch, 0},
		{"level 5", "FR-091", "unhappy (boundary)", fire, sorDraconic(5), fireDmg, false, character.SorcererFeatureNotGained, 0},
		{"no subclass", "FR-091", "unhappy (state)", fire, sorProfile(6, ""), fireDmg, false, character.SorcererFeatureNotGained, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := character.ApplyElementalAffinity(sorID, c.state, c.profile, cha3, c.dealt, c.applied)
			if c.kind != 0 {
				if rej := sorRefusedAs(t, tr); rej.Kind != c.kind {
					t.Errorf("kind = %v, want %v [%s %s]", rej.Kind, c.kind, c.requirement, c.path)
				}
				return
			}
			ch := sorChanged(t, tr)
			ev, ok := ch.Events()[0].(character.ElementalAffinityApplied)
			chosen, _ := c.state.ElementalAffinity()
			if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.Bonus() != c.bonus || ev.DamageType() != chosen || ch.State() != c.state {
				t.Errorf("events = %#v, want a bonus of %d on %v with the state unchanged [%s %s]", ch.Events(), c.bonus, chosen, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-092 to FR-094 (SRD 5.2.1, "Level 14: Dragon Wings", p. 70) - the wings appear (a Bonus Action) once until a Long Rest, are dismissed with no action, and the use is restored for 3 Sorcery Points; a
// Sorcerer that is not a Draconic Sorcerer of level 14 cannot.
// Path: per case (happy / unhappy: state, boundary).
func TestUseDragonWings(t *testing.T) {
	t.Parallel()
	p := sorDraconic(14)
	fresh := sorFresh(p)
	used := sorWith(p, 14, nil, func(s *character.SorcererStateParts) { s.DragonWingsUsed = true })
	active := sorWith(p, 14, nil, func(s *character.SorcererStateParts) { s.DragonWingsUsed, s.DragonWingsActive = true, true })
	points := func(n int) character.SorcererState {
		return sorWith(p, n, nil, func(s *character.SorcererStateParts) { s.DragonWingsUsed = true })
	}
	for _, c := range []struct {
		name, requirement, path string
		state                   character.SorcererState
		profile                 character.SorcererProfile
		action                  character.DragonWingsAction
		kind                    character.SorcererRejectionKind
		want                    character.SorcererStateParts
	}{
		{"activate", "FR-092", "happy", fresh, p, character.DragonWingsActivate, 0, character.SorcererStateParts{SorceryPoints: 14, InnateSorceryUses: 2, DragonWingsUsed: true, DragonWingsActive: true}},
		{"activate after use", "FR-093", "unhappy (state)", used, p, character.DragonWingsActivate, character.SorcererDragonWingsAlreadyUsed, character.SorcererStateParts{}},
		{"dismiss", "FR-092", "happy", active, p, character.DragonWingsDismiss, 0, character.SorcererStateParts{SorceryPoints: 14, InnateSorceryUses: 2, DragonWingsUsed: true}},
		{"dismiss with the wings in", "FR-092", "unhappy (state)", used, p, character.DragonWingsDismiss, character.SorcererDragonWingsNotActive, character.SorcererStateParts{}},
		{"restore for 3 points", "FR-093", "happy", points(3), p, character.DragonWingsRestoreUse, 0, character.SorcererStateParts{SorceryPoints: 0, InnateSorceryUses: 2}},
		{"restore with 4 points", "FR-094", "unhappy (boundary)", points(4), p, character.DragonWingsRestoreUse, 0, character.SorcererStateParts{SorceryPoints: 1, InnateSorceryUses: 2}},
		{"restore with 2 points", "FR-094", "unhappy (boundary)", points(2), p, character.DragonWingsRestoreUse, character.SorcererSorceryPointsInsufficient, character.SorcererStateParts{}},
		{"restore an unspent use", "FR-094", "unhappy (state)", fresh, p, character.DragonWingsRestoreUse, character.SorcererDragonWingsUseNotExpended, character.SorcererStateParts{}},
		{"level 13", "FR-094", "unhappy (boundary)", fresh, sorDraconic(13), character.DragonWingsActivate, character.SorcererFeatureNotGained, character.SorcererStateParts{}},
		{"no subclass", "FR-092", "unhappy (state)", fresh, sorProfile(14, ""), character.DragonWingsActivate, character.SorcererFeatureNotGained, character.SorcererStateParts{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := character.UseDragonWings(sorID, c.state, c.profile, c.action)
			if err != nil {
				t.Fatal(err)
			}
			if c.kind != 0 {
				rej := sorRefusedAs(t, tr)
				if rej.Kind != c.kind || (c.kind == character.SorcererSorceryPointsInsufficient && (rej.Required != 3 || rej.Available != 2)) {
					t.Errorf("rejection = %+v, want %v [%s %s]", rej, c.kind, c.requirement, c.path)
				}
				return
			}
			ch := sorChanged(t, tr)
			if got, want := ch.State().Parts(), sorState(c.profile, c.want).Parts(); !reflect.DeepEqual(got, want) || len(ch.Events()) != 1 {
				t.Errorf("state = %+v with %d events, want %+v [%s %s]", got, len(ch.Events()), want, c.requirement, c.path)
			}
			switch c.action {
			case character.DragonWingsActivate:
				if ev, ok := ch.Events()[0].(character.DragonWingsActivated); !ok || ev.CharacterID() != sorID {
					t.Errorf("event = %#v", ch.Events()[0])
				}
			case character.DragonWingsDismiss:
				if ev, ok := ch.Events()[0].(character.DragonWingsEnded); !ok || ev.CharacterID() != sorID {
					t.Errorf("event = %#v", ch.Events()[0])
				}
			case character.DragonWingsRestoreUse:
				if ev, ok := ch.Events()[0].(character.DragonWingsUseRestored); !ok || ev.CharacterID() != sorID {
					t.Errorf("event = %#v", ch.Events()[0])
				}
			}
		})
	}
	for _, a := range []character.DragonWingsAction{0, 4, -1} {
		if _, err := character.UseDragonWings(sorID, fresh, p, a); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("action %d: err = %v, want ErrInvalidInput", int(a), err)
		}
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-095, FR-096 (SRD 5.2.1, "Level 18: Dragon Companion", p. 70) - Summon Dragon may be cast once without a spell slot until a Long Rest, optionally without Concentration; a second slot-free cast and a
// Sorcerer that is not a Draconic Sorcerer of level 18 are refused.
// Path: per case (happy / unhappy: state, boundary).
func TestUseDragonCompanion(t *testing.T) {
	t.Parallel()
	p := sorDraconic(18)
	for _, waive := range []bool{false, true} {
		ch := sorChanged(t, character.UseDragonCompanion(sorID, sorFresh(p), p, waive))
		ev, ok := ch.Events()[0].(character.DragonCompanionCast)
		if len(ch.Events()) != 1 || !ok || ev.CharacterID() != sorID || ev.WaivedConcentration() != waive || !ch.State().DragonCompanionFreeCastUsed() {
			t.Errorf("waive %v: events %#v, state %+v", waive, ch.Events(), ch.State().Parts())
		}
	}
	used := sorWith(p, 18, nil, func(s *character.SorcererStateParts) { s.DragonCompanionFreeCastUsed = true })
	if rej := sorRefusedAs(t, character.UseDragonCompanion(sorID, used, p, false)); rej.Kind != character.SorcererDragonCompanionFreeCastUsed {
		t.Errorf("second cast: kind = %v", rej.Kind)
	}
	for name, prof := range map[string]character.SorcererProfile{"level 17": sorDraconic(17), "no subclass": sorProfile(18, "")} {
		if rej := sorRefusedAs(t, character.UseDragonCompanion(sorID, sorFresh(prof), prof, false)); rej.Kind != character.SorcererFeatureNotGained || rej.Feature != "dragon_companion" {
			t.Errorf("%s: %+v, want SorcererFeatureNotGained for dragon_companion", name, rej)
		}
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-085 to FR-087, FR-089, FR-134 (SRD 5.2.1, "Level 4: Ability Score Improvement", "Level 19: Epic Boon", p. 66; "Draconic Spells", p. 70) - the grants of a level are the feat categories (General at 4, 8, 12 and 16,
// Epic Boon at 19, none elsewhere), the Draconic Resilience bonus for a Draconic Sorcerer of level 3+, and the Draconic Spells newly granted at exactly that level.
// Path: per level (1 to 20).
func TestSorcererLevelGrantsFor(t *testing.T) {
	t.Parallel()
	def := sorDef(t)
	spells := map[int][]content.SpellID{3: {"alter_self", "chromatic_orb", "command", "dragons_breath"}, 5: {"fear", "fly"}, 7: {"arcane_eye", "charm_monster"}, 9: {"legend_lore", "summon_dragon"}}
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			wantFeats := []content.FeatCategory{}
			switch n {
			case 4, 8, 12, 16:
				wantFeats = append(wantFeats, content.FeatGeneral)
			case 19:
				wantFeats = append(wantFeats, content.FeatEpicBoon)
			}
			g, err := character.SorcererLevelGrantsFor(sorDraconic(n), def)
			if err != nil {
				t.Fatal(err)
			}
			bonus, has := g.HitPointBonus()
			wantSpells := spells[n]
			if wantSpells == nil {
				wantSpells = []content.SpellID{}
			}
			if !slices.Equal(g.FeatGrants(), wantFeats) || !slices.Equal(g.AlwaysPreparedSpells(), wantSpells) || has != (n >= 3) || (has && bonus != character.DraconicResilienceBonus()) {
				t.Errorf("grants = feats %v, spells %v, bonus (%v, %v); want %v, %v and a bonus from level 3", g.FeatGrants(), g.AlwaysPreparedSpells(), bonus, has, wantFeats, wantSpells)
			}
			plain, err := character.SorcererLevelGrantsFor(sorProfile(n, ""), def)
			if _, noBonus := plain.HitPointBonus(); err != nil || noBonus || len(plain.AlwaysPreparedSpells()) != 0 || !slices.Equal(plain.FeatGrants(), wantFeats) {
				t.Errorf("no subclass: grants = %+v, %v; want only the feats", plain, err)
			}
		})
	}
	g := must(character.SorcererLevelGrantsFor(sorDraconic(4), def))
	g.FeatGrants()[0] = content.FeatEpicBoon
	if got := g.FeatGrants()[0]; got != content.FeatGeneral {
		t.Error("FeatGrants returned a shared slice")
	}
}
