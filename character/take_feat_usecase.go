package character

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// TakeFeatCommand is the input of TakeFeat.Run: the character and its feats, the
// feat taken, the grant it is taken under, the choices made and the class features
// held. Construct it with a keyed literal. Run does not alter it.
type TakeFeatCommand struct {
	Character Character
	Feats     TakenFeats
	Feat      content.FeatID
	Grant     FeatGrant
	Choices   FeatChoices
	Held      HeldFeatures
}

// TakeFeatResult is the closed set of outcomes of TakeFeat.Run. Variants:
// TakeFeatSucceeded, TakeFeatRejected, TakeFeatContentMissing.
type TakeFeatResult interface{ isTakeFeatResult() }

// TakeFeatSucceeded is the outcome of a feat taken. Only Run builds one.
type TakeFeatSucceeded struct{ taking FeatTakingChanged }

// Character returns the new Character, with the ability score increase applied.
func (r TakeFeatSucceeded) Character() Character { return r.taking.character }

// Feats returns the new TakenFeats.
func (r TakeFeatSucceeded) Feats() TakenFeats { return r.taking.feats }

// ProficiencyGrants returns the proficiencies the feat gives, for the host to merge.
func (r TakeFeatSucceeded) ProficiencyGrants() ProficiencyGrants { return r.taking.grants }

// GrantedSpells returns the spells the feat gives; the zero value when it gives none.
func (r TakeFeatSucceeded) GrantedSpells() FeatGrantedSpells { return r.taking.spells }

// Events returns a new slice of the events, in order: FeatTaken, then those of the
// ability score increase.
func (r TakeFeatSucceeded) Events() []Event { return r.taking.Events() }

// TakeFeatRejected is the outcome of a feat a rule refused; the Character and the
// TakenFeats of the command stand. Construct it with a keyed literal.
type TakeFeatRejected struct{ Rejection FeatRejection }

// TakeFeatContentMissing is the outcome when the source lacks the feat, a spell or
// a tool of the choices. Only Run builds one; the zero value names nothing.
type TakeFeatContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found, without repeats, ordered by
// content kind and then by id.
func (r TakeFeatContentMissing) Missing() []content.MissingContent { return slices.Clone(r.missing) }

func (TakeFeatSucceeded) isTakeFeatResult()      {}
func (TakeFeatRejected) isTakeFeatResult()       {}
func (TakeFeatContentMissing) isTakeFeatResult() {}

// TakeFeat is the use case that gives a character a feat: it looks up the feat and
// every spell and tool of the choices, asks FeatTakingRules and emits the
// srd5e.character.* signals. It holds no rules logic. Create it with NewTakeFeat;
// the zero value is not usable. It is safe for concurrent use only if its source,
// telemetry and clock are.
type TakeFeat struct {
	source ContentSource
	sig    operationSignals
}

