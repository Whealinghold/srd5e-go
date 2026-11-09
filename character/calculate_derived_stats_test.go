package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func wizardCommand(t *testing.T) character.CalculateDerivedStatsCommand {
	t.Helper()
	return character.CalculateDerivedStatsCommand{
		ClassLevels:     levels(cl("wizard", 5)),
		AbilityScores:   scoresOf(t, 8, 14, 12, 16, 15, 10),
		HitPointMaximum: character.MustHitPointMaximum(27),
	}
}

// Requirement: doc/specs/derived-stats.md FR-009, FR-030, FR-041, FR-050 - a
// level 5 Wizard through the use case: Proficiency Bonus +3 from the total
// level, the stored maximum, AC 10 plus Dexterity, and the Intelligence save.
// Path: happy.
func TestRunLevelFiveWizard(t *testing.T) {
	t.Parallel()
	uc, _ := newUseCase(t, wizardClass(t))
	s := succeeded(t, run(t, uc, wizardCommand(t)))
	if s.ProficiencyBonus().Value() != 3 || s.HitPointMaximum().Value() != 27 || s.ArmorClass() != 12 ||
		s.SavingThrow(ability.Intelligence).Value() != 6 || s.SavingThrow(ability.Strength).Value() != -1 || s.TotalLevel().Value() != 5 {
		t.Errorf("stats: proficiency %d HP %d AC %d Int save %d", s.ProficiencyBonus().Value(), s.HitPointMaximum().Value(), s.ArmorClass(), s.SavingThrow(ability.Intelligence).Value())
	}
}

// Requirement: doc/specs/derived-stats.md FR-009 - a multiclass character through
// the use case gets the Proficiency Bonus of its total level (SRD 5.2.1, p. 25).
// Path: unhappy (rule interaction).
func TestRunMulticlassProficiencyBonus(t *testing.T) {
	t.Parallel()
	uc, _ := newUseCase(t, wizardClass(t))
	cmd := wizardCommand(t)
	cmd.ClassLevels = levels(cl("wizard", 3), cl("fighter", 2))
	if got := succeeded(t, run(t, uc, cmd)).ProficiencyBonus().Value(); got != 3 {
		t.Errorf("3 Wizard / 2 Fighter proficiency bonus = %d, want 3", got)
	}
}

// Requirement: doc/specs/derived-stats.md FR-053 - the armor and Shield ids are
// resolved through the source and used.
// Path: happy.
func TestRunResolvesArmor(t *testing.T) {
	t.Parallel()
	plate, shield := plateArmor(t), shieldOf(t)
	uc, src := newUseCase(t, wizardClass(t), plate, shield)
	cmd := wizardCommand(t)
	cmd.ArmorID, cmd.ShieldID = plate.ID(), shield.ID()
	cmd.MulticlassArmorTraining = trainingOf(t, content.ArmorHeavy)
	s := succeeded(t, run(t, uc, cmd))
	if s.ArmorClass() != 18 || !slices.Equal(s.ArmorTrainingGaps(), []content.ArmorCategory{content.ArmorShield}) {
		t.Errorf("AC %d, gaps %v, want 18 and the Shield", s.ArmorClass(), s.ArmorTrainingGaps())
	}
	if q := src.Queries(); len(q) != 3 {
		t.Errorf("%d lookups, want 3 (class, armor, Shield)", len(q))
	}
}

