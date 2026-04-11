package errors_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jimschubert/spray/ast"
	sprayerr "github.com/jimschubert/spray/errors"
)

func ExampleForEachJoinError() {
	// Simulate a parser returning multiple errors joined together.
	err := errors.Join(
		&sprayerr.ParsingError{Pos: ast.Position{Line: 12, Col: 5}, Message: "unexpected token '}'"},
		&sprayerr.ResolvingError{Pos: ast.Position{Line: 15, Col: 10}, Message: "unknown type 'UserDTO'"},
	)

	if err != nil {
		if wrapped, ok := errors.AsType[sprayerr.JoinUnwrap](err); ok {
			var buf strings.Builder
			sprayerr.ForEachJoinError(wrapped, func(e error) {
				fmt.Fprintf(&buf, "  -> %s\n", e.Error())
			})
			fmt.Print(buf.String())
		}
	}

	// Output:
	//   -> parsing error at 12:5: unexpected token '}'
	//   -> resolver error at 15:10: unknown type 'UserDTO'
}
