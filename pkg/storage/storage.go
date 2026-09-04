// Package storage is the public name of the core storage contract.
//
// # Why this package exists
//
// The storage interfaces live in internal/storage, which the Go tool forbids
// any other module from importing. That is the right home for the
// implementation, and it was the wrong home for the contract: a module such as
// Alchemai Mem implements modules.StorageWrapperProvider, whose method is
//
//	WrapStorage(primary storage.FullStorage) storage.FullStorage
//
// and a type that cannot be named cannot be embedded, so the forwarding shim a
// wrapper needs cannot be written outside the core. Type inference gets a
// module as far as declaring the method, but not as far as returning a
// different value from it, because Go forbids embedding a type parameter in a
// struct. The consequence was that a wrapper module could install nothing and
// enforcement rested entirely on the RM-2 and RM-3 predicates.
//
// A type alias is not a new type: `pkg/storage.FullStorage` and
// `internal/storage.FullStorage` are the same type, so a value of one is a
// value of the other, a method declared with one satisfies an interface
// declared with the other, and an embedded field of one promotes the methods
// of the other. That is all an external wrapper needs.
//
// # Rules for this package
//
// It is a pure alias layer. It declares no interface, no struct and no
// function body of its own, so there is exactly one definition of every name
// and the two spellings cannot drift. Anything that needs logic belongs in
// internal/storage and is re-exported here, never re-implemented.
//
// See ADR 004 in the Alchemai suite knowledge base.
package storage

import "github.com/madeindigio/remembrances-mcp/internal/storage"

// The storage contracts. A module names these to write a wrapper; the core
// keeps implementing the same types under their internal name.
type (
	// Storage is the three-layer memory contract: facts, vectors, graph,
	// documents and events.
	Storage = storage.Storage

	// StatsProvider reports counts and sizes for a principal.
	StatsProvider = storage.StatsProvider

	// CodeStorage is the code indexing contract.
	CodeStorage = storage.CodeStorage

	// StorageWithStats is Storage plus StatsProvider, the shape the MCP tool
	// manager takes.
	StorageWithStats = storage.StorageWithStats

	// FullStorage is the shape modules.ModuleConfig carries and
	// modules.StorageWrapperProvider wraps.
	FullStorage = storage.FullStorage

	// ChangeLister is the pull half of the sync contract (RM-9). It is a
	// standalone interface consumers assert for, not a member of FullStorage.
	ChangeLister = storage.ChangeLister
)

// The named types the contract's signatures use. A wrapper that only forwards
// does not need them, but a wrapper that mediates a method has to spell its
// results, so they are re-exported with the interfaces rather than after the
// first module discovers one is missing.
type (
	// VectorResult is one hit of a vector similarity search.
	VectorResult = storage.VectorResult

	// Entity is a graph node.
	Entity = storage.Entity

	// Relationship is a graph edge.
	Relationship = storage.Relationship

	// GraphResult is one step of a graph traversal.
	GraphResult = storage.GraphResult

	// Document is a knowledge base document.
	Document = storage.Document

	// DocumentResult is one hit of a document search.
	DocumentResult = storage.DocumentResult

	// HybridSearchResult combines vector, graph and key-value hits.
	HybridSearchResult = storage.HybridSearchResult

	// MemoryStats is what StatsProvider returns.
	MemoryStats = storage.MemoryStats

	// ConnectionConfig configures a storage implementation.
	ConnectionConfig = storage.ConnectionConfig

	// Event is a stored temporal event.
	Event = storage.Event

	// EventSearchParams are the arguments of an event search.
	EventSearchParams = storage.EventSearchParams

	// EventSearchResult is one hit of an event search.
	EventSearchResult = storage.EventSearchResult

	// Change is one entry of the change feed; a tombstone carries Deleted.
	Change = storage.Change

	// CodeProject is an indexed project.
	CodeProject = storage.CodeProject

	// CodeFile is an indexed file.
	CodeFile = storage.CodeFile

	// CodeSymbol is an indexed symbol.
	CodeSymbol = storage.CodeSymbol

	// CodeSymbolSearchResult is one hit of a symbol similarity search.
	CodeSymbolSearchResult = storage.CodeSymbolSearchResult

	// CodeIndexingJob is one indexing job record.
	CodeIndexingJob = storage.CodeIndexingJob

	// CodeChunk is one chunk of a large symbol.
	CodeChunk = storage.CodeChunk

	// CodeChunkSearchResult is one hit of a chunk similarity search.
	CodeChunkSearchResult = storage.CodeChunkSearchResult
)

// The tenancy switch and its error, re-exported as values rather than wrapped
// in functions so this package keeps having no logic of its own. A commercial
// module that installs a real principal resolver turns the fail-closed switch
// on through SetRequirePrincipal; without this it could not reach it (RM-10).
var (
	// SetRequirePrincipal turns the legacy no-principal path on or off for
	// every storage implementation in the process.
	SetRequirePrincipal = storage.SetRequirePrincipal

	// RequirePrincipal reports whether the legacy no-principal path is
	// disabled.
	RequirePrincipal = storage.RequirePrincipal

	// ErrNoPrincipal is the fail-closed error: no principal, deny.
	ErrNoPrincipal = storage.ErrNoPrincipal

	// SetEmbeddingDimension sets the vector dimension the schema is built
	// for. A module that provides its own embedder has to be able to declare
	// the dimension it produces.
	SetEmbeddingDimension = storage.SetEmbeddingDimension

	// EmbeddingDimension reports the vector dimension in force.
	EmbeddingDimension = storage.EmbeddingDimension
)
