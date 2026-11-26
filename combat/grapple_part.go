package combat

import "strings"

// GraspingPart is the part of a creature that holds a grappled creature: a hand
// or another body part (SRD 5.2.1, "Grappling", "One Grapple per Hand", p. 182).
// The engine models no anatomy; the host names the part. Variants:
// GraspingHand, GraspingBodyPart. Both are comparable with ==, so two parts are
// equal when they are the same variant with the same index or name. A nil
// GraspingPart is invalid.
type GraspingPart interface{ graspingPart() }

// GraspingHand is the hand of a creature at index, counted from 0. The zero
// value is hand 0. Build it with NewGraspingHand.
type GraspingHand struct{ index int }

// GraspingBodyPart is a body part other than a hand, such as a tentacle or a
// maw, named in the host's own word. Build it with NewGraspingBodyPart; the zero
// value is invalid.
type GraspingBodyPart struct{ name string }

func (GraspingHand) graspingPart()     {}
func (GraspingBodyPart) graspingPart() {}

// NewGraspingHand returns hand index. It returns a *GrappleProblemError (matching
// srderr.ErrInvalidInput) carrying GrappleProblemNegativeHandIndex for a negative
// index. It never panics.
func NewGraspingHand(index int) (GraspingHand, error) {
	if index < 0 {
		return GraspingHand{}, problemError(GrappleProblemNegativeHandIndex)
	}
	return GraspingHand{index}, nil
}

// MustGraspingHand is NewGraspingHand for literals known to be valid. It panics
// with the error NewGraspingHand returns, and only then.
func MustGraspingHand(index int) GraspingHand {
	h, err := NewGraspingHand(index)
	if err != nil {
		panic(err)
	}
	return h
}

// Index returns the hand's index.
func (h GraspingHand) Index() int { return h.index }

// NewGraspingBodyPart returns the body part called name. It returns a
// *GrappleProblemError (matching srderr.ErrInvalidInput) carrying
// GrappleProblemBlankPartName for an empty or whitespace-only name or one with
// leading or trailing whitespace. It never panics.
func NewGraspingBodyPart(name string) (GraspingBodyPart, error) {
	if name == "" || name != strings.TrimSpace(name) {
		return GraspingBodyPart{}, problemError(GrappleProblemBlankPartName)
	}
	return GraspingBodyPart{name}, nil
}

// MustGraspingBodyPart is NewGraspingBodyPart for literals known to be valid. It
// panics with the error NewGraspingBodyPart returns, and only then.
func MustGraspingBodyPart(name string) GraspingBodyPart {
	p, err := NewGraspingBodyPart(name)
	if err != nil {
		panic(err)
	}
	return p
}

// Name returns the body part's name.
func (p GraspingBodyPart) Name() string { return p.name }

// validPart reports whether p is a usable GraspingPart.
func validPart(p GraspingPart) bool {
	switch v := p.(type) {
	case GraspingHand:
		return v.index >= 0
	case GraspingBodyPart:
		return v.name != "" && v.name == strings.TrimSpace(v.name)
	}
	return false
}
