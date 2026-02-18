//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const vetuBinaryName = "vetu"

func TestCreateDelete(t *testing.T) {
	// Create a dummy kernel file that we'll use for creating a VM
	kernelPath := filepath.Join(t.TempDir(), "kernel")
	require.NoError(t, os.WriteFile(kernelPath, []byte(""), 0600))

	vmName := fmt.Sprintf("integration-test-create-delete-%s", uuid.NewString())

	// Create a VM
	_, _, err := vetu("create", "--kernel", kernelPath, vmName)
	require.NoError(t, err)

	// Make sure the VM exists in "vetu list -q" output
	stdout, _, err := vetu("list", "-q")
	require.NoError(t, err)
	require.Contains(t, stdout, vmName)

	// Delete the VM
	stdout, _, err = vetu("delete", vmName)

	// Make sure the VM does not exist in "vetu list -q" output anymore
	stdout, _, err = vetu("list", "-q")
	require.NoError(t, err)
	require.NotContains(t, stdout, vmName)
}

func TestClone(t *testing.T) {
	// Create a dummy kernel file that we'll use for creating a VM
	kernelPath := filepath.Join(t.TempDir(), "kernel")
	require.NoError(t, os.WriteFile(kernelPath, []byte(""), 0600))

	firstVMName := fmt.Sprintf("integration-test-clone-%s-first", uuid.NewString())
	secondVMName := fmt.Sprintf("integration-test-clone-%s-second", uuid.NewString())

	// Create a VM
	_, _, err := vetu("create", "--kernel", kernelPath, firstVMName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, err = vetu("delete", firstVMName)
	})

	// Clone the VM
	_, _, err = vetu("clone", firstVMName, secondVMName)
	require.NoError(t, err)

	// Make sure that both VMs exist in the "vetu list -q" output
	stdout, _, err := vetu("list", "-q")
	require.NoError(t, err)
	require.Contains(t, stdout, firstVMName)
	require.Contains(t, stdout, secondVMName)
}

// TestRunAndSSH ensures that "tart run" correctly starts VMs and
// that we can connect to these VMs over SSH and issue commands.
func TestRunAndSSH(t *testing.T) {
	vmName := fmt.Sprintf("integration-test-run-and-ssh-%s", uuid.NewString())

	// Instantiate a VM with admin:admin SSH access
	_, _, err := vetu("clone", "ghcr.io/cirruslabs/ubuntu-runner-amd64:latest", vmName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, _ = vetu("delete", vmName)
	})

	go func() {
		// Wait for the VMs IP address
		stdout, _, err := vetu("ip", "--wait", "600", vmName)
		require.NoError(t, err)

		ip := strings.TrimSpace(stdout)

		// Connect to the VM over SSH and shutdown it
		err = retry.Do(func() error {
			return sshCommand(ip, "admin", "admin", "sudo halt -p")
		}, retry.Attempts(0), retry.Delay(time.Second), retry.DelayType(retry.FixedDelay))
		require.NoError(t, err)
	}()

	// Run the VM until it is shutdown by our goroutine
	_, _, err = vetu("run", vmName)
	require.NoError(t, err)
}

func vetu(args ...string) (string, string, error) {
	cmd := exec.Command(vetuBinaryName, args...)

	// Capture Vetu's output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

func sshCommand(ip string, username string, password string, command string) error {
	addr := ip + ":22"

	fmt.Printf("connecting via SSH to %s...\n", addr)

	dialer := net.Dialer{
		Timeout: time.Second,
	}

	netConn, err := dialer.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("failed to dial %s: %v\n", addr, err)

		return err
	}

	fmt.Printf("successfully dialed %s, performing SSH handshake...\n", addr)

	sshConfig := &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout: time.Second,
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		return err
	}

	sshClient := ssh.NewClient(sshConn, chans, reqs)

	sshSession, err := sshClient.NewSession()
	if err != nil {
		return err
	}

	fmt.Printf("successfully opened SSH session on %s\n", addr)

	return sshSession.Run(command)
}
