package content

import (
	"maps"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
)

// DeriveMonsterID returns the [MonsterID] of the stat block printed under
// name: the name lower-cased, apostrophes (' and U+2019) removed, each run of
// characters outside ASCII a-z and 0-9 replaced by one underscore, and
// leading and trailing underscores removed ("Saber-Toothed Tiger" is
// "saber_toothed_tiger"). A non-ASCII letter is a separator (SRD 5.2.1,
// "Monsters A-Z", p. 258).
//
// It returns an error wrapping srderr.ErrInvalidInput, with the *[IDError] of
// [ParseID] as its cause, when the name derives to nothing or to a string
// that is not a valid id. It never panics.
func DeriveMonsterID(name string) (MonsterID, error) {
	var b strings.Builder
	pending := false // a separator is waiting for the next kept character
	for _, r := range name {
		switch {
		case r == '\'' || r == '’':
		case r >= 'A' && r <= 'Z':
			r += 'a' - 'A'
			fallthrough
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if pending && b.Len() > 0 {
				b.WriteByte('_')
			}
			pending = false
			b.WriteRune(r)
		default:
			pending = true
		}
	}
	return ParseMonsterID(b.String())
}

// MustDeriveMonsterID is like [DeriveMonsterID] but panics with its error when
// name derives to no valid id.
func MustDeriveMonsterID(name string) MonsterID {
	id, err := DeriveMonsterID(name)
	if err != nil {
		panic(err)
	}
	return id
}

// MonsterFields holds the members of a [Monster] beyond its identity, for
// [NewMonster]. Construct it with a keyed literal.
type MonsterFields struct {
	// Sizes are the size options in printed order, without repeats (at least
	// one; "Medium or Small" is Medium, Small).
	Sizes []CreatureSize
	// Swarm names the size of a swarm's members; the zero value means the
	// monster is not a swarm.
	Swarm SwarmComposition
	// Type is the creature type.
	Type CreatureType
	// Tags are the descriptive tags in printed order, each non-blank.
	Tags []string
	// Alignment is the printed alignment.
	Alignment Alignment
	// ArmorClass is the printed Armor Class, not negative.
	ArmorClass int
	// Initiative is the printed Initiative.
	Initiative StatBlockInitiative
	// HitPoints is the printed Hit Points and Hit Point Dice.
	HitPoints HitPointsEntry
	// Speeds are the printed speeds in order (at least one).
	Speeds []SpeedEntry
	// Abilities are the ability scores and printed saves.
	Abilities AbilityBlock
	// Skills maps a skill to its printed bonus; nil or empty means none.
	Skills map[Skill]int
	// Vulnerabilities, Resistances and Immunities are the defences in printed
	// order. Only Immunities may hold conditions, after its damage types.
	Vulnerabilities, Resistances, Immunities []DefenseEntry
	// Gear is the printed gear item names in order, each non-blank.
	Gear []string
	// Senses are the special senses in printed order.
	Senses []SenseEntry
	// PassivePerception is the printed Passive Perception.
	PassivePerception int
	// Languages is the printed Languages.
	Languages LanguagesEntry
	// ChallengeRating is the printed Challenge Rating.
	ChallengeRating ChallengeRating
	// ExperiencePoints is the printed XP, at least 0. It is not checked
	// against the Challenge Rating.
	ExperiencePoints int
	// ExperiencePointsInLair is the XP printed "in Lair"; 0 means none, and a
	// value must be above ExperiencePoints.
	ExperiencePointsInLair int
	// Traits, Actions, BonusActions and Reactions are the parts of each
	// section in printed order; each may be empty.
	Traits, Actions, BonusActions, Reactions []StatBlockPart
	// LegendaryActions is the Legendary Actions section; the zero value means
	// none.
	LegendaryActions LegendaryActions
}

