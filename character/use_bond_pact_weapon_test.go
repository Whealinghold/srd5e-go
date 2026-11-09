package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
)

// Requirement: doc/specs/class-warlock.md FR-081 to FR-084, FR-086 - the use case loads a conjured weapon and bonds it (PactWeaponBonded, the bond stored, a conjured weapon that disappears later); a Ranged weapon and an entry that is not a weapon are refused (the ItemID as detail); a magic weapon reads no content, bonds without a lookup and is refused when another creature holds it; the command's Character is not altered.
// Path: per case (happy; unhappy: invalid input, state error).
func TestBondPactWeaponBondsTheWeapon(t *testing.T) {
	t.Parallel()
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_blade"))
	source := warWeaponSource(t)
	run := func(src character.ContentSource, cmd character.BondPactWeaponCommand) character.BondPactWeaponResult {
		res, err := must(character.NewBondPactWeapon(src)).Run(t.Context(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	conj, ok := run(source, character.BondPactWeaponCommand{Character: c, Weapon: "longsword"}).(character.BondPactWeaponApplied)
	if !ok || conj.Bond().Weapon() != "longsword" || !conj.Bond().Conjured() {
		t.Fatalf("conjured = %#v", conj)
	}
	pact, _, _ := character.ReadFeatureState(conj.Character().FeatureStates(), character.WarlockPactStateKey())
	if ev, isEv := conj.Events()[0].(character.PactWeaponBonded); !pact.Bond().Bonded() || !isEv || !ev.Conjured() {
		t.Errorf("stored %+v, events %v", pact.Bond(), conj.Events())
	}
	conj.Events()[0] = nil
	if conj.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	if _, held, _ := character.ReadFeatureState(c.FeatureStates(), character.WarlockPactStateKey()); held {
		t.Error("the command's Character was altered")
	}
	idle := contenttest.NewFakeSource()
	magic, ok := run(idle, character.BondPactWeaponCommand{Character: c, Weapon: "flame_tongue", Magic: true}).(character.BondPactWeaponApplied)
	if !ok || magic.Bond().Conjured() || magic.Bond().Weapon() != "flame_tongue" {
		t.Errorf("magic = %#v", magic)
	}
	for name, c := range map[string]struct {
		src character.ContentSource
		cmd character.BondPactWeaponCommand
		isK character.WarlockRejectionKind
	}{
		"ranged":         {source, character.BondPactWeaponCommand{Character: c, Weapon: "longbow"}, character.WarlockNotMeleeSimpleOrMartial},
		"not a weapon":   {contenttest.NewFakeSource(leatherArmor(t)), character.BondPactWeaponCommand{Character: c, Weapon: leatherArmor(t).ID()}, character.WarlockNotMeleeSimpleOrMartial},
		"attuned":        {idle, character.BondPactWeaponCommand{Character: c, Weapon: "flame_tongue", Magic: true, Claim: character.MagicWeaponAttunedByAnotherCreature}, character.WarlockAttunedByAnother},
		"bonded another": {idle, character.BondPactWeaponCommand{Character: c, Weapon: "flame_tongue", Magic: true, Claim: character.MagicWeaponBondedByAnotherWarlock}, character.WarlockBondedByAnotherWarlock},
	} {
		if rej, ok := run(c.src, c.cmd).(character.BondPactWeaponRejected); !ok || rej.Rejection.Kind != c.isK {
			t.Errorf("%s = %#v, want %v", name, rej, c.isK)
		}
	}
	if res, err := must(character.NewBondPactWeapon(source)).Run(t.Context(), character.BondPactWeaponCommand{Character: c}); res != nil || !srderrIsInvalid(err) {
		t.Errorf("zero weapon id = %v, %v", res, err)
	}
	if len(idle.Queries()) != 0 {
		t.Errorf("a magic weapon read content: %v", idle.Queries())
	}
}

// Requirement: doc/specs/class-warlock.md FR-042, FR-141, FR-142 - a conjured weapon the source lacks is a BondPactWeaponContentMissing naming the weapon; a source that fails or answers with another entry is a *srderr.PortError recorded once; a nil source is refused by the constructor.
// Path: per case (unhappy: missing content, collaborator failure).
func TestBondPactWeaponContentFailures(t *testing.T) {
	t.Parallel()
	c := warChar(t, 5, 16, "", warKnows("pact_of_the_blade"))
	warContentFailures(t, character.SpanBondPactWeapon, "pact_weapon", []content.MissingContent{content.ItemID("longsword").Missing()},
		func(src character.ContentSource, opts ...character.Option) (any, error) {
			return must(character.NewBondPactWeapon(src, opts...)).Run(t.Context(), character.BondPactWeaponCommand{Character: c, Weapon: "longsword"})
		}, func(res any) ([]content.MissingContent, bool) {
			m, ok := res.(character.BondPactWeaponContentMissing)
			return m.Missing(), ok
		})
	if !srderrIsInvalid(errOf(character.NewBondPactWeapon(nil))) {
		t.Error("a nil source was accepted")
	}
	if _, err := (character.BondPactWeapon{}).Run(t.Context(), character.BondPactWeaponCommand{Character: c, Weapon: "longsword"}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
}
