package combat_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: per case (table 4.4; Source for every row: SRD 5.2.1, "Resurrection" p. 159)
// Path: per case.
func TestReviveBySpellStrain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spell                   combat.RevivalSpell
		since                   time.Duration
		spirit                  combat.RevivalConsent
		want                    combat.CasterStrain
	}{
		{"364 days 23 hours", "FR-020", "unhappy (boundary)", combat.Resurrection, 365*rrDay - time.Hour, combat.SpiritAccepts, combat.StrainNone},
		{"365 days exactly", "FR-020", "unhappy (boundary)", combat.Resurrection, 365 * rrDay, combat.SpiritAccepts, combat.Strained},
		{"5,000 days", "FR-020", "happy", combat.Resurrection, 5000 * rrDay, combat.SpiritAccepts, combat.Strained},
		{"Raise Dead at its 10 day limit", "FR-020", "unhappy (rule interaction)", combat.RaiseDead, 10 * rrDay, combat.SpiritAccepts, combat.StrainNone},
		{"True Resurrection after 70,000 days", "FR-020", "unhappy (rule interaction)", combat.TrueResurrection, 70000 * rrDay, combat.SpiritAccepts, combat.StrainNone},
		{"Revivify never strains", "FR-020", "unhappy (rule interaction)", combat.Revivify, time.Minute, combat.SpiritAccepts, combat.StrainNone},
		{"the spirit refuses at 400 days", "FR-014, FR-020; Q-5", "unhappy (rule interaction)", combat.Resurrection, 400 * rrDay, combat.SpiritRefuses, combat.Strained},
		{"the spirit refuses at 364 days", "FR-014, FR-020; Q-5", "unhappy (boundary)", combat.Resurrection, 364 * rrDay, combat.SpiritRefuses, combat.StrainNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := rrCommand(t, c.spell)
			cmd.Circumstances = rrCircs(t, c.since, false, false, combat.Intact)
			cmd.Consent = c.spirit
			var got combat.CasterStrain
			switch r := combat.ReviveBySpell(cmd).(type) {
			case combat.ReviveBySpellApplied:
				got = r.Strain
			case combat.ReviveBySpellNotRevived:
				got = r.Strain
			default:
				t.Fatalf("%v: result %T, want applied or not revived", c.spell, r)
			}
			if got != c.want {
				t.Errorf("%v after %v: strain = %v, want %v [%s; %s]", c.spell, c.since, got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-020, FR-014, Q-5 - an ineligible Resurrection cast at 400 days gives no strain (the answer
// carries none), and neither does a creature that is not Dead.
// Path: unhappy (rule interaction).
func TestReviveBySpellStrainNotGivenWithoutCast(t *testing.T) {
	t.Parallel()
	cmd := rrCommand(t, combat.Resurrection)
	cmd.Circumstances = rrCircs(t, 400*rrDay, true, false, combat.Intact)
	if _, ok := combat.ReviveBySpell(cmd).(combat.ReviveBySpellIneligible); !ok {
		t.Errorf("an ineligible Resurrection gave %T, want ineligible", combat.ReviveBySpell(cmd))
	}
	raise := rrCommand(t, combat.RaiseDead)
	raise.Circumstances = rrCircs(t, 400*rrDay, false, false, combat.Intact)
	if _, ok := combat.ReviveBySpell(raise).(combat.ReviveBySpellIneligible); !ok {
		t.Errorf("Raise Dead after 400 days gave %T, want ineligible (it carries no strain)", combat.ReviveBySpell(raise))
	}
	cmd = rrCommand(t, combat.Resurrection)
	cmd.Circumstances = rrCircs(t, 400*rrDay, false, false, combat.Intact)
	cmd.Status = pcStatus(t, 4, 20, 0)
	if nr, ok := combat.ReviveBySpell(cmd).(combat.ReviveBySpellNotRevived); !ok || nr.Strain != combat.StrainNone {
		t.Errorf("a creature that is not Dead: %#v, want NotRevived with no strain", nr)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-021 - while Strained a caster cannot cast spells (cantrips included) and every D20 Test has
// one Disadvantage, a fresh slice each call; with no strain they cast and the slice is empty (SRD 5.2.1, "Resurrection", p. 159).
// Path: unhappy (rule interaction).
func TestCasterStrainEffects(t *testing.T) {
	t.Parallel()
	if combat.Strained.CanCastSpells() || !slices.Equal(combat.Strained.RollCircumstances(), []modifier.RollCircumstance{modifier.Disadvantage{}}) {
		t.Errorf("Strained: CanCastSpells %v, circumstances %v, want false and one Disadvantage", combat.Strained.CanCastSpells(), combat.Strained.RollCircumstances())
	}
	if modifier.Resolve(combat.Strained.RollCircumstances()...) != modifier.WithDisadvantage {
		t.Error("the strain does not resolve to Disadvantage")
	}
	first := combat.Strained.RollCircumstances()
	first[0] = modifier.Advantage{}
	if _, ok := combat.Strained.RollCircumstances()[0].(modifier.Disadvantage); !ok {
		t.Error("RollCircumstances is not a fresh copy")
	}
	var none combat.CasterStrain
	if !none.CanCastSpells() || len(none.RollCircumstances()) != 0 || none != combat.StrainNone {
		t.Errorf("zero strain: CanCastSpells %v, circumstances %v, want true and none", none.CanCastSpells(), none.RollCircumstances())
	}
	for s, want := range map[combat.CasterStrain]string{combat.StrainNone: "none", combat.Strained: "strained", 7: "unknown", -1: "unknown"} {
		if s.String() != want {
			t.Errorf("CasterStrain(%d).String() = %q, want %q", int(s), s.String(), want)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-022 - a Long Rest ends the strain, a Short Rest (or an invalid kind) leaves it, and a second
// strained cast while strained leaves it Strained (strain does not stack).
// Path: unhappy (boundary).
func TestCasterStrainAfterRest(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		start combat.CasterStrain
		rest  character.RestKind
		want  combat.CasterStrain
	}{{combat.Strained, character.LongRest, combat.StrainNone}, {combat.Strained, character.ShortRest, combat.Strained},
		{combat.Strained, 0, combat.Strained}, {combat.StrainNone, character.LongRest, combat.StrainNone}, {combat.StrainNone, character.ShortRest, combat.StrainNone}} {
		if got := c.start.AfterRest(c.rest); got != c.want {
			t.Errorf("%v after %v = %v, want %v", c.start, c.rest, got, c.want)
		}
	}
	cmd := rrCommand(t, combat.Resurrection)
	cmd.Circumstances = rrCircs(t, 400*rrDay, false, false, combat.Intact)
	if first := rrApplied(t, cmd).Strain; first.AfterRest(character.ShortRest) != combat.Strained {
		t.Error("a strained caster who casts again is not Strained")
	}
}
