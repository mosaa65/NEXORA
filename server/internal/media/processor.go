package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Processor struct {
	ffmpegPath  string
	ffprobePath string
}

type VerifyResult struct {
	Path        string `json:"path"`
	Healthy     bool   `json:"healthy"`
	ErrorOutput string `json:"errorOutput,omitempty"`
}

type Track struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec"`
	Language string `json:"language,omitempty"`
	Title    string `json:"title,omitempty"`
}

type InspectResult struct {
	Path        string  `json:"path"`
	Duration    int     `json:"duration"`
	Resolution  string  `json:"resolution,omitempty"`
	VideoCodec  string  `json:"videoCodec,omitempty"`
	AudioTracks []Track `json:"audioTracks"`
	Subtitles   []Track `json:"subtitles"`
}

func NewProcessor(ffmpegPath, ffprobePath string) *Processor {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &Processor{ffmpegPath: ffmpegPath, ffprobePath: ffprobePath}
}

func (p *Processor) Verify(ctx context.Context, path string) (VerifyResult, error) {
	if path == "" {
		return VerifyResult{}, errors.New("path is required")
	}

	command := exec.CommandContext(ctx, p.ffmpegPath, "-v", "error", "-i", path, "-f", "null", "-")
	var stderr bytes.Buffer
	command.Stderr = &stderr

	err := command.Run()
	result := VerifyResult{
		Path:        path,
		Healthy:     err == nil && stderr.Len() == 0,
		ErrorOutput: stderr.String(),
	}
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return result, fmt.Errorf("ffmpeg executable not found at %q", p.ffmpegPath)
		}
		return result, nil
	}
	return result, nil
}

// Inspect obtains technical metadata without decoding the whole video stream.
func (p *Processor) Inspect(ctx context.Context, path string) (InspectResult, error) {
	if strings.TrimSpace(path) == "" {
		return InspectResult{}, errors.New("path is required")
	}
	command := exec.CommandContext(ctx, p.ffprobePath, "-v", "error", "-show_streams", "-show_format", "-of", "json", path)
	output, err := command.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return InspectResult{}, fmt.Errorf("ffprobe executable not found at %q", p.ffprobePath)
		}
		return InspectResult{}, fmt.Errorf("inspect media: %w", err)
	}
	var probe struct {
		Format struct { Duration string `json:"duration"` } `json:"format"`
		Streams []struct {
			Index     int               `json:"index"`
			CodecType string            `json:"codec_type"`
			CodecName string            `json:"codec_name"`
			Width     int               `json:"width"`
			Height    int               `json:"height"`
			Tags      map[string]string `json:"tags"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &probe); err != nil {
		return InspectResult{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	result := InspectResult{Path: path, AudioTracks: []Track{}, Subtitles: []Track{}}
	if seconds, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil && seconds > 0 {
		result.Duration = int(seconds)
	}
	for _, stream := range probe.Streams {
		track := Track{Index: stream.Index, Codec: stream.CodecName, Language: stream.Tags["language"], Title: stream.Tags["title"]}
		switch stream.CodecType {
		case "video":
			if result.VideoCodec == "" {
				result.VideoCodec = stream.CodecName
				if stream.Width > 0 && stream.Height > 0 { result.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height) }
			}
		case "audio": result.AudioTracks = append(result.AudioTracks, track)
		case "subtitle": result.Subtitles = append(result.Subtitles, track)
		}
	}
	return result, nil
}

func (p *Processor) GenerateThumbnail(ctx context.Context, inputPath, outputPath string, at time.Duration) (string, error) {
	if inputPath == "" {
		return "", errors.New("input path is required")
	}
	if outputPath == "" {
		return "", errors.New("output path is required")
	}
	if at <= 0 {
		at = 10 * time.Second
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", err
	}

	command := exec.CommandContext(
		ctx,
		p.ffmpegPath,
		"-y",
		"-ss",
		strconv.FormatFloat(at.Seconds(), 'f', 3, 64),
		"-i",
		inputPath,
		"-frames:v",
		"1",
		"-q:v",
		"2",
		outputPath,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("ffmpeg executable not found at %q", p.ffmpegPath)
		}
		return "", fmt.Errorf("generate thumbnail: %w: %s", err, stderr.String())
	}
	return filepath.ToSlash(outputPath), nil
}
