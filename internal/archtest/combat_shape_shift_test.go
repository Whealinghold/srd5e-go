package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// shapeShiftFiles returns the production files of shape-changing-spells.md: pure domain code in combat, files named shape_shift_*.go
// (shape_shifting.go belongs to magical-effects.md and is not among them).
func shapeShiftFiles(t *testing.T) []archtestFile {
	t.Helper()
	var out []archtestFile
	for _, f := range combatPackage(t).Production() {
		if strings.HasPrefix(f.Path, "combat/shape_shift_") {
			out = append(out, archtestFile{path: f.Path, ast: f.AST})
		}
	}
	if len(out) < 10 {
		t.Fatalf("found %d shape-shift files, want the spec's files", len(out))
	}
	return out
}

// Requirement: doc/specs/shape-changing-spells.md FR-057 - the shape-shift files are silent pure domain code: they import of this module only content, d20 and srderr (no character, toolbox, telemetry, dice or *test package), and name no Concentration type, no use case Instrumenter and no signal constant.
// Path: unhappy (forbidden edge).
func TestShapeShiftFilesImportOnlyThePublishedLanguage(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	for _, f := range shapeShiftFiles(t) {
		for _, imp := range f.ast.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if rest, ok := strings.CutPrefix(p, mod); ok && !slices.Contains([]string{"content", "d20", "srderr"}, rest) {
				t.Errorf("%s imports %s; the shape-changing spells may import only content, d20 and srderr [FR-057]", f.path, rest)
			}
		}
		ast.Inspect(f.ast, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			switch {
			case slices.Contains([]string{"ConcentrationRules", "ConcentrationState", "ConcentrationEffect", "ConcentrationHold"}, id.Name) || id.Name == "Instrumenter":
				t.Errorf("%s names %s; the spec reads Concentration only as a plain fact [FR-038, FR-057]", f.path, id.Name)
			case strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr"):
				t.Errorf("%s names the signal constant %s: the spec owns no signal (section 7)", f.path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-056 - exactly ShapeShiftBegan, ShapeShiftFormChanged, ShapeShiftMadePermanent and ShapeShiftEnded embed the shape-shift event base, all declared in shape_shift_event.go.
// Path: unhappy (duplicate).
func TestShapeShiftEventFamilyHasExactlyFourMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "shapeShiftEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/shape_shift_event.go" {
							t.Errorf("%s is declared in %s, want combat/shape_shift_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"ShapeShiftBegan", "ShapeShiftEnded", "ShapeShiftFormChanged", "ShapeShiftMadePermanent"}; !slices.Equal(members, want) {
		t.Errorf("shape-shift event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-057, section 6 - package combat declares the exported names of the spec once each, declares ShapeShifting once (it is magical-effects.md's) and none of ConcentrationEffect, WildShape*, BeastFormFacts, and no generic variant name.
// Path: unhappy (duplicate).
func TestCombatExportsShapeShiftTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ShapeShiftSpell PolymorphSpell TruePolymorphSpell ShapechangeSpell AnimalShapesSpell ShapeShiftSpellOf ShapeShiftProfile TrueTransformation
		CreatureIntoCreature ObjectIntoCreature CreatureIntoObject ShapechangeTerms StandardShapechange StatBlockShapechange EquipmentOutcome
		MeldsUnusable DropsToGround ResizesToFit TransformsWithCreature FormTemporaryHitPoints MaintainedBy MaintainedByConcentration NotMaintained
		ShapeShiftEndCause ShapeShiftActionCost RetainedStatistic FormStatistics SpeechRestriction SpellcastingRestriction FormRestrictions
		FormFacts NewFormFacts AssumedForm CreatureForm NewCreatureForm ObjectForm NewObjectForm SizeError ObjectFacts TargetRating RatedByChallenge
		NewRatedByChallenge RatedByLevel NewRatedByLevel LevelRangeError FormIneligibility FormEligibility FormEligible FormIneligible FormQuery
		ActiveTransformation ActiveTransformationFields ActiveTransformationProblem NewActiveTransformation MustActiveTransformation
		ShapeShiftEvent ShapeShiftBegan ShapeShiftFormChanged ShapeShiftMadePermanent ShapeShiftEnded
		TargetResponse Willing Unwilling TargetHitPoints TrackedHitPoints UntrackedHitPoints AnimatedObjectTerms
		BeginShapeShiftCommand BeginShapeShiftingResult BeginShapeShiftApplied BeginShapeShiftResisted BeginShapeShiftRefused BeginShapeShiftRejection
		TargetIsDead TargetNotWilling ShapechangeNotSelf ChangeFormCommand ChangeShapeShiftFormResult ChangeFormApplied ChangeFormRefused ChangeFormRejection
		FormChangeNotAllowed NotTransformed ActorNotCaster EndShapeShiftCommand EndShapeShiftingResult EndShapeShiftApplied EndShapeShiftUnaffected
		EndShapeShiftRefused EndShapeShiftRejection ElapseFullDurationResult ElapseShapeShiftMadePermanent ElapseShapeShiftEnded ElapseShapeShiftUnaffected
		AssessShapeShiftResult AssessShapeShiftContinues AssessShapeShiftEnded ShapeShiftSpells ShapeShifting`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, name := range strings.Fields(`ConcentrationEffect BeastFormFacts WildShape WildShapeForm WildShapeState`) {
		if len(have[name]) != 0 {
			t.Errorf("combat declares %s, which belongs to another spec [FR-057]", name)
		}
	}
	for name := range have {
		if strings.HasPrefix(name, "WildShape") {
			t.Errorf("combat declares %s, which belongs to class-druid.md [FR-057]", name)
		}
	}
	for _, f := range shapeShiftFiles(t) {
		for _, decl := range f.ast.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "ShapeShifting" {
					t.Errorf("%s declares ShapeShifting, which belongs to magical-effects.md [FR-057]", f.path)
				}
			}
		}
	}
}
