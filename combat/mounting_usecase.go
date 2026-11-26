package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// mountingUse is what every mounting use case holds: the instrumenter for the shared
// signals and the telemetry for the signals of mounted-and-underwater-combat.md section
// 7. The zero value is silent and usable.
type mountingUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newMountingUse(opts []Option) mountingUse {
	o := newOptions(opts)
	return mountingUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// recordMovementSpent records the movement cost of one accepted mount or dismount, 0 included.
func (u mountingUse) recordMovementSpent(ctx context.Context, op combatOperation, costFeet int) {
	u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricMountingMovementSpent, Unit: "[ft_i]",
		Description: "Movement spent per accepted mount or dismount."}).
		Record(ctx, float64(costFeet), telemetry.String(attrOperation, op.Value()))
}

// controlExtras returns the srd5e.combat.mount_control attribute for c, or none when the
// rider is not mounted.
func controlExtras(c MountControl, ok bool) []telemetry.Attr {
	if !ok {
		return nil
	}
	return []telemetry.Attr{telemetry.String(attrMountControl, c.String())}
}
