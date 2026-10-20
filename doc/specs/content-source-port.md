# content.Source port

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | 2 |
| Depends on | `errors-and-results.md`, `content-schema-and-decoder.md` |

Binding decisions: [shared-types-and-ownership.md](shared-types-and-ownership.md) (rank 0, FR-056): FR-005
and FR-046 (vocabulary and typed ids owned by the decoder spec), FR-064 (the identity types `content.ID`,
`content.IDProblem`, `content.Key`, `content.Kind`, `content.Entry` and `content.MissingContent` are owned by the decoder
spec; this spec depends on it), FR-007 (id format), FR-013 and FR-014
(dependency and phase rules), FR-017 and FR-018 (result naming, `MissingContent`), FR-030 to FR-033
(bundled delivery), FR-034 to FR-039 (schema version, ports, doubles, contract suites), FR-041
(`error.type` values), FR-044 (no use case calls a use case), FR-047 (`BundledSource` owner),
FR-048 (topic spec owns mechanics), FR-068 to FR-072 (Go vocabulary). [errors-and-results.md](errors-and-results.md) supplies
`srderr.PortError`, `srderr.WrapPort`, `srderr.GuardValue` and the result and port
conventions.

Consumers (never listed in the header): the bundled-data specs (each adds its embedded library, FR-064) and
the `character`, `combat` and `toolbox` use cases. `content` has no upstream context (architecture.md §2),
so package `content` imports kernel packages (`srderr`, and the other kernel packages it needs) only; the
bundled adapter lives in the sub-package `content/data` and the double and contract suite in `content/contenttest`.

Direction with [content-schema-and-decoder.md](content-schema-and-decoder.md) (rank 4, lower, listed in the
header): that spec owns the identity types this spec uses (`Kind`, `ID`, `Key`,
`Entry`, `MissingContent`; its FR-142 to FR-148) and the typed ids with their kind mapping (its FR-114).
The port, `LookupResult`, `Query` and `Catalogue` use them; none is defined here. Only the
`data.BundledSource` adapter (FR-059 to FR-064) needs `content.Decoder`, `content.Set`, `content.ValidateSet`
and the embedded document index (its FR-104, FR-114, FR-122, FR-125 to FR-127, FR-129), because
shared-types-and-ownership.md FR-047 fixes the adapter's owner here. The adapter sits in `content/data`; the domain
types and the port never import the decoder or `embed`.

## 1. Purpose and scope

Defines how the rest of the package asks "give me the content entry with this key" without knowing where
content lives. It specifies:

- the `content.Source` driven port (context-aware, synchronous call), keyed by the `content.Key` of `content-schema-and-decoder.md`
  (the identity types and typed ids are owned there, shared-types-and-ownership.md FR-046, FR-064);
- the closed `content.Query` (FR-057) and the closed `content.LookupResult` (`LookupFound` | `LookupNotFound`, which carries
  the `MissingContent` of the schema spec, shared-types-and-ownership.md FR-018);
- `content.Catalogue`, a pure in-memory index every in-process source delegates to, so duplicate-key, lookup
  and ordering rules are written and tested once;
- `content.SourceError` (the typed cause inside a `*srderr.PortError`), `content.InMemorySource` and
  `data.BundledSource` with `data.BundledInvalidError` (FR-059 to FR-064);
- the exported contract suites and `contenttest.FakeSource` (package `content/contenttest`: shared-types-and-ownership.md
  FR-038, FR-039; FR-027, FR-058).

**Out of scope:** the shape of individual entries, every closed vocabulary type and the identity types
(`ID`, `Key`, `Kind`, `Entry`, `MissingContent`; `content-schema-and-decoder.md`,
shared-types-and-ownership.md FR-046, FR-064); JSON decoding, the envelope and
`schemaVersion` (FR-034, FR-035 there); bundled SRD data, `go:embed` wiring and `NOTICE.md` coverage
(shared-types-and-ownership.md FR-030 to FR-033; the bundled-data specs); file or network sources;
search, filtering and pagination beyond "all entries of a kind"; reload, caching or mutation after
construction; "unsupported content version" (a decoder failure, section 9); use cases that call the port and
their telemetry. No SRD game rule is implemented here.

## 2. Source

n/a (package design); the SRD 5.2.1 category headings behind `Kind` are cited by
content-schema-and-decoder.md FR-142. Derived from `doc/rules/architecture.md` §1 to §7 and §9,
`doc/rules/go.md`, `doc/rules/observability.md` §2, §3, §7, `doc/rules/testing.md` §3, §4, §5, §8, §10,
`shared-types-and-ownership.md`, `errors-and-results.md` and `doc/roadmap.md` Phase 2.

## 3. Vocabulary

| Term | Type | Notes |
|---|---|---|
| Identity types | `content.Kind`, `content.ID`, `content.IDProblem`, `content.Key`, `content.Entry`, `content.MissingContent` | **Owned by content-schema-and-decoder.md** (FR-142 to FR-148; shared-types-and-ownership.md FR-064) and used here, not defined. The typed ids and the typed id to `Kind` mapping (`id.Key()`) are its FR-114 and section 3 table. |
| `Source` | driven port (interface in `content`) | Two methods, each taking `context.Context` first and returning an `error` for infrastructure failure (it may do I/O). Named for what the domain needs, not the technology. |
| `Query` | closed interface (value) | What was asked of a source: `LookupQuery{Key}` or `ListKindQuery{Kind}`. Pure value; used by `SourceError` and by the double's recording. |
| `LookupResult` | closed interface (value) | `LookupFound{Entry}` carries the `Entry`; `LookupNotFound{Key}` carries the requested `Key`. Not generic. Variants carry the operation prefix `Lookup` (shared-types-and-ownership.md FR-017); the bare names `Found` and `NotFound` are never exported. |
| `Catalogue` | value object | Immutable, validated index of entries: rejects duplicate keys, answers `Lookup(key)` as a `LookupResult` and `OfKind(kind)` as an ordered copy. Pure and synchronous. |
| `SourceError` | struct implementing `error` | Infrastructure failure in a source; the typed cause a source returns and `srderr.WrapPort` wraps. Carries one non-nil `Query` and the original `Err`; `Unwrap` returns `Err`. There is no separate operation enum: the `Query` variant is the operation. |
| `InMemorySource` | adapter (`content`) | Thin wrapper over a `Catalogue` implementing `Source`. Lives beside the port (shared-types-and-ownership.md FR-037). |
| `data.BundledSource` | adapter (`content/data`) | Serves the embedded SRD libraries: decodes them through `content.Decoder`, validates the set with `content.ValidateSet`, delegates to an `InMemorySource`. The only package allowed to import `embed`. Owner: this spec (shared-types-and-ownership.md FR-047). |
| `data.BundledInvalidError` | error type (`content/data`) | Returned by `data.NewBundledSource` when the embedded documents fail to decode or the merged set fails whole-set validation; carries the decoder's `Problem`s and the `ReferenceProblem`s, in two slices, at least one non-empty. |
| `contenttest.RunSourceContract` | function (`content/contenttest`) | The shared contract suite, for sources that can be built from arbitrary entries. Exported (shared-types-and-ownership.md FR-039). |
| `contenttest.RunReadOnlyContract` | function (`content/contenttest`) | The read-only variant, for sources backed by a fixed fixture. Exported. |
| `contenttest.FakeSource` | test double (`content/contenttest`) | Wraps a `Catalogue`; scriptable failure and recorded queries (`Query`). A double, not an adapter. |

