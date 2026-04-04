package schema

import (
	"slices"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

func parseAndResolve(t *testing.T, sources ...string) *resolver.ResolvedSchema {
	t.Helper()
	p, err := parser.New()
	assert.NoError(t, err)

	stencils := make([]*ast.Stencil, 0, len(sources))
	for _, src := range sources {
		s, err := p.Parse(src)
		assert.NoError(t, err)
		stencils = append(stencils, s)
	}

	r := resolver.New(stencils...)
	schema, err := r.Resolve()
	assert.NoError(t, err)
	return schema
}

func TestScalar(t *testing.T) {
	schema := parseAndResolve(t, `namespace test`)
	b := NewBuilder(*schema)

	tests := []struct {
		name       string
		scalar     string
		wantType   string
		wantFormat string
	}{
		{
			name:     "string",
			scalar:   "string",
			wantType: "string",
		},
		{
			name:     "int",
			scalar:   "int",
			wantType: "integer",
		},
		{
			name:       "float",
			scalar:     "float",
			wantType:   "number",
			wantFormat: "float",
		},
		{
			name:     "boolean",
			scalar:   "boolean",
			wantType: "boolean",
		},
		{
			name:       "uuid",
			scalar:     "uuid",
			wantType:   "string",
			wantFormat: "uuid",
		},
		{
			name:       "timestamp",
			scalar:     "timestamp",
			wantType:   "string",
			wantFormat: "date-time",
		},
		{
			name:       "date",
			scalar:     "date",
			wantType:   "string",
			wantFormat: "date",
		},
		{
			name:   "any produces empty type",
			scalar: "any",
		},
		{
			name:     "unknown defaults to string",
			scalar:   "foobar",
			wantType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.Scalar(tt.scalar)
			assert.Equal(t, tt.wantType, got.Type)
			assert.Equal(t, tt.wantFormat, got.Format)
		})
	}
}

func TestEnum(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		enumName string
		wantEnum []string
	}{
		{
			name: "basic enum",
			src: `
namespace test

enum Role {
  admin
  member
  guest
}
`,
			enumName: "test.Role",
			wantEnum: []string{"admin", "member", "guest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseAndResolve(t, tt.src)
			b := NewBuilder(*schema)

			node, ok := schema.Definition(tt.enumName)
			assert.True(t, ok)
			enum := node.(*ast.Enum)

			got := b.Enum(enum)
			assert.Equal(t, "string", got.Type)
			assert.Equal(t, len(tt.wantEnum), len(got.Enum))
			assert.Equal(t, tt.wantEnum, got.Enum)
		})
	}
}

func TestTypeExpression(t *testing.T) {
	schema := parseAndResolve(t, `namespace test`)

	tests := []struct {
		name               string
		expr               *ast.TypeExpression
		nullableStrategy   NullableStrategy
		wantType           string
		wantFormat         string
		wantNullable       bool
		wantAnyOfLen       int
		wantArrayItemType  string
		wantAnyOfFirstType string
	}{
		{
			name: "scalar string",
			expr: &ast.TypeExpression{
				Base: ast.QualifiedIdent{Parts: []string{"string"}},
			},
			wantType: "string",
		},
		{
			name: "scalar array",
			expr: &ast.TypeExpression{
				Base:    ast.QualifiedIdent{Parts: []string{"int"}},
				IsArray: true,
			},
			wantType:          "array",
			wantArrayItemType: "integer",
		},
		{
			name: "optional scalar with anyOf",
			expr: &ast.TypeExpression{
				Base:       ast.QualifiedIdent{Parts: []string{"string"}},
				IsOptional: true,
			},
			nullableStrategy:   NullableAnyOf,
			wantAnyOfLen:       2,
			wantAnyOfFirstType: "string",
		},
		{
			name: "optional scalar with nullable keyword",
			expr: &ast.TypeExpression{
				Base:       ast.QualifiedIdent{Parts: []string{"boolean"}},
				IsOptional: true,
			},
			nullableStrategy: NullableKeyword,
			wantType:         "boolean",
			wantNullable:     true,
		},
		{
			name: "optional array",
			expr: &ast.TypeExpression{
				Base:       ast.QualifiedIdent{Parts: []string{"string"}},
				IsArray:    true,
				IsOptional: true,
			},
			wantAnyOfLen:       2,
			wantAnyOfFirstType: "array",
		},
		{
			name:     "nil expression",
			expr:     nil,
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder(*schema)
			if tt.nullableStrategy > 0 {
				b = b.WithNullableStrategy(tt.nullableStrategy)
			}

			got := b.TypeExpression(tt.expr)

			if tt.expr == nil {
				assert.Equal(t, (*Schema)(nil), got)
				return
			}

			if tt.wantType != "" {
				assert.Equal(t, tt.wantType, got.Type)
			}
			if tt.wantFormat != "" {
				assert.Equal(t, tt.wantFormat, got.Format)
			}
			if tt.wantNullable {
				assert.True(t, got.Nullable)
			}
			if tt.wantAnyOfLen > 0 {
				assert.Equal(t, tt.wantAnyOfLen, len(got.AnyOf))
				if tt.wantAnyOfFirstType != "" {
					assert.Equal(t, tt.wantAnyOfFirstType, got.AnyOf[0].Type)
				}
				assert.Equal(t, "null", got.AnyOf[1].Type)
			}
			if tt.wantArrayItemType != "" {
				assert.NotEqual(t, (*Schema)(nil), got.Items)
				assert.Equal(t, tt.wantArrayItemType, got.Items.Type)
			}
		})
	}
}

