package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const rangerFavoredEnemyColumn content.ClassColumnID = "favored_enemy"

// RangerFeature names a Ranger feature whose level gates behaviour (class-ranger.md FR-046, FR-096). The zero value is not a
// feature.
type RangerFeature int

// The gating features: the first fourteen and the last are in the Ranger ClassDefinition, the five between in the Hunter subclass.
const (
	RangerFeatureFavoredEnemy RangerFeature = iota + 1
	RangerFeatureWeaponMastery
	RangerFeatureDeftExplorer
	RangerFeatureFightingStyle
	RangerFeatureExtraAttack
	RangerFeatureRoving
	RangerFeatureExpertise
	RangerFeatureTireless
	RangerFeatureRelentlessHunter
	RangerFeatureNaturesVeil
	RangerFeaturePreciseHunter
	RangerFeatureFeralSenses
	RangerFeatureFoeSlayer
	RangerFeatureEpicBoon
	RangerFeatureHuntersLore
	RangerFeatureHuntersPrey
	RangerFeatureDefensiveTactics
	RangerFeatureSuperiorHuntersPrey
	RangerFeatureSuperiorHuntersDefense
	RangerFeatureSubclass
)

type rangerFeatureInfo struct {
	id       content.FeatureID
	name     string
	inHunter bool // the entry is in the Hunter subclass
}

func rangerFeatureInfos() []rangerFeatureInfo {
	return []rangerFeatureInfo{
		{"ranger_favored_enemy", "Favored Enemy", false},
		{"ranger_weapon_mastery", "Weapon Mastery", false},
		{"ranger_deft_explorer", "Deft Explorer", false},
		{"ranger_fighting_style", "Fighting Style", false},
		{"ranger_extra_attack", "Extra Attack", false},
		{"ranger_roving", "Roving", false},
		{"ranger_expertise", "Expertise", false},
		{"ranger_tireless", "Tireless", false},
		{"ranger_relentless_hunter", "Relentless Hunter", false},
		{"ranger_natures_veil", "Nature’s Veil", false},
		{"ranger_precise_hunter", "Precise Hunter", false},
		{"ranger_feral_senses", "Feral Senses", false},
		{"ranger_foe_slayer", "Foe Slayer", false},
		{"ranger_epic_boon", "Epic Boon", false},
		{"hunter_hunters_lore", "Hunter’s Lore", true},
		{"hunter_hunters_prey", "Hunter’s Prey", true},
		{"hunter_defensive_tactics", "Defensive Tactics", true},
		{"hunter_superior_hunters_prey", "Superior Hunter’s Prey", true},
		{"hunter_superior_hunters_defense", "Superior Hunter’s Defense", true},
		{"ranger_ranger_subclass", "Ranger Subclass", false},
	}
}

// Valid reports whether f is one of the gating features.
func (f RangerFeature) Valid() bool { return f >= 1 && int(f) <= len(rangerFeatureInfos()) }

// String returns the printed feature name ("Nature’s Veil"), or "unknown".
func (f RangerFeature) String() string {
	if !f.Valid() {
		return "unknown"
	}
	return rangerFeatureInfos()[f-1].name
}

// RangerProblemKind is what is wrong with a definition NewRangerProgression refused. The zero value is not a kind.
type RangerProblemKind int

const (
	// RangerWrongClass: the definition is not the Ranger's, or the subclass is not the Hunter.
	RangerWrongClass RangerProblemKind = iota + 1
	// RangerMissingFavoredEnemyColumn: the favored_enemy column is missing, not a count column or a level shows no count.
	RangerMissingFavoredEnemyColumn
	// RangerMissingFeatureEntry: a feature entry that gates behaviour (or its fact or options) is missing.
	RangerMissingFeatureEntry
)

// String returns "wrong_class", "missing_favored_enemy_column", "missing_feature_entry" or "unknown".
func (k RangerProblemKind) String() string {
	switch k {
	case RangerWrongClass:
		return "wrong_class"
	case RangerMissingFavoredEnemyColumn:
		return "missing_favored_enemy_column"
	case RangerMissingFeatureEntry:
		return "missing_feature_entry"
	}
	return "unknown"
}

// RangerProgressionError reports one problem NewRangerProgression found. It matches srderr.ErrInvalidInput with errors.Is and
// carries no display text. Feature is the printed name for RangerMissingFeatureEntry and "" otherwise. Build it with a keyed
// literal.
type RangerProgressionError struct {
	Kind    RangerProblemKind
	Feature string
}

// Error renders the error. The text is not a contract.
func (e *RangerProgressionError) Error() string {
	return "character: invalid Ranger progression: " + e.Kind.String() + " " + e.Feature
}

