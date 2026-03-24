package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicroldan/ans/registry/internal/store"
)

func TestReportStore(t *testing.T) {
	s := store.New()
	h := NewStoreHandler(s, nil)

	// Create a store entry directly so we have an ID to report.
	entry := s.Create(store.TestEntry("Test Store"))

	body := `{"reason":"prohibited_content","details":"Store sells illegal items"}`
	req := httptest.NewRequest(http.MethodPost, "/registry/v1/stores/"+entry.ID+"/report", strings.NewReader(body))
	req.SetPathValue("id", entry.ID)
	rec := httptest.NewRecorder()

	h.ReportStore(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"reported"`) {
		t.Fatalf("expected reported status in body, got %s", rec.Body.String())
	}
}

func TestReportStore_NotFound(t *testing.T) {
	s := store.New()
	h := NewStoreHandler(s, nil)

	body := `{"reason":"spam"}`
	req := httptest.NewRequest(http.MethodPost, "/registry/v1/stores/nonexistent/report", strings.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	h.ReportStore(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteStore_Unauthorized(t *testing.T) {
	s := store.New()
	h := NewStoreHandler(s, nil)

	entry := s.Create(store.TestEntry("Test Store"))

	// Set the admin token so the endpoint is configured.
	t.Setenv("REGISTRY_ADMIN_TOKEN", "secret-admin-token")

	req := httptest.NewRequest(http.MethodDelete, "/registry/v1/stores/"+entry.ID, nil)
	req.SetPathValue("id", entry.ID)
	rec := httptest.NewRecorder()

	h.DeleteStore(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteStore_Success(t *testing.T) {
	s := store.New()
	h := NewStoreHandler(s, nil)

	entry := s.Create(store.TestEntry("Test Store"))

	t.Setenv("REGISTRY_ADMIN_TOKEN", "secret-admin-token")

	req := httptest.NewRequest(http.MethodDelete, "/registry/v1/stores/"+entry.ID, nil)
	req.SetPathValue("id", entry.ID)
	req.Header.Set("Authorization", "Bearer secret-admin-token")
	rec := httptest.NewRecorder()

	h.DeleteStore(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify the store is actually deleted.
	if _, ok := s.GetByID(entry.ID); ok {
		t.Fatal("store should have been deleted")
	}
}
