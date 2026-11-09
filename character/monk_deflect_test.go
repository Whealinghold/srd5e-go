package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

var (
	bludgeoning = []content.DamageType{content.DamageBludgeoning}
	fireOnly    = []content.DamageType{content.DamageFire}
)

// Requirement: doc/specs/class-monk.md Table 4.6a, FR-051 to FR-055, FR-062, FR-063 - the reduction is the d10 plus the Dexterity modifier plus the
// Monk level, never below 0, and the damage after never below 0; every row of the table: low roll, high roll, reduction equal to and one below the
// damage, reduction above it, higher level, a negative sum, mixed types, Fire only at levels 3, 12 and 13, and level 2.
// Path: per case (rows of Table 4.6a).
func TestDeflectAttacksTable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name             string
		level, dex, roll int
		total            int
		types            []content.DamageType
		reduction, after int
		unavailable      character.MonkReason
		canRedirect      bool
	}{
		{"low roll", 3, 3, 1, 20, bludgeoning, 7, 13, 0, false},
		{"high roll", 3, 3, 10, 20, []content.DamageType{content.DamagePiercing}, 16, 4, 0, false},
		{"reduction equals damage", 3, 3, 10, 16, []content.DamageType{content.DamageSlashing}, 16, 0, 0, true},
		{"reduction one below damage", 3, 3, 10, 17, []content.DamageType{content.DamageSlashing}, 16, 1, 0, false},
		{"reduction exceeds damage", 3, 3, 10, 10, []content.DamageType{content.DamageSlashing}, 16, 0, 0, true},
		{"higher level", 10, 4, 5, 30, bludgeoning, 19, 11, 0, false},
		{"negative sum", 3, -5, 1, 10, bludgeoning, 0, 10, 0, false},
		{"mixed types", 3, 3, 1, 20, []content.DamageType{content.DamageFire, content.DamageSlashing}, 7, 13, 0, false},
		{"Fire only, level 3", 3, 3, 5, 10, fireOnly, 0, 10, character.MonkNotEligible, false},
		{"Fire only, level 12", 12, 3, 5, 10, fireOnly, 0, 10, character.MonkNotEligible, false},
		{"Fire only, level 13", 13, 3, 5, 30, fireOnly, 21, 9, 0, false},
		{"level 2", 2, 3, 5, 10, []content.DamageType{content.DamageSlashing}, 0, 10, character.MonkLevelTooLow, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := mnkStats(t, c.level)
			wantReason(t, character.DeflectAttacks{}.Availability(stats, c.types), c.unavailable, c.name)
			res, err := character.DeflectAttacks{}.Resolve(stats, mnkAbilities(t, c.dex, 0, c.level), c.types, c.total, c.roll)
			if c.unavailable != 0 {
				wantInvalid(t, err, c.name)
				return
			}
			if err != nil || res.Reduction() != c.reduction || res.DamageAfter() != c.after || res.Rolled() != c.roll || res.CanRedirect() != c.canRedirect {
				t.Errorf("Resolve = %+v, %v; want reduction %d, after %d", res, err, c.reduction, c.after)
			}
		})
	}
	st := mnkStats(t, 3)
	for _, c := range []struct {
		name        string
		roll, total int
		types       []content.DamageType
	}{{"roll 0", 0, 5, bludgeoning}, {"roll 11", 11, 5, bludgeoning}, {"negative total", 5, -1, bludgeoning}, {"no types", 5, 5, nil}, {"an invalid type", 5, 5, []content.DamageType{0}}} {
		_, err := character.DeflectAttacks{}.Resolve(st, mnkAbilities(t, 3, 0, 3), c.types, c.total, c.roll)
		wantInvalid(t, err, c.name)
	}
	wantReason(t, character.DeflectAttacks{}.Availability(character.MonkLevelStats{}, bludgeoning), character.MonkNotAMonk, "zero stats")
}

func melee60(after int) character.MonkRedirectRequest {
	return character.MonkRedirectRequest{DamageAfter: after, Attack: character.MonkMeleeAttack, DamageType: content.DamageSlashing, Target: character.MonkRedirectTarget{Seen: true, DistanceFeet: 5}}
}

