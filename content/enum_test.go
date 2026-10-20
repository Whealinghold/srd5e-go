package content_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// enum is what every vocabulary type of package content offers.
type enum interface {
	~int
	String() string
	Valid() bool
}

// checkVocabulary asserts the shared contract of a closed vocabulary: the
// members in order from ordinal 1, exact case-sensitive parsing, a typed error
// for everything else, and an invalid zero value.
func checkVocabulary[T enum](vocabulary string, want []string, consts []T, parse func(string) (T, error)) []string {
	if len(consts) != len(want) {
		return []string{fmt.Sprintf("%s: %d constants for %d names", vocabulary, len(consts), len(want))}
	}
	var fails []string
	for i, w := range want {
		v, err := parse(w)
		if err != nil || v != consts[i] || int(v) != i+1 || v.String() != w || !v.Valid() {
			fails = append(fails, fmt.Sprintf("%s: parse(%q) = (%d, %v), want ordinal %d, String %q, valid", vocabulary, w, v, err, i+1, w))
		}
		bad := []string{w + " ", " " + w, w + "x"}
		if l := strings.ToLower(w); l != w {
			bad = append(bad, l)
		}
		if u := strings.ToUpper(w); u != w {
			bad = append(bad, u)
		}
		for _, in := range bad {
			fails = append(fails, parseFailure(vocabulary, in, parse)...)
		}
	}
	fails = append(fails, parseFailure(vocabulary, "", parse)...)
	for _, v := range []T{0, T(len(want) + 1), -1} {
		if v.Valid() || v.String() != "unknown" {
			fails = append(fails, fmt.Sprintf("%s: %d is Valid=%v String=%q, want invalid and unknown", vocabulary, int(v), v.Valid(), v.String()))
		}
	}
	return fails
}

func parseFailure[T enum](vocabulary, in string, parse func(string) (T, error)) []string {
	v, err := parse(in)
	var pe *content.ParseError
	if v != 0 || !errors.As(err, &pe) || pe.Vocabulary != vocabulary || pe.Value != in || !errors.Is(err, srderr.ErrInvalidInput) {
		return []string{fmt.Sprintf("%s: parse(%q) = (%d, %v), want 0 and *ParseError{%s, %q} wrapping ErrInvalidInput", vocabulary, in, v, err, vocabulary, in)}
	}
	return nil
}

