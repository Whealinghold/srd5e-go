package combat

import (
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// CombatantKind says which character or monster a combatant is. It carries
// identity only; no rule reads it. Variants: PlayerCharacterCombatant,
// MonsterCombatant.
type CombatantKind interface{ combatantKind() }

// PlayerCharacterCombatant is a combatant that is a Player Character. The zero
// value holds the invalid zero character id. Construct with a keyed literal.
type PlayerCharacterCombatant struct{ ID character.Identifier }

// MonsterCombatant is a combatant that is a Monster. The zero value holds the
// empty monster id. Construct with a keyed literal.
type MonsterCombatant struct{ ID content.MonsterID }

func (PlayerCharacterCombatant) combatantKind() {}
func (MonsterCombatant) combatantKind()         {}
