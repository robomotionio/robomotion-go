//go:build windows

package lmo

import (
	"errors"
	"os"
	"syscall"
	"time"
)

// On Windows, MoveFileEx and CreateFile interact through share modes. Go's
// os.Open opens with FILE_SHARE_READ | FILE_SHARE_WRITE — but NOT
// FILE_SHARE_DELETE — so a concurrent reader prevents the writer's
// MoveFileEx (Access is denied), and a rename-in-flight briefly blocks new
// opens (Sharing violation). POSIX rename(2) has no such races.
//
// We treat these as transient: retry with short exponential backoff. Real
// permission failures (a file actually not readable) still surface because
// they don't clear on retry — the loop just adds a small startup cost.
const (
	maxFileRetries     = 16
	initialFileBackoff = 100 * time.Microsecond
	maxFileBackoff     = 8 * time.Millisecond
)

// Windows error codes not exported as named constants by the syscall pkg.
// Defined inline rather than pulling in golang.org/x/sys/windows.
const (
	errSharingViolation syscall.Errno = 32 // ERROR_SHARING_VIOLATION
	errLockViolation    syscall.Errno = 33 // ERROR_LOCK_VIOLATION
)

func isTransientShareErr(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	switch errno {
	case errSharingViolation, errLockViolation, syscall.ERROR_ACCESS_DENIED:
		return true
	}
	return false
}

func renameAtomic(oldPath, newPath string) error {
	backoff := initialFileBackoff
	var lastErr error
	for i := 0; i < maxFileRetries; i++ {
		err := os.Rename(oldPath, newPath)
		if err == nil {
			return nil
		}
		if !isTransientShareErr(err) {
			return err
		}
		lastErr = err
		time.Sleep(backoff)
		if backoff < maxFileBackoff {
			backoff *= 2
		}
	}
	return lastErr
}

func readFile(p string) ([]byte, error) {
	backoff := initialFileBackoff
	var lastErr error
	for i := 0; i < maxFileRetries; i++ {
		data, err := os.ReadFile(p)
		if err == nil {
			return data, nil
		}
		if !isTransientShareErr(err) {
			return nil, err
		}
		lastErr = err
		time.Sleep(backoff)
		if backoff < maxFileBackoff {
			backoff *= 2
		}
	}
	return nil, lastErr
}
