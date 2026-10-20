package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

func zoneRule(trigger content.ZoneTrigger, unit int) content.ZoneRuleFields {
	return content.ZoneRuleFields{Trigger: trigger, Scope: content.AnyCreature, UnitFeet: unit}
}

// Requirement: spell-effect-data.md FR-048 - UnitFeet is set (at least 1) for
// TravelsWithin and for no other trigger; the region defaults to "body";
// triggers, scope and region are validated.
// Path: per case.
func TestZoneRuleFact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		f       content.ZoneRuleFields
		problem content.BlockProblem // 0 when valid
	}{
		{"TravelsWithin with 1 foot", zoneRule(content.TravelsWithin, 1), 0},
		{"TravelsWithin with 5 feet", zoneRule(content.TravelsWithin, 5), 0},
		{"TravelsWithin with no distance", zoneRule(content.TravelsWithin, 0), content.BlockConstraint},
		{"TravelsWithin with -1", zoneRule(content.TravelsWithin, -1), content.BlockOutOfRange},
		{"CreatureEnters with no distance", zoneRule(content.CreatureEnters, 0), 0},
		{"CreatureEnters with a distance", zoneRule(content.CreatureEnters, 5), content.BlockConstraint},
		{"CreatureEnters with -1", zoneRule(content.CreatureEnters, -1), content.BlockOutOfRange},
		{"no trigger", zoneRule(0, 0), content.BlockInvalidMember},
		{"no scope", content.ZoneRuleFields{Trigger: content.CreatureEnters}, content.BlockInvalidMember},
		{"blank region", content.ZoneRuleFields{Trigger: content.CreatureEnters, Scope: content.AnyCreature, Region: " "}, content.BlockInvalidName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := content.NewZoneRuleFact(c.f)
			if c.problem == 0 {
				if err != nil {
					t.Fatalf("NewZoneRuleFact = %v, want it accepted", err)
				}
				if r.Trigger() != c.f.Trigger || r.UnitFeet() != c.f.UnitFeet || r.Scope() != c.f.Scope || r.Region() != content.DefaultZoneRegion {
					t.Errorf("rule = %v, want the fields given and the default region", r)
				}
				return
			}
			block := "ZoneRuleFact"
			if c.problem == content.BlockInvalidName {
				block = "ZoneRegionID"
			}
			wantProblem(t, err, block, c.problem)
		})
	}
	r := must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, PartIndex: 2, Scope: content.EnemiesOfCaster, Region: "wall", CasterChoice: true}))
	if r.PartIndex() != 2 || r.Region() != "wall" || !r.CasterChoice() || r.Scope() != content.EnemiesOfCaster || r.Region().String() != "wall" {
		t.Errorf("accessors: %v", r)
	}
	if _, err := content.NewZoneRegionID(""); err == nil {
		t.Error("an empty region id was accepted")
	}
	if _, err := content.NewZoneRegionID("body"); err != nil {
		t.Errorf("NewZoneRegionID(body) = %v", err)
	}
}

// Requirement: spell-effect-data.md FR-048 - mobility variants: MaxFeet and Feet
// are at least 1 (0 and 1 tried), and the action and direction are valid.
// Path: unhappy (boundary).
func TestZoneMobility(t *testing.T) {
	t.Parallel()
	_, err := content.NewMovedByCaster(0, content.BonusAction, false)
	wantProblem(t, err, "MovedByCaster", content.BlockOutOfRange)
	_, err = content.NewMovedByCaster(30, 0, false)
	wantProblem(t, err, "MovedByCaster", content.BlockInvalidMember)
	mv, err := content.NewMovedByCaster(1, content.MagicAction, true)
	if err != nil || mv.MaxFeet() != 1 || mv.Action() != content.MagicAction || !mv.LaterTurnsOnly() {
		t.Errorf("NewMovedByCaster(1) = %v, %v", mv, err)
	}
	_, err = content.NewDriftsAtTurnStart(0, content.AwayFromCaster)
	wantProblem(t, err, "DriftsAtTurnStart", content.BlockOutOfRange)
	_, err = content.NewDriftsAtTurnStart(10, 0)
	wantProblem(t, err, "DriftsAtTurnStart", content.BlockInvalidMember)
	d, err := content.NewDriftsAtTurnStart(1, content.AwayFromCasterChosen)
	if err != nil || d.Feet() != 1 || d.Direction() != content.AwayFromCasterChosen {
		t.Errorf("NewDriftsAtTurnStart(1) = %v, %v", d, err)
	}
}

