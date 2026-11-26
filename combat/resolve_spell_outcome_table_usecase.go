package combat

import (
	"context"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

const (
	// maxRerolls is how many further requests of one expression a roll-again chain may make (spell-outcome-tables.md
	// FR-004); with a fair roller a longer chain has a chance below 2 to the power of -100.
	maxRerolls = 100
	// maxPrismaticTargets is the cap of targets of one Prismatic Spray: the combatants of an encounter (plan P10).
	maxPrismaticTargets = 256
)

// ResolveSpellOutcomeTableCommand is the closed set of commands of ResolveSpellOutcomeTable.Resolve. Variants:
// RollTeleportOutcomeCommand, RollConfusionBehaviorCommand, RollPrismaticSprayCommand,
// ResolvePrismaticWallCrossingCommand, RollReincarnationSpeciesCommand, RollBlinkTurnEndCommand.
type ResolveSpellOutcomeTableCommand interface{ isResolveSpellOutcomeTableCommand() }

// RollTeleportOutcomeCommand rolls the Teleportation Outcome table. Build it with a keyed literal; Resolve does not
// change it.
type RollTeleportOutcomeCommand struct {
	// Subject is who teleports.
	Subject TeleportSubject
	// Familiarity is the row the Game Master judged.
	Familiarity TeleportFamiliarity
}

// RollConfusionBehaviorCommand rolls the Confusion behavior for one turn of one target.
type RollConfusionBehaviorCommand struct {
	// CreaturesWithinReach are the creatures the target could attack, without the target, in a fixed order.
	CreaturesWithinReach []CombatantID
}

// RollPrismaticSprayCommand rolls the rays of a Prismatic Spray and the saves of its targets.
type RollPrismaticSprayCommand struct {
	// SaveDC is the spell save DC.
	SaveDC d20.DifficultyClass
	// Targets are the creatures in the Cone, in a fixed order; none is valid.
	Targets []PrismaticSaveTarget
}

// ResolvePrismaticWallCrossingCommand resolves one creature crossing the layers still present in a Prismatic Wall.
type ResolvePrismaticWallCrossingCommand struct {
	// SaveDC is the spell save DC.
	SaveDC d20.DifficultyClass
	// Wall is the state of the wall; the zero value is the intact wall.
	Wall PrismaticWallState
	// Creature is the crossing creature and how it attempts the save.
	Creature PrismaticSaveTarget
}

// RollReincarnationSpeciesCommand rolls the species of a Reincarnate, or takes the one the Game Master chose.
type RollReincarnationSpeciesCommand struct {
	// CreatureType is the type of the dead creature; only Humanoid is accepted.
	CreatureType content.CreatureType
	// DeadFor is how long the creature has been dead; at most 10 days is accepted.
	DeadFor time.Duration
	// ChosenSpecies, when not empty, is the species the Game Master chose; no die is rolled.
	ChosenSpecies content.SpeciesID
}

// RollBlinkTurnEndCommand rolls the 1d6 at the end of the caster's turn.
type RollBlinkTurnEndCommand struct {
	// Location is where the caster is now.
	Location BlinkLocation
}

func (RollTeleportOutcomeCommand) isResolveSpellOutcomeTableCommand()          {}
func (RollConfusionBehaviorCommand) isResolveSpellOutcomeTableCommand()        {}
func (RollPrismaticSprayCommand) isResolveSpellOutcomeTableCommand()           {}
func (ResolvePrismaticWallCrossingCommand) isResolveSpellOutcomeTableCommand() {}
func (RollReincarnationSpeciesCommand) isResolveSpellOutcomeTableCommand()     {}
func (RollBlinkTurnEndCommand) isResolveSpellOutcomeTableCommand()             {}

// ResolveSpellOutcomeTableResult is the closed set of outcomes of ResolveSpellOutcomeTable.Resolve. Variants:
// SpellTableTeleportRolled, SpellTableConfusionRolled, SpellTablePrismaticSprayRolled,
// SpellTablePrismaticWallCrossed, SpellTableReincarnationRolled, SpellTableBlinkRolled, SpellTableRefused. None
// applies anything: the host applies each Damage, condition and arrival.
type ResolveSpellOutcomeTableResult interface{ isResolveSpellOutcomeTableResult() }

// SpellTableTeleportRolled: the Teleport roll.
type SpellTableTeleportRolled struct{ outcome TeleportOutcome }

// Outcome returns the Teleport outcome.
func (r SpellTableTeleportRolled) Outcome() TeleportOutcome { return r.outcome }

// SpellTableConfusionRolled: the Confusion roll for one turn.
type SpellTableConfusionRolled struct{ turn ConfusionTurn }

// Turn returns the behaviour for the turn.
func (r SpellTableConfusionRolled) Turn() ConfusionTurn { return r.turn }

// SpellTablePrismaticSprayRolled: the rays and saves of a Prismatic Spray, one entry per target in command order;
// none when there was no target.
type SpellTablePrismaticSprayRolled struct{ targets []PrismaticTargetResult }

// Targets returns what the spray did to each target, as a new slice.
func (r SpellTablePrismaticSprayRolled) Targets() []PrismaticTargetResult {
	out := make([]PrismaticTargetResult, len(r.targets))
	for i, t := range r.targets {
		out[i] = PrismaticTargetResult{t.ID, slices.Clone(t.rays), t.save, slices.Clone(t.strikes)}
	}
	return out
}

// SpellTablePrismaticWallCrossed: one creature crossing a Prismatic Wall.
type SpellTablePrismaticWallCrossed struct{ crossing PrismaticWallCrossing }

// Crossing returns what the wall did to the creature.
func (r SpellTablePrismaticWallCrossed) Crossing() PrismaticWallCrossing {
	return PrismaticWallCrossing{r.crossing.Creature, slices.Clone(r.crossing.layers)}
}

// SpellTableReincarnationRolled: the species of a Reincarnate, rolled or chosen.
type SpellTableReincarnationRolled struct{ outcome ReincarnationOutcome }

// Outcome returns ReincarnationSpeciesRolled or ReincarnationSpeciesChosen.
func (r SpellTableReincarnationRolled) Outcome() ReincarnationOutcome { return r.outcome }

// SpellTableBlinkRolled: the Blink roll at the end of a turn.
type SpellTableBlinkRolled struct{ outcome BlinkTurnEndOutcome }

// Outcome returns the result of the 1d6.
func (r SpellTableBlinkRolled) Outcome() BlinkTurnEndOutcome { return r.outcome }

// SpellTableRefused: a Reincarnate the rules refuse; no die was rolled. A refusal is a normal rule outcome.
type SpellTableRefused struct{ refusals []SpellTableRefusal }

// Refusals returns every reason, NotHumanoid before DeadTooLong, as a new slice.
func (r SpellTableRefused) Refusals() []SpellTableRefusal { return slices.Clone(r.refusals) }

func (SpellTableTeleportRolled) isResolveSpellOutcomeTableResult()       {}
func (SpellTableConfusionRolled) isResolveSpellOutcomeTableResult()      {}
func (SpellTablePrismaticSprayRolled) isResolveSpellOutcomeTableResult() {}
func (SpellTablePrismaticWallCrossed) isResolveSpellOutcomeTableResult() {}
func (SpellTableReincarnationRolled) isResolveSpellOutcomeTableResult()  {}
func (SpellTableBlinkRolled) isResolveSpellOutcomeTableResult()          {}
func (SpellTableRefused) isResolveSpellOutcomeTableResult()              {}

// ResolveSpellOutcomeTable is the use case that rolls the printed tables of Teleport, Confusion, Prismatic Spray,
// Prismatic Wall, Reincarnate and Blink through the dice port and returns what the table says as plain values (SRD
// 5.2.1, pp. 113-114, 117, 154-155, 158, 168). It applies nothing, rolls no recurring save, produces no event and
// calls no other use case. Create it with NewResolveSpellOutcomeTable; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type ResolveSpellOutcomeTable struct {
	in     instrumenter
	roller Roller
}