// Requirement: per case
// Path: per case.
func TestVocabularies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		run                     func() []string
	}{
		{"Skill", `schema FR-106, SRD 5.2.1, "Skills", p. 9`, "happy", func() []string {
			return checkVocabulary("Skill", []string{
				"Acrobatics", "Animal Handling", "Arcana", "Athletics", "Deception", "History",
				"Insight", "Intimidation", "Investigation", "Medicine", "Nature", "Perception",
				"Performance", "Persuasion", "Religion", "Sleight of Hand", "Stealth", "Survival",
			}, []content.Skill{content.SkillAcrobatics, content.SkillAnimalHandling, content.SkillArcana, content.SkillAthletics, content.SkillDeception, content.SkillHistory, content.SkillInsight, content.SkillIntimidation, content.SkillInvestigation, content.SkillMedicine, content.SkillNature, content.SkillPerception, content.SkillPerformance, content.SkillPersuasion, content.SkillReligion, content.SkillSleightOfHand, content.SkillStealth, content.SkillSurvival}, content.ParseSkill)
		}},
		{"DamageType", `schema FR-107, SRD 5.2.1, "Damage Types", p. 180`, "happy", func() []string {
			return checkVocabulary("DamageType", []string{
				"Acid", "Bludgeoning", "Cold", "Fire", "Force", "Lightning", "Necrotic",
				"Piercing", "Poison", "Psychic", "Radiant", "Slashing", "Thunder",
			}, []content.DamageType{content.DamageAcid, content.DamageBludgeoning, content.DamageCold, content.DamageFire, content.DamageForce, content.DamageLightning, content.DamageNecrotic, content.DamagePiercing, content.DamagePoison, content.DamagePsychic, content.DamageRadiant, content.DamageSlashing, content.DamageThunder}, content.ParseDamageType)
		}},
		{"CreatureType", `schema FR-108, SRD 5.2.1, "Creature Type", p. 179`, "happy", func() []string {
			return checkVocabulary("CreatureType", []string{
				"Aberration", "Beast", "Celestial", "Construct", "Dragon", "Elemental", "Fey",
				"Fiend", "Giant", "Humanoid", "Monstrosity", "Ooze", "Plant", "Undead",
			}, []content.CreatureType{content.CreatureTypeAberration, content.CreatureTypeBeast, content.CreatureTypeCelestial, content.CreatureTypeConstruct, content.CreatureTypeDragon, content.CreatureTypeElemental, content.CreatureTypeFey, content.CreatureTypeFiend, content.CreatureTypeGiant, content.CreatureTypeHumanoid, content.CreatureTypeMonstrosity, content.CreatureTypeOoze, content.CreatureTypePlant, content.CreatureTypeUndead}, content.ParseCreatureType)
		}},
		{"CreatureSize", `schema FR-109, SRD 5.2.1, "Size", p. 254`, "happy", func() []string {
			return checkVocabulary("CreatureSize", []string{"Tiny", "Small", "Medium", "Large", "Huge", "Gargantuan"}, []content.CreatureSize{content.SizeTiny, content.SizeSmall, content.SizeMedium, content.SizeLarge, content.SizeHuge, content.SizeGargantuan}, content.ParseCreatureSize)
		}},
		{"ConditionKind", `schema FR-110, SRD 5.2.1, "Condition", p. 179`, "happy", func() []string {
			return checkVocabulary("ConditionKind", []string{
				"Blinded", "Charmed", "Deafened", "Exhaustion", "Frightened", "Grappled",
				"Incapacitated", "Invisible", "Paralyzed", "Petrified", "Poisoned", "Prone",
				"Restrained", "Stunned", "Unconscious",
			}, []content.ConditionKind{content.ConditionBlinded, content.ConditionCharmed, content.ConditionDeafened, content.ConditionExhaustion, content.ConditionFrightened, content.ConditionGrappled, content.ConditionIncapacitated, content.ConditionInvisible, content.ConditionParalyzed, content.ConditionPetrified, content.ConditionPoisoned, content.ConditionProne, content.ConditionRestrained, content.ConditionStunned, content.ConditionUnconscious}, content.ParseConditionKind)
		}},
		{"Language", `schema FR-111, SRD 5.2.1, "Standard Languages", p. 20`, "happy", func() []string {
			return checkVocabulary("Language", []string{
				"Common", "Common Sign Language", "Draconic", "Dwarvish", "Elvish", "Giant",
				"Gnomish", "Goblin", "Halfling", "Orc",
				"Abyssal", "Celestial", "Deep Speech", "Druidic", "Infernal", "Primordial",
				"Sylvan", "Thieves’ Cant", "Undercommon",
			}, []content.Language{content.LanguageCommon, content.LanguageCommonSignLanguage, content.LanguageDraconic, content.LanguageDwarvish, content.LanguageElvish, content.LanguageGiant, content.LanguageGnomish, content.LanguageGoblin, content.LanguageHalfling, content.LanguageOrc, content.LanguageAbyssal, content.LanguageCelestial, content.LanguageDeepSpeech, content.LanguageDruidic, content.LanguageInfernal, content.LanguagePrimordial, content.LanguageSylvan, content.LanguageThievesCant, content.LanguageUndercommon}, content.ParseLanguage)
		}},
		{"RechargeTrigger", `schema FR-112, SRD 5.2.1, "Limited Usage", p. 257`, "happy", func() []string {
			return checkVocabulary("RechargeTrigger", []string{"shortRest", "longRest", "shortOrLongRest"}, []content.RechargeTrigger{content.RechargeShortRest, content.RechargeLongRest, content.RechargeShortOrLongRest}, content.ParseRechargeTrigger)
		}},
		{"WeaponProperty", `schema FR-113, SRD 5.2.1, "Properties", pp. 89-90`, "happy", func() []string {
			return checkVocabulary("WeaponProperty", []string{
				"Ammunition", "Finesse", "Heavy", "Light", "Loading", "Reach", "Thrown", "Two-Handed", "Versatile",
			}, []content.WeaponProperty{content.WeaponPropertyAmmunition, content.WeaponPropertyFinesse, content.WeaponPropertyHeavy, content.WeaponPropertyLight, content.WeaponPropertyLoading, content.WeaponPropertyReach, content.WeaponPropertyThrown, content.WeaponPropertyTwoHanded, content.WeaponPropertyVersatile}, content.ParseWeaponProperty)
		}},
		{"MasteryProperty", `schema FR-113, SRD 5.2.1, "Mastery Properties", p. 90`, "happy", func() []string {
			return checkVocabulary("MasteryProperty", []string{"Cleave", "Graze", "Nick", "Push", "Sap", "Slow", "Topple", "Vex"}, []content.MasteryProperty{content.MasteryCleave, content.MasteryGraze, content.MasteryNick, content.MasteryPush, content.MasterySap, content.MasterySlow, content.MasteryTopple, content.MasteryVex}, content.ParseMasteryProperty)
		}},
		{"FeatCategory", `schema FR-117, SRD 5.2.1, "Parts of a Feat: Category", p. 87`, "happy", func() []string {
			return checkVocabulary("FeatCategory", []string{"Origin", "General", "Fighting Style", "Epic Boon"}, []content.FeatCategory{content.FeatOrigin, content.FeatGeneral, content.FeatFightingStyle, content.FeatEpicBoon}, content.ParseFeatCategory)
		}},
		{"HitPointDie", `schema FR-132, SRD 5.2.1, "Core Barbarian Traits", p. 28`, "happy", func() []string {
			return checkVocabulary("HitPointDie", []string{"D6", "D8", "D10", "D12"}, []content.HitPointDie{content.HitDieD6, content.HitDieD8, content.HitDieD10, content.HitDieD12}, content.ParseHitPointDie)
		}},
		{"LightLevel", `schema FR-133, SRD 5.2.1, "Light", p. 11`, "happy", func() []string {
			return checkVocabulary("LightLevel", []string{"Bright Light", "Dim Light", "Darkness"}, []content.LightLevel{content.LightBright, content.LightDim, content.LightDarkness}, content.ParseLightLevel)
		}},
		{"DarknessKind", `schema FR-134, SRD 5.2.1, "Special Senses", p. 11`, "happy", func() []string {
			return checkVocabulary("DarknessKind", []string{"Nonmagical", "Magical"}, []content.DarknessKind{content.DarknessNonmagical, content.DarknessMagical}, content.ParseDarknessKind)
		}},
		{"Obscurement", `schema FR-135, SRD 5.2.1, "Obscured Areas", p. 11`, "happy", func() []string {
			return checkVocabulary("Obscurement", []string{"None", "Lightly Obscured", "Heavily Obscured"}, []content.Obscurement{content.ObscurementNone, content.ObscurementLightly, content.ObscurementHeavily}, content.ParseObscurement)
		}},
		{"SpecialSense", `schema FR-136, SRD 5.2.1, "Special Senses", p. 11`, "happy", func() []string {
			return checkVocabulary("SpecialSense", []string{"Blindsight", "Darkvision", "Tremorsense", "Truesight"}, []content.SpecialSense{content.SenseBlindsight, content.SenseDarkvision, content.SenseTremorsense, content.SenseTruesight}, content.ParseSpecialSense)
		}},
		{"Alignment", `schema FR-137, SRD 5.2.1, "The Nine Alignments", p. 21`, "happy", func() []string {
			return checkVocabulary("Alignment", []string{
				"Lawful Good", "Neutral Good", "Chaotic Good", "Lawful Neutral", "Neutral",
				"Chaotic Neutral", "Lawful Evil", "Neutral Evil", "Chaotic Evil", "Unaligned",
			}, []content.Alignment{content.AlignmentLawfulGood, content.AlignmentNeutralGood, content.AlignmentChaoticGood, content.AlignmentLawfulNeutral, content.AlignmentNeutral, content.AlignmentChaoticNeutral, content.AlignmentLawfulEvil, content.AlignmentNeutralEvil, content.AlignmentChaoticEvil, content.AlignmentUnaligned}, content.ParseAlignment)
		}},
		{"DurationUnit", `schema FR-149, SRD 5.2.1, "Duration", p. 106`, "happy", func() []string {
			return checkVocabulary("DurationUnit", []string{"rounds", "minutes", "hours", "days"}, []content.DurationUnit{content.DurationUnitRounds, content.DurationUnitMinutes, content.DurationUnitHours, content.DurationUnitDays}, content.ParseDurationUnit)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, f := range c.run() {
				t.Error(f)
			}
		})
	}
}