// Requirement: doc/specs/derived-stats.md FR-043 - unknown ids give
// ContentMissing holding every unknown id once, in lookup order, with no partial
// stats; every lookup is attempted.
// Path: per case.
func TestRunContentMissing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path    string
		have          []content.Entry
		armor, shield content.ItemID
		want          []content.Key
	}{
		{"class missing", "unhappy (missing content)", nil, "", "", []content.Key{content.ClassID("wizard").Key()}},
		{"class and armor missing", "unhappy (missing content)", nil, "plate_armor", "", []content.Key{content.ClassID("wizard").Key(), content.ItemID("plate_armor").Key()}},
		{"armor and Shield missing", "unhappy (missing content)", []content.Entry{wizardClass(t)}, "plate_armor", "shield_x", []content.Key{content.ItemID("plate_armor").Key(), content.ItemID("shield_x").Key()}},
		{"same id missing twice is listed once", "unhappy (missing content)", []content.Entry{wizardClass(t)}, "plate_armor", "plate_armor", []content.Key{content.ItemID("plate_armor").Key()}},
		{"missing beats slot mismatch", "unhappy (rule interaction)", []content.Entry{wizardClass(t), shieldOf(t)}, "shield_x", "gone_shield", []content.Key{content.ItemID("gone_shield").Key()}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, src := newUseCase(t, c.have...)
			cmd := wizardCommand(t)
			cmd.ArmorID, cmd.ShieldID = c.armor, c.shield
			res := run(t, uc, cmd)
			missing, ok := res.(character.CalculateDerivedStatsContentMissing)
			if !ok {
				t.Fatalf("result = %#v, want ContentMissing", res)
			}
			var got []content.Key
			for _, m := range missing.Missing() {
				got = append(got, m.Key())
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("missing = %v, want %v", got, c.want)
			}
			want := 1
			if c.armor != "" {
				want++
			}
			if c.shield != "" {
				want++
			}
			if len(src.Queries()) != want {
				t.Errorf("%d lookups, want %d (all attempted)", len(src.Queries()), want)
			}
			m := missing.Missing()
			m[0] = content.MissingContent{}
			if missing.Missing()[0].Key().IsZero() {
				t.Error("Missing exposes its slice")
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-053, FR-068 - an id that resolves to
// an entry that does not fit its slot is a slot mismatch with the entry's key
// and the expected slot, the armorId first.
// Path: per case.
func TestRunEquipmentSlotMismatch(t *testing.T) {
	t.Parallel()
	sword := weaponOf("sword_x", content.WeaponMelee, gp(10))
	leather, shield := leatherArmor(t), shieldOf(t)
	cases := []struct {
		name, path    string
		armor, shield content.ItemID
		want          character.CalculateDerivedStatsEquipmentSlotMismatch
	}{
		{"Shield as armorId", "unhappy (invalid input)", shield.ID(), "", character.CalculateDerivedStatsEquipmentSlotMismatch{Key: shield.Key(), Expected: character.BodyArmorSlot}},
		{"body armor as shieldId", "unhappy (invalid input)", "", leather.ID(), character.CalculateDerivedStatsEquipmentSlotMismatch{Key: leather.Key(), Expected: character.ShieldSlot}},
		{"Weapon as armorId", "unhappy (invalid input)", sword.ID(), "", character.CalculateDerivedStatsEquipmentSlotMismatch{Key: sword.Key(), Expected: character.BodyArmorSlot}},
		{"Weapon as shieldId", "unhappy (invalid input)", "", sword.ID(), character.CalculateDerivedStatsEquipmentSlotMismatch{Key: sword.Key(), Expected: character.ShieldSlot}},
		{"both mismatch, armorId reported", "unhappy (rule interaction)", shield.ID(), sword.ID(), character.CalculateDerivedStatsEquipmentSlotMismatch{Key: shield.Key(), Expected: character.BodyArmorSlot}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, _ := newUseCase(t, wizardClass(t), sword, leather, shield)
			cmd := wizardCommand(t)
			cmd.ArmorID, cmd.ShieldID = c.armor, c.shield
			if got := run(t, uc, cmd); got != c.want {
				t.Errorf("result = %#v, want %#v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-037, FR-064 - an inapplicable chosen
// basis comes back as a result holding the choice and the reason.
// Path: unhappy (state error).
func TestRunBasisNotApplicable(t *testing.T) {
	t.Parallel()
	leather := leatherArmor(t)
	uc, _ := newUseCase(t, wizardClass(t), leather)
	cmd := wizardCommand(t)
	cmd.ArmorID = leather.ID()
	cmd.AlternativeBases = []character.AbilityBasedArmorClassBasis{basisOf(t, "barbarian")}
	cmd.ChosenBasis = character.ChooseAlternative{ID: content.MustFeatureID("barbarian_unarmored_defense")}
	want := character.CalculateDerivedStatsArmorClassBasisNotApplicable{Choice: cmd.ChosenBasis, Reason: character.BasisNotApplicable}
	if got := run(t, uc, cmd); got != want {
		t.Errorf("result = %#v, want %#v", got, want)
	}
}

// Requirement: doc/specs/derived-stats.md FR-076, FR-080, FR-063 - an invalid
// command is rejected with ErrInvalidInput before any lookup.
// Path: per case.
func TestRunRejectsInvalidCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*character.CalculateDerivedStatsCommand)
	}{
		{"no class levels", func(c *character.CalculateDerivedStatsCommand) { c.ClassLevels = character.ClassLevels{} }},
		{"negative floor", func(c *character.CalculateDerivedStatsCommand) { c.ArmorClassFloor = -1 }},
		{"invalid armor id", func(c *character.CalculateDerivedStatsCommand) { c.ArmorID = "Not An Id" }},
		{"invalid Shield id", func(c *character.CalculateDerivedStatsCommand) { c.ShieldID = "Not An Id" }},
		{"duplicate alternative ids", func(c *character.CalculateDerivedStatsCommand) {
			c.AlternativeBases = []character.AbilityBasedArmorClassBasis{basisOf(t, "monk"), basisOf(t, "monk")}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, src := newUseCase(t, wizardClass(t))
			cmd := wizardCommand(t)
			c.mutate(&cmd)
			res, err := uc.Run(context.Background(), cmd)
			wantInvalid(t, err, c.name)
			if res != nil || len(src.Queries()) != 0 {
				t.Errorf("result %#v, %d lookups; want none", res, len(src.Queries()))
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md section 6 - the use case needs a
// source; the zero use case and a missing source are invalid; an already
// cancelled ctx is returned unwrapped before any lookup.
// Path: unhappy (invalid input / cancellation).
func TestUseCaseConstructionAndCancellation(t *testing.T) {
	t.Parallel()
	if _, err := character.NewCalculateDerivedStats(); err == nil {
		t.Error("NewCalculateDerivedStats without a source returned no error")
	} else {
		wantInvalid(t, err, "no source")
	}
	if _, err := (character.CalculateDerivedStats{}).Run(context.Background(), wizardCommand(t)); err == nil {
		t.Error("zero use case ran")
	} else {
		wantInvalid(t, err, "zero use case")
	}
	uc, src := newUseCase(t, wizardClass(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := uc.Run(ctx, wizardCommand(t))
	if !errors.Is(err, context.Canceled) || res != nil || len(src.Queries()) != 0 {
		t.Errorf("cancelled Run = %#v, %v, %d lookups", res, err, len(src.Queries()))
	}
}

// panicSource panics on every lookup; nilSource answers with a nil result and a
// nil error; emptyFoundSource answers found with no entry; wrongClassSource
// answers a class key with a Shield.
type panicSource struct{}

func (panicSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	panic(errors.New("boom"))
}

type nilSource struct{}

func (nilSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	return nil, nil //nolint:nilnil // the contract breach under test
}

type emptyFoundSource struct{}

func (emptyFoundSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	return content.LookupFound{}, nil
}

type wrongClassSource struct{ entry content.Entry }

func (s wrongClassSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	return content.LookupFound{Entry: s.entry}, nil
}

// Requirement: doc/specs/derived-stats.md FR-044, FR-074 - a failing, panicking
// or contract-breaking source returns a *srderr.PortError for content.Port that
// still matches its cause, the sentinel for its class, and no result.
// Path: per case.
func TestRunPortFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("backend down")
	cases := []struct {
		name   string
		source character.ContentSource
		is     error
	}{
		{"source error", contenttest.NewFakeSource(wizardClass(t)).FailLookup(cause), cause},
		{"source panic", panicSource{}, srderr.ErrPortPanic},
		{"nil result", nilSource{}, srderr.ErrContractViolation},
		{"found without an entry", emptyFoundSource{}, srderr.ErrContractViolation},
		{"class key answered with another entry", wrongClassSource{entry: shieldOf(t)}, srderr.ErrContractViolation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, err := character.NewCalculateDerivedStats(character.WithSource(c.source))
			if err != nil {
				t.Fatal(err)
			}
			res, err := uc.Run(context.Background(), wizardCommand(t))
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != content.Port || !errors.Is(err, c.is) || srderr.ErrorType(err) != "content_error" {
				t.Errorf("Run = %#v, %v; want a content PortError matching %v", res, err, c.is)
			}
		})
	}
}

// Requirement: doc/specs/derived-stats.md FR-045 - the command is not altered and
// repeated calls give equal results.
// Path: unhappy (determinism, immutability).
func TestRunDoesNotAlterCommand(t *testing.T) {
	t.Parallel()
	uc, _ := newUseCase(t, wizardClass(t))
	cmd := wizardCommand(t)
	cmd.FlatArmorClassBonuses = []character.FlatBonus{itemBonus(t, 1)}
	cmd.HitPointMaximumBonuses = []character.HitPointMaximumBonus{character.MustHitPointMaximumBonus(1, 1, 1, "")}
	before := slices.Clone(cmd.FlatArmorClassBonuses)
	a, b := succeeded(t, run(t, uc, cmd)), succeeded(t, run(t, uc, cmd))
	if !slices.Equal(cmd.FlatArmorClassBonuses, before) || !a.ArmorClassBreakdown().Equal(b.ArmorClassBreakdown()) || a.HitPointMaximum() != b.HitPointMaximum() {
		t.Error("command altered or results differ")
	}
}
