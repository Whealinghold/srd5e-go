package combat_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// maStep is one step of a Multiattack: op is "attack", "extra" or "replace"; part is the part attacked with, used as the
// additional use or, for a replacement, the option's part (option is then the option and replaced the attack part replaced);
// want is "" for a recorded step, "usage" for a usage refusal or the String of the allowance refusal.
type maStep struct {
	op, part, replaced, want string
	option                   content.ReplacementOption
}

func atk(part string) maStep           { return maStep{op: "attack", part: part} }
func atkWant(part, want string) maStep { return maStep{op: "attack", part: part, want: want} }
func extra(part, want string) maStep   { return maStep{op: "extra", part: part, want: want} }
func repl(o content.ReplacementOption, replaced, want string) maStep {
	return maStep{op: "replace", part: o.Part(), option: o, replaced: replaced, want: want}
}

// maRun starts the Multiattack of m (with opts) and plays the steps, checking each outcome and that a refusal changes nothing.
// It returns the final progress and limited-use state.
func maRun(t *testing.T, m content.Monster, state combat.LimitedUseState, steps []maStep, opts ...combat.MultiattackOption) (combat.MultiattackProgress, combat.LimitedUseState) {
	t.Helper()
	started, err := combat.StartMultiattack(m, sbSelf(), state, opts...)
	s, ok := started.(combat.MultiattackStarted)
	if err != nil || !ok {
		t.Fatalf("StartMultiattack = %#v, %v, want MultiattackStarted", started, err)
	}
	p := s.Progress()
	for i, st := range steps {
		var out combat.MultiattackStepOutcome
		switch st.op {
		case "attack":
			out, err = p.RecordAttack(sbAction(t, m, st.part), state)
		case "extra":
			out, err = p.RecordAdditionalUse(sbAction(t, m, st.part), state)
		default:
			out, err = p.RecordReplacement(st.option, sbAction(t, m, st.replaced), state)
		}
		if err != nil {
			t.Fatalf("step %d (%s %s): error %v", i, st.op, st.part, err)
		}
		switch o := out.(type) {
		case combat.MultiattackStepRecorded:
			if st.want != "" {
				t.Fatalf("step %d (%s %s) was recorded, want %q", i, st.op, st.part, st.want)
			}
			p, state = o.Transition.Progress(), o.Transition.LimitedUse()
		case combat.MultiattackNotInAllowance:
			if st.want != o.Reason.String() {
				t.Fatalf("step %d (%s %s) refused as %v, want %q", i, st.op, st.part, o.Reason, st.want)
			}
		case combat.MultiattackStepUsageRefused:
			if st.want != "usage" {
				t.Fatalf("step %d (%s %s) refused for usage, want %q", i, st.op, st.part, st.want)
			}
		default:
			t.Fatalf("step %d: outcome %#v", i, out)
		}
	}
	return p, state
}

// The plans of the cited blocks (SRD 5.2.1, pp. 258-296), made of synthetic parts.
func maAboleth(t *testing.T) content.Monster {
	t.Helper()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Tentacle")), sbExtras(sbExtra("Consume Memories", "Dominate Mind"))))
	return sbMonster(t, &plan, []content.StatBlockPart{sbPart("Tentacle", content.StatBlockPartFields{}),
		sbPart("Consume Memories", content.StatBlockPartFields{}), sbPart("Dominate Mind", content.StatBlockPartFields{Usage: luPerDay(2, 0)})}, nil)
}

func maSimple(t *testing.T, variants []content.MultiattackVariant, names ...string) content.Monster {
	t.Helper()
	plan := sbMultiattack(variants...)
	return sbMonster(t, &plan, sbPlain(names...), nil)
}

func maAssassin(t *testing.T) content.Monster {
	t.Helper()
	return maSimple(t, []content.MultiattackVariant{sbVariant("", sbGroups(sbGroup(3, "Shortsword", "Light Crossbow")), nil)}, "Shortsword", "Light Crossbow")
}

func maBalor(t *testing.T) content.Monster {
	t.Helper()
	return maSimple(t, []content.MultiattackVariant{sbVariant("", sbGroups(sbGroup(1, "Flame Whip"), sbGroup(1, "Lightning Blade")), nil)}, "Flame Whip", "Lightning Blade")
}

func maDevil(t *testing.T) content.Monster {
	t.Helper()
	return maSimple(t, []content.MultiattackVariant{
		sbVariant("", sbGroups(sbGroup(1, "Claws"), sbGroup(1, "Tail")), nil),
		sbVariant("", sbGroups(sbGroup(2, "Hurl Flame")), nil)}, "Claws", "Tail", "Hurl Flame")
}

func maGolem(t *testing.T) content.Monster {
	t.Helper()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Slam")), nil), sbVariant("Hasten", sbGroups(sbGroup(3, "Slam")), nil))
	return sbMonster(t, &plan, sbPlain("Slam", "Bash"), func(f *content.MonsterFields) {
		f.BonusActions = sbPlain("Hasten")
		f.Reactions = sbPlain("Aaa")
	})
}

func maGladiator(t *testing.T, shieldBash content.StatBlockPart) content.Monster {
	t.Helper()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(3, "Spear")), nil, sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbOption("Shield Bash"))))
	return sbMonster(t, &plan, []content.StatBlockPart{sbPart("Spear", content.StatBlockPartFields{}), shieldBash}, nil)
}

func maPlainBash() content.StatBlockPart { return sbPart("Shield Bash", content.StatBlockPartFields{}) }

func maChimera(t *testing.T) content.Monster {
	t.Helper()
	return maSimple(t, []content.MultiattackVariant{sbVariant("", sbGroups(sbGroup(1, "Bite"), sbGroup(1, "Claw")), nil,
		sbRule(1, sbOnlyPart("Claw"), content.ReplaceOnce, sbOption("Fire Breath")))}, "Bite", "Claw", "Fire Breath")
}

func maNaga(t *testing.T) content.Monster {
	t.Helper()
	return maSimple(t, []content.MultiattackVariant{sbVariant("", sbGroups(sbGroup(2, "Bite")), nil,
		sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, sbOption("Poisonous Spittle")))}, "Bite", "Poisonous Spittle")
}

func maBrass(t *testing.T) content.Monster {
	t.Helper()
	return maSimple(t, []content.MultiattackVariant{sbVariant("", sbGroups(sbGroup(3, "Rend")), nil,
		sbRule(2, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbOption("Sleep Breath")))}, "Rend", "Sleep Breath")
}

func maAdultBrass(t *testing.T, spellUses content.SpellFrequency) content.Monster {
	t.Helper()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(3, "Rend")), nil,
		sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbOption("Sleep Breath"), sbSpellOption("Spellcasting", "Scorching Ray", 3))))
	casting := luSpellcasting(luSpells(spellUses, "Scorching Ray"))
	return sbMonster(t, &plan, append(sbPlain("Rend", "Sleep Breath", "Fire Breath"), casting), nil)
}

