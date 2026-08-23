package main

import (
	"context"
	"fmt"
	"nexora/server/internal/scanner"
)

func main() {
	s := scanner.New(scanner.Options{Workers: 2})
	files, err := s.Scan(context.Background(), []string{`C:\Users\mousa\Desktop\مسلسل اجنبي`})
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		return
	}
	fmt.Printf("Scanned %d files:\n", len(files))
	for _, f := range files {
		fmt.Printf("File: %s\n  Title: %q (AR: %q, EN: %q)\n  Season: %d, Ep: %d, IsEp: %v\n  Artwork: %q\n",
			f.Path, f.Parsed.Title, f.Parsed.TitleAR, f.Parsed.TitleEN, f.Parsed.SeasonNumber, f.Parsed.EpisodeNumber, f.Parsed.IsEpisode, f.ArtworkPath)
	}
}
