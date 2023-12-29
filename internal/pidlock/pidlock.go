package pidlock

import (
	"errors"
	"golang.org/x/sys/unix"
	"syscall"
)

var ErrAlreadyLocked = errors.New("already locked")

type PIDLock struct {
	fd uintptr
}

func New(path string) (*PIDLock, error) {
	fd, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &PIDLock{
		fd: uintptr(fd),
	}, nil
}

func (fl *PIDLock) Trylock() error {
	_, err := fl.lockWrapper(unix.F_SETLK, unix.F_WRLCK)

	return err
}

func (fl *PIDLock) Lock() error {
	_, err := fl.lockWrapper(unix.F_SETLKW, unix.F_WRLCK)

	return err
}

func (fl *PIDLock) Unlock() error {
	_, err := fl.lockWrapper(unix.F_SETLK, unix.F_UNLCK)

	return err
}

func (fl *PIDLock) Pid() (int32, error) {
	result, err := fl.lockWrapper(unix.F_GETLK, unix.F_RDLCK)
	if err != nil {
		return 0, err
	}

	return result.Pid, nil
}
func (fl *PIDLock) Close() error {
	return unix.Close(int(fl.fd))
}

func (fl *PIDLock) lockWrapper(operation int, lockType int16) (*unix.Flock_t, error) {
	result := &unix.Flock_t{
		Type:   lockType,
		Whence: unix.SEEK_SET,
		Start:  0,
		Len:    0,
	}

	if err := unix.FcntlFlock(fl.fd, operation, result); err != nil {
		if operation == unix.F_SETLK && errors.Is(err, syscall.EAGAIN) {
			return nil, ErrAlreadyLocked
		}

		return nil, err
	}

	return result, nil
}
