package vmconfig

import "github.com/projectcalico/libcalico-go/lib/net"

type VMConfig struct {
	Version    int     `json:"version,omitempty"`
	Arch       string  `json:"arch,omitempty"`
	Cmdline    string  `json:"cmdline,omitempty"`
	Disks      []Disk  `json:"disks,omitempty"`
	CPUCount   uint8   `json:"cpuCount,omitempty"`
	MemorySize uint64  `json:"memorySize,omitempty"`
	MACAddress net.MAC `json:"macAddress,omitempty"`
}

type Disk struct {
	Name string `json:"path"`
}