// Requirement: schema FR-111 (SRD 5.2.1, p. 20) - the ASCII apostrophe in
// "Thieves' Cant" and the Primordial dialects are not members; IsStandard is
// true for the first 10 only.
// Path: unhappy (invalid input).
func TestLanguageSpecifics(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"Thieves' Cant", "Aquan", "Auran", "Ignan", "Terran"} {
		for _, f := range parseFailure("Language", in, content.ParseLanguage) {
			t.Error(f)
		}
	}
	for l := content.LanguageCommon; l <= content.LanguageUndercommon; l++ {
		if got, want := l.IsStandard(), l <= content.LanguageOrc; got != want {
			t.Errorf("%v.IsStandard() = %v, want %v", l, got, want)
		}
	}
	if content.Language(0).IsStandard() || content.Language(20).IsStandard() {
		t.Error("an invalid Language reports IsStandard")
	}
}

// Requirement: schema FR-106 (SRD 5.2.1, "Skills", p. 9) - each skill's printed ability.
// Path: per case.
func TestSkillAbility(t *testing.T) {
	t.Parallel()
	want := map[content.Skill]ability.Ability{
		content.SkillAcrobatics: ability.Dexterity, content.SkillSleightOfHand: ability.Dexterity, content.SkillStealth: ability.Dexterity,
		content.SkillAnimalHandling: ability.Wisdom, content.SkillInsight: ability.Wisdom, content.SkillMedicine: ability.Wisdom,
		content.SkillPerception: ability.Wisdom, content.SkillSurvival: ability.Wisdom,
		content.SkillArcana: ability.Intelligence, content.SkillHistory: ability.Intelligence, content.SkillInvestigation: ability.Intelligence,
		content.SkillNature: ability.Intelligence, content.SkillReligion: ability.Intelligence,
		content.SkillAthletics: ability.Strength,
		content.SkillDeception: ability.Charisma, content.SkillIntimidation: ability.Charisma,
		content.SkillPerformance: ability.Charisma, content.SkillPersuasion: ability.Charisma,
	}
	if len(want) != 18 {
		t.Fatalf("table has %d skills, want 18", len(want))
	}
	for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
		if got := s.Ability(); got != want[s] {
			t.Errorf("%v.Ability() = %v, want %v [happy]", s, got, want[s])
		}
	}
	if content.Skill(0).Ability() != 0 || content.Skill(19).Ability() != 0 {
		t.Error("an invalid Skill has an ability [unhappy (invalid input)]")
	}
}

