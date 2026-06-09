// Package libraries owns the model and persistence for configured media
// roots. Each library is a (name, path) pair pointing at a directory under
// the configured media root that the scanner walks.
package libraries

import "time"

// Library is a configured media root.
type Library struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
