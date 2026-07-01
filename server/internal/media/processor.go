package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
