package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func checkWish(t *testing.T, c character.Character, choice character.WishChoice) character.WishCheck {
	t.Helper()
	got, err := character.CheckWish(c, choice)
	if err != nil {
		t.Fatalf("CheckWish: %v", err)
	}
	return got
}

// wishRoll is the stress roll (2d4, 1d100) of the tables.
func wishRoll(t *testing.T, days, percentile int) character.WishStressRoll {
	t.Helper()
	return must(character.NewWishStressRoll(days, percentile))
}

func levelSpell(t *testing.T, level int, opts ...spellOpt) content.Spell {
	t.Helper()
	return castSpell(t, spellID(level, 5), level, opts...)
}

// Requirement: per case.
// Path: per case.
func TestCheckWishEffectLimits(t *testing.T) {
	t.Parallel()
	c := wishWizard(t)
	dup := func(level int, opts ...spellOpt) character.WishChoice {
		return must(character.NewDuplicateSpell(levelSpell(t, level, opts...)))
	}
	obj := func(v content.Cost, feet float64, magic bool) character.WishChoice {
		return must(character.NewObjectCreation(v, feet, magic))
	}
	health := func(n int) character.WishChoice { return must(character.NewInstantHealth(n)) }
	res := func(n int) character.WishChoice {
		return must(character.NewResistanceGrant(content.DamageFire, wishCreatures(t, n)...))
	}
	imm := func(n int, s character.SpellImmunitySubject) character.WishChoice {
		return must(character.NewSpellImmunityGrant(s, wishCreatures(t, n)...))
	}
	redo := func(m modifier.RollMode, recent bool) character.WishChoice {
		return must(character.NewRollRedo(m, recent))
	}
	sleep := character.ImmuneToSpell{Spell: "sleep"}
	other := character.ImmuneToOtherEffect{Effect: statement(t, "a curse")}
	for _, tc := range []struct {
		name, requirement, path string
		choice                  character.WishChoice
		reject                  character.ResolveWishRejectionKind // 0: accepted
		effect                  character.WishEffect
	}{
		{"E1 duplicate level 8", "FR-002, FR-004", "unhappy (boundary)", dup(8), 0, character.WishDuplicate{}},
		{"E2 duplicate level 9, Wish itself", "FR-002", "unhappy (boundary)", dup(9), character.WishSpellLevelTooHigh, nil},
		{"E3 duplicate a cantrip", "FR-002", "unhappy (boundary)", dup(0), 0, character.WishDuplicate{}},
		{"E4 duplicate a spell on no Wizard list", "FR-003", "unhappy (rule interaction)", must(character.NewDuplicateSpell(spellMap(t, spellSpec{"sacred_flame_x", 5, []string{"cleric"}})["sacred_flame_x"])), 0, character.WishDuplicate{}},
		{"E5 duplicate a Concentration spell", "FR-004", "happy", dup(3, concentrating()), 0, character.WishDuplicate{}},
		{"E6 object of exactly 25,000 GP", "FR-005", "unhappy (boundary)", obj(gpCost(25000), 10, false), 0, character.WishObject{}},
		{"E6 object of 250,000 SP is 25,000 GP", "FR-005", "unhappy (boundary)", obj(content.MustCost(250000, content.CoinSP, 0), 10, false), 0, character.WishObject{}},
		{"E7 object of 25,001 GP", "FR-005", "unhappy (boundary)", obj(gpCost(25001), 10, false), character.WishObjectValueTooHigh, nil},
		{"E7 object of 2,500,001 CP", "FR-005", "unhappy (boundary)", obj(content.MustCost(2500001, content.CoinCP, 0), 10, false), character.WishObjectValueTooHigh, nil},
		{"E8 object that is a magic item", "FR-006", "unhappy (state error)", obj(gpCost(1), 10, true), character.WishObjectIsMagicItem, nil},
		{"E8b value is checked before magic item", "FR-005, FR-006", "unhappy (rule interaction)", obj(gpCost(30000), 10, true), character.WishObjectValueTooHigh, nil},
		{"E9 object of 300 ft", "FR-007", "unhappy (boundary)", obj(gpCost(1), 300, false), 0, character.WishObject{}},
		{"E10 object of 301 ft", "FR-007", "unhappy (boundary)", obj(gpCost(1), 301, false), character.WishObjectTooLarge, nil},
		{"E10b object of 300.5 ft", "FR-007", "unhappy (boundary)", obj(gpCost(1), 300.5, false), character.WishObjectTooLarge, nil},
		{"E10c magic item is checked before size", "FR-006, FR-007", "unhappy (rule interaction)", obj(gpCost(1), 400, true), character.WishObjectIsMagicItem, nil},
		{"E12 health for 20 others", "FR-008", "unhappy (boundary)", health(20), 0, character.WishInstantHealth{}},
		{"E12 health for 0 others", "FR-008", "unhappy (boundary)", health(0), 0, character.WishInstantHealth{}},
		{"E13 health for 21 others", "FR-008", "unhappy (boundary)", health(21), character.WishTooManyCreatures, nil},
		{"E14 resistance for 10", "FR-009", "unhappy (boundary)", res(10), 0, character.WishResistance{}},
		{"E14 resistance for 1", "FR-009", "unhappy (boundary)", res(1), 0, character.WishResistance{}},
		{"E15 resistance for 11", "FR-009", "unhappy (boundary)", res(11), character.WishTooManyCreatures, nil},
		{"E17 immunity to a spell for 10", "FR-010", "happy", imm(10, sleep), 0, character.WishSpellImmunity{}},
		{"E17 immunity to another effect for 10", "FR-010", "happy", imm(10, other), 0, character.WishSpellImmunity{}},
		{"E18 immunity to another effect for 11", "FR-010", "unhappy (boundary)", imm(11, other), character.WishTooManyCreatures, nil},
		{"E18 immunity to a spell for 11", "FR-010", "unhappy (boundary)", imm(11, sleep), character.WishTooManyCreatures, nil},
		{"E24 roll redo, normal", "FR-013", "happy", redo(modifier.Normal, true), 0, character.WishRollRedo{}},
		{"E25 roll redo, advantage", "FR-013", "happy", redo(modifier.WithAdvantage, true), 0, character.WishRollRedo{}},
		{"E26 roll redo, disadvantage", "FR-013", "happy", redo(modifier.WithDisadvantage, true), 0, character.WishRollRedo{}},
		{"E27 roll redo of an old roll", "FR-013", "unhappy (state error)", redo(modifier.Normal, false), character.WishRollNotRecent, nil},
		{"E28 reshape reality", "FR-014", "happy", must(character.NewReshapeReality(statement(t, "A boon"))), 0, character.WishReshapeReality{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checkWish(t, c, tc.choice)
			rej, rejected := got.Rejection()
			if tc.reject != 0 {
				if !rejected || rej.Kind() != tc.reject || got.Effect() != nil || rej.FeatRejection() != nil {
					t.Errorf("= %+v, %v, want rejection %v [%s %s]", rej, got.Effect(), tc.reject, wishSpec, tc.requirement)
				}
				return
			}
			if rejected || reflect.TypeOf(got.Effect()) != reflect.TypeOf(tc.effect) {
				t.Errorf("= %v, %T, want %T [%s %s]", rej.Kind(), got.Effect(), tc.effect, wishSpec, tc.requirement)
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-004, FR-003, E1, E5 - an accepted duplication hands off the spell, its level and whether it needs
// Concentration, and the Character is unchanged (no stress, no dice).
// Path: happy.
func TestApplyWishDuplicatesWithoutStress(t *testing.T) {
	t.Parallel()
	c := wishWizard(t)
	for _, tc := range []struct {
		level int
		conc  bool
	}{{8, false}, {0, false}, {3, true}} {
		var opts []spellOpt
		if tc.conc {
			opts = append(opts, concentrating())
		}
		sp := levelSpell(t, tc.level, opts...)
		app, err := character.ApplyWish(c, must(character.NewDuplicateSpell(sp)), character.WishStressRoll{})
		if err != nil {
			t.Fatal(err)
		}
		eff := app.Effect().(character.WishDuplicate)
		ev := app.Events()
		if eff.Spell() != sp.ID() || eff.Level() != content.SpellLevel(tc.level) || eff.RequiresConcentration() != tc.conc || !app.Character().Equal(c) || app.Feats().Len() != 0 {
			t.Errorf("level %d: effect %+v, character equal %v", tc.level, eff, app.Character().Equal(c))
		}
		if len(ev) != 1 || ev[0].(character.WishDuplicated).Spell() != sp.ID() || ev[0].(character.WishDuplicated).Level() != content.SpellLevel(tc.level) || ev[0].CharacterID() != caster {
			t.Errorf("level %d: events = %#v", tc.level, ev)
		}
	}
	if character.WishRequiresStress(must(character.NewDuplicateSpell(levelSpell(t, 1)))) || !character.WishRequiresStress(must(character.NewInstantHealth(1))) {
		t.Error("only a duplication is free of stress")
	}
}

// Requirement: per case.
// Path: per case.
func TestApplyWishIncursTheStress(t *testing.T) {
	t.Parallel()
	choice := must(character.NewReshapeReality(statement(t, "A boon")))
	for _, tc := range []struct {
		name, requirement, path string
		prior                   character.Character
		days, percentile        int
		want                    character.WishStress
	}{
		{"S1 shortest, 34 safe", "FR-022, FR-023, FR-024, FR-026", "unhappy (boundary)", wishWizard(t), 2, 34, wishStress(t, 2, true, false)},
		{"S2 longest, 33 locks", "FR-022, FR-025, FR-026", "unhappy (boundary)", wishWizard(t), 8, 33, wishStress(t, 8, true, true)},
		{"S5 weakened 6, newest roll 3 governs", "FR-023", "unhappy (rule interaction)", stressed(t, wishWizard(t), 6, true, false), 3, 50, wishStress(t, 3, true, false)},
		{"S6 weakened 2, newest roll 7 governs", "FR-023", "unhappy (rule interaction)", stressed(t, wishWizard(t), 2, true, false), 7, 50, wishStress(t, 7, true, false)},
		{"S7 the lockout survives", "FR-025", "unhappy (state error)", stressed(t, wishWizard(t), 0, false, false), 5, 100, wishStress(t, 5, true, false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			before := tc.prior
			app, err := character.ApplyWish(tc.prior, choice, wishRoll(t, tc.days, tc.percentile))
			if err != nil {
				t.Fatal(err)
			}
			ev := app.Events()
			end, ok := ev[len(ev)-1].(character.WishStressEndured)
			if got := stressOfChar(t, app.Character()); got != tc.want || len(ev) != 1 || !ok ||
				end.Days() != tc.days || end.Percentile() != tc.percentile || end.LockedOut() != (tc.percentile <= 33) || end.CharacterID() != caster {
				t.Errorf("stress %+v, events %#v, want %+v [%s %s]", got, ev, tc.want, wishSpec, tc.requirement)
			}
			if !tc.prior.Equal(before) || app.Character().ID() != caster || app.Effect().(character.WishReshapeReality).Statement().String() != "A boon" {
				t.Error("the input changed or the effect is wrong")
			}
		})
	}
}

// Requirement: doc/specs/wish-spell.md FR-025, S7 - a Character already locked out cannot cast Wish, so no choice reaches the stress; the
// lockout state is never cleared by IncurWishStress with a high percentile.
// Path: unhappy (state error).
func TestApplyWishKeepsTheLockoutAcrossStress(t *testing.T) {
	t.Parallel()
	got := character.IncurWishStress(wishStress(t, 2, true, true), wishRoll(t, 4, 90))
	if got.CanCastWish() {
		t.Error("a later stress cleared the lockout")
	}
}

// Requirement: doc/specs/wish-spell.md FR-031, E33 - a creature that can never cast Wish again has every choice rejected first, a duplication included.
// Path: unhappy (state error).
func TestCheckWishLockoutIsCheckedFirst(t *testing.T) {
	t.Parallel()
	c := stressed(t, wishWizard(t), 0, false, true)
	for _, choice := range []character.WishChoice{
		must(character.NewDuplicateSpell(levelSpell(t, 1))),
		must(character.NewDuplicateSpell(levelSpell(t, 9))),
		must(character.NewObjectCreation(gpCost(99999), 999, true)),
		must(character.NewRollRedo(modifier.Normal, false)),
		must(character.NewReshapeReality(statement(t, "x"))),
	} {
		rej, ok := checkWish(t, c, choice).Rejection()
		if !ok || rej.Kind() != character.WishCannotCastAgain {
			t.Errorf("%T: rejection = %v, %v", choice, rej.Kind(), ok)
		}
	}
}

// suddenLearning builds the choice replacing alert by newFeat.
func suddenLearning(t *testing.T, feats character.TakenFeats, old string, newFeat content.Feat, requiredBy []content.FeatID, held character.HeldFeatures) character.SuddenLearning {
	t.Helper()
	return must(character.NewSuddenLearning(content.FeatID(old), newFeat, character.NoFeatChoices{}, requiredBy, held, miSpells(t), feats))
}

// Requirement: doc/specs/wish-spell.md FR-011, FR-012, FR-026, E19 to E23, S11 (SRD 5.2.1, "Wish", Sudden Learning, p. 175) - Sudden Learning replaces
// a held feat through the FeatReplacement service with the feat source of the replaced feat, checks "held" then "required by another"
// then the service, nests the service's FeatRejection unchanged, and reports FeatReplaced, FeatTaken, then WishStressEndured.
// Path: per case.
func TestSuddenLearning(t *testing.T) {
	t.Parallel()
	c := wishWizard(t)
	feats := heldFeats(t, alertFeat(), character.NoFeatChoices{})
	asi := must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	asiHeld := heldFeats(t, asiFeat(), asi)
	for _, tc := range []struct {
		name, requirement, path string
		choice                  character.SuddenLearning
		reject                  character.ResolveWishRejectionKind
		feat                    character.FeatRejection
	}{
		{"E19 alert replaced by savage attacker", "FR-011", "happy", suddenLearning(t, feats, "alert", savageFeat(), nil, character.HeldFeatures{}), 0, nil},
		{"E20 the old feat is not held", "FR-011, FR-021", "unhappy (state error)", suddenLearning(t, feats, "savage_attacker", alertFeat(), nil, character.HeldFeatures{}),
			character.WishFeatReplacementRejected, character.FeatNotHeld{Feat: "savage_attacker"}},
		{"E21 the old feat gave an ability score increase", "FR-011", "unhappy (rule interaction)", suddenLearning(t, asiHeld, "ability_score_improvement", savageFeat(), nil, character.HeldFeatures{}),
			character.WishFeatReplacementRejected, character.FeatNotReplaceable{Feat: "ability_score_improvement"}},
		{"E22 the old feat is required by another", "FR-012", "unhappy (state error)", suddenLearning(t, feats, "alert", savageFeat(), []content.FeatID{"alert"}, character.HeldFeatures{}),
			character.WishFeatRequiredByAnother, nil},
		{"E23 the new feat's prerequisite is unmet", "FR-011", "unhappy (invalid input)", suddenLearning(t, feats, "alert", archeryFeat(), nil, character.HeldFeatures{}),
			character.WishFeatReplacementRejected, nil},
		{"the new feat is already held", "FR-011", "unhappy (rule interaction)", suddenLearning(t, heldFeats(t, alertFeat(), character.NoFeatChoices{}, savageFeat(), character.NoFeatChoices{}), "alert", savageFeat(), nil, character.HeldFeatures{}),
			character.WishFeatReplacementRejected, character.FeatNotRepeatable{Feat: "savage_attacker"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checkWish(t, c, tc.choice)
			rej, rejected := got.Rejection()
			if tc.reject != 0 {
				if !rejected || rej.Kind() != tc.reject || got.Effect() != nil {
					t.Fatalf("= %v, %v, want %v [%s %s]", rej.Kind(), rejected, tc.reject, wishSpec, tc.requirement)
				}
				if _, unmet := rej.FeatRejection().(character.FeatPrerequisiteNotMet); tc.feat == nil && tc.reject == character.WishFeatReplacementRejected && !unmet {
					t.Errorf("nested = %#v, want FeatPrerequisiteNotMet", rej.FeatRejection())
				}
				if tc.feat != nil && rej.FeatRejection() != tc.feat {
					t.Errorf("nested = %#v, want %#v", rej.FeatRejection(), tc.feat)
				}
				return
			}
			if rejected || got.Effect() != (character.WishSuddenLearning{}) && got.Effect().(character.WishSuddenLearning).New() != "savage_attacker" {
				t.Fatalf("= %v, %+v", rej.Kind(), got.Effect())
			}
			app, err := character.ApplyWish(c, tc.choice, wishRoll(t, 4, 90))
			if err != nil {
				t.Fatal(err)
			}
			ev := app.Events()
			if len(ev) != 3 || ev[0].(character.FeatReplaced).Old() != "alert" || ev[1].(character.FeatTaken).Feat() != "savage_attacker" || ev[2].(character.WishStressEndured).Days() != 4 {
				t.Fatalf("events = %#v, want FeatReplaced, FeatTaken, WishStressEndured", ev)
			}
			taken := app.Feats().All()
			if len(taken) != 1 || taken[0].Feat() != "savage_attacker" || taken[0].Source() != (character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_asi"}) {
				t.Errorf("feats = %+v, want the new feat under the old feat's source", taken)
			}
			if got := app.Effect().(character.WishSuddenLearning); got.Old() != "alert" || got.New() != "savage_attacker" || !feats.Holds("alert") {
				t.Errorf("effect %+v or the input feats changed", got)
			}
		})
	}
	both := suddenLearning(t, feats, "savage_attacker", alertFeat(), nil, character.HeldFeatures{})
	if rej, _ := checkWish(t, c, both).Rejection(); rej.Kind() != character.WishFeatReplacementRejected {
		t.Errorf("E22b: %v", rej.Kind())
	}
}

// Requirement: doc/specs/wish-spell.md FR-012, E22b - the old feat not held is reported before it is required by another.
// Path: unhappy (state error).
func TestSuddenLearningHeldIsCheckedBeforeRequiredByAnother(t *testing.T) {
	t.Parallel()
	feats := heldFeats(t, alertFeat(), character.NoFeatChoices{}, savageFeat(), character.NoFeatChoices{})
	// "savage_attacker" is held, "initiate" is not: required-by lists only held feats, so the held check is reached with an unheld old id.
	sl := suddenLearning(t, feats, "initiate", alertFeat(), nil, character.HeldFeatures{})
	rej, _ := checkWish(t, wishWizard(t), sl).Rejection()
	if nf, ok := rej.FeatRejection().(character.FeatNotHeld); rej.Kind() != character.WishFeatReplacementRejected || !ok || nf.Feat != "initiate" {
		t.Errorf("= %v %#v", rej.Kind(), rej.FeatRejection())
	}
}

// Requirement: doc/specs/wish-spell.md FR-021, FR-068, E32 - CheckWish and ApplyWish return an error for a zero Character, a nil choice, a zero
// variant and a stress state of another type; ApplyWish also for a choice the rules refuse and for a missing stress roll.
// Path: per case.
func TestWishChecksInvalidInput(t *testing.T) {
	t.Parallel()
	c := wishWizard(t)
	foreign := must(character.WithFeatureState(c, character.MustFeatureStateKey[foreignState]("wish_stress"), foreignState{}))
	reshape := must(character.NewReshapeReality(statement(t, "x")))
	for _, tc := range []struct {
		name   string
		c      character.Character
		choice character.WishChoice
	}{
		{"zero character", character.Character{}, reshape},
		{"nil choice", c, nil},
		{"zero object creation", c, character.ObjectCreation{}},
		{"zero sudden learning", c, character.SuddenLearning{}},
		{"stress state of another type", foreign, reshape},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := character.CheckWish(tc.c, tc.choice); !isInvalid(err) {
				t.Errorf("CheckWish = %v", err)
			}
			if _, err := character.ApplyWish(tc.c, tc.choice, wishRoll(t, 3, 50)); !isInvalid(err) {
				t.Errorf("ApplyWish = %v", err)
			}
		})
	}
	if _, err := character.ApplyWish(c, must(character.NewRollRedo(modifier.Normal, false)), wishRoll(t, 3, 50)); !isInvalid(err) {
		t.Errorf("ApplyWish of a refused choice = %v", err)
	}
	if _, err := character.ApplyWish(c, reshape, character.WishStressRoll{}); !isInvalid(err) {
		t.Errorf("ApplyWish without a roll = %v", err)
	}
}

type foreignState struct{}

func (foreignState) Equal(o character.FeatureState) bool { _, ok := o.(foreignState); return ok }

// Requirement: doc/specs/wish-spell.md FR-065 - the events of an application are a copy.
// Path: unhappy (immutability).
func TestWishApplicationEventsAreACopy(t *testing.T) {
	t.Parallel()
	app, err := character.ApplyWish(wishWizard(t), must(character.NewReshapeReality(statement(t, "x"))), wishRoll(t, 3, 50))
	if err != nil {
		t.Fatal(err)
	}
	app.Events()[0] = nil
	if app.Events()[0] == nil {
		t.Error("Events aliases the application")
	}
}

// Requirement: per case.
// Path: per case.
func TestWishLevelOfCastAndDamageDice(t *testing.T) {
	t.Parallel()
	slot3, sp1 := slotCast(t, 1, 3)
	slot9, sp9 := slotCast(t, 9, 9)
	ritual, spRitual := wishCastOf(t, castCase{spell: castSpell(t, ritualInBook, 1, asRitual()), mode: character.CastModeRitual})
	cantrip, spCantrip := wishCastOf(t, castCase{spell: castSpell(t, spellID(0, 1), 0)})
	feature, spFeature := wishCastOf(t, castCase{spell: castSpell(t, spellID(5, 1), 5), mode: character.CastModeFeatureGranted, featureLevel: 5})
	for _, tc := range []struct {
		name, requirement, path string
		cast                    character.SpellCast
		spell                   content.Spell
		want                    content.SpellLevel
	}{
		{"D2 level 1 spell with a level 3 slot", "FR-040", "unhappy (rule interaction)", slot3, sp1, 3},
		{"D3 level 9 slot", "FR-041", "unhappy (boundary)", slot9, sp9, 9},
		{"D4 Ritual of a level 1 spell uses the spell's level", "FR-040", "unhappy (rule interaction)", ritual, spRitual, 1},
		{"D5 feature-granted level 5 spell", "FR-040", "unhappy (rule interaction)", feature, spFeature, 5},
		{"D6 cantrip", "FR-042", "unhappy (boundary)", cantrip, spCantrip, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := character.WishLevelOfCast(tc.cast, tc.spell); got != tc.want {
				t.Errorf("= %v, want %v [%s %s]", got, tc.want, wishSpec, tc.requirement)
			}
		})
	}
	for level := content.SpellLevel(1); level <= 9; level++ {
		e, err := character.WishDamageDice(level)
		if err != nil || e != dice.MustExpression(int(level), dice.D10(), 0) {
			t.Errorf("WishDamageDice(%d) = %v, %v", level, e, err)
		}
	}
	for _, level := range []content.SpellLevel{0, -1, 10} {
		if _, err := character.WishDamageDice(level); !isInvalid(err) {
			t.Errorf("WishDamageDice(%d) = %v, want invalid input [FR-041]", level, err)
		}
	}
}

// Requirement: doc/specs/wish-spell.md FR-001, FR-061, FR-067 - the rejection kinds, damage reasons and cast origins have snake_case names and an
// unknown value reads "unknown"; the zero kind is invalid.
// Path: happy.
func TestWishEnumNames(t *testing.T) {
	t.Parallel()
	kinds := []string{"cannot_cast_wish_again", "spell_level_too_high", "object_value_too_high", "object_is_magic_item", "object_too_large", "too_many_creatures",
		"roll_not_recent", "feat_required_by_another", "feat_replacement_rejected"}
	for i, want := range kinds {
		if k := character.ResolveWishRejectionKind(i + 1); k.String() != want || !k.Valid() {
			t.Errorf("kind %d = %q, %v", i+1, k.String(), k.Valid())
		}
	}
	for _, k := range []character.ResolveWishRejectionKind{0, 10, -1} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("kind %d = %q, %v", k, k.String(), k.Valid())
		}
	}
	if character.WishOrdinaryCast.String() != "ordinary_cast" || character.WishDuplicateCast.String() != "wish_duplicate" || character.WishCastOrigin(9).String() != "unknown" {
		t.Error("cast origin names")
	}
	for r, want := range map[character.WishDamageNoneReason]string{character.WishNoDamageCantrip: "cantrip", character.WishNoDamageNotActive: "not_active",
		character.WishNoDamageDuplicatedSpell: "duplicated_spell", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("reason %d = %q, want %q", r, r.String(), want)
		}
	}
}
