package core

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicPreservesDestinationOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("render failed")
	err := WriteAtomic(path, 0644, func(w io.Writer) error {
		_, _ = w.Write([]byte("partial"))
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("WriteAtomic error = %v, want %v", err, wantErr)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("destination = %q, want original", got)
	}
}

func TestWriteFileAtomicReplacesDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.pdf")
	if err := WriteFileAtomic(path, []byte("complete"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "complete" {
		t.Fatalf("destination = %q, want complete", got)
	}
}
