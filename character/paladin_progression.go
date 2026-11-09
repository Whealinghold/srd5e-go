package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const paladinChannelColumn content.ClassColumnID = "channel_divinity"

// PaladinFeature names a Paladin feature whose level gates behaviour (class-paladin.md FR-014, FR-125). The zero value is
// not a feature.
type PaladinFeature int

// The gating features: the first eleven are in the Paladin ClassDefinition, the last four in the Oath of Devotion.
const (
	PaladinFeatureFightingStyle PaladinFeature = iota + 1
	PaladinFeatureChannelDivinity
	PaladinFeaturePaladinsSmite
	PaladinFeatureFaithfulSteed
	PaladinFeatureExtraAttack
	PaladinFeatureAuraOfProtection
	PaladinFeatureAbjureFoes
	PaladinFeatureAuraOfCourage
	PaladinFeatureRadiantStrikes
	PaladinFeatureRestoringTouch
	PaladinFeatureAuraExpansion
	PaladinFeatureSacredWeapon
	PaladinFeatureAuraOfDevotion
	PaladinFeatureSmiteOfProtection
	PaladinFeatureHolyNimbus
)

type paladinFeatureInfo struct {
	id     content.FeatureID
	name   string
	inOath bool // the entry is in the Oath of Devotion
}

func paladinFeatureInfos() []paladinFeatureInfo {
	return []paladinFeatureInfo{
		{"paladin_fighting_style", "Fighting Style", false},
		{"paladin_channel_divinity", "Channel Divinity", false},
		{"paladin_paladins_smite", "Paladin’s Smite", false},
		{"paladin_faithful_steed", "Faithful Steed", false},
		{"paladin_extra_attack", "Extra Attack", false},
		{"paladin_aura_of_protection", "Aura of Protection", false},
		{"paladin_abjure_foes", "Abjure Foes", false},
		{"paladin_aura_of_courage", "Aura of Courage", false},
		{"paladin_radiant_strikes", "Radiant Strikes", false},
		{"paladin_restoring_touch", "Restoring Touch", false},
		{"paladin_aura_expansion", "Aura Expansion", false},
		{"oath_of_devotion_sacred_weapon", "Sacred Weapon", true},
		{"oath_of_devotion_aura_of_devotion", "Aura of Devotion", true},
		{"oath_of_devotion_smite_of_protection", "Smite of Protection", true},
		{"oath_of_devotion_holy_nimbus", "Holy Nimbus", true},
	}
}

// Valid reports whether f is one of the gating features.
func (f PaladinFeature) Valid() bool { return f >= 1 && int(f) <= len(paladinFeatureInfos()) }

// String returns the printed feature name ("Paladin’s Smite"), or "unknown".
func (f PaladinFeature) String() string {
	if !f.Valid() {
		return "unknown"
	}
	return paladinFeatureInfos()[f-1].name
}

const oathOfDevotionSpellsID content.FeatureID = "oath_of_devotion_oath_of_devotion_spells"

// PaladinProblemKind is what is wrong with a definition NewPaladinProgression refused. The zero value is not a kind.
type PaladinProblemKind int

const (
	// PaladinWrongClass: the definition is not the Paladin's, or the subclass is not the Oath of Devotion.
	PaladinWrongClass PaladinProblemKind = iota + 1
	// PaladinMissingChannelDivinityColumn: the channel_divinity column is missing, not a count column or a level from 3 shows no count.
	PaladinMissingChannelDivinityColumn
	// PaladinMissingFeatureEntry: a feature entry that gates behaviour (or its always-prepared fact) is missing.
	PaladinMissingFeatureEntry
)

// String returns "wrong_class", "missing_channel_divinity_column", "missing_feature_entry" or "unknown".
func (k PaladinProblemKind) String() string {
	switch k {
	case PaladinWrongClass:
		return "wrong_class"
	case PaladinMissingChannelDivinityColumn:
		return "missing_channel_divinity_column"
	case PaladinMissingFeatureEntry:
		return "missing_feature_entry"
	}
	return "unknown"
}

// PaladinProgressionError reports one problem NewPaladinProgression found. It matches srderr.ErrInvalidInput with errors.Is and
// carries no display text. Feature is the printed name for PaladinMissingFeatureEntry and "" otherwise. Build it with a keyed
// literal.
type PaladinProgressionError struct {
	Kind    PaladinProblemKind
	Feature string
}

// Error renders the error. The text is not a contract.
func (e *PaladinProgressionError) Error() string {
	return "character: invalid Paladin progression: " + e.Kind.String() + " " + e.Feature
}

// Unwrap returns srderr.ErrInvalidInput.
func (*PaladinProgressionError) Unwrap() error { return srderr.ErrInvalidInput }

