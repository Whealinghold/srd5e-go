package character_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-warlock.md FR-081, FR-082, FR-083, FR-086 - Pact of the Blade bonds a conjured Simple or Martial Melee weapon or a magic weapon the Warlock touches; a Ranged weapon, a non-weapon entry and a zero target are refused (detail: the ItemID); a magic weapon another creature is attuned to or another Warlock is bonded with is refused; without the invocation nothing is bonded; the refusals change nothing.
// Path: per case (happy; unhappy: invalid input, state error).
func TestFormPactWeaponBond(t *testing.T) {
	t.Parallel()
	knows := warKnows("pact_of_the_blade")
	cases := []struct {
		name   string
		s      character.WarlockState
		target character.PactWeaponTarget
		want   character.WarlockRejection
		conj   bool
	}{
		{"conjured simple melee", knows, character.ConjuredPactWeapon("club", content.WeaponSimple, content.WeaponMelee), character.WarlockRejection{}, true},
		{"conjured martial melee", knows, character.ConjuredPactWeapon("longsword", content.WeaponMartial, content.WeaponMelee), character.WarlockRejection{}, true},
		{"conjured ranged", knows, character.ConjuredPactWeapon("longbow", content.WeaponMartial, content.WeaponRanged), character.WarlockRejection{Kind: character.WarlockNotMeleeSimpleOrMartial, Item: "longbow"}, false},
		{"conjured non-weapon", knows, character.ConjuredPactWeapon("chain_mail", 0, 0), character.WarlockRejection{Kind: character.WarlockNotMeleeSimpleOrMartial, Item: "chain_mail"}, false},
		{"conjured melee without a category", knows, character.ConjuredPactWeapon("chain_mail", 0, content.WeaponMelee), character.WarlockRejection{Kind: character.WarlockNotMeleeSimpleOrMartial, Item: "chain_mail"}, false},
		{"conjured category without a type", knows, character.ConjuredPactWeapon("club", content.WeaponSimple, 0), character.WarlockRejection{Kind: character.WarlockNotMeleeSimpleOrMartial, Item: "club"}, false},
		{"zero target", knows, character.PactWeaponTarget{}, character.WarlockRejection{Kind: character.WarlockNotMeleeSimpleOrMartial}, false},
		{"magic unclaimed", knows, character.MagicPactWeapon("flame_tongue", character.MagicWeaponUnclaimed), character.WarlockRejection{}, false},
		{"magic attuned by another", knows, character.MagicPactWeapon("flame_tongue", character.MagicWeaponAttunedByAnotherCreature), character.WarlockRejection{Kind: character.WarlockAttunedByAnother}, false},
		{"magic bonded by another", knows, character.MagicPactWeapon("flame_tongue", character.MagicWeaponBondedByAnotherWarlock), character.WarlockRejection{Kind: character.WarlockBondedByAnotherWarlock}, false},
		{"invocation not known", warKnows("pact_of_the_chain"), character.ConjuredPactWeapon("club", content.WeaponSimple, content.WeaponMelee), character.WarlockRejection{Kind: character.WarlockInvocationNotKnown, Option: "pact_of_the_blade"}, false},
	}
	for _, c := range cases {
		tr := character.FormPactWeaponBond(warID, c.s, character.WarlockPactState{}, c.target)
		if c.want.Kind != 0 {
			if got := warPactRefused(t, tr); !reflect.DeepEqual(got, c.want) {
				t.Errorf("%s: rejection = %+v, want %+v", c.name, got, c.want)
			}
			continue
		}
		ch := warPactChanged(t, tr)
		b := ch.State().Bond()
		ev, ok := ch.Events()[0].(character.PactWeaponBonded)
		if len(ch.Events()) != 1 || !ok || ev.CharacterID() != warID || ev.Weapon() != b.Weapon() || ev.Conjured() != c.conj || b.Conjured() != c.conj || !b.Bonded() || len(b.Benefits()) != 2 {
			t.Errorf("%s: bond = %+v, events = %v", c.name, b, ch.Events())
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-081, FR-085 - bonding again ends the bond held: PactWeaponBondEnded for the old weapon comes before PactWeaponBonded for the new one, and the new bond replaces the old.
// Path: unhappy (rule interaction).
func TestFormPactWeaponBondReplacesTheBond(t *testing.T) {
	t.Parallel()
	old := warPact(character.WarlockPactStateParts{Bond: "club", BondConjured: true})
	ch := warPactChanged(t, character.FormPactWeaponBond(warID, warKnows("pact_of_the_blade"), old, character.MagicPactWeapon("flame_tongue", character.MagicWeaponUnclaimed)))
	ev := ch.Events()
	ended, ok1 := ev[0].(character.PactWeaponBondEnded)
	bonded, ok2 := ev[1].(character.PactWeaponBonded)
	if len(ev) != 2 || !ok1 || !ok2 || ended.Weapon() != "club" || !ended.Disappeared() || bonded.Weapon() != "flame_tongue" || ch.State().Bond().Weapon() != "flame_tongue" || ch.State().Bond().Conjured() {
		t.Errorf("events = %v, bond = %+v", ev, ch.State().Bond())
	}
	if old.Bond().Weapon() != "club" {
		t.Error("the input state changed")
	}
	magicOld := warPact(character.WarlockPactStateParts{Bond: "flame_tongue"})
	ev = warPactChanged(t, character.FormPactWeaponBond(warID, warKnows("pact_of_the_blade"), magicOld, character.ConjuredPactWeapon("club", content.WeaponSimple, content.WeaponMelee))).Events()
	if ended, ok := ev[0].(character.PactWeaponBondEnded); len(ev) != 2 || !ok || ended.Weapon() != "flame_tongue" || ended.Disappeared() {
		t.Errorf("replacing a magic bond: events = %v", ev)
	}
}

// Requirement: doc/specs/class-warlock.md FR-085, FR-086 - the bond ends when the feature is used again, when the weapon has been more than 5 feet away for 1 minute or more, or when the Warlock dies; 59 seconds, exactly 5 feet and an unmet condition end nothing; a conjured weapon disappears and a magic weapon remains; with no bond nothing happens.
// Path: per case (boundary: 59 s and 60 s, 5 ft and 6 ft).
func TestEndPactWeaponBond(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		facts character.BondEndFacts
		ends  bool
	}{
		{"nothing", character.BondEndFacts{}, false},
		{"used again", character.BondEndFacts{Action: character.BondActionUsedAgain}, true},
		{"6 feet for 60 seconds", character.BondEndFacts{Separation: character.WeaponSeparation{DistanceFeet: 6, Duration: time.Minute}}, true},
		{"6 feet for 59 seconds", character.BondEndFacts{Separation: character.WeaponSeparation{DistanceFeet: 6, Duration: 59 * time.Second}}, false},
		{"5 feet for an hour", character.BondEndFacts{Separation: character.WeaponSeparation{DistanceFeet: 5, Duration: time.Hour}}, false},
		{"dead", character.BondEndFacts{Life: character.WarlockDead}, true},
	}
	for _, conjured := range []bool{true, false} {
		for _, c := range cases {
			bond := warPact(character.WarlockPactStateParts{Bond: "club", BondConjured: conjured})
			ch := character.EndPactWeaponBond(warID, bond, c.facts)
			if c.ends {
				ev, ok := ch.Events()[0].(character.PactWeaponBondEnded)
				if len(ch.Events()) != 1 || !ok || ch.State().Bond().Bonded() || ev.Disappeared() != conjured || ev.Weapon() != "club" || ev.CharacterID() != warID {
					t.Errorf("%s (conjured %v): events %v, bond %+v", c.name, conjured, ch.Events(), ch.State().Bond())
				}
			} else if len(ch.Events()) != 0 || !ch.State().Equal(bond) {
				t.Errorf("%s (conjured %v): changed %v", c.name, conjured, ch.Events())
			}
		}
	}
	if ch := character.EndPactWeaponBond(warID, character.WarlockPactState{}, character.BondEndFacts{Life: character.WarlockDead}); len(ch.Events()) != 0 {
		t.Errorf("no bond: events %v", ch.Events())
	}
}

// Requirement: doc/specs/class-warlock.md FR-084, FR-087, FR-088 - a bonded pact weapon offers Charisma beside its own abilities (once, in order) and Necrotic, Psychic or Radiant beside its normal damage type (once); without a bond only the weapon's own; ChoosePactWeaponDamageType accepts exactly the listed types and refuses another with the type as detail.
// Path: per case (happy; unhappy: invalid input).
func TestPactWeaponAttackOptions(t *testing.T) {
	t.Parallel()
	bond := warPact(character.WarlockPactStateParts{Bond: "rapier"}).Bond()
	own := []ability.Ability{ability.Dexterity, ability.Strength}
	got := character.PactWeaponAttackOptions(bond, own, content.DamagePiercing)
	if want := []ability.Ability{ability.Strength, ability.Dexterity, ability.Charisma}; !slices.Equal(got.Abilities, want) {
		t.Errorf("abilities = %v, want %v", got.Abilities, want)
	}
	if want := []content.DamageType{content.DamagePiercing, content.DamageNecrotic, content.DamagePsychic, content.DamageRadiant}; !slices.Equal(got.DamageTypes, want) {
		t.Errorf("damage types = %v, want %v", got.DamageTypes, want)
	}
	if own[0] != ability.Dexterity {
		t.Error("the weapon's abilities were sorted in place")
	}
	if got := character.PactWeaponAttackOptions(bond, []ability.Ability{ability.Charisma}, content.DamageNecrotic); len(got.Abilities) != 1 || len(got.DamageTypes) != 3 {
		t.Errorf("repeats: %+v", got)
	}
	none := character.PactWeaponAttackOptions(character.PactWeaponBond{}, own, content.DamagePiercing)
	if !slices.Equal(none.Abilities, []ability.Ability{ability.Strength, ability.Dexterity}) || !slices.Equal(none.DamageTypes, []content.DamageType{content.DamagePiercing}) {
		t.Errorf("no bond: %+v", none)
	}
	for _, t2 := range []content.DamageType{content.DamagePiercing, content.DamageNecrotic, content.DamagePsychic, content.DamageRadiant} {
		if dt, rej := character.ChoosePactWeaponDamageType(bond, content.DamagePiercing, t2); dt != t2 || rej.Kind != 0 {
			t.Errorf("%v = (%v, %+v), want accepted", t2, dt, rej)
		}
	}
	want := character.WarlockRejection{Kind: character.WarlockDamageTypeNotOffered, DamageType: content.DamageFire}
	if dt, rej := character.ChoosePactWeaponDamageType(bond, content.DamagePiercing, content.DamageFire); dt != 0 || !reflect.DeepEqual(rej, want) {
		t.Errorf("Fire = (%v, %+v), want %+v", dt, rej, want)
	}
	if _, rej := character.ChoosePactWeaponDamageType(character.PactWeaponBond{}, content.DamagePiercing, content.DamageRadiant); rej.Kind != character.WarlockDamageTypeNotOffered {
		t.Errorf("no bond Radiant: %+v", rej)
	}
}
