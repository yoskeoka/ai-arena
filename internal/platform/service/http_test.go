package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func TestOperatorAPIVersionIsPublicAndHasExactJSONShape(t *testing.T) {
	const versionSHA = "0123456789abcdef0123456789abcdef01234567"
	api := (&OperatorAPI{auth: &AuthService{}}).WithVersion(versionSHA)
	response := httptest.NewRecorder()

	api.Handler().ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/version", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("GET /version status = %d, body = %s", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("GET /version Content-Type = %q, want application/json", contentType)
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(version) error = %v", err)
	}
	if len(payload) != 1 || payload["version_sha"] != versionSHA {
		t.Fatalf("GET /version payload = %#v, want exactly version_sha", payload)
	}
}

func TestOperatorAPIHealthzReportsLivenessAndWorkerReadiness(t *testing.T) {
	for _, test := range []struct {
		name        string
		workerReady bool
		wantWorker  string
	}{
		{name: "pending", workerReady: false, wantWorker: "NOT_READY"},
		{name: "ready", workerReady: true, wantWorker: "OK"},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := (&OperatorAPI{auth: &AuthService{}}).WithWorkerReadiness(func() bool {
				return test.workerReady
			})
			response := httptest.NewRecorder()

			api.Handler().ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil))

			if response.Code != http.StatusOK {
				t.Fatalf("GET /healthz status = %d, body = %s", response.Code, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("GET /healthz Content-Type = %q, want application/json", contentType)
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal(health) error = %v", err)
			}
			if len(payload) != 2 || payload["api"] != "OK" || payload["worker"] != test.wantWorker {
				t.Fatalf("GET /healthz payload = %#v, want api=OK worker=%s", payload, test.wantWorker)
			}
		})
	}
}

func TestOperatorAPIAdmitsGameBundleWithCreatedResponse(t *testing.T) {
	store, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	registryStore, err := registry.NewInMemoryStore()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewWASIResolver(store)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := registry.New(registryStore, resolver)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := NewArtifactAdmissionService(store, reg)
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("bundle", "game.arena.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(gameBundle(t)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/game-bundles", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	(&OperatorAPI{artifactAdmission: admission}).handleGameBundleUpload(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/game-bundles status = %d, body = %s", response.Code, response.Body.String())
	}
	var admitted GameBundleAdmissionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &admitted); err != nil {
		t.Fatalf("json.Unmarshal(admitted) error = %v", err)
	}
	if admitted.GameID != "test" || admitted.GameVersion != "2.1.0" || admitted.ArtifactID == "" {
		t.Fatalf("admitted = %+v", admitted)
	}
	if admitted.BuildMode != string(registry.BuildModeWASMWASI) || admitted.BuilderID == "" {
		t.Fatalf("admitted runtime identity = %+v", admitted)
	}
	if len(admitted.SupportedRulesets) != 1 || admitted.SupportedRulesets[0] != "regular" {
		t.Fatalf("admitted.SupportedRulesets = %v", admitted.SupportedRulesets)
	}
}

func TestOperatorAPIRetiresPresetMatches(t *testing.T) {
	commands := newTestCommandService(t)
	queue := NewInMemoryQueueStore()
	queries, err := NewQueryService(queue)
	if err != nil {
		t.Fatal(err)
	}
	general := newTestGeneralSubmissionService(t)
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, queue), DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/preset-matches", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("POST /api/v1/preset-matches status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestOperatorAPICreateSignupInvite(t *testing.T) {
	queue := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, queue)
	queries, err := NewQueryService(queue)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	requests := newTestMatchRequestService(t, general, commands, queue)
	authStore := &memoryAuthStore{
		identities: map[string]AuthPrincipal{
			authIdentityKey(AuthIdentity{Provider: authProviderGitHub, Subject: "12345"}): {
				AccountID:     "account-operator",
				Provider:      "github",
				ProviderLogin: "operator-dev",
				Roles:         []string{"operator"},
			},
		},
	}
	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:     "client-id",
		GitHubClientSecret: "client-secret",
	}, authStore, fakeGitHubAuthProvider{})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, requests, DirectArtifactAccessIssuer{}, auth)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	sessionToken, err := authStore.CreateSession(context.Background(), "account-operator", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/signup-invites", bytes.NewBufferString(`{"role":"developer","ttl":"48h"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionToken})
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/signup-invites status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var invite SignupInviteResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &invite); err != nil {
		t.Fatalf("json.Unmarshal(invite) error = %v", err)
	}
	if invite.Role != "developer" {
		t.Fatalf("invite.Role = %q, want developer", invite.Role)
	}
	if invite.InviteToken == "" {
		t.Fatal("invite.InviteToken = empty, want issued token")
	}
	if invite.InviteURL != signupInviteURL(invite.InviteToken) {
		t.Fatalf("invite.InviteURL = %q, want login URL for token", invite.InviteURL)
	}
	if got := authStore.invites[invite.InviteToken]; got != "developer" {
		t.Fatalf("invite store role = %q, want developer", got)
	}
	if !invite.ExpiresAt.After(time.Now().UTC().Add(24 * time.Hour)) {
		t.Fatalf("invite.ExpiresAt = %s, want TTL longer than 24h", invite.ExpiresAt)
	}
}

