package emaildenylist

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// PublicCheckHandler is GET /api/v1/public/email-policy?email=
// Used by the auth callback before accepting a new OAuth session.
func PublicCheckHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(r.URL.Query().Get("email"))
		w.Header().Set("Content-Type", "application/json")
		if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"allowed": false,
				"code":    "AUTH_EMAIL_DOMAIN_DENIED",
				"message": subscriptions.MessageForCode("AUTH_EMAIL_DOMAIN_DENIED"),
			})
			return
		}
		denied, err := IsDenied(r.Context(), db, email)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"allowed": false,
				"code":    "AUTH_EMAIL_DOMAIN_DENIED",
				"message": subscriptions.MessageForCode("AUTH_EMAIL_DOMAIN_DENIED"),
			})
			return
		}
		if denied {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"allowed": false,
				"code":    "AUTH_EMAIL_DOMAIN_DENIED",
				"message": subscriptions.MessageForCode("AUTH_EMAIL_DOMAIN_DENIED"),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"allowed": true,
		})
	}
}
