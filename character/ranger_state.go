package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// RangerFightingStyle is the Fighting Style a Ranger chose, a closed set. Variants: RangerFightingStyleFeat, DruidicWarrior.
type RangerFightingStyle interface{ isRangerFightingStyle() }

// RangerFightingStyleFeat is a Fighting Style feat taken as the Fighting Style. Construct it with a keyed literal.
type RangerFightingStyleFeat struct{ Feat content.FeatID }

// DruidicWarrior is the Druidic Warrior option: two Druid cantrips that count as Ranger spells (SRD 5.2.1, "Level 2: Fighting
// Style", p. 59). Construct it with a keyed literal.
type DruidicWarrior struct{ Cantrips [2]content.SpellID }

func (RangerFightingStyleFeat) isRangerFightingStyle() {}
func (DruidicWarrior) isRangerFightingStyle()          {}

// DeftExplorerChoice is the Deft Explorer choice: the skill that gains Expertise and the two languages learned (SRD 5.2.1, "Level
// 2: Deft Explorer", p. 59). Construct it with a keyed literal.
type DeftExplorerChoice struct {
	Skill     content.Skill
	Languages [2]content.Language
}

// RangerActionCost is the action a Ranger feature costs. The zero value is invalid.
type RangerActionCost int

const (
	// RangerMagicAction: Tireless.
	RangerMagicAction RangerActionCost = iota + 1
	// RangerBonusAction: Nature’s Veil.
	RangerBonusAction
	// RangerReaction: Superior Hunter’s Defense.
	RangerReaction
)

// String returns "magic_action", "bonus_action", "reaction" or "unknown".
func (a RangerActionCost) String() string {
	switch a {
	case RangerMagicAction:
		return "magic_action"
	case RangerBonusAction:
		return "bonus_action"
	case RangerReaction:
		return "reaction"
	}
	return "unknown"
}

// RangerState is a Ranger's choices and expended uses: the uses of Favored Enemy, Tireless and Nature’s Veil spent, Deft
// Explorer, Expertise, the Fighting Style, Hunter’s Prey, Defensive Tactics and the feats taken from the feat grants. The
// maximums follow the Ranger level and the Wisdom modifier, so they are not stored and remaining is maximum minus expended,
// never below zero (class-ranger.md FR-035, FR-074). It is a FeatureState stored under RangerStateKey; an absent state is
// InitialRangerState. Immutable, comparable with ==. Safe for concurrent use.
type RangerState struct {
	favoredSpent, tirelessSpent, veilSpent int
	deft                                   DeftExplorerChoice
	deftChosen                             bool
	expertise                              [2]content.Skill
	expertiseChosen                        bool
	style                                  RangerFightingStyle
	prey, tactics                          string
	generalFeats, epicBoons                int
}

// RangerStateKey returns the FeatureStateKey named ranger_state.
func RangerStateKey() FeatureStateKey[RangerState] {
	return MustFeatureStateKey[RangerState]("ranger_state")
}

// RangerStateParts are the fields of a RangerState, for RestoreRangerState. Construct it with a keyed literal.
type RangerStateParts struct {
	FavoredEnemySpent int
	TirelessSpent     int
	NaturesVeilSpent  int
	DeftExplorer      *DeftExplorerChoice // nil for not chosen
	Expertise         *[2]content.Skill   // nil for not chosen
	FightingStyle     RangerFightingStyle
	HuntersPrey       string // "" for not chosen
	DefensiveTactics  string // "" for not chosen
	GeneralFeats      int
	EpicBoons         int
}

// InitialRangerState returns the state of a Ranger with nothing chosen and no use expended.
func InitialRangerState() RangerState { return RangerState{} }

// RestoreRangerState returns the state of parts. It returns an error matching srderr.ErrInvalidInput for a negative count, a
// Deft Explorer choice with an invalid skill or language or two equal languages, Expertise of an invalid skill or twice the
// same skill, a Fighting Style that is not one of the closed set, with an invalid feat id or invalid or equal cantrips. It never
// panics.
func RestoreRangerState(parts RangerStateParts) (RangerState, error) {
	if parts.FavoredEnemySpent < 0 || parts.TirelessSpent < 0 || parts.NaturesVeilSpent < 0 || parts.GeneralFeats < 0 || parts.EpicBoons < 0 {
		return RangerState{}, invalidInput("Ranger state with a negative count")
	}
	s := RangerState{favoredSpent: parts.FavoredEnemySpent, tirelessSpent: parts.TirelessSpent, veilSpent: parts.NaturesVeilSpent,
		prey: parts.HuntersPrey, tactics: parts.DefensiveTactics, generalFeats: parts.GeneralFeats, epicBoons: parts.EpicBoons}
	if d := parts.DeftExplorer; d != nil {
		if !d.Skill.Valid() || !d.Languages[0].Valid() || !d.Languages[1].Valid() || d.Languages[0] == d.Languages[1] {
			return RangerState{}, invalidInput("Ranger Deft Explorer choice")
		}
		s.deft, s.deftChosen = *d, true
	}
	if e := parts.Expertise; e != nil {
		if !e[0].Valid() || !e[1].Valid() || e[0] == e[1] {
			return RangerState{}, invalidInput("Ranger Expertise choice")
		}
		s.expertise, s.expertiseChosen = *e, true
	}
	switch st := parts.FightingStyle.(type) {
	case nil:
	case RangerFightingStyleFeat:
		if !st.Feat.Valid() {
			return RangerState{}, invalidInput("Ranger Fighting Style feat id")
		}
		s.style = st
	case DruidicWarrior:
		if !st.Cantrips[0].Valid() || !st.Cantrips[1].Valid() || st.Cantrips[0] == st.Cantrips[1] {
			return RangerState{}, invalidInput("Druidic Warrior cantrips")
		}
		s.style = st
	default:
		return RangerState{}, invalidInput("Ranger Fighting Style is not one of the closed set")
	}
	return s, nil
}