func TestOperatorAPIBotRevisionUsesAuthenticatedOwner(t *testing.T) {
	authStore := &memoryAuthStore{identities: map[string]AuthPrincipal{authIdentityKey(AuthIdentity{Provider: authProviderGitHub, Subject: "bot-owner"}): {AccountID: "account-owner", Roles: []string{"developer"}}}}
	auth, err := NewAuthService(AuthConfig{GitHubClientID: "client-id", GitHubClientSecret: "client-secret"}, authStore, fakeGitHubAuthProvider{})
	if err != nil {
		t.Fatal(err)
	}
	token, err := authStore.CreateSession(context.Background(), "account-owner", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	api := &OperatorAPI{auth: auth, botOwnership: NewInMemoryBotOwnershipStore()}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/bots", bytes.NewBufferString(`{"scope":{"scope_id":"scope","max_active_bots_per_owner":1},"bot_name":"Alpha","artifact_id":"digest"}`))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	resp := httptest.NewRecorder()
	api.handleBotRevision(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("bot revision status = %d, body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Bot OwnedBot `json:"bot"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Bot.OwnerAccountID != "account-owner" {
		t.Fatalf("owner = %q", payload.Bot.OwnerAccountID)
	}
}

func TestOperatorAPIAllowsConfiguredCORSOrigins(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/active", nil)
	getReq.Header.Set("Origin", "https://staging.ai-arena.pages.dev")
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if got := getResp.Header().Get("Access-Control-Allow-Origin"); got != "https://staging.ai-arena.pages.dev" {
		t.Fatalf("GET allow-origin = %q, want staging Pages origin", got)
	}
	if got := getResp.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("GET vary = %q, want Origin", got)
	}

	optionsReq := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/api/v1/preset-matches", nil)
	optionsReq.Header.Set("Origin", "https://ai-arena.pages.dev")
	optionsResp := httptest.NewRecorder()
	handler.ServeHTTP(optionsResp, optionsReq)
	if optionsResp.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS /api/v1/preset-matches status = %d, want %d", optionsResp.Code, http.StatusNoContent)
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Origin"); got != "https://ai-arena.pages.dev" {
		t.Fatalf("OPTIONS allow-origin = %q, want production Pages origin", got)
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("OPTIONS allow-methods = %q, want %q", got, "GET, POST, OPTIONS")
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, x-ms-useragent" {
		t.Fatalf("OPTIONS allow-headers = %q, want %q", got, "Content-Type, x-ms-useragent")
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("OPTIONS allow-credentials = %q, want true", got)
	}
}

