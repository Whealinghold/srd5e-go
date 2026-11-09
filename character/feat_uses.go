package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FeatRenewal is a moment that renews a feat's expended use (SRD 5.2.1, "Magic
// Initiate", "Boon of Fate", pp. 87-88). The zero value is invalid. Comparable
// and safe for concurrent use.
type FeatRenewal int

// The renewal moments.
const (
	// RenewOnInitiativeRolled is rolling Initiative.
	RenewOnInitiativeRolled FeatRenewal = iota + 1
	// RenewOnShortRestFinished is finishing a Short Rest.
	RenewOnShortRestFinished
	// RenewOnLongRestFinished is finishing a Long Rest.
	RenewOnLongRestFinished
)

// Valid reports whether r is one of the three moments.
func (r FeatRenewal) Valid() bool {
	return r >= RenewOnInitiativeRolled && r <= RenewOnLongRestFinished
}

// String returns "initiativeRolled", "shortRestFinished", "longRestFinished" or
// "unknown".
func (r FeatRenewal) String() string {
	return featNameOf(int(r), "initiativeRolled", "shortRestFinished", "longRestFinished")
}

// UseRenewedBy is the set of moments that renew a feat's expended use, and a
// FeatUsageLimit variant. It is comparable. The zero value renews on none.
type UseRenewedBy struct{ mask uint8 }

// NewUseRenewedBy returns the set of renewals. It returns an error matching
// srderr.ErrInvalidInput for no renewal or an invalid one. It never panics.
func NewUseRenewedBy(renewals ...FeatRenewal) (UseRenewedBy, error) {
	var u UseRenewedBy
	for _, r := range renewals {
		if !r.Valid() {
			return UseRenewedBy{}, invalidInput("feat renewal")
		}
		u.mask |= 1 << r
	}
	if u.mask == 0 {
		return UseRenewedBy{}, invalidInput("use renewed by no moment")
	}
	return u, nil
}

// Renews reports whether r renews the use.
func (u UseRenewedBy) Renews(r FeatRenewal) bool { return r.Valid() && u.mask&(1<<r) != 0 }

func (UseRenewedBy) isFeatUsageLimit() {}

// FeatUseState is the state of a feat's limited use, a closed set. Variants:
// FeatUseIsAvailable, FeatUseIsExpended.
type FeatUseState interface{ isFeatUseState() }

// FeatUseIsAvailable: the use can be expended.
type FeatUseIsAvailable struct{}

// FeatUseIsExpended: the use was expended and awaits renewal.
type FeatUseIsExpended struct{}

func (FeatUseIsAvailable) isFeatUseState() {}
func (FeatUseIsExpended) isFeatUseState()  {}

// FeatsTransition is the closed outcome of a transition of TakenFeats that a
// rule can refuse. Variants: FeatsChanged, FeatsRejected.
type FeatsTransition interface{ isFeatsTransition() }

// FeatsChanged is the outcome of a transition that took effect: the new
// TakenFeats and the events, in order (possibly none). Only the transitions
// build one.
type FeatsChanged struct {
	feats  TakenFeats
	events []FeatEvent
}

// Feats returns the new TakenFeats.
func (t FeatsChanged) Feats() TakenFeats { return t.feats }

// Events returns a new slice of the events, in order.
func (t FeatsChanged) Events() []FeatEvent { return slices.Clone(t.events) }

// FeatsRejected is the outcome of a transition a rule refused; the input is
// unchanged and no events were produced.
type FeatsRejected struct{ Rejection FeatRejection }

func (FeatsChanged) isFeatsTransition()  {}
func (FeatsRejected) isFeatsTransition() {}

// FeatUses expends and renews the limited uses of held feats (SRD 5.2.1, "Magic
// Initiate", p. 87; "Boon of Fate", p. 88). The zero value is ready to use. It is
// stateless and safe for concurrent use.
type FeatUses struct{}

// Expend expends the use of feat id. A feat held more than once (Magic Initiate)
// expends the first available instance.
//
// The result is FeatsChanged with the one FeatUseExpended, or FeatsRejected with
// FeatNotHeld, FeatHasNoLimitedUse (a held feat with no limited use) or
// FeatUseAlreadyExpended. It returns an error matching srderr.ErrInvalidInput
// for a zero Character or an invalid id. The inputs are unchanged. It never
// panics.
func (FeatUses) Expend(c Character, feats TakenFeats, id content.FeatID) (FeatsTransition, error) {
	if c.IsZero() || !id.Valid() {
		return nil, invalidInput("expend a feat use of a zero character or an invalid id")
	}
	return expendFeatUse(c, feats, id), nil
}

// expendFeatUse is Expend for arguments already checked.
func expendFeatUse(c Character, feats TakenFeats, id content.FeatID) FeatsTransition {
	held, limited, available := false, false, -1
	for i, t := range feats.taken {
		if t.feat != id {
			continue
		}
		held = true
		if t.use != nil {
			limited = true
			if _, ok := t.use.(FeatUseIsAvailable); ok && available < 0 {
				available = i
			}
		}
	}
	switch {
	case !held:
		return FeatsRejected{Rejection: FeatNotHeld{Feat: id}}
	case !limited:
		return FeatsRejected{Rejection: FeatHasNoLimitedUse{Feat: id}}
	case available < 0:
		return FeatsRejected{Rejection: FeatUseAlreadyExpended{Feat: id}}
	}
	next := slices.Clone(feats.taken)
	next[available].use = FeatUseIsExpended{}
	return FeatsChanged{feats: TakenFeats{taken: next}, events: []FeatEvent{FeatUseExpended{id: c.ID(), feat: id}}}
}

// Renew renews every expended use that lists the moment r. The one FeatUsesRenewed
// event lists the feats renewed, in the order held; when nothing is expended or
// listed the result is FeatsChanged with the same feats and no events.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or an
// invalid renewal. The inputs are unchanged. It never panics.
func (FeatUses) Renew(c Character, feats TakenFeats, r FeatRenewal) (FeatsTransition, error) {
	if c.IsZero() || !r.Valid() {
		return nil, invalidInput("renew feat uses of a zero character or an invalid renewal")
	}
	return renewFeatUses(c, feats, r), nil
}

// renewFeatUses is Renew for arguments already checked.
func renewFeatUses(c Character, feats TakenFeats, r FeatRenewal) FeatsTransition {
	next := slices.Clone(feats.taken)
	var renewed []content.FeatID
	for i, t := range next {
		if _, expended := t.use.(FeatUseIsExpended); !expended {
			continue
		}
		if shape, _ := shapeOf(t.feat); shape.renewedBy.Renews(r) {
			next[i].use = FeatUseIsAvailable{}
			renewed = append(renewed, t.feat)
		}
	}
	if len(renewed) == 0 {
		return FeatsChanged{feats: feats}
	}
	return FeatsChanged{feats: TakenFeats{taken: next}, events: []FeatEvent{FeatUsesRenewed{id: c.ID(), feats: renewed, renewal: r}}}
}
