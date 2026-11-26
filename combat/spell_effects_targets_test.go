package combat_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: spell-effects.md FR-093 - ParseSpellObjectID returns a non-blank value unchanged and an error matching ErrBlankSpellObjectID and srderr.ErrInvalidInput for a blank one; MustSpellObjectID panics with that error on a blank string and only then.
// Path: per case.
func TestSpellObjectID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in string
		wantErr  bool
	}{
		{"a name", "door", false}, {"a name with spaces", " a door ", false},
		{"empty", "", true}, {"spaces", "  \t", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			id, err := combat.ParseSpellObjectID(c.in)
			if c.wantErr {
				if !errors.Is(err, combat.ErrBlankSpellObjectID) || !errors.Is(err, srderr.ErrInvalidInput) || id != "" {
					t.Errorf("Parse(%q) = %q, %v, want ErrBlankSpellObjectID [FR-093]", c.in, id, err)
				}
				defer func() {
					if r := recover(); r == nil || !errors.Is(r.(error), combat.ErrBlankSpellObjectID) {
						t.Errorf("Must(%q) panic = %v, want ErrBlankSpellObjectID", c.in, r)
					}
				}()
				combat.MustSpellObjectID(c.in)
				return
			}
			if err != nil || string(id) != c.in || combat.MustSpellObjectID(c.in) != id {
				t.Errorf("Parse(%q) = %q, %v, want it unchanged [FR-093]", c.in, id, err)
			}
		})
	}
}

// Requirement: spell-effects.md FR-093 - ParseSpellObjectID never panics and accepts exactly the strings that are not blank, whatever bytes it is given.
// Path: unhappy (invalid input).
func FuzzParseSpellObjectID(f *testing.F) {
	for _, s := range []string{"", " ", "door", "\x00", " ", "a b", strings.Repeat("x", 4096)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		id, err := combat.ParseSpellObjectID(s)
		if blank := strings.TrimSpace(s) == ""; blank != (err != nil) {
			t.Fatalf("Parse(%q) error = %v, blank = %v", s, err, blank)
		}
		if err == nil && string(id) != s {
			t.Fatalf("Parse(%q) = %q", s, id)
		}
	})
}

// Requirement: spell-effects.md FR-093, FR-097 - NewSpellTarget needs a ref with a non-blank id, accepts the zero creature type and rejects an invalid one or one on an object, rejects an invalid validity or path, an invalid ability and a nil attempt; the saves are copied in and out.
// Path: per case.
func TestNewSpellTarget(t *testing.T) {
	t.Parallel()
	roll := d20.RollAttempt{}
	door := combat.ObjectTargetRef{ID: "door"}
	cases := []struct {
		name string
		p    combat.SpellTargetParams
		ok   bool
	}{
		{"a creature", combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: "a"}}, true},
		{"a typed creature with a save", combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: "a"}, CreatureType: content.CreatureTypeBeast, Saves: map[ability.Ability]d20.SaveAttempt{ability.Wisdom: roll}}, true},
		{"an object", combat.SpellTargetParams{Ref: door}, true},
		{"no ref", combat.SpellTargetParams{}, false},
		{"a creature without an id", combat.SpellTargetParams{Ref: combat.CreatureTargetRef{}}, false},
		{"an object without an id", combat.SpellTargetParams{Ref: combat.ObjectTargetRef{ID: " "}}, false},
		{"an invalid creature type", combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: "a"}, CreatureType: 99}, false},
		{"a negative creature type", combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: "a"}, CreatureType: -1}, false},
		{"a creature type on an object", combat.SpellTargetParams{Ref: door, CreatureType: content.CreatureTypeBeast}, false},
		{"an invalid validity", combat.SpellTargetParams{Ref: door, Validity: 7}, false},
		{"an invalid path", combat.SpellTargetParams{Ref: door, Path: -1}, false},
		{"an invalid ability", combat.SpellTargetParams{Ref: door, Saves: map[ability.Ability]d20.SaveAttempt{0: roll}}, false},
		{"a nil attempt", combat.SpellTargetParams{Ref: door, Saves: map[ability.Ability]d20.SaveAttempt{ability.Wisdom: nil}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewSpellTarget(c.p)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("err = %v, want ok = %v [FR-093]", err, c.ok)
			}
		})
	}
	saves := map[ability.Ability]d20.SaveAttempt{ability.Wisdom: roll}
	target := luMust(combat.NewSpellTarget(combat.SpellTargetParams{Ref: door, Saves: saves}))
	delete(saves, ability.Wisdom)
	if _, ok := target.Save(ability.Wisdom); !ok {
		t.Error("the caller's map changed the target's saves")
	}
	if _, ok := target.Save(ability.Dexterity); ok {
		t.Error("a save appeared for an ability not given")
	}
	if typ, known := target.CreatureType(); known || typ != 0 {
		t.Errorf("object creature type = %v, %v", typ, known)
	}
	if _, has := target.AttackFacts(); has || target.Validity() != combat.TargetCanBeAffected || target.Path() != combat.PathClear || target.Ref() != door {
		t.Error("the defaults of a target are: can be affected, a clear path, no facts")
	}
}