// NewResolveSpellOutcomeTable returns the use case rolling through roller. Options: WithTelemetry, WithClock. It
// returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveSpellOutcomeTable(roller Roller, opts ...Option) (ResolveSpellOutcomeTable, error) {
	if roller == nil {
		return ResolveSpellOutcomeTable{}, invalidInput("spell outcome tables without a dice roller")
	}
	o := newOptions(opts)
	return ResolveSpellOutcomeTable{newInstrumenter(o.tel, o.clock), roller}, nil
}

// tableRoller rolls the dice of one call and remembers the table outcomes it looked up, to count them once the call
// has succeeded.
type tableRoller struct {
	roller   Roller
	table    string
	outcomes []string
}

// total rolls count dice of sides faces and returns their total. A roller failure is a *srderr.PortError.
func (r *tableRoller) total(count, sides int) (int, error) {
	die, err := dice.NewDie(sides)
	if err != nil {
		return 0, err
	}
	e, err := dice.NewExpression(count, die, 0)
	if err != nil {
		return 0, err
	}
	res, err := dice.Roll(r.roller, e)
	if err != nil {
		return 0, err
	}
	return res.Total(), nil
}

// face rolls one die of sides faces.
func (r *tableRoller) face(sides int) (int, error) { return r.total(1, sides) }

