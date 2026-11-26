package combat

import "github.com/Whealinghold/srd5e-go/character"

// MonsterSpellcastingState is what changes for one monster combatant about its
// spellcasting: the Concentration it holds, the casting of a spell with a casting
// time of 1 minute or more that is not finished, and the SpellUseRef of the list
// entry that began it (SRD 5.2.1, "Spellcasting", "Concentration", pp. 257, 179). It
// holds no spell slot, since a monster has none. The host stores one per monster
// combatant beside its LimitedUseState. It is comparable with ==. The zero value is
// invalid (see IsZero); build one with NewMonsterSpellcastingState. Safe for
// concurrent use.
type MonsterSpellcastingState struct {
	combatant     CombatantID
	concentration character.Concentration
	casting       character.CastingInProgress
	castingRef    SpellUseRef
}

// MonsterSpellcastingStateOption sets an optional member of a
// MonsterSpellcastingState. The options are WithMonsterConcentration and
// WithMonsterCasting.
type MonsterSpellcastingStateOption func(*MonsterSpellcastingState)

// WithMonsterConcentration sets the Concentration the monster holds. The zero
// Concentration means none.
func WithMonsterConcentration(c character.Concentration) MonsterSpellcastingStateOption {
	return func(s *MonsterSpellcastingState) { s.concentration = c }
}

// WithMonsterCasting sets the casting in progress and the SpellUseRef of the list
// entry that began it. The zero CastingInProgress means none.
func WithMonsterCasting(c character.CastingInProgress, ref SpellUseRef) MonsterSpellcastingStateOption {
	return func(s *MonsterSpellcastingState) { s.casting, s.castingRef = c, ref }
}

// NewMonsterSpellcastingState returns the state of combatant with the options
// applied; with none it holds no Concentration and no casting.
//
// It returns an error matching srderr.ErrInvalidInput for a zero combatant and for a
// casting in progress that has no Concentration of kind casting on the same spell. It
// never panics.
func NewMonsterSpellcastingState(combatant CombatantID, opts ...MonsterSpellcastingStateOption) (MonsterSpellcastingState, error) {
	if combatant.IsZero() {
		return MonsterSpellcastingState{}, invalidInput("a monster spellcasting state needs a combatant")
	}
	s := MonsterSpellcastingState{combatant: combatant}
	for _, o := range opts {
		o(&s)
	}
	if !s.casting.IsZero() {
		subject, ok := s.concentration.Subject().(character.ConcentrationSubjectSpell)
		if !ok || s.concentration.Kind() != character.ConcentrationCasting || subject.Spell() != s.casting.Spell() {
			return MonsterSpellcastingState{}, invalidInput("a casting in progress needs Concentration of kind casting on the same spell")
		}
	} else {
		s.castingRef = SpellUseRef{}
	}
	return s, nil
}

// Combatants returns a new slice holding the monster combatant.
func (s MonsterSpellcastingState) Combatants() []CombatantID { return []CombatantID{s.combatant} }

// Concentration returns the Concentration held and whether there is one.
func (s MonsterSpellcastingState) Concentration() (character.Concentration, bool) {
	return s.concentration, !s.concentration.IsZero()
}

// CastingInProgress returns the casting in progress, the SpellUseRef of the entry
// that began it and whether there is one.
func (s MonsterSpellcastingState) CastingInProgress() (character.CastingInProgress, SpellUseRef, bool) {
	return s.casting, s.castingRef, !s.casting.IsZero()
}

// IsZero reports whether s is the invalid zero value.
func (s MonsterSpellcastingState) IsZero() bool { return s.combatant.IsZero() }
