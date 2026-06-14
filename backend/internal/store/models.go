package store

import "time"

// TenantStatus enumerates tenant lifecycle states.
type TenantStatus string

const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
)

// Tenant is an isolated customer organization on the platform.
type Tenant struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Slug      string       `json:"slug"`
	Status    TenantStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// DevicePlatform is the RouterOS platform family.
type DevicePlatform string

const (
	PlatformRouterBOARD DevicePlatform = "routerboard"
	PlatformCHR         DevicePlatform = "chr"
	PlatformX86         DevicePlatform = "x86"
	PlatformUnknown     DevicePlatform = "unknown"
)

// DeviceRole classifies a device's role in the SD-WAN topology.
//
//	cpe      — 客户侧设备（Customer Premises Equipment）
//	backbone — SD-WAN 骨干节点 / POP（也是 RouterOS）
//	gateway  — 出口/网关节点（含海外出口）
type DeviceRole string

const (
	RoleCPE      DeviceRole = "cpe"
	RoleBackbone DeviceRole = "backbone"
	RoleGateway  DeviceRole = "gateway"
)

// TrustState models the device onboarding trust lifecycle.
//
//	untrusted -> discovered -> authenticated -> enrolled -> managed
type TrustState string

const (
	TrustUntrusted     TrustState = "untrusted"
	TrustDiscovered    TrustState = "discovered"
	TrustAuthenticated TrustState = "authenticated"
	TrustEnrolled      TrustState = "enrolled"
	TrustManaged       TrustState = "managed"
)

// InterfaceCapability describes a single interface and what it supports.
type InterfaceCapability struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`       // ether | sfp | sfp-plus | wlan | vlan | ...
	SpeedMbps int      `json:"speed_mbps"` // best-known link speed
	Features  []string `json:"features"`   // e.g. poe, l3hw, sfp+, wireless
}

// CapabilityMatrix is the normalized, structured capability set for a device.
// Config decisions are based on capabilities, NOT on a model string.
type CapabilityMatrix struct {
	RouterOSVersion   string                `json:"routeros_version"`
	Architecture      string                `json:"architecture"` // arm | arm64 | mipsbe | tile | x86_64
	BoardName         string                `json:"board_name"`
	Packages          []string              `json:"packages"`
	LicenseLevel      int                   `json:"license_level"`
	DeviceMode        string                `json:"device_mode"` // home | enterprise
	Interfaces        []InterfaceCapability `json:"interfaces"`
	SupportsBGP       bool                  `json:"supports_bgp"`
	SupportsOSPF      bool                  `json:"supports_ospf"`
	SupportsWireGuard bool                  `json:"supports_wireguard"`
	SupportsContainer bool                  `json:"supports_container"`
}

// Device is a managed RouterOS device.
type Device struct {
	ID           string            `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Name         string            `json:"name"`
	MgmtAddress  string            `json:"mgmt_address"`
	Role         DeviceRole        `json:"role"`
	Region       string            `json:"region,omitempty"` // POP/出口地域，如 cn-east、overseas-hk
	Platform     DevicePlatform    `json:"platform"`
	Model        string            `json:"model"`
	Serial       string            `json:"serial"`
	TrustState   TrustState        `json:"trust_state"`
	Capabilities *CapabilityMatrix `json:"capabilities,omitempty"`
	LastSeenAt   *time.Time        `json:"last_seen_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
