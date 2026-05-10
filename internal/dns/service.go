package dns

import (
	"context"
	"errors"
	"net/netip"
	"strings"
)

var (
	ErrInvalidServerAddress = errors.New("invalid DNS server address")
	ErrServerAlreadyExists  = errors.New("DNS server already exists")
	ErrServerNotFound       = errors.New("DNS server not found")
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) ListServers(ctx context.Context) ([]Server, error) {
	return s.repository.ListServers(ctx)
}

func (s *Service) AddServer(ctx context.Context, address string) ([]Server, error) {
	server, err := parseServerAddress(address)
	if err != nil {
		return nil, err
	}

	return s.repository.AddServer(ctx, server)
}

func (s *Service) DeleteServer(ctx context.Context, address string) ([]Server, error) {
	server, err := parseServerAddress(address)
	if err != nil {
		return nil, err
	}

	return s.repository.DeleteServer(ctx, server)
}

func parseServerAddress(address string) (Server, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return Server{}, ErrInvalidServerAddress
	}

	// Поддерживает IPv4 и IPv6
	parsed, err := netip.ParseAddr(address)
	if err != nil {
		return Server{}, ErrInvalidServerAddress
	}

	return Server{
		Address: parsed.String(),
	}, nil
}