// Monster is one stat block of the SRD 5.2.1 chapters "Monsters A-Z" and
// "Animals" (pp. 258-364): the header, the combat highlights, the abilities
// and details, and the traits, actions, bonus actions, reactions and
// legendary actions, all as the stat block prints them. It is an [Entry] of
// kind [KindMonster]. Nothing derived is stored: see
// [Monster.ProficiencyBonus] and [Monster.AbilityModifier]. Immutable, with
// its collections copied on construction and on read; the zero Monster is
// invalid. It holds slices, so it is not comparable with ==; compare with
// [Monster.Equal]. Safe for concurrent use.
type Monster struct {
	id     MonsterID
	name   string
	source SourceCitation
	f      MonsterFields
}

var _ Entry = Monster{}

// monsterSourceIssues returns the problems of the citation of a monster: none
// given (missingField at source), a page outside [MonsterMinPage] to
// [MonsterMaxPage] (outOfRange at source/page) and a section other than the
// two of the SRD (invalidValue at source/section).
func monsterSourceIssues(c SourceCitation) []issue {
	var out []issue
	if c.IsZero() {
		return []issue{{"source", ProblemMissingField}}
	}
	if c.page < MonsterMinPage || c.page > MonsterMaxPage {
		out = append(out, issue{idx("source", "page"), ProblemOutOfRange})
	}
	if !monsterSectionOK(c.section) {
		out = append(out, issue{idx("source", "section"), ProblemInvalidValue})
	}
	return out
}

// sizeIssues returns the problems of the size options: none (constraintViolation
// at sizes), an invalid size (invalidValue at it) and a repeat (duplicate at
// the later one).
func sizeIssues(sizes []CreatureSize) []issue {
	if len(sizes) == 0 {
		return []issue{{"sizes", ProblemConstraintViolation}}
	}
	var out []issue
	seen := map[CreatureSize]struct{}{}
	for i, z := range sizes {
		_, dup := seen[z]
		seen[z] = struct{}{}
		switch {
		case !z.Valid():
			out = append(out, issue{idx("sizes", i), ProblemInvalidValue})
		case dup:
			out = append(out, issue{idx("sizes", i), ProblemDuplicate})
		}
	}
	return out
}

// speedIssues returns the problems of the speed list: none (constraintViolation
// at speeds), a zero entry (invalidValue at it) and an entry whose mode set
// and qualifier an earlier entry has (constraintViolation at the later one).
func speedIssues(speeds []SpeedEntry) []issue {
	if len(speeds) == 0 {
		return []issue{{"speeds", ProblemConstraintViolation}}
	}
	var out []issue
	seen := map[speedKey]struct{}{}
	for i, s := range speeds {
		if len(s.modes) == 0 {
			out = append(out, issue{idx("speeds", i), ProblemInvalidValue})
			continue
		}
		k := s.key()
		if _, dup := seen[k]; dup {
			out = append(out, issue{idx("speeds", i), ProblemConstraintViolation})
		}
		seen[k] = struct{}{}
	}
	return out
}

// xpIssues returns the problem of the XP pair: lair XP that is not above the
// printed XP (constraintViolation at experiencePointsInLair).
func xpIssues(xp, lair int, present bool) []issue {
	if present && lair <= xp {
		return []issue{{"experiencePointsInLair", ProblemConstraintViolation}}
	}
	return nil
}

// multiattackIssues returns the problem of a second part named
// [MultiattackName] in actions (constraintViolation at the later part's
// name). A zero part is skipped.
func multiattackIssues(actions []StatBlockPart) []issue {
	var out []issue
	seen := false
	for i, p := range actions {
		if p.name != MultiattackName {
			continue
		}
		if seen {
			out = append(out, issue{idx("actions", i, "name"), ProblemConstraintViolation})
		}
		seen = true
	}
	return out
}

