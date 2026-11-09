package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	cPoisoned   = content.ConditionPoisoned
	cBlinded    = content.ConditionBlinded
	cStunned    = content.ConditionStunned
	cFrightened = content.ConditionFrightened
	cCharmed    = content.ConditionCharmed
	cProne      = content.ConditionProne
)

// Requirement: doc/specs/class-paladin.md FR-047 to FR-056, Table U (SRD 5.2.1, "Level 1: Lay On Hands", p. 54; "Level 14: Restoring Touch", p. 55) - LayOnHandsRules.Use accepts or rejects
// each of the 17 request rows with the stated remaining pool, the Hit Points restored and every reason in the fixed order invalid_amount, empty_request, condition_not_removable,
// restoring_touch_not_available, exceeds_pool.
// Path: per case (Table U): happy / unhappy (boundary, empty and minimal, invalid input, rule interaction).
func TestLayOnHandsRulesTableU(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	for _, c := range []struct {
		label                     string
		level, remaining, restore int
		remove                    []content.ConditionKind
		wantLeft                  int
		wantRestore               int
		reasons                   []string
	}{
		{"restore part", 5, 25, 10, nil, 15, 10, nil},
		{"restore all", 5, 25, 25, nil, 0, 25, nil},
		{"restore one too many", 5, 25, 26, nil, 0, 0, []string{"exceeds_pool"}},
		{"pool empty", 5, 0, 1, nil, 0, 0, []string{"exceeds_pool"}},
		{"cure Poisoned exactly", 1, 5, 0, []content.ConditionKind{cPoisoned}, 0, 0, nil},
		{"cure Poisoned one short", 1, 4, 0, []content.ConditionKind{cPoisoned}, 0, 0, []string{"exceeds_pool"}},
		{"Restoring Touch at 13", 13, 65, 0, []content.ConditionKind{cBlinded}, 0, 0, []string{"restoring_touch_not_available"}},
		{"Restoring Touch at 14", 14, 70, 0, []content.ConditionKind{cBlinded}, 65, 0, nil},
		{"two conditions", 14, 70, 0, []content.ConditionKind{cBlinded, cStunned}, 60, 0, nil},
		{"combined, exact", 14, 20, 5, []content.ConditionKind{cBlinded, cStunned}, 5, 5, nil},
		{"combined, one over", 14, 14, 0, []content.ConditionKind{cBlinded, cStunned, cFrightened}, 0, 0, []string{"exceeds_pool"}},
		{"Poisoned and Charmed", 14, 20, 0, []content.ConditionKind{cPoisoned, cCharmed}, 10, 0, nil},
		{"not removable", 14, 70, 0, []content.ConditionKind{cProne}, 0, 0, []string{"condition_not_removable"}},
		{"two reasons", 13, 4, 1, []content.ConditionKind{cProne, cBlinded}, 0, 0, []string{"condition_not_removable", "restoring_touch_not_available", "exceeds_pool"}},
		{"empty request", 5, 25, 0, nil, 0, 0, []string{"empty_request"}},
		{"negative amount", 5, 25, -1, nil, 0, 0, []string{"invalid_amount"}},
		{"negative and not removable", 5, 25, -1, []content.ConditionKind{cProne}, 0, 0, []string{"invalid_amount", "condition_not_removable"}},
	} {
		out, reasons := character.LayOnHandsRules{}.Use(p, character.MustPaladinLevel(c.level), c.remaining, character.LayOnHandsRequest{HitPoints: c.restore, Remove: c.remove})
		if !slices.Equal(pldReasons(reasons), c.reasons) && (len(reasons) != 0 || len(c.reasons) != 0) {
			t.Errorf("%s: reasons = %v, want %v", c.label, pldReasons(reasons), c.reasons)
			continue
		}
		if len(c.reasons) > 0 {
			if out.Cost() != 0 || out.HitPoints() != 0 || len(out.Conditions()) != 0 {
				t.Errorf("%s: a rejection carries an outcome %+v", c.label, out)
			}
			continue
		}
		if out.Remaining() != c.wantLeft || out.HitPoints() != c.wantRestore || out.Cost() != c.remaining-c.wantLeft || len(out.Conditions()) != len(c.remove) || out.ActionCost() != character.PaladinBonusAction {
			t.Errorf("%s: outcome = left %d, restore %d, cost %d, conditions %v", c.label, out.Remaining(), out.HitPoints(), out.Cost(), out.Conditions())
		}
	}
	// A condition named twice costs once, and the request slice is not altered.
	req := []content.ConditionKind{cPoisoned, cPoisoned}
	out, reasons := character.LayOnHandsRules{}.Use(p, character.MustPaladinLevel(1), 5, character.LayOnHandsRequest{Remove: req})
	if len(reasons) != 0 || out.Cost() != 5 || len(out.Conditions()) != 1 || len(req) != 2 {
		t.Errorf("duplicate condition = %+v, %v", out, reasons)
	}
	cs := out.Conditions()
	cs[0] = cProne
	if out.Conditions()[0] != cPoisoned {
		t.Error("Conditions returned shared storage")
	}
}

