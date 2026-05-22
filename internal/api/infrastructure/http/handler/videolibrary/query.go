package videolibrary

import (
	"net/http"
	"strconv"
	"strings"

	apiservice "learning-video-recommendation-system/internal/api/application/service"
)

func parseLimit(r *http.Request) (int, error) {
	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if rawLimit == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return 0, apiservice.InvalidRequestError("limit must be an integer")
	}
	if limit < 1 || limit > 100 {
		return 0, apiservice.InvalidRequestError("limit must be between 1 and 100")
	}
	return limit, nil
}

func parseCursor(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("cursor"))
}
