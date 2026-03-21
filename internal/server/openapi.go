package server

// OpenAPIJSON 最小 OpenAPI 3 描述（与实现同步维护）。
const OpenAPIJSON = `{
  "openapi": "3.0.3",
  "info": {"title": "opencode-go", "version": "0.0.1"},
  "paths": {
    "/v1/health": {"get": {"responses": {"200": {"description": "ok"}}}},
    "/v1/sessions": {
      "get": {"responses": {"200": {"description": "list"}}},
      "post": {"responses": {"200": {"description": "create"}}}
    },
    "/v1/sessions/{id}/messages": {"get": {"responses": {"200": {"description": "messages"}}}},
    "/v1/sessions/{id}/complete": {"post": {"responses": {"200": {"description": "complete"}}}},
    "/v1/openapi.json": {"get": {"responses": {"200": {"description": "contract"}}}},
    "/v1/acp/session/event": {"post": {"responses": {"202": {"description": "accepted"}}}}
  }
}`
