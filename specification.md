# Stencil Specification v0.1

## Overview

This is a minimal DSL for documenting APIs and data models. You write `.stencil` files once and compile them to multiple
output formats using the `spray` CLI.

## Use Cases

Stencil is designed to support multiple documentation scenarios:

1. **API Documentation**: Document REST, RPC, or event-driven APIs with complete request/response specifications. Use
   `api` blocks alongside `model` and `input` definitions to describe your application's endpoints.

2. **Data Model Documentation**: Document data structures without any API layer. A suite of `.stencil` files containing
   only `model` definitions (no `api` blocks) is perfectly valid and can be compiled to ER diagrams, database schemas,
   or other data modeling outputs.

3. **Hybrid Documentation**: Combine both of the above approaches. Some files may define only data models, while others
   define APIs that reference those models. You may generate one or more output formats from the same source files,
   depending on your documentation needs.

## Top-Level Structures

A `.stencil` file can be comprised of the following top-level structures:

<dl>
    <dt>namespace</dt>
    <dd>an organizational unit or package (optional, max one per file)</dd>
    <dt>import</dt>
    <dd>code organization utility to import definitions from other files (zero or more)</dd>
    <dt>type</dt>
    <dd>scalar/enum aliases</dd>
    <dt>model</dt>
    <dd>data shapes</dd>
    <dt>input</dt>
    <dd>request body shapes</dd>
    <dt>api</dt>
    <dd>route/procedure definitions</dd>
</dl>


