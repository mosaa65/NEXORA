// Command resolutionprobe runs the entity-resolution pipeline against a real
// library and reports what it decided.
//
// It exists so the design can be judged on real data rather than on unit tests
// alone: it shows, for every file, which work the resolver chose (if any) and
// why, including the exact regressions the old per-file ingest produced.
//
// Usage: go run ./cmd/resolutionprobe [root]
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"nexora/server/internal/identity"
	"nexora/server/internal/scanner"
)

func main() {
	root := `C:\Users\mousa\Desktop\M\NEXORA_STRESS_ESTIRAHAT_V2\استراحة رئيسية`
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	if _, err := os.Stat(root); err != nil {
		fmt.Println("root not found:", root)
		os.Exit(1)
	}

	// Collect the files the scanner sees, with the group context the pipeline
	// would attach, then resolve each one against an empty library.
	sc := scanner.New(scanner.Options{Workers: 8})
	files := make([]scanner.FileInfo, 0, 256)
	if _, err := sc.ScanTree(context.Background(), []string{root}, scanner.ScanOptions{
		Mode:          scanner.ModeFull,
		EmitUnchanged: true,
	}, func(file scanner.FileInfo) error {
		files = append(files, file)
		return nil
	}); err != nil {
		fmt.Println("scan failed:", err)
		os.Exit(1)
	}

	fmt.Printf("scanned %d media files from %s\n", len(files), filepath.Base(root))

	resolver := identity.NewResolver()

	created := 0
	reviewed := 0
	unresolved := 0
	worksByTitle := map[string]int{}

	for i := range files {
		file := files[i]
		resolution := resolver.Resolve(identity.ResolverInput{Evidence: evidenceFor(file)})

		switch resolution.Decision {
		case identity.DecisionCreate:
			created++
			worksByTitle[resolution.NewWorkTitle]++
		case identity.DecisionAuto, identity.DecisionProvisional:
			created++
			if resolution.Work != nil {
				worksByTitle[resolution.Work.TitleEN]++
			}
		default:
			if resolution.State == identity.StateUnresolved {
				unresolved++
			} else {
				reviewed++
			}
		}
	}

	fmt.Println("=== RESOLUTION OUTCOME ON THE REAL LIBRARY ===")
	fmt.Printf("files                        : %d\n", len(files))
	fmt.Printf("attached or would create work: %d\n", created)
	fmt.Printf("queued for review            : %d\n", reviewed)
	fmt.Printf("unresolved (no usable name)  : %d\n", unresolved)
	fmt.Printf("distinct works proposed      : %d\n", len(worksByTitle))
	fmt.Println()

	fmt.Println("=== WORKS THE RESOLVER WOULD CREATE ===")
	names := make([]string, 0, len(worksByTitle))
	for name := range worksByTitle {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  %-46s %d file(s)\n", name, worksByTitle[name])
	}

	fmt.Println()
	fmt.Println("=== THE EXACT REGRESSIONS FROM THE OLD INGEST ===")
	// These were real works in the database before resolution existed.
	badTitles := []string{
		"01", "03", "x2", "10", "1",
		"الحلقة", "الحلقه", "الكنزنت", "منور فديوهات زابيا",
		"Fate Stay Night -", "Fate_Apocrypha",
	}
	regressions := 0
	for _, title := range badTitles {
		if count, exists := worksByTitle[title]; exists {
			fmt.Printf("  STILL CREATED as a work: %-28q (%d) <-- REGRESSION\n", title, count)
			regressions++
		} else {
			fmt.Printf("  correctly rejected      : %q\n", title)
		}
	}
	fmt.Printf("\nregressions: %d\n", regressions)

	fmt.Println()
	fmt.Println("=== SAMPLE DECISIONS WITH REASONING ===")
	shown := 0
	for i := range files {
		if shown >= 10 {
			break
		}
		file := files[i]
		if file.Group.Size < 2 {
			continue
		}
		resolution := resolver.Resolve(identity.ResolverInput{Evidence: evidenceFor(file)})
		relative, _ := filepath.Rel(root, file.Path)
		fmt.Printf("  %s\n", relative)
		fmt.Printf("      work=%q type=%s season=%d episode=%d\n",
			resolution.WorkTitleForDisplay(), resolution.MediaType, resolution.Season, resolution.Episode)
		fmt.Printf("      decision=%s reason=%q\n", resolution.Decision, resolution.ReasonCode)
		shown++
	}

	_ = strings.TrimSpace
}

// evidenceFor converts a scanned file into resolver evidence. This mirrors what
// the persistence layer builds, so the probe measures the real pipeline inputs.
func evidenceFor(file scanner.FileInfo) identity.Evidence {
	return identity.Evidence{
		Path:             file.Path,
		OriginalName:     file.Parsed.Original,
		FileSize:         file.Size,
		Segments:         file.PathSegments,
		ParsedTitle:      file.Parsed.Title,
		ParsedTitleAR:    file.Parsed.TitleAR,
		ParsedTitleEN:    file.Parsed.TitleEN,
		ParsedYear:       file.Parsed.ReleaseYear,
		ParsedSeason:     file.Parsed.SeasonNumber,
		ParsedEpisode:    file.Parsed.EpisodeNumber,
		ParsedPart:       file.Parsed.PartNumber,
		ParserCategory:   file.Parsed.CategorySlug,
		ParserConfidence: float64(file.Parsed.Confidence),
		ParserIsEpisode:  file.Parsed.IsEpisode,
		ParserEpisodeStrong: file.Parsed.EpisodeSource == scanner.SourceFilenamePattern ||
			file.Parsed.EpisodeSource == scanner.SourceKeyword,
		WorkFolderTitle:    file.Context.WorkTitle,
		SeasonFolderNumber: file.Context.SeasonNumber,
		OriginTag:          file.Parsed.OriginTag,
		CategoryHint:       file.Context.Category,
		Group: identity.GroupContext{
			Size:             file.Group.Size,
			EpisodicSiblings: file.Group.EpisodicSiblings,
			EpisodeNumbers:   file.Group.EpisodeNumbers,
			YearSiblings:     file.Group.YearSiblings,
			SeasonFolder:     file.Group.SeasonFolder,
			ConsensusTitle:   file.Group.ConsensusTitle,
			ConsensusVotes:   file.Group.ConsensusVotes,
		},
	}
}
