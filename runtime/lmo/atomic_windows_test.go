//go:build windows

package lmo

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Deterministic Windows-only repros for the share-mode race that
// renameAtomic / readFile defend against. Plain os.Rename / os.ReadFile
// fail immediately under these conditions; the retried variants must
// succeed once the holder releases mid-backoff.

// openExclusiveRead opens a file with dwShareMode = 0 so that any
// other CreateFile (including the one os.Rename / os.ReadFile issue
// internally) returns ERROR_SHARING_VIOLATION until this handle closes.
func openExclusiveRead(t *testing.T, path string) syscall.Handle {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(
		p,
		syscall.GENERIC_READ,
		0, // no sharing — blocks all other opens
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatalf("CreateFile %s exclusive: %v", path, err)
	}
	return h
}

func TestRenameAtomic_RetriesPastSharingViolation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	holder := openExclusiveRead(t, dst)

	// Sanity: plain os.Rename fails immediately while holder is open.
	if err := os.Rename(src, dst); err == nil {
		syscall.CloseHandle(holder)
		t.Fatal("baseline check failed: os.Rename should reject under exclusive holder")
	}

	done := make(chan error, 1)
	go func() {
		done <- renameAtomic(src, dst)
	}()

	// Let renameAtomic burn through a few backoff cycles before releasing.
	time.Sleep(2 * time.Millisecond)
	syscall.CloseHandle(holder)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("renameAtomic should retry past share violation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("renameAtomic blocked past expected retry window")
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("dst content = %q, want %q", got, "new")
	}
}

func TestReadFile_RetriesPastSharingViolation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "blob")
	want := []byte("payload")
	if err := os.WriteFile(p, want, 0644); err != nil {
		t.Fatal(err)
	}

	holder := openExclusiveRead(t, p)

	// Sanity: plain os.ReadFile fails immediately while holder is open.
	if _, err := os.ReadFile(p); err == nil {
		syscall.CloseHandle(holder)
		t.Fatal("baseline check failed: os.ReadFile should reject under exclusive holder")
	}

	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := readFile(p)
		done <- result{data: data, err: err}
	}()

	time.Sleep(2 * time.Millisecond)
	syscall.CloseHandle(holder)

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("readFile should retry past share violation: %v", r.err)
		}
		if string(r.data) != string(want) {
			t.Fatalf("readFile content = %q, want %q", r.data, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("readFile blocked past expected retry window")
	}
}