// Is matches another *PaladinProgressionError of the same Kind and Feature.
func (e *PaladinProgressionError) Is(target error) bool {
	t, ok := target.(*PaladinProgressionError)
	return ok && *t == *e
}

// PaladinPendingChoice is a Paladin-specific choice a level adds, a closed set. The zero value is not a choice.
type PaladinPendingChoice int

// PaladinPendingFightingStyle: the Fighting Style, at the level of its feature entry (2).
const PaladinPendingFightingStyle PaladinPendingChoice = iota + 1

// String returns "fighting_style" or "unknown".
func (c PaladinPendingChoice) String() string {
	if c == PaladinPendingFightingStyle {
		return "fighting_style"
	}
	return "unknown"
}

// PaladinGrant is an always-prepared spell and the Paladin level it is granted at. Build it with a keyed literal.
type PaladinGrant struct {
	Level int
	Spell content.SpellID
}

// PaladinProgression reads the Paladin ClassDefinition (and the Oath of Devotion, when given) once: the Channel Divinity
// column, the level of every gating feature (FR-125), the always-prepared spell grants (FR-058), the feature ids and the feat
// grants of each level. Gates and grants therefore follow the definition; only the SRD formulas are constants. It is immutable
// and holds no port or clock; the zero value is invalid (see IsZero). Safe for concurrent use.
type PaladinProgression struct {
	channel  [MaxLevel]int
	levels   [16]int // indexed by PaladinFeature; 0 for a feature the definition (or the missing Oath) does not give
	features [MaxLevel][]content.FeatureID
	feats    [MaxLevel][]content.FeatCategory
	smite    content.SpellID
	steed    content.SpellID
	class    []PaladinGrant
	oath     []PaladinGrant
	set      bool
}

func grantsOf(e content.FeatureEntry) []PaladinGrant {
	var out []PaladinGrant
	for _, fact := range e.Facts() {
		if a, ok := fact.(content.AlwaysPreparedSpells); ok {
			for _, g := range a.Grants() {
				for _, id := range g.SpellIDs() {
					out = append(out, PaladinGrant{Level: g.Level(), Spell: id})
				}
			}
		}
	}
	return out
}

