package plug

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

func mapFrom(t *testing.T, sources ...string) PluginSchema {
	t.Helper()

	p, err := parser.New()
	assert.NoError(t, err)

	stencils := make([]*ast.Stencil, 0, len(sources))
	for _, src := range sources {
		s, err := p.Parse(src)
		assert.NoError(t, err)
		stencils = append(stencils, s)
	}

	schema, err := resolver.New(stencils...).Resolve()
	assert.NoError(t, err)

	return NewMapper(schema).Map()
}

func TestMapper_Models(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result PluginSchema)
	}{
		{
			name: "basic model with scalar fields",
			input: `
namespace acme.v1

model User {
  id: string
  name: string
  age: int
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, "acme.v1", result.Namespace)
				assert.Equal(t, 1, len(result.Models))
				assert.Equal(t, "User", result.Models[0].Name)
				assert.Equal(t, "acme.v1", result.Models[0].Namespace)
				assert.Equal(t, 3, len(result.Models[0].Fields))
				assert.Equal(t, "id", result.Models[0].Fields[0].Name)
				assert.Equal(t, "string", result.Models[0].Fields[0].Type.FQN)
				assert.True(t, result.Models[0].Fields[0].Type.IsScalar)
				assert.Equal(t, "age", result.Models[0].Fields[2].Name)
				assert.Equal(t, "int", result.Models[0].Fields[2].Type.FQN)
				assert.True(t, result.Models[0].Fields[2].Type.IsScalar)
			},
		},
		{
			name: "array and optional field modifiers",
			input: `
namespace acme.v1

model Post {
  tags: string[]
  title: string?
  content: string
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 1, len(result.Models))
				m := result.Models[0]
				assert.True(t, m.Fields[0].Type.IsArray)
				assert.False(t, m.Fields[0].Type.IsOptional)
				assert.False(t, m.Fields[1].Type.IsArray)
				assert.True(t, m.Fields[1].Type.IsOptional)
				assert.False(t, m.Fields[2].Type.IsArray)
				assert.False(t, m.Fields[2].Type.IsOptional)
			},
		},
		{
			name: "model field referencing another model",
			input: `
namespace acme.v1

model Address {
  street: string
  city: string
}

model User {
  address: Address
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 2, len(result.Models))

				var user PluginModel
				for _, m := range result.Models {
					if m.Name == "User" {
						user = m
						break
					}
				}

				assert.Equal(t, 1, len(user.Fields))
				assert.Equal(t, "acme.v1.Address", user.Fields[0].Type.FQN)
				assert.False(t, user.Fields[0].Type.IsScalar)
			},
		},
		{
			name: "generic model produces monomorphs for concrete usages",
			input: `
namespace acme.v1

model Page<T> {
  items: T[]
  total: int
}

model User {
  id: string
}

model UserList {
  page: Page<User>
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.True(t, len(result.Monomorphs) > 0)

				var pageUser PluginMonomorph
				for _, m := range result.Monomorphs {
					if m.Name == "PageUser" {
						pageUser = m
						break
					}
				}

				assert.Equal(t, "PageUser", pageUser.Name)
				assert.Equal(t, "acme.v1.Page", pageUser.Original)
				assert.Equal(t, 1, len(pageUser.Args))
				assert.Equal(t, "acme.v1.User", pageUser.Args[0].FQN)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := mapFrom(t, tt.input)
			tt.validate(t, result)
		})
	}
}

func TestMapper_Enums(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result PluginSchema)
	}{
		{
			name: "basic enum",
			input: `
namespace acme.v1

enum Role {
  admin
  user
  guest
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 1, len(result.Enums))
				assert.Equal(t, "Role", result.Enums[0].Name)
				assert.Equal(t, "acme.v1", result.Enums[0].Namespace)
				assert.Equal(t, []string{"admin", "user", "guest"}, result.Enums[0].Values)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := mapFrom(t, tt.input)
			tt.validate(t, result)
		})
	}
}

func TestMapper_Inputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result PluginSchema)
	}{
		{
			name: "basic input",
			input: `
namespace acme.v1

input CreateUserInput {
  name: string
  email: string
  role: string?
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 1, len(result.Inputs))
				assert.Equal(t, "CreateUserInput", result.Inputs[0].Name)
				assert.Equal(t, "acme.v1", result.Inputs[0].Namespace)
				assert.Equal(t, 3, len(result.Inputs[0].Fields))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := mapFrom(t, tt.input)
			tt.validate(t, result)
		})
	}
}