// planIssues returns the plan names that do not resolve (content-monsters
// FR-099): each group, additional-use, oneAttackPart and replacement-option
// name must be exactly one Actions part other than the Multiattack, and each
// requiresUsed exactly one part of Actions, Bonus Actions or Reactions. The
// issue is constraintViolation at the name, in the plan of the Multiattack
// part of actions. A plan on any other part (a trait, bonus action, reaction
// or legendary action, whatever its name) is constraintViolation at its plan
// (FR-036, FR-098).
func planIssues(f MonsterFields) []issue {
	actions, bonusActions, reactions := f.Actions, f.BonusActions, f.Reactions
	var out []issue
	for _, l := range []struct {
		name  string
		parts []StatBlockPart
	}{{"traits", f.Traits}, {"bonusActions", bonusActions}, {"reactions", reactions},
		{idx("legendaryActions", "actions"), f.LegendaryActions.actions}} {
		for i, p := range l.parts {
			if !p.f.Plan.IsZero() {
				out = append(out, issue{idx(l.name, i, "plan"), ProblemConstraintViolation})
			}
		}
	}
	attackParts := map[string]int{}
	used := map[string]int{}
	for _, p := range actions {
		used[p.name]++
		if p.name != MultiattackName {
			attackParts[p.name]++
		}
	}
	for _, list := range [][]StatBlockPart{bonusActions, reactions} {
		for _, p := range list {
			used[p.name]++
		}
	}
	for i, p := range actions {
		if p.name != MultiattackName || p.f.Plan.IsZero() {
			continue
		}
		for _, n := range p.f.Plan.names() {
			held := attackParts
			if n.requires {
				held = used
			}
			if held[n.name] != 1 {
				out = append(out, issue{idx("actions", i, "plan", n.path), ProblemConstraintViolation})
			}
		}
	}
	return out
}

// NewMonster returns the stat block id called name, with source and the
// members of f. id must equal [DeriveMonsterID] of name, source must cite a
// page of [MonsterMinPage] to [MonsterMaxPage] in the section
// [MonsterSection] or [MonsterAnimalsSection], and the members must follow the
// rules of content-monsters section 4.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken,
// in the order of the members of [MonsterFields]: BlockInvalidName for a bad
// name, BlockConstraint for an id that differs from the one derived from the
// name, then BlockMissing, BlockOutOfRange, BlockInvalidMember, BlockDuplicate
// or BlockConstraint for the member as its constructor does. It never
// panics.
func NewMonster(id MonsterID, name string, source SourceCitation, f MonsterFields) (Monster, error) {
	const block = "Monster"
	pid, err := ParseID(string(id))
	if err != nil {
		return Monster{}, err
	}
	if err := checkName(block, name); err != nil {
		return Monster{}, err
	}
	if derived, err := DeriveMonsterID(name); err != nil || derived != id {
		return Monster{}, blockError(block, BlockConstraint)
	}
	m := Monster{id: MonsterID(pid.String()), name: name, source: source, f: cloneMonsterFields(f)}
	if is := m.problems(); len(is) > 0 {
		return Monster{}, is[0].blockError(block)
	}
	return m, nil
}

// cloneMonsterFields copies the collections of f, so the monster shares none
// with its caller.
func cloneMonsterFields(f MonsterFields) MonsterFields {
	f.Sizes = slices.Clone(f.Sizes)
	f.Tags = slices.Clone(f.Tags)
	f.Speeds = slices.Clone(f.Speeds)
	f.Skills = maps.Clone(f.Skills)
	f.Vulnerabilities = slices.Clone(f.Vulnerabilities)
	f.Resistances = slices.Clone(f.Resistances)
	f.Immunities = slices.Clone(f.Immunities)
	f.Gear = slices.Clone(f.Gear)
	f.Senses = slices.Clone(f.Senses)
	f.Traits = slices.Clone(f.Traits)
	f.Actions = slices.Clone(f.Actions)
	f.BonusActions = slices.Clone(f.BonusActions)
	f.Reactions = slices.Clone(f.Reactions)
	return f
}