### Kind mapping

The mapping from each top-level typed id to its `Kind` (so `source.Lookup(ctx, id.Key())` works, and
`SubclassID` and `FeatureID` have no kind) and the table of `Kind` values against the SRD headings are
owned by content-schema-and-decoder.md (section 3 and FR-114, FR-142). This spec relies on them and repeats
neither.

## 4. Requirements

Port failure contract (errors-and-results.md section 5): expected absence is the `LookupNotFound` value and an
empty list, never an error; infrastructure failure is the method's `error`, which the caller wraps once as
`*srderr.PortError{Port: content.Port, Op: "lookup" | "list_kind"}`. `content.Port` is `"content"` and
`srderr.ErrorType` of that failure is `content_error`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `Source.Lookup(ctx, Key)` MUST return `(LookupResult, error)`, synchronously, with `context.Context` as its first parameter (it may do I/O; shared-types-and-ownership.md FR-071). It MUST NOT return a channel, a callback or a nil `LookupResult` together with a nil error. | architecture.md §4 (sync or async) | happy |
| FR-008 | `Lookup` MUST return `LookupFound` holding the entry whose key equals the requested key when such an entry exists. | architecture.md §4 (failure contract) | happy |
| FR-009 | `Lookup` MUST return `LookupNotFound` carrying the requested `Key` and a nil error when no entry has that key; it MUST NOT return an error and MUST NOT panic. | architecture.md §4, §6 | unhappy (missing content) |
| FR-010 | `Lookup` MUST NOT return an entry of a different kind for a key whose id exists only under another kind. | architecture.md §4 | unhappy (rule interaction) |
| FR-011 | `Source.ListKind(ctx, Kind)` MUST return every entry of that kind, ordered by `ID` using ordinal (byte, `cmp.Compare` on the string) comparison, so results are deterministic. | architecture.md §3 (determinism) | happy |
| FR-012 | `ListKind` MUST return an empty slice and a nil error, not an error or `LookupNotFound`, when the source has no entries of that kind. | architecture.md §4 | unhappy (empty case) |
| FR-013 | The slice returned by `ListKind` MUST be a copy: mutating it MUST NOT alter the source or a later result. | architecture.md §4 (handoffs) | unhappy (immutability) |
| FR-014 | Repeated calls to `Lookup` and `ListKind` on an unchanged source MUST return equal results: the same `LookupFound` or `LookupNotFound` variant with an entry or key that is `==` (or `Equal` where the entry type is not comparable), and slices of the same length whose elements are pairwise equal, in the same order. This applies to `InMemorySource` and to `FakeSource` given the same entries and script (no clock, randomness or I/O). | testing.md §4 (determinism); architecture.md §3 | happy |
| FR-015 | `LookupResult` MUST be a closed interface with an unexported marker method and exactly the variants `LookupFound` and `LookupNotFound`, listed in its doc comment, so a type switch over it is exhaustive without a default (errors-and-results.md FR-044, FR-045). | architecture.md §3, §4 | happy |
| FR-016 | `LookupFound` MUST expose its `Entry` and `LookupNotFound` MUST expose the requested `Key` and a `Missing()` method equal to the `MissingContent` of that key (content-schema-and-decoder.md FR-147); both MUST be immutable value types (value receivers) and compare by value (two `LookupFound`s are equal iff their entries are equal; two `LookupNotFound`s iff their keys are equal). `LookupResult` is not generic. | architecture.md §3, §4 | happy |
| FR-017 | A source whose backend fails MUST return the failure as the `error` of `Lookup` or `ListKind`, with the zero result; it MUST NOT return `LookupNotFound` or an empty slice for an infrastructure failure, and MUST NOT panic: the failure is returned. Callers MUST wrap each call with `srderr.GuardValue(content.Port, op, ...)` (or `srderr.WrapPort`), which also turns a panic of a third-party source into a `*srderr.PortError` (errors-and-results.md FR-009). | architecture.md §4 (failure contract); errors-and-results.md FR-016, FR-032 | unhappy (collaborator failure) |
| FR-018 | `SourceError` MUST carry one non-nil `Query` (`LookupQuery{Key}` for a `Lookup`, `ListKindQuery{Kind}` for a `ListKind`) and the original error as `Err`, and MUST implement `error` and `Unwrap() error` returning `Err` unchanged, so `errors.Is` and `errors.As` reach the backend error. It has no nullable key or kind field and no separate operation field; a type switch over `Query` is exhaustive. A source returns a `*SourceError`; the caller wraps it as a `*srderr.PortError`, so `errors.As` finds both. | architecture.md §4, §6 | unhappy (collaborator failure) |
| FR-019 | `SourceError`, `LookupResult`, `LookupFound` and `LookupNotFound` MUST carry structured data only: no exported field or method named `Message`, `Description`, `Text` or `DisplayText` or holding pre-formatted display text. `Error()` is for logs and no test depends on it (errors-and-results.md FR-007, FR-020). | architecture.md §6; errors-and-results.md FR-007, FR-020 | unhappy (invalid design) |
| FR-020 | `InMemorySource` MUST construct from a slice of `Entry`, delegate to a `Catalogue` (FR-046), and answer `Lookup` and `ListKind` per FR-008 to FR-014. | architecture.md §5 | happy |
| FR-021 | A `Catalogue`, and therefore `InMemorySource` and `FakeSource`, MUST reject entries with duplicate `Key`s at construction: `NewCatalogue` returns a `*DuplicateKeyError` carrying the duplicate `Key` that matches `srderr.ErrInvalidInput` with `errors.Is` and is found with `errors.As`; `MustCatalogue` panics with that error. The message text is not specified or asserted. | testing.md §4 (duplicate entries); errors-and-results.md FR-027 | unhappy (invalid input) |
| FR-022 | `InMemorySource` MUST behave correctly with no entries: every `Lookup` is `LookupNotFound` and every `ListKind` is empty. | testing.md §4 (empty and minimal) | unhappy (empty case) |
| FR-023 | `InMemorySource` MUST read its input slice exactly once and copy it at construction; later mutation of the caller's slice MUST NOT change what the source returns. | testing.md §4 (immutability) | unhappy (immutability) |
| FR-024 | `InMemorySource` MUST NOT return a `SourceError` for any input, and MUST NOT start a goroutine, timer or other background work: it answers synchronously from memory and does not consult `ctx` (an in-memory lookup cannot block; the parameter exists to satisfy the port). It is safe for concurrent use because it is immutable. | architecture.md §5; testing.md §6 (fast) | happy |
| FR-025 | `Source`, `LookupResult`, `LookupFound`, `LookupNotFound`, `Catalogue`, `Query` and its variants, `SourceError`, `DuplicateKeyError` and `Port` MUST be exported from package `content` (the identity types are exported under content-schema-and-decoder.md FR-148). The bare names `Found` and `NotFound` MUST NOT be exported (shared-types-and-ownership.md FR-017). There is no facade or re-export package. | architecture.md §2, §4, §7 | happy |
| FR-026 | `InMemorySource` MUST be exported from package `content`, beside the port (it is a generic adapter, shared-types-and-ownership.md FR-037), and MUST NOT be exported from `content/contenttest`. It is not a test double. | shared-types-and-ownership.md FR-037, FR-038; architecture.md §7 | happy |
| FR-027 | Package `content/contenttest` MUST provide an exported shared contract suite, run against every `Source` implementation that can be built from an arbitrary entry list (the in-memory source, the test double, and the test-only wrapping adapter), covering FR-008 to FR-014, FR-016, FR-017, FR-018, FR-022 and FR-041 (it has no duplicate-key group: FR-021 is unit-tested on `Catalogue` and the constructors only). The FR-017 and FR-018 checks MUST run only when the caller supplies a failing-backend factory (FR-028), and MUST assert that the call does not panic, returns a `*SourceError` whose `Err` is the backend's error, and is never `LookupNotFound` or an empty slice. A source backed by a fixed fixture that cannot be built from arbitrary entries is covered by the read-only variant of FR-058; `data.BundledSource` is registered with both (FR-047). | architecture.md §7, §9 (bootstrap); testing.md §3, §10; shared-types-and-ownership.md FR-039 | happy |
| FR-028 | The contract suite MUST be the function `contenttest.RunSourceContract(t *testing.T, create func(t *testing.T, entries []content.Entry) content.Source, createFailing func(t *testing.T, entries []content.Entry, backendErr error) content.Source)` in `content/contenttest/contract.go`. `create` takes a list of entries and returns a `Source`. The optional `createFailing` (nil skips the failing group) takes a list of entries and a backend error and returns a `Source` whose backend fails with that error on every call. A failed check is reported through `t`; the checks go through an unexported reporter interface so the suite can be self-tested with deliberately bad adapters (FR-042). The suite is exported so consumers reuse it (FR-050); importing `testing` there is fine because only `_test.go` files import `contenttest`. The read-only variant `RunReadOnlyContract` (FR-058) lives in the same package and follows the same rules. | shared-types-and-ownership.md FR-039; testing.md §3 | happy |
| FR-029 | `contenttest.FakeSource` MUST pass the contract suite when not scripted to fail. | architecture.md §7 (every port ships a double); shared-types-and-ownership.md FR-038 | happy |
| FR-030 | `FakeSource` MUST be scriptable to fail a given error from `Lookup`, from `ListKind`, or from both, and MUST return exactly the supplied error (the same instance, wrapped in a `*SourceError` that carries the `Query`), never a panic. | testing.md §4 (collaborator failure); errors-and-results.md FR-016 | unhappy (collaborator failure) |
| FR-031 | `FakeSource` MUST record every request, as a `Query` (`LookupQuery{Key}` or `ListKindQuery{Kind}`, FR-057), in call order, and return the record as a copy. | testing.md §6 | happy |
| FR-033 | The value types in this spec (`LookupResult`, `LookupFound`, `LookupNotFound`, `Catalogue`, `Query` and its variants) MUST NOT import `context`, `embed`, `testing`, `contenttest` or `data`, and MUST NOT reference `srderr.PortError` (the identity types: content-schema-and-decoder.md FR-148). `SourceError`, `Source` and `Port` may use `context` and `srderr`. | architecture.md §3, §9; errors-and-results.md FR-029 | happy |
| FR-034 | No file in the module may name a consuming project (mechanical scan; the organisation name appears only in the module path and `AGENTS.md`). Test fixtures use clearly synthetic entries and no content from outside SRD 5.2.1; that rule is a review item with no test. | AGENTS.md, "Hard constraints" | happy |
| FR-037 | `SourceError` MUST expose the `Err` it was built with, unchanged (same instance), through the field and `Unwrap`, and MUST be usable as a pointer (`*SourceError`) with `errors.As`. | errors-and-results.md FR-002 | happy |
| FR-038 | The port name is the exported constant `content.Port = "content"`, matching `^[a-z][a-z0-9_]*$` and unique across the module (errors-and-results.md FR-006); `srderr.ErrorType` of a source failure MUST be `content_error` (shared-types-and-ownership.md FR-041 and errors-and-results.md FR-005 and its section 7 registry). The port-name uniqueness check (architecture check A8) MUST include it. | shared-types-and-ownership.md FR-041; errors-and-results.md FR-005, FR-006 | happy |
| FR-039 | `SourceError` MUST be built with the one `Query` value as its only description of the request: `&content.SourceError{Query: q, Err: err}`. Because the request is one closed value, there is no combination of fields that can disagree. A nil `Query` is invalid; `Error()` and `Unwrap()` MUST NOT panic for it. | architecture.md §3 (no nil standing in for a named state), §10; shared-types-and-ownership.md principle 4 | unhappy (invalid design) |
| FR-041 | `InMemorySource` MUST accept entries with the same `ID` under different `Kind`s. | content-schema-and-decoder.md FR-032 | unhappy (rule interaction) |
| FR-042 | The contract suite MUST fail an adapter that leaks a raw backend error (not a `*SourceError`), and one that returns `LookupNotFound` or an empty slice when its backend fails. | errors-and-results.md FR-032 | unhappy (collaborator failure) |
| FR-043 | A scenario test (`package scenarios_test`) importing only public packages (`content`, `content/contenttest`) MUST build an `InMemorySource`, look up a present and an absent key, call `ListKind`, and type-switch exhaustively over `LookupResult` with no default. | testing.md §3 | happy |
| FR-044 | Package `content` MUST have a package doc comment in `doc.go` listing the context's inputs and outputs, and every exported identifier of this spec MUST have a doc comment starting with its name. | architecture.md §4, §7 | happy |
| FR-046 | `Catalogue` MUST be a pure, synchronous, immutable value object constructed from a slice of `Entry` that it reads exactly once and copies. It MUST expose `Lookup(Key)` returning a `LookupResult` (FR-008 to FR-010) and `OfKind(Kind)` returning a copy ordered per FR-011 and empty per FR-012, and MUST reject duplicate keys per FR-021 and accept equal ids under different kinds per FR-041. `InMemorySource` and `FakeSource` MUST delegate to it, so these rules are unit-tested once with plain values and no `ctx`. | shared-types-and-ownership.md principle 1 (testable with plain values); architecture.md §3 | happy |
| FR-047 | Every `Source` implementation the package ships MUST be registered in a contract test (`content/content_contract_test.go`, `content/data/bundled_source_test.go`, `content/contenttest/contract_test.go`). `InMemorySource`, `FakeSource` and the test-only wrapping adapter run `RunSourceContract`. `data.BundledSource` (FR-059) runs both: `RunSourceContract` with a `create` that builds the source from a `fakeDecoder` returning a `content.Set` of the entries and no `createFailing` (it has no backend to fail), and `RunReadOnlyContract` over the real embedded libraries, with the entries decoded from them as `expectedEntries`. Adding an implementation without registering it fails the architecture scan of the contract tests. | shared-types-and-ownership.md FR-031, FR-039, FR-047; architecture.md §9 (bootstrap) | happy |
| FR-049 | `FakeSource` MUST live in `content/contenttest/fake_source.go`, and import only `content` and the standard library (`content` exports `Source` and `Query`). It MUST NOT import `content/data` or any other `content` sub-package (architecture.md §1 and §9; architecture-guardrails.md FR-075). `FakeSource` MUST be constructed from a slice of `Entry` (`contenttest.NewFakeSource(entries ...content.Entry)`) and is the in-memory content double that downstream use-case tests use; only `_test.go` files import `contenttest`. | shared-types-and-ownership.md FR-038; architecture.md §1, §9 | happy |
| FR-050 | Documentation (the README section on writing an adapter, and the godoc of `Source`) MUST tell adapter authors how to reuse the contract suite: it is `contenttest.RunSourceContract` (and `RunReadOnlyContract`), exported from `content/contenttest`. | shared-types-and-ownership.md FR-039 | happy |
| FR-051 | `LookupResult` MUST be defined in package `content` beside `Catalogue`, not in the port's adapter packages, because it is a pure value result that `Catalogue` returns; `Source.Lookup` returns the same type. The catalogue thereby stays free of the port's `context` dependency. | architecture.md §3, §4 | happy |
| FR-057 | `Query` MUST be a closed interface (unexported marker method) with exactly the variants `LookupQuery{Key}` and `ListKindQuery{Kind}`, listed in its doc comment, immutable and compared by value (same variant and same field). `SourceError` (FR-018) and the recording of `FakeSource` (FR-031) both use it; no other type repeats the lookup-or-list modelling. | architecture.md §3 (closed outcomes); shared-types-and-ownership.md principle 4 | happy |
| FR-058 | Package `content/contenttest` MUST provide the read-only suite `RunReadOnlyContract(t *testing.T, source content.Source, expectedEntries []content.Entry)`. It covers FR-008 to FR-014 and FR-016 against `source`, using `expectedEntries` as the fixture (ordering and kind filtering are computed from it; an absent key is generated so it is not in the fixture). It has no empty-source, same-id-under-two-kinds or failing-backend groups (FR-022, FR-041 and FR-017 need arbitrary entries or a backend), and it does not build sources. Failures are reported through `t`. | architecture.md §9 (bootstrap); testing.md §3, §10; shared-types-and-ownership.md FR-039 | happy |
| FR-059 | `data.BundledSource` MUST be defined in package `content/data` (the only package that imports `embed`) and implement `content.Source`. It reads the embedded document texts from the index of embedded documents (content-schema-and-decoder.md FR-122), or from options that replace it (`data.WithDocuments(map[string]string)` and `data.WithDecoder(content.Decoder)`, so a test injects both). `data.NewBundledSource(opts ...Option) (*BundledSource, error)` decodes each text with the `content.Decoder` (default the real JSON decoder), collects the decoded `content.Set`s, calls `content.ValidateSet(sets)` (content-schema-and-decoder.md FR-125 to FR-127), builds one `Catalogue` from the valid set, and delegates `Lookup` and `ListKind` to an `InMemorySource` over its entries. It adds no lookup logic of its own: its results equal an `InMemorySource` over the same validated entries. | shared-types-and-ownership.md FR-030, FR-031, FR-047; content-schema-and-decoder.md FR-104, FR-122, FR-125, FR-127 | happy |
| FR-060 | `BundledSource` MUST decode every embedded document through the one decode path of shared-types-and-ownership.md FR-031: no second JSON reader, no file or network access (`os`, `net`), no import of generator tooling, and no exported identifier of the embedded libraries or the index outside `content/data`. The constructor is synchronous and runs the decode once; `Lookup` and `ListKind` follow FR-024. | shared-types-and-ownership.md FR-030, FR-031, FR-032; content-schema-and-decoder.md FR-104 | happy |
| FR-061 | If any embedded document fails to decode or the merged set fails whole-set validation, `data.NewBundledSource` MUST return a nil source and a `*data.BundledInvalidError`, defined in `content/data` and matching `srderr.ErrInvalidInput` with `errors.Is`. It carries two copied slices: `DecodeProblems` (the decoder's `content.Problem`s of every failed document, in the decoder's order, document by document in index order) and `ReferenceProblems` (the `content.ReferenceProblem`s of the validation failure, in its order); at least one is non-empty, and validation runs only when every document decoded. It MUST NOT return a partial source or a `SourceError`: a bad bundle is a packaging error, not an infrastructure failure. A duplicate `Key` across documents is a `ReferenceProblem` of kind `duplicate`, not a `*DuplicateKeyError`. `data.MustBundledSource` panics with this error. | architecture.md §6; errors-and-results.md FR-027; content-schema-and-decoder.md FR-104, FR-125, FR-127 | unhappy (invalid bundle) |
| FR-062 | Building `BundledSource` twice from the same texts MUST yield sources whose `Lookup` and `ListKind` results are equal entry for entry and in the same order (FR-014). With an empty index it MUST behave as an empty source (FR-022). | testing.md §4 (determinism, empty and minimal) | unhappy (determinism) |
| FR-063 | `BundledSource` MUST NOT depend on any bundled-data spec: it enumerates the index, so a data spec that adds an embedded library needs no change here. References between data are by id; every reference is checked by `ValidateSet` over the whole merged set inside this adapter (FR-059, FR-061; content-schema-and-decoder.md FR-125 to FR-127), and none is checked per document. | shared-types-and-ownership.md FR-047; content-schema-and-decoder.md FR-127 | happy |
| FR-064 | Each content data spec (`content-thin-slice`, `content-species-backgrounds-feats`, `content-equipment`, `content-spells`, `content-monsters`, `content-classes`, `content-magic-items`, the toolbox catalogues) MUST contribute only its embedded library and the entry in the index; none may define a second bundled source, loader or registration. Data pin tests of a data spec run through `BundledSource`. | shared-types-and-ownership.md FR-047, FR-048 | happy |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-schema-and-decoder.md (FR-142) |
| FR-002 | Withdrawn: moved to content-schema-and-decoder.md (FR-143) |
| FR-003 | Withdrawn: moved to content-schema-and-decoder.md (FR-143) |
| FR-004 | Withdrawn: moved to content-schema-and-decoder.md (FR-143) |
| FR-005 | Withdrawn: moved to content-schema-and-decoder.md (FR-143, FR-145) |
| FR-006 | Withdrawn: moved to content-schema-and-decoder.md (FR-145) |
| FR-032 | Withdrawn: moved to content-source-port.md (FR-014) |
| FR-035 | Withdrawn: moved to content-schema-and-decoder.md (FR-143) |
| FR-036 | Withdrawn: moved to content-schema-and-decoder.md (FR-144) |
| FR-040 | Withdrawn: moved to shared-types-and-ownership.md (FR-038) |
| FR-045 | Withdrawn: moved to content-schema-and-decoder.md (FR-146) |
| FR-048 | Withdrawn: moved to content-schema-and-decoder.md (FR-147); the `Missing()` method of `LookupNotFound` is FR-016 here |
| FR-052 | Withdrawn: moved to content-schema-and-decoder.md (FR-114) |
| FR-053 | Withdrawn: moved to content-schema-and-decoder.md (FR-114) |
| FR-054 | Withdrawn: moved to content-schema-and-decoder.md (FR-114) |
| FR-055 | Withdrawn: moved to content-schema-and-decoder.md (FR-114) |
| FR-056 | Withdrawn: moved to content-schema-and-decoder.md (FR-114) |
| FR-065 | Withdrawn: moved to content-schema-and-decoder.md (FR-142) |

