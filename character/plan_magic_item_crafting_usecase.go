package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// MagicItemCraftingSource is the content lookup PlanMagicItemCrafting needs:
// ContentSource plus ListKind for the tool entries and the Spell Scroll Costs
// rows. Any content.Source satisfies it, as does contenttest.FakeSource.
type MagicItemCraftingSource interface {
	ContentSource
	ListKind(ctx context.Context, kind content.Kind) ([]content.Entry, error)
}

// PlanMagicItemCraftingCommand is the input of PlanMagicItemCrafting.Do. Facts is
// the host-supplied MagicItemFacts of the item; RarityChoice picks a rarity for
// an item with none of its own; BaseItem is the Armor, Weapon, AmmunitionVariant
// or wondrous base item the host chose (nil for none); Incorporated says how an
// incorporated item with a cost is handled and is the zero value when the facts
// name none. Construct it with a keyed literal. Do does not alter it.
type PlanMagicItemCraftingCommand struct {
	Facts          content.MagicItemFacts
	RarityChoice   RarityChoice
	BaseItem       content.EquipmentEntry
	Team           CraftingTeam
	Qualifications []CrafterQualification
	Incorporated   IncorporatedItemChoice
	Allowance      AssistantAllowance
}

// MagicItemCraftingIncompleteReason says which content was missing from a
// source that answered. The zero value is not a reason.
type MagicItemCraftingIncompleteReason int

// The reasons.
const (
	CraftingToolSetMissing MagicItemCraftingIncompleteReason = iota + 1
	CraftingScribingTableMissing
)

// PlanMagicItemCraftingResult is the closed set of outcomes of
// PlanMagicItemCrafting.Do. Variants: PlanMagicItemCraftingPlanned,
// PlanMagicItemCraftingRejected, PlanMagicItemCraftingRarityUnresolved,
// PlanMagicItemCraftingContentMissing, PlanMagicItemCraftingContentIncomplete.
type PlanMagicItemCraftingResult interface{ isPlanMagicItemCraftingResult() }

// PlanMagicItemCraftingPlanned: nothing is lacking; Outcome is the requirements
// (an Artifact is a MagicItemNotCraftable outcome, not a rejection).
type PlanMagicItemCraftingPlanned struct{ outcome MagicItemCraftingOutcome }

// Outcome returns the crafting outcome.
func (r PlanMagicItemCraftingPlanned) Outcome() MagicItemCraftingOutcome { return r.outcome }

// PlanMagicItemCraftingRejected: the team lacks something; every shortfall is listed.
type PlanMagicItemCraftingRejected struct{ shortfalls []MagicItemCraftingShortfall }

// Shortfalls returns a new slice of the shortfalls, members first then spells.
func (r PlanMagicItemCraftingRejected) Shortfalls() []MagicItemCraftingShortfall {
	return slices.Clone(r.shortfalls)
}

// PlanMagicItemCraftingRarityUnresolved: no single rarity could be chosen.
type PlanMagicItemCraftingRarityUnresolved struct{ reason RarityNotResolvedReason }

// Reason returns why the rarity is unresolved.
func (r PlanMagicItemCraftingRarityUnresolved) Reason() RarityNotResolvedReason { return r.reason }

// PlanMagicItemCraftingContentMissing: the source lacks the first entry looked up.
type PlanMagicItemCraftingContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the missing entries.
func (r PlanMagicItemCraftingContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// PlanMagicItemCraftingContentIncomplete: the source has no tool entries or no
// usable Spell Scroll Costs table.
type PlanMagicItemCraftingContentIncomplete struct {
	reason MagicItemCraftingIncompleteReason
}

// Reason returns what is missing.
func (r PlanMagicItemCraftingContentIncomplete) Reason() MagicItemCraftingIncompleteReason {
	return r.reason
}

func (PlanMagicItemCraftingPlanned) isPlanMagicItemCraftingResult()           {}
func (PlanMagicItemCraftingRejected) isPlanMagicItemCraftingResult()          {}
func (PlanMagicItemCraftingRarityUnresolved) isPlanMagicItemCraftingResult()  {}
func (PlanMagicItemCraftingContentMissing) isPlanMagicItemCraftingResult()    {}
func (PlanMagicItemCraftingContentIncomplete) isPlanMagicItemCraftingResult() {}

// PlanMagicItemCrafting is the use case that plans the crafting of a magic item:
// it looks the item and what it needs up, resolves its rarity and reports the
// time and cost or every shortfall of the team (SRD 5.2.1, "Crafting Magic
// Items", pp. 206-207). Create it with NewPlanMagicItemCrafting; the zero value is
// not usable. It is safe for concurrent use only if its source, telemetry and
// clock are.
type PlanMagicItemCrafting struct {
	source MagicItemCraftingSource
	sig    operationSignals
}

// NewPlanMagicItemCrafting returns the use case reading through source. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewPlanMagicItemCrafting(source MagicItemCraftingSource, opts ...Option) (PlanMagicItemCrafting, error) {
	if source == nil {
		return PlanMagicItemCrafting{}, invalidInput("plan magic item crafting without a content source")
	}
	return PlanMagicItemCrafting{source: source, sig: newOperationSignals(newOptions(opts))}, nil
}

