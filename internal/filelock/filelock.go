package filelock

import (
	"errors"
	"golang.org/x/sys/unix"
	"syscall"
)

var ErrAlreadyLocked = errors.New("already locked")

type FileLock struct {
	fd uintptr
}

func New(path string) (*FileLock, error) {
	fd, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &FileLock{
		fd: uintptr(fd),
	}, nil
}

func (fl *FileLock) Trylock() error {
	_, err := fl.lockWrapper(unix.F_SETLK, unix.F_WRLCK)

	return err
}

func (fl *FileLock) Lock() error {
	_, err := fl.lockWrapper(unix.F_SETLKW, unix.F_WRLCK)

	return err
}

func (fl *FileLock) Unlock() error {
	_, err := fl.lockWrapper(unix.F_SETLK, unix.F_UNLCK)

	return err
}

func (fl *FileLock) Pid() (int32, error) {
	result, err := fl.lockWrapper(unix.F_GETLK, unix.F_RDLCK)
	if err != nil {
		return 0, err
	}

	return result.Pid, nil
}
func (fl *FileLock) Close() error {
	return unix.Close(int(fl.fd))
}

func (fl *FileLock) lockWrapper(operation int, lockType int16) (*unix.Flock_t, error) {
	result := &unix.Flock_t{
		Type:   lockType,
		Whence: unix.SEEK_SET,
	}

	if err := unix.FcntlFlock(fl.fd, operation, result); err != nil {
		if operation == unix.F_SETLK && errors.Is(err, syscall.EAGAIN) {
			return nil, ErrAlreadyLocked
		}

		return nil, err
	}

	return result, nil
}
