package httpx

import (
	"net/http"
	"strconv"
)

// PathID parses the "{id}" path segment. On failure it writes a 400 response and returns false.
func PathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		WriteError(w, http.StatusBadRequest, "invalid_id", "The id in the URL is invalid.")
		return 0, false
	}
	return id, true
}
