package character_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var warID = character.MustIdentifier("pc-1")

func warChanged(t *testing.T, tr character.WarlockTransition) character.WarlockChanged {
	t.Helper()
	c, ok := tr.(character.WarlockChanged)
	if !ok {
		t.Fatalf("transition = %#v, want WarlockChanged", tr)
	}
	return c
}

func warRefused(t *testing.T, tr character.WarlockTransition) character.WarlockRejection {
	t.Helper()
	r, ok := tr.(character.WarlockRefused)
	if !ok {
		t.Fatalf("transition = %#v, want WarlockRefused", tr)
	}
	return r.Rejection()
}

// Requirement: doc/specs/class-warlock.md FR-105 to FR-109, FR-163, table 4.7.1 (SRD 5.2.1, "Level 2: Magical Cunning", "Level 20: Eldritch Master", p. 72) - for each row MC-01 to MC-10 the slots regained are the lesser of the slots expended and half the maximum rounded up (all at level 20), returned as a WarlockSlotChangeRegainPact for the host, with none when nothing was expended, and the use is spent in every case.
// Path: per case (MC-01 to MC-10).
func TestCompleteMagicalCunning(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                    string
		level, expended, regain int
	}{
		{"MC-01", 2, 2, 1}, {"MC-02", 2, 1, 1}, {"MC-03", 2, 0, 0}, {"MC-04", 10, 2, 1}, {"MC-05", 11, 3, 2}, {"MC-06", 11, 1, 1}, {"MC-07", 17, 4, 2}, {"MC-08", 19, 4, 2}, {"MC-09", 20, 4, 4}, {"MC-10", 20, 3, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			st := warCasting(t, c.level, c.expended)
			got := warChanged(t, character.CompleteMagicalCunning(warID, character.WarlockState{}, warProfile(c.level, ""), time.Minute, st))
			if !got.State().MagicalCunningUsed() {
				t.Error("the use must be spent")
			}
			ev := got.Events()
			if len(ev) != 1 || ev[0].(character.MagicalCunningUsed).SlotsRegained() != c.regain || ev[0].CharacterID() != warID {
				t.Errorf("events = %+v, want one MagicalCunningUsed regaining %d", ev, c.regain)
			}
			change, has := got.SlotChange()
			if c.regain == 0 && has || c.regain > 0 && (!has || change != character.WarlockSlotChange(character.WarlockSlotChangeRegainPact{Count: c.regain})) {
				t.Errorf("slot change = %v, %v; want regain %d", change, has, c.regain)
			}
		})
	}
}

// Requirement: doc/specs/class-warlock.md FR-105, FR-106, FR-108 - Magical Cunning is not gained at level 1; a rite of 59 seconds is refused with the required and given seconds and does not use it up while exactly 60 seconds works; a second use before a Long Rest is refused; a Long Rest restores it.
// Path: per case (boundary 59 and 60 seconds; level 1 and 2; state error).
func TestCompleteMagicalCunningRefusals(t *testing.T) {
	t.Parallel()
	st := warCasting(t, 2, 1)
	if r := warRefused(t, character.CompleteMagicalCunning(warID, character.WarlockState{}, warProfile(1, ""), time.Minute, warCasting(t, 1, 0))); r.Kind != character.WarlockFeatureNotGained || r.Feature != "magical_cunning" {
		t.Errorf("level 1 = %+v", r)
	}
	r := warRefused(t, character.CompleteMagicalCunning(warID, character.WarlockState{}, warProfile(2, ""), 59*time.Second, st))
	if r.Kind != character.WarlockRiteTooShort || r.Required != 60 || r.Given != 59 {
		t.Errorf("59 seconds = %+v", r)
	}
	used := warChanged(t, character.CompleteMagicalCunning(warID, character.WarlockState{}, warProfile(2, ""), 60*time.Second, st)).State()
	if r := warRefused(t, character.CompleteMagicalCunning(warID, used, warProfile(2, ""), time.Hour, st)); r.Kind != character.WarlockMagicalCunningAlreadyUsed {
		t.Errorf("second use = %+v", r)
	}
	// The used-up state is checked before the rite length.
	if r := warRefused(t, character.CompleteMagicalCunning(warID, used, warProfile(2, ""), time.Second, st)); r.Kind != character.WarlockMagicalCunningAlreadyUsed {
		t.Errorf("used and short = %+v", r)
	}
	rested := warChanged(t, must(character.ApplyWarlockRest(warID, used, character.LongRest))).State()
	warChanged(t, character.CompleteMagicalCunning(warID, rested, warProfile(2, ""), time.Minute, st))
	// A character with no Pact Magic slots regains none but still uses it.
	none := warChanged(t, character.CompleteMagicalCunning(warID, character.WarlockState{}, warProfile(2, ""), time.Minute, character.SpellcastingState{}))
	if _, has := none.SlotChange(); has || !none.State().MagicalCunningUsed() {
		t.Error("no Pact Magic slots: nothing to regain, the use still spent")
	}
}

