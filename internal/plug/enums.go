package plug

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Command specifies the plugin operation to perform.
type Command string

const (
	CommandEmitAll Command = "emit_all"
	CommandEmitOne Command = "emit_one"
)

func (c Command) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c *Command) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	case "emit_all":
		*c = CommandEmitAll
	case "emit_one":
		*c = CommandEmitOne
	default:
		return fmt.Errorf("invalid Command %q", s)
	}
	return nil
}

type ContentType string

const (
	ContentTypeText   ContentType = "text"
	ContentTypeBinary ContentType = "binary"
)

func (c ContentType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c *ContentType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	case "text":
		*c = ContentTypeText
	case "binary":
		*c = ContentTypeBinary
	default:
		return fmt.Errorf("invalid ContentType %q", s)
	}
	return nil
}

// Encoding specifies how output content is encoded in JSON.
// This is required because plugin output is transmitted as JSON, so binary content must be encoded (e.g. base64) to avoid corruption.
type Encoding string

const (
	EncodingNone   Encoding = "none"
	EncodingBase64 Encoding = "base64"
)

func (e Encoding) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(e))
}

func (e *Encoding) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch strings.ToLower(s) {
	case "none":
		*e = EncodingNone
	case "base64":
		*e = EncodingBase64
	default:
		return fmt.Errorf("invalid Encoding %q", s)
	}
	return nil
}

type SpecType string

const (
	SpecModel SpecType = "model"
	SpecInput SpecType = "input"
	SpecEnum  SpecType = "enum"
	SpecApi   SpecType = "api"
)

func (s SpecType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *SpecType) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch strings.ToLower(v) {
	case "model":
		*s = SpecModel
	case "input":
		*s = SpecInput
	case "enum":
		*s = SpecEnum
	case "api":
		*s = SpecApi
	default:
		return fmt.Errorf("invalid SpecType %q", v)
	}
	return nil
}

// Method specifies an HTTP verb.
type Method string

const (
	MethodGet     Method = "GET"
	MethodPost    Method = "POST"
	MethodPut     Method = "PUT"
	MethodPatch   Method = "PATCH"
	MethodDelete  Method = "DELETE"
	MethodHead    Method = "HEAD"
	MethodOptions Method = "OPTIONS"
)

func (m Method) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(m))
}

func (m *Method) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch strings.ToUpper(v) {
	case "GET":
		*m = MethodGet
	case "POST":
		*m = MethodPost
	case "PUT":
		*m = MethodPut
	case "PATCH":
		*m = MethodPatch
	case "DELETE":
		*m = MethodDelete
	case "HEAD":
		*m = MethodHead
	case "OPTIONS":
		*m = MethodOptions
	default:
		return fmt.Errorf("invalid Method %q", v)
	}
	return nil
}

type RouteStyle string

const (
	RouteStyleRest   RouteStyle = "rest"
	RouteStyleRpc    RouteStyle = "rpc"
	RouteStyleEvents RouteStyle = "events"
)

func (r RouteStyle) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

func (r *RouteStyle) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch strings.ToLower(v) {
	case "rest":
		*r = RouteStyleRest
	case "rpc":
		*r = RouteStyleRpc
	case "events":
		*r = RouteStyleEvents
	default:
		return fmt.Errorf("invalid RouteStyle %q", v)
	}
	return nil
}

type EventDirection string

const (
	EventPublish   EventDirection = "publish"
	EventSubscribe EventDirection = "subscribe"
)

func (e EventDirection) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(e))
}

func (e *EventDirection) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch strings.ToLower(v) {
	case "publish":
		*e = EventPublish
	case "subscribe":
		*e = EventSubscribe
	default:
		return fmt.Errorf("invalid EventDirection %q", v)
	}
	return nil
}
