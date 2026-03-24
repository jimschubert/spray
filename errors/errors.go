package errors

import (
	"fmt"

	"github.com/jimschubert/spray/ast"
)

// JoinUnwrap is an interface that allows unwrapping a joined error into its aggregated errors.
// Required because a joinError in stdlib has an Unwrap signature not exposed through errors.Unwrap; there might be some other way to access.
type JoinUnwrap interface {
	error
	Unwrap() []error
}

// ParsingError represents an error encountered during parsing.
type ParsingError struct {
	Pos     ast.Position
	Message string
}

func (e *ParsingError) Error() string {
	return fmt.Sprintf("parsing error at %d:%d: %s", e.Pos.Line, e.Pos.Col, e.Message)
}

// ResolvingError represents an error encountered during resolving.
type ResolvingError struct {
	Pos     ast.Position
	Message string
}

func (e *ResolvingError) Error() string {
	return fmt.Sprintf("resolver error at %d:%d: %s", e.Pos.Line, e.Pos.Col, e.Message)
}
