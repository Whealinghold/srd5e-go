package combat_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-traits.md FR-002, FR-003 - a noun is a run of lower-case words (a to z) separated by single spaces, so a space before the full stop of the Swarm template is a text mismatch, never a longer noun (mutation gap: nounEnds kept a trailing space).
// Path: per case.
func TestRecogniseNounStopsBeforeATrailingSpace(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, part, text string }{
		{"space before the full stop", "Swarm", strings.Replace(mtSwarm, "insect.", "insect .", 1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{mtTrait(c.part, c.text)}, nil))
			if rej := r.Unrecognised(); len(r.Traits()) != 0 || len(rej) != 1 || rej[0].Reason() != combat.TraitTextMismatch {
				t.Errorf("traits %v rejections %v, want one TraitTextMismatch", r.Traits(), rej)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-096, FR-135, FR-161 - the Unarmed Strike command is refused with ErrInvalidInput when the scene distance is below the Strike command's, when the Parry choice is a zero trait or an unknown choice, and when the Concentration input is the zero tracking (mutation gap: only a larger scene distance was tested; the Parry and Concentration refusals were never exercised on this command).
// Path: per case.
func TestPerformUnarmedStrikeCommandFurtherRefusals(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	far := unarmedParams(grappleOption(t, ability.Strength, strengthSave(t)))
	far.Distance = 10
	cases := []struct {
		name string
		edit func(*combat.PerformUnarmedStrikeParams)
	}{
		{"scene distance below the strike's", func(p *combat.PerformUnarmedStrikeParams) { p.Strike = unarmedCommand(t, far) }},
		{"Parry with the zero trait", func(p *combat.PerformUnarmedStrikeParams) { p.TargetParry = combat.ParryIfAble{} }},
		{"unknown Parry choice", func(p *combat.PerformUnarmedStrikeParams) { p.TargetParry = struct{ combat.ParryChoice }{} }},
		{"zero Concentration tracking", func(p *combat.PerformUnarmedStrikeParams) { p.TargetConcentration = combat.TracksConcentration{} }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resStrikeParams(t, e, "a", "b", grappleOption(t, ability.Strength, strengthSave(t)))
			c.edit(&p)
			if _, err := combat.NewPerformUnarmedStrikeCommand(p); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-090, FR-096 - the zero use case with a valid command, and a real use case with the zero command, each return ErrInvalidInput and no result; neither may run (mutation gap: the guard needs both halves).
// Path: unhappy (invalid input).
func TestPerformUnarmedStrikeRunRefusesTheZeroValues(t *testing.T) {
	t.Parallel()
	valid, err := combat.NewPerformUnarmedStrikeCommand(resStrikeParams(t, encABC(t), "a", "b", grappleOption(t, ability.Strength, strengthSave(t))))
	if err != nil {
		t.Fatal(err)
	}
	if res, err := (combat.PerformUnarmedStrike{}).Run(ctxBackground(), valid); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	roller := newFixed(5)
	uc := luMust(combat.NewPerformUnarmedStrike(roller))
	if res, err := uc.Run(ctxBackground(), combat.PerformUnarmedStrikeCommand{}); !errors.Is(err, srderr.ErrInvalidInput) || res != nil || len(roller.Requests()) != 0 {
		t.Errorf("zero command = %v, %v with %d rolls", res, err, len(roller.Requests()))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-031, FR-043, FR-044 - the Damage option reports its Hit Kind and its outcome: a hit is a Normal Hit, a natural 20 and an adjacent Paralyzed target are Critical Hits, and a miss is a miss outcome (mutation gap: the unarmed damage has no die, so only the Hit Kind and the outcome tell a Critical Hit or a miss from a hit).
// Path: per case.
func TestPerformUnarmedStrikeReportsItsHitKind(t *testing.T) {
	t.Parallel()
	damage := combat.UnarmedStrikeDamageOption{ArmorClass: 15}
	paralyzed, _ := encStart(t, resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB, combat.Paralyzed{}))
	cases := []struct {
		name  string
		e     combat.Encounter
		faces []int
		hit   bool
		kind  combat.HitKind
	}{
		{"a hit is Normal", encABC(t), []int{14}, true, combat.NormalHit},
		{"a natural 20 is Critical", encABC(t), []int{20}, true, combat.CriticalHit},
		{"an adjacent Paralyzed target is Critical", paralyzed, []int{2, 14}, true, combat.CriticalHit},
		{"a miss", encABC(t), []int{9}, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, _ := resStrike(t, resStrikeParams(t, c.e, "a", "b", damage), c.faces...)
			res := got.Strike().(combat.UnarmedStrikeDamageResolved)
			out := got.Events()[2].(combat.UnarmedStrikePerformed).Outcome().(combat.UnarmedDamageOptionPerformed).Outcome()
			if !c.hit {
				if _, ok := out.(combat.AttackMissOutcome); !ok || res.Hit() {
					t.Errorf("outcome %#v, hit %v, want a miss outcome", out, res.Hit())
				}
				return
			}
			hit, ok := out.(combat.AttackHitOutcome)
			if !ok || hit.HitKind() != c.kind || res.HitKind() != c.kind {
				t.Errorf("outcome %#v, resolved kind %v, want a hit of kind %v", out, res.HitKind(), c.kind)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-161, FR-116 - when the target's Parry is used, the attacker's spent action stays spent: the Parry step is built on the tracker after the attack cost, so committing it does not undo the cost (mutation gap: Parry planned on the tracker before the cost lost the attacker's action).
// Path: unhappy (rule interaction).
func TestPerformUnarmedStrikeParryKeepsTheAttackCost(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	p := resStrikeParams(t, e, "a", "b", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	p.TargetParry = parryChoice(t, true)
	got, _ := resStrike(t, p, 10) // 10 + 5 = 15 hits AC 15, and Parry's +2 turns it into a miss
	if !got.Encounter().Tracker().ReactionSpent(cid("b")) {
		t.Fatal("the Parry was not used, so the test proves nothing")
	}
	if got.Encounter().Tracker().Resources() == e.Tracker().Resources() {
		t.Errorf("the attacker's action is unspent after a Parry: %+v", got.Encounter().Tracker().Resources())
	}
}

// Requirement: doc/specs/monster-traits.md FR-002, FR-003 - an integer slot needs at least one digit, a damage-type slot at least one letter and a noun slot at least one lower-case word, so a template with an empty slot is a text mismatch and never a recognised trait (mutation gap: runLength == 0 checks and the first-word check of nounEnds).
// Path: per case.
func TestMutgapTraitTemplateSlotsNeedTheirText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		part content.StatBlockPart
	}{
		{"integer slot without digits", mtTrait("Regeneration", strings.Replace(mtRegenOne, "10", "", 1))},
		{"damage type slot without letters", mtTrait("Regeneration", strings.Replace(mtRegenTwo, "Acid or Fire", " or Fire", 1))},
		{"noun slot without a word", mtTextPart("Legendary Resistance", strings.Replace(mtLegendary, "the drake fails", "the  fails", 1), luPerDay(3, 4))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := combat.RecogniseTraits(mtMonster(t, []content.StatBlockPart{c.part}, nil))
			if rej := r.Unrecognised(); len(r.Traits()) != 0 || len(rej) != 1 || rej[0].Reason() != combat.TraitTextMismatch {
				t.Errorf("traits %v rejections %v, want one TraitTextMismatch", r.Traits(), rej)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-019, FR-021 - an Unarmed Strike whose cost is already paid asks the tracker for nothing: the strike resolves, no tracker step is committed and the attacker's action stays as it was (mutation gap: the nil cost step of costStep was never run through PerformUnarmedStrike, so the nil checks on the step were not exercised).
// Path: unhappy (rule interaction).
func TestMutgapPerformUnarmedStrikeAlreadyPaidSpendsNothing(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	p := resStrikeParams(t, e, "a", "b", combat.UnarmedStrikeDamageOption{ArmorClass: 15})
	p.Cost = combat.AttackAlreadyPaid{}
	p.TargetParry = parryChoice(t, true)
	got, _ := resStrike(t, p, 10) // a hit that Parry turns into a miss, so the Parry step is the only tracker step
	if got.Encounter().Tracker().Resources() != e.Tracker().Resources() {
		t.Errorf("resources after an already-paid strike = %+v, want %+v (Parry spends the target's reaction only)", got.Encounter().Tracker().Resources(), e.Tracker().Resources())
	}
	if !got.Encounter().Tracker().ReactionSpent(cid("b")) {
		t.Error("the Parry was not used, so the test proves nothing")
	}
}
