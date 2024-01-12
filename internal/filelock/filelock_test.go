package filelock_test

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	envTestHelperTrylockExclusive = "TEST_HELPER_TRYLOCK_EXCLUSIVE"
	envTestHelperTrylockShared    = "TEST_HELPER_TRYLOCK_SHARED"
)

func TestMain(m *testing.M) {
	if lockPath, ok := os.LookupEnv(envTestHelperTrylockExclusive); ok {
		testHelperTrylockExclusive(lockPath)
	} else if lockPath, ok := os.LookupEnv(envTestHelperTrylockShared); ok {
		testHelperTrylockShared(lockPath)
	} else {
		m.Run()
	}
}

func TestTrylockExclusive(t *testing.T) {
	// Create a lock file
	lockPath := touch(t)

	// Acquire a lock
	holderLock, err := filelock.New(lockPath, filelock.LockExclusive)
	require.NoError(t, err)
	require.NoError(t, holderLock.Trylock())

	// Run helper process
	runHelper(t, envTestHelperTrylockExclusive, lockPath)
}

func TestTrylockShared(t *testing.T) {
	// Create a lock file
	lockPath := touch(t)

	// Acquire a lock
	holderLock, err := filelock.New(lockPath, filelock.LockShared)
	require.NoError(t, err)
	require.NoError(t, holderLock.Trylock())

	// Run helper process
	runHelper(t, envTestHelperTrylockShared, lockPath)
}

func testHelperTrylockExclusive(lockPath string) {
	// Try to acquire a lock
	lock, err := filelock.New(lockPath, filelock.LockExclusive)
	if err != nil {
		panic(err)
	}

	err = lock.Trylock()
	if !errors.Is(err, filelock.ErrAlreadyLocked) {
		log.Panicf("expected a filelock.ErrAlreadyLocked error, got %v", err)
	}
}

func testHelperTrylockShared(lockPath string) {
	// Try to acquire a lock
	lock, err := filelock.New(lockPath, filelock.LockShared)
	if err != nil {
		panic(err)
	}

	if err = lock.Trylock(); err != nil {
		log.Panicf("expected no error, got %v", err)
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