// Requirement: doc/specs/class-warlock.md FR-115, FR-116, FR-165 (SRD 5.2.1, "Level 9: Contact Patron", p. 72) - Contact Patron is not gained at level 8 and gives Contact Other Plane with CastModeFeatureGranted at level 9; a second use is refused until a Long Rest restores it.
// Path: per case (boundary 8 and 9; state error).
func TestUseContactPatron(t *testing.T) {
	t.Parallel()
	if r := warRefused(t, character.UseContactPatron(warID, character.WarlockState{}, warProfile(8, ""))); r.Kind != character.WarlockFeatureNotGained || r.Feature != "contact_patron" {
		t.Errorf("level 8 = %+v", r)
	}
	got := warChanged(t, character.UseContactPatron(warID, character.WarlockState{}, warProfile(9, "")))
	g, has := got.SpellGrant()
	if !has || g.Spell != "contact_other_plane" || g.Mode != character.CastModeFeatureGranted || !got.State().ContactPatronUsed() {
		t.Errorf("grant = %+v, %v", g, has)
	}
	if ev := got.Events(); len(ev) != 1 || ev[0].CharacterID() != warID {
		t.Errorf("events = %+v", ev)
	} else if _, ok := ev[0].(character.ContactPatronUsed); !ok {
		t.Errorf("event = %T", ev[0])
	}
	if r := warRefused(t, character.UseContactPatron(warID, got.State(), warProfile(9, ""))); r.Kind != character.WarlockContactPatronAlreadyUsed {
		t.Errorf("second use = %+v", r)
	}
	rested := warChanged(t, must(character.ApplyWarlockRest(warID, got.State(), character.LongRest))).State()
	warChanged(t, character.UseContactPatron(warID, rested, warProfile(9, "")))
}

// Requirement: doc/specs/class-warlock.md FR-117, FR-118, FR-120, table 4.7.2 (SRD 5.2.1, "Level 11: Mystic Arcanum", p. 72) - an arcanum spell is a Warlock spell of a level the character has a slot for; level 9 at level 17 is accepted, a spell of another level is refused with the expected and actual level, a spell off the Warlock list is refused, no slot below level 11, one replacement per level gained and a second refused.
// Path: per case (MA-01 to MA-07; invalid input).
func TestChooseMysticArcanum(t *testing.T) {
	t.Parallel()
	kill, heal := warSpell(t, "power_word_kill"), warSpell(t, "power_word_heal")
	if r := warRefused(t, character.ChooseMysticArcanum(warID, character.WarlockState{}, warProfile(10, ""), kill, 0)); r.Kind != character.WarlockNoArcanumAtLevel {
		t.Errorf("level 10 = %+v", r)
	}
	r := warRefused(t, character.ChooseMysticArcanum(warID, character.WarlockState{}, warProfile(11, ""), kill, 0))
	if r.Kind != character.WarlockArcanumWrongSpellLevel || r.Spell != "power_word_kill" || r.ExpectedLevel != 6 || r.ActualLevel != 9 {
		t.Errorf("level 9 spell at level 11 = %+v", r)
	}
	if r := warRefused(t, character.ChooseMysticArcanum(warID, character.WarlockState{}, warProfile(17, ""), heal, 0)); r.Kind != character.WarlockArcanumNotOnWarlockList || r.Spell != "power_word_heal" {
		t.Errorf("a Bard and Cleric spell = %+v", r)
	}
	if r := warRefused(t, character.ChooseMysticArcanum(warID, character.WarlockState{}, warProfile(13, ""), kill, 0)); r.Kind != character.WarlockArcanumWrongSpellLevel || r.ExpectedLevel != 7 {
		t.Errorf("level 13 expects 7: %+v", r)
	}
	chosen := warChanged(t, character.ChooseMysticArcanum(warID, character.WarlockState{}, warProfile(17, ""), kill, 0))
	if a, ok := chosen.State().Arcanum(9); !ok || a != "power_word_kill" {
		t.Errorf("Arcanum(9) = %v, %v", a, ok)
	}
	ev := chosen.Events()
	if len(ev) != 1 || ev[0].(character.MysticArcanumChosen).SpellLevel() != 9 || ev[0].(character.MysticArcanumChosen).Spell() != "power_word_kill" || ev[0].(character.MysticArcanumChosen).Replaced() || ev[0].CharacterID() != warID {
		t.Errorf("events = %+v", ev)
	}
	again := warChanged(t, character.ChooseMysticArcanum(warID, chosen.State(), warProfile(17, ""), kill, 0))
	if !again.Events()[0].(character.MysticArcanumChosen).Replaced() {
		t.Error("choosing a filled slot again is a replacement")
	}
	if r := warRefused(t, character.ChooseMysticArcanum(warID, chosen.State(), warProfile(17, ""), kill, 1)); r.Kind != character.WarlockArcanumTooManyReplacements || r.Given != 2 {
		t.Errorf("second replacement = %+v", r)
	}
	// A first arcanum choice is not limited by the replacements already made this level.
	warChanged(t, character.ChooseMysticArcanum(warID, character.WarlockState{}, warProfile(17, ""), kill, 1))
}

