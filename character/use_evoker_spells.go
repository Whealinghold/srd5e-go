package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The four per-cast Evoker use cases load nothing: the host passes the Evoker
// SubclassDefinition (the zero value for none) and the facts of the spell. Each has its
// own rejected and not-a-Wizard outcomes.

// ---- ResolvePotentCantrip -----------------------------------------------

// ResolvePotentCantripCommand is the input of ResolvePotentCantrip.Run: a cantrip cast at a
// creature, the kind and outcome of the D20 Test against it and the damage it would do if it
// hit. Construct it with a keyed literal.
type ResolvePotentCantripCommand struct {
	ClassLevels ClassLevels
	Subclass    content.SubclassDefinition
	Spell       WizardSpellFacts
	Kind        d20.Kind
	Outcome     d20.Outcome
	Damage      int
}

// ResolvePotentCantripResult is the closed set of outcomes of ResolvePotentCantrip.Run.
// Variants: PotentCantripHalfDamage, PotentCantripUnchanged, ResolvePotentCantripRejected,
// ResolvePotentCantripNotAWizard.
type ResolvePotentCantripResult interface{ isResolvePotentCantripResult() }

// ResolvePotentCantripRejected is the outcome of a request a rule refused.
type ResolvePotentCantripRejected struct{ Rejection PotentCantripRejection }

// ResolvePotentCantripNotAWizard is the outcome for a character with no Wizard levels.
type ResolvePotentCantripNotAWizard struct{}

func (PotentCantripHalfDamage) isResolvePotentCantripResult()        {}
func (PotentCantripUnchanged) isResolvePotentCantripResult()         {}
func (ResolvePotentCantripRejected) isResolvePotentCantripResult()   {}
func (ResolvePotentCantripNotAWizard) isResolvePotentCantripResult() {}

// ResolvePotentCantrip is the use case that applies Potent Cantrip (SRD 5.2.1, "Level 3:
// Potent Cantrip", p. 82). Create it with NewResolvePotentCantrip.
type ResolvePotentCantrip struct{ wizardUse }

// NewResolvePotentCantrip returns the use case. Options: WithTelemetry, WithClock.
func NewResolvePotentCantrip(opts ...Option) ResolvePotentCantrip {
	return ResolvePotentCantrip{newWizardUse(nil, opts)}
}

// Run decides the effect. It returns an error matching srderr.ErrInvalidInput for the zero use
// case, spell facts with an invalid id or level, a D20 kind or outcome that is not valid and
// a negative damage; or the context's error.
func (u ResolvePotentCantrip) Run(ctx context.Context, cmd ResolvePotentCantripCommand) (ResolvePotentCantripResult, error) {
	if err := wizardCommandProblem(u.set, Identifier{}, false); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spell); err != nil {
		return nil, err
	}
	if !cmd.Kind.Valid() || (cmd.Outcome != d20.Success && cmd.Outcome != d20.Failure) || cmd.Damage < 0 {
		return nil, invalidInput("Potent Cantrip with an invalid D20 kind, outcome or damage")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanResolvePotentCantrip, ValueFeaturePotentCantrip, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return ResolvePotentCantripNotAWizard{}, nil
	}
	gate := EvokerRules{}.Gate(cmd.Subclass, call.level, PotentCantripID)
	out, rej := EvokerRules{}.PotentCantrip(gate, cmd.Spell, cmd.Kind, cmd.Outcome, cmd.Damage)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ResolvePotentCantripRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	if half, ok := out.(PotentCantripHalfDamage); ok {
		return half, nil
	}
	return PotentCantripUnchanged{}, nil
}

// ---- ResolveSculptSpells ------------------------------------------------

// ResolveSculptSpellsCommand is the input of ResolveSculptSpells.Run: an Evocation spell cast
// at CastLevel (the slot level; 0 for a cantrip) and the creatures the caster would protect.
// Construct it with a keyed literal.
type ResolveSculptSpellsCommand struct {
	ClassLevels ClassLevels
	Subclass    content.SubclassDefinition
	Spell       WizardSpellFacts
	CastLevel   int
	Chosen      []SculptCandidate
}

// ResolveSculptSpellsResult is the closed set of outcomes of ResolveSculptSpells.Run.
// Variants: SculptedCreatures, ResolveSculptSpellsRejected, ResolveSculptSpellsNotAWizard.
type ResolveSculptSpellsResult interface{ isResolveSculptSpellsResult() }

// ResolveSculptSpellsRejected is the outcome of a request a rule refused.
type ResolveSculptSpellsRejected struct{ Rejection SculptRejection }

// ResolveSculptSpellsNotAWizard is the outcome for a character with no Wizard levels.
type ResolveSculptSpellsNotAWizard struct{}

