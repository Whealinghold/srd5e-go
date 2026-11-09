package character

import "slices"

// SpeciesTraitsTransition is the closed outcome of a transition of SpeciesTraits that
// a rule can refuse. Variants: SpeciesTraitsChanged, SpeciesTraitsRejected.
type SpeciesTraitsTransition interface{ isSpeciesTraitsTransition() }

// SpeciesTraitsChanged is the outcome of a transition that took effect: the new
// SpeciesTraits, the events in order (possibly none), the effect a used trait
// declares (nil when none) and the windows a rest opened. Only the transitions
// build one.
type SpeciesTraitsChanged struct {
	traits  SpeciesTraits
	events  []SpeciesTraitEvent
	effect  SpeciesTraitEffect
	windows []SpeciesWindow
}

// Traits returns the new SpeciesTraits.
func (c SpeciesTraitsChanged) Traits() SpeciesTraits { return c.traits }

// Events returns a new slice of the events, in order.
func (c SpeciesTraitsChanged) Events() []SpeciesTraitEvent { return slices.Clone(c.events) }

// Effect returns the effect the trait declares, nil when it declares none.
func (c SpeciesTraitsChanged) Effect() SpeciesTraitEffect { return c.effect }

// Windows returns a new slice of the windows a rest opened.
func (c SpeciesTraitsChanged) Windows() []SpeciesWindow { return slices.Clone(c.windows) }

// SpeciesTraitsRejected is the outcome of a transition a rule refused; the input is
// unchanged and no events were produced. Construct it with a keyed literal.
type SpeciesTraitsRejected struct{ Rejection SpeciesTraitRejection }

func (SpeciesTraitsChanged) isSpeciesTraitsTransition()  {}
func (SpeciesTraitsRejected) isSpeciesTraitsTransition() {}

// SpeciesUses expends and renews the limited uses of species traits (SRD 5.2.1, pp.
// 84-86). The zero value is ready to use. It is stateless and safe for concurrent use.
type SpeciesUses struct{}

// Expend expends the use req draws on and returns the effect it declares. It checks, in
// this order: the zero SpeciesTraits (SpeciesBehaviourUnknown), a use the species or
// lineage lacks (SpeciesUseNotOffered), a use with its own use case
// (SpeciesUseHasOwnUseCase), the minimum level (SpeciesUseNotYetAvailable),
// Large Form without room (NotEnoughSpaceForLargeForm) and no use left
// (SpeciesUseExhausted). The limit is read at the character's level now.
//
// The result is SpeciesTraitsChanged with one SpeciesUseExpended, or
// SpeciesTraitsRejected. It returns an error matching srderr.ErrInvalidInput for a zero
// Character, a nil request or one with an invalid shape, benefit or level. The inputs
// are unchanged. It never panics.
func (SpeciesUses) Expend(c Character, t SpeciesTraits, req SpeciesUseRequest) (SpeciesTraitsTransition, error) {
	use, ok := requestedUse(req)
	if c.IsZero() || !ok {
		return nil, invalidInput("expend a species trait use of a zero character or an invalid request")
	}
	return expendSpeciesUse(c, t, req, use), nil
}

// expendSpeciesUse is Expend for arguments already checked.
func expendSpeciesUse(c Character, t SpeciesTraits, req SpeciesUseRequest, use SpeciesUse) SpeciesTraitsTransition {
	level := c.TotalLevel()
	rejection := func() SpeciesTraitRejection {
		switch large, _ := req.(LargeFormRequest); {
		case t.IsZero():
			return SpeciesBehaviourUnknown{}
		case !t.offersRequest(req, use):
			return SpeciesUseNotOffered{}
		case hasOwnUseCase(req):
			return SpeciesUseHasOwnUseCase{}
		case level.Value() < use.MinimumLevel():
			return SpeciesUseNotYetAvailable{MinimumLevel: use.MinimumLevel()}
		case use == LargeFormUse && !large.HasEnoughSpace:
			return NotEnoughSpaceForLargeForm{}
		case t.ExpendedOf(use) >= use.Limit(level):
			return SpeciesUseExhausted{}
		}
		return nil
	}()
	if rejection != nil {
		return SpeciesTraitsRejected{Rejection: rejection}
	}
	effect, _ := SpeciesEffects{}.Of(req, c, t)
	next := t
	next.expended[use-1]++
	return SpeciesTraitsChanged{traits: next, effect: effect, events: []SpeciesTraitEvent{SpeciesUseExpended{id: c.ID(), use: use}}}
}

// Renew takes the RestApplied of this character and clears the expended counts the
// rest renews: a Long Rest every use, a Short Rest (or the benefits of a Short Rest from
// an interrupted Long Rest) only Adrenaline Rush. A Long Rest also opens the windows
// WindowResourceful (Human) and WindowHighElfCantripReplacement (High Elf, which also
// opens the swap). The one SpeciesUsesRenewed event lists the uses cleared and the
// windows opened; with neither, the result is SpeciesTraitsChanged with the same
// traits and no events.
//
// It checks the zero SpeciesTraits (SpeciesBehaviourUnknown) and then an event for
// another character (RestNotForCharacter). It returns an error matching
// srderr.ErrInvalidInput for a zero Character. The inputs are unchanged. It never panics.
func (SpeciesUses) Renew(c Character, t SpeciesTraits, rest RestApplied) (SpeciesTraitsTransition, error) {
	if c.IsZero() {
		return nil, invalidInput("renew species trait uses of a zero character")
	}
	switch {
	case t.IsZero():
		return SpeciesTraitsRejected{Rejection: SpeciesBehaviourUnknown{}}, nil
	case rest.CharacterID() != c.ID():
		return SpeciesTraitsRejected{Rejection: RestNotForCharacter{}}, nil
	}
	kind := rest.BenefitsOf()
	next := t
	var cleared []SpeciesUse
	for u := BreathWeaponUse; u <= GnomeSpellUse; u++ {
		if next.expended[u-1] > 0 && kind.Recharges(u.Renewal()) {
			next.expended[u-1] = 0
			cleared = append(cleared, u)
		}
	}
	var windows []SpeciesWindow
	if kind == LongRest {
		if t.species == speciesHuman {
			windows = append(windows, WindowResourceful)
		}
		if t.cantrip != "" {
			windows = append(windows, WindowHighElfCantripReplacement)
			next.window = true
		}
	}
	if len(cleared) == 0 && len(windows) == 0 {
		return SpeciesTraitsChanged{traits: t}, nil
	}
	return SpeciesTraitsChanged{traits: next, windows: windows,
		events: []SpeciesTraitEvent{SpeciesUsesRenewed{id: c.ID(), uses: cleared, windows: slices.Clone(windows)}}}, nil
}
