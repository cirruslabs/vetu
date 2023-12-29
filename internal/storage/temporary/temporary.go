package temporary

import (
	"errors"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/homedir"
	"github.com/cirruslabs/vetu/internal/sparseio"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

type Hook func(vmDir *vmdirectory.VMDirectory) error

func AtomicallyCopyThrough(srcDir string, dstDir string, hooks ...Hook) error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	// Create an intermediate directory that we'll later
	// os.Rename() into dstDir to achieve the atomicity
	intermediateDir := filepath.Join(baseDir, uuid.NewString())

	if err := os.Mkdir(intermediateDir, 0755); err != nil {
		return err
	}

	lock, err := filelock.New(intermediateDir)
	if err != nil {
		return err
	}
	if err := lock.Trylock(); err != nil {
		return err
	}

	// Copy the files from the source directory
	// to the intermediate directory
	dirEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		srcFile, err := os.Open(filepath.Join(srcDir, dirEntry.Name()))
		if err != nil {
			return err
		}

		srcFileInfo, err := srcFile.Stat()
		if err != nil {
			return err
		}

		dstFile, err := os.Create(filepath.Join(intermediateDir, dirEntry.Name()))
		if err != nil {
			return err
		}

		if err := dstFile.Truncate(srcFileInfo.Size()); err != nil {
			return err
		}

		if err := sparseio.Copy(dstFile, srcFile); err != nil {
			return err
		}

		if err := srcFile.Close(); err != nil {
			return err
		}

		if err := dstFile.Close(); err != nil {
			return err
		}
	}

	vmDir, err := vmdirectory.Load(intermediateDir)
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		if err := hook(vmDir); err != nil {
			return err
		}
	}

	return os.Rename(intermediateDir, dstDir)
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

func GC() error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		path := filepath.Join(baseDir, dirEntry.Name())

		lock, err := filelock.New(path)
		if err != nil {
			// It's quite possible that while iterating and removing the temporary directories,
			// some of the directories were already moved to their final destination, so ignore them
			if os.IsNotExist(err) {
				continue
			}

			return err
		}

		if err := lock.Trylock(); err != nil {
			// Avoid garbage collection if this directory is in use
			if errors.Is(err, filelock.ErrAlreadyLocked) {
				continue
			}

			return err
		}

		if err := os.RemoveAll(path); err != nil {
			return err
		}

		if err := lock.Unlock(); err != nil {
			return err
		}
	}

	return nil
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
