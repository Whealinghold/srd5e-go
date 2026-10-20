package content

// The wire parsers of the closed sets of the spell effect model
// (spell-effect-data.md FR-030, FR-031). Each returns the member spelled
// exactly s, as String gives it (lower camel case): matching is case-sensitive
// and does not trim. Each returns a *[ParseError] wrapping
// srderr.ErrInvalidInput for any other string and never panics.

// ParseEffectCoverage returns the coverage spelled exactly s.
func ParseEffectCoverage(s string) (EffectCoverage, error) {
	v, err := parseEnum("EffectCoverage", effectCoverageNames(), s)
	return EffectCoverage(v), err
}

// ParseTargetSubject returns the target subject spelled exactly s.
func ParseTargetSubject(s string) (TargetSubject, error) {
	v, err := parseEnum("TargetSubject", targetSubjectNames(), s)
	return TargetSubject(v), err
}

// ParseSelfTargetPolicy returns the self-target policy spelled exactly s.
func ParseSelfTargetPolicy(s string) (SelfTargetPolicy, error) {
	v, err := parseEnum("SelfTargetPolicy", selfTargetPolicyNames(), s)
	return SelfTargetPolicy(v), err
}

// ParseSpellAttackKind returns the spell attack kind spelled exactly s.
func ParseSpellAttackKind(s string) (SpellAttackKind, error) {
	v, err := parseEnum("SpellAttackKind", spellAttackKindNames(), s)
	return SpellAttackKind(v), err
}

// ParseSuccessEffect returns the success effect spelled exactly s.
func ParseSuccessEffect(s string) (SuccessEffect, error) {
	v, err := parseEnum("SuccessEffect", successEffectNames(), s)
	return SuccessEffect(v), err
}

// ParseModifierAddition returns the modifier addition spelled exactly s.
func ParseModifierAddition(s string) (ModifierAddition, error) {
	v, err := parseEnum("ModifierAddition", modifierAdditionNames(), s)
	return ModifierAddition(v), err
}

// ParseTurnOwner returns the turn owner spelled exactly s.
func ParseTurnOwner(s string) (TurnOwner, error) {
	v, err := parseEnum("TurnOwner", turnOwnerNames(), s)
	return TurnOwner(v), err
}

// ParseTurnBoundary returns the turn boundary spelled exactly s.
func ParseTurnBoundary(s string) (TurnBoundary, error) {
	v, err := parseEnum("TurnBoundary", turnBoundaryNames(), s)
	return TurnBoundary(v), err
}

// ParseZoneTrigger returns the zone trigger spelled exactly s.
func ParseZoneTrigger(s string) (ZoneTrigger, error) {
	v, err := parseEnum("ZoneTrigger", zoneTriggerNames(), s)
	return ZoneTrigger(v), err
}

// ParseZoneTargetScope returns the zone target scope spelled exactly s.
func ParseZoneTargetScope(s string) (ZoneTargetScope, error) {
	v, err := parseEnum("ZoneTargetScope", zoneTargetScopeNames(), s)
	return ZoneTargetScope(v), err
}

// ParseZoneActionKind returns the zone action kind spelled exactly s.
func ParseZoneActionKind(s string) (ZoneActionKind, error) {
	v, err := parseEnum("ZoneActionKind", zoneActionKindNames(), s)
	return ZoneActionKind(v), err
}

// ParseDriftDirectionRule returns the drift direction rule spelled exactly s.
func ParseDriftDirectionRule(s string) (DriftDirectionRule, error) {
	v, err := parseEnum("DriftDirectionRule", driftDirectionRuleNames(), s)
	return DriftDirectionRule(v), err
}

// ParseDesignationPolicy returns the designation policy spelled exactly s.
func ParseDesignationPolicy(s string) (DesignationPolicy, error) {
	v, err := parseEnum("DesignationPolicy", designationPolicyNames(), s)
	return DesignationPolicy(v), err
}
