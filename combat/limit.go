package combat

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// The per-call design caps of plan P10 for collections a constructor or pure operation reads. They are design
// numbers, not SRD rules.
const (
	// maxInputList is the cap of every per-call input list (cover sources, circumstances, observations and the like).
	maxInputList = 256
	// maxHideCover is the cap of the cover degrees of one HideScene.
	maxHideCover = 4
	// maxGridCreatures, maxTerrainSquares and maxPathDirections cap the creatures of a GridBoard, the squares of a
	// GridTerrain and the directions of one PlanPath call.
	maxGridCreatures  = 256
	maxTerrainSquares = 65536
	maxPathDirections = 4096
	// maxDamageParts is the cap of the damage components and instances of one damage call.
	maxDamageParts = 64
	// maxZoneRules is the cap of the rules of one zone, read by CreateZone; content does not bound them.
	maxZoneRules = 64
)

// LimitExceededError is the error of a constructor or pure operation given a collection over its plan P10 cap.
// Collection names the collection and Max is its cap. It matches srderr.ErrInvalidInput with errors.Is, is
// comparable, and its String is the shared srd5e.combat.result value "limit_exceeded". The input is unchanged.
// Construct with a keyed literal; the zero value names no collection.
type LimitExceededError struct {
	Collection string
	Max        int
}

// Error renders the error. The text is not a contract.
func (e LimitExceededError) Error() string {
	return fmt.Sprintf("combat: %s over its cap of %d", e.Collection, e.Max)
}

// Unwrap returns srderr.ErrInvalidInput.
func (e LimitExceededError) Unwrap() error { return srderr.ErrInvalidInput }

// String returns "limit_exceeded".
func (LimitExceededError) String() string { return valueResultLimitExceeded }

// overCap returns a LimitExceededError for a collection of n elements over max, and nil otherwise.
func overCap(collection string, n, max int) error {
	if n <= max {
		return nil
	}
	return LimitExceededError{Collection: collection, Max: max}
}
