//go:build !linux

package afpacket

import (
	"errors"
)

var ErrNotSupported = errors.New("raw sockets are not supported on this platform")

func RawSocket(ifIndex int) (int, error) {
	return 0, ErrNotSupported
}