// chain rolls 1d<sides>, names each face with look (counted as a table roll) and rolls again while look says so. It
// returns the last face and the number of rerolls. The roll after the 100th reroll is not requested: it is a contract
// violation of the dice port (FR-004).
func (r *tableRoller) chain(sides int, look func(face int) (name string, again bool, err error)) (face, rerolls int, err error) {
	for {
		if face, err = r.face(sides); err != nil {
			return 0, 0, err
		}
		name, again, err := look(face)
		if err != nil {
			return 0, 0, err
		}
		r.outcomes = append(r.outcomes, name)
		if !again {
			return face, rerolls, nil
		}
		if rerolls == maxRerolls {
			return 0, 0, rerollLimit()
		}
		rerolls++
	}
}

// rerollLimit is the error of a roll-again chain that did not end in 100 rerolls.
func rerollLimit() error {
	return srderr.ContractViolation(dice.Port, "roll", "a roll-again table kept asking for another roll")
}

// record counts each table roll of a call that succeeded.
func (r *tableRoller) record(ctx context.Context, tel telemetry.Telemetry) {
	rolls := tel.Meter.Counter(telemetry.Descriptor{Name: metricSpellTableRolls, Unit: "{roll}",
		Description: "Rolls on a printed spell outcome table, by table and outcome."})
	for _, name := range r.outcomes {
		rolls.Add(ctx, 1, telemetry.String(attrSpellTable, r.table), telemetry.String(attrTableOutcome, name))
	}
}

// Resolve rolls the table of cmd, in the request order of spell-outcome-tables.md FR-006, and returns the result. A
// row whose outcome is certain requests no die, and a Reincarnate the rules refuse gives SpellTableRefused with no
// die.
//
// It records the shared signals under the operation resolve_spell_outcome_table with the result spell_table_rolled
// (class applied) or spell_table_refused (class refused), the attribute srd5e.combat.spell_table, and one count of
// srd5e.combat.spell_table_rolls per table roll of a call that returned a result. It never logs and produces no event.
//
// It returns an error matching srderr.ErrInvalidInput for the zero use case and, before any signal, ctx.Err()
// unwrapped for a cancelled ctx. It returns an error matching srderr.ErrInvalidInput, recorded on the span only and
// before any die, for a malformed command (a nil or unknown command, no subject, an invalid familiarity, a repeated
// or zero id, a missing save attempt, a creature type or dead time out of range), and one that also matches
// ErrSpellTableLimit for more than 256 Prismatic Spray targets or 1000 Confusion candidates. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails, panics or answers
// another expression, or when a roll-again chain makes more than 100 rerolls (srderr.ErrContractViolation); the
// failure is recorded once and no partial result is returned.
func (u ResolveSpellOutcomeTable) Resolve(ctx context.Context, cmd ResolveSpellOutcomeTableCommand) (ResolveSpellOutcomeTableResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveSpellOutcomeTable, spanResolveSpellOutcomeTable}
	return run(ctx, u.in, op, func(ctx context.Context) (ResolveSpellOutcomeTableResult, error) {
		r := &tableRoller{roller: u.roller}
		res, err := u.resolve(r, cmd)
		if err != nil {
			return nil, err
		}
		r.record(ctx, u.in.tel)
		return res, nil
	}, describeSpellOutcomeTable)
}

