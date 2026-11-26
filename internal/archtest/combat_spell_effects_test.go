package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// spellEffectsDomainFiles and spellEffectsUseCaseFiles are the production files of spell-effects.md.
func spellEffectsDomainFiles() []string {
	return []string{"combat/spell_effects_targets.go", "combat/spell_effects_plan.go", "combat/spell_effects_resolver.go"}
}

func spellEffectsUseCaseFiles() []string { return []string{"combat/resolve_spell_effects_usecase.go"} }

// otherUseCases are the identifiers of the use cases and the d20 resolver that FR-096 forbids the spell effects files to name.
func otherUseCases() []string {
	return []string{"ResolveAttack", "NewResolveAttack", "ResolveUnarmedStrike", "NewResolveUnarmedStrike", "ApplyDamage", "NewApplyDamage", "RestoreHitPoints",
		"GainTemporaryHitPoints", "ApplyConditionToSet", "Resolver", "NewResolver"}
}

// Requirement: doc/specs/spell-effects.md FR-087, FR-096 - the three domain files import only the kernel, content, character, d20 and the standard library's fmt, maps, slices and strings, never telemetry, context or a roller; they name no signal constant, no Roller and no dice.Roll; every file of the spell effects names no other use case and no d20 Resolver; the use-case file adds context, strconv and telemetry.
// Path: unhappy (forbidden edge).
func TestSpellEffectsFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "character", "content", "d20", "dice", "modifier", "srderr"}
		domain := slices.Contains(spellEffectsDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(spellEffectsUseCaseFiles(), f.Path):
			allowed = append(allowed, "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) || (domain && p == "telemetry") {
					t.Errorf("%s imports %s, which spell-effects.md does not allow", f.Path, p)
				}
				continue
			}
			switch imp.Path {
			case "fmt", "maps", "slices", "strings":
			case "context", "strconv":
				if domain {
					t.Errorf("%s imports %s: the domain files are pure (FR-087)", f.Path, imp.Path)
				}
			default:
				t.Errorf("%s imports %s, which spell-effects.md does not allow", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && domain && id.Name == "dice" && x.Sel.Name == "Roll" {
					t.Errorf("%s calls dice.Roll: the domain requests no die itself (FR-087)", f.Path)
				}
			case *ast.Ident:
				if slices.Contains(otherUseCases(), x.Name) {
					t.Errorf("%s names %s: no use case is called or built (FR-096)", f.Path, x.Name)
				}
				if !domain {
					return true
				}
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(x.Name, prefix) && len(x.Name) > len(prefix) && x.Name[len(prefix)] >= 'A' && x.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, x.Name)
					}
				}
				if x.Name == "Roller" {
					t.Errorf("%s names Roller: the domain holds no roller (FR-087)", f.Path)
				}
			}
			return true
		})
	}
	if want := len(spellEffectsDomainFiles()) + len(spellEffectsUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d spell effects files", seen, want)
	}
}

// Requirement: doc/specs/spell-effects.md FR-054, FR-093 - the two closed sets have exactly the variants the spec lists.
// Path: unhappy (duplicate).
func TestSpellEffectsClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isResolveSpellEffectsResult": {"SpellEffectsRejected", "SpellEffectsResolved", "SpellEffectsTextOnly"},
		"isSpellTargetRef":            {"CreatureTargetRef", "ObjectTargetRef"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/spell-effects.md FR-096 - the constructor takes a Roller and options only.
// Path: unhappy (forbidden edge).
func TestResolveSpellEffectsConstructorTakesOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "NewResolveSpellEffects" {
				continue
			}
			found++
			var params []string
			for _, p := range fn.Type.Params.List {
				switch x := p.Type.(type) {
				case *ast.Ident:
					params = append(params, x.Name)
				case *ast.Ellipsis:
					params = append(params, "..."+x.Elt.(*ast.Ident).Name)
				}
			}
			if got := strings.Join(params, ","); got != "Roller,...Option" {
				t.Errorf("NewResolveSpellEffects takes (%s), want (Roller,...Option)", got)
			}
		}
	}
	if found != 1 {
		t.Errorf("found %d NewResolveSpellEffects, want 1", found)
	}
}

