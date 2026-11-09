package character

import (
	"context"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// CalculateDerivedStatsCommand is the input of CalculateDerivedStats.Run: plain
// values, never a character. Construct it with a keyed literal. ClassLevels,
// AbilityScores and HitPointMaximum are required; everything else may be left
// at its zero value, which means none (ArmorID and ShieldID: nothing worn;
// ChosenBasis: the highest applicable basis; ArmorClassFloor: no floor).
// HitPointMaximum is the stored maximum without any bonus in
// HitPointMaximumBonuses. Run does not alter the command.
type CalculateDerivedStatsCommand struct {
	ClassLevels             ClassLevels
	AbilityScores           ability.Scores
	HitPointMaximum         HitPointMaximum
	HitDicePool             HitDicePool
	SkillProficiencies      SkillProficiencies
	ArmorID                 content.ItemID
	ShieldID                content.ItemID
	ChosenBasis             ArmorClassBasisChoice
	AlternativeBases        []AbilityBasedArmorClassBasis
	MulticlassArmorTraining ArmorTraining
	FlatArmorClassBonuses   []FlatBonus
	ArmorClassFloor         int
	FlatSavingThrowBonuses  []FlatSavingThrowBonus
	AdditionalSavingThrows  SavingThrowProficiencies
	HitPointMaximumBonuses  []HitPointMaximumBonus
	PerceptionRollMode      modifier.RollMode
}

// CalculateDerivedStatsResult is the closed set of outcomes of
// CalculateDerivedStats.Run. Variants: CalculateDerivedStatsSucceeded,
// CalculateDerivedStatsContentMissing,
// CalculateDerivedStatsArmorClassBasisNotApplicable,
// CalculateDerivedStatsEquipmentSlotMismatch.
type CalculateDerivedStatsResult interface{ isCalculateDerivedStatsResult() }

// CalculateDerivedStatsSucceeded is the outcome for a calculation that worked.
type CalculateDerivedStatsSucceeded struct{ Stats DerivedStats }

// CalculateDerivedStatsContentMissing is the outcome when the initial class, the
// armor or the Shield is unknown to the content source. It names at least one
// missing entry. Only Run builds one; the zero value names nothing.
type CalculateDerivedStatsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the unknown entries, in lookup order (initial
// class, armor, Shield) with duplicates removed.
func (r CalculateDerivedStatsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// CalculateDerivedStatsArmorClassBasisNotApplicable is the outcome when the
// chosen Armor Class basis cannot be used.
type CalculateDerivedStatsArmorClassBasisNotApplicable struct {
	Choice ArmorClassBasisChoice
	Reason ArmorClassBasisRejection
}

// CalculateDerivedStatsEquipmentSlotMismatch is the outcome when an id supplied
// as body armor or as a Shield resolved to an entry that does not fit: Key is
// the offending entry and Expected the slot the id was supplied for.
type CalculateDerivedStatsEquipmentSlotMismatch struct {
	Key      content.Key
	Expected ArmorSlot
}

func (CalculateDerivedStatsSucceeded) isCalculateDerivedStatsResult()                    {}
func (CalculateDerivedStatsContentMissing) isCalculateDerivedStatsResult()               {}
func (CalculateDerivedStatsArmorClassBasisNotApplicable) isCalculateDerivedStatsResult() {}
func (CalculateDerivedStatsEquipmentSlotMismatch) isCalculateDerivedStatsResult()        {}

// CalculateDerivedStats is the use case that computes the derived stats of a
// character from plain values: it looks up the initial class, the armor and the
// Shield through the content source, hands everything to DerivedStatsCalculator
// and emits the srd5e.character.derived_stats.* signals. It holds no rules logic
// and no state beyond its injected ports. Create it with
// NewCalculateDerivedStats; the zero value is not usable. It is safe for
// concurrent use only if its source, telemetry and clock are.
type CalculateDerivedStats struct {
	source   ContentSource
	tel      telemetry.Telemetry
	clock    func() time.Time
	calls    telemetry.Counter
	errors   telemetry.Counter
	duration telemetry.Histogram
	levels   telemetry.Histogram
}

// NewCalculateDerivedStats returns the use case, with its telemetry instruments
// created once. Options: WithSource (required), WithTelemetry, WithClock. It
// returns an error matching srderr.ErrInvalidInput when no source is set.
func NewCalculateDerivedStats(opts ...Option) (CalculateDerivedStats, error) {
	o := newOptions(opts)
	if o.source == nil {
		return CalculateDerivedStats{}, invalidInput("derived stats without a content source")
	}
	m := o.tel.Meter
	return CalculateDerivedStats{
		source:   o.source,
		tel:      o.tel,
		clock:    o.clock,
		calls:    m.Counter(telemetry.Descriptor{Name: MetricDerivedStatsCalls, Unit: "{call}", Description: "Derived stats calculations, by outcome."}),
		errors:   m.Counter(telemetry.Descriptor{Name: MetricDerivedStatsErrors, Unit: "{error}", Description: "Content source failures leaving the derived stats use case."}),
		duration: m.Histogram(telemetry.Descriptor{Name: MetricDerivedStatsDuration, Unit: "s", Description: "Duration of one derived stats calculation."}),
		levels:   m.Histogram(telemetry.Descriptor{Name: MetricDerivedStatsLevels, Unit: "{level}", Description: "Total level of successfully calculated characters."}),
	}, nil
}

// Run calculates the derived stats of cmd. A failed lookup of content, a slot
// mismatch and an inapplicable chosen basis are results, in that precedence
// (FR-068); only a source failure is an error.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case or an invalid command (see
// DerivedStatsCalculator.Calculate; also two alternative bases with one id, a
// floor below 0, an invalid armor or Shield id). It returns ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (srderr.ErrorType "content_error") with a nil result when
// the source fails, panics (matching srderr.ErrPortPanic) or breaks its
// contract (matching srderr.ErrContractViolation); the failure is wrapped once,
// recorded once on both spans, the calls and errors counters, and never logged.
func (u CalculateDerivedStats) Run(ctx context.Context, cmd CalculateDerivedStatsCommand) (CalculateDerivedStatsResult, error) {
	if u.source == nil {
		return nil, invalidInput("zero CalculateDerivedStats; use NewCalculateDerivedStats")
	}
	in := cmd.input()
	if err := in.validate(); err != nil {
		return nil, err
	}
	if (cmd.ArmorID != "" && !cmd.ArmorID.Valid()) || (cmd.ShieldID != "" && !cmd.ShieldID.Valid()) {
		return nil, invalidInput("invalid armor or Shield id")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	multiclass := telemetry.Bool(AttrMulticlass, cmd.ClassLevels.Len() > 1)
	ctx, span := u.tel.Tracer.Start(ctx, SpanDerivedStatsCalculate, multiclass)
	defer span.End()
	timer := telemetry.StartTimer(u.clock, u.duration)
	defer timer.Stop(ctx)

	loaded, err := u.load(ctx, cmd)
	if err != nil {
		span.RecordError(err)
		u.finish(ctx, span, telemetry.OutcomeFailure, multiclass)
		u.errors.Add(ctx, 1, telemetry.String(telemetry.AttrErrorType, srderr.ErrorType(err)))
		return nil, err
	}
	if len(loaded.missing) > 0 {
		u.finish(ctx, span, telemetry.OutcomeFailure, multiclass, telemetry.String(AttrDerivedStatsFailure, ValueDerivedStatsContentMissing))
		return CalculateDerivedStatsContentMissing{missing: loaded.missing}, nil
	}
	armor, mismatch := slotted(loaded.armor, BodyArmorSlot)
	if mismatch == nil {
		in.Armor = armor
		in.Shield, mismatch = slotted(loaded.shield, ShieldSlot)
	}
	if mismatch != nil {
		u.finish(ctx, span, telemetry.OutcomeFailure, multiclass, telemetry.String(AttrDerivedStatsFailure, ValueDerivedStatsEquipmentSlotMismatch))
		return *mismatch, nil
	}

	in.InitialClass = loaded.class
	calc, err := DerivedStatsCalculator{}.Calculate(in)
	if err != nil { // the command was validated above, so this is unreachable in practice
		span.RecordError(err)
		span.SetStatus(telemetry.StatusError)
		return nil, err
	}
	switch c := calc.(type) {
	case DerivedStatsComputed:
		u.finish(ctx, span, telemetry.OutcomeSuccess, multiclass)
		u.levels.Record(ctx, float64(c.Stats.totalLevel.n))
		return CalculateDerivedStatsSucceeded(c), nil
	case DerivedStatsBasisNotApplicable:
		u.finish(ctx, span, telemetry.OutcomeFailure, multiclass, telemetry.String(AttrDerivedStatsFailure, ValueDerivedStatsBasisNotApplicable))
		return CalculateDerivedStatsArmorClassBasisNotApplicable(c), nil
	case DerivedStatsSlotMismatch:
		u.finish(ctx, span, telemetry.OutcomeFailure, multiclass, telemetry.String(AttrDerivedStatsFailure, ValueDerivedStatsEquipmentSlotMismatch))
		return CalculateDerivedStatsEquipmentSlotMismatch(c), nil
	default:
		panic(invalidInput("unknown calculation")) // unreachable: DerivedStatsCalculation is closed
	}
}

// input copies the content-free fields of cmd into a calculator input.
func (cmd CalculateDerivedStatsCommand) input() DerivedStatsInput {
	return DerivedStatsInput{
		ClassLevels:             cmd.ClassLevels,
		AbilityScores:           cmd.AbilityScores,
		HitPointMaximum:         cmd.HitPointMaximum,
		HitDicePool:             cmd.HitDicePool,
		SkillProficiencies:      cmd.SkillProficiencies,
		ChosenBasis:             cmd.ChosenBasis,
		AlternativeBases:        cmd.AlternativeBases,
		MulticlassArmorTraining: cmd.MulticlassArmorTraining,
		FlatArmorClassBonuses:   cmd.FlatArmorClassBonuses,
		ArmorClassFloor:         cmd.ArmorClassFloor,
		FlatSavingThrowBonuses:  cmd.FlatSavingThrowBonuses,
		AdditionalSavingThrows:  cmd.AdditionalSavingThrows,
		HitPointMaximumBonuses:  cmd.HitPointMaximumBonuses,
		PerceptionRollMode:      cmd.PerceptionRollMode,
	}
}

// finish sets the outcome and, for a failure, the error status on span, and
// counts the call once.
func (u CalculateDerivedStats) finish(ctx context.Context, span telemetry.Span, outcome string, multiclass telemetry.Attr, extra ...telemetry.Attr) {
	o := telemetry.String(telemetry.AttrOutcome, outcome)
	if outcome == telemetry.OutcomeFailure {
		span.SetStatus(telemetry.StatusError)
	}
	span.SetAttributes(o)
	u.calls.Add(ctx, 1, append([]telemetry.Attr{o, multiclass}, extra...)...)
}

// slotted checks that the entry resolved for slot is an Armor fitting it. An
// absent entry (nil) gives the zero Armor.
func slotted(entry content.Entry, slot ArmorSlot) (content.Armor, *CalculateDerivedStatsEquipmentSlotMismatch) {
	if entry == nil {
		return content.Armor{}, nil
	}
	if a, ok := entry.(content.Armor); ok && armorFitsSlot(a, slot) {
		return a, nil
	}
	return content.Armor{}, &CalculateDerivedStatsEquipmentSlotMismatch{Key: entry.Key(), Expected: slot}
}

// loadedContent is what the lookups of one call found.
type loadedContent struct {
	class         content.ClassDefinition
	armor, shield content.Entry // nil when not requested
	missing       []content.MissingContent
}

// load looks up the initial class, the armor and the Shield, all of them before
// reporting what is missing, inside the load_content span.
func (u CalculateDerivedStats) load(ctx context.Context, cmd CalculateDerivedStatsCommand) (loadedContent, error) {
	ctx, span := u.tel.Tracer.Start(ctx, SpanDerivedStatsLoadContent)
	defer span.End()
	var out loadedContent
	fail := func(err error) (loadedContent, error) {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusError)
		span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
		return loadedContent{}, err
	}
	find := func(key content.Key) (content.Entry, error) {
		entry, found, err := u.lookup(ctx, key)
		if err == nil && !found {
			if m := content.NewMissingContent(key.Kind(), key.ID()); !slices.Contains(out.missing, m) {
				out.missing = append(out.missing, m)
			}
		}
		return entry, err
	}
	entry, err := find(cmd.ClassLevels.Initial().Key())
	if err != nil {
		return fail(err)
	}
	if entry != nil {
		def, ok := entry.(content.ClassDefinition)
		if !ok {
			return fail(srderr.ContractViolation(content.Port, "lookup", "a class key answered with another entry type"))
		}
		out.class = def
	}
	if cmd.ArmorID != "" {
		if out.armor, err = find(cmd.ArmorID.Key()); err != nil {
			return fail(err)
		}
	}
	if cmd.ShieldID != "" {
		if out.shield, err = find(cmd.ShieldID.Key()); err != nil {
			return fail(err)
		}
	}
	outcome := telemetry.OutcomeSuccess
	if len(out.missing) > 0 {
		outcome = telemetry.OutcomeFailure
		span.SetStatus(telemetry.StatusError)
	}
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, outcome))
	return out, nil
}

// lookup asks the source for key and returns the entry and true, or false when
// it is not found. A source failure, a panic and a broken contract come back as a
// *srderr.PortError.
func (u CalculateDerivedStats) lookup(ctx context.Context, key content.Key) (content.Entry, bool, error) {
	return lookupEntry(ctx, u.source, key)
}