// Requirement: spell-effects.md FR-093 - NewSpellTarget never panics, whatever ids, types and enum values it is given, and a target it accepts has the ref it was given.
// Path: unhappy (invalid input).
func FuzzNewSpellTarget(f *testing.F) {
	f.Add("a", "", 0, 0, 0, true)
	f.Add("", "door", 99, 5, -1, false)
	f.Add(" ", " ", -1, 2, 2, true)
	f.Fuzz(func(t *testing.T, creature, object string, typ, validity, path int, asCreature bool) {
		var ref combat.SpellTargetRef = combat.ObjectTargetRef{ID: combat.SpellObjectID(object)}
		if asCreature {
			ref = combat.CreatureTargetRef{ID: combat.CombatantID(creature)}
		}
		got, err := combat.NewSpellTarget(combat.SpellTargetParams{Ref: ref, CreatureType: content.CreatureType(typ),
			Validity: combat.TargetValidity(validity), Path: combat.PathToTarget(path)})
		if err == nil && got.Ref() != ref {
			t.Fatalf("ref = %v, want %v", got.Ref(), ref)
		}
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("err = %v, want ErrInvalidInput", err)
		}
	})
}

// Requirement: spell-effects.md FR-097 - NewSpellAttackFacts keeps the facts of the host, copies their slices in and out, and rejects a negative distance; the zero value holds no facts.
// Path: unhappy (boundary, immutability).
func TestSpellAttackFacts(t *testing.T) {
	t.Parallel()
	if !(combat.SpellAttackFacts{}).IsZero() {
		t.Error("the zero facts are not zero")
	}
	if _, err := combat.NewSpellAttackFacts(combat.SpellAttackFactsParams{Distance: -1}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative distance err = %v", err)
	}
	cover := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	f, err := combat.NewSpellAttackFacts(combat.SpellAttackFactsParams{ArmorClass: 15, CoverSources: cover, Distance: 0})
	if err != nil || f.IsZero() {
		t.Fatalf("facts = %v, %v", f, err)
	}
	cover[0].Cover = combat.TotalCover
	got := f.Params()
	if got.CoverSources[0].Cover != combat.HalfCover || got.ArmorClass != 15 {
		t.Errorf("params = %+v, want the facts as given", got)
	}
	got.CoverSources[0].Cover = combat.TotalCover
	if f.Params().CoverSources[0].Cover != combat.HalfCover {
		t.Error("a returned slice changed the facts")
	}
}

