package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// mscSingle is a rig whose trait lists one entry, at reference (0,0), with frequency f.
func mscSingle(tb testing.TB, f content.SpellFrequency, entry content.MonsterSpellEntry, edit func(*content.MonsterSpellcastingFields)) mscRig {
	tb.Helper()
	return mscNewRig(tb, mscMonster(tb, func(t *content.MonsterSpellcastingFields) {
		t.Groups = []content.MonsterSpellGroup{mscGroup(f, entry)}
		if edit != nil {
			edit(t)
		}
	}))
}

var mscFirst = [2]int{0, 0}

// mscUnchanged fails the test when the rig's inputs differ from before: a refusal or an error changes nothing.
func mscUnchanged(tb testing.TB, before, after mscRig) {
	tb.Helper()
	if before.state != after.state || !before.uses.Equal(after.uses) || !before.caster.Equal(after.caster) {
		tb.Errorf("a state changed")
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-003, FR-004, FR-006, FR-007, CL-1 - a level 1+ spell is cast at the entry's stated version, else at its own level, never lower; a cantrip has no cast-at level and scaling level 1; the scaling level equals the cast level (SRD 5.2.1, "Spellcasting", p. 257; "Archmage", p. 305).
// Path: per case.
func TestCastLevels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		ref                     [2]int
		spell                   string
		castAt, scaling         int
		wantErr                 bool
	}{
		{"lowest level", "FR-003 CL-1", "happy", mscGuardians, "spirit_guardians", 3, 3, false},
		{"stated version", "FR-003 CL-1", "happy", mscLightningBolt7, "lightning_bolt", 7, 7, false},
		{"version equal to level", "FR-003 CL-1", "unhappy (boundary)", mscFireball3, "fireball", 3, 3, false},
		{"version below level", "FR-004 CL-1", "unhappy (boundary)", mscFireball2, "fireball", 0, 0, true},
		{"version on a cantrip", "FR-004 CL-1", "unhappy (invalid input)", mscLightOne, "light", 0, 0, true},
		{"cantrip", "FR-006 CL-1", "unhappy (boundary)", mscLight, "light", 0, 1, false},
		{"top version", "FR-003 CL-1", "unhappy (boundary)", mscConeOfCold9, "cone_of_cold", 9, 9, false},
	}
	spells := mscSpells()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig := mscNewRig(t, mscMonster(t, nil))
			res := rig.cast(mscList(t, c.ref), spells[c.spell])
			if c.wantErr {
				if !errors.Is(res.Err, srderr.ErrInvalidInput) {
					t.Fatalf("Cast() error = %v, want ErrInvalidInput [%s]", res.Err, c.requirement)
				}
				return
			}
			rep := mscPerformed(t, res).Report
			at, has := rep.CastAtLevel()
			if at != c.castAt || has != (c.castAt != 0) || rep.ScalingLevel() != c.scaling {
				t.Errorf("castAt, scaling = %d (%v), %d, want %d, %d [%s]", at, has, rep.ScalingLevel(), c.castAt, c.scaling, monsterSpellcastingSpec+c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-005 - no input raises the level: a part cast without a stated version casts at the spell's level, one with a version casts at it, a version below the level is refused as an error and a version above 9 or below 0 is not a request (SRD 5.2.1, "Spellcasting", p. 257).
// Path: per case.
func TestPartCastLevels(t *testing.T) {
	t.Parallel()
	rig := mscNewRig(t, mscMonster(t, nil))
	spells := mscSpells()
	for _, c := range []struct {
		name, requirement, path string
		castAt, want            int
		wantErr                 bool
	}{
		{"no stated version casts at the spell level", "FR-005", "happy", 0, 2, false},
		{"a version equal to the level", "FR-005", "unhappy (boundary)", 2, 2, false},
		{"a version above the level", "FR-005", "happy", 6, 6, false},
		{"a version below the level is refused", "FR-005", "unhappy (invalid input)", 1, 0, true},
	} {
		res := rig.cast(mscPartCast(t, mscMistyStep, c.castAt, combat.PartSpellcasting{}), spells["misty_step"])
		if c.wantErr {
			if !errors.Is(res.Err, srderr.ErrInvalidInput) {
				t.Errorf("%s: stated level %d: error = %v, want ErrInvalidInput [%s]", c.name, c.castAt, res.Err, c.requirement)
			}
			continue
		}
		if at, _ := mscPerformed(t, res).Report.CastAtLevel(); at != c.want {
			t.Errorf("%s: stated level %d: cast at %d, want %d [%s]", c.name, c.castAt, at, c.want, c.requirement)
		}
	}
	for _, bad := range []int{-1, 10} {
		if _, err := combat.NewMonsterPartCast(mscPart(t, mscMistyStep), bad, combat.PartSpellcasting{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewMonsterPartCast(level %d) error = %v, want ErrInvalidInput", bad, err)
		}
	}
	for _, ok := range []int{1, 9} {
		if _, err := combat.NewMonsterPartCast(mscPart(t, mscMistyStep), ok, combat.PartSpellcasting{}); err != nil {
			t.Errorf("NewMonsterPartCast(level %d) error = %v, want none", ok, err)
		}
	}
	if _, err := combat.NewMonsterPartCast(combat.LimitedUsePartRef{}, 0, combat.PartSpellcasting{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewMonsterPartCast(zero part) error = %v, want ErrInvalidInput", err)
	}
	pc := mscPartCast(t, mscMistyStep, 4, combat.PartSpellcasting{})
	if lv, ok := pc.CastAtLevel(); lv != 4 || !ok || pc.Part() != mscPart(t, mscMistyStep) {
		t.Errorf("PartCast accessors = %d %v %v", lv, ok, pc.Part())
	}
	if _, ok := pc.Spellcasting(); ok {
		t.Errorf("PartCast.Spellcasting() reports values for none")
	}
	if _, ok := mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}).CastAtLevel(); ok {
		t.Errorf("CastAtLevel() reports a level for none")
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-008 - a monster cast expends no spell slot and is not limited by One Spell with a Spell Slot per Turn: two casts in one turn are both performed and the MonsterCastReport has no slot field (SRD 5.2.1, "One Spell with a Spell Slot per Turn", p. 105; "Pit Fiend", p. 315).
// Path: unhappy (rule interaction).
func TestTwoCastsInOneTurn(t *testing.T) {
	t.Parallel()
	rig := mscNewRig(t, mscMonster(t, nil))
	spells := mscSpells()
	res := rig.cast(mscList(t, mscFireball3), spells["fireball"])
	first := mscPerformed(t, res)
	rig = rig.then(t, res.Out)
	second := mscPerformed(t, rig.cast(mscList(t, mscFireball3), spells["fireball"]))
	for i, p := range []combat.MonsterCastPerformed{first, second} {
		if p.Report.Spell() != "fireball" {
			t.Errorf("cast %d reports spell %v, want fireball", i, p.Report.Spell())
		}
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-009, FR-010, FR-011, FR-012, PN-1 - the report carries the printed save DC and spell attack bonus as printed (never recomputed), the ability and its Modifier; a cast needing a number the trait does not print is refused, an unknown or false fact is not (SRD 5.2.1, "Spellcasting", p. 257; "Archmage", p. 305; "Priest Acolyte", p. 316).
// Path: per case.
func TestPrintedNumbers(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	spells["save_and_attack"] = mscSaveAndAttack()
	spells["magic_missile"] = seMagicMissile()
	noNumbers := func(f *content.MonsterSpellcastingFields) { f.HasSaveDC, f.HasAttackBonus = false, false }
	dcOnly := func(f *content.MonsterSpellcastingFields) { f.HasAttackBonus = false }
	cases := []struct {
		name, requirement, path string
		spell                   string
		edit                    func(*content.MonsterSpellcastingFields)
		wantDC                  int // 0: none printed
		wantBonus               int
		hasBonus                bool
		refusal                 combat.MonsterCastRefusal
	}{
		{"DC as printed, not the formula", "FR-009 PN-1", "unhappy (rule interaction)", "fireball", nil, 14, 6, true, 0},
		{"archmage DC", "FR-009 PN-1", "happy", "fireball", func(f *content.MonsterSpellcastingFields) { f.SaveDC = 17 }, 17, 6, true, 0},
		{"attack bonus is the printed sum", "FR-010 PN-1", "happy", "fire_bolt", func(f *content.MonsterSpellcastingFields) { f.AttackBonus = 10 }, 14, 10, true, 0},
		{"a negative printed bonus", "FR-010", "unhappy (boundary)", "fire_bolt", func(f *content.MonsterSpellcastingFields) { f.AttackBonus = -2 }, 14, -2, true, 0},
		{"no numbers, no need", "FR-012 PN-1", "unhappy (empty and minimal)", "light", noNumbers, 0, 0, false, 0},
		{"DC absent, save spell", "FR-012 PN-1", "unhappy (missing content)", "fireball", noNumbers, 0, 0, false, combat.MonsterCastRefusalMissingSaveDC},
		{"bonus absent, attack spell", "FR-012 PN-1", "unhappy (missing content)", "fire_bolt", dcOnly, 14, 0, false, combat.MonsterCastRefusalMissingSpellAttackBonus},
		{"text-only spell, DC absent", "FR-012 PN-1", "unhappy (boundary)", "text_only_save", noNumbers, 0, 0, false, 0},
		{"a false fact is not refused", "FR-012", "unhappy (boundary)", "magic_missile", noNumbers, 0, 0, false, 0},
		{"both missing, the DC is reported first", "FR-031", "unhappy (rule interaction)", "save_and_attack", noNumbers, 0, 0, false, combat.MonsterCastRefusalMissingSaveDC},
		{"only the bonus missing for both", "FR-031", "unhappy (rule interaction)", "save_and_attack", dcOnly, 14, 0, false, combat.MonsterCastRefusalMissingSpellAttackBonus},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			spell := spells[c.spell]
			rig := mscSingle(t, content.SpellFrequencyAtWill{}, mscEntry(spell.Name(), "", 0), c.edit)
			before := rig
			res := rig.cast(mscList(t, mscFirst), spell)
			mscUnchanged(t, before, rig)
			if c.refusal != 0 {
				if got := mscRefused(t, res); got != c.refusal {
					t.Errorf("refusal = %v, want %v [%s]", got, c.refusal, monsterSpellcastingSpec+c.requirement)
				}
				return
			}
			rep := mscPerformed(t, res).Report
			dc, hasDC := rep.SaveDC()
			if int(dc) != c.wantDC || hasDC != (c.wantDC != 0) {
				t.Errorf("SaveDC() = %v, %v, want %d [%s]", dc, hasDC, c.wantDC, c.requirement)
			}
			mods, hasMods := rep.AttackModifiers()
			if hasMods != c.hasBonus {
				t.Fatalf("AttackModifiers() present = %v, want %v", hasMods, c.hasBonus)
			}
			if hasMods {
				_, pb := mods.ProficiencyBonus()
				if mods.Ability().Value() != c.wantBonus || pb || mods.Count() != 1 {
					t.Errorf("AttackModifiers() = ability %d, proficiency %v, count %d, want one slot of %d [%s]", mods.Ability().Value(), pb, mods.Count(), c.wantBonus, c.requirement)
				}
			}
			if rep.Ability() != ability.Intelligence || rep.AbilityModifier().Value() != 3 {
				t.Errorf("Ability() = %v, AbilityModifier() = %v, want intelligence +3 [FR-011]", rep.Ability(), rep.AbilityModifier())
			}
		})
	}
}

// mscSaveAndAttack is a spell that forces a saving throw and makes a spell attack roll.
func mscSaveAndAttack() content.Spell {
	return seSpell("save_and_attack", 2, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("2d6"), nil), seFire, content.AddsNothing)}, content.HalfDamageOnSuccess), seInstances(1, nil)),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d10"), nil), seFire, content.AddsNothing)}, nil), seInstances(1, nil))), nil)
}

// Requirement: doc/specs/monster-spellcasting.md FR-014, FR-015 - a self only spell is refused when the targets hold any creature other than the caster; no targets, the caster alone and a duplicate caster are accepted; a spell without self only is not limited, and the report says which (SRD 5.2.1, "Spellcasting", p. 257; "Green Hag", p. 295).
// Path: per case.
func TestSelfOnlyTargets(t *testing.T) {
	t.Parallel()
	other := combat.MustCombatantID("ally")
	cases := []struct {
		name, requirement, path string
		ref                     [2]int
		spell                   string
		targets                 []combat.CombatantID
		refused                 bool
		limit                   combat.TargetLimit
	}{
		{"no targets", "FR-014", "unhappy (empty and minimal)", mscInvisibility, "invisibility", nil, false, combat.TargetLimitSelfOnly},
		{"the caster alone", "FR-014", "happy", mscInvisibility, "invisibility", []combat.CombatantID{mscID()}, false, combat.TargetLimitSelfOnly},
		{"the caster twice", "FR-014", "unhappy (boundary)", mscInvisibility, "invisibility", []combat.CombatantID{mscID(), mscID()}, false, combat.TargetLimitSelfOnly},
		{"the caster and another", "FR-014", "unhappy (rule interaction)", mscInvisibility, "invisibility", []combat.CombatantID{mscID(), other}, true, 0},
		{"another alone", "FR-014", "unhappy (rule interaction)", mscInvisibility, "invisibility", []combat.CombatantID{other}, true, 0},
		{"another that sorts after the caster", "FR-014", "unhappy (rule interaction)", mscInvisibility, "invisibility", []combat.CombatantID{combat.MustCombatantID("zed")}, true, 0},
		{"a spell without self only", "FR-015", "unhappy (rule interaction)", mscFireball, "fireball", []combat.CombatantID{mscID(), other}, false, combat.TargetLimitUnrestricted},
	}
	spells := mscSpells()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig := mscNewRig(t, mscMonster(t, nil))
			res := rig.cast(mscList(t, c.ref), spells[c.spell], c.targets...)
			if c.refused {
				if got := mscRefused(t, res); got != combat.MonsterCastRefusalSelfOnlyViolated {
					t.Errorf("refusal = %v, want self_only_violated [%s]", got, c.requirement)
				}
				return
			}
			if got := mscPerformed(t, res).Report.TargetLimit(); got != c.limit {
				t.Errorf("TargetLimit() = %v, want %v [%s]", got, c.limit, c.requirement)
			}
		})
	}
	rep := mscPerformed(t, mscNewRig(t, mscMonster(t, nil)).cast(mscList(t, mscInvisibility), spells["invisibility"])).Report
	if rep.RestrictionNote() != "self only, and the hag leaves no tracks while Invisible" {
		t.Errorf("RestrictionNote() = %q, want the printed note unchanged", rep.RestrictionNote())
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-016, FR-029, LC-1 - a no-Concentration entry neither begins Concentration nor ends one in progress; any other Concentration spell ends Concentration in progress (another effect) and begins Concentration of kind effect, events in the order ended, began, cast (SRD 5.2.1, "Concentration", p. 179; "Adult Brass Dragon", p. 269).
// Path: per case.
func TestImmediateConcentration(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	cases := []struct {
		name, requirement, path string
		ref                     [2]int
		spell                   string
		holding                 bool
		wantEvents              []string
		wantHolding             string
	}{
		{"concentration spell, holding another", "FR-029 LC-1", "happy", mscGuardians, "spirit_guardians", true,
			[]string{"monster_concentration_ended", "monster_concentration_began", "monster_spell_cast"}, "spirit_guardians"},
		{"concentration spell, holding nothing", "FR-029", "unhappy (empty and minimal)", mscGuardians, "spirit_guardians", false,
			[]string{"monster_concentration_began", "monster_spell_cast"}, "spirit_guardians"},
		{"waived, holding another", "FR-016 LC-1", "unhappy (rule interaction)", mscShapechange, "shapechange", true,
			[]string{"monster_spell_cast"}, "other_spell"},
		{"waived, holding nothing", "FR-016", "unhappy (empty and minimal)", mscShapechange, "shapechange", false,
			[]string{"monster_spell_cast"}, ""},
		{"spell without Concentration, holding another", "FR-029", "unhappy (rule interaction)", mscMindBlank, "mind_blank", true,
			[]string{"monster_spell_cast"}, "other_spell"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig := mscNewRig(t, mscMonster(t, nil))
			if c.holding {
				rig.state = mscConcentrating(t, "other_spell")
			}
			p := mscPerformed(t, rig.cast(mscList(t, c.ref), spells[c.spell]))
			if got := mscKinds(p.Transition.Events()); !equalStrings(got, c.wantEvents) {
				t.Errorf("events = %v, want %v [%s]", got, c.wantEvents, monsterSpellcastingSpec+c.requirement)
			}
			held, ok := p.Transition.State().Concentration()
			subject := ""
			if ok {
				subject = string(held.Subject().(character.ConcentrationSubjectSpell).Spell())
			}
			if subject != c.wantHolding || (ok && held.Kind() != character.ConcentrationEffect && c.wantHolding == "spirit_guardians") {
				t.Errorf("Concentration = %q (%v), want %q [%s]", subject, held.Kind(), c.wantHolding, c.requirement)
			}
		})
	}
	rig := mscNewRig(t, mscMonster(t, nil))
	rig.state = mscConcentrating(t, "other_spell")
	p := mscPerformed(t, rig.cast(mscList(t, mscGuardians), spells["spirit_guardians"]))
	ended := p.Transition.Events()[0].(combat.MonsterConcentrationEnded)
	began := p.Transition.Events()[1].(combat.MonsterConcentrationBegan)
	if ended.Reason() != character.EndedAnotherEffect || began.ConcentrationKind() != character.ConcentrationEffect ||
		began.Subject() != character.NewConcentrationSubjectSpell("spirit_guardians") || ended.Subject() != character.NewConcentrationSubjectSpell("other_spell") {
		t.Errorf("ended %v/%v, began %v/%v: wrong subjects, reason or kind", ended.Subject(), ended.Reason(), began.Subject(), began.ConcentrationKind())
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-018, FR-019, FR-017, CS-1 - a Verbal or Somatic component the statement does not waive needs a monster that can speak or gesture; a waived one is not checked; a Material component never refuses, costly or not; an unrecognised statement refuses the cast (SRD 5.2.1, "Spellcasting", p. 257; "Verbal (V)", "Somatic (S)", pp. 105-106).
// Path: per case.
func TestComponentsInCasts(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	cases := []struct {
		name, requirement, path string
		spell, statement        string
		speak, gesture          bool
		refusal                 combat.MonsterCastRefusal
	}{
		{"silenced, V waived by the statement", "FR-018 CS-1", "unhappy (state error)", "spoken_word", "requiring no spell components", false, true, 0},
		{"silenced, V needed", "FR-018 CS-1", "unhappy (state error)", "spoken_word", "", false, true, combat.MonsterCastRefusalVerbalMissing},
		{"no hands, S needed", "FR-018 CS-1", "unhappy (state error)", "silent_gesture", "", true, false, combat.MonsterCastRefusalSomaticMissing},
		{"no hands, S waived", "FR-018 CS-1", "unhappy (state error)", "silent_gesture", "requiring no Somatic or Material components", true, false, 0},
		{"silenced, only M waived", "FR-018", "unhappy (state error)", "spoken_word", "requiring no Material components", false, true, combat.MonsterCastRefusalVerbalMissing},
		{"silenced, V and S waived", "FR-017", "happy", "fireball", "requiring no Verbal or Somatic components", false, false, 0},
		{"a spell with neither V nor S, silenced", "FR-018", "unhappy (boundary)", "light_none", "", false, false, 0},
		{"costly material, no statement", "FR-019 CS-1", "unhappy (boundary)", "costly_material", "", true, true, 0},
		{"costly material, M waived", "FR-019", "unhappy (boundary)", "costly_material", "requiring no Material components", true, true, 0},
		{"costly material, no free hands", "FR-019", "unhappy (rule interaction)", "costly_material", "requiring no spell components", false, false, 0},
		{"unknown statement", "FR-017 CS-1", "unhappy (invalid input)", "fireball", "requiring no Arcane components", true, true, combat.MonsterCastRefusalComponentsUnrecognized},
		{"trailing clause", "FR-017 CS-1", "unhappy (invalid input)", "fireball", "requiring no spell components and using Wisdom", true, true, combat.MonsterCastRefusalComponentsUnrecognized},
	}
	spells["light_none"] = mscSpellOf("light_none", 1, mscComponents(false, false))
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			spell := spells[c.spell]
			rig := mscSingle(t, content.SpellFrequencyAtWill{}, mscEntry(spell.Name(), "", 0), func(f *content.MonsterSpellcastingFields) { f.Components = c.statement })
			rig.mc = mscContext(c.speak, c.gesture, combat.ConditionActionAvailabilityAvailable)
			res := rig.cast(mscList(t, mscFirst), spell)
			if c.refusal != 0 {
				if got := mscRefused(t, res); got != c.refusal {
					t.Errorf("refusal = %v, want %v [%s]", got, c.refusal, monsterSpellcastingSpec+c.requirement)
				}
				return
			}
			mscPerformed(t, res)
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-039, FR-021, CS-1 - a part cast takes its ability, DC, bonus and components from its own PartSpellcasting and the Modifier of the monster's score in that ability; absent its own values it takes the trait's ability and numbers but never the trait's components statement; a caster with no trait and no own values is an error; the cast is immediate whatever the printed casting time (SRD 5.2.1, "Cloaker", p. 274; "Pit Fiend", p. 315; "Planetar", p. 315).
// Path: per case.
func TestPartCasts(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	m := mscMonster(t, nil)
	parts := luMust(combat.MonsterSpellcasterForParts(m, mscID()))
	rig := mscNewRig(t, m)
	own := func(f combat.PartSpellcastingFields) combat.PartSpellcasting {
		return luMust(combat.NewPartSpellcasting(f))
	}
	cloaker := own(combat.PartSpellcastingFields{Ability: ability.Wisdom, Components: "requiring no spell components"})
	hellfire := own(combat.PartSpellcastingFields{Ability: ability.Charisma, SaveDC: 21, HasSaveDC: true, Components: "requiring no Material components"})

	t.Run("cloaker: own ability, no DC, no components needed", func(t *testing.T) {
		t.Parallel()
		silent := rig
		silent.caster, silent.mc = parts, mscContext(false, false, combat.ConditionActionAvailabilityAvailable)
		p := mscPerformed(t, silent.cast(mscPartCast(t, mscPhantasms, 0, cloaker), spells["mirror_image"]))
		if _, has := p.Report.SaveDC(); has || p.Report.Ability() != ability.Wisdom || p.Report.AbilityModifier().Value() != 1 {
			t.Errorf("report = ability %v modifier %v, DC present %v, want wisdom +1 and no DC", p.Report.Ability(), p.Report.AbilityModifier(), has)
		}
	})
	t.Run("hellfire: own DC and Charisma, Verbal checked", func(t *testing.T) {
		t.Parallel()
		silenced := rig
		silenced.caster, silenced.mc = parts, mscContext(false, true, combat.ConditionActionAvailabilityAvailable)
		if got := mscRefused(t, silenced.cast(mscPartCast(t, mscMistyStep, 5, hellfire), spells["fireball"])); got != combat.MonsterCastRefusalVerbalMissing {
			t.Errorf("silenced refusal = %v, want verbal_component_missing", got)
		}
		p := mscPerformed(t, parts2(rig, parts).cast(mscPartCast(t, mscMistyStep, 5, hellfire), spells["fireball"]))
		dc, has := p.Report.SaveDC()
		if at, _ := p.Report.CastAtLevel(); int(dc) != 21 || !has || p.Report.AbilityModifier().Value() != 5 || at != 5 {
			t.Errorf("report = DC %v (%v), modifier %v, level %d, want 21, +5, 5", dc, has, p.Report.AbilityModifier(), at)
		}
	})
	t.Run("a part with no own values on a caster with no trait", func(t *testing.T) {
		t.Parallel()
		before := parts2(rig, parts)
		err := before.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["misty_step"]).Err
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("no own values: the trait's ability and numbers, not its components", func(t *testing.T) {
		t.Parallel()
		withStatement := mscNewRig(t, mscMonster(t, func(f *content.MonsterSpellcastingFields) { f.Components = "requiring no spell components" }))
		withStatement.mc = mscContext(false, true, combat.ConditionActionAvailabilityAvailable)
		if got := mscRefused(t, withStatement.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["spoken_word"])); got != combat.MonsterCastRefusalVerbalMissing {
			t.Errorf("refusal = %v, want verbal_component_missing (the trait's statement never applies to a part)", got)
		}
		withStatement.mc = mscContext(true, true, combat.ConditionActionAvailabilityAvailable)
		p := mscPerformed(t, withStatement.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["fireball"]))
		if dc, _ := p.Report.SaveDC(); int(dc) != 14 || p.Report.Ability() != ability.Intelligence {
			t.Errorf("report = DC %v, ability %v, want the trait's 14 and intelligence", dc, p.Report.Ability())
		}
	})
	t.Run("immediate whatever the printed casting time", func(t *testing.T) {
		t.Parallel()
		p := mscPerformed(t, rig.cast(mscPartCast(t, mscPhantasms, 0, cloaker), spells["scrying"]))
		if got := mscKinds(p.Transition.Events()); got[len(got)-1] != "monster_spell_cast" {
			t.Errorf("events = %v, want a finished cast", got)
		}
	})
	t.Run("a part cast of a save spell with no DC of its own or the trait's", func(t *testing.T) {
		t.Parallel()
		noDC := own(combat.PartSpellcastingFields{Ability: ability.Wisdom})
		if got := mscRefused(t, rig.cast(mscPartCast(t, mscPhantasms, 0, noDC), spells["fireball"])); got != combat.MonsterCastRefusalMissingSaveDC {
			t.Errorf("refusal = %v, want missing_save_dc", got)
		}
	})
	t.Run("PartSpellcasting accessors and validation", func(t *testing.T) {
		t.Parallel()
		p := own(combat.PartSpellcastingFields{Ability: ability.Charisma, SaveDC: 21, HasSaveDC: true, AttackBonus: 9, HasAttackBonus: true, Components: "x"})
		dc, hasDC := p.SaveDC()
		bonus, hasBonus := p.AttackBonus()
		if p.Ability() != ability.Charisma || dc != 21 || !hasDC || bonus != 9 || !hasBonus || p.Components() != "x" || p.IsZero() {
			t.Errorf("accessors = %v %d %v %d %v %q", p.Ability(), dc, hasDC, bonus, hasBonus, p.Components())
		}
		ignored := own(combat.PartSpellcastingFields{Ability: ability.Wisdom, SaveDC: 99, AttackBonus: 99})
		if dc, ok := ignored.SaveDC(); ok || dc != 0 {
			t.Errorf("an unset DC reads %d, %v, want none", dc, ok)
		}
		if b, ok := ignored.AttackBonus(); ok || b != 0 {
			t.Errorf("an unset bonus reads %d, %v, want none", b, ok)
		}
		bads := []combat.PartSpellcastingFields{
			{}, {Ability: ability.Wisdom, HasSaveDC: true, SaveDC: 0}, {Ability: ability.Wisdom, HasAttackBonus: true, AttackBonus: -content.MonsterMaxNumber - 1},
		}
		if math.MaxInt > content.MonsterMaxNumber { // the numbers above the maximum exist only at 64-bit width
			above := math.MaxInt
			bads = append(bads, combat.PartSpellcastingFields{Ability: ability.Wisdom, HasSaveDC: true, SaveDC: above},
				combat.PartSpellcastingFields{Ability: ability.Wisdom, HasAttackBonus: true, AttackBonus: above})
		}
		for _, bad := range bads {
			if _, err := combat.NewPartSpellcasting(bad); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("NewPartSpellcasting(%+v) error = %v, want ErrInvalidInput", bad, err)
			}
		}
		edge := combat.PartSpellcastingFields{Ability: ability.Wisdom, HasSaveDC: true, SaveDC: content.MonsterMinSaveDC, HasAttackBonus: true, AttackBonus: -content.MonsterMaxNumber}
		if _, err := combat.NewPartSpellcasting(edge); err != nil {
			t.Errorf("NewPartSpellcasting(at the minimum) error = %v", err)
		}
		edge.SaveDC, edge.AttackBonus = content.MonsterMaxNumber, content.MonsterMaxNumber
		if _, err := combat.NewPartSpellcasting(edge); err != nil {
			t.Errorf("NewPartSpellcasting(at the maximum) error = %v", err)
		}
	})
}

// parts2 returns rig with its caster replaced.
func parts2(rig mscRig, c combat.MonsterSpellcaster) mscRig {
	rig.caster = c
	return rig
}

// Requirement: doc/specs/monster-spellcasting.md FR-001, FR-002 - a ListCast of a ref naming no group or no spell is refused with not_in_spell_list; a caster for parts has no list, so any ListCast by it is refused the same way; the Spell.Name must equal the entry's exactly (SRD 5.2.1, "Spellcasting", p. 257).
// Path: unhappy (invalid input).
func TestListCastReferences(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	m := mscMonster(t, nil)
	rig := mscNewRig(t, m)
	for _, ref := range [][2]int{{9, 0}, {0, 99}, {3, 3}} {
		if got := mscRefused(t, rig.cast(mscList(t, ref), spells["light"])); got != combat.MonsterCastRefusalNotInSpellList {
			t.Errorf("ref %v refusal = %v, want not_in_spell_list", ref, got)
		}
	}
	forParts := parts2(rig, luMust(combat.MonsterSpellcasterForParts(m, mscID())))
	if got := mscRefused(t, forParts.cast(mscList(t, mscLight), spells["light"])); got != combat.MonsterCastRefusalNotInSpellList {
		t.Errorf("ListCast by a caster for parts refusal = %v, want not_in_spell_list", got)
	}
	for _, wrong := range []string{"lighter", "fireball", "ligh"} {
		if err := rig.cast(mscList(t, mscLight), mscSpellOf(wrong, 0)).Err; !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("spell %q for the entry light: error = %v, want ErrInvalidInput", wrong, err)
		}
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-035 - only a caller defect is an error matching srderr.ErrInvalidInput (a zero or mismatched caster, state, limited-use state, spell or context, a nil or zero request, a zero target id); the inputs stay unchanged.
// Path: per case.
func TestCastCallerDefects(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	rig := mscNewRig(t, mscMonster(t, nil))
	other := luMust(combat.NewMonsterSpellcastingState(combat.MustCombatantID("someone_else")))
	otherUses := luMust(combat.StartLimitedUse(mscMonster(t, nil), combat.MustCombatantID("someone_else"), combat.OutsideLair))
	cases := []struct {
		name string
		call func() mscResult
	}{
		{"zero caster", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(combat.MonsterSpellcaster{}, rig.state, rig.uses, mscList(t, mscLight), spells["light"], nil, rig.mc))
		}},
		{"zero state", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(rig.caster, combat.MonsterSpellcastingState{}, rig.uses, mscList(t, mscLight), spells["light"], nil, rig.mc))
		}},
		{"zero limited-use state", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(rig.caster, rig.state, combat.LimitedUseState{}, mscList(t, mscLight), spells["light"], nil, rig.mc))
		}},
		{"state of another combatant", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(rig.caster, other, rig.uses, mscList(t, mscLight), spells["light"], nil, rig.mc))
		}},
		{"limited-use state of another combatant", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(rig.caster, rig.state, otherUses, mscList(t, mscLight), spells["light"], nil, rig.mc))
		}},
		{"zero spell", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(rig.caster, rig.state, rig.uses, mscList(t, mscLight), content.Spell{}, nil, rig.mc))
		}},
		{"zero context", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.Cast(rig.caster, rig.state, rig.uses, mscList(t, mscLight), spells["light"], nil, combat.MonsterCastContext{}))
		}},
		{"zero target", func() mscResult {
			return rig.cast(mscList(t, mscLight), spells["light"], combat.CombatantID(""))
		}},
		{"nil request", func() mscResult { return rig.cast(nil, spells["light"]) }},
		{"zero part cast", func() mscResult { return rig.cast(combat.MonsterPartCast{}, spells["light"]) }},
		{"complete: zero caster", func() mscResult {
			return mscWrap(combat.MonsterSpellcastingRules{}.CompleteCasting(combat.MonsterSpellcaster{}, rig.state, rig.uses, mscList(t, mscLight), spells["light"], nil, rig.mc))
		}},
		{"complete: nil request", func() mscResult {
			started := rig.then(t, mustStart(t, rig, mscGeas, spells["geas"]))
			return started.complete(nil, spells["geas"])
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := rig
			res := c.call()
			if !errors.Is(res.Err, srderr.ErrInvalidInput) || res.Out != nil {
				t.Errorf("Cast() = %#v, %v, want nil and ErrInvalidInput [FR-035]", res.Out, res.Err)
			}
			mscUnchanged(t, before, rig)
		})
	}
}