func (SculptedCreatures) isResolveSculptSpellsResult()             {}
func (ResolveSculptSpellsRejected) isResolveSculptSpellsResult()   {}
func (ResolveSculptSpellsNotAWizard) isResolveSculptSpellsResult() {}

// ResolveSculptSpells is the use case that applies Sculpt Spells (SRD 5.2.1, "Level 6:
// Sculpt Spells", p. 82). Create it with NewResolveSculptSpells.
type ResolveSculptSpells struct{ wizardUse }

// NewResolveSculptSpells returns the use case. Options: WithTelemetry, WithClock.
func NewResolveSculptSpells(opts ...Option) ResolveSculptSpells {
	return ResolveSculptSpells{newWizardUse(nil, opts)}
}

// sculptCommandProblem returns an error matching srderr.ErrInvalidInput for a cast level the
// spell cannot be cast at (below its level, a cantrip above 0, or above 9) and for a chosen
// creature with a zero id or one named twice.
func sculptCommandProblem(cmd ResolveSculptSpellsCommand) error {
	if cmd.CastLevel < int(cmd.Spell.Level) || cmd.CastLevel > content.MaxSpellLevel || (cmd.Spell.IsCantrip() && cmd.CastLevel != 0) {
		return invalidInput("Sculpt Spells with a cast level the spell cannot be cast at")
	}
	for i, c := range cmd.Chosen {
		if c.ID.IsZero() || slices.ContainsFunc(cmd.Chosen[:i], func(o SculptCandidate) bool { return o.ID == c.ID }) {
			return invalidInput("Sculpt Spells with a zero or repeated creature id")
		}
	}
	return nil
}

// Run decides the choice, with the errors of ResolvePotentCantrip.Run and the cast level and
// creature ids the command names.
func (u ResolveSculptSpells) Run(ctx context.Context, cmd ResolveSculptSpellsCommand) (ResolveSculptSpellsResult, error) {
	if err := wizardCommandProblem(u.set, Identifier{}, false); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spell); err != nil {
		return nil, err
	}
	if err := sculptCommandProblem(cmd); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanResolveSculptSpells, ValueFeatureSculptSpells, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return ResolveSculptSpellsNotAWizard{}, nil
	}
	gate := EvokerRules{}.Gate(cmd.Subclass, call.level, SculptSpellsID)
	out, rej := EvokerRules{}.SculptSpells(gate, cmd.Spell, cmd.CastLevel, cmd.Chosen)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ResolveSculptSpellsRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return out, nil
}

// ---- ResolveEmpoweredEvocation ------------------------------------------

// ResolveEmpoweredEvocationCommand is the input of ResolveEmpoweredEvocation.Run: a spell
// being cast, the Intelligence modifier, whether the caster chooses to apply the feature and
// the state of the cast. Construct it with a keyed literal.
type ResolveEmpoweredEvocationCommand struct {
	ClassLevels  ClassLevels
	Subclass     content.SubclassDefinition
	Spell        WizardSpellFacts
	Intelligence ability.Modifier
	Applies      bool
	State        EmpoweredCastState
}

// ResolveEmpoweredEvocationResult is the closed set of outcomes of ResolveEmpoweredEvocation.Run.
// Variants: EmpoweredRoll, ResolveEmpoweredEvocationRejected, ResolveEmpoweredEvocationNotAWizard.
type ResolveEmpoweredEvocationResult interface{ isResolveEmpoweredEvocationResult() }

// ResolveEmpoweredEvocationRejected is the outcome of a request a rule refused.
type ResolveEmpoweredEvocationRejected struct{ Rejection EmpoweredRejection }

// ResolveEmpoweredEvocationNotAWizard is the outcome for a character with no Wizard levels.
type ResolveEmpoweredEvocationNotAWizard struct{}

func (EmpoweredRoll) isResolveEmpoweredEvocationResult()                       {}
func (ResolveEmpoweredEvocationRejected) isResolveEmpoweredEvocationResult()   {}
func (ResolveEmpoweredEvocationNotAWizard) isResolveEmpoweredEvocationResult() {}

// ResolveEmpoweredEvocation is the use case that applies Empowered Evocation (SRD 5.2.1,
// "Level 10: Empowered Evocation", p. 82). Create it with NewResolveEmpoweredEvocation.
type ResolveEmpoweredEvocation struct{ wizardUse }

// NewResolveEmpoweredEvocation returns the use case. Options: WithTelemetry, WithClock.
func NewResolveEmpoweredEvocation(opts ...Option) ResolveEmpoweredEvocation {
	return ResolveEmpoweredEvocation{newWizardUse(nil, opts)}
}

