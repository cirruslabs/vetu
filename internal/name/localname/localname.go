package localname

import "errors"

var ErrRestrictedCharaters = errors.New("local name contains restricted characters, " +
	"please only use [A-Za-z0-9-_]")

type LocalName string

func NewFromString(s string) (LocalName, error) {
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			continue
		}

		if ch >= 'A' && ch <= 'Z' {
			continue
		}

		if ch >= '0' && ch <= '9' {
			continue
		}

		if ch == '-' || ch == '_' {
			continue
		}

		return "", ErrRestrictedCharaters
	}

	return LocalName(s), nil
}

func (localName LocalName) String() string {
	return string(localName)
}
