//go:build !(linux || darwin || android || freebsd || netbsd || openbsd || dragonfly || windows)

package routecache

import "os"

// tryLockFile is a no-op on platforms without flock or LockFileEx. Writes are
// still atomic (temp file + rename), so the file is never corrupted, but two
// processes writing at the same moment can drop one update.
func tryLockFile(*os.File) (bool, error) { return true, nil }

func unlockFile(*os.File) error { return nil }
