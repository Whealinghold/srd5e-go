package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func serviceRow(lo, hi int, a content.Availability, cost content.Cost) content.SpellcastingService {
	return must(content.NewSpellcastingService("service_x", "Service", content.SourceCitation{}, lo, hi, a, price(cost)))
}

// serviceRows returns the printed Spellcasting Services table.
func serviceRows() []content.SpellcastingService {
	return []content.SpellcastingService{
		serviceRow(0, 0, content.AvailabilityVillageTownOrCity, gp(30)),
		serviceRow(1, 1, content.AvailabilityVillageTownOrCity, gp(50)),
		serviceRow(2, 2, content.AvailabilityVillageTownOrCity, gp(200)),
		serviceRow(3, 3, content.AvailabilityTownOrCityOnly, gp(300)),
		serviceRow(4, 5, content.AvailabilityTownOrCityOnly, gp(2000)),
		serviceRow(6, 8, content.AvailabilityCityOnly, gp(20000)),
		serviceRow(9, 9, content.AvailabilityCityOnly, gp(100000)),
	}
}

// Requirement: per case equipSpec FR-052 (SRD 5.2.1, Spellcasting Services
// table, p. 102) - the row for each level 0 to 9, ranges included.
// Path: per case (happy; unhappy boundary).
func TestSpellcastingServicesForSpellLevel(t *testing.T) {
	t.Parallel()
	table := character.MustSpellcastingServices(serviceRows())
	want := []struct {
		gold int
		a    content.Availability
	}{
		{30, content.AvailabilityVillageTownOrCity}, {50, content.AvailabilityVillageTownOrCity}, {200, content.AvailabilityVillageTownOrCity},
		{300, content.AvailabilityTownOrCityOnly}, {2000, content.AvailabilityTownOrCityOnly}, {2000, content.AvailabilityTownOrCityOnly},
		{20000, content.AvailabilityCityOnly}, {20000, content.AvailabilityCityOnly}, {20000, content.AvailabilityCityOnly},
		{100000, content.AvailabilityCityOnly},
	}
	for l, w := range want {
		row := table.ForSpellLevel(levelOf(t, l))
		if row.Price().Cost() != gp(w.gold) || row.Availability() != w.a {
			t.Errorf("level %d = %v %v, want %d GP %v", l, row.Price().Cost(), row.Availability(), w.gold, w.a)
		}
	}
	if got := table.ForSpellLevel(content.SpellLevel(10)); got.SpellLevelMax() != 0 || got.Price() != (content.FixedCost{}) {
		t.Errorf("level 10 = %v, want the zero row", got)
	}
	if got := (character.SpellcastingServices{}).ForSpellLevel(levelOf(t, 1)); got.Price() != (content.FixedCost{}) {
		t.Error("the zero table has no rows")
	}
}

// Requirement: equipSpec FR-053, FR-054 - ranges that leave a gap or overlap, or
// no rows, give a *TableCoverageError (an srderr.ErrInvalidInput error) listing
// the uncovered and overlapping levels in order; MustSpellcastingServices
// panics with that error.
// Path: unhappy (missing content, invalid input).
func TestNewSpellcastingServicesCoverage(t *testing.T) {
	t.Parallel()
	rows := serviceRows()
	a := content.AvailabilityCityOnly
	cases := []struct {
		name        string
		rows        []content.SpellcastingService
		uncovered   []content.SpellLevel
		overlapping []content.SpellLevel
	}{
		{"gap at 4", slices.Concat(rows[:4], []content.SpellcastingService{serviceRow(5, 5, a, gp(1))}, rows[5:]), []content.SpellLevel{4}, nil},
		{"overlap at 4", slices.Concat(rows[:4], []content.SpellcastingService{serviceRow(4, 4, a, gp(1))}, rows[4:]), nil, []content.SpellLevel{4}},
		{"nil", nil, []content.SpellLevel{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := character.NewSpellcastingServices(c.rows)
			var cov *character.TableCoverageError
			if !errors.As(err, &cov) || !slices.Equal(cov.Uncovered, c.uncovered) || !slices.Equal(cov.Overlapping, c.overlapping) {
				t.Fatalf("err = %v, want uncovered %v overlapping %v", err, c.uncovered, c.overlapping)
			}
			wantInvalid(t, err, "coverage")
			defer func() {
				if r := recover(); !errors.Is(r.(error), cov) {
					t.Errorf("Must panicked with %v, want %v", r, cov)
				}
			}()
			character.MustSpellcastingServices(c.rows)
			t.Error("MustSpellcastingServices did not panic")
		})
	}
}

