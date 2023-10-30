package vmconfig_test

import (
	"github.com/cirruslabs/vetu/internal/vmconfig"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestUnsupportedVersion(t *testing.T) {
	vmConfigBytes, err := os.ReadFile(filepath.Join("testdata", "unsupported-version.json"))
	require.NoError(t, err)

	_, err = vmconfig.NewFromJSON(vmConfigBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only version 1 is currently supported, got 9000")
}

func TestEmptyArchitectureField(t *testing.T) {
	vmConfigBytes, err := os.ReadFile(filepath.Join("testdata", "empty-architecture-field.json"))
	require.NoError(t, err)

	_, err = vmconfig.NewFromJSON(vmConfigBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "architecture field cannot empty")
}

func TestEmptyDiskName(t *testing.T) {
	vmConfigBytes, err := os.ReadFile(filepath.Join("testdata", "empty-disk-name.json"))
	require.NoError(t, err)

	_, err = vmconfig.NewFromJSON(vmConfigBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is empty")
}

func TestRestrictedDiskName(t *testing.T) {
	vmConfigBytes, err := os.ReadFile(filepath.Join("testdata", "restricted-disk-name.json"))
	require.NoError(t, err)

	_, err = vmconfig.NewFromJSON(vmConfigBytes)
	require.Error(t, err)
	require.Contains(t, err.Error(), "contains restricted characters")
}