func TestOperatorAPIPreflightCORSContract(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		requestMethod   string
		origin          string
		requested       string
		wantAllow       bool
		wantAllowOrigin string
	}{
		{
			name:            "staging session runtime header",
			path:            "/auth/session",
			requestMethod:   http.MethodGet,
			origin:          "https://staging.ai-arena.pages.dev",
			requested:       "x-ms-useragent",
			wantAllow:       true,
			wantAllowOrigin: "https://staging.ai-arena.pages.dev",
		},
		{
			name:            "production JSON post mixed case and empty tokens",
			path:            "/api/v1/preset-matches",
			requestMethod:   http.MethodPost,
			origin:          "https://ai-arena.pages.dev",
			requested:       " content-type, X-MS-USERAGENT, , ",
			wantAllow:       true,
			wantAllowOrigin: "https://ai-arena.pages.dev",
		},
		{
			name:            "production multipart upload",
			path:            "/api/v1/game-bundles",
			requestMethod:   http.MethodPost,
			origin:          "https://ai-arena.pages.dev",
			requested:       "Content-Type, x-ms-useragent",
			wantAllow:       true,
			wantAllowOrigin: "https://ai-arena.pages.dev",
		},
		{
			name:          "unknown origin",
			path:          "/auth/session",
			requestMethod: http.MethodGet,
			origin:        "https://example.com",
			requested:     "x-ms-useragent",
		},
		{
			name:          "unknown requested header",
			path:          "/auth/session",
			requestMethod: http.MethodGet,
			origin:        "https://staging.ai-arena.pages.dev",
			requested:     "content-type, x-operator-debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := withOperatorCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				t.Fatal("preflight reached the wrapped handler")
			}))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, tt.path, nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", tt.requestMethod)
			req.Header.Set("Access-Control-Request-Headers", tt.requested)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if resp.Code != http.StatusNoContent {
				t.Fatalf("OPTIONS status = %d, want %d", resp.Code, http.StatusNoContent)
			}
			if !tt.wantAllow {
				for _, header := range []string{
					"Access-Control-Allow-Origin",
					"Access-Control-Allow-Headers",
					"Access-Control-Allow-Methods",
					"Access-Control-Allow-Credentials",
				} {
					if got := resp.Header().Get(header); got != "" {
						t.Fatalf("%s = %q, want empty", header, got)
					}
				}
				return
			}

			if got := resp.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowOrigin {
				t.Fatalf("allow-origin = %q, want %q", got, tt.wantAllowOrigin)
			}
			if got := resp.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
				t.Fatalf("allow-methods = %q, want GET, POST, OPTIONS", got)
			}
			if got := resp.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, x-ms-useragent" {
				t.Fatalf("allow-headers = %q, want %q", got, "Content-Type, x-ms-useragent")
			}
			if got := resp.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Fatalf("allow-credentials = %q, want true", got)
			}
			if got := resp.Header().Get("Vary"); got != "Origin" {
				t.Fatalf("vary = %q, want Origin", got)
			}
		})
	}
}

func TestOperatorAPISessionStatusAllowsAnonymousCrossOriginRequest(t *testing.T) {
	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:       "client-id",
		GitHubClientSecret:   "client-secret",
		AllowedReturnOrigins: []string{"https://staging.ai-arena.pages.dev"},
	}, &memoryAuthStore{}, fakeGitHubAuthProvider{})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	resp := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/session", nil)
	req.Header.Set("Origin", "https://staging.ai-arena.pages.dev")
	(&OperatorAPI{auth: auth}).Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("GET /auth/session status = %d, want %d", resp.Code, http.StatusOK)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://staging.ai-arena.pages.dev" {
		t.Fatalf("allow-origin = %q, want staging Pages origin", got)
	}
	var payload SessionStatusResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(session) error = %v", err)
	}
	if payload.AuthMode != authModeEnabled || payload.Authenticated {
		t.Fatalf("session payload = %+v, want enabled unauthenticated session", payload)
	}
}

func TestOperatorAPIDoesNotAllowUnknownCORSOrigin(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/active", nil)
	req.Header.Set("Origin", "https://example.com")
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q, want empty", got)
	}
}

func TestOperatorAPIGeneralRegistrationRoutesRejectUnsupportedMethod(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/v1/game-registrations", nil)
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, body = %s, want %d", resp.Code, resp.Body.String(), http.StatusMethodNotAllowed)
	}
}

