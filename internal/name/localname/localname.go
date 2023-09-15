package localname

type LocalName string

func NewFromString(s string) (LocalName, error) {
	return LocalName(s), nil
}

func (localName LocalName) String() string {
	return string(localName)
}