// Requirement: doc/specs/class-warlock.md FR-119, FR-165 (SRD 5.2.1, "Level 11: Mystic Arcanum", p. 72) - an arcanum spell is cast once without a slot with CastModeFeatureGranted; a second cast is refused until a Long Rest; a spell that is not an arcanum spell (or the empty id) is refused.
// Path: per case (state error).
func TestCastMysticArcanum(t *testing.T) {
	t.Parallel()
	s := warState(warProfile(17, ""), character.WarlockStateParts{Arcanum: [4]content.SpellID{"circle_of_death", "", "", "power_word_kill"}})
	got := warChanged(t, character.CastMysticArcanum(warID, s, "power_word_kill"))
	g, has := got.SpellGrant()
	if !has || g.Spell != "power_word_kill" || g.Mode != character.CastModeFeatureGranted || !got.State().ArcanumCast(9) || got.State().ArcanumCast(6) {
		t.Errorf("grant = %+v, %v", g, has)
	}
	if ev := got.Events(); len(ev) != 1 || ev[0].(character.MysticArcanumCast).SpellLevel() != 9 || ev[0].(character.MysticArcanumCast).Spell() != "power_word_kill" || ev[0].CharacterID() != warID {
		t.Errorf("events = %+v", ev)
	}
	if r := warRefused(t, character.CastMysticArcanum(warID, got.State(), "power_word_kill")); r.Kind != character.WarlockArcanumAlreadyCast || r.Spell != "power_word_kill" {
		t.Errorf("second cast = %+v", r)
	}
	warChanged(t, character.CastMysticArcanum(warID, got.State(), "circle_of_death"))
	for _, spell := range []content.SpellID{"fireball", ""} {
		if r := warRefused(t, character.CastMysticArcanum(warID, s, spell)); r.Kind != character.WarlockNotAnArcanumSpell || r.Spell != spell {
			t.Errorf("%q = %+v", spell, r)
		}
	}
	rested := warChanged(t, must(character.ApplyWarlockRest(warID, got.State(), character.LongRest))).State()
	warChanged(t, character.CastMysticArcanum(warID, rested, "power_word_kill"))
}

// Requirement: doc/specs/class-warlock.md FR-043 to FR-053 - ChooseInvocations stores the selection (a Repeatable option once per time) and returns the one InvocationsChosen event with the picks and the character id.
// Path: happy.
func TestChooseInvocations(t *testing.T) {
	t.Parallel()
	picks := []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("agonizing_blast", "eldritch_blast", ""), warPick("agonizing_blast", "chill_touch", "")}
	sel, problems := character.SelectInvocations(character.MustWarlockLevel(3), 3, picks, nil, warOptions(t), []character.CantripFacts{warCantrips()["eldritch_blast"], warCantrips()["chill_touch"]})
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	// A previously known invocation not in the selection is dropped.
	prior := warKnows("eldritch_mind")
	got := warChanged(t, character.ChooseInvocations(warID, prior, sel))
	if want := []content.FeatureOptionID{"agonizing_blast", "agonizing_blast", "pact_of_the_blade"}; !slices.Equal(got.State().Invocations(), want) {
		t.Errorf("Invocations = %v, want %v", got.State().Invocations(), want)
	}
	ev := got.Events()
	if len(ev) != 1 || ev[0].CharacterID() != warID || !slices.Equal(ev[0].(character.InvocationsChosen).Picks(), picks) {
		t.Errorf("events = %+v", ev)
	}
}

