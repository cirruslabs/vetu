package tuntap

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

func CreateTAP(name string, additionalFlags uint16) (string, *os.File, error) {
	result, err := os.OpenFile("/dev/net/tun", unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open /dev/net/tun: %v", err)
	}

	ifreq, err := unix.NewIfreq(name)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create ifreq: %v", err)
	}

	ifreq.SetUint16(unix.IFF_TAP | unix.IFF_NO_PI | additionalFlags)

	if err := unix.IoctlIfreq(int(result.Fd()), unix.TUNSETIFF, ifreq); err != nil {
		return "", nil, fmt.Errorf("failed to TUNSETIFF: %v", err)
	}

	if err := unix.IoctlIfreq(int(result.Fd()), unix.TUNGETIFF, ifreq); err != nil {
		return "", nil, fmt.Errorf("failed to TUNGETIFF: %v", err)
	}

	return ifreq.Name(), result, nil
}
