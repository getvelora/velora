package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRuntimeDirectoriesCreatesMissingPaths(t *testing.T) {
	root := t.TempDir()

	paths := RuntimePaths{
		Config: filepath.Join(root, "config"),
		Cache:  filepath.Join(root, "cache"),
		Media:  filepath.Join(root, "media"),
	}

	if err := EnsureRuntimeDirectories(paths); err != nil {
		t.Fatalf("ensure runtime directories: %v", err)
	}

	for _, path := range []string{paths.Config, paths.Cache, paths.Media} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}

		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", path)
		}
	}
}

func TestValidateMediaPathAcceptsRootAndDescendants(t *testing.T) {
	for _, candidate := range []string{
		"/media",
		"/media/movies",
		"/media/movies/4k",
	} {
		if err := ValidateMediaPath("/media", candidate); err != nil {
			t.Fatalf("expected %q to be valid: %v", candidate, err)
		}
	}
}

func TestValidateMediaPathRejectsUnsafePaths(t *testing.T) {
	cases := []string{
		"media/movies",
		"/media2/movies",
		"/media/movies/../tv",
		"/media/..",
		"/etc",
	}
	for _, candidate := range cases {
		t.Run(candidate, func(t *testing.T) {
			if err := ValidateMediaPath("/media", candidate); err == nil {
				t.Fatalf("expected %q to be rejected", candidate)
			}
		})
	}
}

func TestValidateResolvedMediaPathRejectsSymlinkInsideMediaRoot(t *testing.T) {
	mediaRoot := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(mediaRoot, "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	if err := ValidateResolvedMediaPath(mediaRoot, link); err == nil {
		t.Fatal("expected symlinked library path to be rejected")
	}
}
