package dns

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestServiceAddServerValidatesAndCallsRepository(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{
			name:    "valid IPv4",
			address: "8.8.8.8",
			want:    "8.8.8.8",
		},
		{
			name:    "valid IPv6",
			address: "2001:0db8:0000:0000:0000:0000:0000:0001",
			want:    "2001:db8::1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepository{}
			service := NewService(repo)

			servers, err := service.AddServer(context.Background(), test.address)
			if err != nil {
				t.Fatalf("AddServer() error = %v", err)
			}

			if repo.addCalls != 1 {
				t.Fatalf("repository AddServer calls = %d, want 1", repo.addCalls)
			}
			if repo.added.Address != test.want {
				t.Fatalf("repository AddServer address = %q, want %q", repo.added.Address, test.want)
			}
			if got := serverAddresses(servers); !slices.Equal(got, []string{test.want}) {
				t.Fatalf("AddServer() = %v, want %v", got, []string{test.want})
			}
		})
	}
}

func TestServiceAddServerRejectsInvalidAddressBeforeRepository(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "empty",
			address: "",
		},
		{
			name:    "blank",
			address: "   ",
		},
		{
			name:    "invalid",
			address: "not-an-ip",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepository{}
			service := NewService(repo)

			servers, err := service.AddServer(context.Background(), test.address)
			if !errors.Is(err, ErrInvalidServerAddress) {
				t.Fatalf("AddServer() error = %v, want %v", err, ErrInvalidServerAddress)
			}
			if servers != nil {
				t.Fatalf("AddServer() servers = %v, want nil", servers)
			}
			if repo.addCalls != 0 {
				t.Fatalf("repository AddServer calls = %d, want 0", repo.addCalls)
			}
		})
	}
}

func TestServiceDeleteServerValidatesBeforeRepository(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	_, err := service.DeleteServer(context.Background(), "2001:0db8:0000:0000:0000:0000:0000:0001")
	if err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}

	if repo.deleteCalls != 1 {
		t.Fatalf("repository DeleteServer calls = %d, want 1", repo.deleteCalls)
	}
	if repo.deleted.Address != "2001:db8::1" {
		t.Fatalf("repository DeleteServer address = %q, want %q", repo.deleted.Address, "2001:db8::1")
	}
}

func TestServiceDeleteServerRejectsInvalidAddressBeforeRepository(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)

	servers, err := service.DeleteServer(context.Background(), "not-an-ip")
	if !errors.Is(err, ErrInvalidServerAddress) {
		t.Fatalf("DeleteServer() error = %v, want %v", err, ErrInvalidServerAddress)
	}
	if servers != nil {
		t.Fatalf("DeleteServer() servers = %v, want nil", servers)
	}
	if repo.deleteCalls != 0 {
		t.Fatalf("repository DeleteServer calls = %d, want 0", repo.deleteCalls)
	}
}

type fakeRepository struct {
	addCalls    int
	deleteCalls int
	listCalls   int
	added       Server
	deleted     Server
}

func (r *fakeRepository) ListServers(context.Context) ([]Server, error) {
	r.listCalls++

	return nil, nil
}

func (r *fakeRepository) AddServer(_ context.Context, server Server) ([]Server, error) {
	r.addCalls++
	r.added = server

	return []Server{server}, nil
}

func (r *fakeRepository) DeleteServer(_ context.Context, server Server) ([]Server, error) {
	r.deleteCalls++
	r.deleted = server

	return []Server{server}, nil
}
