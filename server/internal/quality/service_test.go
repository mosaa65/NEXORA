package quality

import (
	"context"
	"testing"
	"time"
)

func TestMissingEpisodeDetailFormatting(t *testing.T) {
	detail := MissingEpisodeDetail{
		MediaItemID:  1,
		ShowTitleAR:  "هجوم العمالقة",
		ShowTitleEN:  "Attack on Titan",
		CategorySlug: "anime",
		SeasonID:     1,
		SeasonNumber: 1,
		MissingEpNum: 3,
	}

	detail.SuggestedName = "Attack on Titan S01E03.mkv"

	if detail.SuggestedName != "Attack on Titan S01E03.mkv" {
		t.Errorf("expected suggested name, got: %s", detail.SuggestedName)
	}
}

func TestQualityReportStruct(t *testing.T) {
	now := time.Now().UTC()
	report := QualityReport{
		GeneratedAt:          now,
		TotalMedia:           10,
		TotalFiles:           50,
		TotalSizeBytes:       1024 * 1024 * 1024 * 50, // 50 GB
		TotalWastedBytes:     1024 * 1024 * 1024 * 5,  // 5 GB
		DuplicateGroupsCount: 2,
		MissingEpisodesCount: 3,
		CorruptedFilesCount:  1,
	}

	if report.TotalMedia != 10 || report.TotalFiles != 50 {
		t.Errorf("expected 10 media, 50 files, got %d, %d", report.TotalMedia, report.TotalFiles)
	}
	if report.TotalWastedBytes != 5*1024*1024*1024 {
		t.Errorf("expected 5GB wasted bytes, got %d", report.TotalWastedBytes)
	}
}

func TestVerifyFileWithEmptyPath(t *testing.T) {
	svc := NewService(nil, "ffmpeg", "ffprobe")
	_, _, err := svc.VerifyFile(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty file path, got nil")
	}
}
