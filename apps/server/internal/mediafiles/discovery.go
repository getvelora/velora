package mediafiles

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var supportedExtensions = map[string]struct{}{
	".avi":  {},
	".m2ts": {},
	".m4v":  {},
	".mkv":  {},
	".mov":  {},
	".mp4":  {},
	".mpeg": {},
	".mpg":  {},
	".ts":   {},
	".webm": {},
	".wmv":  {},
}

type DiscoveryResult struct {
	Files   []DiscoveredFile
	Ignored int
}

func Discover(ctx context.Context, root string) (DiscoveryResult, error) {
	if err := ctx.Err(); err != nil {
		return DiscoveryResult{}, err
	}

	rootInfo, err := os.Lstat(root)
	if err != nil {
		return DiscoveryResult{}, fmt.Errorf("inspect library root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return DiscoveryResult{}, fmt.Errorf("library root must not be a symbolic link")
	}
	if !rootInfo.IsDir() {
		return DiscoveryResult{}, fmt.Errorf("library root is not a directory")
	}
	result := DiscoveryResult{Files: []DiscoveredFile{}}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			result.Ignored++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			result.Ignored++
			return nil
		}
		if _, supported := supportedExtensions[strings.ToLower(filepath.Ext(entry.Name()))]; !supported {
			result.Ignored++
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result.Files = append(result.Files, DiscoveredFile{
			Path:       filepath.ToSlash(relative),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return DiscoveryResult{}, fmt.Errorf("walk library: %w", err)
	}
	sort.Slice(result.Files, func(i, j int) bool {
		return result.Files[i].Path < result.Files[j].Path
	})
	return result, nil
}