func TestOperatorAPIGeneralRegistrationRoutes(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	gameReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/game-registrations", bytes.NewBufferString(`{"game":{"game_id":"echo-count","game_version":"2.0.0","ruleset_version":"phase2-simultaneous-2turn"}}`))
	gameReq.Header.Set("Content-Type", "application/json")
	gameResp := httptest.NewRecorder()
	handler.ServeHTTP(gameResp, gameReq)
	if gameResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/game-registrations status = %d, body = %s", gameResp.Code, gameResp.Body.String())
	}

	aiReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/ai-submissions", bytes.NewBufferString(fmt.Sprintf(`{"game_registration_id":"echo-count-v2-phase2-simultaneous-2turn","artifact_ref":%q}`, repoJoin(t, "testdata/ai/echo/echo-ai-2turn"))))
	aiReq.Header.Set("Content-Type", "application/json")
	aiResp := httptest.NewRecorder()
	handler.ServeHTTP(aiResp, aiReq)
	if aiResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/ai-submissions status = %d, body = %s", aiResp.Code, aiResp.Body.String())
	}

	var createdAI RegisteredAI
	if err := json.Unmarshal(aiResp.Body.Bytes(), &createdAI); err != nil {
		t.Fatalf("json.Unmarshal(createdAI) error = %v", err)
	}
	if createdAI.ValidationState != ValidationReady {
		t.Fatalf("createdAI.ValidationState = %q, want %q", createdAI.ValidationState, ValidationReady)
	}

	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/ai-submissions", nil))
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/ai-submissions status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Items []RegisteredAI `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(listed) error = %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("len(listed.Items) = %d, want 1", len(listed.Items))
	}
}

func TestOperatorAPIMatchRequestRoutes(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	queries, err := NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	requests := newTestMatchRequestService(t, general, commands, store)
	api, err := NewOperatorAPI(commands, queries, general, requests, DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	gameReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/game-registrations", bytes.NewBufferString(`{"game":{"game_id":"echo-count","game_version":"2.0.0","ruleset_version":"phase2-simultaneous-2turn"}}`))
	gameReq.Header.Set("Content-Type", "application/json")
	gameResp := httptest.NewRecorder()
	handler.ServeHTTP(gameResp, gameReq)
	if gameResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/game-registrations status = %d, body = %s", gameResp.Code, gameResp.Body.String())
	}

	aiPayloads := []string{
		fmt.Sprintf(`{"game_registration_id":"echo-count-v2-phase2-simultaneous-2turn","artifact_ref":%q,"display_name":"Echo 1"}`, repoJoin(t, "testdata/ai/echo/echo-ai-2turn")),
		fmt.Sprintf(`{"game_registration_id":"echo-count-v2-phase2-simultaneous-2turn","artifact_ref":%q,"display_name":"Echo 2"}`, repoJoin(t, "testdata/ai/echo/echo-ai-2turn")),
	}
	aiIDs := make([]string, 0, len(aiPayloads))
	for _, payload := range aiPayloads {
		aiReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/ai-submissions", bytes.NewBufferString(payload))
		aiReq.Header.Set("Content-Type", "application/json")
		aiResp := httptest.NewRecorder()
		handler.ServeHTTP(aiResp, aiReq)
		if aiResp.Code != http.StatusCreated {
			t.Fatalf("POST /api/v1/ai-submissions status = %d, body = %s", aiResp.Code, aiResp.Body.String())
		}
		var createdAI RegisteredAI
		if err := json.Unmarshal(aiResp.Body.Bytes(), &createdAI); err != nil {
			t.Fatalf("json.Unmarshal(createdAI) error = %v", err)
		}
		aiIDs = append(aiIDs, createdAI.AISubmissionID)
	}

	matchReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(fmt.Sprintf(`{"game_registration_id":"echo-count-v2-phase2-simultaneous-2turn","participants":[{"player_id":"p1","ai_submission_id":"%s"},{"player_id":"p2","ai_submission_id":"%s"}],"output_dir":%q}`, aiIDs[0], aiIDs[1], t.TempDir())))
	matchReq.Header.Set("Content-Type", "application/json")
	matchResp := httptest.NewRecorder()
	handler.ServeHTTP(matchResp, matchReq)
	if matchResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/match-requests status = %d, body = %s", matchResp.Code, matchResp.Body.String())
	}
	var created MatchRequest
	if err := json.Unmarshal(matchResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(created) error = %v", err)
	}
	if created.LifecycleState != StateQueued {
		t.Fatalf("created.LifecycleState = %q, want %q", created.LifecycleState, StateQueued)
	}

	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/match-requests", nil))
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/match-requests status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Items []MatchRequest `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(listed) error = %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("len(listed.Items) = %d, want 1", len(listed.Items))
	}
	if listed.Items[0].LatestRunID == "" {
		t.Fatal("listed.Items[0].LatestRunID = empty, want queued run id")
	}
}