// Requirement: doc/specs/class-paladin.md FR-071 to FR-074, Tables B and W (SRD 5.2.1, "Level 3: Channel Divinity", "Level 9: Abjure Foes", p. 55; "Level 3: Sacred Weapon", p. 56) - the effect data of
// Divine Sense (Bonus Action, 10 minutes, 60 feet, Celestial/Fiend/Undead, location, creature type and consecrated places), Abjure Foes (Magic action, 1 minute, 60 feet, Wisdom save) and
// Sacred Weapon (10 minutes, Bright Light 20 feet and Dim Light 20 feet, Radiant) is returned as typed data.
// Path: happy.
func TestPaladinEffectData(t *testing.T) {
	t.Parallel()
	sense := character.DivineSenseEffect{}
	if sense.ActionCost() != character.PaladinBonusAction || sense.DurationMinutes() != 10 || sense.RadiusFeet() != 60 ||
		!slices.Equal(sense.CreatureTypes(), []content.CreatureType{content.CreatureTypeCelestial, content.CreatureTypeFiend, content.CreatureTypeUndead}) ||
		!slices.Equal(sense.Detects(), []character.DetectionKind{character.DetectLocation, character.DetectCreatureType, character.DetectConsecratedOrDesecrated}) {
		t.Errorf("Divine Sense data = %+v", sense)
	}
	for k, want := range map[character.DetectionKind]string{character.DetectLocation: "location", character.DetectCreatureType: "creature_type", character.DetectConsecratedOrDesecrated: "consecrated_or_desecrated", 0: "unknown"} {
		if k.String() != want {
			t.Errorf("DetectionKind %d = %q, want %q", k, k.String(), want)
		}
	}
	sacred := character.SacredWeaponEffect{}
	bright, brightFeet, dim, dimFeet := sacred.Light()
	if sacred.DurationMinutes() != 10 || bright != content.LightBright || brightFeet != 20 || dim != content.LightDim || dimFeet != 20 || !slices.Equal(sacred.DamageTypes(), []content.DamageType{content.DamageRadiant}) {
		t.Errorf("Sacred Weapon data = %+v", sacred)
	}
	if character.CoverGrant(2).String() != "unknown" || character.CoverGrant(-1).String() != "unknown" {
		t.Error("an undefined cover grant is named")
	}
	if character.CoverHalf.String() != "half" || character.CoverGrant(0).String() != "unknown" || !(character.SmiteOfProtectionEffect{}).UntilNextTurn() || (character.SmiteOfProtectionEffect{}).Cover() != character.CoverHalf {
		t.Error("cover data wrong")
	}
	if ab := (character.AbjureFoesEffect{}); ab.ActionCost() != character.PaladinMagicAction || ab.DurationMinutes() != 1 || ab.RadiusFeet() != 60 {
		t.Error("Abjure Foes data wrong")
	}
	for a, want := range map[character.PaladinActionCost]string{character.PaladinBonusAction: "bonus_action", character.PaladinMagicAction: "magic_action", character.PaladinNoAction: "no_action", 0: "unknown"} {
		if a.String() != want {
			t.Errorf("PaladinActionCost %d = %q, want %q", a, a.String(), want)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-077 to FR-083, Table AU (SRD 5.2.1, "Level 6: Aura of Protection", "Level 10: Aura of Courage", p. 55; "Level 7: Aura of Devotion", p. 56) -
// PaladinAuras.Compute gives each of the 14 rows its radius, save bonus and immunities, and an inactive aura with none while Incapacitated.
// Path: per case (Table AU): happy / unhappy (boundary, state error).
func TestPaladinAurasTableAU(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	both := []content.ConditionKind{cCharmed, cFrightened}
	for _, c := range []struct {
		label         string
		level         int
		oath          bool
		cha           int
		incapacitated bool
		active        bool
		radius, bonus int
		immunities    []content.ConditionKind
	}{
		{"before the feature", 5, false, 3, false, false, 0, 0, nil},
		{"first level", 6, false, 3, false, true, 10, 3, nil},
		{"modifier -1", 6, false, -1, false, true, 10, 1, nil},
		{"modifier 0", 6, false, 0, false, true, 10, 1, nil},
		{"modifier +1", 6, false, 1, false, true, 10, 1, nil},
		{"Charmed one level early", 6, true, 2, false, true, 10, 2, nil},
		{"Charmed", 7, true, 2, false, true, 10, 2, []content.ConditionKind{cCharmed}},
		{"Frightened one level early", 9, false, 2, false, true, 10, 2, nil},
		{"Frightened", 10, false, 2, false, true, 10, 2, []content.ConditionKind{cFrightened}},
		{"both", 10, true, 2, false, true, 10, 2, both},
		{"Incapacitated", 10, true, 2, true, false, 0, 0, nil},
		{"last small radius", 17, true, 4, false, true, 10, 4, both},
		{"Aura Expansion", 18, true, 4, false, true, 30, 4, both},
		{"maximum level", 20, true, 5, false, true, 30, 5, both},
	} {
		a := character.PaladinAuras{}.Compute(p, character.MustPaladinLevel(c.level), c.oath, modifierOf(t, c.cha), c.incapacitated)
		if a.Active() != c.active || a.RadiusFeet() != c.radius || a.SaveBonus() != c.bonus || (!slices.Equal(a.Immunities(), c.immunities) && (len(a.Immunities()) != 0 || len(c.immunities) != 0)) {
			t.Errorf("%s: aura = active %v, %d ft, +%d, %v; want %v, %d ft, +%d, %v", c.label, a.Active(), a.RadiusFeet(), a.SaveBonus(), a.Immunities(), c.active, c.radius, c.bonus, c.immunities)
		}
	}
	if a := (character.AuraEffects{}); a.Active() || a.RadiusFeet() != 0 || len(a.Immunities()) != 0 {
		t.Error("the zero aura is active")
	}
	got := character.PaladinAuras{}.Compute(p, character.MustPaladinLevel(20), true, modifierOf(t, 3), false).Immunities()
	got[0] = cProne
	if (character.PaladinAuras{}).Compute(p, character.MustPaladinLevel(20), true, modifierOf(t, 3), false).Immunities()[0] != cCharmed {
		t.Error("Immunities returned shared storage")
	}
}

// Requirement: doc/specs/class-paladin.md FR-081 (SRD 5.2.1, "Level 6: Aura of Protection", p. 55) - a creature benefits from only one Aura of Protection: Choose returns the chosen bonus, no
// candidates give no bonus, and a chosen Paladin that is not a candidate is not_a_candidate.
// Path: unhappy (rule interaction, empty case, invalid input).
func TestPaladinAurasChoose(t *testing.T) {
	t.Parallel()
	offers := []character.AuraOffer{{Paladin: "a", SaveBonus: 3}, {Paladin: "b", SaveBonus: 5}}
	for _, c := range []struct {
		offers []character.AuraOffer
		chosen string
		bonus  int
		reason character.PaladinReason
	}{{offers, "a", 3, 0}, {offers, "b", 5, 0}, {offers, "c", 0, character.PaladinNotACandidate}, {nil, "a", 0, 0}} {
		if bonus, reason := (character.PaladinAuras{}).Choose(c.offers, c.chosen); bonus != c.bonus || reason != c.reason {
			t.Errorf("Choose(%v, %q) = %d, %v; want %d, %v", c.offers, c.chosen, bonus, reason, c.bonus, c.reason)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-085 to FR-087, Table X (SRD 5.2.1, "Level 5: Extra Attack", "Level 11: Radiant Strikes", p. 55; "Weapon Properties", "Thrown", p. 90) -
// AttacksPerAttackAction is 1 at level 4 and 2 from 5; RadiantStrikes.AppliesTo gives each of the six rows of Table X (a thrown Javelin is a Melee weapon), and the extra damage is 1d8
// Radiant.
// Path: per case (Table X): happy / unhappy (boundary, rule interaction).
func TestPaladinExtraAttackAndRadiantStrikes(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	for _, c := range []struct{ level, want int }{{1, 1}, {4, 1}, {5, 2}, {20, 2}} {
		if got := character.AttacksPerAttackAction(p, character.MustPaladinLevel(c.level)); got != c.want {
			t.Errorf("AttacksPerAttackAction(%d) = %d, want %d", c.level, got, c.want)
		}
	}
	for _, c := range []struct {
		label string
		level int
		kind  character.PaladinAttackKind
		want  bool
	}{
		{"below the feature", 10, character.PaladinAttackKindForWeapon(content.WeaponMelee), false},
		{"Melee weapon", 11, character.PaladinAttackKindForWeapon(content.WeaponMelee), true},
		{"Unarmed Strike", 11, character.PaladinUnarmedStrike, true},
		{"thrown Javelin", 11, character.PaladinAttackKindForWeapon(content.WeaponMelee), true},
		{"Ranged weapon", 11, character.PaladinAttackKindForWeapon(content.WeaponRanged), false},
		{"spell attack", 11, character.PaladinSpellAttack, false},
		{"undefined kind", 11, 0, false},
	} {
		if got := (character.RadiantStrikes{}).AppliesTo(p, character.MustPaladinLevel(c.level), c.kind); got != c.want {
			t.Errorf("%s: AppliesTo = %v, want %v", c.label, got, c.want)
		}
	}
	if character.PaladinAttackKindForWeapon(content.WeaponMelee) == 0 || character.PaladinAttackKindForWeapon(content.WeaponRanged) == 0 || character.PaladinSpellAttack == 0 || character.PaladinUnarmedStrike == 0 {
		t.Error("a defined attack kind is the zero kind")
	}
	if character.PaladinAttackKindForWeapon(0) != 0 || character.PaladinAttackKindForWeapon(content.WeaponRanged) != character.PaladinRangedWeapon {
		t.Error("PaladinAttackKindForWeapon wrong")
	}
	expr, typ := (character.RadiantStrikes{}).ExtraDamage()
	if expr.String() != "1d8" || typ != content.DamageRadiant {
		t.Errorf("ExtraDamage = %v %v, want 1d8 Radiant", expr, typ)
	}
}

// Requirement: doc/specs/class-paladin.md FR-076 (SRD 5.2.1, "Divine Sense", p. 55; "Level 3: Sacred Weapon", p. 56; "Level 20: Holy Nimbus", p. 57) - EndPaladinEffect ends an active effect for a
// reason it has (Divine Sense: elapsed, incapacitated; Sacred Weapon: elapsed, ended early, not carried, used again; Holy Nimbus: elapsed, ended early), emits PaladinEffectEnded and
// refunds no Channel Divinity use; an effect that is not active is not_active and a reason the effect lacks is reason_not_applicable, both with the state unchanged.
// Path: per case (unhappy: state error, boundary).
func TestEndPaladinEffect(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	active := pldState(t, 20, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 1, DivineSenseActive: true, SacredWeaponActive: "sword", HolyNimbusActive: true})
	idle := pldFull(t, 20)
	for _, c := range []struct {
		effect character.PaladinEffectKind
		reason character.PaladinEndReason
		want   character.PaladinReason
	}{
		{character.PaladinEffectDivineSense, character.PaladinEndElapsed, 0}, {character.PaladinEffectDivineSense, character.PaladinEndIncapacitated, 0},
		{character.PaladinEffectDivineSense, character.PaladinEndEndedEarly, character.PaladinReasonNotApplicable}, {character.PaladinEffectDivineSense, character.PaladinEndUsedAgain, character.PaladinReasonNotApplicable},
		{character.PaladinEffectSacredWeapon, character.PaladinEndElapsed, 0}, {character.PaladinEffectSacredWeapon, character.PaladinEndEndedEarly, 0},
		{character.PaladinEffectSacredWeapon, character.PaladinEndNotCarried, 0}, {character.PaladinEffectSacredWeapon, character.PaladinEndUsedAgain, 0},
		{character.PaladinEffectSacredWeapon, character.PaladinEndIncapacitated, character.PaladinReasonNotApplicable},
		{character.PaladinEffectHolyNimbus, character.PaladinEndElapsed, 0}, {character.PaladinEffectHolyNimbus, character.PaladinEndEndedEarly, 0},
		{character.PaladinEffectHolyNimbus, character.PaladinEndNotCarried, character.PaladinReasonNotApplicable}, {character.PaladinEffectHolyNimbus, character.PaladinEndIncapacitated, character.PaladinReasonNotApplicable},
		{0, character.PaladinEndElapsed, character.PaladinNotActive}, {character.PaladinEffectDivineSense, 0, character.PaladinReasonNotApplicable},
	} {
		next, ev, reason := character.EndPaladinEffect(id, active, c.effect, c.reason)
		if reason != c.want {
			t.Errorf("end %v for %v = %v, want %v", c.effect, c.reason, reason, c.want)
			continue
		}
		if reason != 0 {
			if next != active || ev != nil {
				t.Errorf("end %v for %v changed the state or emitted %v", c.effect, c.reason, ev)
			}
			continue
		}
		ended, ok := ev.(character.PaladinEffectEnded)
		if !ok || ended.Effect() != c.effect || ended.Reason() != c.reason || ended.CharacterID() != id || next.ChannelDivinityRemaining() != 1 {
			t.Errorf("end %v for %v = %+v, state %+v", c.effect, c.reason, ev, next)
		}
		if again, _, r := character.EndPaladinEffect(id, next, c.effect, c.reason); r != character.PaladinNotActive || again != next {
			t.Errorf("ending %v twice = %v, want not_active", c.effect, r)
		}
	}
	if _, ev, r := character.EndPaladinEffect(id, idle, character.PaladinEffectSacredWeapon, character.PaladinEndElapsed); r != character.PaladinNotActive || ev != nil {
		t.Errorf("an idle Sacred Weapon = %v", r)
	}
	for _, effect := range []character.PaladinEffectKind{character.PaladinEffectDivineSense, character.PaladinEffectSacredWeapon, character.PaladinEffectHolyNimbus} {
		for _, reason := range []character.PaladinEndReason{0, 6, 99, -1} {
			if _, ev, r := character.EndPaladinEffect(id, active, effect, reason); r != character.PaladinReasonNotApplicable || ev != nil {
				t.Errorf("end %v for the undefined reason %d = %v, want reason_not_applicable", effect, reason, r)
			}
		}
	}
	if character.PaladinEffectSacredWeapon.String() != "sacred_weapon" || character.PaladinEffectKind(0).String() != "unknown" || character.PaladinEndNotCarried.String() != "not_carried" || character.PaladinEndReason(0).String() != "unknown" ||
		character.PaladinEffectDivineSense.String() != "divine_sense" || character.PaladinEffectHolyNimbus.String() != "holy_nimbus" || character.PaladinEndElapsed.String() != "elapsed" ||
		character.PaladinEndIncapacitated.String() != "incapacitated" || character.PaladinEndEndedEarly.String() != "ended_early" || character.PaladinEndUsedAgain.String() != "used_again" {
		t.Error("effect or end reason names wrong")
	}
}
