// Package emitter defines the interface that all implementations in this package
// must satisfy, along with utilities for bucketing spec nodes by type.
//
// Every emitter (markdown, jsonschema, mermaid, etc.) implements the Emitter
// interface:
//
//	type Emitter interface {
//		EmitAll() ([]Output, error)
//		EmitOne(typ SpecType, name string) (Output, error)
//	}
//
// Contributors should use CollectAll to partition parsed stencils by SpecType:
//
//	buckets := emitter.CollectAll(stencils...)
//	// buckets[emitter.SpecApi], buckets[emitter.SpecModel], ...
//
// However, note that bucketing will result in processing specs out of order.
//
// Each Output artifact exposes its Filename, Contents, and ContentType
// (ContentText or ContentBinary), allowing the CLI to write artifacts
// to disk as expected by the Emitter.
//
// The emitter/markdown package is the reference implementation. The
// emitter/schema package provides a shared Schema struct and builder used
// by schema-based emitters such as JSON Schema and (eventually) OpenAPI.
package emitter