// Requirement: spell-effect-data.md FR-048 - ExtraFeetPerFoot is at least 0.
// Path: unhappy (boundary).
func TestZoneMovementFacts(t *testing.T) {
	t.Parallel()
	_, err := content.NewZoneMovementFacts(true, false, -1)
	wantProblem(t, err, "ZoneMovementFacts", content.BlockOutOfRange)
	for _, n := range []int{0, 3} {
		m, err := content.NewZoneMovementFacts(true, true, n)
		if err != nil || !m.DifficultTerrain() || !m.SpeedHalved() || m.ExtraFeetPerFoot() != n {
			t.Errorf("NewZoneMovementFacts(true, true, %d) = %v, %v", n, m, err)
		}
	}
	if (content.ZoneMovementFacts{}) != must(content.NewZoneMovementFacts(false, false, 0)) {
		t.Error("the zero ZoneMovementFacts is not the no-effect value")
	}
}

// Requirement: spell-effect-data.md FR-048 - ZoneFacts has at least one rule,
// no repeated rule, a valid mobility, designation, budget and movement; nil and
// zero fields read as their defaults; the rules are copied and compared as a set.
// Path: per case.
func TestZoneFacts(t *testing.T) {
	t.Parallel()
	enter := must(content.NewZoneRuleFact(zoneRule(content.CreatureEnters, 0)))
	travel := must(content.NewZoneRuleFact(zoneRule(content.TravelsWithin, 5)))

	_, err := content.NewZoneFacts(content.ZoneFactsFields{})
	wantProblem(t, err, "ZoneFacts", content.BlockEmpty)
	_, err = content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{enter, enter}})
	wantProblem(t, err, "ZoneFacts", content.BlockDuplicate)
	_, err = content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{{}}})
	wantProblem(t, err, "ZoneRuleFact", content.BlockInvalidMember)
	_, err = content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{enter}, Mobility: content.MovedByCaster{}})
	wantProblem(t, err, "MovedByCaster", content.BlockOutOfRange)
	_, err = content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{enter}, Designation: 9})
	wantProblem(t, err, "ZoneFacts", content.BlockInvalidMember)
	_, err = content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{enter}, DamageBudget: -1})
	wantProblem(t, err, "ZoneFacts", content.BlockOutOfRange)
	_, err = content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{enter}, Movement: content.ZoneMovementFacts{}})
	if err != nil {
		t.Errorf("the zero movement facts: %v", err)
	}

	z := must(content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{enter}}))
	if z.Mobility() != content.ZoneMobility(content.StationaryZone{}) || z.Designation() != content.NoDesignation || z.DamageBudget() != 0 ||
		z.OncePerTurn() || z.StopsMoverOnContact() || z.Movement() != (content.ZoneMovementFacts{}) {
		t.Errorf("defaults = %v", z)
	}
	full := must(content.NewZoneFacts(content.ZoneFactsFields{
		Rules: []content.ZoneRuleFact{enter, travel}, Mobility: content.MovesWithOrigin{}, OncePerTurn: true,
		Designation: content.DesignationAtCastingOnly, DamageBudget: 60, StopsMoverOnContact: true,
		Movement: must(content.NewZoneMovementFacts(true, true, 3)),
	}))
	if len(full.Rules()) != 2 || full.Mobility() != content.ZoneMobility(content.MovesWithOrigin{}) || !full.OncePerTurn() || full.DamageBudget() != 60 ||
		!full.StopsMoverOnContact() || full.Designation() != content.DesignationAtCastingOnly || full.Movement().ExtraFeetPerFoot() != 3 {
		t.Errorf("accessors = %v", full)
	}

	in := []content.ZoneRuleFact{enter, travel}
	copied := must(content.NewZoneFacts(content.ZoneFactsFields{Rules: in}))
	in[0] = travel
	copied.Rules()[1] = enter
	if got := copied.Rules(); got[0] != enter || got[1] != travel {
		t.Errorf("Rules() = %v after mutation, want the original order [FR-007]", got)
	}
	swapped := must(content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{travel, enter}}))
	if !swapped.Equal(copied) {
		t.Error("rules are order-sensitive [FR-008]")
	}
	for name, o := range map[string]content.ZoneFactsFields{
		"rule set":    {Rules: []content.ZoneRuleFact{enter}},
		"other rule":  {Rules: []content.ZoneRuleFact{enter, must(content.NewZoneRuleFact(zoneRule(content.TravelsWithin, 10)))}},
		"once":        {Rules: []content.ZoneRuleFact{enter, travel}, OncePerTurn: true},
		"designation": {Rules: []content.ZoneRuleFact{enter, travel}, Designation: content.DesignationAtCastingOnly},
		"budget":      {Rules: []content.ZoneRuleFact{enter, travel}, DamageBudget: 1},
		"stops":       {Rules: []content.ZoneRuleFact{enter, travel}, StopsMoverOnContact: true},
		"mobility":    {Rules: []content.ZoneRuleFact{enter, travel}, Mobility: content.MovesWithOrigin{}},
		"movement":    {Rules: []content.ZoneRuleFact{enter, travel}, Movement: must(content.NewZoneMovementFacts(true, false, 0))},
	} {
		if copied.Equal(must(content.NewZoneFacts(o))) {
			t.Errorf("zone facts differing in %s are equal", name)
		}
	}
}