// mustStart returns the started outcome of a ListCast of ref.
func mustStart(tb testing.TB, rig mscRig, ref [2]int, spell content.Spell) combat.MonsterCastOutcome {
	tb.Helper()
	res := rig.cast(mscList(tb, ref), spell)
	return mscStarted(tb, res)
}

// Requirement: doc/specs/monster-spellcasting.md FR-020, FR-021, FR-031 - a monster that cannot take actions casts nothing; an N/Day entry spends its own use through the LimitedUseState in the same call, an At Will spell and an unlimited part spend nothing, a part spends its own X/Day, and a spent-out entry is refused before anything is spent (SRD 5.2.1, "Incapacitated", p. 184; "Limited Usage", p. 257; "Mage", p. 305).
// Path: per case.
func TestUsesAreCounted(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	rig := mscNewRig(t, mscMonster(t, nil))

	t.Run("an N/Day Each entry spends its use and then is exhausted", func(t *testing.T) {
		t.Parallel()
		p := mscPerformed(t, rig.cast(mscList(t, mscFireball), spells["fireball"]))
		next := rig.then(t, p)
		if next.uses.AvailabilityOfSpell(mscRef(t, mscFireball)) != combat.UseExhausted || len(p.Transition.LimitedUseEvents()) != 1 {
			t.Errorf("availability = %v, limited-use events = %d, want exhausted and one", next.uses.AvailabilityOfSpell(mscRef(t, mscFireball)), len(p.Transition.LimitedUseEvents()))
		}
		spent := p.Transition.LimitedUseEvents()[0].(combat.LimitedUsePartSpent)
		if spent.Subject() != (combat.SpellSubject{Spell: mscRef(t, mscFireball)}) || spent.Remaining() != 0 {
			t.Errorf("spent event = %+v, want the fireball entry with none left", spent)
		}
		before := next
		if got := mscRefused(t, next.cast(mscList(t, mscFireball), spells["fireball"])); got != combat.MonsterCastRefusalUsesExhausted {
			t.Errorf("refusal = %v, want uses_exhausted", got)
		}
		mscUnchanged(t, before, next)
	})
	t.Run("an At Will spell spends nothing", func(t *testing.T) {
		t.Parallel()
		p := mscPerformed(t, rig.cast(mscList(t, mscLight), spells["light"]))
		if !p.Transition.LimitedUse().Equal(rig.uses) || len(p.Transition.LimitedUseEvents()) != 0 {
			t.Errorf("an At Will cast changed the limited-use state or emitted events")
		}
	})
	t.Run("a part spends its own X/Day", func(t *testing.T) {
		t.Parallel()
		p := mscPerformed(t, rig.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["misty_step"]))
		if got := luPerDayOf(t, p.Transition.LimitedUse(), mscPart(t, mscMistyStep)).Remaining(); got != 2 {
			t.Errorf("Misty Step uses left = %d, want 2", got)
		}
		again := rig.then(t, p)
		for range 2 {
			again = again.then(t, mscPerformedOf(t, again.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["misty_step"])))
		}
		if got := mscRefused(t, again.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["misty_step"])); got != combat.MonsterCastRefusalUsesExhausted {
			t.Errorf("fourth Misty Step refusal = %v, want uses_exhausted", got)
		}
	})
	t.Run("an unlimited part spends nothing", func(t *testing.T) {
		t.Parallel()
		p := mscPerformed(t, rig.cast(mscPartCast(t, mscPhantasms, 0, combat.PartSpellcasting{}), spells["mirror_image"]))
		if !p.Transition.LimitedUse().Equal(rig.uses) || len(p.Transition.LimitedUseEvents()) != 0 {
			t.Errorf("an unlimited part cast changed the limited-use state")
		}
	})
	t.Run("a monster that cannot act casts nothing", func(t *testing.T) {
		t.Parallel()
		blocked := rig
		blocked.mc = mscContext(true, true, combat.ConditionActionAvailabilityNone)
		for _, req := range []combat.MonsterCastRequest{mscList(t, mscLight), mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{})} {
			if got := mscRefused(t, blocked.cast(req, spells["light"])); got != combat.MonsterCastRefusalCannotTakeActions {
				t.Errorf("%T refusal = %v, want cannot_take_actions", req, got)
			}
		}
	})
}

