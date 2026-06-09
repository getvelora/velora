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
