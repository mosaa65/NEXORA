package media

import (
	"bytes"
	"strings"
	"testing"
)

func TestConvertSRTToWebVTT(t *testing.T) {
	srtInput := `1
00:00:01,000 --> 00:00:04,500
هذا ملف ترجمة تجريبي
This is a test subtitle file

2
00:01:05,250 --> 00:01:09,800
NEXORA Media System
نظام إدارة الوسائط
`

	var out bytes.Buffer
	err := ConvertSRTToWebVTT(strings.NewReader(srtInput), &out)
	if err != nil {
		t.Fatalf("ConvertSRTToWebVTT failed: %v", err)
	}

	result := out.String()
	if !strings.HasPrefix(result, "WEBVTT") {
		t.Errorf("expected WEBVTT header, got: %s", result)
	}
	if !strings.Contains(result, "00:00:01.000 --> 00:00:04.500") {
		t.Errorf("expected converted timestamp with period, got: %s", result)
	}
	if !strings.Contains(result, "00:01:05.250 --> 00:01:09.800") {
		t.Errorf("expected converted second timestamp with period, got: %s", result)
	}
}

func TestDetectSubtitleLanguage(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"Inception.2010.Arabic.srt", "Arabic"},
		{"Inception.2010.ar.srt", "Arabic"},
		{"Attack.on.Titan.S01E01.عربي.srt", "Arabic"},
		{"Attack.on.Titan.S01E01.English.srt", "English"},
		{"Attack.on.Titan.S01E01.en.srt", "English"},
		{"Movie.fr.srt", "French"},
		{"Anime.ja.ass", "Japanese"},
		{"Unknown.file.srt", "Undetermined"},
	}

	for _, tt := range tests {
		got := detectSubtitleLanguage(tt.filename)
		if got != tt.expected {
			t.Errorf("detectSubtitleLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
		}
	}
}
