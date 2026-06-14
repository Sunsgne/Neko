// Package inventory implements device onboarding and the capability matrix.
package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/neko/sdwan/backend/internal/routeros"
	"github.com/neko/sdwan/backend/internal/secret"
	"github.com/neko/sdwan/backend/internal/store"
)

// ErrInvalidInput indicates a validation failure.
var ErrInvalidInput = errors.New("invalid input")

// ErrTransitionNotAllowed indicates an illegal trust-state change.
var ErrTransitionNotAllowed = errors.New("trust transition not allowed")

// ErrNotEnrolled indicates the device has no stored credentials.
var ErrNotEnrolled = errors.New("device not enrolled (no stored credentials)")

// IDFunc generates unique identifiers.
type IDFunc func() string

// NowFunc returns the current time.
type NowFunc func() time.Time

// StatusProbe reads live device status over a management protocol.
type StatusProbe interface {
	Status(ctx context.Context, t routeros.Target) (store.DeviceStatus, error)
}

// Service contains device inventory business logic.
type Service struct {
	repo      store.DeviceRepository
	creds     store.CredentialRepository
	collector routeros.Collector
	probe     StatusProbe
	sealer    *secret.Sealer
	id        IDFunc
	now       NowFunc
}

// Deps are the dependencies for the inventory service.
type Deps struct {
	Devices     store.DeviceRepository
	Credentials store.CredentialRepository
	Collector   routeros.Collector
	Probe       StatusProbe
	Sealer      *secret.Sealer
	ID          IDFunc
	Now         NowFunc
}

// NewService builds an inventory service.
func NewService(d Deps) *Service {
	return &Service{
		repo:      d.Devices,
		creds:     d.Credentials,
		collector: d.Collector,
		probe:     d.Probe,
		sealer:    d.Sealer,
		id:        d.ID,
		now:       d.Now,
	}
}

type storedCreds struct {
	Username string `json:"u"`
	Password string `json:"p"`
}

// Enroll stores device credentials (encrypted), connects to the device to pull
// its facts/capabilities, and transitions it to managed. This is real device
// onboarding (托管): the platform holds the credentials and operates the device
// thereafter without anyone logging in.
func (s *Service) Enroll(ctx context.Context, tenantID, id, username, password string) (*store.Device, error) {
	if s.collector == nil || s.creds == nil || s.sealer == nil {
		return nil, fmt.Errorf("%w: enrollment not configured", ErrInvalidInput)
	}
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	target := routeros.Target{Address: d.MgmtAddress, Username: username, Secret: password}
	facts, err := s.collector.Collect(ctx, target)
	if err != nil {
		return nil, err // unreachable / bad creds
	}

	blob, err := json.Marshal(storedCreds{Username: username, Password: password})
	if err != nil {
		return nil, err
	}
	sealed, err := s.sealer.Seal(blob)
	if err != nil {
		return nil, err
	}
	if err := s.creds.Put(ctx, store.Credential{DeviceID: d.ID, Kind: "api", Sealed: sealed}); err != nil {
		return nil, err
	}

	det := routeros.Detect(*facts)
	caps := det.Capabilities
	d.Platform, d.Model = det.Platform, det.Model
	if det.Serial != "" {
		d.Serial = det.Serial
	}
	d.Capabilities = &caps
	d.Enrolled = true
	d.TrustState = store.TrustManaged
	now := s.now()
	d.LastSeenAt, d.UpdatedAt = &now, now
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	// Best-effort immediate status poll.
	_, _ = s.Poll(ctx, tenantID, id)
	return s.repo.Get(ctx, tenantID, id)
}

