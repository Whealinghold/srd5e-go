package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func warPool(tb testing.TB, n int) character.HitDicePool {
	tb.Helper()
	return must(character.NewHitDicePool(map[content.HitPointDie]int{content.HitDieD8: n}))
}

// Requirement: doc/specs/class-warlock.md FR-077, FR-078 - Lifedrinker on a pact weapon hit adds 1d6 Necrotic, Psychic or Radiant damage; expending a Hit Point Die also heals the roll plus the Constitution modifier, at least 1 (roll 1 with -1: 1; roll 8 with +3: 11), and records the die as spent; declining the die heals nothing, spends nothing and leaves the spent dice as they were; the state is unchanged.
// Path: per case (happy; boundary: heal floor).
func TestApplyLifedrinker(t *testing.T) {
	t.Parallel()
	knows := warKnows("lifedrinker")
	for _, c := range []struct {
		name   string
		dt     content.DamageType
		choice character.HitDieChoice
		con    int
		rolled int
		healed int
	}{
		{"necrotic, no die", content.DamageNecrotic, character.HitDieDecline, 3, 5, 0},
		{"psychic, die roll 8 +3", content.DamagePsychic, character.HitDieExpend, 3, 8, 11},
		{"radiant, die roll 1 -1", content.DamageRadiant, character.HitDieExpend, -1, 1, 1},
		{"necrotic, die roll 1 +0", content.DamageNecrotic, character.HitDieExpend, 0, 1, 1},
	} {
		tr, err := character.ApplyLifedrinker(warID, knows, character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, c.dt, c.choice, warPool(t, 3), character.SpentHitDice{},
			content.HitDieD8, warModifier(c.con), c.rolled)
		ch := warChanged(t, tr)
		dmg, hasDmg := ch.Damage()
		healed, hasHealed := ch.HitPointsRegained()
		spent, hasSpent := ch.SpentHitDice()
		ev, ok := ch.Events()[0].(character.LifedrinkerUsed)
		if err != nil || !hasDmg || dmg.Notation() != "1d6" || hasHealed != (c.choice == character.HitDieExpend) || healed != c.healed ||
			hasSpent != hasHealed || (hasSpent && spent.Count(content.HitDieD8) != 1) || !ok || ev.DamageType() != c.dt || ev.HitPointsRegained() != c.healed || ev.CharacterID() != warID ||
			len(ch.Events()) != 1 || !ch.State().Equal(knows) {
			t.Errorf("%s: err %v, damage %v, healed %d/%v, spent %v/%v, events %v", c.name, err, dmg, healed, hasHealed, spent, hasSpent, ch.Events())
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-077, FR-078, FR-149 - Lifedrinker is refused without the invocation, on a miss or a hit with another weapon, a second time in a turn, for Fire (detail: the type) and, when a die is expended, when none of that size remains (a die spent by a first Lifedrinker cannot be spent again), the pool has none of the size or the spent count exceeds the pool; an invalid die or a roll outside its faces is an input error; a refusal changes nothing.
// Path: per case (unhappy: state error, invalid input, boundary).
func TestApplyLifedrinkerRefusals(t *testing.T) {
	t.Parallel()
	knows := warKnows("lifedrinker")
	pool := warPool(t, 1)
	spentOne := must(character.NewSpentHitDice(map[content.HitPointDie]int{content.HitDieD8: 1}))
	type tc struct {
		name   string
		s      character.WarlockState
		attack character.WarlockAttackKind
		turn   character.WarlockTurnFacts
		dt     content.DamageType
		pool   character.HitDicePool
		spent  character.SpentHitDice
		die    content.HitPointDie
		want   character.WarlockRejection
	}
	hit, necrotic, expend := character.WarlockPactWeaponHit, content.DamageNecrotic, character.HitDieExpend
	for _, c := range []tc{
		{"not known", warKnows("eldritch_smite"), hit, character.WarlockTurnFacts{}, necrotic, pool, character.SpentHitDice{}, content.HitDieD8, character.WarlockRejection{Kind: character.WarlockInvocationNotKnown, Option: "lifedrinker"}},
		{"miss", knows, character.WarlockMissed, character.WarlockTurnFacts{}, necrotic, pool, character.SpentHitDice{}, content.HitDieD8, character.WarlockRejection{Kind: character.WarlockNotAHit}},
		{"other weapon", knows, character.WarlockOtherHit, character.WarlockTurnFacts{}, necrotic, pool, character.SpentHitDice{}, content.HitDieD8, character.WarlockRejection{Kind: character.WarlockNotAHit}},
		{"second use", knows, hit, character.WarlockTurnFacts{UsedThisTurn: []character.OncePerTurnFeature{character.OnceLifedrinker}}, necrotic, pool, character.SpentHitDice{}, content.HitDieD8, character.WarlockRejection{Kind: character.WarlockAlreadyUsedThisTurn}},
		{"fire", knows, hit, character.WarlockTurnFacts{}, content.DamageFire, pool, character.SpentHitDice{}, content.HitDieD8, character.WarlockRejection{Kind: character.WarlockDamageTypeNotOffered, DamageType: content.DamageFire}},
		{"die already spent", knows, hit, character.WarlockTurnFacts{}, necrotic, pool, spentOne, content.HitDieD8, character.WarlockRejection{Kind: character.WarlockNoHitPointDie}},
		{"no die of that size", knows, hit, character.WarlockTurnFacts{}, necrotic, pool, character.SpentHitDice{}, content.HitDieD10, character.WarlockRejection{Kind: character.WarlockNoHitPointDie}},
		{"spent exceeds the pool", knows, hit, character.WarlockTurnFacts{}, necrotic, pool, must(character.NewSpentHitDice(map[content.HitPointDie]int{content.HitDieD8: 2})), content.HitDieD8, character.WarlockRejection{Kind: character.WarlockNoHitPointDie}},
	} {
		tr, err := character.ApplyLifedrinker(warID, c.s, c.attack, c.turn, c.dt, expend, c.pool, c.spent, c.die, warModifier(0), 1)
		if got := warRefused(t, tr); err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: (%+v, %v), want %+v", c.name, got, err, c.want)
		}
	}
	// Declining the die needs no pool and ignores the Hit Point Die arguments.
	if _, err := character.ApplyLifedrinker(warID, knows, hit, character.WarlockTurnFacts{}, necrotic, character.HitDieDecline, character.HitDicePool{}, character.SpentHitDice{}, 0, warModifier(0), 0); err != nil {
		t.Errorf("declining the die: %v", err)
	}
	for name, c := range map[string]struct {
		die    content.HitPointDie
		rolled int
	}{"invalid die": {0, 1}, "roll 0": {content.HitDieD8, 0}, "roll above the faces": {content.HitDieD8, 9}} {
		if tr, err := character.ApplyLifedrinker(warID, knows, hit, character.WarlockTurnFacts{}, necrotic, expend, pool, character.SpentHitDice{}, c.die, warModifier(0), c.rolled); tr != nil || !srderrIsInvalid(err) {
			t.Errorf("%s = (%v, %v), want ErrInvalidInput", name, tr, err)
		}
	}
	if tr, err := character.ApplyLifedrinker(warID, knows, hit, character.WarlockTurnFacts{}, necrotic, expend, pool, character.SpentHitDice{}, content.HitDieD8, warModifier(0), 8); err != nil || tr == nil {
		t.Errorf("roll at the faces: (%v, %v)", tr, err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-078 - HitDieChoice is a closed set: only HitDieExpend expends a die, so a value outside the set is read as a decline (no input error for the die arguments, nothing healed or spent).
// Path: unhappy (boundary: value above HitDieExpend).
func TestApplyLifedrinkerUnknownChoiceDeclines(t *testing.T) {
	t.Parallel()
	tr, err := character.ApplyLifedrinker(warID, warKnows("lifedrinker"), character.WarlockPactWeaponHit, character.WarlockTurnFacts{}, content.DamageNecrotic,
		character.HitDieChoice(2), warPool(t, 1), character.SpentHitDice{}, 0, warModifier(0), 0)
	ch := warChanged(t, tr)
	_, hasHealed := ch.HitPointsRegained()
	_, hasSpent := ch.SpentHitDice()
	if err != nil || hasHealed || hasSpent {
		t.Errorf("unknown choice: err %v, healed %v, spent %v; want a decline", err, hasHealed, hasSpent)
	}
}
