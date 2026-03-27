package resolver

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/parser"
)

func parseFile(t *testing.T, src string) *ast.Stencil {
	t.Helper()
	p, err := parser.New()
	assert.NoError(t, err)
	stencil, err := p.Parse(src)
	assert.NoError(t, err)
	return stencil
}

func resolve(t *testing.T, files ...*ast.Stencil) (*ResolvedSchema, *Resolver) {
	t.Helper()
	r := New(files...)
	prog, _ := r.Resolve()
	return prog, r
}

func assertNoErrors(t *testing.T, r *Resolver) {
	t.Helper()
	for _, e := range r.errors {
		t.Errorf("resolver error: %s", e)
	}
}

func assertError(t *testing.T, r *Resolver, containing string) {
	t.Helper()
	for _, e := range r.errors {
		if strings.Contains(e.Error(), containing) {
			return
		}
	}
	t.Errorf("expected error containing %q, got: %v", containing, r.errors)
}

func TestCollectSimpleSymbols(t *testing.T) {
	file := parseFile(t, `
namespace acme.users.v1

enum UserRole {
  admin
  member
}

model User {
  id:   uuid   @primary
  role: UserRole
}

input CreateUserInput {
  role: UserRole
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	for _, name := range []string{
		"acme.users.v1.UserRole",
		"acme.users.v1.User",
		"acme.users.v1.CreateUserInput",
	} {
		if _, ok := resolved.definitions[name]; !ok {
			t.Errorf("expected symbol %q in global table", name)
		}
	}
}

func TestDuplicateDeclaration(t *testing.T) {
	file1 := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}
`)
	file2 := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}
`)
	_, r := resolve(t, file1, file2)
	assertError(t, r, "duplicate type definition: 'acme.v1.User'")
}

func TestNoNamespace(t *testing.T) {
	file := parseFile(t, `
model Post {
  id: uuid @primary
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)
	if _, ok := resolved.definitions["Post"]; !ok {
		t.Error("expected symbol 'Post' without namespace prefix")
	}
}

func TestImportResolution(t *testing.T) {
	common := parseFile(t, `
namespace acme.common.v1

model Page<T> {
  data:  T[]
  total: int
}
`)
	users := parseFile(t, `
namespace acme.users.v1

import acme.common.v1 { Page }

model User {
  id: uuid @primary
}
`)
	resolved, r := resolve(t, common, users)
	assertNoErrors(t, r)

	if _, ok := resolved.Definition("acme.common.v1.Page"); !ok {
		t.Error("expected Page to be defined at acme.common.v1.Page")
	}
}

func TestImportUnknownSymbol(t *testing.T) {
	file := parseFile(t, `
namespace acme.users.v1

import acme.common.v1 { DoesNotExist }

model User {
  id: uuid @primary
}
`)
	_, r := resolve(t, file)
	assertError(t, r, "cannot resolve import: 'DoesNotExist' from 'acme.common.v1'")
}

func TestImportConflict(t *testing.T) {
	common := parseFile(t, `
namespace acme.common.v1

model User {
  id: uuid @primary
}
`)
	users := parseFile(t, `
namespace acme.users.v1

import acme.common.v1 { User }

model User {
  id: uuid @primary
}
`)
	_, r := resolve(t, common, users)
	assertError(t, r, "import 'User' conflicts with locally defined type 'User'")
}

func TestUnknownType(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model Post {
  author: UnknownType
}
`)
	_, r := resolve(t, file)
	assertError(t, r, "unknown type: 'UnknownType'")
}

func TestKnownTypeResolves(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

enum Status { active, inactive }

model User {
  id:     uuid   @primary
  status: Status
}
`)
	_, r := resolve(t, file)
	assertNoErrors(t, r)
}

func TestBuiltinScalarsAlwaysResolve(t *testing.T) {
	file := parseFile(t, `
model Everything {
  a: string
  b: int
  c: float
  d: boolean
  e: uuid
  f: timestamp
  g: date
  h: any
}
`)
	_, r := resolve(t, file)
	assertNoErrors(t, r)
}

func TestGenericArityMismatch(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model Page<T> {
  data:  T[]
  total: int
}

model Bad {
  items: Page
}
`)
	_, r := resolve(t, file)
	assertError(t, r, "type 'Page' requires 1 type argument(s), got 0")
}

func TestGenericArityCorrect(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model Page<T> {
  data:  T[]
  total: int
}

model User {
  id: uuid @primary
}

model UserList {
  page: Page<User>
}
`)
	_, r := resolve(t, file)
	assertNoErrors(t, r)
}

