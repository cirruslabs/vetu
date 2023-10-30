package localname

import (
	"fmt"
	"github.com/cirruslabs/vetu/internal/name/simplename"
)

type LocalName string

func NewFromString(s string) (LocalName, error) {
	if err := simplename.Validate(s); err != nil {
		return "", fmt.Errorf("local name %w", err)
	}

	return LocalName(s), nil
}

func (localName LocalName) String() string {
	return string(localName)
}
