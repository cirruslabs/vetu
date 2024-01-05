// This packages provides a global locking facility
// useful for synchronization between different
// Vetu command invocations.
//
// It uses flock(2) advisory file locking on VETU_HOME
// directory under the hood.
package globallock

import (
	"context"
	"github.com/cirruslabs/vetu/internal/filelock"
	"github.com/cirruslabs/vetu/internal/homedir"
)

// With runs the callback cb under a global lock.
func With[T any](ctx context.Context, cb func() (T, error)) (T, error) {
	var result T

	homeDir, err := homedir.Path()
	if err != nil {
		return result, err
	}

	lock, err := filelock.New(homeDir)
	if err != nil {
		return result, err
	}

	if err := lock.Lock(ctx); err != nil {
		return result, err
	}

	result, err = cb()
	if err != nil {
		return result, err
	}

	if err := lock.Unlock(); err != nil {
		return result, err
	}

	return result, nil
}
