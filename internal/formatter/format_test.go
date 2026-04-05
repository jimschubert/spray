package formatter

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
)

func TestFormatEnum(t *testing.T) {
	tests := []struct {
		name string
		enum *ast.Enum
		opts []Options
		want string
	}{
		{
			name: "multiple elements",
			enum: &ast.Enum{
				Name:     ast.StringLiteral{Value: "UserRole"},
				Elements: []ast.StringLiteral{{Value: "admin"}, {Value: "member"}, {Value: "guest"}},
			},
			want: "enum UserRole {\n  admin\n  member\n  guest\n}\n",
		},
		{
			name: "single element condensed",
			enum: &ast.Enum{
				Name:     ast.StringLiteral{Value: "Singleton"},
				Elements: []ast.StringLiteral{{Value: "only"}},
			},
			want: "enum Singleton { only }\n",
		},
		{
			name: "single element not condensed",
			enum: &ast.Enum{
				Name:     ast.StringLiteral{Value: "Singleton"},
				Elements: []ast.StringLiteral{{Value: "only"}},
			},
			opts: []Options{WithAllowCondensedSpecs(false)},
			want: "enum Singleton {\n  only\n}\n",
		},
		{
			name: "with head comment",
			enum: &ast.Enum{
				Name:        ast.StringLiteral{Value: "Status"},
				Elements:    []ast.StringLiteral{{Value: "active"}, {Value: "inactive"}},
				HeadComment: commentGroup("# status of a thing\n"),
			},
			want: "# status of a thing\nenum Status {\n  active\n  inactive\n}\n",
		},
		{
			name: "with lines between members",
			enum: &ast.Enum{
				Name:     ast.StringLiteral{Value: "Color"},
				Elements: []ast.StringLiteral{{Value: "red"}, {Value: "green"}, {Value: "blue"}},
			},
			opts: []Options{WithLinesBetweenMembers(1)},
			want: "enum Color {\n  red\n\n  green\n\n  blue\n}\n",
		},
		{
			name: "with custom indent",
			enum: &ast.Enum{
				Name:     ast.StringLiteral{Value: "Size"},
				Elements: []ast.StringLiteral{{Value: "small"}, {Value: "large"}},
			},
			opts: []Options{WithIndentSize(4)},
			want: "enum Size {\n    small\n    large\n}\n",
		},
		{
			name: "with no indent",
			enum: &ast.Enum{
				Name:     ast.StringLiteral{Value: "Size"},
				Elements: []ast.StringLiteral{{Value: "small"}, {Value: "large"}},
			},
			opts: []Options{WithIndentSize(0)},
			want: "enum Size {\nsmall\nlarge\n}\n",
		},
		{
			name: "with multiline head comment",
			enum: &ast.Enum{
				Name:        ast.StringLiteral{Value: "Priority"},
				Elements:    []ast.StringLiteral{{Value: "low"}, {Value: "high"}},
				HeadComment: commentGroup("# priority levels", "# for task assignment\n"),
			},
			want: "# priority levels\n# for task assignment\nenum Priority {\n  low\n  high\n}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.enum, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatTypeAlias(t *testing.T) {
	tests := []struct {
		name  string
		alias *ast.TypeAlias
		opts  []Options
		want  string
	}{
		{
			name: "simple alias",
			alias: &ast.TypeAlias{
				Name: ast.StringLiteral{Value: "Email"},
				Type: typeExpr("string"),
			},
			want: "type Email = string\n",
		},
		{
			name: "with head comment",
			alias: &ast.TypeAlias{
				Name:        ast.StringLiteral{Value: "UserID"},
				Type:        typeExpr("uuid"),
				HeadComment: commentGroup("# unique user identifier\n"),
			},
			want: "# unique user identifier\ntype UserID = uuid\n",
		},
		{
			name: "with line comment",
			alias: &ast.TypeAlias{
				Name:        ast.StringLiteral{Value: "Score"},
				Type:        typeExpr("float"),
				LineComment: &ast.Comment{Text: " # range 0..100"},
			},
			want: "type Score = float # range 0..100\n",
		},
		{
			name: "with both comments",
			alias: &ast.TypeAlias{
				Name:        ast.StringLiteral{Value: "Token"},
				Type:        typeExpr("string"),
				HeadComment: commentGroup("# auth token\n"),
				LineComment: &ast.Comment{Text: " # opaque"},
			},
			want: "# auth token\ntype Token = string # opaque\n",
		},
		{
			name: "generic type alias",
			alias: &ast.TypeAlias{
				Name: ast.StringLiteral{Value: "UserList"},
				Type: genericTypeExpr("Page", "User"),
			},
			want: "type UserList = Page<User>\n",
		},
		{
			name: "optional type alias",
			alias: &ast.TypeAlias{
				Name: ast.StringLiteral{Value: "MaybeName"},
				Type: optionalTypeExpr("string"),
			},
			want: "type MaybeName = string?\n",
		},
		{
			name: "array type alias",
			alias: &ast.TypeAlias{
				Name: ast.StringLiteral{Value: "Tags"},
				Type: arrayTypeExpr("string"),
			},
			want: "type Tags = string[]\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.alias, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatModel(t *testing.T) {
	tests := []struct {
		name  string
		model *ast.Model
		opts  []Options
		want  string
	}{
		{
			name: "basic model aligned",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid")),
					field("name", typeExpr("string")),
				},
			},
			want: "model User {\n" +
				"  id  : uuid\n" +
				"  name: string\n" +
				"}\n",
		},
		{
			name: "single field condensed",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Wrapper"},
				Fields: []ast.Field{
					field("value", typeExpr("string")),
				},
			},
			want: "model Wrapper { value: string }\n",
		},
		{
			name: "single field not condensed",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Wrapper"},
				Fields: []ast.Field{
					field("value", typeExpr("string")),
				},
			},
			opts: []Options{WithAllowCondensedSpecs(false)},
			want: "model Wrapper {\n  value: string\n}\n",
		},
		{
			name: "single field condensed with decorator",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Wrapper"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"), decorator("primary")),
				},
			},
			want: "model Wrapper { id: uuid  @primary}\n",
		},
		{
			name: "with generic params",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Page"},
				GenericParams: []ast.StringLiteral{
					{Value: "T"},
				},
				Fields: []ast.Field{
					field("data", arrayTypeExpr("T")),
					field("total", typeExpr("int")),
				},
			},
			want: "model Page<T> {\n" +
				"  data : T[]\n" +
				"  total: int\n" +
				"}\n",
		},
		{
			name: "with multiple generic params",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Result"},
				GenericParams: []ast.StringLiteral{
					{Value: "T"},
					{Value: "E"},
				},
				Fields: []ast.Field{
					field("data", optionalTypeExpr("T")),
					field("error", optionalTypeExpr("E")),
				},
			},
			want: "model Result<T, E> {\n" +
				"  data : T?\n" +
				"  error: E?\n" +
				"}\n",
		},
		{
			name: "with decorators on fields",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"), decorator("primary")),
					field("email", typeExpr("string"), decorator("unique")),
					field("role", typeExpr("UserRole"), decoratorBareArg("default", "member")),
				},
			},
			want: "model User {\n" +
				"  id   : uuid     @primary\n" +
				"  email: string   @unique\n" +
				"  role : UserRole @default(member)\n" +
				"}\n",
		},
		{
			name: "without alignment",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"), decorator("primary")),
					field("email", typeExpr("string")),
				},
			},
			opts: []Options{WithAlignMembers(false)},
			want: "model User {\n  id: uuid @primary\n  email: string\n}\n",
		},
		{
			name: "with lines between members",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Post"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid")),
					field("title", typeExpr("string")),
				},
			},
			opts: []Options{WithLinesBetweenMembers(1)},
			want: "model Post {\n" +
				"  id   : uuid\n" +
				"\n" +
				"  title: string\n" +
				"}\n",
		},
		{
			name: "with line comment on field",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					fieldWithComment("id", typeExpr("uuid"), "# primary key"),
					field("name", typeExpr("string")),
				},
			},
			want: "model User {\n" +
				"  id  : uuid   # primary key\n" +
				"  name: string\n" +
				"}\n",
		},
		{
			name: "indent size 4",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid")),
				},
			},
			opts: []Options{WithIndentSize(4), WithAllowCondensedSpecs(false)},
			want: "model User {\n    id: uuid\n}\n",
		},
		{
			name: "many decorators wrap to next line",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"),
						decorator("primary"),
						decorator("unique"),
						decorator("indexed"),
						decorator("immutable"),
					),
				},
			},
			opts: []Options{WithMaxDecoratorsPerLine(3), WithAllowCondensedSpecs(false)},
			want: "model User {\n" +
				"  id: uuid @primary @unique @indexed\n" +
				"           @immutable\n" +
				"}\n",
		},
		{
			name: "decorators start on next line",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"),
						decorator("primary"),
						decorator("unique"),
					),
				},
			},
			opts: []Options{
				WithDecoratorsStartPosition(DecoratorPositionNextLine),
				WithAllowCondensedSpecs(false),
			},
			want: "model User {\n" +
				"  id: uuid\n" +
				"    @primary @unique\n" +
				"}\n",
		},
		{
			name: "decorators start on next line with alignment",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "User"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"),
						decorator("primary"),
					),
					field("email", typeExpr("string"),
						decorator("unique"),
					),
				},
			},
			opts: []Options{
				WithDecoratorsStartPosition(DecoratorPositionNextLine),
			},
			want: "model User {\n" +
				"  id   : uuid\n" +
				"    @primary\n" +
				"\n" +
				"  email: string\n" +
				"    @unique\n" +
				"}\n",
		},
		{
			name: "same-line decorators with alignment and wrapping",
			model: &ast.Model{
				Name: ast.StringLiteral{Value: "Post"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"),
						decorator("primary"),
						decorator("indexed"),
						decorator("immutable"),
						decorator("searchable"),
					),
					field("title", typeExpr("string"),
						decorator("unique"),
					),
				},
			},
			opts: []Options{WithMaxDecoratorsPerLine(2)},
			want: "model Post {\n" +
				"  id   : uuid   @primary @indexed\n" +
				"                @immutable @searchable\n" +
				"\n" +
				"  title: string @unique\n" +
				"}\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.model, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatInput(t *testing.T) {
	tests := []struct {
		name  string
		input *ast.Input
		opts  []Options
		want  string
	}{
		{
			name: "basic input aligned",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreateUserInput"},
				Fields: []ast.Field{
					field("email", typeExpr("string")),
					field("name", optionalTypeExpr("string")),
				},
			},
			want: "input CreateUserInput {\n" +
				"  email: string\n" +
				"  name : string?\n" +
				"}\n",
		},
		{
			name: "single field condensed",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "DeleteInput"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid")),
				},
			},
			want: "input DeleteInput { id: uuid }\n",
		},
		{
			name: "single field not condensed",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "DeleteInput"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid")),
				},
			},
			opts: []Options{WithAllowCondensedSpecs(false)},
			want: "input DeleteInput {\n  id: uuid\n}\n",
		},
		{
			name: "with decorators",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreateUserInput"},
				Fields: []ast.Field{
					field("email", typeExpr("string"), decorator("unique")),
					field("role", typeExpr("UserRole"), decoratorBareArg("default", "member")),
				},
			},
			want: "input CreateUserInput {\n" +
				"  email: string   @unique\n" +
				"  role : UserRole @default(member)\n" +
				"}\n",
		},
		{
			name: "without alignment",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreateUserInput"},
				Fields: []ast.Field{
					field("email", typeExpr("string")),
					field("name", optionalTypeExpr("string")),
				},
			},
			opts: []Options{WithAlignMembers(false)},
			want: "input CreateUserInput {\n  email: string\n  name: string?\n}\n",
		},
		{
			name: "with lines between members",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreateUserInput"},
				Fields: []ast.Field{
					field("email", typeExpr("string")),
					field("name", typeExpr("string")),
				},
			},
			opts: []Options{WithLinesBetweenMembers(1)},
			want: "input CreateUserInput {\n" +
				"  email: string\n" +
				"\n" +
				"  name : string\n" +
				"}\n",
		},
		{
			name: "with line comment",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreateUserInput"},
				Fields: []ast.Field{
					fieldWithComment("email", typeExpr("string"), "# required"),
					field("name", optionalTypeExpr("string")),
				},
			},
			want: "input CreateUserInput {\n" +
				"  email: string  # required\n" +
				"  name : string?\n" +
				"}\n",
		},
		{
			name: "single field condensed with decorator",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "DeleteInput"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"), decorator("primary")),
				},
			},
			want: "input DeleteInput { id: uuid  @primary\n}\n",
		},
		{
			name: "decorators start on next line",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreateUserInput"},
				Fields: []ast.Field{
					field("email", typeExpr("string"),
						decorator("unique"),
						decorator("indexed"),
					),
				},
			},
			opts: []Options{
				WithDecoratorsStartPosition(DecoratorPositionNextLine),
				WithAllowCondensedSpecs(false),
			},
			want: "input CreateUserInput {\n" +
				"  email: string\n" +
				"    @unique @indexed\n" +
				"}\n",
		},
		{
			name: "same-line decorators with alignment and wrapping",
			input: &ast.Input{
				Name: ast.StringLiteral{Value: "CreatePostInput"},
				Fields: []ast.Field{
					field("id", typeExpr("uuid"),
						decorator("primary"),
						decorator("indexed"),
						decorator("immutable"),
					),
					field("title", typeExpr("string"),
						decorator("unique"),
					),
				},
			},
			opts: []Options{WithMaxDecoratorsPerLine(2)},
			want: "input CreatePostInput {\n" +
				"  id   : uuid   @primary @indexed\n" +
				"                @immutable\n" +
				"\n" +
				"  title: string @unique\n" +
				"}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.input, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatApiREST(t *testing.T) {
	tests := []struct {
		name string
		api  *ast.Api
		opts []Options
		want string
	}{
		{
			name: "basic rest api",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), genericTypeExpr("Page", "User")),
					restRoute("POST", pathSegs(seg("users")), typeExpr("User")),
				},
			},
			want: "api UsersApi {\n" +
				"  GET  /users -> Page<User>\n" +
				"  POST /users -> User\n" +
				"}\n",
		},
		{
			name: "rest api with path params",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users"), param("id")), typeExpr("User")),
					restRoute("DELETE", pathSegs(seg("users"), param("id")), typeExpr("void")),
				},
			},
			want: "api UsersApi {\n" +
				"  GET    /users/:id -> User\n" +
				"  DELETE /users/:id -> void\n" +
				"}\n",
		},
		{
			name: "rest api with decorators on routes",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), genericTypeExpr("Page", "User"),
						decoratorBareArg("auth", "bearer"),
					),
					restRoute("POST", pathSegs(seg("users")), typeExpr("User"),
						decoratorBareArg("auth", "bearer"),
						decoratorBareArg("status", "201"),
					),
				},
			},
			want: "api UsersApi {\n" +
				"  GET  /users -> Page<User> @auth(bearer)\n" +
				"  POST /users -> User       @auth(bearer) @status(201)\n" +
				"}\n",
		},
		{
			name: "rest api with api-level decorators",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				ApiDecorators: []ast.Decorator{
					decoratorBareArg("style", "rest"),
					decoratorBareArg("version", "v2"),
				},
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), typeExpr("User")),
				},
			},
			want: "api UsersApi @style(rest) @version(v2) {\n  GET /users -> User\n}\n",
		},
		{
			name: "rest api with directives",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				ApiDirectives: []ast.Decorator{
					decoratorWithArg("basePath", "path", &ast.StringLiteral{Value: "/api/v1"}),
					decoratorBareArg("auth", "bearer"),
				},
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), typeExpr("User")),
				},
			},
			want: "api UsersApi {\n  @basePath(path: /api/v1)\n  @auth(bearer)\n\n  GET /users -> User\n}\n",
		},
		{
			name: "rest api without alignment",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), genericTypeExpr("Page", "User")),
					restRoute("POST", pathSegs(seg("users")), typeExpr("User")),
				},
			},
			opts: []Options{WithAlignMembers(false)},
			want: "api UsersApi {\n  GET /users -> Page<User>\n  POST /users -> User\n}\n",
		},
		{
			name: "rest api with lines between members three routes",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), typeExpr("User")),
					restRoute("POST", pathSegs(seg("users")), typeExpr("User")),
					restRoute("DELETE", pathSegs(seg("users")), typeExpr("void")),
				},
			},
			opts: []Options{WithLinesBetweenMembers(1)},
			want: "api UsersApi {\n" +
				"  GET    /users -> User\n" +
				"\n" +
				"  POST   /users -> User\n" +
				"\n" +
				"  DELETE /users -> void\n" +
				"}\n",
		},
		{
			name: "rest api with indent size 4",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("GET", pathSegs(seg("users")), typeExpr("User")),
				},
			},
			opts: []Options{WithIndentSize(4)},
			want: "api UsersApi {\n    GET /users -> User\n}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.api, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatApiRPC(t *testing.T) {
	tests := []struct {
		name string
		api  *ast.Api
		opts []Options
		want string
	}{
		{
			name: "basic rpc api",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersService"},
				Style: ast.StyleRPC,
				Routes: []ast.Route{
					rpcRoute("GetUser", typeExpr("GetUserInput"), typeExpr("User"), false),
					rpcRoute("CreateUser", typeExpr("CreateUserInput"), typeExpr("User"), false),
				},
			},
			want: "api UsersService {\n" +
				"  rpc GetUser(GetUserInput)       -> User\n" +
				"  rpc CreateUser(CreateUserInput) -> User\n" +
				"}\n",
		},
		{
			name: "rpc with streaming",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "ChatService"},
				Style: ast.StyleRPC,
				Routes: []ast.Route{
					rpcRoute("GetMessages", typeExpr("GetMessagesInput"), typeExpr("Message"), false),
					rpcRoute("StreamChat", typeExpr("ChatInput"), typeExpr("ChatMessage"), true),
				},
			},
			want: "api ChatService {\n" +
				"  rpc        GetMessages(GetMessagesInput) -> Message\n" +
				"  rpc stream StreamChat(ChatInput)         -> ChatMessage\n" +
				"}\n",
		},
		{
			name: "rpc with decorators",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersService"},
				Style: ast.StyleRPC,
				Routes: []ast.Route{
					rpcRoute("GetUser", typeExpr("GetUserInput"), typeExpr("User"), false,
						decoratorBareArg("auth", "bearer"),
					),
				},
			},
			want: "api UsersService {\n" +
				"  rpc GetUser(GetUserInput) -> User @auth(bearer)\n" +
				"}\n",
		},
		{
			name: "rpc without alignment",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersService"},
				Style: ast.StyleRPC,
				Routes: []ast.Route{
					rpcRoute("GetUser", typeExpr("GetUserInput"), typeExpr("User"), false),
					rpcRoute("CreateUser", typeExpr("CreateUserInput"), typeExpr("User"), false),
				},
			},
			opts: []Options{WithAlignMembers(false)},
			want: "api UsersService {\n  rpc GetUser(GetUserInput) -> User\n  rpc CreateUser(CreateUserInput) -> User\n}\n",
		},
		{
			name: "rest with many decorators wrapped and aligned",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UsersApi"},
				Style: ast.StyleREST,
				Routes: []ast.Route{
					restRoute("POST", pathSegs(seg("users")), typeExpr("User"),
						decoratorBareArg("body", "CreateUserInput"),
						decoratorBareArg("errors", "400"),
						decoratorBareArg("errors", "409"),
						decoratorBareArg("version", "1"),
						decorator("deprecated"),
					),
				},
			},
			opts: []Options{WithMaxDecoratorsPerLine(3)},
			want: "api UsersApi {\n" +
				"  POST /users -> User @body(CreateUserInput) @errors(400) @errors(409)\n" +
				"                      @version(1) @deprecated\n" +
				"}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.api, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatApiEvents(t *testing.T) {
	tests := []struct {
		name string
		api  *ast.Api
		opts []Options
		want string
	}{
		{
			name: "basic events api",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UserEvents"},
				Style: ast.StyleEvents,
				Routes: []ast.Route{
					eventRoute("UserCreated", ast.EventPublish, typeExpr("UserEvent")),
					eventRoute("UserUpdated", ast.EventSubscribe, typeExpr("UserEvent")),
				},
			},
			want: "api UserEvents {\n" +
				"  publish   UserCreated -> UserEvent\n" +
				"  subscribe UserUpdated -> UserEvent\n" +
				"}\n",
		},
		{
			name: "events with decorators",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UserEvents"},
				Style: ast.StyleEvents,
				Routes: []ast.Route{
					eventRoute("UserCreated", ast.EventPublish, typeExpr("UserEvent"),
						decoratorBareArg("topic", "users"),
					),
				},
			},
			want: "api UserEvents {\n" +
				"  publish UserCreated -> UserEvent @topic(users)\n" +
				"}\n",
		},
		{
			name: "events without alignment",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UserEvents"},
				Style: ast.StyleEvents,
				Routes: []ast.Route{
					eventRoute("UserCreated", ast.EventPublish, typeExpr("UserEvent")),
					eventRoute("UserUpdated", ast.EventSubscribe, typeExpr("UserEvent")),
				},
			},
			opts: []Options{WithAlignMembers(false)},
			want: "api UserEvents {\n  publish UserCreated -> UserEvent\n  subscribe UserUpdated -> UserEvent\n}\n",
		},
		{
			name: "events with lines between members three routes",
			api: &ast.Api{
				Name:  ast.StringLiteral{Value: "UserEvents"},
				Style: ast.StyleEvents,
				Routes: []ast.Route{
					eventRoute("UserCreated", ast.EventPublish, typeExpr("UserEvent")),
					eventRoute("UserUpdated", ast.EventSubscribe, typeExpr("UserEvent")),
					eventRoute("UserDeleted", ast.EventPublish, typeExpr("UserEvent")),
				},
			},
			opts: []Options{WithLinesBetweenMembers(1)},
			want: "api UserEvents {\n" +
				"  publish   UserCreated -> UserEvent\n" +
				"\n" +
				"  subscribe UserUpdated -> UserEvent\n" +
				"\n" +
				"  publish   UserDeleted -> UserEvent\n" +
				"}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSpec(t, tc.api, tc.opts...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatImport(t *testing.T) {
	tests := []struct {
		name string
		imp  ast.Import
		opts []Options
		want string
	}{
		{
			name: "single line import",
			imp: ast.Import{
				Path: ast.QualifiedIdent{Parts: []string{"acme", "common", "v1"}},
				Names: []ast.StringLiteral{
					{Value: "Page"},
					{Value: "PaginationInput"},
				},
			},
			want: "namespace test\n\nimport acme.common.v1 { Page, PaginationInput }\n\n",
		},
		{
			name: "multiline import",
			imp: ast.Import{
				Path: ast.QualifiedIdent{Parts: []string{"acme", "common", "v1"}},
				Names: []ast.StringLiteral{
					{Value: "Page"},
					{Value: "PaginationInput"},
				},
			},
			opts: []Options{WithMultilineImports(true)},
			want: "namespace test\n\nimport acme.common.v1 {\n  Page,\n  PaginationInput}\n\n",
		},
		{
			name: "single line with head comment",
			imp: ast.Import{
				Path: ast.QualifiedIdent{Parts: []string{"acme", "common"}},
				Names: []ast.StringLiteral{
					{Value: "Page"},
				},
				HeadComment: commentGroup("# shared types\n"),
			},
			want: "namespace test\n\n# shared types\nimport acme.common { Page }\n\n",
		},
		{
			name: "single line with line comment",
			imp: ast.Import{
				Path: ast.QualifiedIdent{Parts: []string{"acme", "common"}},
				Names: []ast.StringLiteral{
					{Value: "Page"},
				},
				LineComment: &ast.Comment{Text: " # pagination"},
			},
			want: "namespace test\n\nimport acme.common { Page } # pagination\n\n",
		},
		{
			name: "multiline with line comment",
			imp: ast.Import{
				Path: ast.QualifiedIdent{Parts: []string{"acme", "common"}},
				Names: []ast.StringLiteral{
					{Value: "Page"},
					{Value: "Error"},
				},
				LineComment: &ast.Comment{Text: " # shared"},
			},
			opts: []Options{WithMultilineImports(true)},
			want: "namespace test\n\nimport acme.common {\n  Page,\n  Error} # shared\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stencil := &ast.Stencil{
				Namespace: &ast.Namespace{
					Pos:  ast.Position{Line: 1},
					Name: ast.QualifiedIdent{Parts: []string{"test"}},
				},
				Imports: []ast.Import{tc.imp},
			}
			f, err := New(tc.opts...)
			assert.NoError(t, err)
			got, err := f.Format(stencil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestFormatFull(t *testing.T) {
	tests := []struct {
		name    string
		stencil *ast.Stencil
		opts    []Options
		want    string
		wantErr bool
	}{
		{
			name: "complete stencil with all spec types",
			stencil: &ast.Stencil{
				Comments: []*ast.Comment{
					{Pos: ast.Position{Line: 1}, Text: "# User service API"},
				},
				Namespace: &ast.Namespace{
					Pos:  ast.Position{Line: 2},
					Name: ast.QualifiedIdent{Parts: []string{"acme", "users", "v1"}},
				},
				Imports: []ast.Import{
					{
						Path: ast.QualifiedIdent{Parts: []string{"acme", "common"}},
						Names: []ast.StringLiteral{
							{Value: "Page"},
						},
					},
				},
				Specs: []ast.SpecNode{
					&ast.TypeAlias{
						Name: ast.StringLiteral{Value: "Email"},
						Type: typeExpr("string"),
					},
					&ast.Enum{
						Name: ast.StringLiteral{Value: "UserRole"},
						Elements: []ast.StringLiteral{
							{Value: "admin"},
							{Value: "member"},
							{Value: "guest"},
						},
					},
					&ast.Model{
						Name: ast.StringLiteral{Value: "User"},
						Fields: []ast.Field{
							field("id", typeExpr("uuid"), decorator("primary")),
							field("email", typeExpr("Email"), decorator("unique")),
							field("role", typeExpr("UserRole"), decoratorBareArg("default", "member")),
							field("name", optionalTypeExpr("string")),
						},
					},
					&ast.Input{
						Name: ast.StringLiteral{Value: "CreateUserInput"},
						Fields: []ast.Field{
							field("email", typeExpr("string")),
							field("name", optionalTypeExpr("string")),
						},
					},
					&ast.Api{
						Name:  ast.StringLiteral{Value: "UsersApi"},
						Style: ast.StyleREST,
						Routes: []ast.Route{
							restRoute("GET", pathSegs(seg("users")), genericTypeExpr("Page", "User")),
							restRoute("POST", pathSegs(seg("users")), typeExpr("User")),
							restRoute("GET", pathSegs(seg("users"), param("id")), typeExpr("User")),
							restRoute("DELETE", pathSegs(seg("users"), param("id")), typeExpr("void")),
						},
					},
				},
			},
			want: "# User service API\n" +
				"namespace acme.users.v1\n" +
				"\n" +
				"import acme.common { Page }\n" +
				"\n" +
				"type Email = string\n" +
				"\n" +
				"enum UserRole {\n" +
				"  admin\n" +
				"  member\n" +
				"  guest\n" +
				"}\n" +
				"\n" +
				"model User {\n" +
				"  id   : uuid     @primary\n" +
				"  email: Email    @unique\n" +
				"  role : UserRole @default(member)\n" +
				"  name : string?\n" +
				"}\n" +
				"\n" +
				"input CreateUserInput {\n" +
				"  email: string\n" +
				"  name : string?\n" +
				"}\n" +
				"\n" +
				"api UsersApi {\n" +
				"  GET    /users     -> Page<User>\n" +
				"  POST   /users     -> User\n" +
				"  GET    /users/:id -> User\n" +
				"  DELETE /users/:id -> void\n" +
				"}\n",
		},
		{
			name: "namespace only",
			stencil: &ast.Stencil{
				Namespace: &ast.Namespace{
					Pos:  ast.Position{Line: 1},
					Name: ast.QualifiedIdent{Parts: []string{"test"}},
				},
			},
			want: "namespace test\n\n",
		},
		{
			name:    "empty stencil",
			stencil: &ast.Stencil{},
			want:    "",
		},
		{
			name: "custom options integration",
			stencil: &ast.Stencil{
				Namespace: &ast.Namespace{
					Pos:  ast.Position{Line: 1},
					Name: ast.QualifiedIdent{Parts: []string{"acme"}},
				},
				Specs: []ast.SpecNode{
					&ast.Model{
						Name: ast.StringLiteral{Value: "User"},
						Fields: []ast.Field{
							field("id", typeExpr("uuid"), decorator("primary")),
							field("name", typeExpr("string")),
						},
					},
					&ast.Enum{
						Name: ast.StringLiteral{Value: "Role"},
						Elements: []ast.StringLiteral{
							{Value: "admin"},
						},
					},
				},
			},
			opts: []Options{
				WithIndentSize(4),
				WithLinesBetweenSpecs(2),
				WithAlignMembers(false),
				WithAllowCondensedSpecs(false),
			},
			want: "namespace acme\n" +
				"\n\n" +
				"model User {\n" +
				"    id: uuid @primary\n" +
				"    name: string\n" +
				"}\n" +
				"\n\n" +
				"enum Role {\n" +
				"    admin\n" +
				"}\n",
		},
		{
			name: "rpc and events in same file",
			stencil: &ast.Stencil{
				Namespace: &ast.Namespace{
					Pos:  ast.Position{Line: 1},
					Name: ast.QualifiedIdent{Parts: []string{"acme"}},
				},
				Specs: []ast.SpecNode{
					&ast.Api{
						Name:  ast.StringLiteral{Value: "UserService"},
						Style: ast.StyleRPC,
						Routes: []ast.Route{
							rpcRoute("GetUser", typeExpr("GetUserInput"), typeExpr("User"), false),
						},
					},
					&ast.Api{
						Name:  ast.StringLiteral{Value: "UserEvents"},
						Style: ast.StyleEvents,
						Routes: []ast.Route{
							eventRoute("UserCreated", ast.EventPublish, typeExpr("UserEvent")),
						},
					},
				},
			},
			want: "namespace acme\n" +
				"\n" +
				"api UserService {\n" +
				"  rpc GetUser(GetUserInput) -> User\n" +
				"}\n" +
				"\n" +
				"api UserEvents {\n" +
				"  publish UserCreated -> UserEvent\n" +
				"}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := New(tc.opts...)
			assert.NoError(t, err)
			got, err := f.Format(tc.stencil)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestDecoratorPositionFromString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want DecoratorPosition
	}{
		{
			name: "same",
			in:   "same",
			want: DecoratorPositionSameLine,
		},
		{
			name: "next",
			in:   "next",
			want: DecoratorPositionNextLine,
		},
		{
			name: "unknown defaults to same",
			in:   "anything",
			want: DecoratorPositionSameLine,
		},
		{
			name: "empty defaults to same",
			in:   "",
			want: DecoratorPositionSameLine,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DecoratorPositionFromString(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestOptionsLimits(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Options
		checkFn func(t *testing.T, f *Formatter)
	}{
		{
			name: "max decorators per line lower bound",
			opts: []Options{WithMaxDecoratorsPerLine(0)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 1, f.maxDecoratorPerLine)
			},
		},
		{
			name: "max decorators per line upper bound",
			opts: []Options{WithMaxDecoratorsPerLine(100)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 10, f.maxDecoratorPerLine)
			},
		},
		{
			name: "lines between specs lower bound",
			opts: []Options{WithLinesBetweenSpecs(-1)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 0, f.linesBetweenSpecs)
			},
		},
		{
			name: "lines between specs upper bound",
			opts: []Options{WithLinesBetweenSpecs(10)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 2, f.linesBetweenSpecs)
			},
		},
		{
			name: "lines between members lower bound",
			opts: []Options{WithLinesBetweenMembers(-1)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 0, f.linesBetweenMembers)
			},
		},
		{
			name: "lines between members upper bound",
			opts: []Options{WithLinesBetweenMembers(10)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 2, f.linesBetweenMembers)
			},
		},
		{
			name: "indent size lower bound",
			opts: []Options{WithIndentSize(-1)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 0, f.indentSize)
			},
		},
		{
			name: "indent size upper bound",
			opts: []Options{WithIndentSize(100)},
			checkFn: func(t *testing.T, f *Formatter) {
				assert.Equal(t, 8, f.indentSize)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := New(tc.opts...)
			assert.NoError(t, err)
			tc.checkFn(t, f)
		})
	}
}

func TestNewDefaults(t *testing.T) {
	f, err := New()
	assert.NoError(t, err)
	assert.Equal(t, 3, f.maxDecoratorPerLine)
	assert.Equal(t, 1, f.linesBetweenSpecs)
	assert.Equal(t, 0, f.linesBetweenMembers)
	assert.Equal(t, 2, f.indentSize)
	assert.True(t, f.alignMembers)
	assert.True(t, f.allowCondensedSpecs)
	assert.False(t, f.multilineImports)
	assert.Equal(t, DecoratorPositionSameLine, f.decoratorsStartPosition)
}
