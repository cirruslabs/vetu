package pidlock_test

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/pidlock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	envTestHelperTrylock = "TEST_HELPER_TRYLOCK"
	envTestHelperPid     = "TEST_HELPER_PID"
)

func TestMain(m *testing.M) {
	if lockPath, ok := os.LookupEnv(envTestHelperTrylock); ok {
		testHelperTrylock(lockPath)
	} else if lockPath, ok := os.LookupEnv(envTestHelperPid); ok {
		testHelperPid(lockPath)
	} else {
		os.Exit(m.Run())
	}
}

func TestTrylock(t *testing.T) {
	// Create a lock file
	lockPath := touch(t)

	// Acquire a lock
	holderLock, err := pidlock.New(lockPath)
	require.NoError(t, err)
	require.NoError(t, holderLock.Trylock())

	// Run helper process
	runHelper(t, envTestHelperTrylock, lockPath)
}

func TestPid(t *testing.T) {
	// Create a lock file
	lockPath := touch(t)

	// Acquire a lock
	holderLock, err := pidlock.New(lockPath)
	require.NoError(t, err)
	require.NoError(t, holderLock.Trylock())

	// Run helper process
	runHelper(t, envTestHelperPid, lockPath)
}

func testHelperTrylock(lockPath string) {
	// Try to acquire a lock
	lock, err := pidlock.New(lockPath)
	if err != nil {
		panic(err)
	}

	err = lock.Trylock()
	if !errors.Is(err, pidlock.ErrAlreadyLocked) {
		log.Panicf("expected a filelock.ErrAlreadyLocked error, got %v", err)
	}
}

func testHelperPid(lockPath string) {
	// Retrieve the lock file's attached PID
	lock, err := pidlock.New(lockPath)
	if err != nil {
		panic(err)
	}

	pid, err := lock.Pid()
	if err != nil {
		panic(err)
	}

	ppid := unix.Getppid()

	// Ensure that it's the same PID as that of a parent process
	if int(pid) != ppid {
		log.Panicf("expected lock PID to be %d, got %d instead", ppid, pid)
	}
}

func touch(t *testing.T) string {
	path := filepath.Join(t.TempDir(), uuid.NewString())

	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	return path
}

func runHelper(t *testing.T, helperIdent string, lockPath string) {
	testExecutable, err := os.Executable()
	require.NoError(t, err)

	cmd := exec.Command(testExecutable)

	// Do not hide the output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// A simple one-shot IPC through environment variables
	cmd.Env = []string{fmt.Sprintf("%s=%s", helperIdent, lockPath)}

	require.NoError(t, cmd.Run())
}