// Unwrap returns srderr.ErrInvalidInput.
func (*RangerProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *RangerProgressionError of the same Kind and Feature.
func (e *RangerProgressionError) Is(target error) bool {
	t, ok := target.(*RangerProgressionError)
	return ok && *t == *e
}

// RangerPendingChoice is a Ranger choice a level owes and the character has not made, a closed set. The zero value is not a
// choice.
type RangerPendingChoice int

// The pending choices (class-ranger.md FR-097).
const (
	RangerPendingWeaponMastery RangerPendingChoice = iota + 1
	RangerPendingDeftExplorer
	RangerPendingFightingStyle
	RangerPendingSubclass
	RangerPendingHuntersPrey
	RangerPendingAbilityScoreImprovement
	RangerPendingDefensiveTactics
	RangerPendingExpertise
	RangerPendingEpicBoon
)

// String returns the snake_case name or "unknown".
func (c RangerPendingChoice) String() string {
	names := []string{"weapon_mastery", "deft_explorer", "fighting_style", "ranger_subclass", "hunters_prey", "ability_score_improvement",
		"defensive_tactics", "expertise", "epic_boon"}
	if c < 1 || int(c) > len(names) {
		return "unknown"
	}
	return names[c-1]
}

// RangerProgression reads the Ranger ClassDefinition (and the Hunter, when given) once: the Favored Enemy column, the level of
// every gating feature, the always-prepared Hunter’s Mark grant, the Hunter’s Prey and Defensive Tactics option ids and the feat
// grants of each level. Gates and grants therefore follow the definition; only the SRD formulas are constants. It is immutable
// and holds no port or clock; the zero value is invalid (see IsZero). Safe for concurrent use.
type RangerProgression struct {
	favored  [MaxLevel]int
	levels   [21]int // indexed by RangerFeature; 0 for a feature the definition (or the missing Hunter) does not give
	features [MaxLevel][]content.FeatureID
	feats    [MaxLevel][]content.FeatCategory
	mark     []PaladinGrant
	prey     []string
	tactics  []string
	hunter   bool
	set      bool
}

func optionIDs(e content.FeatureEntry) []string {
	var out []string
	for _, o := range e.Options() {
		out = append(out, string(o.ID()))
	}
	return out
}

// NewRangerProgression reads def and, when hunter is not nil, the Hunter subclass. A nil hunter leaves the five Hunter features
// without a level, so they are never available.
//
// It returns an error joining a *RangerProgressionError for every problem found, never only the first, matching
// srderr.ErrInvalidInput: RangerWrongClass for another class (or subclass), RangerMissingFavoredEnemyColumn and
// RangerMissingFeatureEntry for each gating entry (or its always-prepared fact, or the two options of a Hunter feature) missing.
// It never panics.
func NewRangerProgression(def content.ClassDefinition, hunter *content.SubclassDefinition) (RangerProgression, error) {
	if def.ID() != RangerClassID {
		return RangerProgression{}, errors.Join(&RangerProgressionError{Kind: RangerWrongClass})
	}
	var problems []error
	add := func(k RangerProblemKind, feature string) {
		problems = append(problems, &RangerProgressionError{Kind: k, Feature: feature})
	}
	var p RangerProgression
	if hunter != nil && hunter.ID() != HunterSubclassID {
		add(RangerWrongClass, "")
		hunter = nil
	}
	if col, err := def.Column(rangerFavoredEnemyColumn); err != nil || col.Kind() != content.ColumnKindCount {
		add(RangerMissingFavoredEnemyColumn, "")
	} else {
		for level := 1; level <= MaxLevel; level++ {
			row, _ := def.RowAt(level) // a definition holds rows 1 to 20, so the level is in range
			v, _ := row.Value(rangerFavoredEnemyColumn)
			n, ok := v.(content.ColumnCount)
			if !ok {
				add(RangerMissingFavoredEnemyColumn, "")
				break
			}
			p.favored[level-1] = n.Value()
		}
	}
	byID := map[content.FeatureID]content.FeatureEntry{}
	for _, e := range def.Features() {
		byID[e.ID()] = e
	}
	if hunter != nil {
		for _, e := range hunter.Features() {
			byID[e.ID()] = e
		}
	}
	for i, info := range rangerFeatureInfos() {
		e, ok := byID[info.id]
		switch {
		case ok:
			p.levels[i+1] = e.Level()
		case !info.inHunter || hunter != nil:
			add(RangerMissingFeatureEntry, info.name)
		}
	}
	p.mark = grantsOf(byID["ranger_favored_enemy"])
	if len(p.mark) == 0 {
		add(RangerMissingFeatureEntry, "Favored Enemy")
	}
	if hunter != nil {
		p.hunter = true
		if p.prey = optionIDs(byID["hunter_hunters_prey"]); len(p.prey) != 2 {
			add(RangerMissingFeatureEntry, "Hunter’s Prey")
		}
		if p.tactics = optionIDs(byID["hunter_defensive_tactics"]); len(p.tactics) != 2 {
			add(RangerMissingFeatureEntry, "Defensive Tactics")
		}
	}
	if len(problems) > 0 {
		return RangerProgression{}, errors.Join(problems...)
	}
	for level := 1; level <= MaxLevel; level++ {
		gained, _ := def.EntriesGainedAt(level)
		for _, e := range gained {
			p.features[level-1] = append(p.features[level-1], e.ID())
		}
		grants, _ := def.FeatGrantsAt(level)
		p.feats[level-1] = slices.DeleteFunc(grants, func(c content.FeatCategory) bool { return c == content.FeatFightingStyle })
	}
	p.set = true
	return p, nil
}

// MustRangerProgression is NewRangerProgression for definitions known to be valid. It panics with the error
// NewRangerProgression returns, and only then.
func MustRangerProgression(def content.ClassDefinition, hunter *content.SubclassDefinition) RangerProgression {
	p, err := NewRangerProgression(def, hunter)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p RangerProgression) IsZero() bool { return !p.set }

// LevelOf returns the level of the feature entry, and 0 for the zero progression, an invalid feature or a Hunter feature when
// the progression was built without the Hunter (class-ranger.md FR-046).
func (p RangerProgression) LevelOf(f RangerFeature) int {
	if !p.set || !f.Valid() {
		return 0
	}
	return p.levels[f]
}

// has reports whether a Ranger of level has the feature; the feature is never had when it has no level.
func (p RangerProgression) has(f RangerFeature, level RangerLevel) bool {
	at := p.LevelOf(f)
	return at > 0 && level.n >= at
}

// FavoredEnemyUses returns the Favored Enemy column at level: the free casts of Hunter’s Mark (class-ranger.md FR-023, FR-032).
// It is 0 for the zero progression or level.
func (p RangerProgression) FavoredEnemyUses(level RangerLevel) int {
	if !p.set || level.IsZero() {
		return 0
	}
	return p.favored[level.n-1]
}

// FeaturesGainedAt returns the ids of the class features the Ranger gains at level, in printed order, as a new slice.
func (p RangerProgression) FeaturesGainedAt(level RangerLevel) []content.FeatureID {
	if !p.set || level.IsZero() {
		return []content.FeatureID{}
	}
	return append([]content.FeatureID{}, p.features[level.n-1]...)
}

// FeatGrantsUpTo returns how many General (Ability Score Improvement) and how many Epic Boon feats the Ranger has been granted
// from level 1 to level (class-ranger.md FR-064, T6). The Fighting Style grant is not counted: it is chosen under its own choice.
func (p RangerProgression) FeatGrantsUpTo(level RangerLevel) (general, epicBoon int) {
	if !p.set || level.IsZero() {
		return 0, 0
	}
	for l := 1; l <= level.n; l++ {
		for _, c := range p.feats[l-1] {
			switch c {
			case content.FeatGeneral:
				general++
			case content.FeatEpicBoon:
				epicBoon++
			case content.FeatOrigin, content.FeatFightingStyle:
				// not counted: the Fighting Style grant is filtered out and Origin feats are never class grants
			}
		}
	}
	return general, epicBoon
}

// AlwaysPrepared returns the spells a Ranger of level always has prepared: the Hunter’s Mark fact of Favored Enemy (class-ranger.md
// FR-031), as a new slice. Grants are returned unfiltered by the Ranger list, in definition order.
func (p RangerProgression) AlwaysPrepared(level RangerLevel) []content.SpellID {
	out := []content.SpellID{}
	if !p.set || level.IsZero() {
		return out
	}
	for _, g := range p.mark {
		if g.Level <= level.n {
			out = append(out, g.Spell)
		}
	}
	return out
}

// HuntersMark returns the spell Favored Enemy always prepares and casts free, read from the definition's fact; "" for the zero
// progression.
func (p RangerProgression) HuntersMark() content.SpellID {
	if !p.set {
		return ""
	}
	return p.mark[0].Spell
}

// HuntersPreyOptions returns the option ids of Hunter’s Prey, as a new slice; empty without the Hunter.
func (p RangerProgression) HuntersPreyOptions() []string { return append([]string{}, p.prey...) }

// DefensiveTacticsOptions returns the option ids of Defensive Tactics, as a new slice; empty without the Hunter.
func (p RangerProgression) DefensiveTacticsOptions() []string {
	return append([]string{}, p.tactics...)
}
