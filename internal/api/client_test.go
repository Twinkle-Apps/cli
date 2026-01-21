package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient("https://example.com", "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrMissingAPIKey {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}

func TestGetBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/apps/app_123/builds/42" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		resp := BuildResponse{
			Build: Build{
				ID:          42,
				Status:      "available",
				Version:     "1.0.0",
				BuildNumber: "42",
				InsertedAt:  APITime{Time: time.Now()},
				UpdatedAt:   APITime{Time: time.Now()},
			},
			Appcast: Appcast{
				Status:  "published",
				Message: "ok",
				FeedURL: "https://example.com/feed.xml",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-key", server.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.GetBuild(context.Background(), "app_123", "42")
	if err != nil {
		t.Fatalf("get build: %v", err)
	}

	if resp.Build.ID != 42 {
		t.Fatalf("expected build id 42, got %d", resp.Build.ID)
	}
}

func TestCreateUpload(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/v1/apps/app_123/uploads" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body BuildUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Build.Version != "1.0.0" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		resp := BuildUploadResponse{
			BuildID:     BuildID{value: 99},
			UploadURL:   server.URL + "/upload",
			CompleteURL: server.URL + "/complete",
			StatusURL:   server.URL + "/status",
			WaitURL:     server.URL + "/wait",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "test-key", server.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.CreateUpload(context.Background(), "app_123", BuildUploadParams{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	if resp.BuildID.Int() != 99 {
		t.Fatalf("expected build id 99, got %d", resp.BuildID.Int())
	}
}

func TestUploadFile(t *testing.T) {
	var receivedContentType string
	var receivedSize int64

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "build.zip")
	content := []byte("payload")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		receivedContentType = r.Header.Get("Content-Type")
		data, _ := io.ReadAll(r.Body)
		receivedSize = int64(len(data))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient("https://example.com", "test-key", server.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.UploadFile(context.Background(), server.URL, filePath, "application/zip"); err != nil {
		t.Fatalf("upload file: %v", err)
	}

	if receivedContentType != "application/zip" {
		t.Fatalf("expected content type application/zip, got %s", receivedContentType)
	}
	if receivedSize != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), receivedSize)
	}
}

func TestAPITimeUnmarshal(t *testing.T) {
	var parsed APITime
	data := []byte(`"2026-01-19T01:27:39"`)
	if err := parsed.UnmarshalJSON(data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.IsZero() {
		t.Fatal("expected parsed time")
	}
}

func TestBuildIDUnmarshalString(t *testing.T) {
	var id BuildID
	if err := json.Unmarshal([]byte(`"123"`), &id); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if id.Int() != 123 {
		t.Fatalf("expected 123, got %d", id.Int())
	}
}
