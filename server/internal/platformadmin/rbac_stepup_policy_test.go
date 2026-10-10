package platformadmin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/usetrim/trim/server/internal/platformadmin"
)

// Guard: CRUD policy must stay enroll-only (no per-action passkey/TOTP re-prompt).
func TestAdminStepUpCRUDPolicyEnrollOnly(t *testing.T) {
	if platformadmin.AdminStepUpCRUDPolicy != "enroll_only" {
		t.Fatalf("AdminStepUpCRUDPolicy=%q want enroll_only", platformadmin.AdminStepUpCRUDPolicy)
	}
}

func TestRequirePermissionSetsEnrollOnlyHeader(t *testing.T) {
	h := &platformadmin.Handler{}
	mw := h.RequirePermission("admin.access")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No principal -> forbidden, but policy header must still be set first.
	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next must not run without principal")
	})).ServeHTTP(rr, req)
	got := rr.Header().Get("X-Trim-Admin-StepUp-Policy")
	if got != "enroll_only" {
		t.Fatalf("header=%q want enroll_only", got)
	}
	if !strings.Contains(rr.Body.String(), "ADMIN_FORBIDDEN") {
		t.Fatalf("body=%q want ADMIN_FORBIDDEN", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "ADMIN_STEP_UP_REQUIRED") {
		t.Fatalf("must not return ADMIN_STEP_UP_REQUIRED: %s", rr.Body.String())
	}
}
