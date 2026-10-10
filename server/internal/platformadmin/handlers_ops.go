package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/authsettings"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/emaildenylist"
	"github.com/usetrim/trim/server/internal/events"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/notifications"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

func (h *Handler) listEnterpriseInquiries(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	whereParts := []string{}
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		whereParts = append(whereParts, fmt.Sprintf(`(
			lower(ei.email) like $%d
			or lower(coalesce(ei.company_name, '')) like $%d
			or lower(ei.status) like $%d
			or lower(coalesce(ei.message, '')) like $%d
			or ei.id::text like $%d
			or lower(coalesce(p.full_name, '')) like $%d
		)`, argN, argN, argN, argN, argN, argN))
		args = append(args, like)
		argN++
	}
	if statusFilter != "" {
		switch statusFilter {
		case "new", "contacted", "offered", "closed", "activated":
			whereParts = append(whereParts, fmt.Sprintf(`ei.status = $%d`, argN))
			args = append(args, statusFilter)
			argN++
		default:
			h.writeErr(w, http.StatusBadRequest, "ADMIN_STATUS_INVALID")
			return
		}
	}
	where := ""
	if len(whereParts) > 0 {
		where = " where " + strings.Join(whereParts, " and ")
	}
	var total int
	if err := db.QueryRow(r.Context(), `
		select count(*)
		from public.enterprise_inquiries ei
		left join public.profiles p on p.id = ei.user_id
	`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select ei.id::text, ei.user_id::text, ei.email, coalesce(ei.company_name, ''), coalesce(ei.estimated_seats, 0),
		       ei.message, ei.status, coalesce(ei.contract_notes, ''), ei.offered_seat_quantity,
		       ei.created_at::text, ei.updated_at::text, coalesce(p.full_name, '')
		from public.enterprise_inquiries ei
		left join public.profiles p on p.id = ei.user_id
		%s
		order by ei.created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, uid, email, company, message, status, notes, created, updated, fullName string
		var seats int
		var offered *int
		if err := rows.Scan(&id, &uid, &email, &company, &seats, &message, &status, &notes, &offered, &created, &updated, &fullName); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		row := map[string]any{
			"id": id, "user_id": uid, "email": email, "company_name": company,
			"estimated_seats": seats, "message": message, "status": status,
			"status_label":   h.enterpriseInquiryStatusLabel(status),
			"contract_notes": notes, "created_at": created, "updated_at": updated,
			"full_name": fullName,
		}
		if offered != nil {
			row["offered_seat_quantity"] = *offered
		}
		items = append(items, row)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": pagination.BuildMeta(params, total, skipCap)})
}

type patchInquiryBody struct {
	Status              *string `json:"status"`
	ContractNotes       *string `json:"contract_notes"`
	OfferedSeatQuantity *int    `json:"offered_seat_quantity"`
}

func (h *Handler) patchEnterpriseInquiry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body patchInquiryBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.Status == nil && body.ContractNotes == nil && body.OfferedSeatQuantity == nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}

	var currentStatus string
	err := h.DB.QueryRow(r.Context(), `
		select status from public.enterprise_inquiries where id = $1::uuid
	`, id).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	currentStatus = strings.ToLower(strings.TrimSpace(currentStatus))

	// Paid inquiries are immutable. Offered terms stay locked until sales
	// moves the row back to contacted (revise) or closed (withdraw).
	if currentStatus == "activated" {
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_ACTIVATED")
		return
	}
	if currentStatus == "offered" {
		if body.OfferedSeatQuantity != nil || body.ContractNotes != nil {
			h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_OFFERED")
			return
		}
		if body.Status == nil {
			h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_OFFERED")
			return
		}
		next := strings.ToLower(strings.TrimSpace(*body.Status))
		if next != "contacted" && next != "closed" {
			h.writeErr(w, http.StatusBadRequest, "ADMIN_STATUS_INVALID")
			return
		}
		*body.Status = next
	} else if body.Status != nil {
		status := strings.TrimSpace(*body.Status)
		if status != "new" && status != "contacted" && status != "closed" {
			h.writeErr(w, http.StatusBadRequest, "ADMIN_STATUS_INVALID")
			return
		}
		*body.Status = status
	}

	tag, err := h.DB.Exec(r.Context(), `
		update public.enterprise_inquiries set
		  status = coalesce($2, status),
		  contract_notes = coalesce($3, contract_notes),
		  offered_seat_quantity = coalesce($4, offered_seat_quantity),
		  updated_at = now()
		where id = $1::uuid
		  and status <> 'activated'
	`, id, body.Status, body.ContractNotes, body.OfferedSeatQuantity)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_ACTIVATED")
		return
	}
	h.audit(r.Context(), r, "enterprise.inquiry_update", "enterprise_inquiry", id, nil, body, "", false)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_ENTERPRISE_INQUIRY_UPDATED")})
}

// postEnterpriseInquiryActivate is the legacy path; it now sends a Paddle offer
// (status=offered + customer notify). Entitlements apply only after Paddle webhook.
func (h *Handler) postEnterpriseInquiryActivate(w http.ResponseWriter, r *http.Request) {
	h.postEnterpriseInquiryOffer(w, r)
}

