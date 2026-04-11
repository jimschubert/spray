# spray Plugin System

spray ships built-in emitters for Markdown, JSON Schema, and Mermaid. For any other output format you can reference
an **external emitter plugin**. A plugin is an executable that reads JSON from `stdin` and writes JSON to `stdout`.

---

## Invoking a Plugin

```sh
spray create ext <name> [flags] <files...>
```

`<name>` must match an executable named `spray-emitter-<name>` on the search path (see [Discovery](#discovery)).

```sh
# emit OpenAPI from two stencil files using a "spray-emitter-openapi" plugin
spray create ext openapi -o ./out ./api/*.stencil
```

---

## Discovery

spray searches for plugin executables in this order:

1. `~/.spray/plugins/spray-emitter-<name>`
2. Every directory in `$PATH`

---

## Protocol

Plugins communicate via a single JSON round-trip over `stdin`/`stdout`:

1. spray writes a [`PluginRequest`](#pluginrequest) JSON object to the plugin's `stdin` and closes the stream.
2. The plugin reads the request, processes it, and writes a [`PluginResponse`](#pluginresponse) JSON object to `stdout`.
3. spray reads the response and writes output files to disk.

stderr is forwarded to the user's terminal.

---

## Request

### `PluginRequest`

```jsonc
{
  // "emit_all" or "emit_one"
  "command": "emit_all",

  // Fully resolved definition of all input stencil files
  "schema": { ... },

  // Only present with "emit_one": "model", "input", "enum", or "api"
  "spec_type": "model",

  // Only present with "emit_one": simple name of the spec to emit
  "spec_name": "User"
}
```

### Commands

- **`emit_all`**: Return one or more `Output` objects for all specs.
- **`emit_one`**: Return exactly one `Output` object for the spec identified by `spec_type` and `spec_name`.

---

## Response

### `PluginResponse`

```jsonc
{
  // One or more files to write to the output directory
  "outputs": [
    {
      // Destination filename (relative path within the -o output directory)
      "filename": "user.yaml",

      // "text" or "binary"
      "content_type": "text",

      // "none" for UTF-8 text; "base64" for binary content
      "encoding": "none",

      // File content
      "content": "..."
    }
  ],

  // If non-empty, spray reports this error and writes no files
  "error": ""
}
```

---

## Schema Reference

### `PluginSchema`

All collection fields are omitted when empty.

```jsonc
{
  // Primary namespace from the first stencil file (omitted if no explicit namespace)
  "namespace": "acme.v1",

  // All model declarations
  "models": [ ... ],

  // All input declarations
  "inputs": [ ... ],

  // All enum declarations
  "enums": [ ... ],

  // All routes, flattened from all api blocks (one entry per route)
  "apis": [ ... ],

  // Concrete generic instantiations (e.g. Page<User> -> PageUser)
  "monomorphs": [ ... ],

  // Type aliases grouped by namespace FQN
  "aliases": { "acme.v1": [ ... ] },

  // @raw blocks grouped by the FQN of the node they're attached to
  "extensions": { "acme.v1.User": [ ... ] }
}
```

---

### `PluginTypeRef`

Boolean flags are always present.

```jsonc
{
  // Fully qualified name ("acme.v1.User") or scalar name ("string", "uuid")
  "fqn": "acme.v1.User",

  // true when declared as T[]
  "array": true,

  // true when declared as T?
  "optional": false,

  // true for built-in scalar types
  "scalar": false,

  // Generic type arguments (e.g. ["acme.v1.User"] for Page<User>)
  "args": []
}
```

---

### `PluginModel` / `PluginInput`

```jsonc
{
  // Simple name (no namespace prefix)
  "name": "User",

  // Owning namespace FQN (omitted if no explicit namespace)
  "namespace": "acme.v1",

  // Comment block immediately preceding the declaration
  "head_comment": "A registered user account.",

  // Ordered list of fields
  "fields": [ ... ]
}
```

---

### `PluginField`

```jsonc
{
  // Field name
  "name": "email",

  // Resolved type reference
  "type": { "fqn": "string", "array": false, "optional": false, "scalar": true },

  // Comment block preceding the field
  "head_comment": "",

  // Trailing inline comment
  "line_comment": "must be unique",

  // Decorators applied to this field
  "decorators": [ { "name": "unique", "args": [] } ]
}
```

---

### `PluginEnum`

```jsonc
{
  "name": "Role",
  "namespace": "acme.v1",
  "values": [ "admin", "member", "guest" ]
}
```

---

### `PluginApi`

One entry per route. The `style` field determines which other fields are present.

```jsonc
{
  // "rest", "rpc", or "events"
  "style": "rest",

  // "METHOD /path" for REST; procedure or event name otherwise
  "name": "GET /users",

  // Owning namespace FQN
  "namespace": "acme.v1",

  // Comment block preceding the route
  "head_comment": "",

  // HTTP verb (REST only)
  "method": "GET",

  // Route path (REST only)
  "path": "/users",

  // Input type (REST with @body/@query, RPC if present; omitted when route takes no input)
  "input": { "fqn": "acme.v1.PaginationInput", "array": false, "optional": false, "scalar": false },

  // Return type ("void" FQN for routes that return nothing)
  "return": { "fqn": "acme.v1.User", "array": true, "optional": false, "scalar": false },

  // true for rpc stream routes (RPC only)
  "streaming": false,

  // "publish" or "subscribe" (events only)
  "direction": "publish",

  // Route-level decorators
  "decorators": []
}
```

---

### `PluginMonomorph`

Concrete instantiation of a generic model.

```jsonc
{
  // Generated concrete name
  "name": "PageUser",

  // Namespace of the original generic definition
  "namespace": "acme.v1",

  // FQN of the generic template
  "original": "acme.v1.Page",

  // Concrete type arguments substituted into this instance
  "args": [ { "fqn": "acme.v1.User", "array": false, "optional": false, "scalar": false } ]
}
```

---

### `PluginDecorator` / `PluginDecoratorArg`

```jsonc
{
  "name": "default",
  "args": [
    // EITHER: positional arg (e.g. @default(member)): only "value"
    { "value": "member" },

    // OR: named arg (e.g. @relation(field: authorId)): both "name" and "value"
    { "name": "key", "value": "someValue" }
  ]
}
```

---

### `PluginExtension`

```jsonc
{
  // Target emitter name from @raw(target)
  "target": "openapi",

  // Key-value pairs from the @raw block
  "pairs": [
    { "key": "x-internal", "value": true },
    { "key": "x-webhook-signature", "value": "sha256" }
  ]
}
```

---

## Example

### `.stencil` Source

```stencil
namespace acme.v1

type Email = string

enum Role {
  admin
  member
  guest
}

# A registered user account.
model User {
  id:    uuid  @primary
  email: Email @unique
  role:  Role  @default(member)
  name:  string?
}

model Page<T> {
  items: T[]
  total: int
}

input CreateUserInput {
  email: string
  name:  string?
  role:  Role @default(member)
}

api UserService @style(rest) {
  @basePath("/api/v1/users")

  GET  /     -> Page<User>
    @query(PaginationInput)

  GET  /:id  -> User
    @errors(404)

  POST /     -> User
    @body(CreateUserInput)
    @errors(400, 409)
}
```

### Corresponding `PluginRequest` (truncated)

```json
{
  "command": "emit_all",
  "schema": {
    "namespace": "acme.v1",
    "models": [
      {
        "name":         "User",
        "namespace":    "acme.v1",
        "head_comment": "A registered user account.",
        "fields": [
          {
            "name": "id",
            "type": { "fqn": "uuid", "array": false, "optional": false, "scalar": true },
            "decorators": [ { "name": "primary" } ]
          },
          {
            "name": "email",
            "type": { "fqn": "acme.v1.Email", "array": false, "optional": false, "scalar": false },
            "decorators": [ { "name": "unique" } ]
          },
          {
            "name": "role",
            "type": { "fqn": "acme.v1.Role", "array": false, "optional": false, "scalar": false },
            "decorators": [ { "name": "default", "args": [ { "value": "member" } ] } ]
          },
          {
            "name": "name",
            "type": { "fqn": "string", "array": false, "optional": true, "scalar": true }
          }
        ]
      },
      {
        "name":      "Page",
        "namespace": "acme.v1",
        "fields": [
          {
            "name": "items",
            "type": { "fqn": "T", "array": true, "optional": false, "scalar": false }
          },
          {
            "name": "total",
            "type": { "fqn": "int", "array": false, "optional": false, "scalar": true }
          }
        ]
      }
    ],
    "inputs": [
      {
        "name":      "CreateUserInput",
        "namespace": "acme.v1",
        "fields": [
          { "name": "email", "type": { "fqn": "string",      "array": false, "optional": false, "scalar": true } },
          { "name": "name",  "type": { "fqn": "string",      "array": false, "optional": true,  "scalar": true } },
          { "name": "role",  "type": { "fqn": "acme.v1.Role","array": false, "optional": false, "scalar": false },
            "decorators": [ { "name": "default", "args": [ { "value": "member" } ] } ] }
        ]
      }
    ],
    "enums": [
      {
        "name":      "Role",
        "namespace": "acme.v1",
        "values":    [ "admin", "member", "guest" ]
      }
    ],
    "apis": [
      {
        "style": "rest", "name": "GET /",    "namespace": "acme.v1",
        "method": "GET",  "path": "/",
        "return": { "fqn": "acme.v1.PageUser", "array": false, "optional": false, "scalar": false },
        "streaming": false,
        "decorators": [ { "name": "query", "args": [ { "value": "PaginationInput" } ] } ]
      },
      {
        "style": "rest", "name": "GET /:id", "namespace": "acme.v1",
        "method": "GET",  "path": "/:id",
        "return": { "fqn": "acme.v1.User", "array": false, "optional": false, "scalar": false },
        "streaming": false,
        "decorators": [ { "name": "errors", "args": [ { "value": "404" } ] } ]
      },
      {
        "style": "rest", "name": "POST /",   "namespace": "acme.v1",
        "method": "POST", "path": "/",
        "input":  { "fqn": "acme.v1.CreateUserInput", "array": false, "optional": false, "scalar": false },
        "return": { "fqn": "acme.v1.User", "array": false, "optional": false, "scalar": false },
        "streaming": false,
        "decorators": [
          { "name": "body",   "args": [ { "value": "CreateUserInput" } ] },
          { "name": "errors", "args": [ { "value": "400" }, { "value": "409" } ] }
        ]
      }
    ],
    "monomorphs": [
      {
        "name": "PageUser", "namespace": "acme.v1", "original": "acme.v1.Page",
        "args": [ { "fqn": "acme.v1.User", "array": false, "optional": false, "scalar": false } ]
      }
    ],
    "aliases": {
      "acme.v1": [
        { "name": "Email", "type": { "fqn": "string", "array": false, "optional": false, "scalar": true } }
      ]
    }
  }
}
```

### Minimal `PluginResponse`

```json
{
  "outputs": [
    {
      "filename":     "acme.v1.yaml",
      "content_type": "text",
      "encoding":     "none",
      "content":      "... your generated content here ..."
    }
  ]
}
```

---

## Writing a Plugin

A plugin can be written in any language:

- Read a `PluginRequest` JSON object from `stdin` (terminated by EOF).
- Write a `PluginResponse` JSON object to `stdout`.
- Exit `0` on success; exit non-zero or set `"error"` in the response on failure.

### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

type PluginRequest struct {
    Command string       `json:"command"`
    Schema  PluginSchema `json:"schema"`
}

type PluginSchema struct {
    Namespace string        `json:"namespace,omitempty"`
    Models    []PluginModel `json:"models,omitempty"`
    // Inputs, Enums, Apis, … — add fields as needed.
}

type PluginModel struct {
    Name      string        `json:"name"`
    Namespace string        `json:"namespace,omitempty"`
    Fields    []PluginField `json:"fields"`
}

type PluginField struct {
    Name string       `json:"name"`
    Type   PluginType `json:"type"`
}

type PluginType struct {
    FQN    string `json:"fqn"`
    IsArray bool   `json:"array"`
}

type PluginResponse struct {
    Outputs []PluginOutput `json:"outputs"`
    Error   string         `json:"error,omitempty"`
}

type PluginOutput struct {
    Filename    string `json:"filename"`
    ContentType string `json:"content_type"`
    Encoding    string `json:"encoding"`
    Content     string `json:"content"`
}

func main() {
    var req PluginRequest
    if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
        respond(PluginResponse{Error: fmt.Sprintf("decode request: %v", err)})
        return
    }

    output, err := generate(req.Schema)
    if err != nil {
        respond(PluginResponse{Error: fmt.Sprintf("generate: %v", err)})
        return
    }

    respond(PluginResponse{
        Outputs: []PluginOutput{{
            Filename:    "output.txt",
            ContentType: "text",
            Encoding:    "none",
            Content:     output,
        }},
    })
}

func respond(r PluginResponse) {
    if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
        fmt.Fprintf(os.Stderr, "encode response: %v\n", err)
        os.Exit(1)
    }
}

func generate(schema PluginSchema) (string, error) {
    return "implement me", nil
}
```

The types above are minimal. Define additional structs (`PluginEnum`, `PluginApi`, etc.) matching the [schema reference](#schema-reference) as needed.

### Python

```python
#!/usr/bin/env python3
import json, sys

def generate(schema):
    lines = [f"# {schema.get('namespace', 'default')}\n"]
    for model in schema.get("models", []):
        lines.append(f"## {model['name']}\n")
        for field in model.get("fields", []):
            lines.append(f"- `{field['name']}`: {field['type']['fqn']}\n")
    return "".join(lines)

req = json.load(sys.stdin)
content = generate(req["schema"])

json.dump({
    "outputs": [{
        "filename":     "output.md",
        "content_type": "text",
        "encoding":     "none",
        "content":      content,
    }]
}, sys.stdout)
```

The `content` field is always a string. Use `"encoding": "none"` for UTF-8 text. Use `"encoding": "base64"` for binary content (e.g. `base64.b64encode(data).decode()`).

### Naming and Installation

Name your executable `spray-emitter-<name>` and place it in `~/.spray/plugins/` or anywhere on your `$PATH`:

```sh
# install to user plugins directory
cp my-plugin ~/.spray/plugins/spray-emitter-myformat
chmod +x ~/.spray/plugins/spray-emitter-myformat

# invoke
spray create ext myformat -o ./out ./api/*.stencil
```

---

## Consuming `@raw` Extensions

Plugins should look up their target name in the `extensions` map to pick up any `@raw` annotations:

```python
extensions = schema.get("extensions", {})
for fqn, blocks in extensions.items():
    for block in blocks:
        if block["target"] == "myformat":
            for pair in block["pairs"]:
                print(f"{fqn}: {pair['key']} = {pair['value']}")
```

Plugins should ignore blocks with target names they don't recognize.
