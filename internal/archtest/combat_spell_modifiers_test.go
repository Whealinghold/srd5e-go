package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// spellModifierDomainFiles and spellModifierUseCaseFiles are the production files of spell-granted-modifiers.md.
func spellModifierDomainFiles() []string {
	return []string{"combat/spell_modifier_source.go", "combat/spell_modifier_catalogue.go", "combat/spell_modifier_fact.go", "combat/spell_modifier_roll.go",
		"combat/spell_modifier_armor_class.go", "combat/spell_modifier_speed.go", "combat/spell_modifier_damage.go"}
}

func spellModifierUseCaseFiles() []string { return []string{"combat/roll_spell_bonus_dice_usecase.go"} }

// Requirement: doc/specs/spell-granted-modifiers.md FR-080, FR-082, observability.md section 3 - the spell_modifier*.go files import only ability, character, content, d20, dice, modifier and the standard library's slices, never srderr, telemetry, context or a use case, name no signal constant and no ConditionEffects; the use-case file adds context, fmt and telemetry (and still no character or d20).
// Path: unhappy (forbidden edge).
func TestSpellModifierFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "character", "content", "d20", "dice", "modifier"}
		std := []string{"slices"}
		domain := slices.Contains(spellModifierDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(spellModifierUseCaseFiles(), f.Path):
			allowed = []string{"dice", "modifier", "telemetry"}
			std = []string{"context", "fmt", "slices"}
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which spell-granted-modifiers.md does not allow", f.Path, p)
				}
				continue
			}
			if !slices.Contains(std, imp.Path) {
				t.Errorf("%s imports %s, which spell-granted-modifiers.md does not allow", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if id.Name == "ConditionEffects" {
				t.Errorf("%s names ConditionEffects (FR-082)", f.Path)
			}
			if domain {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the domain is silent (FR-080)", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if want := len(spellModifierDomainFiles()) + len(spellModifierUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d spell modifier files", seen, want)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-082 - ReevaluateAttack calls the pure d20.ResolveAttack and never the d20.Resolver use case.
// Path: unhappy (forbidden edge).
func TestReevaluateAttackUsesThePureRule(t *testing.T) {
	t.Parallel()
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/spell_modifier_armor_class.go" {
			continue
		}
		var calls []string
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "d20" {
					calls = append(calls, sel.Sel.Name)
				}
			}
			return true
		})
		if slices.Contains(calls, "Resolver") || slices.Contains(calls, "NewResolver") || !slices.Contains(calls, "ResolveAttack") {
			t.Errorf("%s uses d20.%v, want ResolveAttack and no Resolver", f.Path, calls)
		}
		return
	}
	t.Fatal("combat/spell_modifier_armor_class.go not found")
}

// Requirement: doc/specs/spell-granted-modifiers.md section 5 - the closed sets have exactly the variants the spec lists (RollSpellBonusDiceResult has the one variant SpellBonusDiceRolled).
// Path: unhappy (duplicate).
func TestSpellModifierClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isSpellModifierChoice":      {"ChosenAbility", "ChosenCurse", "ChosenResistanceType", "ChosenSkill", "NoChoice"},
		"isCurseEffect":              {"CurseAbility", "CurseAttackRolls", "CurseDodge", "CurseExtraDamage"},
		"isSpellModifierLookup":      {"SpellModifiersKnown", "SpellModifiersNotModelled"},
		"isRollContext":              {"AbilityCheckContext", "AttackRollContext", "DeathSavingThrowContext", "SavingThrowContext"},
		"isDamageEvent":              {"AttackRollHitEvent", "OtherDamageEvent", "SpellDamageEvent"},
		"isRollSpellBonusDiceResult": {"SpellBonusDiceRolled"},
		"isSpellEffectFact": {"ActionOrBonusActionNotBoth", "EndsIfArmorDonned", "ExtraActionGranted", "ForcedDodgeUnlessWisdomSave", "LethargyOnEnd",
			"MarkMovesOnZeroHitPoints", "NoDamageFromMagicMissile", "OneAttackOnly", "ReactionsForbidden", "SpellFailureChancePercent"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-083, section 6 - package combat exports the spell-granted modifier types, queries and use case once each, and none is a generic variant name.
// Path: happy.
func TestCombatExportsSpellModifierTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		SpellModifierSource NewSpellModifierSource MustSpellModifierSource SpellModifierChoice NoChoice ChosenSkill ChosenAbility ChosenResistanceType ChosenCurse
		CurseEffect CurseAbility CurseAttackRolls CurseDodge CurseExtraDamage ErrDuplicateSource SpellModifierEntry SpellModifierLookup SpellModifiersKnown
		SpellModifiersNotModelled LookupSpellModifiers RollContext AttackRollContext SavingThrowContext DeathSavingThrowContext AbilityCheckContext BonusDie
		NewBonusDie MustBonusDie SpellRollContributions QueryRollContributions SpellArmorClassInputs QueryArmorClass ReevaluateAttack QuerySpeeds DamageEvent
		AttackRollHitEvent SpellDamageEvent OtherDamageEvent QueryDamageRiders SpellDamageReduction QueryDamageReduction SpellEffectFact ExtraActionGranted
		LethargyOnEnd ReactionsForbidden ActionOrBonusActionNotBoth OneAttackOnly SpellFailureChancePercent MarkMovesOnZeroHitPoints
		ForcedDodgeUnlessWisdomSave EndsIfArmorDonned NoDamageFromMagicMissile RollSpellBonusDice NewRollSpellBonusDice RollSpellBonusDiceCommand
		RollSpellBonusDiceResult SpellBonusDiceRolled`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound", "Known", "NotModelled", "Rolled"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-083)", generic)
		}
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-090, FR-092, section 7, plan-phase-4.md P11 - signals.go pins the operation value, the span and the two result values at their catalogue values, and signals.md has a row for the span.
// Path: unhappy (invalid input).
func TestSpellModifierSignalNamesAndCatalogue(t *testing.T) {
	t.Parallel()
	got := map[string]string{}
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/signals.go" {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if v, ok := n.(*ast.ValueSpec); ok && len(v.Values) == 1 {
				if lit, ok := v.Values[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					got[v.Names[0].Name], _ = strconv.Unquote(lit.Value)
				}
			}
			return true
		})
	}
	want := map[string]string{
		"valueOperationRollSpellBonusDice": "roll_spell_bonus_dice", "spanRollSpellBonusDice": "srd5e.combat.roll_spell_bonus_dice",
		"valueResultSpellBonusDiceRolled": "rolled", "valueResultSpellBonusDiceNothingToRoll": "nothing_to_roll",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	if !strings.Contains(catalogue(t), "| `srd5e.combat.roll_spell_bonus_dice` |") {
		t.Error("signals.md has no row for srd5e.combat.roll_spell_bonus_dice")
	}
}
