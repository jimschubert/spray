package errors

import (
	"errors"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
)

func TestForEachJoinError(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  []string
	}{
		{
			name:  "single error",
			input: errors.Join(&ParsingError{Pos: ast.Position{Line: 1, Col: 5}, Message: "bad token"}),
			want:  []string{"parsing error at 1:5: bad token"},
		},
		{
			name: "two joined errors",
			input: errors.Join(
				&ParsingError{Pos: ast.Position{Line: 3, Col: 1}, Message: "unexpected EOF"},
				&ResolvingError{Pos: ast.Position{Line: 10, Col: 4}, Message: "unknown type Foo"},
			),
			want: []string{
				"parsing error at 3:1: unexpected EOF",
				"resolver error at 10:4: unknown type Foo",
			},
		},
		{
			name: "nested join unwraps all leaves",
			input: errors.Join(
				&ParsingError{Pos: ast.Position{Line: 1, Col: 1}, Message: "first"},
				errors.Join(
					&ResolvingError{Pos: ast.Position{Line: 2, Col: 2}, Message: "second"},
					&ResolvingError{Pos: ast.Position{Line: 3, Col: 3}, Message: "third"},
				),
				&ParsingError{Pos: ast.Position{Line: 4, Col: 4}, Message: "fourth"},
			),
			want: []string{
				"parsing error at 1:1: first",
				"resolver error at 2:2: second",
				"resolver error at 3:3: third",
				"parsing error at 4:4: fourth",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			joined, ok := errors.AsType[JoinUnwrap](tc.input)
			assert.True(t, ok, "test input must implement JoinUnwrap")

			var got []string
			ForEachJoinError(joined, func(e error) {
				got = append(got, e.Error())
			})

			assert.Equal(t, tc.want, got)
		})
	}
}
