package service

import (
	"fmt"
	"net/http"
	"strconv"
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
	options, err := decodePublicMatchListOptions(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid public match list query"})
		return
	}
	response, err := a.queries.List(r.Context(), options)
	if err != nil {
		writePublicUnavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func decodePublicMatchListOptions(r *http.Request) (PublicMatchListOptions, error) {
	options := defaultPublicMatchListOptions()
	values := r.URL.Query()
	for key, value := range values {
		if len(value) != 1 {
			return PublicMatchListOptions{}, fmt.Errorf("query %q must occur once", key)
		}
		switch key {
		case "game_id":
			options.GameID = value[0]
		case "ruleset_version":
			options.RulesetVersion = value[0]
		case "game_version_major":
			major, err := parsePositivePublicQueryInt(value[0])
			if err != nil {
				return PublicMatchListOptions{}, err
			}
			options.GameVersionMajor = major
		case "page":
			page, err := parsePositivePublicQueryInt(value[0])
			if err != nil {
				return PublicMatchListOptions{}, err
			}
			options.Page = page
		case "limit":
			limit, err := parsePositivePublicQueryInt(value[0])
			if err != nil || limit > 100 {
				return PublicMatchListOptions{}, fmt.Errorf("limit must be 1..100")
			}
			options.Limit = limit
		case "sort":
			if value[0] != "completed_at" {
				return PublicMatchListOptions{}, fmt.Errorf("unsupported sort")
			}
		case "sort_order":
			if value[0] != "asc" && value[0] != "desc" {
				return PublicMatchListOptions{}, fmt.Errorf("unsupported sort order")
			}
			options.SortOrder = value[0]
		default:
			return PublicMatchListOptions{}, fmt.Errorf("unsupported query %q", key)
		}
	}
	return options, nil
}

func parsePositivePublicQueryInt(raw string) (int, error) {
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("must be a positive integer")
	}
	return int(value), nil
}

func (a *PublicAPI) handleGet(w http.ResponseWriter, r *http.Request) {
	detail, found, err := a.queries.Get(r.Context(), r.PathValue("match_id"))
	if err != nil {
		writePublicUnavailable(w)
		return
	}
	if !found {
		writePublicNotFound(w)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (a *PublicAPI) handleState(w http.ResponseWriter, r *http.Request) {
	response, found, err := a.queries.State(r.Context(), r.PathValue("match_id"))
	if err != nil {
		writePublicUnavailable(w)
		return
	}
	if !found {
		writePublicNotFound(w)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *PublicAPI) handleReplay(w http.ResponseWriter, r *http.Request) {
	response, found, err := a.queries.Replay(r.Context(), r.PathValue("match_id"))
	if err != nil {
		writePublicUnavailable(w)
		return
	}
	if !found {
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
