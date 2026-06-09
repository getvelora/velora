// Package mediafiles discovers and persists video files belonging to configured
// libraries.
package mediafiles

import "time"

const (
	StatusAvailable = "available"
	StatusMissing   = "missing"
)

type File struct {
	ID          int64      `json:"id"`
	LibraryID   int64      `json:"libraryId"`
	Path        string     `json:"path"`
	Size        int64      `json:"size"`
	ModifiedAt  time.Time  `json:"modifiedAt"`
	Status      string     `json:"status"`
	FirstSeenAt time.Time  `json:"firstSeenAt"`
	LastSeenAt  time.Time  `json:"lastSeenAt"`
	MissingAt   *time.Time `json:"missingAt"`
}

type DiscoveredFile struct {
	Path       string
	Size       int64
	ModifiedAt time.Time
}

type ReconcileSummary struct {
	Discovered    int `json:"discovered"`
	Added         int `json:"added"`
	Updated       int `json:"updated"`
	Unchanged     int `json:"unchanged"`
	Restored      int `json:"restored"`
	MarkedMissing int `json:"markedMissing"`
}
