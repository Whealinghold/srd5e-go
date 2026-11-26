package combat_test

import (
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-traits.md Limits (plan P10) - PackTacticsCircumstances takes at most 256 allies; the 257th is a LimitExceededError and the list is unchanged.
// Path: unhappy (boundary).
func TestPackTacticsAllyCap(t *testing.T) {
	t.Parallel()
	tr := luMust(combat.NewPackTacticsTrait(mtRef(t, 0)))
	runCap(t, "pack tactics allies", 256, func(n int) error {
		allies := make([]combat.AllyFact, n)
		for i := range allies {
			allies[i] = combat.AllyFact{DistanceFeet: 30}
		}
		_, err := combat.PackTacticsCircumstances(tr, allies)
		if allies[0] != (combat.AllyFact{DistanceFeet: 30}) {
			t.Error("the allies changed")
		}
		return err
	})
}

// Requirement: doc/specs/monster-traits.md Limits (plan P10) - ApplySiegeMonster takes at most 64 damages; the 65th is a LimitExceededError and the damages are unchanged.
// Path: unhappy (boundary).
func TestApplySiegeMonsterDamageCap(t *testing.T) {
	t.Parallel()
	sm := luMust(combat.NewSiegeMonsterTrait(mtRef(t, 0)))
	runCap(t, "siege monster damages", 64, func(n int) error {
		in := make([]combat.Damage, n)
		for i := range in {
			in[i] = combat.MustDamage(3, content.DamageFire)
		}
		out, err := combat.ApplySiegeMonster(sm, in, combat.DamageRecipientObjectOrStructure)
		if err == nil && (len(out) != n || out[0].Amount() != 6) || in[0].Amount() != 3 {
			t.Errorf("out = %v, in[0] = %v, want doubled copies and an unchanged input", out, in[0])
		}
		return err
	})
}

// Requirement: doc/specs/monster-traits.md Limits (plan P10) - TriggerDeathBurst reads at most 256 events: 256 with a Died is Triggered, 257 is DeathBurstLimitExceeded whatever they hold, and a zero trait is still NotTriggered.
// Path: unhappy (boundary).
func TestTriggerDeathBurstEventCap(t *testing.T) {
	t.Parallel()
	b := luMust(combat.NewDeathBurstTrait(mtRef(t, 0), luMust(content.NewStatBlockSavingThrow(ability.Constitution, 11, "", ""))))
	events := func(n int) []combat.Event {
		out := make([]combat.Event, n)
		out[n-1] = luMust(combat.NewDied(mtID(), combat.MonsterDeath))
		return out
	}
	if _, ok := combat.TriggerDeathBurst(b, mtID(), events(256)).(combat.DeathBurstTriggered); !ok {
		t.Error("256 events with a Died did not trigger")
	}
	over, ok := combat.TriggerDeathBurst(b, mtID(), events(257)).(combat.DeathBurstLimitExceeded)
	if !ok {
		t.Errorf("257 events = %#v, want DeathBurstLimitExceeded", over)
	}
	if _, ok := combat.TriggerDeathBurst(combat.DeathBurstTrait{}, mtID(), events(257)).(combat.DeathBurstNotTriggered); !ok {
		t.Error("a zero trait with 257 events is not NotTriggered")
	}
}

// Requirement: doc/specs/monster-traits.md Limits (plan P10) - RecogniseTraits reads a text of at most 1024 bytes: a Magic Resistance whose text is 1024 bytes is recognised, one of 1025 is rejected as TraitTextMismatch without being scanned.
// Path: unhappy (boundary).
func TestRecogniseTraitsTextCap(t *testing.T) {
	t.Parallel()
	// The template with a noun of k lower-case letters is len(mtMagic) - len("drake") + k bytes.
	text := func(size int) string {
		return strings.Replace(mtMagic, "drake", strings.Repeat("a", size-len(mtMagic)+len("drake")), 1)
	}
	for size, want := range map[int]bool{1024: true, 1025: false} {
		got := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait("Magic Resistance", text(size))}, nil))
		if len(text(size)) != size {
			t.Fatalf("fixture text is %d bytes, want %d", len(text(size)), size)
		}
		recognised := len(got.Traits()) == 1
		rejected := len(got.Unrecognised()) == 1 && got.Unrecognised()[0].Reason() == combat.TraitTextMismatch
		if recognised != want || rejected == want {
			t.Errorf("%d bytes: traits %v, unrecognised %v, want recognised = %v", size, got.Traits(), got.Unrecognised(), want)
		}
	}
}