func TestModel(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

enum Role {
  admin
  member
}

model User {
  id:    uuid   @primary
  name:  string
  email: string?
  role:  Role   @default(member)
}
`)
	b := NewBuilder(*schema)

	node, ok := schema.Definition("test.User")
	assert.True(t, ok)
	model := node.(*ast.Model)

	got := b.Model(model)
	assert.Equal(t, "object", got.Type)

	assert.True(t, slices.Contains(got.Required, "id"), "required should include non-optional id")
	assert.True(t, slices.Contains(got.Required, "name"), "required should include non-optional name")
	assert.False(t, slices.Contains(got.Required, "email"), "email should be optional")
	assert.True(t, slices.Contains(got.Required, "role"), "required should include non-optional role")

	assert.NotEqual(t, (*Schema)(nil), got.Properties["id"], "property should exist for id")
	assert.NotEqual(t, (*Schema)(nil), got.Properties["name"], "property should exist for name")
	assert.NotEqual(t, (*Schema)(nil), got.Properties["email"], "property should exist for email")
	assert.NotEqual(t, (*Schema)(nil), got.Properties["role"], "property should exist for role")

	assert.Equal(t, "string", got.Properties["id"].Type, "id should be type string")
	assert.Equal(t, "uuid", got.Properties["id"].Format, "id should be format uuid")
	assert.Equal(t, "string", got.Properties["name"].Type, "name should be type string")

	assert.Equal(t, 2, len(got.Properties["email"].AnyOf), "email is optional; should be anyOf with type email or null")
	assert.Equal(t, got.Properties["email"].AnyOf[0].Type, "string", "email anyOf first type should be string")
	assert.Equal(t, got.Properties["email"].AnyOf[1].Type, "null", "email anyOf second type should be null")

	assert.Equal(t, "string", got.Properties["role"].Type, "role should be type string")
	assert.Equal(t, []string{"admin", "member"}, got.Properties["role"].Enum, "role should have inline enum values")

	assert.Equal(t, "member", got.Properties["role"].Default, "role should have defined default=member")
}

func TestModel_with_generic_params(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

model Page<T> {
  items: T[]
  total: int
}
`)
	b := NewBuilder(*schema)

	node, ok := schema.Definition("test.Page")
	assert.True(t, ok)
	model := node.(*ast.Model)

	got := b.Model(model)
	assert.Equal(t, "object", got.Type)

	items := got.Properties["items"]
	assert.NotEqual(t, (*Schema)(nil), items)
	assert.Equal(t, "array", items.Type, "items should be an array")
	assert.Equal(t, items.Items, &Schema{}, "items.Items should have empty Schema since T is not concrete here")

	assert.Equal(t, "integer", got.Properties["total"].Type, "total should be an integer")
}

func TestModel_ref_strategy(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

model Address {
  street: string
}

model User {
  name:    string
  address: Address
}
`)

	tests := []struct {
		name     string
		strategy RefStrategy
		wantRef  string
	}{
		{
			name:     "defs strategy",
			strategy: RefDefs,
			wantRef:  "#/$defs/Address",
		},
		{
			name:     "components strategy",
			strategy: RefComponents,
			wantRef:  "#/components/schemas/Address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder(*schema).WithRefStrategy(tt.strategy)
			node, ok := schema.Definition("test.User")
			assert.True(t, ok)
			got := b.Model(node.(*ast.Model))
			assert.Equal(t, tt.wantRef, got.Properties["address"].Ref)
		})
	}
}

func TestInput(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

input CreateUserInput {
  email: string
  name:  string?
}
`)
	b := NewBuilder(*schema)

	node, ok := schema.Definition("test.CreateUserInput")
	assert.True(t, ok)
	input := node.(*ast.Input)

	got := b.Input(input)
	assert.Equal(t, "object", got.Type)
	assert.True(t, slices.Contains(got.Required, "email"))
	assert.False(t, slices.Contains(got.Required, "name"))
	assert.Equal(t, "string", got.Properties["email"].Type)

	assert.Equal(t, 2, len(got.Properties["name"].AnyOf), "name should be optional, anyOf type string and null")
	assert.Equal(t, "string", got.Properties["name"].AnyOf[0].Type)
	assert.Equal(t, "null", got.Properties["name"].AnyOf[1].Type)
}

