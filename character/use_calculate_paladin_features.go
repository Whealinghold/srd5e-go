package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// CalculatePaladinFeaturesCommand is the input of CalculatePaladinFeatures.Run. Subclass is the subclass the Paladin holds ("" for
// none); Charisma and Incapacitated feed the aura. Construct it with a keyed literal. Run does not alter it.
type CalculatePaladinFeaturesCommand struct {
	Character     Character
	Subclass      content.SubclassID
	Charisma      ability.Modifier
	Incapacitated bool
}

// CalculatePaladinFeaturesResult is the closed set of outcomes of CalculatePaladinFeatures.Run. Variants:
// CalculatePaladinFeaturesSummary, CalculatePaladinFeaturesNotAPaladin.
type CalculatePaladinFeaturesResult interface{ isCalculatePaladinFeaturesResult() }

// CalculatePaladinFeaturesSummary is the Paladin features at the character's level, computed from the definition. Only Run builds
// one.
type CalculatePaladinFeaturesSummary struct {
	auras          AuraEffects
	attacks        int
	radiant        bool
	alwaysPrep     []content.SpellID
	featGrants     []content.FeatCategory
	pending        []PaladinPendingChoice
	blessedWarrior []GrantedCantrip
}

// Auras returns the Aura of Protection and what rides on it.
func (s CalculatePaladinFeaturesSummary) Auras() AuraEffects { return s.auras }

// AttacksPerAttackAction returns 1 or, from Extra Attack, 2.
func (s CalculatePaladinFeaturesSummary) AttacksPerAttackAction() int { return s.attacks }

// RadiantStrikes reports whether Radiant Strikes is had; its data is RadiantStrikes.ExtraDamage.
func (s CalculatePaladinFeaturesSummary) RadiantStrikes() bool { return s.radiant }

// AlwaysPreparedSpells returns the spells always prepared, unfiltered by the Paladin list, as a new slice.
func (s CalculatePaladinFeaturesSummary) AlwaysPreparedSpells() []content.SpellID {
	return append([]content.SpellID{}, s.alwaysPrep...)
}

// FeatGrants returns the feat categories granted up to the level (the Fighting Style excluded), as a new slice.
func (s CalculatePaladinFeaturesSummary) FeatGrants() []content.FeatCategory {
	return append([]content.FeatCategory{}, s.featGrants...)
}

// PendingChoices returns the Paladin-specific choices the level grants and the character has not made, as a new slice.
func (s CalculatePaladinFeaturesSummary) PendingChoices() []PaladinPendingChoice {
	return append([]PaladinPendingChoice{}, s.pending...)
}

// BlessedWarriorCantrips returns the Blessed Warrior cantrips to hand to spellcasting, as a new slice.
func (s CalculatePaladinFeaturesSummary) BlessedWarriorCantrips() []GrantedCantrip {
	return append([]GrantedCantrip{}, s.blessedWarrior...)
}

// CalculatePaladinFeaturesNotAPaladin is the outcome for a character with no Paladin levels.
type CalculatePaladinFeaturesNotAPaladin struct{}

func (CalculatePaladinFeaturesSummary) isCalculatePaladinFeaturesResult()     {}
func (CalculatePaladinFeaturesNotAPaladin) isCalculatePaladinFeaturesResult() {}

// CalculatePaladinFeatures is the use case that summarises the Paladin features a level gives (class-paladin.md FR-042, FR-058,
// FR-077 to FR-088). It changes nothing and has no rejection. Create it with NewCalculatePaladinFeatures; the zero value is not
// usable. It is safe for concurrent use only if its telemetry and clock are.
type CalculatePaladinFeatures struct{ paladinUse }

// NewCalculatePaladinFeatures returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression.
func NewCalculatePaladinFeatures(p PaladinProgression, opts ...Option) (CalculatePaladinFeatures, error) {
	u, err := newPaladinUse("calculate Paladin features", p, opts)
	return CalculatePaladinFeatures{u}, err
}

// Run summarises the features of cmd.Character.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u CalculatePaladinFeatures) Run(ctx context.Context, cmd CalculatePaladinFeaturesCommand) (CalculatePaladinFeaturesResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanCalculatePaladinFeatures, ValueClassPaladin, ValueFeatureCalculateFeatures)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return CalculatePaladinFeaturesNotAPaladin{}, nil
	}
	p := u.progression
	oath := oathChosen(cmd.Subclass)
	s := CalculatePaladinFeaturesSummary{
		auras:          PaladinAuras{}.Compute(p, level, oath, cmd.Charisma, cmd.Incapacitated),
		attacks:        AttacksPerAttackAction(p, level),
		radiant:        p.has(PaladinFeatureRadiantStrikes, level),
		alwaysPrep:     p.AlwaysPrepared(level, oath),
		featGrants:     []content.FeatCategory{},
		pending:        []PaladinPendingChoice{},
		blessedWarrior: state.BlessedWarriorCantrips(),
	}
	for l := 1; l <= level.n; l++ {
		s.featGrants = append(s.featGrants, p.FeatEntitlements(MustPaladinLevel(l))...)
	}
	if _, chosen := state.FightingStyle(); !chosen && p.has(PaladinFeatureFightingStyle, level) {
		s.pending = append(s.pending, PaladinPendingFightingStyle)
	}
	run.succeeded(ctx)
	return s, nil
}
