package randommac

import (
	cryptorand "crypto/rand"
	"net"
)

func UnicastAndLocallyAdministered() (net.HardwareAddr, error) {
	mac := make([]byte, 6)

	_, err := cryptorand.Read(mac)
	if err != nil {
		return nil, err
	}

	// Make MAC-address unicast by turning
	// the first LSB bit to zero
	mac[0] &= 0xFE

	// Make MAC-address locally administered by turning
	// the second LSB bit to one
	mac[0] |= 0x02

	return mac, nil
}
