package mediafiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseProbeOutputNormalizesSupportedStreams(t *testing.T) {
	t.Parallel()

	output := []byte(`{
		"format": {
			"format_name": "matroska,webm",
			"format_long_name": "Matroska / WebM",
			"duration": "7265.123",
			"bit_rate": "18432000"
		},
		"streams": [
			{
				"index": 0,
				"codec_type": "video",
				"codec_name": "hevc",
				"codec_long_name": "H.265 / HEVC",
				"profile": "Main 10",
				"level": 153,
				"width": 3840,
				"height": 2160,
				"pix_fmt": "yuv420p10le",
				"bits_per_raw_sample": "10",
				"avg_frame_rate": "24000/1001",
				"color_range": "tv",
				"color_space": "bt2020nc",
				"color_transfer": "smpte2084",
				"color_primaries": "bt2020",
				"tags": {"language": "eng", "title": "Main video"},
				"disposition": {"default": 1, "forced": 0}
			},
			{
				"index": 1,
				"codec_type": "audio",
				"codec_name": "eac3",
				"sample_rate": "48000",
				"channels": 6,
				"channel_layout": "5.1(side)",
				"tags": {"language": "eng"},
				"disposition": {"default": 1, "forced": 0}
			},
			{
				"index": 2,
				"codec_type": "subtitle",
				"codec_name": "subrip",
				"tags": {"language": "spa", "title": "Spanish"},
				"disposition": {"default": 0, "forced": 1}
			},
			{"index": 3, "codec_type": "attachment", "codec_name": "ttf"},
			{"index": 4, "codec_type": "data", "codec_name": "bin_data"}
		]
	}`)

	got, err := ParseProbeOutput(output)
	if err != nil {
		t.Fatalf("ParseProbeOutput: %v", err)
	}

	if got.Format.Name != "matroska,webm" ||
		got.Format.LongName != "Matroska / WebM" ||
		got.Format.DurationMS != 7_265_123 ||
		got.Format.BitRate != 18_432_000 {
		t.Fatalf("unexpected format: %+v", got.Format)
	}
	if len(got.Streams) != 3 {
		t.Fatalf("expected 3 supported streams, got %+v", got.Streams)
	}

	video := got.Streams[0]
	if video.Index != 0 || video.Type != StreamTypeVideo || video.CodecName != "hevc" ||
		video.Level == nil || *video.Level != 153 || video.Width != 3840 ||
		video.Height != 2160 || video.BitDepth != 10 ||
		video.FrameRate != "24000/1001" || video.ColorTransfer != "smpte2084" ||
		video.Language != "eng" || !video.Default || video.Forced {
		t.Fatalf("unexpected video stream: %+v", video)
	}

	audio := got.Streams[1]
	if audio.Type != StreamTypeAudio || audio.SampleRate != 48000 ||
		audio.Channels != 6 || audio.ChannelLayout != "5.1(side)" {
		t.Fatalf("unexpected audio stream: %+v", audio)
	}

	subtitle := got.Streams[2]
	if subtitle.Type != StreamTypeSubtitle || subtitle.Language != "spa" ||
		subtitle.Title != "Spanish" || subtitle.Default || !subtitle.Forced {
		t.Fatalf("unexpected subtitle stream: %+v", subtitle)
	}
}

func TestParseProbeOutputAllowsMissingOptionalFields(t *testing.T) {
	t.Parallel()

	got, err := ParseProbeOutput([]byte(`{
		"format": {"format_name": "mpegts", "duration": "N/A"},
		"streams": [{"index": 0, "codec_type": "video", "codec_name": "h264"}]
	}`))
	if err != nil {
		t.Fatalf("ParseProbeOutput: %v", err)
	}
	if got.Format.DurationMS != 0 || got.Format.BitRate != 0 {
		t.Fatalf("expected unavailable numeric fields to be zero, got %+v", got.Format)
	}
	if len(got.Streams) != 1 || got.Streams[0].Level != nil {
		t.Fatalf("unexpected streams: %+v", got.Streams)
	}
}

func TestParseProbeOutputRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	if _, err := ParseProbeOutput([]byte(`{"format":`)); err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestFFProberClassifiesMissingExecutableAsRuntimeFailure(t *testing.T) {
	t.Parallel()

	prober := NewFFProber("/definitely/not/a/real/ffprobe")
	_, err := prober.Probe(context.Background(), "/media/movie.mkv")
	if err == nil {
		t.Fatal("expected probe error")
	}
	if !errors.Is(err, ErrProbeRuntime) {
		t.Fatalf("expected ErrProbeRuntime, got %v", err)
	}
}

func TestFFProberExecutesAndParsesOutput(t *testing.T) {
	t.Parallel()

	binary := filepath.Join(t.TempDir(), "ffprobe")
	script := `#!/bin/sh
printf '%s' '{"format":{"format_name":"mov,mp4","duration":"2.5"},"streams":[]}'
`
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ffprobe: %v", err)
	}

	result, err := NewFFProber(binary).Probe(context.Background(), "/media/movie.mp4")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if result.Format.Name != "mov,mp4" || result.Format.DurationMS != 2_500 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
