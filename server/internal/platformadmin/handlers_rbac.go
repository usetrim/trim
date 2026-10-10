package platformadmin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/pagination"
)

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	rows, err := db.Query(r.Context(), `
		select code, description, category, step_up_required, sort_order
		from public.platform_permission_catalog
		order by sort_order asc, code asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type item struct {
		Code           string `json:"code"`
		Description    string `json:"description"`
		Category       string `json:"category"`
		StepUpRequired bool   `json:"step_up_required"`
		SortOrder      int    `json:"sort_order"`
	}
	items := make([]item, 0)
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.Code, &it.Description, &it.Category, &it.StepUpRequired, &it.SortOrder); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, it)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(name) like $%d
			or lower(slug) like $%d
			or lower(coalesce(description, '')) like $%d
			or id::text like $%d
		)`, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.platform_roles`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id::text, name, slug, description, is_system, is_owner, created_at::text
		from public.platform_roles
		%s
		order by is_owner desc, name asc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type roleRow struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Slug        string   `json:"slug"`
		Description string   `json:"description"`
		IsSystem    bool     `json:"is_system"`
		IsOwner     bool     `json:"is_owner"`
		CreatedAt   string   `json:"created_at"`
		Permissions []string `json:"permissions"`
	}
	items := make([]roleRow, 0)
	for rows.Next() {
		var row roleRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Slug, &row.Description, &row.IsSystem, &row.IsOwner, &row.CreatedAt); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		prows, err := db.Query(r.Context(), `
			select permission_code from public.platform_role_permissions where role_id = $1::uuid
		`, row.ID)
		if err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		perms := make([]string, 0)
		for prows.Next() {
			var c string
			if prows.Scan(&c) == nil {
				perms = append(perms, c)
			}
		}
		prows.Close()
		row.Permissions = perms
		items = append(items, row)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

// listRoleOptions returns id/name/slug for invite selects (full assignable catalog, no invent).
func (h *Handler) listRoleOptions(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	rows, err := db.Query(r.Context(), `
		select id::text, name, slug
		from public.platform_roles
		order by is_owner desc, name asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type opt struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	items := make([]opt, 0)
	for rows.Next() {
		var it opt
		if err := rows.Scan(&it.ID, &it.Name, &it.Slug); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, it)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createRoleBody struct {
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) postRole(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body createRoleBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(body.Slug))
	if slug == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, slug)
	var roleID string
	err := h.DB.QueryRow(r.Context(), `
		insert into public.platform_roles (name, slug, description, is_system, is_owner, created_by)
		values ($1, $2, $3, false, false, nullif($4, '')::uuid)
		returning id::text
	`, name, slug, strings.TrimSpace(body.Description), p.UserID).Scan(&roleID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := h.setRolePermissions(r, roleID, false, body.Permissions); err != nil {
		h.writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "rbac.role_create", "platform_role", roleID, nil, body, "", step)
	h.writeJSON(w, http.StatusCreated, map[string]string{
		"id":      roleID,
		"message": h.msg("ADMIN_ROLE_CREATED"),
	})
}