// Requirement: per case equipSpec FR-055 (SRD 5.2.1, Spellcasting Services
// table, Availability, p. 102) - Village, Town and City against the three
// availabilities.
// Path: unhappy (rule interaction).
func TestSpellcastingServicesIsAvailable(t *testing.T) {
	t.Parallel()
	table := character.MustSpellcastingServices(serviceRows())
	cases := []struct {
		level int
		size  content.SettlementSize
		want  bool
	}{
		{2, content.SettlementVillage, true}, {3, content.SettlementVillage, false}, {3, content.SettlementTown, true},
		{5, content.SettlementTown, true}, {6, content.SettlementTown, false}, {9, content.SettlementCity, true},
		{0, content.SettlementCity, true}, {9, content.SettlementVillage, false},
	}
	for _, c := range cases {
		if got := table.IsAvailable(table.ForSpellLevel(levelOf(t, c.level)), c.size); got != c.want {
			t.Errorf("level %d in %v = %v, want %v", c.level, c.size, got, c.want)
		}
	}
	zeroSvc := character.SpellcastingServices{}
	if zeroSvc.IsAvailable(content.SpellcastingService{}, content.SettlementCity) || zeroSvc.IsAvailable(table.ForSpellLevel(levelOf(t, 0)), 0) {
		t.Error("an invalid row or size is never available")
	}
}

// Requirement: equipSpec FR-056, FR-057 (SRD 5.2.1, "Spellcasting", p. 102) -
// the total is the listed cost plus the components as Copper Pieces; a rated
// component cost, or a total beyond the largest Cost, is an
// srderr.ErrInvalidInput error.
// Path: unhappy (rule interaction, invalid input).
func TestSpellcastingServicesTotalCost(t *testing.T) {
	t.Parallel()
	table := character.MustSpellcastingServices(serviceRows())
	three := table.ForSpellLevel(levelOf(t, 3))
	if got, err := table.TotalCost(three, gp(100)); err != nil || got != cp(40000) {
		t.Errorf("level 3 with 100 GP = %v, %v, want 40,000 CP", got, err)
	}
	if got, err := table.TotalCost(three, content.Cost{}); err != nil || got != cp(30000) {
		t.Errorf("level 3 with no components = %v, %v", got, err)
	}
	_, err := table.TotalCost(three, content.MustCost(5, content.CoinSP, content.CostPerDay))
	wantInvalid(t, err, "rated components")
	_, err = table.TotalCost(three, content.MustCost(content.MaxCostAmount, content.CoinCP, 0))
	wantInvalid(t, err, "overflow")
	if _, err := table.TotalCost(three, cp(content.MaxCostAmount-30000)); err != nil {
		t.Errorf("largest total: %v", err)
	}
}

// Requirement: equipSpec FR-067 - the table copies the rows it is given.
// Path: unhappy (immutability).
func TestSpellcastingServicesCopiesRows(t *testing.T) {
	t.Parallel()
	rows := serviceRows()
	table := character.MustSpellcastingServices(rows)
	rows[0] = serviceRow(0, 0, content.AvailabilityCityOnly, gp(1))
	if got := table.ForSpellLevel(levelOf(t, 0)).Price().Cost(); got != gp(30) {
		t.Errorf("level 0 after mutating the source = %v, want 30 GP", got)
	}
}
