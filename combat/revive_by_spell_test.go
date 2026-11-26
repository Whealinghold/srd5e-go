package combat_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// rrReasons returns the reasons of an ineligible cast and nil for any other answer.
func rrReasons(res combat.ReviveCreatureBySpellResult) []combat.RevivalIneligibility {
	if i, ok := res.(combat.ReviveBySpellIneligible); ok {
		return i.Reasons()
	}
	return nil
}

// Requirement: per case (table 4.1b, one test per row; the Baseline is an Intact body, not old age, not Undead, a spirit that accepts)
// Path: per case.
func TestReviveBySpellEligibility(t *testing.T) {
	t.Parallel()
	const (
		tw   = combat.TimeWindowExceeded
		old  = combat.DiedOfOldAge
		und  = combat.WasUndead
		nob  = combat.NoBodyToTouch
		part = combat.IntegralPartsMissing
		nam  = combat.NameNotSpoken
	)
	cases := []struct {
		name, requirement, path string
		spell                   combat.RevivalSpell
		since                   time.Duration
		oldAge, undead          bool
		body                    combat.BodyState
		nameSpoken              bool
		want                    []combat.RevivalIneligibility
	}{
		{"inside the minute", `FR-002; SRD 5.2.1, "Revivify", p. 159`, "happy", combat.Revivify, 59 * time.Second, false, false, combat.Intact, false, nil},
		{"exactly one minute", `FR-002; SRD 5.2.1, "Revivify", p. 159`, "unhappy (boundary)", combat.Revivify, time.Minute, false, false, combat.Intact, false, nil},
		{"one second late", `FR-002; SRD 5.2.1, "Revivify", p. 159`, "unhappy (boundary)", combat.Revivify, 61 * time.Second, false, false, combat.Intact, false, []combat.RevivalIneligibility{tw}},
		{"just born dead", `FR-008; SRD 5.2.1, "Revivify", p. 159`, "unhappy (empty and minimal)", combat.Revivify, 0, false, false, combat.Intact, false, nil},
		{"10 days exactly", `FR-002; SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (boundary)", combat.RaiseDead, 10 * rrDay, false, false, combat.Intact, false, nil},
		{"10 days and 1 second", `FR-002; SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (boundary)", combat.RaiseDead, 10*rrDay + time.Second, false, false, combat.Intact, false, []combat.RevivalIneligibility{tw}},
		{"100 years exactly", `FR-002; SRD 5.2.1, "Resurrection", p. 159`, "unhappy (boundary)", combat.Resurrection, 36500 * rrDay, false, false, combat.Intact, false, nil},
		{"past a century", `FR-002; SRD 5.2.1, "Resurrection", p. 159`, "unhappy (boundary)", combat.Resurrection, 36500*rrDay + time.Second, false, false, combat.Intact, false, []combat.RevivalIneligibility{tw}},
		{"200 years exactly", `FR-002; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (boundary)", combat.TrueResurrection, 73000 * rrDay, false, false, combat.Intact, false, nil},
		{"past 200 years", `FR-002; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (boundary)", combat.TrueResurrection, 73000*rrDay + time.Second, false, false, combat.Intact, false, []combat.RevivalIneligibility{tw}},
		{"old age, Revivify", `FR-003; SRD 5.2.1, "Revivify", p. 159`, "unhappy (rule interaction)", combat.Revivify, 0, true, false, combat.Intact, false, []combat.RevivalIneligibility{old}},
		{"old age, Raise Dead", `FR-003 (Q-1); SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (rule interaction)", combat.RaiseDead, 0, true, false, combat.Intact, false, nil},
		{"old age, Resurrection", `FR-003; SRD 5.2.1, "Resurrection", p. 159`, "unhappy (rule interaction)", combat.Resurrection, 0, true, false, combat.Intact, false, []combat.RevivalIneligibility{old}},
		{"old age, True Resurrection", `FR-003; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (rule interaction)", combat.TrueResurrection, 0, true, false, combat.Intact, false, []combat.RevivalIneligibility{old}},
		{"Undead, Revivify", `FR-004 (Q-1); SRD 5.2.1, "Revivify", p. 159`, "unhappy (rule interaction)", combat.Revivify, 0, false, true, combat.Intact, false, nil},
		{"Undead, Raise Dead", `FR-004; SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (rule interaction)", combat.RaiseDead, 0, false, true, combat.Intact, false, []combat.RevivalIneligibility{und}},
		{"Undead, Resurrection", `FR-004; SRD 5.2.1, "Resurrection", p. 159`, "unhappy (rule interaction)", combat.Resurrection, 0, false, true, combat.Intact, false, []combat.RevivalIneligibility{und}},
		{"Undead, True Resurrection", `FR-004; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (rule interaction)", combat.TrueResurrection, 0, false, true, combat.Intact, false, nil},
		{"no body", `FR-005; SRD 5.2.1, "Revivify", p. 159`, "unhappy (rule interaction)", combat.Revivify, 0, false, false, combat.NoBody, false, []combat.RevivalIneligibility{nob}},
		{"no body, Raise Dead", `FR-005; SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (rule interaction)", combat.RaiseDead, 0, false, false, combat.NoBody, true, []combat.RevivalIneligibility{nob}},
		{"no body, Resurrection with a spoken name", `FR-005, FR-006 (the name is ignored); SRD 5.2.1, "Resurrection", p. 159`, "unhappy (rule interaction)", combat.Resurrection, 0, false, false, combat.NoBody, true, []combat.RevivalIneligibility{nob}},
		{"head missing, Raise Dead", `FR-005; SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (rule interaction)", combat.RaiseDead, 0, false, false, combat.MissingIntegralParts, false, []combat.RevivalIneligibility{part}},
		{"head missing, Resurrection", `FR-005; SRD 5.2.1, "Resurrection", p. 159`, "unhappy (rule interaction)", combat.Resurrection, 0, false, false, combat.MissingIntegralParts, false, nil},
		{"head missing, Revivify, by design", `FR-005 (Q-3); SRD 5.2.1, "Revivify", p. 159`, "unhappy (rule interaction)", combat.Revivify, 0, false, false, combat.MissingIntegralParts, false, nil},
		{"head missing, True Resurrection", `FR-005; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (rule interaction)", combat.TrueResurrection, 0, false, false, combat.MissingIntegralParts, false, nil},
		{"arm missing, Raise Dead", `FR-005; SRD 5.2.1, "Raise Dead", p. 157`, "happy", combat.RaiseDead, 0, false, false, combat.MissingNonIntegralParts, false, nil},
		{"arm missing, Revivify", `FR-005; SRD 5.2.1, "Revivify", p. 159`, "happy", combat.Revivify, 0, false, false, combat.MissingNonIntegralParts, false, nil},
		{"no body, name not spoken", `FR-006; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (rule interaction)", combat.TrueResurrection, 0, false, false, combat.NoBody, false, []combat.RevivalIneligibility{nam}},
		{"no body, name spoken", `FR-006; SRD 5.2.1, "True Resurrection", p. 170`, "happy", combat.TrueResurrection, 0, false, false, combat.NoBody, true, nil},
		{"name spoken, body intact", `FR-006; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (boundary)", combat.TrueResurrection, 0, false, false, combat.Intact, true, nil},
		{"name not spoken, body intact", `FR-006; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (boundary)", combat.TrueResurrection, 0, false, false, combat.Intact, false, nil},
		{"several reasons in the fixed order", `FR-007; SRD 5.2.1, "Raise Dead", p. 157`, "unhappy (rule interaction)", combat.RaiseDead, 11 * rrDay, false, true, combat.MissingIntegralParts, false, []combat.RevivalIneligibility{tw, und, part}},
		{"every reason of Resurrection", `FR-007; SRD 5.2.1, "Resurrection", p. 159`, "unhappy (rule interaction)", combat.Resurrection, 40000 * rrDay, true, true, combat.NoBody, false, []combat.RevivalIneligibility{tw, old, und, nob}},
		{"every reason of True Resurrection", `FR-007; SRD 5.2.1, "True Resurrection", p. 170`, "unhappy (rule interaction)", combat.TrueResurrection, 80000 * rrDay, true, false, combat.NoBody, false, []combat.RevivalIneligibility{tw, old, nam}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			cmd := rrCommand(t, c.spell)
			cmd.Circumstances = rrCircs(t, c.since, c.oldAge, c.undead, c.body)
			cmd.NameSpoken = c.nameSpoken
			res := combat.ReviveBySpell(cmd)
			got := rrReasons(res)
			if !slices.Equal(got, c.want) {
				t.Errorf("%v, %v: reasons = %v, want %v [%s; %s]", c.spell, c.name, got, c.want, c.requirement, c.path)
			}
			_, applied := res.(combat.ReviveBySpellApplied)
			if applied != (c.want == nil) {
				t.Errorf("%v, %v: result %T, want applied = %v [%s]", c.spell, c.name, res, c.want == nil, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-004 - True Resurrection reports RestoredToNonUndead true on success when the
// creature was Undead and false otherwise; the other spells never do (SRD 5.2.1, "True Resurrection", p. 170).
// Path: unhappy (rule interaction).
func TestReviveBySpellRestoredToNonUndead(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		spell  combat.RevivalSpell
		undead bool
		want   bool
	}{{combat.TrueResurrection, true, true}, {combat.TrueResurrection, false, false}, {combat.Revivify, true, false}, {combat.Revivify, false, false}} {
		cmd := rrCommand(t, c.spell)
		cmd.Circumstances = rrCircs(t, 0, false, c.undead, combat.Intact)
		if got := rrApplied(t, cmd).RestoredToNonUndead; got != c.want {
			t.Errorf("%v, Undead %v: RestoredToNonUndead = %v, want %v", c.spell, c.undead, got, c.want)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-007 - an ineligible cast changes nothing (the command's status, conditions and
// effects are as passed), emits no event and carries neither ordeal nor strain; a spell that is not a revival spell is ineligible with no
// reasons (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (rule interaction).
func TestReviveBySpellIneligibleChangesNothing(t *testing.T) {
	t.Parallel()
	cmd := rrCommand(t, combat.Resurrection)
	cmd.Circumstances = rrCircs(t, 400*rrDay, true, false, combat.Intact)
	before := rrCommand(t, combat.Resurrection)
	before.Circumstances = cmd.Circumstances
	res, ok := combat.ReviveBySpell(cmd).(combat.ReviveBySpellIneligible)
	if !ok || !slices.Equal(res.Reasons(), []combat.RevivalIneligibility{combat.DiedOfOldAge}) {
		t.Fatalf("ReviveBySpell = %#v, want ineligible for old age", res)
	}
	if !reflect.DeepEqual(cmd, before) {
		t.Error("an ineligible cast changed its command")
	}
	for _, spell := range []combat.RevivalSpell{0, -1, 9} {
		c := rrCommand(t, spell)
		if r, ok := combat.ReviveBySpell(c).(combat.ReviveBySpellIneligible); !ok || len(r.Reasons()) != 0 {
			t.Errorf("spell %d: ReviveBySpell = %#v, want ineligible with no reasons", int(spell), r)
		}
	}
}

// Requirement: per case (table 4.2; baseline: Dead, maximum 20, Poisoned and Blinded, effects c1 curse, c2 contagion, s1 spell, p1 poison)
// Path: per case.
func TestReviveBySpellOutcome(t *testing.T) {
	t.Parallel()
	blind := []content.ConditionKind{content.ConditionBlinded}
	both := []content.ConditionKind{content.ConditionBlinded, content.ConditionPoisoned}
	cases := []struct {
		name, requirement, path string
		spell                   combat.RevivalSpell
		edit                    func(*combat.ReviveBySpellCommand)
		current                 int
		conditions              []content.ConditionKind
		effects                 []string
	}{
		{"Revivify keeps everything", "FR-009, FR-011, FR-012", "happy", combat.Revivify, nil, 1, both, []string{"c1", "c2", "s1", "p1"}},
		{"Raise Dead neutralises poison", "FR-011, FR-012", "unhappy (rule interaction)", combat.RaiseDead, nil, 1, blind, []string{"c1", "c2", "s1"}},
		{"Resurrection", "FR-009, FR-011", "happy", combat.Resurrection, nil, 20, blind, []string{"c1", "c2", "s1"}},
		{"True Resurrection lifts all", "FR-012", "unhappy (rule interaction)", combat.TrueResurrection, nil, 20, blind, []string{"s1"}},
		{"poison condition not from poison", "FR-011", "unhappy (rule interaction)", combat.RaiseDead,
			func(c *combat.ReviveBySpellCommand) { c.Poisons.PoisonedConditionFromPoison = false }, 1, both, []string{"c1", "c2", "s1"}},
		{"host reports s1 elapsed", "FR-011 (the host's own sets are kept)", "unhappy (rule interaction)", combat.Revivify,
			func(c *combat.ReviveBySpellCommand) { c.ElapsedEffects = []combat.OngoingEffectID{meEID("s1")} }, 1, both, []string{"c1", "c2", "p1"}},
		{"host reports Blinded elapsed with the poison", "FR-011 (union)", "unhappy (rule interaction)", combat.RaiseDead,
			func(c *combat.ReviveBySpellCommand) { c.ElapsedConditions = blind }, 1, nil, []string{"c1", "c2", "s1"}},
		{"poison id not held", "FR-013", "unhappy (invalid input)", combat.RaiseDead,
			func(c *combat.ReviveBySpellCommand) {
				c.Poisons = combat.PoisonsAtDeath{Effects: []combat.OngoingEffectID{meEID("p9")}}
			}, 1, both, []string{"c1", "c2", "s1", "p1"}},
		{"nothing held", "FR-013", "unhappy (empty and minimal)", combat.TrueResurrection,
			func(c *combat.ReviveBySpellCommand) {
				c.Conditions, c.Effects, c.Poisons = combat.ConditionSet{}, combat.OngoingEffects{}, combat.PoisonsAtDeath{}
			},
			20, nil, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			cmd := rrCommand(t, c.spell)
			if c.edit != nil {
				c.edit(&cmd)
			}
			res := rrApplied(t, cmd).Revival
			st := res.Status()
			if _, alive := st.LifeState().(combat.Alive); !alive || st.HitPoints().Current() != c.current || st.HitPoints().Maximum() != 20 {
				t.Errorf("status = %+v / %T, want Alive at %d of 20 [%s]", st.HitPoints(), st.LifeState(), c.current, c.requirement)
			}
			got := rrConditionKinds(res.Conditions())
			slices.Sort(got)
			if len(got) != len(c.conditions) || (len(got) > 0 && !slices.Equal(got, c.conditions)) {
				t.Errorf("conditions = %v, want %v [%s]", got, c.conditions, c.requirement)
			}
			if effects := rrEffectIDs(res.Effects()); !slices.Equal(effects, c.effects) {
				t.Errorf("effects = %v, want %v [%s; %s]", effects, c.effects, c.requirement, c.path)
			}
			if kinds := rrKinds(rrApplied(t, cmd).Revival.Events()); kinds[0] != "creature_returned_to_life" {
				t.Errorf("events = %v, want the revival's events first", kinds)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-013 - a spell that neutralises or lifts something does not touch Exhaustion: a creature of
// Exhaustion 3 returns with level 2 under all four spells, as returning-to-life.md FR-011 decides (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (rule interaction).
func TestReviveBySpellExhaustionUntouched(t *testing.T) {
	t.Parallel()
	for _, spell := range []combat.RevivalSpell{combat.Revivify, combat.RaiseDead, combat.Resurrection, combat.TrueResurrection} {
		cmd := rrCommand(t, spell)
		cmd.Conditions = meConditions(combat.ExhaustionCondition{Level: meMust(combat.NewExhaustionLevel(3))}, combat.Poisoned{})
		res := rrApplied(t, cmd).Revival
		lvl := -1
		for _, c := range res.Conditions().Explicit() {
			if e, ok := c.(combat.ExhaustionCondition); ok {
				lvl = e.Level.Value()
			}
		}
		if lvl != 2 {
			t.Errorf("%v: Exhaustion level = %d, want 2", spell, lvl)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-009, FR-010 - a Hit Point maximum of 0 gives ReviveBySpellNotRevived wrapping
// ReviveCreatureMaximumBelowOne, whatever the spell; with RestoredMaximum 12 Resurrection and True Resurrection return the creature at 12
// of 12 and Revivify and Raise Dead at 1 of 12 (FR-049).
// Path: unhappy (boundary).
func TestReviveBySpellMaximumZero(t *testing.T) {
	t.Parallel()
	zero := status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
	twelve := 12
	for _, c := range []struct {
		spell   combat.RevivalSpell
		current int
	}{{combat.Revivify, 1}, {combat.RaiseDead, 1}, {combat.Resurrection, 12}, {combat.TrueResurrection, 12}} {
		cmd := rrCommand(t, c.spell)
		cmd.Status = zero
		nr, ok := combat.ReviveBySpell(cmd).(combat.ReviveBySpellNotRevived)
		if _, below := nr.Revival.(combat.ReviveCreatureMaximumBelowOne); !ok || !below || nr.Strain != combat.StrainNone {
			t.Errorf("%v without a restored maximum: %#v, want NotRevived (MaximumBelowOne)", c.spell, nr)
		}
		cmd.RestoredMaximum = &twelve
		st := rrApplied(t, cmd).Revival.Status()
		if _, alive := st.LifeState().(combat.Alive); !alive || st.HitPoints().Current() != c.current || st.HitPoints().Maximum() != 12 {
			t.Errorf("%v with 12: %+v / %T, want Alive at %d of 12", c.spell, st.HitPoints(), st.LifeState(), c.current)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-049 - a RestoredMaximum below 1 for a creature dead of a maximum of 0 gives
// ReviveBySpellInvalidMaximum carrying the value; the same value on any other status is ignored and the maximum stays.
// Path: unhappy (invalid input).
func TestReviveBySpellRestoredMaximumInvalidOrIgnored(t *testing.T) {
	t.Parallel()
	zeroStatus := status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
	for _, v := range []int{0, -5} {
		cmd := rrCommand(t, combat.Resurrection)
		cmd.Status, cmd.RestoredMaximum = zeroStatus, &v
		if r, ok := combat.ReviveBySpell(cmd).(combat.ReviveBySpellInvalidMaximum); !ok || r.RestoredMaximum != v {
			t.Errorf("RestoredMaximum %d: %#v, want InvalidMaximum carrying it", v, r)
		}
	}
	for _, v := range []int{0, 5} {
		cmd := rrCommand(t, combat.Resurrection)
		cmd.RestoredMaximum = &v
		if st := rrApplied(t, cmd).Revival.Status(); st.HitPoints().Maximum() != 20 || st.HitPoints().Current() != 20 {
			t.Errorf("RestoredMaximum %d on a maximum of 20: %+v, want the maximum to stay 20", v, st.HitPoints())
		}
	}
	// A creature dead of another cause with maximum 20 ignores it as well, Dying or Alive: not Dead means not revived.
	alive := rrCommand(t, combat.Resurrection)
	alive.Status, alive.RestoredMaximum = pcStatus(t, 5, 20, 0), new(int)
	if nr, ok := combat.ReviveBySpell(alive).(combat.ReviveBySpellNotRevived); !ok {
		t.Errorf("an Alive creature: %#v, want NotRevived", nr)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-010, FR-014 - the revival's own refusals come back unchanged inside
// ReviveBySpellNotRevived: a spirit that refuses (ReviveCreatureRefused, status unchanged) and a creature that is not Dead
// (ReviveCreatureNotDead); a refusal and a not-Dead creature leave the conditions and effects as passed (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (state error).
func TestReviveBySpellNotRevived(t *testing.T) {
	t.Parallel()
	refuses := rrCommand(t, combat.Revivify)
	refuses.Consent = combat.SpiritRefuses
	nr, ok := combat.ReviveBySpell(refuses).(combat.ReviveBySpellNotRevived)
	if _, isRefused := nr.Revival.(combat.ReviveCreatureRefused); !ok || !isRefused || !reflect.DeepEqual(nr.Revival.Status(), refuses.Status) ||
		!nr.Revival.Conditions().Equal(refuses.Conditions) || !nr.Revival.Effects().Equal(refuses.Effects) {
		t.Errorf("a refusing spirit: %#v, want NotRevived (Refused) with the state unchanged", nr)
	}
	notDead := rrCommand(t, combat.Revivify)
	notDead.Status = pcStatus(t, 3, 20, 0)
	nr, ok = combat.ReviveBySpell(notDead).(combat.ReviveBySpellNotRevived)
	if _, isNotDead := nr.Revival.(combat.ReviveCreatureNotDead); !ok || !isNotDead || nr.Strain != combat.StrainNone {
		t.Errorf("an Alive creature: %#v, want NotRevived (NotDead) with no strain", nr)
	}
	zeroID := rrCommand(t, combat.Revivify)
	zeroID.Subject = ""
	if nr, ok := combat.ReviveBySpell(zeroID).(combat.ReviveBySpellNotRevived); !ok {
		t.Errorf("a zero subject: %#v, want NotRevived", nr)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-015 - ReviveBySpell does not alter its inputs, returns copied slices (changing a
// returned slice or the slice passed in changes nothing in the result) and gives equal results for equal inputs.
// Path: unhappy (immutability).
func TestReviveBySpellImmutabilityAndDeterminism(t *testing.T) {
	t.Parallel()
	cmd := rrCommand(t, combat.TrueResurrection)
	cmd.ElapsedEffects = []combat.OngoingEffectID{meEID("s1")}
	cmd.ElapsedConditions = []content.ConditionKind{content.ConditionBlinded}
	before := rrCommand(t, combat.TrueResurrection)
	before.ElapsedEffects = slices.Clone(cmd.ElapsedEffects)
	before.ElapsedConditions = slices.Clone(cmd.ElapsedConditions)
	first := combat.ReviveBySpell(cmd)
	if !reflect.DeepEqual(cmd, before) {
		t.Error("ReviveBySpell changed its command")
	}
	if second := combat.ReviveBySpell(cmd); !reflect.DeepEqual(first, second) {
		t.Errorf("equal inputs gave %#v then %#v", first, second)
	}
	events := first.(combat.ReviveBySpellApplied).Revival.Events()
	events[0] = nil
	cmd.ElapsedEffects[0], cmd.Poisons.Effects[0] = meEID("zzz"), meEID("zzz")
	again := first.(combat.ReviveBySpellApplied).Revival.Events()
	if again[0] == nil || again[0].Kind() != "creature_returned_to_life" {
		t.Error("changing a returned event slice changed the result")
	}
	bad := rrCommand(t, combat.Revivify)
	bad.Circumstances = rrCircs(t, time.Hour, true, false, combat.Intact)
	r := combat.ReviveBySpell(bad).(combat.ReviveBySpellIneligible)
	reasons := r.Reasons()
	reasons[0] = combat.NameNotSpoken
	if r.Reasons()[0] != combat.TimeWindowExceeded {
		t.Error("changing a returned reasons slice changed the result")
	}
}