// Requirement: doc/specs/class-warlock.md FR-073 to FR-076, FR-154, FR-163, table 4.6.4 (SRD 5.2.1, "Eldritch Smite", pp. 72-73) - a pact weapon hit expends one Pact Magic slot (WarlockSlotChangeExpendPact at the Slot Level) and deals (slot level + 1)d8 Force damage; Prone is optional, refused for a Gargantuan target and the slot and damage are kept when declined; the invocation, a pact weapon hit, the once per turn limit and a remaining slot are required.
// Path: per case (ESM-01 to ESM-06; state errors).
func TestUseEldritchSmite(t *testing.T) {
	t.Parallel()
	known := warKnows("pact_of_the_blade", "eldritch_smite")
	for _, c := range []struct{ level, slot int }{{1, 1}, {3, 2}, {5, 3}, {7, 4}, {9, 5}, {20, 5}} {
		got := warChanged(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, warCasting(t, c.level, 0), content.SizeMedium, character.SmiteDeclineProne))
		change, has := got.SlotChange()
		dmg, hasDmg := got.Damage()
		if !has || change != character.WarlockSlotChange(character.WarlockSlotChangeExpendPact{Level: c.slot}) || !hasDmg || dmg != dice.MustExpression(c.slot+1, dice.D8(), 0) {
			t.Errorf("level %d: change %v, damage %v", c.level, change, dmg)
		}
		if got.State() != known {
			t.Error("Eldritch Smite changes no Warlock state")
		}
		if e := got.Events()[0].(character.EldritchSmiteUsed); e.SlotLevel() != c.slot || e.Prone() || e.CharacterID() != warID {
			t.Errorf("event = %+v", e)
		}
	}
	st := warCasting(t, 9, 0)
	prone := warChanged(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, st, content.SizeHuge, character.SmiteApplyProne))
	if !prone.Events()[0].(character.EldritchSmiteUsed).Prone() {
		t.Error("Prone applied to a Huge target must be reported")
	}
	if r := warRefused(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, st, content.SizeGargantuan, character.SmiteApplyProne)); r.Kind != character.WarlockProneNotAllowedForSize || r.Size != content.SizeGargantuan {
		t.Errorf("Gargantuan Prone = %+v", r)
	}
	declined := warChanged(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, st, content.SizeGargantuan, character.SmiteDeclineProne))
	if _, has := declined.SlotChange(); !has || declined.Events()[0].(character.EldritchSmiteUsed).Prone() {
		t.Error("declining Prone against a Gargantuan target still smites without Prone")
	}
	if r := warRefused(t, character.UseEldritchSmite(warID, warKnows("pact_of_the_blade"), character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, st, content.SizeMedium, 0)); r.Kind != character.WarlockInvocationNotKnown || r.Option != "eldritch_smite" {
		t.Errorf("not known = %+v", r)
	}
	for _, a := range []character.WarlockAttackKind{character.WarlockMissed, character.WarlockOtherHit} {
		if r := warRefused(t, character.UseEldritchSmite(warID, known, a, character.WarlockTurnFacts{}, st, content.SizeMedium, 0)); r.Kind != character.WarlockNotAHit {
			t.Errorf("attack %d = %+v", a, r)
		}
	}
	turn := character.WarlockTurnFacts{UsedThisTurn: []character.OncePerTurnFeature{character.OnceEldritchSmite}}
	if r := warRefused(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, turn, st, content.SizeMedium, 0)); r.Kind != character.WarlockAlreadyUsedThisTurn {
		t.Errorf("second use = %+v", r)
	}
	other := character.WarlockTurnFacts{UsedThisTurn: []character.OncePerTurnFeature{character.OnceHurlThroughHell}}
	warChanged(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, other, st, content.SizeMedium, 0))
	spent := warCasting(t, 9, 2)
	if r := warRefused(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, spent, content.SizeMedium, 0)); r.Kind != character.WarlockNoPactMagicSlot {
		t.Errorf("no slot = %+v", r)
	}
	if r := warRefused(t, character.UseEldritchSmite(warID, known, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, character.SpellcastingState{}, content.SizeMedium, 0)); r.Kind != character.WarlockNoPactMagicSlot {
		t.Errorf("no Pact Magic = %+v", r)
	}
}

// Requirement: doc/specs/class-warlock.md FR-123 to FR-125, table 4.8.1 (SRD 5.2.1, "Level 3: Dark One's Blessing", p. 76) - an enemy dropped by the Warlock gives the Charisma modifier plus the level in Temporary Hit Points (minimum 1); an enemy dropped by someone else applies at exactly 10 feet and not at 11; a creature that is not an enemy, a Warlock who is not a Fiend Patron and a level below 3 are refused.
// Path: per case (DB-01 to DB-07; boundary 10 and 11 feet).
func TestResolveDarkOnesBlessing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ level, cha, want int }{{3, 16, 6}, {3, 10, 3}, {3, 8, 2}, {3, 6, 1}, {3, 4, 1}, {3, 1, 1}, {20, 20, 25}} {
		got := warChanged(t, character.ResolveDarkOnesBlessing(warID, character.WarlockState{}, warFiend(c.level), warMod(c.cha), character.TargetEnemy, character.BlessingByWarlock, 90))
		if e := got.Events()[0].(character.DarkOnesBlessingGained); e.TemporaryHitPoints() != c.want || e.CharacterID() != warID {
			t.Errorf("level %d Charisma %d: %d, want %d", c.level, c.cha, e.TemporaryHitPoints(), c.want)
		}
	}
	by := func(d int) character.WarlockTransition {
		return character.ResolveDarkOnesBlessing(warID, character.WarlockState{}, warFiend(3), warMod(16), character.TargetEnemy, character.BlessingByAnother, d)
	}
	warChanged(t, by(10))
	warChanged(t, by(0))
	if r := warRefused(t, by(11)); r.Kind != character.WarlockTargetTooFar || r.Distance != 11 {
		t.Errorf("11 feet = %+v", r)
	}
	if r := warRefused(t, character.ResolveDarkOnesBlessing(warID, character.WarlockState{}, warFiend(3), warMod(16), character.TargetNotEnemy, character.BlessingByWarlock, 0)); r.Kind != character.WarlockNotAnEnemy {
		t.Errorf("not an enemy = %+v", r)
	}
	if r := warRefused(t, character.ResolveDarkOnesBlessing(warID, character.WarlockState{}, warProfile(3, ""), warMod(16), character.TargetEnemy, character.BlessingByWarlock, 0)); r.Kind != character.WarlockNotFiendPatron {
		t.Errorf("no subclass = %+v", r)
	}
	if r := warRefused(t, character.ResolveDarkOnesBlessing(warID, character.WarlockState{}, warFiend(2), warMod(16), character.TargetEnemy, character.BlessingByWarlock, 0)); r.Kind != character.WarlockFeatureNotGained || r.Feature != "dark_ones_blessing" {
		t.Errorf("level 2 = %+v", r)
	}
}

