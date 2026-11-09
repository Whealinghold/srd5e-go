package character_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-cleric.md FR-015, FR-017 (SRD 5.2.1, "Cleric Features", p. 36; "Proficiency", p. 8) - the Proficiency Bonus
// column of the bundled Cleric Features table equals the general Proficiency Bonus for the same level, and its Cantrips, Prepared Spells
// and highest Spell Slot level agree with what spellcasting derives for the Cleric. A disagreement fails this consistency test, not a decode.
// Path: per case (the 20 levels): unhappy (rule interaction).
func TestClericTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	def := clcDef(t)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must(def.RowAt(n))
			sc := row.Spellcasting().(content.SlotSpellcastingRow)
			cantrips, _ := sc.Cantrips()
			highest := 0
			for i, s := range sc.SpellSlots() {
				if s > 0 {
					highest = i + 1
				}
			}
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() ||
				cantrips != character.CantripLimit(character.CasterCleric, n) || sc.PreparedSpells() != character.PreparedLimit(character.CasterCleric, n) ||
				highest != character.MaxPreparableLevel(character.CasterCleric, n) {
				t.Errorf("level %d row = bonus %d cantrips %d prepared %d highest slot %d; the rules say %d %d %d %d", n, row.ProficiencyBonus(), cantrips, sc.PreparedSpells(), highest,
					character.ProficiencyBonusFor(character.MustLevel(n)).Value(), character.CantripLimit(character.CasterCleric, n), character.PreparedLimit(character.CasterCleric, n),
					character.MaxPreparableLevel(character.CasterCleric, n))
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-038, FR-049, FR-105, FR-117 (SRD 5.2.1, "Level 2: Channel Divinity", p. 37) - StatsAt reads the Channel Divinity
// maximum from the bundled column (0 below level 2, 2 at levels 2 to 5, 3 at 6 to 17, 4 at 18 to 20) and gives the Divine Spark dice (1, 2 from 7, 3 from 13, 4 from
// 18) and the Preserve Life pool (five times the Cleric level), at every boundary.
// Path: per case (the boundary levels): unhappy (boundary).
func TestClericStatsAt(t *testing.T) {
	t.Parallel()
	p := clcProgression(t)
	for _, c := range []struct{ level, channel, dice int }{
		{1, 0, 1}, {2, 2, 1}, {5, 2, 1}, {6, 3, 1}, {7, 3, 2}, {12, 3, 2}, {13, 3, 3}, {17, 3, 3}, {18, 4, 4}, {20, 4, 4},
	} {
		s := p.StatsAt(character.MustClericLevel(c.level))
		if s.ChannelDivinity() != c.channel || s.DivineSparkDice() != c.dice || s.PreserveLifePool() != 5*c.level || s.Level().Value() != c.level || s.IsZero() {
			t.Errorf("level %d = %d uses, %d d8, pool %d; want %d, %d, %d", c.level, s.ChannelDivinity(), s.DivineSparkDice(), s.PreserveLifePool(), c.channel, c.dice, 5*c.level)
		}
	}
	if !p.StatsAt(character.ClericLevel{}).IsZero() || !(character.ClericProgression{}).StatsAt(character.MustClericLevel(5)).IsZero() || p.IsZero() || !(character.ClericProgression{}).IsZero() {
		t.Error("zero level or zero progression gives non-zero stats")
	}
}

// Requirement: doc/specs/class-cleric.md FR-021, FR-125, table 4.14.3 (SRD 5.2.1, "Cleric Features", p. 36) - the features gained at each level come from the
// bundled definition (11 in all) and the pending choices are divine_order at level 1 and blessed_strikes_option at level 7, never elsewhere.
// Path: per case (the 20 levels): happy / unhappy (boundary).
func TestClericFeaturesGainedAndChoices(t *testing.T) {
	t.Parallel()
	p := clcProgression(t)
	want := map[int][]string{1: {"cleric_spellcasting", "cleric_divine_order"}, 2: {"cleric_channel_divinity"}, 3: {"cleric_cleric_subclass"}, 4: {"cleric_ability_score_improvement"},
		5: {"cleric_sear_undead"}, 7: {"cleric_blessed_strikes"}, 10: {"cleric_divine_intervention"}, 14: {"cleric_improved_blessed_strikes"}, 19: {"cleric_epic_boon"},
		20: {"cleric_greater_divine_intervention"}}
	total := 0
	for n := 1; n <= 20; n++ {
		got := p.FeaturesGainedAt(character.MustClericLevel(n))
		total += len(got)
		if len(got) != len(want[n]) {
			t.Errorf("level %d gains %v, want %v", n, got, want[n])
			continue
		}
		for i, id := range got {
			if string(id) != want[n][i] {
				t.Errorf("level %d feature %d = %s, want %s", n, i, id, want[n][i])
			}
		}
		pending := p.ChoicesOnGaining(character.MustClericLevel(n))
		switch n {
		case 1:
			if len(pending) != 1 || pending[0] != character.ClericPendingDivineOrder {
				t.Errorf("level 1 pending = %v", pending)
			}
		case 7:
			if len(pending) != 1 || pending[0] != character.ClericPendingBlessedStrikes {
				t.Errorf("level 7 pending = %v", pending)
			}
		default:
			if len(pending) != 0 || pending == nil {
				t.Errorf("level %d pending = %#v, want empty", n, pending)
			}
		}
	}
	if total != 11 || character.ClericPendingDivineOrder.String() != "divine_order" || character.ClericPendingBlessedStrikes.String() != "blessed_strikes_option" || character.ClericPendingChoice(0).String() != "unknown" {
		t.Errorf("total features %d or pending names wrong", total)
	}
	got := p.FeaturesGainedAt(character.MustClericLevel(1))
	got[0] = "changed"
	if p.FeaturesGainedAt(character.MustClericLevel(1))[0] == "changed" {
		t.Error("FeaturesGainedAt returned a shared slice")
	}
	if len(p.FeaturesGainedAt(character.ClericLevel{})) != 0 || len((character.ClericProgression{}).ChoicesOnGaining(character.MustClericLevel(1))) != 0 {
		t.Error("zero level or progression gains something")
	}
}

// Requirement: doc/specs/class-cleric.md FR-038 - NewClericProgression refuses a definition of another class, one without the channel_divinity count
// column and one whose rows hold no count from level 2, each with its *ClericProgressionError matching srderr.ErrInvalidInput; the Fighter's definition is not
// the Cleric's.
// Path: per case (unhappy: invalid input).
func TestNewClericProgressionRefusals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		want character.ClericProgressionReason
	}{
		{"another class", fgtDef(t), character.NotTheClericDefinition},
		{"zero definition", content.ClassDefinition{}, character.NotTheClericDefinition},
		{"no channel_divinity column", clcDefFrom(t, func(s string) string { return strings.ReplaceAll(s, "channel_divinity", "channel_x") }), character.ClericColumnMissing},
		{"a dash at level 2", clcDefFrom(t, func(s string) string {
			return strings.Replace(s, `"channel_divinity": 2`, `"channel_divinity": "—"`, 1)
		}), character.ClericRowsIncomplete},
	} {
		_, err := character.NewClericProgression(c.def)
		var pe *character.ClericProgressionError
		if !errors.As(err, &pe) || pe.Reason != c.want || !errors.Is(err, &character.ClericProgressionError{Reason: c.want}) || errors.Is(err, &character.ClericProgressionError{Reason: c.want + 1}) {
			t.Errorf("%s: error = %v, want reason %v", c.name, err, c.want)
		}
		wantInvalid(t, err, c.name)
		if pe != nil && pe.Error() == "" {
			t.Error("empty error text")
		}
	}
	for r, s := range map[character.ClericProgressionReason]string{character.NotTheClericDefinition: "not_the_cleric_definition", character.ClericColumnMissing: "column_missing", character.ClericRowsIncomplete: "rows_incomplete", 0: "unknown"} {
		if r.String() != s {
			t.Errorf("reason %d = %q, want %q", r, r, s)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustClericProgression of the Fighter did not panic")
		}
	}()
	character.MustClericProgression(fgtDef(t))
}