### Cases: `ListKind` ordering (one test per row)

| Label | Entries (ids, insertion order) | Expected order | Source | Path |
|---|---|---|---|---|
| already sorted | `a`, `b`, `c` | `a`, `b`, `c` | FR-011 | happy |
| reverse insertion | `c`, `b`, `a` | `a`, `b`, `c` | FR-011 | happy |
| digit before underscore before letter | `ab`, `a_b`, `a1` | `a1`, `a_b`, `ab` | FR-011 | unhappy (boundary) |
| prefix first | `ab`, `a` | `a`, `ab` | FR-011 | unhappy (boundary) |
| other kind ignored | `a` (spell), `b` (feat), list feat | `b` | FR-010, FR-011 | unhappy (rule interaction) |
| single entry | `a` | `a` | FR-011 | unhappy (empty and minimal) |

## 5. Model

- **Value objects (immutable, by value):** `Catalogue`, `Query` variants. The identity types
  (`Kind`, `ID`, `Key`, `Entry`, `MissingContent`) and typed ids are the schema
  spec's (section 3). No aggregate, no domain event.
- **Result (pure):** the closed interface `LookupResult` with `LookupFound{Entry}` and
  `LookupNotFound{Key}` plus a `Missing()` method (FR-016, FR-051). `MissingContent` is a field of other
  contexts' results, never a shared variant.
