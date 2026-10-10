package platformadmin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/notifications"
)

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if p == nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
		return
	}
	result, err := notifications.List(r.Context(), h.readPool(), r, p.UserID, notifications.AudienceAdmin)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) unreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if p == nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
		return
	}
	count, badge, chrome, err := notifications.UnreadCount(r.Context(), h.readPool(), p.UserID, notifications.AudienceAdmin)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"count":       count,
		"badge_label": badge,
		"chrome":      chrome,
	})
}

func (h *Handler) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if p == nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
		return
	}
	id := chi.URLParam(r, "id")
	err := notifications.MarkRead(r.Context(), h.DB, p.UserID, notifications.AudienceAdmin, id)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "NOTIF_MARK_FAILED")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": h.msg("NOTIF_MARK_READ_DONE"),
	})
}

func (h *Handler) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if p == nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
		return
	}
	if err := notifications.MarkAllRead(r.Context(), h.DB, p.UserID, notifications.AudienceAdmin); err != nil {
		h.writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": h.msg("NOTIF_MARK_ALL_READ_DONE"),
	})
}