// resolve dispatches on the command and sets the table of r.
func (u ResolveSpellOutcomeTable) resolve(r *tableRoller, cmd ResolveSpellOutcomeTableCommand) (ResolveSpellOutcomeTableResult, error) {
	switch c := cmd.(type) {
	case RollTeleportOutcomeCommand:
		r.table = valueSpellTableTeleport
		return u.teleport(r, c)
	case RollConfusionBehaviorCommand:
		r.table = valueSpellTableConfusion
		return u.confusion(r, c)
	case RollPrismaticSprayCommand:
		r.table = valueSpellTablePrismaticSpray
		return u.spray(r, c)
	case ResolvePrismaticWallCrossingCommand:
		r.table = valueSpellTablePrismaticWall
		return u.crossWall(r, c)
	case RollReincarnationSpeciesCommand:
		r.table = valueSpellTableReincarnate
		return u.reincarnate(r, c)
	case RollBlinkTurnEndCommand:
		r.table = valueSpellTableBlink
		return u.blink(r, c)
	}
	return nil, invalidInput("spell outcome command " + typeName(cmd))
}

func (u ResolveSpellOutcomeTable) teleport(r *tableRoller, cmd RollTeleportOutcomeCommand) (ResolveSpellOutcomeTableResult, error) {
	creatures := teleportSubjectCount(cmd.Subject)
	if creatures == 0 || !cmd.Familiarity.Valid() {
		return nil, invalidInput("teleport needs a subject and a familiarity")
	}
	out := TeleportOutcome{familiarity: cmd.Familiarity, subject: cmd.Subject, arrival: ArrivesOnTarget{}}
	if !TeleportNeedsRoll(cmd.Familiarity) {
		return SpellTableTeleportRolled{out}, nil
	}
	for rerolls := 0; ; rerolls++ {
		face, err := r.face(100)
		if err != nil {
			return nil, err
		}
		kind, err := TeleportKindFor(cmd.Familiarity, face)
		if err != nil {
			return nil, err
		}
		r.outcomes = append(r.outcomes, kind.String())
		switch kind {
		case TeleportMishapOutcome:
			mishap := TeleportMishap{}
			for range creatures {
				amount, err := r.total(3, 10)
				if err != nil {
					return nil, err
				}
				d, err := NewDamage(amount, content.DamageForce)
				if err != nil {
					return nil, err
				}
				mishap.damage = append(mishap.damage, d)
			}
			out.mishaps = append(out.mishaps, mishap)
			if rerolls == maxRerolls {
				return nil, rerollLimit()
			}
			continue
		case TeleportOnTarget:
			// the arrival is already ArrivesOnTarget
		case TeleportSimilarArea:
			out.arrival = ArrivesInSimilarArea{}
		case TeleportOffTarget:
			miles, err := r.total(2, 12)
			if err != nil {
				return nil, err
			}
			direction, err := r.face(8)
			if err != nil {
				return nil, err
			}
			if out.arrival, err = NewArrivesOffTarget(miles, CompassDirection(direction)); err != nil {
				return nil, err
			}
		}
		return SpellTableTeleportRolled{out}, nil
	}
}

