package service

import (
	"net/http"
	"strings"
)

// PublicAPI exposes the versioned anonymous spectator HTTP family.
type PublicAPI struct {
	queries *PublicQueryService
}

// NewPublicAPI constructs the public HTTP adapter.
func NewPublicAPI(queries *PublicQueryService) (*PublicAPI, error) {
	if queries == nil {
		return nil, errPublicAPI("queries are required")
	}
	return &PublicAPI{queries: queries}, nil
}

type errPublicAPI string

func (e errPublicAPI) Error() string { return "service: public API " + string(e) }

// Handler builds only anonymous, GET-only public routes.
func (a *PublicAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1-alpha/public/matches", a.handleList)
	mux.HandleFunc("GET /api/v1-alpha/public/matches/{match_id}", a.handleGet)
	mux.HandleFunc("GET /api/v1-alpha/public/matches/{match_id}/state", a.handleState)
	mux.HandleFunc("GET /api/v1-alpha/public/matches/{match_id}/replay", a.handleReplay)
	return withPublicCORS(mux)
}

func (a *PublicAPI) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := a.queries.List(r.Context())
	if err != nil {
		writePublicUnavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *PublicAPI) handleGet(w http.ResponseWriter, r *http.Request) {
	detail, found, err := a.queries.Get(r.Context(), r.PathValue("match_id"))
	if err != nil || !found {
		writePublicNotFound(w)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (a *PublicAPI) handleState(w http.ResponseWriter, r *http.Request) {
	response, found, err := a.queries.State(r.Context(), r.PathValue("match_id"))
	if err != nil || !found {
		writePublicNotFound(w)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *PublicAPI) handleReplay(w http.ResponseWriter, r *http.Request) {
	response, found, err := a.queries.Replay(r.Context(), r.PathValue("match_id"))
	if err != nil || !found {
		writePublicNotFound(w)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func withPublicCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Vary", "Origin")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writePublicNotFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "public match not found"})
}

func writePublicUnavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "public service unavailable"})
}

func cleanPublicMatchID(matchID string) string {
	return strings.TrimSpace(matchID)
}
