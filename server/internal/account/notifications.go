package account

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/notifications"
)

// ListNotifications GET /api/v1/me/notifications
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	result, err := notifications.List(r.Context(), h.readPool(), r, userID, notifications.AudienceUser)
	if err != nil {
		code := err.Error()
		if code == "DATABASE_UNAVAILABLE" {
			writeJSONErr(w, r.Context(), h.readPool(), http.StatusServiceUnavailable, code)
			return
		}
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusBadRequest, code)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// UnreadNotificationCount GET /api/v1/me/notifications/unread-count
func (h *Handler) UnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	count, badge, chrome, err := notifications.UnreadCount(r.Context(), h.readPool(), userID, notifications.AudienceUser)
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count":       count,
		"badge_label": badge,
		"chrome":      chrome,
	})
}

// MarkNotificationRead POST /api/v1/me/notifications/{id}/read
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	id := chi.URLParam(r, "id")
	err := notifications.MarkRead(r.Context(), h.DB, userID, notifications.AudienceUser, id)
	if err == pgx.ErrNoRows {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusNotFound, "NOTIF_MARK_FAILED")
		return
	}
	if err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	ch := siteMsgMap(r.Context(), h.readPool(), []string{"NOTIF_MARK_READ_DONE"})
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": siteMsg(ch, "NOTIF_MARK_READ_DONE"),
	})
}

// MarkAllNotificationsRead POST /api/v1/me/notifications/read-all
func (h *Handler) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if err := notifications.MarkAllRead(r.Context(), h.DB, userID, notifications.AudienceUser); err != nil {
		writeJSONErr(w, r.Context(), h.readPool(), http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	ch := siteMsgMap(r.Context(), h.readPool(), []string{"NOTIF_MARK_ALL_READ_DONE"})
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": siteMsg(ch, "NOTIF_MARK_ALL_READ_DONE"),
	})
}
