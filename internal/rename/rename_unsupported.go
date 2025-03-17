//go:build !linux && !darwin

package rename

import "fmt"

func Rename(oldDir string, newDir string) error {
	return fmt.Errorf("atomic rename of directories is not supported on this platform")
}
