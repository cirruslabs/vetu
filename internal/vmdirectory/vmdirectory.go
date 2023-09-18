package vmdirectory

import (
	"encoding/json"
	"fmt"
	"github.com/cirruslabs/nutmeg/internal/vmconfig"
	"os"
	"path/filepath"
	"runtime"
)

type VMDirectory struct {
	baseDir  string
	vmConfig vmconfig.VMConfig
}

func Initialize(path string) (*VMDirectory, error) {
	vmDir := &VMDirectory{
		baseDir: path,
	}

	if err := vmDir.SetConfig(&vmconfig.VMConfig{
		Version: 1,
		Arch:    runtime.GOARCH,
	}); err != nil {
		return nil, err
	}

	return vmDir, nil
}

func Load(path string) (*VMDirectory, error) {
	vmDir := &VMDirectory{
		baseDir: path,
	}

	vmConfigJSONBytes, err := os.ReadFile(vmDir.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read VM's config: %v", err)
	}

	if err := json.Unmarshal(vmConfigJSONBytes, &vmDir.vmConfig); err != nil {
		return nil, fmt.Errorf("failed to parse VM's config: %v", err)
	}

	return vmDir, nil
}

func (vmDir *VMDirectory) Path() string {
	return vmDir.baseDir
}

func (vmDir *VMDirectory) ConfigPath() string {
	return filepath.Join(vmDir.baseDir, "config.json")
}

func (vmDir *VMDirectory) KernelPath() string {
	return filepath.Join(vmDir.baseDir, "kernel")
}

func (vmDir *VMDirectory) InitramfsPath() string {
	return filepath.Join(vmDir.baseDir, "initramfs")
}

func (vmDir *VMDirectory) Config() vmconfig.VMConfig {
	return vmDir.vmConfig
}

func (vmDir *VMDirectory) SetConfig(vmConfig *vmconfig.VMConfig) error {
	vmConfigJSONBytes, err := json.Marshal(vmConfig)
	if err != nil {
		return err
	}

	if err := os.WriteFile(vmDir.ConfigPath(), vmConfigJSONBytes, 0600); err != nil {
		return err
	}

	vmDir.vmConfig = *vmConfig

	return nil
}