func TestInputDisallowsModelDecorators(t *testing.T) {
	p, err := parser.New()
	assert.NoError(t, err)
	_, err = p.Parse(`
input BadInput {
  id: uuid @primary
}
`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid decorator \"primary\" for input field \"id\" [line: 3, col: 11] (input supports: @default, @raw)")
}

func TestCrossFileTypeResolution(t *testing.T) {
	common := parseFile(t, `
namespace acme.common.v1

model ApiError {
  code:    string
  message: string
}
`)
	users := parseFile(t, `
namespace acme.users.v1

import acme.common.v1 { ApiError }

model UserResponse {
  error: ApiError?
}
`)
	_, r := resolve(t, common, users)
	assertNoErrors(t, r)
}

func TestDefinitionByFQN(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}

enum Status { active, inactive }
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	userDef, ok := resolved.Definition("acme.v1.User")
	if !ok {
		t.Fatal("expected to find User definition")
	}

	if _, isModel := userDef.(*ast.Model); !isModel {
		t.Errorf("expected User to be a Model, got %T", userDef)
	}

	statusDef, ok := resolved.Definition("acme.v1.Status")
	if !ok {
		t.Fatal("expected to find Status definition")
	}

	if _, isEnum := statusDef.(*ast.Enum); !isEnum {
		t.Errorf("expected Status to be an Enum, got %T", statusDef)
	}
}

func TestDefinitionNotFound(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	_, ok := resolved.Definition("acme.v1.NonExistent")
	if ok {
		t.Error("expected Definition to return false for non-existent type")
	}
}

func TestResolveModel(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id:   uuid @primary
  name: string
}

model Post {
  author: User
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	postDefNode, ok := resolved.Definition("acme.v1.Post")
	if !ok {
		t.Fatal("expected to find Post definition")
	}
	post := postDefNode.(*ast.Model)
	authorFieldType := &post.Fields[0].Type

	model, err := resolved.ResolveModel(authorFieldType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.Name.Value != "User" {
		t.Errorf("expected model name 'User', got %q", model.Name.Value)
	}

	if len(model.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(model.Fields))
	}
}