// FavoredEnemySpent returns the expended uses of Favored Enemy.
func (s RangerState) FavoredEnemySpent() int { return s.favoredSpent }

// TirelessSpent returns the expended uses of Tireless.
func (s RangerState) TirelessSpent() int { return s.tirelessSpent }

// NaturesVeilSpent returns the expended uses of Nature’s Veil.
func (s RangerState) NaturesVeilSpent() int { return s.veilSpent }

// DeftExplorer returns the Deft Explorer choice and false when not chosen.
func (s RangerState) DeftExplorer() (DeftExplorerChoice, bool) { return s.deft, s.deftChosen }

// Expertise returns the two Expertise skills of the level 9 feature and false when not chosen, as a new slice.
func (s RangerState) Expertise() ([]content.Skill, bool) {
	if !s.expertiseChosen {
		return []content.Skill{}, false
	}
	return []content.Skill{s.expertise[0], s.expertise[1]}, true
}

// FightingStyle returns the chosen Fighting Style and false when none is chosen.
func (s RangerState) FightingStyle() (RangerFightingStyle, bool) { return s.style, s.style != nil }

// HuntersPrey returns the held Hunter’s Prey option id and false when none is held.
func (s RangerState) HuntersPrey() (string, bool) { return s.prey, s.prey != "" }

// DefensiveTactics returns the held Defensive Tactics option id and false when none is held.
func (s RangerState) DefensiveTactics() (string, bool) { return s.tactics, s.tactics != "" }

// FeatsTaken returns how many General and how many Epic Boon feats were taken from the feat grants.
func (s RangerState) FeatsTaken() (general, epicBoon int) { return s.generalFeats, s.epicBoons }

// DruidicWarriorCantrips returns the Druidic Warrior cantrips as GrantedCantrips for spellcasting: each uses the Ranger’s ability
// and the Druid list, and does not count against the prepared limit (class-ranger.md FR-052). It is empty without Druidic Warrior.
func (s RangerState) DruidicWarriorCantrips() []GrantedCantrip {
	dw, ok := s.style.(DruidicWarrior)
	if !ok {
		return []GrantedCantrip{}
	}
	out := make([]GrantedCantrip, 0, 2)
	for _, id := range dw.Cantrips {
		g, err := NewGrantedCantrip(id, CasterRanger, "druid")
		if err == nil { // the state holds valid ids
			out = append(out, g)
		}
	}
	return out
}

// Equal reports whether other is a RangerState with the same fields.
func (s RangerState) Equal(other FeatureState) bool {
	o, ok := other.(RangerState)
	return ok && o == s
}

// TirelessUses returns the Tireless and Nature’s Veil uses at a Wisdom modifier: the modifier, minimum one (SRD 5.2.1, "Level
// 10: Tireless", p. 59).
func TirelessUses(wisdom ability.Modifier) int { return max(wisdom.Value(), 1) }

// FavoredEnemyRemaining returns the free casts of Hunter’s Mark left at level: the Favored Enemy column minus the expended uses.
func (s RangerState) FavoredEnemyRemaining(p RangerProgression, level RangerLevel) int {
	return remaining(p.FavoredEnemyUses(level), s.favoredSpent)
}

// TirelessRemaining returns the Tireless uses left at the Wisdom modifier.
func (s RangerState) TirelessRemaining(wisdom ability.Modifier) int {
	return remaining(TirelessUses(wisdom), s.tirelessSpent)
}

// NaturesVeilRemaining returns the Nature’s Veil uses left at the Wisdom modifier.
func (s RangerState) NaturesVeilRemaining(wisdom ability.Modifier) int {
	return remaining(TirelessUses(wisdom), s.veilSpent)
}

// readRangerState returns the RangerState c stores, or InitialRangerState when none is stored. It returns an error matching
// srderr.ErrInvalidInput when another type of state sits under the key.
func readRangerState(c Character) (RangerState, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), RangerStateKey())
	if err != nil {
		return RangerState{}, err
	}
	if !ok {
		return InitialRangerState(), nil
	}
	return s, nil
}
