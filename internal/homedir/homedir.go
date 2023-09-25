package homedir

import (
	"os"
	"path/filepath"
)

func Path() (string, error) {
	override, ok := os.LookupEnv("VETU_HOME")
	if ok {
		return override, nil
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userHomeDir, ".vetu"), nil
}
