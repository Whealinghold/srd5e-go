package archtest_test

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// specKind returns the Kind its owning spec names for an event whose Kind is not the snake_case of the class name: item-use-effects.md
// FR-072 calls the classes PotionOfHealingConsumed and HealersKitUseSpent so no exported name repeats a result variant.
func specKind(class string) string {
	return map[string]string{"PotionOfHealingConsumed": "potion_of_healing_used", "HealersKitUseSpent": "healers_kit_used"}[class]
}

// snake converts a class name to snake_case (DroppedToZeroHitPoints to dropped_to_zero_hit_points).
func snake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// returnsString reports whether fn has the single result type string: an event Kind, as opposed to the
// Kind of a Condition, which returns a content.ConditionKind (conditions.md FR-002).
func returnsString(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	id, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && id.Name == "string"
}

// Requirement: doc/specs/combat-foundations.md FR-016, FR-017, FR-019, FR-020 and hit-points-and-damage.md FR-103 -
// every combat type that declares Kind() is an event: its Kind returns one string literal matching
// [a-z][a-z0-9_]*, unique in the package and equal to the snake_case of the class name; it embeds the family base
// (no event implements Event directly), has a value receiver, no exported field, and a past-tense name that does
// not end in Required, Needed or a bare Due.
// Path: unhappy (name collision).
func TestCombatEventDeclarations(t *testing.T) {
	t.Parallel()
	valid := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	structs := map[string]*ast.StructType{}
	type kindMethod struct {
		recv, file string
		value      bool
		literal    string
		ok         bool
	}
	var methods []kindMethod
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok {
						if st, ok := ts.Type.(*ast.StructType); ok {
							structs[ts.Name.Name] = st
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || d.Name.Name != "Kind" || !returnsString(d) {
					continue
				}
				m := kindMethod{file: f.Path}
				switch r := d.Recv.List[0].Type.(type) {
				case *ast.Ident:
					m.recv, m.value = r.Name, true
				case *ast.StarExpr:
					m.recv = r.X.(*ast.Ident).Name
				}
				if len(d.Body.List) == 1 {
					if ret, ok := d.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
						if lit, ok := ret.Results[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							m.literal, _ = strconv.Unquote(lit.Value)
							m.ok = true
						}
					}
				}
				methods = append(methods, m)
			}
		}
	}
	if len(methods) < 15 {
		t.Fatalf("found %d Kind methods, want at least the 15 hit point events", len(methods))
	}
	seen := map[string]string{}
	for _, m := range methods {
		// A transport envelope delegates Kind to the wrapped event (encounter-aggregate.md FR-051).
		wrapper := strings.HasSuffix(m.recv, "Occurred")
		switch {
		case wrapper:
			if m.ok {
				t.Errorf("%s.Kind must delegate to the wrapped event, not return a literal", m.recv)
			}
		case !m.ok || !valid.MatchString(m.literal):
			t.Errorf("%s: %s.Kind must return one snake_case string literal", m.file, m.recv)
		case seen[m.literal] != "":
			t.Errorf("%s.Kind = %q is also the Kind of %s", m.recv, m.literal, seen[m.literal])
		case snake(m.recv) != m.literal && specKind(m.recv) != m.literal:
			t.Errorf("%s.Kind = %q, want the snake_case of the class name %q", m.recv, m.literal, snake(m.recv))
		}
		if !wrapper {
			seen[m.literal] = m.recv
		}
		if !m.value {
			t.Errorf("%s.Kind has a pointer receiver (events are values)", m.recv)
		}
		if strings.HasSuffix(m.recv, "Required") || strings.HasSuffix(m.recv, "Needed") || (strings.HasSuffix(m.recv, "Due") && !strings.HasSuffix(m.recv, "BecameDue")) {
			t.Errorf("%s is named like a request, not a fact (FR-020)", m.recv)
		}
		st := structs[m.recv]
		if st == nil {
			t.Errorf("%s is not a struct", m.recv)
			continue
		}
		embedsBase := false
		for _, fld := range st.Fields.List {
			if len(fld.Names) == 0 {
				if id, ok := fld.Type.(*ast.Ident); ok && strings.HasSuffix(id.Name, "EventBase") {
					embedsBase = true
				}
				continue
			}
			for _, n := range fld.Names {
				if n.IsExported() {
					t.Errorf("%s has the exported field %s (FR-019)", m.recv, n.Name)
				}
			}
		}
		if !embedsBase {
			t.Errorf("%s must embed its family's base, not implement Event directly (FR-016)", m.recv)
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-104 and the Architecture note of section 8 - HitPointStatus holds
// no id: no field of any status-like struct in combat has the type CombatantID.
// Path: unhappy (forbidden edge).
func TestHitPointStatusHoldsNoID(t *testing.T) {
	t.Parallel()
	found := false
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || ts.Name.Name != "HitPointStatus" {
				return true
			}
			found = true
			for _, fld := range st.Fields.List {
				if id, ok := fld.Type.(*ast.Ident); ok && id.Name == "CombatantID" {
					t.Errorf("HitPointStatus has a CombatantID field")
				}
			}
			return true
		})
	}
	if !found {
		t.Error("HitPointStatus not found")
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-110, shared-types-and-ownership.md FR-044 - no use case of
// combat calls another: no production file of the package calls a method named Run.
// Path: unhappy (forbidden edge).
func TestCombatUseCasesDoNotCallEachOther(t *testing.T) {
	t.Parallel()
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Run" {
					t.Errorf("%s calls .Run: a use case must not call another", f.Path)
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-083 - package combat exports every use case, command, result
// variant and handoff type of sections 5 and 6 of the spec.
// Path: happy.
func TestCombatExportsHitPointTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ApplyDamage ApplyDamageCommand ApplyDamageResult DamageApplied DamageRejected NewApplyDamage NewApplyDamageCommand
		RestoreHitPoints RestoreHitPointsCommand RestoreHitPointsResult HealingApplied HealingRejected NewRestoreHitPoints NewRestoreHitPointsCommand
		GainTemporaryHitPoints GainTemporaryHitPointsCommand GainTemporaryHitPointsResult TemporaryHitPointsGainApplied TemporaryHitPointsGainRejected NewGainTemporaryHitPoints NewGainTemporaryHitPointsCommand
		EndTemporaryHitPoints EndTemporaryHitPointsCommand EndTemporaryHitPointsResult TemporaryHitPointsEndApplied TemporaryHitPointsEndRejected NewEndTemporaryHitPoints NewEndTemporaryHitPointsCommand
		MakeDeathSavingThrow MakeDeathSavingThrowCommand MakeDeathSavingThrowResult DeathSavingThrowApplied DeathSavingThrowRejected NewMakeDeathSavingThrow NewMakeDeathSavingThrowCommand
		StabilizeCreature StabilizeCreatureCommand StabilizeCreatureResult StabilizationApplied StabilizationRejected NewStabilizeCreature NewStabilizeCreatureCommand
		RollStableRecoveryDelay RollStableRecoveryDelayCommand RollStableRecoveryDelayResult StableRecoveryDelayRolled NewRollStableRecoveryDelay
		RecoverStableCreature RecoverStableCreatureCommand RecoverStableCreatureResult StableRecoveryApplied StableRecoveryRejected NewRecoverStableCreature NewRecoverStableCreatureCommand
		ReduceHitPointMaximum ReduceHitPointMaximumCommand ReduceHitPointMaximumResult HitPointMaximumReductionApplied HitPointMaximumReductionRejected NewReduceHitPointMaximum NewReduceHitPointMaximumCommand
		RecordExhaustionDeath RecordExhaustionDeathCommand RecordExhaustionDeathResult ExhaustionDeathApplied ExhaustionDeathRejected NewRecordExhaustionDeath NewRecordExhaustionDeathCommand
		HitPoints NewHitPoints MustHitPoints HitPointsError HitPointsProblem Damage NewDamage MustDamage DamageAdjustment FlatAdjustment NewFlatAdjustment HalfDamage
		HealingCeiling NoHealingCeiling HealingCeilingAt NewHealingCeilingAt DamageScope AllDamage DamageOfType DamageAffinities NewDamageAffinities NoAffinities
		ResistantToAllDamage ImmuneToAllDamage DamageResolution HitKind NormalHit CriticalHit DamageDelivery MeleeAttackWithoutKnockOut MeleeAttackKnockOut OtherSource
		TemporaryHitPointsChoice KeepCurrent AcceptNew ZeroHitPointsRule DiesInstantly FallsUnconscious DeathSavingThrowOutcome StableRecoveryDelay NewStableRecoveryDelay
		StabilizingCheck HitPointRejection CreatureIsDead NotDying NotStable HitPointTransition HitPointTransitionApplied HitPointTransitionRefused
		HitPointStatus NewHitPointStatus MustHitPointStatus RestoreHitPointStatus HitPointStatusError HitPointStatusProblem LifeState Alive Dying NewDying Stable Dead NewDead DeathCause
		HitPointEvent DamageTaken DroppedToZeroHitPoints FellUnconscious KnockedOut BecameBloodied HitPointsRestored TemporaryHitPointsGained TemporaryHitPointsEnded
		HitPointMaximumReduced HitPointMaximumRestored DeathSavingThrowMade BecameStable LostStability Recovered Died Roller Option WithTelemetry WithClock`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
}
