//go:build windows

package routecache

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockRangeAll locks the whole file (the maximum byte range LockFileEx accepts).
const lockRangeAll = ^uint32(0)

// tryLockFile takes a non-blocking exclusive LockFileEx lock on f. It reports
// false (and no error) when another process holds the lock.
func tryLockFile(f *os.File) (bool, error) {
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, lockRangeAll, lockRangeAll, ol)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return false, err
}

func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockRangeAll, lockRangeAll, ol)
}
