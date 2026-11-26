package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-060, table P - faces 1 to 7 give red to violet with the damage types Fire, Acid, Lightning, Poison and Cold for the first five, and 8 is the special roll-twice row (SRD 5.2.1, "Prismatic Rays", pp. 154-155); a face outside 1 to 8 is ErrFaceOutOfRange.
// Path: per case.
func TestPrismaticRayFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		face                    int
		color                   combat.PrismaticColor
		special                 bool
		damage                  content.DamageType
	}{
		{"P1 red", "FR-060", "unhappy (boundary)", 1, combat.PrismaticRed, false, content.DamageFire},
		{"P2 orange", "FR-060", "happy", 2, combat.PrismaticOrange, false, content.DamageAcid},
		{"P3 yellow", "FR-060", "happy", 3, combat.PrismaticYellow, false, content.DamageLightning},
		{"P4 green", "FR-060", "happy", 4, combat.PrismaticGreen, false, content.DamagePoison},
		{"P5 blue", "FR-060", "happy", 5, combat.PrismaticBlue, false, content.DamageCold},
		{"P6 indigo", "FR-060", "happy", 6, combat.PrismaticIndigo, false, 0},
		{"P7 violet", "FR-060", "unhappy (boundary)", 7, combat.PrismaticViolet, false, 0},
		{"P8 special", "FR-060", "unhappy (boundary)", 8, 0, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			color, special, err := combat.PrismaticRayFor(c.face)
			dt, isDamage := color.DamageType()
			if err != nil || color != c.color || special != c.special || dt != c.damage || isDamage != (c.damage != 0) {
				t.Errorf("PrismaticRayFor(%d) = %v, %v, %v with damage %v, want %v [%s%s]", c.face, color, special, err, dt, c.color, otSpec, c.requirement)
			}
			if c.face < 8 && (color.Order() != c.face || !color.Valid()) {
				t.Errorf("order of %v = %d, want %d", color, color.Order(), c.face)
			}
		})
	}
	for _, face := range []int{0, 9, -1} {
		if _, _, err := combat.PrismaticRayFor(face); !errors.Is(err, combat.ErrFaceOutOfRange) {
			t.Errorf("PrismaticRayFor(%d) = %v, want ErrFaceOutOfRange", face, err)
		}
	}
	if combat.PrismaticColor(0).Order() != 0 || combat.PrismaticColor(8).Valid() || combat.PrismaticColor(0).String() != "unknown" || combat.PrismaticViolet.String() != "violet" {
		t.Error("an invalid colour has order 0 and no name")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-065, FR-066 - a damage ray gives the full roll as a Damage of the colour's type, halved on a successful save only; indigo on a failed save gives Restrained with an Indigo follow-up at 0 and 0, violet on a failed save gives Blinded with the Violet follow-up pending, and neither gives anything on a successful save (SRD 5.2.1, "Prismatic Rays", pp. 154-155).
// Path: per case.
func TestPrismaticStrikeFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		color                   combat.PrismaticColor
		outcome                 d20.Outcome
		wantOK                  bool
		damage                  content.DamageType
		halved                  bool
		condition               content.ConditionKind
	}{
		{"red failed", "FR-065", "happy", combat.PrismaticRed, d20.Failure, true, content.DamageFire, false, 0},
		{"red saved halves", "FR-065", "happy", combat.PrismaticRed, d20.Success, true, content.DamageFire, true, 0},
		{"orange failed", "FR-065", "happy", combat.PrismaticOrange, d20.Failure, true, content.DamageAcid, false, 0},
		{"yellow saved", "FR-065", "happy", combat.PrismaticYellow, d20.Success, true, content.DamageLightning, true, 0},
		{"green failed", "FR-065", "happy", combat.PrismaticGreen, d20.Failure, true, content.DamagePoison, false, 0},
		{"blue saved", "FR-065", "happy", combat.PrismaticBlue, d20.Success, true, content.DamageCold, true, 0},
		{"indigo failed", "FR-066", "happy", combat.PrismaticIndigo, d20.Failure, true, 0, false, content.ConditionRestrained},
		{"indigo saved gives nothing", "FR-066", "unhappy (rule interaction)", combat.PrismaticIndigo, d20.Success, false, 0, false, 0},
		{"violet failed", "FR-066", "happy", combat.PrismaticViolet, d20.Failure, true, 0, false, content.ConditionBlinded},
		{"violet saved gives nothing", "FR-066", "unhappy (rule interaction)", combat.PrismaticViolet, d20.Success, false, 0, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok, err := combat.PrismaticStrikeFor(c.color, 36, c.outcome)
			if err != nil || ok != c.wantOK {
				t.Fatalf("PrismaticStrikeFor(%v) = %v, %v, %v, want ok=%v [%s%s]", c.color, got, ok, err, c.wantOK, otSpec, c.requirement)
			}
			switch s := got.(type) {
			case combat.PrismaticDamageStrike:
				if s.Damage() != combat.MustDamage(36, c.damage) || s.Halved() != c.halved {
					t.Errorf("damage strike = %v halved %v, want 36 %v halved %v", s.Damage(), s.Halved(), c.damage, c.halved)
				}
			case combat.PrismaticConditionStrike:
				follow := s.FollowUp()
				_, indigo := follow.(combat.IndigoFollowUp)
				_, violet := follow.(combat.VioletFollowUpPending)
				if s.Condition() != c.condition || indigo != (c.color == combat.PrismaticIndigo) || violet != (c.color == combat.PrismaticViolet) {
					t.Errorf("condition strike = %v with follow-up %#v", s.Condition(), follow)
				}
				if i, ok := follow.(combat.IndigoFollowUp); ok && i.Track != (combat.IndigoSaveTrack{}) {
					t.Errorf("the Indigo track starts at 0 and 0, got %v", i.Track)
				}
			}
		})
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-065, FR-066 - an invalid colour and a negative damage total are rejected with an invalid input.
// Path: unhappy (invalid input).
func TestPrismaticStrikeForInvalidInput(t *testing.T) {
	t.Parallel()
	if _, _, err := combat.PrismaticStrikeFor(0, 10, d20.Failure); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero colour = %v, want ErrInvalidInput", err)
	}
	if _, _, err := combat.PrismaticStrikeFor(combat.PrismaticRed, -1, d20.Failure); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative damage = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-071 - the Indigo track starts at 0 and 0, frees on the third success and petrifies on the third failure, whether or not they are consecutive; counts of 2 and 2 still continue; NewIndigoSaveTrack rejects 3 or more of either and a negative count (SRD 5.2.1, "Prismatic Rays", Indigo, p. 155).
