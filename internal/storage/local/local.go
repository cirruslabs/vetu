package local

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/homedir"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/samber/lo"
	"os"
	"path/filepath"
)

func Exists(name localname.LocalName) bool {
	path, err := PathFor(name)
	if err != nil {
		return false
	}

	_, err = os.Stat(path)

	return err == nil
}

func MoveIn(name localname.LocalName, vmDir *vmdirectory.VMDirectory) error {
	path, err := PathFor(name)
	if err != nil {
		return err
	}

	return os.Rename(vmDir.Path(), path)
}

func Open(name localname.LocalName) (*vmdirectory.VMDirectory, error) {
	path, err := PathFor(name)
	if err != nil {
		return nil, err
	}

	return vmdirectory.Load(path)
}

func List() ([]lo.Tuple2[string, *vmdirectory.VMDirectory], error) {
	baseDir, err := initialize()
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var result []lo.Tuple2[string, *vmdirectory.VMDirectory]

	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}

		localName, err := localname.NewFromString(dirEntry.Name())
		if err != nil {
			return nil, err
		}

		vmDir, err := Open(localName)
		if err != nil {
			return nil, err
		}

		result = append(result, lo.T2(dirEntry.Name(), vmDir))
	}

	return result, nil
}

func Delete(name localname.LocalName) error {
	path, err := PathFor(name)
	if err != nil {
		return err
	}

	vmDir, err := Open(name)
	if err != nil {
		return fmt.Errorf("cannot remove VM %s: %v", name.String(), err)
	}

	lock, err := vmDir.FileLock()
	if err != nil {
		return fmt.Errorf("cannot remove VM %s: %v", name.String(), err)
	}

	if err := lock.Trylock(); err != nil {
		return fmt.Errorf("cannot remove VM %s: %v", name.String(), err)
	}
	defer func() {
		_ = lock.Unlock()
	}()

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("cannot remove VM %s: %v", name.String(), err)
	}

	return nil
}

func PathFor(name localname.LocalName) (string, error) {
	baseDir, err := initialize()
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, string(name)), nil
}

func initialize() (string, error) {
	homeDir, err := homedir.Path()
	if err != nil {
		return "", err
	}

	baseDir := filepath.Join(homeDir, "vms")

	// Ensure that the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", err
	}

	return baseDir, nil
}
