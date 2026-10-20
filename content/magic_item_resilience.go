package content

// MagicItemResilience is how a magic item stands up to damage (SRD 5.2.1,
// "Magic Item Resilience", p. 206), a closed set of three members. It has no
// behaviour: how damage is handled is the combat context's. The zero
// MagicItemResilience is invalid. Comparable and safe for concurrent use.
type MagicItemResilience int

// The magic item resiliences.
const (
	ResilienceResistantToAllDamage MagicItemResilience = iota + 1
	ResilienceNonmagicalDurability
	ResilienceImpervious
)

func magicItemResilienceNames() []string {
	return []string{"resistantToAllDamage", "nonmagicalDurability", "impervious"}
}

// Valid reports whether r is one of the three resiliences.
func (r MagicItemResilience) Valid() bool { return enumValid(magicItemResilienceNames(), int(r)) }

// String returns the name of r, or "unknown" for an undefined value.
func (r MagicItemResilience) String() string { return enumString(magicItemResilienceNames(), int(r)) }

// MagicItemResilienceFacts holds the default resilience rule. It holds no
// state; its zero value is the rule. Safe for concurrent use.
type MagicItemResilienceFacts struct{}

// DefaultFor returns the default resilience of an item of category c and
// rarity r: [ResilienceImpervious] for the Artifact in every category, else
// [ResilienceNonmagicalDurability] for potions and scrolls, else
// [ResilienceResistantToAllDamage]. A per-item exception is an override in its
// [MagicItemFacts]. It returns 0, which is not valid, when c or r is not a
// member. It never panics.
func (MagicItemResilienceFacts) DefaultFor(c MagicItemCategory, r ItemRarity) MagicItemResilience {
	switch {
	case !c.Valid() || !r.Valid():
		return 0 // equivalent mutant: replaces the literal 0 with 0
	case r == RarityArtifact: // equivalent mutant: the Artifact is the highest member, so == and >= agree once r is valid
		return ResilienceImpervious
	case c == MagicItemPotion || c == MagicItemScroll:
		return ResilienceNonmagicalDurability
	}
	return ResilienceResistantToAllDamage
}