// Path: per case.
func TestIndigoSaveTrack(t *testing.T) {
	t.Parallel()
	type step = d20.Outcome
	cases := []struct {
		name, requirement, path string
		saves                   []step
		want                    string
	}{
		{"three successes", "FR-071", "unhappy (boundary)", []step{d20.Success, d20.Success, d20.Success}, "freed"},
		{"three failures", "FR-071", "unhappy (boundary)", []step{d20.Failure, d20.Failure, d20.Failure}, "petrified"},
		{"two and two continue", "FR-071", "unhappy (boundary)", []step{d20.Success, d20.Failure, d20.Success, d20.Failure}, "continues 2 2"},
		{"not consecutive still frees", "FR-071", "unhappy (rule interaction)", []step{d20.Success, d20.Failure, d20.Success, d20.Failure, d20.Success}, "freed"},
		{"not consecutive still petrifies", "FR-071", "unhappy (rule interaction)", []step{d20.Failure, d20.Success, d20.Failure, d20.Success, d20.Failure}, "petrified"},
		{"one save", "FR-071", "happy", []step{d20.Failure}, "continues 0 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var track combat.IndigoSaveTrack
			var last combat.IndigoStep
			for _, o := range c.saves {
				last = track.Record(o)
				if cont, ok := last.(combat.IndigoContinues); ok {
					track = cont.Track
				}
			}
			got := ""
			switch v := last.(type) {
			case combat.IndigoFreed:
				got = "freed"
			case combat.IndigoPetrified:
				got = "petrified"
			case combat.IndigoContinues:
				got = "continues " + string(rune('0'+v.Track.Successes())) + " " + string(rune('0'+v.Track.Failures()))
			}
			if got != c.want {
				t.Errorf("after %v = %q, want %q [%s%s]", c.saves, got, c.want, otSpec, c.requirement)
			}
		})
	}
	for _, bad := range [][2]int{{3, 0}, {0, 3}, {-1, 0}, {0, -1}, {4, 4}} {
		if _, err := combat.NewIndigoSaveTrack(bad[0], bad[1]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewIndigoSaveTrack(%d, %d) = %v, want ErrInvalidInput", bad[0], bad[1], err)
		}
	}
	if got, err := combat.NewIndigoSaveTrack(2, 2); err != nil || got.Successes() != 2 || got.Failures() != 2 {
		t.Errorf("NewIndigoSaveTrack(2, 2) = %v, %v", got, err)
	}
	if _, ok := (combat.IndigoSaveTrack{}).Record(0).(combat.IndigoContinues); !ok {
		t.Error("an outcome that is not a success counts as a failure and the first one continues")
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-072 - the Violet Wisdom save ends the blindness on a success and banishes on a failure (SRD 5.2.1, "Prismatic Rays", Violet, p. 155).
// Path: per case.
func TestResolveVioletFollowUp(t *testing.T) {
	t.Parallel()
	if got := combat.ResolveVioletFollowUp(d20.Success); got != (combat.VioletBlindnessEnds{}) {
		t.Errorf("success = %#v, want VioletBlindnessEnds", got)
	}
	if got := combat.ResolveVioletFollowUp(d20.Failure); got != (combat.VioletBanished{}) {
		t.Errorf("failure = %#v, want VioletBanished", got)
	}
}
