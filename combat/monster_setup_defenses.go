package combat

import "github.com/Whealinghold/srd5e-go/content"

// MonsterDefenseKind says which list of a stat block a defence was printed in. The zero
// value is invalid.
type MonsterDefenseKind int

// The three lists, in the order an unapplied defence is reported.
const (
	// DefenseVulnerability: the Vulnerabilities list.
	DefenseVulnerability MonsterDefenseKind = iota + 1
	// DefenseResistance: the Resistances list.
	DefenseResistance
	// DefenseImmunity: the Immunities list.
	DefenseImmunity
)

// String returns "vulnerability", "resistance", "immunity" or "unknown".
func (k MonsterDefenseKind) String() string {
	return enumName([]string{"vulnerability", "resistance", "immunity"}, int(k))
}

// UnappliedDefenseReason says why a printed defence was not applied. The zero value is
// invalid.
type UnappliedDefenseReason int

// The two reasons.
const (
	// DefenseQualified: the entry has a qualifier that makes it conditional on a
	// circumstance the engine does not track.
	DefenseQualified UnappliedDefenseReason = iota + 1
	// DefenseDescribed: the entry is printed text that names neither a damage type
	// nor a condition.
	DefenseDescribed
)

// String returns "qualified", "described" or "unknown".
func (r UnappliedDefenseReason) String() string {
	return enumName([]string{"qualified", "described"}, int(r))
}

// UnappliedDefense is a printed defence a MonsterSetup did not apply, so the host can
// apply it with WithExtraAffinities when the circumstance holds (SRD 5.2.1,
// "Resistances and Vulnerabilities", "Immunities", p. 255). It carries the content
// entry, not text. Comparable with ==; immutable.
type UnappliedDefense struct {
	kind   MonsterDefenseKind
	entry  content.DefenseEntry
	reason UnappliedDefenseReason
}

// Kind returns the list the entry was printed in.
func (d UnappliedDefense) Kind() MonsterDefenseKind { return d.kind }

// Entry returns the printed entry.
func (d UnappliedDefense) Entry() content.DefenseEntry { return d.entry }

// Reason returns why the entry was not applied.
func (d UnappliedDefense) Reason() UnappliedDefenseReason { return d.reason }

// monsterDefenses splits the defences of m into the damage affinities and the condition
// immunities that apply, and the entries that do not, ordered by list (vulnerability,
// resistance, immunity) and then by printed order. An entry with a qualifier or a
// described entry is not applied. Content allows a condition only in Immunities.
func monsterDefenses(m content.Monster) (DamageAffinities, ConditionSet, []UnappliedDefense) {
	var aff DamageAffinities
	var conditions ConditionSet
	var unapplied []UnappliedDefense
	for _, list := range []struct {
		kind    MonsterDefenseKind
		entries []content.DefenseEntry
	}{{DefenseVulnerability, m.Vulnerabilities()}, {DefenseResistance, m.Resistances()}, {DefenseImmunity, m.Immunities()}} {
		for _, entry := range list.entries {
			switch e := entry.(type) {
			case content.DamageTypeDefense:
				if e.Qualifier() != "" {
					unapplied = append(unapplied, UnappliedDefense{list.kind, e, DefenseQualified})
					continue
				}
				b := typeBit(e.DamageType())
				switch list.kind {
				case DefenseVulnerability:
					aff.vuln |= b
				case DefenseResistance:
					aff.resist |= b
				case DefenseImmunity:
					aff.immune |= b
				}
			case content.ConditionDefense:
				if e.Qualifier() != "" {
					unapplied = append(unapplied, UnappliedDefense{list.kind, e, DefenseQualified})
					continue
				}
				conditions.immune |= bit(e.Condition())
			case content.DescribedDefense:
				unapplied = append(unapplied, UnappliedDefense{list.kind, e, DefenseDescribed})
			}
		}
	}
	return aff, conditions, unapplied
}

// unionAffinities returns the affinities holding everything a and b hold.
func unionAffinities(a, b DamageAffinities) DamageAffinities {
	return DamageAffinities{a.resist | b.resist, a.vuln | b.vuln, a.immune | b.immune}
}
