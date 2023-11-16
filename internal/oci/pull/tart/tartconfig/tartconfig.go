package tartconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/projectcalico/libcalico-go/lib/net"
)

var ErrFailedToParse = errors.New("failed to parse Tart VM configuration")

const SupportedVersion = 1

type TartConfig struct {
	Version    int     `json:"version"`
	OS         string  `json:"os"`
	Arch       string  `json:"arch"`
	CPUCount   uint8   `json:"cpuCount,omitempty"`
	MemorySize uint64  `json:"memorySize,omitempty"`
	MACAddress net.MAC `json:"macAddress,omitempty"`
}

func NewFromJSON(tartConfigBytes []byte) (*TartConfig, error) {
	var tartConfig TartConfig

	if err := json.Unmarshal(tartConfigBytes, &tartConfig); err != nil {
		return nil, err
	}

	if tartConfig.Version != SupportedVersion {
		return nil, fmt.Errorf("%w: only version %d is currently supported, got %d",
			ErrFailedToParse, SupportedVersion, tartConfig.Version)
	}

	if tartConfig.Arch == "" {
		return nil, fmt.Errorf("%w: architecture field cannot empty", ErrFailedToParse)
	}

	return &tartConfig, nil
}
