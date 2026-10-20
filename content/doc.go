// Package content holds the game content of System Reference Document 5.2.1
// as typed, immutable values, and the ports that deliver it.
//
// Inputs are JSON documents (the bundled SRD data or a consumer's own) read
// by the strict [JSONDecoder], through the [Decoder] port and the [Loader]
// use case, and the content-source port through which a [Source] supplies
// entries. Outputs are validated entries (species, backgrounds,
// classes, feats, spells, equipment, monsters and magic items)
// that implement [Entry], each identified by a [Key] and gathered in a
// [Set]. Failures are typed problem lists, never panics.
//
// Handoff types: [Key], [Entry] and the typed ids and closed vocabularies
// flow downstream to the character, combat and toolbox contexts; [LookupResult]
// (with [LookupFound] and [LookupNotFound]) and [Query] are what a [Source]
// call returns and reports in a [SourceError]; [Set] is what the
// cross-reference validation ([ValidateSet]) checks; [LoadSetResult] (with
// [LoadSetLoaded] and [LoadSetRejected]) is what [Loader.LoadSet] returns, and
// [Problem] is one path and kind of a rejected document. [Catalogue] and
// [InMemorySource] serve entries from memory; package contenttest has the test
// doubles ([FakeSource], [FakeDecoder]) and the contract suites for adapters.
//
// Reading the results. A rejected document is a value, never a panic: read
// every problem with [Problems], at most [MaxProblems] of them, and never
// log [Problem.Error] unescaped paths of your own (it escapes and bounds them,
// but a path is still document text). [ValidateSet] resolves the references
// between entries over the merged documents and reports through its error
// only: it emits no signal, so a consumer must read what it returns. The
// decoder does not honour context cancellation while a document decodes;
// bound the input with [MaxDocumentBytes]. A cancelled context given to
// [Loader.LoadSet] returns before any signal and is counted nowhere. Text is
// kept character for character, but control and bidirectional characters are
// rejected, so a host still sanitises text for its own display.
//
// This package does no I/O: loading bytes is a [Source] adapter's job, and
// the bundled data lives in the content/data package.
//
// The entry types of the species, background and feat kinds ([Species],
// [Background], [Feat]) are handoff types too; [CustomBackgroundEquipmentProblems]
// validates the equipment of a custom background against a caller-supplied
// [EquipmentValuation].
//
// The [Monster] entry (one stat block of the monsters and animals chapters),
// its value objects ([ChallengeRating], [StatBlockPart], [MultiattackPlan] and
// the rest) and [DeriveMonsterID] are handoff types too; the limits of a stat
// block are the Monster* constants and [MonsterHitPointDieFaces].
//
// The package is built in steps; identifiers named above appear as their
// steps land, see doc/plans/plan-phase-2.md.
package content
