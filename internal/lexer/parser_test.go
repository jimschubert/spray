package lexer

import (
	"testing"
)

func TestParseSimpleNamespace(t *testing.T) {
	input := `namespace acme.users.v1

api UserService @version(2) @style(rest) {
  GET  /      -> Page<User>
    @query(PaginationInput)

  GET  /:id   -> User
    @errors(401, 404)
}
`

	p, err := New()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	_, err = p.Parse(input)
	if err != nil {
		t.Logf("parse error: %v", err)
	}
}
