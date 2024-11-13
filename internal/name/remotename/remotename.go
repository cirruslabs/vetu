package remotename

import (
	"errors"
	"fmt"
	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	"strings"
)

var (
	ErrFailedToParse  = errors.New("failed to parse remote name")
	ErrNotARemoteName = errors.New("not a remote name")
)

type RemoteName struct {
	Registry  string
	Namespace string
	Tag       string
	Digest    digest.Digest
}

func NewFromString(s string) (RemoteName, error) {
	named, err := reference.ParseNamed(s)
	if err != nil {
		if errors.Is(err, reference.ErrNameNotCanonical) {
			return RemoteName{}, fmt.Errorf("%w: %v", ErrNotARemoteName, err)
		}

		return RemoteName{}, fmt.Errorf("%w: %v", ErrFailedToParse, err)
	}

	remoteName := RemoteName{
		Registry:  reference.Domain(named),
		Namespace: reference.Path(named),
	}

	tagged, isTagged := named.(reference.Tagged)
	digested, isDigested := named.(reference.Digested)

	switch {
	case isTagged && isDigested:
		return RemoteName{}, fmt.Errorf("%w: remote name cannot have both a tag and a digest",
			ErrFailedToParse)
	case !isTagged && !isDigested:
		remoteName.Tag = "latest"
	case isTagged:
		remoteName.Tag = tagged.Tag()
	case isDigested:
		remoteName.Digest = digested.Digest()
	}

	return remoteName, nil
}

func (name RemoteName) String() string {
	result := strings.Join([]string{name.Registry, name.Namespace}, "/")

	if name.Tag != "" {
		result += ":" + name.Tag
	}

	if name.Digest != "" {
		result += "@" + name.Digest.String()
	}

	return result
}
