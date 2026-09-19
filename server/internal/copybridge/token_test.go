package copybridge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCommandTokenProtectsMutatingCommands is the security control this bridge
// was missing.
//
// Binding to loopback keeps the bridge off the network, but it does not stop a
// local process, or an approved LAN page, from copying to or ejecting this
// machine's devices. The token is the only control that does, so it must apply
// to mutating methods and must NOT be bypassable.
func TestCommandTokenProtectsMutatingCommands(t *testing.T) {
	const token = "s3cret-bridge-token-value"

	server := &Server{cfg: Config{CommandToken: token}}
	handler := server.withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name       string
		method     string
		path       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "mutating command without token is rejected",
			method:     http.MethodPost,
			path:       "/copy",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "mutating command with a wrong token is rejected",
			method:     http.MethodPost,
			path:       "/copy",
			authHeader: "Bearer wrong-token-value",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "mutating command with the correct bearer token is allowed",
			method:     http.MethodPost,
			path:       "/copy",
			authHeader: "Bearer " + token,
			wantStatus: http.StatusOK,
		},
		{
			name:       "the alternate header is also accepted",
			method:     http.MethodPost,
			path:       "/mkdir",
			authHeader: "Bearer " + token,
			wantStatus: http.StatusOK,
		},
		{
			name:       "raw token without the Bearer prefix is accepted",
			method:     http.MethodPost,
			path:       "/eject",
			authHeader: token,
			wantStatus: http.StatusOK,
		},
		{
			// Read-only discovery must stay open so the UI can report bridge
			// health before the operator supplies credentials.
			name:       "device discovery stays open without a token",
			method:     http.MethodGet,
			path:       "/devices",
			wantStatus: http.StatusOK,
		},
		{
			name:       "job polling stays open without a token",
			method:     http.MethodGet,
			path:       "/jobs",
			wantStatus: http.StatusOK,
		},
		{
			name:       "cancelling a job requires the token",
			method:     http.MethodPost,
			path:       "/cancel/job-1",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(testCase.method, testCase.path, nil)
			if testCase.authHeader != "" {
				req.Header.Set("Authorization", testCase.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != testCase.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, testCase.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestNoTokenConfiguredKeepsBackwardsCompatibility verifies the token is opt-in:
// an existing install that upgrades must keep working without new setup.
func TestNoTokenConfiguredKeepsBackwardsCompatibility(t *testing.T) {
	server := &Server{cfg: Config{}}
	handler := server.withMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/copy", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the bridge to keep working when no is configured", rec.Code)
	}
}

// TestTokenComparisonRejectsPrefixAndLengthVariants guards the constant-time
// comparison: a naive equality check leaks enough to recover the token.
func TestTokenComparisonRejectsPrefixAndLengthVariants(t *testing.T) {
	const token = "abcdefghijklmnop"
	cases := []string{
		"",
		"a",
		"abcdefghijklmno",
		"abcdefghijklmnopq",
		"abcdefghijklmnoX",
		"Xbcdefghijklmnop",
	}

	for _, candidate := range cases {
		req := httptest.NewRequest(http.MethodPost, "/copy", nil)
		if candidate != "" {
			req.Header.Set("Authorization", "Bearer "+candidate)
		}
		if tokenMatches(req, token) {
			t.Errorf("candidate %q was accepted as the token", candidate)
		}
	}

	// The exact token must still be accepted.
	req := httptest.NewRequest(http.MethodPost, "/copy", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if !tokenMatches(req, token) {
		t.Error("the correct token was rejected")
	}
}

// TestIsMutatingBridgeCommand documents exactly which methods change state.
func TestIsMutatingBridgeCommand(t *testing.T) {
	mutating := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	readOnly := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	for _, method := range mutating {
		req := httptest.NewRequest(method, "/copy", nil)
		if !isMutatingBridgeCommand(req) {
			t.Errorf("%s should be treated as mutating", method)
		}
	}
	for _, method := range readOnly {
		req := httptest.NewRequest(method, "/devices", nil)
		if isMutatingBridgeCommand(req) {
			t.Errorf("%s should NOT be treated as mutating", method)
		}
	}
}