type patchRoleBody struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) patchRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var isOwner, isSystem bool
	err := h.DB.QueryRow(r.Context(), `
		select is_owner, is_system from public.platform_roles where id = $1::uuid
	`, roleID).Scan(&isOwner, &isSystem)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_ROLE_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var body patchRoleBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name != "" {
			_, _ = h.DB.Exec(r.Context(), `update public.platform_roles set name = $2, updated_at = now() where id = $1::uuid`, roleID, name)
		}
	}
	if body.Description != nil {
		_, _ = h.DB.Exec(r.Context(), `update public.platform_roles set description = $2, updated_at = now() where id = $1::uuid`, roleID, strings.TrimSpace(*body.Description))
	}
	if body.Permissions != nil {
		if isOwner {
			// Owner must retain full catalog.
			var catalog int
			_ = h.DB.QueryRow(r.Context(), `select count(*) from public.platform_permission_catalog`).Scan(&catalog)
			if len(body.Permissions) < catalog {
				h.writeErr(w, http.StatusForbidden, "ADMIN_ROLE_OWNER_LOCKED")
				return
			}
		}
		if err := h.setRolePermissions(r, roleID, isOwner, body.Permissions); err != nil {
			h.writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "rbac.role_update", "platform_role", roleID, nil, body, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_ROLE_SAVED")})
}

func (h *Handler) setRolePermissions(r *http.Request, roleID string, isOwner bool, codes []string) error {
	if isOwner {
		return nil
	}
	seen := map[string]struct{}{}
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		var ok bool
		err := h.DB.QueryRow(r.Context(), `select true from public.platform_permission_catalog where code = $1`, c).Scan(&ok)
		if err == pgx.ErrNoRows {
			return fmt.Errorf("ADMIN_PERMISSION_UNKNOWN")
		}
		if err != nil {
			return fmt.Errorf("DATABASE_UNAVAILABLE")
		}
		seen[c] = struct{}{}
	}
	if _, err := h.DB.Exec(r.Context(), `delete from public.platform_role_permissions where role_id = $1::uuid`, roleID); err != nil {
		return fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	for c := range seen {
		if _, err := h.DB.Exec(r.Context(), `
			insert into public.platform_role_permissions (role_id, permission_code) values ($1::uuid, $2)
			on conflict do nothing
		`, roleID, c); err != nil {
			return fmt.Errorf("DATABASE_UNAVAILABLE")
		}
	}
	return nil
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var isSystem bool
	err := h.DB.QueryRow(r.Context(), `select is_system from public.platform_roles where id = $1::uuid`, roleID).Scan(&isSystem)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_ROLE_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if isSystem {
		h.writeErr(w, http.StatusForbidden, "ADMIN_ROLE_SYSTEM_LOCKED")
		return
	}
	var inUse int
	if err := h.DB.QueryRow(r.Context(), `
		select count(*) from public.platform_admins where role_id = $1::uuid
	`, roleID).Scan(&inUse); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if inUse > 0 {
		h.writeErr(w, http.StatusConflict, "ADMIN_ROLE_IN_USE")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `delete from public.platform_roles where id = $1::uuid and is_system = false`, roleID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusForbidden, "ADMIN_ROLE_SYSTEM_LOCKED")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "rbac.role_delete", "platform_role", roleID, nil, nil, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_ROLE_DELETED")})
}

func (h *Handler) listAdmins(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(p.email) like $%d
			or lower(r.name) like $%d
			or lower(r.slug) like $%d
			or lower(a.status) like $%d
			or a.user_id::text like $%d
		)`, argN, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	countSQL := `
		select count(*)
		from public.platform_admins a
		join public.profiles p on p.id = a.user_id
		join public.platform_roles r on r.id = a.role_id
	` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select a.user_id::text, p.email, a.role_id::text, r.slug, r.name, r.is_owner, a.status, a.created_at::text
		from public.platform_admins a
		join public.profiles p on p.id = a.user_id
		join public.platform_roles r on r.id = a.role_id
		%s
		order by r.is_owner desc, p.email asc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type row struct {
		UserID      string `json:"user_id"`
		Email       string `json:"email"`
		RoleID      string `json:"role_id"`
		RoleSlug    string `json:"role_slug"`
		RoleName    string `json:"role_name"`
		IsOwner     bool   `json:"is_owner"`
		Status      string `json:"status"`
		StatusLabel string `json:"status_label"`
		CreatedAt   string `json:"created_at"`
	}
	items := make([]row, 0)
	for rows.Next() {
		var it row
		if err := rows.Scan(&it.UserID, &it.Email, &it.RoleID, &it.RoleSlug, &it.RoleName, &it.IsOwner, &it.Status, &it.CreatedAt); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		it.StatusLabel = h.platformAdminStatusLabel(it.Status)
		items = append(items, it)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

type assignAdminBody struct {
	Email  string `json:"email"`
	RoleID string `json:"role_id"`
}

func (h *Handler) postAdmin(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body assignAdminBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	roleID := strings.TrimSpace(body.RoleID)
	if email == "" || roleID == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	var uid string
	err := h.DB.QueryRow(r.Context(), `select id::text from public.profiles where lower(email) = $1`, email).Scan(&uid)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var roleOK bool
	err = h.DB.QueryRow(r.Context(), `select true from public.platform_roles where id = $1::uuid`, roleID).Scan(&roleOK)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_ROLE_NOT_FOUND")
		return
	}
	_, err = h.DB.Exec(r.Context(), `
		insert into public.platform_admins (user_id, role_id, status, invited_by)
		values ($1::uuid, $2::uuid, 'active', nullif($3, '')::uuid)
		on conflict (user_id) do update set role_id = excluded.role_id, status = 'active', updated_at = now()
	`, uid, roleID, p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "rbac.admin_assign", "platform_admin", uid, nil, body, "", step)
	h.writeJSON(w, http.StatusCreated, map[string]string{
		"user_id": uid,
		"message": h.msg("ADMIN_ADMIN_INVITED"),
	})
}

func (h *Handler) deleteAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	p := PrincipalFromContext(r.Context())
	var isOwner bool
	err := h.DB.QueryRow(r.Context(), `
		select r.is_owner from public.platform_admins a
		join public.platform_roles r on r.id = a.role_id
		where a.user_id = $1::uuid
	`, userID).Scan(&isOwner)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_ADMIN_NOT_FOUND")
		return
	}
	if isOwner {
		n, _ := h.countActiveOwners(r.Context())
		if n <= 1 {
			h.writeErr(w, http.StatusForbidden, "ADMIN_CANNOT_REMOVE_LAST_OWNER")
			return
		}
	}
	tag, err := h.DB.Exec(r.Context(), `delete from public.platform_admins where user_id = $1::uuid`, userID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_ADMIN_NOT_FOUND")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "rbac.admin_remove", "platform_admin", userID, nil, nil, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_ADMIN_REMOVED")})
}