- **One rules implementation:** `Catalogue` holds duplicate-key, lookup and ordering rules;
  `InMemorySource` and `FakeSource` add only the `ctx` parameter and the `error` return (and scripting and recording).
- **Package placement** (one package per context, file per concept; no `domain/application/adapters` folder layering):

  | Type | Package and file |
  |---|---|
  | `LookupResult` (`LookupFound`, `LookupNotFound`), `Catalogue`, `DuplicateKeyError`, `Query` (`LookupQuery`, `ListKindQuery`) | `content`: `lookup_result.go`, `catalogue.go`, `query.go` |
  | `Source`, `SourceError`, `Port`, `InMemorySource` | `content`: `source.go`, `source_error.go`, `in_memory_source.go` |
  | `BundledSource`, `BundledInvalidError` | `content/data`: `bundled_source.go` (the only `embed` importer) |
  | `FakeSource`, `RunSourceContract`, `RunReadOnlyContract` | `content/contenttest`: `fake_source.go`, `contract.go` |

  `Catalogue` and the result types import neither `context` nor `srderr.PortError` (FR-033), so the rules
  stay testable with plain values.
- **Invariants:** a `LookupFound` entry's key
  equals the requested key (enforced by `Catalogue` and the contract suite). A `Catalogue`
  holds no duplicate keys (`*DuplicateKeyError`, matching `srderr.ErrInvalidInput`).
