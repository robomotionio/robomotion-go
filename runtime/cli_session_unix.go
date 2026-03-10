//go:build !windows

package runtime

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

// sessionDir returns the platform-specific directory for session files.
func sessionDir() string {
	if runtime.GOOS == "darwin" {
		if tmp := os.Getenv("TMPDIR"); tmp != "" {
			return filepath.Join(tmp, "robomotion-sessions")
		}
	} else {
		if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
			return filepath.Join(xdg, "robomotion-sessions")
		}
	}
	return "/tmp/robomotion-sessions"
}

// sessionListen creates a Unix domain socket listener for the session.
func sessionListen(sessionID string) (net.Listener, error) {
	dir := sessionDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	sockPath := filepath.Join(dir, sessionID+".sock")

	// Clean up stale socket if it exists
	os.Remove(sockPath)

	return net.Listen("unix", sockPath)
}

// sessionDialAddr returns the gRPC dial address for a session.
func sessionDialAddr(sessionID string) string {
	sockPath := filepath.Join(sessionDir(), sessionID+".sock")
	if _, err := os.Stat(sockPath); err != nil {
		return ""
	}
	return "unix://" + sockPath
}

// sessionCleanup removes the socket and metadata files for a session.
func sessionCleanup(sessionID string) {
	dir := sessionDir()
	os.Remove(filepath.Join(dir, sessionID+".sock"))
	os.Remove(filepath.Join(dir, sessionID+".json"))
}
