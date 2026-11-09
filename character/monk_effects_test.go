package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/class-monk.md FR-024, FR-025 - a weapon is a Monk weapon when it is a Simple Melee weapon or a Martial Melee weapon with
// the Light property; Simple and Martial Ranged weapons and a Martial Melee weapon without Light are not.
// Path: per case (happy, unhappy: rule interaction).
func TestIsMonkWeapon(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	for name, want := range map[string]bool{
		"club": true, "spear": true, "dagger": true, "scimitar": true, "shortsword": true,
		"shortbow": false, "light_crossbow": false, "hand_crossbow": false, "longsword": false, "greatsword": false, "rapier": false, "whip": false,
	} {
		if got := character.IsMonkWeapon(ws[name]); got != want {
			t.Errorf("IsMonkWeapon(%s) = %v, want %v", name, got, want)
		}
	}
	if character.IsMonkWeapon(content.Weapon{}) {
		t.Error("the zero weapon is not a Monk weapon")
	}
}

// Requirement: doc/specs/class-monk.md Table 4.3a, FR-026 to FR-030 - the Martial Arts offers apply only while the Monk is unarmed or wields only Monk
// weapons, wears no armor and wields no Shield: the nine rows of the table; the offers give the Bonus Unarmed Strike, the Martial Arts die of the level
// as an option, and Dexterity for attacks and for the Grapple and Shove save DC.
// Path: per case (rows of Table 4.3a).
func TestMonkMartialArtsTable(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	club, spear, greataxe := ws["club"], ws["spear"], ws["greataxe"]
	for _, c := range []struct {
		name          string
		weapons       []content.Weapon
		armor, shield bool
		applies       bool
	}{
		{"unarmed", nil, false, false, true}, {"one Monk weapon", []content.Weapon{club}, false, false, true},
		{"two Monk weapons", []content.Weapon{club, spear}, false, false, true},
		{"Monk weapon and other weapon", []content.Weapon{club, greataxe}, false, false, false},
		{"non-Monk weapon only", []content.Weapon{greataxe}, false, false, false},
		{"armor", nil, true, false, false}, {"Shield", nil, false, true, false},
		{"Monk weapon and Shield", []content.Weapon{club}, false, true, false}, {"armor and Monk weapon", []content.Weapon{club}, true, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			o := character.MonkEffects{}.MartialArts(mnkStats(t, 5), character.MonkLoadout{Weapons: c.weapons, ArmorWorn: c.armor, ShieldWielded: c.shield})
			wantDie := dice.Die{}
			if c.applies {
				wantDie = dice.D8()
			}
			if o.Applies() != c.applies || o.BonusUnarmedStrike() != c.applies || o.DexterityForAttacks() != c.applies || o.DexterityForGrappleAndShoveDC() != c.applies || o.MartialArtsDie() != wantDie {
				t.Errorf("offers = %+v, want applies %v with %v", o, c.applies, wantDie)
			}
		})
	}
	if o := (character.MonkEffects{}).MartialArts(character.MonkLevelStats{}, character.MonkLoadout{}); o.Applies() {
		t.Error("zero stats are offered nothing")
	}
}

// Requirement: doc/specs/class-monk.md FR-031, FR-032 - Unarmored Defense is the AbilityBased basis with the named ability set {Wisdom}, no Shield allowed
// and the id monk_unarmored_defense, the same at every Monk level; the arithmetic is derived-stats.md's.
// Path: per case (unhappy: rule interaction).
func TestMonkUnarmoredDefenseBasis(t *testing.T) {
	t.Parallel()
	b := character.MonkUnarmoredDefenseBasis()
	if b.ID() != "monk_unarmored_defense" || !slices.Equal(b.Abilities(), []ability.Ability{ability.Wisdom}) || b.ShieldAllowed() {
		t.Errorf("basis = %+v, want Wisdom, no Shield, monk_unarmored_defense", b)
	}
	if b.BaseValue() != 10 || b.IsZero() {
		t.Errorf("base value = %d, want 10", b.BaseValue())
	}
}

