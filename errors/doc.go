// Package errors defines project-wide error types used during parsing and
// resolution, along with utilities for handling joined (multi) errors.
//
// The two concrete error types, ParsingError and ResolvingError, both carry
// source Position information (line and column) so CLI output can
// refer to the exact location of the error.
//
// When multiple errors occur during a single phase (e.g. several bad type
// references in one stencil), they are aggregated via errors.Join. Use
// ForEachJoinError to recursively flatten or inspect each error:
//
//	_, err := p.Parse(src)
//	if err != nil {
//		if wrapped, ok := errors.AsType[errors.JoinUnwrap](err); ok {
//			errors.ForEachJoinError(wrapped, func(e error) {
//				fmt.Println(e)
//			})
//		}
//	}
package errors