func (u ResolveSpellOutcomeTable) confusion(r *tableRoller, cmd RollConfusionBehaviorCommand) (ResolveSpellOutcomeTableResult, error) {
	if err := validateConfusionCandidates(cmd.CreaturesWithinReach); err != nil {
		return nil, err
	}
	face, err := r.face(10)
	if err != nil {
		return nil, err
	}
	row, err := ConfusionRowFor(face)
	if err != nil {
		return nil, err
	}
	r.outcomes = append(r.outcomes, row.String())
	turn := ConfusionTurn{row: row}
	switch row {
	case ConfusionRowMovesRandomly:
		d4, err := r.face(4)
		if err != nil {
			return nil, err
		}
		if turn.behavior, err = NewConfusionMovesRandomly([]CompassDirection{North, East, South, West}[d4-1]); err != nil {
			return nil, err
		}
	case ConfusionRowDoesNothing:
		turn.behavior = ConfusionDoesNothing{}
	case ConfusionRowAttacksRandomCreature:
		candidates := cmd.CreaturesWithinReach
		pick := 1
		switch len(candidates) {
		case 0:
			turn.behavior = ConfusionNoCreatureInReach{}
		case 1:
		default:
			if pick, err = r.face(len(candidates)); err != nil {
				return nil, err
			}
		}
		if len(candidates) > 0 {
			turn.behavior = ConfusionAttacksRandomCreature{candidates[pick-1]}
		}
	case ConfusionRowChoosesBehavior:
		turn.behavior = ConfusionTargetChoosesBehavior{}
	}
	return SpellTableConfusionRolled{turn}, nil
}

func (u ResolveSpellOutcomeTable) blink(r *tableRoller, cmd RollBlinkTurnEndCommand) (ResolveSpellOutcomeTableResult, error) {
	if !cmd.Location.Valid() {
		return nil, invalidInput("blink needs a location")
	}
	face, err := r.face(6)
	if err != nil {
		return nil, err
	}
	vanishes, err := BlinkVanishesOn(face)
	if err != nil {
		return nil, err
	}
	switch {
	case !vanishes:
		r.outcomes = append(r.outcomes, "stays")
		return SpellTableBlinkRolled{BlinkStaysOnPlane{cmd.Location}}, nil
	case cmd.Location == BlinkOnEtherealPlane:
		r.outcomes = append(r.outcomes, "vanishes")
		return SpellTableBlinkRolled{BlinkEndsAlreadyEthereal{BlinkReturnFor(BlinkOnEtherealPlane)}}, nil
	}
	r.outcomes = append(r.outcomes, "vanishes")
	return SpellTableBlinkRolled{BlinkVanishesToEtherealPlane{}}, nil
}

func (u ResolveSpellOutcomeTable) reincarnate(r *tableRoller, cmd RollReincarnationSpeciesCommand) (ResolveSpellOutcomeTableResult, error) {
	refusals, err := reincarnationRefusals(cmd.CreatureType, cmd.DeadFor)
	if err != nil {
		return nil, err
	}
	if cmd.ChosenSpecies != "" {
		if _, err := content.ParseSpeciesID(string(cmd.ChosenSpecies)); err != nil {
			return nil, invalidInput("chosen species is not a species id")
		}
	}
	switch {
	case len(refusals) > 0:
		return SpellTableRefused{refusals}, nil
	case cmd.ChosenSpecies != "":
		return SpellTableReincarnationRolled{ReincarnationSpeciesChosen{cmd.ChosenSpecies}}, nil
	}
	face, rerolls, err := r.chain(10, func(face int) (string, bool, error) {
		species, again, err := ReincarnationSpeciesFor(face)
		if again {
			return "roll_again", true, err
		}
		return string(species), false, err
	})
	if err != nil {
		return nil, err
	}
	species, _, err := ReincarnationSpeciesFor(face)
	if err != nil {
		return nil, err
	}
	return SpellTableReincarnationRolled{ReincarnationSpeciesRolled{species, rerolls}}, nil
}

// rayLook names a 1d8 face for the table counter and says to roll again for the special 8 when again is true.
func rayLook(again bool) func(int) (string, bool, error) {
	return func(face int) (string, bool, error) {
		color, special, err := PrismaticRayFor(face)
		if special {
			return "special", again, err
		}
		return color.String(), false, err
	}
}

// rays rolls the rays of one target: one for a 1 to 7, two for an 8, each of the two rerolled while it shows 8.
func (r *tableRoller) rays() ([]PrismaticColor, error) {
	face, _, err := r.chain(8, rayLook(false))
	if err != nil {
		return nil, err
	}
	color, special, err := PrismaticRayFor(face)
	if err != nil {
		return nil, err
	}
	if !special {
		return []PrismaticColor{color}, nil
	}
	var out []PrismaticColor
	for range 2 {
		if face, _, err = r.chain(8, rayLook(true)); err != nil {
			return nil, err
		}
		if color, _, err = PrismaticRayFor(face); err != nil {
			return nil, err
		}
		out = append(out, color)
	}
	return out, nil
}

