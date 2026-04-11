// Package spray is a CLI compiler for .stencil files — a custom DSL for
// documenting APIs and data models. It parses stencil source into an AST
// and compiles to output formats like JSON Schema, Mermaid ER
// diagrams, and Markdown. There is also support for external emitters,
// so users can define their own output formats as well.
//
// A .stencil file defines namespaces, models, inputs, enums, type aliases,
// and APIs. Here's a complete example:
//
//	namespace acme.users.v2
//
//	import acme.common.v1 { Page, PaginationInput, ApiError }
//
//	type Email = string
//
//	enum UserRole {
//	  admin
//	  member
//	  guest
//	}
//
//	model User {
//	  id:        uuid      @primary
//	  email:     Email     @unique
//	  role:      UserRole  @default(member)
//	  name:      string?
//	  createdAt: timestamp @default(now)
//	  posts:     Post[]    @relation
//	}
//
//	model Post {
//	  id:       uuid   @primary
//	  title:    string
//	  body:     string?
//	  authorId: uuid
//	  author:   User   @relation(field: authorId)
//	}
//
//	input CreateUserInput {
//	  email: string
//	  name:  string?
//	  role:  UserRole @default(member)
//	}
//
//	input UpdateUserInput {
//	  name: string?
//	  role: UserRole?
//	}
//
//	model Result<T, E> {
//	  ok:    boolean
//	  data:  T?
//	  error: E?
//	}
//
//	api UserService @version(2) @style(rest) {
//	  @basePath("/api/v2/users")
//	  @auth(bearer)
//
//	  GET  /      -> Page<User>
//	    @query(PaginationInput)
//
//	  GET  /:id   -> Result<User, ApiError>
//	    @errors(401, 404)
//
//	  POST /      -> User
//	    @body(CreateUserInput)
//	    @errors(400, 409)
//
//	  PATCH /:id  -> User
//	    @body(UpdateUserInput)
//	    @errors(400, 401, 404)
//
//	  DELETE /:id -> void
//	    @errors(401, 404)
//
//	  # Still-active v1 route
//	  GET /search -> User[] @version(1) @deprecated("Use GET / with query params")
//	}
//
// Use [github.com/jimschubert/spray/parser] to parse .stencil source,
// [github.com/jimschubert/spray/resolver] to resolve types across files,
// and [github.com/jimschubert/spray/emitter] to compile to output formats.
//
// For full specification of the .stencil language, see the https://github.com/jimschubert/spray/blob/main/specification.md
package spray
