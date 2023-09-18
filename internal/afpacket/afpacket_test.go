//nolint:testpackage // we need to test htons(), which is private
package afpacket

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHtons(t *testing.T) {
	require.Equal(t, 0x3412, htons(0x1234))
}
