package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-fighter.md FR-084, FR-085, FR-086 - Studied Attacks is reported from Fighter level 13, not at 12, whatever the
// subclass, as data (trigger: missed attack roll; applies to: the next attack roll against the same creature; expires: end of the
// character's next turn), and the extra attacks are the attack count of the stats, not separate effects.
// Path: per case (unhappy: boundary).
func TestFighterEffectsStudiedAttacksAndAttacks(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 20; n++ {
		effect, ok := character.FighterEffects{}.StudiedAttacks(character.MustFighterLevel(n))
		if ok != (n >= 13) {
			t.Errorf("level %d: reported = %v, want %v", n, ok, n >= 13)
		}
		if ok && (effect.Trigger() != character.TriggerAttackRollMissed || effect.AppliesTo() != character.NextAttackRollAgainstSameCreature || effect.Expires() != character.ExpiresEndOfYourNextTurn) {
			t.Errorf("level %d: effect = %v %v %v", n, effect.Trigger(), effect.AppliesTo(), effect.Expires())
		}
		if got := (character.FighterEffects{}).AttacksPerAttackAction(fgtStats(t, n)); got != fgtStats(t, n).Attacks() {
			t.Errorf("level %d: attacks = %d, want the stats' %d", n, got, fgtStats(t, n).Attacks())
		}
	}
	if _, ok := (character.FighterEffects{}).StudiedAttacks(character.FighterLevel{}); ok {
		t.Error("the zero level must report nothing")
	}
	for v, name := range map[character.FighterEffectTrigger]string{character.TriggerAttackRollMissed: "attack_roll_missed", character.TriggerStartOfEachOfYourTurns: "start_of_each_of_your_turns", 0: "unknown"} {
		if v.String() != name {
			t.Errorf("trigger %d = %q, want %q", int(v), v.String(), name)
		}
	}
	for _, v := range []int{-1, 7} {
		if character.FighterEffectTarget(v).String() != "unknown" || character.FighterEffectExpiry(v).String() != "unknown" || character.FighterEffectTrigger(v).String() != "unknown" {
			t.Errorf("value %d must be unknown", v)
		}
	}
	if character.NextAttackRollAgainstSameCreature.String() != "next_attack_roll_against_same_creature" || character.FighterEffectTarget(0).String() != "unknown" ||
		character.ExpiresEndOfYourNextTurn.String() != "end_of_your_next_turn" || character.FighterEffectExpiry(0).String() != "unknown" {
		t.Error("effect target and expiry names are wrong")
	}
}

// Requirement: doc/specs/class-fighter.md FR-049, FR-050, FR-125 - Tactical Master offers exactly Push, Sap and Slow (not Topple) from
// Fighter level 9 when the access permits the weapon kind, without a subclass too; below level 9 it is featureNotGained (checked first);
// a kind not chosen, NoMasteryFeature and a nil access are masteryNotUsable.
// Path: per case (unhappy: boundary, state error).
func TestTacticalMasterOptions(t *testing.T) {
	t.Parallel()
	access := character.MasteryFromChoice{Choice: must(createMasteries(t, "greatsword", "longbow", "mace"))}
	for _, c := range []struct {
		name   string
		level  int
		access character.WeaponMasteryAccess
		weapon string
		want   character.FighterReason
	}{
		{"level 8", 8, access, "greatsword", character.FighterFeatureNotGained},
		{"level 8 before the access", 8, character.NoMasteryFeature{}, "greatsword", character.FighterFeatureNotGained},
		{"a kind not chosen", 9, access, "dagger", character.FighterMasteryNotUsable},
		{"no mastery feature", 9, character.NoMasteryFeature{}, "greatsword", character.FighterMasteryNotUsable},
		{"a nil access", 9, nil, "greatsword", character.FighterMasteryNotUsable},
		{"a chosen kind at level 9", 9, access, "greatsword", 0},
		{"a chosen kind at level 20", 20, access, "mace", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.FighterEffects{}.TacticalMasterOptions(character.MustFighterLevel(c.level), c.access, content.ItemID(c.weapon))
			if c.want == 0 {
				offered, ok := got.(character.TacticalMasterOffered)
				want := []content.MasteryProperty{content.MasteryPush, content.MasterySap, content.MasterySlow}
				if !ok || !slices.Equal(offered.Replacements(), want) {
					t.Errorf("= %v, want Push, Sap, Slow", got)
				}
				r := offered.Replacements()
				r[0] = content.MasteryTopple
				if offered.Replacements()[0] != content.MasteryPush {
					t.Error("Replacements returned a shared slice")
				}
				return
			}
			if un, ok := got.(character.TacticalMasterUnavailable); !ok || un.Reason != c.want {
				t.Errorf("= %v, want unavailable %v", got, c.want)
			}
		})
	}
}

// createMasteries builds a Weapon Mastery choice of the named weapon fixtures with the shared rules.
func createMasteries(t *testing.T, names ...string) (character.WeaponMasteryChoice, error) {
	t.Helper()
	res, err := character.CreateWeaponMasteryChoice(ids(names...), resolvedOf(weaponFixtures(t), names...), len(names), character.SimpleOrMartialEligibility{})
	if err != nil {
		return character.WeaponMasteryChoice{}, err
	}
	return res.(character.WeaponMasteryChoiceAccepted).Choice(), nil
}

// Requirement: doc/specs/class-fighter.md FR-164 - MasteryAccess is NoMasteryFeature for a character with no Fighter levels or with no
// Weapon Mastery choice stored, and MasteryFromChoice with the stored choice otherwise; another type of state under the name is an error.
// Path: per case (unhappy: state error, invalid input).
func TestFighterEffectsMasteryAccess(t *testing.T) {
	t.Parallel()
	choice := must(createMasteries(t, "greatsword", "longbow", "mace"))
	stored := must(character.WithFeatureState(fgtFighter(t, 3), character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(choice)))
	storedWizard := must(character.WithFeatureState(fgtChar(t, cl("wizard", 3)), character.WeaponMasteryChoiceKey(), character.NewWeaponMasteryChoiceState(choice)))
	foreign := must(character.WithFeatureState(fgtFighter(t, 3), character.MustFeatureStateKey[character.ChosenSubclass]("weapon_mastery_choice"), must(character.NewChosenSubclass("champion"))))
	for _, c := range []struct {
		name string
		char character.Character
		want character.WeaponMasteryAccess
		err  bool
	}{
		{"no Fighter levels, a stored choice", storedWizard, character.NoMasteryFeature{}, false},
		{"a Fighter with no choice", fgtFighter(t, 3), character.NoMasteryFeature{}, false},
		{"a Fighter with a choice", stored, character.MasteryFromChoice{Choice: choice}, false},
		{"another type of state under the name", foreign, nil, true},
	} {
		got, err := character.FighterEffects{}.MasteryAccess(c.char.ClassLevels(), c.char.FeatureStates())
		if c.err {
			wantInvalid(t, err, c.name)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		_, wantNone := c.want.(character.NoMasteryFeature)
		_, gotNone := got.(character.NoMasteryFeature)
		if wantNone != gotNone || (!wantNone && !got.(character.MasteryFromChoice).Choice.Equal(choice)) {
			t.Errorf("%s: = %v, want %v", c.name, got, c.want)
		}
	}
}
