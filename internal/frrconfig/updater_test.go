package frrconfig

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUpdaterForAddress(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	updater := UpdaterForAddress(server.URL[7:], tmpfile.Name()) // Remove "http://"

	err = updater("test config")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	expectedContent := "test config"
	if string(content) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(content))
	}

	// Test HTTP failure
	server.Close()
	err = updater("test config")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}