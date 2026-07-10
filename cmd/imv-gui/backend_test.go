package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/soira237-risu/ai-image-metadata-organizer/internal/appcore"
)

func TestResetClearsSessionStateOnly(t *testing.T) {
	backend := NewBackend()
	folder := t.TempDir()
	backend.useFolder(folder)

	state := backend.Reset()
	if state.Folder != "" || state.DBPath != appcore.DefaultDBPath {
		t.Fatalf("unexpected reset state: %#v", state)
	}
}

func TestFolderStateIncludesSelectedPathForOpenFile(t *testing.T) {
	backend := NewBackend()
	imagePath := filepath.Join(t.TempDir(), "image.png")

	state := backend.useFile(imagePath)
	if state.Folder != filepath.Dir(imagePath) || state.SelectedPath != imagePath {
		t.Fatalf("unexpected file state: %#v", state)
	}
}

func TestRevealCommandByPlatform(t *testing.T) {
	tests := []struct {
		goos string
		name string
	}{
		{goos: "windows", name: "explorer"},
		{goos: "darwin", name: "open"},
		{goos: "linux", name: "xdg-open"},
	}
	for _, tt := range tests {
		name, args, err := revealCommand(tt.goos, "C:\\images")
		if err != nil {
			t.Fatalf("revealCommand(%q) error: %v", tt.goos, err)
		}
		if name != tt.name || len(args) != 1 || args[0] != "C:\\images" {
			t.Fatalf("unexpected command for %s on host %s: %q %#v", tt.goos, runtime.GOOS, name, args)
		}
	}
	if _, _, err := revealCommand("plan9", "C:\\images"); err == nil {
		t.Fatal("expected unsupported platform error")
	}
}