func (u ResolveSpellOutcomeTable) spray(r *tableRoller, cmd RollPrismaticSprayCommand) (ResolveSpellOutcomeTableResult, error) {
	if len(cmd.Targets) > maxPrismaticTargets {
		return nil, ErrSpellTableLimit
	}
	seen := map[CombatantID]bool{}
	for _, t := range cmd.Targets {
		if !validPrismaticTarget(t) || seen[t.ID] {
			return nil, invalidInput("prismatic spray target needs a distinct id and a save attempt")
		}
		seen[t.ID] = true
	}
	results := make([]PrismaticTargetResult, len(cmd.Targets))
	damaging := false
	for i, t := range cmd.Targets {
		rays, err := r.rays()
		if err != nil {
			return nil, err
		}
		results[i] = PrismaticTargetResult{ID: t.ID, rays: rays}
		damaging = damaging || slices.ContainsFunc(rays, func(c PrismaticColor) bool { _, ok := c.DamageType(); return ok })
	}
	total := 0
	if damaging {
		var err error
		if total, err = r.total(12, 6); err != nil {
			return nil, err
		}
	}
	flow := attackFlow{r.roller}
	for i, t := range cmd.Targets {
		save, err := flow.resolveSave(t.Attempt, cmd.SaveDC)
		if err != nil {
			return nil, err
		}
		results[i].save = save
		for _, ray := range results[i].rays {
			strike, ok, err := PrismaticStrikeFor(ray, total, save.Outcome())
			if err != nil {
				return nil, err
			}
			if ok {
				results[i].strikes = append(results[i].strikes, strike)
			}
		}
	}
	return SpellTablePrismaticSprayRolled{results}, nil
}

func (u ResolveSpellOutcomeTable) crossWall(r *tableRoller, cmd ResolvePrismaticWallCrossingCommand) (ResolveSpellOutcomeTableResult, error) {
	if !validPrismaticTarget(cmd.Creature) {
		return nil, invalidInput("prismatic wall creature needs an id and a save attempt")
	}
	flow := attackFlow{r.roller}
	out := PrismaticWallCrossing{Creature: cmd.Creature.ID}
	for _, color := range cmd.Wall.Layers() {
		save, err := flow.resolveSave(cmd.Creature.Attempt, cmd.SaveDC)
		if err != nil {
			return nil, err
		}
		total := 0
		if _, damage := color.DamageType(); damage {
			if total, err = r.total(12, 6); err != nil {
				return nil, err
			}
		}
		layer := PrismaticLayerCrossing{color: color, save: save}
		strike, ok, err := PrismaticStrikeFor(color, total, save.Outcome())
		if err != nil {
			return nil, err
		}
		if ok {
			layer.strike = strike
		}
		out.layers = append(out.layers, layer)
	}
	return SpellTablePrismaticWallCrossed{out}, nil
}

// describeSpellOutcomeTable classifies a result for telemetry: every rolled or crossed table is applied, a refused
// Reincarnate is refused, and the table is the extra.
func describeSpellOutcomeTable(res ResolveSpellOutcomeTableResult) description {
	name, table, class := valueResultSpellTableRolled, "", resultApplied
	switch res.(type) {
	case SpellTableTeleportRolled:
		table = valueSpellTableTeleport
	case SpellTableConfusionRolled:
		table = valueSpellTableConfusion
	case SpellTablePrismaticSprayRolled:
		table = valueSpellTablePrismaticSpray
	case SpellTablePrismaticWallCrossed:
		table = valueSpellTablePrismaticWall
	case SpellTableReincarnationRolled:
		table = valueSpellTableReincarnate
	case SpellTableBlinkRolled:
		table = valueSpellTableBlink
	case SpellTableRefused:
		name, table, class = valueResultSpellTableRefused, valueSpellTableReincarnate, resultRefused
	default:
		panic(unreachableError("spell outcome table result", res))
	}
	return description{Class: class, Name: name, Extras: []telemetry.Attr{telemetry.String(attrSpellTable, table)}}
}
