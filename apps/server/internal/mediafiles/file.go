// Package mediafiles discovers and persists video files belonging to configured
// libraries.
package mediafiles

import "time"

const (
	StatusAvailable = "available"
	StatusMissing   = "missing"

	InspectionStatusUnprobed = "unprobed"
	InspectionStatusReady    = "ready"
	InspectionStatusError    = "error"

	StreamTypeVideo    = "video"
	StreamTypeAudio    = "audio"
	StreamTypeSubtitle = "subtitle"
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
	Inspection  Inspection `json:"inspection"`
}

type DiscoveredFile struct {
	Path       string
	Size       int64
	ModifiedAt time.Time
	Inspection InspectionResult
}

type ReconcileSummary struct {
	Discovered    int `json:"discovered"`
	Added         int `json:"added"`
	Updated       int `json:"updated"`
	Unchanged     int `json:"unchanged"`
	Restored      int `json:"restored"`
	MarkedMissing int `json:"markedMissing"`
	Probed        int `json:"probed"`
	ProbeFailed   int `json:"probeFailed"`
}

type Inspection struct {
	Status   string     `json:"status"`
	Error    *string    `json:"error"`
	ProbedAt *time.Time `json:"probedAt"`
	Format   *Format    `json:"format"`
	Streams  []Stream   `json:"streams"`
}

type Format struct {
	Name       string `json:"name"`
	LongName   string `json:"longName"`
	DurationMS int64  `json:"durationMs"`
	BitRate    int64  `json:"bitRate"`
}

type Stream struct {
	Index          int    `json:"index"`
	Type           string `json:"type"`
	CodecName      string `json:"codecName"`
	CodecLongName  string `json:"codecLongName"`
	Profile        string `json:"profile"`
	Level          *int   `json:"level"`
	Language       string `json:"language"`
	Title          string `json:"title"`
	Default        bool   `json:"default"`
	Forced         bool   `json:"forced"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	PixelFormat    string `json:"pixelFormat"`
	BitDepth       int    `json:"bitDepth"`
	FrameRate      string `json:"frameRate"`
	ColorRange     string `json:"colorRange"`
	ColorSpace     string `json:"colorSpace"`
	ColorTransfer  string `json:"colorTransfer"`
	ColorPrimaries string `json:"colorPrimaries"`
	SampleRate     int    `json:"sampleRate"`
	Channels       int    `json:"channels"`
	ChannelLayout  string `json:"channelLayout"`
}

type ProbeResult struct {
	Format  Format
	Streams []Stream
}

type InspectionResult struct {
	Attempted bool
	Result    ProbeResult
	Error     string
}