// Requirement: spell-effects.md FR-031 - NewInstanceAssignment needs a part index of 0 or more and at least one valid target ref, and copies the targets in and out.
// Path: per case.
func TestNewInstanceAssignment(t *testing.T) {
	t.Parallel()
	a := combat.CreatureTargetRef{ID: "a"}
	cases := []struct {
		name    string
		part    int
		targets []combat.SpellTargetRef
		ok      bool
	}{
		{"one target", 0, []combat.SpellTargetRef{a}, true},
		{"an object target", 3, []combat.SpellTargetRef{combat.ObjectTargetRef{ID: "door"}}, true},
		{"negative index", -1, []combat.SpellTargetRef{a}, false},
		{"no target", 0, nil, false},
		{"a blank creature", 0, []combat.SpellTargetRef{combat.CreatureTargetRef{}}, false},
		{"a nil ref", 0, []combat.SpellTargetRef{nil}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewInstanceAssignment(c.part, c.targets)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("err = %v, want ok = %v", err, c.ok)
			}
		})
	}
	in := []combat.SpellTargetRef{a}
	as := luMust(combat.NewInstanceAssignment(2, in))
	in[0] = combat.CreatureTargetRef{ID: "z"}
	out := as.Targets()
	out[0] = combat.CreatureTargetRef{ID: "y"}
	if as.PartIndex() != 2 || as.Targets()[0] != a {
		t.Errorf("assignment = %d %v, want part 2 with a", as.PartIndex(), as.Targets())
	}
}

// Requirement: spell-effects.md FR-057, FR-017 - the enums of the plan have their names and the validity of their values: ten rejection reasons in the fixed order, two perceptions, two validities and two paths.
// Path: unhappy (boundary).
func TestSpellEffectsEnums(t *testing.T) {
	t.Parallel()
	want := []string{"wrong_target_count", "no_clear_path", "instance_assignment_mismatch", "missing_save_attempt", "missing_attack_facts",
		"damage_type_not_chosen", "damage_type_not_allowed", "caster_not_allowed", "wrong_target_subject", "attack_not_possible"}
	for i, w := range want {
		k := combat.SpellEffectsRejectionKind(i + 1)
		if !k.Valid() || k.String() != w {
			t.Errorf("reason %d = %q valid %v, want %q", i+1, k, k.Valid(), w)
		}
	}
	for _, k := range []combat.SpellEffectsRejectionKind{0, 11, -1} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("reason %d is valid or named %q", k, k)
		}
	}
	if combat.AppearedToSucceed.String() != "appeared_to_succeed" || combat.SpellDidNothing.String() != "spell_did_nothing" || combat.InvalidTargetPerception(0).String() != "unknown" {
		t.Error("perception names")
	}
	if combat.TargetCanBeAffected.String() != "can_be_affected" || combat.TargetCannotBeAffected.String() != "cannot_be_affected" || combat.TargetValidity(2).Valid() || combat.TargetValidity(-1).Valid() {
		t.Error("validity names or range")
	}
	if combat.PathClear.String() != "clear" || combat.PathBlockedByTotalCover.String() != "blocked_by_total_cover" || combat.PathToTarget(2).Valid() || combat.PathToTarget(-1).Valid() {
		t.Error("path names or range")
	}
}

// Requirement: spell-effects.md FR-097, FR-031 - NewSpellAttackFacts and NewInstanceAssignment never panic, whatever numbers and ids they are given; the facts accept exactly the distances of 0 or more, and an assignment exactly a part index of 0 or more with a valid target.
// Path: unhappy (invalid input).
func FuzzNewSpellAttackFactsAndAssignment(f *testing.F) {
	f.Add(0, 0, "a")
	f.Add(-1, -1, "")
	f.Add(1<<30, 7, " ")
	f.Fuzz(func(t *testing.T, distance, part int, id string) {
		facts, err := combat.NewSpellAttackFacts(combat.SpellAttackFactsParams{Distance: distance})
		if (distance < 0) != (err != nil) || (err == nil && facts.IsZero()) {
			t.Fatalf("distance %d: facts %v err %v", distance, facts, err)
		}
		a, err := combat.NewInstanceAssignment(part, []combat.SpellTargetRef{combat.CreatureTargetRef{ID: combat.CombatantID(id)}})
		valid := part >= 0 && strings.TrimSpace(id) != ""
		if valid != (err == nil) {
			t.Fatalf("part %d id %q: err %v", part, id, err)
		}
		if err == nil && a.PartIndex() != part {
			t.Fatalf("part = %d, want %d", a.PartIndex(), part)
		}
	})
}
