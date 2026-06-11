package mediafiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var ErrProbeRuntime = errors.New("ffprobe runtime failure")

type FFProber struct {
	binary string
}

func NewFFProber(binary string) *FFProber {
	return &FFProber{binary: binary}
}

func (p *FFProber) Probe(ctx context.Context, path string) (ProbeResult, error) {
	cmd := exec.CommandContext(
		ctx,
		p.binary,
		"-v", "error",
		"-show_format",
		"-show_streams",
		"-of", "json",
		"--",
		path,
	)
	output, err := cmd.Output()
	if err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return ProbeResult{}, fmt.Errorf("%w: %v", ErrProbeRuntime, err)
		}
		var pathErr *os.PathError
		if errors.As(err, &pathErr) && pathErr.Op == "fork/exec" {
			return ProbeResult{}, fmt.Errorf("%w: %v", ErrProbeRuntime, err)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(string(exitErr.Stderr))
			if message == "" {
				message = exitErr.Error()
			}
			return ProbeResult{}, fmt.Errorf("ffprobe failed: %s", message)
		}
		return ProbeResult{}, fmt.Errorf("run ffprobe: %w", err)
	}
	return ParseProbeOutput(output)
}

type probeDocument struct {
	Format struct {
		Name     string `json:"format_name"`
		LongName string `json:"format_long_name"`
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		Index          int               `json:"index"`
		Type           string            `json:"codec_type"`
		CodecName      string            `json:"codec_name"`
		CodecLongName  string            `json:"codec_long_name"`
		Profile        string            `json:"profile"`
		Level          *int              `json:"level"`
		Width          int               `json:"width"`
		Height         int               `json:"height"`
		PixelFormat    string            `json:"pix_fmt"`
		BitDepth       string            `json:"bits_per_raw_sample"`
		FrameRate      string            `json:"avg_frame_rate"`
		ColorRange     string            `json:"color_range"`
		ColorSpace     string            `json:"color_space"`
		ColorTransfer  string            `json:"color_transfer"`
		ColorPrimaries string            `json:"color_primaries"`
		SampleRate     string            `json:"sample_rate"`
		Channels       int               `json:"channels"`
		ChannelLayout  string            `json:"channel_layout"`
		Tags           map[string]string `json:"tags"`
		Disposition    struct {
			Default int `json:"default"`
			Forced  int `json:"forced"`
		} `json:"disposition"`
	} `json:"streams"`
}

func ParseProbeOutput(output []byte) (ProbeResult, error) {
	var document probeDocument
	if err := json.Unmarshal(output, &document); err != nil {
		return ProbeResult{}, fmt.Errorf("decode ffprobe output: %w", err)
	}

	result := ProbeResult{
		Format: Format{
			Name:       document.Format.Name,
			LongName:   document.Format.LongName,
			DurationMS: parseDurationMilliseconds(document.Format.Duration),
			BitRate:    parseInt64(document.Format.BitRate),
		},
		Streams: []Stream{},
	}
	for _, item := range document.Streams {
		if item.Type != StreamTypeVideo &&
			item.Type != StreamTypeAudio &&
			item.Type != StreamTypeSubtitle {
			continue
		}
		result.Streams = append(result.Streams, Stream{
			Index:          item.Index,
			Type:           item.Type,
			CodecName:      item.CodecName,
			CodecLongName:  item.CodecLongName,
			Profile:        item.Profile,
			Level:          item.Level,
			Language:       item.Tags["language"],
			Title:          item.Tags["title"],
			Default:        item.Disposition.Default == 1,
			Forced:         item.Disposition.Forced == 1,
			Width:          item.Width,
			Height:         item.Height,
			PixelFormat:    item.PixelFormat,
			BitDepth:       parseInt32(item.BitDepth),
			FrameRate:      item.FrameRate,
			ColorRange:     item.ColorRange,
			ColorSpace:     item.ColorSpace,
			ColorTransfer:  item.ColorTransfer,
			ColorPrimaries: item.ColorPrimaries,
			SampleRate:     parseInt32(item.SampleRate),
			Channels:       item.Channels,
			ChannelLayout:  item.ChannelLayout,
		})
	}
	return result, nil
}

func parseDurationMilliseconds(value string) int64 {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 {
		return 0
	}
	return int64(seconds * 1000)
}

func parseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func parseInt32(value string) int {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed < 0 {
		return 0
	}
	return int(parsed)
}
