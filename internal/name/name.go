package name

import (
	"errors"
	"github.com/cirruslabs/vetu/internal/name/localname"
	"github.com/cirruslabs/vetu/internal/name/remotename"
)

type Name interface{}

func NewFromString(s string) (Name, error) {
	// Try parsing as remote name first
	remoteName, err := remotename.NewFromString(s)
	if err != nil {
		if errors.Is(err, remotename.ErrNotARemoteName) {
			// Fall back to parsing as local name
			return localname.NewFromString(s)
		}

		return nil, err
	}

	return remoteName, nil
}