// planLookups is what the content lookups of one call found.
type planLookups struct {
	entry     content.MagicItemEntry
	incEntry  content.EquipmentEntry
	gearEntry content.EquipmentEntry
	tools     []content.Tool
	scribing  ScrollScribing
}

// Do plans the crafting. Lookups are made in this order: the magic item, the
// spells the facts name, the incorporated item, the gear item of an equipment
// rules basis, then the Equipment list for the tools and Spell Scroll Costs. A
// rarity that cannot be resolved, or an Artifact, ends the call before the
// lookups that follow. Missing content, an unresolved rarity, a shortfall and a
// not-craftable Artifact are results.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case, zero facts or team, or an incorporated choice
// that does not match the facts, and ctx.Err() unwrapped, before any signal, when
// ctx is already cancelled. It returns a *srderr.PortError (error type
// "content_error") with a nil result when the source fails, panics or breaks its
// contract; the failure is recorded once and never logged.
func (u PlanMagicItemCrafting) Do(ctx context.Context, cmd PlanMagicItemCraftingCommand) (PlanMagicItemCraftingResult, error) {
	if u.source == nil || !cmd.Facts.ID().Valid() || cmd.Team.IsZero() {
		return nil, invalidInput("plan magic item crafting: zero use case, facts or team")
	}
	inc, hasInc := cmd.Facts.IncorporatedItem().(content.IncorporatedItem)
	if hasInc != (cmd.Incorporated != 0) || cmd.Incorporated != 0 && cmd.Incorporated != PayPurchaseCost && cmd.Incorporated != CraftIncorporatedItem {
		return nil, invalidInput("plan magic item crafting: incorporated choice does not match the facts")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanPlanMagicItemCrafting, ValueOperationPlanMagicItemCrafting)
	defer run.end(ctx)
	ctx, lspan := u.sig.tel.Tracer.Start(ctx, SpanLoadContent)
	defer lspan.End()
	fail := func(err error) (PlanMagicItemCraftingResult, error) {
		lspan.RecordError(err)
		lspan.SetStatus(telemetry.StatusError)
		lspan.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	var lk planLookups
	entry, found, err := lookupEntry(ctx, u.source, cmd.Facts.ID().Key())
	if err != nil {
		return fail(err)
	}
	if !found {
		return u.missing(ctx, run, lspan, cmd.Facts.ID().Missing())
	}
	var ok bool
	if lk.entry, ok = entry.(content.MagicItemEntry); !ok {
		return fail(srderr.ContractViolation(content.Port, "lookup", "a magic item key answered with another entry type"))
	}
	resolution, err := ResolvedRarity{}.Of(lk.entry.Rarity(), cmd.RarityChoice)
	if err != nil { // unreachable: the entry's rarity spec is validated
		return nil, err
	}
	rarity, resolved := resolution.(ResolvedRarity)
	if !resolved {
		lspan.End()
		run.finish(ctx, ValueResultRejected, ValueRejectionRarityUnresolved)
		return PlanMagicItemCraftingRarityUnresolved{reason: resolution.(RarityNotResolved).Reason}, nil
	}
	if rarity.Rarity == content.RarityArtifact {
		run.finish(ctx, ValueResultPlanned, "")
		return PlanMagicItemCraftingPlanned{outcome: MagicItemNotCraftable{Reason: MagicItemNotCraftableArtifact}}, nil
	}
	spells := cmd.Facts.AllowedSpells()
	for _, s := range spells {
		if _, found, err = lookupEntry(ctx, u.source, s.Key()); err != nil {
			return fail(err)
		} else if !found {
			return u.missing(ctx, run, lspan, s.Missing())
		}
	}
	lookupGear := func(id content.ItemID) (content.EquipmentEntry, bool, error) {
		e, found, err := lookupEntry(ctx, u.source, id.Key())
		if err != nil || !found {
			return nil, found, err
		}
		eq, ok := e.(content.EquipmentEntry)
		if !ok {
			return nil, true, srderr.ContractViolation(content.Port, "lookup", "an equipment key answered with another entry type")
		}
		return eq, true, nil
	}
	if hasInc {
		if lk.incEntry, found, err = lookupGear(inc.Item()); err != nil {
			return fail(err)
		} else if !found {
			return u.missing(ctx, run, lspan, inc.Item().Missing())
		}
	}
	gearBasis, hasGear := cmd.Facts.CraftingBasis().(content.EquipmentRulesCrafting)
	if hasGear {
		if lk.gearEntry, found, err = lookupGear(gearBasis.Item()); err != nil {
			return fail(err)
		} else if !found {
			return u.missing(ctx, run, lspan, gearBasis.Item().Missing())
		}
	}
	all, err := srderr.GuardValue(content.Port, "list_kind", func() ([]content.Entry, error) {
		return u.source.ListKind(ctx, content.KindEquipment)
	})
	if err != nil {
		return fail(err)
	}
	var rows []content.SpellScrollCost
	for _, e := range all {
		switch v := e.(type) {
		case content.Tool:
			lk.tools = append(lk.tools, v)
		case content.SpellScrollCost:
			rows = append(rows, v)
		}
	}
	if len(lk.tools) == 0 {
		return u.incomplete(ctx, run, lspan, CraftingToolSetMissing)
	}
	lk.scribing, _ = NewScrollScribing(rows) // an incomplete table leaves the zero table; Crafting reports it below
	lspan.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess))
	lspan.End()

	category := lk.entry.Category()
	resolvedFacts := MagicItemFactResolution{}.Of(category, cmd.Facts)
	in := MagicItemCraftingInput{Rarity: rarity.Rarity, Consumable: resolvedFacts.Consumable, Basis: cmd.Facts.CraftingBasis(),
		Choice: cmd.Incorporated, IncorporatedCostCopper: inc.CostCopperPieces(), Workers: cmd.Team.Workers()}
	if hasGear {
		if in.GearOutcome, err = (Crafting{}).RequirementsFor(lk.gearEntry, lk.tools, lk.scribing); err != nil {
			run.commandError(err)
			return nil, err
		}
		if nc, bad := in.GearOutcome.(NotCraftable); bad && nc.Reason == NotCraftableScribingTableMissing {
			run.finish(ctx, ValueResultContentIncomplete, "")
			return PlanMagicItemCraftingContentIncomplete{reason: CraftingScribingTableMissing}, nil
		}
	}
	if cmd.Incorporated == CraftIncorporatedItem {
		if in.IncorporatedOutcome, err = (Crafting{}).RequirementsFor(lk.incEntry, lk.tools, lk.scribing); err != nil {
			run.commandError(err)
			return nil, err
		}
	}
	toolOptions, err := MagicItemCrafting{}.ToolsFor(category, cmd.BaseItem, lk.tools)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if short := (MagicItemCrafting{}).Shortfalls(cmd.Team, cmd.Qualifications, toolOptions, spells, cmd.Allowance); len(short) > 0 {
		run.finish(ctx, ValueResultRejected, short[0].Kind.String())
		return PlanMagicItemCraftingRejected{shortfalls: short}, nil
	}
	outcome, err := MagicItemCrafting{}.RequirementsFor(in)
	if err != nil { // unreachable: the input was built from validated parts
		run.commandError(err)
		return nil, err
	}
	run.finish(ctx, ValueResultPlanned, "")
	return PlanMagicItemCraftingPlanned{outcome: outcome}, nil
}

func (u PlanMagicItemCrafting) missing(ctx context.Context, run operationRun, lspan telemetry.Span, m content.MissingContent) (PlanMagicItemCraftingResult, error) {
	lspan.SetStatus(telemetry.StatusError)
	lspan.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
	run.finish(ctx, ValueResultContentMissing, "")
	return PlanMagicItemCraftingContentMissing{missing: []content.MissingContent{m}}, nil
}

func (u PlanMagicItemCrafting) incomplete(ctx context.Context, run operationRun, lspan telemetry.Span, r MagicItemCraftingIncompleteReason) (PlanMagicItemCraftingResult, error) {
	lspan.SetStatus(telemetry.StatusError)
	lspan.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
	run.finish(ctx, ValueResultContentIncomplete, "")
	return PlanMagicItemCraftingContentIncomplete{reason: r}, nil
}
