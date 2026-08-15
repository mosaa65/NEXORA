package media

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SubtitleInfo struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Format   string `json:"format"`
}

var srtTimeRegex = regexp.MustCompile(`(\d{2}:\d{2}:\d{2}),(\d{3})`)

func FindExternalSubtitles(videoPath string) []SubtitleInfo {
	dir := filepath.Dir(videoPath)
	videoBase := strings.ToLower(strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath)))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	subs := make([]SubtitleInfo, 0)
	idx := 1

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".srt" && ext != ".vtt" && ext != ".ass" && ext != ".sub" {
			continue
		}

		subBase := strings.ToLower(strings.TrimSuffix(entry.Name(), ext))
		// Check if it belongs to this video
		if subBase == videoBase || strings.HasPrefix(subBase, videoBase) || strings.Contains(subBase, videoBase) {
			lang := detectSubtitleLanguage(entry.Name())
			title := entry.Name()
			if lang == "ar" || lang == "Arabic" {
				title = "العربية (مترجم)"
			} else if lang == "en" || lang == "English" {
				title = "English"
			}

			subs = append(subs, SubtitleInfo{
				Index:    idx,
				Language: lang,
				Title:    title,
				Path:     filepath.Join(dir, entry.Name()),
				Format:   strings.TrimPrefix(ext, "."),
			})
			idx++
		}
	}

	return subs
}

func ConvertSRTToWebVTT(r io.Reader, w io.Writer) error {
	writer := bufio.NewWriter(w)
	if _, err := writer.WriteString("WEBVTT\n\n"); err != nil {
		return err
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		// Convert SRT timestamps: 00:01:23,456 --> 00:01:25,789 to 00:01:23.456 --> 00:01:25.789
		converted := srtTimeRegex.ReplaceAllString(line, "${1}.${2}")
		if _, err := fmt.Fprintf(writer, "%s\n", converted); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return writer.Flush()
}

func detectSubtitleLanguage(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.Contains(lower, "arabic") || strings.Contains(lower, "ar.") || strings.Contains(lower, "_ar") || strings.Contains(lower, "عربي"):
		return "Arabic"
	case strings.Contains(lower, "english") || strings.Contains(lower, "en.") || strings.Contains(lower, "_en") || strings.Contains(lower, "انجليزي"):
		return "English"
	case strings.Contains(lower, "french") || strings.Contains(lower, "fr.") || strings.Contains(lower, "français"):
		return "French"
	case strings.Contains(lower, "spanish") || strings.Contains(lower, "es.") || strings.Contains(lower, "español"):
		return "Spanish"
	case strings.Contains(lower, "japanese") || strings.Contains(lower, "ja."):
		return "Japanese"
	default:
		return "Undetermined"
	}
}
