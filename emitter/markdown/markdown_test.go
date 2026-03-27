package markdown

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/emitter"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

func parseFile(t *testing.T, src string) *ast.Stencil {
	t.Helper()
	p, err := parser.New()
	assert.NoError(t, err)
	stencil, err := p.Parse(src)
	assert.NoError(t, err)
	return stencil
}

func resolve(t *testing.T, files ...*ast.Stencil) (*resolver.ResolvedSchema, *resolver.Resolver) {
	t.Helper()
	r := resolver.New(files...)
	prog, _ := r.Resolve()
	return prog, r
}

func TestMarkdownEmitter(t *testing.T) {
	src := `
namespace acme.users.v2

type Email = string

enum UserRole {
  admin
  member
  guest
}

model User {
  id:        uuid      @primary
  email:     Email     @unique
  role:      UserRole  @default(member)
  name:      string?
  createdAt: timestamp @default(now)
  posts:     Post[]    @relation
}

model Post {
  id:       uuid   @primary
  title:    string
  body:     string?
  authorId: uuid
  author:   User   @relation(field: authorId)
}

input CreateUserInput {
  email: string
  name:  string?
  role:  UserRole @default(member)
}

input UpdateUserInput {
  name: string?
  role: UserRole?
}

model Result<T, E> {
  ok:    boolean
  data:  T?
  error: E?
}
	
model Page<T> {
  data:  T?
}
	
model ApiError {
  code:    int
  message: string
}

api UserService @version(2) @style(rest) {
  @basePath("/api/v2/users")
  @auth(bearer)

  GET  /      -> Page<User>
    @query(PaginationInput)

  GET  /:id   -> Result<User, ApiError>
    @errors(401, 404)

  POST /      -> User
    @body(CreateUserInput)
    @errors(400, 409)

  PATCH /:id  -> User
    @body(UpdateUserInput)
    @errors(400, 401, 404)

  DELETE /:id -> void
    @errors(401, 404)

  # Still-active v1 route
  GET /search -> User[] @version(1) @deprecated("Use GET / with query params")
}`
	stencil := parseFile(t, src)
	resolved, res := resolve(t, stencil)
	assert.NoError(t, res.Error())

	mdEmitter, err := New(resolved)
	assert.NoError(t, err)

	output, err := mdEmitter.EmitAll()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(output))
	md := output[0]

	assert.Equal(t, emitter.ContentText, md.ContentType())
	assert.Equal(t, "acme_users_v2.md", md.Filename())
	contents := string(md.Contents())

	// headers
	assert.Contains(t, contents, "## APIs")
	assert.Contains(t, contents, "## Enums")
	assert.Contains(t, contents, "## Models")
	assert.Contains(t, contents, "## Inputs")

	// sub-headers
	assert.Contains(t, contents, "### `UserService`")
	assert.Contains(t, contents, "### `UserRole`")
	assert.Contains(t, contents, "### `User`")
	assert.Contains(t, contents, "### `Post`")
	assert.Contains(t, contents, "### `Result<T, E>`")
	assert.Contains(t, contents, "### `Page<T>`")
	assert.Contains(t, contents, "### `ApiError`")
	assert.Contains(t, contents, "### `CreateUserInput`")
	assert.Contains(t, contents, "### `UpdateUserInput`")

	// section spot checks
	// API
	assert.Contains(t, contents, "- **style**: REST")
	assert.Contains(t, contents, "- **basePath**: \"/api/v2/users\"")
	assert.Contains(t, contents, "| GET | / |  | Page<User> | Query: PaginationInput |")
	assert.Contains(t, contents, "| GET | /search |  | User[] | Version: 1 · Deprecated: \"Use GET / with query params\" |")

	// enum
	assert.Contains(t, contents, "* member")

	// model
	assert.Contains(t, contents, "| role | `UserRole` | @default(member) |")
	assert.Contains(t, contents, "| author | `User` | @relation(field: authorId) |")
	assert.Contains(t, contents, "| data | `T?` |  |")

	// input
	assert.Contains(t, contents, "| role | `UserRole` | @default(member) |")
	assert.Contains(t, contents, "| role | `UserRole?` |  |")
}

func TestEmitOne(t *testing.T) {
	src := `
namespace acme.users.v2

enum UserRole {
  admin
  member
  guest
}

model User {
  id:   uuid     @primary
  role: UserRole @default(member)
  name: string?
}

input CreateUserInput {
  email: string
  name:  string?
}

api UserService @version(2) @style(rest) {
  @basePath("/api/v2/users")

  GET  /    -> User[]
  POST /    -> User
    @body(CreateUserInput)
}`

	tests := []struct {
		name         string
		typ          emitter.SpecType
		specName     string
		wantErr      bool
		wantFilename string
		wantContains []string
	}{
		{
			name:         "emits api by name",
			typ:          emitter.SpecApi,
			specName:     "UserService",
			wantFilename: "user_service.md",
			wantContains: []string{
				"### `UserService`",
				"- **style**: REST",
				"- **basePath**:",
				"| GET |",
				"| POST |",
			},
		},
		{
			name:         "emits enum by name",
			typ:          emitter.SpecEnum,
			specName:     "UserRole",
			wantFilename: "user_role.md",
			wantContains: []string{
				"### `UserRole`",
				"* admin",
				"* member",
				"* guest",
			},
		},
		{
			name:         "emits model by name",
			typ:          emitter.SpecModel,
			specName:     "User",
			wantFilename: "user.md",
			wantContains: []string{
				"### `User`",
				"| id | `uuid` | @primary |",
				"| role | `UserRole` | @default(member) |",
				"| name | `string?` |  |",
			},
		},
		{
			name:         "emits input by name",
			typ:          emitter.SpecInput,
			specName:     "CreateUserInput",
			wantFilename: "create_user_input.md",
			wantContains: []string{
				"### `CreateUserInput`",
				"| email | `string` |  |",
				"| name | `string?` |  |",
			},
		},
		{
			name:     "returns error for unknown name",
			typ:      emitter.SpecModel,
			specName: "DoesNotExist",
			wantErr:  true,
		},
	}

	stencil := parseFile(t, src)
	resolved, res := resolve(t, stencil)
	assert.NoError(t, res.Error())

	mdEmitter, err := New(resolved)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := mdEmitter.EmitOne(tt.typ, tt.specName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantFilename, output.Filename())
			assert.Equal(t, emitter.ContentText, output.ContentType())
			contents := string(output.Contents())
			for _, want := range tt.wantContains {
				assert.Contains(t, contents, want)
			}
		})
	}
}