// mscPerformedOf returns the outcome of a cast that must have been performed, for then.
func mscPerformedOf(tb testing.TB, res mscResult) combat.MonsterCastOutcome {
	tb.Helper()
	mscPerformed(tb, res)
	return res.Out
}

// Requirement: doc/specs/monster-spellcasting.md FR-031 - the refusal reasons are returned in the fixed precedence order whatever the order the checks were written in, and a refusal changes nothing.
// Path: per case.
func TestRefusalPrecedence(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	spells["save_and_attack"] = mscSaveAndAttack()
	other := combat.MustCombatantID("ally")
	type setup struct {
		name    string
		rig     func(t *testing.T) mscRig
		req     func(t *testing.T) combat.MonsterCastRequest
		spell   string
		targets []combat.CombatantID
		want    combat.MonsterCastRefusal
	}
	listOf := func(ref [2]int) func(*testing.T) combat.MonsterCastRequest {
		return func(t *testing.T) combat.MonsterCastRequest {
			t.Helper()
			return mscList(t, ref)
		}
	}
	inProgress := func(t *testing.T) mscRig {
		t.Helper()
		rig := mscNewRig(t, mscMonster(t, nil))
		return rig.then(t, mustStart(t, rig, mscGeas, spells["geas"]))
	}
	custom := func(statement string, speak, gesture bool, a combat.ConditionActionAvailability, note string) func(*testing.T) mscRig {
		return func(t *testing.T) mscRig {
			t.Helper()
			rig := mscSingle(t, luPerDayEach(1), mscEntry("save_and_attack", note, 0), func(f *content.MonsterSpellcastingFields) {
				f.Components, f.HasSaveDC, f.HasAttackBonus = statement, false, false
			})
			rig.mc = mscContext(speak, gesture, a)
			return rig
		}
	}
	cases := []setup{
		{"out of the list, incapacitated", func(t *testing.T) mscRig {
			t.Helper()
			rig := mscNewRig(t, mscMonster(t, nil))
			rig.mc = mscContext(true, true, combat.ConditionActionAvailabilityNone)
			return rig
		}, listOf([2]int{9, 9}), "light", nil, combat.MonsterCastRefusalNotInSpellList},
		{"casting in progress, incapacitated", func(t *testing.T) mscRig {
			t.Helper()
			rig := inProgress(t)
			rig.mc = mscContext(true, true, combat.ConditionActionAvailabilityNone)
			return rig
		}, listOf(mscLight), "light", nil, combat.MonsterCastRefusalAlreadyInProgress},
		{"incapacitated, unrecognised statement", custom("requiring no Arcane components", true, true, combat.ConditionActionAvailabilityNone, ""),
			listOf(mscFirst), "save_and_attack", nil, combat.MonsterCastRefusalCannotTakeActions},
		{"unrecognised statement, silenced", custom("requiring no Arcane components", false, false, combat.ConditionActionAvailabilityAvailable, ""),
			listOf(mscFirst), "save_and_attack", nil, combat.MonsterCastRefusalComponentsUnrecognized},
		{"silenced, no hands", custom("", false, false, combat.ConditionActionAvailabilityAvailable, ""),
			listOf(mscFirst), "save_and_attack", nil, combat.MonsterCastRefusalVerbalMissing},
		{"no hands, self only violated", custom("", true, false, combat.ConditionActionAvailabilityAvailable, "self only"),
			listOf(mscFirst), "save_and_attack", []combat.CombatantID{other}, combat.MonsterCastRefusalSomaticMissing},
		{"self only violated, no DC", custom("", true, true, combat.ConditionActionAvailabilityAvailable, "self only"),
			listOf(mscFirst), "save_and_attack", []combat.CombatantID{other}, combat.MonsterCastRefusalSelfOnlyViolated},
		{"no DC, no bonus", custom("", true, true, combat.ConditionActionAvailabilityAvailable, ""),
			listOf(mscFirst), "save_and_attack", nil, combat.MonsterCastRefusalMissingSaveDC},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig := c.rig(t)
			before := rig
			if got := mscRefused(t, rig.cast(c.req(t), spells[c.spell], c.targets...)); got != c.want {
				t.Errorf("refusal = %v, want %v [FR-031]", got, c.want)
			}
			mscUnchanged(t, before, rig)
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-031 - the exhausted-use refusal comes after every other refusal: a spent-out entry that also lacks the printed DC reports the DC.
// Path: unhappy (rule interaction).
func TestExhaustedComesLast(t *testing.T) {
	t.Parallel()
	spell := mscSaveAndAttack()
	rig := mscSingle(t, luPerDayEach(1), mscEntry("save_and_attack", "", 0), func(f *content.MonsterSpellcastingFields) { f.HasSaveDC, f.HasAttackBonus = false, false })
	rig.uses = luSpellSpent(t, rig.uses, mscRef(t, mscFirst))
	if got := mscRefused(t, rig.cast(mscList(t, mscFirst), spell)); got != combat.MonsterCastRefusalMissingSaveDC {
		t.Errorf("refusal = %v, want missing_save_dc before uses_exhausted", got)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-022, FR-035 - a spell with several casting time modes needs one of them: no mode, an unknown mode and a mode for a spell with one casting time are errors; the Magic action mode finishes at once, the 8 hours mode starts a casting of 480 minutes; a casting time too long to count in minutes is an error (SRD 5.2.1, "Plant Growth", p. 153).
// Path: per case.
func TestCastingTimeModes(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	rig := mscNewRig(t, mscMonster(t, nil))
	plant := spells["plant_growth"]
	modes := plant.CastingTimes().(content.SpellCastingTimesModes).Modes()
	foreign := luMust(content.NewCastingTimeMode("Other", content.NewCastingTimeMagicAction()))
	for _, c := range []struct {
		name, requirement, path string
		mode                    content.CastingTimeMode
	}{
		{"no mode", "FR-022", "unhappy (invalid input)", content.CastingTimeMode{}},
		{"a mode of another spell", "FR-022", "unhappy (invalid input)", foreign},
	} {
		if err := rig.cast(combat.NewMonsterListCast(mscRef(t, mscPlantGrowth), c.mode), plant).Err; !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput [%s]", c.name, err, c.requirement)
		}
	}
	if err := rig.cast(combat.NewMonsterListCast(mscRef(t, mscLight), modes[0]), spells["light"]).Err; !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a mode for a spell with one casting time: error = %v, want ErrInvalidInput", err)
	}
	mscPerformed(t, rig.cast(combat.NewMonsterListCast(mscRef(t, mscPlantGrowth), modes[0]), plant))
	started := mscStarted(t, rig.cast(combat.NewMonsterListCast(mscRef(t, mscPlantGrowth), modes[1]), plant))
	casting, _, ok := started.Transition.State().CastingInProgress()
	if !ok || casting.TotalMinutes() != 480 || casting.Mode() != modes[1] {
		t.Errorf("casting = %d minutes, mode %v, want 480 and the 8 hours mode", casting.TotalMinutes(), casting.Mode())
	}
	huge := mscSingle(t, content.SpellFrequencyAtWill{}, mscEntry("huge", "", 0), nil)
	if err := huge.cast(mscList(t, mscFirst), mscSpellOf("huge", 1, mscCastingHours(math.MaxInt))).Err; !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a casting of MaxInt hours: error = %v, want ErrInvalidInput", err)
	}
	if err := huge.cast(mscList(t, mscFirst), mscSpellOf("huge", 1, mscCastingHours(math.MaxInt/60+1))).Err; !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a casting of MaxInt/60+1 hours: error = %v, want ErrInvalidInput", err)
	}
	if got := mscStarted(t, huge.cast(mscList(t, mscFirst), mscSpellOf("huge", 1, mscCastingHours(math.MaxInt/60)))); got.Transition.State().IsZero() {
		t.Errorf("a casting of MaxInt/60 hours did not start")
	}
	hours := mscSingle(t, content.SpellFrequencyAtWill{}, mscEntry("long_ritual", "", 0), nil)
	started = mscStarted(t, hours.cast(mscList(t, mscFirst), spells["long_ritual"]))
	if c, _, _ := started.Transition.State().CastingInProgress(); c.TotalMinutes() != 480 {
		t.Errorf("an 8 hour casting = %d minutes, want 480", c.TotalMinutes())
	}
}