// Requirement: doc/specs/class-monk.md FR-044, FR-045, FR-075, FR-082, FR-086, FR-093, FR-094 - Unarmored Movement is the bonus of the level only without armor and Shield (0
// ft at level 1); the Attack action allows two attacks from level 5 (0 for zero stats); Empowered Strikes from 6; Acrobatic Movement from 9
// unarmored; forgoing food and drink gives no Exhaustion from 10; all saving throw proficiency from 14 (boundaries at 13 and 14).
// Path: per case (unhappy: boundary, rule interaction).
func TestMonkEffectsByLevel(t *testing.T) {
	t.Parallel()
	e := character.MonkEffects{}
	feet := map[int]int{1: 0, 2: 10, 5: 10, 6: 15, 10: 20, 14: 25, 18: 30, 20: 30}
	for n, want := range feet {
		if got := e.UnarmoredMovement(mnkStats(t, n), false, false); got != want {
			t.Errorf("Unarmored Movement at %d = %d, want %d", n, got, want)
		}
		if e.UnarmoredMovement(mnkStats(t, n), true, false) != 0 || e.UnarmoredMovement(mnkStats(t, n), false, true) != 0 {
			t.Errorf("Unarmored Movement at %d must be 0 with armor or a Shield", n)
		}
	}
	for n := 1; n <= 20; n++ {
		s := mnkStats(t, n)
		if got, want := e.AttacksPerAttackAction(s), map[bool]int{true: 2, false: 1}[n >= 5]; got != want {
			t.Errorf("attacks at %d = %d, want %d", n, got, want)
		}
		if e.EmpoweredStrikes(s) != (n >= 6) || e.AcrobaticMovement(s, false, false) != (n >= 9) || e.ForgoingFoodAndDrinkGivesExhaustion(s) != (n < 10) || e.ProficientInAllSavingThrows(s) != (n >= 14) {
			t.Errorf("level %d: flags wrong", n)
		}
		if e.AcrobaticMovement(s, true, false) || e.AcrobaticMovement(s, false, true) {
			t.Errorf("Acrobatic Movement at %d with armor or a Shield", n)
		}
	}
	zero := character.MonkLevelStats{}
	if e.AttacksPerAttackAction(zero) != 0 || e.EmpoweredStrikes(zero) || e.AcrobaticMovement(zero, false, false) || e.ProficientInAllSavingThrows(zero) || e.UnarmoredMovement(zero, false, false) != 0 {
		t.Error("zero stats must give no effect")
	}
}

// Requirement: doc/specs/class-monk.md Table 4.8a, FR-072, FR-073, FR-074 - Slow Fall reduces fall damage by five times the Monk level, never below 0: the
// six rows (level 4 with 30, 20, 10 and 0 damage, level 20 with 120, level 3 not available); a negative damage is an error.
// Path: per case (rows of Table 4.8a).
func TestMonkSlowFall(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                    string
		level, fall, red, after int
		rejected                character.MonkReason
	}{
		{"lowest level", 4, 30, 20, 10, 0}, {"reduction equals damage", 4, 20, 20, 0, 0}, {"reduction exceeds damage", 4, 10, 20, 0, 0},
		{"zero fall damage", 4, 0, 20, 0, 0}, {"top level", 20, 120, 100, 20, 0}, {"below level 4", 3, 30, 0, 30, character.MonkLevelTooLow},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.MonkEffects{}.SlowFall(mnkStats(t, c.level), c.fall)
			if err != nil {
				t.Fatal(err)
			}
			if c.rejected != 0 {
				if r, ok := got.(character.SlowFallRejected); !ok || r.Reason != c.rejected {
					t.Errorf("SlowFall = %#v, want rejected %v", got, c.rejected)
				}
				return
			}
			if a, ok := got.(character.SlowFallApplied); !ok || a.Reduction != c.red || a.DamageAfter != c.after {
				t.Errorf("SlowFall = %#v, want reduction %d, after %d", got, c.red, c.after)
			}
		})
	}
	_, err := character.MonkEffects{}.SlowFall(mnkStats(t, 4), -1)
	wantInvalid(t, err, "negative fall damage")
	if r, ok := must(character.MonkEffects{}.SlowFall(character.MonkLevelStats{}, 5)).(character.SlowFallRejected); !ok || r.Reason != character.MonkNotAMonk {
		t.Errorf("zero stats = %#v, want notAMonk", r)
	}
}

