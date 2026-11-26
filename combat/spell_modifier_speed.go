package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// QuerySpeeds returns the speeds of creature under the active spells in sources: Longstrider adds 10 feet to every
// speed, then Haste doubles every speed, then Slow halves every speed, each rounded down; none falls below 0 and a
// speed the creature lacks stays absent (spell-granted-modifiers.md FR-050 to FR-056). The order is the package's
// reading, since the SRD fixes none (FR-053). A source on another creature, and one for a spell that is not
// modelled, is skipped. The input is not altered.
//
// It returns an error matching ErrDuplicateSource (and srderr.ErrInvalidInput) for two sources with one spell and
// target, and one matching srderr.ErrInvalidInput for a zero source or a zero creature. It never panics.
func QuerySpeeds(sources []SpellModifierSource, creature CombatantID, speeds MovementSpeeds) (MovementSpeeds, error) {
	if err := checkSources(sources); err != nil {
		return MovementSpeeds{}, err
	}
	if creature.IsZero() {
		return MovementSpeeds{}, invalidInput("speed query without a creature")
	}
	has := func(spell content.SpellID) bool {
		return slices.ContainsFunc(sources, func(s SpellModifierSource) bool { return s.spell == spell && s.TargetIs(creature) })
	}
	if has(spellLongstrider) {
		speeds = speeds.AdjustedBy(10)
	}
	if has(spellHaste) {
		speeds = speeds.Doubled()
	}
	if has(spellSlow) {
		speeds = speeds.Halved()
	}
	return speeds, nil
}
