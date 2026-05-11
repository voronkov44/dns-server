package res

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJsonSetsContentTypeStatusCodeAndBody(t *testing.T) {
	response := httptest.NewRecorder()

	Json(response, map[string]string{"status": "ok"}, http.StatusCreated)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json")
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body status = %q, want %q", body["status"], "ok")
	}
}
