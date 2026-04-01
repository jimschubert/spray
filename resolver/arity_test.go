package resolver

import (
	"testing"

	"github.com/jimschubert/spray/ast"
)

func Test_arity(t *testing.T) {
	lit := func(name string) ast.StringLiteral {
		return ast.StringLiteral{Value: name}
	}
	tests := []struct {
		name string
		node ast.SpecNode
		want int
	}{
		{
			name: "model with no generic params",
			node: &ast.Model{
				Name:          lit("User"),
				GenericParams: nil,
			},
			want: 0,
		},
		{
			name: "model with 2 generic params",
			node: &ast.Model{
				Name:          lit("Page"),
				GenericParams: []ast.StringLiteral{lit("T"), lit("U")},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := arity(tt.node); got != tt.want {
				t.Errorf("arity() = %v, want %v", got, tt.want)
			}
		})
	}
}