// Requirement: doc/specs/class-monk.md Table 4.8b, FR-083, FR-084, FR-085 - Evasion takes no damage on a successful Dexterity save and half on a failed one;
// it leaves the effect's own result when the Monk is Incapacitated, the save is another ability or the effect has no half-damage clause; below level 7 it is
// rejected with levelTooLow; invalid facts are an error.
// Path: per case (rows of Table 4.8b).
func TestMonkEvasion(t *testing.T) {
	t.Parallel()
	facts := func(save ability.Ability, o d20.Outcome, half, incap bool) character.MonkEvasionFacts {
		return character.MonkEvasionFacts{Save: save, Outcome: o, HalfDamageOnSuccess: half, Incapacitated: incap, Damage: 20}
	}
	for _, c := range []struct {
		name  string
		level int
		in    character.MonkEvasionFacts
		want  character.EvasionResult
	}{
		{"success", 7, facts(ability.Dexterity, d20.Success, true, false), character.EvasionApplied{DamageTaken: 0}},
		{"failure", 7, facts(ability.Dexterity, d20.Failure, true, false), character.EvasionApplied{DamageTaken: 10}},
		{"Incapacitated success", 7, facts(ability.Dexterity, d20.Success, true, true), character.EvasionUnchanged{}},
		{"Incapacitated failure", 7, facts(ability.Dexterity, d20.Failure, true, true), character.EvasionUnchanged{}},
		{"other save", 7, facts(ability.Constitution, d20.Success, true, false), character.EvasionUnchanged{}},
		{"no half clause", 7, facts(ability.Dexterity, d20.Success, false, false), character.EvasionUnchanged{}},
		{"below level 7", 6, facts(ability.Dexterity, d20.Success, true, false), character.EvasionRejected{Reason: character.MonkLevelTooLow}},
	} {
		got, err := character.MonkEffects{}.Evasion(mnkStats(t, c.level), c.in)
		if err != nil || got != c.want {
			t.Errorf("%s: Evasion = %#v, %v; want %#v", c.name, got, err, c.want)
		}
	}
	bad := facts(ability.Dexterity, d20.Success, true, false)
	for _, f := range []character.MonkEvasionFacts{{Save: bad.Save, Outcome: bad.Outcome, Damage: -1}, {Save: 0, Outcome: d20.Success}, {Save: bad.Save}} {
		_, err := character.MonkEffects{}.Evasion(mnkStats(t, 7), f)
		wantInvalid(t, err, "invalid facts")
	}
	if got := must(character.MonkEffects{}.Evasion(character.MonkLevelStats{}, bad)); got != (character.EvasionRejected{Reason: character.MonkNotAMonk}) {
		t.Errorf("zero stats = %#v", got)
	}
}

// Requirement: doc/specs/class-monk.md FR-091, FR-092 - Self-Restoration offers, from level 10, only the Charmed, Frightened and Poisoned conditions on the Monk
// (in condition order, a new slice), never another, and an empty list when none is on the Monk; below level 10 it is rejected.
// Path: per case (unhappy: rule interaction, boundary).
func TestMonkSelfRestoration(t *testing.T) {
	t.Parallel()
	on := []content.ConditionKind{content.ConditionPoisoned, content.ConditionBlinded, content.ConditionCharmed, content.ConditionStunned}
	got, ok := character.MonkEffects{}.SelfRestoration(mnkStats(t, 10), on).(character.SelfRestorationOffered)
	if !ok || !slices.Equal(got.Removable(), []content.ConditionKind{content.ConditionCharmed, content.ConditionPoisoned}) {
		t.Fatalf("offer = %#v", got)
	}
	got.Removable()[0] = 0
	if got.Removable()[0] == 0 {
		t.Error("Removable returned a shared slice")
	}
	none, ok := character.MonkEffects{}.SelfRestoration(mnkStats(t, 10), []content.ConditionKind{content.ConditionBlinded}).(character.SelfRestorationOffered)
	if !ok || none.Removable() == nil || len(none.Removable()) != 0 {
		t.Errorf("no removable condition = %#v, want an empty non-nil list", none)
	}
	if r, ok := (character.MonkEffects{}).SelfRestoration(mnkStats(t, 9), on).(character.SelfRestorationRejected); !ok || r.Reason != character.MonkLevelTooLow {
		t.Errorf("level 9 = %#v, want levelTooLow", r)
	}
	if r, ok := (character.MonkEffects{}).SelfRestoration(character.MonkLevelStats{}, on).(character.SelfRestorationRejected); !ok || r.Reason != character.MonkNotAMonk {
		t.Errorf("zero stats = %#v, want notAMonk", r)
	}
}

