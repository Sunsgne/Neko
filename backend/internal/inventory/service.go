// Package inventory implements device onboarding and the capability matrix.
package inventory

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/store"
)

// ErrInvalidInput indicates a validation failure.
var ErrInvalidInput = errors.New("invalid input")

// IDFunc generates unique identifiers.
type IDFunc func() string

// NowFunc returns the current time.
type NowFunc func() time.Time

// Service contains device inventory business logic.
type Service struct {
	repo store.DeviceRepository
	id   IDFunc
	now  NowFunc
}

// NewService builds an inventory service.
func NewService(repo store.DeviceRepository, id IDFunc, now NowFunc) *Service {
	return &Service{repo: repo, id: id, now: now}
}

// RegisterInput is the payload for registering a device for onboarding.
type RegisterInput struct {
	Name        string `json:"name"`
	MgmtAddress string `json:"mgmt_address"`
}

// Register creates a device record in the "discovered" trust state. Capability
// discovery happens asynchronously via the inventory worker (Epic 2); here we
// only validate and persist the initial record so onboarding can proceed.
func (s *Service) Register(ctx context.Context, tenantID string, in RegisterInput) (*store.Device, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	addr := strings.TrimSpace(in.MgmtAddress)
	if addr == "" {
		return nil, fmt.Errorf("%w: mgmt_address is required", ErrInvalidInput)
	}
	if net.ParseIP(addr) == nil {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			// allow hostnames, but reject obviously empty/invalid values
			if !validHostname(addr) {
				return nil, fmt.Errorf("%w: mgmt_address must be an IP or hostname", ErrInvalidInput)
			}
		}
	}
	now := s.now()
	d := &store.Device{
		ID:          s.id(),
		TenantID:    tenantID,
		Name:        name,
		MgmtAddress: addr,
		Platform:    store.PlatformUnknown,
		TrustState:  store.TrustDiscovered,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// Get returns a device by id within tenant scope.
func (s *Service) Get(ctx context.Context, tenantID, id string) (*store.Device, error) {
	return s.repo.Get(ctx, tenantID, id)
}

// List returns a page of devices within tenant scope.
func (s *Service) List(ctx context.Context, tenantID string, page store.Page) ([]*store.Device, int, error) {
	return s.repo.List(ctx, tenantID, page)
}

func validHostname(h string) bool {
	if h == "" || len(h) > 253 {
		return false
	}
	for _, label := range strings.Split(h, ".") {
		if label == "" {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}