// Requirement: doc/specs/class-monk.md Table 4.6b, FR-056 to FR-061, FR-137, Table 4.11a - the redirect needs the damage reduced to 0, a seen target
// within 5 feet (melee) or within 60 feet and not behind Total Cover (ranged), and a Focus Point; it deals two Martial Arts rolls plus the Dexterity
// modifier on a failed save (never below 0) of the attack's type against the Monk's Focus save DC; rows: lowest and highest rolls, d8 tier, a negative
// modifier and a negative total.
// Path: per case (rows of Table 4.6b, unhappy: boundary, state error).
func TestDeflectRedirect(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name                       string
		level, dex, rolled, onFail int
	}{{"lowest rolls", 3, 3, 2, 5}, {"highest rolls", 3, 3, 12, 15}, {"d8 tier", 5, 2, 16, 18}, {"negative modifier", 3, -1, 2, 1}, {"negative total", 3, -5, 2, 0}} {
		for _, outcome := range []d20.Outcome{d20.Failure, d20.Success} {
			stats := mnkStats(t, c.level)
			in := mnkState(0, false, 0)
			res, err := character.DeflectRedirect{}.Resolve(stats, in, mnkAbilities(t, c.dex, 3, c.level), melee60(0), c.rolled, outcome)
			wantApplied := c.onFail
			if outcome == d20.Success {
				wantApplied = 0
			}
			if err != nil || res.DamageOnFailedSave() != c.onFail || res.DamageApplied() != wantApplied || res.DamageType() != content.DamageSlashing ||
				res.DifficultyClass() != d20.DifficultyClass(8+3+character.ProficiencyBonusFor(character.MustLevel(c.level)).Value()) || res.State().FocusExpended() != 1 {
				t.Errorf("%s/%v: Resolve = %+v, %v", c.name, outcome, res, err)
			}
			if in != mnkState(0, false, 0) {
				t.Error("Resolve altered its input")
			}
		}
	}
	r := character.DeflectRedirect{}
	st3, none := mnkStats(t, 3), mnkState(0, false, 0)
	ranged := func(dist int, cover bool) character.MonkRedirectRequest {
		return character.MonkRedirectRequest{Attack: character.MonkRangedAttack, DamageType: content.DamageFire, Target: character.MonkRedirectTarget{Seen: true, DistanceFeet: dist, BehindTotalCover: cover}}
	}
	for _, c := range []struct {
		name  string
		stats character.MonkLevelStats
		state character.MonkState
		req   character.MonkRedirectRequest
		want  character.MonkReason
	}{
		{"melee at 5 feet", st3, none, melee60(0), 0},
		{"melee at 6 feet", st3, none, character.MonkRedirectRequest{Attack: character.MonkMeleeAttack, DamageType: content.DamageFire, Target: character.MonkRedirectTarget{Seen: true, DistanceFeet: 6}}, character.MonkNotEligible},
		{"ranged at 60 feet", st3, none, ranged(60, false), 0},
		{"ranged at 61 feet", st3, none, ranged(61, false), character.MonkNotEligible},
		{"ranged behind Total Cover", st3, none, ranged(30, true), character.MonkNotEligible},
		{"target not seen", st3, none, character.MonkRedirectRequest{Attack: character.MonkMeleeAttack, DamageType: content.DamageFire}, character.MonkNotEligible},
		{"negative distance", st3, none, ranged(-1, false), character.MonkNotEligible},
		{"unknown range", st3, none, character.MonkRedirectRequest{DamageType: content.DamageFire, Target: character.MonkRedirectTarget{Seen: true}}, character.MonkNotEligible},
		{"damage left", st3, none, melee60(1), character.MonkNotEligible},
		{"damage left and no points", st3, mnkState(3, false, 0), melee60(1), character.MonkNotEligible},
		{"invalid damage type", st3, none, character.MonkRedirectRequest{Attack: character.MonkMeleeAttack, Target: character.MonkRedirectTarget{Seen: true}}, character.MonkNotEligible},
		{"no points", st3, mnkState(3, false, 0), melee60(0), character.MonkNotEnoughFocusPoints},
		{"level 2", mnkStats(t, 2), none, melee60(0), character.MonkLevelTooLow},
		{"zero stats", character.MonkLevelStats{}, none, melee60(0), character.MonkNotAMonk},
	} {
		wantReason(t, r.Availability(c.stats, c.state, c.req), c.want, c.name)
	}
	_, err := r.Resolve(st3, none, mnkAbilities(t, 3, 3, 3), melee60(0), 1, d20.Failure) // below two dice
	wantInvalid(t, err, "one die")
	_, err = r.Resolve(st3, none, mnkAbilities(t, 3, 3, 3), melee60(0), 13, d20.Failure)
	wantInvalid(t, err, "above two d6")
	_, err = r.Resolve(st3, none, mnkAbilities(t, 3, 3, 3), melee60(0), 7, 0)
	wantInvalid(t, err, "outcome")
	_, err = r.Resolve(st3, mnkState(3, false, 0), mnkAbilities(t, 3, 3, 3), melee60(0), 7, d20.Failure)
	wantInvalid(t, err, "unavailable")
}