// Requirement: doc/specs/class-monk.md Table 4.9b, FR-103, FR-104 - Body and Mind raises Dexterity and Wisdom by 4 to a maximum of 25 at level 20 and never
// lowers a score above 25: the five rows; below level 20 it is rejected; a zero score is an error.
// Path: per case (rows of Table 4.9b).
func TestMonkBodyAndMind(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                       string
		dex, wis, wantDex, wantWis int
	}{{"well below cap", 16, 14, 20, 18}, {"exactly reaches cap", 21, 21, 25, 25}, {"already at cap", 25, 25, 25, 25}, {"one score capped", 22, 18, 25, 22}, {"above cap", 27, 20, 27, 24}} {
		got, err := character.MonkEffects{}.BodyAndMind(mnkStats(t, 20), score(t, c.dex), score(t, c.wis))
		applied, ok := got.(character.BodyAndMindApplied)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.name, got, err)
		}
		ch := applied.Changes()
		want := []character.MonkAbilityChange{{Ability: ability.Dexterity, NewScore: c.wantDex, Maximum: 25}, {Ability: ability.Wisdom, NewScore: c.wantWis, Maximum: 25}}
		if !slices.Equal(ch, want) {
			t.Errorf("%s: changes = %v, want %v", c.name, ch, want)
		}
		ch[0].NewScore = 0
		if applied.Changes()[0].NewScore == 0 {
			t.Error("Changes returned a shared slice")
		}
	}
	if r, ok := must((character.MonkEffects{}).BodyAndMind(mnkStats(t, 19), score(t, 10), score(t, 10))).(character.BodyAndMindRejected); !ok || r.Reason != character.MonkLevelTooLow {
		t.Errorf("level 19 = %#v", r)
	}
	if r, ok := must((character.MonkEffects{}).BodyAndMind(character.MonkLevelStats{}, score(t, 10), score(t, 10))).(character.BodyAndMindRejected); !ok || r.Reason != character.MonkNotAMonk {
		t.Errorf("zero stats = %#v", r)
	}
	_, err := character.MonkEffects{}.BodyAndMind(mnkStats(t, 20), ability.Score{}, score(t, 10))
	wantInvalid(t, err, "zero Dexterity")
	_, err = character.MonkEffects{}.BodyAndMind(mnkStats(t, 20), score(t, 10), ability.Score{})
	wantInvalid(t, err, "zero Wisdom")
}

// Requirement: doc/specs/class-monk.md FR-083, FR-085 - Evasion with 0 damage is valid (a failed save halves 0 to 0), and an outcome that is neither success nor failure (0, or a value above them)
// is an error.
// Path: per case (unhappy: boundary, invalid input).
func TestMonkEvasionBoundaries(t *testing.T) {
	t.Parallel()
	stats := mnkStats(t, 7)
	got, err := character.MonkEffects{}.Evasion(stats, character.MonkEvasionFacts{Save: ability.Dexterity, Outcome: d20.Failure, HalfDamageOnSuccess: true})
	if err != nil || got != (character.EvasionApplied{DamageTaken: 0}) {
		t.Errorf("0 damage = %#v, %v", got, err)
	}
	for _, bad := range []d20.Outcome{0, 3, -1} {
		_, err := character.MonkEffects{}.Evasion(stats, character.MonkEvasionFacts{Save: ability.Dexterity, Outcome: bad, HalfDamageOnSuccess: true, Damage: 10})
		wantInvalid(t, err, "outcome")
	}
}
