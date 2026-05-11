package dns

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFileRepositoryListServers(t *testing.T) {
	path := tempResolvPath(t)
	content := `
# managed elsewhere
search example.local
nameserver 8.8.8.8
options rotate
nameserver 2001:0db8:0000:0000:0000:0000:0000:0001
nameserver not-an-ip
sortlist 10.0.0.0/8
nameserver 1.1.1.1 # inline comment is tolerated
`
	writeFile(t, path, content)

	repo := NewFileRepository(path)

	servers, err := repo.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	want := []string{"8.8.8.8", "2001:db8::1", "1.1.1.1"}
	if got := serverAddresses(servers); !slices.Equal(got, want) {
		t.Fatalf("ListServers() = %v, want %v", got, want)
	}
}

func TestFileRepositoryListServersCreatesMissingFile(t *testing.T) {
	path := tempResolvPath(t)
	repo := NewFileRepository(path)

	servers, err := repo.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	if len(servers) != 0 {
		t.Fatalf("ListServers() = %v, want empty list", servers)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected repository to create missing resolv.conf: %v", err)
	}
}

func TestFileRepositoryAddServerCreatesFileAndReturnsCurrentServers(t *testing.T) {
	path := tempResolvPath(t)
	repo := NewFileRepository(path)

	servers, err := repo.AddServer(context.Background(), Server{Address: "8.8.8.8"})
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	wantServers := []string{"8.8.8.8"}
	if got := serverAddresses(servers); !slices.Equal(got, wantServers) {
		t.Fatalf("AddServer() = %v, want %v", got, wantServers)
	}

	wantContent := "nameserver 8.8.8.8\n"
	if got := readFile(t, path); got != wantContent {
		t.Fatalf("resolv.conf content = %q, want %q", got, wantContent)
	}
}

func TestFileRepositoryAddServerPreservesExistingLines(t *testing.T) {
	path := tempResolvPath(t)
	initial := "# keep this comment\nsearch example.local\nnameserver 1.1.1.1\noptions rotate"
	writeFile(t, path, initial)

	repo := NewFileRepository(path)

	servers, err := repo.AddServer(context.Background(), Server{Address: "8.8.8.8"})
	if err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	wantServers := []string{"1.1.1.1", "8.8.8.8"}
	if got := serverAddresses(servers); !slices.Equal(got, wantServers) {
		t.Fatalf("AddServer() = %v, want %v", got, wantServers)
	}

	wantContent := initial + "\nnameserver 8.8.8.8\n"
	if got := readFile(t, path); got != wantContent {
		t.Fatalf("resolv.conf content = %q, want %q", got, wantContent)
	}
}

func TestFileRepositoryAddServerRejectsDuplicateIPv4(t *testing.T) {
	path := tempResolvPath(t)
	initial := "nameserver 8.8.8.8\nsearch example.local\n"
	writeFile(t, path, initial)

	repo := NewFileRepository(path)

	servers, err := repo.AddServer(context.Background(), Server{Address: "8.8.8.8"})
	if !errors.Is(err, ErrServerAlreadyExists) {
		t.Fatalf("AddServer() error = %v, want %v", err, ErrServerAlreadyExists)
	}
	if servers != nil {
		t.Fatalf("AddServer() servers = %v, want nil on duplicate", servers)
	}

	if got := readFile(t, path); got != initial {
		t.Fatalf("resolv.conf content changed to %q, want %q", got, initial)
	}
}

func TestFileRepositoryAddServerRejectsDuplicateIPv6DifferentTextForms(t *testing.T) {
	path := tempResolvPath(t)
	initial := "nameserver 2001:db8::1\n"
	writeFile(t, path, initial)

	repo := NewFileRepository(path)

	_, err := repo.AddServer(context.Background(), Server{
		Address: "2001:0db8:0000:0000:0000:0000:0000:0001",
	})
	if !errors.Is(err, ErrServerAlreadyExists) {
		t.Fatalf("AddServer() error = %v, want %v", err, ErrServerAlreadyExists)
	}

	if got := readFile(t, path); got != initial {
		t.Fatalf("resolv.conf content changed to %q, want %q", got, initial)
	}
}

func TestFileRepositoryDeleteServerRemovesTargetAndPreservesOtherLines(t *testing.T) {
	path := tempResolvPath(t)
	initial := "# keep this comment\nnameserver 1.1.1.1\nsearch example.local\nnameserver 8.8.8.8\noptions rotate\n"
	writeFile(t, path, initial)

	repo := NewFileRepository(path)

	servers, err := repo.DeleteServer(context.Background(), Server{Address: "1.1.1.1"})
	if err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}

	wantServers := []string{"8.8.8.8"}
	if got := serverAddresses(servers); !slices.Equal(got, wantServers) {
		t.Fatalf("DeleteServer() = %v, want %v", got, wantServers)
	}

	wantContent := "# keep this comment\nsearch example.local\nnameserver 8.8.8.8\noptions rotate\n"
	if got := readFile(t, path); got != wantContent {
		t.Fatalf("resolv.conf content = %q, want %q", got, wantContent)
	}
}

func TestFileRepositoryDeleteServerReturnsNotFound(t *testing.T) {
	path := tempResolvPath(t)
	initial := "nameserver 1.1.1.1\nsearch example.local\n"
	writeFile(t, path, initial)

	repo := NewFileRepository(path)

	servers, err := repo.DeleteServer(context.Background(), Server{Address: "8.8.8.8"})
	if !errors.Is(err, ErrServerNotFound) {
		t.Fatalf("DeleteServer() error = %v, want %v", err, ErrServerNotFound)
	}
	if servers != nil {
		t.Fatalf("DeleteServer() servers = %v, want nil when missing", servers)
	}

	if got := readFile(t, path); got != initial {
		t.Fatalf("resolv.conf content changed to %q, want %q", got, initial)
	}
}

func tempResolvPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "resolv.conf")
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test resolv.conf: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read test resolv.conf: %v", err)
	}

	return string(content)
}

func serverAddresses(servers []Server) []string {
	addresses := make([]string, 0, len(servers))
	for _, server := range servers {
		addresses = append(addresses, server.Address)
	}

	return addresses
}
