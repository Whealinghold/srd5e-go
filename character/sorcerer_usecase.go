package character

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	sorcererMetamagicFeatureID content.FeatureID = "sorcerer_metamagic"
	sorcererFontFeatureID      content.FeatureID = "sorcerer_font_of_magic"
	creatingSpellSlotsTable                      = "Creating Spell Slots"
)

// sorcererUse is what every Sorcerer use case shares: the content source and the feature signals.
type sorcererUse struct {
	source ContentSource
	sig    featureSignals
	set    bool
}

func newSorcererUse(what string, source ContentSource, opts []Option) (sorcererUse, error) {
	if source == nil {
		return sorcererUse{}, invalidInput(what + " without a content source")
	}
	return sorcererUse{source: source, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// precheck returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped for a cancelled context.
func (u sorcererUse) precheck(ctx context.Context, c Character) error {
	if !u.set || c.IsZero() {
		return invalidInput("Sorcerer use case: zero use case or zero Character")
	}
	return ctx.Err()
}

// sorcererStage says how far a Sorcerer call got before the rules ran.
type sorcererStage int

const (
	// sorcererReady: the Sorcerer state is held and the class definition is loaded.
	sorcererReady sorcererStage = iota
	// sorcererNotASorcerer: no Sorcerer levels, or no Sorcerer state; decided before any lookup.
	sorcererNotASorcerer
	// sorcererMissing: the sorcerer class definition is not in the source.
	sorcererMissing
)

// sorcererCall is one Sorcerer call in flight.
type sorcererCall struct {
	run      featureRun
	def      content.ClassDefinition
	profile  SorcererProfile
	state    SorcererState
	hasState bool
	isLevels bool
	stage    sorcererStage
}

// stopped records the content-missing outcome and reports whether the call stopped at it.
func (c sorcererCall) missing(ctx context.Context) bool {
	if c.stage != sorcererMissing {
		return false
	}
	c.run.contentMissing(ctx)
	return true
}

// notSorcerer records the NotASorcerer rejection when the call has no Sorcerer state (or no Sorcerer level) and reports it.
func (c sorcererCall) notSorcerer(ctx context.Context) (SorcererRejection, bool) {
	if c.stage != sorcererNotASorcerer && (c.hasState || !c.isLevels) {
		return SorcererRejection{}, false
	}
	rej := refuse(SorcererNotASorcerer)
	c.run.rejected(ctx, rej.Kind.String())
	return rej, true
}

func sorcererMissingContent() []content.MissingContent {
	return []content.MissingContent{SorcererClassID.Missing()}
}

// metamagicOptions returns the options of the Metamagic feature of the class definition, none when it has none.
func metamagicOptions(def content.ClassDefinition) []content.FeatureOption {
	for _, f := range def.Features() {
		if f.ID() == sorcererMetamagicFeatureID {
			return f.Options()
		}
	}
	return nil
}

// creatingSpellSlots returns the Creating Spell Slots table of the Font of Magic feature and false when the definition has none.
func creatingSpellSlots(def content.ClassDefinition) (content.FeatureTable, bool) {
	for _, f := range def.Features() {
		if f.ID() != sorcererFontFeatureID {
			continue
		}
		for _, t := range f.Tables() {
			if t.Name() == creatingSpellSlotsTable {
				return t, true
			}
		}
	}
	return content.FeatureTable{}, false
}

// writeBack stores s under SorcererStateKey in c. The key is valid and a held state of another type was reported by open, so
// the error is unreachable and returned for the caller to record.
func writeBack(c Character, s SorcererState) (Character, error) {
	return WithFeatureState(c, SorcererStateKey(), s)
}

// commit records a SorcererTransition and writes a changed state back into ch. A refusal is recorded as a rejection and
// returned as the rejection (Kind not zero) with ch unchanged; a changed state is recorded as a success and returned as the new
// Character with the events. The error is unreachable (the key is valid and a held state of another type was reported by open)
// and is recorded on the span.
func (c sorcererCall) commit(ctx context.Context, ch Character, t SorcererTransition) (Character, []Event, SorcererRejection, error) {
	switch v := t.(type) {
	case SorcererRefused:
		c.run.rejected(ctx, v.rejection.Kind.String())
		return ch, nil, v.rejection, nil
	case SorcererChanged:
		next, err := writeBack(ch, v.state)
		if err != nil {
			c.run.commandError(err)
			return ch, nil, SorcererRejection{}, err
		}
		c.run.succeeded(ctx)
		return next, v.events, SorcererRejection{}, nil
	default:
		panic(fmt.Errorf("unreachable: %T", t))
	}
}