// Requirement: doc/specs/spell-effects.md section 7, FR-080, FR-081, FR-085, plan P11 - signals.go pins the operation, span, three metrics, four attribute keys, the value sets and the four result values at their catalogue values, and signals.md has a row for each metric and the span.
// Path: unhappy (invalid input).
func TestSpellEffectsSignalNamesAndCatalogue(t *testing.T) {
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
		"valueOperationResolveSpellEffects": "resolve_spell_effects", "spanResolveSpellEffects": "srd5e.combat.resolve_spell_effects",
		"metricSpellEffectsResolutions": "srd5e.combat.spell_effects.resolutions", "metricSpellEffectsRolls": "srd5e.combat.spell_effects.rolls",
		"metricSpellEffectsTargets": "srd5e.combat.spell_effects.targets", "attrSpellEffectsCoverage": "srd5e.combat.spell_effects.coverage",
		"attrSpellEffectsRejection": "srd5e.combat.spell_effects.rejection", "attrSpellEffectsLevel": "srd5e.combat.spell_effects.level",
		"attrSpellEffectsRoll": "srd5e.combat.spell_effects.roll", "valueSpellCoverageComplete": "complete", "valueSpellCoveragePartial": "partial",
		"valueSpellCoverageTextOnly": "text_only", "valueSpellLevelCantrip": "cantrip", "valueSpellRollAttack": "attack", "valueSpellRollSave": "save",
		"valueSpellRollSaveForced": "save_forced", "valueResultSpellResolved": "resolved", "valueResultSpellTextOnly": "text_only",
		"valueResultSpellRejectedInvalid": "rejected_invalid", "valueResultSpellRejectedRefused": "rejected_refused",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	doc := catalogue(t)
	for _, name := range []string{"srd5e.combat.resolve_spell_effects", "srd5e.combat.spell_effects.resolutions", "srd5e.combat.spell_effects.rolls",
		"srd5e.combat.spell_effects.targets", "srd5e.combat.spell_effects.coverage", "srd5e.combat.spell_effects.rejection",
		"srd5e.combat.spell_effects.level", "srd5e.combat.spell_effects.roll"} {
		if !strings.Contains(doc, "| `"+name+"` |") {
			t.Errorf("signals.md has no row for %s", name)
		}
	}
}

// Requirement: doc/specs/spell-effects.md section 6 (Exports), FR-089 - package combat declares the exported names of the spell effects once each and no generic variant name.
// Path: happy.
func TestCombatExportsSpellEffectsTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ResolveSpellEffects NewResolveSpellEffects ResolveSpellEffectsCommand ResolveSpellEffectsResult SpellEffectsResolved SpellEffectsTextOnly
		SpellEffectsRejected SpellEffectsRejection SpellEffectsRejectionKind SpellTarget SpellTargetParams NewSpellTarget SpellAttackFacts
		SpellAttackFactsParams NewSpellAttackFacts SpellTargetRef CreatureTargetRef ObjectTargetRef SpellObjectID ParseSpellObjectID MustSpellObjectID
		ErrBlankSpellObjectID ErrInvalidSpellEffectsCommand TargetValidity PathToTarget InstanceAssignment NewInstanceAssignment TargetEffects
		PlannedDamage PlannedAmount ConditionToApply InvalidTargetPerception ResolvedPart ResolvedInstance ResolvedSave RolledAmount
		WrongTargetCount NoClearPath InstanceAssignmentMismatch MissingSaveAttempt MissingAttackFacts DamageTypeNotChosen DamageTypeNotAllowed
		CasterNotAllowed WrongTargetSubject SpellAttackNotPossible AppearedToSucceed SpellDidNothing`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-089)", generic)
		}
	}
}
