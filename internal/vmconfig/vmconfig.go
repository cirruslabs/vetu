package vmconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/name/simplename"
	"github.com/projectcalico/libcalico-go/lib/net"
	"runtime"
)

var ErrFailedToParse = errors.New("failed to parse VM configuration")

const CurrentVersion = 1

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
	Name string `json:"name"`
}

func New() *VMConfig {
	return &VMConfig{
		Version: CurrentVersion,
		Arch:    runtime.GOARCH,
	}
}

func NewFromJSON(vmConfigBytes []byte) (*VMConfig, error) {
	var vmConfig VMConfig

	if err := json.Unmarshal(vmConfigBytes, &vmConfig); err != nil {
		return nil, err
	}

	if vmConfig.Version != CurrentVersion {
		return nil, fmt.Errorf("%w: only version %d is currently supported, got %d",
			ErrFailedToParse, CurrentVersion, vmConfig.Version)
	}

	if vmConfig.Arch == "" {
		return nil, fmt.Errorf("%w: architecture field cannot empty", ErrFailedToParse)
	}

	for _, disk := range vmConfig.Disks {
		if err := simplename.Validate(disk.Name); err != nil {
			return nil, fmt.Errorf("%w: disk name %q %v", ErrFailedToParse, disk.Name, err)
		}
	}

	return &vmConfig, nil
}