// Requirement: doc/specs/class-warlock.md FR-131 to FR-133 (SRD 5.2.1, "Level 10: Fiendish Resilience", p. 76) - on finishing a Short or Long Rest a Fiend Patron Warlock of level 10 chooses a damage type other than Force and keeps it until it chooses another; Force, no rest, level 9, no subclass are refused; an invalid damage type is an error.
// Path: per case (boundary 9 and 10; both rests; invalid input).
func TestChooseFiendishResilience(t *testing.T) {
	t.Parallel()
	for _, rest := range []character.RestKind{character.ShortRest, character.LongRest} {
		got := warChanged(t, must(character.ChooseFiendishResilience(warID, character.WarlockState{}, warFiend(10), rest, content.DamageFire)))
		if d, ok := got.State().FiendishResilience(); !ok || d != content.DamageFire {
			t.Errorf("resilience = %v, %v", d, ok)
		}
		if e := got.Events()[0].(character.FiendishResilienceChosen); e.DamageType() != content.DamageFire || e.CharacterID() != warID {
			t.Errorf("event = %+v", e)
		}
		next := warChanged(t, must(character.ChooseFiendishResilience(warID, got.State(), warFiend(10), rest, content.DamageCold)))
		if d, _ := next.State().FiendishResilience(); d != content.DamageCold {
			t.Errorf("a different choice must replace the first: %v", d)
		}
	}
	refuse := func(p character.WarlockProfile, rest character.RestKind, d content.DamageType) character.WarlockRejection {
		return warRefused(t, must(character.ChooseFiendishResilience(warID, character.WarlockState{}, p, rest, d)))
	}
	if r := refuse(warFiend(10), character.LongRest, content.DamageForce); r.Kind != character.WarlockForceNotAllowed {
		t.Errorf("Force = %+v", r)
	}
	if r := refuse(warFiend(10), 0, content.DamageFire); r.Kind != character.WarlockNotAfterRest {
		t.Errorf("no rest = %+v", r)
	}
	if r := refuse(warFiend(9), character.LongRest, content.DamageFire); r.Kind != character.WarlockFeatureNotGained || r.Feature != "fiendish_resilience" {
		t.Errorf("level 9 = %+v", r)
	}
	if r := refuse(warProfile(10, ""), character.LongRest, content.DamageFire); r.Kind != character.WarlockNotFiendPatron {
		t.Errorf("no subclass = %+v", r)
	}
	if _, err := character.ChooseFiendishResilience(warID, character.WarlockState{}, warFiend(10), character.LongRest, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid damage type: %v", err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-127 to FR-130, FR-145, table 4.8.3 (SRD 5.2.1, "Level 6: Dark One's Own Luck", p. 76) - an ability check or a saving throw gets the d10 as a Modifier (1 and 10 at the extremes) and spends a use; the uses are the Charisma modifier (minimum 1); an attack roll, a second use on the roll, no use left, no subclass and level 5 are refused; a roll outside 1 to 10 or an invalid kind is an error; a Long Rest restores the uses.
// Path: per case (DL-01 to DL-05; roll kinds; extremes).
func TestAddDarkOnesOwnLuck(t *testing.T) {
	t.Parallel()
	use := func(s character.WarlockState, p character.WarlockProfile, cha int, kind d20.Kind, on character.LuckOnRoll, rolled int) (character.WarlockTransition, error) {
		return character.AddDarkOnesOwnLuck(warID, s, p, warMod(cha), kind, on, rolled)
	}
	for _, kind := range []d20.Kind{d20.AbilityCheck, d20.SavingThrow} {
		for _, rolled := range []int{1, 10} {
			got := warChanged(t, must(use(character.WarlockState{}, warFiend(6), 16, kind, character.LuckNotUsedOnRoll, rolled)))
			b, has := got.Bonus()
			if !has || b != modifier.New(rolled) || got.State().DarkOnesOwnLuckSpent() != 1 {
				t.Errorf("kind %v roll %d: bonus %v, spent %d", kind, rolled, b, got.State().DarkOnesOwnLuckSpent())
			}
			if e := got.Events()[0].(character.DarkOnesOwnLuckUsed); e.Rolled() != rolled || e.Kind() != kind || e.CharacterID() != warID {
				t.Errorf("event = %+v", e)
			}
		}
	}
	for _, c := range []struct{ cha, uses int }{{8, 1}, {10, 1}, {12, 1}, {14, 2}, {20, 5}} {
		s := character.WarlockState{}
		for i := 0; i < c.uses; i++ {
			s = warChanged(t, must(use(s, warFiend(6), c.cha, d20.SavingThrow, character.LuckNotUsedOnRoll, 5))).State()
		}
		r := warRefused(t, must(use(s, warFiend(6), c.cha, d20.SavingThrow, character.LuckNotUsedOnRoll, 5)))
		if r.Kind != character.WarlockNoUsesLeft || r.Required != c.uses || r.Available != 0 {
			t.Errorf("Charisma %d after %d uses = %+v", c.cha, c.uses, r)
		}
		rested := warChanged(t, must(character.ApplyWarlockRest(warID, s, character.LongRest))).State()
		warChanged(t, must(use(rested, warFiend(6), c.cha, d20.SavingThrow, character.LuckNotUsedOnRoll, 5)))
	}
	refused := func(p character.WarlockProfile, kind d20.Kind, on character.LuckOnRoll) character.WarlockRejection {
		return warRefused(t, must(use(character.WarlockState{}, p, 16, kind, on, 5)))
	}
	if r := refused(warFiend(6), d20.AttackRoll, 0); r.Kind != character.WarlockRollKindNotAllowed {
		t.Errorf("attack roll = %+v", r)
	}
	if r := refused(warFiend(6), d20.SavingThrow, character.LuckUsedOnRoll); r.Kind != character.WarlockAlreadyUsedOnThisRoll {
		t.Errorf("second use on the roll = %+v", r)
	}
	if r := refused(warProfile(6, ""), d20.SavingThrow, 0); r.Kind != character.WarlockNotFiendPatron {
		t.Errorf("no subclass = %+v", r)
	}
	if r := refused(warFiend(5), d20.SavingThrow, 0); r.Kind != character.WarlockFeatureNotGained || r.Feature != "dark_ones_own_luck" {
		t.Errorf("level 5 = %+v", r)
	}
	for _, rolled := range []int{0, 11, -3} {
		if _, err := use(character.WarlockState{}, warFiend(6), 16, d20.SavingThrow, 0, rolled); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("roll %d: %v", rolled, err)
		}
	}
	if _, err := use(character.WarlockState{}, warFiend(6), 16, 0, 0, 5); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid kind: %v", err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-134 to FR-138, FR-163 (SRD 5.2.1, "Level 14: Hurl Through Hell", p. 76) - a hit once per turn at level 14 spends the use; a miss, a second use in the turn, a spent use, level 13 and no subclass are refused; restoring needs a spent use and a remaining Pact Magic slot, returns WarlockSlotChangeExpendPact at the Slot Level and changes nothing on a refusal; the effect is 8d10 Psychic unless the target is a Fiend, with Incapacitated.
// Path: per case (boundary 13 and 14; state errors).
func TestHurlThroughHell(t *testing.T) {
	t.Parallel()
	hit := func(s character.WarlockState, p character.WarlockProfile, a character.WarlockAttackKind, turn character.WarlockTurnFacts) character.WarlockTransition {
		return character.UseHurlThroughHell(warID, s, p, a, turn)
	}
	for _, a := range []character.WarlockAttackKind{character.WarlockPactWeaponHit, character.WarlockOtherHit} {
		got := warChanged(t, hit(character.WarlockState{}, warFiend(14), a, character.WarlockTurnFacts{}))
		if !got.State().HurlThroughHellSpent() || got.Events()[0].CharacterID() != warID {
			t.Errorf("hit %d: %+v", a, got)
		}
		if _, ok := got.Events()[0].(character.HurlThroughHellUsed); !ok {
			t.Errorf("event = %T", got.Events()[0])
		}
	}
	spent := warChanged(t, hit(character.WarlockState{}, warFiend(14), character.WarlockOtherHit, character.WarlockTurnFacts{})).State()
	if r := warRefused(t, hit(spent, warFiend(14), character.WarlockOtherHit, character.WarlockTurnFacts{})); r.Kind != character.WarlockHurlThroughHellAlreadySpent {
		t.Errorf("spent = %+v", r)
	}
	if r := warRefused(t, hit(character.WarlockState{}, warFiend(14), character.WarlockMissed, character.WarlockTurnFacts{})); r.Kind != character.WarlockNotAHit {
		t.Errorf("miss = %+v", r)
	}
	turn := character.WarlockTurnFacts{UsedThisTurn: []character.OncePerTurnFeature{character.OnceHurlThroughHell}}
	if r := warRefused(t, hit(character.WarlockState{}, warFiend(14), character.WarlockOtherHit, turn)); r.Kind != character.WarlockAlreadyUsedThisTurn {
		t.Errorf("second use in the turn = %+v", r)
	}
	smiteOnly := character.WarlockTurnFacts{UsedThisTurn: []character.OncePerTurnFeature{character.OnceEldritchSmite}}
	warChanged(t, hit(character.WarlockState{}, warFiend(14), character.WarlockOtherHit, smiteOnly))
	if r := warRefused(t, hit(character.WarlockState{}, warFiend(13), character.WarlockOtherHit, character.WarlockTurnFacts{})); r.Kind != character.WarlockFeatureNotGained || r.Feature != "hurl_through_hell" {
		t.Errorf("level 13 = %+v", r)
	}
	if r := warRefused(t, hit(character.WarlockState{}, warProfile(14, ""), character.WarlockOtherHit, character.WarlockTurnFacts{})); r.Kind != character.WarlockNotFiendPatron {
		t.Errorf("no subclass = %+v", r)
	}

	st := warCasting(t, 14, 0)
	got := warChanged(t, character.RestoreHurlThroughHell(warID, spent, warFiend(14), st))
	change, has := got.SlotChange()
	if !has || change != character.WarlockSlotChange(character.WarlockSlotChangeExpendPact{Level: 5}) || got.State().HurlThroughHellSpent() {
		t.Errorf("restore = %v, %v, spent %v", change, has, got.State().HurlThroughHellSpent())
	}
	if e := got.Events()[0].(character.HurlThroughHellRestored); e.SlotLevel() != 5 || e.CharacterID() != warID {
		t.Errorf("event = %+v", e)
	}
	refuseRestore := func(s character.WarlockState, p character.WarlockProfile, st character.SpellcastingState) character.WarlockRejection {
		return warRefused(t, character.RestoreHurlThroughHell(warID, s, p, st))
	}
	if r := refuseRestore(spent, warFiend(14), warCasting(t, 14, 3)); r.Kind != character.WarlockNoPactMagicSlot {
		t.Errorf("no slot = %+v", r)
	}
	warChanged(t, character.RestoreHurlThroughHell(warID, spent, warFiend(14), warCasting(t, 14, 2)))
	if r := refuseRestore(character.WarlockState{}, warFiend(14), st); r.Kind != character.WarlockHurlThroughHellUseNotSpent {
		t.Errorf("not spent = %+v", r)
	}
	if r := refuseRestore(spent, warProfile(14, ""), st); r.Kind != character.WarlockNotFiendPatron {
		t.Errorf("no subclass = %+v", r)
	}
	if r := refuseRestore(spent, warFiend(13), st); r.Kind != character.WarlockFeatureNotGained || r.Feature != "hurl_through_hell" {
		t.Errorf("level 13 = %+v", r)
	}

	fiend, other := character.HurlThroughHellFor(content.CreatureTypeFiend), character.HurlThroughHellFor(content.CreatureTypeHumanoid)
	if fiend.Deals || !other.Deals || other.Damage != dice.MustExpression(8, dice.D10(), 0) || other.DamageType != content.DamagePsychic || other.Condition != content.ConditionIncapacitated || fiend.Condition != content.ConditionIncapacitated {
		t.Errorf("effects = %+v, %+v", fiend, other)
	}
}

// Requirement: doc/specs/class-warlock.md FR-139, FR-140 - a Long Rest restores every spent Warlock use and emits one WarlockFeaturesRestored listing only what was spent, in order, and none when nothing was; a Short Rest restores none and emits no event; a kind that is neither is an error.
// Path: per case (Long Rest, Short Rest, empty set, invalid input).
func TestApplyWarlockRest(t *testing.T) {
	t.Parallel()
	p := warFiend(17)
	spent := warState(p, character.WarlockStateParts{Arcanum: [4]content.SpellID{"circle_of_death"}, ArcanumCast: [4]bool{true}, MagicalCunningUsed: true, ContactPatronUsed: true,
		GiftOfTheDepthsCast: true, GiftOfTheProtectorsSpent: true, DarkOnesOwnLuckSpent: 2, HurlThroughHellSpent: true, FiendishResilience: content.DamageFire,
		Invocations: []content.FeatureOptionID{"eldritch_mind"}})
	long := warChanged(t, must(character.ApplyWarlockRest(warID, spent, character.LongRest)))
	want := []character.WarlockRestorableFeature{character.RestoresMagicalCunning, character.RestoresContactPatron, character.RestoresMysticArcanum, character.RestoresGiftOfTheDepths,
		character.RestoresGiftOfTheProtectors, character.RestoresDarkOnesOwnLuck, character.RestoresHurlThroughHell}
	ev := long.Events()
	if len(ev) != 1 || !slices.Equal(ev[0].(character.WarlockFeaturesRestored).Features(), want) || ev[0].CharacterID() != warID {
		t.Fatalf("events = %+v", ev)
	}
	wantState := warState(p, character.WarlockStateParts{Arcanum: [4]content.SpellID{"circle_of_death"}, FiendishResilience: content.DamageFire, Invocations: []content.FeatureOptionID{"eldritch_mind"}})
	if long.State() != wantState {
		t.Errorf("state after a Long Rest = %+v, want %+v", long.State().Parts(), wantState.Parts())
	}
	short := warChanged(t, must(character.ApplyWarlockRest(warID, spent, character.ShortRest)))
	if short.State() != spent || len(short.Events()) != 0 {
		t.Error("a Short Rest restores nothing and emits no event")
	}
	nothing := warChanged(t, must(character.ApplyWarlockRest(warID, wantState, character.LongRest)))
	if nothing.State() != wantState || len(nothing.Events()) != 0 {
		t.Error("a Long Rest with nothing spent emits no event")
	}
	for _, only := range []struct {
		parts character.WarlockStateParts
		f     character.WarlockRestorableFeature
	}{{character.WarlockStateParts{MagicalCunningUsed: true}, character.RestoresMagicalCunning}, {character.WarlockStateParts{ContactPatronUsed: true}, character.RestoresContactPatron},
		{character.WarlockStateParts{GiftOfTheDepthsCast: true}, character.RestoresGiftOfTheDepths}, {character.WarlockStateParts{GiftOfTheProtectorsSpent: true}, character.RestoresGiftOfTheProtectors},
		{character.WarlockStateParts{DarkOnesOwnLuckSpent: 1}, character.RestoresDarkOnesOwnLuck}, {character.WarlockStateParts{HurlThroughHellSpent: true}, character.RestoresHurlThroughHell}} {
		got := warChanged(t, must(character.ApplyWarlockRest(warID, warState(p, only.parts), character.LongRest))).Events()
		if len(got) != 1 || !slices.Equal(got[0].(character.WarlockFeaturesRestored).Features(), []character.WarlockRestorableFeature{only.f}) {
			t.Errorf("only %v spent: events = %+v", only.f, got)
		}
	}
	if _, err := character.ApplyWarlockRest(warID, spent, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid kind: %v", err)
	}
	if character.RestoresMagicalCunning.String() != "magical_cunning" || character.RestoresHurlThroughHell.String() != "hurl_through_hell" || character.WarlockRestorableFeature(0).String() != "unknown" {
		t.Error("WarlockRestorableFeature strings")
	}
}

// Requirement: doc/specs/class-warlock.md FR-033, FR-034, FR-126, table 4.8.2 (SRD 5.2.1, "Fiend Spells", p. 76; "Level 9: Contact Patron", p. 72) - the always-prepared spells are the facts of the class and the chosen subclass at or below the level, cumulatively, read from the definition: none at level 2 without a subclass or before level 3, 4 at level 3 and 4, 6 at 5, 8 at 7, 10 at 9 plus Contact Other Plane, Contact Other Plane alone for a Warlock without a subclass at level 9, and none from a subclass the definition does not list.
// Path: per case (FS-01 to FS-10).
func TestWarlockAlwaysPrepared(t *testing.T) {
	t.Parallel()
	def := warDef(t)
	cases := []struct {
		name  string
		p     character.WarlockProfile
		count int
		first []content.SpellID
	}{
		{"FS-01", warProfile(2, ""), 0, nil}, {"FS-02", warFiend(3), 4, []content.SpellID{"burning_hands", "command", "scorching_ray", "suggestion"}},
		{"FS-03", warFiend(4), 4, nil}, {"FS-04", warFiend(5), 6, nil}, {"FS-05", warFiend(7), 8, nil},
		{"FS-06", warFiend(9), 11, nil}, {"FS-07", warFiend(20), 11, nil}, {"FS-08", warProfile(9, ""), 1, []content.SpellID{"contact_other_plane"}},
		{"FS-09", warProfile(8, ""), 0, nil}, {"FS-10", warProfile(5, "not_a_patron"), 0, nil},
	}
	for _, c := range cases {
		got := character.WarlockAlwaysPrepared(c.p, def)
		if got == nil || len(got) != c.count || (c.first != nil && !slices.Equal(got, c.first)) {
			t.Errorf("%s: %v, want %d spells", c.name, got, c.count)
		}
	}
	nine := character.WarlockAlwaysPrepared(warFiend(9), def)
	for _, id := range []content.SpellID{"fireball", "geas", "insect_plague", "contact_other_plane", "wall_of_fire"} {
		if !slices.Contains(nine, id) {
			t.Errorf("level 9 lacks %s: %v", id, nine)
		}
	}
	if slices.Contains(character.WarlockAlwaysPrepared(warFiend(8), def), "geas") {
		t.Error("Geas is granted at level 9")
	}
}
