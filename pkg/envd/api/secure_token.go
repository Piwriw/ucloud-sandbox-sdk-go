package api

// SecureToken is envd's access token as it travels on the wire: a plain JSON
// string.
//
// This declaration is hand-written, not generated. envd's OpenAPI spec pins the
// Go type of /init's accessToken field with `x-go-type: SecureToken`, naming a
// type that only exists inside envd itself — a memguard-backed struct that
// keeps the token in locked memory (see
// submodules/e2b-runtime/packages/envd/internal/api/secure_token.go). A client
// has no use for that machinery, and /init is an orchestrator-internal endpoint
// this SDK never calls, so the alias below is enough to make the generated code
// compile while keeping the JSON encoding identical.
type SecureToken = string
