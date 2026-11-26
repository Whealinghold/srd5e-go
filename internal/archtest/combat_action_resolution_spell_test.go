package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// spellPathUseCaseFiles are the use-case files that step 10b adds to combat-action-resolution.md: the Concentration and Parry plumbing, the
// stat block save use case and the use cases that wrap the pure services of the domain-only specs (FR-167, FR-168).
func spellPathUseCaseFiles() []string {
	return []string{"combat/concentration_usecase.go", "combat/parry_usecase.go", "combat/resolve_stat_block_save_usecase.go", "combat/wired_usecase.go",
		"combat/held_spell_usecase.go", "combat/change_telepathic_contact_usecase.go", "combat/persistent_zone_usecase.go", "combat/shape_shifting_usecase.go",
		"combat/revival_usecase.go"}
}

// Requirement: doc/specs/combat-action-resolution.md FR-006, FR-007, FR-097 and section 8 (FR-158 to FR-170) - no use-case file of the spell path asks ConditionEffects
// or names a use case or service it must not call (RollStatBlockDamage, EnterContainer, StartActiveEffect, AdvanceActiveEffects, EndActiveEffects, MakeDeathSavingThrow,
// ResolveD20Test, BreakConcentration); the Concentration and Parry decisions live in domain files.
// Path: unhappy (forbidden edge).
func TestSpellPathUseCasesCallNoUseCaseAndAskNoConditionRules(t *testing.T) {
	t.Parallel()
	banned := []string{"ConditionEffects", "RollStatBlockDamage", "EnterContainer", "StartActiveEffect", "AdvanceActiveEffects", "EndActiveEffects",
		"MakeDeathSavingThrow", "ResolveD20Test", "NewBreakConcentration", "BreakConcentrationCommand", "BreakConcentrationResult"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(spellPathUseCaseFiles(), f.Path) {
			continue
		}
		seen++
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(banned, id.Name) {
				t.Errorf("%s names %s: a use case composes domain services and calls no use case (FR-006, FR-007)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(spellPathUseCaseFiles()) {
		t.Fatalf("found %d of %d use-case files", seen, len(spellPathUseCaseFiles()))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-167, FR-168, FR-127, section 5 - the requests and results of the wired use cases are closed sets with exactly the variants
// of their owners: the owner's results are the variants of a use case that wraps several of its services, with no wrapper type.
// Path: unhappy (duplicate).
func TestSpellPathClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isTelepathyRequest": {"EndContact", "InitiateContact", "ReplyToContact", "ReportDistance", "ReportIncapacitated"},
		"isChangeTelepathicContactResult": {"EndTelepathicContactApplied", "EndTelepathicContactNoContact", "InitiateTelepathicContactAlreadyMade", "InitiateTelepathicContactMade",
			"InitiateTelepathicContactRejected", "ReportTelepathDistanceBroken", "ReportTelepathDistanceUnchanged", "ReportTelepathIncapacitatedBroken",
			"ReportTelepathIncapacitatedUnchanged", "TelepathicReplyAllowed", "TelepathicReplyNoContact", "TelepathicReplyNotContacted", "TelepathicReplyNotPermitted"},
		"isZoneTriggerRequest": {"ApplyOptionalEffect", "AssessTrigger", "RecordCastApplication"},
		"isAssessPersistentZoneTriggerResult": {"AssessZoneTriggerDue", "AssessZoneTriggerEnded", "AssessZoneTriggerLimitExceeded", "AssessZoneTriggerStaleTurn", "OptionalZoneEffectApplied", "OptionalZoneEffectRefused",
			"RecordCastApplicationLimitExceeded", "RecordCastApplicationNotOncePerTurn", "RecordCastApplicationRecorded", "RecordCastApplicationStaleTurn", "RecordCastApplicationZoneEnded"},
		"isZoneMoveRequest":            {"MoveZoneRequest", "StartOfTurnRequest"},
		"isMovePersistentZoneResult":   {"MoveZoneMoved", "MoveZoneRefused", "MoveZoneStaleTurn", "StartOfTurnReported", "StartOfTurnStaleTurn"},
		"isZoneEndRequest":             {"EndZoneRequest", "RecordDamageRequest", "RoundStartedRequest"},
		"isEndPersistentZoneResult":    {"EndZoneAlreadyEnded", "EndZoneEnded", "RecordDamageNoBudget", "RecordDamageRecorded", "RecordDamageZoneEnded", "ZoneRoundResult"},
		"isDeathWardRequest":           {"WardDamage", "WardInstantKill"},
		"isResolveStatBlockSaveResult": {"ResolveStatBlockSaveCompleted", "ResolveStatBlockSaveInvalid", "ResolveStatBlockSaveRejected"},
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md section 7, FR-099, FR-100, FR-103, FR-106, FR-111, FR-169, FR-170 - signals.go pins the concentration_checks counter, its attribute and four values,
// the twenty-one new operations with their spans and the result values of the new use cases at their catalogue values, and signals.md has a row for each new metric and span, and for the
// attribute and each of its values.
// Path: unhappy (invalid input).
func TestSpellPathSignalNamesAndCatalogue(t *testing.T) {
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
		"metricConcentrationChecks": "srd5e.combat.concentration_checks", "attrConcentrationOutcome": "srd5e.combat.concentration_outcome",
		"valueConcentrationHeld": "held", "valueConcentrationBroken": "broken", "valueConcentrationEndedIncapacitated": "ended_incapacitated", "valueConcentrationEndedDead": "ended_dead",
		"valueResultStatBlockRefused": "stat_block_refused",
	}
	operations := map[string]string{"ResolveStatBlockSaveEffect": "resolve_stat_block_save_effect", "ReadyCombatantSpell": "ready_combatant_spell", "ReleaseCombatantSpell": "release_combatant_spell",
		"DissipateCombatantSpell": "dissipate_combatant_spell", "ChangeTelepathicContact": "change_telepathic_contact", "CreatePersistentZone": "create_persistent_zone",
		"AssessZoneTrigger": "assess_zone_trigger", "MovePersistentZone": "move_persistent_zone", "EndPersistentZone": "end_persistent_zone", "BeginShapeShift": "begin_shape_shift",
		"ChangeShapeShiftForm": "change_shape_shift_form", "AssessShapeShift": "assess_shape_shift", "EndShapeShift": "end_shape_shift", "ReviveBySpell": "revive_by_spell",
		"IncreaseMaximum": "increase_maximum", "EndMaximumIncrease": "end_maximum_increase", "RestoreMaximum": "restore_maximum", "LesserRestoration": "lesser_restoration",
		"GreaterRestoration": "greater_restoration", "RestorativeHealing": "restorative_healing", "GuardWithDeathWard": "guard_with_death_ward"}
	doc := catalogue(t)
	for name, op := range operations {
		want["valueOperation"+name], want["span"+name] = op, "srd5e.combat."+op
		if !strings.Contains(doc, "| `srd5e.combat."+op+"` |") {
			t.Errorf("signals.md has no row for the span srd5e.combat.%s", op)
		}
	}
	for name, value := range map[string]string{"ReadyNotAnAction": "ready_not_an_action", "ReleaseNoHeldSpell": "release_no_held_spell", "ReleaseNotAReadiedReaction": "release_not_a_readied_reaction",
		"ReleaseReactionMismatch": "release_reaction_mismatch", "DissipateNoHeldSpell": "dissipate_no_held_spell", "ContactInitiatorNotTelepathic": "contact_initiator_not_telepathic",
		"ContactInitiatorIncapacitated": "contact_initiator_incapacitated", "ContactContactedIncapacitated": "contact_contacted_incapacitated", "ContactOutOfRange": "contact_out_of_range",
		"ContactCannotUnderstand": "contact_cannot_understand", "ContactAlreadyMade": "contact_already_made", "ContactReplyNotPermitted": "contact_reply_not_permitted",
		"ContactReplyNotContacted": "contact_reply_not_contacted", "ContactNone": "contact_none", "ContactInvalid": "contact_invalid", "ZoneNoFacts": "zone_no_facts",
		"ZoneCreateInvalid": "zone_create_invalid", "ZoneRefused": "zone_refused", "ZoneStaleTurn": "zone_stale_turn", "ZoneMoveRefused": "zone_move_refused",
		"ZoneAlreadyEnded": "zone_already_ended", "ShapeShiftResisted": "shape_shift_resisted", "ShapeShiftEnded": "shape_shift_ended", "ShapeShiftUnaffected": "shape_shift_unaffected",
		"Revived": "revived", "NotRevived": "not_revived", "Ineligible": "ineligible", "Refused": "refused", "Increased": "increased", "Warded": "warded", "NotWarded": "not_warded"} {
		want["valueResult"+name] = value
		if !strings.Contains(doc, "`"+value+"`") {
			t.Errorf("signals.md does not list the result value %s", value)
		}
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	for _, name := range []string{"srd5e.combat.concentration_checks", "srd5e.combat.concentration_outcome", "held", "broken", "ended_incapacitated", "ended_dead", "stat_block_refused"} {
		if !strings.Contains(doc, "`"+name+"`") {
			t.Errorf("signals.md does not mention %s", name)
		}
	}
}
