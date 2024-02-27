package zerocopy

import (
	"golang.org/x/sys/unix"
)

func Clone(destFd int, srcFd int) error {
	return unix.IoctlFileClone(destFd, srcFd)
}