// Poll connects to an enrolled device using stored credentials and refreshes
// its live status (online/version/cpu/mem/interfaces).
func (s *Service) Poll(ctx context.Context, tenantID, id string) (*store.Device, error) {
	if s.probe == nil || s.creds == nil || s.sealer == nil {
		return nil, fmt.Errorf("%w: polling not configured", ErrInvalidInput)
	}
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	cred, err := s.creds.Get(ctx, d.ID)
	if err != nil {
		return nil, ErrNotEnrolled
	}
	plain, err := s.sealer.Open(cred.Sealed)
	if err != nil {
		return nil, err
	}
	var sc storedCreds
	if err := json.Unmarshal(plain, &sc); err != nil {
		return nil, err
	}
	now := s.now()
	status, perr := s.probe.Status(ctx, routeros.Target{Address: d.MgmtAddress, Username: sc.Username, Secret: sc.Password})
	status.LastPolledAt = &now
	if perr != nil {
		status.Online = false
		status.LastError = perr.Error()
	} else {
		d.LastSeenAt = &now
	}
	d.Status = &status
	d.UpdatedAt = now
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// RegisterInput is the payload for registering a device for onboarding.
type RegisterInput struct {
	Name        string `json:"name"`
	MgmtAddress string `json:"mgmt_address"`
	// Role defaults to cpe. Use "backbone" for SD-WAN 骨干节点/POP (also ROS),
	// or "gateway" for an exit/gateway node (incl. overseas exit).
	Role   store.DeviceRole `json:"role"`
	Region string           `json:"region"`
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
	role := in.Role
	switch role {
	case "":
		role = store.RoleCPE
	case store.RoleCPE, store.RoleBackbone, store.RoleGateway:
	default:
		return nil, fmt.Errorf("%w: role must be cpe|backbone|gateway", ErrInvalidInput)
	}
	now := s.now()
	d := &store.Device{
		ID:          s.id(),
		TenantID:    tenantID,
		Name:        name,
		MgmtAddress: addr,
		Role:        role,
		Region:      strings.TrimSpace(in.Region),
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

// ListByRole returns devices of a given role within tenant scope. An empty
// role returns all devices. Filtering is applied over a wide page since
// fleets per tenant are modest.
func (s *Service) ListByRole(ctx context.Context, tenantID string, role store.DeviceRole) ([]*store.Device, error) {
	all, _, err := s.repo.List(ctx, tenantID, store.Page{Number: 1, Size: 1000})
	if err != nil {
		return nil, err
	}
	if role == "" {
		return all, nil
	}
	out := make([]*store.Device, 0, len(all))
	for _, d := range all {
		dr := d.Role
		if dr == "" {
			dr = store.RoleCPE
		}
		if dr == role {
			out = append(out, d)
		}
	}
	return out, nil
}

// Detect contacts the device, identifies its model/platform/capabilities, and
// advances its trust state from discovered to authenticated. Per requirement
// #5, capabilities are detected (model, version, architecture, packages,
// license, device-mode, interface capabilities) rather than assumed.
func (s *Service) Detect(ctx context.Context, tenantID, id string) (*store.Device, error) {
	if s.collector == nil {
		return nil, fmt.Errorf("%w: no collector configured", ErrInvalidInput)
	}
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	facts, err := s.collector.Collect(ctx, routeros.Target{Address: d.MgmtAddress})
	if err != nil {
		return nil, err
	}
	det := routeros.Detect(*facts)
	caps := det.Capabilities
	d.Platform = det.Platform
	d.Model = det.Model
	if det.Serial != "" {
		d.Serial = det.Serial
	}
	d.Capabilities = &caps
	if CanTransition(d.TrustState, store.TrustAuthenticated) {
		d.TrustState = store.TrustAuthenticated
	}
	now := s.now()
	d.LastSeenAt = &now
	d.UpdatedAt = now
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// SetTrustState transitions a device to the requested trust state, enforcing
// the lifecycle state machine.
func (s *Service) SetTrustState(ctx context.Context, tenantID, id string, to store.TrustState) (*store.Device, error) {
	d, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !CanTransition(d.TrustState, to) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrTransitionNotAllowed, d.TrustState, to)
	}
	d.TrustState = to
	d.UpdatedAt = s.now()
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
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
