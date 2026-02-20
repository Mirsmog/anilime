package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/example/anime-platform/internal/platform/api"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

// decodeJSON reads up to maxRequestBodyBytes from r.Body and decodes JSON into dst.
// On failure it writes a 400 response and returns false.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request, rid string, dst *T) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)).Decode(dst); err != nil {
		api.BadRequest(w, "INVALID_JSON", "Invalid JSON", rid, nil)
		return false
	}
	return true
}
