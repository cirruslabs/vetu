//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/name"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/storage/local"
	"github.com/cirruslabs/vetu/internal/storage/remote"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestPushPull ensures that we can push and pull VMs,
// and that the pushed VM contents = pulled VM contents.
func TestPushPull(t *testing.T) {
	// Instantiate a container registry
	registry := containerRegistry(t)

	// Create a dummy kernel file that we'll use for creating a VM
	tempDir := t.TempDir()

	kernelPath := filepath.Join(tempDir, "kernel")
	fillFileWithRandomBytes(t, kernelPath, 64*humanize.MByte)

	// Create a dummy disk file that we'll use for creating a VM
	diskPath := filepath.Join(tempDir, "disk.img")
	fillFileWithRandomBytes(t, diskPath, 1*humanize.GByte)

	// Create a VM
	vmName := fmt.Sprintf("integration-test-push-pull-%s", uuid.NewString())
	vmNameRemote := fmt.Sprintf("%s/integration-test-push-pull:%s", registry, uuid.NewString())

	_, _, err := vetu("create", "--kernel", kernelPath, "--disk", diskPath, vmName)
	require.NoError(t, err)

	originalVMFilesDigests := calculateVMFilesDigests(t, localname.LocalName(vmName))

	// Push the VM to a registry
	_, _, err = vetu("push", "--insecure", vmName, vmNameRemote)
	require.NoError(t, err)

	// Pull the VM from the registry and make sure it
	// has the same contents as the VM we've pushed
	_, _, err = vetu("pull", "--insecure", vmNameRemote)
	require.NoError(t, err)

	remoteName, err := remotename.NewFromString(vmNameRemote)
	require.NoError(t, err)

	pulledVMFilesDigests := calculateVMFilesDigests(t, remoteName)
	require.Equal(t, originalVMFilesDigests, pulledVMFilesDigests)
}

func fillFileWithRandomBytes(t *testing.T, path string, sizeBytes int64) {
	t.Helper()

	diskFile, err := os.Create(path)
	require.NoError(t, err)

	devUrandomFile, err := os.Open("/dev/urandom")
	require.NoError(t, err)

	_, err = io.Copy(diskFile, io.LimitReader(devUrandomFile, sizeBytes))
	require.NoError(t, err)

	require.NoError(t, devUrandomFile.Close())

	require.NoError(t, diskFile.Close())
}

func calculateVMFilesDigests(t *testing.T, name name.Name) map[string]digest.Digest {
	var vmDir *vmdirectory.VMDirectory
	var err error

	switch typedName := name.(type) {
	case localname.LocalName:
		vmDir, err = local.Open(typedName)
		require.NoError(t, err)
	case remotename.RemoteName:
		vmDir, err = remote.Open(typedName)
		require.NoError(t, err)
	default:
		t.Errorf("unsupported name type: %T", name)
	}

	dirEntries, err := os.ReadDir(vmDir.Path())
	require.NoError(t, err)

	return lo.Associate(dirEntries, func(dirEntry os.DirEntry) (string, digest.Digest) {
		return dirEntry.Name(), calculateFileDigest(t, filepath.Join(vmDir.Path(), dirEntry.Name()))
	})
}

func calculateFileDigest(t *testing.T, path string) digest.Digest {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)

	digest, err := digest.FromReader(file)
	require.NoError(t, err)

	require.NoError(t, file.Close())

	return digest
}

func containerRegistry(t *testing.T) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "registry:2",
			ExposedPorts: []string{"5000/tcp"},
			WaitingFor:   wait.ForHTTP("/v2/"),
		},
		Started: true,
	})
	require.NoError(t, err)

	endpoint, err := container.Endpoint(context.Background(), "")
	require.NoError(t, err)

	return endpoint
}
