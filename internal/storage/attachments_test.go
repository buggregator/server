package storage_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/buggregator/go-buggregator/internal/storage"
)

func TestAttachmentStore_Memory(t *testing.T) {
	store := storage.NewAttachmentStore("memory", "")

	t.Run("store and get", func(t *testing.T) {
		if err := store.Store("event-1/file.txt", []byte("hello")); err != nil {
			t.Fatal(err)
		}
		data, err := store.Get("event-1/file.txt")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "hello" {
			t.Errorf("got %q, want %q", data, "hello")
		}
	})

	t.Run("get nonexistent", func(t *testing.T) {
		_, err := store.Get("nonexistent/file.txt")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("get reader", func(t *testing.T) {
		store.Store("event-2/data.bin", []byte("binary data"))
		rc, err := store.GetReader("event-2/data.bin")
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		data, _ := io.ReadAll(rc)
		if string(data) != "binary data" {
			t.Errorf("got %q, want %q", data, "binary data")
		}
	})

	t.Run("delete by event", func(t *testing.T) {
		store.Store("event-3/a.txt", []byte("a"))
		store.Store("event-3/b.txt", []byte("b"))
		store.Store("event-4/c.txt", []byte("c"))

		store.DeleteByEvent("event-3")

		if _, err := store.Get("event-3/a.txt"); err == nil {
			t.Error("expected event-3/a.txt to be deleted")
		}
		if _, err := store.Get("event-3/b.txt"); err == nil {
			t.Error("expected event-3/b.txt to be deleted")
		}
		// event-4 should remain
		if _, err := store.Get("event-4/c.txt"); err != nil {
			t.Error("event-4/c.txt should still exist")
		}
	})
}

func TestAttachmentStore_DefaultMode(t *testing.T) {
	store := storage.NewAttachmentStore("", "")
	store.Store("test/file", []byte("data"))
	data, err := store.Get("test/file")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("got %q, want %q", data, "data")
	}
}

func TestAttachmentStore_Filesystem(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewAttachmentStore("filesystem", dir)

	t.Run("store and get", func(t *testing.T) {
		if err := store.Store("event-1/file.txt", []byte("fs content")); err != nil {
			t.Fatal(err)
		}

		// Verify file exists on disk
		data, err := os.ReadFile(filepath.Join(dir, "event-1", "file.txt"))
		if err != nil {
			t.Fatalf("file not on disk: %v", err)
		}
		if string(data) != "fs content" {
			t.Errorf("disk content = %q, want %q", data, "fs content")
		}

		// Get via store
		got, err := store.Get("event-1/file.txt")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "fs content" {
			t.Errorf("got %q, want %q", got, "fs content")
		}
	})

	t.Run("get reader filesystem", func(t *testing.T) {
		store.Store("event-2/data.bin", []byte("fs binary"))
		rc, err := store.GetReader("event-2/data.bin")
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		data, _ := io.ReadAll(rc)
		if string(data) != "fs binary" {
			t.Errorf("got %q, want %q", data, "fs binary")
		}
	})

	t.Run("delete by event filesystem", func(t *testing.T) {
		store.Store("event-5/x.txt", []byte("x"))
		store.DeleteByEvent("event-5")

		if _, err := os.Stat(filepath.Join(dir, "event-5")); !os.IsNotExist(err) {
			t.Error("expected event-5 directory to be removed")
		}
	})
}