// problems returns the rules the monster breaks, with paths relative to it,
// in the order of its JSON members. A rule of one member alone and a rule
// between the members of one list both appear here; the decoder reports the
// same rules, member by member, with the same helpers.
func (m Monster) problems() []issue {
	f := m.f
	out := monsterSourceIssues(m.source)
	out = append(out, sizeIssues(f.Sizes)...)
	if !f.Type.Valid() {
		out = append(out, issue{"type", ProblemInvalidValue})
	}
	for i, t := range f.Tags {
		if !validText(t) {
			out = append(out, issue{idx("tags", i), ProblemInvalidValue})
		}
	}
	if !f.Alignment.Valid() {
		out = append(out, issue{"alignment", ProblemInvalidValue})
	}
	if f.ArmorClass < 0 || f.ArmorClass > MonsterMaxNumber {
		out = append(out, issue{"armorClass", ProblemOutOfRange})
	}
	if f.HitPoints.average == 0 {
		out = append(out, issue{"hitPoints", ProblemMissingField})
	}
	out = append(out, speedIssues(f.Speeds)...)
	if f.Abilities.IsZero() {
		out = append(out, issue{"abilities", ProblemMissingField})
	}
	for _, s := range slices.Sorted(maps.Keys(f.Skills)) {
		if !s.Valid() || !monsterNumberOK(f.Skills[s]) {
			out = append(out, issue{idx("skills", s.String()), ProblemInvalidValue})
		}
	}
	out = append(out, nilDefenseIssues("vulnerabilities", f.Vulnerabilities)...)
	out = append(out, nilDefenseIssues("resistances", f.Resistances)...)
	out = append(out, nilDefenseIssues("immunities", f.Immunities)...)
	out = append(out, defenseIssues("vulnerabilities", f.Vulnerabilities, false)...)
	out = append(out, defenseIssues("resistances", f.Resistances, false)...)
	out = append(out, defenseIssues("immunities", f.Immunities, true)...)
	for i, g := range f.Gear {
		if !validText(g) {
			out = append(out, issue{idx("gear", i), ProblemInvalidValue})
		}
	}
	for i, s := range f.Senses {
		if !s.sense.Valid() {
			out = append(out, issue{idx("senses", "special", i), ProblemInvalidValue})
		}
	}
	if !monsterNumberOK(f.PassivePerception) {
		out = append(out, issue{idx("senses", "passivePerception"), ProblemOutOfRange})
	}
	if f.ExperiencePoints < 0 || f.ExperiencePoints > MonsterMaxNumber {
		out = append(out, issue{"experiencePoints", ProblemOutOfRange})
	}
	if f.ExperiencePointsInLair < 0 || f.ExperiencePointsInLair > MonsterMaxNumber {
		out = append(out, issue{"experiencePointsInLair", ProblemOutOfRange})
	}
	out = append(out, xpIssues(f.ExperiencePoints, f.ExperiencePointsInLair, f.ExperiencePointsInLair != 0)...)
	for _, list := range []struct {
		name  string
		parts []StatBlockPart
	}{{"traits", f.Traits}, {"actions", f.Actions}, {"bonusActions", f.BonusActions}, {"reactions", f.Reactions}} {
		for i, p := range list.parts {
			if p.IsZero() {
				out = append(out, issue{idx(list.name, i, "name"), ProblemInvalidValue})
			}
		}
	}
	out = append(out, multiattackIssues(f.Actions)...)
	return append(out, planIssues(f)...)
}

// nilDefenseIssues returns invalidValue at each nil entry of a defence list.
// The decoder's placeholder for an entry with a problem must not leak through
// the constructor.
func nilDefenseIssues(list string, entries []DefenseEntry) []issue {
	var out []issue
	for i, e := range entries {
		if e == nil {
			out = append(out, issue{idx(list, i), ProblemInvalidValue})
		}
	}
	return out
}

