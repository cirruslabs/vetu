package filelock

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"syscall"
	"time"
)

var ErrAlreadyLocked = errors.New("already locked")

type FileLock struct {
	fd int
}

func New(path string) (*FileLock, error) {
	fd, err := unix.Open(path, unix.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return &FileLock{
		fd: fd,
	}, nil
}

func (fl *FileLock) Trylock() error {
	return fl.lockWrapper(unix.LOCK_EX | unix.LOCK_NB)
}

func (fl *FileLock) Lock(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := fl.Trylock(); !errors.Is(err, ErrAlreadyLocked) {
				return err
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (fl *FileLock) Unlock() error {
	return fl.lockWrapper(unix.LOCK_UN)
}

func (fl *FileLock) Close() error {
	return unix.Close(fl.fd)
}

func (fl *FileLock) lockWrapper(how int) error {
	if err := unix.Flock(fl.fd, how); err != nil {
		if errors.Is(err, syscall.EAGAIN) {
			return ErrAlreadyLocked
		}

		return err
	}

	return nil
}
