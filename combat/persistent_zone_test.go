package combat_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/persistent-spell-zones.md FR-005 - a spell whose effects are
// text only, or that has no zone facts, is refused with NoZoneFacts and no event
// (spell-effect-data.md RA-1; SRD 5.2.1, "Effects", p. 106).
// Path: per case.
func TestCreateZoneRefusesSpellsWithoutZoneFacts(t *testing.T) {
	t.Parallel()
	plain := seSpell("plain", 1, seModel(content.CoverageComplete, seArea(content.SubjectCreature, nil), pzParts(1)...), nil)
	cases := []struct {
		name, requirement, path string
		spell                   content.Spell
	}{
		{"text only effects", "FR-005 text-only effects", "unhappy (missing content)", seSpell("text", 1, content.TextOnlyEffects{}, nil)},
		{"modelled without zone facts", "FR-005 no zone facts", "unhappy (missing content)", plain},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := combat.CreateZone(combat.CreateZoneCommand{
				ID: luMust(combat.NewPersistentZoneID("z")), Caster: combat.MustCombatantID("c"),
				Spell: c.spell, Shape: luMust(content.NewSphereArea(5)), Cast: pzTurn(1, "c"),
			})
			want := combat.CreateZoneRejected{Reasons: []combat.CreateZoneRejection{combat.CreateNoZoneFacts}}
			if got, ok := res.(combat.CreateZoneRejected); !ok || !reflect.DeepEqual(got, want) {
				t.Fatalf("CreateZone = %#v, want %#v [%s]", res, want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-003 - every rule's PartIndex
// must name a part of the spell's model, else CreateZoneRejected with
// PartIndexOutOfRange (SRD 5.2.1, "Spike Growth", p. 164).
// Path: per case.
func TestCreateZoneRefusesPartIndexOutsideModel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		parts, part             int
		refused                 bool
	}{
		{"last part of one", "FR-003 index 0 of one part", "unhappy (boundary)", 1, 0, false},
		{"one past the last part", "FR-003 index 1 of one part", "unhappy (boundary)", 1, 1, true},
		{"negative index", "FR-003 negative index", "unhappy (boundary)", 1, -1, true},
		{"second part of two", "FR-003 index 1 of two parts", "happy", 2, 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, c.part, content.AnyCreature)}})
			spell := pzSpell("s", c.parts, facts, nil)
			res := combat.CreateZone(combat.CreateZoneCommand{
				ID: luMust(combat.NewPersistentZoneID("z")), Caster: combat.MustCombatantID("c"),
				Spell: spell, Shape: luMust(content.NewSphereArea(5)), Cast: pzTurn(1, "c"),
			})
			rej, refused := res.(combat.CreateZoneRejected)
			if refused != c.refused {
				t.Fatalf("CreateZone refused = %t, want %t [%s]", refused, c.refused, c.requirement)
			}
			if refused && (len(rej.Reasons) != 1 || rej.Reasons[0] != combat.CreatePartIndexOutOfRange) {
				t.Fatalf("Reasons = %v, want [PartIndexOutOfRange] [%s]", rej.Reasons, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-003 - a TravelsWithin rule
// needs a unit of at least 1 foot and no other trigger may carry one (content's
// NewZoneRuleFact, which the zone relies on).
// Path: unhappy (boundary).
func TestTravelsWithinRuleNeedsAUnit(t *testing.T) {
	t.Parallel()
	if _, err := content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.TravelsWithin, Scope: content.AnyCreature}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("TravelsWithin without a unit: error = %v, want ErrInvalidInput [FR-003]", err)
	}
	if _, err := content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, Scope: content.AnyCreature, UnitFeet: 5}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("CreatureEnters with a unit: error = %v, want ErrInvalidInput [FR-003]", err)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-007 - designations are accepted
// only when the spell designates at casting; duplicates are refused; every problem
// is listed in fixed order (PartIndexOutOfRange, DurationNotZone, DesignationNotAllowed,
// DuplicateExemption).
// Path: per case.
func TestCreateZoneListsEveryProblemInFixedOrder(t *testing.T) {
	t.Parallel()
	noDesignation := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}})
	cases := []struct {
		name, requirement, path string
		spell                   content.Spell
		designated              []combat.CombatantID
		want                    []combat.CreateZoneRejection
	}{
		{"designation allowed", "FR-007 designation allowed at casting", "happy", pzSpirit(), []combat.CombatantID{combat.MustCombatantID("ally")}, nil},
		{"designation not allowed", "FR-007 designation not allowed", "unhappy (invalid input)", pzSpell("d", 1, noDesignation, nil),
			[]combat.CombatantID{combat.MustCombatantID("ally")}, []combat.CreateZoneRejection{combat.CreateDesignationNotAllowed}},
		{"repeated designation", "FR-007 repeated id", "unhappy (invalid input)", pzSpirit(),
			[]combat.CombatantID{combat.MustCombatantID("ally"), combat.MustCombatantID("ally")}, []combat.CreateZoneRejection{combat.CreateDuplicateExemption}},
		{"not allowed and repeated", "FR-007 every problem listed", "unhappy (invalid input)", pzSpell("d", 1, noDesignation, nil),
			[]combat.CombatantID{combat.MustCombatantID("a"), combat.MustCombatantID("a")},
			[]combat.CreateZoneRejection{combat.CreateDesignationNotAllowed, combat.CreateDuplicateExemption}},
		{"bad part and instantaneous", "FR-003 with FR-009 in fixed order", "unhappy (invalid input)",
			pzSpell("p", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 3, content.AnyCreature)}}),
				content.NewSpellDurationInstantaneous()),
			nil, []combat.CreateZoneRejection{combat.CreatePartIndexOutOfRange, combat.CreateDurationNotZone}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := combat.CreateZone(combat.CreateZoneCommand{
				ID: luMust(combat.NewPersistentZoneID("z")), Caster: combat.MustCombatantID("c"),
				Spell: c.spell, Shape: luMust(content.NewSphereArea(5)), Cast: pzTurn(1, "c"), Designated: c.designated,
			})
			if c.want == nil {
				if _, ok := res.(combat.CreateZoneCreated); !ok {
					t.Fatalf("CreateZone = %T, want CreateZoneCreated [%s]", res, c.requirement)
				}
				return
			}
			rej, ok := res.(combat.CreateZoneRejected)
			if !ok || !reflect.DeepEqual(rej.Reasons, c.want) {
				t.Fatalf("CreateZone = %#v, want reasons %v [%s]", res, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-004 - a created zone is Active,
// with an empty ledger and no damage, and emits one ZoneCreated event.
// Path: happy.
func TestCreatedZoneIsActiveAndEmitsZoneCreated(t *testing.T) {
	t.Parallel()
	res := combat.CreateZone(combat.CreateZoneCommand{
		ID: luMust(combat.NewPersistentZoneID("spirit")), Caster: combat.MustCombatantID("cleric"),
		Spell: pzSpirit(), Shape: luMust(content.NewEmanationArea(15)), Cast: pzTurn(1, "cleric"),
	})
	created, ok := res.(combat.CreateZoneCreated)
	if !ok {
		t.Fatalf("CreateZone = %T, want CreateZoneCreated [FR-004]", res)
	}
	z := created.Zone
	if z.Ended() || z.Ledger().Size() != 0 || z.DamageDealt() != 0 {
		t.Fatalf("new zone ended=%t ledger=%d damage=%d, want active, empty, 0 [FR-004]", z.Ended(), z.Ledger().Size(), z.DamageDealt())
	}
	if created.Event.Kind() != "zone_created" || created.Event.ZoneID() != luMust(combat.NewPersistentZoneID("spirit")) {
		t.Fatalf("ZoneCreated = kind %q zone %q [FR-004]", created.Event.Kind(), created.Event.ZoneID())
	}
	if got := created.Event.Combatants(); len(got) != 1 || got[0] != combat.MustCombatantID("cleric") {
		t.Fatalf("ZoneCreated.Combatants() = %v, want the caster [FR-050]", got)
	}
	if z.Spell() != "spirit_guardians" || z.Combatants()[0] != combat.MustCombatantID("cleric") {
		t.Fatalf("zone spell %q caster %q [FR-002]", z.Spell(), z.Combatants()[0])
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-006 - the zone accepts any caster
// id; it does not cross-check the cast (SRD 5.2.1, "Spirit Guardians", p. 164).
// Path: unhappy (rule interaction).
func TestCreateZoneAcceptsAnyCasterID(t *testing.T) {
	t.Parallel()
	for _, caster := range []string{"caster", "someone_else", "a"} {
		zone := pzCreate(t, pzSpirit(), func(c *combat.CreateZoneCommand) { c.Caster = combat.MustCombatantID(caster) })
		if zone.Combatants()[0] != combat.MustCombatantID(caster) {
			t.Fatalf("zone caster = %q, want %q [FR-006]", zone.Combatants()[0], caster)
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-009 and case table 4.7b - a spell
// duration becomes a zone duration (SRD 5.2.1, "Duration", p. 106; "Combat Step by
// Step", p. 13). A refused duration is CreateZoneRejected with DurationNotZone.
// Path: per case.
func TestSpellDurationConvertsToZoneDuration(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		dur                     content.SpellDuration
		override                combat.ZoneDuration
		want                    combat.ZoneDuration
	}{
		{"spirit guardians, 10 minutes", "FR-009 Spirit Guardians p. 164", "happy", luMust(content.NewSpellDurationTimed(10, content.DurationUnitMinutes, content.LimitConcentrationUpTo)), nil, luMust(combat.NewRoundsDuration(100))},
		{"flaming sphere, 1 minute", "FR-009 Flaming Sphere p. 132", "happy", luMust(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitConcentrationUpTo)), nil, luMust(combat.NewRoundsDuration(10))},
		{"web, 1 hour", "FR-009 Web p. 174", "unhappy (boundary)", luMust(content.NewSpellDurationTimed(1, content.DurationUnitHours, content.LimitConcentrationUpTo)), nil, luMust(combat.NewRoundsDuration(600))},
		{"guardian of faith, 8 hours", "FR-009 Guardian of Faith p. 137", "unhappy (boundary)", luMust(content.NewSpellDurationTimed(8, content.DurationUnitHours, content.LimitUpTo)), nil, luMust(combat.NewRoundsDuration(4800))},
		{"seven days", "FR-009 a duration in days is host timed", "unhappy (boundary)", luMust(content.NewSpellDurationTimed(7, content.DurationUnitDays, content.LimitUpTo)), nil, combat.HostTimedDuration{}},
		{"six rounds", "FR-009 a duration in rounds (Tsunami fixture, p. 171)", "unhappy (boundary)", luMust(content.NewSpellDurationTimed(6, content.DurationUnitRounds, content.LimitConcentrationUpTo)), nil, luMust(combat.NewRoundsDuration(6))},
		{"until dispelled", "FR-009 until dispelled (Continual Flame p. 119)", "unhappy (rule interaction)", content.NewSpellDurationUntilDispelled(), nil, combat.UntilDispelled{}},
		{"symbol until dispelled or triggered", "FR-009 Symbol p. 167 with the host override", "unhappy (rule interaction)", content.NewSpellDurationUntilDispelledOrTriggered(), luMust(combat.NewRoundsDuration(100)), luMust(combat.NewRoundsDuration(100))},
		{"symbol without override", "FR-009 Symbol is until dispelled", "unhappy (rule interaction)", content.NewSpellDurationUntilDispelledOrTriggered(), nil, combat.UntilDispelled{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			spell := pzSpell("s", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}}), c.dur)
			zone := pzCreate(t, spell, func(cmd *combat.CreateZoneCommand) { cmd.DurationOverride = c.override })
			if got := zone.Duration(); got != c.want {
				t.Fatalf("Duration() = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-009 - a timed duration whose round count
// is past int is HostTimedDuration, not a wrapped count: the largest minutes and hours that fit
// are rounds, one more unit is host timed (SRD 5.2.1, "Duration", p. 106; "Combat Step by Step",
// p. 13).
// Path: per case.
func TestSpellDurationPastIntIsHostTimed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		amount                  int
		unit                    content.DurationUnit
		want                    combat.ZoneDuration
	}{
		{"largest minutes that fit", "FR-009 minutes at int's limit", "unhappy (boundary)", math.MaxInt / 10, content.DurationUnitMinutes, luMust(combat.NewRoundsDuration(math.MaxInt / 10 * 10))},
		{"one minute past int", "FR-009 minutes past int is host timed", "unhappy (boundary)", math.MaxInt/10 + 1, content.DurationUnitMinutes, combat.HostTimedDuration{}},
		{"largest hours that fit", "FR-009 hours at int's limit", "unhappy (boundary)", math.MaxInt / 600, content.DurationUnitHours, luMust(combat.NewRoundsDuration(math.MaxInt / 600 * 600))},
		{"one hour past int", "FR-009 hours past int is host timed", "unhappy (boundary)", math.MaxInt/600 + 1, content.DurationUnitHours, combat.HostTimedDuration{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dur := luMust(content.NewSpellDurationTimed(c.amount, c.unit, content.LimitUpTo))
			spell := pzSpell("s", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}}), dur)
			if got := pzCreate(t, spell, nil).Duration(); got != c.want {
				t.Fatalf("Duration() = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-009 - an Instantaneous or Special
// duration is refused as DurationNotZone when the command gives no override.
// Path: unhappy (rule interaction).
func TestInstantaneousAndSpecialDurationsAreRefused(t *testing.T) {
	t.Parallel()
	for _, dur := range []content.SpellDuration{content.NewSpellDurationInstantaneous(), content.NewSpellDurationSpecial()} {
		spell := pzSpell("s", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}}), dur)
		res := combat.CreateZone(combat.CreateZoneCommand{
			ID: luMust(combat.NewPersistentZoneID("z")), Caster: combat.MustCombatantID("c"),
			Spell: spell, Shape: luMust(content.NewSphereArea(5)), Cast: pzTurn(1, "c"),
		})
		rej, ok := res.(combat.CreateZoneRejected)
		if !ok || len(rej.Reasons) != 1 || rej.Reasons[0] != combat.CreateDurationNotZone {
			t.Fatalf("CreateZone(%T) = %#v, want [DurationNotZone] [FR-009]", dur, res)
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-009 - NewRoundsDuration gives an
// *ZoneInvalidError below one round (boundary at 0 and 1).
// Path: per case.
func TestNewRoundsDurationBoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		rounds                  int
		wantErr                 bool
	}{
		{"zero rounds", "FR-009 below one round", "unhappy (boundary)", 0, true},
		{"one round", "FR-009 one round", "happy", 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := combat.NewRoundsDuration(c.rounds)
			if c.wantErr {
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Fatalf("NewRoundsDuration(%d) error = %v [%s]", c.rounds, err, c.requirement)
				}
				return
			}
			if err != nil || d.Rounds() != c.rounds {
				t.Fatalf("NewRoundsDuration(%d) = %v, %v [%s]", c.rounds, d, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-010 - the zone records that its
// spell requires Concentration; the engine does not detect Concentration itself.
// Path: happy.
func TestZoneRecordsConcentrationRequirement(t *testing.T) {
	t.Parallel()
	if !pzCreate(t, pzSpirit(), nil).EndsWithConcentration() {
		t.Fatalf("Spirit Guardians EndsWithConcentration = false, want true [FR-010]")
	}
	instant := pzSpell("i", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}}), content.NewSpellDurationUntilDispelled())
	if pzCreate(t, instant, nil).EndsWithConcentration() {
		t.Fatalf("EndsWithConcentration = true for an until-dispelled spell, want false [FR-010]")
	}
}