- **Failures:** result, not error: `LookupNotFound`. Port failure: `*srderr.PortError` wrapping `*SourceError`.
  Validation error: duplicate keys. Panic: `MustCatalogue` and `data.MustBundledSource` only.

## 6. Interface

```go
package content

// Port is the srderr port name of every Source failure.
const Port = "content"

// Source is where the engine gets content from. Calls may do I/O, so they take
// a context. Expected absence is LookupNotFound or an empty slice. Infrastructure
// failure is the returned error, which callers wrap with srderr.WrapPort or
// srderr.GuardValue (Port, "lookup" or "list_kind"). Neither method panics.
type Source interface {
	Lookup(ctx context.Context, key Key) (LookupResult, error)
	// ListKind returns all entries of kind, ordered by ID; empty if there are none.
	ListKind(ctx context.Context, kind Kind) ([]Entry, error)
}

// Key, Kind, ID, Entry, MissingContent: content-schema-and-decoder.md
// (FR-142 to FR-148), defined there, not here.

// LookupResult is the outcome of Lookup. Variants: LookupFound, LookupNotFound.
//
//sumtype:decl
type LookupResult interface{ lookupResult() }

type LookupFound struct{ Entry Entry }
type LookupNotFound struct{ Key Key }

func (n LookupNotFound) Missing() MissingContent

// Catalogue is an immutable index of entries.
type Catalogue struct{ /* unexported */ }

func NewCatalogue(entries []Entry) (Catalogue, error) // *DuplicateKeyError on a duplicate key
func MustCatalogue(entries []Entry) Catalogue
func (c Catalogue) Lookup(key Key) LookupResult
func (c Catalogue) OfKind(kind Kind) []Entry // a copy, ordered by ID

// Query is what was asked of a source. Variants: LookupQuery, ListKindQuery.
//
//sumtype:decl
type Query interface{ query() }

type LookupQuery struct{ Key Key }
type ListKindQuery struct{ Kind Kind }

// SourceError is the typed cause of a source failure.
type SourceError struct {
	Query Query // never nil
	Err   error
}

func (e *SourceError) Error() string
func (e *SourceError) Unwrap() error

var _ Source = (*InMemorySource)(nil)

func NewInMemorySource(entries []Entry) (*InMemorySource, error) // *DuplicateKeyError on a duplicate key
```