func TestMapper_TypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result PluginSchema)
	}{
		{
			name: "scalar type aliases",
			input: `
namespace acme.v1

type Email = string
type UserId = int
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 2, len(result.Aliases["acme.v1"]))
				emailAlias := result.Aliases["acme.v1"][0]
				assert.Equal(t, "Email", emailAlias.Name)
				assert.Equal(t, "string", emailAlias.Type.FQN)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := mapFrom(t, tt.input)
			tt.validate(t, result)
		})
	}
}

func TestMapper_Apis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result PluginSchema)
	}{
		{
			name: "REST api routes",
			input: `
namespace acme.v1

model User {
  id: string
  name: string
}

api UserApi @style(rest) {
  GET /users -> User[]
  GET /users/:id -> User
  POST /users -> User
  DELETE /users/:id -> void
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 4, len(result.Apis))

				getAllRoute := result.Apis[0]
				assert.Equal(t, RouteStyleRest, getAllRoute.Style)
				assert.Equal(t, "GET /users", getAllRoute.Name)
				assert.Equal(t, "GET", string(getAllRoute.Method))
				assert.Equal(t, "/users", getAllRoute.Path)
				assert.Equal(t, "acme.v1.User", getAllRoute.Return.FQN)
				assert.True(t, getAllRoute.Return.IsArray)

				deleteRoute := result.Apis[3]
				assert.Equal(t, "DELETE", string(deleteRoute.Method))
				assert.Equal(t, "/users/:id", deleteRoute.Path)
				assert.Equal(t, "void", deleteRoute.Return.FQN)
			},
		},
		{
			name: "RPC api routes",
			input: `
namespace acme.v1

input CreateUserInput {
  name: string
}

model User {
  id: string
  name: string
}

api UserService @style(rpc) {
  rpc CreateUser -> User
  rpc stream GetUsers -> User[]
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 2, len(result.Apis))

				createRoute := result.Apis[0]
				assert.Equal(t, RouteStyleRpc, createRoute.Style)
				assert.Equal(t, "CreateUser", createRoute.Name)
				assert.Equal(t, "acme.v1.User", createRoute.Return.FQN)
				assert.False(t, createRoute.Streaming)

				streamRoute := result.Apis[1]
				assert.Equal(t, "GetUsers", streamRoute.Name)
				assert.True(t, streamRoute.Streaming)
				assert.Equal(t, "acme.v1.User", streamRoute.Return.FQN)
				assert.True(t, streamRoute.Return.IsArray)
			},
		},
		{
			name: "events api routes",
			input: `
namespace acme.v1

model UserEvent {
  userId: string
  action: string
}

api UserEvents @style(events) {
  publish UserCreated -> UserEvent
  subscribe UserUpdated -> UserEvent
}
`,
			validate: func(t *testing.T, result PluginSchema) {
				assert.Equal(t, 2, len(result.Apis))

				publishRoute := result.Apis[0]
				assert.Equal(t, RouteStyleEvents, publishRoute.Style)
				assert.Equal(t, "UserCreated", publishRoute.Name)
				assert.Equal(t, EventPublish, publishRoute.Direction)
				assert.Equal(t, "acme.v1.UserEvent", publishRoute.Return.FQN)

				subscribeRoute := result.Apis[1]
				assert.Equal(t, EventSubscribe, subscribeRoute.Direction)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := mapFrom(t, tt.input)
			tt.validate(t, result)
		})
	}
}

func TestMapper_EmptySchema(t *testing.T) {
	result := mapFrom(t, `
namespace acme.v1
`)
	assert.Equal(t, "acme.v1", result.Namespace)
	assert.Equal(t, 0, len(result.Models))
	assert.Equal(t, 0, len(result.Inputs))
	assert.Equal(t, 0, len(result.Enums))
	assert.Equal(t, 0, len(result.Apis))
}

func TestMapper_MultipleStencils(t *testing.T) {
	result := mapFrom(t,
		`
namespace acme.v1

model User {
  id: string
}
`,
		`
namespace acme.v2

model User {
  id: string
  name: string
}
`,
	)

	assert.Equal(t, 2, len(result.Models))

	nsCount := make(map[string]int)
	for _, model := range result.Models {
		nsCount[model.Namespace]++
	}

	assert.Equal(t, 1, nsCount["acme.v1"])
	assert.Equal(t, 1, nsCount["acme.v2"])
}

func TestMapper_Comments(t *testing.T) {
	result := mapFrom(t, `
namespace acme.v1

model User {
  # User identifier
  id: string
  name: string # full name
}
`)

	assert.Equal(t, 1, len(result.Models))
	assert.Contains(t, result.Models[0].Fields[0].HeadComment, "User identifier")
	assert.Contains(t, result.Models[0].Fields[1].LineComment, "full name")
}
