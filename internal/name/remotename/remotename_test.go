package remotename_test

import (
	"github.com/cirruslabs/vetu/internal/name/remotename"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidPort(t *testing.T) {
	parsedRemoteName, err := remotename.NewFromString("localhost:8080/what/ever:latest")
	require.NoError(t, err)
	require.Equal(t, remotename.RemoteName{
		Registry:  "localhost:8080",
		Namespace: "what/ever",
		Tag:       "latest",
	}, parsedRemoteName)

	parsedRemoteName, err = remotename.NewFromString("127.0.0.1:8080/what/ever:latest")
	require.NoError(t, err)
	require.Equal(t, remotename.RemoteName{
		Registry:  "127.0.0.1:8080",
		Namespace: "what/ever",
		Tag:       "latest",
	}, parsedRemoteName)
}

func TestValidTag(t *testing.T) {
	parsedRemoteName, err := remotename.NewFromString("ghcr.io/cirruslabs/ubuntu:latest")
	require.NoError(t, err)
	require.Equal(t, remotename.RemoteName{
		Registry:  "ghcr.io",
		Namespace: "cirruslabs/ubuntu",
		Tag:       "latest",
	}, parsedRemoteName)
}

func TestValidDigest(t *testing.T) {
	parsedRemoteName, err := remotename.NewFromString("ghcr.io/cirruslabs/ubuntu@sha256:" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	require.NoError(t, err)
	require.Equal(t, remotename.RemoteName{
		Registry:  "ghcr.io",
		Namespace: "cirruslabs/ubuntu",
		Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}, parsedRemoteName)
}

func TestInvalidNoNamespace(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidNoTagOrDigest(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/what/ever")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidBothTagAndDigest(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/what/ever:latest@sha256:" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidPathTraversalInRegistry(t *testing.T) {
	_, err := remotename.NewFromString("/../ghcr.io/what/ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("../ghcr.io/what/ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("gh/../cr.io/what/ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io../what/ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidPathTraversalInNamespace(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/../what/ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/..what/ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what../ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/../ever:latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidPathTraversalInTag(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/what/ever:../latest")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/ever:late/..st")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/ever:late/../st")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/ever:late../st")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/ever:latest/..")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidPathTraversalInDigest(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/what/ever@sha256:" +
		"../0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/ever@sha256:" +
		"e3b0c44298fc1c149afbf4c8996fb/../7ae41e4649b934ca495991b7852b855")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)

	_, err = remotename.NewFromString("ghcr.io/what/ever@sha256:" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b../")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestInvalidNonSHA256Digest(t *testing.T) {
	_, err := remotename.NewFromString("ghcr.io/what/ever@md5:d41d8cd98f00b204e9800998ecf8427e")
	require.ErrorIs(t, err, remotename.ErrFailedToParse)
}

func TestNotARemoteName(t *testing.T) {
	_, err := remotename.NewFromString("ubuntu")
	require.ErrorIs(t, err, remotename.ErrNotARemoteName)

	_, err = remotename.NewFromString("ubuntu:latest")
	require.ErrorIs(t, err, remotename.ErrNotARemoteName)

	_, err = remotename.NewFromString("ubuntu@sha256:" +
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	require.ErrorIs(t, err, remotename.ErrNotARemoteName)
}
