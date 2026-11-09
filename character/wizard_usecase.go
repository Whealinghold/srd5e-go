package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// wizardUse is what every Wizard use case shares: the content source (nil for the
// pure ones) and the feature signals.
type wizardUse struct {
	source ContentSource
	sig    featureSignals
	set    bool
}

func newWizardUse(source ContentSource, opts []Option) wizardUse {
	return wizardUse{source: source, sig: newFeatureSignals(newOptions(opts)), set: true}
}

// wizardCommandProblem is the validation every Wizard use case runs first: it returns an
// error matching srderr.ErrInvalidInput for the zero use case, and for a zero
// CharacterID where the use case produces an event.
func wizardCommandProblem(set bool, id Identifier, needsID bool) error {
	if !set || (needsID && id.IsZero()) {
		return invalidInput("Wizard use case: zero use case or character id")
	}
	return nil
}

// wizardStage says how far a Wizard call got before the rules ran.
type wizardStage int

const (
	// wizardReady: the Wizard levels are held and the class definition, if wanted, is loaded.
	wizardReady wizardStage = iota
	// wizardNotAWizard: no Wizard levels; decided before any lookup.
	wizardNotAWizard
	// wizardMissing: the wizard class definition is not in the source.
	wizardMissing
)

// wizardCall is one Wizard call in flight.
type wizardCall struct {
	run   featureRun
	def   content.ClassDefinition
	level int
	stage wizardStage
}

// stopped records the NotAWizard or content-missing outcome and reports whether the call
// stopped at one.
func (c wizardCall) stopped(ctx context.Context) bool {
	switch c.stage {
	case wizardNotAWizard:
		c.run.rejected(ctx, ValueRejectionNotAWizard)
		return true
	case wizardMissing:
		c.run.contentMissing(ctx)
		return true
	case wizardReady:
	}
	return false
}

// wizardMissingContent is the one entry a content-missing Wizard result names.
func wizardMissingContent() []content.MissingContent {
	return []content.MissingContent{WizardClassID.Missing()}
}
