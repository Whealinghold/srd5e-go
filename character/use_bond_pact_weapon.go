package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// BondPactWeaponCommand is the input of BondPactWeapon.Run. Weapon is the weapon's equipment id; Magic says it is a magic weapon the
// Warlock touches (no content is read for it) and Claim who else holds it. A conjured weapon is read from the content source. Construct it
// with a keyed literal. Run does not alter it.
type BondPactWeaponCommand struct {
	Character Character
	Weapon    content.ItemID
	Magic     bool
	Claim     MagicWeaponClaim
}

// BondPactWeaponResult is the closed set of outcomes of BondPactWeapon.Run. Variants: BondPactWeaponApplied, BondPactWeaponRejected,
// BondPactWeaponContentMissing.
type BondPactWeaponResult interface{ isBondPactWeaponResult() }

// BondPactWeaponApplied is the outcome of a bond the rules allowed.
type BondPactWeaponApplied struct {
	character Character
	bond      PactWeaponBond
	events    []Event
}

// Character returns the new Character holding the new WarlockPactState.
func (r BondPactWeaponApplied) Character() Character { return r.character }

// Bond returns the new bond.
func (r BondPactWeaponApplied) Bond() PactWeaponBond { return r.bond }

// Events returns a new slice holding the PactWeaponBondEnded event for a bond replaced, then the PactWeaponBonded event.
func (r BondPactWeaponApplied) Events() []Event { return slices.Clone(r.events) }

// BondPactWeaponRejected is the outcome of a bond a rule refused: the Character is unchanged.
type BondPactWeaponRejected struct{ Rejection WarlockRejection }

// BondPactWeaponContentMissing is the outcome when a conjured weapon is not in the source.
type BondPactWeaponContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r BondPactWeaponContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (BondPactWeaponApplied) isBondPactWeaponResult()        {}
func (BondPactWeaponRejected) isBondPactWeaponResult()       {}
func (BondPactWeaponContentMissing) isBondPactWeaponResult() {}

// BondPactWeapon is the use case that bonds a pact weapon with Pact of the Blade (SRD 5.2.1, "Pact of the Blade", p. 74; class-warlock.md
// FR-081 to FR-084, FR-086, FR-141): it loads a conjured weapon and calls FormPactWeaponBond. Create it with NewBondPactWeapon; the zero
// value is not usable.
type BondPactWeapon struct{ warlockUse }

// NewBondPactWeapon returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a nil
// source.
func NewBondPactWeapon(source ContentSource, opts ...Option) (BondPactWeapon, error) {
	u, err := newWarlockUse("bond a pact weapon", source, true, opts)
	return BondPactWeapon{u}, err
}

// Run bonds the weapon. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character, and, recorded on the span, for an invalid weapon id; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a
// *srderr.PortError when the source fails. A refusal is a BondPactWeaponRejected result; a conjured weapon not in the source is a
// ContentMissing result. An equipment entry that is not a weapon is refused as WarlockNotMeleeSimpleOrMartial.
func (u BondPactWeapon) Run(ctx context.Context, cmd BondPactWeaponCommand) (BondPactWeaponResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanBondPactWeapon, ValueFeaturePactWeapon, cmd.Character, false)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return BondPactWeaponRejected{Rejection: rej}, nil
	}
	if !cmd.Weapon.Valid() {
		err := invalidInput("bond a pact weapon with an invalid weapon id")
		call.run.commandError(err)
		return nil, err
	}
	target := MagicPactWeapon(cmd.Weapon, cmd.Claim)
	if !cmd.Magic {
		entries, missing, err := u.lookupAll(ctx, call, ValueFeaturePactWeapon, []content.Key{cmd.Weapon.Key()})
		if err != nil {
			return nil, err
		}
		if len(missing) != 0 {
			call.run.contentMissing(ctx)
			return BondPactWeaponContentMissing{missing: missing}, nil
		}
		if _, ok := entries[0].(content.EquipmentEntry); !ok {
			return nil, call.wrongEntry(ctx, cmd.Weapon.Key())
		}
		target = ConjuredPactWeapon(cmd.Weapon, 0, 0)
		if w, ok := entries[0].(content.Weapon); ok {
			target = ConjuredPactWeapon(cmd.Weapon, w.Category(), w.Type())
		}
	}
	next, changed, rej, err := call.commitPact(ctx, cmd.Character, FormPactWeaponBond(cmd.Character.ID(), call.state, call.pact, target))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return BondPactWeaponRejected{Rejection: rej}, nil
	}
	return BondPactWeaponApplied{character: next, bond: changed.state.bond, events: changed.Events()}, nil
}
