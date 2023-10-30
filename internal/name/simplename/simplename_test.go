package simplename_test

import (
	"github.com/cirruslabs/vetu/internal/name/simplename"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimplename(t *testing.T) {
	require.NoError(t, simplename.Validate("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"))
	require.NoError(t, simplename.Validate("vm-1"))
	require.NoError(t, simplename.Validate("vm_2"))
	require.NoError(t, simplename.Validate("disk.img"))
	require.NoError(t, simplename.Validate("weirdly-named..disk"))

	require.Error(t, simplename.Validate(""), "simple name cannot be empty")
	require.Error(t, simplename.Validate("..whatever"), "simple name cannot contain dots at the start")
	require.Error(t, simplename.Validate("whatever.."), "simple name cannot contain dots at the end")
	require.Error(t, simplename.Validate("vm:latest"), "simple name cannot contain colons")
	require.Error(t, simplename.Validate("vm%"), "simple name cannot contain special characters")
	require.Error(t, simplename.Validate("😐"), "simple name cannot contain non-ASCII characters")
}