func TestResolveModelUnresolved(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model Post {
  author: UnknownType
}
`)
	r := New(file)
	resolved, err := r.Resolve()

	if err == nil {
		t.Error("expected resolver to return error for unknown type during linking")
	}

	if resolved != nil {
		t.Error("expected resolved to be nil when there are resolution errors")
	}
}

func TestResolveModelTypeAssertionFails(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

enum Status { active }

model User {
  id: uuid @primary
  status: Status
}

model Post {
  userStatus: Status
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	postDefNode, ok := resolved.Definition("acme.v1.Post")
	if !ok {
		t.Fatal("expected to find Post definition")
	}
	post := postDefNode.(*ast.Model)
	statusFieldType := &post.Fields[0].Type

	_, err := resolved.ResolveModel(statusFieldType)
	if err == nil {
		t.Error("expected error when resolving enum as model")
	} else if !strings.Contains(err.Error(), "expected 'Status' to be a model") {
		t.Errorf("expected type assertion error, got: %v", err)
	}
}

func TestResolveEnum(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

enum Status {
  active
  inactive
  archived
}

model User {
  id:     uuid @primary
  status: Status
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	userDefNode, ok := resolved.Definition("acme.v1.User")
	if !ok {
		t.Fatal("expected to find User definition")
	}
	user := userDefNode.(*ast.Model)
	statusFieldType := &user.Fields[1].Type

	enum, err := resolved.ResolveEnum(statusFieldType)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if enum.Name.Value != "Status" {
		t.Errorf("expected enum name 'Status', got %q", enum.Name.Value)
	}

	if len(enum.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(enum.Elements))
	}
}

func TestResolveEnumTypeAssertionFails(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}

enum Status { active }

model Post {
  owner: User
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	postDefNode, ok := resolved.Definition("acme.v1.Post")
	if !ok {
		t.Fatal("expected to find Post definition")
	}
	post := postDefNode.(*ast.Model)
	ownerFieldType := &post.Fields[0].Type

	_, err := resolved.ResolveEnum(ownerFieldType)
	if err == nil {
		t.Error("expected error when resolving model as enum")
	} else if !strings.Contains(err.Error(), "expected 'User' to be an enum") {
		t.Errorf("expected type assertion error, got: %v", err)
	}
}

func TestResolveInput(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

input CreateUserInput {
  name:  string
  email: string
}

model User {
  id: uuid @primary
}

api Users @style(rest) {
  GET /users -> User[]
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	inputDefNode, ok := resolved.Definition("acme.v1.CreateUserInput")
	if !ok {
		t.Fatal("expected to find CreateUserInput definition")
	}
	input := inputDefNode.(*ast.Input)

	if input.Name.Value != "CreateUserInput" {
		t.Errorf("expected input name 'CreateUserInput', got %q", input.Name.Value)
	}

	if len(input.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(input.Fields))
	}
}

func TestResolveInputTypeAssertionFails(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}

input CreateUserInput {
  name: string
}

model Post {
  owner: User
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	postDefNode, ok := resolved.Definition("acme.v1.Post")
	if !ok {
		t.Fatal("expected to find Post definition")
	}
	post := postDefNode.(*ast.Model)
	ownerFieldType := &post.Fields[0].Type

	_, err := resolved.ResolveInput(ownerFieldType)
	if err == nil {
		t.Error("expected error when resolving model as input")
	} else if !strings.Contains(err.Error(), "expected 'User' to be an input") {
		t.Errorf("expected type assertion error, got: %v", err)
	}
}

func TestResolveApi(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}

api Users @style(rest) {
  GET /users -> User[]
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	apiDefNode, ok := resolved.Definition("acme.v1.Users")
	if !ok {
		t.Fatal("expected to find Users api definition")
	}

	api := apiDefNode.(*ast.Api)
	if api.Name.Value != "Users" {
		t.Errorf("expected api name 'Users', got %q", api.Name.Value)
	}
}

func TestResolveApiTypeAssertionFails(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}

model Post {
  owner: User
}

api Users @style(rest) {
  GET /users -> User[]
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	postDefNode, ok := resolved.Definition("acme.v1.Post")
	if !ok {
		t.Fatal("expected to find Post definition")
	}
	post := postDefNode.(*ast.Model)
	ownerFieldType := &post.Fields[0].Type

	_, err := resolved.ResolveApi(ownerFieldType)
	if err == nil {
		t.Error("expected error when resolving model as api")
	} else if !strings.Contains(err.Error(), "expected 'User' to be an api") {
		t.Errorf("expected type assertion error, got: %v", err)
	}
}

func TestResolveTypeWithNullTypeExpr(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model User {
  id: uuid @primary
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	scalarExpr := &ast.TypeExpression{
		Base: ast.QualifiedIdent{
			Parts: []string{"string"},
		},
	}

	node, ok := resolved.ResolveType(scalarExpr)
	if ok {
		t.Error("expected ResolveType to return false for scalar types")
	}
	if node != nil {
		t.Error("expected nil node for scalar types")
	}
}

func TestImportWithMultipleSymbols(t *testing.T) {
	common := parseFile(t, `
namespace acme.common.v1

model Page<T> {
  data:  T[]
  total: int
}

model Error {
  code:    string
  message: string
}

enum Status { ok, error }
`)
	users := parseFile(t, `
namespace acme.users.v1

import acme.common.v1 { Page, Error, Status }

model User {
  id: uuid @primary
}
`)
	resolved, r := resolve(t, common, users)
	assertNoErrors(t, r)

	_, ok := resolved.Definition("acme.common.v1.Page")
	if !ok {
		t.Error("expected Page to be defined")
	}

	_, ok = resolved.Definition("acme.common.v1.Error")
	if !ok {
		t.Error("expected Error to be defined")
	}

	_, ok = resolved.Definition("acme.common.v1.Status")
	if !ok {
		t.Error("expected Status to be defined")
	}
}

func TestNamespaceOf(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		typeName  string
		wantNS    string
		wantFound bool
	}{
		{
			name: "explicit namespace",
			src: `
namespace acme.users.v1

model User {
  id: uuid @primary
}
`,
			typeName:  "acme.users.v1.User",
			wantNS:    "acme.users.v1",
			wantFound: true,
		},
		{
			name: "implicit (default) namespace",
			src: `
model Widget {
  id: uuid @primary
}
`,
			typeName:  "Widget",
			wantNS:    "",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := parseFile(t, tt.src)
			resolved, r := resolve(t, file)
			assertNoErrors(t, r)

			node, ok := resolved.Definition(tt.typeName)
			assert.True(t, ok)

			ns, found := resolved.NamespaceOf(node)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantNS, ns)
		})
	}
}

func TestNamespaceOfUnregistered(t *testing.T) {
	file := parseFile(t, `
namespace acme.v1

model Foo {
  id: uuid @primary
}
`)
	resolved, r := resolve(t, file)
	assertNoErrors(t, r)

	_, found := resolved.NamespaceOf(&ast.Model{})
	assert.False(t, found)
}
