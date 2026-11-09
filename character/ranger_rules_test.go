package character_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-ranger.md FR-065, FR-068 (SRD 5.2.1, "Level 5: Extra Attack", p. 59) - a Ranger has one attack per Attack action below level 5 and two from it (a total, not a bonus); the zero facts have one.
// Path: per case (unhappy: boundary).
func TestRangerAttacksPerAttackAction(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ level, want int }{{1, 1}, {4, 1}, {5, 2}, {20, 2}} {
		if got := rgrFacts(t, c.level, character.InitialRangerState(), "").AttacksPerAttackAction(); got != c.want {
			t.Errorf("level %d = %d attacks, want %d", c.level, got, c.want)
		}
	}
	if (character.RangerFacts{}).AttacksPerAttackAction() != 1 {
		t.Error("the zero facts have more than one attack")
	}
}

// Requirement: doc/specs/class-ranger.md FR-066, FR-067, FR-069, FR-070, T3 (SRD 5.2.1, "Level 6: Roving", p. 59; Rules Glossary, "Grappled", p. 182) - Roving adds 10 feet from level 6 while no Heavy armor is worn and the Speed
// can increase, with a Climb and a Swim Speed equal to the Speed from level 6; rows RV-01 to RV-10 of T3.
// Path: per case (T3): happy / unhappy (boundary, rule interaction, invalid input).
func TestRangerSpeed(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label       string
		level, base int
		can, heavy  bool
		walk        int
		climbSwim   bool // has a Climb and a Swim Speed equal to the walk
	}{
		{"RV-01", 5, 30, true, false, 30, false}, {"RV-02", 6, 30, true, false, 40, true}, {"RV-03", 6, 30, true, true, 30, true}, {"RV-04", 20, 35, true, false, 45, true},
		{"RV-05", 6, 25, true, false, 35, true}, {"RV-06", 6, 0, true, false, 10, true}, {"RV-07", 6, 0, true, true, 0, true}, {"RV-09", 6, 0, false, false, 0, true},
		{"RV-10", 6, 30, false, false, 30, true},
	} {
		got := rgrFacts(t, c.level, character.InitialRangerState(), "").Speed(character.MustSpeedInput(c.base, c.can), c.heavy)
		climb, hasClimb := got.Climb()
		swim, hasSwim := got.Swim()
		if got.Walk() != c.walk || hasClimb != c.climbSwim || hasSwim != c.climbSwim || (c.climbSwim && (climb != c.walk || swim != c.walk)) {
			t.Errorf("%s: %+v, want walk %d with climb and swim %v", c.label, got, c.walk, c.climbSwim)
		}
	}
	_, err := character.NewSpeedInput(-5, true)
	wantInvalid(t, err, "RV-08 negative base Speed")
	defer func() {
		if recover() == nil {
			t.Error("RV-08: MustSpeedInput(-5) did not panic")
		}
	}()
	character.MustSpeedInput(-5, true)
}

// Requirement: doc/specs/class-ranger.md FR-082 (SRD 5.2.1, "Level 18: Feral Senses", p. 59) - Blindsight 30 feet from level 18, none below; the result is a new slice.
// Path: per case (unhappy: boundary one below).
func TestRangerSenses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		want  []character.RangerSenseGrant
	}{{17, nil}, {18, []character.RangerSenseGrant{{Sense: content.SenseBlindsight, RangeFeet: 30}}}, {20, []character.RangerSenseGrant{{Sense: content.SenseBlindsight, RangeFeet: 30}}}} {
		got := rgrFacts(t, c.level, character.InitialRangerState(), "").Senses()
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) {
			t.Errorf("level %d senses = %v, want %v", c.level, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-083, T5 (SRD 5.2.1, "Level 20: Foe Slayer", p. 59) - Hunter’s Mark's die is a d10 at level 20 and the spell's own die below it; rows FS-01 to FS-04 of T5 (level 21 cannot be built).
// Path: per case (T5): happy / unhappy (boundary, invalid input).
func TestRangerHuntersMarkDie(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label string
		level int
		want  dice.Die
	}{{"FS-01", 20, dice.D10()}, {"FS-02", 19, dice.D6()}, {"FS-03", 1, dice.D6()}} {
		if got := rgrFacts(t, c.level, character.InitialRangerState(), "").HuntersMarkDie(dice.D6()); got != c.want {
			t.Errorf("%s: level %d die = %v, want %v", c.label, c.level, got, c.want)
		}
	}
	_, err := character.NewRangerLevel(21)
	wantInvalid(t, err, "FS-04 level 21")
}

// Requirement: doc/specs/class-ranger.md FR-078 (SRD 5.2.1, "Level 13: Relentless Hunter", p. 59) - from level 13 damage cannot break Concentration on Hunter’s Mark, and any other spell and any level below 13 can still be broken.
// Path: per case (unhappy: boundary, rule interaction).
func TestRangerDamageCanBreakConcentration(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		spell content.SpellID
		want  bool
	}{{12, "hunters_mark", true}, {13, "hunters_mark", false}, {20, "hunters_mark", false}, {13, "entangle", true}, {1, "entangle", true}} {
		if got := rgrFacts(t, c.level, character.InitialRangerState(), "").DamageCanBreakConcentration(c.spell); got != c.want {
			t.Errorf("level %d %s = %v, want %v", c.level, c.spell, got, c.want)
		}
	}
	if !(character.RangerFacts{}).DamageCanBreakConcentration("hunters_mark") {
		t.Error("the zero facts protect Concentration")
	}
}

