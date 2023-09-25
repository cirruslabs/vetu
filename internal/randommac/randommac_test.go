package randommac_test

import (
	"github.com/cirruslabs/vetu/internal/randommac"
	"github.com/klauspost/oui"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUnicastAndLocallyAdministered(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		hwAddr, err := randommac.UnicastAndLocallyAdministered()
		require.NoError(t, err)

		ouiHwAddr, err := oui.ParseMac(hwAddr.String())
		require.NoError(t, err)
		require.False(t, ouiHwAddr.Multicast())
		require.True(t, ouiHwAddr.Local())
	}
}