// MustMonster is like [NewMonster] but panics with its error when the
// arguments are invalid.
func MustMonster(id MonsterID, name string, source SourceCitation, f MonsterFields) Monster {
	m, err := NewMonster(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return m
}

// Key returns the key of the entry: kind [KindMonster] and its id.
func (m Monster) Key() Key { return m.id.Key() }

// ID returns the monster id.
func (m Monster) ID() MonsterID { return m.id }

// Name returns the name exactly as printed.
func (m Monster) Name() string { return m.name }

// Source returns the citation of the stat block.
func (m Monster) Source() SourceCitation { return m.source }

// Sizes returns the size options in printed order, as a new slice.
func (m Monster) Sizes() []CreatureSize { return slices.Clone(m.f.Sizes) }

// Swarm returns the composition of a swarm, and false for a monster that is
// not a swarm.
func (m Monster) Swarm() (SwarmComposition, bool) { return m.f.Swarm, !m.f.Swarm.IsZero() }

// Type returns the creature type.
func (m Monster) Type() CreatureType { return m.f.Type }

// Tags returns the descriptive tags in printed order, as a new slice.
func (m Monster) Tags() []string { return slices.Clone(m.f.Tags) }

// Alignment returns the printed alignment.
func (m Monster) Alignment() Alignment { return m.f.Alignment }

// ArmorClass returns the printed Armor Class.
func (m Monster) ArmorClass() int { return m.f.ArmorClass }

// Initiative returns the printed Initiative.
func (m Monster) Initiative() StatBlockInitiative { return m.f.Initiative }

// HitPoints returns the printed Hit Points and Hit Point Dice.
func (m Monster) HitPoints() HitPointsEntry { return m.f.HitPoints }

// Speeds returns the speeds in printed order, as a new slice.
func (m Monster) Speeds() []SpeedEntry { return slices.Clone(m.f.Speeds) }

// Abilities returns the ability scores and printed saves.
func (m Monster) Abilities() AbilityBlock { return m.f.Abilities }

// Skills returns the printed skill bonuses, as a new map.
func (m Monster) Skills() map[Skill]int { return maps.Clone(m.f.Skills) }

// Vulnerabilities returns the Vulnerabilities in printed order, as a new
// slice.
func (m Monster) Vulnerabilities() []DefenseEntry { return slices.Clone(m.f.Vulnerabilities) }

// Resistances returns the Resistances in printed order, as a new slice.
func (m Monster) Resistances() []DefenseEntry { return slices.Clone(m.f.Resistances) }

// Immunities returns the Immunities in printed order, as a new slice.
func (m Monster) Immunities() []DefenseEntry { return slices.Clone(m.f.Immunities) }

// Gear returns the gear item names in printed order, as a new slice.
func (m Monster) Gear() []string { return slices.Clone(m.f.Gear) }

// Senses returns the special senses in printed order, as a new slice.
func (m Monster) Senses() []SenseEntry { return slices.Clone(m.f.Senses) }

// PassivePerception returns the printed Passive Perception.
func (m Monster) PassivePerception() int { return m.f.PassivePerception }

// Languages returns the printed Languages.
func (m Monster) Languages() LanguagesEntry { return m.f.Languages }

// ChallengeRating returns the printed Challenge Rating.
func (m Monster) ChallengeRating() ChallengeRating { return m.f.ChallengeRating }

// ExperiencePoints returns the printed XP.
func (m Monster) ExperiencePoints() int { return m.f.ExperiencePoints }

// ExperiencePointsInLair returns the XP printed "in Lair", and false when the
// stat block prints none.
func (m Monster) ExperiencePointsInLair() (int, bool) {
	return m.f.ExperiencePointsInLair, m.f.ExperiencePointsInLair != 0
}

// Traits returns the traits in printed order, as a new slice.
func (m Monster) Traits() []StatBlockPart { return slices.Clone(m.f.Traits) }

// Actions returns the actions in printed order, as a new slice.
func (m Monster) Actions() []StatBlockPart { return slices.Clone(m.f.Actions) }

// BonusActions returns the bonus actions in printed order, as a new slice.
func (m Monster) BonusActions() []StatBlockPart { return slices.Clone(m.f.BonusActions) }

// Reactions returns the reactions in printed order, as a new slice.
func (m Monster) Reactions() []StatBlockPart { return slices.Clone(m.f.Reactions) }

// LegendaryActions returns the Legendary Actions section, and false when the
// monster has none.
func (m Monster) LegendaryActions() (LegendaryActions, bool) {
	return m.f.LegendaryActions, !m.f.LegendaryActions.IsZero()
}

// ProficiencyBonus returns the Proficiency Bonus derived from the Challenge
// Rating, 2 to 9 (SRD 5.2.1, "Proficiency Bonus", p. 256). It is not stored.
func (m Monster) ProficiencyBonus() int { return m.f.ChallengeRating.ProficiencyBonus() }

// AbilityModifier returns the modifier of the score of a, from the Ability
// Modifiers table (SRD 5.2.1, p. 6).
func (m Monster) AbilityModifier(a ability.Ability) ability.Modifier {
	return m.f.Abilities.AbilityModifier(a)
}

// ProficientInSave reports whether the printed save of a differs from its
// ability modifier.
func (m Monster) ProficientInSave(a ability.Ability) bool { return m.f.Abilities.ProficientInSave(a) }

// Multiattack returns the part named [MultiattackName] of the Actions, and
// false when there is none.
func (m Monster) Multiattack() (StatBlockPart, bool) {
	for _, p := range m.f.Actions {
		if p.name == MultiattackName {
			return p, true
		}
	}
	return StatBlockPart{}, false
}

// Spellcasting returns the structured Spellcasting of the first part of the
// Traits, then the Actions, that has one, and false when none has.
func (m Monster) Spellcasting() (MonsterSpellcasting, bool) {
	for _, list := range [][]StatBlockPart{m.f.Traits, m.f.Actions} {
		for _, p := range list {
			if s, ok := p.Spellcasting(); ok {
				return s, true
			}
		}
	}
	return MonsterSpellcasting{}, false
}

// Equal reports whether m and o are the same stat block: every member equal,
// the collections in the same order.
func (m Monster) Equal(o Monster) bool {
	a, b := m.f, o.f
	return m.id == o.id && m.name == o.name && m.source == o.source &&
		slices.Equal(a.Sizes, b.Sizes) && a.Swarm == b.Swarm && a.Type == b.Type && slices.Equal(a.Tags, b.Tags) &&
		a.Alignment == b.Alignment && a.ArmorClass == b.ArmorClass && a.Initiative == b.Initiative &&
		a.HitPoints == b.HitPoints && slices.EqualFunc(a.Speeds, b.Speeds, SpeedEntry.Equal) &&
		a.Abilities == b.Abilities && maps.Equal(a.Skills, b.Skills) &&
		slices.Equal(a.Vulnerabilities, b.Vulnerabilities) && slices.Equal(a.Resistances, b.Resistances) &&
		slices.Equal(a.Immunities, b.Immunities) && slices.Equal(a.Gear, b.Gear) && slices.Equal(a.Senses, b.Senses) &&
		a.PassivePerception == b.PassivePerception && a.Languages.Equal(b.Languages) &&
		a.ChallengeRating == b.ChallengeRating && a.ExperiencePoints == b.ExperiencePoints &&
		a.ExperiencePointsInLair == b.ExperiencePointsInLair &&
		slices.EqualFunc(a.Traits, b.Traits, StatBlockPart.Equal) && slices.EqualFunc(a.Actions, b.Actions, StatBlockPart.Equal) &&
		slices.EqualFunc(a.BonusActions, b.BonusActions, StatBlockPart.Equal) &&
		slices.EqualFunc(a.Reactions, b.Reactions, StatBlockPart.Equal) &&
		a.LegendaryActions.Equal(b.LegendaryActions)
}

// monsterSectionOK reports whether s is one of the two SRD sections a
// monster's source may name.
func monsterSectionOK(s string) bool { return s == MonsterSection || s == MonsterAnimalsSection }