// Requirement: doc/specs/monster-stat-block-actions.md table MA-1 (FR-010 to FR-020) - one test per row of the cited plans.
// Path: per case.
func TestMultiattackTableMA1(t *testing.T) {
	t.Parallel()
	slam := func(n int) []maStep {
		out := make([]maStep, n)
		for i := range out {
			out[i] = atk("Slam")
		}
		return out
	}
	bash := sbOption("Shield Bash")
	cases := []struct {
		name, requirement, path string
		monster                 func(*testing.T) content.Monster
		opts                    []combat.MultiattackOption
		steps                   []maStep
	}{
		{"two Tentacle slots", `MA-1; "Aboleth", p. 258`, "happy", maAboleth, nil, []maStep{atk("Tentacle"), atk("Tentacle")}},
		{"third Tentacle", `MA-1; "Aboleth", p. 258`, "unhappy (boundary)", maAboleth, nil, []maStep{atk("Tentacle"), atk("Tentacle"), atkWant("Tentacle", "allowance_spent")}},
		{"additional use is free", `MA-1; "Aboleth", p. 258`, "happy", maAboleth, nil, []maStep{atk("Tentacle"), atk("Tentacle"), extra("Dominate Mind", ""), atkWant("Tentacle", "allowance_spent")}},
		{"either, not both", `MA-1; "Aboleth", p. 258`, "unhappy (rule interaction)", maAboleth, nil, []maStep{extra("Dominate Mind", ""), extra("Consume Memories", "additional_use_already_made"), extra("Dominate Mind", "additional_use_already_made")}},
		{"additional use not listed", `FR-016; "Aboleth", p. 258`, "unhappy (boundary)", maAboleth, nil, []maStep{extra("Tentacle", "additional_use_not_listed")}},
		{"an attack not in the plan", `FR-013; "Aboleth", p. 258`, "unhappy (invalid input)", maAboleth, nil, []maStep{atkWant("Dominate Mind", "attack_not_allowed")}},
		{"any combination", `MA-1; "Assassin", p. 260`, "unhappy (boundary)", maAssassin, nil, []maStep{atk("Shortsword"), atk("Light Crossbow"), atk("Shortsword"), atkWant("Light Crossbow", "allowance_spent")}},
		{"one part three times", `FR-014; "Assassin", p. 260`, "happy", maAssassin, nil, []maStep{atk("Light Crossbow"), atk("Light Crossbow"), atk("Light Crossbow")}},
		{"order free", `MA-1; "Balor", p. 261`, "happy", maBalor, nil, []maStep{atk("Lightning Blade"), atk("Flame Whip")}},
		{"one each", `MA-1; "Balor", p. 261`, "unhappy (boundary)", maBalor, nil, []maStep{atk("Flame Whip"), atkWant("Flame Whip", "allowance_spent")}},
		{"alternative A", `MA-1; "Barbed Devil", p. 262`, "happy", maDevil, nil, []maStep{atk("Claws"), atk("Tail"), atkWant("Hurl Flame", "attack_not_allowed")}},
		{"alternatives mixed", `MA-1; "Barbed Devil", p. 262`, "unhappy (rule interaction)", maDevil, nil, []maStep{atk("Claws"), atkWant("Hurl Flame", "attack_not_allowed")}},
		{"alternative B", `MA-1; "Barbed Devil", p. 262`, "unhappy (boundary)", maDevil, nil, []maStep{atk("Hurl Flame"), atk("Hurl Flame"), atkWant("Hurl Flame", "allowance_spent"), atkWant("Claws", "attack_not_allowed")}},
		{"conditional, not used", `MA-1; "Clay Golem", p. 274`, "unhappy (rule interaction)", maGolem, nil, append(slam(2), atkWant("Slam", "allowance_spent"))},
		{"conditional, used", `MA-1; "Clay Golem", p. 274`, "happy", maGolem, []combat.MultiattackOption{combat.WithUsedThisTurn([]combat.LimitedUsePartRef{luRef(t, combat.MonsterPartBonusActions, 0)})}, slam(3)},
		{"conditional, used, still a ceiling", `MA-1; "Clay Golem", p. 274`, "unhappy (boundary)", maGolem, []combat.MultiattackOption{combat.WithUsedThisTurn([]combat.LimitedUsePartRef{luRef(t, combat.MonsterPartBonusActions, 0)})}, append(slam(3), atkWant("Slam", "allowance_spent"))},
		{"replacement", `MA-1; "Gladiator", p. 289`, "happy", maGladiatorPlain, nil, []maStep{atk("Spear"), repl(bash, "Spear", ""), atk("Spear"), atkWant("Spear", "allowance_spent")}},
		{"replacement once", `MA-1; "Gladiator", p. 289`, "unhappy (boundary)", maGladiatorPlain, nil, []maStep{repl(bash, "Spear", ""), repl(bash, "Spear", "replacement_limit_reached")}},
		{"replacement first", `MA-1; "Gladiator", p. 289`, "unhappy (rule interaction)", maGladiatorPlain, nil, []maStep{repl(bash, "Spear", ""), atk("Spear"), atk("Spear"), atkWant("Spear", "allowance_spent")}},
		{"replacement scope", `MA-1; "Chimera", p. 273`, "unhappy (rule interaction)", maChimera, nil, []maStep{repl(sbOption("Fire Breath"), "Bite", "replacement_not_listed"), repl(sbOption("Fire Breath"), "Claw", ""), atk("Bite"), atkWant("Claw", "allowance_spent")}},
		{"replacement option not listed", `FR-018; "Chimera", p. 273`, "unhappy (invalid input)", maChimera, nil, []maStep{repl(sbOption("Bite"), "Claw", "replacement_not_listed")}},
		{"any attack, no cap", `MA-1; "Guardian Naga", p. 296`, "happy", maNaga, nil, []maStep{repl(sbOption("Poisonous Spittle"), "Bite", ""), repl(sbOption("Poisonous Spittle"), "Bite", ""), atkWant("Bite", "allowance_spent"), repl(sbOption("Poisonous Spittle"), "Bite", "not_enough_attacks_to_replace")}},
		{"replace two", `MA-1; brass dragons, p. 268`, "unhappy (boundary)", maBrass, nil, []maStep{repl(sbOption("Sleep Breath"), "Rend", ""), atk("Rend"), atkWant("Rend", "allowance_spent")}},
		{"not enough", `MA-1; brass dragons, p. 268`, "unhappy (boundary)", maBrass, nil, []maStep{atk("Rend"), atk("Rend"), repl(sbOption("Sleep Breath"), "Rend", "not_enough_attacks_to_replace")}},
		{"two slots left is enough", `MA-1; brass dragons, p. 268`, "unhappy (boundary)", maBrass, nil, []maStep{atk("Rend"), repl(sbOption("Sleep Breath"), "Rend", ""), atkWant("Rend", "allowance_spent")}},
		{"limit before slots", `FR-018`, "unhappy (rule interaction)", maBrass, nil, []maStep{repl(sbOption("Sleep Breath"), "Rend", ""), repl(sbOption("Sleep Breath"), "Rend", "replacement_limit_reached")}},
		{"option with a spell", `MA-1; "Adult Brass Dragon", p. 269`, "happy", maAdultBrassTwice, nil, []maStep{repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", ""), atk("Rend")}},
		{"option not listed", `MA-1; "Adult Brass Dragon", p. 269`, "unhappy (invalid input)", maAdultBrassTwice, nil, []maStep{repl(sbOption("Fire Breath"), "Rend", "replacement_not_listed")}},
		{"spell option with another level is another option", `FR-018`, "unhappy (invalid input)", maAdultBrassTwice, nil, []maStep{repl(sbSpellOption("Spellcasting", "Scorching Ray", 4), "Rend", "replacement_not_listed")}},
		{"an At Will spell has nothing to spend", `FR-018, FR-007`, "unhappy (boundary)", maAdultBrassAtWill, nil, []maStep{repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", ""), repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", "replacement_limit_reached")}},
		{"a once-limited spell option twice", `FR-018`, "unhappy (boundary)", maAdultBrassOnce, nil, []maStep{repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", ""), repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", "replacement_limit_reached")}},
		{"allowance before usage", `FR-020`, "unhappy (rule interaction)", func(t *testing.T) content.Monster {
			t.Helper()
			plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Claw")), nil))
			return sbMonster(t, &plan, []content.StatBlockPart{sbPart("Claw", content.StatBlockPartFields{Usage: luRecharge(5, 6)})}, nil)
		}, nil, []maStep{atk("Claw"), atkWant("Claw", "allowance_spent")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m := c.monster(t)
			maRun(t, m, luStart(t, m, combat.OutsideLair), c.steps, c.opts...)
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md table MA-1 "usage refused" and FR-020 - a step the plan allows but whose part is
// spent returns MultiattackStepUsageRefused with the progress and the limited-use state unchanged; the spent part is the replacement
// option, the attack or the additional use.
// Path: per case.
func TestMultiattackUsageRefusalChangesNothing(t *testing.T) {
	t.Parallel()
	bash := sbOption("Shield Bash")
	recharge := sbPart("Shield Bash", content.StatBlockPartFields{Usage: luRecharge(5, 6)})
	clawPlan := sbMultiattack(sbVariant("", sbGroups(sbGroup(2, "Claw")), nil))
	clawMonster := sbMonster(t, &clawPlan, []content.StatBlockPart{sbPart("Claw", content.StatBlockPartFields{Usage: luRecharge(5, 6)})}, nil)
	cases := []struct {
		name string
		m    content.Monster
		step maStep
	}{
		{"replacement option spent", maGladiator(t, recharge), repl(bash, "Spear", "usage")},
		{"additional use spent", maAboleth(t), extra("Dominate Mind", "usage")},
		{"replacement spell spent", maAdultBrass(t, luPerDaySpells(1)), repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", "usage")},
		{"attack part spent", clawMonster, atkWant("Claw", "usage")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			state := luStart(t, c.m, combat.OutsideLair)
			spent := maSpendAll(t, c.m, state, c.step)
			p, got := maRun(t, c.m, spent, []maStep{c.step})
			if !got.Equal(spent) || len(p.Attacks()) != 0 {
				t.Error("a usage refusal changed the limited-use state or the progress")
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-010 - StartMultiattack returns MultiattackStarted for a monster whose Multiattack
// part carries a plan, the transition consuming the action like any action (a FeatureActionChoice naming the Multiattack part, the
// granted action given or NoStandardAction), or MultiattackNotAvailable for a monster with no Multiattack part or no plan (SRD 5.2.1,
// "Multiattack", p. 257).
// Path: per case.
func TestStartMultiattack(t *testing.T) {
	t.Parallel()
	m := maAboleth(t)
	state := luStart(t, m, combat.OutsideLair)
	out, err := combat.StartMultiattack(m, sbSelf(), state, combat.WithGrantedAction(combat.GrantsSimpleAction{Action: combat.SimpleStandardAction(1)}))
	started, ok := out.(combat.MultiattackStarted)
	if err != nil || !ok {
		t.Fatalf("StartMultiattack = %#v, %v", out, err)
	}
	choice, isAction := started.Transition().ActionChoice()
	if !isAction || choice.Feature != content.MustFeatureID("fixture_beast__multiattack") || choice.Granted == (combat.NoStandardAction{}) {
		t.Errorf("ActionChoice() = %+v, %v, want the Multiattack feature with the granted action", choice, isAction)
	}
	if events := started.Transition().Events(); len(events) != 1 || events[0].Kind() != "stat_block_part_used" {
		t.Errorf("events = %v, want the part used", eventKinds(events))
	}
	if !started.Transition().LimitedUse().Equal(state) {
		t.Error("starting a Multiattack changed the limited-use state")
	}
	if p := started.Progress(); p.Part() != luRef(t, combat.MonsterPartActions, 0) || p.Combatants()[0] != sbSelf() || len(p.Attacks()) != 0 || p.IsZero() {
		t.Errorf("progress = part %v combatants %v attacks %v, want the Multiattack at 0, the monster and none", p.Part(), p.Combatants(), p.Attacks())
	}
	def, _ := combat.StartMultiattack(m, sbSelf(), state)
	if c, _ := def.(combat.MultiattackStarted).Transition().ActionChoice(); c.Granted != combat.GrantedAction(combat.NoStandardAction{}) {
		t.Errorf("default granted = %#v, want NoStandardAction", c.Granted)
	}
	noPlan := sbMonster(t, nil, sbPlain(content.MultiattackName, "Slam"), nil)
	for name, mm := range map[string]content.Monster{"no Multiattack part": luMonster(t, nil), "no plan": noPlan} {
		got, err := combat.StartMultiattack(mm, sbSelf(), luStart(t, mm, combat.OutsideLair))
		if err != nil || got != (combat.MultiattackNotAvailable{}) {
			t.Errorf("%s: StartMultiattack = %#v, %v, want MultiattackNotAvailable", name, got, err)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-010, FR-049 - each Multiattack starts its own progress; counts never carry
// between turns.
// Path: unhappy (state error).
func TestEachMultiattackStartsItsOwnProgress(t *testing.T) {
	t.Parallel()
	m := maBalor(t)
	state := luStart(t, m, combat.OutsideLair)
	first, _ := maRun(t, m, state, []maStep{atk("Flame Whip")})
	second, _ := maRun(t, m, state, nil)
	if len(first.Attacks()) != 1 || len(second.Attacks()) != 0 || second.AttacksRemaining(sbAction(t, m, "Flame Whip")) != 1 {
		t.Errorf("attacks %v then %v, remaining %d, want a fresh progress", first.Attacks(), second.Attacks(), second.AttacksRemaining(sbAction(t, m, "Flame Whip")))
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-010, FR-049 - StartMultiattack and the steps reject untrusted input with an
// error matching srderr.ErrInvalidInput: a zero or foreign monster, combatant or state, an invalid granted action, a zero or repeated
// used part, a Multiattack with a usage limit, and a plan name that does not resolve.
// Path: per case.
func TestStartMultiattackRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	m := maAboleth(t)
	state := luStart(t, m, combat.OutsideLair)
	ref := luRef(t, combat.MonsterPartActions, 1)
	limitedPlan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Slam")), nil))
	limitedPlanPart := sbPart(content.MultiattackName, content.StatBlockPartFields{Usage: luPerDay(1, 0), Plan: mustPlanOf(t, limitedPlan)})
	limited := sbMonster(t, &limitedPlanPart, sbPlain("Slam"), nil)
	spellless := func() content.Monster {
		plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Rend")), nil,
			sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbSpellOption("Spellcasting", "Scorching Ray", 3))))
		return sbMonster(t, &plan, sbPlain("Rend", "Spellcasting"), nil)
	}()
	cases := []struct {
		name  string
		m     content.Monster
		who   combat.CombatantID
		state combat.LimitedUseState
		opts  []combat.MultiattackOption
	}{
		{"zero monster", content.Monster{}, sbSelf(), state, nil},
		{"zero combatant", m, "", state, nil},
		{"zero state", m, sbSelf(), combat.LimitedUseState{}, nil},
		{"state of another combatant", m, combat.MustCombatantID("other"), state, nil},
		{"nil granted action", m, sbSelf(), state, []combat.MultiattackOption{combat.WithGrantedAction(nil)}},
		{"invalid granted action", m, sbSelf(), state, []combat.MultiattackOption{combat.WithGrantedAction(combat.GrantsDash{})}},
		{"zero used part", m, sbSelf(), state, []combat.MultiattackOption{combat.WithUsedThisTurn([]combat.LimitedUsePartRef{{}})}},
		{"repeated used part", m, sbSelf(), state, []combat.MultiattackOption{combat.WithUsedThisTurn([]combat.LimitedUsePartRef{ref, ref})}},
		{"usage-limited Multiattack", limited, sbSelf(), luStart(t, limited, combat.OutsideLair), nil},
		{"plan casting a spell the monster lacks", spellless, sbSelf(), luStart(t, spellless, combat.OutsideLair), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := combat.StartMultiattack(c.m, c.who, c.state, c.opts...)
			if out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("StartMultiattack = %#v, %v, want a nil outcome and an invalid-input error", out, err)
			}
		})
	}
	var rejected *combat.StatBlockFeatureIDRejected
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Slam")), nil))
	dup := sbMonster(t, &plan, sbPlain("Slam"), func(f *content.MonsterFields) { f.Traits = sbPlain(content.MultiattackName) })
	if _, err := combat.StartMultiattack(dup, sbSelf(), luStart(t, dup, combat.OutsideLair)); !errors.As(err, &rejected) {
		t.Errorf("a Multiattack whose id another part derives: error = %v, want a *StatBlockFeatureIDRejected", err)
	}
	// The steps reject a zero progress, a zero state and a state of another combatant.
	started, _ := combat.StartMultiattack(m, sbSelf(), state)
	p := started.(combat.MultiattackStarted).Progress()
	for name, call := range map[string]func() (combat.MultiattackStepOutcome, error){
		"zero progress attack": func() (combat.MultiattackStepOutcome, error) {
			return combat.MultiattackProgress{}.RecordAttack(ref, state)
		},
		"zero state attack": func() (combat.MultiattackStepOutcome, error) { return p.RecordAttack(ref, combat.LimitedUseState{}) },
		"zero state extra": func() (combat.MultiattackStepOutcome, error) {
			return p.RecordAdditionalUse(ref, combat.LimitedUseState{})
		},
		"zero state replacement": func() (combat.MultiattackStepOutcome, error) {
			return p.RecordReplacement(sbOption("A"), ref, combat.LimitedUseState{})
		},
		"state of another monster combatant": func() (combat.MultiattackStepOutcome, error) {
			return p.RecordAttack(ref, luMust(combat.StartLimitedUse(m, combat.MustCombatantID("other"), combat.OutsideLair)))
		},
	} {
		if out, err := call(); out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %#v, %v, want a nil outcome and an invalid-input error", name, out, err)
		}
	}
}

func mustPlanOf(t *testing.T, part content.StatBlockPart) content.MultiattackPlan {
	t.Helper()
	plan, ok := part.Plan()
	if !ok {
		t.Fatal("part has no plan")
	}
	return plan
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-021 - MultiattackPlanProblemsFor returns one problem per name or reference that does not
// resolve: a replacement spell absent from the Spellcasting; none for a sound plan, a monster with no plan, or a name repeated.
// Path: per case.
func TestMultiattackPlanProblemsFor(t *testing.T) {
	t.Parallel()
	sound := maAdultBrass(t, luPerDaySpells(1))
	missing := func(spell string) content.Monster {
		plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Rend")), nil,
			sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbSpellOption("Spellcasting", spell, 3)),
			sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, sbSpellOption("Spellcasting", spell, 3))))
		return sbMonster(t, &plan, sbPlain("Rend", "Spellcasting"), nil)
	}
	cases := []struct {
		name, requirement, path string
		m                       content.Monster
		want                    []combat.MultiattackPlanProblemKind
	}{
		{"sound plan", `FR-021`, "happy", sound, nil},
		{"no plan", `FR-021`, "unhappy (empty and minimal)", luMonster(t, nil), nil},
		{"spell absent from the Spellcasting", `FR-021`, "unhappy (missing content)", missing("Fireball"), []combat.MultiattackPlanProblemKind{combat.MultiattackPlanSpellMissing}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var got []combat.MultiattackPlanProblemKind
			for _, p := range combat.MultiattackPlanProblemsFor(c.m) {
				got = append(got, p.Kind())
				if p.Name() != "Fireball" {
					t.Errorf("problem names %q, want Fireball", p.Name())
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
	if got := []string{combat.MultiattackPlanPartMissing.String(), combat.MultiattackPlanPartShared.String(), combat.MultiattackPlanSpellMissing.String(), combat.MultiattackPlanProblemKind(0).String()}; !slices.Equal(got, []string{"part_missing", "part_shared", "spell_missing", "unknown"}) {
		t.Errorf("kind names = %v", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-013 - a recorded attack carries one event with the part and its 1-based number
// among the attacks made; FR-016 and FR-018 carry theirs; events are copies.
// Path: happy.
func TestMultiattackEvents(t *testing.T) {
	t.Parallel()
	m := maAboleth(t)
	state := luStart(t, m, combat.OutsideLair)
	started, _ := combat.StartMultiattack(m, sbSelf(), state)
	p := started.(combat.MultiattackStarted).Progress()
	tentacle := sbAction(t, m, "Tentacle")
	r1, _ := p.RecordAttack(tentacle, state)
	t1 := r1.(combat.MultiattackStepRecorded).Transition
	r2, _ := t1.Progress().RecordAttack(tentacle, t1.LimitedUse())
	t2 := r2.(combat.MultiattackStepRecorded).Transition
	for i, tr := range []combat.MultiattackTransition{t1, t2} {
		events := tr.Events()
		e, ok := events[len(events)-1].(combat.MultiattackAttackRecorded)
		if !ok || e.Number() != i+1 || e.Part() != tentacle {
			t.Errorf("attack %d: last event = %#v, want MultiattackAttackRecorded number %d", i+1, events[len(events)-1], i+1)
		}
		combattest.CheckEvent(t, e, "multiattack_attack_recorded", sbSelf())
		combattest.CheckEventsCopy(t, tr.Events)
	}
	mind := sbAction(t, m, "Dominate Mind")
	r3, _ := t2.Progress().RecordAdditionalUse(mind, t2.LimitedUse())
	t3 := r3.(combat.MultiattackStepRecorded).Transition
	events := t3.Events()
	if len(events) != 2 || events[0].Kind() != "limited_use_part_spent" {
		t.Fatalf("events of a limited additional use = %v, want the usage spent then the use recorded", eventKinds(events))
	}
	extraEvent, ok := events[1].(combat.MultiattackAdditionalUseRecorded)
	if !ok || extraEvent.Part() != mind {
		t.Errorf("last event = %#v, want MultiattackAdditionalUseRecorded for Dominate Mind", events[1])
	}
	combattest.CheckEvent(t, extraEvent, "multiattack_additional_use_recorded", sbSelf())
	g := maGladiator(t, maPlainBash())
	gstate := luStart(t, g, combat.OutsideLair)
	gs, _ := combat.StartMultiattack(g, sbSelf(), gstate)
	r4, _ := gs.(combat.MultiattackStarted).Progress().RecordReplacement(sbOption("Shield Bash"), sbAction(t, g, "Spear"), gstate)
	replaced, ok := r4.(combat.MultiattackStepRecorded).Transition.Events()[0].(combat.MultiattackAttackReplaced)
	if !ok || replaced.Part() != sbAction(t, g, "Shield Bash") || replaced.Replaced() != sbAction(t, g, "Spear") {
		t.Errorf("replacement event = %#v", replaced)
	}
	combattest.CheckEvent(t, replaced, "multiattack_attack_replaced", sbSelf())
	if got := len(r4.(combat.MultiattackStepRecorded).Transition.Progress().Attacks()); got != 0 {
		t.Errorf("a replaced attack counted as an attack made: %d attacks", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-019 - the plan is a ceiling: AttacksRemaining counts what is left (a positive
// number is no obligation), narrows with the variants still open, is 0 for a part no open variant lists and counts a replacement's slots.
// Path: per case.
func TestMultiattackAttacksRemaining(t *testing.T) {
	t.Parallel()
	devil := maDevil(t)
	devilState := luStart(t, devil, combat.OutsideLair)
	claws, flame := sbAction(t, devil, "Claws"), sbAction(t, devil, "Hurl Flame")
	start, _ := maRun(t, devil, devilState, nil)
	if start.AttacksRemaining(claws) != 1 || start.AttacksRemaining(flame) != 2 {
		t.Errorf("at the start Claws %d Hurl Flame %d, want 1 and 2", start.AttacksRemaining(claws), start.AttacksRemaining(flame))
	}
	afterClaws, _ := maRun(t, devil, devilState, []maStep{atk("Claws")})
	if afterClaws.AttacksRemaining(claws) != 0 || afterClaws.AttacksRemaining(flame) != 0 || afterClaws.AttacksRemaining(sbAction(t, devil, "Tail")) != 1 {
		t.Errorf("after Claws: Claws %d Hurl Flame %d Tail %d, want 0, 0, 1", afterClaws.AttacksRemaining(claws), afterClaws.AttacksRemaining(flame), afterClaws.AttacksRemaining(sbAction(t, devil, "Tail")))
	}
	brass := maBrass(t)
	afterReplace, _ := maRun(t, brass, luStart(t, brass, combat.OutsideLair), []maStep{repl(sbOption("Sleep Breath"), "Rend", "")})
	if got := afterReplace.AttacksRemaining(sbAction(t, brass, "Rend")); got != 1 {
		t.Errorf("Rend remaining after replacing two of three = %d, want 1", got)
	}
	if got := start.AttacksRemaining(combat.LimitedUsePartRef{}); got != 0 {
		t.Errorf("remaining for the zero part = %d, want 0", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-017 - attacks, additional uses and replacements may be recorded in any order;
// every order of a legal set of steps is accepted (a property over all permutations).
// Path: unhappy (rule interaction).
func TestMultiattackAnyOrderOfALegalSetIsAccepted(t *testing.T) {
	t.Parallel()
	sets := []struct {
		name  string
		m     content.Monster
		steps []maStep
	}{
		{"Aboleth: two Tentacle and one additional use", maAboleth(t), []maStep{atk("Tentacle"), atk("Tentacle"), extra("Consume Memories", "")}},
		{"Assassin: two Shortsword and a Light Crossbow", maAssassin(t), []maStep{atk("Shortsword"), atk("Light Crossbow"), atk("Shortsword")}},
		{"Gladiator: two Spear and a replacement", maGladiator(t, maPlainBash()), []maStep{atk("Spear"), atk("Spear"), repl(sbOption("Shield Bash"), "Spear", "")}},
		{"Brass dragon: a Rend and a replacement of two", maBrass(t), []maStep{atk("Rend"), repl(sbOption("Sleep Breath"), "Rend", "")}},
	}
	for _, set := range sets {
		t.Run(set.name, func(t *testing.T) {
			t.Parallel()
			state := luStart(t, set.m, combat.OutsideLair)
			permutations(set.steps, func(order []maStep) {
				p, _ := maRun(t, set.m, state, order)
				_ = p
			})
		})
	}
}

// permutations calls f with every order of steps.
func permutations(steps []maStep, f func([]maStep)) {
	var rec func(k int)
	cur := slices.Clone(steps)
	rec = func(k int) {
		if k == len(cur) {
			f(slices.Clone(cur))
			return
		}
		for i := k; i < len(cur); i++ {
			cur[k], cur[i] = cur[i], cur[k]
			rec(k + 1)
			cur[k], cur[i] = cur[i], cur[k]
		}
	}
	rec(0)
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - a step returns a new progress and leaves the receiver and the input state
// unchanged; the attacks returned are a copy; Equal compares the plan and the steps.
// Path: unhappy (immutability).
func TestMultiattackProgressIsImmutable(t *testing.T) {
	t.Parallel()
	m := maAssassin(t)
	state := luStart(t, m, combat.OutsideLair)
	started, _ := combat.StartMultiattack(m, sbSelf(), state)
	p := started.(combat.MultiattackStarted).Progress()
	again := started.(combat.MultiattackStarted).Progress()
	out, _ := p.RecordAttack(sbAction(t, m, "Shortsword"), state)
	next := out.(combat.MultiattackStepRecorded).Transition.Progress()
	if !p.Equal(again) || p.Equal(next) || !next.Equal(next) {
		t.Error("a step changed the receiver, or Equal does not tell progresses apart")
	}
	attacks := next.Attacks()
	attacks[0] = combat.LimitedUsePartRef{}
	if next.Attacks()[0] != sbAction(t, m, "Shortsword") {
		t.Error("Attacks returned the progress's own slice")
	}
	other, _ := maRun(t, maBalor(t), luStart(t, maBalor(t), combat.OutsideLair), nil)
	if p.Equal(other) || (combat.MultiattackProgress{}).Equal(p) || !(combat.MultiattackProgress{}).Equal(combat.MultiattackProgress{}) {
		t.Error("Equal does not tell different plans apart")
	}
	if !(combat.MultiattackProgress{}).IsZero() {
		t.Error("the zero progress is not zero")
	}
	for i, n := range []string{combat.MultiattackAttackNotAllowed.String(), combat.MultiattackNotEnoughAttacksToReplace.String(), combat.MultiattackAllowanceRefusal(0).String()} {
		if want := []string{"attack_not_allowed", "not_enough_attacks_to_replace", "unknown"}[i]; n != want {
			t.Errorf("refusal name = %q, want %q", n, want)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-018 - an attack group of the maximum count does not overflow the slot arithmetic.
// Path: unhappy (boundary).
func TestMultiattackHugeCountsDoNotOverflow(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(content.MonsterMaxNumber, "Rend")), nil,
		sbRule(content.MonsterMaxNumber, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, sbOption("Breath"))))
	m := sbMonster(t, &plan, sbPlain("Rend", "Breath"), nil)
	state := luStart(t, m, combat.OutsideLair)
	rend := sbAction(t, m, "Rend")
	p, _ := maRun(t, m, state, []maStep{repl(sbOption("Breath"), "Rend", "")})
	if got := p.AttacksRemaining(rend); got != 0 {
		t.Errorf("remaining after replacing every attack = %d, want 0", got)
	}
	// A second replacement of the maximum count would overflow a 32-bit sum if it were not widened.
	maRun(t, m, state, []maStep{repl(sbOption("Breath"), "Rend", ""), repl(sbOption("Breath"), "Rend", "not_enough_attacks_to_replace"), atkWant("Rend", "allowance_spent")})
	fresh, _ := maRun(t, m, state, nil)
	if got := fresh.AttacksRemaining(rend); got != content.MonsterMaxNumber {
		t.Errorf("remaining at the start = %d, want the maximum count", got)
	}
}

// maSpendAll spends the usage the step needs: the spell of a spell option, or every use of the part the step names.
func maSpendAll(t *testing.T, m content.Monster, state combat.LimitedUseState, st maStep) combat.LimitedUseState {
	t.Helper()
	if _, _, isSpell := st.option.Spell(); isSpell {
		out, ok := state.SpendSpell(luMust(combat.NewSpellUseRef(0, 0))).(combat.LimitedUseSpent)
		if !ok {
			t.Fatal("the spell could not be spent")
		}
		return out.Transition.State()
	}
	part := sbAction(t, m, st.part)
	for state.AvailabilityOf(part) == combat.UseAvailable {
		state = luSpent(t, state, part)
	}
	return state
}

// maGladiatorPlain is the Gladiator plan with an unlimited Shield Bash.
func maGladiatorPlain(t *testing.T) content.Monster {
	t.Helper()
	return maGladiator(t, maPlainBash())
}

// maAdultBrassTwice is the Adult Brass Dragon plan with a Spellcasting of 2/Day.
func maAdultBrassTwice(t *testing.T) content.Monster {
	t.Helper()
	return maAdultBrass(t, luPerDaySpells(2))
}

// maAdultBrassOnce is the Adult Brass Dragon plan with a Spellcasting of 1/Day.
func maAdultBrassOnce(t *testing.T) content.Monster {
	t.Helper()
	return maAdultBrass(t, luPerDaySpells(1))
}

// ExampleMultiattackProgress_RecordAttack starts the Multiattack of a monster that makes two Tentacle attacks and records them; the
// third is refused because the plan allows two.
func ExampleMultiattackProgress_RecordAttack() {
	t := &testing.T{}
	m := maAboleth(t)
	self := combat.MustCombatantID("aboleth")
	state, _ := combat.StartLimitedUse(m, self, combat.OutsideLair)
	started, _ := combat.StartMultiattack(m, self, state)
	progress := started.(combat.MultiattackStarted).Progress()
	tentacle, _ := combat.NewLimitedUsePartRef(combat.MonsterPartActions, 1)
	for range 3 {
		out, _ := progress.RecordAttack(tentacle, state)
		switch o := out.(type) {
		case combat.MultiattackStepRecorded:
			progress, state = o.Transition.Progress(), o.Transition.LimitedUse()
			fmt.Println("recorded", len(progress.Attacks()))
		case combat.MultiattackNotInAllowance:
			fmt.Println("refused", o.Reason)
		}
	}
	// Output:
	// recorded 1
	// recorded 2
	// refused allowance_spent
}

// maAdultBrassAtWill is the Adult Brass Dragon plan with an At Will Spellcasting.
func maAdultBrassAtWill(t *testing.T) content.Monster {
	t.Helper()
	return maAdultBrass(t, content.SpellFrequencyAtWill{})
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-018 - with several variants still open, a refused replacement reports the refusal furthest along (too few attacks before a spent
// limit before not listed), whichever order the variants are in.
// Path: per case.
func TestMultiattackReplacementRefusalAcrossOpenVariants(t *testing.T) {
	t.Parallel()
	spit := sbOption("Spit")
	slots := sbVariant("", sbGroups(sbGroup(1, "Claw")), nil, sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, spit))
	limited := sbVariant("", sbGroups(sbGroup(2, "Claw")), nil, sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, spit))
	unlisted := sbVariant("", sbGroups(sbGroup(2, "Claw")), nil)
	cases := []struct {
		name     string
		variants []content.MultiattackVariant
		want     string
	}{
		{"too few attacks, then a spent limit", []content.MultiattackVariant{slots, limited}, "not_enough_attacks_to_replace"},
		{"a spent limit, then too few attacks", []content.MultiattackVariant{limited, slots}, "not_enough_attacks_to_replace"},
		{"a spent limit, then a variant without the rule", []content.MultiattackVariant{limited, unlisted}, "replacement_limit_reached"},
		{"a variant without the rule, then a spent limit", []content.MultiattackVariant{unlisted, limited}, "replacement_limit_reached"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			plan := sbMultiattack(c.variants...)
			m := sbMonster(t, &plan, sbPlain("Claw", "Spit"), nil)
			steps := []maStep{repl(spit, "Claw", ""), repl(spit, "Claw", c.want)}
			maRun(t, m, luStart(t, m, combat.OutsideLair), steps)
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-012, FR-019 - a variant that requires a part used this turn is open only when the part is among those used, and AttacksRemaining
// is the most over the open variants: the ceiling of the conditional variant with the part used, the other without.
// Path: per case.
func TestMultiattackConditionalVariantCeilings(t *testing.T) {
	t.Parallel()
	m := maGolem(t)
	state := luStart(t, m, combat.OutsideLair)
	slam := sbAction(t, m, "Slam")
	hasten := []combat.MultiattackOption{combat.WithUsedThisTurn([]combat.LimitedUsePartRef{luRef(t, combat.MonsterPartBonusActions, 0)})}
	without, _ := maRun(t, m, state, nil)
	with, _ := maRun(t, m, state, nil, hasten...)
	twoWith, _ := maRun(t, m, state, []maStep{atk("Slam"), atk("Slam")}, hasten...)
	twoWithout, _ := maRun(t, m, state, []maStep{atk("Slam"), atk("Slam")})
	if got := [4]int{without.AttacksRemaining(slam), with.AttacksRemaining(slam), twoWith.AttacksRemaining(slam), twoWithout.AttacksRemaining(slam)}; got != [4]int{2, 3, 1, 0} {
		t.Errorf("Slam remaining without/with Hasten, after two with, after two without = %v, want [2 3 1 0]", got)
	}
	// Another part used this turn does not open the conditional variant.
	other := []combat.MultiattackOption{combat.WithUsedThisTurn([]combat.LimitedUsePartRef{luRef(t, combat.MonsterPartActions, 1)})}
	if p, _ := maRun(t, m, state, nil, other...); p.AttacksRemaining(slam) != 2 {
		t.Errorf("Slam remaining with another part used = %d, want 2", p.AttacksRemaining(slam))
	}
}

// maOverlapping is a monster whose additional uses are entries that share an option or list a single one: (A, B) and (B), or (A) and (B) alone.
func maOverlapping(t *testing.T, entries ...content.AdditionalUse) content.Monster {
	t.Helper()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Claw")), entries))
	return sbMonster(t, &plan, sbPlain("Claw", "A", "B"), nil)
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-016 - each additional-use entry takes one choice: a part listed by two entries may take either, the choices are matched across
// the entries whatever the order (a part found only in a later entry is not blocked by an earlier choice), an entry listing one part takes it once, and a part no entry lists is refused.
// Path: per case.
func TestMultiattackAdditionalUseEntriesAreMatched(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		entries []content.AdditionalUse
		steps   []maStep
	}{
		{"a shared option first, the other part after", []content.AdditionalUse{sbExtra("A", "B"), sbExtra("B")}, []maStep{extra("B", ""), extra("A", ""), extra("A", "additional_use_already_made"), extra("B", "additional_use_already_made")}},
		{"the other part first, the shared option after", []content.AdditionalUse{sbExtra("A", "B"), sbExtra("B")}, []maStep{extra("A", ""), extra("B", ""), extra("B", "additional_use_already_made")}},
		{"single-option entries", []content.AdditionalUse{sbExtra("A"), sbExtra("B")}, []maStep{extra("A", ""), extra("A", "additional_use_already_made"), extra("B", ""), extra("B", "additional_use_already_made")}},
		{"one single-option entry", []content.AdditionalUse{sbExtra("A")}, []maStep{extra("A", ""), extra("B", "additional_use_not_listed"), extra("A", "additional_use_already_made")}},
		{"a part no entry lists", []content.AdditionalUse{sbExtra("A")}, []maStep{extra("Claw", "additional_use_not_listed"), extra("B", "additional_use_not_listed")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m := maOverlapping(t, c.entries...)
			maRun(t, m, luStart(t, m, combat.OutsideLair), c.steps)
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-018 - with several variants open, a replacement is allowed as soon as one of them allows it, whatever the order of the variants; the variants
// that cannot take it close.
// Path: per case.
func TestMultiattackReplacementAllowedByAnyOpenVariant(t *testing.T) {
	t.Parallel()
	spit := sbOption("Spit")
	roomy := sbVariant("", sbGroups(sbGroup(3, "Claw")), nil, sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, spit))
	cramped := sbVariant("", sbGroups(sbGroup(1, "Claw")), nil, sbRule(2, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, spit))
	for name, variants := range map[string][]content.MultiattackVariant{"the variant that allows it first": {roomy, cramped}, "the variant that allows it last": {cramped, roomy}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan := sbMultiattack(variants...)
			m := sbMonster(t, &plan, sbPlain("Claw", "Spit"), nil)
			p, _ := maRun(t, m, luStart(t, m, combat.OutsideLair), []maStep{repl(spit, "Claw", ""), atk("Claw"), atk("Claw"), atkWant("Claw", "allowance_spent")})
			if got := p.AttacksRemaining(sbAction(t, m, "Claw")); got != 0 {
				t.Errorf("Claw remaining = %d, want 0", got)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-018, FR-016 - a replacement that casts a spell spends that spell, found by name among the spells of the Spellcasting, and not another.
// Path: unhappy (rule interaction).
func TestMultiattackReplacementSpendsTheNamedSpell(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(3, "Rend")), nil,
		sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbSpellOption("Spellcasting", "Scorching Ray", 3))))
	casting := luSpellcasting(luSpells(luPerDaySpells(1), "Burning Hands", "Scorching Ray", "Thunderwave"))
	m := sbMonster(t, &plan, append(sbPlain("Rend"), casting), nil)
	_, state := maRun(t, m, luStart(t, m, combat.OutsideLair), []maStep{repl(sbSpellOption("Spellcasting", "Scorching Ray", 3), "Rend", "")})
	for i, want := range []combat.LimitedUseAvailability{combat.UseAvailable, combat.UseExhausted, combat.UseAvailable} {
		if got := state.AvailabilityOfSpell(luMust(combat.NewSpellUseRef(0, i))); got != want {
			t.Errorf("spell %d availability = %v, want %v", i, got, want)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-021 - a replacement spell absent from a Spellcasting that exists (with other spells in it) is MultiattackPlanSpellMissing; one present is not.
// Path: unhappy (missing content).
func TestMultiattackPlanProblemsForASpellAbsentFromTheSpellcasting(t *testing.T) {
	t.Parallel()
	build := func(spell string) content.Monster {
		plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Rend")), nil, sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbSpellOption("Spellcasting", spell, 3))))
		return sbMonster(t, &plan, append(sbPlain("Rend"), luSpellcasting(luSpells(luPerDaySpells(1), "Burning Hands", "Thunderwave"))), nil)
	}
	problems := combat.MultiattackPlanProblemsFor(build("Fireball"))
	if len(problems) != 1 || problems[0].Kind() != combat.MultiattackPlanSpellMissing || problems[0].Name() != "Fireball" {
		t.Errorf("problems = %v, want the Fireball spell missing", problems)
	}
	if got := combat.MultiattackPlanProblemsFor(build("Thunderwave")); len(got) != 0 {
		t.Errorf("problems for a spell the Spellcasting has = %v, want none", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - Equal compares the combatant, the Multiattack part, the plan, the names it resolved, the variants open at the start and every step
// made; progresses that differ in exactly one of these are not equal, whichever the one.
// Path: per case.
func TestMultiattackProgressEqualDistinguishesEveryField(t *testing.T) {
	t.Parallel()
	option := sbSpellOption("Spellcasting", "Scorching Ray", 3)
	multi := func(count int) content.StatBlockPart {
		return sbMultiattack(sbVariant("", sbGroups(sbGroup(count, "Tentacle")), sbExtras(sbExtra("Dominate Mind")),
			sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, option)))
	}
	// build lays the Actions out as listed; "M" stands for the Multiattack and "S" for the Spellcasting with the spells given.
	build := func(count int, layout string, spells ...string) content.Monster {
		return luMonster(t, func(f *content.MonsterFields) {
			for _, name := range strings.Split(layout, ",") {
				switch name {
				case "M":
					f.Actions = append(f.Actions, multi(count))
				case "S":
					f.Actions = append(f.Actions, luSpellcasting(luSpells(luPerDaySpells(1), spells...)))
				default:
					f.Actions = append(f.Actions, sbPart(name, content.StatBlockPartFields{}))
				}
			}
		})
	}
	start := func(m content.Monster, who string) (combat.MultiattackProgress, combat.LimitedUseState) {
		id := combat.MustCombatantID(who)
		state := luMust(combat.StartLimitedUse(m, id, combat.OutsideLair))
		out, err := combat.StartMultiattack(m, id, state)
		if err != nil {
			t.Fatal(err)
		}
		return out.(combat.MultiattackStarted).Progress(), state
	}
	base := build(2, "M,Tentacle,Dominate Mind,S", "Scorching Ray")
	b, bstate := start(base, luSelf)
	tentacle, mind := sbAction(t, base, "Tentacle"), sbAction(t, base, "Dominate Mind")
	recorded := func(out combat.MultiattackStepOutcome, err error) combat.MultiattackProgress {
		if err != nil {
			t.Fatal(err)
		}
		return out.(combat.MultiattackStepRecorded).Transition.Progress()
	}
	other := func(m content.Monster) combat.MultiattackProgress { p, _ := start(m, luSelf); return p }
	otherCombatant, _ := start(base, "other")
	pairs := []struct {
		name string
		a, b combat.MultiattackProgress
	}{
		{"combatant", b, otherCombatant},
		{"plan counts", b, other(build(3, "M,Tentacle,Dominate Mind,S", "Scorching Ray"))},
		{"Multiattack part", other(build(2, "Aaa,Tentacle,M,Dominate Mind,S", "Scorching Ray")), other(build(2, "M,Tentacle,Aaa,Dominate Mind,S", "Scorching Ray"))},
		{"resolved part indexes", b, other(build(2, "M,Dominate Mind,Tentacle,S", "Scorching Ray"))},
		{"resolved spell index", b, other(build(2, "M,Tentacle,Dominate Mind,S", "Burning Hands", "Scorching Ray"))},
		{"an attack made", b, recorded(b.RecordAttack(tentacle, bstate))},
		{"an additional use made", b, recorded(b.RecordAdditionalUse(mind, bstate))},
		{"a replacement made", b, recorded(b.RecordReplacement(option, tentacle, bstate))},
	}
	for _, c := range pairs {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.a.Equal(c.b) || c.b.Equal(c.a) || !c.a.Equal(c.a) || !c.b.Equal(c.b) {
				t.Errorf("progresses differing in %s: Equal(a, b) = %v, Equal(b, a) = %v, want both false and each equal to itself", c.name, c.a.Equal(c.b), c.b.Equal(c.a))
			}
		})
	}
	// The variants open at the start differ with a part used this turn, and nothing else differs.
	gm := maGolem(t)
	gstate := luStart(t, gm, combat.OutsideLair)
	closed, _ := combat.StartMultiattack(gm, sbSelf(), gstate)
	open, _ := combat.StartMultiattack(gm, sbSelf(), gstate, combat.WithUsedThisTurn([]combat.LimitedUsePartRef{luRef(t, combat.MonsterPartBonusActions, 0)}))
	if closed.(combat.MultiattackStarted).Progress().Equal(open.(combat.MultiattackStarted).Progress()) {
		t.Error("progresses with different variants open are equal")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-010 - the Multiattack part is found by its name among the Actions wherever it stands, whatever the names before and after it.
// Path: per case.
func TestStartMultiattackFindsThePartWhereverItStands(t *testing.T) {
	t.Parallel()
	plan := sbMultiattack(sbVariant("", sbGroups(sbGroup(1, "Slam")), nil))
	for name, layout := range map[string][]string{"first": {"M", "Aaa", "Slam", "Zzz"}, "middle": {"Aaa", "Slam", "M", "Zzz"}, "last": {"Aaa", "Slam", "Zzz", "M"}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := luMonster(t, func(f *content.MonsterFields) {
				for _, n := range layout {
					if n == "M" {
						f.Actions = append(f.Actions, plan)
					} else {
						f.Actions = append(f.Actions, sbPart(n, content.StatBlockPartFields{}))
					}
				}
			})
			started, err := combat.StartMultiattack(m, sbSelf(), luStart(t, m, combat.OutsideLair))
			want := slices.Index(layout, "M")
			if err != nil || started.(combat.MultiattackStarted).Progress().Part() != luRef(t, combat.MonsterPartActions, want) {
				t.Errorf("StartMultiattack = %v, %v, want the Multiattack part at %d", started, err, want)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - a limited-use state of a combatant that sorts before or after the progress's is rejected like any other.
// Path: unhappy (invalid input).
func TestMultiattackStepRejectsAStateOfAnotherCombatantWhicheverWay(t *testing.T) {
	t.Parallel()
	m := maAboleth(t)
	state := luStart(t, m, combat.OutsideLair)
	started, _ := combat.StartMultiattack(m, sbSelf(), state)
	p := started.(combat.MultiattackStarted).Progress()
	for _, who := range []string{"aaa", "zzz"} {
		foreign := luMust(combat.StartLimitedUse(m, combat.MustCombatantID(who), combat.OutsideLair))
		if out, err := p.RecordAttack(luRef(t, combat.MonsterPartActions, 1), foreign); out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("state of %q: %#v, %v, want an invalid-input error", who, out, err)
		}
	}
}
