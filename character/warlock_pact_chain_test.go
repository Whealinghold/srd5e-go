package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// warMonster is a minimal stat block of the id, creature type and Challenge Rating.
func warMonster(tb testing.TB, id string, typ content.CreatureType, cr string) content.Monster {
	tb.Helper()
	scores := map[ability.Ability]ability.Score{}
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	return must(content.NewMonster(content.MonsterID(id), id, content.MustSourceCitation("Fixture", content.MonsterSection, 258), content.MonsterFields{
		Sizes: []content.CreatureSize{content.SizeTiny}, Type: typ, Alignment: content.AlignmentUnaligned, ArmorClass: 12,
		Initiative: must(content.NewStatBlockInitiative(0, 10)), HitPoints: must(content.NewHitPointsEntry(9, dice.MustExpression(2, dice.D8(), 0))),
		Speeds:    []content.SpeedEntry{must(content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, ""))},
		Abilities: must(content.NewAbilityBlock(must(ability.NewScores(scores)), saves)), PassivePerception: 10, ChallengeRating: content.MustChallengeRating(cr),
	}))
}

// warFamiliar is a pact state with a familiar of the form.
func warFamiliar() character.WarlockPactState {
	return warPact(character.WarlockPactStateParts{Familiar: "imp"})
}

// Requirement: doc/specs/class-warlock.md FR-089, FR-090 - SelectFamiliarForm accepts each of the seven special forms and refuses an eighth with the MonsterID as detail; a normal form must be a Beast of Challenge Rating 0 (a CR 1 Beast, a CR 0 Fiend are refused); without Pact of the Chain nothing is chosen; the event carries the form and the Speed already chosen is kept.
// Path: per case (happy; unhappy: invalid input; boundary: CR 0 and CR 1/8).
func TestSelectFamiliarForm(t *testing.T) {
	t.Parallel()
	knows := warKnows("pact_of_the_chain")
	beast, fiend := content.CreatureTypeBeast, content.CreatureTypeFiend
	notOffered := func(id string) character.WarlockRejection {
		return character.WarlockRejection{Kind: character.WarlockFamiliarFormNotOffered, Monster: content.MonsterID(id)}
	}
	cases := []struct {
		name    string
		s       character.WarlockState
		form    character.FamiliarForm
		monster content.Monster
		want    character.WarlockRejection
	}{
		{"imp", knows, character.SpecialFamiliarForm("imp"), warMonster(t, "imp", fiend, "1"), character.WarlockRejection{}},
		{"pseudodragon", knows, character.SpecialFamiliarForm("pseudodragon"), warMonster(t, "pseudodragon", content.CreatureTypeDragon, "1/4"), character.WarlockRejection{}},
		{"quasit", knows, character.SpecialFamiliarForm("quasit"), warMonster(t, "quasit", fiend, "1"), character.WarlockRejection{}},
		{"skeleton", knows, character.SpecialFamiliarForm("skeleton"), warMonster(t, "skeleton", content.CreatureTypeUndead, "1/4"), character.WarlockRejection{}},
		{"sphinx of wonder", knows, character.SpecialFamiliarForm("sphinx_of_wonder"), warMonster(t, "sphinx_of_wonder", content.CreatureTypeCelestial, "1"), character.WarlockRejection{}},
		{"sprite", knows, character.SpecialFamiliarForm("sprite"), warMonster(t, "sprite", content.CreatureTypeFey, "1/4"), character.WarlockRejection{}},
		{"venomous snake", knows, character.SpecialFamiliarForm("venomous_snake"), warMonster(t, "venomous_snake", beast, "1/8"), character.WarlockRejection{}},
		{"an eighth special form", knows, character.SpecialFamiliarForm("wolf"), warMonster(t, "wolf", beast, "1/4"), notOffered("wolf")},
		{"normal Beast CR 0", knows, character.NormalFamiliarForm("cat"), warMonster(t, "cat", beast, "0"), character.WarlockRejection{}},
		{"normal Beast CR 1/8", knows, character.NormalFamiliarForm("rat"), warMonster(t, "rat", beast, "1/8"), notOffered("rat")},
		{"normal non-Beast CR 0", knows, character.NormalFamiliarForm("lemure"), warMonster(t, "lemure", fiend, "0"), notOffered("lemure")},
		{"zero form", knows, character.FamiliarForm{}, warMonster(t, "cat", beast, "0"), notOffered("")},
		{"Pact of the Chain not known", warKnows("pact_of_the_tome"), character.SpecialFamiliarForm("imp"), warMonster(t, "imp", fiend, "1"),
			character.WarlockRejection{Kind: character.WarlockInvocationNotKnown, Option: "pact_of_the_chain"}},
	}
	for _, c := range cases {
		prior := warPact(character.WarlockPactStateParts{Familiar: "sprite", FamiliarSpeed: character.FamiliarFlies})
		tr := character.SelectFamiliarForm(warID, c.s, prior, c.form, c.monster)
		if c.want.Kind != 0 {
			if got := warPactRefused(t, tr); !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s: rejection = %+v, want %+v", c.name, got, c.want)
			}
			continue
		}
		ch := warPactChanged(t, tr)
		form, ok := ch.State().Familiar().Form()
		sp, _ := ch.State().Familiar().Speed()
		ev, isEv := ch.Events()[0].(character.FamiliarFormChosen)
		if !ok || form != c.form || sp != character.FamiliarFlies || len(ch.Events()) != 1 || !isEv || ev.Form() != c.form || ev.CharacterID() != warID {
			t.Errorf("%s: familiar = %+v, events %v", c.name, ch.State().Familiar(), ch.Events())
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-091 - forgoing an attack for the familiar consumes exactly one attack and the Reaction is the familiar's; it is refused without Pact of the Chain, without a familiar, with no attack left and with the Reaction spent (checked in that order).
// Path: per case (happy; unhappy: state error; boundary: 1 attack).
func TestForgoAttackForFamiliar(t *testing.T) {
	t.Parallel()
	knows := warKnows("pact_of_the_chain")
	for _, c := range []struct {
		name string
		s    character.WarlockState
		pact character.WarlockPactState
		turn character.WarlockTurnFacts
		want character.WarlockRejectionKind
		left int
	}{
		{"two attacks", knows, warFamiliar(), character.WarlockTurnFacts{AttacksRemaining: 2}, 0, 1},
		{"one attack", knows, warFamiliar(), character.WarlockTurnFacts{AttacksRemaining: 1}, 0, 0},
		{"no attack", knows, warFamiliar(), character.WarlockTurnFacts{}, character.WarlockNoAttackToForgo, 0},
		{"reaction spent", knows, warFamiliar(), character.WarlockTurnFacts{AttacksRemaining: 2, Reaction: character.ActionSpent}, character.WarlockReactionSpent, 0},
		{"no familiar", knows, character.WarlockPactState{}, character.WarlockTurnFacts{AttacksRemaining: 2}, character.WarlockNoFamiliar, 0},
		{"not known", warKnows("pact_of_the_tome"), warFamiliar(), character.WarlockTurnFacts{AttacksRemaining: 2}, character.WarlockInvocationNotKnown, 0},
	} {
		tr := character.ForgoAttackForFamiliar(warID, c.s, c.pact, c.turn)
		if c.want != 0 {
			if got := warPactRefused(t, tr); got.Kind != c.want {
				t.Errorf("%s: kind = %v, want %v", c.name, got.Kind, c.want)
			}
			continue
		}
		ch := warPactChanged(t, tr)
		if _, ok := ch.Events()[0].(character.FamiliarAttackForgone); ch.AttacksRemaining() != c.left || len(ch.Events()) != 1 || !ok {
			t.Errorf("%s: attacks = %d, events %v", c.name, ch.AttacksRemaining(), ch.Events())
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-092, FR-093, FR-096 - Investment of the Chain Master sets a Fly or Swim Speed (another value is an input error), lets the Warlock command the familiar to attack with a Bonus Action and grant it Resistance with the Reaction; each needs the invocation and a familiar, the attack needs the Bonus Action and the Resistance the Reaction, and the Resistance names a valid damage type.
// Path: per case (happy; unhappy: state error, invalid input).
func TestInvestmentOfTheChainMaster(t *testing.T) {
	t.Parallel()
	knows, none := warKnows("pact_of_the_chain", "investment_of_the_chain_master"), warKnows("pact_of_the_chain")
	for _, kind := range []character.FamiliarSpeedKind{character.FamiliarFlies, character.FamiliarSwims} {
		tr, err := character.ChooseFamiliarSpeed(warID, knows, warFamiliar(), kind)
		ch := warPactChanged(t, tr)
		sp, ok := ch.State().Familiar().Speed()
		if _, isEv := ch.Events()[0].(character.FamiliarSpeedChosen); err != nil || !ok || sp != kind || !isEv || ch.Events()[0].(character.FamiliarSpeedChosen).Speed() != kind {
			t.Errorf("speed %v: %v, %+v", kind, err, ch.State().Familiar())
		}
	}
	for _, bad := range []character.FamiliarSpeedKind{0, 3} {
		if tr, err := character.ChooseFamiliarSpeed(warID, knows, warFamiliar(), bad); tr != nil || !srderrIsInvalid(err) {
			t.Errorf("speed %d = (%v, %v), want ErrInvalidInput", bad, tr, err)
		}
	}
	tr, _ := character.ChooseFamiliarSpeed(warID, none, warFamiliar(), character.FamiliarFlies)
	if got := warPactRefused(t, tr); got.Kind != character.WarlockInvocationNotKnown || got.Option != "investment_of_the_chain_master" {
		t.Errorf("not known: %+v", got)
	}
	tr, _ = character.ChooseFamiliarSpeed(warID, knows, character.WarlockPactState{}, character.FamiliarFlies)
	if got := warPactRefused(t, tr); got.Kind != character.WarlockNoFamiliar {
		t.Errorf("no familiar: %+v", got)
	}
	if _, ok := warPactChanged(t, character.CommandFamiliarAttack(warID, knows, warFamiliar(), character.WarlockTurnFacts{})).Events()[0].(character.FamiliarAttackCommanded); !ok {
		t.Error("CommandFamiliarAttack must emit FamiliarAttackCommanded")
	}
	for name, c := range map[string]struct {
		s    character.WarlockState
		pact character.WarlockPactState
		turn character.WarlockTurnFacts
		want character.WarlockRejectionKind
	}{
		"not known":    {none, warFamiliar(), character.WarlockTurnFacts{}, character.WarlockInvocationNotKnown},
		"no familiar":  {knows, character.WarlockPactState{}, character.WarlockTurnFacts{}, character.WarlockNoFamiliar},
		"bonus action": {knows, warFamiliar(), character.WarlockTurnFacts{BonusAction: character.ActionSpent}, character.WarlockNoBonusAction},
	} {
		if got := warPactRefused(t, character.CommandFamiliarAttack(warID, c.s, c.pact, c.turn)); got.Kind != c.want {
			t.Errorf("command, %s: %v, want %v", name, got.Kind, c.want)
		}
	}
	// A spent Bonus Action does not stop the Resistance and a spent Reaction does not stop the command.
	if got := warPactChanged(t, character.CommandFamiliarAttack(warID, knows, warFamiliar(), character.WarlockTurnFacts{Reaction: character.ActionSpent})); len(got.Events()) != 1 {
		t.Error("a spent Reaction must not stop the command")
	}
	ch := warPactChanged(t, character.GrantFamiliarResistance(warID, knows, warFamiliar(), character.WarlockTurnFacts{BonusAction: character.ActionSpent}, content.DamageFire))
	if dt, ok := ch.Resistance(); !ok || dt != content.DamageFire || ch.Events()[0].(character.FamiliarResistanceGranted).DamageType() != content.DamageFire {
		t.Errorf("resistance = %v, %v", dt, ok)
	}
	if _, ok := warPactChanged(t, character.CommandFamiliarAttack(warID, knows, warFamiliar(), character.WarlockTurnFacts{})).Resistance(); ok {
		t.Error("the attack command grants no Resistance")
	}
	for name, c := range map[string]struct {
		s    character.WarlockState
		pact character.WarlockPactState
		turn character.WarlockTurnFacts
		dt   content.DamageType
		want character.WarlockRejectionKind
	}{
		"not known":      {none, warFamiliar(), character.WarlockTurnFacts{}, content.DamageFire, character.WarlockInvocationNotKnown},
		"no familiar":    {knows, character.WarlockPactState{}, character.WarlockTurnFacts{}, content.DamageFire, character.WarlockNoFamiliar},
		"reaction spent": {knows, warFamiliar(), character.WarlockTurnFacts{Reaction: character.ActionSpent}, content.DamageFire, character.WarlockReactionSpent},
		"invalid type":   {knows, warFamiliar(), character.WarlockTurnFacts{}, 0, character.WarlockDamageTypeNotOffered},
	} {
		if got := warPactRefused(t, character.GrantFamiliarResistance(warID, c.s, c.pact, c.turn, c.dt)); got.Kind != c.want {
			t.Errorf("resistance, %s: %v, want %v", name, got.Kind, c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-094, FR-095 - the familiar's Bludgeoning, Piercing or Slashing damage can become Necrotic or Radiant and nothing else (Fire damage is not changeable, Fire is not a target type, the detail is the type); the familiar forces saves at the character's own spell save DC, unchanged; both need Investment of the Chain Master.
// Path: per case (happy; unhappy: invalid input, rule interaction).
func TestFamiliarDamageTypeAndSaveDc(t *testing.T) {
	t.Parallel()
	knows := warKnows("investment_of_the_chain_master")
	for _, dealt := range []content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing} {
		for _, to := range []content.DamageType{content.DamageNecrotic, content.DamageRadiant} {
			if got, rej := character.ChangeFamiliarDamageType(knows, dealt, to); got != to || rej.Kind != 0 {
				t.Errorf("%v to %v = (%v, %+v)", dealt, to, got, rej)
			}
		}
	}
	for name, c := range map[string]struct {
		s             character.WarlockState
		dealt, target content.DamageType
		want          character.WarlockRejection
	}{
		"fire dealt":         {knows, content.DamageFire, content.DamageNecrotic, character.WarlockRejection{Kind: character.WarlockDamageNotChangeable, DamageType: content.DamageFire}},
		"necrotic dealt":     {knows, content.DamageNecrotic, content.DamageRadiant, character.WarlockRejection{Kind: character.WarlockDamageNotChangeable, DamageType: content.DamageNecrotic}},
		"fire target":        {knows, content.DamagePiercing, content.DamageFire, character.WarlockRejection{Kind: character.WarlockTargetTypeNotOffered, DamageType: content.DamageFire}},
		"psychic target":     {knows, content.DamagePiercing, content.DamagePsychic, character.WarlockRejection{Kind: character.WarlockTargetTypeNotOffered, DamageType: content.DamagePsychic}},
		"invocation unknown": {warKnows("pact_of_the_chain"), content.DamagePiercing, content.DamageRadiant, character.WarlockRejection{Kind: character.WarlockInvocationNotKnown, Option: "investment_of_the_chain_master"}},
	} {
		if got, rej := character.ChangeFamiliarDamageType(c.s, c.dealt, c.target); got != 0 || !reflect.DeepEqual(rej, c.want) {
			t.Errorf("%s = (%v, %+v), want %+v", name, got, rej, c.want)
		}
	}
	dc := character.StatsForAbility(must(character.NewFeatureSpellAbility(ability.Charisma)), ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(1))).SaveDC()
	if got, rej := character.FamiliarSaveDc(knows, dc); got != dc || rej.Kind != 0 || got.Value() != 13 {
		t.Errorf("FamiliarSaveDc = (%v, %+v), want %v unchanged", got, rej, dc)
	}
	if got, rej := character.FamiliarSaveDc(warKnows("pact_of_the_chain"), dc); got != (character.SpellSaveDC{}) || rej.Kind != character.WarlockInvocationNotKnown {
		t.Errorf("not known = (%v, %+v)", got, rej)
	}
}
