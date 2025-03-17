package rename

import (
	"golang.org/x/sys/unix"
)

func Rename(oldpath string, newpath string) error {
	err := unix.RenameatxNp(unix.AT_FDCWD, oldpath, unix.AT_FDCWD, newpath, unix.RENAME_SWAP)
	if err != nil {
		return err
	}

	return nil
}
