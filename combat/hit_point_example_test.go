package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// A creature that resists Fire takes half of a Fire hit, and Temporary Hit
// Points are lost first (SRD 5.2.1, "Resistance and Vulnerability", p. 17).
func ExampleApplyDamage() {
	resist, _ := combat.NewDamageAffinities([]combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}}, nil, nil)
	st := combat.MustHitPointStatus(20, combat.FallsUnconscious, resist)
	id := combat.MustCombatantID("goblin-1")
	cmd, _ := combat.NewApplyDamageCommand(id, st, []combat.Damage{combat.MustDamage(15, content.DamageFire)}, nil,
		combat.OtherSource, combat.NormalHit, nil)
	res, _ := combat.NewApplyDamage().Run(context.Background(), cmd)
	applied := res.(combat.DamageApplied)
	fmt.Println(applied.Status().HitPoints().Current(), applied.Events()[0].Kind())
	// Output: 13 damage_taken
}
