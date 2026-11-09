package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func advRogue(t *testing.T) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "rogue", die: "D8", kind: "allOf", abilities: []string{"dexterity"}, subclass: []string{"thief"}})
}

func levelsChar(t *testing.T, entries ...character.ClassLevel) character.Character {
	t.Helper()
	return restored(t, 0, 50, flatScores(t, 10), entries...)
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-041, FR-042,
// FR-044 to FR-047 - Table SC, one row each: the choice is accepted at and above the
// subclass level in the class (not the total level), refused for a class not held,
// a second choice and an unlisted subclass, and the reason reported follows the
// fixed order.
// Path: per case.
func TestSubclassSelectionChoose(t *testing.T) {
	t.Parallel()
	cleric := advCleric(t, "life_domain", "light_domain")
	rogue := advRogue(t)
	type held struct{ class, subclass string }
	for _, c := range []struct {
		name, requirement, path string
		levels                  []character.ClassLevel
		class                   content.ClassDefinition
		subclass                string
		held                    []held
		want                    character.SubclassRejectionReason // 0 for accepted
	}{
		{"at the subclass level", aiSpec + " FR-041, SRD 5.2.1, Level 3: Cleric Subclass, p. 37", "happy", []character.ClassLevel{cl("cleric", 3)}, cleric, "life_domain", nil, 0},
		{"above the subclass level", aiSpec + " FR-044", "unhappy (boundary)", []character.ClassLevel{cl("cleric", 5)}, cleric, "life_domain", nil, 0},
		{"one below the subclass level", aiSpec + " FR-044", "unhappy (boundary)", []character.ClassLevel{cl("cleric", 2)}, cleric, "life_domain", nil, character.SubclassBelowSubclassLevel},
		{"multiclass, class level too low", aiSpec + " FR-044 (the class level, not the total level)", "unhappy (boundary)", []character.ClassLevel{cl("cleric", 2), cl("fighter", 4)}, cleric, "life_domain", nil, character.SubclassBelowSubclassLevel},
		{"multiclass, one in each", aiSpec + " FR-045 (SRD 5.2.1, Level 3: Rogue Subclass, p. 62)", "happy", []character.ClassLevel{cl("cleric", 3), cl("rogue", 3)}, rogue, "thief", []held{{"cleric", "life_domain"}}, 0},
		{"class not held", aiSpec + " FR-042", "unhappy (state error)", []character.ClassLevel{cl("fighter", 4)}, cleric, "life_domain", nil, character.SubclassNotInClass},
		{"already chosen, same", aiSpec + " FR-045", "unhappy (state error)", []character.ClassLevel{cl("cleric", 3)}, cleric, "life_domain", []held{{"cleric", "life_domain"}}, character.SubclassAlreadyChosen},
		{"already chosen, different", aiSpec + " FR-045", "unhappy (state error)", []character.ClassLevel{cl("cleric", 3)}, cleric, "light_domain", []held{{"cleric", "life_domain"}}, character.SubclassAlreadyChosen},
		{"unknown subclass", aiSpec + " FR-046", "unhappy (invalid input)", []character.ClassLevel{cl("cleric", 3)}, cleric, "no_such_subclass", nil, character.SubclassUnknownSubclass},
		{"another class's subclass", aiSpec + " FR-046", "unhappy (invalid input)", []character.ClassLevel{cl("cleric", 3)}, cleric, "thief", nil, character.SubclassUnknownSubclass},
		{"order: not in class beats all", aiSpec + " FR-047", "unhappy (rule interaction)", []character.ClassLevel{cl("fighter", 4)}, cleric, "no_such_subclass", []held{{"cleric", "life_domain"}}, character.SubclassNotInClass},
		{"order: below level beats already chosen", aiSpec + " FR-047", "unhappy (rule interaction)", []character.ClassLevel{cl("cleric", 2)}, cleric, "no_such_subclass", []held{{"cleric", "life_domain"}}, character.SubclassBelowSubclassLevel},
		{"order: already chosen beats unknown", aiSpec + " FR-047", "unhappy (rule interaction)", []character.ClassLevel{cl("cleric", 3)}, cleric, "no_such_subclass", []held{{"cleric", "life_domain"}}, character.SubclassAlreadyChosen},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := levelsChar(t, c.levels...)
			for _, h := range c.held {
				before = holding(t, before, h.class, h.subclass)
			}
			snapshot := before
			tr, err := character.SubclassSelection{}.Choose(before, c.class, content.SubclassID(c.subclass))
			if err != nil {
				t.Fatal(err)
			}
			if !before.Equal(snapshot) {
				t.Error("FR-048: the original character must be unchanged")
			}
			if c.want != 0 {
				r, ok := tr.(character.TransitionRejected[character.SubclassRejected])
				if !ok || r.Rejection().Reason != c.want {
					t.Errorf("result = %#v, want rejection %v [%s]", tr, c.want, c.requirement)
				}
				return
			}
			ok, isOK := tr.(character.TransitionChanged[character.SubclassRejected])
			if !isOK || len(ok.Events()) != 1 {
				t.Fatalf("result = %#v, want accepted [%s]", tr, c.requirement)
			}
			ev := ok.Events()[0].(character.SubclassChosen)
			wantLevel := before.ClassLevels().LevelIn(c.class.ID())
			if ev.CharacterID() != before.ID() || ev.Class() != c.class.ID() || ev.Subclass() != content.SubclassID(c.subclass) || ev.ClassLevel() != wantLevel {
				t.Errorf("event = %#v", ev)
			}
			key := must(character.SubclassStateKey{}.ForClass(c.class.ID()))
			got, found, err := character.ReadFeatureState(ok.Character().FeatureStates(), key)
			if err != nil || !found || got.Subclass() != content.SubclassID(c.subclass) {
				t.Errorf("stored = %v %v %v", got, found, err)
			}
			if ok.Character().FeatureStates().Len() != before.FeatureStates().Len()+1 {
				t.Errorf("states = %d, want one more than %d", ok.Character().FeatureStates().Len(), before.FeatureStates().Len())
			}
			if !ok.Character().ClassLevels().Equal(before.ClassLevels()) {
				t.Error("every other part must be unchanged")
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-107 - Table SP:
// IsPending is true exactly when the class is held, its level is at least the
// subclass level and no subclass is held.
// Path: per case.
func TestSubclassSelectionIsPending(t *testing.T) {
	t.Parallel()
	fighter, wizard := advFighter(t, "champion"), advWizard(t, "evoker")
	for _, c := range []struct {
		name, requirement, path string
		levels                  []character.ClassLevel
		class                   content.ClassDefinition
		held                    string // subclass of the class held, "" for none
		want                    bool
	}{
		{"Fighter 3, none", aiSpec + " FR-107, SRD 5.2.1, Create Your Character, p. 19", "happy", []character.ClassLevel{cl("fighter", 3)}, fighter, "", true},
		{"Fighter 2, none", aiSpec + " FR-107", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 2)}, fighter, "", false},
		{"Fighter 5, champion held", aiSpec + " FR-107", "unhappy (state error)", []character.ClassLevel{cl("fighter", 5)}, fighter, "champion", false},
		{"Fighter 5, wizard not held", aiSpec + " FR-107", "unhappy (state error)", []character.ClassLevel{cl("fighter", 5)}, wizard, "", false},
		{"Fighter 3 / Wizard 1, wizard", aiSpec + " FR-107", "unhappy (rule interaction)", []character.ClassLevel{cl("fighter", 3), cl("wizard", 1)}, wizard, "", false},
		{"Fighter 5, none, above the subclass level", aiSpec + " FR-107 (a pending choice can be made at any later level)", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 5)}, fighter, "", true},
		{"Fighter 3 / Wizard 3, wizard", aiSpec + " FR-107", "happy", []character.ClassLevel{cl("fighter", 3), cl("wizard", 3)}, wizard, "", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch := levelsChar(t, c.levels...)
			if c.held != "" {
				ch = holding(t, ch, c.class.ID().String(), c.held)
			}
			got, err := character.SubclassSelection{}.IsPending(ch, c.class)
			if err != nil || got != c.want {
				t.Errorf("IsPending = %v, %v, want %v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	// A zero character, a zero class and a state of another type are invalid input, never pending.
	cleric := advCleric(t, "life_domain")
	ch := levelsChar(t, cl("cleric", 3))
	bad := must(character.WithFeatureState(ch, character.MustFeatureStateKey[otherState]("subclass_cleric"), otherState{}))
	for name, fn := range map[string]func() (bool, error){
		"zero character":        func() (bool, error) { return character.SubclassSelection{}.IsPending(character.Character{}, cleric) },
		"zero class":            func() (bool, error) { return character.SubclassSelection{}.IsPending(ch, content.ClassDefinition{}) },
		"state of another type": func() (bool, error) { return character.SubclassSelection{}.IsPending(bad, cleric) },
	} {
		if got, err := fn(); got || !isInvalid(err) {
			t.Errorf("IsPending, %s: %v, %v, want false and invalid input", name, got, err)
		}
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-048, FR-107 - the
// subclass lives under "subclass_" plus the class id; a stored state of another type
// under that key is an invalid-input error; a zero character or invalid id is too.
// Path: unhappy (immutability, invalid input).
func TestSubclassStateKeyAndErrors(t *testing.T) {
	t.Parallel()
	k, err := character.SubclassStateKey{}.ForClass("cleric")
	if err != nil || k.Name() != "subclass_cleric" {
		t.Errorf("key = %q, %v", k.Name(), err)
	}
	if _, err := (character.SubclassStateKey{}).ForClass(""); !isInvalid(err) {
		t.Errorf("invalid class: %v", err)
	}
	if _, err := character.NewChosenSubclass(""); !isInvalid(err) {
		t.Errorf("invalid subclass: %v", err)
	}
	a, b := must(character.NewChosenSubclass("life_domain")), must(character.NewChosenSubclass("light_domain"))
	if !a.Equal(a) || a.Equal(b) || a.Equal(otherState{}) {
		t.Error("ChosenSubclass must compare by value and by type")
	}
	cleric := advCleric(t, "life_domain")
	ch := levelsChar(t, cl("cleric", 3))
	bad, err := character.WithFeatureState(ch, character.MustFeatureStateKey[otherState]("subclass_cleric"), otherState{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (character.SubclassSelection{}).Choose(bad, cleric, "life_domain"); !isInvalid(err) {
		t.Errorf("Choose over a state of another type: %v", err)
	}
	if _, err := (character.SubclassSelection{}).IsPending(bad, cleric); !isInvalid(err) {
		t.Errorf("IsPending over a state of another type: %v", err)
	}
	for name, fn := range map[string]func() error{
		"Choose, zero character": func() error {
			_, e := character.SubclassSelection{}.Choose(character.Character{}, cleric, "life_domain")
			return e
		},
		"Choose, invalid subclass": func() error { _, e := character.SubclassSelection{}.Choose(ch, cleric, ""); return e },
		"Choose, zero class": func() error {
			_, e := character.SubclassSelection{}.Choose(ch, content.ClassDefinition{}, "life_domain")
			return e
		},
		"IsPending, zero character": func() error {
			_, e := character.SubclassSelection{}.IsPending(character.Character{}, cleric)
			return e
		},
		"IsPending, zero class": func() error { _, e := character.SubclassSelection{}.IsPending(ch, content.ClassDefinition{}); return e },
	} {
		if err := fn(); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for r, want := range map[character.SubclassRejectionReason]string{
		character.SubclassNotInClass: "not_in_class", character.SubclassBelowSubclassLevel: "below_subclass_level",
		character.SubclassAlreadyChosen: "already_chosen", character.SubclassUnknownSubclass: "unknown_subclass", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("%d = %q, want %q", int(r), r.String(), want)
		}
	}
}

type otherState struct{}

func (otherState) Equal(o character.FeatureState) bool { _, ok := o.(otherState); return ok }
