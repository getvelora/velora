package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getvelora/velora/apps/server/internal/env"
)

type RuntimePaths struct {
	Config string
	Cache  string
	Media  string
}

func RuntimePathsFromEnv() RuntimePaths {
	return RuntimePaths{
		Config: env.OrDefault("VELORA_CONFIG_DIR", "/config"),
		Cache:  env.OrDefault("VELORA_CACHE_DIR", "/cache"),
		Media:  env.OrDefault("VELORA_MEDIA_DIR", "/media"),
	}
}

func EnsureRuntimeDirectories(paths RuntimePaths) error {
	for _, path := range []string{paths.Config, paths.Cache, paths.Media} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	return nil
}

// ValidateMediaPath checks that candidate is an absolute path at or below the
// configured media root without traversal components.
func ValidateMediaPath(mediaRoot, candidate string) error {
	if !filepath.IsAbs(mediaRoot) {
		return fmt.Errorf("media root must be absolute")
	}
	if !filepath.IsAbs(candidate) {
		return fmt.Errorf("library path must be absolute")
	}
	for _, part := range strings.Split(filepath.ToSlash(candidate), "/") {
		if part == ".." {
			return fmt.Errorf("library path must not contain traversal")
		}
	}

	root := filepath.Clean(mediaRoot)
	path := filepath.Clean(candidate)
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("compare library path to media root: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("library path must be within %s", root)
	}
	return nil
}

// ValidateResolvedMediaPath verifies that an existing candidate does not use a
// symlink within the configured media tree.
func ValidateResolvedMediaPath(mediaRoot, candidate string) error {
	if err := ValidateMediaPath(mediaRoot, candidate); err != nil {
		return err
	}

	resolvedRoot, err := filepath.EvalSymlinks(mediaRoot)
	if err != nil {
		return fmt.Errorf("resolve media root: %w", err)
	}
	resolvedCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return fmt.Errorf("resolve library path: %w", err)
	}
	lexicalRelative, err := filepath.Rel(filepath.Clean(mediaRoot), filepath.Clean(candidate))
	if err != nil {
		return fmt.Errorf("compare library path: %w", err)
	}
	resolvedRelative, err := filepath.Rel(
		filepath.Clean(resolvedRoot),
		filepath.Clean(resolvedCandidate),
	)
	if err != nil {
		return fmt.Errorf("compare resolved library path: %w", err)
	}
	if lexicalRelative != resolvedRelative {
		return fmt.Errorf("library path must not contain symbolic links")
	}
	return nil
}
