package afpacket

import (
	"encoding/binary"
	"golang.org/x/sys/unix"
	"os"
)

const bufferSizeBytes = 1 * 1024 * 1024

func RawSocket(ifIndex int) (*os.File, error) {
	rawSocketFD, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, 0)
	if err != nil {
		return nil, err
	}

	// Enable non-blocking mode, otherwise passt won't be able to
	// work with this socket properly
	if err := unix.SetNonblock(rawSocketFD, true); err != nil {
		return nil, err
	}

	// Increase buffer sizes, otherwise the networking will be incredibly slow
	if err := unix.SetsockoptUint64(rawSocketFD, unix.SOL_SOCKET, unix.SO_RCVBUF, bufferSizeBytes); err != nil {
		return nil, err
	}
	if err := unix.SetsockoptUint64(rawSocketFD, unix.SOL_SOCKET, unix.SO_SNDBUF, bufferSizeBytes); err != nil {
		return nil, err
	}

	if err := unix.Bind(rawSocketFD, &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ALL),
		Ifindex:  ifIndex,
	}); err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(rawSocketFD), "raw socket"), nil
}

// htons converts a value from host to network byte order.
func htons(hostshort uint16) uint16 {
	repr := make([]byte, 2)

	binary.BigEndian.PutUint16(repr, hostshort)

	return binary.LittleEndian.Uint16(repr)
}
