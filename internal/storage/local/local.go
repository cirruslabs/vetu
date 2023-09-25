package local

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/homedir"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"os"
	"path/filepath"
)

func Exists(name localname.LocalName) bool {
	baseDir, err := initialize()
	if err != nil {
		return false
	}

	_, err = os.Stat(filepath.Join(baseDir, name.String()))

	return err == nil
}

func MoveIn(name localname.LocalName, vmDir *vmdirectory.VMDirectory) error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	if err := os.Rename(vmDir.Path(), filepath.Join(baseDir, string(name))); err != nil {
		return err
	}

	return nil
}

func Open(name localname.LocalName) (*vmdirectory.VMDirectory, error) {
	baseDir, err := initialize()
	if err != nil {
		return nil, err
	}

	return vmdirectory.Load(filepath.Join(baseDir, string(name)))
}

func List() ([]*vmdirectory.VMDirectory, error) {
	baseDir, err := initialize()
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var result []*vmdirectory.VMDirectory

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

		result = append(result, vmDir)
	}

	return result, nil
}

func Delete(name localname.LocalName) error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	vmDir := filepath.Join(baseDir, name.String())

	_, err = os.Stat(vmDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("cannot remove VM %s as it doesn't exist", name.String())
	}

	return os.RemoveAll(vmDir)
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
