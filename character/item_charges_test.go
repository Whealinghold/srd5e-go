package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-029).
// Path: per case.
func TestNewItemCharges(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Charges", p. 206`
	cases := []struct {
		name, requirement, path string
		max, cur                int
		want                    character.ItemChargesReason
	}{
		{"maximum 0", p, "unhappy (boundary)", 0, 0, character.ItemChargesMaximumBelowOne},
		{"maximum 1, empty", p, "unhappy (boundary)", 1, 0, 0},
		{"current -1", p, "unhappy (boundary)", 7, -1, character.ItemChargesCurrentNegative},
		{"current equals maximum", p, "unhappy (boundary)", 7, 7, 0},
		{"current above maximum", p, "unhappy (boundary)", 7, 8, character.ItemChargesCurrentAboveMaximum},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewItemCharges(c.max, c.cur)
			if c.want == 0 {
				if err != nil || got.Current() != c.cur || got.Maximum() != c.max || got.IsZero() {
					t.Errorf("= %+v, %v", got, err)
				}
				return
			}
			var rej *character.ItemChargesRejected
			if !errors.As(err, &rej) || rej.Reason != c.want || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ItemChargesRejected{Reason: c.want}) || !got.IsZero() {
				t.Errorf("err = %v, want %v [%s]", err, c.want, c.requirement)
			}
			defer func() {
				if recover() == nil {
					t.Error("MustItemCharges did not panic")
				}
			}()
			character.MustItemCharges(c.max, c.cur)
		})
	}
	for r := character.ItemChargesMaximumBelowOne; r <= character.ItemChargesCurrentAboveMaximum; r++ {
		if r.String() == "unknown" {
			t.Errorf("reason %d", r)
		}
	}
	if character.ItemChargesReason(0).String() != "unknown" {
		t.Error("String")
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-030, FR-031 - spending charges).
// Path: per case.
func TestItemChargesSpend(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Charges", p. 206`
	c5 := character.MustItemCharges(7, 5)
	cases := []struct {
		name, requirement, path string
		charges                 character.ItemCharges
		n                       int
		wantCurrent             int
		insufficient            bool
		invalid                 bool
	}{
		{"spend one", p, "happy", c5, 1, 4, false, false},
		{"spend all", p, "unhappy (boundary)", c5, 5, 0, false, false},
		{"spend one too many", p, "unhappy (state error)", c5, 6, 5, true, false},
		{"at zero", p, "unhappy (state error)", character.MustItemCharges(7, 0), 1, 0, true, false},
		{"spend zero", p, "unhappy (invalid input)", c5, 0, 5, false, true},
		{"spend negative", p, "unhappy (invalid input)", c5, -1, 5, false, true},
		{"zero charges", p, "unhappy (invalid input)", character.ItemCharges{}, 1, 0, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := c.charges.Spend(holder, inst("w1"), c.n)
			if c.invalid {
				if res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("= %v, %v", res, err)
				}
				return
			}
			switch r := res.(type) {
			case character.ItemChargesSpendApplied:
				if c.insufficient || r.Charges.Current() != c.wantCurrent || r.Charges.Maximum() != 7 || r.Event.Spent() != c.n || r.Event.CharacterID() != holder || r.Event.InstanceID() != "w1" {
					t.Errorf("applied = %+v [%s]", r, c.requirement)
				}
			case character.MagicItemInsufficientCharges:
				if !c.insufficient || r.Current != c.charges.Current() || r.Requested != c.n {
					t.Errorf("insufficient = %+v", r)
				}
			}
			if err != nil {
				t.Errorf("err = %v", err)
			}
		})
	}
	if _, err := c5.Spend(character.Identifier{}, inst("w"), 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero holder: %v", err)
	}
	if _, err := c5.Spend(holder, "", 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank instance: %v", err)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-032 - remaining charges are known when attuned or after Identify; how many it
// regains only when attuned.
// Path: per case.
func TestKnowledgeOfCharges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		standing          character.AttunementStanding
		identify          bool
		remaining, regain bool
	}{
		{character.StandingAttuned, false, true, true},
		{character.StandingAttuned, true, true, true},
		{character.StandingNotAttuned, true, true, false},
		{character.StandingNotAttuned, false, false, false},
	}
	for _, c := range cases {
		got := character.KnowledgeOfCharges(c.standing, c.identify)
		if got.RemainingKnown() != c.remaining || got.RegainKnown() != c.regain {
			t.Errorf("(%v, %v) = %+v (SRD 5.2.1, \"Charges\", p. 206)", c.standing, c.identify, got)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-034 and table "Charge recovery").
// Path: per case.
func TestRegainedFor(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Charges", p. 206`
	fixed3 := must(content.NewRegainsFixed(3))
	d6 := must(content.NewRegainsRolled(dice.MustExpression(1, dice.D6(), 0)))
	cases := []struct {
		name, requirement, path string
		charges                 character.ItemCharges
		recovery                content.ChargeRecovery
		rolled                  int
		want                    int
		invalid                 bool
	}{
		{"all", p, "happy", character.MustItemCharges(7, 2), content.RegainsAll{}, 0, 5, false},
		{"fixed, below the expended", p, "happy", character.MustItemCharges(7, 2), fixed3, 0, 3, false},
		{"fixed, above the expended", p, "unhappy (boundary)", character.MustItemCharges(7, 6), fixed3, 0, 1, false},
		{"rolled, exactly expended", p, "unhappy (boundary)", character.MustItemCharges(7, 4), d6, 3, 3, false},
		{"rolled, below", p, "happy", character.MustItemCharges(7, 1), d6, 1, 1, false},
		{"rolled, above the expended", p, "unhappy (boundary)", character.MustItemCharges(7, 6), d6, 5, 1, false},
		{"already full", p, "unhappy (empty case)", character.MustItemCharges(7, 7), content.RegainsAll{}, 0, 0, false},
		{"no stated recovery", p, "unhappy (empty case)", character.MustItemCharges(7, 2), content.NoChargeRecovery{}, 0, 0, false},
		{"rolled below zero", p, "unhappy (invalid input)", character.MustItemCharges(7, 3), d6, -1, 0, true},
		{"zero charges", p, "unhappy (invalid input)", character.ItemCharges{}, content.RegainsAll{}, 0, 0, true},
		{"nil recovery", p, "unhappy (invalid input)", character.MustItemCharges(7, 3), nil, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.RegainedFor(c.recovery, c.charges, c.rolled)
			if c.invalid != errors.Is(err, srderr.ErrInvalidInput) || got != c.want {
				t.Errorf("= %d, %v, want %d [%s]", got, err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-033, FR-035, FR-036 - an item recharges when the reported moment matches its
// schedule; a mismatch or an item that does not recharge is not due.
// Path: per case.
func TestChargeRecoveryRulesDue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		time   content.ItemRechargeTime
		moment character.RechargeMoment
		want   bool
	}{
		{content.RechargesAtNextDawn, character.RechargeNextDawn, true},
		{content.RechargesAtNextDawn, character.RechargeSpecifiedTime, false},
		{content.RechargesAtSpecifiedTime, character.RechargeSpecifiedTime, true},
		{content.RechargesAtSpecifiedTime, character.RechargeNextDawn, false},
		{content.DoesNotRecharge, character.RechargeNextDawn, false},
		{content.DoesNotRecharge, character.RechargeSpecifiedTime, false},
	}
	for _, c := range cases {
		if got := (character.ChargeRecoveryRules{}).Due(c.time, c.moment); got != c.want {
			t.Errorf("Due(%v, %v) = %v (SRD 5.2.1, \"The Next Dawn\", p. 206)", c.time, c.moment, got)
		}
	}
	if character.RechargeNextDawn.String() != "next_dawn" || character.RechargeSpecifiedTime.String() != "specified_time" || character.RechargeMoment(0).String() != "unknown" {
		t.Error("RechargeMoment.String")
	}
}