// postEnterpriseInquiryOffer marks an inquiry offered for Paddle checkout.
// Does not grant plan_tier / quotas - payment webhook fulfills entitlements.
func (h *Handler) postEnterpriseInquiryOffer(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_ENTERPRISE_OFFER_FAILED")
		return
	}

	var userID string
	var offered *int
	var status string
	err := h.DB.QueryRow(r.Context(), `
		select coalesce(user_id::text, ''), offered_seat_quantity, status
		from public.enterprise_inquiries
		where id = $1::uuid
	`, id).Scan(&userID, &offered, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_ENTERPRISE_OFFER_FAILED")
		return
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_ENTERPRISE_USER_MISSING")
		return
	}
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	switch normalizedStatus {
	case "activated":
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_ACTIVATED")
		return
	case "offered":
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_OFFERED")
		return
	case "closed":
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_INQUIRY_CLOSED")
		return
	}
	if offered == nil || *offered < 1 {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_ENTERPRISE_SEATS_REQUIRED")
		return
	}
	seats := *offered

	var planID string
	var paddleMonthly, paddleYearly, paddleTopup *string
	err = h.DB.QueryRow(r.Context(), `
		select id, paddle_price_id_monthly, paddle_price_id_yearly, paddle_price_id_topup
		from public.plan_catalog
		where is_active = true and plan_kind = 'enterprise'
		order by plan_rank desc nulls last, sort_order asc nulls last
		limit 1
	`).Scan(&planID, &paddleMonthly, &paddleYearly, &paddleTopup)
	if err != nil || strings.TrimSpace(planID) == "" {
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_PLAN_MISSING")
		return
	}
	hasPrice := (paddleMonthly != nil && strings.TrimSpace(*paddleMonthly) != "") ||
		(paddleYearly != nil && strings.TrimSpace(*paddleYearly) != "")
	if !hasPrice {
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_PRICE_MISSING")
		return
	}

	note := fmt.Sprintf("offered seats=%d plan=%s (awaiting Paddle checkout)", seats, planID)
	tag, err := h.DB.Exec(r.Context(), `
		update public.enterprise_inquiries
		set status = 'offered',
		    contract_notes = case
		      when coalesce(contract_notes, '') = '' then $2
		      else contract_notes || E'\n' || $2
		    end,
		    updated_at = now()
		where id = $1::uuid
		  and status in ('new', 'contacted')
	`, id, note)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_ENTERPRISE_OFFER_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusConflict, "ADMIN_ENTERPRISE_ALREADY_OFFERED")
		return
	}

	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "enterprise.inquiry_offer", "enterprise_inquiry", id, map[string]any{
		"status": status,
	}, map[string]any{
		"status":                "offered",
		"user_id":               userID,
		"plan_tier":             planID,
		"offered_seat_quantity": seats,
	}, note, step)

	_ = notifications.Insert(r.Context(), h.DB, notifications.InsertOpts{
		RecipientID:  userID,
		Audience:     notifications.AudienceUser,
		KindCode:     notifications.KindUserEnterpriseOfferReady,
		BodyArgs:     []string{fmt.Sprintf("%d", seats)},
		DedupeKey:    "enterprise_offer:" + id + ":" + fmt.Sprintf("%d", seats),
		HrefEntityID: id,
	})

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":                    id,
		"status":                "offered",
		"status_label":          h.enterpriseInquiryStatusLabel("offered"),
		"user_id":               userID,
		"plan_tier":             planID,
		"offered_seat_quantity": seats,
		"message":               h.msg("ADMIN_ENTERPRISE_OFFERED"),
	})
}

func (h *Handler) listEmailDenylist(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(domain) like $%d
			or lower(coalesce(reason, '')) like $%d
		)`, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.email_domain_denylist`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select domain, reason, created_at::text from public.email_domain_denylist
		%s
		order by domain asc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var domain, reason, created string
		if rows.Scan(&domain, &reason, &created) == nil {
			items = append(items, map[string]string{"domain": domain, "reason": reason, "created_at": created})
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

type denyEmailBody struct {
	Domain string `json:"domain"`
	Reason string `json:"reason"`
}

func (h *Handler) postEmailDenylist(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body denyEmailBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	domain, ok := emaildenylist.DomainOf("x@" + strings.TrimPrefix(strings.TrimSpace(body.Domain), "@"))
	if !ok {
		domain = strings.ToLower(strings.TrimSpace(body.Domain))
	}
	if domain == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	_, err := h.DB.Exec(r.Context(), `
		insert into public.email_domain_denylist (domain, reason) values ($1, $2)
		on conflict (domain) do update set reason = excluded.reason
	`, domain, reason)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "denylist.email_add", "email_domain", domain, nil, body, reason, step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_DENYLIST_EMAIL_ADDED")})
}

func (h *Handler) deleteEmailDenylist(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	domain := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "domain")))
	tag, err := h.DB.Exec(r.Context(), `delete from public.email_domain_denylist where domain = $1`, domain)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "denylist.email_remove", "email_domain", domain, nil, nil, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_DENYLIST_EMAIL_REMOVED")})
}

func (h *Handler) listIPDenylist(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(cidr::text) like $%d
			or lower(coalesce(reason, '')) like $%d
		)`, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.ip_denylist`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select cidr::text, reason, created_at::text from public.ip_denylist
		%s
		order by cidr
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var cidr, reason, created string
		if rows.Scan(&cidr, &reason, &created) == nil {
			items = append(items, map[string]string{"cidr": cidr, "reason": reason, "created_at": created})
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

type denyIPBody struct {
	CIDR   string `json:"cidr"`
	Reason string `json:"reason"`
}

func (h *Handler) postIPDenylist(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body denyIPBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	cidr := strings.TrimSpace(body.CIDR)
	reason := strings.TrimSpace(body.Reason)
	if cidr == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	_, err := h.DB.Exec(r.Context(), `
		insert into public.ip_denylist (cidr, reason, created_by) values ($1::cidr, $2, nullif($3, '')::uuid)
		on conflict (cidr) do update set reason = excluded.reason
	`, cidr, reason, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "denylist.ip_add", "ip_cidr", cidr, nil, body, reason, step)
	middleware.FlushDenylistCaches(h.Redis)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_DENYLIST_IP_ADDED")})
}

func (h *Handler) deleteIPDenylist(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	cidr := strings.TrimSpace(chi.URLParam(r, "cidr"))
	tag, err := h.DB.Exec(r.Context(), `delete from public.ip_denylist where cidr = $1::cidr`, cidr)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "denylist.ip_remove", "ip_cidr", cidr, nil, nil, "", step)
	middleware.FlushDenylistCaches(h.Redis)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_DENYLIST_IP_REMOVED")})
}

