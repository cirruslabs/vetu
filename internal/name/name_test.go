package name_test

import (
	"github.com/cirruslabs/vetu/internal/name"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocalName(t *testing.T) {
	abstractName, err := name.NewFromString("local")
	require.NoError(t, err)
	require.IsType(t, localname.LocalName(""), abstractName)
}

func TestRemoteName(t *testing.T) {
	abstractName, err := name.NewFromString("ghcr.io/cirruslabs/ubuntu:latest")
	require.NoError(t, err)
	require.IsType(t, remotename.RemoteName{}, abstractName)
}

func TestInvalid(t *testing.T) {
	_, err := name.NewFromString("%")
	require.Error(t, err)
}
