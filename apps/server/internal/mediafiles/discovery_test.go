package mediafiles

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFindsSupportedVideosAndIgnoresOtherFilesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Movie.MKV"), "movie")
	writeTestFile(t, filepath.Join(root, "shows", "Episode.mp4"), "episode")
	writeTestFile(t, filepath.Join(root, "notes.txt"), "notes")
	writeTestFile(t, filepath.Join(root, "poster.jpg"), "poster")

	if err := os.Symlink(
		filepath.Join(root, "Movie.MKV"),
		filepath.Join(root, "linked.mkv"),
	); err != nil {
		t.Fatalf("create file symlink: %v", err)
	}
	if err := os.Symlink(
		filepath.Join(root, "shows"),
		filepath.Join(root, "linked-shows"),
	); err != nil {
		t.Fatalf("create directory symlink: %v", err)
	}

	result, err := Discover(context.Background(), root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if result.Ignored != 4 {
		t.Fatalf("expected 4 ignored entries, got %d", result.Ignored)
	}
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 video files, got %+v", result.Files)
	}
	if result.Files[0].Path != "Movie.MKV" || result.Files[1].Path != "shows/Episode.mp4" {
		t.Fatalf("unexpected paths: %+v", result.Files)
	}
	if result.Files[0].Size != int64(len("movie")) {
		t.Fatalf("unexpected size: %+v", result.Files[0])
	}
	if result.Files[0].ModifiedAt.IsZero() {
		t.Fatal("expected modification time")
	}
}

func TestDiscoverRejectsSymlinkedRoot(t *testing.T) {
	target := t.TempDir()
	root := filepath.Join(t.TempDir(), "library")
	if err := os.Symlink(target, root); err != nil {
		t.Fatalf("create root symlink: %v", err)
	}

	if _, err := Discover(context.Background(), root); err == nil {
		t.Fatal("expected symlinked root to be rejected")
	}
}

func TestDiscoverHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Discover(ctx, t.TempDir()); err == nil {
		t.Fatal("expected canceled discovery to fail")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