```go
package data // content/data

type Option func(*config)

func WithDocuments(texts map[string]string) Option // default: the embedded index
func WithDecoder(d content.Decoder) Option         // default: the real JSON decoder

var _ content.Source = (*BundledSource)(nil)

// NewBundledSource decodes and validates the documents. It returns a
// *BundledInvalidError (errors.Is srderr.ErrInvalidInput) if any decode or the
// validation fails.
func NewBundledSource(opts ...Option) (*BundledSource, error)
func MustBundledSource(opts ...Option) *BundledSource

type BundledInvalidError struct {
	DecodeProblems    []content.Problem          // copies; at least one of the
	ReferenceProblems []content.ReferenceProblem // two slices is non-empty
}
```

```go
package contenttest // content/contenttest

func NewFakeSource(entries ...content.Entry) *FakeSource // panics only via content.MustCatalogue
func (f *FakeSource) FailLookup(err error) *FakeSource   // scripted failure (also FailListKind, FailAll)
func (f *FakeSource) Queries() []content.Query           // a copy, in call order

// RunSourceContract: for sources built from arbitrary entries.
// createFailing may be nil.
func RunSourceContract(t *testing.T,
	create func(t *testing.T, entries []content.Entry) content.Source,
	createFailing func(t *testing.T, entries []content.Entry, backendErr error) content.Source)

// RunReadOnlyContract: for sources backed by a fixed fixture
// (BundledSource): FR-008 to FR-014 and FR-016 only.
func RunReadOnlyContract(t *testing.T, source content.Source, expectedEntries []content.Entry)
```

- **Context, not async** (architecture.md §4): use cases call the port with their `ctx`, then hand plain values to the
  domain. Failure is the returned `error`, never a panic; use cases wrap with `srderr.GuardValue`
  (errors-and-results.md FR-016, FR-032).
- **Typed access:** a use case calls `source.Lookup(ctx, spellID.Key())` and type-switches on the result
  (`case content.LookupFound:` then asserts `found.Entry.(content.Spell)`). `LookupResult` is not generic and has no typed accessor. On
  `LookupNotFound` a use case returns its own `<UseCase>...ContentMissing` variant holding `spellID.Missing()`
  or `notFound.Missing()` (shared-types-and-ownership.md FR-017, FR-018; FR-048).
- **Exports:** package `content`: the FR-025 list (the identity types and
  typed ids live in the same package but are defined by the decoder spec). `content/data`: `BundledSource`,
  `NewBundledSource`, `MustBundledSource`, `BundledInvalidError`, the options. `content/contenttest`: `FakeSource` and the suites
  (FR-026). Every exported identifier has a doc comment (FR-044).
- **Contract suite:** `content/contenttest/contract.go`, exported; two entry points
  (FR-027, FR-058); consumers reuse it as FR-050 describes.
- **In-memory double:** downstream specs use `contenttest.FakeSource`, built from
  synthetic entries (FR-049); production hosts use `content.InMemorySource`. A context that needs a source declares its own
  small consumer-side interface; `content.Source` satisfies it.

## 7. Observability

`Source`, the value types, `InMemorySource` and `FakeSource` are silent by design
(observability.md §3: domain silent; no retries, fallbacks or I/O timing here). A returned `LookupNotFound`
or returned `*srderr.PortError` is recorded by the calling use case (observability.md §5, §7) under its
own context prefix: `LookupNotFound` as `srd5e.outcome=failure` plus the use case's variant attribute (no
`error.type`), a source failure in the error counter with `error.type=content_error`
(FR-038; shared-types-and-ownership.md FR-041, FR-051; errors-and-results.md FR-043). This spec owns no
signal and reserves no names; the error carries `Query` and `Err` for diagnosis
(kind is low-cardinality; the id stays in the result or error).

1. **What can go wrong?** Backend unavailable (`*SourceError`); absent key (`LookupNotFound`);
   duplicate keys (`*DuplicateKeyError`, invalid input).
2. **How would an operator know?** The calling use case's span and counters (use-case specs).
3. **What is needed to diagnose it?** `Query`, `Err`, and for a panic the `PanicError` stack.
4. **What should be counted or timed?** Lookups and `ListKind` sizes, by the calling use case (this layer
   cannot read a clock, observability.md §4).
5. **Is existing instrumentation still correct?** Nothing exists yet.
6. **What should a consumer monitor?** Any `content_error`; a rising `ContentMissing` rate for ids
   believed to exist.
7. **Is anything now noisy?** No: nothing is emitted.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none defined by this spec | n/a | n/a | n/a | Defined by calling use-case specs: alert on `error.type=content_error`; watch `srd5e.outcome=failure` results that are a `ContentMissing` variant |

## 8. Test plan

Tests are table-driven with `t.Run` and `t.Parallel` where safe; external test packages for public behaviour.

