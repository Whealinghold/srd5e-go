package character

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/content"
)

const eldritchInvocationsFeatureID content.FeatureID = "warlock_eldritch_invocations"

// warlockUse is what every Warlock use case shares: the feature signals and, for the ones that read content, the content source.
type warlockUse struct {
	source ContentSource
	sig    featureSignals
	set    bool
}

func newWarlockUse(what string, source ContentSource, needsSource bool, opts []Option) (warlockUse, error) {
	if needsSource && source == nil {
		return warlockUse{}, invalidInput(what + " without a content source")
	}
	return warlockUse{source: source, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// precheck returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped for a cancelled context.
func (u warlockUse) precheck(ctx context.Context, c Character) error {
	if !u.set || c.IsZero() {
		return invalidInput("Warlock use case: zero use case or zero Character")
	}
	return ctx.Err()
}

// warlockStage says how far a Warlock call got before the rules ran.
type warlockStage int

const (
	// The zero stage means ready: the Warlock state is held and, when asked, the class definition is loaded.
	_ warlockStage = iota
	// warlockNotAWarlock: no Warlock levels, decided before any lookup.
	warlockNotAWarlock
	// warlockMissing: the warlock class definition is not in the source.
	warlockMissing
)

// warlockCall is one Warlock call in flight.
type warlockCall struct {
	run     featureRun
	def     content.ClassDefinition
	profile WarlockProfile
	state   WarlockState
	pact    WarlockPactState
	stage   warlockStage
}

// missing records the content-missing outcome and reports whether the call stopped at it.
func (c warlockCall) missing(ctx context.Context) bool {
	if c.stage != warlockMissing {
		return false
	}
	c.run.contentMissing(ctx)
	return true
}

// notWarlock records the NotAWarlock rejection when the character has no Warlock level and reports it.
func (c warlockCall) notWarlock(ctx context.Context) (WarlockRejection, bool) {
	if c.stage != warlockNotAWarlock {
		return WarlockRejection{}, false
	}
	rej := warlockRefuse(WarlockNotAWarlock)
	c.run.rejected(ctx, rej.Kind.String())
	return rej, true
}

func warlockMissingContent() []content.MissingContent {
	return []content.MissingContent{WarlockClassID.Missing()}
}

// eldritchInvocationOptions returns the options of the Eldritch Invocations feature of the class definition, none when it has none.
func eldritchInvocationOptions(def content.ClassDefinition) []content.FeatureOption {
	for _, f := range def.Features() {
		if f.ID() == eldritchInvocationsFeatureID {
			return f.Options()
		}
	}
	return nil
}

// commit records a WarlockTransition and writes a changed state back into ch. A refusal is recorded as a rejection and returned as
// the rejection (Kind not zero) with ch unchanged; a changed state is recorded as a success and returned as the new Character with the
// events. The error is unreachable (the key is valid and a held state of another type was reported by open) and is recorded on the span.
func (c warlockCall) commit(ctx context.Context, ch Character, t WarlockTransition) (Character, WarlockChanged, WarlockRejection, error) {
	switch v := t.(type) {
	case WarlockRefused:
		c.run.rejected(ctx, v.rejection.Kind.String())
		return ch, WarlockChanged{}, v.rejection, nil
	case WarlockChanged:
		next, err := WithFeatureState(ch, WarlockStateKey(), v.state)
		if err != nil {
			c.run.commandError(err)
			return ch, WarlockChanged{}, WarlockRejection{}, err
		}
		c.run.succeeded(ctx)
		return next, v, WarlockRejection{}, nil
	default:
		panic(fmt.Errorf("unreachable: %T", t))
	}
}

// commitPact is commit for a WarlockPactTransition: a refusal is recorded as a rejection and returned with ch unchanged; a changed pact
// state is written back under WarlockPactStateKey, recorded as a success and returned with the new Character.
func (c warlockCall) commitPact(ctx context.Context, ch Character, t WarlockPactTransition) (Character, WarlockPactChanged, WarlockRejection, error) {
	switch v := t.(type) {
	case WarlockPactRefused:
		c.run.rejected(ctx, v.rejection.Kind.String())
		return ch, WarlockPactChanged{}, v.rejection, nil
	case WarlockPactChanged:
		next, err := WithFeatureState(ch, WarlockPactStateKey(), v.state)
		if err != nil { // unreachable: the key is valid and a held state of another type was reported by open
			c.run.commandError(err)
			return ch, WarlockPactChanged{}, WarlockRejection{}, err
		}
		c.run.succeeded(ctx)
		return next, v, WarlockRejection{}, nil
	default:
		panic(fmt.Errorf("unreachable: %T", t))
	}
}
