// Package routeros models RouterOS device facts and derives a normalized
// capability matrix. Per requirement #5, config decisions are driven by
// detected capabilities, NOT by a hardcoded model string.
package routeros

// SystemResource mirrors `/system/resource print` fields of interest.
type SystemResource struct {
	BoardName        string `json:"board-name"`
	Architecture     string `json:"architecture-name"`
	Version          string `json:"version"`
	CPU              string `json:"cpu"`
	CPUCount         string `json:"cpu-count"`
	TotalMemoryBytes int64  `json:"total-memory"`
	Platform         string `json:"platform"` // usually "MikroTik"
}

// RouterboardInfo mirrors `/system/routerboard print`.
// On CHR / x86 installs, Routerboard is false.
type RouterboardInfo struct {
	Routerboard     bool   `json:"routerboard"`
	Model           string `json:"model"`
	SerialNumber    string `json:"serial-number"`
	CurrentFirmware string `json:"current-firmware"`
	UpgradeFirmware string `json:"upgrade-firmware"`
	FirmwareType    string `json:"firmware-type"`
}

// Package mirrors an entry of `/system/package print`.
type Package struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Disabled bool   `json:"disabled"`
}

// License mirrors `/system/license print` (relevant for CHR).
type License struct {
	Level  int    `json:"nlevel"`
	Level6 string `json:"level"` // textual level for RouterBOARD (e.g. "4", "6")
}

// DeviceMode mirrors `/system/device-mode print` (ROS 7.x).
type DeviceMode struct {
	Mode string `json:"mode"` // home | enterprise
}

// Interface mirrors a relevant subset of `/interface print`.
type Interface struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // ether | sfp | wlan | vlan | bridge | wg | ...
	Running bool   `json:"running"`
	MTU     int    `json:"mtu"`
}

// DeviceFacts is the aggregate of raw facts collected from a device.
type DeviceFacts struct {
	Resource    SystemResource
	Routerboard RouterboardInfo
	Packages    []Package
	License     License
	DeviceMode  DeviceMode
	Interfaces  []Interface
}
