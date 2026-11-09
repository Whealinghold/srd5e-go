package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-sorcerer.md FR-048, FR-128 - RestoreSorcererState keeps Sorcery Points within 0 to the maximum of the level, Innate Sorcery uses within 0 to 2, created slots non-negative, the
// Metamagic effects valid and distinct, and the Elemental Affinity type within the five; anything else is an error matching srderr.ErrInvalidInput.
// Path: per case (invalid input / boundary).
func TestRestoreSorcererState(t *testing.T) {
	t.Parallel()
	p := sorDraconic(6)
	for _, c := range []struct {
		name  string
		parts character.SorcererStateParts
		ok    bool
	}{
		{"the empty state", character.SorcererStateParts{}, true},
		{"points at the maximum", character.SorcererStateParts{SorceryPoints: 6}, true},
		{"points above the maximum", character.SorcererStateParts{SorceryPoints: 7}, false},
		{"negative points", character.SorcererStateParts{SorceryPoints: -1}, false},
		{"two uses", character.SorcererStateParts{InnateSorceryUses: 2}, true},
		{"three uses", character.SorcererStateParts{InnateSorceryUses: 3}, false},
		{"negative uses", character.SorcererStateParts{InnateSorceryUses: -1}, false},
		{"a negative created slot", character.SorcererStateParts{CreatedSlots: [5]int{0, -1, 0, 0, 0}}, false},
		{"created slots", character.SorcererStateParts{CreatedSlots: [5]int{1, 2, 0, 0, 1}}, true},
		{"a repeated effect", character.SorcererStateParts{Metamagic: []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicCareful}}, false},
		{"an invalid effect", character.SorcererStateParts{Metamagic: []character.MetamagicEffect{0}}, false},
		{"an effect above the ten", character.SorcererStateParts{Metamagic: []character.MetamagicEffect{11}}, false},
		{"two effects", character.SorcererStateParts{Metamagic: []character.MetamagicEffect{character.MetamagicTwinned, character.MetamagicCareful}}, true},
		{"Fire affinity", character.SorcererStateParts{ElementalAffinity: content.DamageFire}, true},
		{"Thunder affinity", character.SorcererStateParts{ElementalAffinity: content.DamageThunder}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := character.RestoreSorcererState(p, c.parts)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("RestoreSorcererState = %v, want ok %v", err, c.ok)
			}
		})
	}
	if _, err := character.RestoreSorcererState(character.SorcererProfile{}, character.SorcererStateParts{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero profile err = %v, want ErrInvalidInput", err)
	}
	if _, err := character.RestoreSorcererState(sorProfile(1, ""), character.SorcererStateParts{SorceryPoints: 1}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("1 point at level 1 err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-100, FR-128 - the state compares by value, its Parts round trip, and the accessors report every field; the collections it returns are copies.
// Path: happy / unhappy (immutability).
func TestSorcererStateAccessorsAndImmutability(t *testing.T) {
	t.Parallel()
	p := sorDraconic(14)
	parts := character.SorcererStateParts{
		Metamagic:         []character.MetamagicEffect{character.MetamagicTwinned, character.MetamagicCareful},
		SorceryPoints:     9,
		InnateSorceryUses: 1, InnateSorceryActive: true, RestorationUsed: true,
		CreatedSlots:      [5]int{1, 0, 2, 0, 0},
		ElementalAffinity: content.DamageCold, DragonWingsUsed: true, DragonWingsActive: true, DragonCompanionFreeCastUsed: true,
		FreeMetamagicSpent: true, QuickenedUsedThisTurn: true,
	}
	s := sorState(p, parts)
	got := s.Parts()
	parts.Metamagic = []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}
	if !reflect.DeepEqual(got, parts) {
		t.Errorf("Parts = %+v, want %+v", got, parts)
	}
	if !s.Knows(character.MetamagicCareful) || !s.Knows(character.MetamagicTwinned) || s.Knows(character.MetamagicDistant) || s.Knows(0) {
		t.Errorf("Knows reports wrong: %v", s.Metamagic())
	}
	if s.SorceryPoints() != 9 || s.InnateSorceryUses() != 1 || !s.InnateSorceryActive() || !s.RestorationUsed() || !s.DragonWingsUsed() || !s.DragonWingsActive() ||
		!s.DragonCompanionFreeCastUsed() || !s.FreeMetamagicSpent() || !s.QuickenedUsedThisTurn() {
		t.Errorf("accessors disagree with %+v", got)
	}
	if t0, ok := s.ElementalAffinity(); !ok || t0 != content.DamageCold {
		t.Errorf("ElementalAffinity = (%v, %v)", t0, ok)
	}
	if _, ok := sorFresh(p).ElementalAffinity(); ok {
		t.Error("a fresh state has no Elemental Affinity")
	}
	for level, want := range map[int]int{0: 0, 1: 1, 2: 0, 3: 2, 5: 0, 6: 0, -1: 0} {
		if n := s.CreatedSlots(level); n != want {
			t.Errorf("CreatedSlots(%d) = %d, want %d", level, n, want)
		}
	}
	list := s.Metamagic()
	list[0] = character.MetamagicTwinned
	if again := s.Metamagic(); again[0] != character.MetamagicCareful {
		t.Errorf("changing a returned slice altered the state: %v", again)
	}
	same := sorState(p, got)
	other := sorState(p, character.SorcererStateParts{})
	if !s.Equal(same) || s.Equal(other) || s.Equal(character.BardicInspirationPool{}) || s != same {
		t.Error("Equal must hold for equal states only")
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-128 - the state is stored under the typed key named sorcerer and read back through it; the aggregate does not interpret it.
// Path: happy.
func TestSorcererStateKey(t *testing.T) {
	t.Parallel()
	if got := character.SorcererStateKey().Name(); got != "sorcerer" {
		t.Errorf("key = %q, want sorcerer", got)
	}
	s := sorFresh(sorProfile(5, ""))
	c := sorChar(t, 5, 16, "", s)
	if got := sorStateOf(t, c); got != s {
		t.Errorf("stored state = %+v, want %+v", got, s)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-010, FR-129 - SorcererCreationChoices seeds a level 1 state with 0 Sorcery Points, 2 Innate Sorcery uses, no Metamagic, no created slots and no once-per-rest use, adds nothing else, leaves
// its argument unchanged, and names no content; states that already hold the key are returned unchanged.
// Path: happy / unhappy (state).
func TestSorcererCreationChoicesSeed(t *testing.T) {
	t.Parallel()
	var choices character.ClassCreationChoices = character.SorcererCreationChoices{}
	empty := character.FeatureStates{}
	seeded := choices.SeedFeatureStates(empty)
	if empty.Len() != 0 || seeded.Len() != 1 {
		t.Fatalf("lengths = %d and %d, want 0 and 1", empty.Len(), seeded.Len())
	}
	s, ok, err := character.ReadFeatureState(seeded, character.SorcererStateKey())
	if err != nil || !ok || s != (sorFresh(sorProfile(1, ""))) {
		t.Errorf("seeded state = (%+v, %v, %v), want the level 1 state", s, ok, err)
	}
	if again := choices.SeedFeatureStates(seeded); !again.Equal(seeded) {
		t.Error("seeding twice changed the states")
	}
	if keys := choices.ContentKeys(); keys == nil || len(keys) != 0 {
		t.Errorf("ContentKeys = %v, want a non-nil empty slice", keys)
	}
	wrong := must(character.SetFeatureState(empty, character.MustFeatureStateKey[character.BardicInspirationPool]("sorcerer"), character.BardicInspirationPool{}))
	if got := choices.SeedFeatureStates(wrong); !got.Equal(wrong) {
		t.Error("a state of another type under the key must be left alone")
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-090 (SRD 5.2.1, "Level 6: Elemental Affinity", p. 70) - the Elemental Affinity types are Acid, Cold, Fire, Lightning and Poison, as a new slice each time.
// Path: happy.
func TestElementalAffinityTypes(t *testing.T) {
	t.Parallel()
	want := []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageLightning, content.DamagePoison}
	got := character.ElementalAffinityTypes()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ElementalAffinityTypes = %v, want %v", got, want)
	}
	got[0] = content.DamageForce
	if again := character.ElementalAffinityTypes(); again[0] != content.DamageAcid {
		t.Error("the returned slice is shared")
	}
}

// Requirement: doc/specs/class-sorcerer.md section 3, Table OP of content-classes-data.md (SRD 5.2.1, "Metamagic Options", pp. 66-67) - the ten effects map one to one to the option ids careful_spell to twinned_spell in the printed
// alphabetical order, and anything else is no effect.
// Path: per case (happy / unhappy: out of range, unknown id).
func TestMetamagicEffectIDs(t *testing.T) {
	t.Parallel()
	ids := []string{"careful_spell", "distant_spell", "empowered_spell", "extended_spell", "heightened_spell", "quickened_spell", "seeking_spell", "subtle_spell", "transmuted_spell", "twinned_spell"}
	for i, id := range ids {
		e := character.MetamagicEffect(i + 1)
		got, ok := character.MetamagicEffectOf(content.FeatureOptionID(id))
		if !e.Valid() || e.String() != id || string(e.OptionID()) != id || !ok || got != e {
			t.Errorf("effect %d = %q %q valid %v, EffectOf(%q) = (%v, %v)", i+1, e.String(), e.OptionID(), e.Valid(), id, got, ok)
		}
	}
	for _, e := range []character.MetamagicEffect{0, 11, -1} {
		if e.Valid() || e.String() != "unknown" || e.OptionID() != "" {
			t.Errorf("effect %d = %q %q valid %v, want invalid", int(e), e.String(), e.OptionID(), e.Valid())
		}
	}
	if e, ok := character.MetamagicEffectOf("fireball"); ok || e != 0 {
		t.Errorf("EffectOf(fireball) = (%v, %v), want none", e, ok)
	}
}