// Requirement: doc/specs/class-ranger.md FR-081 (SRD 5.2.1, "Level 17: Precise Hunter", p. 59) - from level 17 an attack roll against the marked creature carries Advantage, and none against an unmarked one or below level 17.
// Path: per case (unhappy: boundary, rule interaction).
func TestRangerPreciseHunter(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level  int
		target character.MarkStatus
		want   []modifier.RollCircumstance
	}{
		{16, character.MarkedByHuntersMark, nil}, {17, character.MarkedByHuntersMark, []modifier.RollCircumstance{modifier.Advantage{}}},
		{17, character.UnmarkedByHuntersMark, nil}, {20, character.MarkedByHuntersMark, []modifier.RollCircumstance{modifier.Advantage{}}}, {17, 0, nil},
	} {
		got := rgrFacts(t, c.level, character.InitialRangerState(), "").PreciseHunter(c.target)
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) {
			t.Errorf("level %d target %d = %v, want %v", c.level, c.target, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-084, FR-085 (SRD 5.2.1, "Level 3: Hunter’s Lore", p. 61) - a marked creature's Immunities, Resistances and Vulnerabilities are revealed unchanged with their qualifiers, and NoneApply when
// all three are empty; nothing is revealed for a creature not marked, a character without the Hunter or below level 3, and the lists revealed are copies.
// Path: per case: happy / unhappy (state error, boundary, rule interaction).
func TestRangerHuntersLore(t *testing.T) {
	t.Parallel()
	fire := must(content.NewDamageTypeDefense(content.DamageFire, "from nonmagical attacks"))
	cold := must(content.NewDamageTypeDefense(content.DamageCold, ""))
	poisoned := must(content.NewConditionDefense(content.ConditionPoisoned, ""))
	f := rgrFacts(t, 3, character.InitialRangerState(), hunter)
	reveals, ok := f.HuntersLore(character.MarkedByHuntersMark, []content.DefenseEntry{poisoned}, []content.DefenseEntry{fire}, []content.DefenseEntry{cold}).(character.HuntersLoreReveals)
	if !ok || !reflect.DeepEqual(reveals.Immunities(), []content.DefenseEntry{poisoned}) || !reflect.DeepEqual(reveals.Resistances(), []content.DefenseEntry{fire}) ||
		!reflect.DeepEqual(reveals.Vulnerabilities(), []content.DefenseEntry{cold}) {
		t.Fatalf("reveals = %#v", reveals)
	}
	reveals.Resistances()[0] = cold
	if reveals.Resistances()[0] != content.DefenseEntry(fire) {
		t.Error("Resistances returned shared storage")
	}
	input := []content.DefenseEntry{fire}
	shared := f.HuntersLore(character.MarkedByHuntersMark, nil, input, nil).(character.HuntersLoreReveals)
	input[0] = cold
	if shared.Resistances()[0] != content.DefenseEntry(fire) {
		t.Error("the revealed list shares the caller's storage")
	}
	for _, c := range []struct {
		label  string
		facts  character.RangerFacts
		target character.MarkStatus
		want   character.HuntersLoreResult
	}{
		{"nothing applies", f, character.MarkedByHuntersMark, character.HuntersLoreNoneApply{}},
		{"not marked", f, character.UnmarkedByHuntersMark, character.HuntersLoreNotMarked{}},
		{"no Hunter", rgrFacts(t, 3, character.InitialRangerState(), ""), character.MarkedByHuntersMark, character.HuntersLoreNotHunter{}},
		{"below level 3", rgrFacts(t, 2, character.InitialRangerState(), hunter), character.MarkedByHuntersMark, character.HuntersLoreFeatureNotAvailable{}},
	} {
		if got := c.facts.HuntersLore(c.target, nil, nil, nil); got != c.want {
			t.Errorf("%s = %#v, want %#v", c.label, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-088, FR-089, T8 (SRD 5.2.1, "Level 3: Hunter’s Prey", p. 61) - Colossus Slayer adds 1d8 of unstated damage type to a weapon hit on a creature missing Hit Points once per turn; rows CS-01 to CS-08 of T8.
// Path: per case (T8): happy / unhappy (boundary, state error, rule interaction, missing choice).
func TestRangerColossusSlayer(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label       string
		level       int
		prey        string
		weapon      bool
		hp, max     int
		dealt, want bool
	}{
		{"CS-01", 3, "colossus_slayer", true, 19, 20, false, true}, {"CS-02", 3, "colossus_slayer", true, 20, 20, false, false}, {"CS-03", 3, "colossus_slayer", true, 1, 20, false, true},
		{"CS-04", 3, "colossus_slayer", true, 10, 20, true, false}, {"CS-05", 3, "colossus_slayer", false, 10, 20, false, false}, {"CS-06", 3, "horde_breaker", true, 10, 20, false, false},
		{"CS-07", 3, "", true, 10, 20, false, false}, {"CS-08", 2, "colossus_slayer", true, 10, 20, false, false},
	} {
		rider, got := rgrHunterFacts(t, c.level, c.prey, "").ColossusSlayer(character.ColossusSlayerFacts{WeaponHit: c.weapon, TargetHitPoints: c.hp, TargetMaximum: c.max, DealtThisTurn: c.dealt})
		if got != c.want || (got && (rider.Count != 1 || rider.Die != dice.D8() || rider.Damage != (character.UnstatedDamageType{}))) {
			t.Errorf("%s: rider %+v, %v; want %v", c.label, rider, got, c.want)
		}
	}
	if _, got := rgrFacts(t, 5, rgrState(t, character.RangerStateParts{HuntersPrey: "colossus_slayer"}), "").ColossusSlayer(character.ColossusSlayerFacts{WeaponHit: true, TargetHitPoints: 1, TargetMaximum: 2}); got {
		t.Error("Colossus Slayer applied without the Hunter")
	}
}

// Requirement: doc/specs/class-ranger.md FR-090, T9 (SRD 5.2.1, "Level 3: Hunter’s Prey", p. 61) - Horde Breaker allows one further attack with the same weapon against a different creature within 5 feet, in range and not yet attacked, once per turn;
// each failing condition yields Ineligible with its reason; rows HB-01 to HB-10 of T9.
// Path: per case (T9): happy / unhappy (boundary, state error, rule interaction, missing choice).
func TestRangerHordeBreaker(t *testing.T) {
	t.Parallel()
	ok := func() character.HordeBreakerFacts {
		return character.HordeBreakerFacts{WeaponAttack: true, DifferentCreature: true, DistanceFeet: 5, InWeaponRange: true}
	}
	with := func(edit func(*character.HordeBreakerFacts)) character.HordeBreakerFacts {
		h := ok()
		edit(&h)
		return h
	}
	var eligible character.RangerRiderEligibility = character.RangerRiderEligible{}
	ineligible := func(r character.RangerRiderReason) character.RangerRiderEligibility {
		return character.RangerRiderIneligible{Reason: r}
	}
	for _, c := range []struct {
		label string
		level int
		prey  string
		facts character.HordeBreakerFacts
		want  character.RangerRiderEligibility
	}{
		{"HB-01", 3, "horde_breaker", ok(), eligible},
		{"HB-02", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.DistanceFeet = 6 }), ineligible(character.RangerRiderTooFar)},
		{"HB-03", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.DistanceFeet = 0 }), eligible},
		{"HB-04", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.DifferentCreature, h.DistanceFeet = false, 0 }), ineligible(character.RangerRiderSameCreature)},
		{"HB-05", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.InWeaponRange = false }), ineligible(character.RangerRiderOutOfRange)},
		{"HB-06", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.AlreadyAttacked = true }), ineligible(character.RangerRiderAlreadyAttacked)},
		{"HB-07", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.UsedThisTurn = true }), ineligible(character.RangerRiderUsedThisTurn)},
		{"HB-08", 3, "horde_breaker", with(func(h *character.HordeBreakerFacts) { h.WeaponAttack = false }), ineligible(character.RangerRiderNotWeaponAttack)},
		{"HB-09", 3, "colossus_slayer", ok(), ineligible(character.RangerRiderOptionNotHeld)},
		{"HB-10", 2, "horde_breaker", ok(), ineligible(character.RangerRiderFeatureNotAvailable)},
	} {
		if got := rgrHunterFacts(t, c.level, c.prey, "").HordeBreaker(c.facts); got != c.want {
			t.Errorf("%s = %#v, want %#v", c.label, got, c.want)
		}
	}
	if got := rgrFacts(t, 5, rgrState(t, character.RangerStateParts{HuntersPrey: "horde_breaker"}), "").HordeBreaker(ok()); got != ineligible(character.RangerRiderFeatureNotAvailable) {
		t.Errorf("without the Hunter = %#v", got)
	}
}

