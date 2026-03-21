package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/internal/ast"
)

func TestParseApi_RouteRPC(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectMethod     string
		expectInput      string
		expectReturnType string
		expectStreaming  bool
		expectDecorators []string
		wantErr          bool
	}{
		{
			name:             "RPC route without input parameter",
			input:            `rpc getUser -> User`,
			expectMethod:     "getUser",
			expectInput:      "",
			expectReturnType: "User",
			expectStreaming:  false,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "RPC route with input parameter",
			input:            `rpc getUser(GetUserInput) -> User`,
			expectMethod:     "getUser",
			expectInput:      "GetUserInput",
			expectReturnType: "User",
			expectStreaming:  false,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "RPC route with decorators",
			input:            `rpc getUser -> User @query(PaginationInput) @errors(404)`,
			expectMethod:     "getUser",
			expectInput:      "",
			expectReturnType: "User",
			expectStreaming:  false,
			expectDecorators: []string{"query", "errors"},
			wantErr:          false,
		},
		{
			name:             "RPC stream route",
			input:            `rpc stream getFeed(GetFeedInput) -> FeedItem`,
			expectMethod:     "getFeed",
			expectInput:      "GetFeedInput",
			expectReturnType: "FeedItem",
			expectStreaming:  true,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "RPC stream route without input",
			input:            `rpc stream getFeed -> FeedItem`,
			expectMethod:     "getFeed",
			expectInput:      "",
			expectReturnType: "FeedItem",
			expectStreaming:  true,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "RPC route with array return type",
			input:            `rpc listUsers -> User[]`,
			expectMethod:     "listUsers",
			expectInput:      "",
			expectReturnType: "User[]",
			expectStreaming:  false,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "RPC route with optional return type",
			input:            `rpc findUser(FindUserInput) -> User?`,
			expectMethod:     "findUser",
			expectInput:      "FindUserInput",
			expectReturnType: "User?",
			expectStreaming:  false,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "RPC route with generic return type",
			input:            `rpc getPage(PageInput) -> Page<User>`,
			expectMethod:     "getPage",
			expectInput:      "PageInput",
			expectReturnType: "Page<User>",
			expectStreaming:  false,
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:    "error on missing arrow",
			input:   `rpc getUser User`,
			wantErr: true,
		},
		{
			name:    "error on missing return type",
			input:   `rpc getUser ->`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := `api TestApi @style(rpc) {
				` + tc.input + `
			}`
			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(src)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			var apiSpec *ast.Api
			for _, spec := range stencil.Specs {
				if a, ok := spec.(*ast.Api); ok {
					apiSpec = a
					break
				}
			}
			assert.True(t, apiSpec != nil, "expected to find an Api spec")
			assert.Equal(t, "TestApi", apiSpec.Name.Value)
			assert.Equal(t, 1, len(apiSpec.Routes), "expected exactly one route")

			route := apiSpec.Routes[0]
			rpcRoute, ok := route.(*ast.RpcRoute)
			assert.True(t, ok, "expected route to be RpcRoute")
			assert.Equal(t, tc.expectMethod, rpcRoute.Name.Value)
			assert.Equal(t, tc.expectReturnType, rpcRoute.Return.String())
			assert.Equal(t, tc.expectStreaming, rpcRoute.Streaming)

			if tc.expectInput != "" {
				assert.Equal(t, tc.expectInput, rpcRoute.Input.String())
			}

			if tc.expectDecorators != nil {
				assert.Equal(t, len(tc.expectDecorators), len(rpcRoute.Decorators))
				for i, expectedDec := range tc.expectDecorators {
					assert.Equal(t, expectedDec, rpcRoute.Decorators[i].Name)
				}
			}
		})
	}
}

func TestParseApi_RPC(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectName       string
		expectStyle      ast.ApiStyle
		expectDecorators []string
		expectDirectives []string
		routeCount       int
		wantErr          bool
	}{
		{
			name:        "empty RPC api is allowed",
			input:       "api MyRpcApi @style(rpc) { }",
			expectName:  "MyRpcApi",
			expectStyle: ast.RPC,
			routeCount:  0,
			wantErr:     false,
		},
		{
			name: "RPC api from specification",
			input: `api FeedService @style(rpc) {
  rpc GetUser(GetUserInput) -> User
  rpc stream GetFeed(GetFeedInput) -> FeedItem
  rpc stream Chat(ChatInput) -> ChatMessage
}`,
			expectName:  "FeedService",
			expectStyle: ast.RPC,
			routeCount:  3,
			wantErr:     false,
		},
		{
			name: "RPC api with decorators and directives",
			input: `api MyRpcApi @style(rpc) @version(2) {
  @basePath("/api/rpc")

  rpc GetUser(GetUserInput) -> User
  rpc CreatePost(CreatePostInput) -> Post
}`,
			expectName:       "MyRpcApi",
			expectStyle:      ast.RPC,
			expectDecorators: []string{"style", "version"},
			expectDirectives: []string{"basePath"},
			routeCount:       2,
			wantErr:          false,
		},
		{
			name: "RPC api with mixed streaming and non-streaming routes",
			input: `api MixedService @style(rpc) {
  rpc GetUser -> User
  rpc stream WatchUpdates(WatchInput) -> Update
  rpc ProcessData(ProcessInput) -> Result
}`,
			expectName:  "MixedService",
			expectStyle: ast.RPC,
			routeCount:  3,
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			var apiSpec *ast.Api
			for _, spec := range stencil.Specs {
				if a, ok := spec.(*ast.Api); ok {
					apiSpec = a
					break
				}
			}
			assert.True(t, apiSpec != nil, "expected to find an Api spec")
			assert.Equal(t, tc.expectName, apiSpec.Name.Value)
			assert.Equal(t, tc.expectStyle, apiSpec.Style)

			if tc.expectDecorators != nil {
				assert.Equal(t, len(tc.expectDecorators), len(apiSpec.ApiDecorators))
				for i, expectedDec := range tc.expectDecorators {
					assert.Equal(t, expectedDec, apiSpec.ApiDecorators[i].Name)
				}
			}

			if tc.expectDirectives != nil {
				assert.Equal(t, len(tc.expectDirectives), len(apiSpec.ApiDirectives))
				for i, expectedDir := range tc.expectDirectives {
					assert.Equal(t, expectedDir, apiSpec.ApiDirectives[i].Name)
				}
			}

			assert.Equal(t, tc.routeCount, len(apiSpec.Routes))

			for _, route := range apiSpec.Routes {
				_, ok := route.(*ast.RpcRoute)
				assert.True(t, ok, "expected route to be RpcRoute")
			}
		})
	}
}
