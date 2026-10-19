package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

func TestHealthAndState(t *testing.T) {
	server := newTestServer(t)

	health := httptest.NewRecorder()
	server.Handler().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", health.Code, http.StatusOK)
	}

	state := httptest.NewRecorder()
	server.Handler().ServeHTTP(state, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	if state.Code != http.StatusOK {
		t.Fatalf("state status = %d, want %d", state.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(state.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if _, ok := body["styles"]; !ok {
		t.Fatal("state response does not include styles")
	}
}

func TestParseShareEndpoint(t *testing.T) {
	server := newTestServer(t)
	deepLink := "yokasgs://?payload=game_record"
	shareURL := "https://www.sanguosha.cn/share/index.html?reqdata=" + url.QueryEscape(deepLink)
	body, _ := json.Marshal(map[string]string{"url": shareURL})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/share/parse", bytes.NewReader(body))
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var result map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["deep_link"] != deepLink {
		t.Fatalf("deep_link = %q, want %q", result["deep_link"], deepLink)
	}
}

func TestModelConfigRejectsSecretValueField(t *testing.T) {
	server := newTestServer(t)
	body := []byte(`{
		"base_url":"https://example.com/v1",
		"model":"example",
		"api_key_env":"SANGUOCLIP_MODEL_API_KEY",
		"temperature":0.2,
		"api_key":"must-not-be-accepted"
	}`)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/model", bytes.NewReader(body))
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestExportIsInlineUnlessDownloadRequested(t *testing.T) {
	server := newTestServer(t)
	edit := domain.Edit{
		ID:         "edit-preview",
		Title:      "浏览器预览",
		Status:     "completed",
		OutputName: "edit-preview.mp4",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := server.store.Update(func(state *domain.State) error {
		state.Edits = append(state.Edits, edit)
		return nil
	}); err != nil {
		t.Fatalf("add edit: %v", err)
	}
	if err := os.WriteFile(server.media.ExportPath(edit.OutputName), []byte("video"), 0o600); err != nil {
		t.Fatalf("write export: %v", err)
	}

	inline := httptest.NewRecorder()
	server.Handler().ServeHTTP(inline, httptest.NewRequest(http.MethodGet, "/api/exports/"+edit.ID, nil))
	if inline.Code != http.StatusOK {
		t.Fatalf("inline status = %d, want %d", inline.Code, http.StatusOK)
	}
	if disposition := inline.Header().Get("Content-Disposition"); disposition != `inline; filename="浏览器预览.mp4"` {
		t.Fatalf("inline Content-Disposition = %q", disposition)
	}

	download := httptest.NewRecorder()
	server.Handler().ServeHTTP(download, httptest.NewRequest(http.MethodGet, "/api/exports/"+edit.ID+"?download=1", nil))
	if disposition := download.Header().Get("Content-Disposition"); disposition != `attachment; filename="浏览器预览.mp4"` {
		t.Fatalf("download Content-Disposition = %q", disposition)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server, err := New(t.TempDir(), logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server
}
