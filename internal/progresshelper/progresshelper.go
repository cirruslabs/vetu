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

	applyNonTerminalOpts(progressBar)

	return progressBar
}

func DefaultBytesSilent(maxBytes int64) *progressbar.ProgressBar {
	progressBar := progressbar.DefaultBytesSilent(maxBytes)

	applyNonTerminalOpts(progressBar)

	return progressBar
}

func applyNonTerminalOpts(progressBar *progressbar.ProgressBar) {
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}

	progressbar.OptionThrottle(terminalThrottleDuration)(progressBar)
}