// Requirement: doc/specs/class-monk.md FR-059 - the attack range names are snake_case and an undefined value is unknown.
// Path: per case.
func TestMonkAttackRangeNames(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.MonkAttackRange]string{character.MonkMeleeAttack: "melee", character.MonkRangedAttack: "ranged", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("range %d = %q, want %q", int(r), r.String(), want)
		}
	}
}

// Requirement: doc/specs/class-monk.md FR-051, FR-052, FR-062, FR-054 - only Bludgeoning, Piercing and Slashing qualify below level 13 and every one of the 13 damage types from level 13
// (table 4.6a), alone and with an invalid type or none at all; a zero damage total and a zero reduction are valid; a redirect with a target at distance 0, 3 or 5 (melee) or 0, 30 or 60 (ranged) is in
// range, a negative damage left is refused and a single remaining Focus Point is enough.
// Path: per case (unhappy: boundary, rule interaction).
func TestDeflectAttacksEveryDamageType(t *testing.T) {
	t.Parallel()
	physical := map[content.DamageType]bool{content.DamageBludgeoning: true, content.DamagePiercing: true, content.DamageSlashing: true}
	for dt := content.DamageAcid; dt <= content.DamageThunder; dt++ {
		types := []content.DamageType{dt}
		wantReason(t, character.DeflectAttacks{}.Availability(mnkStats(t, 3), types), map[bool]character.MonkReason{true: 0, false: character.MonkNotEligible}[physical[dt]], dt.String()+" at level 3")
		wantReason(t, character.DeflectAttacks{}.Availability(mnkStats(t, 13), types), 0, dt.String()+" at level 13")
		wantReason(t, character.DeflectAttacks{}.Availability(mnkStats(t, 14), types), 0, dt.String()+" at level 14")
	}
	for _, level := range []int{13, 20} {
		wantReason(t, character.DeflectAttacks{}.Availability(mnkStats(t, level), nil), character.MonkNotEligible, "no types")
		wantReason(t, character.DeflectAttacks{}.Availability(mnkStats(t, level), []content.DamageType{0}), character.MonkNotEligible, "an invalid type")
		wantReason(t, character.DeflectAttacks{}.Availability(mnkStats(t, level), []content.DamageType{content.DamageFire, 0}), character.MonkNotEligible, "a valid and an invalid type")
	}
	res, err := character.DeflectAttacks{}.Resolve(mnkStats(t, 3), mnkAbilities(t, 3, 0, 3), bludgeoning, 0, 4)
	if err != nil || res.DamageAfter() != 0 || !res.CanRedirect() {
		t.Errorf("a zero total = %+v, %v", res, err)
	}
	if res, err := (character.DeflectAttacks{}).Resolve(mnkStats(t, 3), mnkAbilities(t, -5, 0, 3), bludgeoning, 10, 2); err != nil || res.Reduction() != 0 {
		t.Errorf("a reduction of exactly 0 = %+v, %v", res, err)
	}
	r := character.DeflectRedirect{}
	st3 := mnkStats(t, 3)
	for _, dist := range []int{0, 3, 5} {
		req := melee60(0)
		req.Target.DistanceFeet = dist
		wantReason(t, r.Availability(st3, mnkState(0, false, 0), req), 0, "melee target in range")
	}
	for _, dist := range []int{0, 30, 60} {
		req := character.MonkRedirectRequest{Attack: character.MonkRangedAttack, DamageType: content.DamageFire, Target: character.MonkRedirectTarget{Seen: true, DistanceFeet: dist}}
		wantReason(t, r.Availability(st3, mnkState(0, false, 0), req), 0, "ranged target in range")
	}
	wantReason(t, r.Availability(st3, mnkState(0, false, 0), melee60(-1)), character.MonkNotEligible, "negative damage left")
	wantReason(t, r.Availability(st3, mnkState(2, false, 0), melee60(0)), 0, "the last point")
	for _, bad := range []d20.Outcome{0, 3, -1} {
		_, err := r.Resolve(st3, mnkState(0, false, 0), mnkAbilities(t, 3, 3, 3), melee60(0), 7, bad)
		wantInvalid(t, err, "outcome")
	}
}
