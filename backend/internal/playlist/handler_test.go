package playlist

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImportRejectsWrongMethod(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/playlist/import", nil)
	rec := httptest.NewRecorder()

	handler.Import(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestImportRejectsMalformedBody(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/playlist/import", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	handler.Import(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
