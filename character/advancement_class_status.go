package character

import "github.com/Whealinghold/srd5e-go/content"

// AdvancementClassStatus is the closed set of what a level in a class is for
// the character (SRD 5.2.1, "Multiclassing", p. 24): a further level in a class
// it holds, or its first level in a new class. Variants:
// AdvancementClassStatusExisting, AdvancementClassStatusNew.
type AdvancementClassStatus interface{ isAdvancementClassStatus() }

// AdvancementClassStatusExisting is a level in a class the character already
// holds. It grants nothing beyond the level's features.
type AdvancementClassStatusExisting struct {
	// CurrentClassLevel is the level in the class before the level gained.
	CurrentClassLevel int
}

// AdvancementClassStatusNew is a first level in a class the character does not
// hold, which grants the class's MulticlassGrants. Only the previewer builds
// one; the zero value grants nothing.
type AdvancementClassStatusNew struct{ grants content.MulticlassGrants }

// Grants returns the proficiencies the first level in the class grants.
func (s AdvancementClassStatusNew) Grants() content.MulticlassGrants { return s.grants }

func (AdvancementClassStatusExisting) isAdvancementClassStatus() {}
func (AdvancementClassStatusNew) isAdvancementClassStatus()      {}

// PendingAdvancementChoice is the closed set of choices a level leaves open that
// every class shares. Variants: PendingSubclassChoice, PendingFeatGrant. A host
// builds the feat grant for the feats rules from a PendingFeatGrant.
type PendingAdvancementChoice interface{ isPendingAdvancementChoice() }

// PendingSubclassChoice is a subclass the character owes in a class. Construct
// it with a keyed literal.
type PendingSubclassChoice struct {
	Class      content.ClassID
	ClassLevel int
}

// PendingFeatGrant is a feat a class feature grants at a class level. For a
// repeated grant (Ability Score Improvement) Feature is the id of the older
// entry and ClassLevel tells the grants apart. Construct it with a keyed
// literal.
type PendingFeatGrant struct {
	Class      content.ClassID
	ClassLevel int
	Feature    content.FeatureID
	Category   content.FeatCategory
}

func (PendingSubclassChoice) isPendingAdvancementChoice() {}
func (PendingFeatGrant) isPendingAdvancementChoice()      {}
