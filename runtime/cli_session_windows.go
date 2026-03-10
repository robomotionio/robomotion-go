//go:build windows

package runtime

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sessionDir returns the Windows-specific directory for session files.
func sessionDir() string {
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		return filepath.Join(appData, "Robomotion", "sessions")
	}
	return filepath.Join(os.TempDir(), "robomotion-sessions")
}

// sessionListen creates a TCP localhost listener and writes the port to a file.
func sessionListen(sessionID string) (net.Listener, error) {
	dir := sessionDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("tcp listen: %w", err)
	}

	// Write the assigned port to a file
	_, port, _ := net.SplitHostPort(lis.Addr().String())
	portPath := filepath.Join(dir, sessionID+".port")
	if err := os.WriteFile(portPath, []byte(port), 0600); err != nil {
		lis.Close()
		return nil, fmt.Errorf("write port file: %w", err)
	}

	return lis, nil
}

// sessionDialAddr reads the port file and returns the gRPC dial address.
func sessionDialAddr(sessionID string) string {
	portPath := filepath.Join(sessionDir(), sessionID+".port")
	data, err := os.ReadFile(portPath)
	if err != nil {
		return ""
	}
	port := strings.TrimSpace(string(data))
	if _, err := strconv.Atoi(port); err != nil {
		return ""
	}
	return "127.0.0.1:" + port
}

// sessionCleanup removes the port and metadata files for a session.
func sessionCleanup(sessionID string) {
	dir := sessionDir()
	os.Remove(filepath.Join(dir, sessionID+".port"))
	os.Remove(filepath.Join(dir, sessionID+".json"))
}
