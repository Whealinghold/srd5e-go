# Telepathy

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md` |
| Consumed by | `combat-action-resolution.md` (amendment RA-1: the `ChangeTelepathicContact` use case) |

Split from `magical-effects.md` (its FR-071 to FR-090, withdrawn there in favour of
FR-001 to FR-020 here). Rank 23 (`shared-types-and-ownership.md` 4.11): right of
`conditions` (22), so it imports only `combat-foundations.md` (`CombatantID`,
`combat.Event`) and `conditions.md` (`ConditionSet`, `ConditionSet.IsIncapacitated`). It has no
port and returns no `*srderr.PortError`, so `errors-and-results.md` and
`observability-ports.md` are not dependencies. It supersedes no numbered
requirement of another spec.

## 1. Purpose and scope

The Rules Glossary "Telepathy" entry (SRD 5.2.1, p. 190) as pure rules: who may
start a telepathic contact, what breaks it and who may reply. The host holds the
`TelepathicContactState` beside other per-creature values and supplies distances,
Incapacitated status (through `ConditionSet`) and what the contacted creature
understands.

**Out of scope:** interpreting a monster's printed telepathy qualifier text
(`content-monsters.md` FR-029), which the host maps to the typed `ReplyPolicy` of
FR-023 (Q-1); the spells that grant telepathy;
the mental conversation's content; use cases and telemetry (`combat-foundations.md`
FR-029 to FR-035, section 7).

## 2. Source

- SRD 5.2.1, "Telepathy", p. 190; "Incapacitated [Condition]", p. 184.
- Rule files: `doc/rules/architecture.md` §3, §6; `doc/rules/testing.md` §4, §5;
  `doc/rules/observability.md` §2, §3.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Telepathy | `TelepathyCapability` (`NewTelepathic(rangeFeet) (TelepathyCapability, error)`, `MustTelepathic`; the zero value is `NotTelepathic`), `ReplyPolicy` (typed enum `ContactedMayReply` = zero value and default, `ContactedMayNotReply`), `TelepathicContactState` (comparable struct with unexported fields; the zero value is `NoContact`; `InContact(telepath, contacted, rangeFeet, replyPolicy)` builds a contact), `ContactedUnderstanding` (typed enum `UnderstandsALanguage`, `IsTelepathic`, `UnderstandsNeither`), `TelepathicContactRejection` (typed enum with `String()`), `BreakCause` (typed enum `BreakCauseIncapacitated`, `BreakCauseContactedLeftRange`, `BreakCauseTelepathContactedAnother`) | At most one contact per telepath. |
| Incapacitated | `ConditionSet.IsIncapacitated()` from `conditions.md` | Reused. |

Results are operation-prefixed (`shared-types-and-ownership.md` FR-017):
`InitiateTelepathicContactMade`, `...Rejected(TelepathicContactRejection)`,
`...AlreadyMade`; `EndTelepathicContactApplied`, `...NoContact`;
`ReportTelepathIncapacitatedBroken`, `...Unchanged`; `ReportTelepathDistanceBroken`,
`...Unchanged`; `TelepathicReplyAllowed`, `TelepathicReplyNotPermitted`, `TelepathicReplyNoContact`,
`TelepathicReplyNotContacted`. An
`...Unchanged` answer is an applied answer.

## 4. Requirements

Ids are stable. Case-table rows are one test each, citing the FR in the row.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `TelepathyCapability` MUST be built by `NewTelepathic(rangeFeet)` or be the zero value `NotTelepathic`; a range below 1 MUST be rejected: `NewTelepathic` returns an error that is `errors.Is` `ErrTelepathyRange` and `srderr.ErrInvalidInput`, and `MustTelepathic` panics with that error (package decision, Q-2). | SRD 5.2.1, "Telepathy" ("within a specified range"), p. 190 | unhappy (invalid input) |
| FR-002 | `InitiateTelepathicContact` by a `Telepathic` initiator with a contacted creature within range, neither Incapacitated, and the contacted creature understanding at least one language or being telepathic MUST return `InitiateTelepathicContactMade` with `InContact` and one `TelepathicContactMade` event. | SRD 5.2.1, "Telepathy", p. 190 | happy |
| FR-003 | A contacted creature exactly at the telepath's range MUST be reachable; one foot beyond MUST be rejected with reason `ContactedOutOfRange` (package reading of "within": distance at most the range, Q-3). | SRD 5.2.1, "Telepathy", p. 190 | unhappy (boundary) |
| FR-004 | The contacted creature need not share a language with the telepath: contact MUST succeed when it `UnderstandsALanguage` whatever that language is (the command carries no shared-language fact). | SRD 5.2.1, "Telepathy" ("doesn't need to share a language"), p. 190 | unhappy (rule interaction) |
| FR-005 | A contacted creature that neither understands a language nor is telepathic (`UnderstandsNeither`) MUST be rejected with reason `ContactedCannotUnderstand`. | SRD 5.2.1, "Telepathy" ("must be able to understand at least one language or be telepathic itself"), p. 190 | unhappy (state error) |
| FR-006 | A telepathic contacted creature that understands no language (`IsTelepathic`) MUST NOT be rejected for that reason. | SRD 5.2.1, "Telepathy", p. 190 | unhappy (boundary) |
| FR-007 | Initiation MUST NOT depend on sight: the command carries no sight fact, and a contacted creature the telepath cannot see MUST be contactable. | SRD 5.2.1, "Telepathy" ("doesn't need to see a contacted creature"), p. 190 | unhappy (rule interaction) |
| FR-008 | An initiator that is `NotTelepathic` MUST be rejected with reason `InitiatorNotTelepathic`. | SRD 5.2.1, "Telepathy" ("A creature without telepathy ... can't initiate a telepathic conversation"), p. 190 | unhappy (state error) |
| FR-009 | An initiator whose `ConditionSet` has Incapacitated effective (including through Paralyzed, Stunned, Petrified or Unconscious, `conditions.md` FR-013) MUST be rejected with reason `InitiatorIncapacitated`. | SRD 5.2.1, "Telepathy" ("Telepathic contact can't be initiated ... if either ... has the Incapacitated condition"), p. 190; "Incapacitated", p. 184 | unhappy (state error) |
| FR-010 | A contacted creature with Incapacitated effective MUST be rejected with reason `ContactedIncapacitated`. | SRD 5.2.1, "Telepathy", p. 190 | unhappy (state error) |
| FR-011 | When several rejections apply, the reason MUST be the first of: `InitiatorNotTelepathic`, `InitiatorIncapacitated`, `ContactedIncapacitated`, `ContactedOutOfRange`, `ContactedCannotUnderstand`; the state is unchanged (case table 4.1). | Package design (`doc/rules/testing.md` §4, determinism) | unhappy (rule interaction) |
| FR-012 | Starting and ending telepathic contact MUST take no action: the events of `InitiateTelepathicContact` and `EndTelepathicContact` results are exactly the listed `TelepathicContact...` events, with no action-spending event, and the telepathy files MUST NOT import or name `ActionChoice`, `StandardAction` or `ReactionTaken` (name scan in `internal/archtest`, FR-021). | SRD 5.2.1, "Telepathy" ("start or end the telepathic contact at any time (no action required)"), p. 190 | happy |
| FR-013 | `EndTelepathicContact` by the telepath while `InContact` MUST return `EndTelepathicContactApplied` with `NoContact` and one `TelepathicContactEnded` event. | SRD 5.2.1, "Telepathy", p. 190 | happy |
| FR-014 | `EndTelepathicContact` while `NoContact` MUST return `EndTelepathicContactNoContact`, the state unchanged and no events. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-015 | `ReportTelepathIncapacitated` for either creature of an `InContact` pair MUST return `ReportTelepathIncapacitatedBroken` with `NoContact` and one `TelepathicContactBroken` with cause `BreakCauseIncapacitated`; for a creature not in the pair, or with `NoContact`, it MUST return `ReportTelepathIncapacitatedUnchanged`, state unchanged, no events. | SRD 5.2.1, "Telepathy" ("immediately broken if either ... has the Incapacitated condition"), p. 190 | unhappy (rule interaction) |
| FR-016 | `ReportTelepathDistance` MUST return `ReportTelepathDistanceBroken` (cause `BreakCauseContactedLeftRange`) when the contacted creature is beyond the range, and `ReportTelepathDistanceUnchanged` (no events) when the distance is at most the range or there is no contact. | SRD 5.2.1, "Telepathy" ("broken if the contacted creature is no longer within the telepathy's range"), p. 190 | unhappy (boundary) |
| FR-017 | Initiating contact with a different creature while `InContact` MUST break the old contact (cause `BreakCauseTelepathContactedAnother`) and make the new one: events in that order, state `InContact` with the new creature. | SRD 5.2.1, "Telepathy" ("broken ... if the telepath contacts a different creature"), p. 190 | unhappy (rule interaction) |
| FR-018 | Initiating contact with the creature already in contact MUST return `InitiateTelepathicContactAlreadyMade`, state unchanged, no events. | `doc/rules/testing.md` §4 (repeated action) | unhappy (state error) |
| FR-019 | While `InContact` with `ContactedMayReply` (the default), the contacted creature MAY reply, whether or not it is telepathic: `TelepathicReply` MUST return `TelepathicReplyAllowed`. | SRD 5.2.1, "Telepathy" ("the non-telepath can communicate mentally to the telepath until the telepathic connection ends"), p. 190 | happy |
| FR-020 | A reply while `NoContact` MUST return `TelepathicReplyNoContact`, whose result name is `contact_none`, shared with `EndTelepathicContactNoContact` (section 7). A reply by a creature that is not the contacted creature of an `InContact` pair MUST return the distinct `TelepathicReplyNotContacted` (result name `contact_reply_not_contacted`), so a wrong-party call is not mistaken for missing state. | SRD 5.2.1, "Telepathy", p. 190 | unhappy (state error) |
| FR-021 | Layering, events and telemetry follow `combat-foundations.md` FR-015 to FR-019, FR-026 and FR-029 to FR-035: types live in package `combat`, in files named `telepathy*.go` (no use-case file), are pure, import only the kernel packages and the published language of `conditions.md`; events form a closed variant family `TelepathyEvent` (an interface embedding `combat.Event` with an unexported marker method, documented with its three implementers) with unique constant `Kind()` values and `Combatants()` as in table 4.2; the RA-1 use case is `ChangeTelepathicContact` (span suffix `change_telepathic_contact`). | `doc/rules/architecture.md` §1, §2, §9; `shared-types-and-ownership.md` FR-019, FR-044 | unhappy (forbidden edge) |
| FR-022 | Every operation MUST NOT alter its inputs, MUST return defensive copies of slices and MUST give equal results for equal inputs. | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |
| FR-023 | `InitiateTelepathicContact` MUST take an optional `ReplyPolicy` (default `ContactedMayReply`) stored in `InContact`; with `ContactedMayNotReply`, `TelepathicReply` by the contacted creature MUST return `TelepathicReplyNotPermitted` (result name `contact_reply_not_permitted`, refused). The policy changes nothing else: initiation, ending and breaking are unaffected, and the policy takes part in `InContact` equality. The library does not read qualifier text: the host maps it (for example the Otyugh's "doesn't allow the receiving creature to respond telepathically") to the policy. | SRD 5.2.1, "Telepathy", p. 190 (default rule); "Otyugh", p. 313 (monster exception) | unhappy (rule interaction) |

#### Case table 4.1: initiation (FR-002 to FR-011)

Baseline: initiator `NewTelepathic(60)`, distance 30, neither Incapacitated, contacted `UnderstandsALanguage`. Source: SRD 5.2.1, "Telepathy", p. 190.

| Case | Change from baseline | Expected | FR | Path |
|---|---|---|---|---|
| baseline | none | `Made` | FR-002 | happy |
| at range | distance 60 | `Made` | FR-003 | unhappy (boundary) |
| one beyond | distance 61 | rejected `ContactedOutOfRange` | FR-003 | unhappy (boundary) |
| unseen contacted | (no sight fact exists) | `Made` | FR-007 | unhappy (rule interaction) |
| no shared language | contacted understands another language | `Made` | FR-004 | unhappy (rule interaction) |
| telepathic, no language | `IsTelepathic` | `Made` | FR-006 | unhappy (boundary) |
| neither | `UnderstandsNeither` | rejected `ContactedCannotUnderstand` | FR-005 | unhappy (state error) |
| initiator not telepathic | `NotTelepathic` | rejected `InitiatorNotTelepathic` | FR-008 | unhappy (state error) |
| initiator Incapacitated | `ConditionSet` {Stunned} | rejected `InitiatorIncapacitated` | FR-009 | unhappy (state error) |
| contacted Incapacitated | `ConditionSet` {Unconscious} | rejected `ContactedIncapacitated` | FR-010 | unhappy (state error) |
| everything wrong | not telepathic, Incapacitated both, 61, neither | rejected `InitiatorNotTelepathic` | FR-011 | unhappy (rule interaction) |
| both Incapacitated, out of range | initiator and contacted Incapacitated, 61 | rejected `InitiatorIncapacitated` | FR-011 | unhappy (rule interaction) |
| out of range and cannot understand | 61, neither | rejected `ContactedOutOfRange` | FR-011 | unhappy (rule interaction) |

#### Table 4.2: event kinds (FR-021)

| Event | `Kind()` | `Combatants()` order |
|---|---|---|
| `TelepathicContactMade` | `telepathic_contact_made` | the telepath, then the contacted creature |
| `TelepathicContactBroken` | `telepathic_contact_broken` | the telepath, then the contacted creature |
| `TelepathicContactEnded` | `telepathic_contact_ended` | the telepath, then the contacted creature |

## 5. Model

- **Referenced, not defined here:** `CombatantID`, `combat.Event`
  (`combat-foundations.md`); `ConditionSet`, `ConditionSet.IsIncapacitated` (`conditions.md`).
- **Value objects:** the types of section 3. **Events:** the closed variant family
  `TelepathyEvent` (table 4.2). **Domain service** (stateless):
  the package functions `InitiateTelepathicContact` (with `ReplyPolicy`), `EndTelepathicContact`, `ReportTelepathIncapacitated`, `ReportTelepathDistance`, `TelepathicReply`.
- **Invariants** (constructor): FR-001. Everything else is a closed result
  (`doc/rules/architecture.md` §6); no `*srderr.PortError`, no port.

## 6. Interface

- No use case, port or I/O: synchronous pure functions over host-held values.
- Package `combat` exports the types of section 3; the handoff type is `TelepathicContactState`. No test doubles. Each operation is a pure package function (`InitiateTelepathicContact`, `EndTelepathicContact`, `ReportTelepathIncapacitated`, `ReportTelepathDistance`, `TelepathicReply`) taking and returning values; no `context.Context` (pure CPU) and no `error` return, because no input is untrusted except the range (FR-001, handled by `NewTelepathic`).
- Results are interfaces with an unexported marker method, switched exhaustively (`shared-types-and-ownership.md` FR-017; `errors-and-results.md` FR-018); the doc comment of each lists every variant.

## 7. Observability

Silent by design (`doc/rules/observability.md` §3); no signal of its own. The
RA-1 use case `ChangeTelepathicContact` (`change_telepathic_contact`) records
through `CombatTelemetry`. Classification (`combat-foundations.md` FR-030, FR-031):

| Variant | `srd5e.combat.result` | Class |
|---|---|---|
| `InitiateTelepathicContactMade`, `EndTelepathicContactApplied`, `ReportTelepathIncapacitatedBroken`, `...Unchanged`, `ReportTelepathDistanceBroken`, `...Unchanged`, `TelepathicReplyAllowed` | `completed` | applied |
| `InitiateTelepathicContactRejected(r)`, `r` one of `InitiatorNotTelepathic`, `InitiatorIncapacitated`, `ContactedIncapacitated`, `ContactedOutOfRange`, `ContactedCannotUnderstand` | `contact_initiator_not_telepathic`, `contact_initiator_incapacitated`, `contact_contacted_incapacitated`, `contact_out_of_range`, `contact_cannot_understand` respectively | refused |
| `InitiateTelepathicContactAlreadyMade` | `contact_already_made` | refused |
| `TelepathicReplyNotPermitted` | `contact_reply_not_permitted` | refused |
| `TelepathicReplyNotContacted` | `contact_reply_not_contacted` | refused |
| `EndTelepathicContactNoContact`, `TelepathicReplyNoContact` | `contact_none` | refused |

Seven questions: (1) the host passes a wrong distance or condition set, or calls in
the wrong state (`refused`); a range below 1 is rejected by `NewTelepathic`. (2) By
`srd5e.combat.operations` by result once RA-1 lands. (3) Results carry the
rejection reason and the break cause. (4) Each call by operation and result.
(5) Existing instrumentation is unchanged. (6) Sustained `contact_none` or
`contact_out_of_range` means host state drift. (7) Nothing is noisy: no log.

## 8. Test plan

Unit tests sit beside the code (`combat/telepathy*_test.go`, package `combat_test`); architecture checks live in `internal/archtest`.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-011 | unit | Table 4.1, one test per row, with a real `ConditionSet` for Incapacitated |
| FR-012 | unit, architecture | Initiate and end results carry exactly their listed events; the files name no `ActionChoice`, `StandardAction` or `ReactionTaken` |
| FR-013 to FR-020, FR-023 | unit | End, break by Incapacitated and by distance (at range and one beyond), different creature, repeated contact, reply, reply by a non-contacted creature, reply under each `ReplyPolicy` (`ContactedMayNotReply` refuses only the reply) |
| FR-021 | architecture, unit | Import direction; table 4.2 kinds unique |
| FR-022 | unit | Inputs unchanged; equal results for equal inputs |

No doubles or contract suites. Mutation testing (`doc/rules/testing.md` §8): all
changed lines; boundary rows exist for FR-003 and FR-016.

## 9. Open questions and decisions

| Id | Question | Recommendation | Status |
|---|---|---|---|
| Q-1 | A monster's telepathy qualifier (`content-monsters.md` FR-029) may override FR-019. | `ReplyPolicy` input (FR-023). | settled |

Settled, one line each: Q-1 the SRD prints a reply exception on a stat block
(Otyugh, p. 313, "doesn't allow the receiving creature to respond telepathically"),
so it is modelled now as a typed `ReplyPolicy` (FR-023) rather than parsed from
text; the other printed qualifier (Dretch, p. 281, "works only with creatures that
understand Abyssal") needs no input, because the host supplies
`ContactedUnderstanding` (`neither` for a creature that does not understand Abyssal;
interpretation: a qualifier restricting the audience is the same fact as FR-005);
Q-2 ranges are whole feet, minimum 1 (FR-001); Q-3 "within" is inclusive
(FR-003, FR-016).

## 10. Changelog impact

- **Added:** the Telepathy rules of the Rules Glossary in `combat`
  (`doc/specs/telepathy.md`).
- **BREAKING:** no. Everything is new.
