package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// itemUseDomainFiles and itemUseUseCaseFiles are the production files of item-use-effects.md.
func itemUseDomainFiles() []string {
	return []string{"combat/item_use_problem.go", "combat/item_use_kind.go", "combat/item_use_thrown.go", "combat/item_use_oil.go", "combat/item_use_event.go",
		"combat/item_use_poison.go", "combat/item_use_antitoxin.go", "combat/item_use_kit.go"}
}

func itemUseUseCaseFiles() []string {
	return []string{"combat/resolve_thrown_item.go", "combat/use_potion_of_healing.go"}
}

// Requirement: doc/specs/item-use-effects.md FR-070, observability.md section 3 - the item-use domain files import only the kernel, content, d20, character and the standard library's math, slices and time, never telemetry or context, and name no signal constant; the two use-case files add dice, modifier and telemetry.
// Path: unhappy (forbidden edge).
func TestItemUseFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "character", "content", "d20", "dice", "modifier", "srderr"}
		domain := slices.Contains(itemUseDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(itemUseUseCaseFiles(), f.Path):
			allowed = append(allowed, "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which item-use-effects.md does not allow", f.Path, p)
			}
			if domain && (strings.HasSuffix(imp.Path, "/telemetry") || imp.Path == "context") {
				t.Errorf("%s imports %s: the item-use domain is silent (FR-070)", f.Path, imp.Path)
			}
		}
		if !domain {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the domain is silent (FR-070)", f.Path, id.Name)
			}
			return true
		})
	}
	if want := len(itemUseDomainFiles()) + len(itemUseUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d item-use files", seen, want)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-034, FR-061 - no item-use file names a TurnTracker, a ConditionActionAvailability, a BonusActionGrant or a ConditionSet: the cost is data the host spends and Antitoxin touches no condition.
// Path: unhappy (forbidden edge).
func TestItemUseFilesNameNoTurnOrConditionState(t *testing.T) {
	t.Parallel()
	banned := []string{"TurnTracker", "ConditionActionAvailability", "BonusActionGrant", "ConditionSet", "Condition"}
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(itemUseDomainFiles(), f.Path) && !slices.Contains(itemUseUseCaseFiles(), f.Path) {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(banned, id.Name) {
				t.Errorf("%s names %s (FR-034, FR-061)", f.Path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-072 - exactly the seven item-use events embed the item-use event base, all declared in item_use_event.go.
// Path: unhappy (duplicate).
func TestItemUseEventFamilyHasExactlySevenMembers(t *testing.T) {
	t.Parallel()
	var members []string
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, fld := range st.Fields.List {
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "itemUseEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/item_use_event.go" {
							t.Errorf("%s is declared in %s, want combat/item_use_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"AntitoxinDrunk", "AntitoxinWoreOff", "BasicPoisonApplied", "BasicPoisonSpent", "HealersKitUseSpent", "PotionOfHealingConsumed", "ThrownItemUsed"}
	if !slices.Equal(members, want) {
		t.Errorf("item-use event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-071, FR-073 - the closed sets have exactly the variants the spec lists, and no event shares a name with a result variant.
// Path: unhappy (duplicate, name collision).
func TestItemUseClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isResolveThrownItemResult":  {"ThrownItemNotPossible", "ThrownItemResolved"},
		"isUsePotionOfHealingResult": {"PotionOfHealingNotPossible", "PotionOfHealingUsed"},
		"isThrownTarget":             {"ThrownAtCreature", "ThrownAtObject"},
		"isThrownSave":               {"ThrownSaveForced", "ThrownSaveRolled", "ThrownSaveSkipped"},
		"isThrownEffect":             {"ThrownDamageAndBurningStarted", "ThrownDamageDealt", "ThrownNoEffect", "ThrownOilCovered", "ThrownRestrained"},
		"isItemConsumption":          {"ItemConsumed", "ItemNotConsumed"},
		"isOilIgnition":              {"OilIgnited", "OilNotIgnited"},
		"isBasicPoisonExtra":         {"BasicPoisonExtraDamage", "BasicPoisonNoExtraDamage"},
		"isAntitoxinAdvantage":       {"AntitoxinGivesAdvantage", "AntitoxinGivesNothing"},
		"isHealersKitOutcome":        {"HealersKitEmpty", "HealersKitRefused", "HealersKitUsed"},
		"isNetOutcome":               {"NetEscaped", "NetHoldsFast"},
		"isPotionUse":                {"AdministerPotion", "DrinkPotion"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
	have := declared(combatPackage(t))
	for _, name := range []string{"ThrownItemUsed", "BasicPoisonApplied", "BasicPoisonSpent", "AntitoxinDrunk", "AntitoxinWoreOff", "PotionOfHealingConsumed", "HealersKitUseSpent"} {
		for _, other := range []string{"PotionOfHealingUsed", "HealersKitUsed", "ThrownItemResolved"} {
			if name == other {
				t.Errorf("the event %s shares the name of a result variant", name)
			}
		}
		if len(have[name]) != 1 {
			t.Errorf("combat declares the event %s %d times, want once", name, len(have[name]))
		}
	}
}

// Requirement: doc/specs/item-use-effects.md FR-071 - the two use case constructors take a Roller and options only.
// Path: unhappy (forbidden edge).
func TestItemUseConstructorsTakeOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	want := map[string]string{"NewResolveThrownItem": "Roller,...Option", "NewUsePotionOfHealing": "Roller,...Option"}
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || want[fn.Name.Name] == "" {
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
			if got := strings.Join(params, ","); got != want[fn.Name.Name] {
				t.Errorf("%s takes (%s), want (%s)", fn.Name.Name, got, want[fn.Name.Name])
			}
		}
	}
	if found != len(want) {
		t.Errorf("found %d of %d constructors", found, len(want))
	}
}

// Requirement: doc/specs/item-use-effects.md section 7, plan-phase-4.md P11 - signals.go pins the two operation values and spans, the item attribute key and its value, and the three new result values at their catalogue values, and signals.md has a row for each span and for the attribute.
// Path: unhappy (invalid input).
func TestItemUseSignalNamesAndCatalogue(t *testing.T) {
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
		"attrItemUseItem": "srd5e.combat.item_use.item", "valueItemUsePotionOfHealing": "potion_of_healing",
		"valueResultDamagedAndBurning": "damaged_and_burning", "valueResultCoveredInOil": "covered_in_oil", "valueResultRestrained": "restrained",
	}
	doc := catalogue(t)
	for _, op := range strings.Fields("ResolveThrownItem:resolve_thrown_item UsePotionOfHealing:use_potion_of_healing") {
		name, value, _ := strings.Cut(op, ":")
		want["valueOperation"+name] = value
		want["span"+name] = "srd5e.combat." + value
		if !strings.Contains(doc, "| `srd5e.combat."+value+"` |") {
			t.Errorf("signals.md has no row for the span srd5e.combat.%s", value)
		}
	}
	if !strings.Contains(doc, "| `srd5e.combat.item_use.item` |") {
		t.Error("signals.md has no row for the attribute srd5e.combat.item_use.item")
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
}

// Requirement: doc/specs/item-use-effects.md FR-077, section 6 (Exports) - package combat exports the item-use types, rules and use cases once each and no generic variant name.
// Path: happy.
func TestCombatExportsItemUseTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ItemUseKind ItemActionCost ItemUseRules ItemConsumption ItemConsumed ItemNotConsumed ItemSaveDCOf ItemUseField ItemUseError
		ThrownItemKind ThrownItemFacts ThrownItemFactsOf ThrownItemRules ThrownTarget ThrownAtCreature ThrownAtObject ThrownSight ThrownCover ThrownNotPossibleReason ThrownNoEffectReason
		ThrownEffect ThrownDamageDealt ThrownDamageAndBurningStarted ThrownOilCovered ThrownRestrained ThrownNoEffect ThrownSave ThrownSaveRolled ThrownSaveForced ThrownSaveSkipped
		NetRestraint NetOutcome NetEscaped NetHoldsFast OilCoat OilCoatState NewOilCoat OilIgnition OilIgnited OilNotIgnited OilNotIgnitedReason
		BasicPoisonObject BasicPoisonState BasicPoisonCoating BasicPoisonUpdate BasicPoisonExtra BasicPoisonExtraDamage BasicPoisonNoExtraDamage BasicPoisonNoExtraReason BasicPoisonApplication BasicPoisonRules
		AntitoxinEffect AntitoxinDrink AntitoxinUpdate AntitoxinRules AntitoxinAdvantage AntitoxinGivesAdvantage AntitoxinGivesNothing AntitoxinNothingReason PoisonedSaveIntent
		HealersKit NewHealersKit MustHealersKit FreshHealersKit HealersKitOutcome HealersKitUsed HealersKitEmpty HealersKitRefused
		ItemUseEvent ThrownItemUsed BasicPoisonApplied BasicPoisonSpent AntitoxinDrunk AntitoxinWoreOff PotionOfHealingConsumed HealersKitUseSpent
		ResolveThrownItem NewResolveThrownItem ResolveThrownItemInput ResolveThrownItemCommand NewResolveThrownItemCommand MustResolveThrownItemCommand ResolveThrownItemResult ThrownItemResolved ThrownItemNotPossible
		UsePotionOfHealing NewUsePotionOfHealing UsePotionOfHealingCommand NewUsePotionOfHealingCommand MustUsePotionOfHealingCommand UsePotionOfHealingResult PotionOfHealingUsed PotionOfHealingNotPossible
		PotionUse DrinkPotion AdministerPotion PotionNotPossibleReason`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Potion", "Thrown", "Coating", "Antitoxin", "Kit", "Oil", "Used", "Empty", "Refused"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-077)", generic)
		}
	}
}
