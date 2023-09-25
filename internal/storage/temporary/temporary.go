package temporary

import (
	"github.com/cirruslabs/vetu/internal/homedir"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/google/uuid"
	cp "github.com/otiai10/copy"
	"os"
	"path/filepath"
)

func AtomicallyCopyThrough(src string, dest string) error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	copyThroughPath := filepath.Join(baseDir, uuid.NewString())

	if err := cp.Copy(src, copyThroughPath); err != nil {
		return err
	}

	return os.Rename(copyThroughPath, dest)
}

func Create() (*vmdirectory.VMDirectory, error) {
	baseDir, err := initialize()
	if err != nil {
		return nil, err
	}

	vmDirPath := filepath.Join(baseDir, uuid.NewString())

	if err := os.MkdirAll(vmDirPath, 0755); err != nil {
		return nil, err
	}

	return vmdirectory.Initialize(vmDirPath)
}

func initialize() (string, error) {
	homeDir, err := homedir.Path()
	if err != nil {
		return "", err
	}

	baseDir := filepath.Join(homeDir, "tmp")

	// Ensure that the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", err
	}

	return baseDir, nil
}
