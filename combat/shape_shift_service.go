package combat

// ShapeShiftSpells is the stateless domain service of Polymorph, True Polymorph, Shapechange
// and Animal Shapes once they are cast (shape-changing-spells.md). Its operations are Begin,
// ChangeForm, Assess, ElapseFullDuration, End, RecordTemporaryHitPointsReplaced,
// StatisticsOf, RestrictionsOf and Eligibility. The zero value is ready to use. Every
// operation is pure and safe for concurrent use. The name ShapeShifting belongs to the
// Shape-Shifting rule of the magical effects.
type ShapeShiftSpells struct{}