func (h *Handler) listASNDenylist(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			asn::text like $%d
			or lower(coalesce(reason, '')) like $%d
		)`, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.asn_denylist`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select asn, reason, created_at::text from public.asn_denylist
		%s
		order by asn
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var asn int64
		var reason, created string
		if rows.Scan(&asn, &reason, &created) == nil {
			items = append(items, map[string]any{"asn": asn, "reason": reason, "created_at": created})
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

type denyASNBody struct {
	ASN    int64  `json:"asn"`
	Reason string `json:"reason"`
}

func (h *Handler) postASNDenylist(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body denyASNBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.ASN < 1 {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	_, err := h.DB.Exec(r.Context(), `
		insert into public.asn_denylist (asn, reason, created_by) values ($1, $2, nullif($3, '')::uuid)
		on conflict (asn) do update set reason = excluded.reason
	`, body.ASN, reason, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "denylist.asn_add", "asn", fmt.Sprintf("%d", body.ASN), nil, body, reason, step)
	middleware.FlushDenylistCaches(h.Redis)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_DENYLIST_ASN_ADDED")})
}

func (h *Handler) deleteASNDenylist(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	asnStr := chi.URLParam(r, "asn")
	tag, err := h.DB.Exec(r.Context(), `delete from public.asn_denylist where asn = $1`, asnStr)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "denylist.asn_remove", "asn", asnStr, nil, nil, "", step)
	middleware.FlushDenylistCaches(h.Redis)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_DENYLIST_ASN_REMOVED")})
}

func (h *Handler) listChromeMessages(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	where := ""
	args := []any{}
	if q != "" {
		where = `where code ilike $1 or body ilike $1`
		args = append(args, "%"+q+"%")
	}
	var total int
	countSQL := `select count(*) from public.site_messages ` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := `select code, body, updated_at::text from public.site_messages ` + where + ` order by code asc offset $` + fmt.Sprint(len(args)+1) + ` limit $` + fmt.Sprint(len(args)+2)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var code, body, updated string
		if rows.Scan(&code, &body, &updated) == nil {
			items = append(items, map[string]string{"code": code, "body": body, "updated_at": updated})
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": pagination.BuildMeta(params, total, skipCap)})
}

// getChromeUIMap returns every operator chrome row from site_messages (ADMIN_* plus
// shared dialog codes and SITE_HTML_LANG). Full map, no client invent of page size.
func (h *Handler) getChromeUIMap(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	rows, err := db.Query(r.Context(), `
		select code, body
		from public.site_messages
		where code like 'ADMIN_%'
		   or code like 'STATUS_%'
		   or code like 'BILLING_INTERVAL_%'
		   or code in ('DIALOG_CANCEL', 'DIALOG_CLOSE', 'SITE_HTML_LANG')
		order by code asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	messages := make(map[string]string)
	for rows.Next() {
		var code, body string
		if err := rows.Scan(&code, &body); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		messages[code] = body
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

type patchChromeBody struct {
	Body string `json:"body"`
}

func (h *Handler) patchChromeMessage(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	code := chi.URLParam(r, "code")
	var body patchChromeBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if strings.TrimSpace(body.Body) == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
		update public.site_messages set body = $2, updated_at = now() where code = $1
	`, code, body.Body)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	if err := subscriptions.ReloadAndBroadcastSiteMessages(r.Context(), h.DB, h.Redis); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "chrome.message_update", "site_message", code, nil, body, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_CHROME_MESSAGE_SAVED")})
}

func (h *Handler) getAuthSettings(w http.ResponseWriter, r *http.Request) {
	payload, status, code := h.authSettingsPayload(r.Context())
	if code != "" {
		h.writeErr(w, status, code)
		return
	}
	h.writeJSON(w, http.StatusOK, payload)
}

// authSettingsPayload builds the auth settings body. Returns a non-empty error code
// (with its status) instead of a payload when the provider catalog cannot be served.
func (h *Handler) authSettingsPayload(ctx context.Context) (map[string]any, int, string) {
	db := h.readPool()
	providers, err := authsettings.ListAllowedProviders(ctx, db)
	if err != nil {
		return nil, http.StatusInternalServerError, "AUTH_PROVIDERS_UNAVAILABLE"
	}
	rows, err := db.Query(ctx, `
		select code, body from public.site_messages
		where code like 'AUTH_PROVIDER_DISPLAY_%'
		order by code
	`)
	if err != nil {
		return nil, http.StatusInternalServerError, "DATABASE_UNAVAILABLE"
	}
	defer rows.Close()
	catalog := make([]map[string]string, 0)
	for rows.Next() {
		var code, body string
		if err := rows.Scan(&code, &body); err != nil {
			return nil, http.StatusInternalServerError, "DATABASE_UNAVAILABLE"
		}
		id := strings.ToLower(strings.TrimPrefix(code, "AUTH_PROVIDER_DISPLAY_"))
		if id == "" || body == "" {
			continue
		}
		catalog = append(catalog, map[string]string{"id": id, "display_name": body})
	}
	if len(catalog) == 0 {
		return nil, http.StatusServiceUnavailable, "AUTH_PROVIDERS_UNAVAILABLE"
	}
	return map[string]any{
		"allowed_providers": providers,
		"catalog":           catalog,
	}, http.StatusOK, ""
}

type patchAuthBody struct {
	AllowedProviders []string `json:"allowed_providers"`
}

func (h *Handler) patchAuthSettings(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body patchAuthBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	allowedIDs := map[string]bool{}
	rows, err := h.DB.Query(r.Context(), `
		select code from public.site_messages where code like 'AUTH_PROVIDER_DISPLAY_%'
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if rows.Scan(&code) == nil {
			id := strings.ToLower(strings.TrimPrefix(code, "AUTH_PROVIDER_DISPLAY_"))
			if id != "" {
				allowedIDs[id] = true
			}
		}
	}
	clean := make([]string, 0, len(body.AllowedProviders))
	seen := map[string]struct{}{}
	for _, raw := range body.AllowedProviders {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" || !allowedIDs[id] {
			h.writeErr(w, http.StatusBadRequest, "AUTH_PROVIDERS_UNAVAILABLE")
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if err := authsettings.SyncAllowedProviders(r.Context(), h.DB, clean); err != nil {
		h.writeErr(w, http.StatusBadRequest, "AUTH_PROVIDERS_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "auth.settings_update", "auth_settings", "default", nil, map[string]any{"allowed_providers": clean}, "", step)
	payload, status, errCode := h.authSettingsPayload(r.Context())
	if errCode != "" {
		h.writeErr(w, status, errCode)
		return
	}
	payload["message"] = h.msg("ADMIN_AUTH_SETTINGS_SAVED")
	h.writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) getObservabilityEventStats(w http.ResponseWriter, r *http.Request) {
	readDB := h.readPool()
	topN, err := billingsettings.ChartTopN(r.Context(), readDB)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	seriesDays, err := billingsettings.ChartSeriesDays(r.Context(), readDB)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var total, success24h, error24h int
	var tokensBefore, tokensAfter int64
	var cliNotice, dbMinCLI string
	_ = readDB.QueryRow(r.Context(), `
		select
			(select count(*) from public.trim_events),
			(select count(*) filter (where status = 'success')
			 from public.trim_events where created_at >= now() - interval '24 hours'),
			(select count(*) filter (where status = 'error')
			 from public.trim_events where created_at >= now() - interval '24 hours'),
			(select coalesce(sum(tokens_before), 0) from public.trim_events),
			(select coalesce(sum(tokens_after), 0) from public.trim_events),
			(select coalesce(body, '') from public.site_messages where code = 'CLI_FORCE_UPGRADE_NOTICE'),
			(select coalesce(min_cli_version, '') from public.admin_product_settings where id = 'default')
	`).Scan(&total, &success24h, &error24h, &tokensBefore, &tokensAfter, &cliNotice, &dbMinCLI)

	byMode := make([]map[string]any, 0)
	modeRows, err := readDB.Query(r.Context(), `
		select coalesce(nullif(btrim(mode), ''), ''), count(*)::bigint
		from public.trim_events
		group by 1
		order by 2 desc
		limit $1
	`, topN)
	if err == nil {
		defer modeRows.Close()
		for modeRows.Next() {
			var mode string
			var n int64
			if modeRows.Scan(&mode, &n) == nil && mode != "" {
				byMode = append(byMode, map[string]any{"mode": mode, "count": n})
			}
		}
	}

	byModel := make([]map[string]any, 0)
	modelRows, err := readDB.Query(r.Context(), `
		select coalesce(nullif(btrim(model), ''), ''), count(*)::bigint
		from public.trim_events
		where nullif(btrim(model), '') is not null
		group by 1
		order by 2 desc
		limit $1
	`, topN)
	if err == nil {
		defer modelRows.Close()
		for modelRows.Next() {
			var model string
			var n int64
			if modelRows.Scan(&model, &n) == nil && model != "" {
				byModel = append(byModel, map[string]any{"model": model, "count": n})
			}
		}
	}

	effMinCLI := strings.TrimSpace(dbMinCLI)

	groupBy := r.URL.Query().Get("group_by")
	heatmapScope := r.URL.Query().Get("heatmap_scope")
	cacheKey := fmt.Sprintf("chart:admin:v2:%s:%s:%d:%d", groupBy, heatmapScope, seriesDays, topN)
	var charts events.DashboardCharts
	if cached, ok := h.getCachedAdminCharts(r.Context(), cacheKey); ok {
		charts = cached
	} else {
		built, chartErr := events.BuildDashboardCharts(
			r.Context(), readDB, "",
			groupBy,
			heatmapScope,
			seriesDays, topN,
			events.LoadUsageChrome("ADMIN"),
			events.LoadHeatmapChrome("ADMIN"),
		)
		if chartErr != nil {
			h.writeErr(w, http.StatusInternalServerError, "EVENTS_AGG_SERIES_FAILED")
			return
		}
		charts = built
		h.setCachedAdminCharts(r.Context(), cacheKey, charts)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"total_events":  total,
		"tokens_before": tokensBefore,
		"tokens_after":  tokensAfter,
		"last_24h":      map[string]int{"success": success24h, "error": error24h},
		"by_mode":       byMode,
		"by_model":      byModel,
		"cli": map[string]any{
			"min_cli_version":      effMinCLI,
			"force_upgrade_notice": strings.TrimSpace(cliNotice),
		},
		"usage_series":                   charts.UsageSeries,
		"usage_days":                     charts.UsageDays,
		"usage_group_by_options":         charts.UsageGroupByOptions,
		"usage_group_by_selected":        charts.UsageGroupBySelected,
		"usage_title":                    charts.UsageTitle,
		"usage_subtitle":                 charts.UsageSubtitle,
		"usage_y_axis":                   charts.UsageYAxis,
		"usage_today_label":              charts.UsageTodayLabel,
		"usage_group_by_prefix":          charts.UsageGroupByPrefix,
		"usage_empty":                    charts.UsageEmpty,
		"usage_tooltip_breakdown":        charts.UsageTooltipBreakdown,
		"usage_tooltip_daily_total":      charts.UsageTooltipDailyTotal,
		"usage_tooltip_cumulative_total": charts.UsageTooltipCumulativeTotal,
		"usage_tooltip_share_fmt":        charts.UsageTooltipShareFmt,
		"today_day":                      charts.TodayDay,
		"loc_heatmap":                    charts.LocHeatmap,
		"loc_heatmap_total":              charts.LocHeatmapTotal,
		"loc_heatmap_scopes":             charts.LocHeatmapScopes,
		"loc_heatmap_scope_selected":     charts.LocHeatmapScopeSelected,
		"loc_heatmap_title":              charts.LocHeatmapTitle,
		"loc_heatmap_empty_fmt":          charts.LocHeatmapEmptyFmt,
		"loc_heatmap_value_fmt":          charts.LocHeatmapValueFmt,
		"loc_heatmap_weekday_labels":     charts.LocHeatmapWeekdayLabels,
		"loc_heatmap_stats":              charts.LocHeatmapStats,
	})
}

func (h *Handler) listObservabilityWebhooks(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(event_id) like $%d
			or lower(event_type) like $%d
			or lower(coalesce(process_status, '')) like $%d
			or lower(coalesce(last_error, '')) like $%d
		)`, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	_ = db.QueryRow(r.Context(), `select count(*) from public.paddle_webhook_events`+where, args...).Scan(&total)
	listSQL := fmt.Sprintf(`
		select event_id, event_type, processed_at::text,
		       coalesce(process_status, ''), coalesce(last_error, '')
		from public.paddle_webhook_events
		%s
		order by processed_at desc nulls last
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var id, typ string
		var processed *string
		var status, lastErr string
		if rows.Scan(&id, &typ, &processed, &status, &lastErr) == nil {
			item := map[string]string{"event_id": id, "event_type": typ}
			if processed != nil {
				item["processed_at"] = *processed
			}
			if status != "" {
				item["process_status"] = status
				if lbl := h.webhookProcessStatusLabel(status); lbl != "" {
					item["process_status_label"] = lbl
				}
			}
			if lastErr != "" {
				item["last_error"] = lastErr
			}
			items = append(items, item)
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": pagination.BuildMeta(params, total, skipCap)})
}

func (h *Handler) getObservabilityWebhook(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	var typ, status, lastErr string
	var processed *string
	var payload []byte
	err := h.readPool().QueryRow(r.Context(), `
		select event_type, processed_at::text, coalesce(process_status, ''),
		       coalesce(last_error, ''), payload
		from public.paddle_webhook_events where event_id = $1
	`, eventID).Scan(&typ, &processed, &status, &lastErr, &payload)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_WEBHOOK_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	out := map[string]any{
		"event_id": eventID, "event_type": typ,
		"process_status": status, "last_error": lastErr,
	}
	if lbl := h.webhookProcessStatusLabel(status); lbl != "" {
		out["process_status_label"] = lbl
	}
	if processed != nil {
		out["processed_at"] = *processed
	}
	if len(payload) > 0 {
		var raw any
		if json.Unmarshal(payload, &raw) == nil {
			out["payload"] = raw
		} else {
			out["payload_raw"] = string(payload)
		}
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getDistributionStats(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	rng := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("range")))
	if rng == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_RANGE_REQUIRED")
		return
	}
	var interval string
	switch rng {
	case "day":
		interval = "1 day"
	case "month":
		interval = "30 days"
	case "year":
		interval = "365 days"
	default:
		h.writeErr(w, http.StatusBadRequest, "ADMIN_RANGE_INVALID")
		return
	}
	var total int
	db := h.readPool()
	if err := db.QueryRow(r.Context(), `
		select count(*) from public.distribution_daily_stats
		where day >= (current_date - $1::interval)
	`, interval).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	rows, err := db.Query(r.Context(), `
		select day::text, source, metric, country, coalesce(path, ''), value
		from public.distribution_daily_stats
		where day >= (current_date - $1::interval)
		order by day desc, value desc, path asc, country asc
		offset $2 limit $3
	`, interval, params.Skip, params.Limit)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var day, source, metric, country, path string
		var value int64
		if rows.Scan(&day, &source, &metric, &country, &path, &value) == nil {
			if strings.TrimSpace(day) == "" {
				continue
			}
			item := map[string]any{
				"id":      day + "|" + source + "|" + metric + "|" + country + "|" + path,
				"day":     day,
				"source":  source,
				"metric":  metric,
				"country": country,
				"path":    path,
				"value":   value,
			}
			if lbl := h.distSourceLabel(source); lbl != "" {
				item["source_label"] = lbl
			}
			if lbl := h.distMetricLabel(metric); lbl != "" {
				item["metric_label"] = lbl
			}
			items = append(items, item)
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"range": rng,
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

func (h *Handler) postDistributionSync(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	ctx := r.Context()
	day := time.Now().UTC().Format("2006-01-02")

	// Rebuild install aggregates (by country + installer path) for the year window
	// admin can query. Full replace so Sync after schema/path changes is correct.
	if _, err := h.DB.Exec(ctx, `
		delete from public.distribution_daily_stats where source = 'install'
	`); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_DISTRIBUTION_SYNC_FAILED")
		return
	}
	_, err := h.DB.Exec(ctx, `
		insert into public.distribution_daily_stats (day, source, metric, country, path, value)
		select
			h.hit_at::date,
			'install',
			'hits',
			coalesce(h.country, ''),
			left(coalesce(nullif(trim(h.path), ''), '/install.sh'), 256),
			count(*)::bigint
		from public.install_hits h
		where h.hit_at >= (current_date - interval '365 days')
		group by
			h.hit_at::date,
			coalesce(h.country, ''),
			left(coalesce(nullif(trim(h.path), ''), '/install.sh'), 256)
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_DISTRIBUTION_SYNC_FAILED")
		return
	}

	githubConfigured := strings.TrimSpace(h.Config.GitHubToken) != "" && strings.TrimSpace(h.Config.GitHubRepo) != ""
	var ghErr string
	if githubConfigured {
		if err := h.syncGitHubDistribution(ctx); err != nil {
			ghErr = h.distChannelNotice(err)
			h.recordDistSyncError(ctx, "github", ghErr)
		}
	} else {
		ghErr = h.msg("ADMIN_GITHUB_NOT_CONFIGURED")
	}

	notices := make([]string, 0, 4)
	runChannel := func(source, repo, notConfiguredCode string) {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			notices = append(notices, h.msg(notConfiguredCode))
			return
		}
		if strings.TrimSpace(h.Config.GitHubToken) == "" {
			n := h.msg("ADMIN_GITHUB_NOT_CONFIGURED")
			notices = append(notices, n)
			h.recordDistSyncError(ctx, source, n)
			return
		}
		if err := h.syncGitHubRepoTraffic(ctx, source, repo); err != nil {
			n := h.distChannelNotice(err)
			notices = append(notices, n)
			h.recordDistSyncError(ctx, source, n)
		}
	}
	runChannel("homebrew", h.Config.DistHomebrewTapRepo, "ADMIN_DIST_HOMEBREW_NOT_CONFIGURED")
	runChannel("scoop", h.Config.DistScoopBucketRepo, "ADMIN_DIST_SCOOP_NOT_CONFIGURED")
	runChannel("winget", h.Config.DistWingetForkRepo, "ADMIN_DIST_WINGET_NOT_CONFIGURED")

	var marketplaceNotice string
	extID := strings.TrimSpace(h.Config.DistVSCodeExtensionID)
	if extID == "" {
		marketplaceNotice = h.msg("ADMIN_DIST_MARKETPLACE_NOT_CONFIGURED")
		notices = append(notices, marketplaceNotice)
	} else if err := h.syncVSCodeMarketplace(ctx, extID); err != nil {
		marketplaceNotice = h.distChannelNotice(err)
		notices = append(notices, marketplaceNotice)
		h.recordDistSyncError(ctx, "marketplace", marketplaceNotice)
	}

	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "distribution.sync", "distribution", day, nil,
		map[string]any{
			"github_configured":      githubConfigured,
			"homebrew_configured":    strings.TrimSpace(h.Config.DistHomebrewTapRepo) != "",
			"scoop_configured":       strings.TrimSpace(h.Config.DistScoopBucketRepo) != "",
			"winget_configured":      strings.TrimSpace(h.Config.DistWingetForkRepo) != "",
			"marketplace_configured": extID != "",
		}, "", step)

	h.writeJSON(w, http.StatusOK, map[string]any{
		"install_hits_aggregated": true,
		"github_configured":       githubConfigured,
		"github_notice":           ghErr,
		"channel_notices":         notices,
		"pending_label":           h.msg("ADMIN_PENDING_SYNCING"),
		"message":                 h.msg("ADMIN_DISTRIBUTION_SYNCED"),
	})
}

// listAuditFilterOptions returns distinct action / actor / resource_type values from admin_audit_log.
func (h *Handler) listAuditFilterOptions(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	type opt struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	actions := make([]opt, 0)
	aRows, err := db.Query(r.Context(), `
		select distinct action
		from public.admin_audit_log
		where coalesce(trim(action), '') <> ''
		order by action asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	for aRows.Next() {
		var action string
		if err := aRows.Scan(&action); err != nil {
			aRows.Close()
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		action = strings.TrimSpace(action)
		if action == "" {
			continue
		}
		actions = append(actions, opt{Value: action, Label: action})
	}
	aRows.Close()

	actors := make([]opt, 0)
	actorRows, err := db.Query(r.Context(), `
		select distinct a.actor_user_id::text,
		       coalesce(nullif(trim(p.email), ''), a.actor_user_id::text) as label
		from public.admin_audit_log a
		left join public.profiles p on p.id = a.actor_user_id
		where a.actor_user_id is not null
		order by label asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	for actorRows.Next() {
		var id, label string
		if err := actorRows.Scan(&id, &label); err != nil {
			actorRows.Close()
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		id = strings.TrimSpace(id)
		label = strings.TrimSpace(label)
		if id == "" {
			continue
		}
		if label == "" {
			label = id
		}
		actors = append(actors, opt{Value: id, Label: label})
	}
	actorRows.Close()

	resources := make([]opt, 0)
	rRows, err := db.Query(r.Context(), `
		select distinct resource_type
		from public.admin_audit_log
		where coalesce(trim(resource_type), '') <> ''
		order by resource_type asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	for rRows.Next() {
		var rtype string
		if err := rRows.Scan(&rtype); err != nil {
			rRows.Close()
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		rtype = strings.TrimSpace(rtype)
		if rtype == "" {
			continue
		}
		resources = append(resources, opt{Value: rtype, Label: rtype})
	}
	rRows.Close()

	h.writeJSON(w, http.StatusOK, map[string]any{
		"actions":   actions,
		"actors":    actors,
		"resources": resources,
	})
}

func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	actor := strings.TrimSpace(r.URL.Query().Get("actor"))
	resource := strings.TrimSpace(r.URL.Query().Get("resource"))
	createdFrom := strings.TrimSpace(r.URL.Query().Get("created_from"))
	createdTo := strings.TrimSpace(r.URL.Query().Get("created_to"))

	where := `where 1=1`
	args := []any{}
	n := 1
	if action != "" {
		where += fmt.Sprintf(` and action = $%d`, n)
		args = append(args, action)
		n++
	}
	if actor != "" {
		where += fmt.Sprintf(` and actor_user_id::text = $%d`, n)
		args = append(args, actor)
		n++
	}
	if resource != "" {
		where += fmt.Sprintf(` and resource_type = $%d`, n)
		args = append(args, resource)
		n++
	}
	if createdFrom != "" {
		where += fmt.Sprintf(` and created_at::date >= $%d::date`, n)
		args = append(args, createdFrom)
		n++
	}
	if createdTo != "" {
		where += fmt.Sprintf(` and created_at::date <= $%d::date`, n)
		args = append(args, createdTo)
		n++
	}

	db := h.readPool()
	countSQL := `select count(*) from public.admin_audit_log ` + where
	var total int
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id, coalesce(actor_user_id::text, ''), action, resource_type, coalesce(resource_id, ''),
		       coalesce(reason, ''), step_up_used, created_at::text
		from public.admin_audit_log
		%s
		order by created_at desc
		offset $%d limit $%d
	`, where, n, n+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var actorID, act, rtype, rid, reason, created string
		var stepUp bool
		if rows.Scan(&id, &actorID, &act, &rtype, &rid, &reason, &stepUp, &created) == nil {
			items = append(items, map[string]any{
				"id": id, "actor_user_id": actorID, "action": act,
				"resource_type": rtype, "resource_id": rid, "reason": reason,
				"step_up_used": stepUp, "created_at": created,
			})
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": pagination.BuildMeta(params, total, skipCap)})
}

func (h *Handler) getComplianceRetention(w http.ResponseWriter, r *http.Request) {
	row, err := scanJSONSettings(r.Context(), h, "public.admin_retention_settings", "default")
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_RETENTION_SETTINGS_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	h.writeJSON(w, http.StatusOK, row)
}

func (h *Handler) patchComplianceRetention(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body struct {
		TrimEventsTTLDays              *int `json:"trim_events_ttl_days"`
		AuditLogTTLDays                *int `json:"audit_log_ttl_days"`
		AuditExportMaxRows             *int `json:"audit_export_max_rows"`
		AccessReviewAttestationsLimit  *int `json:"access_review_attestations_limit"`
		BreakGlassTTLMinutes           *int `json:"break_glass_ttl_minutes"`
		ForceLogoutTTLSec              *int `json:"force_logout_ttl_sec"`
		TrimEventsPartitionMonthsAhead *int `json:"trim_events_partition_months_ahead"`
		TrimEventsPartitionEnsureSec   *int `json:"trim_events_partition_ensure_sec"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.TrimEventsTTLDays == nil && body.AuditLogTTLDays == nil &&
		body.AuditExportMaxRows == nil && body.AccessReviewAttestationsLimit == nil &&
		body.BreakGlassTTLMinutes == nil && body.ForceLogoutTTLSec == nil &&
		body.TrimEventsPartitionMonthsAhead == nil && body.TrimEventsPartitionEnsureSec == nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.ForceLogoutTTLSec != nil && (*body.ForceLogoutTTLSec < 60 || *body.ForceLogoutTTLSec > 2592000) {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.TrimEventsPartitionMonthsAhead != nil &&
		(*body.TrimEventsPartitionMonthsAhead < 1 || *body.TrimEventsPartitionMonthsAhead > 36) {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.TrimEventsPartitionEnsureSec != nil &&
		(*body.TrimEventsPartitionEnsureSec < 3600 || *body.TrimEventsPartitionEnsureSec > 604800) {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	_, err := h.DB.Exec(r.Context(), `
		insert into public.admin_retention_settings (
		  id, trim_events_ttl_days, audit_log_ttl_days,
		  audit_export_max_rows, access_review_attestations_limit,
		  break_glass_ttl_minutes, force_logout_ttl_sec,
		  trim_events_partition_months_ahead, trim_events_partition_ensure_sec,
		  updated_by, updated_at
		)
		values ('default', $1, $2, $3, $4, $5, $6, $7, $8, nullif($9, '')::uuid, now())
		on conflict (id) do update set
		  trim_events_ttl_days = coalesce(excluded.trim_events_ttl_days, admin_retention_settings.trim_events_ttl_days),
		  audit_log_ttl_days = coalesce(excluded.audit_log_ttl_days, admin_retention_settings.audit_log_ttl_days),
		  audit_export_max_rows = coalesce(excluded.audit_export_max_rows, admin_retention_settings.audit_export_max_rows),
		  access_review_attestations_limit = coalesce(excluded.access_review_attestations_limit, admin_retention_settings.access_review_attestations_limit),
		  break_glass_ttl_minutes = coalesce(excluded.break_glass_ttl_minutes, admin_retention_settings.break_glass_ttl_minutes),
		  force_logout_ttl_sec = coalesce(excluded.force_logout_ttl_sec, admin_retention_settings.force_logout_ttl_sec),
		  trim_events_partition_months_ahead = coalesce(excluded.trim_events_partition_months_ahead, admin_retention_settings.trim_events_partition_months_ahead),
		  trim_events_partition_ensure_sec = coalesce(excluded.trim_events_partition_ensure_sec, admin_retention_settings.trim_events_partition_ensure_sec),
		  updated_by = excluded.updated_by,
		  updated_at = now()
	`, body.TrimEventsTTLDays, body.AuditLogTTLDays, body.AuditExportMaxRows, body.AccessReviewAttestationsLimit, body.BreakGlassTTLMinutes, body.ForceLogoutTTLSec, body.TrimEventsPartitionMonthsAhead, body.TrimEventsPartitionEnsureSec, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if body.TrimEventsPartitionMonthsAhead != nil {
		if err := h.EnsureTrimEventsPartitions(r.Context()); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "ADMIN_EVENTS_PARTITION_ENSURE_FAILED")
			return
		}
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "compliance.retention_update", "admin_retention_settings", "default", nil, body, "", step)
	row, err := scanJSONSettings(r.Context(), h, "public.admin_retention_settings", "default")
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_RETENTION_SETTINGS_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	row["message"] = h.msg("ADMIN_COMPLIANCE_RETENTION_SAVED")
	h.writeJSON(w, http.StatusOK, row)
}

func (h *Handler) postComplianceRetentionPurge(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var reasonBody struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &reasonBody); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	reason := strings.TrimSpace(reasonBody.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	after, errCode, err := h.runRetentionPurge(r.Context())
	if errCode != "" {
		status := http.StatusInternalServerError
		if errCode == "ADMIN_RETENTION_TTL_REQUIRED" {
			status = http.StatusConflict
		}
		if errCode == "ADMIN_EVENTS_PARTITION_AHEAD_MISSING" || errCode == "ADMIN_EVENTS_PARTITION_ENSURE_FAILED" {
			status = http.StatusServiceUnavailable
		}
		h.writeErr(w, status, errCode)
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "compliance.retention_purge", "admin_retention_settings", "default", nil, after, reason, step)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": h.msg("ADMIN_RETENTION_PURGE_DONE"),
		"result":  after,
	})
}

// RunRetentionPurge is used by admin UI and by CRON_SECRET internal route.
func (h *Handler) RunRetentionPurge(ctx context.Context) (map[string]any, error) {
	after, code, err := h.runRetentionPurge(ctx)
	if code != "" {
		return nil, fmt.Errorf("%s", code)
	}
	return after, err
}

func (h *Handler) runRetentionPurge(ctx context.Context) (map[string]any, string, error) {
	var eventsTTL, auditTTL, monthsAhead int
	err := h.DB.QueryRow(ctx, `
		select trim_events_ttl_days, audit_log_ttl_days, trim_events_partition_months_ahead
		from public.admin_retention_settings where id = 'default'
	`).Scan(&eventsTTL, &auditTTL, &monthsAhead)
	if err == pgx.ErrNoRows || eventsTTL <= 0 || auditTTL <= 0 {
		return nil, "ADMIN_RETENTION_TTL_REQUIRED", err
	}
	if err != nil {
		return nil, "DATABASE_UNAVAILABLE", err
	}
	if monthsAhead < 1 || monthsAhead > 36 {
		return nil, "ADMIN_EVENTS_PARTITION_AHEAD_MISSING", nil
	}
	monthsBack := (eventsTTL / 30) + 2
	if monthsBack < 1 {
		monthsBack = 1
	}
	if monthsBack > 120 {
		monthsBack = 120
	}
	var partitionsCreated int
	if err := h.DB.QueryRow(ctx, `
		select public.ensure_trim_events_month_partitions($1, $2)
	`, monthsAhead, monthsBack).Scan(&partitionsCreated); err != nil {
		return nil, "ADMIN_EVENTS_PARTITION_ENSURE_FAILED", err
	}
	var partitionsDropped int
	if err := h.DB.QueryRow(ctx, `
		select public.drop_trim_events_partitions_older_than($1)
	`, eventsTTL).Scan(&partitionsDropped); err != nil {
		return nil, "DATABASE_UNAVAILABLE", err
	}
	tagEvents, err := h.DB.Exec(ctx, `
		delete from public.trim_events
		where created_at < now() - make_interval(days => $1)
	`, eventsTTL)
	if err != nil {
		return nil, "DATABASE_UNAVAILABLE", err
	}
	tagOutbox, err := h.DB.Exec(ctx, `
		delete from public.trim_event_outbox
		where created_at < now() - make_interval(days => $1)
	`, eventsTTL)
	if err != nil {
		return nil, "DATABASE_UNAVAILABLE", err
	}
	tagAudit, err := h.DB.Exec(ctx, `
		delete from public.admin_audit_log
		where created_at < now() - make_interval(days => $1)
	`, auditTTL)
	if err != nil {
		return nil, "DATABASE_UNAVAILABLE", err
	}
	_, _ = h.DB.Exec(ctx, `
		update public.admin_break_glass
		set status = 'expired'
		where status = 'approved' and ends_at is not null and ends_at <= now()
	`)
	after := map[string]any{
		"trim_events_deleted":                tagEvents.RowsAffected(),
		"trim_events_partitions_dropped":     partitionsDropped,
		"trim_events_partitions_created":     partitionsCreated,
		"trim_outbox_deleted":                tagOutbox.RowsAffected(),
		"audit_log_deleted":                  tagAudit.RowsAffected(),
		"trim_events_ttl_days":               eventsTTL,
		"audit_log_ttl_days":                 auditTTL,
		"trim_events_partition_months_ahead": monthsAhead,
	}
	return after, "", nil
}

// EnsureTrimEventsPartitions pre-creates future UTC month partitions from DB settings.
// Call on API boot so inserts never hit a missing month (no DEFAULT partition).
func (h *Handler) EnsureTrimEventsPartitions(ctx context.Context) error {
	if h == nil || h.DB == nil {
		return fmt.Errorf("ADMIN_EVENTS_PARTITION_ENSURE_FAILED")
	}
	var monthsAhead int
	err := h.DB.QueryRow(ctx, `
		select trim_events_partition_months_ahead
		from public.admin_retention_settings where id = 'default'
	`).Scan(&monthsAhead)
	if err == pgx.ErrNoRows || monthsAhead < 1 || monthsAhead > 36 {
		return fmt.Errorf("ADMIN_EVENTS_PARTITION_AHEAD_MISSING")
	}
	if err != nil {
		return err
	}
	var created int
	if err := h.DB.QueryRow(ctx, `
		select public.ensure_trim_events_month_partitions($1, $2)
	`, monthsAhead, 1).Scan(&created); err != nil {
		return err
	}
	return nil
}

func (h *Handler) loadPartitionEnsureSec(ctx context.Context) (int, error) {
	if h == nil || h.DB == nil {
		return 0, fmt.Errorf("ADMIN_EVENTS_PARTITION_ENSURE_SEC_MISSING")
	}
	var sec int
	err := h.DB.QueryRow(ctx, `
		select trim_events_partition_ensure_sec
		from public.admin_retention_settings where id = 'default'
	`).Scan(&sec)
	if err == pgx.ErrNoRows || sec < 3600 || sec > 604800 {
		return 0, fmt.Errorf("ADMIN_EVENTS_PARTITION_ENSURE_SEC_MISSING")
	}
	if err != nil {
		return 0, err
	}
	return sec, nil
}

// PartitionEnsureSec exposes the DB-driven ensure interval for boot fail-closed checks.
func (h *Handler) PartitionEnsureSec(ctx context.Context) (int, error) {
	return h.loadPartitionEnsureSec(ctx)
}

// RunPartitionEnsureLoop re-ensures future month partitions on a DB-driven interval
// so a long-lived API process cannot miss a month boundary without retention cron.
func (h *Handler) RunPartitionEnsureLoop(ctx context.Context) {
	sec, err := h.loadPartitionEnsureSec(ctx)
	if err != nil {
		log.Printf("trim_events partition ensure loop: %v", err)
		return
	}
	t := time.NewTimer(time.Duration(sec) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := h.EnsureTrimEventsPartitions(ctx); err != nil {
				log.Printf("trim_events partition ensure: %v", err)
			}
			next, loadErr := h.loadPartitionEnsureSec(ctx)
			if loadErr != nil {
				log.Printf("trim_events partition ensure interval: %v", loadErr)
			} else {
				sec = next
			}
			t.Reset(time.Duration(sec) * time.Second)
		}
	}
}

func (h *Handler) listBreakGlass(w http.ResponseWriter, r *http.Request) {
	_, _ = h.DB.Exec(r.Context(), `
		update public.admin_break_glass
		set status = 'expired'
		where status = 'approved' and ends_at is not null and ends_at <= now()
	`)
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(reason) like $%d
			or lower(status) like $%d
			or lower(coalesce(elevates_permission, '')) like $%d
			or requester_id::text like $%d
			or coalesce(approver_id::text, '') like $%d
			or id::text like $%d
		)`, argN, argN, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.admin_break_glass`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id::text, requester_id::text, coalesce(approver_id::text, ''), reason, status,
		       coalesce(elevates_permission, ''), coalesce(starts_at::text, ''), coalesce(ends_at::text, ''),
		       created_at::text
		from public.admin_break_glass
		%s
		order by created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var id, req, appr, reason, status, perm, starts, ends, created string
		if err := rows.Scan(&id, &req, &appr, &reason, &status, &perm, &starts, &ends, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]string{
			"id": id, "requester_id": req, "approver_id": appr, "reason": reason,
			"status": status, "status_label": h.breakGlassStatusLabel(status),
			"elevates_permission": perm, "starts_at": starts,
			"ends_at": ends, "created_at": created,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

type breakGlassBody struct {
	Reason             string `json:"reason"`
	ElevatesPermission string `json:"elevates_permission"`
}

func (h *Handler) postBreakGlass(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body breakGlassBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	perm := strings.TrimSpace(body.ElevatesPermission)
	if perm == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	var ok bool
	err := h.DB.QueryRow(r.Context(), `
		select exists (
		  select 1 from public.platform_permission_catalog where code = $1
		)
	`, perm).Scan(&ok)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if !ok {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_BREAK_GLASS_PERM_INVALID")
		return
	}
	var id string
	err = h.DB.QueryRow(r.Context(), `
		insert into public.admin_break_glass (requester_id, reason, elevates_permission)
		values ($1::uuid, $2, $3)
		returning id::text
	`, p.UserID, reason, perm).Scan(&id)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "break_glass.request", "admin_break_glass", id, nil, body, reason, step)

	var requesterEmail string
	_ = h.DB.QueryRow(r.Context(), `select email from public.profiles where id = $1::uuid`, p.UserID).Scan(&requesterEmail)
	_ = notifications.InsertForAdminsWithPermission(
		r.Context(),
		h.DB,
		PermAdminsBreakGlass,
		notifications.KindAdminBreakGlassRequest,
		[]string{strings.TrimSpace(requesterEmail), perm},
		"break_glass:"+id,
		p.UserID,
		id,
	)

	h.writeJSON(w, http.StatusCreated, map[string]string{
		"id":      id,
		"message": h.msg("ADMIN_BREAK_GLASS_REQUESTED"),
	})
}

func (h *Handler) approveBreakGlass(w http.ResponseWriter, r *http.Request) {
	h.resolveBreakGlass(w, r, true)
}

func (h *Handler) denyBreakGlass(w http.ResponseWriter, r *http.Request) {
	h.resolveBreakGlass(w, r, false)
}

func (h *Handler) revokeBreakGlass(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	id := chi.URLParam(r, "id")
	tag, err := h.DB.Exec(r.Context(), `
		update public.admin_break_glass
		set status = 'revoked', ends_at = now()
		where id = $1::uuid and status = 'approved'
		  and (requester_id = $2::uuid or $3 = true)
	`, id, p.UserID, p.IsOwner)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_BREAK_GLASS_NOT_FOUND")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "break_glass.revoke", "admin_break_glass", id, nil, map[string]string{"status": "revoked"}, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_BREAK_GLASS_REVOKED")})
}

func (h *Handler) resolveBreakGlass(w http.ResponseWriter, r *http.Request, approve bool) {
	p := PrincipalFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if approve && !p.IsOwner {
		// Dual control: only owners may approve (requester cannot be owner-self via check below).
		h.writeErr(w, http.StatusForbidden, "ADMIN_PERMISSION_DENIED")
		return
	}
	var requester string
	err := h.DB.QueryRow(r.Context(), `
		select requester_id::text from public.admin_break_glass where id = $1::uuid and status = 'pending'
	`, id).Scan(&requester)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_BREAK_GLASS_NOT_FOUND")
		return
	}
	if requester == p.UserID {
		h.writeErr(w, http.StatusForbidden, "ADMIN_BREAK_GLASS_SELF_APPROVE")
		return
	}
	status := "denied"
	if approve {
		status = "approved"
	}
	var ttlMin *int
	err = h.DB.QueryRow(r.Context(), `
		select break_glass_ttl_minutes from public.admin_retention_settings where id = 'default'
	`).Scan(&ttlMin)
	if approve {
		if err != nil || ttlMin == nil || *ttlMin < 1 {
			h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_BREAK_GLASS_TTL_MISSING")
			return
		}
	}
	ttl := 0
	if ttlMin != nil {
		ttl = *ttlMin
	}
	_, err = h.DB.Exec(r.Context(), `
		update public.admin_break_glass set
		  status = $2,
		  approver_id = $3::uuid,
		  starts_at = case when $2 = 'approved' then now() else starts_at end,
		  ends_at = case when $2 = 'approved' then now() + make_interval(mins => $4) else ends_at end
		where id = $1::uuid and status = 'pending'
	`, id, status, p.UserID, ttl)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	action := "break_glass.deny"
	if approve {
		action = "break_glass.approve"
	}
	h.audit(r.Context(), r, action, "admin_break_glass", id, nil, map[string]string{"status": status}, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_BREAK_GLASS_RESOLVED")})
}
