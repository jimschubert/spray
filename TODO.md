# TODO

## Decorator Ideas

These are potential decorator extensions to consider in future iterations.

- `@cached(ttl: 300)` — Could be used for cache headers or middleware configuration of generated outputs.
- validations or constraints such as `@threshold(value: 3.14)`, `@pattern(/[a-z]/)`, `@min(value)`, `@max(value)`?
  supported by OpenAPI schema generation and likely a useful addition later. Or `@contraint(min=value,max=value,pattern=/[a-z]/)`?
- `@example(label: "hello")` — Named example values for fields or routes. Could drive sample payloads in OpenAPI
  output. Need to review later if `@raw(openapi)` (or other) solves this and simplifies spray.
- pub/sub specific decorators:
  - `@routing(event_type: user.created)` — Topic or routing key mapping
  - `@filter(region: US)` — Subscription filters
  - `@headers(X-Source: auth-service)`
  - `@deadLetter()` — Indicates that failed messages should be sent to a dead letter queue (would this need a name?)

## Streaming

- REST streaming (`@stream` decorator or similar). Server-Sent Events (SSE) is a common pattern for REST APIs that
  push data to clients. Maybe something like `GET /feed -> FeedItem @stream`? Only RPC currently supports streaming (`rpc stream`).