// Requirement: doc/specs/class-ranger.md FR-094, T9 (SRD 5.2.1, "Level 11: Superior Hunter’s Prey", p. 61) - once per turn the mark's extra damage may go to a different visible creature within 30 feet of the first marked one, with
// the die of Hunter’s Mark (d10 from level 20); rows SP-01 to SP-09 of T9.
// Path: per case (T9): happy / unhappy (boundary, invalid input, state error, rule interaction).
func TestRangerSuperiorHuntersPrey(t *testing.T) {
	t.Parallel()
	ok := func(edit func(*character.SuperiorHuntersPreyFacts)) character.SuperiorHuntersPreyFacts {
		h := character.SuperiorHuntersPreyFacts{FirstTarget: character.MarkedByHuntersMark, SpellDie: dice.D6(), DifferentCreature: true, DistanceFeet: 10, Visible: true}
		if edit != nil {
			edit(&h)
		}
		return h
	}
	ineligible := func(r character.RangerRiderReason) character.RangerRiderEligibility {
		return character.RangerRiderIneligible{Reason: r}
	}
	for _, c := range []struct {
		label string
		level int
		facts character.SuperiorHuntersPreyFacts
		want  character.RangerRiderEligibility
	}{
		{"SP-01", 11, ok(func(h *character.SuperiorHuntersPreyFacts) { h.DistanceFeet = 30 }), character.RangerRiderEligible{Die: dice.D6()}},
		{"SP-02", 11, ok(func(h *character.SuperiorHuntersPreyFacts) { h.DistanceFeet = 31 }), ineligible(character.RangerRiderTooFar)},
		{"SP-03", 11, ok(func(h *character.SuperiorHuntersPreyFacts) { h.Visible = false }), ineligible(character.RangerRiderNotVisible)},
		{"SP-04", 11, ok(func(h *character.SuperiorHuntersPreyFacts) { h.DifferentCreature, h.DistanceFeet = false, 0 }), ineligible(character.RangerRiderSameCreature)},
		{"SP-05", 11, ok(func(h *character.SuperiorHuntersPreyFacts) { h.UsedThisTurn = true }), ineligible(character.RangerRiderUsedThisTurn)},
		{"SP-06", 11, ok(func(h *character.SuperiorHuntersPreyFacts) { h.FirstTarget = character.UnmarkedByHuntersMark }), ineligible(character.RangerRiderNotMarked)},
		{"SP-07", 20, ok(nil), character.RangerRiderEligible{Die: dice.D10()}},
		{"SP-08", 10, ok(nil), ineligible(character.RangerRiderFeatureNotAvailable)},
		{"SP-09", 11, ok(nil), character.RangerRiderEligible{Die: dice.D6()}},
	} {
		if got := rgrFacts(t, c.level, character.InitialRangerState(), hunter).SuperiorHuntersPrey(c.facts); got != c.want {
			t.Errorf("%s = %#v, want %#v", c.label, got, c.want)
		}
	}
	if got := rgrFacts(t, 11, character.InitialRangerState(), "").SuperiorHuntersPrey(ok(nil)); got != ineligible(character.RangerRiderFeatureNotAvailable) {
		t.Errorf("without the Hunter = %#v", got)
	}
}

