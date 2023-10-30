package simplename

import "errors"

var (
	ErrIsEmpty                      = errors.New("is empty")
	ErrContainsRestrictedCharacters = errors.New("contains restricted characters, " +
		"please only use [A-Za-z0-9-_]")
)

func Validate(s string) error {
	if s == "" {
		return ErrIsEmpty
	}

	minIdx := 0
	maxIdx := len(s) - 1

	for idx, ch := range s {
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

		weAreInTheMiddle := (idx != minIdx) && (idx != maxIdx)
		if ch == '.' && weAreInTheMiddle {
			continue
		}

		return ErrContainsRestrictedCharacters
	}

	return nil
}
