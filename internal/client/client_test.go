package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestClientListServers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/dns" {
			t.Fatalf("path = %s, want /dns", r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, ServersResponse{Servers: []string{"1.1.1.1", "8.8.8.8"}})
	}))
	defer server.Close()

	client := New(server.URL)

	servers, err := client.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	want := []string{"1.1.1.1", "8.8.8.8"}
	if !slices.Equal(servers, want) {
		t.Fatalf("ListServers() = %v, want %v", servers, want)
	}
}

func TestClientAddServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/dns" {
			t.Fatalf("path = %s, want /dns", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want %q", got, "application/json")
		}

		var request ServerRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if request.Server != "8.8.8.8" {
			t.Fatalf("request server = %q, want %q", request.Server, "8.8.8.8")
		}

		writeJSON(t, w, http.StatusCreated, ServersResponse{Servers: []string{"8.8.8.8"}})
	}))
	defer server.Close()

	client := New(server.URL)

	servers, err := client.AddServer(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}
	if !slices.Equal(servers, []string{"8.8.8.8"}) {
		t.Fatalf("AddServer() = %v, want %v", servers, []string{"8.8.8.8"})
	}
}

func TestClientDeleteServerUsesQueryParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodDelete)
		}
		if r.URL.Path != "/dns" {
			t.Fatalf("path = %s, want /dns", r.URL.Path)
		}
		if got := r.URL.Query().Get("server"); got != "2001:db8::1" {
			t.Fatalf("query server = %q, want %q", got, "2001:db8::1")
		}

		writeJSON(t, w, http.StatusOK, ServersResponse{Servers: []string{}})
	}))
	defer server.Close()

	client := New(server.URL)

	servers, err := client.DeleteServer(context.Background(), "2001:db8::1")
	if err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("DeleteServer() = %v, want empty list", servers)
	}
}

func TestClientAPIErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusConflict, ErrorResponse{Error: "DNS server already exists"})
	}))
	defer server.Close()

	client := New(server.URL)

	_, err := client.AddServer(context.Background(), "8.8.8.8")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("AddServer() error = %v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("APIError status = %d, want %d", apiErr.StatusCode, http.StatusConflict)
	}
	if apiErr.Message != "DNS server already exists" {
		t.Fatalf("APIError message = %q, want %q", apiErr.Message, "DNS server already exists")
	}
}

func TestClientMalformedAPIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"servers":`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL)

	_, err := client.ListServers(context.Background())
	if err == nil {
		t.Fatal("ListServers() error = nil, want malformed response error")
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, statusCode int, payload interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("write JSON response: %v", err)
	}
}
