package vmdirectory_test

import (
	"github.com/cirruslabs/vetu/internal/storage/temporary"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExplicitlyPulled(t *testing.T) {
	vmDir, err := temporary.Create()
	require.NoError(t, err)

	// By default, the VM directory shouldn't be marked as explicitly pulled
	require.False(t, vmDir.ExplicitlyPulled())

	// Mark the VM directory as explicitly pulled and ensure that it reports as so
	require.NoError(t, vmDir.SetExplicitlyPulled(true))
	require.True(t, vmDir.ExplicitlyPulled())

	// Un-mark the VM directory as explicitly pulled and ensure that it reports as so
	require.NoError(t, vmDir.SetExplicitlyPulled(false))
	require.False(t, vmDir.ExplicitlyPulled())
}
