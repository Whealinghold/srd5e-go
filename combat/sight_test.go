package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var sightR combat.SightRules

type ck = []content.ConditionKind

// Requirement: per case (doc/specs/hiding-searching-and-senses.md 4.8a, FR-049 and the rows' own ids)
// Path: per case.
func TestCanSee(t *testing.T) {
	t.Parallel()
	br, di, da := content.LightBright, content.LightDim, content.LightDarkness
	nm, mg := content.DarknessNonmagical, content.DarknessMagical
	heavily := content.ObscurementHeavily
	clear, blocked := combat.LineOfSightClear, combat.LineOfSightBlocked
	see, no := combat.SightRelationCanSee, combat.SightRelationCannotSee
	blind, invis := ck{content.ConditionBlinded}, ck{content.ConditionInvisible}
	cases := []struct {
		name, requirement string
		line              combat.LineOfSight
		sp                combat.SpaceVisibility
		held              combat.SenseRanges
		distance          int
		observer, target  ck
		want              combat.SightRelation
	}{
		{"plain sight", "FR-049", clear, space(br, 0, 0), senses(), 30, nil, nil, see},
		{"Dim Light never stops sight", "FR-049, FR-008", clear, space(di, 0, 0), senses(), 30, nil, nil, see},
		{"Darkness", "FR-049, FR-005", clear, space(da, nm, 0), senses(), 30, nil, nil, no},
		{"Darkness, Darkvision", "FR-049, FR-022", clear, space(da, nm, 0), senses(sr(dv, 60)), 30, nil, nil, see},
		{"Darkness, Darkvision beyond", "FR-049, FR-024", clear, space(da, nm, 0), senses(sr(dv, 60)), 61, nil, nil, no},
		{"magical Darkness, Darkvision", "FR-049, FR-025", clear, space(da, mg, 0), senses(sr(dv, 60)), 30, nil, nil, no},
		{"magical Darkness, Truesight", "FR-049, FR-040", clear, space(da, mg, 0), senses(sr(ts, 30)), 30, nil, nil, see},
		{"magical Darkness, Truesight beyond", "FR-049, FR-040", clear, space(da, mg, 0), senses(sr(ts, 30)), 31, nil, nil, no},
		{"heavy fog, Darkvision", "FR-049, FR-026", clear, space(br, 0, heavily), senses(sr(dv, 60)), 10, nil, nil, no},
		{"heavy fog, Truesight", "FR-049, FR-041", clear, space(br, 0, heavily), senses(sr(ts, 60)), 10, nil, nil, no},
		{"heavy fog, Blindsight", "FR-049, FR-029, FR-101", clear, space(br, 0, heavily), senses(sr(bs, 30)), 30, nil, nil, see},
		{"observer Blinded", "FR-049", clear, space(br, 0, 0), senses(), 5, blind, nil, no},
		{"Blinded, Blindsight", "FR-049, FR-028", clear, space(br, 0, 0), senses(sr(bs, 10)), 10, blind, nil, see},
		{"Blinded, Blindsight beyond", "FR-049, FR-032", clear, space(br, 0, 0), senses(sr(bs, 10)), 11, blind, nil, no},
		{"Blinded, Truesight", "FR-049, FR-047", clear, space(br, 0, 0), senses(sr(ts, 60)), 10, blind, nil, no},
		{"Invisible target", "FR-049", clear, space(br, 0, 0), senses(), 5, nil, invis, no},
		{"Invisible target, Darkvision only", "FR-049", clear, space(br, 0, 0), senses(sr(dv, 60)), 5, nil, invis, no},
		{"Invisible target, Truesight", "FR-049, FR-042", clear, space(br, 0, 0), senses(sr(ts, 30)), 30, nil, invis, see},
		{"Invisible target, Truesight beyond", "FR-049, FR-042", clear, space(br, 0, 0), senses(sr(ts, 30)), 31, nil, invis, no},
		{"Invisible target, Blindsight", "FR-049, FR-030", clear, space(br, 0, 0), senses(sr(bs, 30)), 30, nil, invis, see},
		{"Invisible, Blinded, Blindsight", "FR-049, FR-028, FR-030", clear, space(br, 0, 0), senses(sr(bs, 30)), 30, blind, invis, see},
		{"blocked, Blindsight", "FR-049, FR-031", blocked, space(br, 0, 0), senses(sr(bs, 30)), 5, nil, nil, no},
		{"blocked, Truesight", "FR-049", blocked, space(br, 0, 0), senses(sr(ts, 30)), 5, nil, nil, no},
		{"Tremorsense only, Darkness", "FR-049, FR-038", clear, space(da, nm, 0), senses(sr(trs, 60)), 10, nil, nil, no},
		{"Blindsight one beyond, Darkness", "FR-049, FR-032", clear, space(da, nm, 0), senses(sr(bs, 30)), 31, nil, nil, no},
		{"Blinded among other conditions", "FR-103, FR-049", clear, space(br, 0, 0), senses(), 5, ck{content.ConditionBlinded, content.ConditionPoisoned}, nil, no},
		{"Invisible among other conditions", "FR-103, FR-049", clear, space(br, 0, 0), senses(), 5, nil, ck{content.ConditionInvisible, content.ConditionProne}, no},
		{"other conditions only", "FR-103", clear, space(br, 0, 0), senses(), 5, ck{content.ConditionPoisoned, content.ConditionFrightened}, ck{content.ConditionProne}, see},
		{"empty sets", "FR-103", clear, space(br, 0, 0), senses(), 5, ck{}, ck{}, see},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.NewSightScene(c.distance, c.line, c.sp, c.held, c.observer, c.target)
			if err != nil {
				t.Fatal(err)
			}
			got := sightR.CanSee(s)
			if got != c.want {
				t.Errorf("CanSee = %v, want %v [%s]", got, c.want, c.requirement)
			}
			if again := sightR.CanSee(s); again != got { // FR-050
				t.Error("CanSee is not deterministic [FR-050]")
			}
		})
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-096, FR-097, FR-103 - NewSightScene rejects a negative distance, a
// line of sight that is neither clear nor blocked, an invalid space and an undefined condition kind in either set, each
// with an error matching ErrInvalidInput; the zero SightScene sees nothing.
// Path: unhappy (invalid input).
func TestNewSightSceneRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	ok := space(content.LightBright, 0, 0)
	clear := combat.LineOfSightClear
	for name, c := range map[string]struct {
		d        int
		line     combat.LineOfSight
		sp       combat.SpaceVisibility
		obs, tgt ck
	}{
		"negative distance": {-1, clear, ok, nil, nil},
		"zero line":         {1, 0, ok, nil, nil},
		"line past blocked": {1, 3, ok, nil, nil},
		"zero space":        {1, clear, combat.SpaceVisibility{}, nil, nil},
		"bad observer kind": {1, clear, ok, ck{0}, nil},
		"bad target kind":   {1, clear, ok, nil, ck{99}},
	} {
		if _, err := combat.NewSightScene(c.d, c.line, c.sp, senses(), c.obs, c.tgt); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := combat.NewSightScene(-1, clear, ok, senses(), nil, nil); !errors.Is(err, &combat.SenseRangeError{Reason: combat.SenseRangeReasonNegativeDistance}) {
		t.Errorf("negative distance error = %v", err)
	}
	if got := sightR.CanSee(combat.SightScene{}); got != combat.SightRelationCannotSee {
		t.Errorf("CanSee(zero scene) = %v, want CannotSee", got)
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-100, FR-050 - the condition sets of a SightScene are copied in and
// out: changing the caller's slice after construction, or a returned copy, changes neither the scene nor CanSee.
// Path: unhappy (immutability).
func TestSightSceneCopiesConditionSets(t *testing.T) {
	t.Parallel()
	obs, tgt := ck{content.ConditionBlinded}, ck{content.ConditionPoisoned}
	s, err := combat.NewSightScene(5, combat.LineOfSightClear, space(content.LightBright, 0, 0), senses(), obs, tgt)
	if err != nil {
		t.Fatal(err)
	}
	obs[0], tgt[0] = content.ConditionPoisoned, content.ConditionInvisible
	s.ObserverConditions()[0] = content.ConditionPoisoned
	s.TargetConditions()[0] = content.ConditionInvisible
	if !slices.Equal(s.ObserverConditions(), ck{content.ConditionBlinded}) || !slices.Equal(s.TargetConditions(), ck{content.ConditionPoisoned}) {
		t.Errorf("scene sets = %v, %v", s.ObserverConditions(), s.TargetConditions())
	}
	if got := sightR.CanSee(s); got != combat.SightRelationCannotSee {
		t.Errorf("CanSee after mutation = %v, want CannotSee (still Blinded)", got)
	}
	empty, _ := combat.NewSightScene(5, combat.LineOfSightClear, space(content.LightBright, 0, 0), senses(), nil, nil)
	if empty.ObserverConditions() == nil || empty.TargetConditions() == nil {
		t.Error("empty sets are returned as nil")
	}
}

func FuzzNewSightScene(f *testing.F) {
	f.Add(0, 1, 1, 1, 1, 1)
	f.Add(-1, 0, 0, 0, 0, 0)
	f.Add(1<<20, 9, 9, 9, 9, 9)
	f.Fuzz(func(t *testing.T, distance, line, light, kind, other, cond int) {
		v, err := combat.NewSpaceVisibility(content.LightLevel(light), content.DarknessKind(kind), content.Obscurement(other))
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("space error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if _, err := combat.NewSenseRange(content.SpecialSense(kind), distance); err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("range error %v does not match ErrInvalidInput", err)
		}
		s, err := combat.NewSightScene(distance, combat.LineOfSight(line), v, senses(), ck{content.ConditionKind(cond)}, nil)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("scene error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if got := sightR.CanSee(s); got != combat.SightRelationCanSee && got != combat.SightRelationCannotSee {
			t.Fatalf("CanSee = %v", got)
		}
	})
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-096 - a distance of 0 is a valid scene.
// Path: unhappy (boundary).
func TestNewSightSceneAcceptsDistanceZero(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewSightScene(0, combat.LineOfSightClear, space(content.LightBright, 0, 0), senses(), nil, nil); err != nil {
		t.Errorf("distance 0: %v", err)
	}
}
