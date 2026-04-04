package jsonschema

// Options holds configuration for the JSON Schema emitter.
type Options func(*emitJsonSchema)

// WithIDPrefix returns an Option that sets a prefix for all $id values in emitted schemas.
func WithIDPrefix(prefix string) Options {
	return func(e *emitJsonSchema) {
		e.idPrefix = prefix
	}
}

// WithDraft returns an Option that sets the $schema value for emitted schemas based on the specified draft version.
func WithDraft(draft string) Options {
	return func(e *emitJsonSchema) {
		switch draft {
		case "2020-12":
			e.draft = "https://json-schema.org/draft/2020-12/schema"
		case "2019-09":
			e.draft = "https://json-schema.org/draft/2019-09/schema"
		default:
			// default to latest draft
			e.draft = "https://json-schema.org/draft/2020-12/schema"
		}
	}
}
