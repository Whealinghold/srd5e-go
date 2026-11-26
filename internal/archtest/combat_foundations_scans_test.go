package archtest_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-111 - HitKind has exactly two members, declared in hit_point_inputs.go only,
// and no damage call takes IsCriticalHit as an input; the clean forms stay silent.
// Path: per case.
func TestCombatHitKindScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatHitKind
	const decl = "package combat\n\n// HitKind says whether damage came from a Critical Hit.\ntype HitKind int\n\nconst (\n\t// NormalHit is not.\n\tNormalHit HitKind = iota + 1\n\t// CriticalHit is.\n\tCriticalHit\n)\n"
	cases := []struct {
		name, requirement, path string
		files                   map[string]string
		want                    string // "" means no combat rule fires
	}{
		{"two members in the HitKind file", "FR-111", "happy", map[string]string{"combat/hit_point_inputs.go": decl}, ""},
		{"a third member", "FR-111", "unhappy (invalid input)", map[string]string{"combat/hit_point_inputs.go": decl + "\n// GlancingHit is a third.\nconst GlancingHit HitKind = 3\n\n// Extra groups.\nconst (\n\t// A is one.\n\tA HitKind = 4\n)\n"}, rule},
		{"members declared in another file", "FR-111", "unhappy (invalid input)", map[string]string{"combat/other.go": decl}, rule},
		{"IsCriticalHit handed to a damage call", "FR-111", "unhappy (invalid input)", map[string]string{"combat/use.go": "package combat\n\nfunc use(a attack, e enc) { e.ApplyDamage(a.IsCriticalHit()) }\n"}, rule},
		{"IsCriticalHit nested in a damage call", "FR-111", "unhappy (invalid input)", map[string]string{"combat/use.go": "package combat\n\nfunc use(a attack) { NewDealDamageCommand(f(a.r.IsCriticalHit())) }\n"}, rule},
		{"IsCriticalHit read for the attack outcome", "FR-111", "happy", map[string]string{"combat/use.go": "package combat\n\nfunc use(a attack) bool { return a.IsCriticalHit() }\n"}, ""},
		{"a damage call with a HitKind", "FR-111", "happy", map[string]string{"combat/use.go": "package combat\n\nfunc use(e enc) { e.ApplyDamage(CriticalHit) }\n"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, c.files)
			switch {
			case c.want == "" && len(got) != 0:
				t.Errorf("want silence, got %v [%s]", got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-016, FR-017 - an event file (*_event.go, or a file declaring Kind) imports no
// reflect and holds no %T; another file may.
// Path: per case.
func TestCombatEventFilesScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatEventFiles
	cases := []struct {
		name, requirement, path string
		files                   map[string]string
		want                    string
	}{
		{"a clean event file", "FR-017", "happy", map[string]string{"combat/x_event.go": "package combat\n\n// Kind is the kind.\nfunc (E) Kind() string { return \"x\" }\n"}, ""},
		{"reflect in an _event.go file", "FR-017", "unhappy (invalid input)", map[string]string{"combat/x_event.go": "package combat\n\nimport \"reflect\"\n\nvar _ = reflect.TypeOf\n"}, rule},
		{"%T in an _event.go file", "FR-017", "unhappy (invalid input)", map[string]string{"combat/x_event.go": "package combat\n\nconst m = \"event %T\"\n"}, rule},
		{"reflect in a file declaring Kind", "FR-017", "unhappy (invalid input)", map[string]string{"combat/y.go": "package combat\n\nimport \"reflect\"\n\nvar _ = reflect.TypeOf\n\n// Kind is the kind.\nfunc (E) Kind() string { return \"y\" }\n"}, rule},
		{"%T in a file declaring Kind", "FR-017", "unhappy (invalid input)", map[string]string{"combat/y.go": "package combat\n\nconst m = \"%T\"\n\n// Kind is the kind.\nfunc (E) Kind() string { return \"y\" }\n"}, rule},
		{"%T in a file that is no event file", "FR-017", "happy", map[string]string{"combat/z.go": "package combat\n\nconst m = \"%T\"\n"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, c.files)
			switch {
			case c.want == "" && slices.Contains(got, rule):
				t.Errorf("want no %s, got %v [%s]", rule, got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-031 - a srd5e.combat.result constant has exactly one result class: two description
// literals that class it differently fail, and a constant no file outside signals.go names (so no class) fails.
// Path: per case.
func TestCombatResultClassScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatResultClass
	const signals = "package combat\n\nconst (\n\tvalueResultX = \"x\"\n\tvalueResultClassApplied = \"applied\"\n)\n"
	cases := []struct {
		name, requirement, path string
		files                   map[string]string
		want                    string
	}{
		{"one class", "FR-031", "happy", map[string]string{"combat/signals.go": signals, "combat/a.go": "package combat\n\nvar _ = description{Class: resultApplied, Name: valueResultX}\n"}, ""},
		{"the same class twice", "FR-031", "happy", map[string]string{"combat/signals.go": signals, "combat/a.go": "package combat\n\nvar _ = description{Class: resultApplied, Name: valueResultX}\nvar _ = description{Class: resultApplied, Name: valueResultX}\n"}, ""},
		{"a value named by a call", "FR-031", "happy", map[string]string{"combat/signals.go": signals, "combat/a.go": "package combat\n\nvar _ = f(valueResultX)\n"}, ""},
		{"two classes", "FR-031", "unhappy (rule interaction)", map[string]string{"combat/signals.go": signals, "combat/a.go": "package combat\n\nvar _ = description{Class: resultApplied, Name: valueResultX}\n", "combat/b.go": "package combat\n\nvar _ = description{Class: resultRefused, Name: valueResultX}\n"}, rule},
		{"a value no use case names", "FR-031", "unhappy (invalid input)", map[string]string{"combat/signals.go": signals}, rule},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, c.files)
			switch {
			case c.want == "" && slices.Contains(got, rule):
				t.Errorf("want no %s, got %v [%s]", rule, got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-152 to FR-155 - an Example function of package combat has an Output comment, no default arm, no print or println builtin and no clock, randomness, environment, network or file system; the clean form and a helper that is no Example stay silent.
// Path: per case.
func TestCombatExamplesScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatExamples
	const head = "package combat_test\n\n"
	cases := []struct {
		name, requirement, path string
		src                     string
		want                    string
	}{
		{"a clean example", "FR-152", "happy", head + "import \"fmt\"\n\nfunc ExampleX() {\n\tfmt.Println(1)\n\t// Output:\n\t// 1\n}\n", ""},
		{"an exhaustive switch", "FR-153", "happy", head + "func ExampleX() {\n\tswitch r.(type) {\n\tcase A:\n\tcase B:\n\t}\n\t// Output:\n}\n", ""},
		{"no Output comment", "FR-152", "unhappy (invalid input)", head + "func ExampleX() {\n}\n", rule},
		{"a default arm", "FR-153", "unhappy (invalid input)", head + "func ExampleX() {\n\tswitch r.(type) {\n\tcase A:\n\tdefault:\n\t}\n\t// Output:\n}\n", rule},
		{"the print builtin", "FR-155", "unhappy (invalid input)", head + "func ExampleX() {\n\tprintln(1)\n\t// Output:\n}\n", rule},
		{"a Duration is no clock", "FR-154", "happy", head + "import \"time\"\n\nfunc ExampleX() {\n\t_ = time.Hour\n\t// Output:\n}\n", ""},
		{"the clock", "FR-154", "unhappy (invalid input)", head + "import \"time\"\n\nfunc ExampleX() {\n\t_ = time.Now()\n\t// Output:\n}\n", rule},
		{"randomness", "FR-154", "unhappy (invalid input)", head + "import \"math/rand/v2\"\n\nfunc ExampleX() {\n\t_ = rand.IntN(3)\n\t// Output:\n}\n", rule},
		{"the environment", "FR-154", "unhappy (invalid input)", head + "import \"os\"\n\nfunc ExampleX() {\n\t_ = os.Getenv(\"A\")\n\t// Output:\n}\n", rule},
		{"a helper that is no example", "FR-153", "happy", head + "func helper() {\n\tswitch r.(type) {\n\tdefault:\n\t}\n\tprintln(1)\n}\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, map[string]string{"combat/x_example_test.go": c.src})
			switch {
			case c.want == "" && slices.Contains(got, rule):
				t.Errorf("want no %s, got %v [%s]", rule, got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// ruleCase is one fixture of a combat scan: the files, and the rule that must fire ("" means it must stay silent).
type ruleCase struct {
	name, requirement, path string
	files                   map[string]string
	want                    string
}

// runRuleCases runs each case against the combat scans and checks that rule fires, or is silent when want is "".
func runRuleCases(t *testing.T, rule string, cases []ruleCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, c.files)
			switch {
			case c.want == "" && slices.Contains(got, rule):
				t.Errorf("want no %s, got %v [%s]", rule, got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

const attrSignals = "package combat\n\nconst attrRosterReason = \"srd5e.combat.roster_reason\"\n"

// Requirement: doc/specs/combat-foundations.md FR-041 - the key of every attribute built in combat is an attr constant of signals.go, a d20.Attr key or a telemetry.Attr key.
// Path: per case.
func TestCombatAttrKeyScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatAttrKey
	use := func(body string) map[string]string {
		return map[string]string{"combat/signals.go": attrSignals, "combat/a.go": "package combat\n\nvar _ = " + body + "\n"}
	}
	runRuleCases(t, rule, []ruleCase{
		{"a combat attr constant", "FR-041", "happy", use("telemetry.String(attrRosterReason, \"fled\")"), ""},
		{"a d20 key", "FR-041", "happy", use("telemetry.String(d20.AttrRollMode, \"advantage\")"), ""},
		{"a telemetry key", "FR-041", "happy", use("telemetry.String(telemetry.AttrOutcome, \"success\")"), ""},
		{"a Bool and an Int with a constant", "FR-041", "happy", use("[]any{telemetry.Bool(attrRosterReason, true), telemetry.Int(attrRosterReason, 1)}"), ""},
		{"a string literal", "FR-041", "unhappy (invalid input)", use("telemetry.String(\"srd5e.combat.roster_reason\", \"fled\")"), rule},
		{"a constant signals.go does not declare", "FR-041", "unhappy (invalid input)", use("telemetry.String(attrUnknown, \"x\")"), rule},
		{"a non-attr constant of signals.go", "FR-041", "unhappy (invalid input)", map[string]string{"combat/signals.go": attrSignals + "\nconst spanX = \"srd5e.combat.x\"\n", "combat/a.go": "package combat\n\nvar _ = telemetry.String(spanX, \"x\")\n"}, rule},
		{"a key from a variable", "FR-041", "unhappy (invalid input)", use("telemetry.String(key, \"x\")"), rule},
		{"a selector of another package", "FR-041", "unhappy (invalid input)", use("telemetry.String(other.AttrX, \"x\")"), rule},
		{"another telemetry function", "FR-041", "happy", use("telemetry.StartTimer(\"x\")"), ""},
	})
}

// Requirement: doc/specs/combat-foundations.md FR-033 - an attribute value built in combat names no CombatantID, id, Amount, Message or Text and calls no Error method.
// Path: per case.
func TestCombatAttrValueScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatAttrValue
	use := func(body string) map[string]string {
		return map[string]string{"combat/signals.go": attrSignals, "combat/a.go": "package combat\n\nvar _ = " + body + "\n"}
	}
	runRuleCases(t, rule, []ruleCase{
		{"an enum name", "FR-033", "happy", use("telemetry.String(attrRosterReason, r.Reason.String())"), ""},
		{"a bool of an id comparison", "FR-033", "happy", use("telemetry.Bool(attrRosterReason, c.ID == other)"), ""},
		{"a CombatantID", "FR-033", "unhappy (invalid input)", use("telemetry.String(attrRosterReason, string(CombatantID(x)))"), rule},
		{"an ID selector", "FR-033", "unhappy (invalid input)", use("telemetry.String(attrRosterReason, c.ID.String())"), rule},
		{"a typed id selector", "FR-033", "unhappy (invalid input)", use("telemetry.String(attrRosterReason, string(c.MonsterID))"), rule},
		{"an Amount", "FR-033", "unhappy (invalid input)", use("telemetry.Int(attrRosterReason, r.Amount)"), rule},
		{"a Message", "FR-033", "unhappy (invalid input)", use("telemetry.String(attrRosterReason, r.Message)"), rule},
		{"a Text", "FR-033", "unhappy (invalid input)", use("telemetry.Float(attrRosterReason, Text)"), rule},
		{"an Error call", "FR-033", "unhappy (invalid input)", use("telemetry.String(attrRosterReason, err.Error())"), rule},
	})
}

// Requirement: doc/specs/combat-foundations.md FR-032, FR-035 - only combat/instrumenter.go calls RecordError, SetStatus, Tracer.Start and telemetry.StartTimer; a type named like the call is no hit.
// Path: per case.
func TestCombatSpanCallsScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatSpanCalls
	use := func(file, body string) map[string]string {
		return map[string]string{file: "package combat\n\nfunc f() {\n\t" + body + "\n}\n"}
	}
	runRuleCases(t, rule, []ruleCase{
		{"the instrumenter records the error", "FR-032", "happy", use("combat/instrumenter.go", "span.RecordError(err)\n\tspan.SetStatus(1)\n\t_, s := tel.Tracer.Start(ctx, \"x\")\n\t_ = telemetry.StartTimer(c, h)"), ""},
		{"a type named like the call", "FR-032", "happy", map[string]string{"combat/hazard_record.go": "package combat\n\n// HazardRecordError is a type.\ntype HazardRecordError struct{}\n\nfunc f() { _ = &HazardRecordError{} }\n"}, ""},
		{"an encounter Start", "FR-035", "happy", use("combat/encounter.go", "e.Start(id)"), ""},
		{"RecordError in a use case", "FR-032", "unhappy (invalid input)", use("combat/a.go", "span.RecordError(err)"), rule},
		{"SetStatus in a use case", "FR-032", "unhappy (invalid input)", use("combat/a.go", "span.SetStatus(1)"), rule},
		{"Tracer.Start in a use case", "FR-035", "unhappy (invalid input)", use("combat/a.go", "_, s := t.Tracer.Start(ctx, \"x\")"), rule},
		{"StartTimer in a use case", "FR-035", "unhappy (invalid input)", use("combat/a.go", "_ = telemetry.StartTimer(c, h)"), rule},
	})
}