// Requirement: schema FR-132 (SRD 5.2.1, "Level 1 Hit Points by Class", p. 22) -
// Faces is 6, 8, 10, 12 and converts through dice.NewDie.
// Path: happy.
func TestHitPointDieFaces(t *testing.T) {
	t.Parallel()
	for h, want := range map[content.HitPointDie]int{content.HitDieD6: 6, content.HitDieD8: 8, content.HitDieD10: 10, content.HitDieD12: 12} {
		if got := h.Faces(); got != want {
			t.Errorf("%v.Faces() = %d, want %d", h, got, want)
		}
		if _, err := dice.NewDie(h.Faces()); err != nil {
			t.Errorf("dice.NewDie(%v.Faces()) = %v, want nil", h, err)
		}
	}
	// Requirement: schema FR-132 - the zero value has no faces.
	// Path: unhappy (invalid input).
	if content.HitPointDie(0).Faces() != 0 || content.HitPointDie(5).Faces() != 0 {
		t.Error("an invalid HitPointDie has faces")
	}
}

// Requirement: schema FR-135 - Obscurement is ordered by severity, so the
// constants compare with <.
// Path: unhappy (rule interaction).
func TestObscurementOrder(t *testing.T) {
	t.Parallel()
	if content.ObscurementNone >= content.ObscurementLightly || content.ObscurementLightly >= content.ObscurementHeavily {
		t.Error("Obscurement constants are not ordered by severity")
	}
}

// Requirement: schema FR-109 - sizes are declared in ascending order.
// Path: unhappy (rule interaction).
func TestCreatureSizeOrder(t *testing.T) {
	t.Parallel()
	sizes := []content.CreatureSize{content.SizeTiny, content.SizeSmall, content.SizeMedium, content.SizeLarge, content.SizeHuge, content.SizeGargantuan}
	for i := 1; i < len(sizes); i++ {
		if sizes[i-1] >= sizes[i] {
			t.Errorf("%v is not smaller than %v", sizes[i-1], sizes[i])
		}
	}
}

