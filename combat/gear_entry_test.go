package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type entryStep struct {
	who     string
	trigger combat.AreaTrigger
	round   int
	turnOf  string
	want    string // "roll", "damage" or the String of a rejection
}

// runEntrySteps assesses each step in turn, carrying the ledger of the entries that rolled or hurt.
func runEntrySteps(t *testing.T, gear combat.DeployedGear, steps []entryStep) {
	t.Helper()
	var ledger combat.EntryLedger
	for i, s := range steps {
		a, err := combat.AreaEntryRules{}.Assess(gear, gid(s.who), s.trigger, gearStamp(s.round, s.turnOf), ledger)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		got := ""
		switch v := a.(type) {
		case combat.GearEntryRollRequired:
			got, ledger = "roll", v.Ledger()
		case combat.GearEntryDamageDue:
			got, ledger = "damage", v.Ledger()
		case combat.GearEntryRefused:
			got = v.Reason().String()
		}
		if got != s.want {
			t.Errorf("step %d (%+v) = %q, want %q", i, s, got, s.want)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-016, FR-017, FR-018, FR-021, FR-022, table 4.9.1 - the once-per-turn rows: a creature's first entry on a turn stamp needs the save and a second by it on the same stamp is refused; the same creature on another round, or on another creature's turn, saves again; two creatures on one stamp both save; lit Oil hurts once per turn across entering and ending a turn but twice when the stamps differ; ending a turn in Caltrops does not apply; unlit or burned-through Oil does not burn; Oil whose lighter's turn passed without Expire still burns (SRD 5.2.1, "Caltrops", "Ball Bearings", p. 96; "Oil", p. 98).
// Path: per case.
func TestAreaEntryRulesOncePerTurn(t *testing.T) {
	t.Parallel()
	enters, ends := combat.AreaTriggerEnters, combat.AreaTriggerEndsTurn
	caltrops := gearOf(t, gearOne(t, combat.DeployedGearKindCaltrops))
	oil := gearOf(t, gearBurning(t, 1, "a")) // burns through (3, a)
	unlit := gearOf(t, gearOne(t, combat.DeployedGearKindPouredOil))
	cases := []struct {
		name, path string
		gear       combat.DeployedGear
		steps      []entryStep
	}{
		{"first entry", "happy", caltrops, []entryStep{{"a", enters, 1, "a", "roll"}}},
		{"second entry on the same turn", "unhappy (rule interaction)", caltrops, []entryStep{{"a", enters, 1, "a", "roll"}, {"a", enters, 1, "a", "already_triggered_this_turn"}}},
		{"next round", "unhappy (boundary)", caltrops, []entryStep{{"a", enters, 1, "a", "roll"}, {"a", enters, 2, "a", "roll"}}},
		{"another creature's turn", "unhappy (rule interaction)", caltrops, []entryStep{{"a", enters, 1, "a", "roll"}, {"a", enters, 1, "b", "roll"}}},
		{"two creatures on one turn", "unhappy (rule interaction)", caltrops, []entryStep{{"a", enters, 1, "a", "roll"}, {"b", enters, 1, "a", "roll"}}},
		{"caltrops, ending a turn", "unhappy (invalid input)", caltrops, []entryStep{{"a", ends, 1, "a", "trigger_not_applicable"}}},
		{"oil, entering then ending the turn", "unhappy (rule interaction)", oil, []entryStep{{"a", enters, 1, "a", "damage"}, {"a", ends, 1, "a", "already_triggered_this_turn"}}},
		{"oil, ending the turn then entering", "unhappy (rule interaction)", oil, []entryStep{{"a", ends, 1, "a", "damage"}, {"a", enters, 1, "a", "already_triggered_this_turn"}}},
		{"oil, another's turn then its own", "unhappy (rule interaction)", oil, []entryStep{{"a", enters, 1, "b", "damage"}, {"a", ends, 1, "a", "damage"}}},
		{"oil, two creatures", "unhappy (rule interaction)", oil, []entryStep{{"a", enters, 2, "a", "damage"}, {"b", enters, 2, "a", "damage"}}},
		{"oil, unlit", "unhappy (state error)", unlit, []entryStep{{"a", enters, 1, "a", "oil_not_burning"}}},
		{"oil burned through, a round later", "unhappy (boundary)", oil, []entryStep{{"a", enters, 4, "b", "oil_not_burning"}}},
		{"oil, the last round before it burns out", "unhappy (boundary)", oil, []entryStep{{"a", enters, 3, "b", "damage"}}},
		{"oil, host forgot Expire", "unhappy (state error)", oil, []entryStep{{"c", enters, 3, "b", "damage"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			runEntrySteps(t, c.gear, c.steps)
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-016, FR-019, FR-020, FR-021, FR-022, FR-023 - Assess asks for a Dexterity save at DC 15 for Caltrops, 10 for Ball Bearings and 13 for a Hunting Trap, which takes no ledger record and refuses a sprung trap and ending a turn; lit Oil is 5 Fire damage with no save (SRD 5.2.1, pp. 96-98).
// Path: per case.
func TestAreaEntryRulesAssessment(t *testing.T) {
	t.Parallel()
	for kind, dc := range map[combat.DeployedGearKind]d20.DifficultyClass{combat.DeployedGearKindCaltrops: 15, combat.DeployedGearKindBallBearings: 10, combat.DeployedGearKindHuntingTrap: 13} {
		a, err := combat.AreaEntryRules{}.Assess(gearOf(t, gearOne(t, kind)), gid("a"), combat.AreaTriggerEnters, gearStamp(1, "a"), combat.EntryLedger{})
		r, ok := a.(combat.GearEntryRollRequired)
		if err != nil || !ok || r.DifficultyClass() != dc || r.Ability() != ability.Dexterity {
			t.Errorf("%v: Assess = %#v, %v; want a Dexterity save at DC %d", kind, a, err, dc)
		}
		if (kind == combat.DeployedGearKindHuntingTrap) != (r.Ledger().Len() == 0) {
			t.Errorf("%v: ledger holds %d records; only a Hunting Trap takes none", kind, r.Ledger().Len())
		}
	}
	a, _ := combat.AreaEntryRules{}.Assess(gearOf(t, gearBurning(t, 1, "a")), gid("a"), combat.AreaTriggerEnters, gearStamp(1, "a"), combat.EntryLedger{})
	due, ok := a.(combat.GearEntryDamageDue)
	if !ok || due.Ledger().Len() != 1 {
		t.Fatalf("Assess on lit oil = %#v, want damage due with one record", a)
	}
	fx := due.Outcome().Effects()
	if len(fx) != 1 || fx[0] != (combat.GearDamageEffect{Damage: combat.MustDamage(5, content.DamageFire)}) {
		t.Errorf("oil effects = %#v, want 5 Fire", fx)
	}
	fx[0] = nil
	if due.Outcome().Effects()[0] == nil {
		t.Error("the effects must be a copy")
	}
	trap := gearOf(t, gearSprung(t))
	runEntrySteps(t, trap, []entryStep{{"a", combat.AreaTriggerEnters, 1, "a", "trap_already_sprung"}, {"a", combat.AreaTriggerEndsTurn, 1, "a", "trigger_not_applicable"}})
	armed := gearOf(t, gearOne(t, combat.DeployedGearKindHuntingTrap))
	runEntrySteps(t, armed, []entryStep{{"a", combat.AreaTriggerEndsTurn, 1, "a", "trigger_not_applicable"}, {"a", combat.AreaTriggerEnters, 1, "a", "roll"}, {"a", combat.AreaTriggerEnters, 1, "a", "roll"}})
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-021, FR-071 - Assess returns a GearError for a zero gear or creature (blank_id), a trigger that is not valid or the zero stamp (invalid_value) and an ending of a turn stamped with another creature's turn (end_turn_stamp_mismatch), never a refusal.
// Path: unhappy (invalid input).
func TestAreaEntryRulesAssessInvalid(t *testing.T) {
	t.Parallel()
	oil := gearOf(t, gearBurning(t, 1, "a"))
	cases := []struct {
		name    string
		gear    combat.DeployedGear
		who     combat.CombatantID
		trigger combat.AreaTrigger
		stamp   combat.TurnStamp
		want    combat.GearProblem
	}{
		{"zero gear", combat.DeployedGear{}, "a", combat.AreaTriggerEnters, gearStamp(1, "a"), combat.GearProblemBlankID},
		{"zero creature", oil, "", combat.AreaTriggerEnters, gearStamp(1, "a"), combat.GearProblemBlankID},
		{"invalid trigger", oil, "a", 0, gearStamp(1, "a"), combat.GearProblemInvalidValue},
		{"zero stamp", oil, "a", combat.AreaTriggerEnters, combat.TurnStamp{}, combat.GearProblemInvalidValue},
		{"ending another creature's turn", oil, "a", combat.AreaTriggerEndsTurn, gearStamp(1, "b"), combat.GearProblemEndTurnStampMismatch},
		{"caltrops, ending another creature's turn", gearOf(t, gearOne(t, combat.DeployedGearKindCaltrops)), "a", combat.AreaTriggerEndsTurn, gearStamp(1, "b"), combat.GearProblemEndTurnStampMismatch},
	}
	for _, c := range cases {
		a, err := combat.AreaEntryRules{}.Assess(c.gear, c.who, c.trigger, c.stamp, combat.EntryLedger{})
		if a != nil || !errors.Is(err, &combat.GearError{Problem: c.want}) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: Assess = %v, %v; want a %v GearError", c.name, a, err, c.want)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-019, FR-020, FR-023 - Resolve gives GearEntrySaved for a passed save, else the effects of the gear: Caltrops 1 Piercing and Speed 0; Ball Bearings Prone; a Hunting Trap the 1d4 as Piercing damage, Speed 0 and a caught hold (the given id, DC 13, not restraining, gear id, limit 3) (SRD 5.2.1, pp. 96-98).
// Path: per case.
func TestAreaEntryRulesResolve(t *testing.T) {
	t.Parallel()
	piercing := func(n int) combat.GearEffect {
		return combat.GearDamageEffect{Damage: combat.MustDamage(n, content.DamagePiercing)}
	}
	rules := combat.AreaEntryRules{}
	caltrops, bearings, trap := gearOf(t, gearOne(t, combat.DeployedGearKindCaltrops)), gearOf(t, gearOne(t, combat.DeployedGearKindBallBearings)), gearOf(t, gearOne(t, combat.DeployedGearKindHuntingTrap))
	for _, g := range []combat.DeployedGear{caltrops, bearings, trap} {
		if o, err := rules.Resolve(g, gid("a"), hid("h"), true, 0); err != nil || o != (combat.GearEntrySaved{}) {
			t.Errorf("%v saved: Resolve = %#v, %v; want GearEntrySaved", g.Kind(), o, err)
		}
	}
	o, err := rules.Resolve(caltrops, gid("a"), hid("h"), false, 0)
	if f, ok := o.(combat.GearEntryFailed); err != nil || !ok || len(f.Effects()) != 2 || f.Effects()[0] != piercing(1) || f.Effects()[1] != (combat.GearSpeedZeroEffect{}) {
		t.Errorf("caltrops failed = %#v, %v; want 1 Piercing and Speed 0", o, err)
	}
	o, err = rules.Resolve(bearings, gid("a"), hid("h"), false, 0)
	if f, ok := o.(combat.GearEntryFailed); err != nil || !ok || len(f.Effects()) != 1 || f.Effects()[0] != (combat.GearConditionEffect{Condition: content.ConditionProne}) {
		t.Errorf("ball bearings failed = %#v, %v; want Prone", o, err)
	}
	for n := 1; n <= 4; n++ {
		o, err = rules.Resolve(trap, gid("a"), hid("h"), false, n)
		f, ok := o.(combat.GearEntryFailed)
		if err != nil || !ok || len(f.Effects()) != 3 || f.Effects()[0] != piercing(n) || f.Effects()[1] != (combat.GearSpeedZeroEffect{}) {
			t.Fatalf("trap failed with %d = %#v, %v", n, o, err)
		}
		hold := f.Effects()[2].(combat.GearCaughtEffect).Hold
		want := gearHold(t, combat.GearHoldInput{ID: hid("h"), Kind: combat.HoldKindHuntingTrap, Target: gid("a"), Restraint: combat.HoldRestraintDoesNotRestrain, GearID: gearID("g"), MovementLimitFeet: 3})
		if hold != want || hold.EscapeDifficultyClass() != 13 {
			t.Errorf("caught hold = %+v, want %+v [%s FR-023]", hold, want, gearSpec)
		}
	}
	f := o.(combat.GearEntryFailed)
	f.Effects()[0] = nil
	if f.Effects()[0] == nil {
		t.Error("the effects must be a copy")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-023, FR-071 - Resolve returns a GearError for a zero gear, creature or hold id (blank_id), for Oil, which has no save, and for a failed Hunting Trap save whose damage is outside 1 to 4 (invalid_value); a saved trap needs no damage.
// Path: unhappy (invalid input, boundary).
func TestAreaEntryRulesResolveInvalid(t *testing.T) {
	t.Parallel()
	rules, trap := combat.AreaEntryRules{}, gearOf(t, gearOne(t, combat.DeployedGearKindHuntingTrap))
	for name, c := range map[string]struct {
		gear   combat.DeployedGear
		who    combat.CombatantID
		hold   combat.HoldID
		saved  bool
		damage int
		want   combat.GearProblem
	}{
		"zero gear":         {combat.DeployedGear{}, "a", "h", false, 1, combat.GearProblemBlankID},
		"zero creature":     {trap, "", "h", false, 1, combat.GearProblemBlankID},
		"blank hold id":     {trap, "a", "", false, 1, combat.GearProblemBlankID},
		"oil has no save":   {gearOf(t, gearBurning(t, 1, "a")), "a", "h", false, 0, combat.GearProblemInvalidValue},
		"damage 0":          {trap, "a", "h", false, 0, combat.GearProblemInvalidValue},
		"damage 5":          {trap, "a", "h", false, 5, combat.GearProblemInvalidValue},
		"smallest damage":   {trap, "a", "h", false, math.MinInt, combat.GearProblemInvalidValue},
		"largest damage":    {trap, "a", "h", false, math.MaxInt, combat.GearProblemInvalidValue},
		"oil even if saved": {gearOf(t, gearBurning(t, 1, "a")), "a", "h", true, 0, combat.GearProblemInvalidValue},
	} {
		if o, err := rules.Resolve(c.gear, c.who, c.hold, c.saved, c.damage); o != nil || !errors.Is(err, &combat.GearError{Problem: c.want}) {
			t.Errorf("%s: Resolve = %v, %v; want a %v GearError", name, o, err, c.want)
		}
	}
	if o, err := rules.Resolve(trap, gid("a"), hid("h"), true, 99); err != nil || o != (combat.GearEntrySaved{}) {
		t.Errorf("a saved trap ignores the damage: Resolve = %v, %v", o, err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-018 - the ledger key is gear, creature and stamp: records of two gears, two creatures or two stamps never share; Record returns a new ledger and is idempotent, the ledger is copied and compares equal whatever the order; NewEntryLedger rejects an invalid record (SRD 5.2.1, p. 96).
// Path: unhappy (rule interaction, immutability).
func TestEntryLedger(t *testing.T) {
	t.Parallel()
	rec := func(gear, who string, round int) combat.EntryRecord {
		return combat.EntryRecord{GearID: gearID(gear), Creature: gid(who), Stamp: gearStamp(round, "a")}
	}
	base := combat.EntryLedger{}.Record(rec("g", "a", 1))
	if base.Len() != 1 || !base.Has(rec("g", "a", 1)) || base.Has(rec("h", "a", 1)) || base.Has(rec("g", "b", 1)) || base.Has(rec("g", "a", 2)) {
		t.Error("a record must be found by gear, creature and stamp only")
	}
	if (combat.EntryLedger{}).Len() != 0 || base.Record(rec("g", "a", 1)).Len() != 1 {
		t.Error("Record of a known record must not grow the ledger")
	}
	two := base.Record(rec("g", "b", 1))
	if base.Len() != 1 || two.Len() != 2 {
		t.Error("Record must return a new ledger")
	}
	l1, err := combat.NewEntryLedger([]combat.EntryRecord{rec("g", "a", 1), rec("g", "b", 1), rec("g", "a", 1)})
	l2, err2 := combat.NewEntryLedger([]combat.EntryRecord{rec("g", "b", 1), rec("g", "a", 1)})
	if err != nil || err2 != nil || l1.Len() != 2 || !l1.Equal(l2) || !l2.Equal(l1) || !l1.Equal(two) || l1.Equal(base) || base.Equal(two) {
		t.Errorf("ledgers = %+v %+v: order and repeats must not matter (%v, %v)", l1, l2, err, err2)
	}
	if !l1.Equal(l1) || !(combat.EntryLedger{}).Equal(combat.EntryLedger{}) {
		t.Error("a ledger must equal itself")
	}
	if l3, _ := combat.NewEntryLedger([]combat.EntryRecord{rec("g", "a", 1), rec("g", "b", 1)}); l3.Equal(base.Record(rec("g", "c", 1))) {
		t.Error("ledgers with different records must differ")
	}
	got := l1.Records()
	got[0] = combat.EntryRecord{}
	if l1.Records()[0] == (combat.EntryRecord{}) {
		t.Error("Records must return a copy")
	}
	in := []combat.EntryRecord{rec("g", "a", 1)}
	l4, _ := combat.NewEntryLedger(in)
	in[0] = rec("x", "x", 9)
	if !l4.Has(rec("g", "a", 1)) {
		t.Error("NewEntryLedger must copy its input")
	}
	for name, bad := range map[string]combat.EntryRecord{"blank gear": {Creature: "a", Stamp: gearStamp(1, "a")}, "zero creature": {GearID: "g", Stamp: gearStamp(1, "a")}, "zero stamp": {GearID: "g", Creature: "a"}} {
		if _, err := combat.NewEntryLedger([]combat.EntryRecord{bad}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}
