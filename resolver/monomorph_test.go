package resolver

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func TestMonomorphize(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantKeys   map[string]string // key → expected Name
		wantAbsent []string
	}{
		{
			name: "concrete generic usage deduplicated",
			src: `
namespace acme.v1
model Page<T> { data: T[] }
model User { id: uuid @primary }
model UserList {
  a: Page<User>
  b: Page<User>
}
`,
			wantKeys: map[string]string{
				"acme.v1.Page<acme.v1.User>": "PageUser",
			},
		},
		{
			name: "nested concrete generics",
			src: `
namespace acme.v1
model Page<T> { data: T[] }
model Result<T, E> {
  data: T?
  err:  E?
}
model User { id: uuid @primary }
model ApiError { message: string }
model Envelope { payload: Page<Result<User, ApiError>> }
`,
			wantKeys: map[string]string{
				"acme.v1.Page<acme.v1.Result<acme.v1.User, acme.v1.ApiError>>": "PageResultUserApiError",
				"acme.v1.Result<acme.v1.User, acme.v1.ApiError>":               "ResultUserApiError",
			},
		},
		{
			name: "open type parameter skips indirect instantiation",
			src: `
namespace acme.v1
model Page<T> { data: T[] }
# Note Box<U>#value is an open type, so Box<User> is not a concrete usage of Page<T> and Page<User> should not be monomorphized.
model Box<U> { value: Page<U> }
model User { id: uuid @primary }
model WrappedUser { value: Box<User> }
`,
			wantKeys: map[string]string{
				"acme.v1.Box<acme.v1.User>": "BoxUser",
			},
			wantAbsent: []string{"acme.v1.Page<acme.v1.User>"},
		},
		{
			name: "direct reference produces monomorph, indirect does not",
			src: `
namespace acme.v1

model Page<T> { data: T[] }
model Result<T, E> {
	data: T?
	err: E?
}
model User { id: uuid @primary }
model ApiError { message: string }

# Directly reference Result
model Response { result: Result<User, ApiError> }

# Box<User> indirectly uses Page<User> via generic parameter substitution,
# but Page<User> is never directly referenced, so no monomorph exists.
model Box<U> { value: Page<U> }
model WrappedResults { box: Box<User> }
`,
			wantKeys: map[string]string{
				"acme.v1.Result<acme.v1.User, acme.v1.ApiError>": "ResultUserApiError",
				"acme.v1.Box<acme.v1.User>":                      "BoxUser",
			},
			wantAbsent: []string{"acme.v1.Page<acme.v1.User>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, r := resolve(t, parseFile(t, tt.src))
			assertNoErrors(t, r)
			assert.Equal(t, len(tt.wantKeys), len(resolved.monomorphs))
			for key, wantName := range tt.wantKeys {
				m, ok := resolved.monomorphs[key]
				assert.True(t, ok, "expected monomorph for key %q", key)
				assert.Equal(t, wantName, m.Name)
			}
			for _, key := range tt.wantAbsent {
				_, ok := resolved.monomorphs[key]
				assert.False(t, ok, "expected no monomorph for key %q", key)
			}
		})
	}
}

func TestMonomorphEmitAs(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		caser *cases.Caser
		want  []string
	}{
		{
			name:  "default PascalCase caser",
			key:   "acme.v1.Page<acme.v1.User>",
			caser: nil,
			want:  []string{"Page", "User"},
		},
		{
			name:  "lowercase caser",
			key:   "acme.v1.Page<acme.v1.User>",
			caser: new(cases.Lower(language.English)),
			want:  []string{"page", "user"},
		},
		{
			name:  "uppercase caser",
			key:   "acme.v1.Page<acme.v1.User>",
			caser: new(cases.Upper(language.English)),
			want:  []string{"PAGE", "USER"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mono := new(Monomorph{key: tt.key})

			var got string
			var err error
			if tt.caser == nil {
				got, err = mono.EmitAs()
			} else {
				got, err = mono.EmitAs(*tt.caser)
			}

			assert.NoError(t, err)
			for _, substr := range tt.want {
				assert.Contains(t, got, substr, "expected %q in result %q", substr, got)
			}
		})
	}
}

func TestMonomorphEmitAsMemoization(t *testing.T) {
	mono := &Monomorph{key: "acme.v1.Page<acme.v1.User>"}

	result1, err := mono.EmitAs()
	assert.NoError(t, err)

	result2, err := mono.EmitAs()
	assert.NoError(t, err)

	assert.Equal(t, result1, result2)
	assert.Equal(t, 1, len(mono.memoCache)) // = 1

	// different caser, different result, should memoize a second caser
	caser := cases.Lower(language.English)
	result3, err := mono.EmitAs(caser)
	assert.NoError(t, err)

	assert.NotEqual(t, result1, result3)
	assert.Equal(t, 2, len(mono.memoCache)) // = 2

	result4, err := mono.EmitAs(caser)
	assert.NoError(t, err)

	assert.Equal(t, result3, result4)
	assert.Equal(t, 2, len(mono.memoCache)) // still = 2
}