// FuzzParseEnums feeds one input to every ParseX of the closed vocabularies.
//
// Requirement: schema FR-105, FR-025 - no ParseX panics; an accepted input
// round-trips through String; a rejected one is a *ParseError holding the input.
// Path: unhappy (invalid input).
func FuzzParseEnums(f *testing.F) {
	for _, s := range []string{
		"", " ", "Acid", "acid", "Acrobatics ", "Sleight of Hand", "Thieves’ Cant", "Thieves' Cant",
		"D8", "d8", "shortRest", "Short Rest", "Two-Handed", "Fighting Style", "Lawful Good", "LG",
		"Heavily Obscured", "rounds", "Simple", "Martial", "Light", "Shield", "dice", "count", "Count", "\x00", "\xff\xfe", "Tiny\n", "Wondrous Item", "Wondrous Items", "Very Rare", "Very rare",
		"Evocation", "evocation", "feet", "upTo", "concentrationUpTo", "complete", "creatureOrObject", "halfDamageOnSuccess", "travelsWithin", "atCastingOnly",
	} {
		f.Add(s)
	}
	parsers := map[string]func(string) (string, error){
		"Skill":                  stringer(content.ParseSkill),
		"DamageType":             stringer(content.ParseDamageType),
		"CreatureType":           stringer(content.ParseCreatureType),
		"CreatureSize":           stringer(content.ParseCreatureSize),
		"ConditionKind":          stringer(content.ParseConditionKind),
		"Language":               stringer(content.ParseLanguage),
		"RechargeTrigger":        stringer(content.ParseRechargeTrigger),
		"WeaponProperty":         stringer(content.ParseWeaponProperty),
		"MasteryProperty":        stringer(content.ParseMasteryProperty),
		"FeatCategory":           stringer(content.ParseFeatCategory),
		"HitPointDie":            stringer(content.ParseHitPointDie),
		"LightLevel":             stringer(content.ParseLightLevel),
		"DarknessKind":           stringer(content.ParseDarknessKind),
		"Obscurement":            stringer(content.ParseObscurement),
		"SpecialSense":           stringer(content.ParseSpecialSense),
		"Alignment":              stringer(content.ParseAlignment),
		"DurationUnit":           stringer(content.ParseDurationUnit),
		"WeaponTrainingCategory": stringer(content.ParseWeaponTrainingCategory),
		"ArmorTrainingCategory":  stringer(content.ParseArmorTrainingCategory),
		"ClassColumnKind":        stringer(content.ParseClassColumnKind),
		"MagicItemCategory":      stringer(content.ParseMagicItemCategory),
		"ItemRarity":             stringer(content.ParseItemRarity),
		"SpellSchool":            stringer(content.ParseSpellSchool),
		"DistanceUnit":           stringer(content.ParseDistanceUnit),
		"DurationLimit":          stringer(content.ParseDurationLimit),
		"EffectCoverage":         stringer(content.ParseEffectCoverage),
		"TargetSubject":          stringer(content.ParseTargetSubject),
		"SelfTargetPolicy":       stringer(content.ParseSelfTargetPolicy),
		"SpellAttackKind":        stringer(content.ParseSpellAttackKind),
		"SuccessEffect":          stringer(content.ParseSuccessEffect),
		"ModifierAddition":       stringer(content.ParseModifierAddition),
		"TurnOwner":              stringer(content.ParseTurnOwner),
		"TurnBoundary":           stringer(content.ParseTurnBoundary),
		"ZoneTrigger":            stringer(content.ParseZoneTrigger),
		"ZoneTargetScope":        stringer(content.ParseZoneTargetScope),
		"ZoneActionKind":         stringer(content.ParseZoneActionKind),
		"DriftDirectionRule":     stringer(content.ParseDriftDirectionRule),
		"DesignationPolicy":      stringer(content.ParseDesignationPolicy),
		"StatBlockAttackKind":    stringer(content.ParseStatBlockAttackKind),
		"EffectClauseLabel":      stringer(content.ParseEffectClauseLabel),
		"SpeedMode":              stringer(content.ParseSpeedMode),
		"ReplacementLimit":       stringer(content.ParseReplacementLimit),
	}
	f.Fuzz(func(t *testing.T, in string) {
		for vocabulary, parse := range parsers {
			got, err := parse(in)
			if err == nil {
				if got != in {
					t.Fatalf("%s: %q parsed but String() = %q", vocabulary, in, got)
				}
				continue
			}
			var pe *content.ParseError
			if !errors.As(err, &pe) || pe.Value != in || pe.Vocabulary != vocabulary || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("%s: %q rejected with %v, want *ParseError holding the input", vocabulary, in, err)
			}
		}
	})
}

// stringer adapts a ParseX to a function returning the String of the result.
func stringer[T enum](parse func(string) (T, error)) func(string) (string, error) {
	return func(s string) (string, error) {
		v, err := parse(s)
		return v.String(), err
	}
}