func TestApi_returns_nil(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

api TestApi @style(rest) {
  GET / -> void
}
`)
	b := NewBuilder(*schema)

	node, ok := schema.Definition("test.TestApi")
	assert.True(t, ok)
	api := node.(*ast.Api)
	assert.Equal(t, (*Schema)(nil), b.Api(api))
}

func TestFieldDecorator(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		modelName string
		fieldName string
		checkFunc func(t *testing.T, s *Schema)
	}{
		{
			name: "deprecated decorator",
			src: `
namespace test

model Legacy {
  oldField: string @deprecated
}
`,
			modelName: "test.Legacy",
			fieldName: "oldField",
			checkFunc: func(t *testing.T, s *Schema) {
				_, hasDeprecated := s.Extensions["x-deprecated"]
				assert.True(t, hasDeprecated)
			},
		},
		{
			name: "updatedAt decorator",
			src: `
namespace test

model Tracked {
  updatedAt: timestamp @updatedAt
}
`,
			modelName: "test.Tracked",
			fieldName: "updatedAt",
			checkFunc: func(t *testing.T, s *Schema) {
				assert.True(t, s.ReadOnly)
			},
		},
		{
			name: "custom decorators get x- prefix",
			src: `
namespace test

model Entity {
  id:   string @primary
  name: string @unique
  ref:  string @relation(field: otherId)
}
`,
			modelName: "test.Entity",
			fieldName: "id",
			checkFunc: func(t *testing.T, s *Schema) {
				_, hasPrimary := s.Extensions["x-primary"]
				assert.True(t, hasPrimary, "expected x-primary extension")
				_, hasRawPrimary := s.Extensions["primary"]
				assert.False(t, hasRawPrimary, "should not have raw 'primary' key")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseAndResolve(t, tt.src)
			b := NewBuilder(*schema)
			node, _ := schema.Definition(tt.modelName)
			got := b.Model(node.(*ast.Model))
			tt.checkFunc(t, got.Properties[tt.fieldName])
		})
	}
}

func TestMonomorphSchema_single_param(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

model User {
  id:   uuid
  name: string
}

model Page<T> {
  items: T[]
  total: int
}

api TestApi @style(rest) {
  GET / -> Page<User>
}
`)
	b := NewBuilder(*schema)
	monos := schema.Monomorphs()
	assert.True(t, len(monos) > 0, "expected at least one monomorph")

	// find the Page<User> monomorph
	var pageUser *resolver.Monomorph
	for _, m := range monos {
		name, err := m.EmitAs()
		assert.NoError(t, err)
		if name == "PageUser" {
			pageUser = &m
			break
		}
	}
	assert.NotEqual(t, (*resolver.Monomorph)(nil), pageUser)

	got := b.MonomorphSchema(*pageUser)
	assert.NotEqual(t, (*Schema)(nil), got)
	assert.Equal(t, "object", got.Type)

	// items should be an array of $ref to User (T substituted with User)
	items := got.Properties["items"]
	assert.NotEqual(t, (*Schema)(nil), items)
	assert.Equal(t, "array", items.Type)
	assert.NotEqual(t, (*Schema)(nil), items.Items)
	assert.Equal(t, "#/$defs/User", items.Items.Ref)

	assert.Equal(t, "integer", got.Properties["total"].Type, "total should be an integer")

	assert.True(t, slices.Contains(got.Required, "items"), "items are required and non-optional")
	assert.True(t, slices.Contains(got.Required, "total"), "total is required and non-optioal")
}

