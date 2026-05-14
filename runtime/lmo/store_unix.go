//go:build !windows

package lmo

import "os"

// POSIX rename(2) is atomic and readers don't interfere — no retry needed.
func renameAtomic(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func readFile(p string) ([]byte, error) { return os.ReadFile(p) }