func TestOperatorAPIRunCancelRoute(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	queries, err := NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	requests := newTestMatchRequestService(t, general, commands, store)
	api, err := NewOperatorAPI(commands, queries, general, requests, DirectArtifactAccessIssuer{}, nil)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	created, err := commands.Submit(context.Background(), testSubmission("file://"+repoJoin(t, "testdata/ai/janken/janken-rock-ai")))
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	cancelReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/runs/"+created.Submission.RunID+"/cancel", nil)
	cancelResp := httptest.NewRecorder()
	handler.ServeHTTP(cancelResp, cancelReq)
	if cancelResp.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/runs/{run_id}/cancel status = %d, body = %s", cancelResp.Code, cancelResp.Body.String())
	}
	var canceled ResultListItem
	if err := json.Unmarshal(cancelResp.Body.Bytes(), &canceled); err != nil {
		t.Fatalf("json.Unmarshal(canceled) error = %v", err)
	}
	if canceled.LifecycleState != StateCanceled {
		t.Fatalf("canceled.LifecycleState = %q, want %q", canceled.LifecycleState, StateCanceled)
	}
}

func TestOperatorAPIRankingReadRoute(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	rankingStore, err := NewLocalRankingSnapshotStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalRankingSnapshotStore() error = %v", err)
	}
	rankings, err := NewRankingService(rankingStore, nil)
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}
	scope := RankingScope{
		GameID:         "echo-count",
		GameVersion:    "2.0.0",
		RulesetVersion: "phase2-simultaneous-2turn",
	}
	snapshot := RankingSnapshot{
		Version:          rankingSnapshotVersion,
		Scope:            scope,
		AppliedRunIDs:    []string{"run-1"},
		AppliedMatchIDs:  []string{"match-1"},
		LastAppliedRunID: "run-1",
		CompletedMatches: 1,
		Entries: []RankingEntry{
			{
				BotID:         "bot-echo",
				BotName:       "echo",
				LastPlayerID:  "p1",
				MatchesPlayed: 1,
				FirstPlaces:   1,
				LastRunID:     "run-1",
				LastMatchID:   "match-1",
			},
		},
	}
	if _, err := rankingStore.Put(context.Background(), snapshot); err != nil {
		t.Fatalf("rankingStore.Put() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), DirectArtifactAccessIssuer{}, nil, rankings)
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/rankings?game_id=echo-count&game_version=2.0.0&ruleset_version=phase2-simultaneous-2turn", nil)
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/rankings status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var stored StoredRankingSnapshot
	if err := json.Unmarshal(resp.Body.Bytes(), &stored); err != nil {
		t.Fatalf("json.Unmarshal(stored) error = %v", err)
	}
	if stored.Snapshot.LastAppliedRunID != "run-1" {
		t.Fatalf("stored.Snapshot.LastAppliedRunID = %q, want %q", stored.Snapshot.LastAppliedRunID, "run-1")
	}
}

func newTestGeneralSubmissionService(t *testing.T) *GeneralSubmissionService {
	t.Helper()

	general, err := NewGeneralSubmissionService(repoRoot(t), nil, nil, nil)
	if err != nil {
		t.Fatalf("NewGeneralSubmissionService() error = %v", err)
	}
	return general
}

func newTestMatchRequestService(t *testing.T, general *GeneralSubmissionService, commands *CommandService, queue QueueStore) *MatchRequestService {
	t.Helper()

	requests, err := NewMatchRequestService(general, commands, queue, nil)
	if err != nil {
		t.Fatalf("NewMatchRequestService() error = %v", err)
	}
	return requests
}

func TestStatusCodeForServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: ErrQueueRecordNotFound, want: http.StatusNotFound},
		{name: "bad request", err: fmt.Errorf("%w: %w", ErrBadRequest, errors.New("service: output_dir is required")), want: http.StatusBadRequest},
		{name: "conflict", err: fmt.Errorf("%w: %w", ErrConflict, errors.New("service: run_id already exists")), want: http.StatusConflict},
		{name: "internal prefixed error stays internal", err: errors.New("service: enqueue submission: connection reset"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCodeForServiceError(tt.err); got != tt.want {
				t.Fatalf("statusCodeForServiceError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