| Id | Level | Notes |
|---|---|---|
| FR-007 | architecture | Compile-time assignments in `content/source_test.go`: `var _ content.Source = (*content.InMemorySource)(nil)` and the method signatures take `ctx` first and return `(LookupResult, error)` |
| FR-008, FR-010, FR-012, FR-020, FR-022, FR-041 | contract | Contract suite; FR-022 empty-entries group; FR-041 same id under two kinds, both found |
| FR-009, FR-013 | contract | Contract suite; exact `LookupNotFound` and its key; mutating a returned slice leaves the next result unchanged |
| FR-011 | contract | Contract suite, ordering table above |
| FR-014 | contract | Run for `InMemorySource` and `FakeSource`; element-wise `==` or `Equal` |
| FR-015, FR-016 | unit | `content/lookup_result_test.go`: exhaustive type switch with no default, value equality, accessors; `LookupNotFound.Missing()` equals the `MissingContent` of its key; two contract checks assert `LookupFound.Entry` key equals the request |
| FR-017, FR-018 | contract + unit | Failing-backend group via `createFailing`: `errors.As` a `*SourceError`; no panic; never `LookupNotFound` or empty; `Query` variant and key or kind, and `errors.Is` the backend error, for both operations. In this repo `createFailing` is a test-only adapter in `content/contenttest/contract_test.go` wrapping a failing backend, and a use-case-style call through `srderr.GuardValue` yields a `*srderr.PortError` with `Port == content.Port`. Unit: fields, exhaustive type switch over `Query`, `Unwrap` identity in `content/source_error_test.go` |
| FR-019 | architecture | Scan of `SourceError`, `LookupResult`, `LookupFound`, `LookupNotFound` for fields named `Message`, `Description`, `Text`, `DisplayText` (errors-and-results.md FR-007, check A8) |
| FR-021, FR-046 | unit | `content/catalogue_test.go`: `errors.As` a `*DuplicateKeyError` carrying the duplicate key and `errors.Is` `srderr.ErrInvalidInput` (never the message), `MustCatalogue` panics with that error; lookup present and absent, `OfKind` ordering table, empty, input mutation, returned slice copy, same id under two kinds; no `ctx`. `in_memory_source_test.go` and `fake_source_test.go` assert the duplicate rule through their constructors |
| FR-023, FR-024 | unit | Mutate the input after construction; `runtime.NumGoroutine` unchanged around calls and a context that is already cancelled does not change the result; concurrent calls under `-race` |
| FR-025, FR-026, FR-051 | architecture | Package placement per section 5; exported-name check fails on exported `Found` or `NotFound` (errors-and-results.md check A7); `InMemorySource` in `content`, absent from `contenttest` |
| FR-027, FR-029, FR-047 | contract + architecture | `RunSourceContract` against `InMemorySource`, `FakeSource` (no failing group) and the test-only wrapping adapter (with failing group). `data.BundledSource` runs it through an injected fake decoder (no `createFailing`) and `RunReadOnlyContract` over the real embedded libraries. Architecture scan of the contract tests fails if a shipped `Source` is not registered |
| FR-028 | unit + architecture | Both suites take the same `create` signature; self-test with a recording reporter; no production package imports `contenttest` |
| FR-030, FR-031 | unit | `content/contenttest/fake_source_test.go`: scripted failure per operation, same error instance via `errors.Is`, no panic; recorded order, query variants, copy semantics |
| FR-033 | architecture | Import ban (`context`, `embed`, `testing`, `data`, `contenttest`) for the value-type files, in `internal/archtest` |
| FR-034 | architecture | Mechanical consumer-name scan over the module; the SRD-only fixture rule is a review item with no test |
| FR-037, FR-038, FR-039 | unit | Same-instance `Err`, `errors.As` on a pointer; `Port` matches `^[a-z][a-z0-9_]*$`, unique (check A8), `srderr.ErrorType` equals `content_error`; the struct has one `Query` field, a nil `Query` does not panic in `Error()` or `Unwrap()`, each `Query` variant compiles |
| FR-042 | contract | Suite self-test in `content/contenttest/contract_internal_test.go`: bad adapters that leak a raw error, return `LookupNotFound`, or return an empty slice on backend failure each make a failing-backend check report through the recording reporter |
| FR-043 | scenario | `scenarios/content_source_test.go`, `package scenarios_test`, imports public packages only |
| FR-044 | architecture | Package doc comment on `content`; doc comment on every exported identifier |
| FR-049, FR-050 | architecture | Import check on `content/contenttest/fake_source.go` (only `content` and the standard library) and the allow-list (only `_test.go` files import `contenttest`); README and `Source` godoc contain `contenttest.RunSourceContract` |
| FR-057 | unit | `content/query_test.go`: equality per variant, variants unequal to each other, exhaustive type switch with no default |
| FR-058 | contract | Self-test: read-only suite passes a conforming fixed source; wrong order, entry of another kind or aliased slice is reported; no group builds sources; registered for `BundledSource` |
| FR-059, FR-060, FR-062, FR-063 | adapter | `content/data/bundled_source_test.go`: an injected fake decoder records exactly one call per text and passes the sets to `ValidateSet`; a goroutine count shows no background work; a document with a dangling reference decodes clean and is rejected only by whole-set validation; defaults serve the real index; results equal an `InMemorySource` over the same entries; two builds give equal ordered results; empty index gives an empty source |
| FR-060, FR-063 | architecture | No `os` or `net` import in `content/data`; no generator tooling import; only `content/data` imports `embed`; one decode call site; no data spec's library imported directly |
| FR-061 | adapter | A corrupted text returns `*BundledInvalidError` with `DecodeProblems` equal to the decoder's list and empty `ReferenceProblems`; two texts holding the same key give a `duplicate` problem in `ReferenceProblems` and empty `DecodeProblems`, not a `*DuplicateKeyError`; a corrupted text plus a dangling reference in another text gives empty `ReferenceProblems`, and `DecodeProblems` is ordered by index, document by document; an unresolved whole-set reference an `unresolvedReference` problem; slices are copies, no message text; `errors.Is` `srderr.ErrInvalidInput`; no source returned |
| FR-064 | doc review | Each data spec's library appears in the index; no second bundled source (exported-name scan) |

- **Doubles and suites:** `FakeSource`, `InMemorySource` and the contract suites; fixtures use
  a synthetic `Entry` defined in the tests, not real SRD content. Shared rules are tested once against
  `Catalogue` with plain values.
- **Mutation testing:** `gomu` on `./content` (`Catalogue`, `LookupResult`, `Query`) must
  score at least 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's
  processors (command in testing.md; the identity types are mutated under the schema spec); the adapter and
  double SHOULD be mutated too.
