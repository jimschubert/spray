package emitter

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
)

func TestJoinPathSegments(t *testing.T) {
	tests := []struct {
		name     string
		segments []ast.PathSegment
		want     string
	}{
		{
			name:     "empty segments returns root",
			segments: nil,
			want:     "/",
		},
		{
			name: "single static segment",
			segments: []ast.PathSegment{
				{Value: "users"},
			},
			want: "/users",
		},
		{
			name: "single param segment",
			segments: []ast.PathSegment{
				{Value: "id", IsParam: true},
			},
			want: "/:id",
		},
		{
			name: "multiple static segments",
			segments: []ast.PathSegment{
				{Value: "api"},
				{Value: "v1"},
				{Value: "users"},
			},
			want: "/api/v1/users",
		},
		{
			name: "mixed static and param segments",
			segments: []ast.PathSegment{
				{Value: "users"},
				{Value: "id", IsParam: true},
				{Value: "posts"},
			},
			want: "/users/:id/posts",
		},
		{
			name: "trailing param",
			segments: []ast.PathSegment{
				{Value: "users"},
				{Value: "id", IsParam: true},
			},
			want: "/users/:id",
		},
		{
			name: "multiple params",
			segments: []ast.PathSegment{
				{Value: "orgs"},
				{Value: "orgId", IsParam: true},
				{Value: "members"},
				{Value: "memberId", IsParam: true},
			},
			want: "/orgs/:orgId/members/:memberId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPathSegments(tt.segments)
			assert.Equal(t, tt.want, got)
		})
	}
}

