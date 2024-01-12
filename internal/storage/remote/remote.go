package remote

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/homedir"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/cirruslabs/vetu/internal/vmdirectory"
	"github.com/opencontainers/go-digest"
	"github.com/samber/lo"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Exists(name remotename.RemoteName) bool {
	path, err := PathForResolved(name)
	if err != nil {
		return false
	}

	_, err = os.Stat(path)

	return err == nil
}

func Link(digestedRemoteName remotename.RemoteName, taggedRemoteName remotename.RemoteName) error {
	oldname, err := PathForUnresolved(digestedRemoteName)
	if err != nil {
		return err
	}

	newname, err := PathForUnresolved(taggedRemoteName)
	if err != nil {
		return err
	}

	// Make sure that the old symbolic link does not exist (if any)
	_ = os.Remove(newname)

	return os.Symlink(oldname, newname)
}

func MoveIn(name remotename.RemoteName, digest digest.Digest, vmDir *vmdirectory.VMDirectory) error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	// Figure out the base path (without tag or digest) and make sure it exists
	basePath := filepath.Join(baseDir, name.Registry, name.Namespace)

	if err := os.MkdirAll(basePath, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	// Always create a digest directory containing the actual VM
	concretePath := filepath.Join(basePath, digest.String())

	if err := os.Rename(vmDir.Path(), concretePath); err != nil && !os.IsExist(err) {
		return err
	}

	// Symlink to the digest directory if tag is used
	if name.Tag != "" {
		tagPath := filepath.Join(basePath, name.Tag)

		// Make sure that the old symbolic link does not exist (if any)
		_ = os.Remove(tagPath)

		return os.Symlink(concretePath, tagPath)
	}

	return nil
}

func Open(name remotename.RemoteName) (*vmdirectory.VMDirectory, error) {
	path, err := PathForResolved(name)
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

	var result []lo.Tuple2[string, *vmdirectory.VMDirectory]

	if err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name, err := filepath.Rel(baseDir, filepath.Dir(path))
		if err != nil {
			return err
		}

		if d.Type() == os.ModeSymlink {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}

			vmDir, err := vmdirectory.Load(linkTarget)
			if err != nil {
				return err
			}

			result = append(result, lo.T2(name+":"+d.Name(), vmDir))
		} else if _, err := digest.Parse(d.Name()); err == nil {
			vmDir, err := vmdirectory.Load(path)
			if err != nil {
				return err
			}

			result = append(result, lo.T2(name+"@"+d.Name(), vmDir))
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func Delete(name remotename.RemoteName) error {
	path, err := PathForUnresolved(name)
	if err != nil {
		return err
	}

	// Figure out the base directory for this remote name
	path = filepath.Dir(path)

	var target string
	var method func(path string) error

	// Removal method depends on whether
	// we delete by tag or by digest
	if name.Tag != "" {
		target = filepath.Join(path, name.Tag)
		method = os.Remove
	} else if name.Digest != "" {
		target = filepath.Join(path, name.Digest.String())
		method = os.RemoveAll
	}

	_, err = os.Stat(target)
	if os.IsNotExist(err) {
		return fmt.Errorf("VM doesn't exist")
	}

	if err := method(target); err != nil {
		return err
	}

	return gc()
}

func RegistryLock(name remotename.RemoteName) (*filelock.FileLock, error) {
	baseDir, err := initialize()
	if err != nil {
		return nil, err
	}

	registryDir := filepath.Join(baseDir, name.Registry)

	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return nil, err
	}

	return filelock.New(registryDir, filelock.LockExclusive)
}

func PathForResolved(name remotename.RemoteName) (string, error) {
	path, err := PathForUnresolved(name)
	if err != nil {
		return "", err
	}

	// Path can be a symlink when using tags, so resolve it
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}

	return path, nil
}

func PathForUnresolved(name remotename.RemoteName) (string, error) {
	baseDir, err := initialize()
	if err != nil {
		return "", err
	}

	components := append([]string{baseDir, name.Registry}, strings.Split(name.Namespace, "/")...)

	if name.Tag != "" {
		components = append(components, name.Tag)
	}

	if name.Digest != "" {
		components = append(components, name.Digest.String())
	}

	return filepath.Join(components...), nil
}

func initialize() (string, error) {
	homeDir, err := homedir.Path()
	if err != nil {
		return "", err
	}

	baseDir := filepath.Join(homeDir, "cache", "OCIs")

	// Ensure that the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", err
	}

	return baseDir, nil
}

//nolint:gocognit // doesn't look complex yet
func gc() error {
	baseDir, err := initialize()
	if err != nil {
		return err
	}

	// Collect digest-based paths that are managed by us
	// (i.e. they are in ~/.vetu/cache/OCIs)
	managedPaths := map[string]struct{}{}

	// Collect paths to which the tag-based symbolic links point to
	anyPathToNumReferences := map[string]int{}

	if err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type() == os.ModeSymlink {
			// De-reference the symbolic link
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}

			// Perform garbage collection for tag-based images
			// with broken outgoing references
			_, err = os.Lstat(linkTarget)
			if err != nil {
				if os.IsNotExist(err) {
					if err := os.Remove(path); err != nil {
						return err
					}
				} else {
					return err
				}
			}

			// Count the outgoing reference if it's not broken
			anyPathToNumReferences[linkTarget] += 1
		} else if _, err := digest.Parse(d.Name()); err == nil {
			managedPaths[path] = struct{}{}
		}

		return nil
	}); err != nil {
		return err
	}

	for managedPath := range managedPaths {
		// Only garbage-collect paths that have no incoming references
		if anyPathToNumReferences[managedPath] != 0 {
			continue
		}

		// Only garbage-collect paths that were not pulled explicitly
		vmDir, err := vmdirectory.Load(managedPath)
		if err == nil && vmDir.ExplicitlyPulled() {
			continue
		}

		if err := os.RemoveAll(managedPath); err != nil {
			return err
		}
	}

	return nil
}
