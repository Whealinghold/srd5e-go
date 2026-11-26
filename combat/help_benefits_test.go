package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-035 to FR-039, FR-041, Table D - ability check help is given with a HelpGiven event or refused with the first applicable reason (CannotHelpSelf, HelperLacksProficiency, AssistanceNotPossible), the benefits unchanged.
// Path: per case.
func TestGiveAbilityCheckHelpTableD(t *testing.T) {
	t.Parallel()
	holds := proficienciesOf(t, []content.Skill{content.SkillAthletics}, []content.ItemID{content.MustItemID("thieves_tools")})
	none := proficienciesOf(t, nil, nil)
	cases := []struct {
		row, requirement, path string
		ally                   string
		prof                   combat.AssistedProficiency
		has                    combat.HelpGiverProficiencies
		judgement              combat.AssistanceJudgement
		refusal                combat.AbilityCheckHelpRefusal // 0 means given
	}{
		{"D01", "FR-035", "happy", "y", athletics(), holds, combat.AssistanceJudgementPossible, 0},
		{"D02", "FR-035", "happy", "y", thievesTools(), holds, combat.AssistanceJudgementPossible, 0},
		{"D03", "FR-036", "unhappy (invalid input)", "y", stealth(), holds, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalHelperLacksProficiency},
		{"D04", "FR-036", "unhappy (invalid input)", "y", combat.AssistedTool(content.MustItemID("herbalism_kit")), holds, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalHelperLacksProficiency},
		{"D05", "FR-039", "unhappy (state error)", "y", athletics(), holds, combat.AssistanceJudgementNotPossible, combat.AbilityCheckHelpRefusalAssistanceNotPossible},
		{"D06", "FR-037", "unhappy (invalid input)", "h", athletics(), holds, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalCannotHelpSelf},
		{"D07", "FR-038", "unhappy (rule interaction)", "y", stealth(), holds, combat.AssistanceJudgementNotPossible, combat.AbilityCheckHelpRefusalHelperLacksProficiency},
		{"D08", "FR-038", "unhappy (rule interaction)", "h", stealth(), holds, combat.AssistanceJudgementNotPossible, combat.AbilityCheckHelpRefusalCannotHelpSelf},
		{"D09", "FR-036", "unhappy (empty case)", "y", athletics(), none, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalHelperLacksProficiency},
		{"a tool is not the skill of the same name", "FR-036", "unhappy (invalid input)", "y", combat.AssistedTool(content.MustItemID("athletics")), holds, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalHelperLacksProficiency},
		{"zero judgement", "FR-039", "unhappy (invalid input)", "y", athletics(), holds, 0, combat.AbilityCheckHelpRefusalAssistanceNotPossible},
		{"invalid proficiency", "FR-036", "unhappy (invalid input)", "y", combat.AssistedProficiency{}, holds, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalHelperLacksProficiency},
		{"blank ally", "FR-037", "unhappy (invalid input)", "", athletics(), holds, combat.AssistanceJudgementPossible, combat.AbilityCheckHelpRefusalCannotHelpSelf},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			before := benefitsOf(t, rollHelp(t, "k", "e"))
			out := before.GiveAbilityCheckHelp(gid("h"), combat.CombatantID(c.ally), c.prof, c.has, c.judgement)
			if c.refusal != 0 {
				r, ok := out.(combat.AbilityCheckHelpRefused)
				if !ok || r.Reason() != c.refusal || !r.Benefits().Equal(before) || len(r.Events()) != 0 {
					t.Fatalf("%s: outcome = %#v, want refused %v [%s]", c.row, out, c.refusal, c.requirement)
				}
				return
			}
			g, ok := out.(combat.AbilityCheckHelpGiven)
			want := benefitsOf(t, rollHelp(t, "k", "e"), checkHelp(t, "h", c.ally, c.prof))
			if !ok || !g.Benefits().Equal(want) || !before.Equal(benefitsOf(t, rollHelp(t, "k", "e"))) {
				t.Fatalf("%s: outcome = %#v, want given with the benefit after the existing one [%s]", c.row, out, c.requirement)
			}
			ev := g.Events()
			if len(ev) != 1 {
				t.Fatalf("%s: %d events, want 1", c.row, len(ev))
			}
			combattest.CheckEvent(t, ev[0], "help_given", gid("h"), gid(c.ally))
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-040, FR-041, FR-075, Table E - attack roll help is given for an enemy within 5 feet and refused otherwise, TargetNotAnEnemy before EnemyNotWithinFiveFeet and when the enemy is the helper; it needs no proficiencies.
// Path: per case.
func TestGiveAttackRollHelpTableE(t *testing.T) {
	t.Parallel()
	const given = 0
	cases := []struct {
		row, requirement, path string
		enemy                  string
		target                 combat.AttackAssistTarget
		refusal                combat.AttackRollHelpRefusal
	}{
		{"E01", "FR-040", "happy", "e", combat.AttackAssistTargetEnemyWithinFiveFeet, given},
		{"E02", "FR-040", "unhappy (boundary)", "e", combat.AttackAssistTargetEnemyBeyondFiveFeet, combat.AttackRollHelpRefusalEnemyNotWithinFiveFeet},
		{"E03", "FR-040", "unhappy (invalid input)", "e", combat.AttackAssistTargetNotAnEnemy, combat.AttackRollHelpRefusalTargetNotAnEnemy},
		{"E05", "FR-075", "unhappy (invalid input)", "h", combat.AttackAssistTargetEnemyWithinFiveFeet, combat.AttackRollHelpRefusalTargetNotAnEnemy},
		{"E06", "FR-075", "unhappy (rule interaction)", "h", combat.AttackAssistTargetEnemyBeyondFiveFeet, combat.AttackRollHelpRefusalTargetNotAnEnemy},
		{"E07", "FR-075", "unhappy (rule interaction)", "h", combat.AttackAssistTargetNotAnEnemy, combat.AttackRollHelpRefusalTargetNotAnEnemy},
		{"zero target", "FR-040", "unhappy (invalid input)", "e", 0, combat.AttackRollHelpRefusalTargetNotAnEnemy},
		{"blank enemy", "FR-040", "unhappy (invalid input)", "", combat.AttackAssistTargetEnemyWithinFiveFeet, combat.AttackRollHelpRefusalTargetNotAnEnemy},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			before := benefitsOf(t, checkHelp(t, "k", "y", athletics()))
			out := before.GiveAttackRollHelp(gid("h"), combat.CombatantID(c.enemy), c.target)
			if c.refusal != given {
				r, ok := out.(combat.AttackRollHelpRefused)
				if !ok || r.Reason() != c.refusal || !r.Benefits().Equal(before) || len(r.Events()) != 0 {
					t.Fatalf("%s: outcome = %#v, want refused %v [%s]", c.row, out, c.refusal, c.requirement)
				}
				return
			}
			g, ok := out.(combat.AttackRollHelpGiven)
			if !ok || !g.Benefits().Equal(benefitsOf(t, checkHelp(t, "k", "y", athletics()), rollHelp(t, "h", "e"))) {
				t.Fatalf("%s: outcome = %#v [%s]", c.row, out, c.requirement)
			}
			ev := g.Events()
			if len(ev) != 1 {
				t.Fatalf("%s: %d events, want 1", c.row, len(ev))
			}
			combattest.CheckEvent(t, ev[0], "help_given", gid("h"), gid("e"))
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-042, FR-076 - giving Help leaves the receiver unchanged, returns a Benefits value that is a copy, a refusal carries an equal value, a new HelpBenefits starts empty, and the events are a copy.
// Path: unhappy (immutability).
func TestGivingHelpIsImmutable(t *testing.T) {
	t.Parallel()
	var empty combat.HelpBenefits
	if empty.Len() != 0 || len(empty.Benefits()) != 0 {
		t.Error("a new HelpBenefits is not empty")
	}
	holds := proficienciesOf(t, []content.Skill{content.SkillAthletics}, nil)
	g := empty.GiveAbilityCheckHelp(gid("h"), gid("y"), athletics(), holds, combat.AssistanceJudgementPossible).(combat.AbilityCheckHelpGiven)
	if empty.Len() != 0 || g.Benefits().Len() != 1 {
		t.Error("giving help changed the receiver, or the result is empty")
	}
	list := g.Benefits().Benefits()
	list[0] = nil
	if g.Benefits().Len() != 1 || g.Benefits().Benefits()[0] == nil {
		t.Error("Benefits() is not a copy")
	}
	ev := g.Events()
	ev[0] = nil
	if g.Events()[0] == nil {
		t.Error("Events() is not a copy")
	}
	if _, err := combat.NewHelpBenefits([]combat.HelpBenefit{nil}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil benefit: %v", err)
	}
	if _, err := combat.NewHelpBenefits([]combat.HelpBenefit{combat.AbilityCheckHelp{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero benefit: %v", err)
	}
	src := []combat.HelpBenefit{rollHelp(t, "h", "e")}
	h := benefitsOf(t, src...)
	src[0] = nil
	if h.Benefits()[0] == nil || h.Equal(empty) || !h.Equal(benefitsOf(t, rollHelp(t, "h", "e"))) || h.Equal(benefitsOf(t, rollHelp(t, "h", "f"))) {
		t.Error("NewHelpBenefits shares its argument, or Equal is wrong")
	}
}

// used asserts a consume result: the benefits left and the events' kinds.
func usedAbilityCheck(t *testing.T, out combat.AbilityCheckHelpUseOutcome) (combat.HelpBenefits, []combat.HelpEvent, []modifier.RollCircumstance, bool) {
	t.Helper()
	switch r := out.(type) {
	case combat.AbilityCheckHelpUsed:
		return r.Benefits(), r.Events(), r.Circumstances(), true
	case combat.NoAbilityCheckHelp:
		return r.Benefits(), r.Events(), r.Circumstances(), false
	}
	t.Fatalf("unknown outcome %#v", out)
	return combat.HelpBenefits{}, nil, nil, false
}

// Requirement: doc/specs/attack-and-help-actions.md FR-043 to FR-046, FR-052, Table F - a benefit applies only to the named ally and the same skill or tool, once; matching benefits are all used up and give one Advantage; other benefits are untouched.
// Path: per case.
func TestConsumeForAbilityCheckTableF(t *testing.T) {
	t.Parallel()
	cases := []struct {
		row, requirement, path string
		pending                []combat.HelpBenefit
		checker                string
		prof                   combat.AssistedProficiency
		used                   int // benefits consumed
		left                   []combat.HelpBenefit
	}{
		{"F01", "FR-043", "happy", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}, "y", athletics(), 1, nil},
		{"F02", "FR-043", "happy", []combat.HelpBenefit{checkHelp(t, "h", "y", thievesTools())}, "y", thievesTools(), 1, nil},
		{"F03", "FR-044", "unhappy (rule interaction)", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}, "y", stealth(), 0, []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}},
		{"F04", "FR-044", "unhappy (rule interaction)", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}, "z", athletics(), 0, []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}},
		{"F06", "FR-046", "unhappy (rule interaction)", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics()), checkHelp(t, "k", "y", athletics())}, "y", athletics(), 2, nil},
		{"F07", "FR-044", "unhappy (rule interaction)", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics()), checkHelp(t, "k", "y", stealth())}, "y", athletics(), 1, []combat.HelpBenefit{checkHelp(t, "k", "y", stealth())}},
		{"F08", "FR-043", "unhappy (empty case)", nil, "y", athletics(), 0, nil},
		{"F09", "FR-044", "unhappy (rule interaction)", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}, "y", thievesTools(), 0, []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}},
		{"an attack roll benefit is not ability check help", "FR-044", "unhappy (rule interaction)", []combat.HelpBenefit{rollHelp(t, "h", "y")}, "y", athletics(), 0, []combat.HelpBenefit{rollHelp(t, "h", "y")}},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			pending := benefitsOf(t, c.pending...)
			left, events, circ, usedAny := usedAbilityCheck(t, pending.ConsumeForAbilityCheck(gid(c.checker), c.prof))
			if usedAny != (c.used > 0) || len(events) != c.used || !left.Equal(benefitsOf(t, c.left...)) || !pending.Equal(benefitsOf(t, c.pending...)) {
				t.Fatalf("%s: used %v, %d events, left %v [%s]", c.row, usedAny, len(events), left.Benefits(), c.requirement)
			}
			if wantCirc := c.used > 0; wantCirc != (len(circ) == 1) || (!wantCirc && len(circ) != 0) {
				t.Errorf("%s: circumstances = %v", c.row, circ)
			}
			if len(circ) == 1 {
				if _, ok := circ[0].(modifier.Advantage); !ok {
					t.Errorf("%s: circumstance = %T, want Advantage", c.row, circ[0])
				}
			}
			for _, e := range events {
				combattest.CheckEvent(t, e, "help_benefit_used", e.Combatants()...)
				if len(e.Combatants()) != 2 || e.Combatants()[1] != gid(c.checker) {
					t.Errorf("%s: event names %v, want [helper, %s]", c.row, e.Combatants(), c.checker)
				}
			}
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-045, Table F row F05 - a benefit is used by one check only: a second matching check gets nothing.
// Path: unhappy (state error).
func TestAbilityCheckBenefitIsUsedOnce(t *testing.T) {
	t.Parallel()
	pending := benefitsOf(t, checkHelp(t, "h", "y", athletics()))
	first := pending.ConsumeForAbilityCheck(gid("y"), athletics()).(combat.AbilityCheckHelpUsed)
	if _, ok := first.Benefits().ConsumeForAbilityCheck(gid("y"), athletics()).(combat.NoAbilityCheckHelp); !ok {
		t.Error("the benefit was used twice [F05]")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-073, FR-043, Table F2 - a check that uses a skill and a tool together consumes every matching benefit, one Advantage and one event each in the order given; an empty or nil set matches nothing; one proficiency equals the set of one.
// Path: per case.
func TestConsumeForAbilityCheckUsingTableF2(t *testing.T) {
	t.Parallel()
	hy, ky := checkHelp(t, "h", "y", athletics()), checkHelp(t, "k", "y", thievesTools())
	cases := []struct {
		row     string
		pending []combat.HelpBenefit
		checker string
		profs   []combat.AssistedProficiency
		used    []string // helpers of the consumed benefits, in order
		left    []combat.HelpBenefit
	}{
		{"F10", []combat.HelpBenefit{hy, ky}, "y", []combat.AssistedProficiency{athletics(), thievesTools()}, []string{"h", "k"}, nil},
		{"F10 reversed request", []combat.HelpBenefit{hy, ky}, "y", []combat.AssistedProficiency{thievesTools(), athletics()}, []string{"h", "k"}, nil},
		{"F11", []combat.HelpBenefit{hy, ky}, "y", []combat.AssistedProficiency{athletics()}, []string{"h"}, []combat.HelpBenefit{ky}},
		{"F12", []combat.HelpBenefit{hy, ky}, "y", []combat.AssistedProficiency{thievesTools()}, []string{"k"}, []combat.HelpBenefit{hy}},
		{"F13", []combat.HelpBenefit{hy, checkHelp(t, "k", "y", stealth())}, "y", []combat.AssistedProficiency{athletics(), thievesTools()}, []string{"h"}, []combat.HelpBenefit{checkHelp(t, "k", "y", stealth())}},
		{"F14 empty", []combat.HelpBenefit{hy}, "y", []combat.AssistedProficiency{}, nil, []combat.HelpBenefit{hy}},
		{"F14 nil", []combat.HelpBenefit{hy}, "y", nil, nil, []combat.HelpBenefit{hy}},
		{"F15", []combat.HelpBenefit{hy, checkHelp(t, "k", "z", thievesTools())}, "y", []combat.AssistedProficiency{athletics(), thievesTools()}, []string{"h"}, []combat.HelpBenefit{checkHelp(t, "k", "z", thievesTools())}},
		{"duplicates are harmless", []combat.HelpBenefit{hy}, "y", []combat.AssistedProficiency{athletics(), athletics()}, []string{"h"}, nil},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			pending := benefitsOf(t, c.pending...)
			left, events, circ, usedAny := usedAbilityCheck(t, pending.ConsumeForAbilityCheckUsing(gid(c.checker), c.profs))
			if usedAny != (len(c.used) > 0) || !left.Equal(benefitsOf(t, c.left...)) || !pending.Equal(benefitsOf(t, c.pending...)) {
				t.Fatalf("%s: used %v, left %v [FR-073]", c.row, usedAny, left.Benefits())
			}
			var helpers []string
			for _, e := range events {
				helpers = append(helpers, string(e.Combatants()[0]))
			}
			if !slices.Equal(helpers, c.used) || (usedAny && len(circ) != 1) {
				t.Errorf("%s: events from %v, circumstances %d, want %v and one Advantage", c.row, helpers, len(circ), c.used)
			}
		})
	}
	one := benefitsOf(t, hy)
	a, _, _, _ := usedAbilityCheck(t, one.ConsumeForAbilityCheck(gid("y"), athletics()))
	b, _, _, _ := usedAbilityCheck(t, one.ConsumeForAbilityCheckUsing(gid("y"), []combat.AssistedProficiency{athletics()}))
	if !a.Equal(b) {
		t.Error("F16: the one-proficiency form differs from the set of one")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-076 - the circumstances and events of a used outcome are copies.
// Path: unhappy (immutability).
func TestUsedOutcomesReturnCopies(t *testing.T) {
	t.Parallel()
	used := benefitsOf(t, checkHelp(t, "h", "y", athletics())).ConsumeForAbilityCheck(gid("y"), athletics()).(combat.AbilityCheckHelpUsed)
	circ := used.Circumstances()
	circ[0] = modifier.Disadvantage{}
	ev := used.Events()
	ev[0] = nil
	if _, ok := used.Circumstances()[0].(modifier.Advantage); !ok || used.Events()[0] == nil {
		t.Error("a returned slice of AbilityCheckHelpUsed is not a copy")
	}
	roll := benefitsOf(t, rollHelp(t, "h", "e")).ConsumeForAttackRoll(gid("a"), gid("e"), alliedOf(t, "h")).(combat.AttackRollHelpUsed)
	circ = roll.Circumstances()
	circ[0] = modifier.Disadvantage{}
	ev = roll.Events()
	ev[0] = nil
	if _, ok := roll.Circumstances()[0].(modifier.Advantage); !ok || roll.Events()[0] == nil {
		t.Error("a returned slice of AttackRollHelpUsed is not a copy")
	}
	none := benefitsOf(t).ConsumeForAbilityCheck(gid("y"), athletics()).(combat.NoAbilityCheckHelp)
	noRoll := benefitsOf(t).ConsumeForAttackRoll(gid("a"), gid("e"), alliedOf(t)).(combat.NoAttackRollHelp)
	if none.Circumstances() == nil || none.Events() == nil || noRoll.Circumstances() == nil || noRoll.Events() == nil ||
		len(none.Circumstances())+len(none.Events())+len(noRoll.Circumstances())+len(noRoll.Events()) != 0 {
		t.Error("a No...Help variant must return empty non-nil slices")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-047 to FR-050, FR-052, Table G - a benefit applies only to the next attack roll against its enemy by an attacker whose ally set holds the helper, never to the helper, never to an attacker that is the target; all matching benefits are used up and give one Advantage.
// Path: per case.
func TestConsumeForAttackRollTableG(t *testing.T) {
	t.Parallel()
	he, ke := rollHelp(t, "h", "e"), rollHelp(t, "k", "e")
	cases := []struct {
		row, requirement, path string
		pending                []combat.HelpBenefit
		attacker, target       string
		allied                 []string
		used                   []string // helpers of the benefits consumed
		left                   []combat.HelpBenefit
	}{
		{"G01", "FR-047", "happy", []combat.HelpBenefit{he}, "a", "e", []string{"h"}, []string{"h"}, nil},
		{"G02", "FR-048", "unhappy (rule interaction)", []combat.HelpBenefit{he}, "a", "f", []string{"h"}, nil, []combat.HelpBenefit{he}},
		{"G03", "FR-048", "unhappy (rule interaction)", []combat.HelpBenefit{he}, "a", "e", nil, nil, []combat.HelpBenefit{he}},
		{"G04", "FR-049", "unhappy (rule interaction)", []combat.HelpBenefit{he}, "h", "e", []string{"h"}, nil, []combat.HelpBenefit{he}},
		{"G01 attacker sorts after the enemy and the helper", "FR-047", "happy", []combat.HelpBenefit{he}, "z", "e", []string{"h"}, []string{"h"}, nil},
		{"the benefit names an enemy that sorts after the target", "FR-048", "unhappy (rule interaction)", []combat.HelpBenefit{rollHelp(t, "h", "z")}, "a", "e", []string{"h"}, nil, []combat.HelpBenefit{rollHelp(t, "h", "z")}},
		{"G06", "FR-046", "unhappy (rule interaction)", []combat.HelpBenefit{he, ke}, "a", "e", []string{"h", "k"}, []string{"h", "k"}, nil},
		{"G07", "FR-048", "unhappy (rule interaction)", []combat.HelpBenefit{he, ke}, "a", "e", []string{"h"}, []string{"h"}, []combat.HelpBenefit{ke}},
		{"G08", "FR-047", "unhappy (empty case)", nil, "a", "e", []string{"h"}, nil, nil},
		{"G09", "FR-074", "unhappy (invalid input)", []combat.HelpBenefit{he}, "e", "e", []string{"h"}, nil, []combat.HelpBenefit{he}},
		{"an ability check benefit is not attack roll help", "FR-047", "unhappy (rule interaction)", []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}, "a", "y", []string{"h"}, nil, []combat.HelpBenefit{checkHelp(t, "h", "y", athletics())}},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			pending := benefitsOf(t, c.pending...)
			out := pending.ConsumeForAttackRoll(gid(c.attacker), gid(c.target), alliedOf(t, c.allied...))
			var left combat.HelpBenefits
			var events []combat.HelpEvent
			var circ []modifier.RollCircumstance
			switch r := out.(type) {
			case combat.AttackRollHelpUsed:
				left, events, circ = r.Benefits(), r.Events(), r.Circumstances()
			case combat.NoAttackRollHelp:
				left, events, circ = r.Benefits(), r.Events(), r.Circumstances()
			}
			var helpers []string
			for _, e := range events {
				helpers = append(helpers, string(e.Combatants()[0]))
				combattest.CheckEvent(t, e, "help_benefit_used", gid(string(e.Combatants()[0])), gid(c.attacker), gid(c.target))
			}
			if !slices.Equal(helpers, c.used) || !left.Equal(benefitsOf(t, c.left...)) || !pending.Equal(benefitsOf(t, c.pending...)) || (len(c.used) > 0) != (len(circ) == 1) {
				t.Errorf("%s: used %v, left %v, circumstances %v, want %v [%s]", c.row, helpers, left.Benefits(), circ, c.used, c.requirement)
			}
		})
	}
	// G05: the benefit is gone after one roll.
	first := benefitsOf(t, he).ConsumeForAttackRoll(gid("a"), gid("e"), alliedOf(t, "h")).(combat.AttackRollHelpUsed)
	if _, ok := first.Benefits().ConsumeForAttackRoll(gid("a"), gid("e"), alliedOf(t, "h")).(combat.NoAttackRollHelp); !ok {
		t.Error("the benefit was used twice [G05]")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-050 - the benefit is used up by the roll whether or not the caller later resolves Advantage, Disadvantage or neither: the result holds only the Advantage circumstance and resolves no roll mode, and the kernel cancels it with a Disadvantage the caller holds.
// Path: unhappy (rule interaction).
func TestHelpBenefitIsSpentEvenWhenDisadvantageCancelsIt(t *testing.T) {
	t.Parallel()
	used := benefitsOf(t, rollHelp(t, "h", "e")).ConsumeForAttackRoll(gid("a"), gid("e"), alliedOf(t, "h")).(combat.AttackRollHelpUsed)
	circ := used.Circumstances()
	if len(circ) != 1 {
		t.Fatalf("circumstances = %v, want only Advantage", circ)
	}
	if mode := modifier.Resolve(append(circ, modifier.Disadvantage{})...); mode != modifier.Normal {
		t.Errorf("mode = %v, want Normal (the caller resolves)", mode)
	}
	if used.Benefits().Len() != 0 {
		t.Error("the benefit survived a roll that ended with no Advantage")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-053 to FR-056, Table H - a benefit expires at the start of its helper's next turn, one HelpBenefitExpired each in the order given; another combatant's turn, the ally's included, expires nothing; an empty or used-up set expires nothing; the receiver is unchanged.
// Path: per case.
func TestExpireAtStartOfTurnTableH(t *testing.T) {
	t.Parallel()
	hy, he, ke := checkHelp(t, "h", "y", athletics()), rollHelp(t, "h", "e"), rollHelp(t, "k", "e")
	cases := []struct {
		row, requirement, path string
		pending                []combat.HelpBenefit
		turn                   string
		expired                int
		left                   []combat.HelpBenefit
	}{
		{"H01", "FR-053", "happy", []combat.HelpBenefit{hy, he}, "h", 2, nil},
		{"H02", "FR-053", "unhappy (rule interaction)", []combat.HelpBenefit{hy, ke}, "h", 1, []combat.HelpBenefit{ke}},
		{"H03", "FR-054", "unhappy (boundary)", []combat.HelpBenefit{hy}, "y", 0, []combat.HelpBenefit{hy}},
		{"H04", "FR-054", "unhappy (boundary)", []combat.HelpBenefit{hy}, "k", 0, []combat.HelpBenefit{hy}},
		{"H05", "FR-055", "unhappy (empty case)", nil, "h", 0, nil},
		{"enemy's turn", "FR-054", "unhappy (boundary)", []combat.HelpBenefit{he}, "e", 0, []combat.HelpBenefit{he}},
		{"blank turn", "FR-054", "unhappy (invalid input)", []combat.HelpBenefit{hy}, "", 0, []combat.HelpBenefit{hy}},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			pending := benefitsOf(t, c.pending...)
			res := pending.ExpireAtStartOfTurn(combat.CombatantID(c.turn))
			if got := helpKinds(res.Events()); len(got) != c.expired || !res.Benefits().Equal(benefitsOf(t, c.left...)) || !pending.Equal(benefitsOf(t, c.pending...)) {
				t.Fatalf("%s: events %v, left %v [%s]", c.row, got, res.Benefits().Benefits(), c.requirement)
			}
			for _, k := range helpKinds(res.Events()) {
				if k != "help_benefit_expired" {
					t.Errorf("%s: kind %q", c.row, k)
				}
			}
			if c.expired == 0 && !res.Benefits().Equal(pending) {
				t.Errorf("%s: benefits must equal the receiver when nothing expires [FR-056]", c.row)
			}
			if !res.Equal(pending.ExpireAtStartOfTurn(combat.CombatantID(c.turn))) {
				t.Errorf("%s: the result must equal itself by value", c.row)
			}
		})
	}
	// H06: a benefit used earlier produces no expiry later.
	used := benefitsOf(t, hy).ConsumeForAbilityCheck(gid("y"), athletics()).(combat.AbilityCheckHelpUsed)
	if res := used.Benefits().ExpireAtStartOfTurn(gid("h")); len(res.Events()) != 0 || res.Benefits().Len() != 0 {
		t.Error("a used benefit expired [H06]")
	}
	// order given, and the events are a copy
	res := benefitsOf(t, hy, he).ExpireAtStartOfTurn(gid("h"))
	if got := res.Events(); got[0].Combatants()[1] != gid("y") || got[1].Combatants()[1] != gid("e") {
		t.Errorf("events not in the order given: %v", got)
	}
	ev := res.Events()
	ev[0] = nil
	if res.Events()[0] == nil {
		t.Error("Events() is not a copy")
	}
	if res.Equal(benefitsOf(t, hy, he).ExpireAtStartOfTurn(gid("k"))) {
		t.Error("different results must not be equal")
	}
	var none combat.HelpBenefits
	sameBenefitsOtherEvents := benefitsOf(t, hy).ExpireAtStartOfTurn(gid("h")) // no benefits left, one event
	if sameBenefitsOtherEvents.Equal(none.ExpireAtStartOfTurn(gid("h"))) || none.ExpireAtStartOfTurn(gid("h")).Equal(sameBenefitsOtherEvents) {
		t.Error("results with equal benefits and different events must not be equal")
	}
	sameEventsOtherBenefits := benefitsOf(t, hy).ExpireAtStartOfTurn(gid("k")) // no events, one benefit left
	if sameEventsOtherBenefits.Equal(none.ExpireAtStartOfTurn(gid("k"))) {
		t.Error("results with equal events and different benefits must not be equal")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-069, Table H2 - DropFor removes every benefit naming the combatant as helper, ally or enemy and DropAll removes all, one HelpBenefitDropped each in the order given; nothing removed returns an equal value and no events.
// Path: per case.
func TestDropTableH2(t *testing.T) {
	t.Parallel()
	hy, he, ke := checkHelp(t, "h", "y", athletics()), rollHelp(t, "h", "e"), rollHelp(t, "k", "e")
	cases := []struct {
		row, requirement, path string
		pending                []combat.HelpBenefit
		who                    string // "" means DropAll
		dropped                int
		left                   []combat.HelpBenefit
	}{
		{"H07", "FR-069", "happy", []combat.HelpBenefit{hy, ke}, "e", 1, []combat.HelpBenefit{hy}},
		{"H08", "FR-069", "unhappy (rule interaction)", []combat.HelpBenefit{hy, ke}, "h", 1, []combat.HelpBenefit{ke}},
		{"H09", "FR-069", "happy", []combat.HelpBenefit{hy, ke}, "", 2, nil},
		{"H10", "FR-069", "unhappy (empty case)", nil, "", 0, nil},
		{"H11", "FR-069", "unhappy (boundary)", []combat.HelpBenefit{hy}, "z", 0, []combat.HelpBenefit{hy}},
		{"H12", "FR-069", "unhappy (rule interaction)", []combat.HelpBenefit{hy, he}, "y", 1, []combat.HelpBenefit{he}},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			pending := benefitsOf(t, c.pending...)
			var res combat.HelpBenefitsDrop
			if c.who == "" {
				res = pending.DropAll()
			} else {
				res = pending.DropFor(gid(c.who))
			}
			if got := helpKinds(res.Events()); len(got) != c.dropped || !res.Benefits().Equal(benefitsOf(t, c.left...)) || !pending.Equal(benefitsOf(t, c.pending...)) {
				t.Fatalf("%s: events %v, left %v [%s]", c.row, got, res.Benefits().Benefits(), c.requirement)
			}
			for _, k := range helpKinds(res.Events()) {
				if k != "help_benefit_dropped" {
					t.Errorf("%s: kind %q", c.row, k)
				}
			}
			if c.dropped == 0 && !res.Benefits().Equal(pending) {
				t.Errorf("%s: benefits must equal the receiver when nothing is removed", c.row)
			}
		})
	}
	res := benefitsOf(t, hy, ke).DropAll()
	if got := res.Events(); got[0].Combatants()[0] != gid("h") || got[1].Combatants()[0] != gid("k") {
		t.Errorf("DropAll events not in the order given: %v", got)
	}
	ev := res.Events()
	ev[0] = nil
	if res.Events()[0] == nil {
		t.Error("Events() is not a copy")
	}
	if !res.Equal(benefitsOf(t, hy, ke).DropAll()) || res.Equal(benefitsOf(t, hy).DropAll()) || res.Equal(benefitsOf(t, hy, ke).DropFor(gid("h"))) {
		t.Error("HelpBenefitsDrop equality")
	}
	var zero combat.HelpBenefitsDrop
	var zeroExpiry combat.HelpBenefitsExpiry
	if len(zero.Events()) != 0 || zero.Benefits().Len() != 0 || len(zeroExpiry.Events()) != 0 || !zeroExpiry.Equal(combat.HelpBenefitsExpiry{}) {
		t.Error("the zero results must be empty")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-061 - the same call sequence from the same HelpBenefits gives the same result on every run.
// Path: unhappy (determinism).
func TestHelpIsDeterministic(t *testing.T) {
	t.Parallel()
	holds := proficienciesOf(t, []content.Skill{content.SkillAthletics}, nil)
	run := func() combat.HelpBenefits {
		var h combat.HelpBenefits
		h = h.GiveAbilityCheckHelp(gid("h"), gid("y"), athletics(), holds, combat.AssistanceJudgementPossible).(combat.AbilityCheckHelpGiven).Benefits()
		h = h.GiveAttackRollHelp(gid("k"), gid("e"), combat.AttackAssistTargetEnemyWithinFiveFeet).(combat.AttackRollHelpGiven).Benefits()
		return h.ConsumeForAbilityCheck(gid("y"), athletics()).(combat.AbilityCheckHelpUsed).Benefits()
	}
	if !run().Equal(run()) {
		t.Error("two runs of the same call sequence differ")
	}
}