// NewTakeFeat returns the use case reading content from source. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewTakeFeat(source ContentSource, opts ...Option) (TakeFeat, error) {
	if source == nil {
		return TakeFeat{}, invalidInput("take a feat without a content source")
	}
	return TakeFeat{source: source, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run applies cmd. The feat and every spell and tool the choices name are looked up
// before the result is built.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or signal,
// for the zero use case, a zero Character, an invalid feat id, a zero grant or nil
// choices, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "content_error") with a nil
// result when the source fails, panics or breaks its contract; the failure is
// recorded once and never logged. A refused feat is a TakeFeatRejected result.
func (u TakeFeat) Run(ctx context.Context, cmd TakeFeatCommand) (TakeFeatResult, error) {
	if u.source == nil || cmd.Character.IsZero() || !cmd.Feat.Valid() || cmd.Grant.IsZero() || cmd.Choices == nil {
		return nil, invalidInput("take a feat: zero use case, character or grant, invalid feat id or nil choices")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanTakeFeat, ValueOperationTakeFeat, telemetry.String(AttrFeatSource, featSourceValue(cmd.Grant.source)))
	defer run.end(ctx)

	keys := []content.Key{cmd.Feat.Key()}
	switch v := cmd.Choices.(type) {
	case MagicInitiateChoices:
		keys = append(keys, v.levelOne.Key())
		for _, s := range v.cantrips {
			keys = append(keys, s.Key())
		}
	case SkilledChoices:
		for _, p := range v.picks {
			if tool, ok := p.(SkilledTool); ok {
				keys = append(keys, tool.Tool.Key())
			}
		}
	}
	found, missing, err := lookupFeatEntries(ctx, u.source, u.sig.tel.Tracer, keys)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	var feat content.Feat
	if e, ok := found[cmd.Feat.Key()]; ok {
		if feat, ok = e.(content.Feat); !ok {
			return u.violation(ctx, run, "a feat key answered with another entry type")
		}
		run = run.withAttrs(telemetry.String(AttrFeatCategory, featCategoryValue(feat.FeatCategory())))
	}
	resolved := map[content.SpellID]content.Spell{}
	for k, e := range found {
		if k.Kind() == content.KindSpell {
			s, ok := e.(content.Spell)
			if !ok {
				return u.violation(ctx, run, "a spell key answered with another entry type")
			}
			resolved[s.ID()] = s
		}
	}
	if len(missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return TakeFeatContentMissing{missing: missing}, nil
	}
	taking, err := FeatTakingRules{}.Take(cmd.Character, cmd.Feats, feat, cmd.Grant, cmd.Choices, cmd.Held, resolved)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	switch t := taking.(type) {
	case FeatTakingChanged:
		run.finish(ctx, ValueResultTaken, "")
		return TakeFeatSucceeded{taking: t}, nil
	default:
		rj := t.(FeatTakingRejected).Rejection
		run.finish(ctx, ValueResultRejected, rj.String())
		return TakeFeatRejected{Rejection: rj}, nil
	}
}

func (u TakeFeat) violation(ctx context.Context, run operationRun, detail string) (TakeFeatResult, error) {
	err := portViolation(content.Port, "lookup", detail)
	run.portFailure(ctx, err, srderr.ErrorType(err))
	return nil, err
}

// lookupFeatEntries looks up every distinct key of keys in a child span, and
// returns the entries found by key and the keys not found, without repeats, ordered
// by content kind and then by id. An equipment entry that is not a Tool counts as not found (a Skilled tool pick must be a Tool). A source failure, a panic and a broken contract
// come back as a *srderr.PortError; the span records it.
func lookupFeatEntries(ctx context.Context, source ContentSource, tracer telemetry.Tracer, keys []content.Key) (map[content.Key]content.Entry, []content.MissingContent, error) {
	ctx, span := tracer.Start(ctx, SpanLoadContent)
	defer span.End()
	found := map[content.Key]content.Entry{}
	var missing []content.MissingContent
	seen := map[content.Key]bool{}
	for _, k := range keys {
		if seen[k] {
			continue
		}
		seen[k] = true
		entry, ok, err := lookupEntry(ctx, source, k)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(telemetry.StatusError)
			span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
			return nil, nil, err
		}
		if _, isTool := entry.(content.Tool); ok && (k.Kind() != content.KindEquipment || isTool) {
			found[k] = entry
		} else {
			missing = append(missing, content.NewMissingContent(k.Kind(), k.ID()))
		}
	}
	slices.SortFunc(missing, func(a, b content.MissingContent) int {
		return cmp.Or(cmp.Compare(a.Kind(), b.Kind()), strings.Compare(a.ID().String(), b.ID().String()))
	})
	outcome := telemetry.OutcomeSuccess
	if len(missing) > 0 {
		outcome = telemetry.OutcomeFailure
		span.SetStatus(telemetry.StatusError)
	}
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, outcome))
	return found, missing, nil
}

// featCategoryValue is the ValueFeatCategory* value of c.
func featCategoryValue(c content.FeatCategory) string {
	return featNameOf(int(c), ValueFeatCategoryOrigin, ValueFeatCategoryGeneral, ValueFeatCategoryFightingStyle, ValueFeatCategoryEpicBoon)
}

// featSourceValue is the ValueFeatSource* value of s; a custom background is a background.
func featSourceValue(s FeatSource) string {
	switch s.(type) {
	case BackgroundFeat, CustomBackgroundFeat:
		return ValueFeatSourceBackground
	case SpeciesTraitFeat:
		return ValueFeatSourceSpeciesTrait
	case ClassFeatureFeat:
		return ValueFeatSourceClassFeature
	case EldritchInvocationFeat:
		return ValueFeatSourceEldritchInvocation
	}
	return ValueFeatSourceBonusFeat
}

// portViolation returns the contract violation of a port that answered outside its
// contract, for the use cases of this file's family.
func portViolation(port, op, detail string) error { return srderr.ContractViolation(port, op, detail) }