// Requirement: doc/specs/class-ranger.md FR-092, FR-093 (SRD 5.2.1, "Level 7: Defensive Tactics", p. 61) - Escape the Horde gives an Opportunity Attack against the Ranger Disadvantage and no other attack roll; Multiattack Defense gives an attacker that
// hit the Ranger this turn Disadvantage on its other attack rolls, not the roll that hit and no other creature, the record is empty after EndOfTurn, and neither works unheld, below level 7 or without the Hunter.
// Path: per case: happy / unhappy (rule interaction, boundary, missing choice).
func TestRangerDefensiveTactics(t *testing.T) {
	t.Parallel()
	disadvantage := []modifier.RollCircumstance{modifier.Disadvantage{}}
	escape, multi := rgrHunterFacts(t, 7, "", "escape_the_horde"), rgrHunterFacts(t, 7, "", "multiattack_defense")
	if got := escape.EscapeTheHorde(true); !reflect.DeepEqual(got, disadvantage) {
		t.Errorf("Opportunity Attack = %v", got)
	}
	if got := escape.EscapeTheHorde(false); len(got) != 0 {
		t.Errorf("another attack = %v", got)
	}
	for _, f := range []character.RangerFacts{multi, rgrHunterFacts(t, 6, "", "escape_the_horde"), rgrHunterFacts(t, 7, "", ""), rgrFacts(t, 7, rgrState(t, character.RangerStateParts{DefensiveTactics: "escape_the_horde"}), "")} {
		if got := f.EscapeTheHorde(true); len(got) != 0 {
			t.Errorf("%+v escapes the horde: %v", f, got)
		}
	}
	var record character.MultiattackDefenseRecord
	if got := multi.MultiattackDefense(record, "orc"); len(got) != 0 {
		t.Errorf("the roll that hit got %v", got)
	}
	after := multi.RecordHit(record, "orc")
	if got := multi.MultiattackDefense(after, "orc"); !reflect.DeepEqual(got, disadvantage) {
		t.Errorf("a later roll by the attacker got %v", got)
	}
	if got := multi.MultiattackDefense(after, "goblin"); len(got) != 0 {
		t.Errorf("another creature got %v", got)
	}
	if record.Holds("orc") || !after.Holds("orc") || !after.HitBy("orc").Holds("orc") || after.EndOfTurn().Holds("orc") || len(multi.MultiattackDefense(after.EndOfTurn(), "orc")) != 0 {
		t.Error("the record is wrong")
	}
	twice := after.HitBy("orc").HitBy("goblin")
	if !twice.Holds("orc") || !twice.Holds("goblin") {
		t.Error("the record lost an attacker")
	}
	for _, f := range []character.RangerFacts{escape, rgrHunterFacts(t, 6, "", "multiattack_defense"), rgrFacts(t, 7, rgrState(t, character.RangerStateParts{DefensiveTactics: "multiattack_defense"}), "")} {
		if got := f.RecordHit(record, "orc"); got.Holds("orc") {
			t.Errorf("%+v recorded a hit without Multiattack Defense", f)
		}
		if got := f.MultiattackDefense(after, "orc"); len(got) != 0 {
			t.Errorf("%+v imposed Disadvantage without Multiattack Defense", f)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-033, FR-034, FR-035 (SRD 5.2.1, "Level 1: Favored Enemy", p. 58) - a free cast of Hunter’s Mark is accepted while a use remains, the last one included, spends one use and emits FavoredEnemyCastSpent
// with the uses left; with none left it is rejected with no_uses_remaining and the state stands; the input state is not changed and a level that raises the maximum leaves the spent count, so remaining rises.
// Path: per case: happy / unhappy (boundary, state error, immutability).
func TestUseFavoredEnemy(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	start := character.InitialRangerState()
	one := character.UseFavoredEnemy(id, rgrFacts(t, 1, start, "")).(character.FavoredEnemyUseAccepted)
	two := character.UseFavoredEnemy(id, rgrFacts(t, 1, one.State(), "")).(character.FavoredEnemyUseAccepted)
	last, ok := one.Events()[0].(character.FavoredEnemyCastSpent)
	if !ok || last.Remaining() != 1 || last.CharacterID() != id || one.State().FavoredEnemySpent() != 1 || two.State().FavoredEnemySpent() != 2 || start.FavoredEnemySpent() != 0 {
		t.Fatalf("first use = %+v events %v", one.State(), one.Events())
	}
	if e := two.Events()[0].(character.FavoredEnemyCastSpent); e.Remaining() != 0 {
		t.Errorf("last use leaves %d", e.Remaining())
	}
	one.Events()[0] = nil
	if one.Events()[0] == nil {
		t.Error("Events returned shared storage")
	}
	rej, isRej := character.UseFavoredEnemy(id, rgrFacts(t, 4, two.State(), "")).(character.FavoredEnemyUseRejected)
	if !isRej || rej.Reason != character.RangerNoUsesRemaining {
		t.Errorf("third use at level 4 = %#v", rej)
	}
	if again, ok := character.UseFavoredEnemy(id, rgrFacts(t, 5, two.State(), "")).(character.FavoredEnemyUseAccepted); !ok || again.State().FavoredEnemySpent() != 3 {
		t.Errorf("level 5 raised the maximum: %#v", again)
	}
	if rej, ok := character.UseFavoredEnemy(id, character.RangerFacts{}).(character.FavoredEnemyUseRejected); !ok || rej.Reason != character.RangerFeatureNotAvailable {
		t.Errorf("zero facts = %#v", rej)
	}
}

// Requirement: doc/specs/class-ranger.md FR-079, FR-080 (SRD 5.2.1, "Level 14: Nature’s Veil", p. 59) - Nature’s Veil is rejected below level 14, otherwise it spends a use (the Wisdom modifier, minimum one), grants Invisible until the end of the Ranger's next turn at
// the cost of a Bonus Action and emits NaturesVeilUsed; with none left it is rejected with no_uses_remaining.
// Path: per case: happy / unhappy (boundary, state error).
func TestUseNaturesVeil(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	if rej, ok := character.UseNaturesVeil(id, rgrFacts(t, 13, character.InitialRangerState(), ""), modifierOf(t, 3)).(character.NaturesVeilRejected); !ok || rej.Reason != character.RangerFeatureNotAvailable {
		t.Errorf("level 13 = %#v", rej)
	}
	state := character.InitialRangerState()
	for want := 2; want >= 0; want-- {
		res, ok := character.UseNaturesVeil(id, rgrFacts(t, 14, state, ""), modifierOf(t, 3)).(character.NaturesVeilAccepted)
		if !ok {
			t.Fatalf("use with %d left rejected", want+1)
		}
		state = res.State()
		e := res.Events()[0].(character.NaturesVeilUsed)
		if res.Invisible() != (character.RangerConditionGrant{Condition: content.ConditionInvisible, Expiry: character.GrantEndsAtEndOfNextTurn}) || res.Cost() != character.RangerBonusAction || e.Remaining() != want || e.CharacterID() != id {
			t.Errorf("accepted = %+v, %v, %v", res.Invisible(), res.Cost(), e)
		}
		res.Events()[0] = nil
		if res.Events()[0] == nil {
			t.Error("Events returned shared storage")
		}
	}
	if rej, ok := character.UseNaturesVeil(id, rgrFacts(t, 14, state, ""), modifierOf(t, 3)).(character.NaturesVeilRejected); !ok || rej.Reason != character.RangerNoUsesRemaining {
		t.Errorf("fourth use = %#v", rej)
	}
	if _, ok := character.UseNaturesVeil(id, rgrFacts(t, 14, character.InitialRangerState(), ""), modifierOf(t, -2)).(character.NaturesVeilAccepted); !ok {
		t.Error("a negative Wisdom modifier gives no use")
	}
}

// Requirement: doc/specs/class-ranger.md FR-095 (SRD 5.2.1, "Level 15: Superior Hunter’s Defense", p. 61) - with the Hunter at level 15 and a Reaction, the Resistance to the damage type is granted until the end of the current turn at the cost of a Reaction; without
// the Hunter, below level 15 or without a Reaction it is rejected with not_hunter, feature_not_available or no_reaction.
// Path: per case: happy / unhappy (boundary, state error).
func TestUseSuperiorHuntersDefense(t *testing.T) {
	t.Parallel()
	res, ok := character.UseSuperiorHuntersDefense(rgrFacts(t, 15, character.InitialRangerState(), hunter), content.DamageFire, true).(character.SuperiorHuntersDefenseAccepted)
	if !ok || res.Grant != (character.RangerResistanceGrant{Damage: content.DamageFire, Expiry: character.GrantEndsAtEndOfCurrentTurn}) || res.Cost() != character.RangerReaction {
		t.Fatalf("accepted = %#v", res)
	}
	for _, c := range []struct {
		label    string
		facts    character.RangerFacts
		reaction bool
		want     character.RangerReason
	}{
		{"no Hunter", rgrFacts(t, 15, character.InitialRangerState(), ""), true, character.RangerNotHunter},
		{"level 14", rgrFacts(t, 14, character.InitialRangerState(), hunter), true, character.RangerFeatureNotAvailable},
		{"no Reaction", rgrFacts(t, 15, character.InitialRangerState(), hunter), false, character.RangerNoReaction},
	} {
		if rej, ok := character.UseSuperiorHuntersDefense(c.facts, content.DamageFire, c.reaction).(character.SuperiorHuntersDefenseRejected); !ok || rej.Reason != c.want {
			t.Errorf("%s = %#v, want %v", c.label, rej, c.want)
		}
	}
}

// Requirement: doc/specs/class-ranger.md FR-072, FR-077, T4 (SRD 5.2.1, "Level 10: Tireless", p. 59) - Tireless grants the 1d8 roll plus the greater of 1 and the Wisdom modifier as Temporary Hit Points: rows TL-01 to TL-08 of T4; a roll outside 1 to 8 is invalid input.
// Path: per case (T4): happy / unhappy (boundary, invalid input).
func TestTirelessTemporaryHitPoints(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label          string
		roll, modifier int
		want           int
	}{{"TL-01", 5, 3, 8}, {"TL-02", 8, 5, 13}, {"TL-03", 1, 0, 2}, {"TL-04", 1, -1, 2}, {"TL-05", 8, -5, 9}, {"TL-06", 4, 1, 5}, {"TL-07", 4, 2, 6}, {"TL-08", 3, 10, 13}} {
		got, err := character.TirelessTemporaryHitPoints(c.roll, modifierOf(t, c.modifier))
		if err != nil || got != c.want {
			t.Errorf("%s: roll %d modifier %+d = %d, %v; want %d", c.label, c.roll, c.modifier, got, err, c.want)
		}
	}
	for _, roll := range []int{0, 9, -1} {
		_, err := character.TirelessTemporaryHitPoints(roll, modifierOf(t, 3))
		wantInvalid(t, err, "roll "+strconv.Itoa(roll))
	}
}

// Requirement: doc/specs/class-ranger.md FR-036, FR-075, FR-076, FR-080 (SRD 5.2.1, "Level 1: Favored Enemy", "Level 10: Tireless", "Level 14: Nature’s Veil", pp. 58-59) - a Long Rest resets the expended uses of Favored Enemy, Tireless and Nature’s Veil and a Short Rest leaves them;
// a Short Rest at level 10 or more with Exhaustion level 1 or more returns an ExhaustionReductionGranted of 1, none at Exhaustion 0 or below level 10; a negative Exhaustion level is invalid input; the input state is not changed.
// Path: per case: happy / unhappy (boundary, rule interaction, invalid input, immutability).
func TestApplyRangerRest(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	spent := rgrState(t, character.RangerStateParts{FavoredEnemySpent: 2, TirelessSpent: 1, NaturesVeilSpent: 1, HuntersPrey: "horde_breaker"})
	long := must(character.ApplyRangerRest(id, rgrFacts(t, 14, spent, hunter), character.LongRest, 3))
	if s := long.State(); s.FavoredEnemySpent() != 0 || s.TirelessSpent() != 0 || s.NaturesVeilSpent() != 0 || len(long.Events()) != 0 {
		t.Errorf("Long Rest = %+v %v", long.State(), long.Events())
	}
	if prey, _ := long.State().HuntersPrey(); prey != "horde_breaker" || spent.FavoredEnemySpent() != 2 {
		t.Error("a Long Rest changed the choices or the input")
	}
	for _, c := range []struct {
		label      string
		level      int
		exhaustion int
		events     int
	}{{"level 10, Exhaustion 1", 10, 1, 1}, {"level 10, Exhaustion 6", 10, 6, 1}, {"level 10, Exhaustion 0", 10, 0, 0}, {"level 9, Exhaustion 1", 9, 1, 0}, {"level 20, Exhaustion 2", 20, 2, 1}} {
		short := must(character.ApplyRangerRest(id, rgrFacts(t, c.level, spent, ""), character.ShortRest, c.exhaustion))
		if short.State() != spent || len(short.Events()) != c.events {
			t.Errorf("%s: Short Rest = %+v %v", c.label, short.State(), short.Events())
		}
		for _, e := range short.Events() {
			if g, ok := e.(character.ExhaustionReductionGranted); !ok || g.Amount() != 1 || g.CharacterID() != id {
				t.Errorf("%s: event = %#v", c.label, e)
			}
		}
	}
	short := must(character.ApplyRangerRest(id, rgrFacts(t, 10, spent, ""), character.ShortRest, 1))
	short.Events()[0] = nil
	if short.Events()[0] == nil {
		t.Error("Events returned shared storage")
	}
	_, err := character.ApplyRangerRest(id, rgrFacts(t, 10, spent, ""), character.ShortRest, -1)
	wantInvalid(t, err, "negative Exhaustion")
	if other := must(character.ApplyRangerRest(id, rgrFacts(t, 10, spent, ""), 0, 1)); other.State() != spent || len(other.Events()) != 0 {
		t.Error("an undefined rest kind changed something")
	}
}

// Requirement: doc/specs/class-ranger.md FR-078, FR-084, FR-094 - the Hunter is recognised by its id alone (another subclass id, before or after "hunter" in any order, is not the Hunter), the Concentration rule protects only Hunter’s Mark whatever the id of another spell,
// a creature whose mark status is the zero value is not marked, and the zero MarkStatus and RangerGrantExpiry are neither of their constants.
// Path: per case (unhappy: invalid input, boundary, rule interaction).
func TestRangerRulesEdges(t *testing.T) {
	t.Parallel()
	for _, sub := range []content.SubclassID{"thief", "champion", "abc", "hunter_x"} {
		f := rgrFacts(t, 15, character.InitialRangerState(), sub)
		if _, ok := character.UseSuperiorHuntersDefense(f, content.DamageFire, true).(character.SuperiorHuntersDefenseRejected); !ok {
			t.Errorf("subclass %q is taken for the Hunter", sub)
		}
	}
	f := rgrFacts(t, 13, character.InitialRangerState(), "")
	for _, spell := range []content.SpellID{"zone_of_truth", "a", "hunters_mar", "hunters_mark_"} {
		if !f.DamageCanBreakConcentration(spell) {
			t.Errorf("damage cannot break Concentration on %s", spell)
		}
	}
	if character.MarkStatus(0) == character.MarkedByHuntersMark || character.MarkStatus(0) == character.UnmarkedByHuntersMark ||
		character.RangerGrantExpiry(0) == character.GrantEndsAtEndOfCurrentTurn || character.RangerGrantExpiry(0) == character.GrantEndsAtEndOfNextTurn {
		t.Error("a zero value is one of the constants")
	}
	lore := rgrFacts(t, 3, character.InitialRangerState(), hunter)
	if got := lore.HuntersLore(0, nil, nil, nil); got != (character.HuntersLoreNotMarked{}) {
		t.Errorf("a zero mark status = %#v", got)
	}
	fire := must(content.NewDamageTypeDefense(content.DamageFire, ""))
	one := []content.DefenseEntry{fire}
	for label, lists := range map[string][3][]content.DefenseEntry{"immunity and resistance": {one, one, nil}, "resistance and vulnerability": {nil, one, one}, "immunity and vulnerability": {one, nil, one}} {
		if _, ok := lore.HuntersLore(character.MarkedByHuntersMark, lists[0], lists[1], lists[2]).(character.HuntersLoreReveals); !ok {
			t.Errorf("%s reveals nothing", label)
		}
	}
	prey := rgrHunterFacts(t, 11, "colossus_slayer", "")
	if _, ok := prey.ColossusSlayer(character.ColossusSlayerFacts{WeaponHit: true, TargetHitPoints: 25, TargetMaximum: 20}); ok {
		t.Error("Colossus Slayer applied to a creature above its Hit Point maximum")
	}
	if got := prey.SuperiorHuntersPrey(character.SuperiorHuntersPreyFacts{SpellDie: dice.D6(), DifferentCreature: true, DistanceFeet: 5, Visible: true}); got != (character.RangerRiderIneligible{Reason: character.RangerRiderNotMarked}) {
		t.Errorf("a zero first-target status = %#v", got)
	}
	record := (character.MultiattackDefenseRecord{}).HitBy("orc").HitBy("zombie")
	if !record.Holds("orc") || !record.Holds("zombie") || record.Holds("goblin") || record.Holds("zzz") || record.Holds("a") {
		t.Errorf("record = %+v", record)
	}
	if again := record.HitBy("orc").HitBy("zombie"); !again.Holds("orc") || !again.Holds("zombie") {
		t.Error("hitting twice lost an attacker")
	}
}