- **Telemetry tests:** none (nothing emitted); use-case specs assert the failure is recorded once.
- **Not covered:** only the SRD-only fixture rule in FR-034 (review item). Real-backend adapters are
  consumers' responsibility (FR-050).

**Implementation status.**

- Done (Phase 2, step 3, packages `content` and `content/contenttest`): `Source`, `Port`, `LookupResult` (`LookupFound`, `LookupNotFound`), `Query` (`LookupQuery`, `ListKindQuery`), `Catalogue`, `DuplicateKeyError`, `SourceError` and `InMemorySource` (FR-007 to FR-031, FR-033, FR-037 to FR-039, FR-041 to FR-044, FR-046, FR-049 to FR-051, FR-057); `FakeSource`, `RunSourceContract` and `RunReadOnlyContract` with self-tests against deliberately bad adapters (FR-027 to FR-031, FR-042, FR-049, FR-058); the test-only wrapping adapter and `scenarios/content_source_test.go` (FR-027, FR-043); archtest rule `source-files` (FR-033, FR-049); FR-034 is the existing `consumer-name` scan (A10). FR-047 is met for `InMemorySource`, `FakeSource` and the wrapping adapter only.
- Mutation (gomu, per file): `source_error.go` 3 mutants, 100 (2 not viable; the `Error` message text is now asserted exactly); `in_memory_source.go` 4 mutants, 100; `source.go` has no mutable statements (constant and interface only), nothing to score; `named_text.go` 11 mutants, 100 (2 not viable). No survivors.
- Choices the spec left open: `NewCatalogue` builds on `NewSet`, so a nil entry or an entry with an invalid key is rejected as the `*SetError` of `NewSet` (also wrapping `srderr.ErrInvalidInput`) and only a repeated key is a `*DuplicateKeyError`; `OfKind` returns a non-nil empty slice; the suites compare entries with `reflect.DeepEqual` so entry types that are not comparable work; the contract fixture entries are an unexported synthetic type, so `createFailing` and `create` are built from entries a real adapter must be able to hold; the failing group requires a nil result (nil `LookupResult`, nil slice) together with the `*SourceError`; the `//sumtype:decl` comments of section 6 are omitted because the lint is not applied (go.md section 7); FR-024's goroutine-count check is not made (it is racy next to parallel tests), the concurrent-call and cancelled-context checks are; the A8 scan skips package `content`, so FR-019 is checked by a reflection test in `content/source_test.go`; the FR-047 architecture scan for unregistered sources is not built.
- Done (Phase 2, step 6, package `content/data`): `BundledSource`, `BundledInvalidError`, `NewBundledSource`, `MustBundledSource`, `WithDocuments`, `WithDecoder` and `Files()` (FR-059 to FR-063); the contract registrations of FR-047 (`RunSourceContract` over an injected fake decoder with no failing backend, `RunReadOnlyContract` over the embedded documents); FR-064 holds for the slice data (one index, no second bundled source). The tests are `bundled_source_test.go` and `bundled_source_internal_test.go`.
- Choices the spec left open: the embedded index is the `embed.FS` itself (`fs.ReadDir`), not a second table, and `Files()` returns it as an `fs.FS`; an index entry that cannot be read becomes a document with empty text, which the decoder rejects (an `embed.FS` never fails; the fallback is tested with fixture file systems); a decoder that returns an error holding no `content.Problem` is reported as one `syntax` problem at the root, so a `*BundledInvalidError` always holds a problem; the package comment of `content/data` sits in its one Go file, `bundled_source.go`, because A12 allows no `doc.go` beside it (A9 is relaxed for that package, architecture.md A9).
- Mutation (testing.md §8), `content/data/bundled_source.go`: score 92.3% (41 mutants, 36 killed, 3 survived, 2 not viable); at or above the 80 gate, so no tests added; survivors not individually triaged.

Not yet implemented: the scan that fails an unregistered `Source` (FR-047, no archtest row added) and the doc review of FR-064 for later data specs.

## 9. Open questions and decisions

No question is open and none is escalated.

Settled (one line each):

- Q1 two methods over `Key`; Q5 no streaming (a slice, not `iter.Seq`); Q6 `LookupNotFound`, not nil
  (architecture.md §3); Q7 ordinal sort by id; Q9 `Source` stays in package `content`
  (shared-types-and-ownership.md FR-069: a port is defined by its consumer, next to the vocabulary it uses); Q11 no reserved `srd5e.content.*` names (section 7); Q14 no generic accessor on `LookupResult`. Q10 (`ID`
  is a comparable struct with unexported fields, not a bare string) and Q13 (`MissingContent`) moved with the identity types to
  content-schema-and-decoder.md FR-143, FR-147.
- Q2 contract suites exported from `contenttest`: shared-types-and-ownership.md FR-039. Q3 id syntax and 64 units:
  FR-007 there and content-schema-and-decoder.md FR-143, aligned with its FR-030. Q4 version rejection is a decoder failure:
  FR-034 there. Q8 typed ids owned by the decoder spec: FR-046 there. Q12 `InMemorySource` is an
  adapter only, `FakeSource` the double: FR-037, FR-038 there. Q15 `MagicItemID` maps to `magicItem`,
  `ItemID` to `equipment`: FR-005 and FR-046 there, id requested of content-schema-and-decoder.md FR-114
  (row R2-magic-items). Q16 errors-and-results.md OQ-6 already names `Query`: FR-018, FR-039. Decoder Q3
  wording gone; decoder Q17 (rank conflict) settled by shared-types-and-ownership.md FR-064: the identity types
  moved to the decoder spec and this spec depends on it.

## 10. Changelog impact

Under `Unreleased`:

- **Added:** package `content`: the `Source` port,
  the closed `LookupResult` (`LookupFound`, `LookupNotFound`),
  `Catalogue`, `Query` (`LookupQuery`, `ListKindQuery`), `SourceError`, `InMemorySource`;
  package `content/data`: `BundledSource` with `BundledInvalidError`; package `content/contenttest`: `FakeSource` and the
  exported contract suites. The identity types (`Key`, `Kind`, `ID`,
  `Entry`, `MissingContent`) are recorded by content-schema-and-decoder.md.
- **Changed:** none; a source failure is a `*srderr.PortError` with `Port` `content` and `error.type`
  `content_error` (errors-and-results.md OQ-5).

Everything else is an addition, since nothing earlier exists to change. After
first release, adding a `Kind` value or a `LookupResult` variant is
**BREAKING** (consumers switch over them).
