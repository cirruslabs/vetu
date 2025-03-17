package rename

import (
	"golang.org/x/sys/unix"
)

func Rename(oldpath string, newpath string) error {
	err := unix.Renameat2(unix.AT_FDCWD, oldpath, unix.AT_FDCWD, newpath, unix.RENAME_EXCHANGE)
	if err != nil {
		return err
	}

	return nil
}
