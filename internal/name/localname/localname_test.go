package localname_test

import (
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocalname(t *testing.T) {
	_, err := localname.NewFromString("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
	require.NoError(t, err)
	_, err = localname.NewFromString("vm-1")
	require.NoError(t, err)
	_, err = localname.NewFromString("vm_2")
	require.NoError(t, err)

	_, err = localname.NewFromString("vm.local")
	require.Error(t, err, "local name cannot contain dots")
	_, err = localname.NewFromString("vm:latest")
	require.Error(t, err, "local name cannot contain colons")
	_, err = localname.NewFromString("vm%")
	require.Error(t, err, "local name cannot contain special characters")
	_, err = localname.NewFromString("😐")
	require.Error(t, err, "local name cannot contain non-ASCII characters")
}