// NewPaladinProgression reads def and, when oath is not nil, the Oath of Devotion subclass. A nil oath leaves the four Oath of
// Devotion features without a level, so they are never available.
//
// It returns an error joining a *PaladinProgressionError for every problem found, never only the first, matching
// srderr.ErrInvalidInput: PaladinWrongClass for another class (or subclass), PaladinMissingChannelDivinityColumn and
// PaladinMissingFeatureEntry for each gating entry (or always-prepared fact) missing. It never panics.
func NewPaladinProgression(def content.ClassDefinition, oath *content.SubclassDefinition) (PaladinProgression, error) {
	if def.ID() != PaladinClassID {
		return PaladinProgression{}, errors.Join(&PaladinProgressionError{Kind: PaladinWrongClass})
	}
	var problems []error
	add := func(k PaladinProblemKind, feature string) {
		problems = append(problems, &PaladinProgressionError{Kind: k, Feature: feature})
	}
	var p PaladinProgression
	if oath != nil && oath.ID() != OathOfDevotionSubclassID {
		add(PaladinWrongClass, "")
		oath = nil
	}
	if col, err := def.Column(paladinChannelColumn); err != nil || col.Kind() != content.ColumnKindCount {
		add(PaladinMissingChannelDivinityColumn, "")
	} else {
		for level := 1; level <= MaxLevel; level++ {
			row, _ := def.RowAt(level) // a definition holds rows 1 to 20, so the level is in range
			v, _ := row.Value(paladinChannelColumn)
			if n, ok := v.(content.ColumnCount); ok {
				p.channel[level-1] = n.Value()
			} else if level >= 3 {
				add(PaladinMissingChannelDivinityColumn, "")
				break
			}
		}
	}
	byID := map[content.FeatureID]content.FeatureEntry{}
	for _, e := range def.Features() {
		byID[e.ID()] = e
	}
	if oath != nil {
		for _, e := range oath.Features() {
			byID[e.ID()] = e
		}
	}
	for i, info := range paladinFeatureInfos() {
		e, ok := byID[info.id]
		switch {
		case ok:
			p.levels[i+1] = e.Level()
		case !info.inOath || oath != nil:
			add(PaladinMissingFeatureEntry, info.name)
		}
	}
	p.class = slices.Concat(grantsOf(byID["paladin_paladins_smite"]), grantsOf(byID["paladin_faithful_steed"]))
	for _, g := range grantsOf(byID["paladin_paladins_smite"]) {
		p.smite = g.Spell
	}
	for _, g := range grantsOf(byID["paladin_faithful_steed"]) {
		p.steed = g.Spell
	}
	if p.smite == "" {
		add(PaladinMissingFeatureEntry, "Paladin’s Smite")
	}
	if p.steed == "" {
		add(PaladinMissingFeatureEntry, "Faithful Steed")
	}
	if oath != nil {
		spells, ok := byID[oathOfDevotionSpellsID]
		if !ok || len(grantsOf(spells)) == 0 {
			add(PaladinMissingFeatureEntry, "Oath of Devotion Spells")
		}
		p.oath = grantsOf(spells)
	}
	if len(problems) > 0 {
		return PaladinProgression{}, errors.Join(problems...)
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

// MustPaladinProgression is NewPaladinProgression for definitions known to be valid. It panics with the error
// NewPaladinProgression returns, and only then.
func MustPaladinProgression(def content.ClassDefinition, oath *content.SubclassDefinition) PaladinProgression {
	p, err := NewPaladinProgression(def, oath)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero value.
func (p PaladinProgression) IsZero() bool { return !p.set }

// LevelOf returns the level of the feature entry, and 0 for the zero progression, an invalid feature or an Oath of Devotion
// feature when the progression was built without the Oath (class-paladin.md FR-125).
func (p PaladinProgression) LevelOf(f PaladinFeature) int {
	if !p.set || !f.Valid() {
		return 0
	}
	return p.levels[f]
}

// has reports whether a Paladin of level has the feature; the feature is never had when it has no level.
func (p PaladinProgression) has(f PaladinFeature, level PaladinLevel) bool {
	at := p.LevelOf(f)
	return at > 0 && level.n >= at
}

// ChannelDivinityMaximum returns the maximum Channel Divinity uses at level, read from the table: 0 below level 3, 2 at 3 to 10
// and 3 from 11 for the SRD data. It is 0 for the zero progression or level.
func (p PaladinProgression) ChannelDivinityMaximum(level PaladinLevel) int {
	if !p.set || level.IsZero() {
		return 0
	}
	return p.channel[level.n-1]
}

// FeaturesGainedAt returns the ids of the class features the Paladin gains at level, in printed order, as a new slice; empty for
// a level that gains none or for the zero progression or level.
func (p PaladinProgression) FeaturesGainedAt(level PaladinLevel) []content.FeatureID {
	if !p.set || level.IsZero() {
		return []content.FeatureID{}
	}
	return append([]content.FeatureID{}, p.features[level.n-1]...)
}

// ChoicesOnGaining returns the Paladin-specific choices due on gaining the level: the Fighting Style at the level of its entry.
func (p PaladinProgression) ChoicesOnGaining(level PaladinLevel) []PaladinPendingChoice {
	out := []PaladinPendingChoice{}
	if p.set && !level.IsZero() && level.n == p.levels[PaladinFeatureFightingStyle] {
		out = append(out, PaladinPendingFightingStyle)
	}
	return out
}

// FeatEntitlements returns the feat categories the Paladin gains at level: the ClassDefinition feat grants of the level minus
// the Fighting Style grant, which is chosen under the Fighting Style choice (class-paladin.md FR-088). It is a new slice.
func (p PaladinProgression) FeatEntitlements(level PaladinLevel) []content.FeatCategory {
	if !p.set || level.IsZero() {
		return []content.FeatCategory{}
	}
	return append([]content.FeatCategory{}, p.feats[level.n-1]...)
}

// AlwaysPrepared returns the spells a Paladin of level always has prepared: the Divine Smite and Find Steed facts of the class
// entries and, when oath is true and the progression holds the Oath of Devotion, every Oath grant whose level is at or below
// level (class-paladin.md FR-058). Grants are returned unfiltered by the Paladin list (FR-023), in definition order, as a new
// slice.
func (p PaladinProgression) AlwaysPrepared(level PaladinLevel, oath bool) []content.SpellID {
	out := []content.SpellID{}
	if !p.set || level.IsZero() {
		return out
	}
	grants := p.class
	if oath {
		grants = slices.Concat(grants, p.oath)
	}
	for _, g := range grants {
		if g.Level <= level.n {
			out = append(out, g.Spell)
		}
	}
	return out
}

// FreeCastSpell returns the spell the free cast of the feature casts: Divine Smite for Paladin’s Smite, Find Steed for Faithful
// Steed, both read from the definition's always-prepared facts. It is "" for an invalid feature or the zero progression.
func (p PaladinProgression) FreeCastSpell(f FreeCastFeature) content.SpellID {
	switch {
	case !p.set:
		return ""
	case f == FreeCastPaladinsSmite:
		return p.smite
	case f == FreeCastFaithfulSteed:
		return p.steed
	}
	return ""
}
