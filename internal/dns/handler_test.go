package dns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestHandlerHealthCheck(t *testing.T) {
	router := newTestRouter(t, "")

	response := serveRequest(router, http.MethodGet, "/healthz", "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body HealthResponse
	decodeResponse(t, response, &body)
	if body.Status != "ok" {
		t.Fatalf("status response = %q, want %q", body.Status, "ok")
	}
}

func TestHandlerListServers(t *testing.T) {
	router := newTestRouter(t, "# comment\nnameserver 1.1.1.1\nsearch example.local\nnameserver 8.8.8.8\n")

	response := serveRequest(router, http.MethodGet, "/dns", "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body ServersResponse
	decodeResponse(t, response, &body)

	want := []string{"1.1.1.1", "8.8.8.8"}
	if !slices.Equal(body.Servers, want) {
		t.Fatalf("servers = %v, want %v", body.Servers, want)
	}
}

func TestHandlerAddServer(t *testing.T) {
	router := newTestRouter(t, "")

	response := serveRequest(router, http.MethodPost, "/dns", `{"server":"8.8.8.8"}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body ServersResponse
	decodeResponse(t, response, &body)
	if !slices.Equal(body.Servers, []string{"8.8.8.8"}) {
		t.Fatalf("servers = %v, want %v", body.Servers, []string{"8.8.8.8"})
	}
}

func TestHandlerAddServerInvalidJSON(t *testing.T) {
	router := newTestRouter(t, "")

	response := serveRequest(router, http.MethodPost, "/dns", `{"server":`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestHandlerAddServerRejectsUnknownField(t *testing.T) {
	router := newTestRouter(t, "")

	response := serveRequest(router, http.MethodPost, "/dns", `{"server":"8.8.8.8","extra":true}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestHandlerAddServerDuplicate(t *testing.T) {
	router := newTestRouter(t, "nameserver 8.8.8.8\n")

	response := serveRequest(router, http.MethodPost, "/dns", `{"server":"8.8.8.8"}`)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestHandlerAddServerInvalidIP(t *testing.T) {
	router := newTestRouter(t, "")

	response := serveRequest(router, http.MethodPost, "/dns", `{"server":"not-an-ip"}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestHandlerDeleteServerByQuery(t *testing.T) {
	router := newTestRouter(t, "nameserver 1.1.1.1\nnameserver 8.8.8.8\n")

	response := serveRequest(router, http.MethodDelete, "/dns?server=1.1.1.1", "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body ServersResponse
	decodeResponse(t, response, &body)
	if !slices.Equal(body.Servers, []string{"8.8.8.8"}) {
		t.Fatalf("servers = %v, want %v", body.Servers, []string{"8.8.8.8"})
	}
}

func TestHandlerDeleteServerByJSONBodyFallback(t *testing.T) {
	router := newTestRouter(t, "nameserver 1.1.1.1\nnameserver 8.8.8.8\n")

	response := serveRequest(router, http.MethodDelete, "/dns", `{"server":"8.8.8.8"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body ServersResponse
	decodeResponse(t, response, &body)
	if !slices.Equal(body.Servers, []string{"1.1.1.1"}) {
		t.Fatalf("servers = %v, want %v", body.Servers, []string{"1.1.1.1"})
	}
}

func TestHandlerDeleteServerWithoutQueryOrBody(t *testing.T) {
	router := newTestRouter(t, "nameserver 1.1.1.1\n")

	response := serveRequest(router, http.MethodDelete, "/dns", "")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestHandlerDeleteServerNotFound(t *testing.T) {
	router := newTestRouter(t, "nameserver 1.1.1.1\n")

	response := serveRequest(router, http.MethodDelete, "/dns?server=8.8.8.8", "")

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func newTestRouter(t *testing.T, resolvContent string) http.Handler {
	t.Helper()

	path := filepath.Join(t.TempDir(), "resolv.conf")
	if resolvContent != "" {
		if err := os.WriteFile(path, []byte(resolvContent), 0644); err != nil {
			t.Fatalf("write test resolv.conf: %v", err)
		}
	}

	router := http.NewServeMux()
	repo := NewFileRepository(path)
	service := NewService(repo)
	NewHandler(router, HandlerDeps{Service: service})

	return router
}

func serveRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	var requestBody *strings.Reader
	if body == "" {
		requestBody = strings.NewReader("")
	} else {
		requestBody = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, target, requestBody)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, body interface{}) {
	t.Helper()

	if err := json.NewDecoder(response.Body).Decode(body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}