func TestMonomorphEmitAsMultipleCasersError(t *testing.T) {
	mono := new(Monomorph{key: "acme.v1.Page<acme.v1.User>"})

	_, err := mono.EmitAs(
		cases.Lower(language.English),
		cases.Upper(language.English),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected at most one caser")
}

func TestMonomorphPrefixedWithNamespace(t *testing.T) {
	tests := []struct {
		name             string
		monomorph        Monomorph
		wantPrefixedName string
		wantOriginalName string
	}{
		{
			name: "with namespace",
			monomorph: Monomorph{
				Name:      "PageUser_abc123",
				Namespace: "acme.v1",
			},
			wantPrefixedName: "acme.v1.PageUser_abc123",
			wantOriginalName: "PageUser_abc123",
		},
		{
			name: "without namespace",
			monomorph: Monomorph{
				Name:      "PageUser_abc123",
				Namespace: "",
			},
			wantPrefixedName: "PageUser_abc123",
			wantOriginalName: "PageUser_abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantOriginalName, tt.monomorph.Name)
			prefixed := tt.monomorph.PrefixedWithNamespace()
			assert.Equal(t, tt.wantPrefixedName, prefixed.Name)
			assert.Equal(t, tt.wantOriginalName, tt.monomorph.Name)
		})
	}
}

func TestStripNamespaceFromKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "simple type with matching namespace",
			key:  "acme.v1.Page<acme.v1.User>",
			want: "Page<User>",
		},
		{
			name: "nested generics with matching namespace",
			key:  "acme.v1.Page<acme.v1.Result<acme.v1.User, acme.v1.ApiError>>",
			want: "Page<Result<User, ApiError>>",
		},
		{
			name: "mixed namespaces - strips all namespaces",
			key:  "acme.v1.Page<other.v1.User>",
			want: "Page<User>",
		},
		{
			name: "multiple args with mixed namespaces",
			key:  "acme.v1.Result<acme.v1.User, other.v1.ApiError>",
			want: "Result<User, ApiError>",
		},
		{
			name: "no generic args",
			key:  "acme.v1.User",
			want: "User",
		},
		{
			name: "type from different namespace",
			key:  "other.v2.User",
			want: "User",
		},
		{
			name: "deeply nested generics with matching namespace",
			key:  "acme.v1.Outer<acme.v1.Middle<acme.v1.Inner<acme.v1.Core>>>",
			want: "Outer<Middle<Inner<Core>>>",
		},
		{
			name: "three args with mixed namespaces",
			key:  "acme.v1.Triple<acme.v1.A, other.v1.B, acme.v1.C>",
			want: "Triple<A, B, C>",
		},
		{
			name: "namespace with single segment",
			key:  "base.Model<base.Type>",
			want: "Model<Type>",
		},
		{
			name: "type with no namespace",
			key:  "Page<User>",
			want: "Page<User>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mono := &Monomorph{}
			got := mono.stripNamespaceFromKey(tt.key)
			assert.Equal(t, tt.want, got, "stripNamespaceFromKey(%q) = %q, want %q", tt.key, got, tt.want)
		})
	}
}

func TestMonomorphEmitAsPrefixedWithNamespace(t *testing.T) {
	tests := []struct {
		name       string
		monomorph  Monomorph
		caser      *cases.Caser
		wantEmitAs string
	}{
		{
			name: "default caser",
			monomorph: Monomorph{
				key:       "acme.v1.Page<acme.v1.User>",
				Name:      "PageUser",
				Namespace: "acme.v1",
			},
			caser:      nil,
			wantEmitAs: "AcmeV1PageAcmeV1User",
		},
		{
			name: "lowercase caser",
			monomorph: Monomorph{
				key:       "acme.v1.Page<acme.v1.User>",
				Name:      "PageUser",
				Namespace: "acme.v1",
			},
			caser:      new(cases.Lower(language.English)),
			wantEmitAs: "acmev1pageacmev1user",
		},
		{
			name: "uppercase caser",
			monomorph: Monomorph{
				key:       "acme.v1.Page<acme.v1.User>",
				Name:      "PageUser",
				Namespace: "acme.v1",
			},
			caser:      new(cases.Upper(language.English)),
			wantEmitAs: "ACMEV1PAGEACMEV1USER",
		},
		{
			name: "nested generics",
			monomorph: Monomorph{
				key:       "acme.v1.Page<acme.v1.Result<acme.v1.User, acme.v1.ApiError>>",
				Name:      "PageResultUserApiError",
				Namespace: "acme.v1",
			},
			caser:      nil,
			wantEmitAs: "AcmeV1PageAcmeV1ResultAcmeV1UserAcmeV1ApiError",
		},
		{
			name: "no namespace",
			monomorph: Monomorph{
				key:       "Page<User>",
				Name:      "PageUser",
				Namespace: "",
			},
			caser:      nil,
			wantEmitAs: "PageUser",
		},
		{
			name: "multi-part namespace, lowercase",
			monomorph: Monomorph{
				key:       "org.example.api.Response<org.example.api.Data>",
				Name:      "ResponseData",
				Namespace: "org.example.api",
			},
			caser:      new(cases.Lower(language.English)),
			wantEmitAs: "orgexampleapiresponseorgexampleapidata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefixed := tt.monomorph.PrefixedWithNamespace()
			var got string
			var err error
			if tt.caser == nil {
				got, err = prefixed.EmitAs()
			} else {
				got, err = prefixed.EmitAs(*tt.caser)
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantEmitAs, got, "EmitAs() on PrefixedWithNamespace() = %q, want %q", got, tt.wantEmitAs)
		})
	}
}