The `model` and `input` structures are intentionally separate, following community best practices to protect against
certain types of [input threats](https://owaspai.org/docs/2_threats_through_use/) at the request/input layer. While
`model` types can
be used in API responses to document real-world scenarios where applications expose their data models directly, the
separation ensures that request payloads are explicitly defined and validated through dedicated `input` types.

A `namespace` must exist at the top of a file (anywhere else is invalid). Any `imports` must be grouped at the top of
the file, after the `namespace` if it exists, but before any other declarations.

The remaining structures (`type`, `model`, `input`, `api`) can be in any order and interleaved as needed.

---

## Namespacing

```stencil
namespace acme.users.v2

import acme.common.v1 { Page, PaginationInput, ApiError }
```

Each file may declare at most one namespace. If no namespace is defined, the parser will create an implicit "default" namespace.

An `import` statement is explicit — only listed names are brought into scope (i.e., no "star" patterns like Java).

A namespace may have a single leading comment and an optional line comment:

```stencil
# This is a document-level comment

# This comment is "attached" to namespace
namespace acme.users.v2 # this is a line comment on the namespace declaration
```

---

## Scalars

Built-in scalar types:

| Type        | Description        |
|-------------|--------------------|
| `string`    | UTF-8 string       |
| `int`       | Integer            |
| `float`     | Floating point     |
| `boolean`   | Boolean            |
| `uuid`      | UUID string        |
| `timestamp` | Date + time        |
| `date`      | Date only          |
| `any`       | Unconstrained type |

---

## Type Aliases

Type aliases resemble Go's type aliases, allowing you to create new types based on existing ones:

```stencil
type Email = string
type Cursor = string
```

---

## Enums

Enums look familiar to enums in many programming languages:

```stencil
enum UserRole {
    admin
    member
    guest
}
```

Enum elements may be separated by commas, allowing for compact single-line definitions. The preference is for each 
element on a separate line without commas, but both styles are supported:

**Multi-line (preferred):**
```stencil
enum UserRole {
    admin
    member
    guest
}
```

**Single-line (comma-separated):**
```stencil
enum Status { ACTIVE, INACTIVE }
```

**Mixed (trailing comma allowed):**
```stencil
enum Color {
    RED,
    GREEN,
    BLUE,
}
```

---

## Models

Models represent data [shapes](https://dev.to/tiffengineer/what-is-meant-by-a-shape-in-programming-263c) and these
define how data is stored and manipulated within the application. They can be thought of as the "internal"
representation of data.

A `model` can be have zero or more decorations applied wihch help define relations (similar to relational database
associations).

Examples:

```stencil
model User {
  id:        uuid      @primary
  email:     Email     @unique
  role:      UserRole  @default(member)
  name:      string?
  createdAt: timestamp @default(now)
  updatedAt: timestamp @updatedAt
  posts:     Post[]    @relation
}
```

Each item within a model is called a `field`, and a field is represented as `fieldName: fieldType [decorations]`.

**Available field decorators**

The following decorators are supported for `model` fields. These decorators form a closed set — only the decorators
listed here are valid for model fields (with the exception of `@raw`, which is available as an escape hatch for all
structures).

| Decorator           | Description                                                           |
|---------------------|-----------------------------------------------------------------------|
| `@primary`          | A primary key or identifier                                           |
| `@unique`           | A constraint that this value is unique within the system              |
| `@default(value)`   | The default value applied by the system (`now`, `uuid`, or a literal) |
| `@updatedAt`        | Calculated automatically on update                                    |
| `@relation(field:)` | A foreign-key type of relationship                                    |
| `@deprecated(msg)`  | Marks a field as deprecated                                           |

**Type modifiers:**

- `?` — optional field, indicates the field may be omitted (absent from the structure) or explicitly set to `null`
- `[]` — array (e.g. `Post[]`)
- Both can be combined: `string[]?` (array that may be omitted or null)

---

## Generics

Models can declare unconstrained generic type parameters (e.g., `T`, `E`) that act as placeholders for concrete types.

>[!NOTE]
> Generic type parameters are **only supported** for `model` declarations.

```stencil
model Page<T> {        // T is a type parameter (placeholder)
  data:       T[]
  nextCursor: Cursor?
  total:      int
}

model Result<T, E> {  // T and E are type parameters
  ok:    boolean
  data:  T?
  error: E?
}
```

**Usage with concrete types:**

```stencil
api UserService {
  GET /users -> Page<User>  // T becomes User
  GET /posts -> Page<Post>  // T becomes Post
}
```

On output, generics are [**monomorphized**](https://en.wikipedia.org/wiki/Monomorphization) — `Page<User>` compiles to a
concrete named
schema (e.g. `UserPage` but may vary contextually) in formats that don't support generics natively (OpenAPI, JSON
Schema).

---

## Inputs

An `input` is intended purely for application inputs. This allows for more flexible data shapes and protects against
certain types of [input threats](https://owaspai.org/docs/2_threats_through_use/).

>[!NOTE]
> Generic type parameters are **not supported** for `input` declarations. Inputs must be concrete types.


```stencil
input CreateUserInput {
  email: string
  name:  string?
  role:  UserRole @default(member)
}

input PaginationInput {
  limit:  int    @default(20)
  cursor: Cursor?
}
```

The only decorator supported by `input` is `@default(value)`, which behaves the same as the `@default` decorator for
`model` fields,
applying a default value when the field is omitted from the input.

Inputs intentionally don't support `@primary`, `@relation`, or `@updatedAt` — they're pure payload shapes.

---

## API

An `api` structure defines an endpoint exposed within an application.
An API block supports both REST and RPC style declarations, as well as publish/subscribe event routing.
Decorators can be applied at the API level (applying to all routes) or at the individual route level.

**REST API Example**

```stencil
api UserService @version(2) @style(rest) {
  @basePath("/api/v2/users")
  @auth(bearer)

  GET  /      -> Page<User>  @query(PaginationInput)
  GET  /:id   -> User        @errors(401, 404)
  POST /      -> User        @body(CreateUserInput) @errors(400, 409)
  DELETE /:id -> void        @errors(401, 404)
}
```

**API-level decorators** (on the `api` declaration) which control features within the `api` block:

| Decorator     | Meaning                                                   |
|---------------|-----------------------------------------------------------|
| `@version(n)` | (optional) API version number                             |
| `@style(x)`   | (optional) Route style: `rest` (default), `rpc`, `events` |

**Block-level directives** (inside the `api` block, apply to all routes):

| Directive           | Meaning                                          |
|---------------------|--------------------------------------------------|
| `@basePath("path")` | URL prefix for all routes                        |
| `@auth(scheme)`     | Auth scheme: `bearer`, `apiKey`, `basic`, `none` |

**Route-level decorators:**

| Decorator            | Meaning                      |
|----------------------|------------------------------|
| `@body(InputType)`   | Request body shape           |
| `@query(InputType)`  | Query string params          |
| `@errors(codes...)`  | Expected error status codes  |
| `@summary("text")`   | Short description            |
| `@tag("name")`       | OpenAPI tag grouping         |
| `@version(n)`        | Route-level version override |
| `@deprecated("msg")` | Mark route as deprecated     |

Decorators can be applied inline (all on the same line as the route) or hanging-indented on subsequent lines. 
The order of decorators is _not_ significant. Both styles are equivalent:

**Inline decorators:**
```stencil
GET /:id -> User @errors(401, 404) @summary("Get user by ID")
```

**Hanging-indented decorators:**
```stencil
GET /:id -> User
  @summary("Get user by ID")
  @errors(401, 404)
```

**Path parameters** are defined inline within the route path using colon syntax (`:paramName`). Path parameters are
always of type `string`. For example, `GET /users/:id` defines a path parameter `id` of type `string`.

>[!WARNING]
> While `model` types can be used directly as return types in route definitions (e.g., `GET /:id -> User`), this may
> result in warnings during compilation and could be removed in a future version without a deprecation notice. For
> endpoints that expose data models, consider defining dedicated output models without decorators to represent layered
> object mapping, which is a common pattern in real-world applications.

---

## API Styles

### `@style(rest)` — default

REST is a resource-oriented endpoint including both an HTTP verb and path.

```stencil
api UserService @style(rest) {
  GET    /users     -> Page<User>
  POST   /users     -> User   @body(CreateUserInput)
  DELETE /users/:id -> void
}
```

>[!WARNING]
> Route syntax must match the declared API style. It is an error to define a route as `@style(rest)`, then omit the HTTP
> verb and path _or_ use terms from another style. Style-specific grammar is enforced through semantic validation during
> compilation. For example, the following is invalid:
> ```stencil
> api UserService @style(rest) {
>   # !INVALID - using RPC syntax in a REST api
>   rpc GetUser(GetUserInput) -> User
> }
> ```

### `@style(rpc)`

RPC is an action-oriented remote procedure call. This style supports streaming.

```stencil
api FeedService @style(rpc) {
  rpc GetUser(GetUserInput)            -> User
  rpc stream GetFeed(GetFeedInput)     -> FeedItem
  rpc stream Chat(ChatInput)           -> ChatMessage
}
```

`rpc stream` indicates server-streaming. Bidirectional streaming may be added in a future version.

>[!WARNING]
> Route syntax must match the declared API style. It is an error to define an endpoint as `@style(rpc)`, then apply REST
> or eventing keywords. Style-specific grammar is enforced through semantic validation during compilation. For example,
> the following is invalid:
> ```stencil
> api UserService @style(rpc) {
>   # !INVALID - using event syntax in an RPC api
>   publish   UserCreated -> UserCreatedEvent
>   subscribe UserDeleted -> UserDeletedEvent
> }
> ```

### `@style(events)`

The event style is used for publish/subscribe event-based routing.

```stencil
api UserEvents @style(events) {
  publish   UserCreated -> UserCreatedEvent
  subscribe UserDeleted -> UserDeletedEvent
}
```

Event payloads are plain `model` declarations — no special `event` type.

---

## Versioning

Versioning works at two levels:

```stencil
# API-level: groups all routes under a single version by default
api UserService @version(2) {
  @baseUrl("/api/v2/users")

  GET /:id -> User

  # Route-level override: this route is still on v1
  GET /search -> User[]
    @version(1)
    @deprecated("Use GET / with query params")
}
```

---

## Raw Escape Hatch

Use `@raw` annotations for output-specific features which the compiler doesn't natively support.
Multiple `@raw` blocks can coexist on the same node — each emitter picks the matching target and ignores the rest.

```stencil
model Webhook {
  id:      uuid @primary
  payload: any

  @raw(openapi) {
    "x-webhook-signature": "sha256"
    "x-internal": true
  }

  @raw(mermaid) {
    "style": "fill:#f9f,stroke:#333"
  }

  @raw(blueprint) {
    "some-key": "value"
  }
}
```

`@raw` values support: strings, integers, floats, `true`, `false`, `null`.

The target name is an open string — third-party emitters register their own
target name and consume their matching `@raw` block.

---

## Output Targets

| Target       | Format              | Notes                                                           |
|--------------|---------------------|-----------------------------------------------------------------|
| `openapi`    | OpenAPI 3.x         | (TBD) REST and RPC routes; AsyncAPI considered for events in v2 |
| `jsonschema` | JSON Schema         | (TBD) One schema per `model` and `input`                        |
| `mermaid`    | Mermaid `erDiagram` | (TBD) All models and `@relation` links                          |
| `markdown`   | Markdown tables     | (TBD) Field reference per model + route listing per API         |

---

## Full Example

```stencil
namespace acme.users.v2

import acme.common.v1 { Page, PaginationInput, ApiError }

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
}
```

---

## EBNF Grammar

>[!NOTE]
> The grammar below is intentionally simplified and may contain minor omissions or differences from the worded
> specification. Semantic validation and certain nuances are described throughout this document. The grammar serves as a
> structural guide rather than a complete formal specification.

```ebnf
File            ::= NamespaceDecl? ImportDecl* TopLevelDecl*

NamespaceDecl   ::= "namespace" QualifiedIdent NEWLINE
ImportDecl      ::= "import" QualifiedIdent "{" IdentList "}" NEWLINE

TopLevelDecl    ::= TypeAliasDecl
                  | EnumDecl
                  | ModelDecl
                  | InputDecl
                  | ApiDecl

QualifiedIdent  ::= IDENT ("." IDENT)*
IdentList       ::= IDENT ("," IDENT)*

TypeAliasDecl   ::= "type" IDENT "=" TypeExpr NEWLINE

EnumDecl        ::= "enum" IDENT "{" EnumValue* "}"
EnumValue       ::= IDENT ("," | NEWLINE)?

ModelDecl       ::= "model" IDENT GenericParams? "{" NEWLINE FieldDecl* RawBlock* "}"
InputDecl       ::= "input" IDENT GenericParams? "{" NEWLINE FieldDecl* "}"
FieldDecl       ::= IDENT ":" TypeExpr Decorator* NEWLINE
GenericParams   ::= "<" IDENT ("," IDENT)* ">"

TypeExpr        ::= BaseType GenericArgs? ArraySuffix? OptionalSuffix?
BaseType        ::= QualifiedIdent | ScalarType
GenericArgs     ::= "<" TypeExpr ("," TypeExpr)* ">"
ArraySuffix     ::= "[]"
OptionalSuffix  ::= "?"

ScalarType      ::= "string" | "int" | "float" | "boolean"
                  | "uuid" | "timestamp" | "date" | "any"

ApiDecl         ::= "api" IDENT ApiDecorator* "{" NEWLINE ApiDirective* RouteDecl* "}"
ApiDecorator    ::= "@version" "(" INT ")"
                  | "@style"   "(" ApiStyle ")"
ApiStyle        ::= "rest" | "rpc" | "events"
ApiDirective    ::= "@basePath" "(" STRING ")" NEWLINE
                  | "@auth"     "(" AuthScheme ")" NEWLINE
AuthScheme      ::= "bearer" | "apiKey" | "basic" | "none"

RouteDecl       ::= RestRoute | RpcRoute | EventRoute

RestRoute       ::= HttpMethod RoutePath "->" ReturnType Decorator* NEWLINE
HttpMethod      ::= "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS"
RoutePath       ::= "/" (PathSegment ("/" PathSegment)*)?
PathSegment     ::= IDENT | ":" IDENT

RpcRoute        ::= "rpc" "stream"? IDENT "(" TypeExpr ")" "->" ReturnType Decorator* NEWLINE

EventRoute      ::= EventDirection IDENT "->" TypeExpr Decorator* NEWLINE
EventDirection  ::= "publish" | "subscribe"

ReturnType      ::= TypeExpr | "void"

Decorator       ::= "@" IDENT DecoratorArgs?
DecoratorArgs   ::= "(" DecoratorArgList ")"
DecoratorArgList::= DecoratorArg ("," DecoratorArg)*
DecoratorArg    ::= (IDENT ":")? DecoratorValue
DecoratorValue  ::= STRING | INT | FLOAT | IDENT | TypeExpr

RawBlock        ::= "@raw" "(" IDENT ")" "{" NEWLINE RawPair* "}"
RawPair         ::= STRING ":" RawValue NEWLINE
RawValue        ::= STRING | INT | FLOAT | "true" | "false" | "null"

IDENT           ::= [a-zA-Z_][a-zA-Z0-9_]*
STRING          ::= '"' [^"]* '"'
INT             ::= [0-9]+
FLOAT           ::= [0-9]+ "." [0-9]+
NEWLINE         ::= "\n" | "\r\n"
COMMENT         ::= "#" [^\n]* NEWLINE   (ignored by parser)
WHITESPACE      ::= (" " | "\t")+        (ignored everywhere)
```
