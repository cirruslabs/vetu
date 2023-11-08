package vmdirectory

import (
	"os"
	"path/filepath"
)

func (vmDir *VMDirectory) ExplicitlyPulled() bool {
	_, err := os.Stat(vmDir.explicitlyPulledFilePath())

	return err == nil
}

func (vmDir *VMDirectory) SetExplicitlyPulled(explicitlyPulled bool) error {
	if explicitlyPulled {
		file, err := os.Create(vmDir.explicitlyPulledFilePath())
		if err != nil {
			return err
		}
		defer file.Close()
	} else {
		if err := os.Remove(vmDir.explicitlyPulledFilePath()); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func (vmDir *VMDirectory) explicitlyPulledFilePath() string {
	return filepath.Join(vmDir.baseDir, ".explicitly-pulled")
}