// Run decides the roll modifier, with the errors of ResolvePotentCantrip.Run that apply.
func (u ResolveEmpoweredEvocation) Run(ctx context.Context, cmd ResolveEmpoweredEvocationCommand) (ResolveEmpoweredEvocationResult, error) {
	if err := wizardCommandProblem(u.set, Identifier{}, false); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spell); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanResolveEmpoweredEvocation, ValueFeatureEmpoweredEvocation, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return ResolveEmpoweredEvocationNotAWizard{}, nil
	}
	gate := EvokerRules{}.Gate(cmd.Subclass, call.level, EmpoweredEvocationID)
	out, rej := EvokerRules{}.EmpoweredEvocation(gate, cmd.Spell, cmd.Intelligence, cmd.Applies, cmd.State)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ResolveEmpoweredEvocationRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return out, nil
}

// ---- UseOverchannel -----------------------------------------------------

// UseOverchannelCommand is the input of UseOverchannel.Run. Construct it with a keyed literal.
type UseOverchannelCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Subclass    content.SubclassDefinition
	Resources   WizardResources
	Request     OverchannelRequest
}

// UseOverchannelResult is the closed set of outcomes of UseOverchannel.Run. Variants:
// OverchannelApplied, UseOverchannelRejected, UseOverchannelNotAWizard.
type UseOverchannelResult interface{ isUseOverchannelResult() }

// OverchannelApplied is the outcome of a use the rules allowed: the spell deals maximum
// damage on the turn it is cast, and after the first use the caster takes the Necrotic
// damage rolled here (it ignores Resistance and Immunity).
type OverchannelApplied struct {
	use       int
	slotLevel int
	necrotic  dice.Result
	rolled    bool
	resources WizardResources
	events    []Event
}

// Use returns the use number since the last Long Rest (1 is the first).
func (r OverchannelApplied) Use() int { return r.use }

// SlotLevel returns the level of the slot expended.
func (r OverchannelApplied) SlotLevel() int { return r.slotLevel }

// MaximumDamage reports that the spell deals maximum damage on the turn it is cast; it is always true.
func (r OverchannelApplied) MaximumDamage() bool { return true }

// Necrotic returns the Necrotic damage roll the caster takes immediately after the cast and
// true, or false for the first use.
func (r OverchannelApplied) Necrotic() (dice.Result, bool) { return r.necrotic, r.rolled }

// Resources returns the new WizardResources, with the use count raised by one.
func (r OverchannelApplied) Resources() WizardResources { return r.resources }

// Events returns a new slice holding the OverchannelUsed event.
func (r OverchannelApplied) Events() []Event { return slices.Clone(r.events) }

// UseOverchannelRejected is the outcome of a request a rule refused; it is not a use.
type UseOverchannelRejected struct{ Rejection OverchannelRejection }

// UseOverchannelNotAWizard is the outcome for a character with no Wizard levels.
type UseOverchannelNotAWizard struct{}

func (OverchannelApplied) isUseOverchannelResult()       {}
func (UseOverchannelRejected) isUseOverchannelResult()   {}
func (UseOverchannelNotAWizard) isUseOverchannelResult() {}

// UseOverchannel is the use case that applies Overchannel (SRD 5.2.1, "Level 14:
// Overchannel", p. 82). It rolls only the Necrotic damage. Create it with NewUseOverchannel.
type UseOverchannel struct {
	wizardUse
	roller Roller
}

// NewUseOverchannel returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewUseOverchannel(roller Roller, opts ...Option) (UseOverchannel, error) {
	if roller == nil {
		return UseOverchannel{}, invalidInput("use Overchannel without a roller")
	}
	return UseOverchannel{wizardUse: newWizardUse(nil, opts), roller: roller}, nil
}

// Run decides the use and rolls the Necrotic damage. It returns an error matching
// srderr.ErrInvalidInput for the zero use case or CharacterID and spell facts with an invalid
// id or level; the context's error; and the *srderr.PortError of a failing roller.
func (u UseOverchannel) Run(ctx context.Context, cmd UseOverchannelCommand) (UseOverchannelResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Request.Spell); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanUseOverchannel, ValueFeatureOverchannel, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return UseOverchannelNotAWizard{}, nil
	}
	gate := EvokerRules{}.Gate(cmd.Subclass, call.level, OverchannelID)
	plan, rej := EvokerRules{}.Overchannel(gate, cmd.Request, cmd.Resources.OverchannelUses())
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return UseOverchannelRejected{Rejection: rej}, nil
	}
	expr, hasDamage, err := plan.Necrotic()
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	out := OverchannelApplied{use: plan.Use(), slotLevel: plan.SlotLevel()}
	if hasDamage {
		if out.necrotic, err = dice.Roll(u.roller, expr); err != nil {
			call.run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		out.rolled = true
	}
	if out.resources, err = cmd.Resources.WithOverchannelUses(plan.Use()); err != nil {
		call.run.commandError(err)
		return nil, err
	}
	out.events = []Event{OverchannelUsed{id: cmd.CharacterID, use: plan.Use(), slotLevel: plan.SlotLevel()}}
	call.run.succeeded(ctx)
	return out, nil
}