func TestMonomorphSchema_multi_param(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

model User {
  id: uuid
}

model ApiError {
  code:    int
  message: string
}

model Result<T, E> {
  ok:    boolean
  data:  T?
  error: E?
}

api TestApi @style(rest) {
  GET / -> Result<User, ApiError>
}
`)
	b := NewBuilder(*schema)
	monos := schema.Monomorphs()

	var resultUserApiError *resolver.Monomorph
	for _, m := range monos {
		name, _ := m.EmitAs()
		if name == "ResultUserApiError" {
			resultUserApiError = &m
			break
		}
	}
	assert.NotEqual(t, (*resolver.Monomorph)(nil), resultUserApiError)

	got := b.MonomorphSchema(*resultUserApiError)
	assert.NotEqual(t, (*Schema)(nil), got)

	assert.Equal(t, "boolean", got.Properties["ok"].Type, "ok should be a boolean")

	data := got.Properties["data"]
	assert.NotEqual(t, (*Schema)(nil), data)
	assert.Equal(t, 2, len(data.AnyOf), "data is optional and should be anyOf with $ref to User and null")
	assert.Equal(t, "#/$defs/User", data.AnyOf[0].Ref)
	assert.Equal(t, "null", data.AnyOf[1].Type)

	errProp := got.Properties["error"]
	assert.NotEqual(t, (*Schema)(nil), errProp)
	assert.Equal(t, 2, len(errProp.AnyOf), "error is optional and should be anyOf with $ref to ApiError and null")
	assert.Equal(t, "#/$defs/ApiError", errProp.AnyOf[0].Ref)

	assert.True(t, slices.Contains(got.Required, "ok"), "ok should be required")
	assert.False(t, slices.Contains(got.Required, "data"), "data should be optional")
	assert.False(t, slices.Contains(got.Required, "error"), "error should be optional")
}

func TestMonomorphSchema_with_ref_components(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

model User {
  id: uuid
}

model Page<T> {
  items: T[]
}

api TestApi @style(rest) {
  GET / -> Page<User>
}
`)
	b := NewBuilder(*schema).WithRefStrategy(RefComponents)
	monos := schema.Monomorphs()

	for _, m := range monos {
		name, _ := m.EmitAs()
		if name == "PageUser" {
			got := b.MonomorphSchema(m)
			assert.Equal(t, "#/components/schemas/User", got.Properties["items"].Items.Ref)
			return
		}
	}
	t.Fatal("expected PageUser monomorph")
}

func TestMonomorphSchema_scalar_substitution(t *testing.T) {
	schema := parseAndResolve(t, `
namespace test

model Wrapper<T> {
  value: T
  label: string
}

model Container {
  inner: Wrapper<int>
}
`)
	b := NewBuilder(*schema)
	monos := schema.Monomorphs()

	for _, m := range monos {
		name, _ := m.EmitAs()
		if name == "WrapperInt" {
			got := b.MonomorphSchema(m)
			assert.NotEqual(t, (*Schema)(nil), got)
			assert.Equal(t, "integer", got.Properties["value"].Type, "T should have been substitued with integer")
			assert.Equal(t, "string", got.Properties["label"].Type, "label should have been a string")
			return
		}
	}
	t.Fatal("expected WrapperInt monomorph")
}

func TestMonomorphSchema_non_model_returns_nil(t *testing.T) {
	// construct a monomorph with a non-model Original (e.g. an enum)
	enum := ast.Enum{Name: ast.StringLiteral{Value: "Fake"}}
	var spec ast.SpecNode = &enum

	mono := resolver.Monomorph{
		Original: &spec,
	}

	schema := parseAndResolve(t, `namespace test`)
	b := NewBuilder(*schema)
	assert.Equal(t, (*Schema)(nil), b.MonomorphSchema(mono))
}

func TestModelField_generic_instantiation(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		modelName   string
		fieldName   string
		refStrategy RefStrategy
		wantRef     string
	}{
		{
			name: "generic instantiation refs monomorph",
			src: `
namespace test

model User {
  id: uuid
}

model Page<T> {
  items: T[]
  total: int
}

model Feed {
  page:  Page<User>
  title: string
}
`,
			modelName:   "test.Feed",
			fieldName:   "page",
			refStrategy: RefDefs,
			wantRef:     "#/$defs/PageUser",
		},
		{
			name: "generic instantiation with ref components",
			src: `
namespace test

model User {
  id: uuid
}

model Page<T> {
  items: T[]
}

model Feed {
  page: Page<User>
}
`,
			modelName:   "test.Feed",
			fieldName:   "page",
			refStrategy: RefComponents,
			wantRef:     "#/components/schemas/PageUser",
		},
		{
			name: "plain model uses model name",
			src: `
namespace test

model Address {
  street: string
}

model User {
  address: Address
}
`,
			modelName:   "test.User",
			fieldName:   "address",
			refStrategy: RefDefs,
			wantRef:     "#/$defs/Address",
		},
		{
			name: "multi-param generic instantiation",
			src: `
namespace test

model User {
  id: uuid
}

model ApiError {
  code: int
}

model Result<T, E> {
  ok:    boolean
  data:  T?
  error: E?
}

model UserResponse {
  result: Result<User, ApiError>
}
`,
			modelName:   "test.UserResponse",
			fieldName:   "result",
			refStrategy: RefDefs,
			wantRef:     "#/$defs/ResultUserApiError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseAndResolve(t, tt.src)
			b := NewBuilder(*schema).WithRefStrategy(tt.refStrategy)
			node, _ := schema.Definition(tt.modelName)
			got := b.Model(node.(*ast.Model))
			assert.Equal(t, tt.wantRef, got.Properties[tt.fieldName].Ref)
		})
	}
}
