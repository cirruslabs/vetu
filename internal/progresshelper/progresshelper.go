package progresshelper

import (
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

const terminalThrottleDuration = 5 * time.Second

func DefaultBytes(maxBytes int64, description ...string) *progressbar.ProgressBar {
	progressBar := progressbar.DefaultBytes(maxBytes, description...)

	if !term.IsTerminal(int(os.Stderr.Fd())) {
		progressbar.OptionThrottle(terminalThrottleDuration)(progressBar)
	}

	return progressBar
}
